import React from "react";
import { Tooltip } from "../feedback/Tooltip.jsx";
import { Icon } from "../media/Icon.jsx";

/* Reference to the rule that raised a signal: id@version, mono chip. */
export function SignalRuleRef({ id, version, onClick, style }) {
  const [hov, setHov] = React.useState(false);
  const chip = (
    <span role={onClick ? "button" : undefined} onClick={onClick}
      onMouseEnter={() => setHov(true)} onMouseLeave={() => setHov(false)}
      style={{ display: "inline-flex", alignItems: "center", gap: 6, height: 22, padding: "0 8px", borderRadius: 8, lineHeight: 1,
        background: "var(--surface-sunken)", border: "1px solid " + (hov && onClick ? "color-mix(in srgb, var(--accent) 45%, transparent)" : "var(--border-default)"),
        color: hov && onClick ? "var(--link)" : "var(--text-body)", font: "500 11.5px var(--font-mono)",
        cursor: onClick ? "pointer" : "default", transition: "color var(--dur-fast) var(--ease-out), border-color var(--dur-fast) var(--ease-out)", ...style }}>
      <Icon name="file-code" size={11} />
      {id}@{version}
    </span>
  );
  return <Tooltip content={"Rule " + id + " v" + version + " — definition in source"} side="top">{chip}</Tooltip>;
}
