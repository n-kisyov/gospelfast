package search

import (
	"context"
	"strings"

	"github.com/gospelfast/gospelfast/internal/db"
)

type Result struct {
	ID             int64   `json:"id"`
	BookName       string  `json:"book_name"`
	BookShortName  string  `json:"book_short_name"`
	Chapter        int     `json:"chapter"`
	Verse          int     `json:"verse"`
	Text           string  `json:"text"`
	Snippet        string  `json:"snippet"`
	Rank           float64 `json:"rank"`
}

type Results struct {
	Query      string   `json:"query"`
	Total      int      `json:"total"`
	Page       int      `json:"page"`
	PerPage    int      `json:"per_page"`
	Results    []Result `json:"results"`
}

const defaultPerPage = 30

func Query(ctx context.Context, d *db.DB, translationID, query string, page, perPage int) (*Results, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return &Results{Query: "", Total: 0, Page: page, PerPage: perPage}, nil
	}
	if perPage <= 0 {
		perPage = defaultPerPage
	}
	if page <= 0 {
		page = 1
	}

	offset := (page - 1) * perPage

	dbResults, total, err := d.SearchVerses(ctx, translationID, query, perPage, offset)
	if err != nil {
		return nil, err
	}

	results := make([]Result, len(dbResults))
	for i, r := range dbResults {
		results[i] = Result{
			ID:            r.ID,
			BookName:      r.BookName,
			BookShortName: r.BookShortName,
			Chapter:       r.Chapter,
			Verse:         r.Verse,
			Text:          r.Text,
			Snippet:       r.Snippet,
			Rank:          r.Rank,
		}
	}

	return &Results{
		Query:   query,
		Total:   total,
		Page:    page,
		PerPage: perPage,
		Results: results,
	}, nil
}
