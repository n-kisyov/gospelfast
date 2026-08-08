package api

import (
	"net/http"
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
// @Router       /api/genbooks/{path} [get]
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
