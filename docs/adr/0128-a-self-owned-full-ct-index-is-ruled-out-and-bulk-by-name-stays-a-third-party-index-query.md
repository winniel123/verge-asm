# ADR-0128: A self-owned full CT name-index is ruled out — bulk-by-name stays a third-party index query

- **Status:** Accepted
- **Date:** 2026-08-31
- **Ticket:** [#934 Write the ADR](https://github.com/winniel123/verge-asm/issues/934); decision from [#927 Decide the deployment model](https://github.com/winniel123/verge-asm/issues/927)
- **Map:** [#925 Replace crt.sh and Cert Spotter with verge-asm's own full CT index (spec)](https://github.com/winniel123/verge-asm/issues/925)
- **Research:** [`docs/research/ct-own-index-feasibility-2026-08.md`](../research/ct-own-index-feasibility-2026-08.md) ([#926](https://github.com/winniel123/verge-asm/issues/926))
- **Upholds:** [`docs/spec/ct-source-replacement.md`](../spec/ct-source-replacement.md). This ADR does **not** supersede that spec; it confirms it.

## Context

The Cert Spotter bulk-by-name primary is an **operator-keyed, paid** source. That cost prompted a question: could verge-asm build its **own** full Certificate-Transparency name-index — every CT log ingested to genesis, SANs parsed and indexed, answering `%.example.com` from our own store — and so escape **both** crt.sh and Cert Spotter, with no third-party index and no operator key?

Map #925 charted the question and tested feasibility first (#926) before any design.

## Decision

**verge-asm does not build a self-owned full CT name-index.** Bulk-by-name stays a **third-party index query**:

- **crt.sh** — keyless fallback (`DefaultOn: true`).
- **Cert Spotter** — optional operator-keyed primary (`DefaultOn: false`, `operator-credentialed`), active only when the operator supplies a key.

The **forward-only `ct-tail` engine** (`internal/scan/cttail.go`, `internal/queue/cttail.go`) stays the keyless CT-logs-direct capability: opt-in (`DefaultOn: false`), forward-only, **never backfilled to genesis**.

No code changes. This ADR records a boundary, not a build.

## Why (measured)

#926 re-measured the corpus live on 2026-08-31 and found a per-operator own-index infeasible — and more so than the 2026-07 framing in [`passive-discovery-sources.md`](../research/passive-discovery-sources.md) §2.1:

- **30,360,147,264 log entries** across 39 usable+readonly logs (~5× the earlier framing).
- One-time build to genesis ≈ **176 TB download / ~273 million HTTP fetches / weeks-to-months** on a normal box, before a single query is answered.
- Forward upkeep ≈ **3,200 entries/sec forever** once caught up.
- **No prebuilt dump or syncable index exists** to short-cut it. Only the static-ct-api tiled logs (~15% of the corpus) are natively mirrorable; the RFC 6962 85% can only be scraped entry-by-entry.

The blocker is **not** terms or rate limits — the Chrome CT Log Policy obliges logs to allow retrieval faster than they grow. The blocker is **physics and per-install duplication**: every install would re-scrape the same 176 TB from the same 39 logs.

## Considered options

- **Per-operator full ingest** — rejected. Infeasible on a normal box (above).
- **A verge-run central index** — rejected by #927. Feasible, but it *is* re-building and operating a crt.sh, and it reopens the project's "no central deployment" stance while moving the cost onto the project.
- **A verge-published syncable dataset** — rejected by #927. No third party publishes one, so verge would pay the central ingest cost anyway; a tiled-only mirror is lighter but silently drops the RFC 6962 majority of issuance.

Escaping both third-party indexes is possible **only** under central deployment (the last two options). The operator declined central deployment, so the own-index is dropped.

## Consequences

- The `ct-tail` engine's **`DefaultOn: false` and forward-only, never-backfill invariant is now measured-justified**, not merely cautious. #926's note is its standing evidence.
- The [`ct-source-replacement.md`](../spec/ct-source-replacement.md) §2–§3 split (crt.sh fallback + Cert Spotter primary) **stands unchanged**.
- If a keyless, self-owned bulk CT capability is wanted later, it is a **central-deployment decision for a fresh wayfinder**, not a shippable default. The research note is the answer to re-open first.
