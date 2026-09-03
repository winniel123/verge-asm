# ADR-0124: a backup carries the estate and no secret, and updating from the UI is checked, surfaced and guided — never self-applied

- **Status:** Accepted
- **Date:** 2026-08-26
- **Ticket:** [#664 B1 — ADR-0124: data-only backup; update = check/surface/guide, not self-replace](https://github.com/winniel123/verge-asm/issues/664)
- **Map:** [#658 Consume v3.18.0 — API token surfaces (#390) + Backup & updates (#391)](https://github.com/winniel123/verge-asm/issues/658)
- **Keeps and extends, withdraws nothing:** [ADR-0053](./0053-a-secret-is-held-only-where-its-act-is-performed-and-the-shared-store-holds-none.md)'s rule that *a secret is held only where its act is performed and the shared store holds none* — and its already-decided consequence that *"a database backup carries the whole estate and no credential."* This ADR builds the shipped backup **on** that property; it does not reopen or narrow it.
- **Inherits the runtime constraint of:** [ADR-0001](./0001-stack-and-runtime.md) — one image, two non-root compose services, distroless. That runtime is why the export is Go-native (no `pg_dump` in the image) and why the container cannot rewrite its own image.
- **Re-files, design-first:** the substance of the old, closed [#410](https://github.com/winniel123/verge-asm/issues/410). Its map pre-assigned **ADR-0118**, since reused for report-scheduling ([ADR-0122](./0122-a-report-schedules-cadence-is-a-dispatch-time-so-it-honours-the-clock.md)); this decision takes the **fresh** number 0124 and does not resurrect the clobbered one.

## Context

Design package **v3.18.0** surfaces the operator-facing commitment for #391: the Settings · Instance tab gains three cards — **Backup**, **Restore**, and **Version & updates** — and retires the old `.Instance.Update` callout. The view and its `## Behavior` contract (`design-system/verify/WORK-ORDER-390-391.md`) are now the authoritative spec. This ADR records the two boundaries the deployment **forces**, so the build children (B2/B3/B4/B5) implement a decision rather than improvise one.

Both boundaries are consequences of choices already made, not new ones. They are written down here because each is a place a future session, reading the feature name alone (*"back up the instance," "update from the UI"*), would reach for the obvious implementation and reintroduce a hole this project spent an earlier ADR closing.

## Decision

> **A UI backup carries the estate and its config and carries no secret — it is a Go-native logical dump over the pool `web` already holds, and the session key and the prober key are neither in it nor recoverable from it; on restore they regenerate. "Update from the UI" is check, surface and guide, never self-replace — the container reports its version, checks upstream best-effort, shows migration status and renders the release-authored host steps, and the image swap stays a host action. The UI never composes a shell command beyond the literals a release ships in `.Steps[]`.**

### 1. Backup is data-only and carries no secret — this *keeps* ADR-0053, it does not break it

[ADR-0053](./0053-a-secret-is-held-only-where-its-act-is-performed-and-the-shared-store-holds-none.md) already decided the fact this card is built on: *"a database backup carries the whole estate and no credential, which is a better property to have decided than to discover, and it is the reason a backup does not need to be treated as a keyring."* That property is **kept in full**. This ADR turns the inherited fact into the shipped backup's contract and names the mechanism.

- **The archive contents are the estate and its config — the business tables.** Subjects, the drift timeline, signals, scans, channels, deliveries, retention and instance settings — everything `web` renders and `worker` writes. It is a downloadable attachment the operator can carry to another host and restore.
- **The archive holds no secret, by construction and by intent.** The session signing key lives in the `web-state` volume (`web-state/session.key`) and the prober SSH private key lives in the `worker-state` volume. Under ADR-0053 **neither is ever in Postgres.** A dump that reads the database therefore *cannot* contain them — and this ADR makes that a **rule of the export, not an accident of where the bytes happen to sit**: the backup path reads business tables only, and the two key volumes are never opened by it. A backup that shipped the session key would re-create precisely the hole [ADR-0053](./0053-a-secret-is-held-only-where-its-act-is-performed-and-the-shared-store-holds-none.md) §4.2 exists to prevent — *"a **read-only** leak of the database — a backup, a replica, an export — would otherwise yield live admin sessions"* — so the backup does not carry it.
- **On restore the keys regenerate, and that is a feature, not a gap.** A restored instance comes up with a fresh session key — every prior session lapses, which is loud and acceptable (ADR-0053 already priced *"every session logs out"* as recoverable and non-silent) — and a fresh prober keypair, so the operator re-installs the public half and the prober **re-pins**. The vantage going `unavailable` opens a `Gap` and makes `Exposure` non-constructible ([ADR-0005](./0005-scan-execution-model.md)) until the re-pin, which is the same loud, recoverable failure ADR-0053 accepted for `docker compose down -v`. The Restore card says so in as many words, as a property of the operation rather than fine print.
- **The export is a Go-native logical dump, because the image ships no `pg_dump`.** The distroless runtime ([ADR-0001](./0001-stack-and-runtime.md)) carries no Postgres client binaries, so the backup is **not** a shell-out to `pg_dump`. It is a table-by-table logical dump written in Go over the **pgx pool `web` already holds**, streamed to the operator as an attachment and restorable table-by-table. The precise archive framing (NDJSON-per-table + a manifest carrying the schema version, versus an emitted SQL script) is B3's to settle against the requirement that it be **forward-restorable across a migration bump**. This ADR fixes only that it is logical, Go-native, secret-free, and restorable.

### 2. "Update from the UI" is check, surface and guide — never self-replace

A non-root, distroless container **cannot and must not rewrite its own image.** It cannot, because it runs as an unprivileged user with no Docker socket, no package manager, and a read-only image. And it must not, because a service that can replace the very binary that parses the operator's attack surface is the **maximal** form of the attack surface this product exists to harden against — a compromise of `web` that could also re-image the instance is not a foothold, it is ownership. So "update from the UI" is defined by what the UI *reports and guides*, and bounded by what it will *never do*:

- **Reports the running version.** The UI surfaces `.Instance.Version` (which collapses to `""` on builds that do not stamp it) so the operator can see what is actually running.
- **Checks upstream best-effort, opt-out, air-gap-safe.** When `Release.CheckEnabled`, the **worker** checks the release feed on a daily cadence — best-effort, short timeout, no retry storm. A failure reports nothing rather than alarming. The check is **opt-out**: with it disabled the instance makes **no network call ever**, not even on boot, and the card renders the air-gap copy. This is the only outbound reach the feature has, and it is fully declinable.
- **Shows migration status.** `Migrations.Pending`, computed from the migration table against the binary, renders as a badge — *"schema current"* or *"N migrations pending"* (warn) — so the operator knows whether a restart will migrate before they take one.
- **Renders the literal, release-authored host steps.** When a newer version is seen, the card shows `Release.Latest` and the **literal** guided host steps a release ships in `.Steps[]`. These are release-authored literals. **The UI never composes a shell command** from operator input or its own logic — it displays what the release wrote and nothing else.
- **The image swap stays a host action.** Pulling the new image and recreating the containers is done by the operator on the host, following the surfaced steps. The instance participates by *telling the operator what to run*, never by running it.

## Rationale

### Why the backup rule is stated as an export invariant, not left to where the bytes sit

ADR-0053 makes it *true today* that no secret is in the database, so a naïve "dump everything reachable" would already omit the keys. But "already omit" is an accident of the current schema, and the whole discipline of [ADR-0053](./0053-a-secret-is-held-only-where-its-act-is-performed-and-the-shared-store-holds-none.md) is that a secret-placement property must be **a rule at the site that could violate it**, not a happy consequence one refactor away from breaking. The backup is exactly such a site: it is the one code path whose *job* is to serialise state for carrying off-host, and it is where a future *"also snapshot the volumes so restore is turnkey"* would land. Naming *data-only, no secret* as the export's own invariant is the same move ADR-0053 made for the store — make the violation inexpressible where it would be written, rather than trust that nothing upstream will change.

### Why self-replacement is refused where a lesser tool would offer it

The obvious product feature is a one-click *"Update now"* button. It is refused for the same reason ADR-0053 refuses a *"show secret"* affordance and [ADR-0123](./0123-a-token-api-is-read-only-opt-in-and-a-bearer-path-separate-from-sessions.md) refuses a write-capable token: the capability itself is the hazard, so the design removes the thing rather than guarding it. A button that re-images the instance is a capability `web` would hold, and `web` is the tier an attacker reaches first (ADR-0001, ADR-0053). Guiding the host action keeps the most destructive operation an instance can undergo firmly on the **host** side of the boundary the two-service topology draws, where a compromise of `web` cannot reach it.

### Why best-effort and opt-out, rather than mandatory or absent

An update-check that fails loudly, retries, or blocks boot would make a self-hosted tool's availability depend on a third-party feed — the coupling the whole packaging line avoids. Making it best-effort (silent on failure, no retry storm) keeps the *"is there a newer version?"* signal without that dependency, and making it opt-out keeps an **air-gapped** instance genuinely silent: no boot-time call, no daily call, nothing. The alternative of no check at all leaves the operator to discover updates by other means, which the design package explicitly declines by shipping the Release card.

## Consequences

- **The backup export becomes buildable** (map child B3): a streamed, Go-native logical archive of estate + config over the pgx pool, carrying no secret, with a `.Backup.LastAt/LastSize` record of the last UI-taken backup. B3 settles the archive framing under the forward-restorable-across-a-migration-bump requirement.
- **Restore becomes buildable** (B4): multipart preflight → `.Preflight`, typed-confirm apply → overwrite, **regenerate the session and prober keys**, lapse every session. The key regeneration is not an incidental side effect but a *named guarantee of this ADR* — a restored instance never carries a prior instance's session or prober key.
- **The Version & updates surface becomes buildable** (B2/B5): `.Instance.Version`, the `Migrations.Pending` badge, the Release card and `POST /settings/updates/check`, and the worker's daily best-effort release-feed runner. No path in any of these re-images the instance.
- **[ADR-0053](./0053-a-secret-is-held-only-where-its-act-is-performed-and-the-shared-store-holds-none.md) is preserved and extended, not withdrawn.** Its rule stands unaltered. Its #121 backup consequence gains a dated forward-pointer here recording that ADR-0124 relies on and builds the shipped backup on that fact. Nothing in ADR-0053 is struck.
- **No new refusal is created and none of #391's substance is decided by approximation.** Every hole the design package leaves open (archive format, whether async sealing ever renders `InProgress`) is handed to a named build child, not guessed here.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Back up the volumes too** (session key + prober key) so restore is turnkey | Re-creates the exact hole [ADR-0053](./0053-a-secret-is-held-only-where-its-act-is-performed-and-the-shared-store-holds-none.md) §4.2 closes — a read-only leak of a backup would then mint live admin sessions and hand over a machine outside the estate. The keys regenerate loudly on restore; a turnkey restore that carries them is not worth reopening the property the whole two-volume split exists to hold |
| **Shell out to `pg_dump`** | The distroless image ([ADR-0001](./0001-stack-and-runtime.md)) ships no Postgres client binaries, and adding them widens the image and the attack surface to avoid writing a table-by-table dump `web` can already produce over the pool it holds. A Go-native logical dump has no new dependency and no shell |
| **A one-click "Update now" that re-images the instance** | The maximal attack surface this product hardens against: a service that can replace its own binary, held by the tier an attacker reaches first. Refused at the capability, not guarded — the image swap stays a host action and the UI only guides it |
| **The UI composes the upgrade shell commands itself** | Any command the UI assembles from its own logic or operator input is a command-injection surface over the host. The UI renders only the **literal** release-authored `.Steps[]`; it composes nothing |
| **Mandatory, retrying update check** | Couples instance availability and boot to a third-party feed and denies an air-gapped operator a genuinely silent instance. Best-effort + opt-out keeps the signal and the silence both |
| **Resurrect the pre-assigned ADR-0118** | That number was reused for report-scheduling ([ADR-0122](./0122-a-report-schedules-cadence-is-a-dispatch-time-so-it-honours-the-clock.md)) after #391's old map assigned it. Minting the fresh 0124 keeps the ADR log single-valued, per the map's numbering rule |

## Amendment — [#1240](https://github.com/winniel123/verge-asm/issues/1240): three claims the release pipeline makes true, on §2's own subject

The [#1064](https://github.com/winniel123/verge-asm/issues/1064) release-pipeline map closed 25
tickets. Three of them land here, on §2's subject — *"the container reports its version, checks
upstream best-effort."* **Nothing in this ADR's Decision moves.** Each bullet is a claim about the
world that follows from §2 without an alternative having been rejected, which is what
[ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)'s split
makes an amendment rather than a withdrawal. Three bullets is size, not kind.

This ADR's own Context names why the amendment sits here. A future session reading a feature name
alone would reach for the obvious implementation and reintroduce a closed hole. *"Check for
updates"* is such a name.

**1. `.Steps[]` is never feed-delivered** ([#1071](https://github.com/winniel123/verge-asm/issues/1071)).
§2 says the card renders the **literal**, release-authored host steps and that the UI composes
nothing. A comment in the code promised a feed-delivered list at a later milestone. **That promise
is struck, permanently.** The release feed is an external service, and a feed-delivered step list
lets whoever controls the feed put arbitrary shell text in front of an admin, inside a panel that
reads as authoritative. That is this ADR's refusal to let the UI compose a shell, with one more hop.

The third line of the block also changes, to `docker compose ps web worker`. `web` gains **no**
migration-status mode and the image gains **no** `verge` alias, because a running container has
already applied every embedded migration and a status probe inside it can only answer "current".
The open question after an upgrade is whether the new image landed and came up healthy.

**2. The update check compares numeric cores, so it reports stable releases only**
([#1129](https://github.com/winniel123/verge-asm/issues/1129)). The project cuts **no** pre-release
tag, ever. `parseVersion` and `isNewer` gain no pre-release ordering, and the existing assertion
that a release candidate is not newer than its final release is a deliberate contract rather than an
oversight.

A fork that repoints the feed to serve pre-releases gets **no defence**, deliberately. The default
`/releases/latest` endpoint cannot serve one, and refusing a suffixed feed version would make verge
silently ignore a fork's own shipped release.

**The consequence is stated out loud rather than hidden: a source build is never told that a
release shipped.** A hand-built binary carries no version the check can reason about, so
`parseVersion` fails and the state stays `current`. That is §2's best-effort, no-false-alarm design
working as specified. **A detector that flags an unparseable version as an unofficial build is
refused**, because it teaches the UI a version grammar needed nowhere else.

**3. A withdrawal acts through the feed, so it is invisible to an instance already running the
withdrawn version** ([#1161](https://github.com/winniel123/verge-asm/issues/1161)). A bad release is
repaired by the next patch version and contained by two reversible acts. Containment moves the
feed's answer and the floating image tag. It never deletes a tag or a Release, because deletion
breaks every published verify command and every pinned digest.

An operator already running the withdrawn release sees `current` and is told nothing. Once the feed
serves the older version again, the comparison is false and the state stays `current`. **The
silence window is accepted, stated, and given no new signal**, because `isNewer` compares numeric
cores only and has no vocabulary for "the version you run was withdrawn". The successor release
closes the window. **So a withdrawal contains new installs only.**

One further consequence of §2's guided-not-self-applied rule. The container calls `goose.Up` and
nothing else, and no code path ever runs a down migration. **An image downgrade is therefore not a
schema downgrade**, so each release states whether it carried a migration, and the migration case is
a restore from the pre-upgrade dump rather than an image pin.

`docs/spec/release-pipeline.md` §13, §1 and §14 state the mechanisms.
