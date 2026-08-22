import React from "react";

/* Keyboard key cap. keys="esc" or keys={["mod","K"]} — the "mod" token renders ⌘ on Mac, Ctrl elsewhere. */
export function Kbd({ keys, style }) {
  const isMac = /Mac|iP(hone|ad|od)/.test(navigator.platform || navigator.userAgent);
  const list = (Array.isArray(keys) ? keys : [keys]).map((k) => (k === "mod" ? (isMac ? "\u2318" : "Ctrl") : k));
  return (
    <span style={{ display: "inline-flex", gap: 3, ...style }}>
      {list.map((k, i) => (
        <kbd key={i} style={{ display: "inline-flex", alignItems: "center", justifyContent: "center", minWidth: 18, height: 18, padding: "0 5px", background: "var(--surface-sunken)", border: "1px solid var(--border-default)", borderBottomWidth: 2, borderRadius: 5, font: "500 10.5px var(--font-mono)", color: "var(--text-secondary)" }}>{k}</kbd>
      ))}
    </span>
  );
}
