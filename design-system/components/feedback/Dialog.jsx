import React from "react";

export function Dialog({ open, title, children, footer, onClose, width = 440 }) {
  if (!open) return null;
  return (
    <div onClick={onClose} style={{ position: "fixed", inset: 0, background: "rgba(22,22,15,0.32)", display: "flex", alignItems: "center", justifyContent: "center", zIndex: 100, padding: 24 }}>
      <div role="dialog" aria-modal="true" onClick={(e) => e.stopPropagation()}
        style={{ width, maxWidth: "100%", background: "var(--surface)", border: "1px solid var(--border-ink)", boxShadow: "var(--shadow-hard)", display: "flex", flexDirection: "column", maxHeight: "85vh" }}>
        <div style={{ display: "flex", alignItems: "center", gap: 12, padding: "14px 16px", borderBottom: "1px solid var(--border)" }}>
          <div style={{ font: "600 16px/1.3 var(--font-sans)", color: "var(--ink)" }}>{title}</div>
          {onClose && (
            <button onClick={onClose} aria-label="Close" style={{ marginLeft: "auto", width: 26, height: 26, border: "1px solid transparent", background: "transparent", cursor: "pointer", font: "400 13px var(--font-mono)", color: "var(--text-muted)" }}>✕</button>
          )}
        </div>
        <div style={{ padding: 16, overflow: "auto", font: "400 13px/1.55 var(--font-sans)", color: "var(--text)" }}>{children}</div>
        {footer && <div style={{ display: "flex", justifyContent: "flex-end", gap: 8, padding: "12px 16px", borderTop: "1px solid var(--border)" }}>{footer}</div>}
      </div>
    </div>
  );
}
