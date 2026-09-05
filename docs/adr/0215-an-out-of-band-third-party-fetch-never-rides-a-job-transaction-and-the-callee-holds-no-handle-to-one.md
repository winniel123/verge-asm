# ADR-0215: an out-of-band third-party fetch never rides a job transaction, and the callee holds no handle to one

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1323 ADR gaps: internal/queue (queue, cttail, withdrawal, hot, ctverify, scopegate)](https://github.com/winniel123/verge-asm/issues/1323), gap 4
- **PR that deleted the comment:** [#1322](https://github.com/winniel123/verge-asm/pull/1322)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Rests on:** [ADR-0007](./0007-drift-is-a-timeline-of-spans.md) and [`v1-spec.md`](../spec/v1-spec.md) §2.4, which require the outcome, its observations and its raw output to commit together. That is the transaction this rule protects. Neither says what may not be inside it
- **Sibling of, and not ruled by:** [ADR-0149](./0149-a-consumer-takes-the-data-layer-interface-it-calls-and-the-seam-not-the-package-is-the-unit.md). That ADR rules how wide a consumer's data-layer interface is. This one rules whether a function holds a transaction handle at all. Both are answered by a signature, and neither contains the other
- **Sibling of, and not ruled by:** [ADR-0199](./0199-delivery-imports-queue-never-the-reverse-so-the-worker-takes-the-message-enqueuer-as-an-injected-function-that-joins-the-batch-transaction.md). It rules that the injected message enqueuer **joins** the batch transaction. That is the correct answer for a local database write, and it is the contrast this rule needs: an injected function is not exempt from the transaction, and what decides the side is whether the work reaches a host the operator does not own
- **Sibling of, and not ruled by:** [ADR-0196](./0196-no-outbound-http-client-this-product-builds-follows-a-redirect-and-an-unfollowed-3xx-admits-nothing.md). It rules what an outbound client may do on the wire. This rules where in the control flow it may be called. Both bind every outbound client this product builds

## Context

[`internal/queue/ctverify.go:103`](../../internal/queue/ctverify.go) carried this text, until #1322
deleted it:

```go
// It runs AFTER the terminal tx (its caller commits first), so its log fetches never ride a database transaction
```

The sweep left a compressed line at `ctverify.go:68` and one at
[`internal/queue/worker.go:459`](../../internal/queue/worker.go). The second cites
`ct-source-replacement.md` §5.4, which rules the verification trigger and states nothing about a
transaction. No ADR and no SPEC states the ordering. That is #1323's gap 4.

**The ordering is one line apart, and nothing in either signature announces it.**

```go
	}); err != nil {
		return err
	}
	// A CT verification does network I/O, which must never ride a database transaction (spec §5.4).
	w.autoVerifyCerts(ctx, job, obs)
	return nil
```

The closing brace is `runJobTx`'s. `runJobTx` calls `inTx`, which begins, runs the closure, and
commits (`worker.go:517`). So the call on the next line runs against a committed database.

**The measurement is the size of what the transaction would have held open.** One call to
`autoVerifyCerts` verifies up to `maxAutoVerifyPerJob` certificates, which is **8**
(`ctverify.go:19`). Each certificate's `verifyMaterial` walks every SCT it carries — embedded,
TLS-extension and OCSP-stapled — and each usable SCT costs `checkOneLog` **two** third-party
fetches: a head and then a proof or a tile. The shared `HTTPCTFetcher` client sets
`Timeout: 90 * time.Second` (`crtsh.go:50`), sized for crt.sh's measured 59.6-second answers. A
certificate commonly carries two or three SCTs.

So the upper bound on one terminal path is tens of requests to hosts the operator does not own, each
bounded at 90 seconds and by nothing else. `w.probeTimeout` does not reach it: `worker.go:311`
records that the timeout bracket wraps the probe alone, so the terminal path runs under the parent
context.

The transaction it would have held open writes a `Batch`, every observation, the certificate
material, four membership folds, the message production and the transcript, and it ends with
`markDone` on the job row. Those rows are the ones a second worker's `ClaimJob` and every reader
contend for.

**The rest of `internal/queue` already keeps the same discipline from the other side.** `probe`,
`completeCT` and `completeCTTail` each finish their third-party fetching before they open a
transaction. `autoVerifyCerts` is the one path that runs on the far side of one, and it is the only
one where the ordering is a caller's obligation rather than an obvious consequence of the control
flow.

## Decision

> **A fetch to a host the operator does not own never runs inside a job transaction. The transaction
> opens after the last such fetch, or the fetch runs after the transaction commits. The obligation
> is the caller's, and it is made legible in the callee's signature: an out-of-band fetch takes no
> `*db.Queries` and returns no error.**

Four limbs.

### 1. The bound on a third-party fetch is the third party's, so it may not bound a transaction

A job transaction holds row locks on the `batch`, `span`, `observation` and `job` rows a second
worker contends for. Its duration must be a property of this instance. A CT log's latency is not.

The rule is about the class, not about CT. Any host outside the operator's control has the same
property: a report delivery endpoint, a release feed, a proposer's API. None of them may sit inside
a transaction, for one reason that does not depend on which one it is.

### 2. The seam is the signature, and the callee holds no transaction handle

`autoVerifyCerts(ctx context.Context, job db.ClaimJobRow, obs []wire.Observation)` takes no
`*db.Queries`. Its one write goes through `emitVerifyEvent` to `emitProgress(ctx, w.q, ...)`, and
`w.q` is the pool-backed handle rather than a transaction's. The function could not join the
terminal transaction if a later edit asked it to, because it holds nothing to join.

This is how the rule is enforced, and it is the only enforcement there is. No runtime check fires on
a fetch inside a transaction, and a check that could is not available: `pgx` does not report to the
HTTP client that a transaction is open. A signature that cannot express the wrong thing is the
mechanism, in the same shape
[ADR-0149](./0149-a-consumer-takes-the-data-layer-interface-it-calls-and-the-seam-not-the-package-is-the-unit.md)
uses for reach.

**A helper that threads an open transaction is the opposite case and stays as it is.** `foldOne`,
`closeSpansByID` and their siblings take `qtx *db.Queries` because they run inside the transaction
on purpose. The signature states which side of the boundary a function is on, and that is the
property this rule buys.

### 3. Verification runs after the commit, because the measurement is the obligation

The ordering could have been the other way. `autoVerifyCerts` reads `obs`, which exists before the
transaction opens.

It runs after, so that the durable record of a measurement already taken never waits on a third
party. A verification that ran first would delay the commit by the §Context bound, and a context
cancellation or a crash during it would discard a `Batch` the prober had already paid for.

`autoVerifyCerts` returns no error, and that is the same decision stated in the signature. It cannot
fail the job. A failure to verify leaves the job completed and the estate correct, which is the
right direction: verification mints no subject and stores no durable result
([`ct-source-replacement.md`](../spec/ct-source-replacement.md) §5.2).

### 4. The accepted costs of running after the commit

Two, and both are named rather than mitigated.

- **A process that dies between the commit and the verification simply does not verify.** No retry
  reaches it, because the job is already `done`. The `Batch` is durable and the estate is correct.
  An unverified certificate is the ordinary steady state for every certificate observed before the
  feature was wired.
- **The verification events arrive after the job's completion event.** `emitProgress` writes through
  `NotifyJobProgress` outside any transaction, so a `CT verification ·` line reaches the live stream
  after the job has already reported `done`. The stream is ordered by arrival and persists nothing
  ([`raw-job-output.md`](../spec/raw-job-output.md) §6.2), so this costs an operator no fact.

## Consequences

- **This ADR changes no Go code.** The one caller already commits first.
- **The rule now has a document to hold a new path to.** A second auto-enrichment on the terminal
  path — a WHOIS read, a reputation lookup, a second log family — is the shape that would land
  inside the transaction by accident, because everything else on that path is written as a `qtx`
  closure.
- **Nothing enforces the ordering at run time, and nothing will.** §2's signature rule is what a
  reviewer checks. A caller that moved `w.autoVerifyCerts(ctx, job, obs)` above the closing brace
  would compile, pass every test, and hold a transaction open for tens of minutes only under a slow
  third party.
- **`internal/queue/worker.go:459` carries a wrong citation.** It cites
  `ct-source-replacement.md` §5.4, which rules the verification trigger and the result and states
  nothing about a transaction. The replacement is in this issue's manifest.
- **[`v1-spec.md`](../spec/v1-spec.md) §2.4 gains nothing.** It requires the outcome, the
  observations and the raw output to commit together. That is still true, and this rule is about
  what may not join them.
- **[`CONTEXT.md`](../../CONTEXT.md) gains nothing.** A transaction boundary is a code-structure
  term, not a domain term.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Run the verification inside the terminal transaction** | Holds locks on the `batch`, `span` and `job` rows for the duration of up to tens of third-party requests, each bounded only by a 90-second client timeout. A second worker's `ClaimJob` contends for the same rows. It also lets a CT log's outage discard a measurement the prober already paid for |
| **Run the verification before the transaction opens** | Removes the lock hazard and keeps the latency one. The commit of a measurement already taken would wait on a third party, and a cancellation during the wait loses the `Batch`. §3's ordering costs an unverified certificate on a crash, which is the ordinary state of every certificate anyway |
| **Pass `qtx` to `autoVerifyCerts` so its events join the batch's commit** | Puts a transaction handle in the one function that must not hold one, and deletes §2's whole enforcement. It buys atomicity for an ephemeral progress event that persists nowhere at rest |
| **Assert at run time that no transaction is open before a fetch** | Not expressible. The HTTP client cannot see the pool's state, and `pgx` reports no ambient transaction. A `Worker`-level flag set by `inTx` would be a second source of truth for a fact the signature already carries, and it would fire only in production |
| **State the rule as a comment at every fetch site** | The comment policy admits it at one site at most, and #1322 already cut it to one line. The rule binds `internal/delivery`, `internal/release` and `internal/report` as well as this path, and a per-site comment states it nowhere a new path's author would look |
| **Rule it inside [ADR-0007](./0007-drift-is-a-timeline-of-spans.md) or `v1-spec.md` §2.4** | Those fix what must commit together. This rule fixes what must not be inside the same transaction, which is a different question about the same boundary, and it binds packages that write no span |
