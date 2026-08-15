# The golden corpus: the membership pin block, and the CI matrix that runs it

- **Status:** Accepted — spec content for [#12](https://github.com/winniel123/verge-asm/issues/12)
- **Ticket:** [#143 `resolution-walk`'s golden corpus owes rows pinning the membership-deciding outcomes](https://github.com/winniel123/verge-asm/issues/143)
- **Rulings:** [ADR-0085](../adr/0085-an-obligation-with-no-failing-test-has-no-owner-and-a-boundary-needs-a-row-on-each-side.md) (this file's rules), [ADR-0021](../adr/0021-a-version-leaf-is-a-decision-not-a-binary.md) (the corpus and the gate), [ADR-0041](../adr/0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md) (the obligation), [`packaging-and-configuration.md`](./packaging-and-configuration.md) §1 (the architecture rule)

This is a separate file from the ADR that rules it, on
[`measurement-offers.md`](./measurement-offers.md)'s and
[`packaging-and-configuration.md`](./packaging-and-configuration.md)'s precedent: **the tables below
will be revised and an ADR is a decision that will not.** §2's enumeration in particular moves
whenever a leaf, a declared parameter or a boundary moves, and §7's escrow is written to be
discarded or adopted whole.

Nothing here is a test file or a workflow file. The map's *plan, don't do* rule holds: this
specifies which cells must hold a row, which legs must run, and in what order the assertions fail.
An implementation session writes them.

---

## 0. What this file is for, in one paragraph

[ADR-0041](../adr/0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md) found that
a dependency upgrade can bump a version leaf that the **membership** timeline composes; that a
`Break` between a subject's withdrawal and its return destroys `returned`; and that the repair is a
golden corpus dense enough that a no-op upgrade **provably** does not bump the leaf. It named the
obligation and gave it nobody to fail on. This file is the owner: a checked-in enumeration of
**27 cells**, each of which must hold a corpus row, counted by CI, plus the matrix the rows run on —
and **17 further cells held in escrow** against a routing question this ticket may not answer (§7).

The failure it prevents is estate-wide, silent, and triggered by a no-op upgrade. That is why the
obligation is data rather than prose.

---

## 1. Which leaf this file discharges

[ADR-0021](../adr/0021-a-version-leaf-is-a-decision-not-a-binary.md) names five version leaves inside
the measurement binary. **This file discharges the obligation for `resolution-walk` and no other
leaf.**

| Leaf | Discharged here? | Why |
| --- | --- | --- |
| `resolution-walk` | **Yes** | A `Name` leaves on `NameError` from every available vantage; a cited `Address` is in the estate exactly while a current resolution cites it. Both are this leaf's outputs, and it is the leaf ADR-0041 names |
| `wildcard-discrimination` | **No — routed, see §7** | `Shadowed` is ADR-0041's fifth outcome and is **this** leaf's, not `resolution-walk`'s. Whether that puts the leaf in the membership vector is [#140](https://github.com/winniel123/verge-asm/issues/140)'s successor question |
| `connect-outcome` | No | A port opening is a `Reach` move and never a membership event; a `Service`'s membership is its `Address`'s restated ([ADR-0031](../adr/0031-membership-alerts-at-the-root-of-the-entering-subtree.md)) |
| `tls-handshake` | No | Feeds `certificate` and `tls-acceptance` |
| `http-exchange` | No | Feeds `http-identity` only |

The `Seed`-covered `Address` composes **nothing at all** — a `Seed` is Declared and carries no
vector — so that population's membership timeline cannot break. It is out of the block because it
needs no protection, and that is a property to preserve rather than an accident
([ADR-0047](../adr/0047-an-address-scope-is-its-own-enumeration.md)).

---

## 2. The membership pin block — 27 cells

Every cell must hold at least one corpus row in ADR-0021's form: **`(job-spec fragment, authored peer
script, expected NDJSON)` plus a one-line claim in prose**, authored in DNS presentation format,
run hermetically against an in-process scripted peer. No network, no containers, no fixture images.

Three rules govern the shape of the block, and they are ADR-0085's:

> **A boundary between two outcomes is pinned by two rows, one on each side, authored as a pair and
> failing as a pair.** A row cannot detect a collapse toward itself.

> **A row protects the leaf whose gate runs it, and nothing else.** A row filed against another
> leaf's gate reads green while protecting nothing, which is why §7 is escrow rather than content.

> **A row encoding a spec-verified rather than measured claim says so on the row** — ADR-0021's
> honesty rider, and every boundary in §2.2 is spec-derived today.

### 2.1 Block M1 — outcome pins · **5 cells**

The leaf makes **two** queries for one name and they are read from different peers
([ADR-0070](../adr/0070-a-control-probe-is-asked-from-where-the-answer-it-discriminates-was-asked-from.md)).

| Cell | Query | Expected | Membership job |
| --- | --- | --- | --- |
| M1.1 | declared path | `Resolved(set)` | The `Name` stays; the **set** is what cites `Address`es |
| M1.2 | declared path | `NoData` | The `Name` stays; nothing is cited, so previously-cited `Address`es fall out |
| M1.3 | declared path | `NameError` | **The only outcome that withdraws a `Name`** |
| M1.4 | delegation walk | `Lame` | Suppresses withdrawal — names beneath hold a `Gap`, and there is nobody left to return a Name Error |
| M1.5 | delegation walk | not-`Lame`, with the per-nameserver `serves │ does-not-serve` RRset | The walk's own output, and what a partly-lame delegation records instead of `Lame` |

### 2.2 Block M2 — boundary pins · **9 boundaries × 2 = 18 cells**

Chosen on one criterion: **the wire is genuinely ambiguous there, and two conforming
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

[#140](https://github.com/winniel123/verge-asm/issues/140) has **confirmed** ADR-0041: a withdrawn
subject's timelines **close**. These three cells are therefore load-bearing rather than contingent.
See §5.

| Cell | Claim |
| --- | --- |
| R.1 | A name answering `NameError` from every scripted vantage, then `Resolved` on a later batch, under **one** vector. Expected: the leaf emits `NameError` then `Resolved`, and **nothing in the leaf's output names the transition**. The leaf is not where the transition is decided — the withdrawn period is on no timeline at all, which is what leaves the two spans adjacent |
| R.2 | The same sequence with the leaf's **version moved** between the two batches, the move being a dependency upgrade that touches no declared parameter. Expected: the leaf's output is **byte-identical** on both sides. This is the row that *is* the obligation — it is the proof that the upgrade was a no-op |
| R.3 | `NameError` at one vantage and `Resolved` at another **in the same batch**. Expected: two per-vantage outputs and **no fold**. *Every available vantage* is a quantifier the leaf does not evaluate |

### 2.5 The count

| Block | Cells |
| --- | --- |
| M1 outcome pins | 5 |
| M2 boundary pins | 18 |
| M3 path provenance | 1 |
| R withdrawal→return | 3 |
| **Total, discharged** | **27** |
| *(§7 escrow, undischarged)* | *(17)* |

**27 is the length of a list, not a target.** It is the count of a member of the enumeration, exactly
as [#74](https://github.com/winniel123/verge-asm/issues/74) requires — *each member's count is the
length of its own list* — and it moves when a cell is added or retired. A session that adds a
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

**Legs 1 and 2 are the matrix**; leg 3 is not an architecture and ships no artefact, so
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
  **shipped artefact**; these legs need it for the **binary under test**, and a native build is
  exactly where the toolchain's default is *on*. An unset value runs the corpus against a binary
  that may resolve through the system resolver — a second answer path ADR-0070 has already closed —
  and the corpus would read green while verifying the wrong instrument.

### 3.2 The assertions, in the order they fail

| # | Assertion | Scope | Why it sits here |
| --- | --- | --- | --- |
| A1 | **Self-identity** — each leg runs the corpus **twice in one process**; output must be byte-identical | Every leg | Go randomises map iteration **per iterator**. An unstable corpus makes every assertion below uninterpretable, and a red `arm64` leg indistinguishable from a flake |
| A2 | **Expectation** — each leg's output equals the **one shared** expected-output artefact | Every leg | ADR-0021's gate, first direction: *output moved and the version did not → fail* |
| A3 | **Cross-architecture identity** — legs 1 and 2 agree | Legs 1, 2 | A2 gives it by transitivity; it is asserted separately so the failure reads *architecture divergence* rather than *row 47 moved* |
| A4 | **Contraction differential** — leg 3 equals leg 1 | Leg 3 | The only check in the repository that a declared parameter's fraction was actually evaluated in exact integer arithmetic. A difference names the fraction that escaped |
| A5 | **Coverage** — every cell of §2 holds at least one row | Once | This is what gives the obligation an owner. A missing cell fails the build and the failure **names the cell** |
| A6 | **Gate direction 2** — on `resolution-walk`, a version bump with no moved pin row, no changed declared parameter and no recorded uncovered move fails | Once | ADR-0021's second direction, now with the evidence that lets it fire. This is the assertion that protects `returned` |

**There is no per-architecture expected output, ever.** An `arm64` golden file is the divergence
#124 exists to prevent, written down and blessed — and it is the first repair a session will reach
for when leg 2 goes red. A2's *one shared artefact* is what makes A3 mean anything.

**A6's scope widens to a second leaf if and only if §7's routing question answers that way.** It is
written against `resolution-walk` today because that is the only leaf whose membership role is
settled.

### 3.3 The trigger

**Every pull request and every release. Every leg, every assertion.**

The named failure mode arrives on a dependency PR — the `go.mod` cadence
[#49](https://github.com/winniel123/verge-asm/issues/49) put `resolution-walk` on — and the tempting
design is a `go.mod`-touched path filter. It is refused. The corpus is hermetic by construction, so
there is nothing to save, and a path filter is a second place the obligation can be silently
disabled. Running everything on everything makes the dependency case unremarkable, which is the only
way to be sure it is covered.

---

## 4. Who owes it, who owns it, what enforces it

| Role | Party |
| --- | --- |
| **Owes** | **The release**, per [ADR-0041](../adr/0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md) §*What the release owes* — unchanged |
| **Owns** | **§2's enumeration**, checked in as data. An obligation with no failing test has no owner |
| **Enforces** | **A5** (a cell with no row fails, naming the cell) and **A6** (an unjustified bump on `resolution-walk` fails) |
| **Renders on failure** | The row's **claim**, the old output and the new one — ADR-0021's judgeability property, unchanged |
| **Undischarged** | The `Shadowed` outcome — **§7**, and it has no owner until the routing question is answered |

---

## 5. What #140 settled

[#140](https://github.com/winniel123/verge-asm/issues/140) **confirmed** ADR-0041 rather than
reversing it: a withdrawn subject's timelines **close**. So Block R stands, and A6 is a hard refusal
rather than a redundancy with ADR-0021's existing second gate direction — a `Break` between
withdrawal and return leaves the reopening with nothing legally before it, and an unjustified bump
costs `returned` estate-wide and permanently.

Two corollaries #140 added that this file depends on:

- **The withdrawn period is on no timeline at all** — neither a value nor a `Gap`. That is what
  leaves the span before withdrawal and the span after a return **adjacent**, and it is why
  `returned` is derivable by ordinary machinery wherever no `Break` sits between them. R.1 pins the
  leaf side of it: the leaf emits two outcomes and names no transition.
- **A closed span is free to keep and an open one must be fed.** That asymmetry is why the `Span`
  corpus is never compacted, and it is the reason the pin block's protection is worth having: there
  is no storage-side repair for a lost `returned`, because there was never a storage-side cost to
  keeping what would have proved it.

Had #140 gone the other way, Block R's cells would have retired and A6 would have degraded to
redundancy. Blocks M1–M3 would not have moved either way — they pin the leaf's outcomes, and
membership is decided from those outcomes identically under both readings.

---

## 6. Where this is thin

- **The nine boundaries in §2.2 are reasoned from the DNS specifications, not measured against two
  DNS libraries.** Nobody has diffed two implementations across them. Every row in that block
  therefore carries ADR-0021's spec-verified marker until somebody measures it.
- **Completeness of §2 is not checkable by anything.** A5 counts rows against the enumeration; it
  cannot tell anyone the enumeration named every ambiguous place on the wire. That residue is
  ADR-0021's **uncovered move**, and it is disposed of the same way — recorded as data, reviewable
  and countable, and if it becomes the common case the corpus is failing and the count says so.
- **A4's yield is unmeasured.** No fraction in the declared-parameter set has been shown to diverge
  at `GOAMD64=v3`. If A4 never fires, that is indistinguishable from A4 being unnecessary, and the
  argument for keeping it is that the check is cheap and the failure it catches is silent.
- **That this project's CI will be GitHub Actions is assumed**, no ticket having decided it. The
  legs, the assertions and their order transfer unchanged if it is not; only the runner labels move.

---

## 7. The unrouted fifth outcome — `Shadowed`

**This section is escrow. Nothing in it is discharged, and none of it may be filed into
`resolution-walk`'s corpus.**

### 7.1 The defect

[ADR-0041](../adr/0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md)'s
obligation 2 names five outcomes to pin — `Resolved`, `NoData`, `NameError`, `Lame`, `Shadowed` —
and scopes them to *`resolution-walk`'s golden corpus*. Four are that leaf's.
[ADR-0021](../adr/0021-a-version-leaf-is-a-decision-not-a-binary.md)'s leaf table puts the fifth,
**`Shadowed`, under `wildcard-discrimination`** — a leaf ADR-0021 kept separate on purpose, *"so a
break **names its leaf**"*.

### 7.2 The two candidate homes, and why it cannot be neither

| Home | The claim it requires | Its price |
| --- | --- | --- |
| **A — `wildcard-discrimination`'s own golden corpus** | The membership vector is **wider** than ADR-0041 and [`CONTEXT.md`](../../CONTEXT.md) record: `Shadowed` is a value on `resolution`, under it a `Name` *cannot leave at all*, and bumping the leaf that decides it `Break`s the timeline membership reads | **`wildcard-discrimination`'s control-label count has already moved** — `5` → `9`, [#115](https://github.com/winniel123/verge-asm/issues/115) — which ADR-0021 records as bumping the leaf and `Break`ing `resolution`. If the leaf is in the vector, that move broke **every** `Name`'s membership timeline, and nobody priced it |
| **B — nowhere; the obligation for `Shadowed` is void** | The vector is exactly `resolution-walk` as recorded, and `Shadowed` **withholds a value without deciding presence** — a name under it is *visibly unconfirmed* rather than absent, so no membership fact turns on it | ADR-0041's obligation 2 is wrong to list it, and must be amended at its own site to a four-outcome list. The escrow below is discarded rather than filed |

**It cannot be neither.** One of ADR-0041's sentence and ADR-0021's leaf table is inaccurate, and
which one is a question with a real consequence attached. [#140](https://github.com/winniel123/verge-asm/issues/140)
has raised it as a successor ticket. **This file does not answer it**, because a decision lives in
exactly one place and that place is the successor.

### 7.3 The escrow — 17 cells, written and not filed

If the routing question answers **A**, these seventeen cells are adopted whole into
`wildcard-discrimination`'s corpus and A5's coverage assertion extends over them. If it answers
**B**, they are discarded. Either way the successor rules the routing and does not re-derive the
rows.

| Block | Cells | Contents |
| --- | --- | --- |
| W1 — component signatures | 3 | One per member of the per-component signature union, per `(qtype asked, RR type in the answer)` ([ADR-0068](../adr/0068-a-wildcard-is-discriminated-only-where-its-synthesis-is-determinate.md)): `NoSynthesis` · `Determinate(RRset)` · `Indeterminate` |
| W2 — verdicts | 2 | `Shadowed` · not-`Shadowed`, the latter being a name that **differs at some determinate component** |
| W3 — boundary pins | 8 | Four boundaries, two cells each: **(a)** differs at a **determinate** component against differing **only** at an `Indeterminate` one, which is never consulted; **(b)** **no** component determinate, so every name beneath the parent is `Shadowed`, against at least one determinate-and-differing; **(c)** a probe that **completed** and found no wildcard, licensing everything beneath, against one that **did not complete** — a `Gap`, never a value, since *an undiscriminated answer is never a value*; **(d)** discriminated at **one** component, so no synthesised RRset at **any** qtype, against `Shadowed` holding on `resolution` and on every `dns-record` discriminator |
| W4 — control-label set | 2 | **(1)** the set is **9 random + 1 structured** label, each **exactly one label**, the structured one `<a>-<b>-<c>-<d>` over a random RFC 5737 documentation address ([ADR-0069](../adr/0069-a-control-label-is-one-label-and-the-set-must-falsify-label-independence.md), [#115](https://github.com/winniel123/verge-asm/issues/115)), with a two-label label as the negative case since it falls off the tree at a **deeper encloser** and measures a different wildcard; **(2)** the labels are drawn **per `Batch`** and are **independent samples** — the measured mechanisms being per-label sharding and per-query rotation, never per-time ([`passive-discovery-sources.md`](../research/passive-discovery-sources.md) §13) |
| W5 — the shared path | 1 | A control probe asked from a **different** place than the answer it discriminates yields **no verdict**, never a wrong one. **[measured]** — direct to its own authority, `s3.amazonaws.com` reads a *determinate* `NoSynthesis` at A while a resolver answers every candidate beneath with eight addresses, so a skewed pair discriminates every fictional label and records it `Resolved` with a fabricated set ([ADR-0070](../adr/0070-a-control-probe-is-asked-from-where-the-answer-it-discriminates-was-asked-from.md)) |
| W6 — the suppression | 1 | `Shadowed` where a `NameError` would otherwise have been measured. Expected: `Shadowed`, and **no withdrawal-shaped output**. This is the cell that makes the membership role concrete, and it is the one the routing question is really about |
| **Total** | **17** | |

### 7.4 Why the rows are not filed anyway

**A corpus row protects the leaf whose gate runs it, and nothing else.** A `Shadowed` row sitting in
`resolution-walk`'s corpus is checked when `resolution-walk`'s version is questioned and is silent
when `wildcard-discrimination` bumps — the leaf that actually decides the value. The row would be
exercised by a leaf that cannot move it and ignored by the leaf that can.

That is not a misfiled row. It is a row that **reads green while protecting nothing**, and it would
make A5 report the obligation discharged. An absent cell is countable; a falsely-filed one is not.
