# ADR-0217: the recording-side scope gate gates a denoted dimension alone, and only the edge-fanout facet-less arm fails closed

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1323 ADR gaps: internal/queue (queue, cttail, withdrawal, hot, ctverify, scopegate)](https://github.com/winniel123/verge-asm/issues/1323), gap 7
- **PR that deleted the comment:** [#1322](https://github.com/winniel123/verge-asm/pull/1322)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Rests on:** [ADR-0150](./0150-a-batch-scope-names-its-dimension-in-the-plural-and-a-one-address-fan-out-ships-a-one-element-list-never-a-scalar.md), which rules that a `Batch` scope field names a measurable dimension and ships a list. It is why the gate can read one union shape across every kind, and why an `edge-fanout` job's recorded scope always denotes addresses
- **Rests on:** [ADR-0129](./0129-a-shared-foreign-edge-is-measured-by-fan-out-not-read-from-a-list.md) and its [#954](https://github.com/winniel123/verge-asm/issues/954) amendment, which make the fan-out a **membership-deciding** probe rather than a six-part facet. That is why one arm fails closed and the rest do not
- **Rests on:** [ADR-0201](./0201-a-dispatched-scope-the-leaf-that-reads-it-and-the-recording-gate-agree-on-one-key-name-and-one-address-rendering.md), which rules that the dispatched scope, the leaf, the recorded scope and this gate agree on one key **name** and one address **rendering**. It rules whether a denotation is read at all, and its two failure modes land on opposite sides of this ADR. A **spelling** disagreement leaves the denotation present and drops a legitimate line, which is ADR-0201's stated fault. A **key-name** disagreement leaves the denotation absent, and §1 below then opens the gate. §5 states that consequence
- **Sibling of, and not ruled by:** [ADR-0126](./0126-verbatim-job-output-is-a-fourth-operational-corpus-retired-by-a-duration-dial-that-ships-bounded.md). Its limb 3 keeps the prober transcript **before** this gate, so a dropped line survives as evidence. It rules the corpus. It does not rule what the gate drops

## Context

[`internal/queue/scopegate.go:299`](../../internal/queue/scopegate.go) and `:320` carried this text,
until #1322 deleted it:

```go
// A dimension is gated ONLY where the scope denotes a non-empty authorised set for
// it, so the check can never over-reject a legitimate observation by testing it
// against an empty or absent denotation
```

```go
// nil map => that dimension is not denoted by this scope (or is empty) and is left
// ungated — a job of that kind emits no observation on that dimension, and gating an
// empty set would drop every line.
```

The sweep left two compressed lines at `scopegate.go:76` and `:96`. Nothing on disk states either
rule. That is #1323's gap 7.

**The gate is one union shape read across every job kind.** `parseAuthorizedScope` unmarshals the
`attempted_scope` column into `scopeShape`, which names four address-bearing fields and one
name-bearing field. It sets `a.addrs` only where the union of `addresses`, `services[].address` and
`targets[].address` is non-empty, and `a.names` only where `names` is non-empty. A nil map therefore
means *this scope denotes nothing on that dimension*, and it cannot be told apart from *this scope
denotes an empty set on that dimension*.

**Nine producers write an `AttemptedScope`. Six of them reach this gate.**

| Producer | Site | Fields the union reads | Denotes | Reaches the gate |
| --- | --- | --- | --- | --- |
| `HotJob` | `internal/scan/hot.go:91` | `addresses` | addrs | yes |
| `ColdJob` | `internal/scan/cold.go:115` | `addresses` | addrs | yes |
| `EdgeFanoutJob` | `internal/scan/edgefanout.go:59` | `addresses` | addrs | yes |
| `HTTPIdentityJob` | `internal/scan/httpidentity.go:113` | `targets[].address` | addrs | yes |
| `TLSAcceptanceJob` | `internal/scan/tlsacceptance.go:106` | `services[].address` | addrs | yes |
| `Job` (dns) | `internal/scan/scan.go:72` | `names` | names | yes |
| `ZoneJob` | `internal/scan/zone.go:94` | `domain` only | **neither** | no |
| `CTJob` | `internal/scan/crtsh.go:65` | `domain` only | **neither** | no |
| `CTTailJob` | `internal/scan/cttail.go:155` | `log_id` only | **neither** | no |

The last three are worker-read `Scan`s. `Worker.process` routes them to `completeZone`,
`completeCT` and `completeCTTail` before it ever runs a prober (`worker.go:288`, `:293`, `:297`), so
they never reach `complete` and never reach `gate`.

**So no live producer that reaches the gate denotes nothing.** The both-nil state is reachable by
one route only: `parseAuthorizedScope` returns the zero value where `raw` is empty or does not
unmarshal (`scopegate.go:40`). Our own dispatcher writes that column, so that state is a bug in this
repository rather than a prober's doing.

**This is the qualification the deleted comment omitted.** `gate` short-circuits at
`scopegate.go:118` when both maps are nil and returns every observation untouched. `admits` is never
called. So the fail-closed `edge-fanout` arm at `scopegate.go:97` **never runs** under a scope that
denotes no dimension at all. Its only reachable path is a scope that denotes **names and not
addresses**, which is a dns job. `TestGateDropsEdgeFanoutUnderAScopeDenotingNoAddress`
(`scopegate_test.go:123`) is written against exactly that shape, and
`TestGateLeavesUndenotedDimensionsUngated` (`:82`) locks the short-circuit.

## Decision

> **The recording-side gate gates a dimension only where the job's `AttemptedScope` denotes a
> non-empty authorised set for it. An absent or empty denotation leaves that dimension ungated, and
> a scope that denotes no dimension at all leaves the whole gate a no-op. The one exception is the
> facet-less `edge-fanout` arm, which fails closed, because there an absent address denotation is
> not an undenoted dimension — it is a contradiction between the line and the job that carried it.
> The rule binds every producer of an `AttemptedScope` in `internal/scan`.**

Five limbs.

### 1. An undenoted dimension is a question with no ground, and a question with no ground is not answered "no"

The gate asks one question of each observation: *is this subject inside the ground this job was
dispatched over?* A denotation is what supplies the ground. Where the scope names no ground on a
dimension, the question is unanswerable, and the gate declines to answer it rather than answering it
in the negative.

The failure directions are not symmetric, and that is the whole of the argument.

- **Over-rejection destroys a measurement the estate paid for, and it is silent.** A dropped line
  writes one log entry (`scopegate.go:128`) and no durable record. The span it would have continued
  goes stale, and the operator reads a `Gap` caused by the recorder rather than by the estate.
- **Under-rejection admits a line whose subject the job did not name.** It is the hazard #773 exists
  for, and it is bounded by everything downstream: the membership fold still tests every survivor,
  the `Custody` gate still refuses an unowned address, and the prober transcript still holds the
  pre-gate bytes (ADR-0126 limb 3).

A silent loss of true measurement is worse than a bounded admission of an unauthorised one.

### 2. A job emits observations only on the dimensions its own leaves measure

The gate does not need to answer for an undenoted dimension, because the case does not arise from a
correct producer. A dns job denotes names and emits `resolution` and `dns-record` lines about names.
A hot job denotes addresses and emits `reachability`, `certificate`, `http-identity` and
`tls-acceptance` lines about addresses and services.

The ungated arm therefore costs nothing today. What it buys is that a producer which grows a second
dimension does not silently lose every line on it. The alternative loses those lines with no signal
at all.

### 3. The `edge-fanout` arm fails closed, because it answers a different question

The facet-less arm at `scopegate.go:91` splits on `Kind` rather than on a dimension.

- A facet-less line of **any other kind** is admitted (`scopegate.go:93`). That is limb 1 again.
  `TestGateLeavesOtherFacetlessKindsUngated` locks it.
- A facet-less line of kind `edge-fanout` under a scope with no address denotation is **dropped**.

That drop is not an over-rejection, for one reason: an `edge-fanout` job's scope **always** denotes
addresses. `EdgeFanoutJob.AttemptedScope` marshals `edgeFanoutScope{Addresses: j.Addresses}`, and
`BuildEdgeFanoutJobs` yields a job only where `len(addrs) > 0`. ADR-0150 fixes the field's name and
shape, so the union reads it under every tier.

So a line claiming kind `edge-fanout` under a scope with no address denotation did not come from an
`edge-fanout` job. The gate is not testing a subject against an empty ground. It is reading a
contradiction, and there is no legitimate line behind it to lose.

**The stakes are what make the direction worth stating.** An `edge-fanout` measurement decides
membership rather than recording a value (ADR-0129, #954). An injected line feeds the custody
extension's veto an answer nothing measured (#985), and the veto decides what the product probes
next. Every other facet the gate handles records a value that a later cadence corrects.

`toEdgeFanoutRows` (`edgefanout.go:73`) already refuses a line of that kind on a job of a different
kind. The two gates are independent, and both survive: one reads the job's declared `Kind` and the
other reads the recorded scope's denotation.

### 4. A drop never fails the job

`gate` returns the surviving slice and `complete` proceeds. A dropped line costs one log line
carrying the kind, the facet, the subject and the address.

Failing the job would hand any prober that can emit one out-of-scope line a way to void the whole
`Batch`, including the legitimate observations beside it. A compromised prober would then hold a
denial of service over the estate's own measurement, which is a strictly worse outcome than one
recorded line the fold ignores.

### 5. The rule binds every `AttemptedScope` producer, and nothing enforces that

The gate's protection on a dimension is exactly as strong as the producer's denotation of it. A kind
that stops denoting its dimension — a refactor that renames `addresses`, a scope record that drops
the field, a fan-out that emits an empty list — opens the gate for every job of that kind, silently
and with no test failure in `internal/queue`.

That obligation lands on `internal/scan`, in nine files this package does not own. It is stated here
because there is no other place that sees both halves.

**A key-name disagreement and a spelling disagreement fail in opposite directions**, and
[ADR-0201](./0201-a-dispatched-scope-the-leaf-that-reads-it-and-the-recording-gate-agree-on-one-key-name-and-one-address-rendering.md)
rules that the four sites must agree on both. A renamed key empties the union, so the denotation goes
absent and §1 opens the gate. A changed address rendering leaves the denotation present and every
membership test fails, so the gate drops legitimate lines. The loud failure is a change to the
spelling. The silent one is a change to the key name, and it is the one this limb names.

## Consequences

- **This ADR changes no Go code.** Both directions already ship, and
  `internal/queue/scopegate_test.go` locks seven cases including both.
- **Neither direction is a defect.** They answer different questions, per §1 and §3. The asymmetry
  is the ruling, not an inconsistency to repair.
- **The `default` arm is unreachable today, and it is a fail-open.** Six facet constants exist in
  `internal/measure` — `reachability`, `certificate`, `http-identity`, `resolution`, `dns-record`
  and `tls-acceptance` — and the switch names all six. A **seventh** facet would fall to
  `default: return true` and be ungated on every kind, with no compile error and no test failure.
  That is limb 1's direction applied by omission rather than by decision. **A test that fails when a
  facet constant exists and the gate does not name it ships as its own ticket.**
- **`scopegate.go:96` carries a wrong citation.** It cites ADR-0129 §6, which rules that collection
  is CT plus an active no-SNI handshake and states nothing about a job's scope shape. The claim it
  supports is ADR-0150's. The replacement is in this issue's manifest.
- **`scopegate.go:117` carries a wrong citation.** It cites ADR-0001 for the rule that a drop must
  not fail the job. ADR-0001 chooses the stack and the runtime and rules nothing about it. §4 rules
  it, and the replacement is in the same manifest.
- **The dead `#773` citation is out of scope.** `scopegate.go` carries it four times and the issue
  number does not resolve. #1323 excludes repairing it, and it needs its own record.
- **[`CONTEXT.md`](../../CONTEXT.md) gains nothing.** The gate mints no subject and moves no domain
  term. It decides which lines become observations.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Fail closed on every undenoted dimension** | Drops every legitimate line the moment a producer grows a dimension its scope record does not yet name, and the loss is silent: one log line, no durable record, and a `Gap` the operator reads as an estate fact. The three worker-read kinds would also be gated to nothing if they ever reached the gate |
| **Fail open on the `edge-fanout` arm too, for consistency** | Consistency with a rule that does not apply. §3's arm reads a contradiction rather than an undenoted dimension, and the line it would admit decides membership rather than recording a value. An injected fan-out line changes what the product probes next, and no later cadence corrects it |
| **Refuse the job where the scope denotes no dimension at all** | The state is reachable only from an empty or unparseable `attempted_scope`, which our own dispatcher writes. Failing the job turns a defect in this repository into lost measurement across every job of that kind, and the defect would still be invisible. `parseAuthorizedScope` already names the state and fails open on it deliberately |
| **Gate on the job's `Kind` alone and drop the scope denotation** | The kind says which leaves ran. It does not say which subjects they were dispatched over, and that is the whole question #773 asks. Two hot jobs of the same kind carry different addresses, and the gate must tell one from the other |
| **Make the denotation explicit — an `absent` marker distinct from an empty list** | ADR-0150 §3 already fixes that an empty scope is written as `[]` rather than `null`, and the union collapses both to a nil map. Splitting them adds a third state to nine producers and to one shared shape, to distinguish two cases that this rule treats identically |
| **State the rule as a comment on `admits`** | It binds nine producers in a package `internal/queue` does not own, and the comment policy admits one clause beside one statement. #1322 cut the two-clause version to one line, and that line cannot carry the producer obligation §5 states |
