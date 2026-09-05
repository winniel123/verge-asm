# ADR-0195: The address-scope census renders in the operator's declaration order, and a scope declared twice renders once

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1310 ADR gaps: internal/custody (2/3)](https://github.com/winniel123/verge-asm/issues/1310), gap 4
- **PR that deleted the comment:** [#1309](https://github.com/winniel123/verge-asm/pull/1309)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Rests on:** [ADR-0129](./0129-a-shared-foreign-edge-is-measured-by-fan-out-not-read-from-a-list.md). Its [#956](https://github.com/winniel123/verge-asm/issues/956) amendment created this census and fixed the row's own sentence. It ruled what a row says and it ruled nothing about the sequence the rows arrive in
- **Bounded by:** [ADR-0047](./0047-an-address-scope-is-its-own-enumeration.md), which makes a declared address a subject from the declaration. The declaration is therefore a durable operator act with a recorded position, and this ADR reads that position
- **Sibling of, and not ruled by:** [ADR-0136](./0136-topology-is-a-reading-not-a-census-so-the-graph-caps-rather-than-folds.md). Its §4 rules a deterministic **sorted** order for the topology graph's per-column cap. A graph node population carries no operator-authored sequence, so a sort is the only determinism available there. This census does carry one, so §4 does not reach it and is not withdrawn
- **Sibling of, and not ruled by:** **ADR-0188**, filed from [#1305](https://github.com/winniel123/verge-asm/issues/1305) in the same batch. It rules that the extension-candidate **probe** population is distinct and in first-seen order, so one tick's measurement matches the next. This ADR rules a **render** sequence, so an operator reads their own declaration back. Neither population contains the other

## Context

`internal/custody/scopecensus.go:96` carried this, until [#1309](https://github.com/winniel123/verge-asm/pull/1309) deleted it:

```go
// Entries are in declaration order, so one render matches the next.
```

Four lines above it, at `scopecensus.go:93`, the same comment block carried the duplicate rule:

```go
// The same scope declared twice is one entry — two identical
// rows would render one fact twice and read as two scopes.
```

Both sentences state a rule that binds a consumer outside the file. Nothing on disk states either
one. That is #1310's gap 4. Records 1, 2 and 3 of the same issue are **suppressed**, and no ADR is
written for them: ADR-0129's #944 amendment states the declined row's remedy and the citing name,
ADR-0047 and ADR-0030 remove the reverse-name alternative, and ADR-0129's #956 amendment states the
denominator. `CONTEXT.md` repeats all three.

**The code does what the comment said.** `Estate.AddressScopeCensus` walks `e.AddressScopes` once,
in slice order, and builds a parallel `scopes` slice plus a `counts` map. A prefix already in
`counts` is skipped, so a repeat never joins `scopes` a second time. The output loop then walks
`scopes`, not the map, and appends one entry per surviving scope. Map iteration touches the
measurement side alone, where it decides no order. So the emitted sequence is the caller's slice
order, and a repeat collapses onto its **first** occurrence.

**The production feed supplies declaration order already.** `ListAddressScopeCidrs`
(`db/queries/measurement.sql:28`) reads `FROM seed WHERE kind = 'address'` and closes
`ORDER BY id`. `seed.id` ascends with the declaration, so the slice reaching the census is the
operator's list in the order the operator wrote it. `cmd/web/addressscopecensus.go:41` and
`internal/queue/hot.go:114` both build the `Estate` from that read.

**A duplicate cannot arrive from the database today.** `db/migrations/00003_seeds.sql:26` creates
`UNIQUE INDEX seed_address_cidr_key ON seed (address_cidr)`, and no later migration drops it. The
Postgres `cidr` type also refuses a value with bits set to the right of the mask, so two rows cannot
mask onto one prefix either. `AddressScopes` is nonetheless a plain exported slice field that any
caller fills, and six call sites fill it. The collapse is a rule at that boundary, not a repair of a
database state.

**A renderer exists, and it does not read this census.** The triage record for #1310 states that
*"no renderer consumes the census yet"*. That is wrong. The Coverage screen renders the shared-edge
count today:

| Surface | What it does | Order it renders in |
| --- | --- | --- |
| `design-system/templates/coverage.tmpl:80` | Renders *this scope covers N addresses; M of them present a fan-out above the threshold* inside an aperture meter | Its enclosing `{{range .Meters}}` |
| `cmd/web/cold.go:193` `apertureMeters` | Builds one meter per `db.ListSeedsRow`, address scopes and name scopes interleaved | `ListSeeds` order |
| `db/queries/seeds.sql:16` `ListSeeds` | `ORDER BY s.created_at DESC, s.id DESC` | **Newest declaration first** |
| `cmd/web/addressscopecensus.go:17` `addressScopeSharedEdges` | Reduces the census to a `map[netip.Prefix]int` and returns the map | No order survives |

So the shipped console reaches the count through a **map lookup per seed**, and it renders the
scopes in the reverse of the operator's declaration order. The census's own sequence is discarded at
`cmd/web/addressscopecensus.go` before the template ever sees it.

**No test discriminates the two candidate orders.**
`internal/custody/scopecensus_test.go:19` declares `93.184.216.0/24` then `93.184.217.0/24` and
asserts row 0 and row 1 in that sequence. That sequence is declaration order **and** sorted order at
once, so the assertion passes under either rule.
`scopecensus_test.go:110` declares one prefix twice and asserts `len(got) != 1`. It asserts the
count and never the surviving entry's position. Twelve tests cover this function and none of them
separates this ADR's rule from its rejected alternative.

**Why this needed a ruling rather than a comment.** ADR-0136 §4 gives its sorted order this reason:
*"Deterministic sorted order, because it is stable across reloads and makes no product judgment
about which subject matters more."* The deleted comment gave the same reason in eleven words — *"so
one render matches the next"* — and reached the opposite rule. A reader who arrives at this census
holding ADR-0136 reads a matching motive, finds no order stated for the census, and sorts it. That
reader would be following the only written rule in reach. The two rules are not in conflict, but
nothing said so.

## Decision

> **The address-scope census returns its entries in the order the operator declared the scopes, and
> a scope declared twice returns once, at its first occurrence. ADR-0136 §4's sorted order rules a
> different population and stays in force, unwithdrawn.**

### 1. The order is the operator's declaration order

`AddressScopeCensus` emits one entry per surviving scope in the sequence of `Estate.AddressScopes`.
It applies no sort, and a caller may not sort the result.

The reason is not determinism on its own. Declaration order is already deterministic, so
determinism cannot choose between the two candidate orders. The reason is **that the operator wrote
the sequence**. An operator declares address scopes one at a time and their list has a shape they
remember: the two production ranges first, the office range last, the range a colleague added
yesterday at the bottom. The census exists so the operator can find a scope in their own list and
act on it — the row's remedy is a `Seed` exclusion (ADR-0129's #956 amendment). Rendering the census
in declaration order lets the operator find a scope where they put it. A sort moves it, and buys
nothing that declaration order does not already give.

### 2. A scope declared twice renders once, at its first occurrence

A prefix that appears more than once in `AddressScopes` produces exactly one entry. That entry sits
at the position of the **first** occurrence, and the later occurrences move no other entry.

Two identical rows would render one fact twice and read as two scopes. A census whose row count
exceeds the number of distinct things it counts is not a census.

The collapse is a rule at the `Estate` boundary and not a repair of a database state.
`seed_address_cidr_key` makes the duplicate unreachable from the seed table today, and this rule
binds every caller that fills the slice from anywhere else.

### 3. A withdrawal removes one entry and moves no other

An operator who withdraws an address scope removes that `seed` row. The scope leaves
`ListAddressScopeCidrs`, its entry leaves the census, and every surviving entry keeps its position
relative to every other. Ascending `seed.id` is stable under a deletion, so no reordering follows a
withdrawal.

A re-declaration of the same CIDR after a withdrawal is a **new** act with a new `seed.id`. Its
entry appears at the end of the census, not at the position the withdrawn declaration held. This is
correct and it is stated here because it surprises: the position tracks the live declaration, never
the CIDR. ADR-0134 already rules that the withdrawal is recorded by a tombstone and that the mover
does not survive the act.

### 4. ADR-0136 §4 is not narrowed, and it needs no ADR-0058 withdrawal

ADR-0136 §4 rules a deterministic sorted order for the topology graph's per-column cap. That
population is a set of graph nodes assembled from measured resolution and service edges. **Nothing
in that domain says which node comes first.** No operator authored the sequence, so a sort is the
only determinism available, and any total order will do. §4 also carries a second job this census
does not have: it selects the first N under a cap, so its order decides which subjects are dropped.

The address-scope census has an authored sequence and drops nobody for position. So §4's rule does
not reach it, and this ADR narrows no clause of §4. **A reader must not go looking for an ADR-0058
withdrawal at ADR-0136, because none is owed.** ADR-0136 keeps every word it has.

The shared motive is stated plainly, because it is the thing that misleads: both rules exist so one
tick's output matches the next. They reach different answers because the **populations** differ, not
because the grounds differ. Where a population carries an operator-authored sequence, that sequence
is the order. Where it carries none, a sort supplies one.

### 5. The rule binds the census producer today and the renderer the day one reads it

`AddressScopeCensus` already satisfies §1 and §2, so this ADR changes no Go code.

The Coverage screen does not read the census. It reads the reduced
`map[netip.Prefix]int` and orders the meters by `ListSeeds`, newest declaration first. That surface
is therefore **outside** this rule today, and this ADR does not reach across to change it. The day a
surface renders the census rows themselves, it renders them in the order the census returns them,
and it may not re-sort them.

The mismatch between the two orders is a defect this ruling exposes. It is named in Consequences and
it ships as its own ticket. It is not fixed here.

## Consequences

- **This ADR changes no Go code.** `internal/custody/scopecensus.go` already emits declaration order
  and already collapses a duplicate onto its first occurrence.
- **The triage record for #1310 is wrong on one fact, and the ADR is written to the truth.** It
  states that no renderer consumes the census. `design-system/templates/coverage.tmpl:80` renders the
  shared-edge count today, through `cmd/web/cold.go`'s `apertureMeters`.
- **The Coverage screen renders address scopes newest-first, which is the reverse of this rule's
  order.** It ships as its own ticket. The ticket decides one of two repairs: order the aperture
  meters by ascending `seed.id` for the address-scope half, or have the Coverage handler read the
  census rows rather than the reduced map. The second is the larger change, because the meters
  interleave name scopes and address scopes and the census carries address scopes alone.
- **No test separates this rule from a sorted order.** `scopecensus_test.go:19` declares its two
  scopes in an order that is sorted and declared at once, so it passes under either rule. It ships as
  its own ticket: declare the scopes out of sorted order and assert the declared sequence, and assert
  that a collapsed duplicate keeps the first occurrence's position.
- **ADR-0136 gains no withdrawal and no amendment.** Its §4 stands as written. The cross-reference
  travels in this ADR's front matter alone, so a reader who lands on ADR-0136 first and then on this
  census finds the reconciliation here.
- **`CONTEXT.md` gains nothing.** A render sequence is a property of one surface. It is not a domain
  term, and no clause of `CONTEXT.md` states an order this rule invalidates. The `Custody` entry
  describes the census as a *"census with no denominator"* and says nothing about sequence.
- **The row still carries no product-chosen number and no verdict.** This ADR rules a sequence. It
  adds no field, no threshold and no count, so ADR-0129's #956 constraint and ADR-0013's nag test are
  untouched.
- **A caller that fills `Estate.AddressScopes` from an unordered source produces an unordered
  census.** The rule preserves the caller's order and cannot create one. A future caller that reads
  scopes from a map, or from a query with no `ORDER BY`, breaks the promise without failing a test.
  `ListAddressScopeCidrs` closes `ORDER BY id` today and must keep it.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Sort the census by prefix, following ADR-0136 §4** | It destroys the one fact this population has that the graph's does not. The operator wrote the list, and a sort moves a scope away from where they put it. §4's sort is a last resort for a population with no authored sequence, and importing it here would treat the operator's list as if it had none |
| **Sort by shared-edge count, worst scope first** | It makes the census a triage surface. ADR-0136 §4 refuses severity-first selection for the graph on the same ground, and it applies harder here: the count is display only and gates nothing (ADR-0129 §5), so ranking on it asserts an urgency the model refuses to hold. The list would also reorder under the operator between two cadences as the measurement moves |
| **Leave the order unstated and let each renderer choose** | This is the state that produced the gap. The one shipped surface already chose the reverse order, through a reduction that discards the census sequence, and nobody noticed because nothing was written down |
| **Render a duplicate declaration as two rows** | Two identical rows state one fact twice and read as two scopes. The operator would look for two declarations and find one. The row's remedy is a `Seed` exclusion against a scope, and there is one scope to exclude from |
| **Collapse a duplicate onto its last occurrence** | It moves the entry for no reason a reader can recover. The first occurrence is the act that put the scope in the estate, and the later one adds nothing. Collapsing onto the last would also make the position depend on how many times a caller repeated itself |
| **Rule the order on ADR-0129 as a further amendment** | ADR-0129's subject is that a shared foreign edge is measured by fan-out and not read from a list. Its #956 amendment settled the overlap between a declaration and that measurement, and it created this census as the display for the contradiction. A render sequence is a different question about a surface that amendment built, and filing it inside a five-amendment measurement ADR would bury it |
| **Withdraw ADR-0136 §4's sorted order and rule one order for both populations** | The two populations do not admit one rule. The graph has no authored sequence to preserve, so §4 would lose its only available determinism and gain nothing. §4 also selects the first N under a cap, which this census does not do. ADR-0058 requires a withdrawal at the site that specifies a superseded mechanism, and nothing here supersedes §4 |
| **Fix the Coverage screen's order on this ADR's own branch** | It mixes a production change to `cmd/web/cold.go` and the aperture meters into a documentation change, and it forces the choice between two repairs without its own review. The larger repair rewires the handler from a map lookup to the census rows, across a meter list that also carries name scopes |
