package osis

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestParseContainerFormat(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "test_osis.xml"))
	if err != nil {
		t.Fatal(err)
	}

	type verseRec struct {
		book    string
		chapter int
		verse   int
		text    string
	}
	var verses []verseRec

	result, err := Parse(bytes.NewReader(data), func(book string, chapter, verse int, text string) error {
		verses = append(verses, verseRec{book, chapter, verse, text})
		return nil
	})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result.Title != "TEST" {
		t.Errorf("Title = %q, want TEST", result.Title)
	}
	if len(result.Books) != 2 {
		t.Fatalf("got %d books, want 2", len(result.Books))
	}
	if result.Books[0].OSISID != "Gen" {
		t.Errorf("book[0].OSISID = %q, want Gen", result.Books[0].OSISID)
	}
	if result.Books[0].Name != "Genesis" {
		t.Errorf("book[0].Name = %q, want Genesis", result.Books[0].Name)
	}
	if result.Books[0].ChapterCount != 2 {
		t.Errorf("book[0].ChapterCount = %d, want 2", result.Books[0].ChapterCount)
	}
	if result.Books[1].OSISID != "John" {
		t.Errorf("book[1].OSISID = %q, want John", result.Books[1].OSISID)
	}
	if len(verses) != 9 {
		t.Fatalf("got %d verses, want 9", len(verses))
	}
	if verses[0].chapter != 1 || verses[0].verse != 1 {
		t.Errorf("first verse: chapter=%d verse=%d", verses[0].chapter, verses[0].verse)
	}
	if verses[8].book != "John" || verses[8].chapter != 3 || verses[8].verse != 17 {
		t.Errorf("last verse: book=%s ch=%d v=%d", verses[8].book, verses[8].chapter, verses[8].verse)
	}
}

// TestParseContainerFormatWithNestedMarkup guards against a bug where only
// the first CharData run of a container-style <verse> was kept: text after
// a nested inline element (<note>, <transChange>, <seg>, ...) was silently
// dropped because currentVerse got reset after the first chunk fired the
// handler.
func TestParseContainerFormatWithNestedMarkup(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "test_osis_nested.xml"))
	if err != nil {
		t.Fatal(err)
	}

	type verseRec struct {
		chapter int
		verse   int
		text    string
	}
	var verses []verseRec

	_, err = Parse(bytes.NewReader(data), func(book string, chapter, verse int, text string) error {
		verses = append(verses, verseRec{chapter, verse, text})
		return nil
	})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(verses) != 2 {
		t.Fatalf("got %d verses, want 2", len(verses))
	}
	want0 := "In the very beginning God created the heaven and the earth."
	if verses[0].text != want0 {
		t.Errorf("verse[0].text = %q, want %q", verses[0].text, want0)
	}
	want1 := "And the earth was without form, and void."
	if verses[1].text != want1 {
		t.Errorf("verse[1].text = %q, want %q", verses[1].text, want1)
	}
}

func TestParseMilestoneFormat(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "test_osis_milestone.xml"))
	if err != nil {
		t.Fatal(err)
	}

	type verseRec struct {
		book    string
		chapter int
		verse   int
		text    string
	}
	var verses []verseRec

	result, err := Parse(bytes.NewReader(data), func(book string, chapter, verse int, text string) error {
		verses = append(verses, verseRec{book, chapter, verse, text})
		return nil
	})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(verses) != 3 {
		t.Fatalf("got %d verses, want 3", len(verses))
	}
	if verses[0].text != "In the beginning God created the heaven and the earth." {
		t.Errorf("verse[0] = %q", verses[0].text)
	}
	if verses[2].text != "And God said, Let there be light: and there was light." {
		t.Errorf("verse[2] = %q", verses[2].text)
	}
	if len(result.Books) != 1 {
		t.Errorf("got %d books, want 1", len(result.Books))
	}
	if result.Books[0].OSISID != "Gen" {
		t.Errorf("book[0].OSISID = %q, want Gen", result.Books[0].OSISID)
	}
	if result.Books[0].ChapterCount != 1 {
		t.Errorf("book[0].ChapterCount = %d, want 1", result.Books[0].ChapterCount)
	}
}
