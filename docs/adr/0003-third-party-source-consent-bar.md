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
paths feeds an observation rather than a proposal — ~~RIPEstat's BGP leg is the live
candidate~~ **and there is no longer a candidate: the BGP leg is out of scope for v1 on
[ADR-0063](./0063-a-routing-announcement-names-the-path-not-the-estate.md), struck here at the site
that specifies it per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md).
The conditional stands and is now unwitnessed; see the sixth amendment.**

## Fourth amendment — 2026-08-13

[#57](https://github.com/winniel123/verge-asm/issues/57) went looking for the evidence behind
*asked, nobody replied* and did not find it. The **decision above is unchanged**, and so is
every ship state in the second amendment's table. What is withdrawn is a **claim about the
project's own conduct** that this ADR has been making since its first amendment, and that
four operator-facing surfaces inherited from it.

### The ask is unevidenced, in both directions

The first amendment records that
[#19](https://github.com/winniel123/verge-asm/issues/19),
[#23](https://github.com/winniel123/verge-asm/issues/23),
[#24](https://github.com/winniel123/verge-asm/issues/24) and
[#25](https://github.com/winniel123/verge-asm/issues/25) *"have all closed as **asked, no
answer**"*, and gives the first-run reason as *"we asked, nobody replied, and you can accept
the terms yourself"*. **The first half of that sentence is not something this project can
produce.** Measured across the four tickets, their GitHub timelines, and every file in the
repository's history:

- **No send confirmation of any kind**, on any of the four.
- **No recipient for three of the four.** #19 names `stat@ripe.net`; #23 and #25 say only
  *"Write to APNIC"* / *"Write to LACNIC"*; #24 names *"AFRINIC hostmaster"* with no address,
  the terms text having redacted it.
- **No send date.** The only timestamps in existence are GitHub lifecycle events.
- **No artefact anywhere in the repository.** Every file ever added on any branch is a
  glossary, an ADR, an agent doc, a research note, a prototype or the design system. There
  has never been a correspondence directory, a draft, or a mail log.
- **The four determinations were made in one batch, in under four minutes.** All four were
  assigned between `08:34:06Z` and `08:34:13Z` and closed between `08:37:37Z` and
  `08:37:43Z`. Under the wayfinder skill a session claims a ticket *before any work*, so the
  working window on each was about three and a half minutes, and one session wrote all four
  resolutions in one pass.
- **#24 says the quiet part in its own resolution**: *"With the email
  **unsent-or-unanswered** this stays unreported."* The session recording the non-answer did
  not know whether a question existed.

So the project cannot distinguish **sent and ignored** from **never sent**. That is not a
finding that the emails were never sent — it is a finding that **the project has no record
either way, and never had a mechanism to have one.** Writing to a third party is an act no
agent session can perform; these were `wayfinder:task` tickets of the kind the skill says
must be handed to the human as a checklist, and no such handoff, acceptance or report-back
is recorded.

### *We did not ask* is the same defect with the sign flipped

The tempting correction is to replace *asked, no answer* with *we did not ask*. It is refused.
A surface saying the project did not write is asserting a fact about the project's past acts
on exactly the evidence that failed to support the opposite claim. That is
[ADR-0004](./0004-signals-are-release-coupled-rules.md)'s *absent evidence yields
`not-evaluable`, never "did not fire"* arriving through the copy deck.

The honest statement is the one this project already has vocabulary for. **No reply has ever
been received** — a genuine negative the project can stand behind, since a reply would have
been recorded as everything else in this effort was. **Whether anything was sent is a `Gap`**,
and a `Gap` states its cause: the project keeps no record of outbound communications.

### The toggle does not rest on the ask, and never did

`operator-accepted` **survives untouched**, and the reason is already written down. The
second amendment defines it as *"the project could not clear these terms for the modal
operator and will not read them on a stranger's behalf; the operator, who is the party
actually bound by them, makes the reading and bears it"*, and
[`CONTEXT.md`](../../CONTEXT.md)'s entry says the same. **No clause of either depends on an
email.** The value rests on the project's *refusal to read*, which is a standing position and
not an event. The ask was the corollary's **procedure**; the toggle's legitimacy rests on the
corollary's **substance**.

This matters because the over-correction is expensive and available: concluding that the
toggles are illegitimate would delete the org→prefix entry point in four of five regions,
which is the second amendment's rejected option 1 with a new reason attached. A finding about
the project's filing is not evidence about a stranger's terms.

Two things do change. **The reason line loses its first half** — the wording is
[#57](https://github.com/winniel123/verge-asm/issues/57)'s and renders as *no reply has ever
come, and no record of an approach exists*. And **the ambiguity corollary's procedure has not
been shown to have been followed**, which is a debt against this ADR rather than against the
sources: whether it is discharged by sending or by formally recording that nothing was sent
is an outbound act and belongs to the dev, carried by
[#59](https://github.com/winniel123/verge-asm/issues/59).
**Discharged on 2026-08-14 by the second branch: the dev ruled the four asks and the two defect
reports will not be sent, and that ruling is recorded at
[`docs/correspondence/README.md`](../correspondence/README.md#the-four-asks-and-the-two-defect-reports-will-not-be-sent).
The debt is settled; ~~has not been shown to have been followed~~ is withdrawn as a live claim
about this ADR, and no source's ship state moved on the discharge — which is what the next
sentence already predicted.**
The ship states do not wait on it,
because *ambiguous terms ship off indefinitely* produces the identical result whether the
question was asked and ignored or never put at all. **Only what the project may claim
differs.**

### The project records outbound communications, or may not claim them

Outbound messages were the only class of project act in this effort with no artefact, and
the cost is measurable: [#28](https://github.com/winniel123/verge-asm/issues/28)'s `Coverage`
prototype shipped the sentence *"Asked 2026-06-02. Nobody replied."* — a **fabricated date**,
not a stale one, since it precedes the repository's first commit by two months. A surface
needed a precise fact, no artefact existed to supply one, and prose invented it.

So an outbound message is recorded as an artefact under `docs/correspondence/`, holding the
message **as sent** — recipient, date, full body, and any reply appended verbatim. The
governing rule is [ADR-0005](./0005-scan-execution-model.md)'s applied to the project's own
conduct rather than to a `Batch`: **record what completed, never what was attempted.**
#19–#25 recorded an *intention to write* and four surfaces then read it as evidence of a
send, which is a dead-lettered batch asserting absences it never measured, one layer up.
Absence of a file is therefore evidence of absence, which is what converts today's `Gap` into
a value the moment the dev acts. Drafts live under `docs/correspondence/drafts/` and are
evidence of nothing; sending means moving the file up one level.

The rider that generalises past this ADR: **the project may only claim an act it can
produce**, and this is the first time the rule has been turned on the project itself rather
than on a source or a measurement.

### The four resolution comments stand

#19, #23, #24 and #25 are not edited. [ADR-0007](./0007-drift-is-a-timeline-of-spans.md)
refuses to re-derive history — a wrong record is corrected by a new entry, never by a
rewrite — and a closed ticket is a record of what a session concluded on the day. Each
carries a correcting comment pointing here and at
[#57](https://github.com/winniel123/verge-asm/issues/57) instead.

### One habit, from how the defect reports were lost

#24 and #25 each bundled two different acts into one message: a **question the project
needs answered** and a **defect report the source needs** — AFRINIC's documented terms URL
returning 404, LACNIC serving a 7,014 B script shell at five documented policy addresses.
Both defects remain unreported because they were attached to a question that is itself
unevidenced. **Do not bundle a question the project needs with a courtesy the source needs.**
They have different recipients and different urgency, only one of them is ADR-0003's
business, and the courtesy has no fallback — it is simply dropped with the question. Neither
defect blocks anything: the second amendment's consent record already stores *the retrieval
failure* in the same slot as the document, so a fix at either registry is upside rather than
a dependency.

## Fifth amendment — 2026-08-14

[#78](https://github.com/winniel123/verge-asm/issues/78) widens the rule this ADR has been
applying since [#27](https://github.com/winniel123/verge-asm/issues/27) killed bundling CAIDA. The
rule was recorded as being about **data**: *redistribution is a separate permission from use, and
the party who needs it is the project.* It is not about data. It is about **permissions**, and the
data was incidental to the three cases that produced it.

Nothing already decided moves. This states the general form so that the next case is recognised
before it is built rather than after.

### The rule, restated

**A permission granted to the project does not travel to AGPL-3.0 recipients.**

AGPL-3.0 requires that every recipient of verge-asm receives the rights the project holds and may
convey them onward. A permission that attaches to *us* — by contract, by licence grant, by
credential, or by an individually negotiated waiver — cannot satisfy that requirement, because the
recipient never receives it. A tree resting on such a permission conveys rights it does not have.

### It now has four applications, and the fourth is what forced the restatement

| Case | Instrument | Why it fails |
|---|---|---|
| CAIDA ([#27](https://github.com/winniel123/verge-asm/issues/27)) | A bundled dataset under a **non-transferable** Public-AUA | The permission stops at the project; the file would ship past it |
| Censys, Shodan, Rapid7 ([#71](https://github.com/winniel123/verge-asm/issues/71) §5.1) | A **live-readable** ranking under terms barring incorporation into distributed products | Same shape, reached at runtime rather than at build |
| APNIC's bulk file ([#46](https://github.com/winniel123/verge-asm/issues/46)) | An **agreement the software cannot see** | Adjacent failure: not that the permission fails to travel, but that nothing enforces it — recorded here because the two are easily confused |
| **Nmap's waiver ([#78](https://github.com/winniel123/verge-asm/issues/78) §10.3)** | An **individually granted permission**, offered in writing | **The new one.** No dataset, no file, no credential — just permission, and it still fails |

The fourth is the one that shows the rule was mis-stated. Nmap's NPSL §0 offers it in the
licensor's own words: *"Open source developers who wish to incorporate parts of Covered Software
into free software with conflicting licenses may write Licensor to request a waiver of terms."*
That is a real, available, free-of-charge route past an incompatibility this project actually has.
It is refused, and it is refused on #27's rule — a waiver is granted **to the licensee who asked**,
and every downstream recipient of verge-asm would need one of their own.

**A licence incompatibility is not cured by permission. It is cured by not depending on the thing.**

### What this does and does not license

- It **does not** reopen any ship state in the second amendment's table. No source changes value.
- It **does not** reach `operator-credentialed`. That value already turns on a grant running to
  **the operator**, under their own terms, which is a different party from the project and is
  [ADR-0023](./0023-consent-names-the-door.md)'s subject. An operator's own credential is not a
  permission the project is conveying, and nothing here touches it.
- It **does** mean a session offered a project-specific permission — a waiver, a courtesy grant, an
  academic exception, a "just email us" — must treat it as **no permission at all** for anything
  that ships. Record the offer, decline it, and remove the dependence.
- The **fallback is always the third option**: relicensing is refused by the map, a waiver is
  refused here, so what remains is not depending on the artefact. #78 §9.4 walks that fork to its
  end and takes the third branch.

### The bar this ADR sets is unaffected, and that is the point

The two-limb modal-operator bar governs whether a source **may run**. This governs whether a
permission **may be relied on**. They compose without interacting: a source can clear both limbs
and still be unusable because the only route to it is a permission that stops at us. #78 is the
first case where the second question was the whole question — the NPSL analysis turned out to
require no permission at all (§6.1), so the waiver was never needed and the flag was **lowered
rather than raised**.

## Sixth amendment — 2026-08-15

[#126](https://github.com/winniel123/verge-asm/issues/126) rules the **BGP leg** out of v1 —
[ADR-0063](./0063-a-routing-announcement-names-the-path-not-the-estate.md), *a routing announcement
names the path, not the estate*. **Nothing in the decision, the two limbs, either corollary or any
ship state moves**, and that is the finding worth recording rather than an aside to it.

### The bar cleared the source and the source is still not shipping

RouteViews `api.routeviews.org/asn/<n>` clears **both limbs** with room. Limb 1: CC BY 4.0 permits
automated querying, local storage and cross-run retention, and imposes no retrieval-system clause.
Limb 2: its commercial condition triggers only *"when selling services, products, reports, or other
derivative works based on RouteViews Data to third parties"*, which is the **reseller** shape this
ADR's decision says does not fail. Its whole shipping obligation is attribution, and the fifth
amendment does not reach it — CC BY 4.0 is a public licence, so the permission and the attribution
duty both travel to AGPL-3.0 recipients intact.

It is nonetheless out, on **value**. That separation is the amendment: **clearing this bar is not an
argument for shipping.** The bar decides whether a source *may* run and is silent on whether it is
*worth* running, and #126 is the first case in the corpus where a candidate cleared it cleanly and
was refused anyway. A later session finding a source that clears both limbs has learned that no
toggle, email or reading is owed — and nothing at all about whether the thing earns a place.

### The third amendment's live candidate is withdrawn

That amendment closes with *"The claim returns the moment any of these paths feeds an observation
rather than a proposal — RIPEstat's BGP leg is the live candidate."* The **conditional stands** and is
untouched: a proposer that began producing observations would widen an aperture, and a subject first
observed under a widened aperture is still not *appeared*. What is withdrawn is the **witness**. The
BGP leg was the only named candidate for that promotion, and after ADR-0063 there is none — a route
collector proposes nothing here at any consent tier, so RIPEstat's toggle buys the org→prefix leg and
exactly that, which is what [#47](https://github.com/winniel123/verge-asm/issues/47)'s prompt copy has
said since it was drawn. Struck in place above, per
[ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) and
[#106](https://github.com/winniel123/verge-asm/issues/106)'s rider that a document supersedes itself.

### No table row moves, and the reason is worth stating once

The second amendment's *what ships, in what state* table is a table of **org→prefix** paths. It has
never carried a BGP row, because RIPEstat's `announced-prefixes` was never in the shipped set —
[#19](https://github.com/winniel123/verge-asm/issues/19) shipped RIPEstat off before anything
specified what its toggle covered. So [#20](https://github.com/winniel123/verge-asm/issues/20)'s
*"RouteViews replaces RIPEstat's `announced-prefixes` outright"* — true as a statement about
**capability** — has been read three times since as a statement about the **shipped set**, and the
displacement question #126 inherited was malformed on that reading. There was nothing to displace.
