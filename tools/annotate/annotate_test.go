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

// --- ApplyAlignment ---

func TestApplyAlignment_PlainText(t *testing.T) {
	doc := etree.NewDocument()
	root := doc.CreateElement("l")
	seg := root.CreateElement("seg")
	seg.SetText("春は")

	aligned := [][]Token{
		{{Surface: "春", LemmaRef: "#w.春.h1"}, {Surface: "は", LemmaRef: "#w.は"}},
	}
	ApplyAlignment([]*etree.Element{seg}, aligned)

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
	// tokens: 花, とや, 見らむ
	doc := etree.NewDocument()
	root := doc.CreateElement("l")
	seg := root.CreateElement("seg")
	seg.CreateCharData("花とや")
	app := seg.CreateElement("app")
	lem := app.CreateElement("lem")
	lem.SetText("見らむ")
	rdg := app.CreateElement("rdg")
	rdg.SetText("みえん")

	aligned := [][]Token{{
		{Surface: "花", LemmaRef: "#w.花"},
		{Surface: "とや", LemmaRef: "#w.とや"},
		{Surface: "見らむ", LemmaRef: "#w.見る"},
	}}
	ApplyAlignment([]*etree.Element{seg}, aligned)

	// <app> must still be present.
	apps := seg.SelectElements("app")
	if len(apps) != 1 {
		t.Fatalf("got %d <app> elements, want 1", len(apps))
	}

	// <rdg> inside <app> must be unchanged.
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

func TestApplyAlignment_EmptySegment(t *testing.T) {
	doc := etree.NewDocument()
	root := doc.CreateElement("l")
	seg := root.CreateElement("seg")
	seg.SetText("text")

	ApplyAlignment([]*etree.Element{seg}, [][]Token{{}})

	if len(seg.SelectElements("w")) != 0 {
		t.Error("expected no <w> elements for empty token slice")
	}
}
