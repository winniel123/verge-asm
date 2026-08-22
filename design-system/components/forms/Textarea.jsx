import React from "react";

/* Multi-line Input sibling — same tokens and focus treatment. autoGrow tracks content height. */
export function Textarea({ label, hint, error, mono, rows = 3, autoGrow, style, inputStyle, ...rest }) {
  const [foc, setFoc] = React.useState(false);
  const ref = React.useRef(null);
  const grow = () => { if (autoGrow && ref.current) { ref.current.style.height = "auto"; ref.current.style.height = ref.current.scrollHeight + 2 + "px"; } };
  React.useEffect(grow, []);
  const border = error ? "var(--danger-solid)" : foc ? "var(--accent)" : "var(--border-default)";
  const ring = foc ? "0 0 0 3px color-mix(in srgb, " + (error ? "var(--danger-solid)" : "var(--focus-ring)") + " 18%, transparent)" : "none";
  return (
    <label style={{ display: "flex", flexDirection: "column", gap: 6, fontFamily: "var(--font-ui)", ...style }}>
      {label && <span style={{ fontSize: 12.5, fontWeight: 500, color: "var(--text-body)" }}>{label}</span>}
      <textarea ref={ref} rows={rows} {...rest}
        onFocus={(e) => { setFoc(true); rest.onFocus && rest.onFocus(e); }}
        onBlur={(e) => { setFoc(false); rest.onBlur && rest.onBlur(e); }}
        onInput={(e) => { grow(); rest.onInput && rest.onInput(e); }}
        style={{ resize: autoGrow ? "none" : "vertical", minHeight: 64, padding: "9px 12px", background: "var(--surface)", color: "var(--text-ink)", font: "400 13px/1.5 " + (mono ? "var(--font-mono)" : "var(--font-ui)"), border: "1px solid " + border, borderRadius: 12, outline: "none", boxShadow: ring, transition: "border-color var(--dur-fast) var(--ease-out), box-shadow var(--dur-fast) var(--ease-out)", opacity: rest.disabled ? 0.45 : 1, ...inputStyle }} />
      {error ? <span style={{ fontSize: 11.5, color: "var(--danger)" }}>{error}</span>
        : hint ? <span style={{ fontSize: 11.5, color: "var(--text-muted)" }}>{hint}</span> : null}
    </label>
  );
}
