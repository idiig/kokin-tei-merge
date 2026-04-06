# TODO

## Annotation Issues

### 覧 (ramu) tokenization

`覧` sometimes breaks verb-r-a+mu (e.g., `見らむ` = `見` + `ら` + `む`), rather than treating `らむ` as a whole auxiliary verb token. This would not be processed correctly in the current alignment proposal.

**Impact**: The current align-review tool expects tokens to match the Hachidaishu wordlist segmentation. When Karoku segments differently (splitting what Hachidaishu treats as a single token), manual intervention is required.

**Example cases to investigate**:
- `見らむ` vs `見` + `らむ`
- Other verbs with `-ramu` ending

**Possible solutions**:
1. Pre-process Karoku text to merge certain multi-character sequences before alignment
2. Add alignment rules to handle split-vs-merged token mismatches
3. Document as known limitation requiring manual review
