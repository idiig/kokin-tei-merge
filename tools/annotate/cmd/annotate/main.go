// Command annotate wraps each token in the Karoku 2 poem body with
// <w lemmaRef="…"> elements by aligning Hachidaishu token surfaces against
// Karoku 2 segment text using exact string matching.
//
// Usage:
//
//	go run ./cmd/annotate \
//	  -hachi ../../data/hachidaishu-wordlist.xml \
//	  -input ../../data/kokin-merged.xml \
//	  -output /tmp/kokin-annotated.xml \
//	  -report /tmp/unmatched.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	annotate "github.com/kokin-tei-merge/tools/annotate"
)

type report struct {
	Matched   int   `json:"matched"`
	Skipped   int   `json:"skipped"`
	Unmatched int   `json:"unmatched"`
	Poems     []int `json:"unmatched_poems"`
}

func main() {
	hachiPath := flag.String("hachi", "", "path to hachidaishu-wordlist.xml")
	inputPath := flag.String("input", "", "path to merged XML (input)")
	outputPath := flag.String("output", "", "path to annotated XML (output)")
	reportPath := flag.String("report", "unmatched.json", "path to write unmatched-poem JSON report")
	flag.Parse()

	if *hachiPath == "" || *inputPath == "" || *outputPath == "" {
		fmt.Fprintf(os.Stderr, "Usage: annotate -hachi <file> -input <file> -output <file> [-report <file>]\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	log.Printf("reading hachidaishu wordlist %s", *hachiPath)
	hachiDoc, err := annotate.ReadDocument(*hachiPath)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	log.Printf("reading merged document %s", *inputPath)
	mergedDoc, err := annotate.ReadDocument(*inputPath)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	log.Println("annotating…")
	matched, skipped, unmatched, unmatchedPoems := annotate.AnnotateDoc(hachiDoc, mergedDoc)
	log.Printf("matched=%d  skipped=%d  unmatched=%d", matched, skipped, unmatched)

	log.Printf("writing annotated output %s", *outputPath)
	if err := annotate.WriteDocument(mergedDoc, *outputPath); err != nil {
		log.Fatalf("error writing output: %v", err)
	}

	r := report{
		Matched:   matched,
		Skipped:   skipped,
		Unmatched: unmatched,
		Poems:     unmatchedPoems,
	}
	if unmatchedPoems == nil {
		r.Poems = []int{}
	}

	f, err := os.Create(*reportPath)
	if err != nil {
		log.Fatalf("error creating report: %v", err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(r); err != nil {
		log.Fatalf("error writing report: %v", err)
	}
	log.Printf("report written to %s", *reportPath)
	log.Println("done!")
}
