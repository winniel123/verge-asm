# ADR-0071: A vantage-scoped claim is read only at the vantage that scopes it — and transcribing an owner's table is not authoring one

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#128 Does a name resolving to a private-range address join the v1 signal set?](https://github.com/winniel123/verge-asm/issues/128)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[#128](https://github.com/winniel123/verge-asm/issues/128) — by-catch of
[#48](https://github.com/winniel123/verge-asm/issues/48), graduated from the map's fog on
2026-08-15 — asks whether a name resolving to a private-range address joins the v1 signal set, and
prices itself accurately as **a scope call rather than an evidence question**: publishing an
internal address in public DNS is a real disclosure that nothing in the v1 set catches, the rule
reads `resolution`'s **existing** address set, and the reference data is closed and spec-defined.

Two things it did not price, and each changes the answer's **shape** without changing its direction.

**The fact is vantage-relative.** From an internal vantage a name resolving into RFC 1918 space is
the *correct* configuration and nothing has been disclosed to anybody. It is a disclosure only where
the answer came back to a vantage on the internet. That is exactly the asymmetry
[ADR-0029](./0029-an-alert-fires-on-a-leg.md) used to refuse `sensitive-port-reached-from-internal`
— *"internally a Redis on 6379 is the correct configuration"* — and the map records that refusal as
**wrong rather than deferrable**. So the first live question is whether this candidate dies the same
death.

**The ticket names the reference data as a selection.** *"RFC 1918 / 6598 / 4193"* picks three
allocations out of a registry carrying fifty blocks, and
[ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) gate 2 binds a
project-authored table asserting about the world. A hand-picked three is such a table and there is no
owner anywhere for the claim *these three are the ones that matter*.

ADR-0032 §4 anticipated the candidate by name and left a forward statement about it:

> The nearest live candidate is the map's private-range-address rule, which keys on the IANA
> Special-Purpose Address Registry — a spec-defined closed set naming the addresses themselves, not a
> proxy for them. **It clears, and it clears for a reason that can be checked.**

This ADR performs that check rather than citing it, per the #67 amendment's rule that **a performed
retrieval may not leave a row pointing at itself**.

## Decision

**The rule ships, as `non-globally-reachable-address-resolved-from-internet`, and the v1 rule set
moves from sixteen to seventeen.**

| Concern | Decision |
| --- | --- |
| Does it ship | **Yes.** Neither of [ADR-0065](./0065-a-rule-is-excluded-by-its-fact-or-by-its-aperture-never-by-the-shape-of-the-set.md)'s two exclusion grounds is available, and the cost that remains is nearly nil |
| Name | **`non-globally-reachable-address-resolved-from-internet`** — the registry's own field, plus the vantage class that scopes the claim |
| Subject | The **`Name`**, one firing per name, empty discriminator — [#48](https://github.com/winniel123/verge-asm/issues/48)'s precedent, so a name with A and AAAA answers fires once |
| Which vantages it reads | **Internet-class vantages only**, and the class is in the rule's **name** and its **domain** |
| How they compose | **Existentially** — any available internet-class vantage's current `Resolved` set suffices. It is a **presence** claim, and #48's unanimity is for **absence** claims |
| `Predicate domain` | `Name`s whose internet-class `resolution` holds an **answer** — `Resolved` or `Shadowed` |
| Predicate | Any address in the set falls in a block the registry marks **not globally reachable**, read longest-match |
| `not-evaluable` | `Shadowed`; a disagreement no existential read resolves; **and every `Name` on an install with no internet-class vantage** |
| Outside the domain | `NameError`, `NoData`, `Lame` — no address set, so the question does not arise |
| The table | A **transcription** of the two IANA Special-Purpose Address Registries, [`special-purpose-address-registry.md`](../research/special-purpose-address-registry.md), 50 blocks, firing set 32 |
| ADR-0032 gate 1 | **Applies, and its own closed set is derived**: one member — *the allocating authority designated this block for a purpose other than global reachability* |
| ADR-0032 gate 2 | **Applies and passes by construction**, because the table is transcribed rather than selected. It is **not** inapplicable |
| ADR-0032 gate 3 | **Outside the domain.** The key is the address and the fact is about the address; there is no surrogate |
| Which of #31's two kinds of table | **Neither aperture nor a signature database** — a **verdict** table on #31's cut, governed by #21's instrument rather than #31's, because the byte was already decoded by a key normalisation that carries no version |
| Alerting | **Firing edge is a message** in the drift class, like every rule. **Clearing is silent** — it is not one of ADR-0026's four |
| New vocabulary | **None.** No facet, no field, no offer, no aperture input, no dial, no notification cause, class or trigger |
| Curated tables | **Three becomes four** |

### Two general rules

**1. A claim scoped to a vantage is read only at the vantage that scopes it — and a rule that reads
one carries the vantage in its name and in its domain.**

This is the positive form of ADR-0029's refusal, and it explains both outcomes with one sentence.
`sensitive-port-reached-from-internal` fails because #21's list is attested for what is *never
legitimately internet-facing*, and the internal leg is outside the context that gives its operative
term meaning — [#46](https://github.com/winniel123/verge-asm/issues/46)'s truncated conditional. This
rule succeeds because the vantage is **inside** it: the claim it reads is scoped to the internet, the
name says so, and the domain admits no other vantage class, so the rule can never be evaluated
outside the context that gives it meaning. **The distinction is not which fact is more alarming; it
is whether the scope is carried or assumed.**

The corollary binds in the closed direction: the internal-leg twin of *this* rule —
`non-globally-reachable-address-resolved-from-internal` — is refused here, in advance, on ADR-0029's
own ground. Internally, a name resolving into private space is the correct configuration.

**2. Transcribing an owner's table is not authoring one; selecting from it is.**

ADR-0032 gate 2 binds a **project-authored** table asserting about the world, and the tempting move
here is to call the registry *not ours* and the gate inapplicable. That is wrong, and it is wrong in
the direction that costs something: **the selection is always ours, even when the rows are not.**

> A table whose rows are copied from an owner's artefact and whose **selection predicate is a column
> of that same artefact** is a transcription. Gate 2 applies to it and passes by construction, because
> the artefact attests every row at once ([ADR-0037](./0037-an-attestation-is-retrieved-over-the-artefact-not-over-the-row.md)).
> The moment the selection predicate is a **judgement of ours** — *these three allocations are the
> ones that matter* — the table is authored, and it owes an owner for that judgement like any other.

The ticket's own framing is the worked example of the failing side and the rule corrects it: *RFC
1918 / 6598 / 4193* is a selection with no owner, and it is also simply worse — it misses loopback,
link-local, CGNAT, the three TEST-NETs, benchmarking space, `240/4` and `fe80::/10`, and it has no
answer at all for `192.0.0.9`, which sits inside a `False` block and is marked `True`.

## Rationale

### 1. There is no available ground to exclude it, and ADR-0065 is what makes that decisive

ADR-0065 closed the set of exclusion grounds at two and ruled the shape of the existing set is
neither. Walk both.

**The fact.** Its reference data passes ADR-0004's cadence test with room — 50 rows over a finite
address space, moving at RFC publication cadence, and staleness produces a blind spot rather than a
false verdict ([`special-purpose-address-registry.md`](../research/special-purpose-address-registry.md)
§5). It is named for the fact it reads and not for a conclusion its evidence cannot carry
([ADR-0010](./0010-exposure-composes-two-reaches.md)), not for a protocol
([ADR-0015](./0015-the-value-space-is-the-commitment.md)), and not for the content of its table
([#72](https://github.com/winniel123/verge-asm/issues/72)). Its claim is attested by the body that
made the allocations. **The fact ground is unavailable.**

**The aperture.** This is the decisive half. The observation **already exists**: `resolution`'s
`Resolved(unordered address set)` is measured every cadence, on the declared query path, and AAAA is
in the shipped offer. No facet gains a field, no value space widens, nothing `Break`s, no new
exchange goes on the wire, no `Offer` changes, and no aperture input is added. **The aperture ground
is unavailable, and it is unavailable in the strongest way — not *the aperture is cheap* but *the
aperture is already here*.**

What is left is ADR-0065's third route, which is not a principle at all: **a scope cost weighed and
declined is a complete exclusion.** So the honest question is what the cost actually is, and the
answer is one rule, one census, one table transcription, and the composition in §3. That is the
smallest bill any rule in the v1 set has ever presented, and the fact it buys is one nothing else in
the set catches. Declining it would be declining the cheapest thing on the menu.

**One thing that is *not* a ground and is recorded so it is not reached for.** *The set already has
sixteen rules and none of them is shaped like this* is ADR-0065's struck row, and the DNS half of the
set having been settled by #35 and #48 does not make a third DNS-reading rule an intruder. Rules are
versioned per rule precisely so that no rule's admission is a fact about any other.

### 2. #58's asymmetry is real, applies, and admits rather than refuses — because the vantage is inside the rule

The ticket flags this as the live comparison and it is the right one. Run it properly.

ADR-0029 refused `sensitive-port-reached-from-internal` on a **claim** failure, not a taste failure:
the sensitive list's rows are attested for *never legitimately internet-facing*, so on the internal
leg the list's operative term is void and the rule would rank nothing. The claim was scoped to a
vantage and the rule proposed to read it at a different one.

Our candidate has the identical structure and the opposite outcome, and it is worth being exact about
why, because *"the vantage is what distinguishes them"* is true and is not yet an argument.

- **The claim is likewise vantage-scoped.** *An address the allocating authority marks not globally
  reachable was served to us* is a disclosure **because we asked from the internet**. Asked from
  inside, the same answer is the operator's DNS working correctly.
- **The difference is that the scope is carried rather than assumed.** `sensitive-port-reached-from-internal`
  proposed to leave its claim's scope behind and keep the list. This rule puts the scope in the name
  and in the domain, so a version of it evaluated at the wrong vantage is not a mis-reading of the
  rule — it is **a different rule**, and that rule is refused above.
- **The mirror confirms the cut rather than merely fitting it.** The rule this ADR refuses in advance
  is the internal twin, on exactly ADR-0029's ground. A cut that admits one direction and refuses the
  other, on the same principle, for the same reason ADR-0029 gave, is a cut and not an exception.

**So #58 does not reach this candidate, and the reason generalises past both** — which is why it is
written down as Decision rule 1 rather than as a paragraph about DNS.

### 3. The composition is class-scoped and existential, and both halves are forced

`resolution` is keyed per **vantage**, so a rule over it owes a composition. #48 already wrote one —
ADR-0006's unanimity across every available vantage, composing `Availability`, adopted *"or every
split-DNS name flaps on every run"*. This rule needs a different one, twice over, and neither
departure is a preference.

**Class-scoped, because the all-vantage composition is unusable here.** On the modal install
([#14](https://github.com/winniel123/verge-asm/issues/14): internal, no outside prober) an
all-vantage read fires on every internal name in the estate — reporting the correct configuration as
a finding, at estate scale, which is #58's refusal committed rather than avoided. On a two-class
install with split horizon the vantages disagree and the composed value is permanently
`not-evaluable`, so the rule is dark on exactly the estates the fact matters for. Restricting to the
internet class is what makes the rule read the **public** answer, which is the only answer the claim
is about.

Class-scoped composition is not new machinery: `Reach` is defined as *"what vantages of one `Vantage
class` found"*, and `sensitive-port-reached-from-internet` already reads one class. What is new is
that `resolution` had not previously been read per class, and that is recorded as a cost in §7.

**Existential, because #48's unanimity is an artefact of the claim's direction.** #48 rejected an
asymmetric composition — *"unanimity to assert an absence, one vantage to assert a presence"* —
because applied to `resolved-name-absent-from-zone` it *"fires on every internal-only name in a
split-horizon estate"*. That reasoning is about mixing vantage **classes** on an **absence** claim.
Neither condition holds here: the class is fixed, and the claim is a **presence** — one internet
vantage receiving a private address is a disclosure whether or not a second one received something
else, and requiring the second to agree would silence the rule on every geo-DNS and rotating
authority. **[measured]**, that is not a hypothetical: ADR-0070 recorded one wildcarded name drawing
its addresses from **two disjoint pools at two vantages in one week**, and one authority's per-query
rotation reading as one answer through one resolver and seven through another.

The existential read errs **toward firing**, which is the direction `Vantage class` already chose
for itself and stated: *"a false `exposed` is investigated, a false quiet reading is not."*

### 4. The domain, the registers, and why `Shadowed` is load-bearing here more than anywhere else

Per [ADR-0024](./0024-a-rules-domain-is-the-extension-of-its-name.md) the domain is the extension of
the name — the `Name`s of which *a non-globally-reachable address was resolved from the internet*
could be asserted.

| The internet-class value | Register | Why |
| --- | --- | --- |
| `Resolved(set)` | **inside** — fires or does not | The answer is readable and the fact is true or false of it |
| `Shadowed` | **inside**, `not-evaluable` | A fact about **our own sight**: an answer exists and we cannot attribute it. ADR-0024's cut, unchanged |
| `NameError`, `NoData`, `Lame` | **outside the domain** | No address set at all, so the question does not arise — `plaintext-http-no-https`'s `NoHTTPResponse` exactly one facet across |
| No internet-class vantage configured | **inside**, `not-evaluable` | `sensitive-port-reached-from-internet`'s own behaviour on that install, mirrored rather than invented (ADR-0029) |

`Lame` is **outside** here where #48 made it `not-evaluable` for its two rules, and the difference is
not an inconsistency. #48's rules ask about a `NameError` and a lame delegation makes a `NameError`
unobtainable — the evidence for *that* predicate is absent. This rule asks about an address set, and
a lame delegation means there is no address set for the question to be about. Same value, two rules,
two registers, which is ADR-0024's own demonstration that a domain is a property of the **rule**.

**`Shadowed` carries more weight under this rule than under any other in the set, and the map's own
worked examples are the proof.** The wildcard thread's illustrations of what a bad discrimination
costs are *literally this rule's false positives*:
[ADR-0069](./0069-a-control-label-is-one-label-and-the-set-must-falsify-label-independence.md) records
`traefik.me` answering `10.0.0.1.traefik.me` with `10.0.0.1` and warns that set equality *"reports a
fictional RFC 1918 address as `Resolved`"*;
[ADR-0066](./0066-a-control-probe-is-generated-under-a-names-parent-and-that-population-is-aperture.md)
uses `Resolved([10.4.3.1])` for the same purpose. Under every other rule in the set a fabricated
address set is a wrong value on a timeline. Under this one it is a **fabricated firing on a fictional
name**. So `Shadowed` → `not-evaluable` is not a formality inherited from #35: it is the guard that
keeps this rule off the entire synthesised population, and **ADR-0066, ADR-0068 and ADR-0069 are load
bearing for this rule specifically**. The rule is the first consumer of the discrimination machinery
whose failure mode is a firing rather than a value.

### 5. ADR-0032, gate by gate, discharged rather than assumed

The ticket is explicit that the gate must be worked and not waved through on *the registry is
spec-defined*. It is worked in
[`special-purpose-address-registry.md`](../research/special-purpose-address-registry.md); the
verdicts and the reasons are here.

**Gate 1 — the closed claim set. Applies, and it is derived rather than inherited.** ADR-0032 §3 is
emphatic that a second table owes its **own** set derived from what its rule reads, and that
stretching #21's three is the open-set failure #37 closed. Derived per ADR-0034 — claim first, owner
after — the rule reads an address and a registry's classification of the block containing it, and a
registry of allocations asked about an address can assert exactly one thing: **the allocating
authority designated this block for a purpose other than global reachability**. One member. The
closure is tight for the same reason #21's was: it is read off what the instrument can say, not off
what we would like it to say.

**Gate 2 — attestation by the owner. Applies and passes, and *applies* is the load-bearing word.**
ADR-0032's inapplicable cases are tables about **our own measurement** and tables about the
**operator's own declaration**. This is neither: it asserts about the world. So the gate binds, and it
passes because the table is a transcription whose owner is IANA and whose artefact **is** the
allocation — the one table in the product with no gap between claim and claimant for an attestation
to cross. Per ADR-0037 the retrieval is over the artefact, so one retrieval attests all fifty rows.

Two riders keep this from being a free pass. The **selection predicate must remain the registry's own
column** (Decision rule 2); and where the registry declines to state a value — four `N/A` and
terminated blocks — **we decline to conclude one**, which is disclosed in the table's §6 with the
retrieval that would resolve it named.

**Gate 3 — determinacy. Outside the domain, and ADR-0032 §4 already said why in advance.** The gate
exists because `sensitive-port-reached-from-internet` *cannot read the fact it is named for*: a
`(port, transport)` pair is a **surrogate** for a service, and the port registry disclaims the
inference in capitals. Here the key is the address and the fact is a property of the address; the
registry does not stand in for the classification, it **is** the classification. Nothing is proxying
for anything, so the gate does not fail to generalise — it reads ADR-0024's third register, *the
question does not arise*, which is the shape ADR-0032 §4 predicted and the check it asked for.

### 6. #31's cut, answered directly: a verdict table, and not a signature database

The ticket asks which side of #31's line this falls on. **It is not aperture, and saying otherwise
would be the comfortable answer rather than the true one.** The table does not decide where to look —
`resolution` looks wherever the `Seed`s and the walk take it, and this table is consulted afterwards.
It decides what an answer **means**, which is the verdict side.

Three things follow, and the third is what makes it legal.

**The instrument that governs it is #21's, not #31's.** ADR-0032 §6 partitions on where a table sits
relative to the wire — *before* (what we ask), *at* (what a byte means), *after* (what a value means
normatively). #31's instrument is the middle one, and the bytes here were decoded long before this
table is reached: an A record's 32 bits and an AAAA's 128 become an `Address` **subject key** under
[ADR-0051](./0051-a-subject-key-is-the-thing-denoted-and-its-normalisation-may-never-move.md), by a
normalisation that is fixed at v1, carries no version and composes no leaf. What the registry adds is
normative meaning over an already-keyed value, which is the third position and #21's instrument.

**A verdict table is not automatically a signature database.** #31's line, written before ADR-0032
split three ways, names the failing kind of verdict table; ADR-0032 supplied the failure mode —
*"a false verdict that moves when a vendor's banner moves."* The discriminator that survives both, and
which is stated as **derived rather than measured**:

> **A verdict table is a signature database where staleness produces a false verdict, and it is
> reference data where staleness produces a blind spot.**

`nmap-service-probes`' `match` half is the first kind: a stale row emits a wrong product name, and its
own header solicits unbounded growth. This registry is the second: a block allocated after our release
is simply unregistered as far as our copy is concerned, so the address is outside the firing set and
the rule stays quiet. Nothing already in the table changes meaning underneath a comparison.

**The one place a naive read *would* produce a false verdict is caught, and it is caught by the
owner's own instruction.** The registries nest, with `True` blocks inside `False` ones. Read as *any
containing block*, the table reports `192.0.0.9` — marked `True` — as not globally reachable. Read
longest-match it does not, and longest-match is the IPv6 registry's own footnote on the row where it
matters: *"Unless allowed by a more specific allocation."* The construction detail that would have
made this a signature database is supplied by the owner, which is as good as this gets.

### 7. What it costs, stated in full

- **One rule and one census.** The v1 rule set moves 16 → 17 and every count in the corpus that reads
  sixteen is amended (Consequences).
- **The fourth curated table**, and ADR-0004's *"the count of curated tables stays at three"* moves to
  four. This is the third time that count has moved and the second time by an addition.
- **A second composition over one facet.** `resolution` now carries two — #48's unanimous all-vantage
  read for absence claims, and this rule's class-scoped existential read for a presence claim. Both
  sit **inside their own rules' leaves** and neither is a new named `Derivation`, so nothing widens
  the vector. But [#6](https://github.com/winniel123/verge-asm/issues/6)'s seam rule says every seam
  is a place drift can be manufactured, and two compositions over one facet is a seam. It is priced
  and flagged rather than smoothed over — see Consequences.
- **A second rule that is dark on the modal install.** `sensitive-port-reached-from-internet` is
  permanently `not-evaluable` where no internet vantage is configured; this rule joins it. The
  standing aperture statement on `Coverage` now carries two rules rather than one, and its honesty is
  ADR-0004's #44 amendment's, unchanged.
- **One inherited failure mode, in the loud direction.** `Vantage class` verifies `internet` for any
  vantage holding one address the operator never declared, which `CONTEXT.md` records as *"the loud
  failure and the intended one"*. A vantage inside the operator's own undeclared space resolving
  through their internal resolver will therefore make this rule fire on internal names. The failure
  is not new, the direction is the one already chosen, and the remedy is already named — the
  undeclared space surfaces as a coverage question.

## Consequences

- **[`CONTEXT.md`](../../CONTEXT.md)'s `Signal` entry gains one clause**, and its citation of
  ADR-0024's *sixteen rules* becomes **seventeen**.
- **[ADR-0004](./0004-signals-are-release-coupled-rules.md) is amended**: the seventeenth rule joins
  the v1 set, and the #33 amendment's *"the count of curated tables stays at three"* becomes four.
- **[ADR-0024](./0024-a-rules-domain-is-the-extension-of-its-name.md)'s v1 domain table gains a row**,
  and the count in its #64 amendment moves.
- **[ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) is amended in four
  places**: §4's forward statement about this candidate is **discharged** rather than left pointing at
  itself; the v1 walk gains a row; *thirteen of sixteen* becomes *thirteen of seventeen*; and gate 2
  reaches three rules rather than two.
- **[ADR-0015](./0015-the-value-space-is-the-commitment.md),
  [ADR-0026](./0026-the-facet-layer-is-evidence-not-a-channel.md),
  [ADR-0029](./0029-an-alert-fires-on-a-leg.md) and
  [ADR-0065](./0065-a-rule-is-excluded-by-its-fact-or-by-its-aperture-never-by-the-shape-of-the-set.md)
  each carry a *sixteen* that is now stale and each is marked at the sentence**, per
  [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) as widened
  by [#106](https://github.com/winniel123/verge-asm/issues/106). ADR-0026's *four of sixteen* clearing
  rules is unchanged in membership; only its denominator moves.
- **[#12](https://github.com/winniel123/verge-asm/issues/12) must carry seventeen rules**, and this
  ADR does **not** block it: the rule has predicate content and its table is written, so ADR-0032's
  blocking ground — *a v1 rule with no predicate content cannot be assembled into a spec* — does not
  apply.
- **Nothing is added to the notification vocabulary.** The firing edge is a message in the drift class
  like every rule's; a move that admits a name to the domain at `fired` is carried by the
  `resolution` `Transition` beneath it ([ADR-0033](./0033-a-move-carries-the-rule-that-opens-at-fired.md));
  an opening at `fired` with no move beneath is carried by the entering `Name`'s membership census
  ([ADR-0031](./0031-membership-alerts-at-the-root-of-the-entering-subtree.md)). The clearing edge is
  **silent** — it is not one of ADR-0026's four rules whose clearing condition is a name somebody else
  can claim; a record moving inside the operator's own zone is not an orphaned name being taken.
- **The census gloss states the rule as the rule expresses it**, per
  [#72](https://github.com/winniel123/verge-asm/issues/72): *fires where the address's most specific
  IANA special-purpose allocation is marked not globally reachable* — never the prefix list, and never
  a count over the estate.
- **A safety interaction is surfaced and is deliberately not ruled here.** An `Address` cited by a
  resolution is in the estate, and under a `custody extension` the probing gate opens on it. A name
  resolving to `10.0.0.5` therefore admits `10.0.0.5`, and an internet-class prober asked to probe it
  reaches **a machine on the prober's own network**. That is an
  [ADR-0002](./0002-ownership-gates-probing.md) and #4-profile question rather than a signal question,
  it predates this rule, and this rule makes the exposed population **visible** for the first time. It
  is handed on rather than decided.
- **The `resolution` composition question is handed on.** Two compositions now exist over one facet,
  both inside rules' leaves. Whether they should be named and enumerated — the way `Reach` names its
  class composition — is open, and it is the seam #6 warns about.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Refuse it, as #58 refused `sensitive-port-reached-from-internal`** | The strongest objection and the one the ticket names. It fails because #58's defect is a claim read **outside** the scope that gives it meaning, and this rule carries the scope in its name and its domain, so it can never be evaluated there. The internal twin **is** refused, on ADR-0029's ground, which is what makes this a cut rather than an exception |
| **Refuse it on scope — one more rule, one more census, one more table** | ADR-0065 permits this and it is the only permitted refusal here. It loses on the arithmetic: no facet, no field, no offer, no aperture input, no measurement and no `Break`, against a disclosure class nothing in the set catches. Refusing the cheapest item on the menu needs a reason and there is none |
| **Read the address set from every vantage, per #48's composition** | Fires on every internal name on the modal install — #58's refusal committed rather than avoided — and goes permanently `not-evaluable` on split-horizon estates, which is where the fact matters |
| **Require every internet-class vantage to agree** | Silences the rule on geo-DNS and per-query rotation, which ADR-0070 **[measured]** on real authorities. #48's unanimity is for **absence** claims across mixed classes; neither condition holds here |
| **Read RFC 1918 / 6598 / 4193, as the ticket proposed** | A **selection** with no owner (Decision rule 2), and materially worse: it misses loopback, link-local, CGNAT's v6 analogue, three TEST-NETs, benchmarking space, `240/4` and `fe80::/10`, and has no answer for `192.0.0.9`, which is `True` inside a `False` block |
| **Read the registry as *any containing block*** | Reports `192.0.0.9` — marked globally reachable — as not globally reachable. A false verdict, and the registry's own footnote instructs otherwise |
| **Read `N/A` and terminated cells as `False`** | Supplying a value the owner declined to supply, which is authorship. Disclosed instead, with the retrieval that resolves it named |
| **Call gate 2 inapplicable because the registry is not ours** | The comfortable answer the ticket warned against. The rows are not ours; the **selection** always is, and calling the gate inapplicable is what would license a hand-picked three next time |
| **Name it `internal-topology-disclosure` or `private-address-published`** | *Disclosure* is a conclusion the evidence cannot carry — the address may be a placeholder, a test net or a development convenience — and ADR-0010 refused exactly this for the DNS rules. *Private* names one row-class of the table's content, which #72 forbids |
| **Key it on the `Address` rather than the `Name`** | The fact is about the answer a name produced. Keying on the address would fire once per address and lose the name that published it, and would put a rule on a subject whose membership is disjunctive |
| **Fire per qtype on `dns-record`** | #48 settled this: a name declared with A and AAAA would fire twice for one fact, which ADR-0015 named when it ruled that one fact expressed several ways is one signal |
| **Alert on the clearing edge** | ADR-0026's exception is *a name somebody else can claim*, which is the takeover class. A record repointed inside the operator's own zone is not that, and the repoint is separately visible as `resolution` drift |
| **Defer it to v1.1** | ADR-0065's aperture ground is what makes a deferral clean, and it is unavailable: the observation already exists. A deferral here would be a scope refusal wearing a schedule's clothes |
