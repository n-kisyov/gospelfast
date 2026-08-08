package db

import (
	"context"

	"github.com/gospelfast/gospelfast/internal/bible"
	"github.com/jackc/pgx/v5"
)

func (db *DB) CreateTranslation(ctx context.Context, t *bible.Translation) error {
	return db.Pool.QueryRow(ctx, `
		INSERT INTO translations (short_name, full_name, language, versification_id, description, source_url, copyright, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (short_name) DO UPDATE SET full_name = EXCLUDED.full_name, language = EXCLUDED.language,
		    versification_id = EXCLUDED.versification_id, description = EXCLUDED.description,
		    source_url = EXCLUDED.source_url, copyright = EXCLUDED.copyright,
		    metadata = EXCLUDED.metadata, updated_at = EXCLUDED.updated_at
		RETURNING id
	`, t.ShortName, t.FullName, t.Language, t.VersificationID,
		t.Description, t.SourceURL, t.Copyright, t.Metadata,
		t.CreatedAt, t.UpdatedAt).Scan(&t.ID)
}

func (db *DB) GetTranslation(ctx context.Context, id string) (*bible.Translation, error) {
	t := &bible.Translation{}
	err := db.Pool.QueryRow(ctx, `
		SELECT id, short_name, full_name, language, versification_id, description, source_url, copyright, metadata, created_at, updated_at
		FROM translations WHERE id = $1
	`, id).Scan(&t.ID, &t.ShortName, &t.FullName, &t.Language, &t.VersificationID,
		&t.Description, &t.SourceURL, &t.Copyright, &t.Metadata,
		&t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (db *DB) GetTranslationByShortName(ctx context.Context, shortName string) (*bible.Translation, error) {
	t := &bible.Translation{}
	err := db.Pool.QueryRow(ctx, `
		SELECT id, short_name, full_name, language, versification_id, description, source_url, copyright, metadata, created_at, updated_at
		FROM translations WHERE short_name = $1
	`, shortName).Scan(&t.ID, &t.ShortName, &t.FullName, &t.Language, &t.VersificationID,
		&t.Description, &t.SourceURL, &t.Copyright, &t.Metadata,
		&t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (db *DB) ListTranslations(ctx context.Context) ([]bible.Translation, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, short_name, full_name, language, versification_id, description, source_url, copyright, metadata, created_at, updated_at
		FROM translations ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []bible.Translation
	for rows.Next() {
		var t bible.Translation
		if err := rows.Scan(&t.ID, &t.ShortName, &t.FullName, &t.Language, &t.VersificationID,
			&t.Description, &t.SourceURL, &t.Copyright, &t.Metadata,
			&t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, nil
}

func (db *DB) DeleteTranslation(ctx context.Context, id string) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM translations WHERE id = $1`, id)
	return err
}

func (db *DB) DeleteTranslationByShortName(ctx context.Context, shortName string) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM translations WHERE short_name = $1`, shortName)
	return err
}

func (db *DB) CreateVersification(ctx context.Context, name string) (string, error) {
	var id string
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO versifications (name) VALUES ($1) ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name RETURNING id
	`, name).Scan(&id)
	return id, err
}

func (db *DB) GetVersificationByName(ctx context.Context, name string) (string, error) {
	var id string
	err := db.Pool.QueryRow(ctx, `SELECT id FROM versifications WHERE name = $1`, name).Scan(&id)
	return id, err
}

func (db *DB) CreateBook(ctx context.Context, book *bible.Book) error {
	return db.Pool.QueryRow(ctx, `
		INSERT INTO books (versification_id, name, short_name, testament, book_order, chapter_count)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (versification_id, short_name) DO UPDATE SET
			name = EXCLUDED.name, testament = EXCLUDED.testament,
			book_order = EXCLUDED.book_order, chapter_count = EXCLUDED.chapter_count
		RETURNING id
	`, book.VersificationID, book.Name, book.ShortName, book.Testament, book.BookOrder, book.ChapterCount).Scan(&book.ID)
}

func (db *DB) GetBooksByVersification(ctx context.Context, versificationID string) ([]bible.Book, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, versification_id, name, short_name, testament, book_order, chapter_count
		FROM books WHERE versification_id = $1 ORDER BY book_order
	`, versificationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []bible.Book
	for rows.Next() {
		var b bible.Book
		if err := rows.Scan(&b.ID, &b.VersificationID, &b.Name, &b.ShortName, &b.Testament, &b.BookOrder, &b.ChapterCount); err != nil {
			return nil, err
		}
		books = append(books, b)
	}
	return books, nil
}

func (db *DB) GetBookByShortName(ctx context.Context, versificationID, shortName string) (*bible.Book, error) {
	b := &bible.Book{}
	err := db.Pool.QueryRow(ctx, `
		SELECT id, versification_id, name, short_name, testament, book_order, chapter_count
		FROM books WHERE versification_id = $1 AND short_name = $2
	`, versificationID, shortName).Scan(&b.ID, &b.VersificationID, &b.Name, &b.ShortName, &b.Testament, &b.BookOrder, &b.ChapterCount)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (db *DB) InsertVerses(ctx context.Context, verses []bible.Verse) (int64, error) {
	return db.Pool.CopyFrom(
		ctx,
		pgx.Identifier{"verses"},
		[]string{"translation_id", "book_id", "chapter", "verse", "text"},
		pgx.CopyFromSlice(len(verses), func(i int) ([]any, error) {
			v := verses[i]
			return []any{v.TranslationID, v.BookID, v.Chapter, v.Verse, v.Text}, nil
		}),
	)
}

func (db *DB) GetVerse(ctx context.Context, translationID, bookID string, chapter, verse int) (*bible.Verse, error) {
	v := &bible.Verse{}
	err := db.Pool.QueryRow(ctx, `
		SELECT v.id, v.translation_id, v.book_id, v.chapter, v.verse, v.text, b.name, b.short_name
		FROM verses v
		JOIN books b ON b.id = v.book_id
		WHERE v.translation_id = $1 AND v.book_id = $2 AND v.chapter = $3 AND v.verse = $4
	`, translationID, bookID, chapter, verse).Scan(&v.ID, &v.TranslationID, &v.BookID, &v.Chapter, &v.Verse, &v.Text, &v.BookName, &v.BookShortName)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (db *DB) GetChapterVerses(ctx context.Context, translationID, bookID string, chapter int) ([]bible.Verse, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT v.id, v.translation_id, v.book_id, v.chapter, v.verse, v.text, b.name, b.short_name
		FROM verses v
		JOIN books b ON b.id = v.book_id
		WHERE v.translation_id = $1 AND v.book_id = $2 AND v.chapter = $3
		ORDER BY v.verse
	`, translationID, bookID, chapter)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var verses []bible.Verse
	for rows.Next() {
		var v bible.Verse
		if err := rows.Scan(&v.ID, &v.TranslationID, &v.BookID, &v.Chapter, &v.Verse, &v.Text, &v.BookName, &v.BookShortName); err != nil {
			return nil, err
		}
		verses = append(verses, v)
	}
	return verses, nil
}

type SearchResult struct {
	ID             int64   `json:"id"`
	BookID         string  `json:"book_id"`
	BookName       string  `json:"book_name"`
	BookShortName  string  `json:"book_short_name"`
	Chapter        int     `json:"chapter"`
	Verse          int     `json:"verse"`
	Text           string  `json:"text"`
	Snippet        string  `json:"snippet"`
	Rank           float64 `json:"rank"`
}

func (db *DB) SearchVerses(ctx context.Context, translationID, query string, limit, offset int) ([]SearchResult, int, error) {
	var total int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM verses
		WHERE translation_id = $1 AND tsv @@ plainto_tsquery('english', $2)
	`, translationID, query).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT v.id, v.book_id, b.name, b.short_name, v.chapter, v.verse, v.text,
		       ts_headline('english', v.text, plainto_tsquery('english', $2),
		           'StartSel=<mark>, StopSel=</mark>, MaxWords=40, MinWords=10') AS snippet,
		       ts_rank(v.tsv, plainto_tsquery('english', $2)) AS rank
		FROM verses v
		JOIN books b ON b.id = v.book_id
		WHERE v.translation_id = $1 AND v.tsv @@ plainto_tsquery('english', $2)
		ORDER BY rank DESC
		LIMIT $3 OFFSET $4
	`, translationID, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ID, &r.BookID, &r.BookName, &r.BookShortName, &r.Chapter, &r.Verse, &r.Text, &r.Snippet, &r.Rank); err != nil {
			return nil, 0, err
		}
		results = append(results, r)
	}
	return results, total, nil
}

