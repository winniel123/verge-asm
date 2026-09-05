# ADR-0202: a Vantage's resolver comes from its row alone, and the code-side fallback is removed

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1319 ADR gaps: internal/scan (2/3)](https://github.com/winniel123/verge-asm/issues/1319), gap 3
- **PR that deleted the comment:** [#1318](https://github.com/winniel123/verge-asm/pull/1318)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Rests on:** [ADR-0070](./0070-a-control-probe-is-asked-from-where-the-answer-it-discriminates-was-asked-from.md), which makes the resolver part of the `Vantage`'s identity rather than a prober default. A resolver the code chose belongs to no `Vantage`, so it cannot be part of one's identity
- **Rests on:** [ADR-0121](./0121-the-operator-declared-recursive-resolver-is-trusted-and-exempt-from-the-discovered-authority-egress-guard.md), which exempts the **operator-declared** recursive resolver from the SSRF and rebinding egress guard. The exemption is granted on the ground that an operator declared the address. A code-supplied address takes the exemption without the ground
- **Rests on:** [ADR-0103](./0103-a-vantage-is-one-position-and-the-prober-is-optional-provisioning-detail.md), which makes a `Vantage` one position with the prober as optional detail. The shipped `local` row is the resolver-only case that ADR names
- **Sibling of, and not ruled by:** [ADR-0036](./0036-a-shipped-default-is-the-configuration-that-takes-effect.md), which rules a **third party's** shipped default as evidence for a curated table. It does not govern a disagreement between our own migration and our own code
- **Instance of:** [ADR-0207](./0207-an-enumeration-that-assembles-a-probing-target-set-drops-a-row-it-cannot-fully-name-and-never-fabricates-a-target.md). That ADR rules that a read assembling a probing target set drops a row it cannot fully name and never supplies the missing part. A `Vantage` with no resolver is exactly such a row, and §2's skip-and-record is that rule applied to the `dns` fan-out

## Context

[`internal/scan/scan.go:108`](../../internal/scan/scan.go) carried this text until #1318 deleted it:

```go
	// The resolver is carried on the Vantage and copied onto the job's spec by
	// the dispatcher; jobs built without one fall back to the same default the
	// migration ships (Docker's embedded DNS on the compose deployment), which the
	// operator replaces off compose. See db/migrations/18800_measurement_vantage.sql.
```

The sweep kept one compressed line at the same site, uncited under §4.7:

```go
	// 127.0.0.11 is Docker's embedded DNS, the same default db/migrations/18800 ships.
```

The comment states the equality as a rule to hold. It does not state which of the two values is the
source of truth. **No ADR and no SPEC states either.**

### The two copies

| Copy | Site | Value |
| --- | --- | --- |
| The migration row | [`db/migrations/18800_measurement_vantage.sql`](../../db/migrations/18800_measurement_vantage.sql) | `INSERT INTO vantage (name, class, resolver) VALUES ('local', 'unverified', '127.0.0.11:53');` |
| The code fallback | [`internal/scan/scan.go`](../../internal/scan/scan.go), `resolverFor` | `return "127.0.0.11:53"` when `j.resolver == ""` |

Nothing holds them equal. No test names the fallback at all —
[`internal/scan/scan_test.go`](../../internal/scan/scan_test.go) sets a resolver on every `Vantage` it
builds, so `resolverFor`'s fallback branch is exercised by no test in the tree.

### The fallback is not a rare path. It is the ordinary path for every provisioned Vantage

The triage record reads this as a latent drift hazard. It is more than that, and the extra facts
change which ruling is right.

1. **The column ships blank for a provisioned prober.**
   [`db/migrations/18700_vantages.sql`](../../db/migrations/18700_vantages.sql) declares
   `resolver TEXT NOT NULL DEFAULT ''`, and its own comment says the column *"ships blank for a
   freshly provisioned prober, which the operator then sets"*.
2. **`CreateVantage` never writes it.** [`db/queries/vantages.sql`](../../db/queries/vantages.sql)
   inserts `name`, `host`, `port`, `username`, `availability` and `created_by`. `resolver` is not in
   the column list, so every operator-created `Vantage` starts at `''`.
3. **No query anywhere sets it.** `db/queries/` holds six `UPDATE vantage` statements — latency,
   probe facts, public key, host key, and the two availability marks. **None writes `resolver`.**
   The console has no control for it. The guides tell the operator to change it with raw SQL, and
   they name only the shipped `local` row
   ([`docs/guides/using.md`](../guides/using.md) §4,
   [`docs/guides/running.md`](../guides/running.md)).
4. **The `dns` fan-out reaches every row.** `ListVantagesForDispatch` selects **all** vantages, and
   `BuildDNSJobs` builds one job per vantage. So each provisioned prober `Vantage` gets a `dns` job.
5. **`resolverFor` then fills the blank.** The job's scope carries `127.0.0.11:53`.

So on any install with a provisioned prober, `resolution` observations are attributed to a
`Vantage` whose resolver **nobody chose**. On a remote prober host `127.0.0.11` is either not routed
at all, or it is *that host's* Docker embedded DNS — a different resolver, answering differently, and
recorded as if it were the position's own.

### The operator cannot tell the two apart

That is the whole cost, and it is why the drift is worse than a wrong address. ADR-0070 makes the
resolver **part of the `Vantage`'s identity**. A `resolution` timeline is keyed on
`(subject, facet, discriminator, vantage, source)`. Two rows from one `Vantage` — one taken against
a resolver the operator declared, one against a resolver the code supplied — land on **the same
timeline** and are compared against each other. Nothing on either row records which happened.

The console reads the column and never the fallback, so both screens that name a resolver are wrong
for a blank row. Settings renders `v.Resolver` straight into its vantage list
([`cmd/web/settings.go`](../../cmd/web/settings.go), `fillVantagesSection`). Coverage substitutes an
em dash — [`cmd/web/cold.go`](../../cmd/web/cold.go) sets `resolver = "—"` when the column is empty —
and then tells the operator *"its resolver — was unreachable"* about a dial that went to
`127.0.0.11:53`.

### The default also collects a trust exemption it was not granted

ADR-0121 exempts the operator-declared recursive resolver from the egress guard, because the guard
once refused `127.0.0.11` and dead-lettered every default install
([#612](https://github.com/winniel123/verge-asm/issues/612),
[#239](https://github.com/winniel123/verge-asm/issues/239)). The exemption is keyed on the **path**,
not on the address: [`internal/measure/resolutionwalk/netpeer.go:34`](../../internal/measure/resolutionwalk/netpeer.go)
returns true for `PathDeclared`, and `Exchange` then dials with `trustedDialer()` rather than
`custodyDialer()`.

`Scope.Resolver` is what that path dials. So a value `resolverFor` invented is dialled through the
guard exemption that ADR-0121 granted to an operator's declaration. The ADR's warrant is the
operator's act. The fallback supplies the value and not the act.

### The repo already refuses to fabricate the row it cannot name

This ruling is the direction the code takes everywhere else it meets the same choice.

- `internal/queue/pure.go`, `toObservationParams`: a facet-less line is **skipped, never stored**.
- `internal/measure/edgefanout/run.go`: an unparseable address is *"our own error, never a measured
  value"*, and it is dropped.
- `internal/scan/httpidentity.go`: a service whose address does not parse is skipped rather than
  guessed.
- `Worker.deadLetter`: a failed job records an **empty scope**, never the attempted one.
- [ADR-0154](./0154-a-narrowing-fold-closes-only-what-it-can-attribute-to-a-mover-and-drops-every-other-candidate.md):
  a fold closes only what it can attribute and **drops every other candidate**.

Every one of those drops a row rather than inventing the part it cannot name. `resolverFor` is the
one site that invents it.

## Decision

> **A `Vantage`'s recursive resolver comes from its `vantage` row and from nowhere else. The
> migration's shipped value is the only source of truth. The code supplies no default. A `dns` job
> built for a `Vantage` with no resolver is an error the caller sees, and never a job that runs
> against an address the code chose.**

### 1. `resolverFor`'s fallback is removed

The function's whole body becomes the field read. The literal `"127.0.0.11:53"` leaves the Go tree.
After the change, exactly one artefact in the repository holds that address as an operative value:
`db/migrations/18800_measurement_vantage.sql`. The guides quote it, which is documentation and not a
second copy.

### 2. An absent resolver is an error at the dispatcher, not a value at the leaf

`Job.JobSpec` already returns an error. It returns one when `j.resolver` is empty, naming the
`Vantage`. The dispatcher sees it while it builds the job. Nothing is enqueued, so no `Batch` is
opened, so no silence is licensed — which is the same shape as ADR-0005's dead-letter rule reached
one step earlier.

The failure must stay **legible**, not merely loud. A `Vantage` with no resolver is an operator
configuration state, not a bug in a run, so the dispatch records the skip the way ADR-0005 records
an overlap skip. It must not open a `Batch`, and it must not fail the whole `dns` tick for the
vantages that are configured correctly.

### 3. The absence is never defaulted, in either direction

Not by the code, and not by a `DEFAULT` on the column that would put a live address into a row the
operator never touched. `resolver TEXT NOT NULL DEFAULT ''` stays as it is. The empty string is the
honest statement *"this position has no resolver"*, exactly as its own migration comment says, and
this rule makes that statement reach the operator instead of being answered by the code.

### 4. What this rule does not decide

- **The shipped value.** `127.0.0.11:53` stays, for ADR-0121's reasons, and its own migration
  states them.
- **How the operator sets a provisioned `Vantage`'s resolver.** There is no supported way today, and
  Consequences names that as its own defect.
- **Whether `local` should be the only resolver-only `Vantage`.** ADR-0103 rules that.

## Consequences

- **This ADR requires a code change, and the change is not made here. It ships as its own ticket.**
  Named precisely:
  - **Delete:** the fallback branch and the literal in `resolverFor`
    ([`internal/scan/scan.go`](../../internal/scan/scan.go)). The function collapses into `j.resolver`
    or disappears into its one call site in `Job.JobSpec`.
  - **Add:** an error return in `Job.JobSpec` when `j.resolver` is empty, naming the `Vantage`. The
    signature does not move — it already returns `(wire.JobSpec, error)`.
  - **Callers affected:** `enqueueJob` and `fanOutDNS` in
    [`internal/queue/queue.go`](../../internal/queue/queue.go). `fanOutDNS` decides what a blank
    vantage does to the tick. Per §2 it skips that vantage, records the skip, and enqueues the rest.
    It must not return the error up and abort the whole fan-out, because one unconfigured position
    would then stop every configured one.
  - **Tests:** no existing test covers the fallback branch, so none breaks. The change needs new
    coverage for the error path.
- **A second defect is exposed and it is larger than the first. It ships as its own ticket.** No
  query in `db/queries/` writes `vantage.resolver`, and the console has no control for it. So a
  provisioned prober `Vantage` has **no supported way to acquire a resolver at all**. Today the
  fallback hides that by answering with Docker's embedded DNS. Once §1 lands, the same install
  produces a legible skip instead of a silent wrong attribution — which is the improvement — but the
  operator still needs a way to set the value. The fix is a `SetVantageResolver` query and a control
  beside the existing prober provisioning. **The two tickets should land in that order, or the
  removal turns a silent fault into a blocked `dns` tick at a provisioned vantage.**
- **`CONTEXT.md` gains nothing.** The `Vantage` entry already carries the resolver as part of the
  position's identity, and ADR-0070 rules it.
- **No [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)
  withdrawal is owed.** This ADR supersedes no mechanism. ADR-0121's exemption is unchanged, and it
  becomes more nearly true: after §1 the exempted address is always one an operator declared.
- **`db/migrations/18800_measurement_vantage.sql` is unchanged.** Its long comment is the reasoning
  for the shipped value, and it stays the single site that carries it.
- **Nothing enforces this after the change, and nothing needs to.** Once the second copy is deleted
  there is one value, so there is no equality left to pin.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Keep both copies and pin them equal with a test** | A test can pin the two **texts** equal. It cannot pin their **semantics** equal, and the semantics are the whole fault. After such a test the fallback still answers `127.0.0.11:53` for a provisioned prober vantage whose operator's intent was *no resolver configured yet* — a case the migration's value was never about. The test would pass on every one of those installs while the measurement went to the wrong resolver. It also has nowhere honest to live: the assertion needs the migration text and the Go constant in one place, so it would parse SQL from a Go test to compare a literal against a literal |
| **Keep the fallback and log when it fires** | A log line is not a record. The `Batch` still records a scope, the `resolution` timeline still receives a row, and a later reader comparing two rows on that timeline sees no difference between them. The fault is that a measurement is unattributable, and attribution lives on the row, not in the worker's stdout |
| **Move the default onto the column — `resolver TEXT NOT NULL DEFAULT '127.0.0.11:53'`** | Puts a live, dialled address into every row an operator creates without touching the field, and it makes the row **look** declared when it was not. That is the same fault with the evidence destroyed, because the console would then show a resolver the operator never chose. ADR-0121's exemption would also become unconditional, since every row would carry a declared-looking value |
| **Delete the migration's value instead, and let the code own the default** | Inverts which artefact is the source of truth and loses the reasoning. `18800`'s comment carries #239, #612, ADR-0036 and ADR-0121 in one place, and it explains why the value is a loopback address and why a later session must not "fix" it. A bare Go literal carries none of that, and the operator's documented edit path is a `psql` `UPDATE` against exactly that row |
| **Fail the whole `dns` fan-out when any `Vantage` has no resolver** | One unconfigured position would stop the resolution of every configured one, which turns an operator's incomplete setup into a total measurement outage. ADR-0005 already has the right shape for this: a skip is recorded as a first-class operational event and the rest of the work proceeds |
| **Make `resolverFor` return an error and let the leaf handle the empty resolver** | The leaf already fails legibly on an unreachable resolver — `resolutionwalk.RunWithPeer` returns *"declared resolver unreachable; batch covers nothing"* — but by then a job has been enqueued, claimed, run, and retried five times against nothing. The dispatcher holds the `Vantage` row and can see the fault before any of that happens |
