# The golden corpus: the membership pin block, and the CI matrix that runs it

- **Status:** Accepted — spec content for [#12](https://github.com/winniel123/verge-asm/issues/12)
- **Ticket:** [#143 `resolution-walk`'s golden corpus owes rows pinning the membership-deciding outcomes](https://github.com/winniel123/verge-asm/issues/143). §8 by [#146 Is `wildcard-discrimination` in the membership vector?](https://github.com/winniel123/verge-asm/issues/146). §9 by [#177 Specify ADR-0021's uncovered move — form, home, and what counts as one](https://github.com/winniel123/verge-asm/issues/177). §10 by [#986 The `Custody` golden corpus and its A6 gate](https://github.com/winniel123/verge-asm/issues/986)
- **Rulings:** [ADR-0085](../adr/0085-an-obligation-with-no-failing-test-has-no-owner-and-a-boundary-needs-a-row-on-each-side.md) (this file's rules), [ADR-0086](../adr/0086-membership-composes-every-leaf-that-decides-the-value-it-reads.md) (the membership vector, and §8), [ADR-0021](../adr/0021-a-version-leaf-is-a-decision-not-a-binary.md) (the corpus, the gate, and — by its [#177 annotation](../adr/0021-a-version-leaf-is-a-decision-not-a-binary.md#annotation--177-2026-08-15-the-uncovered-move-is-given-form) — §9's register), [ADR-0041](../adr/0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md) (the obligation), [`packaging-and-configuration.md`](./packaging-and-configuration.md) §1 (the architecture rule)

This is a separate file from the ADR that rules it. It follows
[`measurement-offers.md`](./measurement-offers.md)'s and
[`packaging-and-configuration.md`](./packaging-and-configuration.md)'s precedent: **the tables below
will be revised and an ADR is a decision that will not.** §2's and §8's enumerations in particular
move whenever a leaf, a declared parameter or a boundary moves. **§9's register grows by one row
every time a leaf bumps on an uncovered move** ([ADR-0021](../adr/0021-a-version-leaf-is-a-decision-not-a-binary.md)'s
[#177 annotation](../adr/0021-a-version-leaf-is-a-decision-not-a-binary.md#annotation--177-2026-08-15-the-uncovered-move-is-given-form)
rules its form). ~~§7's escrow is written to be
discarded or adopted whole~~ — **the escrow is spent**: [#146](https://github.com/winniel123/verge-asm/issues/146) ·
[ADR-0086](../adr/0086-membership-composes-every-leaf-that-decides-the-value-it-reads.md) **adopted it
whole**, and §7 is now a pointer to §8 rather than a holding pen. Struck at the site that specifies
it ([ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)):
read alone and in the present tense it sends a session to answer a routing question that has been
answered.

Nothing here is a test file or a workflow file. The map's *plan, don't do* rule holds: this
specifies which cells must hold a row, which legs must run, and in what order the assertions fail.
An implementation session writes them.

---

## 0. What this file is for, in one paragraph

[ADR-0041](../adr/0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md) found three things:

- A dependency upgrade can bump a version leaf that the **membership** timeline composes.
- A `Break` between a subject's withdrawal and its return destroys `returned`.
- The repair is a golden corpus dense enough that a no-op upgrade **provably** does not bump the leaf.

It named the
obligation and gave it nobody to fail on. This file is the owner. It is a checked-in enumeration of
**46 membership cells** (§2 and §8), plus the matrix the rows run on. Each cell must hold a corpus
row, counted by CI.

**§10 is a third block against a different obligation.** Its **6 cells** pin the `Custody`
derivation, which is not a membership leaf and composes into no `Span`. It is here because the gate
and the matrix are here, and it discharges [ADR-0008](../adr/0008-derivation-versions-move-on-content.md)'s
rule — *a declared parameter is pinned by a corpus row whose output the value decides* — rather than
ADR-0041's. Read the two obligations apart: **46 + 6 is not a membership count.**

~~a checked-in enumeration of **27 cells** ... and **17 further cells held in escrow** against a
routing question this ticket may not answer (§7)~~ is **superseded here, at the site that specifies
it** ([ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)).
The routing question is answered — [#146](https://github.com/winniel123/verge-asm/issues/146) ·
[ADR-0086](../adr/0086-membership-composes-every-leaf-that-decides-the-value-it-reads.md) rules that
**`wildcard-discrimination` is in the membership vector**. The block is now **two** blocks against
**two** leaves: **27 cells** for `resolution-walk` (§2) and **19** for `wildcard-discrimination`
(§8 — the escrow's 17, adopted whole, plus one boundary pair ADR-0086 adds). **Escrow: none.**

The failure it prevents is estate-wide, silent, and triggered by a no-op upgrade. That is why the
obligation is data rather than prose.

---

## 1. Which leaf this file discharges

[ADR-0021](../adr/0021-a-version-leaf-is-a-decision-not-a-binary.md) names five version leaves inside
the measurement binary. ~~**This file discharges the obligation for `resolution-walk` and no other
leaf.**~~ **Superseded here, at the site that specifies it**
([ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)): this
file discharges the obligation for **the two leaves that decide the `resolution` value membership
reads**. Those are `resolution-walk` in §2 and `wildcard-discrimination` in §8
([ADR-0086](../adr/0086-membership-composes-every-leaf-that-decides-the-value-it-reads.md)). It
discharges no other leaf.

| Leaf | Discharged here? | Why |
| --- | --- | --- |
| `resolution-walk` | **Yes — §2** | A `Name` leaves on `NameError` from every available vantage; a cited `Address` is in the estate exactly while a current resolution cites it. Both are this leaf's outputs, and it is the leaf ADR-0041 names |
| `wildcard-discrimination` | **Yes — §8** | ~~**No — routed, see §7**~~. `Shadowed` is ADR-0041's fifth outcome and is **this** leaf's. [#146](https://github.com/winniel123/verge-asm/issues/146) · [ADR-0086](../adr/0086-membership-composes-every-leaf-that-decides-the-value-it-reads.md) rules that it **is** in the membership vector: `Shadowed` cites no `Address`, so the verdict decides whether an answer's address set enters the estate |
| `connect-outcome` | No | A port opening is a `Reach` move and never a membership event; a `Service`'s membership is its `Address`'s restated ([ADR-0031](../adr/0031-membership-alerts-at-the-root-of-the-entering-subtree.md)) |
| `tls-handshake` | No | Feeds `certificate` and `tls-acceptance` |
| `http-exchange` | No | Feeds `http-identity` only |

The `Seed`-covered `Address` composes **nothing at all** — a `Seed` is Declared and carries no
vector — so that population's membership timeline cannot break. It is out of the block because it
needs no protection, and that is a property to preserve rather than an accident
([ADR-0047](../adr/0047-an-address-scope-is-its-own-enumeration.md)).

**This section is about the membership obligation and nothing else.** §10's `Custody` block is not a
membership leaf and does not amend the table above; it discharges
[ADR-0008](../adr/0008-derivation-versions-move-on-content.md)'s declared-parameter rule instead.
§10.1 states the difference.

---

## 2. The `resolution-walk` block — 27 cells

Every cell must hold at least one corpus row in ADR-0021's form: **`(job-spec fragment, authored peer
script, expected NDJSON)` plus a one-line claim in prose**. Author each row in DNS presentation format.
Run it hermetically against an in-process scripted peer. No network, no containers, no fixture images.

Three rules govern the shape of the block, and they are ADR-0085's:

> **A boundary between two outcomes is pinned by two rows, one on each side, authored as a pair and
> failing as a pair.** A row cannot detect a collapse toward itself.

> **A row protects the leaf whose gate runs it, and nothing else.** A row filed against another
> leaf's gate reads green while protecting nothing, which is why ~~§7 is escrow rather than
> content~~ **`wildcard-discrimination`'s rows are §8's own block and never §2's, now that both
> leaves are in the membership vector.** Being in one vector is not being one gate

> **A row encoding a spec-verified rather than measured claim says so on the row** — ADR-0021's
> honesty rider, and every boundary in §2.2 is spec-derived today.

### 2.1 Block M1 — outcome pins · **5 cells**

The leaf makes **two** queries for one name and reads them from different peers
([ADR-0070](../adr/0070-a-control-probe-is-asked-from-where-the-answer-it-discriminates-was-asked-from.md)).

| Cell | Query | Expected | Membership job |
| --- | --- | --- | --- |
| M1.1 | declared path | `Resolved(set)` | The `Name` stays; the **set** is what cites `Address`es |
| M1.2 | declared path | `NoData` | The `Name` stays; nothing is cited, so previously-cited `Address`es fall out |
| M1.3 | declared path | `NameError` | **The only outcome that withdraws a `Name`** |
| M1.4 | delegation walk | `Lame` | Suppresses withdrawal — names beneath hold a `Gap`, and there is nobody left to return a Name Error |
| M1.5 | delegation walk | not-`Lame`, with the per-nameserver `serves │ does-not-serve` RRset | The walk's own output, and what a partly-lame delegation records instead of `Lame` |

### 2.2 Block M2 — boundary pins · **9 boundaries × 2 = 18 cells**

Each boundary is chosen on one criterion: **the wire is genuinely ambiguous there, and two conforming
implementations may read it differently.** Each row of this table is two cells.

| # | Boundary | The stimulus that discriminates | Why it lands on membership |
| --- | --- | --- | --- |
| M2.a | `NameError` ↔ `NoData` | An **empty non-terminal**: NOERROR with an empty ANSWER for a name that exists only as an ancestor | One side withdraws a `Name` and the other does not. The highest-stakes boundary in the product |
| M2.b | `NameError` ↔ `Resolved` | **NXDOMAIN carrying a CNAME** in the answer section — does the rcode apply to the qname or to the final name? | A library changing its reading withdraws or restores every aliased name in the estate |
| M2.c | `Resolved` ↔ `NoData` | A **CNAME chain** terminating with no address at the authority, resolvable to addresses on the declared path — the measured `s3.amazonaws.com` shape | Decides whether an address set exists to cite `Address`es at all |
| M2.d | `Resolved` ↔ `Gap` | **TC=1**, once with a TCP fallback that recovers the RRset and once with a fallback that does not | A truncated RRset no fallback recovered is a `Gap`, and a `Gap` is not a withdrawal |
| M2.e | `Lame` ↔ `Gap` | Every delegated authority **REFUSEs** (reached, does not serve) against every authority **silent** (not reached) | *Reached and refused* is a measurement of the operator's infrastructure; *not reached* is our own blindness |
| M2.f | `Lame` ↔ not-`Lame` | A **partly lame** delegation — one authority serves, one does not | A partly-lame delegation is **not** `Lame`; the name still resolves and `resolution` has not moved |
| M2.g | set equality ↔ serialisation | The same address set served in a **different RR order**, and with **0x20 case randomisation** on the qname | The set is unordered and the `Name` key folds ASCII case ([ADR-0055](../adr/0055-a-names-key-is-the-label-sequence-and-we-fold-only-what-the-protocol-folds.md)). Neither may move a span |
| M2.h | set equality ↔ spelling | An AAAA answering `::ffff:203.0.113.5` against an A answering `203.0.113.5` | They fold to **one** `Address` key — family and octets ([ADR-0051](../adr/0051-a-subject-key-is-the-thing-denoted-and-its-normalisation-may-never-move.md)) — so the set has one member, not two |
| M2.i | `NoData` ↔ `Gap` | An authority answering **FORMERR to a query carrying an OPT**, once with an EDNS-less retry that succeeds and once without | ADR-0021 names EDNS defaults as the residual un-coverable dependency case; this is it, made coverable |

### 2.3 Block M3 — the path-provenance pin · **1 cell**

| Cell | Claim |
| --- | --- |
| M3.1 | `Resolved`, `NoData` and `NameError` are read from the **declared path**; `Lame` and the per-nameserver RRset are read from the **delegation walk**, which the query-path parameter does not govern. The row scripts two peers that **disagree**, and the expected NDJSON reads each field from its own peer. A library upgrade that routed the walk through the resolver would delete `Lame` outright, and nothing else in the corpus would notice |

### 2.4 Block R — the withdrawal→return rows · **3 cells**

[#140](https://github.com/winniel123/verge-asm/issues/140) **confirmed** ADR-0041: a withdrawn
subject's timelines **close**. These three cells are therefore load-bearing rather than contingent.
See §5.

| Cell | Claim |
| --- | --- |
| R.1 | A name answering `NameError` from every scripted vantage, then `Resolved` on a later batch, under **one** vector. Expected: the leaf emits `NameError` then `Resolved`, and **nothing in the leaf's output names the transition**. The leaf is not where the transition is decided — the withdrawn period is on no timeline at all, which is what leaves the two spans adjacent |
| R.2 | The same sequence with the leaf's **version moved** between the two batches, the move being a dependency upgrade that touches no declared parameter. Expected: the leaf's output is **byte-identical** on both sides. This is the row that *is* the obligation — it is the proof that the upgrade was a no-op |
| R.3 | `NameError` at one vantage and `Resolved` at another **in the same batch**. Expected: two per-vantage outputs and **no fold**. *Every available vantage* is a quantifier the leaf does not evaluate |

### 2.5 The count

**This section counts `resolution-walk`'s block only.** `wildcard-discrimination`'s block is §8, and
the file's total is in §8.4.

| Block | Cells |
| --- | --- |
| M1 outcome pins | 5 |
| M2 boundary pins | 18 |
| M3 path provenance | 1 |
| R withdrawal→return | 3 |
| **Total, `resolution-walk`** | **27** |
| ~~*(§7 escrow, undischarged)*~~ | ~~*(17)*~~ — **discharged into §8**, [ADR-0086](../adr/0086-membership-composes-every-leaf-that-decides-the-value-it-reads.md) |

**27 is the length of a list, not a target.** It is the count of a member of the enumeration, exactly
as [#74](https://github.com/winniel123/verge-asm/issues/74) requires — *each member's count is the
length of its own list*. It moves when a cell is added or retired. A session that adds a
boundary adds **two** cells, and the total moves by two.

---

## 3. The CI matrix

### 3.1 The legs

Three, all **native**, all with every build setting **stated rather than defaulted**.

| Leg | `GOOS`/`GOARCH` | Microarchitecture | `CGO_ENABLED` | Runner | Gating on |
| --- | --- | --- | --- | --- | --- |
| 1 | `linux/amd64` | `GOAMD64=v1` | `0` | `ubuntu-24.04` | The expectation |
| 2 | `linux/arm64` | `GOARM64=v8.0` | `0` | `ubuntu-24.04-arm` | The expectation, and identity with leg 1 |
| 3 | `linux/amd64` | `GOAMD64=v3` | `0` | `ubuntu-24.04` | **Identity with leg 1 only** |

**Legs 1 and 2 are the matrix.** Leg 3 is not an architecture and ships no artefact, so
[`packaging-and-configuration.md`](./packaging-and-configuration.md) §1.2's membership rule — *an
architecture is in the matrix exactly where the golden corpus is run on it in CI* — is untouched.

Three notes that will otherwise be re-derived:

- **There is no `ubuntu-latest-arm`.** The `-latest` alias exists on `x64` and not on `arm64`. The
  `arm64` leg pins a version label, which this project would want regardless — a floating runner
  image is out-of-band reference data sitting in the CI configuration.
- **`arm64` hosted runners are free on public repositories** and have been GA since 2025-08-07
  (extended to private repositories 2026-01-29). #124's *"anything else is out until somebody pays
  for a runner"* prices these two at zero.
- **`CGO_ENABLED=0` is stated on every leg**, and the reason is not #124's. #124 pins it for the
  **shipped artefact**. These legs need it for the **binary under test**, and a native build is
  exactly where the toolchain's default is *on*. An unset value runs the corpus against a binary
  that may resolve through the system resolver — a second answer path ADR-0070 has already closed.
  The corpus would then read green while verifying the wrong instrument.

### 3.2 The assertions, in the order they fail

| # | Assertion | Scope | Why it sits here |
| --- | --- | --- | --- |
| A1 | **Self-identity** — each leg runs the corpus **twice in one process**; output must be byte-identical | Every leg | Go randomises map iteration **per iterator**. An unstable corpus makes every assertion below uninterpretable, and a red `arm64` leg indistinguishable from a flake |
| A2 | **Expectation** — each leg's output equals the **one shared** expected-output artefact | Every leg | ADR-0021's gate, first direction: *output moved and the version did not → fail* |
| A3 | **Cross-architecture identity** — legs 1 and 2 agree | Legs 1, 2 | A2 gives it by transitivity; it is asserted separately so the failure reads *architecture divergence* rather than *row 47 moved* |
| A4 | **Contraction differential** — leg 3 equals leg 1 | Leg 3 | The only check in the repository that a declared parameter's fraction was actually evaluated in exact integer arithmetic. A difference names the fraction that escaped |
| A5 | **Coverage** — every cell of §2, **of §8 and of §10** holds at least one row | Once | This is what gives the obligation an owner. A missing cell fails the build and the failure **names the cell** |
| A6 | **Gate direction 2** — on `resolution-walk`, on `wildcard-discrimination` **and on the `Custody` derivation (§10)**, a version bump with no moved pin row, no changed declared parameter and no recorded uncovered move fails | Once | ADR-0021's second direction, now with the evidence that lets it fire. This is the assertion that protects `returned`. *"Recorded uncovered move"* means a row in **§9** whose `Leaf` and `Bumped to` match the bump under test — form and validity rule at [ADR-0021's #177 annotation](../adr/0021-a-version-leaf-is-a-decision-not-a-binary.md#annotation--177-2026-08-15-the-uncovered-move-is-given-form) |

**There is no per-architecture expected output, ever.** An `arm64` golden file is the divergence
#124 exists to prevent, written down and blessed. It is the first repair a session will attempt
when leg 2 goes red. A2's *one shared artefact* is what makes A3 mean anything.

~~**A6's scope widens to a second leaf if and only if §7's routing question answers that way.** It is
written against `resolution-walk` today because that is the only leaf whose membership role is
settled.~~ **The condition is met and the widening has happened**, at the site that specifies it
([ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)).
[#146](https://github.com/winniel123/verge-asm/issues/146) ·
[ADR-0086](../adr/0086-membership-composes-every-leaf-that-decides-the-value-it-reads.md) rules that
`wildcard-discrimination` **is** in the membership vector. So **A5 and A6 both run over two leaves**.
A6 is evaluated **per leaf**. A bump of either leaf is tested against **that leaf's** block, because
a row protects the leaf whose gate runs it and nothing else. The two blocks are never pooled.

### 3.3 The trigger

**Every pull request and every release. Every leg, every assertion.**

The named failure mode arrives on a dependency PR — the `go.mod` cadence
[#49](https://github.com/winniel123/verge-asm/issues/49) put `resolution-walk` on. The tempting
design is a `go.mod`-touched path filter. It is refused. The corpus is hermetic by construction, so
there is nothing to save. A path filter is a second place the obligation can be silently
disabled. Running everything on everything makes the dependency case unremarkable, which is the only
way to be sure it is covered.

---

## 4. Who owes it, who owns it, what enforces it

| Role | Party |
| --- | --- |
| **Owes** | **The release**, per [ADR-0041](../adr/0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md) §*What the release owes* — unchanged |
| **Owns** | **§2's and §8's enumerations**, checked in as data. An obligation with no failing test has no owner. **§9's register** owns ADR-0021's uncovered-move escape hatch the same way |
| **Enforces** | **A5** (a cell with no row fails, naming the cell) and **A6** (an unjustified bump on **either** membership leaf, or on the `Custody` derivation, fails — tested against that leaf's own block, with §9 as A6's third limb) |
| **Renders on failure** | The row's **claim**, the old output and the new one — ADR-0021's judgeability property, unchanged |
| **Undischarged** | ~~The `Shadowed` outcome — **§7**, and it has no owner until the routing question is answered~~ — **nothing.** All five of ADR-0041's outcomes are discharged, across two blocks against two leaves ([ADR-0086](../adr/0086-membership-composes-every-leaf-that-decides-the-value-it-reads.md)). Superseded here, at the site that specifies it |

---

## 5. What #140 settled

[#140](https://github.com/winniel123/verge-asm/issues/140) **confirmed** ADR-0041 rather than
reversing it: a withdrawn subject's timelines **close**. So Block R stands. A6 is a hard refusal
rather than a redundancy with ADR-0021's existing second gate direction. A `Break` between
withdrawal and return leaves the reopening with nothing legally before it. An unjustified bump
costs `returned` estate-wide and permanently.

Two corollaries #140 added that this file depends on:

- **The withdrawn period is on no timeline at all** — neither a value nor a `Gap`. That is what
  leaves the span before withdrawal and the span after a return **adjacent**. It is why
  `returned` is derivable by ordinary machinery wherever no `Break` sits between them. R.1 pins the
  leaf side of it: the leaf emits two outcomes and names no transition.
- **A closed span is free to keep and an open one must be fed.** That asymmetry is why the `Span`
  corpus is never compacted. It is the reason the pin block's protection is worth having. There
  is no storage-side repair for a lost `returned`, because there was never a storage-side cost to
  keeping what would have proved it.

Had #140 gone the other way, Block R's cells would have retired and A6 would have degraded to
redundancy. Blocks M1–M3 would not have moved either way — they pin the leaf's outcomes. Membership is
decided from those outcomes identically under both readings.

---

## 6. Where this is thin

**This section is about §2 and §3.** §8's own thin ground is §8.5, and its first bullet inherits the
first bullet here whole.

- **The nine boundaries in §2.2 are reasoned from the DNS specifications, not measured against two
  DNS libraries.** Nobody has diffed two implementations across them. Every row in that block
  therefore carries ADR-0021's spec-verified marker until somebody measures it.
- **Completeness of §2 is not checkable by anything.** A5 counts rows against the enumeration. It
  cannot tell anyone the enumeration named every ambiguous place on the wire. That residue is
  ADR-0021's **uncovered move**. It is disposed of the same way — recorded as data, reviewable
  and countable. If it becomes the common case, the corpus is failing and the count says so. Its
  form is **§9**.
- **A4's yield is unmeasured.** No fraction in the declared-parameter set has been shown to diverge
  at `GOAMD64=v3`. If A4 never fires, that is indistinguishable from A4 being unnecessary. The
  argument for keeping it is that the check is cheap and the failure it catches is silent.
- **That this project's CI will be GitHub Actions is assumed**, no ticket having decided it. The
  legs, the assertions and their order transfer unchanged if it is not. Only the runner labels move.

---

## 7. The fifth outcome — `Shadowed` — and how it was routed

~~**This section is escrow. Nothing in it is discharged, and none of it may be filed into
`resolution-walk`'s corpus.**~~ **Superseded here, at the site that specifies it**
([ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)): read
alone and in the present tense it holds seventeen written cells out of a gate they now belong in.

> **ROUTED — [#146](https://github.com/winniel123/verge-asm/issues/146) ·
> [ADR-0086](../adr/0086-membership-composes-every-leaf-that-decides-the-value-it-reads.md),
> 2026-08-15. The answer is home A.** `wildcard-discrimination` **is** in the membership vector.
> The seventeen escrowed cells are **adopted whole and unamended** into that leaf's corpus, and they
> are now **§8** of this file. The **second** half of §7.4 is what did *not* change: the rows go to
> `wildcard-discrimination`'s gate and never to `resolution-walk`'s, because a row protects the leaf
> whose gate runs it.
>
> §7.1 and §7.2 are kept as the record of the defect and of the option that lost. **They are history,
> not an open question** — nothing in them is to be re-litigated, and §7.3's table has moved to §8
> rather than being duplicated here.

### 7.1 The defect (as found by [#143](https://github.com/winniel123/verge-asm/issues/143))

[ADR-0041](../adr/0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md)'s
obligation 2 names five outcomes to pin — `Resolved`, `NoData`, `NameError`, `Lame`, `Shadowed` —
and scopes them to *`resolution-walk`'s golden corpus*. Four are that leaf's.
[ADR-0021](../adr/0021-a-version-leaf-is-a-decision-not-a-binary.md)'s leaf table puts the fifth,
**`Shadowed`, under `wildcard-discrimination`** — a leaf ADR-0021 kept separate on purpose, *"so a
break **names its leaf**"*.

### 7.2 The two candidate homes, and the one that lost

| Home | The claim it requires | Its price |
| --- | --- | --- |
| **A — `wildcard-discrimination`'s own golden corpus.** ✅ **RULED, [ADR-0086](../adr/0086-membership-composes-every-leaf-that-decides-the-value-it-reads.md)** | The membership vector is **wider** than ADR-0041 and [`CONTEXT.md`](../../CONTEXT.md) record: `Shadowed` is a value on `resolution`, under it a `Name` *cannot leave at all*, and bumping the leaf that decides it `Break`s the timeline membership reads. **ADR-0086 rules for A on a ground this row did not name** — `Shadowed` **cites no `Address`**, so the verdict decides whether an answer's address set enters the estate, which is a cited `Address`'s membership affirmatively rather than by suppression | **`wildcard-discrimination`'s control-label count has already moved** — `5` → `9`, [#115](https://github.com/winniel123/verge-asm/issues/115) — which ADR-0021 records as bumping the leaf and `Break`ing `resolution`. **Priced by ADR-0086 at zero**: nothing has shipped and no `resolution` timeline exists, so there was no membership timeline to break. The reason **expires at the first shipped release** |
| **B — nowhere; the obligation for `Shadowed` is void.** ❌ **The losing option** | The vector is exactly `resolution-walk` as recorded, and `Shadowed` **withholds a value without deciding presence** — a name under it is *visibly unconfirmed* rather than absent, so no membership fact turns on it | ADR-0041's obligation 2 is wrong to list it, and must be amended at its own site to a four-outcome list. **It lost** because `Shadowed` cites nothing and therefore withdraws `Address`es; because membership reads `resolution` and `resolution` is decided by two leaves; and because it would have left ADR-0085's named failure — estate-wide, silent, on a no-op upgrade — standing unpinned one leaf over |

**It cannot be neither.** One of ADR-0041's sentence and ADR-0021's leaf table is inaccurate.
Which one is a question with a real consequence attached. ~~[#140](https://github.com/winniel123/verge-asm/issues/140)
has raised it as a successor ticket. **This file does not answer it**, because a decision lives in
exactly one place and that place is the successor.~~ **Answered**: ADR-0021's leaf table is right and
**ADR-0041's sentence was short by one leaf**. Obligation 2's **list** of five outcomes is correct.
Its **scope** — *`resolution-walk`'s golden corpus* — was the defect, and it is amended at its own
site.

### 7.3 The escrow — ADOPTED WHOLE, and now §8

~~If the routing question answers **A**, these seventeen cells are adopted whole into
`wildcard-discrimination`'s corpus and A5's coverage assertion extends over them. If it answers
**B**, they are discarded.~~ **Superseded here, at the site that specifies it.** The routing question
answered **A**. The seventeen cells are **adopted whole and unamended** and they are **§8** of this
file. A5's coverage assertion extends over them. A6 widens to the leaf. The table has **moved** to
§8.1 rather than being copied, so this file holds one enumeration per leaf and no duplicate.

Escrow worked and the record should say so plainly: the successor **ruled the routing and re-derived
no rows.** [ADR-0085](../adr/0085-an-obligation-with-no-failing-test-has-no-owner-and-a-boundary-needs-a-row-on-each-side.md)
called seventeen-written-for-nothing *"the cheaper error of the two available"* and did not have to
pay it.

### 7.4 Why a row goes to its own leaf's gate — unchanged, and it is what adoption means

**A corpus row protects the leaf whose gate runs it, and nothing else.** A `Shadowed` row sitting in
`resolution-walk`'s corpus is checked when `resolution-walk`'s version is questioned. It is silent
when `wildcard-discrimination` bumps — the leaf that actually decides the value. The row would be
exercised by a leaf that cannot move it and ignored by the leaf that can.

That is not a misfiled row. It is a row that **reads green while protecting nothing**, and it would
make A5 report the obligation discharged. An absent cell is countable. A falsely-filed one is not.

**This rule survives ADR-0086 intact and is the reason §8 is a separate block.** Two leaves in one
vector are not one gate: A6 is evaluated **per leaf** against **that leaf's** block, and the two
blocks are never pooled. Being in one vector is a statement about **comparison**. Being in one block
would be a statement about **protection**, and only the first is true.

---

## 8. The `wildcard-discrimination` block — 19 cells

- **Ruled by** [#146](https://github.com/winniel123/verge-asm/issues/146) · [ADR-0086](../adr/0086-membership-composes-every-leaf-that-decides-the-value-it-reads.md)

`wildcard-discrimination` is in the membership vector, so ADR-0041's obligation reaches it and it
gets a block of its own on §2's terms: every cell holds at least one corpus row in ADR-0021's form —
**`(job-spec fragment, authored peer script, expected NDJSON)` plus a one-line claim in prose**.
Author each row in DNS presentation format. Run it hermetically against an in-process scripted peer, with no
network, no containers and no fixture images. The same three shape rules govern it:

- A boundary is pinned by **a pair of rows, one on each side**.
- A row protects **the leaf whose gate runs it**.
- A row encoding a **spec-verified rather than measured** claim says so on the row.

**Provenance is marked per block, and it matters for review.** §8.1's seventeen cells are
[ADR-0085](../adr/0085-an-obligation-with-no-failing-test-has-no-owner-and-a-boundary-needs-a-row-on-each-side.md)'s
escrow **adopted whole and unamended**. The ticket that priced *both* homes wrote them, which is
better evidence than rows written by the ticket that had already picked one. §8.2's pair is **minted
by ADR-0086** and has had one author and no second reader.

### 8.1 The adopted escrow — 17 cells, unamended

| Block | Cells | Contents |
| --- | --- | --- |
| W1 — component signatures | 3 | One per member of the per-component signature union, per `(qtype asked, RR type in the answer)` ([ADR-0068](../adr/0068-a-wildcard-is-discriminated-only-where-its-synthesis-is-determinate.md)): `NoSynthesis` · `Determinate(RRset)` · `Indeterminate` |
| W2 — verdicts | 2 | `Shadowed` · not-`Shadowed`, the latter being a name that **differs at some determinate component** |
| W3 — boundary pins | 8 | Four boundaries, two cells each: **(a)** differs at a **determinate** component against differing **only** at an `Indeterminate` one, which is never consulted; **(b)** **no** component determinate, so every name beneath the parent is `Shadowed`, against at least one determinate-and-differing; **(c)** a probe that **completed** and found no wildcard, licensing everything beneath, against one that **did not complete** — a `Gap`, never a value, since *an undiscriminated answer is never a value*; **(d)** discriminated at **one** component, so no synthesised RRset at **any** qtype, against `Shadowed` holding on `resolution` and on every `dns-record` discriminator |
| W4 — control-label set | 2 | **(1)** the set is **9 random + 1 structured** label, each **exactly one label**, the structured one `<a>-<b>-<c>-<d>` over a random RFC 5737 documentation address ([ADR-0069](../adr/0069-a-control-label-is-one-label-and-the-set-must-falsify-label-independence.md), [#115](https://github.com/winniel123/verge-asm/issues/115)), with a two-label label as the negative case since it falls off the tree at a **deeper encloser** and measures a different wildcard; **(2)** the labels are drawn **per `Batch`** and are **independent samples** — the measured mechanisms being per-label sharding and per-query rotation, never per-time ([`passive-discovery-sources.md`](../research/passive-discovery-sources.md) §13) |
| W5 — the shared path | 1 | A control probe asked from a **different** place than the answer it discriminates yields **no verdict**, never a wrong one. **[measured]** — direct to its own authority, `s3.amazonaws.com` reads a *determinate* `NoSynthesis` at A while a resolver answers every candidate beneath with eight addresses, so a skewed pair discriminates every fictional label and records it `Resolved` with a fabricated set ([ADR-0070](../adr/0070-a-control-probe-is-asked-from-where-the-answer-it-discriminates-was-asked-from.md)) |
| W6 — the suppression | 1 | `Shadowed` where a `NameError` would otherwise have been measured. Expected: `Shadowed`, and **no withdrawal-shaped output**. This is the cell that makes the membership role concrete, and it is the one the routing question is really about |
| **Subtotal** | **17** | |

**W6 carries a rider ADR-0086 adds and the cell keeps.** It pins the ticket's own ground — the `Name`
suppression case — which ADR-0086 rules **sound and unmeasured**. Beneath a wildcarding parent, RFC
4592 answers NOERROR, so a deleted name never reads `NameError`. Beneath a non-wildcarding parent,
ADR-0066's *a probe that completed and found no wildcard licenses everything beneath it* exempts the
name. The cell's stimulus is therefore a parent that **neither synthesises nor reads determinate at
any component**. That is a shape nobody has measured. **The row is authored and carries the
spec-verified marker.** It is not the block's load-bearing cell. §8.2's pair is.

### 8.2 The citation pin — 1 boundary × 2 = 2 cells · **minted by ADR-0086**

This boundary is the ground ADR-0086 rules on, and the escrow has no cell for it. A block with no row
pinning *`Shadowed` carries no address set* would read green while the ruling's load-bearing claim is
unprotected. That is exactly the defect §7.4 refuses one level down.

| # | Boundary | The stimulus that discriminates | Why it lands on membership |
| --- | --- | --- | --- |
| W7.1 | `Shadowed` — **cites nothing** | A name beneath a wildcarding parent whose synthesised answer **carries addresses**, undiscriminated at every determinate component. Expected: `Shadowed`, and the NDJSON carries **no address set at all** | An answer served for a name that does not exist may **cite no `Address` and open no `Endpoint`**. Every `Address` held only by this citation leaves the estate |
| W7.2 | not-`Shadowed` — **cites** | The same name and the same parent, **differing at a determinate component**. Expected: `Resolved(set)`, and the NDJSON carries the set | The set cites, so the `Address`es are in the estate. A cited `Address` is in the estate **exactly while a current resolution cites it** |

**Authored as a pair and failing as a pair.** A single row here cannot detect the collapse it exists
to catch. A leaf that stopped emitting the address set on the `Resolved` side would pass W7.1
untouched. A leaf that started emitting one on the `Shadowed` side would pass W7.2 untouched.
Between them they pin the one property that makes this leaf a membership leaf.

**Both rows are spec-verified rather than measured**, per ADR-0021's honesty rider, and both say so
on the row. What *is* measured is that the leaf's declared parameter moves the verdict these rows
straddle. [ADR-0068](../adr/0068-a-wildcard-is-discriminated-only-where-its-synthesis-is-determinate.md)
records `surge.sh` as a two-member hash of the query label, `188/172` over 360 labels, where **five
random labels land in one shard 3 times in 30**. So the `5` → `9` move
([#115](https://github.com/winniel123/verge-asm/issues/115)) changes which names beneath that zone are
`Shadowed`, and therefore which `Address`es are cited.

### 8.3 What A6 tests on this leaf

On `wildcard-discrimination`, a version bump with **no moved cell in §8**, no changed declared
parameter and no recorded uncovered move **fails the build**. Its declared parameters, for the
parameter limb of that test, are ADR-0021's four: the **control-label count and construction**, the
**match predicate**, and the **query path** — the last shared with `resolution-walk` and taking one
value per `Batch`.

**Three of the four are project-authored**, which is the whole shape of this leaf's risk and is worth
stating where a release will read it. The query path and the DNS library are already parameters of
`resolution-walk`, so they reached membership before this block existed. What is new to membership is
the control-label count, its construction and the match predicate — all ours, all on the roadmap
cadence rather than `go.mod`'s, all foreseeable. What is also new, and is the reason this block
exists: **a DNS-library change that moves discrimination but not resolution now breaks membership
where it previously did not**. §8 is what makes the no-op case provable instead of assumed.

### 8.4 The count

| Block | Leaf | Cells |
| --- | --- | --- |
| §2 — M1 · M2 · M3 · R | `resolution-walk` | **27** |
| §8.1 — W1–W6, adopted whole | `wildcard-discrimination` | 17 |
| §8.2 — W7, the citation pin | `wildcard-discrimination` | 2 |
| **§8 subtotal** | `wildcard-discrimination` | **19** |
| **File total, all discharged** | | **46** |
| *Escrow* | | **0** |

**46 is the length of two lists, not a target**, on §2.5's rule and
[#74](https://github.com/winniel123/verge-asm/issues/74)'s. A session that adds a boundary to either
block adds **two** cells and moves that block's subtotal and the total by two. **The two subtotals are
never merged into one gate** — they are counted together and run apart.

### 8.5 Where §8 is thin

- **§8.2's pair is minted here and has had no second reader.** ADR-0085 wrote the seventeen adopted
  cells and ADR-0086 reviewed them. One session wrote and ruled
  the two new ones, and they are the block's most load-bearing pair.
- **W6's stimulus is a shape nobody has measured** — a parent that neither synthesises nor reads
  determinate at any component. Every indeterminate zone in ADR-0068's nineteen synthesises. The cell
  is kept because the ruling names the case. It is marked spec-verified because nothing has seen one.
- **§8 inherits §6's first bullet whole.** No boundary in this block has been diffed across two DNS
  libraries either, and the same marker applies to every row in it.

---

## 9. The uncovered-move register

- **Ruled by** [ADR-0021](../adr/0021-a-version-leaf-is-a-decision-not-a-binary.md)'s
  [#177 annotation](../adr/0021-a-version-leaf-is-a-decision-not-a-binary.md#annotation--177-2026-08-15-the-uncovered-move-is-given-form)

ADR-0021 named the gate's third limb — *a version may move on a recorded uncovered move naming the
input class the corpus cannot reach* — but never drew the object. This section is where it lives, on
the same precedent as §2 and §8: **the table is revised here. The rule that governs it is the ADR's,
and stays there.**

A row is the checked-in entry itself: not a description of a gap, but the gap named precisely enough
for CI to match it against a specific leaf's specific version bump.

### 9.1 Fields

| Field | Content |
| --- | --- |
| **Leaf** | One of ADR-0021's six named leaves, exact identifier: `connect-outcome` · `tls-handshake` · `http-exchange` · `resolution-walk` · `wildcard-discrimination` · `datagram-outcome` |
| **Bumped to** | The exact post-bump version string on that leaf's `derivation` row ([ADR-0008](../adr/0008-derivation-versions-move-on-content.md)). Matched by string equality, not description |
| **Input class** | One line of prose naming the class of input the corpus cannot reach, e.g. *"behaviour against servers that close on an unrecognised extension"* — a class, never a single instance |
| **Ticket** | The issue (and PR, once merged) that shipped the bump |
| **Date** | The date the row was checked in |

### 9.2 Validity rule

Full statement and rationale at ADR-0021's #177 annotation. Restated here because this is the file a
session edits when adding a row.

- **One row per `(Leaf, Bumped to)` pair.** A row licenses exactly one version transition of exactly
  one leaf. A later bump of the same leaf needs its own row.
- **Present by the commit A6 evaluates**, in practice added by the same PR that ships the bump —
  exactly as a moved pin row or a changed declared parameter already must be, per §3.3's *every pull
  request*.
- **Append-only.** A row, once merged, is never edited or removed. Recorded rather than resolved: a
  later corpus row covering the same input class does not retire the entry. It only means the leaf's
  *next* bump can be justified the ordinary way.
- **No status field.** There is nothing to mark resolved — see the annotation for why a `resolved`
  column was considered and refused.

### 9.3 What consumes it, today

Only [§3.2](#32-the-assertions-in-the-order-they-fail)'s **A6**, and only for `resolution-walk`,
`wildcard-discrimination` and `custody` — the leaves whose corpus and CI matrix this file specifies.
A row for one of the other four leaves is legal, checked-in data with no consumer yet, on the same
ground ADR-0021 already used for `datagram-outcome`: specified, and not yet load-bearing.

**`custody` is a legal value of the `Leaf` field**, added by §10. It is not one of ADR-0021's six
measurement leaves; it is [ADR-0008](../adr/0008-derivation-versions-move-on-content.md)'s name for
the `Custody` derivation's version, and §10's lock carries it. Everything in §9.1 and §9.2 governs
such a row unchanged.

### 9.4 The register

No uncovered move has been recorded. The table is checked in empty and grows only by an appended row,
never by an edit to one already here.

| Leaf | Bumped to | Input class | Ticket | Date |
| --- | --- | --- | --- | --- |
| *(none recorded)* | — | — | — | — |

---

## 10. The `Custody` block — 6 cells

- **Ticket:** [#986](https://github.com/winniel123/verge-asm/issues/986), under the
  [ADR-0129](../adr/0129-a-shared-foreign-edge-is-measured-by-fan-out-not-read-from-a-list.md) map
  [#981](https://github.com/winniel123/verge-asm/issues/981)
- **Ruled by** [ADR-0008](../adr/0008-derivation-versions-move-on-content.md) (a declared parameter is
  pinned by a corpus row whose output the value decides), ADR-0129 §3 and its
  [#955](https://github.com/winniel123/verge-asm/issues/955) and
  [#956](https://github.com/winniel123/verge-asm/issues/956) amendments
- **Lives at** `internal/custody/corpus/` — rows, script, harness, `corpus.lock.json`, `testdata/`

### 10.1 What this block is not

It is **not a membership block.** The `Custody` derivation composes into no `Span`, so ADR-0041's
estate-wide `Break` — the failure §0 exists to prevent — cannot arrive through it. §1's rule stands
unamended: a `Seed`-covered `Address` composes nothing at all, and it is out of the **membership**
block because it needs no protection there.

What brings `Custody` here instead is ADR-0008. ADR-0129 §3 puts a **project-authored number inside
the derivation** — the fan-out threshold, `custody.SharedEdgeThreshold`, fixed at 100 and never
operator-configurable — and ADR-0008 pins such a number by a corpus row whose output the value
decides. That is a different obligation with the same instrument, so it gets its own block and its
own lock, and the two are **never pooled**: a break in the fan-out boundary moves THIS digest and
bumps `custody/v2`, never `resolution-walk`'s.

The block also holds a row a `Seed`-covered address appears in. That is not §1 reversed. The row
pins that the veto does **not** reach the declaration, which is a fact about the derivation and not
about a membership timeline.

### 10.2 The cells

| Cell | Claim the row pins |
| --- | --- |
| `C1/below-threshold-reached` | An Observed SAN set reducing to **99** distinct registrable domains derives `not-shared`. The custody extension **reaches** the edge, it derives `operator`, and the probing gate opens |
| `C1/at-threshold-vetoed` | An Observed SAN set reducing to **100** derives `shared`. The extension **declines** the reach, the address derives `third-party`, and the gate is shut |
| `C2/seed-covered-at-threshold-operator` | A **`Seed`-covered** address whose SAN set reduces to at least 100 derives `operator` and is **reached**. The veto and the declaration are disjoint limbs, never ranked |
| `C2/seed-covered-stays-a-candidate` | That same address is still an `edge-fanout` candidate. The population reads the **pre-veto** reach, so a later handshake can lift a veto |
| `C3/pending-held` | An in-force `Scan` that has completed **no Batch** holds the reach — neither reaching nor declining. Absence is hold-then-open |
| `C3/scan-not-in-force-reaches` | A `Scan` that is disabled or whose row is absent **reaches** the address even with a shared measurement on record. That is the pre-ADR-0129 behaviour and the zero value |
| `C3/extension-limb-errored-reaches` | A `Scan` that completes a Batch and records **declaration-limb rows alone** is errored on the **extension limb**: its candidates reach and derive `operator`, and the declaration limb keeps its own measurement |
| `C3/measured-candidate-holds-the-rest` | **One measured extension candidate lifts the floor** and the rest stay held. On an install whose `Scan` has run this limb, an unmeasured candidate is a lag bounded by the cadence, never a failure |

**Eight is the length of a list, not a target**, on §2.5's rule. It moves when a cell is added or
retired.

`C1` is ADR-0085's boundary rule in force: two rows, one on each side, authored as a pair and failing
as a pair. `C2` is the guard ADR-0129's #956 amendment asks for and is the strongest one the map
leaves behind — a session that "repairs" the apparent inconsistency by making the veto global fails
this gate at once, on a named row. `C3` exists because every other row carries a measurement, so none
of them would move if a session flipped the hold to a reach.

`C3/pending-held` and `C3/extension-limb-errored-reaches` are **one bit apart** and are authored as a
pair. Both hold an in-force `Scan` with no extension-limb measurement; one has completed a Batch and
one has not. The first holds and the second reaches — the floor is **per limb** since
[#1018](https://github.com/winniel123/verge-asm/issues/1018), so a declaration-limb row does not lift
it. A session that puts the floor back on the whole store moves the errored row's golden and the
digest with it.

`C3/measured-candidate-holds-the-rest` closes that pair's other side. Neither row above moves if a
session read the new floor as *any unmeasured candidate reaches*, because every candidate in them is
unmeasured. This row holds a **measured** candidate beside an unmeasured one, so the widening moves
its golden and is named.

### 10.3 The row shape, and where the threshold enters

A row is an `(estate, observed SAN set per address, expected NDJSON)` tuple, rendered hermetically
through `custody.Estate`. **No network, no database, no containers** — the estate is in-process data.

Each rendered line carries the whole path, from the measurement in to the gate out: the fan-out
count, the `shared-edge` verdict, whether the address is an `edge-fanout` candidate, whether a
declared address scope covers it, the derived `Custody`, and whether the probing gate opens from an
`internet`-class Vantage. A row therefore fails on the path rather than on one boolean.

Two authoring rules carry the whole value of the block:

- **The verdict is computed at render time, never written into the row.** A row declares a SAN set;
  `custody.SharedEdge` — reading the shipped `const` alone — decides which side of the boundary it
  lands on. That is what puts the threshold inside the corpus digest.
- **The boundary counts are absolute integers**, deliberately not `SharedEdgeThreshold` and
  `SharedEdgeThreshold - 1`. Written relative to the constant they would follow a move and keep
  straddling it, pinning the boundary's *shape* and never its *position*. Written absolutely, a move
  to 99 flips the first row's verdict, its golden and the digest at once.

The SAN fixtures reduce under `.invalid`, which RFC 2606 reserves and delegates to nobody, so the
corpus reaches no real estate and no PSL entry can be registered under it. Each set also carries a
wildcard SAN over an already-counted domain and one `iPAddress` SAN, both of which raise the count by
zero — the row is a claim about the **reduction**, not about a list length.

### 10.4 What A6 tests on this block

`custody.Version` (`custody/v2`) is the derivation's version leaf. The lock binds it to the corpus
digest and to the declared-parameter digest — the threshold and the Public Suffix List's own revision
string, which ADR-0129's [#954](https://github.com/winniel123/verge-asm/issues/954) amendment makes
the same kind of input as the threshold.

A threshold move with no version bump fails in **three** places, and
`TestThresholdMoveFailsTheGate` proves each:

1. The **params digest** moves, so `TestCorpusLock` fails — this limb fires even where no row's
   output happened to cross the boundary.
2. The **corpus digest** moves, because the 99 row's rendered verdict, `Custody` and gate all flip.
3. `TestFixtureStraddlesTheThreshold` fails first and by name, telling the session it owes a
   `custody.Version` bump and a re-bless.

The other direction — a bump with nothing else moved — is CI's `corpus-version-gate` job, which reads
this lock beside `resolution-walk`'s. **A6 is evaluated per lock.** The blocks are never pooled.

An **operator act moves this version not at all.** Declaring an address scope satisfies the reach; it
does not change what the derivation computes. Every input to the version is a `const` or a
dependency, so no install can move a `Custody` version without a release — which is ADR-0129 §3 and
the [#55](https://github.com/winniel123/verge-asm/issues/55) constraint in force.

### 10.5 Where §10 is thin

- **`custody/v2` composes into no `drift` component vector.** The measure leaves are in that vector
  because a `Span` reads their outcomes. Whether `Custody` joins it has an estate-wide `Break` on the
  other side of it, and #986 did not take that decision. Today the constant does one job: it names
  the derivation the lock binds.
- **The block pins the derivation, never the store.** Whether the `edge-fanout` `Scan` recorded the
  right rows in the first place is `internal/queue`'s and the leaf's, pinned by their own tests. A
  row here starts from an estate already assembled.
- **Legs 3's contraction differential (A4) tests nothing here.** The threshold is an absolute integer
  and the reduction is a set count, so this block evaluates no fraction. It runs on every leg anyway,
  under §3.3's *running everything on everything*.
