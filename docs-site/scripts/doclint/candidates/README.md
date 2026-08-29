# doclint candidate detectors (deferred)

These are **candidate** detectors, not enabled rules. None is in the `RULES` set, so the doclint
tool never runs one. They are the record of the #824 attempt: the prove-before-enable process
(SPEC §2.5) built each detector, proved it on a fixture corpus (SPEC §6), then measured it on the
real in-scope docs (the #792/#823 method) to score precision, then made the enable-or-defer call.

Both #824 candidates **deferred**. The measured result is on the #824 ticket and in SPEC §7.1.
These files stay so the attempt is reproducible and a later effort does not re-litigate it.

## Files

| File | What it is |
| --- | --- |
| `one-instruction.mjs` | The one-instruction-per-sentence detector. Deferred: ~half precision. |
| `no-ellipsis.mjs` | The no-ellipsis (dropped verb) detector. Deferred: floods the corpus. |
| `tagging.mjs` | Shared sentence split and `pos` tagging, reused from simple-tenses. |
| `measure.mjs` | The measurement harness. Runs each candidate on the real corpus. |

## Reproduce the measurement

Run from the `docs-site/` directory:

```
node scripts/doclint/candidates/measure.mjs
```

It prints, per candidate, the fixture-corpus result and the real-doc flag count with an
evenly-spaced sample for a manual precision score.

## Why each deferred

- **one-instruction-per-sentence.** The detector works on clean fixtures, but real-doc precision
  is only about half: 16 clear true positives of 34 flags (about 47%, up to about two-thirds if
  the borderline heading and option-label cases count). The false positives cluster in three
  classes: a heading read as a prose instruction (the tool has no block-type or section
  awareness — SPEC §7 defers that); a declarative whose capitalized opener the `pos` tagger reads
  as an imperative verb ("Install writes it, and Remove deletes it"); and a coordinator that joins
  two adverbials or objects, not two instructions ("Read alone and in the present tense"). A
  warning that is about half noise trains a writer to ignore it, so precision is below the bar the
  shipped warnings hold (simple-tenses ~100%, passive-voice ~99%).

- **no-ellipsis.** The detector flags a sentence-shaped block with no verb tag, as a dropped-verb
  proxy. On the real corpus it flags 1043 forms across 161 of 177 files. The sample is almost all
  false positives: a normal sentence whose verb the `pos` tagger mistags as a noun, an intentional
  verbless lead-in, or a list-item fragment. The dropped-article and dropped-subject drops are
  worse still, because English drops an article legitimately everywhere. This needs a parse the
  `pos` tagger does not support, exactly as SPEC §2.5 predicted.

## Revisit when

A better tagger lands (SPEC §7 notes `en-pos` claims higher accuracy but is stale), or the tool
gains block-type or section awareness (SPEC §7, deferred). Heading exclusion alone would remove
the largest one-instruction false-positive class.
