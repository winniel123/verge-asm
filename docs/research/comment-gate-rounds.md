# Comment policy validation gate — round ledger

The record of every §3.9 sampling round on the Go delete set. Ticket
[#1136](https://github.com/winniel123/verge-asm/issues/1136), map
[#1131](https://github.com/winniel123/verge-asm/issues/1131). SPEC
[`docs/spec/comment-policy.md`](../spec/comment-policy.md) §3.9.

The two sheets beside this file always hold the **current** round. A superseded round's 100 blocks
stay in git history at the commit named below. This ledger holds the verdicts, which is what stage B
rests on.

A class is accepted at 2 or fewer load-bearing blocks. A class runs at most three rounds. A class
that fails round 3 leaves the v1 delete set and stays in the flag set, because flagging is advisory.

## Round 1 — failed on both docstring classes

Sheets at commit `0fd3975`. Screen as it stood after #1134, unwidened.

| Population | Files read | Admitted | Drawn |
| --- | --- | --- | --- |
| Production Go | 255 | 1,978 | 100 |
| Test Go | 221 | 580 | 100 |

| Population | Class | Read | Load-bearing | Verdict |
| --- | --- | --- | --- | --- |
| Production | `commented-out-code` | 0 | n/a | n/a — no block admitted |
| Production | `docstring-exported-conventional` | 48 | 3 or more | **failed** |
| Production | `docstring-unexported` | 48 | 3 or more | **failed** |
| Production | `section-divider` | 3 | 2 or fewer | accepted |
| Production | `short-label` | 1 | 2 or fewer | accepted |
| Test | `commented-out-code` | 0 | n/a | n/a — no block admitted |
| Test | `docstring-exported-conventional` | 28 | 3 or more | **failed** |
| Test | `docstring-unexported` | 46 | 3 or more | **failed** |
| Test | `section-divider` | 9 | 2 or fewer | accepted |
| Test | `short-label` | 17 | 2 or fewer | accepted |

The read stopped at the failure threshold on a failed class, so its exact count above 2 is not
recorded. Three load-bearing blocks already decide the verdict, and reading the rest of that class
buys nothing.

`commented-out-code` admits zero blocks anywhere in the in-scope Go tree. The SPEC's 22-line figure
for that class was measured over the whole corpus, not over screened own-line Go.

### The widening round 1 produced

Every change below is an addition. §3.2 lets a revision widen the screen and never narrow it, and
`TestTheWideningNarrowedNothing` pins that.

| Signal | Added | Why |
| --- | --- | --- |
| WHY marker | bare `so` | The list carried `so that` only, and the tree's reason clauses use bare `so`. |
| WHY marker | `rather than`, `instead of` | Both name a rejected alternative. |
| WHY marker | `never` | Names a constraint the code cannot show. |
| WHY marker | `there is no`, `there are no` | Names an absent thing, which no declaration can state. |
| Citation | `§ n` | Reaches `v1 spec §4.3` and `SPEC §n`, neither of which is `ADR-nnnn` or `#nnn`. |
| Citation | `CONTEXT.md`, `DF-Fn` | Two reference forms this tree uses. |
| Tool marker | `#nosec` | gosec reads it, and the block around it says why the waiver is safe. |

The bare word `spec` was tried as a citation and dropped. It rides in a URL path
(`https://go.dev/ref/spec`), where the bare-URL signal already withholds the block, and it would
have relabelled that block's reason.

### Effect on the delete set

| Population | Admitted, round 1 | Admitted, round 2 | Change |
| --- | --- | --- | --- |
| Production Go | 1,978 | 1,022 | −956, −48% |
| Test Go | 580 | 404 | −176, −30% |

Per class, production Go: `docstring-exported-conventional` 827 → 462, `docstring-unexported`
1,032 → 444, `section-divider` 88 → 88, `short-label` 31 → 28. The widening lands almost entirely
on the two classes that failed.

## Round 2 — awaiting adjudication

| Population | Files read | Admitted | Drawn |
| --- | --- | --- | --- |
| Production Go | 255 | 1,022 | 100 |
| Test Go | 221 | 404 | 100 |

| Population | Class | Read | Load-bearing | Verdict |
| --- | --- | --- | --- | --- |
| Production | `commented-out-code` | 0 | n/a | n/a — no block admitted |
| Production | `docstring-exported-conventional` | 47 | | |
| Production | `docstring-unexported` | 44 | | |
| Production | `section-divider` | 8 | | |
| Production | `short-label` | 1 | | |
| Test | `commented-out-code` | 0 | n/a | n/a — no block admitted |
| Test | `docstring-exported-conventional` | 18 | | |
| Test | `docstring-unexported` | 40 | | |
| Test | `section-divider` | 13 | | |
| Test | `short-label` | 29 | | |

`section-divider` and `short-label` passed round 1. They are re-drawn here because the sheet samples
the whole admitted set, not one class. A second pass on an accepted class cannot un-accept it.
