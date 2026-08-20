package bible

import "testing"

func TestParseRef(t *testing.T) {
	tests := []struct {
		input    string
		short    string
		name     string
		chapter  int
		verse    int
		verseEnd int
		isRange  bool
	}{
		{"John 3:16", "John", "John", 3, 16, 0, false},
		{"Genesis 1:1", "Gen", "Genesis", 1, 1, 0, false},
		{"Gen 1:1", "Gen", "Genesis", 1, 1, 0, false},
		{"genesis 1:1", "Gen", "Genesis", 1, 1, 0, false},
		{"Psalm 23:1", "Ps", "Psalms", 23, 1, 0, false},
		{"Psalms 119:105", "Ps", "Psalms", 119, 105, 0, false},
		{"1 Corinthians 13:4", "1Cor", "1 Corinthians", 13, 4, 0, false},
		{"Rev 22:21", "Rev", "Revelation", 22, 21, 0, false},
		{"John 3:16-17", "John", "John", 3, 16, 17, true},
		{"Matt 5:3-12", "Matt", "Matthew", 5, 3, 12, true},
		{"Romans 8", "Rom", "Romans", 8, 1, 0, false},
		{"   John   3:16   ", "John", "John", 3, 16, 0, false},
	}

	for _, tt := range tests {
		ref, err := ParseRef(tt.input)
		if err != nil {
			t.Errorf("ParseRef(%q) error: %v", tt.input, err)
			continue
		}
		if ref.BookShortName != tt.short {
			t.Errorf("ParseRef(%q).BookShortName = %q, want %q", tt.input, ref.BookShortName, tt.short)
		}
		if ref.BookName != tt.name {
			t.Errorf("ParseRef(%q).BookName = %q, want %q", tt.input, ref.BookName, tt.name)
		}
		if ref.Chapter != tt.chapter {
			t.Errorf("ParseRef(%q).Chapter = %d, want %d", tt.input, ref.Chapter, tt.chapter)
		}
		if ref.Verse != tt.verse {
			t.Errorf("ParseRef(%q).Verse = %d, want %d", tt.input, ref.Verse, tt.verse)
		}
		if ref.VerseEnd != tt.verseEnd {
			t.Errorf("ParseRef(%q).VerseEnd = %d, want %d", tt.input, ref.VerseEnd, tt.verseEnd)
		}
		if ref.IsRange != tt.isRange {
			t.Errorf("ParseRef(%q).IsRange = %v, want %v", tt.input, ref.IsRange, tt.isRange)
		}
	}
}

// TestParseRefPrefixIsDeterministic guards against a bug where ambiguous
// abbreviation prefixes (matching multiple full book names, e.g. "ph" ->
// philippians/philemon) resolved to a different book on every call because
// the fallback matcher ranged over a Go map, whose iteration order is
// randomized per range.
func TestParseRefPrefixIsDeterministic(t *testing.T) {
	for _, input := range []string{"Ph 4:13", "Jo 3:16", "1 Th 5:1"} {
		ref, err := ParseRef(input)
		if err != nil {
			t.Fatalf("ParseRef(%q) error: %v", input, err)
		}
		want := ref.BookShortName
		for i := 0; i < 50; i++ {
			got, err := ParseRef(input)
			if err != nil {
				t.Fatalf("ParseRef(%q) error: %v", input, err)
			}
			if got.BookShortName != want {
				t.Fatalf("ParseRef(%q) is non-deterministic: got %q on call %d, want %q (from first call)",
					input, got.BookShortName, i, want)
			}
		}
	}
}

func TestParseRefErrors(t *testing.T) {
	bad := []string{"", "  ", "invalid", "xyz 1:1"}
	for _, input := range bad {
		_, err := ParseRef(input)
		if err == nil {
			t.Errorf("ParseRef(%q) should have returned an error", input)
		}
	}
}
