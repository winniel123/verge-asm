# ADR-0103: A Vantage is one position; the prober connection is optional provisioning detail on that same row

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#187 Prober provisioning](https://github.com/winniel123/verge-asm/issues/187) and [#188 Measurement binary](https://github.com/winniel123/verge-asm/issues/188)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

Two tickets landed a `vantage` table each, from opposite ends of the same term.

#188 (the measurement binary) needs a Vantage as the thing the `dns` Scan dispatches over: a **name**, a **class** (`internet`/`internal`/`unverified`) re-verified every batch, and the recursive **resolver** that [ADR-0070](./0070-a-control-probe-is-asked-from-where-the-answer-it-discriminates-was-asked-from-and-the-query-path-is-one-declared-parameter.md) already ruled is part of the position's identity and must live on the row. Its dispatch, queue, batch, and observation tables all FK to `vantage_id`, and it ships one row — `local`, resolver-only — so a name-scope install resolves from first boot with no prober at all.

#187 (prober provisioning) needs a Vantage as a **provisioned prober endpoint**: `host`, `port`, a non-root `username`, an `availability` derived from connection state, the instance-generated SSH `public_key`, the TOFU-pinned `host_key`, and the `created_by` admin. Provisioning is the act CONTEXT.md and [#124](https://github.com/winniel123/verge-asm/issues/124) name as *declaring "this vantage is on the internet"* — there is no `network_position` enum, the intent is carried by the act.

Both created `CREATE TABLE vantage`, and both defined a `ListVantages` query. Applied in order, the second migration fails (`relation "vantage" already exists`); the generated Go fails on the duplicate symbol. The collision is not two tables that happen to share a name — CONTEXT.md's **Vantage** and **Vantage class** entries already describe **one** concept. A provisioned prober *is* a measurement position that has been declared to be on the internet; the `local` row *is* a measurement position with no prober. The two tickets modelled the same term from the two ends of its life and never met in the middle.

## Decision

**There is one `vantage` row. Measurement identity is mandatory on it; the prober connection is optional detail on the same row, present only where a prober was provisioned.**

**1. Measurement identity is mandatory (`name`, `class`, `resolver`).** Every Vantage — `local` and provisioned alike — carries a `UNIQUE` name, a `class` defaulting to `unverified`, and a `resolver`. These are `NOT NULL`; the `dns` dispatch reader never has to reason about a missing identity, and an unset resolver is the empty string, not `NULL`.

**2. The prober connection is optional and nullable, together.** `host`, `port`, `username`, `availability`, `public_key`, `host_key`, and `created_by` are `NULL` for a resolver-only Vantage that has no prober, and are set when an endpoint is provisioned. The shipped `local` row carries none of them — it is a position we resolve *through*, not one we have a machine at. The `(host, port, username)` endpoint uniqueness index still holds; `NULL`s are distinct under it, so any number of prober-less Vantages coexist.

**3. Provisioning maps an endpoint onto a mandatory identity.** A provisioned prober still needs a `name` and a `resolver`. The name is derived from the endpoint as `username@host:port` — unique exactly when the `(host, port, username)` triple is, so it satisfies `name UNIQUE` without a second uniqueness story and reads back to the operator as the endpoint they typed. The resolver ships **blank** (`''`), the honest default: provisioning declares the position is on the internet, but the operator has not yet said *through which recursive resolver* it measures, and they set it afterward. `class` ships `unverified` until a prober observes the instance's presented address; `availability` starts `pending`.

**4. One generated symbol per query.** The web prober list keeps `ListVantages` (scoped to provisioned rows, `WHERE host IS NOT NULL`, so `local` never shows as a prober). The measurement dispatch reader is renamed **`ListVantagesForDispatch`** — it reads only `(name, class, resolver)` over *every* Vantage. The two are different projections of one table and no longer collide.

## Consequences

- **The shipped `local` Vantage survives the merge unchanged**, and the dispatch/queue/observation FKs to `vantage_id` still resolve it. A name-scope-only install still measures from first boot with no prober.
- **Provisioning is unchanged from the operator's side.** #187's flow still takes `(host, port, username)`, the worker still generates the keypair and publishes only the public half, TOFU host-key pinning and `MarkVantageUnavailable` still work, and only the public key reaches web. The single addition is the derived `name`, computed at the call site.
- **The Go model gained nullable prober columns.** `db.Vantage`'s `Host`, `Port`, `Username`, `Availability`, and `CreatedBy` are now nullable (`pgtype`), matching the table; consumers that render provisioned probers read `.String`/`.Int32`, and the measurement dispatch reads its own narrow `ListVantagesForDispatchRow`.
- **A later ticket owns the resolver-on-provision UX.** A provisioned Vantage with a blank resolver will not usefully resolve until the operator sets one; surfacing and prompting for that is out of scope here and left as a forward note.

## Alternatives rejected

| Option | Why it lost |
| --- | --- |
| **Two tables** (`vantage` for probers, `measurement_vantage` for dispatch) | Denies the domain: CONTEXT.md's Vantage is one concept, a provisioned prober *is* a measurement position declared on the internet. Two tables would need a join or a duplicated identity to answer "measure from this prober," and would split the `class` re-verification #188 runs from the provisioning act #187 says performs it. |
| **Keep prober columns `NOT NULL` with defaults** (empty host, port 22, a sentinel creator) | Keeps the Go types as plain scalars, but makes the `local` row lie: an empty host and a fabricated `created_by` FK assert a prober that does not exist. The honest representation of "no prober" is `NULL`, and the `dns` reader must be able to tell a resolver-only position from a provisioned one. |
| **Derive `name` from `host` alone** | Collides for two probers on the same host but different port or username — both legitimate distinct endpoints under the `(host, port, username)` index. `username@host:port` is unique exactly where the endpoint is. |
| **Rename the web `ListVantages` instead of the dispatch one** | The web list is the narrower, provisioning-facing projection and reads naturally as `ListVantages`; the dispatch reader's job (fan out over every position) is what wants the qualifier, hence `ListVantagesForDispatch`. |

## Thin ground

- **The blank-resolver default is a placeholder for a UX not yet designed.** Provisioning declares a position on the internet but leaves it non-resolving until the operator sets a resolver. If a build session finds operators routinely forget, the provisioning flow may need to require a resolver up front — which would move resolver from optional-blank toward mandatory-at-provision and should be re-argued then.
- **`name` derivation is a convention, not a constraint.** `username@host:port` is generated at the call site, not enforced by the schema; nothing stops a future writer inserting a provisioned row with an unrelated name. It holds because there is exactly one provisioning path today.
