package api

import (
	"net/http"
	"strconv"
)

type BookResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ShortName    string `json:"short_name"`
	Testament    string `json:"testament"`
	BookOrder    int    `json:"book_order"`
	ChapterCount int    `json:"chapter_count"`
}

// @Summary      List books for a translation
// @Tags         books
// @Produce      json
// @Param        t  query  string  true  "Translation short name"
// @Success      200  {array}  object
// @Failure      404  {object}  object
// @Router       /api/books [get]
func (h *Handler) ListBooks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	transShort := r.URL.Query().Get("t")
	if transShort == "" {
		writeError(w, http.StatusBadRequest, "query parameter 't' is required")
		return
	}

	trans, err := h.DB.GetTranslationByShortName(ctx, transShort)
	if err != nil {
		writeError(w, http.StatusNotFound, "translation not found: "+transShort)
		return
	}

	cacheKey := h.Cache.BooksKey(trans.ShortName)
	if h.Cache != nil {
		var cached []BookResponse
		if ok, _ := h.Cache.Get(ctx, cacheKey, &cached); ok {
			writeJSON(w, http.StatusOK, cached)
			return
		}
	}

	books, err := h.DB.GetBooksByVersification(ctx, trans.VersificationID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]BookResponse, len(books))
	for i := range books {
		b := &books[i]
		result[i] = BookResponse{
			ID:           b.ID,
			Name:         b.Name,
			ShortName:    b.ShortName,
			Testament:    b.Testament,
			BookOrder:    b.BookOrder,
			ChapterCount: b.ChapterCount,
		}
	}

	if h.Cache != nil {
		_ = h.Cache.Set(ctx, cacheKey, result, 0)
	}

	writeJSON(w, http.StatusOK, result)
}

// @Summary      Get a book by short name
// @Tags         books
// @Produce      json
// @Param        t    query  string  true  "Translation short name"
// @Param        book query  string  true  "Book short name (e.g. Gen)"
// @Success      200  {object}  object
// @Failure      404  {object}  object
// @Router       /api/books/{book} [get]
func (h *Handler) GetBook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	transShort := r.URL.Query().Get("t")
	bookShort := r.PathValue("book")

	trans, err := h.DB.GetTranslationByShortName(ctx, transShort)
	if err != nil {
		writeError(w, http.StatusNotFound, "translation not found: "+transShort)
		return
	}

	b, err := h.DB.GetBookByShortName(ctx, trans.VersificationID, bookShort)
	if err != nil {
		writeError(w, http.StatusNotFound, "book not found: "+bookShort)
		return
	}

	writeJSON(w, http.StatusOK, BookResponse{
		ID:           b.ID,
		Name:         b.Name,
		ShortName:    b.ShortName,
		Testament:    b.Testament,
		BookOrder:    b.BookOrder,
		ChapterCount: b.ChapterCount,
	})
}

// @Summary      Get a chapter's verses
// @Tags         chapters
// @Produce      json
// @Param        t       path  string  true  "Translation short name"
// @Param        book    query  string  true  "Book short name (e.g. Gen)"
// @Param        chapter path  int     true  "Chapter number"
// @Success      200  {object}  object
// @Failure      404  {object}  object
// @Router       /api/chapters/{t}/{chapter} [get]
func (h *Handler) GetChapter(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	transShort := r.PathValue("t")
	bookShort := r.URL.Query().Get("book")
	chapterStr := r.PathValue("chapter")

	if bookShort == "" {
		writeError(w, http.StatusBadRequest, "query parameter 'book' is required")
		return
	}

	chapter, err := strconv.Atoi(chapterStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid chapter number")
		return
	}

	trans, err := h.DB.GetTranslationByShortName(ctx, transShort)
	if err != nil {
		writeError(w, http.StatusNotFound, "translation not found: "+transShort)
		return
	}

	b, err := h.DB.GetBookByShortName(ctx, trans.VersificationID, bookShort)
	if err != nil {
		writeError(w, http.StatusNotFound, "book not found: "+bookShort)
		return
	}

	cacheKey := h.Cache.ChapterKey(transShort, b.BookOrder, chapter)
	if h.Cache != nil {
		var cached ChapterResponse
		if ok, _ := h.Cache.Get(ctx, cacheKey, &cached); ok {
			writeJSON(w, http.StatusOK, cached)
			return
		}
	}

	verses, err := h.DB.GetChapterVerses(ctx, trans.ID, b.ID, chapter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	items := make([]VerseItem, len(verses))
	for i := range verses {
		items[i] = VerseItem{Verse: verses[i].Verse, Text: verses[i].Text}
	}

	resp := ChapterResponse{
		Translation: transShort,
		BookName:    b.Name,
		BookShort:   b.ShortName,
		Chapter:     chapter,
		VersesCount: len(items),
		Verses:      items,
	}

	if h.Cache != nil {
		_ = h.Cache.Set(ctx, cacheKey, resp, 0)
	}

	writeJSON(w, http.StatusOK, resp)
}

type ChapterResponse struct {
	Translation string      `json:"translation"`
	BookName    string      `json:"book_name"`
	BookShort   string      `json:"book_short"`
	Chapter     int         `json:"chapter"`
	VersesCount int         `json:"verses_count"`
	Verses      []VerseItem `json:"verses"`
}

type VerseItem struct {
	Verse int    `json:"verse"`
	Text  string `json:"text"`
}
