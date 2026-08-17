# ADR-0108: A batch whose instrument could not reach its position covers nothing, and the failure is the vantage's, not the subject's

- **Status:** Accepted
- **Date:** 2026-08-16
- **Ticket:** [#249 Make an unreachable vantage resolver loud instead of a silent all-Gap batch](https://github.com/winniel123/verge-asm/issues/249) · [#244 Backend failures are not surfaced in the UI](https://github.com/winniel123/verge-asm/issues/244)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

A `local` vantage whose recursive resolver pointed at nothing produced a `Batch` with
`outcome=completed`, a full recorded scope, and eight rows of `{"rrs": null}` /
`resolution -> {"outcome":"Gap"}`. From every surface the run read as a completed measurement of an
estate that answered nothing — *we looked and there is nothing there* — when the truth was *the
resolver we were told to use could not be reached*, which is *we could not look*. That is the exact
false-reassurance the whole `Coverage`/`Gap` design exists to prevent
([ADR-0036](./0036-a-shipped-default-is-the-configuration-that-takes-effect.md)'s ethos, re-entering
through the measurement path), and [#244](https://github.com/winniel123/verge-asm/issues/244) records
that the same class of defect — a backend failure indistinguishable from an empty result — recurs on
every failure path the product has: the proposer lookup (fixed at
[#251](https://github.com/winniel123/verge-asm/issues/251)), the probe, scan dispatch, and delivery.

Two things were already true in the model and false in the code.

- **`Batch`:** *"The recorded scope is what the batch **completed**, never what it attempted, so a
  batch that failed outright covers nothing and licenses no absence."* The resolver-unreachable batch
  recorded its **attempted** scope and licensed eight absences it never measured.
- **`Availability`:** *"A vantage that has failed every attempt across the window is `unavailable`,
  which opens a `Gap` on the `Reach` of its class."* Nothing in the measurement path derived
  `Availability` from batch outcomes; `MarkVantageUnavailable` existed and was called from nowhere.

So the corpus already names the correct behaviour twice and the running system honoured neither. This
ADR is what closes the gap between the two, and it decides the one thing that was genuinely
undecided: **how *we could not look* is told apart from *we looked and there is nothing there*,
without a heuristic.**

## Decision

**A measurement that could not reach its position is a failed measurement, not an empty successful
one. It covers nothing, it registers as the vantage's unavailability rather than the subject's
absence, and the same rule governs every backend failure path: a failure is legible as a failure and
never renders as a clean empty result.**

Six limbs.

1. **The signal is a transport failure, never a null count.** A resolver is unreachable when the
   exchange to it fails at the socket — a dial or read error against the **declared-path** recursive
   resolver. It is **not** inferred from *every name came back empty*: a total-null batch is a
   heuristic and the ticket is right to refuse it, because an estate that genuinely resolves nothing
   produces the same nulls. `NXDOMAIN`, an empty `NOERROR`, `REFUSED` and `SERVFAIL` are **answers**
   — the resolver was reached and spoke — and stay values. Only *no answer came off the wire at all*
   is *we could not look*.

2. **The failure is batch-scoped, because the resolver is one position for the whole batch.** A
   `Vantage`'s recursive resolver is part of its identity
   ([ADR-0070](./0070-a-control-probe-is-asked-from-where-the-answer-it-discriminates-was-asked-from.md));
   one fixed address answers every name in the batch or none. So the **first** declared-path
   transport failure aborts the batch — there is no honest partial scope to record, and the model
   forbids recording the attempted one. The delegation **walk** is exempt: it dials delegated
   authorities direct, whose per-authority silence is already the `Gap`/`Lame` vocabulary
   ([`leaf.go`'s `walk`](../../internal/measure/resolutionwalk/leaf.go)) and never a statement about
   our own position.

3. **An aborted batch routes through the existing retry → dead-letter path, and covers nothing.** The
   failure surfaces as a prober error, so the worker's retry machinery absorbs a transient blip (five
   attempts across a back-off window) and a **persistently** unreachable resolver dead-letters. A
   dead-lettered `Batch` records the **empty** scope `{"names":[]}` and writes **no** observations —
   the all-`Gap` rows are never committed. This is not new machinery; it is the batch finally taking
   the path `Batch` always specified for a failure.

4. **A terminal `dns` (resolution-walk) batch outcome derives the vantage's `Availability`.** A
   **completed** one marks its vantage `available`; a **dead-lettered** one marks it `unavailable`.
   The window `Availability` is *"concluded from recent batch outcomes over"* is, in v1, the **retry
   sequence within one dispatch**: a dead-letter already means every attempt across that window
   failed, and a single completed batch is proof the position observes again, so recovery is immediate
   on the next success rather than pending a multi-cadence count. Two scopings keep the derivation
   honest, because **`Availability` is a single scalar per vantage** and a vantage runs several `Scan`
   kinds:
   - It runs only where the batch carries a real `Vantage` — the worker-read `zone` and `ct` `Scan`s
     have none ([`batch.vantage_id`](../../db/migrations/18803_measurement_batch.sql) is nullable) and
     never move it.
   - It runs only for the **`resolution-walk`** batch — the one `Scan` that exercises the vantage's
     recursive resolver, and the only capability a resolver outage impairs. A completing **port
     probe** (`hot`/`cold`/`tls-acceptance`, whose batches carry `connect-outcome` / `tls-acceptance`)
     at the same vantage says nothing about resolver health, so it must **not** re-mark the vantage
     `available` and clobber the `unavailable` a dead-lettered `dns` batch set. Every vantage in
     `ListVantagesForDispatch` receives all of these kinds, so without this scoping a single
     port-probe completion would silently re-mask the resolver outage — the exact regression #249
     removes. This is a deliberate coarseness of the single scalar: a **per-capability** availability
     (resolution vs reachability) is a larger model change carried as future work, and v1 scopes the
     scalar's one writer to the resolver capability rather than sharing it across kinds.

5. **The operator-visible signal is the `Gap` on `Reach` the model already routes.** An `unavailable`
   vantage is excluded from its class's presence
   ([`cmd/web/exposure.go`](../../cmd/web/exposure.go)), so `Exposure` that would need it is **absent
   rather than quietly computed** from the class that still answers — a one-legged reading rendered
   under *we never looked* / a `Gap` under *we stopped looking*
   ([ADR-0017](./0017-exposure-needs-both-legs.md),
   [ADR-0095](./0095-the-aperture-statement-counts-what-the-instrument-cannot-report-not-what-it-did-not-look-at.md)).
   `Coverage` surfaces the unavailable vantage directly, so the condition is legible **without an
   operator reading `observation.value` by hand** — #249's second acceptance criterion. No new
   coverage-class member and no new `Gap` register is minted: a resolver our instrument could not
   reach is *we stopped looking* — **our** failure, keyed on what moved
   ([ADR-0064](./0064-a-message-names-what-moved-and-where-nothing-moved-it-says-so.md)) — and it
   already has a register.

6. **The rule generalises to every backend failure path (#244), each on its own surface.** *A failure
   renders as a failure, never as a clean empty result* is not a resolver rule; it is the aperture
   ethos applied to runtime failures. It is discharged per path — and, crucially, on the surface the
   model already assigns that path, never uniformly on `Coverage`:
   - **Proposer** (done, #251): the lookup carries its `perr` to the operator, so a registry outage
     does not read as *your name matched nothing*.
   - **Probe / measurement**: a failed probe batch takes the same path as the resolver case — it
     dead-letters (a durable `Batch` with the empty scope) and marks its `Vantage` unavailable, which
     `Coverage` surfaces. Probe batches carry a vantage, so limbs 3–5 cover them with no extra
     machinery.
   - **Delivery**: a failed POST was **already** recorded durably — a `Delivery` in state
     `undelivered` carrying its `last_error` — but was rendered nowhere. It is now surfaced on the
     **Message it failed to carry** (ADR-0039/ADR-0081's designated surface), never on `Coverage`,
     which a delivery has no cause to touch: an undelivered message reads as *could not deliver*, not
     as *nothing fired*, with the reason as a drill-down (#22).
   - **Scan dispatch**: a fan-out failure is **not** the indistinguishable-from-empty hazard this ADR
     closes — it produces no batch, so its effect is a **missed cadence**, which the currency
     machinery already makes legible as staleness ripening into a `Gap`
     ([ADR-0084](./0084-a-scan-is-a-cadence-over-an-exchange-and-an-uncovered-facet-has-no-currency-bound.md)).
     It stays an operational log line: a durable dispatch-error store is a new subsystem carrying a
     failure the model already tells honestly, and is deferred rather than built here.

## Rationale

### The transport failure is proof; the null count is a guess

The ticket's sharpest constraint is that *a total-null batch is a heuristic, not proof.* It is right,
and limb 1 is built to honour it. The distinction the model already draws — between a **value** and
*we could not look* — maps exactly onto the wire: a DNS response, of **any** rcode, means the
resolver was reached and answered, and `NXDOMAIN`/`NoData`/`REFUSED`/`SERVFAIL` are the values that
carries; **no response off the socket** means we could not look. `decideResolution` was already
faithful to this for a single name — *reached but no usable answer is a `Gap`, never a withdrawal* —
and the only thing missing was that a name for which the **resolver itself** never answered was
folded into the same per-name `Gap` as a name the resolver answered emptily about. Limb 1 separates
them at the one place the difference is observable: the socket. This claims nothing about the
listener — it is named for **the exchange we made**, the construction `CONTEXT.md` requires of every
negative — and it needs no count of how many names came back null.

### A failed batch must cover nothing, and the machinery for that already exists

`Batch` is unambiguous: a batch that failed outright *"covers nothing and licenses no absence,"* and
a dead-lettered batch already records the empty scope `{"names":[]}` *"because asserting the
attempted scope would manufacture absences it never measured."* The defect was never that the model
lacked a failure path; it was that a resolver-unreachable batch never **entered** it — it returned no
error, so the worker read `probeErr == nil` and called `complete()`. Limb 3 routes it into the path
`Batch` always specified, and buys the retry/dead-letter semantics — transient-absorbing,
scope-empty — for free. Recording a **partial** scope for a batch that lost its resolver midway was
refused: a mid-batch resolver loss is not a clean partition of the scope, and `Batch` admits a
partition only *"along any dimension its source still retains completeness over"* — which a blinded
resolver retains over nothing.

### Deriving `Availability` from the outcome is the mechanism the model already wired its consumers to

`Availability` is *"concluded from its recent batch outcomes rather than measured directly,"* and its
consumer — `Exposure`'s class-presence gate — was already written against it: `exposure.go` skips any
vantage that is not `available`, and `ComposeReach` already handles the empty-in-scope set as
`not-evaluable`. The producing half was simply absent. Limb 4 supplies it as the thinnest possible
derivation — the latest terminal outcome — rather than a windowed count, and the choice is the
ticket's own safety direction: a false `unavailable` **opens** a `Gap` and is investigated, while a
false `available` **hides** one, so the responsive rule fails in the loud direction, exactly as
`Vantage class` verifies to `internet` on doubt because *"a false quiet reading is not investigated."*

**On the host-key path.** `MarkVantageUnavailable` is also the trust-on-first-use response to a
pinned host key later mismatching ([ADR-0103](./0103-a-vantage-is-one-position-and-the-prober-is-optional-provisioning-detail.md)),
and the two writers share the one scalar. Today they do not collide: the TOFU mismatch path is not
yet wired (SSH host-key enforcement is deferred, `cmd/worker/vantages.go`), and *dead-lettered →
unavailable* agrees with it rather than fighting it. The `local` resolver-only vantage has no host key
and no prober, and its availability is driven entirely and correctly by its `dns` batch outcomes.
**Forward caveat:** when SSH enforcement lands, *completed → available* clears a security pin only if a
host-key-mismatched **remote** prober cannot complete a `resolution-walk` batch — i.e. only if that
vantage's `dns` measurement is itself gated behind the SSH connection rather than run from the worker
against the resolver address directly. That invariant is the successor ticket's to hold; this ADR
records the dependency rather than assuming it.

### Why one ADR for #249 and #244

[#244](https://github.com/winniel123/verge-asm/issues/244)'s triage kept it as the systemic ticket
and asked that the design *"make 'we could not measure this' a first-class Coverage state, distinct
from a normal empty result,"* coordinating with #249. Walked against the model, that state is **not**
a new coverage-class member to be minted — it is `Availability`, finally produced, rendering through
the `Gap`-on-`Reach` and the *we stopped looking* register the corpus already holds. #249 is the
worked exemplar of the class; this ADR states the class rule (limb 6) and the exemplar's mechanism
(limbs 1–5) together, because inventing a parallel "could not measure" object beside `Gap` and
`Availability` would be the record-with-optional-fields mistake
([ADR-0011](./0011-a-facet-is-six-parts.md)) one layer up — a second way to say *we could not look*,
indistinguishable at a glance from the ones the model already has.

## Consequences

- **[`CONTEXT.md`](../../CONTEXT.md) `Availability` entry** gains the sentence that its `unavailable`
  state is now **derived from terminal batch outcomes** — a completed batch restores it, a
  dead-lettered batch opens it — where before the derivation was specified and unbuilt.
- **The resolution leaf** (`internal/measure/resolutionwalk`) gains an explicit *unreachable* signal
  on `Msg`, set by `NetPeer` on a declared-path socket error and propagated by `RunWithPeer` as a
  batch-aborting error. No emitted observation changes shape and no golden corpus moves — the signal
  is a control field the scripted corpus peers never set, so every existing fold is byte-identical.
- **The worker** derives `Availability` on every terminal outcome of a vantage-bearing batch. The
  decision is a pure function (`availabilityAfterOutcome`) unit-tested apart from the transaction.
- **No coverage-class member, `Gap` register, or `Exposure` value is minted.** The class stands where
  it stood; a resolver-unreachable `Gap` renders under *we stopped looking*.
- **#244's remaining paths** are addressed under limb 6, each on its own surface: the **probe** path
  rides limbs 3–5 (dead-letter + unavailability + `Coverage`); **delivery** failures — already durable
  — are now rendered on their `Message`. The **scan-dispatch** fan-out error is left as an operational
  log with its rationale recorded (its failure is a missed cadence the currency machinery already
  tells, not an empty-success masquerade); a durable dispatch-error surface is carried as a follow-up.
  The proposer path was discharged at #251.
- **A dead-lettered dns batch now marks its vantage unavailable**, which can flip `Exposure` from a
  computed reading to an absent one the first cadence a resolver goes down. That is the intended,
  loud behaviour: a leg that cannot be measured is absent, not quietly carried by the other class.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Infer unreachability from an all-`Gap` / all-null batch** *(the shape the bug report describes)* | The heuristic the ticket refuses by name. An estate that genuinely resolves nothing produces the same nulls, so this collapses *we could not look* back into *there is nothing there* — the defect, restated as its own fix. Limb 1 keys on the socket instead |
| Mint a new first-class *could-not-measure* `Coverage` state beside `Gap` | ADR-0011's record-with-optional-fields mistake one layer up: a second object meaning *we could not look*, indistinguishable from `Availability`/`Gap`, which the model already carries. #244's ask is satisfied by producing `Availability`, not by inventing its rival |
| Give a resolver-unreachable batch a new `Batch` outcome (`failed`, `vantage-unavailable`) | Unnecessary. `dead-lettered` already means *failed after every retry* and already records the empty scope. A third terminal outcome would split the failure population without a reader for the split |
| Record a **partial** scope for a batch that lost its resolver midway | A mid-batch loss is not a partition the source retains completeness over, so recording it licenses absences for names measured under a resolver that was already gone. `Batch` admits partition only where completeness survives it |
| Mark unavailable only after **N** consecutive dead-lettered batches | Faithful to a literal reading of *window*, but with daily batches it buys days of a silent-ish degraded state — the anti-silence goal inverted. A dead-letter is already *failed every attempt across the retry window*; the responsive rule fails loud and self-heals on the next success |
| Leave the failure on the log line, as today | The `#244` defect exactly: a failure legible only to someone reading the server log is indistinguishable, on every operator surface, from a clean empty result |
