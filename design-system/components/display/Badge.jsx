import React from "react";

const soft = {
  neutral: { background: "var(--surface-sunken)", color: "var(--text-muted)", border: "1px solid var(--border)" },
  accent: { background: "var(--accent-soft)", color: "var(--accent)", border: "1px solid transparent" },
  ok: { background: "var(--ok-soft)", color: "var(--ok)", border: "1px solid transparent" },
  warn: { background: "var(--warn-soft)", color: "var(--warn)", border: "1px solid transparent" },
  danger: { background: "var(--danger-soft)", color: "var(--danger)", border: "1px solid transparent" },
};
const solidMap = {
  neutral: { background: "var(--ink)", color: "var(--text-on-ink)" },
  accent: { background: "var(--accent)", color: "#ffffff" },
  ok: { background: "var(--ok)", color: "#ffffff" },
  warn: { background: "var(--warn)", color: "#ffffff" },
  danger: { background: "var(--danger)", color: "#ffffff" },
};

export function Badge({ tone = "neutral", solid, style, children }) {
  return (
    <span style={{
      display: "inline-block", padding: "3px 7px",
      font: "600 10px/1.2 var(--font-mono)", letterSpacing: "0.05em", textTransform: "uppercase",
      whiteSpace: "nowrap",
      ...(solid ? solidMap[tone] : soft[tone]), ...style,
    }}>{children}</span>
  );
}
