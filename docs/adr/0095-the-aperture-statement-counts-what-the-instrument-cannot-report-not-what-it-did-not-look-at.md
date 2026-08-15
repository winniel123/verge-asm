# ADR-0095: The aperture statement counts what the instrument cannot report, not only what it did not look at

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#173 What does `Coverage` render for a `Reach` `Gap` that is not an aperture gap?](https://github.com/winniel123/verge-asm/issues/173)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0083](./0083-silence-decides-only-on-a-connection-oriented-transport.md) specified an honest
connectionless instrument — `answered │ refused │ unanswered` on a sixth leaf `datagram-outcome` —
and left `unanswered` as the union's first member with **no `Reach` projection**. It then named the
bill and handed it here, in terms:

> **`unanswered` would be v1's first facet value with no Derived projection**, and ADR-0010's clean
> identity — *absence of a `Reach` value ⟺ the pair was outside the recorded scope* — would no longer
> hold. That is a real cost, it lands on `Coverage`'s reading of a `Gap`, and it is the first thing
> the successor ticket owes.

[`safe-active-probing.md`](../research/safe-active-probing.md) §13.7 says the same thing shorter:
***Nobody has drawn what that renders.***

This is that ticket, and its scope is exactly one sentence on one surface.
[#44](https://github.com/winniel123/verge-asm/issues/44) put a standing **aperture statement** on
`Coverage`, one line per aperture input;
[ADR-0044](./0044-a-one-off-measurement-has-no-currency.md) specified what the **port-tier line**
says. Today that line names the tier, its cadence and its off state, and carries two figures:
**`5 of 38 sensitive pairs unread`** and **`0 of 17 rules unevaluable`**. The question is what it
says once a `Reach` leg can hold no value for a reason the aperture does not explain.

Four things are given by the ticket and are not re-derived here. ADR-0083 opens a **third route** to
a `Reach` `Gap` — *we looked and the exchange did not decide*. ADR-0010's identity between an absent
`Reach` and an out-of-scope pair **stops holding**. `unanswered` is a **value, not a `Gap`** —
ADR-0011 makes it not an absence. And the flagship already returns `not-evaluable`, through
[ADR-0010](./0010-exposure-composes-two-reaches.md) §4's existing behaviour.

## Decision

**The port-tier line's numerator counts what the shipped instrument cannot report, and *unread* is
only one of its two constituencies. The line carries two figures where it carried one, and the
second is `0` today.**

| Concern | Decision |
| --- | --- |
| Is the third route the aperture statement's business at all | **Yes**, and for one reason: its worst sub-case has **no other carrier** |
| Which line carries it | The **port-tier line**. Transport already sits inside that aperture input; no new line, no eighth input |
| What the line reads today | Unchanged in every figure. The second figure is **`0 of 38`** |
| The first figure | **`N of 38 sensitive pairs unread`** — outside the recorded scope. Definition **unchanged**, and it may **never** be widened to absorb the second |
| The second figure | **`M of 38 sensitive pairs the instrument cannot report as reached`** — inside the recorded scope, probed at cadence, and structurally incapable of producing `reached` |
| Both figures' character | **Constants of the configuration and the shipped leaves.** Set arithmetic over **our own** list; neither reads the estate, neither moves per cycle |
| A per-cycle count of undecided pairs | **Refused** — ADR-0044's own constancy premise, and #44 decision 7's estate-count refusal |
| `0 of 17 rules unevaluable` | **Does not move, and cannot.** The rule speaks; its domain is populated. **That inertness is why the pair figure is required** |
| A `Reach` leg with no value | A `Gap` where the timeline had **already opened**; **nothing at all** where it never did. ADR-0083's *third route to a `Gap`* is exact for the first and loose for the second |
| Which of the two is dangerous | The **never-opened** one. It has no span, no recorded cause, no closing edge and no message — the aperture statement is its **only** surface |
| What the `Gap` half renders as its **cause** | A **fifth register** in #44's absence vocabulary — ***we asked and heard nothing*** |
| Why a fifth register rather than one of the four | ADR-0064 keys a register on **what moved**, and here the mover is **our own instrument working correctly** — neither of its two *us* registers, and neither party's act |
| A third `Reach` value | **Still refused**, by name in ADR-0010 and again in ADR-0083. Nothing here revives it |
| Coverage-class members | **None minted.** The class stands at **ten** |
| What moves the second figure to non-zero | **Opening a UDP tier**, and nothing else in v1's reach |
| Does anything ship in v1 | **No.** UDP is off, so the second figure is `0` and the line renders as it renders today |

## Rationale

### The dangerous sub-case is not a `Gap`, and that is what forces the ruling

ADR-0083 calls this *a third route to a `Gap`*. Walk it against
[`CONTEXT.md`](../../CONTEXT.md)'s `Reach` entry and it is two cases, not one:

> the absence is a `Gap` where the timeline was already running and **nothing at all** where it never
> began — a `Gap` is a span, and an absent timeline has none

A UDP `Service` inside the recorded scope whose every probe returns `unanswered` **never opens a
`Reach` timeline at all**. It is probed daily, it writes a `reachability` observation every cadence,
and its `Reach` leg does not exist. That is not a `Gap`; a `Gap` is a span, and there is no span.

The distinction decides the whole ticket, because **the two cases have wholly different carriers**.

- A `Gap` **records its cause** — [ADR-0014](./0014-only-revealed-generalises.md) makes that the
  reason a fourth opening-family member is unnecessary. It renders on the subject, it can close, and
  its closing notifies in the coverage class. It is fully carried already and needs nothing from this
  ADR.
- A timeline that **never opened** has no span, no cause field, no adjacency, no closing edge and no
  message. `Exposure` needs both legs, so there is no `Exposure` either. The subject exists — a
  `Service` exists for every pair in the recorded scope, open or closed — and **every surface that
  speaks about it is silent about why it says nothing.**

So the losing option answers itself. *Leave it on the subject, where the `Gap` already records its
cause* is a complete answer to half the question and no answer at all to the half that matters. The
aperture statement is the surface #44 built for subjects the operator cannot otherwise account for;
that is precisely why `5 of 38 sensitive pairs unread` is on it rather than on 38 subject pages.

### `0 of 38 sensitive pairs unread` is a clean bill of health, and it is what today's line would print

This is the finding, and it is a defect in a sentence ADR-0083 wrote about its own consequence:

> **Opening the knob with a payload-free instrument moves the five pairs from `not-evaluable` because
> they are outside the recorded scope to `not-evaluable` because the exchange did not decide, and
> changes nothing the operator sees.**

The first clause is true. The second is **false**, and the site that falsifies it is the aperture
statement. The port-tier line's numerator counts sensitive pairs that are **unread**. Open a UDP
tier and those five pairs are read — probed, at cadence, with an observation written every run. The
numerator becomes **`0 of 38 sensitive pairs unread`**, on a line whose whole job is to say what the
product does not cover, while five sensitive pairs return `not-evaluable` forever and the flagship
cannot fire on any of them.

That is [#124](https://github.com/winniel123/verge-asm/issues/124)'s defect on its third outing. Its
first was a `0` standing where `5 unread` was true, because *membership of `verge-core`* was read as
*measurement*. Its second is ADR-0083's own: a `0` arriving inside a `Signal` rather than inside a
coverage line. Its third is here — a `0` arriving because *measurement* is read as *a value
produced*. Each time the arithmetic is right and the predicate has quietly shifted underneath it.

**Membership is not measurement, and measurement is not a verdict.** The three are different, and
the port-tier line has now been wrong about the boundary between the second and third exactly as it
was wrong about the boundary between the first and second.

### The figure has to be a constant, and ADR-0044 is the authority that says so

The obvious repair is a measured figure — *N sensitive pairs undecided this cycle*. It is refused,
and on this project's own ground rather than on taste.

ADR-0044 refused the one-off onboarding sweep on three limbs, and the third is this one verbatim:

> **It breaks the constancy [#44] rested on.** #44 decision 10 discharged the three-densities
> obligation for the aperture statement on the ground that *"the aperture statement is **constant**,
> so it can never escalate"*. A once-only full-range sweep makes the port-tier line non-constant.

A per-cycle undecided count makes it non-constant in exactly the same way, and worse: it would move
with the weather. It also fails #44 decision 7 — **counts of our own rules and lists, never a count
or proportion of the operator's estate** — because *how many pairs went unanswered* is a fact about
the estate wearing a fact about us as a costume, which is
[#28](https://github.com/winniel123/verge-asm/issues/28)'s refused estate-completeness score
arriving through the transport axis rather than the port axis.

What survives the constancy test is a **set-arithmetic** figure: the sensitive pairs whose transport
the shipped instrument cannot use to produce `reached`. That is `|sensitive ∩ udp ∩ in-scope|` — our
list, our tier configuration, our leaves, and no observation anywhere in it. It is as constant as
`5 of 38 unread` is, and it moves for exactly one reason: the operator turned a UDP tier on.

### `cannot report as reached` is the right predicate, and `cannot yield a Reach` is not

Two near-misses were tried and both assert more than we are entitled to.

***Cannot yield a `Reach`*** is false. A payload-free UDP probe that draws an ICMP Port Unreachable
yields `refused`, which projects to `not-reached` unchanged — ADR-0083 kept that member for exactly
this reason. So the leg can hold a value; what it cannot hold is the **positive** one.

***Undecidable*** names a property of the listener. `CONTEXT.md` bars that construction by name, and
ADR-0083 refused `filtered` and `silent` on it three paragraphs running: a negative is **named for
the exchange we made**. What we know is about our instrument, not about what is there.

So the predicate is **the instrument cannot report this pair as `reached`** — a statement about the
shipped leaves and nothing else, and the safety-relevant half. It is the half that matters because
`sensitive-port-reached-from-internet` is a **presence** rule: what an operator needs from these
five pairs is the ability to learn that one of them **is** reachable, and a payload-free datagram
instrument cannot ever tell them so.

### The `Gap` half needs a word, and ADR-0064's own axis is what supplies it

The ticket has two halves. The aperture statement answers the sub-case with no carrier; the sub-case
that **is** a `Gap` carries itself, because a `Gap` **records its cause** — and there is no cause in
the vocabulary that fits.

#44's absence vocabulary runs four registers, and
[ADR-0064](./0064-a-message-names-what-moved-and-where-nothing-moved-it-says-so.md) is the ADR that
explained why there are four rather than three or seven. Its axis is **what moved**:

> the cause is the same in all four and the **mover** is not — twice us, once the estate's authority,
> once the operator's own declaration

- ***we never looked*** — us, our aperture.
- ***we stopped looking*** — us, our failure ([ADR-0010](./0010-exposure-composes-two-reaches.md)).
- ***you stopped answering*** — the estate's authority
  ([#35](https://github.com/winniel123/verge-asm/issues/35)).
- ***you stopped telling us*** — the operator's own declaration
  ([#48](https://github.com/winniel123/verge-asm/issues/48) /
  [ADR-0020](./0020-a-conflict-needs-two-enumerable-sources.md)).

Run the third route down that axis and **nothing moved at all**. The pair is inside the aperture, so
it is not the first. Nothing of ours failed — the batch ran, the datagram went out, an observation
was written — so it is not the second. No authority declined and the operator withdrew nothing, so it
is neither of the last two. The mover is **our own instrument, working exactly as specified**, on a
transport whose silence decides nothing.

So the register is ***we asked and heard nothing***. It is named the way this project has named a
negative five times running — for **the exchange we made**, claiming nothing about what is there —
which is the construction `CONTEXT.md` requires and the one ADR-0083 used to pick `unanswered` over
`filtered`, `open|filtered` and `silent`.

**This contradicts ADR-0064 at one sentence, and vindicates it at the argument.** ADR-0064 says a
vocabulary keyed on what moved *"produces exactly these four and no fifth"*. That is **true of v1 and
its warrant is enumerative** — it is a walk over the absence causes the corpus then had, and the
corpus then had no connectionless leg. Its **axis** is what produces the fifth here, not what bars
it: applied to a new mover the axis yields a new register, which is the axis working. The count takes
a conditional and the reasoning takes nothing. *(Contradicts ADR-0064's *no fifth* clause — but worth
reopening because the clause is an enumeration over a corpus that has since acquired a fifth mover,
and ADR-0064's own criterion is what identifies it.)*

**No message class, cause or member is minted by this.** A register is the wording a `Gap`'s recorded
cause renders as; ADR-0064 established in terms that #44's registers run **under one cause**, so a
fifth register touches neither the four message causes nor the three classes. The coverage class
stands at ten.

### The rules figure cannot detect this, and a session that checks it will conclude nothing moved

`0 of 17 rules unevaluable` does **not** move, and the reason is worth writing down because it is a
trap with a fresh precedent.

A rule counts as unevaluable when it can never speak. `sensitive-port-reached-from-internet` reads a
leg on a `Service` and its domain is populated by the 131 TCP pairs, so it speaks — exactly as
ADR-0044's #124 correction already established. Opening a payload-free UDP tier adds five subjects
on which it returns `not-evaluable` per row, and the figure over **rules** is blind to that by
construction.

So the two figures on the line are not redundant and are not substitutes. **The rules figure is over
our rules; the pairs figure is over our list.** A pass that checks only the first will find `0` and
`0` and report that nothing moved, which is the shape of every desync this map has measured. This is
the third figure on this one line whose numerator has been checked less often than its denominator.

### It is the port-tier line, and there is no eighth aperture input

A transport line of its own is the tidy alternative and it is refused, on the record rather than on
taste. The aperture input has been *port **and transport** tiers* since
[ADR-0025](./0025-an-offer-is-scope-only-where-the-value-enumerates-it.md)'s table, and
[ADR-0014](./0014-only-revealed-generalises.md) enumerates it that way too. Transport is already
inside this input, and splitting it out would price a new aperture input to hold a fact the existing
one already scopes — which would move a live figure (**seven**) for a rendering convenience.

`5 of 38` and the new figure also **partition the same denominator**, which is the strongest reason
they belong on one line: every sensitive pair is read or unread, and every read one is or is not
reportable as `reached`. Two lines would make an operator reconcile one list across two places.

### The two figures must never fuse

The instinct once both exist is to add them — *5 of 38 sensitive pairs produce no reach* covers both
constituencies and is shorter. It is refused, and by ADR-0011's central argument applied one layer
up.

ADR-0011 refused the record-with-optional-fields because *a negative modelled as an absent field is
indistinguishable from we did not look*. Fusing these two figures does that to the **aperture
statement**: the operator loses the ability to tell *we are not looking at this pair* from *we are
looking at this pair every day and cannot report what we find*. Those are different states with
different remedies — the first is fixed by turning a tier on, the second is not fixed by anything
the operator can do, and the second is the one that must never be mistaken for the first.

The remedy asymmetry is the whole point. The `unread` figure is an **invitation**; the new figure is
a statement that no action available to the operator will change it. #22's split between a fault and
an invitation is the same cut, and the aperture statement owes it here for the same reason.

### What this does to ADR-0083's ruling: it strengthens it

Nothing in ADR-0083's decision moves. UDP stays off; the union stays at three members; the leaf
count stays five; `datagram-outcome` stays specified and unshipped.

What moves is its **pricing of the payload-free knob**, and it moves in the direction ADR-0083 was
already arguing. Its case was *zero net new firings* — opening payload-free buys nothing. This ADR
makes it worse than nothing: opening payload-free **degrades the aperture statement**, taking the
`unread` numerator to `0` and requiring the second figure to exist at all. A knob whose first effect
is to make the honesty surface print a clean bill is not a neutral knob.

So the honest v1 statement of the UDP position gains a third form, after *it is expensive* and
*the instrument that would make it worth turning on is the instrument this map already deferred*:

> **Turning UDP on without payloads does not cost nothing. It costs the sentence that says we are
> not looking.**

## Consequences

- **[ADR-0044](./0044-a-one-off-measurement-has-no-currency.md)'s port-tier line specification gains
  a second figure**, valued `0` on every shipped configuration. Its *"counts **our** lists and rules;
  carries **no** count of what is unmeasured"* rider is confirmed rather than amended — the new
  figure is a count over our list and carries no count of the estate.
- **[ADR-0010](./0010-exposure-composes-two-reaches.md) is annotated at three sentences and struck at
  none**, because nothing it says is false today. §1's *"the absence is already a `Gap`"*, §4's
  *"`not-evaluable` where it is a `Gap`"* and §5's *"the two ways a leg can be absent"* each read
  alone in the present tense as though the aperture explains every absent `Reach`. That is
  [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)'s test met
  by a conditional rather than by a supersession, so the repair is a conditional: **the identity
  holds while every probed transport's outcome union projects totally onto `Reach`. TCP's does. UDP's
  does not.**
- **ADR-0010's *`fired` → `not-evaluable` … worded as we stopped looking* is conditionally
  annotated.** On a UDP leg that wording is **false** — we looked, at cadence, and the exchange
  decided nothing. The edge is unchanged and stays **coverage class, member 5**; only its copy is
  falsified, and only the day UDP ships. The copy itself belongs to
  [#120](https://github.com/winniel123/verge-asm/issues/120) / ADR-0064's grammar and is not written
  here.
- **[ADR-0064](./0064-a-message-names-what-moved-and-where-nothing-moved-it-says-so.md)'s *"exactly
  these four and no fifth"* is conditionally struck at the sentence, and its axis is confirmed.** The
  clause is an enumeration over a corpus with no connectionless leg; the **reasoning** — four
  registers, one cause, keyed on **what moved** — is what identifies the fifth mover and is untouched.
  The count is `4` on every shipped configuration. [ADR-0024](./0024-a-rules-domain-is-the-extension-of-its-name.md)'s
  `Gap` rendering row takes the fifth register beside it, and is noted there as a **rendering table
  rather than the vocabulary's enumeration** — it omits #48's *you stopped telling us*, which this
  pass found and did not repair.
- **No coverage-class member is minted and the class stands at ten.** Walked rather than quoted:
  opening a UDP tier is an aperture widening and fires member 2 (`revealed`); a `Reach` `Gap`
  opening on a previously-decided pair is member 5's edge; its closing is member 7; a `Reach`
  timeline that never opens fires **nothing**, which is the defect this ADR routes to the aperture
  statement precisely because no message can carry it.
- **[ADR-0083](./0083-silence-decides-only-on-a-connection-oriented-transport.md)'s *"changes
  nothing the operator sees"* is struck at the sentence.** It is contradicted rather than refined,
  and the contradiction strengthens its conclusion. Marked at both of its sites — the block quote in
  the rationale and the *Ship the UDP leg payload-free* row in its rejected alternatives.
- **[`CONTEXT.md`](../../CONTEXT.md) changes in two entries.** `Gap`'s cause enumeration gains the
  conditional route it lacked, `Reach` having already been given its clause by ADR-0083 and `Gap` not
  — an asymmetry inside one glossary where a fact and its representation live one screen apart.
  `Exposure`'s *"where a configured leg went silent the `Exposure` timeline holds a `Gap` under **we
  stopped looking**"* takes the same conditional as ADR-0010's consequence, and for the same reason:
  read alone it specifies the copy.
- **[`safe-active-probing.md`](../research/safe-active-probing.md) §13.7's *"Nobody has drawn what
  that renders"* is discharged**, and §2.4's port-tier line — the site that **restates** ADR-0044's
  specification — takes the second figure with its zero value.
- **The port-tier line is specified by two ADRs and a research note and is drawn by no prototype.**
  [`prototypes/coverage/`](https://github.com/winniel123/verge-asm/tree/main/prototypes/coverage)
  contains no aperture statement at all — it predates #44 — so **the surface this ADR rules on has
  never been drawn**. Carried as a successor rather than repaired here:
  [ADR-0075](./0075-a-prototype-is-a-dated-record-of-a-reading-never-of-a-rule.md) owes a drawing
  nothing for a figure a ruling moves, and this ADR moves no drawn state because there is none.
- **The observation corpus is untouched, and this is worth stating because it looks like it should
  not be.** `unanswered` is a value, so a UDP pair writes an observation every cadence exactly like
  any other. The Derived leg is what holds nothing. So this ADR reaches
  [ADR-0041](./0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md)'s retention
  arithmetic not at all: no corpus grows, no currency bound moves, and a `Reach` timeline that never
  opens is the cheapest object in the model.
- **Nothing on the map's composed-state line moves.** `verge-core` stays 136 (131 TCP, 5 UDP); the
  aperture statement stays `5 of 38 sensitive pairs unread`; the rule set stays 17; the aperture
  inputs stay seven; the leaves stay five; the facets stay six; the coverage class stays ten. This
  ADR adds a figure whose value is `0` and a conditional to three sentences.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Say nothing on the port-tier line — a `Gap` records its own cause, so render the third route at the subject and nowhere else** *(the option that lost)* | The strongest available answer and it is complete for half the case. ADR-0014 genuinely does make the `Gap` self-accounting, and the aperture statement genuinely is barred from estate counts. It fails on the sub-case that is not a `Gap`: an in-scope pair whose leg **never opens** has no span, no cause field, no closing edge and no message, so *render it at the subject* renders it nowhere. And it leaves the port-tier line printing `0 of 38 sensitive pairs unread` — a clean bill on the one surface built to prevent one |
| A per-cycle count of pairs the exchange did not decide | Non-constant, which falsifies #44 decision 10's premise on ADR-0044's own argument; and it counts the estate, which #44 decision 7 refuses. It would also flap with our own probe rate, ICMP rate limiting having put that rate inside the value (ADR-0083 §13.5) |
| Widen `unread` to mean *produced no `Reach`*, one figure instead of two | Fuses *we are not looking* with *we are looking and cannot report* — ADR-0011's absent-field argument applied to the aperture statement, and it erases the remedy asymmetry that is the operator's whole use for the line |
| A transport line of its own on the aperture statement | The aperture input has been *port **and** transport tiers* since ADR-0025; a new line prices an eighth input to hold a fact the seventh already scopes, and splits one denominator across two places |
| Put it on the rules figure — `1 of 17 rules unevaluable` | False. The rule speaks; its domain is populated by 131 TCP pairs. Reporting a rule as unevaluable because five of its subjects are would make the figure a per-subject count wearing a per-rule name |
| Give `Reach` a third value so the absence is a value | Refused by name in ADR-0010, deleted outright as `unknown` by #40 / ADR-0013, and refused again in ADR-0083. Three refusals; nothing here is new evidence |
| Call the second figure *undecidable*, or *silent* | Names a property of the listener rather than of our exchange — the construction `CONTEXT.md` bars and ADR-0083 refused three times in one section |
| Call it *cannot yield a `Reach`* | False: `refused` still projects to `not-reached`. What the instrument cannot produce is the **positive** value, which is also the only half that matters for a presence rule |
| Render the `Gap` half under *we stopped looking*, reusing register 2 | False in the plainest way — we looked, at cadence, and wrote an observation. It is also the register that routes #22's fault treatment, so it would report the instrument working as the instrument broken |
| Render the `Gap` half under *you stopped answering*, reusing register 3 | Names the listener, which `CONTEXT.md` bars and ADR-0083 refused three times in one section. #35's register is earned by a **delegation walk** that attributes the failure structurally; a datagram buys no attribution at all |
| Accept ADR-0064's *no fifth* and leave the `Gap` half unworded | The clause is an enumeration over a corpus with no connectionless leg, and ADR-0064's **axis** is what identifies the fifth mover. Leaving it unworded means a `Gap` whose recorded cause has no rendering, which is the one thing ADR-0014 built the cause field to prevent |
| Fix it by keeping the UDP knob shut, so the figure never becomes non-zero | That is the status quo and it is also the recommendation — but it is not an answer to the ticket. A surface specification that is correct only while a knob stays shut is the defect ADR-0058 exists to catch, and the knob's price is what #174 is being asked to compute |
| Redraw the `Coverage` prototype with the line on it | Out of bounds for this pass — #144 is dateline-marking every prototype in this wave and four more are drawn in the next. Carried as a successor |
