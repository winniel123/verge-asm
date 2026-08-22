import React from "react";

const LEVELS = ["critical", "high", "medium", "low", "info"];

export function SeverityBadge({ level = "info", size = "md", style }) {
  const l = LEVELS.includes(level) ? level : "info";
  const sm = size === "sm";
  const base = { display: "inline-flex", alignItems: "center", gap: sm ? 5 : 6, height: sm ? 18 : 22, padding: sm ? "0 8px" : "0 10px", borderRadius: 999, fontFamily: "var(--font-mono)", fontSize: sm ? 10 : 11, fontWeight: 600, letterSpacing: "0.05em", textTransform: "uppercase", whiteSpace: "nowrap" };
  if (l === "critical") {
    return <span style={{ ...base, background: "var(--sev-critical-fill)", color: "var(--sev-critical-text)", ...style }}>Critical</span>;
  }
  return (
    <span style={{ ...base, background: "var(--sev-" + l + "-bg)", border: "1px solid var(--sev-" + l + "-border)", color: "var(--sev-" + l + "-fg)", ...style }}>
      <span style={{ width: sm ? 5 : 6, height: sm ? 5 : 6, borderRadius: 999, background: "var(--sev-" + l + "-dot)", flex: "none" }} />
      {l.charAt(0).toUpperCase() + l.slice(1)}
    </span>
  );
}
