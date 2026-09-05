# The release pipeline: a tag becomes a signed, attested, multi-arch release

- **Status:** Accepted — spec content for [#1064](https://github.com/winniel123/verge-asm/issues/1064)
- **Map:** [#1064 Release pipeline](https://github.com/winniel123/verge-asm/issues/1064), 25 closed tickets
- **Rulings:** [ADR-0138](../adr/0138-a-release-pins-every-byte-it-builds-so-it-delegates-no-build-step-and-anchors-identity-in-its-own-workflow.md) and [ADR-0139](../adr/0139-the-probers-origin-is-the-image-that-carries-it-and-a-host-bounds-the-binary-rather-than-verifies-it.md), plus an amendment each on [ADR-0001](../adr/0001-stack-and-runtime.md) and [ADR-0124](../adr/0124-a-backup-carries-data-and-no-secret-and-updating-is-guided-not-self-applied.md). See §18.

This is a separate file from the ADRs that rule it, on
[`packaging-and-configuration.md`](./packaging-and-configuration.md)'s precedent. **The tables
below will be revised and an ADR is a decision that will not.** The asset list in §9 and the
open-point list in §20 both move as the pipeline runs for the first time.

**Nothing here is a workflow file.** The map's *plan, do not build* rule holds. This specifies
`.github/workflows/release.yml` and `.github/workflows/release-scan.yml`, and an implementation
session writes them. Every action SHA below was measured on 2026-09-01 or 2026-09-03.
**Re-measure each one at implementation.**

Each ruling cites the ticket that made it. A reader who wants the argument reads the ticket. A
reader who wants the rule reads this file.

---

## 1. What a release is

### 1.1 The tag

The git tag is `vX.Y.Z`. It keeps a leading `v`. It carries no suffix and no build metadata.
The exact form is `^v[0-9]+\.[0-9]+\.[0-9]+$` ([#1069](https://github.com/winniel123/verge-asm/issues/1069),
[#1078](https://github.com/winniel123/verge-asm/issues/1078)).

The first tag is `v0.1.0`. `CHANGELOG.md` already fixes the line at `0.y.z`.

The format is a hard contract with `internal/release.isNewer`, which parses a dotted numeric
version. A change to the format is a change to the update check.

**A human pushes the tag.** No bot opens a release pull request, and no release automation holds
a key (#1078). The command is `git tag vX.Y.Z <sha> && git push origin vX.Y.Z`.

**`gh release create` is forbidden as the tag mechanism.** It creates the tag and the Release
together. §10 requires the Release last, and a failed run must leave no Release at all.

### 1.2 The version string

`VERGE_VERSION` carries the bare number `X.Y.Z`. The stamp drops the `v` (#1069).

The release workflow derives it as `${GITHUB_REF_NAME#v}`.

Every human-facing site shows the bare number. The sign-in footer, the Settings -> Instance card
and the CT `User-Agent` all read `0.1.0`. RFC 9110 product tokens carry no `v`, which is why the
stamp drops it. The GitHub Release page is the one site that keeps the `v`.

**The feed field is `tag_name`.** `HTTPFetcher.Latest` decodes it into `feedPayload` and rejects
an empty one (`internal/release/fetcher.go`). The `name` field is never read.

**`HTTPFetcher.Latest` trims a leading `v` before it returns `Feed.Version`.** The release cache
then stores one format. Without that trim the Instance card renders `0.1.0` beside `v0.2.0`.

The tolerance in `parseVersion` stays. It strips a leading `v` in
`internal/release/release.go`, so a fork that repoints `VERGE_RELEASE_FEED_URL` may send
either format.

**The pre-1.0 rule appears on the Instance card, beside the version.** A minor bump may carry a
breaking change while the major is `0`.

### 1.3 The project cuts no pre-release tag

**A refusal with reasons, not an omission** ([#1129](https://github.com/winniel123/verge-asm/issues/1129)).
No `-rc`, no `-beta`, no `-alpha`. The update check reports stable releases only.

Four grounds:

1. **The `0.y.z` line already carries the signal.** `README.md` states an alpha notice and
   `CHANGELOG.md` states the pre-1.0 rule. A release candidate for software already labelled
   alpha is a second instability marker over the first.
2. **A pre-release train buys a soak period from an audience that does not exist.** The repository
   has one collaborator and no installed base.
3. **A spent tag is permanent.** §4.3 and §17 make every `vX.Y.Z` tag immutable. A candidate train
   multiplies entries in a namespace nothing can ever clean.
4. **The update check cannot represent one.** See §1.4.

Three layers refuse a pre-release tag, and the trigger is not one of them:

| Layer | Where it lives | What it does |
| --- | --- | --- |
| `tag_name_pattern` on the tag ruleset | repository settings (§17) | refuses the **push**. Nothing is created. |
| the trigger `push: tags: ["v*.*.*"]` | `release.yml` | **does not refuse**. See below. |
| guard 1, `^v[0-9]+\.[0-9]+\.[0-9]+$` | `release.yml`, the `guard` job | refuses inside the run. The exact statement. |

**The trigger stays `v*.*.*` and is not tightened.** GitHub ref filters are globs, and `*` matches
any character except `/`. So the third `*` swallows `0-rc1` and `v0.1.0-rc1` matches the glob.
That reading is derived from GitHub's documented glob semantics and is not measured against a live
push.

A negation such as `!v*-*` is refused on three grounds. A glob is an approximation, and a later
reader misreads a tightened one as exact. **The guard regex must remain the only exact statement
of the tag format.** A refused `guard` job leaves a red run that names the reason, and a
filtered-out tag leaves silence. Reaching this path at all needs the tag ruleset to be missing,
which is an alarming state that earns a loud refusal.

**A fourth site meets a pre-release tag and does not refuse it**
([ADR-0155](../adr/0155-the-docs-site-does-not-enforce-the-tag-policy-so-a-prerelease-tag-is-browsable-and-never-becomes-latest-or-current.md),
2026-09-05). The layer table above lists the sites that refuse a push. It is not a list of every
site that meets such a tag. The docs site reads `git tag -l "v*"` at build time, after the push.
It publishes a pre-release tag as its own browsable `/<tag>/*` version tree, and it never lets that
tag become `latest` or carry the `current` badge. This section is unchanged in its own subject: the
project still cuts no pre-release tag, and ADR-0155 grants no exception to that rule.

### 1.4 A fork's repointed feed gets no defence

`internal/release/fetcher.go` sets `DefaultFeedURL` to this repository's `/releases/latest`
endpoint. That endpoint returns the most recent non-prerelease, non-draft release by definition.
**So the default feed cannot serve a pre-release even if one existed.**

`isNewer("0.2.0-rc1", "0.1.0")` returns true today, and the Instance card renders the cached
string as given. **This stays. No code defends against it.** A fork that serves pre-releases on
its own feed has chosen to, and refusing a suffixed feed version would make verge silently ignore
a fork's own shipped release.

`parseVersion` and `isNewer` gain no pre-release ordering. The numeric-core comparison in
`isNewer` and `parseVersion` (`internal/release/release.go`) is unchanged. `TestIsNewer`
(`internal/release/release_test.go`) already pins
`{"v3.18.0", "v3.18.0-rc1", false}` as an asserted contract.

**One test row is added**, so the other direction stops being an untested side effect:

```go
{"0.2.0-rc1", "0.1.0", true}, // a fork's feed may serve one; core wins (#1129)
```

**The consequence is stated out loud: a source build is never told that a release shipped.** A
hand-built binary carries no version the check can reason about, and `parseVersion` fails,
`isNewer` returns false, and `release_state` stays `current`. That is the no-false-alarm rule
working as designed. A detector that marks an unparseable version as an unofficial build is
**rejected**, because it teaches the UI a version grammar needed nowhere else.

---

## 2. The images

### 2.1 The two image references

```
ghcr.io/winniel123/verge-asm/web
ghcr.io/winniel123/verge-asm/worker
```

The nested three-segment form ([#1077](https://github.com/winniel123/verge-asm/issues/1077)).
`docs/guides/verifying-releases.md` and `docs/research/signing-provenance.md` already
assert it, and the flat `verge-asm-web` form would contradict two committed documents.

Each path is a separate GHCR package with its own settings page. §17 records the two manual
visibility flips that follow.

### 2.2 The tag set

Two tags per image per release. Both point at the multi-arch **index** digest.

| Tag | Property |
| --- | --- |
| `vX.Y.Z` | immutable. Spent the moment it is pushed, and never reused. |
| `latest` | a floating pointer. It moves on every release, and it may move backwards (§14). |

**The never-reused rule binds `vX.Y.Z` only.** `latest` is a pointer, and moving it is its purpose.
The two rules do not conflict, and this paragraph exists so a later reader does not read a
contradiction.

**The minor alias `vX.Y` is rejected.** Pre-1.0 the minor is the breaking unit, so `v0.1` is the
only honest alias, and it dies silently the day `v0.2.0` ships. An operator following `v0.1` then
stops receiving updates while the Instance card keeps reporting `0.2.0` as newer. Their
`docker compose pull` does nothing, forever, and nothing tells them why. Revisit at 1.0.

**A version tag alone is rejected.** §13 freezes a four-line host block whose second line is
`docker compose pull`. That command upgrades only when the tag the compose file names moves.

**Tags do not multiply the signature count.** §7 signs digests. Every tag on an image points at
the one index digest.

### 2.3 The build

**Hand-rolled in `release.yml`. The release calls no Docker reusable workflow, not even for the
build** ([#1080](https://github.com/winniel123/verge-asm/issues/1080)).

One job, one `ubuntu-latest` amd64 runner. `docker/bake-action` over a committed
`docker-bake.hcl`. Two targets, `web` and `worker`.

| Setting | Value |
| --- | --- |
| `platforms` | `["linux/amd64", "linux/arm64"]` |
| `provenance` | `false` |
| `sbom` | `false` |
| output | `type=image,push-by-digest=true,name-canonical=true,push=true` |
| tags | cleared |
| cache backend | none |

The `Dockerfile` already cross-compiles from `$BUILDPLATFORM` and runs no command on the target,
so one amd64 runner covers both platforms with no QEMU
([#1068](https://github.com/winniel123/verge-asm/issues/1068)).

**CI builds `docker-bake.hcl` on every pull request with `push=false`.** A release-only file would
first execute during the release it must produce, and the §4.3 guards do not catch a malformed
bake target. The pull-request run needs no OIDC and no registry login.

**`docker/github-builder` is rejected in every mode**, on three independent grounds:

1. **It signs under Docker's identity.** Its own verify step pins
   `^https://github.com/docker/github-builder/.github/workflows/bake.yml.*$`. Fulcio writes
   `job_workflow_ref` as the certificate subject, so a signature made inside a reusable workflow
   carries that workflow's ref.
2. **It applies the tags inside its own call.** §5 requires the Trivy gate between the digest push
   and the tag. A gate cannot sit inside a reusable workflow call.
3. **It forces a second provenance.** On a public repository it always injects
   `*.attest=type=provenance,mode=max,version=v1`. Every image would carry two provenance
   predicates under two identities, and the `--recursive` signature count would rise from six to
   ten.

A SHA pin does not answer the objection. **A SHA pin freezes the caller line only.** That workflow
runs `npm install` with no integrity hash, and pins three images by tag rather than by digest.
Adopting it would replace this project's pin-every-byte rule with a trust-Docker rule for the
build half. Revisit only if Docker ships a mode that writes no tag and forces no provenance.

**`docker/metadata-action` is rejected.** §2.2 fixes the tag set at two literal strings and §1.1
fixes that a human pushes the tag, so `${GITHUB_REF_NAME}` already is the version.
`org.opencontainers.image.source` moves into the `Dockerfile` as a `LABEL` instead. That label
must be on the image **before** the first publish, or GHCR never links the package and
`GITHUB_TOKEN` loses push permission. **The cost is stated: the workflow hand-writes the
`revision`, `version` and `created` annotations, and this project owns their correctness.**

**Any build cache backend is rejected.** The `Dockerfile` runs `COPY . .` above every `go build`,
so every release commit busts every `go build` layer. `type=gha` would spend the repository's 10 GB Actions cache budget
for a hit only on `go mod download`.

**BuildKit's default provenance is rejected.** The default is on at `mode=min`. Silence would give
each image a second provenance predicate. `provenance = false` is explicit in `docker-bake.hcl`.

**A per-platform runner matrix is rejected.** It rebuilds the shared `builder` stage per runner and
buys nothing for a cross-compiling `Dockerfile`.

### 2.4 The repository `docker-compose.yml`

`web` and `worker` keep the `build:` block they have today. Each gains a literal `image:` key:

```yaml
image: ghcr.io/winniel123/verge-asm/web:latest
image: ghcr.io/winniel123/verge-asm/worker:latest
```

A literal tag, not a variable. One file therefore serves the contributor and the operator, and the
§13 card's `docker compose pull` upgrades a default install.

**No `VERGE_IMAGE_TAG` knob.** The knob exists to let an operator pin, and §9 already built the
pinning mechanism as a generated, signed, digest-pinned Release asset. A tag variable is strictly
weaker, because `:tag` and `@sha256:` are different syntax and one variable cannot carry both.

**No full-reference variable either.** `${VERGE_WEB_IMAGE:-...}` does hold either form. It also
makes the registry operator-editable, which the §15 guide must then qualify.

**How a cloned operator pins a digest: they edit the file.** `docker-compose.yml` already pins
postgres by literal manifest-list digest, with a comment naming the reason. That is the house
precedent, and it is one edit per service.

**Why one file and not two.** §9 rules that the Release asset is the repository file with this
release's digests substituted. With no `image:` key there is nothing to substitute, and the
generator would have to synthesize keys.

**`docker-compose.external-db.yml` needs nothing.** It carries no image reference.

**The `compose` CI job still builds from source.** The `compose` job in `.github/workflows/ci.yml` runs
`docker compose build` before `docker compose up -d`, and the build tags the result as the
`image:` reference, so `up` finds the image locally. That step order is required. Without it the
job would boot a published image and test nothing on the branch. The same holds for
`compose-external-db-tls` in `ci.yml`.

---

## 3. The build stamp

**The stamp wins when it is present. The env fills the gap when it is not**
([#1070](https://github.com/winniel123/verge-asm/issues/1070)).

A binary carrying a non-empty build stamp ignores `VERGE_VERSION`. A binary with an empty stamp
reads `VERGE_VERSION`, and defaults to `dev`.

| Item | Ruling |
| --- | --- |
| Package | one new `internal/buildinfo`, holding `var version string` and `Version()` |
| Stamp mechanism | `-ldflags "-X <module>/internal/buildinfo.version=$VERGE_VERSION"` |
| Dockerfile | `ARG VERGE_VERSION=""` in the `builder` stage, above the `RUN go build` |
| Scope | the `web` and `worker` builds only. The prober is not stamped. |
| Call sites | `server.buildVersion` (`cmd/web/auth.go`), and `cmd/worker/main.go`'s `ctVersion` and its `release.NewChecker` version argument |
| Release workflow | passes the build arg as `${GITHUB_REF_NAME#v}` |

**The trap in the empty default.** `ARG VERGE_VERSION=dev` would make the stamp non-empty on every
build, and the fallback would then be unreachable on every image, forever. The precedence rule
would collapse into "the stamp always wins" without anyone deciding that. **The default must be
the empty string.**

A builder-stage `ARG` does not become a runtime env in the `web` and `worker` stages. The runtime
`VERGE_VERSION` stays the operator's.

**`buildinfo.Version()` is pure and logs nothing.** `server.buildVersion` (`cmd/web/auth.go`) runs
on every SignIn render, so a log line inside the accessor would print on every page load. A separate one-shot boot
check in each of `cmd/web` and `cmd/worker` logs one line when a stamp is present and
`VERGE_VERSION` is also set. **The process never refuses to start.**

**Why the precedence runs this way.** An env that outranks the stamp lets an operator relabel any
released image. A faked-low version makes the update check report `newer` forever, and a faked-high
version silences it permanently. The ruling removes that on a released image, and keeps it on an
unstamped one, where the operator built the binary.

**The release workflow asserts that the built image reports the tag.** After this ruling nothing
else proves the stamp landed.

**`docker-compose.yml` does not change, and the `compose` CI job does not change.** A
compose-built stack stays unstamped and reads `dev`.

**Two fixture edits follow.** `devFixtureVersion` (`cmd/web/devfixtures.go`) is `"v0.9.2"`,
and §1.2 says every human-facing site shows the bare number. It becomes `0.9.2`, along with three
`version` values in `design-system/fixtures/fixtures.json`. The SignIn and Setup pixel goldens are
then re-blessed. `TestSigninFixtureMatchesPackage` (`cmd/web/devfixtures_test.go`) gates the
agreement.

`docs/guides/running.md`'s `VERGE_VERSION` row states that the env applies to an unstamped build only, and that a
released image ignores it.

---

## 4. `release.yml`: the shape

### 4.1 Trigger, concurrency and the fork guard

**Trigger: `on: push: tags: ["v*.*.*"]`. Nothing else**
([#1079](https://github.com/winniel123/verge-asm/issues/1079)).

`release: [published]` is impossible, because §10 makes the workflow create the Release last.
**`workflow_dispatch` is refused**, because it buys only a re-run and §2.2's spent-tag rule means a
re-run is never correct. Cut a new tag instead.

The house rule that `id-token: write` must never reach a `pull_request` run is honoured **by
absence**. `release.yml` has no `pull_request` trigger.

**Concurrency: `group: release`, a global singleton, with `cancel-in-progress: false`.** Two
reasons. A cancellation between the digest push and the sign step would leave unsigned digests in
GHCR with no cleanup path. And a per-ref group would give two tags pushed together two separate
groups, which then race on the floating `latest` tag. The precedent is the `docs-site-pages` concurrency group on `docs-site.yml`'s `deploy` job.

**Fork guard: `if: github.repository == 'winniel123/verge-asm'` on the `guard` job only.** The five
jobs chain linearly through `needs:`, so a skipped `guard` skips all five. Without it, a fork that
pushes a `v*` tag runs the pipeline under **the fork's own Fulcio identity**. The GHCR push fails,
because §2.1 hardcodes the registry path, but `cosign sign-blob` over a locally built
`SHA256SUMS` succeeds first and mints a genuine Sigstore bundle bound to a stranger.

**A release tag fires two workflows.** `docs-site.yml` already triggers on `push: tags: ["v*"]`
and its `deploy` job already holds `id-token: write`. That trigger stays. The republish is wanted,
and the two workflows write different outputs to different places. There is no ordering between
them. Silently deleting a working trigger to tidy the diagram is worse than a documented overlap.

### 4.2 The five jobs and their permissions

| Job | Purpose | Permissions |
| --- | --- | --- |
| `guard` | the four §4.3 checks, the migration diff, the pin check | `contents: read` |
| `build` | bake, push by digest | `contents: read`, `packages: write` |
| `scan` | the §5 Trivy gate, SARIF, the SBOM documents, the stdlib assert | `contents: read`, `packages: read`, `security-events: write` |
| `publish` | tag, cosign sign, attest, sign-blob | `contents: read`, `packages: write`, `id-token: write`, `attestations: write`, `artifact-metadata: write` |
| `release` | create the GitHub Release | `contents: write` |

**The rule the split exists for: `contents: write` and `id-token: write` never coexist in one
job.** Its cost is one artifact hop. `cosign sign-blob` needs `id-token: write`, so it runs in
`publish` and hands `SHA256SUMS.sigstore.json` to `release` as a workflow artifact.

`artifact-metadata: write` is required rather than optional
([#1155](https://github.com/winniel123/verge-asm/issues/1155)). `create-storage-record` defaults
to `true`, and a record is emitted whenever `push-to-registry: true` is set. All six §8 steps set
it. **`create-storage-record: false` was rejected**, because it saves one narrow scope and loses a
GitHub-side discovery record for six attestations.

`security-events: write` sits at job level, never at workflow level. `ci.yml`'s `gosec` job already sets
that shape. The repository default workflow permission is already `read`.

**No machine check enforces the `id-token` rule.** A grep for `id-token: write` beside
`pull_request` raises a false positive on `docs-site.yml`, which is correct code. A true check
must model job-level reachability over the trigger graph, and that cost exceeds its value at three
call sites. Revisit if a fourth holder appears. The rule gets a comment header in `release.yml`
mirroring `scorecard.yml`'s header comment, and one line in `CONTRIBUTING.md`.

**`GITHUB_TOKEN` only. `release.yml` adds zero secrets.** It covers the GHCR push, the SARIF
upload, the attestations and the Release creation. The one act it cannot perform is §17's GHCR
visibility flip, which is manual and needs no token. `persist-credentials: false` on every
checkout, because no step needs a git credential on disk.

`contents: write` on the `release` job cannot reach `main`. The `main protection` ruleset has an
empty bypass list, so it binds `GITHUB_TOKEN` too.

### 4.3 The four guards

All guards run **before any build step**. Each failure exits non-zero, and §4.4's spent-tag rule
then applies (#1078).

**Guard 1 — tag format, three checks.** The tag matches `^v[0-9]+\.[0-9]+\.[0-9]+$`. The tag sorts
strictly above every existing tag. The major is `0` while the README alpha notice stands. This
regex is the exact statement of the tag format, per §1.3.

**Guard 2 — agreement, one check that closes three failure modes.** The tagged commit must be
**exactly** the commit that last modified `CHANGELOG.md`, and that file's newest section heading
must match the tag. This refuses a version mismatch between the generated file and the pushed tag,
a tag on a commit older than the prep merge, and a merge landing in the gap between the prep merge
and the tag push.

A merge that lands in that gap moves the tip past the tagged commit. The check still passes and
the release is still correct. It simply does not contain that merge, which is the honest outcome.

**Guard 3 — ancestry.** `git merge-base --is-ancestor $GITHUB_SHA origin/main`. A tag may be
pushed from any branch today, so the guard cannot live in repository settings. Guard 2 already
implies this one, because `CHANGELOG.md`'s last-modifying commit is on `main`. Both are stated
because they refuse for different reasons.

Refusing is correct rather than marking the Release a pre-release. A release built from an
unmerged commit ships code that never passed the 7 required checks, and §7 would sign it with this
repository's Fulcio identity.

**Guard 4 — empty range.** `git cliff --latest` over a range with no commits produces no body. The
workflow refuses the tag.

**The guard count stays at four.** Two further computations live in `guard` and neither is a fifth
refusal:

- **The migration diff** (§14). `guard` diffs `db/migrations/` between the new tag and the previous
  tag on `origin/main`, and hands the result to `release` as a job output. It always succeeds.
- **The Go pin check** (§12). `scripts/check-go-pins.sh` runs here, and a fetch failure is fatal in
  this job.

### 4.4 The step order

The order is load-bearing. §5 requires the gate between the digest push and the tag, so a blocked
release leaves nothing public.

1. The four guards refuse before any build (§4.3).
2. `actions/checkout` at the tag.
3. `docker/setup-buildx-action` and `docker/login-action`, both SHA-pinned.
4. `docker/bake-action` over `docker-bake.hcl`. Push **by digest**, no tags (§2.3).
5. Trivy scans each **platform manifest digest**. The OS half blocks, the Go half annotates (§5).
6. Trivy writes the eight SBOM documents from those same four scan runs (§6).
7. `docker buildx imagetools create` applies `vX.Y.Z` and `latest` (§2.2).
8. `cosign sign --yes --recursive --registry-referrers-mode legacy` on each index. Six signatures (§7).
9. `actions/attest` on the two index digests. Provenance (§8).
10. `actions/attest` on the four platform digests. SPDX SBOM (§8).
11. Build `SHA256SUMS` and run `cosign sign-blob --bundle` (§7, §9).
12. Create the GitHub Release **last** (§10).

**The Trivy gate binds the platform manifest digests, not the index digest.** `imagetools create`
copies child descriptors by reference, so a platform digest survives the tagging step whatever
happens to the index.

**A blocked release spends its version number.** The tag stays. Nobody deletes it and nobody
re-pushes it. The repair is a fix on `main` followed by the next tag. A dead tag is invisible to
the update check, because `isNewer` reads the feed's `tag_name` and never reads `git tag`. Step 4
keeps the images unreachable by name, so no image leaks either. **Reusing a tag is the only option
that can cause harm**, because anyone who already fetched the tag would find different bytes under
the same name.

---

## 5. Scanning

**Trivy gates the release, but only over the finding class no other check watches**
([#1073](https://github.com/winniel123/verge-asm/issues/1073)).

### 5.1 The gate covers the OS half only

A finding in an **OS package** blocks the release. A finding in a **Go binary** annotates it and
never blocks.

Trivy's Go findings come from `govulndb`, the same database `govulncheck` reads, minus the
reachability filter. CI already blocks every pull request on `govulncheck` for any reachable
advisory. **So a Trivy Go gate can only fail a release on an advisory the house standard already
judged and cleared.** Measured on 2026-09-01 that is exactly one advisory, `GO-2026-5932`,
severity `UNKNOWN`, no fixed version, uncalled
([#1067](https://github.com/winniel123/verge-asm/issues/1067)).

The OS half is different. It is the only watch on `gcr.io/distroless/static-debian12`, and
`govulncheck` has no view of dpkg. The surface is five packages: `base-files`, `ca-certificates`,
`media-types`, `netbase` and `tzdata`. It carries zero advisories today.

### 5.2 The threshold

The gate fires on an OS finding at `HIGH` or `CRITICAL` **that has a fixed version**.
`--ignore-unfixed` makes the gate actionable, because a fixed OS advisory has exactly one repair
and an unfixed one offers none. `HIGH` matches the house standard `gosec` already sets.

Unfixed findings still reach the SARIF and the Release asset. They stay visible without stopping a
release.

### 5.3 Eight invocations

Two Trivy runs per image per architecture:

| Run | Scope | Exit | Destination |
| --- | --- | --- | --- |
| gate | OS packages, `HIGH,CRITICAL`, `--ignore-unfixed` | `--exit-code 1` | the gate |
| report | everything | `--exit-code 0` | SARIF, a Release asset, and the §6 SBOM documents |

Two images times two architectures times two runs is eight invocations. **Each targets a registry
reference**, because a multi-arch tarball fails with `tarball must contain only a single image`.
Trivy scans one platform at a time and never scans a manifest list as a unit.

**Every SARIF upload needs a distinct `category:`.** Eight uploads collide in code scanning without
one, and each would overwrite the last. The scheme is `trivy-<image>-<arch>`.

### 5.4 Two pins, not one

`aquasecurity/trivy-action` SHA-pinned, with its `version:` input pinned to an exact Trivy release.
**Both pins are required.** The action SHA pins the wrapper, and the `version:` input pins the
scanner binary the wrapper downloads. The scanner is the pin that matters.

A hand-rolled `docker run` step is rejected. It rebuilds the database cache and the SARIF template
and buys nothing.

### 5.5 An absent scan is not a pass

Trivy fetches its database from a registry at scan time, and that fetch can fail on a rate limit or
an outage. The step retries against the documented mirror, then **fails the release**. A gate that
opens when its data source is unreachable is not a gate. Failing costs one re-run, and §4.4 already
establishes that the tag survives a failed run.

### 5.6 The waiver

A waiver is an entry in `.trivyignore.yaml` at the repository root. Every entry carries an
`expired_at` date and a stated reason, and it lands through a normal pull request with green CI.

Anyone who can merge to `main` may grant one. That is the same authority that can merge the fix, so
no new permission tier appears.

**A `workflow_dispatch` skip input is rejected.** It disables the gate at the exact moment the gate
has something to say, and it leaves no record in the tree.

---

## 6. The SBOM

**A release carries eight SBOM documents. Trivy writes all of them**
([#1074](https://github.com/winniel123/verge-asm/issues/1074)).

### 6.1 Format and scope

Both **SPDX 2.3 JSON** and **CycloneDX 1.6 JSON**. SPDX is ISO/IEC 5962 and procurement text names
it. CycloneDX is ECMA-424 and Dependency-Track needs it. Trivy writes both from one invocation, so
"pick one" buys nothing. This is a self-hosted product and the operator's scanner is unknown.

**Per image, per architecture. Four documents per format.**

| Image | Platform | SPDX asset | CycloneDX asset |
| --- | --- | --- | --- |
| `web` | `linux/amd64` | `web-linux-amd64.spdx.json` | `web-linux-amd64.cdx.json` |
| `web` | `linux/arm64` | `web-linux-arm64.spdx.json` | `web-linux-arm64.cdx.json` |
| `worker` | `linux/amd64` | `worker-linux-amd64.spdx.json` | `worker-linux-amd64.cdx.json` |
| `worker` | `linux/arm64` | `worker-linux-arm64.spdx.json` | `worker-linux-arm64.cdx.json` |

No single document can describe an index, because the package set differs per architecture. Every
reader gets all four. **An arm64 operator who receives an amd64 SBOM holds a wrong SBOM, not a
partial one.**

### 6.2 The producer

**Trivy. The release adds no new tool.** §5.3 already places four report runs, and those runs gain
two `--format` outputs each, `spdx-json` and `cyclonedx`. Trivy also buys one thing Syft cannot:
the SBOM and the vulnerability report describe the same package set, because one tool read one
image once.

**Syft is rejected.** It is correct, and it is the reference generator. It repeats work a Trivy run
already does and reads each image a second time.

**BuildKit `sbom: true` is rejected.** It emits SPDX only, its copy is unsigned, and it never
reaches the GitHub attestations API. **The cost is stated:** this knowingly gives up the one thing
BuildKit does better, which is composing per-platform attestation manifests into the index with no
bookkeeping.

### 6.3 Attachment and subject

Three attachment points, and one step populates two of them:

1. **In-registry attestation** — for the operator who pulls the image and holds registry credentials.
2. **GitHub attestations API** — for `gh attestation verify` with no registry login.
3. **GitHub Release asset** — for the auditor with no registry access, no cosign and no `gh`.

Points 1 and 2 carry the **SPDX** document only, as four attestations. `push-to-registry: true`
writes both copies from one step. Point 3 carries **all eight** documents.

Shipping SPDX to both the registry and the Release page is deliberate. It lets a Release-page
reader hash the downloaded SPDX against the attested SPDX. A clean split of one format per reader
gives no reader a cross-check.

**Each SBOM attestation binds to the per-platform manifest digest, never to the index digest.** An
SBOM describes one platform, and binding four different documents to one index digest states a
falsehood about three of them.

**This differs from the provenance on purpose.** §8 binds provenance to the index digest, because
one build produces one index. A release therefore carries one provenance attestation per image on
the index, and two SBOM attestations per image on the platform manifests. **The guide must never
imply one digest serves both.**

The operator cost is two extra commands, given in §15.2.

---

## 7. Signing

**The release signs keyless. One Sigstore trust anchor covers the two images, the six image
signatures, the provenance attestations, the SBOM attestations and `SHA256SUMS`**
([#1075](https://github.com/winniel123/verge-asm/issues/1075)).

### 7.1 The limb that decided it

Two limbs, and the second is the stronger.

1. **The air-gap limb is answered, not decisive.** Keyless verifies with no network call, using
   `cosign verify --local-image --trusted-root <file>`. Cosign v3 deleted `--offline`. So ADR-0124
   does not rule keyless out.
2. **The one-anchor limb is decisive.** §8 picked a keyless-only attestation action. §6 requires one
   trust anchor for the whole release. A key-based cosign signature would give the release two
   anchors, so the key route cannot satisfy §6 at all.

ADR-0053 reaches the same answer on its own. Keyless adds no standing secret, and cosign documents
no rotation and no revocation for a self-managed key.

**The cost is a permanent public record.** Every keyless signature writes the repository URL, the
workflow path, the commit SHA, the tag, the runner class and a run link to Rekor.

### 7.2 What carries a signature

`cosign sign` runs with `--recursive`. Without it, only the index digest carries a signature, and
§6 already sends operators to platform digests.

| Artifact | Subject | Count | Written by |
| --- | --- | --- | --- |
| cosign signature | `web` and `worker` index digest | 2 | `cosign sign --recursive` |
| cosign signature | each platform manifest digest | 4 | `cosign sign --recursive` |
| provenance attestation | `web` and `worker` index digest | 2 | `actions/attest` (§8) |
| SPDX SBOM attestation | each platform manifest digest | 4 | `actions/attest` (§8) |
| cosign blob signature | `SHA256SUMS` | 1 | `cosign sign-blob` |

**Eleven subjects in total.**

### 7.3 Why two tools, and why that is still one anchor

The release runs **both** `cosign sign` and `actions/attest`. Keyless puts both on the same OIDC
identity, the same Fulcio CA and the same Rekor log. **Two tools is not two anchors.**

Each tool earns its place. The **attestations** carry a predicate, and a bare signature states
none. The **cosign signature** verifies with one command, with no `gh` and with no downloaded
bundle file, and the `gh` offline path needs one bundle file per subject.

A cosign-only release is rejected, because it throws away the predicates §6 and §8 chose.

### 7.4 The steps

```yaml
- uses: sigstore/cosign-installer@<commit-sha>
  with:
    cosign-release: 'v3.1.3'

- name: Sign the image and its per-platform children
  run: |
    cosign sign --yes --recursive \
      --registry-referrers-mode legacy \
      "${IMAGE}@${INDEX_DIGEST}"

- name: Sign the checksum list
  run: |
    cosign sign-blob --yes \
      --bundle SHA256SUMS.sigstore.json \
      SHA256SUMS
```

`sign-blob` has one output form in cosign v3. `--output-signature` and `--output-certificate` are
gone, so `--bundle` is the tool's choice and not this project's.

**`SHA256SUMS.sigstore.json` is a Release asset.** It is not listed inside `SHA256SUMS`, because it
signs that file.

### 7.5 The signing steps must live in `release.yml`

Fulcio writes `job_workflow_ref` as the certificate subject, and **a reusable workflow changes that
string**.

**Ruling: the sign and attest steps run only in a job defined in
`.github/workflows/release.yml`.** Without this, the identity an operator pins is unknowable, and
it would move again whenever a reusable workflow version moved.

The resulting identity is:

```
https://github.com/winniel123/verge-asm/.github/workflows/release.yml@refs/tags/v0.1.0
```

### 7.6 Versions, and two roles

The pipeline pins `sigstore/cosign-installer` to a commit SHA, and pins `cosign-release: "v3.1.3"`
exactly.

The two roles are stated separately, because one sentence cannot serve both:

- **The operator verifying a release needs a floor.** cosign v3.1.3 or later.
- **The builder needs an exact pin**, like every other byte in this pipeline.

The true floor is **v3.0.5**, where `--local-image` first worked with the new bundle format. That
fact belongs here and not in the operator prerequisite line.

**Every cosign v2 instruction is wrong for the offline path.** The guide must say so. The cosign
air-gap README section is self-declared out of date and still prints `--offline=true`.

### 7.7 `--registry-referrers-mode legacy` is pinned

GHCR serves no OCI referrers API today, measured on 2026-09-03, so cosign uses the
`sha256-<digest>.sig` tag scheme. The guide and the §15.3 kit both describe that layout.

**Pin the mode.** If GHCR ships a referrers API later, an unpinned cosign would silently move the
signature off the `.sig` tag and break a published guide mid-series. A later ADR can flip it on
purpose.

The pin earns more than it first appeared. A legacy `.sig` tag produces a layout that
`cosign verify --local-image` accepts. A referrer signature produces a layout that fails with
`no signatures associated with the image saved in <dir>`. **Unpinning the mode would break the
air-gap path outright** ([#1148](https://github.com/winniel123/verge-asm/issues/1148)).

### 7.8 What signing rules out

- **`trusted_root.json` is not a Release asset.** The file belongs to Sigstore and it changes
  without notice. The decisive reason is circularity: `SHA256SUMS` is signed by a certificate that
  chains to that same root, so a checksum over the root proves nothing to a reader who does not
  already hold it. The operator runs `cosign initialize` and carries the file in.
- **`SHA256SUMS` gets no provenance attestation.** The keyless certificate already records the
  repository, the workflow and the tag ref. A provenance predicate over a checksum list restates
  what the certificate proves. The images differ, because provenance there names the build inputs.
- **`cosign sign -a key=value` annotations.** The certificate already carries the tag ref.
- **A supported key-based variant for forks.** It means shipping the key custody and rotation
  procedure that cosign does not document, and it breaks the one-anchor rule.

### 7.9 What the SPEC says to a fork

One paragraph, stated as a fact.

A keyless signature writes the repository URL, the workflow path, the commit SHA, the tag, the
runner class and a run link to a permanent public log. **A private fork that runs this pipeline
unchanged discloses its repository name and its commit SHAs forever.** A fork that objects edits
the workflow itself. This project supports no key mode.

---

## 8. Provenance

**`actions/attest`, SHA-pinned, with `push-to-registry: true`. The project publishes SLSA v1.0
Build Level 2 and does not chase Level 3**
([#1076](https://github.com/winniel123/verge-asm/issues/1076), #1155).

### 8.1 The generator

`actions/attest@1e69f48acb82d1966a394da916b4c1698aa569d6 # v4.2.2`, measured 2026-09-03.

**Both wrappers are rejected**, and this supersedes the earlier wrapper-pair choice. Measured:
`actions/attest-sbom` and `actions/attest-build-provenance` are composite actions over that same
pinned line. `attest-sbom` forwards `sbom-path` verbatim. `attest-build-provenance` forwards
`predicate-type`, `predicate` and `predicate-path`, and **sets none of them**. The wrapper adds
zero logic.

The reason the wrapper pair was originally kept does not exist. `actions/attest` has an
**`sbom-path`** input that creates an SBOM attestation and forbids a predicate input beside it, so
the direct call never hand-writes a predicate URI. It also writes SLSA provenance **by default**.
And `actions/attest-build-provenance` carries its own deprecation notice naming the same successor.

**The standing rule this produced: a release-signing action must sit on a supported upstream.** A
SHA pin controls *when* the project takes a change. It does not control *whether* one exists to
take. The wrapper proves it: the wrapper SHA-pins its own inner action, so pinning `attest-sbom`
v4.1.0 (its last release, 2026-03-18) pins a stale signer. A Sigstore fix reaching
`actions/attest` would arrive only after a deprecated wrapper published again, which it may never
do. This is §2.3's rule one layer down.

**`slsa-framework/slsa-github-generator` is rejected** on three measured grounds. Its README states
the project is no longer actively maintained. Its workflows **MUST** be referenced by `@vX.Y.Z`,
so a commit SHA fails the build and the SHA-pin preference cannot hold. And it changes how
consumers verify, requiring `slsa-verifier` or a cosign CUE policy instead of `gh`.

### 8.2 Six steps

Six, not fewer, because `push-to-registry` needs a single fully-qualified image reference with a
digest per step. `subject-checksums` cannot batch them.

| # | Attestation | `subject-name` | `subject-digest` | `sbom-path` |
| --- | --- | --- | --- | --- |
| 1 | provenance | `ghcr.io/winniel123/verge-asm/web` | index digest | — |
| 2 | provenance | `ghcr.io/winniel123/verge-asm/worker` | index digest | — |
| 3 | SBOM | `.../web` | `linux/amd64` manifest | `web-linux-amd64.spdx.json` |
| 4 | SBOM | `.../web` | `linux/arm64` manifest | `web-linux-arm64.spdx.json` |
| 5 | SBOM | `.../worker` | `linux/amd64` manifest | `worker-linux-amd64.spdx.json` |
| 6 | SBOM | `.../worker` | `linux/arm64` manifest | `worker-linux-arm64.spdx.json` |

Every step sets `push-to-registry: true`. Inputs the release leaves unset: `predicate-type`,
`predicate` and `predicate-path` on every step, `sbom-path` on steps 1 and 2, `subject-path`,
`subject-checksums`, and `subject-version`. `create-storage-record` and `show-summary` keep their
`true` defaults.

### 8.3 The build level

**SLSA v1.0 Build Level 2.** The guide states that level, states the condition GitHub names for
Level 3, and states plainly that this project does not meet it.

GitHub Docs are the contract: "Artifact attestations by itself provides SLSA v1.0 Build Level 2."
The same page gives one route to Level 3, a reusable workflow "that many repositories across your
organization share". The 2026-01-20 changelog claims Build Level 3 without repeating that
condition, and that wording is marketing.

**Level 3 is not merely unattractive. It contradicts two rulings in this file.** §7.5 requires the
sign and attest steps to live in `release.yml`, because a reusable workflow moves the Fulcio
identity. §2.3 ruled that the build delegates no step to a reusable workflow. Beyond that,
verge-asm is one repository, and no source settles whether one caller plus one private reusable
workflow earns the isolation GitHub describes. **The project does not publish a level it cannot
source.**

### 8.4 Where the attestation lives

**Both copies.** The GitHub attestations API holds one, and GHCR holds the second beside the image.
verge-asm is public, so the API copy also goes to the public Rekor transparency log.

The registry copy is the documented prerequisite for Sigstore Policy Controller admission control.
That is free assurance for a Kubernetes operator, and it costs one input.

An operator who pulls only the image reaches it two ways:

```sh
# bundle from the GitHub attestations API
gh attestation verify oci://ghcr.io/winniel123/verge-asm/web:vX.Y.Z --repo winniel123/verge-asm

# bundle from the registry
gh attestation verify oci://ghcr.io/winniel123/verge-asm/web:vX.Y.Z --repo winniel123/verge-asm --bundle-from-oci
```

**The registry copy is reachable by cosign online, and is a `gh` path offline.** Measured
2026-09-03: `cosign download attestation` **does** reach a native GitHub attestation online, and an
earlier community report did not reproduce. Offline is different.
`cosign verify-attestation --local-image` cannot read a native attestation out of a saved layout,
because `cosign save` writes the referrer bytes into `blobs/` and leaves nothing in `index.json`
pointing at them. §15.3 owns the offline route.

### 8.5 The subject and the count

**The index digest only. Two provenance attestations per release.** §2.3 sets
`provenance = false` and `sbom = false`, so BuildKit writes no competing provenance, each index
keeps two children, and §7.2's six-signature model holds.

**The loose Release assets get no provenance attestation.** The rejected alternative was
`subject-path: SHA256SUMS`. It loses on two counts. The keyless certificate on
`SHA256SUMS.sigstore.json` already records the repository, the workflow path, the commit SHA and
the tag, and provenance earns its place on the images because it names the build inputs. And it
does not remove cosign from the operator's path, because §7.4's blob signature is not withdrawn.

### 8.6 Provenance and the cosign signature overlap, and neither is redundant

- **They share the anchor.** Both sign keyless, through one OIDC identity, one Fulcio CA and one
  Rekor log.
- **They do not share a predicate.** Provenance names the build. A bare signature names nothing.
  §6's SBOM attestation names the contents.
- **They do not share an operator path.** `gh attestation verify` reads the provenance.
  `cosign verify` reads the signature with no `gh` and no downloaded bundle.

No primary source calls either one redundant. **The SPEC writes "the project chose both, and here
is what each buys". It never writes "the other is redundant".**

---

## 9. The release artifact set

A release publishes the two images, three loose files, one signed checksums file, eight SBOM
documents and the Trivy scan asset set
([#1072](https://github.com/winniel123/verge-asm/issues/1072), #1073, #1074).

### 9.1 Published

| # | Asset | Origin |
| --- | --- | --- |
| 1 | `docker-compose.yml` | **generated**. The workflow substitutes this release's image digests into the repository file (§2.4). |
| 2 | `.env.example` | copied verbatim from the tagged tree |
| 3 | `docker-compose.external-db.yml` | copied verbatim from the tagged tree |
| 4 | `SHA256SUMS` | over items 1 to 3 and the eight §6.1 SBOM documents |
| 5 | `SHA256SUMS.sigstore.json` | `cosign sign-blob --bundle` (§7.4) |
| 6-13 | the eight SBOM documents | §6.1 |
| — | the Trivy scan asset set | §5.3. That section owns its names and count. |

**Thirteen named files**, plus the Trivy set.

The three loose files together are the smallest set that installs the instance with no `git clone`.
`docker-compose.yml`'s `web` service uses `${POSTGRES_PASSWORD:?}`, and `.env.example` is the only
file that documents the keys. The external-db mode is documented and supported in
`docs/guides/running.md`, so it is not an edge case.

Only item 1 is generated. Items 2 and 3 are copies taken from the tag the workflow builds, so they
cannot drift by hand.

**Ordering constraint.** The workflow must produce the image digests **before** it writes the
compose asset (§4.4 steps 4 and 11).

### 9.2 Why the loose assets are signed

The compose asset's content is digests. An attacker cannot forge a digest. An attacker **can**
point the file at an older verge image that is still validly signed. That is a rollback, and image
verification alone does not catch it. A signature over `SHA256SUMS` binds the file to this tag.

**One signature over `SHA256SUMS`, not one detached signature per file.** The operator runs one
verify, then `sha256sum -c`. **The command count stays at two whatever the asset count becomes.**
That design only pays if the checksum list grows with the asset list, which is why §6 added its
eight documents to item 4.

Publishing the compose file to GHCR as an OCI artifact was rejected. It would inherit the image
verification path with no second command, but an operator could no longer fetch the file with a
plain download.

### 9.3 Not published

- **Loose prober binaries.** See §16. The worker pushes the binary from inside its own image, and
  no operator path fetches one.
- **A `docker save` image tarball.** A tarball loses the digest identity that the signature and the
  provenance bind to. An operator with no registry reach pulls on a connected host and runs
  `docker save` there.
- **The `deploy/prober/` recipe.** It is a directory, so it would ship as a tarball, and it
  contains no verge binary to pin to a release. **The scope limit is stated plainly: the no-clone
  install covers the instance, not the prober host.** `docs/guides/prober.md` step 2 keeps its
  `git clone`, and that stays correct.
- **A Sigstore trust root.** §7.8 gives the reason.

### 9.4 The two source archives GitHub attaches

GitHub attaches `Source code (zip)` and `Source code (tar.gz)` to every Release, and they cannot be
removed. **The SPEC acknowledges them and disclaims them.** verge signs no source archive, and
`SHA256SUMS` does not cover them. The build input is a git SHA, and the provenance attests the
image. A checksum over an archive that nobody rebuilds from is a promise with no verifier.

---

## 10. The Release object

### 10.1 The body

`release.yml` runs `git cliff --latest` to build the body. It does **not** parse `CHANGELOG.md`.
The generator is deterministic over the same `cliff.toml`, so the body equals the file's newest
section without a parser existing anywhere (#1078).

**Override.** When `docs/release-notes/<tag>.md` exists, the body is that file verbatim and the
generator does not run. It costs three lines in the workflow and gives every release an escape
hatch.

**The first release uses the override.** `main` holds 640 first-parent commits, and `v0.1.0` has no
predecessor tag, so a generated body would carry all 640. `v0.1.0` ships a hand-written
`docs/release-notes/v0.1.0.md`.

**GitHub's native `--generate-notes` is rejected.** It groups by pull-request label, and merged
pull requests in this repository carry **no labels at all**. It would emit one flat list, and
choosing it would impose a labelling habit the repository does not have.

**`cliff.toml` and the non-standard types.** Conventional Commit conformance is 189 of the last 200
commits. The stragglers are `security` (2), `land` (2) and `file` (1). `commit_parsers` maps
`security` into a Security group, and **skips** `land` and `file` as pre-standard noise.

### 10.2 The migration banner

`release` **prepends a one-line banner** above the body, generated body and override body alike
([#1161](https://github.com/winniel123/verge-asm/issues/1161)). The input is the §4.3 migration
diff.

**The banner always appears, in both polarities, as two fixed one-line strings.** A conditional
banner would make absence ambiguous, because an operator could not separate "this release carried
no migration" from "this release predates the banner".

`git-cliff` cannot produce this line. It parses commit messages and has no file-path context.

**`v0.1.0` is the exception.** It has no predecessor tag, so its range is empty-to-tag and every
migration reads as new. The banner states migrations present. That is honest, because a first
install creates the schema and has no rollback target.

Three routes were rejected. A fifth guard refusal would force every migration-carrying release onto
the hand-written override. Making the override the default route contradicts §10.1, which built it
as an escape hatch. And pointing at the Instance card fails outright, because the migrations-pending
badge reports pending against applied on the running instance.

### 10.3 `CHANGELOG.md` survives as generated output

The file is not retired. It stays, and it is machine-written. An offline reader is still served.

Two consequences follow. **Between releases the file lags `main`**, because it is refreshed only at
release-prep. And `docs/spec/doc-lint-tool.md` §1.3 already excludes `CHANGELOG` from doclint, so a
generated file trips no lint.

**The `[Unreleased]` non-empty gate does not exist**, because there is no hand-written
`[Unreleased]` section to gate. §4.3 guard 2 replaces it.

### 10.4 Release-prep, end to end

1. A release-prep pull request runs `git cliff --bumped-version` to name the next version. It runs
   `git cliff -o CHANGELOG.md --tag vX.Y.Z` to regenerate the file. It adds
   `docs/release-notes/vX.Y.Z.md` when the release needs a hand-written body.
2. The pull request merges through the normal squash path. **No bot, no bypass actor, no ruleset
   exception.** `main`'s ruleset already allows a self-merge, because
   `required_approving_review_count` is `0`.
3. The human pushes a bare tag at that squash-merge commit.
4. `release.yml` triggers.

**The checklist carries two further lines** ([#1156](https://github.com/winniel123/verge-asm/issues/1156)):

- Confirm the verification guide carries no "pending the first tagged release" banner.
- After the workflow completes, run the guide's three-command per-platform SBOM verify. **This is
  advisory, never a gate.** See §20.

### 10.5 The version number

`git cliff --bumped-version` names it. The human does not guess.

The rule it implements is stated in prose, so a reader can predict the answer without running the
tool: **any `feat` or any `!` in the range bumps the minor. Everything else bumps the patch.** The
`!` marker has been used once in the repository's history.

### 10.6 Why not release-please

The manual route has already been measured and it failed. `CHANGELOG.md` has been edited **once in
its life**, in the commit that created it, and about 200 commits have landed since. So a route that
asks a human to hand-write the section repeats a proven failure.

release-please fixes that, and its price is a **long-lived GitHub App private key** in repository
secrets. A pull request opened with `GITHUB_TOKEN` does not trigger `on: pull_request` runs, and
`main` requires 7 status checks. So a `GITHUB_TOKEN` release pull request can never satisfy them.

§7.1 chose keyless to avoid holding key material. A release key held only to open a pull request is
a poor trade. **The chosen route removes the hand-written text without adding the key.**

---

## 11. `release-scan.yml`

A second deliverable, distinct from `release.yml` (#1073, #1079).

| Item | Value |
| --- | --- |
| Triggers | `schedule` plus `workflow_dispatch` |
| Cron | `"7 6 * * 1"` — Monday, off the hour, clear of CodeQL's 04:17 and Scorecard's 05:23 |
| Permissions | `contents: read`, `packages: read`, `security-events: write` |
| Subject | the current release's two images, both architectures |
| Reference | the **`latest` GHCR tag**, resolved fresh on every run |
| Outcome | SARIF only. **It never fails.** |

**The dispatch trigger is not optional.** `scorecard.yml`'s own `workflow_dispatch` proves a
cron-only workflow is untestable.

**It reports and never fails.** A code scanning alert is the notification, which is what `codeql`
and `scorecard` already do on their weekly crons. A red scheduled workflow degrades into background
noise, and an auto-filed issue needs a dedupe rule nobody maintains.

**It exists because a released image ages.** An advisory published a month after the tag is
invisible unless something re-scans, and nothing else in this repository watches a published image.

**It no-ops cleanly when no release exists.** That is the state every Monday before `v0.1.0`.

**It follows `latest`, not the feed.** Reading `/releases/latest` was rejected, because it couples
a scan job to the update feed and adds an API call that can return 404 (§14.5). A withdrawal moves
the `latest` tag back, so the cron follows with no extra logic. **One consequence is stated:** while
`latest` still points at a withdrawn version, the cron scans the bad image. That is correct,
because it is exactly what a `docker compose pull` would fetch at that moment.

---

## 12. The Go toolchain pins

**One exact scalar in `.go-version`, and the proof that it shipped is read from the artifact, never
from the text that requested it** ([#1081](https://github.com/winniel123/verge-asm/issues/1081)).

### 12.1 The measurement that decided the design

**A tag written beside a digest is decoration.** Measured 2026-09-03 against the real `golang`
digest `sha256:9fdc884a…`:

```
docker buildx imagetools inspect golang:1.19.0-bookworm@sha256:9fdc884a…   -> resolves
docker buildx imagetools inspect golang:totally-not-a-tag@sha256:9fdc884a… -> resolves
```

Both succeed and both report the same index. Docker resolves the reference by digest and never
validates the tag. **So a text compare against the `Dockerfile`'s builder `FROM` proves nothing about the builder.**

**The digest carries the answer itself.** The same manifest exposes
`org.opencontainers.image.version: 1.26.8-bookworm`. The true builder version is readable out of
the pinned digest, with one manifest fetch, no build and no pull.

### 12.2 The design

1. **The set collapses from five machine-read sites to three.** A new `.go-version` file at the
   repository root holds one bare scalar, `1.26.8`, with no `go` prefix and no `v`. `ci.yml`,
   `codeql.yml` and `commentlint.yml` delete their `GO_VERSION` env block and pass
   `go-version-file: .go-version` to `setup-go`. **Three copies are deleted rather than policed.**
   The remaining sites are `.go-version`, `go.mod`'s `go` line and the `Dockerfile`'s builder `FROM`.
2. **`.go-version` is the source of truth.** `go.mod`'s `go` line is a **floor** and the rest are
   **exact**. The asymmetry is recorded, not flattened, because `go.mod` cannot express an exact
   pin.
3. **Three compares**, in `scripts/check-go-pins.sh`:

   | # | Compare | Role |
   | --- | --- | --- |
   | a | the `Dockerfile` digest's `org.opencontainers.image.version` annotation `==` `.go-version` | **the gate** |
   | b | the builder `FROM`'s decorative tag `==` `.go-version` | hygiene, so the pin comment above it cannot state something false |
   | c | `go.mod`'s `go` line `<=` `.go-version` | the floor test |

4. **A POSIX script, not a Go tool.** `commentlint` runs as `go run ./cmd/commentlint` after
   `setup-go`, so a Go checker would be built by the toolchain it is judging. A script escapes
   that. No composite action, because it buys nothing and adds a second thing to pin. This creates
   `scripts/` as a new top-level directory.
5. **Two call sites, one script.** A step inside the **existing `test` job**, and a step in
   `release.yml`'s `guard` job. It is deliberately **not a new CI job**, because `test` is already
   one of the 7 required checks. A new job would stay non-blocking until a human registered an 8th
   required check, which would grow §17's list to five items. **The cost is stated:** a pin failure
   reads as `test` failing until the log is opened.
6. **A fetch failure is fatal on release and advisory on a pull request.** Compare (a) needs a
   manifest from Docker Hub, and GitHub-hosted runners share IPs that Docker Hub rate-limits. In
   CI: retry, then skip compare (a) with a warning, while (b) and (c) stay blocking. In `guard`:
   retry, then **fail**. A release is rare and deliberate and must not publish on an unproven
   builder. **Mirroring the `golang` base into GHCR was rejected**, because it is a new
   supply-chain surface with its own refresh problem, and §2.3 already rules that a release pins
   every byte itself.
7. **A fourth assert lives in the `scan` job, and it is not redundant.** It reads the shipped
   binary's `stdlib` version out of §5.3's four Trivy runs and compares it to `.go-version`. It
   survives ruling 3 because `GOTOOLCHAIN` defaults to `auto` and `go.mod` is a floor, so a builder
   image may fetch a newer toolchain than it ships. **The annotation proves the builder. This
   proves the product.**

### 12.3 Dependabot

`dependabot.yml` gains a **`docker` ecosystem over three surfaces**, grouped into one weekly pull
request: the root `Dockerfile` (the `golang` digest and two distroless digests),
`deploy/prober/Dockerfile` (`alpine`), and `docker-compose.yml` (the postgres digest).

This deliberately creates the adversary the §12.2 check guards, and it turns a frozen and rotting
digest into a reviewable pull request.

**One caveat is accepted: no CI job builds `deploy/prober/Dockerfile`**, so its bumps land
unproven. A stale hardened SSH target is the worse risk, and its base is alpine, so it can never
trip the §12.2 check.

### 12.4 What this does not do

No new required status check. No fifth manual repository setting. No mirrored base image. No
Go-language checker. **No `toolchain` directive in `go.mod`**, because `CLAUDE.md` already
forbids one and it breaks CI's `-mod=readonly` build.

---

## 13. The Settings -> Instance card

### 13.1 The four-line block

```
# on the host — verge cannot rewrite its own image
docker compose pull
docker compose up -d web worker
docker compose ps web worker
```

The third line is `docker compose ps web worker`
([#1071](https://github.com/winniel123/verge-asm/issues/1071)). **The card stays at four lines and
gains no diagnostic line.** The card is a happy-path upgrade recipe rendered beside a version
number, and a failure branch printed on every render teaches an operator to skip the block.

### 13.2 Why no migration-status mode

Three measured facts remove the job the old command was meant to do:

1. `cmd/web/main.go`'s `main` calls `migrateUp` before it serves, and fails with `log.Fatalf`. **A
   running `web` container has already applied every embedded migration**, so a status probe inside
   it can only answer "current".
2. In the pre-pull position the probe reads the **old** binary's embedded set, so it also answers
   "current".
3. `docker compose exec` fails against a crash-looped container, so the failure case returns a
   compose error and not a migration answer.

**`web` gets no migration-status mode and the image gets no `verge` alias.** The card already
carries the pre-restart answer: `migrationsPending` (`cmd/web/settings.go`) renders the
schema-current badge, and that function is unchanged.

The open question after an upgrade is different. Did the new image land and come up healthy?
`docker compose up -d` does not guarantee it, because compose skips recreation when the resolved
image did not change. `docker compose ps web worker` answers that, mirrors §13.1's
`docker compose up -d web worker` line service-for-service, and reads the `HEALTHCHECK` both Dockerfile targets already declare.

### 13.3 The four sites and the one test

The literal exists in four places, and nothing checks that they agree:

- `updateHostSteps` (`cmd/web/settings.go`)
- the `Version & updates` card's host-steps `<pre>` (`design-system/examples/console/Settings.jsx`)
- `release.steps` (`design-system/fixtures/fixtures.json`)
- the **Guided host steps** bullet (`docs/guides/running.md`)

**All four become the same four lines, byte-identical.** `running.md` currently prints three lines,
because it drops the comment line. It gains that line. A guide block that is deliberately shorter
cannot be told apart from a guide block that drifted.

**One Go test:** `updateHostSteps` must equal `release.steps` in
`design-system/fixtures/fixtures.json`. `cmd/web` already loads that fixture in four tests. The
Markdown and JSX copies stay unchecked, because parsing a Markdown fence or a JSX string literal
from a Go test is a fragile thing to own.

**Which copy is authoritative.** ADR-0110 governs that a `<pre>` block of host steps sits in that
place, in that style, and the JSX example remains the IA spec for the composition. **ADR-0124
governs which commands the block contains, and `updateHostSteps` is authoritative for the text.**
Without this split, a later session reads `Settings.jsx` as upstream and reverts the Go constant.
The design package cannot know that `docker compose exec` fails against a crash-looped container.

### 13.4 `.Steps[]` stays compiled-in, permanently

The comment above `updateHostSteps` (`cmd/web/settings.go`) promised a feed-delivered list "until
B5". **The SPEC strikes that promise**, and the comment now records the refusal instead.

The release feed is an external service. A feed-delivered step list lets whoever controls the feed
put arbitrary shell text in front of an admin, inside a panel that reads as authoritative.
ADR-0124 already refuses to let the UI compose a shell, and taking the text from the network is the
same hazard with one more hop.

### 13.5 The guide

The **Guided host steps** bullet in `docs/guides/running.md` states the boot-time contract, because
the block no longer names migrations. The **Migrations-pending badge** bullet is unchanged.

The `## Upgrades` section gains the failure branch:

> `web` applies migrations at startup and refuses to serve if one fails. If
> `docker compose ps web worker` shows `web` restarting or unhealthy after an upgrade, read
> `docker compose logs -n 50 web`. Each migration runs in its own transaction, so the failing
> migration reverts and the schema stays at the last version that applied.

Measured for the last sentence: no file in `db/migrations/` carries `-- +goose NO TRANSACTION`, and
`migrateUp` calls plain `goose.Up`. An earlier migration in the same run has already committed, so
the claim is "the last version that applied" and never "the previous schema is untouched".

### 13.6 The digest-pinned asset and the frozen block

`docker compose pull` against a digest-pinned file whose digests **just changed** does pull, because
the new digests are not in the local cache. So the four lines work verbatim for an operator running
the §9.1 asset, after one fetch of the new release's `docker-compose.yml`. That is the same act a
source contributor performs with `git pull`.

- A default install follows `:latest`. The four lines are complete.
- An asset install follows digests. The four lines are complete after one fetch.

---

## 14. When a published release is bad

**Repair is always `vX.Y.Z+1`, containment is two reversible acts, and neither is tag surgery or
deletion** (#1161).

**Repair** makes a good release exist. **Containment** stops the bad one from reaching a new host.
They have different levers and different actors, so a roll-forward rule on repair does not settle
containment.

### 14.1 The sharp fact

`migrateUp` (`cmd/web/main.go`) calls `goose.Up` and nothing else. All 68 migrations in `db/migrations/`
carry a `-- +goose Down` block, and **no code path ever runs one**. §13.2 refused the
`verge migrate` route, so no operator command exists either. **An image downgrade is therefore not
a schema downgrade.**

### 14.2 Repair

A bad `vX.Y.Z` is answered by `vX.Y.Z+1`, always, even when the fix is one character. **The version
never skips**, because a skipped number signals a withdrawal that `isNewer` cannot represent.

This is already a machine rule for git. §17's ruleset puts `update`, `deletion` and
`non_fast_forward` on `refs/tags/v*` with an empty bypass, §4.4 rules a pushed tag spent, and §1.1
rules a dead tag permanent.

### 14.3 Downgrade is supported only when the release carried no migration

- **No migration.** An image pin to the previous digest is a clean and complete rollback.
- **A migration landed.** Old code then meets a new schema. That is safe for an additive migration
  and broken for a destructive one, **and the operator has no way to tell them apart**. The route
  is a restore from the pre-upgrade dump, not an image pin.
- **A `goose.Down` route is rejected.** It is out of this map's scope and it reopens §13.2.

§10.2's banner is what tells the operator which kind a release is.

### 14.4 Withdrawal: one lever permitted, two refused

The feed reads `/releases/latest`, which skips draft and pre-release. §17's ruleset governs a git
ref, not a Release object, so it does not reach any of these.

| Lever | Verdict | Reason |
| --- | --- | --- |
| the **pre-release flag** | **permitted** | the one working lever. Assets stay publicly downloadable, so §9 and §15.3 both survive. |
| the **draft flag** | refused | a draft hides its assets, which breaks §9, §6 and §15.3 at once. |
| **deleting the Release** | refused | same asset loss, plus it destroys the CHANGELOG permalink. |

§1.3 refused pre-release **tags**. That refusal does **not** reach the flag, because the flag names
no version string and cannot trip §15.4's negative identity rule.

**The `latest` GHCR tag may move backwards.** It is the only movable pointer a release creates, and
it is what §13.1's `docker compose pull` resolves on a default install. A hand-run
`docker buildx imagetools create -t <image>:latest <image>:vX.Y.Z-1` moves it. **No third workflow
file and no `workflow_dispatch`**, because this is an incident act one person performs once.

**`ghcr.io/...:vX.Y.Z` stays pullable forever.** Deleting a GHCR package version breaks every §15
verify command, breaks §15.3's kit for anyone who already built one, and breaks §9.1's
digest-pinned asset for anyone who pinned it. **Immutability is the property the whole supply chain
rests on. Containment is `latest`, never deletion.** §4.4 already covers the different case of a
**blocked** release, which leaves nothing public at all.

### 14.5 `v0.1.0` cannot be withdrawn

A named exception, and it dissolves permanently once a second release exists.

Both levers fail on the first release. `/releases/latest` returns **404** when no non-prerelease
release remains, and `fetcher.go` returns `status 404`, and `Check` logs it and returns
before it writes, so the cache is untouched. And `latest` has no earlier tag to move back to.

Flagging `v0.1.0` pre-release is therefore **refused**. It would make every instance's check fail
silently forever, which is worse than serving a known-bad version that `v0.1.1` replaces. **The
first release is superseded, never withdrawn.**

### 14.6 The silence window is accepted

An operator already running the withdrawn release sees `current` and is told nothing. Once the feed
serves the older version again, `isNewer(older, running)` is false, so `release.go` writes
`current` and clears the latest fields.

**Accept it, state it out loud, add no signal.** §1.4 already rejected a version detector, and
`isNewer` compares numeric cores only, so the check has no vocabulary for "the version you run was
withdrawn". The successor release closes the window.

**So withdrawal contains new installs only.** The SPEC states that sentence.

### 14.7 The maintainer incident procedure

1. Flag the Release pre-release. The feed stops serving it.
2. Move `latest` back with `imagetools create`. A default `docker compose pull` stops fetching it.
3. Ship `vX.Y.Z+1`, carrying a `docs/release-notes/vX.Y.Z+1.md` override that states the withdrawal
   and its reason.

Containment comes first, because it is two commands and repair is a full pipeline run.

**A withdrawal forces the successor onto the override route.** `CHANGELOG.md` gets nothing. It is
generated from commits, a withdrawal is not a commit, and a hand-edited line would break §10.3.

### 14.8 The operator route: four cells, two shapes

| Case | Route |
| --- | --- |
| No migration, no-clone install | Download the **previous** release's `docker-compose.yml` asset and run `docker compose up -d`. One file swap. |
| No migration, cloned install | Pin the previous digest in the repository's `docker-compose.yml`, the way `docker-compose.yml` already pins postgres. Then `docker compose up -d`. **Never `docker compose pull`**, because that re-resolves `latest`. |
| A migration landed, either shape | `docker compose down`. Restore the pre-upgrade dump into a clean volume. Then start on the pinned old digest. **Restore before start**, because starting the old `web` first puts old code against the new schema. |
| Any cell | The **in-app restore** under `backup-and-restore.md`'s **Restoring — preflight, then a typed confirm** is the wrong tool. It runs inside the `web` the operator is trying to replace. |

### 14.9 Where the prose lands

**The operator half** becomes a `## Rolling back` section in `docs/guides/running.md`, immediately
after `## Upgrades`. Rollback is the inverse of the upgrade, with the same audience and the same
object, and that section already links the pre-upgrade backup drill. `docs/guides/troubleshooting.md`
gets one cross-link in "Where to look next" and holds no procedure of its own. **No new page.**

Two defects are fixed in the same ticket. `backup-and-restore.md`'s **The pre-upgrade backup
drill** shows only the source-build upgrade, which §2.4 made stale for a default install, and it
must show the image-pull upgrade beside it. And that section's *if the upgrade misbehaves*
paragraph must link the new section instead of half-stating it.

**The maintainer half** becomes a new `## Releases` section in `CONTRIBUTING.md`. The deciding
fact: `docs/guides/embed.go` globs `docs/guides/*.md` only, so `CONTRIBUTING.md` does **not** ride
inside the shipped `web` image. **An incident procedure for the release pipeline must not be
compiled into the product it repairs.**

---

## 15. The operator verification contract

**The SPEC states the contract. An implementation ticket writes the prose** (#1156).

The map's destination names two workflow files and does not name the guide, so the prose sits
outside this map and the contract sits inside it. The implementer writes sentences. **The
implementer decides no contract.**

### 15.1 The inventory an operator can verify

2 index digests, 4 platform digests, 6 cosign signatures, 2 provenance attestations, 4 SBOM
attestations, 1 blob signature, and the 13 named §9.1 Release assets.

**The trust anchor:** keyless, one Fulcio identity, and no verge-asm public key (§7).

**The SLSA level:** Build L2 published, L3 named and refused with GitHub's own condition stated
(§8.3).

### 15.2 The commands

Every command is already ruled or measured. **The SPEC invents none.**

**The image, with `gh`** (§8.4). `--repo` alone proves only that some workflow in this repository
signed the image, so the required form pins the signer workflow:

```sh
gh attestation verify \
  oci://ghcr.io/winniel123/verge-asm/web:vX.Y.Z \
  --repo winniel123/verge-asm \
  --signer-workflow winniel123/verge-asm/.github/workflows/release.yml \
  --source-ref refs/tags/vX.Y.Z \
  --deny-self-hosted-runners
```

Measured 2026-09-03: `--signer-workflow` alone fails with
`at least one of the flags in the group [owner repo] is required`, so `--repo` stays
([#1147](https://github.com/winniel123/verge-asm/issues/1147)).

Measured on the same date: `gh attestation verify oci://<image>:<tag>` resolves a tag to the
**index digest**, never a platform digest. `cli/cli` calls `remote.Get` and never `remote.Image`,
so no platform matching happens, and `--bundle-from-oci` does not change it. **That agrees with
§8.5.**

**The image, with cosign** (§7). The exact identity is the primary form:

```sh
cosign verify \
  --certificate-identity https://github.com/winniel123/verge-asm/.github/workflows/release.yml@refs/tags/v0.1.0 \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  ghcr.io/winniel123/verge-asm/web:v0.1.0
```

A script that checks many tags uses the regexp form in a secondary block, and the guide states that
it accepts any tag from this workflow:

```sh
  --certificate-identity-regexp '^https://github\.com/winniel123/verge-asm/\.github/workflows/release\.yml@refs/tags/v' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
```

**The checksum list:**

```sh
cosign verify-blob \
  --bundle SHA256SUMS.sigstore.json \
  --certificate-identity <the identity above> \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  SHA256SUMS
```

**The per-platform SBOM, three commands** (§6.3, #1147). A child-bound attestation is **not
discoverable from the index reference** on either route, so the operator must name the child digest:

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

Step 3 runs four times per release.

**`--predicate-type` is mandatory**, because the command defaults to
`https://slsa.dev/provenance/v1`. **The SPDX URI is version-derived.** `actions/attest/src/sbom.ts`
builds it as `https://spdx.dev/Document/v${version}`, so SPDX 2.3 gives
`https://spdx.dev/Document/v2.3`. A plain `https://spdx.dev/Document` does not match.

**Three `imagetools inspect` caveats the guide must carry.** The format string must include the
variant, because without it `linux/arm/v7` prints as `linux/arm`. A reader must skip an
`unknown/unknown` row, which is a BuildKit attestation manifest. And the command exits 1 on a
non-index reference, which a verge-asm release never is.

### 15.3 The air-gap kit gets its own page

**A new `docs/guides/verifying-releases-airgapped.md`, `section: Operating`, `order: 5`.** The
file does not exist yet, and [#1260](https://github.com/winniel123/verge-asm/issues/1260) writes
it.

The kit is **23 named files, 6 layout directories, 2 tool binaries, 2 trust roots and 12 verify
commands** (#1148). The current guide is 159 lines, so as a section it roughly triples the page,
and it serves a different task on a different day.

Three facts decided the split:

- **A new guide file costs nothing to wire.** `docs/guides/embed.go` embeds `*.md`, and
  `docs-site/src/content.config.ts` globs `*.md` and derives the site nav from front matter with no
  hardcoded section list. No test pins the guide count.
- **A second page buys a second in-app search row.** `loadGuideIndex` (`cmd/web/search.go`) indexes
  title and description only, so a section inside one page is invisible to that search.
- **Both pages ship inside the `web` binary either way.**

**The cost of splitting is drift, so the air-gap page states no contract of its own.** It links back
to the main guide for the identity pin and the tag rule, and holds only the carry-in procedure and
the offline commands.

**Every kit file is produced on the connected host. The air-gapped host only reads.**

| Group | Count | Note |
| --- | --- | --- |
| tools | 2 | `bin/cosign` v3.1.3 or later, `bin/gh` v2.96.0 or later |
| trust roots | 2 | `trusted_root.json` for cosign, `trusted_root.jsonl` for `gh` |
| OCI layouts | 6 directories | one `cosign save` per digest |
| attestation bundles | 6 files | one `gh attestation download` per digest |
| Release assets | 13 named files | §9.1, plus the Trivy set |

**Six layouts, not one.** `cosign save` copies the signature and a cosign `.att` attestation of the
**named digest only**. It drops every child signature, it has no `--recursive` flag, and
`--local-image` takes a directory path rather than a digest. Two saves into one directory do not
merge, because the second replaces `index.json`. **Disk cost: each platform image is stored about
twice.**

**cosign alone does not cover every artifact.** A native GitHub attestation reaches `blobs/` as
orphan bytes that no cosign command can find, so `cosign verify-attestation --local-image` cannot
read the provenance. The command that does reach it is:

```sh
gh attestation verify <layout>/blobs/sha256/<digest> \
  --repo winniel123/verge-asm \
  --bundle <bundle>.jsonl \
  --custom-trusted-root trusted_root.jsonl \
  --format json
```

**One thing no upstream document states, and the guide must:** a `gh` verification of an image uses
a **file**, and that file is the manifest blob inside the saved layout at
`<layout>/blobs/sha256/<digest>`. An OCI digest is the SHA-256 of the manifest bytes, so the layout
supplies the artifact and the kit needs no extra copy.

**Two operator notes.** `--trusted-root` is not optional offline, and cosign fails while updating
the TUF remote mirror without it. And in a non-interactive shell the `gh` success case printed
nothing and returned exit code 0, so the guide tells the reader to use `--format json` or to test
the exit code.

**The air-gapped host runs** six `cosign verify --local-image`, six `gh attestation verify`, one
`cosign verify-blob`, and one `sha256sum -c`.

### 15.4 Which limits the guide states out loud

**In the guide: the negative identity rule.**

> Every published release carries a bare `vX.Y.Z` tag and stamps a bare `X.Y.Z` version. A version
> string carrying a suffix did not come from this pipeline.

It is the cheapest discriminator in the whole guide, because it needs no tooling, no network and no
trust root. **The rider travels with it:** the rule is only as strong as §17's tag ruleset, so the
guide prints it as a **convention**, never as a cryptographic guarantee. The guide also states
§1.4's plain consequence, that a source build is never told a release shipped.

**In the guide: never trust `latest` as an identity.** The existing digest-pinning callout is the
right place, and it gains one sentence naming `latest`. **It must also state that `latest` may move
backwards** (§14.4), so a digest resolved from `latest` yesterday may differ from what `latest`
resolves to today.

**Not in the guide: the unproven link** (§20). A verification guide is a contract, and "this
command may not work" is not a contract.

### 15.5 The banner, and when the rewrite merges

**Delete the banner. Replace it with nothing. Merge the whole rewrite before the tag.**

`embed.go` compiles the guides into the `web` binary, so the guide ships **inside** `v0.1.0`. A
"pending the first tagged release" banner would ride inside the very release it says has not
happened. A guide that ships inside a release also gains no "verified against vX.Y.Z" line, because
the image that carries it already versions it.

**The sequencing conflict, stated.** An earlier draft held the rewrite back until the first release
measured §20's unproven link. That is incoherent, because a rewrite landing after `v0.1.0` means
`v0.1.0`'s image carries the **old** guide and all nine defects. **Shipping known-wrong docs in the
first release is the exact failure the rewrite exists to remove.**

So the rewrite merges as **its own pull request**, before §10.4's release-prep pull request. A doc
rewrite does not ride a mechanical `CHANGELOG.md` regeneration.

### 15.6 The three SBOM routes, separated by purpose

- **The convenience route.** `gh release download <tag> --repo winniel123/verge-asm --pattern
  'web-linux-amd64.spdx.json'`, then `grype sbom:./web-linux-amd64.spdx.json`. Plain SPDX from
  §9.1. It needs no attestation machinery. **This is the route for a re-scan.**
- **The trust route.** The three commands in §15.2, against the **child** digest. **This is the
  only route that proves the SBOM describes that digest.**
- **cosign is not an SBOM route.** §7 signs six digests, and `cosign verify` checks **signatures**.
  The guide gives cosign its own section and names all six signatures, including the four child
  signatures `--recursive` writes.

### 15.7 The nine defects, as the rewrite's checklist

| # | Defect |
| --- | --- |
| 1 | It promises a **CycloneDX** SBOM attestation. §6.3 attests **SPDX only**. |
| 2 | It states there are "no standalone binaries and no Release-asset downloads". §9.1 publishes 13 named loose files. |
| 3 | It binds **every** attestation to the index digest. §6.3 binds the SBOM to the platform digest. |
| 4 | It gives bare `--repo` as the minimum verify command. §15.2 requires three more flags. |
| 5 | It states no SLSA level. §8.3 publishes **SLSA v1.0 Build L2**. |
| 6 | It never mentions the floating `latest` tag (§2.2). |
| 7 | It never mentions §7.2's **six cosign signatures**. An entire artifact class is absent. |
| 8 | Its cosign section verifies the wrong object. It runs `cosign verify-attestation --type slsaprovenance`, and §7's cosign artifact is the **signature**. Whether `--type slsaprovenance` maps to `https://slsa.dev/provenance/v1` in cosign v3.1.3 is **unmeasured**, so the block goes rather than gets patched. |
| 9 | Its SBOM re-scan command probably does not run. It pipes `gh attestation download` output into `grype sbom:`, and that command writes a Sigstore bundle rather than an SBOM document. §15.6's Release-asset route makes the question moot. |

**What survives unchanged.** The "no verge-asm public key" line, the two image names, the `vX.Y.Z`
usage, the `linux/amd64` plus `linux/arm64` platform pair, and the digest-pinning callout.

**One SPEC constraint on the prose, because it is machine-read.** `loadGuideIndex`
(`cmd/web/search.go`) indexes each guide's front-matter title and description only, never the body. **So each page's description
must carry every word an operator would search for.** A word that appears only in the body is
unfindable from inside the product.

---

## 16. The prober binary

**The prober's origin is the worker image that carries it, nothing signs it again, and the vantage
host bounds the binary rather than verifies it**
([#1239](https://github.com/winniel123/verge-asm/issues/1239)).

### 16.1 The release pipeline needs no prober step

The `Dockerfile` builds `prober-linux-amd64` and `prober-linux-arm64` in the **same builder
stage** as `web` and `worker`, and its `worker` stage copies them into the image at `/app/probers`.

`cmd/worker/main.go` reads `VERGE_PROBER_DIR`, default `/app/probers`.
`DirBinaryProvider.Binary` opens `prober-<goos>-<goarch>` from that directory, and serves the
own-arch `VERGE_PROBER_PATH` only when the requested platform is the instance's own
(`internal/remoteexec/binary.go`). **Every path reads the worker image's read-only
filesystem. No path fetches a release asset, and no path reaches the network.**

So verifying the worker image under §7, §8 and §15 already covers the prober binary. **The pushed
binary has never left an image this pipeline signs, attests and describes in an SBOM.**
`release.yml` and `release-scan.yml` are both unchanged.

### 16.2 The release artifact set is unchanged

§9.3 stands, on a ground stronger than its own. A loose signed prober would be a **second copy
under a second subject**, over bytes a signed image already covers, and **no code path consumes
it**. `SHA256SUMS` grows by nothing.

### 16.3 Which boundary this rules on

**Host to verge only.** Verge to host is built, not open: SSH public-key auth, the trust-on-first-use
host-key pin, `Probe`'s bounded prober stdout, and its `uname` arch check that gates the push
(`internal/remoteexec/probe.go`).

### 16.4 Nothing verifies the binary, at either end

A refusal with reasons, in the shape §1.3 and §17 use. Three mechanisms rejected:

| Mechanism | Why it lost |
| --- | --- |
| The worker verifies before it pushes | It would verify a file on its own image layer using a trust root shipped in the same image, so **a compromised image passes its own check**. It also needs a network path to Sigstore, which §15.3's kit exists to avoid. |
| The host verifies after the binary lands | The host needs cosign, a trust root and a network path. `deploy/prober/README.md` keeps that host to `alpine` plus `openssh-server` deliberately, and states the host "is the operator's rather than ours". |
| A detached signature rides the push | It re-signs bytes §7's anchor already covers, and lands them on a host with nothing to check them against. |

**What the operator relies on instead**, five controls that all exist today:

1. Image verification at pull (§7, §8, §15).
2. SSH public-key auth, with `restrict` and `from=<egress>` in `authorized_keys`.
3. The trust-on-first-use host-key pin. A change is a hard failure, never a prompt.
4. The `0700` random temp path, `/tmp/verge-prober-<8 random bytes>` (`tempPath`, `probe.go`).
5. The delete after every run (`Probe`'s deferred `rm -f`, `probe.go`).

### 16.5 The lent-host case

In every install today the vantage-host operator **is** the instance operator, so there is one
trust decision and it is made at image pull. **The condition that would change that is stated: the
vantage-host operator is not the instance operator**, on a lent or shared host.

When it holds, **verge offers that person no origin proof, and says so.** Their lever is the SSH
account, not a signature: a non-root user, `restrict`, `from=`, `cap_drop: [ALL]` and
`no-new-privileges`. **Those bound what the binary may do. They do not prove what it is.**

A sharp consequence: `Probe`'s deferred `rm -f` removes the binary after every run, so **a lent-host operator
cannot inspect afterwards what verge ran**.

Two alternatives rejected. **A published per-release prober digest** reopens §9.3 through a side
door and publishes a digest nothing verifies. **Keeping the pushed binary on disk** trades a real
hygiene control for an inspection window that only helps a party who has already run the binary.

### 16.6 The transitive claim carries its bound

Verifying the worker image covers the prober **for whoever holds that image**. That is the instance
operator. A lent-host operator holds only a binary that arrives over SSH and is deleted. §15 makes
a verification claim a contract, so **the claim and its bound are stated together, in one place**,
on the page the affected reader opens.

### 16.7 Where this lands

- **This section.** The scope statement and the §16.4 refusal.
- **`docs/guides/prober.md`** gains a subsection for the host operator: the five controls, the
  §16.5 condition, and the bounded §16.6 claim. That is prose, so an implementation ticket writes
  it. **It may land after `v0.1.0`**, because `prober.md` today is **silent** on binary trust
  rather than wrong, and a silent page ships nothing false. §15.5's reason does not reach it.
- **`docs/guides/verifying-releases.md` gains nothing.** A prober paragraph there invites the
  reader to hunt for a prober signature that does not exist.
- **`deploy/prober/README.md`'s posture table gains no row.** It lists container controls, and the
  SSH account controls already sit above it.
- **`CONTEXT.md` gains nothing.** The instance-operator and vantage-host-operator split is a
  **deployment role, not a domain term**, and `CONTEXT.md` answers with one `operator` across 1,903
  lines.

---

## 17. The repository settings a human applies by hand

A workflow cannot set these, so the SPEC lists them in one place. **The list is final at four
items.** All 25 closed tickets on this map added none beyond these.

### 17.1 The four items

**1. A tag ruleset.** Target `refs/tags/v*`. Bypass list **empty**. Enforcement active.

| Rule | Verdict |
| --- | --- |
| `update` | take — this is the immutability |
| `deletion` | take |
| `non_fast_forward` | take as a belt |
| `tag_name_pattern` `^v[0-9]+\.[0-9]+\.[0-9]+$` | take |
| `creation` | **rejected** |
| `required_signatures` | **rejected** |

**`creation` is rejected because it would lock the owner out.** The bypass list is empty, so
`creation` refuses a `v*` tag from everyone and no release could ever be cut. It is recorded as a
rejected rule, not as an omission, because a later reader tightening this ruleset reaches for it by
instinct.

**`required_signatures` is rejected on §7.1's own grounds.** A required tag signature reintroduces
the durable key material keyless exists to avoid. Two further reasons: a signed tag must be
annotated, and §1.1 settled on a bare tag. And the threat is a compromised account, which a key on
the same machine as the credential does not mitigate. **It becomes worth revisiting if the
repository gains a second person with push access.**

**`tag_name_pattern` is deliberately redundant with §4.3's guard 1.** Once tags are immutable,
`git push origin v1.2` creates a malformed tag that the guard correctly refuses, and that tag can
then never be deleted or moved. §4.4's spent-tag rule turns a typo into permanent litter. The
ruleset pattern refuses the push, so nothing is created and nothing is spent. **The workflow guard
stays**, because a ruleset lives in repository settings where the repository cannot see it.

**2. `sha_pinning_required: true`.** Measured `false` today. All 13 actions already pin, so the flip
costs nothing and stops a later drift that no reviewer catches.

**3 and 4. The two GHCR package visibility flips.** A first push produces a **private** GHCR
package, and no REST endpoint flips visibility. A human flips each package by hand, on two settings
pages. **Until it is done, every §2.1 image reference resolves for nobody**, and
`docker compose pull` fails for every operator.

Carry one caveat: GitHub docs contradict themselves on whether a package inherits repository
visibility. The private-by-default reading is taken, and **the first real push is the
measurement.** Delete the step if it is disproved.

### 17.2 What is refused

**`allowed_actions: selected` is refused.** SHA pinning already fixes the bytes an action runs. An
allowlist adds a recurring manual gate before CI can run, for no additional integrity guarantee.

### 17.3 Scorecard, and an accepted cap

The live score is 6.5, dated 2026-08-23. `Signed-Releases` and `Packaging` both report `-1` today.

- `releasesAreSigned` matches asset suffixes including `.sigstore.json`, so §9.1's
  `SHA256SUMS.sigstore.json` matches. **That scores 8.**
- `releasesHaveProvenance` matches **only** an `.intoto.jsonl` Release asset. §8.5 put provenance
  in the registry and §9.3 refused loose attestation assets, so the check **caps at 8**.

**Accept 8.** Reaching 10 costs one `.intoto.jsonl` Release asset, which contradicts two settled
decisions. **The cap is a deliberate, cited refusal, so a later session does not "fix" it.**

`Token-Permissions` and `Dangerous-Workflow` both score 10 already, and §4.2's job table holds
both.

---

## 18. Where the records land

**One new ADR for this pipeline, one for the prober, and two amendments. This map edits no
production Go code** ([#1240](https://github.com/winniel123/verge-asm/issues/1240)).

| Record | Subject | Source |
| --- | --- | --- |
| [**ADR-0138**](../adr/0138-a-release-pins-every-byte-it-builds-so-it-delegates-no-build-step-and-anchors-identity-in-its-own-workflow.md) | *a release pins every byte it builds, so it delegates no build step, and its identity is the keyless anchor of its own workflow* | §2.3, §8.3, §7 |
| [**ADR-0139**](../adr/0139-the-probers-origin-is-the-image-that-carries-it-and-a-host-bounds-the-binary-rather-than-verifies-it.md) | *the prober's origin is the image that carries it, and a host bounds the binary rather than verifies it* | §16 |
| [**amendment on ADR-0001**](../adr/0001-stack-and-runtime.md) | the toolchain pins (§12) | #1081 |
| [**amendment on ADR-0124**](../adr/0124-a-backup-carries-data-and-no-secret-and-updating-is-guided-not-self-applied.md) | three bullets (§13.4, §1.3, §14.6) | #1071, #1129, #1161 |

**ADR-0138 has three sections**, one per side of one ruling. §1 the build pins and delegates
nothing. §2 Build L2 and the L3 refusal. §3 the keyless anchor, and the `required_signatures`
refusal that follows from it. They are one ruling seen from three sides, and split apart each cites
the other two as its reason. ADR-0124 is the precedent, because it rules on backup **and** on
updates in one document.

**The number is `0138`, and `0090` stays a gap.** `ADR-0059`'s *ADR-0090 is left unused* line and six sites in
`docs/research/sensitive-ports.md` state in terms that ADR-0090 is left unused. Taking `0090` would
falsify six committed sentences and force an ADR-0058 mark at each. ADR-0124 set the neighbouring
precedent when it refused to resurrect the clobbered `0118`.

**ADR-0139 takes the next number.** ADR-0138 is **closed to any subject outside the tag-to-release
pipeline**, and §16's subject is a runtime trust boundary.

**The ADR-0124 amendment carries three bullets and not three amendments.** ADR-0058's table
separates the conventions: a **mechanism** gets a withdrawal, and a **claim about the world** gets
an amendment. All three bullets are claims about the world. Three bullets is size, not kind.

**The amendment shape is `## Amendment — [#NNN](url): <headline>`, ticket-referenced, with no date
in the heading.** The repository has no dated-amendment convention. See `ADR-0001`'s #119 amendment heading and the seven
amendments in `ADR-0004`.

**Once ADR-0138 merges, a later addition is an `## Amendment` block. It never edits the Decision.**

### 18.1 The rule a later ticket follows

A ticket resolving after #1240 states exactly one of three things:

- **(a)** it adds a bullet to a named existing amendment.
- **(b)** it earns its own ADR, and says why ADR-0138 does not hold it.
- **(c)** it names no ADR, and gives the reason.

### 18.2 Merge order

**The ADRs merge before the `v0.1.0` tag.** The reason is not §15.5's: `embed.go` globs
`docs/guides/*.md` only, so no ADR compiles into the image. The reason is that §10.1 generates the
Release body over the tag range, so **an ADR landing after the tag is absent from the release it
explains**. A reviewer of `release.yml` also needs the ruling that produced it.

### 18.3 The stale ADR-0118 citation

`internal/release/release.go`'s package doc cites ADR-0118 for a feed default whose infrastructure
does not exist, and ADR-0118 is about report scheduling. `internal/release/fetcher.go` carried the
same citation and no longer does — its only ADR citation is ADR-0124, on `feedTimeout`.

**One site now, and an open ticket already owns the file.** Sweep ticket #1165 lists it, and its
§4.8 caps a `package-doc` block at 3 lines, so the sweep most probably **deletes** the sentence.
**After this effort only the ADR number is wrong**, because `release.yml` and GHCR will exist. The
correct citation is ADR-0138.

**This map states the fact and #1165 makes the edit.**

**A separate defect, since discharged.** `cmd/web/auth.go` and `credflow_sessions_test.go` cited
ADR-0118 for #408, which is ADR-0117. Re-measured, no Go site cites ADR-0118 for #408: the one
surviving comment, on the every-other-session revocation in `cmd/web/auth.go`, reads
`(ADR-0117, #408)`. No `adr-gap` issue is needed.

---

## 19. Invariants

These hold across every ruling above. A change to any of them reopens the ticket that set it.

| Invariant | Value |
| --- | --- |
| Guards in `release.yml` | **four** (§4.3) |
| Jobs in `release.yml` | **five** (§4.2) |
| Required status checks | **seven**. No eighth is registered. |
| Manual repository settings | **four** (§17) |
| Workflow files this map delivers | **two** |
| `contents: write` and `id-token: write` in one job | **never** |
| Production Go code this map changes | **none**. §1.4, §3 and §13.3 are work the implementation map cuts. |
| Trust anchors | **one**, keyless (§7) |
| Image signatures per release | **six** (§7.2) |
| Attestations per release | **six** (§8.2) |

---

## 20. Open points

Nothing here blocks the SPEC. Each names who measures it and what a failure reopens.

**Measure before `release.yml` is written:**

1. **That `docker compose pull` pulls a service which also declares `build:`.** The
   `--ignore-buildable` flag is strong evidence that the default pulls buildable services. **If
   this fails, §13.1's `docker compose pull` line cannot stand and §13 reopens.** One `docker compose` run.
2. **What `docker compose up` does when the tag is absent locally and `build:` is present.** The
   Compose specification describes the `pull_policy` default differently with and without a build
   section. CI's build-first order makes this moot for CI, and it is not moot for an operator who
   skips the pull.
3. **Re-measure every action SHA.** `actions/attest` v4.2.2 at
   `1e69f48acb82d1966a394da916b4c1698aa569d6`, and the `sigstore/cosign-installer`,
   `aquasecurity/trivy-action`, `docker/bake-action`, `docker/setup-buildx-action` and
   `docker/login-action` pins.
4. **That `actions/setup-go` reads a bare `.go-version` file and resolves the exact patch** (§12).
   This is documented behaviour and is not yet measured on this repository.

**Measure after `v0.1.0`, as an advisory acceptance step (§10.4):**

5. **That a child-bound SBOM attestation is reachable end to end.** No public GHCR image carries a
   per-platform attestation, so nothing existed to test against. Querying a child digest verbatim
   should reach a child-bound attestation, and **that last link is inference from the measured code
   path, not a measurement**. The first real release is the measurement. **A failure reopens §6.3
   as a pipeline fix, never a documentation fix.** This is stated **nowhere** in operator prose
   (§15.4).
6. **The GHCR visibility default** (§17.1). Delete the runbook step if the first push disproves it.

**Worth knowing, and on no critical path:**

7. **Whether BuildKit runs the shared `builder` stage twice or four times.** Wall-clock only. It
   changes no artifact.
8. **Whether `imagetools create` reproduces the source index digest.** §4.4's platform-digest gate
   removes this from the critical path.
9. **The prober compiles four times.** The `Dockerfile`'s `for a in amd64 arm64` loop builds both
   probers inside every builder instance, and two target platforms make two of the four
   compilations waste. The builder stage's `ENV` line puts a different `GOARCH` in the parent layer, so BuildKit cannot share it. The fix direction is to
   hoist the loop into a `$BUILDPLATFORM` stage. **It changes no shipped artifact and no trust
   property**, and the job runs a few times a year. A later chore ticket may take it.
10. **Whether `SHA256SUMS` covers the §5.3 Trivy asset set.** §9.2's design intent says the
    checksum list grows with the asset list, and no closed ticket states it for the Trivy set
    specifically. The implementation ticket settles it, and §5.3 owns those file names.
