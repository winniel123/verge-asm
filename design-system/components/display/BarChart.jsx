import React from "react";

/* Mini bar chart. data: number[]; labels align 1:1 (sparse ok — empty strings skip). */
export function BarChart({ data = [], labels, height = 72, color = "var(--chart-1)", emphasizeLast = true, showBaseline = true, style }) {
  const max = Math.max(...data, 1);
  const [on, setOn] = React.useState(false);
  React.useEffect(() => { const id = requestAnimationFrame(() => setOn(true)); return () => cancelAnimationFrame(id); }, []);
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 6, ...style }}>
      <div style={{ display: "flex", alignItems: "flex-end", gap: 4, height, borderBottom: showBaseline ? "1px solid var(--chart-grid)" : "none", paddingBottom: showBaseline ? 1 : 0 }}>
        {data.map((v, i) => (
          <span key={i} title={String(v)} style={{ flex: 1, minWidth: 3, height: on ? Math.max(2, (v / max) * height) + "px" : "2px", borderRadius: "3px 3px 0 0", background: color, opacity: emphasizeLast && i !== data.length - 1 ? 0.45 : 1, transition: "height 450ms var(--ease-out)", transitionDelay: Math.min(i * 30, 360) + "ms" }} />
        ))}
      </div>
      {labels && (
        <div style={{ display: "flex", justifyContent: "space-between", gap: 8 }}>
          {labels.filter((l) => l).map((l, i) => (
            <span key={i} style={{ font: "400 9.5px var(--font-mono)", color: "var(--text-muted)", whiteSpace: "nowrap" }}>{l}</span>
          ))}
        </div>
      )}
    </div>
  );
}
