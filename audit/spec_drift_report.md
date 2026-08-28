# Spec Drift Audit — verge-asm — 2026-08-27 (re-run 3 — map #720 verification)

> **Ledger-diffed re-run.** The 2026-08-26 baseline (~118 features) and the register in
> `design-system/AUDIT-LEDGER.md` settled the feature surface; re-run 2 (earlier today) verified the
> P0.7–P0.14 landing commits and left **three** items open: `AL-7` (4 HTTP rules could not fire),
> `NF-1`/`AL-30` (2 of 4 notification causes), and `AL-31` (stale rule-count docs). This run checks
> those three against the **current `origin/main`**, on the ground, commit-by-commit — not by trusting
> commit titles (re-run 2 caught the *first* P0.11 commit `f7cdb25` overclaiming exactly that way).
>
> **Result: all three are resolved on `origin/main`. No new drift.** The remaining action is
> bookkeeping — the local checkout is behind and the ledger resolution is unmerged (see §Repo state).

## Summary

| Status | Count |
|---|---|
| Re-run-2 open items now verified **RESOLVED** on `origin/main` | 3 |
| **New** drift findings this run | 0 |
| Ledger rows re-confirmed unchanged (excluded from counts) | 26 |
| Bookkeeping/merge-state observations (not code drift) | 2 |

**Headline.** Nothing on the product surface drifted since re-run 2 — the movement is all in the
*right* direction. The three items re-run 2 left open have each been closed by a real, load-bearing
commit now on `origin/main`, and I verified each by reading the code the commit adds rather than its
message:

1. **`AL-7` — the 4 HTTP-identity rules now fire.** `80fafac` (P0.11 **redo**, #721/PR #724) lands the
   dispatch the earlier `f7cdb25` only pretended to: a new `http-identity` scan kind (migration
   `23400_http_identity_scan.sql` widens the `scan_kind_check` closed union and ships the scan
   **enabled, daily**), a real producer (`internal/scan/httpidentity.go` `BuildHTTPIdentityJobs`) that
   fans one `http-exchange` job per vantage over the reached-service population, queue fan-out
   (`internal/queue/httpidentity.go`), and scheduler routing (`internal/queue/queue.go`) — mirroring
   the `tls-acceptance` extra-scan pattern. The facet is now persisted, `HTTPResponded` can be true,
   and all four rules can reach `fired`.
2. **`NF-1`/`AL-30` — the `declared-input` cause now fires; `threshold` is honestly deferred.**
   `8d3dc0e` (#722/PR #725) wires `message.DeclaredInput` into `produce.go` (`declaredInputMessages`,
   fed by `foldEstateTransitions`' operator-caused `descoped` closures, linked to the covering
   Exclusion as Source, routed like the flagship/membership legs). Crucially it did **not** fake-wire
   `threshold` — it descopes it with an honest "planned; not yet emitted" caveat in
   `notification-channels.md` and a follow-up (#727) for the clock-driven cert-horizon sweep a faithful
   `threshold` firing needs. Three of four causes now fire; the fourth is honestly documented as not.
3. **`AL-31` — the rule-count docs are corrected.** `f8cbcae` (#723/PR #726) updates `README.md` and
   `docs/guides/signals.md` from the stale "9 of 17" to **15 fire on a default install, 17 with a
   provisioned prober** (4 Name + `tls-1.0-accepted` + 6 certificate + 4 HTTP-identity; the 2
   internet-gated flagship rules are the path to 17/17). Certificate rows moved Dormant→Live; the
   stale `(#700)` / "wait on the leaf" framing retired. The arithmetic is internally consistent with
   re-run 2's interim count of 11 plus the 4 HTTP rules `80fafac` lit.

**Corrected rule count (now matching the docs):**

| Install shape | Rules that fire | Composition |
|---|---|---|
| Default single-host (no prober) | **15 of 17** | 4 Name + `tls-1.0-accepted` + 6 certificate + 4 HTTP-identity |
| With a provisioned internet prober | **17 of 17** | + the 2 internet-gated flagship rules |

---

## Repo state — resolved during this run

At the start of this run the checkout was behind and the ledger resolution unmerged; both were fixed
mid-run by a `git pull`:

1. **Checkout caught up.** Local `HEAD` was `1a79dea` (4 behind); a `git pull --ff-only` fast-forwarded
   it to **`7741ab1`**, now level with `origin/main`. The three code/doc fixes (`80fafac` AL-7,
   `8d3dc0e` AL-30, `f8cbcae` AL-31) are now in the working tree, not just the fetched remote.
2. **Ledger resolution merged.** The pull also brought `7741ab1` (#728), which lands the
   `chore/wayfinder-720-ledger` content on `main`: `design-system/AUDIT-LEDGER.md` now records
   `AL-7 CHARTED→LANDED`, `AL-30`/`AL-31` `PROPOSED→RESOLVED`, and the "map #720 completion"
   amendment-log entry. Ledger and code now agree.

**Remaining (soft):** `main`'s committed `audit/spec_drift_report.md` is still the 2026-08-26 baseline —
this re-run-3 report is uncommitted in the working tree. If the audit artifact is meant to travel with
the repo, this file wants committing.

---

## Verified resolutions — on-the-ground evidence

Each traced by reading the diff the resolving commit adds, on `origin/main`.

### AL-7 (P0.11 redo) — `http-identity` dispatch — **RESOLVED** (`80fafac`, #721/PR #724)
The load-bearing dispatch re-run 2 found missing is now present:
- **Scan kind admitted.** `db/migrations/23400_http_identity_scan.sql` widens the closed union to
  `CHECK (kind IN ('hot','cold','tls-acceptance','zone','dns','ct','http-identity'))` (grown one member
  at a time, as `ct` was in 21100) and ships `INSERT INTO scan (kind, enabled, cadence_seconds) VALUES
  ('http-identity', TRUE, 86400)` — enabled, daily, unassisted.
- **Producer.** `internal/scan/httpidentity.go` `BuildHTTPIdentityJobs` fans one job per Vantage over
  the reached-Service population (reusing `ReachedService`), rendering each as the nameless Endpoint
  and dispatching the `http-exchange` leaf (scan kind ≠ leaf kind, as hot/cold dispatch
  connect-outcome). `internal/queue/httpidentity.go` `fanOutHTTPIdentity` enqueues them;
  `internal/queue/queue.go` routes `scan.HTTPIdentityKind` through the scheduler whitelist and fan-out.
- **Consequence.** reachable endpoint → `http-exchange` dispatched → `http-identity` facet persisted →
  `HTTPResponded` true → `plaintext-http-no-https`, `redirect-does-not-upgrade-to-tls`,
  `redirect-to-host-outside-estate`, `unauthenticated-request-answered` can all reach `fired`.
  Default install **11 → 15**. Producer test: `internal/scan/httpidentity_test.go` (mirrors the
  `tls-acceptance` producer test).
- **Contrast with the overclaim re-run 2 caught:** the earlier `f7cdb25` added only the prober `case`
  + User-Agent; `80fafac` adds the scan kind + job producer that feed it. The dispatch is real this time.

### NF-1 / AL-30 — notification causes — **RESOLVED** (`8d3dc0e`, #722/PR #725)
- **`declared-input` WIRED.** `internal/queue/produce.go` gains `declaredInputMessages`, which reads
  `foldEstateTransitions`' operator-caused `descoped` closures (a Name an operator Exclusion narrowed
  out of the estate; ADR-0111) and folds each into one coverage-class `message.DeclaredInput`, computed
  in the batch transaction, linked to the covering Exclusion as its `LinkSource`, routed to bound
  channels exactly like the flagship/membership legs. A `measured-absent` (world-withdrawn) closure
  carries no source and fires no `declared-input` message — valence-free, per the cause's definition.
  Tests mirror `produce_test.go`.
- **`threshold` honestly DESCOPED (not fake-wired).** A `threshold` firing is a horizon crossing with
  no observation/batch event (the cert value doesn't move at the crossing), so the observation-driven
  producer structurally cannot see it; faithful emission needs a dedicated clock-driven cert-horizon
  sweep with fire-once dedup → follow-up **#727**. `notification-channels.md`'s `threshold` row now
  reads "planned; not yet emitted." This is the honest resolution — three causes fire, the fourth is
  documented as not — so the doc no longer overclaims.

### AL-31 — rule-count docs — **RESOLVED** (`f8cbcae`, #723/PR #726)
`README.md:49-55` and `docs/guides/signals.md` §"Rule status" corrected to **15 fire on a default
install / 17 with a provisioned prober**, with the per-rule table's six certificate rows moved
Dormant→Live (P0.10) and the four HTTP-identity rows shown Live (P0.11). The stale `(#700)` / "wait on
the leaf" framing on the two internet-gated rows retired to "needs a provisioned prober." Ground truth
and docs now agree, and the "dormant is never faked — sits `outside-domain`" honesty note is preserved.

---

## Known findings (ledger) — re-confirmed unchanged, not re-reported

- **AL-2 / AL-3 / AL-5 / AL-6 / AL-8 / AL-9 / AL-10** — the seven CHARTED→LANDED flips re-run 2
  verified (message producer, internet-vantage SSH push/exec, `tls-1.0-accepted` read-back, all six
  certificate rules, drift TransitionDelta chip, coverage live numerator, integrations Send-test). No
  code touching them landed since re-run 2; unchanged. *(AL-6 remains LANDED-FULL 6/6; AL-2 remains
  LANDED with its AL-30 residual now itself resolved; AL-10 remains LANDED at Send-test scope, with
  per-integration estate delivery unwired by re-scoped intent, not drift.)*
- **AL-1 (severity ramp, RULED), AL-4 (superseded by AL-31), AL-11–AL-29** — no code change touching
  these since the ledger; all remain as ruled / landed / by-design. Not re-audited line-by-line.

No new drift surfaced against `origin/main` this run.

## Recommended actions

Code, docs, and ledger are all correct and merged on `main` (checkout now at `7741ab1`). Only two
soft items remain:

1. **Commit this audit artifact** (`audit/spec_drift_report.md` + `feature_inventory.md`) if the report
   is meant to travel with the repo — `main`'s committed copy is still the 2026-08-26 baseline.
2. **Track #727** (clock-driven cert-horizon sweep) as the remaining path to the fourth notification
   cause (`threshold`); it is honestly documented as deferred today, so it is not drift.

## Proposed ledger amendments

Already prepared on `chore/wayfinder-720-ledger` (`9e48e05`) — this run confirms them correct:
- **AL-7** CHARTED → **LANDED** (`80fafac`/#721/PR #724 — scan kind + job producer; 4 HTTP rules fire;
  default 11→15).
- **AL-30** PROPOSED → **RESOLVED** (`8d3dc0e`/#722/PR #725 — `declared-input` wired; `threshold`
  descoped w/ caveat + follow-up #727).
- **AL-31** PROPOSED → **RESOLVED** (`f8cbcae`/#723/PR #726 — README + signals.md → 15/17).

No further ledger rows proposed — this run found nothing the ledger doesn't already anticipate.

## Coverage

- **Scope.** Deliberately narrow: verify the three re-run-2 open items against current `origin/main`,
  and scan for any *new* drift the four ahead-of-local commits might introduce. The ~118-feature
  surface and the 7 LANDED flips are settled (baseline + re-run 2 + ledger) and were not re-audited.
- **Method.** Static reading of the resolving commits' diffs via `git show` (initially against fetched
  `origin/main` objects while the working tree was still at `1a79dea`; then fast-forwarded to
  `7741ab1`), plus targeted grep — the same entry-point-inward discipline, reading the code each commit
  adds rather than trusting its title. Not run locally.
- **Authoritative state.** `main` is now at `7741ab1` (checkout level with `origin/main`) — the truth
  of what ships; the re-run-2 draft it superseded is folded into this report.
- **Unverifiable items:** none.
- **Ledger consulted:** `design-system/AUDIT-LEDGER.md` — now carries the resolved state on `main`
  (AL-7/AL-30/AL-31 + map-#720 amendment, merged via #728/`7741ab1`). It absorbed re-reporting of all
  26 unchanged rows; the 3 open items are resolved, not re-litigated.
