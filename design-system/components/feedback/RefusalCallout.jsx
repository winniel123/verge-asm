import React from "react";
import { Icon } from "../media/Icon.jsx";

export function RefusalCallout({ input, reason, reachable, style }) {
  return (
    <div style={{ display: "flex", gap: 10, alignItems: "flex-start", padding: "12px 14px", background: "var(--danger-soft)", border: "1px solid var(--danger-border)", borderRadius: 12, fontFamily: "var(--font-ui)", ...style }}>
      <span style={{ color: "var(--danger)", display: "inline-flex", marginTop: 1, flex: "none" }}><Icon name="alert-octagon" size={15} /></span>
      <div style={{ display: "flex", flexDirection: "column", gap: 5, minWidth: 0 }}>
        <span style={{ font: "600 13px var(--font-ui)", color: "var(--text-ink)" }}>Declaration refused: <span style={{ font: "600 12.5px var(--font-mono)" }}>{input}</span></span>
        <span style={{ font: "400 12.5px/1.5 var(--font-ui)", color: "var(--text-body)" }}>{reason}</span>
        {reachable && (
          <span style={{ font: "400 12px var(--font-ui)", color: "var(--text-secondary)" }}>
            Reachable set: <span style={{ font: "500 12px var(--font-mono)", color: "var(--text-body)" }}>{reachable}</span> — nothing is auto-corrected; declare it yourself.
          </span>
        )}
      </div>
    </div>
  );
}
