# Design System

Anything with a visual surface — production UI, throwaway prototype, mock, diagram, slide, screenshot — uses the **Verge ASM design system**. There is no second visual language in this project, and no "just for now" exception for prototypes: a prototype in the wrong visual language answers the wrong question.

## Before writing any markup

Invoke the **`verge-asm-design`** skill (`.claude/skills/verge-asm-design/SKILL.md`). It carries the key facts inline and points at the system itself.

The system lives at **`design-system/`** in the repo root:

- `styles.css` — the global entry. Link it once, then use its custom properties. It pulls the seven token files under `tokens/`.
- `tokens/` — `colors.css`, `typography.css`, `spacing.css`, `radius.css`, `elevation.css`, `motion.css`, `base.css`. Treat these as the source of truth.
- `components/` — five folders: `forms/` · `display/` · `feedback/` · `navigation/` · `media/`. Every component ships three files: `Name.jsx` (implementation), `Name.d.ts` (props contract), and `Name.prompt.md` (usage note + example). **Read the `.prompt.md` before using a component.** Imports between components are relative (`../media/Icon.jsx`) — keep the folder structure.
- `examples/` — **the IA spec, ported verbatim** ([ADR-0110](../adr/0110-the-design-system-examples-are-the-consoles-ia-spec-ported-verbatim.md)): `examples/console/` (the console screens + `ConsoleApp.jsx` shell + a screen README), plus `examples/Homepage.jsx` (marketing) and `examples/DocsPage.jsx` (docs). Port each console screen's composition verbatim — same components, layout, spacing, hierarchy, and copy — translating the reference JSX into the app's server-rendered Go templates and swapping only inline sample data for real data of the same shape. `ConsoleApp.jsx` is the shell spec; `screenshots/` are ground truth. Earlier guidance to "rebuild their structure rather than ship them verbatim" is **superseded** — this was the only clause ADR-0110 reversed.
- `docs/AGENT-GUIDE.md` — the compact, agent-facing usage guide (also a good human quick-start). Start here.
- `docs/DESIGN-NOTES.md` — the full rationale: palette math, severity contrast tables, per-batch component history, production notes. Read it for anything non-trivial.

**Never hardcode a value a token already names.** If you find yourself typing `#231f19`, you want `var(--ink)`.

The old "engineered paper" system is frozen to **`design-system-legacy/`** (styles.css + tokens/ + README only). It exists solely for the dated prototypes (see ADR-0075 below); do not use it for new work.

## What is canonical and what is not

The new system was built from the redesign brief and is **already domain-correct** in most places — it ships `signal` never finding, `seed/scope` never target, `channel` never webhook, `vantage` never probe, `annotation` never mute/triage, and it carries drift as a first-class surface (`ChangeBadge`, `TransitionMarker`, the drift palette) and the operator dial as `AnnotationControl`. So the canonical/not-canonical split is narrower than under the old kit, but it still holds: the system is authoritative about **how things look and how copy sounds**, not about **what the product is called or how it is organised** where the two ever diverge.

**Canonical**: colour, type, spacing, geometry, elevation, motion, iconography, and voice.

**Not canonical**: vocabulary and information architecture. The domain owns those. Guardrails, all of which the interface must honour:

- **`signal`, never `finding`.** [`CONTEXT.md`](../../CONTEXT.md) rejects `Finding`. Keep `SeverityBadge` and the severity ramp; keep the word "finding" out of the interface.
- **Domain nouns, not wire nouns.** Say `Name` / `Address` / `Service` / `Endpoint`, never `host` / `IP` / `port` / `URL`. `Host` and `ScanRun` are rejected; `asset` is allowed only as a UI collective noun.
- **No fingerprinting.** The product does not fingerprint the technology it finds — technology fingerprinting is out of scope on drift-integrity grounds. Do not draw or imply it.
- **Drift is the thesis.** Drift is the thing this product exists to do; it is never a secondary screen. The canonical console keeps Drift a top-level screen (nav item 4 of 7), so porting the examples verbatim honours this rather than fighting it — [ADR-0110](../adr/0110-the-design-system-examples-are-the-consoles-ia-spec-ported-verbatim.md) reversed the "examples are not an IA ruling" clause precisely because the examples now *carry* drift-as-thesis; never let a dashboard-first reading demote drift.
- **Severity is exactly `Critical / High / Medium / Low / Info`.** Five levels, those exact words. Use `SeverityBadge`; never rename, add, drop, or restyle a level. Critical is the ramp's only solid fill.
- **Change is its own palette.** The change vocabulary (`appeared / revealed / withdrawn / descoped / returned / changed`) rides the drift tokens (`--drift-gain-*` violet · `--drift-change-*` magenta · `--drift-loss-*` slate) as rounded-rect chips. Severity stays the only pill and the only red; signals leave by being **withdrawn** by the world, never "resolved" by an operator.

When a visual convention and a domain term collide, the domain term wins and the visual convention gets re-skinned around it.

## Requesting a component (never build one here)

> **Verge ASM does not author design-system components. All components are created in Claude Design and imported into `design-system/`. When a screen needs a component the system does not have, do not build it here: write a component-request markdown file from `design-system/COMPONENT-REQUEST.md`, and hand it to the user to give to Claude Design. Restyling within existing tokens/components is fine; creating a new component file in this repo is not.**

This is [ADR-0109](../adr/0109-design-system-components-are-authored-in-claude-design-and-imported.md). Restyling an existing component or template class **within the existing token vocabulary** is ordinary in-repo work; importing a component back from Claude Design (dropping its `.jsx` + `.d.ts` + `.prompt.md` into `components/`) is the intended flow; **creating a new component file in this repo is not.** No repo-authored component is grandfathered — the migration added none, and this keeps it that way.

**The handoff, end to end:**

1. **Confirm the component truly does not exist.** Look across all five `design-system/components/` folders (`forms/`, `display/`, `feedback/`, `navigation/`, `media/`) — the system already ships ~110 components, so most needs are already met, possibly under a domain-correct name (`AnnotationControl`, `ChangeBadge`, `CoverageMeter`, `VantageCard`, …). Read the candidate's `.prompt.md` before concluding it is missing.
2. **Write the request.** Copy `design-system/COMPONENT-REQUEST.md`, fill it in, and save it under `design-system/requests/<component-name>.md`.
3. **Hand it to the user** to give to Claude Design. The component is authored there, comes back as `.jsx` + `.d.ts` + `.prompt.md`, and is imported into `design-system/components/`.

Where the round-trip's wait is unacceptable, the correct move is still the request plus an explicit note of the blockage — never a repo-authored component.

## A prototype is a dated record of a reading, never of a rule

[ADR-0075](../adr/0075-a-prototype-is-a-dated-record-of-a-reading-never-of-a-rule.md), from
[#131](https://github.com/winniel123/verge-asm/issues/131). This guidance is unaffected by the redesign,
with one path change: **prototypes link `design-system-legacy/styles.css`** (the frozen "engineered
paper" system), since the dated prototypes were drawn against it and must not silently re-skin.

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

If a design need genuinely cannot be met inside the system — a colour role with no token, a token that fights a domain term — say so explicitly rather than inventing a one-off:

> _The drift timeline needs a "changed" treatment distinct from the severity ramp; the system has no token for it. Proposing `--drift-changed`._

Additions to the token system are a decision, and get recorded like one. A **missing component** is not this case — that is a Claude Design request (see "Requesting a component" above), never an in-repo build.
