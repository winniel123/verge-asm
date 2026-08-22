import React from "react";

/* MSP org context switcher — org scope, not per-user identity (ADR-0073). orgs: [{id, name, assets?}]. */
export function OrgSwitcher({ orgs = [], active, onChange, style }) {
  const [open, setOpen] = React.useState(false);
  const [hov, setHov] = React.useState(null);
  const ref = React.useRef(null);
  const current = orgs.find((o) => o.id === active) || orgs[0] || { name: "org" };
  React.useEffect(() => {
    if (!open) return;
    const onDoc = (e) => { if (ref.current && !ref.current.contains(e.target)) setOpen(false); };
    const onKey = (e) => { if (e.key === "Escape") setOpen(false); };
    document.addEventListener("mousedown", onDoc);
    document.addEventListener("keydown", onKey);
    return () => { document.removeEventListener("mousedown", onDoc); document.removeEventListener("keydown", onKey); };
  }, [open]);
  return (
    <span ref={ref} style={{ position: "relative", display: "inline-flex", ...style }}>
      <button type="button" onClick={() => setOpen(!open)}
        style={{ display: "inline-flex", alignItems: "center", gap: 6, height: 24, padding: "0 8px 0 10px", borderRadius: 999, background: "var(--surface-sunken)", border: "1px solid var(--border-default)", font: "500 11.5px var(--font-mono)", color: "var(--text-secondary)", cursor: "pointer", whiteSpace: "nowrap", flex: "none" }}>
        {current.name}
        <svg viewBox="0 0 16 16" width="11" height="11" style={{ color: "var(--text-muted)" }}><path d="M4 6l4 4 4-4" fill="none" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round"></path></svg>
      </button>
      {open && (
        <div role="menu" style={{ position: "absolute", zIndex: 95, top: "100%", left: 0, marginTop: 6, width: 224, background: "var(--surface-raised)", border: "1px solid var(--border-default)", borderRadius: 12, boxShadow: "var(--shadow-md)", padding: 5 }}>
          <div style={{ padding: "6px 10px 4px", font: "500 10.5px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--text-muted)" }}>Orgs</div>
          {orgs.map((o) => {
            const isActive = o.id === active;
            return (
              <button key={o.id} type="button" onClick={() => { setOpen(false); onChange && onChange(o.id); }}
                onMouseEnter={() => setHov(o.id)} onMouseLeave={() => setHov(null)}
                style={{ display: "flex", alignItems: "center", gap: 8, width: "100%", height: 32, padding: "0 10px", border: "none", borderRadius: 8, textAlign: "left", cursor: "pointer", background: hov === o.id ? "var(--surface-sunken)" : "transparent", transition: "background var(--dur-fast) var(--ease-out)" }}>
                <span style={{ font: (isActive ? "600" : "500") + " 12px var(--font-mono)", color: isActive ? "var(--text-ink)" : "var(--text-body)" }}>{o.name}</span>
                {o.assets != null && <span style={{ font: "400 10.5px var(--font-mono)", color: "var(--text-muted)" }}>{o.assets.toLocaleString("en-US")}</span>}
                {isActive && (
                  <svg viewBox="0 0 18 18" width="13" height="13" style={{ marginLeft: "auto", color: "var(--link)" }}><path d="M3.5 9.5l3.5 3.5 7.5-8" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"></path></svg>
                )}
              </button>
            );
          })}
        </div>
      )}
    </span>
  );
}
