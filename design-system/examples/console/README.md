# Console UI kit

Interactive mock of the redesigned Verge ASM console. `ConsoleApp.jsx` is the shell. Beside it sit 24 screen and section components, plus `SignalData.jsx`, the signal fixtures Dashboard, Signals, Reports and ReportArtifact share.

The shell routes 19 screens: dashboard (or the first-run checklist), scope, inventory, drift, signals, exposure, coverage, graph, reports, settings, asset, service, endpoint, run, artifact, inbox, search, profile, error. Nine carry a TopNav item; the rest open from a row, the command palette (⌘K), or `window.__vergeGo`. `Integrations.jsx` and `Sources.jsx` render as Settings sections, and `Onboarding.jsx` as a dialog the palette opens. `SignIn.jsx` and `Setup.jsx` stand alone — nothing in this directory mounts them (#2076).

Screens compose `components/` primitives, but not only those: every page frame, header and grid here is hand-written markup styled against `var(--…)` tokens. `Coverage.jsx` goes further and writes the aperture-statement ledger as a raw `<table>`, because `Table.jsx` cells are `nowrap`, ellipsis-clipped and middle-aligned, and a ledger cell wraps and top-aligns.

- Navigate via TopNav; Signals → row click opens the signal detail Drawer; search + severity filter drive the EmptyState.
- Run scan → pulsing nav status + toasts. Add seed → dialog.
- Theme toggle in the nav flips `data-theme="dark"`.
- Reports: trend ReportCards (Sparkline/BarChart), DateRangePicker, scheduled-report table.
- Drift: batch-grouped transition timeline on the drift palette, change-kind filters, inline DiffView.
- Graph: draggable asset graph with severity halos; node click opens the detail Drawer.
- Inventory: subjects grouped by kind, expandable facet rows, search + kind SegmentedControl, gap and proxy Switches, ColumnPicker.
