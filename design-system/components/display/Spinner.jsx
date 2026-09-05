import React from "react";

export function Spinner({ size = 16, label, style }) {
  const s = size;
  return (
    <span style={{ display: "inline-flex", alignItems: "center", gap: 8, ...style }}>
      <svg width={s} height={s} viewBox="0 0 20 20" style={{ animation: "vg-spin 0.9s linear infinite", flex: "none" }} aria-hidden="true">
        <circle cx="10" cy="10" r="8" fill="none" stroke="var(--border-default)" strokeWidth="2.5"></circle>
        <path d="M10 2a8 8 0 0 1 8 8" fill="none" stroke="var(--accent)" strokeWidth="2.5" strokeLinecap="round"></path>
      </svg>
      {label && <span style={{ font: "400 12.5px var(--font-ui)", color: "var(--text-secondary)" }}>{label}</span>}
    </span>
  );
}
