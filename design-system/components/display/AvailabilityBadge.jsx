import React from "react";

/* Vantage availability (Derived) — operational status, not severity. */
const S = {
  available: { bg: "var(--ok-soft)", bd: "var(--ok-border)", fg: "var(--ok)", dot: "var(--ok-solid)" },
  degraded: { bg: "var(--warn-soft)", bd: "var(--warn-border)", fg: "var(--warn)", dot: "var(--warn-solid)" },
  unavailable: { bg: "var(--danger-soft)", bd: "var(--danger-border)", fg: "var(--danger)", dot: "var(--danger-solid)" },
  unverified: { bg: "transparent", bd: "var(--border-strong)", fg: "var(--text-secondary)", dot: "var(--neutral-400)", dashed: true },
};
export function AvailabilityBadge({ state = "available", size = "md", style }) {
  const t = S[state] || S.unverified;
  const sm = size === "sm";
  return (
    <span style={{ display: "inline-flex", alignItems: "center", gap: sm ? 4 : 5, height: sm ? 18 : 20, padding: sm ? "0 6px" : "0 8px", borderRadius: 8, background: t.bg, border: "1px " + (t.dashed ? "dashed" : "solid") + " " + t.bd, color: t.fg, fontFamily: "var(--font-mono)", fontSize: sm ? 10 : 10.5, fontWeight: 600, letterSpacing: "0.04em", whiteSpace: "nowrap", lineHeight: 1, ...style }}>
      <span style={{ width: 5, height: 5, borderRadius: 999, background: t.dot }} />
      {state}
    </span>
  );
}
