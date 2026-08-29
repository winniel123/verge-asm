# ADR-0057: A watch keys on the act that would falsify a cell, never on the tier of its evidence — and a release checks what is closed and reads what is not

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#125 Who revises the curated tables, on what watch, and what does the watch list key on?](https://github.com/winniel123/verge-asm/issues/125)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) §8 named a failure shape
and gave it a list. A row admitted on a **restricting shipped default** loses its footing the moment
that default becomes permissive, and **nobody says anything** — no source publishes a retraction, a
config default flips in a major release. §8 called the exposed rows *the curator's watch list* and
equated it with [#21](https://github.com/winniel123/verge-asm/issues/21)'s disclosed **weak tier**.

Three things have happened since, and together they say the instrument is not built.

**The list's count has been useless five times running.** Its sequence is **3 → 2 → 3 → 2 → 3** with
the *membership* changing at every step — `5432`/`5984`/`9042`, then `5432`/`5984`, then
`+10255`, then `−10255`, then `+10248`. §8's own lesson (*compare members, not counts*) has now been
restated as a rider on three separate amendments, which is a sign the unit is wrong rather than that
readers are careless.

**The identity it keys on was withdrawn and went on being asserted.**
[ADR-0038](./0038-a-constant-is-a-product-only-where-the-quantity-is-readable.md)
([#71](https://github.com/winniel123/verge-asm/issues/71)) ruled that *"§8's watch list is redefined
by shape, not by cause"* — **a weak row is watched wherever something must be noticed for it to stay
right** — and [#102](https://github.com/winniel123/verge-asm/issues/102) wrote that withdrawal into
ADR-0032 §8 itself under
[ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md). **[measured]**
the identity *watch list = weak tier* nonetheless appears as a **reason or a rule** at **19 sites** —
fifteen in [`sensitive-ports.md`](../research/sensitive-ports.md) and four in ADRs (0032, 0050, 0059,
0061), with five further sites *posing* the axis question and one, ADR-0034's, superseded in part.
The sharpest is ADR-0032's own `#95`
amendment box, which sits **below** #102's withdrawal box in the same section, says *"**The weak
tier is the watch list**, so the watch list grows with it"*, and carries `#109` and `#114` strike-throughs
proving it was edited **after** the withdrawal was written. That is #106's intra-document shape at the
one sentence the shape was named to protect.

**And the corpus has measured rows that are exposed and not on the list, twice, and recorded both
times that §8 has no name for the shape.** [#76](https://github.com/winniel123/verge-asm/issues/76)
flagged `2379`/`2380` as *"worth watching **despite not being on this list**, which is a shape §8 does
not currently have a name for"*. [#88](https://github.com/winniel123/verge-asm/issues/88) flagged both
kubelet cells the same way — *"a checklist line in a documentation release branch, which one
contributor can edit in one commit; that is the shape §8 still has no name for"*. And **[measured]**
one such de-attestation has **already happened**: §36.7 records that `623/udp`'s direction disposal had
to be **re-founded on a current artefact**, HPE having retired *Insight Online direct connect* — a row
in the **top** footing tier that was never on the list.

The map carried the governance half of #21's fog since #21 closed, and three notes handed work
straight to it: [`project-authored-constants.md`](../research/project-authored-constants.md) §9 hands
over *"the watch criterion widens per §8.3"*. `sensitive-ports.md` §27.14 records *"the map's curation
patch already carries **whether the watch list should key on tier or on volatility**"*. §31.12 adds
*"artefact class is a third candidate axis"*. Five candidate axes had accumulated — **tier · evidence
age · support count · artefact class · contradiction by the owner's own product documentation** — and
**no proposal for measuring any**. Nobody had said what *evidence age* would even be measured on.
First-seen date, release line and edit cadence give different answers.

The full working is [`sensitive-ports.md`](../research/sensitive-ports.md) **§39**.

## Decision

| Concern | Decision |
| --- | --- |
| **Who revises a curated table** | **The release.** The revising office is an **act**, not a person: a curated table changes only in the act that ships it, and the party is the **project**, never the operator ([ADR-0009](./0009-verge-core-is-a-union.md), ADR-0032 §7). A standing duty on a person is unfalsifiable and has already decayed three times |
| What licenses the edit | **A gate, run over the table *as edited*.** An edit to a curated table is **complete only when the gate is green over the post-edit state**. This is the only place the instrument bites, and it bites where the damage has actually been measured — at the edit |
| Does a red gate block the release | **No — it blocks the *edit*.** Blocking a software release on a documentation defect is disproportionate and would not be honoured; blocking the edit is proportionate and self-enforcing, because the fix cannot land while it still strands what it supersedes |
| **The watch is two instruments, not one** | **A gate over what is closed and a queue over what is open.** A check is **closed** where its population is enumerable and its evidence is bytes the project already holds, plus a finite **named** set of targeted re-fetches. It terminates, so it runs to completion. A check is **open** where its population is somebody else's corpus. It cannot terminate, so it can only be **sampled** — and sampling needs an order |
| **What a release can actually check** | **Every closed check completely, and no open check at all.** Of the eight watch triggers: **four completely** (a footing never checked against the standard it is now held to · silent staleness · a footing move re-arming the class sweep · a drafted document becoming issued), **two in half** (a protocol changing shipped defaults — the *tag* is checkable, the *substance* is not; citation staleness — *resolves and still occurs* is checkable, *still denotes the same thing* is not), and **two not at all** (a primary source changing position · a competing owner starting or stopping documenting its service). **All six defect checks run completely** |
| The six defects and the eight triggers are one instrument | **Yes, and the reason the two lists looked different is that nobody asked which of them terminates.** The gate is **eleven** checks; six come from the defect list and five from the trigger list |
| **What the queue keys on** | **The revision act** — the smallest act by the owner that would falsify the cell, and whether that act publishes a notice we read. Read **off the artefact**, never judged. Five rungs, and they are a total order a curator reads rather than scores |
| The tie-break inside a rung | **How far the owner has moved past the tag the cell was read at, measured in the owner's own release line.** This is what *evidence age* is measured on, and it is neither first-seen date nor our edit cadence — [ADR-0043](./0043-a-clock-reading-rule-bounds-its-evidence-in-the-subjects-own-units.md)'s shape: bound the evidence in the subject's own units, never the observer's |
| **The queue's filter** | **Sole-ground only.** A cell is an item exactly where **one** artefact carries it and no second **independent** artefact carries the same cell — ADR-0046 limb 1's rule read on the positive side. A **corroborator is never a ground** (§2.3), so a corroborator can never remove an item. **What *carries the same cell* denotes is fixed by [ADR-0077](./0077-a-second-ground-counts-only-where-it-would-have-carried-the-cells-proposition-alone.md)** ([#135](https://github.com/winniel123/verge-asm/issues/135)): the second artefact must, **standing alone**, have yielded **the cell's proposition** — what the cell asserts about the world, stripped of our grading of it — and **the test is run per cell, never per row**. *Exists* is not the bar; *at the same tier* is too high a bar, a tier being our own disclosure of evidential distance. An **undetermined** second ground counts as **absent** |
| **The unit** | **A `(cell, artefact, revision act)` triple, never a row and never a count.** A row can be an item twice on two cells; four pairs can be **one** item on one sentence. The count is barred from being quoted as an indicator, and the queue is published as members with their acts |
| Footing tier as the key | **Refused.** A footing tier grades **evidential distance** ([ADR-0059](./0059-a-footing-tier-grades-evidential-distance-never-the-owners-conviction.md)) — the count of premises the reader supplies. Volatility is a property of the **artefact**, not of the distance. **[measured]** they disagree on at least four pairs |
| Support count as the key | **Refused as a key, kept as the filter** — it removes items rather than ordering them |
| Evidence age as the key | **Refused as a key, kept as the gate's bound and the queue's tie-break** — in the owner's release line |
| Contradiction by the owner's own product documentation as an axis | **Refused — it is not on the queue at all.** It fires on a row that is **already wrong**, not one that might become wrong, so it is gate check G6, not a queue item |
| The three cause-piles | **Not collapsed and not re-piled.** *Watched* · *chased* · *scope* stay as **causes of weakness** (ADR-0038 §7) and all three land on **one** queue, because the revision act — not the cause — is what fixes priority. A **scope** weakness is an item only where an artefact could supply the missing modality |
| **Who may move a row** | **Only a human release act.** A machine may **raise** an item; the watch's output is a **question**, never a verdict, because moving a row **authors a claim** and §2.2 bars us from asserting one |
| What a release owes the queue | **A disclosure, never a quantum.** No number of items per release is set: a quantum is a figure nobody can attest, and it would make the queue's *length* meaningful again — which is the failure being fixed. What the release owes is the **bounded residue** ([ADR-0040](./0040-a-specifications-silence-is-not-the-owners-silence.md)): the queue's head, how far down it read, and what it did not read, falsifiable by naming one item |
| Where this lives | **Not `CONTEXT.md`.** The curator is not a subject in the model and the product holds nothing about it; the instrument acts at spec time on documents. It lives in this ADR and in each instrument's own document — ADR-0032 §7 unchanged |

### The five rungs

The rung is the **revision act**: the smallest act by the owner that would falsify the cell, and
whether that act publishes a notice we read. It is read off the artefact — what kind of thing is it,
and how does that kind of thing change — and never off what the sentence says.

| Rung | The act | Announced? |
| --- | --- | --- |
| **1** | A source comment, an unversioned page, or a dataset with no successor — one contributor, one commit, **and nothing rendered changes** | No, and it is invisible even to a reader of the owner's rendered documentation |
| **2** | A continuously-published documentation page with no version pin, or a documentation branch tracking a release line — one contributor, one commit | No release note; the page simply reads differently tomorrow |
| **3** | A **shipped configuration default** — changes only in a software release, carrying a version and usually a changelog entry | Announced by a version we can pin and diff |
| **4** | **Issued prose in a versioned documentation set** — changes on a documentation release with a version | Announced, and the prior version stays retrievable |
| **5** | A **specification** — changes only by a new document with a new number, the old one retrievable forever | Announced, and never silently |

**Rung 1 is where §8's whole hazard lives**, and the corpus already named it twice without a name for
it: `10248/tcp`'s footing is *"a Go source file's comment, which one contributor can change in one
commit without any release note"* (#95), and the docs-source-comment hazard (§38.14 item 6) is the same
rung read from the retrieval end.

### The thirteen gate checks

Each is decidable over bytes the project already holds, or over a **finite named** set of targeted
re-fetches. Six are the detectable-defect list. Five are watch triggers that turn out to terminate. G12
and G13 were added in the same merge, per [#149](https://github.com/winniel123/verge-asm/issues/149) and
[#152](https://github.com/winniel123/verge-asm/issues/152)/[ADR-0099](./0099-a-stated-horizon-is-a-second-comparand-a-tag-match-does-not-discharge.md)
respectively — ruled to be two checks, not one, over disjoint populations (see the rationale below).

| # | Check | Source |
| --- | --- | --- |
| **G1** | A row's footing tier agrees with its class where the two are coupled | defect — [#83](https://github.com/winniel123/verge-asm/issues/83) |
| **G2** | Every graded footing cell has been walked against the tier criterion **as it currently stands**, and the walk's date is at or after the criterion's last amendment. **Its population is the criterion's DOMAIN — the two graded tiers, 24 cells, and NOT all 27 footing cells: the weak tier is outside the domain, [ADR-0059](./0059-a-footing-tier-grades-evidential-distance-never-the-owners-conviction.md)'s conjunction reading an owner statement and a weak cell having none ([#154](https://github.com/winniel123/verge-asm/issues/154), `sensitive-ports.md` §46)** | defect — [#93](https://github.com/winniel123/verge-asm/issues/93), [#98](https://github.com/winniel123/verge-asm/issues/98), [#101](https://github.com/winniel123/verge-asm/issues/101), [#107](https://github.com/winniel123/verge-asm/issues/107); **and triggers 3 and 6, which are this check read at two moments** |
| **G3** | No row rests on an owner's documented hard failure its shipped bytes do not enforce | defect — [#92](https://github.com/winniel123/verge-asm/issues/92) |
| **G4** | No sentence the edit supersedes stands unmarked anywhere the edit's terms occur, **the edited file included** | defect — ADR-0058, [#102](https://github.com/winniel123/verge-asm/issues/102), [#106](https://github.com/winniel123/verge-asm/issues/106) |
| **G5** | No exclusion's stated ground has been withdrawn | defect — [#104](https://github.com/winniel123/verge-asm/issues/104) |
| **G6** | No artefact **already held** carries an owner affirmation defeating a held row | defect — [#109](https://github.com/winniel123/verge-asm/issues/109), [#112](https://github.com/winniel123/verge-asm/issues/112), [#114](https://github.com/winniel123/verge-asm/issues/114). Finding a **new** affirmation is a queue item; grepping one we already downloaded is not |
| **G7** | Every quotation retrieved from **raw source bytes** is present in the owner's **rendered** artefact | trigger — new, forced by §38.14 item 6; the ADR-0045 rider at §39.6 |
| **G8** | Every citation resolves, and its quoted string is still a **token** of the artefact at the tag named | trigger 5's closed half — §37.14, §38.16 |
| **G9** | No shipped constant is the product of a fraction and a moving world quantity the subject carries | trigger 4 — [ADR-0034](./0034-derive-the-claim-before-looking-for-the-owner.md), ADR-0038 limb 3 |
| **G10** | Every document refused as **unissued** is re-tested for issuance | trigger 7 — [ADR-0045](./0045-an-owners-documentation-is-what-it-has-issued.md), [#86](https://github.com/winniel123/verge-asm/issues/86). A targeted re-fetch over a named finite list, currently of one |
| **G11** | For every footing cell, the owner's **current release tag** against the tag the cell was read at; a difference marks the cell **stale-against-tag** | trigger 2's closed half, and the queue's tie-break |
| **G12** | Every cell in G2's domain whose carrying artefact sits at rung 1 or rung 2 (no retrievable tag): a targeted re-fetch, diffing the quoted sentence the cell rests on against the sentence on record; a sentence removed or materially weakened marks the cell **demoted-untagged** | new — [#149](https://github.com/winniel123/verge-asm/issues/149), `sensitive-ports.md` §47 |
| **G13** | For every graded cell whose held ground quotes an owner-declared horizon: compare the owner's current tag/date (already fetched by G11) against that horizon; where current ≥ horizon and nothing records whether the promised change occurred, mark the cell **horizon-passed-unverified** | new — [#152](https://github.com/winniel123/verge-asm/issues/152), [ADR-0099](./0099-a-stated-horizon-is-a-second-comparand-a-tag-match-does-not-discharge.md), `sensitive-ports.md` §49 |

**G11 is where *evidence age* lands, and it is the whole answer to what age is measured on.** *This
footing was read at `v1.34.1` and the owner is at `v1.37.0`* is a fact about the subject in the
subject's own units and is mechanically checkable. *This footing is fourteen months old* is a fact
about **us** and says nothing — a specification thirty years old is fresher than a documentation-branch
line from last month, and every rung above says why.

### G12, ruled by [#149](https://github.com/winniel123/verge-asm/issues/149)

[ADR-0077](./0077-a-second-ground-counts-only-where-it-would-have-carried-the-cells-proposition-alone.md)
ruled that a tier demotion is not a de-attestation and routed the hazard to **the gate**, then recorded
the gate's own shortfall rather than repairing it: *"a tier demotion on an artefact with no retrievable
tag is caught by neither G2 nor G11."* [#149] confirms the shortfall is real (walked systematically
against every one of the eleven checks above — none reaches it. G8's *"still a token … at the tag
named"* and G11's tag-diff both presuppose a tag, G6 only greps bytes already held, G10's re-fetch tests
a different boolean) and rules that **no existing check's population can be widened to close it** —
unlike G2's domain (#154) or G8's population (§21 of `sensitive-ports.md`), this needs a new **test**,
not a wider population of an old one. **The population is finite and named, so the check terminates**:

| Check | Population | Test | Verdict |
| --- | --- | --- | --- |
| **G12** | Every cell in **G2's domain** (the two graded footing tiers) whose carrying artefact sits at **rung 1 or rung 2** — no retrievable tag. Currently: `10250/tcp`'s and `10255/tcp`'s footing cells, both carried by `security-checklist.md` (`sensitive-ports.md` §41.3) | **Targeted re-fetch**, over the named population only: pull the artefact's current bytes and diff the quoted sentence the cell rests on against the sentence on record | A sentence removed or materially weakened, such that the cell would fall to a lower tier under the criterion **as it currently stands** (G2's own test), marks the cell **demoted-untagged** |

**This is a different check from #152's**, not the same one under a different name. #152's gap is a
comparison of two facts **already on record** — the owner's current tag/date (which G11 already tracks)
against a horizon the artefact **itself states** — and needs no retrieval at all. This check's gap is
that **no tag exists to compare**, so the only instrument that can reach it is one that re-reads the
artefact's current bytes. Different inputs, different termination shape, both real. The merge session
numbered this check **G12** and #152's **G13** — lower ticket number first, no other significance to the
order.

**The gate now runs thirteen checks.** Merged into the table above as **G12**.

## Rationale

### 1. The queue exists because the reading budget is finite, and that is why the axis question was unanswerable

Nobody could propose a measurement for the five axes because nobody had said what the list is **for**.
Once the list is *the order in which a finite reading budget is spent*, the axis is forced: the list
must key on **the probability that a ground has moved with nothing saying so, per unit of reading**.
That is volatility, and volatility is a property of the artefact.

The five axes then stop competing, because they were being asked to do one job when they do four
different ones. Tier grades distance and belongs to the disclosure. Age bounds currency and belongs to
the gate. Support count decides *whether* a cell is exposed and belongs to the filter. Contradiction by
the owner's own product documentation reports a row that is **already wrong** and belongs to the gate.
**Exactly one of the five is about how easily the ground moves, and it is artefact class — read as the
revision act rather than as a prestige ranking.**

### 2. Tier and volatility are orthogonal, and the corpus has measured the disagreement

ADR-0059 fixed what a footing tier grades: **evidential distance**, counted in premises the reader
supplies, *"never the owner's conviction"*. Nothing in that quantity is about how easily the sentence
can be edited. **[measured]**, from cells already in the note:

| Pair | Footing tier | Rung |
| --- | --- | --- |
| `10248/tcp` | **weak** | **1** — a config-API doc comment |
| `445/tcp` (+ `139/tcp`, `137/udp`, `138/udp`) | **prohibition** | **2** — one continuously-published page, and §13.1 measures **no configuration artefact behind it** |
| `623/udp` | **prohibition** | **2** — vendor default-value documentation, **and it has already been de-attested once** (§36.7) |
| `2379`, `2380/tcp` | **prohibition** | **2** — a `THREAT_MODEL.md` three months old and absent from two live release lines (§15.9) |
| `5432/tcp` | **weak** | **3** — `listen_addresses` in a shipped sample, moving only on a major release |

**The top tier holds the more volatile artefacts here and the weak tier holds the more stable one.**
A list keyed on tier therefore spends the budget in close to the wrong order, which is the strongest
possible form of the complaint that its count has been useless: the count was measuring the wrong
thing, so of course it moved without meaning anything.

### 3. Support count is the filter, and it explains a correction the corpus made for the wrong stated reason

§17.6 already links the two halves: *"Every row in it rests on a negative … so every one is
**sole-ground** and every one is **exposed**."* ADR-0046 limb 1 says only a **sole-ground** negative is
exposed. The same arithmetic on the positive side says a cell carried by two **independent** artefacts
cannot be silently de-attested by one act, because two acts must occur.

That disposes of #88's open worry cleanly. `10255/tcp` was taken **off** the list because it joined the
prohibition tier — a statement about tier. The reason it actually stopped being exposed is that it
acquired a **second** ground: if the checklist line goes, the row falls back to `readOnlyPort: 0` and
**demotes a tier rather than losing a cell**. #88's rider — *"both kubelet cells are now exposed to its
mirror"* — is therefore **answered and refused**: exposure to a demotion is not exposure to
de-attestation, and this queue is a de-attestation queue. Tier was taking the credit for work support
count was doing.

> **Confirmed and re-sited at the cell by
> [ADR-0077](./0077-a-second-ground-counts-only-where-it-would-have-carried-the-cells-proposition-alone.md)**
> ([#135](https://github.com/winniel123/verge-asm/issues/135)). **The paragraph above is correct about
> `10255`'s FOOTING cell and is not a disposal of the ROW**, this ADR's own unit being the cell.
> §18.7 measures both fallbacks — `10250` to the scoping tier, `10255` to the weak tier — so in both
> cases the second artefact yields the same **proposition** at a lower tier and the act demotes.
> `sensitive-ports.md` §39.4 nonetheless recorded *both rows* as *not an item*, and that is superseded:
> **[measured]** §19.12 records `10250`'s Claim 3 as having *"no second support on this row"*, so
> `10250`'s **claim** cell is sole-ground and is a queue item. #88's rider stays refused for the two
> footing cells and is **live** for `10255`'s claim cell, which no section has tested.

### 4. The unit is a triple, and that is the repair for five useless counts

When `10255` left and `10248` joined, **two items changed and the count did not move.** A count over
rows cannot show that. A list whose members are `(cell, artefact, revision act)` triples can: a row may
be an item twice on two cells, and **four pairs are one item** where one Microsoft sentence carries
`445`, `139`, `137` and `138` with no configuration artefact behind any of them. That concentration is
invisible on a row list and is the single largest exposure in the table.

This is [ADR-0010](./0010-exposure-composes-two-reaches.md)'s lesson in a new place: *a state enum is a
projection, not the thing.* A count over a watch list is a projection that discards exactly the
coordinate that moves.

### 5. A withdrawal that supplies no replacement does not hold

ADR-0038 withdrew the identity and named no successor. **[measured]** the identity was then asserted as
a reason or a rule at **19 sites**, including inside this ADR's own §8 **below** the withdrawal box,
in a box demonstrably edited afterwards.

ADR-0058's repair is at the **sentence**, and #106 established that a document supersedes itself. #125
adds the other half of the diagnosis: **a sentence that names no successor is re-derived by the next
session that needs one.** Withdrawing *watch list = weak tier* left every subsequent pass with a watch
to update and only one enumeration to update it with, so each pass reached for the weak tier again and
wrote the identity back. **The repair for a regenerating clause is a replacement, not a stronger
strike-through** — and the sweep of the 19 sites is worth nothing without one, which is why #125 does
both in one pass.

### 6. Who revises: the office is an act, because a person has already failed three times

The alternative — *a named curator, with a standing duty to watch* — is what the corpus has been doing
implicitly, and it is measurably not working. `9042/tcp` sat on the watch list after leaving the weak
tier at #69, and *"nobody propagated the change"* (§16.8). Kubernetes' `security-checklist.md` has been
unmodified since **2025-02-28** while the pairs it carries moved twice (§37.11 item 5) and nobody was
watching it. The identity regenerated at 19 sites. Each is the failure of a standing duty, and a
standing duty has no artefact — so its absence is indistinguishable from its discharge, which is
exactly what [#57](https://github.com/winniel123/verge-asm/issues/57) refused for outbound acts.

**A gate on the edit has an artefact by construction**: it either ran over the post-edit state or it
did not, and its output is a record. It also fires where the damage has actually been measured — the
merge notes record that the 2026-08-15 batch's *"only defect was ADR-0009's #114 amendment failing to
discharge its own #109 figures further up the same file"*, which is a defect **of an edit**.

### 7. A machine may raise; only a release may rule

Every gate check is mechanical enough to automate, and the temptation is to let the automation act.
It may not. Moving a row **authors a claim**, and §2.2's first sentence is *the claim may not be
asserted by us*. ADR-0037 limb 2 adds that a row moves only on a retrieval **scoped to the row**. A
sweep — human or not — may **route** and never rule. So the watch's output is a **question**: *this
cell's artefact has moved past the tag it was read at; is the cell still true?* The answer is a
retrieval, and the retrieval is a ticket.

## Consequences

- **ADR-0032 §8's watch list is superseded as an instrument.** The three-row enumeration is a correct
  record of a *weak tier* and is no longer the watch. §8's `#95` amendment box is marked at the
  sentence. The fifteen sites in `sensitive-ports.md` are marked at their clauses (§39.7).
- **The queue as of evidence already held is ~~eight items over ten `(port, transport)` pairs~~ nine
  items over eleven pairs and two non-port cells**, against a three-row list — and ~~two~~ **three of
  the pairs it adds are in the top footing tier**. Enumerated with grounds at §39.4 and §41.4. **Moved
  by [#135](https://github.com/winniel123/verge-asm/issues/135) /
  [ADR-0077](./0077-a-second-ground-counts-only-where-it-would-have-carried-the-cells-proposition-alone.md)**,
  which added `10250/tcp`'s **claim** cell — sole-ground on `ports-and-protocols.md`'s
  `Used By: Self, Control plane` (§19.12) — after fixing the filter's bar and applying it per cell. It
  is still **provisional**: the independence test has been run only over the cells the corpus has
  already measured, and the full pass over all 38 pairs plus the frequency half is
  [#134](https://github.com/winniel123/verge-asm/issues/134).
  > **SUPERSEDED as a statement of the queue's membership by [#134](https://github.com/winniel123/verge-asm/issues/134)**,
  > marked here per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md).
  > **The provisionality is discharged**: the per-cell test has now been run over every cell of all 38
  > pairs and both non-port cells. **The live register is [`sensitive-ports.md`](../research/sensitive-ports.md)
  > §43.3 and it is stated over MEMBERS.** The replacement supplied here is a **pointer to a set**, not a
  > new number — §39.2 bars the queue's count as an indicator and
  > [ADR-0078](./0078-a-residue-is-disclosed-by-the-act-that-leaves-it.md) bars it from the residue
  > entry, so a figure is the one thing this clause may not be replaced with. Every item enumerated above
  > stands with its ground and its rung unchanged; the register **adds** cells and removes none, the bar
  > being one-way. Where a cell's proposition is carried by a **set** of artefacts no one of which yields
  > it alone, the item's artefact coordinate holds the set and the item enters at the most volatile
  > carrier's rung —
  > [ADR-0076](./0076-a-conjunctively-carried-cell-is-one-item-entered-at-the-rung-of-its-most-volatile-carrier.md).
- **The queue is a superset of the weak tier, so the ruling removes no row from anybody's attention.**
  All three weak-tier rows stay on it. That makes the change cheap in the only direction that could
  cost something.
- **`certificate-weak-key-or-signature`'s table contributes zero items.** **[measured]**
  [`weak-key-and-signature.md`](../research/weak-key-and-signature.md) §9.3: its residue is a **scope**
  weakness and *"no shipped default is involved, so silent de-attestation cannot reach them"*. Its
  cure-availability item — SP 800-131A Rev. 3 going final — sits at rung 5.
- **`certificate-expiring`'s horizon contributes one cell, not none.** ADR-0038 scoped ADR-0034's
  *"nothing to watch"*: the fraction removes the **quantity**, never the **attestation**.
- **`verge-core`'s frequency half is the queue's other rung-1 item**, and it is a **cure-availability**
  item rather than a de-attestation one — nothing can de-attest data already stale by eighteen years.
  The queue holds both kinds and orders them on the same key, which is what discharges
  `project-authored-constants.md` §9's hand-off.
- **No row, class, tier or coverage figure moves.** The list stays at **38 pairs**, classes
  `12 / 7 / 19`, tiers `13 / 11 / 3`, coverage **27 of 38**. `verge-core` stays at **136**. **§4.6 goes
  from 23 to 24 entries** on `9443/tcp` (§39.5). [ADR-0008](./0008-derivation-versions-move-on-content.md)
  is **not triggered** — `sensitive-port-reached-from-internet` is byte-identical and a governance
  instrument is not reference data.
- **ADR-0045 gains a rider**, at §39.6: an owner's **unrendered** bytes are not issued, so a retrieval
  that reads raw source must state whether the string it quotes is rendered. G7 is the check.
- **Nothing reaches the interface.** ADR-0032 §7 is untouched and its condition is unchanged: the weak
  tier earns a screen only when the operator acquires a sanctioned way to disagree with a row.
- **`CONTEXT.md` is not amended**, and that is a ruling rather than an omission — see the Decision's
  last row.

## Alternatives rejected

| Option | Why it lost |
| --- | --- |
| **Keep the weak tier as the watch list** | It is the incumbent and it is the option that lost. It keys on **evidential distance** where the hazard is **artefact volatility**; **[measured]** the two disagree on at least four pairs and the top tier holds the more volatile artefacts. Five consecutive membership changes invisible to its own count are the symptom, and `623/udp`'s already-executed de-attestation on a top-tier row is the refutation |
| **A named curator with a standing duty to watch** | The status quo by default, and it has failed three measured times with no artefact to show for any of them. A duty whose discharge and whose absence look identical is what #57 refused |
| **Block the release on a red gate** | Disproportionate, and it would not be honoured by a one-person project — so it would decay into decoration, which is exactly what §7's *permanent caveat* rule forbids. Gating the **edit** is proportionate, self-enforcing, and fires where the damage was measured |
| **A quantum — *N* queue items per release** | It manufactures a number nobody can attest (ADR-0034), and it restores meaning to the queue's **length**, which is the failure being repaired. The obligation is the **bounded residue**, not the quantity |
| **Evidence age as the key, measured in calendar days** | Our units, not the subject's — ADR-0043's refusal exactly. It ranks a thirty-year-old RFC below a documentation line from last month, which is backwards on every rung |
| **A sixth axis** | Ruled out by [#110](https://github.com/winniel123/verge-asm/issues/110) and **not re-added**: a disposal whose supporting artefact goes stale while its verdict stands is **citation staleness** (G8), not de-attestation. Recorded so nobody adds it again |
| **A fourth pile beside *watched*, *chased* and *scope*** | ADR-0038 refused it and this ADR does not reopen it. The piles are causes; the queue keys on the act. Adding a pile builds the machinery the key removes |
| **Let the automated sweep move rows** | It would author a claim, which §2.2 bars us from doing, and it would move a row on a retrieval not scoped to the row, which ADR-0037 limb 2 bars. A machine raises; a release rules |
| **Put the watch on a screen** | ADR-0032 §7's four reasons stand unchanged, and the queue is *more* clearly the curator's than the weak tier was: it names artefacts and commits, which the operator can act on even less than a tier |

---

## Amendment — [#136](https://github.com/winniel123/verge-asm/issues/136): the residue disclosure is sited, and the *lives in each instrument's own document* clause does not site it

**Status: this ADR is confirmed, not corrected.** Every ruling above stands. One clause is
**distinguished**, and it is written here rather than only in the new ADR because this ADR's own §5
measured what happens otherwise: *a sentence that names no successor is re-derived by the next session
that needs one*, nineteen times over.

**The clause.** The Decision's last row — *"Where this lives … It lives in this ADR and in each
instrument's own document — ADR-0032 §7 unchanged"* — sites **the watch**. It does **not** site the
**bounded-residue disclosure** that the *"What a release owes the queue"* row makes the release owe,
and a session reading it alone would put a residue statement in each curated table's own note. That
reading is refused.

**[ADR-0078](./0078-a-residue-is-disclosed-by-the-act-that-leaves-it.md)
([#136](https://github.com/winniel123/verge-asm/issues/136), 2026-08-15) sites it**, and the reason it
cannot sit in a table's document is this ADR's own ruling: **the three cause-piles land on one queue
because the revision act, not the cause, fixes priority.** One queue is **one order across four
curated tables**, and *how far down it read* is a fact about the **order**. **[measured]**
[`sensitive-ports.md`](../research/sensitive-ports.md) §39.4 carries two cells that are not port cells
— `verge-core`'s frequency half and `certificate-expiring`'s fraction — interleaved by **rung** among
the port cells, the frequency half at **rung 1** sorting above every rung-2 and rung-3 port item. And
`weak-key-and-signature.md` contributes **zero**, so a per-table statement there would be a standing
statement of nothing, which ADR-0032 §7 forbids as a permanent caveat.

| Concern | Where #136 lands |
| --- | --- |
| **Which document** | [`docs/spec/curated-table-watch.md`](../spec/curated-table-watch.md) — the release's account of its curated tables, holding the register's siting, the **residue ledger**, and a reserved place for the gate's record |
| **What form** | **One dated entry per release, appended and never rewritten.** Not a standing section — a residue describes a **spend**, and restated in the present tense one release later it is false. Not gate-emitted — the gate fires on the **edit**, attests what it **checked** rather than what a person **read**, and *is* the instrument of what terminates |
| **How it states its bound** | **Extensively and intensively.** The items **not read** are **named**, because the register is ours and enumerable and a described boundary over an enumerable set is a count with the number filed off. Per item **read**, the artefacts opened and the **class boundary** of that opening are stated, because reading somebody else's corpus does not terminate |
| **What the gate contributes** | **G11's `stale-against-tag` marks**, cited per unread item as the reason it was passed over. *A terminating instrument may supply inputs to a disclosure about what does not terminate, and may never sign it* |
| **The quantum** | **Still refused**, and #136 does not reopen it. An entry with an empty head is fully compliant and says why the head is empty |
| **The count** | **Still barred.** No length is quoted in an entry — not the register's, not the head's, not the residue's |
| **ADR-0032 §7** | **Unchanged and unreopened**, and marked at its own site with the same distinction. The watch document is a curator's document reaching no interface, and by §7's fourth ground it is **not** an operator-facing release note either |

**Nothing in this ADR's queue, gate, rungs, filter or unit moves**, and no row, class, tier or coverage
figure moves with it. #136's ruling is stated over *the register as of the release's tag*, whatever it
contains — and that independence has now been **tested rather than asserted**:
[#135](https://github.com/winniel123/verge-asm/issues/135) tightened the sole-ground filter and moved
the register's membership while #136 was open, **without touching a word of the entry form**.
[#134](https://github.com/winniel123/verge-asm/issues/134)'s per-cell walk will move it again. **The
tightening is one-way** — the filter narrows what counts as a ground, so cells can only be added — so
every argument #136 makes over the register's **members** survives its growth, and any argument made
over its **length** would already have failed. **No figure for the register's size appears in ADR-0078
or in the watch document**, by construction and for this reason.
