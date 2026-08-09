package admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gospelfast/gospelfast/internal/cache"
	"github.com/gospelfast/gospelfast/internal/db"
)

type pageData struct {
	Title    string
	MetaDesc string
	Content  template.HTML
}

type WebHandler struct {
	DB       *db.DB
	Cache    *cache.Cache
	Jobs     *JobManager
	base     *template.Template
	templates map[string]*template.Template
}

func NewWebHandler(d *db.DB, c *cache.Cache, jm *JobManager, templatesDir string) (*WebHandler, error) {
	funcMap := template.FuncMap{
		"sub": func(a, b int) int { return a - b },
		"add": func(a, b int) int { return a + b },
		"gt":  func(a, b int) bool { return a > b },
		"lt":  func(a, b int) bool { return a < b },
	}

	base, err := template.New("base.html").Funcs(funcMap).ParseFiles(filepath.Join(templatesDir, "base.html"))
	if err != nil {
		return nil, fmt.Errorf("parse base: %w", err)
	}

	h := &WebHandler{
		DB:        d,
		Cache:     c,
		Jobs:      jm,
		base:      base,
		templates: make(map[string]*template.Template),
	}

	pageDeps := map[string][]string{
		"dashboard.html": {},
		"import_status.html": {},
	}

	adminDir := filepath.Join(templatesDir, "admin")
	for name := range pageDeps {
		path := filepath.Join(adminDir, name)
		if _, err := os.Stat(path); err != nil {
			continue
		}
		tmpl, err := base.Clone()
		if err != nil {
			return nil, err
		}
		paths := []string{path}
		for _, dep := range pageDeps[name] {
			paths = append(paths, filepath.Join(adminDir, dep))
		}
		tmpl, err = tmpl.ParseFiles(paths...)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", name, err)
		}
		h.templates[name] = tmpl
	}

	return h, nil
}

func (h *WebHandler) renderFragment(w http.ResponseWriter, name string, data any) {
	tmpl, ok := h.templates[name]
	if !ok {
		http.Error(w, "template not found: "+name, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("render fragment %s: %v", name, err)
	}
}

func (h *WebHandler) renderPage(w http.ResponseWriter, name string, data any, title, metaDesc string) {
	tmpl, ok := h.templates[name]
	if !ok {
		http.Error(w, "template not found: "+name, http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		log.Printf("render %s: %v", name, err)
		http.Error(w, "render error", http.StatusInternalServerError)
		return
	}

	pd := pageData{
		Title:    title,
		MetaDesc: metaDesc,
		Content:  template.HTML(buf.String()),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.base.ExecuteTemplate(w, "base.html", pd); err != nil {
		log.Printf("render base: %v", err)
	}
}

type dashboardData struct {
	Translations []translationRow
	Jobs         []*Job
}

type translationRow struct {
	ID         string
	ShortName  string
	FullName   string
	Language   string
	CreatedAt  string
	VerseCount int
}

func (h *WebHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	data := dashboardData{Jobs: h.Jobs.List()}

	translations, err := h.DB.ListTranslations(ctx)
	if err == nil {
		for _, t := range translations {
			var verseCount int
			h.DB.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM verses WHERE translation_id = $1", t.ID).Scan(&verseCount)
			data.Translations = append(data.Translations, translationRow{
				ID:         t.ID,
				ShortName:  t.ShortName,
				FullName:   t.FullName,
				Language:   t.Language,
				CreatedAt:  t.CreatedAt.Format("2006-01-02"),
				VerseCount: verseCount,
			})
		}
	}

	h.renderPage(w, "dashboard.html", data, "Gospelfast — Admin", "Import and manage Bible translations")
}

// @Summary      Start import job
// @Tags         admin
// @Security     BasicAuth
// @Accept       multipart/form-data
// @Param        source formData string true "Source URL"
// @Param        name   formData string false "Translation name"
// @Param        format formData string false "Format (rawzip, osis, vpl)"
// @Success      200
// @Router       /admin/api/imports [post]
func (h *WebHandler) StartImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	source := strings.TrimSpace(r.FormValue("source"))
	name := strings.TrimSpace(r.FormValue("name"))
	format := strings.TrimSpace(r.FormValue("format"))
	if format == "" {
		format = "rawzip"
	}
	if source == "" {
		http.Error(w, "source URL is required", http.StatusBadRequest)
		return
	}
	if name == "" {
		name = filepath.Base(strings.TrimSuffix(source, ".zip"))
	}

	job := h.Jobs.Start(source, name, format)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.renderFragment(w, "import_status.html", map[string]any{"Job": job, "Percent": 0})
}

// @Summary      Get import job status
// @Tags         admin
// @Security     BasicAuth
// @Produce      json
// @Param        id path string true "Job ID"
// @Success      200  {object}  Job
// @Router       /admin/api/imports/{id} [get]
func (h *WebHandler) ImportStatus(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")
	if jobID == "" {
		http.Error(w, "job id required", http.StatusBadRequest)
		return
	}

	job := h.Jobs.Get(jobID)
	if job == nil {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}

	pct := 0
	if job.Total > 0 {
		pct = job.Progress * 100 / job.Total
	}

	if job.Status == StatusComplete || job.Status == StatusFailed {
		w.Header().Set("HX-Refresh", "true")
	}

	if strings.Contains(r.Header.Get("Accept"), "application/json") {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(job)
		return
	}

	h.renderFragment(w, "import_status.html", map[string]any{"Job": job, "Percent": pct})
}
