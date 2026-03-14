// Package wordlist extracts a TEI dictionary from inline morphological annotations.
package wordlist

import "strings"

// MSD holds the parsed fields from a pipe-delimited msd attribute string.
// Example input: "UPosTag=NOUN|IPAPosTag=名詞-一般|UniDicPosTag=名詞-普通名詞-一般|LemmaReading=とし|Kanji=年|KanjiReading=とし|WLSPH=1.1630|WLSP=1.1630|WLSPDescription=体-関係-時間-年"
type MSD struct {
	UPosTag         string
	IPAPosTag       string
	UniDicPosTag    string
	LemmaReading    string
	Kanji           string
	KanjiReading    string
	WLSPH           string
	WLSP            string
	WLSPDescription string
}

// ParseMSD parses a pipe-delimited msd string into an MSD struct.
// Unknown keys are silently ignored.
func ParseMSD(s string) MSD {
	var m MSD
	if s == "" {
		return m
	}
	for _, field := range strings.Split(s, "|") {
		k, v, ok := strings.Cut(field, "=")
		if !ok {
			continue
		}
		switch k {
		case "UPosTag":
			m.UPosTag = v
		case "IPAPosTag":
			m.IPAPosTag = v
		case "UniDicPosTag":
			m.UniDicPosTag = v
		case "LemmaReading":
			m.LemmaReading = v
		case "Kanji":
			m.Kanji = v
		case "KanjiReading":
			m.KanjiReading = v
		case "WLSPH":
			m.WLSPH = v
		case "WLSP":
			m.WLSP = v
		case "WLSPDescription":
			m.WLSPDescription = v
		}
	}
	return m
}
