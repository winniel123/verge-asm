import React from "react";

/* columns: [{ key, label, width, align, mono, muted, render(row) }] */
export function Table({ columns = [], rows = [], rowKey = "id", onRowClick, selectedKey, dense, style }) {
  const [hoverKey, setHoverKey] = React.useState(null);
  const padY = dense ? 6 : 9;
  return (
    <table style={{ width: "100%", borderCollapse: "collapse", background: "var(--surface)", ...style }}>
      <thead>
        <tr>
          {columns.map((c) => (
            <th key={c.key} style={{
              padding: `8px 16px`, textAlign: c.align || "left", width: c.width,
              font: "600 10px/1.2 var(--font-mono)", letterSpacing: "0.06em", textTransform: "uppercase",
              color: "var(--text-muted)", borderBottom: "2px solid var(--border-ink)", whiteSpace: "nowrap",
            }}>{c.label}</th>
          ))}
        </tr>
      </thead>
      <tbody>
        {rows.map((row, i) => {
          const k = row[rowKey] != null ? row[rowKey] : i;
          const selected = selectedKey != null && k === selectedKey;
          return (
            <tr key={k}
              onClick={onRowClick ? () => onRowClick(row) : undefined}
              onMouseEnter={() => setHoverKey(k)}
              onMouseLeave={() => setHoverKey(null)}
              style={{
                cursor: onRowClick ? "pointer" : "default",
                background: selected ? "var(--accent-soft)" : hoverKey === k && onRowClick ? "var(--surface-sunken)" : "transparent",
                boxShadow: selected ? "inset 2px 0 0 var(--accent)" : "none",
                transition: "background var(--duration) var(--ease)",
              }}>
              {columns.map((c) => (
                <td key={c.key} style={{
                  padding: `${padY}px 16px`, textAlign: c.align || "left",
                  borderTop: i === 0 ? "none" : "1px solid var(--border-soft)",
                  font: c.mono ? "400 12px/1.35 var(--font-mono)" : "400 13px/1.35 var(--font-sans)",
                  color: c.muted ? "var(--text-muted)" : "var(--text)",
                  whiteSpace: c.nowrap ? "nowrap" : "normal", verticalAlign: "middle",
                }}>{c.render ? c.render(row) : row[c.key]}</td>
              ))}
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}
