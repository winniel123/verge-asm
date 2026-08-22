import React from "react";

const COLORS = { ok: "var(--ok-solid)", warn: "var(--warn-solid)", danger: "var(--danger-solid)", neutral: "var(--neutral-400)", running: "var(--accent)" };

export function StatusDot({ status = "neutral", label, micro, style }) {
  const pulse = status === "running";
  return (
    <span style={{ display: "inline-flex", alignItems: "center", gap: 8, ...style }}>
      <span style={{ width: 8, height: 8, borderRadius: 999, flex: "none", background: COLORS[status] || COLORS.neutral, animation: pulse ? "vg-pulse 1.8s var(--ease-out) infinite" : "none" }} />
      {label && (micro
        ? <span style={{ font: "500 11px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: pulse ? "var(--accent)" : "var(--text-muted)" }}>{label}</span>
        : <span style={{ font: "400 12.5px var(--font-ui)", color: "var(--text-secondary)" }}>{label}</span>)}
    </span>
  );
}
