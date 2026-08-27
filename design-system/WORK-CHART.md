# Work chart — design v3 → verge-asm

> **Executed.** The corrective follow-up is PARITY-CHART.md — Wayfinder starts there. Future domain–spec collisions go through SPEC-CHANGE.md, never port-side judgment.

Generated 2026-08-22T22:13:00Z against winniel123/verge-asm@main. Baseline: the repo's design-system/ holds the v2 port (plus Carousel); everything below is what v3 adds or changes. Specs live in examples/, ground truth in screenshots/, portable source in components/ + tokens/.

Orchestrator guidance: every console screen renders through cmd/web/templates.go. Give each work item its own worktree for handler + partial logic, but serialize templates.go registrations through ONE integration tree (or land W0 first and rebase). Items marked ∥ touch disjoint handlers and are parallel-safe.

## W0 — mechanical, land first (single tree)
- W0.1 Replace design-system/ in the repo with this package's tokens/, components/, docs/, examples/, screenshots/ (1:1 copy). Fixes the corrupted screenshots (13 identical 4 KB files) and picks up the component deltas: VideoPlayer.jsx (new), TopNav.jsx (new props: onOpenProfile, onOpenMessage deep-link), Select.jsx listbox notes.
- W0.2 Two-role model: admin + viewer everywhere. The design dropped "operator". Check accounts/roles enums and copy.

## W1 — changed screens (implement the delta) 
- W1.1 ∥ Scope: zone-file card (dated re-supply upload, staleness → gap) + org-name registry search feeding Proposals. Spec examples/console/Scope.jsx · shots 02 · targets cmd/web/seeds.go, exclusions.go, proposals.go, internal/scan/zone.go.
- W1.2 ∥ Inventory: row click → Asset detail; saved views/columns/density as specced. Spec Inventory.jsx · shot 03 · targets cmd/web/inventory.go, subjects.go.
- W1.3 ∥ Drift: "Batch detail" entry to Run detail. Spec Drift.jsx · shot 04 · targets cmd/web/subjects.go, scans.go.
- W1.4 ∥ Reports: row menu → "View last delivery" (artifact). Spec Reports.jsx · shots 09→10 · targets cmd/web/messages.go.
- W1.5 Settings (large — split by section): Team (2 roles; change-role / require-re-enrollment / remove flows, invite dialog), Audit log, Instance health, Sources catalogue (consent tiers, admin-only toggles, terms dialog), Port aperture (locked sensitive tier, editable frequency tier), Vantages + prober provisioning (key render, host-key pin, egress declaration). Spec Settings.jsx + Sources.jsx · shots 15–18 · targets cmd/web/settings.go, sources.go, vergecore.go, probers.go, auth.
- W1.6 ∥ SignIn: forgot/reset, TOTP enrollment (secret, confirm, recovery codes), invite acceptance. Spec SignIn.jsx · shot signin.jpg · target cmd/web/auth.go.
- W1.7 Top nav shell: inbox icon with per-message deep-link, avatar menu (Profile, Settings, palette hint). Spec ConsoleApp.jsx + components/navigation/TopNav.jsx · target templates.go shell.

## W2 — new screens ∥ (each its own item)
- W2.1 Asset detail — ports census, DNS, TLS cert, provenance, signals-here, drift trail. Spec AssetDetail.jsx · shot 12 · targets cmd/web/subjects.go, exposure.go.
- W2.2 Run detail — stages, log, outcome, vantage health. Spec RunDetail.jsx · shot 11 · targets cmd/web/scans.go, scantrigger.go.
- W2.3 Report artifact — the delivered report rendered; doubles as PDF/email spec. Spec ReportArtifact.jsx · shot 10 · target internal/message/render.go.
- W2.4 Inbox — read/unread, mark-all-read, per-class detail, jump links. Spec Inbox.jsx · shot 13 · target cmd/web/messages.go.
- W2.5 Search results — grouped, highlighted. Spec SearchResults.jsx · shot 20 · target cmd/web/handlers.go.
- W2.6 Profile — identity, password + 2FA status, sessions, personal tokens. Spec Profile.jsx · shot 14 · target cmd/web/auth.go.
- W2.7 Exposure page — both-legs table + WITHHELD state when no internet vantage. Spec Exposure.jsx · shot 06 · target cmd/web/exposure.go.
- W2.8 Coverage page — aperture meters, coverage messages, gaps, unevaluable rules. Spec Coverage.jsx · shot 07 · targets cmd/web/handlers.go, cold.go.
- W2.9 First-run checklist home — the four steps, step 4 gated on an internet vantage. Spec FirstRun.jsx · shot 19 · target cmd/web/handlers.go.
- W2.10 Setup page — single-use token mints the admin account. Spec Setup.jsx · shot setup.jpg · target cmd/web/auth.go (/setup).
- W2.11 Error pages — 404 / 403 / 500 (500 carries a copyable incident id). Spec ErrorPage.jsx · shot 21 · target cmd/web/handlers.go.
- W2.12 Onboarding wizard — seeds → cadence → channel → review, queues first scan. Spec Onboarding.jsx · target cmd/web/handlers.go.
- W2.13 Integrations screen — tile grid + consent + install states. Spec Integrations.jsx · target cmd/web/settings.go.

## Acceptance (every item)
1. Side-by-side with its screenshot at 1440px, light theme: layout, spacing, hierarchy indistinguishable. Colors come from tokens, not the JPEG.
2. Copy verbatim from the spec file — vocabulary is binding (signal/seed/channel/vantage/annotation; withdrawn, never resolved).
3. Interactions match the spec's behavior notes (e.g. role-change Save stays disabled until the role differs; sources terms dialog gates enabling; exposure withheld state names why).
4. Both themes render (data-theme="dark" flips tokens); focus rings and reduced-motion behavior come free from tokens/base.css.
