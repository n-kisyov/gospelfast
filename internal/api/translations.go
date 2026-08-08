package api

import (
	"encoding/json"
	"net/http"

	"github.com/gospelfast/gospelfast/internal/bible"
)

type TranslationResponse struct {
	ID              string          `json:"id"`
	ShortName       string          `json:"short_name"`
	FullName        string          `json:"full_name"`
	Language        string          `json:"language"`
	VersificationID string          `json:"versification_id"`
	Description     string          `json:"description,omitempty"`
	SourceURL       string          `json:"source_url,omitempty"`
	Copyright       string          `json:"copyright,omitempty"`
	Metadata        json.RawMessage `json:"metadata,omitempty"`
	CreatedAt       string          `json:"created_at"`
}

func translationToResponse(t *bible.Translation) TranslationResponse {
	return TranslationResponse{
		ID:              t.ID,
		ShortName:       t.ShortName,
		FullName:        t.FullName,
		Language:        t.Language,
		VersificationID: t.VersificationID,
		Description:     t.Description,
		SourceURL:       t.SourceURL,
		Copyright:       t.Copyright,
		Metadata:        t.Metadata,
		CreatedAt:       t.CreatedAt.Format("2006-01-02"),
	}
}

// @Summary      List all translations
// @Tags         translations
// @Produce      json
// @Success      200  {array}  object
// @Router       /api/translations [get]
func (h *Handler) ListTranslations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.Cache != nil {
		var cached []TranslationResponse
		if ok, _ := h.Cache.Get(ctx, h.Cache.TranslationsKey(), &cached); ok {
			writeJSON(w, http.StatusOK, cached)
			return
		}
	}

	translations, err := h.DB.ListTranslations(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]TranslationResponse, len(translations))
	for i := range translations {
		result[i] = translationToResponse(&translations[i])
	}

	if h.Cache != nil {
		_ = h.Cache.Set(ctx, h.Cache.TranslationsKey(), result, 0)
	}

	writeJSON(w, http.StatusOK, result)
}
