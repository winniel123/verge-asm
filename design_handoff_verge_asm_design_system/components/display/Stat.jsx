import React from "react";
import { DeltaChip } from "./DeltaChip.jsx";

export function Stat({ label, value, delta, deltaTone = "neutral", caption, live, style }) {
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 4, minWidth: 0, ...style }}>
      <span style={{ display: "flex", alignItems: "center", gap: 8, font: "500 11px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--text-muted)" }}>
        {label}
        {live && <span style={{ width: 7, height: 7, borderRadius: 999, background: "var(--accent)", animation: "vg-pulse 1.8s var(--ease-out) infinite" }} />}
      </span>
      <span style={{ display: "flex", alignItems: "baseline", gap: 8 }}>
        <span style={{ font: "600 28px var(--font-mono)", color: "var(--text-ink)", lineHeight: 1.1 }}>{value}</span>
        {delta && <DeltaChip value={delta} tone={deltaTone} style={{ transform: "translateY(-2px)" }} />}
      </span>
      {caption && <span style={{ font: "400 11.5px var(--font-ui)", color: "var(--text-muted)" }}>{caption}</span>}
    </div>
  );
}
