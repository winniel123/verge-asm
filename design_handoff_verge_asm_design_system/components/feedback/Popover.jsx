import React from "react";

/* Non-modal floating panel anchored to its trigger. Closes on outside click / Escape. */
export function Popover({ trigger, open, onOpenChange, align = "start", side = "bottom", width = 260, children, style }) {
  const [internal, setInternal] = React.useState(false);
  const isOpen = open != null ? open : internal;
  const setOpen = (v) => { if (onOpenChange) onOpenChange(v); if (open == null) setInternal(v); };
  const ref = React.useRef(null);
  React.useEffect(() => {
    if (!isOpen) return;
    const onDoc = (e) => { if (ref.current && !ref.current.contains(e.target)) setOpen(false); };
    const onKey = (e) => { if (e.key === "Escape") setOpen(false); };
    document.addEventListener("mousedown", onDoc);
    document.addEventListener("keydown", onKey);
    return () => { document.removeEventListener("mousedown", onDoc); document.removeEventListener("keydown", onKey); };
  }, [isOpen]);
  const pos = { position: "absolute", zIndex: 95 };
  if (side === "bottom") { pos.top = "100%"; pos.marginTop = 6; } else { pos.bottom = "100%"; pos.marginBottom = 6; }
  if (align === "end") pos.right = 0; else pos.left = 0;
  return (
    <span ref={ref} style={{ position: "relative", display: "inline-flex", ...style }}>
      <span onClick={() => setOpen(!isOpen)} style={{ display: "inline-flex" }}>{trigger}</span>
      {isOpen && (
        <div style={{ ...pos, width, background: "var(--surface-raised)", border: "1px solid var(--border-default)", borderRadius: 12, boxShadow: "var(--shadow-md)", padding: 14, animation: "vg-pop-in var(--dur-base) var(--ease-out)", fontFamily: "var(--font-ui)" }}>
          {children}
        </div>
      )}
    </span>
  );
}
