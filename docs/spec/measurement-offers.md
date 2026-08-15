# What the measurement binary offers

- **Status:** Accepted — spec content for [#12](https://github.com/winniel123/verge-asm/issues/12)
- **Ruling:** [ADR-0030](../adr/0030-an-offer-is-admitted-on-a-finding-or-on-a-falsity-it-prevents.md)
- **Ticket:** [#62 What does the measurement binary actually offer, now that a default may not stand in for a declaration?](https://github.com/winniel123/verge-asm/issues/62)

[ADR-0025](../adr/0025-an-offer-is-scope-only-where-the-value-enumerates-it.md) ruled that **every
offer the measurement binary makes is enumerated in the job spec and passed to the library
explicitly**, and that the `Batch` records what went on the wire. It did not say what the offers
are. This document is the enumeration. It is deliberately a separate file from the ADR that rules
it, because it is a **list that will be revised** and an ADR is a decision that will not.

**Every row carries the evidence standard it rests on, and the standards are not the same.** This
follows [#21](https://github.com/winniel123/verge-asm/issues/21)'s *publish the weak tier rather
than smoothing it*: the DNS retry budget in §5 rests on nothing but judgement and says so, next to
rows resting on an RFC.

Three marks are used throughout:

| Mark | Means |
| --- | --- |
| `[spec]` | Attested by the protocol's own specification |
| `[owner]` | Attested by the party that owns the thing — the library's own documentation, or the protocol community's coordinated position |
| `[thin]` | Chosen. No attestation, no measurement. Revisable, and the revision price is stated |

**Offerability is unmeasured in this repo.** Every claim below about what Go's `crypto/tls`,
`net/http` and DNS client can put on the wire is read from those packages' documentation, not
exercised — no Go toolchain has been run against this repo, and
[#31](https://github.com/winniel123/verge-asm/issues/31) could not even start Docker. §1.4's
**build-time offerability check** is what converts that whole column from `[owner]` to measured, on
the first build. Until it runs, read the column as claimed.

---

## 1. The TLS candidate set

This is the one list with a deadline. Widening it after v1 `Break`s every `tls-acceptance`
timeline **and** every `certificate` timeline in the estate (ADR-0030 §3); the other four lists
cost `revealed` plus one message or less. So this list is settled first and settled **wide**.

### 1.1 One list, two exchanges

The declared candidate set is **one list**, carried by both TLS exchanges the binary makes:

| Exchange | Purpose | Handshakes | Records the set as |
| --- | --- | --- | --- |
| The **`certificate` handshake** | Read the presented chain | One per `Endpoint` | Recorded scope of its `Batch` |
| The **`tls-acceptance` enumeration** | Enumerate what the listener accepts | Several per `Service` (§1.5) | Recorded scope of its `Batch` |

They are keyed here to **purpose, not tier** — which scan tier each runs on is
[#61](https://github.com/winniel123/verge-asm/issues/61)'s question and nothing in this document
turns on the answer. If #61 merges them into one batch, the ruling degenerates gracefully: one
batch, one candidate set, and §3's no-ALPN rule is unaffected because ALPN belongs to a different
leaf.

ADR-0025 said the two "need not match". They do match, and the reason is not tidiness. The whole
forcing measurement behind ADR-0025 is that a narrow offer on the `certificate` handshake hides a
TLS-1.0-only listener as `NoTLS`; a narrow offer on the enumeration hides the same listener from
`tls-1.0-accepted` directly. Two lists means two chances to make the same mistake, on two facets,
against one box. The cost objection — the enumeration is many handshakes — is real and is answered
by §1.5's enumeration strategy, not by a shorter list. **The cost argument argues about the
exchange, never about the offer.**

### 1.2 Protocol versions

| Candidate | Limb | Evidence |
| --- | --- | --- |
| **TLS 1.0** | Acceptance is a finding | `[spec]` RFC 8996: *"TLS 1.0 MUST NOT be used. Negotiation of TLS 1.0 from any version of TLS MUST NOT be permitted."* Reads the v1 signal `tls-1.0-accepted` |
| **TLS 1.1** | Acceptance is a finding | `[spec]` RFC 8996 covers 1.1 in the same prohibition. **No v1 rule reads it** — recorded anyway, per ADR-0015: the value space is the commitment and the rule set is free forever |
| **TLS 1.2** | Absence makes the measurement false | `[owner]` Without it the modal listener accepts nothing and reads `TLSRefused` |
| **TLS 1.3** | Absence makes the measurement false | `[owner]` A TLS-1.3-only listener — increasingly the default on managed load balancers — would read `TLSRefused`, a false finding in the worst direction |

**SSLv3 is not declared.** Go removed SSLv3 and cannot offer it at any setting (ADR-0025,
inherited, `[owner]` and unmeasured here). Declaring a candidate known to be unofferable puts a row
in the job spec that provably never reaches the wire, which is
[ADR-0013](../adr/0013-custody-is-control-and-extends-by-declaration.md)'s deleted `unknown` — *a
value nothing can emit is worse than an invented state, because it reads as a real distinction
forever*. ADR-0025's *"an unofferable candidate is a visible absence from the scope record"*
describes a **failure mode staying visible**, not a licence to declare vapour.

**The stated cost.** An SSLv3-only, RC4-only or 3DES-refusing-everything-else listener lands in
`TLSRefused`, and v1 cannot tell those three apart. `TLSRefused` read together with the batch's
recorded candidate set *is* the finding — *the peer spoke TLS and refused all of this* — and that is
the whole of what v1 says about the most legacy boxes in an estate.

### 1.3 Cipher suites

**Suites are declared for TLS 1.0–1.2 only.** Go's `Config.CipherSuites` is documented as ignored
for TLS 1.3, so the three TLS 1.3 suites are the library's choice and not ours `[owner]`. Under
ADR-0025's own rule they therefore **may not sit inside `tls-acceptance`'s value**: a per-candidate
negative over candidates we did not choose moves estate-wide on a library upgrade with nothing in
the world having changed, which is *TLS 1.0/1.1 negotiated* a third time, arriving through the
scope record where nobody is watching. And no such row would earn its place anyway — all three TLS
1.3 suites are AEAD, and none is a finding.

So under TLS 1.3, `tls-acceptance` records **that the version was accepted, and nothing about
suites**. See ADR-0030 §4 and the Out-of-scope entry.

Declared suites, by Go constant name. Every row is `[owner]` for offerability and carries its own
attestation for the limb.

**Limb 1 — acceptance is a finding:**

| Suite | The finding |
| --- | --- |
| `TLS_RSA_WITH_3DES_EDE_CBC_SHA` | `[spec]` RFC 8429 (BCP 227) deprecates 3DES; RFC 8996 §1 names 3DES among TLS 1.0's defects. Sweet32 |
| `TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA` | As above, with forward secrecy — a distinct configuration |
| `TLS_RSA_WITH_AES_128_CBC_SHA` | `[spec]` No forward secrecy (static RSA key exchange) **and** non-AEAD; RFC 8996 §1 cites *"absence of AEAD ciphers"* |
| `TLS_RSA_WITH_AES_256_CBC_SHA` | As above |
| `TLS_RSA_WITH_AES_128_CBC_SHA256` | As above |
| `TLS_RSA_WITH_AES_128_GCM_SHA256` | `[spec]` AEAD, but no forward secrecy — the static-RSA finding alone |
| `TLS_RSA_WITH_AES_256_GCM_SHA384` | As above |
| `TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA` | `[spec]` Non-AEAD with forward secrecy — the CBC/Lucky13 family |
| `TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA` | As above |
| `TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256` | As above |
| `TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA` | As above, ECDSA certificate |
| `TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA` | As above |
| `TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256` | As above |

**Limb 2 — absence would make the measurement false.** These are not findings. They are offered so
that a correctly-configured listener accepts *something* and does not read `TLSRefused`:

| Suite | Why its absence would lie |
| --- | --- |
| `TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256` | The modal TLS 1.2 configuration, RSA certificate |
| `TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384` | As above |
| `TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256` | The modal TLS 1.2 configuration, ECDSA certificate. Omitting the ECDSA variants makes every ECDSA-certificate host read `TLSRefused` |
| `TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384` | As above |
| `TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256` | Preferred by listeners without AES hardware acceleration |
| `TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256` | As above, ECDSA certificate |

**Not declared, and why:**

| Candidate | Why not |
| --- | --- |
| RC4 suites | Unofferable — ADR-0025 records that Go cannot speak RC4 at any setting `[owner]`, inherited and unmeasured here. Lands in `TLSRefused` |
| NULL, EXPORT, anonymous suites | Not implemented by Go at any setting `[owner]`. **This is the sharpest missing capability in the list**: *does this listener accept a NULL cipher* is the first question a security reviewer asks, and v1 cannot ask it |
| The three TLS 1.3 suites | Not ours to declare (above) |

### 1.4 The declared set is literal, and CI fails if it stops being offerable

The tempting form is a **derived** set — `tls.CipherSuites() ∪ tls.InsecureCipherSuites()`, in the
manner of [ADR-0009](../adr/0009-verge-core-is-a-union.md)'s `verge-core`, a definition that cannot
fail rather than a list that can go stale. It is refused, and the reason is the whole point of
ADR-0025: **a derived set's content moves when Go moves**, so a Go upgrade would silently widen or
narrow the offer and `Break` every `tls-acceptance` timeline for a dependency reason. That is
ADR-0025's *"a Go upgrade cannot widen the TLS offer"* made false again, through the one door it
did not check.

So the declared set is **literal**. And a literal set has the opposite failure — a Go upgrade that
drops a suite makes a *declared* candidate unofferable, at which point the wire record narrows and
every `tls-acceptance` timeline moves for a reason nothing in the model names.

> **The build fails if any declared candidate is not offerable by the linked library.**

This is [ADR-0021](../adr/0021-a-version-leaf-is-a-decision-not-a-binary.md)'s bidirectional corpus
gate applied to a declared offer: a version that moves for nothing fails the build as loudly as an
output that moves for free, and now so does an *offer that stops going out*. A Go release dropping
3DES becomes a build failure and a deliberate, priced narrowing — never a silent one. It is also
what discharges the unmeasured-offerability caveat at the top of this document, on the first build.

**The residue, stated.** The check is one-directional: declared ⊆ offerable. A suite Go gains that
we did not declare is simply not asked about. That is a narrowing relative to the world, it is the
correct default, and it costs a deliberate widening to fix.

### 1.5 The enumeration strategy, and why it makes one list affordable

Naive enumeration is one handshake per candidate — four versions by nineteen suites, and #4 §6
budgets **5 handshakes/s per host**. That is the cost objection to a wide list, and it is answered
by the strategy rather than by the list.

- **Versions:** one handshake per version with `MaxVersion` pinned. Four handshakes.
- **Suites, per version 1.0–1.2:** iterative narrowing — offer every candidate, record the selected
  suite, remove it, repeat until the listener refuses. That costs *accepted + 1* handshakes, not
  *candidates*. A modern TLS-1.2 listener accepting six suites costs seven; a listener refusing the
  version costs one.

The final round is what licenses the negatives: the remaining suites were offered **together** and
all refused, so each is a per-candidate negative honestly obtained.

One caveat the glossary already anticipated: a listener may refuse an ECDSA suite because its
certificate is RSA, not because the suite is disabled. `tls-acceptance` records *accepted*, which
is the measured verb — *supported* is the capability claim the measurement cannot carry
([`CONTEXT.md`](../../CONTEXT.md)) — so this is correct rather than a defect.

### 1.6 SNI is not a candidate — it is the subject

The `certificate` handshake sends **SNI equal to the `Endpoint`'s name**, and the nameless
`Endpoint` sends **no SNI extension**. This is not an offer and does not enter any candidate set:
`certificate` keys on `Endpoint` precisely because two names on one `(Address, port)` legitimately
present different chains ([ADR-0027](../adr/0027-a-source-may-admit-without-observing.md)), so SNI
selects *which subject we are measuring* rather than *what we are asking about it*. A session
reaching to put SNI in the candidate set is proposing to make the timeline key a scope dimension.

A listener requiring SNI, probed through the nameless `Endpoint`, lands in `TLSRefused` — which is
the bucket ADR-0025 created it for.

### 1.7 Neither the offer nor the enumeration is operator-configurable in v1

#4 §9's eighteen knobs include the enumeration **cadence** (which is #61's) and do not include the
candidate **set**. It stays out of the operator's hands, and the reason is
[ADR-0009](../adr/0009-verge-core-is-a-union.md)'s generalised one step: *a port the operator can
hide is a signal the operator can silence* becomes **an offer the operator can narrow is a finding
the operator can silence**. Narrowing the TLS offer is the one edit that would turn off
`tls-1.0-accepted` while leaving every screen reporting normally.

---

## 2. The prober's qtype set

Adding a qtype after v1 opens a `dns-record` timeline that did not exist, because the qtype is a
**discriminator in the timeline key** (ADR-0011; ADR-0025 §"An aperture widening `Break`s…"). So it
costs `revealed` plus one message and no `Break`. The bar is therefore ordinary, and this list may
grow.

The only enumerated qtype list previously in this repo is
[`docs/research/passive-discovery-sources.md`](../research/passive-discovery-sources.md) §3.1, from
**[#3](https://github.com/winniel123/verge-asm/issues/3)** — #62's body cites it as #13, which is
the reNgine trial and contains no such list.

**v1 queries seven qtypes**, explicitly and never as `ANY` — RFC 8482 permits a server to return a
subset of RRsets for `ANY`, so an `ANY` answer cannot license an absence `[spec]`:

| qtype | Limb | Evidence |
| --- | --- | --- |
| **A** | Absence makes the measurement false | `[spec]` The address set is `resolution`'s value and feeds `Address`, `Custody` and every probing decision |
| **AAAA** | Absence makes the measurement false | `[spec]` ADR-0011 makes the address set *"A and AAAA together, unordered"*, so omitting AAAA makes `resolution` wrong rather than partial. #3 §3.1: v6-only exposure is routinely forgotten |
| **CNAME** | Both | `[spec]` `resolution` is a **walk** that follows CNAMEs (ADR-0011), and `cname-target-name-error` is a v1 signal ([#35](https://github.com/winniel123/verge-asm/issues/35)) |
| **NS** | Acceptance is a finding | `[spec]` `lame-delegation` is a v1 signal and the NS qtype holds the `(nameserver, serves │ does-not-serve)` RRset where partial lameness lives (ADR-0011). RFC 8499 §7 |
| **SOA** | Absence makes the measurement false | `[spec]` [ADR-0020](../adr/0020-a-conflict-needs-two-enumerable-sources.md) requires the zone-file batch to record the **zone**, not the registrable domain — *a zone stops at a delegation* — and SOA is what establishes whether a name is its own zone. RFC 2308 |
| **MX** | Acceptance is a finding | `[owner]` #3 §3.1: MX hosts are themselves exposed assets. ADR-0011's own worked hazard is *collapse NODATA into Name Error and every MX-only name in the estate retires*, which presupposes MX is queried |
| **TXT** | Acceptance is a finding | `[owner]` **and this is the thinnest admitted row.** #3 §3.1 measures TXT as the richest single qtype — SPF `include:`/`ip4:` enumerate third parties and often owned space; DMARC yields posture. **No v1 signal reads it.** Admitted anyway under ADR-0015: the field is decided independently of whether a rule reads it |

**The wildcard control probe runs this same set, and mints no sixth list.**
[ADR-0066](../adr/0066-a-control-probe-is-generated-under-a-names-parent-and-that-population-is-aperture.md)
rules that `wildcard-discrimination` generates its control labels under the **parents** of the
`Name`s in the batch's resolution scope, over the seven qtypes above — because `Shadowed` is
committed on `dns-record` for *any* qtype, so a three-qtype control probe would leave synthesised
MX, TXT, NS and SOA answers recorded as the name's own records. The **population** of probe sites is
not an offer and is not enumerated here: it is a function of the batch's scope, held on the `Batch`
as the **seventh aperture input**. This document stays at **five offers**.

**The seven earn their keep a second way, and it is measured.**
[ADR-0068](../adr/0068-a-wildcard-is-discriminated-only-where-its-synthesis-is-determinate.md) reads
the control probe's answers **per `(qtype, RR type)` component** and discriminates only where a
component held still across every control label. **[measured]** 2026-08-14, `appspot.com`'s only
determinate *positive* component in the seven is **MX** — its A and AAAA both rotate — so under the
withdrawn three-qtype clause that parent would have no positive determinate component at all and
every name beneath it would be suppressed. The qtype set is still one offer and this list does not
move; the widening simply pays twice. The **match predicate** that reads them is a declared
parameter of the leaf, not an offer, and has no row here.

**And the control labels themselves are not an offer either.**
[ADR-0069](../adr/0069-a-control-label-is-one-label-and-the-set-must-falsify-label-independence.md)
values the leaf's last unvalued parameter — ~~**5 random labels + 1 structured label**~~ **9 random
labels + 1 structured label** ([#115](https://github.com/winniel123/verge-asm/issues/115) raises the
count on a measured mechanism; the construction is untouched), each
**exactly one label**, the structured one being `<a>-<b>-<c>-<d>` over a random RFC 5737
documentation address, and **every one of the ~~six~~ ten runs the seven qtypes above**, since a component
is defined over *all n* labels and a label covering fewer leaves that definition ill-formed. Nothing
about a control label is a candidate on the wire — it is a label sequence we construct, admitting
and citing nothing — so this document stays at **five offers**. The qtype count in this table is the
only thing the control probe reads from here, and it is unchanged.

**Not queried in v1:**

| qtype | Why not |
| --- | --- |
| **CAA** | **Its only recorded rationale has been withdrawn.** #3 §3.1 admits CAA for *"a mismatch between CAA and observed CT issuance is a drift signal"* — and [#56](https://github.com/winniel123/verge-asm/issues/56) ruled CT mis-issuance detection and **any CT-fed facet** out of scope. On its own, *the CAA record changed* proves too much: it is true of every qtype. Named loser, and it is the row a session would otherwise carry in by inertia |
| **PTR** | Two grounds. #3 §3.1 admits it only *"inside ranges the org owns"*, and [#26](https://github.com/winniel123/verge-asm/issues/26) measured that population at under 1% while [ADR-0013](../adr/0013-custody-is-control-and-extends-by-declaration.md) made the modal entry point a **name** scope holding no address scopes at all. And PTR keys on an `Address`, not a `Name`, so `dns-record` would need a second subject — a modelling change this ticket may not make unilaterally. Recorded as fog |
| **SRV** | Requires guessing `_service._proto` labels, which is name-guessing, which is what #3 §3.2's wildcard poison-signature machinery exists to contain. No attestation, no signal |
| **DNSSEC records** (`RRSIG`/`DNSKEY`/`DS`) | Nothing reads them, and see §4 on the DO bit |

**Correction to the map.** The **Not yet specified** retention patch reads *"`dns-record` multiplies
by six qtypes"*. Six is uncited — no document in this repo names it — and the answer is **seven**.

---

## 3. The ALPN list and the HTTP versions spoken

A declared parameter of `http-exchange` (ADR-0025). Widening it moves values on running
`http-identity` timelines, so it costs a leaf bump and a `Break` on that facet — real, but one
facet rather than two.

### 3.1 The list is `h2, http/1.1`, in that order, and it is complete

ADR-0025 ruled the offer; #62 asks whether it is the *whole* list. It is, by exhaustion over the
ALPN identifiers that can mean anything in a TLS-over-TCP ClientHello:

| Candidate | Verdict |
| --- | --- |
| `h2` | **Offered.** `[spec]` RFC 7540/9113. An h2-only-over-TLS listener — a gRPC service, the modal case in a small estate — answers rather than reading `NoHTTPResponse` |
| `http/1.1` | **Offered.** `[spec]` RFC 9112. Go's `net/http` client speaks it |
| `h3` | **Not offered.** HTTP/3 runs over QUIC on UDP, so it cannot be selected in a TLS-over-TCP ClientHello. This is a **transport** gap, not an ALPN one: an HTTP/3-only service has no TCP `Service` to hang an `Endpoint` off, so it is invisible one layer earlier |
| `h2c` | **Not offered.** h2c is negotiated by `Upgrade` or prior knowledge, never by ALPN. ADR-0025 already ruled the cleartext h2c case out of scope |
| `http/1.0` | **Not offered.** Go's `net/http` has no 1.0 client `[owner]` |

Order is declared and recorded because the offer is recorded **by content**; RFC 7301 §3.2 makes
selection the server's, so order is advisory and never a claim.

### 3.2 `no_application_protocol` resolves to `NoHTTPResponse`, and needs no new variant

RFC 7301 §3.2: a server supporting none of the client's advertised protocols **SHALL** respond with
a fatal `no_application_protocol` alert `[spec]`. The handshake fails. #62 asks what is recorded.

**Not `TLSRefused`.** That value means *the peer spoke TLS and accepted no candidate we offered*,
where a candidate is a version or a suite. Here the peer accepted our version and our suite and
refused our **application protocol** — filing that under `TLSRefused` is a value naming a property
of the listener that our own offer decided, which is the `NotHTTP` defect ADR-0025 had just
renamed away, one facet across.

The resolution is to scope the offer to the exchange that needs it:

> **The `certificate` handshake sends no ALPN extension at all.** ALPN is a declared parameter of
> `http-exchange`, not of `tls-handshake`, and RFC 7301 fires the alert only in response to an ALPN
> extension — so the `certificate` handshake cannot provoke it, and `certificate`'s value stops
> being a function of another leaf's parameter.

Then on the `http-exchange` connection, where ALPN *is* sent, `no_application_protocol` means the
exchange we made returned no HTTP response, which is **`NoHTTPResponse`** exactly as ADR-0025
defined it — *a negative is named for the exchange we made*. **No new variant, and no value-space
widening.**

The consequence is that the `certificate` handshake and the HTTP exchange are **separate
connections**. Sharing one connection would make `certificate` unreadable on any listener that
refuses our ALPN list, which is one leaf's parameter deciding another leaf's value. The extra
handshake is the price of one fact having one home, and it lands on #4 §6's per-host budget — noted
for #61, which owns the cadence that multiplies it.

---

## 4. The EDNS option set and advertised buffer size

A declared parameter of `resolution-walk` and `wildcard-discrimination` (ADR-0025). Under ADR-0025's
rule that **a truncated answer is never a value**, none of these can silently move a value — they
move only how often the fallback path in §5 is taken. This is the cheapest of the five lists and it
says so.

| Setting | Value | Evidence |
| --- | --- | --- |
| EDNS(0) | Present, version 0 | `[spec]` RFC 6891 |
| UDP payload size | **1232** | `[owner]` The DNS Flag Day 2020 coordinated position, derived from IPv6's required 1280-byte MTU, and the shipped default of BIND, Unbound and Knot — which is #21's *documented shipped default* attestation class, from the parties that own the question. 4096 provokes IP fragmentation, which middleboxes drop |
| DO bit (DNSSEC OK) | **Clear** | `[thin]`, and the cost is stated: v1 records nothing about DNSSEC and a validation failure is invisible. Setting DO enlarges every answer with RRSIGs and so multiplies §5's fallback rate, buying records no declared qtype holds and no v1 rule reads |
| **DNS Cookie** (RFC 7873) | **Sent**, with the single §5.3 retry on `BADCOOKIE` | `[spec]` See below |
| EDNS Client Subnet | **Not sent, in either form** | ADR-0025, and see below |
| NSID, Padding, Keepalive | Not sent | Nothing reads them |

### 4.1 Cookies design the `Lame` hazard out rather than guarding against it

ADR-0025 raised a real hazard: an authority requiring DNS cookies may answer a cookieless query
with `BADCOOKIE` or `REFUSED`, and a delegation walk reading a refusal as *this nameserver does not
serve the zone* emits a false `Lame` — a v1 signal firing on our own transport. Its remedy is a
prohibition: `resolution-walk` **may not convert a transport-level refusal into a zone-level
value**.

That prohibition stands as the backstop. But a better remedy was available and unnamed: **send a
cookie.** RFC 7873's exchange is designed for exactly this — the client sends an 8-byte client
cookie, a server requiring cookies replies `BADCOOKIE` with a server cookie, and the client retries
once with the full cookie and is answered `[spec]`. With the cookie sent and the one retry
performed, `BADCOOKIE` never reaches the walk's logic at all.

This is the project's standing preference for **structural over disciplinary** — ADR-0007's `Break`,
ADR-0009's union — applied to a prohibition that was going to be enforced by care.

### 4.2 No ECS, and the `/0` opt-out loses too

ADR-0025 refused ECS as *a `Vantage` in an option's clothes*: it makes a geo-aware authority's
answer a function of the subnet we claimed, which is the job the `vantage` component of the
timeline key already does. Not re-derived.

The **losing option worth naming** is the zero-length / `/0` ECS opt-out (RFC 7871 §7.1.2), which a
later session will reach for as a determinism improvement — *tell the authority not to tailor, and
answers stop varying by vantage*. It loses on ADR-0025's own ground read backwards: suppressing
geo-variation destroys the very fact the `vantage` key exists to record. **v1 sends no ECS extension
in either form.**

---

## 5. The DNS transport and fallback policy

A declared parameter of `resolution-walk` (ADR-0025). The edge it produces is `Gap` → value, which
[ADR-0014](../adr/0014-only-revealed-generalises.md) already ruled is not `revealed` — so this is
the list with the least at stake, and it is also the one resting on the least evidence.

### 5.1 When TCP is attempted

| Trigger | Evidence |
| --- | --- |
| **The TC bit is set** | `[spec]` RFC 1035 §4.2.1, RFC 7766. This is the mechanism behind ADR-0025's *a truncated answer is never a value* — the binary never has to guess that an answer is incomplete |
| **UDP attempts exhausted with no response** | `[thin]` One TCP attempt before recording nothing. The alternative is a coverage loss caused by our own transport and recorded as *we could not say* — true, avoidable, and coverage is what #22 and #28 make expensive |

### 5.2 The budget

**Per nameserver: two UDP attempts (initial plus one retry), then one TCP attempt.** `[thin]` —
there is no measurement and no attestation behind these numbers, and this document does not pretend
otherwise.

What is *not* thin is the shape. `resolution-walk` queries the **delegated authorities directly**
([#35](https://github.com/winniel123/verge-asm/issues/35)), so the walk's redundancy is
**horizontal** — across nameservers — and a deep per-nameserver retry stack buys a second copy of
redundancy the walk already has, at the cost of multiplying load against the operator's own
authorities under #4's safety frame.

The revision price is stated rather than hidden: these are declared parameters of one leaf, so
changing them bumps `resolution-walk` and `Break`s `resolution` and `dns-record` for one cadence.
Not free, not estate-wide.

### 5.3 What is recorded when TCP also fails — and #62's own framing is one step too eager

#62 asks for *"a `Gap`, per #54, never a partial RRset"*. The second half stands. The first half is
wrong, and the model's own machinery gives a better answer.

A failed query does **not** write a `Gap`. Under [ADR-0005](../adr/0005-scan-execution-model.md) a
`Batch` records the scope it **completed**, so the failed `(Name, qtype)` pair is simply **absent
from the recorded scope**. Under ADR-0014 a batch whose recorded scope excludes a thing never
touches its timeline — so nothing is written, the open `Span` stands, and **currency does the
rest**: the timeline ages into a `Gap` only if the failure persists past the bound.

That is ADR-0014's closing-custody-gate case verbatim (*a `Gap` does not open where measurement
merely stopped… currency does the rest*), and it is strictly better than writing a `Gap` per
failure, because a transient DNS blip no longer flaps the coverage class. It is the damping ADR-0007
refused to build, obtained structurally instead of as a threshold.

> **A failed query narrows the batch's recorded scope. It never writes a value, never writes a
> `Gap`, and never folds a partial RRset.**

---

## 6. What each list would cost to change after v1

| List | Post-v1 widening costs | Therefore |
| --- | --- | --- |
| **TLS candidate set** | A `Break` on every `tls-acceptance` **and** every `certificate` timeline in the estate (ADR-0030 §3) | Settled now, settled wide. The only list with a deadline |
| **qtype set** | `revealed` plus one message. The qtype is in the key, so a new timeline **opens** | May grow. CAA and PTR are deferred, not discarded |
| **ALPN list** | A `Break` on `http-identity` — the offer sits inside the value's conditioning, not in a key | Middle. Closed by exhaustion rather than by judgement, so the risk is low |
| **EDNS options / buffer** | Nothing. A truncated answer is never a value, so these move latency and not values | Cheapest. Revisable at will |
| **DNS transport / fallback** | A `resolution-walk` leaf bump; `Break`s `resolution` and `dns-record` for one cadence | Cheap enough to justify ruling the numbers on thin ground |
