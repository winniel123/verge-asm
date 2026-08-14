# A subject key is the thing denoted, not the text that named it — and its normalisation may never move

- **Status:** Accepted
- **Date:** 2026-08-14
- **Ticket:** [#89 What is an `Address`'s natural key — the text, or the address it denotes?](https://github.com/winniel123/verge-asm/issues/89)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)
- **Amends:** [ADR-0011](./0011-a-facet-is-six-parts.md) in one detail

## Context

[`CONTEXT.md`](../../CONTEXT.md) says the four subjects *"split on **who supplies the key**"*, and
that the world supplies *"an `Address`'s IP"*. It has never said what form that key takes, and for
IPv4 the omission was close to invisible: a dotted quad has one conventional spelling.

For IPv6 one address has many legal spellings — `2001:db8::1`, `2001:0db8:0000:0000:0000:0000:0000:0001`,
`2001:DB8::1`, `2001:db8:0:0:0:0:0:1` — which is why RFC 5952 exists. This is live **today** rather
than prospectively: AAAA is in the shipped resolution offer on the *absence-makes-the-measurement-false*
limb ([`measurement-offers.md`](../spec/measurement-offers.md) §2), so IPv6 `Address` subjects already
arrive by resolution and are already probed.
[ADR-0049](./0049-an-address-scope-is-family-agnostic-and-the-cap-counts-addresses.md) exposed the
seam rather than causing it, and said so in its own consequences: ruling that *a CIDR is a CIDR* is the
point at which somebody asks what `2001:db8::1` is equal to.

It is a seam in [#6](https://github.com/winniel123/verge-asm/issues/6)'s sense — **a place drift can be
manufactured** — and it is a seam in two places at once, both inside the comparison path.

- **Subject identity.** Two spellings of one address are two `Address` subjects, each holding a
  `Service` subtree, its `Reach` timelines and its `Custody` verdict. A source that changes its
  rendering — a resolver library upgrade, a prober leaf bump under
  [#49](https://github.com/winniel123/verge-asm/issues/49) — retires one subject and enters another.
  That is a withdrawal and a membership message for an event in which nothing moved.
- **A facet value.** `resolution`'s value is *"A and AAAA together, unordered"*
  ([ADR-0011](./0011-a-facet-is-six-parts.md)), compared as a set. Two spellings of one member make
  the set unequal and the `Span` closes.

ADR-0011 already requires every facet to carry a **canonicaliser**, so the second half has an obvious
home. The first does not: `Address` is a **subject key**, and no ADR says a subject key has a
canonicaliser at all — nor what it would mean for one to be versioned, given that moving it re-keys
subjects rather than moving a value.

## Decision

**An `Address`'s natural key is the address, never the text. The text is ours, not the world's.**

| Concern | Decision |
| --- | --- |
| What the key is | **The address** — its **family** and its **octets**, four for IPv4 and sixteen for IPv6 |
| Comparison | **Octet equality**, family-matched. No string comparison anywhere in the model |
| Rendering | RFC 5952 §4 for IPv6, dotted quad for IPv4 — computed **on read**, never stored, never compared |
| Is the rendering versioned? | **No.** It is a pure function of the key and consults nothing else — ADR-0011's differ argument, unchanged |
| Does a subject key get an ADR-0011 canonicaliser? | **No.** A key's normalisation is a different kind of object: **total, specified, and fixed at v1** |
| Is it a leaf in the `Derivation` vector? | **No leaf, no version, no `Break`.** The vector gains nothing |
| What it may consult | **The one value being keyed, and nothing else** — no reference data, no second observation, no declared parameter |
| What enforces that | A **key corpus** in CI whose gate is inverted: **no row may ever move**. Not a bump condition — a build failure |
| `::ffff:0:0/96` (IPv4-mapped) | **Folded to its IPv4 address at the boundary.** One subject, and it is the IPv4 subject |
| `::/96`, `64:ff9b::/96`, `2002::/16`, `2001::/32` | **Not folded.** Every one is a real IPv6 address that routes, or contains `::` and `::1` |
| Ambiguous textual forms | **Refused, never interpreted** — leading-zero, hexadecimal and fewer-than-four-part IPv4 forms have two denotations, so they have no key |
| An address that cannot be keyed | **Not a subject.** It is absent from the `Batch`'s recorded scope, writes no value and no `Gap` |
| `Seed` containment | **The same form.** An address is inside a CIDR iff the families match and the first *n* bits are equal |
| `Custody` and `Vantage class` | Both run on the key. Neither ever sees a spelling |
| `Service` and `Endpoint` | **Inherit for free.** A composed key holds the **subject**, never its rendering |
| `resolution`'s address set | Members are held in the **key form**; the facet canonicaliser composes it and does not restate it |
| The received text | **Not retained.** A source changing its rendering is a fact the model is built not to notice |
| `Address`'s lifecycle | **Unchanged — it still has none.** Nothing here gives it one |

## Rationale

### The world hands us octets, and the text is our own

This is the whole ruling and everything else follows from it.

The ticket poses the choice as *the text a source handed us* against *the address it denotes*, and the
first option dissolves on contact with what the sources actually deliver. **On the path that matters,
there is no text.**

| Source | What arrives | Where a text would come from |
| --- | --- | --- |
| A resolver answer, A | 32 bits of RDATA `[spec]` RFC 1035 §3.4.1 | Whatever printed it |
| A resolver answer, AAAA | 128 bits of RDATA `[spec]` RFC 3596 §2.2 | Whatever printed it |
| A certificate `iPAddress` SAN | An OCTET STRING of 4 or 16 octets `[spec]` RFC 5280 §4.2.1.6 | Whatever printed it |
| The operator's typed `Seed` | Text | The operator |
| The operator's zone file | Text | The operator |

So *the text a source handed us* is a fiction over the flagship path. What a resolver hands us is
sixteen octets; the string `2001:db8::1` is produced downstream by a call in our own binary, and which
string it produces is a property of that call. Keying on it makes subject identity a function of a
rendering step that is not part of the measurement — which is
[ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md)'s test, *can this thing's output move
while the world does not*, answered in the affirmative in the worst place available.

`CONTEXT.md` was already right and had merely been read loosely: **the world supplies the address; the
text is ours.** The two text-bearing sources are both the operator's, and the operator typed a
spelling of something rather than the something itself.

### A canonicaliser is priced; a key normalisation is prevented

This is the question the ticket sharpens, and it is not a question about IP addresses at all.

The tempting answer is uniformity: normalisation is inside the comparison path, so it gets a
version, and moving it costs a `Break`. That is right for a facet canonicaliser and wrong here, and
the difference is what the model can *represent*.

A `Break` is a boundary between two `Span`s **on one timeline**. It says: these two values may not be
compared. That is a sentence about values, and the model has the object to carry it.

Re-keying a subject is not that sentence. It says: this timeline belongs to a different row — the
subject `2001:0db8::1` is retired, the subject `2001:db8::1` is entered, and the timelines beneath
the first are the timelines of the second. The model has **no object for that**. `appeared`,
`returned` and `revealed` are the three named ways in ([ADR-0014](./0014-only-revealed-generalises.md));
`Gap` is the absence of a value; `Break` is the absence of a licence to compare. None of them is
*this subject is that subject*. Carrying a key move would take a sixth thing — a subject merge — and
a subject merge is a lifecycle, which `Address` does not have and which this ticket forbids giving it.

So the two normalisations part on their failure mode, and the rule generalises past addresses:

> A **canonicaliser** is versioned because the model can price it being wrong: it costs a `Break`.
> A **key normalisation** may not be versioned, because the model cannot price it being wrong — so it
> must be **prevented** instead.

That is this project's standing preference stated one level up. ADR-0007 enforced comparability with
a `Break` rather than discipline; ADR-0011 made the TLS candidate set batch scope rather than trusting
care; [ADR-0025](./0025-an-offer-is-scope-only-where-the-value-enumerates-it.md) declared the offer
rather than reading a default. Here the structural move is the opposite shape and the same instinct:
where there is no mechanism to absorb a mistake, remove the ability to make one.

### The bargain: versionlessness is bought by refusing to author the function

*"It may never move"* is not a decision anyone can simply take. A function is fixable only if its
content is fixed, and a function whose content is **ours** is not.

That is the whole reason a facet canonicaliser has to be versioned. It makes choices we author and
will revise: whether TTL is in the value, whether the address set is ordered, whether `TLSRefused` and
`NoTLS` are one variant or two. Every one of those was argued and several have already moved.

A key normalisation for an address makes exactly one choice — fold `::ffff:0:0/96` — and that choice
is **read off a specification rather than authored**. RFC 4291 §2.5.5.2 says what a mapped address is;
we do not get a vote. Everything else in the function is the wire format, which is also not ours.

So the constraint that buys versionlessness is a constraint on *content*, and it is stated as a
prohibition rather than a hope:

> A key normalisation may consult **only the single value being keyed**. Never reference data, never
> another observation, never a declared parameter, never a library's opinion.

That is exactly the constraint ADR-0011 put on the **differ**, and it is worth naming as the same
constraint, because it buys the same freedom for the same reason: a pure function of the thing itself
has no content that can move, so there is nothing for a version to track. ADR-0011's worked failure
transfers verbatim — *the moment a differ wants to say the SAN list gained a **wildcard**, it is
classifying* — and its key-side twin is the moment a key function wants to consult a table of
prefixes to decide what an address *means*.

And the discipline is a mechanism, not a habit. Every named derivation has a golden corpus
([ADR-0008](./0008-derivation-versions-move-on-content.md)); the key normalisation gets one too, and
its gate is **inverted**. ADR-0021's gate is bidirectional — a moved row justifies a bump, a bump with
no moved row fails. Here there is no version to bump, so there is only one direction left and it is
absolute: **a moved row fails the build.** A row is a claim in prose plus its octets, in ADR-0021's own
form, and the claims are sentences like *the mapped form of an IPv4 address keys as that IPv4 address*.

What is honestly left over is stated rather than smoothed: if the function must move anyway, it moves
as a **migration that re-keys the estate**, deliberately outside every mechanism the model has —
expensive, visible, and not something a release does casually. That is the cost of the bargain, and
naming it is better than pretending the case cannot arise.

### The mapped block is a representation; the four that look like it are addresses

`::ffff:198.51.100.1` is the ticket's hardest limb, because a dual-stacked prober may legitimately
report either form for one listener.

It folds, and the ground is that **`::ffff:0:0/96` is the one block defined as a way of writing an
IPv4 address rather than as an address**. RFC 4291 §2.5.5.2 defines it to *"represent the addresses of
IPv4 nodes as IPv6 addresses"*, and no packet is routed to one: a listener answering there answered on
IPv4. Its only producer is a socket API — a dual-stack `AF_INET6` socket — which makes it the same
defect as the text one layer down, our own instrument's internal representation leaking into a key.
Go's `net.IP` holding IPv4 in a sixteen-byte mapped form is the concrete instance waiting to happen.

Refusing the fold has a measurable cost and it lands on the gate. `Custody` is a **total lookup** over
`Seed`s; under an unfolded key, `::ffff:198.51.100.1` is not inside the operator's declared
`198.51.100.0/24`, so the address reads `third-party` and the probing gate **closes on the operator's
own machine**. `Vantage class` fails in the other direction and worse: ADR-0049's every-address test
finds one uncovered address and verifies `internet`, so a dual-stacked internal prober reporting its
own address in mapped form moves the whole estate's observations onto the alerting leg. One listener,
two subjects, two `Reach` timelines and two `Custody` verdicts, decided by which socket saw it.

The fold is **exactly that block and nothing else**, and the four near neighbours lose for two
different reasons:

| Block | Why it is not folded |
| --- | --- |
| `::/96` — IPv4-compatible, RFC 4291 §2.5.5.1 | **Deprecated** by that section, so nothing should produce it and folding it would invent a meaning. Decisively, it **contains `::` and `::1`** — the unspecified and loopback addresses — which would fold to `0.0.0.0` and `0.0.0.1`. That alone settles it |
| `64:ff9b::/96` — NAT64, RFC 6052 §2.1 | A real IPv6 address that routes. The embedded IPv4 address is reachable only *through a translator*, so the two denote different network locations and folding them asserts a reachability we did not measure |
| `2002::/16` — 6to4, RFC 3056 | As above. A distinct address with a distinct path |
| `2001::/32` — Teredo, RFC 4380 | As above |

The line is therefore not *does an IPv4 address appear in the bits* — it does in all five — but
**does the specification define this as a representation or as an address**. One block is a
representation. The rest are addresses that happen to embed one, and folding them would be the model
concluding something about routing from a bit pattern, which is a derivation wearing a key's clothes.

### Containment runs on the key, and it is family-matched rather than family-aware

`Custody` is a total lookup over CIDRs and `Vantage class` re-verifies against address scopes every
batch. Both are containment tests, and containment is natural over the address and unnatural over its
text — `2001:db8::1` and `2001:0db8::1` would answer differently to one CIDR, and the thing answering
is the probing gate.

So a `Seed`'s address scope is held in the same form — family, prefix octets, length — and containment
is: **families equal, first *n* bits equal.** That is one rule with no branch, which is ADR-0049's
family-agnosticism arriving at the level below it. Families never mix, so no cross-family containment
question arises, and after the fold there is no sixteen-octet value denoting an IPv4 address for one
to arise about.

ADR-0049 is **confirmed rather than amended**. `1,024 addresses` is unchanged, `/118` is unchanged, and
the `/128` route by which a v6-only internal prober verifies `internal` now works for a reason it was
previously only assumed to: the prober's address and the declared scope are compared as addresses.

### The composed keys inherit, and there is one function with one implementation

`Service` is `(Address, port, transport)` and `Endpoint` is `(Name, Service)`. Both compose from
subjects already in the estate, so both inherit for free — but only under a prohibition worth writing
down, because the alternative is what a schema does by default:

> A composed key holds the **subject**, never its rendering.

A `Service` keyed on the string `"2001:db8::1"` with a port beside it would reinstate the entire
problem one level up, with a second normalisation site nobody would remember to keep equal. That is
[#6](https://github.com/winniel123/verge-asm/issues/6)'s seam manufactured by a data type.

There is one normalisation, one implementation, and it runs wherever an address enters the model —
each `(facet, source)` decoder that yields addresses, and the `Seed` declaration parse. ADR-0011's
policing rule applies with its direction reversed and its force unchanged: *a decoder that helpfully
normalises is an unversioned canonicaliser wearing a parser's clothes*, and the key-side twin is a
decoder that helpfully renders — hands on a string, and the key function is left to re-parse our own
output.

The one facet consequence follows and is small. `resolution`'s `Resolved(unordered address set)` holds
members **in the key form**, because those members are the very things that admit the `Address`
subjects; a set whose members were not the addresses they admitted would be incoherent. The facet
canonicaliser therefore has nothing to decide here — it composes the key form rather than restating
it, which is why ADR-0011's *six parts* is untouched and a facet gains no seventh part. **The facet
half of the ticket's seam is discharged by the key, not beside it.**

`Endpoint`'s `Name` is the residue and it is named in the consequences: nothing here decides it.

### An ambiguous text form has no key, so it is refused rather than interpreted

The pathological IPv4 spellings the ticket lists fall out of the ruling rather than needing a rule.

`010.1.1.1` is `10.1.1.1` under a strict dotted-quad parser and `8.1.1.1` under the `inet_aton(3)`
family of forms, which read a leading zero as octal, `0x` as hexadecimal, and fewer than four parts as
a packed integer. Two parsers, two addresses, one string.

An identifier with two denotations does not denote, so it has no key. Interpreting it would mean
**choosing** one denotation — and that choice would be ours, authored, revisable, and therefore
exactly the movable content the versionless bargain forbids. Refusing it costs nothing that anyone
wanted: no resolver produces one, because no resolver produces text.

The refusal has to be total, so the domain is stated. The accepted textual forms are strict dotted quad
for IPv4 and RFC 4291 §2.2's forms for IPv6; **everything else is not an address**, and an address that
cannot be keyed is not a subject. It is absent from the `Batch`'s recorded scope, exactly as
[`measurement-offers.md`](../spec/measurement-offers.md) §5.3 already rules for a failed query — no
value, no `Gap`, and currency does the rest. The count is worth surfacing rather than swallowing: a
source that starts emitting unkeyable addresses is a source that has changed shape, which is the one
thing the received text would have told us and the reason we do not need to keep it.

### Where this is thin, stated rather than smoothed

- **The re-key migration is described and has never been performed.** *It moves as a migration rather
  than as a version* is a coherent escape hatch and nobody has designed one, priced one, or written
  what an operator is shown while it runs. The ruling's safety rests on the case not arising, and the
  reason to expect it not to — that the function's content is a specification rather than ours — is an
  argument, not a measurement.
- **No source in this repo has been measured emitting a mapped-form address.** The hazard is derived
  from what a dual-stack socket API does and from Go's `net.IP` representation, both read from
  documentation rather than exercised — the same unmeasured-offerability status
  [`measurement-offers.md`](../spec/measurement-offers.md) carries at its head. The fold is cheap and
  the failure it prevents is a closed probing gate, so it is worth taking on argued grounds; the flag
  is that they are argued.
- **`Name` is left with the identical seam open**, deliberately. This ADR's *rule* binds it — a `Name`
  key is the name, its normalisation is fixed and versionless — but the *function* is not decided
  here, and it needs its own evidence: case-insensitivity, the trailing dot, and A-label against
  U-label are three questions with three specifications behind them, and the operator's zone file and
  our own resolver are two sources that may disagree on all three. Ruling it as by-catch would be
  manufacturing the consensus this map's notes forbid.

## Consequences

- **[`CONTEXT.md`](../../CONTEXT.md) changes in four places.** `Subject` states that a natural key is
  the thing denoted and that its normalisation is fixed rather than versioned; `Address` states the
  key's form, the mapped fold, the rendering and the refusal of ambiguous spellings; `Seed` states
  that an address scope is held in the same form and that containment is family-matched; and
  `Service` states that a composed key holds the subject rather than its rendering, which `Endpoint`
  inherits through it. **No term is added and no term changes meaning.**
- **The `Derivation` vector gains nothing, and this is worth stating because a session will look.**
  There is no `address-key` leaf, no version, and no `Break` cause. ADR-0021's five prober leaves stay
  five; [ADR-0008](./0008-derivation-versions-move-on-content.md)'s composition rule has nothing new
  to absorb.
- **A facet is still six parts.** ADR-0011 is amended in exactly one detail — `resolution`'s address-set
  members are held in the key form — and its rule, its parts and its count are untouched. The
  amendment is a statement about where a fact lives, not a new obligation on a facet.
- **[ADR-0006](./0006-subjects-leave-by-measurement.md)'s cascade rule is untouched and now has a
  stated referent.** *A subject leaves when a component of its natural key leaves* reads on the
  component **subject**, not on a spelling of one, which is what makes `Endpoint`'s two-legged cascade
  well defined for an address that two sources rendered differently.
- **[ADR-0049](./0049-an-address-scope-is-family-agnostic-and-the-cap-counts-addresses.md) is
  confirmed, not amended**, and its consequence naming this seam is discharged. Its `/128` route and
  its every-address containment test both now run on a stated form.
- **A new build-time artefact, and it is the smallest corpus in the project.** The key normalisation
  gets a corpus of `(input, expected octets, claim in prose)` rows, and its gate is one-directional
  and inverted: **a moved row fails the build, unconditionally.** There is no bump that discharges it.
- **A residue is owed a ticket: `Name`'s key.** Case, the trailing dot, and A-label against U-label —
  live today, since the operator's zone file and our own resolver are the model's only two-source
  facet and the pair most likely to disagree on all three. This ADR's rule binds it; its function is
  undecided.
- **Nothing retains what a source said.** The received spelling is not stored, so a rendering change
  anywhere upstream is invisible to the model — which is the property, not a loss. The only diagnostic
  it would have served is covered by the unkeyable-input count above.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **The key is the text a source handed us** — the ticket's first option, and the one with no machinery at all | On the flagship path there **is no text**: an A record is 32 bits, an AAAA record is 128, and a SAN `iPAddress` is an OCTET STRING. The string is produced by a call in our own binary, so keying on it makes subject identity a function of a rendering step that is not part of the measurement. Two spellings then hold two `Service` subtrees, two `Reach` timelines and two `Custody` verdicts for one host |
| **A versioned key canonicaliser — a leaf in the `Derivation` vector, moving costs a `Break`** | The uniform-looking option, and the one the ticket asks about directly. A `Break` is a sentence about two values on one timeline; re-keying is a sentence about which row a timeline belongs to, and **the model has no object for it**. Carrying it would take a subject merge, which is a lifecycle — and `Address` is the one subject with none, which this ticket forbids changing |
| **Fold every IPv4-in-IPv6 embedding** — compatible, NAT64, 6to4, Teredo alongside mapped | `::/96` contains `::` and `::1`, which would fold to `0.0.0.0` and `0.0.0.1`. The other three are real IPv6 addresses that route, reachable only by their own paths, so folding them concludes something about routing from a bit pattern — a derivation wearing a key's clothes |
| **Fold nothing; `::ffff:198.51.100.1` is its own subject** | The probing gate closes on the operator's own machine, since `Custody`'s total lookup finds the mapped form outside their declared `/24`; and ADR-0049's every-address test verifies a dual-stacked internal prober as `internet`, moving the estate onto the alerting leg. One listener, two subjects, decided by which socket saw it |
| **Store and compare the RFC 5952 §4 text** — canonical, human-readable, one spelling per address | It is a correct rendering and a bad key: it re-introduces a string comparison in the gate, makes the key a function of a formatting rule, and would make a rendering fix an estate-wide re-key. RFC 5952 is kept, on the read side, where being a pure projection of the key costs nothing |
| **Retain the source's spelling as provenance beside the key** | A second representation of one fact — the objection that already refused storing a `Transition` and a `Break` — and worse, a stored spelling is a spelling some screen will render, which puts two strings back on one subject. What it would diagnose is covered by counting unkeyable inputs |
| **Interpret the `inet_aton(3)` forms rather than refusing them** | `010.1.1.1` denotes two different addresses under two ordinary parsers, so it does not denote. Picking one would be **our** choice — authored, revisable, movable — which is precisely the content the versionless bargain exists to exclude. Nothing wanted it: no resolver emits text |
| **Give `resolution`'s canonicaliser its own address form, and discharge the facet half there** | Two forms for one thing, and the set could then hold a member that is not the address it admitted. The facet half is discharged **by** the key rather than beside it, which is also why a facet gains no seventh part |
| **Decide `Name`'s normalisation here as well, since the rule is general** | The rule does bind it; the function needs its own evidence — case, the trailing dot, and A-label against U-label, against two sources that may disagree on all three. Ruling it as by-catch of an address ticket is the manufactured consensus the map's notes refuse |
