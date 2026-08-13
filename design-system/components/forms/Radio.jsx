import React from "react";

export function Radio({ label, checked, onChange, name, value, disabled, style }) {
  const [focus, setFocus] = React.useState(false);
  return (
    <label style={{ display: "inline-flex", alignItems: "center", gap: 8, cursor: disabled ? "default" : "pointer", opacity: disabled ? 0.45 : 1, font: "400 13px var(--font-sans)", color: "var(--text)", ...style }}>
      <input type="radio" name={name} value={value} checked={!!checked} disabled={disabled}
        onChange={() => onChange && onChange(value)}
        onFocus={() => setFocus(true)} onBlur={() => setFocus(false)}
        style={{ position: "absolute", opacity: 0, width: 1, height: 1 }} />
      <span aria-hidden="true" style={{
        width: 15, height: 15, flex: "none", boxSizing: "border-box", borderRadius: "50%",
        border: "1px solid var(--border-ink)", background: "var(--surface)",
        display: "inline-flex", alignItems: "center", justifyContent: "center",
        boxShadow: focus ? "var(--focus-ring)" : "none",
      }}>
        <span style={{ width: 7, height: 7, borderRadius: "50%", background: checked ? "var(--ink)" : "transparent", transition: "background var(--duration) var(--ease)" }} />
      </span>
      {label}
    </label>
  );
}
