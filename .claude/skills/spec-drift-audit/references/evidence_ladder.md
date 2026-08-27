# Evidence Ladder

Classification rules for feature status. Apply these the same way every time so that two features with the same status are actually in the same shape.

## The core test

For every feature, ask: **if a user tried to use this right now, what would happen?**

- They'd get the documented result → Implemented
- They'd get part of it, or it works only in some cases → Partial
- They'd hit an error, a placeholder, a no-op, or fake data → Stub
- There's nothing to try → Missing

Everything below is elaboration on that question.

## Implemented

All of the following:

1. There's a user-reachable entry point (route, CLI command, UI action, scheduled job, config option that changes behavior).
2. Following the call path from that entry point reaches code that performs the claimed operation on real inputs and produces real outputs.
3. Any data the feature claims to persist has a storage path (model/schema/migration + write) and, if displayed, a read path bound to the display.
4. At least one of: a test exercises the real path (not fully mocked); or the logic is simple enough that reading it is sufficient; or you ran it.

Notes:
- Feature-flagged but complete → Implemented, annotate "behind flag `X`, default off."
- Works but has bugs → Implemented, annotate the bug. This audit is about existence, not quality.
- Complete but with a documented limitation that the docs acknowledge → Implemented.

## Partial

Real logic exists and does real work, but a meaningful, *claimed* piece is absent. Always state both halves. Common shapes:

- Backend/API complete, no UI (or vice versa)
- One of N claimed sources/providers/protocols implemented (e.g. claims "Shodan, Censys, and BinaryEdge"; only Shodan client does real fetches)
- Happy path only; documented error handling / retry / pagination absent
- Manual trigger works; documented scheduling/automation doesn't exist
- Data is collected and stored but never surfaced anywhere
- Write path exists, read path doesn't (or the reverse)
- Works for one asset type, spec claims all types

Distinguish from Stub: Partial has a working slice a user could benefit from today. Stub has nothing usable.

## Stub

An entry point or scaffold exists, but no real work happens. Signatures:

- Function body is `pass`, `...`, `return None`, `return []`, `return {}`, `return True`, or a hardcoded literal with no computation
- `raise NotImplementedError` / `throw new Error("not implemented")` / `TODO` as the body
- Handler returns static/sample/fixture data in production code
- UI element renders but its handler is empty, logs to console, or shows "Coming soon"
- CLI flag is parsed and then never read
- Config option is validated and stored but nothing branches on it
- Class exists with method signatures matching the spec, bodies are placeholders
- Module exists and is complete but is never imported, registered, routed, or scheduled from anywhere reachable (dead code masquerading as a feature — call this out specifically as "unwired")

Stubs are the most dangerous category because every surface-level check passes.

## Missing

No file, function, route, model, or UI element corresponds to the claim. Say what you searched for so the user can sanity-check (e.g. "grepped for `dedup`, `duplicate`, `fingerprint`, `merge` across `src/` — no hits outside tests").

## Undocumented

Reverse drift: something works in the code and no claim source mentions it. Found by scanning routes, CLI commands, UI nav, and scheduled jobs for things not in the inventory. Report these — they're free wins.

## Unverifiable

You couldn't determine status. Legitimate reasons: depends on an external service with no local stand-in; requires credentials; behavior lives in a compiled binary or vendored blob; the code is too dynamic to trace statically and you can't run it. Illegitimate reason: didn't have time — in that case say "not audited" in the coverage section instead.

## Evidence format

Every classification carries:

```
Status: <one of the six>
Evidence: <file:lines> — <what's there>
Gap (if any): <what's claimed but absent>
Search (if Missing): <terms/paths grepped>
```

Example:

```
Feature: Asset deduplication across discovery sources
Claimed: README.md §Features, docs/spec.md §3.2
Status: Stub
Evidence: core/assets/dedup.py:1-34 — `Deduplicator.merge()` body is `# TODO: implement fuzzy matching` + `return assets`
Gap: No dedup logic; `ingest/pipeline.py:88` calls it, so ingestion "works" but every source produces duplicate rows
```

## Common misclassifications

| Looks like | Actually is | Why |
|---|---|---|
| Implemented (route exists, returns 200) | Stub | Handler returns fixture data |
| Implemented (module is 300 lines) | Stub (unwired) | Never imported from anything reachable |
| Missing (no file named after the feature) | Implemented | Feature lives inside a differently-named module; grep for behavior, not names |
| Partial | Implemented | The "missing" piece is a documented limitation, not a claim |
| Implemented (tests pass) | Stub | Tests mock the entire implementation |
| Stub | Implemented behind flag | Body is real; flag gate is what you saw returning early |
