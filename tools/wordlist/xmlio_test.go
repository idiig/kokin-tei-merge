package wordlist

import (
	"strings"
	"testing"

	"github.com/beevik/etree"
)

const testXML = `<?xml version="1.0" encoding="UTF-8"?>
<TEI xmlns="http://www.tei-c.org/ns/1.0" xml:lang="ja">
  <teiHeader>
    <fileDesc>
      <titleStmt><title>Test</title></titleStmt>
      <publicationStmt><p>Test</p></publicationStmt>
      <sourceDesc><p>Test</p></sourceDesc>
    </fileDesc>
    <encodingDesc>
      <classDecl>
        <taxonomy xml:id="NDC">
          <bibl><title>Nippon Decimal Classification</title></bibl>
        </taxonomy>
      </classDecl>
    </encodingDesc>
  </teiHeader>
  <text>
    <body>
      <div type="anthology" n="Kokinshu">
        <lg type="waka" n="1">
          <l>
            <w pos="N.g" lemma="年" msd="UPosTag=NOUN|IPAPosTag=名詞-一般|UniDicPosTag=名詞-普通名詞-一般|LemmaReading=とし|Kanji=年|KanjiReading=とし|WLSPH=1.1630|WLSP=1.1630|WLSPDescription=体-関係-時間-年">年</w>
            <w pos="P.c.g" lemma="の" msd="UPosTag=ADP|IPAPosTag=助詞-格助詞-一般|UniDicPosTag=助詞-格助詞|LemmaReading=の|Kanji=の|KanjiReading=の|WLSPH=8.0061">の</w>
            <app>
              <rdg>
                <w pos="N.g" lemma="一年" msd="UPosTag=NOUN|IPAPosTag=名詞-一般|UniDicPosTag=名詞-普通名詞-一般|LemmaReading=ひととせ|Kanji=一年|KanjiReading=ひととせ|WLSPH=1.1950|WLSP=1.1950|WLSPDescription=体-関係-量-単位">一とせ</w>
              </rdg>
              <rdg>
                <w pos="N.Num" lemma="一" msd="UPosTag=NUM|IPAPosTag=名詞-数|UniDicPosTag=名詞-数詞|LemmaReading=いち|Kanji=一|KanjiReading=-|WLSPH=1.1950|WLSP=1.1950|WLSPDescription=体-関係-量-単位">-</w>
                <w pos="N.g" lemma="年" msd="UPosTag=NOUN|IPAPosTag=名詞-一般|UniDicPosTag=名詞-普通名詞-一般|LemmaReading=とし|Kanji=年|KanjiReading=-|WLSPH=1.1630|WLSP=1.1630|WLSPDescription=体-関係-時間-年">-</w>
              </rdg>
            </app>
          </l>
        </lg>
      </div>
    </body>
  </text>
</TEI>`

func parseTestDoc(t *testing.T) *etree.Document {
	t.Helper()
	doc := etree.NewDocument()
	if err := doc.ReadFromString(testXML); err != nil {
		t.Fatalf("failed to parse test XML: %v", err)
	}
	return doc
}

func TestExtractTokens(t *testing.T) {
	doc := parseTestDoc(t)
	tokens, compounds, modernRefs := ExtractTokens(doc)

	if len(tokens) != 5 {
		t.Fatalf("expected 5 tokens, got %d", len(tokens))
	}

	if tokens[0].Lemma != "年" {
		t.Errorf("tokens[0].Lemma = %q, want %q", tokens[0].Lemma, "年")
	}
	if tokens[0].MSD.LemmaReading != "とし" {
		t.Errorf("tokens[0].MSD.LemmaReading = %q, want %q", tokens[0].MSD.LemmaReading, "とし")
	}

	// Compound: 一年 with parts 一, 年 (keyed by lemma).
	parts, ok := compounds["一年"]
	if !ok {
		t.Fatal("expected compound entry for 一年")
	}
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(parts))
	}
	if parts[0].Lemma != "一" {
		t.Errorf("parts[0].Lemma = %q, want %q", parts[0].Lemma, "一")
	}
	if parts[1].Lemma != "年" {
		t.Errorf("parts[1].Lemma = %q, want %q", parts[1].Lemma, "年")
	}

	if len(modernRefs) != 0 {
		t.Errorf("expected 0 modern refs, got %d", len(modernRefs))
	}
}

func TestExtractTokens_ModernForm(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<TEI xmlns="http://www.tei-c.org/ns/1.0" xml:lang="ja">
  <teiHeader><fileDesc><titleStmt><title>T</title></titleStmt>
  <publicationStmt><p>T</p></publicationStmt><sourceDesc><p>T</p></sourceDesc></fileDesc></teiHeader>
  <text><body><div><lg type="waka" n="1"><l>
    <app>
      <rdg><w pos="V.free" lemma="留む" msd="UPosTag=VERB|LemmaReading=とどむ|WLSPH=2.1240">とどめ</w></rdg>
      <rdg><w pos="V.free" lemma="とめる" msd="UPosTag=VERB|LemmaReading=とめる|WLSPH=2.1240">とめる</w></rdg>
    </app>
  </l></lg></div></body></text>
</TEI>`

	doc := etree.NewDocument()
	if err := doc.ReadFromString(xml); err != nil {
		t.Fatal(err)
	}

	_, _, modernRefs := ExtractTokens(doc)
	ref, ok := modernRefs["留む"]
	if !ok {
		t.Fatal("expected modern ref for 留む")
	}
	if ref.Lemma != "とめる" {
		t.Errorf("modern ref lemma = %q, want %q", ref.Lemma, "とめる")
	}
}

func TestFlattenApps(t *testing.T) {
	doc := parseTestDoc(t)

	apps := doc.FindElements("//app")
	if len(apps) == 0 {
		t.Fatal("test XML should have <app> elements")
	}

	count := FlattenApps(doc)
	if count != 1 {
		t.Errorf("FlattenApps returned %d, want 1", count)
	}

	apps = doc.FindElements("//app")
	if len(apps) != 0 {
		t.Errorf("expected 0 <app> after flatten, got %d", len(apps))
	}

	found := false
	for _, w := range doc.FindElements("//l/w") {
		if w.SelectAttrValue("lemma", "") == "一年" {
			found = true
			if w.Text() != "一とせ" {
				t.Errorf("flattened w text = %q, want %q", w.Text(), "一とせ")
			}
		}
	}
	if !found {
		t.Error("flattened <w lemma=一年> not found in <l>")
	}
}

func TestBuildBackDiv(t *testing.T) {
	entries := []*Entry{
		{
			ID:            "w.年",
			Lemma:         "年",
			LemmaReadings: []string{"とし"},
			Grams:         []GramInfo{{Pos: "N.g", UPosTag: "NOUN", IPAPosTag: "名詞-一般", UniDicPosTag: "名詞-普通名詞-一般"}},
			Senses:        []Sense{{N: 1, WLSPH: "1.1630", WLSP: "1.1630", WLSPDescription: "体-関係-時間-年"}},
		},
	}

	back := BuildBackDiv(entries, nil, nil)

	entry := back.FindElement("//entry")
	if entry.SelectAttrValue("xml:id", "") != "w.年" {
		t.Errorf("entry xml:id = %q, want %q", entry.SelectAttrValue("xml:id", ""), "w.年")
	}
	if entry.SelectAttrValue("type", "") != "simplex" {
		t.Errorf("entry type = %q, want %q", entry.SelectAttrValue("type", ""), "simplex")
	}

	form := entry.SelectElement("form")
	orth := form.SelectElement("orth")
	if orth.Text() != "年" {
		t.Errorf("orth text = %q", orth.Text())
	}
	pron := form.SelectElement("pron")
	if pron.Text() != "とし" {
		t.Errorf("pron text = %q", pron.Text())
	}

	sense := entry.SelectElement("sense")
	ana := sense.SelectAttrValue("ana", "")
	if ana != "#WLSPH.1.1630 #WLSP.1.1630" {
		t.Errorf("sense ana = %q", ana)
	}
	if sense.SelectElement("usg") != nil {
		t.Error("sense should not have <usg>")
	}
}

func TestBuildBackDiv_MultipleReadings(t *testing.T) {
	entries := []*Entry{
		{
			ID:            "w.下",
			Lemma:         "下",
			LemmaReadings: []string{"した", "もと"},
			Grams:         []GramInfo{{Pos: "N.g", UPosTag: "NOUN"}},
			Senses:        []Sense{{N: 1, WLSPH: "1.1741"}},
		},
	}

	back := BuildBackDiv(entries, nil, nil)
	entry := back.FindElement("//entry")
	form := entry.SelectElement("form")
	prons := form.SelectElements("pron")
	if len(prons) != 2 {
		t.Fatalf("expected 2 <pron>, got %d", len(prons))
	}
	if prons[0].Text() != "した" {
		t.Errorf("pron[0] = %q", prons[0].Text())
	}
	if prons[1].Text() != "もと" {
		t.Errorf("pron[1] = %q", prons[1].Text())
	}
}

func TestBuildBackDiv_Compound(t *testing.T) {
	entries := []*Entry{
		{
			ID:            "w.一年",
			Lemma:         "一年",
			LemmaReadings: []string{"ひととせ"},
			Grams:         []GramInfo{{Pos: "N.g", UPosTag: "NOUN"}},
			Senses:        []Sense{{N: 1, WLSPH: "1.1950", WLSP: "1.1950"}},
			IsCompound:    true,
			Parts: []CompoundPart{
				{Lemma: "一"},
				{Lemma: "年"},
			},
		},
	}

	back := BuildBackDiv(entries, nil, nil)
	entry := back.FindElement("//entry")
	if entry.SelectAttrValue("type", "") != "compound" {
		t.Error("expected type=compound")
	}

	forms := entry.SelectElements("form")
	if len(forms) < 2 {
		t.Fatalf("expected at least 2 <form>, got %d", len(forms))
	}
	compForm := forms[1]
	if compForm.SelectAttrValue("type", "") != "compound" {
		t.Errorf("form type = %q, want compound", compForm.SelectAttrValue("type", ""))
	}
	refs := compForm.SelectElements("ref")
	if len(refs) != 2 {
		t.Fatalf("expected 2 refs, got %d", len(refs))
	}
	if refs[0].SelectAttrValue("target", "") != "#w.一" {
		t.Errorf("ref[0] target = %q", refs[0].SelectAttrValue("target", ""))
	}
}

func TestBuildBackDiv_ModernForm(t *testing.T) {
	entries := []*Entry{
		{
			ID:            "w.留む",
			Lemma:         "留む",
			LemmaReadings: []string{"とどむ"},
			Grams:         []GramInfo{{Pos: "V.free", UPosTag: "VERB"}},
			Senses:        []Sense{{N: 1, WLSPH: "2.1240"}},
			Modern:        &ModernRef{Lemma: "とめる"},
		},
	}

	back := BuildBackDiv(entries, nil, nil)
	entry := back.FindElement("//entry")

	forms := entry.SelectElements("form")
	if len(forms) < 2 {
		t.Fatalf("expected at least 2 <form>, got %d", len(forms))
	}
	modForm := forms[1]
	if modForm.SelectAttrValue("type", "") != "modern" {
		t.Errorf("form type = %q, want modern", modForm.SelectAttrValue("type", ""))
	}
	ref := modForm.SelectElement("ref")
	if ref.SelectAttrValue("target", "") != "#w.とめる" {
		t.Errorf("ref target = %q", ref.SelectAttrValue("target", ""))
	}
}

func TestBuildBackDiv_Hom(t *testing.T) {
	entries := []*Entry{
		{
			ID:            "w.春",
			Lemma:         "春",
			LemmaReadings: []string{"はる"},
			Grams: []GramInfo{
				{Pos: "N.g", UPosTag: "NOUN", IPAPosTag: "名詞-一般"},
				{Pos: "N.Adv", UPosTag: "NOUN", IPAPosTag: "名詞-副詞可能"},
			},
			Senses: []Sense{{N: 1, WLSPH: "1.1624", WLSP: "1.1624"}},
		},
	}

	back := BuildBackDiv(entries, nil, nil)
	entry := back.FindElement("//entry")

	homs := entry.SelectElements("hom")
	if len(homs) != 2 {
		t.Fatalf("expected 2 <hom>, got %d", len(homs))
	}
	if homs[0].SelectAttrValue("xml:id", "") != "w.春.h1" {
		t.Errorf("hom[0] xml:id = %q", homs[0].SelectAttrValue("xml:id", ""))
	}
	if entry.SelectElement("gramGrp") != nil {
		t.Error("entry should not have direct <gramGrp> when using <hom>")
	}
}

func TestBuildBackDiv_MultipleSenses(t *testing.T) {
	entries := []*Entry{
		{
			ID:            "w.下",
			Lemma:         "下",
			LemmaReadings: []string{"した"},
			Grams:         []GramInfo{{Pos: "N.g", UPosTag: "NOUN"}},
			Senses: []Sense{
				{N: 1, WLSPH: "1.1101", WLSP: "1.1101", WLSPDescription: "体-関係-類"},
				{N: 2, WLSPH: "1.1710", WLSP: "1.1710", WLSPDescription: "体-関係-空間的関係-点"},
			},
		},
	}

	back := BuildBackDiv(entries, nil, nil)
	senses := back.FindElements("//sense")
	if len(senses) != 2 {
		t.Fatalf("expected 2 senses, got %d", len(senses))
	}
	if senses[0].SelectAttrValue("xml:id", "") != "w.下.s1" {
		t.Errorf("sense[0] xml:id = %q", senses[0].SelectAttrValue("xml:id", ""))
	}
}

func TestBuildBackDiv_WLSPHOnly(t *testing.T) {
	entries := []*Entry{
		{
			ID:            "w.の",
			Lemma:         "の",
			LemmaReadings: []string{"の"},
			Grams:         []GramInfo{{Pos: "P.c.g", UPosTag: "ADP"}},
			Senses:        []Sense{{N: 1, WLSPH: "8.0061"}},
		},
	}

	back := BuildBackDiv(entries, nil, nil)
	sense := back.FindElement("//sense")
	if sense.SelectAttrValue("ana", "") != "#WLSPH.8.0061" {
		t.Errorf("ana = %q", sense.SelectAttrValue("ana", ""))
	}
}

func TestBuildBackDiv_WithClassification(t *testing.T) {
	entries := []*Entry{
		{
			ID:            "w.年",
			Lemma:         "年",
			LemmaReadings: []string{"とし"},
			Grams:         []GramInfo{{Pos: "N.g"}},
			Senses:        []Sense{{N: 1, WLSPH: "1.1630"}},
		},
	}

	wlsphItems := []ClassItem{{ID: "WLSPH.1.1630", Label: "1.1630", Desc: "体-関係-時間-年"}}
	wlspItems := []ClassItem{{ID: "WLSP.1.1630", Label: "1.1630", Desc: "体-関係-時間-年"}}

	classWLSPH := BuildClassificationDiv("classWLSPH", "分類語彙表 (WLSPH)", wlsphItems)
	classWLSP := BuildClassificationDiv("classWLSP", "分類語彙表 (WLSP)", wlspItems)

	back := BuildBackDiv(entries, classWLSPH, classWLSP)

	divs := back.SelectElements("div")
	if len(divs) != 3 {
		t.Fatalf("expected 3 divs in <back>, got %d", len(divs))
	}
	if divs[0].SelectAttrValue("type", "") != "dictionary" {
		t.Error("first div should be dictionary")
	}
	if divs[1].SelectAttrValue("xml:id", "") != "classWLSPH" {
		t.Error("second div should be classWLSPH")
	}
	if divs[2].SelectAttrValue("xml:id", "") != "classWLSP" {
		t.Error("third div should be classWLSP")
	}
}

func TestTransformBody(t *testing.T) {
	doc := parseTestDoc(t)
	tokens, _, _ := ExtractTokens(doc)
	entries := BuildEntries(tokens)

	TransformBody(doc, entries)

	for _, w := range doc.FindElements("//w") {
		for _, attr := range []string{"pos", "lemma", "msd"} {
			if w.SelectAttrValue(attr, "MISSING") != "MISSING" {
				t.Errorf("w still has %s", attr)
			}
		}
		lemmaRef := w.SelectAttrValue("lemmaRef", "")
		if lemmaRef == "" {
			t.Errorf("w missing lemmaRef (text=%q)", w.Text())
		}
		if !strings.HasPrefix(lemmaRef, "#w.") {
			t.Errorf("lemmaRef should start with #w., got %q", lemmaRef)
		}
	}
}

func TestUpdateHeader(t *testing.T) {
	doc := parseTestDoc(t)
	UpdateHeader(doc)

	found := false
	for _, tax := range doc.FindElements("//taxonomy") {
		if tax.SelectAttrValue("xml:id", "") == "WLSP" {
			found = true
			break
		}
	}
	if !found {
		t.Error("WLSP taxonomy not found in header")
	}

	interp := doc.FindElement("//interpretation")
	if interp == nil {
		t.Fatal("no <interpretation> in header")
	}
	p := interp.SelectElement("p")
	if p == nil {
		t.Fatal("no <p> in interpretation")
	}
	if len(p.SelectElements("gi")) < 2 {
		t.Error("expected at least 2 <gi> elements")
	}
	if len(p.SelectElements("att")) < 1 {
		t.Error("expected at least 1 <att> element")
	}
}

func TestUpdateHeader_Idempotent(t *testing.T) {
	doc := parseTestDoc(t)
	UpdateHeader(doc)
	UpdateHeader(doc)

	count := 0
	for _, tax := range doc.FindElements("//taxonomy") {
		if tax.SelectAttrValue("xml:id", "") == "WLSP" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 WLSP taxonomy, got %d", count)
	}
}

func TestInsertBack(t *testing.T) {
	doc := parseTestDoc(t)
	entries := []*Entry{
		{
			ID:            "w.年",
			Lemma:         "年",
			LemmaReadings: []string{"とし"},
			Grams:         []GramInfo{{Pos: "N.g"}},
			Senses:        []Sense{{N: 1, WLSPH: "1.1630"}},
		},
	}
	back := BuildBackDiv(entries, nil, nil)
	InsertBack(doc, back)

	text := doc.FindElement("//TEI/text")
	if text == nil {
		t.Fatal("no <text> element")
	}
	if text.SelectElement("back") == nil {
		t.Fatal("no <back> element after InsertBack")
	}
}

func TestRoundTrip(t *testing.T) {
	doc := parseTestDoc(t)

	tokens, compounds, modernRefs := ExtractTokens(doc)
	entries := BuildEntries(tokens)
	MarkCompounds(entries, compounds, modernRefs)

	flatCount := FlattenApps(doc)
	if flatCount == 0 {
		t.Error("expected at least 1 app flattened")
	}

	TransformBody(doc, entries)
	back := BuildBackDiv(entries, nil, nil)
	InsertBack(doc, back)
	UpdateHeader(doc)

	doc.Indent(2)
	xmlStr, err := doc.WriteToString()
	if err != nil {
		t.Fatalf("failed to serialize: %v", err)
	}

	doc2 := etree.NewDocument()
	if err := doc2.ReadFromString(xmlStr); err != nil {
		t.Fatalf("failed to re-parse: %v", err)
	}

	// No pos/lemma/msd on <w>.
	for _, w := range doc2.FindElements("//w") {
		for _, attr := range []string{"pos", "lemma", "msd"} {
			if w.SelectAttrValue(attr, "MISSING") != "MISSING" {
				t.Errorf("%s still present", attr)
			}
		}
	}

	// No <app>.
	if len(doc2.FindElements("//app")) > 0 {
		t.Error("<app> still present")
	}

	// All lemmaRef resolve.
	idSet := make(map[string]bool)
	for _, el := range doc2.FindElements("//entry") {
		if id := el.SelectAttrValue("xml:id", ""); id != "" {
			idSet[id] = true
		}
	}
	for _, el := range doc2.FindElements("//hom") {
		if id := el.SelectAttrValue("xml:id", ""); id != "" {
			idSet[id] = true
		}
	}
	for _, w := range doc2.FindElements("//w") {
		ref := w.SelectAttrValue("lemmaRef", "")
		if ref == "" {
			continue
		}
		target := strings.TrimPrefix(ref, "#")
		if !idSet[target] {
			t.Errorf("lemmaRef %q unresolved", ref)
		}
	}

	if doc2.FindElement("//back") == nil {
		t.Error("no <back>")
	}

	for _, entry := range doc2.FindElements("//entry") {
		typ := entry.SelectAttrValue("type", "")
		if typ != "simplex" && typ != "compound" {
			t.Errorf("entry %q type=%q", entry.SelectAttrValue("xml:id", ""), typ)
		}
	}
}

func TestBuildAnaRef(t *testing.T) {
	tests := []struct {
		name  string
		sense Sense
		want  string
	}{
		{"both", Sense{WLSPH: "1.1630", WLSP: "1.1630"}, "#WLSPH.1.1630 #WLSP.1.1630"},
		{"WLSPH only", Sense{WLSPH: "8.0061"}, "#WLSPH.8.0061"},
		{"different codes", Sense{WLSPH: "1.1742", WLSP: "1.5240"}, "#WLSPH.1.1742 #WLSP.1.5240"},
		{"neither", Sense{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildAnaRef(tt.sense)
			if got != tt.want {
				t.Errorf("buildAnaRef() = %q, want %q", got, tt.want)
			}
		})
	}
}
