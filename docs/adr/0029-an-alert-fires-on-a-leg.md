# ADR-0029: An alert fires on a leg, never on a state — and only the internet leg alerts

- **Status:** Accepted
- **Date:** 2026-08-13
- **Ticket:** [#58 On a one-vantage-class install there is no Exposure — which bare Reach transitions wake someone?](https://github.com/winniel123/verge-asm/issues/58)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0017](./0017-exposure-needs-both-legs.md) ruled that `Exposure` exists only where both legs
hold a value, and that a one-vantage-class install renders the surviving leg's `Reach` on its own
board. That board has real transitions on it and nothing had decided which of them, if any, are
messages.

[#58](https://github.com/winniel123/verge-asm/issues/58) framed the forcing fact as the modal
no-prober install — the one [#14](https://github.com/winniel123/verge-asm/issues/14) deliberately
built to be *not crippled* — having **an alerting surface of exactly nothing**. That is false, and
the correction is the first thing this ADR settles, because every argument for minting an internal
alert was resting on it.

On a one-vantage-class internal install:

- **Nine of the ten v1 signals are evaluable and fire**, and their transitions are drift. Every
  `certificate` rule, both redirect rules, `tls-1.0-accepted`, `plaintext-http-no-https` (which
  reads `reachability` directly, not `Exposure`), and all four DNS rules
  ([#35](https://github.com/winniel123/verge-asm/issues/35),
  [#48](https://github.com/winniel123/verge-asm/issues/48)). [ADR-0010](./0010-exposure-composes-two-reaches.md)
  refused to gate any of them on internet reach, precisely so this would be true.
- **Membership drift runs** — `Name`s, `Address`es, `Service`s and `Endpoint`s appear and withdraw.
  *Corrected by [#63](https://github.com/winniel123/verge-asm/issues/63) /
  [ADR-0031](./0031-membership-alerts-at-the-root-of-the-entering-subtree.md): true of the drift,
  false of the messages — only `Name` and `Address` membership notifies. See the amendment below.*
- **The coverage class runs in full** — all eight members.
- **The clock class runs** — certificate expiry.

What the install lacks is exactly two things, and both are about a measurement it has not made: the
flagship escalation, and `sensitive-port-reached-from-internet`, which is permanently
`not-evaluable` there. Its honest statement about that is the standing aperture statement
[#44](https://github.com/winniel123/verge-asm/issues/44) already put on `Coverage`.

The second thing the ticket surfaced is that **the internal install is not the only one-legged
install.** [#14](https://github.com/winniel123/verge-asm/issues/14) §7 explicitly supports deploying
the whole stack on a VPS, where the uniform check classifies the vantage as `internet` and the
install runs one class — the *other* one. Under ADR-0017 that install has no `Exposure` either, and
its bare leg is the leg the product exists to watch. Any rule phrased over *bare `Reach`* rather
than over *which leg* silently answers for both.

## Decision

**Alerting reads a `Reach` leg, never an `Exposure` state — and only the internet leg alerts.**

**1. `Exposure` is a board axis and a census, and is never an alert source.** No message anywhere in
the product is triggered by an `Exposure` transition.

**2. The flagship alert is the internet `Reach` going `not-reached` → `reached`, on a `Service`.**
This is not a new event. ADR-0010 already found the flagship to be *"a column move, not a cell
move"* — the internet leg moving — and ADR-0017 restated it as *"the block on one side of"* the
internet leg's boundary. Both described the flagship in terms of the leg while the alert was still
notionally attached to the composed value. This ADR moves the alert to where its own definition
already lived. It is drift, *the world moved*, in the drift class.

**3. Internal `Reach` transitions are recorded and not alerted, in both directions.** They fall on
[#17](https://github.com/winniel123/verge-asm/issues/17)'s **`withdrawn`** side: an internal port
opening or closing is the commonest intentional change on that leg, and alerting on it trains the
operator to ignore the channel — which is the cost #17 refused to pay for decommissioning.

**4. The internet leg alerts in one direction only.** `reached` → `not-reached` is recorded and not
alerted: on the alerting leg it is the shape #17 declined, since a port closing to the internet is
overwhelmingly the operator's own remediation, and the ways it could mean something worse are each
carried elsewhere — a closing custody gate, a `Vantage` becoming `unavailable`, and a service moving
address, which is a `Service` `appeared` beside it. Flap suppression on this alert belongs to the
notification patch, which owns all damping ([ADR-0007](./0007-drift-is-a-timeline-of-spans.md)).
*Corrected by [#63](https://github.com/winniel123/verge-asm/issues/63) /
[ADR-0031](./0031-membership-alerts-at-the-root-of-the-entering-subtree.md): the third carrier is
an **`Address`** `appeared`, not a `Service` `appeared`, and it is narrower than stated. See the
amendment below.*

**5. The predicate does not change when a second vantage class arrives.** It is keyed on
`Vantage class`, which is a property of the leg and not of the deployment, so it reads identically
on a one-legged and a two-legged install. Adding a vantage of a class **starts** that leg and
therefore starts its alerting behaviour; it changes no rule and silences nothing.

**6. The first vantage of a class *opens* `Exposure`; it does not `Break` it.** ADR-0017's
consequence to the contrary is withdrawn — see the amendment recorded in that ADR. `Vantage class`
remains an enumerated aperture input, so the widening is detected and yields `revealed` on the
`Reach` and `Exposure` timelines it opens, and one message in the coverage class under *we changed
how we look*, trigger 1.

**7. That message carries the census of any Derived value the widening started.**
[#42](https://github.com/winniel123/verge-asm/issues/42) fixed the aperture payload at *"a count of
timelines opened and no comparison at all"*. A count alone cannot tell the operator whether to get
out of bed on the one widening that starts the value the product is named for. A census is not a
comparison ([ADR-0008](./0008-derivation-versions-move-on-content.md)), so it is computed once at
the cause, is a description and never a `Transition`, and carries no difference set: *an internet
vantage is now configured; 1,412 `Exposure` timelines opened; 37 read `exposed`, 4 `edge-only`.*
The 37 are **not** alerted individually — they are openings, nothing moved, and that is what
ADR-0017's aperture ruling exists to secure.

## Consequences

- **The board's hero and the alert predicate become one predicate.** ADR-0017 §5 made the hero the
  block on one side of the internet leg's boundary; this ADR makes the alert the same leg move. So
  the count on the board and the count in the channel are read from one computation, which is
  [#50](https://github.com/winniel123/verge-asm/issues/50)'s rule that a number appearing on two
  screens is read from one computation or it is a defect — arriving here across a screen and a
  channel rather than across two screens.
- **The internet-only install keeps the product's headline.** Under a bare-`Reach`-is-silent rule it
  would have lost it, silently, on a posture #14 §7 documents and supports. Under a
  bare-`Reach`-is-loud rule the internal install would have gained a channel that fires on every
  deploy. Cutting on the leg is what makes both come out right, and it is ADR-0010's *a rule reads a
  leg, never a state* generalised from predicates to notification.
- **`sensitive-port-reached-from-internet` needs no sibling, and could not have one.** A
  `sensitive-port-reached-from-internal` rule was the obvious way to give the internal install a
  sharp alert, and it fails on evidence rather than on taste:
  [#21](https://github.com/winniel123/verge-asm/issues/21)'s list is attested for what is *never
  legitimately internet-facing*. Internally a Redis on 6379 is the correct configuration — that is
  the entire reason `verge-core` probes it — so the list ranks nothing on that leg. Reading it there
  is [#46](https://github.com/winniel123/verge-asm/issues/46)'s truncated-conditional shape: a claim
  used outside the context that gives its only operative term meaning.
- **The stated cost is that an internal port opening tells nobody.** It is recorded, it renders on
  the board, and it has no other carrier — a `Service` `appeared` fires when its `Address` appears,
  across the whole of `verge-core` including the closed ports, so membership does not cover it. The
  reason this is acceptable is not that the event is unimportant but that the install has not
  measured the question the product asks, and the remedy the operator wants is a prober, not a
  channel. Minting the alert would let a one-legged install feel covered while measuring nothing
  about internet exposure — #14's false-reassurance failure re-entering through the notification
  layer, which is the one layer #14's own guard does not reach.
- **Nothing is added to the notification vocabulary.** No fifth cause, no third trigger, no fourth
  class, and no ninth member of the coverage class: the first vantage of a class yields `revealed`,
  which is already member two. The only change to the vocabulary is a **payload** widening on an
  existing trigger, in §7.
- **ADR-0017's aperture protection survives the `Break` being withdrawn.** It was never the `Break`
  that carried it. ADR-0017's own third consequence states the mechanism — *"with no one-legged
  names there is no cell for an opening to move into: the `Exposure` timeline simply opens"* — and
  openings emit no `Transition` whether or not a vacuous `Break` is also written beside them.
- **Two judgements rest on unmeasured base rates**, and are flagged rather than dressed as settled.
  That internal port openings are the commonest intentional change on the internal leg, and that
  internet port closings are overwhelmingly remediation, are both assertions of the same kind #17
  made about decommissioning and neither it nor this ADR measured. If either is wrong the direction
  cut is wrong, and the correction is cheap: it moves one predicate in the notification layer and
  touches no timeline, because every one of these transitions is recorded regardless.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **A `sensitive-port-reached-from-internal` signal** — give the internal install a sharp alert through the existing signal machinery rather than through the leg | The sharpest-looking option and the one that fails hardest. #21's list is evidenced for internet-facing legitimacy; internally every port on it is a correct configuration, so the list carries no ranking there. It also adds no measurement, which makes it look free — but a rule that reads a list outside the domain it was attested for is #46's defect, not a cheap win |
| **Alert both directions on the bare leg, whichever class it is** | Gives the modal internal install a message per deploy. #17 priced exactly this and declined: the commonest intentional change is the one that teaches the operator to ignore the channel, and once ignored it is ignored for the flagship too |
| **Alertable while alone, silent once an internet vantage arrives** | The ticket's own middle option, and it loses three times. The predicate would read the install's configuration rather than the measurement, which is the hazard the `Vantage class` carve-out exists to prevent one level up. The operator who adds a prober would find their internal alerts silently ceasing, indistinguishable from the product breaking. And a rule whose trigger is a fact about which vantages the operator runs, dressed as a verdict about a `Service`, is the `internal-only` defect a fifth time |
| **Keep `Exposure` as the only alert source and rule bare `Reach` silent** | Consistent, and it drops the flagship entirely on the internet-only install #14 §7 supports — the product's headline event, lost to a deployment choice the product itself classifies automatically. It also contradicts ADR-0010 and ADR-0017, both of which already define the flagship as a leg move |
| **Suppress the aperture message's census, per #42's literal payload** | #42 wrote *count of timelines opened, no comparison* to stop the safer payload inheriting the riskier one's apparatus, and that concern is met: a census is not a comparison. Withholding it would hand the operator *1,412 timelines opened* on the night their first internet vantage lands, and make them go and look to find out whether anything is exposed |
| **Alert the newly-composed `exposed` services individually when the first prober arrives** | The escalation burst ADR-0017 exists to prevent. Nothing moved; we started looking. It would report a configuration act as 37 world events, which is what `revealed` is for |

## Amendment — [#63](https://github.com/winniel123/verge-asm/issues/63): two sentences about
membership are wrong, and the ruling they support survives

[ADR-0031](./0031-membership-alerts-at-the-root-of-the-entering-subtree.md) ruled `appeared`, and
in doing so tested the assumption this ADR made about it twice. **The load-bearing claim holds**:
membership drift is a live alerting surface on a one-vantage-class install, because `Name` and
`Address` `appeared` and `returned` all fire there. Nothing in the Decision moves. Two supporting
sentences are withdrawn as written.

**The Context bullet *"Membership drift runs — `Name`s, `Address`es, `Service`s and `Endpoint`s
appear and withdraw"* is true of the drift and false of the messages.** All four kinds drift and
all four are recorded; only `Name` and `Address` membership is ever a message, because `Service`
and `Endpoint` have keys the model composes from what it already holds and so can bring no ground
it was not already accounting for. The install's alerting surface is still four things deep.

**Decision 4's third carrier is an `Address` `appeared`, not a `Service` `appeared`, and it is
narrower.** There is no `Service` `appeared` message under ADR-0031. Where a service moves to an
address **new to the estate**, the `Address` `appeared` message fires with its census and the
carrier is intact. Where it moves to an address **already in the estate**, no membership message
fires at all: the residue is a `resolution` `Transition` recorded on the `Name`'s own timeline,
and whether that notifies is not yet decided. The direction cut on the internet leg therefore
rests on three carriers of which one is now conditional, and the other two — a closing custody
gate and a `Vantage` becoming `unavailable` — are untouched.

**This ADR's stated cost survives with a stronger reason.** *"A `Service` `appeared` fires when
its `Address` appears, across the whole of `verge-core` including the closed ports, so membership
does not cover it"* reached the right conclusion by the wrong route. Membership does not cover an
internal port opening because a `Service` `appeared` is **never** a message — and the version of
membership that would have covered it, one where only answering ports enter the estate, is
refused by ADR-0031 precisely because it would re-admit through membership the transition this
ADR ruled silent.
