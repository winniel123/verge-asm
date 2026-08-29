---
title: Verifying a release
section: Operating
order: 4
description: Verify a published verge-asm image's keyless signature, SLSA build provenance, and SBOM before you run it.
---

# Verifying a release

Every tagged verge-asm release publishes exactly **two** container images to the GitHub
Container Registry:

- `ghcr.io/winniel123/verge-asm/web`
- `ghcr.io/winniel123/verge-asm/worker`

Each is a **multi-arch manifest list** (`linux/amd64` + `linux/arm64`) carrying two signed,
keyless [Sigstore](https://www.sigstore.dev/) attestations bound to the image's index digest:

- a **SLSA build-provenance** attestation — the signature-plus-build claim: *this digest was
  built by this repository's release workflow, at a tag, on GitHub's hosted runners*.
- a **CycloneDX SBOM** attestation — the software bill of materials, cataloguing the image's
  OS and Go-module layers, which you can extract and re-scan for new CVEs without a rebuild.

There are **no standalone binaries and no Release-asset downloads** to verify — the images are
the whole artifact set. This guide is the downstream *consumer's* checklist: run it before you
pull an image into a real deployment. (To build and test from source yourself, see
[verifying.md](verifying.md).)

> **Pending the first tagged release.** The release pipeline that produces these attestations
> is not yet live and no `vX.Y.Z` tag has shipped. So the commands below cannot resolve an
> image today. They are the verification *contract* — the exact commands a release will be
> verifiable with — and become runnable the moment the first tag is published.

Throughout, replace `vX.Y.Z` with the release tag you are verifying (e.g. `v0.1.0`). Repeat
each command for both the `web` and `worker` images. The tag resolves to the multi-arch
**index** digest — the same digest the attestations are bound to.

---

## Prerequisites

- The [GitHub CLI](https://cli.github.com/) (`gh`) — the simplest verifier. It needs no
  keys and talks to the GitHub attestations API and GHCR directly.
- For the SBOM re-scan: [grype](https://github.com/anchore/grype) (or Trivy).
- Optionally, [cosign](https://docs.sigstore.dev/) — an alternative verifier for consumers
  who do not use `gh` (see [below](#verifying-without-gh-cosign)).

The attestations are keyless: there is no verge-asm public key to fetch or trust on the side.
Trust is anchored in the **workflow identity** recorded in the Sigstore certificate. That
identity is what the `--repo` (and, more strictly, `--signer-workflow`) checks below assert.

---

## Verify signature and build provenance

This is the core check — it confirms the image was built and signed by this repository's
release workflow, not by an impersonator.

```sh
# Verify against the source repository (minimum check):
gh attestation verify \
  oci://ghcr.io/winniel123/verge-asm/web:vX.Y.Z \
  --repo winniel123/verge-asm

# repeat for the worker image:
gh attestation verify \
  oci://ghcr.io/winniel123/verge-asm/worker:vX.Y.Z \
  --repo winniel123/verge-asm
```

For **stronger assurance**, pin the exact signer workflow. This asserts not just that the
image came from this repo, but that it was built by the release workflow specifically:

```sh
gh attestation verify \
  oci://ghcr.io/winniel123/verge-asm/web:vX.Y.Z \
  --repo winniel123/verge-asm \
  --signer-workflow winniel123/verge-asm/.github/workflows/release.yml
```

A successful verify prints the matched attestation and the signer identity. The certificate's
subject is
`https://github.com/winniel123/verge-asm/.github/workflows/release.yml@refs/tags/vX.Y.Z`,
issued by `https://token.actions.githubusercontent.com`. A non-zero exit means the image is
unsigned, tampered with, or was not produced by this repository — **do not run it**.

> **Pin to the digest for durable references.** A tag can be re-pointed. A digest cannot.
> For a deployment manifest or an air-gapped record, resolve the tag to its index digest once
> (`docker buildx imagetools inspect ghcr.io/winniel123/verge-asm/web:vX.Y.Z`) and verify the
> `…/web@sha256:<digest>` form thereafter.

---

## Verify the SBOM is present and signed

The SBOM rides as its own signed attestation with a CycloneDX predicate. Confirm it is present
and valid:

```sh
gh attestation verify \
  oci://ghcr.io/winniel123/verge-asm/web:vX.Y.Z \
  --repo winniel123/verge-asm \
  --predicate-type https://cyclonedx.org/bom
```

---

## Pull the SBOM out and re-scan for new CVEs

The operational payoff of a shipped SBOM: you can re-scan an *already-published* image against
today's vulnerability data without pulling or rebuilding it. Download the attested CycloneDX
document and feed it to grype:

```sh
gh attestation download \
  oci://ghcr.io/winniel123/verge-asm/web:vX.Y.Z \
  --repo winniel123/verge-asm

grype sbom:./<downloaded-cyclonedx>.json
```

`gh attestation download` writes the attestation bundle to the working directory. Point grype
(or `trivy sbom …`) at the CycloneDX JSON it contains. Because the SBOM catalogues the OS and
language layers — not just the Go module graph — this surfaces base-image CVEs that a
source-only scan would miss.

---

## Verifying without `gh` (cosign)

The attestations are standard Sigstore artifacts, so [cosign](https://docs.sigstore.dev/) can
verify them too — useful for consumers who do not have the GitHub CLI. Verify against the
image **digest** (resolve the tag first, as above):

```sh
cosign verify-attestation \
  --type slsaprovenance \
  ghcr.io/winniel123/verge-asm/web@sha256:<digest> \
  --certificate-identity-regexp '^https://github.com/winniel123/verge-asm/\.github/workflows/release\.yml@.*' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
```

The `--certificate-identity-regexp` and `--certificate-oidc-issuer` flags are the cosign
equivalent of `gh`'s `--signer-workflow` pin: they assert the attestation was signed by this
repository's release workflow, via GitHub's OIDC issuer.

---

## What a passing verification tells you

- The image digest you are about to run was **built and signed by this repository's release
  workflow**, at a tag, on GitHub-hosted runners (SLSA build provenance). No one else rebuilt
  or substituted it.
- A **CycloneDX SBOM** for that exact digest exists and is signed, so you can audit its
  contents and re-scan it against new CVEs at any time.

It does **not** assert the absence of vulnerabilities — that is what the SBOM re-scan above is
for. Verification proves *provenance and integrity*. The scan tells you about *current
exposure*.
