// Command wordlist extracts a TEI dictionary from inline <w> annotations
// in the Hachidaishu TEI XML, placing it in <back> and adding lemmaRef
// links from body tokens.
//
// Usage:
//
//	go run ./cmd/wordlist -input data/hachidaishu-patched.xml -output data/hachidaishu-wordlist.xml -wlsph data/wlsph.json
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/beevik/etree"
	wordlist "github.com/kokin-tei-merge/tools/wordlist"
)

func main() {
	input := flag.String("input", "", "path to input TEI XML file")
	output := flag.String("output", "", "path to output TEI XML file")
	wlsphPath := flag.String("wlsph", "", "path to wlsph.json classification file")
	flag.Parse()

	if *input == "" || *output == "" {
		fmt.Fprintf(os.Stderr, "Usage: wordlist -input <file> -output <file> [-wlsph <file>]\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	log.Printf("reading %s", *input)
	doc, err := wordlist.ReadDocument(*input)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	log.Println("extracting tokens...")
	tokens, compounds, modernRefs := wordlist.ExtractTokens(doc)
	log.Printf("found %d tokens", len(tokens))

	log.Println("building entries...")
	entries := wordlist.BuildEntries(tokens)
	wordlist.MarkCompounds(entries, compounds, modernRefs)
	log.Printf("built %d entries (%d compounds)", len(entries), countCompounds(entries))

	log.Println("building pron entries...")
	pronEntries := wordlist.BuildPronEntries(tokens)
	log.Printf("built %d pron entries", len(pronEntries))

	log.Println("flattening <app> blocks...")
	flatCount := wordlist.FlattenApps(doc)
	log.Printf("flattened %d <app> blocks", flatCount)

	log.Println("transforming body...")
	wordlist.TransformBody(doc, pronEntries)

	// Build classification lists.
	usedWLSPH := wordlist.WLSPHCodesFromTokens(tokens)

	var classWLSPH, classWLSP *etree.Element
	if *wlsphPath != "" {
		log.Printf("loading WLSPH from %s", *wlsphPath)
		cats, err := wordlist.LoadWLSPH(*wlsphPath)
		if err != nil {
			log.Fatalf("error loading WLSPH: %v", err)
		}
		wlsphItems := wordlist.BuildWLSPHItems(cats)
		wlsphItems = wordlist.AddMissingWLSPH(wlsphItems, usedWLSPH)
		classWLSPH = wordlist.BuildClassificationDiv("classWLSPH", "分類語彙表 (WLSPH)", wlsphItems)
		log.Printf("WLSPH: %d items (%d from JSON, rest from data)", len(wlsphItems), len(cats))
	}

	wlspCodes := wordlist.ExtractWLSPCodes(tokens)
	if len(wlspCodes) > 0 {
		wlspItems := wordlist.BuildWLSPItems(wlspCodes)
		classWLSP = wordlist.BuildClassificationDiv("classWLSP", "分類語彙表 (WLSP)", wlspItems)
		log.Printf("extracted %d WLSP codes", len(wlspCodes))
	}

	log.Println("building dictionary...")
	back := wordlist.BuildBackDiv(entries, pronEntries, classWLSPH, classWLSP)
	wordlist.InsertBack(doc, back)

	log.Println("updating header...")
	wordlist.UpdateHeader(doc)

	log.Printf("writing %s", *output)
	if err := wordlist.WriteDocument(doc, *output); err != nil {
		log.Fatalf("error writing output: %v", err)
	}

	log.Println("done!")
}

func countCompounds(entries []*wordlist.Entry) int {
	n := 0
	for _, e := range entries {
		if e.IsCompound {
			n++
		}
	}
	return n
}
