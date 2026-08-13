import React from "react";

const SIZES = { sm: { word: 13, chip: 10, pad: "2px 5px" }, md: { word: 15, chip: 11, pad: "2px 6px" }, lg: { word: 28, chip: 15, pad: "3px 8px" } };

export function Wordmark({ size = "md", onInk, style }) {
  const s = SIZES[size] || SIZES.md;
  return (
    <span style={{ display: "inline-flex", alignItems: "baseline", gap: 7, ...style }}>
      <span style={{ font: `700 ${s.word}px/1 var(--font-sans)`, letterSpacing: "-0.01em", color: onInk ? "var(--text-on-ink)" : "var(--ink)" }}>Verge</span>
      <span style={{ font: `600 ${s.chip}px/1.2 var(--font-mono)`, padding: s.pad, background: onInk ? "var(--paper)" : "var(--ink)", color: onInk ? "var(--ink)" : "var(--paper)" }}>ASM</span>
    </span>
  );
}
