import React from "react";
import { Tag } from "../display/Tag.jsx";

/* In-app message inbox — one row per firing of one cause. messages: [{id, cls, text, time, unread}]. */
export function MessageList({ messages = [], onOpen, style }) {
  const [hov, setHov] = React.useState(null);
  return (
    <div style={{ display: "flex", flexDirection: "column", fontFamily: "var(--font-ui)", ...style }}>
      {messages.map((m, i) => (
        <button key={m.id} type="button" onClick={() => onOpen && onOpen(m)}
          onMouseEnter={() => setHov(m.id)} onMouseLeave={() => setHov(null)}
          style={{ display: "flex", alignItems: "flex-start", gap: 10, width: "100%", padding: "10px 8px", border: "none", borderTop: i ? "1px solid var(--row-sep)" : "none", background: hov === m.id ? "var(--surface-sunken)" : "transparent", cursor: onOpen ? "pointer" : "default", textAlign: "left", borderRadius: 8, transition: "background var(--dur-fast) var(--ease-out)" }}>
          <span style={{ width: 7, height: 7, borderRadius: 999, marginTop: 5, flex: "none", background: m.unread ? "var(--accent)" : "transparent" }} />
          <span style={{ display: "flex", flexDirection: "column", gap: 3, minWidth: 0, flex: 1 }}>
            <span style={{ display: "flex", alignItems: "center", gap: 8 }}>
              <Tag>{m.cls}</Tag>
              <span style={{ marginLeft: "auto", font: "400 11px var(--font-mono)", color: "var(--text-muted)", whiteSpace: "nowrap" }}>{m.time}</span>
            </span>
            <span style={{ font: (m.unread ? "500" : "400") + " 12.5px/1.5 var(--font-ui)", color: "var(--text-body)" }}>{m.text}</span>
          </span>
        </button>
      ))}
      {!messages.length && <span style={{ font: "400 12.5px var(--font-ui)", color: "var(--text-muted)", padding: 8 }}>No messages.</span>}
    </div>
  );
}
