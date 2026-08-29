# Derivation versions move on content, and a Break clamps the horizon

- **Status:** Accepted
- **Date:** 2026-08-13
- **Ticket:** [#18 How does a Derived value's effective version avoid moving every release?](https://github.com/winniel123/verge-asm/issues/18)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0007](./0007-drift-is-a-timeline-of-spans.md) made comparison legal only within one
derivation and enforced it with a `Break` rather than by discipline. It also named the bill it
had run up. An `Exposure` span's effective version composes at least four inputs — the exposure
rule, `Availability`'s version, the `reachability` canonicaliser, and the staleness bound `k` —
and [ADR-0004](./0004-signals-are-release-coupled-rules.md)'s composition rule makes each
compose automatically. A release touching any one breaks **every `Exposure` span in the estate
at once**. The exposure board is a matrix over a time window, so one break inside that
window was taken to make the whole board not-comparable.

Read that way the model gets *worse as it gets more correct*: every newly composed input is
another release-coupled trigger, and the screen carrying the entire differentiation spends its
life saying *not comparable* — honest and useless.

The composite cannot be shrunk. Each of the four genuinely changes what an `Exposure` span
concludes:

- the rule obviously
- `Availability`'s window because it decides whether exposure is constructible at all
- the `reachability` canonicaliser because it decides what a connect outcome *is*
- `k` because it decides where a `Gap` opens, which decides where spans close

So there is no composition trick that drops an input, and the two available levers are **how
often the composite moves** and **what a `Break` actually costs**.

## Decision

| Concern | Decision |
| --- | --- |
| What moves a version | An **output-affecting change only** — never because a release shipped |
| What stops it moving | A **golden corpus in CI**, over every named versioned derivation, **per rule** |
| Version shape | A **vector** of named component versions — one leaf per named `Derivation` |
| Composition | Absorb the **whole vector** of every derivation read, **flattened and deduped** |
| Where it lives | A `derivation` row; every `Span` carries a reference to it |
| Break scope | **Every span of that derivation, uniformly** — no predicate, no recompute-to-test |
| Break storage | **None** — the edge and its component diff are derived on read |
| A view across a break | **Clamps its horizon** to the most recent break; the census still renders — but a census does **not** survive the derivation's precondition failing |
| Duration and counts | **Never cross a break**; render as a **labelled floor** |
| Alerting across a break | One **re-baseline message** per alerting derivation whose vector moved |
| The difference set | A **description computed once at the cause** — never a `Transition` |

## Rationale

### "Release-coupled" was a ceiling, not a coupling

[ADR-0004](./0004-signals-are-release-coupled-rules.md) used the phrase to mean that a rule's
reference data may change **only** at release boundaries — an upper bound on change frequency,
and the property that separates a `Signal` from a signature database. This ticket read it as
*a version moves with each release*, which is a different and far worse claim, and most of the
alarm follows from the misreading. A version tracks the **content** of a derivation. The
release cadence is only the ceiling on how often that content is permitted to move.

With that removed the arithmetic changes. `k` is fixed at 2 and nothing has proposed moving it.
The availability window is fixed on [ADR-0005](./0005-scan-execution-model.md)'s blast-radius
grounds. The `reachability` canonicaliser maps a small closed value space — unlike the
certificate canonicaliser, which is genuinely churny and does not feed `Exposure` at all. The
exposure rule is a five-state table settled in [#14](https://github.com/winniel123/verge-asm/issues/14).
That is one or two bumps in the first year and close to none after.

The churn that is real lands elsewhere. [#21](https://github.com/winniel123/verge-asm/issues/21)
measured attesting sources moving much faster than release-coupling assumes — BOD 22-01 revoked
mid-2026, CISA's CPG 2.W renumbered into 3.S, CIS abandoning named ports for an abstraction,
Prometheus *softening* its position after gaining native TLS. All of it lands on the
sensitive-port list, which breaks `sensitive-port-exposed` and nothing else.

So the problem is real but concentrated, and it is solved **once** rather than twice: one
mechanism, calibrated on the case that actually bites.

### A version moves on output, and a corpus is what says so

The ticket raised *bump only on output-affecting change* and dismissed it as "the kind of
discipline this project has consistently refused to rely on". That is right about discipline
and wrong about the options, because between discipline and automation sits a mechanical gate.

Neither obvious instrument tracks the right thing. A content hash of the derivation's **code**
bumps on every refactor, breaking the estate for a rename. A content hash of its **declared
parameters** is silent on a behavioural bug fix in code — the case that most needs to break.
What we care about is neither: it is the derivation's **output**.

So the version is hand-maintained per derivation, and CI holds a checked-in **golden corpus** —
fixed observations with their expected derived outputs. If an output moves and the version did
not, the build fails. The judgement stays human, because no machine knows that a refactor was
*meant* to be behaviour-preserving. The **check** is mechanical. That is the same move
[ADR-0007](./0007-drift-is-a-timeline-of-spans.md) made in enforcing comparability structurally
rather than by rule. This project has refused discipline throughout the comparison path. It has
never refused a failing test.

The corpus covers **every** named versioned derivation, not only the span-producing ones, and
per rule, matching ADR-0004's per-rule versioning. For a signal a row is *these observations →
fires / does not fire / `not-evaluable` / outside the domain* — the fourth outcome added by
[ADR-0024](./0024-a-rules-domain-is-the-extension-of-its-name.md), which put the rule's
`Predicate domain` inside the rule's own leaf, so a domain edit moves a row and must move the
leaf. For the sensitive-port list the output is which
`(port, transport)` pairs the rule fires on, generable from
[#21](https://github.com/winniel123/verge-asm/issues/21)'s cited table. Scoping the gate to
spans alone would leave the churniest thing in the product ungated, which is precisely
backwards.

### The version is a vector, and its leaves are derivations

A composed scalar can say *something moved* and nothing more.
[#22](https://github.com/winniel123/verge-asm/issues/22) settled that the not-comparable
treatment is **one visual treatment carrying stated reasons**, and a break that cannot name its
cause cannot meet that.

So the stored thing is a **vector of named component versions**, held in one `derivation` row
that every `Span` references. Composition is by absorbing the whole vector of every derivation
read, flattened and deduped:

- `Exposure` → `{exposure, availability, reachability-canon, currency}`
- `sensitive-port-exposed` → `{its own rule, sensitive-list}` ∪ `Exposure`'s

One leaf per **named derivation**, not one per knob: a derivation's version covers its rule and
its declared parameters together, so the availability window sits *inside* `availability`
rather than beside it, and `k` sits inside `currency` — which every span in the model composes,
including one folded straight from observations.

Flattened rather than nested, so equality stays a set comparison and a break names a **leaf**.
*The reachability canonicaliser moved* is a sentence an operator can act on. A path through the
composition graph is a stack trace.

The component set is declared **per derivation**, never as one global schema. A shared schema
means adding a component perturbs every derivation at once — which is the set-wide version
ADR-0004 refused, arriving through another door.

The `Break` itself **stores nothing**. Both spans carry a reference to their `derivation` row,
so the edge and its component diff are computed on read. That is the treatment ADR-0007 already
gives `Transition`, for the reason it gave: storing it is a second representation of one fact.

### A break is uniform, because the alternative fails silently

With a vector in hand a narrower break becomes expressible: a release could ship a predicate —
*this change can only have moved spans whose value was `edge-only`* — and write breaks only
where it matches. It is the largest available reduction in break volume, and it is refused.

Weigh the failure modes against each other rather than the savings. A break that fires too
widely fails **loudly**: the board clamps, names the leaf that moved, and recovers within a
cadence. A predicate that is too narrow fails **silently**: we compare across a real derivation
change and emit a `Transition` that is an artefact of our own release, estate-wide — the single
failure this entire apparatus exists to prevent. The corpus catches a wrong predicate only
where it happens to cover that case.

The neighbouring lever goes the same way. *Recompute history to test whether it would have
differed, without storing the recomputation* makes comparability a function of today's release,
which is ADR-0007's stated objection to re-derivation moved one level up and no better for the
move.

What makes uniformity affordable is the next section: a break costs one cadence, not a dark
screen. The predicate would buy a large reduction in a cost that is no longer large, and attach
a silent failure mode to it.

### A break clamps the horizon rather than blanking the board

The ticket assumed a break inside the board's window makes the whole board not-comparable. That
does not follow. A `Break` is an **edge on a timeline**, not a loss of data — what it withdraws
is the licence to reach *back across* it.

So a view bounds its horizon by the most recent break on the timelines it reads: *change since
the derivation upgraded four hours ago*. Immediately after a break no timeline has a
predecessor under the new derivation, so the matrix has no rows — but the **current-state
census renders in full**, because a census is not a comparison. The board degrades to *here is
where you stand; change resumes at the next cycle*, and is a matrix again after one cadence.

This is not a new rendering. [#22](https://github.com/winniel123/verge-asm/issues/22)
established that **day one and degraded day are the same object** — "no internet vantage, so
exposure cannot be constructed" and "vantage unavailable for three days, so exposure cannot be
constructed" are one statement differing only in whether the capability was ever present. A
derivation upgrade is a third instance of it: no history under this derivation *yet*. Built
separately, the post-upgrade state — the one that appears while the operator's attention is on
the upgrade — would get the least care of the three.

Breaks are also **per timeline**. A change to the certificate canonicaliser breaks certificate
timelines and leaves the board untouched. Only a move in one of `Exposure`'s own four leaves
clamps it.

### A census survives a break; it does not survive a precondition failing

*Added by [#28](https://github.com/winniel123/verge-asm/issues/28), which rendered both states
side by side and found that "a census is not a comparison" is true and not sufficient.*

"The census still renders" above is a claim about **breaks**, and it must not be read as a claim
about degradation generally. A census renders across a break because every subject still holds a
value — the break withdraws the licence to compare two of them, not the licence to read one. That
survives because the derivation still *ran*.

It does not survive the derivation being **non-constructible**. When an internet `Vantage` is
unavailable, `Exposure` has no definition ([#14](https://github.com/winniel123/verge-asm/issues/14)),
so there is no state for any subject to hold and nothing to count. The honest rendering is not a
census reading zero of each state — it is the absence of the census, beside the subject count that
*is* still true: *1,290 services are in the estate; none of them holds an exposure state.* A board
that answered "0 exposed" there would be asserting the strongest possible claim about a thing it
had not measured, which is the failure the no-false-absence rule exists to prevent, arriving
through the census instead of through a `Batch`.

The general form, which binds any surface rendering a Derived value:

> A **census** of a Derived value survives a `Break` and does not survive that value's precondition
> failing. A `Break` costs you the comparison. A precondition failing costs you the value.

One consequence for the census itself: where a comparison view partitions subjects into a matrix
plus a not-comparable band, the census must be **derived from that partition** — the states plus a
`no reading this cycle` row — rather than counted independently. Counted independently it silently
disagrees with the board whenever a `Batch` dead-letters, and the two numbers are then both
presented as facts.

### Duration does not cross a break, and has to say so

This is the real cost, and it is not the board. ADR-0007 made duration the concrete form of the
differentiation — *exposed for eleven days*, *flapped forty times this week*, the sentences no
count-shaped diff can produce — and a break truncates every one of them in the estate at once.

Worse, it truncates them **plausibly**. After an upgrade *exposed for 4 days* reads as a fact
rather than an artefact, and the operator has no way to tell the difference. Strictly we cannot
even assert the value was unchanged across the break, because that assertion *is* the forbidden
comparison.

So duration never crosses, and every truncated duration or count renders as a **floor with its
reason attached**: *exposed for at least 4 days (derivation changed 4 days ago)*, *flapped 12
times in the 4 days since*. A bare understated number is a lie the operator cannot detect. A
labelled floor is true. This is the one place a derivation upgrade genuinely degrades the
product's headline claim, and it degrades it for a full window rather than for a cadence.

### The alert hole, and what may be said across a break

The clamp rescues the board and not the alert stream. Across a break no `Transition` is emitted,
so a genuine `firewalled` → `exposed` straddling the upgrade is **never alerted** — the flagship
event, dropped silently, on the night the operator is watching an upgrade instead.

Dual-running both derivations was rejected in both forms. **Backward** dual-run — computing the
new derivation over past observations so an overlap exists — is re-derivation with a fig leaf:
the old rows survive, but every transition on the new timeline is still a function of today's
release. **Forward-only** dual-run avoids that and does buy the boundary comparison on the old
derivation, at the price of doubled derivation cost, a retirement policy, and deliberately
showing the operator conclusions we have just decided are wrong for a full window.

The coverage class carries it instead. One **re-baseline message** per alerting derivation whose
vector moved: *your `Exposure` derivation changed; under the new one twelve services are exposed
where ten were, and these three differ.* It is ADR-0004's *"your rules changed"* presentation
given an actual payload. It fires **on the cause, never per affected subject** per ADR-0007,
where the cause is our own release. And an empty difference set suppresses the message entirely,
which is legal precisely because ADR-0007 put all damping in notification. The break is still
written and the horizon still clamps — an empty difference set is a fact about values, not a
licence to compare them.

One constraint keeps it honest. The difference set is a **description computed once at the
cause**: never persisted as a `Transition`, never entering drift history, never appearing on the
board. Without that the forbidden comparison walks back in through the notification layer —
which is exactly how it re-entered through the failure path in
[ADR-0005](./0005-scan-execution-model.md), and is worth naming the second time.

### A declared parameter is authored by the project and ships in the release

*Added by [#60](https://github.com/winniel123/verge-asm/issues/60), which found the Consequences
sentence below contradicting [ADR-0004](./0004-signals-are-release-coupled-rules.md)'s v1 set in
plain text, both on `main`.*

*"Nothing here becomes operator-configurable"* is **narrowed in what it governs and widened in
what it reaches**, and it is the widening that matters. Read literally it is a claim about the
whole product, which is false — [#22](https://github.com/winniel123/verge-asm/issues/22)'s coverage
alert threshold is operator-configurable and always was. Read as this ADR's Consequences section
frames it, it is a claim about the four `Exposure` leaves it was written over, which is too narrow
to have caught `certificate-expiring`'s `N`. The rule it was reaching for is neither:

> A **declared parameter** is authored by the project and ships in the release. An operator's dial
> may sit anywhere **outside** every derivation, and nowhere inside one.

That governs every derivation this ADR versions rather than the four it was written over — the
availability window, `k`, [ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md)'s
timeouts, retry counts and wire libraries, and now `certificate-expiring`'s horizon — and it says
nothing at all about the notification layer, where #22's threshold lives and stays.

The line is #22's own, restated in this ADR's vocabulary: **inside the comparison path or outside
it**. A declared parameter is by construction inside, because this ADR put parameters *inside their
leaf* rather than beside it, so moving one moves a version and a moved version is a `Break`. That
is the whole mechanism, and an operator's settings field is the one actor in the system that can
move a version without a release, without a corpus row moving, and without knowing it did.

Two supports, both of which arrived after this ADR and neither of which it could have cited.

**ADR-0021's gate has no honest reading under a per-install parameter.** The gate is bidirectional
— a version moves only on a moved corpus row, a changed declared parameter, or a recorded uncovered
move — and all three are checkable *because the parameter set is declared data in the repository*.
A parameter living in an install's database is checkable by nothing, so the corpus gates a function
no operator runs and the build's guarantee is void wherever the dial was turned.

**[ADR-0023](./0023-consent-names-the-door.md) already made this move one property across.**
`consent` is authored by the project and ships in the release. An operator's act **satisfies** it
and never moves it, because reading the value as the carrier gives the model two representations of
one fact. A declared parameter is the same shape: the leaf names what decided the output, and an
install-local parameter makes the name stop naming it while the string stays equal. Two installs on
one release would compare as comparable and would not be.

What an operator may still have is unchanged and worth stating, because the refusal reads broader
than it is. Everything outside every derivation stays theirs — #22's alert threshold, notification
routing and ~~all flap suppression~~ **any flap suppression that is ever built**
([ADR-0007](./0007-drift-is-a-timeline-of-spans.md) put damping
there on purpose, and [#119](https://github.com/winniel123/verge-asm/issues/119) /
[ADR-0039](./0039-a-channel-carries-the-message-never-the-estate-and-a-delivery-is-an-operational-record.md)
records that v1 builds none — routing by class and #22's threshold are the whole of it), the
frequency half of `verge-core`
([ADR-0009](./0009-verge-core-is-a-union.md)), `Seed`s and exclusions, source enablement, and the
`custody extension`. The dial is refused in exactly one place, and it is the place where turning it
is indistinguishable from shipping a release.

## Consequences

- **[`CONTEXT.md`](../../CONTEXT.md) gains `Derivation`**, and `Break`'s first cause is restated:
  not *a derivation version changed* but *a `Derivation` vector changed*. The phrase "derivation
  version" had been load-bearing across ADR-0004, ADR-0005 and ADR-0007 without ever being a
  glossary term.
- **Every named versioned derivation acquires a second artefact** — its golden corpus — and it
  is a build-time obligation rather than a test-suite nicety. Without it, "bump on
  output-affecting change" is discipline again, and the corpus catches only what it covers.
- **Retention's hard floor is now read per derivation.** ADR-0007 ruled that the open span and
  the one preceding it may never be compacted. Across a break the predecessor sits under a
  different vector and carries no licence anyway, so the floor holds *within one derivation* —
  and a break inside the retained window makes `returned` detection unrecoverable rather than
  merely shallow.

  > **Amended by [#121](https://github.com/winniel123/verge-asm/issues/121) ·
  > [ADR-0041](./0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md), in both
  > halves.** ADR-0007's floor is withdrawn **at its own site** as a live compaction rule — the
  > `Span` corpus is never compacted at all, on arithmetic rather than on principle — and survives
  > as the precondition on any compaction a later version ships. The per-derivation reading above is
  > **confirmed and is the load-bearing half**, and *unrecoverable rather than merely shallow* now
  > has its mechanism written down: a withdrawn subject's timelines **close**, so a `Break` between
  > the withdrawal and the return leaves the reopening with nothing legally before it, no
  > `Transition` is derived, and the membership message fires reading **`appeared`**. ~~The leaf that
  > does it is `resolution-walk`~~ — **there are two**: `resolution-walk` **and**
  > `wildcard-discrimination`, which [ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md)
  > puts on a **dependency** cadence and which every `Name`'s and every cited `Address`'s membership
  > composes, because membership reads `resolution` and those two leaves decide that value jointly
  > ([#146](https://github.com/winniel123/verge-asm/issues/146) ·
  > [ADR-0086](./0086-membership-composes-every-leaf-that-decides-the-value-it-reads.md); struck here
  > at the site that states it, per
  > [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)).
  > So this is a **release** obligation and not a retention setting: the release states the
  > loss, **each membership leaf's** golden corpus must pin the membership-deciding outcomes so a
  > no-op upgrade provably does not bump the leaf — two blocks against two gates,
  > [`golden-corpus.md`](../spec/golden-corpus.md) §2 and §8 — and the membership vector may not be
  > widened **by a release**.
- **The coverage class gains a member whose cause is us.** A `Vantage` going `unavailable` and
  coverage crossing a threshold are the world or our own infrastructure failing. A re-baseline
  is our release. After ADR-0006's `resolving → shadowed` and ADR-0007's `revealed` and `owned`
  → `third-party`, the three notification classes now also have to carry *we changed how we
  look*.
- **The board and `Subjects` inherit a labelling obligation.** Every duration and count they
  render must be able to declare itself a floor, which is a presentation requirement neither was
  prototyped against.
- **Nothing here becomes operator-configurable.** ADR-0005 fixed the availability window and
  ADR-0007 fixed `k` on blast-radius grounds. A dial that moved a version would be the same
  failure with an extra step. *(Narrowed and widened by
  [#60](https://github.com/winniel123/verge-asm/issues/60) — see "A declared parameter is authored
  by the project and ships in the release" above. This sentence is a rule about **declared
  parameters of derivations**, not about the product: #22's coverage alert threshold is
  operator-configurable and is outside every derivation.)*

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| Versions coupled to releases | ADR-0004's "release-coupled" is a ceiling on how often content may change, never a claim that a version moves because a release shipped |
| Content hash of the derivation's code | Bumps on every refactor — breaks the whole estate for a rename |
| Content hash of declared parameters only | Silent on a behavioural bug fix in code, which is the case that most needs to break |
| Hand-maintained version with no gate | Discipline, refused everywhere else in the comparison path |
| One composed scalar version | Says *something moved* and never which leaf, so a `Break` cannot meet [#22](https://github.com/winniel123/verge-asm/issues/22)'s stated-reason requirement |
| A fixed global component schema | Adding a component perturbs every derivation at once — ADR-0004's set-wide version by another door |
| Nested rather than flattened vectors | Equality stops being a set comparison, and a break names a path instead of a leaf |
| A declared blast-radius predicate per bump | Fails **silently** when too narrow, emitting release artefacts as `Transition`s estate-wide; the uniform break fails loudly and now costs one cadence |
| Recomputing history to test comparability | Makes the break set a function of today's release — ADR-0007's objection to re-derivation, one level up |
| Storing the `Break` and its component diff | A second representation of one fact; both spans already carry their `derivation` reference |
| Blanking the board across a break | A break withdraws the licence to reach back, not the data — and a census is not a comparison |
| Backward dual-run across a transition window | Re-derivation with the old rows kept; every transition on the new timeline is still a function of today's release |
| Forward-only dual-run | Doubles derivation cost, needs a retirement policy, and shows the operator conclusions we have just decided are wrong |
| Carrying duration across a break | Asserts the value was unchanged, which is the comparison the break forbids |
| Re-baseline emitted as a `Transition` | Puts the forbidden comparison into drift history through the notification layer |
