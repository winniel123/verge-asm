# ADR-0073: An operator dial carries no author, however specific its target — and a per-rule count of mutes renders the partition the numbers refuse

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#129 Draw the annotated Signals row and the annotation list — and does an `Annotation` carry its author?](https://github.com/winniel123/verge-asm/issues/129)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

Two documents written on the same day give opposite answers to the same question, and neither knows
about the other.

[ADR-0016](./0016-an-annotation-moves-a-message-never-a-number.md) rider 6 specifies that an
annotation *"carries the operator's rationale in their own words beside the key, **the author** and
the time"*, and its Rationale repeats it: the object *"has a target, an author, a time and a
reason"*. [#127](https://github.com/winniel123/verge-asm/issues/127) then ruled operator attribution
**out of v1** — *"Nowhere in the model is an operator act written down with an actor on it. Named
accounts create **identity**; they do not create a **log**"* — and the ruling covers every Declared
term by name, `Annotation` included in its own enumeration.

`Annotation` is the one place where the collision bites, because it is **the one stored
operator-authored object in the Declared layer**. Every other Declared term holds a value the
project's vocabulary supplies: a `Seed`'s scope, a `Scan`'s cadence, a `Source`'s enablement, a
`Channel`'s URL. An `Annotation` holds **prose a person wrote**. A field naming who wrote it is
therefore not a gratuitous addition; it is the field the object most obviously invites. So it is
either withdrawn with a reason or kept as the named exception with a reason. It may not go on
quietly sitting in one ADR while another rules it out.

The second half of this ADR comes from drawing the screen, which had never been drawn. ADR-0016's
content is almost entirely **what may not be rendered** — three barred renderings, stated as
prohibitions on numbers — and prohibitions on numbers do not survive contact with a layout.

## Decision

### 1. An `Annotation` carries no author

**No account, no name, no initials, no avatar, no "declared by" cell — not stored and not
rendered.** `Annotation` joins every other Declared term in holding no actor, and #127's ruling is
therefore total rather than nearly total.

The ground is not #127's own, because **#127's two arguments do not reach this object** and saying
so is part of the ruling:

| #127's argument for refusing an author on `Seed` | Whether it reaches `Annotation` |
| --- | --- |
| The field is true at creation and silently stale after the first edit, so keeping it honest means attributing every edit, which is the act log | **It does not reach.** An `Annotation` cannot be edited — ADR-0016 rider 4, an edit is a **new** record, the `Proposal` construction. The field would be true forever |
| It would sit inside a Declared term, the layer the probing gate reads, and operator identity must not be joinable to the gate | **It barely reaches.** ADR-0016's entire content is what may **not** read an annotation; it sits outside every derivation, and the gate reads `Seed` and `Custody`, not this |

The ground that does hold is **what the field would make the object be**.

ADR-0016's legality argument is that an annotation is *"the same kind of object as flap suppression,
differing only in being keyed on a subject rather than on a channel"* — an operator dial, sitting
where `CONTEXT.md` says every operator dial may sit, **outside every derivation**. Every other dial
in the model is unattributed: notification routing, the coverage alert threshold, flap suppression,
a `Channel`, a `Scan`'s cadence. **An author is the single edit that stops it being a dial and makes
it a record**, and a record of operator acts is exactly what #127 declined to build. The specificity
of the target is what tempts the field and is not a reason for it: a dial pointed at one pair is
still a dial.

And the consumer test settles it. **The reader of an author cell is a second person**, and #11 ships
**one mutating role**. On the modal install the column is a constant in a per-row cell — the same
name on every row — which is #28's two-numbers hazard with the second number carrying no
information, and it is the identical objection
[ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) §7 used to keep the
evidence tier off this same screen. The column discriminates only where more than one account can
mutate, and that is **#127's own reopening condition, written before this screen existed**.

**So this is not an exception to #127 and not a second ruling. It is #127's ruling reaching its last
site, and it reopens on #127's condition and on no other.** One trigger, not two.

### 2. The declaration instant stays, and the asymmetry owes its reason

ADR-0016 rider 6 named two fields together. One is struck and one is kept, so the cut is stated:

**An `Annotation` carries the instant it was declared.** It creates no identity, it names nobody,
and it is a fact about our own store rather than about a person or about the estate. It is not a
timeline: a timeline is a sequence of `Span`s and needs two values, and this has one, with no
successor and nothing to difference against — so `Annotation` remains a Declared term that does not
drift. It is kept because rider 6's own justification requires it: *"a mute with no stated reason is
one nobody can review later"*, and an **undated** reason is not reviewable either. *We accepted this
in April* is the first thing a reviewer needs and the last thing the reason text will reliably say.

`Annotation` is consequently the only Declared term carrying an instant. That is admitted rather
than hidden, and it is the same asymmetry that already earns the object its name: it is the only
Declared term carrying operator prose.

### 3. A count of annotations may exist once, over the list, and never per rule

The list's own length is one number over one enumerable list, which is what ADR-0016 rider 3
permits. **The same figure cut per rule is barred.** `sensitive-port-reached-from-internet ·
3 fired · 1 muted` is a subtraction the operator finishes — `2 outstanding` — and the partition
rider 3 refuses has arrived without anybody rendering it.

**A rule row's mute marker is therefore a glyph and never a quantity.** It says *something beneath
this is annotated*; it does not say how much.

### 4. The instant renders as an absolute date, never as an age

`accepted 412 days ago`, particularly with a colour that deepens, is an **expiry the operator
implements by eye**. Rider 4 refused an expiry because a state that changes because time passed with
no measurement behind it is `Finding`'s lifecycle arriving through a clock; a screen that ages the
instant reinstates it in the one place rider 4 does not reach. The date is mono, absolute,
uncoloured, and the list is not sorted by staleness.

### 5. The row carries the last recorded firing, and never a count of firings

ADR-0016 makes the annotated pair's `not-fired` → `fired` `Transition` *"recorded and not a
message"*. The recorded edge is a **measurement** and may be shown: it is how the operator sees the
mute doing work. **A count of the firings a mute has swallowed may not.** It ranks acceptances by
noise avoided, which is severity through the back door on the screen ADR-0004 kept severity off. One
instant, no ordering claim.

### 6. The enumeration is its own block on `Signals`, not a section of a rule

The list sits beneath the rule list as its own block. It may not be nested inside the rule blocks,
even with no number attached and no `accepted` label anywhere: **spatial containment is a claim**,
and rendering the annotations as children of a rule's census reads as a subdivision of `fired`
whatever the arithmetic says. This is the drawing's central finding and it is not deducible from
ADR-0016's text, which forbids the partition as a *number*.

`Annotation`'s home stays `Signals`, per ADR-0016. **No new destination**; the nav remains
**Exposure · Subjects · Signals · Seeds · Coverage · Settings**.

## Rationale

### The option that lost: `Annotation` is the named exception and carries its author

This is a serious answer and it was not waved past. The case for it, at full strength:

- The one operator act whose consequence is a **silence** is the one act most worth attributing. *Who
  decided we do not need to hear about this?* is a better question than *who added this seed?*, and
  #127 refused the second while leaving the first unasked.
- Both of #127's actual arguments fail against this object, as the table above concedes. An
  annotation cannot be edited, so the field never goes stale; and nothing derived reads an
  annotation, so identity is not joinable to the probing gate through it.
- The object already holds free prose. A paragraph a person wrote, with no indication of which
  person, is a strange artefact — and the operator can defeat the ruling in one keystroke by typing
  their own name into the reason box.
- It is one nullable column against a real accountability question, which is close to free.

**It loses on what the field makes the object.** ADR-0016 survived the `Finding` objection —
#117 recorded it verbatim: *"giving operator opinion a **noun** with a target, an author and a date
is most of the way to the work-item store `Finding` was refused for; the only thing missing is the
word"* — and answered it with three structural guards: no timeline, no status field, no partitionable
census. **None of those three guards addresses the author.** #117 defeated an objection whose stated
content was *target + author + date* by removing neither the author nor the date. That is the hole
this ADR closes: `(target, author, date, prose)` is a ticket with the status field taken out, and
`(target, date, prose)` is a dial with a note on it. The word `Finding` is not what was refused; the
shape was.

And it loses on the consumer, which is the same ground #127 lost on and the same ground it is
honest about: **thin**. *The modal install has one admin* is a reading of #11's role split and not a
measurement. It is load-bearing here as it was there, which is precisely why the two rulings should
share one reopening condition rather than acquiring one each.

The keystroke objection is conceded and not repaired. An operator may type their name into the
reason box, and v1 does not stop them. **What the product does not do is assert it**: there is no
field, no cell, nothing to sort or filter on, and nothing that claims the text names its writer. A
free-text box that happens to contain a name is not attribution, and building the field to
pre-empt the keystroke would be building the thing to prevent somebody imitating it.

### The option that lost: store the author, render nothing

Refused, and refused by quotation — #127 §4 already ruled it: *"If it is worth writing it is worth
rendering; if it is not worth rendering it is not worth writing."* A stored actor with no reader is
a retention liability under no policy, and [#121](https://github.com/winniel123/verge-asm/issues/121)
ships **no expiry on either retirable corpus**, so it would accumulate operator identity forever
against a consumer that does not exist.

### The option that lost: strike the instant along with the author

Symmetry is the argument, and it is the wrong instinct. #127's ruling is about **actors**, and it
says so in every sentence of it — *named accounts create identity*, *the actor*, *who did this*. A
date names nobody. Striking it would leave `Annotation` holding a paragraph of prose with no
indication of when it was written, on an object with **no expiry by rule**, which means standing
mutes that nobody can even judge the age of. That is a worse artefact than either alternative, and
it is refused on rider 6's own reasoning rather than on taste.

### Why the drawing found three things the text could not

ADR-0016 states its prohibitions as prohibitions on **numbers**: do not partition the census, do not
show a delta, do not drop a row from a payload. Every one of the three findings above is a way of
producing the barred effect **without producing the barred number**:

- **Layout** produces the partition (§6) — annotations nested under a census read as a slice of it.
- **Arithmetic the reader finishes** produces the partition (§3) — `3 fired` beside `1 muted`.
- **Time rendering** produces the expiry (§4) — an age is a countdown drawn as a fact.

This is [#28](https://github.com/winniel123/verge-asm/issues/28)'s precedent earning itself again: a
screen with no prototype gets one, because a decision expressed as *what may not be computed* is not
the same decision as *what may not be seen*.

### Variant B ships; A and C lost on the drawing

Three variants were drawn — `prototypes/signals-annotated/`, `?variant=A|B|C`.

- **A, inline** — the enumeration inside each rule block. **Lost.** It is the layout that renders the
  partition, per §6. It is also the most natural reading of rider 3's *"rendered on the row"*, which
  is exactly why it had to be drawn: *on the row* means on the **subject's** row, not inside the
  **rule's** census.
- **C, ledger** — ordered by the recorded edge, an inbox of pages that did not arrive. **Lost on
  two.** It invites the suppressed-firings count §5 bars, and it goes **empty on a quiet estate**
  while four acceptances stand — so the operator cannot enumerate what they have muted, which is
  rider 6's whole reason for the object existing. An honest secondary view, not the primary one.
- **B, listed** — a non-numeric marker on the rule row and the enumeration as its own block.
  **Ships.**

## Consequences

- **[ADR-0016](./0016-an-annotation-moves-a-message-never-a-number.md) is amended at two sites**, and
  the amendment is a withdrawal at the site that specifies the mechanism
  ([ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)): rider 6's
  *"the author and the time"* and the Rationale's *"a target, an author, a time and a reason"*. Read
  alone and in the present tense, either sentence would cause a competent session to build the field.
  A withdrawal note is posted on [#117](https://github.com/winniel123/verge-asm/issues/117), whose
  resolution carries the same clause, on #127's own precedent of posting one on #11.
- **[`CONTEXT.md`](../../CONTEXT.md)'s `Annotation` entry gains one clause** on rider 6 — it carries
  no author, and it carries the declaration instant. The entry was already silent on the author, so
  this is an addition rather than a repair; the silence is what let the two ADRs disagree unnoticed.
- **[#127](https://github.com/winniel123/verge-asm/issues/127)'s ruling is now total.** *No operator
  act is written down anywhere with an actor on it* holds without exception, and its Out-of-scope
  entry needs no amendment: the reopening condition it already carries — **the spec admits a second
  party who can mutate** — is this ADR's reopening condition too.
- **No number in the corpus moves.** The v1 rule set stays at **seventeen**, every census is as
  specified, `Coverage` is untouched, no `Derivation` leaf gains an input, and no aperture input is
  added. The rule-row mute marker adds no figure by construction.
- **The `Signals` screen gains one block and no destination.** The nav is unchanged, the notification
  classes stay a partition of three, and the annotation list is the sole place an operator can
  enumerate what they have accepted.
- **`Annotation` is the only Declared term carrying an instant**, and the only one carrying operator
  prose. Both exceptions are now written down in one place rather than inferred.
- **Six refused renderings, not three.** ADR-0016's three are joined by the per-rule count, the
  rendered age, and the suppressed-firings count. All six are drawn, struck and labelled in the
  prototype's refusals panel, so the next session finds the refusals rather than re-deriving them.

  > **A SEVENTH rendering, and it is the first that is REQUIRED** —
  > [#163](https://github.com/winniel123/verge-asm/issues/163) ·
  > [ADR-0092](./0092-an-operator-dials-movement-is-not-a-cause-and-an-annotation-never-lapses.md),
  > 2026-08-15. A row whose `(subject, signal-name)` key names a subject in **no current population**
  > is **marked** as such. It is added here because six refusals read alone specify that nothing
  > further may be rendered on this block, and because the defect it repairs is this ADR's own §6
  > justification failing in the reassuring direction: the list exists so the operator can enumerate
  > what they have muted, and an unmarked dead row asserts an acceptance that accepts nothing. Four
  > properties keep it inside every rule this ADR set. It is **derived on read** from the current
  > subject population and stored nowhere, so rider 4's *no status field* is untouched. It is caused
  > by a **measurement** — the subject withdrew ([ADR-0006](./0006-subjects-leave-by-measurement.md))
  > — and never by time, which is §4's whole distinction. It carries **no valence, no quantity, no
  > colour scale and no sort**, per §3 and
  > [ADR-0064](./0064-a-message-names-what-moved-and-where-nothing-moved-it-says-so.md) §3. And it
  > **reverses without an act**: a returning key is the same pair, so the mark goes and the mute is
  > live again. **It is not drawn**, and this ADR's own history — three of its six findings came only
  > from the layout — is the reason that is flagged rather than assumed away.
- **§1's dial argument reaches a second question it was not written for** — #163 · ADR-0092. *A dial
  pointed at one pair is still a dial* refused the author field here; it also decides that the dial's
  **movement** fires no message, since no other dial in the model emits one and the specificity of
  this one's target is again the temptation rather than the reason. Confirmed, not amended.
- **A stated, unfixed cost.** v1 cannot answer *who accepted this risk* on a multi-admin install, and
  an operator who wants the answer will type a name into the reason box, where the product asserts
  nothing about it. That is the honest state and it is not repaired.
- **Thin ground, flagged.** Both this ruling and #127's rest on *the modal install has one mutating
  account*, which is a reading of #11 and not a measurement. Sharing one reopening condition is the
  mitigation.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **`Annotation` carries an author as the named exception**, because it is the one stored operator-authored object and the one act whose consequence is a silence | Both of #127's arguments genuinely fail against it, and it still loses: the author is the edit that turns a dial into a record, and a record of operator acts is what #127 declined to build. On the modal install the column is a constant in a per-row cell — ADR-0032 §7's objection on this same screen. It discriminates only under #127's own reopening condition |
| **Store the author, render nothing**, so v1.1 adds the cell with no rework | #127 §4 verbatim: *if it is worth writing it is worth rendering*. A stored actor with no reader accumulates identity forever under #121's unbounded retention |
| **Strike the instant as well**, for symmetry with the author | #127 rules on **actors**; a date names nobody. An undated standing mute on an object with no expiry cannot be judged at all, which defeats rider 6's own reason for requiring a stated reason |
| **Render initials, or an avatar, instead of an account name** | The same field with worse legibility. Identity is identity at any resolution, and a two-letter cell still discriminates on a three-admin install and still reads as a constant on a one-admin install |
| **Print the count of annotations beside each rule's `fired`**, since the count is legal over the list | The cut is what is barred, not the counting. Per rule it is a subtraction the reader finishes, and `outstanding` is a population no rule versions (#74, ADR-0024) |
| **Nest the annotation list inside its rule block** (variant A) | Spatial containment is a claim. Rider 3's *rendered on the row* means the subject's row; nesting under a rule's census renders the partition the numbers refuse |
| **Make the recorded edge a count** — *14 firings suppressed* | Ranks acceptances by noise avoided, which is severity on the screen ADR-0004 kept severity off. The last recorded firing is one measurement and makes no ordering claim |
| **Age the declaration instant**, so stale acceptances surface | An expiry the operator implements by eye. Rider 4 refused an expiry; drawing it as an age reinstates it where rider 4 does not reach |
| **Lead the surface with the recorded edges** (variant C) | It goes empty on a quiet estate while acceptances stand, so the operator cannot enumerate what they have muted — rider 6's reason for the object. And it invites the suppressed-firings count |
| **Add a seventh destination for annotations**, or a `Settings` surface | ADR-0016 fixed the home at `Signals` and refused a seventh destination. Nothing here reopens it, and the list is small, enumerable and read beside the rules it mutes |
