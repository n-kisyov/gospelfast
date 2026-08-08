package osis

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type BookInfo struct {
	OSISID       string
	Name         string
	ChapterCount int
}

type ParsedData struct {
	Title string
	Books []BookInfo
}

type VerseHandler func(bookOSISID string, chapter int, verse int, text string) error

type parser struct {
	decoder *xml.Decoder
	handler VerseHandler
	data    ParsedData

	currentBook    string
	currentChapter int
	currentVerse   int
	inMilestone    bool
	milestoneBuf   strings.Builder
	inTitle        bool
	titleBuf       strings.Builder
}

func Parse(r io.Reader, handler VerseHandler) (*ParsedData, error) {
	p := &parser{
		decoder: xml.NewDecoder(r),
		handler: handler,
	}
	if err := p.walk(); err != nil {
		return nil, err
	}
	return &p.data, nil
}

func (p *parser) walk() error {
	for {
		tok, err := p.decoder.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("xml token: %w", err)
		}

		switch t := tok.(type) {
		case xml.StartElement:
			p.handleStart(t)
		case xml.EndElement:
			p.handleEnd(t)
		case xml.CharData:
			p.handleChars(string(t))
		}
	}
}

func (p *parser) handleStart(e xml.StartElement) {
	name := stripNS(e.Name.Local)

	switch name {
	case "osisText":
		if v := attr(e, "osisIDWork"); v != "" {
			p.data.Title = v
		}
	case "div":
		if attr(e, "type") == "book" {
			p.currentBook = attr(e, "osisID")
			if p.currentBook == "" {
				p.currentBook = attr(e, "annotateRef")
				p.currentBook = strings.TrimPrefix(p.currentBook, "Bible.")
			}
			p.data.Books = append(p.data.Books, BookInfo{OSISID: p.currentBook})
		}
	case "chapter":
		if p.currentBook != "" {
			osisID := attr(e, "osisID")
			p.currentChapter = extractChapter(osisID)
			if p.currentChapter == 0 {
				p.currentChapter = extractChapterNum(e)
			}
			if p.currentChapter > 0 && len(p.data.Books) > 0 {
				last := &p.data.Books[len(p.data.Books)-1]
				if p.currentChapter > last.ChapterCount {
					last.ChapterCount = p.currentChapter
				}
			}
		}
	case "title":
		p.inTitle = true
		p.titleBuf.Reset()
	case "verse":
		if sID := attr(e, "sID"); sID != "" {
			p.currentVerse = extractVerse(sID)
			p.inMilestone = true
			p.milestoneBuf.Reset()
			return
		}
		if eID := attr(e, "eID"); eID != "" {
			if p.inMilestone {
				p.inMilestone = false
				if p.handler != nil && p.currentVerse > 0 {
					_ = p.handler(p.currentBook, p.currentChapter, p.currentVerse,
						strings.TrimSpace(p.milestoneBuf.String()))
				}
			}
			p.currentVerse = 0
			return
		}
		osisID := attr(e, "osisID")
		if osisID != "" {
			p.currentVerse = extractVerse(osisID)
		}
	}
}

func (p *parser) handleEnd(e xml.EndElement) {
	name := stripNS(e.Name.Local)

	switch name {
	case "div":
		p.currentBook = ""
		p.currentChapter = 0
		p.currentVerse = 0
	case "chapter":
		p.currentChapter = 0
		p.currentVerse = 0
	case "title":
		if p.inTitle {
			p.inTitle = false
			title := strings.TrimSpace(p.titleBuf.String())
			if title != "" && len(p.data.Books) > 0 {
				last := &p.data.Books[len(p.data.Books)-1]
				if last.Name == "" {
					last.Name = title
				}
			}
		}
	case "verse":
		if !p.inMilestone {
			p.currentVerse = 0
		}
	}
}

func (p *parser) handleChars(data string) {
	if p.inMilestone {
		p.milestoneBuf.WriteString(data)
		return
	}
	if p.inTitle {
		p.titleBuf.WriteString(data)
		return
	}
	if p.currentVerse > 0 && p.handler != nil {
		_ = p.handler(p.currentBook, p.currentChapter, p.currentVerse, strings.TrimSpace(data))
		p.currentVerse = 0
	}
}

func attr(e xml.StartElement, name string) string {
	for _, a := range e.Attr {
		if stripNS(a.Name.Local) == name {
			return a.Value
		}
	}
	return ""
}

func stripNS(name string) string {
	if idx := strings.Index(name, ":"); idx != -1 {
		return name[idx+1:]
	}
	return name
}

func extractChapter(osisID string) int {
	parts := strings.SplitN(osisID, ".", 2)
	if len(parts) < 2 {
		return 0
	}
	n, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0
	}
	return n
}

func extractChapterNum(e xml.StartElement) int {
	s := attr(e, "n")
	if s == "" {
		s = attr(e, "chapter")
	}
	if s == "" {
		return 0
	}
	n, _ := strconv.Atoi(s)
	return n
}

func extractVerse(osisID string) int {
	parts := strings.SplitN(osisID, ".", 3)
	if len(parts) < 3 {
		return 0
	}
	n, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0
	}
	return n
}
