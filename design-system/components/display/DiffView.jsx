import React from "react";

export function DiffView({ lines = [], title, style }) {
  const row = (l, i) => {
    const add = l.type === "add", rem = l.type === "remove";
    return (
      <div key={i} style={{ display: "flex", gap: 10, padding: "3px 12px", background: add ? "var(--ok-soft)" : rem ? "var(--danger-soft)" : "transparent" }}>
        <span style={{ width: 10, flex: "none", font: "600 12px var(--font-mono)", color: add ? "var(--ok)" : rem ? "var(--danger)" : "var(--text-muted)" }}>{add ? "+" : rem ? "\u2212" : ""}</span>
        <span style={{ font: "400 12px/1.6 var(--font-mono)", color: add ? "var(--ok)" : rem ? "var(--danger)" : "var(--text-secondary)", whiteSpace: "pre-wrap", overflowWrap: "anywhere" }}>{l.text}</span>
      </div>
    );
  };
  return (
    <div style={{ border: "1px solid var(--border-default)", borderRadius: 12, overflow: "hidden", background: "var(--surface)", ...style }}>
      {title && <div style={{ padding: "7px 12px", borderBottom: "1px solid var(--row-sep)", font: "500 11px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--text-muted)" }}>{title}</div>}
      <div style={{ padding: "6px 0" }}>{lines.map(row)}</div>
    </div>
  );
}
