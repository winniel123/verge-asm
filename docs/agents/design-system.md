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

**The form is fixed, and it is copied rather than composed** —
[#144](https://github.com/winniel123/verge-asm/issues/144) applied it to all eleven prototypes that
lacked it, so a new prototype matches what the other thirteen already draw. Paste the CSS block —
`prototypes/seeds/`'s dashed accent box, no new token — and render `dateline()` as the **first child
of `.shell`**, from inside the file's single `chrome()` / `shell()` function so it draws on every
variant, fill, screen and state:

```css
  .anno { border: 1px dashed var(--accent); background: var(--accent-soft);
          padding: var(--space-3) var(--space-4); margin-bottom: var(--space-5);
          font-size: var(--text-sm); line-height: var(--leading-body); max-width: 92ch; }
  .anno .ml { display: block; margin-bottom: var(--space-2); color: var(--accent); }
  .anno.no { border-color: var(--danger); background: var(--danger-soft); }
  .anno.no .ml { color: var(--danger); }
  .anno p { margin: 0 0 var(--space-2); }
  .anno p:last-child { margin-bottom: 0; }
  .anno code { font-family: var(--font-mono); font-size: var(--text-sm); }
```

```html
  <div class="anno">
    <span class="ml">prototype · not part of the design</span>
    <p><b>Issue #N — "&lt;the question the ticket asked&gt;". Drawn YYYY-MM-DD.</b> Throwaway:
      something to decide against, not production UI and not a component library.</p>
    <p><b>Every quantity on these screens is a dated reading, not a current value</b> — the corpus
      as it stood on that date, and nothing here has been re-filled since. The only live absolutes
      are the map's <code>THE CURRENT COMPOSED STATE</code> line (issue #1, <i>Notes</i>): read a
      figure there before believing one here. What is <i>not</i> a dated reading is the state these
      screens draw — the acts they offer, the populations they show as non-empty and the sentences
      they put in the product's mouth. A later ruling that makes one of those unreachable is marked
      here in place and never redrawn.</p>
  </div>
```

Three riders on filling it in:

- **The date is the date the prototype was *drawn*, never the date the dateline was added.** For a
  new prototype those are the same day. For one being marked later, read it off the file's own
  history — `git log --diff-filter=A --date=short -- prototypes/<name>/index.html` — because
  stamping today's date on a two-day-old reading makes the dateline the false-currency artefact it
  exists to prevent.
- **Where a later ticket restated copy on the rendered surface, name it too**, as `#28`'s and
  `#47`'s datelines do. The dateline's three parts are the ticket, the date and the composed-state
  clause; a second hand on the surface is part of the first.
- **The dateline is artefact-scoped and unconditional; the mark below is drawing-scoped.** They are
  two different objects and neither substitutes for the other. `prototypes/signal-evaluability/`
  carries both and keeps them apart.

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
