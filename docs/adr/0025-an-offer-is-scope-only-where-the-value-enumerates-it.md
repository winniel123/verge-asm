# An offer is scope only where the value enumerates it — and a default is not a declaration

- **Status:** Accepted
- **Date:** 2026-08-13
- **Ticket:** [#54 Is everything our client offers an aperture input, or only the TLS candidate set?](https://github.com/winniel123/verge-asm/issues/54)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0011](./0011-a-facet-is-six-parts.md) took *which versions and ciphers we offer* out of the
`tls-acceptance` value and made it **the `Batch`'s recorded scope**, on the ground that *what we
offer decides what the value can say*: a batch that offered nine ciphers can never assert the tenth
was refused. That made the TLS candidate set an aperture input.

The argument is not about TLS, and nobody had asked how far it reaches.
[ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md) had to enumerate every decision the
measurement binary makes in order to name its `Derivation` leaves, and hit three offers it could
not place — **the ALPN list and the HTTP versions we speak**, **EDNS options and buffer size**, and
**the DNS transport and fallback policy** — recording them as its own open question. It also
created a second home competing for the same fact: a third-party wire library is a **declared
parameter** of a leaf wherever it speaks the protocol on our behalf, and `crypto/tls` choosing the
ClientHello is exactly the fact ADR-0011 had just made batch scope.

Two things give this a deadline.
[ADR-0015](./0015-the-value-space-is-the-commitment.md) established that a facet's value space is
the commitment and the signal set over it is free — so *does `http-identity` record this* is a v1
question where *does a rule read it* is not.
[ADR-0014](./0014-only-revealed-generalises.md) established that aperture is what a `Batch` records
as its completed scope and that the named inputs stay **enumerated**, because a widening is
detected by diffing named dimensions — so a dimension that arrives after v1 cannot say what the
earlier batches offered.

## Decision

| Concern | Decision |
| --- | --- |
| Does ADR-0011's argument generalise to every client-side offer? | **No.** It generalises to every offer the **value enumerates over**, and to no other |
| The test | An offer is batch scope exactly where the value carries a **per-candidate negative** — *we asked about X and X was not there*. Otherwise it is a **declared parameter** of the leaf that made it |
| ALPN and the HTTP versions we speak | **Declared parameter** of `http-exchange`. Not an aperture input |
| EDNS options and buffer size | **Declared parameter** of `resolution-walk` and `wildcard-discrimination`. Not an aperture input |
| DNS transport and fallback policy | **Declared parameter** of `resolution-walk`. Not an aperture input |
| The enumerated aperture input list | **Unchanged at six** — and it was six, not five, from [ADR-0017](./0017-exposure-needs-both-legs.md) |
| Can a library **default** be recorded honestly? | **No, in either home.** A default is not a declaration |
| What v1 must therefore do | Every offer the binary makes is **enumerated in the job spec and passed to the library explicitly**; the `Batch` records what went on the wire, never the library's identity |
| What an aperture widening costs | A **`Break`** on the timelines it touches; **`revealed`** only on the timelines it *opens*. Sorted by whether the dimension sits in the **key** or in the **value** |
| `http-identity`'s negative | **Renamed** `NotHTTP` → **`NoHTTPResponse`**. The value space is **not** widened |
| The negotiated HTTP version | **Never recorded** — it is *TLS 1.0/1.1 negotiated* again, a function of our own offer |
| `certificate`'s negative | Gains a third variant: `Presented(chain) │ **TLSRefused** │ NoTLS` |
| The trap — one change paying a `revealed` *and* a `Break` | **Structurally unavailable** once the offer is declared; and it would **not** have composed, because both prices land on one object |

## Rationale

### The argument was never about offering; it was about a value that enumerates an offer

The tempting reading of ADR-0011 is *anything our client chooses can move a value, so anything our
client chooses is aperture*. That reading is already false in this repo, and
[ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md) is the counterexample: the connect
timeout can move `connected` → `no-response` on every slow host in the estate, and it is a
**declared parameter** of `connect-outcome`, not an aperture input. Nobody argued that case; it was
simply obvious. What was not obvious is *why*.

The reason is the shape of the value, not the existence of the offer. `tls-acceptance`'s value
**is a subset of the candidate set**: every candidate we offered and did not get back is a negative
sitting inside the value. That is [ADR-0009](./0009-verge-core-is-a-union.md)'s `{161}` defect
exactly — a record asserting an absence over something it never touched — and the recorded scope is
the mechanism that already exists for it. The connect timeout indexes nothing: `no-response` is one
member of a three-value closed union, not a set with a missing member.

> **An offer belongs to the `Batch`'s recorded scope exactly where the value carries a
> *per-candidate negative*. Where the offer merely *conditions* a single value, it is a declared
> parameter of the leaf that made it.**

The test recovers all six settled aperture inputs and correctly excludes the settled parameters,
which is the tell that it is the cut and not a rationalisation:

| Input | Per-candidate negative? | Home |
| --- | --- | --- |
| Enabled sources | A source never asked would be an absent per-source timeline | Scope |
| Port and transport tiers | A pair never probed would be an absent `Service` | Scope |
| The custody gate | An address never probed would be an absent everything | Scope |
| The queried qtype set | A qtype never asked would be an empty RRset | Scope |
| The TLS candidate set | A candidate never offered would read as refused | Scope |
| `Vantage class` | A class never run would be a leg `Exposure` composes as though absent | Scope |
| Connect timeout, retries | `no-response` is one value, not a set with a hole | Parameter |
| Handshake / request / query timeouts | Same | Parameter |
| Control-label count | `Shadowed` is one value | Parameter |

### The three candidates, each for its own reason

**The ALPN list and the HTTP versions we speak — parameter.** `http-identity`'s value space is
`Responded(status, Location, WWW-Authenticate, Server, title) │ NotHTTP`. Nothing in it is
enumerated over an HTTP-version list; there is no *we offered h2 and h2 was refused*. The negative
is a **single global one**, and a single global negative is a conditioned value, not a set with a
hole. What the offer does to it is a naming problem, answered below, and not an aperture problem.

**EDNS options and buffer size — parameter, and the reason is different.** The ticket's worry is
real and sharper than the ALPN one: a 512-byte advertised buffer against a large TXT RRset provokes
truncation, and a truncated RRset recorded as the value is *smaller than the world*. But that is a
correctness failure, not an aperture one, and it is **detectable in band**: the TC bit is a
spec-defined field that says the answer is incomplete, so the binary never has to guess. So the
rule that discharges it is one sentence and it is not about aperture:

> **A truncated answer is never a value.** `resolution-walk` retries over the fallback transport, or
> it records no value — a `Gap`, *we could not say* — and it never folds a partial RRset.

With that rule in force, changing the advertised buffer changes only how often the fallback path is
taken. It moves latency, not values. One EDNS option is a genuine exception and is dealt with by
refusing it: **EDNS Client Subnet is a `Vantage` in an option's clothes** — it makes a geo-aware
authority's answer a function of the subnet we claimed, which is the job the `vantage` component of
the timeline key already does. v1 sends no ECS, and if a later version does, ECS belongs in the
**key**, never in the scope record.

The second EDNS hazard needs saying because it lands on a v1 signal: an authority that requires DNS
cookies may answer a cookieless query with `BADCOOKIE` or `REFUSED`, and a delegation walk that
reads a refusal as *this nameserver does not serve the zone* would emit a false `Lame`.
`resolution-walk` **may not convert a transport-level refusal into a zone-level value**. That is
`Lame`'s own definition holding — *a measurement of the operator's infrastructure, not a failure of
ours* — against an instrument nobody had tested it against, and the discriminating facts
(`BADCOOKIE`, rcode 23) are spec-defined fields, so it is
[#31](https://github.com/winniel123/verge-asm/issues/31)'s zero-row verdict table and not a
signature database.

**The DNS transport and fallback policy — parameter, and it is decided by a rule already on the
books.** The ticket phrases the hazard as *whether we retry over TCP decides whether a large RRset
is `Resolved` or nothing* — and it names its own answer, because **nothing is a `Gap`**, and a
`Gap` is a legitimate object that records its cause. Enabling fallback in a release turns `Gap`s
into values, and [ADR-0014](./0014-only-revealed-generalises.md) has already ruled that edge:
*"Treat `Gap` → value as `revealed` — the timeline has history; nothing was revealed, we resumed."*
An aperture input produces an **opening**. This produces an ordinary adjacency in the coverage
class. It cannot be an aperture input, on ADR-0014's own text, with no appeal to the test above.

### A default is not a declaration, and this is the load-bearing half

The ticket asks whether an offer that is a **library default** rather than a declared parameter can
be recorded honestly at all. It cannot — and the answer is the same in both homes, which is what
disposes of the two-homes problem.

**Aperture needs content.** ADR-0014 makes a widening detectable *by diffing named dimensions across
batches*. Diffing `go1.24.1` against `go1.24.2` says something moved; it does not say whether the
offer widened or **narrowed**, and a narrowing is the opposite of `revealed`. A scope record that
cannot tell the two apart cannot license anything.

**A parameter needs content too.** ADR-0021 lets a leaf version move on *a changed declared
parameter*, and makes that mechanically checkable *because the parameter set is declared data*. A
parameter whose value is *whatever the library defaults to* is not declared data. It is
[ADR-0008](./0008-derivation-versions-move-on-content.md)'s rejected **hash of the parameters** —
silent on a behavioural change — wearing a declaration's clothes.

So the remedy is not a better record. It is to stop defaulting:

> **Every offer the measurement binary makes is enumerated in the job spec and passed to the library
> explicitly, and the `Batch` records what went on the wire.** The library's identity is a parameter
> covering only what remains its to decide.

That draws the line between the two mechanisms cleanly, and it is a line rather than an overlap:
the moment we enumerate an offer and pass it down, **the library stops deciding it**. What is left
to `net/http` is header-parsing strictness — what counts as HTTP — which we cannot enumerate and
which is exactly what ADR-0021 named the library parameter for. What is left to `crypto/tls` is how
it behaves given our list. Nothing is in both places.

The rule has a stated ceiling rather than a hidden one. A library may be **unable** to offer a
candidate we declare — Go cannot speak SSLv3 or RC4 at any setting — so the recorded scope is what
went on the wire and never what we intended. An unofferable candidate is an absence from the scope
record, visible, and not a silent one.

### The forcing measurement: `crypto/tls`'s default hides the finding the product exists to make

This is not a tidiness argument, and the worked example is in the flagship direction.

Go's TLS client has defaulted to `MinVersion: TLS 1.2` since Go 1.18. Take the every-run handshake
that feeds `certificate`, left on library defaults, against a legacy box that speaks **TLS 1.0
only**. The server sends a `protocol_version` alert, no handshake completes, and the facet records
`NoTLS`. The service drops out of the population `tls-1.0-accepted` evaluates over — so a v1 signal
reports nothing **on exactly the estate where it is true**, which is the failure ADR-0011 refused
when it kept the measured negatives out of `Gap`, arriving instead through a dependency's default.

Nobody chose that. A default chose it, and no record in the model would have shown it: the `Batch`
would have recorded a candidate set nobody wrote, and the leaf's parameter set would have recorded
`crypto/tls`, which is true and says nothing.

So the declared candidate set for the `certificate` handshake is **wide on purpose** — every version
we intend to be able to detect, which includes the ones we intend to report as findings. Note that
the every-run handshake and `tls-acceptance`'s weekly enumeration are **different `Batch`es on
different tiers**, so each records its own candidate set, and they need not match.

### An aperture widening `Break`s where the dimension sits in the value

Pricing this ticket's answer exposed a mis-statement that has been propagating since
[ADR-0007](./0007-drift-is-a-timeline-of-spans.md), and it has to be corrected here because the
whole question is *what does being an aperture input cost*.

ADR-0007 says of the qtype set and the TLS candidate set: *"Both start timelines that did not
exist, so both yield `revealed` rather than `appeared`"*, and
[`CONTEXT.md`](../../CONTEXT.md) repeats it for `tls-acceptance` — *"widening the offer is an
aperture change yielding `revealed`"*. **It is true of the qtype set and false of the TLS candidate
set.** The qtype is a **discriminator in the timeline key**, so adding CAA opens a `dns-record`
timeline that did not exist. A cipher is not in any key: `tls-acceptance` has one timeline per
`Service` and adding a candidate moves its **value**. `revealed` is an *opening* kind
([ADR-0014](./0014-only-revealed-generalises.md)); a value moving on a running timeline is not an
opening at all. It is a **`Break`** — which is already `Break`'s second cause, so nothing new is
needed.

The general rule, which sorts all six inputs and needs no case analysis:

> **An aperture widening `Break`s the timelines it touches and `revealed`s the timelines it opens.**
> A dimension in the **key** opens; a dimension inside the **value** breaks.

Four of the six open (sources, port and transport tiers, the custody gate, the qtype set) and two
break (the TLS candidate set, `Vantage class`). This is not new doctrine — it is
[ADR-0017](./0017-exposure-needs-both-legs.md) five hours ago, which priced `Vantage class` as
*"`Break`s every `Exposure` timeline on the aperture cause, not the version cause"* and did not
generalise. ADR-0014's *"a `Break` on an aperture widening that only creates timelines is vacuous"*
is the same rule read from the other end.

The consequence is not cosmetic: widening the TLS offer is **expensive**, a `Break` on every
`tls-acceptance` timeline in the estate, where `CONTEXT.md` currently reads as though it were free.

### `http-identity` records what the exchange returned, and the value space does not widen

`NotHTTP` asserts a property of the listener. What was measured is that **the exchange we made**
returned no HTTP response — and with an `http/1.1`-only offer, an h2-only listener lands there
having done nothing wrong. That is a name reading as a claim about the world while its predicate
mentions us, which this map has now caught five times (`Host`, `sensitive-port-exposed`,
`internal-only`, `filtered`, *TLS 1.0/1.1 negotiated*). The remedy is the one used every time
withdrawal was unavailable: **rename**. `NotHTTP` becomes **`NoHTTPResponse`**. Nothing has shipped,
so the price is **vacuous** — not waived, in ADR-0009's sense: `Break` is a property of timelines
that exist and there are none.

The value space is **not** widened, and refusing to widen it is the substantive half.

*Adding a variant distinguishing "the peer spoke a version we did not offer"* is refused because
nothing can emit it. Over TLS the case is measurable — ALPN selection is a spec-defined field — but
the residual case is **h2c with prior knowledge in cleartext**, and telling that from non-HTTP bytes
needs a second probe sending an HTTP/2 connection preface at whatever is on the port, which is a new
class of risk against production that [#4](https://github.com/winniel123/verge-asm/issues/4)'s
profile has not budgeted and which is precisely the cost
[ADR-0015](./0015-the-value-space-is-the-commitment.md) deferred the wire-protocol prober over.
[ADR-0013](./0013-custody-is-control-and-extends-by-declaration.md) deleted `unknown` from
`Ownership` for having no producer, and *"a value nothing can emit is worse than an invented state,
because it reads as a real distinction forever"* binds here unchanged.

*Recording the negotiated HTTP version* is refused harder, because it is a mistake this project has
already made and corrected: a negotiated parameter is a function of **our own offer**, so it moves
estate-wide on a library upgrade with nothing in the world having changed, and it answers the wrong
question. That is `certificate` carrying *TLS 1.0/1.1 negotiated*, verbatim, one facet across.

The right fix is not a wider value space but a **wider offer**, and it is nearly free: the declared
ALPN offer is `h2, http/1.1`, so an h2-only-over-TLS listener — a gRPC service being the modal case
in a small estate — **answers**, and `Responded(...)` is correct rather than modelled. The residue
is stated rather than hidden: **h2c with prior knowledge in cleartext is indistinguishable from
non-HTTP bytes in v1**, and `NoHTTPResponse` is honest about it where `NotHTTP` was not.

An *`http-acceptance`* facet enumerating which HTTP versions a listener accepts is the symmetric
object to `tls-acceptance` and it is **out of scope**, priced by ADR-0015 at `revealed` plus one
message with no `Break` whenever anyone wants it.

### `certificate` had a hole where the other half of ADR-0011's partition should be

ADR-0011 justified `NoTLS` as a real measurement rather than an inference by citing
[#31](https://github.com/winniel123/verge-asm/issues/31)'s measured partition — *`wrong version
number` against `unexpected eof`* — a TLS speaker refusing our offer against a listener that is not
TLS at all. It then gave `certificate` the value space `Presented(chain) │ NoTLS`, which has **one**
value for the two sides of that partition. The justification and the value space disagree, and the
disagreement was invisible while the offer was the library's, because nobody could say what had been
refused.

So `certificate` gains a third variant, **`TLSRefused`**: the peer spoke TLS and accepted no
candidate we offered. It is emittable — a TLS alert is RFC 8446 §6, a spec-defined field, zero
verdict rows — and it is not a rare curiosity. With a declared wide offer it is the bucket that
holds SSLv3-only, RC4-only and 3DES-only listeners that Go **cannot** offer at any setting, plus
servers requiring SNI probed through a nameless `Endpoint`. Those are the most legacy boxes in an
estate, and `NoTLS` files them under *not a TLS server*.

This is a value-space widening, so it is not additive and it has ADR-0015's deadline: after v1 it
would `Break` every `certificate` timeline in the estate. Before v1 the price is vacuous, for the
same reason the rename's is.

The honest caveat, marked rather than dressed: the **mechanism** is sound and the **frequency** is
unmeasured. Nobody has counted TLS-1.0-only or SNI-required listeners in a small-org estate, and
this ADR does not pretend to have. It is the same position ADR-0021 took on its uncovered-move
branch.

### The trap, and why ADR-0009's composition would not have saved it

The ticket names the trap precisely: an offer recorded in **both** places pays twice for one change,
a `revealed` and a `Break`, where [ADR-0009](./0009-verge-core-is-a-union.md) found such composition
clean only because the two prices land on **different objects**.

Check it rather than assume it, as the ticket asks. ADR-0009's case was a sensitive-list revision:
the signal's version governs comparability, the new `Service` timeline starts fresh — two objects.
Here both prices would land on the **`tls-acceptance` timeline of the same `Service`**. They do not
compose; they double. ADR-0009's finding does not transfer, and the trap is real.

The declaration rule disarms it structurally rather than by care. One fact has one home: either we
enumerate the offer, in which case it is scope and the library is not deciding it — so the library's
version moving does not move the offer — or it is the library's, in which case there is no scope
dimension to widen. There is no configuration in which one change pays both prices.

What survives, correctly, is **two changes each paying once on the same object**: a Go upgrade bumps
`tls-handshake` and `Break`s `certificate` and `tls-acceptance`, and our own decision to widen the
offer is an aperture widening that `Break`s `tls-acceptance`. Two causes, one object, two changes.
That is fine and is not what the trap named.

[ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md)'s own worked example is the doubled
case written down as though it were normal — *"a Go upgrade fires two messages of one class:
widening the TLS offer is an aperture change yielding `revealed`, while the leaf bump is a
re-baseline"*. Under this ADR **a Go upgrade cannot widen the TLS offer**, because the offer is
ours. It bumps the leaf and nothing else.

## Consequences

- **[`CONTEXT.md`](../../CONTEXT.md) changes in four places.** `tls-acceptance` states that the
  offer is **declared by us and recorded as wire content, never a library default**, and that
  widening it is a `Break` rather than a `revealed`; `Certificate` gains `TLSRefused`; `Facet` and
  `Shadowed` carry the `NotHTTP` → `NoHTTPResponse` rename; `Batch` states that a recorded scope
  dimension is recorded **by content**.
- **[ADR-0007](./0007-drift-is-a-timeline-of-spans.md)'s *"Aperture has five inputs"* paragraph is
  corrected twice.** The count is **six** (`Vantage class` joined at
  [ADR-0017](./0017-exposure-needs-both-legs.md)), and *"both start timelines that did not exist"*
  is true of the qtype set and false of the TLS candidate set.
- **[ADR-0011](./0011-a-facet-is-six-parts.md) is amended in two details.** Its *"a library upgrade
  that widens the offer is estate-wide drift we can name"* describes a world where the library owns
  the offer, which this ADR ends; and `certificate`'s value space gains a third variant, filling the
  hole its own `wrong version number` / `unexpected eof` partition implied.
- **[ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md)'s open question is discharged
  and its worked example corrected.** The three offers it could not place are all parameters, its
  library-as-parameter rule stands with its scope narrowed to what we do not enumerate, and its Go
  upgrade fires **one** message rather than two.
- **`http-exchange` and `resolution-walk` gain declared parameters they did not have** — the ALPN
  offer and HTTP versions spoken; the EDNS option set and advertised buffer; the transport and
  fallback policy. `tls-handshake`'s TLS library parameter is joined by the **declared candidate
  set**, which is scope rather than parameter and lives on the `Batch`.
- **Two prober obligations enter the spec, both discharging false-value risks rather than aperture
  ones.** A truncated DNS answer is never a value. A transport-level refusal is never a zone-level
  value, so no `BADCOOKIE` or cookie-driven `REFUSED` may produce `Lame`.
- **The aperture input list is unchanged at six.** This ticket adds none — which is the answer, and
  is why [#12](https://github.com/winniel123/verge-asm/issues/12) can carry a closed enumeration.
- **What the offers actually *are* is now required v1 spec content and is not decided here** — the
  concrete TLS candidate list, the prober's qtype list, the ALPN list, the EDNS buffer and the
  fallback policy. Opened as its own ticket, blocking #12.
- **The golden corpus inherits the offers.** Under ADR-0021 a leaf version may move on a changed
  declared parameter, so each of these offers is now checkable data rather than an implicit fact,
  and the bidirectional gate applies to them unchanged.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| Every client-side offer is an aperture input | Sweeps in the connect timeout, which ADR-0021 already made a parameter without argument; and it would put dimensions in the scope record whose widening has no per-candidate negative to license |
| Only the TLS candidate set, as a stated exception for TLS | Leaves the reason unstated, so the next facet whose value enumerates an offer gets it wrong; the property is the value's shape, not the protocol |
| ALPN as an aperture input | `http-identity` carries a single global negative, not a set with a hole — and widening the offer moves values on running timelines, which is a `Break` either way, so scope buys nothing and claims a licence it cannot use |
| EDNS buffer size as an aperture input | Its failure is detectable in band from the TC bit, so it can never silently move a value once a truncated answer is refused as a value |
| DNS transport fallback as an aperture input | The edge it produces is `Gap` → value on a running timeline, which ADR-0014 has already ruled is not `revealed` |
| Recording the library version as the aperture dimension | A widening cannot be told from a narrowing by diffing version strings, and ADR-0014 makes diffing named dimensions the whole detection mechanism |
| Recording the library version as the declared parameter and leaving the offer implicit | ADR-0008's rejected hash-of-the-parameters, silent on the change that matters — and it is what hides a TLS-1.0-only listener as `NoTLS` |
| Keeping `NotHTTP` | A value name asserting a property of the listener that our offer decided — the `Host` defect for the fifth time |
| A `NotHTTP` variant for *spoke a version we did not offer* | Unemittable for the residual cleartext h2c case without a second probe #4 has not budgeted; ADR-0013's `unknown` deletion binds |
| Recording the negotiated HTTP version on `http-identity` | *TLS 1.0/1.1 negotiated* one facet across: a function of our own offer that moves estate-wide on a library upgrade |
| An `http-acceptance` facet in v1 | A whole new facet, which ADR-0015 prices at `revealed` plus one message whenever it is wanted — so there is no deadline and no reason to buy it now |
| Leaving `certificate` at `Presented │ NoTLS` | Files SSLv3-only, RC4-only and SNI-required listeners under *not a TLS server*, and contradicts the partition ADR-0011 cited to justify `NoTLS` |
| `TLSRefused` as a `Gap` | We looked and got an answer; `Gap` is reserved for *we did not look*, per ADR-0011's closed-union rule |
| Calling a widened TLS offer `revealed`, as ADR-0007 and `CONTEXT.md` do | `revealed` is an opening kind and no timeline opens; ADR-0017 already priced the symmetric case as a `Break` |

## Amendment — [#62](https://github.com/winniel123/verge-asm/issues/62): the test is asked of the offer, and widening the TLS offer breaks two facets

Recorded by [ADR-0030](./0030-an-offer-is-admitted-on-a-finding-or-on-a-falsity-it-prevents.md),
which wrote out the five offers this ADR left as its last consequence. Writing them exposed two
defects here, both in the TLS half.

**1. The test as stated gives one offer two homes, which rebuilds the trap this ADR disarmed.**
The Decision table's *"An offer is batch scope exactly where **the value** carries a per-candidate
negative"* reads as a question about a facet. The TLS candidate set feeds **two** facets with
different value shapes — `tls-acceptance`'s value is a subset of the set (scope), while
`certificate`'s is a single global negative (parameter of `tls-handshake`). Since `tls-handshake`
feeds both facets, that split makes one widening pay a `Break` on `tls-acceptance` twice: once for
the aperture widening and once for the leaf bump. Same object, one change, doubled — *"they do not
compose; they double"*, verbatim, arriving through the door the *one fact has one home* argument
does not cover.

This ADR half-knew: its Consequences say the declared candidate set is *"scope rather than parameter
and lives on the `Batch`"* with no per-facet split, contradicting its own test. **The consequence
line is right and the test's wording is withdrawn**, replaced by:

> An offer is scope where **any** value it feeds carries a per-candidate negative, and then it is
> scope for **every** batch that makes it — including batches whose own facet carries only a global
> negative. Otherwise it is a declared parameter of the leaf that made it.

The correction changes only the TLS case; ALPN, EDNS and the transport policy stay parameters
exactly as placed above.

**2. *"a `Break` on every `tls-acceptance` timeline in the estate"* understates the price by one
facet.** With the set recorded as scope on the `certificate` batch too, widening the offer `Break`s
`certificate` as well — correctly, since a wider offer genuinely moves `TLSRefused` →
`Presented(chain)` on a legacy box. Every argument above for settling the list wide *before* v1
holds a fortiori.

**3. Two smaller corrections.** *"Every offer the measurement binary makes is enumerated in the job
spec"* has an exception Go imposes: `Config.CipherSuites` is ignored for TLS 1.3, so the TLS 1.3
suite list cannot be declared — ADR-0030 §4 removes it from `tls-acceptance`'s value rather than
letting the library own a per-candidate negative. And *"an unofferable candidate is an absence from
the scope record, visible, and not a silent one"* is discharged by a **build-time offerability
check** (ADR-0030 §5), not by the scope record alone, which would only reveal the narrowing after
the `Break` had fired.

The **aperture input count is unchanged at six**, and *"this ticket adds none"* stands for #62 too.
