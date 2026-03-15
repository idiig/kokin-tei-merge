// Package annotate provides alignment and annotation logic for wrapping
// Karoku 2 segment text in <w lemmaRef="…"> elements using Hachidaishu tokens.
package annotate

import (
	"strings"
	"unicode/utf8"
)

// Token is a single annotated word: its surface form and lemma reference.
// Lemma holds the kanji/mixed orth from form[@type='lemma']/orth.
// Reading holds the kana pronunciation from form[@type='lemma']/pron[@notation='kana'].
// Both are used as rune-count proxies when Surface is a placeholder ("???").
type Token struct {
	Surface  string
	LemmaRef string
	Lemma    string // dictionary orth (kanji/mixed form)
	Reading  string // dictionary kana pronunciation
}

// absInt returns the absolute value of x.
func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// effectiveRunes returns the rune count used for proportional estimation.
// Priority: Reading (kana, best proxy for Karoku) → Lemma (orth, dictionary
// form) → Surface (raw Hachidaishu text). Karoku uses kana orthography while
// Hachidaishu may use kanji, so kana reading gives the most accurate length.
func effectiveRunes(tok Token) int {
	if tok.Reading != "" {
		return utf8.RuneCountInString(tok.Reading)
	}
	if tok.Lemma != "" {
		return utf8.RuneCountInString(tok.Lemma)
	}
	return utf8.RuneCountInString(tok.Surface)
}

// effectiveFirstRune returns the first rune of the token's canonical form,
// using Reading → Lemma → Surface priority (matching effectiveRunes).
// Returns utf8.RuneError if all fields are empty.
func effectiveFirstRune(tok Token) rune {
	s := tok.Reading
	if s == "" {
		s = tok.Lemma
	}
	if s == "" {
		s = tok.Surface
	}
	r, _ := utf8.DecodeRuneInString(s)
	return r
}

// tokenSurfaceScore returns a score estimating how well surf matches tok.
// Higher is better. Signals: initial char match (+3/-3), reading/lemma length
// match (+n or -penalty), and exact original surface match (+10).
func tokenSurfaceScore(surf string, tok Token) int {
	n := utf8.RuneCountInString(surf)
	s := 0

	ef := effectiveFirstRune(tok)
	sf, _ := utf8.DecodeRuneInString(surf)
	if ef != utf8.RuneError && sf == ef {
		s += 3
	} else if ef != utf8.RuneError {
		s -= 3
	}

	if tok.Reading != "" {
		r := utf8.RuneCountInString(tok.Reading)
		if n == r {
			s += r
		} else {
			s -= absInt(n - r)
		}
	}
	if tok.Lemma != "" {
		l := utf8.RuneCountInString(tok.Lemma)
		if n == l {
			s += l
		} else {
			s -= absInt(n - l)
		}
	}
	if surf == tok.Surface {
		s += 10
	}
	return s
}

// segAssignScore scores assigning toks to segText in EstimateSplits Phase 2.
func segAssignScore(toks []Token, segText string) int {
	var concat string
	for _, t := range toks {
		concat += t.Surface
	}
	if concat == segText {
		return 1000
	}
	surfaces := splitSegByRunes(segText, toks)
	total := 0
	for i, surf := range surfaces {
		total += tokenSurfaceScore(surf, toks[i])
	}
	return total
}

// AlignPoem attempts to align the ordered token list against the ordered
// segment texts using exact prefix matching. It returns per-segment token
// slices (one slice per segment) and true if every token was consumed and
// every segment character was consumed. On any mismatch it returns nil, false.
func AlignPoem(tokens []Token, segTexts []string) ([][]Token, bool) {
	result := make([][]Token, len(segTexts))
	tokenIdx := 0

	for si, seg := range segTexts {
		pos := 0
		for pos < len(seg) {
			if tokenIdx >= len(tokens) {
				// Segment text remains but no more tokens.
				return nil, false
			}
			surf := tokens[tokenIdx].Surface
			// Check whether the remaining segment text starts with this surface.
			if len(seg)-pos < len(surf) || seg[pos:pos+len(surf)] != surf {
				return nil, false
			}
			result[si] = append(result[si], tokens[tokenIdx])
			pos += len(surf)
			tokenIdx++
		}
	}

	// All tokens must be consumed.
	if tokenIdx != len(tokens) {
		return nil, false
	}
	return result, true
}

// EstimateSplits distributes tokens across segments. It first attempts a
// greedy exact-prefix pass: tokens whose surface exactly matches the front of
// the current segment are greedily consumed. When an exact match fails (e.g.
// a placeholder "???" or an orthographic variant), the remaining tokens and
// segments are handled by a DP that maximises segAssignScore.
// Empty segments (whitespace-only or structural) receive zero tokens.
func EstimateSplits(tokens []Token, segTexts []string) []int {
	if len(segTexts) == 0 || len(tokens) == 0 {
		return nil
	}

	splits := make([]int, len(segTexts))
	remaining := make([]string, len(segTexts))
	copy(remaining, segTexts)

	// Advance si past any leading empty segs.
	si := 0
	for si < len(remaining) && remaining[si] == "" {
		si++
	}

	// Phase 1: greedy exact prefix matching.
	ti := 0
	for ti < len(tokens) && si < len(segTexts) {
		surf := tokens[ti].Surface
		rem := remaining[si]
		if surf == "???" || !strings.HasPrefix(rem, surf) {
			break
		}
		splits[si]++
		remaining[si] = rem[len(surf):]
		if remaining[si] == "" {
			si++
			for si < len(segTexts) && remaining[si] == "" {
				si++
			}
		}
		ti++
	}

	if ti == len(tokens) {
		return splits
	}

	// Phase 2: DP to distribute remaining tokens [ti:] across non-empty
	// remaining segments. Score each assignment with segAssignScore.
	type segEntry struct {
		idx  int
		text string
	}
	var nonEmpty []segEntry
	for i := si; i < len(segTexts); i++ {
		if remaining[i] != "" {
			nonEmpty = append(nonEmpty, segEntry{i, remaining[i]})
		}
	}
	remTokens := tokens[ti:]
	if len(nonEmpty) == 0 {
		if si < len(segTexts) {
			splits[si] += len(remTokens)
		}
		return splits
	}

	nSeg := len(nonEmpty)
	nTok := len(remTokens)

	// Degenerate: fewer tokens than segments — assign 1 per seg where possible.
	if nTok <= nSeg {
		for i := 0; i < nTok; i++ {
			splits[nonEmpty[i].idx]++
		}
		return splits
	}

	const minScore = -(1 << 30)
	// dp[i][j] = best total score for nonEmpty[0..i-1] using remTokens[0..j-1].
	dp := make([][]int, nSeg+1)
	from := make([][]int, nSeg+1) // from[i][j] = k tokens assigned to seg i-1
	for i := range dp {
		dp[i] = make([]int, nTok+1)
		from[i] = make([]int, nTok+1)
		for j := range dp[i] {
			dp[i][j] = minScore
		}
	}
	dp[0][0] = 0

	for i := 1; i <= nSeg; i++ {
		segText := nonEmpty[i-1].text
		// j = tokens consumed so far; each remaining seg needs at least 1.
		for j := i; j <= nTok-(nSeg-i); j++ {
			for k := 1; k <= j-(i-1); k++ {
				if dp[i-1][j-k] == minScore {
					continue
				}
				score := dp[i-1][j-k] + segAssignScore(remTokens[j-k:j], segText)
				if score > dp[i][j] {
					dp[i][j] = score
					from[i][j] = k
				}
			}
		}
	}

	// Backtrack.
	j := nTok
	for i := nSeg; i >= 1; i-- {
		k := from[i][j]
		splits[nonEmpty[i-1].idx] += k
		j -= k
	}
	return splits
}
