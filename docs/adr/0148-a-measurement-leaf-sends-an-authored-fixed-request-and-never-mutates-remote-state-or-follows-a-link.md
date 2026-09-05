# ADR-0148: A measurement leaf sends an authored, fixed request and never mutates remote state or follows a link

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1279 ADR gaps: `internal/measure/httpexchange`, `internal/measure/edgefanout`](https://github.com/winniel123/verge-asm/issues/1279), gap 2
- **Found by:** [#1169](https://github.com/winniel123/verge-asm/issues/1169) / PR [#1278](https://github.com/winniel123/verge-asm/issues/1278), which deleted the two `http-exchange` comments that carried the rule
- **Map:** [#1131 Comment policy: build commentlint and sweep the tree](https://github.com/winniel123/verge-asm/issues/1131)
- **Rests on:** [ADR-0025](./0025-an-offer-is-scope-only-where-the-value-enumerates-it.md) (an offer is authored and recorded by content, never a library default) and [ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md) (a declared parameter ships in the release and is never a dial)
- **Bounds:** [ADR-0121](./0121-the-operator-declared-recursive-resolver-is-trusted-and-exempt-from-the-discovered-authority-egress-guard.md), whose egress guard is the mechanism that keeps §3's one second hop bounded
- **Not gap 1:** the network-seam rule went to [#1272](https://github.com/winniel123/verge-asm/issues/1272) and has its own ADR. This one rules on the **request shape**, not on where the socket is opened

## Context

`internal/measure/httpexchange` fixes two values. `Params.Method` is `GET` and `Params.Path` is `/`,
both authored in `DefaultParams` and recorded on the `Batch` by content under ADR-0025. Until
[#1169](https://github.com/winniel123/verge-asm/issues/1169) they carried two comments that said
*why*: the exchange asks for identity and never mutates, and it is a single request to the root and
never a crawl. Those comments were deleted, correctly — a rule that binds code outside the file it
sits in does not live in a field comment.

**The rule they carried has no durable home.** Every statement of it in the corpus is narrower than
the rule:

- `docs/spec/v1-spec.md` §3.3's safety table fixes the **values** — `GET /` only, redirects not
  followed. A table of knobs and defaults is not a rule about what a probe may do.
- `docs/spec/packaging-and-configuration.md` §5.3 refuses *HTTP probe paths* as a knob, and refuses
  it partly on the ground that **there is no path list in v1**. Read alone, that reasoning expires
  the moment somebody adds a list.
- `docs/research/safe-active-probing.md` §4.1 argues for `GET` over `HEAD` and notes GET is Safe by
  RFC 9110 §9.2.1. §10 forbids state-changing methods **in any default probe**. Both are about
  `http-exchange` and both are hedged by *default*.

So the strongest statement in the repository is a per-leaf default. **No document says the rule binds
every measurement leaf**, and three of the four sites above, read alone and in the present tense,
would let a competent session build a crawler and call it compliant.

The rule is not an HTTP fact. `edge-fanout` opens one no-SNI TLS handshake per candidate address and
reads the certificate that address serves; `connect-outcome` completes one TCP connect per
`(address, port)` pair with host discovery skipped; `resolution-walk` asks seven read-only qtypes.
Each is the same rule in a different protocol, and each holds today by convention alone.

## Decision

> **A measurement leaf puts an authored, fixed request shape on the wire. It never mutates remote
> state, and it never expands its own target set from what a response contains: no link is followed,
> no path is guessed, and nothing is crawled. A leaf's targets come from the job spec, and the job
> spec comes from custody.**

### 1. The request shape is authored, and that is what makes it fixed

The shape is a **declared parameter** in ADR-0021's sense and an **enumerated offer** in ADR-0025's:
it is authored in the release, recorded on the `Batch` by content, gated by the leaf's golden corpus,
and never operator-configurable. That is already why `Method` is `"GET"` and `Path` is `"/"` in
`httpexchange.DefaultParams`.

This ADR adds the half ADR-0025 does not carry. ADR-0025 rules on **how** a shape is chosen and
recorded. It says nothing about **which** shapes are admissible. A `POST` with an authored body,
recorded by content and locked by a corpus, satisfies ADR-0025 completely. §2 is the constraint that
rules it out.

### 2. A leaf never mutates remote state

The instrument's whole claim is that measuring an estate leaves it as it was found. Every leaf sends
only what a passive observer of the service could have caused: a connect, a handshake, a read-only
query, a safe method.

| Leaf | What goes on the wire | The bound |
| --- | --- | --- |
| `connect-outcome` | one full TCP connect per `(address, port)`, host discovery **skipped** | `DefaultProfile` in `internal/measure/connectoutcome/offers.go`. Never SYN, so the leaf runs non-root with no added capability |
| `http-exchange` | one `GET /`, 64 KB capped body read, redirects recorded and never followed | `DefaultParams` in `internal/measure/httpexchange/leaf.go`. GET is Safe by [RFC 9110 §9.2.1](https://www.rfc-editor.org/rfc/rfc9110.html#section-9.2.1) |
| `tls-acceptance` | N ClientHellos from a literal candidate set | `DefaultCandidateSet`. A handshake offer is not a request for a resource at all |
| `edge-fanout` | one no-SNI TLS handshake per candidate address, port 443 | `internal/measure/edgefanout/leaf.go`. It takes no server name; that is the measurement's shape, not a caller's choice |
| `resolution-walk` | queries in seven read-only qtypes — `A`, `AAAA`, `CNAME`, `NS`, `SOA`, `MX`, `TXT` | The `Qtype` constants are literal. **No `AXFR`, no `IXFR`, no `UPDATE`** |
| `wildcard-discrimination` | control-label queries reusing `resolution-walk`'s offers | `DefaultParams` in `internal/measure/wildcarddiscrim/run.go` |
| `blanket-discrimination` | control-port connects inside a fixed band, composed by `connect-outcome` | `DefaultParams` in `internal/measure/blanketdiscrim/leaf.go`. Not dispatched on its own (ADR-0104 §2) |

The rule binds a leaf that has not shipped as firmly as one that has. `datagram-outcome`
([ADR-0083](./0083-silence-decides-only-on-a-connection-oriented-transport.md)) is specified and
unbuilt, and whoever builds it inherits §1 and §2 with it.

### 3. A leaf never crawls, and the one second hop is bounded and named

Crawling is a **target-set** property, not a method property. A leaf's targets arrive in the job
spec. A leaf never adds a target because a response mentioned one.

`http-exchange` is the clean case and it is structural rather than disciplinary: `exchange.go` sets
`CheckRedirect` to `http.ErrUseLastResponse`, so the 3xx is returned as-is and its `Location` is
recorded as identity. One request goes out per target — never a redirect's next hop.

**There is exactly one place a leaf dials an address a response named, and it is stated here rather
than left for a future session to find and generalise.** `resolution-walk`'s delegation walk asks
`NS` for a name and then asks each returned nameserver for the `SOA`
(`internal/measure/resolutionwalk/leaf.go`, `walk`). Four properties make it a bounded second query
rather than a frontier:

1. **Fixed depth of one.** The `SOA` answers are read; they are never walked from.
2. **The qtype pair is authored.** `NS` then `SOA`, not whatever the answer suggests.
3. **The frontier cannot grow.** A nameserver that names another nameserver produces no third query.
4. **Every discovered authority passes the egress guard.** ADR-0121 exempts the operator-declared
   resolver and gates a query with a non-empty `Server` — the discovered half — through the
   pre-flight vet and the Control-hooked dialer.

A future leaf that wants a second hop must show all four. **Absent all four it is a crawl**, and
`resolution-walk` is not precedent for one.

### 4. What custody supplies, a leaf does not discover

`edge-fanout`'s candidate addresses come from custody, not from a list a response handed us
([ADR-0129](./0129-a-shared-foreign-edge-is-measured-by-fan-out-not-read-from-a-list.md)). That is
the same rule seen from the other side: expanding the aperture is an act of custody, which opens
timelines, records a scope on the `Batch` and carries its coverage class. A leaf that expanded its
own target set would widen the aperture with no `Seed`, no recorded scope and no `revealed` — an
aperture change nothing can diff, which is exactly what ADR-0025's authored-offer rule and
`measurement-offers.md` §1.7 exist to prevent in the other direction.

### 5. This is not the hermetic-corpus rule and not the network-seam rule

Three adjacent rules are easy to conflate, so the boundary is drawn.

- [ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md) and
  [ADR-0085](./0085-an-obligation-with-no-failing-test-has-no-owner-and-a-boundary-needs-a-row-on-each-side.md)
  govern **how a leaf is tested** — a golden corpus, hermetic, no network.
- [#1272](https://github.com/winniel123/verge-asm/issues/1272)'s ADR governs **where the socket is
  opened** — behind an interface, so one code path takes a production adapter and a scripted fake.
- **This ADR governs what goes on the wire when the production adapter runs.** A leaf could satisfy
  both of the others and still POST.

## Consequences

- **Five documentation sites are reconciled in this change**, under
  [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md): `v1-spec.md`
  §3.3, `packaging-and-configuration.md` §5.3's *HTTP probe paths* row, and
  `safe-active-probing.md` §4.1, §9's *HTTP probe paths* row and §10. Each stated a narrow or
  per-leaf version in the present tense. Each is now marked as an **instance** of this rule rather
  than as the rule.
- **This ADR changes no Go code.** Every leaf already complies. What was missing was the statement,
  not the behaviour.
- **`packaging-and-configuration.md` §5.3's path-list refusal stops resting on an absence.** It
  refused *HTTP probe paths* partly because no list exists to edit. After this ADR the refusal
  survives the day somebody proposes one.
- **`CONTEXT.md` gains nothing.** No term is added. `Derivation`, `Batch` and `Seed` already carry
  the vocabulary, and this is a constraint on a `Derivation`, not a new noun.
- **A new leaf inherits a checklist.** §2's table gains a row, and §3's four properties are the test
  any second hop must pass.
- **The exposed-panel question stays closed and now has a rule behind it.**
  [#5](https://github.com/winniel123/verge-asm/issues/5) refused path-probe-plus-matcher
  fingerprinting on scope. §3 refuses it a second time, on target-set grounds, so reopening it takes
  an ADR rather than a preference.
- **The cost is stated.** A single `GET /` sees only what the root serves. An estate whose signal
  lives one path deeper is measured less richly than a crawler would measure it, and this ADR
  accepts that permanently rather than as a v1 limitation.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **A crawling probe** — follow links, or walk a path list, to widen coverage per target | It is an aperture change with no carrier. A leaf that adds targets from a response widens what is measured with no `Seed`, no recorded scope on the `Batch` and no `revealed`, so the operator cannot diff it and coverage becomes unstatable. It also multiplies request volume by an unbounded factor against a live service, which every rate ceiling in `connect-outcome`'s `SafetyProfile` is sized against. §4 |
| **Follow redirects, since the hop is only one deep** | Already refused twice, and the reasons compound. A 301 to a different host stops measuring the asset being tracked (`safe-active-probing.md` §4.3), and following moves the `status` and the `title` that `http-exchange` decides, so it is a declared parameter and never a dial ([#124](https://github.com/winniel123/verge-asm/issues/124), ADR-0021). The `Location` is recorded, which is the half that is the finding |
| **A request method that changes remote state** — `POST`, `PUT`, `DELETE`, `PATCH`, DNS `UPDATE`, `AXFR`/`IXFR` | It breaks the instrument's constitutive claim: measuring an estate leaves it as it was found. GET is Safe by RFC 9110 §9.2.1 precisely because the client "does not request, and does not expect, any state change on the origin server". A mutating probe running unattended on a cadence against production is an outage waiting for a schedule, and no finding is worth it. `safe-active-probing.md` §10 already forbids it for HTTP; this generalises it |
| **A mutating method behind an opt-in flag** | It fails ADR-0021's gate before it fails on safety: the method is a declared parameter of the leaf and no declared parameter is ever operator-configurable. An opt-in also relocates the decision to whoever is on call during an incident |
| **Credential submission**, even default-login checks | `safe-active-probing.md` §10, unchanged. Unattended, it is repeated authentication against production, and a successful login is a state change by any reading |
| **Claim or register a suspected-dangling resource to confirm it** | `safe-active-probing.md` §7.4, unchanged. It is the sharpest possible mutation of somebody else's state, and confirmation is not worth acquiring the asset |
| **Leave the rule per-leaf, and repeat it in each leaf's package doc** | The state this ADR fixes. Seven leaves stated seven local versions and the generalisation existed nowhere, so PR [#1278](https://github.com/winniel123/verge-asm/issues/1278) deleting two comments removed the only prose carrying it. A comment that binds code outside its file fails the comment policy's own gates |
| **A section on [ADR-0025](./0025-an-offer-is-scope-only-where-the-value-enumerates-it.md)** | ADR-0025 rules on how an offer is authored and recorded, and an authored `POST` recorded by content satisfies it completely. This ADR rules on which shapes are admissible at all, which is a different subject |
| **A section on [ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md)** | Same fault one level over. ADR-0021's subject is what may move a leaf's `Version` and what may be a dial. The mutation ban binds a shape that no dial could reach |
| **Fold it into gap 1's network-seam ADR** ([#1272](https://github.com/winniel123/verge-asm/issues/1272)) | That ADR rules on where the socket is opened, so a test can drive it. A leaf can satisfy it perfectly and still crawl. §5 |
