# ADR-0049: An address scope is family-agnostic, and the range cap counts addresses rather than prefix lengths

- **Status:** Accepted
- **Date:** 2026-08-14
- **Ticket:** [#85 Can an address-scope `Seed` be an IPv6 prefix, and what does the range cap mean if it can?](https://github.com/winniel123/verge-asm/issues/85)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)
- **Amends:** [ADR-0047](./0047-an-address-scope-is-its-own-enumeration.md), [ADR-0013](./0013-custody-is-control-and-extends-by-declaration.md)

## Context

[ADR-0047](./0047-an-address-scope-is-its-own-enumeration.md) ruled that an address-scope `Seed`
**enumerates** — every address inside a declared CIDR is a subject from the declaration, walked every
cadence — and that a `/22` per-scope range size cap is what makes that affordable. Its whole
arithmetic is IPv4: the enumeration is stated as **2^(32−n)**, the cap as `/22`, the cost table as
packets ÷ 200 pkt/s over 1,024 addresses. Nothing in it is false and nothing in it is about IPv6.

Under the bounding reading the question did not arise, because a declared prefix produced no targets
and its family was as free as its size. Under the ruling neither is free.

Three facts make this a seam rather than an omission, and [#85](https://github.com/winniel123/verge-asm/issues/85)
names all three.

- **IPv6 `Address` subjects already exist and are already probed.** AAAA is in the shipped
  resolution offer, admitted on the *absence-makes-the-measurement-false* limb
  ([`measurement-offers.md`](../spec/measurement-offers.md) §2, citing
  [#3](https://github.com/winniel123/verge-asm/issues/3) §3.1's *v6-only exposure is routinely
  forgotten*). A name scope with a `custody extension` therefore reaches IPv6 today, with no ticket
  and no decision. **This ADR is about the address scope alone.**
- **The cap does not port by changing a number.** `/22` in IPv6 is 2¹⁰⁶ addresses. A `/64` — the
  architectural subnet floor, since RFC 4291 §2.5.1 requires a 64-bit interface identifier for
  essentially every unicast address `[spec]` — is 2⁶⁴. There is no prefix length at which *walk all
  of it* is a coherent request.
- **`Vantage class` reads literal address-scope `Seed`s and nothing else** to decide which side of
  the boundary a prober sits on ([ADR-0013](./0013-custody-is-control-and-extends-by-declaration.md)
  §6, and it refuses the `custody extension` for that check **by design**). A prober with no
  declarable address scope therefore verifies `internet` — the loud, intended failure — but reachable
  for a reason nobody chose, if the operator had no way to declare the scope at all.

## Decision

**An address scope is family-agnostic. There is no IPv6 rule, because the range size cap was never a
statement about prefix lengths — it is a count of addresses, and `/22` is that count's IPv4
spelling.**

| Concern | Decision |
| --- | --- |
| May an address-scope `Seed` be an IPv6 prefix? | **Yes, and the model contains no family check anywhere.** A CIDR is a CIDR; the `Seed` kind does not split and no third kind appears |
| The cap's unit | **1,024 addresses per scope**, operator-configurable, checked at declaration, never on a sum. `/22` in IPv4 and `/118` in IPv6 are the same knob at the same setting. **The shipped value does not move; only its unit is stated** |
| Which IPv6 prefixes are therefore declarable | **`/118` and longer**, of which the member anyone actually uses is the **`/128`** — one address, one subject |
| Which are not | **`/117` and shorter**, which includes every prefix an operator is assigned: `/64`, `/56`, `/48`. Refused **by the existing cap**, with the existing error and the existing knob — not by a rule about IPv6 |
| Does an IPv6 address scope enumerate? | **Yes, identically** — all 2^(128−n) of them, which is exactly why so few are admissible. Enumeration is one hop, `Seed` → `Address`, unchanged |
| Does it carry a `Coverage` denominator? | **Yes, identically** — arithmetic over the declaration, at most 1,024. The enormous-denominator case never arises, because the declaration that would produce it is not admissible |
| An IPv6 address scope that only **bounds** — gate without sweep | **Refused.** See the rationale; it is ADR-0013 §4's silently-growing overreach with the *walked* property removed |
| Sweeping IPv6 space | **Not something v1 does, and not something any configuration makes it do.** One `/64` on the daily tier is ~4.1 × 10¹¹ years at §6.3's shipped ceiling. Out of scope, and the reopen condition is a different instrument, never a different number |
| The IPv6 estate's route | A **name scope with a `custody extension`**, which is family-agnostic by construction and already works. Plus `/128` address scopes for the individual addresses the operator wants covered literally |
| `Vantage class`'s containment test | Over **every address the `Vantage` holds**, of either family. One uncovered address and it verifies `internet` — the closed direction, matching `Custody`'s *everything not covered is `third-party`* |
| A v6-only internal prober | Declares its own address as a `/128` address scope. One address, admissible under the cap, and §6's test then passes. **The residue closes itself, and only because the cap is family-agnostic** |
| The refusal surface | A refused **IPv6** declaration must name the `custody extension` as the route. Naming the knob is correct for IPv4 and a trap for IPv6. **Discharged by [ADR-0052](./0052-a-declaration-refusal-names-a-route-and-never-takes-it.md)**, which generalises it: a refusal names a route and never takes it, because the IPv6 route reaches a *different set* while raising the cap reaches the *same* one. The knob is **named in order to shut it** rather than omitted — silence does not remove a setting that Settings and the IPv4 refusal both point at |

## Rationale

### The cap was always a count, and `/22` was notation

This is the whole ruling and everything else follows from it.

ADR-0047 defends the cap on one ground and the defence is quoted here because it decides this
ticket too: it *"refuses a declaration the **shipped configuration cannot measure**, which is a
statement about us"*, and it *"adjudicates **cost, not truth**"*. Cost is measured in packets, packets
are measured per address, and the number of addresses in a prefix is what the cap has always been
about. `/22` is the IPv4 spelling of *1,024 addresses* in exactly the way `254 of 254` was the
IPv4-shaped spelling of a denominator ADR-0047 had to correct to `256`. **Units slip in this repo,
and this is the same slip one level up.**

Restating the unit invents nothing. The shipped default stays at 1,024 addresses; every figure
ADR-0047 states stays exactly true; and there is no second knob, no family branch, and no new
threshold. That last point is the [#27](https://github.com/winniel123/verge-asm/issues/27) test the
ticket carries — *do not invent a threshold inside the safety path* — and a unit restatement passes it
trivially, because the only thing that changes is how one existing number is written down.

The consequence for IPv6 is arithmetic and needs no survey of anybody's estate. 1,024 addresses is
`/118`. The addressing architecture's subnet floor is `/64` `[spec]`, which is 2⁶⁴ addresses — larger
than the cap by a factor of **2⁵⁴**, about 1.8 × 10¹⁶. So every prefix an operator is assigned is
refused, and it is refused by the same instrument that refuses a typo'd `/8`, saying the same thing:
*we cannot measure that.*

### It really is unmeasurable, and the arithmetic is ours rather than anybody's opinion

Following ADR-0047's own method — attempts ÷ §6.3's 200 pkt/s global ceiling, against
[ADR-0009](./0009-verge-core-is-a-union.md)'s ~~~140 pairs~~ **136 pairs, 131 probed** on the daily
tier:

| Scope | Addresses | Attempts per hot pass | One pass at 200 pkt/s |
| --- | --- | --- | --- |
| `/22` (IPv4, the shipped cap) | 1,024 | 143,360 | **11 min 57 s** |
| `/118` (IPv6, the same cap) | 1,024 | 143,360 | **11 min 57 s** |
| `/64` (IPv6, the smallest thing anyone is assigned) | 18,446,744,073,709,551,616 | 2.58 × 10²¹ | **≈ 4.1 × 10¹¹ years** |

> **`~140` was never `verge-core`'s size** — **[measured]** by
> [#97](https://github.com/winniel123/verge-asm/issues/97), the frequency half is **123, all TCP** and
> the union is **136 pairs, 131 probed on default settings** ([`sensitive-ports.md`](../research/sensitive-ports.md)
> §29, composed with [#95](https://github.com/winniel123/verge-asm/issues/95)). At 131 the table reads
> **134,144 attempts / 11 min 11 s** for both `/22` and `/118`, and **2.42 × 10²¹ / ≈ 3.8 × 10¹¹ years**
> for the `/64`. **The original figures stand per the name-and-withdraw convention and the ruling does
> not move**: 3.8 × 10¹¹ years is still roughly thirty times the age of the universe, which is the whole
> of the argument.

That is roughly thirty times the age of the universe, for the *daily* tier, on the *smallest* IPv6
subnet the architecture defines. It is not an expensive request that a knob could buy; it is a
request no instrument can serve.

So the honest v1 statement, and the one the `Seeds` screen has to be able to make, is: **verge-asm
cannot sweep IPv6 space, and no configuration makes it able to.** That is a fact about the address
space and about our shipped rate limits, both of which we can state without measuring a single
operator.

### The losing option, and why ADR-0013's stated reason does **not** dispose of it

The serious alternative is an IPv6 address scope that **only bounds**: it opens the `Custody` gate
and feeds `Vantage class`, produces no target list, and carries no `Coverage` denominator. This is
the reading ADR-0047 rejected for IPv4 — but it must be re-argued here rather than inherited,
because ADR-0047 rejected it on the **flagship message**, and the flagship argument is unavailable
in IPv6 for a reason that has nothing to do with our choices.

ADR-0047's decisive argument was that an operator declares `198.51.100.0/24` *because they want to
know when a machine appears in that space*, and that only enumeration lets that arrive as
`not-reached` → `reached` rather than as a line in a census. In IPv6 **nobody can offer that job at
all** — see the arithmetic above — so bounding does not silence the flagship there; the flagship was
never on offer. The argument that decided #81 does not transfer, and pretending it does would be
manufacturing consensus with a quotation.

[#85](https://github.com/winniel123/verge-asm/issues/85) points at
[ADR-0013](./0013-custody-is-control-and-extends-by-declaration.md)'s rejected-alternatives row
against a third `Seed` kind — *"nothing here needs a new key or lifecycle"* — and asks whether the
reason still holds. **Checked, and it does not reach this.** A bounding-only IPv6 scope needs no new
key (it is a CIDR) and no new lifecycle (`Address` has none, alone among the subjects). The row is
about keys and lifecycles and it is silent here. The losing option has to be beaten on its own
merits, and it is, on three:

**One — it is ADR-0013 §4's silently-growing overreach with the safety property removed.** §4 is the
ground the `custody extension` was decided on: a stale literal declaration *"holds the gate **open on
a stranger's machine** — and nothing notices … the only overreach in the model that grows silently
over time."* ADR-0047's own amendment to ADR-0013 records that §4 is sound *because* the address is
walked. A bounding-only `/48` holds the gate open over 2⁸⁰ addresses **none of which is ever
measured**, permanently, with no cadence at which anything could notice. That is the same overreach,
with the one property that bounds it deleted, at twenty-four more orders of magnitude. The
instrument that exists precisely for this hazard — the `custody extension`, *measured rather than
typed*, self-correcting, and family-agnostic already — is strictly better at the same job.

**Two — it breaks the invariant `Coverage` speaks from.** ADR-0047 put the cap at declaration rather
than at scan time to keep `Custody` and enumeration **coextensive**: *"every address a `Seed` covers
is measured becomes an invariant, and invariants are what `Coverage` can speak from."* A
bounding-only scope is by construction a `Seed` covering addresses that are not measured — the
silent aperture, arriving through the front door instead of through truncation. And it would give the
model an address scope with **no denominator**, which collapses ADR-0047's cleanest sentence: *an
address scope has a `Coverage` denominator because it enumerates; a name scope has none because it
does not.* Under the ruling that sentence survives verbatim and family-blind. Under the alternative
it needs an exception written into the one line the whole ADR compresses into.

**Three — the job is already done.** The two jobs #85 conflates part cleanly, and only one of them
was ever open:

| The operator wants | Served by | Status |
| --- | --- | --- |
| *The v6 addresses my names resolve to are mine — probe them* | A name scope with a `custody extension`. AAAA is in the shipped qtype set, the extension's extent is measured, and neither has ever been family-aware | **Already works, today, with no change** |
| *Tell me when a machine appears in `2001:db8::/48` that I have no name for* | Nothing, ever | **Impossible, not refused** |

So a bounding-only kind would add the model's worst instrument for a job its best instrument already
performs, and would not touch the job that is actually missing, because that job cannot be performed.

### Refusing by family would have cost something, and it is not obvious in advance

The other losing option is the tidy-looking one: **check the family and refuse IPv6 address scopes
outright.** It reaches the same practical place for `/64` and above, so the difference looks
cosmetic. It is not, and the `Vantage class` limb is what shows it.

ADR-0013 §6 makes `Vantage class` read **literal address-scope `Seed`s only, never the extension**,
and that is deliberate rather than incidental — a vantage whose class moved because a DNS answer
changed *"would shuttle observations between the two legs of `Exposure` and manufacture drift in the
flagship value."* So the `custody extension`, which answers everything else about IPv6, is refused
for this one check by design. A prober that holds only an IPv6 address inside the operator's own
network would then have **no legal route** to verify `internal` under a family refusal: the
extension is barred, and the declaration is barred, so it verifies `internet` forever and
over-reports `exposed`.

Under the family-agnostic cap it has a route, and the route is the model working rather than a
workaround: the operator declares the prober's own address as a **`/128` address scope**. One
address, well inside the cap, enumerated at a cost of one `Address` row and `|verge-core|` `Service`
rows. §6's containment test then passes with nothing added to it.

That is the argument that separates the two options, and it is only visible because the cap is
denominated in addresses. **A family check would have been a claim about IPv6; the cap is a claim
about us** — which is the exact distinction ADR-0047 used to admit the cap where
[#27](https://github.com/winniel123/verge-asm/issues/27)'s was refused, and it does not survive
being re-expressed as a family branch.

The `/128` route inherits ADR-0013 §4's warning about literal declarations perishing dangerously,
and it is worth saying that the warning does not bite here: §4's hazard is a cloud provider
reallocating a released address to a stranger, and the operator's own prober is an address they
control and can see. It is also the one case where ADR-0013's *"the operator types `/32`s"* rejected
row is not in play, because this is not the estate — it is one machine of ours.

### `Vantage class`'s containment test is over every address the vantage holds

This is the one limb this ADR **adds** rather than reads out of an existing document, and it is
flagged as such.

`CONTEXT.md` and ADR-0013 §6 say the class is *"re-verified every batch against the address-scope
`Seed`s the system already holds"*. They do not say what a **dual-stacked** prober is verified
against, and until AAAA-fed addresses met an address scope there was nothing to decide. Two readings:
**any** address covered, or **every** address covered.

It is **every**, on the ground ADR-0013 §6 already states about its own failure mode: *"a false
`exposed` is investigated, a false quiet reading is not."* The permissive reading lets a prober whose
IPv4 address sits in a declared scope verify `internal` while its IPv6 address sits somewhere
undeclared — and a vantage wrongly verified `internal` moves observations onto the leg that
**never alerts**, which is the quiet failure. `Custody` already picked this direction in terms:
*everything not covered is `third-party`, which is the closed direction.* The class check takes the
same one.

Two riders keep this small. A **target's** family never moves a leg: `Vantage class` is a property of
where the prober sits, not of what it is probing, so an `internal` prober measuring an IPv6 target
records on the `internal` leg exactly as it does for IPv4. And §6's deliberate two-sets divergence is
untouched — the extension still may not decide a vantage's class, in either family.

> **Amended 2026-08-15 by [#124](https://github.com/winniel123/verge-asm/issues/124): the
> quantifier stands and the *set* is narrowed to the addresses the vantage is observed to
> **present**.** **`every`** is not reopened — the closed-direction argument above is untouched and
> is what the narrowing runs under.
>
> What this ADR left open is the question one layer down: *which* addresses a vantage holds, and how
> we learn them. The answer is [#14](https://github.com/winniel123/verge-asm/issues/14)'s own two
> self-contained checks and nothing else — a prober's presented address is the one the instance
> dialled, known by construction; the instance's is `SSH_CLIENT` as the prober reports it. **An
> interface address is not a presented address.** The forcing argument is **this ADR's own cap**:
> read literally, a NATed instance could verify `internal` only by declaring its own LAN as an
> address scope, and the 1,024-address cap **refuses** anything above a `/22` — so an operator on
> `10.0.0.0/8` could never verify one and `Exposure` would be unreachable by construction on their
> install. The narrowing is forced rather than chosen.
>
> **The dual-stack residue this section was written for is narrowed rather than closed, and that is
> disclosed rather than smoothed.** A prober dialled over IPv4 has an IPv6 egress **nobody observed**,
> so `every` runs over a set that does not contain it. That is a smaller hole than the permissive
> `any` reading and it is not nothing: closing it needs a second observation from a second family,
> which the instrument does not make and which this ticket did not buy. Ticketed rather than guessed.

### Where this is thin, stated rather than smoothed

- **No measurement exists of how operators hold or subdivide IPv6 space**, and the ticket flags it.
  The ruling is **deliberately insensitive** to that gap: the cap refuses whatever exceeds 1,024
  addresses regardless of what anybody typically holds, and the only fact about IPv6 the decision
  leans on is the architectural `/64` floor, which is a specification `[spec]` rather than a survey.
  What the missing measurement affects is **how often the refusal fires**, not whether it is right.
  No research limb is filed on that ground, and this paragraph is the flag.
- **The large-holder route is asserted, not measured, and this ADR inherits that from ADR-0047**,
  which flagged it in the same words for IPv4: nobody has measured whether estates over the cap are
  actually name-reachable. There is a reason to expect it to be *more* true in IPv6 than in IPv4 —
  a host you cannot find by sweeping is a host you find by name — but that is reasoning about
  deployment reality of the kind ADR-0013 §4 marks *motivating rather than load-bearing*, and it is
  marked that way here too.
- **Raising the knob above the cap is the operator's, and for IPv6 it is a worse bargain than for
  IPv4.** ADR-0047 accepted that a knob-raised `/8` is admissible though it cannot complete inside
  its cadence; an IPv6 `/64` is the same shape, unimaginably further along it, and ADR-0005's overlap
  rule turns it into a permanent skip with `Coverage` pinned near zero forever. **No ceiling is
  invented** — that would be #27's threshold arriving in the one place this model has kept clear —
  so the whole weight falls on the refusal copy naming the extension rather than the knob. ~~That is a
  surface obligation carried by an unfinished screen, which is a real dependency and is stated as
  one.~~ **The screen is drawn and the obligation is discharged** —
  [ADR-0052](./0052-a-declaration-refusal-names-a-route-and-never-takes-it.md) and
  [#123](https://github.com/winniel123/verge-asm/issues/123),
  [`prototypes/seeds/`](../../prototypes/seeds/index.html).

## Consequences

- **[`CONTEXT.md`](../../CONTEXT.md)'s `Seed` entry** states the cap in addresses, states that a
  CIDR is family-agnostic, and states the IPv6 consequence. **`Vantage class`** gains the
  every-address-held limb. No term is added and no term changes meaning.
- **[ADR-0047](./0047-an-address-scope-is-its-own-enumeration.md) is amended, not reopened.** Every
  figure it states is unchanged and every argument it makes survives; what it gains is the unit of
  its own cap and the family-blindness of its own rule.
- **[ADR-0013](./0013-custody-is-control-and-extends-by-declaration.md) §6 is amended** with the
  containment test's extent, and §4's warning is noted as not reaching the prober's own `/128`.
- **[`safe-active-probing.md`](../research/safe-active-probing.md) §9's cap row gains its unit.** The
  value does not move.
- **`1,024 addresses` is now the ceiling on what any single address-scope declaration can open, in
  either family.** ADR-0047's ≈287,000 `Reach` timelines for a `/22` is therefore the **maximum** an
  onboarding declaration can produce at the shipped default, rather than one point in an open range.
  The map's aperture-at-magnitude patch loses a route to a larger number rather than gaining one.
- **v1's IPv6 story is statable in one sentence**, which it was not before: *IPv6 addresses enter the
  estate by resolution and are probed like any other; IPv6 space is not swept, and an IPv6 address
  scope is admissible only at the sizes the shipped configuration can measure.*
- **The `Seeds` screen owns a refusal it did not know it had.** #81 gave it the cap's refusal and
  knob; this adds that the IPv6 refusal must route to the `custody extension` and must not offer the
  knob as the remedy.
- **A subject-key seam is exposed rather than created**, and is ticketed: `Address`'s natural key has
  no stated canonical form, and IPv6 has many textual spellings of one address where IPv4 effectively
  has one. This is live **today** through AAAA and is not caused by this ruling — but a ruling that
  says *a CIDR is a CIDR* is the point at which somebody will ask what `2001:db8::1` is equal to.
  [#89](https://github.com/winniel123/verge-asm/issues/89). **Discharged by
  [ADR-0051](./0051-a-subject-key-is-the-thing-denoted-and-its-normalisation-may-never-move.md)**,
  which keys an `Address` on the address rather than on any spelling of it and holds an address scope
  in the same form — so this ADR is **confirmed rather than amended**, and its `/128` route and its
  every-address containment test now run on a stated form instead of an assumed one.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Refuse IPv6 address scopes by a family check** — the tidy-looking option, and the one the ticket lists first | It leaves a v6-only internal `Vantage` with **no legal route** to verify `internal`, because ADR-0013 §6 bars the `custody extension` from that check by design; under the family-agnostic cap the prober's own `/128` is admissible and the residue closes itself. It is also the model's only family-aware rule, needing its own error, its own knob semantics and its own explanation of why AAAA is fine but a v6 CIDR is not — and it makes a claim about **IPv6** where the cap makes a claim about **us**, which is the precise distinction that admitted the cap where #27's was refused |
| **Admit an IPv6 address scope that only bounds** — gate and `Vantage class` input, no target list, no denominator. A third `Seed` behaviour | ADR-0013 §4's silently-growing overreach with the *walked* property removed and the magnitude raised by twenty-four orders of magnitude; it breaks ADR-0047's coextensivity invariant, which is what `Coverage` speaks from; and the job it does is already done better by the `custody extension`, which is measured rather than typed and has never been family-aware. **ADR-0013's third-kind row does not dispose of it** — that row is about keys and lifecycles, and this needs neither — so it is refused on these three grounds and not on that quotation |
| **Port the cap by prefix length: keep `/22` for IPv4 and pick a second, IPv6 prefix cap** | Two knobs where there was one, and the second would have to be chosen against a population nobody has measured — an invented threshold, which is exactly what #27 refused and what ADR-0047 had to work to stay clear of. Denominating the one existing knob in addresses invents nothing: the shipped value does not move |
| **Give an IPv6 address scope a `Coverage` denominator equal to its true arithmetic size** | `0 of 18,446,744,073,709,551,616` is exact, honest, and reads as total failure — the case #28 and #44 decision 7 exist to prevent, arriving through correct arithmetic rather than through an estimate. It never arises under this ruling, because the declaration that would produce it is not admissible, and it is named here so nobody re-derives it |
| **Cap IPv6 scopes but exempt the `custody extension`, as ADR-0047 already does for IPv4** | Not an alternative — this is the ruling. Stated as a row because a session skimming for *IPv6 is capped* will otherwise assume the extension is capped too. It is not, in either family: its extent is measured, and capping a measured set is ADR-0013 §7's fails-silently failure |
| **Invent a hard ceiling the knob may not pass, so nobody can declare a `/64`** | A threshold inside the safety path with no owner and no derivation behind its value — #27's shape exactly. ADR-0047 already accepted that a knob-raised declaration may be uncompletable and left the consequence with the operator; IPv6 makes that bargain worse without making it different. The remedy is the refusal copy, which is a surface, not a number |
| **Sweep IPv6 space with a smarter instrument** — a candidate list from routing data, PTR walking, or DNSSEC zone walking, instead of brute force | Every version is a **new source** with its own admission, consent and completeness properties, not a change to a cap; PTR is out of the v1 qtype set ([#62](https://github.com/winniel123/verge-asm/issues/62)) and reverse-walking an owned range was measured at under 1% of installs ([#26](https://github.com/winniel123/verge-asm/issues/26)). Out of scope on the map, with that as the reopen condition |
| **Say nothing and let the implementation infer the family rule from ADR-0047's `2^(32−n)`** | The exponent is the only thing in ADR-0047 that is accidentally IPv4, and it sits in the sentence a reader takes the enumeration rule from. Left alone it reads as a family restriction nobody decided, which is how the `254` denominator got drawn |
