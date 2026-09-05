import React from "react";

const S = {
  scheduled: { bg: "var(--surface-sunken)", bd: "var(--border-default)", fg: "var(--text-secondary)", dot: "var(--neutral-400)" },
  running: { bg: "var(--accent-soft)", bd: "transparent", fg: "var(--link)", dot: "var(--accent)", pulse: true },
  complete: { bg: "var(--ok-soft)", bd: "var(--ok-border)", fg: "var(--ok)", dot: "var(--ok-solid)" },
  failed: { bg: "var(--danger-soft)", bd: "var(--danger-border)", fg: "var(--danger)", dot: "var(--danger-solid)" },
};
export function BatchStatus({ status = "scheduled", scope, size = "md", style }) {
  const t = S[status] || S.scheduled;
  const sm = size === "sm";
  return (
    <span style={{ display: "inline-flex", alignItems: "center", gap: sm ? 4 : 6, height: sm ? 18 : 20, padding: sm ? "0 6px" : "0 8px", borderRadius: 8, background: t.bg, border: "1px solid " + t.bd, color: t.fg, fontFamily: "var(--font-mono)", fontSize: sm ? 10 : 10.5, fontWeight: 600, letterSpacing: "0.04em", whiteSpace: "nowrap", lineHeight: 1, ...style }}>
      <span style={{ width: 5, height: 5, borderRadius: 999, background: t.dot, animation: t.pulse ? "vg-pulse 1.8s var(--ease-out) infinite" : "none" }} />
      {status}
      {scope && <span style={{ fontWeight: 400, opacity: 0.8 }}>· {scope}</span>}
    </span>
  );
}
