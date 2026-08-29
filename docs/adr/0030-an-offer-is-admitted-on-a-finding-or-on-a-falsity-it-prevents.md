# An offer is admitted on a finding it reaches or a falsity it prevents — and the expensive list gets the lower bar

- **Status:** Accepted
- **Date:** 2026-08-13
- **Ticket:** [#62 What does the measurement binary actually offer, now that a default may not stand in for a declaration?](https://github.com/winniel123/verge-asm/issues/62)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)
- **The enumeration itself:** [`docs/spec/measurement-offers.md`](../spec/measurement-offers.md)

## Context

[ADR-0025](./0025-an-offer-is-scope-only-where-the-value-enumerates-it.md) ruled that **every offer
the measurement binary makes is enumerated in the job spec and passed to the library explicitly**,
and that a library default cannot be recorded honestly in either home. It then recorded, as its own
last consequence, that *"what the offers actually are is now required v1 spec content and is not
decided here"* — five lists, none of which had ever been written down.

This ADR carries the **rulings**. The five lists live in
[`docs/spec/measurement-offers.md`](../spec/measurement-offers.md), separately, because a list that
will be revised does not belong inside a decision that will not.

Two things settled elsewhere are inputs and are not re-derived here. A curated candidate list
deciding *where to look* is aperture and not
[#31](https://github.com/winniel123/verge-asm/issues/31)'s forbidden signature database. And the
recorded scope is what went on the wire, not what we intended.

## Decision

| Concern | Decision |
| --- | --- |
| What admits a candidate to an offer | **Two limbs**: its acceptance is a **finding**, or its absence would make the **measurement false**. Nothing else |
| Does #21's claim/attestation standard govern an offer? | **No.** An offer asserts nothing about the world, so there is no claim to attest. It is a third instrument |
| Which list gets the strictest bar | **None of them — and the expensive list gets the *lowest* bar.** The asymmetric price sits on **omission**, not inclusion |
| The TLS candidate set | **One list, carried by both TLS exchanges.** Versions TLS 1.0–1.3; suites declared for 1.0–1.2 only. SSLv3 and RC4 not declared, because unofferable |
| TLS 1.3 cipher suites | **Excluded from `tls-acceptance`'s value.** Go does not let them be declared, and a per-candidate negative over candidates we did not choose is *TLS 1.0/1.1 negotiated* a third time |
| The declared set's form | **Literal, never derived from the library's own lists** — and **CI fails if any declared candidate is not offerable** |
| The prober's qtype set | **Seven** — A, AAAA, CNAME, MX, NS, SOA, TXT. CAA and PTR deferred |
| The ALPN list | **`h2, http/1.1`, and that is the whole list**, closed by exhaustion |
| `no_application_protocol` | **`NoHTTPResponse`.** No new value-space variant. And the **`certificate` handshake sends no ALPN extension**, so it can never provoke the alert |
| The EDNS option set | EDNS(0), buffer **1232**, DO **clear**, **DNS Cookie sent** with RFC 7873's one retry, **no ECS in either form** |
| DNS transport and fallback | TCP on the TC bit and on UDP exhaustion; **two UDP attempts then one TCP**, per nameserver, on **thin ground** and marked |
| What a failed query writes | **Nothing.** It narrows the batch's recorded scope; currency decides whether a `Gap` opens |
| ADR-0025's test, corrected | The test is asked **of the offer**, never per facet — otherwise one offer gets two homes and the doubling trap rebuilds itself |
| Widening the TLS offer | `Break`s **`certificate` as well as `tls-acceptance`**. ADR-0025 priced only the second |
| The enumerated aperture input count | ~~**Unchanged at six.**~~ **Unchanged by this ADR** — *the figure is withdrawn; it is **seven** since [ADR-0066](./0066-a-control-probe-is-generated-under-a-names-parent-and-that-population-is-aperture.md) added the control-probe population, which is not an offer.* This ADR adds none |
| Operator-configurable offers | **None in v1.** An offer the operator can narrow is a finding the operator can silence |

## Rationale

### 1. An offer needs its own evidence standard, because it asserts nothing

This repo has two instruments and neither fits.
[#21](https://github.com/winniel123/verge-asm/issues/21)'s claim/attestation/determinacy standard
governs what a **curated list may assert** — it admits a row only on a named claim attested by the
owner of the thing. [#31](https://github.com/winniel123/verge-asm/issues/31)'s spec-defined-field
test governs what a **measurement may conclude**. The map already suspects these are complementary
rather than rival, and this ticket is the case that shows why: **an offer is neither.**

An offer makes no claim. It decides what question we ask. So the failure modes are not a false
verdict — they are a **blind spot** (too narrow) and a **cost** (too wide), and the standard has to
be cut on those:

> **A candidate is admitted where its acceptance is a finding, or where its absence would make the
> measurement false.** The first limb asks *what would we fail to see if we did not ask this?*; the
> second asks *what would we assert wrongly if we did not ask this?*

The second limb is the one nobody would have written down, and it carries most of the TLS suite
list: `TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256` is not a finding and never will be, and omitting it
makes every ECDSA-certificate host in the estate read `TLSRefused` — a false negative in the
flagship direction, which is the exact failure ADR-0025 exists to prevent, arriving through a list
that was curated *too carefully*.

### 2. The expensive list gets the lower bar, which is the opposite of what sessions will assume

#62 is explicit that the prices are asymmetric and that the TLS list is the one with a deadline.
The natural inference — *the expensive decision deserves the stricter standard* — is **backwards**,
and getting it wrong is how this ticket fails.

The asymmetry sits on **omission**. Omitting a TLS candidate costs the product a finding on exactly
the estate where it is true, and costs a `Break` on two facets to fix later. Including one costs
one extra handshake, inside a budget #4 already set. So the bar on the expensive list must be *low*
— err wide, deliberately — and the strictness belongs on the *cheap* lists, where an unneeded row
buys a timeline per subject forever.

That is what actually happened to the two lists. The TLS set admits everything Go can offer that is
not junk. The qtype set **refuses two rows** — CAA and PTR — that a session would have carried in
by inertia, and CAA's refusal is the sharper one: its only recorded rationale in this repo is *"a
mismatch between CAA and observed CT issuance is a drift signal"*, and
[#56](https://github.com/winniel123/verge-asm/issues/56) ruled CT mis-issuance detection and **any
CT-fed facet** out of scope. The rationale was withdrawn three tickets ago and the row survived it,
which is [#47](https://github.com/winniel123/verge-asm/issues/47)'s hazard — *a decision that
changes what a thing **is** should grep for claims about what it **does*** — arriving inside a
research document instead of an ADR.

### 3. ADR-0025's test must be asked of the offer, not of each facet — or the trap it disarmed rebuilds itself

This is the load-bearing correction and it was invisible until the TLS list was actually written.

ADR-0025's test reads: *an offer is batch scope exactly where **the value** carries a per-candidate
negative; otherwise it is a declared parameter of the leaf that made it.* Asked per facet, the TLS
candidate set gets **two different answers**, because it feeds two facets with different value
shapes:

- `tls-acceptance`'s value **is** a subset of the candidate set → **scope**.
- `certificate`'s value is `Presented(chain) │ TLSRefused │ NoTLS`, a single global negative →
  **parameter** of `tls-handshake`.

Follow that through and ADR-0025's own trap reopens. `tls-handshake` feeds **both** facets
([ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md)'s leaf table), so widening the
offer would pay a `Break` on `tls-acceptance` for the aperture widening **and** bump
`tls-handshake`, which `Break`s `tls-acceptance` again. Two prices, one change, **the same object**
— which ADR-0025 named precisely and thought it had disarmed: *"they do not compose; they double."*
Its disarming move (*one fact has one home*) closes the library-versus-us door and leaves this one
open.

ADR-0025 half-knew. Its Consequences say the declared candidate set is *"scope rather than
parameter and lives on the `Batch`"*, flatly, with no per-facet split — which contradicts its own
test as stated. This ADR resolves the inconsistency **in favour of the consequence line**:

> **The test is asked of the offer, not of each facet the offer touches.** An offer is scope where
> **any** value it feeds carries a per-candidate negative, and then it is scope for **every** batch
> that makes it — including batches whose own facet carries only a global negative. Otherwise it is
> a declared parameter.

The correction is minimal by construction: it changes only the TLS case. ALPN feeds `http-identity`
alone. EDNS and transport feed `resolution` and `dns-record`, and neither value enumerates over an
EDNS option or a transport. All three stay parameters, exactly as ADR-0025 placed them.

**One consequence follows immediately and ADR-0025 did not price it.** If the candidate set is a
recorded scope dimension of the `certificate` batch too, then widening the offer `Break`s
`certificate` as well — and it should, because a wider offer genuinely moves `TLSRefused` →
`Presented(chain)` on a legacy box. ADR-0025 priced widening at *"a `Break` on every
`tls-acceptance` timeline"*. The true price is **two facets**, which makes the case for settling the
list wide now stronger than the ticket's framing already made it.

This adds **no seventh aperture input**. It is the same input recorded on more batches, and the
enumerated count stays at six. *(**The figure is withdrawn.** The seventh arrived later and from
elsewhere: [ADR-0066](./0066-a-control-probe-is-generated-under-a-names-parent-and-that-population-is-aperture.md)'s
**control-probe population**, which is not an offer, so nothing in this section is disturbed —
*this ADR adds none* stands.)*

### 4. Go will not let the whole TLS offer be declared, and the residue must leave the value

ADR-0025's rule is *every offer the binary makes is enumerated in the job spec and passed to the
library explicitly*. There is a hole in it that only appears on contact with the library:
**`Config.CipherSuites` is documented as ignored for TLS 1.3.** The three TLS 1.3 suites are Go's
choice and cannot be made ours.

Under ADR-0025's own reasoning that settles it. A per-candidate negative over candidates **we did
not choose** would move estate-wide on a library upgrade with nothing in the world having changed —
which is *TLS 1.0/1.1 negotiated* a third time, and worse than the first two because it arrives
through the **scope record**, where the model's detection mechanism is diffing named dimensions and
the diff would fire for a reason nothing names.

So `tls-acceptance` records, under TLS 1.3, **that the version was accepted and nothing about
suites**. The asymmetry looks arbitrary and is not: it is the boundary of what we are permitted to
declare. And nothing is lost — all three TLS 1.3 suites are AEAD and none is a finding, so no row
would have been admitted on either limb anyway.

The rejected repair worth naming is **pinning the Go version so the 1.3 list is fixed**. That is
*the toolchain is a parameter of every leaf*, which ADR-0021 identifies as
[ADR-0008](./0008-derivation-versions-move-on-content.md)'s rejected *versions move on releases*
arriving through `go.mod`.

### 5. A declared offer needs a build-time offerability check, or ADR-0025 decays silently

ADR-0025 promises that *"an unofferable candidate is an absence from the scope record, visible, and
not a silent one"*. As written, the only thing discharging that promise is the batch record — so
noticing requires diffing scope records across batches, **after** the `Break` has already fired on
two facets. That is detection after the fact.

The mechanism that actually discharges it is the mirror of ADR-0021's bidirectional corpus gate:

> **The build fails if any declared candidate is not offerable by the linked library.**

ADR-0021 made a version that moves for nothing fail the build as loudly as an output that moves for
free. The same shape applies to an offer that **stops going out**: a Go release dropping 3DES
becomes a build failure and a deliberate, priced narrowing, never a silent one.

It also fixes this ADR's own weakest column. Every claim in the spec document about what Go can put
on the wire is read from documentation and **unmeasured in this repo** — no Go toolchain has been
run here. The check converts that column from claimed to measured on the first build, which is
[#21](https://github.com/winniel123/verge-asm/issues/21)'s *publish the weak tier* with a named
route out of the weak tier rather than a permanent caveat.

The tempting alternative is a **derived** set — `tls.CipherSuites() ∪ tls.InsecureCipherSuites()`,
in the manner of [ADR-0009](./0009-verge-core-is-a-union.md)'s `verge-core`: a definition that
cannot fail rather than a list that can go stale. It is refused, and refusing it is the point: a
derived set's content **moves when Go moves**, so a Go upgrade silently widens or narrows the offer
and `Break`s the estate for a dependency reason. That is ADR-0025's *"a Go upgrade cannot widen the
TLS offer"* made false again. ADR-0009's move is right where the inputs are ours and wrong where one
of them is a third party's.

### 6. `no_application_protocol` is answered by scoping the offer, not by widening a value space

RFC 7301 §3.2 makes a server that shares no ALPN protocol with the client send a fatal
`no_application_protocol` alert. The handshake fails, and nothing in the model said where that goes.

`TLSRefused` is the tempting home and it is wrong: that value means *the peer accepted no candidate
we offered*, where a candidate is a version or a suite, and here the peer accepted both and refused
our **application protocol**. Filing it there is a value naming a property of the listener that our
own offer decided — the `NotHTTP` defect ADR-0025 had just renamed away, one facet across, and the
sixth instance of a pattern this map now catches reliably.

Widening `certificate`'s value space is refused on ADR-0015's discipline. The pre-v1 price is
vacuous but ADR-0025 has just added `TLSRefused` and a value space that grows every time somebody
finds an edge is not a closed union.

The answer is that **ALPN never belonged on that connection**:

> The `certificate` handshake sends **no ALPN extension**. ALPN is a declared parameter of
> `http-exchange`, and RFC 7301 fires the alert only in response to an ALPN extension.

The hole closes without a new variant, and `certificate`'s value stops being a function of another
leaf's parameter — which it should never have been. On the `http-exchange` connection, where ALPN
*is* sent, the alert means the exchange we made returned no HTTP response: **`NoHTTPResponse`**,
which is what ADR-0025 named it for.

The stated cost is one more handshake per endpoint, because the `certificate` handshake and the
HTTP exchange are now unambiguously **separate connections**. That lands on #4 §6's per-host budget,
multiplied by whatever cadence [#61](https://github.com/winniel123/verge-asm/issues/61) rules.

### 7. Cookies design the false-`Lame` hazard out; the prohibition becomes a backstop

ADR-0025 identified a v1 signal firing on our own transport — an authority requiring DNS cookies
answering `BADCOOKIE` or `REFUSED`, read by the walk as *this nameserver does not serve the zone* —
and remedied it with a prohibition: `resolution-walk` may not convert a transport-level refusal into
a zone-level value.

The prohibition is correct and stays. But RFC 7873 already solves this in the protocol: send an
8-byte client cookie, and on `BADCOOKIE` retry once with the server cookie you were just handed.
With that, `BADCOOKIE` never reaches the walk's logic. This is the project's standing preference for
**structural over disciplinary** — ADR-0007's `Break`, ADR-0009's union, ADR-0022's singular confirm
— applied to a rule that was otherwise going to be enforced by care in a code path nobody revisits.

### 8. A failed query writes nothing, and #62's own framing was one step too eager

#62 asks what is recorded when TCP fallback also fails, and answers itself: *"a `Gap`, per #54,
never a partial RRset."* The second half stands. The first is wrong, and correcting it is free
because the machinery already exists.

Under [ADR-0005](./0005-scan-execution-model.md) a `Batch` records the scope it **completed**, so a
failed `(Name, qtype)` pair is simply **absent from the recorded scope**. Under
[ADR-0014](./0014-only-revealed-generalises.md) a batch whose recorded scope excludes a thing never
touches its timeline. So nothing is written, the open `Span` stands, and **currency does the rest**
— the timeline ages into a `Gap` only if the failure persists past the bound. That is ADR-0014's
closing-custody-gate case verbatim.

It is strictly better than writing a `Gap` per failure, because a transient DNS blip no longer flaps
the coverage class — the damping ADR-0007 refused to build as a threshold, obtained structurally
instead.

### 9. None of the five is operator-configurable, and the reason generalises a rule already on the books

#4 §9 lists eighteen knobs. The TLS enumeration **cadence** is among them — and that is
#61's — while the candidate **set** is not. This ADR keeps all five offers out of the operator's
hands, on [ADR-0009](./0009-verge-core-is-a-union.md)'s rule one step out:

> *A port the operator can hide is a signal the operator can silence* becomes **an offer the
> operator can narrow is a finding the operator can silence.**

Narrowing the TLS offer is the single edit that turns off `tls-1.0-accepted` estate-wide while every
screen keeps reporting normally, which is [#22](https://github.com/winniel123/verge-asm/issues/22)'s
refused suppression reaching the one layer beneath every guard the model has.

This is an **input to** [#60](https://github.com/winniel123/verge-asm/issues/60), not a ruling on
it. #60 is deciding whether a *rule's* declared parameter may be operator-configurable and what an
edit costs. If it rules that a declared parameter may be operator-editable in general, the five
offers here are excepted by this section rather than by #60's silence.

## Consequences

- **[`docs/spec/measurement-offers.md`](../spec/measurement-offers.md) is new**, and is v1 spec
  content that discharges ADR-0025's last consequence and unblocks
  [#12](https://github.com/winniel123/verge-asm/issues/12). It carries the five lists row by row
  with a per-row evidence mark, and a whole column marked unmeasured until the offerability check
  runs.
- **A new build obligation**: CI fails if any declared candidate is not offerable by the linked
  library. It is the offer's half of ADR-0021's bidirectional gate.
- **[ADR-0025](./0025-an-offer-is-scope-only-where-the-value-enumerates-it.md) is amended in two
  places** — its test is asked of the offer rather than per facet, and widening the TLS offer
  `Break`s `certificate` as well as `tls-acceptance`.
- **`certificate`'s handshake carries no ALPN extension**, and the `certificate` handshake and the
  HTTP exchange are separate connections. One extra handshake per endpoint against #4 §6's budget.
- **`resolution-walk` gains a DNS Cookie and one RFC 7873 retry.** ADR-0025's prohibition on
  converting a transport refusal into a zone-level value becomes a backstop rather than the primary
  guard.
- **`tls-acceptance` records no cipher suite under TLS 1.3.**
- **Two qtypes are deferred** — CAA and PTR — priced at `revealed` plus one message whenever anyone
  wants them. PTR additionally needs `dns-record` to key on `Address`, which is left as fog.
- **The map's *"`dns-record` multiplies by six qtypes"* is corrected to seven.**
- **The aperture input list is unchanged ~~at six~~ by this ADR.** *(The figure is withdrawn — **seven** since [ADR-0066](./0066-a-control-probe-is-generated-under-a-names-parent-and-that-population-is-aperture.md).)*
- **What is decided on thin ground and marked as such**: the DNS retry and fallback budget, and the
  DO bit. Neither has a measurement or an attestation behind it, and both are cheap to revise.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| Govern offers with #21's claim/attestation/determinacy standard | An offer asserts nothing about the world, so there is no claim to attest. Applying it would have excluded the entire second limb — every suite offered so that a correct listener is not mis-measured |
| Give the expensive list the stricter bar | Backwards. The asymmetric price sits on omission, so strictness on the TLS list buys blind spots and costs a two-facet `Break` to repair |
| Two TLS candidate sets, one per exchange | ADR-0025 licensed it (*"they need not match"*) and it is two chances to make one mistake against one box. The cost objection it answers is about the **exchange**, and is answered by iterative narrowing instead |
| A derived suite set, `tls.CipherSuites() ∪ tls.InsecureCipherSuites()` | ADR-0009's union move applied where one input is a third party's: its content moves when Go moves, so a Go upgrade silently re-widens the offer — ADR-0025's central promise, broken through its one unchecked door |
| Pin the Go version so the TLS 1.3 suite list is fixed | *The toolchain is a parameter of every leaf*, which ADR-0021 names as ADR-0008's rejected *versions move on releases* arriving through `go.mod` |
| Declare SSLv3 and RC4 anyway, and let the wire record show the absence | A job-spec row that provably never reaches the wire is ADR-0013's deleted `unknown` — a declaration nothing can emit reads as a real distinction forever. `TLSRefused` plus the recorded set already says it |
| Record the three TLS 1.3 suites in `tls-acceptance`'s value | A per-candidate negative over candidates Go chose for us: *TLS 1.0/1.1 negotiated* a third time, arriving through the scope record where the detection mechanism itself would misfire |
| `no_application_protocol` → `TLSRefused` | A value naming a property of the listener that our own ALPN offer decided — the `NotHTTP` defect one facet across |
| `no_application_protocol` → a fourth `certificate` variant | Buys with a value-space widening what scoping the offer buys for nothing, and a closed union that grows on every edge is not closed |
| Share one connection between the `certificate` handshake and the HTTP exchange | Saves a handshake and makes `certificate`'s value a function of `http-exchange`'s ALPN parameter — one leaf's parameter deciding another leaf's value |
| Keep CAA in the qtype set | Its only recorded rationale is the CT-issuance comparison #56 ruled out of scope. *The record changed* is true of every qtype and admits all of them |
| Query PTR in v1 | Serves the sub-1% population #26 measured, on a `Seed` kind ADR-0013 made non-modal, and needs `dns-record` to key on a second subject |
| Set the DO bit | Multiplies fallback rate to collect records no declared qtype holds and no v1 rule reads |
| Send a `/0` EDNS Client Subnet to opt out of geo-tailoring | Suppresses exactly the variation the `vantage` key exists to record — ADR-0025's ECS refusal read backwards |
| Write a `Gap` on a failed DNS query | ADR-0005 and ADR-0014 already give the answer: the scope record narrows and currency decides. Writing one per failure flaps the coverage class on a transient blip |
| Make the TLS candidate set operator-configurable | An offer the operator can narrow is a finding the operator can silence — ADR-0009's port rule one level beneath every guard the model has |
