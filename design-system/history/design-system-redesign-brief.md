# Redesign brief — Verge ASM design system

**For:** a design pass (hand this whole file to the designer).
**What we want:** replace the current visual language. Keep what the product *says* and *means*; change how it *looks and feels*.
**Date:** 2026-08-22.

---

## 1. The product in one paragraph

Verge ASM is a free, open-source (AGPL-3.0), self-hostable **attack surface management** web app. It continuously discovers an organization's internet-facing assets — domains, subdomains, IPs, ports, services, TLS certs — watches them for **drift**, and raises **signals** on a Critical / High / Medium / Low / Info scale. Audience: security engineers, red/blue teams, IT admins, self-hosters, and MSPs running many client orgs. There are **three surfaces**: the web app (console), a marketing/landing site, and a docs site. All three share the design system.

---

## 2. The direction we're moving toward

**Clean, friendly, modern SaaS.** The current system is a deliberately austere "engineered paper" datasheet — flat, sharp-cornered, near-monochrome, unflashy to a fault. We want to move *away* from the raw-instrument feel toward a polished, approachable product without losing the credibility a security tool needs.

Concretely, the redesign should:

- **Round the geometry.** Today radius is `0px` everywhere. Introduce a real radius scale (cards, inputs, buttons, chips).
- **Add real depth.** Replace the hard offset shadow (`6px 6px 0`) with a soft, modern elevation system.
- **Open up the spacing.** More whitespace and breathing room. It can stay information-dense where data density matters (tables, the graph), but it should not feel cramped by default.
- **Warm up and broaden the palette.** More color, used with intent — a friendlier primary, supportive neutrals, tasteful use of tints for surfaces and states. Move past "one blue and otherwise grayscale."
- **Friendlier type.** The current UI font is the Helvetica system stack. Choose a warmer, more contemporary typeface pairing (a humanist/grotesk sans for UI + prose). Keep a monospace for technical values (see §3).
- **Soften "instrument" severity.** Severity color is currently the single loudest thing on every screen. Keep severity *meaningful and unmistakable*, but let it sit inside a calmer, more balanced palette rather than dominating.

**North star feeling:** a modern security SaaS a team is happy to look at all day — clear, calm, confident, trustworthy. Not a brochure, not a toy, not a terminal.

---

## 3. What is FIXED (do not change these)

These are load-bearing for the product and are **out of scope** for the redesign. The visuals get reskinned *around* them.

1. **Content voice & copy rules.** Terse, precise, engineer-to-engineer, open-source-warm.
   - Sentence case everywhere (headings, buttons, labels, nav) — never Title Case.
   - Imperative verbs on actions: `Add target`, `Run scan`, `Export CSV`.
   - No exclamation marks, no emoji, no marketing-breathless copy. Address the user as "you", never "we".
   - Terse relative timestamps (`4m`, `1h`, `3d`) with absolute ISO 8601 on hover.
   - Numbers use thousands separators; deltas are signed with a true minus (`+12`, `−5`).
   - Empty states = fact + next action: `No targets yet. Add a domain or CIDR range to start scanning.`

2. **Technical values are always monospace.** Hostnames, IPs, ports, CVE ids, hashes, versions, counts, timestamps render in a mono face (`edge-gw-03.acmecorp.io`, `CVE-2026-1187`, `:5900`). You may *choose which* mono typeface; you may not drop the convention.

3. **Severity semantics.** The scale is exactly **Critical, High, Medium, Low, Info** — five levels, these exact words, never synonymized, always ordered this way. The color ramp must remain instantly legible and ordinal (most severe reads as most urgent). You may restyle the ramp's exact hues and the badge form; you may not merge levels, rename them, or make two adjacent levels hard to tell apart. Accessible contrast on every severity chip is required.

4. **Three surfaces, one system.** Console + marketing + docs all draw from the same tokens and components.

5. **Open-source identity stays visible in the chrome.** Version string (`v0.9.2`), `AGPL-3.0`, and a GitHub link belong in the product frame.

---

## 4. What is OPEN (redesign freely)

Everything visual. Specifically:

- **Color** — the entire palette: background(s), surfaces, primary/accent, neutrals, semantic status (ok/warn/danger), and the severity ramp's exact hues. Consider whether a **dark mode** should ship (nice-to-have, not required).
- **Typography** — UI/prose typeface, the mono typeface, the full size scale, weights, line heights, and any signature type motif.
- **Geometry** — radius scale, border weights and roles, whether hairlines still do the layout work.
- **Elevation** — the shadow system, how floating layers (dialogs, menus, tooltips) read.
- **Spacing & density** — the grid unit, default paddings, control heights, content max-widths.
- **States** — hover / press / selected / focus / disabled treatments.
- **Motion** — easing, durations, and whether anything is allowed to be more expressive than "functional only."
- **Iconography** — the current system substitutes Lucide (flagged as a substitution); the designer may confirm or replace it.
- **Imagery / marketing** — the current app uses zero imagery and marketing leans on typographic compositions and terminal blocks. Open to a friendlier approach on marketing/docs.

---

## 5. Current system snapshot (what you're replacing)

So the designer knows the starting point precisely. **This is the *old* look — the thing to move on from — not a spec to match.**

**Design idea (old):** "Engineered paper." Warm paper background, near-black ink, flat surfaces, 1px hairlines, sharp corners, one working blue, severity color as the loudest thing on screen.

**Color (old):**
- Paper `#f7f7f4` page · white `#ffffff` surfaces · sunken `#f2f2ee` · inverted ink `#16160f`.
- Ink `#16160f` · body `#1a1a16` · muted `#6b6b62` · faint `#9a9a90`.
- Borders: default hairline `#d8d8d0`, row separator `#ecece4`, structural ink `#16160f`.
- Single accent: working blue `#2d4fd4` (hover `#2440ad`, soft `#e9edfb`).
- Semantic: ok `#2f9e44` · warn `#b08800` · danger `#c92a2a` (each with a `-soft` tint).
- Severity ramp: Critical `#c92a2a` → High `#e8590c` → Medium `#ffd43b` (ink text) → Low `#1971c2` → Info `#868e96`.

**Type (old):**
- UI/prose: `"Helvetica Neue", Helvetica, Arial, sans-serif`. Technical: `"IBM Plex Mono", …`.
- Base 13px / 1.55. KPI numerals 26px mono semibold. Size scale ran 10 → 48px.
- Signature motif: the **micro-label** — 10px mono semibold uppercase, 0.06em tracking, muted (`OPEN FINDINGS`). Used for table heads, KPI captions, section eyebrows. Large sans headings tracked −0.01em.

**Space & geometry (old):** 4px grid. Table rows pad 9px/16px. Controls 32px tall (26px compact, 40px marketing). Cards pad 16px, pages 24px. App content maxes at 1440px, prose at 720px. **Radius `0px` everywhere** — only status dots are round.

**Elevation & rules (old):** No soft shadows anywhere. Floating layers used a **hard offset shadow** `6px 6px 0 rgba(22,22,15,.1)` + 1px ink border. 2px ink rule under the app header and under table header rows; 1px ink outlines on emphasized containers. Ordinary cards: white + 1px `#d8d8d0`.

**States (old):** Hover → sunken `#f2f2ee`; links underline; primary button lightens to `#3a3a30`. Selected row → `#e9edfb` fill + 2px accent inset bar on the left edge. Focus → square ring, 2px paper + 2px accent. Disabled → 45% opacity.

**Motion (old):** Functional only — 120ms background/opacity, 240ms dialog fade. One ambient animation: a 1.6s opacity blink on live "scan running" status dots. Nothing bounces or springs.

---

## 6. Components & screens the system has to dress

The redesign must cover these (existing inventory — keep the set, restyle it):

- **Forms:** Button, IconButton, Input, Select, Checkbox, Radio, Switch.
- **Display:** Card, Badge, **SeverityBadge**, Tag, **Stat** (KPI), **StatusDot**, Table primitives.
- **Feedback:** Dialog, Toast, Tooltip, EmptyState.
- **Navigation:** TopNav, Tabs, Footer.
- **Console screens:** dashboard, inventory, signals (+ detail), asset graph, reports.
- **Marketing:** homepage. **Docs:** a docs page.

The ASM-specific primitives worth special attention: **SeverityBadge**, **StatusDot** (including the live "scan running" pulse), **Stat** (the KPI number blocks), and **EmptyState**.

---

## 7. Deliverables we'd like back

1. A short **design rationale** — the new idea in a sentence or two, and the reasoning for the key moves (palette, type, radius, elevation).
2. A **token set** — color, typography, spacing, radius, elevation, motion — as named values (drop-in for a `tokens/`-style CSS layer). Preserve semantic roles where possible so existing component code can re-point to new values.
3. A **severity ramp** proposal with accessible-contrast notes for each of the five levels.
4. **Restyled specimens** of the components in §6 (at minimum: Button set, Input/Select, Card, SeverityBadge, Stat, Table, Dialog, TopNav, EmptyState).
5. At least one **console screen** mocked end-to-end (dashboard or the signals list) so the system is shown working under real density.
6. Light **and** dark, *if* you propose dark mode — otherwise light only, but say so.

---

## 8. Traps & watch-outs (project-specific)

- **The word is "signal," not "finding."** The old system and its readme say "findings" and "fingerprints." The product's canonical vocabulary uses **signal**, rejects `finding`/`host`/`fingerprint`, and allows `asset` only as a UI collective noun. **Keep the domain words out of any new copy you write** — when a visual convention and a domain term collide, the domain term wins. Restyle around it.
- **Severity must survive the "friendlier" pass.** The single biggest risk in a clean-SaaS move is muting severity into prettiness. Critical has to still read as *critical* at a glance in a dense table. Test the ramp in context, not just as swatches.
- **Don't lose density where it earns its keep.** Tables and the asset graph carry a lot of data. "Generous whitespace" is the default, not a mandate to halve the rows-per-screen on data views.
- **Contrast & accessibility are non-negotiable** for a tool people scan for real problems. WCAG AA minimum on text and on every status/severity chip.
- **It's self-hosted OSS, not a vendor product.** Friendly, yes; salesy, no. The chrome should still feel like software you own and can read the source of.

---

## 9. One-line summary to lead with

> Take Verge ASM from an austere flat "engineered paper" datasheet to a clean, friendly, modern security SaaS — rounder, softer, roomier, more colorful — while keeping its terse engineer voice, its always-mono technical values, and an unmistakable five-level severity scale.
