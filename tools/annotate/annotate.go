package annotate

import (
	"fmt"
	"strconv"
	"strings"

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
	doc.Indent(2)
	doc.WriteSettings.CanonicalAttrVal = true
	doc.WriteSettings.CanonicalEndTags = false
	doc.WriteSettings.CanonicalText = true
	return doc.WriteToFile(path)
}

// SegText extracts the canonical text of a <seg>, including text inside
// <app><lem> but excluding <rdg> variants and other non-text elements.
func SegText(seg *etree.Element) string {
	var sb strings.Builder
	collectLemText(seg, &sb)
	return sb.String()
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

// HachiTokens extracts the ordered token list for poem n from the Hachidaishu
// document (looks for <lg type="waka" n="N">).
func HachiTokens(doc *etree.Document, n int) []Token {
	path := fmt.Sprintf("//lg[@type='waka'][@n='%d']", n)
	lg := doc.FindElement(path)
	if lg == nil {
		return nil
	}
	var tokens []Token
	for _, w := range lg.FindElements(".//w") {
		ref := w.SelectAttrValue("lemmaRef", "")
		tokens = append(tokens, Token{Surface: w.Text(), LemmaRef: ref})
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

		ApplyAlignment(segs, aligned)
		matched++
	}
	return
}

// ApplyAlignment rewrites <seg> elements in-place, inserting <w lemmaRef="…">
// elements around tokens while preserving existing child structure such as
// <app>/<lem>/<rdg>. Text inside <lem> is annotated; <rdg> is left unchanged.
func ApplyAlignment(segs []*etree.Element, aligned [][]Token) {
	for i, seg := range segs {
		ti := 0
		seg.Child = rewriteChildren(seg.Child, aligned[i], &ti)
	}
}

// rewriteChildren walks a child list, wrapping text tokens in <w> elements and
// descending into <app><lem> for annotation while preserving all other nodes.
func rewriteChildren(children []etree.Token, tokens []Token, ti *int) []etree.Token {
	var result []etree.Token
	for _, child := range children {
		switch t := child.(type) {
		case *etree.CharData:
			result = append(result, wrapText(t.Data, tokens, ti)...)
		case *etree.Element:
			if t.Tag == "app" {
				if lem := t.SelectElement("lem"); lem != nil {
					lem.Child = rewriteChildren(lem.Child, tokens, ti)
				}
				result = append(result, t)
			} else {
				result = append(result, t)
			}
		default:
			result = append(result, child)
		}
	}
	return result
}

// wrapText consumes tokens from tokens[*ti:] that match the leading text and
// returns a slice of <w> elements. Any remaining text that is not consumed by
// tokens (e.g. whitespace) is emitted as CharData.
func wrapText(text string, tokens []Token, ti *int) []etree.Token {
	var result []etree.Token
	pos := 0
	for pos < len(text) {
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
	// Any unconsumed text (e.g. whitespace between elements) passes through.
	if pos < len(text) {
		result = append(result, &etree.CharData{Data: text[pos:]})
	}
	return result
}
