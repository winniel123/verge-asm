import React from "react";

export function Checkbox({ label, description, checked, indeterminate, onChange, disabled, style }) {
  const [foc, setFoc] = React.useState(false);
  return (
    <label style={{ display: "flex", gap: 10, alignItems: "flex-start", cursor: disabled ? "default" : "pointer", opacity: disabled ? 0.45 : 1, fontFamily: "var(--font-ui)", ...style }}>
      <input type="checkbox" checked={!!checked} disabled={disabled} onChange={(e) => onChange && onChange(e.target.checked)}
        onFocus={() => setFoc(true)} onBlur={() => setFoc(false)}
        style={{ position: "absolute", opacity: 0, width: 1, height: 1 }} />
      <span aria-hidden="true" style={{ width: 18, height: 18, flex: "none", marginTop: 1, borderRadius: 6, display: "inline-flex", alignItems: "center", justifyContent: "center", background: checked || indeterminate ? "var(--accent)" : "var(--surface)", border: "1px solid " + (checked || indeterminate ? "var(--accent)" : "var(--border-strong)"), boxShadow: foc ? "0 0 0 2px var(--surface), 0 0 0 4px var(--focus-ring)" : "none", transition: "background var(--dur-fast) var(--ease-out)" }}>
        {indeterminate ? <svg viewBox="0 0 18 18" width="12" height="12"><path d="M4 9h10" fill="none" stroke="var(--on-accent)" strokeWidth="2.2" strokeLinecap="round"></path></svg> : checked ? <svg viewBox="0 0 18 18" width="12" height="12"><path d="M3.5 9.5l3.5 3.5 7.5-8" fill="none" stroke="var(--on-accent)" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round"></path></svg> : null}
      </span>
      <span style={{ display: "flex", flexDirection: "column", gap: 2 }}>
        {label && <span style={{ fontSize: 13, color: "var(--text-body)", lineHeight: 1.4 }}>{label}</span>}
        {description && <span style={{ fontSize: 11.5, color: "var(--text-muted)", lineHeight: 1.45 }}>{description}</span>}
      </span>
    </label>
  );
}
