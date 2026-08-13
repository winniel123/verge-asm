import React from "react";

export function Tag({ active, onRemove, onClick, style, children }) {
  const [hover, setHover] = React.useState(false);
  return (
    <span
      onClick={onClick}
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => setHover(false)}
      style={{
        display: "inline-flex", alignItems: "center", gap: 6, padding: "4px 8px",
        font: "500 12px/1.2 var(--font-sans)", whiteSpace: "nowrap",
        border: `1px solid ${active ? "var(--border-ink)" : "var(--border)"}`,
        background: active ? "var(--ink)" : hover && (onClick || onRemove) ? "var(--surface-sunken)" : "var(--surface)",
        color: active ? "var(--text-on-ink)" : "var(--text)",
        cursor: onClick ? "pointer" : "default",
        transition: "background var(--duration) var(--ease)",
        ...style,
      }}
    >
      {children}
      {onRemove && (
        <span role="button" aria-label="Remove" onClick={(e) => { e.stopPropagation(); onRemove(); }}
          style={{ cursor: "pointer", font: "400 11px var(--font-mono)", color: active ? "var(--text-on-ink)" : "var(--text-muted)", lineHeight: 1 }}>✕</span>
      )}
    </span>
  );
}
