# ADR-0083: Silence decides only on a connection-oriented transport, and the honest connectionless value projects onto nothing

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#141 Can `connect-outcome` produce an honest UDP value?](https://github.com/winniel123/verge-asm/issues/141)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[#124](https://github.com/winniel123/verge-asm/issues/124) asked what the shipped instrument would
*return* if the UDP knob were ever opened, and found that nobody had. `connect-outcome` decides
`connected │ refused │ no-response` ([ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md)),
and a connected UDP socket puts no packet on the wire — `connect(2)` on a datagram socket only fixes
the peer address in our own kernel — so `connected` would be a fact about us rather than about the
world. #124 could not close it and ticketed it here, leaving
[`packaging-and-configuration.md`](../spec/packaging-and-configuration.md) §6 saying *"whether v1's
instrument could measure a UDP `Service` honestly at all is not settled here"* and
[`safe-active-probing.md`](../research/safe-active-probing.md) §9 saying *"the knob has nothing
honest to turn on"*.

The question is narrow and this ADR keeps it narrow. **Whether UDP is on is not in question** —
[#4](https://github.com/winniel123/verge-asm/issues/4) §2.5 put it off on measured signal-to-cost
grounds, [ADR-0009](./0009-verge-core-is-a-union.md) says in terms that ruling stands, and turning it
on would be an aperture change. What is in question is whether the instrument **could ever report
honestly**, because until that is answered ADR-0009's UDP leg has nothing to turn on.

Two live figures bound the work and neither moves here. `verge-core` is **136 pairs — 131 TCP and 5
UDP**, and the aperture statement reads **`5 of 38 sensitive pairs unread`** rather than `0`, because
the five UDP pairs (`69`, `137`, `138`, `623`, `11211`) reach `verge-core` from the sensitive list
alone and **membership of `verge-core` is not measurement**.

## Decision

| Concern | Decision |
| --- | --- |
| Can `connect-outcome` produce an honest UDP value | **No — and neither could a widened `connect-outcome`.** The leaf is wrong, not only the union |
| Is an honest UDP instrument constructible at all | **Yes.** Three values, two of which decide |
| The honest connectionless union | **`answered │ refused │ unanswered`** |
| `connected` | **Never produced for a UDP `Service`.** It names a state of our own kernel |
| `refused` | **Reused unchanged.** An ICMP Destination Unreachable / Port Unreachable is an RST's fact in kind — the host is up and nothing is bound. **The other type-3 codes fold in with it**, on the projection they share |
| `no-response` | **Not reused**, and this is the crux. It projects to `not-reached`, and that projection asserts more than a connectionless exchange decided |
| `answered` | **A datagram came back to the socket we sent from** — not necessarily from the port we probed, which TFTP's own reply rule forces. It reads **that** bytes returned, never **which**, so it does not cross [#5](https://github.com/winniel123/verge-asm/issues/5)'s fingerprinting line |
| The socket the leaf uses | **Unconnected**, and for a second reason independent of the ticket's — a connected datagram socket filters on peer address *and port*, so it would drop TFTP's reply |
| `unanswered` | A **value**, never a `Gap`, and the union's first member with **no `Reach` projection** |
| What the operator sees where the exchange did not decide | **`not-evaluable`** — [ADR-0010](./0010-exposure-composes-two-reaches.md) §4's own written behaviour for a `Gap` on the leg, arriving unchanged |
| The `Reach` `Gap` this opens | A **third route** ADR-0010 did not enumerate: not *we never looked* and not *we stopped looking*, but **we looked and the exchange did not decide** |
| Which facet | **One.** `reachability`, one flat union, partitioned by the `Service` key's transport |
| Where the UDP decision lives | A **sixth leaf, `datagram-outcome`** — specified here, **not shipped** |
| v1's `reachability` union | **Unchanged at three members** |
| v1's leaf count | **Unchanged at five** |
| The elicitation payload | An **aperture input** the day UDP ships, and the reason the knob is still not worth opening |
| Does UDP turn on | **No.** That is an aperture change and it is not this ticket's to make |

## Rationale

### The dishonest member is not the only dishonest thing — the leaf is

The ticket frames this as a value-space repair, and the cheapest reading is *swap one member and the
union is honest*. That reading is wrong one level up.

ADR-0021 settled that **a version leaf is named for what it decides**, and it recorded what
`connect-outcome` decides from: *"`connect-outcome` takes no bytes whatever. Its stimulus is a socket
event and a clock: SYN-ACK, RST, or silence for n seconds."* A connectionless measurement's stimulus
is a datagram **we compose** and an ICMP message or datagram **we receive**. Different stimulus,
different declared parameters, different golden-corpus medium — the leaf's corpus rows would carry a
payload, and `connect-outcome`'s carry no bytes at all.

Widening `connect-outcome` to cover both would put two decisions under one version, and the cost is
concrete rather than aesthetic: a change to a UDP payload would bump the leaf that **every TCP
`reachability` timeline in the estate composes**, so a UDP edit would `Break` the flagship value
estate-wide. ADR-0021's central property — *"no leaf is composed by every timeline"*, which is what
makes the unsurvivable fallback structurally unreachable — would be spent to save one name.

So the honest instrument is a **sixth leaf, `datagram-outcome`**, named the same way its sibling is:
for the outcome of the exchange it makes. It is specified here and shipped nowhere; **v1's leaf count
stays five**, and a session reading ADR-0021's table should not move it.

### `refused` survives and `no-response` does not, which inverts the ticket's emphasis

The ticket says *no member of it is honest for UDP* and then singles out `connected`. Walk all three
and the set does not fail the way the sentence implies.

- **`connected` fails**, exactly as stated. There is no handshake to complete.
- **`refused` holds.** An ICMP Destination Unreachable / Port Unreachable is the same fact in kind as
  an RST: the host is up, the datagram arrived, and nothing is bound to that port. The standing
  objection — that a middlebox can emit it — is symmetric with TCP, where a middlebox can emit an RST,
  and ADR-0011 accepted that for `refused` already. One value, one projection, no new case in the
  differ.
- **`no-response` fails, and not for the reason `connected` does.** `connected` is dishonest about
  *what we measured*. `no-response` is honest about what we measured and dishonest about **what it
  projects**.

That is the finding, and it is worth stating plainly because the cheap answer — *replace `connected`,
keep the other two* — is the option that came closest to winning.

### Silence is a value, and for a connectionless exchange it projects onto nothing

For TCP, `no-response` → `not-reached` is sound. We sent a SYN and reached no listener that would
answer; ADR-0011 states the residual honestly — *"`no-response` is the one reachability value that may
be about us"* — and both `refused` and `no-response` project to `not-reached`, so nothing false is
alerted.

For a connectionless exchange the identical projection is unsound **in the dangerous direction**. A
live, internet-reachable, unauthenticated listener that simply does not answer the datagram we sent
would read `not-reached`, and `sensitive-port-reached-from-internet` would report a clean bill of
health on precisely the pairs the sensitive list exists for. That is
[ADR-0010](./0010-exposure-composes-two-reaches.md)'s own founding defect — *"it simply never fires on
the more alarming case"* — reproduced not by a list falling behind but by a projection asserting more
than the exchange decided. And it is the defect #124 just caught, one layer down: a `0` standing where
`5 unread` was true, arriving this time inside a `Signal` rather than inside a coverage line.

So silence needs its own member. Three things constrain what it can be, and together they leave one
answer.

**It is a value, not an absence.** ADR-0011's central argument is that a negative modelled as an
absent field is *"indistinguishable from we did not look"*. Recording nothing where the exchange did
not decide would leave the operator unable to tell *we probed `11211/udp` and got nothing* from *we
did not probe `11211/udp`*, which is [#28](https://github.com/winniel123/verge-asm/issues/28)'s and
[#44](https://github.com/winniel123/verge-asm/issues/44)'s refused clean bill of health.

**It is not a `Gap`.** ADR-0011 reserves `Gap` for *we did not look, and nothing else*. We looked.

**It projects onto nothing.** ADR-0010 gives `Reach` two values and refuses a third **by name** —
*"there is no third value for we did not look… adding a `not-checked` value would re-invent `Gap`"* —
and [#40](https://github.com/winniel123/verge-asm/issues/40) / ADR-0013 deleted `unknown` outright for
being a Derived value with no producer. Both bars hold here. So `unanswered` is the union's first
member with **no `Reach` projection**: the leg holds no value that batch, and under ADR-0010 §5 the
`Exposure` timeline behaves as it already does when a leg has none.

The name is chosen the way this project has chosen four times running — the measured word over the
taxonomy's. Nmap's word is `open|filtered`, which is two conclusions joined by a bar and not a single
value at all; `filtered` names a conclusion the exchange cannot establish, refused for TCP already;
`silent` names a property of the listener rather than of our exchange, which is the thing
`CONTEXT.md` bars when it says a negative is *named for the exchange we made*. **`unanswered` names
our datagram and claims nothing about what is there** — which is precisely the claim we are entitled
to.

### `not-evaluable` is the instrument working, and it is where the knob's value goes

The consequence reads as a defect and is not one. ADR-0010 §4 already writes the behaviour: the
flagship signal *"fires where the internet `Reach` is `reached`, does not fire where it is
`not-reached`, and is `not-evaluable` where it is a `Gap`."* An `unanswered` UDP `Service` therefore
returns `not-evaluable`, through the mechanism that already exists, with no new lifecycle state and
no new rule — which is
[ADR-0006](./0006-subjects-leave-by-measurement.md)'s habit satisfied rather than dodged.

But it prices the knob, and the price is the reason this ADR does not end in a recommendation to open
it. **A datagram sent with no protocol-specific payload elicits nothing from almost anything.** That
is why nmap ships `nmap-payloads` at all, and it is the mechanism behind the coverage figures §2.5
already quotes — 39 % at top-100 UDP and 49 % at top-1000, i.e. even a thousand-port UDP scan misses
half of what is open. A payload-free `datagram-outcome` would therefore return `unanswered` on the
five sensitive UDP pairs essentially always, and `unanswered` returns `not-evaluable`.

Which is **exactly what those five pairs report today**. ADR-0009's Consequences say so — the five
*"remain `not-evaluable` on default settings, by design and visibly"* — and ADR-0004 §117 says the
same. So:

> **Opening the knob with a payload-free instrument moves the five pairs from `not-evaluable` because
> they are outside the recorded scope to `not-evaluable` because the exchange did not decide, and
> ~~changes nothing the operator sees~~ CHANGES ONE THING THE OPERATOR SEES, AND IT CHANGES IT FOR
> THE WORSE.** It buys probe traffic, a sixth leaf, a golden corpus of a new
> medium and an eighth aperture input, and it buys **zero** net new firings.

> **The struck clause is CONTRADICTED, not refined — 2026-08-15 by
> [#173](https://github.com/winniel123/verge-asm/issues/173) ·
> [ADR-0095](./0095-the-aperture-statement-counts-what-the-instrument-cannot-report-not-what-it-did-not-look-at.md),
> the successor this ADR named.** The signal-level claim is correct and untouched: the five pairs
> return `not-evaluable` before and after, so the operator's `Signals` rows do not move. The
> **aperture statement** does. [ADR-0044](./0044-a-one-off-measurement-has-no-currency.md)'s
> port-tier line counts sensitive pairs that are **unread**, meaning outside the recorded scope —
> and a payload-free UDP tier **reads** all five. The line goes to **`0 of 38 sensitive pairs
> unread`**, a clean bill of health on the one surface built to prevent one, while the five go on
> returning `not-evaluable` forever. **Zero net new firings is right; *changes nothing* is not.**
>
> **This strengthens the ruling above rather than weakening it.** The knob does not merely buy
> nothing — it costs the sentence that says we are not looking. ADR-0095's repair is a **second
> figure** on that line, valued `0` today: *`M of 38 sensitive pairs the instrument cannot report as
> `reached``*. Nothing in this ADR's Decision moves; UDP stays off, the union stays at three
> members, the leaf count stays five, and `datagram-outcome` stays specified and unshipped.

That is [ADR-0015](./0015-the-value-space-is-the-commitment.md)'s wire-prober refusal arriving in a
second costume — *"buys six ports and zero net new firings"* — and it decides the knob's fate on the
same ground.

**So the elicitation payload is the whole of the knob's value, and it is the instrument this map
already put out of scope.** A per-pair payload table with a per-protocol encoder is
`listener-negotiation`'s dispatch table under another name, deferred by ADR-0015, and its first
unclosed obligation is the one ADR-0015 named: *"§7.2 argues a wrong dispatch guess fails safe for the
data, and never asks whether it is safe for the listener"* — now aimed at production over a transport
with no handshake to fail on, and in one case (`69/udp`) with a payload that is a **read request**
rather than a hello. The honest statement of v1's UDP position is therefore no longer *UDP is off
because it is expensive*. It is:

> **The instrument that would make UDP worth turning on is the instrument this map already deferred.**

The payload table's classification, ruled here so the successor need not re-derive it:
[#31](https://github.com/winniel123/verge-asm/issues/31) / ADR-0008 cut it — *a table deciding where
to look is aperture; a table deciding what an answer means is a signature database.* A payload table
decides which pairs can produce a positive at all, so it is **aperture**, and it is an **eighth
aperture input** the day UDP ships. It stays on the legal side of #5 only because `answered` reads
*that* a datagram returned and never *which bytes*; the moment a rule reads the payload's content it
is a verdict table and #5 bites.

### One facet, because the subject key already carries the transport

A `udp-reachability` facet is the obvious alternative and it loses on the model's own record.
ADR-0011 states that *"`reachability` never needed a discriminator **only because its subject
`Service` already carries port and transport**"*, and calls that *"the same fix applied one level up,
hand-made, before anyone named it."* A second facet would re-encode the transport in the **facet's
name** — the discriminator arriving through the front door of the thing ADR-0011 fixed.

It costs more than the duplication, too. `Reach` would acquire two producers, and every `Service`
would hold a permanent `Gap` on whichever facet its transport is not — a facet structurally absent on
most of its subjects, and an arbitration rule for a case where exactly one producer can ever speak.

So: one facet, one flat union of five members, and the TCP/UDP partition enforced where this project
enforces things — **on the golden corpus**, where CI can check that no row with transport `udp`
produces `connected` and no row with transport `tcp` produces `answered` or `unanswered`. A mechanism,
not a judgement, which is ADR-0011's own standard for the additive test.

The precedent for a flat union with circumstance-restricted members already exists twice over:
`certificate`'s `TLSRefused` can only arise where TLS was attempted, and `dns-record`'s `Shadowed`
only under a wildcard.

**The composition is unchanged and already written.** `Reach` is a class-scoped `Vantage
composition` quantified `existential`
([#138](https://github.com/winniel123/verge-asm/issues/138) / ADR-0080), and this union composes with
that rule rather than amending it: the leg is `reached` where **any** vantage of the class read
`answered`, `not-reached` where **every** deciding vantage read `refused`, and where no vantage of
the class decided at all the set of deciding vantages is empty — which ADR-0080 already rules is
`not-evaluable` and never vacuous. Nothing here makes anything a `Vantage composition` that was not
one; `datagram-outcome` is a **prober** leaf in ADR-0021's sense, deciding a value, not composing
observations.

**The other ICMP unreachable codes fold into `refused` rather than earning a member.** An
administratively-prohibited or host-unreachable reply is a different fact from a port-unreachable —
something on the path declined to carry the datagram, rather than the host declining to bind — but
both are *something authoritative said this datagram is not reaching a listener*, both project
`not-reached`, and `connect-outcome` already folds the identical case for TCP without anyone having
named it. Splitting them would be a distinction no consumer uses. Recorded so the successor knows it
was decided rather than missed.

### The widening is strictly additive, so there is no deadline — and this ADR refuses to land-grab

The instinct at this point is to widen `reachability` to five members **in v1**, on
ADR-0015's *"a facet's value space is decided once."* That instinct is the argument ADR-0015 killed
map-wide, and its one surviving form does not reach this case.

ADR-0011's test: *"a canonicaliser change is **strictly additive** when every corpus row whose output
moved previously produced **no observation at all** — and CI can check exactly that, so this is a
mechanism and not a judgement."* Every row whose output would move here is a **UDP** row, and no UDP
row has ever produced an observation, because no `Batch`'s recorded scope has ever contained a UDP
pair. The widening is therefore strictly additive by construction and checkable, so it costs **no
`Break`** — a count of timelines opened and no comparison at all — whenever it lands.

ADR-0015's *"widening an existing facet's value space is expensive"* was argued on a **field** added
to `http-identity`, which moves the output of rows that all produced observations. A **variant** that
fires only where nothing was ever recorded is the case ADR-0011 explicitly carved out. The two are
consistent and a session must not read the first as covering the second.

**So v1's union stays at three members.** This ADR specifies the value space and does not spend it.
Any future ticket arguing the UDP members must ship in v1 *because adding them later costs
comparability* is using the constraint ADR-0015 withdrew.

### The hazard nobody has named: ICMP rate limiting puts our own probe rate inside the value

This is the finding the ticket did not ask for and the successor must not lose.

Hosts rate-limit the emission of ICMP error messages. Probing many closed UDP pairs on one address
therefore yields `refused` for some and `unanswered` for others **in the same world**, and the split
is decided by **our own probe rate**. ADR-0021's alternatives table already refuses precisely this
shape, one layer up: *"Adaptive back-off inside `connect-outcome` — it halves the rate, never the
deadline — and had it moved the deadline, **a value would depend on how busy the run was**."* For
`connect-outcome` that is safely true, which is why back-off ships as an operator knob at
`safe-active-probing.md` §9 and clears
[`packaging-and-configuration.md`](../spec/packaging-and-configuration.md)'s gate 1.

**It is not true for `datagram-outcome`.** On the UDP leg, rate reaches the value. Two things follow
and both are the successor's:

1. **The per-host rate and the retry count are declared parameters of `datagram-outcome`**, and
   adaptive back-off may **not** compose it — or the leaf becomes non-deterministic and fails the
   golden-corpus gate, which is
   [`project-authored-constants.md`](../research/project-authored-constants.md) §6.1's existing
   argument arriving on a second leaf.
2. **`safe-active-probing.md` §9's *adaptive back-off aggressiveness* knob would fail gate 1 the day
   UDP ships**, for the leg it reaches. It survives today because the only leaf it touches is one it
   cannot move.

And it sharpens an open fog patch rather than creating one. The map already asks *how many `Span`s
does an unstable network write on the `reachability` facet*, noting `refused` ↔ `no-response` is *"the
span corpus's only unbounded generator and it is silent by design."* A UDP leg adds a **second and
worse** generator: `refused` ↔ `unanswered` flaps on **our own rate** rather than on the network, so it
is unbounded for a reason the operator cannot fix and we can.

## Consequences

- **ADR-0009's UDP leg has a specified instrument for the first time, and still nothing worth turning
  on.** The leg is `verge-core`'s UDP pairs, and it now has a named leaf, a value space and a
  projection rule. What it does not have is a way to reach `answered`, and that is the deferred wire
  prober's to supply.
- **No live figure moves.** `verge-core` stays **136 pairs — 131 TCP, 5 UDP**; the aperture statement
  stays **`5 of 38 sensitive pairs unread`**; the v1 rule set stays **17**; the aperture inputs stay
  **seven**; the facets stay **six**; the leaves stay **five**; `reachability`'s union stays
  **three members**. This ADR specifies and spends nothing. A session reading it as a widening should
  re-read the additive section.
- **`unanswered` would be v1's first facet value with no Derived projection**, and ADR-0010's clean
  identity — *absence of a `Reach` value ⟺ the pair was outside the recorded scope* — would no longer
  hold. That is a real cost, it lands on `Coverage`'s reading of a `Gap`, and it is the first thing
  the successor ticket owes.
  *(**Paid, 2026-08-15 — [#173](https://github.com/winniel123/verge-asm/issues/173) ·
  [ADR-0095](./0095-the-aperture-statement-counts-what-the-instrument-cannot-report-not-what-it-did-not-look-at.md).**
  ADR-0010 takes a **conditional** at three sentences and a strike at none — the identity holds while
  every probed transport's outcome union projects totally onto `Reach`, which TCP's does and this
  ADR's does not. `Coverage`'s reading is a **second figure** on the port-tier line, valued `0` on
  every shipped configuration. And the route splits: where the leg had already opened it is a `Gap`,
  and where it never opened it is **nothing at all** — the second having no span and therefore no
  recorded cause, which is why the aperture statement had to carry it.)*
- **[ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) fires at
  three sites**, all saying the question is open or the knob has nothing honest to open:
  `safe-active-probing.md` §9's UDP row, `packaging-and-configuration.md` §6's closing paragraph, and
  §2.5, which routes a reader to the cost argument alone. All three are amended in place. ADR-0021 and
  ADR-0009 take annotations rather than strikes, because **nothing they state is superseded** — the
  union really is three members and the leaves really are five.
- **`CONTEXT.md`'s `Reach` entry gains one clause.** *"`reached` or `not-reached`, and nothing else"*
  read alone would cause a competent session to build a total projection over any transport, which is
  ADR-0058's test met inside the glossary.
- **All five sensitive UDP pairs are elicitable in principle, and the walk is spec-verified rather
  than measured** — [`safe-active-probing.md`](../research/safe-active-probing.md) §13.6 carries it
  per pair with its footing, and it inherits ADR-0021's rider that *a corpus row inherits the
  evidential status of the claim it encodes*. Two of the five carry caveats that change the pricing
  rather than the ruling: `69/udp`'s reply arrives **from an ephemeral port**, and `11211/udp` has
  been **off in memcached's own shipped default since 1.5.6**. A retrieval per pair against
  implementations, not only specifications, is owed before any payload ships.
- **A licence flag is raised and lowered rather than left implicit.** Nmap's payloads no longer live
  in a standalone `nmap-payloads` file; they are built from `nmap-service-probes`, which is NPSL
  data. [#78](https://github.com/winniel123/verge-asm/issues/78) cleared *deriving* `verge-core` from
  `nmap-services` and does **not** clear selecting rows out of a second nmap data file — and
  [#128](https://github.com/winniel123/verge-asm/issues/128)'s rule cuts the wrong way here, since
  *selecting from* a table **is** authoring. Nothing is lifted by this ADR, so nothing forces a
  licence change today; the payload table must be **authored against the protocols' own
  specifications**, which the walk in §13.6 shows is possible for all five and necessary for one —
  `138/udp`, for which nmap has no payload at all.
- **This ADR does not turn UDP on and could not.** That is an aperture change and it belongs to
  whoever prices the payload table.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Replace `connected` and reuse the other two — a three-member UDP union** *(the option that lost)* | The cheapest answer, strictly additive by construction, and what a session reading the ticket's own emphasis would write. It fails on `no-response`: that value projects to `not-reached`, so a live internet-reachable listener that ignores our datagram would read as a negative reachability verdict and the flagship would report a clean bill of health on the pairs the sensitive list exists for. ADR-0010's founding defect, reproduced by a projection instead of by a list |
| Widen `connect-outcome` rather than adding a leaf | Puts two decisions under one version, so a UDP payload edit `Break`s every TCP `reachability` timeline — spending ADR-0021's *no leaf is composed by every timeline* to save a name |
| A `udp-reachability` facet | Re-encodes the transport in the facet's **name**, which ADR-0011 records the subject key already carries; and it gives `Reach` two producers with an arbitration rule for a case where only one can ever speak |
| A transport-tagged union — `Tcp(…) │ Udp(…)` | Puts the invariant in the type, which is the tidier model, and moves the output of **every** row that ever produced an observation. Not strictly additive, so it `Break`s every TCP timeline for a property CI already checks on the corpus |
| Record no observation where the exchange did not decide | ADR-0011's central argument verbatim: a negative modelled as absence is indistinguishable from *we did not look*, and the operator could not tell an unanswered probe from an unprobed pair |
| Call it a `Gap` | `Gap` is reserved for *we did not look*, and we looked |
| A third `Reach` value, or `unknown` | Refused by name in ADR-0010 and deleted outright by #40 / ADR-0013 for having no producer. `unanswered` needs neither: `Reach` simply holds no value, and ADR-0010 §4 already says what the rule returns then |
| A `not-probed` state distinct from `not-evaluable` | ADR-0009 already considered and rejected it for these exact five pairs; nothing here revives it |
| Name it `filtered`, `open|filtered` or `silent` | `filtered` names a conclusion a datagram cannot establish, refused for TCP already; `open|filtered` is two conclusions joined by a bar and not one value; `silent` names a property of the listener rather than of our exchange, which `CONTEXT.md` bars |
| Widen `reachability` to five members in v1, on *the value space is decided once* | The land-grab argument ADR-0015 killed map-wide. Its surviving form covers a **field**, and a variant firing only where nothing was recorded is ADR-0011's strictly-additive carve-out — free whenever it lands, and CI-checkable |
| Ship the UDP leg payload-free, since the values are honest | Honest and useless: `answered` is unreachable without a payload, so the five pairs move from `not-evaluable` for one reason to `not-evaluable` for another ~~and the operator sees no change~~. ADR-0015's *zero net new firings* refusal, in a second costume. **~~No change~~ is corrected at this cell by [#173](https://github.com/winniel123/verge-asm/issues/173) · [ADR-0095](./0095-the-aperture-statement-counts-what-the-instrument-cannot-report-not-what-it-did-not-look-at.md): the aperture statement's port-tier line goes from `5 of 38 sensitive pairs unread` to `0 of 38`, because *unread* means outside the recorded scope and a payload-free tier reads all five. The refusal is unchanged and better founded — it is worse than useless, not merely useless.** |
| Turn UDP on as part of this ruling | An aperture change, and a scope matter this ticket was explicitly not given |
