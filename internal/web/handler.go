package web

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gospelfast/gospelfast/internal/cache"
	"github.com/gospelfast/gospelfast/internal/db"
)

type pageData struct {
	Title    string
	MetaDesc string
	Content  template.HTML
}

type pageTemplate struct {
	page *template.Template
	name string
}

type Handler struct {
	DB       *db.DB
	Cache    *cache.Cache
	base     *template.Template
	pages    map[string]*pageTemplate
	templatesDir string
}

func NewHandler(d *db.DB, c *cache.Cache, templatesDir string) (*Handler, error) {
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

	h := &Handler{
		DB:       d,
		Cache:    c,
		base:     base,
		pages:    make(map[string]*pageTemplate),
		templatesDir: templatesDir,
	}

	pageDeps := map[string][]string{
		"home.html":             {"search.html"},
		"reader.html":           {"reader_chapter.html"},
		"reader_chapter.html":   {},
		"compare.html":          {},
		"compare_result.html":   {},
		"search.html":           {},
		"genbook.html":          {},
	}

	for name, deps := range pageDeps {
		path := filepath.Join(templatesDir, name)
		if _, err := os.Stat(path); err != nil {
			log.Printf("template %s not found, skipping", name)
			continue
		}
		tmpl, err := base.Clone()
		if err != nil {
			return nil, fmt.Errorf("clone for %s: %w", name, err)
		}
		paths := []string{path}
		for _, dep := range deps {
			paths = append(paths, filepath.Join(templatesDir, dep))
		}
		tmpl, err = tmpl.ParseFiles(paths...)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", name, err)
		}
		h.pages[name] = &pageTemplate{page: tmpl, name: name}
	}

	return h, nil
}

func (h *Handler) renderPage(w http.ResponseWriter, pageName string, data any, title, metaDesc string) {
	pt, ok := h.pages[pageName]
	if !ok {
		http.Error(w, "template not found: "+pageName, http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	if err := pt.page.ExecuteTemplate(&buf, pageName, data); err != nil {
		log.Printf("render %s: %v", pageName, err)
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

type translationItem struct {
	ID        string
	ShortName string
	FullName  string
}

type homeData struct {
	Translations []translationItem
	Selected     string
	Query        string
	Total        int
	Page         int
	PageSize     int
	NextPage     int
	Results      []searchResultItem
	VOTDRef      string
	VOTDText     string
}

type searchResultItem struct {
	BookShortName string
	Chapter       int
	Verse         int
	Snippet       template.HTML
	Rank          float64
}

func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	data := homeData{Selected: "KJV"}

	translations, err := h.DB.ListTranslations(ctx)
	if err == nil {
		for _, t := range translations {
			data.Translations = append(data.Translations, translationItem{
				ID: t.ID, ShortName: t.ShortName, FullName: t.FullName,
			})
		}
	}

	trans, err := h.DB.GetTranslationByShortName(ctx, "KJV")
	if err == nil {
		today := time.Now().Format("2006-01-02")
		seed := int64(hashString(today))
		verse, err := h.DB.GetRandomVerse(ctx, trans.ID, seed)
		if err == nil {
			data.VOTDRef = fmt.Sprintf("%s %d:%d", verse.BookName, verse.Chapter, verse.Verse)
			data.VOTDText = verse.Text
		}
	}

	h.renderPage(w, "home.html", data, "Gospelfast — Search", "Fast full-text search Bible study tool")
}

func hashString(s string) uint64 {
	var h uint64 = 5381
	for i := 0; i < len(s); i++ {
		h = ((h << 5) + h) + uint64(s[i])
	}
	return h
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query().Get("q")
	tShort := r.URL.Query().Get("t")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	perPage := 30

	data := homeData{Query: q, Selected: tShort, Page: page, PageSize: perPage, NextPage: page + 1}

	if tShort == "" {
		tShort = "KJV"
	}

	if q != "" {
		trans, err := h.DB.GetTranslationByShortName(ctx, tShort)
		if err == nil {
			dbResults, total, err := h.DB.SearchVerses(ctx, trans.ID, q, perPage, (page-1)*perPage)
			if err == nil {
				data.Total = total
				for _, r := range dbResults {
					data.Results = append(data.Results, searchResultItem{
						BookShortName: r.BookShortName,
						Chapter:       r.Chapter,
						Verse:         r.Verse,
						Snippet:       template.HTML(r.Snippet),
						Rank:          r.Rank,
					})
				}
			}
		}
	}

	h.renderPage(w, "search.html", data, "Gospelfast — Search", "Bible search results")
}

type readerData struct {
	Translations []translationItem
	Selected     string
	Books        []bookItem
	BookName     string
	BookShort    string
	Chapter      int
	ChapterCount int
	Verses       []verseItem
}

type bookItem struct {
	ShortName    string
	Name         string
	ChapterCount int
}

type verseItem struct {
	Verse int
	Text  string
}

func (h *Handler) Reader(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	data := readerData{Selected: "KJV", BookShort: "Gen", Chapter: 1}

	if t := r.URL.Query().Get("t"); t != "" {
		data.Selected = t
	}
	if b := r.URL.Query().Get("book"); b != "" {
		data.BookShort = b
	}
	if c := r.URL.Query().Get("chapter"); c != "" {
		data.Chapter, _ = strconv.Atoi(c)
	}

	translations, err := h.DB.ListTranslations(ctx)
	if err == nil {
		for _, t := range translations {
			data.Translations = append(data.Translations, translationItem{ID: t.ID, ShortName: t.ShortName, FullName: t.FullName})
		}
	}

	trans, err := h.DB.GetTranslationByShortName(ctx, data.Selected)
	if err == nil {
		books, err := h.DB.GetBooksByVersification(ctx, trans.VersificationID)
		if err == nil && len(books) > 0 {
			for _, b := range books {
				data.Books = append(data.Books, bookItem{ShortName: b.ShortName, Name: b.Name, ChapterCount: b.ChapterCount})
			}

			book, _ := h.DB.GetBookByShortName(ctx, trans.VersificationID, data.BookShort)
			if book != nil {
				data.BookName = book.Name
				data.ChapterCount = book.ChapterCount
				verses, _ := h.DB.GetChapterVerses(ctx, trans.ID, book.ID, data.Chapter)
				for _, v := range verses {
					data.Verses = append(data.Verses, verseItem{Verse: v.Verse, Text: v.Text})
				}
			}
		}
	}

	h.renderPage(w, "reader.html", data, "Gospelfast — Reader", "Read the Bible online")
}

func (h *Handler) ReaderChapter(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tShort := r.URL.Query().Get("t")
	bookShort := r.URL.Query().Get("book")
	chapter, _ := strconv.Atoi(r.URL.Query().Get("chapter"))

	data := readerData{Selected: tShort, BookShort: bookShort, Chapter: chapter}

	trans, err := h.DB.GetTranslationByShortName(ctx, tShort)
	if err == nil {
		book, err := h.DB.GetBookByShortName(ctx, trans.VersificationID, bookShort)
		if err == nil {
			data.BookName = book.Name
			data.ChapterCount = book.ChapterCount
			dbVerses, _ := h.DB.GetChapterVerses(ctx, trans.ID, book.ID, chapter)
			for _, v := range dbVerses {
				data.Verses = append(data.Verses, verseItem{Verse: v.Verse, Text: v.Text})
			}
		}
	}

	h.renderPage(w, "reader_chapter.html", data, "Gospelfast — Reader", "Read the Bible online")
}

type compareData struct {
	Translations []translationItem
}

func (h *Handler) Compare(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	data := compareData{}

	translations, err := h.DB.ListTranslations(ctx)
	if err == nil {
		for _, t := range translations {
			data.Translations = append(data.Translations, translationItem{ID: t.ID, ShortName: t.ShortName, FullName: t.FullName})
		}
	}

	h.renderPage(w, "compare.html", data, "Gospelfast — Compare", "Compare Bible translations side by side")
}

func (h *Handler) CompareResults(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ref := r.URL.Query().Get("ref")
	tA := r.URL.Query().Get("a")
	tB := r.URL.Query().Get("b")
	ref = strings.ReplaceAll(ref, "+", " ")

	data := map[string]any{"Reference": ref}

	parsed, err := parseSimpleRef(ref)
	if err != nil {
		data["Error"] = "Invalid reference: " + ref
		h.renderPage(w, "compare_result.html", data, " — Compare", "Compare Bible translations")
		return
	}

	textA, _ := fetchVerseText(ctx, h.DB, tA, parsed)
	textB, _ := fetchVerseText(ctx, h.DB, tB, parsed)

	if textA != "" {
		data["TextA"] = tA + ": " + textA
	}
	if textB != "" {
		data["TextB"] = tB + ": " + textB
	}

	h.renderPage(w, "compare_result.html", data, " — Compare", "Compare Bible translations")
}

type simpleRef struct {
	book    string
	chapter int
	verse   int
}

func parseSimpleRef(ref string) (*simpleRef, error) {
	ref = strings.TrimSpace(ref)
	bookMap := map[string]string{
		"genesis": "Gen", "exodus": "Exod", "leviticus": "Lev", "numbers": "Num",
		"deuteronomy": "Deut", "joshua": "Josh", "judges": "Judg", "ruth": "Ruth",
		"1 samuel": "1Sam", "2 samuel": "2Sam", "1 kings": "1Kgs", "2 kings": "2Kgs",
		"1 chronicles": "1Chr", "2 chronicles": "2Chr", "ezra": "Ezra", "nehemiah": "Neh",
		"esther": "Esth", "job": "Job", "psalm": "Ps", "psalms": "Ps",
		"proverbs": "Prov", "ecclesiastes": "Eccl", "song of solomon": "Song",
		"isaiah": "Isa", "jeremiah": "Jer", "lamentations": "Lam", "ezekiel": "Ezek",
		"daniel": "Dan", "hosea": "Hos", "joel": "Joel", "amos": "Amos",
		"obadiah": "Obad", "jonah": "Jonah", "micah": "Mic", "nahum": "Nah",
		"habakkuk": "Hab", "zephaniah": "Zeph", "haggai": "Hag", "zechariah": "Zech",
		"malachi": "Mal",
		"matthew": "Matt", "mark": "Mark", "luke": "Luke", "john": "John",
		"acts": "Acts", "romans": "Rom", "1 corinthians": "1Cor", "2 corinthians": "2Cor",
		"galatians": "Gal", "ephesians": "Eph", "philippians": "Phil", "colossians": "Col",
		"1 thessalonians": "1Thess", "2 thessalonians": "2Thess", "1 timothy": "1Tim",
		"2 timothy": "2Tim", "titus": "Titus", "philemon": "Phlm", "hebrews": "Heb",
		"james": "Jas", "1 peter": "1Pet", "2 peter": "2Pet", "1 john": "1John",
		"2 john": "2John", "3 john": "3John", "jude": "Jude", "revelation": "Rev",
	}

	parts := strings.SplitN(ref, " ", 2)
	if len(parts) < 2 {
		return nil, strconv.ErrSyntax
	}
	bookLower := strings.ToLower(parts[0])
	versePart := parts[1]

	short, ok := bookMap[bookLower]
	if !ok {
		return nil, strconv.ErrSyntax
	}

	colon := strings.Index(versePart, ":")
	if colon < 0 {
		return nil, strconv.ErrSyntax
	}
	ch, _ := strconv.Atoi(versePart[:colon])
	v, _ := strconv.Atoi(versePart[colon+1:])

	return &simpleRef{book: short, chapter: ch, verse: v}, nil
}

func fetchVerseText(ctx context.Context, database *db.DB, shortName string, ref *simpleRef) (string, bool) {
	trans, err := database.GetTranslationByShortName(ctx, shortName)
	if err != nil {
		return "", false
	}
	book, err := database.GetBookByShortName(ctx, trans.VersificationID, ref.book)
	if err != nil {
		return "", false
	}
	verse, err := database.GetVerse(ctx, trans.ID, book.ID, ref.chapter, ref.verse)
	if err != nil {
		return "", false
	}
	return verse.Text, true
}

type genbookPageData struct {
	Translations []translationItem
	Selected     string
	Title        string
	CurrentTitle string
	CurrentPath  string
	ParentPath   string
	Content      string
	TOC          []genbookTOCEntry
}

type genbookTOCEntry struct {
	Path  string
	Title string
}

func (h *Handler) RenderError(w http.ResponseWriter, code int, title, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	pd := pageData{
		Title:    fmt.Sprintf("%d — %s", code, title),
		MetaDesc: msg,
		Content: template.HTML(fmt.Sprintf(
			`<div class="max-w-2xl mx-auto text-center py-16">
				<div class="text-6xl font-bold text-gray-200 dark:text-gray-700 mb-4">%d</div>
				<h1 class="text-xl font-semibold text-gray-800 dark:text-gray-200 mb-2">%s</h1>
				<p class="text-gray-500 dark:text-gray-400 mb-6">%s</p>
				<a href="/" class="text-blue-600 hover:underline text-sm">Go home</a>
			</div>`, code, title, msg)),
	}
	_ = h.base.ExecuteTemplate(w, "base.html", pd)
}

func (h *Handler) Genbook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	data := genbookPageData{Selected: "ENOCH", CurrentPath: "/"}

	if t := r.URL.Query().Get("t"); t != "" {
		data.Selected = t
	}
	if p := r.URL.Query().Get("path"); p != "" {
		data.CurrentPath = p
	}

	translations, err := h.DB.ListTranslations(ctx)
	if err == nil {
		for _, t := range translations {
			data.Translations = append(data.Translations, translationItem{
				ID: t.ID, ShortName: t.ShortName, FullName: t.FullName,
			})
		}
	}

	trans, err := h.DB.GetTranslationByShortName(ctx, data.Selected)
	if err != nil {
		h.renderPage(w, "genbook.html", data, "Gospelfast — Book", "Read general books and apocrypha")
		return
	}

	data.Title = trans.FullName

	tocParent := data.CurrentPath
	if tocParent == "" {
		tocParent = "/"
	}

	entries, err := h.DB.GetGenbookPaths(ctx, trans.ID, tocParent)
	if err == nil {
		for _, e := range entries {
			data.TOC = append(data.TOC, genbookTOCEntry{Path: e.Path, Title: e.Title})
		}
	}

	if lastSlash := strings.LastIndex(data.CurrentPath, "/"); lastSlash > 0 {
		data.ParentPath = data.CurrentPath[:lastSlash]
	} else if data.CurrentPath != "/" && data.CurrentPath != "" {
		data.ParentPath = "/"
	}

	if data.CurrentPath != "/" && data.CurrentPath != "" {
		entry, err := h.DB.GetGenbookEntry(ctx, trans.ID, data.CurrentPath)
		if err == nil {
			data.CurrentTitle = entry.Title
			data.Content = entry.Content
		}
	}

	h.renderPage(w, "genbook.html", data, "Gospelfast — "+data.Selected, "Read "+data.Selected)
}
