import React from "react";

const TONES = { ok: "var(--ok)", warn: "var(--warn)", danger: "var(--danger)", neutral: "var(--text-faint)", accent: "var(--accent)" };

export function Toast({ tone = "neutral", title, detail, onDismiss, style }) {
  return (
    <div style={{ display: "flex", alignItems: "flex-start", gap: 10, width: 360, maxWidth: "100%", background: "var(--surface)", border: "1px solid var(--border-ink)", boxShadow: "var(--shadow-hard-sm)", padding: "11px 12px", boxSizing: "border-box", ...style }}>
      <span style={{ width: 8, height: 8, borderRadius: "50%", background: TONES[tone], flex: "none", marginTop: 5 }} />
      <div style={{ minWidth: 0, flex: 1 }}>
        <div style={{ font: "600 13px/1.35 var(--font-sans)", color: "var(--ink)" }}>{title}</div>
        {detail && <div style={{ font: "400 12px/1.45 var(--font-sans)", color: "var(--text-muted)", marginTop: 2 }}>{detail}</div>}
      </div>
      {onDismiss && (
        <button onClick={onDismiss} aria-label="Dismiss" style={{ border: "none", background: "transparent", cursor: "pointer", font: "400 12px var(--font-mono)", color: "var(--text-muted)", padding: 2, flex: "none" }}>✕</button>
      )}
    </div>
  );
}
