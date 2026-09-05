import React from "react";
import { Icon } from "../media/Icon.jsx";

function ContextItem({ it, onPick }) {
  const [hov, setHov] = React.useState(false);
  const danger = it.tone === "danger";
  return (
    <button type="button" role="menuitem" disabled={it.disabled} onClick={onPick}
      onMouseEnter={() => setHov(true)} onMouseLeave={() => setHov(false)}
      style={{ display: "flex", alignItems: "center", gap: 9, width: "100%", textAlign: "left", padding: "7px 9px", background: hov && !it.disabled ? (danger ? "var(--danger-soft)" : "var(--surface-sunken)") : "transparent", border: "none", borderRadius: 8, cursor: it.disabled ? "default" : "pointer", opacity: it.disabled ? 0.45 : 1, color: danger ? "var(--danger)" : "var(--text-body)", font: "500 12.5px var(--font-ui)", transition: "background var(--dur-fast) var(--ease-out)" }}>
      {it.icon && <Icon name={it.icon} size={13} style={{ color: danger ? "var(--danger)" : "var(--text-muted)", flex: "none" }} />}
      <span style={{ flex: 1 }}>{it.label}</span>
      {it.shortcut && <span style={{ font: "400 11px var(--font-mono)", color: "var(--text-muted)" }}>{it.shortcut}</span>}
    </button>
  );
}

export function ContextMenu({ items = [], children, open: openProp, x: xProp, y: yProp, onClose }) {
  const controlled = children == null;
  const [st, setSt] = React.useState(null);
  const open = controlled ? !!openProp : !!st;
  const x = controlled ? xProp : st ? st.x : 0;
  const y = controlled ? yProp : st ? st.y : 0;
  const ref = React.useRef(null);
  const [pos, setPos] = React.useState(null);
  const close = React.useCallback(() => { if (controlled) { onClose && onClose(); } else setSt(null); }, [controlled, onClose]);
  React.useLayoutEffect(() => {
    if (!open) { setPos(null); return; }
    const el = ref.current;
    const w = el ? el.offsetWidth : 200, h = el ? el.offsetHeight : 120;
    setPos({ left: Math.max(8, Math.min(x, window.innerWidth - w - 8)), top: Math.max(8, Math.min(y, window.innerHeight - h - 8)) });
  }, [open, x, y, items.length]);
  React.useEffect(() => {
    if (!open) return;
    const onDoc = (e) => { if (!ref.current || !ref.current.contains(e.target)) close(); };
    const onKey = (e) => { if (e.key === "Escape") close(); };
    document.addEventListener("mousedown", onDoc);
    document.addEventListener("keydown", onKey);
    window.addEventListener("scroll", close, true);
    window.addEventListener("resize", close);
    return () => { document.removeEventListener("mousedown", onDoc); document.removeEventListener("keydown", onKey); window.removeEventListener("scroll", close, true); window.removeEventListener("resize", close); };
  }, [open, close]);
  const panel = open ? (
    <div ref={ref} role="menu" onContextMenu={(e) => e.preventDefault()}
      style={{ position: "fixed", left: pos ? pos.left : x, top: pos ? pos.top : y, visibility: pos ? "visible" : "hidden", minWidth: 190, background: "var(--surface)", border: "1px solid var(--border-default)", borderRadius: 12, boxShadow: "var(--shadow-md)", padding: 5, zIndex: 150, animation: "vg-pop-in var(--dur-base) var(--ease-out)", fontFamily: "var(--font-ui)" }}>
      {items.map((it, i) => it === "-"
        ? <div key={i} style={{ height: 1, background: "var(--row-sep)", margin: "5px 4px" }} />
        : <ContextItem key={i} it={it} onPick={() => { close(); it.onSelect && it.onSelect(); }} />)}
    </div>
  ) : null;
  if (controlled) return panel;
  return (
    <span style={{ display: "contents" }} onContextMenu={(e) => { e.preventDefault(); setSt({ x: e.clientX, y: e.clientY }); }}>
      {children}
      {panel}
    </span>
  );
}
