package vpl

import (
	"bufio"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/gospelfast/gospelfast/internal/bible"
)

var tagRe = regexp.MustCompile(`<[^>]*>`) 

type VerseRec struct {
	BookShort string
	Chapter   int
	Verse     int
	Text      string
}

func Parse(r io.Reader) ([]VerseRec, error) {
	var verses []VerseRec
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		ref, err := bible.ParseRef(line)
		if err != nil {
			continue
		}

		// Extract the text after the reference
		spaceIdx := strings.Index(line, " ")
		if spaceIdx < 0 {
			continue
		}
		colonIdx := strings.Index(line, ":")
		if colonIdx < 0 {
			continue
		}
		textStart := colonIdx + 1
		for textStart < len(line) && line[textStart] >= '0' && line[textStart] <= '9' {
			textStart++
		}
		if textStart < len(line) && line[textStart] == ' ' {
			textStart++
		}
		text := strings.TrimSpace(line[textStart:])
		text = tagRe.ReplaceAllString(text, "")
		text = cleanUTF8(text)
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}

		verses = append(verses, VerseRec{
			BookShort: ref.BookShortName,
			Chapter:   ref.Chapter,
			Verse:     ref.Verse,
			Text:      text,
		})
	}

	return verses, scanner.Err()
}

func cleanUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	return strings.ToValidUTF8(s, " ")
}
