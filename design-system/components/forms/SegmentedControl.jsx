import React from "react";

/* 2-4 way exclusive toggle on a sunken track. Short labels only; more options = Select. */
export function SegmentedControl({ options = [], value, onChange, size = "sm", label, style }) {
  const opts = options.map((o) => (typeof o === "string" ? { value: o, label: o } : o));
  const h = size === "sm" ? 22 : 28;
  return (
    <div role="radiogroup" aria-label={label} style={{ display: "inline-flex", gap: 2, padding: 2, background: "var(--surface-sunken)", border: "1px solid var(--border-default)", borderRadius: 10, ...style }}>
      {opts.map((o) => {
        const on = o.value === value;
        return (
          <button key={o.value} type="button" role="radio" aria-checked={on} onClick={() => onChange && onChange(o.value)}
            style={{ height: h, padding: "0 10px", display: "inline-flex", alignItems: "center", gap: 6, background: on ? "var(--surface)" : "transparent", color: on ? "var(--text-ink)" : "var(--text-secondary)", border: "none", borderRadius: 8, boxShadow: on ? "var(--shadow-xs)" : "none", cursor: "pointer", font: (on ? "600" : "500") + " 12px var(--font-ui)", whiteSpace: "nowrap", transition: "background var(--dur-fast) var(--ease-out), color var(--dur-fast) var(--ease-out)" }}>
            {o.icon}{o.label}
          </button>
        );
      })}
    </div>
  );
}
