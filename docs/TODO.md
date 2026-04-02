# TODO: Post-processing Tasks

## Homograph Disambiguation

### なむ (namu) Homograph Resolution

**Issue:** The lemma `なむ` has two homographic entries in the wordlist:
1. `#なむ` — standalone word (complete form)
2. `#なむ.な-む` — combination of `ぬ` + `む` (two morphemes)

**Resolution Strategy:**
When encountering `なむ` tokens in post-processing, determine the correct homograph by examining the inflection of the preceding verb:

- **If the previous `<w>` element ends with あ-row (ア行) inflection:** Use `#なむ.な-む` (ぬ + む combination)
  - Example: 行か + なむ → 行かなむ (ika-namu)
  - Rationale: あ-row ending indicates 未然形 (mizenkei), which combines with ぬ + む
  
- **Otherwise (い-row/イ行 or other):** Use `#なむ` (standalone emphatic particle)
  - Example: 咲き + なむ → 咲きなむ (saki-namu)
  - Rationale: Non あ-row indicates different conjugation, standalone なむ particle

**Implementation:**
Create a post-processing script to:
1. Find all `<w lemmaRef="#なむ">` elements
2. Check the surface form (reading) of the preceding `<w>` element in the same `<seg>`
3. Determine if it ends with あ-row kana (か, が, さ, ざ, た, だ, な, は, ば, ぱ, ま, や, ら, わ, etc.)
4. Update `@lemmaRef` accordingly:
   - あ-row ending → `lemmaRef="#なむ.な-む"`
   - Other → `lemmaRef="#なむ"`

**Priority:** Medium (affects lexical accuracy)
