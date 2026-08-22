import React from "react";

/* Disclosure list. items: [{id, title, content}]. Single-open by default. */
export function Accordion({ items = [], multiple, defaultOpen = [], style }) {
  const [open, setOpen] = React.useState(defaultOpen);
  const toggle = (id) => setOpen((o) => o.indexOf(id) !== -1 ? o.filter((x) => x !== id) : multiple ? o.concat(id) : [id]);
  return (
    <div style={{ display: "flex", flexDirection: "column", fontFamily: "var(--font-ui)", ...style }}>
      {items.map((it, i) => {
        const isOpen = open.indexOf(it.id) !== -1;
        return (
          <div key={it.id} style={{ borderTop: i ? "1px solid var(--row-sep)" : "none" }}>
            <button type="button" aria-expanded={isOpen} onClick={() => toggle(it.id)}
              style={{ display: "flex", alignItems: "center", gap: 10, width: "100%", padding: "14px 2px", border: "none", background: "transparent", cursor: "pointer", textAlign: "left" }}>
              <span style={{ flex: 1, font: "500 13.5px var(--font-ui)", color: "var(--text-ink)" }}>{it.title}</span>
              <svg viewBox="0 0 16 16" width="14" height="14" style={{ color: "var(--text-muted)", transform: isOpen ? "rotate(180deg)" : "none", transition: "transform var(--dur-base) var(--ease-out)", flex: "none" }}>
                <path d="M4 6l4 4 4-4" fill="none" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round"></path>
              </svg>
            </button>
            <div style={{ display: "grid", gridTemplateRows: isOpen ? "1fr" : "0fr", transition: "grid-template-rows var(--dur-slow) var(--ease-out)" }}>
              <div style={{ overflow: "hidden", minHeight: 0 }}>
                <div style={{ padding: "0 24px 16px 2px", font: "400 13px/1.6 var(--font-ui)", color: "var(--text-secondary)", opacity: isOpen ? 1 : 0, transition: "opacity var(--dur-base) var(--ease-out)" }}>{it.content}</div>
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
}
