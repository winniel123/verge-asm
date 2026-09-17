---
number: 2155
title: "A rival the target no longer declares stays `unproven`, because F12 measures no true positive"
slug: a-rival-the-target-no-longer-declares-stays-unproven-because-f12-measures-no-true-positive
date: 2026-09-17
status: accepted
source: fix
ticket: 2155
proof: {test: "docs-site/scripts/sweep-line-anchors.test.mjs::a rival the target no longer declares stays unproven"}
relations:
  - {kind: bounds, adr: 2277, clause: "5"}
---

# ADR-2155: A rival the target no longer declares stays `unproven`, because F12 measures no true positive

## Decision

> **A rival the target no longer declares keeps the verdict `unproven`.** `rivalName` scores an
> identifier against the target's current inventory. A withdrawn name matches nothing, and that
> verdict stands.
>
> A reader asks why, because the scoring inverts. A name the target still declares scores `suspect`
> and degrades the token. Withdrawing that declaration scores weaker.
> `docs/spec/citation-anchor-repair.md` §5.2 records the inversion.
>
> Rejected: raise the withdrawn name to `suspect`, position `before`, where no inventory in the tree
> carries it. §9 fact F12 measures what §5.3's rails leave that rule: 13 occurrences over 11
> anchors, every one a false positive, over an empty true-positive population.
>
> Reversal degrades those anchors to a bare path. §6.3 forbids repointing one, so each loses the
> line a later repair would read.

## 1. Context

`docs-site/scripts/sweep/corroborate.mjs#rivalName` reads the cited target's own declared set. It
matches every identifier of the citing line against that set alone. A name the target once declared
and has since dropped matches nothing, so the function returns `unproven`.
`docs-site/scripts/sweep/derive.mjs#derive` degrades only a `suspect` verdict at position `before`,
so the token converts and the sweep writes an anchor over the drifted citation.

[#2155](https://github.com/winniel123/verge-asm/issues/2155) filed that as an inversion. The
strongest drift evidence the corroborator can hold — the target dropped the very name the prose
spells — produces the weakest verdict it can award.

The inversion is measured. One citing line, varying only the declared set:

| The target declares | Verdict |
| --- | --- |
| nothing matching the name | `unproven`, no rival |
| the name | `suspect`, rival, position `before` |

`cmd/web/deltas.go` is the site. It declared `certExpiryWindow` at `80198db`, and no Go file
declares that name now. ADR-0186 names that path and spells that name, and the sweep wrote
`#deltasStore` there.

## 2. What the tree test is, and what shipped before this ruling

The repair the issue proposed is the **tree test**: a name no declaration anywhere inside §1.2's
boundary carries scores `suspect`, position `before`. Withdrawal is then evidence rather than the
absence of it.

Two pull requests landed ahead of this ruling, both at its own instruction.

- [#2208](https://github.com/winniel123/verge-asm/issues/2208) wrote the boundary into
  `docs/spec/citation-anchor-repair.md` §5.3. A weak name scores `unproven` however the tree reads.
  Limb 1 closes a name the tree spells in a position no declaration vocabulary carries. Limb 2
  closes a standard-library field or method name. The same section added the rail that decides this
  ADR: **a rule raising `unproven` to `suspect` reaches only the shape whose false-positive rate a
  §9 fact records.**
- [#2014](https://github.com/winniel123/verge-asm/issues/2014) took that measurement. It is §9 fact
  F12.

So the ruling wrote its own condition first, and then read what the condition returned.

## 3. What F12 returned

F12 judges 626 written anchors and calls 512 `unproven`. 208 of them spell a name the target lacks,
in a span the guard reads. Those 208 sit on 195 citing lines and spell 459 such names.

| Shape | What the name is | §5.3 | Occurrences |
| --- | --- | --- | --- |
| A | some inventory in the tree carries it | closed | 124 |
| C | weak under limb 1 | closed | 321 |
| B | no inventory carries it, and one carries a case-folded spelling | **open** | 8 |
| D | residue | **open** | 6 |

**The tree test reaches B and D alone.** That is 14 occurrences over 12 anchors, as the run
measured it. §5.2 rule 4 has since dropped one shape-D occurrence, the `fa97` cut out of the commit
hash `860fa97`, so the reach against the guard the tree ships is 13 occurrences over 11 anchors.

**A hand read finds every one of them a false positive.** F12 tabulates the sites. A table column
header. A database column. A struct field of the anchor's own type. A withdrawn local whose
declaration still computes the stat. Eight retired line anchors spelled as a bare basename, where
`artifactdoc.go` folds on its extension onto a declared `Go`.

**The true-positive population on this boundary is empty.** The true positives left the corpus
before it was measured. [#2011](https://github.com/winniel123/verge-asm/pull/2011) degraded the
ADR-0186 rows by hand, so no name they spell is a written anchor's rival today.

## 4. The rejected alternative

**Ship the tree test, and accept the false positives.**

The argument for it is the SPEC's own, and it is quoted in §4.3: *"the cost of a false positive is
small and the cost of a miss is not."* The cost is bounded. §6.3 rules that a suspect anchor
degrades to a bare path and that no tool repoints it, and §8 rules that no check refuses a bare
path. So a wrong degrade reds no gate.

**That trade assumes a miss exists to prevent.** After §5.3's two rails, on this corpus, none does.
The rule would create 13 degrades and recover no fact. A trade that spends only on one side is not
a trade.

Two narrower forms were weighed and refused with it.

**Ship it narrowed to shape D alone.** Six occurrences, five after rule 4. All are false positives
under the same hand read, so the narrowing changes the rate and not the verdict.

**Ship it and wait for shape B.** Shape B needs no rule. All eight are retired line anchors, and
the conversion of [#2120](https://github.com/winniel123/verge-asm/issues/2120) empties the shape on
its own.

## 5. What this ADR does not rule

**That the inversion is acceptable.** It is not. §1 states it, and
`docs/spec/citation-anchor-repair.md` §5.2 keeps the record so a later reader meets it. This ADR
rules that the repair costs more than the defect on the corpus as measured, and nothing more.

**Which reading limb 1 takes.** F12 records a second reading, which asks whether the tree spells a
dotted name's tail rather than the whole name. Under it the raised population falls to three
occurrences, and the rate reads 3 of 3. ADR-2277 §5 handed that reading here. It has no live rule
to serve now, so it lapses with the tree test, and this ADR declares `{kind: bounds, adr: 2277,
clause: "5"}` to record where that hand-off stops. `bounds` derives no status and writes no marker,
so ADR-2277 is untouched.

**§2's Terms table.** It defines a rival as a declaration the target really has, and a suspect
anchor as one with a rival. The tree test would have awarded `suspect` with no rival in the target,
so the ruling's pull request owed §2 a repair. The refusal discharges that debt. The table
describes the tree as it stands.

**The two relocations.** `weakKeyOrSignature` and `selfSignedOf` are live under an exported
initial, and `sameName` is case-sensitive.
[#2011](https://github.com/winniel123/verge-asm/pull/2011) holds them. Folding the case inside
`sameName` is no repair either: over all 459 names it makes three a rival of the anchor's own
target, all three before the citation, so all three would degrade a sound anchor.

## 6. Consequences

No behaviour changes. `rivalName` scores a withdrawn rival today as this ADR rules it should, and
the ruling is what was missing.

A citation whose prose names a declaration the target has dropped still converts. The sweep writes
an anchor there, and the guard reports nothing. The drift such a citation carries is real, and this
ADR leaves it uncaught by the corroborator. The history arm is the instrument that reaches it: §9
fact F9 records the arm naming `certExpiryWindow` among the 17 anchors it reproduces.

`docs/spec/citation-anchor-repair.md` §5.3 no longer holds a rule pending. It states a boundary,
and this ADR is the reading of F12's rate that the same section asks a reader to take.

**If the corpus changes, re-measure before re-opening.** F12 names the two commands that reproduce
its inputs. The decision rests on a rate over one corpus, so a corpus that grows a true positive
reopens it.

## 7. Proof

The proof names `a rival the target no longer declares stays unproven`, in
`docs-site/scripts/sweep-line-anchors.test.mjs`. It calls `rivalName` twice on one citing line
spelling `certExpiryWindow`, and varies only the declared set.

| The target declares | Asserted verdict | What it locks |
| --- | --- | --- |
| `Alpha` | `unproven`, rival `null` | the verdict this ADR rules, on the withdrawn name |
| `Alpha`, `certExpiryWindow` | `suspect`, rival `certExpiryWindow`, position `before` | the inversion §1 records, and its `before` position |

The two rows together are the reproduction §1 tabulates. A pull request that shipped the tree test
would fail the first row.

**This suite is not run by any required check, so the guard is manual.** `test:sweep` appears in no
workflow, because [#1975](https://github.com/winniel123/verge-asm/issues/1975) keeps the sweep tool
out of the required checks' path. ADR-2277 §8 records the same gap, and
[#2279](https://github.com/winniel123/verge-asm/issues/2279) holds it.
