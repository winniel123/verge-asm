# ADR-0092: An operator dial's movement is not a cause — and an `Annotation` never lapses, its subject withdraws

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#163 Does withdrawing an `Annotation` reach anybody?](https://github.com/winniel123/verge-asm/issues/163)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)
- **Amends:** [ADR-0016](./0016-an-annotation-moves-a-message-never-a-number.md),
  [ADR-0073](./0073-an-operator-dial-carries-no-author-however-specific-its-target.md)
- **Confirms without amending:** [ADR-0074](./0074-an-aperture-narrowing-that-takes-its-carrier-with-it-fires-at-the-scope.md),
  [ADR-0014](./0014-only-revealed-generalises.md)

## Context

Two questions were open, and [#129](https://github.com/winniel123/verge-asm/issues/129) recorded
that they are one: whether a **Declared act's lapse or reversal speaks**. This ADR answers both
limbs, because answering either alone would have to guess at the other.

**The reversal limb**, [#163](https://github.com/winniel123/verge-asm/issues/163).
[ADR-0016](./0016-an-annotation-moves-a-message-never-a-number.md) rider 4 makes withdrawal *"an
operator act, and it is one click"*, and the object has no timeline, no status and no expiry — so
withdrawal leaves **nothing behind**. The annotation list is consequently a record of who muted what
*and still has it muted*. Re-arming a pair is the **loud** direction and may be right to leave
silent, but nobody had asked.

**The lapse limb**, the map's *whether a lapsed acceptance is worth a message* patch, by-catch of
[#117](https://github.com/winniel123/verge-asm/issues/117). ADR-0016 rider 5 keys an annotation on
the exact subject and states the cost: *"in a cloud estate a redeploy onto a new address is a new
`Service` key, so the acceptance lapses and is re-declared. That is the loud failure and the intended
one."* The patch stayed fog because both candidate repairs are barred by name — propagating opinion
down a derived tree by [#17](https://github.com/winniel123/verge-asm/issues/17)'s refused glob, and
keying on something coarser than a subject by
[ADR-0051](./0051-a-subject-key-is-the-thing-denoted-and-its-normalisation-may-never-move.md) — and
because *nobody has asked whether the lapse itself should speak*.

**Why it is live now, and it is not a symmetry itch.**
[#130](https://github.com/winniel123/verge-asm/issues/130) ·
[ADR-0074](./0074-an-aperture-narrowing-that-takes-its-carrier-with-it-fires-at-the-scope.md) ruled
that a `Seed` narrowing over ground nothing else cites fires one coverage-class message at the scope,
on a **carrier-failure** argument: the act removes the subject that would have held the `Gap`, so
*"there is no bearer anywhere on which the removal is a fact"*, and a message is the only object that
can date it. Withdrawing an `Annotation` has the same surface shape — a Declared act whose only
record the act itself destroys. Whether ADR-0074's rule **generalises to a Declared reversal** is the
whole of #163.

And [#120](https://github.com/winniel123/verge-asm/issues/120) ·
[ADR-0064](./0064-a-message-names-what-moved-and-where-nothing-moved-it-says-so.md) §1 supplied the
instrument both limbs lacked: **a message names what moved, and what moved is read from the fold.**
That turns *should this speak?* into a question with a procedure behind it.

What is settled and is **not** reopened here:

- ADR-0016's six riders and its three barred routes
- ADR-0073's no-author ruling and its six refused renderings
- the **four causes** and **three classes** that partition messages, the coverage class at **ten**
  members and the clock class at **three** (ADR-0064, ADR-0074)
- ADR-0014's *only `revealed` generalises*

## Decision

**A `Message` is one firing of one cause. An operator dial's movement is not one of the four causes,
so neither declaring nor withdrawing an `Annotation` fires anything. Where the act's only consequence
is that messages resume, the messages themselves are the carrier. And an `Annotation` never lapses —
the record is unchanged and it is the *subject it names* that withdraws, so the mover is the estate
and the estate's own messages already fire.**

| Concern | Decision |
| --- | --- |
| **Declaring** an `Annotation` | **Silent.** No message, no record beyond the row itself |
| **Withdrawing** an `Annotation` | **Silent.** The carrier is the pair's own next `not-fired` → `fired` firing, which the withdrawal restores |
| An acceptance **"lapsing"** on a redeploy | **Nothing happens to the `Annotation`.** It did not lapse, expire or change state. The `Service` it names **withdrew**; the mover is the **estate** |
| What speaks on a lapse, then | ADR-0031's membership message on the entering root with its census, and the new pair's own `not-fired` → `fired` — both already specified, both unmuted because the new key carries no annotation |
| A cause, a class or a member | **None minted.** Four causes, three classes, coverage **ten**, clock **three**, all unmoved |
| What the annotation list owes | A row whose key is in **no current population** is **marked** as naming a withdrawn subject. **Derived on read** from the current subject population — stored nowhere, no status field, no expiry, no ordering claim |
| If the key **returns** | The row is live again. A key is the thing denoted (ADR-0051), so a `returned` subject is the same `(subject, signal-name)` pair and the mute takes effect again with no operator act |
| Auto-withdrawing a row whose subject left | **Refused.** A deletion nobody performed, on the one object whose whole job is to be enumerable |
| Why ADR-0074 does not reach this | An `Annotation` is **not an aperture input** — it is read by nothing in the measurement path and appears in no `Batch`'s recorded scope. ADR-0074 governs aperture acts; this is a dial |
| The general rule | **A message needs a cause, and a dial's movement is not one.** It reaches every dial in the model, not only this one |

## Rationale

### The four causes are exhaustive, and a dial's movement is none of them

`CONTEXT.md` defines a `Message` as *"one firing of one cause"*, and ADR-0064 §1 fixes the four:
**the world moved · we stopped looking · we changed how we look · a clock crossed.** Walk the
withdrawal against all four and it matches none.

- **The world moved.** Nothing was measured. The pair holds the same `Span`s at the same values on
  the same cadence before and after the click. The fold sees no adjacency it did not see yesterday.
- **We stopped looking.** We did not. ADR-0016's decision is that an annotated pair *"is still
  measured on its cadence, still holds every `Span` it held, still sits inside the rule's `Predicate
  domain` at the same version, and is still counted under `fired`"*. The measurement was never
  narrowed, so there is nothing to resume.
- **We changed how we look.** Aperture is *"what a `Batch` records as its completed scope"*
  (ADR-0014). An annotation is in no batch's scope, moves no `Derivation` leaf, and is read by
  nothing derived — ADR-0016's Consequences call it *"the rare one whose entire content is what may
  **not** be read"*. The batch scope diff that detects every aperture move is **identical** across
  the act.
- **A clock crossed.** No threshold, and rider 4 refused an expiry precisely so that no clock ever
  reaches this object.

**This is not a gap in the enumeration. It is the enumeration working.** The corpus has already
refused a fifth cause once, for an object of exactly this kind, and the sentence is verbatim in the
glossary: a `Delivery` *"is **never itself a `Message`** — a delivery failure is not the world moving,
our looking changing, or a clock crossing, so it has **no cause and gets no fifth one**"*
([ADR-0039](./0039-a-channel-carries-the-message-never-the-estate-and-a-delivery-is-an-operational-record.md)).
A dial moving is the same test applied to the same enumeration and it fails the same way.

ADR-0064's rule is stated as falsifiable — *point at a message whose subject is not the thing the
fold says moved.* Run it forwards and it also decides what may not exist: a message whose sentence
has **no** subject the fold can supply. The fourth grammar row is the only subjectless form, and it
is reserved for *a clock crossed*, where the sentence's whole content is **that no measurement
moved**. A withdrawal message would have to borrow that form while being caused by neither a clock
nor a measurement, and the sentence it produces — *nothing moved and nothing is different* — is a
message with no news in it.

### The carrier does not fail here — the act creates one

This is the argument that decides #163 against ADR-0074's parity, and it is worth stating precisely
because the surface shapes really are alike.

ADR-0074 minted a member on a **carrier failure**: the act *"removes the subject that would have
held the `Gap`"*, so the fact has **nowhere to live**, and *"a message is therefore the only object
that can date the act at all"*. ADR-0026's sole-carrier rule — *a `Transition` is a message exactly
where it is the sole carrier of a fact the operator asked for* — had **no application**, because
there was no carrier at all.

**Withdrawing an `Annotation` is the opposite operation on that same test. It does not destroy a
carrier. It restores one.** The mute's whole content is that a `not-fired` → `fired` edge on one pair
is *"recorded and is not a message"*. Withdraw it and that edge becomes a message again. So the
consequence of the act — *this pair can page you now* — is carried by **the message the act
re-enables**, at the moment the fact becomes true of the estate, in the sentence the operator
actually asked for.

The obvious objection is that the pair may never fire again, so the operator may never learn.
**Correct, and it costs nothing**: a re-armed mute on a pair that does not fire has no consequence
to learn about. The fact is interesting exactly when it has an effect, and it has an effect exactly
when it produces a message. That is a carrier that is complete rather than merely present, which is
more than the `Gap` gives an ordinary narrowing.

The same test run on the **arming** direction gives the other half, and it also passes. A declared
mute's carrier is the **annotation list**, which ADR-0073 §6 made its own block on `Signals` and
whose completeness is structural: every live mute is a row, because withdrawal removes the row and
there is no status that could leave a stale one. ADR-0074 rejected `Coverage` as the narrowing's
carrier on two grounds — it is *"a standing state and never an event"*, and it is *"written for
exclusions, while a smaller CIDR replaces the declaration outright and leaves no exclusion row to
render"*. **The second ground is what actually did the work there, and it does not transfer**: the
annotation list has no missing rows by construction. The first ground reduces, correctly, to *this
standing state is incomplete* rather than to *standing states cannot carry*.

### No dial in this model emits a message when it moves, and this one is a dial

ADR-0016's legality argument for the term is that an annotation is *"the same kind of object as flap
suppression, differing only in being keyed on a subject rather than on a channel"* — an operator dial
sitting where `CONTEXT.md` says every dial may sit, **outside every derivation**. ADR-0073 §1 then
made that analogy load-bearing rather than illustrative: *"a dial pointed at one pair is still a
dial"*, and it refused the author field on exactly that ground.

Take the dial family as it stands and ask what each emits when the operator moves it.

| Dial | Message on the act? |
| --- | --- |
| Notification routing — which classes reach which `Channel` | None specified anywhere |
| The coverage alert threshold ([#22](https://github.com/winniel123/verge-asm/issues/22)) | None |
| A `Channel`'s URL, secret or class subset (ADR-0039) | None |
| A `Scan`'s cadence | None |
| The `Dispatch` retention dial (ADR-0081) | None |
| An `Annotation` | **This ticket** |

Routing is the sharpest of the five, because it is the **same silencing power at a coarser grain**.
An operator who routes the drift class off their only channel silences seventeen rules over the
entire estate, and nothing in the corpus proposes that the act emit a message. An annotation
silences **one edge on one pair**. Minting a message for the smaller act while the larger one stays
silent would be the model rendering a distinction it does not hold.

So the ruling either extends to the whole family or it is an exception owed a reason — and ADR-0073
has just declined to make this object the family's exception once, over the author, on the reasoning
that *"the specificity of the target is what tempts the field and is not a reason for it."* The
identical sentence answers the identical temptation here.

### ADR-0074 does not reach this, and the discriminating line is aperture

ADR-0074's own Decision names its trigger twice: it fires *"where the act takes the carrier with
it"*, and its class is **coverage** — *we changed how we look*. Both halves presuppose that the act
is an **aperture** act. Its Consequences make the reach explicit when naming the one neighbour it
declined to rule: *"Disabling a `Source` is **also an aperture narrowing**, and a subject that source
alone admitted loses its `Citation` and leaves — the same carrier failure. … **The rule stated here
reaches it**."*

An `Annotation` is not on that side of the line. Enabled sources are an enumerated aperture input
(ADR-0014). An annotation is in no batch's recorded scope, and by ADR-0016 it is read by nothing that
produces a value. So the rule ADR-0074 states does not reach it — not because a distinction was drawn
to keep it out, but because its trigger's subject matter is absent.

**This is ADR-0014 doing its job rather than being worked around.** ADR-0014 exists to refuse
symmetry: *only `revealed` generalises*, and the opening family got one general member while the
closing family got none, *"on the ground that the model names causes rather than mirror images."*
ADR-0074 was careful to say it was **not** arguing symmetry — *"what is minted here is minted because
the fact has nowhere else to live"* — and minting a message here **because the `Seed` direction has
one** would be the defect both ADRs name.

The hazards also part cleanly, and they part in the direction that matters. ADR-0074's whole safety
argument is that a `Seed` narrowing is *"the product's quiet-shrink route"* — ADR-0009's *a port the
operator can hide is a signal the operator can silence* surviving at the one door nobody checked, and
the repair was **to make the destructive route loud**. Withdrawing an annotation is the **anti-silencing
act**: it is the operator turning the product's volume back up. There is no blindness hazard on this
edge to report, and a message reporting the absence of a hazard is the receipt ADR-0074 itself
refused for a different reason.

### An `Annotation` never lapses, and the word names an event that does not exist

This is the lapse limb, and running ADR-0064 §1's fold test on it produces a sharper answer than the
patch expected.

*What does the fold say moved when an acceptance lapses?* **The estate.** On a cloud redeploy the
name repoints, the old `Address` stops being cited and withdraws, and its `Service`s go with it —
`Service` membership being *"its `Address`'s membership restated"*. A new `Address` enters, a new
`Service` key exists, and the pair the operator muted is not the pair the rule now fires on.

**Nothing happens to the `Annotation` at all.** It is Declared and does not drift. It holds no
timeline to move on, no status to change and — rider 4 — no expiry to reach. It still names the key
it always named, with the same reason and the same instant. What changed is that the key it names is
in no current population, and `CONTEXT.md`'s `Subject` entry already says exactly what that means: a
withdrawn subject is *"a member of no current population at all … absent from every current-state
listing by construction and is reached **only by its own key**."* An annotation row holds a key. It is
the one legal reading of a withdrawn subject on a screen, and it is not an accident that it is the
one that survives.

**So *lapses* is the wrong word, and the wrongness is the kind ADR-0058 exists for.** Read alone and
in the present tense, *"a redeploy onto a new address lapses the acceptance"* would cause a competent
session to build a **lapse** — an event on the annotation, or a `lapsed` state on the row, which is
rider 4's refused status field arriving through a synonym and `Finding`'s lifecycle behind it. The
sentences are narrowed at their sites rather than struck, because their **substance** is right and
was always right: the mute stops taking effect, the operator is paged, and that is the loud failure
ADR-0016 chose deliberately over a travelling mute.

Two things fall out that were not visible before the word was fixed.

**The mute is dormant rather than destroyed, and it needs no re-declaration.** ADR-0016 rider 5 says
the acceptance *"is re-declared"*. That is true of the redeploy case, where the new `Service` is a
genuinely different key and the operator must decide about it afresh. It is **not** true of a
withdrawal-and-return: an `Address` re-cited by a later resolution is the same key, the same
`Service`, the same pair, and the row that was always there takes effect again with nobody clicking
anything. That follows from ADR-0051 — a key is the thing denoted — and from ADR-0082, which puts the
withdrawn period on no timeline and leaves the spans either side **adjacent**.

**And the exposure is narrower than the corpus implies.** The lapse is a **name-scope-plus-custody-extension**
phenomenon. On an address-scope install the `Address` is a subject *from the declaration* and its
`Service`s exist *open or closed*, so a redeploy inside the declared scope withdraws nothing and the
annotation goes on applying. The cost ADR-0016 stated is real and is not universal.

### What the list owes instead, and why it is not a status, an expiry or a valence

The one genuine defect this ticket found is not a missing message. It is that after the subject
withdraws, the annotation list — whose entire justification is rider 6's *"an operator who cannot
enumerate what they have muted has not accepted a risk, they have lost one"* — carries a row asserting
a mute that mutes nothing. The operator reads it and believes they have accepted a risk they have
not. That is [#14](https://github.com/winniel123/verge-asm/issues/14)'s false reassurance arriving
inside the operator's own dial.

**The row is therefore marked as naming a subject in no current population**, and four properties
keep the marker on the right side of every rule that governs this screen.

- **It is derived on read, not stored.** It is a join against the current subject population,
  computed when the block renders, exactly as a census is *"current state and never a comparison"*.
  Nothing is written, so rider 4's *no status field* is untouched and no `Annotation` acquires a
  second value to drift on.
- **It is a measurement, not a clock.** The subject left *by measurement*
  ([ADR-0006](./0006-subjects-leave-by-measurement.md)) — not because time passed. That is the exact
  distinction rider 4's expiry refusal rests on, and ADR-0073 §4 policed once already when it barred
  rendering the instant as an **age**. A marker that appears because a Name Error was measured is
  the opposite of a countdown drawn as a fact.
- **It carries no valence and no quantity.** It states what is the case. There is no *expired*, no
  *stale*, no *orphaned*, no colour that deepens, no sort order, no count of how many rows carry it
  — ADR-0073 §3's glyph-not-quantity rule and ADR-0064 §3's valence refusal both reach it.
- **It reverses without an act.** If the key returns, the mark goes and the mute is live again,
  because the mark was never a state — it was a reading.

**It is a seventh rendering on a screen with six refusals, and it is the first that is permitted.**
ADR-0073's Consequences enumerate six refused renderings and say the refusals are drawn so the next
session finds them. This one is added beside them as **required**, with its reason, so that the
enumeration does not read as *nothing further may be rendered here*.

### What this does not buy, stated rather than argued away

**v1 cannot answer *when was this un-muted*.** The withdrawal is undated and unrecorded, and nothing
this ADR adds recovers it. That is a real loss and it is worth naming, because the best form of the
losing case is built on it: ADR-0073 §2 kept the declaration instant on the ground that *"an undated
standing mute on an object with no expiry cannot be reviewed at all"*, and the interval's other end
is now undated by the same reasoning that dated its start.

It is not repaired here because **the thing that is missing is an operator-act record**, which
[#127](https://github.com/winniel123/verge-asm/issues/127) ruled out of v1 with a reopening condition
already attached: *the spec admits a second party who can mutate*. ADR-0073 pinned this object to
that condition rather than minting a second trigger, and this residue lands inside it unchanged.
**One trigger, not three.**

The distinction ADR-0074 drew is what keeps this honest, and it is worth keeping visible: **the actor
question and the audience question are different.** This ruling does **not** rest on *the operator
knows because they did it* — that is #127's thin single-admin ground, and a ruling built on it would
inherit #127's condition as a **trigger** rather than as the home of its residue. The ruling rests on
there being no cause, on the released message being a carrier, and on parity within the dial family.
All three hold however many operators there are.

## Consequences

- **[`CONTEXT.md`](../../CONTEXT.md)'s `Annotation` entry gains three clauses and no term is added.**
  The three clauses are:
  - that declaring and withdrawing are both silent and why
  - that the record does not lapse and matches again if its key returns
  - that the list marks a row naming a withdrawn subject

  The *lapses* wording in that entry is narrowed at the site that specifies it.
- **[ADR-0016](./0016-an-annotation-moves-a-message-never-a-number.md) is amended at two sites**, per
  [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) and with a
  replacement supplied rather than a strike alone
  ([ADR-0057](./0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md)): rider 5's *"the
  acceptance lapses and is re-declared"* and the Consequences' *"a redeploy onto a new address lapses
  it"*. Read alone, either would build a lapse event or a `lapsed` state. **Rider 4's *withdrawal is
  one click* is confirmed, not amended** — it is this ADR's premise.
- **[ADR-0073](./0073-an-operator-dial-carries-no-author-however-specific-its-target.md) gains one
  amendment note.** Its six refused renderings are joined by one **required** one, and its §1 dial
  argument is confirmed as reaching a second question it was not written for.
- **No message, so no cause, no class and no member.** Coverage stays at **ten**, clock at **three**,
  the causes at **four** and the classes at **three**. Nothing on the map's composed-state line
  moves. **This is the answer [#158](https://github.com/winniel123/verge-asm/issues/158) needs from
  this ticket**: there is no message here to route, by class or by cause, and none to word.
- **[ADR-0074](./0074-an-aperture-narrowing-that-takes-its-carrier-with-it-fires-at-the-scope.md) is
  confirmed and bounded, not amended.** Its rule is an **aperture** rule. The boundary is now written
  down, so the next session that meets a Declared act asks *does this move the aperture?* before
  asking *does it take its carrier with it?* Its named-and-unruled neighbour — disabling a `Source` —
  is on the **aperture** side of that boundary and is untouched by this ADR in either direction.
- **[ADR-0014](./0014-only-revealed-generalises.md) is confirmed.** No transition name is minted in
  any family, no fourth cause, and the refusal of symmetry is applied rather than merely cited.
- **Nothing new is stored, and no retention question opens.** The marker is a read. The annotation
  row is unchanged. No corpus gains a row or a dial.
- **Neither barred repair is reopened.** #17's glob and ADR-0051's coarser key are exactly as barred
  as they were, which is what the fog patch named as the reason it stayed fog. The patch is
  discharged **without** them, because the question it could not answer — *does the lapse itself
  speak?* — turns out to be answerable by the fold test alone.
- **[ADR-0004](./0004-signals-are-release-coupled-rules.md)'s v1.1 reopening condition is unaffected**
  and slightly better founded: *once `Annotation` and versioning have run in production* now names an
  object whose full act surface — declare, withdraw, and what happens when its subject leaves — is
  specified.
- **A stated, unfixed cost.** v1 answers *what is muted now* and never *what was muted and is not*.
  The residue is #127's and reopens on #127's condition.
- **`prototypes/signals-annotated/` is owed nothing and is deliberately not swept**, per
  [ADR-0075](./0075-a-prototype-is-a-dated-record-of-a-reading-never-of-a-rule.md). Its drawn
  statement — *an annotation is withdrawn in one click and leaves nothing behind, so it is a record
  of who muted what and still has it muted* — is a **reading this ADR confirms**, and no drawn state
  is made unreachable. The marker below is a case the drawing never held rather than one it now holds
  wrongly. Recorded because the absence of a sweep should read as a decision rather than an omission.
- **Thin ground, flagged, and it is the marker rather than the ruling.** ADR-0073's central finding
  was that *"prohibitions on numbers do not survive contact with a layout"* and that three of its six
  refusals were visible only once the screen was drawn. The marker specified above is **not drawn**,
  and it is added to the one screen in the corpus whose history says drawings find what text cannot.
  A successor draws it into `prototypes/signals-annotated/`.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Withdrawal fires one message, by parity with ADR-0074** — the strongest losing option. Its best form: the annotation list is the mute's only record, withdrawal removes the row, so the act is undated and unrecorded, which is literally the carrier failure #130 minted a member for; ADR-0073 §2 dated the interval's start on exactly this reasoning; and the consequence is real, since a pair that was silent starts paging | **The losing option.** It fails at the first gate: a `Message` is *one firing of one cause*, and a dial's movement is none of the four — `Delivery`'s *"no cause and gets no fifth one"* is the same test on the same enumeration. The carrier does not fail, it is **created**: the fact *this pair can page you now* is carried by the message the act re-enables, at the instant it matters. ADR-0074's trigger is an **aperture** act and an annotation is not an aperture input. Its hazard was **blindness** and this act's direction is **loudness**, so there is nothing to report. And it would owe a message to routing, the coverage threshold, flap suppression, a `Channel` and a `Scan`'s cadence, or make this object the dial family's exception — which ADR-0073 declined to do for the author on reasoning that applies here verbatim. What survives of it is a real residue, and it is #127's |
| **The lapse fires its own message** — *your acceptance of `10.0.0.5:5432/tcp` no longer applies* | The fold says the **estate** moved, not the dial: the `Service` withdrew and another entered. Those movements already have messages — the membership message with its census, and the new pair's own `not-fired` → `fired`, unmuted because the new key carries no annotation. A third message for one fold is ADR-0026 §6's *one cause, one message* broken, and its sentence would have to take the estate's subject while being about our dial, which ADR-0064 §1 forbids by name |
| **Auto-withdraw the row when its subject withdraws**, so the list never lies | A deletion nobody performed, on the one object whose justification is that the operator can enumerate what they have muted — it destroys the record precisely when the record has something to say. It is also wrong on return: the same key coming back would find the mute gone with no act behind its absence, and ADR-0082 makes the spans either side adjacent specifically so that a return is not a fresh start |
| **Give the row a `lapsed` status**, set when its subject withdraws | Rider 4's refused status field with a new name, and the whole point of the `Annotation`/`Proposal` construction — an edit is a **new** record — is that nothing on it ever changes. The honest object is a **reading**, computed on render and stored nowhere |
| **Leave the list unmarked** — the incumbent | The operator's enumeration of what they have muted becomes false in the reassuring direction: rows asserting acceptances that accept nothing. That is rider 6's own reason for the object defeated from the inside, and #14's false reassurance reaching the one surface #14's guard was never pointed at |
| **Mark it by ageing or colouring the row**, so stale acceptances surface | ADR-0073 §4's refusal exactly: an expiry the operator implements by eye. The marker is caused by a **measurement** and says so; a treatment that deepens with time reinstates the clock rider 4 refused |
| **Count the marked rows, or sort by them** | ADR-0073 §3's rule that a marker is a glyph and never a quantity, and #74's — a count invites a subtraction the reader finishes, and *live mutes* is a population no rule versions |
| **Fire on withdrawal but not on declaration**, since only withdrawal destroys its own record | Concedes the whole argument and then applies it to one side. If a dial's movement is a cause, arming is as much a movement as disarming — and arming is the **silencing** direction, which is the one a hazard argument would reach first. The asymmetry is real but it is about **carriers**, not about causes, and both carriers hold |
| **Record the withdrawal without rendering it**, so v1.1 has the history | #127 §4 verbatim: *if it is worth writing it is worth rendering; if it is not worth rendering it is not worth writing.* A stored act record with no reader is the operator-act record #127 declined to build, arriving through a corpus nobody named |
| **Route the question to `Coverage`**, as `Seed` exclusions are | ADR-0016 rider 6 ruled it: an annotation does **not** appear on `Coverage`, because exclusions are there for shrinking the estate and this shrinks nothing. Nothing here reopens it, and the annotation list is the complete carrier `Coverage` would only partially be |
| **Let the annotation travel to the redeployed `Service`, so nothing lapses** | ADR-0016's own rejected alternative, on #17's refused glob, and not reopened. This ADR fixes the **word** and leaves the **cost**, which was chosen deliberately: a lapsed mute pages somebody, a travelling mute silences a subject nobody chose |
