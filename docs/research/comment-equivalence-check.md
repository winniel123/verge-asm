# Comment-stripping equivalence check: a feasibility spike

Wayfinder ticket [#1058](https://github.com/winniel123/verge-asm/issues/1058), map
[#1054](https://github.com/winniel123/verge-asm/issues/1054). Prototype run **2026-09-01**
against `main` at `b548431`.

Ruling 15 requires every sweep PR to prove that no non-comment byte changed. This spike built a
rough checker for all nine in-scope surfaces and measured it against the whole tree. It decides
nothing. It reports what a checker can and cannot prove, and at what cost.

**Prototype.** About 900 lines of Go plus 90 lines of JavaScript, written in one session. The
source is not in the repo. Appendix A states the method well enough to rebuild it.

---

## 1. Verdict

| Surface | Files | Verdict | Mechanism |
| --- | ---: | --- | --- |
| `go-prod` | 241 | **Sound** | `go/scanner` token stream |
| `go-test` | 203 | **Sound** | `go/scanner` token stream |
| `sql` | 106 | **Sound** | purpose-built PostgreSQL tokenizer |
| `css` | 18 | **Sound** | purpose-built tokenizer |
| `d.ts` | 109 | **Sound** | purpose-built JS/TS lexer |
| `ts` | 7 | **Sound** | purpose-built JS/TS lexer |
| `mjs` | 19 | **Sound** | purpose-built lexer, cross-checked against esbuild |
| `jsx` | 141 | **Sound with caveats** | esbuild canonical form |
| `tmpl` | 24 | **Sound with caveats** | `text/template/parse`, and a delete rule the SPEC must state |
| `html` | 24 | **Not achievable** | a comment's bytes are output bytes |
| `astro` | 6 | **Not achievable** | needs `@astrojs/compiler`, which the repo does not install |

**Headline.** 703 of the 898 in-scope files reach a sound check with no condition. A further 165
reach one under a stated condition. 30 files do not: 24 `.html` and 6 `.astro`. Those 30 hold
2,225 comment lines by the [#1055 survey](comment-corpus-taxonomy.md), which is 6.1% of the
corpus.

---

## 2. What the check is

For each file the checker builds a **skeleton**: the canonical sequence of non-comment tokens,
with protected machine directives kept as tokens. Two files are equivalent when their skeletons
match exactly.

**The skeleton is a token stream, not text.** This matters. A comment can act as a token
separator, so a text-level strip-and-compare accepts a real code change:

```sql
SELECT/*c*/a   -- strip the comment as text -> SELECTa
```

`SELECT a` and `SELECTa` are different SQL. A text comparison of the stripped forms sees no
difference, because both sides strip to the same string. A token comparison sees two tokens
against one. Every surface below therefore compares tokens.

For markup the same rule cannot apply, because markup text is output rather than tokens.
Section 5 reports what follows from that.

---

## 3. Method

The checker runs four tests per file.

1. **Lex.** Build the skeleton. A file that does not lex is unverifiable, and the count is the
   honest measure of coverage.
2. **Completeness.** Delete every unprotected comment, then rebuild. The skeleton must not move.
   A failure means the checker rejects a legitimate sweep.
3. **Soundness.** Insert one letter into the last identifier outside any comment, then rebuild.
   The skeleton must move. A failure means the checker accepts a real code change.
4. **Directive protection.** Delete one protected directive, then rebuild. The skeleton must
   move. This makes ruling 10 an enforced property rather than a convention the sweep is
   trusted to honour.

The spike measured three delete strategies. The result for markup depends on which one the
sweep uses.

| Experiment | Markup text compared as | Comment deleted as |
| --- | --- | --- |
| A | whitespace-normalized | the whole line |
| B | byte-exact | the whole line |
| C | byte-exact | the comment's byte range only |

---

## 4. Results

`comments` counts comment blocks, not comment lines. It does not reconcile with the #1055
survey. `prot` counts protected directives. `amb` counts the regex-against-division decisions
the JS lexer had to resolve.

**Experiment A — whitespace-normalized text, whole-line delete.**

| Surface | Files | lexFail | stripFail | mutMiss | dirMiss | comments | prot | amb |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `astro` | 6 | 6 | 0 | 0 | 0 | 0 | 0 | 0 |
| `css` | 18 | 0 | 0 | 0 | 0 | 114 | 0 | 0 |
| `d.ts` | 109 | 0 | 0 | 0 | 0 | 301 | 0 | 0 |
| `go-prod` | 241 | 0 | 0 | 0 | 0 | 20,620 | 11 | 0 |
| `go-test` | 203 | 0 | 0 | 0 | 0 | 7,729 | 0 | 0 |
| `html` | 24 | 5 | 0 | 0 | 0 | 487 | 0 | 0 |
| `jsx` | 141 | 140 | 0 | 0 | 0 | 1 | 0 | 0 |
| `mjs` | 19 | 0 | 0 | 0 | 0 | 392 | 0 | 68 |
| `sql` | 106 | 0 | 0 | 0 | 0 | 3,251 | 384 | 0 |
| `tmpl` | 24 | 0 | 0 | 0 | 0 | 42 | 0 | 0 |
| `ts` | 7 | 0 | 0 | 0 | 0 | 87 | 0 | 28 |

**Experiments B and C — byte-exact text.** Only the two markup surfaces move. Every other row is
identical to A, because a token stream carries no whitespace.

| Surface | B stripFail | C stripFail |
| --- | ---: | ---: |
| `html` | 19 of 19 lexed | 19 of 19 lexed |
| `tmpl` | 24 of 24 | **0 of 24** |

**The esbuild cross-check**, run separately over the JavaScript family.

| Surface | Files | parseFail | emptyCanon | fixpointFail | crossRun | crossFail |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `d.ts` | 109 | 0 | 109 | 0 | 109 | 0 |
| `jsx` | 141 | 0 | 0 | 0 | 1 | 0 |
| `mjs` | 19 | 0 | 0 | 0 | 19 | 0 |
| `ts` | 7 | 0 | 0 | 0 | 7 | 0 |

---

## 5. Per-surface findings

### Go — sound

444 files, 28,349 comment blocks, zero failures across all four tests. `go/scanner` is the whole
mechanism. It returns each comment as a token, so the checker needs no comment-finding logic of
its own.

**The semicolon hazard is real in principle and absent in practice.** A Go block comment that
spans lines acts as a newline for automatic semicolon insertion. So deleting one can change the
token stream, which is a code change wearing a comment's clothes. The completeness test found
zero such cases in 444 files. The check catches the case if it ever arises, because the inserted
semicolon is a token.

Ruling 10's protected set is small and measured: **11 directives, all in production Go**. The
prototype's pattern list is `//go:`, `//line `, `//export `, `//sys `, `//nosec`, `//nolint`,
`//lint:`, `// +build` and `// #cgo`. That list is wider than the tree needs, which is the right
direction for a pattern-class rule.

### SQL — sound

106 files, 3,251 comment blocks, **384 protected directives** (250 `-- name:` and 134 `+goose`).
Zero failures. Deleting a directive moves the skeleton in every file tested, so ruling 10 holds
by construction rather than by care.

The tokenizer handles the three PostgreSQL traps: `''` inside a single-quoted string,
dollar-quoted strings with a tag, and nested `/* */` comments. It treats a run of operator
characters as one token, which is what makes the `SELECT/*c*/a` case in section 2 fail correctly.

### CSS — sound

18 files, 114 comment blocks, zero failures. The surface is as simple as it looks. `/*! */` is
treated as protected, because a minifier preserves it. The tree holds none.

### TypeScript and `.mjs` — sound

**`.d.ts` is the surface that matters here: 109 of the 116 `.ts` files.** esbuild is the obvious
tool and it is the wrong one. It erases a declaration file to the empty string. The measurement
is unambiguous: 109 of 109 canonical forms are empty, and 109 of 109 code mutations are
invisible to it. Any design that reaches for esbuild on TypeScript loses this surface silently.

The hand lexer covers it: 109 files, 301 comment blocks, zero failures. A `.ts` file cannot
contain JSX, because TypeScript reads `<T>` there as a generic. The lexer's hardest case never
arises on this surface.

**The regex-against-division decisions are independently validated.** The lexer had to resolve
96 of them (68 in `.mjs`, 28 in `.ts`). For all 26 files, the lexer's own comment-stripped output
and the original produce identical esbuild canonical forms. A misread regex would have shifted
one of them.

### `.jsx` — sound with caveats

**No lexer can do this surface.** Inside a JSX text run, `//` and `/*` are literal characters.
The prototype's lexer refuses 140 of 141 files rather than guess, which is the correct failure.

esbuild is the mechanism. It parses all 141, its output is a fixed point (141 of 141) and it is
mutation-sensitive (141 of 141). Two caveats:

1. **A `.tsx` file would lose its type annotations**, as `.d.ts` does. The repo holds zero `.tsx`
   files, so the caveat is inert today. It stops being inert the day someone adds one.
2. The completeness evidence is weaker here than elsewhere. esbuild's output carries no comments
   by construction, so the fixed-point test is the evidence. There is no independent stripper to
   cross-check against, because building one is the problem esbuild solves.

**Five `.html` files also need this path.** They carry `<script type="text/babel">` blocks
holding real JSX. The forms card holds 2,594 bytes of it.

### `.tmpl` — sound, on one condition

24 of 24 files parse with `text/template/parse` in `ParseComments | SkipFuncCheck` mode. That
mode is the whole trick. It builds the tree without the application's function map, and it keeps
`{{/* */}}` nodes visible so their byte ranges are exact. Whitespace trim markers need no rule of
their own. The parser applies a trim to the adjacent text node, so a lost trim shows up as a
changed text node.

**The delete strategy decides the verdict.**

- Delete the comment's whole line: byte-exact comparison fails in **24 of 24** files. The line's
  own newline reaches the browser, so removing it changes what the server sends.
- Delete the comment's byte range and leave the line: byte-exact comparison passes **24 of 24**.

So the check is sound for `.tmpl`, and the SPEC must state the delete rule that makes it sound.
The residue is a blank line, which `gofmt` does not touch and a human reviewer reads as noise
rather than as risk.

No `.tmpl` file holds an HTML comment, so the `.html` problem below does not reach this surface.

One implementation note that cost time. A comment between two text runs splits one `TextNode`
into two, and deleting the comment merges them back. Join an adjacent text run before it becomes
a token. Otherwise every such file reports a false difference.

### `.html` — not achievable

**An HTML comment is output.** Deleting `<!-- x -->` removes those bytes from what the browser
receives. No delete strategy avoids it. The measurement confirms it: 19 of 19 lexable files fail
byte-exact comparison under both experiment B and experiment C. A whitespace-normalized
comparison passes. It no longer proves ruling 15's property, and it also accepts a reflowed
paragraph of visible page text.

The remaining 5 files do not lex at all, for the `text/babel` reason above.

**The stakes are lower than the 2,139 comment lines suggest.** Three measurements:

1. `design-system/designfs.go` embeds `templates/*.tmpl`, `tokens/*.css` and `fixtures/*.json`.
   **No `.html` file ships in the binary.**
2. The 24 files are 5 design-system component cards and 19 files under `prototypes/`. Both are
   development artifacts.
3. No golden file and no test in `internal/` or `cmd/` asserts on an HTML comment.

Recommendation: rule `.html` out of the sweep. Ticket #1060 may still choose to sweep it. Then
the sweep rests on human review with no equivalence proof. The SPEC should say that plainly
rather than imply a guarantee the check cannot give.

### `.astro` — not achievable

6 files. The frontmatter fence lexes as TypeScript. The template body mixes JSX expression
comments with HTML comments, and separating them needs `@astrojs/compiler`. The repo does not
install it. `docs-site/package.json` also declares no coupling to the Go application. So the
checker would reach across two build lanes to use it. Ticket #1060 already owns the
question of whether `.astro` is in scope at all. This is a reason to rule it out.

---

## 6. Caveats on the evidence

**Circularity.** For SQL, CSS and HTML one hand-written lexer both finds the comments and builds
the skeleton. A lexer that misreads a comment misreads it consistently, and the test still
passes. Four surfaces are free of this: Go and `.tmpl` use standard-library parsers, `.jsx` uses
esbuild, and `.mjs`, `.ts` and `.d.ts` are cross-checked against esbuild. **SQL and CSS have no
independent check.** SQL is the one to worry about, because it holds 3,251 comment blocks and 384
load-bearing directives. A production check should cross-check the SQL stripper against `sqlc
generate` output and a `goose` dry run.

**One reported zero is vacuous.** The `d.ts` cross-check reads `crossFail: 0`, but both sides
canonicalize to the empty string, so it proves nothing. The `.d.ts` evidence is the hand lexer
alone.

**The token check is whitespace-insensitive by construction.** It would accept a sweep that also
reformatted the file. If ruling 15 means literal byte equality outside the comments, the token
check needs a cheap second gate. One example: assert that every hunk in the diff is a deletion.
The two gates answer different questions, so the SPEC should require both.

**Line endings.** The working tree is CRLF and the repository stores LF. The token checks are
unaffected, because the lexers treat `\r` as whitespace. A byte-exact markup check must normalize
line endings first, or run only in CI.

---

## 7. Cost

The prototype took one session. A production checker is smaller, because it drops the three
experiment modes and the mutation harness, and because two surfaces are ruled out.

The per-PR cost is low. The check runs over the changed files, not the tree, and every mechanism
is a parser the repository already depends on. The Go and `.tmpl` paths need no new dependency at
all. The `.jsx` path needs esbuild, which `docs-site` already installs.

The real cost is the SQL tokenizer. It is the only component that is both load-bearing and
unvalidated, and section 6 names the two cross-checks that would fix that.

---

## 8. What the SPEC should take from this

1. Define equivalence as **token-stream identity**, not stripped-text identity. Section 2 gives
   the SQL counter-example that rules out the text form.
2. Keep protected directives **in** the skeleton as tokens. That turns ruling 10 into an enforced
   property.
3. State the `.tmpl` delete rule: **delete the comment's byte range, leave its line**. Without
   it the check fails on every `.tmpl` file. With it the check passes on every one.
4. Do not use esbuild for TypeScript. It erases `.d.ts` to nothing, and `.d.ts` is 109 of the 116
   `.ts` files.
5. Use esbuild for `.jsx`, and record that a future `.tsx` file would need a different tool.
6. Rule `.html` and `.astro` out of the sweep. Together they are 30 files and about 6% of the
   corpus, none of it shipped in the binary.
7. Pair the token check with a diff-shape assertion, so the pair proves both "no code changed"
   and "nothing but comments was touched".
8. Cross-check the SQL stripper against `sqlc generate` and a `goose` dry run before the first
   SQL sweep ticket lands.

---

## Appendix A. Rebuilding the prototype

| Surface | Comment source | Skeleton |
| --- | --- | --- |
| Go | `go/scanner` with `ScanComments` | `(tok.String(), literal)` per token, comments dropped, directives kept |
| SQL | hand tokenizer | word, number, string, dollar-quote, quoted identifier, operator run, punctuation |
| CSS | hand tokenizer | word, string, punctuation |
| JS, TS | hand lexer with a previous-token rule for `/` | word, string, template, regex, maximal-munch operator |
| JSX | `esbuild.transformSync` with `loader: 'jsx'`, `jsx: 'preserve'`, `legalComments: 'none'`, `minifyWhitespace: true` | the emitted code |
| Template | `text/template/parse`, `ParseComments \| SkipFuncCheck` | node kinds, adjacent text joined |
| HTML | hand tokenizer, with `script` and `style` bodies routed to the JS and CSS lexers | tag, text, doctype |

Three details that are easy to get wrong, all of which cost time in this spike:

1. A closing `</script>` matches an opening-tag pattern unless the pattern rejects the `/`.
   Without the rejection the rest of the document is read as script body.
2. TypeScript generics look like JSX to a lexer. Decide by file extension, never by content.
3. Mutation-test the **last** identifier in a file, not the first. Most files open with a comment
   header, so a mutation near the start lands inside a comment and reports a false pass.
