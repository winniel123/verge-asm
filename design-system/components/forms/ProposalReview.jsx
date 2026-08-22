import React from "react";
import { Button } from "./Button.jsx";
import { Checkbox } from "./Checkbox.jsx";
import { Tag } from "../display/Tag.jsx";

/* Registry proposals: confirm ONE at a time, decline MANY — the asymmetry is deliberate.
   proposals: [{id, value, kind, source}]. */
export function ProposalReview({ proposals = [], onConfirm, onDecline, style }) {
  const [marked, setMarked] = React.useState([]);
  const toggle = (id) => setMarked((m) => (m.indexOf(id) !== -1 ? m.filter((x) => x !== id) : m.concat(id)));
  return (
    <div style={{ display: "flex", flexDirection: "column", fontFamily: "var(--font-ui)", ...style }}>
      {proposals.map((p, i) => (
        <div key={p.id} style={{ display: "flex", alignItems: "center", gap: 12, padding: "10px 2px", borderTop: i ? "1px solid var(--row-sep)" : "none" }}>
          <span onClick={() => toggle(p.id)} style={{ display: "flex", cursor: "pointer" }}>
            <Checkbox checked={marked.indexOf(p.id) !== -1} onChange={() => {}} style={{ pointerEvents: "none" }} />
          </span>
          <span style={{ font: "500 12.5px var(--font-mono)", color: "var(--text-ink)" }}>{p.value}</span>
          <Tag>{p.kind}</Tag>
          <span style={{ font: "400 11.5px var(--font-ui)", color: "var(--text-muted)", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>{p.source}</span>
          <Button size="sm" variant="secondary" style={{ marginLeft: "auto", flex: "none" }} onClick={() => onConfirm && onConfirm(p)}>Confirm</Button>
        </div>
      ))}
      <div style={{ display: "flex", alignItems: "center", gap: 10, paddingTop: 12, borderTop: proposals.length ? "1px solid var(--row-sep)" : "none" }}>
        <span style={{ font: "400 11.5px var(--font-ui)", color: "var(--text-muted)" }}>Confirm one at a time; decline in bulk.</span>
        <Button size="sm" variant="ghost" style={{ marginLeft: "auto" }} disabled={!marked.length}
          onClick={() => { onDecline && onDecline(marked); setMarked([]); }}>
          Decline selected{marked.length ? " (" + marked.length + ")" : ""}
        </Button>
      </div>
    </div>
  );
}
