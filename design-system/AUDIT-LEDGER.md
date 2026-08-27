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
| AL-6 | 6 certificate rules always `not-evaluable` (CertDetails nil by design) | CHARTED | P0.10 |
| AL-7 | 4 HTTP rules dormant — `http-identity` measurer built but never dispatched; no User-Agent | CHARTED | P0.11 + E7 |
| AL-8 | Drift `returned`/`withdrawn`/`descoped` + TransitionDelta chip fixture-only; `internal/estate` unwired | LANDED (wiring) + CHARTED (chip) | Wiring landed in #637/51141e8 (collision #36 confirmed; audit's "unwired" was stale). Chip residual = P0.12 re-scoped by #36: vs previous equal-length window, suppress without one |
| AL-9 | Coverage counted/total meter has no live numerator; per-zone stale callout hardcoded nil | CHARTED | P0.13 (meters ruled in #19c) |
| AL-10 | 8 integration tiles: install state real, downstream effect missing; "Send test" no-op | RULED (partial) | #26j catalogue-PRG ruling; label fix = E13; Send-test affordance = round-3 verify item (collision if specced) |
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
