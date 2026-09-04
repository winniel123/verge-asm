# Comment policy

- **Status:** Accepted — spec content for [#1054](https://github.com/winniel123/verge-asm/issues/1054)
- **Governs:** the `## Comments` section of [`CLAUDE.md`](../../CLAUDE.md)
- **Wayfinder map:** [Comment policy conformance audit (#1054)](https://github.com/winniel123/verge-asm/issues/1054)
- **Ticket:** [#1063 Write `docs/spec/comment-policy.md`](https://github.com/winniel123/verge-asm/issues/1063)

This document specifies the comment policy for verge-asm. It fixes the boundaries of the
`CLAUDE.md` no-comments rule. It specifies the `commentlint` tool, the comment-stripping
equivalence check, and the order of the sweep across the tree.

This document is a SPEC only. It cleans no file. A downstream `/to-tickets` and implement effort
runs the sweep. This document itself follows STE-flavored mode (see the
[documentation style standard](documentation-style-standard.md) §2.1).

Eight tickets on map #1054 settled the content below. Two of them produced measured research
reports:

- [`docs/research/comment-corpus-taxonomy.md`](../research/comment-corpus-taxonomy.md) surveyed the
  corpus and named its classes.
- [`docs/research/comment-equivalence-check.md`](../research/comment-equivalence-check.md) proved
  the equivalence check per surface.

This SPEC restates the taxonomy normatively. It cites the survey for counts only. Where the survey
and a parser disagree, the parser wins. The survey is a point-in-time fact base. It is never the
tool's contract.

---

## 1. Purpose and scope

### 1.1 What this SPEC decides

The `CLAUDE.md` rule forbids code comments and names two exceptions. It does not say how to apply
those exceptions to about 36,572 comment lines already in the tree. This SPEC closes that gap in
nine parts:

| Part | Section |
| --- | --- |
| The taxonomy and the protected directives | §2 |
| The mechanical pass | §3 |
| The WHY rubric | §4 |
| The equivalence check | §5 |
| The `commentlint` tool | §6 |
| The sweep order and the cut list | §7 |
| The ADR-gap protocol | §8 |
| The `CLAUDE.md` replacement text | §9 |
| Worked examples | Appendix A |

### 1.2 The measured corpus

The in-scope, non-generated corpus holds **36,572 comment lines** in **10,667 blocks** across
**898 files**. It falls into 14 non-empty classes. The survey §4 holds the per-class counts.

Three measured facts shape every decision below.

1. **The sweep compresses more than it deletes.** 49.9% of comment lines sit in a block that names
   an ADR or an issue.
2. **The judgment residue is large.** 4,405 blocks sit in a Go declaration position. 1,729 of them
   (39%, 11,021 lines) also carry a marker worth salvaging.
3. **A zero-risk mechanical pass clears almost nothing.** `section-divider` plus
   `commented-out-code` is 308 lines, which is 0.8% of the corpus. §3.1 states the bar that lifts
   the pass off that floor.

### 1.3 In-scope surfaces

The sweep covers **8 surfaces**. `commentlint verify` supports **9**. The ninth, `.html`, is
lexable but has no sound equivalence check, so §1.4 excludes it.

| Surface | Files | In the sweep | `verify` |
| --- | ---: | --- | --- |
| Go, production and `_test.go` | 444 | Yes | Yes |
| `.sql` (`db/queries` only) | 106 | Yes | Yes |
| `.mjs`, `.ts` and `.d.ts` | 135 | Yes | Yes |
| `.jsx` | 141 | Yes | Yes |
| `.tmpl` | 24 | Yes | Yes |
| `.css` | 18 | Yes | Yes |
| `.html` | 24 | No | Fails closed |
| `.astro` | 6 | No | Fails closed |

### 1.4 Out of scope

Each exclusion has its own reason. A reader who applies one reason to another exclusion misreads
the boundary.

| Excluded | Reason |
| --- | --- |
| Markdown documentation | `doclint` and the [documentation style standard](documentation-style-standard.md) already govern it. |
| CI workflow YAML | A workflow file is configuration a human reads cold. `doclint.yml`'s own header is load-bearing. |
| `internal/db` | It is `sqlc`-generated. Any edit reverts on the next `sqlc generate`. |
| `db/migrations/*.sql` | A migration is history, not maintained source. §4.1 gate A asks whether a fact is recoverable from the code, and a migration has no current code to test against. 1,516 comment lines, 1,037 of them citations. |
| `prototypes/` | Disposable mock source. About 640 comment lines. It ships in no binary and it is not maintained. |
| `.html` | A comment's bytes are output bytes, so ruling 15 is unsatisfiable. 19 of 19 lexable files fail byte-exact comparison under both delete strategies. |
| `.astro` | It needs `@astrojs/compiler`, which the repo does not install. Installing it would cross the two build lanes `docs-site/package.json` keeps apart. |
| Authoring the ADRs the sweep surfaces | §8.10 triages the backlog and stops there. |
| Any product behaviour change | This effort deletes comments and adds a lint tool. Nothing else. |

Two qualifications on the `db/migrations` exclusion:

- `db/queries` stays **in** scope. It is live sqlc input and it tracks the current schema.
- The exclusion covers `db/migrations/*.sql` only. The 3 Go files under `db/migrations`, holding 59
  comment lines, stay in scope.

Neither `.html` nor `.astro` ships in the binary. `designfs.go` embeds only `templates/*.tmpl`,
`tokens/*.css` and `fixtures/*.json`. 19 of the 24 `.html` files sit under `prototypes/`, so the
`.html` exclusion and the `prototypes/` exclusion overlap heavily.

### 1.5 Terms

| Term | Definition |
| --- | --- |
| **Block** | A run of own-line comment lines with no blank line and no code between them, in one comment style. A `/* */` comment is one block. A protected directive line is its own block. |
| **Trailing comment** | A comment that shares a line with code. It is never part of an own-line block. |
| **Declaration position** | A block whose next line declares a Go identifier, a struct field, a `const` or `var` group member, or a `package` clause. |
| **Skeleton** | The token stream a file yields with every comment removed and every protected directive kept. §5.1 defines it. |
| **Residue** | The blocks the mechanical pass declines. An agent judges each one. |
| **Keeps ledger** | The table in a sweep PR body listing every surviving and rewritten comment. §4.9 defines it. |
| **ADR gap** | A decision stated only in a comment the sweep is about to delete. §8.1 defines it. |

---

## 2. The taxonomy

### 2.1 Precedence and the decisive tests

A block takes the **first** class it matches, in this order. The order puts machine-protected
classes first, then structural position, then content markers. This table is normative.

| # | Class | Decisive test |
| ---: | --- | --- |
| 1 | `generated-header` | Body matches `Code generated .*DO NOT EDIT`. |
| 2 | `directive` | The block matches a §2.3 pattern. A directive line forms its own block. |
| 3 | `todo` | First line opens with `TODO`, `FIXME`, `XXX`, `HACK` or `BUG`. |
| 4 | `commented-out-code` | The payload parses as Go **and** carries a code-only token. §6.6 states the detector. |
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

Two classes are empty in this tree. `todo` holds zero blocks. `commented-out-code` holds 22 lines,
and reading them shows most are prose containing bracket literals. Both rules still ship, because
`commentlint` is a ratchet against new comments.

**`step-narration` is a false-positive guard, not a target class.** It exists because the obvious
change-narration regex over-fires on test prose. §3.5 forbids `commentlint` from flagging it.

### 2.2 Go declaration position

`commentlint` finds declaration position with `go/ast`, never with a line-based state machine. The
survey used a state machine and records that it is heuristic. Expect a small difference between the
survey's counts and the tool's. Trust the parser.

### 2.3 The protected-directive patterns

Ruling 10 fixes the form as a pattern class rather than an exact list. These patterns are
normative. 484 lines depend on them. Deleting one changes the build, the generated code, or what a
required status check reports.

| Surface | Patterns |
| --- | --- |
| Go | `//go:`, `// +build`, `//nolint`, `//lint:`, `//revive:`, `// #nosec`, `//#nosec` |
| SQL | `-- name:`, `-- +goose` |
| JS and TS | `eslint`, `@ts-check`, `@ts-expect-error`, `prettier-ignore`, `c8 ignore`, `@jsx` |
| CSS | `stylelint-`, `postcss-` |

**To add a pattern, add it to this table.** A directive line forms its own block, so a directive
never absorbs the prose beneath it.

**A `#nosec` justification is prose, and prose wraps.** The own-line block that opens on the line
below a `#nosec` line is the **waiver tail**. `gosec` reads the whole comment group, so that tail is
part of the waiver. **§3.2 withholds a waiver tail from the delete pass under the `tool-marker`
signal.** Measured: without the rule, `strip` deleted two justification lines from
`cmd/web/backup.go` (#1274).

**The tail is withheld from the delete pass alone. Every ratchet rule still reads it.** A tail that
opens with `TODO` still reports `todo-marker`. A `#nosec` line is not a way to silence the ratchet
for the lines below it.

**This list is a floor, not a proven-complete list.** The 2026-09-01 measurement claimed completeness
and missed `#nosec`. `gosec` reads `#nosec`, and `gosec` is one of the 7 required status checks, so
deleting one changes what that check reports (#1274). Widen this table whenever you find a tool that
reads a comment. Never narrow it.

Four findings, measured 2026-09-01 and re-measured with the tool on 2026-09-03:

1. No cgo. The tree holds no `import "C"`, no `//export` and no `#cgo`. That removes the most
   dangerous comment class in Go.
2. No `//go:generate`.
3. SQL holds 387 of the 484 directive lines. Go holds 96: 81 `#nosec` blocks that span 85 lines,
   and 11 lines under the other five patterns. Five more Go blocks name `#nosec` in prose, and the
   tool reads none of them as a directive. A grep finds 1 line in `.jsx`, because the `.jsx` lexer
   needs `esbuild`. `.mjs`, `.ts`, `.css` and `.tmpl` hold none.
4. **No JSDoc tag is consumed by any tool in this repo.** A raw grep reports 13,533 `@param` and
   8,884 `@component`, and every one sits in untracked `docs-site/node_modules`. Tracked files hold
   zero. `docs-site` uses no typedoc and no ts-morph.

The `/** */` field comments in `design-system/components/*.d.ts` reach an editor through the
TypeScript language service. They reach no build step. §4.3 keeps them for the reader, not for a
toolchain.

---

## 3. The mechanical pass

### 3.1 The bar is screened risk

A class qualifies for the mechanical delete pass when **all three** hold:

1. A parser or a class-precedence regex identifies it structurally.
2. A ruling in this SPEC makes it deletable.
3. The block passes the §3.2 negative screen.

**The bar is screened risk, not zero risk.** Under a zero-risk bar the safe set is 308 lines, or
0.8% of the corpus. The screened bar admits marker-free Go declaration position and lifts the pass
to about **8,200 lines**. Two controls bound the residual risk. A human reviews every sweep PR
(ruling 6). The §5 equivalence check proves that no non-comment byte moved (ruling 15).

### 3.2 The negative screen

A block is withheld from the mechanical pass when it carries any of these signals.

| Signal | Test |
| --- | --- |
| Citation | `ADR-nnnn` or `#nnn` |
| External spec | `RFC`, `IANA`, `X.509`, `BCP`, `NIST`, `PKIX`, `ISO nnnn` |
| WHY marker | the reason and hazard word list, §2.1 rule 12 |
| Bare URL | an `http` or `https` URL with no citation beside it |
| Tool marker | a `Deprecated:` paragraph opener, a `#nosec` body, or a §2.3 waiver tail |

Three properties of the screen:

- **It is block-wide.** One signal anywhere in the block withholds every line of that block.
- **It is uniform.** It applies to every class in the delete set, `section-divider` included. A
  banner can carry a citation.
- **It is tuned loose.** The cost is asymmetric. A false keep costs agent time. A false delete loses
  a reason permanently. This list is the floor. A later revision may widen it. **A later revision
  must never narrow it.**

The waiver tail is the one signal a block's own body does not carry. §2.3 states why: `gosec` reads
the comment group, and the block is a smaller unit than the group.

Two signals are **not** screen signals. A strict history marker is a delete signal, because
`CLAUDE.md` forbids change narration. The loose narration set (`now`, `was`) is the survey's named
false-positive trap.

### 3.3 The unit is the whole block

The pass deletes a block whole or it leaves the block untouched. **It never edits inside a block.** A
five-line docstring whose third line carries a gotcha is withheld by the screen and reaches the
agent intact.

A partial block edit is a judgment call in a machine's clothes. Block atomicity is what makes the
diff reviewable by eye.

### 3.4 Trailing comments are out

1,427 blocks share a line with code. **The mechanical pass does not touch them.** They are the
highest-risk-per-line edits in the tree, they stress the §5 check hardest, and they are 3.9% of the
corpus. An agent judges all of them.

**The `short-label` overlap.** Most trailing comments are short, so `short-label` and the trailing
population overlap heavily. **The mechanical `short-label` delete reaches own-line blocks only.** A
trailing short label stays with the agent. Worked example 14 in Appendix A is this exact case. An
unstated overlap here is what a sweep agent resolves wrongly and silently.

### 3.5 Two sets: delete inside flag

"Safe to flag for a human" is a weaker bar than "safe to auto-delete", so the two sets differ.

**The delete set.** Four seed classes, Go only in v1:

1. `section-divider` (286 lines).
2. `commented-out-code` (22 lines).
3. Marker-free `docstring-exported-conventional` (6,359 lines before the screen).
4. Own-line `short-label` (1,401 lines).

**The delete set is a test plus a seed list, not a closed list.** Any class this SPEC rules deletable
enters the mechanical set when it passes the §3.1 test. §4.3 rules `docstring-unexported` deletable,
so a **marker-free** `docstring-unexported` block deletes mechanically. A marker-free block carries
nothing to salvage by construction, because the screen tests for exactly the signals a salvage step
would rescue. A marked block is withheld and reaches the agent for the §4.4 rewrite.

Two declaration-position classes stay out of the delete set:

- **`package-doc`** keeps at a cap (§4.8), so it never deletes mechanically, marker-free or not.
- **`docstring-exported-other`** is judged as test prose (§4.3), so an agent decides it.

**The flag set** is the delete set plus five ratchet rules:

1. Any own-line block in a Go declaration position, marker or not. Ruling 8 caps it at one line
   either way.
2. `short-label`, own-line or trailing.
3. Strict change narration.
4. Any `TODO`, `FIXME`, `XXX` or `HACK` marker. The tree holds zero today, so the ratchet holds that
   at zero.
5. A production citation block longer than one line.

**Two never-flag rules: `step-narration` and `prose-other`.** Ruling 12 forbids the tool from
guessing at intent, and both classes need intent to judge.

`commentlint` lints changed files only (ruling 11), so it is a ratchet against new comments, not a
sweep tool. A ratchet that catches only box-drawing banners catches nothing, because nobody writes
those by hand. That is why the flag set is wider than the delete set.

Flag rule 1 is **not** safe against the sweep's own output on its own. §4.5 moves every salvaged
line out of declaration position, but declaration position is wider than a package-scope
declaration, and an in-body `var`, `const` or `type` sits in it too. §4.5 carries the parser fact
and the three placements that clear the rule. Flag rule 1 is safe against the sweep's own output
only when the agent uses one of them.

### 3.6 The per-surface grid

The surfaces are different objects. `docstring-exported-conventional` has no meaning outside Go. Go
uses `go/ast`. Every other surface uses a regex.

**v1 deletes in Go only.** Every non-Go cell reads `agent`. A non-Go `section-divider` cell upgrades
to `mechanical` only after a measured equivalence check for that surface. The cost is small, because
the non-Go seed classes hold a few hundred lines.

| Class | Go | SQL | JS, TS, JSX | `.tmpl` | CSS |
| --- | --- | --- | --- | --- | --- |
| `directive` | protected | protected | protected | protected | protected |
| `section-divider` | mechanical | agent | agent | agent | agent |
| `commented-out-code` | mechanical | agent | agent | agent | agent |
| `short-label`, own-line | mechanical | agent | agent | agent | agent |
| `docstring-exported-conventional` | mechanical | n/a | n/a | n/a | n/a |
| `docstring-unexported` | mechanical when marker-free | n/a | n/a | n/a | n/a |
| `package-doc` | agent, capped | n/a | n/a | n/a | n/a |
| `docstring-exported-other` | agent | n/a | n/a | n/a | n/a |
| every other class | agent | agent | agent | agent | agent |

This restriction binds the **delete** pass. It does not bind the flag set. `commentlint lint` is
advisory, so it may flag on a surface the pass will not edit.

### 3.7 Test files are included, and sampled apart

`_test.go` files are in the mechanical pass. Test Go holds 1,068 of the 6,359
`docstring-exported-conventional` lines, because a `TestXxx` function is exported. Ruling 2 makes
"the ADR or issue this test pins" a WHY exception, and the screen already catches that shape as a
citation.

**The §3.9 gate samples production Go and test Go separately.** One shared sample would hide a
failure specific to one population.

### 3.8 Delete semantics and the residue manifest

**Delete semantics, Go.** The pass removes the block's own lines. It removes a blank line only where
the deletion would otherwise leave two consecutive blank lines. It then runs `gofmt`. The §5
equivalence check runs on the result.

**That rule is Go-shaped.** §6.5 holds the per-surface table. The `.tmpl` row differs and is already
measured.

**The residue manifest.** The pass writes a manifest for each package: file, line span, class, and
the screen signal that fired. It goes to a scratch path. **It is never committed.** Without it, every
sweep ticket repeats the classification.

The manifest and the keeps ledger are different objects. The manifest routes work and stays in
scratch. The ledger is review evidence and sits in the PR body.

### 3.9 The validation gate

Run this gate on §7.2 stage C, the pilot slice.

1. Sample 100 blocks that the screen admits for deletion.
2. A human reads them.
3. Accept the class at **2 or fewer load-bearing blocks**.
4. On a failure, widen the screen (§3.2) and re-sample. A single failure does not abandon the class.

Sample production Go and test Go separately (§3.7).

**The termination rule.** A class runs at most **three** sampling rounds. If round three fails, the
class leaves the delete set for v1, and every block in it becomes agent-judged. The class stays in
the flag set, because flagging is advisory.

Three rounds is the bound because each round costs a human 100 blocks of close reading. Three rounds
is 300 blocks, which already exceeds the keeps-ledger review of several sweep tickets. A class that
cannot reach 2 in 100 after two widenings is not mechanically decidable on this tree. The sweep does
not stall on the failure. Stage B removes less, and the residue grows.

---

## 4. The WHY rubric

**A sweep agent holds this section and Appendix A open while it works.** Every other section serves a
tool or a planner. This one serves the agent judging a block.

### 4.1 The test for WHY

A comment survives only when it passes **both** gates.

**Gate A — recoverability.** Could a competent reader recover this fact by reading the declaration,
its body, and its callers? If yes, the comment dies.

**Gate B — causality.** Does the lost fact name a cause **outside this code**? A decision, an
external constraint, a hazard, a cost, or a rejected alternative all qualify. Prose that explains
what the code achieves does not qualify, even where the code is dense.

Gate A alone over-keeps. Almost any prose adds some fact, so an agent applying gate A literally keeps
most of `prose-other` and most of the 11,021 declaration-position salvage lines. Gate B is what
separates a reason from a restatement.

**Gate A is per-block, so run it across the ticket as well.** A rule restated in four docstrings is
unrecoverable from each of those four declarations. It therefore passes gate A four times in
isolation, and four copies survive. Before you keep a line, search the ticket's files for the same
rule. **One block keeps the rule. Every other statement of it dies.** Keep it in the block the rule
is most about, not in the first block you read. #1169 reports cross-block duplication, not per-block
judgment, as the dominant deletion driver in that ticket.

**The cross-block test reaches `main`, not only the ticket.** In a late `cmd/web` file the dominant
deletion driver is a rule a merged sweep already left on `main`, not a sibling block inside the
ticket. 35 of #1209's 86 deletions collapsed that way, #1208's 14, #1207's 10, #1211's 10 and
#1210's 4. An
agent reading "across the ticket" literally keeps every one of them. Search the merged sweep output
for the rule before you keep it. Nothing is dropped for an owner to catch here, because the owner is
on `main` already. §7.4 requires the PR body's boundary table to hold the two as separate rows.

**The cross-block test compares the rule, not the surface claim.** Two survivors may state
opposite-sounding facts and both be true. `internal/scan`'s `tailReadableState` keeps "a retired log
may 404", and `AllLogs` keeps "a retired shard still answers a point-check" (#1189). The tail
refuses a cadence on a flaky log. Verification tries one point-check once. The rules differ, so both
blocks survive. An agent that compares the wording alone deletes one of them.

**The package doc may be the keeping block.** §4.8 caps the package doc at 3 lines and does not rule
whether the package doc may hold a cross-block rule. It may. A file-wide rule that names no symbol
belongs there. `internal/scan/zone.go`'s supply-instant rule was stated 4 times, and the package doc
was the right home (#1189). §4.8's 3-line cap still binds.

**Where the package doc is closed, the keeping block is a detached file-header block.** A split
package gives `main.go` to another ticket, so §4.8's route is unavailable and a file-wide rule that
names no symbol has no legal home. Put it above the first declaration, detached by a blank line
(§4.5 placement 3). #1204 authored one for 4 rules stated across 17 screen headers, and #1205 for a
file-wide admin-act rule. It is not always needed: #1206 found every file-wide rule a
statement-relative home. In #1207 the block had a second motive, because the text it replaced was
spliced onto `type dnsRecordValue` and placement 3's blank line is also the unweld.

**A protected directive may be the keeper.** A directive line may serve as the cross-block keeper.
§2.3 forbids the sweep to edit that line, so the rule then rests on untouchable text. A later sweep
can neither compress that line to the §4.4 form nor repair a citation inside it. #1182 kept
`internal/measure/connectoutcome/tls.go`'s `#nosec G402` waiver as the keeper of the
record-not-verify rule and deleted the rule's two other statements. #1194 did the same with
`internal/queue/worker.go:41`'s `#nosec G204` waiver, which states ADR-0001's job-spec-in contract.
**Choose a directive as the keeper only where its own text states the whole rule and carries no
citation.** Keep a §4.4 survivor beside the directive where the waiver states one part of the rule.
Do the same where the rule needs a citation a later sweep may have to repair.

**A cross-block keeper must be swept output or a document.** Never collapse a rule onto unswept
residue in another stage's surface. `db/queries/dispatch.sql` restates four of `cmd/web/scans.go`'s
rules and is stage-D2 residue a later ticket may delete, so #1208 kept a Go survivor for each of the
four rather than lean on it.

**Check for an orphaned document reference before you delete a block.** A block may be the only one
in the file that names the document its siblings cite by a bare `§n`. Check this before you delete
it. Then either keep the naming, or rewrite each surviving sibling to name the document itself. This
is a defect the sweep creates, not one it finds. `internal/scan/ctreliability.go` and
`cttail_test.go:234` left four `§` references pointing at nothing, and #1190 repaired both.
`transcript.go`, `cttail.go` and `ctverify.go` held 15 bare `§n` references with one naming block
each, and every survivor now names the document in full.

**Compare the deleted block's citations against the keeper's.** The check above asks whether the
deleted block names a document. It does not ask whether the deleted block holds the *better*
citation. #1206 deleted a field docstring as a duplicate and then found it carried `v1-spec §5.3`
for the anchor-slug rule, against the survivor's `(#286)`, which is T10 IA reconciliation and says
nothing about anchor slugs. #1205 repaired two survivor citations in a second pass, after both had
passed `lint`, `verify` and a naive citation check.

**The orphan check runs in both directions.** A document may point into the corpus, and deleting the
block breaks the document instead. `docs/guides/reports.md:68-69` sends an operator to "the handler
comment in `cmd/web/reports.go`", and #1210 deleted the comment it names (#1354). `docs/guides/` is
the likely home, because a guide is written for an operator being sent to the code. Grep `docs/` for
the file you are sweeping before you delete a block a document may name.

**The tiebreaker prompt.** When a comment sits on the edge, ask a third question. Could a reader who
did not know this fact make a change that is wrong, and that the test suite would not catch? **This
is a prompt, not a gate.** The tree holds 49,214 lines of test source and heavy contract tests. As a
hard gate it would delete genuine hazard notes whose hazard a test already pins.

### 4.2 The tie-break: when you cannot decide, delete

**When you cannot decide, delete.** This is an instruction, not a principle.

Two facts drive it:

1. **Nothing is lost permanently.** The tree is version-controlled, and `git log -S` recovers any
   deleted comment.
2. **A diff shows a reviewer what was deleted and never what was kept.** Ruling 6 puts a human on
   every sweep PR. Delete-by-default puts every judgment call in front of that human.
   Keep-by-default hides every one of them.

Scale confirms the choice. `prose-other` holds 4,952 lines. The declaration-position salvage holds
11,021 lines. A keep-by-default tie-break would make the sweep close to a no-op on the exact
population this rubric exists to decide.

§4.9 is the counterweight this rule obliges.

### 4.3 The class rulings

| Class | Lines | Ruling |
| --- | ---: | --- |
| `docstring-unexported` | 9,421 | **Delete**, with salvage under §4.1. |
| `docstring-exported-other` | 2,670 | **Not a docstring.** Judge it under §4.6. |
| `short-label` | 1,401 | **Delete mechanically**, own-line blocks only (§3.4). |
| `package-doc` | 768 | **Keep, capped.** See §4.8. |
| `.d.ts` field prose | ~230 | **Keep.** Carve-out. |

**`docstring-unexported`.** An unexported docstring has strictly less claim than an exported one.
`go doc` does not render it. Ruling 4's logic covers it a fortiori.

**`docstring-exported-other`.** 98% of it sits in test files. The survey reads it as behaviour
assertion. Judging it as a docstring misclassifies it.

**`short-label`.** Six words or fewer cannot carry a reason. The one risk is a bare `// ADR-0021`,
and the §3.2 screen already withholds a block that carries a citation.

**`.d.ts` field prose.** A `.d.ts` file has no implementation. Its field prose **is** the interface,
not a comment on code. Deleting it leaves the types alone and destroys the contract a caller reads.
§2.3 measured that no tool in this repo consumes a JSDoc tag, so this carve-out rests on the reader,
not on a toolchain.

**Both carve-outs bind the sweep only.** `package-doc` at a 3-line cap and `.d.ts` field prose are
exceptions a sweep agent applies to existing comments. §9 does not carry them into `CLAUDE.md`, and
§9.4 states why. This is a decision, not an oversight.

### 4.4 The rewrite rule

A surviving comment takes this form:

```
// <reason clause> (ADR-nnnn §x.y, #nnn)
```

Four constraints bind **each line**:

1. It states the constraint or the cause. It never states the behaviour.
2. It does not open with an identifier name.
3. It holds 25 words or fewer, which is the description cap the
   [documentation style standard](documentation-style-standard.md) already sets.
4. It holds 100 characters or fewer, which is where the tree already wraps. Go comment lines in `cmd`
   and `internal` reach 93 characters at the maximum. **Measure this after `gofmt -w`, not as you
   type it.** `gofmt` aligns a trailing comment to the widest field in its run, so a survivor that
   fits when written can overrun after formatting. `internal/auth/session.go`'s `Token` field
   reached 105 characters that way (#1166).

**A trailing survivor is squeezed from both sides.** §3.5 ratchet rule 2 flags a `short-label`
trailing or own-line, and §6.6 classifies a one-line block of 6 payload words or fewer as one.
Constraint 4 caps what a long declaration leaves of 100 characters after `gofmt`. Where no text
satisfies both, the reason moves out of declaration position (§4.5).

**A citation or a why-marker escapes ratchet rule 2.** §6.6 orders `citation`, `external-spec` and
`why-note` above `short-label`. A survivor in the §4.4 form with a citation therefore never
classifies as a short label, whatever its length. A bare reason clause with neither must clear six
words.

**The degenerate case.** Where a block's only content is "this exists because ADR-nnnn says so", and
no independent constraint is recoverable, the survivor is the bare citation: `// ADR-0021 §3.3`.
Half the corpus cites something, so most of the 18,266 cited lines land in this form.

**A `file:line` reference is not a citation.** A line number rots on the next edit to the target,
and nothing exists to repair it to that stays right. Name the symbol instead. `authorizedScope`
cited `worker.go:357`, which is now `func markRetried` (#1198). The rule reaches a template line
too: `cmd/web/scans.go`'s `StreamHref` block cited `rundetail.tmpl` "line ~151" and "~216" (#1208).
Name the `define` or the datum instead.

**Ruling 8 caps a citation, not a reason.** A block may hold **one line per independent reason**. No
block-level line cap applies. Worked example 13 forces this: `cmd/web/devfixtures.go:281` carries two
unrelated constraints, and neither restates the other. A hard one-line cap would delete one of them
to satisfy a format, which is the error §4.7 rejects. The discipline that prevents bloat is "every
line passes §4.1", not a number.

**The independence test.** Would a reader who acts on either reason alone do the same thing? If yes,
the two are one reason, and you merge them. `CoversAddressScope` carried "one binding" and "do not
add a second un-narrowed predicate", which read as one instruction, so #1186 merged them.

**"No block-level cap" holds only for an uncited block.** §3.5's ratchet rule 5
(`RuleCitationOverOneLine`) flags a production block of class `Citation` longer than one line. A
cited production block is therefore capped at one line. Two cited survivors may never sit adjacent.
See §4.5 for the placement consequence.

**A multi-line block is legal when uncited.** `citation-over-one-line` fires on class `Citation`
alone, so an uncited block of independent reasons passes at any line count. `internal/scan`'s
`embeddedLogList` carries three independent uncited reasons on three lines, and `lint` reports zero
(#1189). Do not split a legal block out of caution.

**Where ratchet rule 5 and placement conflict, drop the citation.** Write both reasons as one
uncited block. Never split a reason away from the statement it is about. `completeCTTailTiled`'s
checkpoint guard needed two independent reasons above one `if`, and both went uncited (#1198).
`ProgressChannel` is a package-scope `const` with exactly one placement slot, and its two reasons
landed as one uncited two-line block (#1199).

### 4.5 Where a salvaged line goes

**A salvaged line moves out of declaration position. Declaration position is left empty.**

Two reasons:

1. It keeps §3.5 flag rule 1 from firing on the sweep's own output. That rule flags **any** own-line
   block in a Go declaration position, marker or not.
2. A reason about a `defer` or a retry belongs beside that line, not above the signature.

**Only an `*ast.GenDecl` and an `*ast.Field` block placement 1.** Every other node accepts a
survivor directly above it, because `go/ast` hangs no `Doc` span there. This is the usable form of
the rule.

| Shape | Node | A survivor directly above it |
| --- | --- | --- |
| `var max string`, `const x = 1`, an in-body `type` | `*ast.GenDecl` | Trips flag rule 1 |
| A struct field, an interface method | `*ast.Field` | Trips flag rule 1 |
| `q := url.Values{}`, `chunk := 0` | `*ast.AssignStmt` | Legal (#1190) |
| A one-comment `case` body | `*ast.CaseClause` | Legal (#1198) |
| A composite-literal element | — | Legal (#1182) |

**An in-body `var`, `const` or `type` is still declaration position.** `go/ast` attaches a `Doc`
span to a `GenDecl` inside a function body exactly as it does at package scope. A line placed above
`var got []string` trips flag rule 1 as surely as one above the signature. Three of the four agents
in the second sweep batch hit this, each at the cost of a lint round (#1166, #1168, #1169).

**The position directly *after* an in-body `var` is legal.** The rule above reads as if a comment
between `var x T` and the next statement were unsafe. It is not. `go/ast` attaches `Doc` to a
*preceding* group only, so the hazard is the line above the `var` and never the line below it.
#1202 used the slot to fit a second cited reason into a function whose budget was exhausted, and
#1206 and #1207 confirmed it.

**`:=` is not declaration position.** `go/ast` attaches no `Doc` span to an `*ast.AssignStmt`, so
placement 1 works directly above a `:=` (#1190).

**An interface method is a fourth declaration-position shape.** `*ast.Field` covers a struct field
and an interface method alike. `internal/remoteexec/conn.go`'s `Conn` carried four method
docstrings, and no body to hold them (#1170).

**A one-comment `case` body is a placement-1 target.** Put the survivor inside the arm its reason is
about (#1198). Do not use placement 3 above the `switch`, because the reason is about one arm.

**Three placements, in this order of preference:**

1. **Above the statement the reason is about**, inside the body. Where the reason attaches to no
   single statement, put the line above **the first statement of the body that is not a
   declaration**.
2. **Trailing on the declaration line**, where the reason is about the declaration itself. The line
   must still meet §4.4's 100-character cap, measured after `gofmt -w`, and its six-word floor where
   the line carries no citation.
3. **Above the declaration, detached by a blank line.** A blank line breaks `go/ast`'s `Doc`
   attachment, so flag rule 1 does not fire.

**Placement 3's blank line goes below the survivor.** That line is the load-bearing one, because it
is what detaches the comment. #1187 put the blank line above a struct-field survivor instead, and
`go-decl-comment` still fired.

**Inside a struct field list, placement 3 needs a blank line on each side.** The line below detaches
the comment. The line above stops the comment reading as the previous field's trailing group
(#1186).

**Judge placement 3 by that mechanism, not by a list of declaration kinds.** It holds wherever the
blank line breaks the `Doc` span. Confirmed positions: inside a `const` block, inside a struct field
list, above a package-scope `var`, and an interface method (#1178, nine survivors).

**Placement 2 ranks above placement 3 for an `*ast.Field`.** Placement 3 is the escape where
placement 2 will not fit, not the default for a field. #1190 found placement 2 better four times,
notably `CTSource.DisplayName`, where placement 3 would have put two blank lines inside a four-line
interface.

**A wide interface inverts that preference.** `cmd/web/handlers.go`'s `store` declares 178 methods,
and only a handful leave trailing room inside §4.4's 100-character cap, so placement 3 is the
default there and placement 2 the exception (#1203). The narrow-interface reading above is right for
its case and the wide-interface reading for its own. Measure the trailing room before you choose.

**Placement 3 is what a declaration with no body needs.** A `const`, `var`, `type` or `interface` at
package scope has no statement to sit above. A declaration line long enough to break §4.4's
100-character cap has no room for a trailing comment either. `internal/measure/useragent.go` is the
clearest case: one `const`, no function, an 84-character declaration line.
`internal/commentlint/screen/screen.go`'s `var (` block took the same shape.

**Read the initialiser, not the `var` keyword.** Two `var` shapes look alike and differ.

| Declaration | Body? | Placement |
| --- | --- | --- |
| `var libraryCiphers = func() map[string]uint16 { … }()`, `internal/measure/tlsacceptance/library.go:10` | Yes | Placement 1, inside the literal. Placement 3 here is wrong (#1172). |
| `var nonGlobalTargets = []T{…}` | No | A composite literal holds no statement, so placement 1 in the usual sense is unavailable (#1184). |

**A composite literal's element is a placement-1 target.** A survivor above an element —
`PerHostConnPerSec: 50,` or `SANDNS: leaf.DNSNames,` — is not declaration position, and `lint`
accepts it. Six survivors in #1182 landed this way.

**Prefer the element that opens the alignment run.** An element is a legal target, and a safe one
only where it opens the run. #1200 placed a survivor above `MaxAttempts:`, the 8th of 9 elements.
That split the run, and `gofmt -w` reformatted two lines. Check `gofmt` after you place a survivor
above a middle element.

**Ratchet rule 5 constrains placement, not only wording.** §4.4 states the wording half. A block is
a run of contiguous own-line comments, so **two cited survivors may never sit adjacent.** They need
a statement between them.

Inside a body, the statement count is a budget on how many cited reasons fit. `internal/drift`'s
`CountDelta` has a single `return`, so its body offers one block position (#1176). **That budget is
not the function's whole budget.** Placement 3 above the `func`, plus placement 1 in the body, gives
two cited lines, because the signature line separates the two blocks (#1200, `resolutionNameKey`).

**All three placements can be illegal at once.** `internal/custody/corpus`'s `SANsBelowThreshold`
and `SANsAtThreshold` have one-line bodies, so placement 1 needs a reformat §5.1 forbids. Their
trailing room is 33 characters, too little to clear ratchet rule 2's six-word floor inside the
100-character cap, so placement 2 fails. Placement 3 was the only legal option.

**§3.5's corrected safety claim rests on placement 3 existing.** Flag rule 1 is safe against the
sweep's own output only where every survivor has a placement. Placement 3 is the one that survives a
declaration with no body, no trailing room, and no legal reformat. Read it as load-bearing, not as a
third preference.

**A sweep never reformats code to make room for a comment.** §5.1 forbids it. A one-line function
body such as `internal/wire/wire.go`'s `Overflowed` has nowhere to put a line. Splitting it across
lines makes `go/scanner` insert an automatic `SEMICOLON` before the closing brace. The skeleton
gains a token, and `verify` reports the file changed. #1167 measured this before relying on it, and
used placement 3 instead.

**Where no placement holds, delete the line.** §4.2 is the tie-break, and it applies here like
anywhere else.

**A build-tag reason goes above the platform-specific expression.** The reason a `//go:build` pair
exists is about the **file**, and every placement above is statement-relative or
declaration-relative. Placement 1 is still the target, and the statement to pick is the one the tag
exists for. #1201's survivor sits in `internal/queue/proberexit_unix.go`, above the
`ps.Sys().(syscall.WaitStatus)` type assertion, and it got there by the fallback rule rather than by
this one. The windows twin's own block died under §4.10 and left nothing behind.

### 4.6 Test files

Ruling 2 makes "the ADR or issue this test pins" an explicit WHY exception. Three rules set how far
it reaches.

1. **Uncited prose that asserts what the test already asserts is deleted.** The test's name and its
   body are the assertion. A comment restating them is the WHAT that `CLAUDE.md` forbids. This
   reaches 2,613 lines of `docstring-exported-other` and 1,491 lines of test `prose-other`.
2. **A cited comment survives where it carries a rule the test does not assert**, compressed to the
   §4.4 form.
3. **An uncited comment that states a reason is judged like any other comment**, under §4.1. The
   citation requirement governs behaviour-assertion prose only.

**Rule 1 has a mechanical tell.** Where the `t.Errorf` or `t.Fatalf` message quotes the claim, the
claim is recoverable from the body and gate A fails with no cross-block test. Three of #1201's six
deletions turn on it. This is §7.6's `Row.Claim` finding generalised out of the corpus packages:
wherever the failure message states the rule, the comment above it is redundant.

**The tell has a production twin.** `coverageMessages`' docstring was quoted almost verbatim by the
operator-facing string its own body builds (#1206). Read the strings a declaration composes before
you keep a docstring above it.

Rule 3 is load-bearing. `cmd/web/adr0130_contract_test.go:498` is an uncited WHY inside a test file.
It explains a classification decision. An absolute citation requirement would delete it. §4.7 keeps
it, and §4.7 wins.

**Rule 2's qualifier is the whole rule.** §4.1 keeps a comment only where it passes both gates. A
bare "a cited comment survives" contradicts that inside a test file, and reads as categorical when
it is not. The qualifier is the discriminator the merged precedents use. #1184 applied it to two
`§3.3` citations and split them in opposite directions. `safety_test.go`'s uniform-halving block
died. `leaf_test.go`'s profile-table line was kept.

**The sharper form: a cited test comment survives when its clause names a consequence the assertion
cannot fail on.** #1197 decided all 48 of its test blocks with that form and left no residue. It
split `#1035`, `#1036` and `#1018` citations in opposite directions inside one file. Use this form
where rule 2's wording leaves a block undecided.

**A benchmark is a different surface from a test.** Rule 1 deletes prose that asserts what the test
already asserts. **A benchmark asserts nothing.** Its output is numbers a human reads. Every reading
instruction it carries is therefore unrecoverable by construction: which number means what, and
which comparison is the point. The naive rule-1 reading deletes the lot.
`BenchmarkToEdgeFanout`'s 37-line docstring was the largest block in #1197, and it yielded three
survivors.

**The test-local hazard is the class most likely to survive.** Do not search a test file for domain
prose. The reason is structural. A domain rule has a production home, and §4.1's cross-block test
keeps the rule there. A fixture hazard has nowhere else to live. Six of #1200's eleven test
survivors are facts about the test's own construction. The `nil` `qtx` **is** the assertion, the
fake reproduces `ExecProber`'s `ctx.Err()`, and a one-entry Merkle tree's leaf hash is its root.

### 4.7 A reason with no ADR to cite

**An uncited reason survives.** The citation in the §4.4 form is optional.

Ruling 9 exists so a sweep never stalls. It does not exist to destroy reasons. A hazard note ("this
retries because the upstream 502s on cold start") is a fact about the world. It is not an
undocumented decision. Deleting it to satisfy a citation format inverts the rule's purpose.

**A follow-up issue opens only where the comment asserts a decision** — a rule someone chose that
ought to be an ADR. A hazard, a cost note, or an external constraint survives silently and opens
nothing. §8.2 narrows the trigger further with two more gates.

**A survivor may not cite a document that does not exist.** Repair has purchase only where a live
document states the rule, so "prefer repair to deletion" is too broad. Three dangling families are
measured, and they take three routes.

| Family | Repairable? | Route |
| --- | --- | --- |
| `PARITY-CHART.md`, `SPEC-CHANGE.md` | Sometimes | Re-cite the live source. Precedents: #1182 → #464, #1178 → `docs/spec/v1-spec.md` §3.5, #1199 → `docs/spec/raw-job-output.md` §6.2 |
| `docs/research/ct-bulk-primary-2026-08.md`, `docs/research/ct-logs-direct-2026-08.md` | Usually | Re-cite `docs/spec/ct-source-replacement.md`, which is on `main` |
| `AUDIT-LEDGER`, `AL-2`, `AL-25`, `P0.7`, `DF-F4`, `G2` | **Not by token** | The token is unrepairable. Search the surface below for the rule. Where nothing states it, keep the reason uncited and record the gap |

`docs/spec/raw-job-output.md` §6.2 names `SPEC-CHANGE` collision #40 by number and states "persists
nothing at rest", which is the rule the comment carried (#1199).

Neither research file is on `main`. `docs/spec/ct-source-replacement.md:26` records that they live
on the unpushed branches `research/cert-spotter-primary` and `research/ct-logs-direct-facts`.

**The third family names nothing a citation can reach.** Verified on `origin/main` at `5928e2b`:
`AUDIT-LEDGER`, `AL-2`, `AL-25`, `P0.7` and `DF-F4` each appear in zero files under `docs/` and in
`CONTEXT.md`. #1194 and #1196 confirmed this independently. `G2` is the token that resolves falsely
instead, and the next rule below measures it.

**The third family's reach is measured and narrow.** It reaches the production files of
`internal/queue` and stops at the test surface. `progress.go` carries the first family instead, and
`progress_test.go` carries neither (#1199, #1200). Grep your own files. Never trust the family's
stated reach.

**Route 3 rules the token, never the rule.** "The token names nothing" and "the rule is unwritten"
are two claims, and only the first is established. An agent that conflates them ships an uncited
survivor **and** files an ADR gap for a rule already on disk. The failure is silent. #1204 did
exactly this: it filed the never-fabricate rule as gap 2 of **#1333**, and `docs/adr/0110` line 74
states it. Three tickets reached the correction independently.

| Dead token | The live source that states the rule |
| --- | --- |
| `DF-F4` on `internal/queue/worker.go:331` | `docs/spec/raw-job-output.md` §2.4 (#1328) |
| `PARITY-CHART P2.2`, `P0.1` on `cmd/web/signals.go` | `docs/spec/v1-spec.md` §6.5 (#1207) |
| `SPEC-CHANGE`, the never-fabricate family | `docs/adr/0110` line 74 (#1205) |
| A bare `SPEC §n` on `cmd/web/scans.go` | `docs/spec/scans-monitor-bounding.md` §2.2, §2.3, §3, §4 (#1208) |
| `#778` on `cmd/web/inventory.go` | `docs/adr/0125`, Decision ground 5 (#1211) |

Also reconfirmed as targets: `docs/spec/v1-spec.md` §4.1, §5.2 and §5.3.

**The search surface is five places:** `docs/spec/`, `docs/adr/`, `docs/research/`, `docs/guides/`
and `CONTEXT.md`. `docs/guides/` is the one an agent forgets, and the one an operator-facing product
rule tends to sit in. #1205 moved three of its four gap candidates out of the gap list with
`docs/guides/accounts.md` and `docs/guides/running.md`.

**The cheapest repair is the cited issue's own title.** A dead token is often the title prefix of
the issue cited beside it. `P0.3`↔#444, `P0.6`↔#499, `P2.4b`↔#468 and `R4-D4`↔#759 all repaired
that way in `cmd/web/reports.go`, and only `P0.2` resolved nowhere (#1210). Read the issue title
and drop the prefix before you hunt for a document.

**Four surfaces are never a repair target, and all four answer a grep.**

| Surface | Why it answers |
| --- | --- |
| `docs/spec/comment-policy.md` | This SPEC names every dead token it rules on — `AUDIT-LEDGER`, `AL-2`, `AL-25`, `P0.n`, `P2.n`, `R4-D4`, `DF-Fn`, `G2`, `SPEC-CHANGE`, `PARITY-CHART` and each `#nn` collision id — in order to record that they are dead. The set grows with each amendment, so treat any hit in this file as a hit on this row. |
| `docs/research/comment-*` | `comment-gate-test.md` is the §3.9 gate sheet and quotes the corpus verbatim. |
| `docs/research/sql-stripper-*` | `sql-stripper-cross-check.md` is #1143's record of the same corpus. |
| Any `.tmpl` comment | `design-system/templates/scope.tmpl` states `DF-F1` and `DF-F2`. A template comment is not a document, and it is stage-D3 sweep territory itself. |

Each of the four went from zero hits to one when this effort's own artefacts landed, so the
mechanical check now returns the wrong answer and a reader may trust it. #1205 and #1206 reached
this independently. The test below is the defence: a hit must **state the rule**, not state that
the token is dead.

**A citation resolves only when all four of these hold.** A file-existence check is not enough. A
grep for the token alone is not enough.

1. **The named section states the rule the comment states.** `G2` appears in 6 files under `docs/`,
   in two unrelated senses. Five use `G2` as ADR-0057's gate-check label over graded footing cells:
   `docs/adr/0057`, `0059`, `0077`, `0120` and `docs/research/sensitive-ports.md`. `docs/adr/0116`
   line 3 uses `G2` for the retired design-system parity gate. An agent grepping the token alone
   keeps a dead citation and believes it repaired one. A bare `AL-25` or `G2` is not a citation at
   all under §2.1's decisive test, which is what makes this dangerous.
2. **The document is not Superseded or Withdrawn.** `docs/adr/0116` **establishes** the
   `SPEC-CHANGE` collision protocol and the G1/G2 parity gates, then withdraws them in its status
   line, and its body still describes both at length. A citation to it passes a file check, passes
   a section check and passes test 1, because the section genuinely states the rule in order to
   retire it. A dead token fails a grep honestly. This one fails no check at all. **Read the status
   line of every document you cite.** #1202 found it, and #1203 and #1204 confirmed it
   independently. #1203 had already shipped a survivor citing it before the catch.
3. **The section states the rule on its own authority.** `ADR-0120` is `Accepted`, names
   `cmd/web/devfixtures.go` by path, and states the no-fabrication rule almost verbatim — but only
   by quoting the retired `SPEC-CHANGE` protocol, saying "this protocol" outright. It is not a
   target, and #1204 withdrew the repair. `docs/spec/v1-spec.md` §6.5 states its rule in its own
   voice and merely attributes it, so its "(P2.2. The design package is normative for
   functionality)" does not disqualify it (#1207). The surface form is identical, so read the
   sentence and not the token.
4. **The section rules, rather than forwarding to the rule.** ADR-0129's #954 amendment states the
   vantage rule and ends "(§5)". §5 rules "v1 ships fan-out alone". A section that forwards has not
   resolved (#1328). `ADR-0129 §5` is therefore a false resolution of the `G2` shape sitting inside
   the ADR corpus, like `ADR-0116`: the section exists, the document is `Accepted`, the text carries
   the right word, and it rules something else.

**A `#nnn` may be a design-collision id whose number resolves to an unrelated issue.** This is the
worst family, because `#nnn` is §2.1's own decisive test for the `citation` class, so nothing
mechanical separates it from a real citation. Three shapes are measured: a **bare** `#nnn`
(`collision #40`, six times in #1208, resolving to a live issue titled "What is the right Seed
primitive for a cloud-resident estate?"), a `#nnn<letter>` (`#21`, `#21i` and `#21j` in
`cmd/web/signals.go`, plus `#26f`, `#26c`, `#26h` and `#21d` in `cmd/web/settings.go`), and
`SPEC-CHANGE #nn`. Verified dead: `#20`, `#20a`, `#20b`, `#20d`, `#20e`, `#21d`, `#22a`–`#22f`,
`#23b`, `#23e`, `#23h`, `#34`, `#35`, `#40`.

**The issue API is necessary and not sufficient.** A `#nnn` whose issue was deleted returns
HTTP 410 and is invisible offline. `(#738)` on `trustedProxies` is one, and nothing under `docs/`
distinguishes it from a live citation (#1203). So query the API. But a collision id's number
resolves live, and the API says nothing about whether that issue is the rule's source. **Read the
issue title.**

**A wrong citation is worse than a dead one**, because it survives a file-existence check. Nine
instances are measured.

| Citation | What the named section states | What the rule needs |
| --- | --- | --- |
| `(ADR-0027 §7)`, five times in `internal/queue/crtsh.go` | ADR-0027 has no §7 | `docs/research/passive-discovery-sources.md` §7 (#1196) |
| `(research §3.3)` on `certSpotterInterval` | AXFR and operator-supplied zone data | §2.3, the ten-queries-an-hour cap (#1196) |
| `(ADR-0134 §5)` on "the two instants' counts need not agree" | Not that rule | `#1046` (#1197) |
| `(ADR-0129 §5, #987)` in `edgefanoutmessage_test.go` | "v1 ships fan-out alone" | The #944 amendment (#1200) |
| `(spec §2.4)` in `certspotter_test.go` | Where the key is held | Nothing. The Bearer-header rule is stated nowhere (#1200) |
| `(ADR-0129 §6)` on `internal/queue/edgefanout.go:268` | Not the vantage rule the comment states | `#956` (#1328) |
| `(ADR-0130 §3)` on `cmd/web`'s `server.routes` | "Redirects preserve the submitting URL" | Nothing. Validating a redirect against the route table is stated nowhere (#1203) |
| `(#286)` on the anchor-slug rule in `cmd/web/seeds.go` | T10 IA reconciliation | `docs/spec/v1-spec.md` §5.3 (#1206) |
| `ADR-0081` on the host-only redaction rule, five sites | A Delivery-has-no-cause rule | Nothing. The host-only rendering rule is stated nowhere (#1354) |

**`ADR-0130 §3` is the shape that falls through both repairs.** It names a live section of a live
document that rules something else, so §8.3 suppresses the gap and this section has nothing to
repair to. Name it in the PR body, because nothing else carries it.

**A repair is a package-wide fact.** Name the repaired token in the PR body, so a sibling ticket in
a split package greps for it. #1196 repaired `(ADR-0027 §7)` five times in `crtsh.go`. The same
token then appeared in `pure_test.go:64` and twice in `crtsh_test.go`, both owned by other tickets.

**Wrong citations cluster, so a repair checks every citation to that document in that file.** #1328
was opened for two citations and its file held three. `edgefanout.go:268` cites `ADR-0129 §6` where
the rule is #956, and only a sweep of every `ADR-0129` reference in the file found it.

**`ADR-0081` is misattributed, and both a blanket replace and a blanket repair target are wrong.**
It contains zero occurrences of "host", "token" or "URL". **Five** sites cite it for the host-only
redaction rule — `internal/message/render.go:395`, `cmd/web/reportdelivery_test.go:20` and `:102`,
and `cmd/web/reports.go:1114` and `:1223`. Its **eleven** other uses cite it for a
Delivery-has-no-cause rule, which it does state: `cmd/web/messages.go:21`, `:94`, `:270` and `:508`,
`cmd/web/messages_test.go:122`, and six `internal/db` sites, which are three comments each written
into `querier.go` and into its own query file. Those eleven are correct and stay.

**No document on `main` states the host-only rendering rule, so the five have no repair target.**
`ADR-0053` was nominated for it and does not carry it. `ADR-0053` rules secret *custody* — `web`
renders "set" or "not set" and never a value — and it names "a webhook URL with a token in it" only
as an example inside its Context. It fails test 1 above. Route 3 applies:
the five go uncited and the rule is recorded as a gap. **#1354** tracks the repair, and its own
"Done when" list names `ADR-0053` and three sites, so it needs correcting before it runs.

**Check that a survivor's citation resolves before the PR opens.** The ratchet does not provide this
check. **Four defects have reached `main` in merged sweep output, and two are still open.**

| Defect | Landed by | State |
| --- | --- | --- |
| `internal/queue/worker.go` cited `DF-F4`, and `edgefanout.go` carried `(ADR-0129 §5)` on a rule §5 does not state | #1194, PR #1317 | **Repaired.** #1328 merged as `b158035` and fixed a third the file also held. |
| `ADR-0081` on the host-only redaction rule, five sites | Batches 5 to 10 | Open. **#1354**, and see the correction above. |
| `cmd/web/seeds.go:453` says the head block inlines `tokens/*.css` "only when this datum is set". 42 `cmd/web` sites set it `true` and 0 set it `false` | #1206, PR #1340 | Open. **#1355**. |

Do not repair an open one under a sweep ticket.

### 4.8 The `package-doc` cap

**Three lines, plus a content constraint.** The block may state the package's purpose and its ADRs.
**It may never describe a symbol inside the package.**

57 blocks hold 768 lines at a median of 12 lines per block, so this is the longest-per-block class in
the corpus. Three lines cuts it to about 170 lines. `cmd/web/main.go:1` already sits at exactly three
lines and needs no edit.

The SPEC states both a number and a content rule. A number is what a sweep agent can apply. A content
rule alone would reopen the judgment this section closes.

### 4.9 The keeps ledger

**Every sweep PR body carries a keeps ledger.** It lists every surviving and every rewritten comment
as `file:line`, old to new.

§4.2 argues for delete-by-default on the ground that a diff shows the reviewer every deletion and no
keep. That argument obliges this ledger. Without it the reviewer has no view of the judgments the
rubric made in favour of keeping.

**One deletion earns a row: a comment deleted under §4.10 as false.** Give it `false` as its reason.
The diff already shows the text that left. It does not show that the sweep judged the claim wrong
rather than redundant, and that is the judgment a reviewer needs to see.

**Where §4.1 and §4.10 both fire on one block, the ledger records `false`.** A duplicate is visible
in the diff, because the sibling that keeps the rule is still in the tree. A false claim is not.
`internal/queue/proberexit_windows.go`'s block was both, and #1201 recorded `false`. The opposite
choice loses the §4.10 record entirely.

The ledger carries a second table, the gaps table (§8.6). A ticket that finds no ADR gap states
`ADR gaps: none` (§8.7).

The volume is manageable. The salvage population is about 1,729 declaration blocks tree-wide, so a
per-package slice yields a ledger of tens. §7.1 sizes the cap against that.

### 4.10 A comment that is false

**Neither §4.1 gate asks whether the fact is true.** Gate A tests recoverability and gate B tests
causality, so a comment that states a wrong fact can pass both and survive.

**The verdict: delete it, record the deletion, and never re-author it.** A sweep compresses
comments. Rewriting a false claim into a true one is a product judgment, and neither the §5.1 token
gate nor the §5.6 diff-shape gate reviews one. §4.9 gives the deletion a keeps-ledger row, so the
human on the PR sees the call.

Three measured cases, all from the second sweep batch:

- `internal/measure/httpexchange/exchange.go`: the `+1` on the body `LimitReader` claimed the fold
  distinguishes exactly-filled from overran. `Identity` does no such thing.
- The same file: a paragraph called a transport error "our own blindness, never an identity value".
  It sits four lines below the ADR-0011 ruling that the error folds to the `no-http-response`
  **value**.
- `internal/report/dispatcher.go`: the `Dispatcher` docstring said the off-instance send "is a
  later ticket (#508/T7)". That ticket landed, and `notify.go` is the send.

§4.2 reaches the same three deletions, because an agent that cannot decide deletes. It reaches them
for the wrong reason. A false comment that passes both §4.1 gates cleanly would survive without this
ruling.

**A comment spliced onto the wrong symbol is false too.** All three cases above are comments the
world falsified: a ticket landed, a field was added, a caller changed. A splice was never true, and
nothing moved under it. `internal/custody/fanout.go:185`, above `func numericTopLabel`, is a
truncated copy of `isLDHDomain`'s docstring, welded mid-sentence onto `numericTopLabel`'s own text.
`isLDHDomain`'s docstring still sits complete 25 lines below. As written the block asserts that
`numericTopLabel` tests the LDH character allowlist and is "an allowlist, never a blocklist". It
does neither (#1186). The verdict above still applies. Its rationale does not, so an agent reading
this section may not recognise the case.

**A splice is invisible to `commentlint`.** That block classifies `docstring-unexported`, and the
screen's `RFC` hit withheld it from mechanical deletion, so it reached stage D intact.

**Six more falsifiers are measured. Only the first three are the product moving under the code.**

1. **The repo environment.** `internal/queue/proberexit_windows.go` justified the file by "its tests
   run on a windows dev host". There is no windows dev host: `CLAUDE.md` records the move to Ubuntu
   Server 24.04 on 2026-09-03 (#1201). This is the class a sweep agent is best placed to catch and
   least likely to look for, because the falsifying fact sits in `CLAUDE.md` rather than in the code
   being read. Delete the false justification, not the build tag. Whether a windows build is still
   wanted is the product judgment this section forbids.
2. **A live `Accepted` ADR.** ADR-0124's #1240 amendment names `cmd/web/settings.go`'s
   `updateHostSteps` comment and strikes its promise permanently (#1205). A document falsified it,
   not the code. This is mechanically detectable wherever an ADR names a symbol.
3. **A `.tmpl` contract.** `shell.tmpl`'s `head` define falsifies `cmd/web`'s `.DesignTokens` claim
   (#1208, **#1355**).
4. **The function's own body.** `lastUsed` "renders an em dash when never presented" and the body
   returns `""`. `redirectTo`'s "302 for a nuanced one", and its three call sites pass 301, 303 and
   301 (#1202, #1203). Not the world, not a splice.
5. **The file growing around the claim.** `handler()`'s "the only mutation that exists yet,
   POST /accounts" was true once. The function now registers about 50 mutating routes (#1203).
   Nothing moved under it.
6. **Implication alone.** `writeSignalsExportCSV` draws a contrast the body contradicts, while
   neither clause is itself false (#1207).

**A dead citation is not by itself a falsity verdict.** `cmd/web/seeds.go:700` cites the superseded
ADR-0116 and its prose is merely redundant, so it dies on gate A with no `false` row. `:842`
restates the retired doctrine as live and earns one (#1206). Read the prose, not the token.

**Half a block may be false.** This section rules on blocks and has no verdict for a half-false
sentence. #1204's `// the byte-exactness gate before the pixels`, 11 instances, names a live drift
test and only its "before the pixels" framing is retired. §4.1 is what saved it, not this section.
**Where the false clause and the true one are independent reasons, delete the false clause and keep
the other (§4.4). Where one sentence carries both, delete the block.**

**The retired handoff doctrine is a class, and its tell is vocabulary rather than a citation.**
ADR-0109 and ADR-0116 are Superseded. `CLAUDE.md` records that `design-system/` is the source of
truth and may be edited in-repo. No `G1`/`G2` gate and no byte-comparison step exists in
`.github/workflows/`, verified twice.

**Pair the status-line test (§4.7) with a phrase grep, as two independent detectors.** The
status-line test is a citation test and this defect is mostly uncited. `cmd/web/reports.go` carried
**7** assertions of the withdrawn doctrine and exactly **1** named an ADR (#1210). The other six
cite nothing, a dangling `#23e`, `SPEC-CHANGE`, or a live issue number. `cmd/web/graph.go:20-27`
asserts the doctrine in full — "the frozen design-owned graph.tmpl … embedded read-only … the repo
authors no markup/CSS/JS for this route … kept byte-for-byte" — and cites only `#583` (#1211).
Across the 28 unswept `cmd/web` production files, 18 carry the phrasing with zero `ADR-0109` or
`ADR-0116` reference.

The phrase list: `byte-for-byte`, `frozen design`, `design-owned`, `normative for look`, `re-skin`,
`authors no markup`.

**Every phrase hit needs a human read, and both false-positive directions are measured.** #1208
found 15 further uses of "frozen" that were bare modifiers on a live template. A template embedded
through `designfs`, or a shared parse set, is a plain mechanical fact and not the doctrine. #1210
found the other direction at `reports.go:896`, where "PeriodLabel/RangeLabel **re-skin** the header"
is relabelling in the ordinary sense. And a live thing can wear the retired framing: `:730` and
`:999` name a byte-compare against a golden, and `devfixtures_test.go` does read
`design-system/fixtures/fixtures.json` — but it pins fixture *values*, not a rendered byte-compare.
That is the half-false shape above.

**A true claim the type cannot express is not false.** Delete it under §4.1, and give it no `false`
row. `custody_test.go:58` claimed a fixture expressed a declared-but-unextended name scope, which
`Estate` has no field for. The sentence is true of the product and only unrepresentable in the type,
so #1187 deleted it under §4.1.

**A closed-`#nnn` deferral clause is a candidate for review, never a verdict.** The detection is
mechanical: a comment matching a deferral phrase near a `#nnn` whose issue is closed. The detection
is sound. The verdict is not. Read the clause before you delete it. Three instances match the
detection and are false:

- The `#508/T7` clause above, in `dispatcher.go`.
- `internal/measure/connectoutcome/certificate.go` — "a later ticket adds an HTTP step (#198)".
- `internal/queue/worker.go` — `Prober`'s Transcript is "ABSENT until #840 captures it".

**Two clauses naming a closed issue were true.** #1198 read two clauses naming #874 and #877 as
"deferred alongside that issue's own deferral", not as "pending that issue". A mechanical rule built
on the detection alone flags both. **A `commentlint` rule for the detection is a separate call from
this ticket.**

**A third shape dies elsewhere.** `httpidentity.go` held a `P0.11` reference that narrates work
which has landed (#1200). Change narration dies under §3.2's negative screen and under §3.5's
flag rule 3, not under this section.

**A false claim opens no ADR gap.** §8.2 gate A needs a rule someone chose. Where the comment's
claim is wrong, there is no such rule to record.

---

## 5. The equivalence check

Ruling 15 requires every sweep PR to prove that no non-comment byte changed.

### 5.1 Equivalence is token-stream identity

**The check compares token streams. It does not compare stripped text.** This is the load-bearing
definition.

A comment can act as a token separator. `SELECT/*c*/a` strips to `SELECTa`, which is different SQL. A
text comparison accepts that. A token comparison rejects it.

**Protected directives stay in the skeleton as tokens.** That turns ruling 10 into an enforced
property rather than a convention the sweep is trusted to honour.

### 5.2 The per-surface verdict

The check is achievable on **9 of the 11 surfaces**, covering **868 of the 898 in-scope files**.

| Surface | Files | Verdict | Mechanism |
| --- | ---: | --- | --- |
| Go, production and test | 444 | **Sound** | `go/scanner` token stream |
| `.sql` | 106 | **Sound** | purpose-built PostgreSQL tokenizer |
| `.css` | 18 | **Sound** | purpose-built tokenizer |
| `.d.ts`, `.ts`, `.mjs` | 135 | **Sound** | purpose-built JS lexer, cross-checked against esbuild |
| `.jsx` | 141 | **Sound with caveats** | esbuild canonical form |
| `.tmpl` | 24 | **Sound with caveats** | `text/template/parse`, plus the §5.4 delete rule |
| `.html` | 24 | **Not achievable** | a comment's bytes are output bytes |
| `.astro` | 6 | **Not achievable** | needs `@astrojs/compiler` |

Go measured zero failures across lex, completeness, soundness and directive protection. `//go:build`
survives: the tree holds 11 protected Go directives, and deleting one moves the skeleton. SQL
measured 384 protected directives and zero failures. The SQL tokenizer handles `''` escapes,
dollar-quoted strings and nested block comments.

### 5.3 Tooling constraints

Three constraints are measured, and each one costs a session to rediscover.

1. **Do not use esbuild for TypeScript.** It erases a `.d.ts` file to the empty string, measured at
   109 of 109 canonical forms empty and 109 of 109 code mutations invisible. The tree has 116 `.ts`
   files, and 109 of them are `.d.ts`, so an esbuild-based design loses the dominant TS surface
   silently.
2. **Use esbuild for `.jsx`.** No hand lexer can read JSX, because JSX text makes `//` literal.
   esbuild parses all 141 files as a fixed point and is mutation-sensitive.
3. **A future `.tsx` file needs a third tool.** It would meet the `.d.ts` erasure problem under
   esbuild. The repo has no `.tsx` file today.

Decide the lexer by file extension, never by content. TypeScript generics look like JSX to a lexer.

### 5.4 The `.tmpl` delete rule

**Delete the comment's byte range. Leave its line.**

Deleting the comment's whole line fails byte-exact comparison in **24 of 24** files, because the
line's own newline reaches the browser. Deleting the byte range only passes **24 of 24**. No `.tmpl`
file in the tree holds an HTML comment.

### 5.5 Two caveats

**The circularity caveat.** SQL and CSS have no independent cross-check. The same hand lexer both
finds the comments and builds the skeleton. A lexer that misreads a comment misreads it
consistently, so the check still passes. This is the one unvalidated risk in the design.

**The mitigation is a one-time gate.** Cross-check the SQL stripper against `sqlc generate` and a
`goose` dry run **before the first SQL sweep ticket lands** (§7.2 stage D2). It needs binaries that
`go test` does not have, so it is a scheduled gate, not a unit test.

**The whitespace caveat.** A token check is whitespace-insensitive by construction, so it would
accept a reformat. §5.6 is the second gate that gives ruling 15 its literal property.

### 5.6 The two-tier diff-shape gate

**This gate governs the §7.2 stage-B mechanical PR. It does not govern a stage-D judgment sweep.** A
stage-D ticket rewrites a survivor to the §4.4 form, which adds a comment line, so its hunk is
neither a pure deletion nor whitespace-only. The pilot's own merged diff (#1138) has that shape, and
so does every PR in the first two sweep batches. What binds a judgment sweep is §5.1 token identity,
which `commentlint verify` runs on every sweep PR (§6.11).

A strict "every hunk is a deletion" gate and the §3.8 formatter step cannot both hold. Removing a
blank line inside a `const` or `struct` block merges two `gofmt` alignment runs, so `gofmt`
re-columns untouched code. The token gate passes, because a token stream carries no whitespace. The
strict diff-shape gate fails.

**The gate is two-tier.** A sweep PR passes when one of these holds for each hunk:

1. The hunk is a pure deletion.
2. The hunk is **whitespace-only**: its added and removed lines are byte-identical after every
   whitespace run collapses to nothing.

Both tiers are mechanical. Together they say "no code changed, and nothing but comments and layout
was touched". This keeps the formatter step intact and names the escape hatch rather than leaving it
implicit.

---

## 6. The `commentlint` tool

### 6.1 Shape and location

`commentlint` is **one Go binary with three subcommands**, built on **one per-surface lexer** that
every subcommand shares.

| Item | Decision |
| --- | --- |
| Binary | one, `cmd/commentlint` |
| Subcommands | `lint`, `strip`, `verify` |
| Packages | `internal/commentlint/{surface,screen,rule,strip,verify,scope}` |
| Dependencies | standard library only |

The single lexer is the load-bearing choice. Three jobs need comment extraction: flag, delete, and
prove. Three copies of that extraction is how the delete pass removes a block that the proof step
then rejects. One object removes the drift by construction.

`cmd/` holds only shipped binaries today, and the Dockerfile names each of the three explicitly. A
fourth `cmd/` package therefore does not reach the image. It does reach `go vet ./...`,
`go test ./...`, `gosec`, `govulncheck` and CodeQL, because all five scan the whole module. The
standard-library constraint keeps that cost at zero new advisories.

**A nested module under `tools/` is refused.** It buys dependency isolation that a zero-dependency
tool does not need, and it costs a second `go.sum` that nobody remembers to update.

If `.jsx` ever needs esbuild, invoke the node esbuild that `docs-site` already installs. Do not add a
Go dependency for it.

### 6.2 The extractor seam

**One interface, one call, two outputs.** A `Lex` call returns both the comment blocks with their
byte ranges and the skeleton token stream.

Two interfaces are refused, because they would state a seam that does not exist. §5.5 records that
for SQL and CSS a single hand lexer does both jobs. One interface makes that shared fate visible, and
it puts the §5.5 cross-check on one testable object.

§1.5 defines a block. A protected directive line is its own block, so a directive never absorbs the
prose beneath it. The own-line block that opens on the line below a `#nosec` line is the **waiver
tail**. `Lex` marks it, and §2.3 states what the mark buys.

### 6.3 The command-line surface

```
commentlint lint   [--github] [--in-scope-only] [paths...]
commentlint strip  [--write] [--manifest PATH] paths...
commentlint verify --base <ref> [paths...]
```

**`lint`** with no path and no `--in-scope-only` lints the whole in-scope tree. This is the writer's
path, copied from `doclint`. `--in-scope-only` never triggers that fallback, so an empty changed set
lints nothing.

**`strip`** prints the residue manifest and changes no file unless `--write` is given. The manifest
is JSON Lines, one record per **declined** block: file, line span, class, and the screen signal that
fired. The default path is `.commentlint/residue.jsonl`. **That path needs a `.gitignore` entry**,
because §3.8 says the manifest is never committed.

`--manifest` names where `--write` saves the manifest. **`--manifest` without `--write` is a usage
error, exit 2.** A dry run prints the manifest to stdout, so a named path that saves nothing reads as
a saved file that is missing (#1274).

**`verify`** reads the pre-sweep content with `git show <base>:<path>`, driven by an explicit
`--base`. The tool does not compute the merge base itself. With an explicit base, the CI job and a
local agent behave identically, and a shallow CI clone cannot change the answer.

### 6.4 Surface coverage differs per subcommand

| Subcommand | Surfaces in v1 |
| --- | --- |
| `lint` | all nine lexable surfaces |
| `strip` | **Go only.** A non-Go path is a usage error, exit 2 |
| `verify` | **all nine lexable surfaces** |

§3.6 fixes the delete pass at Go-only. It says nothing about the other two subcommands, and they are
not the same question. Ruling 15 governs **every** sweep PR, including an agent-judged SQL slice that
`strip` never touches.

A `strip` that silently does nothing on a non-Go path is how a sweep agent believes a slice is done.
A `verify` limited to Go would leave ruling 15 with a hole exactly where the sweep carries the most
risk.

`.html` and `.astro` are excluded from all three subcommands.

### 6.5 Per-surface delete semantics

| Surface | Delete rule |
| --- | --- |
| Go | remove the block's own lines, then `gofmt` (§3.8) |
| `.tmpl` | delete the comment's byte range, leave its line (§5.4) |
| others | measured when the surface's sweep is scheduled |

Each row lands when its surface's sweep lands. The `.tmpl` row is recorded now, because it is
measured and would otherwise be lost.

### 6.6 The rule set and the one heuristic

| Class | Decision procedure |
| --- | --- |
| `section-divider` | a repeated-punctuation or box-drawing run |
| `short-label` | word count on a one-line block, own-line or trailing (§3.5 ratchet rule 2) |
| `docstring-exported-conventional` | `go/ast` Doc comment whose first word is the identifier's name |
| `commented-out-code` | the payload parses as Go **and** carries a code-only token |

`commented-out-code` is the exception, because nothing structural marks it. The parse alone is not
enough: `// see the note above` parses as an expression statement. The code-only token requirement is
what rejects it. The token set is `:=`, `{`, `}`, `return`, `func`, and comparable tokens.

The class is bounded at 22 lines, and the §3.9 sample gate covers it. Dropping it to flag-only is
refused, because that would reopen §3.5 for a risk the sample gate already measures.

**One class per block for the delete decision, by the §2.1 precedence order. Multi-label for flags.**
Deleting needs exactly one answer. Flagging is advisory, so a block that trips three ratchet rules
reports three.

Off Go, `commented-out-code` has no parser, so it is regex-only and advisory. `strip` is Go-only in
v1, so this never reaches a delete.

**Rule ids.** Where a rule maps to a taxonomy class, the id **is** the class name: `section-divider`,
`commented-out-code`, `short-label`, `docstring-exported-conventional`. The ratchet-only ids are
`go-decl-comment`, `change-narration`, `todo-marker`, `citation-over-one-line`. One vocabulary across
the survey, this SPEC, the manifest and the annotation is worth more than matching `doclint`'s `no-`
prefix.

Ruling 12 forbids a WHY heuristic. **No rule here guesses at intent.** Every rule is decided by a
parser, a shape, or a word list.

### 6.7 Output and exit codes

| Exit | Meaning |
| --- | --- |
| 0 | clean, or any violation in `--github` mode |
| 1 | a violation, in human mode |
| 2 | a file failed to lex, or a usage error |

**A lex failure is not a violation.** It means the tool cannot judge the file. Folding it into the
violation count is how a sweep silently skips a file.

Human mode prints one `file:line -> rule` line per violation. `--github` mode prints one GitHub
Actions annotation per violation, plus a step summary of counts by rule. **The summary reports the
lex-failure count on its own line.** The exit code already separates the two, and the summary must
not re-merge them.

**`verify` fails closed.** Any changed in-scope file that does not lex fails the job. Any changed
`.html` or `.astro` file fails the job, because a sweep PR touching one is a scoping error. The
`--in-scope-only` flag is the release valve when a sweep PR must carry an unrelated file.

### 6.8 Test strategy

**Fixtures are inline Go strings. There is no `testdata/` directory.**

This is not a style preference. `CLAUDE.md` records that every `TestCorpusExpectation` in
`internal/measure/*/corpus` and `internal/custody/corpus` fails in the container, because
`.gitattributes` checks goldens out as CRLF. A `strip` before-and-after golden pair is exactly that
shape. A `testdata/` fixture would inherit the same failure and would train an agent to ignore a red
test. An inline string carries the line endings the test author wrote.

Three layers:

1. Table tests per rule, with inline fixtures.
2. A **corpus test** that runs `Lex` over every in-scope file in the tree and asserts zero lex
   failures. This turns §5.2's per-surface numbers into a regression gate.
3. A **property test**: every `strip` fixture's output passes `verify` against its input.

The SQL cross-check (§5.5) is not a unit test. It is a one-time gate on the first SQL sweep ticket.

### 6.9 CI

**One workflow file, `commentlint.yml`, with two jobs.** They share the `fetch-depth: 0` checkout,
the `setup-go` step and the path filter. One file keeps the sweep's whole CI story readable in one
place. This is a mild departure from `doclint.yml`, which is one file and one job.

- Trigger: `pull_request`, types `opened`, `synchronize`, `reopened`, `labeled`. Paths: `**/*.go`,
  `**/*.sql`, `**/*.mjs`, `**/*.ts`, `**/*.jsx`, `**/*.tmpl`, `**/*.css`, plus the tool and the
  workflow itself.
- `permissions: contents: read`. Per-ref concurrency with `cancel-in-progress`.
- **Job one, `lint`.** `continue-on-error: true`. It runs `lint --github --in-scope-only` over the
  three-dot diff. Advisory under ruling 5.
- **Job two, `verify`.** Gated on
  `contains(github.event.pull_request.labels.*.name, 'sweep:comments')`. **No `continue-on-error`.**
  It runs `verify --base`.

`fetch-depth: 0` is load-bearing for both jobs. A shallow clone has no merge base.

**`labeled` is load-bearing for the `verify` gate.** GitHub defaults `types` to `opened`,
`synchronize` and `reopened`. Labelling a pull request is a second API call, and GitHub builds the
`opened` run's payload two to four seconds after the pull request opens. **The label and the payload
race.** When the label loses, the `opened` run reads an empty label list, skips `verify`, and reports
a check that proves nothing.

The first stage-D1 batch measured both outcomes on 2026-09-03 (#1273):

| PR | Label applied after the PR opened | `verify` on the first run |
| --- | ---: | --- |
| #1266 | 1 s | ran |
| #1270 | 1 s | ran |
| #1268 | 12 s | **skipped** |
| #1271 | 14 s | **skipped** |

The two that lost the race closed and reopened their pull request to force a real run. A race a
session wins half the time is worse than a plain failure, because the green check looks the same
either way. `labeled` removes the race: the label itself fires the run, so the payload cannot
predate it.

A path filter on a `pull_request` event is evaluated against the pull request's own diff, not against
an event file list, so it holds on a `labeled` event the same way it holds on a `synchronize` event.

Two costs follow, and both are accepted. Any label, not only `sweep:comments`, re-runs the workflow;
GitHub offers no label-name filter on the trigger, and a re-run of an advisory `lint` is cheap. And
the per-ref `cancel-in-progress` group means a near-simultaneous `opened` run and `labeled` run
cancel one another, so a sweep PR can end with no `verify` row at all. §7.7 reads that case.

### 6.10 The `sweep:comments` label

The repo has no `sweep` label today. Labels here are namespaced: `wayfinder:`, `implementation:`,
`sec:`, `severity:`, `confidence:`. The new label is **`sweep:comments`**. It matches the convention,
and it leaves room for a later sweep of a different kind without a rename.

Creating it is a one-line `gh label create` in the first implementation ticket.

### 6.11 How ruling 15 gates

`main` is protected by a ruleset with 7 required status checks. Adding an 8th is a
repository-settings change.

**The `verify` job is not added to the required set now.** It is keyed on `sweep:comments`, it has no
`continue-on-error`, and it therefore shows a real red X on exactly the PRs ruling 15 governs. It is
silent on every other PR. This gives ruling 15 teeth without a settings change, and it is stronger
than "a human remembered to look".

Promotion to a required check is deferred. §10 parks it with the same decision that promotes
`commentlint lint` from advisory.

---

## 7. The sweep

### 7.1 The size cap

**600 comment lines and 20 files per PR.** One escape: **a single file that exceeds 600 alone forms
its own ticket and is never split.**

The cap binds the packing of the tail, not the large files. The solo-file escape already handles the
large files. 600 is the point where a keeps ledger still reads in one sitting, at 15 to 30 entries.
Above 600 the ledger stops being scannable, and the ledger is the only gate the judgment half has.

A 400-line cap is refused. It cuts the same large files but splits the tail into about 68 sessions,
21 more than 600 buys, for no extra review value.

### 7.2 The four stages

Four stages. **Each stage blocks the next.**

**Stage A. Prerequisites.** In this order:

1. The `commentlint` binary (§6). `strip` must exist before stage B runs.
2. The `commentlint.yml` workflow (§6.9), the `sweep:comments` label (§6.10), and the `adr-gap` label
   (§8.9). A sweep PR cannot carry ruling 15 without the workflow.
3. The `CLAUDE.md` amendment (§9). A sweep session is itself an agent. It reads `CLAUDE.md` every
   session and the SPEC almost never, so the amended rule must be in `CLAUDE.md` before the first
   sweep.

**Stage B. The mechanical PR.** One tree-wide PR. `commentlint strip` runs over all in-scope Go, and
`verify` proves token identity. **This PR is exempt from the §7.1 size cap.** Its correctness rests
on the equivalence check and on the §3.9 sample gate. Human reading is not the gate. That is what
"mechanically decidable" means. Reviewing the same class 47 times buys nothing.

**Stage C. The pilot.** One ticket: **`internal/measure/resolutionwalk`**. It fits one ticket. It
carries a package doc, exported and unexported docstrings, and 47 citation lines. It therefore
exercises every class the rubric rules on. It also sits beside a golden corpus, so it proves the
§7.6 rule in the same PR.

**The pilot blocks every other judgment ticket.** If the §4 rubric or the keeps-ledger format
misfires, it misfires once.

**Stage D. The sweep.** By surface family, ascending size inside each family:

1. **D1. Go.** Stage B has already cleared it, so its judgment residue is the smallest per line and
   the rubric is proven on it.
2. **D2. SQL** (`db/queries` only). One homogeneous shape, the sqlc `-- name:` block, so it is nearly
   a second mechanical class. **The §5.5 cross-check is a precondition of the first D2 ticket.**
3. **D3. Web assets** (`.mjs`, `.ts`, `.jsx`, `.tmpl`, `.css`). Smallest and least load-bearing.
4. **D4. The ADR-gap triage** (§8.10). It blocks on every other stage-D ticket.

`strip` is Go-only in v1, so families D2 and D3 are 100% judgment with `verify` as the only automated
gate.

### 7.3 How `cmd/web` sub-slices

**By file, in descending comment-line order, greedily packed to the cap.**

Named feature areas are not a separate boundary. `cmd/web` is already one file per feature area
(`auth.go`, `seeds.go`, `scans.go`, `settings.go`).

**Production files and test files do not ride in the same ticket.** Pairing `handlers.go` (658 lines)
with `handlers_test.go` (404 lines) gives 1,062 lines, well past the escape threshold. Sweep
`cmd/web` production first, then `cmd/web` tests, so the test tickets run with the production rubric
already proven.

`cmd/web` holds 13,952 comment lines across 139 files. 62 production files hold 9,439 lines and 77
test files hold 4,513.

### 7.4 The cut list is computed after stage B

**The cap binds what a ticket touches, which is the judgment residue. The residue is measurable only
once `strip` has run.** So this SPEC states the packing rule and an indicative table. `/to-tickets`
runs `commentlint lint` against the post-stage-B tree to get the real numbers.

**The packing rule.**

1. Walk the packages of one surface family in ascending residue order.
2. Add the next package to the current ticket while the ticket stays at or under 600 lines and 20
   files.
3. Never split a package across two tickets, unless the package alone exceeds a cap.
4. Never merge two surface families into one ticket.

**Indicative sizing, measured before stage B.** Treat it as an order-of-magnitude guide.

| Stage | Comment lines | Tickets at 600 |
| --- | ---: | ---: |
| A. Prerequisites | n/a | 3 |
| B. Mechanical PR | ~8,200 removed | 1 |
| C. Pilot | 334 | 1 |
| D1. Go residue | ~20,100 | ~34 |
| D2. SQL (`db/queries`) | 1,735 | 3 |
| D3. Web assets | ~2,650 | ~5 |
| D4. ADR-gap triage | n/a | 1 |
| **Total** | | **~48** |

Four `cmd/web` files exceed 600 before the pass and would take the solo escape: `auth.go` (797),
`devfixtures.go` (764), `handlers.go` (658), `settings.go` (616). **Stage B may drop all four below
the cap.** This is the clearest evidence for computing the list after stage B.

The 20-file cap does not bind the Go tail. The largest tail pairing, `internal/retention` plus
`internal/measure/tlsacceptance`, is 542 lines across 14 files.

**A split package costs reasons at the ticket boundary.** Rule 3 permits the split where the package
alone exceeds a cap, and §7.5 rule 1 permits the halves in parallel. Neither prices it. Three splits
are measured, and each PR body names the reasons that ticket dropped.

| Split | Tickets | Reasons dropped |
| --- | --- | ---: |
| `internal/measure/connectoutcome`, two ways | #1182, #1184 | 5 |
| `internal/custody`, three ways | #1186 (2), #1187 (7), #1188 (9) | 18 |
| `internal/queue`, four ways | #1197, #1198, #1199, #1200 | 35 |

The four-way split is the first of one package.

**A split can leave a file with zero survivors.** Two of #1188's seven files, `coverage_test.go` and
`reachability_test.go`, ended with none, because every rule they state lives verbatim in a file
another ticket owns. That is the correct §4.1 outcome. It means a reviewer must check that the
owning ticket kept the rule.

**A test-from-production split costs more than a production split.** #1200 dropped 24 reasons at the
ticket boundary, the largest of any ticket in seven batches. Seven of its twenty files ended with
zero survivors. The cause is structural. A `_test.go` docstring restates its production file's
header sentence for sentence. The three-way `internal/custody` split cost 18 across three tickets,
and this one ticket cost 24. **Keep a test file in the same ticket as the production file it
exercises, where the size cap allows it.**

**The closing ticket of a split keeps least.** Every production home is occupied by the time it
runs, so its files most often end empty. Four of #1201's five files kept nothing. A reviewer reading
the closing ticket should expect that shape rather than read it as a lazy sweep. §4.6's
test-local-hazard rule pulls the same way, in proportion to how much the test builds: #1200 got six
survivors from fourteen test files and #1201 zero from three, and both are correct.

**A third boundary shape: the cross-package collapse against `main`.** The rule a ticket drops may
be held by a survivor a merged sweep already left on `main`, not by a sibling in the split. Nothing
is dropped for an owner to catch, because the owner is on `main` already. #1209 collapsed 35 of its
86 deletions that way, #1208 14, #1207 10 against `internal/signal`, #1211 10 and #1210 4.

**The PR body's boundary table must separate two rows: reasons dropped for a sibling ticket's owner
to catch, and reasons collapsed against a survivor already on `main`.** Without that split a
reviewer cannot tell a correct collapse from an over-deletion. #1208 dropped **zero** of the first
kind and 14 of the second. #1207 supplied the table this rule is written from.

### 7.5 Parallel sweeps

1. **Parallel is allowed when two tickets touch disjoint file sets.** §7.4 keeps a package whole
   where the size cap allows it. Where §7.4 splits a package, it gives each file to one ticket, so
   the file sets stay disjoint by construction. §7.4 prices the boundary cost of that split.
2. **At most 4 sweep PRs are open at once.** The strict up-to-date policy means every merge
   re-triggers CI on every other open branch. Four bounds that churn.
3. **Never hand-resolve a conflict in a sweep PR.** Close the branch and re-run the ticket fresh from
   `main`. A hand-resolved comment hunk breaks the only guarantee the equivalence check gives.
4. **Update every other open branch after each merge.** `gh pr update-branch` does not exist in
   `gh` 2.45.0, the version on the dev machine. The command runs `gh pr --help` and updates
   nothing. Run this instead:

   ```sh
   gh api --method PUT repos/winniel123/verge-asm/pulls/<n>/update-branch
   ```

   `docs/agents/issue-tracker.md` records the trap in "The `gh pr update-branch` trap".
5. **Hand a cross-ticket fact to the sibling tickets.** A split package makes a wrong citation a
   package-wide fact, and each instance lands with a different agent. Where a ticket repairs a
   citation, or drops a reason at the boundary, its PR body names the token or the reason. §4.7
   carries the citation half. #1196 repaired `(ADR-0027 §7)` five times in `crtsh.go`, and the same
   token then appeared in `pure_test.go:64` and twice in `crtsh_test.go`, owned by other tickets.
   #1196 dropped "a `Worker` built without `WithMessages` writes no message" into #1194's half, and
   #1194 kept it and sharpened it.
6. **Measure a package doc in your own file set. Never inherit a sibling PR's claim about one.** PR
   #1314 stated that every file in `internal/queue` opens `package queue` on line 1, so §4.8 never
   applies. That is wrong. `queue.go:1-7` is a real `package-doc` block, and `queue.go` is the only
   file of the package's 51 that has one (#1198). The claim was passed to all four agents of that
   batch as fact, and only luck stopped a mis-sweep.
7. **A batch brief is not evidence.** Rule 6 states the package-doc case, and the same failure
   recurred in batch 8. The brief named `docs/adr/0116` as a live `SPEC-CHANGE` hit and told all
   four agents to read it and re-cite it. #1202 found the withdrawn status line first. #1203 and
   #1204 were corrected mid-run, and #1203 had already shipped a survivor citing a withdrawn rule.
   **An agent verifies a citation target's status line itself** (§4.7).
8. **Give every parallel agent a per-ticket scratchpad path.** A shared `scratchpad/gap.md` or
   `scratchpad/pr.md` collides silently. #1202's file overwrote #1204's, and #1204 briefly published
   #1202's gap content to issue #1333 before catching it. No merged artefact was affected. A batch
   brief names the path, for example `scratchpad/<ticket>/gap.md`.

### 7.6 The golden corpora

`internal/measure/*/corpus` and `internal/custody/corpus` hold about 857 Go comment lines across 7
packages. **They need no special slicing. Sweep them like any other package.**

The CRLF half is settled twice over. A token-stream check treats a carriage return as whitespace, so
a CRLF working tree would not affect the check. And the dev machine no longer produces one:
`.gitattributes` is `* text=auto`, and the Linux machine checks every file out LF.

**One rule covers the residual risk: a sweep PR never regenerates a golden.** A
`TestCorpusExpectation` failure is a real regression. A sweep session must treat it as one, and must
never "fix" it with `-update`. An earlier version of this section called those failures known and
pre-existing. They were a CRLF artifact of the retired native-Windows setup, and `CLAUDE.md` now
records that the trap is gone. All four tickets in the first sweep batch ran the full suite. Every
`TestCorpusExpectation` passed, in the 5 corpus packages those tickets reached and in CI's
`golden-corpus` job.

**A corpus package's `Row.Claim` field makes most of its `rows.go` fail gate A** (#1174). Each row
carries a full-sentence prose claim in its own literal, and the test's failure message quotes it.
Every descriptive per-row paragraph is therefore recoverable from the declaration. **Only the guard
half survives** — why the row exists, and which silent move it catches. The other six corpus
packages carry the same field, so #1163, #1164, #1165 and the remaining corpus tickets reuse the
reading.

**Read `docs/spec/golden-corpus.md` §10.3 and §10.4 before you sweep a corpus package.** Those two
sections already state four of `internal/custody/corpus`'s comments almost verbatim. Gate A is
written against code, so those comments survive. The SPEC section settles what the survivor cites.

### 7.7 Definition of done

A sweep ticket is done when **six** conditions hold:

1. `commentlint verify` **ran** on the PR and is green.
2. `commentlint lint` reports zero flags on the ticket's files.
3. The PR body carries a keeps ledger (§4.9) and a gaps table (§8.6).
4. Every survivor's citation was checked to resolve.
5. `gofmt -l` ran on the ticket's own file set, and the PR body names every formatting hunk the
   ticket did not ask for.
6. A human approves the PR.

**The sweep session applies `sweep:comments` to its own pull request.** Nothing else applies it.
Stage A (§7.2) creates the label; it does not put it on a PR. An unlabelled sweep PR has no gate at
all.

**Condition 1 asks for two facts, not one.** A `verify` job that the `sweep:comments` gate skipped
also leaves the PR green, and it proves nothing. The sweep session confirms the run itself:

```sh
gh pr checks <pr> | grep -E '^(lint|verify)\b'
```

Read the word, not the colour. GitHub's check mosaic renders a skipped job the same shade as a passed
one. The command can report four states, and only the first is done:

| `verify` row | Meaning | What the session does |
| --- | --- | --- |
| `pass` | The job ran and the equivalence held. | Condition 1 is met. |
| `skipping` | The label was not on the PR when the workflow last fired. | Add or re-add `sweep:comments`. The `labeled` trigger (§6.9) fires a fresh run. |
| `pending` | The run is still in flight. | Wait, then read again. |
| *no row at all* | No commentlint run completed. It was cancelled (§6.9), or no in-scope path changed. | Re-add the label to force a run. A sweep PR with no in-scope path is not a sweep PR. |

Do not read the pipeline's exit status. `gh pr checks` exits non-zero while any check is still
pending, measured as 8 on 2026-09-03, so a zero exit is not the signal. `grep` finding nothing is
silence, not a pass.

**Three tool traps, measured on the dev machine.**

| Command | What happens | The working route |
| --- | --- | --- |
| `gh pr checks <n> --json …` | Returns a non-JSON error string on `gh` 2.45.0. | The tab-separated plain output is the only readable form. |
| `commentlint verify --base <ref>` with no path list | Exits 2 with a usage string (#1210). | Pass the ticket's own file set. |
| `gh pr edit <n> --body-file …` | Aborts on the Projects-classic deprecation and leaves the body unchanged (#1211). | `gh api --method PATCH repos/winniel123/verge-asm/pulls/<n> --input <file>` |

`docs/agents/issue-tracker.md` records the same family for `gh pr update-branch` and Projects
classic.

**Condition 2 does not prove conformance on its own.** `lint` flags only the mechanically-decidable
classes. Condition 3 is the judgment gate. Delete-by-default makes a wrong delete visible in the
diff and a wrong keep invisible. The reviewer therefore checks the keeps.

**Ratchet rule 5 leaves a hole, so §4.4 cannot be delegated to `lint`.**
`RuleCitationOverOneLine` fires on class `Citation` alone. §6.6's classifier reaches `Citation`
after the declaration-position classes and before `external-spec` and `why-note`, and matches
`ADR-\d{4}|#\d+|§\s*\d|CONTEXT\.md`. **A two-line block whose wording trips `why-note` is never
flagged, at any length.** Apply §4.4's one-line-per-reason discipline by hand. The same property has
a constructive reading, which §4.4 states: it is what makes a legitimate multi-reason uncited block
legal. `internal/scan`'s `embeddedLogList` carries three uncited reasons on three lines and draws
zero `lint` findings (#1189).

**Condition 4 has no mechanical check.** The ratchet provides none. §4.7 carries the four tests a
citation must pass, and the measured false resolves. Neither a file-existence check nor a bare token
grep is enough. **The check needs the issue API**, because a deleted `#nnn` returns HTTP 410 and
nothing under `docs/` distinguishes it from a live one. It also needs the issue **title**, because a
design-collision id's number resolves to an unrelated live issue. Four defects have already landed
on `main` in merged sweep output: two citations repaired by **#1328**, the `ADR-0081` misattribution
in **#1354**, and a false `.DesignTokens` claim in **#1355**.

**A citation repair re-runs §4.4's length check.** A repair lengthens the line, and the
100-character cap binds after `gofmt`. Two of #1328's three repairs measured 110 and 103 runes
naively and had to be reworded to 97 and 99. A ticket told to "change a pointer and nothing else"
cannot honour that literally without landing a fresh §4.4 defect. Nothing mechanises the re-check.

**Condition 5 keeps a formatting hunk out of a sweep diff.** No workflow runs `gofmt`, and five
tracked Go files were unformatted on `origin/main` at `172bbc1`. #1188 deleted eight trailing labels
whose alignment run had drifted a column, which reformatted `internal/custody/egressguard_test.go`
as a side effect. The equivalence check still passed, because a token comparison does not see
whitespace. §4.5's no-reformat rule is therefore not enforced mechanically. But the diff carries a
formatting change the ticket did not ask for, and a reviewer may read it as scope creep. Say in the
PR body when a hunk is pre-existing drift rather than the ticket's own change. **#1311 tracks the
remaining files, and the question of a CI `gofmt` gate is a separate call from this SPEC.**

**The sweep is done when every ticket in the cut list is merged.** There is no global re-audit and
**no residual-line-count target**. A line target would push a session to delete a keep to reach it.

---

## 8. The ADR-gap protocol

### 8.1 The term

**The thing recorded is an ADR gap.** A decision is stated in the code, no ADR states it, and the
sweep is about to delete the only statement of it.

`CONTEXT.md` already spends "candidate" on four unrelated measurement concepts, so "salvage
candidate" would collide with the ubiquitous language. "ADR gap" names the defect, not the process.
It also reads correctly in both directions. An agent files a record against a gap, and the gap
closes when an ADR lands.

**The term is not added to `CONTEXT.md`.** `CONTEXT.md` is the product domain glossary. This is a
repo-hygiene process term, so this SPEC defines it.

### 8.2 The trigger

A block opens an ADR gap only when **all three** gates hold.

| Gate | Test |
| --- | --- |
| **A — decision** | The comment asserts a rule someone chose. A hazard, a cost note, or an external constraint survives silently and opens nothing. |
| **B — scope** | The rule binds code **outside the file the comment sits in**. A rule local to one function never promotes. |
| **C — production** | The block is not in a `_test.go` file. |

Gate A alone is unbounded against a ceiling of 1,373 blocks, and thirty sweep sessions would read it
thirty ways. **Gates B and C are what make the reading repeatable.** Every ADR on disk states a
cross-cutting rule, so gate B is what an ADR already is on this repo. Gate C holds because a decision
worth an ADR also appears in the production code the test pins.

**Expected yield: 250 to 340 gap-bearing blocks.** A 25-block sample judged against both gates passes
at roughly 40%. Treat that rate as a rough read, not a measurement.

### 8.3 What suppresses a record

**A block that cites any durable written source opens no gap.** The suppressing set is an ADR, an
issue, a `docs/spec` file, a section reference `§`, or `CONTEXT.md`.

**A sweep agent must not re-derive this set.**

98 of the 846 candidate production blocks (12%) carry a citation the crude screen missed. The set is
82 bare `§` citations, 17 to `CONTEXT.md`, and 10 naming a spec file. Opening a record for each
would file about 98 issues whose whole content is "this rule is written in a different file". That
is a docs-refactor argument, and §1.4 excludes the ADR corpus's own health.

§4.4 already handles these correctly. The comment keeps its citation and is compressed.

**Three measured shapes defeat the suppressing rule mechanically.** In each, the agent records the
gap and states why.

1. **A citation that names a source in order to overturn it.** #1180's gap 1 cites ADR-0024 in order
   to overturn it, and `SPEC-CHANGE.md`, the document that recorded the overturn, no longer exists.
   The citation points at the position the rule contradicts, and the suppression above fires
   mechanically. The agent recorded the gap anyway, in the PR body and in **#1300**. A dangling
   citation does not merely fail to resolve. It can invert the meaning of the surviving line.
2. **A citation to a retired document.** §4.7 gives the three dangling families and their routes.
   Where the third family applies and nothing on disk states the rule, the reason goes uncited under
   §4.7 **and** the gap is recorded.
3. **An ADR that states the rule only in its Context, quoting the code it is about to lose.**
   ADR-0134 states the "drop an unattributable row" rule only in its Context, as a blockquote of
   the code's own comment. The ADR then says "the same rule binds here". Its Decision never rules
   it. The citation resolves textually while the ADR quotes the very comment the sweep deletes.
   #1197 recorded the gap anyway, with the reasoning, in **#1326**.

**A dangling citation is not evidence that the rule is unwritten.** The suppressing rule fires on a
citation that resolves, so a dead token passes straight through it and the gap gets recorded. Shape
2 above is right that the reason goes uncited. It is not a licence to skip the search. **Search
§4.7's five-place surface before you record the gap.** #1204 filed the never-fabricate rule as gap 2
of **#1333** and `docs/adr/0110` line 74 states it. #1205 moved three of its four gap candidates out
of the list the same way, using `docs/guides/`. #1209 filed nothing at all: ten candidates, and
every rule that passed §8.2's gates was already written.

**A deleted issue does not suppress a record, and neither does a collision id.** The rule above says
"an issue" suppresses. A `#nnn` whose issue was deleted returns HTTP 410, and nothing under `docs/`
distinguishes it from a live citation (#1203). A design-collision id resolves to a live issue that
is not the rule's source (§4.7). Query the issue API and read the title before you let a `#nnn`
suppress a record.

### 8.4 The container

**One `adr-gap` issue per sweep ticket.** It lists every gap that ticket found. The backlog is bounded
at about 30 issues.

Three alternatives are refused:

1. **One issue per gap block** gives 250 to 340 issues, which is the problem this protocol prevents.
2. **One issue per distinct rule** is the right unit and is not buildable. Up to 4 parallel sweep
   PRs run across roughly 30 independent sessions. No session can see another's gaps, so dedup at
   capture time is guesswork.
3. **A single whole-sweep issue, or an appended file such as `docs/adr/gaps.md`**, puts every
   parallel PR on one shared write target. For a file that is a merge conflict, and §7.5 forbids
   hand-resolving one, so a conflict costs a re-run of the whole ticket.

Dedup moves to triage, where the whole population is visible at once (§8.10).

### 8.5 The record's fields

Each entry in the issue holds **five** fields.

| Field | Content |
| --- | --- |
| Rule | The rule in one sentence, in the agent's own words. |
| Symbol | The enclosing declaration's name. |
| Location | `file:line`, read at deletion time. |
| Text | The deleted comment, verbatim. |
| PR | The sweep PR's number. |

**Issue title:** `ADR gaps: <sweep ticket scope>`, for example `ADR gaps: internal/queue`.

**The PR number replaces the commit SHA.** The commit that deletes the comment does not exist when
the agent writes the record. Squash-only merges in `CONTRIBUTING.md` then rewrite every SHA an agent
could record. A `git log -S` against a dead SHA finds nothing. The PR number survives the squash and
reaches the diff, the keeps ledger, and the review conversation.

**The Rule field is the one downstream work depends on.** §8.10 dedups by comparing rules, not texts.
`internal/queue/reaper.go`, `internal/retention/observation.go` and
`internal/retention/transcript.go` state one worker-loop error rule, uncited, in three wordings. A
triager comparing verbatim texts sees it three times. A triager comparing rule sentences sees it
once.

**The record keeps the verbatim text as well as the locator.** A locator alone bets that a future
author will chase it. Citations already rot here: the standing fact is 88 distinct ADRs cited
against 133 on disk.

A free-text "why it looked ADR-worthy" field is refused. The `adr-gap` label already asserts that, and
no downstream step reads it.

### 8.6 The PR body's gaps table

**Every sweep PR body carries a gaps table: rule sentence and `file:line` only.** The other three
fields stay in the issue.

The verbatim deleted text is already in front of the reviewer, as a deletion in the diff under
review. Reproducing it in the PR body is noise at the moment attention is scarcest. In the issue it
is not noise, because the issue outlives the diff and a future ADR author has no diff open.

The table is what lets the reviewer dismiss a batch in one comment without leaving the PR. **The
agent files. The human ratifies at review.**

### 8.7 The zero case

**A sweep ticket that finds no gaps files no issue, and its keeps ledger states `ADR gaps: none`.**

Silence cannot separate two cases: the agent applied the trigger and found none, or the agent never
applied the trigger. Across about 30 sessions that difference is the whole integrity of ruling 9
itself. The explicit line costs one line and is the only evidence the reviewer gets.

**An empty issue is refused.** It pollutes the backlog §8.10 must read.

**A ticket body may say otherwise, and this section wins.** Every open sweep ticket body carried an
unconditional "open one `adr-gap` issue for this ticket" in step 7, written before this section
existed. #1201 and #1209 both reached zero gaps, both resolved the conflict this way, and both named
it in the PR body. The bodies are amended. Where any instruction outside this SPEC asks for an empty
issue, refuse it and say so in the PR body.

### 8.8 An `adr-gap` issue is never a sub-issue of a map

**Do not link an `adr-gap` issue as a sub-issue of the wayfinder map or the implementation map.** It
carries `Found by #<sweep ticket>` in its body. §8.10's ticket finds the backlog by label.

This is a prohibition, not an omission. `docs/agents/issue-tracker.md` defines the frontier as the
map's open sub-issues, minus those with an open blocker or an assignee. An `adr-gap` issue is open,
unblocked and unassigned by construction. Linking it as a sub-issue therefore puts **every gap record
on the implementation frontier permanently**, and a later `/implement` session picks one up as its
ticket. The map also never closes. A map closes when every child closes, and a gap closes only when
someone writes an ADR. §1.4 excludes that work.

A sweep agent reaches for "child issue of the map" by pattern. This is the trap that needs saying.

### 8.9 Labels

**On filing: `adr-gap` plus `needs-triage`.** On disposal: remove `needs-triage`, then apply either
`ready-for-human` or `wontfix`.

`adr-gap` is new and must be created in stage A (§7.2). The rest is the canonical vocabulary in
`docs/agents/triage-labels.md`. A gap record is ordinary triage input, so it needs one new noun, not a
new vocabulary. A `sec:*`-style family is refused: that pattern earns its shape from a category axis,
and ADR gaps have one category.

### 8.10 Disposal

**The last ticket of stage D triages the whole `adr-gap` backlog** (§7.2, D4). It is an
implementation-map ticket, so it has an owner and a scheduled moment. It does four things:

1. Read every open `adr-gap` issue by label.
2. Dedup by **rule sentence**, not by text. Collapse duplicates into one surviving record that lists
   every location.
3. Close the noise and the duplicates, labelled `wontfix` or as duplicates.
4. Label each survivor `ready-for-human` and remove `needs-triage`.

The ticket is done at a triaged, deduplicated, `ready-for-human` backlog. **Authoring the ADRs stays
out of scope** (§1.4).

Per-PR triage is refused for two reasons. It cannot dedup, because a reviewer sees one PR. And it asks
a reviewer to judge ADR-worthiness while reviewing a deletion diff. A stated cadence with no ticket
rots.

---

## 9. The `CLAUDE.md` replacement text

### 9.1 The text

The section keeps its `## Comments` heading and its position as the last section of the file. Stage A
applies this text verbatim.

````markdown
## Comments

Write a comment only when it passes both gates:

1. **Unrecoverable** — a competent reader cannot recover the fact from the declaration, its body, and its callers.
2. **External cause** — the fact names a decision, a constraint, a hazard, a cost, or a rejected alternative outside this code.

When you cannot decide, write nothing.

A surviving comment takes the form `// <reason clause> (ADR-nnnn §x.y, #nnn)`, 25 words or fewer. The citation is optional. Put it beside the statement its reason is about. Declaration position stays empty.

A machine directive (`//go:`, `-- +goose`, `eslint`) is not a comment. Write one when the tool needs it.

Restating what the code does fails gate 1. Narrating a change ("updated to handle X") fails both. Explain a change in your response.

When you edit code, delete a comment the edit makes redundant or wrong. Write a comment I explicitly request.
````

15 lines, against 14 today.

Two wordings are load-bearing:

- **"Declaration position stays empty"** carries §4.5 in four words. It is what stops §3.5 flag rule
  1 firing on the sweep's own output. It also reads on SQL and CSS, where "the function body" would
  not. "Declaration position" reaches an in-body `var` or `const` as well (§4.5), so the four words
  cover that case without naming it.
- **"is not a comment"** does the directive job with no pattern list. The patterns stay in §2.3.

The replacement holds **zero prohibitions**. Today's section holds five and never states the target
behaviour. An agent has to invert five bans to learn what a good comment is. The two gate-failure
sentences diagnose rather than ban.

**Scope is stated surface-agnostically.** The rule binds every source file this repo maintains,
whatever the language. It states no file-type list. Every §1.4 exclusion is a reason about
**deletion**, and none of them licenses writing a new comment. Copying that list inline would drift,
and would invite the misread that `prototypes/` is a comment-friendly zone.

### 9.2 The length ratchet

**15 lines, and adding a line to this section requires removing one.**

A bare number is a fact about one edit, and it goes stale the first time someone has a good reason to
add. The ratchet binds every future edit. It costs one sentence here rather than a line in
`CLAUDE.md`.

### 9.3 Two jobs, one reader

**The amendment serves the authoring agent only.** It does not serve the sweep agent.

Ruling 14 argues that an agent reads `CLAUDE.md` every session and a SPEC almost never. That argument
describes the agent deciding whether to **write** a comment. A sweep agent is handed its rubric by its
own ticket, and the sweep ends while the context load would not.

Three rulings bind **both** jobs, so they stay inline: the two gates (§4.1), the rewrite form (§4.4),
and body placement (§4.5). The rest of §4 stays here.

**The rule stays in `CLAUDE.md`.** It does not move to `docs/agents/`. A context pointer's wording
decides when an agent reaches the material. This rule's trigger is "whenever you write code", a
branch that fires every session. A pointer with no discriminating condition is a pointer that
belongs inline. The five existing `docs/agents/` docs each carry a real condition. This one does
not.

**`commentlint` is not named in `CLAUDE.md`.** The `lint` job is advisory, so its existence changes
nothing an agent writes. That would be a no-op line paying load every turn. The `verify` job's
`sweep:comments` gate is real, but it fires only on sweep PRs, which are implementation tickets that
read this SPEC.

### 9.4 The two carve-outs stay sweep-scoped

§4.3 keeps `package-doc` at a 3-line cap and keeps `.d.ts` field prose whole. Both are exceptions to
the two gates: a package doc stating purpose plausibly fails gate B, so an agent applying §4.1
literally writes none.

**Neither goes inline.** New packages are occasional, and new `.d.ts` files are close to never. The
design-system handoff that produced them was retired 2026-08-28 (ADR-0116). The failure costs one
missing 3-line block, recoverable at any time. A permanent line every session is the wrong side of
that trade.

The regression path is checked and closed. The last line of §9.1's text fires on "redundant or
wrong", and a package doc stating purpose plus an ADR is neither. A later session will not delete what
the sweep kept.

### 9.5 What was pruned

| Today's text | Verdict |
| --- | --- |
| "Never write comments that describe WHAT the code does" | **Folded into gate 1.** It restates the same test. |
| The change-narration paragraph | **Compressed to one clause** in the gate-failure line. §3.5 flag rule 3 enforces it mechanically. |
| "delete any existing comments that are redundant or inaccurate" | **Kept, rewritten positive.** This is the ratchet that holds the tree clean after the sweep. |
| "Comments I explicitly request" | **Kept.** One clause, and it is the escape hatch. |

Three of §4.4's four rewrite constraints are pruned as restatement. "States the cause, not the
behaviour" is gate B restated. "Does not open with an identifier name" is a docstring shape the gates
already kill. "100 characters or fewer" follows from 25 words, and the tree already wraps at 93.

---

## 10. Not yet specified

- **Promotion of `commentlint` from advisory to required.** Two halves of one decision: whether `lint`
  becomes blocking, and whether `verify` joins the 7 required status checks. Promoting `verify` would
  drop the `sweep:comments` label gate (§6.11). Neither half is decidable before the sweep measures a
  false-positive rate.
- **Whether tooling generates the keeps ledger.** §4.9 fixes the requirement. Whether `commentlint`
  emits a ledger skeleton, or the sweep agent writes it by hand, is an implementation choice. The
  implement effort decides it.
- **The ADRs on disk that no comment cites.** 35 of 134 are uncited by any in-scope comment, and 45
  are uncited by production Go alone. The sweep may show that some are dead. That is a docs question,
  and it may belong to a separate effort.
- **Delete semantics for the six non-Go surfaces.** §6.5 records Go and `.tmpl`. Each remaining row is
  measured when its surface's sweep is scheduled.

---

## 11. Handoff

This SPEC is the destination of map #1054.

The next effort is `/to-tickets` on this document, then implement. That effort builds the tool and
runs the sweep. It starts from §7.2 (the four stages) and §7.4 (the packing rule), and it computes the
real cut list after stage B.

**One ticket per session.** A sweep session takes one ticket, opens its PR, and stops.

---

## Appendix A. Worked examples

Fifteen blocks, read from `main` at `b548431`. **A sweep agent reads this appendix more than any other
part of this SPEC.**

| # | Location | Class | Verdict | Reason |
| ---: | --- | --- | --- | --- |
| 1 | `cmd/web/main.go:1` | `package-doc` | Keep as-is | Already at the §4.8 cap. States purpose and ADR-0001. Names no symbol. |
| 2 | `cmd/prober/main.go:45` | `citation` | Rewrite | Sentence 1 restates `connectoutcome.Run`. The pacing constraint is the only cause. |
| 3 | `cmd/web/backup.go:178` `// RFC3339 UTC` | trailing `external-spec` | Keep | The wire format is unrecoverable from `string`. External spec, two words. |
| 4 | `cmd/web/backup.go:174` `// always "manifest"` | trailing | Delete | States a value. Names no cause. |
| 5 | `cmd/web/backup.go:175` `// archive format version, not schema` | trailing | Delete | Ambiguous. It disambiguates from the next field but names no cause. §4.2 breaks the tie. |
| 6 | `cmd/web/addressscopecensus.go:45` | `docstring-unexported`, 9 lines | Delete, salvage one line | Paragraph 1 restates the return. Paragraph 2 names a cross-module rule. |
| 7 | `cmd/web/addresscap_test.go:64` | test `docstring-exported-conventional` | Rewrite | Cites ADR-0127, so §4.6 keeps it. It opens with the identifier, which §4.4 forbids. |
| 8 | `cmd/web/addressscopecensus_test.go:87` | `docstring-exported-other` | Delete | Uncited. The test name already states the assertion. |
| 9 | `cmd/web/addresscap_test.go:39` | `step-narration` | Delete | Uncited. Restates the two lines below it. |
| 10 | `cmd/web/addresscap_test.go:47` | test `citation` | Rewrite | Cites ADR-0127. Drop the leading step narration. |
| 11 | `internal/queue/withdrawal.go:99` | `prose-other` | Keep, compressed | Explains why the read sits below the guard. Moving it above is a plausible wrong edit. Opens no issue. |
| 12 | `cmd/web/inbox_test.go:82` | test `prose-other` | Delete | Uncited. Restates the assertion below. |
| 13 | `cmd/web/devfixtures.go:281` | `change-narration` plus two reasons | Rewrite to two lines | Drop the history. Keep both constraints, one per line. Opens a follow-up issue. |
| 14 | `cmd/web/asset_test.go:31` `// ports census` | trailing `short-label` | Delete | A label with no cause. Agent-judged under §3.4, not mechanical. |
| 15 | `design-system/components/display/Sparkline.d.ts:7` | `.d.ts` field | Keep | §4.3 carve-out. The default is unrecoverable from `color?: string`. |

Six keeps, six deletes, and three rewrites that are compressions rather than deletions. The split
matches the survey's headline: **the sweep's dominant act is compression, not deletion.**

`cmd/web/seedfixtures.go:122` holds `//nolint:errcheck`. §2.3 protects it, and no rule in this rubric
reaches it.

The rewrites in full:

```go
// 2  before: three lines naming the leaf, then the constraint
// after:
// Paced by the §3.3 safety limiter, which never changes a verdict (ADR-0021).

// 6  before: nine lines above func addressScopeSharedEdges
// after: declaration position empty. Inside the body, above the first statement:
// Absence and zero are the same here — the open-then-label absence rule
// (custody.Estate.AddressScopeCensus).

// 7  before: TestAddressCapHasNoUpperBound covers ADR-0127's load-bearing ruling: ...
// after:
// ADR-0127: nothing gates a cap above the operator's own — beyond 2^32 is priced, not gated.

// 11 before: two lines
// after:
// The estate is built only where there is something to withdraw. The steady state answers empty.

// 13 before: six lines mixing history with two constraints
// after:
// The design-system profile example is consistent only at TOTP-ON, so seed it verbatim.
// The dev session mint bypasses the password and TOTP challenge, so no secret is needed.
```
