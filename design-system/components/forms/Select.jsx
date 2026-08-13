import React from "react";

export function Select({ label, hint, size = "md", options = [], style, ...rest }) {
  const [focus, setFocus] = React.useState(false);
  const h = size === "sm" ? 26 : size === "lg" ? 40 : 32;
  return (
    <label style={{ display: "flex", flexDirection: "column", gap: 6, font: "500 12px var(--font-sans)", color: "var(--ink)", ...style }}>
      {label}
      <span style={{ position: "relative", display: "block" }}>
        <select
          onFocus={() => setFocus(true)}
          onBlur={() => setFocus(false)}
          style={{
            width: "100%", height: h, padding: "0 28px 0 10px",
            background: "var(--surface)", border: "1px solid var(--border-ink)", borderRadius: 0,
            font: size === "sm" ? "400 12px var(--font-sans)" : "400 13px var(--font-sans)", color: "var(--text)",
            appearance: "none", WebkitAppearance: "none", outline: "none", cursor: "pointer",
            boxShadow: focus ? "var(--focus-ring)" : "none",
          }}
          {...rest}
        >
          {options.map((o) => {
            const v = typeof o === "string" ? { value: o, label: o } : o;
            return <option key={v.value} value={v.value}>{v.label}</option>;
          })}
        </select>
        <span style={{ position: "absolute", right: 9, top: "50%", transform: "translateY(-52%)", pointerEvents: "none", font: "400 10px var(--font-mono)", color: "var(--text-muted)" }}>▾</span>
      </span>
      {hint && <span style={{ font: "400 11px var(--font-sans)", color: "var(--text-muted)" }}>{hint}</span>}
    </label>
  );
}
