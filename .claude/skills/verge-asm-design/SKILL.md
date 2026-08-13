---
name: verge-asm-design
description: The Verge ASM design system — colors, type, spacing, components, and voice. Use for any interface, mock, prototype, slide, or production UI work on verge-asm. Invoke before writing markup or choosing a colour.
user-invocable: true
---

# Verge ASM design system

The system lives at **`design-system/`** in the repo root, not in this skill directory. This file is the entry point; `design-system/readme.md` is the full specification. Read it before doing anything non-trivial.

## Design idea

**Engineered paper.** An instrument, not a brochure: warm paper background, near-black ink, flat surfaces, 1px hairlines, sharp corners, one working blue. It should read like a well-set datasheet — dense, legible, completely unflashy. Severity colour is the loudest thing on any screen, on purpose.

## Key facts

- **Colour**: warm paper `#f7f7f4`, near-black ink `#16160f`, white working surfaces, one accent blue `#2d4fd4`. Severity scale: critical `#c92a2a` / high `#e8590c` / medium `#ffd43b` (+ ink text) / low `#1971c2` / info `#868e96`.
- **Type**: Helvetica stack for UI and prose; **IBM Plex Mono for every technical value** — hostnames, addresses, ports, fingerprints, versions, counts, timestamps. Base 13px/1.55. Signature motif is the **micro-label**: 10px mono semibold uppercase, 0.06em tracking.
- **Geometry**: 4px grid, **0px radius everywhere** (only status dots are circles), hairline rules `#d8d8d0`, **2px ink rules** for structure (under the app header, under table header rows).
- **Elevation**: no soft shadows. Floating layers get a **hard offset shadow** `6px 6px 0 rgba(22,22,15,.1)` (3px for small layers) plus a 1px ink border. Everything else is flat.
- **No** gradients, textures, photos, blur, transparency effects, or bouncy motion. Motion is functional: 120ms background/opacity, 240ms dialog fade. The one ambient animation is `verge-pulse` on live status dots.
- **Voice**: terse sentence case, imperative actions (`Run scan`, `Export CSV`), address the user as "you", never "we". **No emoji, no exclamation marks.** Empty states are fact + next action.
- **Wordmark**: typed — `Verge` (sans, bold) + an `ASM` mono chip. There is **no drawn logo; never invent one.**
- **Icons**: Lucide via CDN — a flagged substitution, not a shipped set. Unicode glyphs (`→ ★ ✓ ▾ ✕ ● ↑ ↓`) are idiomatic.

## Where things are

| Path | What |
| --- | --- |
| `design-system/styles.css` | Global entry — link this, then use the custom properties. Imports all of `tokens/`. |
| `design-system/tokens/` | `fonts` · `colors` · `typography` · `spacing` · `effects` · `base` |
| `design-system/components/` | `forms/` · `display/` · `feedback/` · `navigation/` — each `.jsx` has a `.d.ts` contract and a `.prompt.md` usage note |
| `design-system/guidelines/` | Foundation specimen cards |
| `design-system/ui_kits/` | `app/` (console) · `website/` (homepage) · `docs/` (docs page) |
| `design-system/Direction Explorations.dc.html` | The three original directions; `1c` was chosen |

Never hardcode a hex value that a token already names. Reach for `var(--…)`.

## Guardrails specific to this repo

The design system was authored from a product brief written **before** the domain model was settled, so parts of it contradict decisions this repo has since made. The **visual layer is canonical**; the kit's vocabulary and information architecture are **not**.

- **`Finding` is a rejected term.** [`CONTEXT.md`](../../../CONTEXT.md) rejects it and uses **`Signal`**. The kit ships a `Findings` nav section, a `Findings.jsx` screen, and a `SeverityBadge`. Keep the component and the severity ramp; **do not put the word "finding" in the interface.**
- **`Host` and `ScanRun` are also rejected**; `Asset` survives only as a UI collective noun ("847 assets"), never as a modelled thing. The four subjects are `Name`, `Address`, `Service`, `Endpoint`.
- **Technology fingerprinting is out of scope** — ruled out on drift-integrity grounds. The kit's readme describes the product as fingerprinting what it finds. It does not.
- **The kit's IA is not a decision.** Its Dashboard-first, findings-centric layout predates the map's thesis that **drift** is the product. There is no drift screen in the kit at all. Treat `ui_kits/app/` as *reference look*, not as the answer to what the screens are or what the landing view is.
- **`Signal` is derived and versioned, never diffed.** When the rule set changes, the interface must say *your rules changed*, never *your exposure changed* — a distinct visual treatment, not a footnote.
