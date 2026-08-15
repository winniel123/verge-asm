# ADR-0047: An address scope is its own enumeration, a name scope is not — and the range cap is what makes that affordable

- **Status:** Accepted
- **Date:** 2026-08-14
- **Ticket:** [#81 Does an address-scope `Seed` enumerate, or does it only bound?](https://github.com/winniel123/verge-asm/issues/81)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)
- **Amends:** [ADR-0002](./0002-ownership-gates-probing.md), [ADR-0013](./0013-custody-is-control-and-extends-by-declaration.md)

## Context

The repo answers this both ways, in documents that are each load-bearing, and nothing had needed
to distinguish them.

**It only bounds.** [`CONTEXT.md`](../../CONTEXT.md)'s `Seed` declares *"a boundary, **not a
starting point**"*. [ADR-0013](./0013-custody-is-control-and-extends-by-declaration.md) makes *is
this address covered by a `Seed`?* a **lookup**, total and free.
[#48](https://github.com/winniel123/verge-asm/issues/48) says a `Seed` *"produces no observations,
so it is not a source … it declares where the estate ends and asserts nothing about what is
inside."*

**It enumerates.** [`safe-active-probing.md`](../research/safe-active-probing.md) §3.4 adopts the
`-Pn` posture — *"attempt the requested scanning functions against **every** target IP address
specified"* — on the express ground that *"targets are seeded by the operator, not discovered by
sweeping. The operator types in domains and **IP ranges** they own; the tool does not need to
establish liveness before probing, it needs to record 'no ports responded' as a legitimate,
diffable observation."* §9 then ships a **`/22` per-target range size cap** whose stated purpose is
stopping *"a typo'd `/8` from becoming a multi-day scan"*. A cap on range size is unexplainable if
a range never walks.

[#80](https://github.com/winniel123/verge-asm/issues/80) forced it. Every load figure in
[ADR-0044](./0044-a-one-off-measurement-has-no-currency.md) is stated **per address**, with the
estate limb given twice, because the two readings differ by **three orders of magnitude on one
declaration**: a full-range pass is minutes under *bounds* and 3.9–11.6 days under *enumerates* at
§9's own shipped cap. ADR-0044 surfaced the seam rather than settling it, and this is the last
open blocker on [#12](https://github.com/winniel123/verge-asm/issues/12).

## Decision

**An address-scope `Seed` enumerates. A name-scope `Seed` does not. The difference is not a
preference about probing: it is that an address scope **is** its own complete extension and a name
scope is not.**

| Concern | Decision |
| --- | --- |
| Does a declared CIDR produce probe targets? | **Yes — every address in it, walked every cadence, whether or not anything has ever answered there** |
| Which addresses, exactly | **All ~~2^(32−n)~~ 2^(width−n) of them** — the `32` is **WITHDRAWN** by the [#85](https://github.com/winniel123/verge-asm/issues/85) amendment below, which names this sentence as *"the only accidentally-IPv4 thing in this document"*; the enumeration is family-agnostic. Network and broadcast addresses are not exempted, because that would mean inferring the operator's subnetting, which we cannot measure and must not guess. A reserved address answers nothing and reads `not-reached`, honestly |
| What puts an address in the estate before anything is observed | The `Seed`, by the **second limb of `Address`'s own membership rule**. Its `Citation` hops to the `Seed` — a terminus `Citation` already contemplates |
| Does a name scope enumerate? | **No.** Its address extension is **measured** — resolutions, and only under a `custody extension`. No name is ever generated from a declaration (§10's refused wordlist) |
| How deep the enumeration goes | **Exactly one hop**, `Seed` → `Address`. `Service` is *computed* from an `Address` and the recorded port scope ([#63](https://github.com/winniel123/verge-asm/issues/63)); everything below `Service` needs an **open port**. The enumeration stops on its own |
| A declared address nothing ever answers on | An `Address` subject whose `Service`s hold `Reach` = `not-reached` — a **measured value**. Not a `Gap`, and not *nothing at all* |
| §3.4's `-Pn` paragraph | **Correct, and not corrected.** It is the clearest statement of this decision in the repository |
| `Seed`'s *"a boundary, not a starting point"* | **Kept, and scoped to what it always refused** — a name-scope starting point, and a starting gun. It never spoke to an address scope's own arithmetic |
| §9's ~~`/22`~~ **1,024-address** per-target range size cap | A **validation on the declaration** — §3.4's *"cannot be entered"* — operator-configurable, applied **per scope** and never to a sum. The cap's unit is **addresses**, not prefix lengths, per the [#85](https://github.com/winniel123/verge-asm/issues/85) amendment below; `/22` is its IPv4 spelling and `/118` its IPv6 one |
| An address scope wider than the cap | **Not declarable** at the shipped default. Custody at that scale belongs to a name scope's `custody extension`, which is ADR-0013's own stated preference |
| Declaring or widening an address scope | An **aperture widening**: `revealed`, **one** coverage-class message at the scope carrying a count of timelines opened. **Never** 1,024 `appeared` messages |
| Narrowing one (an exclusion, or a smaller CIDR) | The addresses **leave**, taking their timelines, unless a current resolution still cites them — **no `Gap`**. Where one does cite them, the gate closes and currency opens the `Gap` as it already did |
| What `Coverage` may say about a `/22` seed | **All of it** — *1,024 declared, 1,024 measured*. Address counts are arithmetic over a CIDR, which [#50](https://github.com/winniel123/verge-asm/issues/50) already ruled is not a coverage figure |
| The daily load at the cap | **12–36 minutes per day** at §6.3's 200 pkt/s global ceiling. The default fits inside its own cadence |
| ADR-0044's two estate limbs | **Collapse to one.** 3.9–11.6 days is now the shipped worst case at the cap, not a hypothesis |

## Rationale

### The bounding reading silences the flagship message on the one estate the instrument exists for

This is the decisive argument and it is not about load.

An operator declares `198.51.100.0/24` because they want to know when a machine appears in that
space. Six months later one does, listening on 6379/tcp.

**Under enumeration**, that address has been a subject since the declaration and its `Service`s have
held `Reach` = `not-reached` on the internet leg every day since. The listener coming up is
`not-reached` → `reached` — *"the product's flagship message, fired whether or not the other leg
exists"* — and `sensitive-port-reached-from-internet` rides it.

**Under bounding**, in either of its forms, the flagship cannot fire:

- If unresolved addresses are never probed, nothing looks at `198.51.100.77` at all. The machine is
  invisible for as long as no name of the operator's resolves to it, which on unmanaged space is
  forever. The declaration bought nothing.
- If the range is walked but only *answering* addresses become subjects — the strongest middle
  position, and the one worth taking seriously — the `Address` **enters** when the listener comes
  up. [ADR-0031](./0031-membership-alerts-at-the-root-of-the-entering-subtree.md) and
  [#63](https://github.com/winniel123/verge-asm/issues/63) then bind: a newly-entered subject *"has
  no transitions at all: its `Reach` leg opens **at** `reached` rather than moving to it … so the
  flagship predicate, the projection and all ten rules are transition-shaped and match none of
  them."* The news arrives as a **census line on a membership message** and by nothing else.

So the middle position converts *the product's loudest event* into *a line in a census*, and it does
so **only** on address-scope estates — precisely the estates that declared a range in order to watch
it. It also breaks `CONTEXT.md`'s own rule in terms: *"a port opening is a `Reach` move and **never**
a membership event."* Under it, the first port opening on dark declared space is necessarily a
membership event.

That is not a cost to weigh. It is the instrument failing at its only job, and it is why the model
already says a `Service` exists *"for every `(port, transport)` in the recorded scope, **open or
closed**, which is what gives `unreachable` a subject to be a verdict about."* The dark rows are
not overhead. They are the apparatus.

§3.4 says the same thing one layer down and had said it since #4: the tool *"needs to record 'no
ports responded' as a legitimate, diffable observation."* You cannot record that about an address
that is not a subject.

### Two shipped prototypes already drew the enumerating reading

Before the argument, the drawings. Two prototypes on `main` render an address-scope `Seed` and both
render it enumerated — which means the reading was never merely available, it was already built
against.

[`prototypes/coverage/index.html`](../../prototypes/coverage/index.html) carries
`{ scope:"203.0.113.0/24", vantage:"internet-01", n:254, of:254 }`, rendered as *"254 of 254
**subject-vantage pairs**"*, with a failure row reading `n:0, of:254`. **The `/24` already has a
coverage denominator, and it is the size of the CIDR.** The screen's own checklist says declaring a
scope *"gives coverage a denominator. Until one exists there is no figure to report — not 0%."*
Under the bounding reading that denominator has no source: `of:` would have to be *however many
addresses happened to resolve*, which is not a number the screen could stand behind.

[`prototypes/landing-view/index.html`](../../prototypes/landing-view/index.html) is more direct
still. It gives the `/24` seed `names: 0` and then lists two subjects under it — `203.0.113.41` on
6379/tcp and `203.0.113.42` on 5900/tcp — each with a `Name` of `—`, and it draws the provenance
line as `citation 203.0.113.0/24 → seed (declared address scope)`. Nameless addresses under a seed
with no names, citing the seed directly: **nothing but enumeration produces those two rows**, and
the `Citation`-hops-to-`Seed` mechanism this ADR relies on is already drawn.

Prototypes are not authority — but two independent sessions reaching for the same shape, and
neither noticing it was contested, is evidence about which reading the model actually supports.

### Six load-bearing structures already presuppose enumeration, and one of them is a safety argument

None of these is a stray sentence; each is doing work something else depends on.

1. **`Address`'s membership rule.** *"In the estate exactly while a current resolution cites it **or
   a `Seed` covers it**."* [#48](https://github.com/winniel123/verge-asm/issues/48) reads the two
   grounds as expressly **disjunctive** — *"so there is no shared timeline and nothing to
   arbitrate."* Coverage by a `Seed` is on its own sufficient.
2. **[#63](https://github.com/winniel123/verge-asm/issues/63) on who supplies a key.** *"`Name` (the
   FQDN) and `Address` (the IP) have keys the **world** supplies — a source names one, **a
   resolution or a `Seed`** names the other."* A `Seed` is listed as an `Address` key supplier
   beside a resolution.
3. **The nameless `Endpoint`, said twice.** `CONTEXT.md`: the `Name` may be absent, *"a real,
   distinguishable measurement mode rather than a null in a key, and **the only one available on an
   address-scope `Seed` where no name is known yet**."*
   [ADR-0006](./0006-subjects-leave-by-measurement.md) says it independently — *"the only endpoint
   available on an address-scope `Seed` **before any name is known**"* — and builds a carve-out in
   its withdrawal rule to accommodate it. Under bounding, every address in the estate arrived
   through a resolution, so a `Name` is always known and this measurement mode has **no producer at
   all**. It exists because address scopes enumerate, and one ADR has already written code-shaped
   rules around it.
4. **[ADR-0013](./0013-custody-is-control-and-extends-by-declaration.md) §4's self-correction
   argument.** A stale literal `/32` *"holds the gate **open on a stranger's machine** — and nothing
   notices … the only overreach in the model that grows silently over time."* If an address-scope
   `Seed` only bounded, a released elastic address that nothing of the operator's resolves to would
   never be a subject and never be touched, and there would be no overreach to fear. **§4 is sound
   only under enumeration**, and §4 is the ground the `custody extension` was decided on.
5. **[#50](https://github.com/winniel123/verge-asm/issues/50) on exclusions.** *"Declining
   `198.51.100.128/25` inside a confirmed `198.51.100.0/24` carves a hole in the boundary. **128
   addresses stop being probed**, and their `Custody` reads `third-party` from that date."* An
   exclusion over dark space stops nothing unless the enclosing scope was being walked.
6. **[#48](https://github.com/winniel123/verge-asm/issues/48)'s own routing.** It sends *"I declared
   a /24 and you found nothing in it"* to `Coverage` as *"a statement about a **declared scope we
   completed**."* A scope is completed by walking it, and that complaint is not a sentence an
   operator can utter unless all 256 were looked at.

7. **[ADR-0005](./0005-scan-execution-model.md)'s batch partitioning was written for this.** It
   rejects *"a batch defined as 'the prober against a whole address scope'"* — as a **transaction
   length** problem, not as a category error — and settles on *"our prober against one address is
   enumerable over `(that address, that port set)`, so **one address per batch**."* A `Scan` over a
   `/22` is therefore 1,024 batches per dispatch, each recording its own completed scope, and the
   per-host rate limit stays intra-job with no cross-worker coordination. That machinery is already
   built and it has no other customer: under bounding, a scope's batch count would be however many
   addresses resolved, and the whole passage would be about nothing.

Item 6 is worth dwelling on, because #48's neighbouring sentence — *"asserts nothing about what is
inside"* — is the single best line the bounding case has. It survives this ruling untouched and
means what it said: a `Seed` is **not a source**, holds no timeline, and makes no claim about
whether anything is listening. It supplies **scope**; the prober supplies **values**. Those are
different jobs, and #48 was separating a `Seed` from a source, not a scope from a target list.

### The gloss survives because it was never about addresses

*"It declares a boundary, not a starting point"* has been read three ways, and two of them are
right.

- **Against a name scope it is the whole rule and does real work.** Declaring `example.com` does not
  license generating names from it. §10 refuses *"brute-force enumeration of … subdomains by
  wordlist"* by name; names arrive from CT, the zone file and resolution, and the `Seed` decides
  which of them are inside. There, *starting point* means exactly *seed of a search*, and it is
  refused.
- **Against time it is [ADR-0044](./0044-a-one-off-measurement-has-no-currency.md)'s reading** — a
  declaration is not a starting **gun**, because §6.4 says *"Never scan on config save."* That
  survives unamended: declaring a `/22` queues, it does not fire.
- **Against an address scope's arithmetic it says nothing**, and this is the reading that is
  withdrawn. A name scope cannot enumerate because its membership is unbounded and unknowable
  without a source. An address scope's membership is **total, arithmetic, and requires no source at
  all**: `198.51.100.0/24` *is* 256 addresses in the way `example.com` is not any set of names. The
  glossary's own `Completeness` entry already has the word for it — *"an `enumerable` source returns
  a complete set over a declared scope"* — and an address scope is the one declaration in the model
  that is complete on its face.

So the two `Seed` kinds are **not two flavours of one thing**, and the model is better for saying
so: a name scope is a **custody and filtering** declaration, an address scope is a **sweep**
declaration. Both gate. Only one enumerates.

### The enumeration is one hop deep and stops without a rule telling it to

The obvious fear is that enumeration cascades. It does not, and nothing new is needed to stop it.

- `Seed` → `Address` is arithmetic. **This is the whole of it.**
- `Address` → `Service` was already computed, not enumerated from a declaration:
  [#63](https://github.com/winniel123/verge-asm/issues/63) — *"Given an `Address` in the estate and
  the `Batch`'s recorded port scope, the set of `Service` subjects is **computed**"* — and the port
  scope is [ADR-0044](./0044-a-one-off-measurement-has-no-currency.md)'s business, not this ADR's.
  On a default install that is `verge-core`, and the cold tier stays opt-in per `Seed` scope.
- `Service` → `Endpoint`, `certificate`, `http-identity`, `tls-acceptance` all require an **open
  port**, so they are bounded by measurement automatically. A dark address costs one `Address` row
  and `|verge-core|` `Service` rows and nothing else.
- **No `Name` is ever enumerated.** There is no reverse-DNS in v1 — PTR is out of the qtype set
  (#62), and ADR-0013's own thin-ground section leaves a PTR measurement explicitly unpriced. So an
  address-scope estate produces nameless `Endpoint`s, which is exactly what `CONTEXT.md` says it
  produces.

The ticket asked whether the two readings are really one thing under a `Reach` leg. They are, and
the shape falls out of the existing subject algebra rather than needing a rule: **an address scope
enumerates for reachability and bounds for everything else**, because reachability is the only facet
whose subject can be composed without an observation.

### The cap is a validation on the declaration, and it is not the cap #27 refused

§3.4 and §9 describe one knob from two ends, and §3.4's end is the operative one: a fat-fingered
`/8` *"**cannot be entered**"*. So the `/22` cap is checked when the operator declares an address
scope — or confirms a `Proposal` into one — and not while a batch runs.

Three reasons to put it there rather than truncating at scan time.

**It keeps `Custody` and enumeration coextensive.** A cap that filtered the target list would leave
the gate open over 16 million addresses we had decided not to look at — a silent aperture, which is
the exact failure [#44](https://github.com/winniel123/verge-asm/issues/44)'s aperture statement
exists to prevent, and it would need a shortfall count that decision 7 would then have to
adjudicate. At the declaration there is no shortfall, because there is no partial scope. **Every
address a `Seed` covers is measured** becomes an invariant, and invariants are what `Coverage` can
speak from.

**It is not [#27](https://github.com/winniel123/verge-asm/issues/27)'s refused cap, and the
difference is structural rather than verbal.** #27 refused *"a size cap … an invented threshold
sitting inside the safety path"* over **registry expansion** — a cap deciding, on a third party's
file, whose addresses are `operator`. This cap sits nowhere near the derivation: `Custody` remains
the total lookup ADR-0013 §2 made it, over whatever `Seed`s exist. What this bounds is a **Declared
act's admissibility**, in front of the operator, with an error they can read and a knob they can
raise. #27's cap failed silently in both directions; this one cannot fail silently at all, because
its only failure mode is a declaration that does not take.

**Nobody is stranded, and the ones who would have been are better served.** The operator who
genuinely holds a `/9` — 96 ARIN holders exceed a million addresses, against a **median holder of
512**, which is inside the cap — is not asking us to sweep 8.4 million addresses monthly, and no
configuration could. Their route is a name scope with a `custody extension`: the gate opens over
exactly the addresses their names resolve to, it is **measured rather than typed**, and ADR-0013 §4
already calls that strictly safer than a literal declaration. They then declare address scopes only
for the ranges they actually want swept blind. That is the division of labour the two `Seed` kinds
were for, and this ruling is what makes it legible.

The cap is **per scope, never a sum** — §9 says *per target* — so four `/22`s are four deliberate
acts and 4,096 addresses. It is a knob for the same reason every §9 knob is one: *"there is a
specific way the default is wrong for some real operator."*

**The objection worth answering in terms** is ADR-0013's own sentence: declaring `3.0.0.0/8` is
*"an ordinary false declaration … which the model has never tried to prevent because it cannot
prevent a false name-scope seed either."* A cap does prevent it, so the sentence looks contradicted.
It is not, and the distinction is the whole of why this cap is admissible where #27's was not: **the
model still never adjudicates truth.** The cap does not believe the `/8` is false — it makes no
claim about that at all, and a `/8` declared with the knob raised is accepted, false or not. What it
refuses is a declaration the **shipped configuration cannot measure**, which is a statement about
us. ADR-0013's sentence is about falsity and survives verbatim; falsity is still uncheckable and
still unchecked, and the escape hatch is still a boundary the operator redraws.

Two facts keep the cost of all this small and both are measured. The address-scope population is
**under 1 %** of installs — [#26](https://github.com/winniel123/verge-asm/issues/26)'s measurement,
carried into [`measurement-offers.md`](../spec/measurement-offers.md) as the ground for ruling PTR
out — and ADR-0013 made the modal entry point a **name** scope holding no address scopes at all. So
enumeration's cost lands on a small minority of installs, deliberately, by their own declaration,
on the axis they declared it for. And the median ARIN holder holds **512** addresses, inside the
cap. The tail this ruling inconveniences is the tail that should be using a `custody extension`
anyway.

**One consequence is owed to a closed ticket.** [#50](https://github.com/winniel123/verge-asm/issues/50)
drew the confirm control with the example label `Confirm 8,388,608 addresses`, on the rule that
*"nothing branches on the number."* The control's shape survives — no warning interstitial, no
second gesture, no per-address anything — but at the shipped default that particular act is refused
by validation, and the example wants restating at a size the default admits. #50's own rule that
address counts are **arithmetic over a CIDR, not a measurement of an estate** is the load-bearing
half and it is untouched.

### What `Coverage` may honestly say about a `/22` seed — and why it is not #28's refused score

This is the question the ruling most changes, and the answer is more generous than expected.

[#28](https://github.com/winniel123/verge-asm/issues/28) refused an estate-completeness score
because *"the entire point of a `Seed` being a scope declaration is that we cannot"* measure how big
an estate ought to be. [#44](https://github.com/winniel123/verge-asm/issues/44) decision 7 carried
that into the aperture statement: **counts of our own rules and lists, never a count or proportion
of the operator's estate.**

An address scope does not offend either, and [#50](https://github.com/winniel123/verge-asm/issues/50)
already said why in terms: ***"No figure on this screen is a coverage figure. Address counts are
arithmetic over a CIDR, not a measurement of an estate, so #28 is satisfied."***

So the **address axis of an address-scope `Seed` is the one aperture dimension in the product with a
total, exact, non-estimated denominator** — not because we discovered how big the estate is, but
because the operator typed a number and arithmetic closes it. `Coverage` may state *1,024 addresses
declared in `198.51.100.0/22`; 1,024 measured on the daily tier* and it is honest, closed and ours
on both sides.

[`prototypes/coverage/index.html`](../../prototypes/coverage/index.html) had already drawn exactly
that — `254 of 254` for a `/24`, and `0 of 254` when the vantage went dead — so this ruling
ratifies a screen rather than redesigning one. Two corrections fall out of it and both are small.
The denominator should read **256**, not 254: excluding the network and broadcast addresses infers a
subnetting we did not measure, and the reserved pair costs two connects that read `not-reached`
like any other silence. And the fault string *"`${m.of - m.n}` names not reached"* is name-shaped
copy on a row whose units are addresses — the same slip this ADR exists to stop, one layer up.

Three riders keep this from generalising into the thing that was refused.

- **It does not extend to a name scope.** `CONTEXT.md` already calls the `custody extension`'s
  rendering *"a **census with no denominator**, since how many addresses it *ought* to cover is
  completeness of the estate"*. That stands, and the contrast is now the cleanest statement of this
  whole ADR: **an address scope has a denominator because it enumerates; a name scope has none
  because it does not.** One sentence covers both the membership rule and the coverage rule.
- **It does not extend to ports.** ADR-0044 refused *65,395 of 65,535 unread* and that refusal is
  untouched; the port-tier line still states the tier, its cadence and its off state.
- **It does not extend to undeclared space.** An operator who holds a range and never declares it
  has no denominator, gets no number, and surfaces in #28's propose half as prose. Enabling nothing
  can raise the figure — which is #28's own test, passed.

**[#80](https://github.com/winniel123/verge-asm/issues/80)'s ~~`0 of 37 sensitive pairs unread`~~
~~`0 of 41 sensitive pairs unread`~~ **`0 of 40 sensitive pairs unread`** (#109 removed `1433/tcp`)
survives unchanged and is strengthened.** It is a count over *our* list, evaluated per `Service`, and
[ADR-0009](./0009-verge-core-is-a-union.md) puts every sensitive pair inside the daily tier by
construction. Enumeration multiplies the `Service`s the claim ranges over and changes nothing about
the claim; what it adds is that the population is now a **declared, counted** one rather than
whatever happened to resolve. Likewise `0 of 16 rules unevaluable`.

### The arithmetic, and what it does to ADR-0044

ADR-0044 stated every figure per address and gave the estate limb twice because this question was
open. It can now be stated once. For an address-scope `Seed` the multiplier is the **declared size**,
at most **1,024** at the shipped cap.

Working from §6.3's shipped defaults against `verge-core` at ADR-0009's ~~~140 pairs~~ **136 pairs, 131
probed** (see the correction below), one `/22`, at the
**200 pkt/s global ceiling** (the per-host caps do not bind across 1,024 hosts):

| Tier | Attempts per pass | Answering | Dropping, 2 retries |
| --- | --- | --- | --- |
| **Hot (`verge-core`, daily)** | 143,360 | **11 min 57 s** | **35 min 50 s** |
| **Cold (full range, monthly, opt-in)** | 67,108,864 | **3.9 days** | **11.6 days** |

> **`~140` was never `verge-core`'s size, and the hot row therefore overstates.** **[measured]** by
> [#97](https://github.com/winniel123/verge-asm/issues/97): the frequency half is **123, all TCP**, and
> the union is **136 pairs — 131 probed on default settings**, UDP being off
> ([`sensitive-ports.md`](../research/sensitive-ports.md) §29, composed with
> [#95](https://github.com/winniel123/verge-asm/issues/95)'s two admissions). At 131 the hot row reads
> **134,144 attempts · 11 min 11 s answering · 33 min 32 s dropping**. **The cold row does not move** —
> it is the full range and reads no port list. **The original figures are left standing per the
> name-and-withdraw convention and no ruling moves**: this ADR's conclusions turn on the **cold** tier's
> 3.9-to-11.6-day pass against a monthly cadence, and the hot tier's overstatement is in the
> conservative direction.

Three things follow.

**The default fits, and comfortably.** A `/22` — the largest scope the shipped configuration
admits — costs 12 to 36 minutes a day. That is the figure ADR-0044 could not state, and it is what
makes this ruling affordable rather than merely correct.

**ADR-0044's losing option loses harder.** *Ship the cold `Scan` enabled at monthly cadence* was
rejected partly because *"at §9's shipped `/22` cap it is a 3.9-to-11.6-day pass, monthly."* That
limb was conditional on this ticket. It is now unconditional: the cap's own maximum is a scope that
cannot complete inside a monthly cadence, so ADR-0005's overlap rule turns it into a permanent skip.
Nothing in ADR-0044 is amended; its conditional becomes a fact.

**The estate sizing in [ADR-0001](./0001-stack-and-runtime.md) is a prior, not a bound, and one
legal declaration doubles it.** ADR-0001 sizes *"~500 live assets against the ~140-port hot set …
~70k reachability observations per day — ~25M rows/year"* — **at the measured 131 probed ports, ~66k
per day and ~24M rows/year; the ratio this paragraph turns on is unchanged.** A single `/22` is
**1,024 addresses and 143,360 reachability observations per day, ~52M rows/year** (**134,144 and ~49M
at 131**) — 2× ADR-0001's whole sized estate, from
one line of typing, before any name scope. The retention question on the map is sized against
resolved addresses and must be sized against **declared scope size**, which is the one input to it
the operator sets directly. This is a real consequence and it is stated rather than absorbed.

### The `Gap` interaction, in #42 and #48's own vocabulary

The ticket was right that an address that enumerates but never answers is a different object from an
address that was never a subject. There are **four** objects and the model already names all of
them; enumeration is what populates the first, which under bounding is unreachable.

| The address | What it holds | Ruled by |
| --- | --- | --- |
| Inside a declared scope, walked, nothing ever answered | An `Address` subject; `Service`s exist; `Reach` = **`not-reached`, a measured value** | This ADR; `Service` *"open or closed"* |
| Walked, then we stopped — custody withdrawn while a resolution still cites it, vantage `unavailable`, currency lapsed | A **`Gap`**, recording its cause | [ADR-0014](./0014-only-revealed-generalises.md), ADR-0013 §5 |
| Left a declared scope — excluded, or the scope narrowed — and nothing cites it | The **subject leaves**, taking its timelines. **No `Gap`**, because no subject is left to hold one | ADR-0013 §5's table, by parity |
| Never declared, never cited | **Nothing at all** — no subject, no timeline, no `Gap`, no row ever | [#44](https://github.com/winniel123/verge-asm/issues/44) decision 6 |

Row one is the load-bearing addition and the reason the whole question mattered. `not-reached` is a
**value** decided by the `connect-outcome` leaf, not an absence — so it is diffable, it is what
`unreachable` is a verdict about, and it is what a listener later moves off. `Reach`'s own entry
already anticipated the distinction: the absence is *"a `Gap` where the timeline was already running
and **nothing at all** where it never began."*

Two adjacencies fall out and neither needs new machinery. Declaring or widening an address scope
moves the **custody gate**, one of the ~~five~~ aperture inputs — *the count is withdrawn; it is
**seven** since [ADR-0017](./0017-exposure-needs-both-legs.md) and
[ADR-0066](./0066-a-control-probe-is-generated-under-a-names-parent-and-that-population-is-aperture.md),
and the gate's membership is untouched* — so it is `revealed` on every timeline it
opens and **one** coverage-class message at the scope carrying *"a count of timelines opened and no
comparison at all"* — #63's *"adding a `Seed` later behaves identically"*, applied. It is emphatically
**not** 1,024 membership messages; ADR-0022's unit rule (*the unit is the scope, never the address*)
and ADR-0013's once-per-scope extension message are the same shape and settle it. And an `Address`
inside a declared scope therefore never `appear`s at all — `appeared` on an `Address` stays the
resolution route — which is a tidier partition than the model had before.

The message is **large**: one `/22` opens 1,024 × 140 × 2 ≈ **287,000 `Reach` timelines** under a
single count. That is the magnitude the map's open patch asks about, and it is within the shape
ADR-0044 already accepted for the cold-tier enable (~1.3 million). [ADR-0031](./0031-membership-alerts-at-the-root-of-the-entering-subtree.md)
is what keeps it to one message: a `Service` entering is never a message.

## Consequences

- **[`CONTEXT.md`](../../CONTEXT.md)'s `Seed` entry gains the enumeration limb** and scopes *"a
  boundary, not a starting point"* to the two things it refuses. `Address`'s second membership limb
  is confirmed rather than amended, and gains the fact that a `Seed`-covered address is a subject
  from the declaration. `Custody` gains the narrowing case. No term is added.
- **[ADR-0002](./0002-ownership-gates-probing.md) is amended**: its gate table's *"probing
  permitted"* now has a stated **extension** on the `operator` side for an address scope, and the
  amendment records that this is not #27's refused size cap.
- **[ADR-0013](./0013-custody-is-control-and-extends-by-declaration.md) is amended** to record that
  §4's self-correction argument presupposes enumeration and is the stronger for it, and that the
  range cap does **not** apply to a `custody extension`, whose extent is measured and self-limiting.
- **[ADR-0044](./0044-a-one-off-measurement-has-no-currency.md) is confirmed, not amended.** Its
  per-address figures stand; its two estate limbs collapse to one; its unresolved-seam bullet is
  discharged by this ADR; and its rejection of the enabled cold `Scan` stops being conditional on
  this ticket. Its ~~`0 of 37 sensitive pairs unread`~~ ~~`0 of 41`~~ **`0 of 40`** is untouched by this ticket; the
  denominator moved with the sensitive list — down by one when [#109](https://github.com/winniel123/verge-asm/issues/109) removed `1433/tcp` — and the numerator is `0` for every value of it.
- **[`safe-active-probing.md`](../research/safe-active-probing.md) §3.4 is confirmed and cited
  rather than corrected**, and §9's `Target range size cap` row gains *where* the cap is applied —
  at declaration, per scope, never to a sum.
- **[#50](https://github.com/winniel123/verge-asm/issues/50)'s confirm control is unaffected in
  shape and owes one example a restatement** — `Confirm 8,388,608 addresses` describes an act the
  shipped default refuses. That is aligned with, not opposed to, the screen's own framing: it
  already records that *"ADR-0013 makes declining all of it the expected act for the majority of
  installs."*
- **Three prototype deltas, all small, none structural.**
  [`coverage`](../../prototypes/coverage/index.html) reads `of: 254` for a `/24` where the ruling
  says 256, and carries `names not reached` as the fault string on an address-scope row.
  [`landing-view`](../../prototypes/landing-view/index.html) tells the operator *"declare one and
  the first batch starts immediately"*, which §6.4's *"never scan on config save"* and ADR-0044
  already contradict — surfaced here as by-catch, not fixed here.
- **[ADR-0001](./0001-stack-and-runtime.md)'s ~500-address sizing is a prior a single legal
  declaration doubles.** The retention patch on the map must be sized against declared scope size.
- **[#12](https://github.com/winniel123/verge-asm/issues/12) can state what the estate *is*:** the
  addresses a resolution currently cites, plus every address inside a declared address scope, and
  nothing else. Both limbs are now decidable without measurement.
- **IPv6 is now a sharp question rather than a latent one.** AAAA is in the shipped resolution offer
  ([`measurement-offers.md`](../spec/measurement-offers.md)), so IPv6 `Address` subjects already
  arrive by resolution; but an *address scope* over IPv6 enumerates, and `/22` is IPv4 notation that
  means 2¹⁰⁶ addresses in v6. Ticketed separately — under the bounding reading it did not arise.

## Amendment — [#85](https://github.com/winniel123/verge-asm/issues/85): the cap counts addresses, and the exponent was the only accidentally-IPv4 thing here

[ADR-0049](./0049-an-address-scope-is-family-agnostic-and-the-cap-counts-addresses.md) states this
ADR's own cap in the unit it always had. **Nothing here is amended in substance and no figure moves.**

- **The range size cap is `1,024` addresses per scope**, checked at declaration, per scope, never on
  a sum, operator-configurable. **`/22` is that count's IPv4 spelling and `/118` is its IPv6 one** —
  one knob at one setting, not two rules. Everything this ADR argues about the cap — that it
  adjudicates *cost, not truth*, that it is not [#27](https://github.com/winniel123/verge-asm/issues/27)'s
  refused cap, that it does not reach a `custody extension` — is unchanged and now family-blind.
- **An address scope is family-agnostic.** A CIDR is a CIDR; there is no family check anywhere in
  the model, the `Seed` kind does not split, and no third kind appears. The enumeration reads
  **2^(width−n)** — the `32` above is the only accidentally-IPv4 thing in this document, and it sits
  in the sentence a reader takes the enumeration rule from.
- **The consequence for IPv6 is arithmetic, not a policy.** `/118` and longer are declarable, of
  which the usable member is the `/128`; `/117` and shorter are refused by the cap, which is every
  prefix an operator is assigned. One `/64` on the daily tier is ≈ 4.1 × 10¹¹ years at §6.3's
  ceiling, so **IPv6 space is not swept and no configuration makes it sweepable**. The IPv6 estate's
  route is a **name scope with a `custody extension`**, which is family-agnostic and already works,
  since AAAA is in the shipped resolution offer.
- **This ADR's ≈287,000 `Reach` timelines for a `/22` is therefore a ceiling**, not a point in an
  open range: 1,024 addresses is the most any single address-scope declaration can open, in either
  family, at the shipped default.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **An address scope only bounds: it opens the `Custody` gate and nothing else, and addresses enter solely by citation** — the *losing option*, and the reading `Seed`'s own gloss most naturally supports | **It lost on the flagship message, not on the gloss.** A listener appearing in declared space could never produce `not-reached` → `reached`, because there would be no prior span; the news would arrive as a census line, and only on address-scope estates. It also leaves the nameless `Endpoint` with no producer, makes ADR-0013 §4's *"gate open on a stranger's machine"* unsound, makes #50's *"128 addresses stop being probed"* false, makes #48's *"a declared scope we completed"* unutterable, and deletes the *"no ports responded"* observation §3.4 says the tool exists to record. Five load-bearing structures against one sentence that was written about name scopes |
| **Walk the range, but make subjects only of addresses that answer** — the strongest middle position, and the one that bounds the subject count by observation | Fails identically on the flagship, and worse: it makes the first port opening on a dark address a **membership** event, which `CONTEXT.md` forbids in terms (*"a port opening is a `Reach` move and never a membership event"*). It also makes a `Batch`'s recorded scope stop meaning what ADR-0005 says it means — #63's finding, verbatim: *"a population defined by what answered licenses nothing"* |
| **Enumerate, and let the cap truncate the target list at scan time** | Leaves `Custody` open over addresses we decided not to look at — a silent aperture with a shortfall count that #44 decision 7 would then have to adjudicate. Validating at declaration means there is no partial scope and no number to argue about |
| **No cap; let the operator declare whatever they like and price it** | §3.4's *"cannot be entered"* is a validation, and a `/8` is 4,096 × the cost of the largest scope the defaults can complete. A default that cannot finish inside its own cadence is not a default (ADR-0044) |
| **No cap, on ADR-0013's own ground that declaring `3.0.0.0/8` is *"an ordinary false declaration … which the model has never tried to prevent"*** | The sharpest objection to the cap, and it misses by one word. The model does not adjudicate **truth** and still does not: the cap makes no claim about whether the `/8` is theirs, and accepts it once the knob is raised. It refuses a declaration **we cannot measure**, which is a fact about our shipped configuration rather than about their estate |
| **Exempt network and broadcast addresses from the enumeration** | Which addresses those are depends on the operator's subnetting, which we never measure. Guessing it is an inference the project refuses everywhere else, and the saving is two connects per `/24` that read `not-reached` like any other silence |
| **Cap the sum of declared address scopes rather than each one** | §9 says *per target*. A sum makes the tenth declaration fail for reasons nine earlier ones caused, which is an invented threshold behaving exactly as #27 said one would |
| **Apply the range cap to a `custody extension` as well** | Its extent is measured and self-correcting, not typed; capping it would silently drop addresses the operator's own names resolve to, which is the failing-silently ADR-0013 §7 refused. The cap is a check on a **Declared** act |
| **Enumerate names from a name scope for symmetry** | §10's refused wordlist brute-force, and the half of *"a boundary, not a starting point"* that is entirely right |
| Enumerate, and reverse-resolve each address so `Endpoint`s get names | A new measurement carrying ADR-0011's six obligations; PTR is out of the v1 qtype set (#62) and ADR-0013 left it explicitly unpriced. The nameless `Endpoint` is the model's existing answer |
| Fire one `appeared` per address on declaration | 1,024 messages for one operator act. ADR-0022's unit rule, ADR-0013's once-per-scope extension message, and #63's *"adding a `Seed` later behaves identically"* all give the same answer: one coverage-class message at the scope |
| **Refuse enumeration because it makes the estate an inventory** — `CONTEXT.md`'s first line is *"its subject is not inventory but change"* | The strongest form of the losing case, and it inverts. An inventory is a list of live hosts; a subject holding a `not-reached` span is the **apparatus of change**, and it is the only apparatus that can date a listener's arrival. Nor are the dark rows invented state in ADR-0013 §2's sense: `unknown` was a Derived value with no producer, while `not-reached` is measured by `connect-outcome` every cadence |
| Leave the seam open and let the implementation decide | It is the difference between a minutes-long and an eleven-day scan on one declaration, the denominator of every count in the product, and the last thing #12 was waiting for |
