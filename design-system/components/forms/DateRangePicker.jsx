import React from "react";
import { Popover } from "../feedback/Popover.jsx";
import { Button } from "./Button.jsx";
import { Icon } from "../media/Icon.jsx";

const PRESETS = ["Last 24h", "Last 7d", "Last 30d", "Last 90d"];

function PresetRow({ label, active, onClick }) {
  const [hov, setHov] = React.useState(false);
  return (
    <button type="button" onClick={onClick} onMouseEnter={() => setHov(true)} onMouseLeave={() => setHov(false)}
      style={{ display: "flex", alignItems: "center", gap: 8, width: "100%", height: 30, padding: "0 10px", border: "none", borderRadius: 8, textAlign: "left", cursor: "pointer", background: active ? "var(--accent-soft)" : hov ? "var(--surface-sunken)" : "transparent", color: active ? "var(--link)" : "var(--text-body)", font: (active ? "600" : "500") + " 12.5px var(--font-ui)", transition: "background var(--dur-fast) var(--ease-out)" }}>
      {label}
      {active && <span style={{ marginLeft: "auto", display: "inline-flex", color: "var(--link)" }}><Icon name="check" size={13} /></span>}
    </button>
  );
}

export function DateRangePicker({ value, onChange, presets = PRESETS, align = "end", style }) {
  const [open, setOpen] = React.useState(false);
  const [start, setStart] = React.useState((value && value.start) || "");
  const [end, setEnd] = React.useState((value && value.end) || "");
  const label = value ? value.label || (value.start + " \u2013 " + value.end) : presets[1];
  const pick = (l) => { onChange && onChange({ label: l }); setOpen(false); };
  const applyCustom = () => { if (start && end) { onChange && onChange({ label: start + " \u2013 " + end, start, end }); setOpen(false); } };
  const inp = (v, set, ph) => (
    <input value={v} placeholder={ph} onChange={(e) => set(e.target.value)}
      style={{ width: "100%", height: 28, padding: "0 8px", border: "1px solid var(--border-default)", borderRadius: 8, background: "var(--surface)", color: "var(--text-ink)", font: "400 11.5px var(--font-mono)", outline: "none", boxSizing: "border-box" }} />
  );
  return (
    <Popover open={open} onOpenChange={setOpen} align={align} width={232} style={style}
      trigger={<Button variant="secondary" icon={<Icon name="calendar" size={14} />}>
        <span style={{ font: "500 12px var(--font-mono)" }}>{label}</span>
      </Button>}>
      <div style={{ display: "flex", flexDirection: "column", gap: 3 }}>
        {presets.map((p) => <PresetRow key={p} label={p} active={label === p} onClick={() => pick(p)} />)}
        <div style={{ height: 1, background: "var(--row-sep)", margin: "6px 2px" }} />
        <span style={{ font: "500 11px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--text-muted)", padding: "2px 2px 4px" }}>Custom (ISO 8601)</span>
        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 6 }}>
          {inp(start, setStart, "2026-08-01")}
          {inp(end, setEnd, "2026-08-22")}
        </div>
        <Button size="sm" style={{ marginTop: 8, width: "100%" }} onClick={applyCustom} disabled={!start || !end}>Apply range</Button>
      </div>
    </Popover>
  );
}
