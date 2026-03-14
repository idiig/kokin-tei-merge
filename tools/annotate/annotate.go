package annotate

import (
	"fmt"
	"strconv"

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
// rewrites each <seg> child to contain <w lemmaRef="…"> elements.
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
			segTexts[i] = seg.Text()
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

// ApplyAlignment rewrites <seg> elements in-place, replacing their text
// content with <w lemmaRef="…"> children according to the aligned token slices.
func ApplyAlignment(segs []*etree.Element, aligned [][]Token) {
	for i, seg := range segs {
		seg.Child = nil
		for _, tok := range aligned[i] {
			w := etree.NewElement("w")
			w.CreateAttr("lemmaRef", tok.LemmaRef)
			w.SetText(tok.Surface)
			seg.AddChild(w)
		}
	}
}
