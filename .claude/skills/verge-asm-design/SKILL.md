---
name: verge-asm-design
description: The Verge ASM design system — colors, type, spacing, components, and voice. Use for any interface, mock, prototype, slide, or production UI work on verge-asm. Invoke before writing markup or choosing a colour.
user-invocable: true
---

# Verge ASM design system

The system lives at **`design-system/`** in the repo root, not in this skill directory. This file is the entry point; `design-system/README.md` is the full handoff and `design-system/docs/AGENT-GUIDE.md` is the compact usage guide. Read those before doing anything non-trivial.

The old "engineered paper" datasheet look has been retired to `prototypes/design-system-legacy/` (frozen — styles + tokens only). Do **not** pull colours, type, or geometry from there; everything below is the current system.

## Design idea

**A calm instrument.** Clean, friendly, modern security SaaS you can watch all day — rounder, softer, roomier, more colourful than the old datasheet, while keeping the terse engineer voice, always-mono technical values, and an unmistakable five-level severity scale. Soft warm-stone surfaces, one confident azure for action, layered soft elevation. Severity is the only loud voice in the room, on purpose.

## Key facts

- **Colour** — warm-stone neutrals (light page `#f9f7f5` / ink `#231f19`; dark page `#15120f`, surfaces `#1e1b17` / `#282520`). One action **azure `#037ac0`** (dark mode `#6bbeff` on `#063352`) with hover / active / soft steps. Semantic **ok / warn / danger** each have text, solid, soft, and border tokens. Backgrounds are flat colour — no gradients, textures, or imagery in the console.
  - **Severity ramp** (its own five-level scale, AA in both modes). Light: Critical white on `#bf3631` · High `#a04400` on `#ffe9d6` · Medium `#8d5600` on `#ffeecc` · Low `#00728b` on `#d7f7ff` · Info `#536579` on `#ebf2f9`. Dark: Critical white on `#c44039` · High `#eba57b` on `#352414` · Medium `#d7b16a` on `#322713` · Low `#62c8df` on `#192c2e` · Info `#a9b3bf` on `#23272c`. Critical is the **only solid fill and the only pill-red**; High→Info are tinted pills with a solid dot, fading to stay ordinal. Use `SeverityBadge`.
  - **Drift palette** (change is a separate language): `--drift-gain-*` violet · `--drift-change-*` magenta · `--drift-loss-*` slate — AA-tuned, both modes. Carries `appeared / revealed / withdrawn / descoped / returned / changed` as **rounded-rect** chips (`ChangeBadge`). Never the severity ramp.
  - **Coverage + staleness**: `CoverageMeter` / `GapBadge` denominator + census reads; bronze `--stale-*` currency states (`StalenessBadge`). Chart series `--chart-1..4`. None of these reuse severity or drift colour.
- **Type** — **Instrument Sans** for UI + prose (headings semibold, −0.015em tracking); **Geist Mono** for *every* technical value, KPI numerals (28px semibold), and the micro-label. Base UI **13px** / 1.5; docs prose 15–16px. Fonts load via `@import` at the top of `tokens/typography.css` (self-host woff2 for production). Signature motif is the **micro-label**: 11px mono, medium, uppercase, 0.07em tracking, `--text-muted` (`OPEN SIGNALS`).
- **Geometry** — a real radius scale: **8** tags/code → **12** controls → **16** cards/tables → **24** dialogs → **pills** for chips. 4px spacing grid; 36px controls (30 compact, 44 marketing); cards pad 20px, pages 32px; tables dense (rows 10px/16px, 8px dense). Console maxes at 1440px, prose at 760px. Hairlines (`--border-default`) still do quiet work, but cards **lift with shadow instead of outlining with ink** — the old 2px structural ink rules are gone.
- **Elevation** — four **soft layered shadows** (`xs` flat controls → `sm` cards → `md` menus/popovers → `lg` dialogs). No hard offset shadows anywhere. Cards: `--surface` fill, 1px `--border-default`, `--r-lg`, `--shadow-sm`, 20px pad; emphasis via the micro-label header row, not heavier borders.
- **States & motion** — focus is a rounded 2px accent ring offset by 2px of surface (`box-shadow: 0 0 0 2px var(--surface), 0 0 0 4px var(--focus-ring)`), `:focus-visible` only. Selected row: `--row-selected` fill + 3px rounded accent bar on the left. Motion is functional and quick: 120ms hovers, 180ms control states, 280ms floating layers (`vg-pop-in`, 6px rise + 0.985 scale); toasts `vg-toast-in`. The one ambient animation is the scan-running pulse (`vg-pulse`, 1.8s ring). Nothing bounces. `prefers-reduced-motion` collapses all durations to 1ms (already in `tokens/base.css`).
- **No** gradients, textures, photos, or blur; the only transparency is the dialog scrim (`rgba(21,18,15,0.4)` light / `rgba(0,0,0,0.55)` dark, no blur). Surfaces are always opaque.
- **Voice** — terse sentence case everywhere (headings, buttons, labels, nav — never Title Case), imperative actions (`Add seed`, `Run scan`, `Export CSV`), address the user as "you", never "we". **No emoji, no exclamation marks.** Relative timestamps (`4m`) with ISO 8601 on hover; deltas signed with a true minus (`−5`). Empty states are fact + next action.
- **Icons** — **Lucide** via `components/media/Icon.jsx` (1.75px stroke at 16px, `currentColor`). The wrapper renders from the Lucide CDN UMD script; in a bundled app it's a **one-file swap to `lucide-react`** (same names) — every consumer goes through this one wrapper. Vocabulary: `radar` scans, `globe` domains, `server` IPs/services, `shield-alert` signals, `network` graph, `file-text` reports, `git-branch` drift. No emoji, no unicode glyphs as icons.
- **Logo** — the pulse glyph (`components/media/Logo.jsx`): a signal dot inside two watch rings, accent azure only, rings never animate in chrome. Explicitly **placeholder-quality**; never invent a different mark.
- **Dark mode** ships — set `data-theme="dark"` on `<html>` or any subtree root; tokens flip, no JS context needed. Warm graphite (not cool slate) keeps the brand temperature.

## Where things are

| Path | What |
| --- | --- |
| `design-system/styles.css` | Global entry — import once at the root; pulls all seven token files. |
| `design-system/tokens/` | `colors.css` · `typography.css` · `spacing.css` · `radius.css` · `elevation.css` · `motion.css` · `base.css` |
| `design-system/components/` | `forms/` · `display/` · `feedback/` · `navigation/` · `media/` — each component ships `.jsx` + `.d.ts` contract + `.prompt.md` usage note |
| `design-system/components/media/Icon.jsx` | The single Lucide wrapper every consumer routes through (one-file swap to `lucide-react`) |
| `design-system/examples/` | Design references: `console/` (screens + `ConsoleApp.jsx` shell), `Homepage.jsx` (marketing), `DocsPage.jsx` (docs) — recreate their structure against real routing/data, don't ship verbatim |
| `design-system/examples/console/` | Dashboard · Scope · Inventory · Drift · Signals · GraphView · Reports · Settings · SignIn · Integrations |
| `design-system/docs/AGENT-GUIDE.md` | Compact usage guide for agents (good human quick-start) |
| `design-system/docs/DESIGN-NOTES.md` | Full rationale: palette math, severity contrast tables, component history, production notes |

Never hardcode a hex value that a token already names. Reach for `var(--…)`.

## Authoring and editing in `design-system/`

`design-system/` is the shared UI asset home and may be edited in-repo — templates, tokens, and components alike. (The former handoff workflow, where components were authored only in Claude Design and imported wholesale, was retired 2026-08-28; ADR-0109 and ADR-0116 are superseded.)

Editing here is now ordinary work, but the whole point of a design system is coherence, so hold the line while you do it:

- **Reuse before adding.** The system already ships ~110 components across `forms/ display/ feedback/ navigation/ media/`, often under a domain-correct name (`AnnotationControl`, `ChangeBadge`, `CoverageMeter`, …). Read the candidate's `.prompt.md` before concluding one is missing.
- **Stay in the token vocabulary.** Use `var(--…)`; never hardcode a value a token already names. A new component keeps the one type scale, one severity ramp, one elevation model, and light/dark in step.
- **A new component ships its contract.** When you do add one, give it the same trio the rest carry: `Name.jsx`, `Name.d.ts` (props), `Name.prompt.md` (usage note), so the next session can use it.

## Domain guardrails (the domain term always wins over a visual convention)

The vocabulary is load-bearing and enforced in copy, not just style. The new system already ships **domain-correct** components (`ChangeBadge`, `AnnotationControl`, the Drift and Signals screens) — keep it that way; never regress the interface to a rejected term.

- **`signal`** never *finding*. Signals are **withdrawn by the world** (the world moved), never *resolved* by operators. `WithdrawnMark` / `AnnotationControl` model this; the one operator dial is an annotation (accepted risk + reason), never a mute/status/triage.
- **`seed` / `scope`** never *target*.
- **`Name` / `Address` / `Service` / `Endpoint`** are the four subjects — never *host / IP / port / URL* as modelled things. `asset` survives only as a UI collective noun ("847 assets").
- **`channel`** never *webhook / integration*.
- **`vantage`** never *probe / scanner / agent*.
- **`annotation`** never *mute / status / triage*.
- **No technology fingerprinting** — ruled out on drift-integrity grounds. The product discovers, watches for drift, and raises signals; it does not fingerprint what it finds.
- **Change vocabulary is its own palette** — `appeared / revealed / withdrawn / descoped / returned / changed` render in the drift palette (violet / magenta / slate) as rounded-rect chips, **never** the severity ramp.
- **Severity is exactly Critical / High / Medium / Low / Info** — five levels, those exact words, via `SeverityBadge`. Never restyle, rename, extend, or recolour the levels.
