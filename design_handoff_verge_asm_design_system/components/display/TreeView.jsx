import React from "react";

/* Name-hierarchy tree. nodes: [{id, label, count?, sev?, children?}]. Labels are mono by default. */
function Row({ n, depth, openIds, toggle, onSelect, selectedId }) {
  const [hov, setHov] = React.useState(false);
  const isOpen = openIds.indexOf(n.id) !== -1;
  const kids = n.children || [];
  const selected = selectedId === n.id;
  return (
    <div>
      <div onClick={() => { if (kids.length) toggle(n.id); onSelect && onSelect(n); }}
        onMouseEnter={() => setHov(true)} onMouseLeave={() => setHov(false)}
        style={{ display: "flex", alignItems: "center", gap: 7, padding: "5px 8px", paddingLeft: 8 + depth * 18, borderRadius: 8, cursor: "pointer", background: selected ? "var(--accent-soft)" : hov ? "var(--surface-sunken)" : "transparent", transition: "background var(--dur-fast) var(--ease-out)" }}>
        {kids.length ? (
          <svg viewBox="0 0 16 16" width="12" height="12" style={{ color: "var(--text-muted)", transform: isOpen ? "rotate(0)" : "rotate(-90deg)", transition: "transform var(--dur-base) var(--ease-out)", flex: "none" }}>
            <path d="M4 6l4 4 4-4" fill="none" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round"></path>
          </svg>
        ) : <span style={{ width: 12, flex: "none" }} />}
        {n.sev && <span style={{ width: 6, height: 6, borderRadius: 999, background: "var(--sev-" + n.sev + "-dot)", flex: "none" }} />}
        <span style={{ font: (selected ? "600" : "500") + " 12.5px var(--font-mono)", color: selected ? "var(--link)" : "var(--text-body)", whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis" }}>{n.label}</span>
        {n.count != null && <span style={{ marginLeft: "auto", font: "400 10.5px var(--font-mono)", color: "var(--text-muted)" }}>{n.count}</span>}
      </div>
      {kids.length > 0 && (
        <div style={{ display: "grid", gridTemplateRows: isOpen ? "1fr" : "0fr", transition: "grid-template-rows var(--dur-base) var(--ease-out)" }}>
          <div style={{ overflow: "hidden", minHeight: 0 }}>
            {kids.map((k) => <Row key={k.id} n={k} depth={depth + 1} openIds={openIds} toggle={toggle} onSelect={onSelect} selectedId={selectedId} />)}
          </div>
        </div>
      )}
    </div>
  );
}

export function TreeView({ nodes = [], defaultOpen = [], onSelect, selectedId, style }) {
  const [openIds, setOpenIds] = React.useState(defaultOpen);
  const toggle = (id) => setOpenIds((o) => (o.indexOf(id) !== -1 ? o.filter((x) => x !== id) : o.concat(id)));
  return (
    <div style={{ display: "flex", flexDirection: "column", fontFamily: "var(--font-ui)", ...style }}>
      {nodes.map((n) => <Row key={n.id} n={n} depth={0} openIds={openIds} toggle={toggle} onSelect={onSelect} selectedId={selectedId} />)}
    </div>
  );
}
