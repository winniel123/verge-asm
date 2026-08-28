# Audit ledger — settled findings register

Written 2026-08-27 from the spec-drift audits of 2026-08-26 and 2026-08-27. This file travels with the package into `design-system/` and is the memory between audits.

**For any future spec/drift audit (and for Wayfinder):** the spec-drift-audit skill reads this file in its Step 1 and applies its ledger rule in Step 3 — consult this ledger BEFORE reporting. A finding that matches a row here is **not a new finding** — cite it as `known (AL-nn)` with one line, and report only (a) genuinely new drift, (b) a row whose Status changed on the ground (e.g. a CHARTED item landed, or a RULED premise no longer holds). Add the instruction "diff findings against design-system/AUDIT-LEDGER.md first" to every audit prompt. When a charted item lands or a new ruling supersedes a row, update the row — the ledger is append-and-amend, never silently pruned.

Status vocabulary:
- **RULED** — decided by design/owner; recorded in SPEC-CHANGE.md (#n). Not drift. Re-reporting it is audit noise.
- **CHARTED** — real, acknowledged gap with a work item in PARITY-CHART.md (P0.x/E-n/W-n). Audits may report progress against it, not rediscover it.
- **BY DESIGN** — intentional behavior, documented; often looks like drift from outside.
- **LANDED** — audit-verified built; listed so stale docs/comments don't resurrect it.

| AL | Finding (audit phrasing) | Status | Where decided / charted |
| --- | --- | --- | --- |
| AL-1 | Severity doc↔code contradiction (docs say none; code ships 5-level ramp) | RULED | #1/P0.1 + #28: ramp is normative, docs are the stale side; fix = E3 |
| AL-2 | Message generation missing → Inbox structurally empty, no channel delivery fires | CHARTED | P0.7 (constructors/transport exist; only the wire is absent) |
| AL-3 | Internet vantage hollow — SSH pins key + times latency, never pushes/execs; jobs hairpin on instance host | CHARTED | P0.8 (ADR-0103 stands; VantageCard chips are declared fixture stubs until then) |
| AL-4 | Only 4 of 17 signals fire on a default install (README says 5) | CHARTED | E1/E2 docs now; rules go live via P0.8–P0.11 |
| AL-5 | `tls-1.0-accepted` — facet measured + persisted, never read back | CHARTED | P0.9 |
| AL-6 | 6 certificate rules always `not-evaluable` (CertDetails nil by design) | CHARTED (split) | #37: CertDetails goes per-attribute readable. P0.10a lights expired+expiring (not_after) now; P0.10b (CertVersion v3 leaf + policies) lights the other 4. SANMatchesName never defaults false — absent SANs = not-evaluable, never a mismatch |
| AL-7 | 4 HTTP rules dormant — `http-identity` measurer built but never dispatched; no User-Agent | CHARTED | P0.11 + E7 |
| AL-8 | Drift `returned`/`withdrawn`/`descoped` + TransitionDelta chip fixture-only; `internal/estate` unwired | LANDED (wiring) + CHARTED (chip) | Wiring landed in #637/51141e8 (collision #36 confirmed; audit's "unwired" was stale). Chip residual = P0.12 re-scoped by #36: vs previous equal-length window, suppress without one |
| AL-9 | Coverage counted/total meter has no live numerator; per-zone stale callout hardcoded nil | CHARTED | P0.13 (meters ruled in #19c) |
| AL-10 | 8 integration tiles: install state real, downstream effect missing; "Send test" no-op | RULED + CHARTED | #26j catalogue-PRG (label = E13). "Send test" ruled #38 → refined #39: the no-op had no target because an integration holds none; #39 adds a nullable integration→channel binding (drawer selector, #17 pattern) and gates Send test on it — posts via the bound Channel's SendSigned, honest-disabled when unbound. Design landed v3.23.0; repo wires P0.14 |
| AL-11 | 4 RIR proposer tiles runnerless (RIPEstat, RIPE DB, APNIC-registry, LACNIC) | RULED | #30 — catalogued-not-executing bucket; docs = E8 |
| AL-12 | `enabledProposers` gates on `unencumbered` only — would suppress future accepted runners | CHARTED | W1 (fix rides the first real RIR runner) |
| AL-13 | Reports Run-now never notifies (download-only receipt) | BY DESIGN | #29 — operator is present; only cadence ticks deliver |
| AL-14 | `/reports/export?format=pdf` undocumented | RULED | #29 — spec-normative (#23c); document it (E10) |
| AL-15 | Schedule cadence presets/cron epoch-floored | RULED → likely LANDED | #31 BUILD; audit reads `DispatchTick`→`prevFire` as honoring both — round-3 verify closes it |
| AL-16 | Reports dispatch/delivery backend "missing" (#290/#291) | LANDED | P0.6/#497 shipped; stale comments = E10 |
| AL-17 | Report notify: link-only POST, no estate in body | RULED + LANDED | #16/P0.6b (ADR-0039 stands) — audit confirms shipped |
| AL-18 | API tokens / bearer auth missing | LANDED | #390 (v3.18.0 spec) — `resolveAPIToken` real; off-by-default `api_enabled` is BY DESIGN |
| AL-19 | Backup/restore "documented procedure, no code" | LANDED | #391 (v3.18.0 spec) — NDJSON archive, preflight + typed-confirm restore, key regen |
| AL-20 | Backup "data-only, no secrets" overpromises (archive carries password/TOTP/token hashes, SSO + channel secrets) | CHARTED | R3-D1 (design copy) + E12 (doc lead); caveat already disclosed in code + doc body |
| AL-21 | Password docs "8–72" vs enforced 12-char floor | RULED | #19d — 12+ is the spec; docs = E9 |
| AL-22 | Signals page renders fired instances only; guide's "three verdicts" census is retired | RULED | P2.2 flat table is normative (tests assert no census); docs = E4 |
| AL-23 | Org switcher nullable-pending-store | RULED | #33 — RETIRED permanently (ADR-0073); static chip only |
| AL-24 | Stale comments: reports.go "no backend", graph.go "apex not wired" | CHARTED | E10/E11 — code is correct, comments lie |
| AL-25 | Populated Inbox / drift fixtures only behind `VERGE_DEV` | BY DESIGN | devMode set once from env (main.go:118); real deployments never serve fixtures — honesty discipline, not drift |
| AL-26 | Dashboard tiles degrade to "—" via `Has*` flags, never fabricated zeros | BY DESIGN | Spec's own empty pattern (Acceptance §7) |
| AL-27 | SMTP / estate-bearing email absent | RULED | ADR-0039 + #16 — estate never leaves the instance; notify-with-link only |
| AL-28 | Four IdP SSO guides rest on one generic OIDC connector | BY DESIGN | Matches docs; SAML retired per #26e/ADR-0113 |
| AL-29 | Only a generic signed-webhook channel type exists | BY DESIGN | Spec says so; vendor formatting layers are AL-10's future work |

Amendment log:
- 2026-08-27 — created (29 rows) from the 2026-08-27 re-run audit; supersedes re-reporting of both prior audits' findings.
- 2026-08-27 — AL-8 amended (estate wiring LANDED #637; chip = P0.12). AL-6 split (#37). AL-10 amended (#38→#39).
- 2026-08-27 — added AL-30–AL-40 from the dogfood session (round 4). Design-side findings landed in the UI kit + spec; repo-side charted as R4-R1…R4-R5. AL-30 carries the open design ruling R4-Q1.

| AL-30 | Cloudflare/WAF-fronted domain floods inventory with non-actionable edge-shared spans | CHARTED + OPEN RULING | Repo R4-R1 (classify/suppress provider-fronted addresses); surfacing UI blocked on ruling R4-Q1 |
| AL-31 | Deleting a seed returns an internal server error | CHARTED | Repo R4-R2 (handler/tx bug + regression) |
| AL-32 | CT scans keep retrying, never settle | CHARTED | Repo R4-R3 (retry/terminal-state logic) |
| AL-33 | Export graph → PNG produces an image unlike the graph | CHARTED | Repo R4-R4 (export renderer) |
| AL-34 | Org-Discovery CIDRs can't be hot-scanned / never scan | CHARTED | Repo R4-R5 (scan dispatch eligibility for org-sourced ranges) |
| AL-35 | Large inventory = endless scroll | LANDED (design) + CHARTED (repo) | R4-D1: sticky controls, collapsible counted groups, 25-row cap + show-all in UI kit + spec; repo pages/windows the served rows |
| AL-36 | No Test action on the Channels settings page | LANDED (design) + CHARTED (repo) | R4-D2: per-channel Send test (always testable — channel holds its URL); repo wires SendSigned |
| AL-37 | "Scan running" not visible across all views | LANDED (design) + CHARTED (repo) | R4-D3: persistent pulsing pill in global TopNav → Scans; repo shell reads the in-flight flag |
| AL-38 | Activity heatmap missing boxes for zero-activity days | LANDED (design) + CHARTED (repo) | R4-D4: bordered empty cell (contrast fix; component already mapped all days); repo emits one cell/day incl zeros |
| AL-39 | Dashboard Coverage card text overflows | LANDED (design) | R4-D5: label ellipsis + min-width:0, staleness line wraps |
| AL-40 | Settings nav shifts width between tabs | LANDED (design, confirmed fixed rail) + CHARTED (repo, cosmetic) | R4-D6: rail already fixed 210px; residual = page scrollbar toggling → repo `scrollbar-gutter: stable` |
| AL-41 | Each job should stream its exact output | LANDED (design) + CHARTED (repo) | R4-D7: RunDetail streams per-job stdout live (LogViewer live); repo streams real per-job output |

- 2026-08-28 — v3.24.1 template export: R4-Q1 RULED (#762) → AL-30 amended; graph PNG fix landed → AL-33 amended; AL-35/36/37/40/41 design half now in the served templates too (were UI-kit-only in v3.24.0).
