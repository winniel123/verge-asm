import React from "react";

const H = { sm: 26, md: 32, lg: 40 };

export function IconButton({ variant = "ghost", size = "md", label, disabled, style, children, ...rest }) {
  const [hover, setHover] = React.useState(false);
  const [focus, setFocus] = React.useState(false);
  const variants = {
    secondary: { background: hover && !disabled ? "var(--surface-sunken)" : "var(--surface)", border: "1px solid var(--border-ink)" },
    ghost: { background: hover && !disabled ? "var(--surface-sunken)" : "transparent", border: "1px solid transparent" },
  };
  return (
    <button
      type="button"
      aria-label={label}
      title={label}
      disabled={disabled}
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => setHover(false)}
      onFocus={() => setFocus(true)}
      onBlur={() => setFocus(false)}
      style={{
        display: "inline-flex", alignItems: "center", justifyContent: "center",
        width: H[size], height: H[size], flex: "none",
        color: "var(--ink)", borderRadius: 0,
        cursor: disabled ? "default" : "pointer", opacity: disabled ? 0.45 : 1,
        boxShadow: focus ? "var(--focus-ring)" : "none", outline: "none",
        transition: "background var(--duration) var(--ease)",
        ...variants[variant], ...style,
      }}
      {...rest}
    >
      {children}
    </button>
  );
}
