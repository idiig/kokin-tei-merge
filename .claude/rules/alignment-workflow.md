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
年	#w.年	# ✓
の	#w.の.h1	# ✓

# — seg 3 [✗  need: ひとゝせを  got: 一とせを]
一とせ	#w.一年	# ? check surface   ← change left column to ひとゝせ
を	#w.を.h1	# ✓
```

Rules:
- Edit only the **left column** (surface) to match Karoku text exactly.
- Blank lines separate segments — one group per segment.
- Delete all non-comment lines to skip the poem.

## Key Constraint

Surfaces in each group must concatenate exactly to the Karoku segment text.
The lemmaRef always points to the Hachidaishu dictionary entry — do not change it.

## Environment

Run inside `nix develop` so that `go`, `hx`, and `tmux` are on PATH.
Output file: `data/kokin-annotated.xml` (persisted in repo, not `/tmp`).
