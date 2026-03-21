package wordlist

import (
	"fmt"
	"log"

	"github.com/beevik/etree"
)

const teiNS = "http://www.tei-c.org/ns/1.0"

// ExtractTokens walks the XML document and collects all <w> elements with
// their pos, lemma, msd attributes. It also detects compounds, modern forms,
// and alternative senses from <app><rdg> blocks.
// Returns (tokens, compoundMap keyed by lemma, modernRefs keyed by lemma).
func ExtractTokens(doc *etree.Document) ([]Token, map[string][]CompoundPart, map[string]ModernRef) {
	var tokens []Token
	compounds := make(map[string][]CompoundPart)
	modernRefs := make(map[string]ModernRef)

	// Find all <app> elements to detect compounds and modern forms.
	for _, app := range doc.FindElements("//app") {
		info := analyzeApp(app)
		if info == nil {
			continue
		}
		if len(info.Parts) > 0 {
			if _, exists := compounds[info.CompoundLemma]; !exists {
				compounds[info.CompoundLemma] = info.Parts
			}
		}
		if info.Modern != nil {
			if _, exists := modernRefs[info.CompoundLemma]; !exists {
				modernRefs[info.CompoundLemma] = *info.Modern
			}
		}
	}

	// Mark <w> elements inside 2nd+ <rdg> of any <app>.
	secondRdgWs := make(map[*etree.Element]bool)
	for _, app := range doc.FindElements("//app") {
		rdgs := app.SelectElements("rdg")
		for _, rdg := range rdgs[1:] {
			for _, w := range rdg.SelectElements("w") {
				secondRdgWs[w] = true
			}
		}
	}

	// Collect all <w> elements.
	for _, w := range doc.FindElements("//w") {
		pos := w.SelectAttrValue("pos", "")
		lemma := w.SelectAttrValue("lemma", "")
		msdStr := w.SelectAttrValue("msd", "")
		msd := ParseMSD(msdStr)
		surface := w.Text()

		tokens = append(tokens, Token{
			Pos:         pos,
			Lemma:       lemma,
			MSD:         msd,
			Surface:     surface,
			InSecondRdg: secondRdgWs[w],
		})
	}

	return tokens, compounds, modernRefs
}

// AppInfo holds all information extracted from a single <app> block.
type AppInfo struct {
	CompoundLemma string
	Parts         []CompoundPart
	Modern        *ModernRef
}

// analyzeApp extracts compound, decomposition, modern form, and alternative
// sense information from a single <app> block.
func analyzeApp(app *etree.Element) *AppInfo {
	rdgs := app.SelectElements("rdg")
	if len(rdgs) < 2 {
		return nil
	}

	firstRdg := rdgs[0]
	firstWords := firstRdg.SelectElements("w")
	if len(firstWords) != 1 {
		return nil
	}

	compW := firstWords[0]
	compLemma := compW.SelectAttrValue("lemma", "")

	info := &AppInfo{
		CompoundLemma: compLemma,
	}

	for _, rdg := range rdgs[1:] {
		words := rdg.SelectElements("w")
		switch {
		case len(words) > 1:
			// Decomposition rdg.
			if len(info.Parts) == 0 {
				for _, w := range words {
					info.Parts = append(info.Parts, CompoundPart{
						Lemma: w.SelectAttrValue("lemma", ""),
					})
				}
			}
		case len(words) == 1:
			w := words[0]
			wLemma := w.SelectAttrValue("lemma", "")

			if wLemma != compLemma {
				// Different lemma → modern form reference.
				if info.Modern == nil {
					info.Modern = &ModernRef{
						Lemma: wLemma,
					}
				}
			}
			// Same lemma with different classification → already handled
			// via token collection (alternative senses merge into entry).
		}
	}

	return info
}

// FlattenApps replaces each <app> element with the first <rdg>'s content.
// Returns the number of <app> blocks flattened.
func FlattenApps(doc *etree.Document) int {
	apps := doc.FindElements("//app")
	count := 0
	for _, app := range apps {
		rdgs := app.SelectElements("rdg")
		if len(rdgs) == 0 {
			continue
		}

		parent := app.Parent()
		if parent == nil {
			continue
		}

		firstRdg := rdgs[0]
		children := firstRdg.ChildElements()

		for _, child := range children {
			firstRdg.RemoveChild(child)
			parent.InsertChildAt(app.Index(), child)
		}

		parent.RemoveChild(app)
		count++
	}
	return count
}

// pronRefKey identifies a token's Dict A hom by reading and lemma.
type pronRefKey struct {
	reading string
	lemma   string
}

// TransformBody adds lemmaRef to each <w> and removes pos, lemma, msd.
// lemmaRef points to Dict A hom IDs (reading.orth format).
func TransformBody(doc *etree.Document, pronEntries []*PronEntry) {
	refMap := make(map[pronRefKey]string)
	for _, pe := range pronEntries {
		for _, hom := range pe.Homs {
			refMap[pronRefKey{pe.Reading, hom.Lemma}] = "#" + hom.ID
		}
	}

	for _, w := range doc.FindElements("//w") {
		lemma := w.SelectAttrValue("lemma", "")
		msdStr := w.SelectAttrValue("msd", "")
		msd := ParseMSD(msdStr)

		// Use surface kana (KanjiReading) as primary key; fall back to LemmaReading.
		reading := msd.KanjiReading
		if reading == "" || reading == "???" || reading == "-" {
			reading = msd.LemmaReading
		}

		k := pronRefKey{reading, lemma}
		if id, ok := refMap[k]; ok {
			w.CreateAttr("lemmaRef", id)
		} else if lemma != "" {
			w.CreateAttr("lemmaRef", "#"+EntryID(lemma))
			log.Printf("warning: no pron entry for reading=%q lemma=%q, using fallback", reading, lemma)
		} else {
			log.Printf("warning: no entry for lemma=%q", lemma)
		}

		w.RemoveAttr("pos")
		w.RemoveAttr("lemma")
		w.RemoveAttr("msd")
	}
}

// BuildBackDiv creates the <back> element tree with two dictionary divs and classification divs.
// Dict A (reading-index): kana-first, hom per orthographic form.
// Dict B (dictionary): lemma-keyed, with gramGrp and sense.
func BuildBackDiv(entries []*Entry, pronEntries []*PronEntry, classWLSPH, classWLSP *etree.Element) *etree.Element {
	back := etree.NewElement("back")

	// Dict A: reading index.
	if len(pronEntries) > 0 {
		idxDiv := back.CreateElement("div")
		idxDiv.CreateAttr("type", "reading-index")
		idxHead := idxDiv.CreateElement("head")
		idxHead.SetText("Reading Index")

		for _, pe := range pronEntries {
			entry := idxDiv.CreateElement("entry")
			entry.CreateAttr("xml:id", pe.ID)
			form := entry.CreateElement("form")
			form.CreateAttr("type", "pron")
			pron := form.CreateElement("pron")
			pron.CreateAttr("notation", "kana")
			pron.SetText(pe.Reading)
			for _, hom := range pe.Homs {
				h := entry.CreateElement("hom")
				h.CreateAttr("n", fmt.Sprintf("%d", hom.N))
				h.CreateAttr("xml:id", hom.ID)
				ref := h.CreateElement("ref")
				ref.CreateAttr("target", "#"+hom.RefID)
			}
		}
	}

	// Dict B: dictionary.
	div := back.CreateElement("div")
	div.CreateAttr("type", "dictionary")
	head := div.CreateElement("head")
	head.SetText("Wordlist")

	for _, e := range entries {
		entry := div.CreateElement("entry")
		entry.CreateAttr("xml:id", e.ID)
		if e.IsCompound {
			entry.CreateAttr("type", "compound")
		} else {
			entry.CreateAttr("type", "simplex")
		}

		// <form type="lemma"> with one <orth> and multiple <pron>.
		form := entry.CreateElement("form")
		form.CreateAttr("type", "lemma")
		orth := form.CreateElement("orth")
		orth.CreateAttr("xml:lang", "ja")
		orth.SetText(e.Lemma)
		for _, reading := range e.LemmaReadings {
			pron := form.CreateElement("pron")
			pron.CreateAttr("notation", "kana")
			pron.SetText(reading)
		}

		// <form type="compound"> for compound entries.
		if e.IsCompound && len(e.Parts) > 0 {
			compForm := entry.CreateElement("form")
			compForm.CreateAttr("type", "compound")
			for _, p := range e.Parts {
				ref := compForm.CreateElement("ref")
				ref.CreateAttr("target", "#"+EntryID(p.Lemma))
				ref.SetText(p.Lemma)
			}
		}

		// <form type="modern"> for modern form cross-references.
		if e.Modern != nil {
			modForm := entry.CreateElement("form")
			modForm.CreateAttr("type", "modern")
			ref := modForm.CreateElement("ref")
			ref.CreateAttr("target", "#"+EntryID(e.Modern.Lemma))
			ref.SetText(e.Modern.Lemma)
		}

		// Grammar info.
		if e.NeedsHom() {
			for i, g := range e.Grams {
				hom := entry.CreateElement("hom")
				hom.CreateAttr("n", fmt.Sprintf("%d", i+1))
				hom.CreateAttr("xml:id", e.HomID(i+1))
				addGramGrp(hom, g)
			}
		} else if len(e.Grams) > 0 {
			addGramGrp(entry, e.Grams[0])
		}

		// Senses with @ana referencing classification lists.
		for _, s := range e.Senses {
			sense := entry.CreateElement("sense")
			if e.NeedsSenseIDs() {
				sense.CreateAttr("xml:id", e.SenseID(s.N))
			}

			ana := buildAnaRef(s)
			if ana != "" {
				sense.CreateAttr("ana", ana)
			}

			if s.WLSPDescription != "" {
				def := sense.CreateElement("def")
				def.CreateAttr("xml:lang", "ja")
				def.SetText(s.WLSPDescription)
			}
		}
	}

	if classWLSPH != nil {
		back.AddChild(classWLSPH)
	}
	if classWLSP != nil {
		back.AddChild(classWLSP)
	}

	return back
}

// buildAnaRef constructs the @ana attribute value from a Sense.
func buildAnaRef(s Sense) string {
	var refs []string
	if s.WLSPH != "" {
		refs = append(refs, "#WLSPH."+s.WLSPH)
	}
	if s.WLSP != "" {
		refs = append(refs, "#WLSP."+s.WLSP)
	}
	if len(refs) == 0 {
		return ""
	}
	result := refs[0]
	for _, r := range refs[1:] {
		result += " " + r
	}
	return result
}

func addGramGrp(parent *etree.Element, g GramInfo) {
	gramGrp := parent.CreateElement("gramGrp")
	pos := gramGrp.CreateElement("pos")
	pos.CreateAttr("value", g.Pos)
	pos.SetText(g.Pos)
	if g.UPosTag != "" {
		gram := gramGrp.CreateElement("gram")
		gram.CreateAttr("type", "UPosTag")
		gram.SetText(g.UPosTag)
	}
	if g.IPAPosTag != "" {
		gram := gramGrp.CreateElement("gram")
		gram.CreateAttr("type", "IPAPosTag")
		gram.SetText(g.IPAPosTag)
	}
	if g.UniDicPosTag != "" {
		gram := gramGrp.CreateElement("gram")
		gram.CreateAttr("type", "UniDicPosTag")
		gram.SetText(g.UniDicPosTag)
	}
}

// InsertBack inserts the <back> element after </body> in the document.
func InsertBack(doc *etree.Document, back *etree.Element) {
	tei := doc.SelectElement("TEI")
	if tei == nil {
		log.Fatal("no <TEI> root element found")
	}
	text := tei.SelectElement("text")
	if text == nil {
		log.Fatal("no <text> element found")
	}
	text.AddChild(back)
}

// UpdateHeader adds WLSP taxonomy to <classDecl> and interpretation note to <encodingDesc>.
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

	alreadyExists := false
	for _, tax := range classDecl.SelectElements("taxonomy") {
		if tax.SelectAttrValue("xml:id", "") == "WLSP" {
			alreadyExists = true
			break
		}
	}
	if !alreadyExists {
		tax := classDecl.CreateElement("taxonomy")
		tax.CreateAttr("xml:id", "WLSP")
		bibl := tax.CreateElement("bibl")
		title := bibl.CreateElement("title")
		title.SetText("分類語彙表 (Word List by Semantic Principle)")
		ptr := bibl.CreateElement("ptr")
		ptr.CreateAttr("target", "https://clrd.ninjal.ac.jp/btsj/")
	}

	editorialDecl := enc.SelectElement("editorialDecl")
	if editorialDecl == nil {
		editorialDecl = enc.CreateElement("editorialDecl")
	}

	if editorialDecl.SelectElement("interpretation") == nil {
		interp := editorialDecl.CreateElement("interpretation")
		p := interp.CreateElement("p")
		p.SetText("Each ")
		gi1 := p.CreateElement("gi")
		gi1.SetText("w")
		p.CreateText(" element carries a ")
		att1 := p.CreateElement("att")
		att1.SetText("lemmaRef")
		p.CreateText(" attribute pointing to the corresponding ")
		gi2 := p.CreateElement("gi")
		gi2.SetText("entry")
		p.CreateText(" in the back-matter dictionary. Lemma, POS, and morphological features are recorded only in the dictionary entry.")
	}
}

// ReadDocument reads a TEI XML file into an etree Document.
func ReadDocument(path string) (*etree.Document, error) {
	doc := etree.NewDocument()
	doc.ReadSettings.PreserveCData = true
	if err := doc.ReadFromFile(path); err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return doc, nil
}

// WriteDocument writes an etree Document to a file with standard XML settings.
func WriteDocument(doc *etree.Document, path string) error {
	doc.Indent(2)
	doc.WriteSettings.CanonicalAttrVal = true
	doc.WriteSettings.CanonicalEndTags = false
	doc.WriteSettings.CanonicalText = true
	return doc.WriteToFile(path)
}
