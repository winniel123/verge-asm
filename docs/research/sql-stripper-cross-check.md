# SQL stripper cross-check gate

The one-time gate that closes the §5.5 circularity risk on the SQL surface. Ticket
[#1143](https://github.com/winniel123/verge-asm/issues/1143), map
[#1131](https://github.com/winniel123/verge-asm/issues/1131). SPEC
[`docs/spec/comment-policy.md`](../spec/comment-policy.md) §5.5, §6.5, §6.8. Run **2026-09-03**
against `main` at `c6c8bf4`.

**Verdict: pass. The SQL lexer needs no widening, and the first D2 sweep ticket
([#1227](https://github.com/winniel123/verge-asm/issues/1227)) is unblocked.**

---

## 1. The risk this closes

SPEC §5.5 names one unvalidated risk in the whole design. The SQL lexer both finds the comments
and builds the skeleton, so a lexer that misreads a comment misreads it the same way twice and
`commentlint verify` still reports clean. The lexer cannot check itself.

The gate answers that with two independent parsers. Neither shares a line of code with
`internal/commentlint/surface`:

| Oracle | What it re-parses | What a lexer error would do |
| --- | --- | --- |
| `sqlc` v1.31.1, over `pg_query_go` — the real PostgreSQL grammar | the 251 queries in `db/queries` | change a column set, a param type, a row struct, a method signature, or fail outright |
| `goose` v3.27.3, and PostgreSQL 16.15 executing the result | the 68 migrations in `db/migrations` | change the migrated schema, or fail outright |

---

## 2. What was stripped

A throwaway harness read each file with `surface.SQL{}.Lex`, deleted the byte range of every
non-directive block and trailing comment, and wrote the result to a scratch tree. It is not in
the repo; §7 states the method well enough to rebuild it. No stripped SQL was committed.

| Corpus | Files | Blocks deleted | Comment lines deleted | Directives kept | Glued comments |
| --- | ---: | ---: | ---: | ---: | ---: |
| `db/queries` | 39 | 244 | 1,492 | 251 | 0 |
| `db/migrations` | 68 | 185 | 1,388 | 136 | 0 |

Every `-- name:` and `-- +goose` marker survived, counted both by the lexer and by an independent
`grep` of the stripped tree. The two agree.

1,492 deleted lines plus 251 directive lines is 1,743, which reproduces the map's D2 line-scan
estimate of 1,743 comment lines exactly.

---

## 3. The `sqlc` leg

```
go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate
git diff --numstat -- internal/db/
```

**A control run first.** `sqlc generate` over the unstripped tree leaves `internal/db` untouched:
`git status --porcelain -- internal/db/` reports 0 files. On Linux the CRLF trap
`CLAUDE.md` warns about does not fire, so there is no generated-file noise to filter.

**The stripped run exits 0.** All 251 queries re-parse under the PostgreSQL grammar. A lexer that
had eaten a real token would have produced a syntax error here, and did not.

### 3.1 The acceptance criterion as written cannot pass

`git diff --numstat -- internal/db/` after the stripped run is **not** empty. 39 of the 44
generated files move. This is not a lexer error. `sqlc` derives the Go doc comment on each
generated method from the SQL comment above its `-- name:` directive, and it embeds the query
text verbatim in a `const`, blank line included. Deleting a SQL comment therefore moves generated
bytes by design:

```diff
 const getInstanceHealth = `-- name: GetInstanceHealth :one
+
 SELECT
     pg_database_size(current_database())::bigint AS db_size_bytes,
```
```diff
-// The instance-health tab's live database facts (#633, WORK-ORDER-DOGFOOD-R1 item 3):
-// ...
 func (q *Queries) GetInstanceHealth(ctx context.Context) (GetInstanceHealthRow, error) {
```

`querier.go` moves by 0 insertions and 1,482 deletions: a pure doc-comment removal, with no
interface line touched.

**The criterion needs restating.** The sound comparison drops what a comment delete is allowed to
move — Go comment lines, whole-line SQL comments inside an embedded query, blank lines, and the
embedded query text itself — and requires everything else to match byte for byte.

### 3.2 The comparison that does hold

| Comparison | Result |
| --- | --- |
| Generated tree, minus comment lines and blank lines | identical across all 44 files |
| Generated tree, minus the embedded query text (5,959 lines compared) | identical |
| `func (q *Queries)` methods | 252 against 252 |
| `Querier` interface members | 251 against 251 |
| Query `const` declarations | 251 against 251 |
| `type` declarations (row and param structs) | 247 against 247 |

Five generated files do not move at all: `db.go`, `models.go`, `heartbeat.sql.go`, and the two
`*_separation_test.go` files.

---

## 4. The `goose` leg

### 4.1 `goose validate` — the dry run

```
go run github.com/pressly/goose/v3/cmd/goose@v3.27.3 -v -dir <dir> validate
```

`goose validate` reads the `-- +goose` markers and reports each migration's transaction flag and
its Up and Down statement counts. It opens no database.

**It cannot run on `db/migrations` directly.** `goose` also collects Go migrations, and the
directory holds `embed.go` and two `_test.go` files, so the command dies at
`failed to get version from file "db/migrations/embed.go": no filename separator '_' found`. The
gate ran it on a copy holding the 68 `.sql` files only.

The base and stripped tables are identical over all 68 migrations: same type, same transaction
flag, same Up count, same Down count, same order.

### 4.2 A real `goose up`, which is the stronger check

`goose validate` counts statements. It does not prove they still mean the same thing. So the gate
also migrated two fresh databases on the PostgreSQL image the compose stack pins
(`postgres:16-bookworm@sha256:60f4761b…`, server 16.15), one from the original migrations and one
from the stripped set, and compared the result:

```
goose -dir <dir> postgres "postgres://…/<db>?sslmode=disable" up
pg_dump --schema-only --no-owner --no-privileges -d <db>
```

Both runs migrated to version 24900. Both dumps are 2,447 lines. Dropping `pg_dump`'s
per-run `\restrict` nonce leaves 2,445 DDL lines with the same SHA-256,
`f7f740ef1c771d12aac294ab2037374c2ba7e07bdf9f1e325fbbaa46d5677f87`.

**PostgreSQL executed both migration sets and built the same schema, byte for byte.**

---

## 5. The corpus proves less than it looks, so an adversarial fixture was added

The tree's SQL exercises almost none of the constructs where a hand lexer can go wrong. Measured
over `db/queries` and `db/migrations` on 2026-09-03:

| Construct | Occurrences in the tree |
| --- | ---: |
| block comment `/* */` | 0 |
| nested block comment | 0 |
| dollar-quoted body `$$ … $$` | 0 |
| `E'…'` string with a backslash escape | 0 |
| `--` inside a quoted literal | 0 |
| `-- +goose StatementBegin` | 0 |

Every comment in the tree is a plain `--` line comment. A corpus pass alone therefore says little
about the lexer's hard cases. The gate built a fixture that covers each of them and put it through
the same two oracles.

**The `sqlc` fixture.** 13 queries against a 4-column schema: `--` inside a single-quoted string,
a doubled quote, a `$tag$` dollar quote holding `--`, a bare `$$` quote holding `/* */`, an
`E'…\\…'` string, a `"a--b"` quoted identifier, a block comment, a nested block comment, a glued
block comment, a trailing comment on a code line, a block comment between two operands, a comment
beside a `$1::bigint` param, and a comment beside a cast.

Every literal survives the strip intact, and the generated API is identical across 314 compared
lines. The four remaining text hunks are the inline comments themselves, removed from inside the
embedded query string.

**The `goose` fixture.** Two migrations covering `-- +goose StatementBegin`/`StatementEnd` around
a `plpgsql` function whose `$$` body holds a `--` comment, a trailing comment on a column
definition, `'a -- not a comment'` inserted as data, and `-- +goose NO TRANSACTION` with prose
directly beneath it.

All 9 markers survive. `goose validate` reports the same table, including the `✘` transaction flag
that `NO TRANSACTION` sets. Migrated live, both databases produce identical schema dumps, the
literal `a -- not a comment` round-trips into the row, and `SELECT bump(41)` returns 42 from the
stripped side — so the dollar-quoted function body reached PostgreSQL whole.

The comment inside the `$$` body is **not** deleted. The lexer reads a dollar quote as one token,
which is the correct and safe direction.

---

## 6. Findings

**No lexer miss was found, and `internal/commentlint/surface` is unchanged by this ticket.** Four
facts came out of the run.

### 6.1 A glued SQL comment is not deletable under ruling 15

`SELECT/* c */id` cannot be swept at all. §5.1 keeps a `GLUE` token in the skeleton wherever a
comment separates two tokens, so any delete removes that token and `verify` reports the file
changed — even when the delete leaves a space behind. The gate's skeleton comparison names it
exactly: `base line 29 holds GLUE, head line 29 holds IDENT "id"`.

This is the pinned intent of `TestSQLSeparatingCommentMovesTheSkeleton`, not a defect. It is a
conservative refusal: it blocks a legal delete rather than admitting an illegal one.

**The tree holds zero glued SQL comments**, so no D2 ticket can hit it as the tree stands. §6.5's
SQL delete rule should state the exclusion when it is written.

### 6.2 The §3.9-style byte comparison does not transfer to `internal/db`

Section 3.1 above. A D2 sweep PR will move `internal/db` whenever it deletes a comment above a
`-- name:` directive, and `sqlc` CI regenerates and diffs `internal/db`, so **a D2 sweep must ship
the regenerated `internal/db` with it.** That is a change in generated Go, not in SQL, and ruling
15 does not reach it.

### 6.3 `goose` rejects a block comment before the `Down` marker

Found while building the fixture. A `/* … */` sitting between the last Up statement and
`-- +goose Down` fails the parse with `unexpected unfinished SQL query … missing semicolon?`. No
migration in the tree does this, and migrations are out of sweep scope, but the trap is real for a
future migration author.

### 6.4 The ticket's figures had drifted

| Figure | Ticket | Measured 2026-09-03 |
| --- | ---: | ---: |
| Files in `db/queries` | 106 | 39 |
| `-- name:` directives | 250 | 251 |
| `-- +goose` markers | 134 | 136 |

The 106 is the whole tree's `.sql` count from SPEC §5.2, not the `db/queries` count. #1140 already
re-measured that to 107, of which 39 are in scope.

### 6.5 The skeleton side, checked in the one direction the gate can reach

The gate compared each base and stripped skeleton the way `verify` does: **39 of 39 clean.** It
then mutated one SELECT-list identifier per stripped file to prove the comparison is not blind:
**35 caught, 4 skipped** where the probe found no bare identifier after `SELECT `, and **none
missed**.

This is weaker than the comment-range result. `sqlc` and PostgreSQL prove the comment *ranges*
are right; nothing outside the lexer proves its *token stream* is fine-grained enough. The
mutation probe covers the smallest realistic code change and no more. That residue stays open.

---

## 7. Rebuilding the harness

About 200 lines of Go in two files, plus two shell comparisons. Both modes import
`internal/commentlint/surface` and nothing else from the repo.

**Strip mode**, `sqlgate <in-dir> <out-dir>`. For each `.sql` file: `surface.SQL{}.Lex(src)`, then
collect `res.Blocks` and `res.Trailing`, skip any block with `Directive` set, sort the rest by
`Start`, and write the source with each `[Start, End)` range removed. Where the bytes on both
sides of a removed range are non-space, write one space in its place. Report the file, block,
line, directive and glue counts.

**Skeleton mode**, `sqlgate -skeleton <base-dir> <head-dir>`. Lex both sides, compare
`res.Skeleton` element by element with `Token.Equal`, and report the first divergence with both
line numbers. Then rewrite the first identifier after `SELECT ` in the head file to `zz_mutant`
and require the comparison to flag it.

**The `sqlc` comparison.** Generate from both query trees, then for each generated file drop the
lines between `^const [A-Za-z_]+ = \`` and the closing backtick, drop `//` and `--` comment lines,
drop blank lines, trim trailing space, and diff the two trees.

**The `goose` comparison.** `goose -v -dir <dir> validate` on each tree and diff the tables. Then
`goose -dir <dir> postgres <dsn> up` into two fresh databases on the pinned PostgreSQL 16 image,
`pg_dump --schema-only --no-owner --no-privileges` from each, drop the `\restrict` and
`\unrestrict` nonce lines, and compare.

---

## 8. Tool versions

| Tool | Version |
| --- | --- |
| Go | 1.26.8 |
| `sqlc` | v1.31.1 (the version CI pins) |
| `goose` | v3.27.3 (the version `go.mod` pins) |
| PostgreSQL | 16.15, image `postgres:16-bookworm@sha256:60f4761b9035e0b8d5218f701a8c3382f641bf12b1604822574cf5be3baeb537` |
| Repo | `main` at `c6c8bf4` |
