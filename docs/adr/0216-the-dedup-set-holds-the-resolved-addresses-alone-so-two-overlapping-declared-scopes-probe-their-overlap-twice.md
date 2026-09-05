# ADR-0216: the dedup set holds the resolved addresses alone, so two overlapping declared scopes probe their overlap twice

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1323 ADR gaps: internal/queue (queue, cttail, withdrawal, hot, ctverify, scopegate)](https://github.com/winniel123/verge-asm/issues/1323), gap 6
- **PR that deleted the comment:** [#1322](https://github.com/winniel123/verge-asm/pull/1322)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Amends:** [ADR-0127](./0127-the-address-scope-range-cap-has-no-ceiling-a-large-scope-is-priced-not-gated.md). Its *"Does the enumeration stream?"* row says *"Single-probing is required (dedup against the small resolved set)."* Read alone, the first clause forbids what the code does. The amendment states the clause's true reach at its own site, per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md), and its wording is in this issue's manifest
- **Rests on:** [ADR-0127](./0127-the-address-scope-range-cap-has-no-ceiling-a-large-scope-is-priced-not-gated.md) again, for the ground. Its guarantee that *"memory is never a ceiling bound … so no record holds the whole scope"* is what forces the double probe, and its no-upper-bound ruling is what makes the alternative impossible rather than expensive
- **Rests on:** [ADR-0047](./0047-an-address-scope-is-its-own-enumeration.md), which rules that a declared CIDR produces *"every address in it, walked every cadence"*. A scope's walk is a property of that scope's own declaration
- **Bounded by:** [ADR-0133](./0133-an-address-exclusion-is-a-limb-of-the-custody-derivation.md) §3, which puts the exclusion skip on every tier that enumerates. A duplicate walk never re-walks an excluded address, on either pass
- **Sibling of, and not ruled by:** [ADR-0195](./0195-the-address-scope-census-renders-in-declaration-order-and-a-scope-declared-twice-renders-once.md). It rules that the address-scope **census** renders a scope declared twice once. This rules that the **enumeration** walks it twice. The two are not in conflict and the pair is worth holding together: the census answers *what did the operator declare*, and the enumeration answers *what is probed this tick*. §4 below is why the second must not adopt the first's dedup
- **Sibling of, and not ruled by:** [ADR-0188](./0188-a-fan-out-record-resolves-over-an-estate-as-a-pure-function-of-it-so-the-floor-clears-on-entry-and-the-candidates-are-first-seen-order.md). It rules `ExtensionCandidates` distinct and in first-seen order, which is the set `EdgeFanoutPopulation` seeds its dedup map from. It rules that half of the population. This rules the declared-scope half, which is deduped against that set and never against itself

## Context

[`internal/queue/hot.go:171`](../../internal/queue/hot.go) carried this text, until #1322 deleted it:

```go
// Two declared scopes that OVERLAP are not deduped against each other, so the overlap
// probes twice; that is the accepted cost of not holding the whole scope in a map, and
// each probe is its own idempotent Batch.
```

The sweep left one compressed line at `hot.go:139`. Nothing on disk states the rule, and
[ADR-0127](./0127-the-address-scope-range-cap-has-no-ceiling-a-large-scope-is-priced-not-gated.md)
carries a clause a reader reaches first that appears to forbid it. That is #1323's gap 6, and the
conflict is why it needed a ruling rather than a transcription.

**Two enumerators state the rule, and they agree line for line.**

| Enumerator | Site | What the dedup set holds | Callers |
| --- | --- | --- | --- |
| `candidateAddrs` | [`internal/queue/hot.go:120`](../../internal/queue/hot.go) | `resolved`, sized `len(resolved)` | `fanOutHot` (`hot.go:35`), `fanOutCold` (`cold.go:29`) |
| `Estate.EdgeFanoutPopulation` | [`internal/custody/candidates.go:36`](../../internal/custody/candidates.go) | `ExtensionCandidates()`, sized `len(candidates)` | the `edge-fanout` `Scan` |

In both, the scope loop **reads** the set and never writes to it. `hot.go:140` is
`if _, ok := seen[a]; ok { continue }` with no matching insert, and `candidates.go:53` is the same
line. So an address enumerated out of a declared scope is compared against the resolved set and is
never added to it.

**The rule is locked by exactly one test, in the package that is not the one gap 6 names.**
`TestEdgeFanoutPopulationDoesNotDedupOverlappingScopes`
(`internal/custody/candidates_test.go:209`) declares `104.16.132.0/31` and `104.16.132.0/30` and
asserts the yielded sequence is `.0 .1 .0 .1 .2 .3` — the overlap, twice, in declaration order.
`internal/queue/candidate_test.go` holds seven tests and none of them overlaps two scopes.
`TestCandidateAddrsUnionsMultipleScopes` uses two **disjoint** `/31`s.

**What the dedup set is actually for.** `TestCandidateAddrsDedupsResolvedAndEnumerated`
(`candidate_test.go:27`) is the test that fixes it. A resolved address that also falls inside a
declared scope is yielded once, from the resolved arm. That is the whole job of `seen`, and it is
word for word what ADR-0127's parenthesis says: *dedup against the small resolved set*.

**The alternative is not expensive. It is unbounded.** ADR-0127's ruling is that the operator cap
has **no upper bound** — *"a ceiling would be #27's invented threshold sitting in the safety path"* —
and it accepts an IPv6 scope above 2^32 as an ordinary over-large scope. A dedup set over the
enumerated addresses must hold one map entry per address walked. One declared IPv4 `/8` is
**16,777,216** entries keyed on a 24-byte `netip.Addr`, so hundreds of megabytes for one scope. An
IPv6 `/104` is the same count, and a `/96` is 2^32. There is no size at which the map is safe,
because there is no size at which a declaration is refused.

**One figure in the triage record is wrong, and it matters here.** The brief for this ruling states
that ADR-0133 §3 *"caps a tick at 65,536 addresses"*. It does not. §3's sentence is *"an excluded
`/16` inside a declared `/8` is 65,536 addresses walked per tick and refused one at a time"*, and
that number is the size of a `/16`, offered as the cost the exclusion skip avoids. The nearest
65,536 in the tree is `seed.maxEnumCapHint = 1 << 16` (`internal/seed/seed.go:98`), whose own
comment reads *"caps the size guess only, never a walk"*. **No rule caps a tick.** ADR-0127 removed
the only ceiling there was.

## Decision

> **The candidate enumerator's dedup set holds the resolved addresses alone. It never absorbs an
> address that a declared scope contributed. Two declared scopes that overlap therefore probe their
> overlap once each, by choice. The cost is a duplicate measurement, it is linear in the
> declarations, and it is reported through the surfaces ADR-0127 already fixed. ADR-0127's
> single-probing clause binds the resolved set, and it does not bind an enumerated scope.**

Five limbs.

### 1. ADR-0127's clause is bounded by its own parenthesis

*"Single-probing is required (dedup against the small resolved set)"* states the requirement and its
domain in one sentence. The domain is the resolved set. Both enumerators implement exactly that.

The clause is amended rather than reversed, because the first half read alone would make the shipped
code a violation, and a reader lands on the first half. ADR-0058 requires the correction at
ADR-0127's own site, and the wording is recorded in this issue's manifest.

### 2. The ground is ADR-0127's own memory guarantee, and it is not a preference

ADR-0127's row promises that *"memory is never a ceiling bound … so no record holds the whole
scope"*. A dedup set over the enumerated addresses **is** a record that holds the whole scope. The
two cannot both be true.

Under a no-upper-bound cap the guarantee is load-bearing and the dedup set is not merely costly. It
grows with the declaration, and no declaration is refused. `seed.EnumerateAddresses` is an
`iter.Seq` for the same reason, and its own comment says so.

### 3. A scope's walk is a property of that scope's declaration, not of the others

ADR-0047 rules that a declared CIDR produces every address in it, walked every cadence. A dedup set
shared across scopes breaks that. Declare `10.0.0.0/8`, then declare `10.1.0.0/16`, and the second
declaration would contribute nothing at all: every address it names was already yielded by the
first. Whether a scope walks would depend on the order in which the operator happened to declare
their scopes.

The product has no surface that would explain that. `Coverage` states `declared / current` for the
estate, and no `Batch` records which declared scope produced its address — `HotJob` carries a
vantage, an address list and the port sets, and `ColdJob` carries the same shape with a port range.
So a scope silently contributing nothing would be silent everywhere.

### 4. The cost is a duplicate measurement, and it is linear

The walk is `Σ|scope_i|` rather than `|∪ scope_i|`. Two identical declared `/16`s cost 131,072
addresses per tick instead of 65,536. There is no super-linear term and no term that depends on the
overlap's shape.

What the duplicate buys the operator is nothing, and what it costs is bounded three ways.

- **It cannot produce a wrong reading.** ADR-0005 makes each probe its own `Batch`, and the
  membership fold is idempotent over a repeat observation of the same subject. The second `Batch`
  continues the span the first one continued. No `Gap` opens and no drift fires.
- **It never re-walks an excluded address.** ADR-0133 §3 puts the exclusion predicate inside the
  scope loop on every tier. `hot.go:144` and `candidates.go:61` each apply it on every pass.
- **The lag it causes is reported, not hidden.** ADR-0127 already rules this. A scope that cannot
  finish inside its cadence is *"reported, never hidden or refused"*; `Coverage` states
  `declared / current` and `Scans` states the predicted and effective cadence; ADR-0005's overlap
  rule turns an uncompletable pass into a legible skip. An operator who declares a large overlap
  sees the effective cadence stretch and the current count fall behind the declared count, on the
  surfaces that already exist for a scope that is simply large.

### 5. The rule does not reach the resolved arm, and it does not reach a within-scope repeat

- **The resolved set is still deduped**, against itself and against every scope. That is §1's
  domain, and `TestCandidateAddrsDedupsResolvedAndEnumerated` locks it.
- **One scope never yields the same address twice.** `seed.EnumerateAddresses` walks a masked prefix
  once. The duplicate arises between two declarations and nowhere else.
- **The exclusion predicate is not applied to the resolved arm.** That is ADR-0133 §1's ruling — an
  exclusion cuts the `Seed` limb alone — and it is untouched here.

## Consequences

- **This ADR changes no Go code.** Both enumerators already behave this way.
- **[ADR-0127](./0127-the-address-scope-range-cap-has-no-ceiling-a-large-scope-is-priced-not-gated.md)
  gains one amended clause**, at its own site, in the *"Does the enumeration stream?"* row. The
  anchor and the replacement are in this issue's manifest, and the amendment is now applied in that
  row. Without it, the next reader of that
  row grades the shipped enumerator a defect and writes the dedup set that ADR-0127's neighbouring
  guarantee forbids.
- **`candidateAddrs` has no test for the rule, and that is a defect.** A change that added
  `seen[a] = struct{}{}` inside the scope loop would pass all seven tests in
  `internal/queue/candidate_test.go`, and it would silently make one declared scope's coverage a
  function of the others. The custody twin is locked and this one is not. **The mirror test ships as
  its own ticket**, asserting the `/31`-inside-`/30` sequence against `candidateAddrs`.
- **An operator who declares a large overlap pays twice and is told.** No new surface is needed and
  none is added. The lag reads exactly as it does for one scope of the summed size.
- **[`CONTEXT.md`](../../CONTEXT.md) gains nothing.** No domain term moves. A duplicate probe mints
  no subject and changes no membership.
- **Nothing enforces the rule beyond the custody test.** The two enumerators are in different
  packages and share no code. A change to one is not a change to the other, and this ADR is the only
  thing that says they must agree.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Rule ADR-0127's clause correct and record the double probe as a defect** | The fix is a dedup set over every enumerated address, which contradicts ADR-0127's own guarantee two clauses later that no record holds the whole scope. Under a cap with no upper bound the set is unbounded, so the defect could not be repaired at all for a declared `/8` or for any IPv6 scope above `/104` |
| **Dedup only below a size threshold — hold the map for a small scope, stream a large one** | Two behaviours for one declaration, switching on a number with no owner and no derivation. ADR-0049 rejected exactly that shape, and ADR-0127 rejected the ceiling that would have to set the number. It also gives a scope's coverage a discontinuity: adding one address to a scope changes whether a *different* scope walks its overlap |
| **Subtract the overlap with prefix arithmetic before enumerating** | `internal/custody/candidates.go:62` already refuses this in one line — *"never prefix arithmetic: subtraction is easy to get wrong at the family boundary"*. Subtracting one CIDR from another yields a set of prefixes, the IPv4-in-IPv6 mapped forms make containment subtle, and a wrong subtraction silently stops probing declared ground |
| **Deduplicate at the queue instead, with a unique key on the enqueued job** | Moves an unbounded set from the dispatcher's memory into an index on the `job` table, and the fan-out is streamed and chunk-committed (ADR-0127), so the key would have to survive across commits within one tick. It also makes a second declaration's job silently vanish, which is §3's invisible outcome with an extra moving part |
| **Fold overlapping declarations at declaration time, so the operator holds one merged scope** | Destroys the operator's own statement of their estate. Two ranges declared for two reasons are two facts, and `Coverage` and every future per-scope surface read the declarations. ADR-0047 makes a declaration the unit, and a merge would answer a question the operator did not ask |
| **Leave the rule uncited and let ADR-0127's clause stand** | The state #1323 recorded. A reader reaches ADR-0127 first, reads *single-probing is required*, and grades the enumerator wrong. The next change writes the map, and it is the map ADR-0127's own memory guarantee forbids |
