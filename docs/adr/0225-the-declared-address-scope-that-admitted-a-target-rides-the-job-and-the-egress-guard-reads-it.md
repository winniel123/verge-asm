---
number: 225
title: "The declared address scope that admitted a target rides the job, the egress guard reads that scope at the socket, and a discovered address stays refused"
slug: the-declared-address-scope-that-admitted-a-target-rides-the-job-and-the-egress-guard-reads-it
date: 2026-09-07
status: accepted
source: fix
ticket: 1610
proof: {none: "predates the governance SPEC"}
relations:
  - {kind: rests-on, adr: 121}
  - {kind: rests-on, adr: 79}
  - {kind: amends, adr: 79}
  - {kind: amends, adr: 222, clause: "3"}
  - {kind: amends, adr: 222, clause: "4"}
---

# ADR-0225: The declared address scope that admitted a target rides the job, the egress guard reads that scope at the socket, and a discovered address stays refused

- **Amends:** ADR-0222 §3 and §4. ~~That file is not on this branch.~~ **Applied at ADR-0222's own site by [#1626](https://github.com/winniel123/verge-asm/issues/1626).** See "The amendment this branch cannot make"

## Context

`custody.EgressGuard` returns a `net.Dialer.Control` function. It reads the dialed address
alone. It refuses any address `IsNonGloballyReachable` marks.

ADR-0079 rules that a non-globally-reachable address is connected to where a declared address
scope covers it and the vantage is not `internet`-class. Its worked table states the case:
*"Operator declares `10.0.0.0/24`; instance is the only prober → **Probed, unchanged.** The
internal estate keeps its measurements."*

The shipped binary does not do that. The guard closes the route at the socket.

### Measured by symbol on `38eec12`

An `Estate` holding one address scope of `10.0.0.0/24`:

| Call | Result |
| --- | --- |
| `Estate.Derive(10.0.0.5)` | `operator` |
| `Estate.MayProbe(10.0.0.5, ClassUnverified)` | `true` |
| `Estate.MayProbe(10.0.0.5, ClassInternal)` | `true` |
| `Estate.MayProbe(10.0.0.5, ClassInternet)` | `false` |
| `EgressGuard("connectoutcome")("tcp", "10.0.0.5:80", nil)` | refusal |

### The path is live

`BuildHotJobs` was run against that estate, one address of `10.0.0.5`, and one vantage named
`local` that presents no address. It yielded **one** `connect-outcome` job of class `unverified`
over **131** TCP ports. `exposure.VerifyClass` derives `unverified` because the vantage presents
no address, so `vc.IsInternet()` is false. Every one of those 131 dials reaches
`NetConnector.Connect`, and the guard refuses every one.

An operator who declares their office LAN gets `resolution` and `dns-record` at full aperture.
They get no connect measurement. Nothing in the console states the cause.

### The four dispatch gates and the four install sites are already symmetric

`Estate.MayProbe` is called at exactly four dispatch sites. `custody.EgressGuard` is installed at
exactly four socket sites. The two sets correspond.

| Dispatch gate | Socket install site |
| --- | --- |
| `scan.BuildHotJobs` | `connectoutcome.NetConnector.Connect` |
| `scan.BuildColdJobs` | `connectoutcome.NetHandshaker.Handshake` |
| `scan.BuildTLSAcceptanceJobs` | `tlsacceptance.NetEnumerator.Handshake` |
| `scan.BuildHTTPIdentityJobs` | `httpexchange.NetExchanger.Exchange` |

`internal/measure/edgefanout` reuses `connectoutcome.NetHandshaker` and is a fifth dial consumer.
It applies no custody gate, because it runs before its target is a member (ADR-0129).
`internal/measure/resolutionwalk` is a sixth dial path. ADR-0121 governs it.

### The precedent already ships a declared value through the job spec

ADR-0121 exempts the operator-declared recursive resolver from the guard. That resolver reaches
the prober as `scope.Resolver`, a field of the JSON job spec that `cmd/prober` reads from stdin.
`NetPeer.Exchange` reads it and selects `trustedDialer()` in place of `custodyDialer()`.

So the job spec is already the channel that carries an operator-declared value to a dial decision.
This ADR uses the same channel for the second operator-declared value. It does not open a new one.

## Decision

> **A declared address scope is the operator's realm claim, so the scope that admitted a target
> rides the job that probes it, and the egress guard permits an address that same scope contains.
> The guard never accepts a verdict. It re-reads the declaration against the address it is about
> to dial. An address inside no declared scope stays refused at every one of the four install
> sites.**

Five limbs.

### 1. One read serves the gate and the guard, so the two cannot drift

`Estate.ProbeRealm(addr, vc) (netip.Prefix, bool)` replaces the body of `Estate.MayProbe`.
`MayProbe` is now the second result of `ProbeRealm`.

`ProbeRealm` returns a **valid** prefix only when it also returns **true**. A refused address
yields the zero `netip.Prefix`. A globally reachable address yields the zero `netip.Prefix` and
`true`, because it needs no realm.

Two properties follow, and both are structural.

- **No refused address can enter a realm.** The only source of a scope is `ProbeRealm`, and it
  supplies one only beside an admission.
- **A dispatch site that forgets the realm loses a measurement and opens no destination.** The
  job then carries no realm, and the guard refuses the dial exactly as today. The residue of an
  error runs toward refusal.

### 2. A `Realm` holds declared CIDRs alone, never a predicate and never a verdict

`custody.Realm` wraps a list of `netip.Prefix`. Its zero value contains nothing.

The guard takes a `Realm` and applies `Prefix.Contains` to the address it is about to dial. That
is the same matcher `Estate.coveringAddressScope` applies. There is one containment rule and one
column, which is the property ADR-0079 built and named: *"there is no seam where two readings
could drift apart"*.

The guard takes no callback and no boolean. A caller cannot hand it a rule. A caller can hand it
only CIDRs an operator typed, and the guard still tests the address against them.

### 3. The realm rides `wire.JobSpec`, because the realm is a dispatch fact and not a leaf fact

`wire.JobSpec` gains one field, `Realm []string`. Four leaf scopes would have been four places to
forget. `Batch` already sits at this level for the same reason.

The field is rendered by `Realm.CIDRs` at dispatch. It is read by `custody.ParseRealm` in each
leaf's `Run`. A malformed entry drops rather than widening. An absent field parses to the zero
`Realm`, so a job row written before this ADR keeps today's behaviour. No migration is needed,
because `jobs.spec` already stores the marshalled `wire.JobSpec` as JSONB.

### 4. The seam at each install site is an unexported field, so the compiler holds it

Each of the four dialer types gains an unexported `realm custody.Realm` field. Only the leaf's own
package sets it, and only from `spec.Realm`.

ADR-0222 §1 rules that a dial-control seam is an unexported field whose zero value installs the
guard. This ADR obeys that rule rather than amending it. Three properties hold, and the compiler
enforces all three.

- A package outside the leaf cannot set the field.
- The zero value exempts nothing, so a caller that supplies no realm gets the whole guard.
- `edgefanout` builds `connectoutcome.NetHandshaker` from outside that package, so it carries the
  zero `Realm` and its guard stays unconditional.

`internal/measure/realmseam_test.go` pins all four sites.
`TestEveryInstallSiteHoldsTheRealmSeamUnexported` reads the four dialer types by reflection. It
fails when a site lacks the field, when the field has the wrong type, and when the field is
exported. `TestAnOutsideCallerGetsNoRealm` asserts that a dialer built outside its package carries
no realm.

### 5. A discovered address stays refused, and the tests pin it

The distinction is trust origin, as ADR-0121 rules. The trusted value is the **CIDR**, not the
address. An address is admitted because an operator typed a CIDR that contains it.

`internal/custody/proberealm_test.go` holds the boundary test.
`TestADiscoveredPrivateAddressGetsNoRealmAndStaysRefused` builds an estate with a custody extension
over `example.com` and one declared scope of `10.0.0.0/24`. The zone publishes `192.168.1.7`,
`169.254.169.254` and `127.0.0.1`. For each address and for all three vantage classes, the test
asserts three things.

- `Derive` returns `third-party`, because an extension declares no realm (ADR-0079).
- `ProbeRealm` returns `false` and the zero `netip.Prefix`.
- `EgressGuard`, holding the legitimate `10.0.0.0/24` realm, refuses the dial.

`internal/scan/realm_test.go` pins the same refusal one level up.
`TestADiscoveredPrivateAddressGetsNoHotJob` and
`TestReachedServiceJobsOverADiscoveredPrivateAddressAreNotDispatched` assert that no job is built
at all.

## What widens, stated exactly

**Before this ADR** a shipped binary dialed globally reachable addresses alone.

**After this ADR** a shipped binary dials globally reachable addresses, plus an address that
satisfies all three of these conditions at once.

1. A declared address scope contains the address.
2. The custody gate admitted the address from a vantage that is not `internet`-class.
3. The job that carries the dial carries that same declared scope.

That set is exactly ADR-0079's ruling. The binary was **stricter** than the ruling, and silently
so. This ADR moves the binary to the ruling and no further.

**What stays refused**, unchanged, at all four sites:

- Any non-globally-reachable address inside no declared address scope.
- Any address reached through a custody extension alone, because `extensionReaches` already
  refuses a non-globally-reachable address.
- Any address dialed from an `internet`-class vantage, because `ProbeRealm` yields no scope there.
- Any address dialed by `edgefanout`, which carries no realm.
- Any address dialed by a leaf whose job spec carries no realm.
- Any non-literal host, which `net.SplitHostPort` and `netip.ParseAddr` reject before the realm is
  consulted.

## The residues, stated rather than hidden

**An operator can declare a scope that contains a hazardous address.** `169.254.169.254` sits
inside `169.254.168.0/22`. ADR-0049's cap of 1,024 addresses admits a `/22`, so that scope is
declarable. An operator who declares it, and who runs the prober on a cloud instance, retrieves
instance metadata into `http-identity`.

This residue is not new and it is not widened by this ADR. ADR-0079 Route 2 already ruled that a
declared address scope opens the gate over every address it covers. This ADR makes that ruling
reach the socket. The act is a typed CIDR on a surface the operator authored, bounded by a cap, and
visible in the console. It is the class of error ADR-0013's Consequences say the model *"has never
tried to prevent, because it cannot prevent a false name-scope seed either"*.

**The realm travels in the job spec.** An actor who can write a job row, or who can write to the
prober's stdin, can name a CIDR. That actor already administers the instance or already controls
the SSH channel to the prober. This is the same reach ADR-0121 accepted for the declared resolver,
and its Rationale prices it: *"An actor who can set it already administers the instance."* The
value never flows from scan data, a proposer result, or a request surface.

**The guard is no longer independent of the job spec for a declared address.** For an address
inside a declared scope, the gate and the guard now agree by construction, so the guard catches no
error the gate makes about that address. For every other address the guard is unchanged and
remains the unconditional backstop. The set where the backstop is retired is the set the operator
declared.

## Consequences

- **`internal/custody` gains `ProbeRealm`, `Realm`, `Realm.With`, `Realm.Contains`, `Realm.CIDRs`
  and `ParseRealm`.** `MayProbe` keeps its signature and its meaning.
- **`custody.EgressGuard` takes a second argument.** The zero `Realm` reproduces the old
  behaviour, so every call site states its realm explicitly.
- **`wire.JobSpec` gains `Realm []string`.** No migration, and no `sqlc` regeneration.
- **Four job builders stamp the realm**, at the same statement that gates.
- **Four dialer types gain an unexported `realm` field.** The seam is the one ADR-0222 §1 rules.
- **No emitted observation changes and no golden corpus moves.** The change is confined to the
  dispatch stamp and the live network adapter. The hermetic corpora script their own connectors
  and never dial.
- **Aperture inputs stay at seven.** The realm records no new dimension. It re-states, at the
  socket, the declaration the gate already read. The `Batch` scope record is by content and is
  unchanged.
- **The v1 rule set stays at seventeen**, and `Custody` gains no value.
- **`CONTEXT.md` gains nothing.** No domain term moves. `Realm` is the word ADR-0079 already uses.
- **ADR-0222's reopening condition is met.** Its §3 named the captured-public-prefix rig as the
  supported one for an end-to-end leaf measurement. A declared `127.0.0.0/22` now reaches a
  loopback listener through the production path, which `internal/measure/connectoutcome/realm_net_test.go`
  demonstrates without any test-only seam.

## Reopening condition

This ADR is reopened by one event: **ADR-0079 withdraws Route 2**, so that a declared address
scope no longer opens the custody gate over a non-globally-reachable address. The realm then has
nothing to carry, and every limb here is deleted rather than amended.

It is **not** reopened by a session that wants the guard to read something other than a declared
address scope. §2 refuses that shape, and the refusal depends on no fact a later tree can change.

It is **not** reopened by a session that finds a fifth dial site. §1's rule governs it. A site
that stamps no realm refuses exactly as today, so adding one is safe and is a one-field change.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Amend ADR-0079's worked table**, so the declared-scope route no longer reaches a connect | It is the legitimate other branch, and it is refused on the measurement. ADR-0079's cost section names declaring an address scope as the repair for the extension case. If that repair reaches no connect, the cost section is false, and ADR-0079 silently adopts the outcome its own *"Bar only the internet class"* and *"Require the class to be internal"* rows reject. Those rows say the result *"deletes internal probing of private space on every install without a prober — which is most of them."* An ADR must not reach a result it rejected in writing |
| **Carry a boolean verdict on the job**, so the guard skips the check when the dispatcher says so | The guard would then accept a claim rather than make a reading. A wrong verdict would exempt every address in the job. §2's `Realm` makes the guard re-test the address against the CIDR, so a wrong realm exempts only what that CIDR covers |
| **Give the guard a `func(netip.Addr) bool` predicate** | It widens the guard into an arbitrary predicate, which the ticket's constraint 3 forbids and which ADR-0222's rejected-alternatives table already prices. A predicate is a rule a caller supplies. A `Realm` is data the operator declared, read by the one rule |
| **Carry every declared address scope on every job** | A job over a public target would then exempt the operator's whole private space for no reason. §1 carries the scope that admitted this job's target, and nothing else |
| **Read the realm from the prober host's own interfaces** | It is a realm taxonomy we author, which ADR-0079 refuses in its *"The population is the owner's column and nothing finer"* section. It also gives an SSH-run external prober a different answer from a local one |
| **Site the realm in each of the four leaf scopes** | Four wire fields and four places to forget, which is the drift the ticket's constraint 4 names. The realm is kind-agnostic, so it belongs beside `Batch` |
| **Exempt loopback, so a Go test can reach a listener** | ADR-0222 refused this and its ground holds. Loopback is the address the guard most needs to refuse. This ADR reaches a loopback listener only through a declared `127.0.0.0/22`, which is an operator act and not a carve-out |
| **Add the realm to `edgefanout` as well** | It applies no custody gate, so it has no admitting scope to carry. A realm with no gate behind it is an exemption with no author |
| **Repair `connectoutcome` alone**, since it carries `Reach` | Three of the four sites would then still refuse the operator's declared LAN. `tls-acceptance` and `http-identity` both ride a reached `Service`, so the operator would get a connect and nothing else |

## Amendment to ADR-0079's Consequences — two sentences are withdrawn

ADR-0079's Consequences state two things that this ADR makes false. ADR-0058 applies, so they are
withdrawn at the site that specifies them and the replacement is stated.

**First.** *"**Nothing in the measurement binary changes**, because nothing implements the current
behaviour."* That sentence was true of ADR-0079's own repair. It is false of ADR-0079's ruling as
a whole, because nothing in the measurement binary implemented the **permission** either. The
measurement binary now changes. Four dialer types read a realm, and four job builders stamp one.

**Second.** *"**The shipped configuration sends packets to strictly fewer destinations.** Every
clause here removes a destination and none adds one."* That sentence is true of ADR-0079 measured
against the tree that preceded it. It is false of this ADR measured against the shipped binary.
This ADR adds destinations, and "What widens, stated exactly" above enumerates them. The
destinations it adds are the ones ADR-0079's Route 2 always permitted and the guard always refused.

ADR-0079's worked table is **not** amended. Its row *"Operator declares `10.0.0.0/24`; instance is
the only prober → **Probed, unchanged.**"* stated the ruling correctly. This ADR makes the shipped
binary agree with it.

## The amendment this branch cannot make

ADR-0222 §4 files this contradiction and refuses to settle it. Its Consequences say *"The mismatch
stays live until #1610 lands."* Its Reopening condition names this exact ruling and says its §3 is
*"superseded rather than amended"*.

ADR-0222 arrives on `main` from PR [#1614](https://github.com/winniel123/verge-asm/pull/1614),
which is open. This branch was cut from `origin/main` before that PR landed, so the file is not
here. This ADR does not create a copy of it.

The `adr-sections` gate reports four `unresolvable-adr` violations against this file while ADR-0222
is absent. The gate is not a required check. The four violations clear when PR #1614 merges and
this branch is updated.

The amendment is **owed** and is recorded here at the superseding site. It must be applied at
ADR-0222's own site, per ADR-0058, once PR #1614 merges. It has three parts.

1. §4 is discharged. The mismatch is repaired by this ADR.
2. §3 is superseded. The captured-public-prefix rig is no longer the only supported way to run a
   leaf measurement end to end. A declared address scope is the other way.
3. The Consequences sentence *"The mismatch stays live until #1610 lands"* is withdrawn.

> **DISCHARGED by [#1626](https://github.com/winniel123/verge-asm/issues/1626).** PR #1614 merged, and the
> three parts above are applied at ADR-0222's own site. The four `unresolvable-adr` violations
> cleared with that merge, and the gate is green. This section stays as the record of the debt and
> of its payment.
