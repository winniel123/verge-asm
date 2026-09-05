import React from "react";

const S = {
  exposed: { bg: "var(--danger-soft)", bd: "var(--danger-border)", fg: "var(--danger)" },
  firewalled: { bg: "var(--ok-soft)", bd: "var(--ok-border)", fg: "var(--ok)" },
  "not-reached": { bg: "var(--surface-sunken)", bd: "var(--border-default)", fg: "var(--text-secondary)" },
  unverified: { bg: "transparent", bd: "var(--border-strong)", fg: "var(--text-secondary)", dashed: true },
};
const LABEL = { "not-reached": "not reached" };
export function ExposureBadge({ state = "unverified", size = "md", style }) {
  const t = S[state] || S.unverified;
  const sm = size === "sm";
  return (
    <span style={{ display: "inline-flex", alignItems: "center", height: sm ? 18 : 20, padding: sm ? "0 7px" : "0 9px", borderRadius: 8, background: t.bg, border: "1px " + (t.dashed ? "dashed" : "solid") + " " + t.bd, color: t.fg, fontFamily: "var(--font-mono)", fontSize: sm ? 10 : 10.5, fontWeight: 600, letterSpacing: "0.04em", whiteSpace: "nowrap", lineHeight: 1, ...style }}>
      {LABEL[state] || state}
    </span>
  );
}
