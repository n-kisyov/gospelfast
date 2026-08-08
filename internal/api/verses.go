package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gospelfast/gospelfast/internal/bible"
)

type VerseResponse struct {
	Reference string `json:"reference"`
	BookName  string `json:"book_name"`
	Chapter   int    `json:"chapter"`
	Verse     int    `json:"verse"`
	VerseEnd  int    `json:"verse_end,omitempty"`
	Text      string `json:"text"`
	IsRange   bool   `json:"is_range,omitempty"`
}

// @Summary      Get verses by reference
// @Tags         verses
// @Produce      json
// @Param        ref  query  string  true  "Bible reference (e.g. John+3:16 or John+3:16-18)"
// @Param        t    query  string  true  "Translation short name"
// @Success      200  {object}  object
// @Failure      400  {object}  object
// @Failure      404  {object}  object
// @Router       /api/verses [get]
func (h *Handler) GetVerses(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	refStr := r.URL.Query().Get("ref")
	transShort := r.URL.Query().Get("t")

	if refStr == "" {
		writeError(w, http.StatusBadRequest, "query parameter 'ref' is required")
		return
	}
	if transShort == "" {
		writeError(w, http.StatusBadRequest, "query parameter 't' is required")
		return
	}

	ref, err := bible.ParseRef(strings.ReplaceAll(refStr, "+", " "))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid reference: "+err.Error())
		return
	}

	trans, err := h.DB.GetTranslationByShortName(ctx, transShort)
	if err != nil {
		writeError(w, http.StatusNotFound, "translation not found: "+transShort)
		return
	}

	book, err := h.DB.GetBookByShortName(ctx, trans.VersificationID, ref.BookShortName)
	if err != nil {
		writeError(w, http.StatusNotFound, "book not found: "+ref.BookShortName)
		return
	}

	v, err := h.DB.GetVerse(ctx, trans.ID, book.ID, ref.Chapter, ref.Verse)
	if err != nil {
		writeError(w, http.StatusNotFound, "verse not found")
		return
	}

	resp := VerseResponse{
		Reference: refStr,
		BookName:  v.BookName,
		Chapter:   v.Chapter,
		Verse:     v.Verse,
		Text:      v.Text,
	}

	if ref.IsRange {
		endV, err := h.DB.GetVerse(ctx, trans.ID, book.ID, ref.Chapter, ref.VerseEnd)
		if err == nil {
			resp.VerseEnd = endV.Verse
			resp.Text = strings.TrimSpace(resp.Text + " " + endV.Text)
			resp.IsRange = true
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// @Summary      Search verses
// @Tags         search
// @Produce      json
// @Param        q       query  string  true  "Search query"
// @Param        t       query  string  true  "Translation short name"
// @Param        page    query  int     false "Page number"
// @Param        per_page query int     false "Results per page"
// @Success      200  {object}  object
// @Failure      400  {object}  object
// @Router       /api/search [get]
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	query := r.URL.Query().Get("q")
	transShort := r.URL.Query().Get("t")

	if query == "" {
		writeError(w, http.StatusBadRequest, "query parameter 'q' is required")
		return
	}
	if transShort == "" {
		writeError(w, http.StatusBadRequest, "query parameter 't' is required")
		return
	}

	trans, err := h.DB.GetTranslationByShortName(ctx, transShort)
	if err != nil {
		writeError(w, http.StatusNotFound, "translation not found: "+transShort)
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage <= 0 {
		perPage = 30
	}

	cacheKey := h.Cache.SearchKey(query, transShort, page)
	if h.Cache != nil {
		var cached map[string]any
		if ok, _ := h.Cache.Get(ctx, cacheKey, &cached); ok {
			writeJSON(w, http.StatusOK, cached)
			return
		}
	}

	results, total, err := h.DB.SearchVerses(ctx, trans.ID, query, perPage, (page-1)*perPage)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	type resultItem struct {
		BookName      string  `json:"book_name"`
		BookShortName string  `json:"book_short_name"`
		Chapter       int     `json:"chapter"`
		Verse         int     `json:"verse"`
		Text          string  `json:"text"`
		Snippet       string  `json:"snippet"`
		Rank          float64 `json:"rank"`
	}

	items := make([]resultItem, len(results))
	for i, r := range results {
		items[i] = resultItem{
			BookName:      r.BookName,
			BookShortName: r.BookShortName,
			Chapter:       r.Chapter,
			Verse:         r.Verse,
			Text:          r.Text,
			Snippet:       r.Snippet,
			Rank:          r.Rank,
		}
	}

	resp := map[string]any{
		"query":    query,
		"total":    total,
		"page":     page,
		"per_page": perPage,
		"results":  items,
	}

	if h.Cache != nil {
		_ = h.Cache.Set(ctx, cacheKey, resp, 0)
	}

	writeJSON(w, http.StatusOK, resp)
}
