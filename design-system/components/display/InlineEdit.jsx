import React from "react";
import { Icon } from "../media/Icon.jsx";

export function InlineEdit({ value, onChange, mono, placeholder = "\u2014", label, style }) {
  const [editing, setEditing] = React.useState(false);
  const [draft, setDraft] = React.useState(value || "");
  const [hov, setHov] = React.useState(false);
  const ref = React.useRef(null);
  React.useEffect(() => { if (editing && ref.current) { ref.current.focus(); ref.current.select(); } }, [editing]);
  const font = mono ? "500 12.5px var(--font-mono)" : "400 13px var(--font-ui)";
  const commit = () => { setEditing(false); const v = draft.trim(); if (v && v !== value && onChange) onChange(v); else setDraft(value || ""); };
  if (editing) {
    return (
      <input ref={ref} value={draft} aria-label={label} onChange={(e) => setDraft(e.target.value)} onBlur={commit}
        onKeyDown={(e) => { if (e.key === "Enter") commit(); else if (e.key === "Escape") { setEditing(false); setDraft(value || ""); } }}
        style={{ font, color: "var(--text-ink)", background: "var(--surface)", border: "1px solid var(--accent)", borderRadius: 8, padding: "3px 8px", outline: "none", boxShadow: "0 0 0 3px color-mix(in srgb, var(--focus-ring) 18%, transparent)", minWidth: 120, maxWidth: 320, ...style }} />
    );
  }
  return (
    <button type="button" title="Click to edit" aria-label={label ? "Edit " + label : "Edit"} onClick={() => { setDraft(value || ""); setEditing(true); }}
      onMouseEnter={() => setHov(true)} onMouseLeave={() => setHov(false)}
      style={{ display: "inline-flex", alignItems: "center", gap: 6, background: hov ? "var(--surface-sunken)" : "transparent", border: "1px dashed " + (hov ? "var(--border-strong)" : "transparent"), borderRadius: 8, padding: "3px 8px", cursor: "text", font, color: value ? "var(--text-ink)" : "var(--text-muted)", transition: "background var(--dur-fast) var(--ease-out), border-color var(--dur-fast) var(--ease-out)", ...style }}>
      {value || placeholder}
      <Icon name="pencil" size={11} style={{ color: "var(--text-muted)", opacity: hov ? 1 : 0, transition: "opacity var(--dur-fast) var(--ease-out)" }} />
    </button>
  );
}
