package seed

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gospelfast/gospelfast/internal/bible"
	importpkg "github.com/gospelfast/gospelfast/internal/import"
)

const apiBase = "https://bible-api.com"

type chapterVerses struct {
	Verses []struct {
		BookName string `json:"book_name"`
		Chapter  int    `json:"chapter"`
		Verse    int    `json:"verse"`
		Text     string `json:"text"`
	} `json:"verses"`
}

func DownloadKJV(ctx context.Context, p *importpkg.Pipeline, progress importpkg.ProgressFn) error {
	tmpDir, err := os.MkdirTemp("", "gospelfast-seed-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	vplPath := filepath.Join(tmpDir, "kjv.vpl")
	f, err := os.Create(vplPath)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(f)

	var totalChapters, verseCount int
	for _, b := range bible.KJVBooks {
		totalChapters += b.ChapterCount
	}

	client := &http.Client{Timeout: 30 * time.Second}
	done := 0

	for _, book := range bible.KJVBooks {
		for ch := 1; ch <= book.ChapterCount; ch++ {
			select {
			case <-ctx.Done():
				w.Flush()
				f.Close()
				return ctx.Err()
			default:
			}

			data, err := fetchChapter(client, book.Name, ch)
			if err != nil {
				log.Printf("seed: %s %d: %v", book.Name, ch, err)
				time.Sleep(500 * time.Millisecond)
				continue
			}

			for _, v := range data.Verses {
				fmt.Fprintf(w, "%s %d:%d %s\n", book.Name, ch, v.Verse, v.Text)
				verseCount++
			}

			done++
			if progress != nil && done%10 == 0 {
				progress("downloading KJV", done, totalChapters)
			}

			time.Sleep(1500 * time.Millisecond)
		}
	}

	w.Flush()
	f.Close()

	if verseCount == 0 {
		return fmt.Errorf("seed: downloaded 0 verses")
	}

	log.Printf("Seed: downloaded %d verses from %d chapters", verseCount, done)

	if progress != nil {
		progress("importing KJV verses", 0, verseCount)
	}

	return p.ImportFile(ctx, vplPath, "KJV", importpkg.FormatVPL, progress)
}

func fetchChapter(client *http.Client, bookName string, chapter int) (*chapterVerses, error) {
	url := fmt.Sprintf("%s/%s%%20%d?translation=kjv", apiBase, urlBook(bookName), chapter)

	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt*5) * time.Second)
		}

		resp, err := client.Get(url)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode == http.StatusOK {
			var data chapterVerses
			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				resp.Body.Close()
				lastErr = err
				continue
			}
			resp.Body.Close()
			return &data, nil
		}

		body, _ := io.ReadAll(io.LimitReader(resp.Body, 200))
		resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			wait := 30 * time.Second
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if s, err := time.ParseDuration(ra + "s"); err == nil {
					wait = s
				}
			}
			log.Printf("seed: rate limited on %s %d, waiting %v...", bookName, chapter, wait)
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
			time.Sleep(wait)
			continue
		}

		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return nil, lastErr
}

func urlBook(name string) string {
	return strings.ReplaceAll(name, " ", "%20")
}
