package api

import (
	"net/http"
	"strconv"
	"strings"
)

type GenbookResponse struct {
	Entries []genbookEntryResp `json:"entries"`
}

type genbookEntryResp struct {
	Path    string `json:"path"`
	Title   string `json:"title"`
	Content string `json:"content,omitempty"`
}

// @Summary      Browse genbook entries
// @Tags         genbooks
// @Produce      json
// @Param        t           query  string  true  "Translation short name"
// @Param        path        query  string  false "Parent path"
// @Success      200  {object}  GenbookResponse
// @Router       /api/genbooks [get]
func (h *Handler) ListGenbooks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	transShort := r.URL.Query().Get("t")
	parentPath := r.URL.Query().Get("path")
	if parentPath == "" {
		parentPath = "/"
	}

	trans, err := h.DB.GetTranslationByShortName(ctx, transShort)
	if err != nil {
		writeError(w, http.StatusNotFound, "translation not found")
		return
	}

	entries, err := h.DB.GetGenbookPaths(ctx, trans.ID, parentPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := GenbookResponse{}
	for _, e := range entries {
		resp.Entries = append(resp.Entries, genbookEntryResp{
			Path:  e.Path,
			Title: e.Title,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

// @Summary      Get genbook entry content
// @Tags         genbooks
// @Produce      json
// @Param        t     query  string  true  "Translation short name"
// @Param        path  query  string  true  "Entry path"
// @Success      200  {object}  genbookEntryResp
// @Router       /api/genbooks/entry [get]
func (h *Handler) GetGenbookEntry(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	transShort := r.URL.Query().Get("t")
	entryPath := r.URL.Query().Get("path")

	trans, err := h.DB.GetTranslationByShortName(ctx, transShort)
	if err != nil {
		writeError(w, http.StatusNotFound, "translation not found")
		return
	}

	entry, err := h.DB.GetGenbookEntry(ctx, trans.ID, entryPath)
	if err != nil {
		writeError(w, http.StatusNotFound, "entry not found")
		return
	}

	writeJSON(w, http.StatusOK, genbookEntryResp{
		Path:    entry.Path,
		Title:   entry.Title,
		Content: entry.Content,
	})
}

// @Summary      Get commentary for a verse
// @Tags         commentary
// @Produce      json
// @Param        t    query  string  true  "Commentary short name"
// @Param        book query  string  true  "Book short name"
// @Param        ref  query  string  true  "Verse reference (e.g. Gen.1.1)"
// @Success      200  {object}  object
// @Router       /api/commentary [get]
func (h *Handler) GetCommentary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	transShort := r.URL.Query().Get("t")
	bookShort := r.URL.Query().Get("book")

	chapter, verse := 0, 0
	if ref := r.URL.Query().Get("ref"); ref != "" {
		parts := strings.SplitN(ref, ".", 3)
		if len(parts) >= 2 {
			chapter, _ = strconv.Atoi(parts[1])
		}
		if len(parts) >= 3 {
			verse, _ = strconv.Atoi(parts[2])
		}
	}

	trans, err := h.DB.GetTranslationByShortName(ctx, transShort)
	if err != nil {
		writeError(w, http.StatusNotFound, "translation not found")
		return
	}

	book, err := h.DB.GetBookByShortName(ctx, trans.VersificationID, bookShort)
	if err != nil {
		writeError(w, http.StatusNotFound, "book not found")
		return
	}

	entry, err := h.DB.GetCommentary(ctx, trans.ID, book.ID, chapter, verse)
	if err != nil {
		writeError(w, http.StatusNotFound, "no commentary for this verse")
		return
	}

	writeJSON(w, http.StatusOK, entry)
}
