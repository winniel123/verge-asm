import React from "react";
import { Button } from "./Button.jsx";
import { Select } from "./Select.jsx";
import { Tag } from "../display/Tag.jsx";

const KINDS = [{ value: "name", label: "Exact name" }, { value: "subtree", label: "Subtree" }, { value: "address", label: "Address scope" }];

export function ExclusionEditor({ exclusions = [], onAdd, onRemove, style }) {
  const [kind, setKind] = React.useState("subtree");
  const [draft, setDraft] = React.useState("");
  const [foc, setFoc] = React.useState(false);
  const [leaving, setLeaving] = React.useState(null);
  const add = () => { if (draft.trim() && onAdd) { onAdd(kind, draft.trim()); setDraft(""); } };
  const remove = (i, key) => {
    if (leaving) return;
    setLeaving(key);
    setTimeout(() => { setLeaving(null); onRemove && onRemove(i); }, 240);
  };
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 12, fontFamily: "var(--font-ui)", ...style }}>
      <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
        {exclusions.map((x, i) => {
          const key = x.kind + x.value;
          const isLeaving = leaving === key;
          return (
          <div key={key} style={{ display: "grid", gridTemplateRows: isLeaving ? "0fr" : "1fr", transition: "grid-template-rows var(--dur-base) var(--ease-out)" }}>
          <div style={{ overflow: "hidden", minHeight: 0 }}>
          <div style={{ display: "flex", alignItems: "center", gap: 10, padding: "7px 10px", background: "var(--surface-sunken)", borderRadius: 10, opacity: isLeaving ? 0 : 1, transition: "opacity var(--dur-base) var(--ease-out)", animation: "vg-fade-in var(--dur-base) var(--ease-out)" }}>
            <Tag>{x.kind}</Tag>
            <span style={{ font: "400 12.5px var(--font-mono)", color: "var(--text-body)", overflowWrap: "anywhere" }}>{x.kind === "subtree" ? "*." + x.value : x.value}</span>
            {onRemove && (
              <button type="button" aria-label="Remove exclusion" onClick={() => remove(i, key)}
                style={{ marginLeft: "auto", display: "inline-flex", alignItems: "center", justifyContent: "center", width: 20, height: 20, border: "none", borderRadius: 999, background: "transparent", color: "var(--text-muted)", cursor: "pointer" }}>
                <svg viewBox="0 0 10 10" width="9" height="9"><path d="M2 2l6 6M8 2l-6 6" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round"></path></svg>
              </button>
            )}
          </div>
          </div>
          </div>
        );})}
        {!exclusions.length && <span style={{ font: "400 12.5px var(--font-ui)", color: "var(--text-muted)" }}>No exclusions. Everything declared is in scope.</span>}
      </div>
      <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
        <Select size="sm" options={KINDS} value={kind} onChange={(e) => setKind(e.target.value)} style={{ width: 150 }} />
        <input value={draft} placeholder={kind === "address" ? "203.0.113.128/25" : "old-blog.acmecorp.io"}
          onChange={(e) => setDraft(e.target.value)} onKeyDown={(e) => { if (e.key === "Enter") add(); }}
          onFocus={() => setFoc(true)} onBlur={() => setFoc(false)}
          style={{ flex: 1, height: 30, padding: "0 10px", background: "var(--surface)", color: "var(--text-ink)", font: "400 12px var(--font-mono)", border: "1px solid " + (foc ? "var(--accent)" : "var(--border-default)"), borderRadius: 10, outline: "none", boxShadow: foc ? "0 0 0 3px color-mix(in srgb, var(--focus-ring) 18%, transparent)" : "none" }} />
        <Button size="sm" variant="secondary" disabled={!draft.trim()} onClick={add}>Add exclusion</Button>
      </div>
    </div>
  );
}
