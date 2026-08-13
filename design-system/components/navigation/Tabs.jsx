import React from "react";

export function Tabs({ items = [], value, onChange, style }) {
  const [hover, setHover] = React.useState(null);
  return (
    <div style={{ display: "flex", gap: 20, borderBottom: "1px solid var(--border)", ...style }}>
      {items.map((label, i) => {
        const active = value != null ? value === label : i === 0;
        return (
          <button key={label} onClick={() => onChange && onChange(label, i)}
            onMouseEnter={() => setHover(i)} onMouseLeave={() => setHover(null)}
            style={{
              appearance: "none", background: "transparent", border: "none", cursor: "pointer",
              padding: "8px 2px 9px", marginBottom: -1,
              font: `500 13px var(--font-sans)`,
              color: active ? "var(--ink)" : hover === i ? "var(--ink)" : "var(--text-muted)",
              borderBottom: active ? "2px solid var(--accent)" : "2px solid transparent",
              transition: "color var(--duration) var(--ease)",
            }}>{label}</button>
        );
      })}
    </div>
  );
}
