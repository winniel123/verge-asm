# SBOM tooling, format, and attachment — research (ticket #424)

Research-only survey for wayfinder map #420 ("Release pipeline with artifact
signing + SBOM"). This captures facts and options; the decision belongs to
ticket #425 (T5). Sibling ticket #421 (T1) decides whether releases ship as
**container images** (→ GHCR) and/or **Go binaries** (→ GitHub Releases), so
every axis below is evaluated for *both* artifact shapes.

verge-asm is a Go codebase shipping `web` / `worker` / `prober` (Dockerfiles
present), with existing supply-chain coverage from **govulncheck** and
**Dependabot**. An SBOM should *complement* those, not duplicate them (see
§4). All GitHub Actions in this repo are SHA-pinned — any adopted action must
be pinned to a commit SHA, not a tag.

---

## 1. Generation tooling

### Anchore syft (the common default)
- CLI + Go library that generates an SBOM "from container images and
  filesystems". Scan sources are selected by scheme prefix:
  `docker:`, `podman:`, `containerd:`, `docker-archive:`, `oci-archive:`,
  `oci-dir:`, `registry:` (image sources) and **`dir:`** (any directory on
  disk) / **`file:`** (single file). Source:
  <https://oss.anchore.com/docs/guides/sbom/scan-targets/>,
  <https://github.com/anchore/syft>
- **Go source / module graph WITHOUT a built image:** yes. `syft dir:.` (or
  `syft .`) recursively catalogs the checked-out tree; the `go.mod` cataloger
  reads the module graph directly — no image or compiled binary required. Syft
  also has a **Go binary cataloger** that reads the build info embedded in a
  compiled Go binary, so it can enrich an SBOM from the shipped artifact too.
- **Container image (OS + language layers):** yes, and this is syft's core
  strength. Pointed at an image it catalogs both OS packages (apk/dpkg/rpm) and
  language deps (Go modules, npm, pip, etc.) across layers. It can read the
  image from a registry (`registry:`) without a local Docker daemon.
- **Output formats:** SPDX (JSON + tag-value), CycloneDX (JSON + XML), syft's
  native JSON, and **GitHub dependency-snapshot JSON** (feeds the dependency
  submission API directly). Can convert between formats. Source:
  <https://github.com/anchore/syft>,
  <https://oss.anchore.com/docs/guides/sbom/>
- Ships as the official **`anchore/sbom-action`** for CI, which wraps syft and
  can additionally submit to GitHub's dependency graph.

### Alternatives

| Tool | Go source | Image (OS+lang) | Formats | Notes |
|------|-----------|-----------------|---------|-------|
| **Trivy** (Aqua) | yes (`fs`/`repo`) | yes (strong OS-layer coverage) | CycloneDX, SPDX, GitHub dep-snapshot | Also a scanner (see §4) — one binary for SBOM **and** vuln scan. Heavier config surface. |
| **cdxgen** (OWASP/CycloneDX) | yes (source + module) | yes | **CycloneDX only** | Deep language/dependency metadata; native to the CycloneDX ecosystem; auto-submits to Dependency-Track. Not SPDX. |
| **GitHub dependency graph export** | manifest-based only | no image layers | GitHub's own SPDX export | Static scan of checked-in `go.mod`; misses transitive/OS/image-layer components. Zero-tooling but least complete. |
| **govulncheck `-format sarif`/json** | n/a — not an SBOM tool | no | n/a | Reachability scanner, not a bill of materials. Listed only to rule out. |

Takeaway: **syft covers both artifact shapes (source dir + image), both major
formats, and emits GitHub's snapshot format** — it is the least-friction single
tool for a repo that may ship both binaries and images. Trivy is the strongest
"one tool does SBOM + scanning" alternative if consolidation is preferred.

---

## 2. Format — SPDX vs CycloneDX

- Both are ISO-recognized, widely supported, and emitted by syft/Trivy. The
  choice is about *consumers*, not capability.
- **What GitHub consumes:** the **dependency submission API accepts both
  SPDX and CycloneDX**; the official actions transpose either into the
  dependency graph, after which **Dependabot alerts sync automatically into the
  Security tab**. GitHub's own dependency-graph *export* is SPDX. Source:
  <https://docs.github.com/en/enterprise-cloud@latest/code-security/supply-chain-security/understanding-your-software-supply-chain/using-the-dependency-submission-api>
- **What `actions/attest-sbom` accepts:** SPDX **or** CycloneDX, JSON-serialized
  only. Source: <https://github.com/actions/attest-sbom>
- **CycloneDX** carries richer *security-oriented* metadata (vulnerabilities,
  VEX, richer license/provenance) and is the native format for the OWASP
  security-tool ecosystem (Dependency-Track, cdxgen). **SPDX** is the
  license-compliance lineage, is what GitHub natively exports, and is an
  ISO/IEC 5962 standard.
- **Verifiers / downstream scanners** (grype, Trivy) happily read **either**.

Takeaway: For this repo the format choice is low-risk in both directions.
**CycloneDX JSON** leans slightly ahead because downstream use here is
security scanning (grype/Trivy) and it is the richer format for that; **SPDX
JSON** is the safer pick if the priority is matching GitHub's native export and
broadest third-party compliance tooling. Either can be attached and submitted;
syft can emit both, so the pipeline is not locked in.

---

## 3. Attachment method

Four mechanisms, each pairing naturally with a different artifact shape:

| Method | Best for | What it produces | Verify with |
|--------|----------|------------------|-------------|
| **`actions/attest-sbom`** (GitHub-native, Sigstore) | **images AND binaries** | Signed in-toto attestation binding subject digest → SBOM; stored in GitHub attestations API. Public repos use public-good Sigstore; keyless. | `gh attestation verify` |
| **cosign attestation** on the image | **container images** in GHCR | SBOM stored as an OCI referrer attached to the image in the registry, keyless via Sigstore/OIDC | `cosign verify-attestation` |
| **GitHub Release asset** | **Go binaries** | SBOM `.json` uploaded next to the binary on the Releases page | manual / any SBOM reader |
| **Dependency submission API** | repo-level (not per-artifact) | Uploads component list into the **dependency graph** → Dependabot alerts in Security tab | GitHub Security tab |

Notes / primary sources:
- **`actions/attest-sbom` is now a thin wrapper being deprecated in favor of
  `actions/attest`** (inputs stay compatible). It signs with short-lived
  Sigstore certs and needs `id-token: write`, `attestations: write`,
  `contents: read`. It attests either a container image (by digest) or a built
  file. Source: <https://github.com/actions/attest-sbom>,
  <https://github.com/marketplace/actions/attest-sbom>
- **cosign** attaches the SBOM as a registry referrer — the SBOM travels *with*
  the image and can be discovered by digest anywhere the image is pulled. This
  is the idiomatic choice when T1 lands on **images → GHCR**.
- A **GitHub Release asset** is the low-ceremony choice for **binaries**; pair
  it with an `actions/attest-sbom` (or `actions/attest-build-provenance`)
  attestation so the asset is also verifiable, not just downloadable.
- The **dependency submission API** is orthogonal to the above — it is not
  per-release artifact attachment; it feeds the *repo's* dependency graph so
  Dependabot sees the fuller (transitive/image-layer) component set. Source:
  <https://docs.github.com/en/enterprise-cloud@latest/code-security/supply-chain-security/understanding-your-software-supply-chain/using-the-dependency-submission-api>

Natural pairings:
- **Images (GHCR):** `actions/attest-sbom` and/or **cosign attestation** on the
  image digest. Both keyless via Sigstore/OIDC; no long-lived keys to manage.
- **Binaries (GitHub Releases):** SBOM as a **Release asset** + an
  **`actions/attest-sbom`** attestation on the binary.
- **Either way:** optionally also **submit to the dependency graph** so
  Dependabot's view is complete.

---

## 4. Vulnerability tie-in (how the SBOM complements govulncheck + Dependabot)

- **grype** (Anchore) and **Trivy** can both scan an SBOM file directly
  (`grype sbom:./sbom.json`), offline, no rebuild. Because the SBOM already
  lists every component, scanning it is fast and reproducible, and lets you
  **re-scan a *published* release later** against newly disclosed CVEs without
  rebuilding — the key operational value of shipping the SBOM. Source:
  <https://github.com/anchore/grype>, <https://anchore.com/grype/>
- **How this complements what's already here:**
  - **govulncheck** does Go-specific *reachability* analysis — it flags only
    vulnerabilities in code paths the binary actually calls. High signal, but
    **Go-only** and blind to OS packages, base-image layers, and non-Go
    components in the container.
  - **Dependabot** watches *declared manifests* (`go.mod`) in the repo and
    opens PRs — pre-release, source-level, and again **Go-module-only**.
  - **SBOM + grype/Trivy** covers the **as-shipped artifact**: OS packages in
    the base image, every transitive/vendored component, across all
    ecosystems, and enables **continuous re-scanning of already-published
    releases**. This is exactly the gap the other two leave.
- So the three layers stack: Dependabot (declared deps, pre-merge) →
  govulncheck (reachable Go vulns, CI) → SBOM+grype (full shipped artifact,
  including image layers, re-scannable post-release).

---

## RECOMMENDATION MATRIX

| Tool | Go source (no image) | Image scan (OS+lang) | Formats emitted | Attachment fit | Maturity |
|------|:---:|:---:|-----------------|----------------|----------|
| **syft** | yes (`dir:` / go.mod) | yes (registry/daemon) | SPDX, CycloneDX, GitHub snapshot | pairs with attest-sbom, cosign, Release asset, dep-graph | very mature; de-facto default; official `anchore/sbom-action` |
| **Trivy** | yes (`fs`/`repo`) | yes (strong OS layers) | CycloneDX, SPDX, GitHub snapshot | same targets; also scans | very mature; SBOM **and** scanner in one binary |
| **cdxgen** | yes | yes | CycloneDX only | any (CycloneDX consumers) | mature in OWASP ecosystem; single-format |
| **GitHub dep-graph export** | manifest only | no | SPDX (GitHub) | dep-graph / Security tab | native but least complete |

| Attachment | Pairs with | Keyless/Sigstore | Verify |
|------------|-----------|:---:|--------|
| `actions/attest-sbom` (→ `actions/attest`) | images **and** binaries | yes | `gh attestation verify` |
| cosign attestation | images (GHCR) | yes | `cosign verify-attestation` |
| GitHub Release asset | binaries | n/a | manual / any reader |
| Dependency submission API | repo dep-graph (not per-artifact) | n/a | Security tab / Dependabot |

### Leaning for decision ticket #425
**syft** is the natural generator — it reads the Go source/module graph *and*
container images, emits every format we might need (including GitHub's snapshot
format), and has an official CI action; **Trivy** is the fallback worth noting
because it folds SBOM generation and vulnerability scanning into one tool.
Prefer **CycloneDX JSON** if the driving use is downstream security scanning
(grype/Trivy), or **SPDX JSON** to align with GitHub's native export — syft
emits both, so this stays reversible. For attachment, match T1's artifact
choice: **`actions/attest-sbom` (migrating to `actions/attest`) plus a cosign
attestation for images in GHCR**, and **an SBOM Release asset plus an
attestation for binaries** — with optional dependency-graph submission so
Dependabot sees the full component set. Remember to **SHA-pin** any adopted
action.
