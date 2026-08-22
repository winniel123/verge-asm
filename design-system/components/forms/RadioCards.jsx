import React from "react";

/* Card-style exclusive choice for options that deserve a description. 2-4 options; more = Select. */
export function RadioCards({ options = [], value, onChange, columns, label, style }) {
  return (
    <div role="radiogroup" aria-label={label} style={{ display: "grid", gridTemplateColumns: columns ? "repeat(" + columns + ", 1fr)" : "repeat(auto-fit, minmax(180px, 1fr))", gap: 10, ...style }}>
      {options.map((o) => {
        const on = o.value === value;
        return (
          <button key={o.value} type="button" role="radio" aria-checked={on} disabled={o.disabled}
            onClick={() => onChange && onChange(o.value)}
            style={{ textAlign: "left", display: "flex", flexDirection: "column", gap: 6, padding: "12px 14px", background: on ? "var(--accent-soft)" : "var(--surface)", border: "1.5px solid " + (on ? "var(--accent)" : "var(--border-default)"), borderRadius: 14, cursor: o.disabled ? "default" : "pointer", opacity: o.disabled ? 0.45 : 1, fontFamily: "var(--font-ui)", transition: "border-color var(--dur-fast) var(--ease-out), background var(--dur-fast) var(--ease-out)" }}>
            <span style={{ display: "flex", alignItems: "center", gap: 8, minWidth: 0 }}>
              <span aria-hidden="true" style={{ width: 14, height: 14, borderRadius: 999, flex: "none", border: on ? "4.5px solid var(--accent)" : "1.5px solid var(--border-strong)", background: "var(--surface)", boxSizing: "border-box", transition: "border var(--dur-fast) var(--ease-out)" }} />
              <span style={{ font: "600 13px var(--font-ui)", color: "var(--text-ink)", whiteSpace: "nowrap" }}>{o.title}</span>
              {o.meta && <span style={{ marginLeft: "auto", font: "400 10.5px var(--font-mono)", color: "var(--text-muted)", flex: "none" }}>{o.meta}</span>}
            </span>
            {o.description && <span style={{ font: "400 12px/1.5 var(--font-ui)", color: "var(--text-secondary)" }}>{o.description}</span>}
          </button>
        );
      })}
    </div>
  );
}
