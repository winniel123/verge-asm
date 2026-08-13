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

## Flag conflicts, don't silently resolve them

If a design need genuinely cannot be met inside the system — a component that does not exist, a colour role with no token — say so explicitly rather than inventing a one-off:

> _The drift timeline needs a "changed" treatment distinct from the severity ramp; the system has no token for it. Proposing `--drift-changed`._

Additions to the system are a decision, and get recorded like one.
