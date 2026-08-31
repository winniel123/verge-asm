# Bound the `/scans` in-flight monitor page

- **Status:** Ready for `/to-tickets` — the terminal artefact of [Map: Bound the /scans in-flight monitor page](https://github.com/winniel123/verge-asm/issues/930)
- **Ticket:** [#958 Consolidate the four decisions into the handoff spec](https://github.com/winniel123/verge-asm/issues/958)
- **Decisions assembled:** [#932 inventory](https://github.com/winniel123/verge-asm/issues/932), [#931 mechanism](https://github.com/winniel123/verge-asm/issues/931), [#933 history](https://github.com/winniel123/verge-asm/issues/933), [#953 summary contents](https://github.com/winniel123/verge-asm/issues/953)

This document is the map's destination: a plan-only spec complete enough to hand to an implementation session without that session re-litigating any decision the map already made. It does not restate reasoning — a decision lives in exactly one place, its ticket. It assembles and narrates. Every file and line anchor below is a **dated record** as of this document's composition (2026-08-31); verify against the tree before you edit.

**How to read this**
1. Start at §1 for why the page is unbounded and the shape of the fix.
2. Read §2–§5 for the change set, one surface at a time, with anchors.
3. Read §6 for the acceptance criteria an implementation must meet.
4. Read the four decision tickets for the reasoning behind any one rule.

Consult the `verge-asm-design` skill before you write any markup.

---

## 1. Destination and scope

The `/scans` monitor page renders as one huge scroll because the active-dispatch card draws an inline per-job columnar table — one row per vantage (`settings.tmpl:349-361`). A `Scan` fans out into one `Job` per vantage, so fan-out grows that table without bound. Run detail (`/runs/{dispatch-id}`) already exists and is already the per-dispatch drill-down, already linked from every monitor row (#932).

The bounding mechanism is **(b) summarize + drill** (#931). The monitor stops rendering the per-job table inline. The active-dispatch card shows a compact, stateless summary. Full per-job detail stays on the already-built run-detail page. Every change in this spec is **stateless by construction**, so it survives the in-flight `<meta http-equiv=refresh>`: no shifting rows, no snap-shut, no lost scroll or page-state.

**In scope:** the active-dispatch card (§2), the Recent-dispatches history (§3), the handler (§4), and the SQL (§5).
**Out of scope:** an all-dispatches index page. It is past "bound the monitor". Older dispatch detail stays reachable at `/runs/{id}` by direct link.

## 2. Active-dispatch card — replace the job table with a state-chip rollup

**File:** `design-system/templates/settings.tmpl`, Running-now section (lines 336–363). Source: #931, #953.

1. **Keep** the header count line (`:343`, `{{.Completed}} / {{.Live}} jobs · {{.Percent}}%`) and the progress meter (`:346`, `<div class="st-meter">`). The noun stays "jobs", not "vantages" — retries and supersessions make jobs not equal to vantages, so "vantages committed" would misstate the count. Add no second big count. (#953 point 1.)
2. **Remove** the inline per-job table and its guard: delete the whole `{{if .Jobs}} … {{end}}` block, lines **348–362** (the `<table class="st-table">` is 349–361). (#931; #953.)
3. **Add**, in its place, one row of **state-rollup chips**: one pill per job state, label + count, count in Geist Mono, reusing the `st-badge` tints already used in the removed table at `:356`. No new component. (#953 point 2.)
   - Show `running` (`st-badge accent`) and `done` (`st-badge ok`) **always**.
   - Show `ready` (`st-badge neutral`) and `dead` (`st-badge danger`) **only when count > 0**.
   - **Omit** `superseded` — it belongs on run detail.
4. **No per-vantage status strip.** It draws one cell per job, so it regrows with fan-out — the same unbounded shape being removed. It may return later as a separate fog item if operators miss it. (#953 point 3.)
5. **No timing.** No ETA (there is no estimate model, and it is out of the plan-only scope). No elapsed timer. Keep the existing relative `dispatched {{.DispatchedAt}}` line (`:342`). All timing stays on run detail. (#953 point 5.)
6. **Failure stays a count, not a name.** A dead vantage surfaces only as the `dead` count chip in danger colour. A name would pull per-vantage rows back into the summary; run detail already lists the names, and the operator drills for the who. (#953 point 4.)
7. **Add a "View all N jobs" drill button** in the card, linking to `/runs/{dispatch-id}`. `N` is the total job count for the dispatch. The scan-kind link at the card top (`:341`, `{{.Href}}`) already points to `/runs/{id}`; this adds an explicit labelled affordance because summarize + drill is the model. (#953 point 6.)

## 3. Recent-dispatches history — dedicate its own window and add a truncation callout

**File:** `design-system/templates/settings.tmpl`, Recent-dispatches section (lines 371–393). Source: #933.

1. **Decouple the shared cap.** Today Active and History are both carved from one `ListDispatchProgress(scansHistoryLimit)` read, split in the handler (see §4). A burst of in-flight jobs eats the shared 50 and silently shrinks completed history. Give History its **own dedicated window**; read Active in-flight **separately** — Active is bounded by reality, because few scans run at once. (#933 points 2, 4.)
2. **Cap value: 50**, now dedicated to History. This keeps today's visible depth exactly. The only behaviour change is that history no longer shrinks when scans are busy. Keep the `scansHistoryLimit = 50` constant (`scans.go:52`) as the History cap. (#933 point 4.)
3. **Truncation callout.** When the History window truncates, render one stateless line **below the history table** (after `:386`): "Showing the 50 most recent dispatches." No link, no pager — there is no period selector and no `/runs` index (`/runs/{id}` is per-dispatch only). Match the Drift `.dr-callout` pattern (`drift.tmpl:140-141`: an `.ic` svg + a `.tx` span, guarded by `{{if .Truncated}}`). Design the exact markup at implement time with `verge-asm-design`. (#933 points 1, 3.)
4. **Truncation detection:** fetch with `LIMIT N+1` (fetch 51, show 50, `Truncated = len > 50`). (#933 implementation note.)
5. **Rejected, do not build:** a Signals `?page=` pager (it reintroduces page-state), and stay-as-is silent eviction. (#933 point 1.)

## 4. Handler — split the shared read

**File:** `cmd/web/scans.go`, `fillScansSection` (~lines 148–176). Source: #932, #933, #953.

1. Split the single `ListDispatchProgress(ctx, scansHistoryLimit)` read (`:148`, split into `active`/`history` at `:152-176`) into **two reads**:
   - an **Active in-flight** read (in-flight dispatches only; no fixed cap — bounded by reality);
   - a **History** read capped at `scansHistoryLimit + 1` for truncation detection (§3).
2. For each **active** dispatch, keep loading `ListJobsForDispatch` (`:158`) **only to compute the per-state counts** for the rollup chips and the total `N` for the drill button. The card renders no per-job rows now, so the per-job view rows for the card become unnecessary. Decide at implement time whether to keep folding jobs into `dv.Jobs` or replace it with a small counts struct (`{running, ready, done, dead, total}`). The per-job `Href` wiring at `:162-169` served the removed table and can go with it.
3. Set `data["Active"]`, `data["History"]`, and a new `Truncated` flag for the History callout.

## 5. SQL

**File:** `db/queries/dispatch.sql`. Source: #932, #933.

1. `ListJobsForDispatch` (`:66-86`, no LIMIT) **stays as-is and stays used**: run detail still loads it, and the handler still needs the full per-dispatch job set to compute the rollup counts (§4). It is no longer the monitor's row-multiplier, because the card renders none of its rows.
2. Support the split read (§4): an Active-in-flight query and a History `LIMIT $1` query (called with 51).
3. Any migration or query change must ship regenerated `internal/db` (sqlc; see `CLAUDE.md`).

## 6. Acceptance criteria

1. The active-dispatch card renders no `<table>`. It shows the header line, the meter, a state-chip rollup (running/done always; ready/dead when > 0; no superseded), and a "View all N jobs" button to `/runs/{dispatch-id}`.
2. No per-vantage strip, no ETA, and no elapsed timer appear on the card.
3. The Recent-dispatches history reads from its own dedicated 50-row window, independent of in-flight volume. A busy queue does not shrink it.
4. When history truncates, one stateless callout reads "Showing the 50 most recent dispatches", with no link and no pager. It renders only when the window truncates.
5. The card and the history both survive the in-flight `<meta refresh>` with no shifting rows, no snap-shut, and no lost scroll or page-state.
6. `sqlc generate` is clean and regenerated `internal/db` is committed. The required CI checks pass.

## 7. Anchor table

| What | File:line |
|---|---|
| Active card header count line (keep) | `settings.tmpl:343` |
| Progress meter (keep) | `settings.tmpl:346` |
| Relative dispatched line (keep) | `settings.tmpl:342` |
| Per-job table + guard (remove) | `settings.tmpl:348-362` (table 349-361) |
| `st-badge` tints in the removed table | `settings.tmpl:356` |
| Scan-kind link to `/runs/{id}` (keep) | `settings.tmpl:341` |
| Recent-dispatches section | `settings.tmpl:371-393` (table 374-386) |
| History cap constant | `scans.go:52` (`scansHistoryLimit = 50`) |
| Shared read to split | `scans.go:148` (`ListDispatchProgress`) |
| Active/history split loop | `scans.go:152-176` |
| Per-active job load | `scans.go:158` (`ListJobsForDispatch`) |
| dispatch `Href` = `/runs/{id}` | `scans.go:1184` |
| Per-job query (keep) | `dispatch.sql:66-86` (no LIMIT) |
| Drift callout pattern to match | `drift.tmpl:140-141` (`.dr-callout`) |
