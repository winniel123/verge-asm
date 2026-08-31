# ADR-0132: a CA bundle is a trust anchor, not the config file §5.1 forbids, so it mounts read-only into the external-db override — never the base

- **Status:** Accepted
- **Date:** 2026-08-31
- **Ticket:** [#950 CA trust-anchor mount convention + the ADR](https://github.com/winniel123/verge-asm/issues/950)
- **Map:** [#947 TLS-verified and external Postgres in Docker Compose](https://github.com/winniel123/verge-asm/issues/947)
- **Keeps, withdraws nothing:** [ADR-0053](./0053-a-secret-is-held-only-where-its-act-is-performed-and-the-shared-store-holds-none.md) — *a secret is held only where its act is performed.* A CA root is **public** trust material, not a secret, so it never raises the question ADR-0053 answers. This ADR mounts a public file; it holds no credential and adds no keyring.
- **Inherits the runtime constraint of:** [ADR-0001](./0001-stack-and-runtime.md) — one image, two non-root compose services, distroless. That runtime is why the image ships **no** CA bundle and why the operator supplies the trust anchor from the host.

## Context

Map [#947](https://github.com/winniel123/verge-asm/issues/947) adds `verify-full` and
external-Postgres support to compose. `verify-full` needs a CA bundle to reach `web` and
`worker` so `pgx` can validate the database server certificate. Packaging spec §5.1 and
[`running.md`](../guides/running.md) both state, in as many words, **"there is no
configuration file to mount"** — the environment holds only what must exist before the
database does, and everything Declared is a row edited through the UI by an authenticated
admin, because [#11](https://github.com/winniel123/verge-asm/issues/11) requires an author
in the audit trail for *who changed the seed list, who launched a scan against production*.

A CA bundle has to arrive as a file. It cannot be a UI-edited row: it must exist **before**
the database connection, which is the exact class §5.1 keeps in the environment, and `pgx`
reads it off a filesystem path (`PGSSLROOTCERT`), not from Postgres. So the feature meets
the one line the packaging discipline draws hardest against. This ADR records that the line
is **not** crossed, and fixes the mount convention the build children implement, so a future
session reading *"mount a CA file"* does not either reopen §5.1 or reach for the base
compose and break CI.

## Decision

> **A CA bundle for `verify-full` is a trust anchor, not the "configuration file" §5.1
> forbids: it carries no author, owes no audit trail, and holds no secret, so mounting it
> does not reopen §5.1. It is supplied by the operator from the host, shipped in no image,
> and bind-mounted read-only into both `web` and `worker` at
> `/etc/verge/ca/root.crt`, with `PGSSLROOTCERT` pointing at exactly that path. The mount
> lives **only** in `docker-compose.external-db.yml`; the base `docker-compose.yml` gains
> no bind mount and keeps its three services, so the CI `compose` job is untouched.**

### 1. A CA bundle is a trust anchor, and it passes §5.1's own test

§5.1 does not forbid every file. It draws a test: *"if a change to it should appear in the
audit trail, it may not live in the environment."* The forbidden thing is **operator
configuration that moves the product's aperture** — a seed, an exclusion, a scan cadence,
notification routing — the acts [#11](https://github.com/winniel123/verge-asm/issues/11)
requires an author for. A CA root is none of that:

- **It carries no author and owes no audit trail.** Rotating the CA an external database
  presents is an infrastructure act on the host, not a product decision an admin takes in
  the UI. Nothing about *what the instrument measures* changes when the trust anchor
  changes. A change to it should **not** appear in the estate's audit trail, so by §5.1's
  own test it may live outside the database.
- **It moves no aperture.** §5.2's dial gate asks whether a knob sits outside every
  `Derivation`, whether it can silence a finding by narrowing, and whether it moves a named
  dimension. A CA bundle answers *outside, cannot, does not*: it gates the transport `web`
  and `worker` use to reach their own database and never touches a measurement value.
- **It holds no secret.** A CA root certificate is **public** by construction. Unlike the
  session key and the prober key that [ADR-0053](./0053-a-secret-is-held-only-where-its-act-is-performed-and-the-shared-store-holds-none.md)
  keeps out of Postgres, this file is safe to read, safe to log, and safe in a backup. The
  secret question ADR-0053 answers does not even arise.

So the file §5.1 forbids and the file `verify-full` needs are **different kinds of file**.
§5.1's rule stands unaltered; this ADR does not narrow it.

### 2. The mount lives in the external-db override only, never the base

The read-only bind is added to `docker-compose.external-db.yml` and to nothing else.

- **`verify-full` only has meaning against an external database.** The override is *the*
  external-database mode: it drops the bundled `postgres` and takes `DATABASE_URL` from the
  environment. A CA anchor for validating a server certificate is only ever needed there.
  The bundled in-compose Postgres speaks plaintext on the compose network by the operator's
  own default (`sslmode=disable`), and taking it to TLS is out of scope for map #947. So the
  mount belongs exactly where the feature it serves lives.
- **The base compose keeps its three services and no bind mount.** The CI `compose` job
  boots **only** the base `docker-compose.yml` and asserts exactly **three** healthy
  services. Adding an inert bind to the base would risk that job and force a placeholder
  source (a `/dev/null`-style default) that reads as a hack. Keeping the mount in the
  override means the base file does not change, and the CI invariant holds untouched.

### 3. Shape: read-only, single-file, both services, operator-supplied

- **Read-only.** The services validate against the anchor; they never write it. The bind is
  `:ro`. Non-root users (`65532` on `web`/`worker`) read a read-only bind without trouble.
- **A single-file bind, not a directory.** The bind maps the operator's host bundle to
  exactly `/etc/verge/ca/root.crt`. The container sees only the one trust file, which is the
  tightest surface. An operator who must present more than one root concatenates them into
  the one PEM bundle, as `PGSSLROOTCERT` already expects.
- **`/etc/verge/ca/root.crt`.** The filename matches libpq's own default `sslrootcert`
  name (`root.crt`), and the `/etc/verge/` namespace reads as system trust material rather
  than app state. `PGSSLROOTCERT` is set to this literal path.
- **Both `web` and `worker`.** Both open the pool and both must validate the server, so both
  mount the anchor and both carry `PGSSLROOTCERT`.
- **The image ships no bundle.** Per [ADR-0001](./0001-stack-and-runtime.md) the distroless
  image carries no operator trust material. The host supplies the operator's own bundle; the
  image bakes nothing in.

## Rationale

### Why the ruling is written down rather than left to the reader

The whole force of §5.1 is a single, memorable rule — *no config file to mount* — that a
session is meant to reach for by reflex. The first time a feature legitimately needs a file,
that reflex either wrongly blocks the feature or is quietly abandoned, and next time nobody
remembers which. Naming the CA bundle a **trust anchor** and running it through §5.1's own
audit-trail test records that this one file is admitted **because** it fails the test that
defines the forbidden class, not in spite of the rule. The rule keeps its teeth; the
exception is principled and bounded, not a precedent for mounting arbitrary config.

### Why the override, not an inert base mount

An inert bind in the base — harmless until `POSTGRES_SSLROOTCERT` is set — sounds tidier
because it keeps one file. It is not worth it. It needs a placeholder default source so the
base still boots with no bundle present, and that placeholder is exactly the kind of
hard-to-read hack a later session deletes without understanding. It also puts a new mount in
the file the CI `compose` job boots, next to the three-service assertion, for a code path
that job never exercises. Scoping the mount to the override keeps the base file — and the CI
contract — exactly as they are.

### Why a public trust anchor is not the secret ADR-0053 guards

[ADR-0053](./0053-a-secret-is-held-only-where-its-act-is-performed-and-the-shared-store-holds-none.md)
keeps the session key and the prober key out of Postgres and out of any backup because a
read-only leak of the database would otherwise mint live sessions or hand over a machine. A
CA **root** is the opposite kind of bytes: it is published so that anyone can validate
against it. Mounting it read-only creates no secret to leak and no keyring to guard, so it
sits comfortably beside the existing non-secret mounts without touching ADR-0053's property.

## Consequences

- **The SPEC's mount-convention text is now fixed** (map ticket
  [#951](https://github.com/winniel123/verge-asm/issues/951)): `docs/spec/packaging-and-configuration.md`
  §5.1 gains a reconciling note that a **trust anchor** is admitted as a read-only mount, and
  the override is specified to bind the operator bundle to `/etc/verge/ca/root.crt` with
  `PGSSLROOTCERT` set to it. `running.md`'s "no config file to mount" line gains the same
  reconciliation.
- **The live-TLS CI job's `sslrootcert` path is settled** for the still-fogged CI-shape work:
  it mounts the throwaway server's CA at `/etc/verge/ca/root.crt` through the override.
- **§5.1 is preserved, not withdrawn.** Its rule stands unaltered; it gains one dated
  forward-pointer recording that ADR-0132 admits a trust anchor as a distinct kind of file
  that passes §5.1's own audit-trail test.
- **No base compose change and no CI-invariant change.** The three-service `compose` job is
  untouched because the mount lives only in the override.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Add the bind to the base compose, inert until `POSTGRES_SSLROOTCERT` is set** | Needs a placeholder default source (a `/dev/null`-style hack) so the base still boots with no bundle, and puts a new mount in the file the CI `compose` job boots beside its three-service assertion — for a path that job never runs. The override carries it with no base change |
| **Take bundled Postgres to TLS on the compose network** | Out of scope for map #947, and the operator chose plaintext (`sslmode=disable`) for the in-compose hop. A CA anchor is only needed against an external database, which is exactly the override |
| **Mount `/app/ca/root.crt`, beside the existing `/app/state` mounts** | Those are app **state** volumes. A trust anchor is system material, not app state; `/etc/verge/ca/` reads correctly and the `root.crt` name matches libpq's own default |
| **Bind the containing directory `/etc/verge/ca:ro`** | Widens the mount for no gain. An operator with several roots concatenates them into the one PEM bundle `PGSSLROOTCERT` already reads. The single-file bind is the tightest surface |
| **Ship a CA bundle inside the image** | [ADR-0001](./0001-stack-and-runtime.md)'s distroless image carries no operator trust material, and a baked-in bundle would be stale the moment the operator's database rotated its CA. The operator supplies their own from the host |
| **Reopen §5.1 to permit config files** | The file is admitted **because** it fails §5.1's audit-trail test that defines the forbidden class, not by weakening the rule. §5.1 stands; only a trust anchor is named as a distinct kind of file |
