import React from "react";

/* Label/hint/error plumbing around any control. Give the control the same id. */
export function FormField({ id, label, hint, error, required, children, style }) {
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 6, fontFamily: "var(--font-ui)", ...style }}>
      {label && (
        <label htmlFor={id} style={{ font: "500 12.5px var(--font-ui)", color: "var(--text-body)" }}>
          {label}{required && <span style={{ color: "var(--danger)" }}> *</span>}
        </label>
      )}
      {children}
      {error ? <span style={{ fontSize: 11.5, color: "var(--danger)" }}>{error}</span>
        : hint ? <span style={{ fontSize: 11.5, color: "var(--text-muted)" }}>{hint}</span> : null}
    </div>
  );
}

/* Top-of-form error list. errors: [{label, message?, fieldId?}] — fieldId links focus to the control. */
export function FormErrorSummary({ errors = [], title, style }) {
  if (!errors.length) return null;
  return (
    <div role="alert" style={{ display: "flex", flexDirection: "column", gap: 8, padding: "12px 14px", background: "var(--danger-soft)", border: "1px solid var(--danger-border)", borderRadius: 12, fontFamily: "var(--font-ui)", ...style }}>
      <span style={{ font: "600 12.5px var(--font-ui)", color: "var(--danger)" }}>
        {title || (errors.length === 1 ? "1 field needs attention" : errors.length + " fields need attention")}
      </span>
      <ul style={{ margin: 0, padding: "0 0 0 16px", display: "flex", flexDirection: "column", gap: 4 }}>
        {errors.map((e, i) => (
          <li key={i} style={{ font: "400 12.5px var(--font-ui)", color: "var(--danger)" }}>
            {e.fieldId ? (
              <a href={"#" + e.fieldId} onClick={(ev) => { ev.preventDefault(); const el = document.getElementById(e.fieldId); if (el) el.focus(); }}
                style={{ color: "inherit", textDecoration: "underline" }}>{e.label}</a>
            ) : <span style={{ fontWeight: 500 }}>{e.label}</span>}
            {e.message ? " \u2014 " + e.message : ""}
          </li>
        ))}
      </ul>
    </div>
  );
}
