import React from "react";
import { GapBadge } from "./GapBadge.jsx";
import { StalenessBadge } from "./StalenessBadge.jsx";
import { RelativeTime } from "./RelativeTime.jsx";

export function CoverageMessageList({ messages = [], style }) {
  return (
    <div style={{ display: "flex", flexDirection: "column", fontFamily: "var(--font-ui)", ...style }}>
      {messages.map((m, i) => (
        <div key={m.id || i} style={{ display: "grid", gridTemplateColumns: "auto 1fr auto", gap: 12, alignItems: "start", padding: "11px 0", borderTop: i ? "1px solid var(--row-sep)" : "none" }}>
          <span style={{ paddingTop: 1 }}>
            {m.kind === "gap" ? <GapBadge size="sm" label={m.badge || "gap"} /> : <StalenessBadge size="sm" kind={m.kind} bound={m.bound} />}
          </span>
          <span style={{ display: "flex", flexDirection: "column", gap: 2, minWidth: 0 }}>
            <span style={{ font: "600 12px var(--font-mono)", color: "var(--text-ink)", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>{m.subject}</span>
            <span style={{ font: "400 12.5px/1.5 var(--font-ui)", color: "var(--text-secondary)" }}>{m.text}</span>
          </span>
          {m.when && <RelativeTime value={m.when} iso={m.iso} side="left" />}
        </div>
      ))}
    </div>
  );
}
