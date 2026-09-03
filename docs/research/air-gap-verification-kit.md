# Research: what a `cosign save` layout captures, and the air-gap kit

Research ticket: [#1148](https://github.com/winniel123/verge-asm/issues/1148) (part of
map [#1064](https://github.com/winniel123/verge-asm/issues/1064)). This is investigation
only. The operator-guide rewrite owns the prose that an operator reads.

The release model under test is the one that
[#1075](https://github.com/winniel123/verge-asm/issues/1075),
[#1074](https://github.com/winniel123/verge-asm/issues/1074) and
[#1076](https://github.com/winniel123/verge-asm/issues/1076) fixed. A release carries 2
index signatures, 4 per-platform signatures, 2 provenance attestations, 4 SBOM
attestations and 1 blob signature over `SHA256SUMS`.

---

## Short answer

| Question | Answer |
| --- | --- |
| Does `cosign save` copy child signatures? | No. Measured. |
| Does `cosign save` copy attestations? | It copies a cosign `.att` attestation on the named digest. It does not make a native GitHub attestation reachable. Measured. |
| Can `cosign verify --local-image` target a child digest? | No. The flag takes a directory path and nothing else. Measured. |
| Can `cosign verify-attestation --local-image` verify from a layout? | Yes for a cosign `.att` attestation. No for a native GitHub attestation. The command that reaches the second one is `gh attestation verify --bundle --custom-trusted-root`. Measured. |
| One saved layout per digest? | Yes. Six digests need six directories. A second save into one directory replaces `index.json`. Measured. |
| Does the kit need the `gh` path as well as cosign? | Yes. cosign alone cannot verify the 6 native GitHub attestations offline. Measured. |

---

## Measurement setup

Every measurement below ran on 2026-09-03.

- Host: Windows 11 Pro 10.0.26200. Docker Engine 29.6.2. The image store is the
  containerd snapshotter.
- cosign: the container image `ghcr.io/sigstore/cosign/cosign:v3.1.3`, image digest
  `sha256:9e5c2f2edc34351160407ca3416c61855bdf9403c3c5936e0f0be7fc261611b8`.

```
$ docker run --rm ghcr.io/sigstore/cosign/cosign:v3.1.3 version
GitVersion:    v3.1.3
GitCommit:     11926fa5bbbbde47e88fc006b625a17769b743b2
GitTreeState:  clean
BuildDate:     2026-08-05T23:43:27Z
GoVersion:     go1.26.4
Compiler:      gc
Platform:      linux/amd64
```

- `gh` version 2.96.0 (2026-07-02), on the Windows host.
- A run with `--network none` proves that a cosign command makes no network call.
- A run with `HTTPS_PROXY=http://127.0.0.1:1` and `GH_TOKEN=invalid` proves the same for
  a `gh` command.

### Test subjects

verge-asm publishes no signed release yet. Four public images stand in. Each one was
checked for the property it must demonstrate before it was used.

| Subject | Shape | Why it was chosen |
| --- | --- | --- |
| `ghcr.io/sigstore/cosign/cosign@sha256:f1946d0f...` (v2.5.3) | OCI index, 5 platforms, legacy `.sig` tags on the index **and** on each child | Closest match to the verge-asm model. #1075 pins `--registry-referrers-mode legacy`. |
| `ghcr.io/sigstore/cosign/cosign@sha256:9e5c2f2e...` (v3.1.3) | OCI index, 6 platforms, OCI 1.1 referrer signatures on the index and on each child | Shows what happens when the signature is a referrer rather than a `.sig` tag. |
| `cgr.dev/chainguard/static@sha256:f51c2493...` | OCI index, cosign `.att` attestation on the index | A cosign-written attestation. |
| `ghcr.io/coder/coder@sha256:92be096e...` | Docker manifest list, 2 native GitHub `https://slsa.dev/provenance/v1` referrers on the index, plus cosign `.att` SPDX attestations | The only subject that carries a native GitHub attestation pushed to a registry. |

`gcr.io/distroless/static-debian12` was rejected as a subject. cosign v3.1.3 cannot match
a predicate type on its attestation, and it fails the same way online and offline. That
failure is a property of the subject, so the subject proves nothing about `cosign save`.

---

## 1. `cosign save` does not copy a child signature

**Measured.** The v2.5.3 index and its `linux/amd64` child both carry a signature in the
registry.

```
$ docker run --rm ghcr.io/sigstore/cosign/cosign:v3.1.3 tree \
    ghcr.io/sigstore/cosign/cosign@sha256:920845e07017a9abe50a0e5a4b883cbc761228691ea80ebb16661b522e71a0bd
└── 🔐 Signatures for an image tag: ghcr.io/sigstore/cosign/cosign:sha256-920845e0....sig
   ├── 🍒 sha256:13fa89371d5e222e1e38af59518cf0f33ca6b498c967a0eb1d9184a1d24794f7
```

Save the index.

```
$ docker run --rm -v "$PWD:/work" -w /work --user 0:0 \
    ghcr.io/sigstore/cosign/cosign:v3.1.3 save \
    ghcr.io/sigstore/cosign/cosign@sha256:f1946d0f30fc8e3777b02f2201e02efdba9fe38f4918162f937052fac98e083f \
    --dir /work/legacy
```

The layout holds three things. It holds the index. It holds all 5 child manifests and
their layers. It holds one `sigs` entry, and that entry is the signature of the index.

```json
{
   "schemaVersion": 2,
   "mediaType": "application/vnd.oci.image.index.v1+json",
   "manifests": [
      { "digest": "sha256:f1946d0f...", "annotations": { "kind": "dev.cosignproject.cosign/imageIndex" } },
      { "digest": "sha256:0e1804ec...", "annotations": { "kind": "dev.cosignproject.cosign/sigs" } }
   ]
}
```

The child signature layer `sha256:13fa8937...` is absent from `blobs/sha256/`. The child
manifest `sha256:920845e0...` is present. So the layout carries the child image and drops
the proof that the child was signed.

**The source agrees.** `cmd/cosign/cli/save.go` in v3.1.3 resolves one entity and writes
it. It never walks the children of an index.

```go
if _, ok := se.(oci.SignedImageIndex); ok {
    sii, err := ociremote.SignedImageIndex(ref, regClientOpts...)
    ...
    return layout.WriteSignedImageIndex(opts.Directory, sii)
}
```

`pkg/oci/layout/write.go` then appends the index and calls `writeSignedEntity`, which
reads `se.Signatures()` and `se.Attestations()` for that one entity only.

`cosign save` has no `--recursive` flag and no `--platform` flag. The v3.1.3 flag list
holds `--dir`, the registry connection flags, `--output-file`, `--timeout` and
`--verbose`
([cosign_save.md](https://github.com/sigstore/cosign/blob/v3.1.3/doc/cosign_save.md)).

---

## 2. `cosign save` copies one kind of attestation

**A cosign `.att` attestation is copied. Measured.** The chainguard subject produced a
layout with an `atts` entry.

```
$ ... save cgr.dev/chainguard/static@sha256:f51c2493... --dir /work/cg-layout
$ cat cg-layout/index.json
      { "digest": "sha256:f51c2493...", "annotations": { "kind": "dev.cosignproject.cosign/imageIndex" } },
      { "digest": "sha256:66c81181...", "annotations": { "kind": "dev.cosignproject.cosign/sigs" } },
      { "digest": "sha256:23470e09...", "annotations": { "kind": "dev.cosignproject.cosign/atts" } }
```

**A native GitHub attestation is not reachable. Measured.** The coder subject carries two
`https://slsa.dev/provenance/v1` referrers on its index. `cosign save` wrote a layout that
names only the `.att` attestations.

```
$ ... save ghcr.io/coder/coder@sha256:92be096e... --dir /work/coder-layout
$ cat coder-layout/index.json
      { "digest": "sha256:92be096e...", "annotations": { "kind": "dev.cosignproject.cosign/imageIndex" } },
      { "digest": "sha256:818471e6...", "annotations": { "kind": "dev.cosignproject.cosign/atts" } }
```

The two provenance referrer manifests and their two bundle layers are present in
`blobs/sha256/`. Nothing in `index.json` points to them. They are orphan bytes.

**Why the bytes arrive and the pointer does not.** `SaveCmd` runs a loop over the
referrers of the target digest before it saves the target.

```go
for _, manifest := range indexManifest.Manifests {
    if manifest.ArtifactType == "" { continue }
    ...
    err = layout.WriteSignedImage(opts.Directory, si)
}
```

Each `WriteSignedImage` and each `WriteSignedImageIndex` starts with
`layout.Write(path, empty.Index)`. That call resets `index.json`. So every loop iteration
erases the entry the last one wrote, and the final `WriteSignedImageIndex` erases them
all. The blobs stay because a layout write appends blobs and never prunes them.

This reading is mine, from the v3.1.3 source. I found no cosign issue that reports it.
Treat the cause as unconfirmed. The effect is measured.

---

## 3. `--local-image` cannot target a child digest

**Measured.** The flag takes a directory path. Appending a digest makes cosign look for a
directory with that name.

```
$ ... verify ... --local-image "/work/legacy@sha256:920845e0..."
Error: checking local image format: loading OCI layout from /work/legacy@sha256:920845e0...:
stat /work/legacy@sha256:920845e0.../index.json: no such file or directory
```

The v3.1.3 help text says the same thing. The flag states "whether the specified image is
a path to an image saved locally via 'cosign save'"
([cosign_verify.md](https://github.com/sigstore/cosign/blob/v3.1.3/doc/cosign_verify.md)).

**The layout root verifies offline.** This confirms the #1066 finding with a measurement.

```
$ docker run --rm --network none -v "$PWD:/work" -w /work --user 0:0 \
    ghcr.io/sigstore/cosign/cosign:v3.1.3 verify \
    --certificate-identity-regexp ".*" --certificate-oidc-issuer-regexp ".*" \
    --trusted-root /work/trusted_root.json --local-image /work/legacy

Verification for /work/legacy --
The following checks were performed on each of these signatures:
  - The cosign claims were validated
  - Existence of the claims in the transparency log was verified offline
  - The code-signing certificate was verified using trusted certificate authority certificates
```

The reported subject is the index digest `sha256:f1946d0f...`. The 5 child digests get no
verdict.

**`--trusted-root` is not optional.** The same command without the flag, and with
`--network none`, fails.

```
WARNING: Could not fetch trusted_root.json from the TUF repository. Continuing with
individual targets. Error from TUF: ... dial tcp: lookup tuf-repo-cdn.sigstore.dev
Error: setting up clients and keys: getting rekor public keys: updating local metadata
and targets: error updating to TUF remote mirror: tuf: failed to download 13.root.json
```

---

## 4. `verify-attestation --local-image` reaches one attestation kind

**A cosign `.att` attestation verifies offline. Measured.**

```
$ docker run --rm --network none -v "$PWD:/work" -w /work --user 0:0 \
    ghcr.io/sigstore/cosign/cosign:v3.1.3 verify-attestation --type spdx \
    --certificate-identity-regexp ".*" --certificate-oidc-issuer-regexp ".*" \
    --trusted-root /work/trusted_root.json --local-image /work/cg-layout

Verification for /work/cg-layout --
...
Certificate subject: https://github.com/chainguard-images/images/.github/workflows/release.yaml@refs/heads/main
Certificate issuer URL: https://token.actions.githubusercontent.com
```

**A native GitHub attestation does not. Measured.**

```
$ ... verify-attestation --type slsaprovenance1 ... --local-image /work/coder-layout
Error: none of the attestations matched the predicate type: slsaprovenance1,
found: https://spdx.dev/Document,https://spdx.dev/Document
```

The same command against the registry succeeds, so the loss happens at save time and not
at verify time.

```
$ ... verify-attestation --type slsaprovenance1 ... ghcr.io/coder/coder@sha256:92be096e...
Verification for ghcr.io/coder/coder@sha256:92be096e... --
...
{"payload":"eyJfdHlwZSI6Imh0dHBzOi8vaW4tdG90by5pby9TdGF0ZW1lbnQvdjEi...
```

### The command that does reach it

`gh attestation verify`, with a bundle file and a custom trusted root. The artifact
argument is a file, and `gh` hashes that file
([gh_attestation_verify](https://cli.github.com/manual/gh_attestation_verify)).

An OCI digest is the SHA-256 of the manifest bytes. A saved layout stores those bytes at
`blobs/sha256/<digest>`. So the layout supplies the artifact file, and the kit needs no
extra copy of the manifest.

**Measured, with the network blackholed.** `HTTPS_PROXY` and `HTTP_PROXY` point at
`127.0.0.1:1`, and `GH_TOKEN` is `invalid`.

```
$ gh attestation verify \
    coder-layout/blobs/sha256/92be096e4ad26bd6490a40d0c19d69a729290f439db6ebc1f7a03b292b4fadb9 \
    --repo coder/coder \
    --bundle ghkit/sha256-92be096e....jsonl \
    --custom-trusted-root ghkit/trusted_root.jsonl \
    --format json
exit=0  stdout bytes=26284
```

Three controls prove the check is real.

1. `--repo coder/wrong-repo` exits 1 with `Error: verifying with issuer "sigstore.dev"`.
2. The child manifest blob in place of the index manifest blob exits 1 with the same
   error. The digest binding holds.
3. The same command without `--custom-trusted-root` exits 1 with
   `error creating Sigstore verifier: no valid Sigstore verifiers could be initialized`.

One operator note. In a non-interactive shell the success case printed nothing on stdout
and nothing on stderr, and it returned exit code 0. Use `--format json` to get output, or
test the exit code.

---

## 5. Six digests need six directories

**Measured.** A save of one child digest produces a layout that verifies offline.

```
$ ... save ghcr.io/sigstore/cosign/cosign@sha256:920845e0... --dir /work/child-amd64
$ cat child-amd64/index.json
      { "digest": "sha256:920845e0...", "annotations": { "kind": "dev.cosignproject.cosign/image" } },
      { "digest": "sha256:70e2bd5e...", "annotations": { "kind": "dev.cosignproject.cosign/sigs" } }

$ ... verify ... --trusted-root /work/trusted_root.json --local-image /work/child-amd64
Verification for /work/child-amd64 --
...
"docker-manifest-digest":"sha256:920845e07017a9abe50a0e5a4b883cbc761228691ea80ebb16661b522e71a0bd"
```

**Two saves into one directory do not merge. Measured.** After a save of the index and
then a save of the child into the same `--dir`, `index.json` names the child and the
child signature only. The index entry is gone.

So the kit needs one directory per digest that an operator will verify. A release has 6
signed digests, so a complete kit has 6 directories.

**The cost is disk.** Each index layout already holds every child image. Each child layout
holds that child again. For the cosign v2.5.3 subject the index layout is 310 MB and one
child layout is 66 MB. A 6-directory kit therefore stores each platform image about twice.

An operator who accepts a narrower claim can carry 2 directories instead of 6. That
operator verifies the 2 index signatures and gets no cosign verdict on a platform digest.
#1075 sends operators to platform digests, so the 6-directory kit is the one the guide
must describe.

---

## 6. The kit carries two trust roots, in two formats

cosign reads a `trusted_root.json`. `gh` reads a `trusted_root.jsonl`. They are separate
files and separate commands.

- `cosign initialize` writes
  `~/.sigstore/root/tuf-repo-cdn.sigstore.dev/targets/trusted_root.json`. Measured size on
  2026-09-03: 6787 bytes.
- `gh attestation trusted-root` writes to stdout. Measured size on 2026-09-03: 34634
  bytes.

Both files come from Sigstore, and the cosign README states that the contents "will change
without notification". The staleness contract that
[#1072](https://github.com/winniel123/verge-asm/issues/1072) wrote for one file now
applies to two.

---

## 7. Side findings

These were measured on the way. Each one touches a closed ruling.

**GHCR serves no OCI Referrers API. Measured.** `#1066` recorded this as community
evidence, unconfirmed. It now has a measurement.

```
$ curl -H "Authorization: Bearer <token>" \
    https://ghcr.io/v2/sigstore/cosign/cosign/referrers/sha256:9e5c2f2e...
HTTP 404  {"errors":[{"code":"MANIFEST_UNKNOWN","message":"manifest unknown"}]}
```

The tag fallback works instead. `GET /v2/<repo>/manifests/sha256-<digest>` returns an
index of the referrers. That fallback is how cosign and `gh` both find a pushed
attestation on GHCR.

**The `artifactType` wrinkle of cosign issue #4641 is real. Measured.** In the fallback
index, GHCR reports each referrer as
`"artifactType":"application/vnd.oci.empty.v1+json"`. The referrer manifest itself
declares `"artifactType":"application/vnd.dev.sigstore.bundle.v0.3+json"`. A tool that
filters the fallback index by `artifactType` sees the wrong value.

**`cosign download attestation` does reach a native GitHub attestation. Measured.** #1076
recorded community evidence, unconfirmed, that the command answers "manifest unknown". The
command did not fail here.

```
$ ... download attestation ghcr.io/coder/coder@sha256:92be096e...
{"mediaType":"application/vnd.dev.sigstore.bundle.v0.3+json","verificationMaterial":{...
```

It returned 4 payloads. Two are `https://slsa.dev/provenance/v1` from the referrers. Two
are `https://spdx.dev/Document` from the `.att` tag. The certificate SAN is
`https://github.com/coder/coder/.github/workflows/release.yaml@refs/tags/v2.37.0`.

This does not change the ruling of #1076. The command needs the registry, so it is an
online path. The offline path still needs `gh`.

**A saved layout also transfers the image. Measured.** A tar of the layout loads into the
daemon.

```
$ tar -C legacy -cf legacy.tar .
$ docker load -i legacy.tar
Loaded image ID: sha256:f1946d0f30fc8e3777b02f2201e02efdba9fe38f4918162f937052fac98e083f
Loaded image ID: sha256:0e1804ec096b44313e5d696f65c2d01c18058e448da0c688652cdc091429154d
```

The load reports the signature manifest as a second image. That is cosmetic. This host
runs the containerd image store. A daemon on the classic graph driver may reject an OCI
layout tar. That case is not measured.

**`actions/attest-sbom` is deprecated. Measured from its `action.yml` on 2026-09-03.** Its
first step is:

```yaml
run: |
  echo "::warning::actions/attest-sbom has been deprecated, please use actions/attest instead"
```

It then calls `actions/attest@1e69f48acb82d1966a394da916b4c1698aa569d6 # v4.2.2`. #1074
picked `actions/attest-sbom`, and #1076 kept `actions/attest-build-provenance` to match it
as a pair. The pair argument is now weaker on one side. This ticket does not re-open the
choice.

**The SBOM predicate type carries a version.** `actions/attest` `src/sbom.ts` builds the
SPDX predicate type as `` `https://spdx.dev/Document/v${version}` ``, taken from the
`spdxVersion` field. An SPDX 2.3 document therefore yields
`https://spdx.dev/Document/v2.3`. CycloneDX yields `https://cyclonedx.org/bom`. The
operator command must pass the versioned string, not the bare `https://spdx.dev/Document`.

---

## The carry-in kit, file by file

The image names below follow #1075. #1077 owns the final names. `<tag>` is the release
tag. `<web-index>` and the five sibling placeholders are the 6 digests of the release.

An operator gets the 6 digests on the connected host with the #1074 command:

```sh
docker buildx imagetools inspect ghcr.io/winniel123/verge-asm/web:<tag> \
  --format '{{ range .Manifest.Manifests }}{{ .Platform.OS }}/{{ .Platform.Architecture }} {{ .Digest }}
{{ end }}'
```

Every file below is produced on the **connected host**. Nothing in the kit is produced on
the air-gapped host. The air-gapped host only reads.

### Tools

| File | Command that produces it |
| --- | --- |
| `bin/cosign` | Download the v3.1.3 or later release binary from `https://github.com/sigstore/cosign/releases`. |
| `bin/gh` | Download the v2.96.0 or later release binary from `https://github.com/cli/cli/releases`. |

### Trust roots

| File | Command that produces it |
| --- | --- |
| `trusted_root.json` | `cosign initialize` then `cp ~/.sigstore/root/tuf-repo-cdn.sigstore.dev/targets/trusted_root.json .` |
| `trusted_root.jsonl` | `gh attestation trusted-root > trusted_root.jsonl` |

### OCI layouts, one directory per signed digest

| Directory | Command that produces it |
| --- | --- |
| `web-index/` | `cosign save ghcr.io/winniel123/verge-asm/web@sha256:<web-index> --dir web-index` |
| `web-linux-amd64/` | `cosign save ghcr.io/winniel123/verge-asm/web@sha256:<web-amd64> --dir web-linux-amd64` |
| `web-linux-arm64/` | `cosign save ghcr.io/winniel123/verge-asm/web@sha256:<web-arm64> --dir web-linux-arm64` |
| `worker-index/` | `cosign save ghcr.io/winniel123/verge-asm/worker@sha256:<worker-index> --dir worker-index` |
| `worker-linux-amd64/` | `cosign save ghcr.io/winniel123/verge-asm/worker@sha256:<worker-amd64> --dir worker-linux-amd64` |
| `worker-linux-arm64/` | `cosign save ghcr.io/winniel123/verge-asm/worker@sha256:<worker-arm64> --dir worker-linux-arm64` |

Each directory holds `oci-layout`, `index.json` and a `blobs/sha256/` tree. The file
`<dir>/blobs/sha256/<digest>` is the manifest of that digest, and the `gh` commands below
use it as the artifact argument.

### Attestation bundles, one file per attested digest

`gh attestation download` names the output file after the digest. It writes
`sha256:<digest>.jsonl` on Linux and macOS, and `sha256-<digest>.jsonl` on Windows. The
Windows form is the one measured here.

| File | Command that produces it |
| --- | --- |
| `sha256:<web-index>.jsonl` | `gh attestation download oci://ghcr.io/winniel123/verge-asm/web@sha256:<web-index> --repo winniel123/verge-asm --predicate-type https://slsa.dev/provenance/v1` |
| `sha256:<worker-index>.jsonl` | `gh attestation download oci://ghcr.io/winniel123/verge-asm/worker@sha256:<worker-index> --repo winniel123/verge-asm --predicate-type https://slsa.dev/provenance/v1` |
| `sha256:<web-amd64>.jsonl` | `gh attestation download oci://ghcr.io/winniel123/verge-asm/web@sha256:<web-amd64> --repo winniel123/verge-asm --predicate-type https://spdx.dev/Document/v2.3` |
| `sha256:<web-arm64>.jsonl` | `gh attestation download oci://ghcr.io/winniel123/verge-asm/web@sha256:<web-arm64> --repo winniel123/verge-asm --predicate-type https://spdx.dev/Document/v2.3` |
| `sha256:<worker-amd64>.jsonl` | `gh attestation download oci://ghcr.io/winniel123/verge-asm/worker@sha256:<worker-amd64> --repo winniel123/verge-asm --predicate-type https://spdx.dev/Document/v2.3` |
| `sha256:<worker-arm64>.jsonl` | `gh attestation download oci://ghcr.io/winniel123/verge-asm/worker@sha256:<worker-arm64> --repo winniel123/verge-asm --predicate-type https://spdx.dev/Document/v2.3` |

An index digest and a platform digest never collide, because provenance binds the index
and the SBOM binds the platform. So the 6 files have 6 distinct names.

### Release assets

Download each one from the release page. The command is
`gh release download <tag> --repo winniel123/verge-asm --pattern '<name>'`.

| File | Source |
| --- | --- |
| `docker-compose.yml` | Release asset, generated by the release workflow (#1072). |
| `.env.example` | Release asset, copied from the tag (#1072). |
| `docker-compose.external-db.yml` | Release asset, copied from the tag (#1072). |
| `SHA256SUMS` | Release asset (#1072). |
| `SHA256SUMS.sigstore.json` | Release asset, the `cosign sign-blob --bundle` output (#1075). |
| `web-linux-amd64.spdx.json` | Release asset (#1074). |
| `web-linux-arm64.spdx.json` | Release asset (#1074). |
| `worker-linux-amd64.spdx.json` | Release asset (#1074). |
| `worker-linux-arm64.spdx.json` | Release asset (#1074). |
| `web-linux-amd64.cdx.json` | Release asset (#1074). |
| `web-linux-arm64.cdx.json` | Release asset (#1074). |
| `worker-linux-amd64.cdx.json` | Release asset (#1074). |
| `worker-linux-arm64.cdx.json` | Release asset (#1074). |
| The Trivy scan asset set | Release asset (#1073). #1073 owns the file names and the count. This ticket does not settle them. |

### Count

The kit holds 2 tool binaries, 2 trust roots, 6 layout directories, 6 attestation bundles
and 13 named release assets. That is 23 named files plus 6 directories, plus the Trivy
asset set that #1073 owns.

### What the air-gapped host then runs

Six cosign verifies, one per layout.

```sh
cosign verify \
  --certificate-identity https://github.com/winniel123/verge-asm/.github/workflows/release.yml@refs/tags/<tag> \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --trusted-root ./trusted_root.json \
  --local-image ./web-index
```

Six `gh` verifies, one per bundle. The artifact is the manifest blob in the matching
layout.

```sh
gh attestation verify ./web-index/blobs/sha256/<web-index> \
  --repo winniel123/verge-asm \
  --signer-workflow winniel123/verge-asm/.github/workflows/release.yml \
  --bundle ./sha256:<web-index>.jsonl \
  --custom-trusted-root ./trusted_root.jsonl \
  --format json
```

The SBOM form adds `--predicate-type https://spdx.dev/Document/v2.3` and points at a
platform layout.

One blob verify, then one checksum check.

```sh
cosign verify-blob --bundle ./SHA256SUMS.sigstore.json \
  --certificate-identity https://github.com/winniel123/verge-asm/.github/workflows/release.yml@refs/tags/<tag> \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --trusted-root ./trusted_root.json \
  ./SHA256SUMS
sha256sum -c SHA256SUMS
```

`cosign verify-blob` accepts `--trusted-root` in v3.1.3. Its own help text prints the
example `cosign verify-blob --bundle artifact.sigstore.json --trusted-root
trusted_root.json <blob>`. The offline behaviour of that exact command is not measured
here, because a keyless blob signature needs an OIDC identity that this host cannot
obtain.

Loading the images is a separate step. A tar of each index layout loads with
`docker load`, measured above.

---

## Where the evidence is thin

1. No verge-asm release exists. Every measurement uses a stand-in image. The stand-in for
   the verge-asm signing model is cosign v2.5.3, which signs with legacy `.sig` tags and
   with `--recursive`. It matches the #1075 model on both points.
2. No stand-in carries an SBOM attestation on a child manifest. The claim that a child
   attestation is also dropped rests on the source read in section 1 and on the measured
   loss of the child signature. It is not measured directly.
3. The cause of the referrer clobber is my reading of the v3.1.3 source. I found no
   upstream issue that reports it.
4. `cosign verify-blob` offline is not measured.
5. `docker load` from an OCI layout tar is measured on a containerd image store only.
6. The `gh attestation` offline path is measured against `coder/coder`, not against a
   `--signer-workflow` that this project controls.

---

## What this changes on the map

**#1075's `--recursive` model: confirmed.** Nothing here argues against signing the
children. The children still need a signature, because #1074 sends an operator to a
platform digest. What changes is the kit shape, not the signing shape. `cosign save` does
not follow `--recursive`, so the operator saves 6 layouts instead of 1.

**#1075's `--registry-referrers-mode legacy` pin: confirmed, and it now earns more.** A
legacy `.sig` tag produces a layout that `cosign verify --local-image` accepts. A referrer
signature produces a layout that fails with `no signatures associated with the image saved
in <dir>`. That was measured against the cosign v3.1.3 image. Unpinning the mode would
break the air-gap path outright.

**#1072's refusal to ship `trusted_root.json`: confirmed.** The reasoning holds, and the
measurement adds weight. The file is required, and cosign fails without it offline. It is
still Sigstore's file, it still changes without notice, and a copy on a release page still
ages badly. The one change is arithmetic. The operator now runs two initialize steps and
carries two files, because `gh` needs its own `trusted_root.jsonl`.

**#1076's provenance route: confirmed, and the kit gains the `gh` offline path.** The
ruling that provenance is a `gh` path and not a cosign path is correct offline as well as
online. `cosign save` leaves the native attestation as orphan bytes, and
`cosign verify-attestation --local-image` cannot see it. So the air-gap kit needs
`gh attestation download`, `gh attestation trusted-root`, `--bundle` and
`--custom-trusted-root`. cosign alone does not cover every artifact.

**#1074's per-platform SBOM subject: leaves open.** The subject choice is untouched. The
cost is now visible. Four SBOM attestations on four platform digests mean four more `gh`
bundle files and four more verify commands in the guide.

**One new item for the operator-guide rewrite.** The guide must state that a `gh`
verification of an image uses a file, and that the file is the manifest blob inside the
saved layout. That step is not obvious, and no upstream document states it.
