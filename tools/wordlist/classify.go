package wordlist

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/beevik/etree"
)

// WLSPHCategory represents one entry from the wlsph.json file.
type WLSPHCategory struct {
	Code   string        `json:"code"`
	Source string        `json:"source"`
	Pos    CategoryField `json:"pos"`
	Group  CategoryField `json:"group"`
	Field  CategoryField `json:"field"`
	Near   CategoryField `json:"near_synonymy"`
	Syn    CategoryField `json:"synonymy"`
}

// CategoryField is a code+category pair in the JSON.
type CategoryField struct {
	Code     string `json:"code"`
	Category string `json:"category"`
}

// Description returns the human-readable label for the category,
// joining non-empty hierarchical components with "-".
func (c *WLSPHCategory) Description() string {
	parts := []string{c.Pos.Category}
	if c.Group.Category != "" {
		parts = append(parts, c.Group.Category)
	}
	if c.Field.Category != "" {
		parts = append(parts, c.Field.Category)
	}
	if c.Near.Category != "" {
		parts = append(parts, c.Near.Category)
	}
	if c.Syn.Category != "" && c.Syn.Category != c.Near.Category {
		parts = append(parts, c.Syn.Category)
	}
	return strings.Join(parts, "-")
}

// WLSPHFile is the top-level structure of wlsph.json.
type WLSPHFile struct {
	Categories []WLSPHCategory `json:"categories"`
}

// LoadWLSPH reads the wlsph.json file and returns category data.
func LoadWLSPH(path string) ([]WLSPHCategory, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var f WLSPHFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return f.Categories, nil
}

// WLSPCode holds extracted info for a WLSP classification item.
type WLSPCode struct {
	Code        string // e.g. "1.1630"
	Description string // e.g. "体-関係-時間-年"
}

// ExtractWLSPCodes collects unique WLSP codes from parsed MSD data.
func ExtractWLSPCodes(tokens []Token) []WLSPCode {
	seen := make(map[string]string) // code → description
	for _, tok := range tokens {
		if tok.MSD.WLSP == "" {
			continue
		}
		if _, exists := seen[tok.MSD.WLSP]; !exists {
			seen[tok.MSD.WLSP] = tok.MSD.WLSPDescription
		}
	}

	codes := make([]WLSPCode, 0, len(seen))
	for code, desc := range seen {
		codes = append(codes, WLSPCode{Code: code, Description: desc})
	}
	sort.Slice(codes, func(i, j int) bool {
		return codes[i].Code < codes[j].Code
	})
	return codes
}

// BuildClassificationDiv creates a <div type="classification"> element.
func BuildClassificationDiv(xmlID, heading string, items []ClassItem) *etree.Element {
	div := etree.NewElement("div")
	div.CreateAttr("type", "classification")
	div.CreateAttr("xml:id", xmlID)
	head := div.CreateElement("head")
	head.SetText(heading)
	list := div.CreateElement("list")
	for _, item := range items {
		li := list.CreateElement("item")
		li.CreateAttr("xml:id", item.ID)
		label := li.CreateElement("label")
		label.SetText(item.Label)
		if item.Desc != "" {
			desc := li.CreateElement("desc")
			desc.SetText(item.Desc)
		}
	}
	return div
}

type ClassItem struct {
	ID    string
	Label string
	Desc  string
}

// BuildWLSPHItems converts WLSPH categories to classification items,
// deduplicating by code (the source JSON may contain duplicates).
func BuildWLSPHItems(cats []WLSPHCategory) []ClassItem {
	seen := make(map[string]bool)
	var items []ClassItem
	for _, c := range cats {
		if seen[c.Code] {
			continue
		}
		seen[c.Code] = true
		items = append(items, ClassItem{
			ID:    "WLSPH." + c.Code,
			Label: c.Code,
			Desc:  c.Description(),
		})
	}
	return items
}

// WLSPHCodesFromTokens collects the set of WLSPH codes referenced by tokens.
func WLSPHCodesFromTokens(tokens []Token) map[string]bool {
	codes := make(map[string]bool)
	for _, tok := range tokens {
		if tok.MSD.WLSPH != "" {
			codes[tok.MSD.WLSPH] = true
		}
	}
	return codes
}

// AddMissingWLSPH appends placeholder items for WLSPH codes used in the data
// but not present in the JSON source. This ensures all @ana references resolve.
func AddMissingWLSPH(items []ClassItem, usedCodes map[string]bool) []ClassItem {
	existing := make(map[string]bool)
	for _, item := range items {
		existing[item.Label] = true
	}

	var missing []string
	for code := range usedCodes {
		if !existing[code] {
			missing = append(missing, code)
		}
	}
	sort.Strings(missing)

	for _, code := range missing {
		items = append(items, ClassItem{
			ID:    "WLSPH." + code,
			Label: code,
		})
	}
	return items
}

// BuildWLSPItems converts WLSP codes to classification items.
func BuildWLSPItems(codes []WLSPCode) []ClassItem {
	items := make([]ClassItem, len(codes))
	for i, c := range codes {
		items[i] = ClassItem{
			ID:    "WLSP." + c.Code,
			Label: c.Code,
			Desc:  c.Description,
		}
	}
	return items
}
