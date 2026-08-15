# ADR-0076: A conjunctively-carried cell is one queue item, entered at the rung of its most volatile carrier

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#134 Walk the queue's sole-ground filter per cell over all 38 pairs plus the frequency half](https://github.com/winniel123/verge-asm/issues/134)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0057](./0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md) built the de-attestation queue
on a **`(cell, artefact, revision act)`** triple and keyed it on the **revision act** — *the smallest
act by the owner that would falsify the cell*.
[ADR-0077](./0077-a-second-ground-counts-only-where-it-would-have-carried-the-cells-proposition-alone.md)
fixed its filter: a second artefact removes a cell only where it would have carried **that cell's
proposition** standing alone, tested per **cell** and never per row.
[`sensitive-ports.md`](../research/sensitive-ports.md) §41.2 states the test in four steps, and every
one of them is written on a cell **C** carried by **an** artefact **A**.

**[#134](https://github.com/winniel123/verge-asm/issues/134)'s walk is the first pass to run that test
over every cell of the composed table rather than over the cells already measured, and it met a shape
the test does not describe.** A cell's proposition can be carried by a **set** of artefacts, no one of
which yields it standing alone. **[measured]**, at four independent sites in the corpus:

- **Claim 1 is two steps answered from two artefacts.** §10.1's walk answers Step 1 — *is publication
  the purpose?* — from the owner's prose, and Step 2 — *what authority does the anonymous caller get?* —
  from the shipped dispatch. §25.2 read ten of those Step 2 cells *"off the dispatch at a named tag"*
  while their Step 1 cells sit in documentation. Neither artefact yields *Claim 1 holds for this pair*.
- **§24.6 says so in its own title.** `10259/tcp` and `10257/tcp` are *"admitted on Claim 3, and **not
  on the table cell alone**"*: the cell is the owner's ports table **plus** a restricting loopback
  default in the owner's own installer.
- **Claim 2 is a conjunction by construction.** §2.1: *"Credentials or session content in cleartext,
  **with a standardised encrypted successor reachable on a different port**. The successor clause
  matters."*
- **§16.4's withdrawn placement had the same shape**, the position and the port-binding coming from two
  parts of one page rather than one sentence.

ADR-0077's **step 3** already disposes of the verdict: *"if every remaining artefact yields a different
proposition — a different cell, **a fragment of the cell**, or nothing — then C is a queue item."* Each
carrier of a conjunction yields a fragment, so the cell **is** an item. **What step 3 does not say is
how many items, or at which rung** — and both matter, because the queue is not a set of alarms. It is
*"the order in which a finite reading budget is spent"* (§39.3), and ADR-0057 gives support count
exactly one job: *"it removes items rather than ordering them."*

Read literally, the triple's key makes a two-carrier cell **two** triples, so the same cell appears
twice in one reading order at two rungs. A list that contains one thing twice is not an order, and
[ADR-0078](./0078-a-residue-is-disclosed-by-the-act-that-leaves-it.md)'s residue entry names the
extensive residue **member by member** — so a duplicated member is a disclosure defect and not merely an
inelegance.

## Decision

| Concern | Decision |
| --- | --- |
| **How many items is a conjunctively-carried cell?** | **One.** The register's members are **cells**; the artefact coordinate of the triple takes the **set** of carriers, and the item names every one of them |
| **At which rung does it enter?** | **At the rung of its MOST VOLATILE carrier** — the lowest-numbered rung among them. Where every carrier is load-bearing, **any** carrier's act falsifies the cell, so *the smallest act that would falsify it* (ADR-0057's key) is the act that moves the weakest link |
| **What is a *carrier*?** | An artefact **without which the cell's proposition does not follow** from the artefacts this note cites for the row. An artefact that is redundant with another — either would do alone — is a **second ground** under ADR-0077 and the cell is not an item at all |
| **Does this widen the filter?** | **No.** ADR-0077 decides *whether* a cell is an item and is untouched. This decides *where the item sits and how many times it appears*, which ADR-0077 does not reach |
| **Does a carrier's own rung matter after entry?** | Only as the **tie-break**, unchanged — §39.3's *how far the owner has moved past the tag the cell was read at*, read on the carrier that fixed the rung |
| **What does the item disclose?** | **Every carrier, named**, so that a reader can see which act the rung was taken from. The residue entry's intensive bound (ADR-0078) is then stated per **carrier opened**, not per item |
| **The count** | **Barred, as everywhere.** §39.2 and ADR-0078 §42.6. This ADR reduces a duplication; it does not license quoting what it reduces |

### Worked, from measurements already in the corpus

| Cell | Carriers | Rungs | Enters at |
| --- | --- | --- | --- |
| `10249/tcp`'s **claim** | Kubernetes' metrics documentation (Step 1, §27.3) · `serveMetrics` at `v1.34.0` (Step 2, §27.2) | 2 · 3 | **rung 2** |
| `2181/tcp`'s **claim** | the ZooKeeper Administrator's Guide (Step 1) · `apache/zookeeper` `release-3.9.5` shipped bytes (Step 2, §25.2) | 4 · 3 | **rung 3** |
| `10259`/`10257`'s **claim** | `ports-and-protocols.md`'s `Used By: **Self**` · kubeadm's `--bind-address=127.0.0.1` (§24.6) | 2 · 3 | **rung 2** |

**And the case it is NOT.** `27017`/`27018`/`27019`'s footing is carried by mongodb.com's *Security
hardening* **and** by `bindIp: 127.0.0.1` in MongoDB Inc's own packaging, and **either would do alone**
(§13.2). That is redundancy, not conjunction: ADR-0077's step 2 fires, the cell is **not an item**, and
this ADR never runs.

## Rationale

### 1. The weakest link is what ADR-0057's key already says

ADR-0057 did not key on *the artefact*; it keyed on *the smallest act by the owner that would falsify
the cell, and whether that act publishes a notice we read*. For a conjunction, the set of falsifying
acts is the **union** over the carriers, and the smallest member of a union is the smallest member of
any of its parts. **The rule is therefore a reading of ADR-0057 rather than an addition to it** — which
is the same relationship ADR-0077 bears to ADR-0057's *carries the same cell*, and it is the second time
in two tickets that applying the stated key to the object it names has settled a question a pass had
been resolving by intuition.

### 2. Listing the cell at every carrier's rung was the option that lost, and it lost on the order

The alternative is faithful to the triple's key: a cell with two carriers is two triples, so list it
twice. It is refused on three measurements.

**It destroys the order.** §39.3 fixes the queue as a **reading order** over a finite budget. A reader
who works down it meets `2181`'s claim cell at rung 3 and again at rung 4 and has no instruction about
what to do the second time — and reading it the first time discharged both, because the reading is of a
**cell**.

**It corrupts the residue disclosure.** ADR-0078 names the extensive residue **member by member**, and
its falsification test is *"naming one register item that appears in neither the head nor the residue"*.
A cell present at two rungs is in the head **and** in the residue after one reading, which makes the test
return a false positive on a correctly-written entry.

**It restores meaning to a count through the door left open.** §42.4 measured this failure mode once
already: *"the count returns through whichever door is left machine-operable."* Duplicating members
makes the register's length a function of how many artefacts a cell happens to cite, which is the one
quantity §39.2 bars and the one ADR-0034 calls manufactured.

### 3. Why not the *least* volatile carrier

The mirror rule — enter at the **highest**-numbered rung, on the ground that a conjunction is only as
exposed as its sturdiest support — is coherent and backwards. A conjunction falls when **any** conjunct
falls, so its exposure is that of its weakest support, not its strongest. Taking the least volatile
carrier would sort `10259`/`10257`'s claim cell below `5432`'s footing on the strength of a shipped
default, while the cell's actual exposure is a documentation-branch commit in `kubernetes/website` that
publishes no release note. **That is the queue spending its budget close to backwards**, which is exactly
the defect §39.3 measured in the footing tier and built the rung ladder to repair.

### 4. Why the carriers are named rather than summarised

ADR-0057's watch output is *"a question — this cell's artefact has moved past the tag it was read at; is
the cell still true?"* A reader cannot ask that question of an item whose artefact coordinate reads *"an
owner sentence and a shipped default"*. Naming every carrier is also what makes the rung **falsifiable**:
the item can be refuted by naming a carrier at a lower rung than the one the item entered at, which is
ADR-0040's falsification discipline applied to the register's own form.

### 5. This is a decision the run forced, and it is not the run

[#133](https://github.com/winniel123/verge-asm/issues/133) minted no ADR because *"every rule it applied
already existed"* — a run of an instrument rather than a decision about one. #134 applied ADR-0077 to
every cell and found a class of cell the instrument's stated test does not describe. **The register is
#134's deliverable and it cannot be stated without this rule**: three downstream tickets transcribe the
register member by member, and *how many members a conjunctive cell contributes, and where each sits* is
the difference between a register and a list. The rule is minted rather than folded into the walk so that
the walk's output can be re-derived by someone who disagrees with it.

## Consequences

- **The register at `sensitive-ports.md` §43.3 is stated with this rule applied**, and every
  conjunctively-carried item names its carriers and the carrier its rung was taken from. Each is marked
  `ADR-0076` at its ground, and **the shape is stated over its members rather than counted** — §39.2's
  bar reaches a count of the register's parts as much as a count of the register.
- **ADR-0077 is confirmed by use and is not amended.** Its four-step test decides membership; this rule
  decides form and position, which its Decision table does not reach. §41.2 is unchanged.
- **ADR-0057 is confirmed by use and is not amended.** Its unit stays the `(cell, artefact, revision
  act)` triple; what is fixed is that the artefact coordinate may hold a **set**, which its own
  four-pairs-one-sentence measurement had already required in the other direction.
- **ADR-0078's entry form is unchanged and its intensive bound gains a referent.** For a
  conjunctively-carried item, the bound is stated **per carrier opened** — the class boundary of each
  opening — which is strictly more falsifiable than one bound per item and needs no change to the form
  at [`curated-table-watch.md`](../spec/curated-table-watch.md) §2.1.
- **No row, class, footing tier or coverage figure moves**, and
  [ADR-0008](./0008-derivation-versions-move-on-content.md) is **not triggered** — a governance
  instrument is not reference data and `sensitive-port-reached-from-internet` is byte-identical.
- **`CONTEXT.md` is not amended**, on ADR-0057's own last Decision row: the curator is not a subject in
  the model and the product holds nothing about it.
- **No count is introduced anywhere.** The rule reduces a duplication that would have made the
  register's length depend on citation habits; it does not make the length quotable. §39.2 and ADR-0078
  §42.6 are untouched.
- **[#151](https://github.com/winniel123/verge-asm/issues/151) is not pre-empted.** Whether step 2 counts
  adequate **artefacts** or falsifying **acts** decides *whether* a cell is an item; this rule runs only
  after that question is answered, on cells that are items under either answer. `sensitive-ports.md`
  §43.5 names every cell whose membership the two readings would move.

## Alternatives rejected

| Option | Why it lost |
| --- | --- |
| **List the cell once per carrier — the incumbent read of the triple's key** | **The option that lost**, and it is faithful to ADR-0057's key read literally. It destroys the queue's character as an **order** (§39.3), it makes ADR-0078's *name one item in neither head nor residue* test return false positives on a correct entry, and it makes the register's length a function of citation habits, which is §42.4's *the count returns through whichever door is left machine-operable* |
| **Enter at the *least* volatile carrier's rung** | Backwards on the queue's own axis. A conjunction falls when any conjunct falls, so it is exposed at its **weakest** support. This reading sorts a cell exposed to an unannounced documentation commit below one exposed to a versioned release, which is §39.3's measured defect re-imported |
| **Enter at the carrier the note cites first** | Not a property of the world. It makes the reading order a function of how a section happened to be written, which is the same class of error as keying the watch on the footing tier — a fact about **us** standing in for a fact about the owner (ADR-0059) |
| **Split the conjunction into one cell per conjunct** | Attractive, and it is a **claim-gate** change wearing a governance costume: it would make *Claim 2's cleartext half* and *Claim 2's successor half* separate cells of the table, which is §2.1's closed claim set being re-cut by a filter. ADR-0077 refused the queue deciding its own criterion mid-pass; a queue re-cutting the table's cells is the same error one level up |
| **Rule it out of scope — let the walk state the cell however it finds it** | Refused for #134's own reason. Three tickets transcribe this register **member by member**; a register whose members are ambiguous in number and position is a list. ADR-0057's rationale 5 applies directly — *a sentence that names no successor is re-derived by the next session that needs one* — and *how many items is a two-carrier cell?* would be re-derived by every one of the three |
| **Fold the rule into `sensitive-ports.md` §43 without an ADR** | It is a rule about the **instrument**, not about the table, and §43 is a **run**. A rule stated only inside the output of one run is invisible to the next run, which is the shape §39.7 measured at nineteen sites |
