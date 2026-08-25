# Package version

current: 3.8.0
exported: 2026-08-25

Convention: MAJOR.ROUND.PATCH — MAJOR is the package generation, ROUND increments per sync/Wayfinder round, PATCH per mid-round ruling or spec fix. The design workspace bumps this on every export; the repo copy always states what landed.

## Log
- 3.8.0 — 2026-08-25 · batch 2 (screens 7–9): exposure.tmpl (withheld first-class, delta chip, both-legs table), drift.tmpl (spec range picker with custom ISO pair, client-side kind toggles, collapsible batch groups, framed diff cards), rundetail.tmpl (Outcome = transitions + new signals per 2026-08-24 binding ruling, colored log levels, run-missing retired to error.tmpl); reconciliation #20; fixtures + states for all three; WORK-ORDER-7-9-BATCH2.md; SignIn/Setup/Coverage marked LANDED
- 3.7.0 — 2026-08-25 · batch 1 (screens 4–6): signin.tmpl (11 auth states, segmented code input, SSO not-configured state), setup.tmpl, coverage.tmpl (denominator meters for address scopes, relative-time messages); reconciliation #19; SignIn/Setup specs realigned to username identity; fixtures + states for all three; WORK-ORDER-4-6-BATCH1.md; Profile marked LANDED
- 3.6.0 — 2026-08-25 · screen 3 (Profile): profile.tmpl (PRG holes kept; spec cards/dialogs/badges/tables), reconciliation #18 (plain revoke confirm, TOTP-off state, toast mapping), profile fixtures + six goldens states, WORK-ORDER-3-PROFILE.md; ErrorPage marked LANDED
- 3.5.1 — 2026-08-25 · batching rule expanded: parallel-session mechanics (one screen/branch each, single package version per batch, append-only shared files, serial merges with G1+G2 re-runs)
- 3.5.0 — 2026-08-25 · screen 2 (ErrorPage): error.tmpl (six kinds, spec copy verbatim, hover copy control, chrome-when-signed-in rule), fixture accounts + deterministic incident id, six capture states, WORK-ORDER-2-ERROR.md; Inventory marked LANDED (operator confirmed)
- 3.4.1 — 2026-08-24 · claude/skills/consume-design-package/SKILL.md added — the repo-side skill encoding the v4 consuming contract (land a version, convert a screen, wire holes, gates, escalation)
- 3.4.0 — 2026-08-24 · P4.0 pilot artifacts: templates/inventory.tmpl (design-owned, self-contained, exact component values; carries recordrows + subject-missing), fixtures/fixtures.json (view contract + clock), verify/{states,config}.json, verify/PILOT-P4.0.md work order
- 3.3.2 — 2026-08-24 · WAYFINDER-MAP.md: per-screen conversion order (22 stops, Inventory pilot, shell last with crop-to-main rule), the 4-step loop with operator visual sign-off gates, batching rule
- 3.3.1 — 2026-08-24 · determinism section: one canonical pinned Linux container renders BOTH goldens and repo captures (goldens regenerated in-container from templates+fixtures, never workspace screenshots); packaged fonts; Windows/macOS runs advisory
- 3.3.0 — 2026-08-24 · WORKFLOW v4 ruled and charted: design-authored .tmpl view layer, fixture contract, hard CI pixel gate (G1 verbatim + G2 pixelmatch 3%), P4 migration tree (P4.0 Inventory pilot first); CLAUDE.md block v2
- 3.2.4 — 2026-08-24 · collision #13 ruled (U6): Inventory re-specced onto the shipped open-span read — dead controls retired, client-side kind/gaps/filter scope added; shot 03 recaptured
- 3.2.3 — 2026-08-24 · collision #17 ruled (P0.6c schedule→channel binding); wizard Delivery step + list Delivery column specced (shot 15); docs artifact copy fixed
- 3.2.2 — 2026-08-24 · collision #16 ruled (P0.6b notify-with-link delivery; ADR-0039 stands)
- 3.2.1 — 2026-08-24 · collision #15 ruled (P0.6 dispatch + delivery backend); SPEC-CHANGE log at 15 entries
- 3.2.0 — 2026-08-24 · round-2 chart: verified Landed ledger, rulings #7/#11/#13/#14, Profile spec catch-up (account model + SSO section, shot 14)
- 3.1.0 — 2026-08-24 · parity ruling + PARITY-CHART; SPEC-CHANGE escalation protocol; Export CSV specced
- 3.0.0 — 2026-08-22 · v3 full package: 26 screens, ~112 components, WORK-CHART
