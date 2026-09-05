import React from "react";

const FAMILY = { appeared: "gain", revealed: "gain", returned: "gain", withdrawn: "loss", descoped: "loss", changed: "change" };

export function ChangeGlyph({ change, size = 10 }) {
  const p = { fill: "none", stroke: "currentColor", strokeWidth: 1.6, strokeLinecap: "round", strokeLinejoin: "round" };
  return (
    <svg viewBox="0 0 10 10" width={size} height={size} style={{ flex: "none" }}>
      {change === "appeared" && <path d="M5 1.6v6.8M1.6 5h6.8" {...p}></path>}
      {change === "revealed" && <circle cx="5" cy="5" r="3.4" {...p} strokeDasharray="2.1 1.7"></circle>}
      {change === "returned" && <g><path d="M8.2 5.8A3.3 3.3 0 1 1 7.4 2.8" {...p}></path><path d="M7 1l0.6 2L5.6 3.4" {...p}></path></g>}
      {change === "withdrawn" && <path d="M1.6 5h6.8" {...p}></path>}
      {change === "descoped" && <g><path d="M1.6 6.4h6.8" {...p}></path><circle cx="5" cy="2.6" r="1" fill="currentColor" stroke="none"></circle></g>}
      {change === "changed" && <path d="M5 1.8L8.4 8H1.6Z" {...p}></path>}
    </svg>
  );
}

export function ChangeBadge({ change = "changed", reason, size = "md", style }) {
  const fam = FAMILY[change] || "change";
  const sm = size === "sm";
  return (
    <span style={{ display: "inline-flex", alignItems: "center", gap: sm ? 4 : 6, height: sm ? 18 : 20, padding: sm ? "0 6px" : "0 8px", borderRadius: 8, background: "var(--drift-" + fam + "-bg)", border: "1px solid var(--drift-" + fam + "-border)", color: "var(--drift-" + fam + "-fg)", fontFamily: "var(--font-mono)", fontSize: sm ? 10 : 10.5, fontWeight: 600, letterSpacing: "0.04em", whiteSpace: "nowrap", ...style }}>
      <ChangeGlyph change={change} size={sm ? 9 : 10} />
      {change}
      {reason && <span style={{ fontWeight: 400, opacity: 0.75 }}>· {reason}</span>}
    </span>
  );
}
