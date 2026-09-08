---
name: adr-review
description: Fresh-context review of a pull request that adds one ADR file. Takes a PR number and nothing else. Runs the four reviewer rows of SPEC docs/spec/adr-governance.md §8, posts one verdict comment per head SHA in the fixed marker format, and re-runs the failed `adr-review` job. Use when an author session, or its orchestrator, spawns a reviewer for an ADR PR, or when a human asks for an ADR review by PR number.
user-invocable: true
---

# ADR review

You are the fresh-context reviewer of SPEC `docs/spec/adr-governance.md` §8. Your input is one PR number. You receive no transcript, no proposal block, and no map body. Do not ask for them. Everything you judge comes from the PR itself, `docs/adr/index.json`, and the proof.

The required check `adr-review` (`docs-site/scripts/check-adr-review.mjs`) holds the PR until one comment on it opens with a `pass` marker for the head SHA. This skill writes that comment. The check is mechanical and reads only the marker line. The judgment is yours.

`gh` renders no default view on this repo. Every `gh pr view` and `gh issue view` needs `--json`. See `docs/agents/issue-tracker.md`.

## 1. Fetch the inputs

1. Read the PR:

   ```sh
   gh pr view <n> --json number,title,body,headRefOid,files
   ```

   `headRefOid` is the head SHA. The marker names it in full, all 40 hex characters.

2. Find the added ADR file. `gh pr view` lists paths without a status, so ask the REST API:

   ```sh
   gh api repos/winniel123/verge-asm/pulls/<n>/files --paginate --jq '.[] | select(.status == "added") | .filename' | grep -E '^docs/adr/[0-9]{4,}-.+\.md$'
   ```

   Exactly one line must come back. If the PR adds no ADR file, or more than one, stop. The check already fails those on its own, and no verdict changes that. Say so and end.

3. Read the four inputs at the head SHA:

   ```sh
   gh pr diff <n>
   gh api "repos/winniel123/verge-asm/contents/<adr path>?ref=<sha>" -H 'Accept: application/vnd.github.raw+json'
   gh api "repos/winniel123/verge-asm/contents/docs/adr/index.json?ref=<sha>" -H 'Accept: application/vnd.github.raw+json'
   ```

   The fourth input is the proof. Read `proof:` in the file's front matter.
   - `{test: "<path>::<name>"}`: fetch `<path>` at the head SHA the same way. Find the `Test` function, or for a `.mjs` path the `test("<name>")` title.
   - `{ticket: <n>}`: `gh issue view <n> --json state,title,body`.
   - `{none: "<reason>"}`: the reason is the input.

4. Count the markers already on the PR:

   ```sh
   gh api repos/winniel123/verge-asm/issues/<n>/comments --paginate --jq '.[].body' | grep '^<!-- adr-review sha='
   ```

   If a marker already names the head SHA, stop and say so. A second marker for one SHA fails the check whatever its verdict, and nobody deletes a marker. A changed verdict needs changed content, and changed content pushes a new SHA.

   If three `fail` markers already sit on the PR, stop. Read `## 4. The cap`.

## 2. Judge the four rows

Run the rows of SPEC `docs/spec/adr-governance.md` §8 in the order the table gives them. Quote the SPEC by row number. Do not restate the pass condition in your own words.

- **Row 1** reads the Decision block. The block is the text under the first `## Decision` heading, up to the next heading. Find each of the three parts and name the sentence that carries it. A missing part fails the row.
- **Row 2** reads the H1 and the Decision block together. Find the one rule both state. A second rule in either fails the row.
- **Row 3** reads the proof. For `test`, quote the assertion line that exercises the rule. For `ticket`, confirm the issue is open and quote the sentence that names the rule. For `none`, accept the reason or refuse it. Your evidence cell holds the quote or the acceptance.
- **Row 4** reads `index.json`. Name the three nearest ADRs by number and title, and for each say in one clause why it does not already rule this question. Fewer than three named fails the row.

One evidence line per row. On a `fail` row the evidence names the fix. Any `fail` row makes the verdict `fail`.

## 3. Post the verdict

Write the comment to a file and post it:

```sh
gh pr comment <n> --body-file <file>
```

The format is fixed. The marker is the first line. The table has four rows in SPEC order. The last line names the context. Prose may follow the table. Nothing sits inside it but a verdict and one evidence line.

```markdown
<!-- adr-review sha=<40 hex> verdict=pass -->

| Row | Verdict | Evidence |
| --- | --- | --- |
| 1. Three-part test (§8) | pass | <one line> |
| 2. One claim (§8) | pass | <one line> |
| 3. Proof is real (§8) | pass | <one line> |
| 4. No prior ruling (§8) | pass | <one line> |

Fresh-context subagent. Inputs: PR diff, `index.json`, the proof test.
```

Write `verdict=fail` in the marker when any row is `fail`. Post one comment per SHA, never two.

## 4. Re-run the check

A comment does not re-trigger a `pull_request` workflow. Find the `doclint` run for the head SHA and re-run its failed jobs:

```sh
gh run list --workflow doclint.yml --commit <sha> --json databaseId,status,conclusion
gh run rerun <run id> --failed
```

If the run is still in progress, wait for it with `gh run watch <run id>`, then re-run. The re-run reads your comment and goes green on a `pass` marker. Report the run id in your final message. A human re-runs by hand only when this step did not.

## 5. The cap

Three reviews per PR. Count the `fail` markers on the PR before you start. After the third `fail` verdict, the orchestrator that spawned you stops, leaves the PR open, and posts one comment that states the cap. It does not spawn a fourth review. A human decides what happens to the PR.

## 6. Do not

- Do not edit the ADR, the PR body, or any file. You review. The author fixes.
- Do not delete or edit a marker comment, yours or another.
- Do not approve the PR through a GitHub review. Every PR here comes from one account, and GitHub refuses a self-approval. The marker comment is the verdict.
- Do not read the author's session, the map, or the proposal block, even when handed to you.
