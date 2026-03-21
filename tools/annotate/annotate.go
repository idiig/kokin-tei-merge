package annotate

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/beevik/etree"
)

// ReadDocument reads a TEI XML file preserving CData.
func ReadDocument(path string) (*etree.Document, error) {
	doc := etree.NewDocument()
	doc.ReadSettings.PreserveCData = true
	if err := doc.ReadFromFile(path); err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return doc, nil
}

// WriteDocument writes an etree Document with canonical formatting.
func WriteDocument(doc *etree.Document, path string) error {
	trimMixedContentWhitespace(doc.Root())
	doc.Indent(2)
	doc.WriteSettings.CanonicalAttrVal = true
	doc.WriteSettings.CanonicalEndTags = false
	doc.WriteSettings.CanonicalText = true
	return doc.WriteToFile(path)
}

// trimMixedContentWhitespace walks the element tree and strips trailing
// whitespace from CharData nodes that have element siblings. This prevents
// Indent(2) from producing large gaps in mixed-content elements like
// <seg>花と\n               <app>...</app></seg>.
func trimMixedContentWhitespace(el *etree.Element) {
	if el == nil {
		return
	}
	hasElementSibling := false
	for _, child := range el.Child {
		if _, ok := child.(*etree.Element); ok {
			hasElementSibling = true
			break
		}
	}
	if hasElementSibling {
		for _, child := range el.Child {
			if cd, ok := child.(*etree.CharData); ok {
				cd.Data = strings.TrimRight(cd.Data, " \t\n\r")
			}
		}
	}
	for _, child := range el.Child {
		if e, ok := child.(*etree.Element); ok {
			trimMixedContentWhitespace(e)
		}
	}
}

// stripSpace removes all Unicode whitespace from s.
// Used to normalise XML-indented text (newlines, spaces) from <seg> content.
func stripSpace(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
}

// SegText extracts the canonical text of a <seg>, including text inside
// <app><lem> but excluding <rdg> variants and other non-text elements.
// All Unicode whitespace (XML indentation, newlines) is removed.
func SegText(seg *etree.Element) string {
	var sb strings.Builder
	collectLemText(seg, &sb)
	return stripSpace(sb.String())
}

// SegMeta holds the canonical text of one <seg> and any variant readings
// found inside <app> elements. LemText and RdgText are empty when no <app>
// is present. RdgTokens is populated only when the seg is already annotated
// (has <w> elements inside <rdg>) and is used by GenerateDraft to show
// individual rdg token rows instead of a placeholder.
type SegMeta struct {
	Text      string  // SegText(seg): lem-canonical, whitespace-stripped
	LemText   string  // concatenated <lem> text from all <app> children
	RdgText   string  // concatenated <rdg> text from all <app> children
	RdgTokens []Token // non-nil when rdg is already annotated with <w> elements
}

// ExtractSegMetas extracts canonical text and apparatus metadata for each
// <seg>. If a seg contains <app> elements, LemText and RdgText are populated
// with the concatenated (whitespace-stripped) text of all <lem> and <rdg>
// children respectively.
func ExtractSegMetas(segs []*etree.Element) []SegMeta {
	metas := make([]SegMeta, len(segs))
	for i, seg := range segs {
		metas[i].Text = SegText(seg)
		for _, app := range seg.SelectElements("app") {
			if lem := app.SelectElement("lem"); lem != nil {
				var sb strings.Builder
				collectLemText(lem, &sb)
				metas[i].LemText += stripSpace(sb.String())
			}
			if rdg := app.SelectElement("rdg"); rdg != nil {
				var sb strings.Builder
				collectLemText(rdg, &sb)
				metas[i].RdgText += stripSpace(sb.String())
			}
		}
	}
	return metas
}

// TokensFromAnnotatedSegs extracts Token slices from segs that have already
// been annotated with <w> elements. It returns:
//   - lemTokens: flat list of all lem <w> tokens across all segs (direct + inside <app><lem>)
//   - splits: number of lem tokens per seg
//   - metas: SegMeta for each seg, with RdgTokens populated from existing <rdg><w> elements
//
// Returns (nil, nil, nil) if the segs are not annotated (no <w> elements found).
func TokensFromAnnotatedSegs(segs []*etree.Element) (lemTokens []Token, splits []int, metas []SegMeta) {
	// Check if any seg has <w> children.
	hasAnnotation := false
	for _, seg := range segs {
		if len(seg.FindElements(".//w")) > 0 {
			hasAnnotation = true
			break
		}
	}
	if !hasAnnotation {
		return nil, nil, nil
	}

	for _, seg := range segs {
		var segLemToks []Token
		collectLemTokens(seg, &segLemToks)
		lemTokens = append(lemTokens, segLemToks...)
		splits = append(splits, len(segLemToks))

		// Build meta from existing structure.
		meta := SegMeta{}
		// Reconstruct text from lem tokens.
		for _, t := range segLemToks {
			meta.Text += t.Surface
		}
		// Collect rdg tokens from <app><rdg><w>.
		for _, app := range seg.SelectElements("app") {
			if lem := app.SelectElement("lem"); lem != nil {
				var sb strings.Builder
				collectLemText(lem, &sb)
				meta.LemText += stripSpace(sb.String())
			}
			if rdg := app.SelectElement("rdg"); rdg != nil {
				var sb strings.Builder
				collectLemText(rdg, &sb)
				meta.RdgText += stripSpace(sb.String())
				for _, w := range rdg.FindElements(".//w") {
					meta.RdgTokens = append(meta.RdgTokens, Token{
						Surface:  w.Text(),
						LemmaRef: w.SelectAttrValue("lemmaRef", ""),
					})
				}
			}
		}
		metas = append(metas, meta)
	}
	return
}

// collectLemTokens walks el and appends <w> tokens found in lem positions:
// direct <w> children, and <w> inside <app><lem>. <rdg> content is skipped.
func collectLemTokens(el *etree.Element, tokens *[]Token) {
	for _, child := range el.Child {
		e, ok := child.(*etree.Element)
		if !ok {
			continue
		}
		switch e.Tag {
		case "w":
			*tokens = append(*tokens, Token{
				Surface:  e.Text(),
				LemmaRef: e.SelectAttrValue("lemmaRef", ""),
			})
		case "app":
			if lem := e.SelectElement("lem"); lem != nil {
				collectLemTokens(lem, tokens)
			}
		default:
			collectLemTokens(e, tokens)
		}
	}
}

// collectLemText recursively collects text content, treating <app><lem> as the
// canonical reading and skipping <rdg>.
func collectLemText(el *etree.Element, sb *strings.Builder) {
	for _, child := range el.Child {
		switch t := child.(type) {
		case *etree.CharData:
			sb.WriteString(t.Data)
		case *etree.Element:
			switch t.Tag {
			case "app":
				if lem := t.SelectElement("lem"); lem != nil {
					collectLemText(lem, sb)
				}
			case "rdg":
				// Skip variant readings.
			default:
				collectLemText(t, sb)
			}
		}
	}
}

// lemmaInfo holds the dictionary form and kana reading for a single entry.
type lemmaInfo struct {
	Orth    string // form[@type='lemma']/orth (kanji/mixed)
	Reading string // form[@type='lemma']/pron[@notation='kana']
}

// buildLemmaInfo builds a map from Dict A hom ID (reading.lemma, e.g. "おもひ.思ふ")
// to the orth and kana reading from the corresponding Dict B entry.
func buildLemmaInfo(doc *etree.Document) map[string]lemmaInfo {
	// Build Dict B: entry ID → (orth, reading).
	dictB := make(map[string]lemmaInfo)
	for _, entry := range doc.FindElements("//back//div[@type='dictionary']/entry") {
		id := entry.SelectAttrValue("xml:id", "")
		if id == "" {
			continue
		}
		info := lemmaInfo{}
		if orth := entry.FindElement("form[@type='lemma']/orth"); orth != nil {
			info.Orth = orth.Text()
		}
		if pron := entry.FindElement("form[@type='lemma']/pron[@notation='kana']"); pron != nil {
			info.Reading = pron.Text()
		}
		dictB[id] = info
	}

	// Build Dict A hom map: hom ID → lemmaInfo (via <ref target="#dictBID">).
	m := make(map[string]lemmaInfo)
	for _, hom := range doc.FindElements("//back//div[@type='reading-index']/entry/hom") {
		homID := hom.SelectAttrValue("xml:id", "")
		if homID == "" {
			continue
		}
		ref := hom.SelectElement("ref")
		if ref == nil {
			continue
		}
		target := strings.TrimPrefix(ref.SelectAttrValue("target", ""), "#")
		if info, ok := dictB[target]; ok {
			m[homID] = info
		}
	}
	return m
}

// buildDictAMap builds a map from (reading, lemma) → hom xml:id from Dict A.
func buildDictAMap(doc *etree.Document) map[[2]string]string {
	m := make(map[[2]string]string)
	for _, hom := range doc.FindElements("//back//div[@type='reading-index']/entry/hom") {
		homID := hom.SelectAttrValue("xml:id", "")
		if homID == "" {
			continue
		}
		dot := strings.Index(homID, ".")
		if dot < 0 {
			continue
		}
		m[[2]string{homID[:dot], homID[dot+1:]}] = homID
	}
	return m
}

// kanaVariants returns candidate kana forms to try when a surface does not
// directly match a Dict A reading key. The Karoku manuscript systematically
// writes voiced kana as their unvoiced counterparts (清濁の差), while
// Hachidaishu KanjiReadings use the voiced form.
// Historical kana→modern substitutions (ひ→い, ふ→う) are NOT applied
// because Hachidaishu KanjiReadings already use historical kana.
// kanaPairs lists substitutions tried when a surface does not match a Dict A key.
// Two kinds:
//  1. 清濁 (voicing): Karoku清音 → Hachidaishu濁音 (e.g. わひ→わび, さら→ざら)
//  2. 歴史的仮名 (historical kana): Karoku modern → Hachidaishu historical
//     (e.g. おら→をら, え→ゑ, い→ゐ)
var kanaPairs = [][2]rune{
	// 清濁
	{'か', 'が'}, {'き', 'ぎ'}, {'く', 'ぐ'}, {'け', 'げ'}, {'こ', 'ご'},
	{'さ', 'ざ'}, {'し', 'じ'}, {'す', 'ず'}, {'せ', 'ぜ'}, {'そ', 'ぞ'},
	{'た', 'だ'}, {'ち', 'ぢ'}, {'つ', 'づ'}, {'て', 'で'}, {'と', 'ど'},
	{'は', 'ば'}, {'ひ', 'び'}, {'ふ', 'ぶ'}, {'へ', 'べ'}, {'ほ', 'ぼ'},
	// 歴史的仮名
	{'お', 'を'}, {'え', 'ゑ'}, {'い', 'ゐ'},
}

func kanaVariants(s string) []string {
	runes := []rune(s)
	var variants []string
	for _, pair := range kanaPairs {
		for i, r := range runes {
			if r == pair[0] {
				v := make([]rune, len(runes))
				copy(v, runes)
				v[i] = pair[1]
				variants = append(variants, string(v))
			}
		}
	}
	return variants
}

// RefineTokenRefs updates lemmaRef values in tokens using Dict A lookup.
// For each token where surface ≠ current reading, it tries:
//  1. (surface, lemma) exact lookup in Dict A
//  2. (normalized_surface, lemma) lookup for historical kana variants
//
// This corrects lemmaRefs left over from migration that used the lemma
// reading instead of the actual surface kana.
func RefineTokenRefs(tokens []Token, doc *etree.Document) []Token {
	dictA := buildDictAMap(doc)
	result := make([]Token, len(tokens))
	copy(result, tokens)

	for i, tok := range result {
		homID := strings.TrimPrefix(tok.LemmaRef, "#")
		dot := strings.Index(homID, ".")
		if dot < 0 {
			continue
		}
		currentReading := homID[:dot]
		lemma := homID[dot+1:]
		surface := tok.Surface
		if surface == "" || surface == currentReading {
			continue
		}
		// Layer 1: exact surface lookup.
		if newID, ok := dictA[[2]string{surface, lemma}]; ok {
			result[i].LemmaRef = "#" + newID
			continue
		}
		// Layer 2: kana variant lookup.
		for _, variant := range kanaVariants(surface) {
			if newID, ok := dictA[[2]string{variant, lemma}]; ok {
				result[i].LemmaRef = "#" + newID
				break
			}
		}
	}
	return result
}

// HachiTokens extracts the ordered token list for poem n from the Hachidaishu
// document (looks for <lg type="waka" n="N">). Lemma and Reading are populated
// from the dictionary <back> for use as rune-count proxies.
func HachiTokens(doc *etree.Document, n int) []Token {
	path := fmt.Sprintf("//lg[@type='waka'][@n='%d']", n)
	lg := doc.FindElement(path)
	if lg == nil {
		return nil
	}
	lemmas := buildLemmaInfo(doc)
	var tokens []Token
	for _, w := range lg.FindElements(".//w") {
		ref := w.SelectAttrValue("lemmaRef", "")
		lemmaKey := strings.TrimPrefix(ref, "#")
		info := lemmas[lemmaKey]
		tokens = append(tokens, Token{
			Surface:  w.Text(),
			LemmaRef: ref,
			Lemma:    info.Orth,
			Reading:  info.Reading,
		})
	}
	return tokens
}

// AnnotateDoc walks all <l n="N"> elements in mergedDoc's body, looks up the
// corresponding Hachidaishu token list, runs AlignPoem, and if fully matched
// rewrites each <seg> child to contain <w lemmaRef="…"> elements while
// preserving existing structural elements like <app>/<lem>/<rdg>.
//
// It returns counts of matched, skipped (no hachi entry), and unmatched poems,
// along with the list of unmatched poem numbers.
func AnnotateDoc(hachiDoc, mergedDoc *etree.Document) (matched, skipped, unmatched int, unmatchedPoems []int) {
	for _, l := range mergedDoc.FindElements("//body//l") {
		nStr := l.SelectAttrValue("n", "")
		n, err := strconv.Atoi(nStr)
		if err != nil {
			continue
		}

		tokens := HachiTokens(hachiDoc, n)
		if tokens == nil {
			skipped++
			continue
		}

		segs := l.SelectElements("seg")
		segTexts := make([]string, len(segs))
		for i, seg := range segs {
			segTexts[i] = SegText(seg)
		}

		aligned, ok := AlignPoem(tokens, segTexts)
		if !ok {
			unmatched++
			unmatchedPoems = append(unmatchedPoems, n)
			continue
		}

		groups := make([]SegGroup, len(aligned))
		for j, toks := range aligned {
			groups[j] = SegGroup{Lem: toks}
		}
		ApplyAlignment(segs, groups)
		matched++
	}
	return
}

// ApplyAlignment rewrites <seg> elements in-place, inserting <w lemmaRef="…">
// elements around tokens while preserving existing child structure such as
// <app>/<lem>/<rdg>. Text inside <lem> is annotated; <rdg> is annotated when
// group.Rdg is non-nil, otherwise left unchanged.
func ApplyAlignment(segs []*etree.Element, groups []SegGroup) {
	for i, seg := range segs {
		ti := 0
		seg.Child = rewriteChildren(seg.Child, groups[i].Lem, &ti)
		if groups[i].Rdg != nil {
			ri := 0
			applyRdgTokens(seg, groups[i].Rdg, &ri)
		}
	}
}

// applyRdgTokens walks all <app><rdg> elements inside seg and rewrites their
// text content with <w> elements, consuming rdgTokens left-to-right.
func applyRdgTokens(seg *etree.Element, rdgTokens []Token, ri *int) {
	for _, app := range seg.SelectElements("app") {
		if rdg := app.SelectElement("rdg"); rdg != nil {
			rdg.Child = rewriteRdgChildren(rdg.Child, rdgTokens, ri)
		}
	}
}

// rewriteRdgChildren wraps CharData inside a <rdg> element with <w> elements.
// Existing <w> children are unwrapped to their text content first so that
// re-applying a draft overwrites any previous annotation correctly.
// XML indentation whitespace is stripped before matching so that multi-line
// formatting does not produce spurious blank nodes in the output.
func rewriteRdgChildren(children []etree.Token, tokens []Token, ti *int) []etree.Token {
	var result []etree.Token
	for _, child := range children {
		switch t := child.(type) {
		case *etree.CharData:
			text := stripSpace(t.Data)
			if text == "" {
				continue
			}
			result = append(result, wrapText(text, tokens, ti)...)
		case *etree.Element:
			if t.Tag == "w" {
				// Unwrap existing <w> and re-annotate its text.
				text := stripSpace(t.Text())
				if text != "" {
					result = append(result, wrapText(text, tokens, ti)...)
				}
			} else {
				result = append(result, t)
			}
		default:
			result = append(result, child)
		}
	}
	return result
}

// rewriteChildren walks a child list, wrapping text tokens in <w> elements and
// descending into <app><lem> for annotation while preserving all other nodes.
// XML indentation whitespace in CharData is stripped before matching so that
// multi-line <seg> formatting does not produce spurious blank nodes.
func rewriteChildren(children []etree.Token, tokens []Token, ti *int) []etree.Token {
	var result []etree.Token
	for _, child := range children {
		switch t := child.(type) {
		case *etree.CharData:
			text := stripSpace(t.Data)
			if text == "" {
				continue
			}
			result = append(result, wrapText(text, tokens, ti)...)
		case *etree.Element:
			if t.Tag == "app" {
				if lem := t.SelectElement("lem"); lem != nil {
					lem.Child = rewriteChildren(lem.Child, tokens, ti)
				}
				result = append(result, t)
			} else if t.Tag == "w" {
				// Unwrap existing <w> and re-annotate its text.
				text := stripSpace(t.Text())
				if text != "" {
					result = append(result, wrapText(text, tokens, ti)...)
				}
			} else {
				result = append(result, t)
			}
		default:
			result = append(result, child)
		}
	}
	// Flush any empty-surface tokens that trail after all text is consumed
	// (e.g. an elided token at the very end of a segment).
	for *ti < len(tokens) && tokens[*ti].Surface == "" {
		w := etree.NewElement("w")
		w.CreateAttr("lemmaRef", tokens[*ti].LemmaRef)
		result = append(result, w)
		(*ti)++
	}
	return result
}

// wrapText consumes tokens from tokens[*ti:] that match the leading text and
// returns a slice of <w> elements. Any remaining text that is not consumed by
// tokens (e.g. whitespace) is emitted as CharData.
//
// Tokens with an empty Surface (elided in Karoku but present in Hachidaishu)
// are emitted as empty <w> elements (no text content) at the current position.
func wrapText(text string, tokens []Token, ti *int) []etree.Token {
	var result []etree.Token
	pos := 0

	drainEmpty := func() {
		for *ti < len(tokens) && tokens[*ti].Surface == "" {
			w := etree.NewElement("w")
			w.CreateAttr("lemmaRef", tokens[*ti].LemmaRef)
			result = append(result, w)
			(*ti)++
		}
	}

	for pos < len(text) {
		drainEmpty()
		if *ti >= len(tokens) {
			break
		}
		surf := tokens[*ti].Surface
		if !strings.HasPrefix(text[pos:], surf) {
			break
		}
		w := etree.NewElement("w")
		w.CreateAttr("lemmaRef", tokens[*ti].LemmaRef)
		w.SetText(surf)
		result = append(result, w)
		pos += len(surf)
		(*ti)++
	}
	drainEmpty() // flush any trailing empty-surface tokens
	// Any unconsumed text (e.g. whitespace between elements) passes through.
	if pos < len(text) {
		result = append(result, &etree.CharData{Data: text[pos:]})
	}
	return result
}
