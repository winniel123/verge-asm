import React from "react";

export function Stat({ label, value, delta, deltaTone = "neutral", tone, style }) {
  const deltaColor = { ok: "var(--ok)", danger: "var(--danger)", neutral: "var(--text-muted)" }[deltaTone];
  return (
    <div style={{ padding: "16px 20px", ...style }}>
      <div style={{ font: "600 10px/1 var(--font-mono)", letterSpacing: "0.06em", textTransform: "uppercase", color: "var(--text-muted)", marginBottom: 8 }}>{label}</div>
      <div style={{ font: "600 26px/1.15 var(--font-mono)", color: tone === "danger" ? "var(--danger)" : "var(--ink)" }}>{value}</div>
      {delta != null && <div style={{ font: "500 11px/1.2 var(--font-mono)", color: deltaColor, marginTop: 4 }}>{delta}</div>}
    </div>
  );
}
