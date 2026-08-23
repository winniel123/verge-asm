---
name: security-audit
description: Whole-repo security audit that hunts for vulnerabilities across the codebase, infra, dependencies, and CI, then emits a Wayfinder map (chart) of confirmed-finding and go-deeper tickets for another session to finish. Use when asked to audit the repo for vulnerabilities/exploits, run a security audit, or produce a security posture chart. Not for reviewing a diff or branch changes — that is /security-review.
user-invocable: true
---

# Security audit

One run **hunts** for real vulnerabilities, **confirms** what it can, and leaves a **Wayfinder map** that is both a findings report and a handoff frontier: confirmed vulns become remediation tickets, surfaces you couldn't finish become go-deeper tickets, and the next session picks up the frontier cold. This is a whole-repo standing audit — distinct from `/security-review`, which reviews pending branch changes.

Audits run AFK. Rule on the recommendations here and justify in the map body rather than stopping to ask.

## The run, in order

### 1. Orient

- Read `CONTEXT.md` and the ADRs under `docs/adr/` that touch auth, scanning, delivery, or deployment. Name findings in the glossary's vocabulary.
- Check for an existing open audit map: a `wayfinder:map` issue titled `Security audit — …`. If one exists, you will **supersede** it in step 5 (default) — note its number now.
- Fix the surface list from **Scope & categories** below against the current tree. If a category's entry-point paths have moved, follow the code, not the cache.

### 2. Scan

Run the cheap deterministic tools through Docker where the image is present, and **degrade gracefully** — a missing tool is a note in the map's Scan-coverage line, never a failed run:

- `govulncheck ./...` — known-vuln deps + reachable call paths (CI runs none of these, so this is signal, not duplication).
- `gosec ./...` — Go SAST.
- `gitleaks detect` / `trufflehog` over the **full git history**, not just the worktree.
- `trivy` over the built image (`Dockerfile`, `deploy/prober/Dockerfile`) for OS/base-image CVEs.

Tool output **seeds** the hunt; it does not replace it. The findings that matter most here — SSRF in request builders, authz gaps — are ones scanners miss.

### 3. Hunt

Fan out **one bounded pass** of surface-scoped hunters, roughly one per category in **Scope & categories**. Each hunter reads its lane, reasons about the threat classes named for it, and returns structured candidate findings (file:line + a failure scenario). Prefer a Workflow or parallel sub-agents so lanes run concurrently and their file dumps stay out of your context.

Bounded means bounded: a hunter that runs out of budget on a large surface returns what it confirmed **and** flags the surface as time-boxed. Time-boxed surfaces become audit tickets in step 5 — that is the tail-capture mechanism, so a single run does not need to be exhaustive. It needs to be honest about what it did and didn't finish.

### 4. Confirm

Every candidate finding passes an **adversarial confirm gate** before it can become a finding ticket: a skeptic sub-agent tries to *refute* it, defaulting to refuted when uncertain.

- A **finding ticket** requires a concrete failure scenario **and** a file:line. No scenario ⇒ not a finding.
- Survives refutation ⇒ label `confirmed`. Plausible but unrefuted ⇒ still files, labeled `plausible`. Refuted-but-interesting ⇒ downgrade to an **audit ticket** (“this looked suspicious, investigate”), not a finding.
- Informational / won't-fix / low-confidence noise ⇒ a line in the map's Notes, **not** a child issue.

A chart of authoritative-sounding garbage is worse than a smaller true one. This gate is the whole reason the chart can be trusted.

### 5. Chart

Mint a **new** map and file the children. First ensure the label set exists — `gh label create <name> --force` for each `security:*`, `severity:*`, and `sec:*` label you'll apply; `--force` makes it idempotent, so re-runs are harmless and a missing label never hard-fails the `issue create`. The exact `gh`/`gh api` incantations — sub-issue linking, native `blocked_by` dependencies, the frontier query, and **the two traps (`--paginate`, and gating on a blocker's `state` not the summary count)** — live in `docs/agents/issue-tracker.md` under *Wayfinding operations*. Follow that doc; do not restate its commands here.

- **Map**: `gh issue create --label wayfinder:map`, title `Security audit — YYYY-MM-DD`. Body uses the three sections in **The map body is the report**.
- **Supersede** the prior open audit map from step 1: close it with a comment pointing at the new map's number. Skip only if the user asked for parallel maps.
- **Finding tickets** — one per confirmed/plausible vuln. Labels: `wayfinder:task`, `security:finding`, one `severity:*`, one `sec:*`. Body = the **Finding ticket** template.
- **Audit tickets** — one per time-boxed / unreached surface. Labels: `wayfinder:research`, `security:audit`, one `sec:*`. Body = the **Audit ticket** template.
- **Cap guard** — the map holds at most ~100 sub-issues. If findings + audit tickets would exceed it, file every **Critical/High** finding as a ticket, roll **Low/Info** findings into the map's Notes as a table, and record the rollup. Findings are independently fixable, so add no blocking edges — the map is a flat parallel frontier.

### 6. Hand off

Post the map number and a one-paragraph summary. The map body is already the report and the frontier; a cold session runs the frontier query from `issue-tracker.md` and starts on the first unblocked, unassigned child.

## Scope & categories

Five scope classes, each a `sec:*` label. Entry points are for *this* repo (Go + sqlc + Postgres, three binaries `cmd/{web,worker,prober}`); verify against the tree in step 1.

- **`sec:ssrf-egress`** — *ASM outbound surface, first-class.* The product makes attacker-influenced outbound requests: crt.sh CT fetch (`internal/queue/crtsh.go`, `internal/scan/crtsh.go`), DNS resolution-walk (`internal/measure/resolutionwalk/`), TLS/HTTP probing (`internal/measure/{tlsacceptance,httpexchange,connectoutcome}`), delivery webhooks (`internal/delivery/runner.go`). Look for: request targets built from seed/scan data without egress validation, redirect following, DNS-rebinding, unbounded fan-out / resource exhaustion, response-size limits.
- **`sec:ssrf-egress` (SSH push)** — *first-class, easily missed.* `cmd/worker` execs the prober locally **and** pushes it over SSH to external vantage boxes (`deploy/prober/`: `entrypoint.sh`, `sshd_config`). Look for: host-key handling, command injection into the SSH invocation, key material handling, `from=`/`restrict` enforcement.
- **`sec:authn-session` / `sec:authz`** — largest surface, hand-rolled. `cmd/web/{handlers.go,auth.go,sso.go,settings_sso.go}`, `internal/auth/` (bcrypt, stateless signed session cookies, TOTP, single-use setup token, password reset, invites, personal tokens; OIDC via `coreos/go-oidc`). Look for: missing `requireLogin`/`requireAdmin` on a route, cookie signing/verification flaws, TOTP replay, setup/reset token reuse, SSO issuer/sub binding, open redirect on `/callback`, IDOR.
- **`sec:injection`** — DB is fully sqlc-parameterized (`internal/db/*.sql.go` — generated, do not hand-audit). The real risk is **hand-written SQL or `fmt.Sprintf` outside `internal/db`**; grep for it. Also template/HTML injection in `cmd/web/templates_*.go`, and command/path injection anywhere a subprocess or filepath is built from input.
- **`sec:secrets`** — env-only config (`internal/env/env.go`). Look for: secrets committed to git history (step 2 tools), secrets in logs, the file-backed session signing key (`internal/auth/key.go`, `/app/state`) handling.
- **`sec:supply-chain`** — `go.mod`/`go.sum` vs `govulncheck`; unpinned or suspicious deps.
- **`sec:infra-deploy`** — `Dockerfile`, `deploy/prober/Dockerfile`, both `docker-compose.yml`. Look for: container privileges/capabilities, exposed ports, healthcheck/secret leakage, non-root enforcement.
- **`sec:ci`** — `.github/workflows/`. Look for: over-broad `GITHUB_TOKEN` permissions, script injection via untrusted inputs, unpinned actions. (Note: CI currently runs no SAST/dep/secret scanning — a legitimate `sec:ci` finding in itself.)
- **`sec:crypto`** — hashing/signing/random choices in `internal/auth/`; look for weak algorithms, non-constant-time comparison, predictable tokens.

## Severity ramp

Five levels, matching the product's own console vocabulary — labels `severity:{critical,high,medium,low,info}`. Rank by **impact × likelihood** in one line per finding; no CVSS vectors. `critical` is reserved for pre-auth or trivially-exploitable loss of confidentiality/integrity.

## Finding ticket template

```
**Surface / category / severity** — <sec:*> · severity:<level> · confidence:<confirmed|plausible>
**Location** — <file:line>, <function or route>
**Claim** — <one sentence: what the vulnerability is>
**Failure scenario** — <concrete inputs/state → unsafe outcome; the thing a fix must prevent>
**Impact × likelihood** — <the one-line severity justification>
**Suggested fix** — <concrete, repo-idiomatic>
**Verification** — <the test to add or command to run that proves it fixed>

Part of #<map>
```

## Audit ticket template

```
**Scope boundary** — <exact files/dirs this ticket covers>
**Threat classes to look for** — <the sec:* threat classes relevant to this surface>
**Entry points** — <starting files>
**How to report** — File each confirmed vuln as a new finding ticket on map #<map> using the finding template. Close this ticket with a summary comment when the surface is swept.
**Definition of done** — <what "swept" means for this surface>

Part of #<map>
```

## The map body is the report

The map's three sections carry the whole run — no separate report artifact:

- **Notes** — scope swept, which scanners ran and which were skipped (Scan-coverage), counts by severity × category, and the Low/Info rollup table if the cap guard fired.
- **Decisions-so-far** — the confirmed-findings rollup, severity-ranked, each linking its ticket. Record the reasoning for any recommendation you ruled on so a later session sees the *why*.
- **Fog** — surfaces time-boxed or not reached — the exact set that became audit tickets. This is the frontier the next session works.
