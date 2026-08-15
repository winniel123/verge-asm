# ADR-0074: A narrowing is carried by the subject it leaves behind — and where it takes the carrier with it, it fires at the scope

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#130 Does narrowing a `Seed` reach anybody?](https://github.com/winniel123/verge-asm/issues/130)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)
- **Amends:** [ADR-0047](./0047-an-address-scope-is-its-own-enumeration.md),
  [ADR-0014](./0014-only-revealed-generalises.md)

## Context

[ADR-0047](./0047-an-address-scope-is-its-own-enumeration.md) settled both directions of an address
scope in one table and argued only one of them.

- **Widening**: *"Declaring or widening an address scope … an **aperture widening**: `revealed`,
  **one** coverage-class message at the scope carrying a count of timelines opened. **Never** 1,024
  `appeared` messages."* Four paragraphs of rationale sit under it.
- **Narrowing**: *"Narrowing one (an exclusion, or a smaller CIDR) … The addresses **leave**, taking
  their timelines, unless a current resolution still cites them — **no `Gap`**."* Its companion row
  in the four-object table cites its authority as **[ADR-0013](./0013-custody-is-control-and-extends-by-declaration.md)
  §5's table, by parity**.

Neither narrowing sentence mentions a message, and neither was arguing about one: both rule on the
**span mechanism** — does a `Gap` open? — and ADR-0013 §5, the parity source, is a table about
`Custody` moves caused by **measurement**, closing on a claim about drift rather than about
messages. The silence on the notification question is a **by-catch of an enumeration ticket**, not a
ruling. [#119](https://github.com/winniel123/verge-asm/issues/119), the ticket that owned
notification, never saw it, because it was not in the notification corpus.

[#127](https://github.com/winniel123/verge-asm/issues/127) saw it, named it *"the asymmetric hole,
and the sharpest residue this ticket leaves"*, and routed it here with one constraint that governs
this ADR's whole shape: it **needs no actor to answer**. A narrowing message carries a count exactly
as the widening one does. Nothing here reopens #127's refusal of an operator-act record, and no
Declared term acquires an author.

**One refinement of #127's framing, because it changes what is being priced.** *"No message and no
record"* is right about the timelines and wrong about everything else. The subjects' `Span`s **up to
the act** are closed and kept — [ADR-0041](./0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md)
ships no expiry on the span corpus and never compacts it — and the next `Batch` records a smaller
completed scope, which is the **same diff that detects a widening**, since ADR-0014 defines aperture
as *"what a `Batch` records as its completed scope"*. So the detection input already exists and
nothing new is stored.

What is genuinely gone is the **subject**, and after [#140](https://github.com/winniel123/verge-asm/issues/140)
· ADR-0082 that can be stated exactly: **the withdrawn period is on no timeline at all — neither a
value nor a `Gap`.** The history stops; nothing records that it stopped, or when. Only the message
was missing, and it is the only object that could ever carry the instant.

## Decision

**A narrowing is carried by the subject it leaves behind. Where the act takes that subject with it,
there is no carrier at all, and the narrowing fires one coverage-class message at the scope.**

| Concern | Decision |
| --- | --- |
| Narrowing a `Seed` over ground nothing else cites | **One coverage-class message, at the scope**, on the act |
| Narrowing over ground a current resolution still cites | **Nothing new fires.** The subject survives, the gate closes, currency opens the `Gap` — [ADR-0026](./0026-the-facet-layer-is-evidence-not-a-channel.md) §6's *one cause, one message* |
| A mixed narrowing | **One message at the scope** for the subjects withdrawn; the survivors ride their `Gap`s. Both fire **at the cause**, so the operator sees one event |
| A narrowing that withdraws **no** subject | **Silent.** The trigger is inhabitance of the withdrawn set; an empty residue fires nothing |
| Which `Seed` kind | **Both.** The rule reads membership ground, not enumeration, so it does not split on ADR-0047's address/name cut |
| Class | **Coverage** — *we changed how we look*. No fifth cause, no fourth class |
| Member | **A TENTH member of the coverage class.** The cost is real and is paid deliberately; see *The tenth member is a real cost* below |
| Payload | A count of **subjects withdrawn** and of the timelines they take out of the estate, at the scope, **no comparison and no rows** — plus **the loss, named** |
| Wording | **Not this ADR's.** [#120](https://github.com/winniel123/verge-asm/issues/120) owns the copy |
| Per-subject messages | **Refused**, as in the widening direction — ADR-0022's unit rule and ADR-0047's own *"never 1,024 `appeared` messages"* |
| An operator-act record | **Untouched.** #127 stands; this message names no actor and sits in no Declared term |

## Rationale

### `Seed` narrowing is the product's quiet-shrink route, and the corpus already knew it

This is the decisive argument, and it is a safety argument rather than an accounting one. It was
made in this repository before this ticket existed.

[ADR-0009](./0009-verge-core-is-a-union.md) closed the port-list route to muting a rule by name —
***"a port the operator can hide is a signal the operator can silence"*** — and pointed the operator
at the scope declaration instead. [ADR-0016](./0016-an-annotation-moves-a-message-never-a-number.md)
then walked the consequence in terms:

> *"Refuse the annotation as well and the **only** remaining act is the one
> [#17](https://github.com/winniel123/verge-asm/issues/17) already flagged as the **quiet-shrink
> route**: narrow the `Seed`. That stops the measurement, takes the subject's timelines with it, and
> **buys silence at the price of the drift detection the product exists for**."*

[ADR-0006](./0006-subjects-leave-by-measurement.md) says the same thing about the same act from the
other end: exclusions get a place on `Coverage` *"because they are the one route by which an operator
can **silently shrink the estate until the board looks clean**."*

So three ADRs already hold that narrowing a `Seed` is the one destructive, board-cleaning act in the
product. ADR-0016 was content to leave the operator that route, and it was right to be: the model
should not prevent an operator from redrawing their own boundary. **What it never examined is that
the destructive route is also the quiet one.** ADR-0009's sentence, read exactly, is a rule about
what must not be possible — *a signal the operator can silence* — and the `Seed` route satisfies it
only because it destroys the measurement openly enough to be called expensive. It is not open. It
emits nothing.

The repair is not to close the route. It is to make it **loud**, which costs the operator nothing
they wanted and removes the property ADR-0009 refuses.

### The structural reason it is silent is a carrier failure, not a policy

Every other aperture narrowing leaves a subject behind to hold a `Gap`, and the `Gap` is the carrier.

- Withdraw a `custody extension` while resolutions still cite the addresses → the addresses stay, the
  gate closes, currency opens a `Gap` *"naming the operator's own act"* (`CONTEXT.md`, ADR-0013 §5).
  That is coverage member 3.
- Turn the cold port tier off → *"a narrowing, which opens a `Gap` on every timeline it fed once
  currency lapses"* ([ADR-0044](./0044-a-one-off-measurement-has-no-currency.md)). That is member 8.
- A signal loses its evidence → `fired` → `not-evaluable`, member 5, *worded as we stopped looking*
  ([ADR-0010](./0010-exposure-composes-two-reaches.md)).

Narrow a `Seed` over ground nothing else cites, and **the subject goes with it**. There is no
timeline to hold a `Gap`, no row to render one on, and nothing for a `Transition` to be derived over.
ADR-0026's sole-carrier rule — *a `Transition` is a message exactly where it is the sole carrier of a
fact the operator asked for* — has **no application**, because there is no carrier at all. The model
routes *we stopped looking* through the subject, and this is the one act that removes the subject.

[#140](https://github.com/winniel123/verge-asm/issues/140) · ADR-0082 sharpens this to its strongest
form. It **confirms** ADR-0041's *a withdrawn subject's timelines close* — the sentence is no longer
flagged as thin — and states the corollary: **the withdrawn period is on no timeline at all, neither
a value nor a `Gap`.** Two of its grounds bear directly here. A **closed span is free to keep while
an open span must be fed**, which is why nothing is retained to mark the interval. And **withdrawal
needs every available vantage to agree, so it is a fact about the subject, while a `Span` is keyed
per `(vantage, source)`** — the fact is of a kind only the subject can bear, and the subject is what
the act removes. So the carrier failure is total: not merely *no reachable owner*, but **no timeline
anywhere on which the removal is a fact**. A message is therefore the only object that can date the
act at all.

**This is why it is not a symmetry argument, and it matters that it is not.** ADR-0014 exists to
refuse symmetry: *only `revealed` generalises*, and the opening family got one general member while
the closing family got none, on the ground that the model names causes rather than mirror images.
Minting a message *because the other direction has one* would be that defect. What is minted here is
minted because the fact has **nowhere else to live**.

The only object that survives the act is the `Seed`, and the scope is already the firing site the
widening uses. **No new firing site is added.**

### The corpus's standing *shrinking is silent* precedent does not reach this

The strongest objection is that this project silences the shrinking direction everywhere:
`withdrawn` (ADR-0006, #17 — and `withdrawn` is never a message on any subject kind, in any
direction); the internal `Reach` leg both ways and the internet leg's `reached` → `not-reached`
([ADR-0029](./0029-an-alert-fires-on-a-leg.md) §3, §4); `Resolved` → `NoData` and *"any move that
only removes addresses"* (ADR-0026 §1); ADR-0013 §7's *"a departure does not fire it"*; and
[ADR-0025](./0025-an-offer-is-scope-only-where-the-value-enumerates-it.md)'s *"a narrowing is the
opposite of `revealed`"*.

**Every one is a drift-class silence about the *world* shrinking, each on a stated ground that does
not transfer.** #17's is decommissioning noise. ADR-0029 §4's is that a closing port is
*"overwhelmingly the operator's own remediation"* **plus three named carriers that catch the cases
where it is not**. ADR-0013 §7's is that a departure is *"§4's self-correction working and the gate
narrowing"* — a rider guarding against **over-reach**, and a closing gate reduces over-reach, so it
has no hazard to report. ADR-0025's is about what a **scope record** can license, not about who is
told.

A `Seed` narrowing is not the world shrinking. It is **our aperture shrinking by a Declared act** —
the coverage class's own subject matter. And the coverage class has no *shrinking is silent* rule; it
has the opposite. *We stopped looking* is one of the four causes, and members 3, 5 and 8 are all
messages fired because we stopped looking at something.

The question was never *may a shrink speak?* It is **why does this shrink alone not speak, when the
class named for it speaks everywhere else?**

### The coverage figure can only be improved by shrinking what it counts

ADR-0047 gave the address axis the one exact, non-estimated denominator in the product: `Coverage`
may state *1,024 addresses declared in `198.51.100.0/22`; 1,024 measured on the daily tier*, and it
is *"honest, closed and ours on both sides"*.

Narrow that scope to a `/32`. `Coverage` reads **1 declared, 1 measured**. Still 100%, still honest,
of a different thing — [#43](https://github.com/winniel123/verge-asm/issues/43)'s rule verbatim:
*"100% coverage" is always coverage of something — check what, before quoting it.* Under the
incumbent silence the product's own completeness figure moves in the **reassuring** direction with
nothing to date it against. That is [#14](https://github.com/winniel123/verge-asm/issues/14)'s
false-reassurance failure arriving through the one layer #14's guard does not reach.

ADR-0006 already routed exclusions to `Coverage` for exactly this reason, and that surface is the
best thing the incumbent silence has. It is not enough, twice over: it is a **standing state and
never an event**, so it says nothing at the instant and nothing afterwards to anyone not looking at
that screen; and ADR-0006's route is written for **exclusions**, while a *smaller CIDR* replaces the
declaration outright and leaves no exclusion row to render.

The narrowing is also the exact inverse of ADR-0047's decisive argument for enumerating at all. That
argument was that under a bounding reading *"a machine appearing in that space"* could never fire the
flagship, because no prior span would exist. **A narrowing restores that condition by declaration,
over the removed ground, permanently.** The instrument's own justification for the dark rows is the
measure of what a narrowing costs.

### The tenth member is a real cost, and it is paid rather than argued away

The tidy answer is that this is a third **trigger** on an existing member, since ADR-0014 already
merged the re-baseline and the aperture widening into *"one class with two triggers, named for the
cause and not the mechanism"*. **That reading is wrong and is not taken.** The class in that sentence
is the *cause*-class — one of the four causes — and its two triggers are **two separate members** of
the coverage class: `revealed` is member 2 (ADR-0029, ADR-0031 both call it that) and the
`Derivation` vector move is member 4. The class enumerates by trigger, one member per row.

So this is **member ten**, and the coverage class stands at **ten** from here.

That is the honest price and it is worth naming, because two ADRs took visible care not to pay it —
ADR-0026's *"Nothing is minted … the class stays at nine"* and ADR-0033's *"No fourth class. No new
coverage-class member."* Both are **dated records of what those tickets did**, and both were right:
in each case an existing member already carried the fact. Here nothing carries it, which is the whole
ruling. A member is minted exactly when the alternative is that a fact reaches nobody — which is the
test those two ADRs were applying, arriving at the opposite answer because the input is different.

**What does not move**: no fifth cause, no fourth class, no transition name in the closing family,
and no change to the **clock class, which stays at three**. The census payload still has **five**
producers — this message carries a **count**, not a census.

### The payload names the loss, because there is no repair

ADR-0014 fixed the widening payload as *"a count of timelines opened and **no comparison at all**"*,
and that discipline holds here: no difference set, no rows (ADR-0039 §3), one computation at the
cause.

The narrowing payload takes one element that is not a mirror. **It states what can no longer be
told**: a listener appearing in the removed ground after this act is invisible, and no later message
recovers it. The precedent is exact and one document across — ADR-0041's third element on the
vector-move payload, which *"must also **state the loss**"* because *"it cannot be corrected
afterwards — history is never re-derived — so naming it is the whole of the remedy."*

Stating a loss is not a comparison: ADR-0029 §7 already settled that a census is not one, and a bare
count with a named consequence is weaker still.

### What the operator already knowing does not buy

The best form of the losing case is that the operator typed the narrowing thirty seconds ago, so the
message is a receipt, and ADR-0039 §1 makes the channel *the escalation, not the product*.

**The store copy is not an escalation.** ADR-0039 §1 writes and renders every `Message`
*"unconditionally — no configuration, no enable, no routing, no `Delivery`, and no way to turn it
off"*. *Does it reach anybody* is answered in the store first; the channel follows from the class.

**The reader is not necessarily the actor.** A `Channel` is a URL pointed at a destination this
project knows nothing about.

**And leaning on *the operator knows because they did it* borrows #127's thin ground.** #127 rested
its deferral on *the modal install has one mutating role*, flagged it as *"a reading of #11's role
split, not a measurement"*, and wrote a reopening condition against it. A ruling built on that
assumption inherits the condition. This one does not need it: the fact has no carrier however many
operators there are. The **actor** question and the **audience** question are different, and only the
first is #127's.

### Volume, and why no damping is owed

A narrowing is a Declared act, so its rate is bounded by operator typing rather than by a cadence.
Address scopes are under 1% of installs ([#26](https://github.com/winniel123/verge-asm/issues/26)),
the message fires once at the scope, and v1 ships no coalescing because class routing suffices
(ADR-0039 §6). An act-rate-bounded message cannot be the measured volume that reopens that ruling.

One act producing both a message at the scope and `Gap`s beneath the survivors is ADR-0014's own
accepted shape in the opening direction: *"re-opening the probing gate produces both kinds of edge at
once … One operator action, two vocabularies. That is tolerable only because ADR-0007 already fires
alerting at the cause rather than per affected subject, so the operator never sees the seam."*

## Consequences

- **[`CONTEXT.md`](../../CONTEXT.md)'s `Seed` entry gains one sentence** and **no term is added**.
  The entry already specifies the widening message at that exact site; the narrowing case is written
  beside it so the contrast stops being a specification of silence. `Custody`'s and `Address`'s
  narrowing sentences are **not** amended — they specify the span mechanism, which is untouched.
- **The coverage class moves from nine members to TEN**, and that figure belongs on the map's
  composed-state line. ADR-0026's and ADR-0033's *"the class stays at nine"* are dated records of
  those tickets and are left standing per the name-and-withdraw convention; the map's own rule is
  that a count written once and copied is the defect.
  [#120](https://github.com/winniel123/verge-asm/issues/120) is reading **nine** and must carry
  **ten**. Its other two figures are untouched: the **clock class is still three** and the census
  payload still has **five** producers.
- **ADR-0047 is amended at both narrowing rows.** Per
  [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md), a reader
  taking those rows alone beside a widening row two lines up that **does** specify a message would
  build the silence — the contrast is itself a specification. The span mechanism in both rows is
  untouched and is this ADR's premise, and the four-object row's *"by parity"* attribution is
  narrowed: parity holds for the span mechanism and **not** for the message.
- **ADR-0014 is amended at two sentences** — *"one class with two triggers"* and *"the payloads
  differ and should"*. Nothing in its Decision moves and **`revealed` does not generalise to the
  closing direction**: no transition name is minted, in this or any family.
- **ADR-0013 §7's *"a departure does not fire it"* is confirmed, not amended.** It governs a
  `custody extension` narrowing **because the world moved** — §4's self-correction, whose hazard is
  over-reach. This ADR governs a **Declared act** removing membership ground, whose hazard is
  blindness. Recorded explicitly, because the two sentences read as a conflict and are not one.
- **ADR-0009's *a port the operator can hide is a signal the operator can silence* is discharged one
  route further**, and ADR-0016's quiet-shrink route stops being quiet. Neither is amended: both
  named the hazard correctly and neither claimed it was closed.
- **[`safe-active-probing.md`](../research/safe-active-probing.md) §11's per-target-disable sentence
  is repaired.** It reads *"an aperture narrowing … a `Seed` **exclusion** … **which opens a `Gap`
  and says so** — never a hidden mute"*, which contradicted ADR-0047's *no `Gap`* on the mechanism
  while asserting the notification this ADR now supplies. Half of it was wrong before this ticket and
  the other half is right after it: no `Gap`, and it does say so.
- **Nothing new is stored and no retention question opens.** The detection input is the `Batch`
  scope diff ADR-0014 already defined; the departed `Span`s were already retained by ADR-0041; the
  `Message` corpus already exists under ADR-0039.
- **`Seed` deletion is not assumed to exist.** The corpus's narrowing vocabulary is exclusions and
  smaller CIDRs; no ADR, glossary entry or shipped surface offers a delete. If one is ever built it
  is the maximal narrowing and fires by this rule, and the message carrying the key of a scope that
  no longer exists is correct — a `Message` is Operational, *"written once and never recomputed"*,
  so a dangling key is a dated fact rather than a broken join.
- **One neighbour is named and not ruled.** Disabling a `Source` is also an aperture narrowing, and
  a subject that source alone admitted (ADR-0027) loses its `Citation` and leaves — the same carrier
  failure. v1 routes source enablement to `Coverage` as a standing state and no message, which is
  [#47](https://github.com/winniel123/verge-asm/issues/47)'s and
  [#15](https://github.com/winniel123/verge-asm/issues/15)'s territory rather than this ticket's. The
  rule stated here reaches it; whether it fires there is a successor, and it is the only place this
  ADR is knowingly incomplete.
- **[#140](https://github.com/winniel123/verge-asm/issues/140) · ADR-0082 is upstream and
  strengthens this rather than conditioning it.** The dependency was written to survive either
  ruling — the trigger is inhabitance of the withdrawn set, and the payload counts **subjects
  withdrawn**, so neither reads the shape of what a withdrawal leaves behind. #140 confirmed
  ADR-0041's sentence and added the corollary that the withdrawn period is on **no timeline at all**,
  which converts this ADR's carrier argument from *no reachable owner* to *no bearer anywhere*. Had
  #140 gone the other way — a re-openable `withdrawn` span — the ruling would be unchanged and one
  paragraph weaker.
- **#127's *"no message and no record"* is refined rather than reversed** — the record of the history
  **up to** the act exists in the retained `Span`s and the act itself is visible in the `Batch` scope
  diff; what does not exist is any timeline on which the removal is a fact. Its ruling, no
  operator-act record, is untouched.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Keep the silence** — the incumbent, and what ADR-0047 currently specifies. Its best form: the corpus silences every shrink five times over, the operator performed the act, a message to its own author is a receipt, exclusions already surface on `Coverage`, and a tenth member is a real cost for an event under 1% of installs will ever see | **The losing option.** It fails ADR-0026's own formula for everything the model silences — *"recorded, queryable, and silent"* — because this is the one case that is silent **and has no reachable owner**, the act removing the subject that would have held the `Gap`. Its precedents are all **drift-class** silences about the **world**, each on a ground that does not transfer. Its `Coverage` carrier is a standing state rather than an event and is written for exclusions, not for a replaced CIDR. And its remaining ground — *the operator knows, they typed it* — is #127's thin single-admin assumption borrowed, with #127's reopening condition attached. Above all it leaves ADR-0016's **quiet-shrink route** quiet, which is ADR-0009's *a signal the operator can silence* surviving at the one door nobody checked |
| **A confirmation or receipt at the act instead of a message** — #50's confirm control and #123's variant-C copy are precedent, and the widening act already renders one | Reaches **only the actor, only at the instant**, and nobody afterwards — which is the half the residue is about. It is also not an alternative: the surface is #123's and the copy #120's, and a count in the confirm dialogue is compatible with the message rather than a substitute for it. Note that the corpus currently refuses a confirmation step for the adjacent narrowing act on the ground that *"off is the safe direction"* — true of over-reach, false of blindness, which is this ADR's whole distinction |
| **Write a record and render no message** | #127 §4's general form: *"if it is worth writing it is worth rendering; if it is not worth rendering it is not worth writing."* Unnecessary twice over — the `Batch` scope record and the retained `Span`s already hold it |
| **Call it a third trigger on an existing member, so the class stays at nine** | The tidy answer, and it misreads ADR-0014. Its *"one class"* is the **cause**-class; its two triggers are members **2** and **4**, enumerated separately everywhere the class is counted. Keeping the figure at nine would be buying a number by mis-describing the enumeration — the defect the map's dated-figures discipline exists to prevent |
| **A fourth cause, or a closing-family transition name mirroring `revealed`** | ADR-0014's central ruling. `revealed` generalises because it describes **our looking**; the message is at the scope and no timeline gets a new word |
| **One message per withdrawn address** | 768 messages for one operator act. ADR-0022's unit rule, ADR-0013's once-per-scope shape and ADR-0047's *"never 1,024 `appeared` messages"* all agree |
| **Fire on every narrowing, including where every affected subject survives** | Two messages for one fact: the survivors' `Gap`s already carry it, and ADR-0026 §6's *one cause, one message* forbids the second representation |
| **Route it to the drift class, since the estate got smaller** | The estate did not get smaller. **Our boundary did.** Filing an observer act under *the world moved* is ADR-0031's *rooting an appearance at the `Seed`* defect with the sign flipped, and it would land on a channel routed for world events |
| **Close the quiet-shrink route instead — refuse or gate the narrowing** | ADR-0016 and ADR-0013 both hold that the operator redraws their own boundary and the model never adjudicates the truth of a declaration. The defect was never that the route exists; it was that it was the **quiet** one |
| **Carry the actor, since we are minting a message about a Declared act anyway** | #127, verbatim and unreopened. The message names a scope and a count; operator identity stays unjoinable to the probing gate |
| **Amend ADR-0013 §7 so a custody-extension departure fires too** | A different cause with a different hazard — a measured narrowing that reduces over-reach. Confirming it is the honest act; amending it would be symmetry doing the reasoning again |
