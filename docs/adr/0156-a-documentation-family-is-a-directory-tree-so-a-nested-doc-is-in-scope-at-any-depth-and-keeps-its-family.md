# ADR-0156: a documentation family is a directory tree, so a nested doc is in scope at any depth and keeps its family

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1398 ADR gaps: docs-site web assets (source-resolution, doclint scope)](https://github.com/winniel123/verge-asm/issues/1398), gap 2
- **PR that deleted the comment:** [#1397](https://github.com/winniel123/verge-asm/pull/1397)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Bounds:** [`doc-lint-tool.md`](../spec/doc-lint-tool.md) §1.3 and [`documentation-style-standard.md`](../spec/documentation-style-standard.md) §1.1, at each table's own site, per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)

## Context

[`docs-site/scripts/doclint/scope.mjs`](../../docs-site/scripts/doclint/scope.mjs) carried these two
blocks, until #1397 deleted them:

```
 * In-scope file discovery (SPEC §1.3).
```

```
/**
 * Every `.md` file under a directory, at any depth. The recursion closes a silent
 * scope hole: a doc nested in a subdirectory of a family (for example
 * `docs/research/2026/report.md`) is still in the family, so the tool must lint it.
```

The cited section does not state the rule. [`doc-lint-tool.md`](../spec/doc-lint-tool.md) §1.3 is a
flat table of five family paths and four root files. It settles no depth.
`grep -i 'depth\|recurs\|nested\|subdirector'` over that file returns nothing.

§1.3 also forwards: *"The tool lints the same file set the style standard §1.1 governs."*
[`documentation-style-standard.md`](../spec/documentation-style-standard.md) §1.1 is the same flat
table with no depth rule, so the forward resolves to a second silence. #790, #817 and #822 do not
state it either. That is #1398's gap 2.

The deleted text frames the recursion as a decision. It says the recursion *"closes a silent scope
hole"*. Nothing durable records that decision.

### The rule binds three sites, and they must agree

| Site | What it does |
| --- | --- |
| `scope.mjs:20` | `markdownFilesUnder` recurses into a subdirectory |
| `scope.mjs:46` | `isInScope` accepts a path where `rel === dir` or `rel` starts with `dir/` |
| `.github/workflows/doclint.yml:26` | the `pull_request` paths filter is `docs/adr/**` and four siblings |

`inScopeFiles` and `isInScope` are both exported. `docs-site/scripts/doclint.mjs:17` and `:19`
consume them, and `docs-site/scripts/doclint/candidates/measure.mjs` consumes `inScopeFiles`.

The two tool functions serve two different modes. `inScopeFiles` builds the whole-tree set for the
local command ([`doc-lint-tool.md`](../spec/doc-lint-tool.md) §5.1). `isInScope` filters the changed
files CI hands the tool through `--in-scope-only` (§5.2). **The two modes must select the same set.**

### CI already reads the families to any depth

`doclint.yml`'s paths filter uses `**`, so a change to `docs/research/2026/report.md` triggers the
job. If `isInScope` read one level, the job would run, filter that path out, and print *"no in-scope
docs changed"*. The workflow would report a clean pass on a doc it never linted. That is worse than
either consistent reading, because the green job is evidence a reviewer trusts.

### The case is unexercised today

No family directory under `docs/` has a subdirectory. `find docs/adr docs/spec docs/agents
docs/guides docs/research -mindepth 1 -type d` returns nothing on 2026-09-05. So the rule binds a
case nobody has created, and nothing written records which way it goes.

The only thing that pins it is a test, `docs-site/scripts/doclint.test.mjs:329`, *"isInScope accepts
a doc nested in a family subdirectory"*. A test is not a durable source, and that file belongs to
sibling ticket #1234.

## Decision

> **A documentation family in [`documentation-style-standard.md`](../spec/documentation-style-standard.md)
> §1.1 and [`doc-lint-tool.md`](../spec/doc-lint-tool.md) §1.3 is a directory tree, not a one-level
> listing. A `.md` file under a family directory is in that family at any depth, and the doc-lint
> tool lints it. Read `docs/adr/*` and its four siblings as `docs/adr/**`. Depth never changes the
> family a path selects, so `docs/research/2026/report.md` is a research doc.**

Four limbs.

### 1. The row names a family, not a directory shape

Each table's second column is headed **Family**. The row `docs/research/*` names the research
family, and the path is how a reader finds it. A depth cut-off would make family membership a
property of where an author filed the document rather than of what the document is.

The style standard's §3 table then assigns an STE mode per family, and §8 assigns an execution tier
per family. Both read the family, not the path shape. A nested research note that fell out of the
research family would carry no mode and no tier, which neither table can express.

### 2. Both tool functions read to any depth, and neither is the authority

`markdownFilesUnder` recurses. `isInScope` matches a prefix. They agree today, and this ADR is what
they agree with. A future change to one is a change to both.

The failure this closes is not a wrong answer. It is **two right answers in one tool**: the local
whole-tree command lints a nested doc, and the CI changed-file path drops it. A writer would then see
a violation locally that CI never reports.

### 3. The CI paths filter is already the any-depth reading

`doclint.yml` filters on `docs/adr/**`. That was written against §1.3's five family paths and four
root files, and it reads to any depth. (`doclint.yml:20`'s own comment says *"nine families"*, which
counts the four root files as families. The table has five.) This ADR makes the tool's reading match the trigger's, rather than leaving a nested doc
in the one state that produces a green job and no lint.

### 4. What this ADR does not reach

- **[`doc-lint-tool.md`](../spec/doc-lint-tool.md) §1.2's out-of-scope paths.**
  `docs/correspondence/` and `docs/wayfinder/` are already written as directories. They exclude to
  any depth for the same reason a family includes to any depth, and this ADR states no new rule for
  them.
- **The four root files.** `CONTEXT.md`, `CLAUDE.md`, `README.md` and `SECURITY.md` are exact names
  at the repository root. Depth has no meaning for them, and `isInScope` matches them by equality.
- **Non-Markdown files.** A family tree admits a `.md` file. `docs/guides/embed.go` stays out under
  §1.2, and the extension test in both functions is unchanged.
- **Which rules run.** [`doc-lint-tool.md`](../spec/doc-lint-tool.md) §1.4 already rules that the
  path selects the family and the family selects the mode. This ADR settles the depth half of that
  path read. It changes no rule set.
- **Whether a nested doc should exist.** The tree has none. This ADR rules what happens if one
  appears. It neither invites nor forbids one.

## Consequences

- **[`doc-lint-tool.md`](../spec/doc-lint-tool.md) §1.3 gains one sentence** under its table, and
  **[`documentation-style-standard.md`](../spec/documentation-style-standard.md) §1.1 gains one
  sentence** under its. Each table row reads `*` as a tree. ADR-0058 requires the edit at each site,
  because a reader of either table alone would take the flat reading.
- **`scope.mjs:19` gains this ADR's citation and moves.** The survivor sat in declaration position,
  which `CLAUDE.md` keeps empty. It now sits beside the recursive call it explains.
- **No lint result changes.** No family directory has a subdirectory today, so `inScopeFiles`
  returns the same list before and after.
- **`docs-site/scripts/doclint.test.mjs:329` now has a durable source.** The assertion is unchanged
  and this ADR does not touch that file. It belongs to sibling ticket #1234.
- **[`CONTEXT.md`](../../CONTEXT.md) gains nothing.** Family and in-scope are documentation-process
  terms. The glossary carries product domain terms.
- **A sixth family is added by adding a row.** The depth reading comes with the row and needs no
  further decision.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Read `docs/adr/*` as one level, and lint no nested doc** | The silent scope hole the deleted comment named. A doc under `docs/research/2026/` would be an in-scope family's document that no rule ever reads, and the CI paths filter would still trigger on it. It also makes family membership depend on the directory an author chose |
| **Leave the depth unsettled and let each site decide** | The state this ADR found. `markdownFilesUnder` recurses and `isInScope` matches a prefix, and both were uncited, so a later edit to either had nothing to check itself against. Two modes of one tool would then disagree about one file |
| **State the rule in the tool and not in the SPECs** | A reader of §1.3's flat table gets the wrong answer, and §1.3 is the document a ticket cites. ADR-0058 rules that a bounded clause is marked at its own site, and the table is the site |
| **State it only in [`doc-lint-tool.md`](../spec/doc-lint-tool.md) §1.3** | §1.3 forwards to the style standard §1.1 for the file set. A depth rule in the forwarding document and not in the forwarded one leaves the two tables disagreeing about their shared set |
| **Tighten the `doclint.yml` paths filter to one level to match a flat reading** | Fixes the disagreement in the direction that loses documents. It also puts the scope decision in a workflow file, where the tool cannot read it, and the local command would still lint a different set |
| **Fail the tool on a nested doc, so an author must flatten the tree** | Turns a scope question into a lint error about file layout, which no rule in §2's set is about. It also refuses a structure the family tables never forbade |
| **Let the test carry the rule** | `doclint.test.mjs:329` pins the behaviour and states no ground. A test records that someone chose this, not why, and [`comment-policy.md`](../spec/comment-policy.md) §8.2 gate C keeps a test out of the record for the same reason |
