import React from "react";

export function Switch({ checked, onChange, label, disabled, style }) {
  const [foc, setFoc] = React.useState(false);
  return (
    <label style={{ display: "inline-flex", gap: 10, alignItems: "center", cursor: disabled ? "default" : "pointer", opacity: disabled ? 0.45 : 1, fontFamily: "var(--font-ui)", ...style }}>
      <input type="checkbox" role="switch" checked={!!checked} disabled={disabled} onChange={(e) => onChange && onChange(e.target.checked)}
        onFocus={() => setFoc(true)} onBlur={() => setFoc(false)}
        style={{ position: "absolute", opacity: 0, width: 1, height: 1 }} />
      <span aria-hidden="true" style={{ width: 36, height: 20, flex: "none", borderRadius: 999, position: "relative", background: checked ? "var(--accent)" : "var(--neutral-300)", boxShadow: foc ? "0 0 0 2px var(--surface), 0 0 0 4px var(--focus-ring)" : "none", transition: "background var(--dur-base) var(--ease-out)" }}>
        <span style={{ position: "absolute", top: 2, left: checked ? 18 : 2, width: 16, height: 16, borderRadius: 999, background: "#ffffff", boxShadow: "0 1px 2px rgba(35,31,25,0.2)", transition: "left var(--dur-base) var(--ease-out)" }} />
      </span>
      {label && <span style={{ fontSize: 13, color: "var(--text-body)" }}>{label}</span>}
    </label>
  );
}
