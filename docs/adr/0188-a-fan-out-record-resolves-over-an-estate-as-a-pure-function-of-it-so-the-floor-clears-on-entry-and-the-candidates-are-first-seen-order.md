# ADR-0188: A fan-out record resolves over an `Estate` as a pure function of it, so the floor clears on entry and the candidates are first-seen order

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1305 ADR gaps: internal/custody 1/3](https://github.com/winniel123/verge-asm/issues/1305), gaps 2 and 4
- **PR that deleted the comment:** [#1306](https://github.com/winniel123/verge-asm/pull/1306)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Rests on:** [ADR-0129](./0129-a-shared-foreign-edge-is-measured-by-fan-out-not-read-from-a-list.md). It rules that a fan-out **measurement** decides a shared foreign proxy edge, and that no provider list decides it. It also fixes that measurement's population at the custody-extension candidates plus the declared address scopes. ADR-0129 rules the derivation. This ADR rules the execution of that derivation, and neither contains the other
- **Sibling of, and not ruled by:** [ADR-0195](./0195-the-address-scope-census-renders-in-declaration-order-and-a-scope-declared-twice-renders-once.md), from [#1310](https://github.com/winniel123/verge-asm/issues/1310), which rules that the address-scope census renders in **declaration** order. That is a render sequence, so an operator reads back the order of their own declaration. This ADR rules a **probe population** in first-seen order, so one tick matches the next. Two rules over two populations, and neither contains the other

## Context

Two comments in `internal/custody` stated execution properties of the ADR-0129 derivation.
[#1306](https://github.com/winniel123/verge-asm/pull/1306) deleted both. Nothing on disk states
either one. Those are [#1305](https://github.com/winniel123/verge-asm/issues/1305)'s gaps 2 and 4.

`internal/custody/veto.go:197`, pre-sweep, on `EdgeFanout.overExtension`:

```go
// It is TOTAL and IDEMPOTENT: it resolves the verdict from the inputs alone and clears an
// inbound one, so re-resolving a record over a second estate cannot carry a stale errored
// reading into it.
```

`internal/custody/candidates.go:36`, pre-sweep, on `Estate.ExtensionCandidates`:

```go
// Addresses are distinct and in first-seen order. The dedup is what makes the Scan one
// handshake per address: two in-zone names flattening to the same edge is the modal case,
// and a second handshake against that edge would report what the first already did. The
// order is deterministic so one tick's fan-out matches the next.
```

The two properties meet at one statement, `internal/custody/custody.go:54`:

```go
e.edgeFanout = f.overExtension(e.ExtensionCandidates())
```

One expression takes a measurement record and an `Estate`, and returns the record resolved
against that `Estate`. Limb 3 rules the input to that expression. Limbs 1 and 2 rule the
expression. That is why one ADR carries both.

### The floor is a property of the pair, never of the record

`EdgeFanout.ExtensionErrored` is the per-limb floor that
[#1018](https://github.com/winniel123/verge-asm/issues/1018) added. It is **false** in every
record the store reader returns. `internal/queue/edgefanout.go:296` builds the record and sets
three fields. The floor is not among them:

```go
return custody.EdgeFanout{Enabled: true, BatchCompleted: completed, Shared: shared}
```

`internal/queue/edgefanoutread_test.go:288` pins that:

> `ExtensionErrored = true` — this package holds no candidate set and may not resolve the floor

So the repo writes `ExtensionErrored` at exactly **two** statements, `veto.go:46` and
`veto.go:58`. Both sit inside `overExtension`. The first is the clear. The second is the verdict.

The record cannot carry the floor, because the floor is not a property of the record. Four sites
assemble an `Estate`, and the same store rows resolve to two different floors across them.

| Site | Candidate set | Record read | Floor `overExtension` resolves |
| --- | --- | --- | --- |
| `internal/queue/hot.go:117` | The in-zone direct-A targets | `EdgeFanoutUnbounded()` | May be **true**: an in-force `Scan`, a completed `Batch`, and no measured candidate |
| `cmd/web/custodycensus.go:99` | The same in-zone targets | `EdgeFanoutOver(estate.ExtensionCandidates())`, so `Partial` is true | The same verdict on the same rows |
| `cmd/web/addressscopecensus.go:43` | **Empty.** The `Estate` carries `AddressScopes` alone, with no `Resolutions` and no `ExtendedZones` | `EdgeFanoutUnbounded()` | Always **false**, on `overExtension`'s `len(candidates) == 0` arm |
| `internal/custody/corpus/harness.go:44` | The corpus row's own resolutions | A record built from the row | Whatever that row's candidates resolve |

Row 3 is the measurement that makes the rule non-obvious. The address-scope census reads the same
`edge_fanout` rows as the hot dispatcher, over an `Estate` that declares no extension. The
extension floor over that `Estate` is false, and it must be, because that census reads
`e.edgeFanout.Enabled` alone at `internal/custody/scopecensus.go:14`. A floor carried in from a
prior resolution is a value the second `Estate` never earned.

### The clear reads as dead code, and that is the hazard

`Estate.edgeFanout` is unexported. No production caller reads a resolved record back out of one
`Estate` and hands it to a second. `EdgeFanout.ExtensionErrored` is **exported**, so any caller
may set it. A reader who checks only the live paths finds that nothing hands in a true floor
today. That reader then finds the first statement of the function body:

```go
func (f EdgeFanout) overExtension(candidates []netip.Addr) EdgeFanout {
	f.ExtensionErrored = false
```

That assignment is the whole idempotence guarantee, and it reads as redundant. The deleted comment
was the only text on disk that said why it is there.

### The candidate order is load-bearing three steps downstream

`Estate.ExtensionCandidates` walks `e.Resolutions` once, holds a `map` of admitted addresses, and
appends to a slice. The order is the order of the first `Resolution` that carries each address.
The dedup drops the rest.

`Estate.EdgeFanoutPopulation` yields those candidates first, in that order, and then the declared
address scopes. `internal/queue/edgefanout.go:30` hands that sequence to
`scan.BuildEdgeFanoutJobs`, which cuts it into chunks of `EdgeFanoutAddressesPerJob = 50`. The
chunk index then names the `Batch`:

```go
spec, err := j.JobSpec(fmt.Sprintf("scan:%d:edges:%d", scanID, j.Chunk))
```

So the order decides which addresses share a job, which addresses share a `Batch`, and what each
job's `AttemptedScope` states. An unordered population re-cuts every chunk on every tick, and no
two ticks compare.

The dedup carries a second cost. `internal/queue/edgefanout.go:108` drops a repeated address
inside one batch and records it as an error:

> `address %s measured twice in one batch`

A duplicate that lands inside **one** chunk becomes a dropped row. A duplicate that straddles two
chunks becomes a **second handshake** against an edge the first handshake already measured. The
deleted comment names the case that produces it: two in-zone names flattening to one CDN edge.

The determinism reaches the store on both limbs, and no test in `internal/custody` sees that.
`NameCitedAddresses` (`db/queries/measurement.sql:135`) ends `ORDER BY subject_key, address`, and
`ListAddressScopeCidrs` (`db/queries/measurement.sql:28`) ends `ORDER BY id`. `hotEstate` is
called from **nine** sites, and each one re-reads and re-builds. The tick-to-tick claim holds only
because both queries are ordered.

## Decision

> **Resolving a fan-out record over an `Estate` is a pure function of that pair. `overExtension`
> is total: it derives the errored floor from the record and the candidate set alone, and it has
> no failure arm. `overExtension` is idempotent: it clears an inbound floor before it derives one,
> so a record resolved over one `Estate` carries nothing into a second. `ExtensionCandidates`
> supplies the candidate set, distinct and in first-seen order, so one tick's `edge-fanout`
> population matches the next, and one edge that many in-zone names cite is handshaked once.**

### 1. `overExtension` is total

Every input maps to a result. It returns an `EdgeFanout` for a disabled `Scan` and for an
incomplete `Batch`. It returns one for an empty candidate set and for a nil `Shared` map. It
returns one for a candidate set that no row measures. It returns no error and it panics on
nothing.

The reason is the direction ADR-0129 §2 fixed. The floor exists to make a measurement failure
**loud**. A resolution that itself failed would add a second, silent failure mode under the first
one. Every caller would then choose a reach for that second failure.

### 2. `overExtension` clears the inbound floor before it derives one

The first statement of the body assigns `false`. **That statement is not dead code. A session that
finds no caller passing `true` may not delete it.**

The reason is that the floor is a property of the `(record, Estate)` pair and never of the record.
One record read from the store may apply to two estates. §Context's table holds the live case. The
hot dispatcher and the address-scope census read the same `edge_fanout` rows. Their two estates
hold different candidate sets: the in-zone targets, and the empty set. Those two estates resolve
to two floors, and each floor comes from its own candidate set.

`ExtensionErrored` is exported, so the type enforces nothing at the field. The clear is where the
rule is enforced, and it costs one assignment.

`TestWithEdgeFanoutClearsAnInboundErroredReading` (`internal/custody/veto_test.go:314`) pins it. It
hands `WithEdgeFanout` a record carrying `ExtensionErrored: true` beside a **measured** candidate,
and it asserts both halves. The resolved floor is false, and `Derive` returns `third-party` for
that measured shared edge. The second assertion is the one that matters. A carried floor lifts
`inForce`, and a shared edge the `Scan` measured then reads as `operator`.

### 3. `ExtensionCandidates` returns addresses distinct and in first-seen order

First-seen is the order of the first `Resolution` in `e.Resolutions` that carries each address. A
repeat is skipped and never re-ordered. A rejected address records nothing, so a later resolution
on an in-zone owner still admits it.

**The order is deterministic because the fan-out population must match from one tick to the next.**
§Context traces the three steps. The population feeds `BuildEdgeFanoutJobs`, the jobs cut at 50
addresses, and the chunk index names the `Batch`. Nothing downstream re-sorts.

**The list is distinct because one edge is handshaked once.** Many in-zone names citing one CDN
edge is the modal shape this derivation exists for. A second handshake against that edge reports
what the first reported, and it lands either as a dropped row or as a wasted connect.

`TestExtensionCandidatesAreDistinctInFirstSeenOrder` (`internal/custody/candidates_test.go:48`)
pins it. Three in-zone names resolve to two addresses, the first and the third to the same one,
and the assertion is the exact two-element sequence `104.16.132.229`, `104.16.132.230`. A sorted
order fails it, and a map-iteration order fails it.

### 4. What this rule does not reach

- **The declared-address-scope half of `EdgeFanoutPopulation`.** That half enumerates each declared
  prefix in turn ([ADR-0127](./0127-the-address-scope-range-cap-has-no-ceiling-a-large-scope-is-priced-not-gated.md)),
  and its order is the order of the prefixes. Limb 3 rules the extension-candidate list, which is
  the half the veto reads.
- **Which addresses are candidates.** ADR-0129 and its
  [#956](https://github.com/winniel123/verge-asm/issues/956) amendment rule the population. Limb 3
  rules its order and its distinctness alone.
- **The verdict the floor stands for.** ADR-0129 §2 rules that an errored limb reaches rather than
  withholds. Limbs 1 and 2 rule how that verdict is computed, never what it means.
- **Any render order.** The census surfaces are ruled separately.
  [ADR-0195](./0195-the-address-scope-census-renders-in-declaration-order-and-a-scope-declared-twice-renders-once.md)
  rules that the address-scope census renders in declaration order.
- **The store-side `ORDER BY`.** Limb 3 binds `internal/custody`. The two queries it depends on sit
  in `db/queries/measurement.sql`, and §Consequences names that gap.

## Consequences

- **This ADR changes no Go code.** `overExtension` and `ExtensionCandidates` already behave as
  ruled, and two tests already pin them.
- **`internal/custody/veto.go:46` is protected from a dead-code deletion.** Before this, the only
  defence was a comment, and [#1306](https://github.com/winniel123/verge-asm/pull/1306) removed it.
- **A defect this ruling exposes.** The tick-to-tick guarantee rests on two `ORDER BY` clauses that
  nothing pins. `NameCitedAddresses` ends `ORDER BY subject_key, address`.
  `ListAddressScopeCidrs` ends `ORDER BY id`. Limb 3's determinism is a property of
  `ExtensionCandidates` given an `Estate`. The tick-to-tick claim also needs the `Estate` to stay
  stable. `TestExtensionCandidatesAreDistinctInFirstSeenOrder` builds its `Resolutions` from a
  literal, so it passes with either clause removed. **It ships as its own ticket.** Pin both
  clauses in a store-level test, and cite this ADR at each one. A comment inside a query body moves
  `sqlc`'s generated string, so that ticket must regenerate `internal/db`.
- **A second defect, smaller.** `EdgeFanout.ExtensionErrored` is exported, and no caller outside
  `internal/custody` may set it. `internal/queue` builds the record and leaves the field false on
  purpose. A test states the reason. The field is exported only because `EdgeFanout` crosses a
  package boundary as a value. Unexporting it and adding a setter moves the guarantee from a
  convention to the type. **It ships as its own ticket.**
  [ADR-0133](./0133-an-address-exclusion-is-a-limb-of-the-custody-derivation.md) §2 is the shape to
  weigh it against.
- **Two `.go` comments return, each citing a limb.** `veto.go` gains one beside the clear, and
  `candidates.go` gains one beside the append. Both facts are unrecoverable from the code, and both
  name a constraint outside their own file.
- **`CONTEXT.md` gains nothing.** Idempotence and iteration order are execution properties, not
  domain terms. `CONTEXT.md`'s `edge-fanout` paragraph already carries the population.
- **[ADR-0129](./0129-a-shared-foreign-edge-is-measured-by-fan-out-not-read-from-a-list.md) gains
  one cross-reference line.** A reader who lands on the derivation now learns that its execution is
  ruled separately.
- **Nothing enforces limbs 1 and 2 beyond the two tests.** No check fires on a resolver that reads
  a field it did not clear. Review carries the rule.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **A fifth amendment to [ADR-0129](./0129-a-shared-foreign-edge-is-measured-by-fan-out-not-read-from-a-list.md)** | ADR-0129 rules the custody-extension derivation itself, and it already carries four amendments, for [#944](https://github.com/winniel123/verge-asm/issues/944), [#954](https://github.com/winniel123/verge-asm/issues/954), [#955](https://github.com/winniel123/verge-asm/issues/955) and [#956](https://github.com/winniel123/verge-asm/issues/956). Neither rule here changes that derivation. Both are execution properties of it. A fifth box buries an execution rule inside a derivation ruling, and a reader looking for either rule has no reason to open ADR-0129 |
| **Two ADRs, one per rule** | The two rules meet at one statement, `custody.go:54`. The candidate list is the input the resolution is total over, and the resolution is the reason that input must be a set. A split puts the argument for `WithEdgeFanout`'s purity in one file and the argument for its argument in another |
| **Merge limb 3 with [#1310](https://github.com/winniel123/verge-asm/issues/1310)'s declaration-order rule, both being order rules in one package** | Two populations and two reasons. Limb 3 governs a **probe** population, and its reason is that one tick's chunking matches the next. #1310 governs a **render** sequence, and its reason is that an operator reads back the order of their own declaration. A merged rule states one reason, and the other population then holds a rule that nothing argued for |
| **Rule a sorted order for the candidates, matching [ADR-0136](./0136-topology-is-a-reading-not-a-census-so-the-graph-caps-rather-than-folds.md) §4** | ADR-0136 sorts a **graph** that an operator reads, where a sort is what the reader wants. Here the population is enumerated into jobs and never displayed. A sort costs a comparison per candidate and buys the same determinism the first-seen walk gives free. It also drops the link between the population's order and the resolution order the two queries already fix |
| **Delete `f.ExtensionErrored = false` and require every caller to pass a cleared record** | Puts the guarantee on four assembly sites and on every site added later. The field is exported, so a site that forgets it produces the silent failure ADR-0129 §2 refuses: every extension candidate held, for as long as the condition lasts, with no error anywhere |
| **Return an `error` from `overExtension` on an unresolvable record** | There is no unresolvable record. Every input maps to a floor, so the error arm is unreachable. It also forces each of the four assembly sites to choose a reach for a failure that cannot happen, and a wrong choice there is silent |
| **Drop the dedup and let `toEdgeFanoutRows` reject the repeat** | That rejection is per **batch**. Chunks cut at 50, so two names citing one edge either lose a row inside one chunk or buy a second handshake across two. The dedup is also what makes the `seen` map in `EdgeFanoutPopulation` hold the real population size |
| **State both rules in [`golden-corpus.md`](../spec/golden-corpus.md)** | That spec rules what the corpus pins. The corpus fixes one `Estate` per row and renders it once, so it observes neither idempotence across two estates nor order across two ticks. Both rules would sit in a document whose own mechanism cannot test them |
