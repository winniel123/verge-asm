# A facet is six parts, and every canonical form is a closed union

- **Status:** Accepted
- **Date:** 2026-08-13
- **Ticket:** [#36 Canonical form per facet: when are two observations of one facet equal?](https://github.com/winniel123/verge-asm/issues/36)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[#7](https://github.com/winniel123/verge-asm/issues/7) concentrated the whole comparison problem
in one place on purpose: one `Observation` covers every facet, so adding a facet was supposed to
mean writing a **canonicaliser** and never a drift implementation. That left canonicalisation as
the single site where a mistake manufactures drift estate-wide in one release, and
[ADR-0007](./0007-drift-is-a-timeline-of-spans.md) deferred the actual forms rather than settling
them.

ADR-0007 fixed two things and explicitly not the third. Comparison is over a stored canonical
**value**, structurally, not a hash — the product's entire output is *what* changed. And a
canonicaliser is a **versioned derivation**, so changing one is a `Break` rather than drift. What
it did not fix is the forms themselves, and a `Break` is honest but expensive: *we can always ship
a version bump* is not a licence to get these wrong.

The ticket arrived with two stale premises, both of which made the work larger.
[#35](https://github.com/winniel123/verge-asm/issues/35) had already added **`Lame`** as a fifth
`resolution` outcome, so the ticket's own table was a row short. And the five facets were already
six: [#16](https://github.com/winniel123/verge-asm/issues/16) filed *TLS 1.0/1.1 negotiated* under
`certificate`, which turns out not to be a property of a certificate at all.

## Decision

| Concern | Decision |
| --- | --- |
| Source-shape handling | A **decoder** per `(facet, source)`, versioned separately from the canonicaliser |
| Canonical form | A **closed tagged union** per facet — never a record with optional fields |
| Timeline key | Gains a **facet-defined discriminator**; qtype for `dns-record`, empty for the rest |
| `resolution` | `NameError │ NoData │ Lame │ Shadowed │ Resolved(unordered address set)` — **no CNAME chain** |
| `dns-record` | One timeline **per qtype**; TTL **excluded**; `Shadowed` and `Lame` are values here too |
| `dns-record`, qtype NS | An RRset of `(nameserver, serves │ does-not-serve)` pairs — where partial lameness lives |
| `reachability` | `connected │ refused │ no-response` — three values, projecting to `Reach`'s two |
| `certificate` | `Presented(ordered chain fingerprints, leaf first) │ NoTLS`, on **`Endpoint`** — see [ADR-0027](./0027-a-source-may-admit-without-observing.md) |
| `http-identity` | `Responded(status, Location, WWW-Authenticate, Server, title) │ NotHTTP` |
| Negotiated TLS parameters | **Not `certificate`.** A sixth facet, **`tls-acceptance`** |
| `tls-acceptance` | Accepted versions and accepted ciphers; the **candidate set is batch scope**, never value |
| TLS attempt scope | **Every open `Service`**, opportunistically — no implicit-TLS port list |
| `Endpoint` with no name | `Name` may be **absent** — *the default response to a client that names nothing* |
| `Endpoint` cascade | Closes on **either** leg — its `Name` or its `Service` |
| Differ | **Unversioned**, and may consult nothing but the two values |
| Adding a union variant | **No `Break`** where strictly additive, checked on the corpus; one re-baseline message |
| Multi-measurement values | Decided by the **measurement binary within one batch**, never assembled later |
| Aperture inputs | Now **five** — sources, port tiers, the ownership gate, the qtype set, the TLS candidate set |
| What a facet is | **Six parts**: value space, decoder per source, canonicaliser, differ, discriminator, batch-scope obligation |

## Rationale

### The decoder is a second derivation, and per-source keying already paid for it

ADR-0007 implied one canonicaliser per facet. But a timeline is keyed **per source**, and two
sources deliver one facet in unrecognisably different shapes: our resolver's wire answer against
the operator's zone file, on `dns-record`. One canonicaliser swallowing both shapes means fixing
the zone-file parser moves its version and `Break`s every `dns-record` timeline in the estate —
including the ones our own resolver produced, which nothing touched.

*Amended by [#56](https://github.com/winniel123/verge-asm/issues/56) /
[ADR-0027](./0027-a-source-may-admit-without-observing.md) in its example, not its rule.* This
paragraph originally led with *"a live TLS handshake against a `crt.sh` JSON row"*, and that pair
cannot exist: a CT log entry witnesses issuance rather than presentation, so it can produce neither
variant of this facet's union, and **[measured] `crt.sh`'s JSON carries no certificate fingerprint
at all** while `Certificate` is shared by fingerprint. `crt.sh` therefore has no `certificate`
decoder and no decoder of any kind. The rule is unchanged and better founded on the pair that
survives, which is the one with **measured** churn —
[ADR-0020](./0020-a-conflict-needs-two-enumerable-sources.md)'s *stripper per provider convention,
forever* — and the only two-source facet in v1.

One canonicaliser per `(facet, source)` avoids that and costs more than it saves: the two sources
then produce values in different spaces, and ADR-0007's *source conflict is reported, never
resolved* has nothing to report with. You cannot report a conflict between values you cannot
compare.

So it splits in two. A **decoder** per `(facet, source)` turns whatever the source emits into the
facet's value space; a **canonicaliser** per facet puts a value into canonical form. The value
space is shared, so conflicts stay reportable; the parser churn is confined to the timelines of
the source that churned — which is precisely what per-source keying already bought and nothing had
yet spent. The boundary needs policing rather than trusting: **a decoder that helpfully normalises
is an unversioned canonicaliser wearing a parser's clothes.**

### A closed union, because the negatives are the load-bearing part

Every facet here can come back with something that is not the thing it measures — a name that does
not exist, a port that answered nothing, a 443 that completed no handshake, an 8080 that spoke
bytes that were not HTTP. Modelled as a record with optional fields, all of those collapse into
absent fields and become indistinguishable from *we did not look*. Modelled as a closed tagged
union, each is a value with a name.

This is not tidiness, and [#16](https://github.com/winniel123/verge-asm/issues/16)'s set proves it.
**Plaintext HTTP with no HTTPS** reads `NoTLS` directly. Routed to a `Gap` instead, that signal
returns `not-evaluable` on exactly the estates where it is true — the worst available failure
direction for a security product, and the third time this map has caught that same shape after
`corroborative` silence and `firewalled` versus `internal-only`. `Gap` stays reserved for **we did
not look**, and nothing else.

### `resolution` and `dns-record` divide on walk versus reading

`resolution` answers *does this name exist and where does it point*; `dns-record` records *what an
authority served for a qtype*. One is a walk — it follows CNAMEs and, since
[#35](https://github.com/winniel123/verge-asm/issues/35), queries the delegated authorities
directly — and the other is a reading.

That decides where the CNAME chain lives, and it lives in `dns-record`. Inside `resolution`, a CDN
rotating an intermediate alias closes the span with an **identical address set on both sides**:
drift in the route we took, not in the answer we got. `cname-target-name-error` then reads two
subjects — this `Name`'s CNAME record and the *target* `Name`'s `resolution` — which is what it was
always describing anyway.

Addresses are an **unordered set**, A and AAAA together, because round-robin reordering is not
change. The cost is real and correct: a load balancer changing its rotation is invisible to this
facet.

### The key needed a discriminator, and `reachability` had one all along

`dns-record` cannot be one timeline per `Name`. Under ADR-0007's key there is no room for qtype, so
one value must hold every qtype at once — and then a batch that queried MX but not TXT writes a
value asserting an empty TXT RRset it never measured. That is
[ADR-0009](./0009-verge-core-is-a-union.md)'s `{161}` defect arriving through the key instead of
the port list, which is the second time this exact mistake has been found in a different costume.

So the timeline key gains a **facet-defined discriminator**, empty for four facets and the qtype
for `dns-record`. The generalisation is worth stating because it is evidence rather than
convenience: `reachability` never needed a discriminator **only because its subject `Service`
already carries port and transport**. That is the same fix applied one level up, hand-made, before
anyone named it.

TTL is **excluded** from the value, and as a deliberate loss rather than an oversight. The
authoritative TTL is stable *when we reach the authority*, but the moment any source that is not
our own direct query feeds this facet — a zone file, a passive source — the value is a cache
artefact and diffs every record every run. A field whose honesty depends on which source produced
it is the seam rule inside one canonical value. The pre-migration TTL drop is a genuine signal and
it is not worth that price.

`Shadowed` belongs here as well as on `resolution`: a wildcard synthesises answers for **any**
qtype, so without it every synthesised MX under a wildcard reads as a real record. `Lame` likewise
— we reached the delegated authorities and they refused to serve, so every qtype is equally
unanswerable, and ADR-0006's habit applies verbatim: the thing already has an observed value, and
inventing a state means inventing a transition. Names *beneath* the lame delegation hold a `Gap` on
both facets, unchanged from [#35](https://github.com/winniel123/verge-asm/issues/35).

One wrinkle is accepted rather than smoothed. #35 routed **partial** lameness to `dns-record` "per
nameserver", so the NS qtype's value is an RRset of `(nameserver, serves │ does-not-serve)` pairs
and not a name list. `dns-record` is therefore not one value space but a small family indexed by
its own discriminator, and its differ is per qtype.

### `reachability` keeps the refusal, and names what it measured

TCP connect yields three outcomes: SYN-ACK, RST, and nothing. [ADR-0010](./0010-exposure-composes-two-reaches.md)
gave `Reach` two values and no third, but that is the Derived leg and the observation is free to be
richer. `refused` → `no-response` is a firewall moving from REJECT to DROP — a real change in the
operator's estate, and free to record, because both project to `not-reached`, so `Exposure` does
not move and nothing false is alerted.

The third value is **`no-response`**, not nmap's `filtered`. *Filtered* names a conclusion — that
something filtered it — which a connect attempt cannot establish, and this project has now chosen
the measured word over the taxonomy's word three times running (`Lame`, `no-response`,
`tls-acceptance`).

The honest caveat is stated rather than engineered away: **`no-response` is the one reachability
value that may be about us.** Unlike the DNS case there is no delegation walk to buy attribution
structurally, so nothing infers a broken prober from a run of `no-response`, and a wholly blind
vantage is caught by `Availability`, which already exists and is the right instrument.

### `certificate` was carrying a measurement of our own client

`Certificate` is immutable and shared by fingerprint, so this facet should be nearly free — and it
is: an **ordered list of chain fingerprints, leaf first**, order being on the wire and a server
reordering its chain being a config change.

What was not free is what [#16](https://github.com/winniel123/verge-asm/issues/16) put here. *TLS
1.0/1.1 negotiated* is a function of **our own ClientHello**: upgrade the TLS stack, drop TLS 1.0
from the client's offer, and the value moves on every endpoint in the estate at once with nothing
in the world having changed. And it answers the wrong question — our client picking TLS 1.3 says
nothing about what else the server would have taken, while the operator wants to know whether the
server still **accepts** TLS 1.0.

So server-accepted versions and ciphers become a sixth facet, **`tls-acceptance`**, fed by
[#4](https://github.com/winniel123/verge-asm/issues/4)'s weekly enumeration, and the signal is
renamed **`tls-1.0-accepted`** — ADR-0010 having already ruled that a signal named for something
its evidence cannot carry is the `Host` defect.

The candidate set is the trap, because *which versions and ciphers we offer* decides what the value
can say, and a library upgrade changes it. It does **not** go in the value. **It is the batch's
recorded scope**, which is ADR-0005 and ADR-0009's existing mechanism doing exactly the job it was
built for: a batch that offered nine ciphers can never assert the tenth was refused. That settles
the defect structurally rather than by care, and makes the candidate set the fifth aperture input,
so a library upgrade that widens the offer is estate-wide drift we can name rather than absorb.

TLS is attempted on **every open `Service`**, not on a curated implicit-TLS port list. That deletes
an aperture input rather than pricing one, and ADR-0009 is a fresh and expensive lesson in what
happens to a hand-maintained port table nobody derives. It also makes `NoTLS` honest everywhere
instead of a `Gap` on every unlisted port, and it finds implicit TLS on odd ports, which is exactly
what a small org accidentally leaves listening.
[#31](https://github.com/winniel123/verge-asm/issues/31) measured that the failure partitions
cleanly — `wrong version number` against `unexpected eof` — so the negative result is a real
measurement and not an inference.

*Amended by [#54](https://github.com/winniel123/verge-asm/issues/54) /
[ADR-0025](./0025-an-offer-is-scope-only-where-the-value-enumerates-it.md) in two details.* The
partition cited just above has **two** sides and this ADR gave `certificate` one value for both, so
the facet becomes `Presented(chain) │ TLSRefused │ NoTLS` — a peer that spoke TLS and accepted no
candidate we offered is not *not a TLS server*, and collapsing them files SSLv3-only, RC4-only and
SNI-required listeners under the wrong name. And *"a library upgrade that widens the offer is
estate-wide drift we can name"* describes a world where the library owns the offer: the candidate
set is **declared by us and recorded as wire content**, so a library upgrade cannot widen it, and
widening it ourselves costs a `Break` on this facet rather than a `revealed`. `http-identity`'s
`NotHTTP` is renamed **`NoHTTPResponse`** by the same ADR, and the aperture-input count in the
table above was already **six** at [ADR-0017](./0017-exposure-needs-both-legs.md).

### `http-identity` stops at named, spec-defined fields

[#4](https://github.com/winniel123/verge-asm/issues/4) proposed status, title, `Server`, framework
headers, content-length, a normalised body hash, favicon MMH3, Wappalyzer tech-detect and JARM, and
flagged the body-normalisation function as "small but load-bearing" without writing it.
[#5](https://github.com/winniel123/verge-asm/issues/5) had already killed tech-detect.

**In:** status code; the `Location` header verbatim on 3xx, redirects being unfollowed because the
redirect *is* the finding; the `WWW-Authenticate` scheme list on 401, which
[#31](https://github.com/winniel123/verge-asm/issues/31)'s HTTP-shaped rule reads; the `Server`
header verbatim; and `<title>`.

**Out:** body hash, content-length, favicon hash, tech-detect, JARM.

The line is [#31](https://github.com/winniel123/verge-asm/issues/31)'s own test — every admitted
field is read from a **spec-defined field with a zero-row verdict table** — plus one addition it
did not need: **the set of admitted field names is itself closed and release-coupled.** That is
what excludes `X-Powered-By` and its relatives, since *which headers count* is an open growing list,
while admitting `Server`, which RFC 9110 §10.2.4 defines by name.

The body hash is the one to refuse hardest. Its normalisation function is an unbounded corpus — a
stripper per nonce format, forever — which is ADR-0004's out-of-band tell, and every miss diffs
that endpoint on **every single run**. `<title>` is the genuine close call, since a dynamic title
does the same thing at lower volume; it stays because *this host now serves a different app* is the
`http-identity` half of the map's differentiation and nothing else in the admitted set carries it.

### An `Endpoint` may have no name, and the cascade always had two legs

An address-scope seed yields `203.0.113.10:443` with no `Name`, so there is no `Endpoint`, no
certificate observation, and no SAN harvest — which #4 called the highest-value feedback loop in
the tool. So `Endpoint`'s `Name` may be **absent**, meaning *the default response to a client that
names nothing*. That is not a null in a natural key: it is a real, distinguishable measurement mode
that exists on the wire and that we make anyway.

It exposes something in ADR-0006, and the exposure is narrower than it first looked. That ADR's
rule is already general — *a subject leaves when a component of its natural key leaves* — and it
is its **worked example** that names one component only: *"an `Endpoint` whose `Name` has gone is
not a measurable thing"*. A nameless endpoint has no `Name` to withdraw, so a reader carrying the
example rather than the rule keeps a live endpoint under a dead port forever. Stated explicitly
rather than derived: **an `Endpoint` closes when either its `Name` or its `Service` withdraws**,
and a nameless one has one leg. The `Service` leg was always necessary; it went unnoticed because
a withdrawing `Service` normally arrives alongside a withdrawing `Name`.

### The differ is free because it is constrained

A `Transition` is derived on read and never stored, so the differ sits **outside** stored history
while describing it. Left unversioned and unconstrained, a bug fix in it silently changes how a
two-year-old transition reads with no `Break` — the observer inside the comparison path's
*rendering*. Versioned and composed into the span, improving a diff renderer breaks the estate,
which is the absurdity ADR-0008 exists to prevent.

It is unversioned, and the constraint is what buys it: **a differ is a pure function of exactly two
canonical values and may consult nothing else.** Under that constraint it has no content to move —
what it renders is fully determined by two values whose canonicaliser *is* versioned, so a differ
change is a bug fix in a projection.

The failure it forbids is concrete and tempting. The moment a differ wants to say *the SAN list
gained a **wildcard***, it is classifying; classification is reference data; and it has become a
derivation that must be versioned. That is #31's zero-row verdict table applied to rendering, and
anything richer than a diff is a `Signal`, which is versioned already and costs nothing new.

### Adding a variant to a union is not a reinterpretation

`Lame` is the worked example and it is not hypothetical — #35 added it last week. Under ADR-0008
the golden corpus shows a moved output, the `resolution` canonicaliser's version bumps, and breaks
are uniform with no predicate. Shipping `Lame` would therefore blind every `resolution` timeline in
the estate for a cadence, in exchange for a variant that fires on a handful of names.

A canonicaliser change is **strictly additive** when every corpus row whose output moved previously
produced **no observation at all** — and CI can check exactly that, so this is a mechanism and not
a judgement. Where it holds, no `Break` is needed, because the edge those rows cross is `Gap` →
value and nothing was ever comparable across a `Gap`. Where any moved row previously held a
*value*, it is a reinterpretation and breaks uniformly, unchanged.

This is the same cut ADR-0007 made for aperture, arriving at the canonicaliser from the other side:
**creating values where none existed costs `revealed` or nothing; reinterpreting an input that
already had a value costs a `Break`.** Additive is not free, though — it produces a burst of
first-values across the estate — so it fires **one re-baseline message at the cause**, ADR-0008's
existing mechanism at zero comparison cost.

*Amended by [#42](https://github.com/winniel123/verge-asm/issues/42) /
[ADR-0014](./0014-only-revealed-generalises.md) in two details, neither disturbing the rule.* The
edge those rows cross is a timeline **opening**, not `Gap` → value: a row that previously produced
no observation at all has no span of any kind, and a `Gap` is a span. Both are safe for the same
reason and the rule is unchanged, but they are different edges and #42 turned on telling them
apart. And **"one re-baseline message" names the message *class*, not ADR-0008's trigger** — a
re-baseline fires when a `Derivation` vector moves, and an additive widening moves no vector. One
class, two triggers, and the payloads differ: a vector move carries a difference set computed
across a `Break`, an additive widening carries a count of timelines opened and no comparison at
all.

### What takes two measurements is decided by the prober

`Shadowed` breaks the architecture as stated everywhere else. Deciding it means comparing this
name's answer against the parent zone's **measured wildcard poison signature** — a *different*
observation. Either the canonicaliser is not a pure function of one observation, and now carries a
cross-observation dependency with its own currency and staleness problem inside the comparison
path, or the decision happens earlier.

It happens earlier, as a general rule: **any facet value requiring more than one measurement to
establish is decided by the measurement binary inside a single batch, never assembled afterwards.**
The random-label probe and the name's query happen at one moment, from one vantage, in one batch,
so `Shadowed` is a genuine property of the measurement rather than a conclusion drawn about it.

[#35](https://github.com/winniel123/verge-asm/issues/35) is direct precedent and settles the
objection that this smuggles a derivation into the prober: the delegation walk is exactly this
move, chosen because buying the distinction afterwards is the *whose fault was it* inference that
turns a value into a coverage gap wearing a value's clothes. `NoTLS` and `NotHTTP` join `Shadowed`
and `Lame` in this class.

The cost is real and is not hidden: **the measurement binary is now load-bearing for four facet
values**, so a prober version is an input to what a value *is*.

### #7's rule does not survive; what it protected does

*Adding a facet means writing a canonicaliser* is false after contact with six real ones. A facet
needs a **value space**, a **decoder per source**, a **canonicaliser**, a **differ**, a
**discriminator** (possibly empty), and a **batch-scope obligation** naming what its silence
covers. Six parts, three of them versioned.

The guarantee underneath is untouched and is the part that mattered: **adding a facet still never
means writing a drift implementation.** The fold, `Span`, `Break`, `Gap` and `Transition` are all
facet-agnostic and none of them changed here.

Saying the six parts out loud is the point, because the failure mode is specific: a session adds a
seventh facet, writes a canonicaliser, and silently skips the batch-scope obligation — which is the
`{161}` defect arriving through a facet instead of a port list, for the third time.

## Consequences

- **[`CONTEXT.md`](../../CONTEXT.md) changes in six places.** `Facet` gains the six parts and the
  sixth facet; `Endpoint` gains the absent name and the two-legged cascade; `Certificate` loses any
  implication that negotiated parameters live there; `Shadowed` and `Lame` become values on
  `dns-record` as well as `resolution` and are marked prober-decided; `Span` records that the key
  carries a facet-defined discriminator; and `tls-acceptance` is named.
- **[ADR-0007](./0007-drift-is-a-timeline-of-spans.md) is amended twice.** The timeline key is
  `(subject, facet, discriminator, vantage, source)`, and aperture has **five** inputs rather than
  three — the qtype set and the TLS candidate set join enabled sources, port tiers and the
  ownership gate. Both were already true in substance; neither was written.
- **[ADR-0006](./0006-subjects-leave-by-measurement.md) is amended once, and only in its
  example.** Its cascade rule was already general; what it now says out loud is that an
  `Endpoint` closes on either leg, `Name` or `Service`, because a key component may be absent
  from the start.
- **[#16](https://github.com/winniel123/verge-asm/issues/16)'s signal set changes in one row.**
  *TLS 1.0/1.1 negotiated* becomes `tls-1.0-accepted`, moves from `certificate` to
  `tls-acceptance`, and its evidence moves from the daily handshake to the weekly enumeration. The
  old name stands unrewritten in #16.
- **The measurement binary needs a version and a corpus, and that is now unavoidable.** Four facet
  values are decided inside the prober, so a prober version is an input to what a value *is* — but
  a prober version moves every release for reasons that touch nothing, which is ADR-0008's original
  problem with no golden corpus available, because a prober's corpus is wire transcripts rather
  than structured rows. Opened as its own ticket.
- **The golden-corpus fog patch is narrowed and no longer conditional.** It had been recorded as
  evaporating if [#41](https://github.com/winniel123/verge-asm/issues/41) took the HTTP-shaped rule
  alone. It does not: the prober decides `Shadowed`, `Lame`, `NoTLS` and `NotHTTP` regardless of
  #41. Every *canonicaliser* corpus, by contrast, is structured rows — decoders absorb the wire —
  so the hard half is entirely the prober's.
- **Every open `Service` now carries at least one `certificate` timeline**, most holding `NoTLS`
  forever. Storage, not correctness, and it lands on the retention patch rather than here.
  *Corrected by [ADR-0027](./0027-a-source-may-admit-without-observing.md): this originally read
  "a `certificate` timeline now exists for every open `Service`", which read as a keying claim and
  contradicted this ADR's own rationale.* The facet keys on **`Endpoint`**; what is true per
  `Service` is the **floor**, the nameless `Endpoint`'s timeline, and the count multiplies again
  by names per service.
- **`tls-acceptance` puts a weekly-cadence input under a v1 signal.** ADR-0007's currency rule is
  `k` cadences of the covering `Scan`, so this is legal without new machinery — but
  `tls-1.0-accepted` is now the slowest-moving signal in the set, and a `Gap` opens two weeks after
  the weekly tier stops rather than two days.
  *Amended by [ADR-0028](./0028-a-facets-cadence-is-the-cadence-of-its-exchange.md): "the weekly
  tier" reads as [#4](https://github.com/winniel123/verge-asm/issues/4)'s top-1000 port `Scan` and
  should read "its weekly `Scan`" — the enumeration is a fourth `Scan` over the open `Service`
  population, not a port tier, since `verge-core`'s sensitive-only members are not in top-1000.
  This consequence is otherwise vindicated: `tls-1.0-accepted` is the slowest-moving signal in the
  set, and it is so precisely because this ADR's own "daily handshake" for `certificate` is
  correct.*

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| One canonicaliser per facet, swallowing every source shape | A zone-file parser fix breaks every `dns-record` timeline including our own resolver's — originally argued from a `crt.sh` `certificate` decoder, withdrawn by [ADR-0027](./0027-a-source-may-admit-without-observing.md) |
| One canonicaliser per `(facet, source)` | Two sources' values land in different spaces, and ADR-0007's *report the conflict* has nothing to compare |
| A record with optional fields per facet | Collapses every measured negative into *we did not look*, and `plaintext-HTTP-with-no-HTTPS` goes `not-evaluable` where it is true |
| The CNAME chain inside `resolution` | A rotated intermediate alias closes the span with identical addresses on both sides — drift in our route, not the world's answer |
| One `dns-record` timeline per `Name`, all qtypes in one value | A batch that queried MX and not TXT asserts an empty TXT RRset — ADR-0009's `{161}` defect through the key |
| TTL in `dns-record`'s value | Honest only when we reach the authority; a cache artefact from any other source, and it diffs every record every run |
| `filtered` as the third `reachability` value | Names a conclusion a connect attempt cannot establish; `no-response` names what we measured |
| Negotiated TLS parameters on `certificate` | A function of our own ClientHello — a library upgrade moves every endpoint at once — and it answers the wrong question |
| Our client's offer as a named versioned derivation | Coherent, and buys a `Break` on every Go upgrade to preserve a value that answers the wrong question |
| A curated implicit-TLS port list | Prices an aperture input rather than deleting one, and ADR-0009 is a fresh lesson in hand-maintained port tables |
| A normalised body hash in `http-identity` | An unbounded corpus of strippers — ADR-0004's out-of-band tell — and every miss diffs that endpoint every run |
| `Endpoint` re-keyed on `Service` with an SNI discriminator | Demotes `Endpoint` to a view, forcing #7's rationale for it to be re-argued for `http-identity` too |
| Accepting no certificate facts on address-seeded estates | Kills SAN harvesting, #4's highest-value feedback loop, exactly where the operator gave us least to start from |
| A versioned differ | Improving a diff renderer would break the estate — the absurdity ADR-0008 was written to prevent |
| A uniform `Break` for a strictly additive variant | Blinds every timeline of the facet for a cadence in exchange for a variant firing on a handful of names |
| `Shadowed` assembled by the canonicaliser from two observations | Puts a cross-observation dependency, with its own currency problem, inside the comparison path |
