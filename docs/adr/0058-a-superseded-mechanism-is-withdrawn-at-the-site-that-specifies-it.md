# ADR-0058: A superseded mechanism is withdrawn at the site that specifies it, not only at the site that supersedes it

- **Status:** Accepted
- **Date:** 2026-08-14
- **Ticket:** [#97 Is the hot set actually a superset? `10259` and `10257` were asserted into it, never measured](https://github.com/winniel123/verge-asm/issues/97)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[`sensitive-ports.md`](../research/sensitive-ports.md) §6 specified the relationship between the two
port lists as *"a one-directional invariant enforced at build time … with a test that fails the build
otherwise"*.

[ADR-0009](./0009-verge-core-is-a-union.md) dissolved that mechanism eleven months of tickets ago. Its
Decision table carries two adjacent rows — *"The invariant: **Dissolved.** `sensitive ⊆ verge-core`
holds by construction"* and *"Enforcement: **None, anywhere** — no build-time test, no config-load
check, no runtime check"* — and it made `verge-core` the union `frequency-set ∪ sensitive-list`, so
containment became **analytic**: `S ⊆ F ∪ S` is true of every `S` and every `F`, and the violating
state is not expressible. The ADR chose a definition over a test **on failure mode**, not on cost:
*"a test is a mechanism that can be absent, and the state it guards against is one a definition simply
cannot express."*

**ADR-0009 saw the reader this would strand, and answered with a pointer.** Its Consequences open:

> The `sensitive ⊆ hot` test [#21](https://github.com/winniel123/verge-asm/issues/21) §6 specified is
> **not written**. A reader finding §6 and no test in the codebase should read this ADR: the invariant
> was not dropped, it was made unfalsifiable.

§6 itself was left standing, unamended, in the present tense.

**Two passes then read §6 and wrote its mechanism forward.** This is the forcing measurement, and it
is a chain rather than an instance:

1. **[#91](https://github.com/winniel123/verge-asm/issues/91)** admitted `10259/tcp` and `10257/tcp`
   and wrote, **into ADR-0009's own body** as an amendment: *"**§6's one-directional invariant fires
   for the first time in earnest.** Neither new pair is in the frequency half, so *every pair on the
   sensitive list MUST be a member of the hot set* forces **two hot-set additions**."*
   [`sensitive-ports.md`](../research/sensitive-ports.md) §24.12 restated it. The mechanism was
   re-asserted inside the document that dissolved it, two screens below the row dissolving it.
2. **[#97](https://github.com/winniel123/verge-asm/issues/97)** inherited it at one further hop and
   stated the stakes as *"an addition to the sensitive list that is not in the hot set does not
   silently degrade, it **fails the build**"* — pricing itself as blocking
   [#12](https://github.com/winniel123/verge-asm/issues/12) on *"handing implementation a build-time
   invariant with an unverified term is handing it a build that may not go green."*

The third hop reaches code. #12 is the spec; carried forward, it instructs implementation to write the
build-time containment test ADR-0009 refused — a skippable guard over an unfalsifiable property, which
would also invite a later reader to conclude the union is maintained by the test rather than by the
definition.

The pointer did its job for readers who arrived holding the ADR. Both of these arrived holding §6.

**A second instance surfaced in the same pass, on a different pair of documents.**
[`safe-active-probing.md`](../research/safe-active-probing.md) §2.3's Management/OOB limb still reads
`161 (TCP), 623` — the exact two members ADR-0009's Decision table removed. A session enumerating the
frequency half from §2.3's own text gets **125** and never learns otherwise;
[#78](https://github.com/winniel123/verge-asm/issues/78) caught it only because it happened to be
counting for a licence question, and recorded the corrected **123** in a note nothing points to from
here. Same shape, different documents, no ticket in between.

## Decision

**When a decision supersedes a mechanism, the document that *specifies* the mechanism is amended in
the same change.** A note in the superseding document does not discharge the obligation.

| Concern | Decision |
| --- | --- |
| Where the withdrawal is written | At the **superseded** site, in the same change — and at the superseding site too, which is where the reasoning lives |
| What the superseded site gets | A **withdrawal**, naming what no longer holds and pointing at the decision — never a silent deletion, per the note's existing name-and-withdraw convention |
| Whether a pointer from the superseding document suffices | **No.** It reaches only a reader who already holds it, which is the reader who did not need it |
| Scope | Any **mechanism** — a test, a gate, a check, an enforcement point, a procedure. Not figures, which already have `FIGURE DELTA`, and not claims about the world, which have the amendment convention |
| What counts as discharge | The superseded sentence can no longer be read, in isolation and in the present tense, as specifying something that exists |
| Who does it | The pass that supersedes. It is the last party who knows both states |

**The test for a reader:** if the superseded sentence, read alone and out of context, would cause a
competent session to build or specify the thing — it is not withdrawn.

> **The unit is the *sentence*, and the #106 amendment below settles that it is.** *Site* here means
> the clause, not the file — so **a document supersedes itself**, and an amendment appended to a file
> does not discharge a clause standing unmarked earlier in that same file. Read *"the document that
> specifies the mechanism is amended"* as *"the sentence that specifies the mechanism is marked"*
> throughout; the two readings differ only in the intra-document case, and
> [#106](https://github.com/winniel123/verge-asm/issues/106) rules that case **in**.

## Rationale

### A pointer is addressed to the reader who does not need it

ADR-0009's *"a reader finding §6 and no test in the codebase should read this ADR"* is conditioned on
noticing an absence. But the failure mode is not a reader who looks for the test and cannot find it;
it is a reader who **believes the test exists and never looks**, because §6 says so in the present
tense and there is no reason to doubt it. #91 did not go looking for a test — it wrote *"the invariant
fires"* and moved on, correctly by §6's lights.

This is the same asymmetry [ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md)
already relies on one level down: an obligation discharged by *the reader will check* is discharged
only for readers with a reason to check.

### It is the *structural* move this project makes everywhere, applied to prose

ADR-0009 preferred a definition to a test because *"a definition cannot fail, because the violating
state is not expressible"*, and cited
[ADR-0007](./0007-drift-is-a-timeline-of-spans.md)'s `Break` and
[ADR-0008](./0008-derivation-versions-move-on-content.md)'s golden corpus as the same move. This ADR
applies it to the documents: a superseded sentence that has been **struck** cannot be read forward,
while one that has merely been **contradicted elsewhere** can be, and was, twice.

Amending the superseded site is not a documentation-hygiene preference. It is the only version of the
withdrawal whose violating state is not expressible.

### The cost is small and lands on the party who can pay it

The pass that supersedes holds both states in hand — it has just read the old mechanism in order to
replace it. Every later reader holds only one. Deferring the edit does not avoid it; it relocates it
to a session that must first discover the discrepancy, which is what
[#97](https://github.com/winniel123/verge-asm/issues/97) spent most of its budget on and what
[#78](https://github.com/winniel123/verge-asm/issues/78) did by accident.

### It does not license rewriting history

The withdrawal **names** what no longer holds; it does not delete the record. That is the convention
[`sensitive-ports.md`](../research/sensitive-ports.md) already uses everywhere — §3's `161/udp` row is
*"left standing … marked here rather than deleted, per the name-and-withdraw convention"* — and the
reason is unchanged: a deleted sentence takes its reasoning with it, and a reader who arrives via an
old citation finds nothing rather than a redirection. What this ADR adds is that the marking is
**owed**, and owed at the superseded site, rather than being available to whoever notices.

## Consequences

- **[`sensitive-ports.md`](../research/sensitive-ports.md) §6, §6.1, §6.3, §7.1 and §1's summary row
  are amended in place** by §29, and ADR-0009's #91 amendment has its *"forces two hot-set
  additions"* clause withdrawn. [`safe-active-probing.md`](../research/safe-active-probing.md) §2.3
  is annotated for ADR-0009's two removals.
- **[#12](https://github.com/winniel123/verge-asm/issues/12) must carry the union definition and
  §29.3's decomposition, and must not carry a build-time containment test.** This is the consequence
  that reaches code, and it is a subtraction from what #97 was priced to deliver.
- **The map's curation triggers gain nothing.** This is not a watch — it creates no obligation to
  re-read anything on a cadence, which is the shape
  [ADR-0046](./0046-a-negatives-corpus-is-its-owners-class-list-and-only-a-sole-ground-negative-is-exposed.md)
  refused on cost. It is a **step in a change that is already happening**, discharged when that
  change lands.
- **It is detectable rather than watched, and cheaply**, because both sides are written down: a
  document specifying a mechanism, and an ADR dissolving it. It joins the map's *detectable defects*
  group alongside *a row whose footing tier disagrees with its class*
  ([#83](https://github.com/winniel123/verge-asm/issues/83)) rather than its trigger list.
- ~~**The existing corpus is not swept.** Two instances are known and both are repaired here. Whether
  others exist is unmeasured, and a sweep is not opened — it would be a read of every superseding
  decision in the repository against every document it names, which is exactly the obligation the
  bullet above declines.~~ It is ticketed as a bounded question instead.
  > **Spent, by [#102](https://github.com/winniel123/verge-asm/issues/102)'s annotation below** — the
  > bounded sweep was opened, run and closed, and the population is in that annotation. This bullet is
  > marked here rather than only there because the withdrawal was written 88 lines down in the same
  > file, which is the intra-document defect [#106](https://github.com/winniel123/verge-asm/issues/106)
  > rules on: **this ADR carried an instance of its own shape.** What survives is the **cost estimate**
  > — the sweep really did cost a read of the whole corpus, and #102 priced it at 197 moves over
  > 39,382 lines.
- [`CONTEXT.md`](../../CONTEXT.md) needs no change. No term is added and none is amended.

## Alternatives rejected

**Leave the pointer and rely on the reader** — ADR-0009's own choice, made explicitly and in good
faith. Rejected on its measured failure rate: two of the passes that met it wrote the mechanism
forward, one of them into the superseding ADR's own body. The pointer is not wrong; it is addressed
to a reader who has already noticed the problem, and the failing readers had not.

**Delete the superseded sentence.** Rejected on the same ground the note's name-and-withdraw
convention was adopted: a deletion takes the reasoning with it, breaks inbound citations, and leaves a
later reader unable to tell a withdrawal from an omission. §6's argument for *why the coupling points
this way* is still live and still correct; only its final clause about enforcement is not.

**Make it a curation trigger on the map.** Rejected because a trigger implies a watch, and the watch
here would be *re-read every superseded document forever* — the standing obligation ADR-0046 exists to
deny, and the ground on which #93's ninth trigger lost. The discharge point is a change that is
already in flight, so it costs nothing to attach it there and everything to attach it to a clock.

**Restrict it to ADRs.** Rejected because the load-bearing instance runs the wrong way: the superseded
site here is a **research note** (§6) and the superseding site is an ADR, and the second instance is
note-to-ADR as well ([`safe-active-probing.md`](../research/safe-active-probing.md) §2.3 against
ADR-0009). A rule that only reaches ADRs would have caught neither.

## Annotation — [#102](https://github.com/winniel123/verge-asm/issues/102), 2026-08-14: the corpus was swept, and the shape has a population

#97 flagged this ADR's own thinnest ground: it *"rests on one measured instance with two hops, not on
a population"*. The bounded sweep #102 opened has been run, and **it has a population — six
superseding decisions, not one.**

**Corpus and extent, stated per
[ADR-0046](./0046-a-negatives-corpus-is-its-owners-class-list-and-only-a-sole-ground-negative-is-exposed.md)
as amended by [#93](https://github.com/winniel123/verge-asm/issues/93).** Swept at `main`
`c0881ae`: 52 ADRs, 11 research notes, `CONTEXT.md`, `docs/spec/`, `docs/agents/` — 39,382 lines.
**No ADR in this repository carries `Status: Superseded`**, so every supersession here is partial and
inline, and the amendment surface is the whole corpus of superseding moves: **38** heading-level
amendment/annotation sections plus **68** inline `> **Amended` blockquotes. **197 superseding moves
were examined and 76 were in scope as mechanisms.** The map archives were excluded by construction —
re-writing a snapshot is what an archive exists to prevent. This negative is **dated**, not permanent.

**The rule generalises past its own founding pair, and the direction of the generalisation matters.**
This ADR's Alternatives section rejected *restrict it to ADRs* because both founding instances ran
note-to-ADR. The sweep found the shape running **every** way: ADR-to-ADR
([ADR-0038](./0038-a-constant-is-a-product-only-where-the-quantity-is-readable.md) against
[ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) §8 and
[ADR-0034](./0034-derive-the-claim-before-looking-for-the-owner.md) §4, neither of which cited it),
ADR-to-note ([ADR-0015](./0015-the-value-space-is-the-commitment.md) against
[`insecure-listener-rules.md`](../research/insecure-listener-rules.md) §9.1, which contains no
reference to ADR-0015 at all), note-to-note
([`weak-key-and-signature.md`](../research/weak-key-and-signature.md) §13 against its own §9, where
the superseding section says outright *"Nothing above is rewritten"*), and **ADR-to-glossary**, which
is the earliest instance measured and predates this ADR by a day.

**The earliest instance cost a whole ticket.** [#27](https://github.com/winniel123/verge-asm/issues/27)
withdrew the registry half of `Ownership`'s derivation at
[ADR-0002](./0002-ownership-gates-probing.md); `CONTEXT.md`'s `Vantage class` and `Ownership` entries
went on specifying a registry read in the present tense, and
[#39](https://github.com/winniel123/verge-asm/issues/39) *"was filed and worked against a premise
ADR-0002 had already withdrawn"* — [`docs/agents/domain.md`](../agents/domain.md), which has carried a
**narrower, glossary-only statement of this rule since 2026-08-13** and which nothing here cited.
That rider is now cross-linked to this ADR in both directions.

**Two of the instances reach code rather than only prose**, which is the hop this ADR exists to stop:

- [`safe-active-probing.md`](../research/safe-active-probing.md) **§2.5** still specified the
  hand-picked UDP opt-in list `53, 123, 161, 500, 623, 1900, 5353` that ADR-0009 superseded — a list
  ADR-0009 measured as **missing four sensitive pairs**. §2.3 one section up was struck by #97; §2.5
  was not. Withdrawn in place by #102.
- [`insecure-listener-rules.md`](../research/insecure-listener-rules.md) **§9.2** excluded
  `smb-signing-not-required` from v1 on §9.1's *never per protocol* principle, which ADR-0015 ruled
  *"wrong as stated"*. The note's own §12 q2 asked the question; ADR-0015 answered it; nothing carried
  the answer back. The principle is withdrawn in place and **the verdict is ticketed rather than
  moved**, because a v1 signal is a row and not a document.

**The classification does not change: this stays a *detectable* defect and does not become a curation
trigger** — and the sweep strengthens rather than weakens that. The map's eight triggers are all
watches on the **world** moving; this defect fires when **we** move, so the trigger list is the wrong
list on shape before it is the wrong list on cost. The cost argument survives too: the watch a trigger
implies is *re-read every superseded document forever*, and #102's own run — 197 moves over 39,382
lines — is the price sample for one pass of it, against an expected yield that is zero except when a
pass has just superseded something. What the sweep does change is the **recording**: this ADR claimed
membership of the map's detectable-defects group and the map's curation patch still listed three
without it. It is now named there as the **fourth**.

**Thin ground, and it is a boundary this annotation draws rather than finds.** The sweep separated
**cross-document** instances, where the specifying document is silent and often does not cite the
superseding one at all, from **intra-document** ones, where a trailing amendment section amends the
file but the Decision-table row it supersedes stands unmarked hundreds of lines above. This ADR's
Decision says the withdrawal goes *"at the superseded **site**"* and its test is about a **sentence**
read alone — on which reading the intra-document cases are instances too. ~~They are reported rather
than repaired, because calling them instances widens this rule from *amend the other document* to
*mark the sentence*, which is a real extension and is not #102's to make silently.~~ The clearest is
[ADR-0047](./0047-an-address-scope-is-its-own-enumeration.md), whose amendment **names the sentence**
— *"the `32` … sits in the sentence a reader takes the enumeration rule from"* — and leaves it
unmarked 420 lines above.

> **The held-back clause is WITHDRAWN by the #106 amendment below**, which took the extension #102
> declined to make silently and made it explicitly: the intra-document cases **are** instances, and
> the ones this annotation enumerated are repaired rather than reported. The boundary this paragraph
> drew is dissolved; what survives is its **description** of the two populations, which is accurate.

## Amendment — [#106](https://github.com/winniel123/verge-asm/issues/106), 2026-08-14: the unit is the **sentence**, so a document supersedes itself

**Ruled: this ADR reaches a superseded sentence inside the document that supersedes it.** *Site* means
the clause, not the file. An amendment section appended to a file does **not** discharge a clause
standing unmarked elsewhere in that same file, however plainly the amendment states the withdrawal —
and it does not matter whether the amendment sits above the superseded sentence or below it
([ADR-0004](./0004-signals-are-release-coupled-rules.md)'s #60 amendment is 168 lines *above* the
Consequences bullet it withdraws).

**The narrow reading loses on this ADR's own founding measurement.** #102's annotation above held the
intra-document cases back as a weaker sub-shape, on the ground that the Rationale below argues from
*"a pointer **from the superseding document**"*. That framing explains why a pointer fails; it does
not bound where the rule reaches — and the Decision above already says *"at the superseded **site**"*
while the test is expressly about *"the superseded **sentence**, read alone"*. Taking the Rationale
over the Decision would be the narrow reading's price. But the decisive fact is in the Context above:
the forcing chain that founds this ADR is **itself intra-document**.
[#91](https://github.com/winniel123/verge-asm/issues/91) re-asserted the dissolved invariant
**inside [ADR-0009](./0009-verge-core-is-a-union.md)'s own body**, 217 lines below the Decision row
reading *"The invariant: **Dissolved**"* and *"Enforcement: **None, anywhere**"*;
[#95](https://github.com/winniel123/verge-asm/issues/95) then did it again 33 lines further down; and
[#97](https://github.com/winniel123/verge-asm/issues/97)'s withdrawal was written only in its own
amendment section, 87 lines past #91's clause, leaving that clause live until
[#102](https://github.com/winniel123/verge-asm/issues/102) struck it. **One file, three failures, and
a rule stopping at the file boundary catches none of them.** This is the objection the *Restrict it to
ADRs* alternative below already lost on, one turn sharper: there the load-bearing instance ran the
wrong *direction*; here it does not leave the document at all.

**The failure mode is indifferent to file boundaries.** The Rationale's diagnosis is a reader who
*"**believes** the mechanism exists and never looks"*, believing it because the sentence says so in
the present tense. Nothing in that mechanism consults where the contradicting sentence happens to
live. A 421-line ADR is not read whole; it is read at its Decision table.

**The extension is what #102 declined to make silently, and it is made here explicitly**, with its
cost stated below.

### What discharges it — the minimum mark

**Strike the clause and name the decision that withdrew it, at the clause.** Two forms, both already
in the corpus:

- **In prose** — `~~strike~~` the superseded clause and put a blockquote directly beneath it naming
  the withdrawing decision and stating what survives. This is verbatim what
  [ADR-0059](./0059-a-footing-tier-grades-evidential-distance-never-the-owners-conviction.md) does in
  three places for [#101](https://github.com/winniel123/verge-asm/issues/101), and what
  [`weak-key-and-signature.md`](../research/weak-key-and-signature.md) §9.2 does for #102's own repair.
- **In a table row** — `~~strike~~` the superseded text and carry a short **bold inline pointer in the
  same cell**. A blockquote cannot live inside a Markdown table, so the mark must be inline.

**The *"a Decision table row has little space"* constraint is refuted by measurement rather than by
argument. [measured]** the corpus already carries **20 marked table rows** — **9** with a struck cell
([`sensitive-ports.md`](../research/sensitive-ports.md),
[`safe-active-probing.md`](../research/safe-active-probing.md),
[`nmap-services-licence.md`](../research/nmap-services-licence.md)) and **11** carrying an inline
`Amended by` / `Superseded by` / `withdrawn by` pointer ([ADR-0011](./0011-a-facet-is-six-parts.md),
`sensitive-ports.md`) — across **118 strike marks on 67 lines**, all of that *before* this pass. The
form was available at every site this amendment repairs, and at several of them it was used one row
away.

A **full box** is not owed at the superseded site. The box belongs at the superseding site, which is
where the reasoning lives; the superseded site owes only enough that the sentence can no longer be
read forward.

### Where the rule stops — voice, not section

`## Alternatives rejected` is **out of scope**, and so is a `## Context` section narrating a prior
state — but **not by exemption**. They fall outside because of the test this ADR already states: an
entry whose grammar is *"Rejected on …"* or *"§6 specified …"*, read alone and in the present tense,
specifies nothing a session could build. The rule reaches a sentence that asserts, **in the
document's operative voice, that the mechanism holds**.

Two riders, because the line is a voice test and not a heading test:

- **A dated `## Amendment` heading does not put its bullets out of scope.** #91's amendment carried a
  dateline and an attribution and still wrote *"the invariant **fires** … **forces** two hot-set
  additions"* in the present tense. That is why it was struck.
- **The practical guard:** if the sentence needs *"as it then stood"* added to make it read
  historically, it was in the operative voice and owes the mark.

### The price, and the classification is unchanged

**It stays a *detectable* defect and does **not** become a ninth curation trigger.** #102's ruling on
shape binds and is unaffected: the map's eight triggers are watches on the **world** moving, and this
fires when **we** move. Nothing here creates an obligation to re-read anything on a cadence.

**The widening makes the defect *cheaper* to detect, not dearer** — which is the one place it improves
on the cross-document case rather than merely extending it. A cross-document instance needs the
superseding document read against every document it names; an intra-document instance is **one file
read**, both sides inside the same `grep`. The obligation on the superseding pass is likewise smaller:
the pass is already editing that file, and in the sharpest instances it has **already located the
sentence** — ADR-0047's #85 amendment says *"the `32` … sits in the sentence a reader takes the
enumeration rule from"* and then leaves it standing 416 lines up, and ADR-0004's #60 amendment opens
*"The Consequences below read …"*. Both had the citation in hand and stopped one edit short.

**The practice had already run ahead of the rule.** #102, in the same pass that declined this
extension in principle, discharged an intra-document instance in fact:
[`weak-key-and-signature.md`](../research/weak-key-and-signature.md) §9.2's retrieval obligation is
marked at the sentence against §13's amendment in the same file. So is #102's own strike inside
ADR-0009. What this amendment removes is the discretion, not the technique.

### The population, and it is repaired rather than reported

**[measured]** Swept at `main` `9a8f1df` over **71 files — 41,941 lines**: `docs/adr/` (54 ADRs,
16,064 lines), `docs/research/` (11 notes, 24,365), `CONTEXT.md` (899), `docs/agents/` (4, 207) and
`docs/spec/` (1, 406). The corpus has grown ~2,559 lines and two ADRs since #102's 39,382 at
`c0881ae`. `docs/wayfinder/` is excluded by construction, since re-writing a snapshot is what an
archive exists to prevent; `docs/spec/` and `docs/correspondence/` carry **no superseding move at
all** and contribute nothing. This negative is **dated**, not permanent, and it is re-armed by the next
supersession rather than by a clock.

**196 superseding moves were examined** — **37** heading-level `## Amendment` / `## Annotation`
sections and **74** inline `> **Amended` blockquotes, the two forms #102 counted (38 and 68 at
`c0881ae`), **plus a third convention #102's counts did not include: 85 inline italic
*`Amended by …`* / *`Corrected by …`* notes**, 71 of them in
[`sensitive-ports.md`](../research/sensitive-ports.md) alone. All 21 ADRs carrying a heading-level
amendment section were read end to end.

**The population is 25 sites, over 13 superseding decisions, across 12 documents** — #60, #62, #63,
#65, #67, #68, #71 (through [ADR-0038](./0038-a-constant-is-a-product-only-where-the-quantity-is-readable.md)),
#78, #85, #93, #95, #101 and #102, in ADR-0004, ADR-0006, ADR-0014, ADR-0025, ADR-0026, ADR-0028,
ADR-0032, ADR-0037, ADR-0046, ADR-0047, ADR-0059 and **this ADR**. #106's ticket enumerated six of
them; **nineteen had never been named.** All 25 are repaired in place on this ticket's branch —
**36 passages marked in all**, because eleven of them are second and third restatements of a clause
already withdrawn elsewhere in the same file. That is the first practical finding: **discharging a
site is a `grep` of the file for the clause, not a single edit.**

**The root cause is a misreading of the name-and-withdraw convention, and two documents state it
outright.** [ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md)'s #67
amendment says *"Three rows above say `certificate-expiring`'s `N` is **inside gate 2 and currently
unattested** … **They stand unrewritten per the name-and-withdraw convention**"*, and
[ADR-0004](./0004-signals-are-release-coupled-rules.md)'s #67 amendment says *"'shipped at **30
days**' above **stands unrewritten per this repo's name-and-withdraw convention** and is
superseded"*. Both read the convention as a **licence for silence**. It never was: the sentence they
inherit is `sensitive-ports.md` §3's *"left standing … **marked here** rather than deleted"*, and the
Rationale above already says so — *what this ADR adds is that the marking is **owed***. Six of the 25
sites are downstream of that one misreading, and it is why the population concentrates rather than
scattering.

**The shape does not respect document order, and an order-based rule would miss 9 of the 22 the sweep
confirmed.** In ADR-0004, ADR-0006, ADR-0028 and ADR-0032, the superseding section sits **above** the
unmarked sentence — the amendment lands mid-file and the stale clause is in the Consequences below
it. *Site* catches these; *"a trailing amendment supersedes an earlier row"* does not. This is why
the ruling is stated on the **clause**, and not on section order.

**The densest file is ADR-0032, with eight sites — and it is the file that invented the disclosure
obligation this amendment extends.** One of its unmarked clauses is the only one in the population
that reaches code: *"**#68 blocks #12.** A v1 rule with no predicate content cannot be assembled into
a spec"*, which its own #68 blockquote 400 lines above had already discharged —
[`weak-key-and-signature.md`](../research/weak-key-and-signature.md) was written and #68 stopped
blocking [#12](https://github.com/winniel123/verge-asm/issues/12). A session reading that bullet
alone re-blocks the spec on a discharged blocker. That is the third hop, and it is the hop this ADR
exists to stop.

**Every concentration is a one-sibling-got-the-mark failure**, which is #102's own diagnosis of
ADR-0009 generalised: in ADR-0032, ADR-0028, ADR-0026 and ADR-0009 alike, *some* sites carry the mark
and their siblings do not. #102 committed the same failure in the same pass — it struck ADR-0032 §8's
watch-list equation and left the Consequences restatement of it standing 159 lines below. The defect
is not carelessness about the rule; it is stopping at the first instance of the clause.

**Where the boundary genuinely bites, it bites narrowly.** The nearest misses are supersessions
written **1 to 3 lines** from the sentence ([ADR-0003](./0003-third-party-source-consent-bar.md)'s
third amendment, [ADR-0013](./0013-custody-is-control-and-extends-by-declaration.md)'s #55 heading) —
effectively at the site, and not repaired. The out-of-scope set behaved as the voice test predicts:
counts and figures ([ADR-0031](./0031-membership-alerts-at-the-root-of-the-entering-subtree.md)'s
*"all ten signal rules"*), claims about the world
([ADR-0050](./0050-an-owners-category-statement-reaches-the-members-its-own-artefacts-place-inside-it.md)'s
CISA sentence), withdrawn *reasons* under standing rulings
([ADR-0024](./0024-a-rules-domain-is-the-extension-of-its-name.md)), and
`## Alternatives rejected` entries.

**One deliberate non-repair, recorded rather than actioned.**
[`sensitive-ports.md`](../research/sensitive-ports.md) §10.6's *"Ruling: `161/udp` stays on the list,
disclosed"* is withdrawn by §11.6 some 370 lines below and stands unmarked — but it is a **row verdict**,
which the Scope row above puts under the amendment convention rather than under this rule. It is
noted here so the next sweep does not re-find it as a miss.

**This ADR carried an instance of its own shape**, which is the sharpest thing the sweep found: its
Consequences bullet *"The existing corpus is not swept … a sweep is not opened"* stood unmarked while
#102's annotation below recorded the sweep as run and closed. It is marked above.

### Thin ground

**The population's size is measured; its *rate* is not.** #102 could price the cross-document rule
against a measured failure rate — two passes wrote the mechanism forward, one of them into the
superseding ADR's own body. This amendment cannot say how often an unmarked *intra*-document clause
has been read forward, because the corpus records catches rather than misses:
[ADR-0030](./0030-an-offer-is-admitted-on-a-finding-or-on-a-falsity-it-prevents.md) quoted ADR-0025's
unmarked test row in the present tense and **noticed**, and the ADR-0009 chain is the one measured
miss. So the ruling rests on the **reach of this ADR's own words** and on its **own founding
instance**, not on a second failure rate. It is ruled anyway: the correction is one strike per clause,
paid by a pass already editing the file, against 25 sentences that were readable forward — one of
which re-blocks [#12](https://github.com/winniel123/verge-asm/issues/12) on a discharged blocker.

### Consequences of this amendment

- **The Decision above reads on the sentence.** Its *"the document that specifies the mechanism is
  amended"* is to be read as *"the sentence that specifies the mechanism is marked"*; the marker under
  the Decision table says so.
- **Nothing in the map's trigger list moves.** This stays in the *detectable defects* group, as #102's
  annotation ruled and for #102's reason.
- **The name-and-withdraw convention is restated, not changed** — *left standing **and marked***. Two
  documents had read it as licensing silence and are corrected in place.
- **[#12](https://github.com/winniel123/verge-asm/issues/12) gains nothing and loses one false
  blocker**: ADR-0032's *"#68 blocks #12"* is marked as discharged at the sentence.
- [`docs/agents/domain.md`](../agents/domain.md)'s rider gains one clause — *grep the document you are
  writing in*.
- [`CONTEXT.md`](../../CONTEXT.md) needs no change. No term is added and none is amended.
- **No ADR is minted**, per [#101](https://github.com/winniel123/verge-asm/issues/101)'s precedent that
  a clarification of an existing rule is amended in place.
- **The negative is dated at `9a8f1df`** and is re-armed by the next supersession, never by a clock.
