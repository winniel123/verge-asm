import React from "react";

const SIZES = { sm: { h: 30, px: 12, fs: 12, r: 10, gap: 6 }, md: { h: 36, px: 16, fs: 13, r: 12, gap: 8 }, lg: { h: 44, px: 20, fs: 14, r: 12, gap: 8 } };

export function Button({ children, variant = "primary", size = "md", icon, disabled, onClick, type = "button", style, ...rest }) {
  const [hov, setHov] = React.useState(false);
  const [act, setAct] = React.useState(false);
  const [foc, setFoc] = React.useState(false);
  const s = SIZES[size] || SIZES.md;
  const v = {
    primary: { background: act ? "var(--accent-active)" : hov ? "var(--accent-hover)" : "var(--accent)", color: "var(--on-accent)", border: "1px solid transparent" },
    secondary: { background: hov ? "var(--surface-sunken)" : "var(--surface)", color: "var(--text-body)", border: "1px solid var(--border-strong)" },
    ghost: { background: hov ? "var(--surface-sunken)" : "transparent", color: hov ? "var(--text-body)" : "var(--text-secondary)", border: "1px solid transparent" },
    danger: { background: act ? "var(--danger-active)" : hov ? "var(--danger-hover)" : "var(--danger)", color: "var(--on-danger)", border: "1px solid transparent" },
  }[variant] || {};
  return (
    <button {...rest} type={type} disabled={disabled} onClick={onClick}
      onMouseEnter={() => setHov(true)} onMouseLeave={() => { setHov(false); setAct(false); }}
      onMouseDown={() => setAct(true)} onMouseUp={() => setAct(false)}
      onFocus={() => setFoc(true)} onBlur={() => setFoc(false)}
      style={{ display: "inline-flex", alignItems: "center", justifyContent: "center", gap: s.gap, height: s.h, padding: "0 " + s.px + "px", borderRadius: s.r, fontFamily: "var(--font-ui)", fontSize: s.fs, fontWeight: 600, cursor: disabled ? "default" : "pointer", opacity: disabled ? 0.45 : 1, transition: "background var(--dur-fast) var(--ease-out)", boxShadow: foc ? "0 0 0 2px var(--surface), 0 0 0 4px var(--focus-ring)" : "none", whiteSpace: "nowrap", ...v, ...style }}>
      {icon}{children}
    </button>
  );
}
