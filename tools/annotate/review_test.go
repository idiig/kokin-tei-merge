package annotate

import (
	"strings"
	"testing"
)

// --- ParseDraft ---

func TestParseDraft_ValidTwoGroups(t *testing.T) {
	content := `# Poem 1
# Columns: surface (edit this) | lemmaRef (do not edit)
春	#w.春.h1	# ✓
は	#w.は	# ✓

けり	#w.けり	# ✓
`
	segTexts := []string{"春は", "けり"}
	got, err := ParseDraft(content, segTexts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected result, got nil")
	}
	if len(got) != 2 {
		t.Fatalf("got %d groups, want 2", len(got))
	}
	if got[0][0].Surface != "春" || got[0][1].Surface != "は" {
		t.Errorf("seg 0 wrong: %v", got[0])
	}
	if got[1][0].Surface != "けり" {
		t.Errorf("seg 1 wrong: %v", got[1])
	}
}

func TestParseDraft_SkipWhenNoDataLines(t *testing.T) {
	content := `# Poem 1
# instructions only, no data
`
	got, err := ParseDraft(content, []string{"春は"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil (skip), got %v", got)
	}
}

func TestParseDraft_WrongGroupCount(t *testing.T) {
	content := `春	#w.春.h1
は	#w.は
`
	_, err := ParseDraft(content, []string{"春は", "けり"})
	if err == nil {
		t.Fatal("expected error for wrong group count")
	}
	if !strings.Contains(err.Error(), "want 2") {
		t.Errorf("error message unexpected: %v", err)
	}
}

func TestParseDraft_SurfaceMismatch(t *testing.T) {
	content := `春	#w.春.h1
`
	_, err := ParseDraft(content, []string{"夏"})
	if err == nil {
		t.Fatal("expected mismatch error")
	}
	if !strings.Contains(err.Error(), "segment 1") {
		t.Errorf("error message unexpected: %v", err)
	}
}

func TestParseDraft_ThreeColumnFormat(t *testing.T) {
	// Third column (hint comment) must be silently ignored.
	content := "春\t#w.春.h1\t# ? check surface\n"
	got, err := ParseDraft(content, []string{"春"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got[0][0].LemmaRef != "#w.春.h1" {
		t.Errorf("lemmaRef wrong: %q", got[0][0].LemmaRef)
	}
}

func TestParseDraft_InvalidLine(t *testing.T) {
	content := "no-tab-here\n"
	_, err := ParseDraft(content, []string{"no-tab-here"})
	if err == nil {
		t.Fatal("expected error for invalid line")
	}
}

func TestParseDraft_EmptySurface(t *testing.T) {
	content := "\t#w.春.h1\n"
	_, err := ParseDraft(content, []string{""})
	if err == nil {
		t.Fatal("expected error for empty surface")
	}
}

func TestParseDraft_TrailingBlankLinesIgnored(t *testing.T) {
	// Trailing blank lines after the last token group must not create phantom groups.
	content := "春\t#w.春.h1\nは\t#w.は\n\n\n"
	got, err := ParseDraft(content, []string{"春は"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("got %d groups, want 1", len(got))
	}
}

// --- GenerateDraft ---

func TestGenerateDraft_ContainsHeader(t *testing.T) {
	tokens := []Token{{Surface: "春", LemmaRef: "#w.春.h1"}}
	out := GenerateDraft(1, tokens, []string{"春"}, nil)
	if !strings.Contains(out, "Poem 1") {
		t.Error("missing poem number")
	}
	if !strings.Contains(out, "surface (edit this)") {
		t.Error("missing column hint")
	}
}

func TestGenerateDraft_FlatWithoutSplits(t *testing.T) {
	tokens := []Token{
		{Surface: "春", LemmaRef: "#w.春.h1"},
		{Surface: "は", LemmaRef: "#w.は"},
	}
	out := GenerateDraft(1, tokens, []string{"春は"}, nil)
	if !strings.Contains(out, "春\t#w.春.h1") {
		t.Error("missing token line")
	}
	// No seg header in flat mode.
	if strings.Contains(out, "seg 1") {
		t.Error("unexpected seg header in flat mode")
	}
}

func TestGenerateDraft_WithSplitsShowsSegHeaders(t *testing.T) {
	tokens := []Token{
		{Surface: "春", LemmaRef: "#w.春.h1"},
		{Surface: "は", LemmaRef: "#w.は"},
		{Surface: "けり", LemmaRef: "#w.けり"},
	}
	segTexts := []string{"春は", "けり"}
	splits := []int{2, 1}
	out := GenerateDraft(1, tokens, segTexts, splits)
	if !strings.Contains(out, "seg 1") || !strings.Contains(out, "seg 2") {
		t.Error("missing seg headers")
	}
}

func TestGenerateDraft_MatchedTokenMarkedOK(t *testing.T) {
	tokens := []Token{{Surface: "春", LemmaRef: "#w.春.h1"}}
	out := GenerateDraft(1, tokens, []string{"春"}, []int{1})
	if !strings.Contains(out, "✓") {
		t.Error("matched token should be marked ✓")
	}
}

func TestGenerateDraft_MismatchedTokenMarkedCheck(t *testing.T) {
	tokens := []Token{{Surface: "一とせ", LemmaRef: "#w.一年"}}
	out := GenerateDraft(1, tokens, []string{"ひとゝせ"}, []int{1})
	if !strings.Contains(out, "? check surface") {
		t.Error("mismatched token should be marked '? check surface'")
	}
	if !strings.Contains(out, "✗") {
		t.Error("seg header should show ✗ mismatch")
	}
}

func TestGenerateDraft_RoundTrip(t *testing.T) {
	// GenerateDraft output must be parseable by ParseDraft when surfaces match.
	tokens := []Token{
		{Surface: "春", LemmaRef: "#w.春.h1"},
		{Surface: "は", LemmaRef: "#w.は"},
		{Surface: "けり", LemmaRef: "#w.けり"},
	}
	segTexts := []string{"春は", "けり"}
	splits := []int{2, 1}
	draft := GenerateDraft(1, tokens, segTexts, splits)
	got, err := ParseDraft(draft, segTexts)
	if err != nil {
		t.Fatalf("ParseDraft failed on GenerateDraft output: %v", err)
	}
	if len(got) != 2 || len(got[0]) != 2 || len(got[1]) != 1 {
		t.Errorf("unexpected parse result: %v", got)
	}
}
