# ADR-0091: The routing unit is the class — and the cause is refused as a routing key, on grounds that do not expire

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#158 Is the routing unit the class or the cause?](https://github.com/winniel123/verge-asm/issues/158)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0039](./0039-a-channel-carries-the-message-never-the-estate-and-a-delivery-is-an-operational-record.md)
§4 routes a `Channel` by **class** *"and by nothing finer"*, and gave one reason: a per-rule or
per-subject filter is *"an operator-authored predicate over a versioned rule set"* that *"fails
silently the first time a rule is renamed"*.

[ADR-0064](./0064-a-message-names-what-moved-and-where-nothing-moved-it-says-so.md) then did two
things that look, from ADR-0039's side, like the cause becoming visible in its own right. It made the
class a property of the **firing** rather than of the rule, so the three certificate-lifetime rules
land in two different classes on different nights. And it wrote the coverage class's membership down
at **ten** across **two** of the four causes.

The ticket's premise follows: ADR-0039's stated reason does not reach the **cause**, because a cause
is not an operator-authored predicate — it is the model's own closed union of four, carried on every
message as a field, and an operator cannot rename it. The want it is asked to serve is *tell me when
the world moved, and put everything about our own looking somewhere quieter*. The priced cost is one
enum widening on `Channel`.

**What is not reopened here**: the four causes, the three classes, the class-per-firing rule, the
message set, the wording grammar, and ADR-0039 §6's refusal of coalescing and flap suppression. This
ADR decides one thing — which field a `Channel` subscribes on — and it decides it in the direction
that adds nothing.

### The two axes, laid beside each other

The three classes are the four causes with **two of them merged**, and nothing else:

| Cause | Class |
| --- | --- |
| the world moved | `drift` |
| we stopped looking | `coverage` |
| we changed how we look | `coverage` |
| a clock crossed | `clock` |

So *route on the cause* means, exactly and only, **split the coverage class in two**. Every other
property the two axes have is shared: both are the model's own closed unions, both are unrenameable,
both are carried on the message, and — this is the one worth stating — **both are read per firing.**

## Decision

**The routing unit is the `Channel`'s subset of the three classes, and the cause is refused as a
routing key. The refusal is not a deferral: the cause is the wrong axis rather than a right axis
arriving too early. Where the coverage class must one day be split, the corpus already supplies the
cut and it is the *mover*, never the cause.**

### 1. One of the ticket's two grounds evaporates on inspection, and the question halves

*Under ADR-0064 the clock class is assigned per firing, so the three certificate-lifetime rules land
in two different classes on different nights* is true, and it is **not an instance of the class being
coarser than the cause.** The cause moves in lockstep across that same seam: a `certificate-expired`
firing whose `certificate` span moved in the fold has the cause *the world moved*, and one whose span
did not has the cause *a clock crossed*. Drift and clock are those two causes, one each.

What ADR-0064 §2 made finer is **the class relative to the rule** — the thing ADR-0039 §4 never
claimed and ADR-0026 §5 had asserted wrongly. It is an argument that class routing is *more*
expressive than anybody had written down, not that it is coarse. Read as the ticket reads it, it
would equally be an argument that the **cause** is too coarse, since a per-rule cause assignment
fails on the identical deploy case.

So the second ground is withdrawn and the question reduces to the first: **may an operator route *we
stopped looking* separately from *we changed how we look*?** That is the whole of what the cause
buys, and the rest of this ADR is about that one cut.

### 2. The want the ticket states is already expressible, and the cut it asks for is a different want

*Tell me when the world moved, and put everything about our own looking somewhere quieter* names two
buckets. *Everything about our own looking* is causes 2 and 3 together — which is the coverage class,
in its extension, exactly. The operator writes it today: `drift` on the loud channel, `coverage` and
`clock` on the quiet one, zero latency, zero threshold, nothing new on `Channel`.

The ticket's motivating sentence is therefore a **cut on class**, and it is shipped. What routing on
cause additionally buys is the split *inside* the quiet bucket — and no want for that is stated,
here or anywhere in the corpus. **That is the first thing wrong with the ticket's cost line**: it
prices a feature nobody has asked for, against a want that is already served.

### 3. ADR-0064 measured that the cause is not an operator-facing unit, and the measurement runs against this ticket

The ticket cites ADR-0064 as making the cause visible. ADR-0064's actual finding about the cause is
the opposite one, and it is a measurement over the corpus rather than an argument.

[#44](https://github.com/winniel123/verge-asm/issues/44)'s absence vocabulary runs **four registers
under one cause** — *we never looked* · *we stopped looking* · *you stopped answering* · *you stopped
telling us* — and ADR-0064 §1 states why there are four: *"the cause is the same in all four and the
**mover** is not — twice us, once the estate's authority, once the operator's own declaration."* Its
rejected-alternatives table draws the conclusion in terms: **"cause is demonstrably not the unit the
wording keys on"**, and *"a per-cause register would have to level* we stopped looking *with* you
stopped telling us*, which*
[ADR-0020](./0020-a-conflict-needs-two-enumerable-sources.md) *built precisely to keep apart."*

Routing is a blunter instrument than wording — it does not merely level two sentences, it puts them
on the same wire or takes them off it together. **An axis already measured too coarse to key the
sentence cannot be the right one to key the channel.**

The concrete cost is nameable. Under cause routing, [#48](https://github.com/winniel123/verge-asm/issues/48)'s
*you stopped telling us — the zone for `example.com` has aged out of its window* rides the same
subscription as a prober outage and a lapsed `custody extension`. It is the operator's own act, it is
fixable in an afternoon, and ADR-0020 added it as *"the first the operator causes"* specifically so
it would not be levelled with our failures. Cause routing levels it, at the layer that decides
whether anybody is told at all.

### 4. The price is not one enum widening: the corpus holds no cause assignment for the coverage class, and one member is on record as having neither

Walked against the corpus rather than quoted, the coverage class's **ten** members carry a stated
cause for **six**:

| Cause | Members with the cause stated | Site |
| --- | --- | --- |
| *we changed how we look* | **2** (`revealed`), **4** (the `Derivation` vector move), **10** (the `Seed` narrowing at the scope) | ADR-0014's #130 amendment; [ADR-0074](./0074-an-aperture-narrowing-that-takes-its-carrier-with-it-fires-at-the-scope.md) |
| *we stopped looking* | **3**, **5**, **8** | ADR-0074: *"members 3, 5 and 8 are all messages fired because we stopped looking at something"* |
| **neither, stated** | **7** — a `Gap` closing | [ADR-0014](./0014-only-revealed-generalises.md) Consequences: *"we can see again, and things differ* **is neither drift, nor a clear, nor** we stopped looking*"* — and it is not *we changed how we look* either, because our looking did not change; we resumed |
| **none stated** | **9**, and the two ADR-0008 put in the class | Member 9 is [ADR-0013](./0013-custody-is-control-and-extends-by-declaration.md)'s *"first coverage-class member caused by neither our act nor the operator's"*. [ADR-0008](./0008-derivation-versions-move-on-content.md) puts a `Vantage` going `unavailable` and `Coverage` crossing a threshold in the class as *"the world or our own infrastructure failing"* — a third agency again |

So making the cause routable is not one enum widening. It is a **ten-row classification the project
would have to author and then own forever**, at least one row of which the corpus has already ruled
belongs to neither value, with a third agency loose in the class that the two-value cut has no place
for. That is a curated classification asserting about our own product, with no owner, no attestation
and no watch — [#125](https://github.com/winniel123/verge-asm/issues/125)'s shape one layer across —
and an obligation with no failing test has no owner
([ADR-0085](./0085-an-obligation-with-no-failing-test-has-no-owner-and-a-boundary-needs-a-row-on-each-side.md)).

**And it fails silently.** The eleventh coverage member gets minted by a ticket that is thinking
about something else, nobody assigns it a cause, and it lands wherever the default puts it — on a
channel the operator believed they had configured, or off one they believed they had not. The ticket
is right that ADR-0039 §4's *stated* failure mode — a renamed rule — does not reach the cause. It is
wrong that no failure mode does. **The same failure arrives through a different door, and the door is
one the class does not have**, because a class is assigned by ADR-0064 §2's fold test on every firing
and cannot be forgotten.

### 5. The reason to want it is volume, the volume is unmeasured, and §6's condition is the one to trip

Strip §2 and §3 away and one honest motivation survives: *the widening announcements are noise and I
want to keep the `Gap` messages.* It is unmeasured, and the model bounds it by construction. A
message fires **at the cause with a census**, so an aperture widening is **one** message per declared
act however many timelines it opens — that is precisely what ADR-0064 §4's seven-figure sentence
exists to render. A `Seed` narrowing is one message at the scope. A re-baseline is one message per
alerting derivation per release. Nothing in the class has been measured to arrive at a rate a
subscription cannot absorb.

**The counter deserves stating rather than hiding, which is why this ground is third and not first.**
A finer routing axis is *not* coalescing: it delays nothing, holds nothing, reads no clock of its own
and changes no predicate, so ADR-0039 §6's *every proposed suppression rests on an unmeasured base
rate* does not condemn it the way it condemned a digest window. What §6 does condemn is **shipping it
now**. ADR-0039 §4 admitted class routing on the ground that it *"earns its place rather than being
one dial too many"*, and a volume control admitted before anybody has been drowned is the dial that
argument refuses. §6 already states the condition in the words this ticket would have to satisfy: **a
measured volume in the class routing cannot rescue.** It has not tripped.

### 6. If the coverage class is ever split, the cut is the mover — and it is already computed

This is the part worth carrying forward, because it converts a refusal into a bearing.

ADR-0064 §1 built a **four-row total function** from the fold to what the sentence is about, and it
is falsifiable in terms — *point at a message whose subject is not the thing the fold says moved*:

| What the fold says moved | The **mover** |
| --- | --- |
| an object in the estate | that object |
| our aperture, or a rule of ours | **us** |
| the operator's own declared input | **the operator** |
| nothing — a threshold was crossed | **nothing** |

That axis is also a four-way refinement of the three classes, splitting coverage in two — but it
splits it **{us} against {the operator}** rather than {changed how we look} against {stopped
looking}. Five properties make it the better cut, and every one of them is a property the cause
lacks:

1. **It is already assigned on every message, by construction**, because ADR-0064 §1's rule is total
   and is read from the fold at the cause. It needs no classification exercise and cannot be
   forgotten on member eleven. Where the cause is unassigned the mover is not: ADR-0064 §6 assigns
   member 9's mover explicitly — an `Address` entering beneath a live `custody extension` fires *the
   gate your declaration holds open has moved*, **coverage, subject us**.
2. **It is the axis the operator can act on.** *You stopped telling us* is a thing they fix; *we
   stopped looking* is a thing we fix. That is the difference a second channel is for.
3. **It keeps apart exactly what ADR-0020 built to be kept apart**, where the cause levels it (§3).
4. **It is falsifiable**, which no cause assignment over the coverage class would be.
5. **It costs the same enum**, four values on `Channel`, and no new field on `Message`.

It is **not adopted here**, and the reason is §5 alone: it is still a second volume control with no
measured volume behind it, and it is a wider change than a refinement of the existing subset. It is
named so that ADR-0039 §6's reopening condition has a candidate attached to it rather than an open
invitation. **If §6 trips, the answer is the mover. The cause is closed.**

### 7. What binds

- **A `Channel` subscribes on a subset of the three classes and on nothing else.** Unchanged, and now
  ruled rather than inherited.
- **The cause is a field on the `Message` and on the POST body, and it is not a routing key.** It is
  read by the operator, never by the router. This is stated so that its presence in the payload
  (`notification-channels.md` §3.1) is not mistaken for a routing axis in waiting.
- **No fifth cause, no fourth class, no new coverage-class member, no severity, no per-rule or
  per-subject filter, and no operator-authored predicate.** Nothing here mints anything.
- **ADR-0039 §6's reopening condition is unchanged in its trigger and gains a named candidate**: a
  measured volume in the class routing cannot rescue, and the axis to reach for is the **mover**.

## Consequences

- **[`CONTEXT.md`](../../CONTEXT.md)'s `Channel` entry is amended at its sentence.** *"Routing is by
  class and nothing finer, a per-rule or per-subject filter being an operator-authored predicate over
  a versioned rule set"* gives a reason that reaches the per-rule filter and **not the cause**, so
  read alone and in the present tense it leaves the cause open — which is the reading this ticket
  arrived on. Per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)
  the reason is widened at the site that specifies it, with the cause named and refused. No term is
  added and no term moves.
- **[`docs/spec/notification-channels.md`](../spec/notification-channels.md) §5's `Cause` row loses a
  reason that has expired.** It read *"the four causes are a wording distinction and belong to
  #120"*. #120 has ruled, so under
  [ADR-0057](./0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md) — *a withdrawal that
  supplies no replacement does not hold* — the row is given this ADR's reason rather than a pointer
  to a closed ticket. §9 carries the reasoning.
- **Two stale sentences in `notification-channels.md` §6 are repaired at the site that specifies
  them**, because this ruling depends on both. *"The ACME flap has no remedy in v1 … Flagged, not
  ruled here"* was withdrawn by ADR-0064 §2 at ADR-0039's site and never at this one; read alone it
  would have a competent session re-open #120. And *"for all sixteen rules"* is the retired rule-set
  figure — ADR-0026 §5's own site already reads **seventeen**.
- **One cross-document figure desync is named and repaired.** ADR-0064's Context paragraph says the
  third cause carries *"two triggers whose payloads differ"*, and its own amendment box says
  *"nothing else in this paragraph moves"*. ADR-0074 made it **three** triggers with **three**
  payloads, at ADR-0014's own site, in the same parallel batch — so the amendment box's *nothing
  else* was true of the clock class and the census producers, which it re-checked, and false of the
  trigger count, which it did not. It moves no conclusion in either ADR and it moves none here: three
  triggers under one cause makes the coverage class **coarser** than the ticket priced it, which
  strengthens the ticket's own best argument and loses to §3 and §4 anyway.
- **The interface gains nothing.** No fourth checkbox, no second axis, no cause filter on the message
  list. ADR-0064's *"no filter but class"* is confirmed rather than moved.
- **Nothing in the composed state moves.** Coverage stands at ten, the clock class at three, the
  census payload at five producers, the causes at four and the classes at three.
- **Decided on thin ground in one place, and it is not dressed as a derivation.** §5's claim that the
  coverage class's volume is absorbable rests on the census-at-the-cause structure and on **no
  measurement of a running install** — this project has never had one. It is the same thin ground
  ADR-0039 §6 and ADR-0064 both disclosed, and it is disclosed here rather than inherited quietly.
  What is **not** thin is §3 and §4: those are readings of the corpus, and either alone refuses the
  cause.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Route on the cause — a four-value enum on `Channel` replacing the three-class subset** | The ticket's own proposal and the option that loses. Its price is wrong: four of the coverage class's ten members carry no cause on record and ADR-0014 rules **member 7 has neither**, so the enum widening is the visible tip of a ten-row classification with no owner and no failing test, which fails silently on member eleven — ADR-0039 §4's failure mode through a different door (§4). And the axis was measured coarse by the very ADR the ticket cites: #44's four registers sit under **one** cause, so cause routing levels #48's *you stopped telling us* with a prober outage, which ADR-0020 exists to prevent (§3) |
| **Route on the cause, but only as a refinement *inside* the coverage class — keep three classes and add one sub-toggle** | The cheapest shape of the same idea, and it inherits §4 whole: the sub-toggle still needs all ten members assigned. It adds a defect of its own — the operator's dial no longer matches the vocabulary the message renders, which is exactly ADR-0064 §3's *"an operator who cannot read the class cannot use it"* inverted |
| **Route on the mover — estate · us · the operator · nothing** | The **better** finer axis, and the one to reach for if ADR-0039 §6 ever trips. It loses **here** on §5 alone: no measured volume, and ADR-0039 §4 admitted class routing precisely on the ground that it is not one dial too many. Recorded as a named candidate rather than a rejection on the merits — the merits are in its favour |
| **Do nothing and record nothing — leave ADR-0039 §4 as it stands** | The zero-cost answer and it is not available. ADR-0039 §4's stated reason genuinely does not reach the cause, which is how this ticket arrived, and a sentence whose reason under-reaches regenerates the question at the next session that reads it — ADR-0057's *a withdrawal that supplies no replacement does not hold*, in the register of a justification rather than a strike |
| **Widen the reason at ADR-0039 §4 alone, and skip the glossary** | Fails ADR-0058 on the sentence. `CONTEXT.md`'s `Channel` entry carries the same under-reaching reason in the present tense, and the glossary is what every session reads first |
| **Mint a fifth cause for *our own looking changed* so the split falls out of the causes** | Barred by the ticket and right to be barred. A fifth cause needs a reason and not a slot (ADR-0026, ADR-0039 §5), and this one is a routing convenience wearing an ontology's clothes — it would also break ADR-0064 §1's total mapping from fold to sentence, which is the only falsifiable thing in the message layer |
| **A per-coverage-member subscription — ten checkboxes** | The reductio, and worth stating because it is where cause routing's logic ends. It is an operator-authored predicate over a versioned member list, which is ADR-0039 §4 verbatim, and the member list moves — it has moved twice in two days, 8 → 9 → 10 |
| **Defer: rule nothing, re-file behind a volume measurement** | Leaves the question live and the reason under-reaching, so it is re-derived by the next session, which is the shape ADR-0057 measured. §5 already **is** the deferral, expressed as a condition with a named candidate rather than as an open ticket |
