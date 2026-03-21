// Command fixrefs updates lemmaRef attributes in kokin-annotated.xml to use
// the surface-kana-based Dict A hom ID instead of the lemma-reading-based ID
// produced by the original migration.
//
// For each <w>surface</w> in the body, if the surface is a kana string that
// matches a Dict A entry key (reading), the lemmaRef is updated to
// #surface.lemma, replacing the old #lemmareading.lemma.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/beevik/etree"
)

// voicedVariants returns surface candidates with Karoku unvoiced kana replaced
// by the voiced counterpart used in Hachidaishu KanjiReadings.
var voicedPairs = [][2]rune{
	{'か', 'が'}, {'き', 'ぎ'}, {'く', 'ぐ'}, {'け', 'げ'}, {'こ', 'ご'},
	{'さ', 'ざ'}, {'し', 'じ'}, {'す', 'ず'}, {'せ', 'ぜ'}, {'そ', 'ぞ'},
	{'た', 'だ'}, {'ち', 'ぢ'}, {'つ', 'づ'}, {'て', 'で'}, {'と', 'ど'},
	{'は', 'ば'}, {'ひ', 'び'}, {'ふ', 'ぶ'}, {'へ', 'べ'}, {'ほ', 'ぼ'},
}

func voicedVariants(s string) []string {
	runes := []rune(s)
	var variants []string
	for _, pair := range voicedPairs {
		for i, r := range runes {
			if r == pair[0] {
				v := make([]rune, len(runes))
				copy(v, runes)
				v[i] = pair[1]
				variants = append(variants, string(v))
			}
		}
	}
	return variants
}

func main() {
	wordlist := flag.String("wordlist", "", "path to hachidaishu-wordlist.xml")
	input    := flag.String("input",    "", "path to kokin-annotated.xml (input)")
	output   := flag.String("output",   "", "path to output file (may be same as input)")
	flag.Parse()

	if *wordlist == "" || *input == "" || *output == "" {
		fmt.Fprintf(os.Stderr, "Usage: fixrefs -wordlist <file> -input <file> -output <file>\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	log.Printf("reading wordlist %s", *wordlist)
	wl := etree.NewDocument()
	wl.ReadSettings.PreserveCData = true
	if err := wl.ReadFromFile(*wordlist); err != nil {
		log.Fatalf("reading wordlist: %v", err)
	}

	// Build Dict A map: (reading, lemma) → hom xml:id
	type key struct{ reading, lemma string }
	dictA := make(map[key]string)
	for _, hom := range wl.FindElements("//back//div[@type='reading-index']/entry/hom") {
		homID := hom.SelectAttrValue("xml:id", "")
		if homID == "" {
			continue
		}
		dot := strings.Index(homID, ".")
		if dot < 0 {
			continue
		}
		dictA[key{homID[:dot], homID[dot+1:]}] = homID
	}
	log.Printf("Dict A: %d homs loaded", len(dictA))

	log.Printf("reading annotated %s", *input)
	ann := etree.NewDocument()
	ann.ReadSettings.PreserveCData = true
	if err := ann.ReadFromFile(*input); err != nil {
		log.Fatalf("reading input: %v", err)
	}

	updated, kept, notfound := 0, 0, 0
	for _, w := range ann.FindElements("//body//w") {
		ref := w.SelectAttrValue("lemmaRef", "")
		if ref == "" {
			continue
		}
		homID := strings.TrimPrefix(ref, "#")
		dot := strings.Index(homID, ".")
		if dot < 0 {
			kept++
			continue
		}
		currentReading := homID[:dot]
		lemma          := homID[dot+1:]
		surface        := w.Text()
		// Only fix when surface differs from current reading (otherwise already correct).
		if surface == "" || surface == currentReading {
			kept++
			continue
		}
		// Layer 1: exact surface lookup.
		if newID, ok := dictA[key{surface, lemma}]; ok {
			if newID != homID {
				w.RemoveAttr("lemmaRef")
				w.CreateAttr("lemmaRef", "#"+newID)
				updated++
			} else {
				kept++
			}
			continue
		}
		// Layer 2: voiced substitution (Karoku清音→Hachidaishu濁音).
		// e.g. はふき→はぶき, わひ→わび
		found := false
		for _, variant := range voicedVariants(surface) {
			if newID, ok := dictA[key{variant, lemma}]; ok {
				w.RemoveAttr("lemmaRef")
				w.CreateAttr("lemmaRef", "#"+newID)
				updated++
				found = true
				break
			}
		}
		if !found {
			notfound++
		}
	}
	log.Printf("updated=%d  kept=%d  not-found=%d", updated, kept, notfound)

	log.Printf("writing %s", *output)
	ann.Indent(2)
	ann.WriteSettings.CanonicalAttrVal = true
	ann.WriteSettings.CanonicalText = true
	if err := ann.WriteToFile(*output); err != nil {
		log.Fatalf("writing output: %v", err)
	}
	log.Println("done!")
}
