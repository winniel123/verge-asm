# The value space is the commitment, not the signal set

- **Status:** Accepted
- **Date:** 2026-08-13
- **Ticket:** [#41 Does the observation-driven listener rule ship in v1, and at what granularity?](https://github.com/winniel123/verge-asm/issues/41)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[#31](https://github.com/winniel123/verge-asm/issues/31) established that an observation-driven
insecure-listener rule **can** be built without crossing
[#5](https://github.com/winniel123/verge-asm/issues/5)'s fingerprinting line, and
[ADR-0014](./0014-only-revealed-generalises.md) priced its arrival at `revealed` plus one message
with no `Break`. What neither settled is whether the thing is worth building, and
[`docs/research/insecure-listener-rules.md`](../research/insecure-listener-rules.md) §10 makes that
a real choice: the HTTP-shaped half costs nothing and covers
[#21](https://github.com/winniel123/verge-asm/issues/21)'s largest exclusion category, while the
wire-protocol prober costs a facet, a canonicaliser, a golden corpus of a new kind, per-protocol
encoders and an expanded safety surface. It buys **six ports**, adding **zero** net new firings
across all 38 rows of the sensitive list.

The ticket framed the decision with a deadline: *"adding a signal after v1 costs every operator a
comparability cycle; adding one before costs nothing."* That sentence is what this ADR overturns,
and everything else follows from replacing it.

## Decision

| Concern | Decision |
| --- | --- |
| The deadline the ticket asserted | **False.** Adding a signal post-v1 costs no `Break` at all |
| What *is* expensive to defer | **Widening an existing facet's value space** — that is a `Break` estate-wide |
| `http-identity` records the status class | **Yes, in v1**, whether or not any rule reads it |
| The HTTP-shaped rule | **Ships**, as one signal: `unauthenticated-request-answered` |
| The wire-protocol prober and `listener-negotiation` | **Does not ship in v1.** Out of scope for this map |
| `listener-offers-no-encryption` | **Never enters v1** — its HTTP half already shipped as #16's *Plaintext HTTP with no HTTPS* |
| Vantage gating on the new signal | **None.** ADR-0010 already refused it for the HTTP signals |
| A signal covering exactly one protocol | **Legitimate.** The defect is a signal named for a *protocol*, not one that covers a single protocol |
| SMB signing | Admissible in principle; **out of v1** for want of the prober, not for want of a principle |
| Two signals firing on one `Service` | **Both fire, no de-duplication** |

## Rationale

### The deadline was false, and the real deadline is one level down

A `Break` has two causes ([ADR-0008](./0008-derivation-versions-move-on-content.md)): a
`Derivation` vector moved, or the aperture widened. Test the ticket's claim against both.

Adding a **new signal** post-v1 moves no existing vector — it is a new rule, a new leaf, a new
timeline — and its own timeline is an **opening**, on which ADR-0014 records a `Break` is vacuous
because there is no predecessor to break from. Adding a **new facet** post-v1 is the same shape:
no `Service` held a `listener-negotiation` value, so every timeline opens, and ADR-0011's
strictly-additive rule prices it at `revealed` plus one coverage-class message carrying a count and
no comparison. Neither costs a comparability cycle. **The ticket's constraint does not survive its
own blockers**, and it was written before them.

But the symmetric case fails, and it is the finding. **Widening the value space of a facet that
already has observations is not additive.** ADR-0011's test is that every corpus row whose output
moved must previously have produced *no* observation. A new field on `http-identity` moves the
output of rows that all produced observations, so it **`Break`s every HTTP timeline in the
estate**.

So the expensive commitment is not the signal set at all:

> **A facet's value space is decided once and widened at the cost of a `Break`; the signal set over
> it is free forever.** Adding a facet is cheap, adding a field to an existing facet is not.

That inverts the ticket's urgency exactly. *Does this rule ship in v1* is a scope question with no
deadline. *Does `http-identity` record the status class* is a v1 decision that is expensive to get
wrong, and it must be answered **yes independently of whether any rule reads it** — the field is
the commitment, the rule is not.

Once the field is recorded, the rule over it costs nothing, covers #21's largest exclusion
category, and there is no reason left to hold it back. It ships because it became free, not
because a deadline forced it.

### The split falls out, and for a better reason than the cost table

The research note recommended the split on build cost. The version algebra recommends the same
split on a sharper criterion, and the two agreeing is the tell that it is right:

- The HTTP-shaped half is a **field in an existing facet**. Deferring it costs a `Break`
  estate-wide. **Ship it.**
- The wire-protocol half is a **wholly new facet**. Deferring it costs `revealed` plus one message.
  **Defer it.**

The rule is not *build the cheap thing first*. It is *commit to what is expensive to change and
stay uncommitted on what is not* — which is the same instinct ADR-0008 applied to derivation
versions and ADR-0011 applied to closed unions, one level up.

### What deferring the prober actually costs, stated rather than minimised

Three well-founded detections are lost from v1, and none of them is marginal:

- **AMQP on 5672** is the clearest case in the whole survey of reaching where the curated list
  cannot. RFC 4505 states that announcing `ANONYMOUS` *is* the fact, the field is mandatory, the
  exchange is eight constant bytes, and RabbitMQ ships the mechanism enabled by default and tells
  operators to remove it in production.
- **IMAP on 143** is the best-founded row in the note — RFC 9051 §6.2.3 makes `LOGINDISABLED` a
  conditional MUST, which is the same footing RFC 8996 gives `tls-1.0-accepted`, the strongest any
  signal in the v1 set has.
- **RDP on 3389**, which #21 had to exclude because remote administration over an untrusted network
  is what the protocol is *for*, and where *does this listener require CredSSP* is a question the
  port could never answer.

Set against that, a cost the note does not analyse: **§7.2 argues a wrong dispatch guess fails safe
for the data, and never asks whether it is safe for the listener.** Speaking PostgreSQL's
`SSLRequest` at whatever is actually on 5432 is a new class of risk against production that
[#4](https://github.com/winniel123/verge-asm/issues/4)'s profile has not budgeted for. That gap is
a reason to defer rather than a reason to refuse, and it is the first thing the ticket that picks
this up should close.

### The new signal reads a fact, not a conclusion

`unauthenticated-request-answered` fires where the `GET /` v1 already performs — sent with no
`Authorization` header, as v1 always does — returned a `2xx` status class rather than `401` or
`403`. Reference data: RFC 9110's status classes. **Zero rows**, which is
[#31](https://github.com/winniel123/verge-asm/issues/31)'s zero-row verdict table in its purest
form.

The name is doing deliberate work. [ADR-0010](./0010-exposure-composes-two-reaches.md) established
that a signal named for a conclusion its evidence cannot carry is the `Host` defect, and
*anonymous access is permitted* is exactly such a conclusion — a `200` on `/` may be a login page,
and separating those needs per-product knowledge that is a verdict table. The measurement is
`401`/`403` versus `2xx` and the name says so.

Three riders bound it:

- **A `3xx` is neither.** A redirect to a login page is a refusal in practice and a `302` in the
  standard, and calling it either would be a judgement RFC 9110 does not license. It lands outside
  the predicate, so the signal does not fire — which is honest, because the endpoint did not answer
  with a representation.
- **The rule reads the first response and never follows a redirect.** Following one is a second
  request and an aperture widening. #16's *redirect that does not upgrade to TLS* already reads the
  `Location` **header** rather than chasing it, so this inherits a settled habit rather than
  inventing one.
- **`WWW-Authenticate` is corroboration, never a requirement.** RFC 9110 §15.5.2 makes it a MUST
  and `api.github.com` was measured returning a `401` without it, so keying on the header would
  report a well-protected endpoint as unprotected.

### No vantage gate, and the commonness is not a defect

The instinct is to fire only from an internet vantage, since an unauthenticated app on an intranet
is ordinary. **ADR-0010 already considered and refused exactly this** for the cert, TLS and HTTP
signals: their evidence presupposes a reach, so a gate "would change nothing except to make
internally-observed defects `not-evaluable` — reporting less than we measured, in order to express
a severity the model refuses to carry." That binding is inherited unchanged, so the signal's
effective vector is `{ the rule, http-identity-canon }` — **two leaves**, reading no Derived value,
against the six the research note's §9.3 projected for a vantage-composing design.

Which leaves the honest objection: the signal is **true of most of the estate**, because most
listeners answering `GET /` are meant to. The note killed the connect-plus-handshake rule in §3.2
on the words *a signal that fires on most of what responds is not a signal*, and that sentence does
not transfer here. §3.2's rule was **conflated** — one value covering plaintext-by-design,
edge-terminated TLS and genuinely-absent TLS. This one is precise and merely common, and the two
are different failures.

Commonness is not disqualifying in this model by construction. ADR-0004 settled that signals carry
no severity and ADR-0010 restated that the differential belongs to the transition that surfaced it.
So a broad, honest, unranked census whose **transitions** are the product's subject is the design
working, not failing: `refused` → `answered` on an endpoint that was requiring authentication
yesterday is precisely what an operator wants waking for, and its base rate is low.

The consequence is a presentation obligation and it is routed, not solved here: a surface must not
render this signal's census as a findings list. Any narrowing of the *rule* to reduce the noise
would be model-layer damping, which ADR-0007 refused outright because a flap count is a fact the
operator wants and damping destroys it permanently.

### A second representation? No — that objection would delete the whole signal set

The status code is already in `http-identity`, and `401` → `200` is already a `Transition` on that
timeline. So the signal looks like a second representation of one fact, the shape ADR-0007 refuses.

It is not, and the test is general. Every signal in #16's v1 set is a **named projection of facet
values already stored**: certificate expiry reads `not_after`, self-signed reads issuer against
subject, `sensitive-port-reached-from-internet` reads a `Reach` timeline. `CONTEXT.md` defines a
`Signal` as *a named, versioned rule evaluated over observations* — projection is what a signal
**is**. What ADR-0007 refuses is storing a second copy, and a signal stores nothing: it has no
lifecycle of its own, its lifecycle is its evidence's.

Recorded because the objection is a natural one and a future session will raise it again.

### A signal covering one protocol is legitimate; one *named* for a protocol is not

The research note's §9.2 excluded SMB signing because "a rule covering one protocol is a
per-protocol signal by another name", and its §12 q2 doubted the reasoning. The doubt is correct
and the principle as stated is wrong.

§9.1's actual harm is **growth coupled to a corpus** — "a signal set that grows with the dispatch
table is a name per product, which is the signature database's silhouette." That harm is not
caused by covering one protocol. It is caused by the signal being *named for* the protocol, so
that admitting one commits us to admitting the next. The corrected test:

> **A signal is named for the fact it reads, and its scope is however many protocols happen to
> express that fact.** One is fine when the fact is genuinely single-protocol. Three protocols
> expressing one fact must be one signal, never three.

That is [ADR-0010](./0010-exposure-composes-two-reaches.md)'s naming rule with the count removed
from it, and it is why `smtp-starttls-absent` / `imap-starttls-absent` / `pop3-stls-absent` are
forbidden while a single-member rule is not: the first three are one fact wearing three names.

SMB signing is therefore admissible whenever the prober that can read
`SMB2_NEGOTIATE_SIGNING_REQUIRED` exists. It stays out of v1 because that prober does not, and it
would be a third signal in any case — signing is **integrity**, which is neither of the two facts
the other rules read.

### Overlapping signals both fire, and suppression would fuse their versions

`unauthenticated-request-answered` and `sensitive-port-reached-from-internet` both fire on the
HTTP-shaped rows of the sensitive list — Elasticsearch, CouchDB, kubelet, Docker. On #21's
determinacy-excluded ports, which is where the new coverage actually is, there is no overlap at all
by construction, since those ports are precisely the ones the sensitive list does not carry.

Both fire, and nothing de-duplicates them. The taste argument — two signals saying nearly the same
thing is ranking pressure — is severity through the back door, refused by ADR-0004. The structural
argument is the one that settles it: **making one rule's firing depend on another's would fuse
their `Derivation` vectors**, so a version move in either breaks both, and ADR-0004's per-rule
versioning exists precisely so one edit never silences the set. Which signal to show first is a
presentation question, and it belongs to
[#44](https://github.com/winniel123/verge-asm/issues/44) and
[#22](https://github.com/winniel123/verge-asm/issues/22).

### Deferring the prober deletes the `not-evaluable` problem it would have created

The note's §11 adds three `not-evaluable` routes beyond the four inherited from
`sensitive-port-reached-from-internet`, and calls route 5 — the protocol is not in the dispatch
table — "the largest new route", covering most of the sensitive list permanently. Route 6, the
fact being unreachable inside the credential line, is permanent for every protocol outside the six.

**All three arise only from the wire prober.** There is no dispatch table to miss, no hello to go
unanswered, and for HTTP the authentication fact is reachable wherever a response was obtained. So
the ticket's fourth question — what the `not-evaluable` surface looks like — resolves to *no new
routes at all*. #41 hands #44 nothing it did not already have, which is worth stating because the
question was posed on the assumption that the prober ships.

## Consequences

- **`http-identity`'s value space must carry the status class in v1**, independently of this
  signal. It is the one part of this decision with a deadline, and #16's redirect signals already
  depended on it implicitly.
- **The v1 signal set gains exactly one member**, `unauthenticated-request-answered`, over an
  existing facet and an existing subject, with a two-leaf version vector and zero rows of reference
  data. No new facet, prober, dispatch table, canonicaliser, golden corpus or safety surface.
- **[`CONTEXT.md`](../../CONTEXT.md) is amended in two entries** — `Facet` records that widening an
  existing value space costs a `Break` while adding a facet does not, and `Signal` records the
  named-for-the-fact rule and that single-protocol scope is not disqualifying.
- **The land-grab argument is dead map-wide.** Any future ticket arguing for v1 inclusion on
  *adding it later costs comparability* is using a constraint ADR-0011 and ADR-0014 withdrew. The
  argument survives in exactly one form: widening an existing facet's value space.
- **The wire-protocol prober is out of scope for this map**, with §5's per-protocol costing
  preserved in the research note for whoever picks it up. The first thing that ticket must close is
  the safety question §7.2 skipped.
- **The research note's §12 is discharged**: q1, q2 and q5 are answered here, q3 was answered by
  ADR-0014, q4 and q7 travel with the deferred prober, and q6 belongs to
  [#33](https://github.com/winniel123/verge-asm/issues/33)/[#37](https://github.com/winniel123/verge-asm/issues/37).
- **A presentation obligation lands on #44**: a signal that is true of most of the estate must not
  be rendered as a findings list, and the noise may not be fixed in the rule.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| Ship both halves in v1 | Buys six ports and zero net new firings on the sensitive list, for a facet, a canonicaliser, a new kind of corpus and an unbudgeted safety surface — and deferring it is now priced at one message |
| Ship neither | Leaves `http-identity` without the status class, which is the one choice here that is expensive to reverse |
| Defer the HTTP rule but record the status class anyway | Coherent, and it buys nothing: once the field is recorded the rule is free, and it covers #21's largest exclusion category |
| Gate the new signal on internet reach | Refused by ADR-0010 for these exact signals — severity disguised as evaluability |
| Narrow the rule so it fires less often | Model-layer damping, refused by ADR-0007; the flap count is a fact the operator wants |
| Name it `listener-permits-anonymous-access` | Claims a conclusion the evidence cannot carry — a `200` on `/` may be a login page. The `Host` defect ADR-0010 named |
| Treat a `3xx` as a refusal | A judgement RFC 9110 does not license, and the first row of a verdict table |
| Suppress the second signal where both fire on one `Service` | Severity through the back door, and it would fuse two `Derivation` vectors so a move in either breaks both |
| Keep §9.2's rule that a single-protocol signal is illegitimate | Excludes rules for the wrong reason; the harm is a name per protocol, not a scope of one |
| One combined `insecure-listener` signal | Different evidence, different remediation — it would fire without telling the operator which thing to do |

## Annotation — [#104](https://github.com/winniel123/verge-asm/issues/104), 2026-08-14: the SMB row is ratified, and it leaves one obligation behind

The Decision row *"SMB signing — admissible in principle; **out of v1** for want of the prober, not
for want of a principle"* **stands unchanged**, and #104 ratified it after carrying it back to
[`insecure-listener-rules.md`](../research/insecure-listener-rules.md) §9.2, which had never heard of
it. ~~**The v1 rule set stays at sixteen.**~~ **SEVENTEEN since
[#128](https://github.com/winniel123/verge-asm/issues/128)**, which admitted
`non-globally-reachable-address-resolved-from-internet`
([ADR-0071](./0071-a-vantage-scoped-claim-is-read-only-at-the-vantage-that-scopes-it.md)) — marked at
the sentence per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)
as widened by [#106](https://github.com/winniel123/verge-asm/issues/106). **The SMB row and this ADR's
own ruling are untouched**: #128 adds a rule whose aperture already exists, which is the opposite case
to the one deferred here and does not disturb it.

Two things this ADR did not say, added rather than corrected.

**Both of §9.2's grounds are gone, not one.** This ADR withdrew the per-protocol half explicitly. Its
surviving half — *"it is integrity rather than confidentiality, so it fits neither rule"* — is a claim
about §9.2's **two rules** and not about SMB, so it never excluded anything. This ADR's own *"it would
be a third signal in any case"* already treated *third* as the shape of the answer. Generalised as
[ADR-0065](./0065-a-rule-is-excluded-by-its-fact-or-by-its-aperture-never-by-the-shape-of-the-set.md):
an exclusion rests on **the fact** or on **the aperture**, and the shape of the existing set is
neither.

**The deferral is free for the rule and not for the field.** This ADR's own central finding applies to
the thing it deferred: the integrity fact is neither of `listener-negotiation`'s two proposed fields
(§7.4), so it needs a **third** — and a facet's value space is decided **once**. While that facet does
not exist the third field costs nothing either way. The day it ships with two fields, adding the third
`Break`s every timeline it holds. So whoever builds the prober owes the **field** decision at
specification time, ahead of and independently of any rule reading it — exactly what this ADR ruled
for `http-identity`'s status class. Recorded at §8.2.

**One thing this ADR left open is closed rather than carried forward.** The corrected test — *"its
scope is however many protocols happen to express that fact"* — is a question, and #104 answered it by
measurement: across eight protocols read against their own specifications, SMB is the **only** one
where the integrity requirement is simultaneously server-originated, pre-credential, listener-scoped
and a requirement rather than a capability. LDAP and SNMPv3 do not express it before authentication at
all, NFS expresses it per-export or per-filehandle and (v3) mutates the server's mount list to answer,
and SSH's answer is a constant *required*. So the rule is legitimately single-protocol, and the
conclusion inverts the instinct: **a cross-protocol integrity abstraction would carry one real
implementation and a set of degenerate ones.** The table is at
[`insecure-listener-rules.md`](../research/insecure-listener-rules.md) §9.2.
