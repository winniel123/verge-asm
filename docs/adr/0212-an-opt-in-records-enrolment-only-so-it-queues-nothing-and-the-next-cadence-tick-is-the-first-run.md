# ADR-0212: an opt-in records enrolment only, so it queues nothing and fires nothing, and the tier's next cadence tick is its first run

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1321 ADR gaps: internal/queue (#1199)](https://github.com/winniel123/verge-asm/issues/1321), gap 6
- **PR that deleted the comment:** [#1327](https://github.com/winniel123/verge-asm/pull/1327)
- **Rests on:** [ADR-0044](./0044-a-one-off-measurement-has-no-currency.md), which rules that opt-in is **per `Seed` scope**, that the cold `Scan` ships **configured and disabled**, and which refuses *"Fire the sweep at `Seed` declaration"* in its rejected alternatives. It rules the **declaration** act and the granularity. It does not rule the **opt-in** act. This ADR extends its reach to that act and withdraws nothing it says
- **Rests on:** [`v1-spec.md`](../spec/v1-spec.md) §3.4, whose cold row states the tier *"never runs unasked, including at onboarding"*. It rules the **unasked** case. The opt-in is the ask, and §3.4 does not say what the ask does. This ADR extends its reach to the ask and withdraws nothing it says
- **Bounded by:** [ADR-0005](./0005-scan-execution-model.md), which rules that a manual run dispatches an existing `Scan`. §5 states that a manual dispatch after an opt-in is a second operator act, and is not a consequence of the opt-in
- **Corrects:** the deleted comment's own wording. It claimed the cadence tick is *the sole firing*, and §5 measures that this is false

## Context

`internal/queue/cold.go:22` carried this, in the file-header block above `fanOutCold`, until #1327
deleted it:

```go
// ... It is dispatched only when the operator has
// opted a scope in — which flips the cold Scan enabled and puts it in
// ListEnabledScans — and even then produces no jobs when the opted-in scope
// admits no address, a legible empty state. Nothing here enables the tier or
// fires it on opt-in: the web opt-in handler reconciles the enabled flag, and
// this fan-out runs only on the monthly cadence tick.
```

The rule binds the web opt-in handler as much as the dispatcher, and it is the one act neither
ADR-0044 nor the v1 SPEC rules.

**What the opt-in handler actually does is three statements.** `setColdScope`
(`cmd/web/cold.go:57`) parses the seed id, then calls `OptInColdScope` — one `INSERT INTO
cold_scan_scope (seed_id, created_by)` — then `SyncColdScanEnabled`, which is
`SET enabled = EXISTS (SELECT 1 FROM cold_scan_scope)`, then redirects. There is no `EnqueueJob`, no
`Trigger`, and no dispatch on that path.

**The enrolment is read at the tick, never carried to it.** `fanOutCold`
(`internal/queue/cold.go:15`) calls `coldScope`, which reads `ListColdScopeSeeds` at dispatch time
and builds the tier's address set from every opted-in seed at once. So the fan-out is **one dispatch
over the union of the enrolled scopes**, not one dispatch per enrolment. That property is what makes
the ruling below cheap, and §4 turns on it.

**Two sources already touch the topic, and neither reaches the act.**

| Source | What it rules | Why it does not contain this |
| --- | --- | --- |
| ADR-0044, rejected alternatives | Refuses *"Fire the sweep at `Seed` declaration"* on §6.4's *"Never scan on config save"* and on `Seed`'s *"a boundary, not a starting point"* | A `Seed` declaration is a different act on a different object. A `Seed` may be declared and never opted in — the tier ships with an **empty** scope list |
| ADR-0044, Decision table | Opt-in is per `Seed` scope, and the tier ships configured and disabled | Rules the granularity of the opt-in and the tier's shipped state. Says nothing about what happens at the moment of the opt-in |
| `v1-spec.md` §3.4, cold row | *"never runs unasked, including at onboarding"* | Rules the unasked case. The opt-in **is** an ask, so §3.4 stops exactly where this question starts |

`cmd/web/cold.go:59` already cites both, which is why the gap went unnoticed: the handler looks
covered, and the citation is true of the half those sources rule.

**The deleted comment overstates, and the survivor still does.** `Dispatcher.Trigger`
(`internal/queue/queue.go:98`) reads the `Scan` and, where `Enabled` is true, calls `fanOut` — which
routes the cold kind to the streamed fan-out (`queue.go:111`). `cmd/web/scantrigger.go:52` refuses a
trigger only **while the scan is disabled**, which is the state the opt-in has just ended, and its
copy says so: *"the disabled cold tier cannot be triggered at all"*. So an operator may opt a scope
in and then press *Run now*. The monthly tick is **not** the sole firing, and §5 states the honest
rule.

## Decision

> **An opt-in records enrolment and nothing else. It queues no job and it fires no fan-out. The scope
> joins the tier, and the tier's next cadence tick is its first run — unless the operator performs a
> separate manual dispatch, which is a second act and never a consequence of the opt-in.**

### 1. The opt-in writes enrolment and reconciles the enabled flag

Those are the two writes, and there is no third. `OptInColdScope` records the enrolment.
`SyncColdScanEnabled` derives the tier's enabled flag from whether **any** scope is enrolled, so the
flag is a function of the enrolment set rather than a second thing an operator sets. Opting the last
scope out disables the tier by the same statement.

### 2. The enrolment is read at the tick, and is never carried to it

The tier's fan-out reads `cold_scan_scope` when it runs. Nothing is staged, nothing is queued, and no
state travels between the click and the tick beyond the enrolment row itself. A scope enrolled and
withdrawn between two ticks is never measured, and that is correct.

### 3. An opt-in is a statement about the future, not a request for work now

This is the ground. The operator is naming which scopes the tier covers **from now on**. The tier's
cadence is what decides when coverage is taken, and ADR-0028 sets a currency bound against that
cadence.

The second ground is load. Firing on the act makes the tier's load a function of when operators
happen to click, which is exactly the unpredictability a cadence exists to remove. `#4` §6.4 states
that harm as a number — *"otherwise editing 20 targets fires 20 simultaneous scans"* — and §Context's
measurement shows what the enrolment shape buys instead: twenty enrolments are still **one** dispatch
at the next tick, over the union of the twenty scopes.

### 4. How this reads `#4` §6.4, and which reading it takes

`#4` §6.4 says, in full: *"**Never scan on config save.** Adding a target should queue a scan, not
fire one — otherwise editing 20 targets fires 20 simultaneous scans."*

The sentence admits two readings of *queue*:

- **(a) enqueue a job** that the next dispatcher tick drains.
- **(b) enrol into a cadence** whose own tick runs the work.

**This ADR takes (b) for this tier.** Three grounds, in order of weight.

**§6.4's own stated harm is concentration, and (a) does not remove it.** Reading (a) removes the
synchronous fire and keeps the pile-up: twenty opt-ins put twenty full-range sweeps on one tick.
Reading (b) removes both, because the cold fan-out is one dispatch over the union of the enrolled
scopes (§Context). The clause's `otherwise` names the harm, and only (b) answers it.

**Reading (a), applied to this tier, mints the object ADR-0044 rules is not expressible.** A sweep
queued by an opt-in is a full-range measurement with no cadence of its own, so `k × cadence` has no
value and every timeline it opens either reads current forever or opens a `Gap` at a time no rule can
compute. That is ADR-0044's decisive argument against the one-off, arriving through a different door.

**ADR-0044 itself uses reading (a), and it is not ruling this question when it does.** Its rationale
says *"A queued sweep runs at the next tick, under ±20 % jitter, inside operator-set quiet hours"* —
and it says it to defeat the claim that an onboarding sweep happens while the operator watches. It is
an argument about attendance, made against a tier that runs daily. It is not a definition of *queue*
for a monthly opt-in tier, and this ADR contradicts nothing in it.

**Where the two readings conflict, they conflict on one thing:** whether an opt-in owes the operator
a run sooner than the tier's cadence. This ADR says no.

### 5. The bound: a manual dispatch is a second act

An enabled cold `Scan` can be dispatched on demand (`internal/queue/queue.go:98`,
`cmd/web/scantrigger.go`). This ADR does not change that and does not refuse it.

What it refuses is the causal claim. The opt-in did not fire the sweep. The operator did, in a second
act, on a separate screen, against a control whose copy names what it runs. That is the condition
`#4` §6.4 exists to require: the person is present, and the run is asked for. ADR-0005 already rules
that such a run **dispatches an existing `Scan`, never an ad-hoc one-off**, so the manual path opens
no object this model cannot hold.

The deleted comment's *"the monthly cadence tick is the sole firing"* is therefore wrong, and this
ADR does not carry it forward.

### 6. What this rule does not reach

- **What the tier measures once it runs.** ADR-0047 rules the address-scope enumeration and ADR-0127
  rules that a large scope is priced rather than gated.
- **The rate the sweep runs at.** ADR-0137's safety budget bounds it, and it bounds a manual dispatch
  exactly as it bounds a cadence tick.
- **Opting out.** It is the same statement in reverse and needs no separate rule: the row goes, and
  `SyncColdScanEnabled` recomputes the flag.

## Consequences

- **No production behaviour changes.** The handler, the sync query and the fan-out already have this
  shape.
- **The survivor comment in `internal/queue/cold.go` asserts something false.** *"the monthly cadence
  tick is the sole firing"* does not hold while `Dispatcher.Trigger` fires an enabled scan. The
  correction and this ADR's citation are recorded in this issue's manifest rather than edited here.
- **`cmd/web/cold.go` gains this ADR's citation.** Its comment states the rule correctly and cites
  only the two sources that rule the neighbouring halves.
- **No defect in behaviour is created or found.** An operator who opts in a large scope and then
  dispatches manually gets a large sweep, which ADR-0127 rules is priced rather than gated and
  ADR-0137 rate-bounds. That is an operator act with existing bounds, not a hole.
- **A second opt-in tier is admitted by this rule** without a new decision, if one ever ships. A tier
  that must run at the moment of the opt-in is a different case and needs its own.
- **`CONTEXT.md` gains nothing.** `Scan` and `Seed` already carry their entries, and the act of
  enrolling a scope is a decision rather than a term.
- **This ADR is not a `wontfix` on ADR-0044 or the SPEC.** Both are confirmed and both gain reach.
  Neither is amended, so [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)
  requires no mark at either site.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Fire the fan-out on opt-in** | Puts an operator's first click straight onto a 1–65535 sweep of every address the scope admits. `#4` §6.4's *"Never scan on config save"* forbids the shape, and ADR-0044 already refuses the neighbouring act at `Seed` declaration on the same ground |
| **Enqueue a cold job on opt-in for the next dispatcher tick** — reading (a) of §6.4 | Mints a full-range measurement with no cadence and therefore no currency bound, which ADR-0044 rules is not expressible. It also keeps §6.4's own stated harm, because twenty opt-ins still land twenty sweeps on one tick, where the enrolment shape lands one |
| **Close the gap as `wontfix`, naming ADR-0044 and `v1-spec.md` §3.4** | Neither rules the act. ADR-0044 refuses firing at `Seed` **declaration**, which is a different act on a different object, and §3.4 rules the **unasked** case. The opt-in is the ask, and the handler cites both sources today while the act itself is unruled |
| **Amend ADR-0044 rather than write this** | Under ADR-0058 an amendment marks a superseded mechanism. Nothing in ADR-0044 is superseded: its rejected alternative stands verbatim and this ADR agrees with it. A rule ADR-0044 never stated is a new decision, not a correction to an old one |
| **Run a one-off sweep of the newly enrolled scope alone, then fall onto the cadence** | ADR-0044's whole finding, restated at a smaller scope: no cadence, no currency bound, and spans the currency machinery cannot age. It also breaks the aperture statement's constancy that #44 decision 10 rests on |
| **Keep the deleted comment's wording — "the monthly cadence tick is the sole firing"** | False. `Dispatcher.Trigger` (`internal/queue/queue.go:98`) fires the cold fan-out for an enabled `Scan`, and the opt-in is precisely what enables it. A rule stated on a false premise fails the first time a reader checks it |
| **Refuse a manual dispatch of the cold tier as well** | It would overturn ADR-0005's manual run for one tier, and it would remove the one path on which the operator really is present and watching — which is the condition §6.4 asks for, rather than the one it forbids |
