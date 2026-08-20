package bible

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type VerseRef struct {
	BookName      string
	BookShortName string
	Chapter       int
	Verse         int
	VerseEnd      int
	IsRange       bool
}

var bookNameToShort = map[string]string{
	"genesis":        "Gen",
	"exodus":          "Exod",
	"leviticus":       "Lev",
	"numbers":         "Num",
	"deuteronomy":     "Deut",
	"joshua":          "Josh",
	"judges":          "Judg",
	"ruth":            "Ruth",
	"1 samuel":        "1Sam",
	"2 samuel":        "2Sam",
	"1 kings":         "1Kgs",
	"2 kings":         "2Kgs",
	"1 chronicles":    "1Chr",
	"2 chronicles":    "2Chr",
	"ezra":            "Ezra",
	"nehemiah":        "Neh",
	"esther":          "Esth",
	"job":             "Job",
	"psalm":           "Ps",
	"psalms":          "Ps",
	"proverbs":        "Prov",
	"ecclesiastes":    "Eccl",
	"song of solomon": "Song",
	"song of songs":   "Song",
	"isaiah":          "Isa",
	"jeremiah":        "Jer",
	"lamentations":    "Lam",
	"ezekiel":         "Ezek",
	"daniel":          "Dan",
	"hosea":           "Hos",
	"joel":            "Joel",
	"amos":            "Amos",
	"obadiah":         "Obad",
	"jonah":           "Jonah",
	"micah":           "Mic",
	"nahum":           "Nah",
	"habakkuk":        "Hab",
	"zephaniah":       "Zeph",
	"haggai":          "Hag",
	"zechariah":       "Zech",
	"malachi":         "Mal",
	"matthew":         "Matt",
	"mark":            "Mark",
	"luke":            "Luke",
	"john":            "John",
	"acts":            "Acts",
	"romans":          "Rom",
	"1 corinthians":   "1Cor",
	"2 corinthians":   "2Cor",
	"galatians":       "Gal",
	"ephesians":       "Eph",
	"philippians":     "Phil",
	"colossians":      "Col",
	"1 thessalonians": "1Thess",
	"2 thessalonians": "2Thess",
	"1 timothy":       "1Tim",
	"2 timothy":       "2Tim",
	"titus":           "Titus",
	"philemon":        "Phlm",
	"hebrews":         "Heb",
	"james":           "Jas",
	"1 peter":         "1Pet",
	"2 peter":         "2Pet",
	"1 john":          "1John",
	"2 john":          "2John",
	"3 john":          "3John",
	"jude":            "Jude",
	"revelation":      "Rev",
}

var shortNameToBook = map[string]string{
	"gen":    "Genesis",
	"exod":   "Exodus",
	"lev":    "Leviticus",
	"num":    "Numbers",
	"deut":   "Deuteronomy",
	"josh":   "Joshua",
	"judg":   "Judges",
	"ruth":   "Ruth",
	"1sam":   "1 Samuel",
	"2sam":   "2 Samuel",
	"1kgs":   "1 Kings",
	"2kgs":   "2 Kings",
	"1chr":   "1 Chronicles",
	"2chr":   "2 Chronicles",
	"ezra":   "Ezra",
	"neh":    "Nehemiah",
	"esth":   "Esther",
	"job":    "Job",
	"ps":     "Psalms",
	"prov":   "Proverbs",
	"eccl":   "Ecclesiastes",
	"song":   "Song of Solomon",
	"isa":    "Isaiah",
	"jer":    "Jeremiah",
	"lam":    "Lamentations",
	"ezek":   "Ezekiel",
	"dan":    "Daniel",
	"hos":    "Hosea",
	"joel":   "Joel",
	"amos":   "Amos",
	"obad":   "Obadiah",
	"jonah":  "Jonah",
	"mic":    "Micah",
	"nah":    "Nahum",
	"hab":    "Habakkuk",
	"zeph":   "Zephaniah",
	"hag":    "Haggai",
	"zech":   "Zechariah",
	"mal":    "Malachi",
	"matt":   "Matthew",
	"mark":   "Mark",
	"luke":   "Luke",
	"john":   "John",
	"acts":   "Acts",
	"rom":    "Romans",
	"1cor":   "1 Corinthians",
	"2cor":   "2 Corinthians",
	"gal":    "Galatians",
	"eph":    "Ephesians",
	"phil":   "Philippians",
	"col":    "Colossians",
	"1thess": "1 Thessalonians",
	"2thess": "2 Thessalonians",
	"1tim":   "1 Timothy",
	"2tim":   "2 Timothy",
	"titus":  "Titus",
	"phlm":   "Philemon",
	"heb":    "Hebrews",
	"jas":    "James",
	"1pet":   "1 Peter",
	"2pet":   "2 Peter",
	"1john":  "1 John",
	"2john":  "2 John",
	"3john":  "3 John",
	"jude":   "Jude",
	"rev":    "Revelation",
}

func ParseRef(ref string) (*VerseRef, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, fmt.Errorf("empty reference")
	}

	bookShort, versePart, err := extractBookName(ref)
	if err != nil {
		return nil, err
	}

	chapter, verseNum, verseEnd, isRange, err := parseVersePart(versePart)
	if err != nil {
		return nil, err
	}

	return &VerseRef{
		BookName:      ShortToBookName(bookShort),
		BookShortName: bookShort,
		Chapter:       chapter,
		Verse:         verseNum,
		VerseEnd:      verseEnd,
		IsRange:       isRange,
	}, nil
}

func parseVersePart(s string) (chapter, verse, verseEnd int, isRange bool, err error) {
	colonIdx := strings.Index(s, ":")
	if colonIdx == -1 {
		chapterStr := trimTrailing(s)
		chapter, err = strconv.Atoi(chapterStr)
		if err != nil {
			return 0, 0, 0, false, fmt.Errorf("invalid chapter: %q", s)
		}
		return chapter, 1, 0, false, nil
	}

	chapter, err = strconv.Atoi(s[:colonIdx])
	if err != nil {
		return 0, 0, 0, false, fmt.Errorf("invalid chapter in %q", s)
	}

	verseStr := trimTrailing(s[colonIdx+1:])
	if dashIdx := strings.Index(verseStr, "-"); dashIdx != -1 {
		verse, err = strconv.Atoi(verseStr[:dashIdx])
		if err != nil {
			return 0, 0, 0, false, fmt.Errorf("invalid verse in %q", s)
		}
		verseEnd, err = strconv.Atoi(trimTrailing(verseStr[dashIdx+1:]))
		if err != nil {
			return 0, 0, 0, false, fmt.Errorf("invalid verse end in %q", s)
		}
		return chapter, verse, verseEnd, true, nil
	}

	verse, err = strconv.Atoi(verseStr)
	if err != nil {
		return 0, 0, 0, false, fmt.Errorf("invalid verse in %q", s)
	}
	return chapter, verse, 0, false, nil
}

func trimTrailing(s string) string {
	if idx := strings.Index(s, " "); idx != -1 {
		return s[:idx]
	}
	return s
}

// bookNamesSorted holds the keys of bookNameToShort in a fixed, deterministic
// order so prefix matching below doesn't depend on Go's randomized map
// iteration order.
var bookNamesSorted = sortedBookNames()

func sortedBookNames() []string {
	names := make([]string, 0, len(bookNameToShort))
	for name := range bookNameToShort {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func ShortToBookName(short string) string {
	if name, ok := shortNameToBook[strings.ToLower(short)]; ok {
		return name
	}
	return short
}

func extractBookName(ref string) (shortName, remainder string, err error) {
	words := strings.Fields(ref)

	for n := len(words); n >= 1; n-- {
		candidate := strings.ToLower(strings.Join(words[:n], " "))
		if short, ok := bookNameToShort[candidate]; ok {
			return short, strings.Join(words[n:], " "), nil
		}
		if short, ok := bookNameToShort[candidate+"s"]; ok {
			return short, strings.Join(words[n:], " "), nil
		}
	}

	for n := len(words) - 1; n >= 1; n-- {
		candidate := strings.ToLower(strings.Join(words[:n], " "))
		for _, name := range bookNamesSorted {
			if strings.HasPrefix(name, candidate) {
				return bookNameToShort[name], strings.Join(words[n:], " "), nil
			}
		}
	}

	return "", "", fmt.Errorf("unknown book in reference: %q", ref)
}
