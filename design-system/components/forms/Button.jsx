import React from "react";

const H = { sm: 26, md: 32, lg: 40 };
const PX = { sm: 10, md: 12, lg: 16 };
const FS = { sm: 12, md: 13, lg: 14 };

export function Button({ variant = "primary", size = "md", icon, disabled, style, children, ...rest }) {
  const [hover, setHover] = React.useState(false);
  const [focus, setFocus] = React.useState(false);
  const variants = {
    primary: { background: hover && !disabled ? "#3a3a30" : "var(--ink)", color: "var(--text-on-ink)", border: "1px solid var(--ink)" },
    secondary: { background: hover && !disabled ? "var(--surface-sunken)" : "var(--surface)", color: "var(--ink)", border: "1px solid var(--border-ink)" },
    ghost: { background: hover && !disabled ? "var(--surface-sunken)" : "transparent", color: "var(--ink)", border: "1px solid transparent" },
    danger: { background: hover && !disabled ? "#b02525" : "var(--danger)", color: "#ffffff", border: "1px solid var(--danger)" },
  };
  return (
    <button
      type="button"
      disabled={disabled}
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => setHover(false)}
      onFocus={() => setFocus(true)}
      onBlur={() => setFocus(false)}
      style={{
        display: "inline-flex", alignItems: "center", justifyContent: "center", gap: 8,
        height: H[size], padding: `0 ${PX[size]}px`,
        font: `500 ${FS[size]}px var(--font-sans)`,
        borderRadius: 0, cursor: disabled ? "default" : "pointer",
        opacity: disabled ? 0.45 : 1,
        boxShadow: focus ? "var(--focus-ring)" : "none",
        outline: "none", transition: "background var(--duration) var(--ease)",
        ...variants[variant], ...style,
      }}
      {...rest}
    >
      {icon}{children}
    </button>
  );
}
