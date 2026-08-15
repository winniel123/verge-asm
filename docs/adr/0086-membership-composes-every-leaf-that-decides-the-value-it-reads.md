# ADR-0086: Membership composes every leaf that decides the value it reads — `wildcard-discrimination` is in the vector

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#146 Is `wildcard-discrimination` in the membership vector?](https://github.com/winniel123/verge-asm/issues/146)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)
- **Spec content:** [`golden-corpus.md`](../spec/golden-corpus.md) §8 — the escrow, adopted

## Context

[ADR-0041](./0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md) records that
membership composes `resolution-walk` and nothing else, and its third obligation forbids a release
from widening that vector. [`CONTEXT.md`](../../CONTEXT.md)'s `Transition` entry says the same
sentence. Both are wrong, and this ADR is where that is said.

Three prior tickets walked up to the question and each correctly declined it.
[#140](https://github.com/winniel123/verge-asm/issues/140) ·
[ADR-0082](./0082-a-withdrawn-subjects-timelines-close-and-the-withdrawn-period-is-on-no-timeline.md)
raised it as an open item against obligation 3 and ruled only that a withdrawn subject's timelines
close. [#143](https://github.com/winniel123/verge-asm/issues/143) ·
[ADR-0085](./0085-an-obligation-with-no-failing-test-has-no-owner-and-a-boundary-needs-a-row-on-each-side.md)
found that ADR-0041's five-outcome pin list **spans two leaves while naming one**, discharged the
four that are genuinely `resolution-walk`'s, and put the fifth — `Shadowed` — into a written escrow
of seventeen cells rather than filing it against a gate that cannot move it. Its own alternatives
table names the ruling it would not make: *"`wildcard-discrimination` is in the membership vector, on
the ground that a bump `Break`s `resolution`* — **it is a defensible reading and it is not this
ticket's to make."*

The constraint that makes the question hard is that it cannot be answered by shrugging. **One of
ADR-0041's sentence and ADR-0021's leaf table is inaccurate**, and there is no third option: either
`Shadowed` is not `wildcard-discrimination`'s output, or the vector is wider than the record says.
ADR-0021's leaf table is not in doubt — the two leaves were split deliberately *"so a break names its
leaf"* — so the pressure lands entirely on ADR-0041's sentence.

Two riders arrive with the ticket and both are binding.

- **A `Break` fires on the vector moving, never on the value moving.** So *"a bump cannot flip
  presence"* answers nothing: the question is whether the leaf belongs to the procedure that decided
  a presence value, not how often its output differs.
- **The retroactive consequence is unpriced.** [#115](https://github.com/winniel123/verge-asm/issues/115)
  moved this leaf's control-label count `5` → `9`, which ADR-0021 records as bumping the leaf and
  `Break`ing `resolution`. If the leaf is in the vector, that move broke **every** `Name`'s membership
  timeline. That price is paid below rather than deferred again.

## Decision

**`wildcard-discrimination` is in the membership vector, for a `Name` and for a cited `Address`
alike. The membership vector is exactly `resolution-walk` **and** `wildcard-discrimination` — the two
leaves that decide the `resolution` value membership reads — and it is still the *narrowest* vector
that decides presence, because those are the only two leaves that decide it.**

The general rule this rests on, stated so a later session does not re-derive it one facet at a time:

> **A derivation composes every leaf that decides the value it reads.** Where a value is decided by
> two leaves jointly, a reader of that value composes the **union** of them. There is no reading of
> half a decision procedure.

| Concern | Decision |
| --- | --- |
| Is `wildcard-discrimination` in the membership vector? | **Yes.** For a `Name` and for a cited `Address` |
| The vector, stated | `Name`: `resolution-walk` + `wildcard-discrimination`. Cited `Address`: the same two. `Seed`-covered `Address`: **nothing at all**, unchanged |
| The deciding ground | **`Shadowed` cites no `Address`.** The verdict decides whether an answer's address set enters the estate, which is a cited `Address`'s membership **affirmatively** rather than by suppression |
| The ticket's own ground — suppression of `NameError` | **True, and the weaker of the two.** It is reachable only beneath an unstable parent that does not synthesise, a shape nobody has measured. The ruling does not rest on it |
| Is this a *widening* under ADR-0041 obligation 3? | **No.** Obligation 3 binds a **release**; this is a correction of the record. The vector is **discovered** from what decides presence, never declared |
| The rider obligation 3 gains | A leaf omitted from the record was never thereby outside the vector. Without it, obligation 3 is satisfied by editing a sentence |
| Which of ADR-0041's two sentences was wrong | Obligation 2's **scope**, not its **list**. All five outcomes are membership-deciding; scoping them to *`resolution-walk`'s golden corpus* is the defect |
| The retroactive price of #115's `5` → `9` | **Zero, and the reason expires.** Nothing has shipped and no `resolution` timeline exists — ADR-0068's own precedent row. It is free exactly once |
| The recurring price | Three project-authored parameters — control-label count, construction, match predicate — reach membership that did not before. **No new dependency does**: the DNS library and the query path are already parameters of both leaves |
| The new dependency exposure, stated exactly | A library change that moves discrimination but **not** resolution now breaks membership where it previously did not. That is what §8's cells make provable |
| ADR-0085 §7's escrow | **ADOPTED WHOLE.** Seventeen cells, unamended, filed into `wildcard-discrimination`'s corpus as [`golden-corpus.md`](../spec/golden-corpus.md) §8 |
| What this ruling adds beyond the escrow | **One boundary pair, W7 — the citation pin.** The escrow has no cell for the ground the ruling turns on |
| The pin block's size | `resolution-walk` **27** · `wildcard-discrimination` **19** (17 adopted + 2 new) · escrow **0**. Total **46** |
| A5 — coverage | Extends over §8. A missing cell in either block fails the build and names the cell |
| A6 — gate direction 2 | **Widens to `wildcard-discrimination`**, exactly as `golden-corpus.md` §3.2 said it would *"if and only if §7's routing question answers that way"* |
| ADR-0021's *"the honest default is to bump"* | True for **three** leaves (`connect-outcome`, `tls-handshake`, `http-exchange`), **false for two** (`resolution-walk`, `wildcard-discrimination`) |
| ADR-0021's *no leaf is composed by every timeline* | **Survives.** `wildcard-discrimination` composes `resolution`, `dns-record` and membership, and touches `certificate`, `http-identity` and `reachability` not at all |

## Rationale

### The ground the ruling turns on is the cited `Address`, and the ticket did not ask about it

The ticket asks the question on the `Name`: under `Shadowed` a `Name` *cannot leave at all*, so does
deciding that a subject does not leave count as deciding presence? That framing invites a long
argument about whether suppression is a decision. It is the wrong place to look, and the right place
is one line away in the glossary.

**`Shadowed` cites nothing.** [`CONTEXT.md`](../../CONTEXT.md)'s own entry says it: *"it admits
nothing, since an answer served for a name that does not exist may cite no `Address` and open no
`Endpoint`."* And a cited `Address` *"is in the estate exactly while a current resolution cites it or
a `Seed` covers it."*

Put those two sentences together and the conclusion is immediate and needs no theory of suppression:

> Beneath a wildcarding parent, `resolution-walk` measures `Resolved(set)`. If the verdict is
> not-`Shadowed`, that set is the recorded value and it **cites** — the `Address`es enter the estate,
> `Endpoint`s open beneath them, `Exposure` reads them. If the verdict is `Shadowed`, the recorded
> value cites **nothing** and every `Address` held only by that citation **leaves the estate**.

That is not a suppressed departure. It is a departure, caused by this leaf's verdict, on the second
of the two membership-bearing subject kinds ADR-0041 traced to `resolution-walk`. The glossary
already states the consequence in the direction that alarms — *"a false `Resolved` fabricates an
address set that cites `Address`es, opens `Endpoint`s and feeds `Exposure`"* — which is the model
saying out loud, in the entry for this very value, that this leaf's verdict decides which `Address`es
are in the estate. Nobody had read it as a statement about the vector.

**And it is measured that moving the leaf's declared parameter moves that verdict.**
[ADR-0068](./0068-a-wildcard-is-discriminated-only-where-its-synthesis-is-determinate.md) records
`surge.sh` as a two-member hash of the query label, `188/172` over 360 labels: **five random labels
land in one shard 3 times in 30**, reading a false `Determinate` and discriminating names that a
nine-label draw reads `Indeterminate`. So the exact parameter #115 moved, `5` → `9`, changes which
names beneath a measured zone are `Shadowed`, and therefore which addresses are cited, and therefore
which `Address` subjects are in the estate. The presence flip is not hypothetical and it is not
reasoned from the specification; it is in the repository, measured, in the ADR that valued the
predicate.

### The ticket's own ground is true and is the weaker one, and saying so is the review

The `Name` case survives the same test but on thinner evidence, and it would be dishonest to present
it as the load-bearing half when it is not.

A `Name` withdraws exactly where `resolution` reads `NameError` at a cross-class `Vantage
composition`. `Shadowed` overwrites whatever `resolution-walk` measured. So the flip that decides a
`Name`'s presence is **`NameError` recorded as `Shadowed`** — and that pair is hard to reach, because
of the protocol rather than because of the model. RFC 4592 makes a covering wildcard answer NOERROR,
so beneath a parent that synthesises, a deleted name reads `Resolved` or `NoData` and never
`NameError`; and where the parent does **not** synthesise, ADR-0066's *a probe that completed and
found no wildcard licenses everything beneath it* exempts the name and the `NameError` stands.

The residue is the parent that neither synthesises nor reads determinate — every component
`Indeterminate`, so *"where no component is determinate every name beneath that parent is
`Shadowed`"*, over a name the authority is answering NXDOMAIN for. It is reachable and **nobody has
measured one.** Every indeterminate zone in ADR-0068's nineteen is a hosting provider that
synthesises.

So the honest statement is: **the `Name` ground is sound and unmeasured; the `Address` ground is
sound and measured.** The ruling is the same either way, because the vector is one vector — a `Name`
and the `Address`es it cites are decided from one `resolution` value by one pair of leaves, and a
vector that held for one and not the other would have to be recorded per subject kind, which is a
complication bought with nothing.

### Reading a value composes the leaves that decided it — the structural argument, which needs no flip at all

Both grounds above are arguments from consequence: *the verdict can move presence.* There is a
stronger argument that does not depend on finding a reachable flip, and it is the one that should
survive if a later session breaks my reachability analysis.

[ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md) already rules it, for signals, and
the rule is general: *"Where a rule reads two timelines from two sources it composes the **union** of
their leaves, so a source with no prober leaf — an operator's zone file — does not shield the rule
from the other side's bumps."*

Membership reads `resolution`. `resolution` is decided by **two** leaves — `resolution-walk` decides
`NameError │ NoData │ Lame │ Resolved`, `wildcard-discrimination` decides `Shadowed`, and the
recorded value is one or the other. There is no projection of `resolution` that membership could read
which only one leaf reaches, because the leaves do not partition the value space by facet or by
discriminator; they partition it by **which of them got to decide this observation**. A reader of the
value composes both, exactly as a reader of two timelines composes both.

This is also what keeps the answer narrow, and the narrowness is obligation 3's whole point.
`connect-outcome`, `tls-handshake` and `http-exchange` decide values membership does not read, so
they stay out — the vector is two leaves and not five, and *narrowest* still means something.

### `Shadowed` is epistemic, and that is not a route out

The best form of the losing option is that `Shadowed` is a statement about our own sight rather than
about the world. The glossary supports it directly: `Predicate domain` calls `not-evaluable` *"a
value about **our own sight** (`Shadowed`)"*, and
[ADR-0014](./0014-only-revealed-generalises.md) keeps aperture — a property of *looking* — out of
membership, which is a property of a *subject*. On that reading `wildcard-discrimination` decides
what we can see and `resolution-walk` decides what is there, and only the second is presence.

It loses on three independent grounds.

**The model already refused to make it the object that would have kept it out.** A `Gap` is the
object for *we could not say*, and a `Gap` *"never withdraws a subject."* The model considered that
home for exactly this fact and split it: where the control probe **did not complete**, the name
records a `Gap`; where it completed and the answer was not discriminated, the name records
`Shadowed`, *"recorded as a measured value rather than discarded."* Two adjacent cases, deliberately
given two different objects, and only one of them is the one that never withdraws a subject.
`Shadowed` is on the value side of a line the model drew on purpose.

**Epistemic content does not make a value inert.** A value that changes which subjects are in the
estate is a membership input whatever its content is about, and `Shadowed` changes it — that is the
citation argument above, and it is measured. ADR-0014's line is about **aperture**, which is recorded
on the `Batch` and per-timeline; `Shadowed` is recorded on a facet timeline as a value with a `Span`,
and it is not in the aperture list — ADR-0068 checked that explicitly and left the count at seven.

**And it would leave the failure ADR-0085 exists to prevent standing, one leaf over.** Under the
losing option `Shadowed` needs no membership pin at all, so a `wildcard-discrimination` bump on a
dependency upgrade would silently change which `Address`es are in the estate with nothing pinning the
outcome and nothing failing. That is ADR-0085's named shape — estate-wide, silent, triggered by a
no-op upgrade — relocated rather than answered.

### Pricing #115, which is the thing three tickets deferred

The retroactive consequence is **zero**, and the reason is a fact about the calendar rather than
about the model.

Nothing has shipped. There is no instance, no estate, no `resolution` timeline and no `Span`, so
there is no membership timeline for the `5` → `9` move to have broken. ADR-0021 already says it of
that very move — *"free while nothing has shipped"* — and
[ADR-0068](./0068-a-wildcard-is-discriminated-only-where-its-synthesis-is-determinate.md) prices its
own ruling the same way in its decision table: *"Cost of the ruling: **Zero.** Nothing has shipped and
no `resolution` timeline exists."* This ADR is the third to draw on that credit and it should be the
last, because **it is free exactly once and the freedom ends at the first shipped release.** After
v1.0 the same move costs `returned` across every `Name` and every cited `Address`, permanently and
unrepairably.

The recurring price is the number worth having, and it is smaller than the framing suggests. The
vector gains one leaf; it does **not** gain a new dependency. ADR-0021 puts the DNS library and the
query path in the declared parameters of **both** leaves, so every dependency-cadence and
`Vantage`-cadence cause that reaches membership through `wildcard-discrimination` already reached it
through `resolution-walk`. What is genuinely new is three parameters — **control-label count,
control-label construction, and the match predicate** — and all three are **project-authored**. They
move on our roadmap, not on `go.mod`'s, which means the release can see the break coming and
obligation 1's *state the loss* is dischargeable rather than retrospective. That is a materially
cheaper class of hazard than the one ADR-0041 and ADR-0082 wrote obligation 3 against.

One thing genuinely gets worse and it is stated rather than netted off: **the same DNS library now
reaches membership through two gates instead of one.** A library change that moves discrimination but
not resolution bumps `wildcard-discrimination` alone and breaks membership where it previously did
not. That is precisely what §8's nineteen cells make provable, and it is the reason the escrow is
adopted rather than merely re-homed.

### Obligation 3 is not violated, and it needs a rider or it never can be

The reflex objection is that this ADR does the thing ADR-0041 forbids. It does not, and the
distinction is worth writing down because it will be reached for again.

Obligation 3 binds a **release**: do not add a leaf to the set of things that decide presence,
because the cost lands on `returned` rather than on the versioning ledger. This ADR adds nothing to
what decides presence. `wildcard-discrimination` has decided the `resolution` value since ADR-0021
named it, and `Shadowed` has cited no `Address` since the glossary said so. What moves is the
**record**, which was short by one leaf.

But that distinction is only usable with a rider, and without it obligation 3 is unfalsifiable — a
release could satisfy *do not widen the vector* by leaving a leaf out of the sentence. So:

> **The membership vector is discovered from what decides presence, never declared.** A leaf omitted
> from the record was never thereby outside the vector, and finding one is a correction rather than a
> widening. What obligation 3 forbids is a release **making** a leaf decide presence that did not.

### The escrow is adopted whole, and it is one pair short

ADR-0085 wrote §7's seventeen cells *"to be adopted or discarded whole"* by this ticket, and priced
its own risk honestly: *"if the routing question answers that `Shadowed` decides no subject's
presence, the correct number is zero and the escrow is discarded... that is the cheaper error of the
two available."* The routing question answers A. **The seventeen are adopted whole and unamended**,
and the escrow's bet paid — the successor ruled the routing and re-derived no rows, which is exactly
what escrow is for.

Adopting whole is not the same as finding the block complete, and it is not. **W6 — `Shadowed` where
a `NameError` would otherwise have been measured** — is the cell ADR-0085 called *"the one the
routing question is really about"*, and it pins the ticket's own ground, the `Name` suppression case.
The ground this ruling actually turns on has **no cell at all**: nothing in the seventeen pins that
`Shadowed` carries no address set and therefore cites no `Address`.

Leaving that unpinned would reproduce ADR-0085's own defect one level down — a block that reads green
while the ruling's load-bearing claim is unprotected. So one boundary is added, as two cells, per
ADR-0085's *a boundary is pinned by a pair of rows, one on each side*: **W7, the citation pin.** It is
new content this ruling forces, not an amendment to the escrow, and it is labelled that way in §8 so
a reviewer can see which seventeen were inherited and which two were minted here.

## Consequences

- **[`CONTEXT.md`](../../CONTEXT.md) is amended in one entry and gains no term.** `Transition`'s
  *"a `Name`'s and a cited `Address`'s membership both compose `resolution-walk`"* is superseded at
  the site that specifies it, per
  [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md): read alone
  and in the present tense it states a one-leaf vector, which is the record this ADR corrects. The
  `Seed`-covered `Address` half of that sentence is untouched and stays true.
- **[`golden-corpus.md`](../spec/golden-corpus.md) gains §8 and is amended in five places.** §1's
  leaf table, §2.5's count, §3.2's A6 scope note, §4's *Undischarged* row and §7's escrow banner all
  named a routing question that is now answered. The file's totals move **27 + 17 escrow → 46
  discharged, 0 escrow**.
- **ADR-0041's obligation 2 is amended at its own site**, and the amendment is to its **scope**
  rather than to its list. Its five outcomes are all membership-deciding; *`resolution-walk`'s golden
  corpus* is the half that was wrong. ADR-0085 routed this correction to the ticket that owns it and
  this is that ticket.
- **ADR-0041's obligation 3 gains the discovery rider** and is otherwise unchanged. It still binds,
  it still forbids what it always forbade, and it is now falsifiable.
- **[ADR-0082](./0082-a-withdrawn-subjects-timelines-close-and-the-withdrawn-period-is-on-no-timeline.md)'s
  open item is discharged at the clause that raises it.** Its ruling that a withdrawn subject's
  timelines close is untouched and is what makes this one cost anything.
- **[ADR-0085](./0085-an-obligation-with-no-failing-test-has-no-owner-and-a-boundary-needs-a-row-on-each-side.md)
  is amended at four sites** — the routed-fifth-outcome section, the *fifth outcome* and *pin block*
  decision rows, the alternatives-table row that declined this ruling, and the thin-ground bullet
  pricing the escrow. Its central rule — *a row protects the leaf whose gate runs it* — is what makes
  the adoption correct rather than a re-filing.
- **[ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md) is annotated in two places.** The
  `wildcard-discrimination` row of the leaf table gains that this leaf reaches membership, and
  *"the release where we cannot tell"* gains its second exception. **The leaf count stays five** and
  the leaf table's split stays exactly as it is — this ADR is a reason the split was right, since a
  merged leaf would have put the whole membership vector on one name and lost the ability to say
  which half moved.
- **ADR-0006 and ADR-0008 carry the one-leaf sentence in their ADR-0041 annotations** and are amended
  at those clauses, per ADR-0058 and per #106's rule that a document supersedes itself. Neither
  ruling moves; both name `resolution-walk` alone where the vector has two leaves.
- **[ADR-0072](./0072-absence-is-a-property-of-a-cell-and-withdrawn-is-the-only-population.md)'s
  floor-obligation clause is amended by a phrase.** *A `Name`'s membership composes `resolution-walk`,
  so it clamps and reads `≥`* keeps its conclusion — it still clamps — and gains the second leaf, so
  the `Subjects` column's two renderings are unchanged.
- **No aperture input is added and the count stays seven.** ADR-0068 already checked that determinacy
  decides what a covered subject's answer means rather than which subjects were covered, and nothing
  here moves that.
- **No new object, value, parameter or transition name.** The vector is a set and it gains a member
  that was already deciding.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **B — `Shadowed` withholds a value without deciding presence; the vector is `resolution-walk` alone and ADR-0041's obligation 2 is amended to a four-outcome list, the escrow discarded** | **The losing option, and it is a genuinely good one — it is what the record currently says, it is cheaper, and `Shadowed` really is epistemic.** It loses on the citation: `Shadowed` cites no `Address`, so the verdict withdraws every `Address` held only by that citation, which is a departure and not a suppressed one — and ADR-0068's `surge.sh` measurement shows the leaf's own declared parameter moving that verdict. It also loses structurally, because membership reads `resolution` and `resolution` is decided by two leaves; and it would leave ADR-0085's named failure — estate-wide, silent, on a no-op upgrade — standing unpinned one leaf over |
| **Rule the vector wider for the cited `Address` and not for the `Name`, since only the first ground is measured** | Defensible on the evidence and refused on the object. One `resolution` value is decided by one pair of leaves and read by both populations; a per-subject-kind vector would have to be recorded, rendered and compared per subject kind, which is complication bought with nothing. The `Name` ground is sound and unmeasured, which is a note in the thin section rather than a second vector |
| **Merge `resolution-walk` and `wildcard-discrimination` into one leaf, since both are now in the vector** | ADR-0021's closest call, and this ADR is a reason it went the right way. A merged leaf puts the whole membership vector under one name, so a break could no longer say whether resolution or discrimination moved — and it would drag `resolution-walk`'s dependency-cadence bumps onto discrimination's corpus and back again. The split costs a line in the vector and buys the sentence an operator can act on |
| **File the `Shadowed` rows into `resolution-walk`'s corpus now that both leaves are in the vector** | Being in one vector is not being one gate. ADR-0085's rule is untouched: a row protects the leaf whose gate runs it, and a `Shadowed` row under `resolution-walk`'s gate is silent when the leaf that decides the value bumps. The rows go to `wildcard-discrimination`'s corpus, which is what adoption means |
| **Adopt the escrow and stop there, leaving W7 unwritten** | The escrow pins the ticket's framing and not the ruling's ground. A block with no cell for *`Shadowed` carries no address set* reads green while the load-bearing claim is unprotected, which is ADR-0085's own defect reproduced one level down |
| **Discard the escrow and re-derive the rows against the ruling** | Wasteful and worse: seventeen cells written by the ticket that priced both homes are better evidence than seventeen written by the ticket that had already picked one. Escrow exists to be adopted whole, and adopting it whole is also the honest test of whether escrow works |
| **Defer again — the `Name` ground is unmeasured, so wait for a measurement** | Two tickets have already deferred it and each deferral moved the defect rather than shrinking it. The `Address` ground needs no new measurement, the retroactive price is zero **only while nothing has shipped**, and deferring past v1.0 converts a free correction into an estate-wide one |
| **Treat `Shadowed` as aperture, so it lands on the `Batch` and out of membership by ADR-0014** | ADR-0068 explicitly declined to move the aperture list for determinacy, on the ground that it decides what a covered subject's answer *means* rather than which subjects were covered. `Shadowed` holds a `Span` on a facet timeline; aperture does not |
| **Rule it out of the vector and rely on ADR-0021's second gate direction to stop the bump** | ADR-0085 answered this for `resolution-walk` and the answer transfers: the gap was never the rule, it was the evidence. A gate that forbids an unjustified bump does nothing when no row would have moved to justify one |

## Where this is thin, stated rather than smoothed

- **The `Name` ground is reasoned and not measured, and the ruling is written so that it does not
  matter.** The `NameError`-recorded-as-`Shadowed` flip needs a parent that neither synthesises nor
  reads determinate at any component, and every indeterminate zone in ADR-0068's nineteen
  synthesises. If somebody proves that shape unreachable, the `Address` ground and the structural
  argument both stand and nothing in the decision table moves — but the *stated* reason for the
  `Name` half would then be the structural one alone, which is weaker prose and the same ruling.
- **The reachability analysis is mine and rests on RFC 4592's NOERROR behaviour plus ADR-0066's
  no-wildcard licence.** It is specification-derived, so it inherits ADR-0021's honesty rider: any
  §8 row encoding it says so on the row. Nobody has taken a wildcarding authority and a deleted name
  and watched what comes back.
- **The retroactive price of zero is a fact about the calendar and not about the model, and this is
  the third ADR to spend it.** ADR-0068 and ADR-0021 both priced a `wildcard-discrimination` move at
  zero on the same ground. The credit is real and it is finite, and a session reading any of the three
  after v1.0 must not read *free* as a property of the move.
- **W7 is minted here rather than escrowed, so it has had no second reader.** The seventeen adopted
  cells were written by ADR-0085 and reviewed by this ticket; the two new ones have been written and
  ruled by one session. They are the block's most load-bearing pair and the least examined.
- ~~**`returned`'s predicate across a two-leaf vector is not settled here and is not mine.** A subject
  holds many timelines whose last spans may have closed under different vectors, and the vector now
  has two members that can move independently — so *closed under a different vector* has a second way
  to happen. [#148](https://github.com/winniel123/verge-asm/issues/148) owns it, and it reads this
  ruling rather than the other way round.~~
  **SETTLED** by [#148](https://github.com/winniel123/verge-asm/issues/148) ·
  [ADR-0097](./0097-returned-composes-every-witness-a-presence-read-rests-on.md): `returned` requires
  no `Break` on any witness a presence read relies on — checking both leaf components this ADR adds —
  conjoined across every witness [ADR-0080](./0080-a-vantage-composition-is-cross-class-or-class-scoped-and-only-one-takes-a-quantifier.md)'s
  quantifiers select. Nothing in this ADR's decision table moves.
