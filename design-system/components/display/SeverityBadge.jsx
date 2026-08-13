import React from "react";

const MAP = {
  critical: { background: "var(--sev-critical)", color: "#ffffff" },
  high: { background: "var(--sev-high)", color: "#ffffff" },
  medium: { background: "var(--sev-medium)", color: "var(--sev-medium-text)" },
  low: { background: "var(--sev-low)", color: "#ffffff" },
  info: { background: "var(--sev-info)", color: "#ffffff" },
};

export function SeverityBadge({ severity = "info", style }) {
  const s = MAP[severity] || MAP.info;
  return (
    <span style={{
      display: "inline-block", padding: "3px 7px", minWidth: 52, textAlign: "center", boxSizing: "border-box",
      font: "600 10px/1.2 var(--font-mono)", letterSpacing: "0.05em", textTransform: "uppercase",
      whiteSpace: "nowrap", ...s, ...style,
    }}>{severity}</span>
  );
}
