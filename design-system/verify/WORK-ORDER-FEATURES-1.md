# Work order — dogfood features 1–4 (package v3.16.0)

Use the consume-design-package skill. Templates changed: scope.tmpl, settings.tmpl. fixtures.json + states.json extended. Land wholesale, after (or with) v3.15.0 and batch 8.

## DF-F1 — bulk scope declaration (paste-split)

### Behavior
- **Entry points**: the declared-scopes field on /scope (admin only, unchanged).
- **State transitions**: submit → PRG → /scope default (chips grown) and/or refusal rendering (one callout per refused token). scope·refusal state unchanged in states.json (same route + js).
- **Data contract**: POST /seeds, field `scope` = raw string. Server tokenizes on commas, whitespace, and newlines — the same tokenizer onboarding's seedsadd uses (cmd/web/onboarding.go); empty tokens drop. Each token validates independently: shape-inferred kind, the {{.AddressCap}}-address cap, duplicate check. Successes commit; failures fill `.Refusals[{Input,Reason,Reachable(nullable)}]` in declaration order. The single `.Refusal` hole retires.
- **Actions & side effects**: each declared seed is its own dated act. Flash on ≥1 success: `N scopes declared` (+ description `M refused — see the callouts` when M>0). All-refused → no flash, callouts only.
- **Edge cases**: duplicate token in one paste → first declares, second refuses `already declared`; a token that is both refused and duplicated reports its first failure only; no token-count cap (the cap is per-scope addresses).

## DF-F2 — bulk zone-file upload

### Behavior
- **Entry points**: the zone drop/file picker on /scope (now `multiple`).
- **State transitions**: choose/drop N files → one multipart POST /seeds/zone with N `zonefile` parts → PRG → /scope (zone rows updated, per-file refusals listed).
- **Data contract**: per file: apex inferred from the zone content (unchanged); apex outside the name scopes → a row in `.ZoneErrors[{File,Reason}]` (replaces `.ZoneError`); unparseable file → `not a zone file`. Accepted files each record their own dated act — the upload instant is that file's observation instant.
- **Actions & side effects**: flash on ≥1 success: `N zone files supplied` (+ `M refused` description when M>0). Removal detection windows restart per accepted apex.
- **Edge cases**: two files for the same apex in one drop → both acts recorded in order, the later is the current supply; zero accepted → no flash, refusal rows only; scroll preserved (requestSubmit, landed in v3.15.0).

## DF-F3 — live scan logs via RunDetail

### Behavior
- **Entry points**: Running now — the dispatch's scan kind is now a link (`.Active[].Href`); Recent dispatches rows link when `.History[].Href` is set. Both route to /runs/{dispatch}.
- **Data contract**: `.Active[]` gains `ID` + `Href`; `.History[]` gains nullable `Href` — populate for every dispatch with a run page. rundetail.tmpl is UNCHANGED: Status already carries `running`, `.Active` already renders the LIVE pulse.
- **Actions & side effects**: while a run's status=running the handler sets `.Refresh=5` (the head's meta-refresh hole) so the log tails on a 5s cadence; `.Log` is append-only within a run; Outcome holes stay `—` until the run concludes (existing contract). On conclusion `.Refresh` returns 0 and the page renders the terminal state.
- **Edge cases**: a dispatch stopped/terminated via DF-F4 renders its terminal status on the run page — the stopped/terminated badge treatments ship in the next design patch; until then map both onto the failed badge with the literal word in the label hole.
- **Fixtures/states**: rundetail fixture not keyed by run id — repo adds the running run 1408 per this order. settings fixture's active dispatch is now id 1408 with href /runs/1408.

## DF-F3b — per-job live output (v3.16.1 addendum)

### Behavior
- **Entry points**: each job id in the Running-now jobs table is a link (`.Jobs[].Href` = `/runs/{run}?job={id}`, nullable).
- **Data contract**: with `?job={id}` the run handler sets `.JobFilter{ID,Kind,Vantage,ClearHref}` (ClearHref = the bare run route) and filters `.Log` to that job's rows before rendering — the template renders whatever `.Log` it is given, no client filtering. The loghead chip renders from `.JobFilter`.
- **Actions & side effects**: live refresh unchanged (`.Refresh=5` while running — the filter survives refresh since it lives in the URL). Clearing the filter is a plain navigation.
- **Edge cases**: unknown job id → empty `.Log` + the chip still renders (honest empty: "No log to show"); a superseded job's id still links (its rows exist).
- **Fixtures**: settings active-dispatch jobs carry `href` (/runs/1408?job={id}).

## DF-F4 — stop dispatch / terminate

### Behavior
- **Entry points**: admin-only controls on each Running-now dispatch row: `Stop dispatch` (?stop={id}) and `Terminate` (?terminate={id}) — PRG dialogs (`.StopTarget{ID,ScanKind,Pending,Running}`, `.TerminateTarget{ID,ScanKind,Running}`, both nullable; counts are live at render).
- **State transitions**: scans → ?stop= dialog → POST /scans/stop → PRG scans; scans → ?terminate= dialog → POST /scans/terminate → PRG scans. Scrim/Cancel returns to ?tab=scans. New states: settings·scans-stop-confirm, settings·scans-terminate-confirm (in states.json, id 1408).
- **Actions & side effects**:
  - POST /scans/stop {id}: pending/ready jobs → cancelled; running jobs finish and commit; dispatch recorded `stopped · partial`. Flash `Dispatch stopped` / `N pending jobs cancelled · M running finishing`.
  - POST /scans/terminate {id}: running jobs killed, uncommitted work discarded; committed batches stand (append-only). Dispatch recorded `terminated`. Flash `Scan terminated` / `N jobs stopped`.
- **Edge cases**: non-admin POST → 403 (same gate as the trigger); id unknown or already terminal → PRG with danger flash `Dispatch already concluded`; a job mid-commit either lands whole or not at all (job atomicity — repo owns); the disabled cold tier can never be in flight, so never cancellable; `Run now` still refuses while the same scan is in flight (unchanged).

## Gates
G1: land wholesale. G2: two new settings dialog states need goldens — regen with #27f (full-page) or standalone for settings + scope + the v3.15.0 five.
