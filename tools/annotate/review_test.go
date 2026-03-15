package annotate

import (
	"strings"
	"testing"
)

// --- splitSegByRunes (DP) ---

func TestSplitSegByRunes_TwoTokensExactReading(t *testing.T) {
	// "おもひそめ" split between Reading "おもひ" and "そめ" → [おもひ, そめ]
	tokens := []Token{
		{Surface: "???", Reading: "おもひ", LemmaRef: "#w.思ふ"},
		{Surface: "???", Reading: "そめ", LemmaRef: "#w.染む"},
	}
	got := splitSegByRunes("おもひそめ", tokens)
	if len(got) != 2 {
		t.Fatalf("want 2 parts, got %d", len(got))
	}
	if got[0] != "おもひ" || got[1] != "そめ" {
		t.Errorf("got %v, want [おもひ そめ]", got)
	}
}

func TestSplitSegByRunes_SingleToken(t *testing.T) {
	tokens := []Token{{Surface: "春", LemmaRef: "#w.春.h1"}}
	got := splitSegByRunes("はる", tokens)
	if len(got) != 1 || got[0] != "はる" {
		t.Errorf("single token should get full text, got %v", got)
	}
}

func TestSplitSegByRunes_ExactSurfaceBonus(t *testing.T) {
	// Token surface "は" exactly matches → should be assigned "は".
	tokens := []Token{
		{Surface: "春", LemmaRef: "#w.春.h1"},
		{Surface: "は", LemmaRef: "#w.は"},
	}
	got := splitSegByRunes("春は", tokens)
	if got[0] != "春" || got[1] != "は" {
		t.Errorf("got %v, want [春 は]", got)
	}
}

// --- ParseDraft ---

func TestParseDraft_ValidTwoGroups(t *testing.T) {
	content := `# Poem 1
# Columns: surface (edit this) | lemmaRef (do not edit)
春	#w.春.h1	# ✓
は	#w.は	# ✓

けり	#w.けり	# ✓
`
	segTexts := []string{"春は", "けり"}
	got, err := ParseDraft(content, segTexts, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected result, got nil")
	}
	if len(got) != 2 {
		t.Fatalf("got %d groups, want 2", len(got))
	}
	if got[0].Lem[0].Surface != "春" || got[0].Lem[1].Surface != "は" {
		t.Errorf("seg 0 wrong: %v", got[0].Lem)
	}
	if got[1].Lem[0].Surface != "けり" {
		t.Errorf("seg 1 wrong: %v", got[1].Lem)
	}
}

func TestParseDraft_SkipWhenNoDataLines(t *testing.T) {
	content := `# Poem 1
# instructions only, no data
`
	got, err := ParseDraft(content, []string{"春は"}, nil)
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
	_, err := ParseDraft(content, []string{"春は", "けり"}, nil)
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
	_, err := ParseDraft(content, []string{"夏"}, nil)
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
	got, err := ParseDraft(content, []string{"春"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got[0].Lem[0].LemmaRef != "#w.春.h1" {
		t.Errorf("lemmaRef wrong: %q", got[0].Lem[0].LemmaRef)
	}
}

func TestParseDraft_InvalidLine(t *testing.T) {
	content := "no-tab-here\n"
	_, err := ParseDraft(content, []string{"no-tab-here"}, nil)
	if err == nil {
		t.Fatal("expected error for invalid line")
	}
}

func TestParseDraft_EmptySurface(t *testing.T) {
	content := "\t#w.春.h1\n"
	_, err := ParseDraft(content, []string{""}, nil)
	if err == nil {
		t.Fatal("expected error for empty surface")
	}
}

func TestParseDraft_TrailingBlankLinesIgnored(t *testing.T) {
	// Trailing blank lines after the last token group must not create phantom groups.
	content := "春\t#w.春.h1\nは\t#w.は\n\n\n"
	got, err := ParseDraft(content, []string{"春は"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("got %d groups, want 1", len(got))
	}
}

func TestParseDraft_RdgRowsValid(t *testing.T) {
	content := "見らむ\t#w.見る\nrdg\tみえ\t#w.見ゆ\nrdg\tん\t#w.む\n"
	got, err := ParseDraft(content, []string{"見らむ"}, []string{"みえん"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got[0].Rdg) != 2 {
		t.Fatalf("want 2 rdg tokens, got %d", len(got[0].Rdg))
	}
	if got[0].Rdg[0].Surface != "みえ" || got[0].Rdg[0].LemmaRef != "#w.見ゆ" {
		t.Errorf("rdg[0] wrong: %+v", got[0].Rdg[0])
	}
	if got[0].Rdg[1].Surface != "ん" || got[0].Rdg[1].LemmaRef != "#w.む" {
		t.Errorf("rdg[1] wrong: %+v", got[0].Rdg[1])
	}
}

func TestParseDraft_RdgRowsDeletedLeaveNil(t *testing.T) {
	content := "見らむ\t#w.見る\n"
	got, err := ParseDraft(content, []string{"見らむ"}, []string{"みえん"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got[0].Rdg != nil {
		t.Errorf("want nil Rdg when no rdg rows, got %v", got[0].Rdg)
	}
}

func TestParseDraft_RdgRowsNotAllowedWhenNoRdgText(t *testing.T) {
	content := "春\t#w.春.h1\nrdg\tみ\t#w.見ゆ\n"
	_, err := ParseDraft(content, []string{"春"}, []string{""})
	if err == nil {
		t.Fatal("expected error: rdg rows without rdg text")
	}
	if !strings.Contains(err.Error(), "no <rdg> text") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseDraft_RdgSurfaceMismatch(t *testing.T) {
	content := "見らむ\t#w.見る\nrdg\tみ\t#w.見ゆ\n"
	_, err := ParseDraft(content, []string{"見らむ"}, []string{"みえん"})
	if err == nil {
		t.Fatal("expected rdg mismatch error")
	}
	if !strings.Contains(err.Error(), "rdg segment 1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseDraft_LemAfterRdgIsError(t *testing.T) {
	content := "見らむ\t#w.見る\nrdg\tみえん\t#rdg.?\n春\t#w.春\n"
	_, err := ParseDraft(content, []string{"見らむ春"}, []string{"みえん"})
	if err == nil {
		t.Fatal("expected error for lem row after rdg row")
	}
	if !strings.Contains(err.Error(), "lem row found after rdg row") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseDraft_RdgMissingLemmaRef(t *testing.T) {
	content := "見らむ\t#w.見る\nrdg\tみえん\n"
	_, err := ParseDraft(content, []string{"見らむ"}, []string{"みえん"})
	if err == nil {
		t.Fatal("expected error for rdg row with missing lemmaRef")
	}
	if !strings.Contains(err.Error(), "three columns") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseDraft_RdgPlaceholderLemmaRef(t *testing.T) {
	content := "見らむ\t#w.見る\nrdg\tみえん\t#rdg.?\n"
	got, err := ParseDraft(content, []string{"見らむ"}, []string{"みえん"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got[0].Rdg[0].LemmaRef != "#rdg.?" {
		t.Errorf("placeholder lemmaRef wrong: %q", got[0].Rdg[0].LemmaRef)
	}
}

func TestParseDraftGroups_SkipsRdgRows(t *testing.T) {
	content := "春\t#w.春.h1\nrdg\tみえん\t#rdg.?\n"
	groups := ParseDraftGroups(content)
	if len(groups) != 1 {
		t.Fatalf("want 1 group, got %d", len(groups))
	}
	if len(groups[0]) != 1 || groups[0][0].Surface != "春" {
		t.Errorf("rdg row should be skipped, got %v", groups[0])
	}
}

// --- GenerateDraft ---

func TestGenerateDraft_ContainsHeader(t *testing.T) {
	tokens := []Token{{Surface: "春", LemmaRef: "#w.春.h1"}}
	out := GenerateDraft(1, tokens, []SegMeta{{Text: "春"}}, nil)
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
	out := GenerateDraft(1, tokens, []SegMeta{{Text: "春は"}}, nil)
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
	metas := []SegMeta{{Text: "春は"}, {Text: "けり"}}
	splits := []int{2, 1}
	out := GenerateDraft(1, tokens, metas, splits)
	if !strings.Contains(out, "seg 1") || !strings.Contains(out, "seg 2") {
		t.Error("missing seg headers")
	}
}

func TestGenerateDraft_MatchedTokenMarkedOK(t *testing.T) {
	tokens := []Token{{Surface: "春", LemmaRef: "#w.春.h1"}}
	out := GenerateDraft(1, tokens, []SegMeta{{Text: "春"}}, []int{1})
	if !strings.Contains(out, "✓") {
		t.Error("matched token should be marked ✓")
	}
}

func TestGenerateDraft_MismatchedTokenMarkedCheck(t *testing.T) {
	// No Reading/Lemma set → effectiveFirst = '一' (from Surface "一とせ").
	// Assigned surface "ひとゝせ" starts with 'ひ' ≠ '一' → initial char mismatch.
	tokens := []Token{{Surface: "一とせ", LemmaRef: "#w.一年"}}
	out := GenerateDraft(1, tokens, []SegMeta{{Text: "ひとゝせ"}}, []int{1})
	if !strings.Contains(out, "? check initial char") {
		t.Error("initial char mismatch should be marked '? check initial char'")
	}
	if !strings.Contains(out, "✗") {
		t.Error("seg header should show ✗ mismatch")
	}
}

func TestGenerateDraft_InitialCharMismatch(t *testing.T) {
	// Reading set to われ → effectiveFirst = 'わ'. Surface め ≠ わ → warning.
	tokens := []Token{{Surface: "我", LemmaRef: "#w.我", Reading: "われ"}}
	out := GenerateDraft(1, tokens, []SegMeta{{Text: "め"}}, []int{1})
	if !strings.Contains(out, "? check initial char") {
		t.Error("initial char mismatch (め≠わ) should be flagged")
	}
}

func TestGenerateDraft_InitialCharOKSurfaceDiffers(t *testing.T) {
	// Reading おもふ → effectiveFirst 'お'. Assigned surface おもひ starts with
	// 'お' → initial char OK, but surface differs → "was: ..." marker.
	tokens := []Token{{Surface: "おもふ", LemmaRef: "#w.思ふ", Reading: "おもふ"}}
	out := GenerateDraft(1, tokens, []SegMeta{{Text: "おもひ"}}, []int{1})
	if !strings.Contains(out, "was: おもふ") {
		t.Error("surface mismatch with correct initial char should show 'was: <orig>'")
	}
	if strings.Contains(out, "? check initial char") {
		t.Error("correct initial char should not trigger initial char warning")
	}
}

func TestGenerateDraft_RoundTrip(t *testing.T) {
	// GenerateDraft output must be parseable by ParseDraft when surfaces match.
	tokens := []Token{
		{Surface: "春", LemmaRef: "#w.春.h1"},
		{Surface: "は", LemmaRef: "#w.は"},
		{Surface: "けり", LemmaRef: "#w.けり"},
	}
	metas := []SegMeta{{Text: "春は"}, {Text: "けり"}}
	segTexts := []string{"春は", "けり"}
	splits := []int{2, 1}
	draft := GenerateDraft(1, tokens, metas, splits)
	got, err := ParseDraft(draft, segTexts, nil)
	if err != nil {
		t.Fatalf("ParseDraft failed on GenerateDraft output: %v", err)
	}
	if len(got) != 2 || len(got[0].Lem) != 2 || len(got[1].Lem) != 1 {
		t.Errorf("unexpected parse result: %v", got)
	}
}

func TestGenerateDraft_ShowsRdgComment(t *testing.T) {
	tokens := []Token{{Surface: "見らむ", LemmaRef: "#w.見る"}}
	metas := []SegMeta{{Text: "見らむ", LemText: "見らむ", RdgText: "みえん"}}
	out := GenerateDraft(1, tokens, metas, []int{1})
	if !strings.Contains(out, "# app: lem=見らむ | rdg=みえん") {
		t.Errorf("missing rdg comment line, got:\n%s", out)
	}
}

func TestGenerateDraft_EmitsRdgRow(t *testing.T) {
	tokens := []Token{{Surface: "見らむ", LemmaRef: "#w.見る"}}
	metas := []SegMeta{{Text: "見らむ", LemText: "見らむ", RdgText: "みえん"}}
	out := GenerateDraft(1, tokens, metas, []int{1})
	if !strings.Contains(out, "rdg\tみえん\t#rdg.?") {
		t.Errorf("missing rdg row, got:\n%s", out)
	}
}

func TestGenerateDraft_NoRdgRowWhenRdgTextEmpty(t *testing.T) {
	tokens := []Token{{Surface: "春", LemmaRef: "#w.春.h1"}}
	metas := []SegMeta{{Text: "春"}}
	out := GenerateDraft(1, tokens, metas, []int{1})
	if strings.Contains(out, "rdg\t") {
		t.Error("should not emit rdg row when RdgText is empty")
	}
	if strings.Contains(out, "# app:") {
		t.Error("should not emit # app: comment when RdgText is empty")
	}
}

func TestGenerateDraft_RoundTripWithRdgRows(t *testing.T) {
	// Draft with rdg row must parse correctly; placeholder lemmaRef is preserved.
	tokens := []Token{
		{Surface: "花とや", LemmaRef: "#w.花"},
		{Surface: "見らむ", LemmaRef: "#w.見る"},
	}
	metas := []SegMeta{{Text: "花とや見らむ", LemText: "見らむ", RdgText: "みえん"}}
	segTexts := []string{"花とや見らむ"}
	rdgTexts := []string{"みえん"}
	splits := []int{2}
	draft := GenerateDraft(1, tokens, metas, splits)
	if !strings.Contains(draft, "rdg\tみえん\t#rdg.?") {
		t.Fatalf("draft missing rdg row:\n%s", draft)
	}
	got, err := ParseDraft(draft, segTexts, rdgTexts)
	if err != nil {
		t.Fatalf("ParseDraft failed: %v", err)
	}
	if len(got) != 1 || len(got[0].Lem) != 2 {
		t.Errorf("unexpected lem result: %v", got)
	}
	if len(got[0].Rdg) != 1 || got[0].Rdg[0].Surface != "みえん" || got[0].Rdg[0].LemmaRef != "#rdg.?" {
		t.Errorf("unexpected rdg result: %v", got[0].Rdg)
	}
}
