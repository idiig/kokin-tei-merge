// Package annotate provides alignment and annotation logic for wrapping
// Karoku 2 segment text in <w lemmaRef="…"> elements using Hachidaishu tokens.
package annotate

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
