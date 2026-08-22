import React from "react";

export function Tag({ children, onRemove, style }) {
  const [hov, setHov] = React.useState(false);
  return (
    <span style={{ display: "inline-flex", alignItems: "center", gap: 6, height: 22, padding: "0 8px", borderRadius: 8, background: "var(--surface-sunken)", border: "1px solid var(--border-default)", color: "var(--text-secondary)", fontFamily: "var(--font-mono)", fontSize: 11.5, whiteSpace: "nowrap", ...style }}>
      {children}
      {onRemove && (
        <button type="button" aria-label="Remove" onClick={onRemove} onMouseEnter={() => setHov(true)} onMouseLeave={() => setHov(false)}
          style={{ display: "inline-flex", alignItems: "center", justifyContent: "center", width: 14, height: 14, border: "none", borderRadius: 999, background: hov ? "var(--border-strong)" : "transparent", color: "var(--text-muted)", cursor: "pointer", padding: 0 }}>
          <svg viewBox="0 0 10 10" width="8" height="8"><path d="M2 2l6 6M8 2l-6 6" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round"></path></svg>
        </button>
      )}
    </span>
  );
}
