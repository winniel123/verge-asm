import React from "react";

export function Input({ label, hint, error, mono, size = "md", prefix, style, inputStyle, ...rest }) {
  const [focus, setFocus] = React.useState(false);
  const h = size === "sm" ? 26 : size === "lg" ? 40 : 32;
  return (
    <label style={{ display: "flex", flexDirection: "column", gap: 6, font: "500 12px var(--font-sans)", color: "var(--ink)", ...style }}>
      {label}
      <span style={{ display: "flex", alignItems: "center", height: h, background: "var(--surface)", border: `1px solid ${error ? "var(--danger)" : "var(--border-ink)"}`, boxShadow: focus ? "var(--focus-ring)" : "none", transition: "box-shadow var(--duration) var(--ease)" }}>
        {prefix && <span style={{ padding: "0 0 0 10px", font: "400 12px var(--font-mono)", color: "var(--text-faint)" }}>{prefix}</span>}
        <input
          onFocus={() => setFocus(true)}
          onBlur={() => setFocus(false)}
          style={{ flex: 1, minWidth: 0, height: "100%", padding: "0 10px", border: "none", outline: "none", background: "transparent", font: mono ? "400 12px var(--font-mono)" : "400 13px var(--font-sans)", color: "var(--text)", ...inputStyle }}
          {...rest}
        />
      </span>
      {error ? <span style={{ font: "400 11px var(--font-sans)", color: "var(--danger)" }}>{error}</span>
        : hint ? <span style={{ font: "400 11px var(--font-sans)", color: "var(--text-muted)" }}>{hint}</span> : null}
    </label>
  );
}
