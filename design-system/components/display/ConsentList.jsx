import React from "react";
import { Icon } from "../media/Icon.jsx";
import { Tag } from "./Tag.jsx";

export function ConsentList({ grants = [], style }) {
  return (
    <ul style={{ listStyle: "none", margin: 0, padding: 0, display: "flex", flexDirection: "column", gap: 2, fontFamily: "var(--font-ui)", ...style }}>
      {grants.map((g, i) => (
        <li key={i} style={{ display: "flex", alignItems: "flex-start", gap: 10, padding: "8px 10px", borderRadius: 10, background: g.write ? "color-mix(in oklab, var(--warn) 6%, transparent)" : "transparent" }}>
          <span style={{ display: "inline-flex", alignItems: "center", justifyContent: "center", width: 20, height: 20, borderRadius: "50%", background: g.write ? "color-mix(in oklab, var(--warn) 14%, transparent)" : "var(--surface-sunken)", border: "1px solid " + (g.write ? "color-mix(in oklab, var(--warn) 35%, transparent)" : "var(--border-default)"), flex: "none", marginTop: 1 }}>
            <Icon name={g.write ? "pencil" : "eye"} size={11} style={{ color: g.write ? "var(--warn)" : "var(--text-secondary)" }} />
          </span>
          <span style={{ display: "flex", flexDirection: "column", gap: 2, minWidth: 0 }}>
            <span style={{ display: "flex", alignItems: "center", gap: 8, font: "500 13px var(--font-ui)", color: "var(--text-ink)" }}>
              {g.scope}
              {g.write && <Tag>writes</Tag>}
            </span>
            {g.detail && <span style={{ font: "400 12px/1.55 var(--font-ui)", color: "var(--text-muted)" }}>{g.detail}</span>}
          </span>
        </li>
      ))}
    </ul>
  );
}
