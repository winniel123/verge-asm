import React from "react";

export function Radio({ label, checked, onChange, name, disabled, style }) {
  const [foc, setFoc] = React.useState(false);
  return (
    <label style={{ display: "flex", gap: 10, alignItems: "center", cursor: disabled ? "default" : "pointer", opacity: disabled ? 0.45 : 1, fontFamily: "var(--font-ui)", ...style }}>
      <input type="radio" name={name} checked={!!checked} disabled={disabled} onChange={() => onChange && onChange()}
        onFocus={() => setFoc(true)} onBlur={() => setFoc(false)}
        style={{ position: "absolute", opacity: 0, width: 1, height: 1 }} />
      <span aria-hidden="true" style={{ width: 18, height: 18, flex: "none", borderRadius: 999, display: "inline-flex", alignItems: "center", justifyContent: "center", background: "var(--surface)", border: "1px solid " + (checked ? "var(--accent)" : "var(--border-strong)"), boxShadow: foc ? "0 0 0 2px var(--surface), 0 0 0 4px var(--focus-ring)" : "none", transition: "border-color var(--dur-fast) var(--ease-out)" }}>
        {checked && <span style={{ width: 8, height: 8, borderRadius: 999, background: "var(--accent)" }} />}
      </span>
      {label && <span style={{ fontSize: 13, color: "var(--text-body)" }}>{label}</span>}
    </label>
  );
}
