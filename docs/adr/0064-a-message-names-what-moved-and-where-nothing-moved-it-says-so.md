# ADR-0064: A message names what moved — and what moved is read from the fold, never from the rule

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#120 Can one vocabulary carry all four message causes — and what does a seven-figure count read as?](https://github.com/winniel123/verge-asm/issues/120)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0039](./0039-a-channel-carries-the-message-never-the-estate-and-a-delivery-is-an-operational-record.md)
settled where a `Message` goes and refused, in terms, to settle what it says: *"What the operator
does not get: the message set, any predicate, the payload, **the wording**"*. The wording is the
project's, and nothing in the corpus has ever said what one of these sentences reads like.

What is settled and is **not** reopened here: **four causes** — the world moved · we stopped
looking · we changed how we look · a clock crossed — partitioned into **three classes** that
partition **messages** rather than events; the coverage class carrying ~~**nine**~~ **ten** members
and two of
the four causes, the third cause carrying **two triggers whose payloads differ and must not be
levelled** ([ADR-0014](./0014-only-revealed-generalises.md)); the clock class carrying **three**
([ADR-0004](./0004-signals-are-release-coupled-rules.md)); the census payload's **five producers**
([ADR-0033](./0033-a-move-carries-the-rule-that-opens-at-fired.md)); routing by class and by nothing
finer, and **no coalescing and no flap suppression in v1** (ADR-0039 §4, §6).

> **`nine` is superseded here, at the site that states it — the class is `ten`.**
> [#130](https://github.com/winniel123/verge-asm/issues/130) ·
> [ADR-0074](./0074-an-aperture-narrowing-that-takes-its-carrier-with-it-fires-at-the-scope.md)
> resolved in the same parallel batch as this ADR and minted a tenth member: a `Seed` narrowing that
> removes membership ground nothing else cites fires **one coverage-class message at the scope**,
> because the act takes the subject that would have carried the news away with it. Both passes
> branched from `4001e4c` and neither could see the other; this ADR read **nine** and confirmed it,
> which was correct when it was written and is not now. **Nothing else in this paragraph moves** —
> the clock class is still **three** and the census payload still has **five producers**, both
> re-checked on merge. Recorded by the merging session, per
> [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md): the
> superseded count, read alone and in the present tense, would have a competent session build a
> nine-member class.
>
> **And *"two triggers"* is superseded here too, by the same ADR — it is `THREE`.** Recorded by
> [#158](https://github.com/winniel123/verge-asm/issues/158) ·
> [ADR-0091](./0091-the-routing-unit-is-the-class-and-the-cause-is-refused-as-a-routing-key.md).
> ADR-0074 minted the `Seed` narrowing as a **third trigger** of *we changed how we look* and a
> **third payload** that must not be levelled with the other two, marking both at
> [ADR-0014](./0014-only-revealed-generalises.md)'s own site. The paragraph above says *"nothing else
> in this paragraph moves"* and names what it re-checked — the clock class and the census producers,
> both of which held. The trigger count was not among them, so the disclaimer is true of what it
> checked and false of this. **It moves no conclusion in either ADR**: three triggers under one cause
> makes the coverage class **coarser** than nine-and-two did, and every argument in this document
> that turns on the class's coarseness is strengthened rather than disturbed.

Three things arrive here needing an answer.

**Nobody has written a sentence.** Twenty-odd messages have been ruled into existence one ticket at
a time, each named by its cause and none by its words. Five **wording pairs** are outstanding — two
messages that can arrive from one fold and must not read as duplicates — and each was deferred to
"the notification patch" on the assumption that a patch would arrive with a vocabulary in it.

**ADR-0039 flagged a class-assignment conflict, refused to rule it, and named this ticket.**
[ADR-0026](./0026-the-facet-layer-is-evidence-not-a-channel.md) §5 puts `not-fired` → `fired` in the
**drift** class *"for all seventeen"* rules;
[#60](https://github.com/winniel123/verge-asm/issues/60) and ADR-0004 put the three
certificate-lifetime rules in the **clock** class, because they become true with no new observation.
Both cannot be right, routing is by class, and ADR-0039 §6's stated residue — *the ACME flap has no
remedy in v1* — turns on which one is.

**The aperture-widening message has never been drawn at magnitude.** ADR-0014 fixes its payload as
*a count of timelines opened and no comparison at all*.
[#80](https://github.com/winniel123/verge-asm/issues/80) measured that count for the largest
widening a v1 operator can perform — a seven-figure number on a ten-address estate;
[#81](https://github.com/winniel123/verge-asm/issues/81) added a six-figure one that happens **at
onboarding**; [#85](https://github.com/winniel123/verge-asm/issues/85) capped the second, leaving the
seven-figure case exactly **one** producer.

### The corpus has already answered this shape twice, in two other places

[#22](https://github.com/winniel123/verge-asm/issues/22) was asked whether four different routes to
an incomplete answer need four treatments, and ruled **one treatment, stated reasons** — one visual
form carrying a reason slot, refusing four forms *"precisely because collapsing them trains the
operator to dismiss all of them"* and refusing four treatments because a reason is a slot, not a
design.

[#74](https://github.com/winniel123/verge-asm/issues/74) was asked whether a census member that is
most of the estate needs handling of its own, and ruled that **it takes no treatment of its own —
"its length was never what would have made it a findings list"**, with the rows off the card
entirely and the member enumerable in full.

Those are this ticket's two halves, one object down. What is left is to check whether the message
layer breaks either — and to find the reading of *cause* that survives contact with the fold.

## Decision

**A message names what moved. What moved is read from the fold — from the two adjacent spans the
message already sees — and never from the rule or the class that produced it. That one reading
decides both the message's class and the sentence's subject, and where nothing moved the sentence
says so.**

One grammar carries all four causes, because the causes differ in exactly one place: what the
sentence is about. That is a slot, and #22 already ruled that a difference which fits in a slot does
not earn a second treatment.

### 1. One grammar, and the subject is what moved

Every `Message` is one sentence with three parts: **what moved**, **what it now is**, and **what we
counted**. The first part is filled from the fold:

| What the fold says moved | The sentence is about | Worked form |
| --- | --- | --- |
| an object in the estate | that object, named by its key | *`admin.example.com:443/tcp` is now reached from the internet.* |
| our aperture, or a rule of ours | **us** | *We widened the port aperture on `10.0.0.0/28`.* |
| the operator's own declared input | **the operator** | *You stopped telling us: the zone for `example.com` has aged out of its window.* |
| **nothing** — a threshold was crossed | nothing; the sentence has no mover | *`admin.example.com:443/tcp`'s certificate expired on 2026-08-15. No measurement moved.* |

The rule is total and it is falsifiable: point at a message whose subject is not the thing the fold
says moved. It forbids three failures, and each is one this project exists to prevent.

- An **aperture** message with the estate as its subject — *214 services appeared* — reports our own
  act as growth in the operator's attack surface. That is ADR-0003's *a subject first observed under
  a widened aperture is not "appeared"* arriving at the sentence, and it is the whole of the
  seven-figure problem (§4).
- A **clock** message with the estate as its subject — *a certificate went bad* — reports the
  passage of time as movement, which ADR-0004 named as the thing *"a product whose claim is what
  moved since last time must not"* do.
- A **drift** message in the first person — *we detected an open port* — puts the observer where the
  fact belongs. The estate moved; we are not the news.

**The fourth form is the one nobody would have written.** *A clock crossed* has no agent, and the
honest sentence says so rather than manufacturing one. **Every clock-class message states that no
measurement moved**, in those terms, as a clause and not a footnote. It is the only cause of which
that is true, and saying it out loud is what keeps the clock class from reading as drift with a date
on it.

**The corpus had already built this rule without stating it, and the proof is the absence
vocabulary.** #44's absence vocabulary has **four registers** under **one** cause — *we never
looked* · *we stopped looking* ([ADR-0010](./0010-exposure-composes-two-reaches.md)) · #35's *you
stopped answering* · #48's *you stopped telling us*, the fourth added by
[ADR-0020](./0020-a-conflict-needs-two-enumerable-sources.md) as *"the first the operator causes"*.
Four registers, one cause, and nothing ever said why four. **This is why**: the cause is the same in
all four and the **mover** is not — twice us, once the estate's authority, once the operator's own
declaration. A vocabulary keyed on the cause would have to level them; one keyed on what moved
produces exactly these four and no fifth.
[ADR-0024](./0024-a-rules-domain-is-the-extension-of-its-name.md)'s fifth register —
*we measured; this rule cannot read the answer* — falls out of the same test, and
[ADR-0043](./0043-a-clock-reading-rule-bounds-its-evidence-in-the-subjects-own-units.md)'s rider on
it (*"the copy must not read as an outage"* where the cause is a short-lived certificate rather than
our machinery) is that rule policing itself.

### 2. A message's class is a property of the firing, not of the rule

This settles ADR-0039's flagged conflict, by taking the ticket's own premise literally: **the
classes partition messages, not events.** A message is one firing. So a class is read from the fold
per firing, and both prior rulings survive as written about different firings.

**The test is the one ADR-0033 §3 already uses and it needs no new machinery: did the fact the rule
reads move in this fold?**

- `certificate-expired` fires on an `Endpoint` whose `certificate` span is **unchanged** and whose
  `not_after` the clock crossed overnight → **clock class**. Nothing was measured; a threshold was
  passed.
- `certificate-expired` fires on an `Endpoint` where a deploy installed a **new** certificate that
  is already expired → **drift class**. The `certificate` timeline moved in the same fold. The world
  moved, in the direction the product exists to report.

**Where both are true in one fold, the world moved wins.** A renewal that installs a certificate
already past its horizon is a fact about the estate the operator can act on; the calendar is not.
The tie-break is stated rather than left to whoever implements it, and it breaks **loud**, on the
footing of *a dead-lettered `Delivery` licenses no silence*.

The arithmetic does not move. The clock class's **three** members are the three certificate-lifetime
rules, unchanged, and they are still the only rules in the v1 set that can become true with no new
observation (ADR-0004). What changes is that eligibility is not assignment. The other fourteen rules
read no clock at all, so every firing of theirs is drift by construction and ADR-0026 §5 is untouched
for them. `not-fired` → `fired` remains a **message** for all seventeen: nothing is silenced,
nothing is added, and only the class label on three rules' firings is now read from evidence rather
than from a rule's name.

**The ACME flap gets the remedy ADR-0039 §6 said it had not got.** The flap is
`certificate-expiring` firing every renewal cycle on an estate whose certificates change on
schedule — the span is unchanged and the horizon is crossed, so **every firing of the flap is clock
class**, and an operator who routes the clock class off a channel silences it exactly. The clearing
edge is already silent (ADR-0026 §5: four of seventeen alert on clearing, and they are the DNS
four), so the flap is one message per cycle and routing reaches all of it. Meanwhile a certificate
that arrives *already* expiring still reaches the drift channel, so the silencing costs the operator
nothing they would have wanted. **This is a class assignment doing work ADR-0039 §6 refused to give
a threshold.** It is not coalescing under another name: it delays nothing, holds nothing, reads no
clock of its own, and changes no predicate — the rule fires identically either way and only the
label on the message moves.

### 3. The vocabulary carries no valence word, and there is no severity

**No message may contain a word that says whether the news is good or bad.** Not *resolved*,
*fixed*, *cleared*, *improved*, *critical*, *warning*, *high*, *low* or *OK*. Not a colour standing
for one, and not a severity field standing for all of them.

**This is not new law; it is existing law reaching the object it was always about.**
[`CONTEXT.md`](../../CONTEXT.md)'s `Signal` entry already says *"a signal carries no severity: it is
a named fact"* — and then adds *"and urgency belongs to the transition that surfaced it"*, which,
read alone and in the present tense, hands the severity straight to the message. That clause is
narrowed here per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md):
urgency belongs to the transition in the sense that the transition is what makes the fact worth
saying, and in no sense that puts a grade on it.

Four independent reasons converge, and no one of them is load-bearing alone.

1. **A clear is not always good news** — [#35](https://github.com/winniel123/verge-asm/issues/35),
   [#48](https://github.com/winniel123/verge-asm/issues/48). ADR-0004 already fixes the wording at
   its specifying site: a clearing message *"is reported as **this changed** and never as
   **resolved**"*, and *"`fired` → `not-evaluable` … must never be worded as a clear either"*. Four
   rules of seventeen clear because somebody else may have claimed the operator's orphaned name, and
   on those four a clear can be the attack having succeeded. A vocabulary with a word for *good* has
   to remember not to use it at four sites; a vocabulary with no such word cannot get it wrong.
2. **A widening is neither.** A seven-figure count of timelines opened is not an alarm and not an
   all-clear; it is a description of our own aperture. Any valence on it is manufactured.
3. **Severity would be a threshold in the notification layer set from nothing.** Every proposed
   suppression in this project has died on an unmeasured base rate (ADR-0039 §6; the map has warned
   about the shape five times), and a severity is that threshold rendered instead of applied.
4. **It would perform the collapse #22 refused.** `Coverage` cuts four routes apart *"precisely
   because collapsing them trains the operator to dismiss all of them"*. A severity column is that
   collapse performed on the message store, and it is how an alerting product arrives at an operator
   who reads nothing below *critical*.

**What replaces it is the class, and the class is already there** — three of them, routable, each
saying one thing about who moved. That is the discrimination a severity pretends to offer, derived
rather than asserted, and it is why class routing earns its place as v1's only volume control.

**The four clearing messages keep their obligation and lose the word.** They state the edge, they
state that the clearing condition is a name a third party can claim, and they **name which timeline
gave way** — #48's attribution obligation and ADR-0020's *name which of the two sources moved*,
discharged inside the sentence:

> *`cname-target-name-error` stopped firing on `www.example.com`. Its CNAME target
> `legacy.example.net` now resolves; the timeline that gave way is `legacy.example.net`'s
> `resolution`. A target that starts existing is the operator re-provisioning it or another party
> claiming it.*

That is ADR-0026 §5's *"this improved and you should still look"* register with the first half
deleted and its whole content preserved.

### 4. A count is stated with its factors, and never as a bare product

**Every count in a message is rendered as the factors that produced it, followed by the product.
This is total — it is what a four-figure count gets and what a seven-figure count gets — and it is
why the seven-figure case needs no treatment of its own.**

> *We widened the port aperture on `10.0.0.0/28`. The full-range tier is now on: 65,404 more
> `(port, transport)` pairs × 16 addresses × 2 `Reach` legs — 2,092,928 timelines opened. Nothing is
> compared.*

The number an operator can check is never the product. It is the factors: they enabled one tier,
over a scope they declared, and the leg count is the model's. Each factor is an act they performed
or a list we ship; the product is arithmetic over them and is the only part of the sentence they
cannot audit. **Rendering the product alone is what makes seven figures unreadable, and it is not
the magnitude that does it.**

This is [#67](https://github.com/winniel123/verge-asm/issues/67)'s rule one layer across — *where a
constant is a fraction of a moving world quantity, ship the fraction rather than its product* — for
the same reason in both places: a product hides which of its inputs moved.

**And the magnitude question is #74's, already answered.** #74 asked whether a census member that is
most of the estate needs handling of its own and ruled that it does not — *"its length was never
what would have made it a findings list"* — with the card carrying no rows and the member enumerable
in full. A seven-figure count is that ruling's own case at a larger number: **length is not a
property that earns a treatment.** So the answer to *what does a seven-figure count read as* is that
it reads as a description of our aperture, legible through its factors, in exactly the sentence the
four-figure case gets.

Three riders, and the third is the only cost the magnitude actually carries.

- **The count is exact and is never rounded.** It is a count of our own objects, known to the unit;
  rounding it would be the one place in the product where a number we hold exactly is rendered
  approximately, against #22's *one treatment, stated reasons*. The corpus's `~1.3 million` and
  `≈287,000` are **the map's arithmetic over a stated estate size**, not the product's output, and
  this ADR restates neither: the product reads its factors at the cause, from the live tier and the
  declared scope.
- **It is a count of our own and never of the world.** #44 decision 7 governs, unchanged: counts of
  our own rules and lists, never a count or proportion of the operator's estate. *65,404 pairs we
  now watch* is admissible; *65,404 of 65,535 unread* is the refused estate-completeness score, and
  ADR-0044 rejected it in those words.
- **Enumerability does not weaken at magnitude, and this is the seven-figure case's one real cost.**
  #74's shape holds: the message card carries **no rows at all**, and the set behind the count is
  enumerable in full, never sampled, ranked, grouped or truncated. So a two-million-timeline
  drill-down **paginates**; it does not cap, and it offers no "top" anything. That obligation was
  already there and the magnitude only makes it expensive.

### 5. There is no magnitude-conditional treatment, and the guard sits on the act instead

No banner, no bar, no interstitial, no approximation, no second confirmation, and no threshold above
which the message changes shape. **A rendering that changes at a magnitude makes the message's
content a function of how big it is rather than of what happened** — ADR-0039 §7's own losing
argument with *magnitude* substituted for *when*, and it fails the same way: two installs performing
one act send different messages for one event.

Where a magnitude genuinely needs guarding, the corpus already guards it **before** the message.
ADR-0044 made the cold-tier enable **per `Seed` scope rather than estate-wide** on exactly this
ground — *"a switch whose consequence is a seven-figure count is not a switch to offer estate-wide"* —
and [ADR-0049](./0049-an-address-scope-is-family-agnostic-and-the-cap-counts-addresses.md) capped an
address scope at 1,024 addresses. The guard belongs on the **act**, where the operator can still
choose, and not on the **message**, which arrives after they cannot.

### 6. The five wording pairs are discharged by the grammar rather than one at a time

Five pairs are outstanding — #28/#22's, #55/#51's, ADR-0031's, ADR-0026's and ADR-0033's — each two
messages that can arrive from one fold and must not read as duplicates. **They are discharged
structurally: in every one of the five the two messages differ in class or in subject, so §1's
grammar makes them different sentences with nobody hand-tuning a pair.** The hardest is ADR-0031's:
an `Address` entering beneath a live `custody extension` fires *the gate your declaration holds open
has moved* (coverage, subject **us**) and *the estate grew, and here is what answered* (drift,
subject **the estate**). Two movers, two classes, two sentences.

Where a future pair shares both class and subject, the grammar does **not** discharge it and the
pair is owed a ruling. None of the five is that case, and stating the exception is what stops this
section being read as a claim that pairs can never collide.

### 7. What is withdrawn, and where

Per ADR-0058, each is marked at the site that **specifies** it, with a replacement supplied rather
than a strike alone ([ADR-0057](./0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md)).

- **ADR-0026 §5's *"drift class, for all seventeen"*** — narrowed at its sentence. It is a message
  for all seventeen; it is drift class for fourteen unconditionally and for the three
  certificate-lifetime rules only where the span they read moved in the same fold.
- **ADR-0026 §5's *"this improved and you should still look"* register** — replaced by §3's
  valence-free form, which keeps the second half and deletes the first. ADR-0004's *this changed,
  never resolved* is the specifying site and is confirmed rather than moved.
- **ADR-0004's clock-class sentence** — confirmed in its count and qualified in its reach: three
  rules are eligible for the clock class, and eligibility is not assignment.
- **`CONTEXT.md`'s `Signal` entry, *"urgency belongs to the transition that surfaced it"*** —
  narrowed, per §3. Read alone it licenses a severity on the message.
- **ADR-0038's *"the class #60 ruled is `certificate-expiring`'s only carrier"*** — a paraphrase
  about **carriers** whose literal reading is a per-rule class assignment. Marked at its sentence.
  The clock class remains `certificate-expiring`'s only *carrier*; it is not its only *class*.
- **ADR-0039's *"flagged, not ruled: it is a question about causes, and causes are #120's"*** —
  discharged by §2, and its §6 residue *the ACME flap has no remedy in v1* narrowed: class routing
  reaches it.

## Consequences

- **[`CONTEXT.md`](../../CONTEXT.md) is amended in two entries and gains no term.** `Message` gains
  the naming rule, the class-is-per-firing rule and the valence refusal; `Signal` has its *urgency
  belongs to the transition* clause narrowed. No entry is added, because a vocabulary is a property
  of an object the glossary already holds.
- **The interface gains no severity column, no colour scale and no filter but class.** The global
  message element (ADR-0039's Consequences) renders class, subject, sentence and time, and sorts by
  time alone — there is nothing else to sort by, which is the point. Recorded here rather than
  promoted, on ADR-0039's own precedent that the message surface's IA lives in the ticket.
- **The notification layer reads one more thing at the cause, and it is already in the fold.** For
  the three clock-reading rules the class is decided by whether the `certificate` span moved in the
  same fold — ADR-0033 §3's test, over the two adjacent spans, never reaching back across a `Break`.
  No timeline is touched, nothing is stored, and no predicate moves.
- **ADR-0039 §6's reopening condition is unchanged and better bounded.** v1 still ships no
  coalescing and no flap suppression; §2 removes the one **named** flap from the population that
  would have argued for one. The three unmeasured volumes — ADR-0026's re-point, ADR-0033's
  `NoHTTPResponse` → `Responded`, ADR-0031's `Name` `appeared` — are untouched and still unmeasured.
- **The seven-figure fog patch is discharged and produces no new object.** The patch asked whether
  magnitude needs a rule of its own. It does not, #74 had already ruled the identical question one
  object down, and saying so is the whole of the discharge.
- **No prototype is built and the reason is stated.** The fog patch's own ground for not being sharp
  was that *nobody has drawn either object at that scale*, and the eleven existing prototypes draw
  screens rather than messages. Drawing a message at 2,092,928 would test a **layout**, and the
  question here is what the sentence **says** — §1 and §4's worked forms are the drawing at the
  resolution the question needs. A message-surface prototype is a real successor and it should be
  built against this vocabulary rather than instead of it.
- **One arithmetic desync is named and is not repaired here.** ADR-0044's `65,395 pairs` and
  ADR-0047's `1,024 × 140 × 2 ≈ 287,000` both carry the retired **`~140`** hot-set figure, which
  ADR-0001, [ADR-0005](./0005-scan-execution-model.md) and
  [ADR-0009](./0009-verge-core-is-a-union.md) each measured and withdrew — `verge-core` is **136
  pairs, 131 of them TCP**, and UDP is outside v1's aperture. Read with 131, the full-range delta is
  **65,404** and a `/22` opens **≈268,000**, not ≈287,000. Neither correction moves any conclusion:
  the seven-figure case is still seven figures and still has exactly one producer. It is flagged
  rather than swept because this ADR **restates neither figure** — §4 makes the product read its
  factors live — so propagating the error was the only risk this ruling carried, and that risk is
  closed.
  **Since repaired**: the merging session verified the arithmetic and struck both products at their
  own sites — ADR-0044's `65,404` and ADR-0047's `≈268,000`, with ADR-0047's ceiling clause and
  ADR-0049's maximum clause corrected alongside. Nothing above is withdrawn; this bullet is the
  record of the finding, and the repair is at the four sites it names.
- **Decided on thin ground in one place, and it is not dressed as a derivation.** §3's refusal of
  severity is an argument about what trains an operator to stop reading, and this project has never
  had an operator. The four reasons are structural and they converge, but none is a measurement, and
  severity is the ruling here most likely to be asked for by the first person who installs. The
  reopening condition is stated: **a measured** case of an operator missing a drift-class message
  because the class was too broad — not a request for one.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **A message catalogue — one hand-authored string per message kind** | The obvious build, and it is a **curated table asserting about the product**, with no owner, no attestation and no watch — [#125](https://github.com/winniel123/verge-asm/issues/125)'s shape one layer across. It rots the way this corpus has already watched a figure rot four times: a rule stated once and applied at twenty sites has to be re-remembered at twenty sites, and #35's *not always good news* is exactly the one forgotten at the nineteenth. It also hides the class inside the prose, so the operator cannot learn the vocabulary — and class routing is the only volume control v1 ships, so an operator who cannot read the class cannot use it |
| **Four registers, hand-cut per cause, with no shared grammar** | The honest-looking middle, and it answers the ticket's question *no*. It is falsified by the corpus rather than by argument: #44's absence vocabulary already runs **four** registers under **one** cause, so cause is demonstrably not the unit the wording keys on. A per-cause register would have to level *we stopped looking* with *you stopped telling us*, which ADR-0020 built precisely to keep apart |
| **Key the vocabulary on the class rather than the cause** | Worse than per-cause, and for a measurable reason: the coverage class holds ~~**nine**~~ **ten** members (#130 · ADR-0074, merged after this pass) across **two** causes with **two** triggers under one of them, whose payloads ADR-0014 rules differ. Three registers over ~~nine~~ ten members is levelling by construction — **the rejection strengthens as the class grows**, so the count moving does not disturb it |
| **Class per rule — the three certificate rules are always clock class** | The tidiest reading and the one ADR-0004 supports on its face. It loses on the deploy: a certificate that arrives **already expired** is the world moving, and filing it under *a clock crossed* tells the operator that nothing happened but time, which is false about their estate. Worse, it is false in the direction that costs — an operator routing the clock class away to escape the ACME flap goes silent on a real deploy defect, which is [#14](https://github.com/winniel123/verge-asm/issues/14)'s false reassurance arriving through the routing layer |
| **Class per rule — the three certificate rules are always drift class** | Empties the clock class and withdraws the fourth cause, which the ticket bars and is right to bar: *a clock crossed* has a genuine referent, ADR-0004 measured that exactly three rules have it, and ADR-0038 needs it as `certificate-expiring`'s carrier. It also hands the ACME flap back to a channel that cannot silence it |
| **A severity field, or a colour scale, on the message** | The most-requested thing in any alerting product and the one this model cannot supply honestly. A clear can be an attack (#35), a widening is neither good nor bad, and the threshold behind any severity is set from an unmeasured base rate. It also performs on the message store the collapse #22 refused on `Coverage` — four routes cut apart so the operator does not learn to dismiss all of them |
| **A distinct rendering above a magnitude threshold — a banner, a bar, an approximation, a second confirmation** | Makes the message's **content** a function of how big it is rather than of what happened, so two installs performing one act send different messages — ADR-0039 §7's losing argument with magnitude for time. #74 already ruled the identical question one object down: length earns no treatment. And the guard belongs on the **act**, where ADR-0044 and ADR-0049 already put it |
| **Round or abbreviate a seven-figure count — *≈1.3 million*** | It is a count of our own objects, known to the unit, and it would be the only number in the product rendered less precisely than we hold it. The unreadability is caused by rendering a **product** with its factors hidden, and §4 fixes the cause rather than the symptom |
| **Suppress the product entirely and show only the factors** | The mirror error. Refusing to state how many timelines now exist is #14's false reassurance in the quiet direction, and it is still a content change triggered by magnitude |
| **Put the count on `Coverage`'s standing aperture statement instead of in a message** | #44 decision 10 discharged the three-densities obligation for the statement on the ground that it is **constant**. A widening is an event with an instant, so it is not constant and cannot ride a constant surface — and ADR-0044 already refused to put a port count there under #44 decision 7 |
| **Let the operator author or template the wording** | ADR-0039 §4 refused it already: operator-authored content in the alerting path is [#16](https://github.com/winniel123/verge-asm/issues/16)'s refusal one layer across, and the failure mode is a malformed sentence on the night the flagship fires |
| **Discharge the five wording pairs one at a time, as five rulings** | Five judgements where a predicate was available — ADR-0009's refused move, at the notification layer. In all five the two messages already differ in class or subject, so the grammar separates them for free, and a per-pair ruling would freeze wording that the sixth pair would then have to match by hand |
