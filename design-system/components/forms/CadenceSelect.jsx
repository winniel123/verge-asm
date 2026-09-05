import React from "react";
import { Select } from "./Select.jsx";

const PRESETS = ["Every 6h", "Daily \u00b7 08:00", "Weekly \u00b7 mon 09:00", "Monthly \u00b7 1st", "Custom\u2026"];

export function CadenceSelect({ label = "Cadence", value = PRESETS[0], customValue = "", onChange, style }) {
  const custom = value === PRESETS[4];
  const [foc, setFoc] = React.useState(false);
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 8, ...style }}>
      <Select label={label} options={PRESETS} value={value} onChange={(e) => onChange && onChange(e.target.value, customValue)} />
      {custom && (
        <input value={customValue} placeholder="0 8 * * 1" spellCheck={false}
          onChange={(e) => onChange && onChange(value, e.target.value)}
          onFocus={() => setFoc(true)} onBlur={() => setFoc(false)}
          style={{ height: 32, padding: "0 10px", background: "var(--surface)", color: "var(--text-ink)", font: "400 12px var(--font-mono)", border: "1px solid " + (foc ? "var(--accent)" : "var(--border-default)"), borderRadius: 10, outline: "none" }} />
      )}
    </div>
  );
}
