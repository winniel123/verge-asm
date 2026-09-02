import React from "react";

/* GitHub-style activity-by-day grid. values: number[] oldest-first, one per day (columns = weeks).
   Intensity ramps on --chart-1 — activity is volume, never severity. */
export function HeatmapCalendar({ values = [], cell = 12, gap = 3, label, unit = "scans", startLabel, endLabel = "today", style }) {
  const max = values.reduce((m, v) => Math.max(m, v), 1);
  const [on, setOn] = React.useState(false);
  React.useEffect(() => { const id = requestAnimationFrame(() => setOn(true)); return () => cancelAnimationFrame(id); }, []);
  const level = (v) => (v <= 0 ? 0 : Math.max(1, Math.ceil((v / max) * 4)));
  const bg = (l) => (l === 0 ? "var(--surface-sunken)" : "color-mix(in srgb, var(--chart-1) " + [0, 28, 48, 72, 100][l] + "%, var(--surface))");
  const cellBorder = (l) => "1px solid " + (l === 0 ? "var(--row-sep)" : "transparent");
  return (
    <div style={{ display: "inline-flex", flexDirection: "column", gap: 8, fontFamily: "var(--font-ui)", ...style }}>
      <div role="img" aria-label={label} style={{ display: "grid", gridTemplateRows: "repeat(7, " + cell + "px)", gridAutoFlow: "column", gridAutoColumns: cell + "px", gap }}>
        {values.map((v, i) => (
          <span key={i} title={v + " " + unit} style={{ width: cell, height: cell, borderRadius: 3, background: bg(level(v)), border: cellBorder(level(v)), boxSizing: "border-box", opacity: on ? 1 : 0, transition: "opacity 320ms var(--ease-out)", transitionDelay: Math.min(Math.floor(i / 7) * 12, 700) + "ms" }} />
        ))}
      </div>
      <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
        {startLabel && <span style={{ font: "400 10.5px var(--font-mono)", color: "var(--text-muted)" }}>{startLabel}</span>}
        <span style={{ marginLeft: "auto", font: "400 10.5px var(--font-mono)", color: "var(--text-muted)" }}>less</span>
        {[0, 1, 2, 3, 4].map((l) => <span key={l} style={{ width: 10, height: 10, borderRadius: 3, background: bg(l), border: cellBorder(l), boxSizing: "border-box" }} />)}
        <span style={{ font: "400 10.5px var(--font-mono)", color: "var(--text-muted)" }}>more</span>
        {endLabel && <span style={{ font: "400 10.5px var(--font-mono)", color: "var(--text-muted)", marginLeft: 8 }}>{endLabel}</span>}
      </div>
    </div>
  );
}
