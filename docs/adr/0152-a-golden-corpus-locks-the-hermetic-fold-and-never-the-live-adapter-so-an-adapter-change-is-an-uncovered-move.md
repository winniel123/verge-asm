# ADR-0152: a golden corpus locks the hermetic fold and never the live adapter, so an adapter change is an uncovered move

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1298 ADR gaps: internal/measure/connectoutcome (1/2)](https://github.com/winniel123/verge-asm/issues/1298), gap 2
- **PR that deleted the comment:** [#1297](https://github.com/winniel123/verge-asm/pull/1297)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Also stated, uncited, at two other leaves:**
  [`tlsacceptance/enumerate.go:100`](../../internal/measure/tlsacceptance/enumerate.go) and
  [`resolutionwalk/netpeer.go:40`](../../internal/measure/resolutionwalk/netpeer.go). One rule, three
  wordings, no record
- **Rests on:** [ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md), whose gate runs both
  ways and whose third limb is the recorded uncovered move, and
  [ADR-0008](./0008-derivation-versions-move-on-content.md), under which a version moves on content
- **Adjacent:** [ADR-0142](./0142-a-corpus-input-is-generated-deterministically-and-the-random-draw-is-productions-alone.md)
  (the corpus's own generated input) and
  [ADR-0140](./0140-a-network-seam-is-a-runtime-parameter-the-caller-supplies-never-a-build-tag-and-never-a-hardcoded-client.md)
  (the adapter is a runtime parameter the caller passes)

## Context

[`internal/measure/connectoutcome/tls.go:248`](../../internal/measure/connectoutcome/tls.go) carries
this, above `classifyDialError`:

```go
// The split is best-effort and unversioned by the corpus — the golden rows pin the fold,
// not this live classification.
```

`classifyDialError` maps a failed TLS dial to a `TLSOutcome` and to the `Unreachable` flag
[ADR-0151](./0151-a-field-no-emitter-renders-is-cross-leaf-plumbing-and-plumbing-moves-no-leaf-version.md)
rules. It is called from one place, `NetHandshaker.Handshake` at
[`tls.go:148`](../../internal/measure/connectoutcome/tls.go).

**Five leaves have this shape, and the corpus swaps the adapter out in every one.**

| Leaf | Live adapter | What a corpus row runs instead |
| --- | --- | --- |
| `tls-handshake`, certificate block | `NetHandshaker` ([`tls.go:121`](../../internal/measure/connectoutcome/tls.go)) | `certcorpus`'s `scriptHandshaker`, through `RunExchange` |
| `connect-outcome` | `NetConnector` ([`leaf.go:68`](../../internal/measure/connectoutcome/leaf.go)) | `corpus`'s `scriptConnector`, through `RunWithConnector` |
| `http-exchange` | `NetExchanger` ([`exchange.go:117`](../../internal/measure/httpexchange/exchange.go)) | `corpus`'s `scriptExchanger`, through `RunWithExchanger` |
| `tls-acceptance` | `NetEnumerator` ([`enumerate.go:102`](../../internal/measure/tlsacceptance/enumerate.go)) | `corpus`'s `scriptEnumerator`, through `RunWithEnumerator` |
| `resolution-walk` | `NetPeer` ([`netpeer.go:20`](../../internal/measure/resolutionwalk/netpeer.go)) | `corpus`'s scripted `Peer`, through `RunWithPeer` |

So no line of any live adapter runs inside a corpus render. `RenderRow` calls the leaf with the
scripted seam, and `CorpusDigest` hashes what the leaf wrote. The adapter is not in the bytes.

**Three of the five sites say so, uncited, in three wordings.**

- `connectoutcome/tls.go:248` — *best-effort and unversioned by the corpus*.
- `tlsacceptance/enumerate.go:100` — *the hermetic golden corpus pins the accept-fold, not this
  path, so its errors are best-effort*.
- `resolutionwalk/netpeer.go:40` — *the golden corpus scripts an in-process Peer, so nothing here is
  exercised by it*.

**ADR-0021 rules the near half and never the far half.** Its Decision table fixes the corpus medium
as authored and fixes the corpus as hermetic against an in-process scripted peer. Both rows are
about what the leaf *talks to*. Neither says what the leaf's own network adapter is then gated by.
ADR-0142 already found one blind spot on the same clause and qualified it there. This is a second
one, in the opposite direction: ADR-0142 asks what enters a render, and this ADR asks what a render
can never reach.

**The gate's third limb already exists and is empty.** ADR-0021 rules that a leaf's version may move
only where a corpus row's output moved, a declared parameter changed, or an **uncovered move** was
recorded naming the input class the corpus cannot reach.
[`golden-corpus.md`](../spec/golden-corpus.md) §9 gives that record its fields, its validity rule and
its table. §9.4 records none, and §9.3 states that a row for `tls-handshake` is legal checked-in data
with no consumer yet.

**The wording that was deleted invites one over-read.** *Unversioned by the corpus* is true. *Needs
no version bump* does not follow, and ADR-0021 already says the opposite for the release where an
author cannot tell: **the honest default is to bump**. A change to `classifyDialError` can move a
real host's `certificate` value from `no-tls` to `tls-refused`, and under ADR-0008 that is exactly
the change a version must carry.

**A unit test pins the split and is not the gate.**
`TestClassifyDialErrorSplitsThePhase` in
[`dialphase_test.go:14`](../../internal/measure/connectoutcome/dialphase_test.go) enumerates the
error shapes. It catches a regression. It moves no digest, so it can never fail the version gate.

## Decision

> **A golden corpus row locks the hermetic fold alone. Every live network adapter is swapped out
> before a row renders, so no row and no digest can catch a change to one. That is a hole in the
> gate and not a licence. A change to a live adapter's classification that can move a value in
> production is an **uncovered move**: it bumps the leaf and it is recorded in
> [`golden-corpus.md`](../spec/golden-corpus.md) §9 against that leaf and its new version.**

Five limbs.

### 1. What a row locks

A corpus row locks the leaf's fold: the code between the seam and the emitted NDJSON. That is the
part the row's claim is a sentence about, and it is the part `CorpusDigest` hashes.

A row locks nothing on the far side of the seam. The scripted peer decides the row's input, so the
row is silent about the adapter that decides production's input.

### 2. *Best-effort* is a statement about the corpus, never about the version

A comment at an adapter may say the corpus does not pin it. That sentence is true and it is the
whole of what it says.

It does not say the classification is unimportant. It does not say a change to it is free. It does
not license a leaf version to stand still while the leaf's production output moves. **A session
reading *unversioned by the corpus* as *needs no bump* has read a fact about the gate as a
permission.**

### 3. The route is the uncovered move, and the class is the wire

An adapter change that can move a production value takes ADR-0021's third limb. The `Input class`
field names the class the corpus cannot reach — for `classifyDialError`, the shapes of dial failure
a real kernel and `crypto/tls` return. The row licenses one `(Leaf, Bumped to)` pair, is present by
the commit the gate evaluates, and is append-only, exactly as §9.2 states.

A change that cannot move a production value needs no row and no bump. A rename, a rewritten switch
with the same mapping, and a new branch for an error shape that already fell through to the same
answer are all of that kind.

### 4. A test at the adapter is worth having and is not the gate

The adapter deserves its unit test, and `connect-outcome` has one. The test is what makes an
intended change legible to a reviewer.

It is not the version gate, because the gate reads three things and a unit test is none of them. Do
not let a green adapter test stand in for the record §9 wants.

### 5. The rule binds every live adapter under `internal/measure`

The five in the table above, and any adapter a later leaf adds. The shape is the same everywhere: a
`Net*` type behind an interface the corpus scripts. The rule follows the shape and not the package.

## Consequences

- **[`tls.go:248`](../../internal/measure/connectoutcome/tls.go) states the corpus half and cites
  this ADR.** The word *unversioned* leaves, because it is the half a later session over-reads.
- **The `tls-acceptance` and `resolution-walk` sites state the same rule uncited.** They sit outside
  #1298's scope and are untouched here. Each takes the citation when its own ticket touches it.
- **[`golden-corpus.md`](../spec/golden-corpus.md) gains no row.** No adapter has changed and §9.4
  stays empty. The section already admits a `tls-handshake` row, so no amendment is owed.
- **No production behaviour changes, and no `corpus.lock.json` moves.** This ADR rules what a future
  change owes.
- **[ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md) gains one bullet** in its
  metadata, naming this ADR beside ADR-0142 as the second reading its hermeticity clause does not
  reach. [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)
  wants the bound at the site that states the clause.
- **[`CONTEXT.md`](../../CONTEXT.md) gains nothing.** A live adapter is a seam and not a domain term.
- **One question is left open**, and this ADR does not settle it. ADR-0021 makes `crypto/tls` a
  declared parameter of `tls-handshake` where the library speaks the protocol for us.
  `HandshakeParams` carries no library version, so `ParamsDigest` does not move on a Go upgrade. How
  that parameter is recorded is a separate ticket.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Read the deleted comment as written** — the split is unversioned, so a change to it is free | It reads a fact about the corpus as a permission about the version. Under [ADR-0008](./0008-derivation-versions-move-on-content.md) a production value that moves without a bump is compared across a change we made, estate-wide and silently. That is the one failure the whole version apparatus exists to prevent |
| **Cover the adapter with corpus rows** — run a real dial in the corpus | It is live listeners in containers, which ADR-0021 refuses on three grounds: the expected output moves when the fixture image moves, hostile peers are not configurations of real software, and #31 already failed to start Docker on the machine that needed it |
| **Make a unit test the gate** for the adapter | A unit test moves no digest, so a bump it should have forced still passes CI. It also pins the mapping this project chose rather than the class of input the corpus cannot reach, which is what §9's row is for |
| **Widen the params digest** to cover the adapter's mapping | It is a hash of the code by another name — [ADR-0008](./0008-derivation-versions-move-on-content.md)'s rejected content hash, which bumps the estate for a rename |
| **Say nothing and leave the boundary to each site** | The state three sites were already in. Three wordings of one rule, one of which invites the over-read limb 2 refuses, and no record anywhere |
| **Amend [`golden-corpus.md`](../spec/golden-corpus.md)** instead of writing an ADR | §1 of that file discharges `resolution-walk`, `wildcard-discrimination` and `Custody`. This rule binds five live adapters across four other leaves, so the amendment would grow past its own subject — the same ground [ADR-0142](./0142-a-corpus-input-is-generated-deterministically-and-the-random-draw-is-productions-alone.md) gave |
| **A section on [ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md)** | Its subject is the version-leaf vector and the authored corpus medium. This rule is about the code the corpus can never run, and it names an obligation on a future change rather than a property of the corpus. ADR-0021 takes the bounding bullet its own clause needs and no more |
