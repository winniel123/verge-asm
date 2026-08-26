---
name: spec-drift-audit
description: Audit a codebase for "phantom features" — capabilities that specs, READMEs, roadmaps, API docs, UI menus, or CLI help claim exist but that are missing, stubbed, hardcoded, or only partially wired up. Produces a feature-by-feature implementation status report with evidence and file references. Use this whenever the user asks what's actually implemented vs. documented, mentions spec drift, wants to know if the docs are lying, asks "does feature X really work", wants a gap analysis or implementation coverage report, is prepping a release or README and wants it honest, or is losing track of which planned features are done. Trigger even if they don't say "audit" — "which of these actually work?" or "is the README accurate?" counts.
---

# Spec Drift Audit

Find the gap between what a project *says* it does and what the code *actually* does. The output is a status report that a maintainer can act on: which features are real, which are fake, which are half-done, and exactly where to look.

The hard part isn't reading code — it's being rigorous about what counts as "implemented." A feature with a route handler that returns `{"status": "todo"}` looks implemented from a grep. A UI button with a click handler that does nothing looks implemented from the UI. This skill exists to stop those from slipping through.

## Workflow

### Step 1: Map where claims live

Before reading code, inventory every place the project makes promises. Use `scripts/find_claim_sources.sh` (or do it by hand) to locate:

- **Spec/design docs**: `docs/`, `specs/`, `design/`, `ARCHITECTURE.md`, `SPEC.md`, RFCs, ADRs
- **README and marketing surfaces**: `README.md`, `docs/index.md`, feature lists, comparison tables, landing page copy in the repo
- **Roadmaps and changelogs**: `ROADMAP.md`, `CHANGELOG.md`, GitHub issues/milestones if accessible
- **API contracts**: OpenAPI/Swagger, GraphQL schemas, protobuf, route registrations
- **User-facing surfaces**: CLI `--help` strings and subcommand registrations, UI navigation/menus/settings pages, config file schemas and example configs
- **Tests**: test names are claims too (`test_asset_deduplication` implies deduplication exists)

Don't skip the UI and CLI surfaces. Those are where phantom features hurt users most — a menu item that does nothing is worse than a doc paragraph that's aspirational.

Ask the user which sources are authoritative if there's conflict (e.g. README vs. spec). Default: the spec is the intent, the README is the promise, the code is the truth.

### Step 2: Extract a feature inventory

Turn the claim sources into a flat list of discrete, checkable features. One row per capability, not per document section. Good granularity:

- Too coarse: "Asset discovery"
- Right: "Subdomain enumeration via passive DNS", "Port scanning of discovered hosts", "Scheduled re-scans of existing assets", "Asset deduplication across sources"
- Too fine: "Handles IPv6 in the subdomain enumerator's result parser"

For each feature, record where it was claimed (file + line/section) and the verbatim or near-verbatim claim. Save this inventory to a working file (`audit/feature_inventory.md` or similar) before verifying anything — it's the checklist and it keeps the audit honest about coverage.

For a large project, batch by subsystem and confirm the inventory with the user before verification. It's cheap to fix "you're missing the notification system" here and expensive to discover it after the report is written.

### Step 3: Verify each feature against the code

This is the core. For each feature, find the implementation and classify it using the evidence ladder in `references/evidence_ladder.md`. Read that file before classifying anything — the whole value of the audit rests on applying the levels consistently.

The short version of the levels:

| Status | Meaning |
|---|---|
| **Implemented** | End-to-end path exists, does real work, has at least one test or is trivially verifiable |
| **Partial** | Real logic exists but a meaningful piece is missing (e.g. backend done, no UI; happy path only; one of three claimed sources) |
| **Stub** | Entry point exists (route, function, menu item, CLI flag) but the body is a placeholder, TODO, hardcoded return, `NotImplementedError`, or no-op |
| **Missing** | Nothing in the code corresponds to the claim |
| **Undocumented** | Exists in code but not in any claim source (reverse drift — worth reporting, the docs are under-selling) |
| **Unverifiable** | Couldn't determine (e.g. depends on an external service you can't inspect); say why |

How to actually verify, in rough order of cost:

1. **Follow the entry point inward.** Start from the user-facing surface (route, CLI command, UI action) and trace to the code that does work. Stop when you hit real logic or a dead end. A feature is only as real as its shallowest link.
2. **Look for stub signatures.** Grep the traced code for `TODO`, `FIXME`, `XXX`, `NotImplemented`, `pass` as sole body, `return None`/`return []`/`return {}` with no computation, `raise NotImplementedError`, `console.log("not implemented")`, hardcoded sample data, `mock`/`fake`/`dummy` in non-test code, commented-out blocks, feature flags defaulting to off.
3. **Check the wiring, not just the existence.** A scanner module can exist and be complete but never be registered, imported, scheduled, or reachable from any route. Check that the thing is actually called from somewhere that a user can trigger.
4. **Check the data path.** For anything that claims to persist or display data: is there a schema/model/migration for it? Does the write path exist? Does the read path exist? Does the UI actually bind to it?
5. **Check tests.** A test that exercises the real path is strong evidence. A test that mocks the whole thing away is neutral. No test is a yellow flag, not a red one.
6. **Run it if you can.** If there's a dev setup, actually invoking the feature beats any amount of reading. Don't spend an hour getting the environment up for one feature, but if it's already running, use it.

Record evidence for every classification: file paths, line ranges, and a one-line reason. "Partial — `scanners/portscan.py:40-118` implements TCP connect scan but the spec claims SYN scan and UDP; neither present; not scheduled anywhere (grep `portscan` in `scheduler/` returns nothing)." That level of specificity is what makes the report usable.

Never mark something Implemented because the function name matches. Read the body.

### Step 4: Write the report

Use the template in `assets/report_template.md`. Put the summary table first — the maintainer wants the shape of the problem in ten seconds. Then details grouped by subsystem, worst-first within each group (Missing and Stub before Partial before Implemented).

Include a "Recommended doc changes" section: for each Missing/Stub feature, suggest the specific edit — remove the claim, move it to a roadmap, or add a "planned" badge. This turns the audit into a PR rather than a complaint.

End with a "Coverage" note: which claim sources were audited, which were skipped and why, and how many features were Unverifiable. An audit that's silent about its own gaps is just another unreliable doc.

## Calibration notes

- **Be exact about what "partial" means.** "Partial" without a reason is useless. Always say what's present and what's absent.
- **Distinguish feature-flagged-off from stub.** Code that's complete but gated behind a flag is Implemented (with a note), not Stub. Code that's gated and *also* incomplete is Partial.
- **Don't penalize documented limitations.** If the README says "UDP scanning is not yet supported," that's not drift — that's honesty. Only flag claims, not disclaimers.
- **Reverse drift matters.** Undocumented working features are the cheapest wins in the whole report — they just need a sentence in the README.
- **Security tooling has a specific failure mode.** For a scanner/ASM platform, watch for integrations that "support" a data source by having a client class with the auth flow written and every fetch method stubbed. Also watch for detection/finding types listed in a UI filter dropdown that no scanner ever emits. Both are common and both look fully real from the outside.
- **When the codebase is large**, do subsystem by subsystem and check in between them rather than producing one giant report at the end. Partial results early beat complete results never.

## Bundled files

- `references/evidence_ladder.md` — Detailed classification criteria with examples. Read before verifying.
- `scripts/find_claim_sources.sh` — Lists likely claim-source files and greps for stub signatures across the repo. Run at the start of Step 1 and again during Step 3.
- `assets/report_template.md` — Report structure. Copy and fill in.
