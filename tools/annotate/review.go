package annotate

import (
	"fmt"
	"strings"
)

// GenerateDraft produces the editable draft file content for one unmatched poem.
// splits is the number of tokens per segment (used to pre-insert blank line
// separators and per-token match hints); if nil, all tokens appear flat.
func GenerateDraft(n int, tokens []Token, segTexts []string, splits []int) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# Poem %d\n", n)
	sb.WriteString("#\n")
	sb.WriteString("# Columns: surface (edit this) | lemmaRef (do not edit)\n")
	sb.WriteString("# Rules:\n")
	sb.WriteString("#   - Each blank-line group = one segment (exactly one group per segment).\n")
	sb.WriteString("#   - Surfaces in each group must concatenate to the segment text exactly.\n")
	sb.WriteString("#   - Delete all non-comment lines to skip this poem.\n")
	sb.WriteString("#\n")

	if len(splits) == len(segTexts) {
		ti := 0
		for si, count := range splits {
			// Compute current surface concatenation for this segment.
			var concat string
			for k := 0; k < count && ti+k < len(tokens); k++ {
				concat += tokens[ti+k].Surface
			}
			status := "✓"
			if concat != segTexts[si] {
				status = fmt.Sprintf("✗  need: %s  got: %s", segTexts[si], concat)
			}
			fmt.Fprintf(&sb, "# — seg %d [%s]\n", si+1, status)

			// Emit tokens with per-token match hint.
			pos := 0
			segText := segTexts[si]
			for k := 0; k < count && ti < len(tokens); k++ {
				tok := tokens[ti]
				marker := "? check surface"
				if pos < len(segText) && strings.HasPrefix(segText[pos:], tok.Surface) {
					marker = "✓"
					pos += len(tok.Surface)
				}
				fmt.Fprintf(&sb, "%s\t%s\t# %s\n", tok.Surface, tok.LemmaRef, marker)
				ti++
			}
			sb.WriteString("\n")
		}
	} else {
		// No split hint — flat list, no hints.
		for _, tok := range tokens {
			fmt.Fprintf(&sb, "%s\t%s\n", tok.Surface, tok.LemmaRef)
		}
	}
	return sb.String()
}

// ParseDraft parses a user-edited draft file. It returns per-segment token
// slices when the draft is valid, nil (no error) when the user skipped the
// poem (deleted all data lines), or an error when the draft is malformed.
//
// Lines have 2 or 3 tab-separated columns: surface, lemmaRef, optional comment.
// Validation rules:
//   - The number of blank-line-separated groups must equal len(segTexts).
//   - For each group, concatenating the surfaces must equal segTexts[i].
func ParseDraft(content string, segTexts []string) ([][]Token, error) {
	lines := strings.Split(content, "\n")

	var groups [][]Token
	var current []Token

	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.TrimSpace(line) == "" {
			if len(current) > 0 {
				groups = append(groups, current)
				current = nil
			}
			continue
		}
		// Allow 2 or 3 tab-separated columns (3rd is hint comment).
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid line (expected surface TAB lemmaRef): %q", line)
		}
		surface := strings.TrimSpace(parts[0])
		lemmaRef := strings.TrimSpace(parts[1])
		if surface == "" {
			return nil, fmt.Errorf("empty surface on line: %q", line)
		}
		current = append(current, Token{Surface: surface, LemmaRef: lemmaRef})
	}
	if len(current) > 0 {
		groups = append(groups, current)
	}

	// User deleted all data lines → skip.
	if len(groups) == 0 {
		return nil, nil
	}

	if len(groups) != len(segTexts) {
		return nil, fmt.Errorf("got %d token group(s), want %d (one per segment)", len(groups), len(segTexts))
	}

	for i, group := range groups {
		var concat string
		for _, tok := range group {
			concat += tok.Surface
		}
		if concat != segTexts[i] {
			return nil, fmt.Errorf("segment %d: surfaces %q ≠ Karoku text %q", i+1, concat, segTexts[i])
		}
	}

	return groups, nil
}
