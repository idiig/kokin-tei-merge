package wordlist

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// Token represents a single <w> element extracted from the body.
type Token struct {
	Pos        string // pos attribute
	Lemma      string // lemma attribute
	MSD        MSD    // parsed msd attribute
	Surface    string // text content of <w>
	InCompound bool   // true if this <w> is inside an <app><rdg> with siblings
	InSecondRdg bool  // true if this <w> is inside the 2nd or later <rdg> of an <app>
}

// SenseKey uniquely identifies a sense within an entry.
type SenseKey struct {
	WLSPH           string
	WLSP            string
	WLSPDescription string
}

// Sense represents a <sense> element in the dictionary entry.
type Sense struct {
	N               int
	WLSPH           string
	WLSP            string
	WLSPDescription string
}

// GramInfo holds grammar info for a single POS usage.
type GramInfo struct {
	Pos          string // e.g. "N.g"
	UPosTag      string
	IPAPosTag    string
	UniDicPosTag string
}

// GramKey uniquely identifies a grammar group.
type GramKey struct {
	Pos          string
	UPosTag      string
	IPAPosTag    string
	UniDicPosTag string
}

// CompoundPart records a component lemma of a compound word.
type CompoundPart struct {
	Lemma string
}

// ModernRef records a modern-Japanese form cross-reference.
type ModernRef struct {
	Lemma string
}

// Entry represents a dictionary <entry> in Dict B (<back><div type="dictionary">).
// Keyed by Lemma alone; an entry may have multiple readings.
type Entry struct {
	ID            string   // xml:id, e.g. "年"
	Lemma         string
	LemmaReadings []string // unique readings, sorted
	Grams         []GramInfo
	Senses        []Sense
	IsCompound    bool
	Parts         []CompoundPart // only if IsCompound
	Modern        *ModernRef     // modern form cross-reference, if any
}

// PronHom represents one orthographic form (lemma) for a given reading in Dict A.
type PronHom struct {
	N     int
	ID    string // xml:id, e.g. "はる.春"
	Lemma string // the lemma this hom represents, e.g. "春"
	RefID string // Dict B entry ID, e.g. "春"
}

// PronEntry represents a reading-indexed entry in Dict A (<back><div type="reading-index">).
type PronEntry struct {
	ID      string // xml:id, e.g. "はる"
	Reading string // kana reading
	Homs    []PronHom
}

// EntryID returns the xml:id for a Dict B entry given its lemma.
func EntryID(lemma string) string {
	return sanitizeNCName(lemma)
}

// PronEntryID returns the xml:id for a Dict A reading entry.
func PronEntryID(reading string) string {
	return sanitizeNCName(reading)
}

// PronHomID returns the xml:id for a Dict A hom (reading.orth).
func PronHomID(reading, orth string) string {
	return sanitizeNCName(reading) + "." + sanitizeNCName(orth)
}

// sanitizeNCName replaces characters that are not valid in XML NCName.
// NCName allows: letter, digit, '.', '-', '_', CombiningChar, Extender,
// plus CJK ideographs and kana. We keep those and replace everything else.
func sanitizeNCName(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if isNCNameChar(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}

func isNCNameChar(r rune) bool {
	if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
		return true
	}
	if r == '_' || r == '-' || r == '.' {
		return true
	}
	// CJK ideographs (U+4E00–U+9FFF), Hiragana (U+3040–U+309F),
	// Katakana (U+30A0–U+30FF), CJK Extension A (U+3400–U+4DBF).
	if r >= 0x3040 && r <= 0x9FFF || r >= 0x3400 && r <= 0x4DBF {
		return true
	}
	// Halfwidth/fullwidth forms, CJK compatibility.
	if r >= 0xF900 && r <= 0xFAFF || r >= 0xFF00 && r <= 0xFFEF {
		return true
	}
	// Unicode letter/digit (catches other scripts).
	if unicode.IsLetter(r) || unicode.IsDigit(r) {
		return true
	}
	return false
}

// BuildEntries groups tokens into dictionary entries.
// Tokens with the same Lemma are merged into one entry.
// Multiple readings → collected in LemmaReadings.
// Multiple POS → multiple GramInfo; multiple WLSP → multiple Sense.
func BuildEntries(tokens []Token) []*Entry {
	entryMap := make(map[string]*Entry) // keyed by Lemma
	gramSeen := make(map[string]map[GramKey]bool)
	senseSeen := make(map[string]map[SenseKey]bool)
	readingSeen := make(map[string]map[string]bool)

	for _, tok := range tokens {
		key := tok.Lemma

		e, exists := entryMap[key]
		if !exists {
			e = &Entry{
				ID:    EntryID(key),
				Lemma: key,
			}
			entryMap[key] = e
			gramSeen[key] = make(map[GramKey]bool)
			senseSeen[key] = make(map[SenseKey]bool)
			readingSeen[key] = make(map[string]bool)
		}

		// Collect unique readings.
		if tok.MSD.LemmaReading != "" && !readingSeen[key][tok.MSD.LemmaReading] {
			readingSeen[key][tok.MSD.LemmaReading] = true
			e.LemmaReadings = append(e.LemmaReadings, tok.MSD.LemmaReading)
		}

		// Add gram info if new.
		gk := GramKey{
			Pos:          tok.Pos,
			UPosTag:      tok.MSD.UPosTag,
			IPAPosTag:    tok.MSD.IPAPosTag,
			UniDicPosTag: tok.MSD.UniDicPosTag,
		}
		if !gramSeen[key][gk] {
			gramSeen[key][gk] = true
			e.Grams = append(e.Grams, GramInfo{
				Pos:          tok.Pos,
				UPosTag:      tok.MSD.UPosTag,
				IPAPosTag:    tok.MSD.IPAPosTag,
				UniDicPosTag: tok.MSD.UniDicPosTag,
			})
		}

		// Add sense if new (at least WLSPH must be present).
		if tok.MSD.WLSPH != "" {
			sk := SenseKey{
				WLSPH:           tok.MSD.WLSPH,
				WLSP:            tok.MSD.WLSP,
				WLSPDescription: tok.MSD.WLSPDescription,
			}
			if !senseSeen[key][sk] {
				senseSeen[key][sk] = true
				e.Senses = append(e.Senses, Sense{
					WLSPH:           tok.MSD.WLSPH,
					WLSP:            tok.MSD.WLSP,
					WLSPDescription: tok.MSD.WLSPDescription,
				})
			}
		}
	}

	// Collect entries, sort readings, number senses.
	entries := make([]*Entry, 0, len(entryMap))
	for _, e := range entryMap {
		sort.Strings(e.LemmaReadings)
		for i := range e.Senses {
			e.Senses[i].N = i + 1
		}
		entries = append(entries, e)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].ID < entries[j].ID
	})

	return entries
}

// BuildPronEntries groups tokens by surface kana string, creating Dict A entries.
// The primary key is KanjiReading (the actual kana of the inflected form as it
// appears in text), falling back to LemmaReading when absent.
// Each unique (reading, lemma) pair becomes one PronHom.
func BuildPronEntries(tokens []Token) []*PronEntry {
	type key struct{ reading, lemma string }
	seen := make(map[key]bool)
	readingHoms := make(map[string][]PronHom) // reading → homs

	for _, tok := range tokens {
		reading := tok.MSD.KanjiReading
		if reading == "" || reading == "???" || reading == "-" {
			reading = tok.MSD.LemmaReading
		}
		if reading == "" {
			continue
		}
		k := key{reading, tok.Lemma}
		if seen[k] {
			continue
		}
		seen[k] = true
		readingHoms[reading] = append(readingHoms[reading], PronHom{
			Lemma: tok.Lemma,
			RefID: EntryID(tok.Lemma),
		})
	}

	entries := make([]*PronEntry, 0, len(readingHoms))
	for reading, homs := range readingHoms {
		sort.Slice(homs, func(i, j int) bool {
			return homs[i].Lemma < homs[j].Lemma
		})
		for i := range homs {
			homs[i].N = i + 1
			homs[i].ID = PronHomID(reading, homs[i].Lemma)
		}
		entries = append(entries, &PronEntry{
			ID:      PronEntryID(reading),
			Reading: reading,
			Homs:    homs,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].ID < entries[j].ID
	})
	return entries
}

// MarkCompounds marks entries as compound and attaches modern form refs.
// compoundTokens maps compound lemma → list of component parts.
// modernRefs maps lemma → modern form lemma.
func MarkCompounds(entries []*Entry, compoundTokens map[string][]CompoundPart, modernRefs map[string]ModernRef) {
	for _, e := range entries {
		if parts, ok := compoundTokens[e.Lemma]; ok {
			e.IsCompound = true
			e.Parts = parts
		}
		if ref, ok := modernRefs[e.Lemma]; ok {
			e.Modern = &ref
		}
	}
}

// NeedsHom returns true if the entry has multiple distinct POS values
// and should use <hom> elements instead of a single <gramGrp>.
func (e *Entry) NeedsHom() bool {
	if len(e.Grams) <= 1 {
		return false
	}
	first := e.Grams[0].Pos
	for _, g := range e.Grams[1:] {
		if g.Pos != first {
			return true
		}
	}
	return false
}

// NeedsSenseIDs returns true if the entry has multiple senses,
// requiring xml:id on each <sense>.
func (e *Entry) NeedsSenseIDs() bool {
	return len(e.Senses) > 1
}

// SenseID returns the xml:id for the nth sense of this entry (1-based).
func (e *Entry) SenseID(n int) string {
	return fmt.Sprintf("%s.s%d", e.ID, n)
}

// HomID returns the xml:id for the nth hom of this entry (1-based).
func (e *Entry) HomID(n int) string {
	return fmt.Sprintf("%s.h%d", e.ID, n)
}

