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
- **P0.10 CertDetails fold** — `not_after`/issuer/algorithm/chain already ride the `certificate` observation; construct `CertDetails` in `buildEndpointFacts` (signals.go:1195-1221). Lights the 6 certificate rules.
- **P0.11 http-identity dispatch** — the `httpexchange` measurer is fully built but no prober switch case or scan kind dispatches it, and it sets no User-Agent (exchange.go:220). Add the dispatch + the UA. Lights the 4 HTTP rules; makes README's "one GET /, identifiable UA" true.
- **P0.12 (re-scoped by collision #36) TransitionDelta chip** — estate wiring is LANDED (see Landed ledger); the sole residual is drift.go:331's hardcoded `""`. Build the compare per #36's ruling: delta = current period window's transition count minus the immediately preceding equal-length window's, same scope/filters as `.TransitionCount`, presets and custom ranges alike; signed nonzero, "0" for zero, empty string (chip suppressed) when no complete previous window exists. Spec, fixture ("+2"), and drift.tmpl caption unchanged. Wayfinder: map #681.
- **P0.13 Coverage live numerator + stale callout** — address-scope counted/total meter has no live numerator (cold.go:283-291; fixture 198/214) and the per-zone stale callout is hardcoded `nil` (cold.go:259). Spec meters are ruled (#19c); build both reads.

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

## Verify at the round-3 gate

- Round-2 items above (U1–U4, D1/D2/D6) — round-2 gate never ran; confirm or file collisions.
- #31 cadence: confirm `prevFire` against every preset + a real cron expression, then mark #31 closed.
- Integrations "Send test": audit reads it as a no-op (integrations.go:298-305). If the spec drawer carries the affordance, that's a collision — file it; do not silently keep a dead button.
- P1.9 `integrationsEnabled` compile-time flag (carried from round 2).
- P1.1 nav pill set/order · P1.5 palette groups · P2.3 Graph severity tints (carried, still unconfirmed).

## P4 — v4 workflow migration

Unchanged; see WORKFLOW.md §Migration chart. All 26 screens LANDED through batch 8; P4 tree governs any re-emission.

## Blocked on design

- (none — R3-D1 is charted design work, not a blocker for repo rounds.)

## Acceptance (unchanged)

1–4 as WORK-CHART §Acceptance. 5 no dropped affordance; 6 no added affordance (additions go through SPEC-CHANGE.md); 7 deltas/series render real computed values, unavailable reads degrade via the spec's own empty/skeleton pattern. **8 (new): before filing any drift finding, consult AUDIT-LEDGER.md — matching rows are cited, not re-reported.**
