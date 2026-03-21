# kokin-tei-merge

## Project Overview

Merging two TEI XML sources for the Kokinwakashu (古今和歌集): a lexically annotated source (Hachidaishu dataset) and a bibliographically annotated source (Karoku 2 manuscript).

## Language Policy

- **Conversation**: Japanese
- **Documentation (docs/, README, CLAUDE.md)**: English
- **Code and commit messages**: English
- **XML content**: Japanese (original text)

## Data Sources

- `data/hachidaishu-patched.xml` — Lexical/morphological annotations (subset of Hachidaishu)
- `data/Kokinwakashu_200003050_20240922.xml` — Bibliographic annotations (Karoku 2 manuscript transcription)

## Tech Stack

- **Data format**: TEI XML (P5)
- **Programming language**: Go
- **Editor**: Helix (`hx`) — used in interactive alignment workflow
- **Environment**: Nix flake (`nix develop`); `go`, `hx`, `tmux` provided by devShell
- **LLM assistance**: Aligning lexical annotations between the two sources

## Project Structure

```
data/
  hachidaishu-patched.xml              — Hachidaishu lexical annotations (source)
  Kokinwakashu_200003050_20240922.xml  — Karoku 2 manuscript (source)
  hachidaishu-wordlist.xml             — Extracted wordlist
  kokin-merged.xml                     — Merged TEI (Hachidaishu dict in <back>)
  kokin-annotated.xml                  — Annotated output (<w lemmaRef> in body)

tools/
  merge/      — Embeds Hachidaishu <back> into Karoku 2 base XML
  query/      — Side-by-side poem viewer (Hachidaishu vs Karoku 2)
  annotate/   — Rule-based token alignment + interactive Helix review
    cmd/annotate/      — Batch annotator CLI
    cmd/align-review/  — Interactive alignment: prepare + apply subcommands
  migrate/    — One-off data correction tools
    cmd/fixrefs/       — Bulk-corrects lemmaRef values (3-layer kana lookup)

.claude/
  commands/   — Slash commands: /align-poem N, /apply-poem N
  rules/      — Technical rules (git-workflow, alignment-workflow)
docs/         — Project documentation (English)
```

## Session Initialization

At the start of each conversation, run `nix-shell` in the project root to load the environment. All tool commands must be run inside the nix shell (prefix with `nix develop --command bash -c "..."`).

## lemmaRef Convention

`lemmaRef` values use the two-layer Dict A format: `#reading.lemma`
(e.g. `#われ.我`, `#かかれ.掛かる`). The reading is the Hachidaishu
KanjiReading for that token — not the dictionary base form.

After any bulk re-annotation, run `fixrefs` to correct systematic errors:

```bash
cd tools/migrate && go run ./cmd/fixrefs \
  -wordlist ../../data/hachidaishu-wordlist.xml \
  -input  ../../data/kokin-annotated.xml \
  -output ../../data/kokin-annotated.xml
```

## Important Constraints

- Do not modify original text content in XML; only add structural annotations
- Preserve TEI namespace (`http://www.tei-c.org/ns/1.0`) in all XML processing
- Validate against TEI P5 schema after modifications
- Do not call `Indent()` on documents with mixed content (text + elements) —
  it inserts spurious blank lines around text nodes

## Claude Code Self-Maintenance

When general rules, common workflows, or reusable skills emerge during a conversation, update the relevant CLAUDE configuration:

- **Project-level instructions** → `CLAUDE.md` (language policy, constraints, conventions)
- **Technical rules** → `.claude/rules/<topic>.md` (git workflow, coding style, XML processing)
- **Cross-project learnings** → `~/.claude/projects/.../memory/MEMORY.md` (auto memory)
