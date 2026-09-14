import React from "react";

const TONE = {
  danger: { bg: "var(--danger-soft)", bd: "var(--danger-border)", fg: "var(--danger)" },
  warn: { bg: "var(--warn-soft)", bd: "var(--warn-border)", fg: "var(--warn)" },
  neutral: { bg: "var(--surface-sunken)", bd: "var(--border-default)", fg: "var(--text-secondary)" },
  absent: { bg: "transparent", bd: "var(--border-strong)", fg: "var(--text-secondary)", dashed: true },
};
const LABEL = {
  reached: "reached",
  "not-reached": "not reached",
  "never-looked": "never looked",
  "stopped-looking": "stopped looking",
};
export function ReachLegBadge({ state = "never-looked", legClass = "internal", size = "md", style }) {
  // Only the internet leg's reached is the move the product alerts on (ADR-0029).
  const tone =
    state === "reached"
      ? legClass === "internet"
        ? "danger"
        : "neutral"
      : state === "not-reached"
        ? "neutral"
        : state === "stopped-looking"
          ? "warn"
          : "absent";
  const t = TONE[tone];
  const sm = size === "sm";
  return (
    <span style={{ display: "inline-flex", alignItems: "center", height: sm ? 18 : 20, padding: sm ? "0 7px" : "0 9px", borderRadius: 8, background: t.bg, border: "1px " + (t.dashed ? "dashed" : "solid") + " " + t.bd, color: t.fg, fontFamily: "var(--font-mono)", fontSize: sm ? 10 : 10.5, fontWeight: 600, letterSpacing: "0.04em", whiteSpace: "nowrap", lineHeight: 1, ...style }}>
      {LABEL[state] || LABEL["never-looked"]}
    </span>
  );
}
