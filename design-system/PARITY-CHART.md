# Parity chart — make verge-asm match the design exactly

Generated 2026-08-24T02:14:21Z against winniel123/verge-asm@main (tree d13a12d6d404), auditing the executed v3 WORK-CHART port. Specs live in examples/, ground truth in screenshots/, portable source in components/ + tokens/. **Wayfinder: this chart is the entry point.** The collision protocol that prevents this class of drift recurring is SPEC-CHANGE.md — land it with P0.

## The ruling (binding, from the project owner)

The design in this package is normative for **look AND functionality**. The port's standing doctrine — "the domain term wins and the visual convention gets re-skinned around it" / "fabricated mock data is re-skinned to honest current-state facts + empty-states" (reports.go precedent; ADR-0024/ADR-0110 readings; docs/agents/design-system.md) — is **retired for composition and data shape**. Consequences:

1. Where the domain lacks a datum the design renders (severity, per-signal instances, deltas, trends, latency), the fix is to **build the datum**, never to empty-state or reshape the spec's region.
2. Empty states are for genuinely empty data only, using the spec's own empty patterns. A spec region may not render as a placeholder pointing elsewhere.
3. No dropped affordances, no added ones. What renders in the design preview is what ships.
4. Vocabulary rules are unchanged and still binding: signal / seed / channel / vantage / annotation; withdrawn, never resolved.

Record this as an ADR superseding the re-skin precedent before starting P1/P2, and adopt SPEC-CHANGE.md as the standing protocol: every future domain–spec collision is filed there and ruled on by design — never resolved port-side. Land its §Stop and escalate block into the repo's CLAUDE.md as part of P0, so any needed design decision stops the item, notifies the operator, and prints the hand-off prompt for the design workspace.

## Orchestrator guidance

P0 first (schema + derivations; everything else reads them). P1 is one tree (templates_shell.go is shared). P2 items marked ∥ touch disjoint files and parallelize after P0 lands. P3 is the closing gate.

## P0 — model prerequisites (land first)

- P0.1 **Severity + per-instance signals.** Five-level ramp (critical / high / medium / low / info) assigned to each rule, and per-instance signal rows (rule × subject) carrying first-seen / last-seen instants and a mintable `SIG-####` id. Shape spec: examples/console/SignalData.jsx. Targets: internal/signal/, db/migrations, cmd/web/signals.go. CONTEXT.md's "a signal carries no severity" is superseded by the ruling.
- P0.2 **Vs-last-batch deltas** for stat tiles: open signals, critical, assets watched, exposed services, certs expiring, exposure counts. Derivable from existing span/transition history. Targets: internal/drift/, handlers.go, exposure.go.
- P0.3 **Trend series** for Reports: signals over time, scans-per-day heatmap intensities, mean-time-to-withdrawal from transition history. Targets: reports.go, internal/drift/.
- P0.4 **Ancillary reads the spec renders:** certs-expiring-≤30d count (TLS corpus), per-vantage latency in ms (probers), "last full scan Xm ago · next in Yh Zm" instants (scheduler). Targets: handlers.go, probers.go, scans.go.

## P1 — shell (templates_shell.go) ← ConsoleApp.jsx + components/navigation/TopNav.jsx

- P1.1 Nav pills exactly and in order: Dashboard · Scope · Inventory · Drift · Signals (count) · **Exposure** · **Coverage** · Graph · Reports. Exposure and Coverage are missing today.
- P1.2 Remove the GitHub link from navactions — not in the spec's TopNav.
- P1.3 Inbox bell opens the recent-messages menu (per-message deep-link into Inbox, "View all") per TopNav's onOpenMessage; today it is a bare link to /inbox.
- P1.4 Org switcher opens its menu (the real org + asset count) per OrgSwitcher; today the button is inert.
- P1.5 Command palette parity with ConsoleApp.jsx: Screens group adds Inbox, Profile, Exposure, Coverage, Sources (settings deep-link), Port aperture (settings deep-link); Actions adds First-run onboarding; add the Assets group (top current Names, each → /asset/{key}). "Search everything" overflow stays as wired.
- P1.6 Footer (components/navigation/Footer.jsx) renders on console screens.
- P1.7 Toasts fire on acts (scan started/complete, saves, org switch) per ConsoleApp; the stack exists but nothing ever posts to it. Post-redirect toast rendering is fine; look/behavior per ToastStack.jsx.
- P1.8 Avatar shows the signed-in account's initials, not literal "VA".
- P1.9 Integrations surface ships enabled (`integrationsEnabled` is compiled false); the design shows it reachable from palette and Settings.

## P2 — screens: restore re-skinned regions

- P2.1 ∥ **Dashboard** (templates_dashboard.go, handlers.go) ← Dashboard.jsx · shot 01. Header sub-line (last scan / next in); ONE framed card with five Stat cells + deltas (Open signals · Critical · Assets watched · Exposed services · Certs expiring ≤30d) replacing the five loose .kpi tiles; Progress row with detail while scanning; dismissible warn Banner with Retry; By-severity bars with real counts; Coverage card with CoverageMeters + StalenessBadge; Vantages card with latency; Most-recent Signals table (Severity / Signal / Asset / Port / Seen, 6 rows, row-click → Signals). Both placeholder empty-state regions ("Signals carry no severity", "Coverage detail is on its own screen") are deleted.
- P2.2 ∥ **Signals** (templates_signals.go, signals.go) ← Signals.jsx + SignalData.jsx. Flat per-instance table — SeverityBadge, SIG id, signal, asset, port, seen — with severity filter, sort, text filter, pagination; tabs with counts; AnnotationControl; row Drawer; typed-name descope ConfirmDialog (already present). The per-rule census grouping leaves the screen (keep it data-side for the CSV if wanted). Export CSV stays — it IS in the spec (header, secondary button + download icon; now specced to export the current tab's filtered rows; disabled-when-empty degrades per the spec's empty pattern).
- P2.3 ∥ **Graph** (templates_graph.go, graph.go) ← GraphView.jsx. Severity-tinted nodes and severity in the drawer per spec (needs P0.1); the presence-only single-accent re-skin goes.
- P2.4 ∥ **Reports** (templates_reports.go, reports.go) ← Reports.jsx · shots 09→10. Restore the three re-skinned regions: trend KPIs with deltas (incl. the time-to-withdrawal KPI in its spec slot), heatmap with real intensities, the spec's period options. Keep row menu → View last delivery.
- P2.5 ∥ **Search** (templates_search.go, search.go) ← SearchResults.jsx. SeverityBadge back on signal rows (P0.1); restore the Documentation group, indexed over docs/guides/ (the dropped-group rationale "no content store" no longer holds — the guides are the store).
- P2.6 ∥ **Exposure** (templates_exposure.go, exposure.go) ← Exposure.jsx. Deltas on the stat band (P0.2). The runtime-conditional WITHHELD/board rendering stands — the spec's SegmentedControl is a preview affordance satisfied by real state.
- P2.7 **Hold sweep.** Grep cmd/web for `re-skinned|fabricated|honest|domain-incompatible`; every remaining hold either now has its datum (P0) or gets a work item. No spec region may empty-state while its datum exists.

## P3 — closing gate

Side-by-side pass of all 26 screens against screenshots/ at 1440px, light AND dark, before sign-off — including the screens the port marked "verbatim" (Scope, Inventory, Drift, Settings, SignIn, Setup, Onboarding, Profile, Inbox, AssetDetail, RunDetail, ReportArtifact, Coverage, FirstRun, Error). Verbatim claims are unverified until this pass.

## Acceptance (adds to WORK-CHART.md §Acceptance)

5. **No dropped affordance:** every control, column, filter, menu item, and state visible in the spec file exists and operates.
6. **No added affordance:** nothing user-facing renders that the spec doesn't show (current extra: GitHub nav link). A wanted addition goes through SPEC-CHANGE.md first.
7. Deltas and series render real computed values; an unavailable read degrades via the spec's own empty/skeleton pattern, never a bespoke placeholder region.
