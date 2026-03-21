package annotate

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// GenerateDraft produces the editable draft file content for one unmatched poem.
// splits is the number of tokens per segment (used to pre-insert blank line
// separators and per-token match hints); if nil, all tokens appear flat.
// When a segment contains an <app> element, a read-only comment line is emitted
// showing the lem and rdg texts so the user can see the variant while editing:
//
//	# app: lem=見らむ | rdg=みえん
func GenerateDraft(n int, tokens []Token, metas []SegMeta, splits []int) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# Poem %d\n", n)
	sb.WriteString("#\n")
	sb.WriteString("# Columns: surface (edit this) | lemmaRef (space-separated list OK, e.g. #w.つ.つれ #w.つ.h1)\n")
	sb.WriteString("# Rules:\n")
	sb.WriteString("#   - Each blank-line group = one segment (exactly one group per segment).\n")
	sb.WriteString("#   - Surfaces in each group must concatenate to the segment text exactly.\n")
	sb.WriteString("#   - A token elided in Karoku (only の/が/か): leave surface empty (TAB then lemmaRef).\n")
	sb.WriteString("#   - Delete all non-comment lines to skip this poem.\n")
	sb.WriteString("#\n")

	if len(splits) == len(metas) {
		ti := 0
		for si, count := range splits {
			segText := metas[si].Text
			// Compute current surface concatenation for this segment.
			var concat string
			for k := 0; k < count && ti+k < len(tokens); k++ {
				concat += tokens[ti+k].Surface
			}
			status := "✓"
			if concat != segText {
				status = fmt.Sprintf("✗  need: %s  got: %s", segText, concat)
			}
			fmt.Fprintf(&sb, "# — seg %d [%s]\n", si+1, status)
			// Apparatus comment: show rdg variant so the user can see it while editing.
			if metas[si].RdgText != "" {
				fmt.Fprintf(&sb, "# app: lem=%s | rdg=%s\n", metas[si].LemText, metas[si].RdgText)
			}

			// Emit tokens with per-token match hint.
			segMatched := (concat == segText)
			var surfaces []string
			if segMatched {
				// Exact match: use Hachidaishu surfaces directly.
				for k := 0; k < count && ti+k < len(tokens); k++ {
					surfaces = append(surfaces, tokens[ti+k].Surface)
				}
			} else {
				// Mismatch: split Karoku seg text proportionally by token rune counts.
				tokSlice := tokens[ti : ti+count]
				surfaces = splitSegByRunes(segText, tokSlice)
			}
			for k := 0; k < count && ti < len(tokens); k++ {
				tok := tokens[ti]
				surf := surfaces[k]
				efFirst := effectiveFirstRune(tok)
				sfFirst, _ := utf8.DecodeRuneInString(surf)
				initialOK := efFirst == utf8.RuneError || efFirst == sfFirst
				var marker string
				switch {
				case !initialOK:
					marker = fmt.Sprintf("? check initial char (%c≠%c)", sfFirst, efFirst)
				case surf == tok.Surface:
					marker = "✓"
				default:
					marker = fmt.Sprintf("was: %s", tok.Surface)
				}
				fmt.Fprintf(&sb, "%s\t%s\t# %s\n", surf, tok.LemmaRef, marker)
				ti++
			}
			// Emit rdg rows: individual tokens when already annotated, placeholder otherwise.
			if len(metas[si].RdgTokens) > 0 {
				for _, rt := range metas[si].RdgTokens {
					fmt.Fprintf(&sb, "rdg\t%s\t%s\n", rt.Surface, rt.LemmaRef)
				}
			} else if metas[si].RdgText != "" {
				fmt.Fprintf(&sb, "rdg\t%s\t#rdg.?\n", metas[si].RdgText)
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

// splitSegByRunes splits segText into len(tokens) parts using a DP that
// maximises tokenSurfaceScore for each token/surface assignment.
// Each token receives at least one rune; the last token absorbs any remainder.
func splitSegByRunes(segText string, tokens []Token) []string {
	segRunes := []rune(segText)
	m := len(segRunes)
	n := len(tokens)

	if n == 0 || m == 0 {
		return make([]string, n)
	}
	if n == 1 {
		return []string{segText}
	}
	// Degenerate: more tokens than runes — give each token one rune if possible.
	if m <= n {
		result := make([]string, n)
		for i := 0; i < n; i++ {
			if i < m {
				result[i] = string(segRunes[i])
			}
		}
		return result
	}

	const minScore = -(1 << 30)
	// dp[i][j] = best score for tokens[0..i-1] using segRunes[0..j-1].
	dp := make([][]int, n+1)
	cut := make([][]int, n+1) // cut[i][j] = k runes assigned to token i-1
	for i := range dp {
		dp[i] = make([]int, m+1)
		cut[i] = make([]int, m+1)
		for j := range dp[i] {
			dp[i][j] = minScore
		}
	}
	dp[0][0] = 0

	for i := 1; i <= n; i++ {
		tok := tokens[i-1]
		// Each remaining token needs at least 1 rune.
		for j := i; j <= m-(n-i); j++ {
			for k := 1; k <= j-(i-1); k++ {
				if dp[i-1][j-k] == minScore {
					continue
				}
				surf := string(segRunes[j-k : j])
				score := dp[i-1][j-k] + tokenSurfaceScore(surf, tok)
				if score > dp[i][j] {
					dp[i][j] = score
					cut[i][j] = k
				}
			}
		}
	}

	// Backtrack.
	result := make([]string, n)
	j := m
	for i := n; i >= 1; i-- {
		k := cut[i][j]
		result[i-1] = string(segRunes[j-k : j])
		j -= k
	}
	return result
}

// SegGroup holds the aligned tokens for one segment's <lem> content and,
// optionally, its <rdg> content. Rdg is nil when no rdg annotation was
// requested — the <rdg> element is then left unannotated (current behaviour).
type SegGroup struct {
	Lem []Token
	Rdg []Token // nil = leave <rdg> unannotated
}

// ParseDraftGroups parses a draft file leniently, returning lem-only token
// groups without validating surfaces against segment texts. Returns nil if the
// file has no data lines. Malformed tab-lines and rdg-prefixed rows are
// silently skipped. Used by mergeWithExistingDraft.
func ParseDraftGroups(content string) [][]Token {
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
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 2 {
			continue
		}
		// Skip rdg-annotation rows — they are not part of the lem token list.
		if strings.TrimSpace(parts[0]) == "rdg" {
			continue
		}
		surface := strings.TrimSpace(parts[0])
		lemmaRef := strings.TrimSpace(parts[1])
		if surface == "" && lemmaRef == "" {
			continue
		}
		current = append(current, Token{Surface: surface, LemmaRef: lemmaRef})
	}
	if len(current) > 0 {
		groups = append(groups, current)
	}
	if len(groups) == 0 {
		return nil
	}
	return groups
}

// isElideableParticle reports whether lemmaRef refers to one of the permitted
// elideable particles: の (no), が (ga), か (ka).
// lemmaRef is in new two-layer format "reading.lemma" (e.g. "の.の", "が.が").
// Multiple space-separated refs are supported — any matching ref returns true.
func isElideableParticle(lemmaRef string) bool {
	for _, ref := range strings.Fields(lemmaRef) {
		ref = strings.TrimPrefix(ref, "#")
		if idx := strings.Index(ref, "."); idx >= 0 {
			lemma := ref[idx+1:]
			switch lemma {
			case "の", "が", "か":
				return true
			}
		}
	}
	return false
}

// ParseDraft parses a user-edited draft file. It returns per-segment SegGroups
// when the draft is valid, nil (no error) when the user skipped the poem
// (deleted all data lines), or an error when the draft is malformed.
//
// Lines have 2 or 3 tab-separated columns: surface, lemmaRef, optional comment.
// Rdg-annotation rows have "rdg" as the first column and must appear after all
// lem rows in their group (before the blank-line separator). # app: comment
// lines are silently ignored.
//
// rdgTexts[i] is the expected concatenation of rdg surfaces for segment i
// (empty when the segment has no <app><rdg>). ParseDraft validates rdg
// surfaces against rdgTexts when rdg rows are present.
func ParseDraft(content string, segTexts []string, rdgTexts []string) ([]SegGroup, error) {
	lines := strings.Split(content, "\n")

	var groups []SegGroup
	var curLem []Token
	var curRdg []Token
	inRdgPhase := false

	flushGroup := func() {
		if len(curLem) > 0 {
			g := SegGroup{Lem: curLem}
			if len(curRdg) > 0 {
				g.Rdg = curRdg
			}
			groups = append(groups, g)
		}
		curLem = nil
		curRdg = nil
		inRdgPhase = false
	}

	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.TrimSpace(line) == "" {
			flushGroup()
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid line (expected surface TAB lemmaRef): %q", line)
		}
		col0 := strings.TrimSpace(parts[0])
		if col0 == "rdg" {
			// rdg annotation row: rdg TAB surface TAB lemmaRef
			if len(parts) < 3 {
				return nil, fmt.Errorf("rdg row requires three columns (rdg TAB surface TAB lemmaRef): %q", line)
			}
			surf := strings.TrimSpace(parts[1])
			ref := strings.TrimSpace(parts[2])
			if surf == "" {
				return nil, fmt.Errorf("empty surface on rdg row: %q", line)
			}
			inRdgPhase = true
			curRdg = append(curRdg, Token{Surface: surf, LemmaRef: ref})
			continue
		}
		// Lem row.
		if inRdgPhase {
			return nil, fmt.Errorf("lem row found after rdg row in the same group: %q", line)
		}
		surface := col0
		lemmaRef := strings.TrimSpace(parts[1])
		if surface == "" {
			if lemmaRef == "" {
				return nil, fmt.Errorf("empty surface and lemmaRef on line: %q", line)
			}
			if !isElideableParticle(lemmaRef) {
				return nil, fmt.Errorf("empty surface only allowed for の/が/か particles, got %q: %q", lemmaRef, line)
			}
		}
		curLem = append(curLem, Token{Surface: surface, LemmaRef: lemmaRef})
	}
	flushGroup()

	// User deleted all data lines → skip.
	if len(groups) == 0 {
		return nil, nil
	}

	if len(groups) != len(segTexts) {
		return nil, fmt.Errorf("got %d token group(s), want %d (one per segment)", len(groups), len(segTexts))
	}

	for i, g := range groups {
		// Validate lem surfaces.
		var lemConcat string
		for _, tok := range g.Lem {
			lemConcat += tok.Surface
		}
		if lemConcat != segTexts[i] {
			return nil, fmt.Errorf("segment %d: surfaces %q ≠ Karoku text %q", i+1, lemConcat, segTexts[i])
		}
		// Validate rdg surfaces when present.
		if len(g.Rdg) > 0 {
			rdgExpected := ""
			if i < len(rdgTexts) {
				rdgExpected = rdgTexts[i]
			}
			if rdgExpected == "" {
				return nil, fmt.Errorf("segment %d: rdg rows present but segment has no <rdg> text", i+1)
			}
			var rdgConcat string
			for _, tok := range g.Rdg {
				rdgConcat += tok.Surface
			}
			if rdgConcat != rdgExpected {
				return nil, fmt.Errorf("rdg segment %d: surfaces %q ≠ rdg text %q", i+1, rdgConcat, rdgExpected)
			}
		}
	}

	return groups, nil
}
