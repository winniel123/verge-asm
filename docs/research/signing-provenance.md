# Survey: Artifact Signing + Build-Provenance Options

Research ticket: [#423](https://github.com/winniel123/verge-asm/issues/423) (part of
map [#420](https://github.com/winniel123/verge-asm/issues/420) "Release pipeline with
artifact signing + SBOM"). This is **investigation only** — the decision belongs to
ticket [#425](https://github.com/winniel123/verge-asm/issues/425).

## Scope reminder

verge-asm ships `web`/`worker`/`prober` (Go, Dockerfiles present) and has **no release
pipeline yet**. Sibling ticket #421 will decide whether releases are:

- **container images** → GHCR (`ghcr.io/winniel123/verge-asm/...`), and/or
- **Go binaries / blobs** → GitHub Releases.

This survey covers **both** artifact types. Repo conventions to respect in any snippet:
all GitHub Actions are **SHA-pinned**, and `GITHUB_TOKEN` uses **least privilege**
(explicit per-job `permissions:`). Keyless (OIDC) approaches are preferred.

---

## 1. Signing

### 1a. Sigstore cosign — keyless (OIDC via GitHub Actions)

**Mechanism.** In keyless mode cosign asks the OIDC provider (the GitHub Actions runner's
`id-token`) for a token, sends it to **Fulcio**, which issues a *short-lived* (~10 min)
X.509 signing certificate binding the ephemeral public key to the OIDC identity. Cosign
signs with the ephemeral private key, then records the signature + certificate in
**Rekor**, the public transparency log. No long-lived private key is ever stored — the
ephemeral key is discarded after signing. (Sigstore docs.)

**Identity that ends up in the certificate.** For a signature produced *inside a workflow*:
- issuer (`--certificate-oidc-issuer`) = `https://token.actions.githubusercontent.com`
- identity (`--certificate-identity`) = the full workflow ref, e.g.
  `https://github.com/winniel123/verge-asm/.github/workflows/release.yml@refs/tags/v1.2.3`

  (The interactive/user login issuer `https://github.com/login/oauth` is a *different*
  thing — that's for a human running cosign at a laptop, not what a CI build uses.)

**Container image (→ GHCR).**
```bash
cosign sign ghcr.io/winniel123/verge-asm/web@sha256:<digest>
```
Signature is stored **in the registry next to the image**, as an OCI referrer /
`.sig` tag (OCI 1.1 referrers API). `cosign tree <image>` shows attached signatures &
attestations. Requires `packages: write` to push the signature, `id-token: write` for OIDC.
Always sign **by digest**, not by mutable tag.

**Release binary / blob (→ GitHub Releases).**
```bash
cosign sign-blob ./web_linux_amd64 --bundle web_linux_amd64.sigstore.json
```
There is no registry to attach to, so cosign emits a **bundle** (`*.sigstore.json`,
signature + cert + Rekor proof). You upload that bundle **alongside the binary as a Release
asset**. Legacy flag form produces separate `--output-signature` / `--output-certificate`
files; the single `--bundle` is the modern, simplest distribution unit.

### 1b. cosign with a managed key

`cosign generate-key-pair` (or a KMS/`--key`), then `cosign sign --key ...`. Downsides for
this repo: a **long-lived private key** to store (KMS or an Actions secret), rotate, and
guard — exactly what keyless avoids. No transparency by default. Only compelling if you
need to sign **outside** an OIDC-capable environment or want an offline air-gapped trust
root. **Not a fit** for a small public repo building on GitHub-hosted runners.

### 1c. Alternatives

- **GitHub artifact attestations** (see §2) — themselves signed via Sigstore keyless; for
  many repos they *replace* a separate `cosign sign` step because an attestation is already
  a signed statement about the artifact.
- **Notary / Notation (CNCF, Notary v2)** — registry-image signing, more common in
  enterprise/Azure ecosystems, heavier trust-policy setup. Overkill here.
- **GPG-signed checksums** (`SHA256SUMS` + detached `.asc`) — traditional, but long-lived
  keys and no transparency; weaker than Sigstore and not keyless.

---

## 2. Build provenance / SLSA

### 2a. GitHub native artifact attestations — `actions/attest-build-provenance`

Generates a signed **SLSA build provenance** predicate (in-toto format) binding the
artifact's digest to *where and how it was built* (repo, workflow, commit, trigger, runner).
Signed with a short-lived **Sigstore** cert (public-good Sigstore for public repos; GitHub's
private Sigstore for private/GHEC). The signed attestation is uploaded to GitHub's
**attestations API** and associated with the repo. Current major tag: **`@v4`** (a thin
wrapper over `actions/attest`).

**Container image:**
```yaml
- uses: actions/attest-build-provenance@<pin-v4-sha>
  with:
    subject-name: ghcr.io/winniel123/verge-asm/web
    subject-digest: sha256:<digest>
    push-to-registry: true   # also stores attestation as an OCI referrer in GHCR
```
**Binary / blob:**
```yaml
- uses: actions/attest-build-provenance@<pin-v4-sha>
  with:
    subject-path: dist/web_linux_amd64   # or a glob for multiple
```

**Permissions** (per-job, least-privilege):
| artifact | id-token | contents | attestations | packages |
|---|---|---|---|---|
| binary | write | read | write | — |
| container image (+push-to-registry) | write | read | write | write |

**SLSA level reached.**
- Provenance existing at all = **SLSA Build L1**.
- On GitHub-hosted runners with the signed attestation = **SLSA Build L2 by default**
  (hosted platform, signed provenance tied to infrastructure).
- **SLSA Build L3** is reachable **only** by moving the build+attest into a **reusable
  workflow** called by tag, which provides the run-to-run isolation and lets a verifier
  prove *which* trusted workflow produced the artifact. A plain inline job does **not**
  reach L3. (GitHub docs: "Using artifact attestations and reusable workflows to achieve
  SLSA v1 Build Level 3".)

### 2b. slsa-framework generators (`slsa-github-generator`, `slsa-verifier`)

A set of reusable workflows (BYOB framework) that build + emit **SLSA3+** provenance for Go,
container images, generic files, etc., verified with the separate **`slsa-verifier`** CLI.
**Important maturity note:** the project is **no longer actively maintained**, and its own
README/GitHub now **recommends GitHub's native artifact attestations instead**. Last release
v2.1.0 (Feb 2025). For a *new* pipeline, choosing this over native attestations means
adopting an unmaintained dependency and a second verify CLI.

### 2c. Overlap with cosign

- An **attestation is itself a signed statement**; you do not need `cosign sign` *and*
  `attest-build-provenance` to get a signature — the attestation already gives signed
  provenance. Adding a bare `cosign sign` on top only adds a plain "this identity signed
  this digest" signature with no provenance payload.
- Native attestations are **verifiable with cosign too** (Sigstore bundles), not only with
  `gh` — so choosing native does not lock consumers into the `gh` CLI.
- Practical takeaway: **one provenance-attestation step covers both "signed" and "has
  provenance"**; a separate cosign signing step is largely redundant for this repo.

---

## 3. Verification UX (consumer commands)

**GitHub native attestation:**
```bash
# container image
gh attestation verify oci://ghcr.io/winniel123/verge-asm/web:v1.2.3 \
  -R winniel123/verge-asm
# binary
gh attestation verify ./web_linux_amd64 -R winniel123/verge-asm
```
Trust is expressed as `-R <owner/repo>` (or `--signer-workflow`); `gh` fetches the
attestation from the API/registry and checks the Sigstore signature + that provenance came
from that repo. Lowest-friction UX; needs the `gh` CLI logged in.

**cosign (image):**
```bash
cosign verify ghcr.io/winniel123/verge-asm/web@sha256:<digest> \
  --certificate-oidc-issuer=https://token.actions.githubusercontent.com \
  --certificate-identity=https://github.com/winniel123/verge-asm/.github/workflows/release.yml@refs/tags/v1.2.3
# or --certificate-identity-regexp='^https://github.com/winniel123/verge-asm/.+' to allow any tag
```

**cosign (blob with bundle):**
```bash
cosign verify-blob ./web_linux_amd64 --bundle web_linux_amd64.sigstore.json \
  --certificate-oidc-issuer=https://token.actions.githubusercontent.com \
  --certificate-identity-regexp='^https://github.com/winniel123/verge-asm/.+'
```
Consumer **must** supply both the issuer and the expected identity — for CI-signed
artifacts these are the `token.actions.githubusercontent.com` issuer and the workflow-ref
identity. cosign can also verify the native GitHub attestation bundle.

**slsa-verifier (only if 2b chosen):**
```bash
slsa-verifier verify-artifact ./web_linux_amd64 \
  --provenance-path web.intoto.jsonl \
  --source-uri github.com/winniel123/verge-asm --source-tag v1.2.3
```

---

## 4. Maturity & fit (small public repo, GitHub-hosted runners)

| dimension | native attestations | cosign keyless | slsa-github-generator |
|---|---|---|---|
| setup cost | **lowest** (1 action step) | low–med (install cosign, sign steps) | med–high (wire reusable wf) |
| maturity | GA, GitHub-maintained, default-recommended | mature (Sigstore GA) | **unmaintained; deprecated in favor of native** |
| perms | `id-token:write`, `attestations:write` (+`packages:write` for image push) | `id-token:write` (+`packages:write` for image sig) | via reusable workflow |
| pinning story | pin `actions/attest-build-provenance@<sha>` like every other action | install cosign via `sigstore/cosign-installer@<sha>`, pin version | pin reusable wf `@vX.Y.Z` (tag, not sha, for verifier compat) |
| consumer tooling | `gh` (or cosign) | cosign | slsa-verifier (extra CLI) |
| public-repo cost | free (public-good Sigstore) | free | free |

Public repos automatically use the **public-good Sigstore** instance, so there is no cost or
extra infra. The one wrinkle for the repo's SHA-pin convention: the SLSA reusable-workflow
path (for L3, both native-L3 and slsa-github-generator) wants a **version tag**, not a SHA,
for verifier identity matching — that's a documented, accepted exception, not a violation of
least-privilege.

---

## RECOMMENDATION MATRIX (for decision ticket #425)

| Option | What it signs | Keyless? | SLSA level | Perms needed | Verify cmd | Maturity | Fit for verge-asm |
|---|---|---|---|---|---|---|---|
| **A. `actions/attest-build-provenance` (inline)** | image digest + binary/blob (signed provenance) | ✅ Sigstore | **L2** default | `id-token:write`, `attestations:write` (+`packages:write` for image) | `gh attestation verify` (or cosign) | GA, GitHub-recommended | **Best default** — one step, both artifact types, no key mgmt |
| **B. Option A moved into a reusable workflow** | same | ✅ | **L3** | same, in the reusable wf | `gh attestation verify --signer-workflow …` | GA | Best if L3 is a stated goal; small extra wiring |
| **C. cosign keyless `sign` / `sign-blob`** | image sig / blob-bundle (signature only, no provenance) | ✅ Sigstore | n/a (signature, not provenance) | `id-token:write` (+`packages:write` for image) | `cosign verify` / `verify-blob` w/ identity+issuer | mature | Redundant if A/B used; only if a bare signature is specifically wanted |
| **D. cosign managed key** | image / blob | ❌ long-lived key | n/a | key secret / KMS | `cosign verify --key` | mature | **Reject** — reintroduces key custody |
| **E. slsa-github-generator + slsa-verifier** | image / binary provenance | ✅ | **L3** | reusable wf | `slsa-verifier` (extra CLI) | **unmaintained; deprecated toward native** | **Reject** for new pipeline |

**Leaning for #425.** Adopt **GitHub native artifact attestations
(`actions/attest-build-provenance@v4`, SHA-pinned)** as the single signing+provenance
mechanism for both GHCR images and Release binaries — it is keyless, GitHub-maintained,
covers both artifact types in one step, and verifies with the ubiquitous `gh` CLI (or
cosign), so a separate cosign signing step is redundant. Ship it **inline first for SLSA
Build L2** (near-zero setup); if the decision ticket wants **SLSA Build L3**, promote the
build+attest into a **tag-referenced reusable workflow** — the only added cost. Avoid managed
keys (D) and the unmaintained slsa-github-generator (E).

---

## Sources (primary)

- Cosign — signing containers: https://docs.sigstore.dev/cosign/signing/signing_with_containers/
- Cosign — other/blob types: https://docs.sigstore.dev/cosign/signing/other_types/
- Cosign — verifying (verify / verify-blob, identity/issuer flags): https://docs.sigstore.dev/cosign/verifying/verify/
- Sigstore CI quickstart (GitHub Actions OIDC): https://docs.sigstore.dev/quickstart/quickstart-ci/
- SLSA v1.0 build levels: https://slsa.dev/spec/v1.0/levels
- GitHub Docs — Using artifact attestations to establish provenance: https://docs.github.com/en/actions/security-guides/using-artifact-attestations-to-establish-provenance-for-builds
- GitHub Docs — Artifact attestations + reusable workflows to reach SLSA v1 Build L3: https://docs.github.com/actions/security-guides/using-artifact-attestations-and-reusable-workflows-to-achieve-slsa-v1-build-level-3
- Action README — actions/attest-build-provenance (v4): https://github.com/actions/attest-build-provenance
- GitHub Blog — Enhance build security and reach SLSA L3 with Artifact Attestations: https://github.blog/enterprise-software/devsecops/enhance-build-security-and-reach-slsa-level-3-with-github-artifact-attestations/
- slsa-framework/slsa-github-generator (maintenance status; recommends native): https://github.com/slsa-framework/slsa-github-generator
- Sigstore blog — cosign verifying GitHub Artifact Attestation bundles: https://blog.sigstore.dev/cosign-verify-bundles/
