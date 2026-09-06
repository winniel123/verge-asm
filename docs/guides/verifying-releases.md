---
title: Verifying a release
section: Operating
order: 4
description: Verify a published verge-asm release supply chain with gh attestation verify, cosign and grype — keyless Sigstore, Fulcio and Rekor signatures, SLSA v1.0 Build Level 2 (L2) provenance, SPDX SBOM attestations, CycloneDX SBOM assets, the signed SHA256SUMS checksum list, GHCR container image index and per-platform digests, the floating latest tag, and the 13 named Release assets.
---

# Verifying a release

Every tagged verge-asm release publishes exactly **two** container images to the GitHub
Container Registry:

- `ghcr.io/winniel123/verge-asm/web`
- `ghcr.io/winniel123/verge-asm/worker`

Each image is a **multi-arch manifest list** covering `linux/amd64` and `linux/arm64`. Each image
carries the release tag `vX.Y.Z` and the floating `latest` tag. Both tags point at the same index
digest.

This guide is the downstream *consumer's* checklist. Run it before you pull an image into a real
deployment. To build and test from source yourself, see [verifying.md](verifying.md).

Throughout, replace `vX.Y.Z` with the release tag you verify, for example `v0.1.0`. Repeat each
command for the `web` image and for the `worker` image.

---

## What one release publishes

| Artifact | Subject | Count |
| --- | --- | --- |
| index digest | one per image | 2 |
| platform manifest digest | one per image, per platform | 4 |
| cosign signature | the 2 index digests and the 4 platform digests | 6 |
| SLSA provenance attestation | the index digest | 2 |
| SPDX SBOM attestation | the platform manifest digest | 4 |
| cosign blob signature | `SHA256SUMS` | 1 |
| named Release assets | the Release page | 13 |

The 13 named assets are `docker-compose.yml`, `env.example`, `docker-compose.external-db.yml`,
`SHA256SUMS`, `SHA256SUMS.sigstore.json` and the eight SBOM documents. The Release page also
carries the Trivy scan asset set. GitHub attaches two source archives to every Release, and verge
signs neither of them.

**Provenance and the SBOM bind to different digests.** One build produces one index, so provenance
binds to the index digest. One SBOM describes one platform, so each SBOM attestation binds to a
platform manifest digest. **No single digest serves both.**

---

## The trust anchor

The release signs keyless. There is no verge-asm public key to fetch or trust on the side.

Trust rests on the **workflow identity** the Sigstore certificate records. One Fulcio identity
covers the six image signatures, the two provenance attestations, the four SBOM attestations and
the `SHA256SUMS` signature. The identity is:

```
https://github.com/winniel123/verge-asm/.github/workflows/release.yml@refs/tags/vX.Y.Z
```

The issuer is `https://token.actions.githubusercontent.com`.

The release runs two signing tools, `cosign` and `actions/attest`. Both use one OIDC identity, one
Fulcio CA and one Rekor transparency log. **Two tools are not two anchors.**

---

## The SLSA level

The project publishes **SLSA v1.0 Build Level 2**.

GitHub Docs are the contract, and they set the level this way:

- Artifact attestations by themselves provide SLSA v1.0 Build Level 2.
- The same page gives one route to Build Level 3, a reusable workflow that many repositories across
  an organization share.
- **verge-asm does not meet that condition, and it claims no Level 3.**

The release workflow keeps its sign and attest steps in `release.yml`, and the build delegates no
step to a reusable workflow.

---

## Prerequisites

- The [GitHub CLI](https://cli.github.com/) (`gh`). It needs no keys, and it reads the GitHub
  attestations API and GHCR directly.
- [cosign](https://docs.sigstore.dev/) v3.1.3 or later, for the six image signatures and the
  `SHA256SUMS` signature.
- [grype](https://github.com/anchore/grype) or Trivy, for the SBOM re-scan.
- Docker with `buildx`, to list the per-platform digests.

---

## 1. Verify the image provenance with `gh`

`--repo` alone proves only that some workflow in this repository signed the image. The required
form pins the signer workflow, the tag ref and the runner class:

```sh
gh attestation verify \
  oci://ghcr.io/winniel123/verge-asm/web:vX.Y.Z \
  --repo winniel123/verge-asm \
  --signer-workflow winniel123/verge-asm/.github/workflows/release.yml \
  --source-ref refs/tags/vX.Y.Z \
  --deny-self-hosted-runners
```

`--signer-workflow` alone fails with `at least one of the flags in the group [owner repo] is
required`. So `--repo` stays.

This command resolves a tag to the **index digest**, and never to a platform digest. The provenance
attestation binds to that same index digest, so the resolution is right for this check.

A non-zero exit means the image is unsigned, altered, or not a product of this repository.
**Do not run it.**

> **Pin to the digest for durable references.** A tag can be re-pointed. A digest cannot.
> For a deployment manifest or an air-gapped record, resolve the tag to its index digest once
> (`docker buildx imagetools inspect ghcr.io/winniel123/verge-asm/web:vX.Y.Z`) and verify the
> `…/web@sha256:<digest>` form thereafter. **Never trust `latest` as an identity.** `latest` is a
> floating pointer, and it may move **backwards**. So the digest `latest` resolved yesterday may
> differ from the digest `latest` resolves to today.

---

## 2. Verify the image signatures with cosign

`cosign verify` checks a **signature**. A signature carries no predicate, so this step is not an
SBOM check and not a provenance check. A release carries six signatures, because `cosign sign`
runs with `--recursive`. It signs the two index digests and the four platform manifest digests.

The exact identity is the primary form:

```sh
cosign verify \
  --certificate-identity https://github.com/winniel123/verge-asm/.github/workflows/release.yml@refs/tags/v0.1.0 \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  ghcr.io/winniel123/verge-asm/web:v0.1.0
```

A script that checks many tags uses the regexp form instead. This form accepts any tag from this
workflow:

```sh
  --certificate-identity-regexp '^https://github\.com/winniel123/verge-asm/\.github/workflows/release\.yml@refs/tags/v' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
```

**Every cosign v2 instruction is wrong for the offline path.** cosign v3 deleted `--offline`. The
cosign air-gap README section declares itself out of date, and it still prints `--offline=true`.

---

## 3. Verify the Release assets

`SHA256SUMS` covers the three loose files and the eight SBOM documents. One blob signature covers
`SHA256SUMS`:

```sh
cosign verify-blob \
  --bundle SHA256SUMS.sigstore.json \
  --certificate-identity <the identity above> \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  SHA256SUMS
```

Then run `sha256sum -c SHA256SUMS` in the directory you downloaded into. The signature proves the
list. The checksum run proves the files.

This pair matters most for `docker-compose.yml`. The workflow substitutes this release's image
digests into that file. An attacker cannot forge a digest. An attacker **can** point the file at an
older verge image that is still validly signed. The signature over `SHA256SUMS` binds the file to
this tag, and image verification alone does not catch that rollback.

---

## 4. Verify a per-platform SBOM

Each SBOM attestation binds to a platform manifest digest. A child-bound attestation is **not
discoverable from the index reference** on either route, so you name the child digest yourself.
Authenticate to the registry, list the platform digests, then verify one digest:

```sh
docker login ghcr.io

docker buildx imagetools inspect ghcr.io/winniel123/verge-asm/web:vX.Y.Z \
  --format '{{ range .Manifest.Manifests }}{{ .Platform.OS }}/{{ .Platform.Architecture }}{{ if .Platform.Variant }}/{{ .Platform.Variant }}{{ end }} {{ .Digest }}
{{ end }}'

gh attestation verify \
  oci://ghcr.io/winniel123/verge-asm/web@sha256:<platform-digest> \
  --repo winniel123/verge-asm \
  --predicate-type https://spdx.dev/Document/v2.3 \
  --signer-workflow winniel123/verge-asm/.github/workflows/release.yml \
  --source-ref refs/tags/vX.Y.Z \
  --deny-self-hosted-runners
```

The third command runs four times per release, once for each platform digest of each image.

**`--predicate-type` is mandatory.** The command defaults to `https://slsa.dev/provenance/v1`. The
SPDX URI is version-derived. `actions/attest` builds it as `https://spdx.dev/Document/v${version}`,
so SPDX 2.3 gives `https://spdx.dev/Document/v2.3`. A plain `https://spdx.dev/Document` matches
nothing.

Three caveats apply to `imagetools inspect`:

- The format string must include the variant. Without the variant, `linux/arm/v7` prints as
  `linux/arm`.
- Skip the `unknown/unknown` row. That row is a BuildKit attestation manifest, not a platform.
- The command exits 1 on a non-index reference. A verge-asm release image is never one.

**The attestations carry SPDX only.** The Release page carries both SPDX and CycloneDX documents,
and no CycloneDX attestation exists.

---

## 5. Re-scan an SBOM for new CVEs

A shipped SBOM lets you re-scan an *already-published* image against today's vulnerability data,
with no pull and no rebuild. Take the plain SPDX document from the Release assets:

```sh
gh release download vX.Y.Z --repo winniel123/verge-asm --pattern 'web-linux-amd64.spdx.json'

grype sbom:./web-linux-amd64.spdx.json
```

This route needs no attestation machinery. The SBOM catalogues the OS layers and the language
layers, so the scan surfaces base-image CVEs that a source-only scan misses.

Download the document for **your** platform. Four SPDX documents ship per release, one per image
per platform. An arm64 operator who reads an amd64 SBOM holds a wrong SBOM, not a partial one.

### Which SBOM route serves which purpose

| Route | Purpose |
| --- | --- |
| Release asset plus `grype` (step 5) | the re-scan route. It needs no attestation machinery. |
| `gh attestation verify` against a child digest (step 4) | the trust route. It is the only route that proves the SBOM describes that digest. |
| `cosign verify` (step 2) | signatures only. cosign is **not** an SBOM route. |

---

## The cheapest check needs no tooling

> Every published release carries a bare `vX.Y.Z` tag and stamps a bare `X.Y.Z` version. A version
> string carrying a suffix did not come from this pipeline.

This check needs no tooling, no network and no trust root. It is the cheapest discriminator in the
guide. **Read it as a convention, and never as a cryptographic guarantee.** It is only as strong as
the repository's tag ruleset.

One plain consequence follows. A source build is never told that a release shipped. A hand-built
binary carries no version the update check can parse, so the instance keeps reporting its release
state as current.

---

## What a passing verification tells you

- The index digest you are about to run came from **this repository's release workflow**, at a tag,
  on GitHub-hosted runners. That claim is SLSA v1.0 Build Level 2 provenance.
- Six cosign signatures cover the two index digests and the four platform manifest digests.
- One signed SPDX SBOM describes each platform manifest digest, so you can audit it and re-scan it.
- `SHA256SUMS` and its blob signature bind the other 11 named Release assets to this tag.

It tells you two further things by omission:

- It asserts no absence of vulnerabilities. Step 5 answers that question. Verification proves
  *provenance and integrity*. A scan reports *current exposure*.
- A clean `docker compose up` proves nothing about a release. With the tag absent locally, `up`
  attempts the pull, warns, builds from source, and exits 0.
