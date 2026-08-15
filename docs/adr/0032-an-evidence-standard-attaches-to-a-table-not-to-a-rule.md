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
| Does it generalise to the other fifteen v1 rules | ~~**It reaches two of them, and thirteen have no table at all**~~ — walked below, not asserted. **THREE of seventeen since [#128](https://github.com/winniel123/verge-asm/issues/128)**, which admitted `non-globally-reachable-address-resolved-from-internet` and its transcription of the IANA Special-Purpose Address Registry ([ADR-0071](./0071-a-vantage-scoped-claim-is-read-only-at-the-vantage-that-scopes-it.md)); thirteen still have no table at all |
| Gate 1, the closed set of three claims | **Does not generalise, and cannot.** Its closure is derived from what an internet vantage supplies, and exactly one v1 rule reads `Exposure`. A second table needs **its own** closed set, derived the same way from what its rule reads |
| Gate 2, attestation by the owner | **Generalises fully.** It binds every project-authored table that asserts something about the **world** |
| Gate 3, determinacy | **The surrogate gate.** It binds any table whose key is not the fact the rule names. v1 has exactly one surrogate, so elsewhere the question does not arise — this is *outside the domain*, not a pass |
| Where the gate does **not** apply | A table about **our own measurement** (`k`, the availability window, the prober's retry budget, the frequency half of `verge-core`, the coverage threshold). There is no owner to attest a fact about us |
| Two instruments or three | **Three, and they partition on what the table does** — what we **ask** (ADR-0030), what we **conclude** (#31), what we **assert** (#21). One rule may compose tables under two of them |
| `certificate-expiring`'s `N = 30 days` | ~~**Inside gate 2 and currently unattested.** It ships, disclosed, and the retrieval is [#67](https://github.com/winniel123/verge-asm/issues/67).~~ **`N` IS ATTESTED** — the [#67](https://github.com/winniel123/verge-asm/issues/67) amendment below, which also finds the claim recorded against it was never derived. A row may not move on a re-reading of text already held |
| `certificate-weak-key-or-signature`'s thresholds | ~~**A third curated table, and it has never been written.** It must be authored under gate 2, which is [#68](https://github.com/winniel123/verge-asm/issues/68) — and it **blocks [#12](https://github.com/winniel123/verge-asm/issues/12)**~~ — **the table HAS been written** ([`weak-key-and-signature.md`](../research/weak-key-and-signature.md), five rows) and **#68 no longer blocks [#12](https://github.com/winniel123/verge-asm/issues/12)**, per the #68 amendment below |
| The weak-tier disclosure | **Generalises — to every instrument's own document, and to no screen in v1.** Its consumer is the curator, not the operator |
| An attestation moving under a shipped rule | **Already fully specified**, and needs nothing new: it is an output-affecting change at release cadence. What is new is the failure *shape* — §10.4's one-way rule makes de-attestation **silent** |
| Where this lives | **A new ADR.** ADR-0004 governs *may this ship as a rule*; this governs *what licenses the content of a table*, whose population includes tables no rule reads |

> **Amended by [ADR-0036](./0036-a-shipped-default-is-the-configuration-that-takes-effect.md)**
> ([#69](https://github.com/winniel123/verge-asm/issues/69)). Gate 2 is the gate that generalises, so
> its refinements generalise with it. A **shipped default** means the configuration that **takes
> effect** and that the party **documents as its default** — an example file is silent in both
> directions — and installing another party's bytes transfers **operativeness, not ownership**, so a
> distributor's packaging corroborates and is never sole grounds where the claim is about something it
> did not design. Any future table admitted under gate 2 inherits all four limbs. No row moved and no
> rule version moved; one footing did.

## Amendment — [#67](https://github.com/winniel123/verge-asm/issues/67): `N` is attested, and the claim recorded against it was never derived

Three rows above say `certificate-expiring`'s `N` is **inside gate 2 and currently unattested** —
the Decision table, §5's gate-2 table, and the v1 walk. ~~They stand unrewritten per the
name-and-withdraw convention.~~ **`N` is attested**, and the route to the attestation is a defect in
this ADR rather than a discovery about the world.

> **That reading of the convention is WITHDRAWN by [#106](https://github.com/winniel123/verge-asm/issues/106).**
> Name-and-withdraw is *left standing **and marked***, never *left standing unmarked* — the
> [`sensitive-ports.md`](../research/sensitive-ports.md) §3 sentence it derives from reads *"left
> standing … **marked here** rather than deleted"*. Under
> [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) as widened
> by #106, an amendment does not discharge a clause elsewhere in its own file, so all three rows —
> and the three the #68 amendment below leaves the same way — are now marked at the sentence.

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

> **Amendment — [#68](https://github.com/winniel123/verge-asm/issues/68).** Two rows above are now
> stale in one direction and are left standing per the name-and-withdraw convention.
> `certificate-weak-key-or-signature`'s table **has been written** —
> [`docs/research/weak-key-and-signature.md`](../research/weak-key-and-signature.md), five rows —
> so it is no longer *"never written"*, and **#68 no longer blocks
> [#12](https://github.com/winniel123/verge-asm/issues/12)**. The Consequences' live question about
> §10.5's owner definition is **answered**: the definition survives and gains one clause naming a
> **cryptographic primitive** as an artefact class, recorded in
> [ADR-0035](./0035-a-cryptographic-primitives-owner-is-its-specifier.md). Three findings above are
> confirmed rather than changed — gate 1's *derive your own set* survived its second application and
> produced **two** claims, not three; gate 3 read **outside the domain** as predicted; and the weak
> tier is disclosed in the instrument's own document. ADR-0035 adds one thing this ADR did not
> anticipate: a **third kind of weak row** — a *scope* weakness, which is neither watched nor chased.

> **Amendment — [#73](https://github.com/winniel123/verge-asm/issues/73): the disclosure obligation
> gains a third limb.** §7 requires a disclosed weakness to **name the retrieval** that resolves it;
> #67's amendment above added that where the retrieval has been **performed**, the disclosure says what
> it established. #73 is the first retrieval to come back **partly empty** — positive for two rows,
> empty for a third — and that case needs its own sentence. **Where a retrieval has run and a residue
> survives, the disclosure carries the corpus actually searched, enumerated; what was found and which
> rows it reached; and the smallest extension of the corpus that could still change the answer.** The
> residue is then **bounded**, not permanent: *permanent* is unfalsifiable and is what §7 forbids,
> while *bounded* is falsified by naming one document outside the boundary. The general finding behind
> it — **a specification's silence is not the owner's silence**, so a negative must enumerate an
> owner's **document classes** and not only its documents — is
> [ADR-0040](./0040-a-specifications-silence-is-not-the-owners-silence.md). No gate changes and no rule
> version moves; the weak tier of `certificate-weak-key-or-signature`'s table goes from three rows to
> one limb of one row.

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
thirteen of ~~sixteen~~ *seventeen* rules before this ADR is consulted** — the denominator moved at
[#128](https://github.com/winniel123/verge-asm/issues/128) and the thirteen did not.

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
whose predicate keys on a surrogate owes a determinacy argument.** ~~The nearest live candidate is the
map's private-range-address rule, which keys on the IANA Special-Purpose Address Registry — a
spec-defined closed set naming the addresses themselves, not a proxy for them. It clears, and it
clears for a reason that can be checked.~~

> **DISCHARGED by [#128](https://github.com/winniel123/verge-asm/issues/128)** — and marked rather
> than left standing, because the #67 amendment above rules that **a performed retrieval may not leave
> a row pointing at itself**. The candidate shipped as
> **`non-globally-reachable-address-resolved-from-internet`**, and the check named here was **run**
> rather than cited: the key is the address and the fact is a property of the address, so the registry
> does not stand in for the classification, it **is** the classification. Gate 3 reads **outside the
> domain** — ADR-0024's third register — exactly as predicted. The forward statement is spent;
> [ADR-0071](./0071-a-vantage-scoped-claim-is-read-only-at-the-vantage-that-scopes-it.md) §5 holds the
> worked gate-by-gate discharge and
> [`special-purpose-address-registry.md`](../research/special-purpose-address-registry.md) holds the
> table. The *live gate* sentence above it is untouched and still binds on the next candidate.

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
| `certificate-expiring`'s `N = 30 days` | The world — *this is the last point the operator can still act* | ~~**Applies. Currently unattested**~~ — **ATTESTED**, and this claim gloss was **never derived**, per the [#67](https://github.com/winniel123/verge-asm/issues/67) amendment above |
| `certificate-weak-key-or-signature`'s thresholds | The world — *this key or algorithm is weak* | **Applies. Never written**, [#68](https://github.com/winniel123/verge-asm/issues/68) |
| `k` cadences; the availability window; the coverage threshold | Our own measurement | Inapplicable. ADR-0008 governs alone |
| The prober's timeout and retry budget | Our own measurement | Inapplicable. ADR-0021's leaves govern |
| The frequency half of `verge-core` | Where we look | Inapplicable — aperture, and #31's line governs |
| `Custody`, and the estate `redirect-to-host-outside-estate` reads | The operator's own declaration | Inapplicable. Declared input, ADR-0013 |
| The IANA Special-Purpose Address Registry, transcribed | The world — *the allocating authority designated this block for a purpose other than global reachability* | **Applies and passes by construction**, [#128](https://github.com/winniel123/verge-asm/issues/128). The table is a **transcription** whose selection predicate is the artefact's own `Globally Reachable` column, so one retrieval attests all fifty rows (ADR-0037). *Applies* is load-bearing: the rows are not ours but the **selection** always is, and a hand-picked *RFC 1918 / 6598 / 4193* would have been an authored table with no owner — [ADR-0071](./0071-a-vantage-scoped-claim-is-read-only-at-the-vantage-that-scopes-it.md) |

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

> **The `Annotation` question is closed and this conditional does NOT fire** —
> [#117](https://github.com/winniel123/verge-asm/issues/117),
> [ADR-0016](./0016-an-annotation-moves-a-message-never-a-number.md), 2026-08-15. An `Annotation` is
> an act on the **message** and never on the **verdict**: the rule fires, the census counts it, the
> table is untouched, and no evidence is weighed. What §7 was waiting for is an act that changes what
> a rule **concludes**, and v1 ships none — ADR-0016 bars the domain route, the subject route and the
> span route by name. So the weak tier stays in the instrument's own document, and the three
> objections above are undisturbed: the second one in particular still bites, since a tier identical
> on every firing of a row would be a constant sitting beside a per-subject decision. **Do not read
> the arrival of `Annotation` as the trigger.** The condition stands as written and is still unmet.

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

So ~~**the weak tier is the curator's watch list**~~, and the disclosure #21 made for honesty acquires an
operational job it was not written for.

> **That equation is WITHDRAWN — the watch list is defined by *shape*, not by the weak tier**, per
> [ADR-0038](./0038-a-constant-is-a-product-only-where-the-quantity-is-readable.md)
> ([#71](https://github.com/winniel123/verge-asm/issues/71)), and written here by
> [#102](https://github.com/winniel123/verge-asm/issues/102) under
> [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) because
> ADR-0038 recorded the redefinition only in its own Consequences and this ADR never cited it.
> ADR-0038's words: *"§8's watch list is **redefined by shape**, gaining `verge-core`'s frequency half
> and the weak-key table without gaining a pile."*
>
> **No enumerated row moves on this.** ADR-0038 says its *"three shipped-default rows — now two after
> [#69](https://github.com/winniel123/verge-asm/issues/69) — are unaffected"*, and the row
> enumeration below is governed by the #76 / #88 / #95 amendments, not by this box. What is withdrawn
> is the **identity** *watch list = weak tier*: a session reading this sentence alone builds a watch
> scoped to one disclosure, when the exposure is *any row whose grounds can move with no document
> changing anywhere* — which is the sentence in the box above and is wider than the tier. That is the disclosure's consumer, and it is the second and
independent reason §7 keeps it in the document rather than on a screen. It also gives the map's
curation patch a concrete first task instead of a posture.

`161/udp` is disclosed too and is **not** on this watch list: its weakness is a corroborator standing
where an owner should, which a release cannot change. That is a retrieval (#66), and the two kinds of
weak row must not be collapsed — one is watched, the other is chased.

**Marked as derived, not measured.** This section is read off §10.4's rule plus #21's measured
finding that sources move. Nobody has checked whether any of the three defaults has already moved;
that check belongs to the curation patch, which is where the watch lives.

> **Amendment — [#76](https://github.com/winniel123/verge-asm/issues/76): the watch list is
> re-enumerated, and it was wrong in both directions.** The three rows named above —
> **`5432/tcp`, `5984/tcp` and `9042/tcp`** — are left standing per the name-and-withdraw convention
> and are **superseded**. `9042/tcp` left the weak tier at
> [#69](https://github.com/winniel123/verge-asm/issues/69)
> ([`sensitive-ports.md`](../research/sensitive-ports.md) §12.7), when its shipped `conf/cassandra.yaml`
> was found to carry an owner prohibition naming the port — so the watch list has been carrying a row
> that does not belong on it since before this ADR was written, and no session propagated the change
> here. **[measured]** #76 adds `10255/tcp` kubelet, which rests on `readOnlyPort` *"Default: 0
> (disabled)"* in `kubernetes/kubernetes` `v1.34.1` and on no owner prose at all (§15.5).
>
> > **The rows exposed to silent de-attestation are `5432/tcp`, `5984/tcp` and `10255/tcp`.**
>
> **Three riders, and the second is the reason this amendment is worth making rather than noting.**
> The count is coincidentally three again, so a reader comparing counts rather than members would see
> nothing move. `2379/tcp` and `2380/tcp` etcd were expected by §13.7 to join this list and instead
> went to the **prohibition** tier on `THREAT_MODEL.md` at `etcd-io/etcd` `v3.7.1` — but that document
> is three months old and absent from the `3.5` and `3.6` lines, so §15.9 flags the pair as worth
> watching **despite not being on this list**, which is a shape §8 does not currently have a name for.
> And `4369/tcp` is a **chased** row rather than a watched one, in `161/udp`'s sense above: its
> weakness is that RabbitMQ, a non-owner, stands where the owner should (§10.5, §15.6), which a release
> cannot change and a retrieval can — routed to
> [#84](https://github.com/winniel123/verge-asm/issues/84).
>
> **No gate changes and no rule version moves.** A footing is evidence for a claim and not a claim, so
> `sensitive-port-reached-from-internet` is byte-identical and the list stays at **37 pairs**.

> **Amendment — [#88](https://github.com/winniel123/verge-asm/issues/88): `10255/tcp` comes off the
> watch list, and this is the enumeration's third correction in a week.** The line above is left
> standing per the name-and-withdraw convention and is **superseded**. **[measured]**
> [#79](https://github.com/winniel123/verge-asm/issues/79) §17.6 retrieved Kubernetes' security
> checklist — *"The Kubernetes API, kubelet API and etcd are not exposed publicly on Internet"* — and
> [ADR-0050](./0050-an-owners-category-statement-reaches-the-members-its-own-artefacts-place-inside-it.md)
> rules that an owner's **category** statement reaches the members the owner's own artefacts place
> inside it. `10255/tcp` therefore rests on an owner prohibition and not on `readOnlyPort` alone, and
> it joins the **explicit prohibition** tier with `10250/tcp`
> ([`sensitive-ports.md`](../research/sensitive-ports.md) §18.6).
>
> > **The rows exposed to silent de-attestation are `5432/tcp` and `5984/tcp`.**
>
> **Three riders.** The sequence is now **3 → 2 → 3 → 2** with the *membership* changing at every
> step, so §8's own lesson — a reader comparing counts rather than members sees nothing move — has a
> second and sharper instance: this correction and #76's are indistinguishable by count from the two
> before them. **The exposure this section names is a flippable default, and both kubelet cells are
> now exposed to its mirror** — a **checklist line in a documentation release branch**, which one
> contributor can edit in one commit; that is the shape §8 still has no name for, flagged for
> `2379`/`2380` by #76 and now carried by two more ports (`sensitive-ports.md` §18.7). And **both
> cells are conditional on [#83](https://github.com/winniel123/verge-asm/issues/83)**, which may
> remove either kubelet row; a footing cannot outlive its row.
>
> **No gate changes and no rule version moves.** The list stays at **37 pairs**.

> **Amendment — [#95](https://github.com/winniel123/verge-asm/issues/95): `10248/tcp` joins the watch
> list, and the sequence is now 3 → 2 → 3 → 2 → 3.** The line above is left standing per the
> name-and-withdraw convention and is **superseded**. `10248/tcp` kubelet healthz is admitted to the
> sensitive list in the **weak** footing tier, on a restricting `healthzBindAddress: "127.0.0.1"` shipped
> default **and nothing else** — no owner sentence, and **[measured]** `security-checklist.md` at
> `release-1.34` does not name the port, so ADR-0050's category statement does not reach it
> ([`sensitive-ports.md`](../research/sensitive-ports.md) §27.5, §27.12). ~~**The weak tier is the watch
> list, so the watch list grows with it.**~~
>
> > **THAT SENTENCE IS SUPERSEDED, AND IT WAS ALREADY WITHDRAWN WHEN IT WAS WRITTEN.** The box four
> > amendments above — [#102](https://github.com/winniel123/verge-asm/issues/102)'s, carrying ADR-0038's
> > *"§8's watch list is **redefined by shape**"* — withdrew the identity *watch list = weak tier*, and
> > this box re-asserts it below the withdrawal in the same section. That is
> > [#106](https://github.com/winniel123/verge-asm/issues/106)'s intra-document shape at the one
> > sentence the shape was named to protect, and it is visible from this file alone: the box carries
> > `#109` and `#114` strike-throughs, so it was edited **after** the withdrawal was written.
> > **[#125](https://github.com/winniel123/verge-asm/issues/125) supplies the replacement the
> > withdrawal never named** — [ADR-0057](./0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md).
>
> > **The rows exposed to silent de-attestation are `5432/tcp`, `5984/tcp` and `10248/tcp`.**
>
> **Two riders.** `10248` is exposed in the sharpest available form: its default is documented in a
> **published config-API doc comment** rather than in prose, so the artefact a curator must watch is a
> Go source file's comment, which one contributor can change in one commit without any release note.
> And §8's own lesson fires a **third** time — this correction is indistinguishable by count from the
> `3 → 2 → 3` before it, and it is the *membership* that moved.
>
> **Recorded in the merge reconciliation rather than by #95 itself**, which stated the delta in its own
> figure table and did not reach this file. **No gate changes and no rule version moves** on this
> account; a footing tier is not reference data. The list is ~~**41 pairs**~~ **40 pairs since
> [#109](https://github.com/winniel123/verge-asm/issues/109)**, which removed `1433/tcp` on the claim
> gate — class totals `12 / 7 / 21`. *(Struck in the #108/#109/#110 merge reconciliation; the figure
> above was correct as of its own pass and is withdrawn as an absolute, not as a record.)*
> **~~40 pairs~~ 38 pairs since [#114](https://github.com/winniel123/verge-asm/issues/114)**, which
> removed `9200/tcp` and `9300/tcp` on the same claim gate — class totals ~~`12 / 7 / 21`~~
> **`12 / 7 / 19`**, coverage **27 of 38** at tiers **13 · 11 · 3 · 11**. **§8's watch list is
> unchanged at three rows** — `5432`, `5984`, `10248` — no weak-tier member having moved.

> **Amendment — [#125](https://github.com/winniel123/verge-asm/issues/125): this section's *list* is
> superseded by an *instrument*, and the enumeration above stops being a watch.** Every row-count in
> the four amendment boxes above is a correct record of a **weak tier**, which is a real and unchanged
> object; **none of them is the watch.**
> [ADR-0057](./0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md) holds the ruling and
> [`sensitive-ports.md`](../research/sensitive-ports.md) **§39** the working.
>
> **The watch is two instruments.** A **gate** over what is **closed** — eleven checks, decidable over
> bytes already held plus a finite named set of targeted re-fetches, run to completion **over the table
> as edited**, an edit being complete only when the gate is green over the post-edit state. And a
> **queue** over what is **open** — somebody else's corpus, which never terminates and is therefore
> sampled. **[measured]** of the eight triggers a release can check **four completely**, **two in half**
> and **two not at all**; all six defect checks it can run completely, and the six defects and five of
> the eight triggers turn out to be **one instrument** — the reason the two lists looked different is
> that nobody had asked which of them **terminates**.
>
> **The queue keys on the *revision act*** — the smallest act by the owner that would falsify the cell,
> and whether that act publishes a notice we read — on five rungs read off the artefact, filtered to
> **sole-ground** cells, with the unit a **`(cell, artefact, revision act)` triple** and **its count
> barred as an indicator**. That is what retires this section's `3 → 2 → 3 → 2 → 3`: the sequence was a
> count over **rows** of a quantity that moves **per cell**. The other four candidate axes each get a
> different job — tier stays the **disclosure**, evidence age becomes the gate's bound and the queue's
> tie-break **in the owner's release line**, support count becomes the **filter**, and contradiction by
> the owner's own product documentation leaves the queue for the **gate**.
>
> **[measured] the footing tier is the wrong key, and this section had already recorded the
> counterexamples twice.** #76 flagged `2379`/`2380` as worth watching *"despite not being on this
> list … a shape §8 does not currently have a name for"*; #88 flagged both kubelet cells the same way.
> And one such de-attestation has **already happened** — `sensitive-ports.md` §36.7 records that
> `623/udp`'s direction disposal had to be re-founded, **HPE having retired the artefact carrying it**.
> `623/udp` is in the **top** footing tier and was never on this list.
>
> **The queue as of evidence already held is ~~eight items over ten `(port, transport)` pairs~~ nine
> items over eleven pairs and two non-port cells** — moved by
> [#135](https://github.com/winniel123/verge-asm/issues/135) /
> [ADR-0077](./0077-a-second-ground-counts-only-where-it-would-have-carried-the-cells-proposition-alone.md),
> which fixed the filter's bar (*carries the same cell* = **would have yielded the cell's proposition
> standing alone**) and applied it **per cell rather than per row**, adding `10250/tcp`'s claim cell —
> and it is a **superset** of the weak tier — all three rows above stay on it, so this
> supersession takes nothing off anybody's attention. `certificate-weak-key-or-signature`'s table
> contributes **zero** items, its residue being a **scope** weakness that de-attestation cannot reach;
> `certificate-expiring`'s horizon contributes **one**, ADR-0038 having removed the **quantity** from
> the watch and never the **attestation**; `verge-core`'s frequency half is the queue's other rung-1
> item, which discharges `project-authored-constants.md` §9's hand-off without a fourth pile.
>
> **§7 is untouched** — nothing here reaches a screen, and its condition is unchanged. **No gate
> changes, no row moves and no rule version moves**; the list stays at **38 pairs**, `12 / 7 / 19`,
> tiers `13 / 11 / 3`, coverage **27 of 38**, and **§4.6 goes to 24** on `9443/tcp`.

## The v1 walk — all ~~sixteen~~ **seventeen** rules, walked rather than asserted

The claim that thirteen rules carry no curated table is checkable, so it is checked.

> **The count moved to seventeen at [#128](https://github.com/winniel123/verge-asm/issues/128)**, which
> admitted a seventeenth rule carrying a table inside gate 2. The heading and the arithmetic below are
> marked at the sentence per
> [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) as widened
> by [#106](https://github.com/winniel123/verge-asm/issues/106). **Thirteen rules carrying nothing is
> unchanged**; what moved is the denominator and the gate-2 numerator.

| Rule | Curated table it reads | Instrument |
| --- | --- | --- |
| `certificate-expired` | none — `not_after` vs the clock, RFC 5280 field | — |
| `certificate-not-yet-valid` | none — `not_before` vs the clock | — |
| `certificate-expiring` | **`N = 30 days`** | **#21 gate 2 — ~~unattested~~ ATTESTED**, per the [#67](https://github.com/winniel123/verge-asm/issues/67) amendment above |
| `certificate-self-signed` | none — issuer ≡ subject, self-signature verifies. **No trust store**, which is why the rule is not named `untrusted` | — |
| `certificate-weak-key-or-signature` | **thresholds and a deprecated-algorithm set — ~~never written~~ WRITTEN**, [`weak-key-and-signature.md`](../research/weak-key-and-signature.md), five rows | **#21 gate 2 — [#68](https://github.com/winniel123/verge-asm/issues/68), which no longer blocks [#12](https://github.com/winniel123/verge-asm/issues/12)** |
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
| `non-globally-reachable-address-resolved-from-internet` | **the two IANA Special-Purpose Address Registries, transcribed** — [`special-purpose-address-registry.md`](../research/special-purpose-address-registry.md), 50 blocks, firing set 32 | **#21 gate 2 — attested by construction**; gate 1's own closed set derived at one member; gate 3 **outside the domain**. [#128](https://github.com/winniel123/verge-asm/issues/128), [ADR-0071](./0071-a-vantage-scoped-claim-is-read-only-at-the-vantage-that-scopes-it.md) |

~~**Thirteen of sixteen carry nothing this ADR governs.** Two carry a table inside gate 2 and one of
those two has no content yet.~~ **Thirteen of SEVENTEEN carry nothing this ADR governs**, per
[#128](https://github.com/winniel123/verge-asm/issues/128); **three** carry a table inside gate 2 and
all three now have content. One carries a table inside all three gates, and it is the one #21 was
written for.

## Consequences

- **[ADR-0004](./0004-signals-are-release-coupled-rules.md) is amended once**, in the place #60
  already corrected: its Consequences call `sensitive-port-reached-from-internet` *the only signal
  whose reference data we curate*; #60 made it two; it is ~~**three**, the third being
  `certificate-weak-key-or-signature`'s unwritten table~~ **FOUR since
  [#128](https://github.com/winniel123/verge-asm/issues/128)** — the third is
  `certificate-weak-key-or-signature`'s table, now written, and the fourth is
  `non-globally-reachable-address-resolved-from-internet`'s transcription of the IANA
  Special-Purpose Address Registry ([ADR-0071](./0071-a-vantage-scoped-claim-is-read-only-at-the-vantage-that-scopes-it.md)).
  Marked at the sentence per
  [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) as widened
  by [#106](https://github.com/winniel123/verge-asm/issues/106).
- **A curated table's cost gains an item.** Under ADR-0024 a rule is four parts plus one cost. Where
  a rule carries a table that asserts about the world, the table owes a **closed claim set derived
  from what the rule reads**, an **owner attestation per row**, and a **determinacy argument where
  its key is a surrogate**. This is not a fifth part of a rule; it is the table's own accounting.
- **Gate 2 is not a shipping gate.** A table with an unattested row **ships, disclosed**, exactly as
  `161/udp` and `5432/tcp` do. #37's precedent binds: a row may not move on a re-reading of text
  already held, only on a **retrieval**. So #67 and #68 are retrievals, not blockers on the number
  they concern.
- ~~**[#68](https://github.com/winniel123/verge-asm/issues/68) blocks
  [#12](https://github.com/winniel123/verge-asm/issues/12).**~~ A v1 rule with no predicate content
  cannot be assembled into a spec, and #12 must carry sixteen rules per
  [#64](https://github.com/winniel123/verge-asm/issues/64). #67 does not block it — `N` is fixed at
  30 and ~~the spec can carry it with the disclosure~~ **the disclosure is spent, `N` being attested
  per the #67 amendment above**.
  > **The blocker is DISCHARGED**, per the #68 amendment above: the table **has been written**
  > ([`weak-key-and-signature.md`](../research/weak-key-and-signature.md), five rows), so the rule has
  > predicate content and **#68 no longer blocks #12**. The general reasoning — a rule with no
  > predicate content cannot be assembled into a spec — stands. Marked at the sentence per
  > [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) as widened
  > by [#106](https://github.com/winniel123/verge-asm/issues/106); this is the one site in this file
  > where the unmarked clause reaches **[#12](https://github.com/winniel123/verge-asm/issues/12)**.
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
- **The map's curation patch gains its first concrete task** — ~~the three shipped-default rows are the
  watch list~~, and nobody has checked whether any of the three defaults has already moved.
  > **The equation is WITHDRAWN at §8 above and here too.** The watch list is defined by **shape**,
  > not by the weak tier, per
  > [ADR-0038](./0038-a-constant-is-a-product-only-where-the-quantity-is-readable.md)
  > ([#71](https://github.com/winniel123/verge-asm/issues/71)).
  > [#102](https://github.com/winniel123/verge-asm/issues/102) struck the §8 statement and left this
  > restatement of it standing 159 lines below — the one-sibling-got-the-mark failure #102 itself
  > recorded inside [ADR-0009](./0009-verge-core-is-a-union.md), committed again in the same pass.
  > Marked per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)
  > as widened by [#106](https://github.com/winniel123/verge-asm/issues/106).

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
