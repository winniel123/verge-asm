# ADR governance

- **Status:** Accepted — spec content for [#1638](https://github.com/winniel123/verge-asm/issues/1638)
- **Wayfinder map:** [Map: ADR governance (#1638)](https://github.com/winniel123/verge-asm/issues/1638)
- **Ticket:** [#1646 What ends the moratorium, what CLAUDE.md says afterwards, and the landing order](https://github.com/winniel123/verge-asm/issues/1646)

This SPEC rules how an ADR is written, numbered, related, retired, proved, and checked. Each heading
names the ticket that decided it. That ticket's resolution holds the detail and the worked examples.

## 1. Scope

The SPEC governs the files under `docs/adr/` and the contract between a code comment and the ADR it
cites. It does not govern `CONTEXT.md`, `docs/spec/*`, or `docs/research/*`. It rewrites, shortens,
renumbers, or retitles no existing ADR.

## 2. Terms

| Term | Meaning |
| --- | --- |
| Acting ADR, target | The ADR whose front matter declares a relation, and the ADR it names. |
| Clause | A numbered heading in the target, per `docs/spec/comment-policy.md` §4.7, as a dotted string: `"3.1"`. |
| Decision block | The first `## Decision` heading through to the next heading. |
| Marker | One tool-written blockquote line under an affected heading, ending in an `<!-- adr-marker -->` sentinel. |
| Index | The committed `index.json` under `docs/adr/`, regenerated from front matter. |
| Decision proposal block | Five PR-body fields: Thesis (16 words or fewer), Site (`file:line`), Alternative, Reversal, Proof. One unticked "Ratified" checkbox ends it. |
| Touch | A PR diff that includes an ADR file, a line that cites it, or the code beside a comment that cites it. |

## 3. The file (#1641)

The file is `docs/adr/<number>-<slug>.md`, zero-padded to four digits and never more. It opens with
YAML front matter, one blank line, then the H1. The parser loads the YAML core schema, so an unquoted
date stays a string.

| Field | Required | Rule |
| --- | --- | --- |
| `number` | yes | The filename prefix. Above 227 it equals `ticket`. |
| `title` | yes | The H1, less an optional `ADR-NNNN: ` prefix. Above 227 the prefix is present and the title is 16 words or fewer. |
| `slug` | yes | The filename stem after `<number>-`. |
| `date` | yes | `YYYY-MM-DD`. |
| `status` | yes | `accepted` or `withdrawn`, hand-set. |
| `source` | yes | `grilling`, `fix`, or `sweep`. |
| `proof` | yes | `{test: "<path>::<name>"}`, `{ticket: <n>}`, or `{none: "<reason>"}`. |
| `relations` | no | List of `{kind, adr, clause}`. |
| `ticket` | above 227 | Integer or list. |
| `map`, `pr` | no | Integers. |
| `withdrawal` | when withdrawn | `{date, reason, moved-to}`. `moved-to` is a path, an ADR number, or `none`. |

A new ADR's number is the GitHub issue that ticketed it, with no fallback. One issue yields at most
one ADR. ADR-0001 to ADR-0227 keep their numbers. Above 227 the Decision block is the first `##` in
the body, 150 words or fewer, and numbered sections follow it.

An ADR never amends itself in place. It changes only through a later ADR with an `amends`, `retires`,
or `supersedes` relation. The citation `ADR-nnnn #nnn` names a legacy in-file amendment and is valid
at 227 and below only.

## 4. Relations and status (#1641)

Six kinds, closed. An edge lives once, on the acting ADR.

| Kind | Scope | Derives | `clause` |
| --- | --- | --- | --- |
| `amends`, `retires` | clause | `amended` | required when the target numbers a heading |
| `supersedes` | whole ADR | `superseded` | forbidden |
| `rests-on`, `bounds` | either | nothing | optional |
| `sibling` | whole ADR | nothing | forbidden |

A `clause` never names an unnumbered heading. On a target with no numbered heading, `amends` and
`retires` omit `clause` and act on the whole file. `amended` and `superseded` derive from incoming
relations. Precedence: `withdrawn`, `superseded`, `amended`, `accepted`. Every edge counts for the
life of the file. No file is ever deleted.

## 5. Markers (#1644)

The tool writes one blockquote line per `amends`, `retires`, or `supersedes` relation, and one per
`withdrawn` status. A hand never writes one. The line is the first line under the affected heading.
It names the acting ADR by thesis title and date, links `./<number>-<slug>.md`, and ends in
`<!-- adr-marker <kind> <n> -->`. A clause-scoped marker sits under its numbered heading. The rest
sit under the H1, ordered `withdrawn`, `supersedes`, then ascending acting number. A blockquote
without the sentinel is prose, so a legacy hand-written line stays. A spec or research target keeps
the sentence rule of ADR-0058.

## 6. Proof (#1641, #1642)

A `test` proof is `<path>::<name>`: the `Test` function of a `.go` file, or the `test("…")` title of a
`.mjs` file. CI checks the path exists and names it. A `ticket` proof is not checked offline. A `none`
proof carries a one-line reason the reviewer accepts.

A `source: sweep` ADR is proved or withdrawn in the PR that next touches it. Proved: the three-part
test holds, no live document states the rule, and the code exhibits it. The session replaces the
`none` proof. Withdrawn: one of the three fails, and `reason` names it. `moved-to` names where the
rule lives. It is `none` when the rule was false or is recoverable from the code. The PR body carries
one line per ADR touched. The review gate does not run.

## 7. Authoring (#1642)

An ADR records one decision that passes three tests. It is hard to reverse. A reader without context
would ask why. The session chose it over a named alternative.

A deleted or compressed comment that passes gates A, B, and C of `docs/spec/comment-policy.md` §8.2
earns one decision proposal block in the PR body. The comment survives in code with no citation.
Only a human opens the issue: one per ticked block, labelled `ready-for-agent`, never a sub-issue of
a map. Its number becomes the ADR's number. An unticked block at merge is a refusal. A PR with no gap
states `ADR gaps: none`. The `adr-gap` label stays on its 43 closed issues, retired. Under
orchestration a subagent returns the block, and only the orchestrator writes an ADR, one PR at a time.

## 8. Review (#1643)

A fresh-context subagent reviews every PR that adds an ADR file. It receives the PR number and the
`adr-review` skill, and nothing else.

| Row | Pass condition |
| --- | --- |
| 1. Three-part test | The Decision block names the reversal cost, why a reader would ask, and the rejected alternative. |
| 2. One claim | The title and the Decision block state the same single rule. |
| 3. Proof is real | `test`: the reviewer quotes the assertion. `ticket`: the issue is open and names the rule. `none`: the reviewer accepts the reason. |
| 4. No prior ruling | The reviewer names the three nearest ADRs from the index and states why each differs. |

The verdict is one PR comment per reviewed SHA. It opens with
`<!-- adr-review sha=<40hex> verdict=pass|fail -->`, then one table row per check with a `pass` or
`fail` cell and one evidence line. Any `fail` row fails the verdict. After three `fail` verdicts the
orchestrator stops and says so. The human reads the Decision block in the PR body, then the table,
then squash-merges. The merge records the reading.

## 9. CI checks (#1640, #1643, #1644, #1645)

Every check is a Node script in `docs-site/scripts/` that reuses the parser of
`check-adr-sections.mjs`. The `adr-sections` job grows two steps. `adr-review` is a new required
check. It runs on every PR, passes when the PR adds no ADR file, and listens to `edited`.

| Check | Fails when |
| --- | --- |
| Citation | `ADR-\d{4,}` names no file. A `§` names no numbered heading, or names a heading by word. `ADR-nnnn #nnn` appears above 227. |
| Schema | A required field is missing. `number` differs from `ticket` above 227. A `test` proof does not resolve. |
| Relations | A `clause` does not resolve, or is missing where the target numbers a heading. An edge is written on both sides. |
| Decision block | Above 227 it is not the first `##`, or exceeds 150 words. |
| Index, markers | The committed index or any marker differs from the regeneration. |
| Review | The PR adds an ADR file and no single `pass` marker matches the head SHA. Two markers share a SHA. Two ADR files. The PR body's `## Decision` differs from the file. |

## 10. Legacy backfill (#1639, #1641, #1645)

One script converts all 226 files in one PR. It deletes the header lines it consumed: `Status`,
`Date`, `Ticket`, `Map`, every PR line, and every relation line that mapped with no qualifier. It
keeps the 45 unmappable lines and every qualified line as prose. It writes an inverse header as a
forward edge on the target. ADR-0145 supersedes ADR-0109 and ADR-0116. Nothing is `withdrawn`.

Every legacy ADR gets `proof: {none: "predates the governance SPEC"}`. `source` is `sweep` when a
sweep header kind exists, `grilling` when a `Map` line exists, else `fix`. The human reads the printed
source table once. The 22 plain H1s keep no prefix. The four `§Name` citations, three inside ADR
bodies, become the bare form. That repair and the header deletion change no meaning, so §1 holds.
The same PR deletes the script and names the commit that holds it.

## 11. `CLAUDE.md` text (#1646)

Until the `main protection` ruleset requires `adr-review`, `CLAUDE.md` carries an interim
`### ADR moratorium`: add no ADR file, and a decision becomes a decision proposal block. The ruleset
edit ends the moratorium. The next PR replaces the section with this text, verbatim.

````markdown
### Writing an ADR

An ADR records one decision that passes three tests. It is hard to reverse. A reader without context would ask why. The session chose it over a named alternative. When one fails, write no ADR. Keep the reason in code, and put the rest in the PR body.

Only a human opens the issue that becomes an ADR, and its number is the ADR's number. A deleted comment that passes the three tests earns a decision proposal block in the PR body, never an issue. Under orchestration only the orchestrator writes an ADR, one PR at a time.

An ADR PR adds one file: YAML front matter, a Decision block under 150 words, a proof. A fresh-context subagent runs the `adr-review` skill, and the required `adr-review` check holds the merge. A later ADR changes an earlier one through a relation, and the tool writes the marker. Never hand-edit a marker. See `docs/spec/adr-governance.md`.
````

Seven lines. Adding a line requires removing one. The `## Comments` citation form is unchanged.

## 12. Landing order (#1646)

Eight tickets, one per step. Steps 2 and 3 may run in parallel, and so may 7 and 8. No check step
joins CI before the files it checks exist.

1. Checker refactor: exported functions, heading line and level, `\d{4,}`, the `§Name` refusal with
   the four-site repair, the legacy `#nnn` cap.
2. Front matter parser, schema validation, index generator with `--check`, and tests.
3. Marker tool with `--write` and `--check`, and tests.
4. Backfill (§10), the two check steps in `adr-sections`, and the human read.
5. Review gate: the check script, the `adr-review` job, the `adr-review` skill.
6. The human requires `adr-review` in the ruleset. A PR then lands the §11 text. HITL.
7. The comment-policy amendment at the six places the #1642 resolution drafts, and the `adr-gap`
   label description.
8. ADR-1644, the ADR-0058 amendment, with the Decision block the #1644 resolution drafts.

The reference shape is `docs-site/scripts/prototype-adr-index.mjs` on branch
`prototype/adr-governance`. Never merge that branch.

## 13. Not yet specified

- How `docs-site` renders the index and the Decision block. Nothing in §12 needs it.

## 14. Handoff

This SPEC is the destination of map #1638. `/to-tickets` on this document cuts the eight tickets of
§12.
