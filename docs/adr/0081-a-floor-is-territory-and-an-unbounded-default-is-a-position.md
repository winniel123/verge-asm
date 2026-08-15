# ADR-0081: A floor is territory and an unbounded default is a position the projection prices — and a corpus with no reader of its own travels with the corpus that reads it

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#139 Coverage owes the retention projection and the two dials with their floors](https://github.com/winniel123/verge-asm/issues/139)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0041](./0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md) ruled retention
and left `Coverage` a rendering obligation it recorded as undischarged: *"the projection from declared
scope size, and the two retention dials with their floors. It is not prototyped; #28's screen predates
this."* This ADR discharges it, and drawing it moved three things that a description could not have
caught.

Three constraints arrive fixed and are not re-argued here.

- **v1 ships no expiry on either retirable corpus.** Both dials default to unbounded, on the ground
  that the ceiling install is ~13 GB a year in one `docker compose` volume and no default number is
  derivable from anything.
- **The floor may never be presented as an operator choice**, and **retention may never be the tighter
  clamp** — a `Break` clamp is visible and names the leaf that moved, and a retention horizon names
  nothing.
- **A group that renders when populated must render when empty, and say why**
  ([#47](https://github.com/winniel123/verge-asm/issues/47) · ADR-0008).

And one thing arrived *after* ADR-0041 and is the reason this is an ADR rather than a ticket note.
[#119](https://github.com/winniel123/verge-asm/issues/119) added a **fourth Operational corpus** — the
`Message` store and the `Delivery` records against it — and handed the retention question forward:
[`notification-channels.md`](../spec/notification-channels.md) says *"nothing is lost by compacting it
aggressively"* and points at [#121](https://github.com/winniel123/verge-asm/issues/121), which had
already ruled and had written *"the operational record — `Dispatch` **alone**"*. The two documents were
not in contact. The screen is where they meet, because the screen has to draw the corpus one way or
the other.

## Decision

| Concern | Decision |
| --- | --- |
| A dial's floor, as a control | **Territory, never a refused value.** The range below the floor is drawn as ground the operator does not own. It is not a value they may pick and be corrected for |
| A floor's expression | **A multiple and a named `Scan`, computed live** — `k` × cadence(`zone`) — never a day count, per [ADR-0038](./0038-a-constant-is-a-product-only-where-the-quantity-is-readable.md) |
| An unbounded default | **A stated terminal position the handle is parked on**, priced by the projection beside it and grounded by one sentence saying why no number ships. Never an empty field, never `∞`, never *unlimited* |
| Where the dials live | **On `Coverage`, editable there.** A dial whose only legitimate justification is a projection may not be separated from the projection |
| How the clamp invariant is made true | **One ordered list of every clamp in force, tightest first, the binding one alone drawn in ink.** Retention appears in that list in its place, and where it would sort first it is drawn as not biting |
| The fourth Operational corpus — `Message` | **No dial.** Retained while the operator may still read it back, which nothing bounds. The store is written unconditionally and *cannot fail*; a retention dial is a way for it to fail |
| The fourth Operational corpus — `Delivery` | **No dial. It travels with its `Message`**, exactly as `Batch` travels with its observations |
| The number of dials | **Two, unchanged.** Seven corpus rows, two dials |
| The projection on an install with no address scope | **Renders, and states that it has no declared denominator.** It may show what we hold, never a forecast of what the operator has |
| The discarded group | **Renders in every state, including the one where it is always empty in v1**, and says why |
| Which facets the observation dial reaches | **A rendered enumeration**, because two facets currently have no covering `Scan` and therefore no floor — [#142](https://github.com/winniel123/verge-asm/issues/142) |

## Rationale

### A floor drawn as a validation error tells the operator the value was theirs

This is the finding the ticket's *the floor may never be presented as an operator choice* turns into a
drawing, and it only became visible once the control was rendered.

The obvious build is a number field with a minimum: the operator types `30`, presses save, and gets
*minimum is 60 days*. That interface has already conceded the point. An error is a response to a
**choice**, so a rejected value is a value the operator was offered and got wrong — and the next thing
they do is look for the setting that lifts the minimum, because a minimum that can be stated can
usually be raised. The floor is not a limit on the operator's choice; it is the boundary of the region
where a choice exists at all. Below it the dial is not *too small*, it is **not the operator's** — the
currency bound belongs to `k` and a cadence, and the `Dispatch` floor belongs to `Coverage`'s own
question.

So the control renders two territories and one of them is ground: hatched, dashed at its edge,
outside the track the handle moves in. Reaching into it produces no error, because nothing was
rejected; it produces the derivation — `k` × cadence, and the name of the `Scan` supplying the cadence
— which is the honest answer to *why can I not go there*, and which names the thing the operator
**would** change to move the floor. **A floor that names the `Scan` that sets it is a floor the
operator can move by changing the world rather than by arguing with a form.**

### An unbounded default is only a defect if it is drawn as an absence

The ticket's hard half: make an unbounded default legible without making it look like a defect. Three
renderings were drawn and two are defects on sight.

An **empty field** reads as unconfigured, and an unconfigured field is a chore — the operator's model
is *somebody has not finished setting this up*. **`∞` or *unlimited*** in a numeric field reads as a
sentinel, which is worse: a sentinel is what a program writes when it has no answer, and the operator
correctly infers that nobody decided.

What works is the third, and it is three moves rather than one:

1. **A position, not an absence.** The track has a labelled terminal stop — *keep everything* — and the
   handle sits **on** it. There is no state in which the control has no value. A parked handle is a
   decision; an empty field is a gap.
2. **A price, not a promise.** Directly beneath it, the projection: *one year of evidential
   observations is 97,925,120 rows, about 13 GB in one `compose` volume.* Arithmetic over what the
   operator typed, exact, re-rendered when a scope widens. **This is the whole reason the dials and
   the projection may not be separated.** A default that costs nothing to state is a default nobody
   audits; a default whose cost is rendered next to it is one the operator can accept on purpose.
3. **A ground, not a shrug.** One sentence: *no default is derivable from anything — ninety days is
   invented and a year is invented, so v1 ships neither.* The failure mode being defended against is
   not *the operator thinks the value is wrong*; it is *the operator thinks nobody chose*.

The generalisation is the ADR: **an unbounded default is legible exactly where it is drawn as a
position, priced by a projection and grounded by a stated reason.** Any one of the three alone reads
as a defect. This binds every future default-off and default-unbounded setting in the product, and
there will be more of them, because ADR-0038 keeps refusing invented constants.

### The clamp invariant is a sort order, not a sentence

*Retention may never be the tighter clamp* is a rule about two horizons, and a screen that states it in
prose has to be believed. Drawing the horizon as **one ordered list of every clamp in force** makes it
structural instead.

Each clamp — the first `Batch`, a `Break`, each retention dial — renders as a row with the instant it
binds at, sorted tightest first, and only the first row is drawn in ink. The invariant then has a
visible shape: **if retention were ever the tighter clamp it would sort into first place**, which is
the one position the design does not allow it to occupy. Where a dial would cut earlier than the clamp
above it, the row renders *inert* and struck through, and the clamp above stays in ink. In v1 the
retention rows are always inert, which is the point — the operator sees the two dials sitting in the
same list as the `Break` that actually binds, and sees them losing.

This also makes the `Break` do the work ADR-0008 already gave it. The binding row names the leaf that
moved. A retention row can never name anything, and that asymmetry is legible in the list rather than
argued in a footnote.

### `Delivery` travels with its `Message`, and it is ADR-0041's own finding one layer across

ADR-0041's sharpest result was that the ticket's three-corpus cut was **one member short**: `Batch`
looks like *what we ran* and belongs with the observations, because the comparison path reads it and
deleting it on the operational schedule strands an observation's scope. The same shape is here, with a
different reader.

`notification-channels.md` says *"nothing is lost by compacting it aggressively"*, and the reasoning
is that nothing in the comparison path reads a `Delivery`. That is true and it is the wrong test —
**ADR-0041's rule is *what may still read it*, and the comparison path is not the only reader.**
`CONTEXT.md`'s `Message` entry says a message *"holds read-state and **its delivery outcomes**"*, and
ADR-0039 makes a dead-lettered message render *"unread and marked undelivered"*. So the delivery
outcome is read — by the rendering of the message it was against.

Discard it and the message stops saying *we tried to escalate this and could not*. What it says
instead is *we told you*, which is **`a dead-lettered Delivery licenses no silence` defeated by
storage** — the same failure ADR-0041 refused when it kept `Batch` out of the operational schedule,
and the same failure ADR-0005 refused when it ruled that a dead-lettered `Batch` licenses no absence.

So a `Delivery` is retained while the `Message` it is against is retained. It gets no dial, no default
and no clock. The finer alternative — compact *succeeded* deliveries, keep failed ones — is rejected
below.

### `Message` gets no dial, on a structural ground and an arithmetic one

The symmetry argument for a third dial is real and deserves stating: an **evidential observation** is
retained only for *a person asking what we actually measured*, and it gets a dial. A `Message` is
retained only for a person asking *what was I told*. Same kind of reader; why not the same
instrument?

Two answers, and the first is the one that decides it.

**The store is the thing that cannot fail.** ADR-0039's whole architecture of refusals — no email, no
vendor body shapes, no pull feed, no template — is affordable *because* every message is written and
rendered unconditionally, so a channel that is misconfigured, disabled, dead or never created *"loses
no message and hides no fact"*. A retention dial on the message store is precisely a supported way for
the store to lose a message. It would take the property every other notification refusal was
purchased with and sell it for disk that is not under pressure.

**And the disk is not under pressure.** The observation corpus earns its dial by growing without bound
— ~98M rows a year at the ceiling. Messages fire **at the cause**, once, with a census, and the model
has spent a great deal of effort making sure a thousand openings are one message. A dial over a corpus
that does not grow is a horizon bought for nothing, which is exactly the trade ADR-0041 refused when
it declined to compact `Span`s to save 135 MB once.

### The observation dial's floor is an enumeration, because two facets have no floor at all

The observation dial is one control over a corpus whose currency bound is **per timeline** — `k`
cadences of the *tightest covering* `Scan` for that `(subject, facet, port, vantage)`. One dial cannot
carry a per-timeline bound, so its floor must be the **longest** bound in force, or it discards
something still live.

Rendering that forced the enumeration, and the enumeration found the hole
[#142](https://github.com/winniel123/verge-asm/issues/142) is open on. Four facets have a covering
`Scan`. **`resolution` and `dns-record` do not** — except on the timelines a supplied zone file
sources, which the `zone` `Scan` covers. An observation with no covering `Scan` has no currency bound,
so it never becomes evidential, so **this dial never reaches it and it is never discarded.**

That is drawn as a named exclusion beneath the dial rather than left implicit, on the same ground as
#47's empty group: a dial whose reach is silent is a dial the operator believes covers everything. And
it is why the dial's reach is a **rendered list**: #142 moves rows between the two halves of that list
and moves nothing else on the screen.

### Two floors that compute to one number today, and are not one rule

The `Dispatch` floor is `k` cadences of the **slowest enabled** `Scan`, because below it `Coverage`
cannot answer whether the slowest scan ran. The observation floor is `k` cadences of the **slowest
covering** `Scan`, because below it a derivation loses an observation it is still allowed to read.
Different questions, different populations — and on every install drawn here they produce the same
number.

That coincidence is the reason each dial renders **its own derivation** rather than the two sharing
one floor line. A shared number invites the next session to implement one floor and use it twice, and
the two come apart the moment a `Scan` is enabled that covers nothing — which `zone` on an install
with no name scope holding a zone file is already close to.

### One dated figure this drawing found, and the rule that makes it not matter

ADR-0041 states the multiple rule — *"stated as a multiple and never as a day count, the cadence being
a quantity the operator moves"* — and then, two sentences later, states a day count: *"today that is a
fortnight with `tls-acceptance` slowest, two months with the cold tier enabled."*

`zone` is the fourth `Scan` and ships at **monthly**
([#124](https://github.com/winniel123/verge-asm/issues/124)), so on any install that supplies a zone
file the slowest enabled `Scan` is `zone` and the floor is **two months, not a fortnight** — with no
cold tier anywhere near it. The figure is a dated record and this ADR does not repair it; the
structural repair is that the screen **computes** the floor from the enabled `Scan` set and renders
the multiple, so the interface cannot carry a stale fortnight in the first place. Same shape as
ADR-0004's *existing suppression* contradiction that #119 found: a document stating a rule and then
violating it in its own next paragraph.

## Consequences

- **[`CONTEXT.md`](../../CONTEXT.md) is amended in two entries and gains no term.** `Dispatch` loses
  *"the whole of the operational record"*, which #119 falsified by adding two more Operational terms,
  and gains the enumeration of which of the three has a dial. `Delivery` gains that it travels with its
  `Message`.
- **[`notification-channels.md`](../spec/notification-channels.md)'s *nothing is lost by compacting it
  aggressively* is withdrawn at the site that specifies it**, per
  [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md), **with a
  replacement rather than a strike**, per ADR-0057: a sentence naming no successor is re-derived by
  the next session that needs one.
- **ADR-0041's `Coverage` rendering obligation is discharged**, and its *two dials* survives contact
  with a fourth corpus. The screen renders **seven corpus rows and two dials**.
- **The dials live on `Coverage` and this does not make it an operations screen.** #22 refused the
  operations framing and that stands: these two controls change nothing operational, they change how
  far back `Coverage`'s own answer reaches, which is its question in the time axis. `Settings` links
  here rather than carrying a copy.
- **The retention panel carries no coverage figure, ever** — no percentage, no bar, no threshold, and
  none of #28's three statement densities. It is the mirror of #28's rule that the *propose* half
  carries no number: a coverage figure here would say retention is a completeness fact, and it is a
  consequence of what the operator declared.
- **A fourth register on `Coverage`.** #28 established three — a fault, an invitation, and a boundary
  the operator drew. This is a fourth and it is the quiet one: *a consequence of what you declared*.
  It may never take the fault or the invitation treatment.
- **This ADR binds beyond retention.** Any bounded operator dial draws its floor as territory; any
  default-unbounded or default-off setting is a position, a price and a ground.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **A third dial for the fourth Operational corpus** — the reading `notification-channels.md` hands forward, and the strongest loser here | It fails ADR-0041's own test. *Nothing in the comparison path reads it* is not *nothing reads it*: a message holds its delivery outcomes, so discarding a dead-lettered `Delivery` converts *we could not reach you* into *we told you*. And a dial on the `Message` store is a supported way for the one surface that cannot fail to fail — the property every notification refusal in ADR-0039 was purchased with |
| **Compact succeeded `Delivery` rows, keep failed ones** — the finer cut, and genuinely lossless on its face | It makes the retention rule read the row's **outcome**, so the corpus's rule is a function of what happened in it. It also loses *delivered at 14:02 after two attempts*, which is read by the same person reading the message. And it buys a two-tier rule inside a corpus that is a rounding error — real complexity for no disk |
| **A floor enforced as a validation error on a number field** | An error is a response to a choice, so a rejected value is a value the operator was offered. It puts the floor inside the operator's territory and then tells them off for standing there, and it teaches them to look for the setting that lifts the minimum |
| **`∞`, `unlimited`, or an empty field for the unbounded default** | Both read as *nobody decided*. A sentinel is what a program writes when it has no answer; an empty field is a chore. The defence needed is against *nobody chose*, not against *the number is wrong* |
| **Put the dials in `Settings` and render a read-only horizon on `Coverage`** | The dial's only legitimate justification is the projection, and the projection is on `Coverage`. Splitting them puts the number on a screen that cannot show what it costs, which is the exact failure ADR-0041's projection exists to prevent |
| **State *retention may never be the tighter clamp* as copy beneath the horizon** | An invariant stated in prose has to be believed. As a sort order over every clamp in force it is visible: retention would have to occupy first place, and first place is drawn in ink and names a leaf |
| **Hide the projection panel on installs with no declared address scope** | It teaches the >99% of installs that declare no address scope that there is nothing here to know. It renders and says it has no declared denominator — #47's rule, and the case it was written for |
| **Project a row count from the measured subject count where nothing is declared** | It is a forecast about an estate we have already ruled we cannot enumerate — #28's refused completeness score arriving through arithmetic. What we hold may be shown as a count of what we hold; it may not be multiplied out into a year |
| **One floor computed once and used by both dials**, since they agree on every install drawn | They agree by coincidence over two different populations — slowest **enabled** against slowest **covering**. Sharing the number invites the next session to build one floor, and the two come apart the first time a `Scan` covering nothing is enabled |
| **Render the floors as day counts, since the screen knows the cadences** | ADR-0038, and ADR-0041 already demonstrates the failure by stating a fortnight two sentences after forbidding day counts. The multiple and the named `Scan` cannot go stale; a fortnight already has |
| **Show the discarded group only once something has been discarded** | A group that appears only after a loss is a group nobody reads until after they have lost something. #47 verbatim, and v1 is the case where it is *always* empty, which is what makes it the test rather than the exception |

## Where this is thin, stated rather than smoothed

- **That an unbounded default reads as chosen rather than broken rests on a rendering judgement, not
  on a measurement.** Nobody has put the three renderings in front of an operator. The claim is that
  a parked handle on a labelled stop, a price and a stated ground defeat *nobody decided*; it is
  argued from what the two rejected renderings obviously signal, which is weaker than testing it.
- **The `Message` corpus's size is asserted rather than computed.** *Messages are a rounding error* is
  an inference from *alerting fires at the cause with a census*, and nobody has measured how many
  messages a year an unstable estate writes. If it is wrong, the structural argument still holds — the
  store may not acquire a way to fail — but the second leg of the reasoning goes.
- **The observation floor being the *longest* bound in force is derived here rather than cited.**
  ADR-0041 says the dial is *"floored at the currency bound"* and the currency bound is per-timeline;
  turning one per-timeline bound into one floor over one corpus is this ADR's step, and the
  conservative direction was chosen without an argument that the tight direction is unsafe for every
  timeline it would touch.
