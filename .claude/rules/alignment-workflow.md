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

lemmaRef now resolves to the most specific inflected form when available:
- `#w.立つ.たて` — inflected form (preferred)
- `#w.立つ` — lemma entry (fallback when no inflected form matches)

Resolution is automatic during draft generation (layer 1: exact `kanjiReading`
match; layer 2: last-rune match). Do not manually construct inflected form IDs.

## Already-Annotated Poems

When `/align-poem N` is run on a poem that already has `<w>` elements in
`kokin-annotated.xml`, the draft is reconstructed directly from the XML:
- All segments will show `[✓]` (surfaces already match)
- Existing inflected form refs are preserved
- rdg rows show individual already-annotated tokens (not a placeholder)

This makes re-opening an annotated poem safe — the draft reflects the actual
state of the XML.

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
