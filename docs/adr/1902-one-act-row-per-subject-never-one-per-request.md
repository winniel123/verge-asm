---
number: 1902
title: "One Act row per subject, never one per request"
slug: one-act-row-per-subject-never-one-per-request
date: 2026-09-13
status: accepted
source: grilling
ticket: 1902
map: 1786
proof: {test: "cmd/web/act_limb1_test.go::TestDeclineRecordsOneActPerSubject"}
relations:
  - {kind: amends, adr: 92}
  - {kind: rests-on, adr: 1891}
---

# ADR-1902: One Act row per subject, never one per request

## Decision

**An `Act` is written once per subject the act directed, never once per request.** A submit that
moves two dials writes two rows. A decline of 200 proposals writes 200 rows, which reads as a defect
and is not one. A declaration that directs one subject writes one.

Two grounds. A request-level row needs a list-valued subject, and `subject` is a **rendered** column,
so the Subject cell would acquire a second thing to render. And a multi-subject handler bails
mid-loop with the applied part already committed, so a per-subject row is the only shape that can be
true about a partial batch.

Rejected: one row per request, carrying a list-valued subject.

Reversal rewrites a closed union after it ships, into a corpus that is never deleted.

**So declaring an `Annotation` leaves one record beyond its own row, not none.**

## 1. Context

[ADR-1891](./1891-an-operator-act-is-recorded-as-a-fifth-operational-corpus-under-a-four-limb-predicate.md)
records an operator act as an `Act`, the fifth Operational corpus, under a four-limb predicate. It
rules **what** is recorded. It does not rule **how many rows** one submit writes, and the two
questions come apart the moment a handler directs more than one subject.

Four shipped handlers direct more than one:

| Handler | Subjects | Rows |
| --- | --- | --- |
| `POST /proposals/decline` (`cmd/web/proposals.go`) | the checked proposal ids | one per proposal declined |
| `POST /seeds` (`cmd/web/seeds.go`) | the scopes in a pasted list | one per scope declared |
| `POST /seeds/zone` (`cmd/web/seeds.go`) | the apexes in the upload | one per apex accepted |
| `POST /coverage/retention` (`cmd/web/retentionpanel.go`) | the observation currency, the dispatch cadence | one per dial that moved |

[`docs/spec/audit-act.md`](../spec/audit-act.md) §7.6 ruling 3 states the rule and §2.2 works one
case. This ADR is the record, because the rule is a decision about the corpus rather than a detail of
four handlers.

## 2. Why a list-valued subject fights the union

[`docs/spec/audit-act.md`](../spec/audit-act.md) §4 closes the stored value over 61 variants, each
naming its own typed subject, and §4.1 bars a list-valued subject in every one of them. §4.4 gives
the ground: **Subject is a rendered column**, one of the four the shipped table already has.

A request-level row makes the cell carry two things at once — what the act directed, and how many.
`Proposals declined · 200 subjects` is a count with the subjects lost, and
`203.0.113.0/24, 198.51.100.8/29, …` is a cell the shipped `white-space: nowrap` clips silently.
Neither renders the act.

The fence is not a style preference. [ADR-0209](./0209-a-closed-union-we-author-refuses-an-unknown-member-and-writes-no-row.md)
holds that a union we author refuses an unknown member, and every variant's payload field is a
`string` or an `int64` — a rule `internal/act` tests directly. A list-valued payload is the first
member that rule would have to admit, and it would be admitted for the convenience of one shape of
handler.

**[ADR-0150](./0150-a-batch-scope-names-its-dimension-in-the-plural-and-a-one-address-fan-out-ships-a-one-element-list-never-a-scalar.md)
points the other way and does not bind here.** A `Batch` scope is a list because the batch's
dimension is plural — the scope is one fact about one batch. An `Act` subject is not one fact about
one request. It is the subject the principal directed, and there are two of them.

## 3. Why a partial batch settles it

The rendering argument alone would leave a reader free to prefer the shorter table. The partial batch
does not.

`declineLookup` loops over the checked ids, does two unrelated writes per iteration, and **bails
mid-loop** through `s.serverError` with every earlier iteration already committed. The shape predates
the corpus. It is the handler the corpus had to record, not one written for it.

A request-level row can only be written after the loop, and then it is false in both directions:
written, it claims subjects the loop never reached; withheld, it erases the declines that did commit.
**A per-subject row is the only shape that can be true about a partial batch**, because the rows are
written from the subjects that committed and from nothing else. The multi-subject handlers reach
that two ways — `declineLookup` records inside its own loop, while `declareSeed` and `uploadZoneFile`
collect the accepted subjects and record them in a second loop, which §7.6 places after the mutation on
purpose. Either way the list a refusal truncates is the list the rows are written from.

`TestAPartialDeclineBatchRecordsOnlyTheAppliedSubjects` holds this. It fails the second proposal's
exclusion write and asserts the corpus holds the first subject alone.

This also keeps the corpus inside ADR-1891's rule that an act is recorded when a principal **did**
the thing. A row covering 200 subjects when 137 committed records an act that did not happen.

## 4. The price, named rather than hidden

**A 200-item decline writes 200 rows.** A reader who meets that page first reads a defect. The SPEC
accepts the price at §7.6, and this ADR accepts it for the same reason: the rows are true, and the
alternative is a shorter table that is not.

Two facts bound the price.

- **The corpus is unbounded and never deleted** (§5.1). Rows are not reclaimed, so the cost is
  permanent rather than a spike.
- **No dial trims it.** The reader's only scope is the date range of §6.2, which narrows the page and
  not the corpus.

Nothing here is a licence to fold rows later. The fold is the rejected alternative, and §7.6's price
is what buys the refusal.

## 5. What reversal costs

The union ships with the corpus. Reversing this rule afterwards changes the payload of every variant a
multi-subject handler writes, from a scalar to a list.

The corpus is append-only and never deleted, so the rows written under the scalar shape stay. The
decoder then reads **both** shapes for the life of the install, keyed on nothing the row carries —
`action` is unchanged by the fold, and there is no version column. Every consumer of the Subject cell
reads both too.

That is the whole of *hard to reverse*: not the edit, but the two shapes the corpus is left holding.

## 6. What this carries into ADR-0092

[ADR-0092](./0092-an-operator-dials-movement-is-not-a-cause-and-an-annotation-never-lapses.md) rules
that an operator dial's movement is not one of the four causes, so declaring an `Annotation` fires no
`Message`. **That outcome stands and is not touched here.** Its table states the outcome this way:

> **Declaring** an `Annotation` | **Silent.** No message, no record beyond the row itself

The first clause is about causation and holds. **The second clause is false the day the corpus
ships.** Declaring an `Annotation` is a limb-1 act (`POST /annotations`, `annotation.declared`), so it
writes an `Act`. Read alone and in the present tense, *"no record beyond the row itself"* carries no
qualifier and says **nowhere**.

Under this ADR's rule the count is exact. A declaration directs one subject — the
`(subject key, signal name)` pair — so it writes **one** row, not none and not one per submit.
`TestAnnotationActsRecordThePairAndNoProse` asserts the pair and no prose.

**The relation rides here rather than on ADR-1891, and that is mechanism rather than preference.**
ADR-1891 and ADR-1892 have merged, and a merged ADR's relation list is not hand-edited
([ADR-1644](./1644-for-an-adr-target-a-withdrawal-is-a-tool-written-marker-not-a-hand-edited-sentence.md);
[`docs/spec/adr-governance.md`](../spec/adr-governance.md) §3, §5). ADR-0092 numbers no heading, so the
relation is whole-ADR and the marker sits under the H1. The marker tool writes it. No hand does.

**The sibling cell is left alone.** *"**Withdrawing** an `Annotation` | **Silent.** The carrier is the
pair's own next `not-fired` → `fired` firing"* names its carrier, so its claim is about causation and
not about what is recorded. [`docs/spec/audit-act.md`](../spec/audit-act.md) §8 · B already rules that
sense outside the withdrawal set, on ADR-0092's *"with no operator act"*.

## 7. Three near neighbours, and why none of them rules this

| ADR | Why it differs |
| --- | --- |
| [ADR-1891](./1891-an-operator-act-is-recorded-as-a-fifth-operational-corpus-under-a-four-limb-predicate.md) | Rules **which** acts are recorded, through a predicate over principals. Cardinality is not in the predicate: every limb is satisfied once by a submit that directs two hundred subjects |
| [ADR-0126](./0126-verbatim-job-output-is-a-fourth-operational-corpus-retired-by-a-duration-dial-that-ships-bounded.md) | The nearest corpus precedent, and the `action TEXT` plus `subject JSONB` row shape follows its `Transcript`. It rules retention and the secret, never the grain of a row |
| [ADR-0077](./0077-a-second-ground-counts-only-where-it-would-have-carried-the-cells-proposition-alone.md) | The nearest *per-X* ruling — a filter run *"per cell, never per row"*. It governs the derivation path, which §2 of ADR-1891 fences away from an `Act` entirely |

## 8. Consequences

- **No migration and no schema change.** The rule is about how many rows a handler writes, not about
  what a row holds. `act` is unchanged.
- **The rule is uniform across all 61 classes.** A per-limb or per-handler carve-out would put a
  cardinality classifier inside the conformance test of §7, which reads call sites and not counts.
- **A submit that moves nothing writes nothing.** One row per subject **directed**, and a dial reset
  to its standing value directed no subject.
  `TestCoverageRetentionWritesNothingWhenNeitherDialMoves` holds this.
- **ADR-0092's `Message` ruling is untouched.** Declaring or withdrawing an `Annotation` still fires
  no message. The `Act` is not a `Message`, and no cause is minted.

## 9. Where this is thin, stated rather than smoothed

**Three of the four handlers take an operator-sized list, so the price is wider than one route.**
`POST /proposals/decline` takes the checked ids. `POST /seeds` takes a pasted list, and
`parseSeedTokens` splits it on commas and whitespace with **no count cap** — the address cap bounds a
prefix's size, never the list's length — so a pasted hundred writes a hundred `seed.declared` rows.
`POST /seeds/zone` takes the files in one submit. Only the retention panel is bounded, at two dials.

So the 200-row page is the ordinary shape of three routes rather than the worst case of one, and a
reader may fairly say the rule is being decided by its largest submit. The answer is that the
partial-batch argument of §3 holds on all three, and it does not depend on the count.

**The rendered table has not been read at this scale.** The Subject cell ships with its ellipsis
treatment (`design-system/templates/settings.tmpl:914`) and the reader ships with its date range, but
nobody has read an audit page holding a 200-row decline. If that page reads badly, the repair is in
the reader — grouping rows on the page — and never a fold in the corpus.

## 10. Proof

`{test: "cmd/web/act_limb1_test.go::TestDeclineRecordsOneActPerSubject"}`. The test declines two
proposals in one submit and asserts two `proposal.declined` rows, subject by subject:

> `want := []string{"address 203.0.113.0/24", "address 198.51.100.8/29"}`

Its sibling `TestAPartialDeclineBatchRecordsOnlyTheAppliedSubjects` fails the second subject's write
and asserts the first alone, which is the §3 ground.

The other three routes of §1 are held by a test each, and two of them assert the partial batch as
well:

| Route | Test |
| --- | --- |
| `POST /seeds` | `TestDeclareSeedRecordsOneActPerScope` — three tokens, one refused, two rows |
| `POST /seeds/zone` | `TestZoneUploadRecordsOneActPerApex` |
| `POST /coverage/retention` | `TestCoverageRetentionWritesTwoRowsWhenBothDialsMove`, with `…WritesOneRowWhenOneDialMoves` beside it |
