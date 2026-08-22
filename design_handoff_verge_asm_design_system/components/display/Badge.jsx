import React from "react";

const TONES = {
  neutral: { bg: "var(--surface-sunken)", fg: "var(--text-secondary)", bd: "var(--border-default)" },
  accent: { bg: "var(--accent-soft)", fg: "var(--link)", bd: "transparent" },
  ok: { bg: "var(--ok-soft)", fg: "var(--ok)", bd: "var(--ok-border)" },
  warn: { bg: "var(--warn-soft)", fg: "var(--warn)", bd: "var(--warn-border)" },
  danger: { bg: "var(--danger-soft)", fg: "var(--danger)", bd: "var(--danger-border)" },
};

export function Badge({ children, tone = "neutral", dot, style }) {
  const t = TONES[tone] || TONES.neutral;
  return (
    <span style={{ display: "inline-flex", alignItems: "center", gap: 6, height: 22, padding: "0 10px", borderRadius: 999, background: t.bg, border: "1px solid " + t.bd, color: t.fg, fontFamily: "var(--font-ui)", fontSize: 12, fontWeight: 500, whiteSpace: "nowrap", ...style }}>
      {dot && <span style={{ width: 6, height: 6, borderRadius: 999, background: "currentColor" }} />}
      {children}
    </span>
  );
}
