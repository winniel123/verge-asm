# ADR-0019: The probing gate is total over an `Address` — authority is the objection, so no port opens it

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#118 What does an install that is honest about holding custody of nothing see?](https://github.com/winniel123/verge-asm/issues/118)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)
- **Amends:** [ADR-0002](./0002-ownership-gates-probing.md), [ADR-0014](./0014-only-revealed-generalises.md)

## Context

[ADR-0002](./0002-ownership-gates-probing.md)'s Decision table has three rows. Two of them have
since been rewritten — `unknown` was deleted by [#40](https://github.com/winniel123/verge-asm/issues/40)
because *"the sentence above has no producer"*, and the `owned` row was extended by
[#81](https://github.com/winniel123/verge-asm/issues/81). The middle row has never been touched:

| Ownership | Probing permitted |
| --- | --- |
| `third-party` | Only the ports the `Name` implies (443, 80), or nothing pending explicit operator opt-in |

Read alone and in the present tense, that row says a `third-party` address **is connected to**, on
80 and 443, without any operator act. It is the only place in the repository that says so, and
everything written since says the opposite:

- **ADR-0002's own rejected-alternatives table**, four screens further down: *"Probe `third-party`
  addresses at a gentler rate — Rate is not the objection, authority is. **A slow scan of a
  stranger's host is still a scan of a stranger's host.**"* A two-port scan of a stranger's host is
  the same sentence with a different quantifier.
- **[ADR-0013](./0013-custody-is-control-and-extends-by-declaration.md)** built the row's *other*
  limb. *"Nothing pending explicit operator opt-in"* is the `custody extension`, and it shipped. A
  disjunction one of whose limbs has been built and named is not still a disjunction.
- **[#51](https://github.com/winniel123/verge-asm/issues/51)'s declaration copy**, which ADR-0013 §7
  makes load-bearing for a safety property, tells the operator in their own words: *"Without it,
  those addresses are treated as somebody else's and **nothing connects to them**."* If 80 and 443
  are probed, that sentence is a false statement made on the one surface that carries the hazard.
- **`CONTEXT.md`'s `Custody`** says it *"governs what may be probed"* with no port qualifier
  anywhere, and its two values are total over the address.

The row also names an object the model cannot produce. *The ports the `Name` implies* has no
mechanism behind it: nothing in the model derives a port from a name, and building one would mean a
table mapping names or schemes to ports — an **eighth aperture input**, against the seven
[ADR-0014](./0014-only-revealed-generalises.md) enumerates and the map counts. That is the same
disposal `unknown` got and for the same reason.

[#118](https://github.com/winniel123/verge-asm/issues/118) forced it, and forced it with a live
casualty rather than an argument. [#44](https://github.com/winniel123/verge-asm/issues/44)'s
prototype draws the modal cloud-resident install — the install this whole thread exists for — with
`certificate-expiring` holding **616** subjects and `plaintext-http-no-https` holding **409**, on
the copy *"only the ports your names imply are probed there — 80 and 443"*. That is
[ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)'s test
answered by demonstration: the superseded sentence, read alone and in the present tense, caused a
competent session to build the thing.

## Decision

**The probing gate is total over an `Address`. A `third-party` address is not connected to on any
port, by any tier, at any rate. `Custody` is the only input that opens it, and there is no port-,
protocol- or rate-shaped carve-out that opens it partially.**

| Concern | Decision |
| --- | --- |
| ADR-0002's `third-party` row, limb 1 — *"only the ports the `Name` implies (443, 80)"* | **Withdrawn at the site that specifies it.** It names no object the model can produce, it is refuted by its own document's alternatives table, and it contradicts the operator-facing copy that carries ADR-0013 §7's hazard |
| ADR-0002's `third-party` row, limb 2 — *"or nothing pending explicit operator opt-in"* | **Kept, and it is the whole row.** The opt-in is the `custody extension`, built by ADR-0013 |
| Does a public name resolving to an address license a connect to it? | **No.** The name being published is an invitation to *its* service; it is not the address holder's consent to be measured, and we cannot tell which of the two we are talking to |
| Is a narrower probe a lesser act? | **No.** Volume, rate and port count are all quantifiers over one act whose objection is authority. ADR-0002 already ruled this for rate; this generalises the ruling it made |
| What an install holding custody of nothing measures | `resolution` and `dns-record`, at full aperture, on every `Name` in the estate. **A query is not a connect** — the gate is over an `Address`, and DNS is asked of an authority about a name |
| What it does **not** measure | `reachability`, and therefore `certificate`, `http-identity` and `tls-acceptance`, which all ride or follow that exchange. Four facets of six |
| The `Service` population of such an install | **Empty.** A `Service` enters by being observed, and nothing observes it |
| Which v1 rules such an install can run | Exactly the rules whose evidence is a DNS answer. The rest have an **empty `Predicate domain`** and render as a no-population panel, never as a census of zeroes |

### The cost, stated rather than hidden

**The modal operator gets a DNS-only product until they make a custody claim.**
[#26](https://github.com/winniel123/verge-asm/issues/26) established the cloud-resident operator as
the modal case, so this is the common path and not an edge. Before this ADR, a reader of ADR-0002's
table could believe that operator was getting the certificate rules and the plaintext rule for
free. They are not, and they never were — no code has changed, only the sentence describing it.

That cost is **already priced and already paid**, twice. ADR-0013's Consequences: *"An operator on
shared hosting loses nothing by leaving it off, and gains a false assertion by turning it on. That
is the intended shape."* ADR-0002's Consequences: *"Coverage is deliberately narrower than a tool
that scans everything it reaches. An operator whose estate is mostly SaaS-fronted sees fewer ports.
That is the intended trade."* This ADR does not choose the trade. It stops the repository from
describing it two ways.

## Rationale

### Limb 1 fails the only test that matters here — it cannot be built without a new aperture input

The strongest case for keeping limb 1 is not comfort, it is that 80 and 443 are **what the name is
for**. A public name resolving to an address is a standing invitation for every browser on earth to
connect there on those ports, and a TCP connect from us is indistinguishable from one of them. On
that reading, limb 1 is not a compromise on authority at all — it is the observation that authority
was already granted by publication.

It is a real argument and it loses on two counts.

**It grants the wrong party's consent.** Publication is the *name holder's* act. The gate turns on
whether we may probe the **listener**, and on shared hosting the listener is somebody else's — one
address serving the operator's site and four other tenants'. `Custody` was renamed from `Ownership`
by ADR-0013 for exactly this: *the party who can consent to a scan is whoever controls the
listener*. A name the operator published cannot consent on the address holder's behalf, and the
whole reason the modal operator declines the extension is that those two parties are different
people.

**It is unbuildable as stated.** *The ports the `Name` implies* is not a measurement, a declaration
or a derivation — it is a table, and a table that decides *where to look* is aperture
([#31](https://github.com/winniel123/verge-asm/issues/31), ADR-0008). Adding it means an eighth
aperture input, a dimension on every `Batch`'s recorded scope, and a `revealed` opening whenever it
moves. None of that exists, and nobody has ever proposed it. A clause that cannot be implemented
without machinery nobody has costed is not a live decision. It is a sentence that outlived its
draft.

### The two limbs were never alternatives — one of them shipped

The row's `or` is the tell. It was written on 2026-08-01, before `Custody`, before the `custody
extension`, and before [#26](https://github.com/winniel123/verge-asm/issues/26) established who the
modal operator is. It offered a choice between a narrow default and an opt-in, and left it open
because the opt-in did not exist yet.

ADR-0013 built the opt-in. Once one limb of an unresolved disjunction is specified, named, given
operator-facing copy and made the carrier of a safety property, the disjunction has resolved — and
the other limb is not a surviving fallback, it is the road not taken still signposted. This is
[ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)'s
population, one row wide.

### Two facets of six is not a consolation prize

The reading that makes this ADR look severe is that the honest install *sees nothing*. It does not.

`resolution` is the facet that puts every address in the estate in the first place, and the facet
that withdraws them. `dns-record` runs seven qtypes on every name. Between them they carry the
whole `Name` side of the estate, every membership event on the two subjects that can be a root
([ADR-0031](./0031-membership-alerts-at-the-root-of-the-entering-subtree.md)), and the rules whose
evidence is a DNS answer. ADR-0002's own Consequences said so and then deferred the drawing of it:
*"For a `third-party` one, the operator's exposure belongs to the `Name` — this name is served from
infrastructure you do not control — and it is available without scanning a stranger."*

What that sentence must **not** be read as is a licence to mint a rule. See the amendment to
ADR-0002 below.

### Why this is an ADR and not a ticket-local IA call

[#10](https://github.com/winniel123/verge-asm/issues/10),
[#28](https://github.com/winniel123/verge-asm/issues/28),
[#44](https://github.com/winniel123/verge-asm/issues/44) and
[#51](https://github.com/winniel123/verge-asm/issues/51) all declined ADRs on the ground that IA
decisions live in the ticket, and #118's rendering half follows them. This half is not rendering.
It changes what the measurement binary is permitted to put on the wire against an address the
operator has not claimed, which is the founding safety decision of the project, and it withdraws a
row of the ADR that made it. That belongs in `docs/adr/`.

## Consequences

- **[ADR-0002](./0002-ownership-gates-probing.md) is amended in two places** — its Decision table's
  `third-party` row loses limb 1, and its *"this is a finding worth surfacing in its own right"*
  consequence is qualified (below).
- **[ADR-0014](./0014-only-revealed-generalises.md)'s deferred rendering cost is discharged** by
  #118 rather than merely restated. Its *"every surface says this is not yours beside the aged
  value"* now has a drawn surface and a stated altitude rule.
- **[`CONTEXT.md`](../../CONTEXT.md)'s `Custody` entry gains one clause** — the gate is total over
  the address, and the retained value is read by rules until it ages.
- **Nothing in the measurement binary changes.** No leaf's output moves, no `Derivation` version
  moves, no `Break` is written. The clause was never implemented. This ADR retires the sentence,
  not a behaviour. Concretely: no golden-corpus row moves, which is the bidirectional gate in
  [ADR-0008](./0008-derivation-versions-move-on-content.md) confirming that this is a documentation
  repair and not an output-affecting change.
- **The aperture input count stays at seven.** Limb 1 would have made an eighth. Refusing it is
  what keeps the map's live absolute correct.
- **[#44](https://github.com/winniel123/verge-asm/issues/44)'s `?fill=modal` prototype overstates
  the honest install** by two populated rules — `certificate-expiring` at 616 subjects and
  `plaintext-http-no-https` at 409 — and by its copy *"only the ports your names imply are probed
  there — 80 and 443"*. #44's **resolution** is unaffected: its table already reads *"`third-party`
  address, sensitive ports never probed — timeline never existed, subject no"*, which is this ADR.
  The defect is prototype-local and is recorded rather than silently rewritten, a closed ticket's
  artefact not being another ticket's to re-fill.

## Amendment to ADR-0002's *"a finding worth surfacing in its own right"*

ADR-0002's Consequences say that for a `third-party` address the operator's exposure belongs to the
`Name`, and that *"this is a finding worth surfacing in its own right"*. Read alone and in the
present tense that sentence would cause a competent session to mint a rule — and it uses the word
`CONTEXT.md` refuses.

**It is surfaced as a census and never as a `Signal`.** A rule reading *this name is served from
infrastructure you do not control* would have a `Predicate domain` of every `Name` in the estate,
fire on ~100% of the modal install permanently, and read a **Declared act** rather than a
measurement — the operator's own configuration handed back to them as a standing defect. Being true
of most of the estate is not disqualifying on its own
([ADR-0065](./0065-a-rule-is-excluded-by-its-fact-or-by-its-aperture-never-by-the-shape-of-the-set.md)),
and the exclusion here is not the shape of the set: it is that the fact is **Declared**, so the rule
would have no evidence to cite and no transition anybody could act on. A `Signal` is *"a named,
versioned rule evaluated over observations"*, and there is no observation here.

The fact is real and is rendered where the estate is enumerated: an `Address` carries its `Custody`
and its `Citation`, and `Coverage` carries one standing line. That is ADR-0002's intent, in the
vocabulary the project settled on afterwards.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| Keep limb 1 — probe 80 and 443 on `third-party` addresses | Grants the *name holder's* consent for the *listener holder's* equipment, which is the exact distinction `Ownership` → `Custody` was renamed to draw. Needs an eighth aperture input nobody has costed. And it makes #51's shipped declaration copy — *"nothing connects to them"* — false on the surface ADR-0013 §7 makes load-bearing |
| Keep limb 1 but make it operator-configurable | An operator dial deciding what goes on the wire against a stranger's host. The one party the setting affects is the one who cannot see it |
| Keep limb 1 for `443/tcp` only, on the ground that TLS is the invitation | The same argument with a smaller number, and the smaller number is ADR-0002's *"gentler rate"* row in different clothes |
| Withdraw the row silently, since nothing implements it | The clause has already produced a wrong prototype once. ADR-0058 exists because an unwithdrawn sentence is a live instruction, and *nothing implements it* is exactly the state in which it is most dangerous |
| Fold this into #118's ticket comment with no ADR | Follows #10/#28/#44/#51's precedent, and that precedent is about **IA**. This retires a row of the project's founding safety decision |
| Mint `name-served-from-third-party-infrastructure` as a v1 `Signal` | Reads a Declared act, cites no observation, fires on ~100% of the modal estate permanently, and offers no act. A permanent bill of ill health is the same defect as the clean bill ADR-0004 refuses, with the sign flipped |
