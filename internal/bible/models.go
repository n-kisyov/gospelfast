package bible

import (
	"encoding/json"
	"time"
)

type Versification struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Book struct {
	ID               string `json:"id"`
	VersificationID  string `json:"versification_id"`
	Name             string `json:"name"`
	ShortName        string `json:"short_name"`
	Testament        string `json:"testament"`
	BookOrder        int    `json:"book_order"`
	ChapterCount     int    `json:"chapter_count"`
}

type Translation struct {
	ID               string          `json:"id"`
	ShortName        string          `json:"short_name"`
	FullName         string          `json:"full_name"`
	Language         string          `json:"language"`
	ModuleType       string          `json:"module_type"`
	VersificationID  string          `json:"versification_id"`
	Description      string          `json:"description,omitempty"`
	SourceURL        string          `json:"source_url,omitempty"`
	Copyright        string          `json:"copyright,omitempty"`
	Metadata         json.RawMessage `json:"metadata,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type Verse struct {
	ID             int64  `json:"id"`
	TranslationID  string `json:"translation_id"`
	BookID         string `json:"book_id"`
	Chapter        int    `json:"chapter"`
	Verse          int    `json:"verse"`
	Text           string `json:"text"`
	BookName       string `json:"book_name,omitempty"`
	BookShortName  string `json:"book_short_name,omitempty"`
}
