<!-- Fill this in, save under `design-system/requests/<name>.md`, hand to the user for Claude Design. Do not build the component in this repo. -->

# Component request

A fill-in-the-blanks handoff for a **new** Verge ASM design-system component. Fill every section, then hand this file to the user to give to Claude Design.

The rule this template exists to serve (see `AUTHORING.md` and [ADR-0109](../docs/adr/0109-design-system-components-are-authored-in-claude-design-and-imported.md)):

> Verge ASM does not author design-system components. All components are created in Claude Design and imported into `design-system/`. When a screen needs a component the system does not have, do not build it here: write a component-request markdown file from `design-system/COMPONENT-REQUEST.md`, and hand it to the user to give to Claude Design. Restyling within existing tokens/components is fine; creating a new component file in this repo is not.

Reference material to match against before you write: `docs/DESIGN-NOTES.md` (tokens, voice, severity/change/coverage rules) and the existing `components/` tree.

---

## 1. Component name and folder

- **Name** (system naming style — PascalCase, terse, domain vocabulary; e.g. `SeverityBadge`, `CoverageMeter`, `AnnotationControl`):
  `______`
- **Folder** (pick one): `forms/` · `display/` · `feedback/` · `navigation/` · `media/`
  `______`
- **One-line purpose** (what it is, in the system's voice — sentence case, terse):
  `______`

## 2. Why it's needed

- **The screen / use it unblocks** (which console, marketing, or docs surface, and what can't ship without it):
  `______`
- **No existing component covers it.** List the near-misses you checked in `components/` and why each falls short:
  - `______` — rejected because `______`
  - `______` — rejected because `______`
  - `______` — rejected because `______`

  (If any near-miss can be reached by restyling within existing tokens/components, stop — that is not a new component. See `AUTHORING.md`.)

## 3. Behavior contract

- **Controlled props** — this system's components are controlled. Name them explicitly:
  - Value: `value` / `onChange` (or `______`)
  - Open/dismiss (if a floating or dismissible layer): `open` / `onClose` (or `______`)
  - Other required props (with types): `______`
- **States** — describe each that applies (reference the states section of `docs/DESIGN-NOTES.md`):
  - Hover: `______`
  - Press: `______`
  - Selected: `______`
  - Focus (2px accent ring, offset by 2px surface): `______`
  - Disabled (45% opacity): `______`
- **Keyboard / focus expectations** (tab order, arrow/roving focus, Enter/Escape, focus trap + restore for floating layers, live-region announcements):
  `______`

## 4. Domain fit

- **Vocabulary** it must respect (from `docs/DESIGN-NOTES.md` § Content fundamentals — e.g. **signal** not finding, **seed/scope** not target, **channel** not webhook, **vantage** not probe, **annotation** not mute, signals are **withdrawn** not resolved):
  `______`
- **Palette rules** — call out which of these languages the component lives in, and confirm it does not borrow another's color:
  - **Severity** — Critical / High / Medium / Low / Info; Critical is the only solid chip and the only red: `______`
  - **Change / drift** — appeared / revealed / withdrawn / descoped / returned / changed (`--drift-*`): `______`
  - **Coverage / gaps / staleness** — `--stale-*` bronze, gap/denominator reads (never severity or drift color): `______`
  - **Chart series** — `--chart-1..4`, never severity colors: `______`
- **Copy patterns** it carries (sentence case, imperative actions, monospace technical values, terse relative timestamps, empty state = fact + next action):
  `______`

## 5. Tokens only

Build entirely on existing tokens (`tokens/colors.css`, `typography.css`, `spacing.css`, `radius.css`, `elevation.css`, `motion.css`). Do **not** invent a value.

- Tokens this component will use: `______`
- **Token gaps** — if a needed color/size/radius/shadow/duration does not exist, flag it here explicitly (do not hardcode a substitute). Name the gap and propose where it belongs:
  `______`

## 6. Deliverables expected back

From Claude Design, for import into the correct `components/<folder>/`:

- `Name.jsx` — the component
- `Name.d.ts` — the typed prop contract
- `Name.prompt.md` — usage guide (feeds `ui_kits/docs/reference.html`)
- Renders correctly in **light and dark** mode
- All text/background pairs clear **WCAG AA** (≥ 4.5:1 normal text)
