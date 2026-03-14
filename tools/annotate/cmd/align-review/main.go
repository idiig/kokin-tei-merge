// Command align-review has two subcommands:
//
//	prepare -poem N   — write draft to /tmp/kokin-align-N.txt and open Helix
//	apply   -poem N   — parse /tmp/kokin-align-N.txt and write <w> into XML
//
// Typical workflow:
//
//	go run ./cmd/align-review prepare -hachi … -input … -poem 1
//	# edit in Helix, :wq when done
//	go run ./cmd/align-review apply -hachi … -input … -output … -poem 1
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"

	annotate "github.com/kokin-tei-merge/tools/annotate"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: align-review <prepare|apply> [flags]\n")
		os.Exit(1)
	}
	switch os.Args[1] {
	case "prepare":
		runPrepare(os.Args[2:])
	case "apply":
		runApply(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand %q. Use prepare or apply.\n", os.Args[1])
		os.Exit(1)
	}
}

// draftPath returns the well-known path for a poem's draft file.
func draftPath(n int) string {
	return fmt.Sprintf("/tmp/kokin-align-%d.txt", n)
}

// openInTmux opens the draft file in a new tmux window.
func openInTmux(n int, path string) {
	cmd := exec.Command("tmux", "new-window", "-n",
		fmt.Sprintf("align-%d", n),
		fmt.Sprintf("hx %s", path),
	)
	if err := cmd.Run(); err != nil {
		fmt.Printf("Could not open tmux window. Open manually:\n\n  hx %s\n\n", path)
	} else {
		fmt.Printf("Helix opened in tmux window (align-%d).\n", n)
	}
}

// injectError prepends an error comment block to the draft file.
func injectError(path string, err error) {
	content, readErr := os.ReadFile(path)
	if readErr != nil {
		return
	}
	prefix := fmt.Sprintf("# ERROR: %v\n# Fix the line(s) above and save again (:wq).\n#\n", err)
	_ = os.WriteFile(path, append([]byte(prefix), content...), 0644)
}

// runPrepare writes the draft to /tmp/kokin-align-N.txt and opens it in Helix.
func runPrepare(args []string) {
	fs := flag.NewFlagSet("prepare", flag.ExitOnError)
	hachiPath := fs.String("hachi", "", "path to hachidaishu-wordlist.xml")
	inputPath := fs.String("input", "", "path to annotated XML")
	poemN := fs.Int("poem", 0, "poem number to prepare")
	fs.Parse(args)

	if *hachiPath == "" || *inputPath == "" || *poemN == 0 {
		fmt.Fprintf(os.Stderr, "Usage: align-review prepare -hachi <file> -input <file> -poem N\n")
		fs.PrintDefaults()
		os.Exit(1)
	}

	hachiDoc, err := annotate.ReadDocument(*hachiPath)
	if err != nil {
		log.Fatalf("reading hachi: %v", err)
	}
	mergedDoc, err := annotate.ReadDocument(*inputPath)
	if err != nil {
		log.Fatalf("reading input: %v", err)
	}

	tokens := annotate.HachiTokens(hachiDoc, *poemN)
	if tokens == nil {
		log.Fatalf("poem %d not found in hachidaishu", *poemN)
	}
	lElem := mergedDoc.FindElement(fmt.Sprintf("//body//l[@n='%d']", *poemN))
	if lElem == nil {
		log.Fatalf("poem %d not found in merged XML", *poemN)
	}
	segs := lElem.SelectElements("seg")
	segTexts := make([]string, len(segs))
	splits := make([]int, len(segs))
	for i, seg := range segs {
		segTexts[i] = seg.Text()
		ws := seg.SelectElements("w")
		if segTexts[i] == "" && len(ws) > 0 {
			// Already annotated: reconstruct text and splits from existing <w>.
			var concat string
			for _, w := range ws {
				concat += w.Text()
			}
			segTexts[i] = concat
			splits[i] = len(ws)
		}
	}
	// If splits are all zero (not yet annotated), drop the hint.
	total := 0
	for _, s := range splits {
		total += s
	}
	if total == 0 {
		splits = nil
	}

	draft := annotate.GenerateDraft(*poemN, tokens, segTexts, splits)
	path := draftPath(*poemN)
	if err := os.WriteFile(path, []byte(draft), 0644); err != nil {
		log.Fatalf("writing draft: %v", err)
	}

	openInTmux(*poemN, path)
	fmt.Printf("Run /apply-poem %d when done.\n", *poemN)
}

// runApply reads /tmp/kokin-align-N.txt, validates, and writes <w> into the XML.
// On validation failure, it injects the error into the draft and reopens Helix.
func runApply(args []string) {
	fs := flag.NewFlagSet("apply", flag.ExitOnError)
	hachiPath := fs.String("hachi", "", "path to hachidaishu-wordlist.xml")
	inputPath := fs.String("input", "", "path to annotated XML (input)")
	outputPath := fs.String("output", "", "path to annotated XML (output)")
	poemN := fs.Int("poem", 0, "poem number to apply")
	fs.Parse(args)

	if *hachiPath == "" || *inputPath == "" || *outputPath == "" || *poemN == 0 {
		fmt.Fprintf(os.Stderr, "Usage: align-review apply -hachi <file> -input <file> -output <file> -poem N\n")
		fs.PrintDefaults()
		os.Exit(1)
	}

	mergedDoc, err := annotate.ReadDocument(*inputPath)
	if err != nil {
		log.Fatalf("reading input: %v", err)
	}

	lElem := mergedDoc.FindElement(fmt.Sprintf("//body//l[@n='%d']", *poemN))
	if lElem == nil {
		log.Fatalf("poem %d not found in merged XML", *poemN)
	}
	segs := lElem.SelectElements("seg")
	segTexts := make([]string, len(segs))
	for i, seg := range segs {
		segTexts[i] = seg.Text()
		if segTexts[i] == "" {
			// Reconstruct from existing <w> children.
			var concat string
			for _, w := range seg.SelectElements("w") {
				concat += w.Text()
			}
			segTexts[i] = concat
		}
	}

	path := draftPath(*poemN)
	content, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("reading draft %s: %v (run prepare first)", path, err)
	}

	aligned, parseErr := annotate.ParseDraft(string(content), segTexts)
	if parseErr != nil {
		// Inject error into draft and reopen Helix.
		injectError(path, parseErr)
		openInTmux(*poemN, path)
		fmt.Printf("Validation failed. Fix the error in Helix, then run /apply-poem %d again.\n", *poemN)
		os.Exit(1)
	}
	if aligned == nil {
		log.Printf("poem %d: skipped (no data lines in draft)", *poemN)
		return
	}

	annotate.ApplyAlignment(segs, aligned)
	log.Printf("poem %d: aligned (%d segments)", *poemN, len(segs))

	if err := annotate.WriteDocument(mergedDoc, *outputPath); err != nil {
		log.Fatalf("writing output: %v", err)
	}
	log.Printf("written to %s", *outputPath)
}
