import React from "react";
import { Icon } from "../media/Icon.jsx";

const SIZES = { sm: 26, md: 32, lg: 36 };

export function IconButton({ icon, label, variant = "ghost", size = "md", disabled, active, onClick, style }) {
  const [hov, setHov] = React.useState(false);
  const [foc, setFoc] = React.useState(false);
  const d = SIZES[size] || 32;
  const v = {
    ghost: { background: hov || active ? "var(--surface-sunken)" : "transparent", color: hov || active ? "var(--text-body)" : "var(--text-secondary)", border: "1px solid transparent" },
    secondary: { background: hov ? "var(--surface-sunken)" : "var(--surface)", color: "var(--text-body)", border: "1px solid var(--border-strong)" },
    primary: { background: hov ? "var(--accent-hover)" : "var(--accent)", color: "var(--on-accent)", border: "1px solid transparent" },
  }[variant] || {};
  return (
    <button type="button" aria-label={label} title={label} disabled={disabled} onClick={onClick}
      onMouseEnter={() => setHov(true)} onMouseLeave={() => setHov(false)}
      onFocus={() => setFoc(true)} onBlur={() => setFoc(false)}
      style={{ display: "inline-flex", alignItems: "center", justifyContent: "center", width: d, height: d, borderRadius: 10, cursor: disabled ? "default" : "pointer", opacity: disabled ? 0.45 : 1, transition: "background var(--dur-fast) var(--ease-out)", boxShadow: foc ? "0 0 0 2px var(--surface), 0 0 0 4px var(--focus-ring)" : "none", ...v, ...style }}>
      <Icon name={icon} size={size === "sm" ? 14 : 16} />
    </button>
  );
}
