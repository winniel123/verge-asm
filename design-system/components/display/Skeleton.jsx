import React from "react";

export function Skeleton({ shape = "line", width, height, lines = 1, mono, style }) {
  const base = { background: "var(--border-default)", animation: "vg-skeleton 1.6s var(--ease-in-out) infinite" };
  if (shape === "circle") {
    const d = width || 28;
    return <span aria-hidden="true" style={{ ...base, display: "inline-block", width: d, height: d, borderRadius: 999, ...style }} />;
  }
  if (shape === "block") {
    return <span aria-hidden="true" style={{ ...base, display: "block", width: width || "100%", height: height || 80, borderRadius: 12, ...style }} />;
  }
  const h = height || (mono ? 12 : 13);
  return (
    <span aria-hidden="true" style={{ display: "flex", flexDirection: "column", gap: 8, width: width || "100%", ...style }}>
      {Array.from({ length: lines }).map((_, i) => (
        <span key={i} style={{ ...base, display: "block", height: h, borderRadius: 6, width: i === lines - 1 && lines > 1 ? "62%" : "100%" }} />
      ))}
    </span>
  );
}
