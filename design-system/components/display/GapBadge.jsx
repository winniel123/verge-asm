import React from "react";

export function GapBadge({ label = "gap", size = "md", style }) {
  const sm = size === "sm";
  return (
    <span style={{ display: "inline-flex", alignItems: "center", gap: sm ? 4 : 5, height: sm ? 18 : 20, padding: sm ? "0 6px" : "0 8px", borderRadius: 8, background: "transparent", border: "1px dotted var(--border-strong)", color: "var(--text-secondary)", fontFamily: "var(--font-mono)", fontSize: sm ? 10 : 10.5, fontWeight: 600, letterSpacing: "0.04em", whiteSpace: "nowrap", ...style }}>
      <svg viewBox="0 0 10 10" width={sm ? 9 : 10} height={sm ? 9 : 10} style={{ flex: "none" }}>
        <rect x="1.6" y="1.6" width="6.8" height="6.8" rx="1.5" fill="none" stroke="currentColor" strokeWidth="1.4" strokeDasharray="1.6 1.6"></rect>
      </svg>
      {label}
    </span>
  );
}
