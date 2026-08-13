# ADR-0003: A source ships enabled only if the modal operator clears its terms

- **Status:** Accepted
- **Date:** 2026-08-13
- **Ticket:** [#15 Which third-party discovery sources may ship enabled by default, given their ToS?](https://github.com/winniel123/verge-asm/issues/15)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

verge-asm is AGPL-3.0 and self-hosted, so its default configuration runs on the
infrastructure of people the project never meets, querying third parties the project has no
relationship with. A source enabled by default is therefore a decision the project makes
*on a stranger's behalf* about terms that stranger never read.

The passive discovery research
([#3](https://github.com/winniel123/verge-asm/issues/3)) applied a filter to this: a source
whose terms forbid automated or bulk querying, or restrict use to personal or
non-commercial purposes, cannot be a shipped default. It then recommended **RIPEstat** as a
Tier-1 default while recording that RIPEstat's own T&Cs are non-commercial and forbid
re-packaging — the same property that had disqualified six other sources. The map's target
user is a small-org security owner, which is a commercial entity. The exception needed
either a justification or a reversal, and the filter needed a statement precise enough to
apply consistently.

Reading the excluded sources' terms side by side shows why the filter kept producing an
exception: "non-commercial" was doing three unrelated jobs across the list.

- HackerTarget restricts content to *"your personal and non-commercial use only"* **and**
  forbids transmitting or storing it *"in any other website or other form of electronic
  retrieval system"*. The first clause is about **who the operator is**; the second is about
  **what the software does**, and it describes an asset database exactly.
- SSLMate's unauthenticated tier is scoped *"for personal or evaluation purposes"* — about
  who the operator is.
- RIPEstat has neither. Its permitted use explicitly includes *"network analysis, network
  monitoring and debugging and research"*, and its commercial prohibition names *"providing
  paid services, products or any other derivatives"* — a clause about **reselling the
  source's data**, not about a company using it on itself.

Three different clause types had collapsed into one word, so the filter could not be applied
without an exception each time it met a source whose restriction was of the third kind.

## Decision

**A source ships enabled by default only if the modal operator — a small commercial
organisation inventorying its own estate, doing exactly what verge-asm does — is inside its
terms.** Two limbs, tested in order:

1. **The software's inherent behaviour** must be permitted: automated querying, storing
   results in a database, retaining them across runs. A clause forbidding storage in another
   retrieval system, or copying a significant part of a database, fails here regardless of
   who the operator is.
2. **The modal operator's identity and purpose** must be inside the terms. "Personal",
   "non-commercial" or "evaluation" use only fails. A clause barring *resale or
   redistribution of the source's data* does not fail, because the operator is not doing
   that.

Two corollaries fix the cases the bar does not directly decide:

**Ambiguity is asked about, not read.** Where terms are genuinely in tension, the project
writes to the source operator and ships the source **off by default until an answer
arrives** — indefinitely if none does. Silence is not consent, and no elapsed interval
converts it into consent. This map has measured rather than assumed at every prior step; a
legal reading performed on strangers' behalf is the same kind of assumption.

**Absence of terms clears the bar.** A source that publishes no terms presents nothing to
breach. Its risks are operational and are governed as operational risks, not laundered
through a compliance rule.

Consent becomes a property of `Source` in [`CONTEXT.md`](../../CONTEXT.md) —
`unencumbered`, `operator-accepted`, `operator-credentialed` — so that "may this run without
the operator saying so" is a modelled fact rather than a deployment convention.

## Consequences

- **The bar disqualifies two sources, not seven.** Only **HackerTarget** (fails both limbs)
  and **unauthenticated Cert Spotter** (fails limb 2) are excluded on terms. CIRCL
  (discretionary vetted access), Rapid7 Sonar (dataset discontinued, now key-gated), ICANN
  CZDS (account gate), Wayback CDX (429 on the first unauthenticated request) and bgp.tools
  (login wall) are excluded for **availability**, which needs no policy at all. Recording
  this matters: a bar that appears to govern seven sources invites later sessions to reach
  for it as a general-purpose worry bucket, and it is not one.
- **CZDS is reclassified**, not merely excluded. Its output *is* a zone file, so it belongs
  to the Tier-0 operator-supplied ground-truth path rather than among sources verge-asm
  queries.
- **RIPEstat ships off by default**, pending
  [#19](https://github.com/winniel123/verge-asm/issues/19). It remains one click away under
  `operator-accepted`, so what is lost is first-run *depth*, not the capability.
- **First-run discovery depth is registry-dependent, and says so.** ARIN's `entities?fn=`
  org-name path carries the keyless default set and covers North America only. An operator
  with RIPE, APNIC, LACNIC or AFRINIC space is told at first run which capability is not
  available and what unlocks it — the annotation pattern
  [#10](https://github.com/winniel123/verge-asm/issues/10) already established. Whether that
  gap can be closed keylessly is [#20](https://github.com/winniel123/verge-asm/issues/20).
- **Changing a source's default state in a later release is an aperture change for every
  existing install.** Turning RIPEstat on in a point release would make every RIPE-region
  operator's estate appear to grow overnight — a change in the observer surfacing as a
  change in the world, delivered by us, estate-wide, via an upgrade. This binds future
  sessions changing defaults, not just this decision. The handling belongs to
  [#8](https://github.com/winniel123/verge-asm/issues/8); the rule is that a subject first
  observed under a widened aperture is not "appeared".
- **Disabling is safe, enabling is not.** Every opt-in source in play is `corroborative`, so
  its silence never asserted absence and turning it off cannot manufacture a removal. The
  entire aperture risk sits on the enable side.
- **The bar is a README obligation too.** An operator inventorying their own estate is
  inside ARIN's *"Internet operational or technical research"* permission; someone selling
  ASM-as-a-service built on verge-asm is inside its prohibition on use *"as part of a
  commercial service or product"*, and the AGPL does not exempt them. The distinction is
  documented rather than enforced, because the software cannot tell the two apart.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Structural bar** — gate only on what the software does; treat every clause about the operator's identity as a disclosure | Cleanly applied, and it keeps RIPEstat. But it ships the *modal* user into a default-on breach whenever a source says "non-commercial", and the modal user is precisely who the map is built for. Being wrong here costs credibility for a security product; being wrong the other way costs coverage |
| **Declared-status bar** — ask at onboarding whether the deployment is commercial, and compute the default set from the answer | Buys one source for a permanent onboarding question, and invites operators to self-certify into permissions they may not hold. It also puts the project in the position of having designed the loophole |
| **Presume unknown terms restrictive** — crt.sh to Tier 2 for publishing no terms | Would drop the only keyless way to get all certificates for `%.example.com` in one request, on the basis of a document that does not exist. crt.sh's real risks — 50% request failure, spurious 404s, an operator who has cut limits twice for abuse — are answered by the hard 5 req/min throttle, an identifying User-Agent, per-domain caching, and the rule that a non-200 produces no observation rather than an observation of absence |
| **Ship the ambiguous source on, with a deadline for the source operator to object** | Converts "we did not get permission" into "we waited long enough", which is the same guess the corollary declines to make, with a timer attached |

## Amendment — 2026-08-13

The four registry enquiries this ADR's ambiguity corollary set in motion —
[#19](https://github.com/winniel123/verge-asm/issues/19) (RIPE NCC),
[#23](https://github.com/winniel123/verge-asm/issues/23) (APNIC),
[#24](https://github.com/winniel123/verge-asm/issues/24) (AFRINIC) and
[#25](https://github.com/winniel123/verge-asm/issues/25) (LACNIC) — have all closed as
**asked, no answer**. The decision above is unchanged and its *"indefinitely if none does"*
branch is now the spec's settled answer rather than a pending state. Three corrections to
the Consequences follow.

**"What is lost is first-run *depth*, not the capability" is false for two regions.** That
line was written while RIPEstat was the only source in question, before
[#20](https://github.com/winniel123/verge-asm/issues/20) measured the keyless fallbacks. It
holds for APNIC (84.36% opaque-id coverage via CAIDA ⋈ delegated-stats) and AFRINIC (100%).
It does **not** hold for **RIPE** — CAIDA carries no opaque-id for RIPE, 0 of 39,640 — or
for **LACNIC**, whose CAIDA records carry no organisation names at all. For those two
regions the operator's toggle is not extra depth; it is the entirety of org→prefix coverage.

**"First-run discovery depth is registry-dependent" understates the current position.** The
recorded framing of ARIN as carrying the keyless default set alone is stale in the
operator's favour: [#20](https://github.com/winniel123/verge-asm/issues/20) closed AFRINIC
and APNIC keylessly. The residue is Europe and Latin America, and the reason to state at
first run is now uniform and final — *we asked, nobody replied, and you can accept the terms
yourself* — which is not a wait.

**`operator-accepted` is carrying two cases the two-limb bar does not decide.** APNIC's
*"stored in a retrieval system"* restriction is a **limb 1** question, and limb 1 fails
*"regardless of who the operator is"* — so an operator has no standing to consent past it,
and a limb-2 instrument is being applied to it. And LACNIC has no retrievable terms, so the
consent record specified in [#15](https://github.com/winniel123/verge-asm/issues/15)
(account, timestamp, terms URL) cannot be completed and its prompt has no restriction to
state; AFRINIC is a narrower instance, its terms readable but its documented URL a 404. Both
are open as [#34](https://github.com/winniel123/verge-asm/issues/34), which may yet amend
this ADR further.

## Second amendment — 2026-08-13

[#34](https://github.com/winniel123/verge-asm/issues/34) resolves what an operator is doing
at the toggle. The decision above is unchanged; one sentence of it is **scoped**, one value
it introduced is **redefined**, and the consent record
[#15](https://github.com/winniel123/verge-asm/issues/15) specified is **replaced**.

### The first amendment named the wrong axis

It treated APNIC as the outlier — a limb-1 question wearing a limb-2 remedy. Reading all
four non-answers side by side, the axis that matters is not which limb the ambiguity sits
in but whether the residual unknown **varies by operator or is constant across every
install**:

| Source | Unresolved question | Varies by operator? |
| --- | --- | --- |
| RIPEstat | does *"re-package, compile"* reach writing prefixes to an inventory database? | **No** |
| RIPEstat | is the operator reselling a service built on it? | Yes |
| RIPE Database | is estate inventory an Article 3 purpose? (Art. 4.1) | **No** |
| APNIC | does the retrieval-system clause's carve-out cover us? | **No** |
| APNIC | is inventory an *"operational purpose approved by APNIC"*? | Yes — approval is per-party |
| AFRINIC | does the closed permitted-use list name our use? | **No** |
| LACNIC | unknowable — the terms cannot be retrieved | — |

`operator-accepted` was designed for the SSLMate shape — *"personal or evaluation
purposes"*, where the project genuinely cannot know who the operator is and the operator
genuinely can. **Every one of the four toggles gates at least one constant question**, so
APNIC was never special, and an answer scoped to APNIC would have left the same defect
standing under three other toggles.

### Limb 1 governs the default, not the capability

*"A clause forbidding storage in another retrieval system … fails here regardless of who
the operator is"* is a test for **shipping enabled by default** — the decision sentence it
sits under says so — and it is silent on whether the capability may exist behind an
informed opt-in. Read the other way it would delete the org→prefix entry point in four of
five regions to protect a rule about defaults, at a point where
[#27](https://github.com/winniel123/verge-asm/issues/27) has established the address-seed
path is 100% covered in all five.

### `operator-accepted` is the operator's reading, not their compliance

The value does **not** mean the operator has certified to us that they are inside the
terms. It means: *the project could not clear these terms for the modal operator and will
not read them on a stranger's behalf; the operator, who is the party actually bound by
them, makes the reading and bears it.* That is this ADR's founding move applied one step
further out, not an exception to it — and it is why the rejected **declared-status bar**
stays rejected: that alternative computed the *default set* from a self-certification,
where this leaves the default off and moves nothing.

The reading only holds if the operator is told what is unresolved, so the prompt states it
**in the project's own words, never the source's**, in **two marked groups**:

- **what you may be able to resolve** — the operator-varying rows above; they can seek
  APNIC's approval and they know whether they resell;
- **what nobody has been able to resolve** — the constant rows, plus *we asked, nobody
  replied*.

Collapsing the two into one list buries the only actionable half under questions no one can
answer, which reads as *this is hopeless, leave it off*.

### Unretrievable terms do not get a fourth value

LACNIC asserts binding terms and serves a 7,014 B JavaScript shell at the URL its own RDAP
advertises. The toggle **survives**: the operator can do the one thing the project cannot —
ask LACNIC as a resource holder in their own region — and refusing the toggle forecloses
that permanently. `Consent` gains **no fourth value**; *we could not read the terms* is an
epistemic fact about our own assessment, and putting it on a property whose values all
answer *whose permission does this run on* would be a second property wearing an enum
case's clothes. It belongs in the prompt and the record, which is where the operator meets
it. *"Absence of terms clears the bar"* still does not reach this — unreadable is not
absent, or any source could earn `unencumbered` by breaking its own link.

### The consent record stores the document, not a pointer

[#15](https://github.com/winniel123/verge-asm/issues/15)'s **account, timestamp and terms
URL** is withdrawn. The record stores the **retrieved terms themselves** — bytes, a hash,
the retrieval timestamp — with the URL as metadata, and where retrieval failed it stores
**the failure** in the same slot. A record that cannot produce what was consented to is not
a record: AFRINIC's documented `afrinic.net/whois/terms` 404s, and a live URL can be
rewritten after acceptance with nothing to compare against. Note the boundary
[#27](https://github.com/winniel123/verge-asm/issues/27) drew: the copy is made **by the
operator's own install, at their own acceptance**, and is never bundled or redistributed by
the project.

### What ships, in what state, and what the operator is asserting

| Registry | Keyless (`unencumbered`) path | Live path | At the toggle, the operator asserts |
| --- | --- | --- | --- |
| **ARIN** | full org→prefix, ships on | — | — |
| **AFRINIC** | full org→prefix via CAIDA ⋈ delegated-stats | RDAP, `operator-accepted` | their own reading of a closed permitted-use list that does not name our use; terms readable, documented URL dead |
| **APNIC** | 84.36% of opaque-ids | registry, `operator-accepted` | their own reading of the retrieval-system carve-out, and whether they hold or will seek APNIC approval |
| **RIPE** | none | RIPEstat + RIPE Database, `operator-accepted` | their own reading of *"re-package, compile"* and of Article 3's purpose list, and whether they resell |
| **LACNIC** | none | registry, `operator-accepted` | that they accept a source whose terms **nobody has been able to retrieve**, stated as such |

### Consequences of this amendment

- **The name `operator-accepted` is kept.** It is underspecified about its object rather
  than false — unlike [#32](https://github.com/winniel123/verge-asm/issues/32)'s
  `sensitive-port-exposed`, which collided with a state that exists in the model and that
  the rule did not read. A glossary entry is the right instrument for underspecification,
  and a rename would strand the name across five closed tickets and this ADR.
- **The aperture rule is untouched.** Enabling any of these still widens the aperture for
  that install, and a subject first observed under a widened aperture is not *appeared*.
  For RIPE and LACNIC it is a zero→full widening, the largest the product can make.
- **This does not decide [#46](https://github.com/winniel123/verge-asm/issues/46).** A
  header that *clearly* bars us is not an ambiguity, and no toggle cures a plain no — so
  whether APNIC's bulk file may be indexed still turns on the clear-or-ambiguous fork that
  ticket owns.

## Third amendment — 2026-08-13

Two closed tickets landed against this ADR in the same cadence and neither edited it, to
avoid colliding with the other. Both corrections are recorded here.

### The clear-or-ambiguous fork has a third position

[#46](https://github.com/winniel123/verge-asm/issues/46) resolved the question the second
amendment's last bullet left open, and in doing so **withdrew that bullet's framing**.
*"A header that clearly bars us is not an ambiguity, and no toggle cures a plain no"*
treats *clear* and *barring* as one thing. They are separable, and APNIC's bulk-file header
is the case that separates them: it is **clear and conditionally permitting** — a plain
prohibition with a plain carve-out (*"Except for **agreed** Internet operational purposes"*)
and a **published mechanism** for reaching it, since
&lt;https://www.apnic.net/manage-ip/using-whois/bulk-access/&gt;, which reproduces the header
verbatim, says *"Requestors must sign the AUP agreement and lodge it with APNIC."*

So the ask-don't-read corollary is not triggered, [#27](https://github.com/winniel123/verge-asm/issues/27)'s
*clear text is not ambiguity* governs, and **nothing was sent** — the AUP is signed by the
party holding the data, so the project cannot be the requestor at all. Before reaching for
the corollary, look for the mechanism and ask **whose** act it is.

The full instrument is recorded separately as
[ADR-0018](./0018-a-clear-conditional-is-not-an-ambiguity.md), which is deliberately a new
ADR rather than a fourth amendment here: it states a general test about reading terms, not
a correction to this bar. It also broadens `operator-credentialed` — a **grant the source
actually made to that operator** (an API key or a countersigned agreement alike) as against
`operator-accepted`'s **reading the operator makes and bears** — and establishes that
`consent` keys on the **instrument**, never on the registry behind it, so one registry may
hold two values. The second amendment's table row for APNIC is therefore true of the
**live** path and does not govern the **bulk** one.

### Enabling these five widens no aperture

[#47](https://github.com/winniel123/verge-asm/issues/47) **withdraws** the second
amendment's Consequences bullet reading *"The aperture rule is untouched. Enabling any of
these still widens the aperture for that install… For RIPE and LACNIC it is a zero→full
widening, the largest the product can make."*

All five registry paths *propose*; none produces an observation.
[ADR-0012](./0012-a-proposer-is-not-a-source.md) states that nothing about a proposer widens
an aperture, and [#43](https://github.com/winniel123/verge-asm/issues/43) prices a
proposer's addition or removal at *no `revealed`, no `Break`, no version bump* — the only
class of third-party dependency in this project of which that is true. The withdrawn
sentence was written while these paths were still called `Source`s and **survived the
rename by three tickets**, which is the hazard worth carrying: a decision that renames what
a thing *is* should grep the closed tickets for claims about what it *does*.

What is true of RIPE and LACNIC is a **capability** statement, not an aperture one: without
the toggle there is no org→prefix path in those regions at all. The enablement prompt states
it that way and states the negative in terms — nothing enters the estate, a proposal is read
by nothing until confirmed, no timeline moves. The aperture act is confirming a `Proposal`
into a `Seed`, which is where ADR-0012 put it and where
[ADR-0022](./0022-confirmation-is-singular.md) now draws it.

The rule the withdrawn sentence generalised from is **unchanged**: enabling a genuine
`Source` widens the aperture, and a subject first observed under a widened aperture is not
*appeared*. It simply does not reach these five. The claim returns the moment any of these
paths feeds an observation rather than a proposal — RIPEstat's BGP leg is the live
candidate.
