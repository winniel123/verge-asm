import React from "react";

export function Input({ label, hint, error, mono, prefix, size = "md", style, inputStyle, ...rest }) {
  const [foc, setFoc] = React.useState(false);
  const h = size === "sm" ? 30 : 36;
  const border = error ? "var(--danger-solid)" : foc ? "var(--accent)" : "var(--border-default)";
  const ring = foc ? "0 0 0 3px color-mix(in srgb, " + (error ? "var(--danger-solid)" : "var(--focus-ring)") + " 18%, transparent)" : "none";
  return (
    <label style={{ display: "flex", flexDirection: "column", gap: 6, fontFamily: "var(--font-ui)", ...style }}>
      {label && <span style={{ fontSize: 12.5, fontWeight: 500, color: "var(--text-body)" }}>{label}</span>}
      <span style={{ display: "flex", alignItems: "center", gap: 8, height: h, padding: "0 12px", background: "var(--surface)", border: "1px solid " + border, borderRadius: 12, boxShadow: ring, transition: "border-color var(--dur-fast) var(--ease-out), box-shadow var(--dur-fast) var(--ease-out)", opacity: rest.disabled ? 0.45 : 1 }}>
        {prefix && <span style={{ display: "inline-flex", color: "var(--text-muted)", flex: "none" }}>{prefix}</span>}
        <input {...rest}
          onFocus={(e) => { setFoc(true); rest.onFocus && rest.onFocus(e); }}
          onBlur={(e) => { setFoc(false); rest.onBlur && rest.onBlur(e); }}
          style={{ flex: 1, minWidth: 0, border: "none", outline: "none", background: "transparent", color: "var(--text-ink)", fontFamily: mono ? "var(--font-mono)" : "var(--font-ui)", fontSize: mono ? 12.5 : 13, ...inputStyle }} />
      </span>
      {error ? <span style={{ fontSize: 11.5, color: "var(--danger)" }}>{error}</span>
        : hint ? <span style={{ fontSize: 11.5, color: "var(--text-muted)" }}>{hint}</span> : null}
    </label>
  );
}
