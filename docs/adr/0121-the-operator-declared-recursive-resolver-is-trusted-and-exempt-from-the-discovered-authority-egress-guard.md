# ADR-0121: The operator-declared recursive resolver is trusted, and exempt from the discovered-authority egress guard

- **Status:** Accepted
- **Date:** 2026-08-26
- **Ticket:** [#612 DNS scans still dead-letter out of the box: default vantage resolver 127.0.0.11:53 is refused by the #335 loopback egress guard](https://github.com/winniel123/verge-asm/issues/612)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

Two individually-correct decisions collided, and their intersection broke every default install.

- [#239](https://github.com/winniel123/verge-asm/issues/239) set the shipped `local` vantage's
  recursive resolver to `127.0.0.11:53` (Docker's embedded DNS), so the only deployment the docs
  describe — `docker compose` — resolves out of the box
  ([ADR-0036](./0036-a-shipped-default-is-the-configuration-that-takes-effect.md): the default that
  ships is the one that runs).
- [#335](https://github.com/winniel123/verge-asm/issues/335) hardened the measurement dial path
  against SSRF/DNS-rebinding: `resolutionwalk`'s dialer carries a Control hook that refuses to open a
  socket to any non-globally-reachable address, a rebinding-proof backstop mirroring the delivery
  runner's hook ([#325](https://github.com/winniel123/verge-asm/issues/325)). It followed
  [#324](https://github.com/winniel123/verge-asm/issues/324), the pre-flight vet that bars a walk
  authority whose name resolves into a non-globally-reachable range.

The `#335` Control hook was installed on the *shared* dialer used for every exchange, so it also fired
on the dial to the declared resolver. `127.0.0.11` is inside `127.0.0.0/8` — non-globally-reachable —
so the hook refused it, `Exchange` reported `Unreachable`, and per
[ADR-0108](./0108-a-batch-whose-instrument-could-not-reach-its-position-covers-nothing-and-the-failure-is-the-vantages.md)
the `resolution-walk` batch aborted, retried through the back-off ladder, dead-lettered, and marked
the vantage unavailable. Every fresh `docker compose` install's `dns` scan failed deterministically.
This made the `#239` fix a no-op at runtime: it swapped one refused loopback (`127.0.0.1`) for another
(`127.0.0.11`) that the later-merged `#335` guard equally refuses — `#239`'s own triage note records
that a runtime repro was not run against the egress block.

The genuinely-undecided question: **does the custody egress guard apply to the resolver the operator
declared, or only to the authorities the walk discovers?**

## Decision

**The Vantage's recursive resolver is trusted configuration, not attacker-influenced data, so it is
exempt from the SSRF/rebinding egress guard. The guard applies only to *discovered* delegation-walk
authorities. A dial to the declared resolver is subject to neither the `#324` pre-flight vet nor the
`#335` Control-hooked dialer; a dial to a discovered authority remains subject to both.**

The distinction is trust origin, not address.

- The **declared resolver** is part of the Vantage's identity
  ([ADR-0070](./0070-a-control-probe-is-asked-from-where-the-answer-it-discriminates-was-asked-from.md)),
  supplied out of band by an operator who already controls the deployment. An operator who points it
  at a loopback or private-LAN address is configuring their own infrastructure — there is no
  privilege boundary to cross and no attacker-controlled input in the value, so gating it guards
  nothing and only breaks legitimate deployments (compose's `127.0.0.11`, a bare-metal `10.0.0.53`).
- A **discovered walk authority** is named verbatim in NS RDATA that an in-scope but hostile zone
  controls (`leaf.go`'s `walk` sets `Query.Server = rr.Data`). That is exactly the population `#324`
  and `#335` exist to stop from reaching `169.254.169.254`, `127.0.0.1`, an RFC1918/ULA host, or an
  internal name. Its guard is unchanged, including the rebinding backstop.

The two dials to the declared resolver — every declared-path query, and the delegation walk's
**initial NS query**, which carries no `Server` and is therefore asked of the resolver — are both
exempt. Only a query naming a discovered authority (non-empty `Server`) is gated. In `NetPeer.Exchange`
this is the single predicate `dialingResolver := q.Path == PathDeclared || q.Server == ""`, which
selects both the resolver as the target and a plain `trustedDialer` in place of the `custodyDialer`,
and skips the pre-flight vet.

## Rationale

### The guard's own threat model already scoped it to discovered targets

`#324`/`#335` were written against attacker-controlled RDATA: the pre-flight vet is `PathWalk`-only by
construction, and the `#335` comment justifies the backstop entirely in terms of a walk authority's
name rebinding between the vet and the dial. Catching the declared resolver was an unintended side
effect of installing the Control hook on the shared dialer, not a considered decision that the
resolver needed guarding. This ADR restores the asymmetry the guard was designed to have.

### Exempting the resolver grants an attacker no new capability

The resolver value flows only from the `vantage.resolver` column, set by an admin via migration or the
vantage UI — never from scan data, proposer results, or any request surface. An actor who can set it
already administers the instance. So the exemption widens no attacker's reach; it only lets a
legitimate operator point the instrument at the recursive resolver their network actually uses,
including the private ones that are the norm on compose and bare metal.

### This is the fix that keeps #239's goal without weakening security

The alternatives considered in triage each cost something this one does not (see the table). Exempting
the declared resolver restores the out-of-the-box `docker compose` experience `#239` promised, serves
bare-metal/host-network installs whose resolver is legitimately on a private LAN, and leaves the
attacker-facing walk path — the only path that carries untrusted target addresses — guarded by both
`#324` and `#335`, verified by the unchanged `TestWalkRefusesCloudMetadataAuthority`,
`TestWalkServerReachableGate`, `TestCustodyDialerControlRefusesNonGlobal`, and
`TestExchangeRefusesRebindToPrivate`.

## Consequences

- **`internal/measure/resolutionwalk/netpeer.go`** splits the dialer by trust origin: `trustedDialer`
  (no Control hook) for the declared resolver, `custodyDialer` (the `#335` backstop) for discovered
  authorities. The pre-flight vet condition changes from `q.Path == PathWalk` to `!dialingResolver`,
  which additionally exempts the walk's initial NS query (a resolver dial) that the old condition
  wrongly gated.
- **The shipped default `127.0.0.11:53` now works at runtime.** The seed value from `#239` is
  unchanged; `db/migrations/18800_measurement_vantage.sql` gains a comment recording that the loopback
  value is deliberate and must not be "fixed" back to a public resolver.
- **No emitted observation changes and no golden corpus moves.** The change is confined to the live
  network adapter; the hermetic corpus scripts its own peer and never dials, so every fold is
  byte-identical (`TestCorpusLock` green).
- **The attacker-facing walk path is unchanged.** Both custody guards remain on every discovered
  authority; the SSRF/rebinding regression tests pass unmodified.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Seed a public resolver (e.g. `1.1.1.1:53`) as the shipped default** | Routes every default install's DNS measurement through a third party, requires outbound UDP/53 the deployment may not have, and still leaves any operator with a private-LAN resolver blocked by the guard. Treats the symptom (this one address) not the collision (the guard scope) |
| **Ship `local` with no resolver and require the operator to declare one** | Regresses `#239`'s explicit goal — real observations on the first `dns` trigger with no manual step — and adds a UI-surfacing task, all to avoid a guard that should not have fired on trusted config in the first place |
| **Exempt only the loopback range for the declared resolver** | Ad hoc: it fixes compose but still blocks a legitimate bare-metal RFC1918 resolver, and it keys the exemption on the address rather than the trust origin, which is the actual distinction |
| **Suppress the CodeQL/guard finding and keep gating the resolver** | Keeps the product broken out of the box to satisfy a guard aimed at a different population; the guard is right for discovered authorities and wrong for the declared resolver, so the fix is to scope it, not to silence it |
