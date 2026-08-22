import React from "react";
import { Input } from "./Input.jsx";
import { Select } from "./Select.jsx";
import { IconButton } from "./IconButton.jsx";
import { Button } from "./Button.jsx";
import { Icon } from "../media/Icon.jsx";

/* Two-sided attribute mapping: IdP claim (free mono text) -> app field (closed set).
   Duplicate targets are flagged, not auto-resolved. Controlled: mappings + onChange. */
export function MappingEditor({ mappings = [], onChange, fromLabel = "IdP claim", toLabel = "Verge field", fromPlaceholder = "e.g. email", toOptions = [], addLabel = "Add mapping", style }) {
  const set = (i, patch) => onChange(mappings.map((m, j) => (j === i ? { ...m, ...patch } : m)));
  const remove = (i) => onChange(mappings.filter((_, j) => j !== i));
  const add = () => onChange(mappings.concat({ from: "", to: "" }));
  const counts = {};
  mappings.forEach((m) => { if (m.to) counts[m.to] = (counts[m.to] || 0) + 1; });
  const dupes = Object.keys(counts).filter((k) => counts[k] > 1);
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 10, fontFamily: "var(--font-ui)", ...style }}>
      <div style={{ display: "grid", gridTemplateColumns: "minmax(0,1fr) 16px minmax(0,1fr) 28px", gap: 8, alignItems: "center" }}>
        <span style={{ font: "500 11px var(--font-mono)", letterSpacing: "0.06em", textTransform: "uppercase", color: "var(--text-muted)" }}>{fromLabel}</span>
        <span />
        <span style={{ font: "500 11px var(--font-mono)", letterSpacing: "0.06em", textTransform: "uppercase", color: "var(--text-muted)" }}>{toLabel}</span>
        <span />
        {mappings.map((m, i) => {
          return (
            <React.Fragment key={i}>
              <Input size="sm" mono placeholder={fromPlaceholder} value={m.from} onChange={(e) => set(i, { from: e.target.value })} aria-label={fromLabel + " " + (i + 1)} />
              <Icon name="arrow-right" size={13} style={{ color: "var(--text-muted)" }} />
              <Select size="sm" value={m.to} onChange={(e) => set(i, { to: e.target.value })} options={toOptions} placeholder="Choose field" aria-label={toLabel + " " + (i + 1)} />
              <IconButton icon="x" label={"Remove mapping " + (i + 1)} onClick={() => remove(i)} />
            </React.Fragment>
          );
        })}
      </div>
      {mappings.length === 0 && <span style={{ font: "400 12px var(--font-ui)", color: "var(--text-muted)" }}>No mappings — sign-ins will carry no attributes.</span>}
      {dupes.length > 0 && <span style={{ font: "400 12px var(--font-ui)", color: "var(--danger)" }}>Two claims map to {dupes.join(", ")} — the last assertion wins; remove one.</span>}
      <div><Button size="sm" variant="ghost" icon={<Icon name="plus" size={13} />} onClick={add}>{addLabel}</Button></div>
    </div>
  );
}
