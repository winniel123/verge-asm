# Feature Inventory — verge-asm spec-drift audit — 2026-08-26

Claim sources: README.md, CONTEXT.md, docs/spec/v1-spec.md, docs/spec/*, docs/guides/*.md (23 guides).
Authoritative order (per skill default): spec = intent, README/guides = promise, code = truth.

Verification farmed to 7 parallel subsystem auditors (results pending). Subsystems:

1. **Signals + measurement feed** (agent A) — 17 rules, verdicts, CSV export, annotations, curated tables, severity contradiction.
2. **Discovery** (agent B) — seeds (name/address scope), custody+probe-gating, each source/proposer (crt.sh, RDAP/RIR, etc.), consent tiers, admin toggle, scan cadence dispatch, on-demand trigger.
3. **Probing/vantages/exposure** (agent C) — prober NDJSON contract, probe-safety claims (TCP-connect/UA/single-GET/rate-limit/unprivileged), local + SSH-pushed vantage, host-key pin, Exposure two-leg.
4. **Drift/coverage/estate/reading surfaces** (agent D) — drift timeline, transition grammar, Span comparability, Coverage/Gap, inventory/graph/search/dashboard/subject/asset/run pages (fixtures vs real).
5. **Notifications + reports** (agent E) — message generation, channel declaration, worker POST delivery, idempotency/retry, report schedule, cadence dispatch, artifact, PDF.
6. **Accounts/auth/SSO** (agent F) — setup token, roles+enforcement, invites, TOTP, API tokens, sessions, password change/reset, OIDC SSO + linking.
7. **Integrations + ops** (agent G) — each integration tile + downstream effect, retention pruning, backup/restore, migration-on-boot, healthchecks, worker job-claim/scaling, self-test.

## Cross-cutting facts established directly (not delegated)

- **17 signal rules exist as real Eval() logic** (internal/signal/rules.go=5 Name, endpoint.go=10, service.go=2); all 17 run by EvaluateCorpus (corpus.go:16-31). README claims "5 of 17 ship (Name-only)". Open Q (agent A): do endpoint/service rules receive measured cert/http/tls/exposure facts in production, or are they Partial (logic real, no input)?
- **Route surface** (cmd/web/handlers.go:477-850) maps cleanly to guides; no obvious phantom top-level nav. Dev-only routes gated (GET /dev/*).
- **Scan tiers** real in code: cold.go (full-range, opt-in, ships disabled), plus hot/zone/dns/tls-acceptance per spec §3.4. `verge-core` = union of frequency-set ∪ sensitive-list (spec §3.5).
- **Facets** named in code (internal/): resolution, dns-record, certificate, http-identity, tls-acceptance (+ reachability implied) — matches README "six facets".
- **Severity contradiction lead**: signals.md/README say signals "carry NO severity", yet internal/signal/severity.go + every rule has Severity()→SevMedium/High/Critical. Agent A verifying whether surfaced in UI.
