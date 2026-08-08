package admin

import (
	"encoding/json"
	"net/http"

	"github.com/gospelfast/gospelfast/internal/cache"
	"github.com/gospelfast/gospelfast/internal/db"
)

type Handler struct {
	DB    *db.DB
	Cache *cache.Cache
}

func NewHandler(d *db.DB, c *cache.Cache) *Handler {
	return &Handler{DB: d, Cache: c}
}

// @Summary      List translations (admin)
// @Tags         admin
// @Security     BasicAuth
// @Produce      json
// @Success      200  {object}  object
// @Router       /admin/api/translations [get]
func (h *Handler) ListTranslations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	translations, err := h.DB.ListTranslations(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	type response struct {
		ID        string `json:"id"`
		ShortName string `json:"short_name"`
		FullName  string `json:"full_name"`
		Language  string `json:"language"`
		CreatedAt string `json:"created_at"`
	}

	result := make([]response, len(translations))
	for i := range translations {
		t := &translations[i]
		result[i] = response{
			ID:        t.ID,
			ShortName: t.ShortName,
			FullName:  t.FullName,
			Language:  t.Language,
			CreatedAt: t.CreatedAt.Format("2006-01-02"),
		}
	}

	writeJSON(w, http.StatusOK, result)
}

// @Summary      Delete a translation
// @Tags         admin
// @Security     BasicAuth
// @Param        id  path  string  true  "Translation ID"
// @Success      204
// @Router       /admin/api/translations/{id} [delete]
func (h *Handler) DeleteTranslation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")

	if err := h.DB.DeleteTranslation(ctx, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if h.Cache != nil {
		_ = h.Cache.DelPattern(ctx, "chapter:*")
		_ = h.Cache.Del(ctx, h.Cache.TranslationsKey())
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
