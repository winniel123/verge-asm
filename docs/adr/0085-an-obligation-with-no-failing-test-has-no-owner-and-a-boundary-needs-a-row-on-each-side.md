# ADR-0085: An obligation with no failing test has no owner — and a boundary is pinned only by a row on each side

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#143 `resolution-walk`'s golden corpus owes rows pinning the membership-deciding outcomes](https://github.com/winniel123/verge-asm/issues/143)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)
- **Spec content:** [`golden-corpus.md`](../spec/golden-corpus.md) — the enumeration and the matrix, which will be revised

## Context

[ADR-0041](./0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md) §*What the
release owes* wrote down a failure mode in full and gave it nobody to fail on:

> a DNS-library upgrade that changes retry timing but cannot change whether a name resolves must
> **provably** not bump the leaf — and *provably* means the corpus carries rows that pin the
> membership-deciding outcomes specifically: `Resolved`, `NoData`, `NameError`, `Lame`, `Shadowed`.

The failure it names is the worst shape this project has: **estate-wide, silent, and triggered by a
no-op upgrade.** A dependency bump moves `resolution-walk`'s leaf. The leaf is inside the vector the
membership timeline composes. A `Break` between a subject's withdrawal and its return leaves the
reopening with nothing legally before it. The membership message fires reading **`appeared`** when
the truth is `returned`, and history is never re-derived so it cannot be corrected afterwards.

Three things had not been done, and the ticket exists because of the third.

**The obligation has no home.** It is a sentence inside a retention ADR. Nothing in the repository
counts it, nothing fails when it is unmet, and the only reader who would ever discover the gap is
the one already suffering it.

**The obligation has no content.** *Pin the membership-deciding outcomes* names five words. It does
not say how many rows, against what stimuli, or how a reviewer knows the set is complete.

**And [#124](https://github.com/winniel123/verge-asm/issues/124) added a second, independent reason
the same corpus must be pinned, on an axis nobody had connected to this one.** Go's specification
permits fused multiply-add, Go's `arm64` backend emits it and baseline `amd64` has none, so
[`packaging-and-configuration.md`](../spec/packaging-and-configuration.md) §1.2 ruled that **an
architecture is in the matrix exactly where the golden corpus is run on it in CI**. That makes the
corpus the instrument deciding *which architectures ship*, which is a release question, on the same
artefact that decides whether `returned` survives a dependency upgrade. Retention was already
*"partly a release question"*. This is why.

### One of the five outcomes is not this leaf's, and that is a defect rather than a detail

ADR-0041's list is five outcomes long and the ADR scopes it to *`resolution-walk`'s golden corpus*.
But [ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md)'s leaf table puts **`Shadowed`
under `wildcard-discrimination`**, a separate leaf, kept separate on purpose *"so a break **names its
leaf**"*. Four of the five are `resolution-walk`'s. The fifth is not.

That cannot be papered over, because the two ways of repairing it have different prices and one of
them is expensive:

- **Either the rows belong to a different leaf's corpus** — in which case `wildcard-discrimination`
  is in the membership vector, which [`CONTEXT.md`](../../CONTEXT.md) and ADR-0041 both record as
  `resolution-walk` alone.
- **or the membership vector is wider than ADR-0041 records** — with the same consequence, and one
  more: `wildcard-discrimination`'s control-label count **already moved**, `5` → `9`
  ([#115](https://github.com/winniel123/verge-asm/issues/115)), which ADR-0021 records as bumping
  that leaf and `Break`ing `resolution`. If the leaf is in the vector, that move broke **every**
  `Name`'s membership timeline, and nobody priced it.

**It cannot be neither.** [#140](https://github.com/winniel123/verge-asm/issues/140) has raised the
vector question as a successor ticket, so it has a home, and it is not this one. This ADR therefore
discharges the obligation **for the four outcomes that are genuinely `resolution-walk`'s** and routes
the fifth, specified but explicitly undischarged. Pinning `Shadowed` rows to `resolution-walk`'s
corpus would be worse than leaving them unwritten: it would put a leaf's rows in another leaf's gate,
where a bump of the leaf that decides them moves nothing, and the obligation would read discharged
while protecting nothing.

> **ROUTED AND ANSWERED — [#146](https://github.com/winniel123/verge-asm/issues/146) ·
> [ADR-0086](./0086-membership-composes-every-leaf-that-decides-the-value-it-reads.md), 2026-08-15.**
> Recorded at this section per
> [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md): read alone
> and in the present tense, *"specified but explicitly undischarged"* sends a competent session to
> leave seventeen written cells unfiled. **The second horn is the true one — the membership vector is
> wider than ADR-0041 records.** `Shadowed` **cites no `Address`**, so the discrimination verdict
> decides whether an answer's address set enters the estate, which is a cited `Address`'s membership
> affirmatively rather than by suppression; and membership reads `resolution`, which two leaves decide
> jointly. The escrow is **adopted whole** into `wildcard-discrimination`'s corpus as
> [`golden-corpus.md`](../spec/golden-corpus.md) §8, with one boundary pair added for the citation
> ground. **The unpriced `5` → `9` consequence is priced at zero** — nothing has shipped and no
> `resolution` timeline exists — **and the reason expires at the first shipped release.** This ADR's
> own rule is what makes the adoption correct rather than a re-filing: a row protects the leaf whose
> gate runs it, so the two blocks are **counted together and run apart**.

## Decision

| Concern | Decision |
| --- | --- |
| Who owes the obligation | **The release**, per ADR-0041's own framing — unchanged |
| Who **owns** it | **A checked-in enumeration of cells**, in [`golden-corpus.md`](../spec/golden-corpus.md), that CI counts. An obligation with no failing test has no owner |
| Where a missing row is caught | **CI, on the coverage assertion** — a cell with no row fails the build, and the failure names the cell |
| Which leaf is discharged | **`resolution-walk`, and only it** *by this ADR*. Four outcomes: `Resolved`, `NoData`, `NameError`, `Lame`, plus the per-nameserver RRset the delegation walk decides. **The fifth leaf is discharged by [ADR-0086](./0086-membership-composes-every-leaf-that-decides-the-value-it-reads.md)**, so the file now discharges two |
| The fifth outcome | ~~**`Shadowed` is routed, not pinned.**~~ **ROUTED AND NOW PINNED** — [#146](https://github.com/winniel123/verge-asm/issues/146) · [ADR-0086](./0086-membership-composes-every-leaf-that-decides-the-value-it-reads.md) rules home **A**: `wildcard-discrimination` **is** in the membership vector, and the escrow is adopted whole as [`golden-corpus.md`](../spec/golden-corpus.md) **§8**. The two candidate homes stay recorded in §7 as history |
| The pin block | **27 cells** *for `resolution-walk`* — 24 outcome, boundary and provenance pins, plus 3 withdrawal→return rows. ~~**17 further cells are specified and held in escrow** against the `Shadowed` routing~~ — **the escrow is spent**: those 17 are now §8's, adopted whole, plus a citation pin ADR-0086 adds, for **19** against `wildcard-discrimination`. **File total 46, escrow 0** |
| How a boundary is pinned | **By a pair of rows, one on each side.** A single row cannot detect a collapse toward itself |
| The CI matrix | **Three legs**, all gating, all native, all on **one** expected-output artefact: `amd64`/`GOAMD64=v1` · `arm64`/`GOARM64=v8.0` · `amd64`/`GOAMD64=v3` asserting **identity with leg 1** |
| Every leg's build settings | **Stated explicitly, never defaulted** — `CGO_ENABLED=0` above all. The Go defaults are a function of the runner, and the native `arm64` leg is where they flip |
| The assertions | **Six**, and they fail in order — self-identity · expectation · cross-architecture identity · contraction differential · coverage · gate direction 2 |
| Self-identity first | Each leg runs the corpus **twice in one process** and requires byte-identical output. Go randomises map iteration **per iterator**, so an unstable corpus makes every downstream assertion uninterpretable |
| Per-architecture expected output | **Refused, permanently.** An architecture-specific golden file is the divergence written down and blessed |
| Emulated legs | **Refused**, and *not* on the FMA ground — on the scripted clock |
| When the matrix runs | **Every pull request and every release.** The corpus is hermetic, so there is no cheaper trigger to design around |
| What the pin block licenses | ADR-0021's gate, **second direction**: on `resolution-walk` **and, since [ADR-0086](./0086-membership-composes-every-leaf-that-decides-the-value-it-reads.md), on `wildcard-discrimination`** — a version bump with no moved pin row and no changed parameter **fails the build**. Evaluated **per leaf** against that leaf's own block |

## Rationale

### An unrouted obligation is discharged by routing it, never by guessing its home

The tempting resolution is to write the `Shadowed` rows into `resolution-walk`'s corpus anyway, on
the grounds that the rows are the same rows wherever they live and a later session can move them.

It is wrong, and the reason is the whole mechanism of ADR-0021's gate. **A corpus row protects the
leaf whose gate runs it, and nothing else.** A `Shadowed` row sitting in `resolution-walk`'s corpus
is checked when `resolution-walk`'s version is questioned and is silent when
`wildcard-discrimination` bumps — which is the leaf that actually decides the value. The row would
be exercised by a leaf that cannot move it and ignored by the leaf that can. That is not a
misfiled row. It is a row that **reads green while protecting nothing**, which is strictly worse
than an absent row, because an absent cell is countable and A5 fails on it.

So the disposal is:

> **Where an obligation names an outcome whose leaf is undecided, the obligation is discharged for
> the outcomes whose leaf is settled and *routed* for the rest. A row filed against the wrong gate
> is not a partial discharge — it is a false one.**

This is [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)'s
shape run forward rather than backward. ADR-0058 asks whether a sentence read alone would cause a
competent session to build the wrong thing. Here the question is whether a *row* read alone would
cause a competent session to believe the wrong thing is protected. It would.

The rows themselves are not wasted. §7 of [`golden-corpus.md`](../spec/golden-corpus.md) specifies
all seventeen in the same form as the discharged block, so the successor ticket inherits a written
enumeration and rules only the routing question. Escrow is cheaper than re-derivation and it is
visible, which an omission is not.

> **The bet is settled and it paid** — [#146](https://github.com/winniel123/verge-asm/issues/146) ·
> [ADR-0086](./0086-membership-composes-every-leaf-that-decides-the-value-it-reads.md) ruled the
> routing (**home A**), adopted all seventeen **whole and unamended**, and re-derived nothing. Noted
> here so the paragraph does not read, alone and in the present tense, as an open question. The one
> correction the successor made is recorded in the thin section below: the escrow had **no cell** for
> the ground the ruling turned on, and ADR-0086 added a boundary pair for it.

### Why a corpus row, and not the rule everybody reaches for first

The reflex repair is *do not bump a membership-composing leaf on a dependency upgrade*. It is wrong
twice.

It is wrong as a **policy**, because this project converts discipline into failing tests everywhere
in the comparison path and has refused the discipline form every time it has been offered. And it is
wrong as a **reading of the gate**, because ADR-0021 already forbids an unjustified bump — the
second direction of the bidirectional gate says *version moved and nothing justified it → fail*.

So what actually goes wrong on the upgrade? Not the gate. **The author.** ADR-0021's
*"the release where we cannot tell"* is explicit that where nobody can say whether output moved,
*"the honest default is to bump"* — and it prices that default at *"one facet family clamped for one
cadence"*, which was true when it was written and is **no longer true for `resolution-walk`**. For
that leaf the honest default costs estate-wide loss of `returned`, permanently, and the release
cannot know it happened.

The pin block is what stops *"we cannot tell"* from being the answer. With it, the upgrade runs the
corpus, the pin rows do not move, and the second direction of the gate stops being a formality and
becomes an **active refusal**: the bump is not merely unjustified, it is *demonstrably* unjustified,
because the block's coverage is itself a checked property.

> **The corpus does not stop the leaf from bumping. It removes the excuse for bumping — and the
> gate's second direction does the rest.**

### A boundary is pinned only by a pair of rows

This is the rule the block's size comes from, and it is the one a session writing rows will
otherwise get wrong.

A corpus row is a stimulus and an expected output. A row expecting `NameError` proves that *this*
stimulus yields `NameError`. It does **not** prove that a neighbouring stimulus still yields
`NoData` — and a library upgrade whose failure mode is *collapsing the neighbourhood onto one
answer* passes that row untouched. The row cannot see a collapse toward itself.

> **A boundary between two outcomes is pinned by two rows, one on each side, authored as a pair and
> failing as a pair.** A boundary with one row is not pinned; it is decorated.

That is why the block is enumerated by **boundaries** rather than by outcomes, and why the boundary
count doubles into the row count. It is also why the enumeration is checked-in data: a *missing*
pair is countable, and ADR-0021 already established that a corpus reads as a list of sentences so a
missing claim is visible.

The boundaries themselves are chosen on one criterion — **the wire is genuinely ambiguous there, and
two conforming implementations may read it differently.** NXDOMAIN carrying a CNAME in the answer
section, an empty non-terminal answering NOERROR with no records, a truncated RRset and its TCP
fallback, an authority that REFUSEs against one that is silent: these are the places DNS libraries
have historically differed, and each of them lands on the withdrawal edge. The enumeration is in
[`golden-corpus.md`](../spec/golden-corpus.md) §2 and it will be revised. The criterion is here and
will not.

### Three legs, one expected output — and the leg that is not an architecture

Running the corpus on two architectures proves each architecture matches its own expectation. It
proves the two architectures agree **only where the expectation is one artefact**, shared, byte-
exact, and never per-architecture. So:

> **There is no per-architecture expected output, ever.** An `arm64` golden file is the divergence
> #124 exists to prevent, written down and blessed — and it is the first repair a session will reach
> for when the `arm64` leg goes red.

The third leg is `amd64` at `GOAMD64=v3`, and it is **not an architecture in the matrix**. #124
pinned `GOAMD64` at `v1` because *"at `v3` the same binary gains FMA on the same architecture, so an
unpinned level makes the divergence a property of who ran the build."* That pin is a *precaution*.
Leg 3 turns it into a *measurement*: it builds at `v3`, runs the same corpus against the same
expected output, and fails where it differs from leg 1. A difference is proof that a fraction
escaped exact integer arithmetic, and the failure names it.

This is the enforcement `packaging-and-configuration.md` §1.3 rule 2 was written without. It reaches
past this ticket's leaf — it guards every corpus, because every declared parameter expressed as a
fraction is in its blast radius — and that reach is stated rather than hidden. See
[`project-authored-constants.md`](../research/project-authored-constants.md) §12.

Leg 3 ships nothing, so #124's membership rule is untouched: an architecture is still in the matrix
exactly where the corpus runs on it, and `GOAMD64=v3` is a microarchitecture level with no artefact.

Two things retrieved for this ticket sharpen leg 3 and are recorded because a later session will
otherwise re-derive them. **Go's `arm64` backend contracts unconditionally** — the rewrite rules
fuse `FADDD a (FMULD x y)` into `FMADDD` behind a `useFMA` predicate that carries no architecture or
CPU-feature gate at all, and whose only off switch is a bisection debug variable the Go team
documents with the words *"if you have an architecture-dependent FP glitch, this will help you find
it"*. **There is no `GOARM64` level that turns it off**, FMADD being ARMv8.0 baseline. So the
`amd64` `v1`/`v3` pair is the *only* lever this project has, and leg 3 uses it in the one direction
available: not to make the two architectures match, but to make a contraction-eligible expression
**visible**. And it is genuinely only a detector — `v3` narrows the gap and does not close it, the
two backends' rewrite-rule sets not being the same set. Leg 3 is not a claim that `v3` and `arm64`
agree. A green leg 3 says *no expression on this path was contraction-eligible*, which is the
property #124's rule 2 actually asserts.

### Every leg states its build settings, because the defaults are a property of the runner

#124 pinned `CGO_ENABLED=0` *"always, for every artefact"* and gave the strong reason: with cgo
available Go's `net` package may use the **system** resolver, which is a second answer path for a
question [ADR-0070](./0070-a-control-probe-is-asked-from-where-the-answer-it-discriminates-was-asked-from.md)
has already fixed to one. That is a rule about the **shipped artefact**. It does not reach the
**binary the corpus runs against**, and that is where it bites.

`CGO_ENABLED`'s default is not a constant: it is on for a native build where a C toolchain is
present and off when cross-compiling. Every leg of this matrix is **native** — that is the whole
point of legs 1 and 2 — so the legs are precisely the builds where the default is *on*. A leg that
leaves it unset therefore runs the corpus against a binary that may resolve through a path the
shipped binary cannot use, and the corpus would be verifying the wrong instrument while reading
green.

> **A corpus leg builds the binary under test with the shipped build settings, every one of them
> stated rather than defaulted.** A default is a function of the runner, and a runner is not part of
> the release.

This is [#69](https://github.com/winniel123/verge-asm/issues/69)'s *a shipped default is the
configuration that takes effect and that the project documents as its default* applied to our own
build, and ADR-0021's *a default is not a declaration* applied one layer out from the job spec.

### Native runners, and the reason that is *not* the reason

Emulated legs are refused, and the ground matters because the obvious ground is wrong.

The obvious ground is that emulation would not exercise the real instruction stream. That is false —
an arm64 binary under emulation executes arm64 instructions, FMA included, and would reproduce the
divergence faithfully.

The real ground is ADR-0021's own stated fear. **`resolution-walk` and `connect-outcome` are decided
against a scripted clock** — a timeout, a retry budget, silence for *n* seconds. Emulation's
slowdown makes a clock-sensitive corpus intermittently red for reasons that are not ours, which
ADR-0021 identifies as *"the fastest known way to train a team to ignore a gate"* and refuses in the
live-listener alternative. A gate the team has learned to re-run is not a gate.

So the legs are native, and #124's *"anything else is out until somebody pays for a runner"* is
priced correctly: the price is a native runner, not a build flag and not a QEMU line.

**And for these two architectures the price is zero, which is retrieved rather than assumed.**
GitHub's own changelog took `arm64` hosted runners to general availability for public repositories
on **2025-08-07** — `ubuntu-24.04-arm` and `ubuntu-22.04-arm`, at no cost, standard runners being
free and unlimited on public repositories — and extended them to private repositories on
**2026-01-29**. This project is public and AGPL-3.0, so both legs run on free standard runners and
#124's assumption that *"CI runs the corpus on both natively"* is now a checked fact rather than a
hope. One rider, because it will otherwise cost a session an afternoon: **there is no
`ubuntu-latest-arm`.** The `-latest` alias exists on `x64` and does not exist on `arm64`, so the
`arm64` leg pins a version label — which this project would want anyway, a floating runner image
being out-of-band reference data in the CI configuration.

### The corpus must reproduce against itself before it can be compared across anything

This is the assertion that has to fire first, and it is not about architecture at all.

Go's map iteration order is unspecified by the specification and **actively randomised by the
runtime per iterator** — not per process, per *iterator*, so two range loops over one map in one
process yield different orders. Go 1's release notes say why the randomisation exists, and the
sentence is uncomfortably close to this ticket: the old behaviour *"differed across hardware
platforms"* and made tests *"fragile and non-portable"*, so the language made the disorder explicit
rather than letting it be discovered.

`resolution-walk`'s highest-stakes output is `Resolved(`**`unordered`** ` address set)`. The model
says unordered. The NDJSON is bytes, and bytes have an order. If the leaf serialises that set from a
map range, the corpus's expected output is unstable **within one runner**, and every cross-leg
comparison downstream is comparing noise. A red `arm64` leg would then be indistinguishable from a
flake, which is the failure that trains a team to re-run a gate.

Two consequences, and the second is a corpus-row obligation rather than a CI one:

1. **Every leg runs the corpus twice in one process and requires byte-identical output**, and this
   assertion fails before any other. Twice *in one process* rather than twice in one job, because
   per-iterator randomisation is the stronger property and a two-job comparison would miss it.
2. **`Resolved`'s address set is serialised in a stated total order** — ascending by `Address` key,
   which is family then octets compared as octets
   ([ADR-0051](./0051-a-subject-key-is-the-thing-denoted-and-its-normalisation-may-never-move.md)) —
   and the ordering is a property of the **serialisation**, never of the value. The set stays
   unordered in the model. This is the one place the corpus imposes something the domain does not
   have, and it is imposed on the transport rather than on the fact.

### The trigger is every pull request, and the `go.mod` cadence is why

The named failure mode arrives on a **dependency** PR — the cadence
[#49](https://github.com/winniel123/verge-asm/issues/49) put `resolution-walk` on. The tempting
design is a `go.mod`-touched trigger.

It is refused for being clever. The corpus is hermetic by construction — no network, no containers,
no fixture images, which ADR-0021 made *"a hard requirement rather than a preference"* — so it is
cheap enough that there is nothing to save, and a path-filtered trigger is a second place the
obligation can be silently disabled. **Every pull request, every leg.** The `go.mod` case is then
not a special case at all, which is the only way to be sure it is covered.

## Consequences

- **[`golden-corpus.md`](../spec/golden-corpus.md) is new**, and holds the two things that will be
  revised: the 27-cell enumeration, ~~the 17-cell escrow~~ **§8's 19-cell block (the escrow adopted
  whole by [ADR-0086](./0086-membership-composes-every-leaf-that-decides-the-value-it-reads.md), plus
  a citation pin)**, and the three-leg matrix. It follows
  [`measurement-offers.md`](../spec/measurement-offers.md)'s precedent — *the tables will be revised
  and an ADR is a decision that will not.*
- **ADR-0041's obligation 2 is defective and is *routed*, not amended here.** Its five-outcome list
  spans two leaves while naming one. [#140](https://github.com/winniel123/verge-asm/issues/140) is
  amending that ADR on its own branch and has ticketed the vector question, so the correction is
  made once, at one site, by the ticket that owns it. This ADR states the defect, discharges what it
  can, and takes nothing that is #140's.
  **The correction has since been made**, at obligation 2's own site, by
  [#146](https://github.com/winniel123/verge-asm/issues/146) ·
  [ADR-0086](./0086-membership-composes-every-leaf-that-decides-the-value-it-reads.md): **the list of
  five is right and the scope was wrong** — the outcomes are pinned in two blocks against two gates.
- **[ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md)'s *"the release where we cannot
  tell"* gains a scope.** *"The honest default is to bump"* and its price — *"one facet family
  clamped for one cadence"* — are true for three leaves and false for `resolution-walk`. ~~Whether
  they are false for a fourth is the routing question.~~ **They are false for a fourth**
  ([ADR-0086](./0086-membership-composes-every-leaf-that-decides-the-value-it-reads.md)): the honest
  default is true for **three** — `connect-outcome`, `tls-handshake`, `http-exchange` — and **false
  for two**, `resolution-walk` and `wildcard-discrimination`. Superseded here, at the site that
  states it.
- **ADR-0021's gate gains no third direction.** The second direction already refuses an unjustified
  bump. What was missing was the evidence that would let it fire, and that is the pin block. This is
  deliberate: a third direction would be a new mechanism where an old one was merely unfed.
- **`packaging-and-configuration.md` §1.2's membership rule is unchanged and now has a referent.**
  *"An architecture is in the matrix exactly where the corpus is run on it"* previously pointed at a
  corpus whose contents and CI shape were unspecified. Legs 1 and 2 are that corpus. Leg 3 is not an
  architecture.
- **`packaging-and-configuration.md` §1.4's `CGO_ENABLED=0` reasoning gains a second reader.** It is
  stated there as a property of the shipped artefact. The corpus legs need it as a property of the
  **build under test**, and native builds are where the toolchain default disagrees. The rule is
  #124's and its reach is extended here rather than restated there — a pointer is cheaper than a
  duplicate, and that file is #124's.
- **`project-authored-constants.md` gains §12** — the differential leg as the enforcement §8.1's
  *ship the rule* cure never had, and the note's population as its blast radius.
- **[`CONTEXT.md`](../../CONTEXT.md) is amended in one entry, by one clause.** `Derivation`'s
  golden-corpus sentence says the gate runs *in CI* and stops there. It now says the corpus runs on
  every shipped architecture against **one** expected output. No term is added, and the
  membership-vector sentence in `Transition` is **deliberately untouched** — that sentence is what
  the routing question is about. **It has since been amended** by
  [#146](https://github.com/winniel123/verge-asm/issues/146) ·
  [ADR-0086](./0086-membership-composes-every-leaf-that-decides-the-value-it-reads.md), which is the
  ticket that owns it: the vector is `resolution-walk` **and** `wildcard-discrimination`.
- **`Resolved`'s address set acquires a serialisation order and no ordering.** Ascending by
  `Address` key on the NDJSON, and unordered in the model, which is where ADR-0021's three-corpora
  chain already puts the boundary: the wire stops at the binary's edge, and a total order on bytes
  is not a total order on a set.
- **The block's completeness is a claim about the boundaries, not about DNS.** CI can count cells
  against the enumeration. It cannot tell anyone the enumeration named every ambiguous place on the
  wire. That residue is ADR-0021's **uncovered move**, unchanged and now with a population it
  applies to.
- **Nothing here is a workflow file.** The map's *plan, don't do* rule holds: this specifies legs,
  assertions, triggers and their order, and an implementation session writes the YAML.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Leave the obligation as ADR-0041's sentence and let the corpus author read it** — the smallest change, and what the ticket would have been closed as | It is the state the ticket names: a failure mode with no owner. Nothing counts the rows, nothing fails when they are absent, and the first reader to discover the gap is the operator whose `returned` became `appeared` |
| **Pin all five outcomes to `resolution-walk`'s corpus, as ADR-0041's sentence literally instructs** | **The losing option, and it loses on the gate rather than on tidiness.** A row protects the leaf whose gate runs it. A `Shadowed` row in `resolution-walk`'s corpus is silent when `wildcard-discrimination` bumps — the leaf that actually decides the value — so the obligation would read discharged while protecting nothing. A false discharge is worse than an absent cell, which A5 at least counts |
| **Rule the vector question here — `wildcard-discrimination` is in the membership vector, on the ground that a bump `Break`s `resolution`** | It is a defensible reading and it is not this ticket's to make. #140 has ticketed it, a decision lives in exactly one place, and the ruling carries an unpriced consequence — #115's `5` → `9` move would retroactively have broken every `Name`'s membership timeline. That deserves its own ticket, not a paragraph in this one. **[#146](https://github.com/winniel123/verge-asm/issues/146) · [ADR-0086](./0086-membership-composes-every-leaf-that-decides-the-value-it-reads.md) is that ticket and it ruled the same way** — on a stronger ground than *a bump `Break`s `resolution`*, namely that `Shadowed` **cites no `Address`** — and it paid the price: **zero**, because nothing has shipped, with the expiry named. Deferring was right; the successor had the room to price it and this one did not |
| **Say nothing about the fifth outcome and discharge four quietly** | The defect would survive undetected in ADR-0041's text, and the next session to read obligation 2 would write the rows against the wrong gate. Routing costs a section and buys the successor a written enumeration |
| **A policy: never bump a membership-composing leaf on a dependency upgrade** | Discipline where this project ships failing tests, and it is also already covered — ADR-0021's second gate direction forbids an unjustified bump. The gap was never the rule, it was the evidence |
| **A third gate direction, specific to membership leaves** | A new mechanism where an old one was unfed. The second direction fires correctly once the pin block exists; adding a third would leave two rules for one fact |
| **Pin outcomes only — one row per outcome, as ADR-0041's sentence reads literally** | A single row cannot detect a collapse toward itself, so those rows survive exactly the upgrade shape they exist to catch. This is the rule the block's size comes from |
| **A per-architecture expected-output file, so the `arm64` leg can go green** | It is the divergence #124 exists to prevent, recorded as the fix. Two installs would then hold equal derivation vectors over values produced by two implementations, which is exactly the invisible failure §1.2 describes |
| **Emulated `arm64` under QEMU instead of a native runner** | Rejected, but not on fidelity — emulation reproduces FMA faithfully. It is the scripted clock: an intermittently red gate is a gate the team learns to re-run, ADR-0021's own stated reason for refusing live-listener fixtures |
| **Cure contraction with the specification's explicit-conversion rule instead** — `r = float64(x*y) + z`, which the Go spec documents as preventing fusion | Per-expression discipline in the one place this project has most consistently refused it: it must be re-applied correctly by every future author of every fraction, and nothing fails when it is forgotten. #124's integer-arithmetic cure is checkable by leg 3; a conversion rule is checkable by review only. Recorded because the spec offers it and the next session will find it |
| **Leave `CGO_ENABLED` to the toolchain, since #124 already pins it for the artefact** | #124's pin reaches the shipped artefact and not the binary under test, and the corpus legs are native builds — exactly the case where the default is *on*. The corpus would verify a binary with a resolver path the release cannot use, and read green doing it |
| **Assert cross-architecture identity without first asserting self-identity** | Go randomises map iteration per iterator, so an unstable corpus makes a red `arm64` leg indistinguishable from a flake. Ordering the assertions is what keeps the cross-architecture failure interpretable |
| **Run the matrix only on PRs touching `go.mod` / `go.sum`** | The corpus is hermetic and cheap; there is nothing to save, and a path filter is a second place the obligation can be disabled without anyone noticing. Every PR makes the dependency case unremarkable |
| **Drop leg 3 — `GOAMD64` is pinned, so `v3` never ships** | The pin is a precaution nothing measures. Leg 3 is the only assertion in the repository that a declared parameter's fraction was actually evaluated in integer arithmetic, and #124 wrote that rule with no enforcement at all |
| **Put the enumeration in this ADR** | It is a list and its membership moves whenever a leaf, a parameter or a boundary moves. `measurement-offers.md` and `packaging-and-configuration.md` both split for this reason |

## Where this is thin, stated rather than smoothed

- **The nine boundaries are reasoned from the DNS specifications and from ADR-0021's and ADR-0070's
  measured findings — they are not measured against two DNS libraries.** Nobody has taken two
  implementations and diffed them across these boundaries, so the claim *these are the places
  conforming implementations differ* is a specification-derived expectation. It inherits ADR-0021's
  own honesty rider: a row encoding a spec-verified claim **says so on the row**.
- **27 is the length of a list, and the list is a judgement.** The count is exact and derived
  ([#74](https://github.com/winniel123/verge-asm/issues/74) — *each member's count is the length of
  its own list*), but completeness of the enumeration is not checkable by anything. An
  under-enumerated block passes its own coverage assertion silently, which is the same shape as
  ADR-0021's uncovered move and is disposed of the same way.
- ~~**The escrow's size is a guess about a question that is not answered.**~~ **NO LONGER THIN —
  the guess was taken and it paid.** [#146](https://github.com/winniel123/verge-asm/issues/146) ·
  [ADR-0086](./0086-membership-composes-every-leaf-that-decides-the-value-it-reads.md) answers the
  routing question **A** and adopts all seventeen **whole and unamended**, so the cheaper error was
  not paid and the successor re-derived no rows. Struck at the site that states the thinness, per
  [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md). One thing
  the escrow got **wrong** and ADR-0086 repairs: the seventeen pin the *suppression* ground and have
  **no cell** for the ground the ruling actually turns on — that `Shadowed` carries no address set and
  therefore cites no `Address`. ADR-0086 adds that boundary as a pair, taking the block to **19**.
  The lesson is not that escrow is unsafe. It is that **an escrow written against a question's
  framing may miss the ground its answer lands on**, and the adopting ticket has to check.
- **Leg 3's yield is unmeasured.** No fraction in the declared-parameter set has been shown to
  diverge at `v3`. The argument is that the check is cheap and the failure it catches is silent.
  If it never fires, that is the pin working, and there is no way to distinguish that from the leg
  being unnecessary.
- **The runner availability is retrieved and the project's CI is not.** `ubuntu-24.04-arm` is GA and
  free on public repositories, which is checked. That this project's CI will be GitHub Actions is
  assumed, because no ticket has decided it. If it is not, the legs and the six assertions transfer
  unchanged and only the labels move.
- **`arm64`'s contraction being ungated is read off the compiler's rewrite rules, not off a document
  the Go team wrote for readers.** No release note or prose page states it. `_gen/ARM64.rules` and
  `_gen/AMD64.rules` do. It is the owner's own bytes and it is source rather than documentation,
  which is a weaker footing than
  [ADR-0059](./0059-a-footing-tier-grades-evidential-distance-never-the-owners-conviction.md)'s top
  tier and is labelled here rather than smoothed. The **specification** sentence permitting fusion is
  top-tier and is what the rule actually rests on. The backend detail only says the hazard is live
  today.
