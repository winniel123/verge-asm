# ADR-0023: `consent` names the door, never who walked through it

- **Status:** Accepted
- **Date:** 2026-08-13
- **Ticket:** [#52 Can an operator's own act move a source's `consent`, or is it fixed per instrument?](https://github.com/winniel123/verge-asm/issues/52)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)
- **Sharpens:** [ADR-0018](./0018-a-clear-conditional-is-not-an-ambiguity.md) §3 and §4, which stated this
  decision's second limb as a reason not to ship one instrument rather than as a property of the tier.

## Context

[ADR-0018](./0018-a-clear-conditional-is-not-an-ambiguity.md) broadened `operator-credentialed` to cover
**a grant the source actually made to that operator** — an API key or a countersigned agreement alike — as
against `operator-accepted`'s **reading the operator makes and bears**. That is settled and this ADR does
not reopen it.

It left a question neither ticket asked. `consent` is written once, on the instrument. The APNIC live
registry path makes the gap visible, and it is live in v1:
[#34](https://github.com/winniel123/verge-asm/issues/34)'s table says the operator at that toggle asserts
*"their own reading of the retrieval-system carve-out, **and whether they hold or will seek APNIC
approval**"*. Those are now two different values under ADR-0018. An operator who has merely made the
reading is `operator-accepted`. An operator who went and obtained APNIC's approval — a real, published,
per-party process ADR-0018 measured — is on the footing `operator-credentialed` describes, for the same
instrument, in the same install.

`CONTEXT.md` said `consent` *"decides what is in the aperture, which makes it a property of the observation
pipeline rather than of the deployment"*. If the value can move on an operator's act, the model owes an
account of what moves with it.

## What is actually in the spec

**[measured]** across the repository, 2026-08-13. Every non-`unencumbered` `consent` value in the v1 spec
sits on a **proposer** — the five registry paths of #34's table, every one of which
[#47](https://github.com/winniel123/verge-asm/issues/47) established *proposes* and none of which observes.

**`operator-credentialed` has no v1 instance at all.** Its occurrences are the glossary definition,
ADR-0018's ruling on the APNIC bulk file (which does not ship),
`docs/research/asn-less-operators.md` §13 restating that, and
`docs/research/non-arin-prefix-coverage.md` §2's list of the three tier names. The two candidates that
would have carried it are `unusable` (RIPE RIS raw data, §8.9) and unshipped (PeeringDB, §8 —
noted there as *"credentialed even for permitted users"* precisely because clause 7 makes registration a
gate).

So the question arrives against an empty set on the `Source` side, and against a class that
[#43](https://github.com/winniel123/verge-asm/issues/43) prices at zero on the proposer side. Both halves
are stated rather than smoothed, because the answer below does not depend on either and would be the same
if v1 shipped a dozen credentialed sources.

## Decision

### 1. `consent` names the door, never who walked through it

**`consent` is a property of the instrument, authored by the project, shipped in the release, and the same
for every install.** An operator's own act never *moves* it. The act **satisfies** it.

The ticket's question conflates two facts the model already holds separately:

- **which door the instrument sits behind** — `unencumbered` (no door), `operator-accepted` (the operator
  takes on a reading the project declined to make), `operator-credentialed` (the source granted this
  operator). This is `consent`.
- **whether this install has walked through it** — the **consent record** ADR-0003's second amendment
  specifies (the retrieved bytes, hash and timestamp, or the retrieval failure), or, for a credentialed
  instrument, the credential in use.

The second is per-install and always was: whether an operator holds an API key is deployment state, and
`operator-credentialed` has depended on it since [ADR-0003](./0003-third-party-source-consent-bar.md)
introduced the value. Nothing new is needed to carry an operator's act, and reading `consent` as the
carrier would give the model two representations of one fact — the defect
[ADR-0014](./0014-only-revealed-generalises.md) refused when it declined to name a `Gap`'s closing edge.

### 2. Only one of the three values reaches outside the install, and that is what decides mobility

The three values are not symmetric in what they assert about, and the asymmetry is the whole argument.

| Value | Asserts | Witnessed by the install? |
| --- | --- | --- |
| `unencumbered` | the project found nothing to clear | Project-authored, ships in the release |
| `operator-accepted` | the operator took on a reading and bears it | **Yes** — the acceptance act *is* the whole of the value, and #47's receipt records it |
| `operator-credentialed` | **the source granted this operator permission** | Only where the instrument enforces it |

`operator-accepted` is self-verifying. Since [#34](https://github.com/winniel123/verge-asm/issues/34) it
explicitly does **not** mean the operator certified they are inside the terms — it means they took on a
reading — so there is no external fact for the record to be wrong about. `operator-credentialed` is the one
value that makes a claim about a **third party's own conduct**, and an install can only know that claim is
true if the third party enforces it.

**So `operator-credentialed` is honest only where the instrument enforces the grant — where the request
fails without it, and the request succeeding *is* the evidence.** An unenforced grant recorded as this
value is [ADR-0003](./0003-third-party-source-consent-bar.md)'s rejected **declared-status bar**, which
*"invites operators to self-certify into permissions they may not hold"* and *"puts the project in the
position of having designed the loophole"*.

ADR-0018 §4 already said this, as a reason not to ship one instrument: *"Every other `operator-credentialed`
source in the spec is structurally honest: without the credential the request fails. `ftp.apnic.net` checks
nothing."* This ADR promotes that from a shipping ruling to a property of the tier, because the reasoning
never depended on which instrument it was applied to.

### 3. The general form: an operator may be taken at their word about themselves, never about a third party

The rule above has an obvious counter-example inside this map, and meeting it is what makes the rule
trustworthy rather than merely convenient.

[ADR-0013](./0013-custody-is-control-and-extends-by-declaration.md) §7 accepts an operator declaration the
software **cannot** verify: the `custody extension` cannot see an `ALIAS` flattened into an A record at a
zone apex, so the hazard rides on the declaration's wording. That is an unwitnessed declared claim, load
bearing for a safety property, and it was accepted.

The two cases part on **whose conduct is being asserted**.

- The custody extension is a claim about the **operator's own estate**, and
  [#28](https://github.com/winniel123/verge-asm/issues/28) settled that estate completeness is a thing *the
  operator is the only source for*. There is nobody else to ask. Taking their word is not a shortcut; it is
  the only instrument that exists.
- `operator-credentialed` is a claim about **APNIC's conduct**, and APNIC is right there. Where the source
  can be made to answer — by enforcing — an operator's assertion about it is a guess wearing a record's
  clothes.

So: **an operator may be taken at their word about themselves, and never about a third party.** This is
[ADR-0003](./0003-third-party-source-consent-bar.md)'s founding move pointed the other way. That ADR
refused to read a stranger's terms on the operator's behalf; this refuses to let the operator assert a
stranger's permission on the project's behalf. `Vantage` is the same instinct a third time — *declared as
intent and re-verified every batch rather than trusted as configuration*.

### 4. The forcing case dissolves, and the operator loses nothing

APNIC's **live registry path** is served anonymously — that is the premise of every keyless measurement in
[#20](https://github.com/winniel123/verge-asm/issues/20) and
[#38](https://github.com/winniel123/verge-asm/issues/38), and it is why the path's defect is an HTTP 200
with an empty array rather than a 401. It checks nothing, so it **can never carry
`operator-credentialed`**, and an operator who obtains APNIC's approval stays `operator-accepted`.

That is not a demotion, and this is the part worth stating plainly because the ticket reads as though it
were. `operator-accepted` was never a statement of doubt about the operator's position — post-#34 it says
the reading is theirs to make and bear. An operator holding APNIC's approval makes that reading trivially:
#47's group-1 row asks *whether APNIC has approved, or would approve, your use*, and they can answer it
**yes, for me, verifiably**. The value already describes them correctly.

**What the operator's act moves is a group-1 row from open to closed — not the value.** #47 already put the
right container under it: the consent record *"holds what the operator was shown, not only what they
accepted — the two groups' contents at the time"*. The act lands there, in the operator's own answer to a
question the prompt already asks, and needs no new machinery.

### 5. The record stores no assertion of a grant

[#34](https://github.com/winniel123/verge-asm/issues/34) requires the record to store the retrieved terms
or the retrieval failure. It does **not** gain a slot for *the operator says they hold APNIC approval*.
Storing that would be the declared-status bar relocated into the record, and it would read as the thing
that moved the value. It is also the shape #34 refused when it declined a fourth `Consent` value: a fact
about somebody's assessment does not belong on the object that answers *whose permission does this run on*.

**A grant with no artefact is therefore recordable exactly when the source enforces it**, because then the
artefact is the working request. A countersigned agreement that the source enforces is recordable;
ADR-0018's broadening is correct as a definition. A countersigned agreement the source does not enforce is
not, and in v1 that describes every instance of the class — which is why the tier's v1 population is empty
rather than merely small.

### 6. The enablement prompt does not ask which of the two the operator is doing

There is one act at the toggle and the prompt renders one control, exactly as
[#47](https://github.com/winniel123/verge-asm/issues/47) ruled. A fork asking the operator whether they are
accepting a reading or exercising a grant would be the operator selecting their own tier, which is the
declared-status bar with the choice made explicit, and it would offer a tier the instrument cannot honour.

One clause is owed, and it is a clause rather than a block. #47's APNIC group-1 row invites the act —
*"You can put that question to APNIC as one of their account holders"* — without saying what the act does.
Under this decision it does nothing to the software, so the row says so: **holding the approval answers the
question for the operator and changes nothing the software does, because the query is served to anyone and
the approval is not something the software can see.** Left unsaid, the row reads as an offer the product
cannot honour, which is [#22](https://github.com/winniel123/verge-asm/issues/22)'s invitation test failing
one row down.

## Consequences

- **`consent` is spec state, not install state, and a release is the only thing that moves it.** A source
  whose terms change, or an instrument that starts enforcing, moves its value in a release. The cost of
  that lands where ADR-0003 already put it — on the **default**, since *"changing a source's default state
  in a later release is an aperture change for every existing install"* — and never on the value itself.
- **For a proposer the move is free, and that is a smaller result than it looks.** ADR-0012, #43 and
  ADR-0003's third amendment price a proposer's addition, removal or state change at no `revealed`, no
  `Break`, no version bump. Since every non-`unencumbered` value in v1 sits on a proposer, **no consent
  value in the v1 spec has an aperture cost at all**. The ticket expected this escape to make the proposer
  half short and the `Source` half interesting; it does neither, because the question turns on **what a
  value asserts**, not on what enabling costs, and that answers identically for both halves.
- **`CONTEXT.md`'s aperture clause was stale and is corrected.** *"It decides what is in the aperture"* has
  been false for proposers since [ADR-0012](./0012-a-proposer-is-not-a-source.md), and #47 withdrew the
  same claim from ADR-0003 without reaching the glossary — the second time a rename outran a claim about
  what the thing *does*. The entry now splits the two cases and keeps *"a property of the observation
  pipeline rather than of the deployment"*, which this decision strengthens rather than weakens.
- **A test for any property this map adds later**: ask whose conduct each value asserts. A value that
  asserts a third party's conduct needs that third party to enforce it, or the model is carrying a guess.
  Applied to what exists, `authority`, `completeness` and `Custody` all pass — they assert about the
  project, the operator, or a measurement we took.
- **This does not reopen #34's five-registry table**, which was never in question, and adds no fourth
  `Consent` value, which #34 and ADR-0018 have now each refused once.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **One toggle in two positions** — the operator selects `operator-accepted` or `operator-credentialed` at the prompt | The operator choosing their own tier is ADR-0003's rejected **declared-status bar** with the choice made explicit rather than inferred. It also offers a position the live registry path cannot honour, since APNIC's query is served to anyone |
| **Two enablement paths** — a second toggle for the credentialed route to the same registry | Two toggles behind one anonymous endpoint, which is [ADR-0012](./0012-a-proposer-is-not-a-source.md)'s own rejected alternative verbatim: *"two toggles behind one request, and a distinction the operator cannot act on in #47's prompt"*. The distinction is real in the world and unactionable in the software, which is the definition of a control that should not exist |
| **`consent` is per-install mutable state** — the value is deployment state the operator's act writes | Attractive because `operator-credentialed` already depends on an install-local key, and wrong about which fact that is: the key satisfies the door, it does not choose which door there is. Making the value mutable gives the model two representations of one fact and puts the operator's assertion about a **third party** into a field the software reads |
| **Record the grant without gating on it** — store *the operator holds APNIC approval* in the consent record and leave the value alone | The honest-looking middle, and it is the declared-status bar relocated one object along. #34 refused a fourth `Consent` value on the ground that reaches this too: a fact about somebody's assessment does not belong on the object answering *whose permission does this run on* |
| **Rule the question free on the proposer escape and stop** — the ticket's own suggested shortcut | The escape holds and answers only the **cost** limb. Stopping there would leave *can it move* unanswered for a `Source`, and would have answered it by accident and wrongly if v1 ever ships one — the escape is about aperture, and the answer is about evidence |
| **Narrow the ruling to APNIC** | #34's own lesson, one ADR along: an answer scoped to APNIC leaves the same defect standing under every other toggle. The defect here is not APNIC's, it is the tier's |
