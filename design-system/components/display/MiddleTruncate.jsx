import React from "react";

export function MiddleTruncate({ value = "", tail = 12, mono = true, style }) {
  const cut = Math.max(0, value.length - tail);
  const head = value.slice(0, cut), tl = value.slice(cut);
  const font = mono ? "500 12.5px var(--font-mono)" : "400 13px var(--font-ui)";
  if (!head) return <span title={value} style={{ font, whiteSpace: "nowrap", ...style }}>{value}</span>;
  return (
    <span title={value} style={{ display: "inline-flex", minWidth: 0, maxWidth: "100%", font, whiteSpace: "nowrap", verticalAlign: "bottom", ...style }}>
      <span style={{ overflow: "hidden", textOverflow: "ellipsis", minWidth: 0 }}>{head}</span>
      <span style={{ flex: "none" }}>{tl}</span>
    </span>
  );
}
