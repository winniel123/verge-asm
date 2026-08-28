# Parity chart v3.3 — round 3

Updated 2026-08-27 from the spec-drift audit of winniel123/verge-asm@main (`d94c4db`). **Wayfinder: start here.** The exact-parity ruling (design normative for look AND functionality, ADR-0116) and the SPEC-CHANGE.md escalation protocol remain in force. **New this round: AUDIT-LEDGER.md** — every audit finding is either settled there or charted below; future audits consult the ledger before reporting.

## Landed — verified (do NOT redo or re-audit)

Round-1 ledger (verified 2026-08-24 at e7cb9c0e0041):
- P0.1 severity ramp + per-instance `SIG-####` signals (handlers.go #442, signals.go, sev classes in shell CSS; asserted by signals_test).
- P0.2 vs-last-batch deltas (deltas.go; withheld-without-previous-batch degrade tested).
- P0.3 Reports trend series, MTTW + delta, heatmap with real intensities.
- P0.4 certs-expiring ≤30d datum via collision #8 (leaf `not_after`, CertVersion v2); scan instants wired.
- P1.2–P1.4, P1.6–P1.8 shell: inbox bell menu, footer, toast pipeline, initials avatar. (Org switcher since RETIRED by #33.)
- P2.1 Dashboard, P2.2 Signals flat per-instance table (tests assert no census renders), P2.4 Reports + P2.4b DiscoverySeries, P2.5 Search Documentation group, P2.6 Exposure stat-band deltas.
- P2.7 hold sweep (#10 Mark-unread, #12 Scope TreeView built). P3 gate ran (#454). Escalation protocol landed.

Audit-verified 2026-08-27 (`d9193f5..d94c4db`; operator gate-confirm at next sync):
- **#390/#391 (v3.18.0) — LANDED.** API v1 read-only (5 GETs off real store builders, off-by-default `api_enabled`, 404-before-auth, 405 non-GET), `vg_pat_` bearer auth (SHA-256 hash lookup, live role read), backup NDJSON stream (allowlist, raw pool), restore (preflight, server-side typed `restore`, TRUNCATE+replay in one tx, prober keys → pending, sessions deleted, `auth.RotateKey` on commit), guided update check (daily, gated, air-gap safe, never self-replaces).
- **P0.6 dispatch + delivery backend — SHIPPED** (#497): minute-poll dispatcher, per-(schedule, tick) idempotency, run-now download-only receipt (#29), artifact recomputes from period bounds, real fpdf PDF. Cadence/cron: audit reads `DispatchTick`→`prevFire` as honoring presets + real cron — closes #31 pending the verify item below.
- **P0.6b notify-with-link — SHIPPED**: `report.NewNotifyRunner` POSTs a link-only body via `delivery.SendSigned` (SSRF-guarded, HMAC-signed, retry/dead-letter). The report notice is production's only outbound POST — per-message delivery waits on P0.7.
- **#32 estate wiring — LANDED** (#637, merge 51141e8, confirmed via collision #36): `foldEstateTransitions` runs in the batch tx (worker.go:283); all six drift kinds fire; legend reads true. Do NOT re-file as build work.

## Round 2 — still open (carried; Wayfinder confirms at gate)

- U1 Subject drill-ins (shots 22–24) · U2 Settings → Sessions (shot 25) · U3 missing-subject/missing-run ErrorPage kinds (shot 26) · U4 settings-forbidden (shot 27).
- D1 docs left-rail IA (docs/DOCS-IA.md) · D2 VersionSelect · D6 prev/next cards + TOC anchors. (D3/D4 need no repo change.)
- P0.5 prober-connect latency datum (collision #7 ruling stands).
- P2.10 severity in the offline artifact model (collision #11) — now unblocked by P0.6 landing.
- U6 Inventory re-port (collision #13 ruled; spec Inventory.jsx · shot 03): JS-only delta over the shipped template — kind scope, Gaps-only, text filter, filtered-empty EmptyState; retired controls stay retired; no server-side search (ADR-0105).
- D5/P3b closing gate: side-by-side, light AND dark, 1440px, console vs screenshots/, docs vs docs.jpg.

## Round 3 — spec-audit intake (2026-08-27)

Every row traces to the audit. Build items follow the standing exact-parity ruling: these are specced regions whose data was never built — the spec stands, the datum gets built.

### Build items (repo)

- **P0.7 Message producer wire** — the audit's highest-leverage gap. Fold signal/drift transitions → `message.Flagship`/`message.Membership` → `InsertMessage` → `EnqueueForMessage`. Constructors, Inbox, bell, transport, retry/dead-letter all exist and are tested; only this wire is absent. Lights up: Inbox + bell on a real install, `delivery` table, per-message channel POSTs. Until it lands, E5/E6 banners apply. Sequence after P0.12 if membership messages should ride estate transitions.
- **P0.8 Internet-vantage push/remote-exec subsystem** (ADR-0103, `deploy/prober/`). Today `vantages.go` only pins the host key and times latency; every job runs on the instance host. Build: `ssh.NewSession` binary push + remote exec, `uname` arch check, `SSH_CLIENT` egress read, identifiable User-Agent on the probe path. Lights up: flagship `sensitive-port-reached-from-internet`, `non-globally-reachable…` (README's 5th rule), Exposure's real internet leg, VantageCard `HostKeyFingerprint`/`Platform`/`Egress` chips (currently declared fixture stubs). Largest item — chart as its own tree like P0.6 was.
- **P0.9 tls-acceptance fold** — one query from working: facet is measured weekly and persisted; add `ListServiceTLSAcceptance` + read in `buildServiceFacts` (signals.go:1086). Lights `tls-1.0-accepted`.
- **P0.10a CertDetails per-attribute fold** (re-scoped by collision #37) — make `CertDetails` per-attribute readable (each field independently nullable; no single pointer gates the set) in `buildEndpointFacts` (signals.go:1195-1221). Light only the `not_after`-derived rules now: `certificate-expired`, `certificate-expiring`. The other four evaluate per-attribute and stay not-evaluable until P0.10b. **Hard constraint:** SANMatchesName never defaults false — absent SANs → not-evaluable, never a mismatch verdict (#37 honesty ruling).
- **P0.10b full certificate leaf** (deferred datum build, #37) — bump the tls-handshake leaf to **CertVersion v3**: parse `not_before` + leaf SANs, specify weak-key/signature + self-signed determination policies, re-gate certcorpus. Lights `certificate-not-yet-valid`, `certificate-hostname-san-mismatch`, `certificate-weak-key-or-signature`, `certificate-self-signed`. Deferred and batched (not piecemeal): the CertVersion bump breaks `certificate` timelines by design (ADR-0082). Sequence with any other cert-leaf change. The 4 rules render the existing not-evaluable state until it lands.
- **P0.11 http-identity dispatch** — the `httpexchange` measurer is fully built but no prober switch case or scan kind dispatches it, and it sets no User-Agent (exchange.go:220). Add the dispatch + the UA. Lights the 4 HTTP rules; makes README's "one GET /, identifiable UA" true.
- **P0.12 (re-scoped by collision #36) TransitionDelta chip** — estate wiring is LANDED (see Landed ledger); the sole residual is drift.go:331's hardcoded `""`. Build the compare per #36's ruling: delta = current period window's transition count minus the immediately preceding equal-length window's, same scope/filters as `.TransitionCount`, presets and custom ranges alike; signed nonzero, "0" for zero, empty string (chip suppressed) when no complete previous window exists. Spec, fixture ("+2"), and drift.tmpl caption unchanged. Wayfinder: map #681.
- **P0.13 Coverage live numerator + stale callout** — address-scope counted/total meter has no live numerator (cold.go:283-291; fixture 198/214) and the per-zone stale callout is hardcoded `nil` (cold.go:259). Spec meters are ruled (#19c); build both reads.

- **P0.14 Integrations "Send test" + channel binding** (charted #38, re-scoped #39) — #38's "handler-only" premise is void: an integration tile holds no delivery target (integration_state = slug/state only; migration 20600 keeps Channels independent — "an integration is NOT a channel"). Per #39 build a **nullable integration→channel binding** (new integration_state FK; mirrors the report-schedule→channel pattern #17): (1) schema — the binding column; (2) drawer — a "Delivery channel" Select (bound Channel or "Not connected"), set post-install; (3) `testIntegration` posts the formatted test through the *bound Channel's* existing SendSigned transport + renders the spec toast on enqueue success; unbound → Send test disabled (no fake toast, #37 honesty); failure → existing degrade. settings.tmpl gains the selector + gated button. **Design landed HERE** (Integrations.jsx + ui_kits/console, v3.23.0); repo mirrors it. No screenshot change (no Integrations shot in the ground-truth set; slot 21 = Error 404).

### Watch (repo, latent — fix rides its trigger)

- **W1** `enabledProposers` gates on `unencumbered` consent only (proposals.go:333) — would silently suppress any future operator-accepted RIR runner after its `NoRunner` flag clears. Fix alongside the first real RIR `proposer.Source` (#30's bucket).

### Design-side (this workspace, next patch)

- **R3-D1 Backup card copy review** — the audit shows the archive carries `password_hash`/`totp_secret`/`token_hash`/SSO + channel secrets verbatim; only session-minting keys are excluded. The v3.18.0 "data-only, no secrets" card framing overpromises. Re-word the Settings · Backup card (and E12's doc lead) toward "same secret-leak posture as pgdata; session keys excluded and regenerated on restore." Spec + ui_kits + settings.tmpl together when executed.

### Errata (repo docs + stale comments — no design decision; execute as one docs pass)

- E1 `docs/guides/signals.md` — per-rule status column/banner: 4 Name rules fire on default install; 13 dormant (6 cert → P0.10, 4 HTTP → P0.11, 2 internet-gated → P0.8, tls-1.0 → P0.9).
- E2 `README.md:50-52` — "ships 5" → 4 (the 5th needs P0.8).
- E3 severity docs (`signals.md:26-30`, `v1-spec.md:488-489`, README) — describe the shipped ramp; #28 ruled docs the stale side.
- E4 signals guide/spec "three verdicts rendered" — describes the retired census; document the flat fired-instances table (P2.2 is normative).
- E5 `docs/guides/notification-channels.md` — banner: per-message generation unwired until P0.7; machinery built.
- E6 `README.md:135` "worker POSTs each message" — qualify until P0.7.
- E7 `README.md:35` "identifiable User-Agent, one GET /" — qualify until P0.11/P0.8.
- E8 `docs/guides/sources.md` — four RIR proposers "catalogued, not yet functional" (#30 ruled).
- E9 `authentication.md`/`accounts.md` — password "8–72" → "12–72" (#19d ruled).
- E10 `cmd/web/reports.go:51-54,932` — remove stale "no dispatch/delivery backend (#290/#291)" comments; document `/reports/export?format=pdf` (#29 ruled it normative).
- E11 `cmd/web/graph.go:55-58` — remove stale "domain apex not wired" comment (`classifyNameTypes` derives it).
- E12 `docs/guides/backup-and-restore.md` — lead with the pgdata-posture phrasing (pairs with R3-D1).
- E13 `docs/guides/integrations.md` — states table: "Installed and delivering" → declared intent only; drop unreachable `needs-config`; tiles marked "delivery/formatting not yet built."
- E14 `docs/guides/prober.md` — SSH push/exec, `uname` arch check, `SSH_CLIENT` egress read marked "planned" until P0.8; today SSH pins the key + times latency.

## Round 4 — dogfood intake (2026-08-27)

Dogfood session on main. UX findings, not spec drift. Design-side items landed in the UI kit + spec (v3.24.0) and the **served templates were regenerated in v3.24.1** (the actual export — v3.24.0 was planning-only). Repo-owned items charted as build work; each finding carries an AUDIT-LEDGER row (AL-30–AL-41).

### Design — landed this pass (ui_kits/console + examples/console + components)

- **R4-D1 Inventory endless scroll** (Inventory.jsx + component none) — controls bar now **sticky** (stays while scrolling a large corpus); groups are **collapsible** with a subject count and a **Collapse/Expand all**; each group caps at 25 rows with a **"Show all N"** expander. Repo: template must page/window server-rendered rows the same way (the tmpl still emits the full corpus). tmpl regen on handoff refresh.
- **R4-D2 Test channel action** (Settings.jsx · Channels) — per-channel **Send test** button; a channel always holds its own URL, so unlike an integration (#38/#39) it is always testable, no binding gate. Repo: wire to the channel's `SendSigned` (the delivery worker already exists). settings.tmpl channels block regen on refresh.
- **R4-D3 Scan-running visible everywhere** (TopNav) — replaced the faint micro-dot with a **persistent pulsing "Scan running" pill** in the top nav (present on every view since the nav is global), clickable → Settings · Scans. Repo: the shell partial reads the same in-flight flag.
- **R4-D4 Heatmap zero-day cells** (HeatmapCalendar) — empty days now render a bordered `--border-default` cell (was near-invisible `--row-sep`), so "nothing happened" reads as a present box. Component already mapped every day; this is a contrast fix. Repo: the served heatmap must emit one cell per day incl. zeros (dogfood shows gaps).
- **R4-D5 Coverage card overflow** (CoverageMeter + Dashboard) — meter label truncates with ellipsis + `min-width:0`; the staleness line wraps instead of overflowing the 380px card.
- **R4-D6 Settings nav shift** (Settings grid) — rail was already fixed at 210px + `minmax(0,1fr)` content; confirmed no width change per tab. Residual shift is the page scrollbar toggling between short/tall tabs → repo adds `scrollbar-gutter: stable` on the app scroll container (charted, cosmetic).
- **R4-D7 Job streams exact output** (RunDetail) — opening a **running** job now streams that job's own stdout live (`LogViewer live`, per-job output, auto-follow), not a filtered slice of the batch log. Also brought the package's stale examples/RunDetail up to the UI kit (it predated the job-filter feature). Repo: stream real per-job output to this view (P0.14-style wire; the run/job model exists).

### Repo — charted build work (no design decision unless noted)

- **R4-R1 Cloudflare/WAF-fronted false positives** — a domain behind Cloudflare et al. fills inventory with edge-shared, non-actionable spans. Classify provider-fronted addresses and suppress/segregate them from actionable inventory. **Carries a design question** (below) — do not build the surfacing UI until ruled.
- **R4-R2 Seed delete → 500** — deleting a seed errors. Handler/tx bug; fix + regression test.
- **R4-R3 CT scans keep retrying** — the `ct` job never settles, re-dispatches endlessly. Scanner retry/terminal-state logic.
- **R4-R4 Graph → PNG export wrong** — the export engine lives in the design-owned graph.tmpl (frozen port). **Design-side fix landed v3.24.1**: the old exporter serialized the live SVG, whose `var(--…)` fills/strokes collapse to black inside the isolated raster image — rewrote it to inline each element's resolved presentation styles, reset the pan/zoom transform (export the whole graph), bake the page background, and render at 2×. Repo re-verifies against the on-screen graph.
- **R4-R5 Org-Discovery CIDRs don't scan** — CIDRs added by Org Discovery can't be hot-scanned and appear to never scan at all. Scan dispatch/scope-eligibility bug for org-sourced ranges.

- **R4-Q1 provider-fronted spans in Inventory — RULED 2026-08-28 (#762).** Blanket-responder (Cloudflare/WAF) spans surface **shown by default**, with a "proxy edge" badge on the reach-Gap and **demoted in place** by the existing value-before-Gap sort (no new sort key), plus an ADR-0072-legal **"Hide proxy edge" toggle** (default SHOW). Hide-by-default and separate-bucket were rejected (ADR-0104/0105 — the estate shows what it holds; the operator chooses to hide, the system never hides for them). Landed in inventory.tmpl + Inventory.jsx (v3.24.1); repo classifier already exists (feeds ProxyEdge). Closes the round-4 open ruling.

## Verify at the round-3 gate

- Round-2 items above (U1–U4, D1/D2/D6) — round-2 gate never ran; confirm or file collisions.
- #31 cadence: confirm `prevFire` against every preset + a real cron expression, then mark #31 closed.
- Integrations "Send test": **RULED — #38 → refined #39 (channel binding).** Confirm P0.14: the drawer's Delivery-channel binding persists, Send test posts via the bound Channel's SendSigned + renders the spec toast, and is disabled ("Connect a channel to test") when unbound — no silent redirect, no fake toast. Then close verify item V.
- P1.9 `integrationsEnabled` compile-time flag (carried from round 2).
- P1.1 nav pill set/order · P1.5 palette groups · P2.3 Graph severity tints (carried, still unconfirmed).

## P4 — v4 workflow migration

Unchanged; see WORKFLOW.md §Migration chart. All 26 screens LANDED through batch 8; P4 tree governs any re-emission.

## Blocked on design

- (none — R3-D1 is charted design work, not a blocker for repo rounds.)

## Acceptance (unchanged)

1–4 as WORK-CHART §Acceptance. 5 no dropped affordance; 6 no added affordance (additions go through SPEC-CHANGE.md); 7 deltas/series render real computed values, unavailable reads degrade via the spec's own empty/skeleton pattern. **8 (new): before filing any drift finding, consult AUDIT-LEDGER.md — matching rows are cited, not re-reported.**
