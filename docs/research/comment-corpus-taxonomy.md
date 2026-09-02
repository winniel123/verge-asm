# Comment corpus taxonomy

Wayfinder ticket [#1055](https://github.com/winniel123/verge-asm/issues/1055), map
[#1054](https://github.com/winniel123/verge-asm/issues/1054). Survey run **2026-09-01** against
`main` at `f6163e4`. The map measured its own baseline at `5a06c3a`, four commits earlier.
Section 2 reconciles the two.

This survey decides nothing. It produces the fact base that tickets #1056 through #1062 need.
Every count below is a point-in-time fact and will drift as the tree changes.

**Method.** A purpose-built lexer read every tracked file with an in-scope extension. The lexer
tracks string and raw-string state, so a `//` inside a Go string never counts as a comment. It
groups adjacent own-line comments into one **block**, and it classifies each block by the
precedence rules in the appendix. A **comment line** is a source line that any comment touches.
A block-comment interior line therefore counts. Section 2 shows why that choice matters.

**Scope.** Tracked files with these extensions: `.go`, `.sql`, `.mjs`, `.ts`, `.jsx`, `.html`,
`.tmpl`, `.css`, `.astro`. Ruling 3 names every one of these except `.astro`. Section 2 reports
`.astro` as an unlisted surface. Ruling 10 excludes `internal/db`. That exclusion removes 44
files and 3,161 comment lines. Every generated Go file in the tree sits in `internal/db`. So the
exclusion and the "generated" test agree exactly.

---

## 1. Headline

The in-scope, non-generated corpus holds **36,572 comment lines** in
**10,667 blocks** across **898 files**. Of 444 non-generated Go files, 440 carry a comment.

| Surface | Files | Comment lines | Source lines | Comment density |
| --- | ---: | ---: | ---: | ---: |
| `go-prod` | 241 | 20,620 | 63,322 | 32.6% |
| `go-test` | 203 | 7,726 | 49,214 | 15.7% |
| `sql` | 106 | 3,251 | 6,172 | 52.7% |
| `html` | 24 | 2,139 | 16,019 | 13.4% |
| `mjs` | 19 | 1,217 | 2,937 | 41.4% |
| `ts` | 116 | 559 | 2,404 | 23.3% |
| `tmpl` | 24 | 521 | 7,518 | 6.9% |
| `jsx` | 141 | 307 | 8,551 | 3.6% |
| `css` | 18 | 143 | 566 | 25.3% |
| `astro` | 6 | 86 | 316 | 27.2% |
| **Total** | **898** | **36,572** | **157,019** | **23.3%** |

Three facts govern the sweep more than any other:

1. **Half the corpus cites an ADR or an issue.** 49.9% of comment lines sit in a block
   naming `ADR-nnnn` or `#nnn`. Ruling 8 keeps those, capped at one line. So the sweep's
   dominant act is compression, not deletion.
2. **Go declaration-position comments are half the whole corpus.** They hold 19,223 lines,
   52.6% of every in-scope comment line. Ruling 4 deletes the exported ones, and no Go
   carve-out exists.
3. **Almost nothing is mechanically safe to delete without judgment.** The classes that a tool
   can delete on its own — dividers, commented-out code, TODO markers — hold
   308 lines together. That is 0.8% of the corpus.

---

## 2. Reconciliation with the map baseline

The map's baseline totals 32,645 lines. This survey totals 36,572. The two differ for one
reason. **The baseline counted lines that OPEN a comment. This survey counts every line a
comment occupies.** A block comment's interior lines are invisible to the baseline.

`.html` proves the mechanism exactly. The 24 `.html` files hold 26 lines matching `<!--` and 468
lines matching `//` or `/*`. Those sum to 494, the baseline's figure to the line. This survey
counts 2,139, because the design-system card files carry long block comments.

| Surface | Map baseline | This survey | Ratio |
| --- | ---: | ---: | ---: |
| `go-prod` | 19,917 | 20,620 | 1.04× |
| `go-test` | 6,980 | 7,726 | 1.11× |
| `sql` | 3,132 | 3,251 | 1.04× |
| `mjs` | 1,141 | 1,217 | 1.07× |
| `ts` | 565 | 559 | 0.99× |
| `html` | 494 | 2,139 | 4.33× |
| `jsx` | 238 | 307 | 1.29× |
| `tmpl` | 91 | 521 | 5.73× |
| `css` | 87 | 143 | 1.64× |
| `astro` | not measured | 86 | — |

Two findings follow.

**The baseline is not one method.** `.html` reconciles exactly under the opener rule. `.sql` does
not. 3,251 SQL lines contain `--`, and the baseline reports 3,132. `.css` reports 87 against 114
opener lines. So the per-surface figures came from per-surface greps of differing strictness.
Treat the baseline as an order-of-magnitude guide, not as a target to reconcile against.

**`.astro` is missing from ruling 3.** Six `.astro` files under `docs-site` hold 86 comment
lines. They are executable source and they are not Markdown, so ruling 3's stated principle
covers them. Ticket #1060 should either slice them in or rule them out explicitly.

The map's per-package figures track this survey with the same offset. The map reports 13,311
comment lines in `cmd/web` across 139 files, and this survey reports 13,952 across the same 139
files. The map reports 3,176 in `internal/queue`, and this survey reports 3,420. The map reports
782 in the worst file, `cmd/web/auth.go`, and this survey reports 797.

---

## 3. The classes

Each class below is mutually exclusive. A block takes the first class it matches, in the
appendix's precedence order. Section 7 reports the overlapping signals separately, because a
docstring that cites an ADR is one block with two properties.

### Protected machine directive — `directive`

**397 lines, 397 blocks, median 1 line per block.**

A line the toolchain reads. Deleting it changes the build or the generated code. SQL holds 384
of the 397 lines: 250 `-- name:` openers that sqlc reads, and 134 `+goose` markers that the
migration runner reads. Go holds 11 and `.jsx` holds 2. The lexer treats a directive line as its
own block, so a directive never absorbs the prose beneath it.

```
-- name: ClearRecoveryCodes :exec

//go:build !unix

//nolint:errcheck // no-op after a successful Commit
```

— `db/queries/recovery_code.sql:1, cmd/web/diskstat_other.go:1, cmd/web/seedfixtures.go:122`

### Package doc — `package-doc`

**768 lines, 57 blocks, median 12 lines per block.**

A comment immediately above a `package` clause. 57 blocks, and the median is 12 lines, so this is
the longest-per-block class in the corpus. These read as orientation for a whole package rather
than as a restatement of a signature. Ruling 4 names exported-identifier docstrings, and a
package clause declares no identifier, so ruling 4 does not decide this class. Ticket #1057 must.

```
Command web is the only listener in the deployment: it serves the
operator UI, applies database migrations on startup, and is the
container docker compose's healthcheck watches (ADR-0001, §4.2).
```

— `cmd/web/main.go:1`

### Citation — `citation`

**9,075 lines, 1,178 blocks, median 5 lines per block.**

A block that names an ADR or an issue and that sits in no declaration position. This is the
largest non-docstring class. SQL holds 2,108 lines of it. Those are the prose headers above
migrations and queries. `.html` holds 1,557 lines and `.tmpl` holds 460. Those are the
design-system provenance headers. Ruling 8 keeps one line of each production citation. So this
class converts rather than disappears.

```
The connect-outcome leaf: the daily hot Scan's TCP connect (never SYN),
deciding the reachability facet for each Service in scope. It is paced by
the §3.3 safety limiter, which never changes a verdict (ADR-0021).
```

— `cmd/prober/main.go:45`

### External-spec requirement — `external-spec`

**42 lines, 31 blocks, median 1 line per block.**

A block citing an RFC, IANA, X.509 or a comparable outside authority, and sitting in no
declaration position. The primary count is small because most such references sit inside a
docstring or a citation block. Section 7 reports the true reach: 144 blocks and 765 lines
mention an outside spec. CLAUDE.md names an external spec requirement as an explicit WHY
exception.

```
RFC3339 UTC
```

— `cmd/web/backup.go:178`

### WHY note — `why-note`

**840 lines, 289 blocks, median 2 lines per block.**

A block that states a reason, a hazard or a constraint, and that sits in no declaration position.
This is the shape CLAUDE.md exception 1 protects. The primary count is small for the same reason
the previous class is small. Section 7 reports 1,300 blocks and 10,173 lines carrying a WHY
marker across every class.

```
The onboarding checklist is a sequence of steps, so its redirects are deliberate
page MOVES rather than a return to the page acted from.
```

— `cmd/web/adr0130_contract_test.go:498`

### Conventional exported docstring — `docstring-exported-conventional`

**6,359 lines, 1,475 blocks, median 3 lines per block.**

A comment above an exported declaration, opening with that declaration's own name. This is the
Go convention that `go doc` renders. Ruling 4 deletes it. Production Go holds 5,291 lines and
test Go holds 1,068, because a `TestXxx` function is exported.

```
TestAddressCapHasNoUpperBound covers ADR-0127's load-bearing ruling: nothing gates a
value above the operator's own cap — a value beyond 2^32 (no IPv4 purpose) is
accepted, priced not gated.
```

— `cmd/web/addresscap_test.go:64`

### Non-conventional exported docstring — `docstring-exported-other`

**2,670 lines, 856 blocks, median 3 lines per block.**

A comment above an exported declaration that does NOT open with that name. Test files hold 2,613
of the 2,670 lines, against 57 in production Go. So production Go follows the naming convention
almost without exception. Test files almost never do. A test-file block in this class reads as an
assertion of behaviour rather than as a docstring. That is why it does not open with the function
name. Ticket #1057 should treat this class as test prose, not as a docstring.

```
A scope with no address above the threshold renders no row. A measured NOT-shared
edge is measured, and it is not evidence against the declaration.
```

— `cmd/web/addressscopecensus_test.go:87`

### Unexported docstring — `docstring-unexported`

**9,421 lines, 2,012 blocks, median 4 lines per block.**

A comment above an unexported declaration, a struct field, or a `const`/`var` group member. This
is the single largest class. `cmd/web` alone holds 6,203 lines of it. Ruling 4 names exported
identifiers, so this class needs its own ruling from ticket #1057.

```
addressScopeSharedEdges reads how many addresses inside each declared address scope
fan-out measured as shared, keyed by the masked scope. A scope with none is ABSENT
from the map rather than present at zero, which is what makes the caller's lookup
render a row only where the evidence exists.
```

— `cmd/web/addressscopecensus.go:45`

### Section divider — `section-divider`

**286 lines, 208 blocks, median 1 line per block.**

A banner or rule that separates regions of a file. The count includes box-drawing runs. An
earlier pass of this survey missed those. `cmd/web` holds 54 lines and `internal/scan` holds 19.
This class is mechanically decidable and carries no content. So ticket #1056 can take it whole.

```
--- class A: a refusal is a redirect, never a body ------------------------
```

— `cmd/web/adr0130_contract_test.go:302`

### Change narration (history) — `change-narration`

**90 lines, 33 blocks, median 2 lines per block.**

A block naming the code's own history: `superseded`, `deprecated`, `no longer`, `renamed`.
CLAUDE.md forbids exactly this shape. The corpus holds far less of it than the rule's prominence
suggests — 90 lines in 33 blocks.

```
the "no-TOTP viewer" ticket prose is
superseded by the frozen package; the TOTP-off branch is proven by unit test, not a
golden
```

— `cmd/web/devfixtures.go:281`

### Step narration — `step-narration`

**249 lines, 99 blocks, median 2 lines per block.**

A block using `now` or `was` to mark a step inside a test, not a change to the code. This class
exists because the obvious change-narration regex over-fires on test prose. Splitting it moved
249 lines out of the forbidden-narration count. Ticket #1059 must not let commentlint flag this
shape, because it is a false positive against a rule about the code's history.

```
The same /20 now declares — the raised cap took effect at declaration.
```

— `cmd/web/addresscap_test.go:39`

### Short label — `short-label`

**1,401 lines, 1,401 blocks, median 1 line per block.**

A one-line comment of six words or fewer that matches no other class. 1,401 blocks, every one a
single line. Test Go holds 694 lines and production Go holds 291. These are field labels and
test-table row labels. They carry no WHY, so ruling 4's logic deletes them, but no ruling names
them yet.

```
ports census
```

— `cmd/web/asset_test.go:31`

### Commented-out code — `commented-out-code`

**22 lines, 20 blocks, median 1 line per block.**

Effectively absent. 20 blocks matched, and reading them shows most are prose containing brace or
bracket literals rather than disabled code. Treat this class as empty. A candidate class the
ticket asked to test, and the corpus refuted it.

### TODO and FIXME markers — `todo`

**0 lines, 0 blocks, median 0 lines per block.**

**Zero.** No `TODO`, `FIXME`, `XXX` or `HACK` marker exists anywhere in the in-scope tree. A
plain grep over every in-scope extension returns no file. This candidate class is empty, so
ticket #1061's ADR-salvage protocol has no marker backlog to absorb, and commentlint needs no
rule for it.

---

## 4. Lines per class

| Class | Lines | Share | Blocks | Median block |
| --- | ---: | ---: | ---: | ---: |
| `directive` | 397 | 1.1% | 397 | 1 |
| `package-doc` | 768 | 2.1% | 57 | 12 |
| `citation` | 9,075 | 24.8% | 1,178 | 5 |
| `external-spec` | 42 | 0.1% | 31 | 1 |
| `why-note` | 840 | 2.3% | 289 | 2 |
| `docstring-exported-conventional` | 6,359 | 17.4% | 1,475 | 3 |
| `docstring-exported-other` | 2,670 | 7.3% | 856 | 3 |
| `docstring-unexported` | 9,421 | 25.8% | 2,012 | 4 |
| `section-divider` | 286 | 0.8% | 208 | 1 |
| `change-narration` | 90 | 0.2% | 33 | 2 |
| `step-narration` | 249 | 0.7% | 99 | 2 |
| `short-label` | 1,401 | 3.8% | 1,401 | 1 |
| `commented-out-code` | 22 | 0.1% | 20 | 1 |
| `todo` | 0 | 0.0% | 0 | 0 |
| `generated-header` | 0 | 0.0% | 0 | 0 |
| `prose-other` | 4,952 | 13.5% | 2,611 | 1 |
| **Total** | **36,572** | **100%** | **10,667** | |

---

## 5. Class against surface

| Class | `go-prod` | `go-test` | `sql` | `html` | `tmpl` | `mjs` | `ts` | `jsx` | `css` | `astro` |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `directive` | 11 | 0 | 384 | 0 | 0 | 0 | 0 | 2 | 0 | 0 |
| `package-doc` | 768 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| `citation` | 3,674 | 522 | 2,108 | 1,548 | 460 | 508 | 129 | 84 | 10 | 32 |
| `external-spec` | 25 | 13 | 0 | 0 | 0 | 0 | 3 | 1 | 0 | 0 |
| `why-note` | 186 | 220 | 117 | 107 | 22 | 159 | 19 | 10 | 0 | 0 |
| `docstring-exported-conventional` | 5,291 | 1,068 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| `docstring-exported-other` | 57 | 2,613 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| `docstring-unexported` | 8,461 | 960 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| `section-divider` | 94 | 45 | 0 | 146 | 0 | 0 | 0 | 1 | 0 | 0 |
| `change-narration` | 28 | 22 | 24 | 1 | 0 | 7 | 1 | 7 | 0 | 0 |
| `step-narration` | 89 | 81 | 53 | 13 | 0 | 6 | 1 | 6 | 0 | 0 |
| `short-label` | 291 | 694 | 1 | 73 | 4 | 70 | 171 | 7 | 90 | 0 |
| `commented-out-code` | 3 | 0 | 0 | 0 | 1 | 10 | 2 | 3 | 3 | 0 |
| `prose-other` | 1,642 | 1,491 | 564 | 251 | 34 | 457 | 233 | 186 | 40 | 54 |
| **Total** | **20,620** | **7,726** | **3,251** | **2,139** | **521** | **1,217** | **559** | **307** | **143** | **86** |

---

## 6. Class against package

The ten packages holding the most comment lines. Together they hold
26,040 lines, 71% of the corpus.

| Package | `directive` | `package-doc` | `citation` | `external-spec` | `why-note` | `doc-exported-conv` | `doc-exported-other` | `doc-unexported` | `section-divider` | `change-narr` | `step-narr` | `short-label` | `commented-out-code` | `prose-other` | Total |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `cmd/web` | 3 | 3 | 2,680 | 7 | 220 | 763 | 1,437 | 6,203 | 54 | 23 | 100 | 505 | 0 | 1,954 | **13,952** |
| `internal/queue` | 2 | 7 | 685 | 2 | 40 | 455 | 407 | 1,521 | 0 | 5 | 33 | 39 | 0 | 224 | **3,420** |
| `db/queries` | 250 | 0 | 1,071 | 0 | 44 | 0 | 0 | 0 | 0 | 20 | 28 | 0 | 0 | 322 | **1,735** |
| `db/migrations` | 135 | 4 | 1,037 | 0 | 73 | 41 | 0 | 4 | 0 | 4 | 25 | 1 | 0 | 251 | **1,575** |
| `internal/scan` | 1 | 158 | 69 | 6 | 15 | 706 | 131 | 215 | 19 | 2 | 5 | 42 | 0 | 107 | **1,476** |
| `internal/custody` | 0 | 22 | 33 | 3 | 26 | 694 | 200 | 238 | 0 | 0 | 6 | 69 | 0 | 73 | **1,364** |
| `internal/measure/connectoutcome` | 1 | 19 | 37 | 1 | 0 | 380 | 40 | 147 | 0 | 4 | 11 | 28 | 0 | 40 | **708** |
| `internal/message` | 0 | 26 | 28 | 0 | 0 | 343 | 61 | 135 | 0 | 0 | 0 | 47 | 0 | 55 | **695** |
| `internal/signal` | 0 | 15 | 134 | 2 | 23 | 258 | 32 | 53 | 0 | 0 | 2 | 14 | 0 | 61 | **594** |
| `design-system/templates` | 0 | 0 | 460 | 0 | 22 | 0 | 0 | 0 | 0 | 0 | 0 | 4 | 1 | 34 | **521** |

---

## 7. Overlapping signals

Section 4 assigns one class per block. These signals overlap, so they do not sum to the total.
Ticket #1057 needs this table more than section 4, because it measures how often a deletable
class carries protected content.

| Signal | Blocks | Lines | Share of corpus |
| --- | ---: | ---: | ---: |
| ADR citation (`ADR-nnnn`) | 1,351 | 11,476 | 31.4% |
| Issue citation (`#nnn`) | 1,730 | 13,592 | 37.2% |
| Either citation | 2,511 | 18,266 | 49.9% |
| External-spec reference | 144 | 765 | 2.1% |
| WHY marker | 1,300 | 10,173 | 27.8% |
| History marker (strict) | 183 | 1,697 | 4.6% |
| Narration marker (loose) | 659 | 5,125 | 14.0% |
| Bare URL | 7 | 81 | 0.2% |
| Go declaration position | 4,405 | 19,223 | 52.6% |
| Trailing, shares a line with code | 1,427 | 1,429 | 3.9% |

**The intersection that decides the sweep.** 4,405 blocks sit in a Go declaration position,
holding 19,223 lines. Of those, **1,729 blocks (11,021 lines) also carry a WHY marker, an ADR, an
issue or a spec reference.** Ruling 4 deletes the docstring. Ruling 8 and CLAUDE.md exception 1
keep the reason inside it. So roughly 39% of docstrings need a human or an agent to salvage a line
before deletion. This single number sizes the judgment residue, and it is the survey's most
load-bearing output.

**ADR reach.** Comments cite 99 distinct ADRs, and production Go comments cite 89.
134 ADR files sit in `docs/adr`. So 35 ADRs are cited by no in-scope comment. The map already
carries this as fog. This survey confirms the gap and sizes it.

---

## 8. Shapes that resisted classification

`prose-other` holds 4,952 lines in 2,611 blocks, 13.5% of the corpus. 1,472 of those blocks are a
single line. This is the residue ruling 6 hands to an agent. Reading it shows four recurring
shapes, none of which a regex separates from the others.

**1. Behaviour prose above a statement, not a declaration.** The commonest shape. It explains what
the next few lines achieve, in domain terms. It carries no ADR and no explicit reason word, so no
marker catches it, yet deleting it discards the only statement of intent.

```
The estate is read ONLY where there is something to withdraw. In the steady
state the query above answers empty and the derivation is never built.
```

— `internal/queue/withdrawal.go:99`

**2. Test-step prose.** Test files hold 1,491 residue lines against production Go's 1,710, and test
files hold far more residue blocks. A test-step comment names the state a step establishes. It is
neither a docstring nor a WHY note.

```
The messages themselves are still listed under the All filter.
```

— `cmd/web/inbox_test.go:82`

**3. TypeScript declaration-file field prose.** `.ts` residue is almost all `.d.ts` field
documentation in `design-system/components`. The Go declaration-position test does not apply to
TypeScript, so these fall through. Ticket #1057 should decide whether a `.d.ts` field comment is a
docstring under ruling 4.

```
Default var(--chart-1). Use --chart-2..4 for additional series
```

— `design-system/components/display/Sparkline.d.ts:7`

**4. Long provenance and design headers.** `.html`, `.tmpl` and `.mjs` residue is dominated by
multi-line headers that mix a spec reference, a box-drawing banner and prose. They resist a
single class because one block genuinely holds three shapes.

Two further limits apply to every number in this survey.

**The classifier is heuristic, not a parser.** It reads Go declarations with a line-based state
machine that trusts gofmt, not with `go/ast`. Ruling 13 already requires commentlint to use
`go/ast`. Ticket #1059 should expect its own counts to differ from this survey's by a small
margin, and should trust the parser over this document.

**Precedence hides content.** A docstring that cites an ADR counts once, under the docstring
class. Section 7 exists because of this, and section 7's numbers are the safe ones to plan
against.

---

## 9. `cmd/web`, per file

Ruling 7 needs `cmd/web` sub-sliced. 139 files hold 13,952 comment lines. No file holds zero.
62 production files hold 9,439 lines and 77 test files hold 4,513 lines.

The distribution is steep. The top 8 files hold 4,605 lines, a third of the package. A size cap
near 400 lines per ticket puts each of the top 8 files in a ticket of its own. The cumulative
column below lets ticket #1060 cut the tail at any cap it chooses.

| # | File | Comment lines | Cumulative | File lines |
| ---: | --- | ---: | ---: | ---: |
| 1 | `auth.go` | 797 | 797 | 2,654 |
| 2 | `devfixtures.go` | 764 | 1,561 | 3,112 |
| 3 | `handlers.go` | 658 | 2,219 | 1,145 |
| 4 | `settings.go` | 616 | 2,835 | 2,001 |
| 5 | `seeds.go` | 455 | 3,290 | 1,272 |
| 6 | `scans.go` | 440 | 3,730 | 1,436 |
| 7 | `subjects.go` | 439 | 4,169 | 1,674 |
| 8 | `signals.go` | 436 | 4,605 | 1,716 |
| 9 | `handlers_test.go` | 404 | 5,009 | 3,261 |
| 10 | `reports.go` | 376 | 5,385 | 1,264 |
| 11 | `inventory.go` | 247 | 5,632 | 789 |
| 12 | `reports_test.go` | 241 | 5,873 | 1,239 |
| 13 | `sources.go` | 226 | 6,099 | 680 |
| 14 | `cold.go` | 211 | 6,310 | 608 |
| 15 | `messages.go` | 206 | 6,516 | 622 |
| 16 | `drift.go` | 184 | 6,700 | 469 |
| 17 | `restore.go` | 182 | 6,882 | 650 |
| 18 | `adr0130_contract_test.go` | 177 | 7,059 | 706 |
| 19 | `reports_schedule.go` | 175 | 7,234 | 715 |
| 20 | `devfixtures_test.go` | 154 | 7,388 | 1,854 |
| 21 | `sources_test.go` | 149 | 7,537 | 849 |
| 22 | `integrations.go` | 147 | 7,684 | 554 |
| 23 | `sso.go` | 139 | 7,823 | 555 |
| 24 | `graph.go` | 136 | 7,959 | 443 |
| 25 | `custodycensus_test.go` | 135 | 8,094 | 536 |
| 26 | `custodycensus.go` | 131 | 8,225 | 251 |
| 27 | `subjects_test.go` | 131 | 8,356 | 623 |
| 28 | `backup.go` | 129 | 8,485 | 337 |
| 29 | `backurl.go` | 129 | 8,614 | 266 |
| 30 | `flash.go` | 125 | 8,739 | 255 |
| 31 | `proposals.go` | 121 | 8,860 | 392 |
| 32 | `signals_test.go` | 116 | 8,976 | 613 |
| 33 | `rundetail_test.go` | 111 | 9,087 | 691 |
| 34 | `drift_test.go` | 110 | 9,197 | 514 |
| 35 | `api_v1.go` | 105 | 9,302 | 398 |
| 36 | `scantrigger.go` | 105 | 9,407 | 308 |
| 37 | `driftfeed.go` | 102 | 9,509 | 352 |
| 38 | `inventory_test.go` | 102 | 9,611 | 446 |
| 39 | `search.go` | 102 | 9,713 | 348 |
| 40 | `exposure.go` | 96 | 9,809 | 278 |
| 41 | `deltas.go` | 94 | 9,903 | 342 |
| 42 | `scope_prg_test.go` | 87 | 9,990 | 373 |
| 43 | `annotations_test.go` | 85 | 10,075 | 441 |
| 44 | `sso_test.go` | 85 | 10,160 | 697 |
| 45 | `proposals_test.go` | 82 | 10,242 | 507 |
| 46 | `integrations_test.go` | 81 | 10,323 | 424 |
| 47 | `onboarding.go` | 78 | 10,401 | 297 |
| 48 | `search_test.go` | 78 | 10,479 | 309 |
| 49 | `scans_test.go` | 75 | 10,554 | 500 |
| 50 | `rawoutput.go` | 74 | 10,628 | 364 |
| 51 | `reports_export.go` | 74 | 10,702 | 271 |
| 52 | `backurl_test.go` | 73 | 10,775 | 487 |
| 53 | `asset_test.go` | 72 | 10,847 | 280 |
| 54 | `progress.go` | 72 | 10,919 | 225 |
| 55 | `profile_test.go` | 71 | 10,990 | 475 |
| 56 | `settings_test.go` | 70 | 11,060 | 545 |
| 57 | `api_auth.go` | 69 | 11,129 | 170 |
| 58 | `seedfixtures.go` | 64 | 11,193 | 175 |
| 59 | `settings_fixtures.go` | 64 | 11,257 | 643 |
| 60 | `auth_test.go` | 63 | 11,320 | 529 |
| 61 | `chrome.go` | 63 | 11,383 | 305 |
| 62 | `main.go` | 62 | 11,445 | 277 |
| 63 | `ratelimit.go` | 62 | 11,507 | 172 |
| 64 | `graph_test.go` | 61 | 11,568 | 281 |
| 65 | `hardening_test.go` | 61 | 11,629 | 359 |
| 66 | `settings_prg_sso_test.go` | 61 | 11,690 | 336 |
| 67 | `rawoutput_test.go` | 59 | 11,749 | 319 |
| 68 | `settings_prg_test.go` | 59 | 11,808 | 296 |
| 69 | `errors.go` | 58 | 11,866 | 162 |
| 70 | `scope_withdrawal_preview_test.go` | 58 | 11,924 | 362 |
| 71 | `scope_bulk_test.go` | 56 | 11,980 | 337 |
| 72 | `addressscopecensus.go` | 52 | 12,032 | 105 |
| 73 | `probers.go` | 52 | 12,084 | 141 |
| 74 | `settings_sso.go` | 52 | 12,136 | 267 |
| 75 | `signin_test.go` | 52 | 12,188 | 441 |
| 76 | `annotations.go` | 49 | 12,237 | 136 |
| 77 | `vantageclass.go` | 48 | 12,285 | 193 |
| 78 | `restore_test.go` | 47 | 12,332 | 298 |
| 79 | `backup_test.go` | 45 | 12,377 | 300 |
| 80 | `cold_coverage_test.go` | 45 | 12,422 | 242 |
| 81 | `exclusions.go` | 45 | 12,467 | 190 |
| 82 | `vergecore.go` | 45 | 12,512 | 107 |
| 83 | `clientip.go` | 43 | 12,555 | 125 |
| 84 | `shell.go` | 43 | 12,598 | 154 |
| 85 | `messages_test.go` | 42 | 12,640 | 262 |
| 86 | `scantrigger_test.go` | 42 | 12,682 | 283 |
| 87 | `profile_sessions_test.go` | 41 | 12,723 | 232 |
| 88 | `templates_shell.go` | 40 | 12,763 | 66 |
| 89 | `addresscap_test.go` | 38 | 12,801 | 184 |
| 90 | `inbox_test.go` | 38 | 12,839 | 245 |
| 91 | `seeds_test.go` | 38 | 12,877 | 223 |
| 92 | `integrations_channel_test.go` | 37 | 12,914 | 292 |
| 93 | `profile_fixtures_test.go` | 37 | 12,951 | 188 |
| 94 | `progress_test.go` | 37 | 12,988 | 231 |
| 95 | `reportdelivery_test.go` | 37 | 13,025 | 179 |
| 96 | `subjectdetail_test.go` | 37 | 13,062 | 145 |
| 97 | `clientip_test.go` | 35 | 13,097 | 208 |
| 98 | `addressscopecensus_test.go` | 34 | 13,131 | 212 |
| 99 | `api_auth_test.go` | 33 | 13,164 | 248 |
| 100 | `credflow_sessions_test.go` | 32 | 13,196 | 161 |
| 101 | `settings_sessions_test.go` | 31 | 13,227 | 206 |
| 102 | `signals_cert_test.go` | 31 | 13,258 | 247 |
| 103 | `session_test.go` | 30 | 13,288 | 206 |
| 104 | `onboarding_test.go` | 29 | 13,317 | 195 |
| 105 | `scans_stop_terminate_test.go` | 29 | 13,346 | 215 |
| 106 | `vantageclass_test.go` | 29 | 13,375 | 250 |
| 107 | `dashboard_test.go` | 27 | 13,402 | 99 |
| 108 | `exposure_test.go` | 27 | 13,429 | 165 |
| 109 | `probers_test.go` | 26 | 13,455 | 232 |
| 110 | `polish_test.go` | 24 | 13,479 | 89 |
| 111 | `reportartifact_test.go` | 23 | 13,502 | 134 |
| 112 | `templates_inbox.go` | 23 | 13,525 | 33 |
| 113 | `deltas_test.go` | 22 | 13,547 | 145 |
| 114 | `custody_test.go` | 21 | 13,568 | 167 |
| 115 | `first_run_test.go` | 21 | 13,589 | 123 |
| 116 | `inventory_fixture_test.go` | 21 | 13,610 | 199 |
| 117 | `templates_inventory.go` | 21 | 13,631 | 53 |
| 118 | `zone_test.go` | 21 | 13,652 | 189 |
| 119 | `channels_sendtest.go` | 20 | 13,672 | 83 |
| 120 | `custody.go` | 19 | 13,691 | 44 |
| 121 | `declaration_confirm_test.go` | 19 | 13,710 | 68 |
| 122 | `scans_history_window_test.go` | 19 | 13,729 | 139 |
| 123 | `cold_test.go` | 18 | 13,747 | 148 |
| 124 | `ratelimit_test.go` | 18 | 13,765 | 104 |
| 125 | `chrome_scanpill_test.go` | 17 | 13,782 | 83 |
| 126 | `api_v1_test.go` | 16 | 13,798 | 98 |
| 127 | `templates_reportartifact.go` | 16 | 13,814 | 26 |
| 128 | `error_test.go` | 15 | 13,829 | 103 |
| 129 | `settings_api_test.go` | 15 | 13,844 | 100 |
| 130 | `vergecore_test.go` | 15 | 13,859 | 127 |
| 131 | `templates_error.go` | 14 | 13,873 | 24 |
| 132 | `templates_settings.go` | 14 | 13,887 | 24 |
| 133 | `channels_sendtest_test.go` | 13 | 13,900 | 117 |
| 134 | `search_fixtures_test.go` | 12 | 13,912 | 69 |
| 135 | `exclusions_test.go` | 10 | 13,922 | 168 |
| 136 | `templates_reports.go` | 10 | 13,932 | 20 |
| 137 | `logutil.go` | 9 | 13,941 | 17 |
| 138 | `diskstat_unix.go` | 7 | 13,948 | 25 |
| 139 | `diskstat_other.go` | 4 | 13,952 | 11 |

### Whole-tree file distribution

| Comment lines in file | Files | Lines held |
| --- | ---: | ---: |
| 0 | 41 | 0 |
| 1–24 | 484 | 4,407 |
| 25–99 | 276 | 13,825 |
| 100–249 | 84 | 12,117 |
| 250–499 | 9 | 3,385 |
| 500+ | 4 | 2,835 |

Ruling 7 slices per package. This table shows why a per-package slice alone will not hold. 13
files carry 250 lines or more. 4 of those carry 500 or more. A per-file cap has to sit alongside
the per-package rule.

---

## 10. What the deciding tickets should carry forward

This survey decides nothing. These are the facts each open ticket depends on.

| Ticket | Fact it needs |
| --- | --- |
| #1056, safe classes | The mechanically-safe classes hold 308 lines total: `section-divider` 286 and `commented-out-code` 22. `todo` is empty. A mechanical pass alone clears under 1% of the corpus. |
| #1057, WHY rubric | 1,729 of 4,405 declaration-position blocks carry a reason worth salvaging. `prose-other` adds 2,611 blocks. The rubric must also rule on `package-doc` (57 blocks), `docstring-unexported` (2,012 blocks) and `.d.ts` field prose. |
| #1058, equivalence check | `.tmpl` and `.html` comments sit inside significant whitespace, and both surfaces are block-comment dominated. `.html` holds 2,139 lines against the baseline's 494, so the check has four times more to prove than the map assumed. |
| #1059, commentlint | Only `directive`, `section-divider` and Go declaration position are mechanically decidable. `step-narration` (249 lines) is the false-positive trap for any change-narration rule. |
| #1060, slicing | Section 9's cumulative column. `.astro` is unruled. 13 files tree-wide carry 250 or more comment lines. |
| #1061, ADR salvage | 35 of 134 ADRs are cited by no in-scope comment. No TODO marker exists, so there is no marker backlog. |
| #1062, CLAUDE.md wording | The rule must name `package-doc`, `docstring-unexported`, `short-label` and test-file prose. Ruling 4 names only exported identifiers, and those four classes hold 14,260 lines between them. |

---

## Appendix. Classification rules

A block takes the first class it matches. This order puts machine-protected classes first, then
structural position, then content markers.

| # | Class | Decisive test |
| ---: | --- | --- |
| 1 | `generated-header` | Body matches `Code generated .*DO NOT EDIT`. |
| 2 | `directive` | Go: `//go:`, `// +build`, `//nolint`, `//lint:`, `//revive:`. SQL: `-- name:`, `-- +goose`. JS and TS: `eslint`, `@ts-check`, `@ts-expect-error`, `prettier-ignore`, `c8 ignore`, `@jsx`. CSS: `stylelint-`, `postcss-`. A directive line forms its own block. |
| 3 | `todo` | First line opens with `TODO`, `FIXME`, `XXX`, `HACK` or `BUG`. |
| 4 | `commented-out-code` | 60% or more of non-empty lines match a code shape, AND under half the words are lowercase words. |
| 5 | `section-divider` | 60% or more of non-empty lines are a rule or a banner: a run of 4 or more of `-=*_~#+`, or a box-drawing run. |
| 6 | `package-doc` | Own-line block whose next line is a `package` clause. |
| 7 | `docstring-exported-conventional` | Own-line block whose next line declares an exported Go identifier, and whose first word is that identifier. |
| 8 | `docstring-exported-other` | Same position, exported identifier, first word is not that identifier. |
| 9 | `docstring-unexported` | Same position, unexported identifier, struct field, or `const`/`var` group member. |
| 10 | `citation` | Body matches `ADR-nnnn` or `#nnn`. |
| 11 | `external-spec` | Body matches `RFC`, `IANA`, `X.509`, `BCP`, `NIST`, `PKIX` or `ISO nnnn`. |
| 12 | `why-note` | Body matches a reason or hazard word: `because`, `otherwise`, `so that`, `avoid`, `workaround`, `race`, `panics`, `deliberately`, and comparable terms. |
| 13 | `change-narration` | Body matches a history word: `no longer`, `previously`, `renamed`, `superseded`, `deprecated`, `as of nnnn`, and comparable terms. |
| 14 | `step-narration` | Body matches the loose narration set (`now`, `was`) but not the strict history set. |
| 15 | `short-label` | One line, six words or fewer. |
| 16 | `prose-other` | Everything else. |

**Go declaration position.** A line-based state machine finds it, not `go/ast`. The machine
trusts two gofmt guarantees. A top-level declaration starts at column 0. Its closing `}` or `)`
also sits at column 0. Inside a function body no line counts as a declaration. Inside a
`type ... struct {`, `const (` or `var (` group, a single-tab-indented identifier counts.

**Comment extraction** uses a per-surface lexer that tracks string state. Go raw strings never
yield a false comment. Nor do JS template literals or SQL quoted literals. Single-quoted and
double-quoted strings end at a newline, which bounds any mis-lex to one line. Extraction for
`.html` and `.tmpl` handles `<!-- -->`, `{{/* */}}`, and the `<script>` and `<style>` regions
inside them. Extraction for `.astro` splits the frontmatter fence from the markup body.
