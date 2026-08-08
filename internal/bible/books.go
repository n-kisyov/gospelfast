package bible

type Testament string

const (
	TestamentOld        Testament = "OT"
	TestamentNew        Testament = "NT"
	TestamentApocrypha  Testament = "AP"
)

type CanonicalBook struct {
	Name         string
	ShortName    string
	Testament    Testament
	BookOrder    int
	ChapterCount int
}

var KJVBooks = []CanonicalBook{
	{"Genesis", "Gen", TestamentOld, 1, 50},
	{"Exodus", "Exod", TestamentOld, 2, 40},
	{"Leviticus", "Lev", TestamentOld, 3, 27},
	{"Numbers", "Num", TestamentOld, 4, 36},
	{"Deuteronomy", "Deut", TestamentOld, 5, 34},
	{"Joshua", "Josh", TestamentOld, 6, 24},
	{"Judges", "Judg", TestamentOld, 7, 21},
	{"Ruth", "Ruth", TestamentOld, 8, 4},
	{"1 Samuel", "1Sam", TestamentOld, 9, 31},
	{"2 Samuel", "2Sam", TestamentOld, 10, 24},
	{"1 Kings", "1Kgs", TestamentOld, 11, 22},
	{"2 Kings", "2Kgs", TestamentOld, 12, 25},
	{"1 Chronicles", "1Chr", TestamentOld, 13, 29},
	{"2 Chronicles", "2Chr", TestamentOld, 14, 36},
	{"Ezra", "Ezra", TestamentOld, 15, 10},
	{"Nehemiah", "Neh", TestamentOld, 16, 13},
	{"Esther", "Esth", TestamentOld, 17, 10},
	{"Job", "Job", TestamentOld, 18, 42},
	{"Psalms", "Ps", TestamentOld, 19, 150},
	{"Proverbs", "Prov", TestamentOld, 20, 31},
	{"Ecclesiastes", "Eccl", TestamentOld, 21, 12},
	{"Song of Solomon", "Song", TestamentOld, 22, 8},
	{"Isaiah", "Isa", TestamentOld, 23, 66},
	{"Jeremiah", "Jer", TestamentOld, 24, 52},
	{"Lamentations", "Lam", TestamentOld, 25, 5},
	{"Ezekiel", "Ezek", TestamentOld, 26, 48},
	{"Daniel", "Dan", TestamentOld, 27, 12},
	{"Hosea", "Hos", TestamentOld, 28, 14},
	{"Joel", "Joel", TestamentOld, 29, 3},
	{"Amos", "Amos", TestamentOld, 30, 9},
	{"Obadiah", "Obad", TestamentOld, 31, 1},
	{"Jonah", "Jonah", TestamentOld, 32, 4},
	{"Micah", "Mic", TestamentOld, 33, 7},
	{"Nahum", "Nah", TestamentOld, 34, 3},
	{"Habakkuk", "Hab", TestamentOld, 35, 3},
	{"Zephaniah", "Zeph", TestamentOld, 36, 3},
	{"Haggai", "Hag", TestamentOld, 37, 2},
	{"Zechariah", "Zech", TestamentOld, 38, 14},
	{"Malachi", "Mal", TestamentOld, 39, 4},
	{"Matthew", "Matt", TestamentNew, 40, 28},
	{"Mark", "Mark", TestamentNew, 41, 16},
	{"Luke", "Luke", TestamentNew, 42, 24},
	{"John", "John", TestamentNew, 43, 21},
	{"Acts", "Acts", TestamentNew, 44, 28},
	{"Romans", "Rom", TestamentNew, 45, 16},
	{"1 Corinthians", "1Cor", TestamentNew, 46, 16},
	{"2 Corinthians", "2Cor", TestamentNew, 47, 13},
	{"Galatians", "Gal", TestamentNew, 48, 6},
	{"Ephesians", "Eph", TestamentNew, 49, 6},
	{"Philippians", "Phil", TestamentNew, 50, 4},
	{"Colossians", "Col", TestamentNew, 51, 4},
	{"1 Thessalonians", "1Thess", TestamentNew, 52, 5},
	{"2 Thessalonians", "2Thess", TestamentNew, 53, 3},
	{"1 Timothy", "1Tim", TestamentNew, 54, 6},
	{"2 Timothy", "2Tim", TestamentNew, 55, 4},
	{"Titus", "Titus", TestamentNew, 56, 3},
	{"Philemon", "Phlm", TestamentNew, 57, 1},
	{"Hebrews", "Heb", TestamentNew, 58, 13},
	{"James", "Jas", TestamentNew, 59, 5},
	{"1 Peter", "1Pet", TestamentNew, 60, 5},
	{"2 Peter", "2Pet", TestamentNew, 61, 3},
	{"1 John", "1John", TestamentNew, 62, 5},
	{"2 John", "2John", TestamentNew, 63, 1},
	{"3 John", "3John", TestamentNew, 64, 1},
	{"Jude", "Jude", TestamentNew, 65, 1},
	{"Revelation", "Rev", TestamentNew, 66, 22},
}

func GetKJVBook(shortName string) *CanonicalBook {
	for i := range KJVBooks {
		if KJVBooks[i].ShortName == shortName {
			return &KJVBooks[i]
		}
	}
	return nil
}

func ShortNameToOrder(shortName string) int {
	if b := GetKJVBook(shortName); b != nil {
		return b.BookOrder
	}
	return -1
}

func OrderToShortName(order int) string {
	for _, b := range KJVBooks {
		if b.BookOrder == order {
			return b.ShortName
		}
	}
	return ""
}
