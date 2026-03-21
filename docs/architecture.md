# Architecture

## Overview

This project merges two TEI XML sources for the Kokinwakashu:

| Source | Content | File |
|---|---|---|
| Hachidaishu | Lexical/morphological annotations | `data/hachidaishu-patched.xml` |
| Karoku 2 | Bibliographic transcription (manuscript) | `data/Kokinwakashu_200003050_20240922.xml` |

The output is `data/kokin-annotated.xml` — the Karoku 2 text with `<w lemmaRef>` annotations
derived from Hachidaishu.

## Data Pipeline

```
hachidaishu-patched.xml
        │
        ▼
tools/annotate/cmd/extract-wordlist   →  data/hachidaishu-wordlist.xml
        │                                  (Dict A + Dict B in <back>)
        ▼
tools/merge/cmd/merge                 →  data/kokin-merged.xml
  + Kokinwakashu_200003050_20240922.xml    (Karoku base + Hachidaishu <back>)
        │
        ▼
tools/annotate/cmd/annotate           →  data/kokin-annotated.xml  (partial)
  (rule-based, exact match)               unmatched.json
        │
        ▼
tools/annotate/cmd/align-review       →  data/kokin-annotated.xml  (complete)
  (interactive, Helix-based)
        │
        ▼
tools/migrate/cmd/fixrefs             →  data/kokin-annotated.xml  (corrected)
  (bulk lemmaRef correction)
```

## Two-Layer Dictionary

`hachidaishu-wordlist.xml` `<back>` contains two indexes:

### Dict A — Reading Index (`<div type="reading-index">`)

Keyed by **KanjiReading** (actual surface kana of each token). Each entry has
one or more `<hom>` elements with `xml:id="reading.lemma"`.

```xml
<entry>
  <form><orth>われ</orth></form>
  <hom xml:id="われ.我">...</hom>
</entry>
```

`lemmaRef` in `kokin-annotated.xml` always points here: `#reading.lemma`.

### Dict B — Dictionary (`<div type="dictionary">`)

Keyed by **lemma** (base form). Contains `gramGrp` and `sense` for each entry.
Dict A hom elements reference Dict B via `corresp`.

## lemmaRef Resolution

When generating a draft alignment, `HachiTokens()` reads the wordlist body
`<w>` elements. Each `<w>` already carries the correct `lemmaRef="#reading.lemma"`.

`fixrefs` corrects lemmaRefs that were set incorrectly during migration:

| Layer | Method | Example |
|---|---|---|
| 0 | Kanji surface → wordlist body map | `我` → `#われ.我` |
| 1 | Exact `(surface, lemma)` → Dict A | `かかれ` → `#かかれ.掛かる` |
| 2 | Kana variant `(surface′, lemma)` → Dict A | `わひ` → `#わび.侘ぶ` (清濁), `おら` → `#をら.折る` (歴史的仮名), `かゝれ` → `#かかれ.掛かる` (ゝ展開) |

## Tools

### `tools/annotate`

Go package + CLI tools for token alignment.

- `annotate.go` — core functions: `HachiTokens`, `AlignPoem`, `ApplyAlignment`,
  `RefineTokenRefs`, `buildDictAMap`, `kanaVariants`, `expandIterationMarks`
- `review.go` — draft file generation/parsing: `GenerateDraft`, `ParseDraft`,
  `ParseDraftGroups`
- `cmd/annotate` — batch annotator (rule-based pass)
- `cmd/align-review` — interactive alignment (`prepare` + `apply` subcommands)

### `tools/merge`

Embeds Hachidaishu `<back>` into Karoku 2 base XML.

- `cmd/merge` — produces `kokin-merged.xml`; `-sync` flag updates `<back>` in
  an existing annotated file without touching the body

### `tools/migrate`

One-off migration / correction tools.

- `cmd/fixrefs` — bulk-corrects `lemmaRef` values using the 3-layer lookup
  described above. Safe to re-run; uses `CanonicalAttrVal` but no re-indent
  to avoid whitespace changes in mixed-content elements.

### `tools/query`

Side-by-side poem viewer (Hachidaishu vs Karoku 2).
