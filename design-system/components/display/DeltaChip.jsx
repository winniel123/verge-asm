import React from "react";

export function DeltaChip({ value, tone = "neutral", size = "sm", style }) {
  const s = String(value == null ? "" : value).trim();
  const up = s.charAt(0) === "+";
  const down = s.charAt(0) === "-" || s.charAt(0) === "\u2212";
  const c = tone === "good" ? { bg: "var(--ok-soft)", fg: "var(--ok)", bd: "var(--ok-border)" }
    : tone === "bad" ? { bg: "var(--danger-soft)", fg: "var(--danger)", bd: "var(--danger-border)" }
    : { bg: "var(--surface-sunken)", fg: "var(--text-secondary)", bd: "var(--border-default)" };
  const h = size === "md" ? 22 : 18;
  return (
    <span style={{ display: "inline-flex", alignItems: "center", gap: 4, height: h, padding: "0 7px", borderRadius: 999, background: c.bg, border: "1px solid " + c.bd, color: c.fg, font: "600 " + (size === "md" ? 12 : 11) + "px var(--font-mono)", lineHeight: 1, whiteSpace: "nowrap", ...style }}>
      {(up || down) && (
        <svg viewBox="0 0 10 10" width="8" height="8" aria-hidden="true" style={{ transform: down ? "rotate(180deg)" : "none" }}>
          <path d="M5 8.5V1.5M1.8 4.7L5 1.5l3.2 3.2" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round"></path>
        </svg>
      )}
      {s}
    </span>
  );
}
