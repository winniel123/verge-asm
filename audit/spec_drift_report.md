# Spec Drift Audit — verge-asm — 2026-08-26

## Summary

| Status | Count |
|---|---|
| Implemented | 77 |
| Partial | 20 |
| Stub | 10 |
| Missing | 9 |
| Undocumented / reverse-drift | 2 |
| Doc↔code contradiction | 1 |
| **Total features audited** | **~118** |

**Headline.** verge-asm is a real, substantial system — the DNS/TCP/TLS measurement spine, the drift/span engine, auth/SSO, the report renderer, and the ops plumbing are genuinely built, wired, and tested, and the code is unusually honest about its own seams. The drift is concentrated in three places, and it is structural rather than cosmetic: **(1)** the entire outbound-notification pipeline never fires because nothing generates a `Message` (channels, deliveries, retries, and all 8 integration tiles are real machinery hanging off a producer that does not exist); **(2)** the "internet vantage" — the outside observer the whole *Exposure* thesis rests on — is hollow: the SSH binary-push is unimplemented, so no external measurement ever happens; **(3)** consequently only **4 of the 17 signal rules can fire on a default install**, not the 5 the README claims and not the 17 the signals guide describes in present tense. Add 8 do-nothing integration tiles, 4 do-nothing RIR-proposer tiles, and a flat doc↔code contradiction over signal *severity*, and those are the gaps a user hits first.

**Highest-impact gaps** (ordered by how fast a user hits them):

1. **Notifications never fire** — `Message` generation is **Missing**. No message is ever written in a real deployment, so no channel POST, no integration, and no report-ready notice ever emits. The product's core "tell me when something changed" promise is inert. (`internal/message/*` constructors + `InsertMessage` exist and are tested, but have zero production callers.)
2. **Internet vantage is hollow** — SSH push/remote-exec is a **Stub**; every job runs on the instance host regardless of vantage. *Exposure* (the documented landing view) can never be genuinely constructed from an outside observer, and the flagship signal can never fire.
3. **Only 4 of 17 signals can fire** — README says 5 (overcounts by one); `docs/guides/signals.md` describes all 17 in firing present-tense with no per-rule status.
4. **8 integration tiles do nothing** — install/disconnect state is real; downstream effect is **Missing** for every tile. The guide's "installed = delivering" is false.
5. **4 phantom RIR-proposer tiles** — RIPEstat, RIPE Database, APNIC-registry, LACNIC render consent dialogs and toggles but have no runner and are filtered out even when enabled; they emit zero proposals.
6. **Severity contradiction** — three docs (signals guide, v1-spec, README) emphatically say signals carry *no* severity and *no dial can add one*; the shipped engine and UI are built around a 5-level severity ramp (badge, filter, default sort, CSV column).
7. **Transition grammar is half-live** — of `appeared`/`returned`/`revealed` the README names, only `appeared` (and `changed`) ever fire; the rest are legend-only, yet the six-word legend always renders.

---

## Findings by subsystem

### Signals

The 17 rules are all real, pure, tested `Eval` functions, and `EvaluateCorpus` runs all 17. Firability is gated entirely by whether the **measurement pipeline folds the facts a rule reads**. On a default install the pipeline produces `resolution`/`dns-record` (dns scan, enabled daily), `reachability`+`certificate` (hot scan, enabled), and `tls-acceptance` (weekly) — all at the single seeded `local`/`unverified` vantage. It does **not** produce `http-identity` and produces **no internet-class** data.

#### The 4 non-internet Name rules (`lame-delegation`, `cname-target-name-error`, `zone-declared-name-returns-name-error`, `resolved-name-absent-from-zone`) — **Implemented**
- Claimed in: `docs/guides/signals.md` §Name signals; `README.md:49-52` ("ships 5… Name-only rules").
- Evidence: real `Eval` at `internal/signal/rules.go:59-194`; fed by the enabled `dns` scan (`db/migrations/18801`), folded by `buildNameFacts`/`composeResolution` (`cmd/web/signals.go:880-1048`). These genuinely fire out of the box.

#### `non-globally-reachable-address-resolved-from-internet` (the claimed 5th Name rule) — **Partial**
- Claimed in: `docs/guides/signals.md:88-97`; counted in README's "5".
- Evidence: rule + IANA classifier real (`rules.go:209-233`, `address.go`). Domain requires `HasInternetVantage`, set only from an `internet`-class value.
- Gap: no internet vantage ships (only `local`/`unverified`, `db/migrations/18800:21`) and no external measurement is dispatched (`cmd/worker/main.go:80-82` "No measurement is dispatched over the connection yet (#8, #14)"). Always `outside-domain`. **README's "5 Name-only rules" overcounts by one — only 4 fire.**

#### The 6 certificate Endpoint rules (`certificate-expired`, `-not-yet-valid`, `-expiring`, `-self-signed`, `-weak-key-or-signature`, `-hostname-san-mismatch`) — **Partial**
- Claimed in: `docs/guides/signals.md:106-135`.
- Evidence: rule logic real + tested (`endpoint.go:154-216`); the `certificate` facet **is** measured (`connectoutcome/certificate.go`, `tls.go:100-117`).
- Gap: `buildEndpointFacts` decodes only `{outcome, chain}` and leaves `CertDetails` **nil by design** (`cmd/web/signals.go:1146-1149, 1221-1230`). No production code computes `Expired/SelfSigned/WeakKeyOrSignature/SANMatchesName/Expiring/NotYetValid` — `CertDetails` is constructed only in tests. Every one of these rules returns `not-evaluable` for a presented cert and `outside-domain` otherwise; none can ever reach `fired`.

#### The 4 HTTP Endpoint rules (`plaintext-http-no-https`, `redirect-does-not-upgrade-to-tls`, `redirect-to-host-outside-estate`, `unauthenticated-request-answered`) — **Partial** (measurer is Stub-unwired)
- Claimed in: `docs/guides/signals.md:136-153`.
- Evidence: rule logic real + tested (`endpoint.go:218-364`). The `httpexchange` measurer is fully built (`internal/measure/httpexchange/`).
- Gap: `http-identity` is **never produced** — `cmd/prober/main.go:54-61` has no `httpexchange.Kind` case (falls to an empty-observation default), and no scan kind dispatches it (the scan-kind CHECK is `hot/cold/tls-acceptance/zone/dns/ct`, `db/migrations/21100:18`; `httpexchange.Run` has no caller). `HTTPResponded` is always false → all four are always `outside-domain`.

#### `sensitive-port-reached-from-internet` (the "flagship") — **Partial**
- Claimed in: `docs/guides/signals.md:165-172`; `docs/spec/v1-spec.md:524-525`.
- Evidence: rule real (`service.go:150-176`); `reachability` measured; sensitive-port table real + wired (`vergecore.IsSensitive`, `signals.go:1125`).
- Gap: predicate needs an internet-class reach leg (`signals.go:1133`) that never exists — same unshipped-vantage gap. The product's flagship signal never fires on a default install.

#### `tls-1.0-accepted` — **Partial** (sharpest single drift)
- Claimed in: `docs/guides/signals.md:160-163`.
- Evidence: rule real (`service.go:112-136`); the `tls-acceptance` facet **is measured weekly and persisted** (`db/migrations/19900:31`, prober handles the kind, spanfold stores it).
- Gap: `buildServiceFacts` **never reads** the `tls-acceptance` facet — it queries only reachability spans and leaves `TLSHandshakeCompleted` false (`cmd/web/signals.go:1086-1088`); no `ListServiceTLSAcceptance` query exists. Data is collected, stored, and then discarded on read, so the rule is always `outside-domain`.

#### Signal verdicts — engine **Implemented**, UI rendering **Partial**
- Claimed in: `docs/guides/signals.md:42-56` and `docs/spec/v1-spec.md:620-622` ("only three are rendered").
- Evidence: all four `Outcome` values are honored engine-side (`signal.go:29-43`, `corpus.go`).
- Gap: the UI retired the per-rule census; the Signals page now paints a flat table of **fired instances only** (`signals.go:34-36`, `deriveSignalInstances` iterates only `c.Fired`, :1310). `not-fired`/`not-evaluable` are computed but never shown. The "three are rendered" language in the guide and spec describes the retired view.

#### Signal CSV export — **Implemented** (`signals.go:202-251`, filter-consistent, `csvSafe` on the asset cell).
#### Signal annotations / acceptance — **Implemented** (`cmd/web/annotations.go`, orphan/withdrawn derived on read, matches ADR-0092).
#### Signal drawer drift/history — **Implemented** (real span join; Tags/CVE/Desc/DetectedBy honestly omitted outside devMode).

#### Curated reference data — **Mixed**
- Sensitive-port list — **Implemented** (`internal/vergecore/verge-core.tsv`, wired).
- IANA special-purpose ranges — **Implemented** (`internal/signal/address.go`, wired).
- Expiry horizon `N` (⅓ validity, ½ if ≤10 days) — **Missing as code**: no production code computes `Expiring`; only a bare field exists (`endpoint.go:38`). The rule described at `signals.md:117-122` is unimplemented.
- Weak-key / deprecated-signature table — **Missing as code**: `WeakKeyOrSignature` is a bare bool never computed in production; the table cited to `docs/research/weak-key-and-signature.md` has no Go folding it into `CertDetails`.

#### Severity — **Doc↔code contradiction**
- Docs say **no** severity: `signals.md:26-30` ("A signal carries no severity… there is no dial to add one"), `v1-spec.md:488-489`, README echoes it.
- Code **has** severity and paints it: `internal/signal/severity.go` defines a 5-level ramp; every rule assigns one; the UI shows a severity badge (`signals.go:820-825`), a severity **filter** (`:448`), severity-**sorted** default order (`:774`), and a severity CSV column (`:247`). `severity.go:12-15` states it supersedes the "no severity" reading (SPEC-CHANGE #1 / P0.1). The guide and v1-spec are stale.

### Discovery

Seeds, custody, the three keyless RIR proposers, crt.sh, source toggling, and scan dispatch are all real and wired. The drift is a cluster of phantom proposer tiles.

#### RIPEstat / RIPE Database / APNIC-registry / LACNIC proposers — **Stub-unwired** (×4, phantom tiles)
- Claimed in: `docs/guides/sources.md` §Proposers table; catalogued `cmd/web/sources.go:88-111`.
- Evidence: each exists only as a catalogue tile with consent-group terms and an admin toggle. No `proposer.Source` implements any of them — `DefaultRegistry` wires exactly three (arin, afrinic, apnic-caida; `proposer.go:84-90`). Worse, `enabledProposers` (`cmd/web/proposals.go:326-338`) filters to `Consent==consentUnencumbered`, so an `operator-accepted` proposer is **never** placed in the enabled map even after an admin accepts its terms and toggles it on.
- Gap: enabling any of the four emits zero proposals; LACNIC has no discovery code at all. The consent dialogs, terms groups, and toggles are UI no discovery path consumes.

#### Consent tiers (proposer side) — **Partial** (dead branch)
- Evidence: the tier gate is real for **sources** (crt.sh terms dialog works, `sources.go:323-326`). For **proposers** it is a dead branch — the only operator-accepted entries are the four no-runner tiles above, which `enabledProposers` drops anyway. Accepting their terms gates access to a capability that does not exist.

#### Implemented in discovery
- Seed name scope (publicsuffix eTLD+1, rejects subdomains) — `seed.go:26-42`, `seeds.go:179-190`.
- Seed address scope (IPv6-safe cap, refuses over-cap with callout) — `seed.go:46-67`, `seeds.go:156-178`.
- Custody extension declaration — `custody.go:23-40`.
- Custody derivation + probe-gating (`MayProbe` reached by hot/cold fan-out) — `internal/custody/*`, `scan/hot.go:58-61`, `cold.go:111-117`.
- crt.sh CT source (SSRF-guarded, throttled 5/min, admits SAN names, ships enabled daily) — `internal/scan/crtsh.go` + `internal/queue/crtsh.go`, `worker/main.go:60-61`.
- ARIN (RDAP), AFRINIC + APNIC (CAIDA⋈delegated-stats) proposers — `proposer/arin.go`, `proposer/caida.go`; all wired.
- Admin-only source toggling (viewer-read, admin-POST, upsert-in-place) — `handlers.go:728-732`.
- Proposal confirm→Seed / decline→exclusion — `proposals.go:233-322`.
- Scan cadence dispatch (per-minute tick, advisory lock, `(scan,scheduled_time)` idempotency) — `queue/queue.go:47-173`.
- On-demand scan trigger (CLI `-trigger` + web admin, 4 guardrails) — `scantrigger.go:86-149`.

#### Honestly-disclosed absences (not drift)
- HackerTarget / Cert Spotter = `Barred` (excluded on terms), non-toggleable — `sources.go:112-121`.
- `operator-credentialed` consent tier — reserved; docs say no v1 source uses it — **Missing (disclosed)**.
- `NoRunner` "catalogued-not-executing" bucket — mechanism exists, zero live entries (disclosed).

### Probing, vantages & exposure

The measurement spine (resolution-walk, wildcard, connect-outcome incl. cert handshake, tls-acceptance), Reach/Exposure pure logic, host-key pinning, and the deploy recipe are all real. The internet-vantage half is hollow.

#### SSH-pushed remote internet vantage — **Stub**
- Claimed in: `README.md:66-69`; `docs/guides/prober.md:24-28`.
- Evidence: the worker's only `Prober` is the local `ExecProber` (`worker/main.go:60`); every job runs locally regardless of `VantageID`. The one SSH client (`worker/vantages.go:166-193`) only `ssh.Dial`s to time latency + pin the host key, then closes — its own comment says "no measurement is dispatched over it here." No `ssh.Session`, no binary upload, no remote exec exists.
- Gap: the prober is never shipped or run on a remote host; an "internet" vantage physically probes from the instance host — the hairpinning the docs warn against.

#### Single `GET /` HTTP probe — **Stub-unwired**; Identifiable User-Agent — **Missing**
- Claimed in: `README.md:35`.
- Evidence: `httpexchange/exchange.go:203-243` is a real single-`GET /` exchanger (redirects off, 64 KB cap) but is never dispatched (see the HTTP signal rules). It also never sets a `User-Agent`, so Go's default `Go-http-client/1.1` would be sent if it ran. No verge-identifiable probe UA is ever emitted. (The identifiable UA at `queue/crtsh.go` is the CT *source* fetcher, not the prober.)

#### `uname` arch check — **Missing**; `SSH_CLIENT` egress observation — **Missing**
- Claimed in: `prober.md` §3.2 (refuse non-Linux/unsupported at provisioning) and §3.3/§5 ("your egress is X… offers it for declaration").
- Evidence: no `uname`/session-exec and no `SSH_CLIENT` read anywhere. `Egress` exists only as a fixture struct field (`probers.go:38`). The documented step-3→step-5 flow that "unlocks exposure" cannot occur.

#### Vantage registration UI — **Partial**
- Evidence: provisioning (`ParseEndpoint`+insert), keypair generation, and pin-status are real (`probers.go:51-85`, `vantages.go:39-97`). But the card's `HostKeyFingerprint`, `Platform`, and `Egress` chips are never populated from live data — fixture-only (`probers.go:31-40`).

#### Exposure two-leg construction — **Partial**
- Claimed in: `first-run.md:42-56` ("exists only where both legs hold a value").
- Evidence: `internal/exposure/exposure.go` fully and correctly implements the 2×2 composition and is hermetically tested (`Constructible = internetPresent && internalPresent`, :302).
- Gap: the internet leg can never be genuinely populated (no remote push + no egress confirmation), so end-to-end "exposed/firewalled" from a real outside observer is unreachable. The math is real; the data source is not wired.

#### Implemented in probing
- Prober stdin/stdout NDJSON contract — `cmd/prober/main.go:25-62`, `queue/worker.go:35-57`.
- TCP connect never SYN (`net.Dialer.DialContext`, closes immediately) — `connectoutcome/leaf.go:115-138`.
- Rate-limiting (per-host + global Pacer/Backoff, adaptive) — `connectoutcome/safety.go`, `run.go:78-81`.
- Unprivileged (connect needs no cap; recipe runs `65532`, `cap_drop: ALL`) — `deploy/prober/docker-compose.yml:15-19`.
- Local-exec internal vantage — `worker/main.go:60`, `worker.go:40-47`.
- Host-key pin (TOFU, hard-fail on change) — `vantage/hostkey.go:41-77` (exercised only on the latency dial).
- Reach computation (fold connect→reached/not-reached, existential `ComposeReach`, blanket→Gap) — `connectoutcome/leaf.go:71-98`, `exposure.go:71-87`.
- TLS/certificate measurer (real `crypto/tls`+`x509`, rides reachability) — `connectoutcome/tls.go`, `certificate.go:131`.
- tls-acceptance weekly enumeration — `measure/tlsacceptance/enumerate.go`.
- DNS resolution-walk + wildcard-discrimination — `measure/resolutionwalk`, `wildcarddiscrim`.
- `deploy/prober/` recipe (persistent ed25519 host key, pubkey-only, `restrict`, `from=`) — complete + hardened (serves only latency-ping/pin today).

### Drift, coverage & reading surfaces

The drift engine core and every reading surface are real and bind to live DB data in production (all fixtures are strictly `VERGE_DEV`-gated pixel-harness paths, never served to a real estate). Two real gaps.

#### Transition grammar (`appeared`/`returned`/`revealed`/`withdrawn`/`descoped`) — **Partial**
- Claimed in: `README.md:40`; `reading-the-estate.md` §Drift.
- Evidence: the grammar is fully coded (`drift/transition.go`, classified in `driftfeed.go:82-151`); `appeared` and `changed` fire for real.
- Gap: `returned`/`revealed`/`withdrawn`/`descoped` never fire — they need a span closed **with a closure reason**, and no production path writes one (`foldObservationsIntoSpans` calls `CloseSpan` with no reason, `spanfold.go:96`; `drift.OpeningKind` has zero production callers). The code admits it (`driftfeed.go:34-40`). The six-word legend nonetheless always renders (`drift.go:82-89`), showing words that can never appear.

#### Estate membership / withdrawal package — **Stub-unwired** (root cause of the above)
- Evidence: `internal/estate/{membership,withdrawal,address}.go` are fully implemented and unit-tested, but **no production file imports `internal/estate`** (grep of the import path = only its own tests). The web layer derives Name membership on read via a local `suppressesNameMembership` (`subjects.go:66-68`) instead. The "subject left the estate" composition the dormant transition kinds depend on is never invoked at ingest.

#### Coverage computation — **Partial**
- Evidence: `coveragePage` binds real reads for seeds, zones, blanketed-reach gaps, unavailable vantages, and unevaluable rules (`cmd/web/cold.go:188-390`).
- Gap: (a) the address-scope `counted/total` meter has **no live numerator** (`cold.go:280-291`; the counted/total form appears only under the devMode fixture); (b) the per-zone stale callout is hardcoded empty (`"StaleZones": nil`, `cold.go:259`).

#### Implemented in drift/reading
- Drift timeline fold (ingest → feed) — `drift/fold.go`, `queue/spanfold.go:35-114`, `driftfeed.go`.
- Span comparability / Break rule (new span on unequal Derivation vector even if value-identical; `changed` suppressed across a Break) — `drift.go:77-88`, `spanfold.go:74-78`, `driftfeed.go:118-120`.
- Gap surfacing (tagged at fold; rendered on Coverage/Inventory/drift-diff) — `spanfold.go:167-180`, `inventory.go:184`.
- Inventory, Graph, Search, Dashboard, Subject detail, Asset detail, Run detail, Scans monitor — all **Implemented**, binding to real open-span / dispatch / signal corpora (`inventory.go:506`, `graph.go:414`, `search.go:238`, `auth.go:735`, `subjects.go:357/669/1260`, `scans.go:234/93`). Dashboard KPIs use `Has*` flags → em-dash, never a fabricated zero.

### Notifications, delivery & reports

The channel config, delivery transport, report scheduler, artifact renderer, and PDF engine are all real and worker-wired. The subsystem is fatally gated at its **source**: nothing generates a `Message`.

#### Message generation from signal/drift transitions — **Missing** (load-bearing)
- Claimed in: `docs/guides/notification-channels.md` §"What fires" ("A message is computed once at the cause"); `docs/spec/notification-channels.md` §7.
- Evidence: the pure constructors (`message/flagship.go:64`, `membership.go:56`, `narrowing.go:67`) and `InsertMessage` (`db/messages.sql.go:62`) exist and are unit-tested, but **no production code calls any of them** (grep outside `_test.go` = nothing; the two live `NarrowingReceipt` uses are UI previews, not persisted firings).
- Gap: nothing wires a fold/transition to writing a Message. In a real deployment the `message` table is never populated, so no message is generated, shown in the Inbox, or routed anywhere. The read/render side (Inbox, read-state, delivery views) is fully built but has no producer.

#### Worker POST-to-channel delivery — **Stub** (complete but unwired at source)
- Evidence: the transport is production-grade and worker-driven — `delivery/runner.go` (SSRF-guarded doer, `SendSigned`, HMAC-SHA256 signing, retry/dead-letter). But the only enqueue function, `EnqueueForMessage` (`runner.go:116`), has **zero callers** and depends on a Message that never exists. The delivery loop runs every 5 s against a queue nothing ever fills — no POST is ever sent in a real deployment.
- Related mechanisms, all real but never exercised for the same reason: **Delivery-as-record + idempotency** (`ON CONFLICT … DO NOTHING`, `FOR UPDATE SKIP LOCKED`) and **retry / dead-letter** (5 attempts, exponential, `Decide`/`RetryDelivery`) — both **Implemented (unexercised)**.

#### Worker report-cadence dispatch — **Partial** (cadence precision)
- Evidence: the dispatcher polls each minute, locks per-schedule, and inserts one receipt per `(schedule, tick)` idempotently (`report/dispatcher.go:46-127`).
- Gap: `CadenceWindow` (`report/cadence.go:27`) extracts only a duration and floors to **Unix-epoch boundaries** — the presets' declared times ("Daily · 08:00", "Weekly · mon 09:00") are ignored, and a "Custom…" cron string is **not interpreted** (falls through to the weekly window, `:36`). Runs fire epoch-aligned, not at the declared hour/day; cron cadence is declared-but-not-honored.

#### Off-instance report delivery (link-only "report-ready") — **Implemented**, but docs say it is *not built* (reverse drift)
- Claimed absent in: `docs/guides/reports.md` §"Status, read this first" ("What is not built is off-instance delivery… the wizard captures no recipient/channel field… collision #17 AWAITING DESIGN").
- Evidence: it is in fact built and wired — the wizard has a Delivery step with a channel select (`reports_schedule.go:558-582`), `channel_id` is persisted on create/edit (`:365`, `:423`), the dispatcher enqueues one link-only `report-ready` notification per bound run (`dispatcher.go:142`), and `notify.go` POSTs it via `delivery.SendSigned` and flips the receipt to `delivered`. The guide's status note, the `delivery_target` "Not yet wired" row, and the Run-now code comment all describe an earlier state.
- Asymmetry (real): **Run-now never notifies** — `runReportScheduleNow` inserts a `generated` receipt and never enqueues a notification even when a channel is bound; only the on-cadence dispatcher sends.

#### `/reports/export?format=pdf` — **Undocumented**
- Evidence: `reports_export.go:59` accepts `format=pdf` and serves a real PDF (`:120`), yet `reports.md` documents only `csv`/`json` and the handler's own header comment says "PDF is deliberately NOT offered here" (`:27-33`). Works today; both the guide and the comment are stale.

#### Implemented in notifications/reports
- Channel declaration (https-only, loopback-http exception, IP-literal SSRF refusal, write-only secret, admin-only) — `settings.go:449-597`, `validateChannelURL:1235`.
- Report schedule create/edit/delete (4-step wizard, per-step gates, newest-first list) — `reports_schedule.go:328-499`.
- Report generation (recomputes figures from receipt period bounds via real ledger reads) — `reports.go:1099-1180`, `dispatcher.go:127`.
- Delivered-report artifact surface (`/reports/delivery`, `/pdf`, empty-state, period-dated filename) — `reports.go:976-1063`. *(Minor stale seam: the footer host reads the legacy empty `DeliveryTarget`, so a bound report shows no destination host.)*
- Report PDF generation — **real pure-Go `go-pdf/fpdf` renderer** (`message/pdf.go:250`), not HTML-only. Refutes the obvious phantom.
- Reports analytics regions (real scans-per-day heatmap; "by severity"/"over time" honest empty-states, never fabricated series) — `reports_export.go:88-225`.

### Accounts, authentication & SSO

Almost entirely real end-to-end — 16 of 17 features Implemented, including the parts most likely to be scaffolding.

#### API-token authentication — **Missing** (honestly disclosed)
- Claimed in: `authentication.md` §"API tokens" — which itself states the gap ("no request-authentication path consumes one yet").
- Evidence: tokens mint/list/revoke work (`auth.go:2031-2092`), but no inbound path resolves a `vg_pat_…` bearer (grep for `Authorization`/`Bearer`/`vg_pat_` finds only the *outbound* delivery assertion that no bearer is sent). `last_used_at` is read but never written, so "Last used" is always an em-dash. Not phantom drift — the guide discloses it — but there is no way to call an API with a token today.

#### Password length — **doc drift**
- Evidence: `accounts.md` and `authentication.md` say passwords are "8–72"; `validatePassword` (`auth.go:2239-2248`) enforces a **12**-char floor (per SPEC-CHANGE #19d). The docs are stale → should read "12–72".

#### Implemented in auth (selected, all real + wired)
- First-run setup token (256-bit, single-use by window-close, `VERGE_SETUP_TOKEN` pin) — `main.go:176-193`, `auth.go:141-191`.
- Admin/viewer role **enforcement** — `requireAdmin` is genuine authorization (role read live per request), mounted on every mutating route (`auth.go:108-116`, `handlers.go:777-820`). Not a stored-but-ignored role.
- Invites + full lifecycle, role change (last-admin guard), account removal (self/typed-name/last-admin/FK guards) — `settings.go:292-442`.
- TOTP enrollment (160-bit, encrypted at rest, in-proc QR, 8 recovery codes) + login challenge (per-account+IP lockout, **atomic single-use step**, recovery fallback) — real RFC 6238 (`internal/auth/totp.go`, `auth.go:1199-505`).
- 2FA re-enroll (admin), API token create/revoke, personal + admin session listing/revocation (server-side registry), password change (revoke-others), password reset (enum-safe, single-use, out-of-band by design) — real.
- OIDC SSO: provider config (https-only, generic OIDC) + login/callback (**genuine `go-oidc`/`oauth2`: discovery, PKCE, nonce, id_token verification, sub-based binding**, TOTP still enforced after SSO, unbound identity refused) + identity link/unlink — `settings_sso.go`, `sso.go:98-484`.
- Minor: `setAccountRole` lacks the explicit self-row refusal `removeAccount` has; `accounts.md`'s "cannot change your own role" is UI-only for the non-last-admin case.

### Integrations & ops

Ops (retention, migrations, healthchecks, worker scaling, backup docs) is solid. Integrations are eight phantom tiles.

#### Integration tiles — Slack, PagerDuty, Teams, Jira, Linear, Splunk, Elastic, S3 export — **Partial** (×8)
- Claimed in: `docs/guides/integrations.md` — each tile with a concrete promise ("Signals and drift summaries as formatted messages", "Critical signals open incidents", "Nightly NDJSON snapshots to your bucket", etc.); state table: `installed` = "Installed and delivering."
- Evidence: the install-declaration state machine is real and persisted (`integrations.go:262-292`, admin-gated, unknown-slug→400).
- Gap: **nothing consumes an installed row.** No Slack/PagerDuty/Jira/Linear/Splunk/Elastic client, no card/issue/HEC/ECS formatting, no S3 client or nightly job exists; the delivery worker is integration-agnostic and never reads `integration_state`. Install writes a row and renders a badge — nothing is ever delivered, formatted, or exported. The author admits it (`integrations.go:22-29`). **"What an installed integration does downstream" is Missing**; the state-table adjectives "delivering"/"deliveries resume" contradict both the guide's own thesis ("an integration is never a delivery channel") and the code.
- `needs-config` tile state — **Stub** (unreachable: no code path ever writes it; display-vocabulary only). "Send test" (`testIntegration`) — **Stub** (no-op that validates the id and redirects).

#### Install / disconnect transitions — **Implemented**
- Evidence: `handlers.go:815-820` (admin-gated), real upsert/delete (`integrations.go:262-292`, `db/integrations.sql.go:53`), PRG redirect. The state *machine* is genuinely end-to-end; only the *effect* of being installed is absent.

#### Implemented in ops
- Retention — dispatch pruning (`dispatch_cadence_multiple`, floor 2 cadences, worker sweeper every 24 h, real `DELETE`) and observation pruning (`observation_currency_days`, per-timeline `DELETE` with live-tier protection) — `internal/retention/*`, `worker/main.go:132-149`. Ship unbounded (inert by default), as documented.
- Backup / restore — **documented operator procedure, no code** (correct): `backup-and-restore.md` is pure `pg_dump`/`pg_restore`/volume-tar against the real service/volume names; no phantom "backup button" is implied.
- Migration-on-boot (web runs `goose.Up` before binding; worker migrates nothing) — `main.go:56-58, 205-220`.
- Healthchecks (`-healthcheck` flags; web GETs `/healthz` which does a real DB heartbeat; worker opens a Postgres pool) — `main.go` both binaries, `Dockerfile:36-46`. *(Minor: `docker-compose.yml` defines a `healthcheck:` block only for postgres; web/worker rely on the image `HEALTHCHECK`, so the guide's "compose runs on an interval" is satisfied by the Dockerfile directive, not a compose entry.)*
- `POSTGRES_PASSWORD` required-not-defaulted (`${POSTGRES_PASSWORD:?}` in all three services; `.env.example` ships it empty) — `docker-compose.yml:14,39,55`.
- Worker job-claim locking / multi-worker scaling (`FOR UPDATE SKIP LOCKED` on every queue; advisory-lock + unique-key dispatch fan-out; byte-identical workers) — `db/queries/*.sql`, `queue/queue.go:113-140`.
- Worker self-test (execs the prober with a noop job at startup, non-fatal) — `worker/main.go:90-182`, `selftest.go`.

---

## Recommended doc changes

Edits that make the docs match the code today.

1. **`docs/guides/signals.md`** — this is the highest-priority doc fix. Add a per-rule status column or a prominent banner: **only the 4 non-internet Name rules fire on a default install.** The other 13 are present but dormant. Mark the 6 certificate rules and `tls-1.0-accepted` as "rule shipped; measured-fact fold pending", the 4 HTTP rules as "measurer not yet dispatched", and the 2 internet-gated rules (`non-globally-reachable…`, `sensitive-port-reached-from-internet`) as "requires an internet vantage (not yet shipped)".
2. **`README.md:49-52`** — change "ships 5 of them (the Name-only rules)" to **4** (`non-globally-reachable-address-resolved-from-internet` needs an internet vantage and cannot fire).
3. **Reconcile the severity contradiction.** Either update `docs/guides/signals.md:26-30`, `docs/spec/v1-spec.md:488-489`, and the README to describe the shipped severity ramp, or (if "no severity" is still the intended design) treat the shipped severity UI as the drift. Per `severity.go:12-15` the ruling already favors severity — so the docs are the stale side and should be updated.
4. **`docs/guides/signals.md:42-56` and `docs/spec/v1-spec.md:620-622`** — the "three verdicts are rendered" census view was retired; the UI shows fired instances only. Update to describe the shipped instance table.
5. **`docs/guides/integrations.md`** — mark all 8 tiles as "install records intent; delivery/formatting is not yet built" and remove the state-table adjectives "Installed and delivering" / "deliveries resume". Consider dropping `needs-config` from the documented states (unreachable) until a config flow exists.
6. **`docs/guides/sources.md`** — mark RIPEstat, RIPE Database, APNIC-registry, and LACNIC as "catalogued, not yet functional" (no runner; enabling emits no proposals), or remove the tiles until a runner exists. Note that the `operator-accepted` consent tier currently gates nothing runnable.
7. **`docs/guides/prober.md`** — mark the SSH binary-push / remote-exec, the `uname` arch check, and the `SSH_CLIENT` egress read as "planned" — today SSH is used only to measure latency and pin the host key. Note that an "internet" vantage does not yet probe from the remote host.
8. **`README.md:35`** — the "identifiable User-Agent" and "one `GET /` per endpoint" claims describe the (unwired) HTTP leaf; qualify them as pending until `http-identity` is dispatched.
9. **`docs/guides/notification-channels.md`** — add a status banner that message generation is not yet wired, so no channel delivery fires end-to-end today (the channel/delivery machinery is built and waiting on the producer).
10. **`docs/guides/reports.md`** — **remove the "off-instance delivery is not built / collision #17 AWAITING DESIGN" status note** (it is built and wired); instead document the channel-select Delivery step, and note that **Run-now is download-only** (only the scheduled tick notifies). Correct the cadence description: runs fire on epoch-aligned windows, and preset times / custom cron are not yet honored. Document `format=pdf` on `/reports/export` (or remove the endpoint's PDF branch) and fix the stale header comment.
11. **`docs/guides/authentication.md` and `docs/guides/accounts.md`** — change the password bound from "8–72" to **"12–72"**.

## Recommended code changes (small, high-leverage)

- **`enabledProposers`** (`cmd/web/proposals.go:333`) excludes every `operator-accepted` proposer — but since none of those four has a runner anyway, the fix is to remove the tiles, not the filter. If RIR proposers are wanted, they need real `proposer.Source` implementations wired into `DefaultRegistry`.
- **`tls-1.0-accepted` is one query away from working**: the `tls-acceptance` facet is already measured and persisted; `buildServiceFacts` (`cmd/web/signals.go:1086`) just never reads it. Adding a `ListServiceTLSAcceptance` query + populating `TLSHandshakeCompleted` would light up this rule with no new measurement.
- **Certificate rules**: `not_after` and the chain are already in the `certificate` observation; a `CertDetails` constructor in `buildEndpointFacts` would move the 6 cert rules from always-`not-evaluable` to live (expiry/self-signed/SAN are computable from data on hand today).
- **`internal/estate`** is complete and tested but imported by zero production code — wiring `CloseWithdrawal`/membership into `spanfold` closure would light up the four dormant transition kinds.

## Coverage

- **Claim sources audited:** `README.md`, `docs/spec/v1-spec.md`, `docs/spec/notification-channels.md`, `docs/spec/measurement-offers.md`, `docs/spec/curated-table-watch.md`, `docs/spec/packaging-and-configuration.md`, all 23 `docs/guides/*.md`, the full `cmd/web` route table, `docker-compose.yml`, `Dockerfile`, `.env.example`, and the relevant `internal/*` packages and `db/migrations`.
- **Claim sources skipped:** `CONTEXT.md` (145 KB) was used as background domain reference, not exhaustively feature-mapped — its content is upstream of the spec/guides, which were audited directly. `docs/adr/`, `docs/research/`, and `docs/correspondence/` were consulted only where a guide cited them. The `design-system/` package (visual layer) was out of scope for functional drift.
- **Subsystems not separately audited:** the `web/` embedded front-end bundle (25 MB) was treated via its server-side handlers and templates, not as an independent claim source.
- **Unverifiable items:** 0 — every classification is backed by a file:line trace. A handful of "Implemented (unexercised)" mechanisms (delivery idempotency/retry) are real in code but cannot be exercised end-to-end today because their upstream producer is missing; these are noted as such rather than marked Unverifiable.
- **Method:** static reading + grep across the Go tree, entry-point-inward tracing from routes/CLI/scan-kinds to the code that does work, verified per the evidence ladder. Fanned out across 7 parallel subsystem passes; findings cross-checked where subsystems intersect (notably: the unshipped internet vantage and the unwired `http-identity` measurer each independently explain a cluster of dormant signal rules).
