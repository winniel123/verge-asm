The workhorse data table: mono micro-label heads over a 2px ink rule, 9px row padding, soft row separators.

```jsx
<Table
  rowKey="id" onRowClick={open} selectedKey={sel}
  columns={[
    { key: "sev", label: "Severity", width: 96, render: r => <SeverityBadge severity={r.sev} /> },
    { key: "title", label: "Finding" },
    { key: "asset", label: "Asset", mono: true, muted: true, width: 220 },
    { key: "age", label: "Age", mono: true, muted: true, align: "right", width: 70 },
  ]}
  rows={rows}
/>
```

Technical columns get `mono` (+ usually `muted`). Put it in a `Card pad={false}`.
