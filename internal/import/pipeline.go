package importer

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gospelfast/gospelfast/internal/bible"
	"github.com/gospelfast/gospelfast/internal/db"
	"github.com/gospelfast/gospelfast/internal/import/osis"
	"github.com/gospelfast/gospelfast/internal/import/sword"
	"github.com/gospelfast/gospelfast/internal/import/vpl"
)

type Format int

const (
	FormatOSIS   Format = iota
	FormatRawzip
	FormatVPL
)

type ProgressFn func(stage string, current, total int)

type Pipeline struct {
	DB *db.DB
}

func New(db *db.DB) *Pipeline {
	return &Pipeline{DB: db}
}

func (p *Pipeline) ImportFile(ctx context.Context, source, name string, format Format, progress ProgressFn) error {
	tmpDir, err := os.MkdirTemp("", "gospelfast-import-*")
	if err != nil {
		return fmt.Errorf("temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		downloaded := filepath.Join(tmpDir, "download")
		if progress != nil {
			progress("downloading "+source, 0, 0)
		}
		client := &http.Client{Timeout: 60 * time.Second}
		req, err := http.NewRequestWithContext(ctx, "GET", source, nil)
		if err != nil {
			return fmt.Errorf("download: %w", err)
		}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("download: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("download: HTTP %d", resp.StatusCode)
		}
		f, err := os.Create(downloaded)
		if err != nil {
			return err
		}
		if _, err := io.Copy(f, resp.Body); err != nil {
			f.Close()
			return err
		}
		f.Close()
		source = downloaded
		if progress != nil {
			progress("download complete", 100, 100)
		}
	}

	versification := "KJV"

	if format == FormatRawzip {
		if progress != nil {
			progress("downloading and converting rawzip", 0, 0)
		}
		var meta *sword.ModuleMeta
		convertedPath, meta, err := sword.ConvertRawzip(source, tmpDir)
		if err != nil {
			return fmt.Errorf("rawzip conversion: %w", err)
		}
		if meta != nil {
			if name == "" {
				name = meta.ModName
			}
			if meta.Versification != "" {
				versification = meta.Versification
			}
		}
		if name == "" {
			name = filepath.Base(strings.TrimSuffix(source, ".zip"))
		}
		if progress != nil {
			progress("rawzip converted", 100, 100)
		}

		if meta != nil && meta.ModDrv == "RawGenBook" {
			return p.importGenbook(ctx, convertedPath, name, progress)
		}

		source = convertedPath
		format = FormatVPL
	}

	switch format {
	case FormatVPL:
		return p.importVPL(ctx, source, name, versification, progress)
	case FormatOSIS:
		return p.importOSIS(ctx, source, name, versification, progress)
	default:
		return fmt.Errorf("unknown format")
	}
}

func (p *Pipeline) ImportOSISData(ctx context.Context, data []byte, name string, progress ProgressFn) error {
	tmpFile, err := os.CreateTemp("", "gospelfast-osis-*.xml")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.Write(data); err != nil {
		return err
	}
	tmpFile.Close()
	return p.importOSIS(ctx, tmpFile.Name(), name, "KJV", progress)
}

type verseBatch struct {
	BookOSISID string
	Verses     []bible.Verse
}

func (p *Pipeline) importOSIS(ctx context.Context, xmlPath, name, versification string, progress ProgressFn) error {
	f, err := os.Open(xmlPath)
	if err != nil {
		return fmt.Errorf("open xml: %w", err)
	}
	defer f.Close()

	batches := make(map[string]*verseBatch)
	var totalVerses int

	data, err := osis.Parse(f, func(bookOSISID string, chapter, verse int, text string) error {
		if text == "" {
			return nil
		}
		batch, ok := batches[bookOSISID]
		if !ok {
			batch = &verseBatch{BookOSISID: bookOSISID, Verses: make([]bible.Verse, 0, 1000)}
			batches[bookOSISID] = batch
		}
		batch.Verses = append(batch.Verses, bible.Verse{
			Chapter: chapter,
			Verse:   verse,
			Text:    text,
		})
		totalVerses++
		return nil
	})
	if err != nil {
		return fmt.Errorf("parse osis: %w", err)
	}

	if name == "" {
		name = data.Title
		if name == "" {
			name = filepath.Base(strings.TrimSuffix(xmlPath, ".xml"))
		}
	}

	if progress != nil {
		progress("parsed OSIS", totalVerses, totalVerses)
	}

	_ = p.DB.DeleteTranslationByShortName(ctx, strings.ToUpper(name))

	versificationName := versification
	versificationID, err := p.DB.CreateVersification(ctx, versificationName)
	if err != nil {
		return fmt.Errorf("create versification: %w", err)
	}

	bookOSISMap := make(map[string]string)
	for bookOrder, bookInfo := range data.Books {
		_, exists := batches[bookInfo.OSISID]
		if !exists {
			continue
		}
		shortName := osisIDToShort(bookInfo.OSISID)
		testament := "OT"
		if bookOrder >= 39 {
			testament = "NT"
		}
		nameVal := bookInfo.Name
		if nameVal == "" {
			nameVal = shortName
		}
		book := &bible.Book{
			VersificationID: versificationID,
			Name:            nameVal,
			ShortName:       shortName,
			Testament:       testament,
			BookOrder:       bookOrder + 1,
			ChapterCount:    0,
		}
		batch := batches[bookInfo.OSISID]
		for _, v := range batch.Verses {
			if v.Chapter > book.ChapterCount {
				book.ChapterCount = v.Chapter
			}
		}
		if err := p.DB.CreateBook(ctx, book); err != nil {
			return fmt.Errorf("create book %s: %w", bookInfo.OSISID, err)
		}
		bookOSISMap[bookInfo.OSISID] = book.ID
	}

	trans := &bible.Translation{
		ShortName:       strings.ToUpper(name),
		FullName:        name,
		Language:        "en",
		VersificationID: versificationID,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	if err := p.DB.CreateTranslation(ctx, trans); err != nil {
		return fmt.Errorf("create translation: %w", err)
	}

	inserted := 0
	for osisID, batch := range batches {
		bookID, ok := bookOSISMap[osisID]
		if !ok {
			continue
		}
		batchSize := 5000
		for i := 0; i < len(batch.Verses); i += batchSize {
			end := min(i+batchSize, len(batch.Verses))
			dbVerses := make([]bible.Verse, end-i)
			for j, v := range batch.Verses[i:end] {
				dbVerses[j] = bible.Verse{
					TranslationID: trans.ID,
					BookID:        bookID,
					Chapter:       v.Chapter,
					Verse:         v.Verse,
					Text:          v.Text,
				}
			}
			if _, err := p.DB.InsertVerses(ctx, dbVerses); err != nil {
				return fmt.Errorf("insert batch: %w", err)
			}
			inserted += len(dbVerses)
			if progress != nil {
				progress("inserting verses", inserted, totalVerses)
			}
		}
	}

	fmt.Printf("Imported translation %q: %d verses in %d books\n", name, inserted, len(bookOSISMap))
	return nil
}

func (p *Pipeline) importVPL(ctx context.Context, path string, name string, versification string, progress ProgressFn) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	recs, err := vpl.Parse(f)
	if err != nil {
		return fmt.Errorf("parse vpl: %w", err)
	}
	if len(recs) == 0 {
		return fmt.Errorf("no verses found in VPL file")
	}
	if progress != nil {
		progress("parsed VPL", len(recs), len(recs))
	}

	versificationID, err := p.DB.CreateVersification(ctx, versification)
	if err != nil {
		return err
	}

	bookNames := make(map[string]int)
	for _, r := range recs {
		if _, ok := bookNames[r.BookShort]; !ok {
			bookNames[r.BookShort] = len(bookNames)
		}
	}

	bookIDs := make(map[string]string)
	for short, order := range bookNames {
		canon := bible.GetKJVBook(short)
		nameVal := short
		if canon != nil {
			nameVal = canon.Name
		}
		testament := "OT"
		if order >= 39 {
			testament = "NT"
		}
		book := &bible.Book{
			VersificationID: versificationID,
			Name:            nameVal,
			ShortName:       short,
			Testament:       testament,
			BookOrder:       order + 1,
			ChapterCount:    0,
		}
		for _, r := range recs {
			if r.BookShort == short && r.Chapter > book.ChapterCount {
				book.ChapterCount = r.Chapter
			}
		}
		if err := p.DB.CreateBook(ctx, book); err != nil {
			return err
		}
		bookIDs[short] = book.ID
	}

	if name == "" {
		name = filepath.Base(strings.TrimSuffix(path, filepath.Ext(path)))
	}

	_ = p.DB.DeleteTranslationByShortName(ctx, strings.ToUpper(name))

	trans := &bible.Translation{
		ShortName:       strings.ToUpper(name),
		FullName:        name,
		Language:        "en",
		VersificationID: versificationID,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	if err := p.DB.CreateTranslation(ctx, trans); err != nil {
		return err
	}

	total := len(recs)
	inserted := 0
	batchSize := 5000
	for i := 0; i < len(recs); i += batchSize {
		end := min(i+batchSize, len(recs))
		batch := recs[i:end]

		dbVerses := make([]bible.Verse, len(batch))
		for j, r := range batch {
			dbVerses[j] = bible.Verse{
				TranslationID: trans.ID,
				BookID:        bookIDs[r.BookShort],
				Chapter:       r.Chapter,
				Verse:         r.Verse,
				Text:          r.Text,
			}
		}

		if _, err := p.DB.InsertVerses(ctx, dbVerses); err != nil {
			return fmt.Errorf("insert batch: %w", err)
		}
		inserted += len(dbVerses)
		if progress != nil {
			progress("inserting verses", inserted, total)
		}
	}

	return nil
}

func (p *Pipeline) importGenbook(ctx context.Context, impPath, name string, progress ProgressFn) error {
	data, err := os.ReadFile(impPath)
	if err != nil {
		return err
	}

	entries := sword.ParseGenbookIMP(data)
	if len(entries) == 0 {
		return fmt.Errorf("no entries found in genbook module")
	}

	if progress != nil {
		progress("parsed genbook", len(entries), len(entries))
	}

	_ = p.DB.DeleteTranslationByShortName(ctx, strings.ToUpper(name))

	versID, err := p.DB.CreateVersification(ctx, "genbook")
	if err != nil {
		return err
	}

	trans := &bible.Translation{
		ShortName:       strings.ToUpper(name),
		FullName:        name,
		Language:        "en",
		VersificationID: versID,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := p.DB.CreateTranslation(ctx, trans); err != nil {
		return fmt.Errorf("create translation: %w", err)
	}

	dbEntries := make([]db.GenbookEntry, len(entries))
	for i, e := range entries {
		dbEntries[i] = db.GenbookEntry{
			Path:    e.Path,
			Title:   e.Title,
			Content: e.Content,
		}
	}

	inserted, err := p.DB.InsertGenbookEntries(ctx, trans.ID, dbEntries)
	if err != nil {
		return fmt.Errorf("insert genbook entries: %w", err)
	}

	fmt.Printf("Imported genbook %q: %d entries\n", name, inserted)
	return nil
}

func osisIDToShort(osisID string) string {
	m := map[string]string{
		"Gen": "Gen", "Exod": "Exod", "Lev": "Lev", "Num": "Num", "Deut": "Deut",
		"Josh": "Josh", "Judg": "Judg", "Ruth": "Ruth",
		"1Sam": "1Sam", "2Sam": "2Sam", "1Kgs": "1Kgs", "2Kgs": "2Kgs",
		"1Chr": "1Chr", "2Chr": "2Chr", "Ezra": "Ezra", "Neh": "Neh", "Esth": "Esth",
		"Job": "Job", "Ps": "Ps", "Prov": "Prov", "Eccl": "Eccl", "Song": "Song",
		"Isa": "Isa", "Jer": "Jer", "Lam": "Lam", "Ezek": "Ezek", "Dan": "Dan",
		"Hos": "Hos", "Joel": "Joel", "Amos": "Amos", "Obad": "Obad",
		"Jonah": "Jonah", "Mic": "Mic", "Nah": "Nah", "Hab": "Hab",
		"Zeph": "Zeph", "Hag": "Hag", "Zech": "Zech", "Mal": "Mal",
		"Matt": "Matt", "Mark": "Mark", "Luke": "Luke", "John": "John",
		"Acts": "Acts", "Rom": "Rom", "1Cor": "1Cor", "2Cor": "2Cor",
		"Gal": "Gal", "Eph": "Eph", "Phil": "Phil", "Col": "Col",
		"1Thess": "1Thess", "2Thess": "2Thess", "1Tim": "1Tim", "2Tim": "2Tim",
		"Titus": "Titus", "Phlm": "Phlm", "Heb": "Heb",
		"Jas": "Jas", "1Pet": "1Pet", "2Pet": "2Pet",
		"1John": "1John", "2John": "2John", "3John": "3John",
		"Jude": "Jude", "Rev": "Rev",
	}
	if s, ok := m[osisID]; ok {
		return s
	}
	return osisID
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
