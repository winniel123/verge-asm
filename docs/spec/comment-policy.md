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
normative. 397 lines depend on them. Deleting one changes the build or the generated code.

| Surface | Patterns |
| --- | --- |
| Go | `//go:`, `// +build`, `//nolint`, `//lint:`, `//revive:` |
| SQL | `-- name:`, `-- +goose` |
| JS and TS | `eslint`, `@ts-check`, `@ts-expect-error`, `prettier-ignore`, `c8 ignore`, `@jsx` |
| CSS | `stylelint-`, `postcss-` |

**To add a pattern, add it to this table.** A directive line forms its own block, so a directive
never absorbs the prose beneath it.

**This list is complete for this tree, measured 2026-09-01.** Four findings support that:

1. No cgo. The tree holds no `import "C"`, no `//export` and no `#cgo`. That removes the most
   dangerous comment class in Go.
2. No `//go:generate`.
3. SQL holds 384 of the 397 directive lines: 250 `-- name:` openers and 134 `+goose` markers. Go
   holds 11 and `.jsx` holds 2.
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

Three properties of the screen:

- **It is block-wide.** One signal anywhere in the block withholds every line of that block.
- **It is uniform.** It applies to every class in the delete set, `section-divider` included. A
  banner can carry a citation.
- **It is tuned loose.** The cost is asymmetric. A false keep costs agent time. A false delete loses
  a reason permanently. This list is the floor. A later revision may widen it. **A later revision
  must never narrow it.**

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

Flag rule 1 is safe against the sweep's own output, because §4.5 moves every salvaged line into the
function body. No salvaged line stays in declaration position.

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
   and `internal` reach 93 characters at the maximum.

**The degenerate case.** Where a block's only content is "this exists because ADR-nnnn says so", and
no independent constraint is recoverable, the survivor is the bare citation: `// ADR-0021 §3.3`.
Half the corpus cites something, so most of the 18,266 cited lines land in this form.

**Ruling 8 caps a citation, not a reason.** A block may hold **one line per independent reason**. No
block-level line cap applies. Worked example 13 forces this: `cmd/web/devfixtures.go:281` carries two
unrelated constraints, and neither restates the other. A hard one-line cap would delete one of them
to satisfy a format, which is the error §4.7 rejects. The discipline that prevents bloat is "every
line passes §4.1", not a number.

### 4.5 Where a salvaged line goes

**A salvaged line moves into the body. Declaration position is left empty.** The line sits above the
statement its reason is about. Where the reason governs the whole function and attaches to no single
statement, the line sits above the first statement of the body.

Two reasons:

1. It keeps §3.5 flag rule 1 from firing on the sweep's own output. That rule flags **any** own-line
   block in a Go declaration position, marker or not.
2. A reason about a `defer` or a retry belongs beside that line, not above the signature.

### 4.6 Test files

Ruling 2 makes "the ADR or issue this test pins" an explicit WHY exception. Three rules set how far
it reaches.

1. **Uncited prose that asserts what the test already asserts is deleted.** The test's name and its
   body are the assertion. A comment restating them is the WHAT that `CLAUDE.md` forbids. This
   reaches 2,613 lines of `docstring-exported-other` and 1,491 lines of test `prose-other`.
2. **A cited comment survives**, compressed to the §4.4 form.
3. **An uncited comment that states a reason is judged like any other comment**, under §4.1. The
   citation requirement governs behaviour-assertion prose only.

Rule 3 is load-bearing. `cmd/web/adr0130_contract_test.go:498` is an uncited WHY inside a test file.
It explains a classification decision. An absolute citation requirement would delete it. §4.7 keeps
it, and §4.7 wins.

### 4.7 A reason with no ADR to cite

**An uncited reason survives.** The citation in the §4.4 form is optional.

Ruling 9 exists so a sweep never stalls. It does not exist to destroy reasons. A hazard note ("this
retries because the upstream 502s on cold start") is a fact about the world. It is not an
undocumented decision. Deleting it to satisfy a citation format inverts the rule's purpose.

**A follow-up issue opens only where the comment asserts a decision** — a rule someone chose that
ought to be an ADR. A hazard, a cost note, or an external constraint survives silently and opens
nothing. §8.2 narrows the trigger further with two more gates.

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

The ledger carries a second table, the gaps table (§8.6). A ticket that finds no ADR gap states
`ADR gaps: none` (§8.7).

The volume is manageable. The salvage population is about 1,729 declaration blocks tree-wide, so a
per-package slice yields a ledger of tens. §7.1 sizes the cap against that.

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
prose beneath it.

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
| `short-label` | word count on a one-line own-line block |
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

- Trigger: `pull_request`. Paths: `**/*.go`, `**/*.sql`, `**/*.mjs`, `**/*.ts`, `**/*.jsx`,
  `**/*.tmpl`, `**/*.css`, plus the tool and the workflow itself.
- `permissions: contents: read`. Per-ref concurrency with `cancel-in-progress`.
- **Job one, `lint`.** `continue-on-error: true`. It runs `lint --github --in-scope-only` over the
  three-dot diff. Advisory under ruling 5.
- **Job two, `verify`.** Gated on
  `contains(github.event.pull_request.labels.*.name, 'sweep:comments')`. **No `continue-on-error`.**
  It runs `verify --base`.

`fetch-depth: 0` is load-bearing for both jobs. A shallow clone has no merge base.

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

### 7.5 Parallel sweeps

1. **Parallel is allowed when two tickets touch disjoint file sets.** The §7.4 packing rule
   guarantees this by construction, because no package is split across tickets.
2. **At most 4 sweep PRs are open at once.** The strict up-to-date policy means every merge
   re-triggers CI on every other open branch. Four bounds that churn.
3. **Never hand-resolve a conflict in a sweep PR.** Close the branch and re-run the ticket fresh from
   `main`. A hand-resolved comment hunk breaks the only guarantee the equivalence check gives.
4. Run `gh pr update-branch <n>` after each merge.

### 7.6 The golden corpora

`internal/measure/*/corpus` and `internal/custody/corpus` hold about 857 Go comment lines across 7
packages. **They need no special slicing. Sweep them like any other package.**

The CRLF half is settled: a token-stream check treats a carriage return as whitespace, so the CRLF
working tree does not affect the check.

**One rule covers the residual risk: a sweep PR never regenerates a golden.** The
`TestCorpusExpectation` failures in the container are known and pre-existing, exactly as `CLAUDE.md`
records. A sweep session must not mistake them for its own regression, and must not "fix" them with
`-update`.

### 7.7 Definition of done

A sweep ticket is done when **four** conditions hold:

1. `commentlint verify` is green on the PR.
2. `commentlint lint` reports zero flags on the ticket's files.
3. The PR body carries a keeps ledger (§4.9) and a gaps table (§8.6).
4. A human approves the PR.

**Condition 2 does not prove conformance on its own.** `lint` flags only the mechanically-decidable
classes. Condition 3 is the judgment gate. Delete-by-default makes a wrong delete visible in the
diff and a wrong keep invisible. The reviewer therefore checks the keeps.

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
  not.
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
// The frozen golden package is internally consistent only at TOTP-ON, so seed it verbatim.
// The dev session mint bypasses the password and TOTP challenge, so no secret is needed.
```
