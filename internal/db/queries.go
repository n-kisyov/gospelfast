package db

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/gospelfast/gospelfast/internal/bible"
	"github.com/jackc/pgx/v5"
)

func (db *DB) CreateTranslation(ctx context.Context, t *bible.Translation) error {
	return db.Pool.QueryRow(ctx, `
		INSERT INTO translations (short_name, full_name, language, module_type, versification_id, description, source_url, copyright, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (short_name) DO UPDATE SET full_name = EXCLUDED.full_name, language = EXCLUDED.language,
		    module_type = EXCLUDED.module_type, versification_id = EXCLUDED.versification_id,
		    description = EXCLUDED.description, source_url = EXCLUDED.source_url,
		    copyright = EXCLUDED.copyright, metadata = EXCLUDED.metadata, updated_at = EXCLUDED.updated_at
		RETURNING id
	`, t.ShortName, t.FullName, t.Language, t.ModuleType, t.VersificationID,
		t.Description, t.SourceURL, t.Copyright, t.Metadata,
		t.CreatedAt, t.UpdatedAt).Scan(&t.ID)
}

func (db *DB) GetTranslation(ctx context.Context, id string) (*bible.Translation, error) {
	t := &bible.Translation{}
	err := db.Pool.QueryRow(ctx, `
		SELECT id, short_name, full_name, language, COALESCE(module_type, 'bible'), versification_id, description, source_url, copyright, metadata, created_at, updated_at
		FROM translations WHERE id = $1
	`, id).Scan(&t.ID, &t.ShortName, &t.FullName, &t.Language, &t.ModuleType, &t.VersificationID,
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
		SELECT id, short_name, full_name, language, COALESCE(module_type, 'bible'), versification_id, description, source_url, copyright, metadata, created_at, updated_at
		FROM translations WHERE short_name = $1
	`, shortName).Scan(&t.ID, &t.ShortName, &t.FullName, &t.Language, &t.ModuleType, &t.VersificationID,
		&t.Description, &t.SourceURL, &t.Copyright, &t.Metadata,
		&t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (db *DB) ListTranslations(ctx context.Context) ([]bible.Translation, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, short_name, full_name, language, COALESCE(module_type, 'bible'), versification_id, description, source_url, copyright, metadata, created_at, updated_at
		FROM translations ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []bible.Translation
	for rows.Next() {
		var t bible.Translation
		if err := rows.Scan(&t.ID, &t.ShortName, &t.FullName, &t.Language, &t.ModuleType, &t.VersificationID,
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
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO books (versification_id, name, short_name, testament, book_order, chapter_count)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (versification_id, short_name) DO UPDATE SET
			name = EXCLUDED.name, testament = EXCLUDED.testament,
			book_order = EXCLUDED.book_order, chapter_count = EXCLUDED.chapter_count
		RETURNING id
	`, book.VersificationID, book.Name, book.ShortName, book.Testament, book.BookOrder, book.ChapterCount).Scan(&book.ID)

	if err == nil {
		return nil
	}

	if strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "duplicate key") {
		return db.Pool.QueryRow(ctx, `
			SELECT id FROM books WHERE versification_id = $1 AND short_name = $2
		`, book.VersificationID, book.ShortName).Scan(&book.ID)
	}

	return err
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

type GenbookEntry struct {
	ID            string `json:"id"`
	TranslationID string `json:"translation_id"`
	Path          string `json:"path"`
	Title         string `json:"title"`
	Content       string `json:"content"`
}

func (db *DB) InsertGenbookEntries(ctx context.Context, translationID string, entries []GenbookEntry) (int64, error) {
	return db.Pool.CopyFrom(
		ctx,
		pgx.Identifier{"genbooks"},
		[]string{"translation_id", "path", "title", "content"},
		pgx.CopyFromSlice(len(entries), func(i int) ([]any, error) {
			e := entries[i]
			return []any{translationID, e.Path, e.Title, e.Content}, nil
		}),
	)
}

func (db *DB) GetGenbookPaths(ctx context.Context, translationID, parentPath string) ([]GenbookEntry, error) {
	likeChild := parentPath + "/%"
	likeGrandchild := parentPath + "/%/%"
	if parentPath == "/" || parentPath == "" {
		likeChild = "/%"
		likeGrandchild = "/%/%"
	}
	rows, err := db.Pool.Query(ctx, `
		SELECT id, translation_id, path, COALESCE(title, ''), content
		FROM genbooks
		WHERE translation_id = $1 AND path LIKE $2 AND path NOT LIKE $3
		ORDER BY path
	`, translationID, likeChild, likeGrandchild)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []GenbookEntry
	for rows.Next() {
		var e GenbookEntry
		if err := rows.Scan(&e.ID, &e.TranslationID, &e.Path, &e.Title, &e.Content); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func (db *DB) GetGenbookEntry(ctx context.Context, translationID, path string) (*GenbookEntry, error) {
	e := &GenbookEntry{}
	err := db.Pool.QueryRow(ctx, `
		SELECT id, translation_id, path, COALESCE(title, ''), content
		FROM genbooks WHERE translation_id = $1 AND path = $2
	`, translationID, path).Scan(&e.ID, &e.TranslationID, &e.Path, &e.Title, &e.Content)
	if err != nil {
		return nil, err
	}
	return e, nil
}

type CommentaryEntry struct {
	ID            int64  `json:"id"`
	TranslationID string `json:"translation_id"`
	BookID        string `json:"book_id"`
	BookName      string `json:"book_name,omitempty"`
	Chapter       int    `json:"chapter"`
	Verse         int    `json:"verse"`
	Content       string `json:"content"`
}

func (db *DB) InsertCommentaryEntries(ctx context.Context, translationID string, entries []CommentaryEntry) (int64, error) {
	return db.Pool.CopyFrom(
		ctx,
		pgx.Identifier{"commentary_entries"},
		[]string{"translation_id", "book_id", "chapter", "verse", "content"},
		pgx.CopyFromSlice(len(entries), func(i int) ([]any, error) {
			e := entries[i]
			return []any{translationID, e.BookID, e.Chapter, e.Verse, e.Content}, nil
		}),
	)
}

func (db *DB) GetCommentary(ctx context.Context, translationID, bookID string, chapter, verse int) (*CommentaryEntry, error) {
	e := &CommentaryEntry{}
	err := db.Pool.QueryRow(ctx, `
		SELECT c.id, c.translation_id, c.book_id, b.name, c.chapter, c.verse, c.content
		FROM commentary_entries c
		JOIN books b ON b.id = c.book_id
		WHERE c.translation_id = $1 AND c.book_id = $2 AND c.chapter = $3 AND c.verse = $4
	`, translationID, bookID, chapter, verse).Scan(
		&e.ID, &e.TranslationID, &e.BookID, &e.BookName, &e.Chapter, &e.Verse, &e.Content,
	)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (db *DB) ListCommentariesForBook(ctx context.Context, translationID, bookID string) ([]CommentaryEntry, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT c.id, c.translation_id, c.book_id, b.name, c.chapter, c.verse, c.content
		FROM commentary_entries c
		JOIN books b ON b.id = c.book_id
		WHERE c.translation_id = $1 AND c.book_id = $2
		ORDER BY c.chapter, c.verse
	`, translationID, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []CommentaryEntry
	for rows.Next() {
		var e CommentaryEntry
		if err := rows.Scan(&e.ID, &e.TranslationID, &e.BookID, &e.BookName, &e.Chapter, &e.Verse, &e.Content); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

func (db *DB) CreateUser(ctx context.Context, username, passwordHash, role string) (*User, error) {
	u := &User{}
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO users (username, password_hash, role) VALUES ($1, $2, $3)
		RETURNING id, username, role, created_at
	`, username, passwordHash, role).Scan(&u.ID, &u.Username, &u.Role, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (db *DB) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	u := &User{Username: username}
	err := db.Pool.QueryRow(ctx, `
		SELECT id, password_hash, role, created_at FROM users WHERE username = $1
	`, username).Scan(&u.ID, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (db *DB) CreateSession(ctx context.Context, userID string, expiresAt time.Time) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)

	_, err := db.Pool.Exec(ctx, `
		INSERT INTO sessions (token, user_id, expires_at) VALUES ($1, $2, $3)
	`, token, userID, expiresAt)
	if err != nil {
		return "", err
	}
	return token, nil
}

func (db *DB) GetSession(ctx context.Context, token string) (*User, error) {
	u := &User{}
	err := db.Pool.QueryRow(ctx, `
		SELECT u.id, u.username, u.role, u.created_at
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.token = $1 AND s.expires_at > NOW()
	`, token).Scan(&u.ID, &u.Username, &u.Role, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (db *DB) DeleteSession(ctx context.Context, token string) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM sessions WHERE token = $1`, token)
	return err
}

func (db *DB) CleanSessions(ctx context.Context) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM sessions WHERE expires_at < NOW()`)
	return err
}

func (db *DB) UserCount(ctx context.Context) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}

func (db *DB) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := db.Pool.Query(ctx, `SELECT id, username, role, created_at FROM users ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (db *DB) DeleteUser(ctx context.Context, id string) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}

// GetRandomVerse deterministically picks a verse for the given seed (e.g. a
// hash of today's date, so "verse of the day" stays stable all day) while
// spreading picks uniformly across the whole translation. It previously
// ordered by (id*seed) % 7919, which — since 7919 is prime and virtually
// every seed is coprime to it — collapsed to always favoring the handful of
// verses whose id happens to be a multiple of 7919, regardless of seed.
func (db *DB) GetRandomVerse(ctx context.Context, translationID string, seed int64) (*bible.Verse, error) {
	var count int64
	if err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM verses WHERE translation_id = $1
	`, translationID).Scan(&count); err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, pgx.ErrNoRows
	}

	if seed < 0 {
		seed = -seed
	}
	offset := seed % count

	v := &bible.Verse{}
	err := db.Pool.QueryRow(ctx, `
		SELECT v.id, v.translation_id, v.book_id, v.chapter, v.verse, v.text, b.name, b.short_name
		FROM verses v JOIN books b ON b.id = v.book_id
		WHERE v.translation_id = $1
		ORDER BY v.id
		OFFSET $2 LIMIT 1
	`, translationID, offset).Scan(&v.ID, &v.TranslationID, &v.BookID, &v.Chapter, &v.Verse, &v.Text, &v.BookName, &v.BookShortName)
	if err != nil {
		return nil, err
	}
	return v, nil
}

