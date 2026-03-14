package wordlist

import "testing"

func TestParseMSD(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  MSD
	}{
		{
			name:  "full msd with all fields",
			input: "UPosTag=NOUN|IPAPosTag=名詞-一般|UniDicPosTag=名詞-普通名詞-一般|LemmaReading=とし|Kanji=年|KanjiReading=とし|WLSPH=1.1630|WLSP=1.1630|WLSPDescription=体-関係-時間-年",
			want: MSD{
				UPosTag:         "NOUN",
				IPAPosTag:       "名詞-一般",
				UniDicPosTag:    "名詞-普通名詞-一般",
				LemmaReading:    "とし",
				Kanji:           "年",
				KanjiReading:    "とし",
				WLSPH:           "1.1630",
				WLSP:            "1.1630",
				WLSPDescription: "体-関係-時間-年",
			},
		},
		{
			name:  "particle with no WLSP/WLSPDescription",
			input: "UPosTag=ADP|IPAPosTag=助詞-格助詞-一般|UniDicPosTag=助詞-格助詞|LemmaReading=の|Kanji=の|KanjiReading=の|WLSPH=8.0061",
			want: MSD{
				UPosTag:      "ADP",
				IPAPosTag:    "助詞-格助詞-一般",
				UniDicPosTag: "助詞-格助詞",
				LemmaReading: "の",
				Kanji:        "の",
				KanjiReading: "の",
				WLSPH:        "8.0061",
			},
		},
		{
			name:  "auxiliary with WLSP but no WLSPDescription",
			input: "UPosTag=AUX|IPAPosTag=助動詞|UniDicPosTag=助動詞|LemmaReading=き|Kanji=し|KanjiReading=し|WLSPH=9.0010|WLSP=9.0010",
			want: MSD{
				UPosTag:      "AUX",
				IPAPosTag:    "助動詞",
				UniDicPosTag: "助動詞",
				LemmaReading: "き",
				Kanji:        "し",
				KanjiReading: "し",
				WLSPH:        "9.0010",
				WLSP:         "9.0010",
			},
		},
		{
			name:  "empty string",
			input: "",
			want:  MSD{},
		},
		{
			name:  "field without equals sign ignored",
			input: "UPosTag=VERB|badfield|LemmaReading=みる",
			want: MSD{
				UPosTag:      "VERB",
				LemmaReading: "みる",
			},
		},
		{
			name:  "value containing equals sign",
			input: "UPosTag=NOUN|WLSPDescription=体-関係=特殊",
			want: MSD{
				UPosTag:         "NOUN",
				WLSPDescription: "体-関係=特殊",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseMSD(tt.input)
			if got != tt.want {
				t.Errorf("ParseMSD(%q)\n  got  %+v\n  want %+v", tt.input, got, tt.want)
			}
		})
	}
}
