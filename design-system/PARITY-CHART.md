# Parity chart v3.2 — round 2

Synced 2026-08-24T14:52Z against winniel123/verge-asm@main (tree e7cb9c0e0041), verifying the executed round-1 chart. **Wayfinder: start here.** The exact-parity ruling (design normative for look AND functionality, ADR-0116) and the SPEC-CHANGE.md escalation protocol are in force — round 1 proved them: 8 collisions filed, 4 ruled and built in-run, 4 correctly stopped and now ruled below.

## Landed — verified at head (do NOT redo or re-audit)

- P0.1 severity ramp + per-instance `SIG-####` signals (handlers.go #442, signals.go, sev classes in shell CSS; asserted by signals_test).
- P0.2 vs-last-batch deltas (deltas.go: dashboardDeltas / signalDeltas / exposureCountDeltas + signDelta; withheld-without-previous-batch degrade tested).
- P0.3 Reports trend series, MTTW + delta, heatmap with real intensities (reports.go; intensity steps mirror HeatmapCalendar.jsx; empty-heatmap honesty tested).
- P0.4 certs-expiring ≤30d datum built via collision #8 (leaf `not_after` captured, CertVersion v2); scan instants wired. Latency moved to P0.5 below.
- P1.2–P1.4, P1.6–P1.8 shell: inbox bell menu (unread dots, per-message deep-link, View all), org switcher menu (asset counts, check), footer, toast pipeline (query-blob across PRG + stack per ToastStack.jsx), initials avatar.
- P2.1 Dashboard (severity table `dsig-*`, framed stat band with deltas; placeholder regions gone), P2.2 Signals flat per-instance table (tests assert no census renders), P2.4 Reports incl. #9's "New assets discovered" card (P2.4b: ListSubjectFirstAppearances → DiscoverySeries), P2.5 Search Documentation group restored over docs/guides/, P2.6 Exposure stat-band deltas (tested).
- P2.7 hold sweep ran — produced collisions #10–12, all ruled and built: #10 Mark-unread mutation (messages.go, monotonic-read comment withdrawn), #12 Scope declared-name TreeView (renderSeeds → details TreeView with per-leaf max severity).
- P3 gate ran (#454); its findings are collisions #10–14.
- Escalation protocol landed (repo CLAUDE.md + ADR-0116) and worked as designed.

## Rulings on the four AWAITING DESIGN collisions (this sync — update SPEC-CHANGE.md log)

- Collision #7 → **NEW P0.5**: build the latency datum. Measure round-trip at the prober connect (the same connect that pins the host key), store nullable on the vantage, surface via probers.go; Dashboard Vantages renders the spec's pending "—" until a first measurement exists. A dedicated measurement ticket (worker + migration + read), not cmd/web-only.
- Collision #11 → **NEW P2.10**: build severity into the offline artifact model. RenderArtifact carries per-signal severity → the by-severity bars + SeverityBadge column per ReportArtifact.jsx; renders real values when the delivery backend lands (until then the body's empty-state stands — it is genuinely empty). Reconciles ADR-0024 with P0.1 for internal/message.
- Collision #13 → **U6, BLOCKED ON DESIGN — do not touch.** The Inventory spec predates the shipped subject/facet/span model (its columns are the older exposure-centric mock), so BulkActionsBar / SavedViews / TagInput filters have nothing real to bind. Inventory gets re-specced in the design workspace onto the real model; the fate of those three controls is decided there. Repo keeps its current Inventory meanwhile. Round-1's P2.8 is VOID — superseded by this ruling.
- Collision #14 → **CLOSED, no repo change.** Spec catch-up ships in this package: Profile.jsx realigned to the shipped account model (username identity — no display-name/email fields, no token Scope select, no recovery-code rotation; sessions with End-this-session + confirm dialogs) and gains the Access · single sign-on "Linked identities" section (provider table, unlink, link-an-identity row) matching templates_profile.go. Settings → Sessions was already specced (U2 · shot 25). Shot 14 recaptured.

## Executable now — round 2

- U1 Subject drill-ins (SubjectDetail.jsx, shots 22–24 · templates_inventory.go "service"/"endpoint", subjects.go; Name page renders AssetDetail per ruling).
- U2 Settings → Sessions (Settings.jsx SessionsSection, shot 25 · templates_settings.go, /settings/sessions/*).
- U3 missing-subject / missing-run ErrorPage kinds (shot 26) · U4 settings-forbidden kind (shot 27).
- D1 docs left-rail IA — frontmatter edits per docs/DOCS-IA.md (six sections, deterministic orders, short SSO titles).
- D2 VersionSelect via Tv's listVersions (releases + main·dev, newest tagged current).
- D4 ruling stands: render.jsx markdown→DS map IS the article spec; no fabricated article on a real route.
- D6 prev/next cards + TOC anchor links (spec in examples/DocsPage.jsx; Astro partial off nav-build's flattened NavSection[]).
- D3 needs no repo change (search-as-palette adopted into spec).
- P0.5 and P2.10 (rulings above).
- P0.6 **Dispatch + delivery backend** (collision #15, ruled by the owner 2026-08-24: BUILD IT LIVE). The on-cadence dispatcher and delivery backend #344 deferred (#285/#290/#291 chain, tracking #497): scheduled reports dispatch on their cadence, deliveries record receipts, and the 501/disabled scheduling surface comes alive per Reports.jsx (enabled New-schedule wizard → create; editable recurring list with Edit/Delete). The spec is unchanged — it was the target all along. Unblocks: P2.9 (schedule create/list live), P2.10's rendering (real artifacts to carry severity), reportDeliveryPage + Inbox delivery receipts on real deliveries. This is a subsystem — chart it as its own work-item tree, parallel to U/D items.
- P0.6b **Off-instance send = notify-with-link** (collision #16, owner ruling 2026-08-24). A delivery to a recipient sends only "your report is ready" + a session-authed URL to the in-instance artifact — the estate digest never leaves the instance (ADR-0039 stands; no estate-bearing SMTP). Build the minimal notification transport for that message only. The spec's recipient fields stay as-is; the artifact renders in-instance per P2.10.
- D5/P3b closing gate: side-by-side of round-2 screens plus the verify list below, light AND dark, 1440px, console against screenshots/ and docs against docs.jpg.

## Verify at the round-2 gate (round-1 claims not independently confirmed)

- P1.1 nav pill set and order (Exposure · Coverage present).
- P1.5 palette groups (Assets group, Sources/Port-aperture deep-links, First-run action).
- P1.9 Integrations reachable — the shell comment still describes `integrationsEnabled` as compile-time false; confirm the surface ships enabled or file a collision.
- P2.3 Graph severity-tinted nodes + drawer severity.
- P2.9 resolves via P0.6 (collision #15) — the 501 refusal was intentional pending the delivery backend; once P0.6 lands, verify create/list/edit/delete per Reports.jsx.

## Blocked on design

- U6 Inventory re-spec onto the subject/facet/span model (collision #13). Being designed in the design workspace; lands in a future package.

## Acceptance (unchanged)

1–4 as WORK-CHART §Acceptance. 5 no dropped affordance; 6 no added affordance (additions go through SPEC-CHANGE.md); 7 deltas/series render real computed values, unavailable reads degrade via the spec's own empty/skeleton pattern.
