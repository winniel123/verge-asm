import React from "react";
import { Icon } from "../media/Icon.jsx";

const DAYS = ["Mo", "Tu", "We", "Th", "Fr", "Sa", "Su"];
const iso = (d) => d.getFullYear() + "-" + String(d.getMonth() + 1).padStart(2, "0") + "-" + String(d.getDate()).padStart(2, "0");
export function Calendar({ month, selected, onSelect, events = {}, min, max, cell = 36, unit = "run", label, style }) {
  const init = month ? new Date(month + "-01T00:00:00") : selected ? new Date(selected + "T00:00:00") : new Date();
  const [view, setView] = React.useState([init.getFullYear(), init.getMonth()]);
  const [focusIso, setFocusIso] = React.useState(null);
  const btns = React.useRef({});
  const vy = view[0], vm = view[1];
  const first = new Date(vy, vm, 1);
  const lead = (first.getDay() + 6) % 7;
  const cells = [];
  for (let i = 0; i < 42; i++) cells.push(new Date(vy, vm, 1 - lead + i));
  const todayIso = iso(new Date());
  const monthLabel = first.toLocaleDateString("en-US", { month: "long", year: "numeric" });
  const nav = (dm) => setView(function (v) { const d = new Date(v[0], v[1] + dm, 1); return [d.getFullYear(), d.getMonth()]; });
  const move = (from, days) => {
    const d = new Date(from + "T00:00:00");
    d.setDate(d.getDate() + days);
    const s = iso(d);
    if (d.getFullYear() !== vy || d.getMonth() !== vm) setView([d.getFullYear(), d.getMonth()]);
    setFocusIso(s);
    requestAnimationFrame(() => { const b = btns.current[s]; if (b) b.focus(); });
  };
  const onKey = (e, s) => {
    const step = { ArrowLeft: -1, ArrowRight: 1, ArrowUp: -7, ArrowDown: 7 }[e.key];
    if (step) { e.preventDefault(); move(s, step); }
    else if (e.key === "PageUp") { e.preventDefault(); nav(-1); }
    else if (e.key === "PageDown") { e.preventDefault(); nav(1); }
  };
  const focusTarget = focusIso || selected || todayIso;
  const navBtn = { width: 26, height: 26, display: "inline-flex", alignItems: "center", justifyContent: "center", background: "transparent", border: "1px solid var(--border-default)", borderRadius: 8, color: "var(--text-secondary)", cursor: "pointer", padding: 0 };
  return (
    <div style={{ display: "inline-flex", flexDirection: "column", gap: 10, fontFamily: "var(--font-ui)", ...style }}>
      <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
        <span style={{ font: "600 13.5px var(--font-ui)", color: "var(--text-ink)" }}>{monthLabel}</span>
        <span style={{ marginLeft: "auto", display: "inline-flex", gap: 6 }}>
          <button type="button" aria-label="Previous month" onClick={() => nav(-1)} style={navBtn}><Icon name="chevron-left" size={14} /></button>
          <button type="button" aria-label="Next month" onClick={() => nav(1)} style={navBtn}><Icon name="chevron-right" size={14} /></button>
        </span>
      </div>
      <div role="grid" aria-label={label || monthLabel} style={{ display: "grid", gridTemplateColumns: "repeat(7, " + cell + "px)", gap: 2 }}>
        {DAYS.map((d) => <span key={d} role="columnheader" style={{ font: "500 10px var(--font-mono)", letterSpacing: "0.06em", textTransform: "uppercase", color: "var(--text-muted)", textAlign: "center", paddingBottom: 4 }}>{d}</span>)}
        {cells.map((d) => {
          const s = iso(d);
          const isSel = selected === s, isToday = s === todayIso, out = d.getMonth() !== vm;
          const dis = (min && s < min) || (max && s > max);
          const count = events[s] || 0;
          return (
            <button key={s} type="button" role="gridcell" aria-selected={isSel || undefined} disabled={dis || undefined}
              ref={(el) => { btns.current[s] = el; }} tabIndex={s === focusTarget ? 0 : -1}
              onKeyDown={(e) => onKey(e, s)} onClick={() => { setFocusIso(s); if (onSelect) onSelect(s); }}
              title={count ? count + " " + unit + (count === 1 ? "" : "s") : undefined}
              style={{ width: cell, height: cell, display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center", gap: 3, background: isSel ? "var(--accent)" : "transparent", color: isSel ? "var(--on-accent)" : out ? "var(--text-muted)" : "var(--text-body)", opacity: dis ? 0.35 : out && !isSel ? 0.55 : 1, border: "1px solid " + (isToday && !isSel ? "var(--accent)" : "transparent"), borderRadius: 9, font: (isToday || isSel ? "600" : "400") + " 12px var(--font-mono)", cursor: dis ? "default" : "pointer", transition: "background var(--dur-fast) var(--ease-out), border-color var(--dur-fast) var(--ease-out)", padding: 0 }}>
              <span>{d.getDate()}</span>
              <span aria-hidden="true" style={{ display: "flex", gap: 2, height: 4 }}>
                {Array.from({ length: Math.min(count, 3) }).map((_, di) => (
                  <span key={di} style={{ width: 4, height: 4, borderRadius: 2, background: isSel ? "var(--on-accent)" : "var(--chart-1)" }} />
                ))}
              </span>
            </button>
          );
        })}
      </div>
    </div>
  );
}
