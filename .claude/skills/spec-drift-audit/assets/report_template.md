# Spec Drift Audit — <project> — <date>

## Summary

| Status | Count |
|---|---|
| Implemented | |
| Partial | |
| Stub | |
| Missing | |
| Undocumented | |
| Unverifiable | |

**Headline:** <One or two sentences. E.g. "Of 41 claimed features, 22 work end-to-end. The 9 stubs are concentrated in third-party integrations, where client scaffolding exists but fetch logic does not. The UI advertises 4 finding types no scanner emits.">

**Highest-impact gaps** (things a user would hit fastest):
1. <feature> — <status> — <one line>
2. …

## Findings by subsystem

Within each subsystem, order: Missing, Stub, Partial, Implemented, Undocumented.

### <Subsystem name>

#### <Feature name> — **<Status>**
- **Claimed in:** `<file>` §<section> — "<short claim>"
- **Evidence:** `<file>:<lines>` — <what's there>
- **Gap:** <what's claimed but absent> *(omit for Implemented)*
- **Search:** <terms/paths> *(Missing only)*
- **Note:** <flags, bugs, caveats> *(optional)*

#### …

### <Next subsystem>

…

## Recommended doc changes

Concrete edits that make the docs match the code today. One bullet per Missing/Stub/Partial feature.

- `README.md` §Features: remove "Automatic asset deduplication" or change to "Asset deduplication *(planned — see #<issue>)*"
- `docs/integrations.md`: mark Censys and BinaryEdge as "scaffolded, not functional"
- …

## Recommended code changes (optional)

Where the fix is small enough to note — e.g. "unwired: add `portscan` to `scheduler/registry.py` and it works."

- …

## Coverage

- **Claim sources audited:** <list>
- **Claim sources skipped:** <list + why>
- **Subsystems not audited:** <list + why>
- **Unverifiable items:** <count> — see individual entries for reasons
- **Method:** static reading / grep / ran locally (state which, per subsystem if mixed)
