# Work order — batch 2: Exposure + Drift + RunDetail (map #7–#9, package v3.8.0)

Use the consume-design-package skill. Second parallel batch: up to 3 sessions, one screen each, one branch each, all pinned to v3.8.0, shared files append-only, PRs merge serially with G1+G2 re-runs (WAYFINDER-MAP §Batching). Screens 1–6 are LANDED — do not touch.

## Screen 7 — Exposure (`templates/exposure.tmpl` replaces templates_exposure.go)

Defines and hole names kept ("exposure" + "expleg"): `.Withheld .Exposed .Firewalled .NotReached .HasDeltas .ExposedDelta.Change` (int; the `signDelta` funcmap entry stays) and `.Rows[{Asset,Svc,Internal,Internet,Since}]`. No new holes. The spec's "Spec state" segmented control is a design-workspace affordance — withheld renders from `.Withheld` only.
Reconciliation (SPEC-CHANGE #20f, ruled): the withheld action links `/settings/vantages` (provisioning a prober is a vantage act), not `/scope` — route or alias that path to Settings → Vantages. Table-empty keeps its honest copy with a Go-to-Scope action.

## Screen 8 — Drift (`templates/drift.tmpl` replaces templates_drift.go)

Defines kept ("drift" + "changeglyph"). Holes kept: `.Periods[{Token,Label}] .Period .HasEvents .BatchID .BatchLabel .Truncated .FeedLimit .Kinds[{Change,Family}] .Groups[…] .Movement`. New holes to wire:
- `.PeriodLabel` — the trigger label: the active preset's label, or "start – end" for a custom range.
- `.Groups[]` gains `.Collapsed` — true for groups older than the two most recent batches.
- `.TransitionCount` (int) + `.TransitionDelta` (nullable signed string, e.g. "+2", vs the previous period; empty string suppresses the chip).
- Custom range: the popover's form submits GET `/drift?start=YYYY-MM-DD&end=YYYY-MM-DD`; handler resolves it to a period window and sets `.Period` to a stable custom token for the export link.
Reconciliations (SPEC-CHANGE #20, ruled): period control is the spec's range picker — presets stay token links inside the popover, custom ISO pair added (#20b); the repo's "Derived · drift" microlabel above the h1 drops (#20c); kind chips are client-side toggles and batch groups collapse client-side — view JS ships in the tmpl (ADR-0105 precedent), so `.Groups` always carries the full period feed; the Movement tally follows the `.Kinds` vocabulary order (#20g). Batch detail routes `/runs/{id}`.

## Screen 9 — RunDetail (`templates/rundetail.tmpl` replaces templates_rundetail.go)

Define "run" kept, `.Run` struct kept: `.Title .Status .Scope .Meta .Stages[{Num,Title,Detail,Done,Current,Last}] .Log[{Tag,Level,Text}] .Active .Params[{K,V}] .Vantages[{Name,Latency,Status}]`. Changed holes:
- `.Completed`/`.Dead` retire → `.Transitions` + `.NewSignals` (strings; "—" until the run's diff stage has concluded). Reconciliation (#20a, ruled): the 2026-08-24 binding ruling applies — the Outcome card is the spec's; the drift feed and signals are keyed by batch, so BUILD the join (transitions folded from this batch's diff; signals first raised in it). ADR-0041's corpus separation stands for dispatch execution; the read joins derived stores, it does not move the comparison path.
- `.Degraded` becomes nullable `{Vantage, Detail}` (e.g. "ap-south-1" + "missed 2 of 3 checks").
Log levels render as colored text per LogViewer.jsx (#20e) — the level-pill treatment retires. The "run-missing" define retires (#20d): a missing run routes to error.tmpl's missing-run kind (landed screen 2).

## Fixtures (fixtures.json → exposure / drift / rundetail)

- exposure: 6 rows as specced; 14 exposed (+2 delta, HasDeltas true), 41 firewalled, 7 not reached. Variant `no-internet-vantage` (no internet-class vantage) drives the withheld golden.
- drift: period 7d of 4 preset tokens; 3 groups / 7 events incl. one diff (nginx banner) and the 2026-08-21 group collapsed; movement tally; transition_count 7, delta "+2"; batch_id `1407` ↔ label 2026-08-22T14:00Z.
- rundetail: run `1407` — complete · all scopes · "standard profile · 3 vantages · 3m 12s"; 4 stages all done; 7 log lines (one warn, one error); transitions "7", new_signals "3"; degraded ap-south-1; 5 params; 3 vantages (ap-south-1 degraded, latency "—").
Deterministic ids: 1407 is the fixture dispatch id (1408 stays the MISSING one for error goldens).

## Goldens (crop `main`, × light/dark @1440)

exposure: default · withheld (variant). drift: default · feed-expanded (click collapsed group headers) · range-open (click the period trigger). rundetail: default (/runs/1407).

## Acceptance

Byte-served from the three tmpls; no repo-authored markup/CSS remains for these routes; drift kind toggles filter events/groups and recount pills (single-active toggle resets to all); group collapse animates; period popover opens/closes on outside click + Escape and Apply enables only with both dates; exposure withheld renders from data; run Outcome shows transitions + new signals joined from the batch; G1+G2 green; SPEC-CHANGE gains no silent workarounds.
