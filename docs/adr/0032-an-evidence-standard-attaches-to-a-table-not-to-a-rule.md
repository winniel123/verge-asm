# ADR-0032: An evidence standard attaches to a table, not to a rule — and there are three of them

- **Status:** Accepted
- **Date:** 2026-08-14
- **Ticket:** [#33 Does the claim/attestation/determinacy standard generalise to the other nine signals?](https://github.com/winniel123/verge-asm/issues/33)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[#21](https://github.com/winniel123/verge-asm/issues/21) built an evidence standard for one
signal's curated list: a **named claim** from a closed set of three, **attested** by the source
that owns the thing, plus a **determinacy** gate. [#37](https://github.com/winniel123/verge-asm/issues/37)
repaired all four of its defects and derived the claim set by construction rather than by
enumeration. The standard did real work — it inverted the ticket's own source ordering, excluded
every remote-administration port vendor lists lead with, and published its own weakest tier rather
than smoothing it in.

The question this ADR answers is **reach**. *"State the claim, and cite the source that owns it"*
is not port-specific, and the map has carried a fog patch since #31 asking whether #21's standard
and [#31](https://github.com/winniel123/verge-asm/issues/31)'s spec-defined-field test are two
instruments or one. [ADR-0030](./0030-an-offer-is-admitted-on-a-finding-or-on-a-falsity-it-prevents.md)
then ruled an **offer** a third instrument — admitted on a finding or a falsity and never on an
attestation — so the honest shape was already suspected to be three rather than two.

Three things had accumulated in between and none of them was applied to this question.
[ADR-0015](./0015-the-value-space-is-the-commitment.md) and
[ADR-0010](./0010-exposure-composes-two-reaches.md) settled that **a signal is named for the fact
it reads**. [ADR-0024](./0024-a-rules-domain-is-the-extension-of-its-name.md) made a rule **four
parts** and fixed its domain as the extension of its name. And
[#60](https://github.com/winniel123/verge-asm/issues/60) found a **second** signal whose reference
data we curate, falsifying [ADR-0004](./0004-signals-are-release-coupled-rules.md)'s claim that
`sensitive-port-reached-from-internet` was the only one — and recorded that second one's rationale
as *stated but unverified against a source*.

Together those mean the question as the ticket asks it — *does the standard generalise to the other
nine signals?* — has a false presupposition in it, and finding that is most of the answer.

## Decision

| Concern | Decision |
| --- | --- |
| What the standard governs | **A curated table, never a rule.** A rule carrying no table has nothing for it to govern, and *does the standard apply to `certificate-expired`?* is malformed, not hard |
| Does it generalise to the other fifteen v1 rules | **It reaches two of them, and thirteen have no table at all** — walked below, not asserted |
| Gate 1, the closed set of three claims | **Does not generalise, and cannot.** Its closure is derived from what an internet vantage supplies, and exactly one v1 rule reads `Exposure`. A second table needs **its own** closed set, derived the same way from what its rule reads |
| Gate 2, attestation by the owner | **Generalises fully.** It binds every project-authored table that asserts something about the **world** |
| Gate 3, determinacy | **The surrogate gate.** It binds any table whose key is not the fact the rule names. v1 has exactly one surrogate, so elsewhere the question does not arise — this is *outside the domain*, not a pass |
| Where the gate does **not** apply | A table about **our own measurement** (`k`, the availability window, the prober's retry budget, the frequency half of `verge-core`, the coverage threshold). There is no owner to attest a fact about us |
| Two instruments or three | **Three, and they partition on what the table does** — what we **ask** (ADR-0030), what we **conclude** (#31), what we **assert** (#21). One rule may compose tables under two of them |
| `certificate-expiring`'s `N = 30 days` | **Inside gate 2 and currently unattested.** It ships, disclosed, and the retrieval is [#67](https://github.com/winniel123/verge-asm/issues/67). A row may not move on a re-reading of text already held |
| `certificate-weak-key-or-signature`'s thresholds | **A third curated table, and it has never been written.** It must be authored under gate 2, which is [#68](https://github.com/winniel123/verge-asm/issues/68) — and it **blocks [#12](https://github.com/winniel123/verge-asm/issues/12)** |
| The weak-tier disclosure | **Generalises — to every instrument's own document, and to no screen in v1.** Its consumer is the curator, not the operator |
| An attestation moving under a shipped rule | **Already fully specified**, and needs nothing new: it is an output-affecting change at release cadence. What is new is the failure *shape* — §10.4's one-way rule makes de-attestation **silent** |
| Where this lives | **A new ADR.** ADR-0004 governs *may this ship as a rule*; this governs *what licenses the content of a table*, whose population includes tables no rule reads |

## Amendment — [#67](https://github.com/winniel123/verge-asm/issues/67): `N` is attested, and the claim recorded against it was never derived

Three rows above say `certificate-expiring`'s `N` is **inside gate 2 and currently unattested** —
the Decision table, §5's gate-2 table, and the v1 walk. They stand unrewritten per the
name-and-withdraw convention. **`N` is attested**, and the route to the attestation is a defect in
this ADR rather than a discovery about the world.

**§3 obliged a second table to derive its own closed claim set from what its rule reads, and this
ADR did not perform that derivation for `N`.** It filled the slot with #60's prose gloss — *"the
world — this is the last point the operator can still act"* — and §5's table records that as the
claim. That sentence is about the **operator's remediation capacity**, which is not an artefact
anyone designs and which `certificate-expiring` reads nothing about, so no owner for it could exist
at any depth of search.

Performed, the derivation closes at one member: the rule reads `not_before`, `not_after` and the
clock, a horizon is a **cut in an interval**, and a cut in an interval can assert only a **position
within it**. The claim is *the certificate is inside the portion of its own validity period in which
its issuer says replacement is due* — and its owner is the **issuing CA**, the party that fixed the
interval. §10.5 keys on the artefact, and the interval is the artefact.

`N` therefore **moves**, from 30 days to **one third of the certificate's validity period, and one
half of it below a 10-day validity**, attested by the IETF on form (RFC 9773 §1) and by the issuer
on value. [ADR-0034](./0034-derive-the-claim-before-looking-for-the-owner.md) holds the ruling;
[`docs/research/acme-renewal-timing.md`](../research/acme-renewal-timing.md) holds the retrieval.

**Two things above are amended as rules rather than as rows.**

**The disclosure obligation gains a second limb.** *A disclosed weakness names the retrieval or
measurement that would resolve it* has nothing to say once that retrieval has been run, and #67 is
the first time one has been. Where the named retrieval has been **performed**, the disclosure names
what the retrieval established, what remains unestablished, and the condition that would move the
row. **A performed retrieval may not leave a row pointing at itself.**

**§8's two piles are not exhaustive.** A weak row was either **watched** (a footing a release can
silently remove) or **chased** (a footing a retrieval could establish). `N` was filed as chased and
is now neither: a row that reads the moving quantity from the subject at evaluation time has nothing
to watch and nothing to chase. That third state is the one to aim for, and ADR-0034 §4 gives the
rule that reaches it — **where a constant is the product of a fraction and a moving world quantity,
ship the fraction.**

**[#68](https://github.com/winniel123/verge-asm/issues/68) inherits the order of operations and is
still not pre-empted.** This ADR's Consequences flag *who owns a claim about the WebPKI's key and
algorithm floor* as live for #68. That stands. What #67 adds is that #68 must **derive
`certificate-weak-key-or-signature`'s claim set from what the rule reads before hunting for the
owner** — because an underived claim and a genuine absence are indistinguishable from inside a
retrieval, and this ADR has now produced one of each.

## Rationale

### 1. The unit is the table, and the ticket's question presupposes otherwise

#21's standard was written about a list and has only ever been applied to a list. Nothing in it
takes a rule as an argument. Read the three gates back: *a named claim* is a property of a row, *an
attestation* is a property of a row's footing, *determinacy* is a property of a row's key. There is
no third thing a rule has that a table does not.

So asking whether the standard generalises to `certificate-expired` is the same shape as
[#39](https://github.com/winniel123/verge-asm/issues/39)'s *what is SWIP's `authority`?* — a
question that presupposes an object, asked of something that has none. `certificate-expired`
compares `not_after` against the clock. There is no row, no claim and no owner, and the honest
answer is not *it passes trivially* but **there is nothing there**.

That distinction is not pedantry, because the two answers behave differently under pressure. *It
passes trivially* invites a session to write a claim class for it, which is how a gate becomes a
ritual and how #37's *unfalsifiability arriving slowly* gets in. *There is nothing there* closes the
question permanently and points the next session at the thing that actually needs governing.

### 2. Naming a rule for the fact it reads is what shrank the standard's reach, tick by tick

The ticket's own worked example is the clearest case and it points the other way from the ticket's
expectation. It says of *TLS 1.0/1.1*: **"the owning source is plainly the IETF's own deprecation."**

That is true of the *reason the rule is worth having* and false of the *rule*.
[#36](https://github.com/winniel123/verge-asm/issues/36) re-founded the rule as **`tls-1.0-accepted`**,
which asserts that a version identifier defined in RFC 8446's registry was accepted on the wire. It
does not assert that TLS 1.0 is weak. RFC 8996 is therefore not an attestation the rule needs — it
is drill-down prose, in exactly the position ADR-0004 put *takeover* for the DNS rules: a conclusion
the evidence suggests, hedged where it can be hedged, and outside the rule.

The same move accounts for two more of the sixteen. **`certificate-self-signed`** needs no trust
store, and a trust store would be a curated table about the world of precisely the kind this ADR
governs — the rule is named `self-signed` and not `untrusted`, and that naming is why v1 ships no
root bundle. **`plaintext-http-no-https`** lost its `80/tcp` literal at ADR-0024, and the literal
was a curated port table arriving one rule at a time.

So the standard's reach did not shrink because this ticket ruled it narrow. It shrank because
ADR-0010, ADR-0015, ADR-0024 and #36 kept renaming rules onto the facts they read, and each rename
deleted a table. **The naming discipline is the evidence standard's first line, and it discharges
thirteen of sixteen rules before this ADR is consulted.**

### 3. Gate 1 is a theorem about the only rule that reads `Exposure`

#37 closed the claim set by construction:

> A permitted claim must name a **mismatch between an assumption the protocol makes and something an
> internet vantage supplies** — an unknown principal, an untrusted path, a caller outside the
> boundary. Three properties, three claims.

The derivation runs entirely through `Exposure`'s definition. ADR-0010 and
[#32](https://github.com/winniel123/verge-asm/issues/32) established that
`sensitive-port-reached-from-internet` is **the only v1 signal that reads `Exposure`**. So the
closed set of three has exactly one rule inside its scope, and that is a theorem from where the
closure was derived rather than an accident of which tables exist — the same shape as
[#63](https://github.com/winniel123/verge-asm/issues/63)'s kind restriction being a theorem from
who supplies the key.

Test it against the one other table that asserts about the world. `certificate-expiring`'s `N`
names a mismatch between a certificate's stated validity window and the lead time an operator needs
to act. That is not a property of a vantage at all, and none of the three fits. Widening the three
until it does is the failure #37 refused: an open set makes the standard's tightness contingent on
the next odd-shaped case.

**So a second curated table does not inherit #21's three claims. It owes its own closed set, derived
the same way — from what the rule reads — and the derivation is part of the table's cost.** That is
the load-bearing generalisation, and it is a generalisation of the *method*, not of the *set*.

**Thin ground, inherited and marked.** #37 flagged its own closure as resting on the claim that an
internet vantage supplies exactly three things, read off `Exposure`'s definition rather than
measured. This section inherits that thin ground whole and adds no measurement to it.

### 4. Gate 3 is the surrogate gate, and v1 has exactly one surrogate

Determinacy reads, in the general, as *the input implies one interpretation*. Read that way it looks
like it should bind everywhere and pass everywhere, which is what a ritual looks like.

Read at #21 §2.4 it is narrower and sharper. The gate exists because
`sensitive-port-reached-from-internet` **cannot read the fact it is named for**. §2.4 states it
outright: the signal *"claims a port associated with a sensitive service is reachable from an
internet vantage — it does not claim the service is running."* The `(port, transport)` key is a
**surrogate** for the service, the registry disclaims the inference in capitals, and twelve of the
best-known rows are squatted rather than registered. Determinacy is the gate that keeps the
surrogate honest.

Every other v1 rule reads the fact it is named for, directly, and it does so **because ADR-0024 made
the domain the extension of the name**. `certificate-expired` reads `not_after`, which *is* the
expiry. `unauthenticated-request-answered` reads a status class, which *is* the answer. There is no
proxy anywhere else in the set.

So gate 3 does not fail to generalise. Applied to the other fifteen it is ADR-0024's own third
register — **outside the domain**: the question does not arise, and it is rendered nowhere. The
instrument gets the same three-way treatment the model gives a signal, which is the check that the
answer is the right shape.

It stays a live gate, and the statement is falsifiable rather than decorative: **any future rule
whose predicate keys on a surrogate owes a determinacy argument.** The nearest live candidate is the
map's private-range-address rule, which keys on the IANA Special-Purpose Address Registry — a
spec-defined closed set naming the addresses themselves, not a proxy for them. It clears, and it
clears for a reason that can be checked.

### 5. Gate 2 generalises, and the line is *about the world* versus *about our own measurement*

The discriminator was already on the books and needed only to be pointed at this.
#60 wrote of `N`: *"Unlike `k`, it makes a claim about the **world** rather than about our own
measurement, so the reason is stated rather than buried."*

> **A project-authored table is inside gate 2 exactly where it asserts something about the world.
> Where it is about our own measurement, no owner exists, and the gate does not apply — it is
> inapplicable, not passed.**

| Table | About | Gate 2 |
| --- | --- | --- |
| The 38 sensitive `(port, transport)` pairs | The world — *this exposure is never correct* | **Applies.** Attested for 37 rows; `161/udp` disclosed, [#66](https://github.com/winniel123/verge-asm/issues/66) |
| `certificate-expiring`'s `N = 30 days` | The world — *this is the last point the operator can still act* | **Applies. Currently unattested**, [#67](https://github.com/winniel123/verge-asm/issues/67) |
| `certificate-weak-key-or-signature`'s thresholds | The world — *this key or algorithm is weak* | **Applies. Never written**, [#68](https://github.com/winniel123/verge-asm/issues/68) |
| `k` cadences; the availability window; the coverage threshold | Our own measurement | Inapplicable. ADR-0008 governs alone |
| The prober's timeout and retry budget | Our own measurement | Inapplicable. ADR-0021's leaves govern |
| The frequency half of `verge-core` | Where we look | Inapplicable — aperture, and #31's line governs |
| `Custody`, and the estate `redirect-to-host-outside-estate` reads | The operator's own declaration | Inapplicable. Declared input, ADR-0013 |

The last row answers the ticket's third worked example directly. *A redirect to a host outside the
estate* depends on `Custody`, which is Derived — but Derived **from the operator's own `Seed`s**, so
there is no claim of ours in it and nothing to attest. Its version vector composes `Custody`'s leaf,
which is the whole of the machinery it needs.

### 6. Three instruments, and they partition on what the table does

The map suspected #21 and #31 were complementary. They are, and the cut is cleaner than
complementary: the three instruments partition the places a project-authored table can sit relative
to the wire.

| Where the table sits | Instrument | What it governs | Its failure mode |
| --- | --- | --- | --- |
| **Before** the wire — what we ask | ADR-0030's two limbs | An **offer**: a candidate is admitted where its acceptance is a finding, or where its absence would make the measurement false | A blind spot, or a cost. Never a false verdict |
| **At** the wire — what a byte means | [#31](https://github.com/winniel123/verge-asm/issues/31)'s spec-defined-field test | A **conclusion**: a table deciding *where to look* is aperture; a table deciding *what an answer means* is a signature database | A false verdict that moves when a vendor's banner moves |
| **After** the wire — what a value means normatively | [#21](https://github.com/winniel123/verge-asm/issues/21)'s standard, as repaired by [#37](https://github.com/winniel123/verge-asm/issues/37) | An **assertion**: a named claim, attested by the owner, over a determinate key | A laundered opinion — *commonly attacked* wearing *never correct*'s clothes |

They do not compete, because they answer different questions about different objects, and a table is
under exactly one of them. The proof that the partition is on **artefacts** rather than on **rules**
is that a single rule composes tables under two instruments at once:

- `sensitive-port-reached-from-internet` reads the **sensitive list** (#21) and, through
  `verge-core`, an **aperture** set (#31). Two instruments, one rule.
- `tls-1.0-accepted` carries **no assertion at all**, and its domain — *the `Service` completed at
  least one handshake in the batch's candidate set* — is an **offer** (ADR-0030). A rule with
  nothing under #21 still has something under ADR-0030.

**And the MongoDB case, which is what put this on the map, resolves without either instrument
bending.** #31 excluded MongoDB *"on #21's grounds"* while admitting everything else on its own
test. Under this partition that was exactly right and never an overlap: the *dispatch* to MongoDB's
wire protocol is #31's, but *"MongoDB permits anonymous access as shipped"* is an **assertion about
the world**, so gate 2 governs it, and #31 was correct to record it as a non-finding rather than
assert it from common knowledge. The two instruments were operating on two different objects inside
one paragraph, which is why it read as a collision.

### 7. The weak-tier disclosure generalises to every instrument's document, and to no screen in v1

#21 published which rows rest on the weakest evidence tier rather than hiding the unevenness. That
practice is right and it has **already** generalised once without anyone ruling it: ADR-0030 §5
cites *"#21's publish the weak tier with a named route out of the weak tier rather than a permanent
caveat"* and ships `docs/spec/measurement-offers.md` with a whole column marked unmeasured. So the
disclosure obligation is confirmed as binding on **all three instruments**, not on #21's alone, and
its stronger form is confirmed with it: **a disclosed weakness names the retrieval or measurement
that would resolve it**, or it is a permanent caveat and decays into decoration.

What the ticket actually asks is whether it must reach the **interface**. It must not, in v1, and
four things say so.

**It is severity.** ADR-0004 refused severity because it ranks a static backlog, which is the
`Finding` model [#7](https://github.com/winniel123/verge-asm/issues/7) rejected, and `CONTEXT.md`
lists *severity* under `Signal`'s avoided words. A per-row evidence tier is a confidence ranking the
operator will triage on. It is severity re-entering through the one door the model left open, and it
would arrive labelled as honesty.

**It does not vary per subject.** The tier is a property of the row in **our** table, so it is
identical on every firing of that row. Rendering a constant in a per-subject column is
[#28](https://github.com/winniel123/verge-asm/issues/28)'s two-numbers hazard with the second number
carrying no information at all.

**It names no act the operator can take.** ADR-0009 settled that the operator edits the frequency
half of `verge-core` alone, because *a port the operator can hide is a signal they can silence*. A
disclosure that invites the operator to disagree with a row offers a door the model deliberately
locked, and the only surface where they could act on it is the one
[#51](https://github.com/winniel123/verge-asm/issues/51) barred from acquiring per-row controls.

**It has a real consumer, and that consumer is not the operator.** See §8.

**Where it would go, if it ever goes anywhere.** Not a badge — [#44](https://github.com/winniel123/verge-asm/issues/44)'s
**standing aperture statement on `Coverage`**, which is explicitly *counts of our own rules and
lists, closed and enumerable, never a figure over the estate*. The weak tier is exactly that shape
and would need no new surface. **The condition that moves it is named**: if the operator ever
acquires a sanctioned way to disagree with a row — the map's open `Annotation` question — the
disclosure becomes that act's input and earns that home. Until then it stays in the instrument's own
document.

### 8. An attestation moving is already specified — and §10.4 made de-attestation silent

The ticket asks what happens when an attestation changes underneath a shipped rule, and carries #21's
measurement that primary sources move much faster than release-coupling assumes: BOD 22-01 revoked,
CISA's CPG renumbered, CIS abandoning named ports, Docker deprecating unauthenticated TCP, Prometheus
**softening** after gaining native TLS.

**The mechanism needs nothing new and must not be given anything new.** An attestation moving is not
an observation, so nothing happens until somebody ships a release. A revision is an output-affecting
change under [ADR-0008](./0008-derivation-versions-move-on-content.md): the rule's leaf moves, the
rule `Break`s uniformly, one cadence and one rule, and #21 §7.2 already priced it. ADR-0004's
release-cadence test is — per [#18](https://github.com/winniel123/verge-asm/issues/18)'s correction —
a **ceiling on how often reference data may change**, which is precisely the licence to let a moved
attestation sit until the next release. The unanswered half is *who is watching*, and that is the
map's existing curation patch, not a new object.

**What is new is a failure shape that #37 created and did not name.** §10.4 ruled that a shipped
default attests **only where it restricts**: a permissive default is the absence of an act, and
neither admits nor excludes. Follow it forward. A row admitted on a *restricting* default loses its
footing the moment that default becomes permissive — and **nobody says anything**. No source
publishes a retraction; a config default flips in a major release. Under the symmetric reading #37
rejected, the flip would at least have been an exclusion with a direction; under the one-way rule it
is the quiet removal of an admission route.

> **Silent de-attestation is the one way a row can lose its grounds with no document changing
> anywhere.** The rows exposed to it are exactly the rows resting on a restricting default alone —
> **5432/tcp, 5984/tcp and 9042/tcp** — which are exactly the three rows #21 disclosed as its weak
> tier.

So **the weak tier is the curator's watch list**, and the disclosure #21 made for honesty acquires an
operational job it was not written for. That is the disclosure's consumer, and it is the second and
independent reason §7 keeps it in the document rather than on a screen. It also gives the map's
curation patch a concrete first task instead of a posture.

`161/udp` is disclosed too and is **not** on this watch list: its weakness is a corroborator standing
where an owner should, which a release cannot change. That is a retrieval (#66), and the two kinds of
weak row must not be collapsed — one is watched, the other is chased.

**Marked as derived, not measured.** This section is read off §10.4's rule plus #21's measured
finding that sources move. Nobody has checked whether any of the three defaults has already moved;
that check belongs to the curation patch, which is where the watch lives.

## The v1 walk — all sixteen rules, walked rather than asserted

The claim that thirteen rules carry no curated table is checkable, so it is checked.

| Rule | Curated table it reads | Instrument |
| --- | --- | --- |
| `certificate-expired` | none — `not_after` vs the clock, RFC 5280 field | — |
| `certificate-not-yet-valid` | none — `not_before` vs the clock | — |
| `certificate-expiring` | **`N = 30 days`** | **#21 gate 2 — unattested, [#67](https://github.com/winniel123/verge-asm/issues/67)** |
| `certificate-self-signed` | none — issuer ≡ subject, self-signature verifies. **No trust store**, which is why the rule is not named `untrusted` | — |
| `certificate-weak-key-or-signature` | **thresholds and a deprecated-algorithm set — never written** | **#21 gate 2 — [#68](https://github.com/winniel123/verge-asm/issues/68)** |
| `certificate-hostname-san-mismatch` | none — RFC 6125 matching, leftmost-label wildcards. **Reaching for a public suffix list would create one** | — |
| `tls-1.0-accepted` | none asserted; its domain is the **TLS candidate set** | ADR-0030 |
| `plaintext-http-no-https` | none — the `80/tcp` literal was withdrawn at ADR-0024 | — |
| `redirect-does-not-upgrade-to-tls` | none — the `Location` scheme, RFC 9110 | — |
| `redirect-to-host-outside-estate` | none — `Custody`, Derived from the operator's `Seed`s | — |
| `unauthenticated-request-answered` | none — status classes, RFC 9110 | — |
| `sensitive-port-reached-from-internet` | **the 38 pairs**; and, through `verge-core`, the **frequency set** | **#21, all three gates**; #31 for the aperture half |
| `lame-delegation` | none — ADR-0004: *no reference data at all* | — |
| `cname-target-name-error` | none — as above | — |
| `zone-declared-name-returns-name-error` | none — ADR-0004: *zero rows of reference data* | — |
| `resolved-name-absent-from-zone` | none — as above; the zone file is Declared input | — |

**Thirteen of sixteen carry nothing this ADR governs.** Two carry a table inside gate 2 and one of
those two has no content yet. One carries a table inside all three gates, and it is the one #21 was
written for.

## Consequences

- **[ADR-0004](./0004-signals-are-release-coupled-rules.md) is amended once**, in the place #60
  already corrected: its Consequences call `sensitive-port-reached-from-internet` *the only signal
  whose reference data we curate*; #60 made it two; it is **three**, the third being
  `certificate-weak-key-or-signature`'s unwritten table.
- **A curated table's cost gains an item.** Under ADR-0024 a rule is four parts plus one cost. Where
  a rule carries a table that asserts about the world, the table owes a **closed claim set derived
  from what the rule reads**, an **owner attestation per row**, and a **determinacy argument where
  its key is a surrogate**. This is not a fifth part of a rule; it is the table's own accounting.
- **Gate 2 is not a shipping gate.** A table with an unattested row **ships, disclosed**, exactly as
  `161/udp` and `5432/tcp` do. #37's precedent binds: a row may not move on a re-reading of text
  already held, only on a **retrieval**. So #67 and #68 are retrievals, not blockers on the number
  they concern.
- **[#68](https://github.com/winniel123/verge-asm/issues/68) blocks
  [#12](https://github.com/winniel123/verge-asm/issues/12).** A v1 rule with no predicate content
  cannot be assembled into a spec, and #12 must carry sixteen rules per
  [#64](https://github.com/winniel123/verge-asm/issues/64). #67 does not block it — `N` is fixed at
  30 and the spec can carry it with the disclosure.
- **[#68](https://github.com/winniel123/verge-asm/issues/68) will test §10.5's owner definition on a
  case it was not built for.** *Owner* keys on the artefact — the party that designed the protocol
  or authors the reference implementation. A claim about the WebPKI's key and algorithm floor is
  owned by a body that designed neither X.509 nor any implementation of it. That is a live question
  for #68 and is deliberately not pre-empted here.
- **The disclosure obligation binds all three instruments**, in each instrument's own document, and
  in its stronger form: a disclosed weakness names the retrieval or measurement that resolves it.
- **No screen changes.** The weak tier reaches no surface in v1, and the condition that would move it
  onto `Coverage`'s standing aperture statement is named rather than left to judgement.
- **`CONTEXT.md` is not edited**, deliberately — concurrent sessions are in that file, and #37 set
  the precedent. The edit this ADR would make is one clause on `Signal`: *where a rule reads a
  curated table asserting about the world, the table is governed by ADR-0032 and the rule's naming
  discipline is what usually means there is no such table.*
- **The map's curation patch gains its first concrete task** — the three shipped-default rows are the
  watch list, and nobody has checked whether any of the three defaults has already moved.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Adopt the standard wholesale for all sixteen rules** | Two of its three gates have no object elsewhere, so adopting it whole means inventing a claim class for *a certificate is expiring* — #37's *unfalsifiability arriving slowly* — and running a determinacy gate that every candidate passes forever. A gate nothing fails is a ritual, and ADR-0004's #35 amendment already caught this exact confusion once: *a bar erected to protect the list rather than the property* |
| **Rule it a one-off and leave it with the port list** | Leaves two curated tables asserting about the world with no evidence standard at all, one of which #60 already recorded as unverified. It would also be the third time this map declined to generalise something already general — #31 called its own line *reusable well beyond this ticket*, and #37 found *a row moves on retrieval, never on a re-reading* to be a general precedent |
| **Fold this into ADR-0004 as an amendment** | Wrong population. ADR-0004's subject is **rules** and its test is *may this ship*; this ADR's subject is **tables**, including tables no rule reads — the TLS offer and the dispatch table. Conflating *may this ship* with *is this content licensed* is what the ticket warned makes both harder to apply, and ADR-0004 has already been misread as a second gate twice |
| **Two instruments, with #21 and #31 as complements** | True but under-specified, and it strands ADR-0030, which had already ruled itself a third and was reachable from neither. The three-way partition on *ask / conclude / assert* is exhaustive over where a project-authored table sits relative to the wire, and it explains the MongoDB case without either instrument bending |
| **Partition the instruments by rule rather than by artefact** | Falsified by two rules: `sensitive-port-reached-from-internet` composes tables under #21 **and** #31, and `tls-1.0-accepted` carries nothing under #21 while carrying an offer under ADR-0030. A rule is not the unit |
| **Render the weak tier on `Signals`** | Severity by another name, on a product that refused severity; a constant rendered in a per-subject column; and an invitation to disagree with a row through a door ADR-0009 and #51 both locked |
| **Extend #21's three claims to cover a second table** | The three are derived from what an **internet vantage** supplies, and only one v1 rule reads `Exposure`. Stretching them is exactly the open-set failure #37 closed — the next odd-shaped case renegotiates the admission rules |
| **Hold `N = 30` or the weak-key table out of the release until attested** | Gate 2 has never been a shipping gate — `161/udp` and `5432/tcp` ship disclosed. Making it one now would retroactively delete rows on grounds #37 refused, and would move a row on a re-reading rather than on a retrieval |
| **Build a watch mechanism for moving attestations here** | ADR-0008 already specifies what a revision costs and ADR-0004's cadence test already licenses the staleness. What is missing is a **human watch**, which is the map's curation patch and is not an object this ADR can mint |
