import React from "react";

export function HoverCard({ content, delay = 350, side = "bottom", children, style }) {
  const [open, setOpen] = React.useState(false);
  const [pos, setPos] = React.useState(null);
  const ref = React.useRef(null);
  const panelRef = React.useRef(null);
  const timers = React.useRef({});
  const show = () => { clearTimeout(timers.current.hide); timers.current.show = setTimeout(() => setOpen(true), delay); };
  const hide = () => { clearTimeout(timers.current.show); timers.current.hide = setTimeout(() => setOpen(false), 150); };
  React.useEffect(() => () => { clearTimeout(timers.current.show); clearTimeout(timers.current.hide); }, []);
  React.useLayoutEffect(() => {
    if (!open || !ref.current) { setPos(null); return; }
    const r = ref.current.getBoundingClientRect();
    const el = panelRef.current;
    const w = el ? el.offsetWidth : 260, h = el ? el.offsetHeight : 120;
    let top = side === "top" ? r.top - h - 8 : r.bottom + 8;
    if (top + h > window.innerHeight - 8) top = r.top - h - 8;
    if (top < 8) top = r.bottom + 8;
    const left = Math.max(8, Math.min(r.left, window.innerWidth - w - 8));
    setPos({ top, left });
  }, [open, side]);
  React.useEffect(() => {
    if (!open) return;
    const close = () => setOpen(false);
    window.addEventListener("scroll", close, true);
    return () => window.removeEventListener("scroll", close, true);
  }, [open]);
  return (
    <span ref={ref} onMouseEnter={show} onMouseLeave={hide} style={{ display: "inline-flex", minWidth: 0, ...style }}>
      {children}
      {open && (
        <div ref={panelRef} onMouseEnter={show} onMouseLeave={hide}
          style={{ position: "fixed", top: pos ? pos.top : -9999, left: pos ? pos.left : -9999, visibility: pos ? "visible" : "hidden", zIndex: 130, background: "var(--surface)", border: "1px solid var(--border-default)", borderRadius: 14, boxShadow: "var(--shadow-md)", padding: 14, minWidth: 220, maxWidth: 320, animation: "vg-fade-in var(--dur-base) var(--ease-out)", fontFamily: "var(--font-ui)" }}>
          {content}
        </div>
      )}
    </span>
  );
}
