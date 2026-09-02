# Verge ASM design system — "Clear signal"

Redesign of the Verge ASM visual language, built from `uploads/design-system-redesign-brief.md` (2026-08-22). Verge ASM is a free, open-source (AGPL-3.0), self-hostable attack surface management app: it discovers internet-facing assets, watches for drift, and raises **signals** on a Critical / High / Medium / Low / Info scale. Three surfaces share this system: the console, the marketing site, and the docs site.

## Design rationale

**The idea: a calm instrument.** The old "engineered paper" datasheet becomes a modern security SaaS you can watch all day — soft warm-stone surfaces, one confident azure for action, and severity as the only loud voice in the room. Key moves:

- **Palette** — warm stone neutrals keep the paper warmth; a brighter azure (`#037ac0`) replaces the austere working blue; semantic tints give states room to breathe. Dark mode ships (§ Dark mode).
- **Type** — Instrument Sans (humanist grotesk, warm and contemporary) for UI + prose; Geist Mono for every technical value. The mono micro-label eyebrow survives as the system's signature.
- **Geometry** — a real radius scale (8 / 12 / 16 / 24, pills for chips) replaces 0px-everywhere.
- **Elevation** — layered soft shadows replace the hard 6px offset; floating layers get a gentle pop-in.
- **Severity** — Critical is the ramp's only solid fill, so it stays the loudest thing in a dense table even inside the calmer palette. All five chips clear WCAG AA in both modes.

## Sources

- `uploads/design-system-redesign-brief.md` — the redesign brief (product context, fixed rules, old-system snapshot). No codebase, Figma, font binaries, or logo files were provided.
- `uploads/design-system-component-gaps.md` — peer-feature gap audit + domain glossary (change vocabulary, seed/scope naming, the §12 "don’t build" list).

## Content fundamentals (fixed — from the brief, not restyled)

- Terse, precise, engineer-to-engineer, open-source-warm. Address the user as "you", never "we".
- **Sentence case everywhere** — headings, buttons, labels, nav. Never Title Case.
- Imperative verbs on actions: `Add seed`, `Run scan`, `Export CSV`. (**seed**, not target — the component-gaps glossary supersedes the original brief’s example, per the client’s call.)
- No exclamation marks, no emoji, no marketing-breathless copy.
- Terse relative timestamps (`4m`, `1h`, `3d`); absolute ISO 8601 on hover.
- Numbers use thousands separators; deltas signed with a true minus: `+12`, `−5`.
- Empty states = fact + next action: `No seeds yet. Add a domain or CIDR range to start scanning.`
- **Vocabulary** (glossary in `design-system-component-gaps.md`): **signal** never finding · **seed/scope** never target · **channel** never webhook/integration · **vantage** never probe/scanner/agent · **annotation** never mute/status/triage · signals leave by being **withdrawn** (the world moved), operators never "resolve" · `asset` only as a UI collective noun.
- Technical values are **always monospace**: hostnames, IPs, ports, CVE ids, hashes, versions, counts, timestamps.
- Open-source identity stays in the chrome: version string (`v0.9.2`), `AGPL-3.0`, GitHub link.

## Visual foundations

- **Color vibe.** Warm stone neutrals (page `#f9f7f5`, ink `#231f19`) — the old paper warmth, modernized. One action azure with hover/active/soft steps. Semantic ok/warn/danger each have text, solid, soft, and border tokens. Severity is its own five-level ramp (below). Backgrounds are flat color; no gradients, textures, or imagery in the console.
- **Type.** UI/prose: Instrument Sans, headings semibold with −0.015em tracking. Mono: Geist Mono for all technical values, KPI numerals (28px semibold), and the micro-label. Base UI size 13px / 1.5; docs prose 15–16px. Micro-label motif: 11px mono, medium, uppercase, 0.07em tracking, `--text-muted` (`OPEN SIGNALS`).
- **Spacing.** 4px grid. Controls 36px (30 compact, 44 marketing). Cards pad 20px, pages 32px. Tables stay dense: rows pad 10px/16px (8px in dense views). Console maxes at 1440px, prose at 760px.
- **Geometry & borders.** Radii: 8 tags/code → 12 controls → 16 cards/tables → 24 dialogs → pill chips. Hairlines still do quiet work (`--border-default`), but cards now lift with shadow instead of outlining with ink; the old 2px structural ink rules are gone.
- **Elevation.** Four soft layered shadows (`xs` flat controls → `sm` cards → `md` menus/popovers → `lg` dialogs). No hard offsets anywhere.
- **States.** Hover: sunken fill on rows/ghost controls; buttons darken one step (lighten in dark mode). Press: one further step, no shrinking. Selected row: `--row-selected` fill + 3px rounded accent bar on the left. Focus: rounded 2px accent ring offset by 2px of surface (`box-shadow: 0 0 0 2px var(--surface), 0 0 0 4px var(--focus-ring)`). Disabled: 45% opacity.
- **Motion.** Functional and quick: 120ms hovers, 180ms control states, 280ms floating layers with `vg-pop-in` (6px rise + 0.985 scale). Toasts slide up (`vg-toast-in`). The one ambient animation stays: the scan-running pulse (`vg-pulse`, 1.8s ring). Nothing bounces.
- **Transparency & blur.** Dialog scrim only: `rgba(21,18,15,0.4)` light / `rgba(0,0,0,0.55)` dark, no blur. Surfaces are always opaque.
- **Imagery.** Console uses none. Marketing composes real product UI (cards, chips, tables from this system) as its imagery instead of photos or illustration.
- **Cards.** `--surface` fill, 1px `--border-default`, `--r-lg`, `--shadow-sm`, 20px pad. Emphasis via the micro-label header row, not heavier borders.

## Severity ramp (accessible-contrast notes)

Critical is the only solid chip — the single loudest element in the system. High → Info are tinted pills with a solid dot, fading in intensity to stay ordinal. Chips are pill-shaped, 11px mono uppercase labels, always the exact words Critical / High / Medium / Low / Info.

Light mode (text : background):
- Critical — white on `#bf3631` — **5.54:1**
- High — `#a04400` on `#ffe9d6` — **5.38:1**
- Medium — `#8d5600` on `#ffeecc` — **5.30:1**
- Low — `#00728b` on `#d7f7ff` — **4.93:1**
- Info — `#536579` on `#ebf2f9` — **5.31:1**

Dark mode:
- Critical — white on `#c44039` — **5.07:1**
- High — `#eba57b` on `#352414` — **7.21:1**
- Medium — `#d7b16a` on `#322713` — **7.24:1**
- Low — `#62c8df` on `#192c2e` — **7.53:1**
- Info — `#a9b3bf` on `#23272c` — **≈7:1**

All ≥ 4.5:1 (WCAG AA for normal text). Low sits in cyan-teal (distinct from the action azure) so severity never reads as interactive. **Change is a separate language:** the drift palette (`--drift-gain-*` violet · `--drift-change-*` magenta · `--drift-loss-*` slate, AA-tuned, both modes) carries `appeared / revealed / withdrawn / descoped / returned / changed` as rounded-rect chips — severity stays the only pill and the only red. Row-level emphasis: Critical rows may carry a `--danger-soft` left edge tint in tables; never color whole rows.

## Dark mode

Ships. Warm graphite (not cool slate) keeps the brand temperature: page `#15120f`, surfaces `#1e1b17` / `#282520`. Primary buttons invert to bright-fill-with-ink-text (`#6bbeff` on `#063352`) for AA. Severity chips become translucent-feeling deep tints with bright text — measured ratios above. Toggle by setting `data-theme="dark"` on `<html>` or any subtree root.

**One group does not flip: the console.** `--surface-console` / `--text-on-console` / `--warn-on-console` / `--danger-on-console` carry the same value in `:root` and in `[data-theme="dark"]`. A log or code panel reads as a terminal, and a terminal is dark in every theme, so those surfaces cannot ride `--surface-inverted`. "Inverted" means the opposite of the page ground, which is right for a tooltip or a floating toast and wrong for a console. Log and code surfaces take the console group; `Tooltip` and `BulkActionsBar` stay on `--surface-inverted`. Measured against the `#231f19` console ground: `--text-on-console` 14.7:1, `--warn-on-console` 7.9:1, `--danger-on-console` 6.9:1.

## Iconography

**Lucide, confirmed** (the old system's substitution is now the choice): 1.75px stroke at 16px, `currentColor`, matching Instrument Sans's tone. Loaded from CDN (`https://unpkg.com/lucide@latest/dist/umd/lucide.min.js`) and rendered via the `Icon` component (`<i data-lucide>` under the hood) — no icon binaries are stored in this repo. Lucide carries no brand marks, so GitHub links are plain text links, not icons. No emoji, no unicode glyphs as icons. Common vocabulary: `radar` scans, `globe` domains, `server` IPs/services, `shield-alert` signals, `network` graph, `file-text` reports, `git-branch` drift.

**Logo: the pulse glyph** (chosen from the redesign’s options) — a solid signal dot inside two watch rings, drawn from the scan-pulse motif. Accent azure only, never severity colors; rings never animate in chrome; glyph works alone at ≥16px; `tile` variant for favicons. Rendered by the `Logo` component (`components/media/Logo.jsx`). Honest caveat: an in-system placeholder-quality mark, not professional brand work — replace when a real mark exists.

## Intentional additions

- `Icon` — thin Lucide wrapper so kits/consumers share one glyph API. Reason: the brief names Lucide as the (confirmable) icon set but had no icon component.
- `Logo` — the pulse-glyph brand mark + wordmark lockup (user-picked option 1b).
- `Pagination`, `Progress`, `Skeleton`, `Banner`, `Popover`, `DropdownMenu` — user-requested extensions beyond the brief’s §6 inventory (page controls, scan progress, loading states, persistent inline alerts, floating panels, row-action menus).
- `Drawer`, `Timeline`, `KeyValueList`, `DiffView` — signal-detail & drift batch (user-requested): slide-in detail panel, event history, definition grid, before/after drift block.
- `Sparkline`, `BarChart`, `ReportCard`, `DateRangePicker` — reports & data-viz batch (user-requested); chart series use `--chart-1..4`, never severity colors.
- `TagInput`, `Breadcrumb`, `CodeBlock`, `Kbd`, `Avatar` — input & chrome batch (user-requested): multi-value filters, micro-label trail, copyable code on inverted ink, key caps, initials chips (no photos).
- `BulkActionsBar`, `CommandPalette` + Table multi-select — inventory & bulk-ops batch (user-requested).
- `Graph` + `GraphLegend` — asset-graph batch (user-requested): typed nodes, severity halos, pan/zoom.
- `Callout`, `Accordion`, `Stepper`, `FileDrop` — docs & long-form batch (user-requested): prose asides, disclosure lists, install steps, drag-and-drop imports.
- `ChangeBadge`, `TransitionMarker` + Timeline batch-groups + drift tokens — drift surface (gaps-doc §1): the change vocabulary as its own visual language.
- `SavedViews`, `ColumnPicker`, `CoverageMessageList`, `SignalRuleRef` — the final gaps-doc P2s: named filter sets and column visibility on Inventory, coverage facts on Scope, rule provenance in the signal drawer.
- `ConfirmDialog` + a11y hardening — destructive-act confirmation (typed confirm for the worst), focus traps in Dialog/Drawer/CommandPalette with focus restore, polite live region on the toast stack.
- `CodeInput` — TOTP/MFA verify field (segmented digits, auto-advance, paste-distributes; pairs with SecretInput’s write-only enrollment).
- `ContextMenu`, `HoverCard`, `JSONViewer`, `MiddleTruncate`, `RadioCards`, `DeltaChip` — tier E refinements, plus Table keyboard nav (roving j/k focus, Enter opens, `onRowContextMenu`) and a generated component reference at `ui_kits/docs/reference.html` (built from each component’s `.prompt.md` + `.d.ts`).
- `Textarea`, `FormField` + `FormErrorSummary`, `SegmentedControl`, `HeatmapCalendar`, `InlineEdit` — tier D, with Table `density` + opt-in `virtual` windowing and a system-wide reduced-motion pass.
- `TimeSeriesChart`, `Wizard`, `VersionSelect` \u2014 tier C: an axed multi-series line chart for Reports (Sparkline stays the inline form), a multi-step setup dialog (constructive counterpart to ConfirmDialog \u2014 wired as Reports\u2019 New schedule), and the docs version picker.
- `RelativeTime`, `CopyValue`, `TreeView`, `Combobox`, `NumberInput`, `Slider`, `ToastStack` — quality-of-life set: the canonical timestamp pattern, single-value copy, name-hierarchy tree, typeahead select, numeric inputs, stacked toast queue.
- `OrgSwitcher`, `SettingsNav`, `SplitButton` + Graph minimap — shell batch (gaps-doc §11): MSP org scope, sectioned settings nav, export split button; global typeahead search is served by CommandPalette (⌘K).
- `SecretInput`, `ChannelForm`, `MessageList`, `LogViewer`, `CadenceSelect`, `BatchStatus`, `VantageCard`, `AvailabilityBadge`, `ExposureBadge` — ops, channels & vantages (gaps-doc §6/§7/§9): write-only secrets, class-routed channels, message inbox, streaming batch logs, reach as STATE chips (never a score). DeliveryTable composes from Table.
- `ProposalReview`, `ExclusionEditor`, `CustodyToggle`, `RefusalCallout` + TagInput per-token validation — seeds & scope onboarding (gaps-doc §5): confirm-one/decline-many, three exclusion kinds, read-only custody census, refusals that name the reachable set.
- Table sort + sticky header, `Spinner`, `ErrorState` — table power & universal states (gaps-doc §2/§11); virtualization noted as a production concern.
- `CoverageMeter`, `GapBadge`, `StalenessBadge` — coverage & gaps (gaps-doc §4): denominator/census reads, dotted cell-level absence, bronze `--stale-*` currency states — none reuse severity or drift color.
- `AnnotationControl`, `WithdrawnMark` — signals model corrections (gaps-doc §3/§12): the one operator dial (accepted risk + reason, no state/expiry/author) replacing mute/resolve; dashed mark for keys in no current population.

## Production notes

Mock-only shortcuts a production build should replace:

- **Table virtualization** — now ships as opt-in `virtual` + fixed `rowHeight` windowing (#257); variable-height rows still need a measured virtualizer.
- **Reduced motion** — honored globally: duration tokens collapse to 1ms and loops run once under `prefers-reduced-motion: reduce`.
- **Row virtualization** (#257) — Table's `maxHeight` scroll is the mock; virtualize rows past ~200.
- **Form validation** — per-field errors exist; add form-level summaries for long forms.
- **ToastStack timers** — restart-free ttls are handled; pause-on-hover is not.

## Index

- `styles.css` — global entry; imports everything under `tokens/`.
- `tokens/` — `colors.css`, `typography.css`, `spacing.css`, `radius.css`, `elevation.css`, `motion.css`, `base.css`.
- `guidelines/` — foundation specimen cards (colors, type, spacing, geometry, elevation, states, motion, severity).
- `components/forms/` — Button, IconButton, Input, Select, Checkbox, Radio, Switch, DateRangePicker, TagInput, FileDrop, ProposalReview, ExclusionEditor, CustodyToggle, SecretInput, CadenceSelect, ChannelForm, SplitButton, Combobox, NumberInput, Slider.
- `components/display/` — Card, Badge, SeverityBadge, Tag, Stat, StatusDot, Table, Pagination, Progress, Skeleton, Timeline, KeyValueList, DiffView, Sparkline, BarChart, TimeSeriesChart, ReportCard, DeltaChip, JSONViewer, MiddleTruncate, CodeBlock, Kbd, Avatar, Graph, Accordion, Stepper, ChangeBadge, TransitionMarker, WithdrawnMark, CoverageMeter, GapBadge, StalenessBadge, Spinner, AvailabilityBadge, ExposureBadge, BatchStatus, LogViewer, VantageCard, RelativeTime, CopyValue, TreeView.
- `components/feedback/` — Dialog, Toast, Tooltip, EmptyState, Banner, Popover, DropdownMenu, Drawer, BulkActionsBar, CommandPalette, Callout, AnnotationControl, ErrorState, RefusalCallout, MessageList, ToastStack, ConfirmDialog, SavedViews, ColumnPicker, CoverageMessageList, SignalRuleRef.
- `components/navigation/` — TopNav, Tabs, Footer, Breadcrumb, OrgSwitcher, SettingsNav.
- `components/media/` — Icon (Lucide wrapper), Logo (pulse glyph + wordmark).
- `ui_kits/console/` — eight screens end-to-end: dashboard, scope, inventory, drift, signals, graph, reports, settings (light/dark toggle, ⌘K palette). Every component in the library is surfaced across the kits.
- `ui_kits/marketing/` — homepage.
- `ui_kits/docs/` — docs page.
- `assets/` — `ds-fallback-loader.js` (renders kits/cards even before the component bundle compiles). No logo or image assets exist (none provided).
- `new-components.html` — visual gallery of every post-brief component, isolated.
- `SKILL.md` — agent-facing usage guide.
