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
