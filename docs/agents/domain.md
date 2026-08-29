# Domain Docs

How the engineering skills should consume this repo's domain documentation when they explore the codebase.

**This repo is single-context**: one `CONTEXT.md` and one `docs/adr/` at the root.

## Before exploring, read these

- **`CONTEXT.md`** at the repo root, or
- **`CONTEXT-MAP.md`** at the repo root if it exists — it points at one `CONTEXT.md` per context. Read each one relevant to the topic.
- **`docs/adr/`** — read the ADRs that touch the area you are about to work in. In a multi-context repo, also check `src/<context>/docs/adr/` for context-scoped decisions.

If any of these files do not exist, **proceed silently**. Do not flag their absence. Do not suggest creating them upfront. The `/domain-modeling` skill creates them lazily when a session actually resolves terms or decisions. You reach that skill through `/grill-with-docs` and `/improve-codebase-architecture`.

## File structure

Single-context repo (most repos):

```
/
├── CONTEXT.md
├── docs/adr/
│   ├── 0001-event-sourced-orders.md
│   └── 0002-postgres-for-write-model.md
└── src/
```

Multi-context repo (presence of `CONTEXT-MAP.md` at the root):

```
/
├── CONTEXT-MAP.md
├── docs/adr/                          ← system-wide decisions
└── src/
    ├── ordering/
    │   ├── CONTEXT.md
    │   └── docs/adr/                  ← context-specific decisions
    └── billing/
        ├── CONTEXT.md
        └── docs/adr/
```

## Use the glossary's vocabulary

When your output names a domain concept (in an issue title, a refactor proposal, a hypothesis, a test name), use the term that `CONTEXT.md` defines. Do not drift to synonyms the glossary explicitly avoids.

If the concept you need is not in the glossary yet, that is a signal. Either you are inventing language the project does not use (reconsider), or there is a real gap (note it for `/domain-modeling`).

## Land glossary and ADR changes on `main` before the session ends

**`CONTEXT.md` is one file, and it must not fork.** Every session reads it. A session reads whichever
copy its branch inherited. So an edit left on an unmerged branch is invisible to every later session,
including later sessions that edit the same paragraph.

If you write to `CONTEXT.md` or `docs/adr/`, merge that work to `main` as part of closing the ticket.
Do not leave it on a long-lived branch for review later.

This is not hypothetical. On 2026-08-13, [#27](https://github.com/winniel123/verge-asm/issues/27)
removed registry data from the `Ownership` derivation. It correctly amended both ADR-0002 and the
glossary, on a branch that was never merged. Ten branches later, `main`'s `CONTEXT.md` was 9.6 KB
against the working branch's 24 KB. Every session read a different glossary. Someone filed
[#39](https://github.com/winniel123/verge-asm/issues/39) and worked it against a premise ADR-0002 had
already withdrawn. Recovering it meant reconciling three stranded commits and rewriting 40
branch-pinned links.

Two riders. First: **link to `blob/main/…`, never to `blob/<your-branch>/…`.** A branch link rots the
moment someone deletes the branch. It reads as current until then. Second: **when a decision changes
what may be *read*, grep the glossary for the clauses it invalidates.** #27 changed the inputs to the
probing gate. `Vantage class` kept saying it re-verified against registry ranges, because nobody
searched for the other place that sentence lived.

> **The second rider is the glossary-shaped case of a general rule. The general rule is now written
> down.** [ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)
> — *a superseded mechanism is withdrawn at the site that **specifies** it, not only at the site that
> **supersedes** it* — reaches every document, not only `CONTEXT.md`. Its two load-bearing instances
> run **note-to-ADR** rather than ADR-to-glossary. Its test: *if the superseded sentence, read alone
> and in the present tense, would cause a competent session to build the thing, it is not withdrawn.*
> The `Vantage class` case above is
> [#102](https://github.com/winniel123/verge-asm/issues/102)'s instance 6. It is the **earliest** one
> measured. It cost a whole ticket: someone filed
> [#39](https://github.com/winniel123/verge-asm/issues/39) and worked it against a premise ADR-0002
> had already withdrawn. Grep the glossary. Then grep everything else the decision names.
>
> **And then grep the document you are writing in.** Per
> [#106](https://github.com/winniel123/verge-asm/issues/106), ADR-0058 is on the **sentence**. So a
> document supersedes itself. An amendment appended to a file does not discharge the clause it
> supersedes hundreds of lines above in that same file. The rule's own forcing measurement is of that
> shape. [#91](https://github.com/winniel123/verge-asm/issues/91) re-asserted a dissolved invariant
> **inside ADR-0009's own body**, two screens below the Decision row that dissolved it.

## Flag ADR conflicts

If your output contradicts an existing ADR, surface the conflict explicitly. Do not override it silently.

> _Contradicts ADR-0007 (event-sourced orders) — but worth reopening because…_
