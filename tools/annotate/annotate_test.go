package annotate

import (
	"testing"

	"github.com/beevik/etree"
)

// --- ApplyAlignment ---

func TestApplyAlignment_Basic(t *testing.T) {
	doc := etree.NewDocument()
	root := doc.CreateElement("l")
	seg1 := root.CreateElement("seg")
	seg1.SetText("春は")
	seg2 := root.CreateElement("seg")
	seg2.SetText("けり")

	aligned := [][]Token{
		{{Surface: "春", LemmaRef: "#w.春.h1"}, {Surface: "は", LemmaRef: "#w.は"}},
		{{Surface: "けり", LemmaRef: "#w.けり"}},
	}
	ApplyAlignment([]*etree.Element{seg1, seg2}, aligned)

	ws1 := seg1.SelectElements("w")
	if len(ws1) != 2 {
		t.Fatalf("seg1: got %d <w> elements, want 2", len(ws1))
	}
	if ws1[0].Text() != "春" || ws1[0].SelectAttrValue("lemmaRef", "") != "#w.春.h1" {
		t.Errorf("seg1[0] wrong: text=%q lemmaRef=%q", ws1[0].Text(), ws1[0].SelectAttrValue("lemmaRef", ""))
	}

	ws2 := seg2.SelectElements("w")
	if len(ws2) != 1 {
		t.Fatalf("seg2: got %d <w> elements, want 1", len(ws2))
	}
	if ws2[0].Text() != "けり" {
		t.Errorf("seg2[0] text wrong: %q", ws2[0].Text())
	}
}

func TestApplyAlignment_ClearsExistingContent(t *testing.T) {
	doc := etree.NewDocument()
	root := doc.CreateElement("l")
	seg := root.CreateElement("seg")
	// Pre-existing <w> element.
	old := seg.CreateElement("w")
	old.SetText("old")

	aligned := [][]Token{{{Surface: "新", LemmaRef: "#w.新"}}}
	ApplyAlignment([]*etree.Element{seg}, aligned)

	ws := seg.SelectElements("w")
	if len(ws) != 1 {
		t.Fatalf("got %d <w> elements, want 1", len(ws))
	}
	if ws[0].Text() != "新" {
		t.Errorf("expected '新', got %q", ws[0].Text())
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
