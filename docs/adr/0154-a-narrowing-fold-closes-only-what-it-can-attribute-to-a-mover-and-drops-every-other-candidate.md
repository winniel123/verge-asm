# ADR-0154: A narrowing fold closes only what it can attribute to a mover, and drops every other candidate

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1326 ADR gaps: internal/queue (seed withdrawal limbs)](https://github.com/winniel123/verge-asm/issues/1326), gap 2
- **PR that deleted the comment:** [#1325](https://github.com/winniel123/verge-asm/pull/1325)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Rules what three ADRs assumed:** [ADR-0133](./0133-an-address-exclusion-is-a-limb-of-the-custody-derivation.md) §8.1, [ADR-0134](./0134-a-seed-withdrawal-is-recorded-by-a-tombstone-because-the-mover-does-not-survive-the-act.md) and [ADR-0135](./0135-a-name-seed-withdrawal-states-one-act-and-its-tombstone-carries-the-domain-alone.md) each rest on this rule and none of them rules it. ADR-0134's Context is amended to point here, per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)
- **Rests on:** [ADR-0087](./0087-a-closure-records-the-ground-it-rests-on-and-there-are-three-grounds.md) (a closure records its ground, and the union of grounds is closed at three)
- **Sibling of, and not ruled by:** [ADR-0153](./0153-a-narrowing-mover-carries-no-precedence-so-the-first-covering-row-is-the-whole-attribution-rule.md). That ADR rules which mover the fold takes when several cover the candidate. This ADR rules what happens when none does

## Context

Three acts narrow the estate, and each has a fold that closes the spans the act removed.

| Act | Fold | Site |
| --- | --- | --- |
| Address exclusion | `composeAddressWithdrawals` | [`internal/queue/withdrawal.go:44`](../../internal/queue/withdrawal.go) |
| Address `Seed` withdrawal | `composeWithdrawnGround` | [`internal/queue/seedwithdrawal.go:67`](../../internal/queue/seedwithdrawal.go) |
| Name `Seed` withdrawal | `composeWithdrawnNameGround` | [`internal/queue/nameseedwithdrawal.go:65`](../../internal/queue/nameseedwithdrawal.go) |

All three walk a candidate row set and `continue` past any row they cannot attribute to a declared
mover. None of the three closes such a row. #1325 deleted the statement of that rule from
`seedwithdrawal.go` and compressed the survivor. The deleted text read:

> A row it cannot attribute to a tombstone is DROPPED, not closed, exactly as the exclusion act drops
> an unattributable row. A closure with no mover to name is a withdrawal the operator cannot trace
> back to their own act.

**The rule is written nowhere, and the ADR that looks like its home only quotes the code.**
ADR-0134's Context reproduces `composeAddressWithdrawals`' own comment as a blockquote and then says
"The same rule binds here". Its Decision, §1 to §7, never rules it. Its Alternatives table repeats
the sentence once more, as a reason for refusing a sweep. So the ADR uses the code as its evidence
that the rule already exists, and the code cited the ADR back.

[`comment-policy.md`](../spec/comment-policy.md) §8.3 names this exact shape as its third
mechanical defeat of the suppressing rule, and names #1326 as the record. ADR-0133 §8.1 discharged
the exclusion act without stating the rule either, and ADR-0135 inherits it through ADR-0134.

**The discipline is wider than the three folds.** `decideNameDeparture`
(`internal/queue/membership.go:119`) closes a Name `descoped` only where an exclusion covers it, and
`coveringExclusionKey` carries a comment saying it must mirror that same test so the cited boundary
is the one that removed the Name.

**The mover requirement is not a rule about every closure.** The same function closes a Name
`measured-absent` on a measurement ground and names no declared act at all. ADR-0087 keeps three
closure grounds, and only `descoped` asserts that somebody moved.

**The drop is a guard today, not a routine path.** `ListAddressExclusionWithdrawals`,
`ListSeedWithdrawalCandidates` and `ListNameSeedWithdrawalCandidates` all pre-filter to subjects the
mover covers, so the SQL gate and the Go covering test are written to agree. The ruling therefore
decides which way the fold fails when they disagree.

## Decision

> **A narrowing fold closes a candidate span only where it can NAME the declared mover that removed
> the ground, and only where every survivor test fails. Every other candidate is DROPPED and stays
> open. A closure the operator cannot trace back to their own act is worse than a span left open.
> The rule binds all three narrowing acts, and it attaches to the `descoped` ground rather than to
> closure in general.**

Five limbs.

### 1. Closing is the positive case, and open is the default

Each fold reaches a closure only after it decides three things. The subject key parses into the key
the mover is expressed in. A covering mover exists (ADR-0153 picks which one). No survivor still
holds the ground.

A candidate that fails any one of those is skipped. The fold appends no span id, counts no timeline
and names no subject in the receipt.

### 2. The ground is that `descoped` asserts an act

ADR-0087 closes the union of closure grounds at three, and `descoped` is the one that says our
aperture stopped covering the subject. A closure written with no mover to name would record
`descoped` while naming no act, which makes it indistinguishable from `uncited` to the operator
reading it. ADR-0134's Alternatives table rejected an estate-wide sweep on exactly this pair of
grounds.

The receipt is the second half of the same argument. A narrowing fires at the scope and carries a
count (ADR-0074). A span with no scope to fire at has nothing to count under.

### 3. A dropped candidate is not a lost one

The drop costs the fold two reads and closes nothing. It does not discard the act.

- The exclusion act reads its live `exclusion` row every fold, so the same candidate returns on the
  next completed job (ADR-0133 §8.1).
- Both `Seed` withdrawal acts spend a tombstone only once its ground is exhausted, so a tombstone
  that closed nothing stays pending and is read again (ADR-0134 §5.1, ADR-0135 §5).

This is what makes the drop safe. The alternative failure direction is not.

### 4. The mover requirement is bounded to `descoped`

A `measured-absent` closure rests on what the estate measured, and an `uncited` closure rests on the
absence of a citation. Neither names a declared act, and neither is reached through a narrowing
fold. This ADR does not add a mover requirement to them.

### 5. The drop is silent, and that is accepted

No fold logs the drop, opens a `Gap` or writes a `Message`. The three candidate queries pre-filter to
covered subjects, so a dropped row means the SQL gate and the Go covering test disagree, which is a
defect in this package rather than a fact about the estate. A `Coverage` line or a `Message` would
put an internal disagreement in front of the operator as though it were an estate reading, and the
span stays open and visible either way.

## Consequences

- **[ADR-0134](./0134-a-seed-withdrawal-is-recorded-by-a-tombstone-because-the-mover-does-not-survive-the-act.md)
  gains one pointer in its Context**, at the blockquote it rests on, per ADR-0058. A reader who
  arrives holding the code's `ADR-0134` citation must not have to find this ADR first.
- **[`internal/queue/seedwithdrawal.go`](../../internal/queue/seedwithdrawal.go) cites this ADR**
  instead of ADR-0134 on the drop. The old citation resolved to a document that quotes the rule and
  does not rule it.
- **[`internal/queue/withdrawal.go`](../../internal/queue/withdrawal.go) and
  [`internal/queue/nameseedwithdrawal.go`](../../internal/queue/nameseedwithdrawal.go) are not
  edited here.** `withdrawal.go` belongs to another sweep ticket, and the name fold states no ground
  to correct. Either may take this citation when its own ticket runs.
- **No production behaviour changes.** All three folds already drop.
- **A fourth narrowing act must state its mover.** An act that cannot name one is a different case
  and needs its own decision, which is what ADR-0134 §7's two named gaps will each face.
- **[`CONTEXT.md`](../../CONTEXT.md) gains nothing.** `descoped`, exclusion and `Seed` withdrawal all
  have entries, and this rules a fold rather than a term.
- **Nothing enforces this.** No test pins the drop at all three sites, so review carries the rule.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Close the unattributable candidate anyway**, with an empty scope | It writes a `descoped` closure that names no act. The operator cannot tell it from an `uncited` departure, and ADR-0087 keeps those two grounds distinct on purpose. The receipt would also have no scope to fire at (ADR-0074) |
| **Close it under a synthetic scope**, such as the subject key itself | It states a narrowing at a site the operator never declared. The coverage message would name a scope that appears in no `Seed` and no exclusion, and a `Message` is written once and never recomputed |
| **Sweep the estate for spans no declaration covers**, instead of driving from the mover | ADR-0133 §8.1 and ADR-0134 both refused it, for want of a mover and because a sweep cannot separate `descoped` from `uncited`. This ADR is the rule those refusals rest on |
| **Log the drop, or raise it as a `Gap`** | A dropped row means this package disagrees with its own SQL gate. That is a defect, not an estate reading, and `Coverage` would carry an operational fact keyed on nothing that moved. The span stays open either way |
| **Amend ADR-0134's Decision and file no ADR** | The rule binds three acts across three files, and ADR-0134 rules one of them. ADR-0133 discharged the exclusion act before ADR-0134 existed. Filing the three-act rule inside the one-act ADR would put it where two of its three sites have no reason to look |
| **Leave the rule where it is** | It survives as a Context blockquote of a comment the sweep deletes, plus one line in a rejected-alternatives table. #1325 could delete the last live statement of it, and the code's own citation would then point at a quotation of itself |
