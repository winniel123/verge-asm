# Verge ASM design system

**Verge ASM** is a free, open-source (AGPL-3.0) defensive **attack surface management** web application. It continuously discovers an organization's internet-facing assets — domains, subdomains, IPs, open ports, services, TLS certificates — fingerprints what it finds, raises findings on a Critical/High/Medium/Low/Info scale, renders the surface as a graph, and exports reports. It is self-hostable and community-driven.

**Audiences:** security engineers and red/blue teams, IT admins and sysadmins, self-hosters and homelabbers, and MSPs managing many client orgs. The open-source identity is prominent: version string, AGPL-3.0 license, and GitHub link are visible in the product chrome.

**Surfaces covered:** the web app (console), the marketing/landing site, and the docs site.

**Sources:** no codebase, Figma, or brand assets were provided. This system was designed from scratch; the direction ("utilitarian open-source light") was chosen by the user from three explorations in `Direction Explorations.dc.html`.

**Logo:** none was provided and none was invented. Wherever a mark would go, the wordmark is set in plain type: `Verge` (sans, bold) followed by an `ASM` chip (mono, white-on-ink). If a real logo exists, drop it into `assets/` and update the components.

---

## Design idea

**Engineered paper.** An instrument, not a brochure: warm paper background, near-black ink, flat surfaces, 1px hairlines, sharp corners, one working blue. It should feel like a well-set datasheet — dense, legible, and completely unflashy. Severity color is the loudest thing on any screen, on purpose.

---

## CONTENT FUNDAMENTALS

**Tone: terse and precise, with open-source warmth.** Written by an engineer who respects your time. Never marketing-breathless, never cute.

- **Sentence case everywhere** — headings, buttons, labels, nav. Never Title Case. (`Latest findings`, not `Latest Findings`.)
  - Exception: mono **micro-labels** are uppercase by CSS (`OPEN FINDINGS`).
- **Imperative verbs on actions:** `Add target`, `Run scan`, `Export CSV`, `Acknowledge`. No "please", no gerunds.
- **Technical values are always mono:** hostnames, IPs, ports, CVE ids, hashes, versions, counts, timestamps. `edge-gw-03.acmecorp.io`, `CVE-2026-1187`, `:5900`.
- **Terse relative timestamps:** `4m`, `22m`, `1h`, `3d` — absolute ISO 8601 on hover/detail (`2026-07-30 14:02 UTC`).
- **Numbers:** thousands separators (`1,284`); deltas signed (`+12`, `−5` with a true minus).
- **No exclamation marks. No emoji. No jargon-softening.** Say `4 critical findings`, not "Uh oh!".
- **Address the user as "you"; never "we".** It's self-hosted software, not a vendor speaking. Passive is fine for the scanner: `Scan completed`, `9 new services discovered`.
- **Empty states = fact + next action:** `No targets yet. Add a domain or CIDR range to start scanning.`
- **Severity words are exact and never synonymized:** Critical, High, Medium, Low, Info.
- **Community strings are plain:** `v0.9.2`, `AGPL-3.0`, `Star on GitHub`, `Report an issue`.

**Sample voice:**
> Scan completed in 4m 12s. 1,284 assets checked, 9 new services, 1 new critical finding.
> Verge ASM is free software, licensed AGPL-3.0. Self-host it, read it, fork it.

---

## VISUAL FOUNDATIONS

**Color.** Warm paper (`#f7f7f4`) page, white working surfaces, near-black ink (`#16160f`). One accent: working blue `#2d4fd4` for links, active nav, selection, focus. Semantic green/amber/red for status. The severity scale (red `#c92a2a` → orange `#e8590c` → yellow `#ffd43b` → blue `#1971c2` → gray `#868e96`) is the loudest color on screen; keep everything else quiet so it reads. Medium-yellow takes ink text, all other severity fills take white.

**Type.** Two families: the Helvetica stack for UI and prose; IBM Plex Mono for everything technical (see CONTENT). Base 13px/1.55. KPI numerals 26px mono semibold. Signature motif: the **micro-label** — 10px mono semibold uppercase, 0.06em tracking, muted color — used for table heads, KPI captions, and section eyebrows. Large sans headings track slightly tight (−0.01em).

**Spacing & density.** 4px grid. Dense-but-readable: table rows pad 9px vertical / 16px horizontal; controls are 32px tall (26px compact in tables, 40px marketing). Cards pad 16px, pages 24px, app content maxes at 1440px, prose at 720px.

**Backgrounds.** Flat paper. No gradients, no textures, no photos, no blur, no transparency effects. Inverted ink panels (`#16160f`) are the only "rich" background — used for terminal/log blocks and marketing bands.

**Borders & rules.** Hairlines do the layout work: `#d8d8d0` default, `#ecece4` between rows. Ink rules mark structure: **2px ink under the app header and under table header rows**; 1px ink outlines emphasized containers (dialogs, terminal blocks, marketing frames). Ordinary cards: white with a 1px `#d8d8d0` border.

**Corners.** 0px radius, everywhere. The only circles are status dots.

**Elevation.** No soft shadows. Floating layers (dialogs, menus, tooltips) sit on a **hard offset shadow** — `6px 6px 0 rgba(22,22,15,.1)` (3px for small layers) — plus a 1px ink border. Everything else is flat.

**States.** Hover: background shifts to sunken `#f2f2ee` (rows, ghost buttons), links underline, primary buttons lighten to `#3a3a30`. Press: primary returns to full ink; no scale transforms. Selected row: `#e9edfb` accent-soft fill with a 2px accent inset bar on the left edge. Focus: square ring — 2px paper + 2px accent (`--focus-ring`). Disabled: 45% opacity, no color changes.

**Motion.** Functional only: 120ms ease background/opacity shifts; 240ms for dialog fade-in. The one ambient animation is `verge-pulse` (1.6s opacity blink) on live status dots. Nothing bounces, slides, or springs.

**Imagery.** The app uses none. Marketing/docs use typographic compositions, terminal blocks, and real product UI in 1px-ink frames instead of illustration or photography.

---

## ICONOGRAPHY

No icon set was provided, so the system standardizes on **Lucide** (ISC licensed, CDN-served) — its 1.75px stroke, square-cap geometry matches the hairline aesthetic. **This is a substitution; flagged.** If the product later ships its own set, swap at the `Icon` usage sites.

- Load: `<script src="https://unpkg.com/lucide@latest"></script>` then `lucide.createIcons()`, or copy inline SVGs from lucide.dev.
- Sizes: 14px inline-with-text, 16px in buttons/table cells, 18px nav. Stroke 1.75. Color: inherit (`currentColor`).
- Canonical picks: `radar` (scan), `globe` (domains), `server` (hosts), `shield-alert` (findings), `share-2` (graph), `file-text` (reports), `settings`, `github`, `search`, `plus`, `download`, `chevron-down`, `x`, `check`, `external-link`.
- **Unicode as functional glyphs is idiomatic:** `→` view-all links, `★` GitHub star, `●` status dots (styled span, not char), `↑ ↓` sort direction, `✓` done, `−`/`+` deltas.
- **No emoji, ever.**

---

## Index

- `styles.css` — global entry; imports everything in `tokens/`.
- `tokens/` — `fonts.css`, `colors.css`, `typography.css`, `spacing.css`, `effects.css`, `base.css`.
- `guidelines/` — foundation specimen cards (Design System tab).
- `components/dev-loader.js` — dev fallback: builds the component namespace + runs JSX until the compiler emits `_ds_bundle.js`; inert afterwards.
- `components/forms/` — Button, IconButton, Input, Select, Checkbox, Radio, Switch.
- `components/display/` — Card, Badge, SeverityBadge, Tag, Stat, StatusDot, Table primitives.
- `components/feedback/` — Dialog, Toast, Tooltip, EmptyState.
- `components/navigation/` — TopNav, Tabs, Footer.
- `ui_kits/app/` — console screens: dashboard, inventory, findings (+detail), graph, reports.
- `ui_kits/website/` — marketing homepage.
- `ui_kits/docs/` — docs site page.
- `SKILL.md` — agent skill entry point.
- `Direction Explorations.dc.html` — the three original direction mockups (1c was chosen).

**Intentional additions** (no source defined components; standard set plus): `SeverityBadge`, `StatusDot`, `Stat`, `EmptyState`, `TopNav`, `Footer` — the ASM-specific primitives every screen needs.
