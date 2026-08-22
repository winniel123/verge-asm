import React from "react";
import { Tag } from "../display/Tag.jsx";

/* Multi-value filter input: type, Enter/comma commits a mono tag; Backspace removes the last. */
export function TagInput({ label, hint, values = [], onChange, suggestions = [], placeholder, validate, style }) {
  const [draft, setDraft] = React.useState("");
  const [foc, setFoc] = React.useState(false);
  const [error, setError] = React.useState(null);
  const inputRef = React.useRef(null);
  const commit = (raw) => {
    const v = (raw || "").trim().replace(/,$/, "");
    if (!v) return;
    if (validate) { const err = validate(v); if (err) { setError(err); return; } }
    setError(null);
    if (values.indexOf(v) === -1 && onChange) onChange(values.concat(v));
    setDraft("");
  };
  const remove = (v) => onChange && onChange(values.filter((x) => x !== v));
  const onKey = (e) => {
    if (e.key === "Enter" || e.key === ",") { e.preventDefault(); commit(draft); }
    else if (e.key === "Backspace" && !draft && values.length) remove(values[values.length - 1]);
  };
  const open = foc && draft && suggestions.filter((s) => s.toLowerCase().includes(draft.toLowerCase()) && values.indexOf(s) === -1).length > 0;
  const matches = suggestions.filter((s) => s.toLowerCase().includes(draft.toLowerCase()) && values.indexOf(s) === -1).slice(0, 6);
  return (
    <label style={{ display: "flex", flexDirection: "column", gap: 6, fontFamily: "var(--font-ui)", position: "relative", ...style }}>
      {label && <span style={{ fontSize: 12.5, fontWeight: 500, color: "var(--text-body)" }}>{label}</span>}
      <span onClick={() => inputRef.current && inputRef.current.focus()}
        style={{ display: "flex", alignItems: "center", flexWrap: "wrap", gap: 6, minHeight: 36, padding: "5px 10px", background: "var(--surface)", border: "1px solid " + (error ? "var(--danger-solid)" : foc ? "var(--accent)" : "var(--border-default)"), borderRadius: 12, boxShadow: foc ? "0 0 0 3px color-mix(in srgb, var(--focus-ring) 18%, transparent)" : "none", transition: "border-color var(--dur-fast) var(--ease-out), box-shadow var(--dur-fast) var(--ease-out)", cursor: "text" }}>
        {values.map((v) => <Tag key={v} onRemove={() => remove(v)}>{v}</Tag>)}
        <input ref={inputRef} value={draft} placeholder={values.length ? "" : placeholder}
          onChange={(e) => { setDraft(e.target.value); setError(null); }} onKeyDown={onKey}
          onFocus={() => setFoc(true)} onBlur={() => setTimeout(() => setFoc(false), 120)}
          style={{ flex: 1, minWidth: 80, border: "none", outline: "none", background: "transparent", color: "var(--text-ink)", font: "400 12.5px var(--font-mono)" }} />
      </span>
      {open && (
        <div style={{ position: "absolute", top: "100%", left: 0, marginTop: 6, zIndex: 95, width: "100%", background: "var(--surface-raised)", border: "1px solid var(--border-default)", borderRadius: 12, boxShadow: "var(--shadow-md)", padding: 5, animation: "vg-pop-in var(--dur-base) var(--ease-out)" }}>
          {matches.map((s) => (
            <button key={s} type="button" onMouseDown={(e) => { e.preventDefault(); commit(s); }}
              style={{ display: "block", width: "100%", textAlign: "left", padding: "6px 10px", border: "none", borderRadius: 8, background: "transparent", color: "var(--text-body)", font: "400 12px var(--font-mono)", cursor: "pointer" }}
              onMouseEnter={(e) => e.currentTarget.style.background = "var(--surface-sunken)"}
              onMouseLeave={(e) => e.currentTarget.style.background = "transparent"}>{s}</button>
          ))}
        </div>
      )}
      {error ? <span style={{ fontSize: 11.5, color: "var(--danger)" }}>{error}</span>
        : hint ? <span style={{ fontSize: 11.5, color: "var(--text-muted)" }}>{hint}</span> : null}
    </label>
  );
}
