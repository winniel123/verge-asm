# The staticcheck backlog this tree carries, and the promotion question it raises

- **Status:** Measured — 2026-09-07
- **Ticket:** [#1568 CI runs no unused-code pass, so a thirteen-declaration dead cluster survived a whole migration](https://github.com/winniel123/verge-asm/issues/1568)
- **Raised by:** [PR #1548](https://github.com/winniel123/verge-asm/pull/1548), which deleted the cluster
- **Tree under test:** commit `baa3762`
- **Tool:** staticcheck 2026.2.1 (module version `v0.8.1`), default check set, no `staticcheck.conf`

CI now runs staticcheck as an advisory job in [`.github/workflows/ci.yml`](../../.github/workflows/ci.yml).
The job gates no merge and it exits 0 on any finding.
This note records the size of the backlog before anyone argues about promotion.

---

## 1. What ran, and why this version

The command:

```sh
go run honnef.co/go/tools/cmd/staticcheck@v0.8.1 ./...
```

staticcheck 2026.2.1 names `go 1.26.0` in its own `go.mod`, so it reads the language
version [`.go-version`](../../.go-version) pins.
Every earlier release predates Go 1.26.
On 2026-09-07 the module proxy holds no newer release.
The CI job installs the same exact version.

The run kept the default check set.
The `-tests` flag kept its default of true, so a declaration only a test reaches counts as reached.

## 2. The raw counts

| Check | Findings | What the check reports |
| --- | --- | --- |
| S1016 | 14 | A struct literal that a type conversion replaces |
| U1000 | 8 | A declaration nothing reaches |
| SA1019 | 5 | A deprecated standard-library symbol |
| SA4000 | 2 | Identical expressions across a `!=` |
| SA4006 | 1 | A value nothing reads |
| **Total** | **30** | |

All eight U1000 findings sit in `cmd/web`, which is package `main`.

A second run set `-tests=false`, which drops every test file as a root.
That run reports 19 U1000 findings rather than 8.
The extra 11 declarations serve a test and nothing else.
They are not dead, and this note does not list them.

## 3. The U1000 findings, classified

### 3.1 Probable real — all eight

| Declaration | File | Line | Ground |
| --- | --- | --- | --- |
| `errRestoreSchema` | `cmd/web/restore.go` | 60 | Three sibling sentinels in the same `var` block reach a `return`. This one reaches none. The schema-mismatch branch redirects with the literal `"schema"` instead. |
| `seedCreateError` | `cmd/web/seeds.go` | 616 | A wrapper over `isUniqueViolation`. Nine sites call `isUniqueViolation` directly and no site calls the wrapper. |
| `(*server).isNameSeed` | `cmd/web/seeds.go` | 855 | A method under an unexported name. No `cmd/web` interface declares that name and no call site uses it. |
| `paletteAsset` | `cmd/web/shell.go` | 61 | Only `currentAssets` builds this struct, and nothing calls `currentAssets`. The shell palette runs on `paletteGroup` in `cmd/web/chrome.go` instead. |
| `(*server).currentAssets` | `cmd/web/shell.go` | 66 | `auth.go` feeds the shell's `PaletteGroups` from `paletteGroupsProd`. That successor never calls this one. |
| `subjectPageData` | `cmd/web/subjects.go` | 134 | The name page renders `assetPageData` through a `map[string]any`. Its two sibling structs still render. This third one does not. |
| `nameCitationHop` | `cmd/web/subjects.go` | 582 | Its one caller is `buildCitation`, and nothing calls `buildCitation`. |
| `(*server).buildCitation` | `cmd/web/subjects.go` | 628 | `assetProvenance` replaced it. The successor still calls `terminatingNameSeed`, which is why that helper survives the sweep. |

Six of the eight form two clusters, and each cluster reads as a superseded
predecessor that the successor left behind.
One cluster is `paletteAsset` with `currentAssets`.
The other is `subjectPageData` with `nameCitationHop` and `buildCitation`.
Both match the shape [PR #1548](https://github.com/winniel123/verge-asm/pull/1548)
deleted from [`internal/message/render.go`](../../internal/message/render.go).

### 3.2 Probable false positive — none

[#1568](https://github.com/winniel123/verge-asm/issues/1568) names three vectors that
make U1000 unreliable, and this repo uses all three.
Each vector fails against these eight names.

- **Template lookup.** `html/template` reaches an exported method or an exported field
  only. All eight names start lowercase, so no template under
  [`design-system/templates/`](../../design-system/templates) can name one. The single
  exported surface among them is the `Key` and `Href` pair on `paletteAsset`, and only
  the dead `currentAssets` ever builds that struct.
- **`embed`.** Six `go:embed` directives exist in the tree and none sits in `cmd/web`.
  A directive names a file path, never a Go declaration.
- **Reflection.** `cmd/web` mentions `reflect` in test files only. Every use there
  compares a value of some other type.

A fourth vector deserves a note. The tool reads one build configuration, so a
declaration that only a build-tagged file reaches would look dead.
`cmd/web` holds one such pair, `diskstat_unix.go` and `diskstat_other.go`.
A repo-wide grep for each of the eight names returns the declaration and nothing else,
in either file of that pair.

### 3.3 What this classification is not

This is a spot-check, and it proves nothing.
The method was a repo-wide grep for each name, plus a read of the declaration and of the
code that replaced it.
A grep misses a name that some string builds at run time.
Nothing in `cmd/web` builds a Go identifier that way today, and no reader should treat
that as a guarantee for tomorrow.

Treat the table in §3.1 as eight candidates for a deletion ticket, not as eight
authorised deletions.
Clearing the backlog is separate work and belongs on its own tickets.

## 4. The other 22 findings

This note does not classify them, because [#1568](https://github.com/winniel123/verge-asm/issues/1568)
asks about U1000 only. The shape of the rest:

- **S1016, 14 findings.** Nearly all pair two sqlc row types with the same field set,
  such as `ListDispatchProgressRow` against `ListActiveDispatchProgressRow`. The
  suggested conversion couples two generated types that `sqlc generate` may separate
  again, so a maintainer should judge each one rather than accept the fix.
- **SA1019, 5 findings.** Deprecated standard-library symbols. Three concern
  `crypto/dsa`, which the certificate work reads on purpose. Two concern the `X` and `Y`
  fields on an ECDSA public key, deprecated in Go 1.26.
- **SA4000, 2 findings.** Identical expressions across a `!=`, both in a measure leaf
  test. These deserve a read, because a comparison against itself usually means a typo.
- **SA4006, 1 finding.** A `want` value that nothing reads, in `internal/scan`.

## 5. Open decision for the maintainer: promotion

> **RESOLVED, 2026-09-07.** The maintainer accepted the recommendation below.
> The §3.1 backlog is empty, and the job gates on U1000.
>
> - [PR #1628](https://github.com/winniel123/verge-asm/pull/1628) deleted all eight
>   declarations in §3.1. staticcheck reports zero U1000 rows on `main`.
> - The `staticcheck` job stops exiting 0 and fails on any U1000 row.
> - The gate is narrower than the report. **No `staticcheck.conf` was added.** The
>   default check set still runs and every finding still reaches the job summary,
>   so the §4 findings stay visible without blocking a merge. This reaches the
>   recommendation's goal with one job rather than two.
> - Registering `staticcheck` on `main`'s ruleset stays the maintainer's act. Until
>   they perform it the job fails a pull request's checks list without blocking its
>   merge.
>
> §3 and §4's counts are the 2026-09-07 measurement and are left unedited. §3.1's
> eight rows no longer exist in the tree.

**This session did not decide this, and it holds no authority to.** Promotion changes
what blocks a merge on `main`, and `main`'s ruleset is a repository setting.

**The recommendation.** Promote U1000 alone, and leave the other checks advisory.
U1000 is the check that answers the ticket, its backlog is eight rows rather than 30,
and every row here reads as real. A `staticcheck.conf` holding `checks = ["U1000"]` for
a gating job keeps the advisory job's wider report intact.

**The schedule.** Two steps, in this order.

1. Clear the eight rows in §3.1 through deletion tickets, one cluster per ticket.
2. Once staticcheck reports zero U1000 findings, flip the job and register the check.

**What must hold first.** All of the following, before the flip:

- The `staticcheck` job reports zero U1000 findings on `main`.
- The pinned version stays pinned. An unpinned tool that gains a check overnight would
  break `main` with no commit to blame.
- The maintainer accepts an 8th required status check on the ruleset. Nothing in a pull
  request can register one, so this act is theirs alone.
- The job stops exiting 0. That edit is one line in
  [`.github/workflows/ci.yml`](../../.github/workflows/ci.yml).

**The cost of leaving it advisory.** An advisory check can go red, or in this case stay
quietly green, and nobody notices. The `compose` job in the same file carries the same
known cost. The job summary exists to reduce it, and it does not remove it.
