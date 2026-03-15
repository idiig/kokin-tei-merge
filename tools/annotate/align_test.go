package annotate

import (
	"testing"
)

// --- tokenSurfaceScore ---

func TestTokenSurfaceScore_ExactSurface(t *testing.T) {
	tok := Token{Surface: "春", LemmaRef: "#w.春.h1"}
	s := tokenSurfaceScore("春", tok)
	if s < 10 {
		t.Errorf("exact surface match should yield score ≥ 10, got %d", s)
	}
}

func TestTokenSurfaceScore_InitialCharMatch(t *testing.T) {
	tok := Token{Surface: "おもふ", Reading: "おもふ", LemmaRef: "#w.思ふ"}
	sMatch := tokenSurfaceScore("おもひ", tok)    // initial 'お' matches
	sMiss := tokenSurfaceScore("そめ", tok)       // initial 'そ' ≠ 'お'
	if sMatch <= sMiss {
		t.Errorf("initial char match should score higher: match=%d miss=%d", sMatch, sMiss)
	}
}

func TestTokenSurfaceScore_ReadingLengthBonus(t *testing.T) {
	// Reading "おもひ" = 3 runes; surf "おもひ" = 3 runes → length bonus
	tok := Token{Surface: "思ふ", Reading: "おもひ", LemmaRef: "#w.思ふ"}
	sExact := tokenSurfaceScore("おもひ", tok)    // reading length matches
	sShort := tokenSurfaceScore("お", tok)        // too short
	if sExact <= sShort {
		t.Errorf("reading length match should score higher: exact=%d short=%d", sExact, sShort)
	}
}

// --- EstimateSplits (DP Phase 2) ---

func TestEstimateSplits_Poem1001Seg3(t *testing.T) {
	// Poem 1001 regression: seg "おもひそめ" (5 runes) should get 2 tokens
	// (Reading: おもひ + Reading: そめ), not 3.
	tokens := []Token{
		{Surface: "???", LemmaRef: "#w.思ふ", Reading: "おもひ"},
		{Surface: "???", LemmaRef: "#w.染む", Reading: "そめ"},
		{Surface: "???", LemmaRef: "#w.我", Reading: "わ"},
	}
	segs := []string{"おもひそめ", "わ"}
	splits := EstimateSplits(tokens, segs)
	if splits[0] != 2 {
		t.Errorf("seg 0 'おもひそめ': want 2 tokens, got %d", splits[0])
	}
	if splits[1] != 1 {
		t.Errorf("seg 1 'わ': want 1 token, got %d", splits[1])
	}
}

func TestEstimateSplits_ExactPhase1(t *testing.T) {
	// Phase 1 exact match should still work.
	tokens := []Token{
		{Surface: "春", LemmaRef: "#w.春.h1"},
		{Surface: "は", LemmaRef: "#w.は"},
		{Surface: "けり", LemmaRef: "#w.けり"},
	}
	segs := []string{"春は", "けり"}
	splits := EstimateSplits(tokens, segs)
	if splits[0] != 2 || splits[1] != 1 {
		t.Errorf("got splits %v, want [2 1]", splits)
	}
}

func TestAlignPoem(t *testing.T) {
	tests := []struct {
		name      string
		tokens    []Token
		segTexts  []string
		wantOK    bool
		wantSegs  [][]string // surface per segment, nil means we don't check
	}{
		{
			name: "exact match single segment",
			tokens: []Token{
				{Surface: "春", LemmaRef: "#w.春.h1"},
				{Surface: "は", LemmaRef: "#w.は"},
			},
			segTexts: []string{"春は"},
			wantOK:   true,
			wantSegs: [][]string{{"春", "は"}},
		},
		{
			name: "exact match multiple segments",
			tokens: []Token{
				{Surface: "年", LemmaRef: "#w.年"},
				{Surface: "の", LemmaRef: "#w.の.h1"},
				{Surface: "内", LemmaRef: "#w.内"},
				{Surface: "に", LemmaRef: "#w.に.h1"},
				{Surface: "春", LemmaRef: "#w.春.h1"},
				{Surface: "は", LemmaRef: "#w.は"},
			},
			segTexts: []string{"年の内に", "春は"},
			wantOK:   true,
			wantSegs: [][]string{{"年", "の", "内", "に"}, {"春", "は"}},
		},
		{
			name: "leftover segment text → mismatch",
			tokens: []Token{
				{Surface: "春", LemmaRef: "#w.春.h1"},
			},
			segTexts: []string{"春は"},
			wantOK:   false,
		},
		{
			name: "unconsumed tokens → mismatch",
			tokens: []Token{
				{Surface: "春", LemmaRef: "#w.春.h1"},
				{Surface: "は", LemmaRef: "#w.は"},
			},
			segTexts: []string{"春"},
			wantOK:   false,
		},
		{
			name: "token not at start of remaining text → mismatch",
			tokens: []Token{
				{Surface: "は", LemmaRef: "#w.は"},
			},
			segTexts: []string{"春は"},
			wantOK:   false,
		},
		{
			name:     "empty tokens and empty segments",
			tokens:   []Token{},
			segTexts: []string{},
			wantOK:   true,
			wantSegs: [][]string{},
		},
		{
			name:     "empty tokens non-empty segment → mismatch",
			tokens:   []Token{},
			segTexts: []string{"春"},
			wantOK:   false,
		},
		{
			name: "multi-char surface token",
			tokens: []Token{
				{Surface: "けり", LemmaRef: "#w.けり"},
			},
			segTexts: []string{"けり"},
			wantOK:   true,
			wantSegs: [][]string{{"けり"}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := AlignPoem(tc.tokens, tc.segTexts)
			if ok != tc.wantOK {
				t.Fatalf("AlignPoem ok = %v, want %v", ok, tc.wantOK)
			}
			if !tc.wantOK || tc.wantSegs == nil {
				return
			}
			if len(got) != len(tc.wantSegs) {
				t.Fatalf("len(got) = %d, want %d", len(got), len(tc.wantSegs))
			}
			for si, wantSeg := range tc.wantSegs {
				if len(got[si]) != len(wantSeg) {
					t.Errorf("seg[%d]: got %d tokens, want %d", si, len(got[si]), len(wantSeg))
					continue
				}
				for ti, wantSurf := range wantSeg {
					if got[si][ti].Surface != wantSurf {
						t.Errorf("seg[%d][%d].Surface = %q, want %q", si, ti, got[si][ti].Surface, wantSurf)
					}
				}
			}
		})
	}
}
