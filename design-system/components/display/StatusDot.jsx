import React from "react";

const TONES = { ok: "var(--ok)", warn: "var(--warn)", danger: "var(--danger)", accent: "var(--accent)", neutral: "var(--text-faint)" };

export function StatusDot({ tone = "ok", pulse, label, size = 8, style }) {
  const c = TONES[tone] || TONES.neutral;
  return (
    <span style={{ display: "inline-flex", alignItems: "center", gap: 7, ...style }}>
      <span style={{ width: size, height: size, flex: "none", borderRadius: "50%", background: c, animation: pulse ? "verge-pulse 1.6s infinite" : "none" }} />
      {label && <span style={{ font: "500 11px/1.2 var(--font-mono)", color: c === "var(--text-faint)" ? "var(--text-muted)" : c }}>{label}</span>}
    </span>
  );
}
