import React from "react";

/* Pulse glyph: one solid signal inside two watch rings. Accent azure only — never severity colors. */
export function Logo({ size = 20, withWordmark = true, wordmarkSize, inverted, tile, style }) {
  const stroke = tile ? "#ffffff" : inverted ? "var(--primary-400)" : "var(--accent)";
  const glyph = (
    <svg width={size} height={size} viewBox="0 0 44 44" style={{ flex: "none", display: "block" }}>
      <circle cx="22" cy="22" r="19" fill="none" stroke={stroke} strokeWidth={size < 24 ? 3 : 2} opacity="0.4"></circle>
      <circle cx="22" cy="22" r="12" fill="none" stroke={stroke} strokeWidth={size < 24 ? 3.5 : 2.5} opacity="0.75"></circle>
      <circle cx="22" cy="22" r="5.5" fill={stroke}></circle>
    </svg>
  );
  const mark = tile ? (
    <span style={{ display: "inline-flex", alignItems: "center", justifyContent: "center", width: size * 1.65, height: size * 1.65, borderRadius: size * 0.46, background: "var(--accent)", flex: "none", ...style }}>{glyph}</span>
  ) : glyph;
  if (!withWordmark) return tile ? mark : <span style={{ display: "inline-flex", ...style }}>{glyph}</span>;
  const ws = wordmarkSize || Math.round(size * 0.85);
  return (
    <span style={{ display: "inline-flex", alignItems: "center", gap: Math.max(6, Math.round(size * 0.35)), ...style }}>
      {mark}
      <span style={{ display: "inline-flex", alignItems: "baseline", gap: Math.max(4, Math.round(ws * 0.28)) }}>
        <span style={{ font: "650 " + ws + "px var(--font-ui)", letterSpacing: "-0.025em", color: inverted ? "var(--text-on-inverted)" : "var(--text-ink)", lineHeight: 1 }}>verge</span>
        <span style={{ font: "600 " + Math.max(9, Math.round(ws * 0.55)) + "px var(--font-mono)", letterSpacing: "0.08em", color: inverted ? "var(--primary-400)" : "var(--accent)" }}>ASM</span>
      </span>
    </span>
  );
}
