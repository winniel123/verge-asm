# ADR-0138: a release pins every byte it builds, so it delegates no build step, and its identity is the keyless anchor of its own workflow

- **Status:** Accepted
- **Date:** 2026-09-03
- **Ticket:** [#1240 Which ADRs this effort writes, and where the stale ADR-0118 citation lands](https://github.com/winniel123/verge-asm/issues/1240)
- **Map:** [#1064 Release pipeline: a tag becomes a signed, attested, multi-arch release](https://github.com/winniel123/verge-asm/issues/1064)
- **Decides, from three sides:** [#1080](https://github.com/winniel123/verge-asm/issues/1080) (the build delegates nothing), [#1076](https://github.com/winniel123/verge-asm/issues/1076) (Build L2 and the L3 refusal), [#1079](https://github.com/winniel123/verge-asm/issues/1079) (the permission scope and the keyless anchor)
- **Spec:** [`docs/spec/release-pipeline.md`](../spec/release-pipeline.md), which states the mechanism this ADR rules on
- **Rests on:** [ADR-0053](./0053-a-secret-is-held-only-where-its-act-is-performed-and-the-shared-store-holds-none.md) — a secret is held only where its act is performed. §3 is that rule reaching the release.
- **Inherits the runtime of:** [ADR-0001](./0001-stack-and-runtime.md) — one image, two compose services, distroless, and a cross-compiling build

## Context

The map that produced this ADR closed 25 tickets. Three of them look like separate questions and
are not.

[#1080](https://github.com/winniel123/verge-asm/issues/1080) asked whether the release calls
Docker's reusable build workflows or hand-rolls the pipeline.
[#1076](https://github.com/winniel123/verge-asm/issues/1076) asked which SLSA provenance route the
release takes and which build level it claims.
[#1079](https://github.com/winniel123/verge-asm/issues/1079) asked what permissions the workflow
holds and who may push a release tag.

Each was answered on its own. Then each turned out to cite the other two as its reason.

The build pins every byte, **therefore** it refuses the one documented route to SLSA Build Level 3,
because that route moves the build into a reusable workflow. **Therefore** the identity every
signature rests on stays inside this repository's own workflow. And because that identity is the
release's only trust anchor, the release carries no long-lived key, which is why a required tag
signature is refused.

Split into three ADRs, a reader must hold all three to understand any one.
[ADR-0124](./0124-a-backup-carries-data-and-no-secret-and-updating-is-guided-not-self-applied.md) is
the precedent for merging them, because it rules on backup **and** on updates in one document.

**The number.** `0090` is the only gap in the ADR log, and it is not an accident. `ADR-0059:510`
and six sites in `docs/research/sensitive-ports.md` state in terms that ADR-0090 is left unused.
Taking it would falsify six committed sentences and force an
[ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) mark at
each. ADR-0124 set the neighbouring precedent when it refused to resurrect the clobbered ADR-0118.
`0138` costs nothing.

## Decision

> **A release build pins every byte it runs and delegates no build step to a third party. The
> project therefore publishes SLSA v1.0 Build Level 2 and refuses the reusable-workflow route to
> Level 3, because that route moves the Fulcio identity every signature rests on out of this
> repository's own workflow — and that identity is the release's only trust anchor, which is why a
> release carries no long-lived key.**

### 1. The build pins every byte, and delegates no build step

`.github/workflows/release.yml` hand-rolls the pipeline. It calls no Docker reusable workflow, not
even for the build. The shape is `docker/bake-action` over a committed `docker-bake.hcl`, on one
amd64 runner, with `provenance = false` and `sbom = false`.

`docker/github-builder` was read in full at `v1.17.0`, 1415 lines, and it fails on three
independent counts. It signs under Docker's own Fulcio identity. It applies the image tags inside
its own call, so the Trivy gate cannot sit between the digest push and the tag. And on a public
repository it always injects a second BuildKit provenance, which would put two predicates under two
identities on every image.

**The pinning question has a sharper answer than the objection needed.** A reusable workflow **may**
be pinned to a commit SHA, and pinning does not break the verification that workflow performs. So
pinning is not the objection. **The objection is that a SHA pin freezes the caller line only.** It
does not freeze what the workflow fetches at run time. That workflow runs an `npm install` with no
integrity hash in three separate jobs, and it pins three images by tag rather than by digest.

**So the house rule does not survive delegation.** This project cannot enforce pin-every-byte
inside a third party. Adopting `github-builder` would not satisfy the rule. It would **replace** the
rule with a trust-Docker rule for the build half. That swap is a legitimate choice for some
projects. It is a real cost here, and no document may claim the pin rule still holds if the swap is
ever made.

**The rule reaches one layer further down**
([#1155](https://github.com/winniel123/verge-asm/issues/1155)). **A release-signing action must sit
on a supported upstream.** A SHA pin controls *when* the project takes a change. It does not control
*whether* one exists to take. A deprecated wrapper action SHA-pins its own inner action, so pinning
the wrapper pins a stale signer, and a fix landing upstream arrives only if the deprecated wrapper
publishes again.

### 2. The project publishes SLSA v1.0 Build Level 2, and refuses Level 3

The release attests with `actions/attest`, SHA-pinned, with `push-to-registry: true`. The operator
guide states Build Level 2, states the condition GitHub names for Level 3, and states plainly that
this project does not meet it.

GitHub Docs are the contract: "Artifact attestations by itself provides SLSA v1.0 Build Level 2."
The same page gives one route to Level 3, a reusable workflow "that many repositories across your
organization share". A 2026-01-20 changelog claims Build Level 3 without repeating that condition,
and that wording is treated as marketing.

**Level 3 is not merely unattractive. It contradicts §1 and §3.** §1 refuses to delegate a build
step. §3 requires the signing identity to stay in this repository's own workflow. GitHub's one
documented route to Level 3 breaks both.

Beyond that, verge-asm is one repository, and no source settles whether one caller plus one private
reusable workflow earns the isolation GitHub describes. **The project does not publish a level it
cannot source.**

`slsa-framework/slsa-github-generator` was rejected on three measured grounds. Its README states
the project is no longer actively maintained. Its workflows **must** be referenced by a `vX.Y.Z`
tag, so a commit SHA fails the build and §1 cannot hold. And it changes how consumers verify,
requiring a second tool instead of `gh`.

**One consequence is accepted rather than fixed.** OpenSSF Scorecard's `releasesHaveProvenance`
check matches only an `.intoto.jsonl` Release asset. Provenance lives in the registry and in the
GitHub attestations API, so the check caps at 8 of 10. Reaching 10 costs an asset that contradicts
two settled decisions. **The cap is a cited refusal, so a later session does not "fix" it.**

### 3. The release's identity is the keyless anchor of its own workflow, so it holds no key

The release signs keyless. **One Sigstore trust anchor covers the two images, the six image
signatures, the provenance attestations, the SBOM attestations and the checksum list.**

Fulcio writes `job_workflow_ref` as the certificate subject, and **a reusable workflow changes that
string**. So the sign and attest steps run only in a job defined in
`.github/workflows/release.yml`. Without that rule, the identity an operator pins is unknowable, and
it would move again whenever a delegated workflow version moved.

The resulting identity is the one an operator pins:

```
https://github.com/winniel123/verge-asm/.github/workflows/release.yml@refs/tags/vX.Y.Z
```

**A key-based route cannot satisfy the one-anchor rule at all.** The chosen attestation action is
Sigstore keyless only and offers no key mode, so a key-based signature would give one release two
anchors. An operator who verifies the image and then skips the SBOM, because the SBOM needs a
different command and a different root, has verified half the release.

[ADR-0053](./0053-a-secret-is-held-only-where-its-act-is-performed-and-the-shared-store-holds-none.md)
reaches the same answer on its own. **Keyless adds no standing secret.** Cosign documents no
rotation and no revocation for a self-managed key, so this project would have to write that
procedure itself.

**A required tag signature is therefore refused**, and this is the rule following from the anchor
rather than a separate judgment. A `required_signatures` rule on the tag ruleset reintroduces
exactly the durable key material keyless exists to avoid. Two further reasons. A signed tag must be
annotated, and the release procedure settled on a bare tag. And the threat is a compromised
account, which a key on the same machine as the credential does not mitigate. **It becomes worth
revisiting if the repository gains a second person with push access.**

**The workflow adds zero secrets.** `GITHUB_TOKEN` covers the registry push, the SARIF upload, the
attestations and the Release creation.

**The cost is stated, not smoothed.** A keyless signature writes the repository URL, the workflow
path, the commit SHA, the tag, the runner class and a run link to a permanent public log. A
**private** fork that runs this pipeline unchanged discloses its repository name and its commit
SHAs forever. A fork that objects edits the workflow itself. **This project supports no key mode.**

## Consequences

- **`release.yml` owns its own step order, action bumps and OCI annotations.** That is real,
  recurring work, and §1 accepts it. It buys a pipeline whose every input is identified by a SHA or
  a digest, and whose signatures carry this repository's identity.
- **The verification guide states a level and a refusal together.** Build Level 2 published, Level 3
  named, and GitHub's own condition quoted. A guide that stated a level without its condition would
  be making a claim this project cannot source.
- **`contents: write` and `id-token: write` never share a job.** §3's anchor is only as good as the
  blast radius of the token beside it. The cost is one artifact hop, because the blob signature is
  produced in the job holding `id-token: write` and consumed by the job holding `contents: write`.
- **A fork cannot mint a release under this identity, but it can mint one under its own.** A
  repository guard on the first job stops that, because a blob signature would otherwise succeed
  before the registry push failed.
- **This ADR is closed to any subject outside the tag-to-release pipeline.** A runtime trust
  boundary is not a section here. See
  [ADR-0139](./0139-the-probers-origin-is-the-image-that-carries-it-and-a-host-bounds-the-binary-rather-than-verifies-it.md).
- **A later addition is an `## Amendment` block.** It never edits the Decision.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **`docker/github-builder`, in any mode** | Three independent counts in §1. It signs under Docker's identity, it applies the tags inside its own call so the scan gate cannot precede them, and it forces a second BuildKit provenance. Revisit only if Docker ships a mode that writes no tag and forces no provenance |
| **A reusable workflow for the build half only**, with signing kept local | It still fetches unpinned bytes at run time, so §1's rule does not survive the boundary. The build output would be trusted on Docker's word rather than on a pin this project holds |
| **`slsa-framework/slsa-github-generator`** for a stronger paper claim | Unmaintained by its own README, and it forbids a commit-SHA pin outright, which suspends §1. It also changes the command every consumer runs |
| **Chasing SLSA Build Level 3** | Its one documented route breaks §1 and §3 at once. And no source settles whether one repository plus one private reusable workflow earns the isolation GitHub describes. The project does not publish a level it cannot source |
| **A key-based cosign signature**, or a key mode offered to forks | Gives one release two trust anchors, which breaks the SBOM ruling that binds them to one. It also means writing the key custody and rotation procedure cosign does not document, and holding standing key material ADR-0053 keeps out |
| **`required_signatures` on the tag ruleset** | Reintroduces the durable key §3 exists to avoid. A signed tag must also be annotated, and the release procedure settled on a bare tag. The threat is a compromised account, which a key on the same machine does not mitigate |
| **One `.intoto.jsonl` Release asset**, to score Scorecard 10 of 10 | It contradicts the index-digest provenance subject and the refusal to ship loose attestation assets. The cap at 8 is accepted and cited |
| **Three separate ADRs**, one per ticket | Each cites the other two as its reason, so a reader must hold all three to understand any one. ADR-0124 is the precedent for one document over two subjects |
| **Taking the `0090` gap** | `ADR-0059:510` and six sites in `docs/research/sensitive-ports.md` state that ADR-0090 is left unused. Taking it falsifies six committed sentences and forces an ADR-0058 mark at each |
