import React from "react";
import { Icon } from "../media/Icon.jsx";

function MenuItem({ item, close }) {
  const [hov, setHov] = React.useState(false);
  const danger = item.tone === "danger";
  return (
    <button type="button" disabled={item.disabled}
      onClick={(e) => { e.stopPropagation(); close(); item.onSelect && item.onSelect(); }}
      onMouseEnter={() => setHov(true)} onMouseLeave={() => setHov(false)}
      style={{ display: "flex", alignItems: "center", gap: 9, width: "100%", height: 30, padding: "0 10px", border: "none", borderRadius: 8, textAlign: "left", cursor: item.disabled ? "default" : "pointer", opacity: item.disabled ? 0.45 : 1, background: hov && !item.disabled ? (danger ? "var(--danger-soft)" : "var(--surface-sunken)") : "transparent", color: danger ? "var(--danger)" : "var(--text-body)", font: "500 12.5px var(--font-ui)", transition: "background var(--dur-fast) var(--ease-out)" }}>
      {item.icon && <span style={{ display: "inline-flex", color: danger ? "var(--danger)" : "var(--text-secondary)" }}><Icon name={item.icon} size={14} /></span>}
      <span style={{ flex: 1, whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis" }}>{item.label}</span>
      {item.shortcut && <span style={{ font: "400 10.5px var(--font-mono)", color: "var(--text-muted)" }}>{item.shortcut}</span>}
    </button>
  );
}

export function DropdownMenu({ trigger, items = [], align = "end", width = 200, style }) {
  const [open, setOpen] = React.useState(false);
  const [up, setUp] = React.useState(false);
  const ref = React.useRef(null);
  React.useEffect(() => {
    if (!open) return;
    const onDoc = (e) => { if (ref.current && !ref.current.contains(e.target)) setOpen(false); };
    const onKey = (e) => { if (e.key === "Escape") setOpen(false); };
    document.addEventListener("mousedown", onDoc);
    document.addEventListener("keydown", onKey);
    return () => { document.removeEventListener("mousedown", onDoc); document.removeEventListener("keydown", onKey); };
  }, [open]);
  const pos = up ? { position: "absolute", zIndex: 95, bottom: "100%", marginBottom: 6 } : { position: "absolute", zIndex: 95, top: "100%", marginTop: 6 };
  if (align === "end") pos.right = 0; else pos.left = 0;
  const toggle = (e) => {
    e.stopPropagation();
    if (!open && ref.current) {
      const r = ref.current.getBoundingClientRect();
      const est = items.length * 30 + 16;
      setUp(window.innerHeight - r.bottom < est && r.top > est);
    }
    setOpen(!open);
  };
  return (
    <span ref={ref} style={{ position: "relative", display: "inline-flex", ...style }}>
      <span onClick={toggle} style={{ display: "inline-flex" }}>{trigger}</span>
      {open && (
        <div role="menu" onClick={(e) => e.stopPropagation()} style={{ ...pos, width, background: "var(--surface-raised)", border: "1px solid var(--border-default)", borderRadius: 12, boxShadow: "var(--shadow-md)", padding: 5, animation: "vg-pop-in var(--dur-base) var(--ease-out)" }}>
          {items.map((it, i) => it === "-"
            ? <div key={"s" + i} style={{ height: 1, background: "var(--row-sep)", margin: "5px 4px" }} />
            : <MenuItem key={it.label} item={it} close={() => setOpen(false)} />)}
        </div>
      )}
    </span>
  );
}
