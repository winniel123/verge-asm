# A source may admit without observing, and a decoder translates shape and never fact

- **Status:** Accepted
- **Date:** 2026-08-13
- **Ticket:** [#56 Which subject does a certificate-transparency observation key on, and is CT a certificate source at all?](https://github.com/winniel123/verge-asm/issues/56)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0011](./0011-a-facet-is-six-parts.md) requires a **decoder per `(facet, source)`** and gives
*"a live TLS handshake against a `crt.sh` JSON row"* as its worked example of two sources
delivering one facet in unrecognisably different shapes.
[ADR-0020](./0020-a-conflict-needs-two-enumerable-sources.md) then walked the v1 source matrix and
found that pair cannot exist: `certificate` is `Presented(chain) │ NoTLS`, a fact about what
something served on the wire, and **a CT log entry names no `Endpoint` and no `Service`**. It
opened this ticket rather than guessing, because [ADR-0015](./0015-the-value-space-is-the-commitment.md)
makes a facet's keying and value space the one class of question with a v1 deadline.

This is a **defect, not an omission**: two accepted decisions name a thing that cannot exist as
named. And it could not be settled without first settling a second defect in the same document —
ADR-0011 describes `certificate` against **two different subjects**, its consequences saying *"a
`certificate` timeline now exists for every open `Service`"* while its rationale argues from
`Endpoint` (*"an address-scope seed yields `203.0.113.10:443` with no `Name`, so there is no
`Endpoint`, no certificate observation, and no SAN harvest"*).

`crt.sh` is not a hypothetical source. It ships as v1's flagship keyless discovery source
([#3](https://github.com/winniel123/verge-asm/issues/3)), cleared by
[ADR-0003](./0003-third-party-source-consent-bar.md) on the absence of terms and throttled by
[ADR-0005](./0005-scan-execution-model.md). So the question was never *do we query CT* but **what
facet and subject its rows feed**.

## Decision

| Concern | Decision |
| --- | --- |
| Subject of `certificate` | **`Endpoint`**, single-valued there for the same reason `http-identity` is |
| Is CT a `certificate` source | **No** |
| Is CT a source of *any* facet | **No.** It observes nothing and has no decoder of any kind |
| Is CT still a `Source` | **Yes** — it **admits** `Name`s, on `authority: inferred` |
| ADR-0012's test | **Sharpened**: *does it admit subjects*, never *does it produce observations* |
| `Citation` for a CT-admitted `Name` | The **`Batch`** that returned it; the chain still terminates at the `Seed` |
| A fifth `Subject` kind for *a certificate that exists* | **Refused** |
| A CT-fed facet on `Name` | **Not in v1.** No deadline — a new facet is `revealed` plus one message |
| Mis-issuance detection | **Out of scope for v1**, on three grounds, one of them measured |
| ADR-0011's decoder rule | **Survives**; only its illustration is withdrawn |
| Cost of this correction | **Zero.** No `certificate` timeline was ever fed by CT, and none has shipped |

## Rationale

### `certificate` keys on `Endpoint`, and SNI is the whole argument

[#7](https://github.com/winniel123/verge-asm/issues/7) put `certificate` on `Endpoint` in its
original facet table and [`CONTEXT.md`](../../CONTEXT.md) has said so ever since — *"what changes
is which certificate an `Endpoint` presents"*. ADR-0011's consequence sentence is the outlier.

The forcing argument is the one that put `http-identity` there, unmodified: **two names on one
`(Address, port)` legitimately present different certificates.** Keyed on `Service`, `certificate`
is multi-valued, which leaves two options and both are failures this map has already refused.
Either the timeline arbitrates between the names — the arbitration
[ADR-0007](./0007-drift-is-a-timeline-of-spans.md) refuses outright — or it records whichever name
the prober reached last, which manufactures drift on every run against every SNI virtual host in
the estate. `Endpoint` exists precisely to make this class of fact single-valued, and the
certificate is the second such fact rather than a new case.

What ADR-0011's consequence was reaching for is true and is preserved: TLS is attempted on **every
open `Service`**, so every open `Service` carries at least one `certificate` timeline — the
**nameless `Endpoint`'s**, the *default response to a client that names nothing* the same ADR
introduced. The sentence was shorthand made true only through that endpoint, and it read as a
keying claim.

### A CT row can produce neither variant of the value space, and cannot even name the certificate

`certificate` is a closed union with two variants, and both are outcomes of a wire exchange:
`Presented(ordered chain fingerprints, leaf first)` is what a listener served us, and `NoTLS` is
what we measured when it served us nothing. **A CT log entry can produce neither.** It witnesses
that a certificate was *issued and logged*; it says nothing about whether anything ever presented
it, and it can never be the evidence for a measured negative. `NoTLS` against a CT row is not a
hard case — it is the tell that the two facts are not in one value space.

The measurement closes it beyond argument. **[measured,
[#3](https://github.com/winniel123/verge-asm/issues/3) §2.2] `crt.sh`'s `output=json` returns
`issuer_ca_id`, `issuer_name`, `common_name`, `name_value`, `id`, `entry_timestamp`, `not_before`,
`not_after`, `serial_number` and `result_count` — and no fingerprint of any kind.** `Certificate`
is *"held as an immutable value and shared by fingerprint"* and the facet's value is *ordered chain
fingerprints*. So the one CT instrument that ships cannot express the key the facet is made of.
The decoder ADR-0011 specified has an **empty domain**: there is no total function from a `crt.sh`
row to that value space, and no amount of care in writing one would produce a partial one either.

That generalises past this facet, and it is the rule worth carrying:

> **A decoder translates a source's *shape* into a facet's value space. It never translates one
> fact into another.** ADR-0011 already said *a decoder that helpfully normalises is an
> unversioned canonicaliser wearing a parser's clothes*; a decoder that **infers the subject** is
> one step worse — it is manufacturing an observation nobody made.

### CT is still a `Source`, because `authority` is admission and admission is not a facet

[ADR-0012](./0012-a-proposer-is-not-a-source.md) says a thing that produces no observations is not
a `Source` and yields `Proposal`s instead. Read literally that makes `crt.sh` a proposer, which is
absurd on this map's own rulings: [ADR-0022](./0022-confirmation-is-singular.md) makes confirming a
`Proposal` **singular and permanent**, so a 400-name CT answer would become 400 confirmations, and
every CT-discovered name would sit unread until clicked.

The literal reading is not what ADR-0012 argued. Its reasoning is entirely about **admission** —
*"`authority` governs whose word is enough to put a subject in the estate"* — and its verdict on a
registry path is that *"it puts no subject in the estate and its silence licenses nothing"*. A
proposer fails **both** limbs. `crt.sh` passes the first outright: a SAN puts a `Name` in the
estate with no operator act, which is not a new claim but the one ADR-0006 already rests on when it
rules that under a wildcard *"a certificate SAN or a zone-file entry is admitted as shadowed"*
while a brute-forced label is discarded. Its `completeness` does real work too, in the closed
direction: `corroborative` is what stops a shorter `crt.sh` `200` reading as *those names are
gone*.

So the test is sharpened rather than reversed:

> **A `Source` is a thing whose word can put a subject in the estate. Observing a facet is the
> usual way it does that, and not the only one.**

ADR-0012's headline overshot its own rationale by one word, and this is the third time on this map
that a term has been found reading as two things — the `Host` defect, in a document written to fix
an instance of it.

### The `Citation` hop is the `Batch`, and the measurement decided that too

`Citation` is *the single-hop link from a subject to the observation that introduced it*, and there
is now no observation. The obvious repair is to route the hop through a `Certificate` value,
mirroring #7's own worked chain — *`vpn-old.example.com` ← that SAN observation ← that endpoint's
certificate ← that service ← that address ← the seed CIDR*. **The missing fingerprint kills it**:
with no fingerprint on the row there is no `Certificate` to point at, and reconstructing one means
a second fetch per row at 5 req/min against a source measured at 4 failures in 8 identical
requests. That is not a decoder, it is a second source.

The hop is the **`Batch`** — already Observed, already recording source, scope, vantage and time,
and already fired for `crt.sh`. Nothing is lost: an admitting source's entire content, once the
identities are read off, *is* the set of names it admitted, and recording that set as citations to
its batch preserves #7's stated payoff verbatim — *"if a source turns out to be bad, identification
by traversal of everything it introduced"*. The chain terminates where #7 requires, because a
`crt.sh` batch's recorded scope is the name-scope `Seed` its query was built from.

The one thing this leaves — a `Name` in the estate with no facet timeline of its own — is not a hole
and ADR-0006 measured why: *"since every known name is re-resolved every cycle, the premise that a
certificate-discovered name has only `corroborative` sources bearing on it is false for every name
outside a wildcard."* A CT-admitted name acquires a `resolution` timeline from our own `enumerable`
resolver within one cadence, and leaves by Name Error like any other. **CT needs no timeline for
the model to work, and never had one.**

### The two rejected subjects, and why the fifth kind is the worse of them

**A `certificate` observation on every `Endpoint` whose `Name` is a SAN** asserts presentation
nobody measured — the mirror image of the no-false-absence rule this map has enforced eight times,
and the more dangerous direction because a fabricated *presence* is not caught by any scope record.
It fails structurally before it fails morally: an `Endpoint` is `(Name, Service)` and a SAN names no
`Service`, so the fan-out would have to invent one per open port, fabricating subjects. And it
would put a `corroborative` value on the same key as our own `measured` handshake, where under
per-source keying the CT timeline **never closes** — so every ordinary certificate rotation would
report a permanent conflict.

**A fifth `Subject` kind for *a certificate that exists*** fails harder. A `Subject` is *anything an
observation can be about*, each with its own lifecycle; #7 made `Certificate` a value on the ground
that it is signed, therefore immutable, therefore cannot drift, and a subject whose every span holds
the same value forever is a timeline with nothing in it. Worse, CT is **append-only**, so such a
subject could never leave — which is ADR-0006's opening sentence verbatim, *"an append-only
inventory, which is half a drift product"*. And it would seat in the estate a subject the operator
has no `Custody` of by construction, since a mis-issued certificate is by definition somebody
else's act and [ADR-0013](./0013-custody-is-control-and-extends-by-declaration.md) left `Custody`
no third value.

### Mis-issuance detection is out of v1, and the ticket's premise about why was wrong

*A certificate exists for your name that nothing in your estate presents* is a real finding and it
is the reason anyone wants CT in this facet. #48 refused it on two grounds and this ticket was
opened believing **only the keying ground was fixable here**. That is backwards, and the correction
is the useful part:

- **The keying ground is confirmed, not repaired.** The fix is that CT holds **no timeline at all**
  — no facet, no value, no `Span`. A `Signal` is *a named, versioned rule evaluated over
  observations*, and there are none to evaluate. Settling the subject did not unlock the rule; it
  removed the last thing a rule could have read.
- **#48's conflation ground stands untouched.** CT is append-only, so a certificate for a long-dead
  name sits in the log forever and the rule fires on every historical certificate in the estate.
  ADR-0015 licenses *commonness* and expressly distinguishes **conflation**, which is this: *a
  certificate was mis-issued* and *a certificate outlived its name* are opposite facts under one
  predicate.
- **A third ground, measured, that #48 did not have.** The rule is a set difference between logged
  and presented certificates, and **the join key does not exist on the instrument that ships** —
  `crt.sh` returns no fingerprint. The one CT instrument that does return one, SSLMate Cert
  Spotter's `cert_sha256`, was **excluded on terms by ADR-0003 for failing limb 2**, its
  unauthenticated tier being scoped *"for personal or evaluation purposes"*.

So this is genuinely deferrable rather than merely deferred, and the price is written down: it
needs a **new facet** on `Name` recording issuance, which ADR-0011's strictly-additive rule and
ADR-0015 price at **`revealed` plus one message with no `Break`**. Whoever picks it up inherits one
warning: a set-valued monotone facet fed by an instrument with a documented **999-row cap that
truncates silently under an HTTP 200** is [ADR-0009](./0009-verge-core-is-a-union.md)'s `{161}`
defect arriving through the *value* instead of the key.

### The decoder rule survives, and is better founded without its example

ADR-0011's sentence names two decoder pairs and only the first is fictional. The second — *"our
resolver's wire answer against the operator's zone file"* — is live, and ADR-0020 confirmed it as
the **only** two-source facet in v1. It is also the pair with **measured** churn: ADR-0020 refused
to read the zone's records precisely because that decoder needs *a stripper per provider
convention, forever*. So the argument for splitting decoder from canonicaliser keeps the instance
that actually pays for it, and loses the one that never could.

### The correction is free, which is why it had a deadline and no cost

ADR-0015 gives keying and value-space questions the only v1 deadline, because getting one wrong and
correcting it later moves the output of rows that already produced observations. This one is
discharged in the **cheap direction**: no `certificate` timeline was ever fed by `crt.sh`, nothing
has shipped, and removing a source that could not have produced a value is not a widening, a
narrowing, or a reinterpretation. There is no `Break`, no `revealed`, and no re-baseline message.
Note this is **not** ADR-0009's pre-release exemption being taken — that was refused as *vacuous
rather than waived*, and nothing here needs it: the correction would cost nothing on a shipped
install either.

## Consequences

- **[`CONTEXT.md`](../../CONTEXT.md) is amended in four entries.** `Source` records that a source
  may admit without observing and states the sharpened test; `Endpoint` records that the presented
  certificate chain is single-valued there beside HTTP identity; `Certificate` records that
  certificate transparency is not a source of the `certificate` facet; `Citation` records that
  where a source admits without observing, the hop is its `Batch`.
- **[ADR-0011](./0011-a-facet-is-six-parts.md) is amended in three places** — its `certificate` row
  names `Endpoint`, its decoder worked example loses the `crt.sh` pair, and its consequence *"a
  `certificate` timeline now exists for every open `Service`"* is corrected. The six parts, the
  closed unions and the decoder rule itself are untouched.
- **[ADR-0012](./0012-a-proposer-is-not-a-source.md)'s test is sharpened, not reversed.** Its
  ruling on registry proposers is unaffected — they fail the admission limb as squarely as they
  failed the observation one — and its own rationale is what the sharpened test is taken from.
- **[ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md) loses `crt.sh` from two
  lists**, both of which assumed it feeds a timeline. Its argument survives on the operator's zone
  file, which is the other member of both.
- **`certificate` timelines are per `Endpoint`, so retention is larger than the fog patch prices
  it.** [#36](https://github.com/winniel123/verge-asm/issues/36) recorded *most `certificate`
  timelines hold `NoTLS` forever*, counted per `Service`; the count multiplies again by names per
  service, and the nameless `Endpoint` is the floor rather than the whole.
- **`crt.sh` is the first `Source` in the model with no facet**, so any surface enumerating what a
  source contributes must not assume a timeline. It contributes `Name`s and a `Batch`, which is
  exactly what [#28](https://github.com/winniel123/verge-asm/issues/28)'s coverage **fill** half
  already needs from it — note this does **not** move it to the propose half, since it admits
  without asking anyone.
- **Two documents disagree on which tier feeds `certificate`, and this ADR does not settle it.**
  ADR-0011 has TLS attempted opportunistically on every open `Service` and calls `tls-1.0-accepted`
  *"the slowest-moving signal in the set"* on `tls-acceptance`'s weekly cadence, while
  [ADR-0014](./0014-only-revealed-generalises.md) says *"TLS is attempted on the weekly tier; its
  `certificate` timeline opens six days later"* and `CONTEXT.md` repeats it. It decides
  `certificate`'s currency bound and therefore when a `Gap` opens under the expiry signals.
  Recorded as stale on one side or the other rather than guessed at, and opened as its own ticket.
  *Settled by [ADR-0028](./0028-a-facets-cadence-is-the-cadence-of-its-exchange.md): ADR-0011 is
  vindicated and ADR-0014's example is withdrawn. The handshake is a step in the exchange that
  produces `reachability`, so there is no single tier — `certificate` inherits the `Service`'s.*
- **Nothing about the discovery aperture moves.** `crt.sh` ships enabled exactly as ADR-0003 left
  it, admitting exactly the names it admitted before. This ADR changes what we say it does, not
  what it does.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| `certificate` keyed on `Service` | Multi-valued under SNI — it either forces the arbitration ADR-0007 refuses or records whichever name was probed last, manufacturing drift on every SNI vhost every run |
| CT feeds `certificate` on every `Endpoint` whose `Name` is a SAN | Asserts presentation nobody measured, fabricates `Endpoint`s from SANs that name no `Service`, and pairs a never-closing `corroborative` timeline against our handshake so every rotation reports a permanent conflict |
| A fifth `Subject` kind — *a certificate that exists* | Immutable, so its timeline holds one value forever; append-only, so it can never leave — ADR-0006's *append-only inventory, which is half a drift product*; and it seats a subject in the estate that `Custody` has no value for |
| CT gets its own issuance facet on `Name` in v1 | ADR-0015 prices a new facet at `revealed` plus one message, so there is no deadline — and a monotone set value fed by a 999-row cap that truncates silently under an HTTP 200 is the `{161}` defect through the value |
| `crt.sh` is a proposer, per ADR-0012 read literally | ADR-0022 makes confirmation singular and permanent, so a 400-name answer becomes 400 clicks and every CT name is read by nothing until one lands |
| Route the `Citation` through a `Certificate` value, per #7's worked chain | **[measured]** `crt.sh`'s JSON carries no fingerprint, and `Certificate` is shared by fingerprint — reconstructing one is a fetch per row at 5 req/min against a source measured at 4 failures in 8 identical requests |
| Ship mis-issuance detection in v1 | Three grounds: CT holds no timeline for a rule to read, append-only makes it conflated rather than merely common, and the join key does not exist on the instrument that ships |
| Reach the join key via SSLMate Cert Spotter's `cert_sha256` | ADR-0003 already excluded its unauthenticated tier on terms — *"for personal or evaluation purposes"* fails limb 2 for the modal operator |
| Keep ADR-0011's `crt.sh` decoder example and note it as aspirational | The pair has an empty domain; an example that cannot be built is not aspiration, and the surviving pair is the one with measured churn |
| Drop `crt.sh` from `Source` entirely and treat CT names as a `Seed`-like declaration | A `Seed` is the operator's assertion of where the estate ends; CT is a third party's word, which is exactly what `authority: inferred` exists to grade |
