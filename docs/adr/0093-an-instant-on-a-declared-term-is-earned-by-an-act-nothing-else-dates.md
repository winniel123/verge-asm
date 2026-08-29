# ADR-0093: An instant on a Declared term is earned by an act nothing else dates — and every Declared act but one is dated by what it moved

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#165 Is a declaration instant a field the Declared layer may carry at all?](https://github.com/winniel123/verge-asm/issues/165)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0073](./0073-an-operator-dial-carries-no-author-however-specific-its-target.md) §2 kept the
declaration instant on `Annotation` and struck the author beside it, and closed with a sentence that
is a rule in everything but name:

> *"`Annotation` is consequently the only Declared term carrying an instant. That is admitted rather
> than hidden, and it is the same asymmetry that already earns the object its name: it is the only
> Declared term carrying operator prose."*

Two things are wrong with leaving that where it stands, and they pull in opposite directions.

**The ground it gives is not the ground it used.** §2's actual argument is *"it is not a timeline: a
timeline is a sequence of `Span`s and needs two values, and this has one, with no successor and
nothing to difference against"*. That argument is about a **field**, and a field does not know which
term it is on. Read alone and in the present tense it licenses an instant on `Seed`, on `Scan`
config, on a `Channel` — on anything in the layer. The *prose* clause then arrives one sentence later
as the limit, and prose is a correlate of the real discriminator rather than the discriminator
itself. A Declared term that acquired operator prose tomorrow would inherit an instant it has not
earned, and the corpus is already one drawing away from that: `prototypes/seeds/` renders a free-text
`why` on every exclusion row, beside a rendered date.

**And the claim of uniqueness was already false on `main` when it was written.** ADR-0073 was drawn
against `prototypes/signals-annotated/` and did not open the others. Two prototypes render a
declaration instant on a Declared term today:

- **[`prototypes/seeds/`](../../prototypes/seeds/index.html)** renders `declared <month>` in every
  seed block's header, `declared <month>` on every exclusion row, and a date column in the declined
  region.
- **[`prototypes/source-enablement/`](../../prototypes/source-enablement/index.html)** renders a
  consent record carrying `accepted at 2026-08-13 14:02:31 UTC` — **and `accepted by l.winnie`
  beside it**, which is [#127](https://github.com/winniel123/verge-asm/issues/127) and ADR-0073 §1's
  *"no account, no name, no initials, no avatar, no 'declared by' cell — not stored and not
  rendered"* drawn on the rendered surface.

Neither prototype ever *ruled* that the field exists — a prototype is a dated record of a reading,
never of a rule ([ADR-0075](./0075-a-prototype-is-a-dated-record-of-a-reading-never-of-a-rule.md)) —
but both are drawn states a session would build from, and both are evidence that the layer-wide
question is live rather than academic.

The question is therefore not *may `Annotation` keep its instant* — ADR-0073 settled that and nothing
here reopens it. It is **what earns it**, stated as a test the next Declared term can be put to.

### What binds and is not re-derived

- **[#127](https://github.com/winniel123/verge-asm/issues/127)** established that every Declared term
  holds a current value with no timeline, that a `Seed` is **edited after it is created** —
  *"exclusions are added, a scope is narrowed"* — and that the estate's own data dates a `Seed`:
  **the earliest `revealed` beneath the scope**.
- **#127's out-of-scope entry** refuses the reconstructible half **on principle** — *a second history
  over the one layer defined as not drifting* — and reopens on nothing short of the Declared layer
  acquiring a timeline.
- **[#6](https://github.com/winniel123/verge-asm/issues/6)'s** *every seam is a place drift can be
  manufactured* is the hazard, and it is priced in §3 rather than invoked.

## Decision

**An instant is not a field the Declared layer may carry.** It is a per-term exception, earned on a
**conjunctive two-limb test**, and `Annotation` is the only term in v1 that passes it.

### Limb 1 — the term is replaced, never edited

The instant must have **one value for the object's whole life**, acquiring no successor. This is
ADR-0073 §2's premise stated as the condition it actually is: *one instant with no successor* is not
a property of instants, it is a property of **objects that are never mutated**.

Two Declared terms are in that class, and both by an explicit ruling rather than by habit:
`Annotation` — *"editing one is a **new** `Annotation`, as with `Proposal`"* — and `Proposal` —
*"a record re-offered with different contents is a **new** `Proposal`, never an existing one
changed."*

On an edited term the field has only two futures and both are refused already:

- **It stays put and goes stale.** That is the **declared-status bar
  [ADR-0003](./0003-third-party-source-consent-bar.md) rejected**, and it is verbatim the ground
  #127 §3 used to kill the minimal attribution field on `Seed`: *"A field that is true at creation
  and silently stale afterwards asserts more than it carries."*
- **It moves on the edit.** Then a Declared term holds a value with **two readings over time**, which
  is a timeline on the one layer defined as not drifting — #127's out-of-scope entry, refused on
  principle, reopening on nothing less than a different model.

### Limb 2 — the act leaves no dated residue, and a named v1 reader needs it dated

Nothing else in the Observed or Operational corpus may already date the act, **and** a consumer that
exists in v1 must need the date. Both halves are required: a wish is not a reader, and #127 §4 is the
rule — *"If it is worth writing it is worth rendering; if it is not worth rendering it is not worth
writing."*

### The enumeration, and what the limbs do to it

Every operator act the Declared layer records, and what dates it:

| Declared act | Dated by | Limb 1 | Limb 2 |
| --- | --- | --- | --- |
| Declare or widen a `Seed` | `revealed` plus one coverage-class message at the scope (ADR-0047); afterwards, the **earliest `revealed` beneath the scope** (#127) | **fails** — edited | **fails** |
| Narrow a `Seed`, or add an exclusion | **one coverage-class message at the scope** ([#130](https://github.com/winniel123/verge-asm/issues/130) · [ADR-0074](./0074-an-aperture-narrowing-that-takes-its-carrier-with-it-fires-at-the-scope.md)), and a `Message` carries **the instant of the cause** | **fails** — it *is* the edit | **fails** |
| Declare a `custody extension` | the gate opens; `revealed` on what it now covers | **fails** — on/off | **fails** |
| Withdraw a `custody extension` | the gate closes and currency opens a `Gap` *naming the operator's own act* — coverage member 3 | **fails** | **fails** |
| Enable or disable a `Source` | the `Batch`'s **recorded source set** — [#8](https://github.com/winniel123/verge-asm/issues/8) and #127 §2: *"a flipped default appears in the recorded source set exactly as an operator toggle does"* | **fails** — toggled | **fails** |
| Confirm a `Proposal` | it becomes a `Seed`; row 1 | **passes** | **fails** |
| Decline a `Proposal` | it becomes a `Seed` exclusion; row 2 | **passes** | **fails** |
| Declare a `Vantage` | first of a class: `revealed` plus one coverage-class message; and it is re-verified **every `Batch`** | mixed | **fails** |
| Change `Scan` config | the next `Dispatch`, and the `Batch`'s **recorded completed scope** — ADR-0014's aperture-detection input | **fails** — edited | **fails** |
| Create or edit a `Channel` | a `Delivery`, on the next message of a class it receives | **fails** — edited | **fails, conditionally** |
| Declare an `Annotation` | **nothing** | **passes** | **passes** |

`Annotation` is alone because **its whole effect is a message that does not fire**. A `not-fired` →
`fired` `Transition` on an annotated pair is *recorded and is not a message*. The acceptance's only
consequence is a silence, and a silence has no instant. ADR-0073's own variant C lost on exactly this
fact — an ordering by recorded firings *"goes **empty on a quiet estate** while four acceptances
stand"* — so an `Annotation` can live its entire life with **no dated residue anywhere in the
model**, while ADR-0073's reviewability reader needs it dated: *"a mute with no stated reason is one
nobody can review later"*, and an undated reason is not reviewable either, on an object with **no
expiry by rule**.

**So the rule is not that prose earns the instant. It is that a silent act earns it, and `Annotation`
is the only act in the model whose product is a silence.**

### `Channel` is the near-miss, and it is named rather than hidden

`Channel` fails limb 1 on the glossary's default reading — nothing states that a `Channel` is
replaced rather than edited, and its URL, secret and class subset are plainly editable — but that
reading is an inference from silence rather than a ruling, and it is the thinnest cell in the table.
Its limb-2 residue is **conditional**: a `Channel` created and never delivered to has none. It fails
anyway on limb 2's second half, which is not conditional at all — **nobody has named a reader**, and
#127 §4 governs. Nothing turns on the mutability question: were a later ticket to rule a `Channel`
replaced rather than edited, it would still carry no instant.

### What this does *not* do

- **`Annotation`'s instant is confirmed, not re-decided.** ADR-0073 §2 stands in outcome. Only its
  stated ground moves.
- **No actor anywhere.** #127 and ADR-0073 §1 are untouched, and this ADR adds no field to any
  Declared term.
- **No operator-act record.** Minting a dated record of a `Seed` declaration is #127's refused
  operator-act record with the actor column left out. It is out of scope, and it stays out on #127's
  own reopening condition.

## Rationale

### The option that lost: generalise — the layer may carry an instant on any term

This is the reading ADR-0073 §2's text supports on its face, and it was argued at strength before it
was refused.

- **The argument §2 actually gives has nothing to do with prose.** A timeline needs two values. One
  instant has none to difference against. That is true of an instant on a `Seed`, a `Channel` and a
  `Scan` exactly as it is true of one on an `Annotation`. Restricting it to the one term the drawing
  session happened to have open is arbitrary.
- **The layer's charter is not violated.** *Declared: what the operator tells us — no, it is input.*
  The prohibition is on **comparison**, and a single scalar cannot be compared. There is no rule to
  break.
- **The asymmetry reads backwards.** `Seed` is the highest-stakes Declared act in the product — it is
  what opens the probing gate — and under the incumbent it is the one act that cannot say when it was
  declared, while a mute can.
- **Two independent sessions already drew it**, in `seeds/` and `source-enablement/`, without either
  noticing it needed a decision. When drawings converge unprompted, that is evidence.
- **It costs one non-derived column per term**, which is close to free.

**It loses on three things.**

**1. It misreads its own premise.** *One instant with no successor* holds only for an object that is
never mutated, and `Annotation` and `Proposal` are the only Declared terms with that property — both
by an explicit ruling. Every other Declared term is edited in place, **by name in the glossary or in
#127's own text**. On those terms the generalisation does not deliver a single un-differenceable
instant at all. It delivers either a stale field ADR-0003 already rejected, or a moving one, which is
the Declared layer acquiring a timeline. On `Seed` the generalisation is therefore not merely wrong,
it is the scope change #127's out-of-scope entry refuses on principle — arriving as a field rather
than as a proposal, which is exactly how that entry warns it would arrive.

**2. #6's hazard, priced.** A `Seed` instant is a **second answer to a question the estate already
answers**. #127 established the first: the earliest `revealed` beneath the scope. The two disagree by
construction — the Declared scalar is fixed at first declaration while the measured one moves as the
scope is narrowed, re-widened, excluded from and re-measured — and **the Declared one is the one a
drift query will join to**, because it is a scalar on the row rather than an aggregate over
timelines. That is a seam between two datings of one act, on the **input side**, which #127 §5
already identified as where it is worse: *"one `WHERE seed_version = …` in a drift query is `ScanRun`
back through the front door, on the input side, where it is worse — because Declared inputs gate
probing."* #6's rule is that every seam is a place drift can be manufactured. This is a seam whose
two sides are answers to one question and neither is labelled as which.

**3. The consumer test, on `Channel` and `Scan` config.** ADR-0073 kept `Annotation`'s instant
because *rider 6's own justification requires it* — a cited, existing reader. No sentence anywhere in
the corpus requires a `Channel`'s or a `Scan`'s date. #127 §4 governs and it is quoted above.

**And the drawings do not rescue it.** ADR-0075 limb 1 and limb 2 exist precisely to stop a drawing
being read as a ruling. Neither #123's `seeds/` nor #47's `source-enablement/` ever put the field to
a decision. Both reached for it as scaffolding, which is what makes them a **sighting** rather than a
precedent. That they converged is real evidence of what feels natural, and it is answered rather than
waved past: what feels natural is *the surface should say when this happened*, and under this ruling
it still can — from `revealed`, from the coverage-class message, from the `Batch`. What it may not do
is **assert the date as a property of the declaration**.

### The option that lost: strike `Annotation`'s instant too, for uniformity

The clean layer-wide *no*. It loses on ADR-0073's own ground, one day old and unweakened: striking it
leaves a paragraph of operator prose with no indication of when it was written, on an object with no
expiry, so *standing mutes nobody can even judge the age of*. Nothing in this ticket supplies a new
reason to revisit that, and uniformity for its own sake is what §2 was right to refuse.

### The option that lost: mint a dated record of the declaration act instead

Put the instant on the Operational side — a row per operator act, carrying the act and the time and
no actor — and the Declared terms stay clean.

**Refused, and not by this ADR.** That is #127's operator-act record with one column left out, and
#127 ruled it out of scope: *"what is out is the **record**, not the screen."* Its deferral is on a
consumer that does not exist and its reopening condition is the spec admitting a second party who can
mutate. Leaving the actor off changes neither. Recorded here because this is exactly where a session
would reach for it, and it is a **scope change** rather than a design choice.

### The option that lost: rule per surface rather than per term

*Let each screen decide what it dates.* It loses on the same ground ADR-0073 §6 found by drawing:
rendering is a claim. A date beside a `Seed` says the product **holds** that date, and there is no
rendering of a field the store does not have. Per-surface is how the two prototypes acquired the
field in the first place.

### Where the ground is thin

The **reader** half of limb 2 is a judgement, not a measurement. *Nobody has named a consumer for a
`Channel`'s date* is true of the corpus today and is not proof that none exists. It is the same shape
of ground #127 and ADR-0073 both flagged, and it is flagged here for the same reason. The mitigation
is the same too: limb 2 is a conjunction, and every term that fails on the reader also fails on the
residue or on limb 1, so **no cell in the table turns on the thin half alone**.

## Consequences

- **[`CONTEXT.md`](../../CONTEXT.md) gains a Declared-layer preamble** stating the rule, and the
  `Annotation` entry's instant clause gains the ground. The layer had no preamble and the rule has no
  per-term home, which is what let the question stay invisible — the same silence ADR-0073 named when
  it added the author clause.
- **[ADR-0073](./0073-an-operator-dial-carries-no-author-however-specific-its-target.md) is amended
  at two sites** under [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)
  — §2's closing paragraph and the Consequences bullet. Read alone and in the present tense, both
  would cause a competent session to give a new prose-carrying Declared term an instant. **The
  outcome is confirmed. The ground is replaced**, per ADR-0057 §5 — a withdrawal supplying no
  successor does not hold.
- **Two prototypes are marked and neither is redrawn**, per ADR-0075 limb 2 and limb 3, both being
  files this pass held open. `prototypes/seeds/` — a rendered `declared <month>` is a field the
  product does not have. `prototypes/source-enablement/` — the consent record's `accepted at`, and
  **`accepted by l.winnie` beside it**, which was already condemned by #127 and ADR-0073 §1 before
  this ticket existed and had gone unmarked. The retrieval receipt's own `requested … at` is
  **untouched and correct**: it dates *our* fetch, which is an operational fact about the project's
  conduct, not a property of the operator's declaration.
- **The model's existing operator-act instant is confirmed as correctly placed.** A `declared`
  source's observation takes its instant from **the operator's supply act**, and that instant sits on
  `Observation` — Observed — where the currency bound reads it. That is the pattern this ADR
  generalises: **an act's instant lives on the record of what the act moved, never on the declaration
  itself.** It is also why an on-disk zone path is refused — *a mount has no supply act, so it has no
  instant*.
- **No number in the corpus moves.** No aperture input, no message class, no census, no rule count.
  Nothing is stored that was not stored, and nothing that was stored is removed.
- **A stated, unfixed cost.** v1 cannot answer *when did I declare this scope* from the declaration
  itself. It can answer it from the earliest `revealed` beneath the scope, which is a different fact
  with a different failure mode — it dates when we **first saw** something there, not when the
  operator **typed** it, and on a scope that has never yielded a subject it answers nothing at all.
  That gap is real and is not repaired.
- **Reopens** where a Declared term arrives that is **replaced rather than edited** and whose act
  leaves **no dated residue** — the test is the reopening condition, and it is designed to be applied
  rather than argued. A second term passing both limbs is an addition under this rule, not an
  exception to it.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Generalise: any Declared term may carry an instant**, since ADR-0073 §2's not-a-timeline argument is about the field rather than the term | The premise is *one instant with no successor*, which holds only for an object never mutated. Every Declared term but `Annotation` and `Proposal` is edited in place, so the field is either stale — ADR-0003's declared-status bar, #127 §3's own kill — or moving, which is a timeline on the layer defined as not drifting |
| **`Annotation`'s prose is what earns it**, as ADR-0073 §2 says | Prose is a correlate, not the ground. A future prose-carrying Declared term would inherit the instant, and `prototypes/seeds/` already draws free-text `why` on an exclusion beside a rendered date. The real discriminator is that an `Annotation`'s act produces a **silence**, which nothing dates |
| **Put an instant on `Seed` alone**, it being the highest-stakes act | The one term where #127 already supplies the date from the estate — earliest `revealed` beneath the scope — so it is a second answer to one question that can disagree with the first, on the input side, which #127 §5 calls the worse side. #6's seam, exactly |
| **Strike `Annotation`'s instant for uniformity** | ADR-0073's own refusal, one day old: an undated standing mute on an object with no expiry cannot be judged at all, which defeats rider 6's reason for requiring a stated reason |
| **Mint a dated operator-act record and keep the Declared terms clean** | #127's refused record with the actor column removed. Out of scope on #127's ruling and reopening condition, and a scope change rather than a design choice |
| **Let each surface decide what it dates** | Rendering is a claim (ADR-0073 §6). There is no rendering of a field the store does not hold, so per-surface *is* per-term with the decision hidden — which is how both prototypes acquired it |
| **Take the two prototypes as precedent** | ADR-0075: a prototype is a dated record of a reading, never of a rule. Neither drawing put the field to a decision, which makes them sightings — and one of them also renders `accepted by`, which no reading of the corpus permits |
