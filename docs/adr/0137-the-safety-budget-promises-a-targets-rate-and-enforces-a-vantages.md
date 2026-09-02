# ADR-0137: The safety budget promises a target's rate and enforces a vantage's, and the gap is disclosed

- **Status:** Accepted
- **Date:** 2026-09-02
- **Ticket:** [#1107 Move the active-scan safety budget into Postgres so it survives scaling the worker](https://github.com/winniel123/verge-asm/issues/1107)
- **Follows:** [#1092](https://github.com/winniel123/verge-asm/issues/1092), which found that the limiter is per prober process, and [#1105](https://github.com/winniel123/verge-asm/issues/1105), which recorded the gap in prose without closing it
- **Constrained by:** [ADR-0005](./0005-scan-execution-model.md) (one address per batch, and the intra-job coordination claim this ADR corrects), [ADR-0127](./0127-the-address-scope-range-cap-has-no-ceiling-a-large-scope-is-priced-not-gated.md) (a cost the operator chose is priced, not gated)
- **Relates to:** [#1106](https://github.com/winniel123/verge-asm/issues/1106), which holds the ADR-0005 amendment this ADR supplies the corrected unit for

## Context

`SafetyProfile` declares 50 conn/s and 20 in-flight connections per host, and 200 pkt/s across
targets. The numbers are declared parameters of the `connect-outcome` leaf, under a golden-corpus
lock, and every `Batch` records them by content. They are the project's undertaking about what it
does to a network it is pointed at.

[#1092](https://github.com/winniel123/verge-asm/issues/1092) found that the limiter providing them
runs inside one prober process. `NewPacer` is built in the leaf's `Run` entrypoint, and the worker
execs a fresh prober per job, so one pacer exists per job with no cross-process state.
[#1105](https://github.com/winniel123/verge-asm/issues/1105) recorded that honestly in the comments
and the running guide, and forbade `--scale worker=N`. It did not restore the property, and it left
`GlobalPacketsPerSec` documented as a ceiling that never binds: one address per batch means the
20 ms per-host interval is always later than the 5 ms global one.

#1107 asked whether the budget should move into Postgres on the `ct_throttle` pattern
([ADR-0106](./0106-the-ct-poll-is-a-scan-that-schedules-and-a-ct-admission-is-a-name-citing-its-batch.md)),
and named one question as load-bearing: **whether the prober can reach Postgres at all.**

The code answers it, and the answer is no. `remoteexec.Probe` pushes the prober binary over SSH to
the remote vantage host and runs it there. The job spec travels on stdin and NDJSON comes back on
stdout. A prober on a remote vantage has no DSN, no pool, and no route to the database. A
reservation inside the prober is therefore not implementable for the vantage the project exists to
support.

That forced the question underneath, which had never been asked: **whose rate is the ceiling a
ceiling on?** "Per host" does not say which side of the wire it means, and the three issues above
each assumed a different answer without noticing.

## Decision

### 1. The budget **promises a target's rate**

The numbers derive from what a destination tolerates — its SYN backlog, its connection table, its
intrusion detection. None of those care how many sources the traffic arrived from. So the honest
statement of what the budget is *for* is the rate **one target receives**, summed over every vantage
and every process we run.

This is the strict reading, and it is the one the running guide already implies when it calls the
budget an undertaking about a network we are pointed at.

### 2. It **enforces a vantage's rate**, and the gap is **disclosed, not closed**

The promise in §1 cannot be enforced without dividing every vantage's rate by the vantage count.
That is refused. Comparing what different vantages see is the reason to declare more than one, and a
scheme that slows every vantage as vantages are added makes the capability defeat itself. It is also
the failure mode #1092 rejected in option 3: a share held by a participant that is no longer there.

So enforcement is **per `Vantage`**: one vantage emits at most the declared rate at one target. The
consequence is stated rather than smoothed — **a target inside N declared vantages receives up to N
times the declared rate**, and nothing prevents that.

Nothing gates the vantage count either. That is ADR-0127's rule applied one axis over: a cost the
operator deliberately declared is priced at policy time and not refused by a threshold with no owner.
The disclosure lives in the `Safety budget` glossary term, in the running guide, and here.

The `Batch` record needs no new field. After §5 it records a **per-vantage** figure that is true per
vantage, and the target-inbound total is the vantage count times that figure — derivable by any
reader who has both. A computed multiplier on the `Batch` would be a derived value with no reader,
which ADR-0005 refuses.

### 3. The per-target ceiling needs **no cross-process coordination**, and never did

ADR-0005 argued that the per-host limit is *intra-job and needs no coordination*, because one
address's ports are never split across concurrent jobs. The conclusion is right. **The unit is
wrong.** `fanOutHot` emits one job per `(Vantage, Address)` pair, so a host inside N declared
vantages sits in N jobs, and the limit is not intra-job.

The correct unit is **intra-`(Vantage, Address)`-pair**. Under §2's per-vantage enforcement that is
exactly the unit the ceiling is stated over, so the property holds: two concurrent jobs of one
`Dispatch` never share a target from one position, at **any** worker count.

This is the ADR's most consequential finding, and it is a subtraction. **The per-host hazard #1092
described does not require a Postgres reservation, a grant, or any coordination at all.** It is
carried by the fan-out shape. `--scale worker=N` was never the thing that broke it.

The amendment recording this in ADR-0005 belongs to
[#1106](https://github.com/winniel123/verge-asm/issues/1106), which already holds that decision.
This ADR supplies the corrected unit and does not make the edit.

### 4. **Cadence lag** is the one hole, and a **dispatch gate** closes it

§3 holds *within one `Dispatch`*. Dispatch idempotency is `(scan, scheduled_time)` under a per-scan
advisory lock, and nothing gates a new tick on the previous dispatch draining. Where a `hot` scan
does not finish inside its cadence — the raised-cap case ADR-0127 priced — the next tick enqueues a
duplicate job for every pair while the first dispatch's are still pending. Two workers can then run
one `(Vantage, Address)` pair concurrently, and the per-target ceiling doubles.

So the fix is a **lag gate on the `hot` dispatch**: while the previous dispatch holds non-terminal
jobs (`ready` or `running`), the tick is **skipped and recorded**, never deferred. A deferred tick
would carry a `scheduled_time` it did not run at, which is the wrong-record failure #1092 raised. A
recorded skip reuses the vocabulary ADR-0005 already has for an overlapping tick, and it makes
cadence lag **visible** to the operator — the signal that the cap they raised has a cost.

Two bounds on it:

- **`hot` only.** `cold` and `edgefanout` also fan out streamed, and lag is a general property of
  all three. But only `hot` connects to a target, and this gate exists for a safety property.
  Extending it would smuggle a scheduling change into a safety change.
- **It refuses to arm when the stale-`running` reaper is disabled** (`VERGE_STALE_JOB_TIMEOUT <= 0`).
  With the reaper off, a job wedged in `running` never terminates, and the gate would then skip every
  future `hot` tick on one stuck row — converting a tuning choice into a silent stop of all active
  measurement. It logs and falls through instead. A safety mechanism that can permanently stop
  measurement is worse than the doubling it prevents, and its failure would surface only as an
  absence.

### 5. `GlobalPacketsPerSec` is **per-vantage**, is renamed, and takes the `Version` bump

Under §2 the aggregate ceiling governs one vantage, not the estate. The field's name is wrong twice
over: it never bound at all under one-address-per-batch (#1105), and its scope was never global.

It is renamed in the Go field **and in the JSON tag**. `SafetyProfile.Digest()` hashes
`json.Marshal`, so the tag change moves the digest and forces `Version` off `connect-outcome/v1`.
That `Break` is accepted.

Renaming only the Go field would hold the digest stable and leave a wrong name on the wire and on
every recorded `Batch`. That trade is refused: it preserves the defect in the durable artifact in
order to protect a corpus, and #1092's sharpest objection was precisely that the record would be
wrong rather than merely incomplete.

### 6. The aggregate ceiling binds only by a **grant**, and the grant is **falsifiable**

§2's per-vantage enforcement is not free at N workers. Nothing serialises probers per vantage host:
`remoteexec.Probe` pushes a fresh binary per invocation, so N workers can run N concurrent probers on
one remote vantage, each pacing to the full budget.

The mechanism that would bind it is **not** a reservation — §1's answer rules that out — but a
**grant**. The worker already sends the prober a `JobSpec` on stdin, over the same SSH channel. It
can carry a budget grant computed by the worker from a Postgres reservation at claim time: one round
trip **per job**, never on the connect path, and expiring with `probeTimeout`, which answers #1092's
objection to a share held by a departed holder.

The grant is **not built by this ADR**, and it is deliberately falsifiable. It is blocked on a
wall-clock measurement of a `hot` scan at the default address-scope cap. **If that scan finishes
comfortably inside its cadence at one worker, nobody scales, no second prober runs on a vantage, and
the grant guards a case that does not occur** — at which point the honest outcome is to keep the
per-vantage scope in prose and close the grant unbuilt.

## Consequences

- **`Safety budget` enters the glossary.** The difference between what the budget promises and what
  it enforces is load-bearing after this ADR and had no home before it. Four issues each re-derived
  the scope in prose because no term fixed it.
- **The per-host half of #1107 is subtracted, not built.** §3 shows the property already holds at any
  worker count. The reservation #1107 proposed would have been the most expensive option for the
  least marginal safety.
- **ADR-0005's conclusion survives and its argument is narrowed.** #1106 makes that edit.
- **`--scale worker=N` is still forbidden** until §6's grant lands or is closed unbuilt. §4's gate
  fixes the duplicate-pair hazard, which is a one-worker defect; it does not make scaling safe on its
  own.
- **A `Version` bump to `connect-outcome/v2` is owed**, with a re-bless of
  `internal/measure/connectoutcome/corpus/corpus.lock.json` and the two tests asserting the string
  (`internal/drift/reachability_test.go`, `internal/signal/service_test.go`). `certcorpus` does not
  carry it.
- **Cadence lag becomes visible.** After §4 an operator who raised the cap past what one worker
  drains sees recorded skips rather than silent rate doubling.
- **Nothing here is measured.** The rate at N workers has never been captured, before or after, and
  #1092 said so first. §3's subtraction and §4's hole are both read off the fan-out and the dispatch
  path, not off a packet capture. §6's measurement is the first one owed.

## Alternatives rejected

**Reserve packet budget in Postgres from the prober, on the `ct_throttle` shape.** Rejected on
capability, not on cost. A prober on a remote vantage runs over SSH with no database route, so the
reservation cannot exist where the packets are emitted. The cost objection #1092 raised — a round
trip on a connect path defending a 20 ms interval — is real but secondary; the mechanism fails before
it is reached.

**Divide the budget by a live worker count.** Rejected, as in #1092. It needs a worker registry the
repo does not have, and it is wrong during the window where a departed worker still holds a share.
§6's grant is the shape that survives, because a per-job lease expires on its own.

**Halve each vantage's rate so the target-inbound promise is enforced exactly.** Rejected. It makes
declaring a second vantage slow the first, so the capability degrades the more it is used, and
comparing vantages is the whole reason to have more than one. §2 discloses the multiplier instead.

**Gate or cap the vantage count.** Rejected on ADR-0127's ratified reasoning: a threshold in the
safety path with no owner and no derivation behind its value, refusing a declaration the operator
deliberately made.

**Defer a lagging `hot` tick rather than skip it.** Rejected. A deferred tick runs at an instant its
`scheduled_time` denies, and the record is then wrong in exactly the way #1092 objected to. A skip is
already the corpus's word for a window that did not run.

**Rename the Go field only, holding the JSON tag to protect the corpus lock.** Rejected. It keeps a
name known to be wrong on every `Batch` we will ever record, to avoid a version bump that costs one
lock file and two test assertions.
