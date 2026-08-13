import React from "react";
import { Wordmark } from "../display/Wordmark.jsx";

export function TopNav({ items = [], onSelect, status, right, style }) {
  const [hover, setHover] = React.useState(null);
  return (
    <div style={{ display: "flex", alignItems: "center", gap: 28, padding: "0 24px", height: 52, background: "var(--surface)", borderBottom: "2px solid var(--border-ink)", boxSizing: "border-box", ...style }}>
      <Wordmark size="md" />
      <nav style={{ display: "flex", gap: 2, alignSelf: "stretch", alignItems: "stretch" }}>
        {items.map((it, i) => (
          <a key={it.label} href={it.href || "#"}
            onClick={(e) => { if (onSelect) { e.preventDefault(); onSelect(it, i); } }}
            onMouseEnter={() => setHover(i)} onMouseLeave={() => setHover(null)}
            style={{
              display: "inline-flex", alignItems: "center", padding: "0 12px",
              font: "500 13px var(--font-sans)", textDecoration: "none",
              color: it.active ? "var(--ink)" : hover === i ? "var(--ink)" : "var(--text-muted)",
              boxShadow: it.active ? "inset 0 -2px 0 var(--accent)" : "none",
              transition: "color var(--duration) var(--ease)",
            }}>{it.label}</a>
        ))}
      </nav>
      <div style={{ marginLeft: "auto", display: "flex", alignItems: "center", gap: 16 }}>
        {status}{right}
      </div>
    </div>
  );
}
