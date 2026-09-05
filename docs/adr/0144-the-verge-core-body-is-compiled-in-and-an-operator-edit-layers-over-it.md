# ADR-0144: The `verge-core` body is compiled in, and an operator edit layers over it rather than replacing it

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1267 ADR gaps: internal/vergecore, internal/vantage, db/migrations, the golden corpora](https://github.com/winniel123/verge-asm/issues/1267) §1
- **Spec:** [`docs/spec/v1-spec.md`](../spec/v1-spec.md) §3.5
- **Bounded by, and not an amendment to:** [ADR-0009](./0009-verge-core-is-a-union.md), which rules the union `frequency-set ∪ sensitive-list` and which half an operator may move. It does not rule where the body comes from
- **Withdrawal convention:** [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)

## Context

[#4](https://github.com/winniel123/verge-asm/issues/4) specified that `verge-core` ship *"as an
editable list file, not compiled in"*. The code does the opposite and has since the package landed.

`internal/vergecore/vergecore.go:47` carries `//go:embed verge-core.tsv`. Line 50 is
`var shippedList = mustParse(shipped)`, and it is the **only** production parse in the tree. `Default`
(line 52) hands out a clone of that one list. No configuration path, no file path and no database
column reaches `Parse`.

An operator edit does exist, and it **layers**. `internal/queue/hot.go:165` reads the
`verge_core_frequency_edit` rows and returns `vergecore.Default().WithFrequencyEdits(fe)`.
`cmd/web/settings.go:787-789` builds the settings view the same way, from `vergecore.Default()`. The
shipped body is the base at both sites, and the operator's rows are deltas over it.

Four documents still specify the withdrawn shape. `docs/spec/v1-spec.md:247` says `verge-core` *"is
shipped as an editable list file"*. [ADR-0009](./0009-verge-core-is-a-union.md):15 quotes #4's clause.
`docs/research/safe-active-probing.md:245` states it as a recommendation and `:1284` as a knob
requirement. `db/migrations/19400_hot_scan_verge_core.sql:16` repeats it in the comment above the very
table that implements the layering. Read alone and in the present tense, each would send a session to
build a replacement path.

The remaining limb is `Parse` itself. It is exported, and **no caller outside `internal/vergecore`
exists**. `internal/vergecore/vergecore_test.go` does not call it either — every case there goes
through `Default()`. The rule that kept the export was stated only in a code comment, which
[#1266](https://github.com/winniel123/verge-asm/issues/1266) deleted under the comment policy's gate A.

## Decision

> **The `verge-core` body is compiled into the binary. An operator changes the effective list by
> layering frequency-edit deltas over the shipped base, never by supplying a body of their own. No
> code path parses a non-shipped body, and none is added. `Parse` stays exported so a test may supply
> an alternative body, and for no other reason.**

### 1. The embed is ratified; the spec is what was stale

The code is correct and stays as it is. #4's clause is withdrawn at every site that states it, per
ADR-0058. This ADR changes **no Go code**.

### 2. A replaceable body would let an operator move the sensitive half

This is the reason the refusal is structural rather than a matter of taste.

`verge-core.tsv` carries the half on each row — `port<TAB>transport<TAB>half`. `Parse` reads that
third field and files the pair into `frequency` or `sensitive`. So a body an operator supplies is a
body in which the operator writes the `sensitive` column.

ADR-0009 and v1 spec §3.5 forbid exactly that. The sensitive half is authored by the release, because
moving it would move a version and `Break` the estate without a release and without a golden-corpus
row moving. An operator-supplied body defeats that rule at its root: it does not merely add or drop a
sensitive pair, it lets the operator **relabel** one as `frequency` and then remove it through the
edit path the product already offers. `IsSensitive` — which is what the settings UI's edit guard reads
(`vergecore.go:105`) — would answer from the operator's own file, so the guard would be asking the
attacker's document whether the attacker may act.

The layering shape has no such hole, and the property is arithmetic rather than checked. The union
keeps a pair a `remove` edit drops where that pair is also sensitive, so a frequency edit can never
subtract from the sensitive half. That is the invariant
`db/migrations/19400_hot_scan_verge_core.sql` already records and
`internal/vergecore/vergecore_test.go` already pins.

### 3. `Parse` stays exported, for test bodies alone

`Parse` is the parser `mustParse(shipped)` calls, and it is exported so a test may parse an
alternative body without reaching into the package.

**It has no caller today.** Not in production, not in `internal/vergecore/vergecore_test.go`, and not
in any package outside `internal/vergecore`. The export is a deliberately reserved seam, and this ADR
is the record that it is one — recorded here because the code comment that used to carry the rule was
deleted, and because an unexplained exported function with no caller is exactly the shape a later
session deletes or, worse, wires to a config file.

The bound the export carries: **any new caller of `Parse` outside a test is a change to this
Decision.** It is not a small change, because §2 is what it would give up.

### 4. Nothing gains an enforcement mechanism

No lint rule, no test and no build gate is added to keep `Parse` test-only. The rule is a one-line
Decision over an unexported-in-practice seam, and the check would guard a state that a code review
reads off the import graph. This ADR is the statement; the grep is the audit.

## Consequences

- **Four documents and one migration comment lose the withdrawn clause.** They are listed in §Context
  and marked in the same change, per ADR-0058.
- **`internal/vergecore` is unchanged.** `Parse` keeps its name, signature and export.
- **An operator who wants a different port set has one lever**, the frequency-edit rows, and it
  reaches the frequency half alone. That is the same lever v1 spec §3.5 already describes.
- **`CONTEXT.md` gains nothing.** *Where the body is compiled from* is a packaging fact, not a domain
  term, and the glossary's `Derivation` entry already says a declared parameter is authored by the
  project and ships in the release.
- **The refusal is reversible at a stated price.** A replaceable body becomes worth revisiting only
  where the file format stops carrying the `half` column, which is the whole of §2's objection.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **An operator-supplied replacement list file**, read at start-up or from a mount | The body carries the `half` column, so the operator would author the **sensitive** half. ADR-0009 and §3.5 forbid moving it: doing so moves a version and `Break`s the estate with no release and no golden-corpus row moved. `IsSensitive` — the settings edit guard — would then answer from the operator's own file, so the guard would consult the document it exists to constrain |
| **A replacement file restricted to the frequency half** | It is `WithFrequencyEdits` with a worse interface: a whole-body swap where a delta already works, with no record of what the operator changed and no per-port audit row. It also parses a non-shipped body, so it reopens the §2 surface for a capability the layering path already delivers |
| **Unexport `Parse`** | The test seam is the reason it exists, and this ADR names that reason where a reader will find it. Unexporting it also forecloses the one legitimate future caller — a test in another package — for no gain, since §4 declines the enforcement it would buy |
| **Delete `Parse`'s export and inline the parser into `mustParse`** | Same loss as above, plus it entangles the parser with the panic wrapper. The parse errors are the parser's contract and are worth reading on their own |
| **Add a lint rule or a test that fails on a non-test caller of `Parse`** | A mechanism that can be absent, guarding a state a reviewer reads off the import graph. ADR-0009 already prefers a definition over a test where the definition holds |
| **Amend [ADR-0009](./0009-verge-core-is-a-union.md) instead of ruling here** | ADR-0009's Decision is the **composition** of `verge-core` and which half is operator-editable. Where the body is loaded from is a different subject, and #4's clause is stale in its Context rather than in its Decision. That clause is withdrawn there under ADR-0058; the ruling lives here |
| **Correct the spec and write no ADR** | The embed is a decision with a security rationale that no document carries. Correcting §3.5 alone leaves the *why* nowhere, and the next session to read `Parse` finds an exported function with no caller and no reason |
