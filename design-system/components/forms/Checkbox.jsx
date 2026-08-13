import React from "react";

export function Checkbox({ label, checked, onChange, disabled, style }) {
  const [focus, setFocus] = React.useState(false);
  return (
    <label style={{ display: "inline-flex", alignItems: "center", gap: 8, cursor: disabled ? "default" : "pointer", opacity: disabled ? 0.45 : 1, font: "400 13px var(--font-sans)", color: "var(--text)", ...style }}>
      <input type="checkbox" checked={!!checked} disabled={disabled}
        onChange={(e) => onChange && onChange(e.target.checked)}
        onFocus={() => setFocus(true)} onBlur={() => setFocus(false)}
        style={{ position: "absolute", opacity: 0, width: 1, height: 1 }} />
      <span aria-hidden="true" style={{
        width: 15, height: 15, flex: "none", boxSizing: "border-box",
        border: "1px solid var(--border-ink)",
        background: checked ? "var(--ink)" : "var(--surface)",
        color: "var(--text-on-ink)", display: "inline-flex", alignItems: "center", justifyContent: "center",
        font: "700 10px/1 var(--font-mono)",
        boxShadow: focus ? "var(--focus-ring)" : "none",
        transition: "background var(--duration) var(--ease)",
      }}>{checked ? "✓" : ""}</span>
      {label}
    </label>
  );
}
