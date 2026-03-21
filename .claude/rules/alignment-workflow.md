# Alignment Workflow

## Overview

Token alignment is a two-pass process:

1. **Rule-based pass** (`tools/annotate/cmd/annotate`) — exact string match;
   unmatched poems written to `unmatched.json`.
2. **Interactive pass** (`tools/annotate/cmd/align-review`) — Helix-based
   manual alignment for unmatched poems.

## Slash Commands

| Command | Action |
|---|---|
| `/align-poem N` | Generate draft for poem N, open Helix in tmux window `align-N` |
| `/apply-poem N` | Parse draft, validate, write `<w>` into `data/kokin-annotated.xml` |

Before executing `/apply-poem N`, always ask:
> "Apply alignment for poem N to kokin-annotated.xml? (y/n)"
Proceed only on explicit confirmation. On validation failure, `apply` injects
the error into the draft and reopens Helix automatically — user fixes and runs
`/apply-poem N` again.

After a successful `/apply-poem N`, ask:
> "Open the next poem for alignment? If so, which poem number?"

## Draft File Format

Draft files live at `/tmp/kokin-align-N.txt`. Format:

```
# Columns: surface (edit this) | lemmaRef (do not edit)
# — seg 1 [✓]
年	#w.年.とし	# ✓
の	#w.の.h1	# ✓

# — seg 2 [✓]
# app: lem=見らむ | rdg=みえん
花	#w.花	# ✓
とや	#w.とや	# ✓
見らむ	#w.見る.みら	# ✓
rdg	みえ	#w.見ゆ.みえ
rdg	ん	#w.む.む

# — seg 3 [✗  need: ひとゝせを  got: 一とせを]
一とせ	#w.一年	# ? check surface   ← change left column to ひとゝせ
を	#w.を.h1	# ✓
```

Rules:
- Edit only the **left column** (surface) to match Karoku text exactly.
- Blank lines separate segments — one group per segment.
- Delete all non-comment lines to skip the poem.
- `rdg` rows (three columns: `rdg TAB surface TAB lemmaRef`) must appear after
  all lem rows in their group. Surfaces must concatenate to the `<rdg>` text.

## lemmaRef Format

lemmaRef points to a Dict A hom in the two-layer dictionary structure:

```
#reading.lemma
```

Examples:
- `#われ.我` — `われ` is the KanjiReading (surface kana), `我` is the lemma
- `#かかれ.掛かる` — inflected form (KanjiReading of the actual token)
- `#に.ぬ` — auxiliary `ぬ` read as `に` (already-inflected form)

The reading part is the Hachidaishu KanjiReading for that token, not the
dictionary base form. This is set automatically during draft generation via
Dict A lookup — do not manually construct IDs.

**Multiple refs** are supported as a space-separated list in the second column:

```
つれ	#つれ.つ #に.ぬ	# two refs for disambiguation
```

The list is written verbatim into `lemmaRef="…"` in the XML.

## Already-Annotated Poems

When `/align-poem N` is run on a poem that already has `<w>` elements,
`prepare` uses a two-step strategy:

1. **Approach A** — Try re-alignment with fresh Hachidaishu tokens. If
   `AlignPoem` succeeds, lemmaRefs are refreshed from the wordlist (correct
   KanjiReading-based IDs).
2. **Approach B** — If alignment fails (orthographic divergence), keep the
   existing annotation and run `RefineTokenRefs` to fix kana mismatches
   (清濁の差, 歴史的仮名遣い, ゝ/ゞ expansion).

Re-opening an annotated poem is safe — the draft shows the current XML state
and all segments will show `[✓]` if surfaces still match.

## Key Constraint

Surfaces in each group must concatenate exactly to the Karoku segment text.
The lemmaRef points to the most specific available entry — do not change it.

## Wordlist Update Workflow

After regenerating `hachidaishu-wordlist.xml` (e.g., adding inflected forms),
sync the `<back>` into both merged and annotated files:

```bash
cd tools/merge && go run ./cmd/merge \
  -wordlist ../../data/hachidaishu-wordlist.xml \
  -base ../../data/Kokinwakashu_200003050_20240922.xml \
  -output ../../data/kokin-merged.xml \
  -sync ../../data/kokin-annotated.xml
```

## Environment

Run inside `nix develop` so that `go`, `hx`, and `tmux` are on PATH.
Output file: `data/kokin-annotated.xml` (persisted in repo, not `/tmp`).
