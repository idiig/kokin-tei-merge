// Command query displays a poem side-by-side from the Hachidaishu (annotated)
// and Karoku 2 (surface-only) sources, to support token alignment review.
//
// Usage:
//
//	go run . -n 1 \
//	  -hachi data/hachidaishu-wordlist.xml \
//	  -karoku data/Kokinwakashu_200003050_20240922.xml
//
// Use -json to emit machine-readable JSON for LLM pipeline consumption.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/beevik/etree"
)

// PoemData is the JSON-serialisable representation of one poem's data.
type PoemData struct {
	Poem        int      `json:"poem"`
	Hachidaishu []Token  `json:"hachidaishu,omitempty"`
	Karoku      []string `json:"karoku,omitempty"`
}

// Token is a single annotated word from the Hachidaishu source.
type Token struct {
	Surface  string `json:"surface"`
	LemmaRef string `json:"lemmaRef"`
}

func main() {
	n := flag.Int("n", 0, "poem number")
	hachiPath := flag.String("hachi", "", "path to hachidaishu-wordlist.xml")
	karokuPath := flag.String("karoku", "", "path to Karoku 2 XML file")
	asJSON := flag.Bool("json", false, "emit JSON instead of plain text")
	flag.Parse()

	if *n <= 0 || *hachiPath == "" || *karokuPath == "" {
		fmt.Fprintf(os.Stderr, "Usage: query -n <poem> -hachi <file> -karoku <file> [-json]\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	hachiDoc := mustRead(*hachiPath)
	karokuDoc := mustRead(*karokuPath)

	hachiTokens := hachiPoem(hachiDoc, *n)
	karokuSegs := karokuPoem(karokuDoc, *n)

	if hachiTokens == nil && karokuSegs == nil {
		fmt.Fprintf(os.Stderr, "poem #%d not found in either file\n", *n)
		os.Exit(1)
	}

	if *asJSON {
		data := PoemData{
			Poem:        *n,
			Hachidaishu: hachiTokens,
			Karoku:      karokuSegs,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(data); err != nil {
			log.Fatalf("json encode: %v", err)
		}
		return
	}

	fmt.Printf("Poem #%d\n", *n)
	fmt.Println(strings.Repeat("─", 60))

	fmt.Println("Hachidaishu (annotated tokens):")
	if hachiTokens == nil {
		fmt.Println("  (not found)")
	} else {
		for _, t := range hachiTokens {
			fmt.Printf("  %-12s  %s\n", t.Surface, t.LemmaRef)
		}
	}

	fmt.Println()
	fmt.Println("Karoku 2 (surface segments):")
	if karokuSegs == nil {
		fmt.Println("  (not found)")
	} else {
		for i, seg := range karokuSegs {
			fmt.Printf("  [%d] %s\n", i+1, seg)
		}
	}

	fmt.Println(strings.Repeat("─", 60))
}

// hachiPoem finds <lg type="waka" n="N"> and returns its <w> tokens.
func hachiPoem(doc *etree.Document, n int) []Token {
	path := fmt.Sprintf("//lg[@type='waka'][@n='%d']", n)
	lg := doc.FindElement(path)
	if lg == nil {
		return nil
	}
	var tokens []Token
	for _, w := range lg.FindElements(".//w") {
		ref := w.SelectAttrValue("lemmaRef", "")
		tokens = append(tokens, Token{Surface: w.Text(), LemmaRef: ref})
	}
	return tokens
}

// karokuPoem finds <l n="N"> and returns the text of each <seg> child.
func karokuPoem(doc *etree.Document, n int) []string {
	path := fmt.Sprintf("//l[@n='%d']", n)
	l := doc.FindElement(path)
	if l == nil {
		return nil
	}
	var segs []string
	for _, seg := range l.SelectElements("seg") {
		segs = append(segs, seg.Text())
	}
	return segs
}

func mustRead(path string) *etree.Document {
	doc := etree.NewDocument()
	doc.ReadSettings.PreserveCData = true
	if err := doc.ReadFromFile(path); err != nil {
		log.Fatalf("reading %s: %v", path, err)
	}
	return doc
}
