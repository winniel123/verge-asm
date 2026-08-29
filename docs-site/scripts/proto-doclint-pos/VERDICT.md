# Verdict — wayfinder #792 (POS-rule precision)

Throwaway prototype. Branch `proto/792-pos-precision`. Do not ship.

## Question

Do the two POS-dependent doc-lint rules reach acceptable precision on real
verge-asm docs, using the `pos`/FastTag tagger that `retext-pos` wraps?

- Rule C — noun-cluster cap (flag a run of 4 or more stacked nouns).
- Rule D — simple-tense (flag `have`/`has`/`had` before a past participle).

## Method

The prototype extracts prose from the markdown AST. It drops code, inline code,
tables, front-matter, and blockquotes. It then tags each sentence with the `pos`
tagger and applies both rules. It prints every flag with the sentence and the
per-word POS tags. A human scored the flags by eye. Corpus: all 176 files across
the 9 in-scope families.

Caveat on the corpus: these docs were rewritten to the style standard recently.
So the corpus measures the false-positive rate on clean text well. It cannot
measure recall, because the violations were already removed.

## Answer

### Rule D — simple-tense: SHIP as an advisory WARNING, not a hard error.

- Precision as a detector is about 100%. The run produced 634 flags across roughly
  180 distinct participle forms. The scored sample found zero false positives.
- The `have`/`has`/`had` anchor plus a VBN participle is reliable. The grammatical
  position after `have`/`has`/`had` forces the participle tag, so the tagger does
  not confuse it.
- Style standard §4.1 rule 4 gives this rule a judgment residue. A compound form
  may stay when it carries current relevance or a hedge. So the tool must not fail
  a build on it. It must flag it for a human to read.
- Volume is high on clean docs (634 flags). Many are legitimate current-relevance
  uses (for example "nobody has measured", "has been GA since 2025"). A hard error
  would be wrong. A warning is right.

### Rule C — noun-cluster cap: DEFER. Does not reach acceptable precision.

- Even after stripping GFM tables and breaking noun runs on punctuation, the run
  produced 410 flags (down from 4184 raw). True positives were near zero on these
  clean docs.
- Every scored survivor was one of three noise types:
  1. A verb tagged as a noun ("stores", "counts", "disables", "issues", "lists",
     "costs", "ships", "rules").
  2. A number tagged as a noun ("four", "one", "six", "five", "three", "second").
  3. A proper product name a human exempts ("Microsoft Entra admin centre").
- The `pos` tagger cannot separate a noun from a verb for a word that ends in "-s".
  It also tags numerals as nouns. The per-token error compounds across the 4-token
  window. This matches the #791 research hedge exactly.
- Recommendation: do not ship Rule C in v1, not even as a warning. The
  signal-to-noise ratio is too low. Keep it as a §4.1 human-review gate only.
  Revisit if a better tagger becomes available (see #791: `en-pos` claims higher
  token accuracy but is stale and needs custom wiring).

## Tooling finding for the SPEC

The retext prose stack duplicates words at the current package versions. The
tokenizer turns "a b c" into "a b b c c" and "one, two, three" into
"one two two two three three three". Both the `remark-retext` bridge and a bare
`retext-english` pipeline show it. So the SPEC must not assume the retext bridge
works out of the box. Options: pin a working `parse-latin`/`parse-english`
version, use the `pos` tagger with a direct lexer (this prototype's path), or add
a de-duplication guard. The precision numbers above are unaffected, because the
prototype used the `pos` tagger directly.

## Impact on the map

- Updates Q8. Q8 gated both rules behind this prototype. Rule D promotes to a
  v1 warning. Rule C defers out of v1.
- Feeds #793 (write the SPEC): specify Rule D as a warning with a documented
  judgment residue, drop Rule C from v1, and record the tokenizer caveat.
