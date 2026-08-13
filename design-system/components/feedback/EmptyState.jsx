import React from "react";

export function EmptyState({ title, detail, action, style }) {
  return (
    <div style={{ border: "1px dashed var(--border)", background: "var(--surface)", padding: "40px 24px", display: "flex", flexDirection: "column", alignItems: "center", textAlign: "center", gap: 6, ...style }}>
      <div style={{ font: "600 14px/1.35 var(--font-sans)", color: "var(--ink)" }}>{title}</div>
      {detail && <div style={{ font: "400 12px/1.5 var(--font-sans)", color: "var(--text-muted)", maxWidth: 380 }}>{detail}</div>}
      {action && <div style={{ marginTop: 12 }}>{action}</div>}
    </div>
  );
}
