# ADR-0139: the prober's origin is the image that carries it, and a host bounds the binary rather than verifies it

- **Status:** Accepted
- **Date:** 2026-09-03
- **Ticket:** [#1239 How a prober binary proves its origin when the worker pushes it to a vantage host](https://github.com/winniel123/verge-asm/issues/1239)
- **Map:** [#1064 Release pipeline: a tag becomes a signed, attested, multi-arch release](https://github.com/winniel123/verge-asm/issues/1064)
- **Spec:** [`docs/spec/release-pipeline.md`](../spec/release-pipeline.md) §16
- **Why not a section on [ADR-0138](./0138-a-release-pins-every-byte-it-builds-so-it-delegates-no-build-step-and-anchors-identity-in-its-own-workflow.md):** that ADR is closed to any subject outside the tag-to-release pipeline. The prober push is a **run-time** boundary.
- **Bounded by, and not an amendment to:** [ADR-0103](./0103-a-vantage-is-one-position-and-the-prober-is-optional-provisioning-detail.md), whose Decision is a `vantage` table merge. A supply-chain amendment there would be off-subject.
- **Rests on:** [ADR-0001](./0001-stack-and-runtime.md) (one image, three binaries, one Go module) and [ADR-0053](./0053-a-secret-is-held-only-where-its-act-is-performed-and-the-shared-store-holds-none.md) (the instance generates the prober keypair and only the public half leaves)

## Context

The worker pushes a `prober-linux-<arch>` binary to a remote host at run time and runs it there. It
is the one binary this project produces that executes on a machine the project does not own. The
release pipeline signs, attests and describes everything else it ships. **Nothing signs this
binary**, and the map had never said so out loud.

The question that produced this ADR framed the gap as a second trust boundary in need of a
signature. Three of its premises were loose, and the third points the wrong way.

**ADR-0103 does not make the worker push a binary.** Its Decision is a `vantage` table merge:
measurement identity is mandatory on the row, and the prober connection is optional detail on the
same row. The push rule is
[ADR-0001](./0001-stack-and-runtime.md) plus `packaging-and-configuration.md` §1.5.

**The prober is not outside the release artifact set.** `Dockerfile:40-45` builds
`prober-linux-amd64` and `prober-linux-arm64` in the **same builder stage** as `web` and `worker`,
and `Dockerfile:68` copies them into the worker image at `/app/probers`.

**The code's trust boundary points the other way.** `internal/remoteexec/probe.go:126` names the
prober untrusted and caps its stdout, and `internal/remoteexec/conn.go` pins the host key on first
use. Verge distrusts the host, and that half is built.

The sharp fact the question missed: **the binary the worker pushes has never left an image this
project already signs.**

## Decision

> **The prober binary's origin is the worker image that carries it. Nothing signs it a second
> time, at either end, and the vantage host bounds what the binary may do rather than proving what
> it is. Where the vantage-host operator is not the instance operator, verge offers that person no
> origin proof and says so.**

### 1. The origin is the image, and the release pipeline needs no prober step

`cmd/worker/main.go:59` reads `VERGE_PROBER_DIR`, default `/app/probers`.
`DirBinaryProvider.Binary` opens `prober-<goos>-<goarch>` from that directory, and serves the
own-arch fallback only where the requested platform is the instance's own
(`internal/remoteexec/binary.go:36-52`).

**Every read path hits the worker image's read-only filesystem. No path fetches a release asset,
and no path reaches the network.**

So an operator who verifies the worker image has verified the prober. The image signature, the
provenance attestation and the SBOM all cover every byte in that image. **`release.yml` and
`release-scan.yml` are both unchanged by this ADR.**

**No loose prober binary ships as a Release asset.** A second signed copy would be a second subject
over bytes a signed image already covers, and **no code path consumes it**. That also preserves the
property `deploy/prober/README.md` names as load-bearing: the instance ships the exact binary it
wants at each invocation, so version skew between instance and vantage is structurally impossible.

### 2. Nothing verifies the binary, at either end

A refusal with reasons. Three mechanisms were weighed and each fails on its own ground.

| Mechanism | Why it lost |
| --- | --- |
| The worker verifies before it pushes | It would verify a file on its own image layer, using a trust root shipped in the same image. **A compromised image passes its own check.** It also needs a network path to Sigstore, which the air-gap kit exists to avoid |
| The host verifies after the binary lands | The host needs cosign, a trust root and a network path. `deploy/prober/README.md` keeps that host to `alpine` plus `openssh-server` deliberately, and states the host "is the operator's rather than ours" |
| A detached signature rides the push | It re-signs bytes the release anchor already covers, and lands them on a host with nothing to check them against |

**What the operator relies on instead**, five controls that all exist today:

1. Image verification at pull.
2. SSH public-key auth, with `restrict` and `from=<egress>` in `authorized_keys`. The instance
   generates the keypair and only the public half leaves it (ADR-0053).
3. The trust-on-first-use host-key pin. A change is a hard failure, never a prompt.
4. The `0700` random temp path, `/tmp/verge-prober-<8 random bytes>` (`probe.go:104,112`).
5. The delete after every run (`probe.go:118`).

### 3. The boundary this rules on is host to verge, and the other half is already built

Verge to host is settled code, not an open decision: public-key auth, the host-key pin, the bounded
prober stdout, and the `uname` arch check that gates the push at `probe.go:96-101` and refuses a
mismatched binary rather than shipping one.

### 4. A lent host gets no origin proof, and the condition is stated

In every install today the vantage-host operator **is** the instance operator. There is one trust
decision and it is made at image pull. **The condition that would change that is named rather than
left implicit: the vantage-host operator is not the instance operator**, on a lent or shared host.

When it holds, **verge offers that person no origin proof.** Their lever is the SSH account, not a
signature: a non-root user, `restrict`, `from=`, `cap_drop: [ALL]` and `no-new-privileges`. **Those
bound what the binary may do. They do not prove what it is.**

**One consequence is sharp and is stated rather than smoothed.** `probe.go:118` removes the binary
after every run, so a lent-host operator **cannot inspect afterwards what verge ran**.

### 5. The transitive claim carries its bound wherever it is printed

Verifying the worker image covers the prober **for whoever holds that image**. That is the instance
operator. A lent-host operator holds only a binary that arrives over SSH and is deleted.

A guide making a verification claim is a contract, so **the claim and its bound are stated
together, in one place**, on the page the affected reader opens.

## Consequences

- **`docs/guides/prober.md` gains one subsection** for the host operator: the five §2 controls, the
  §4 condition, and the bounded §5 claim. It may land after the first release, because that page is
  **silent** on binary trust today rather than wrong, and a silent page ships nothing false.
- **`docs/guides/verifying-releases.md` gains nothing.** A prober paragraph there would invite the
  reader to hunt for a prober signature that does not exist.
- **`deploy/prober/README.md`'s posture table gains no row.** It lists container controls, and the
  SSH account controls already sit above it.
- **`CONTEXT.md` gains nothing.** The instance-operator and vantage-host-operator split is a
  deployment role, not a domain term, and the glossary answers with one `operator` throughout.
- **This ADR changes no production Go code.** Every control it names already exists.
- **The refusal is reversible at a stated price.** Signing the pushed binary becomes worth
  revisiting if a lent-host deployment becomes real, and §2's table is the list of what such a
  change would have to solve.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Ship a loose, signed `prober-linux-<arch>` as a Release asset** | A second copy under a second subject, over bytes a signed image already covers, that **no code path consumes**. The worker reads its own image and never fetches a release asset. It also reintroduces the version skew the fresh-push design removes |
| **The worker verifies the binary before it pushes** | Self-referential. The trust root ships in the image under audit, so a compromised image passes its own check. It also needs a Sigstore network path from the worker |
| **The host verifies the binary after it lands** | The vantage host is deliberately `alpine` plus `openssh-server` and is the operator's rather than ours. A verify step needs cosign, a trust root and a network path on a machine this project does not own |
| **A detached signature pushed beside the binary** | Re-signs bytes the release anchor already covers, and lands them where nothing can check them |
| **Publish a per-release prober digest** for a lent-host operator to compare | Reopens the loose-artifact refusal through a side door, and publishes a digest nothing verifies |
| **Stop deleting the pushed binary**, so a host operator can inspect it | Trades a real hygiene control for an inspection window that only helps a party who has already run the binary |
| **A section on [ADR-0138](./0138-a-release-pins-every-byte-it-builds-so-it-delegates-no-build-step-and-anchors-identity-in-its-own-workflow.md)** | That ADR is closed to subjects outside the tag-to-release pipeline. This is a run-time boundary |
| **An amendment on [ADR-0103](./0103-a-vantage-is-one-position-and-the-prober-is-optional-provisioning-detail.md)** | Its Decision is a schema ruling about the `vantage` row. A supply-chain amendment there is off-subject, which is the same fault that closed ADR-0138 to this subject |
| **A bullet on the ADR-0124 amendment** | Under [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)'s split, an amendment carries a claim about the world and a withdrawal carries a mechanism. This ruling refuses **three mechanisms** |
