# ADR-0186: a certificate rule's severity is rated by the client failure it can show today, never by certificate quality

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1301 ADR gaps: internal/signal](https://github.com/winniel123/verge-asm/issues/1301), gap 2
- **PR that deleted the comment:** [#1302](https://github.com/winniel123/verge-asm/pull/1302)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Rests on:** [ADR-0116](./0116-the-design-package-is-normative-for-look-and-functionality.md),
  whose live remnant rules that a signal carries a five-level severity assigned per rule — that ADR
  built the ramp and rated nothing on it — and
  [ADR-0185](./0185-a-severity-is-the-operator-facing-grade-so-it-composes-into-no-version-vector-and-is-not-a-fifth-part-of-a-rule.md),
  which keeps a grade out of the rule's version vector, so a re-rating costs no `Break`. That is why
  a rating principle has to be written: the grade is cheap to move, and the only thing that holds it
  is an argument
- **Bounded by:** [ADR-0024](./0024-a-rules-domain-is-the-extension-of-its-name.md). Its v1 rule
  table fixes each certificate rule's `Predicate domain` and its `not-evaluable` case, and it has no
  severity column. Every grade in this ADR sits beside a domain that ADR already ruled, and none
  moves one
- **Sibling of, and not ruled by:**
  [ADR-0064](./0064-a-message-names-what-moved-and-where-nothing-moved-it-says-so.md), which refuses
  a severity field on a `Message`. It rules the notification layer. This ADR rules how a rule's own
  grade is argued. A message still carries no severity

## Context

`internal/signal/endpoint.go:205` carried this, on the `var` block that declares the five
certificate-detail rules, until #1302 compressed it to one line:

```go
// The five shipped certificate-detail rules, in ADR-0024 table order. Each names
// the one parsed-leaf boolean it reads; the domain, not-evaluable, and
// fired/not-fired control flow live once, in certDetailRule.Eval above.
// Severities, per rule: an expired or not-yet-valid leaf breaks TLS for clients
// today (critical / high); a weak key or signature is a high-value forgery risk;
// a self-signed leaf and an approaching expiry are medium warnings.
```

**Nothing on disk states the rating principle.** ADR-0024's v1 rule table has four columns and none
is a severity. The [v1 SPEC §5.2](../spec/v1-spec.md) names the five levels and says every rule
ships at one, and says nothing about which. `CONTEXT.md`'s `Signal` entry says a grade is *assigned
per rule* and does not say by what. ADR-0116 built the ramp because the design rendered it, and
rated nothing on it. So the five grades were authored at P0.1 and the argument behind them was
written in one comment, which #1302 deleted.

### The grades, and the leaf boolean each rule reads

The domain of all five is one condition — `presentedCert`, at `internal/signal/endpoint.go:107`:
the `certificate` facet was measured and its outcome is `Presented`. `NoTLS` is outside all of them,
per ADR-0024's table. The `not-evaluable` case is the same for all five: the parsed leaf is absent,
or its one boolean is unset.

| Rule | The boolean, and where it is folded | Grade |
| --- | --- | --- |
| `certificate-expired` | `notAfter` is at or before now — `cmd/web/signals.go:1053` | `critical` |
| `certificate-not-yet-valid` | `notBefore` is after now — `cmd/web/signals.go:1062` | `high` |
| `certificate-weak-key-or-signature` | RSA under 2048 bits, ECDSA under 224, DSA under 2048/224, or an MD5 or SHA-1 signature on any chain link that is not self-signed — `weakKeyOrSignature`, `cmd/web/signals.go:1151` | `high` |
| `certificate-self-signed` | the leaf's subject equals its issuer **and** its self-signature verifies — `selfSignedOf`, `cmd/web/signals.go:1083` | `medium` |
| `certificate-expiring` | `notAfter` is in the future and within the expiry window — `cmd/web/signals.go:1054` | `medium` |

### The ranking is not the one a quality reading produces, and the gap is two bands wide

Exactly **two** of the seventeen shipped rules are `critical`:
`sensitive-port-reached-from-internet` and `certificate-expired`. The top band is scarce, so every
placement in it is a claim.

Rank the same five by how bad the certificate is, which is what a CVSS-style or hygiene reading
does, and the order inverts at both ends.

- **`certificate-weak-key-or-signature` is the worst certificate in the set.** A 1024-bit RSA key or
  an MD5 chain signature is a forgeable credential, and no clock fixes it. A quality reading puts it
  at the top. It ships at `high`, below `certificate-expired`.
- **`certificate-self-signed` is the most-flagged certificate finding in the industry**, and a
  quality reading puts it at or near the top: it is a certificate no public trust store will accept.
  It ships at `medium`, two bands below `certificate-expired`.
- **`certificate-expired` is, by quality, unremarkable.** It may be a perfectly formed 2048-bit
  certificate from a public CA, one day past `notAfter`. It ships at `critical`.

### The case where the quality reading fails, in this product

Verge ASM measures an estate from a vantage that is **not one of the operator's clients**. A
self-signed leaf on an internal service is the ordinary shape of a private PKI, a service mesh, or a
device that ships its own certificate. The clients that use it may hold the root, and every one of
them may complete the handshake.

Under the quality reading that endpoint sits at the top of the operator's inbox, above an expired
public certificate on the login page that is failing every browser right now. The operator works the
list in order and reaches the thing that is actually broken second. **The ranking inverts what must
be acted on first**, and it does so on the commonest configuration in the estate rather than on an
edge case.

The measurement cannot tell us whether a self-signed leaf is failing anybody. `SelfSignatureVerifies`
tells us the leaf signed itself. Nothing in the `certificate` value tells us which roots the
operator's clients carry. A grade that asserts a client is failing, where the measurement cannot
show one, is a claim the evidence does not carry.

## Decision

> **A certificate rule's severity is rated by the client failure the measurement can demonstrate
> today, and never by how bad the certificate is. Two things fix the band and nothing else does:
> whether a client is failing now, and whether that failure is unconditional or turns on a trust
> decision the operator may already have made. This ADR re-rates no rule. It records the principle
> that produced the shipped grades, and it is the document a future re-rating must argue against.**

Five limbs.

### 1. *Breaks TLS for a client today* means a failure now, not a property

The test is a demonstrated failure at the instant of evaluation. Three readings are excluded by it.

- **Not a latent weakness.** A certificate that could be forged in future is not, by that alone, a
  failure today. Limb 2 admits `certificate-weak-key-or-signature` on a different ground.
- **Not a hygiene deviation.** A certificate that departs from a policy nobody's client enforces is
  not a failure.
- **Not a prediction.** A certificate that will fail on a known date is graded for the failure it
  has today, which is none. That is `certificate-expiring`, and it is why the rule sits at `medium`
  rather than at the band of the failure it forecasts.

### 2. The ranking, applied to all five

Two axes, in order: **is a client failing now**, then **is the failure unconditional or
configurable**. The shipped grades fall out of them.

| Rule | Is a client failing now? | Conditional on what? | Band, and why it sits there |
| --- | --- | --- | --- |
| `certificate-expired` | **Yes** | Nothing. No trust store, no pinning and no configuration accepts an expired leaf | `critical`. The failure is unconditional, present, and gets worse rather than better with time. It is the only certificate rule that meets all three |
| `certificate-not-yet-valid` | **Yes** | Nothing. Unconditional, exactly as above | `high`. It is one band below `certificate-expired` on duration alone: the failure ends by itself at `notBefore`, with no operator act. An expired leaf never self-heals |
| `certificate-weak-key-or-signature` | **Yes** | Nothing, for a current client | `high`. Current clients hard-fail an MD5 or SHA-1 chain signature and a sub-2048-bit RSA key outright, so the failure is present and unconditional. Where an old client still accepts one, the session it gets is forgeable — the transport completes and protects nobody. It is broken in both readings, so the band does not turn on which client we imagine |
| `certificate-self-signed` | **We cannot show one** | The client's trust store, which the operator may have configured | `medium`. It fails a client using a public trust store and completes for a client holding the root. The measurement cannot tell the two apart, so the grade states a warning rather than a failure |
| `certificate-expiring` | **No** | Nothing yet. The failure is dated and certain | `medium`. It is the same failure as `certificate-expired`, not yet arrived. It ranks with `certificate-self-signed` and below every rule where a client is failing now |

The two `medium` rules are `medium` for two different reasons, and both reasons are the principle:
one has no demonstrable failure because we cannot see the client's trust store, the other has none
because the failure has not started.

### 3. The sixth certificate rule takes the same principle

`certificate-hostname-san-mismatch` is a certificate rule and is not one of the five: it is its own
type at `internal/signal/endpoint.go:149`, because ADR-0024 gives it a second domain condition — the
`Endpoint` must have a `Name`.

It ships at `high`, and the principle places it there. A client that reaches the endpoint by the
name we hold fails the handshake now, unconditionally, with no trust-store escape. The failure is
conditional only on **which name the client used**, and the name it is measured against is the one
the estate holds for that endpoint, so the condition is met by the clients that exist rather than by
a configuration choice.

**This is the ADR's thinnest limb, and it is stated as thin.** The distance from
`certificate-expired`'s `critical` rests on the mismatch failing per-name where an expiry fails per
endpoint. A future re-rating that argued the two are the same failure would be arguing inside this
principle, not against it, and §5 says what such an argument owes.

### 4. What the grade is not, and what it may not become

- **It is not a score.** The [v1 SPEC §5.2](../spec/v1-spec.md) rules that a firing carries no
  per-finding score. The grade is the rule's, one of five bands, identical on every instance the
  rule raises (ADR-0116).
- **It is not evidence, and it is not a census term.** A census counts subjects in three registers
  over one population. The grade ranks rows the census already produced. It composes into no version
  vector (ADR-0185) and moves no member.
- **It is not a claim about the operator's environment.** The principle grades the failure the
  measurement can show. Where the measurement cannot show one, the band drops. It never fills the
  gap with an assumption about which clients exist.
- **It never damps.** Rating a rule low is not a reason to narrow its domain, and the two decisions
  are made in different places. A rule fires or does not fire on its predicate alone
  (ADR-0024). The grade decides only where the fired row sits in the operator's order.

### 5. What a re-rating owes

Under ADR-0185 a re-rating costs no `Break` and no comparability cycle. That makes the argument the
only thing holding a band, so this ADR fixes what the argument has to be.

A re-rating of a certificate rule must show a **change in the client failure**: that a failure is
now demonstrable where it was not, that a conditional failure has become unconditional, or that a
client population which used to fail no longer does. A published deprecation that turns a
conditional rejection into a universal one is such a change.

An argument that the certificate is worse than the band says, or that the industry rates it higher,
or that operators expect it at the top, is not such a change. It is the reading §Context measured and
this ADR refused.

## Consequences

- **No rule's grade moves, and this ADR changes no Go code.** All five shipped grades already follow
  the principle. The ruling is the argument, written down where the code comment used to hold it.
- **The five grades are now defensible without reading `endpoint.go`.** Before this, an operator or
  a reviewer asking why a self-signed certificate is `medium` had the deleted comment or nothing.
- **[`CONTEXT.md`](../../CONTEXT.md) gains nothing.** The glossary's `Signal` entry states that a
  grade is assigned per rule and states no rule's grade. This principle is scoped to one rule family
  and adds no domain term. No glossary clause is invalidated by it.
- **`internal/signal/endpoint.go`'s surviving one-line comment gains a citation.** It states this
  rule uncited today. Recorded in this batch's manifest.
- **The other twelve rules' grades are still unwritten.** `sensitive-port-reached-from-internet` is
  the second `critical` in the set and rests on ADR-0010's two reaches rather than on a client
  failure, so it is not covered by this principle and needs its own. That is a gap, not a defect,
  and it is not opened here.
- **`certificate-expiring`'s window is a defect this ADR rates around, and does not repair.** The
  shipped predicate reads a flat 30 days, `certExpiryWindow` at `cmd/web/deltas.go:18`.
  [ADR-0043](./0043-a-clock-reading-rule-bounds-its-evidence-in-the-subjects-own-units.md) ruled the
  horizon to be `N = ⅓ × (not_after − not_before)`, and `½ ×` below a 10-day validity, and gave the
  three clock-reading certificate rules an evaluability guard on the observation's age against that
  horizon. Neither is in the code, and a flat 30 is the value ADR-0043 §7.3 names as the failure it
  repaired. **The band is unaffected** — `certificate-expiring` is `medium` because no client is
  failing yet, whatever the horizon that decides *yet*. **The predicate ships as its own ticket**,
  tracked alongside the version-vector consequence
  [ADR-0185](./0185-a-severity-is-the-operator-facing-grade-so-it-composes-into-no-version-vector-and-is-not-a-fifth-part-of-a-rule.md)
  records.
- **Nothing enforces the principle.** No check reads a grade, and a new certificate rule can ship at
  any band. Review carries it, and §5 is what review holds an author to.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Rate by certificate quality — how bad the certificate is** | Puts `certificate-self-signed` above `certificate-expired`. In this product that means an internal service whose own clients hold its root outranks a public login page failing every browser right now. It grades the commonest benign configuration in an estate as the most urgent thing in the inbox, and the operator reaches the broken endpoint second. It also asserts a client failure the measurement cannot show: nothing in the `certificate` value says which roots the operator's clients carry |
| **Rate by remediation effort — how hard the fix is** | `certificate-expired` and `certificate-expiring` have the identical fix, one renewal, and would collapse into one band. The whole distinction the ramp draws here is *is it failing now*, which effort cannot see. It would also make the grade move when a deployment pipeline changed and the certificate did not |
| **Import CVSS or another external scoring model** | Every certificate condition here is a configuration state on the operator's own asset, not a vulnerability in a product, so the vector's exploitability and impact metrics have no defensible values to take. It would import an authored numeric score into a model that refuses per-finding scores ([v1 SPEC §5.2](../spec/v1-spec.md)) and put a table nobody in this repo owns into the ranking path, which is the shape [ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) makes attestable and nobody has attested |
| **Grade per instance, from the subject's exposure** | A `critical` for an internet-reachable expired certificate and a `low` for an internal one. It contradicts ADR-0116's per-rule assignment, and exposure is already a rule of its own — `sensitive-port-reached-from-internet`, and `Exposure`'s two reaches ([ADR-0017](./0017-exposure-needs-both-legs.md)) — so it would say one fact twice and let the second saying damp the first |
| **Let the operator re-rate a rule from the console** | Makes the grade an operator dial, and every dial in this model sits outside every derivation and moves a message rather than a number. It would also make two installs' inboxes incomparable and put the argument this ADR writes down beyond review |
| **Fold the principle into [ADR-0024](./0024-a-rules-domain-is-the-extension-of-its-name.md)'s rule table as a sixth column** | That table's subject is the `Predicate domain` — which subjects a rule is asked about. A grade says nothing about the population and would put a ranking rule inside the document that exists to keep ranking out of the domain. ADR-0024's whole argument is that narrowing a rule to make it quieter is damping; a severity column beside it invites exactly that trade |
| **Fold it into [ADR-0116](./0116-the-design-package-is-normative-for-look-and-functionality.md)** | That ADR is `Superseded`, and [#1410](https://github.com/winniel123/verge-asm/issues/1410) fixed what survives it at one bullet. A live rating principle behind a `Superseded` status word is a rule readers are told to distrust |
| **Say nothing — the grades are shipped and stable** | They are stable and undefended. The grade is the one part of a rule an operator argues with, ADR-0185 makes a re-rating free of any `Break`, and the only written argument for the current bands was in a comment #1302 deleted. The next re-rating request would be settled by whoever answered it |
