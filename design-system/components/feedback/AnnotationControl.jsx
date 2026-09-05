import React from "react";
import { Button } from "../forms/Button.jsx";
import { Textarea } from "../forms/Textarea.jsx";

export function AnnotationControl({ annotation, onAnnotate, onRemove, style }) {
  const [draft, setDraft] = React.useState("");
  if (annotation) {
    return (
      <div style={{ display: "flex", flexDirection: "column", gap: 8, padding: 14, background: "var(--surface-sunken)", borderRadius: 12, fontFamily: "var(--font-ui)", ...style }}>
        <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
          <span style={{ display: "inline-flex", alignItems: "center", gap: 6, height: 20, padding: "0 8px", borderRadius: 8, background: "var(--surface)", border: "1px solid var(--border-default)", color: "var(--text-secondary)", font: "600 10.5px var(--font-mono)", letterSpacing: "0.05em" }}>
            <span style={{ width: 6, height: 6, borderRadius: 999, background: "var(--neutral-500)" }} />
            accepted risk
          </span>
          {onRemove && <Button variant="ghost" size="sm" style={{ marginLeft: "auto" }} onClick={onRemove}>Remove annotation</Button>}
        </div>
        <p style={{ margin: 0, font: "400 13px/1.55 var(--font-ui)", color: "var(--text-body)" }}>{annotation.reason}</p>
      </div>
    );
  }
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 8, fontFamily: "var(--font-ui)", ...style }}>
      <span style={{ font: "500 11px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--text-muted)" }}>Annotation</span>
      <Textarea value={draft} placeholder="Why this risk is accepted" rows={3} onChange={(e) => setDraft(e.target.value)} />
      <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
        <Button size="sm" disabled={!draft.trim()} onClick={() => { onAnnotate && onAnnotate(draft.trim()); setDraft(""); }}>Accept this risk</Button>
        <span style={{ font: "400 11.5px var(--font-ui)", color: "var(--text-muted)" }}>Applies to this subject–signal pair. No expiry, no status, no author.</span>
      </div>
    </div>
  );
}
