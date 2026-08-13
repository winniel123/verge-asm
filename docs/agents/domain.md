# Domain Docs

How the engineering skills should consume this repo's domain documentation when exploring the codebase.

**This repo is single-context**: one `CONTEXT.md` and one `docs/adr/` at the root.

## Before exploring, read these

- **`CONTEXT.md`** at the repo root, or
- **`CONTEXT-MAP.md`** at the repo root if it exists — it points at one `CONTEXT.md` per context. Read each one relevant to the topic.
- **`docs/adr/`** — read ADRs that touch the area you're about to work in. In multi-context repos, also check `src/<context>/docs/adr/` for context-scoped decisions.

If any of these files don't exist, **proceed silently**. Don't flag their absence; don't suggest creating them upfront. The `/domain-modeling` skill (reached via `/grill-with-docs` and `/improve-codebase-architecture`) creates them lazily when terms or decisions actually get resolved.

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

When your output names a domain concept (in an issue title, a refactor proposal, a hypothesis, a test name), use the term as defined in `CONTEXT.md`. Don't drift to synonyms the glossary explicitly avoids.

If the concept you need isn't in the glossary yet, that's a signal — either you're inventing language the project doesn't use (reconsider) or there's a real gap (note it for `/domain-modeling`).

## Land glossary and ADR changes on `main` before the session ends

**`CONTEXT.md` is one file, and it must not fork.** Every session reads it, and a session reads
whichever copy its branch inherited — so an edit left on an unmerged branch is invisible to every
later session, including sessions that go on to edit the same paragraph.

If you write to `CONTEXT.md` or `docs/adr/`, merge that work to `main` as part of closing the
ticket. Don't leave it on a long-lived branch for review later.

This is not hypothetical. On 2026-08-13, [#27](https://github.com/winniel123/verge-asm/issues/27)
took registry data out of the `Ownership` derivation and correctly amended both ADR-0002 and the
glossary — on a branch that was never merged. Ten branches later, `main`'s `CONTEXT.md` was 9.6 KB
against the working branch's 24 KB, every session was reading a different glossary, and
[#39](https://github.com/winniel123/verge-asm/issues/39) was filed and worked against a premise
ADR-0002 had already withdrawn. Recovering it meant reconciling three stranded commits and
rewriting 40 branch-pinned links.

Two riders. **Link to `blob/main/…`, never to `blob/<your-branch>/…`** — a branch link rots the
moment the branch is deleted and reads as current until it does. And **when a decision changes what
may be *read*, grep the glossary for the clauses it invalidates**: #27 changed the inputs to the
probing gate, and `Vantage class` went on saying it re-verified against registry ranges because
nobody searched for the other place that sentence lived.

## Flag ADR conflicts

If your output contradicts an existing ADR, surface it explicitly rather than silently overriding:

> _Contradicts ADR-0007 (event-sourced orders) — but worth reopening because…_
