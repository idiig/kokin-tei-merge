package annotate

import (
	"testing"

	"github.com/beevik/etree"
)

// --- SegText ---

func TestSegText_PlainText(t *testing.T) {
	doc := etree.NewDocument()
	seg := doc.CreateElement("seg")
	seg.SetText("年の内に")
	if got := SegText(seg); got != "年の内に" {
		t.Errorf("got %q, want %q", got, "年の内に")
	}
}

func TestSegText_WithApp(t *testing.T) {
	// <seg>花とや<app><lem wit="#国">見らむ</lem><rdg wit="#前">みえん</rdg></app></seg>
	doc := etree.NewDocument()
	seg := doc.CreateElement("seg")
	seg.CreateCharData("花とや")
	app := seg.CreateElement("app")
	lem := app.CreateElement("lem")
	lem.CreateAttr("wit", "#国")
	lem.SetText("見らむ")
	rdg := app.CreateElement("rdg")
	rdg.CreateAttr("wit", "#前")
	rdg.SetText("みえん")

	if got := SegText(seg); got != "花とや見らむ" {
		t.Errorf("got %q, want %q", got, "花とや見らむ")
	}
}

func TestSegText_SkipsRdg(t *testing.T) {
	doc := etree.NewDocument()
	seg := doc.CreateElement("seg")
	app := seg.CreateElement("app")
	lem := app.CreateElement("lem")
	lem.SetText("正")
	rdg := app.CreateElement("rdg")
	rdg.SetText("副")

	if got := SegText(seg); got != "正" {
		t.Errorf("got %q, want %q (rdg should be excluded)", got, "正")
	}
}

// --- ExtractSegMetas ---

func TestExtractSegMetas_NoApp(t *testing.T) {
	doc := etree.NewDocument()
	seg := doc.CreateElement("seg")
	seg.SetText("春は")
	metas := ExtractSegMetas([]*etree.Element{seg})
	if len(metas) != 1 {
		t.Fatalf("want 1 meta, got %d", len(metas))
	}
	if metas[0].Text != "春は" {
		t.Errorf("Text = %q, want %q", metas[0].Text, "春は")
	}
	if metas[0].LemText != "" || metas[0].RdgText != "" {
		t.Errorf("expected empty LemText/RdgText, got %q / %q", metas[0].LemText, metas[0].RdgText)
	}
}

func TestExtractSegMetas_WithApp(t *testing.T) {
	// <seg>花とや<app><lem wit="#国">見らむ</lem><rdg wit="#前">みえん</rdg></app></seg>
	doc := etree.NewDocument()
	seg := doc.CreateElement("seg")
	seg.CreateCharData("花とや")
	app := seg.CreateElement("app")
	lem := app.CreateElement("lem")
	lem.CreateAttr("wit", "#国")
	lem.SetText("見らむ")
	rdg := app.CreateElement("rdg")
	rdg.CreateAttr("wit", "#前")
	rdg.SetText("みえん")

	metas := ExtractSegMetas([]*etree.Element{seg})
	if len(metas) != 1 {
		t.Fatalf("want 1 meta, got %d", len(metas))
	}
	if metas[0].Text != "花とや見らむ" {
		t.Errorf("Text = %q, want %q", metas[0].Text, "花とや見らむ")
	}
	if metas[0].LemText != "見らむ" {
		t.Errorf("LemText = %q, want %q", metas[0].LemText, "見らむ")
	}
	if metas[0].RdgText != "みえん" {
		t.Errorf("RdgText = %q, want %q", metas[0].RdgText, "みえん")
	}
}

func TestExtractSegMetas_MultipleSegs(t *testing.T) {
	doc := etree.NewDocument()
	// Seg 0: plain text, no app.
	seg0 := doc.CreateElement("seg")
	seg0.SetText("春は")
	// Seg 1: has app.
	seg1 := doc.CreateElement("seg")
	app := seg1.CreateElement("app")
	lem := app.CreateElement("lem")
	lem.SetText("正")
	rdg := app.CreateElement("rdg")
	rdg.SetText("副")

	metas := ExtractSegMetas([]*etree.Element{seg0, seg1})
	if len(metas) != 2 {
		t.Fatalf("want 2 metas, got %d", len(metas))
	}
	if metas[0].RdgText != "" {
		t.Errorf("seg 0 should have empty RdgText, got %q", metas[0].RdgText)
	}
	if metas[1].LemText != "正" || metas[1].RdgText != "副" {
		t.Errorf("seg 1: LemText=%q RdgText=%q, want 正/副", metas[1].LemText, metas[1].RdgText)
	}
}

// --- ApplyAlignment ---

func TestApplyAlignment_PlainText(t *testing.T) {
	doc := etree.NewDocument()
	root := doc.CreateElement("l")
	seg := root.CreateElement("seg")
	seg.SetText("春は")

	groups := []SegGroup{{Lem: []Token{
		{Surface: "春", LemmaRef: "#w.春.h1"},
		{Surface: "は", LemmaRef: "#w.は"},
	}}}
	ApplyAlignment([]*etree.Element{seg}, groups)

	ws := seg.SelectElements("w")
	if len(ws) != 2 {
		t.Fatalf("got %d <w> elements, want 2", len(ws))
	}
	if ws[0].Text() != "春" || ws[1].Text() != "は" {
		t.Errorf("unexpected token text: %q %q", ws[0].Text(), ws[1].Text())
	}
}

func TestApplyAlignment_PreservesApp(t *testing.T) {
	// <seg>花とや<app><lem>見らむ</lem><rdg>みえん</rdg></app></seg>
	// tokens: 花, とや, 見らむ  — Rdg nil → <rdg> left unchanged
	doc := etree.NewDocument()
	root := doc.CreateElement("l")
	seg := root.CreateElement("seg")
	seg.CreateCharData("花とや")
	app := seg.CreateElement("app")
	lem := app.CreateElement("lem")
	lem.SetText("見らむ")
	rdg := app.CreateElement("rdg")
	rdg.SetText("みえん")

	groups := []SegGroup{{Lem: []Token{
		{Surface: "花", LemmaRef: "#w.花"},
		{Surface: "とや", LemmaRef: "#w.とや"},
		{Surface: "見らむ", LemmaRef: "#w.見る"},
	}}}
	ApplyAlignment([]*etree.Element{seg}, groups)

	// <app> must still be present.
	apps := seg.SelectElements("app")
	if len(apps) != 1 {
		t.Fatalf("got %d <app> elements, want 1", len(apps))
	}

	// <rdg> inside <app> must be unchanged (Rdg is nil).
	rdgEl := apps[0].SelectElement("rdg")
	if rdgEl == nil || rdgEl.Text() != "みえん" {
		t.Errorf("<rdg> missing or changed")
	}

	// <lem> must contain <w> elements.
	lemEl := apps[0].SelectElement("lem")
	ws := lemEl.SelectElements("w")
	if len(ws) != 1 || ws[0].Text() != "見らむ" {
		t.Errorf("<lem> should contain <w>見らむ</w>, got %v", ws)
	}

	// Direct <w> children of <seg> before <app>.
	directWs := seg.SelectElements("w")
	if len(directWs) != 2 {
		t.Fatalf("got %d direct <w> before <app>, want 2", len(directWs))
	}
	if directWs[0].Text() != "花" || directWs[1].Text() != "とや" {
		t.Errorf("unexpected direct tokens: %q %q", directWs[0].Text(), directWs[1].Text())
	}
}

func TestApplyAlignment_RdgAnnotated(t *testing.T) {
	// <seg>花とや<app><lem>見らむ</lem><rdg>みえん</rdg></app></seg>
	// Rdg non-nil → <rdg> gets <w> children.
	doc := etree.NewDocument()
	root := doc.CreateElement("l")
	seg := root.CreateElement("seg")
	seg.CreateCharData("花とや")
	app := seg.CreateElement("app")
	lem := app.CreateElement("lem")
	lem.SetText("見らむ")
	rdg := app.CreateElement("rdg")
	rdg.SetText("みえん")

	groups := []SegGroup{{
		Lem: []Token{
			{Surface: "花とや", LemmaRef: "#w.花"},
			{Surface: "見らむ", LemmaRef: "#w.見る"},
		},
		Rdg: []Token{
			{Surface: "みえ", LemmaRef: "#w.見ゆ"},
			{Surface: "ん", LemmaRef: "#w.む"},
		},
	}}
	ApplyAlignment([]*etree.Element{seg}, groups)

	apps := seg.SelectElements("app")
	if len(apps) != 1 {
		t.Fatalf("got %d <app> elements, want 1", len(apps))
	}
	// <rdg> must now contain <w> elements.
	rdgEl := apps[0].SelectElement("rdg")
	ws := rdgEl.SelectElements("w")
	if len(ws) != 2 {
		t.Fatalf("<rdg>: got %d <w> elements, want 2", len(ws))
	}
	if ws[0].Text() != "みえ" || ws[0].SelectAttrValue("lemmaRef", "") != "#w.見ゆ" {
		t.Errorf("rdg[0]: got %q %q", ws[0].Text(), ws[0].SelectAttrValue("lemmaRef", ""))
	}
	if ws[1].Text() != "ん" || ws[1].SelectAttrValue("lemmaRef", "") != "#w.む" {
		t.Errorf("rdg[1]: got %q %q", ws[1].Text(), ws[1].SelectAttrValue("lemmaRef", ""))
	}
}

func TestApplyAlignment_RdgNilLeavesRdgUntouched(t *testing.T) {
	doc := etree.NewDocument()
	root := doc.CreateElement("l")
	seg := root.CreateElement("seg")
	app := seg.CreateElement("app")
	lem := app.CreateElement("lem")
	lem.SetText("見らむ")
	rdg := app.CreateElement("rdg")
	rdg.SetText("みえん")

	groups := []SegGroup{{
		Lem: []Token{{Surface: "見らむ", LemmaRef: "#w.見る"}},
		Rdg: nil,
	}}
	ApplyAlignment([]*etree.Element{seg}, groups)

	rdgEl := seg.FindElement("//rdg")
	if rdgEl == nil {
		t.Fatal("<rdg> missing")
	}
	if rdgEl.Text() != "みえん" {
		t.Errorf("<rdg> text changed: %q", rdgEl.Text())
	}
	if len(rdgEl.SelectElements("w")) != 0 {
		t.Error("<rdg> should have no <w> children when Rdg is nil")
	}
}

func TestApplyAlignment_EmptySegment(t *testing.T) {
	doc := etree.NewDocument()
	root := doc.CreateElement("l")
	seg := root.CreateElement("seg")
	seg.SetText("text")

	ApplyAlignment([]*etree.Element{seg}, []SegGroup{{Lem: nil}})

	if len(seg.SelectElements("w")) != 0 {
		t.Error("expected no <w> elements for empty token slice")
	}
}
