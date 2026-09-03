# What digest `gh attestation verify` resolves for a multi-arch tag

Research ticket: [#1147](https://github.com/winniel123/verge-asm/issues/1147) (part of
map [#1064](https://github.com/winniel123/verge-asm/issues/1064) "Release pipeline").
This is **investigation only**. It builds on closed research
[#1065](https://github.com/winniel123/verge-asm/issues/1065),
[#1074](https://github.com/winniel123/verge-asm/issues/1074),
[#1076](https://github.com/winniel123/verge-asm/issues/1076) and
[#1077](https://github.com/winniel123/verge-asm/issues/1077).

## Short answer

`gh attestation verify oci://<image>:<tag>` resolves the tag to the **index digest**.
It does not pick a platform digest. It never reads the children of the index.
`--bundle-from-oci` does not change that resolution.

An attestation bound to a child manifest digest is **not** discoverable from the index
reference. The operator must name the child digest in the reference. That costs the
operator one extra command and one extra concept.

## Measurement environment

Every measurement in this document ran on **2026-09-03**.

| item | value |
| --- | --- |
| `gh --version` | `gh version 2.96.0 (2026-07-02)` |
| `docker buildx version` | `github.com/docker/buildx v0.35.0-desktop.2 b554ce1decd8b509893b1e7c6227eabfb923d094` |
| host platform | `windows/amd64`, Docker Desktop engine `linux/amd64` |
| `gh` account | `winniel123`, token scopes `gist`, `read:org`, `repo` |

The measurement subject is `ghcr.io/coder/coder:latest`. It is a public GHCR image.
It is a genuine manifest list with three platforms. It carries GitHub artifact
attestations. A second subject, `ghcr.io/github/github-mcp-server:latest`, gives an
independent reading on an OCI image index.

```
$ docker buildx imagetools inspect ghcr.io/coder/coder:latest
Name:      ghcr.io/coder/coder:latest
MediaType: application/vnd.docker.distribution.manifest.list.v2+json
Digest:    sha256:92be096e4ad26bd6490a40d0c19d69a729290f439db6ebc1f7a03b292b4fadb9

Manifests:
  Name:      ...@sha256:06ca8fb728910f9045d78100f480a0a5d880eca78543c6de16eef647e4e4cb0e
  Platform:  linux/amd64

  Name:      ...@sha256:460859068b9e1bfbea81febb132ae7a58fb9f7fd7289aa4cdc15d92de039c5a9
  Platform:  linux/arm64

  Name:      ...@sha256:00823b036181c2a67a42871c259e83fe8d2d3a9ef2c4ccc31a2d3421fe2d7127
  Platform:  linux/arm/v7
```

The host is `amd64`. The `amd64` child digest differs from the index digest. So the
"index digest or local platform digest" question has a distinguishable answer here.

---

## 1. Which digest the CLI resolves a tag to

**The index digest.** Measured, and confirmed against the `cli/cli` source.

### Measurement 1 — verify the tag, bundle from the GitHub API

```
$ gh attestation verify oci://ghcr.io/coder/coder:latest --repo coder/coder \
    --format json --jq '.[].verificationResult.statement.subject'
[{"digest":{"sha256":"92be096e4ad26bd6490a40d0c19d69a729290f439db6ebc1f7a03b292b4fadb9"},"name":"ghcr.io/coder/coder"}]
[{"digest":{"sha256":"92be096e4ad26bd6490a40d0c19d69a729290f439db6ebc1f7a03b292b4fadb9"},"name":"ghcr.io/coder/coder"}]
exit=0
```

The verified subject digest is `92be096e…`. That is the index digest. It is not the
`amd64` child digest `06ca8fb7…`.

### Measurement 2 — an index with no attestation names the digest it looked up

```
$ gh attestation verify oci://ghcr.io/github/github-mcp-server:latest \
    --repo github/github-mcp-server
Error: HTTP 404: Not Found (https://api.github.com/repos/github/github-mcp-server/attestations/sha256:fbec75de11c255213fa08d80fb166abe73d851fff631c51c0079872967720699?per_page=30&predicate_type=https%3A%2F%2Fslsa.dev%2Fprovenance%2Fv1)
exit=1
```

`sha256:fbec75de…` is the index digest of that image. The error URL states the digest
the CLI queried. This subject is an `application/vnd.oci.image.index.v1+json` index, so
the behaviour holds for both index media types.

### The code path

GitHub Docs do not state which digest a tag resolves to for a manifest list. The
container-image example on the "Use artifact attestations" page shows a tag reference
and says nothing about architecture. So the source settles it.

`pkg/cmd/attestation/artifact/image.go` calls `client.GetImageDigest`:

```go
digest, nameRef, err := client.GetImageDigest(named.String())
```

`pkg/cmd/attestation/artifact/oci/client.go` implements it with `remote.Get`:

```go
desc, err := c.get(name, remote.WithAuthFromKeychain(authn.DefaultKeychain))
...
return &desc.Digest, name, nil
```

`c.get` is `remote.Get` from `go-containerregistry`. Its documentation states: "Get
returns a remote.Descriptor for the given reference. The response from the registry is
left un-interpreted, for the most part."

That is the whole answer. `remote.Get` performs no platform matching. The sibling
`remote.Image` does perform platform matching, and the CLI does not call it. The
descriptor digest is therefore the digest of the manifest the registry served for the
reference. For a tag on a manifest list, that is the index digest.

`runVerify` computes that digest once and reuses it for every downstream call.

**Sources.**
`https://github.com/cli/cli/blob/v2.96.0/pkg/cmd/attestation/artifact/oci/client.go`,
`https://github.com/cli/cli/blob/v2.96.0/pkg/cmd/attestation/artifact/image.go`,
`https://pkg.go.dev/github.com/google/go-containerregistry/pkg/v1/remote#Get`.

---

## 2. Whether `--bundle-from-oci` changes the resolution

**No.**

### Measurement 3 — the same tag, bundle from the registry

```
$ gh attestation verify oci://ghcr.io/coder/coder:latest --repo coder/coder \
    --bundle-from-oci --format json --jq '.[].verificationResult.statement.subject'
[{"digest":{"sha256":"92be096e4ad26bd6490a40d0c19d69a729290f439db6ebc1f7a03b292b4fadb9"},"name":"ghcr.io/coder/coder"}]
[{"digest":{"sha256":"92be096e4ad26bd6490a40d0c19d69a729290f439db6ebc1f7a03b292b4fadb9"},"name":"ghcr.io/coder/coder"}]
exit=0
```

The subject digest is identical to measurement 1.

The flag selects the attestation **source**, not the digest. `runVerify` derives the
digest before it branches. `getAttestations` then reads the GitHub API or the registry
with the digest already fixed. The manual agrees: "By default, the command fetches
attestations via the GitHub API using `--owner` or `--repo` values. To fetch from the
artifact's OCI registry instead, use `--bundle-from-oci`."

**One thing `--bundle-from-oci` does change is the failure message.** Measurement 5
below records it.

**Sources.** `https://cli.github.com/manual/gh_attestation_verify`,
`https://github.com/cli/cli/blob/v2.96.0/pkg/cmd/attestation/verify/verify.go`.

---

## 3. Whether a child-bound attestation is discoverable from the index reference

**No, on both routes.** A child-bound attestation is reachable only when the operator
names the child digest.

### Measurement 4 — the GitHub attestations API, child digest, no attestation

```
$ gh attestation verify oci://ghcr.io/coder/coder@sha256:06ca8fb728910f9045d78100f480a0a5d880eca78543c6de16eef647e4e4cb0e \
    --repo coder/coder
Error: HTTP 404: Not Found (https://api.github.com/repos/coder/coder/attestations/sha256:06ca8fb728910f9045d78100f480a0a5d880eca78543c6de16eef647e4e4cb0e?per_page=30&predicate_type=https%3A%2F%2Fslsa.dev%2Fprovenance%2Fv1)
exit=1
```

The CLI queried the child digest verbatim. It did not fall back to the index digest.
It did not walk the index. The `coder` provenance sits on the index and stays
unreachable from this reference.

### Measurement 5 — the registry route, child digest, no attestation

```
$ gh attestation verify oci://ghcr.io/coder/coder@sha256:460859068b9e1bfbea81febb132ae7a58fb9f7fd7289aa4cdc15d92de039c5a9 \
    --repo coder/coder --bundle-from-oci
Error: no attestations found in the OCI registry. Retry the command without the --bundle-from-oci flag to check GitHub for the attestation
exit=1
```

### Measurement 6 — the attestations API is keyed on the subject digest alone

```
$ gh api repos/coder/coder/attestations/sha256:92be096e…  # index digest
2 attestations, each with subject digest 92be096e… and predicateType https://slsa.dev/provenance/v1

$ gh api repos/coder/coder/attestations/sha256:06ca8fb7…  # amd64 child
{"message":"Not Found","documentation_url":"https://docs.github.com/rest/repos/attestations#list-attestations","status":"404"}

$ gh api repos/coder/coder/attestations/sha256:46085906…  # arm64 child
{"message":"Not Found",…,"status":"404"}
```

The REST reference states the parameter plainly: `subject_digest` "should be set to the
attestation's subject's SHA256 digest, in the form sha256:HEX_DIGEST". The
documentation says nothing about OCI manifest structure. The endpoint is a digest
lookup. It has no view of parents or children.

### Measurement 7 — GHCR does not serve the OCI Referrers API, but does serve the fallback tag

```
$ curl -H "Authorization: Bearer <anon pull token>" \
    https://ghcr.io/v2/coder/coder/referrers/sha256:92be096e…
http=404
{"errors":[{"code":"MANIFEST_UNKNOWN","message":"manifest unknown"}]}

$ curl -H "Authorization: Bearer <anon pull token>" \
    https://ghcr.io/v2/coder/coder/manifests/sha256-92be096e…
http=200
{"mediaType":"application/vnd.oci.image.index.v1+json","schemaVersion":2,"manifests":[
  {"digest":"sha256:5f6c02ed…","artifactType":"application/vnd.dev.sigstore.bundle.v0.3+json",
   "annotations":{"dev.sigstore.bundle.predicateType":"https://slsa.dev/provenance/v1", …}},
  {"digest":"sha256:9c28f299…","artifactType":"application/vnd.dev.sigstore.bundle.v0.3+json", …}]}
```

This is the primary measurement that #1065 could not make. #1065 recorded community
evidence, unconfirmed, that GHCR does not implement the Referrers API. The measurement
above confirms the 404. It also shows why `--bundle-from-oci` still works.

The OCI distribution-spec defines the fallback. It states: "If the referrers API
returns a 404, the client MUST fallback to pulling the referrers tag schema." The tag
replaces the digest colon with a hyphen, giving `sha256-<hex>`.
`go-containerregistry`'s `remote.Referrers` implements that fallback, and the `gh` OCI
client calls `remote.Referrers(ref.Context().Digest(digest))`.

The fallback tag is keyed on the exact subject digest. So the registry route has the
same blindness as the API route. It looks up one digest and stops.

The `gh` OCI client also filters the referrer list. It keeps only descriptors whose
`ArtifactType` starts with `application/vnd.dev.sigstore.bundle`. A cosign-style
referrer does not match. Measured on `ghcr.io/aquasecurity/trivy:latest`, whose
fallback-tag referrers carry `artifactType: application/vnd.oci.empty.v1+json`.

### Can a child-bound attestation be created at all?

The `actions/attest` README states the constraint on `push-to-registry`: it "Requires
that the resolved subject is a single fully-qualified OCI image reference with a
SHA-256 digest". The action's own source normalises the subject to a bare
`registry/repository` reference plus a digest, and pushes the referrer against that
digest. Nothing in that path inspects the manifest. A child manifest digest is a
manifest digest like any other.

So a child-bound attestation is creatable, and it is reachable by a child-digest
reference. **This last step is inference from the measured code path, not a positive
measurement.** No public per-platform-attested GHCR image was found to measure it
against. See measurement 8.

### Measurement 8 — no public per-platform example found

Six public GHCR projects were probed on 2026-09-03. Each index digest and each child
digest went to `GET /repos/{owner}/{repo}/attestations/{subject_digest}`.

| image | index digest carries attestations | any child digest carries attestations |
| --- | --- | --- |
| `ghcr.io/coder/coder:latest` | yes, 2 | no, 404 on all 3 |
| `ghcr.io/astral-sh/ruff:latest` | yes, 1 | no, 404 on all 4 |
| `ghcr.io/astral-sh/ty:latest` | yes, 1 | no, 404 on all 4 |
| `ghcr.io/mitmproxy/mitmproxy:latest` | yes, 1 | no, 404 on all 2 |
| `ghcr.io/elk-zone/elk:latest` | yes, 1 | no, 404 on all 2 |
| `ghcr.io/aquasecurity/trivy:latest` | no, 404 | no, 404 on all 8 |

Every attested project in this sample binds to the index digest. The sample is small
and it is not random. It supports one claim only. The index-bound pattern is the
common one, and a per-platform subject is unusual enough that no example turned up.

`ghcr.io/aquasecurity/trivy` has registry referrers on its index digest but no GitHub
attestations API entry. Its referrers are cosign-style, not Sigstore bundles.

### Measurement 9 — what digest a `docker pull` gives the operator

```
$ docker pull -q alpine:3.22
$ docker image inspect alpine:3.22 --format '{{ json .RepoDigests }}'
["alpine@sha256:14358309a308569c32bdc37e2e0e9694be33a9d99e68afb0f5ff33cc1f695dce"]
$ docker buildx imagetools inspect alpine:3.22   # index digest
Digest:    sha256:14358309a308569c32bdc37e2e0e9694be33a9d99e68afb0f5ff33cc1f695dce
```

A `docker pull` of a multi-arch tag records the **index** digest in `RepoDigests`. It
does not record the platform digest of the image it actually ran. So the digest an
operator gets by the obvious route is the index digest. A per-platform subject is
invisible on that route as well.

---

## 4. The operator commands that verify a per-platform SBOM attestation

Three commands. The first two exist only to find the child digest.

```sh
# 1. Log in. GitHub Docs require this for any oci:// reference.
docker login ghcr.io

# 2. List the platform digests of the index.
docker buildx imagetools inspect ghcr.io/winniel123/verge-asm/web:vX.Y.Z \
  --format '{{ range .Manifest.Manifests }}{{ .Platform.OS }}/{{ .Platform.Architecture }}{{ if .Platform.Variant }}/{{ .Platform.Variant }}{{ end }} {{ .Digest }}
{{ end }}'

# 3. Verify the SBOM attestation on one platform digest.
gh attestation verify \
  oci://ghcr.io/winniel123/verge-asm/web@sha256:<platform-digest> \
  --repo winniel123/verge-asm \
  --predicate-type https://spdx.dev/Document/v2.3 \
  --signer-workflow winniel123/verge-asm/.github/workflows/release.yml \
  --source-ref refs/tags/vX.Y.Z \
  --deny-self-hosted-runners
```

Step 3 repeats once per platform, and once per image. #1074 rules four SBOM
attestations per release. So the operator runs step 3 four times.

**`--predicate-type` is mandatory here.** The manual states the default: "By default,
the command enforces the `https://slsa.dev/provenance/v1` predicate type." An SBOM
attestation carries a different predicate, so the default rejects it.

**The exact SPDX predicate URI is version-derived.** `actions/attest` builds it from
the SBOM itself:

```ts
const version = spdxVersion.split('-')[1]
return { type: `https://spdx.dev/Document/v${version}`, params: sbom }
```

An SPDX 2.3 document therefore gives `https://spdx.dev/Document/v2.3`. A plain
`https://spdx.dev/Document` does not match. #1074 fixes the format at SPDX 2.3, so the
URI above is correct for this project. Re-measure it against a real release before the
guide lands. A CycloneDX document would give `https://cyclonedx.org/bom`.
Source: `https://github.com/actions/attest/blob/main/src/sbom.ts`.

**`--bundle-from-oci` is an optional fourth form.** Add it to step 3 when the operator
prefers the registry copy over the GitHub API. Measurements 3 and 5 show that it
changes neither the digest nor the required flags.

**A note outside this ticket's bullets.** `actions/attest-sbom` now emits a deprecation
warning and delegates to `actions/attest@v4.2.2`. Read on 2026-09-03 at
`https://github.com/actions/attest-sbom/blob/main/action.yml`. #1076 chose the wrapper
pair deliberately. That choice may need a second look.

### Measurement 10 — `--signer-workflow` does not satisfy the owner-or-repo requirement

#1076 §5 named this point as unmeasured. It is measured now.

```
$ gh attestation verify oci://ghcr.io/coder/coder:latest \
    --signer-workflow coder/coder/.github/workflows/release.yaml
at least one of the flags in the group [owner repo] is required
exit=1

$ gh attestation verify oci://ghcr.io/coder/coder:latest
at least one of the flags in the group [owner repo] is required
exit=1

$ gh attestation verify oci://ghcr.io/coder/coder:latest --repo coder/coder \
    --signer-workflow coder/coder/.github/workflows/release.yaml \
    --format json --jq '.[].verificationResult.signature.certificate.subjectAlternativeName'
https://github.com/coder/coder/.github/workflows/release.yaml@refs/tags/v2.37.0
https://github.com/coder/coder/.github/workflows/release.yaml@refs/tags/v2.37.0
exit=0
```

`--signer-workflow` alone fails. It needs `--owner` or `--repo` beside it. The manual
already states the rule: "At minimum, either `--owner` or `--repo` is required."
The command in #1076 §5 carries `--repo`, so it is correct as written.

---

## 5. Whether `docker buildx imagetools inspect --format` is reliable enough to publish

**Yes for this project, with two stated caveats.** The `--format` flag is documented.
It defaults to `{{.Manifest}}` and takes Go template syntax. The template fields are
`.Name`, `.Manifest`, `.Image`, `.Provenance` and `.SBOM`.
Source: `https://docs.docker.com/reference/cli/docker/buildx/imagetools/inspect/`.

### Measurement 11 — the #1074 format string against a real GHCR index

```
$ docker buildx imagetools inspect ghcr.io/coder/coder:latest \
    --format '{{ range .Manifest.Manifests }}{{ .Platform.OS }}/{{ .Platform.Architecture }} {{ .Digest }}
{{ end }}'
linux/amd64 sha256:06ca8fb728910f9045d78100f480a0a5d880eca78543c6de16eef647e4e4cb0e
linux/arm64 sha256:460859068b9e1bfbea81febb132ae7a58fb9f7fd7289aa4cdc15d92de039c5a9
linux/arm sha256:00823b036181c2a67a42871c259e83fe8d2d3a9ef2c4ccc31a2d3421fe2d7127
exit=0
```

It works. It needs no `docker login` against a public GHCR image.

**Caveat 1. The #1074 string drops the platform variant.** The third row reads
`linux/arm`. The real platform is `linux/arm/v7`. Two `arm` variants in one index would
print two identical labels. The fix is one conditional:

```
{{ .Platform.OS }}/{{ .Platform.Architecture }}{{ if .Platform.Variant }}/{{ .Platform.Variant }}{{ end }}
```

Measured on the same image, that prints `linux/arm/v7`. verge-asm ships `linux/amd64`
and `linux/arm64` only, so the defect does not bite today. The guide should still carry
the safe form.

### Measurement 12 — an index that carries BuildKit attestation manifests

```
$ docker buildx imagetools inspect ghcr.io/github/github-mcp-server:latest --format '<same>'
linux/amd64 sha256:48b071b92a297eb9b8ddb8dd87ccb4c75dbca6b0867eff034de4148722e0d164
linux/arm64 sha256:fdb156085a90973733ee6f3c1b212de14559c0e91b0ba05c35a0a2164505f88b
unknown/unknown sha256:0c87586c75d187febca542869abcd747c63a540895c008482dedc44af74badd2
unknown/unknown sha256:6e6b0498e20812a5f5f4f26104cc06d101892e3f3e4eef2865d3e6940bf45417
```

**Caveat 2. BuildKit attestation manifests print as `unknown/unknown`.** They carry
`vnd.docker.reference.type: attestation-manifest`. An operator who verifies every
printed digest would waste two commands and read two failures.
[#1080](https://github.com/winniel123/verge-asm/issues/1080) sets `provenance = false`
and `sbom = false`, so a verge-asm index carries no such child. The guide should
nonetheless tell the reader to skip an `unknown/unknown` row.

### Measurement 13 — the template errors on a single-platform image

```
$ docker buildx imagetools inspect ghcr.io/dependabot/dependabot-core:latest --format '<same>'
ERROR: template: :1:18: executing "" at <.Manifest.Manifests>: can't evaluate field Manifests in type interface {}
exit=1
```

This is a hard error and a non-zero exit. Docker's own reference states the cause: for
a non-manifest-list reference, the command "returns a single manifest object rather
than a collection". A verge-asm release always publishes an index, so this path should
not occur. A reader who copies the command onto some other image will hit it. One
sentence in the guide covers it.

**Verdict.** The command is reliable enough to print, with the variant-aware template
and the two caveats stated.

---

## Notes against the local tree

- `docs/guides/verifying-releases.md` must carry a three-command SBOM path and a
  one-command provenance path. #1074 already flagged that this file names the wrong
  predicate. This research adds the exact SPDX URI and the variant-aware template.
- #1077 fixes the image names as `ghcr.io/winniel123/verge-asm/web` and
  `ghcr.io/winniel123/verge-asm/worker`. Both commands above use the nested form.
- #1080 keeps each index at two children. So step 2 of the SBOM path prints two rows
  per image, and step 3 runs twice per image.

---

## What this changes on the map

**#1076's index-digest verify command: confirms.** The provenance subject is the index
digest, and `gh attestation verify oci://<image>:<tag>` resolves that tag to the index
digest. The two agree. The single-command provenance path in #1076 §5 is correct as
written, and measurement 10 closes the one point that section left open.

**#1074's per-platform SBOM subject-digest ruling: leaves open.** The ruling is not
overturned. The CLI can reach a child manifest digest, because a digest reference
bypasses index resolution entirely. Measurement 4 shows the CLI querying a child digest
verbatim. So the per-platform subject remains verifiable.

The ruling does carry a cost that is now measured rather than assumed.

- The operator needs two extra commands and a second digest concept. #1074 predicted
  one extra step. Measurements 11 to 13 show it needs a variant-aware template and two
  documented caveats.
- Neither route discovers a child-bound attestation from the tag. Measurements 4, 5, 6
  and 7 show that both the API and the registry look up exactly one digest.
- The digest an operator gets from `docker pull` is the index digest. Measurement 9
  shows it. A per-platform subject is invisible on the natural path.
- No public GHCR project in a six-project sample binds an attestation to a child
  manifest. Measurement 8. The pattern is unusual, so the guide carries more of the
  explanation than a common pattern would need.
- The final link in the chain is inference, not measurement. No public image exists to
  prove a child-bound attestation verifies end to end. The first verge-asm release is
  the measurement. If it fails, #1074 must be reopened.

**#1065's GHCR referrers caveat: confirms, and upgrades the evidence.** #1065 recorded
community evidence, unconfirmed. Measurement 7 is a primary measurement. GHCR returns
404 `MANIFEST_UNKNOWN` on the Referrers API. GHCR serves the distribution-spec fallback
tag with HTTP 200. `--bundle-from-oci` works because `go-containerregistry` implements
that fallback.

---

## Sources

Primary.

- `gh attestation verify` manual: https://cli.github.com/manual/gh_attestation_verify
- `cli/cli` OCI client, v2.96.0: https://github.com/cli/cli/blob/v2.96.0/pkg/cmd/attestation/artifact/oci/client.go
- `cli/cli` image artifact, v2.96.0: https://github.com/cli/cli/blob/v2.96.0/pkg/cmd/attestation/artifact/image.go
- `cli/cli` verify command, v2.96.0: https://github.com/cli/cli/blob/v2.96.0/pkg/cmd/attestation/verify/verify.go
- `go-containerregistry` `remote.Get`: https://pkg.go.dev/github.com/google/go-containerregistry/pkg/v1/remote#Get
- `go-containerregistry` `remote.Referrers` and the fallback tag: https://github.com/google/go-containerregistry/blob/main/pkg/v1/remote/referrers.go
- OCI distribution-spec, Referrers API and Referrers Tag Schema: https://github.com/opencontainers/distribution-spec/blob/main/spec.md
- GitHub REST, list attestations: https://docs.github.com/en/rest/repos/attestations
- GitHub Docs, use artifact attestations: https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/use-artifact-attestations
- `actions/attest` README and source: https://github.com/actions/attest
- `actions/attest` SBOM predicate: https://github.com/actions/attest/blob/main/src/sbom.ts
- `actions/attest-sbom` action definition: https://github.com/actions/attest-sbom/blob/main/action.yml
- Docker Docs, `buildx imagetools inspect`: https://docs.docker.com/reference/cli/docker/buildx/imagetools/inspect/

Secondary.

- None. Every claim above rests on a primary source or on a dated measurement.
