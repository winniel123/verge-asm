import React from "react";
import { Button } from "./Button.jsx";

export function SecretInput({ label = "Signing secret", isSet, onSave, hint, style }) {
  const [editing, setEditing] = React.useState(!isSet);
  const [draft, setDraft] = React.useState("");
  const [foc, setFoc] = React.useState(false);
  React.useEffect(() => { setEditing(!isSet); }, [isSet]);
  const save = () => { if (draft && onSave) onSave(draft); setDraft(""); setEditing(false); };
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 6, fontFamily: "var(--font-ui)", ...style }}>
      <span style={{ fontSize: 12.5, fontWeight: 500, color: "var(--text-body)" }}>{label}</span>
      {editing ? (
        <div style={{ display: "flex", gap: 8 }}>
          <input type="password" value={draft} autoComplete="new-password" placeholder="paste a secret"
            onChange={(e) => setDraft(e.target.value)}
            onFocus={() => setFoc(true)} onBlur={() => setFoc(false)}
            style={{ flex: 1, height: 36, padding: "0 12px", background: "var(--surface)", color: "var(--text-ink)", font: "400 12.5px var(--font-mono)", border: "1px solid " + (foc ? "var(--accent)" : "var(--border-default)"), borderRadius: 12, outline: "none", boxShadow: foc ? "0 0 0 3px color-mix(in srgb, var(--focus-ring) 18%, transparent)" : "none" }} />
          <Button size="md" variant="secondary" disabled={!draft} onClick={save}>Save secret</Button>
          {isSet && <Button size="md" variant="ghost" onClick={() => { setEditing(false); setDraft(""); }}>Cancel</Button>}
        </div>
      ) : (
        <div style={{ display: "flex", alignItems: "center", gap: 10, height: 36, padding: "0 12px", background: "var(--surface-sunken)", borderRadius: 12 }}>
          <span style={{ font: "600 13px var(--font-mono)", color: "var(--text-muted)", letterSpacing: "0.15em" }}>••••••••••</span>
          <span style={{ font: "400 11.5px var(--font-ui)", color: "var(--text-muted)" }}>set — write-only, never rendered back</span>
          <Button size="sm" variant="ghost" style={{ marginLeft: "auto" }} onClick={() => setEditing(true)}>Replace</Button>
        </div>
      )}
      {hint && <span style={{ fontSize: 11.5, color: "var(--text-muted)" }}>{hint}</span>}
    </div>
  );
}
