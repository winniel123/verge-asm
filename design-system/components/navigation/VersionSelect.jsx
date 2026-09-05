import React from "react";
import { Icon } from "../media/Icon.jsx";

export function VersionSelect({ versions = [], value, onChange, style }) {
  const [open, setOpen] = React.useState(false);
  const [active, setActive] = React.useState(-1);
  const [hov, setHov] = React.useState(false);
  const ref = React.useRef(null);
  React.useEffect(() => {
    if (!open) return;
    const onDoc = (e) => { if (ref.current && !ref.current.contains(e.target)) setOpen(false); };
    const onKey = (e) => {
      if (e.key === "Escape") setOpen(false);
      else if (e.key === "ArrowDown") { e.preventDefault(); setActive((a) => Math.min(versions.length - 1, a + 1)); }
      else if (e.key === "ArrowUp") { e.preventDefault(); setActive((a) => Math.max(0, a - 1)); }
      else if (e.key === "Enter" && active >= 0) { e.preventDefault(); onChange && onChange(versions[active].value); setOpen(false); }
    };
    document.addEventListener("mousedown", onDoc);
    document.addEventListener("keydown", onKey);
    return () => { document.removeEventListener("mousedown", onDoc); document.removeEventListener("keydown", onKey); };
  }, [open, active, versions, onChange]);
  const cur = versions.filter((v) => v.value === value)[0] || versions[0] || {};
  return (
    <span ref={ref} style={{ position: "relative", display: "inline-flex", ...style }}>
      <button type="button" aria-haspopup="listbox" aria-expanded={open} onClick={() => { setOpen(!open); setActive(-1); }} onMouseEnter={() => setHov(true)} onMouseLeave={() => setHov(false)}
        style={{ display: "inline-flex", alignItems: "center", gap: 5, padding: "3px 7px 3px 9px", background: open || hov ? "var(--surface-sunken)" : "transparent", border: "1px solid var(--border-default)", borderRadius: 8, cursor: "pointer", font: "500 11px var(--font-mono)", color: "var(--text-secondary)", transition: "background var(--dur-fast) var(--ease-out)" }}>
        {cur.value}
        <Icon name="chevron-down" size={11} style={{ transition: "transform var(--dur-base) var(--ease-out)", transform: open ? "rotate(180deg)" : "none" }} />
      </button>
      {open && (
        <div role="listbox" style={{ position: "absolute", top: "calc(100% + 6px)", right: 0, minWidth: 168, background: "var(--surface)", border: "1px solid var(--border-default)", borderRadius: 12, boxShadow: "var(--shadow-md)", padding: 5, zIndex: 120, animation: "vg-pop-in var(--dur-base) var(--ease-out)" }}>
          {versions.map((v, vi) => {
            const sel = v.value === value;
            return (
              <button key={v.value} type="button" role="option" aria-selected={sel}
                onClick={() => { onChange && onChange(v.value); setOpen(false); }}
                onMouseEnter={() => setActive(vi)}
                style={{ display: "flex", alignItems: "center", gap: 8, width: "100%", textAlign: "left", padding: "6px 8px", background: active === vi ? "var(--surface-sunken)" : "transparent", border: "none", borderRadius: 8, cursor: "pointer", font: "500 11.5px var(--font-mono)", color: "var(--text-body)" }}>
                {v.value}
                {v.tag && <span style={{ font: "500 9.5px var(--font-mono)", letterSpacing: "0.06em", textTransform: "uppercase", padding: "1.5px 6px", borderRadius: 999, background: v.tag === "current" ? "var(--accent-soft)" : "var(--surface-sunken)", color: v.tag === "current" ? "var(--accent)" : "var(--text-muted)", border: v.tag === "current" ? "none" : "1px solid var(--border-default)" }}>{v.tag}</span>}
                {sel && <Icon name="check" size={12} style={{ marginLeft: "auto", color: "var(--accent)" }} />}
              </button>
            );
          })}
        </div>
      )}
    </span>
  );
}
