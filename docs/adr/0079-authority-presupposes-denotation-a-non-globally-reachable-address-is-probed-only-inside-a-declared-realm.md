# ADR-0079: Authority presupposes denotation — a non-globally-reachable address is one subject per realm, so it is probed only inside a declared one

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#137 Does the probing gate open on a non-globally-reachable address, and what does an internet-class prober do with it?](https://github.com/winniel123/verge-asm/issues/137)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)
- **Amends:** [ADR-0002](./0002-ownership-gates-probing.md), [ADR-0013](./0013-custody-is-control-and-extends-by-declaration.md), [ADR-0019](./0019-the-probing-gate-is-total-over-an-address.md)

## Context

[ADR-0019](./0019-the-probing-gate-is-total-over-an-address.md) made the probing gate **total over
an `Address`** and named the objection: **authority**. No port, protocol or rate opens it partially,
and `Custody` is the only input that opens it at all. That settled *whose equipment may we connect
to*.

[#128](https://github.com/winniel123/verge-asm/issues/128) then admitted
`non-globally-reachable-address-resolved-from-internet` and, in doing so, made a population visible
that predates it. A `Name` in a custody-extending scope resolves to `10.0.0.5`. The address is
cited, so it is a subject (`CONTEXT.md`, `Address`). The chain never left the declared zone, so
[ADR-0013](./0013-custody-is-control-and-extends-by-declaration.md) §3's extension covers it and
`Custody` derives **`operator`**. ADR-0019's gate is total, so it opens **wholly**. The prober is
then asked to connect to `10.0.0.5`.

**`10.0.0.5` is not a machine. It is a machine per network.** The prober connects to whichever one
is on *its* network — the operator's if it happens to sit there, the hosting provider's if the
instance runs on a VPS, another tenant's if the provider shares private space. Every safety
sentence the project has written is about *whose* listener we may touch; none of them noticed that
this address does not name a listener until somebody says which network to ask on.

**Three corrections to the framing this ticket inherited**, all of which change where the fix can
be sited.

**There are two routes in, not one.** #128's hand-off names the `custody extension`. A declared
**address scope** over `10.0.0.0/24` is the other, and it is not the rarer of the two: under
[ADR-0047](./0047-an-address-scope-is-its-own-enumeration.md) every address inside a declared CIDR
is a subject **from the declaration**, walked every cadence, whether or not anything resolves to
it. An operator declaring their office LAN to get an internal leg is doing the ordinary thing. So a
repair sited in the extension alone leaves the other half open.

**#128's firing set is not the exposed population**, in either direction. It is *narrower* — it
never sees an address that arrived from an address-scope `Seed`, because no name resolved to it.
And it is *absent* exactly where the hazard is worst: the rule's `Predicate domain` is
**internet-class** `resolution`, so on an install with no internet-class vantage — the modal
one-class install, and the one most likely to be a small VPS — the rule is permanently
`not-evaluable` while the gate is wide open and the prober is asked. **A signal that goes dark on
the estates carrying the hazard cannot be the carrier of the safety property.**

**The exposure is not only who we touch. It is what we then say about the operator.** The internet
`Reach` leg over a non-globally-reachable address is not a measurement we are missing; it is one
that **cannot exist**, because there is no internet path to that address by construction. A
connect from an internet-class vantage does not measure it weakly — it measures a **different
subject** and files the answer on the flagship timeline. If the stranger's machine answers on 22,
the internet leg reads `reached`, `Exposure` composes to `exposed`, and the product fires *the move
it exists to catch* about a service that is not internet-reachable at all.

`Vantage class`'s standing trade — *"a false `exposed` is investigated, a false quiet reading is
not"* — does **not** rescue this. That trade prices a wrong **verdict** about the right subject. This
is the right verdict about the wrong subject, and an operator investigating it finds their own
machine innocent and learns nothing, because the machine that answered was never theirs to look at.

## Decision

**Authority is the right test and it is not the first one. The probing gate asks *may we connect to
this address*, which presupposes that the address denotes one machine; a non-globally-reachable
address denotes one per realm, so a denotation precondition runs in front of the gate. Such an
address is connected to only where a declared address scope covers it and only from a `Vantage`
that is not `internet`-class.**

| Concern | Decision |
| --- | --- |
| Which addresses are in the population | The ones the reading rule in [`special-purpose-address-registry.md`](../research/special-purpose-address-registry.md) §2 already fixes: **most specific registered block, `Globally Reachable` = `False`**. The same read, by the same rule, for both consumers — §8 of that note records the second one |
| Does `Custody` change | **No.** The derivation is untouched and such an address still derives `operator` under an extension or a scope. `operator` becomes **necessary and not sufficient**; the precondition is not a `Custody` value and mints no third one |
| Is ADR-0019's totality weakened | **No.** ADR-0019 forbids a carve-out that opens the gate **partially**. This never opens anything the gate closes; it closes further, in front. A question with no subject does not reach the gate to be answered |
| Route 1 — a `custody extension` | **Does not open the gate over such an address.** ADR-0013 §6 already refuses to let an extension decide which realm the **prober** is in; this is the same refusal about the **address** |
| Route 2 — a declared **address scope** | **Opens it**, unchanged. A literal CIDR is the operator's own realm claim, and it is the only realm claim the model has |
| Which vantage may be asked | Any that is **not `internet`-class**. An `internet`-class `Vantage` is one whose presented address no address scope covers — by the class's own definition it is outside every realm the operator declared, so it is never in the same realm as such an address |
| What the internet `Reach` leg holds | **Nothing at all** where it never began, and a `Gap` where it was running — `Reach`'s existing disposal, reached by the address being absent from that class's `Batch` recorded scope **by content**. `Exposure` is then a one-legged reading and gets no name, per [ADR-0017](./0017-exposure-needs-both-legs.md) |
| `resolution` and `dns-record` | **Untouched, at full aperture.** A query is not a connect (ADR-0019). The address is still admitted, still cited, still a subject, and #128's rule still fires on the disclosure |
| A new `Signal` | **No.** Refused below |
| A new aperture input | **No.** The custody gate is already a named dimension of the `Batch` scope record ([ADR-0014](./0014-only-revealed-generalises.md)); this changes that dimension's condition rather than standing beside it. The count stays at **seven** |
| A new notification cause, class or coverage member | **None.** The fact is rendered on the `Address`, beside the `Custody` and `Citation` ADR-0019 already put there |

### Worked, because the cases are what carry it

| Case | Today | After |
| --- | --- | --- |
| Name in an extending scope resolves to `10.0.0.5`; no address scope | `operator`, probed from every vantage | Not connected to from any vantage. Still a subject; #128 still fires |
| Operator declares `10.0.0.0/24`; instance is the only prober | Probed | **Probed, unchanged.** The internal estate keeps its measurements |
| Operator declares `10.0.0.0/24`; an external prober also runs | Both probe it; the external one measures a stranger | The internal one probes; the external one is never asked |
| Name resolves to `127.0.0.1` or `169.254.169.254` under an extension | Probed — the prober measures **itself**, and on `169.254.169.254` retrieves cloud instance metadata into `http-identity` | Not connected to |
| Name resolves to a public AWS address under an extension | Probed | **Probed, unchanged.** The extension's motivating case ([#26](https://github.com/winniel123/verge-asm/issues/26)) is entirely globally-reachable space and is untouched |
| Name resolves to a `2002::` or `2001::/32` address | Probed | **Probed, unchanged** — `N/A`, so outside the population. Both blocks embed a globally unique IPv4 address, so they denote one machine (§8 of the note) |

## Rationale

### The failing test is denotation, and that is why ADR-0019 survives intact

The tempting reading of this ticket is that ADR-0019 was too strong or too weak — that a total gate
either over-permits here, or that *authority* needs a rate-shaped or vantage-shaped exception after
all. Both readings are wrong, and the second is the dangerous one, because ADR-0019 spent its whole
length refusing exactly that shape.

`Custody` answers **whose listener is it**. That question has an answer only once *a listener* has
been identified, and an address is how the model identifies one. For every globally-reachable
address that identification is free: the octets denote one interface on earth. For a
non-globally-reachable address it is not free and the model never bought it — the octets denote one
interface *per realm*, and nothing in the model names a realm.

So the two tests are in series and they are asking different things: **denotation asks *of what?*,
authority asks *may we?*.** A gate is not weakened by a question that never reaches it. Nor is the
gate's totality touched: after this ADR, a `third-party` address is still connected to on no port at
no rate from nowhere, and an `operator` address that denotes one machine is still connected to
wholly.

This is also why the precondition is **not** a third `Custody` value. `unknown` was deleted by
ADR-0013 §2 because nothing could produce it; reintroducing an indeterminate custody state here
would be worse than that — it would be a *determinate* fact (the operator really does control
`10.0.0.5` on their network) recorded under a word that says we could not tell.

### The instrument is not new — ADR-0013 §6 already refuses to let an extension decide a realm

This is the load-bearing argument, and it is that the project already made this exact call once,
about the other party.

> **§6. `Vantage class` reads literal address-scope `Seed`s only, never the extension.** *"A prober
> in the operator's VPC must not verify `internal` in one batch and `internet` in the next because
> **a DNS answer changed**."*

That is a rule about **which realm a thing is in**, and it says a `custody extension` may not
settle it, because an extension is a set the world recomputes rather than a boundary the operator
drew. §6 applied it to the prober and stopped there — nobody had asked the same question about the
target, because until #128 nobody had noticed that a target can be realm-relative too.

Applying it to the address is not a new instrument, a new key, a new lookup or a new table. It is
one sentence's worth of generalisation, and §6's own closing line invited it: *"This is the one
place where two things both fairly called the operator's addresses deliberately mean different
sets. It is stated here because a future session will otherwise unify them."* This session did not
unify them. It found a third thing the same distinction governs.

The asymmetry that remains once you see it is indefensible: today an extension may not tell us
which side of the boundary our **own prober** sits on, but may tell us to send that prober packets
addressed to a realm nobody declared.

### The population is the owner's column and nothing finer, because anything finer is ours

The obvious better fix is a **realm taxonomy**: `127.0.0.0/8` and `::1` are host-scoped and denote
the prober itself; `fe80::/10` is link-scoped; `10/8`, `172.16/12`, `192.168/16`, `100.64/10` and
`fc00::/7` are realm-scoped; the three TEST-NETs, `2001:db8::/32`, `3fff::/20`, `240.0.0.0/4`,
`100::/64` and `255.255.255.255/32` denote no machine at all. Each class wants a different
disposal, and a rule cut that way would let an internal prober keep probing `10.0.0.0/24` while
never touching loopback.

**That taxonomy would be ours, and it is refused for the reason #128 refused *RFC 1918 / 6598 /
4193*:** it is a selection with no owner. The registries publish exactly one cut — a binary
`Globally Reachable` column — and hand-sorting its 32 `False` rows into three or four realm kinds
is authorship of the precise kind
[ADR-0071](./0071-a-vantage-scoped-claim-is-read-only-at-the-vantage-that-scopes-it.md) draws the
line against, sited inside the safety path where the project has refused invented structure since
[#27](https://github.com/winniel123/verge-asm/issues/27)'s size cap and
[#31](https://github.com/winniel123/verge-asm/issues/31)'s exclusion list.

So the binary column supplies the population, and the **operator's declaration** supplies what the
column cannot: which realm. Between them there is nothing left for us to invent — and the
host-scoped cases the taxonomy was wanted for are covered anyway, because nobody's extension
declares `127.0.0.1`'s realm either.

### Why a declared address scope is a realm claim and an extension is not

Both are Declared acts, so the distinction has to be earned rather than asserted.

A **declared address scope** is a literal CIDR the operator typed. Typing `10.0.0.0/24` is only
meaningful as a statement about *their own* network — nobody types a private CIDR to describe
somebody else's — and it is checked at declaration against
[ADR-0049](./0049-an-address-scope-is-family-agnostic-and-the-cap-counts-addresses.md)'s cap, so it
is bounded, visible and cannot fail silently. It is the same escape hatch ADR-0002's Consequences
have named since day one: *"the fix is the operator declaring an address scope — which is a `Seed`,
so the mechanism already exists."*

A **`custody extension`** is one tick on a name scope, about a set that is **recomputed forever** by
whoever administers the operator's DNS. ADR-0013 §7 already records what that costs when the
recomputation goes somewhere the operator did not intend — the apex-flattening hole, *"the common
configuration, not an edge"*. In globally-reachable space that hole is bounded: the worst case is
that we probe a CDN edge, which is wrong but is at least the machine the record names. In
non-globally-reachable space there is no such floor, because the record does not name a machine.
**An extension's failure mode over private space is not over-reach, it is mis-reference**, and
ADR-0013's self-correction property — *"when the name stops resolving there, the address leaves on
its own"* — has no purchase on it: the address stays perfectly resolvable and stays the wrong
machine.

### No eighth aperture input, and no second reading of the table

Two counts that a session could reasonably fear this ADR moves, both checked.

**Aperture inputs stay at seven.** ADR-0014 fixed the criterion — *"aperture is what a `Batch`
records as its completed scope"* — and enumerated **the custody gate** as one dimension of that
record. This ADR changes what that dimension admits; it does not add a dimension beside it. Nothing
new needs recording on a `Batch`, because the scope record is *by content* — an address we did not
ask about is absent from it already. When the registry moves a block from `False` to `True` in some
future release, addresses inside it become probeable and their `Service` timelines **open**, which
is `revealed` on the input that already owns that behaviour. ADR-0019 refused limb 1 partly to keep
this count at seven; that refusal was about a **name → port** table, a dimension the gate does not
have. This is the dimension the gate already is.

**The table gains a consumer, not a second reading.** #128 asked which of
[#31](https://github.com/winniel123/verge-asm/issues/31)'s two kinds the registry is and answered
*not aperture — the table does not decide where to look*. **That answer is true of the rule and
false of the gate**, and the honest statement is that the note is now a **verdict** table in one
consumer and an **aperture** table in the other. That is not a defect and not a precedent: it is
`sensitive-ports.md`'s shape exactly, whose 38 pairs both feed a claim to
`sensitive-port-reached-from-internet` and enter `verge-core`, which decides where to look
([#29](https://github.com/winniel123/verge-asm/issues/29)). What matters is that both consumers
read the **same column by the same reading rule**, so there is no seam where two readings could
drift apart ([#6](https://github.com/winniel123/verge-asm/issues/6)). Recorded at the note, in §8.

### The `N/A` cells: the same read, and the residues run opposite ways for one reason

The note's §2 refuses to read the four `N/A` and terminated cells as `False`, because supplying a
value the owner declined to supply is authorship. §6 discloses the residue: it runs **toward
silence**, so a `2002::` address published in public DNS does not fire.

The tempting move for a safety consumer is to flip that — read `N/A` as *not globally reachable*
and refuse to probe, on the ground that the closed direction is the safe one and the objection to
authorship is about publishing a claim rather than about declining to send a packet. **It is
refused**, and not on tidiness.

It is refused because it is **not** the closed direction, once you ask what the closure is *for*.
The hazard is reaching a **different machine**, and all four blocks are immune to it: `2002::/16`
and `2001::/32` embed a globally unique IPv4 address, so a 6to4 or Teredo address denotes exactly
one host worldwide; `192.88.99.0/24` is globally routed anycast, which the registries themselves
treat as compatible with reachability (`192.0.0.9`, `192.31.196.0/24` and the AS112 blocks all
carry `True`); and `2001:10::/28` is terminated and unrouted, so a packet to it reaches nothing at
all. Reading them as `False` would buy no safety and would cost the one thing the note is for —
one column, one reading rule, two consumers.

So the residues part cleanly and the reason is a single sentence: **the signal's residue is toward
silence because a claim we cannot found is not made; the gate's residue is toward probing because a
denotation the owner declined to deny is not in doubt.** Both are the same refusal to supply the
owner's missing value, read against two different questions.

### Why this is a gate and not a warning

The remaining live alternative is to leave the behaviour alone and tell the operator: let #128's
rule fire, put a line on `Coverage`, let them withdraw the extension if they mind.

It fails on the population, measured against the model rather than argued. #128's rule is
`not-evaluable` on every `Name` of an install with no internet-class vantage. That install is the
one whose prober is most likely to be somewhere the operator did not think about — a VPS, a
colocated box, a laptop — and it is the install where every probe of `10.0.0.5` is a probe of the
hosting provider's network with nothing anywhere in the product saying so. **The warning is absent
precisely on the estates that need it**, and a safety mechanism whose coverage is the inverse of
its hazard is not a mechanism.

It fails a second time on ADR-0002's founding sentence, which is about who runs this software:
*"verge-asm is AGPL-3.0 and self-hosted, so it is run by people we never meet, against estates we
never see, with defaults nobody reviews."* A default that connects to a stranger's private network
and is corrected only by an operator reading a line is the shape that ADR ruled against.

## The cost, stated rather than hidden

**An operator whose internal estate is reached only through a `custody extension` gets no internal
probing of it.** They get `resolution` and `dns-record` at full aperture, #128's rule, and nothing
that rides a TCP connect — which is ADR-0019's honest-install shape, arriving here for a different
reason. The repair is one they can perform: declare the ranges as address scopes.

**That repair is bounded by ADR-0049's cap, and the cap is a dial.** 1,024 addresses per scope
refuses anything above a `/22`, so an enterprise on `10.0.0.0/16` declares 64 scopes or raises the
cap, which ADR-0002's #81 amendment makes operator-configurable. ADR-0013's #81 amendment says the
model's route for a holding above the cap is *"a name scope with an extension"* — **that route is
now closed for non-globally-reachable space specifically**, and the sentence is amended below
rather than left to be re-derived.

**One residue survives and it takes two operator errors.** An operator who declares `10.0.0.0/24`
and runs the instance somewhere that is not on that network — a VPS — will probe the VPS
provider's private space. Both halves are Declared acts that are simply false, which is the class
ADR-0013's Consequences say the model *"has never tried to prevent, because it cannot prevent a
false name-scope seed either"*. It is narrower than the state this ADR replaces in every way that
matters: it needs a typed CIDR rather than a tick, it is bounded by the cap rather than by the
size of a zone, and it is visible on a surface the operator authored.

## Consequences

- **[ADR-0002](./0002-ownership-gates-probing.md)'s gate gains a precondition, not a row.** Its
  table is about `Custody` values and is unchanged. Its *"classification needs an operator escape
  hatch — and the fix is the operator declaring an address scope"* is now load-bearing for a second
  reason.
- **[ADR-0013](./0013-custody-is-control-and-extends-by-declaration.md) is amended in two places** —
  §3's transitivity gains a second stopping condition, and the #81 amendment's *"an operator whose
  holding exceeds the cap declares a name scope with an extension"* is qualified. Amendments below.
- **[ADR-0019](./0019-the-probing-gate-is-total-over-an-address.md) is untouched and is now
  load-bearing in a second direction.** Its totality is what makes this ADR necessary: because no
  narrower probe is available, the only available disposal for an address we cannot locate is not
  to connect at all.
- **[`CONTEXT.md`](../../CONTEXT.md)'s `Custody` entry gains one clause** — `operator` is necessary
  and not sufficient, and the two conditions.
- **[`special-purpose-address-registry.md`](../research/special-purpose-address-registry.md) gains
  §8** — the second consumer, the shared reading rule, and the two residues.
- **Nothing in the measurement binary changes**, because nothing implements the current behaviour.
  No golden-corpus row moves, no `Derivation` version moves, no `Break` is written — the same
  bidirectional check ADR-0019 cited under
  [ADR-0008](./0008-derivation-versions-move-on-content.md). This ADR retires a permission, not a
  behaviour.
- **The shipped configuration sends packets to strictly fewer destinations.** Every clause here
  removes a destination and none adds one.
- **The v1 rule set stays at seventeen.** No rule is added, removed or re-scoped, and #128's is
  untouched in all four of its parts.
- **`Coverage` gains no member and the notification layer gains nothing.** An address entering the
  population enters as a subject, so the membership census already carries it
  ([ADR-0031](./0031-membership-alerts-at-the-root-of-the-entering-subtree.md)).

## Amendment to ADR-0013 §3 — transitivity has a second stopping condition

§3 says *"transitivity stops where the resolution chain leaves the declared zone"*, and that is the
whole of the test today. It stops for a second reason as well: **the extension does not cover an
address whose most specific registered block reads `Globally Reachable = False`**, whether or not
the chain stayed inside the zone. `api.example.com A 10.0.0.5` is a direct A record inside the
declared zone and extends nothing.

The two stopping conditions are different in kind and both are measurements rather than lists. The
first asks *is this still the operator's zone?* and the second asks *does this answer name a
machine?* Neither is a signature database, and the second reads a table the project already
transcribed for another consumer.

`Custody` itself is unaffected: such an address covered by a declared **address scope** is
`operator`, exactly as before.

## Amendment to ADR-0013's #81 amendment — the above-cap route is closed for private space

That amendment reads: *"An operator whose holding exceeds the cap declares a **name** scope with an
extension: the gate opens over exactly the addresses their names resolve to."* For an operator
whose above-cap holding is **non-globally-reachable** — which is most operators with an above-cap
holding, since ADR-0049's cap bites at a `/22` and RFC 1918 space is where `/16`s live — that route
no longer opens the gate.

It is stated here rather than left implicit because the sentence reads forward as an instruction and
would otherwise be re-derived by the next session that needs one, which is
[#125](https://github.com/winniel123/verge-asm/issues/125)'s companion rule to
[ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md): a
withdrawal that supplies no replacement does not hold. **The replacement is address scopes plus the
cap dial**, and the trade is stated in this ADR's cost section rather than hidden in a strike.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| Leave it — `Custody` says `operator`, so probe it | The address does not name a machine, so `operator` is a true statement about a listener we have not identified. It ships a connect to a stranger's private network as a default to every self-hosted deployment, which is the sentence ADR-0002 was written to prevent |
| Warn instead of gate — let #128's rule and a `Coverage` line carry it | The rule is `not-evaluable` on every install with no internet-class vantage, which is where the hazard is worst. A safety mechanism whose coverage is the inverse of its hazard is not one |
| Derive `third-party` for such an address | False about control — the operator really does hold custody of `10.0.0.5` on their own network — and `Custody`'s whole definition is control of the listener. It would also make the extension's census lie about what it covers |
| A third `Custody` value for *cannot be located* | `unknown` was deleted because nothing could produce it; this would be worse, recording a determinate fact under a word that says we could not tell. The precondition is not a custody value |
| Bar only the `internet` class, keeping the extension route for internal probers | Closes the stranger's-network case and leaves the prober-probes-itself case wide open: `127.0.0.1` and `169.254.169.254` published in a zone are reached by an *internal* prober too, and on the second the product retrieves cloud instance metadata into `http-identity` on an instance the map calls a high-value target. Separating those cases needs a realm taxonomy, which is ours to invent and therefore refused |
| Require the class to be `internal` rather than merely not `internet` | `internal` is verifiable only where an external prober observed the instance's presented address (ADR-0013 §6, #124's rider), so this deletes internal probing of private space on every install without a prober — which is most of them. The bar is on the class we can *prove* is in the wrong realm, not on every class we have not proved is in the right one |
| A realm taxonomy — host-scoped, link-scoped, realm-scoped, denotes-nothing | The right shape and the wrong author. Hand-sorting the registry's 32 `False` rows is a selection with no owner, sited inside the safety path — #128's refusal of *RFC 1918 / 6598 / 4193* and ADR-0071's line, one level down |
| Read the four `N/A` and terminated cells as `False` for this consumer | Buys no safety — all four denote one host worldwide or nothing at all — and costs the note its single reading rule, putting a seam between two consumers of one column |
| Site the fix in the `custody extension` alone | Misses the declared-address-scope route entirely, which is how an operator's own LAN arrives and is the commoner path for private space |
| Site it in `Address` — refuse to admit a non-globally-reachable address as a subject | Blinds #128's rule, whose whole evidence is that address appearing in a public answer, and deletes the disclosure the ticket exists downstream of. The address is a perfectly good subject; what it is not is a destination |
| Mint a `Signal` for it | Reads a Declared act crossed with a table, fires on the whole population permanently, and offers the operator an act they already chose. ADR-0019's amendment disposed of the identical shape |
| Probe it more gently, or on fewer ports, from the internet class | ADR-0019's whole length, and it is worse here than there: the objection is not that the act is too large but that its target is the wrong machine, and no quantifier fixes a subject error |
