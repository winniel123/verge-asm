import React from "react";

export function KeyValueList({ items = [], columns = 2, sunken = true, style }) {
  return (
    <div style={{ display: "grid", gridTemplateColumns: "repeat(" + columns + ", minmax(0, 1fr))", gap: "14px 20px", padding: sunken ? 16 : 0, background: sunken ? "var(--surface-sunken)" : "transparent", borderRadius: sunken ? 12 : 0, fontFamily: "var(--font-ui)", ...style }}>
      {items.map((it) => (
        <div key={it.k} style={{ display: "flex", flexDirection: "column", gap: 3, gridColumn: it.span ? "span " + it.span : "auto", minWidth: 0 }}>
          <span style={{ font: "500 11px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--text-muted)" }}>{it.k}</span>
          <span style={{ font: (it.mono === false ? "400 12.5px/1.5 var(--font-ui)" : "400 12.5px var(--font-mono)"), color: "var(--text-body)", overflowWrap: "anywhere" }}>{it.v}</span>
        </div>
      ))}
    </div>
  );
}
