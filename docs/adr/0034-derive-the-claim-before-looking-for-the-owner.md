# ADR-0034: Derive the claim before looking for the owner — and ship the fraction, not its product

- **Status:** Accepted
- **Date:** 2026-08-14
- **Ticket:** [#67 Is `certificate-expiring`'s 30-day horizon attested by an owner, or does the number move?](https://github.com/winniel123/verge-asm/issues/67)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)
- **Findings:** [`docs/research/acme-renewal-timing.md`](../research/acme-renewal-timing.md)

## Context

[ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) ruled that
[#21](https://github.com/winniel123/verge-asm/issues/21)'s attestation gate binds every
project-authored table that asserts something about the **world**, and named
`certificate-expiring`'s `N = 30 days` as one of exactly three such tables — *inside gate 2 and
currently unattested*. [#60](https://github.com/winniel123/verge-asm/issues/60) had fixed the value
and recorded its reason as **stated but unverified against a source**. #67 is the retrieval, and
[#37](https://github.com/winniel123/verge-asm/issues/37)'s precedent is why it had to be one: *a row
may not move on a re-reading of text already held; a verdict changes on retrieval.*

The retrieval was expected to come back empty. It came back the other way, and the reason it did is
the whole of this ADR.

ADR-0032 §3 imposed an obligation on any second curated table and then did not discharge it:

> **So a second curated table does not inherit #21's three claims. It owes its own closed set,
> derived the same way — from what the rule reads — and the derivation is part of the table's cost.**

For `N`, that derivation was never performed. ADR-0032 filled the slot by quoting #60's prose gloss —
*the world: this is the last point the operator can still act* — and then went looking for an owner
for **that sentence**. There is none, and there could not be: it is a claim about the operator's
remediation capacity, which is not an artefact anyone designs, and which
`certificate-expiring` reads nothing about.

Performing the derivation made the owner appear immediately.

## Decision

| Concern | Decision |
|---|---|
| Order of operations under gate 2 | **Derive the claim from what the rule reads, then look for the owner.** A search for an owner conducted against an underived claim is not evidence of absence |
| What an underived claim looks like from inside a retrieval | **Exactly like a genuine absence**, which is why the order has to be a rule rather than good practice |
| `certificate-expiring`'s claim set | **Closed at one member by construction**: *the certificate is inside the portion of its own validity period in which its issuer says replacement is due.* The rule reads one interval and one instant, and a cut in an interval can assert only a position within it |
| Who owns that claim | **The issuing CA** — the party that fixed the validity period. §10.5 keys on the artefact, and the interval is the artefact |
| `N`'s value | **Moves.** `⅓ × (not_after − not_before)`, and `½ ×` that where the validity period is **10 days or less** |
| Its attestation | **Two owners, one half each.** The **IETF** attests the *form* ([RFC 9773](https://www.rfc-editor.org/rfc/rfc9773.txt) §1); the **issuer** attests the *value* (Let's Encrypt's Integration Guide, implemented in `boulder`) |
| Constants generally | **Where a constant is the product of a fraction and a moving world quantity, ship the fraction.** The product is a measurement taken at a date and it goes stale silently |
| §10.4's one-way default rule | **Scoped.** It applies to defaults over which *no act* is an available option. For a **required parameter** the permissive/restrictive axis is undefined and the rule is **outside the domain** — it neither admits nor excludes |
| What a required-parameter default is instead | **A corroborator** under §2.3. It states the maintainer's own behaviour, never a fact about the world, and may never carry a row alone |
| ADR-0032's disclosure obligation | **Amended.** *Name the retrieval that would resolve it* gains a second limb: where the retrieval has been **performed**, the disclosure names what remains unestablished and the condition that would move it |
| Is this a dial | **No.** All three of #60's grounds concern **per-install** variation; a per-certificate fraction is project-authored, fixed at the release, and identical in every install |
| Does it block [#12](https://github.com/winniel123/verge-asm/issues/12) | **No.** #67 never blocked it. #12 carries the fraction instead of the integer |

## Rationale

### 1. An underived claim and a genuine absence are indistinguishable from inside a retrieval

This effort has now performed three owner-hunts that came back empty, and the third was not like the
first two.

`111/tcp` ([`sensitive-ports.md`](../research/sensitive-ports.md) §2.7) and `161/udp` (§11) both
ended *we could not find anyone entitled to say so*, and in both the absence was real: the claim was
well-formed, the owners were correctly identified, and the sentence did not exist. Both produced the
right outcome — one exclusion, one removal.

`N` looked identical. A well-formed-sounding claim, an obvious place to look, nothing there. Had #67
stopped where the corpus ran out, it would have written a third *no owner exists* finding, and it
would have been **wrong** — not because it searched badly, but because it was searching for the
owner of a sentence the table does not assert.

**The failure mode has no signature.** There is no observable difference between *nobody wrote this
down* and *this is not the claim*, because both present as an exhausted search. The only defence is
to derive the claim first, and the only way to make that reliable is to require it.

> **A retrieval that returns no owner establishes nothing until the claim it searched for was derived
> from what the rule reads.**

### 2. The derivation, performed

`certificate-expiring` reads `not_before`, `not_after` and the clock. Its horizon is a **cut in the
interval `[not_before, not_after]`**.

One interval and one instant admit exactly one kind of assertion: **a position within that
interval.** There is no second property available, so the set closes by construction rather than by
enumeration — the same move [#37](https://github.com/winniel123/verge-asm/issues/37) made when it
derived gate 1's three claims from what an internet vantage supplies, and the same shape ADR-0032 §3
called a generalisation of the **method** rather than of the set.

| Candidate | Derivable from what the rule reads? | Owner |
|---|---|---|
| *The certificate is inside the portion of its validity period in which its issuer says replacement is due* | **Yes** | **The issuer** — it fixed the interval |
| *The operator can still act* — #60's gloss, inherited by ADR-0032 | **No.** Imports the operator's remediation capacity; the rule reads nothing about the operator | **None, by construction** |

The second is not a *worse* claim, it is **not this table's claim**. `certificate-expiring` is named
for the certificate, and [ADR-0024](./0024-a-rules-domain-is-the-extension-of-its-name.md) makes a
rule's domain the extension of its name — the discipline ADR-0032 §2 credited with deleting thirteen
tables before the gate is consulted. Here the same discipline, applied one level in, **repairs** a
table rather than deleting one.

### 3. The issuer owns the interval, so it owns positions within it

§10.5 defines an owner as *the party that designed the protocol, or that authors the reference
implementation, speaking about the thing it designed or wrote*, and its load-bearing rider is that
the rule keys on the **artefact**, not the party.

The artefact here is the certificate's validity period, and the party that fixes it is the CA. Let's
Encrypt's FAQ puts it as plainly as an owner ever does: *"There is no way to adjust these lifetimes,
there are no exceptions."* A party with unilateral control of an interval is entitled to say where
within it replacement is due — that is not a preference it expresses about someone else's artefact,
it is a statement about its own.

Let's Encrypt qualifies on the two conventional limbs as well: ISRG authored RFC 9773, and `boulder`
is the ACME server implementation ARI was built in. But **the issuance limb is the one that
generalises**, and it is what makes the finding portable to a CA that wrote no RFC.

**The scope limit is real and is disclosed rather than smoothed.** Let's Encrypt owns Let's Encrypt
certificates, exactly as §11.8 held that Cisco would own its own SNMP agent and not SNMP. v1 applies
one fraction to every certificate, which is an extension beyond the attestation and is the row's
named weakness. It is not cured by Certbot, lego and cert-manager applying the same fraction
issuer-agnostically, because a client default is a corroborator (§5).

### 4. The value: ship the fraction, because the product goes stale silently

> **Where a project-authored constant is the product of a fraction and a moving world quantity, ship
> the fraction.**

*(**§4's rule gains a reach limb**, per
[ADR-0038](./0038-a-constant-is-a-product-only-where-the-quantity-is-readable.md), written here by
[#102](https://github.com/winniel123/verge-asm/issues/102) under
[ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md). It applies
**where the moving quantity is readable from the subject at evaluation time**, and is **inapplicable —
not violated —** where it is not. Read without the limb, the sentence above licenses shipping the
fraction over a quantity nothing can read, which ADR-0038 rules out of the rule's reach entirely. Its
measured finding: **the rule's reach in this repository is one instance**.)*

`30` is not a quantity anyone attested. It is `90 × ⅓`, evaluated once, at a certificate lifetime
with a published expiry date. The owner's own words carry the arithmetic and its scope in one
breath: *"we recommend renewing certificates automatically when they have a third of their total
lifetime left … For Let's Encrypt's current 90-day certificates, that means renewing 30 days before
expiration."* The rule is the fraction; the thirty is an illustration. **#60 shipped the illustration
and discarded the rule.**

**This is ADR-0032 §8's silent de-attestation one level down, and it differs in both directions.**

- **Worse:** de-attestation needs a maintainer to flip a default. Silent staleness needs **nobody to
  do anything**. The world parameter moves on a schedule its owner published, and the repository's
  number becomes wrong on arithmetic while every document it cites still says what it said.
- **Better:** it is **preventable by construction**. A row expressing the fraction reads the moving
  quantity from the subject at evaluation time, so it cannot go stale. There is no watch to keep.

**The measurement that forces it rather than merely recommending it.** Certificate lifetimes are now
**plural**: Let's Encrypt's default is 90 days, its `tlsserver` profile is 45, its short-lived
profile is 160 hours and generally available since 2026-01-15, and the CA/Browser Forum's ceiling
steps to 100 days in 2027 and 47 days in 2029. Under a fixed `N = 30`, `certificate-expiring` is
**true for the entire life of every six-day certificate** — a predicate that cannot be false has
stopped partitioning, and [#53](https://github.com/winniel123/verge-asm/issues/53) has just made the
census the thing the operator reads. No fixed day count survives that spread: any `N` ≥ 6 days is
degenerate on short-lived certificates and any `N` < 6 days is useless on 90-day ones. **So the
answer could not have been a different fixed number**, which is the possibility #67's own constraints
left open and which measurement rather than preference closed.

The `½`-below-10-days limb is taken **whole** rather than in half, because taking an attested rule's
first clause and inventing its second is the failure the standard exists to prevent. The 10-day
threshold appears identically in the issuer's prose, in `boulder`, in Certbot and in lego.

### 5. §10.4 is scoped, and a required-parameter default is a corroborator

#67 was asked whether a renewal-lead-time default attests under §10.4's one-way rule. It does not,
and the reason generalises past this case.

§10.4 holds that a shipped default attests **only where it restricts**, because a restriction is a
**costly act** the maintainer paid for, while a permissive default is the **absence** of an act.
That dichotomy presupposes that *doing nothing* is available and that the permissive default is what
doing nothing looks like. `listen_addresses` has that shape.

A renewal trigger does not. A client that renews at no time does not function; every possible value
is an act and none is the absence of one. The axis is **undefined**, not merely hard to read.

> **§10.4 is scoped to defaults over which *no act* is a real option. Where the default is a
> **required parameter**, §10.4 is *outside the domain* — ADR-0024's third register — and it neither
> admits a row nor excludes one.**

That is the same three-way treatment ADR-0032 §4 gave gate 3, now applied to gate 2's third form,
and it is stated as a rule so the next required-parameter default does not relitigate it.

**What such a default is instead.** It remains a maintainer position in code, but a position about
**the maintainer's own behaviour**. Certbot's ⅓ says *this is when Certbot renews*; it does not say
*this is when a certificate must be replaced*, because Certbot neither issued the certificate nor
set its lifetime. So it lands in §2.3's corroborator tier and may never carry a row alone.

**The row does not need it.** Let's Encrypt's Integration Guide is §2.2's *second* form — the
vendor's own documentation, in prose, about the thing it issues. The row is carried by a first-party
position and the four client defaults sit exactly where corroborators belong. That is the inverse of
`161/udp`'s failure, and the contrast is the check that the standard is being applied and not
recited.

### 6. Why this is a move on a retrieval, and not #37's forbidden re-reading

Three facts were not in the repository and could only be retrieved.

- **RFC 9773 exists and is finished** — Standards Track, June 2025. #67's own brief called it *"the
  draft/RFC"*. Its §1 names a fixed lead time as creating *"significant barriers against the issuing
  Certification Authority (CA) changing certificate lifetimes"*, and pointedly does **not** so name
  a percentage of validity. That is the protocol's owner ruling against `N`'s **form**.
- **Certbot removed its fixed 30-day threshold in 4.0.0** and documents having done so: *"Prior to
  Certbot 4.0.0 the threshold was a fixed 30 days."* #60's premise was already false when #60 wrote
  it, and nothing on `main` could have shown that.
- **Six-day certificates are generally available**, since 2026-01-15 — which is what makes any fixed
  `N` degenerate rather than merely imprecise.

And the claim's re-derivation is not a re-reading either: ADR-0032 §3 **required** this table to
derive its own claim set, and the obligation was outstanding. Discharging an obligation that was
named and skipped is not revisiting one that was met.

**#60's rejected alternative is revived, and only one of its two grounds falls.** #60 considered and
refused *"`N` derived from the certificate's own validity period"* on two grounds: that it was *"a
formula we invented sitting inside the comparison path with no attestation"*, and that it makes the
census *"incomparable between endpoints"*. The first is **falsified by retrieval** — it is the
issuer's formula, not ours. The second **survives as stated and is overridden on argument**: a fixed
`N` already makes the census incoherent across endpoints, counting *every six-day certificate always,
every 45-day certificate for two thirds of its life, every 398-day certificate for 7.5% of its life*
in one column, while the fraction counts one thing everywhere — certificates in the last third of
their validity. That override is reasoning rather than measurement and the findings note flags it.

### 7. What the disclosure says once the retrieval has been performed

ADR-0032 made the disclosure obligation's stronger form binding on all three instruments: *a
disclosed weakness names the retrieval or measurement that would resolve it*, or it decays into a
permanent caveat. #67 is the first time that retrieval has actually been run to completion on a
disclosed row, and the obligation as written has nothing to say afterwards.

> **Amendment to ADR-0032.** Where the named retrieval has been **performed**, the disclosure names
> (a) what the retrieval established, (b) what remains unestablished, and (c) the condition that
> would move the row. A performed retrieval may not leave a row pointing at itself.

For `N` that reads: the retrieval established the form (IETF) and the value (the issuer); what
remains unestablished is that **one CA's fraction governs every issuer's certificates**; and the
condition that would move it is another CA publishing a different fraction for its own certificates.

**And `N` leaves both of ADR-0032 §8's piles.** It was filed as **chased** — a footing a retrieval
could establish. It is not now **watched** either, because ~~a row that reads the moving quantity from
the subject at evaluation time has nothing to watch~~. Two piles were thought exhaustive; a third
state exists and it is the one to aim for.

> **That clause is SCOPED, not true as written** — per
> [ADR-0038](./0038-a-constant-is-a-product-only-where-the-quantity-is-readable.md)
> ([#71](https://github.com/winniel123/verge-asm/issues/71)), written here by
> [#102](https://github.com/winniel123/verge-asm/issues/102) under
> [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md), because
> ADR-0038 recorded the scoping only in its own Consequences and **this ADR never cited it**.
>
> ADR-0038's words: *"a fraction removes the **quantity** from the watch, **never the
> attestation**."* `⅓`, `½` and the 10-day threshold remain the issuer's published values, the issuer
> may revise any of them, and that is an ordinary §8 attestation move. **The hazard is live and
> measured:** the CA/B Forum moved its short-lived definition from ≤10 days to **≤7 days on
> 2026-03-15** while the issuer, `boulder`, Certbot and lego all still use **10 days**. Read alone,
> the struck clause takes `N` off the watch entirely; what leaves the watch is the **number**, and
> the **attestation stays on it**.

## Consequences

- **[ADR-0004](./0004-signals-are-release-coupled-rules.md)'s #60 amendment is annotated.** Its
  *"shipped at **30 days**"* stands unrewritten per the name-and-withdraw convention and is
  superseded by the fraction. Its rationale sentence — *"30 days is where the ACME clients in the
  modal estate already trigger renewal, so it is the last point at which the operator still has the
  action the signal is telling them to take"* — is **withdrawn in both halves**: the first is a
  frequency claim excluded by §2.5 *and* false as of retrieval, and the second does not follow from
  it. It follows **against** it — a certificate at the point where the modal client renews is one
  whose automation is firing on schedule, which is the moment the operator has *no* action, not their
  last chance to take one.
- **[ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) is amended twice.**
  Its gate-2 table and its sixteen-rule walk both record `certificate-expiring` as *unattested, #67*;
  it is **attested**, and the claim recorded beside it is not the claim the table makes. And its
  disclosure obligation gains the performed-retrieval limb above.
- **The curated count stays three, and one of the three is now a different kind of object.** The
  sensitive list and the weak-key table are tables of values. `certificate-expiring`'s is a table of
  a **fraction**, which is why it cannot go stale and why it needs no watch.
- **`certificate-expiring` gains an input and loses nothing.** It reads `not_before` as well as
  `not_after`. `certificate-not-yet-valid` already reads `not_before`, so this is **no new
  measurement**, no facet change, and no ADR-0011 cost. Its `Predicate domain` is untouched.
- **The corpus obligation for clock-reading rules widens.** #60 established that such a row carries
  its **evaluation instant** as part of its input; it now carries **`not_before`** too, or the row's
  expected output is underdetermined. Three rules are affected and the widening is additive.
- **The cost is one `Break` on one rule for one cadence**, exactly as
  [ADR-0008](./0008-derivation-versions-move-on-content.md) and #60 priced a revision. It is
  **vacuous before the first install**, and §11.6's argument for acting now applies unchanged: the
  price today is zero and after v1 it is a comparability cycle.
- **No dial is minted and none is made legal.** #60's three grounds all concern per-install
  variation; the fraction is project-authored, ships in the release, and CI gates exactly the
  function every install evaluates. [#22](https://github.com/winniel123/verge-asm/issues/22)'s
  inside-versus-outside-the-comparison-path line is untouched.
- **The flap is unchanged.** On a healthy ACME estate the rule fires and clears every cycle under the
  fraction exactly as under 30. [ADR-0007](./0007-drift-is-a-timeline-of-spans.md) put all damping in
  notification so the model keeps the flap count as a fact, and #60 routed
  [#16](https://github.com/winniel123/verge-asm/issues/16)'s ACME complaint there. That routing
  stands.
- **[#12](https://github.com/winniel123/verge-asm/issues/12) carries the fraction**, not the integer.
  It was never blocked by #67 and is not blocked now.
- **[#68](https://github.com/winniel123/verge-asm/issues/68) inherits §1's rule and is not
  pre-empted.** Before hunting for the owner of *this key or algorithm is weak*, #68 derives that
  table's claim set from what `certificate-weak-key-or-signature` reads. ADR-0032 flagged the *who*
  as live for #68 and this ADR still does not answer it — what it supplies is the **order of
  operations**, and a warning that an empty result there means nothing until the derivation is done.
- **`CONTEXT.md` is not edited**, deliberately — concurrent sessions are in that file and #37 and
  ADR-0032 both set the precedent. The edit this ADR would make is one clause on `Signal`: *a rule's
  declared parameters express fractions of quantities the rule reads, wherever the quantity is one
  the world moves.*
- **The port-curation patch's third input is discharged and does not join the watch list.** The patch
  named `certificate-expiring`'s horizon as a curated input with the same open governance question as
  the other two. It no longer has one: nobody revises a fraction when a lifetime changes.

## Alternatives rejected

| Alternative | Why not |
|---|---|
| **Leave `N` at 30, disclosed, and record *no owner exists*** | The outcome the ticket expected, and it rests on accepting #60's gloss as the table's claim. ADR-0032 §3 obliged this table to derive its claim set and nobody had; performing the obligation makes an owner appear. It would also ship a predicate that **cannot be false** on every six-day certificate, which #53 has just made the operator's denominator |
| **Move to a different fixed number of days** | The ticket's own second constraint contemplated this, and measurement closes it: lifetimes now span 160 hours to 398 days, so any `N` ≥ 6 days is degenerate on short-lived certificates and any `N` < 6 days is useless on 90-day ones. No fixed day count exists that works |
| **Read the CA's ARI window per certificate instead of carrying a parameter** | Technically available — RFC 9773 §4.1's request is an unauthenticated GET keyed on AKI and serial, both observed, and §4.3.1 expressly contemplates third-party monitors. Refused because the window is a **CA load-balancing** instrument (*"dynamic time-based load balancing"*), which rendered to an operator as a replacement horizon is §2.7's laundering with a better-looking source; because `boulder` centres that window on the same ⅓ point the backstop prescribes, so it buys ±1% of validity; because no non-ACME certificate has one; and because it is an outbound per-certificate request to every CA in the WebPKI. Out of scope for v1, and a genuinely different signal if it ever returns |
| **Take the ⅓ rule without the ½-below-10-days limb** | Taking an attested rule's first clause and inventing its second is precisely what the standard exists to prevent. The threshold appears identically in the issuer's prose, in `boulder`, in Certbot and in lego |
| **Admit the row on the client defaults** (Certbot, lego, cert-manager agree) | Four implementations agreeing is a **frequency observation** and §2.5 excludes it. §5 additionally rules a required-parameter default a corroborator: it states the maintainer's behaviour, not a fact about the certificate. The row is carried by the issuer's prose and does not need them |
| **Extend §10.4 so a permissive-looking default attests here** | §10.4's axis is undefined for a required parameter, so extending it means inventing a reading rather than applying one. *Outside the domain* is the honest register and it is the one ADR-0024 already provides |
| **Make `N` operator-configurable now that it is per-certificate** | Not reopened and not reopenable here. #60's three grounds concern per-install variation and all three still bite. A per-**certificate** fraction is not a per-**install** dial, and conflating them is the confusion this ADR's §12 in the findings note exists to prevent |
| **Fold this into ADR-0032 as an amendment** | Two of its three rulings are not about tables at all — the order-of-operations rule governs how any owner-hunt is conducted, and *ship the fraction* governs every project-authored constant including the ones ADR-0032 puts **outside** gate 2 for being about our own measurement. Wrong population, the objection ADR-0032 itself made to folding into ADR-0004 |
| **Record the staleness finding as a watch-list entry** | It is the opposite of a watch item. A watch exists because someone must notice a change; expressing the fraction means nobody has to. Filing it as a watch would build the machinery the fix removes |
