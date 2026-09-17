---
number: 2277
title: "A field access corroborates nothing, and `sameName` keeps its suffix-only test"
slug: a-field-access-corroborates-nothing-and-samename-keeps-its-suffix-only-test
date: 2026-09-17
status: accepted
source: fix
ticket: 2277
proof: {test: "docs-site/scripts/sweep-line-anchors.test.mjs::a field access on the anchor's own type corroborates nothing"}
---

# ADR-2277: A field access corroborates nothing, and `sameName` keeps its suffix-only test

## Decision

> **A field access on a declared type corroborates nothing, and `sameName` keeps its suffix-only
> test.** A citing `signalRow.seenAge` neither corroborates the anchor `signalRow` nor rivals it. It
> scores `unproven`, and the dotted-name question moves to limb 1 of the tree test, which
> [#2155](https://github.com/winniel123/verge-asm/issues/2155) owns.
>
> A reader asks why the most ordinary way prose names a declaration fails to support it. Because
> `sameName` serves both arms of `rivalName`. Corroboration returns before the rival scan, so one
> prefix arm would suppress rivals that rule 1 reaches and manufacture rivals at the same time.
>
> Rejected: a prefix arm on `sameName`. `docs/spec/citation-anchor-repair.md` §5.3 bars relaxing
> rule 1, forbids a second matcher, and the tree holds no harness to measure the net.
>
> Reversal moves an unmeasured number of sound anchors, in both directions at once.

## 1. Context

`docs-site/scripts/sweep/corroborate.mjs#sameName` answers true when two names are equal, or when
one ends with a dot and the other. It matches a **suffix** and not a prefix. So
`retention.Retirer.Run` names the declaration `Retirer.Run`, and `signalRow.seenAge` names nothing.

[#2257](https://github.com/winniel123/verge-asm/issues/2257) filed that asymmetry as a defect. A
field access on a declared type is an ordinary way for prose to name that declaration, and the
citing line at ADR-0179 line 33 spells exactly that pair. The name falls between both arms: it
corroborates nothing, and `docs/spec/citation-anchor-repair.md` §5.3 does not close it to the tree
test either.

§9 fact F12 counts it. `signalRow.seenAge` is one of the 14 raised occurrences, and the fact's site
table records it as a false positive, "a struct field of the anchor's own type, so the prose
supports the anchor rather than rivalling it".

The obvious repair is one clause: give `sameName` a prefix arm. This ADR rules that the clause is
not available.

## 2. Why the obvious repair is not local

`sameName` has two callers inside `docs-site/scripts/sweep/corroborate.mjs#rivalName`, and they
want opposite things.

| Arm | What it asks | What a match does |
| --- | --- | --- |
| Corroboration | does the identifier name the anchor's own region? | returns `corroborated`, before any rival is picked |
| Rival | does the identifier name some **other** declaration the target has? | yields the rival, and rule 1 degrades the token |

The corroboration arm runs first and returns. So the two arms are not independent: every extra
corroboration is a rival that never gets scanned for.

One test predicate serves both. `docs/spec/citation-anchor-repair.md` §5.3 requires that, in as many
words: *It must not reimplement `rivalName`. One derivation serves every caller.* So a prefix arm
cannot be given to one arm and withheld from the other.

## 3. The two directions, and why the net is unreadable

A prefix arm widens both arms at once.

- **Fewer degradations.** A citing line spelling `signalRow.seenAge` beside an anchor on `signalRow`
  now corroborates. The verdict returns early, so a rival that rule 1 would have reached is never
  picked. Sound anchors that degrade today stop degrading — and so do drifted ones.
- **More degradations.** The same widened test also decides the rival. Any identifier that prefixes
  a declared name the target carries now matches it, so names that score `unproven` today become
  `suspect`, and rule 1 degrades them.

Which effect dominates is a property of the corpus, not of the source. Nothing in the matcher
answers it, and neither does a reading of the 459 names F12 counted, because F12 scored each name
against a one-name inventory rather than against a target's whole declaration set.

**There is no harness.** §9 fact F12 records its own method under *How the shapes were derived*: the
run was hand-instrumented, it blanked every other span per probe, and the tree holds no harness for
it. §5.3 forbids writing a second matcher to measure the first. So the measurement that would settle
this cannot be taken with the tools the tree has.

## 4. The rejected alternative

**Give `sameName` a prefix arm, and take the widening as an improvement on its face.**

The argument for it is real. A field access genuinely names its type. Raising it to `corroborated`
removes a measured false positive from F12, and it is one clause of code.

It is rejected on three grounds, and the first is decisive.

1. **§5.3 bars it.** *It must not lower the bias toward degrading* and *A builder never relaxes rule
   1*. Widening corroboration suppresses rivals that rule 1 would otherwise reach. That is the
   relaxation the sentence names, whatever the intent behind it.
2. **It is unmeasured in the direction that matters.** §5.3 also rules that a rule raising a verdict
   reaches only a shape whose false-positive rate a §9 fact records. No fact records the rate for
   the rival half of this widening, because no run has taken it.
3. **It buys one occurrence.** Against the repaired guard, F12's raised population is 13 occurrences
   over 11 anchors. `signalRow.seenAge` is one of them. The clause spends the matcher's central
   contract on a single site.

## 5. The cheaper reading, and why it is not this ADR's to take

§9 fact F12 records a second reading of the same evidence. Limb 1 of the tree test may ask whether
the tree spells a dotted name's **tail** — `seenAge` of `signalRow.seenAge`, `password_hash` of
`account.password_hash` — rather than the whole name.

That reading reaches further than a prefix arm and costs less. It empties shape B, drops shape D
from 6 occurrences to 3, and cuts the raised population from 14 occurrences to 3. It touches
`sameName` not at all, so it moves neither arm of `rivalName`.

It is out of scope here for one reason: the tree test is unimplemented.
[#2155](https://github.com/winniel123/verge-asm/issues/2155) owns it, and that ruling binds it to
ship in one pull request with its own §9 measurement. Deciding limb 1's reading inside this ADR
would take that ruling's first step away from it, and would do so without the measurement §5.3
requires.

So this ADR hands the dotted name to #2155 and rules only on `sameName`.

## 6. What this ADR does not rule

**Whether a rule of #2155's shape should ship at all.** F12 measures 13 false positives in 13 over
an empty true-positive population. §5.3 hands that rate to a reader by name, and this ADR is not
that reading.

**Which reading limb 1 takes.** §5 gives it to #2155, unanswered.

**Case folding inside `sameName`.** F12 measured it separately: a folding `sameName` makes 3 of the
459 names a rival of the anchor's own target, all 3 spelled before the citation, so all 3 would
degrade a sound anchor. That is a different change to the same function, and it stays rejected on
its own measurement.

**Shape B.** [#2257](https://github.com/winniel123/verge-asm/issues/2257) rules it needs no matcher
repair, because the line-anchor conversion of
[#2156](https://github.com/winniel123/verge-asm/issues/2156) empties it.

## 7. Consequences

No code changes. The matcher behaves today as this ADR rules it should, and the ruling is what was
missing.

A citing line that names a declaration through a field access keeps scoring `unproven`. The anchor
converts, and the guard reports nothing. That is the honest verdict while the question is open: the
line is evidence the matcher cannot read, and §5.2 rule 2 exists because a silent decision either way
is worse than a report.

F12 keeps its figures. Nothing here changes the matcher, so nothing here needs a re-measurement.

Reversal moves an unmeasured number of sound anchors in both directions at once, with no harness to
say how many, or which way.

## 8. Proof

The proof names `a field access on the anchor's own type corroborates nothing`, in
`docs-site/scripts/sweep-line-anchors.test.mjs`. That test locks both halves. `signalRow.seenAge` against the region `signalRow`
scores `unproven`, and `web.signalRow` against the same region still scores `corroborated`. The
first is the prefix arm this ADR rejects. The second is the suffix arm it keeps.
