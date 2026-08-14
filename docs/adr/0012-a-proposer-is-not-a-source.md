# A proposer is not a `Source`

- **Status:** Accepted
- **Date:** 2026-08-13
- **Ticket:** [#39 Is the ARIN SWIP customer path a Source in its own right, and may it gate probing?](https://github.com/winniel123/verge-asm/issues/39)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[#26](https://github.com/winniel123/verge-asm/issues/26) found that ARIN reaches
provider-aggregatable renters through SWIP **customer objects** — `C…` handles, compelled by
NRPM §4.2.3.7.1 down to a /29 — which is the population everyone had assumed was unreachable
without an ASN. [#39](https://github.com/winniel123/verge-asm/issues/39) was opened to ask what
kind of `Source` that path is, and it framed the question as a **safety** question: `authority`
governs admission, [ADR-0002](./0002-ownership-gates-probing.md) makes `Ownership` gate probing,
and a customer object is written by *the upstream provider* — neither the operator's `declared`
assertion nor our own `measured` observation. A stale SWIP record would therefore admit an
`Address` on a third party's clerical accuracy, and we would then send packets at it.

That framing did not survive the day it was written. [#27](https://github.com/winniel123/verge-asm/issues/27)
amended ADR-0002 hours earlier: `Ownership` is computed from **`Seed`s alone**, registry lookups
of every kind now only *propose* address scopes that the operator confirms, and *"an unconfirmed
proposal is read by nothing"*. The amendment states the consequence directly — registry accuracy
stops being a safety property. A stale SWIP /29 yields a **wrong proposal the operator declines**,
which is the same failure mode as every other registry path and costs coverage rather than safety.

With the safety question gone, what remained was a modelling question the ticket had not asked,
and could not answer in the form it did ask. Its three sub-questions — what is this path's
`authority`, its `completeness`, its `consent` — presuppose that it is a `Source`. `CONTEXT.md`
defines a **Source** as *"anything that can produce observations"*, and every one of the three
properties is defined in terms of observations: `authority` governs *"whose word is enough to put
a subject in the estate"*, `completeness` governs whether a source's **silence** may mean absence,
and a `Batch` exists to record the scope that silence covers.

After #27, a registry lookup does none of that. It puts no subject in the estate and its silence
licenses nothing. [#28](https://github.com/winniel123/verge-asm/issues/28) had already cut along
this exact line from the other direction, on coverage: a source that **fills a declared scope**
moves the coverage figure, a source that **proposes new scopes** cannot move it at all, so *"the
propose half carries no percentage, no bar, and no `unknown` in a cost column, ever."*

Two decisions had therefore each cut half of the same distinction — #27 on the probing gate, #28
on coverage — and neither drew the conclusion. The propose half was still wearing the word
`Source`, which is the `Host` defect: one term reading as two different things to two readers.

## Decision

**A source that proposes address scopes is not a `Source`. What it produces is a `Proposal`, and a
`Proposal` carries `consent` alone.**

### `Proposal` is a Declared term

A `Proposal` is a candidate address scope offered to the operator, real only once confirmed into a
`Seed`. It is filed under **Declared**, and the direction of travel is deliberately stated in the
glossary because a reader will trip on it: every other Declared term is something the operator
tells us, and this is something we tell the operator.

The other three layers were each considered and each fails. It is not **Observed** — there is no
observation, no subject, no facet, no vantage. It is not **Derived** — it concludes nothing about
the estate. It is not **Operational** — that group records what the system *did*, and a proposal is
not a record of an act. On the layer table's one operative test, *does it drift?*, it behaves
exactly as `Seed` does: no, it is input.

One noun, not two. A registry path is described as *a source that proposes*, at `Proposal`, rather
than being given a second noun of its own — this glossary refuses vocabulary aggressively and a
producer term would earn nothing that the produced term does not already carry.

### It carries `consent`, and only `consent`

- **`authority` is dropped.** It governs admission, and a proposal admits nothing. This is the
  ticket's own hardest question answered by dissolution rather than by argument: *what does the
  SWIP path's `authority` have to be?* has no answer because the question is **malformed**. It asks
  for the admission-strength of a thing that does not admit.
- **`completeness` is dropped.** It governs whether silence is evidence. #28 already ruled that the
  propose half moves no coverage figure, which is the same statement: its silence licenses nothing,
  so there is no `Batch` scope to record and nothing for `enumerable` or `corroborative` to
  distinguish.
- **`consent` survives unchanged**, and it is the same property rather than a copy of it. `consent`
  governs whether we may **make the request** — an act that happens whether or not anything
  observable comes back — so it is the one property that was never about observations.
  [ADR-0003](./0003-third-party-source-consent-bar.md) therefore continues to govern proposers in
  full, including its off-by-default outcomes.

### The ARIN SWIP path ships, as one proposer

It is **one** proposer, not two. The org objects and the `C…` customer objects come back in a
single response from a single endpoint under a single permission — `entities?fn=` returned 257
entities of which **227 carried `C…` handles**. Splitting them would put two toggles behind one
request and would make [#47](https://github.com/winniel123/verge-asm/issues/47)'s enablement prompt
render a distinction the operator cannot act on.

The two record kinds have genuinely different caveats — the /29 floor, and a name string typed by
the upstream ISP rather than by the RIR — and those are carried on the individual `Proposal`, which
records **which kind of record produced it**. That is what the operator needs in order to judge it,
and it keeps the caveat visible without splitting the source.

It **ships in v1**, on grounds worth stating plainly because the ticket invited the opposite:
[#15](https://github.com/winniel123/verge-asm/issues/15) already ships `entities?fn=` in the keyless
default set, and the customer objects arrive in a response verge-asm already fetches. Not shipping
would mean writing code to **discard** the majority of that response. This is not an addition; it is
ceasing to discard.

### A `Proposal` is retained in both directions

A confirmed `Proposal` is retained as provenance on the `Seed` it became, so *why is this here?* has
an honest answer one hop above the `Seed` — without disturbing `Citation`, which terminates at the
`Seed` exactly as [#7](https://github.com/winniel123/verge-asm/issues/7) requires.

A declined `Proposal` is recorded as a **`Seed` exclusion**, so it is not re-offered every cadence.
This looks like a suppression, which this project has refused twice
([#22](https://github.com/winniel123/verge-asm/issues/22),
[#29](https://github.com/winniel123/verge-asm/issues/29)), and it is not one:
[ADR-0006](./0006-subjects-leave-by-measurement.md) settled that the operator's *"this is not mine"*
is a `Seed` exclusion and never an `Annotation`, because it is a claim about where the estate ends.
Declining a proposed CIDR is precisely that claim.

**This extends `Seed` exclusions to address scopes.** `CONTEXT.md` defined them as *"exact names or
subtrees"*, and they now cover CIDRs too.

### Two stale registry clauses in `CONTEXT.md` are corrected

Both are consequences of #27 that were not carried into the glossary when it landed.

**`Vantage class`** said a class is re-verified each batch *"against the seeds **and registry ranges
the system already holds**"* — a live read of registry data by a safety-adjacent check, which
contradicts #27's *"read by nothing"*. It now reads `Seed`s alone. The consequence is stated rather
than buried: a vantage inside operator address space the operator never declared is classified
`internet`, which under [ADR-0010](./0010-exposure-composes-two-reaches.md) corrupts the internet
`Reach` leg and **over-reports `exposed`**. That is the loud failure and it is the intended one — a
false `exposed` gets investigated, a false `internal-only` does not — and the undeclared space
surfaces on [#22](https://github.com/winniel123/verge-asm/issues/22)'s `Coverage`, which is where
#27 already routed this same cost.

**`Ownership`** said it is *"computed against seeds **and registry data**"*. Withdrawn; `Seed`s
alone, per #27.

## Consequences

- **The three-property interrogation only applies to the fill half.** Any session reaching for a
  source's `authority` should first ask whether the thing **admits subjects**. If it proposes
  scopes, two of the three questions have no answer, and getting one anyway means inventing it.
  *Sharpened by [#56](https://github.com/winniel123/verge-asm/issues/56) /
  [ADR-0027](./0027-a-source-may-admit-without-observing.md), which found this bullet originally
  read "whether the thing produces observations" — one word wider than this ADR's own rationale,
  which argues throughout from admission.* Certificate transparency **admits** `Name`s and observes
  no facet, so the literal test would have made `crt.sh` a proposer and put 400 CT names behind
  ADR-0022's singular confirmation. A proposer fails **both** limbs — it admits nothing *and* its
  silence licenses nothing — so nothing about this ADR's ruling on registry paths moves.
- **`Proposal` is the third thing a registry path can be.** A registry source is now one of: a
  `Source` that observes, a proposer that yields `Proposal`s, or both — and the properties it
  carries follow from which.
- **ADR-0003 still governs proposers in full.** `consent` surviving means the four off-by-default
  registry outcomes ([#19](https://github.com/winniel123/verge-asm/issues/19),
  [#23](https://github.com/winniel123/verge-asm/issues/23),
  [#24](https://github.com/winniel123/verge-asm/issues/24),
  [#25](https://github.com/winniel123/verge-asm/issues/25)) are untouched by this ADR. Nothing here
  widens an aperture.
- **A volume constraint, not a consent question.** Turning `C…` handles into prefixes means one
  fetch per handle — 227 in the measured case — against a registry with no published rate limit.
  The permission is unchanged ([#15](https://github.com/winniel123/verge-asm/issues/15)); the
  request count is a constraint for whoever specifies the fetch.
- **[#40](https://github.com/winniel123/verge-asm/issues/40) inherits the `Seed` change.** It
  reconsiders the `Seed` primitive wholesale for a cloud-resident estate and should start from
  address-scope exclusions existing, not rediscover them.
- **`Proposal` has no timeline and cannot drift.** Being Declared, it is outside the comparison
  path entirely — so the ticket's question of whether a **stale** SWIP record is a value or a `Gap`
  dissolves with the rest. A stale record is a bad proposal, and a bad proposal is declined.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| A proposer is a `Source` whose properties apply degenerately | Leaves `authority` on a thing that admits nothing, which is an invitation for a later session to read it as one — and #27 and #28 had already cut this distinction twice without naming it |
| A proposer is a `Source`, but the glossary notes the properties are meaningful only for the fill half | Same defect, documented rather than fixed. One term still reads as two things, which is the reason `Host` was refused |
| A separate noun for the producer as well as the produced | Earns nothing `Proposal` does not already carry, in a glossary that refuses vocabulary on principle |
| Model the SWIP customer path as a second `Source`/proposer alongside org→prefix | They arrive in one response under one permission; two toggles behind one request, and a distinction the operator cannot act on in [#47](https://github.com/winniel123/verge-asm/issues/47)'s prompt |
| Do not ship the SWIP path — the ticket explicitly invited this | The request is already made and the data already returned; declining means writing code to discard it. [#38](https://github.com/winniel123/verge-asm/issues/38) also priced the PA-renter layer as the population beneath the 128,233 delegation holders, and ARIN is the one registry where it is keyless |
| Keep a declined `Proposal` as an `Annotation` | ADR-0006 settled that *"this is not mine"* is a boundary claim and belongs to `Seed` |
