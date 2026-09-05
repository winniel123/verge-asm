import React from "react";

export function NumberInput({ label, value = 0, onChange, min, max, step = 1, unit, hint, style }) {
  const [foc, setFoc] = React.useState(false);
  const clamp = (n) => Math.min(max != null ? max : Infinity, Math.max(min != null ? min : -Infinity, n));
  const set = (n) => { if (!isNaN(n) && onChange) onChange(clamp(n)); };
  const stepBtn = (dir, glyph) => (
    <button type="button" aria-label={dir > 0 ? "Increase" : "Decrease"} onClick={() => set((Number(value) || 0) + dir * step)}
      style={{ display: "flex", alignItems: "center", justifyContent: "center", width: 22, height: 16, border: "none", background: "transparent", color: "var(--text-muted)", cursor: "pointer", padding: 0 }}
      onMouseEnter={(e) => e.currentTarget.style.color = "var(--text-ink)"}
      onMouseLeave={(e) => e.currentTarget.style.color = "var(--text-muted)"}>
      <svg viewBox="0 0 12 8" width="9" height="6"><path d={glyph} fill="none" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round"></path></svg>
    </button>
  );
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 6, fontFamily: "var(--font-ui)", ...style }}>
      {label && <span style={{ fontSize: 12.5, fontWeight: 500, color: "var(--text-body)" }}>{label}</span>}
      <span style={{ display: "flex", alignItems: "center", gap: 8, height: 36, padding: "0 6px 0 12px", background: "var(--surface)", border: "1px solid " + (foc ? "var(--accent)" : "var(--border-default)"), borderRadius: 12, boxShadow: foc ? "0 0 0 3px color-mix(in srgb, var(--focus-ring) 18%, transparent)" : "none", width: "fit-content" }}>
        <input value={value} inputMode="numeric"
          onChange={(e) => { const n = Number(e.target.value); if (e.target.value === "" || !isNaN(n)) onChange && onChange(e.target.value === "" ? "" : n); }}
          onFocus={() => setFoc(true)} onBlur={(e) => { setFoc(false); set(Number(e.target.value) || (min != null ? min : 0)); }}
          style={{ width: 72, border: "none", outline: "none", background: "transparent", color: "var(--text-ink)", font: "400 13px var(--font-mono)", textAlign: "right" }} />
        {unit && <span style={{ font: "400 11.5px var(--font-mono)", color: "var(--text-muted)" }}>{unit}</span>}
        <span style={{ display: "flex", flexDirection: "column" }}>
          {stepBtn(1, "M2 6l4-4 4 4")}
          {stepBtn(-1, "M2 2l4 4 4-4")}
        </span>
      </span>
      {hint && <span style={{ fontSize: 11.5, color: "var(--text-muted)" }}>{hint}</span>}
    </div>
  );
}
