# ADR-0004: Signals are release-coupled rules, and comparability is versioned per rule

- **Status:** Accepted
- **Date:** 2026-08-13
- **Ticket:** [#16 Risk signals: which ones does v1 emit from cert and protocol facts alone?](https://github.com/winniel123/verge-asm/issues/16)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

The map promises that v1 reports *"risk signals that fall out of probing for free"*, and
[#7](https://github.com/winniel123/verge-asm/issues/7) fixed what a `Signal` is — a named,
versioned rule evaluated over observations, citing its evidence, with no lifecycle of its
own. What was never settled is **which** signals ship, and that turns out to be blocked on a
prior question: where the line sits between a legal signal and the technology fingerprinting
[#5](https://github.com/winniel123/verge-asm/issues/5) rejected.

The obvious lines do not cut. *"Does the rule contain a human judgement?"* excludes nothing —
*TLS 1.0 is weak* is a judgement. *"Is the evidence verifiable by the operator?"* excludes
nothing either; a fingerprint's banner is right there to read. Meanwhile the candidate signal
list contained items — matching admin-panel titles, default-install pages — that are plainly
a signature database wearing a different hat.

The harm #5 actually identified is narrower than "judgement": it is **reference data mutating
underneath a comparison**, so the estate re-diffs without anyone having shipped anything. That
points at the property that does cut.

## Decision

**A rule may ship as a `Signal` only if its reference data changes at release cadence.** The
test is a question about our own intentions: *would we ever want to push updates to this list
out of band?* If yes, it is a signature database and it is out. In practice the proxy is
whether the reference set is **closed and enumerable** (protocol versions, key algorithms,
dates, port numbers, the operator's own seeds) or an **open corpus that grows without bound**
(page titles, banners, default-install fingerprints).

Four rules follow from treating signals this way.

**No severity.** A signal is a named fact with evidence. Severity exists to rank a static
backlog, which is the `Finding` mental model [#7](https://github.com/winniel123/verge-asm/issues/7)
rejected; in a product whose subject is change, urgency comes from the transition that
surfaced the signal.

**Absent evidence yields `not-evaluable`, never "did not fire."**
[ADR-0002](./0002-ownership-gates-probing.md) ~~limits probing on `third-party` addresses to the
ports the `Name` implies~~ — **amended by [#118](https://github.com/winniel123/verge-asm/issues/118)
/ [ADR-0019](./0019-the-probing-gate-is-total-over-an-address.md): the gate is **total** over the
address and no port is probed there at all** — so port-facts rules have no evidence there.
Reporting that as clean would give every SaaS-fronted address in the estate a bill of health it
never earned. The correction makes the sentence *stronger*, not weaker: it is not that only some
ports are unread there, it is that none is.

**A new signal is alertable if and only if the effective rule version was unchanged between
the two evaluations.** A signal is a pure function of its inputs and its rule; hold the rule
constant and any change in the signal set is attributable to the world, which makes it drift.
Across a rule change the two sets are **not compared at all** and the presentation is *"your
rules changed"*.

**Signals may read Derived values, and their effective version composes the versions of
everything they read.** Versions are **per rule**, not one global rule-set version.

### Amendment — [#35](https://github.com/winniel123/verge-asm/issues/35): the test is the gate; the v1 list is its output

The cadence test above is the **only** bar a rule must clear to ship as a `Signal`. The v1
set named under Consequences is that test's *output* at the time [#16](https://github.com/winniel123/verge-asm/issues/16)
ran, not a second gate a later rule must also pass, and "the set is closed" was a fact about
that session rather than a rule anyone adopted.

This is stated because the opposite reading was already costing real work.
[#8](https://github.com/winniel123/verge-asm/issues/8) declined to admit *zone-declared names
that do not resolve* and recorded it as fog explicitly on the grounds that the set was
closed, and [#17](https://github.com/winniel123/verge-asm/issues/17) declined to assert
`lame-delegation` for the same reason. Both were right to defer to this ADR and wrong about
what it says.

So a candidate rule is admitted on the cadence test alone. What a proposal still owes is the
ordinary accounting any addition owes — what it measures, what its `not-evaluable` case is,
and what new measurement it requires — and the last of those is a **scope** cost to be
weighed, never a correctness objection. A bar erected to protect the list rather than the
property would have excluded #35 on exactly that confusion.

### Amendment — [#44](https://github.com/winniel123/verge-asm/issues/44): `not-evaluable` needs a subject, and this ADR's worked example does not have one

The rule above stands unchanged. What is corrected is its **example**, and the correction
matters because the example is what an implementer would build from.

[ADR-0014](./0014-only-revealed-generalises.md) settled that a `Batch` whose recorded scope
excludes a thing **never touches its timeline**, so a `Gap` — which is a span — does not open
where there was never a timeline to interrupt. Applied here: on a `third-party` address ~~only
the ports the `Name` implies are probed~~ **nothing is probed at all**
([#118](https://github.com/winniel123/verge-asm/issues/118) /
[ADR-0019](./0019-the-probing-gate-is-total-over-an-address.md)), so no `Service` on **any**
`(port, transport)` pair — sensitive-list or otherwise — is ever *observed* there. No subject
exists, and the population is empty rather than merely thinned. A signal cannot return
`not-evaluable` about a subject that is not in the estate, so the sentence *"port-facts rules
have no evidence there"* describes something real that this term cannot express.

The **harm** the rule names is unchanged and is if anything larger, because it arrives through
the **census** rather than through a per-subject outcome: a rule reporting *0 fired, 0 did not
fire* over a population that is empty for an aperture reason is the bill of health the estate
never earned, written as a number instead of a word. The modal cloud-resident install of
[#26](https://github.com/winniel123/verge-asm/issues/26) with the `custody extension` left off
— correctly left off — hits exactly that.

So the rule is carried by **two** mechanisms, split on whether the subject exists:

- **The subject exists and its evidence does not** — per-subject `not-evaluable`, on `Signals`,
  as the third member of the rule's census, counted over the rule's **predicate domain** and
  never over the timelines it happens to hold. Each row carries the cause, which is read off
  the `Gap` where there is one.
- **No subject exists, because we never looked** — no row is possible, ever. It is an
  **aperture** fact and it renders as a standing statement on `Coverage`, carrying counts of
  our own rules and lists (closed and enumerable) and never a count or proportion of the
  operator's estate, for the reason [#28](https://github.com/winniel123/verge-asm/issues/28)
  gave the propose half of source coverage.

The same correction applies to [#29](https://github.com/winniel123/verge-asm/issues/29)'s claim
that a batch measuring no UDP makes the six UDP pairs *"report `not-evaluable` … obtained for
free rather than by policy"*. They report nothing at all: no `Service` is observed on 161/udp,
so there is no subject to hold the outcome. That is why the string appears nowhere in #28's
prototype, and why [#21](https://github.com/winniel123/verge-asm/issues/21) §6.1's stated
justification needed a screen rather than an inference.

### Amendment — [#53](https://github.com/winniel123/verge-asm/issues/53): what a proposal owes is four parts, and the first of them is the domain

The #35 amendment above calls its list *"the ordinary accounting any addition owes"* and leaves it
informal. [ADR-0024](./0024-a-rules-domain-is-the-extension-of-its-name.md) makes it structural,
because #44's census gave the missing item a consumer: the **predicate domain** is the denominator
the operator reads, and nothing said where it was written down.

A rule is **four parts** — its `Predicate domain`, its predicate, its `not-evaluable` case, and its
version vector — plus one **cost**, the new measurement it requires, which stays exactly what this
ADR already made it: weighed, never a correctness objection. The parts are gated by the rule's
golden corpus, whose row shape gains a fourth outcome, *outside the domain*.

Two things this ADR says elsewhere are sharpened rather than changed. *Absent evidence yields
`not-evaluable`* is now one of three registers and not two: a value about the **world** under which
the rule's question does not arise puts the subject outside the domain, a value about **our own
sight** yields `not-evaluable`, and no value at all is a `Gap`. That is why `Shadowed` keeps the
`not-evaluable` this ADR gives it under both DNS rules while `NoTLS` does not get it under the
certificate rules. And the v1 set's *plaintext HTTP with no HTTPS* loses the port literal it had
been carrying: its domain is *`Endpoint`s that answered HTTP*, so it fires on plaintext HTTP wherever
it listens rather than on 80/tcp alone.

### Amendment — [#60](https://github.com/winniel123/verge-asm/issues/60): `N` is release-coupled, and both halves of *"one rule, operator-configurable N"* are withdrawn

The Consequences below read *"expiring within N days (one rule, operator-configurable N)"*. That
parenthetical stands unrewritten, per this repo's name-and-withdraw convention, and **both of its
halves are withdrawn**. They were wrong for different reasons and only one of them had been
noticed.

**`N` is not operator-configurable.** It is a declared parameter of `certificate-expiring`, so
under [ADR-0008](./0008-derivation-versions-move-on-content.md) it sits inside that rule's own leaf
and under [ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md) a changed declared
parameter is one of the three things that may move a version. A dial on it is therefore a settings
field that `Break`s a rule — uniformly, since ADR-0008 refused predicate-scoped breaks — clamping
every `certificate-expiring` timeline in the estate, rendering its durations as labelled floors for
a full window, and firing a re-baseline message. `N` is now **fixed at the release**, project-authored
like every other declared parameter, and ~~shipped at **30 days**~~ **shipped as one third of the
certificate's own validity period** — one half where that period is 10 days or less.

> **SUPERSEDED** by the [#67](https://github.com/winniel123/verge-asm/issues/67) amendment below,
> which names this very phrase. `N` is no longer a number of days at all. That amendment says the
> phrase *"stands unrewritten per this repo's name-and-withdraw convention"* — but the convention is
> *left standing **and marked***, never *left standing unmarked*, so it is marked here per
> [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) as widened
> by [#106](https://github.com/winniel123/verge-asm/issues/106). Everything else in this paragraph —
> that a dial would `Break` uniformly and clamp every timeline — stands.

Three things kill the dial, and the first two were already on `main`.

[#35](https://github.com/winniel123/verge-asm/issues/35) ruled this for both DNS rules and its
reasoning transfers with nothing changed: *"[#22](https://github.com/winniel123/verge-asm/issues/22)
drew the configurable/fixed line at inside-versus-outside the comparison path. A signal is squarely
inside, so an operator-configurable `N` was out on existing precedent."* That was this question
answered, for two rules, and never carried back to the one rule that had an `N`.

This ADR's own stated reason for the dial does not survive
[ADR-0015](./0015-the-value-space-is-the-commitment.md).
[#16](https://github.com/winniel123/verge-asm/issues/16) made `N` configurable because *"a 30-day
warning is right for manual renewals and noise for a team on ACME"* — and **noise is not a defect**:
ADR-0015 settled that a signal being true of most of the estate is the design working, that the
transitions are the subject, and that narrowing a rule to quieten it is the model-layer damping
[ADR-0007](./0007-drift-is-a-timeline-of-spans.md) refused. An operator-configurable `N` is that
same damping with the dial handed outward, which is worse than the version it refused, because the
person pulling it cannot see what it costs.

And the corpus stops gating the thing that runs. ADR-0008 makes the golden corpus the mechanism
that replaces discipline, and [ADR-0014](./0014-only-revealed-generalises.md) calls it *"the whole
guarantee"*. Under a per-install `N`, CI gates one function and every install evaluates a different
one, under the same version string — so two installs on one release hold different content behind
one leaf, and a leaf is meant to name what decided the output.

**"One rule" is withdrawn too, and [ADR-0024](./0024-a-rules-domain-is-the-extension-of-its-name.md)
had already withdrawn it without saying so.** ADR-0024's v1 domain table names
`certificate-expired`, `-not-yet-valid` and `-expiring` as three rules sharing one domain, and this
ADR's own per-rule versioning and #44's per-rule census make three names three leaves and three
censuses. #16's *"expired is N=0"* was a compression, not a model.

Two consequences fall out of the split, and the second corrects a claim made below.

**Of the three clock-reading rules, exactly one contains a number we chose.** `certificate-expired`
and `certificate-not-yet-valid` compare `not_after` and `not_before` against the clock and take no
parameter — they are world facts. Only `-expiring` carries a horizon, which is what makes it the
only member of the family that could ever have been dialled.

**The clock class has three members, not one**, so the Consequences sentence below — *"a
clock-crossing signal — a certificate expiring"* — understates it. All three certificate-lifetime
rules become true or false with no new observation, and they are the only rules in the v1 set of
which that is true.

Two obligations follow for the corpus and for what may be said about `N`.

**A corpus row for a clock-reading rule carries its evaluation instant as part of its input.**
Without it a row's output moves every day with no version change, and ADR-0021's bidirectional gate
either fails the build for nothing or passes by accident. This is the only place in the corpus where
an input is not a measurement.

**`certificate-expiring` is the second signal whose reference data we curate**, which falsifies the
Consequences claim below that `sensitive-port-exposed` is the only one. `N` passes the cadence test
trivially — one integer is as closed and enumerable as reference data gets — and it is fixed by
fiat on the same footing as `k` and the availability window, revisable at release cadence for one
`Break` on one rule for one cadence. Unlike `k`, it makes a claim about the **world** rather than
about our own measurement, so the reason is stated rather than buried: ~~30 days is where the ACME
clients in the modal estate already trigger renewal, so it is the last point at which the operator
still has the action the signal is telling them to take.~~

> **WITHDRAWN IN BOTH HALVES** by the [#67](https://github.com/winniel123/verge-asm/issues/67)
> amendment below. The first half is a **frequency** claim, excluded from evidence by
> [#21](https://github.com/winniel123/verge-asm/issues/21) §2.5 and **false as of retrieval** —
> Certbot removed its fixed 30-day threshold in 4.0.0 and lego in v5. The second half does not follow
> from the first and follows against it. `N` is **one third of the certificate's validity period**,
> one half where that period is 10 days or less.

**The operator gets no dial, and none is minted elsewhere.** #16's real complaint — an ACME estate
renewing at 30 days makes the rule fire and clear every cycle — is a **flap**, and ADR-0007 put all
damping in notification precisely so the model keeps the flap count as a fact. ~~The remedy is the
notification layer's existing suppression~~, not a second constant and not a per-operator threshold.
If one is ever wanted it is legal, because #22's line puts it outside the comparison path — but it
is not wanted now, and minting it here would be the second constant
[#28](https://github.com/winniel123/verge-asm/issues/28) refused.

> **`existing` is WITHDRAWN** by [#119](https://github.com/winniel123/verge-asm/issues/119) /
> [ADR-0039](./0039-a-channel-carries-the-message-never-the-estate-and-a-delivery-is-an-operational-record.md).
> There was no notification-layer suppression to be the remedy, and this paragraph says so itself two
> sentences later — *"if one is ever wanted it is legal ... but it is not wanted now"*. The
> contradiction was internal to one paragraph. **v1 ships none**: no suppression, no coalescing, no
> digest window. The ACME flap's actual v1 treatment is **routing by class**, and whether that
> reaches it turns on an unsettled class assignment — ADR-0026 §5 puts `not-fired` → `fired` in the
> **drift** class for all sixteen rules while #60 and this ADR put the three certificate-lifetime
> rules in the **clock** class. Flagged to
> [#120](https://github.com/winniel123/verge-asm/issues/120), which owns the causes.

### Amendment — [#33](https://github.com/winniel123/verge-asm/issues/33): the curated count is three, and what licenses a table's content is a separate question from this ADR's

Two things are corrected, both in the Consequences below, and neither touches this ADR's test.

**"`sensitive-port-reached-from-internet` is the only signal whose reference data we curate" is now
wrong by two.** #60's amendment above already made it two; walking all sixteen rules for
[ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) makes it **three**. The
third is **`certificate-weak-key-or-signature`**, whose key-size floor and deprecated-algorithm set
have never been written down anywhere in this repo — it appears in the Consequences list below and in
[ADR-0024](./0024-a-rules-domain-is-the-extension-of-its-name.md)'s domain table, and nowhere else.
That is [#68](https://github.com/winniel123/verge-asm/issues/68), which **blocks**
[#12](https://github.com/winniel123/verge-asm/issues/12).

> **Closed by #68.** The table is now written —
> [`docs/research/weak-key-and-signature.md`](../research/weak-key-and-signature.md), five rows, with
> [ADR-0035](./0035-a-cryptographic-primitives-owner-is-its-specifier.md) recording the general parts.
> **It passes this ADR's cadence test with three orders of magnitude of headroom** — roughly one edit
> per row per decade, measured over twenty-two years — and #68 §7 names the two things that would have
> failed it: a **key-compromise blocklist**, which grows continuously, and **encoding NIST's dated
> transitions as date comparisons**, which would move the rule's output at midnight with no version
> bump. The count of curated tables stays at three.

**The cadence test is not an evidence standard, and this ADR should not be read as one.** The test
here asks *may this ship as a rule* — a question about how often reference data may move.
[#21](https://github.com/winniel123/verge-asm/issues/21)'s claim/attestation/determinacy standard
asks *is this row's content licensed*, and ADR-0032 rules that it attaches to a **table**, never to a
rule, so thirteen of the sixteen rules have nothing for it to govern. Two questions, two documents:
this ADR's sentence *"that is a statement about cadence, not correctness — hence #21"* is exactly
right and is the boundary. What a table asserting about the **world** additionally owes — an
attestation per row, its own closed claim set derived from what its rule reads, and a determinacy
argument where its key is a surrogate — is ADR-0032's, and it is the **table's** accounting rather
than a fifth part of a rule.

### Amendment — [#67](https://github.com/winniel123/verge-asm/issues/67): `N` is a fraction of the certificate's own lifetime, and the #60 rationale above is withdrawn

The #60 amendment's ruling on `N`'s **status** is untouched and is confirmed: `N` is a declared
parameter, project-authored, fixed at the release, and never operator-configurable. What moves is
its **value**, and what is withdrawn is the sentence that justified the value.

**`N` is no longer a number of days.** It is **one third of the certificate's validity period
(`not_after − not_before`), and one half of it where that period is 10 days or less.** *"shipped at
**30 days**"* above stands unrewritten per this repo's name-and-withdraw convention and is
superseded. [ADR-0034](./0034-derive-the-claim-before-looking-for-the-owner.md) holds the reasoning;
[`docs/research/acme-renewal-timing.md`](../research/acme-renewal-timing.md) holds the retrieval.

**The rationale sentence above is withdrawn in both halves.** It reads: *"30 days is where the ACME
clients in the modal estate already trigger renewal, so it is the last point at which the operator
still has the action the signal is telling them to take."*

- The first half is a **frequency** claim, which
  [#21](https://github.com/winniel123/verge-asm/issues/21) §2.5 excludes from evidence — and it is
  **false as of retrieval**. Certbot removed its fixed 30-day threshold in 4.0.0 (*"Prior to Certbot
  4.0.0 the threshold was a fixed 30 days"*), lego removed the same default in v5, and cert-manager
  never had one. Only `acme.sh` still ships a flat 30.
- The second half **does not follow from the first, and follows against it.** If the modal client
  renews at 30 days, a certificate at 30 days remaining is one whose automation is firing on
  schedule — which is the moment the operator has *no* action, not their last chance to take one.

**Its replacement is attested by two owners.** The **IETF** attests the *form*: RFC 9773 §1 names a
fixed lead time as creating *"significant barriers against the issuing Certification Authority (CA)
changing certificate lifetimes"*, and pointedly does not so name a percentage of validity. The
**issuing CA** attests the *value*: Let's Encrypt's Integration Guide recommends renewing *"when
they have a third of their total lifetime left"*, halving *"for certificates with a validity period
under 10 days"*, and `boulder` implements exactly that.

**What forces the change rather than merely recommending it.** Certificate lifetimes are now plural
and shrinking — Let's Encrypt's default is 90 days, its `tlsserver` profile 45, its short-lived
profile 160 hours and generally available since 2026-01-15, with the CA/Browser Forum's ceiling
stepping to 100 days in 2027 and 47 in 2029. Under a fixed `N = 30`, `certificate-expiring` is
**true for the entire life of every six-day certificate**, so the predicate stops partitioning and
[#53](https://github.com/winniel123/verge-asm/issues/53)'s census carries no information over that
population. No fixed day count survives the spread.

**Three things this does not do.** It mints **no dial** — #60's three grounds all concern
*per-install* variation and a per-certificate fraction is identical in every install, so ADR-0021's
gate still gates exactly the function every install runs. It changes **no measurement** —
`not_before` is already read by `certificate-not-yet-valid`. And it does not disturb the ACME **flap**
routing: the rule still fires and clears every cycle on a healthy ACME estate, and
~~[ADR-0007](./0007-drift-is-a-timeline-of-spans.md)'s notification-layer damping is still the
remedy~~ — **withdrawn** by [#119](https://github.com/winniel123/verge-asm/issues/119) /
[ADR-0039](./0039-a-channel-carries-the-message-never-the-estate-and-a-delivery-is-an-operational-record.md)
on the same ground as the #60 amendment above: ADR-0007 granted a **licence** to damp there and no
damping was ever built, and v1 builds none. The routing is undisturbed either way — the point this
sentence was making survives; the mechanism it named does not.

**One obligation widens.** The #60 amendment requires a corpus row for a clock-reading rule to carry
its **evaluation instant** as part of its input. It now carries **`not_before`** as well, or the
row's expected output is underdetermined.

## Consequences

- **The v1 set is:** certificate expired / not-yet-valid / expiring within ~~N days (one rule,
  operator-configurable N)~~ **a fraction of the certificate's own lifetime** — see below —,
  self-signed, hostname-SAN mismatch, weak key or signature
  algorithm, TLS 1.0/1.1 negotiated, plaintext HTTP with no HTTPS, a redirect that does not
  upgrade to TLS, a redirect to a host outside the operator's estate, and
  `sensitive-port-exposed`. Excluded: directory-listing pages, default-install pages,
  admin-panel titles.
  > **Both halves of *"one rule, operator-configurable N"* are WITHDRAWN** by the
  > [#60](https://github.com/winniel123/verge-asm/issues/60) amendment above, which names this very
  > sentence, and the horizon was then re-specified by the
  > [#67](https://github.com/winniel123/verge-asm/issues/67) amendment as a **fraction of the
  > certificate's own lifetime**. `N` is a **declared parameter**,
  > project-authored and fixed at the release; it is **never operator-configurable**. Marked here
  > rather than only at the amendments, per
  > [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) as widened
  > by [#106](https://github.com/winniel123/verge-asm/issues/106).
- **Two DNS signals join the set** under [#35](https://github.com/winniel123/verge-asm/issues/35):
  **`lame-delegation`** (the `Name`'s `resolution` is `Lame` — its delegated nameservers were
  reached and none serves it) and **`cname-target-name-error`** (the CNAME chain terminates at
  a target returning a Name Error). Both carry **no reference data at all**, which is a cleaner
  pass than anything already in the set. Both are named for the predicate they read rather than
  the risk they suggest, per [ADR-0010](./0010-exposure-composes-two-reaches.md) — *takeover* is
  a conclusion DNS-only evidence does not establish, and it belongs in the drill-down prose
  where it can be hedged. Neither contains a threshold: persistence is the span's duration, not
  a count the rule takes. On a `Shadowed` name both yield **`not-evaluable`** — there is no
  delegation of its own to walk and no chain of its own to follow.
- **Two source-disagreement signals join the set** under
  [#48](https://github.com/winniel123/verge-asm/issues/48) and
  [ADR-0020](./0020-a-conflict-needs-two-enumerable-sources.md):
  **`zone-declared-name-returns-name-error`** (the operator's zone file holds a record for a
  `Name` whose `resolution` composes to `NameError` across every available vantage) and
  **`resolved-name-absent-from-zone`** (a `Name` whose `resolution` composes to `Resolved`, inside
  a zone the operator supplied, that the file does not contain). Zero rows of reference data, no
  new measurement, and **no new field on any facet** — so neither carried an
  [ADR-0015](./0015-the-value-space-is-the-commitment.md) deadline; they ship because the zone
  file's `completeness` has no other consumer. Both are `not-evaluable` on `Lame`, on `Shadowed`,
  where no declared source is enumerable over the name, and where the declared timeline has aged
  into a `Gap`. `NoData` is an ordinary non-firing evaluation. The first is the only v1 signal
  whose firing population is the **withdrawn** one, and the first pair whose messages must name
  **which of the two sources moved**.
- **Stages 3–4 of [#4](https://github.com/winniel123/verge-asm/issues/4)'s takeover ladder stay
  out** — a CNAME-target provider fingerprint list and a known-error-body matcher are
  `can-i-take-over-xyz`, which is worthless release-coupled. So the shipped signals are
  narrower than the takeover detection their subject matter suggests, and the naming above is
  what stops that from being a misrepresentation.
- **These two are the first signals to alert on *clearing*.** `cname-target-name-error` stops
  firing when the target starts existing — which is the operator re-provisioning **or somebody
  else claiming the orphaned name**; `lame-delegation` clears when a dangling NS domain gets
  registered. On these rules alone, a clear may be the attack having succeeded, so it is
  reported as *this changed* and never as *resolved*. `fired → not-evaluable` remains the
  coverage class per [#32](https://github.com/winniel123/verge-asm/issues/32) and must never be
  worded as a clear either.
- **`sensitive-port-exposed` is the only signal whose reference data we curate**, which makes
  it the one to watch. It passes the test because the list moves at release cadence, but that
  is a statement about cadence, not correctness — hence
  [#21](https://github.com/winniel123/verge-asm/issues/21).
- **Reading `Exposure` was chosen over re-deriving reachability.** Writing the port signal
  over vantage-stamped observations directly would be a second implementation of the flagship
  derivation, and its divergence from the first would surface as false drift — the seam rule.
  The price is that bumping the `Exposure` derivation correctly makes the signal
  non-comparable without anyone touching its rule.
- **Per-rule versioning bounds the blast radius of an edit.** Under one global version, fixing
  a typo in the self-signed rule would silently suppress drift alerting on exposed database
  ports estate-wide for a cycle — a security product going quiet as a side effect of an
  unrelated change. The cost is that *"your rules changed"* is a per-rule statement rather
  than one banner.
- **Notifications have at least two classes.** A clock-crossing signal — a certificate
  expiring — becomes true with no new observation and no rule change. It may be worth telling
  the operator about, but it is **not drift**, and a product whose claim is *what moved since
  last time* must not report the passage of time as movement. Transitions and thresholds are
  different notification classes and [#8](https://github.com/winniel123/verge-asm/issues/8)
  inherits the distinction.
- **Same-derivation-or-no-comparison is now general.** It has appeared three times — batch
  scope in [#5](https://github.com/winniel123/verge-asm/issues/5), source completeness in
  [#7](https://github.com/winniel123/verge-asm/issues/7), derivation version in
  [#10](https://github.com/winniel123/verge-asm/issues/10) — and this is the fourth.
  [#18](https://github.com/winniel123/verge-asm/issues/18) is therefore not a question about
  the exposure board; it is a question about a rule binding every Derived value, and the
  answer should be stated once.
- **`not-evaluable` is likewise the third instance of one idea**, alongside `corroborative`
  silence and `firewalled` vs `internal-only`. Whether the three want a single name is
  recorded as open on the map.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Verifiability test** — a signal is legal if the operator can check the verdict from the cited evidence | Excludes nothing. A fingerprint cites the banner it matched, and the operator can read it |
| **Ship signature matching but version the corpus** | Technically consistent, but a corpus worth having is one updated continuously, and a corpus updated only at release cadence is a corpus nobody maintains. The honest choice is to have one properly or not at all |
| **Severity on signals** | Imports the ranked-backlog model behind `Finding`, adds a second thing to version and dispute, and answers a question the exposure board already answers better. The accepted cost: an operator facing many signals has no intrinsic ranking |
| **One global rule-set version** | Simpler to present, but makes every rule edit suppress comparability for every signal, so an unrelated fix silences the best signal in the product for a cycle |
| **Operator-authored rules in v1** | Puts an un-versioned, un-reviewed derivation inside the comparison path this ADR exists to make trustworthy, and the first thing anyone writes is the signature matching deliberately excluded above. Plausible for v1.1 once `Annotation` and versioning have run in production |
