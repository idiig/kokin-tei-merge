// Command merge embeds the Hachidaishu wordlist into a Karoku 2 TEI XML base
// file, producing a merged output with the dictionary and classification lists
// prepended to Karoku 2's existing <back> element.
//
// Usage:
//
//	go run ./cmd/merge -wordlist data/hachidaishu-wordlist.xml \
//	  -base data/Kokinwakashu_200003050_20240922.xml \
//	  -output data/kokin-merged.xml
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	merge "github.com/kokin-tei-merge/tools/merge"
)

func main() {
	wordlistPath := flag.String("wordlist", "", "path to hachidaishu-wordlist.xml")
	basePath := flag.String("base", "", "path to Karoku 2 base XML file")
	outputPath := flag.String("output", "", "path to output merged XML file")
	flag.Parse()

	if *wordlistPath == "" || *basePath == "" || *outputPath == "" {
		fmt.Fprintf(os.Stderr, "Usage: merge -wordlist <file> -base <file> -output <file>\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	log.Printf("reading wordlist %s", *wordlistPath)
	wordlistDoc, err := merge.ReadDocument(*wordlistPath)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	log.Println("extracting back divs from wordlist...")
	divs, err := merge.ExtractBackDivs(wordlistDoc)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	log.Printf("found %d divs to embed", len(divs))

	log.Printf("reading base %s", *basePath)
	baseDoc, err := merge.ReadDocument(*basePath)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	log.Println("prepending divs to base <back>...")
	if err := merge.PrependToBack(baseDoc, divs); err != nil {
		log.Fatalf("error: %v", err)
	}

	log.Println("updating header...")
	merge.UpdateHeader(baseDoc)

	log.Printf("writing %s", *outputPath)
	if err := merge.WriteDocument(baseDoc, *outputPath); err != nil {
		log.Fatalf("error writing output: %v", err)
	}

	log.Println("done!")
}
