import React from "react";

/* Single-select typeahead. options: strings or {value, label, hint}. */
export function Combobox({ label, options = [], value, onChange, placeholder, style }) {
  const opts = options.map((o) => (typeof o === "string" ? { value: o, label: o } : o));
  const sel = opts.find((o) => o.value === value);
  const [open, setOpen] = React.useState(false);
  const [q, setQ] = React.useState("");
  const [idx, setIdx] = React.useState(0);
  const [foc, setFoc] = React.useState(false);
  const ref = React.useRef(null);
  const matches = opts.filter((o) => !q || (o.label + " " + (o.hint || "")).toLowerCase().includes(q.toLowerCase()));
  React.useEffect(() => {
    if (!open) return;
    const onDoc = (e) => { if (ref.current && !ref.current.contains(e.target)) setOpen(false); };
    document.addEventListener("mousedown", onDoc);
    return () => document.removeEventListener("mousedown", onDoc);
  }, [open]);
  const pick = (o) => { onChange && onChange(o.value); setOpen(false); setQ(""); };
  const onKey = (e) => {
    if (e.key === "ArrowDown") { e.preventDefault(); setOpen(true); setIdx((n) => (n + 1) % (matches.length || 1)); }
    else if (e.key === "ArrowUp") { e.preventDefault(); setIdx((n) => (n - 1 + (matches.length || 1)) % (matches.length || 1)); }
    else if (e.key === "Enter") { e.preventDefault(); if (open && matches[idx]) pick(matches[idx]); }
    else if (e.key === "Escape") setOpen(false);
  };
  return (
    <div ref={ref} style={{ display: "flex", flexDirection: "column", gap: 6, position: "relative", fontFamily: "var(--font-ui)", ...style }}>
      {label && <span style={{ fontSize: 12.5, fontWeight: 500, color: "var(--text-body)" }}>{label}</span>}
      <input value={open ? q : (sel ? sel.label : "")} placeholder={sel ? sel.label : placeholder}
        onChange={(e) => { setQ(e.target.value); setOpen(true); setIdx(0); }}
        onFocus={() => { setFoc(true); setOpen(true); setQ(""); }}
        onBlur={() => setFoc(false)} onKeyDown={onKey}
        style={{ height: 36, padding: "0 12px", background: "var(--surface)", color: "var(--text-ink)", font: "400 12.5px var(--font-mono)", border: "1px solid " + (foc ? "var(--accent)" : "var(--border-default)"), borderRadius: 12, outline: "none", boxShadow: foc ? "0 0 0 3px color-mix(in srgb, var(--focus-ring) 18%, transparent)" : "none" }} />
      {open && matches.length > 0 && (
        <div style={{ position: "absolute", top: "100%", left: 0, right: 0, marginTop: 6, zIndex: 95, background: "var(--surface-raised)", border: "1px solid var(--border-default)", borderRadius: 12, boxShadow: "var(--shadow-md)", padding: 5, animation: "vg-pop-in var(--dur-base) var(--ease-out)", maxHeight: 220, overflowY: "auto" }}>
          {matches.map((o, i) => (
            <button key={o.value} type="button" onMouseDown={(e) => { e.preventDefault(); pick(o); }} onMouseEnter={() => setIdx(i)}
              style={{ display: "flex", alignItems: "center", gap: 8, width: "100%", height: 30, padding: "0 10px", border: "none", borderRadius: 8, textAlign: "left", cursor: "pointer", background: i === idx ? "var(--surface-sunken)" : "transparent", font: "500 12px var(--font-mono)", color: o.value === value ? "var(--link)" : "var(--text-body)" }}>
              {o.label}
              {o.hint && <span style={{ marginLeft: "auto", font: "400 10.5px var(--font-mono)", color: "var(--text-muted)" }}>{o.hint}</span>}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
