# ADR-0151: a field no emitter renders is cross-leaf plumbing, and plumbing moves no leaf version

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1298 ADR gaps: internal/measure/connectoutcome (1/2)](https://github.com/winniel123/verge-asm/issues/1298), gap 1
- **PR that deleted the comment:** [#1297](https://github.com/winniel123/verge-asm/pull/1297)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Rests on:** [ADR-0011](./0011-a-facet-is-six-parts.md), which makes every facet's canonical form a
  closed tagged union and never a record with optional fields, and
  [ADR-0008](./0008-derivation-versions-move-on-content.md), which moves a leaf version on the
  derivation's output
- **Adjacent:** [ADR-0129](./0129-a-shared-foreign-edge-is-measured-by-fan-out-not-read-from-a-list.md)'s
  **#954 amendment**, *The measurement is membership-deciding, not a facet*, under which `edge-fanout`
  is not a facet and opens no timeline. Not §6, which rules the opposite way and which that amendment
  withdraws

## Context

`HandshakeResult` is one Go struct read by two leaves. `internal/measure/connectoutcome` emits the
`certificate` facet from it. `internal/measure/edgefanout` folds it into the `edge-fanout` leaf's
own outcome. The struct carries one field that is not a certificate fact.

[`internal/measure/connectoutcome/tls.go:85`](../../internal/measure/connectoutcome/tls.go) carried
this, until #1297 compressed it to one trailing line:

```go
// Unreachable reports that the dial failed in its CONNECT phase — the connect was
// refused or reset, it timed out, or the egress guard refused the socket — so no
// peer was ever reached, nothing spoke TLS and nothing turned us down. A handshake
// that fails after the connect completed is never unreachable, however it failed.
// It is carried only with a negative Outcome, and its zero value (false) reads as
// *a peer answered*, which is what a scripted Handshaker models.
//
// The `certificate` facet never reads it: that leaf handshakes a Service the connect
// already reported `reached`, so it folds an unreachable dial into `no-tls` and its
// two negatives stay two. A leaf that dials an address nobody has reached yet must
// keep the case apart — `edge-fanout` reads this to render its own `unreachable`
// value. It is never part of a facet value, so it moves no params digest and forces
// no CertVersion bump.
Unreachable bool
```

**Every claim in that block holds in the tree today.**

| Claim | Where it is checkable |
| --- | --- |
| The `certificate` value space has no `unreachable` variant | `certificateValue` in [`certificate.go:24`](../../internal/measure/connectoutcome/certificate.go) has nine JSON fields and no such field |
| The `certificate` handshake follows a `reached` connect | [`certificate.go:153`](../../internal/measure/connectoutcome/certificate.go) runs `Probe` first and skips the handshake unless the outcome is `Reached` |
| `edge-fanout` reads the field and renders its own value | [`edgefanout/leaf.go:62`](../../internal/measure/edgefanout/leaf.go) reads it ahead of the TLS negatives and returns `Unreachable` |
| The field moves no digest and no row | The word appears in no file under `connectoutcome/corpus` or `connectoutcome/certcorpus`, and in no field of `HandshakeParams` |
| The zero value models a peer that answered | `certcorpus`'s `scriptHandshaker` returns `HandshakeResult{Outcome: co.NoTLS}` and never sets the field |

**The same shape is already in a second leaf, and it is spelled differently.**
`resolutionwalk.Result.Unreachable` carries the tag `json:"-"`
([`resolutionwalk/leaf.go:98`](../../internal/measure/resolutionwalk/leaf.go)). It decides a
dead-letter at [`run.go:44`](../../internal/measure/resolutionwalk/run.go) and reaches no emitted
value. One leaf keeps the fact out of the value by omitting it from the emitter's struct. The other
keeps it out with a struct tag. Neither states the rule the two share.

**Nothing on disk states it.** `unreachable` under `docs/` names `Exposure`'s fourth cell
([ADR-0017](./0017-exposure-needs-both-legs.md)) and `edge-fanout`'s stored union
([`db/migrations/24500_edge_fanout_scan.sql`](../../db/migrations/24500_edge_fanout_scan.sql)).
`CONTEXT.md` names the `Exposure` value alone. ADR-0011 rules what a facet's value space is and
says nothing about the other fields of the Go struct an emitter reads.

## Decision

> **A field on a measurement leaf's result type is a facet value only where an emitter renders it.
> Every other field is cross-leaf plumbing. Plumbing moves no params digest, moves no corpus row,
> and forces no leaf-version bump. `HandshakeResult.Unreachable` is plumbing. A leaf that dials an
> address no connect has reached reads it. A leaf whose handshake follows a `reached` connect folds
> an unreachable dial into an existing negative and adds no value.**

Six limbs.

### 1. The emitter separates the two, not the struct

A leaf's result type is a Go convenience. It carries what the leaf learned. The facet value is what
the emitter writes, and it is the closed union ADR-0011 rules.

So the test is one grep, not a judgement. Read the emitter's value type. A field it renders is a
facet value. A field it does not render is plumbing, whatever the result type suggests.

`EmitCertificate` builds `certificateValue`. That type has no unreachable field, so the field is
plumbing on this leaf.

### 2. Plumbing moves no version

A leaf version moves on the leaf's output ([ADR-0008](./0008-derivation-versions-move-on-content.md)).
Plumbing reaches no output. Adding a plumbing field, removing one, or changing what one means moves
no corpus row and no params digest, so it forces no bump and opens no `Break`.

This is a consequence and not a licence. Limb 6 states the day it stops holding.

### 3. Only a leaf that dials an unreached address may read it

`edge-fanout` dials one address per candidate edge and no connect has reached that address first. A
failed dial there is a fact about the address, so the leaf keeps it apart and renders `unreachable`
as the fourth value of its own union.

The `certificate` facet is the other case. `RunExchange` handshakes a target only after `Probe`
returned `Reached`. An unreachable dial on that path is a Service that closed between two probes, or
the egress guard refusing the socket. It is not news about reachability, and the facet already
carries the reachability answer on its own `reachability` observation. So the leaf folds it into
`no-tls` and the facet keeps two negatives.

**A third leaf must choose one of these two.** It never invents a third treatment of the field.

### 4. The zero value reads as *a peer answered*

`false` is not *unknown*. It asserts that the dial completed and a peer was there.

This is what makes a scripted `Handshaker` correct without a line of its own. A fake that returns a
negative outcome and leaves the field alone models a peer that answered and refused. A fake that
means *nothing was there* sets the field.

### 5. The field reports the peer, never our own bad input

`Unreachable` is true for a dial that failed in its connect phase. The block above names the three
causes: a refusal or a reset, a timeout, and the egress guard refusing the socket. A handshake that
fails after the connect completed is never unreachable, however it failed.

An address this project got wrong is none of the three, and reporting it as unreachable would
manufacture a measurement.
[`resolutionwalk/netpeer.go:48`](../../internal/measure/resolutionwalk/netpeer.go) already rules this
way at its own site, on [ADR-0108](./0108-a-batch-whose-instrument-could-not-reach-its-position-covers-nothing-and-the-failure-is-the-vantages.md):
*a build error is our own bug, not the network's, so it is never `Unreachable`*.

**One site does not conform, and it is unreachable from production today.**
[`tls.go:131`](../../internal/measure/connectoutcome/tls.go) sets the field on its
invalid-address guard. Both production callers filter such an address before the call —
`Scope.targets()` in [`run.go:46`](../../internal/measure/connectoutcome/run.go) skips an address
`netip.ParseAddr` rejects, and `edgeTarget` in
[`edgefanout/run.go:59`](../../internal/measure/edgefanout/run.go) does the same — so no observation
renders from that branch. This ADR states the rule and changes no code. The branch is a defensive
backstop and a later ticket owns it.

### 6. The bound: an emitter that renders it ends the exemption

Limb 2 holds because no emitter renders the field. The day one does, the field is a facet value, and
every gate that guards a value guards it: a corpus row, the params digest where it becomes a
declared parameter, and a version bump when its rendering moves.

`edge-fanout` renders its own `unreachable` **outcome**, which is that leaf's value and not this
field. The two are one grep apart and must not be read as one thing.

## Consequences

- **[`tls.go:85`](../../internal/measure/connectoutcome/tls.go)'s surviving line gains this ADR's
  citation.** The line already states the operational half. It gained no ground and no citation from
  the sweep.
- **No production behaviour changes.** Both leaves already have the shape this ADR states.
- **`internal/measure/edgefanout` stays uncoupled from `connectoutcome`'s version lock.** It reads a
  field that is in no row and in no digest, so `CertVersion` moving does not reach it, and its own
  reading of the field moves nothing of `connectoutcome`'s.
- **`resolutionwalk`'s `json:"-"` gains nothing.** The tag already keeps the field out of the value,
  and that leaf sits outside #1298's scope. It takes the citation when its own ticket touches it.
- **[`CONTEXT.md`](../../CONTEXT.md) gains nothing.** `HandshakeResult` is a Go type and not a domain
  term. The glossary carries `Exposure`'s `unreachable`, which is a different object with the same
  spelling.
- **A new leaf that shares a result type owes limb 1's grep**, and a new plumbing field owes nothing
  else.
- **One follow-up is left open:** the invalid-address guard at `tls.go:131` sets a field limb 5 says
  it may not set.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Make `unreachable` a fourth `certificate` outcome**, so both leaves read one union | It records a reachability answer on a certificate timeline, where the `reachability` facet already carries one. Two representations of one fact, and the second one moves a version leaf every time the first one moves |
| **Give `edge-fanout` its own handshaker** rather than share `HandshakeResult` | The share is what brings `connectoutcome`'s egress guard along (#743, stated at `edgefanout/leaf.go:78`). A second dial path is a second place the guard can be forgotten, and the guard is the rebinding backstop |
| **Split the struct** into an emitted part and a transport part | Two types where the grep in limb 1 answers the question for nothing. It also forces every fake to construct two values, and limb 4's zero value stops reading as an assertion |
| **Rule that plumbing must never exist**, and let each leaf dial for itself | Deletes the seam [ADR-0140](./0140-a-network-seam-is-a-runtime-parameter-the-caller-supplies-never-a-build-tag-and-never-a-hardcoded-client.md) rules and the corpus depends on. A leaf with no shared result type has no scripted peer either |
| **Read the rule off `resolutionwalk`'s `json:"-"` tag** and require the tag everywhere | The tag works only where the result type is also the emitted type. `connectoutcome` emits a separate `certificateValue`, so there is no field to tag, and the convention would be unstatable at half the sites |
| **Leave it in a comment on the field** | It is 13 lines against a 25-word budget, the sweep cut it to one line, and the half it cut is the half that binds `edgefanout` and the version lock |
| **Fix `tls.go:131` in this ADR's ticket** | This ticket writes a record of a deleted decision. Changing a guard's return value is a code change with its own review, and the branch is unreachable from both production callers today |
