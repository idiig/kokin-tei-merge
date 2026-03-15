// Command merge embeds the Hachidaishu wordlist into a Karoku 2 TEI XML base
// file, producing a merged output with the dictionary and classification lists
// prepended to Karoku 2's existing <back> element.
//
// Usage:
//
//	go run ./cmd/merge -wordlist data/hachidaishu-wordlist.xml \
//	  -base data/Kokinwakashu_200003050_20240922.xml \
//	  -output data/kokin-merged.xml
//
// To also sync the <back> of an already-annotated file:
//
//	go run ./cmd/merge -wordlist data/hachidaishu-wordlist.xml \
//	  -base data/Kokinwakashu_200003050_20240922.xml \
//	  -output data/kokin-merged.xml \
//	  -sync data/kokin-annotated.xml
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
	syncPath := flag.String("sync", "", "path to already-annotated XML to sync <back> into (optional)")
	flag.Parse()

	if *wordlistPath == "" || *basePath == "" || *outputPath == "" {
		fmt.Fprintf(os.Stderr, "Usage: merge -wordlist <file> -base <file> -output <file> [-sync <file>]\n")
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

	log.Println("replacing back divs in base...")
	if err := merge.ReplaceBackDivs(baseDoc, divs); err != nil {
		log.Fatalf("error: %v", err)
	}

	log.Println("updating header...")
	merge.UpdateHeader(baseDoc)

	log.Printf("writing %s", *outputPath)
	if err := merge.WriteDocument(baseDoc, *outputPath); err != nil {
		log.Fatalf("error writing output: %v", err)
	}

	if *syncPath != "" {
		log.Printf("syncing <back> into %s", *syncPath)
		syncDoc, err := merge.ReadDocument(*syncPath)
		if err != nil {
			log.Fatalf("error reading sync target: %v", err)
		}
		if err := merge.ReplaceBackDivs(syncDoc, divs); err != nil {
			log.Fatalf("error replacing back in sync target: %v", err)
		}
		merge.UpdateHeader(syncDoc)
		if err := merge.WriteDocument(syncDoc, *syncPath); err != nil {
			log.Fatalf("error writing sync target: %v", err)
		}
		log.Printf("synced %s", *syncPath)
	}

	log.Println("done!")
}
