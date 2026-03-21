// Command migrate rewrites lemmaRef attributes in kokin-annotated.xml
// from the old format (w.lemma, w.lemma.hN, w.lemma.form) to the new
// two-layer kana-first format (reading.lemma).
//
// The mapping is derived from the existing <back> in the input file.
//
// Usage:
//
//	go run ./cmd/migrate -input data/kokin-annotated.xml -output data/kokin-annotated.xml
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/beevik/etree"
)

func main() {
	input := flag.String("input", "", "path to kokin-annotated.xml")
	output := flag.String("output", "", "path to output file (may be same as input)")
	flag.Parse()

	if *input == "" || *output == "" {
		fmt.Fprintf(os.Stderr, "Usage: migrate -input <file> -output <file>\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	log.Printf("reading %s", *input)
	doc := etree.NewDocument()
	doc.ReadSettings.PreserveCData = true
	if err := doc.ReadFromFile(*input); err != nil {
		log.Fatalf("error reading input: %v", err)
	}

	log.Println("building old→new ID mapping from <back>...")
	mapping := buildMapping(doc)
	log.Printf("built %d mappings", len(mapping))

	log.Println("rewriting lemmaRef in <body>...")
	rewrote, skipped := rewriteLemmaRefs(doc, mapping)
	log.Printf("rewrote %d, skipped %d", rewrote, skipped)

	log.Printf("writing %s", *output)
	doc.Indent(2)
	doc.WriteSettings.CanonicalAttrVal = true
	doc.WriteSettings.CanonicalText = true
	if err := doc.WriteToFile(*output); err != nil {
		log.Fatalf("error writing output: %v", err)
	}
	log.Println("done!")
}

// buildMapping reads the existing <back><div type="dictionary"> and constructs
// a map from old xml:id values to new reading.orth IDs.
func buildMapping(doc *etree.Document) map[string]string {
	m := make(map[string]string)

	for _, entry := range doc.FindElements("//back//div[@type='dictionary']/entry") {
		oldID := entry.SelectAttrValue("xml:id", "")
		if oldID == "" {
			continue
		}

		// Extract lemma from old ID: strip "w." prefix.
		lemma := strings.TrimPrefix(oldID, "w.")

		// Find primary reading from <form type="lemma"><pron>.
		reading := primaryReading(entry)
		if reading == "" {
			log.Printf("warning: no reading for entry %q, skipping", oldID)
			continue
		}

		newID := reading + "." + lemma

		// Map the lemma entry itself.
		m["#"+oldID] = "#" + newID

		// Map all <hom> children (collapse to same new ID regardless of n).
		for _, hom := range entry.SelectElements("hom") {
			if homID := hom.SelectAttrValue("xml:id", ""); homID != "" {
				m["#"+homID] = "#" + newID
			}
		}

		// Map all <form type="inflected"> children (collapse to same new ID).
		for _, form := range entry.SelectElements("form") {
			if form.SelectAttrValue("type", "") == "inflected" {
				if formID := form.SelectAttrValue("xml:id", ""); formID != "" {
					m["#"+formID] = "#" + newID
				}
			}
		}
	}

	return m
}

// primaryReading returns the first <pron notation="kana"> text inside
// <form type="lemma"> of an entry element.
func primaryReading(entry *etree.Element) string {
	for _, form := range entry.SelectElements("form") {
		if form.SelectAttrValue("type", "") != "lemma" {
			continue
		}
		for _, pron := range form.SelectElements("pron") {
			if t := pron.Text(); t != "" {
				return t
			}
		}
	}
	return ""
}

// rewriteLemmaRefs rewrites lemmaRef attributes in all <w> elements in <body>.
func rewriteLemmaRefs(doc *etree.Document, mapping map[string]string) (rewrote, skipped int) {
	for _, w := range doc.FindElements("//body//w") {
		old := w.SelectAttrValue("lemmaRef", "")
		if old == "" {
			continue
		}
		// lemmaRef allows only a single pointer (teidata.pointer).
		// If multiple space-separated refs exist, keep the first mappable one.
		parts := strings.Fields(old)
		if len(parts) > 1 {
			log.Printf("warning: multiple lemmaRef values %q — keeping first mappable", old)
		}
		var newVal string
		for _, ref := range parts {
			normalized := ref
			if !strings.HasPrefix(normalized, "#") {
				normalized = "#" + normalized
			}
			if newRef, ok := mapping[normalized]; ok {
				newVal = newRef
				break
			}
		}
		if newVal == "" {
			log.Printf("warning: no mapping for lemmaRef %q", old)
			skipped++
			continue
		}
		w.RemoveAttr("lemmaRef")
		w.CreateAttr("lemmaRef", newVal)
		rewrote++
	}
	return
}
