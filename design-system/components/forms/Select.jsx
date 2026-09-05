import React from "react";

/* A native select popup cannot be styled, so the panel is a tokened listbox. */
export function Select({ label, options = [], value, defaultValue, onChange, size = "md", disabled, style }) {
  const opts = options.map((o) => (typeof o === "string" ? { value: o, label: o } : o));
  const [internal, setInternal] = React.useState(defaultValue !== undefined ? defaultValue : (opts[0] ? opts[0].value : ""));
  const val = value !== undefined ? value : internal;
  const sel = opts.find((o) => o.value === val);
  const [open, setOpen] = React.useState(false);
  const [idx, setIdx] = React.useState(Math.max(0, opts.findIndex((o) => o.value === val)));
  const [foc, setFoc] = React.useState(false);
  const [hov, setHov] = React.useState(-1);
  const ref = React.useRef(null);
  const [pos, setPos] = React.useState(null);
  const h = size === "sm" ? 30 : 36;
  React.useLayoutEffect(() => {
    if (!open || !ref.current) { setPos(null); return; }
    const measure = () => {
      if (!ref.current) return;
      const r = ref.current.getBoundingClientRect();
      const panelH = Math.min(244, opts.length * 30 + 12);
      const flip = r.bottom + 6 + panelH > window.innerHeight - 8 && r.top - 6 - panelH > 8;
      setPos(flip
        ? { left: r.left, width: r.width, bottom: window.innerHeight - r.top + 6 }
        : { left: r.left, width: r.width, top: r.bottom + 6 });
    };
    measure();
    // the anchor keeps moving after the first measure: webfont swap, or a panel animating in
    const t = setTimeout(measure, 120);
    return () => clearTimeout(t);
  }, [open, opts.length]);
  React.useEffect(() => {
    if (!open) return;
    const close = () => setOpen(false);
    window.addEventListener("scroll", close, true);
    window.addEventListener("resize", close);
    return () => { window.removeEventListener("scroll", close, true); window.removeEventListener("resize", close); };
  }, [open]);
  React.useEffect(() => {
    if (!open) return;
    const onDoc = (e) => { if (ref.current && !ref.current.contains(e.target)) setOpen(false); };
    const onKey = (e) => { if (e.key === "Escape") setOpen(false); };
    document.addEventListener("mousedown", onDoc);
    document.addEventListener("keydown", onKey);
    return () => { document.removeEventListener("mousedown", onDoc); document.removeEventListener("keydown", onKey); };
  }, [open]);
  const pick = (o) => {
    if (value === undefined) setInternal(o.value);
    if (onChange) onChange({ target: { value: o.value } });
    setOpen(false);
  };
  const onKeyDown = (e) => {
    if (disabled) return;
    if (e.key === "ArrowDown" || e.key === "ArrowUp") {
      e.preventDefault();
      if (!open) { setOpen(true); setIdx(Math.max(0, opts.findIndex((o) => o.value === val))); return; }
      const d = e.key === "ArrowDown" ? 1 : -1;
      setIdx((n) => (n + d + opts.length) % opts.length);
    } else if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      if (open && opts[idx]) pick(opts[idx]); else { setOpen(true); setIdx(Math.max(0, opts.findIndex((o) => o.value === val))); }
    }
  };
  return (
    <label style={{ display: "flex", flexDirection: "column", gap: 6, fontFamily: "var(--font-ui)", position: "relative", ...style }}>
      {label && <span style={{ fontSize: 12.5, fontWeight: 500, color: "var(--text-body)" }}>{label}</span>}
      <span ref={ref} style={{ position: "relative", display: "flex", opacity: disabled ? 0.45 : 1 }}>
        <button type="button" role="combobox" aria-expanded={open} aria-haspopup="listbox" disabled={disabled}
          onClick={() => setOpen(!open)} onKeyDown={onKeyDown}
          onFocus={() => setFoc(true)} onBlur={() => setFoc(false)}
          style={{ display: "flex", alignItems: "center", gap: 8, width: "100%", height: h, padding: "0 12px", background: "var(--surface)", color: "var(--text-ink)", fontFamily: "var(--font-ui)", fontSize: 13, textAlign: "left", border: "1px solid " + (open || foc ? "var(--accent)" : "var(--border-default)"), borderRadius: 12, outline: "none", boxShadow: foc || open ? "0 0 0 3px color-mix(in srgb, var(--focus-ring) 18%, transparent)" : "none", cursor: disabled ? "default" : "pointer", transition: "border-color var(--dur-fast) var(--ease-out), box-shadow var(--dur-fast) var(--ease-out)" }}>
          <span style={{ flex: 1, whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis" }}>{sel ? sel.label : ""}</span>
          <svg viewBox="0 0 16 16" width="14" height="14" style={{ color: "var(--text-muted)", flex: "none", transform: open ? "rotate(180deg)" : "none", transition: "transform var(--dur-base) var(--ease-out)" }}>
            <path d="M4 6l4 4 4-4" fill="none" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round"></path>
          </svg>
        </button>
        {open && pos && (
          <div role="listbox" style={{ position: "fixed", ...pos, zIndex: 140, background: "var(--surface-raised)", border: "1px solid var(--border-default)", borderRadius: 12, boxShadow: "var(--shadow-md)", padding: 5, maxHeight: 244, overflowY: "auto", animation: "vg-pop-in var(--dur-base) var(--ease-out)" }}>
            {opts.map((o, i) => {
              const active = o.value === val;
              const hot = i === idx || i === hov;
              return (
                <button key={o.value} type="button" role="option" aria-selected={active}
                  onClick={(e) => { e.stopPropagation(); pick(o); }}
                  onMouseEnter={() => { setHov(i); setIdx(i); }} onMouseLeave={() => setHov(-1)}
                  style={{ display: "flex", alignItems: "center", gap: 8, width: "100%", height: 30, padding: "0 10px", border: "none", borderRadius: 8, textAlign: "left", cursor: "pointer", background: hot ? "var(--surface-sunken)" : "transparent", color: active ? "var(--link)" : "var(--text-body)", font: (active ? "600" : "500") + " 12.5px var(--font-ui)", transition: "background var(--dur-fast) var(--ease-out)" }}>
                  <span style={{ flex: 1, whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis" }}>{o.label}</span>
                  {active && (
                    <svg viewBox="0 0 18 18" width="13" height="13" style={{ color: "var(--link)", flex: "none" }}><path d="M3.5 9.5l3.5 3.5 7.5-8" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"></path></svg>
                  )}
                </button>
              );
            })}
          </div>
        )}
      </span>
    </label>
  );
}
