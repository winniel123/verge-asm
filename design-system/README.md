# Handoff: Verge ASM design system

## Overview
Complete design system for Verge ASM — a free, open-source (AGPL-3.0), self-hostable attack surface management app. It covers three surfaces (product console, marketing site, docs site): design tokens with light + dark modes, ~110 React components, composition examples for eleven screens, and the content/vocabulary rules the product depends on.

## About these files
Two kinds of material, and the distinction matters:

1. **`tokens/`, `styles.css`, and `components/` are portable source, not just references.** Every component is plain React (function components + hooks) styled with CSS custom properties. No npm dependencies beyond React itself — sibling `import`s only. You can lift these files into the target codebase nearly as-is (see Integration), or treat them as the exact spec while re-implementing on your own primitives. Each component ships three files: `Name.jsx` (implementation), `Name.d.ts` (props contract), `Name.prompt.md` (usage note + example).
2. **`examples/` are design references.** The console screens (`examples/console/*.jsx`), marketing homepage, and docs page show how the components compose into real screens, with realistic data inline. They were authored in a design workspace — recreate their structure in your app's routing/data layer rather than shipping them verbatim. (Their two HTML preview shells were workspace scaffolding and are not included.)

## Fidelity
**High-fidelity.** Colors, type, spacing, radii, shadows, motion, and copy are final. Both themes are AA-checked (severity ramp ratios are documented in `docs/DESIGN-NOTES.md`). Recreate pixel-perfectly; where your stack forces a substitution, match the token values.

## Dependencies
- **React 18+** (components use hooks; no class components, no portals required)
- **Fonts** — Google Fonts: `Instrument Sans` (UI/prose, 400–700 + italics) and `Geist Mono` (all technical values). Loaded via `@import` at the top of `tokens/typography.css`; self-host the woff2s for production.
- **Icons** — Lucide. The kit's `components/media/Icon.jsx` renders via the Lucide UMD script (`https://unpkg.com/lucide@latest/dist/umd/lucide.min.js`). In a bundled codebase, replace Icon's internals with `lucide-react` (same names: `radar`, `shield-alert`, `git-branch`, …) — every consumer goes through this one wrapper, so it's a one-file swap.
- **No other runtime deps.** Charts, graph, calendar, virtual table, video player are hand-rolled on SVG/DOM.

## Integration steps
1. Copy `tokens/` and `styles.css` into the app; import `styles.css` once at the root. It pulls the seven token files (colors, typography, spacing, radius, elevation, motion, base resets — slim scrollbars, selection, reduced-motion handling included).
2. Copy `components/` (five folders: `forms/`, `display/`, `feedback/`, `navigation/`, `media/`). Keep the folder structure — imports are relative (`../media/Icon.jsx`). JSX compiles under any standard React build (Vite, Next, CRA); for TypeScript, rename to `.tsx` and fold in the adjacent `.d.ts` types.
3. Swap `Icon.jsx` to `lucide-react` (step above) and self-host fonts.
4. **Dark mode**: set `data-theme="dark"` on `<html>` (or any subtree root). No JS theme context needed — tokens flip.
5. Rebuild screens from `examples/` against real routing and data. `examples/console/ConsoleApp.jsx` shows shell wiring: TopNav, org switcher, ⌘K palette, toast stack, theme toggle.
6. Replace the placeholder `Logo` (pulse glyph) when a real brand mark exists — it is explicitly placeholder-quality.

## Screens (examples/)
Console (`examples/console/`): **Dashboard** (KPIs, severity mix, vantage health, running-scan banner) · **Scope** (seeds, TagInput validation + refusals, proposals, exclusions, custody, coverage) · **Inventory** (saved views, column picker, density, hover peeks) · **Drift** (change timeline grouped by batch, diff views) · **Signals** (sortable table, tabs Open/Annotated/Withdrawn, detail drawer with annotation control, context menus, typed descope confirm) · **Graph** (pan/zoom/minimap, node drawer) · **Reports** (time-series chart, heatmap calendar, schedule wizard) · **Settings** (scans, vantages, channels, messages, delivery, access/SSO, integrations) · **SignIn** (credentials + TOTP CodeInput + IdP SSO). Plus `Homepage.jsx` (marketing) and `DocsPage.jsx` (docs with sidebar, version picker, callouts, FAQ).

## Interactions & behavior conventions
- All inputs are **controlled** (`value` + `onChange`); overlays are controlled (`open` + `onClose`).
- Focus: 2px accent ring offset by 2px surface, `:focus-visible` only. Dialog/Drawer/CommandPalette trap Tab and restore focus on close. Toasts announce via a polite live region.
- Motion tokens: 120ms hover, 180ms control, 280ms floating layers (`vg-pop-in`); exits animate before unmount (Drawer 280ms, Banner/Toast collapse-fade 240ms). `prefers-reduced-motion` collapses all durations to 1ms globally — already in `tokens/base.css`.
- Destructive acts always pass through `ConfirmDialog` (typed-name gate for the worst); menus never fire destruction directly.
- Tables: row click opens detail, roving keyboard focus (j/k/arrows, Enter), opt-in `virtual` windowing for long lists (fixed row height; use a measured virtualizer for variable heights in production).

## Design tokens
Everything lives in `tokens/*.css` as custom properties — treat the files as the source of truth. Headlines: warm-stone neutrals (page `#f9f7f5` / ink `#231f19` light; graphite `#15120f` / `#1e1b17` dark), action azure `#037ac0` (dark mode `#6bbeff` on `#063352`), semantic ok/warn/danger in text/solid/soft/border steps, the five-level severity ramp (Critical is the only solid fill and the only pill-red), drift palette (violet gain / magenta change / slate loss), coverage + staleness tones, chart series `--chart-1..4`. Radii 8/12/16/24 + pills. Four layered shadows. 4px spacing grid; 36px controls; 13px base UI size.

## Content rules (fixed — enforced in copy, not just style)
Sentence case everywhere; imperative verbs on actions (`Add seed`, `Run scan`); no exclamation marks, no emoji; terse relative timestamps (`4m`) with ISO 8601 on hover; signed deltas with a true minus (`−5`); technical values always mono. **Vocabulary is load-bearing:** *signal* never finding · *seed/scope* never target · *channel* never webhook/integration · *vantage* never probe/scanner/agent · *annotation* never mute/triage · signals are *withdrawn* by the world, never "resolved" by operators. Full glossary and rationale in `docs/DESIGN-NOTES.md`.

## Assets
None bundled — by design. Icons come from Lucide (CDN or `lucide-react`); fonts from Google Fonts; the logo is a code-drawn placeholder (`components/media/Logo.jsx`); marketing "imagery" is composed product UI, not photos. Nothing else to license or migrate.

## Files
- `README.md` — this document
- `styles.css` — single import that pulls all tokens
- `tokens/` — colors, typography, spacing, radius, elevation, motion, base (7 files)
- `components/forms|display|feedback|navigation|media/` — ~110 components × (`.jsx` + `.d.ts` + `.prompt.md`)
- `examples/console/` — 11 screen compositions + shell (`ConsoleApp.jsx`) + screen README
- `examples/Homepage.jsx`, `examples/DocsPage.jsx` — marketing and docs surfaces
- `docs/DESIGN-NOTES.md` — full design rationale: palette math, severity contrast tables, per-batch component history, production notes
- `docs/AGENT-GUIDE.md` — compact usage guide written for AI agents working in this system (also a good human quick-start)
