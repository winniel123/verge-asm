import React from "react";

export function Card({ title, eyebrow, action, emphasized, pad = true, style, bodyStyle, children }) {
  return (
    <div style={{ background: "var(--surface)", border: `1px solid ${emphasized ? "var(--border-ink)" : "var(--border)"}`, display: "flex", flexDirection: "column", ...style }}>
      {(title || action || eyebrow) && (
        <div style={{ display: "flex", alignItems: "center", gap: 12, padding: "12px 16px", borderBottom: "1px solid var(--border-soft)" }}>
          <div style={{ minWidth: 0 }}>
            {eyebrow && <div style={{ font: "600 10px/1 var(--font-mono)", letterSpacing: "0.06em", textTransform: "uppercase", color: "var(--text-muted)", marginBottom: title ? 5 : 0 }}>{eyebrow}</div>}
            {title && <div style={{ font: "600 14px/1.35 var(--font-sans)", color: "var(--ink)" }}>{title}</div>}
          </div>
          {action && <div style={{ marginLeft: "auto", display: "flex", gap: 8, alignItems: "center", flex: "none" }}>{action}</div>}
        </div>
      )}
      <div style={{ padding: pad ? 16 : 0, flex: 1, ...bodyStyle }}>{children}</div>
    </div>
  );
}
