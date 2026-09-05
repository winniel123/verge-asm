# ADR-0142: a corpus input is generated deterministically, and the random draw is production's alone

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1283 ADR gaps: internal/auth, internal/measure/blanketdiscrim, internal/proposer, internal/seed](https://github.com/winniel123/verge-asm/issues/1283) — gap 1, the deleted `FixedPorts` comment
- **Why not an amendment to [`golden-corpus.md`](../spec/golden-corpus.md):** that file binds three subjects — `resolution-walk` (§2), `wildcard-discrimination` (§8) and the `Custody` derivation (§10) — and §1 states it "discharges no other leaf". The rule below already binds `blanketdiscrim`'s corpus, which that file names nowhere. The amendment would have to grow past its own subject to carry it
- **Rests on:** [ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md) (the corpus is authored, hermetic, and is the version gate's evidence) and [ADR-0008](./0008-derivation-versions-move-on-content.md) (a version leaf moves on content, so a moved row must mean something moved)
- **Adjacent:** [ADR-0104](./0104-an-undiscriminated-reach-is-a-gap-and-a-blanket-responder-is-measured-not-listed.md) (the `blanket-discrimination` leaf and its control-port band) and [ADR-0069](./0069-a-control-label-is-one-label-and-the-set-must-falsify-label-independence.md) (the control-label set)

## Context

Two leaves generate their own input. `CryptoPorts` draws `ControlPortCount` = 8 ports from
`crypto/rand` inside the 49152–65535 band (`internal/measure/blanketdiscrim/ports.go:20-35`).
`CryptoLabels` draws 9 random labels and 1 structured label over an RFC 5737 address
(`internal/measure/wildcarddiscrim/labels.go:32-50`). **These are the only two `crypto/rand` sites
under `internal/measure`.** Both draws are the defence working: a control set an origin can predict
is a control set an origin can allowlist.

A corpus row renders by running the real leaf against an authored peer and hashing the NDJSON it
writes. So the render walks straight through the generator. Both corpora already handle it, and
each handled it separately: `blanketdiscrim`'s passes `FixedPorts{P: []uint16{50001, 50002, 50003}}`
(`corpus/rows.go:8`), and `wildcarddiscrim`'s passes `DeterministicLabels{}`
(`corpus/script.go:90`). The second was written without a rule to read, because there was none.

**ADR-0021's hermeticity argument reaches the peer and stops at it.** Its ruling is that the fixture
is in-process, "no network, no container, no image", and that a captured transcript "bought nothing
and cost determinism". Both sentences are about what the leaf *talks to*. A generator inside the
binary under test is neither a peer nor a fixture, so the paragraph does not reach it. A session
reading ADR-0021 for the whole determinism condition finds one half of it.

**The ordering half is already ruled and is not this one.** `ports.go:17` states that a `PortGen`
returns sorted ports "so the control probe is order-stable for the golden corpus (ADR-0021)".
Sorting gives a stable order over a set that is still unstable. Order-stability and determinism are
two properties, and only the first had a ruling.

**What a violation costs.** `golden-corpus.md` §3.2 runs A1 (self-identity, twice in one process),
A2 (each leg against one shared expected artefact) and A3 (cross-architecture identity). A single
`crypto/rand` draw inside a render fails A1 first, on every leg, on every PR — and A1's stated
purpose is to keep a red `arm64` leg distinguishable from a flake. A nondeterministic corpus is the
one defect that makes every assertion beneath it unreadable.

## Decision

> **Every input a golden corpus feeds the code under test is generated deterministically. A corpus
> generator never draws from `crypto/rand` or `math/rand`, and never reads a clock, the environment
> or the filesystem outside its own `testdata`. Where production draws, the generator is an injected
> seam and the corpus supplies a fixed implementation of it. Production's implementation does not
> change.**

### 1. What the rule binds

Every corpus package in the repository — the eight that carry a `corpus.lock.json`:

| Package | Leaf or derivation |
| --- | --- |
| `internal/measure/resolutionwalk/corpus` | `resolution-walk` |
| `internal/measure/wildcarddiscrim/corpus` | `wildcard-discrimination` |
| `internal/measure/connectoutcome/corpus` | `connect-outcome` |
| `internal/measure/connectoutcome/certcorpus` | the certificate block off the same leaf |
| `internal/measure/httpexchange/corpus` | `http-exchange` |
| `internal/measure/tlsacceptance/corpus` | `tls-handshake` |
| `internal/measure/blanketdiscrim/corpus` | `blanket-discrimination` |
| `internal/custody/corpus` | the `Custody` derivation |

Seven of those directories are named `corpus`; `certcorpus` is the eighth and is bound the same way.
**`internal/measure/edgefanout` has no corpus package today.** The rule binds it on the day it gains
one, and that day needs no amendment here.

### 2. The seam is the mechanism, and both existing seams stay as written

`PortGen` with `FixedPorts` and `LabelGen` with `DeterministicLabels` are the shape. Neither changes.
Production keeps `CryptoPorts` and `CryptoLabels` and keeps drawing.

**The fixed value need not equal the shipped one.** `blanketdiscrim`'s corpus renders with three
ports where `ControlPortCount` is eight, and that is correct rather than sloppy: the shipped
parameter is pinned by `ParamsDigest`, which hashes `DefaultParams()` — the count and both band
bounds — on its own line in `corpus.lock.json`. The row's bytes pin the leaf's behaviour; the digest
pins the leaf's declared parameters. A corpus that had to render with the shipped count would be
pinning the same fact twice, and A6 already reads the digest.

### 3. What the rule is not

- **Not a claim that the drawn value is unimportant.** The band and the count are ADR-0104's
  defence, and a predictable control set is exactly the failure that defence exists against.
- **Not an ordering rule.** Sorting is ADR-0021's, stated at `ports.go:17`, and both properties are
  owed.
- **Not a licence to make production deterministic.** The *Alternatives rejected* table refuses that.

### 4. The A1 gate is the enforcement, and it is not a complete detector

Each corpus package's `TestCorpusSelfIdentity` renders twice in one process and compares bytes. It
catches a draw **that reaches the output**. It does not catch a draw whose value never reaches the
rendered NDJSON — which is the case for both generators today, and is why the fixed sets are cheap.

So the rule is written rather than left to the test. A generator whose value stays out of the bytes
today can reach them tomorrow, and the session that makes it reach them will not be the session that
chose the draw.

## Consequences

- **A new corpus package with a generated input owes a fixed implementation of that seam**, and a
  new generator in production owes a seam. Neither obligation was stated anywhere before this ADR.
- **This ADR changes no Go code.** `FixedPorts`, `CryptoPorts`, `DeterministicLabels` and
  `CryptoLabels` are all unchanged, and so is every `corpus.lock.json`.
- **`golden-corpus.md` §3.2's A1 gains a qualification.** Its cell names Go's map-iteration
  randomisation as the instability A1 guards against. That is one source and, read alone, it reads
  as the whole one.
- **[ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md) gains a qualification at its
  hermeticity clause**, for the same reason: the in-process peer is a condition on determinism and
  not the whole of it.
- **The rule reaches leaves outside the membership vector.** `connect-outcome`, `http-exchange`,
  `tls-handshake` and the `Custody` derivation are all bound, and `golden-corpus.md` §1 discharges
  none of them. That reach is the ground for an ADR rather than a SPEC amendment.
- **`CONTEXT.md` gains nothing.** A corpus generator is a test seam, not a domain term.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Amend [`golden-corpus.md`](../spec/golden-corpus.md)** instead of writing an ADR | That file binds `resolution-walk`, `wildcard-discrimination` and `Custody`, and §1 says so in as many words. The rule already binds `blanketdiscrim`'s corpus, which the file never names, so the amendment would have to grow past its own subject on its first use. A file that specifies three blocks is the wrong home for a rule over eight packages |
| **A random or time-seeded corpus input** — draw fresh each render, or seed a PRNG from the clock | Fails A1 inside one process, on every leg, on every PR, and the failure reads as a flake rather than as a cause. It also destroys the row's reviewable claim, which is ADR-0021's whole argument for authoring: a reviewer cannot read *which* ports a row used without running the generator, so the diff CI renders on a failure stops being a sentence a human can judge |
| **A fixed PRNG seed**, drawing from `math/rand` with a constant | Deterministic within a toolchain and not across one. `math/rand/v2` makes no stream-stability promise, so the corpus would pin a Go release rather than a leaf, and a toolchain bump would move rows that nothing in this project touched — ADR-0004's out-of-band reference data arriving in the harness by another door. It also buys nothing the fixed set does not |
| **Draw once and record the drawn value into the golden file** | The file then pins a number nobody chose and no claim explains, and every `-update` run re-draws it, rewriting rows nothing changed. That is the capture medium ADR-0021 refused, one value wide |
| **Make production deterministic too**, so no seam is needed | Turns the control set into a public list an origin can allowlist, which is the detector ADR-0104 built the random band to avoid. The seam exists because the two contexts want opposite properties |
| **A section on [ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md)** | Its subject is the version-leaf vector and the authored corpus medium, for the leaves ADR-0041 named. This rule binds the `Custody` derivation and `blanketdiscrim`, neither of which ADR-0021 rules. It is qualified at the clause instead, per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) |
| **A comment on `FixedPorts`** — where the rule was | It bound one symbol in one package. The rule binds eight packages and two seams, and the evidence that a comment could not carry it is that `DeterministicLabels` was written without reading it |
