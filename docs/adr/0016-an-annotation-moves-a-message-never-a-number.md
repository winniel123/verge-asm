# ADR-0016: An `Annotation` moves a message, never a number

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#117 Is operator opinion worth a Declared-layer term, or does it collapse into signal suppression?](https://github.com/winniel123/verge-asm/issues/117)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

`Annotation` entered [`CONTEXT.md`](../../CONTEXT.md) at
[#7](https://github.com/winniel123/verge-asm/issues/7) as the **residue of a refusal**, not as a
term anybody had designed. `Finding` was rejected because it *"invites a stored object that
accumulates state and gets diffed"*, and the fog patch it displaced had to land somewhere:

> The accepted cost: the fog item **"Finding lifecycle — triage, suppression, accepted-risk"** now
> has no stored object to hang on. Suppression attaches to a `(subject, signal-name)` pair as an
> `Annotation` — Declared-layer, operator-authored.

That sentence has been the whole specification for a hundred tickets, and **every ticket that
reached for the term pushed it away.** Three of its four conceivable jobs are now barred by name:

| Job | Barred by | On what ground |
| --- | --- | --- |
| Remove a subject from the estate | [#17](https://github.com/winniel123/verge-asm/issues/17) · [ADR-0006](./0006-subjects-leave-by-measurement.md) | *"using opinion to remove a subject is the suppression [#22](https://github.com/winniel123/verge-asm/issues/22) refused wearing a different hat"* — the act is a boundary claim and belongs to `Seed` |
| Narrow a rule's population so it stops firing | [#53](https://github.com/winniel123/verge-asm/issues/53) · [ADR-0024](./0024-a-rules-domain-is-the-extension-of-its-name.md) | An accepted risk is a subject *of which the fact is still true*, so it is **inside** the `Predicate domain`. *"`Annotation` loses its last route into the comparison path"* |
| Mark a drift record seen, triaged or acknowledged | [#8](https://github.com/winniel123/verge-asm/issues/8) · [ADR-0007](./0007-drift-is-a-timeline-of-spans.md) | *"Spans are **immutable**. No `acknowledged`/`triaged` state, or #7's `Finding` rejection is undone with better provenance"* |

[#22](https://github.com/winniel123/verge-asm/issues/22) closed a fourth door before it opened —
*"No annotation or suppression of coverage gaps in v1 … this means inventing a second annotation
target"* — leaving the term with exactly one target kind and, on the face of it, nothing to do with
it.

**And yet two live documents already spend it as a shipped mechanism**, so it cannot be quietly
dropped either. [`sensitive-ports.md`](../research/sensitive-ports.md) §4.2 answers the one named
operator population the sensitive list cannot serve — a listed port behind an authenticating
gateway — with *"an `Annotation`, not a list change. That mechanism already exists"*. And
[ADR-0004](./0004-signals-are-release-coupled-rules.md)'s exclusion of operator-authored rules
reopens *"for v1.1 once `Annotation` and versioning have run in production"*, which presupposes it
ships. A third document, [ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md)
§7, hangs a conditional on it.

The glossary entry is meanwhile **wider than anything the model permits**, and wide in the exact
direction the model keeps refusing:

> Operator opinion attached to a subject — **a suppression**, an accepted risk.

Read alone and in the present tense — [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)'s
test — *a suppression* attached to *a subject* would cause a competent session to build both the
thing #22 refused and the second annotation target #22 declined to invent. It is a specifying site
carrying a mechanism three decisions have superseded, and it has never been repaired.

## Decision

**`Annotation` stays, as a Declared-layer term with exactly one job: it is an operator declaration
about one `(subject, signal-name)` pair, and its whole effect is on the *message*. It moves no
number.**

The mechanical statement, and the two halves are the decision:

| | |
| --- | --- |
| **What it does** | A `not-fired` → `fired` `Transition` on an annotated pair is **recorded and is not a message**. |
| **What it may never do** | Move a count, a timeline, a domain, a census or a subject. |

So an annotated pair still enters the estate, is still measured on its cadence, still holds every
`Span` it held, still evaluates under the same `Predicate domain` at the same version, and is still
counted under `fired` in that rule's census on `Signals`. The operator has not disagreed with the
measurement and cannot; they have said *do not wake me for this one*, which is a statement about the
channel and about nothing else.

Six riders bind, and each closes a door somebody will otherwise open.

**1. It reaches the firing edge and nothing else.** Not `fired` → `not-fired`, which is a message on
exactly the four rules whose clearing condition is a name a third party could claim — the operator
accepted a risk, not a takeover. Not `fired` → `not-evaluable` or `not-evaluable` → any value, which
are **coverage class**: *we stopped looking* is #22's refused suppression however it is spelled, and
an annotation cannot reach one in any case, since a coverage-class message is not keyed on a signal
name.

**2. It never removes a row from a census it appears in.** A census is **payload**, enumerable in
full and never sampled, ranked, grouped or truncated
([#74](https://github.com/winniel123/verge-asm/issues/74),
[ADR-0031](./0031-membership-alerts-at-the-root-of-the-entering-subtree.md)). So an annotated rule
that opens at `fired` beneath a `Reach` move still appears in that move's census, and the message
still fires — because the message is the `Reach` leg's, not the rule's. **The mute is on the pair's
own edge, never on somebody else's payload.**

**3. It may not partition a census.** `fired` is not cut into *accepted* and *unaccepted*. A census
member is *"the only cut of a rule's population that the rule itself versions"* (#74), and a cut
authored by the operator is a second, unversioned population over one rule — ADR-0024's rejected
*"two independently-authored populations"*, and [#28](https://github.com/winniel123/verge-asm/issues/28)'s
two-numbers hazard. The annotations are their own enumerable list, whose count is the length of that
list, and they are rendered **on the row** and never **as a slice of the rule**.

**4. It has no timeline, no status and no expiry.** It is Declared, and Declared does not drift.
Editing one is a **new** `Annotation` and never an existing one changed — the construction `Proposal`
already uses for the same reason. An expiry is specifically refused: a state that changes because
time passed, with no measurement behind it, is ADR-0006's cardinal sin and `Finding`'s lifecycle
arriving through a clock. Withdrawal is an operator act, and it is one click.

**5. It is keyed on the exact subject and never travels.** An annotation on a `Name` does not reach
the `Service`s under the addresses that name resolves to. A rule that propagated operator opinion
down a derived tree is #17's refused glob — *a rule whose blast radius nobody can see as they write
it*. **The cost is stated rather than hidden**: in a cloud estate a redeploy onto a new address is a
new `Service` key, so the acceptance lapses and is re-declared. That is the loud failure and the
intended one — a lapsed mute pages somebody, a travelling mute silences a subject nobody chose.

**6. It states its reason, and it is not on `Coverage`.** An annotation carries the operator's
rationale in their own words beside the key, the author and the time; a mute with no stated reason
is one nobody can review later. It does **not** appear on `Coverage`, and the contrast with `Seed`
exclusions is the whole point: #17 put exclusions there because they are *"the one route by which an
operator can quietly shrink the estate until the board looks clean"*, and an annotation shrinks
nothing. Its home is `Signals`. **No seventh destination**, and no new notification class — the
three remain a partition of messages.

## Rationale

### The one surviving job is legal by a rule the model already has

This is not a new latitude. The model already sanctions operator-configurable damping, and states
exactly where it may sit:

> An operator's dial may sit anywhere **outside** every derivation and nowhere inside one, which is
> where the coverage alert threshold, notification routing and **all flap suppression** already are.
> — [`CONTEXT.md`](../../CONTEXT.md), `Derivation`, from [ADR-0008](./0008-derivation-versions-move-on-content.md)

And ADR-0007 chose that placement deliberately, in the sentence that refused hysteresis:
*"damping at the model layer destroys it permanently, damping at notification does not."*
ADR-0029 repeats it for the flagship alert. So a per-pair mute is **the same kind of object as flap
suppression**, differing only in being keyed on a subject rather than on a channel — and being keyed
on a subject is precisely what makes it need a name. Routing and flap suppression are global dials
and need none; this one has a target, an author, a time and a reason, and an operator who cannot
enumerate what they have muted has not accepted a risk, they have lost one.

### Deleting the term does not remove the pressure, it routes it to the destructive instrument

The strongest case for refusal is that a term with one thin job and three barred ones is dead
vocabulary, and this map deletes those on sight — `internal-only` ([ADR-0017](./0017-exposure-needs-both-legs.md)),
`unknown` on `Custody` ([ADR-0013](./0013-custody-is-control-and-extends-by-declaration.md)), the
fifth `Exposure` name ([#45](https://github.com/winniel123/verge-asm/issues/45)). It loses on what
the operator does next.

`sensitive-ports.md` §4.2 names a population the model deliberately cannot serve any other way: an
operator who has genuinely put an authenticating gateway in front of a listed port.
[ADR-0009](./0009-verge-core-is-a-union.md) closed the port-list route by name — *"a port the
operator can hide is a signal the operator can silence"* — and pointed them at the scope
declaration instead. Refuse the annotation as well and the **only** remaining act is the one #17
already flagged as the quiet-shrink route: narrow the `Seed`. That stops the measurement, takes the
subject's timelines with it, and buys silence at the price of the drift detection the product exists
for.

**The two acts are different and the model has an instrument for only one of them.** ADR-0009's case
is *stop looking*, and `Seed` is the right answer to it. This case is *keep looking, do not page me*,
which is the ticket's own phrasing — **accepted risk on a thing we are still measuring** — and there
is no instrument for it. Refusing to build the cheap, non-destructive one does not leave the
operator without an option; it leaves them with the expensive, destructive one, and it makes the
board's cleanliness a function of how tired they are.

### Why it is Declared, and why it is not `Finding` returning

Declared is *"what the operator tells us — no, it is input"*. An annotation is input to the
notification layer and to nothing else, which is the same shape that earns `Proposal` its layer:
*"it earns its layer because its only consumer is the operator's declaration act, and because like
`Seed` it is input and does not drift."*

The `Finding` objection is that a named operator-facing object **accumulates state and gets diffed**.
Riders 3 and 4 are what stop it, and they are not discipline:

- It holds **no timeline**, so there is nothing to diff. Declared terms do not drift, and the
  three-layer table is the enforcement.
- It holds **no status field** — no `open`, `closed`, `expired`, `risk-accepted-until`. An edit
  produces a new record.
- It may not **partition a census**, so the product can never render *9 accepted, 3 outstanding*,
  which is the queue-shaped surface a work-item store is built to serve.

What is left is a mute with a reason attached. That is genuinely not a finding; it is the same
category of thing as a notification rule, held per subject.

### It does not fire ADR-0032's conditional

ADR-0032 §7 withheld the per-row evidence tier from the interface and **named the condition that
would move it**: *"if the operator ever acquires a sanctioned way to disagree with a row — the map's
open `Annotation` question — the disclosure becomes that act's input and earns that home."*

**The condition is not met, and the distinction is exact.** An annotation is an act on the *channel*,
downstream of the verdict entirely: the signal fires, the census counts it, the table is untouched,
and no evidence is weighed. The act ADR-0032 was waiting for is one that changes what the rule
**concludes** — and §7's own reasoning shows why the tier still must not ship to the screen if it
did not: the tier is *"identical on every firing of that row"*, so putting a per-row constant beside
a per-subject decision is #28's two-numbers hazard with the second number carrying no information,
and inviting the operator to doubt a row is the door ADR-0009 locked.

An operator who mutes a gateway-fronted `5432` is not saying the row is wrong. They are saying it is
right and they have handled it. **`sensitive-ports.md` §4.2's sentence therefore survives, with one
precision it never carried**: the annotation does not take the row off, does not stop the signal
firing and does not change the census. It stops the phone ringing.

### The glossary sentence is the thing this ticket actually had to repair

Three decisions superseded the word *suppression* in `Annotation`'s definition — ADR-0006 for the
subject-removal reading, ADR-0024 for the domain reading, ADR-0007 for the drift-record reading —
and **none of them repaired the site that specifies it**, which is the failure ADR-0058 was minted
for. #22 declined to invent a second annotation target while the glossary went on saying the term
attaches to *a subject*. Under ADR-0058's test both clauses fail: read alone, in the present tense,
they would cause a competent session to build exactly the mechanisms that were refused. The
repair is at `CONTEXT.md`, and it is a larger part of this ticket's value than the ruling.

## Consequences

- **[`CONTEXT.md`](../../CONTEXT.md)'s `Annotation` entry is rewritten**, and the word *suppression*
  is withdrawn at the site that specifies it. The entry now names the one job, the six riders and
  the three barred routes with their decisions, so the next session that reaches for the term finds
  the refusals rather than re-deriving them.
- **No number in the corpus moves.** The v1 rule set stays at **sixteen**, every census stays as
  specified, `Coverage`'s numerator and denominator are untouched, and no `Derivation` leaf gains an
  input. This ADR is the rare one whose entire content is what may *not* be read.
- **No new notification class and no new destination.** The three classes remain a partition of
  messages ([ADR-0007](./0007-drift-is-a-timeline-of-spans.md)); the nav remains
  **Exposure · Subjects · Signals · Seeds · Coverage · Settings**, and the annotation list lives on
  `Signals`.
- **[ADR-0024](./0024-a-rules-domain-is-the-extension-of-its-name.md) is annotated.** Its
  Consequences say *"the map's open question about whether accepted risk is worth a Declared-layer
  term is untouched"*; the question is now closed, and the sentence read alone would report it open.
- **[ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) §7's conditional is
  annotated as *not fired***, with its reason, so that the next session to open that file does not
  read the arrival of `Annotation` as the trigger.
- **[`sensitive-ports.md`](../research/sensitive-ports.md) §4.2 gains a precision** rather than a
  correction: the residual gateway case is an `Annotation`, and the annotation mutes the message
  while the row, the firing and the census all stand.
- **[ADR-0004](./0004-signals-are-release-coupled-rules.md)'s v1.1 reopening condition is now
  well-founded.** *"Once `Annotation` and versioning have run in production"* previously named a term
  with no specification behind it; there is now something that can run.
- **A stated, unfixed cost.** An acceptance is keyed on the exact subject, so a redeploy onto a new
  address lapses it. v1 does not solve this, and the two candidate solutions — propagating opinion
  down a derived tree, or keying on something coarser than a subject — are refused by #17 and by
  ADR-0051 respectively.
- **`Annotation` is now a term with a decision using it**, which it had not been since #7. The
  parallel is `authority`, which sat in the glossary through four tickets with *"no decision ever
  using it"* until ADR-0007 found its job. The difference is that this one's job is to be read by
  nothing derived, and that had to be written down before it could be trusted.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Delete the term.** Three of four jobs are barred, the fourth is thin, and dead vocabulary invites a workflow | Loses on what the operator does next. `sensitive-ports.md` §4.2's gateway population and ADR-0004's reopening condition both go false, and the only remaining act is narrowing the `Seed` — #17's quiet-shrink route, which stops the measurement to buy silence. Refusing the cheap instrument does not remove the pressure |
| Rule it **out of scope** for v1 and defer to v1.1 | It is already in the glossary and already spent by two live documents, so deferral is a decision to leave a specifying site saying *suppression* for another release. Deferral is also not free here in the way an aperture deferral is ([ADR-0065](./0065-a-rule-is-excluded-by-its-fact-or-by-its-aperture-never-by-the-shape-of-the-set.md)): nothing about the measurement surface changes to discharge it |
| Let it narrow a `Predicate domain`, so the rule simply does not fire | ADR-0024 ruled this by name and the ruling is not reopened here: the fact is still true of the subject, so the subject is inside the domain, and excluding it is *"the model-layer damping `Drift` refuses"*. It would also move a versioned leaf on an operator's act, which is the settings-field hazard `Derivation` bars |
| Give it an **expiry** — *accept for 90 days* | A state that changes because time passed with no measurement behind it. ADR-0006 refused exactly this shape for membership, and it is `Finding`'s lifecycle arriving through a clock rather than through a status field |
| Let it annotate a **coverage gap**, so three dead hosts stop holding the figure at 99% | #22 refused it and gave the reason: the correct fix is the host leaving the inventory, which is #17's question, and suppression here papers over it. Rider 1 keeps it unreachable structurally as well as by rule |
| Split the `fired` census into *accepted* and *outstanding* | Two independently-authored populations over one rule (ADR-0024), a denominator the rule does not version (#74), and the queue surface that makes the object a work item |
| Let an annotation on a `Name` **travel** to the `Service`s beneath it, so a redeploy does not lapse it | A rule whose blast radius nobody can see as they write it — #17's refused glob, applied to muting instead of to exclusion. The lapse is the loud failure and is preferred to a silent one |
| Make it **Operational** rather than Declared | Operational records *"what the system did"*. An annotation is what the operator did, and it is input to a consumer, which is what Declared holds |
| Keep the glossary wording and rule only on the scope | The wording **is** the defect. Three decisions superseded *suppression* and none repaired the specifying site; leaving it is ADR-0058's failure repeated with this ADR as the new supersession that also fails to reach back |
| Treat the arrival of `Annotation` as firing ADR-0032 §7's condition | Muting a message is not disagreeing with a row. Nothing the operator does changes what the rule concludes, so the weak-tier disclosure has no act to be an input to — and §7's other two objections to shipping it are untouched |
