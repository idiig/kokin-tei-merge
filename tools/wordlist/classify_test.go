package wordlist

import (
	"os"
	"testing"
)

func TestWLSPHCategoryDescription(t *testing.T) {
	tests := []struct {
		name string
		cat  WLSPHCategory
		want string
	}{
		{
			name: "full hierarchy",
			cat: WLSPHCategory{
				Code: "1.1000",
				Pos:  CategoryField{Category: "体"},
				Group: CategoryField{Category: "抽象的関係"},
				Field: CategoryField{Category: "事柄"},
				Near: CategoryField{Category: "こそあど"},
				Syn:  CategoryField{Category: "こそあど"},
			},
			want: "体-抽象的関係-事柄-こそあど",
		},
		{
			name: "syn differs from near",
			cat: WLSPHCategory{
				Code: "1.1010",
				Pos:  CategoryField{Category: "体"},
				Group: CategoryField{Category: "抽象的関係"},
				Field: CategoryField{Category: "事柄"},
				Near: CategoryField{Category: "事柄"},
				Syn:  CategoryField{Category: "事柄"},
			},
			want: "体-抽象的関係-事柄-事柄",
		},
		{
			name: "empty lower levels",
			cat: WLSPHCategory{
				Code: "CH.11.0000",
				Pos:  CategoryField{Category: "地名-埼玉県"},
				Group: CategoryField{Category: ""},
				Field: CategoryField{Category: ""},
				Near: CategoryField{Category: ""},
				Syn:  CategoryField{Category: ""},
			},
			want: "地名-埼玉県",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cat.Description()
			if got != tt.want {
				t.Errorf("Description() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractWLSPCodes(t *testing.T) {
	tokens := []Token{
		{MSD: MSD{WLSP: "1.1630", WLSPDescription: "体-関係-時間-年"}},
		{MSD: MSD{WLSP: "1.1624", WLSPDescription: "体-関係-時間-季節"}},
		{MSD: MSD{WLSP: "1.1630", WLSPDescription: "体-関係-時間-年"}}, // dup
		{MSD: MSD{WLSPH: "8.0061"}},                                   // no WLSP
	}

	codes := ExtractWLSPCodes(tokens)

	if len(codes) != 2 {
		t.Fatalf("expected 2 codes, got %d", len(codes))
	}
	// Should be sorted.
	if codes[0].Code != "1.1624" {
		t.Errorf("codes[0].Code = %q, want %q", codes[0].Code, "1.1624")
	}
	if codes[1].Code != "1.1630" {
		t.Errorf("codes[1].Code = %q, want %q", codes[1].Code, "1.1630")
	}
	if codes[1].Description != "体-関係-時間-年" {
		t.Errorf("codes[1].Description = %q", codes[1].Description)
	}
}

func TestBuildWLSPHItems(t *testing.T) {
	cats := []WLSPHCategory{
		{
			Code: "1.1000",
			Pos:  CategoryField{Category: "体"},
			Group: CategoryField{Category: "抽象的関係"},
			Field: CategoryField{Category: "事柄"},
			Near: CategoryField{Category: "こそあど"},
			Syn:  CategoryField{Category: "こそあど"},
		},
	}

	items := BuildWLSPHItems(cats)

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].ID != "WLSPH.1.1000" {
		t.Errorf("ID = %q, want %q", items[0].ID, "WLSPH.1.1000")
	}
	if items[0].Label != "1.1000" {
		t.Errorf("Label = %q, want %q", items[0].Label, "1.1000")
	}
}

func TestBuildWLSPItems(t *testing.T) {
	codes := []WLSPCode{
		{Code: "1.1630", Description: "体-関係-時間-年"},
	}

	items := BuildWLSPItems(codes)

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].ID != "WLSP.1.1630" {
		t.Errorf("ID = %q, want %q", items[0].ID, "WLSP.1.1630")
	}
}

func TestBuildClassificationDiv(t *testing.T) {
	items := []ClassItem{
		{ID: "WLSPH.1.1000", Label: "1.1000", Desc: "体-抽象的関係-事柄-こそあど"},
		{ID: "WLSPH.1.1010", Label: "1.1010", Desc: "体-抽象的関係-事柄-事柄"},
	}

	div := BuildClassificationDiv("classWLSPH", "分類語彙表 (WLSPH)", items)

	if div.SelectAttrValue("xml:id", "") != "classWLSPH" {
		t.Errorf("xml:id = %q", div.SelectAttrValue("xml:id", ""))
	}
	head := div.SelectElement("head")
	if head == nil || head.Text() != "分類語彙表 (WLSPH)" {
		t.Errorf("head = %v", head)
	}
	list := div.SelectElement("list")
	if list == nil {
		t.Fatal("no <list>")
	}
	listItems := list.SelectElements("item")
	if len(listItems) != 2 {
		t.Fatalf("expected 2 items, got %d", len(listItems))
	}
	if listItems[0].SelectAttrValue("xml:id", "") != "WLSPH.1.1000" {
		t.Errorf("item[0] xml:id = %q", listItems[0].SelectAttrValue("xml:id", ""))
	}
}

func TestLoadWLSPH(t *testing.T) {
	// Create a temporary JSON file.
	content := `{"categories": [
		{"code": "1.1000", "source": "wlsp",
		 "pos": {"code": "1", "category": "体"},
		 "group": {"code": "1", "category": "抽象的関係"},
		 "field": {"code": "0", "category": "事柄"},
		 "near_synonymy": {"code": "0", "category": "こそあど"},
		 "synonymy": {"code": "0", "category": "こそあど"}}
	]}`

	f, err := os.CreateTemp("", "wlsph-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()

	cats, err := LoadWLSPH(f.Name())
	if err != nil {
		t.Fatalf("LoadWLSPH: %v", err)
	}
	if len(cats) != 1 {
		t.Fatalf("expected 1 category, got %d", len(cats))
	}
	if cats[0].Code != "1.1000" {
		t.Errorf("code = %q", cats[0].Code)
	}
	if cats[0].Description() != "体-抽象的関係-事柄-こそあど" {
		t.Errorf("description = %q", cats[0].Description())
	}
}
