# Design System

Anything with a visual surface — production UI, throwaway prototype, mock, diagram, slide, screenshot — uses the **Verge ASM design system**. There is no second visual language in this project, and no "just for now" exception for prototypes: a prototype in the wrong visual language answers the wrong question.

## Before writing any markup

Invoke the **`verge-asm-design`** skill (`.claude/skills/verge-asm-design/SKILL.md`). It carries the key facts inline and points at the system itself.

The system lives at **`design-system/`** in the repo root:

- `styles.css` — the global entry. Link it, then use its custom properties.
- `tokens/` — colours, typography, spacing, effects, fonts, base.
- `components/` — `forms/` · `display/` · `feedback/` · `navigation/`. Every `.jsx` has a `.d.ts` contract and a `.prompt.md` usage note; read the `.prompt.md` before using a component.
- `guidelines/` — foundation specimen cards.
- `ui_kits/` — reference screens for the console, marketing site, and docs site.
- `readme.md` — the full specification. Read it for anything non-trivial.

**Never hardcode a value a token already names.** If you find yourself typing `#16160f`, you want `var(--ink)`.

## What is canonical and what is not

The system was authored from a product brief that predates this repo's domain model, so it is authoritative about **how things look and how copy sounds**, and not authoritative about **what the product is called or how it is organised**.

**Canonical**: colour, type, spacing, geometry, elevation, motion, iconography, and voice.

**Not canonical**: the kit's vocabulary and information architecture. Specifically —

- The kit ships a **`Findings`** section and screen. [`CONTEXT.md`](../../CONTEXT.md) rejects `Finding` and uses **`Signal`**. Keep `SeverityBadge` and the severity ramp; keep the word out of the interface.
- `Host` and `ScanRun` are likewise rejected. `Asset` is allowed as a UI collective noun only.
- The kit's readme says the product **fingerprints** what it finds. It does not — technology fingerprinting is out of scope on drift-integrity grounds.
- `ui_kits/app/` is a **reference look**, not an IA decision. It is Dashboard-first and findings-centric, and contains no drift screen at all — while drift is the thing this product exists to do.

When a visual convention and a domain term collide, the domain term wins and the visual convention gets re-skinned around it.

## A prototype is a dated record of a reading, never of a rule

[ADR-0075](../adr/0075-a-prototype-is-a-dated-record-of-a-reading-never-of-a-rule.md), from
[#131](https://github.com/winniel123/verge-asm/issues/131).

**Writing one.** Carry a **dateline on the rendered surface** — ticket, date, and one clause pointing
at the map's `THE CURRENT COMPOSED STATE` line as the only live absolutes. The `PROTOTYPE — throwaway`
provenance in the HTML comment is not enough: nobody reads a prototype in its source, and every
recorded instance of a prototype being believed happened in a browser.

**Meeting one whose figure a later ruling invalidated.** Two questions, in order:

1. **Did the ruling move a quantity?** Then it is a **figure**, it is dated, and **nothing is owed** —
   no rewrite, no mark, no ticket.
2. **Did the ruling make the drawn state unreachable** — an act the product now refuses, a population
   that must now be hollow, a sentence the product no longer says? Then it is a **rule drawn after its
   withdrawal**, and it is owed the mark.

**The mark**: leave the drawing standing, and add a dashed annotation box (`.anno`, as
`prototypes/seeds/` uses) on the **condemned variant or fill only** — never the whole file — naming
the ruling and **stating what the surface would draw now**. A strike with no successor is re-derived
by the next session that needs one ([ADR-0057](../adr/0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md) §5).
**Never redraw**: the wrong screen is the evidence.

**Who owes it**: the pass that supersedes, **and only where it already holds the prototype** — opened
it, cited it, or its ticket names it. Nobody ever owes a search for prototypes. This is
[#106](https://github.com/winniel123/verge-asm/issues/106)'s *grep the document you are writing in* at
one more hop: **mark the artefact you already opened.**

## Flag conflicts, don't silently resolve them

If a design need genuinely cannot be met inside the system — a component that does not exist, a colour role with no token — say so explicitly rather than inventing a one-off:

> _The drift timeline needs a "changed" treatment distinct from the severity ramp; the system has no token for it. Proposing `--drift-changed`._

Additions to the system are a decision, and get recorded like one.
