# Console UI kit

Interactive mock of the redesigned Verge ASM console — eight screens end-to-end (dashboard, scope, inventory, drift, signals, graph, reports, settings), composed from `components/` primitives only.

- Navigate via TopNav; Signals → row click opens the signal detail dialog; search + severity filter drive the EmptyState.
- Run scan → pulsing status + toasts. Add target → dialog.
- Theme toggle in the nav flips `data-theme="dark"`.
- Reports: trend ReportCards (Sparkline/BarChart), DateRangePicker, scheduled-report table.
- Drift: batch-grouped transition timeline on the drift palette, change-kind filters, inline diffs.
- Graph: pannable/zoomable asset graph with severity halos; node click opens the detail Drawer.
- Inventory: TagInput filters, multi-select rows, floating bulk-actions bar.
