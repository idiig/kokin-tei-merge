package wordlist

import "testing"

func TestBuildEntries_SingleToken(t *testing.T) {
	tokens := []Token{
		{
			Pos:   "N.g",
			Lemma: "年",
			MSD: MSD{
				UPosTag:         "NOUN",
				IPAPosTag:       "名詞-一般",
				UniDicPosTag:    "名詞-普通名詞-一般",
				LemmaReading:    "とし",
				WLSPH:           "1.1630",
				WLSP:            "1.1630",
				WLSPDescription: "体-関係-時間-年",
			},
		},
	}

	entries := BuildEntries(tokens)

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.ID != "w.年" {
		t.Errorf("ID = %q, want %q", e.ID, "w.年")
	}
	if e.Lemma != "年" {
		t.Errorf("Lemma = %q, want %q", e.Lemma, "年")
	}
	if len(e.LemmaReadings) != 1 || e.LemmaReadings[0] != "とし" {
		t.Errorf("LemmaReadings = %v, want [とし]", e.LemmaReadings)
	}
	if len(e.Grams) != 1 {
		t.Fatalf("expected 1 gram, got %d", len(e.Grams))
	}
	if e.Grams[0].Pos != "N.g" {
		t.Errorf("Grams[0].Pos = %q, want %q", e.Grams[0].Pos, "N.g")
	}
	if len(e.Senses) != 1 {
		t.Fatalf("expected 1 sense, got %d", len(e.Senses))
	}
	if e.Senses[0].WLSP != "1.1630" {
		t.Errorf("Senses[0].WLSP = %q, want %q", e.Senses[0].WLSP, "1.1630")
	}
}

func TestBuildEntries_DuplicateTokensMerge(t *testing.T) {
	tok := Token{
		Pos:   "N.g",
		Lemma: "春",
		MSD: MSD{
			UPosTag:      "NOUN",
			LemmaReading: "はる",
			WLSPH:        "1.1624",
			WLSP:         "1.1624",
		},
	}
	entries := BuildEntries([]Token{tok, tok, tok})

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if len(entries[0].Grams) != 1 {
		t.Errorf("expected 1 gram (deduped), got %d", len(entries[0].Grams))
	}
	if len(entries[0].Senses) != 1 {
		t.Errorf("expected 1 sense (deduped), got %d", len(entries[0].Senses))
	}
}

func TestBuildEntries_MultipleReadings(t *testing.T) {
	tokens := []Token{
		{
			Pos:   "N.g",
			Lemma: "下",
			MSD: MSD{
				UPosTag:      "NOUN",
				LemmaReading: "した",
				WLSPH:        "1.1741",
			},
		},
		{
			Pos:   "N.g",
			Lemma: "下",
			MSD: MSD{
				UPosTag:      "NOUN",
				LemmaReading: "もと",
				WLSPH:        "1.1770",
			},
		},
	}

	entries := BuildEntries(tokens)

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (same lemma), got %d", len(entries))
	}
	e := entries[0]
	if e.ID != "w.下" {
		t.Errorf("ID = %q, want %q", e.ID, "w.下")
	}
	// Readings should be sorted.
	if len(e.LemmaReadings) != 2 {
		t.Fatalf("expected 2 readings, got %d", len(e.LemmaReadings))
	}
	if e.LemmaReadings[0] != "した" || e.LemmaReadings[1] != "もと" {
		t.Errorf("LemmaReadings = %v, want [した もと]", e.LemmaReadings)
	}
}

func TestBuildEntries_MultipleSenses(t *testing.T) {
	tokens := []Token{
		{
			Pos:   "N.g",
			Lemma: "下",
			MSD: MSD{
				UPosTag:         "NOUN",
				LemmaReading:    "した",
				WLSPH:           "1.1101",
				WLSP:            "1.1101",
				WLSPDescription: "体-関係-類",
			},
		},
		{
			Pos:   "N.g",
			Lemma: "下",
			MSD: MSD{
				UPosTag:         "NOUN",
				LemmaReading:    "した",
				WLSPH:           "1.1710",
				WLSP:            "1.1710",
				WLSPDescription: "体-関係-空間的関係-点",
			},
		},
	}

	entries := BuildEntries(tokens)

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if len(e.Senses) != 2 {
		t.Fatalf("expected 2 senses, got %d", len(e.Senses))
	}
	if !e.NeedsSenseIDs() {
		t.Error("expected NeedsSenseIDs() = true")
	}
	if e.SenseID(1) != "w.下.s1" {
		t.Errorf("SenseID(1) = %q, want %q", e.SenseID(1), "w.下.s1")
	}
}

func TestBuildEntries_MultiPOS(t *testing.T) {
	tokens := []Token{
		{
			Pos:   "N.g",
			Lemma: "春",
			MSD: MSD{
				UPosTag:      "NOUN",
				IPAPosTag:    "名詞-一般",
				UniDicPosTag: "名詞-普通名詞-一般",
				LemmaReading: "はる",
				WLSPH:        "1.1624",
			},
		},
		{
			Pos:   "N.Adv",
			Lemma: "春",
			MSD: MSD{
				UPosTag:      "NOUN",
				IPAPosTag:    "名詞-副詞可能",
				UniDicPosTag: "名詞-普通名詞-副詞可能",
				LemmaReading: "はる",
				WLSPH:        "1.1624",
			},
		},
	}

	entries := BuildEntries(tokens)

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if len(e.Grams) != 2 {
		t.Fatalf("expected 2 grams, got %d", len(e.Grams))
	}
	if !e.NeedsHom() {
		t.Error("expected NeedsHom() = true for multi-POS entry")
	}
	if e.HomID(1) != "w.春.h1" {
		t.Errorf("HomID(1) = %q, want %q", e.HomID(1), "w.春.h1")
	}
}

func TestBuildEntries_NoWLSPH(t *testing.T) {
	tokens := []Token{
		{
			Pos:   "N.g",
			Lemma: "何",
			MSD: MSD{
				UPosTag:      "NOUN",
				LemmaReading: "なに",
			},
		},
	}

	entries := BuildEntries(tokens)

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if len(entries[0].Senses) != 0 {
		t.Errorf("expected 0 senses when no WLSPH, got %d", len(entries[0].Senses))
	}
}

func TestNeedsHom_SingleGram(t *testing.T) {
	e := &Entry{
		Grams: []GramInfo{{Pos: "N.g"}},
	}
	if e.NeedsHom() {
		t.Error("NeedsHom() should be false for single gram")
	}
}

func TestNeedsHom_SamePOS(t *testing.T) {
	e := &Entry{
		Grams: []GramInfo{
			{Pos: "N.g", IPAPosTag: "名詞-一般"},
			{Pos: "N.g", IPAPosTag: "名詞-固有名詞"},
		},
	}
	if e.NeedsHom() {
		t.Error("NeedsHom() should be false when all POS are identical")
	}
}

func TestMarkCompounds(t *testing.T) {
	entries := []*Entry{
		{ID: "w.春霞", Lemma: "春霞"},
		{ID: "w.春", Lemma: "春"},
	}
	compounds := map[string][]CompoundPart{
		"春霞": {
			{Lemma: "春"},
			{Lemma: "霞"},
		},
	}
	modernRefs := map[string]ModernRef{}

	MarkCompounds(entries, compounds, modernRefs)

	if !entries[0].IsCompound {
		t.Error("expected 春霞 to be marked as compound")
	}
	if len(entries[0].Parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(entries[0].Parts))
	}
	if entries[0].Parts[0].Lemma != "春" {
		t.Errorf("Parts[0].Lemma = %q, want %q", entries[0].Parts[0].Lemma, "春")
	}
	if entries[1].IsCompound {
		t.Error("expected 春 not to be compound")
	}
}

func TestMarkCompounds_ModernForm(t *testing.T) {
	entries := []*Entry{
		{ID: "w.留む", Lemma: "留む"},
	}
	compounds := map[string][]CompoundPart{}
	modernRefs := map[string]ModernRef{
		"留む": {Lemma: "とめる"},
	}

	MarkCompounds(entries, compounds, modernRefs)

	if entries[0].Modern == nil {
		t.Fatal("expected modern ref to be set")
	}
	if entries[0].Modern.Lemma != "とめる" {
		t.Errorf("Modern.Lemma = %q, want %q", entries[0].Modern.Lemma, "とめる")
	}
}

func TestEntryID(t *testing.T) {
	tests := []struct {
		lemma, want string
	}{
		{"年", "w.年"},
		{"春", "w.春"},
		{"の", "w.の"},
		{"いか−", "w.いか_"}, // U+2212 MINUS SIGN → underscore
	}
	for _, tt := range tests {
		got := EntryID(tt.lemma)
		if got != tt.want {
			t.Errorf("EntryID(%q) = %q, want %q", tt.lemma, got, tt.want)
		}
	}
}

func TestBuildEntries_Sorted(t *testing.T) {
	tokens := []Token{
		{Pos: "N.g", Lemma: "春", MSD: MSD{LemmaReading: "はる", WLSPH: "1.1624"}},
		{Pos: "N.g", Lemma: "年", MSD: MSD{LemmaReading: "とし", WLSPH: "1.1630"}},
		{Pos: "ADP", Lemma: "の", MSD: MSD{LemmaReading: "の", WLSPH: "8.0061"}},
	}

	entries := BuildEntries(tokens)

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	for i := 1; i < len(entries); i++ {
		if entries[i].ID < entries[i-1].ID {
			t.Errorf("entries not sorted: %q < %q", entries[i].ID, entries[i-1].ID)
		}
	}
}
