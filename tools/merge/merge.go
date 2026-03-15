// Package merge provides functions for embedding a Hachidaishu wordlist
// into a Karoku 2 TEI XML base file.
package merge

import (
	"fmt"
	"strings"

	"github.com/beevik/etree"
)

// ExtractBackDivs returns the dictionary and classification divs from
// the hachidaishu-wordlist <back> element, in document order.
func ExtractBackDivs(wordlistDoc *etree.Document) ([]*etree.Element, error) {
	back := wordlistDoc.FindElement("//back")
	if back == nil {
		return nil, fmt.Errorf("no <back> element in wordlist document")
	}

	var divs []*etree.Element
	for _, child := range back.ChildElements() {
		if child.Tag == "div" {
			divs = append(divs, child)
		}
	}
	if len(divs) == 0 {
		return nil, fmt.Errorf("no <div> elements found in wordlist <back>")
	}
	return divs, nil
}

// PrependToBack inserts the given elements at the beginning of the <back>
// element in doc (which already contains Karoku 2 colophons / listPerson).
// The <back> is created inside <text> if absent.
func PrependToBack(doc *etree.Document, divs []*etree.Element) error {
	tei := doc.SelectElement("TEI")
	if tei == nil {
		return fmt.Errorf("no <TEI> root element")
	}
	text := tei.SelectElement("text")
	if text == nil {
		return fmt.Errorf("no <text> element")
	}
	back := text.SelectElement("back")
	if back == nil {
		back = text.CreateElement("back")
	}

	// Insert in reverse order at position 0 so they end up in original order.
	for i := len(divs) - 1; i >= 0; i-- {
		div := divs[i].Copy()
		back.InsertChildAt(0, div)
	}
	return nil
}

// ReplaceBackDivs removes any existing wordlist divs (type="dictionary" or
// type="classification") from <back>, then prepends the new divs from the
// updated wordlist. Safe to run on both freshly-merged and already-annotated
// files without producing duplicate entries.
func ReplaceBackDivs(doc *etree.Document, divs []*etree.Element) error {
	tei := doc.SelectElement("TEI")
	if tei == nil {
		return fmt.Errorf("no <TEI> root element")
	}
	text := tei.SelectElement("text")
	if text == nil {
		return fmt.Errorf("no <text> element")
	}
	back := text.SelectElement("back")
	if back == nil {
		back = text.CreateElement("back")
	}

	// Remove existing wordlist divs.
	for _, div := range back.SelectElements("div") {
		t := div.SelectAttrValue("type", "")
		if t == "dictionary" || t == "classification" {
			back.RemoveChild(div)
		}
	}

	// Insert new divs in reverse order at position 0.
	for i := len(divs) - 1; i >= 0; i-- {
		div := divs[i].Copy()
		back.InsertChildAt(0, div)
	}
	return nil
}

// UpdateHeader adds WLSP/WLSPH taxonomy entries to <classDecl> and an
// interpretation note to <editorialDecl> in doc's <encodingDesc>.
func UpdateHeader(doc *etree.Document) {
	tei := doc.SelectElement("TEI")
	if tei == nil {
		return
	}
	header := tei.SelectElement("teiHeader")
	if header == nil {
		return
	}
	enc := header.SelectElement("encodingDesc")
	if enc == nil {
		enc = header.CreateElement("encodingDesc")
	}

	classDecl := enc.SelectElement("classDecl")
	if classDecl == nil {
		classDecl = enc.CreateElement("classDecl")
	}

	addTaxonomy(classDecl, "WLSPH", "分類語彙表 (Word List by Semantic Principles for Historical Japanese)")
	addTaxonomy(classDecl, "WLSP", "分類語彙表 (Word List by Semantic Principle)")

	editorialDecl := enc.SelectElement("editorialDecl")
	if editorialDecl == nil {
		editorialDecl = enc.CreateElement("editorialDecl")
	}
	if editorialDecl.SelectElement("interpretation") == nil {
		interp := editorialDecl.CreateElement("interpretation")
		p := interp.CreateElement("p")
		p.SetText("Lexical annotations from the Hachidaishu dataset are embedded in the back-matter dictionary. Each ")
		gi := p.CreateElement("gi")
		gi.SetText("w")
		p.CreateText(" element in the body carries a ")
		att := p.CreateElement("att")
		att.SetText("lemmaRef")
		p.CreateText(" attribute pointing to the corresponding ")
		gi2 := p.CreateElement("gi")
		gi2.SetText("entry")
		p.CreateText(" in the back-matter dictionary.")
	}
}

func addTaxonomy(classDecl *etree.Element, xmlID, title string) {
	for _, tax := range classDecl.SelectElements("taxonomy") {
		if tax.SelectAttrValue("xml:id", "") == xmlID {
			return
		}
	}
	tax := classDecl.CreateElement("taxonomy")
	tax.CreateAttr("xml:id", xmlID)
	bibl := tax.CreateElement("bibl")
	t := bibl.CreateElement("title")
	t.SetText(title)
}

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
