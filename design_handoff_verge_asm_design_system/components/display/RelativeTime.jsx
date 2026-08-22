import React from "react";
import { Tooltip } from "../feedback/Tooltip.jsx";

/* The canonical timestamp pattern: terse relative value, absolute ISO 8601 on hover. */
export function RelativeTime({ value, iso, side = "top", style }) {
  return (
    <Tooltip content={iso} mono side={side}>
      <span style={{ font: "500 12.5px var(--font-mono)", color: "var(--text-body)", borderBottom: "1px dashed var(--border-strong)", cursor: "default", whiteSpace: "nowrap", ...style }}>{value}</span>
    </Tooltip>
  );
}
