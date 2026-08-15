# ADR-0063: A routing announcement names the path, not the estate

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#126 Is the BGP leg worth having at all, and does RouteViews join the default set?](https://github.com/winniel123/verge-asm/issues/126)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

The **BGP leg** is the org → ASN → announced-prefixes chain: resolve an organisation to its
autonomous system numbers, ask a route collector which prefixes those ASes originate, and offer the
result to the operator as candidate address scopes. It has been in the corpus since
[#3](https://github.com/winniel123/verge-asm/issues/3) recommended RIPEstat's `announced-prefixes`
call in the Tier-1 default set, and it has been carried forward on one sentence, restated three
times: that it *"reflects **BGP reality** rather than registry paperwork, so it catches space the org
announces but has not tidily registered"*
([`passive-discovery-sources.md`](../research/passive-discovery-sources.md) §4.5).

Four decisions have since moved the ground under it, none of them aimed at it.

- **[#20](https://github.com/winniel123/verge-asm/issues/20)** replaced the instrument and measured
  the yield in the same pass. **RouteViews** `api.routeviews.org/asn/<n>` is keyless, **CC BY 4.0**,
  and the only source measured in this project whose rate limit a naive client cannot turn into an
  error — it paces to ~1 req/s rather than returning 429. And **[measured] 0 of 100** announced
  Safaricom IPv4 prefixes fall outside a registered block; **2 of 11** for Mythic Beasts. The note
  recommended shipping it *"because it is clean and cheap, not because the registry path is
  incomplete without it."*
- **[#26](https://github.com/winniel123/verge-asm/issues/26)** ruled the modal operator
  **registry-less** — 128,233 organisations worldwide hold any RIR delegation, under 1% of the
  persona — and **43.0%** of that registered population holds no ASN at all. So the axis this leg is
  keyed on selects a fraction of a fraction.
- **[ADR-0012](./0012-a-proposer-is-not-a-source.md)** ruled that a thing producing no observations
  is not a `Source`. A route collector admits nothing and its silence licenses nothing, so it yields
  `Proposal`s and carries `consent` alone. [#47](https://github.com/winniel123/verge-asm/issues/47)
  and [ADR-0003](./0003-third-party-source-consent-bar.md)'s third amendment then established that a
  proposer's toggle widens no aperture.
- **[#19](https://github.com/winniel123/verge-asm/issues/19)** shipped RIPEstat **off** under
  `operator-accepted`, and [#47](https://github.com/winniel123/verge-asm/issues/47) scoped that
  toggle to the org→prefix leg alone — *"the honest current position is that RIPEstat's org→prefix
  leg proposes and nothing else about it is in."*

Put together, the ticket's two questions come apart. **Nothing about permission is open**: CC BY 4.0
clears both of ADR-0003's limbs with room, and clearing a consent bar is not an argument for
shipping — ADR-0003 governs whether a source **may** run and is silent on whether it is **worth**
running. And the displacement question has a null answer: RIPEstat's `announced-prefixes` is not in
the shipped set, so there is nothing there for RouteViews to displace. What is actually open is
**value**, and it has never been asked.

## Decision

**A routing announcement names who carries packets toward a prefix, never who controls what listens
in it. It is evidence about the path and never about the estate.**

So a route collector is not a proposer of address scopes, and **the BGP leg does not ship in v1**.
It is recorded on the map's out-of-scope line, on the ground below and with the reopening condition
below.

### The difference set decomposes, and none of its parts is undiscovered estate

Take an AS's announced set and difference it against the registered rows the operator's opaque-id
groups. The residue is exhaustively three things:

| Limb | What the rows are | Where they already belong |
| --- | --- | --- |
| 1 | **More-specifics of registered space** | The operator's own custody, already proposed by the registry leg — and once confirmed, an address scope **enumerates**, so every address in the more-specific is already a subject ([ADR-0047](./0047-an-address-scope-is-its-own-enumeration.md)). The announcement adds decomposition, never territory. **[measured] 100 of 100** for Safaricom |
| 2 | **Space registered to somebody else** that this AS carries — a customer, a transit arrangement | **Not the operator's.** `Custody` is control of what listens ([ADR-0013](./0013-custody-is-control-and-extends-by-declaration.md)); a transit provider carries packets toward its customer's prefix and controls nothing inside it. Proposing it invites the operator to draw a boundary around a third party's network |
| 3 | **Space registered to the operator under a different handle** — a subsidiary | The org-name box's job, and structurally so: a subsidiary holds a different opaque-id, which is exactly why [#43](https://github.com/winniel123/verge-asm/issues/43) shipped the CAIDA join |

The leg's only **unique** limb is 2, and limb 2 is the wrong answer. Limbs 1 and 3 are reached by
paths that already ship.

### The one non-zero yield ever measured is the error mode wearing the finding's clothes

The case for the leg rests on a single figure: 2 of 11 announced Mythic Beasts prefixes fell outside
every registered CIDR, *"roughly 18% of announced IPv4 prefixes … invisible to the registry path
alone."* Mythic Beasts is **a hosting company**, and #20's own gloss of those two rows names the
mechanism: *"space announced from a transit or customer relationship rather than held under the org
handle."* That is limb 2 stated in the note that offered it as limb-2-free yield.

**A yield figure measured on the population where the instrument's error mode lives is not a yield
figure.** The corpus has refused this shape once already without naming it: #20 declined PeeringDB's
`netixlan` records because *"treating [them] as owned `Address` subjects would put third-party IXP
infrastructure inside an operator's estate."* IXP peering-LAN addresses and carried customer prefixes
fail identically — both are routing facts about the path, offered as facts about the holder.

### An announcement is a configuration the operator authored

Every argument the corpus has ever made for a proposer is that the estate exceeds memory: *"Mythic
Beasts, a small UK hosting company, holds 19 network objects … Nobody types that list from memory
correctly"*; *"subsidiaries are exactly where forgotten exposure lives."* Registry rows earn that
because they are paperwork filed years ago by people who have left.

**A BGP announcement is not paperwork. It is a live router configuration somebody in the operator's
own organisation maintains this week.** The map's Note is explicit about what to do with facts of
that kind: *"The operator-supplied zone file is the product's spine. A defensive tool can ask for
ground truth an offensive recon script must guess at. Every guess-based technique is gap-filling for
what the zone doesn't cover."* The BGP leg is a guess-based technique reconstructing a declaration
the operator's organisation already holds — and reconstructing it **badly**, with limb 2 mixed
inseparably in. An operator who originates an announcement can declare the address scope.

### A proposer's cost is paid in confirmations, not in HTTP requests

*"Clean and cheap"* prices the request. The request is not the cost.
[ADR-0022](./0022-confirmation-is-singular.md) makes confirmation **singular** — one scope at a time,
deliberately, because confirming and declining fail in opposite directions. So a proposer's real
price is the length of the queue it puts in front of the operator.

**[measured]**, from #20's own figures:

| Organisation | Registry leg proposes | BGP leg adds | New territory |
| --- | --- | --- | --- |
| Safaricom (AS33771) | 10 IPv4 blocks | **100** announced prefixes, every one a subset of those 10 | **0** |
| Cloudflare (AS13335) | — | **5,336** announced prefixes | not measured |
| Mythic Beasts (4 ASNs) | 19 network objects | 17 announced prefixes | 2, of limb-2 shape |

An order-of-magnitude multiplication of a singular-confirmation queue, for zero territory in the one
case where both sides were counted, is not cheap. It is the most expensive thing a proposer can do,
because the resource it spends is the operator's attention at the one act in the product where the
probing gate actually opens.

### RouteViews as an instrument: cleared, and it does not arise

Recorded so that no later session re-derives it. **RouteViews clears ADR-0003 entirely.** CC BY 4.0
permits automated querying, local storage and cross-run retention (limb 1); the commercial clause
triggers only *"when selling services, products, reports, or other derivative works based on
RouteViews Data to third parties"*, which is the reseller shape ADR-0003 says does not fail (limb 2).
Its whole shipping obligation is attribution. Had the leg shipped, RouteViews would be its
instrument, `unencumbered`, and it would displace **nothing** — RIPEstat's `announced-prefixes` is
not in the shipped set and has not been since #19.

**The instrument was never the problem, and a clean licence is not a reason to ship.** ADR-0065
refuses an exclusion resting on a fact about our own artefact rather than about the candidate; this
is the same move in the admitting direction — *it is permitted and it is cheap* are facts about the
terms and the wire, not about what the operator learns.

## Rationale

### Why this is a capability ruling with an evidence-shaped residue

The map's out-of-scope convention wants the honest ground named, and this ruling has two.

**Limbs 1 and 3 are capability.** The instrument structurally adds nothing there, at any coverage
percentage and after any measurement: a more-specific of a confirmed scope is already enumerated, and
a subsidiary's registration is reached by a name search or by nothing. **Limb 2 is wrongness** — the
rows are real and they are not the operator's.

**The residue is evidence, and it is the reopening condition.** There is one shape the decomposition
does not obviously exhaust: **a provider-aggregatable assignment announced by the organisation that
uses it and registered under its upstream LIR's handle**. Such a prefix is the operator's custody,
sits outside every row their own opaque-id groups, and outside ARIN — where
[#39](https://github.com/winniel123/verge-asm/issues/39)'s SWIP `C…` customer objects reach it down
to a /29 — may be reachable by no path but this one.

Nobody has measured how often that happens. Both of #20's comparisons are large-ish holders with
their own PI space; neither is that case, and n=2 with a 0 and an 18% is not a rate. What bounds the
exposure is #26: reaching it at all requires the operator to hold their **own ASN**, which is at most
57% of a registered population that is itself under 1% of the persona — and an organisation running
its own AS over borrowed address space is the one operator most certain to know which prefixes it
originates.

Ruled anyway, on asymmetry of cost, and the direction is the opposite of
[#43](https://github.com/winniel123/verge-asm/issues/43)'s. Being wrong this way leaves a
sub-fraction of a sub-percent able to declare a scope they configured themselves. Being wrong the
other way ships a queue of a third party's prefixes into the one act that opens the probing gate,
on every install, forever.

### The two questions in the title, in order

**Is the BGP leg worth having at all?** No, on the decomposition above.

**Does RouteViews join the default set, and what does it displace?** The question does not arise, and
its second half was already malformed when it was written. It presupposes RIPEstat's
`announced-prefixes` is in the shipped set. #19 shipped RIPEstat off; #47 scoped its toggle to
org→prefix. There has been nothing to displace since 2026-08-13, and #20's *"replaces RIPEstat's
`announced-prefixes` outright"* — true as a statement about capability — has been read three times
since as a statement about the shipped set.

### What this does not touch

- **RouteViews is not blacklisted.** No verdict about its terms, availability or quality is disturbed.
  What is ruled out is the **leg**, and the instrument is out with it only because it serves nothing
  else.
- **CAIDA's `as-org2info` is unaffected.** #43 ships it for the org→prefix path; its ASN column is
  not what carried that ruling.
- **The delegated-stats sibling expansion is unaffected**, including the fact that a held row's
  opaque-id groups the holder's **ASNs** alongside their prefixes. That grouping is registry data
  about a holder and stays exactly as useful as #27 found it; nothing here reads it as a route.

## Consequences

- **No `Source` is added or removed, and no aperture input moves.** A proposer widens no aperture
  ([ADR-0012](./0012-a-proposer-is-not-a-source.md), ADR-0003's third amendment), so declining one
  yields no `revealed`, no `Break` and no version bump. **Aperture inputs stay at seven.**
- **`CONTEXT.md`'s `Custody` entry gains one clause** — the party who announces a route is not the
  party who controls the listener. `Custody` is the site a future session looks for this, and
  [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) sends them
  there.
- **`Proposal`'s record kinds stay at two** — an RIR delegation, or a compelled reassignment written
  by an upstream provider. A route announcement would have been a third; it is not one.
- **Four sites that still specify the BGP leg in the present tense are withdrawn in place**, per
  ADR-0058: [`passive-discovery-sources.md`](../research/passive-discovery-sources.md) §1 and §4.5,
  [`non-arin-prefix-coverage.md`](../research/non-arin-prefix-coverage.md) §1, §6.1, §10.2 and §11
  findings 9–10, and this ADR's own companion at
  [ADR-0003](./0003-third-party-source-consent-bar.md), whose third amendment names *"RIPEstat's BGP
  leg"* as the live candidate for a proposer becoming a `Source`. A sixth amendment there withdraws
  the candidate and keeps the rule.
- **The rule reaches every future routing-derived source**, and there are several already refused
  piecemeal: RIPE RIS / `riswhois` (#20, on terms), IRR/RADB (#20, no org entry point), bgp.tools
  (ADR-0003, availability), PeeringDB `netixlan` (#20, on this rule unnamed). Any of them returning
  under better terms is answered here rather than re-argued: **the terms were never what was wrong
  with them.**
- **A general test for admitting a proposer, stated because #43 established the positive half and
  nothing stated the negative.** A proposer earns v1 by naming the capability it **uniquely** reaches
  — #43's *sibling expansion cannot cross an opaque-id boundary* — and a proposer whose difference
  set decomposes into rows other paths already reach, plus rows that are somebody else's, has named
  none.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Ship RouteViews in the default set on #20's recommendation** — it is `unencumbered`, keyless, and its rate limit cannot be turned into an error | The recommendation prices the request and not the confirmations, and it rests on a yield figure measured on a hosting company whose own description names the error mode. Clean and cheap is a fact about the wire and the terms; it is not a capability. **This is the option that lost, and it lost on the decomposition rather than on the price** |
| **Ship it, but only where the registry leg is thin** — RIPE and LACNIC, which have no keyless org→prefix path at all | The defect is not regional. Limb 2 is limb 2 in every region, and the ASN the chain needs comes from the same registry data the leg was meant to compensate for. It would also make the thinnest regions the ones served by the least trustworthy rows |
| **Ship it as `operator-accepted`, off by default** | There is nothing for `consent` to gate. `consent` names the door ([ADR-0023](./0023-consent-names-the-door.md)) and CC BY 4.0's door is open; a value chosen to express *we are not sure this is useful* would be a second property wearing an enum case's clothes, which ADR-0003's second amendment already refused for a different case |
| **Defer it on the aperture ground — *deferrable with no rework*** ([ADR-0065](./0065-a-rule-is-excluded-by-its-fact-or-by-its-aperture-never-by-the-shape-of-the-set.md)) | Tempting, because a proposer genuinely is free to add or drop in any release. But the aperture ground says *the rule is admissible and the observation does not exist yet*, and here the observation exists and is retrievable today. Recording it as deferred would hide a **capability** verdict behind a schedule, which is the rot ADR-0065 exists to prevent |
| **Ship it as a decomposition instrument** — a registered `/16` exceeds the 1,024-address range cap and cannot be confirmed at the shipped default, while announced `/24`s fit | A real interaction and a real hole, but it is a hole in the `Proposal`-to-`Seed` path and not an argument for this source. A routing decomposition is the network engineer's, not the estate's: an announced more-specific asserts nothing about where listeners are, and a registered block may hold listeners outside every more-specific it announces. Surfaced as fog on its own account rather than used to rescue this |
| **Keep the leg and filter limb 2** — drop announced prefixes registered to a third party | The filter is the registry lookup, so the filtered output is limb 1 ∪ limb 3, which is what the registry leg and the org-name box already return. The leg survives only by becoming the thing it was supposed to improve on |
| **Say nothing and leave the fog patch open** | The patch has been open since #3, and three documents have gone on recommending the leg in the present tense the whole time. An unruled question that four decisions have quietly answered is exactly the shape ADR-0058 was minted for |
