import React from "react";

export function Switch({ label, checked, onChange, disabled, style }) {
  const [focus, setFocus] = React.useState(false);
  return (
    <label style={{ display: "inline-flex", alignItems: "center", gap: 8, cursor: disabled ? "default" : "pointer", opacity: disabled ? 0.45 : 1, font: "400 13px var(--font-sans)", color: "var(--text)", ...style }}>
      <input type="checkbox" role="switch" checked={!!checked} disabled={disabled}
        onChange={(e) => onChange && onChange(e.target.checked)}
        onFocus={() => setFocus(true)} onBlur={() => setFocus(false)}
        style={{ position: "absolute", opacity: 0, width: 1, height: 1 }} />
      <span aria-hidden="true" style={{
        width: 34, height: 18, flex: "none", boxSizing: "border-box", position: "relative",
        border: "1px solid var(--border-ink)",
        background: checked ? "var(--ink)" : "var(--surface)",
        boxShadow: focus ? "var(--focus-ring)" : "none",
        transition: "background var(--duration) var(--ease)",
      }}>
        <span style={{
          position: "absolute", top: 2, left: checked ? 18 : 2, width: 12, height: 12,
          background: checked ? "var(--paper)" : "var(--ink)",
          transition: "left var(--duration) var(--ease), background var(--duration) var(--ease)",
        }} />
      </span>
      {label}
    </label>
  );
}
