---
number: 1945
title: "One address-scope binding per comparison, held by a seam and fenced by an AST gate"
slug: one-address-scope-binding-per-comparison-held-by-a-seam-and-fenced-by-an-ast-gate
date: 2026-09-14
status: accepted
source: grilling
ticket: 1945
proof: {ticket: 1940}
relations:
  - {kind: amends, adr: 1895, clause: "9"}
  - {kind: rests-on, adr: 133, clause: "4"}
  - {kind: rests-on, adr: 161}
---

# ADR-1945: One address-scope binding per comparison, held by a seam and fenced by an AST gate

## Decision

**Every leg of one comparison classifies under one address-scope binding.** The comparison takes
that binding from a struct it builds once. The `internal/queue` package holds the shape already, in
`batchLegs.covered`. `cmd/web` gains the mirror at `exposureCountDeltas`. An AST call-graph test in
each package fails when a comparison's graph reaches a binding producer twice.

Rejected: count `vantageclass.Derive` call sites and fail on a new one. It is cheaper and it is
blind — the three acquisitions added on 2026-09-14 reuse derivation points that already existed.
Rejected: one seam for every caller. Three callers hold three different coverage-error fallbacks, so
a shared seam either drops them or carries all three to ten callers that reach none.

A reader asks why, because
[ADR-1895](./1895-a-vantage-class-reads-present-configuration-so-a-historic-observation-carries-none.md)
§4 already states the rule and two tests already hold it. Neither counts acquisitions.

Reversal deletes two test files and inlines one field.

## 1. What ADR-1895 §9 left open, and the two censuses that moved apart under it

[ADR-1895](./1895-a-vantage-class-reads-present-configuration-so-a-historic-observation-carries-none.md)
§4 rules that the two legs of a comparison read one boundary. Its §9 names the gap in its own
words: *"The census in §1 is a reading of the tree on this date, not a fence. Nothing stops an
eleventh call site, and no check counts them."* It hands the repair to a later session, and only on
a condition — *"if that becomes a real risk"*.

ADR-1895 is dated 2026-09-13. On 2026-09-14 three pull requests each landed one more acquisition of
an address-scope binding:

| Commit | Site | What it reads |
| --- | --- | --- |
| `fee4ec0` ([#1983](https://github.com/winniel123/verge-asm/pull/1983)) | `apertureVantageClasses` (`cmd/web/aperturestatement.go`) | every `Vantage` row |
| `d45a973` ([#1982](https://github.com/winniel123/verge-asm/pull/1982)) | `assetReachLegs` (`cmd/web/subjects.go`) | the open reachability spans |
| `ad7d814` ([#2009](https://github.com/winniel123/verge-asm/pull/2009)) | `serviceReachLegs` (`cmd/web/subjects.go`) | the open spans for one service |

**All three are correct.** None compares across time. Each reads present state and asks which class a
live row falls in, which is what the predicate is for.

Two censuses cover this ground, and the day moved them apart. The distance between them is the
argument.

**The derivation census did not move.** ADR-1895 §1 counts ten call sites reaching
`vantageclass.Derive`. It still reads ten. Every one of the three new readers arrives at a derivation
point that already existed — `apertureVantageClasses` through `listedVantageClasses` and
`vantageFactsClass`, and both `subjects.go` readers through `collapseReachLegs`. ADR-1895 §9's
eleventh call site never arrived, and a check waiting for it would have waited through all three.

**The acquisition census moved by three.** The triage of
[#1940](https://github.com/winniel123/verge-asm/issues/1940) counted ten acquisitions across seven
files. A reading of `origin/main` at `0a12dfb` finds thirteen, across nine: eleven in `cmd/web`
through `addressScopeCovered`, and two in `internal/queue` through `coveredAddressScope`.

So the surface the rule is about grows while the number ADR-1895 §1 wrote down stands still, and it
grows through correct additions. The fence cannot be a cap on either population. It has to be a
statement about one comparison's call graph.

## 2. The seam covers the comparison path only

The rule is about a comparison, so the seam covers one. The `batchLegs` struct already carries
`covered` as a field. Then `readBatchLegs` sets it once, and both `legsFromCurrent` and
`legsFromAt` read it. The `exposureCountDeltas` function gains the same shape. One struct holds the
binding and the two snapshots, and both legs consume it. Both functions bind once today, each under
a comment that says so, and neither under a check.

The alternative was a seam every caller takes. It loses on three callers, and it loses on each of
them differently. Each meets a coverage error its own way, and each states its reason beside the
fallback:

| Caller | On a coverage error |
| --- | --- |
| `dashboardData` (`cmd/web/auth.go`) | binds to a predicate that covers nothing, and renders |
| `fillVantagesSection` (`cmd/web/settings.go`) | the same fallback |
| `apertureVantageClasses` (`cmd/web/aperturestatement.go`) | returns a withheld read, and renders no class |

The first two take a class that over-reports a vantage as `internet` over one that mislabels it
`internal`. The third refuses that trade. The row it feeds is the remedy for a missing address
scope, and a wrong class there hides the remedy.

A universal seam must carry all three shapes to ten callers that reach none of them. Or it drops
them, and changes behaviour at three sites that are correct today. Neither is a price this decision
is buying anything with.

`rulesOpenedByGapClose` in `internal/queue/gapclose.go` stays outside the seam for the same reason.
It acquires its binding directly, reads present state through `legsFromCurrent`, and starts no
comparison. The gate allowlists it by name.

## 3. What the gate proves, and three things it does not

Each gate parses its own package. It resolves the call graph of each comparison function, and fails
when that graph reaches a binding producer more than once. The model is
`cmd/web/act_conformance_test.go`, which fences an unrelated rule the same way. That file opens by
stating what it does not reach. This section is that statement.

**The root set is a list, not a derivation.** The gate walks from named comparison functions. A
wholly new comparison escapes it, because nobody adds its function to the root set. That is the
acquisition census of §1 again, shortened from thirteen entries to two.

What changes is the failure mode. A stale entry now fails the test rather than passing in silence.
The root set and the allowlist are asserted live against the package, the way
`TestActExemptionsAndPendingEntriesAreLive` asserts its own.

**It counts acquisitions, never values.** One acquisition on the graph passes. The gate does not
prove that both legs read the same variable. A comparison that binds once, then rebinds the name
before the second leg, would pass.

ADR-1895 §10's two tests hold that instead. Each measures the same rows on both legs, and fails on
any disagreement. The gate holds the shape, and those two hold the value. Neither replaces the other.

**Dynamic dispatch is invisible.** A binding reached through a function value, a struct field, or an
interface method is not a call on the static graph. Both producers are called directly at every site
today, and the gate reads that tree. It would not read a tree where one of them arrived through a
stored `func(netip.Addr) bool`.

## 4. Why a derivation-site count is the cheaper fence and the worse one

The rejected alternative is a check that counts `vantageclass.Derive` call sites. It fails when the
number moves. It needs no call-graph walk and no root set. §1's finding is that it fails in both
directions at once.

**It stays silent when it should speak.** The three acquisitions of 2026-09-14 each reuse a
derivation point that already existed. So the count stayed at ten through all three. Every one of
them widened the surface this rule governs, and none of them moved the number.

**It speaks when it should stay silent.** A derivation point added inside a present-state reader is
correct, and the count fires on it anyway. The fact that separates a correct read from a split
binding is not how many derivations exist. It is which of them sit on one comparison's graph.

A gate that fires on every addition trains the next session to raise the number and continue. That
is the failure mode
[ADR-0161](./0161-the-backup-allowlist-and-the-exclusion-list-partition-the-business-schema-so-a-new-table-is-classified-by-a-human-or-the-test-fails.md)
avoids. Its exclusion list carries a reason per entry rather than a total, so a human classifies a
new table rather than counting it.

## 5. Consequences

- **No migration and no schema change.** Nothing new is recorded, and no query moves.
- **Both producers stay.** `addressScopeCovered` and `coveredAddressScope` keep every present-state
  caller they have. Neither is deleted, and neither moves out of reach.
- **[ADR-0133](./0133-an-address-exclusion-is-a-limb-of-the-custody-derivation.md) §4's one-binding
  comment keeps its job.** That comment fences a wider invariant, which is one binding shared by
  batch gating and every render. This gate fences a narrower one, over a single comparison's call
  graph. It covers a part of the same ground, and replaces none of it.
- **ADR-1895 §10's two per-site tests stay unchanged.** §3 states what each layer holds.
- **ADR-1895 §9's claim narrows rather than retires.** A second acquisition inside either named
  comparison now fails CI. A fourteenth present-state acquisition is still free, and a comparison
  written outside the root set still escapes. This ADR amends that clause, and does not close it.
- **`CONTEXT.md` gains `Address-scope binding`**: the predicate over presented addresses, derived
  from the address-scope `Seed`s held at one instant, under which a `Vantage class` is read.

## 6. Proof

`{ticket: 1940}`. [#1940](https://github.com/winniel123/verge-asm/issues/1940) is open, and its
acceptance criteria name this rule as the thing to show. Each package gets a test that fails when a
second acquisition lands inside that package's comparison. Each gate opens with a block naming what
it proves and what it does not. Each allowlist is asserted live, so a stale entry fails.

The proof is a ticket rather than a test because the test does not exist yet. This ADR adds one
file. #1940 lands the seam, both gates, and the `CONTEXT.md` term. The test proof belongs to that
pull request.
