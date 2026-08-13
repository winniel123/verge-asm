# Verge ASM — app UI kit

Interactive recreation of the Verge ASM console: Dashboard, Inventory, Findings (list + detail), Graph, and Reports, composed from the system components (`window` bundle namespace — see `index.html`).

- `index.html` — interactive shell: section nav, New scan dialog, toasts, footer.
- `data.js` — shared fake dataset (`window.VergeData`).
- `Dashboard.jsx`, `Inventory.jsx`, `Findings.jsx`, `GraphView.jsx`, `Reports.jsx` — one screen per file, registered on `window.VergeApp`.

No original product exists (from-scratch system) — these screens define the reference look: paper page, stat strips with hairline dividers, dense tables in flush Cards, ink terminal blocks for evidence, severity as the loudest color.
