# Verge ASM design system

The shared UI asset home for Verge ASM — a free, open-source (AGPL-3.0), self-hostable attack surface management app. It holds the design tokens (light + dark), the served HTML templates, the React components, composition examples, and the content/vocabulary rules the product depends on.

This directory is the **source of truth** and may be edited in-repo. (The former design-system handoff workflow — where markup was authored in a separate package and byte-compared into this repo through gates G1/G2 — was retired 2026-08-28; see superseded ADR-0109 and ADR-0116.)

## Who consumes what

- **The web app** (`cmd/web`) embeds and serves `templates/*.tmpl`, `tokens/*.css`, and `fixtures/fixtures.json` through `designfs.go` (`//go:embed`). Editing a template here changes the served UI directly.
- **The docs-site** (`docs-site/`, Astro) imports `components/**` and `tokens/*.css` via its `@ds/*` alias (`docs-site/astro.config.mjs`, `docs-site/tsconfig.json`).

## Files

- `designfs.go` — embeds `templates/`, `tokens/`, and `fixtures/` and exposes them as a read-only `fs.FS` for the web app.
- `templates/` — the served Go `html/template` files (one per screen).
- `tokens/` — colors, typography, spacing, radius, elevation, motion, base (7 files).
- `styles.css` — single import that pulls all seven token files.
- `components/forms|display|feedback|navigation|media/` — React components (`.jsx` + `.d.ts` + `.prompt.md`) consumed by the docs-site.
- `examples/console/` — screen/flow compositions + shell (`ConsoleApp.jsx`); the console's IA reference.
- `examples/Homepage.jsx`, `examples/DocsPage.jsx` — marketing and docs surfaces.
- `fixtures/fixtures.json` — the fixture corpus the web app renders in dev/test.
- `docs/DESIGN-NOTES.md` — full design rationale: palette math, severity contrast tables, component history, production notes.
- `docs/AGENT-GUIDE.md` — compact usage guide for agents working in this system.
- `docs/DOCS-IA.md` — the docs-site left-rail information architecture.

## Design tokens

Everything lives in `tokens/*.css` as custom properties — treat the files as the source of truth. Warm-stone neutrals (page `#f9f7f5` / ink `#231f19` light; graphite `#15120f` / `#1e1b17` dark), action azure `#037ac0` (dark mode `#6bbeff` on `#063352`), semantic ok/warn/danger in text/solid/soft/border steps, the five-level severity ramp (Critical is the only solid fill and the only pill-red), drift palette (violet gain / magenta change / slate loss), coverage + staleness tones, chart series `--chart-1..4`. Radii 8/12/16/24 + pills. Four layered shadows. 4px spacing grid; 36px controls; 13px base UI size. Both themes are AA-checked (ratios in `docs/DESIGN-NOTES.md`). **Dark mode**: set `data-theme="dark"` on `<html>` (or any subtree root) — no JS theme context needed, tokens flip.

## Dependencies (components)

- **React 18+** (hooks; no class components).
- **Fonts** — Google Fonts: `Instrument Sans` (UI/prose) and `Geist Mono` (technical values), loaded via `@import` in `tokens/typography.css`.
- **Icons** — Lucide, via the single `components/media/Icon.jsx` wrapper.
- No other runtime deps — charts, graph, calendar, virtual table, and video player are hand-rolled on SVG/DOM.

## Interactions & behavior conventions

- All inputs are **controlled** (`value` + `onChange`); overlays are controlled (`open` + `onClose`).
- Focus: 2px accent ring offset by 2px surface, `:focus-visible` only. Dialog/Drawer/CommandPalette trap Tab and restore focus on close.
- Motion tokens: 120ms hover, 180ms control, 280ms floating layers; `prefers-reduced-motion` collapses all durations to 1ms (in `tokens/base.css`).
- Destructive acts always pass through `ConfirmDialog` (typed-name gate for the worst); menus never fire destruction directly.
- Tables: row click opens detail, roving keyboard focus (j/k/arrows, Enter), opt-in `virtual` windowing for long lists.

## Content rules (enforced in copy, not just style)

Sentence case everywhere; imperative verbs on actions (`Add seed`, `Run scan`); no exclamation marks, no emoji; terse relative timestamps (`4m`) with ISO 8601 on hover; signed deltas with a true minus (`−5`); technical values always mono. **Vocabulary is load-bearing:** *signal* never finding · *seed/scope* never target · *channel* never webhook/integration · *vantage* never probe/scanner/agent · *annotation* never mute/triage · signals are *withdrawn* by the world, never "resolved" by operators. Full glossary and rationale in `docs/DESIGN-NOTES.md`.
