package annotate

import (
	"testing"
)

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
