// Package annotate provides alignment and annotation logic for wrapping
// Karoku 2 segment text in <w lemmaRef="…"> elements using Hachidaishu tokens.
package annotate

import "unicode/utf8"

// Token is a single annotated word: its surface form and lemma reference.
type Token struct {
	Surface  string
	LemmaRef string
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

// EstimateSplits distributes tokens across segments proportionally by rune
// count when exact alignment is not possible. It returns the number of tokens
// assigned to each segment, suitable for use as the splits argument to
// GenerateDraft.
func EstimateSplits(tokens []Token, segTexts []string) []int {
	if len(segTexts) == 0 || len(tokens) == 0 {
		return nil
	}

	totalTokenRunes := 0
	for _, t := range tokens {
		totalTokenRunes += utf8.RuneCountInString(t.Surface)
	}
	totalSegRunes := 0
	for _, s := range segTexts {
		totalSegRunes += utf8.RuneCountInString(s)
	}
	if totalSegRunes == 0 || totalTokenRunes == 0 {
		return nil
	}

	// Precompute cumulative segment rune boundaries.
	cumBoundaries := make([]int, len(segTexts))
	cum := 0
	for i, s := range segTexts {
		cum += utf8.RuneCountInString(s)
		cumBoundaries[i] = cum
	}

	splits := make([]int, len(segTexts))
	si := 0
	tokenCum := 0
	for _, tok := range tokens {
		tokRunes := utf8.RuneCountInString(tok.Surface)
		tokenCum += tokRunes
		// Advance segment based on whether this token's midpoint crosses the boundary.
		// Midpoint of this token is at (tokenCum - tokRunes/2) / totalTokenRunes.
		// Use cross-multiplication to avoid division:
		//   (2*tokenCum - tokRunes) / (2*totalTokenRunes) >= cumBoundaries[si] / totalSegRunes
		// ⟺  (2*tokenCum - tokRunes) * totalSegRunes >= 2 * cumBoundaries[si] * totalTokenRunes
		if si < len(segTexts)-1 {
			if (2*tokenCum-tokRunes)*totalSegRunes >= 2*cumBoundaries[si]*totalTokenRunes {
				si++
			}
		}
		splits[si]++
	}
	return splits
}
