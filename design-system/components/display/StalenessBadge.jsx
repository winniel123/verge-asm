import React from "react";

/* Currency states on the bronze stale palette — never severity or drift colors.
   kinds: stale (old observation), not-evaluable, silent ("you stopped telling us"). */
const LABELS = { stale: "stale", "not-evaluable": "not evaluable", silent: "no reports" };

export function StalenessBadge({ kind = "stale", bound, size = "md", style }) {
  const sm = size === "sm";
  return (
    <span style={{ display: "inline-flex", alignItems: "center", gap: sm ? 4 : 5, height: sm ? 18 : 20, padding: sm ? "0 6px" : "0 8px", borderRadius: 8, background: "var(--stale-bg)", border: "1px solid var(--stale-border)", color: "var(--stale-fg)", fontFamily: "var(--font-mono)", fontSize: sm ? 10 : 10.5, fontWeight: 600, letterSpacing: "0.04em", whiteSpace: "nowrap", lineHeight: 1, ...style }}>
      <svg viewBox="0 0 10 10" width={sm ? 9 : 10} height={sm ? 9 : 10} style={{ flex: "none" }}>
        <circle cx="5" cy="5" r="3.6" fill="none" stroke="currentColor" strokeWidth="1.4"></circle>
        <path d="M5 3.2V5l1.4 1" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round"></path>
      </svg>
      {LABELS[kind] || kind}
      {bound && <span style={{ fontWeight: 400, opacity: 0.8 }}>· {bound}</span>}
    </span>
  );
}
