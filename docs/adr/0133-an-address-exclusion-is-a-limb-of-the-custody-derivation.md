# ADR-0133: An address exclusion is a limb of the `Custody` derivation and cuts the `Seed` limb alone

- **Status:** Accepted
- **Date:** 2026-09-01
- **Ticket:** [#1022 An address exclusion enforces nothing](https://github.com/winniel123/verge-asm/issues/1022)
- **Amends:** [ADR-0012](./0012-a-proposer-is-not-a-source.md), [ADR-0013](./0013-custody-is-control-and-extends-by-declaration.md)
- **Constrained by:** [ADR-0129](./0129-a-shared-foreign-edge-is-measured-by-fan-out-not-read-from-a-list.md) §3 and its [#956](https://github.com/winniel123/verge-asm/issues/956) amendment

## Context

[ADR-0012](./0012-a-proposer-is-not-a-source.md) §125 extended `Seed` exclusions to address
scopes: `CONTEXT.md` had defined them as *"exact names or subtrees"*, and they now cover CIDRs
too. `CONTEXT.md` states the consequence in two places — an address is in the estate while a
current resolution cites it **or** a `Seed` covers it, and *"leaving a declared scope, by
exclusion or by a narrower declaration, withdraws it and takes its timelines with it, unless a
current resolution still cites it"*.

**No gate ever read that limb.** [#1022](https://github.com/winniel123/verge-asm/issues/1022)
records the evidence and this ADR does not restate it in full. In summary, an `address`
exclusion is validated, canonicalised, stored and drawn as a chip, and then read by nothing:

- `ListAddressScopeCidrs` (`db/queries/measurement.sql:33`) selects the declared scopes with no
  join to `exclusion`.
- `queue.candidateAddrs` (`internal/queue/hot.go:147`) and
  `custody.Estate.EdgeFanoutPopulation` (`internal/custody/candidates.go:119`) enumerate every
  declared scope. Neither consults an exclusion.
- `Estate` (`internal/custody/custody.go:59`) holds no exclusion field, so `Derive` returns
  `operator` for any address a scope covers and `MayProbe` (`internal/custody/gate.go:44`) opens.
- `internal/queue/membership.go` has `nameExcluded` (`:323`) and no address analogue. Both it and
  `coveringExclusionKey` (`:127`) skip every row where `e.Name.Valid` is false.

Two costs follow. **The declaration preview promises a withdrawal that never happens.**
`POST /exclusions/preview` renders a receipt off the live span corpus saying N subjects are
*taken out of the estate*, and that *a listener answering inside the range after this act is not
seen* (`internal/message/render.go:100`). `message.Narrowing` — the function that would write
the coverage message the preview names — has no production caller. **And
[#989](https://github.com/winniel123/verge-asm/issues/989)'s census row names a remedy that
cannot clear it.** The row reads *"exclude them from this scope if they are not yours"*. An
operator who does exactly that sees the same row at the same count on every later load. The
comment at `internal/custody/scopecensus.go:29` already records this and points here.

The one enforcement that does exist is unrelated to probing: declining a registry `Proposal`
writes an address exclusion so the proposal is not re-offered (`cmd/web/proposals.go:358`). That
is ADR-0012's decline path.

## Decision

### 1. An address exclusion cuts the `Seed` limb alone

`Derive` has two limbs that return `operator`: `coveredByAddressScope` and `coveredByExtension`.
An address exclusion narrows the **first** and leaves the **second** standing. An address inside
an excluded range that a `custody extension` also reaches still derives `operator`, and is still
probed.

This is forced by ADR-0129 §3 as its [#956](https://github.com/winniel123/verge-asm/issues/956)
amendment sharpened it. That amendment fixed the limbs as **disjoint and never ranked** —
`shared-edge` carries no weight on the `Seed` limb, so a `Seed`-covered address derives
`operator` at any fan-out count. A global exclusion would rank one limb above the other in the
opposite direction and undo the same law.

It also follows from the membership rule. The two limbs of *in the estate* are disjunctive, so an
address a current resolution still cites does not leave when the declaration stops covering it.
An exclusion that shut the gate over such an address would contradict `CONTEXT.md` directly.

Stated as a bound: **the set an exclusion removes is never larger than the set the declaration
added.** *Not mine* is a claim about the operator's own declaration. It does not overrule their
own name resolving at the address.

### 2. The `Estate` carries exclusions in an unexported field, set by one constructor

`Estate` gains an unexported address-exclusion field, written only through a
`WithAddressExclusions` constructor, in the shape `WithEdgeFanout` already established
(`internal/custody/custody.go:70`). Containment reuses the family-matched rule the coverage
predicate already applies.

The field is unexported for the reason that comment gives. Three sites build an `Estate` literal
today — `cmd/web/vantageclass.go:40`, `internal/queue/produce.go:385` and
`cmd/web/addressscopecensus.go:72`. A new **exported** field is silently zero at each of them and
the compiler reports nothing. The zero value means *no exclusions*, which is the safe reading for
an assembler that has not opted in.

The alternative — subtracting the exclusions from the declared prefixes at assembly time — is
rejected in the table below.

### 3. The exclusion narrows the enumeration, not only the gate

The gate is already **total**: `MayProbe` runs over every enumerated candidate at all four gate
sites (`internal/scan/hot.go:76`, `internal/scan/cold.go:131`,
`internal/scan/httpidentity.go:101`, `internal/scan/tlsacceptance.go:104`). So narrowing the gate
alone is sufficient for correctness — no probe fires at an excluded address either way.

It is not sufficient for cost. An excluded `/16` inside a declared `/8` is 65,536 addresses walked
per tick and refused one at a time. So `EdgeFanoutPopulation` and `candidateAddrs` skip an
excluded address as well, as a `continue` beside the filters each loop already runs — never as
prefix arithmetic.

**The cold tier needs its own change.** `fanOutCold` passes `scope.AddressPrefixes` from
`coldScope`, not `estate.AddressScopes` (`internal/queue/cold.go:50`). A change that touches only
the hot path leaves the cold sweep walking the excluded range.

### 4. The Vantage-class coverage predicate narrows with it

`CoversAddressScope` is a thin wrapper over `coveredByAddressScope`. So §2 also narrows the
`covered` predicate the Vantage-class derivation binds at `cmd/web/vantageclass.go:40` and
`internal/queue/produce.go:385`. A vantage whose egress sits inside a newly excluded range stops
being covered, and `exposure.VerifyClass` may reclassify it.

**This is accepted rather than worked around.** [#711](https://github.com/winniel123/verge-asm/issues/711)'s
invariant is one binding used identically by batch gating and every render, and the comment at
`cmd/web/vantageclass.go:15` exists to keep it that way. A second, un-narrowed predicate for
classification alone would leave two coverage rules that a later session must hold in step. The
consequence is consistent on its own terms: the operator has said the range is not theirs, so a
prober inside it is not inside the estate.

### 5. This moves `custody/v2` to `custody/v3`

`internal/custody/version.go` lists what moves the version, and **the two coverage limbs
themselves** are on that list. §1 changes a coverage limb, so the bump is owed and is a `Break`
rather than drift.

The bump is cheap today. Nothing composes `Custody` into a `drift` component vector, which
`golden-corpus.md` §10.5 records as an open question that
[#986](https://github.com/winniel123/verge-asm/issues/986) did not take.

**An operator act still moves the version not at all.** The shipped rule changes once, at this
release. Declaring an exclusion afterwards changes an `Estate` input and not what the derivation
computes, exactly as declaring a scope already does.

### 6. The corpus gains a `C4` block of three rows

`golden-corpus.md` §10.2 holds eight cells across `C1`/`C2`/`C3`. A `C4` block is added:

| Cell | Claim the row pins |
| --- | --- |
| `C4/excluded-inside-a-scope-is-third-party` | An address inside a declared scope and inside an address exclusion derives `third-party`, and the gate is shut |
| `C4/the-same-address-without-the-exclusion` | The same address with the exclusion absent derives `operator` and the gate opens, so the refusal above is the exclusion and not the fixture |
| `C4/excluded-but-extension-reached-is-operator` | An excluded address the custody extension **also** reaches derives `operator`. The exclusion cuts the `Seed` limb alone |

The third row is the load-bearing one. It is the guard `C2/seed-covered-at-threshold-operator`
already provides in the other direction, and §10.2 calls that "the strongest guard the map leaves
behind". §1 has the same failure mode reversed: a later session reads *the operator said not
mine* and makes the exclusion global. Without the row nothing catches it.

§10.2's eight-cell sentence moves with the table, on that section's own rule that eight is the
length of a list and not a target.

### 7. The census filters on read; no measurement is deleted

`AddressScopeCensusEntry.SharedEdges` counts stored `edge_fanout_observation` rows. Stopping the
enumeration stops new rows and removes none of the rows on record, so the
[#989](https://github.com/winniel123/verge-asm/issues/989) row would not clear on its own.

The census therefore **drops observations for addresses an exclusion now covers, on read**.
Nothing is pruned. The measurement happened and the record is true;
[ADR-0006](./0006-subjects-leave-by-measurement.md)'s *subjects leave by measurement* argues
against a declaration erasing a measurement.

### 8. The withdrawal limb is a separate ticket

Two limbs are broken and they sit in different packages with different blast radii.

- **The probe gate** — §1 through §7 — is `internal/custody` and its two enumeration callers.
- **The withdrawal** — closing the affected spans with the `descoped` reason and writing the
  first production call to `message.Narrowing` — is the membership fold. That is the `Span` path
  [ADR-0041](./0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md)'s estate-wide `Break`
  guards, and `golden-corpus.md` §10.1 is explicit that `Custody` is **not** a membership block.

They ship as two tickets, gate first. The preview receipt is **not** re-worded to match the
broken code in the meantime: it states the model's intent correctly, and editing it to promise
less would write the defect into the interface.

#### 8.1 Discharged by [#1032](https://github.com/winniel123/verge-asm/issues/1032) (2026-09-01)

~~The withdrawal limb is a separate ticket.~~ **Both limbs have shipped.** The withdrawal lives in
`internal/queue/withdrawal.go`. It closes the affected spans with the `descoped` reason and makes
the first production call to `message.Narrowing`. #1032 left two questions open and hit a third.
All three are ruled here, and the code comment points back at this section.

**When the withdrawal runs.** On the next membership fold, not at declaration time. Two things
force that. A closure cites the folding batch ([ADR-0111](./0111-a-span-cites-the-batch-that-folded-it.md)),
and a web handler holds no batch. And §3 stops enumerating an excluded address, so no observation
about one need ever arrive again. A withdrawal scoped to the subjects a batch observed — the rule
`foldEstateTransitions` applies to Names — could therefore never reach it. The fold is driven from
the declaration side instead and reads the exclusion corpus. The accepted bound: the withdrawal
lands on the next completed job, so an estate running no jobs holds its spans open until one
completes.

**Whether the preview's counts must match the act's.** They are not required to. The listing query
is the twin of `PreviewExclusionWithdrawal` and reads the same CTE shapes, so the two agree over an
estate that has not moved. The estate may move between them. Each count is a measurement at its own
instant.

**An address the custody extension still reaches does NOT leave.** This is forced by §1 and #1032
did not see it. §1 keeps the extension limb standing under an exclusion, and §3 skips an excluded
address out of the scope enumeration alone. Such an address is still enumerated, still probed and
still measured. Closing its timelines would reopen them on the next batch and close them again on
the one after — a `descoped` departure and a `Narrowing` message every cadence, for an address the
gate never stopped probing. The survivor test is `custody.Estate.Derive` itself, so the membership
decision and the probe gate read one derivation and cannot disagree. It is the same shape as the
resolution survivor `CONTEXT.md` already states: another limb still holds the address, so the
narrowing does not remove it, and the set an exclusion removes stays no larger than the set the
declaration added.

**One message per act, not one per subject.** `message.Narrowing` is the count-carrying form
[ADR-0074](./0074-an-aperture-narrowing-that-takes-its-carrier-with-it-fires-at-the-scope.md) defines — a scope
and a count, no comparison and no rows. N per-subject `declared-input` rows for one operator act
would be the census the receipt exists to replace. The Name limb keeps its per-subject
`declared-input` message, because a name exclusion withdraws one Name and has no aggregate to state.

**Scope of this discharge.** Both limbs of the EXCLUSION act have shipped. `CONTEXT.md` names a
second narrowing act — a `Seed` that narrows or is withdrawn — and this ADR rules nothing about
it. That act is
[ADR-0134](./0134-a-seed-withdrawal-is-recorded-by-a-tombstone-because-the-mover-does-not-survive-the-act.md)
and [#1040](https://github.com/winniel123/verge-asm/issues/1040). It cannot reuse the machinery
here: this fold reads the live `exclusion` corpus, and a `Seed` delete destroys its own mover.

## Consequences

- An address exclusion becomes enforcing for the first time. An operator who excludes a range
  stops probing it on the next cadence, on both tiers.
- A vantage inside a newly excluded range may change class, per §4. This is the largest
  behavioural consequence and needs a test at the class site.
- `custody/v3` ships with a re-blessed corpus and a moved lock digest. CI's
  `corpus-version-gate` refuses the bump with nothing moved, and `TestCorpusLock` refuses the
  move with no bump, so the two land together.
- **No migration.** The `exclusion` table already holds `address` rows carrying a CIDR
  (`db/migrations/00004_exclusions.sql:20`). The change needs a new `sqlc` query and therefore a
  regenerated `internal/db`, which the `sqlc` check enforces.
- ~~The preview receipt stays a promise the code does not yet keep, until the §8 ticket lands. That
  is a known and time-boxed inconsistency rather than an accepted one.~~ **The §8 ticket
  ([#1032](https://github.com/winniel123/verge-asm/issues/1032)) has landed. The code keeps the
  promise: declaring an address exclusion closes the withdrawn timelines and writes the coverage
  message. See §8.1.**

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Filter `ListAddressScopeCidrs` in SQL** | The query feeds five consumers, including the Vantage-class offer and `produce.CoversAddressScope`. Filtering there changes all of them at once with no site able to opt out, and puts the rule outside the package the corpus locks, so no row can pin it |
| **A third limb of `MayProbe`** | It refuses after `Derive` has already returned `operator`, so it also shuts the gate over an address the extension reaches. That is §1's rejected semantics reached by accident |
| **Subtract the exclusions at assembly time** | CIDR subtraction turns one excluded `/25` into a set of covering prefixes and is easy to get wrong at the family boundary. It also leaves `custody` unchanged, so the corpus can pin nothing |
| **A global exclusion, cutting every limb** | Ranks the limbs, which ADR-0129's #956 amendment forbids, and contradicts `CONTEXT.md`'s disjunctive membership rule. It would also be a set removal larger than the declaration it narrows |
| **Keep the class predicate un-narrowed** | Leaves two coverage predicates that must be held in step by hand. #711's invariant is one binding, and `cmd/web/vantageclass.go:15` exists to prevent exactly this drift |
| **Prune `edge_fanout_observation` on declare** | Destroys a true measurement to clear a display. ADR-0006 puts departures on measurement, not on a declaration erasing one |
| **Re-word the preview receipt instead of fixing the gate** | Writes the defect into the interface. The receipt describes the model correctly; the code is what is wrong |
