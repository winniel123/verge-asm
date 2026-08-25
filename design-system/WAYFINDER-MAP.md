# WAYFINDER MAP — per-screen exact-parity audit (v4 loop)

The operating chart for bringing every web-UI screen to exact design parity under WORKFLOW.md. One screen at a time until the loop is proven, then batches. **Do not start a screen until the previous one has operator visual sign-off.**

## The loop (every screen, no exceptions)

1. **Design emits** (design workspace): `templates/<screen>.tmpl` (Go html/template, self-contained design-owned markup + scoped style + view JS), the screen's slice of `fixtures/fixtures.json`, Phase-A goldens (1440 × light/dark × states, rendered in the canonical container). Package VERSION bumps; operator lands it wholesale.
2. **Wayfinder item** (repo): embed the tmpl via `//go:embed` and route it; DELETE the repo-authored template + that screen's CSS delta; shape handler structs to the tmpl's declared holes (field names are the contract; the fixture file shows every value); seed fixtures; run G1 (byte-compare) + G2 (pixel, canonical container) locally; PR. Any impossibility → SPEC-CHANGE stop-and-escalate. NEVER edit the tmpl, css, goldens, or threshold.
3. **Operator visual confirm** (you): fixture-seeded repo screen next to the design preview, 1440, light AND dark, plus the screen's listed open states. Checklist: type sizes/weights, spacing rhythm, hover + focus states, exact copy, row data equals fixtures, theme toggle, no layout shift on interaction. Confirm in the design workspace ("inventory confirmed" / list mismatches).
4. **Design marks it LANDED** in this map. Next screen begins.

A confirmed screen is frozen: later changes to it start at step 1 with a version bump, never in-repo.

## Prerequisites (Wayfinder, before or with the pilot)

- P4.1 fixture loader (`verge seed --fixtures`, injectable clock) — pilot-scoped is fine.
- P4.2 harness (`verify/`: Playwright + pixelmatch + render-goldens + pinned container digest; CI checks G1/G2) — pilot-scoped is fine; full wiring by screen 3.
- CLAUDE.md block v2 (WORKFLOW.md §CLAUDE.md) replaces the v1 design-decisions block.

## The map (order of conversion)

| # | Screen | Route | tmpl | States to golden | Status |
| --- | --- | --- | --- | --- | --- |
| 1 | Inventory | /inventory | inventory.tmpl | record expansion open · gaps-only on · filtered-empty | **LANDED — operator confirmed 2026-08-24** |
| 2 | ErrorPage (all kinds) | /404 /403 /500 … | error.tmpl | — | **LANDED — operator confirmed 2026-08-25** |
| 3 | Profile | /profile | profile.tmpl | new-token dialog · end-session confirm | **LANDED — operator confirmed 2026-08-25** |
| 4 | SignIn (+ flows) | /signin … | signin.tmpl | TOTP step · forgot/reset · invite | **LANDED — operator confirmed 2026-08-25** |
| 5 | Setup | /setup | setup.tmpl | — | **LANDED — operator confirmed 2026-08-25** |
| 6 | Coverage | /coverage | coverage.tmpl | — | **LANDED — operator confirmed 2026-08-25** |
| 7 | Exposure | /exposure | exposure.tmpl | withheld state | **BATCH 2 — artifacts in v3.8.0** |
| 8 | Drift | /drift | drift.tmpl | feed item expanded | **BATCH 2 — artifacts in v3.8.0** |
| 9 | RunDetail | /runs/{id} | rundetail.tmpl | — | **BATCH 2 — artifacts in v3.8.0** |
| 10 | Scope | /scope | scope.tmpl | declared-name tree expanded · exclusion confirm · zone-file card | queued |
| 11 | Dashboard | / | dashboard.tmpl | scanning progress row · banner dismissed | queued |
| 12 | Signals | /signals | signals.tmpl | drawer open · descope confirm · annotation control open | queued |
| 13 | AssetDetail | /asset/{key} | asset.tmpl | — | queued |
| 14 | SubjectDetail | /subjects/… | subjectdetail.tmpl | withdrawn state | queued |
| 15 | Graph | /graph | graph.tmpl | node drawer open | queued |
| 16 | Reports | /reports | reports.tmpl | wizard steps 1–4 · row menu open | queued |
| 17 | ReportArtifact | /reports/… | reportartifact.tmpl | — | queued |
| 18 | Inbox | /inbox | inbox.tmpl | message open · mark-unread menu | queued |
| 19 | SearchResults | /search | search.tmpl | — | queued |
| 20 | Onboarding + FirstRun | /onboarding, / | onboarding.tmpl, firstrun.tmpl | wizard steps | queued |
| 21 | Settings (all sections) | /settings/* | settings.tmpl | one golden per section incl. sessions, SSO, vantages+prober flow, channels, sources, aperture | queued |
| 22 | Shell/chrome (LAST) | all | shell.tmpl | palette open · inbox bell menu · org switcher · toasts | queued |

Shell converts last because every earlier screen renders inside the current chrome; its goldens to that point mask chrome diffs by cropping to `<main>` (the harness crops until #22 lands, then goldens re-render full-page — a one-time golden regen, version-bumped).

Docs site: separate tree after #22 (its pipeline already renders design-owned assets; it gets D-map items then).

## Batching rule

Screens 1–3 strictly serial with operator sign-off between. From #4 on, up to 3 screens per batch — still one sign-off message per batch. Settings and Shell always solo.

Parallel sessions (multiple Claude Code sessions or Wayfinder subagents) are fine within a batch under these rules:
- **One screen per session, one branch/worktree each.** Templates are disjoint by construction; the collision points are shared files only: the embed registration, the fixture loader, verify/states.json wiring, CI config.
- **Same landed package version for the whole batch.** Never mix versions across sessions; a mid-batch package bump restarts the batch.
- **Shared-file edits are append-only** (add your screen's loader case / states entry; never reshape another screen's).
- **Merge serially**: first PR merges, the rest rebase and re-run G1+G2 before merging — a green branch is stale the moment another lands.
- The operator sign-off still covers the whole batch at once, after ALL its PRs merge.
