# ADR-0214: CT verification checks presence, never log integrity, so it authenticates no log and its tiled arm compares one slot

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1323 ADR gaps: internal/queue (queue, cttail, withdrawal, hot, ctverify, scopegate)](https://github.com/winniel123/verge-asm/issues/1323), gap 2
- **PR that deleted the comment:** [#1322](https://github.com/winniel123/verge-asm/pull/1322)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Withdraws a clause of:** [`ct-source-replacement.md`](../spec/ct-source-replacement.md) §5.1, which promises *"a self-recomputed tile inclusion proof (static-ct-api)"*. The tiled arm compares one slot in a hash tile. The sentence is withdrawn at its own site, per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md), and the replacement wording is recorded in this issue's manifest
- **Rules what [ADR-0190](./0190-the-ct-log-list-is-a-build-time-artefact-pinned-in-the-image-refreshed-only-by-a-release-and-carrying-no-log-public-keys.md) §5 took as a measurement.** That ADR strips the log public keys from the pinned snapshot, and its first ground is *"there is no verifier, so there is nothing a key would feed"*. It reads the absence of a signature check off the code. §1 and §2 below rule that absence as a scope decision, so its successor clause — *"a change that verifies a CT log signature re-embeds the keys, and it is a change to this Decision"* — is now bounded by a rule rather than by an observation
- **Rests on:** [ADR-0193](./0193-a-stapled-ocsp-response-only-narrows-the-sct-set-and-no-usable-sct-is-unverifiable-never-not-logged.md), which rules the verdict union: `not-logged` has exactly one warrant, an error outranks an absence, and `unverifiable` is silent at the operator surface. This ADR rules what the check **claims**. That one rules what it **says** when it cannot claim it. Gap 3 of this issue is that rule, and it is not restated here
- **Rests on:** [ADR-0027](./0027-a-source-may-admit-without-observing.md), which fences certificate material away from the `certificate` facet value. It is why a verification verdict is an ephemeral event and reaches no drift reading

## Context

[`internal/queue/cttail.go:179`](../../internal/queue/cttail.go) and
[`internal/queue/ctverify.go:31`](../../internal/queue/ctverify.go), `:233` and `:275` carried this
text, until #1322 deleted it:

```go
// it is NOT the full consistency proof: signature and inclusion verification
// stay deferred with #874's, the checkpoint kept verbatim to enable them later.
```

```go
// No log signature is checked — the root compared against is the one the log itself served (§4.4).
```

Nothing on disk states the rule, and the CT SPEC states something stronger than the code does. That
is #1323's gap 2.

**The two point-check arms do different amounts of work, and the deleted comment blurred them.**

| Arm | Site | What it fetches | What it proves |
| --- | --- | --- | --- |
| RFC 6962 | `ctverify.go:178` | `get-sth`, then `get-proof-by-hash` | `scan.VerifyInclusion` runs the RFC 6962 §2.1.1 inner/border recompute over the audit path, and compares the computed root against `sha256_root_hash` **from the same `get-sth` response** |
| static-ct-api | `ctverify.go:215` | `checkpoint`, then one hash tile | `scan.LeafHashInTile` compares the leaf hash against `tileHashes[index % 256]`. One slot. No recompute, and no path to the checkpoint root |

So the RFC arm **does** recompute an audit path, against a root the log supplied in the same
exchange. The triage record on #1323 states that no audit path is recomputed anywhere. That is
wrong for the RFC arm, and this ADR is written to the code.

**No log signature is verified on either arm, and the key material is not in the tree.**
`scan.ParseSTH` (`internal/scan/cttail.go:183`) unmarshals one field, `tree_size`. RFC 6962's
`tree_head_signature` is never read. `scan.ParseCheckpoint` (`internal/scan/cttail.go:308`) splits
the C2SP note on newlines, reads line 1 as a decimal tree size, and keeps the body in `Raw`. The
signature lines below the head are never parsed. `internal/scan/log_list.json` carries 26 RFC logs
and 22 tiled logs, and **zero** of the 48 entries carry a `key` field —
[ADR-0190](./0190-the-ct-log-list-is-a-build-time-artefact-pinned-in-the-image-refreshed-only-by-a-release-and-carrying-no-log-public-keys.md)
§5 rules the stripping and gives its own reasons.

**The SPEC promises more than the tiled arm does.**
[`ct-source-replacement.md`](../spec/ct-source-replacement.md) §5.1 says the point-check asks for
*"`get-proof-by-hash` (RFC 6962) or a self-recomputed tile inclusion proof (static-ct-api)"*. Read
alone and in the present tense, that sentence specifies a recompute to the checkpoint root. The code
compares one slot.

## Decision

> **This product is not a CT auditor. Its point-check answers one question — is this certificate
> present in a log that says it holds it — and it never answers whether the log told the truth. It
> verifies no log signature, it holds no tree head between calls, and its tiled arm compares one
> tile slot rather than recomputing a proof. A `logged` verdict states presence and never log
> integrity.**

Four limbs.

### 1. Presence, not integrity, is the scope, and a half-built auditor is worse than neither

An inclusion proof that means anything needs a tree head the checker trusts. Trusting a head means
holding it, checking its signature against a pinned key, checking consistency against the head held
before it, and doing something when two heads for one log disagree. That last obligation is the
whole of the role: a monitor that gossips heads and acts on a split view. A checker that recomputes
a proof against a root the same request supplied has done arithmetic, not verification, because the
log chose both sides of the comparison.

The product takes on none of those obligations, and this ADR rules that it should not. It has no
gossip peer, no durable head store per log, no key set, and no operator surface on which a split
view could mean anything.

**A half-built auditor is worse than an honest presence check**, because it states a guarantee it
cannot keep. An operator who reads *inclusion proof verified* reasonably concludes that a lying log
would have been caught. Under the shipped code it would not.

### 2. No log signature is checked, and this is refused at this scope rather than scheduled

The signature check is not on a roadmap this ADR holds open. It is outside what the product claims
to do, for §1's reason. A later product that takes on the auditor's obligations may add it, and this
ADR forbids nothing about that. What it forbids is the intermediate state where the code checks a
signature and the surfaces imply the rest of the role.

The deleted comment said the check *"stays deferred"*. It does not. Deferred names work that is
scheduled, and nothing schedules this.

**The stored signed head stays, and it is not a scheduled check.** The tail writes it through
`AdvanceCTLogCursor(..., SignedHead: signedHead)` (`cttail.go:228`). It is raw evidence at near-zero
cost, in the same spirit as the transcript corpus.

**ADR-0190 §5's successor clause is bounded here.** It says a change that verifies a CT log
signature re-embeds the keys and is a change to its own Decision. Under this ADR that change is also
a change to the product's scope, so it is not a two-line addition to a parser.

### 3. Each arm proves what its protocol makes cheap, and neither proves more than that

The RFC arm recomputes the audit path, because `get-proof-by-hash` hands it one and the recompute is
local arithmetic. The root it compares against is the one `get-sth` served in the same exchange. The
value of the recompute is bounded and real: a log cannot answer with a proof for a leaf hash it does
not hold without also serving a root that matches the fabricated proof, so a careless or broken
response fails. A deliberate one does not.

The tiled arm compares one slot. static-ct-api serves no proof endpoint. Recomputing to the
checkpoint root there means fetching the sibling hash tiles up the tree, which is a different
request pattern and a different cost, and it would still compare against a root the log served. The
slot compare answers *does this log's own tile hold this leaf hash at this index*, and that is the
presence question §1 fixes as the scope.

**`ct-source-replacement.md` §5.1's "a self-recomputed tile inclusion proof" is withdrawn.** It
describes work the code does not do and should not do at this scope. Under ADR-0058's test, a
session holding §5.1 alone would build the recompute. The replacement wording is in this issue's
manifest.

### 4. The rule does not reach three adjacent questions

Each is real, and naming them stops a later session reading the silence as a ruling.

- **What the check says when it cannot claim presence.** ADR-0193 rules it. `not-logged` has one
  warrant, an error outranks an absence, and `unverifiable` is silent on the auto path.
- **Whether an SCT's own signature is valid.** Nothing in the tree checks it, and ADR-0193 §5
  already records that its own rule neither requires nor forbids it. §1 here supplies the ground:
  an SCT signature is a claim about a log's promise, which is the integrity question.
- **What the tail does with its checkpoint.** `cttail.go:129` runs a shrink check and says in the
  same breath that it is not the consistency proof. That check needs no key, because a tree size
  below the cursor is a contradiction inside one log's own answers.

## Consequences

- **This ADR changes no Go code.**
- **A log that lies about a slot is not detected.** State the exposure plainly. The product can
  return `logged` for a certificate a log claims to hold and does not. It leaves exposed an operator
  who reads `logged` as evidence that the certificate is publicly accountable and would be found by
  a third-party monitor. It does **not** leave exposed the operator acting on the product's actual
  purpose: `NOT logged` is the notable signal
  ([`ct-source-replacement.md`](../spec/ct-source-replacement.md) §5.4), and ADR-0193 §3 guarantees
  no instrument failure produces it. The product's drift readings are unaffected either way, because
  ADR-0027 keeps every CT input out of the `certificate` facet value.
- **One SPEC edit is recorded, not made.** §5.1's tile sentence is withdrawn. The anchor and the
  replacement are in `docs/adr/.pending/1323.md`. §5.4's amendment is **not** recorded here:
  ADR-0193 already carries it, in #1308's manifest, against the same anchor.
- **A false `not-logged` is reachable on the tiled arm, and it is a defect.** `ctverify.go:230` reads
  `if index >= sth.TreeSize { return checkNotFound }`. An SCT is a promise to include within the
  log's maximum merge delay, so a certificate observed minutes after issue legitimately carries an
  index past a current checkpoint. That is *not yet included*, which is a doubt rather than a denial,
  and ADR-0193 §3's warrant does not cover it. `scan.LeafHashInTile` already states the correct
  reading for the sibling case at `internal/scan/ctverify.go:395` — *"a short head tile has not yet
  reached the index: not-yet-inclusion, never a mismatch"* — and returns `checkErrored`. The two
  sites disagree. **The one-line change to `checkErrored` ships as its own ticket**, against
  ADR-0193's rule rather than against this one.
- **The residual false-`not-logged` risk on the RFC arm is a middlebox, not a log.** `checkRFC` reads
  `400` and `404` as the log's denial, which is what RFC 6962 gives those statuses. A proxy or CDN in
  front of the log can serve either.
  [ADR-0106](./0106-the-ct-poll-is-a-scan-that-schedules-and-a-ct-admission-is-a-name-citing-its-batch.md)
  measured exactly that against crt.sh — two spurious `404`s from a URL that served a 95 KB body
  seconds earlier — and ruled every non-200 transient there. The point-check cannot follow that rule
  without discarding the only denial the RFC protocol offers. **Whether a `not-logged` needs a
  second, independent confirmation before it emits a `warn` ships as its own ticket.**
- **§5.4's on-demand re-check is unbuilt.** ADR-0193's Consequences already record it and give it a
  ticket. This ADR adds nothing to that.
- **[`CONTEXT.md`](../../CONTEXT.md) gains nothing.** A verification verdict is an ephemeral event
  and mints no subject.
- **Two code comments carry a wrong or a stale citation.** `ctverify.go:179` cites
  `ct-source-replacement.md` §4.4, which rules the tail's cadence and states nothing about the
  point-check's root. `ctverify.go:250` says the audit-path recompute *"stays deferred"*, which §2
  replaces. Both replacements are in this issue's manifest.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Rule the SPEC correct and record the missing tile recompute as a defect** | It buys a recompute against a root the same log served, which is §1's arithmetic rather than verification. The cost is real — sibling hash tiles up the tree on every point-check, against a log the operator does not own — and the guarantee bought is none. It also leaves the far larger absence, the signature, unruled and reading as an oversight |
| **Verify the log signature and stop there** | Needs 48 public keys the tree does not carry, which ADR-0190 §5 strips for reasons of its own, plus a policy for a key that rotates. It buys the wrong half: a valid signature over a head proves the log signed a head, and the product has nothing to compare that head against and no peer to gossip with. It is the half-built auditor §1 refuses, and it would let a surface say *verified* |
| **Take on the monitor's role — hold heads, check consistency, act on a split view** | A different product. It needs durable per-log head storage, a gossip peer or a second source of heads, a consistency-proof path, and an operator surface for a split view, which is a claim about a third party's honesty the product has no way to act on. Nothing in the v1 estate model reads such a claim |
| **Keep the deleted comment's word, "deferred"** | It names work that is scheduled, and nothing schedules this. A reader who believes the check is coming writes the intermediate state §2 refuses, and cites the comment as the plan |
| **Withdraw §5.1's RFC half as well** | The RFC half is accurate. `get-proof-by-hash` is what the code asks for, and `scan.VerifyInclusion` recomputes the audit path RFC 6962 §2.1.1 specifies. Withdrawing an accurate sentence to make a symmetric edit would remove a true statement from the SPEC |
| **Fold this into [ADR-0193](./0193-a-stapled-ocsp-response-only-narrows-the-sct-set-and-no-usable-sct-is-unverifiable-never-not-logged.md)** | That ADR's subject is the verdict union and the warrant for an accusation. Its §5 explicitly excludes the inclusion proof's internals and an SCT's cryptographic validity. This rule is the scope decision behind those exclusions, and it withdraws a SPEC sentence ADR-0193 does not touch |
