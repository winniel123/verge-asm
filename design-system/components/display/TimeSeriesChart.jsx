import React from "react";

export function TimeSeriesChart({ series = [], labels = [], hoverLabels, height = 220, yFormat, label, showLegend, style }) {
  const wrapRef = React.useRef(null);
  const [w, setW] = React.useState(0);
  const [hover, setHover] = React.useState(null);
  const [drawn, setDrawn] = React.useState(false);
  React.useEffect(() => { if (w > 0 && !drawn) { const id = requestAnimationFrame(() => setDrawn(true)); return () => cancelAnimationFrame(id); } }, [w]);
  React.useEffect(() => {
    const el = wrapRef.current;
    if (!el) return;
    const ro = new ResizeObserver((es) => setW(es[0].contentRect.width));
    ro.observe(el);
    setW(el.getBoundingClientRect().width);
    return () => ro.disconnect();
  }, []);
  const colors = ["var(--chart-1)", "var(--chart-2)", "var(--chart-3)", "var(--chart-4)"];
  const n = series.length ? Math.max.apply(null, series.map((s) => s.data.length)) : 0;
  const legend = showLegend != null ? showLegend : series.length > 1;
  const fmt = yFormat || ((v) => (v >= 1000 ? v / 1000 + "k" : String(v)));
  if (!series.length || n < 2) return <div ref={wrapRef} style={{ height, ...style }} />;
  const PL = 40, PR = 8, PT = 10, PB = 22, H = height;
  const max0 = Math.max.apply(null, series.map((s) => Math.max.apply(null, s.data)));
  const raw = (max0 || 1) / 4;
  const mag = Math.pow(10, Math.floor(Math.log10(raw)));
  const norm = raw / mag;
  const step = (norm <= 1 ? 1 : norm <= 2 ? 2 : norm <= 5 ? 5 : 10) * mag;
  const top = Math.max(step, Math.ceil(max0 / step) * step);
  const ticks = [];
  for (let v = 0; v <= top + 1e-9; v += step) ticks.push(v);
  const iw = Math.max(0, w - PL - PR);
  const x = (i) => PL + (i / (n - 1)) * iw;
  const y = (v) => PT + (1 - v / top) * (H - PT - PB);
  const onMove = (e) => {
    const r = e.currentTarget.getBoundingClientRect();
    const i = Math.round(((e.clientX - r.left - PL) / iw) * (n - 1));
    setHover(Math.max(0, Math.min(n - 1, i)));
  };
  const hl = (hoverLabels || labels)[hover] || "";
  const flip = hover != null && x(hover) > w - 150;
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 10, ...style }}>
      <div ref={wrapRef} style={{ position: "relative" }}>
        {w > 0 && (
          <svg width={w} height={H} viewBox={"0 0 " + w + " " + H} role="img" aria-label={label}
            onMouseMove={onMove} onMouseLeave={() => setHover(null)} style={{ display: "block", overflow: "visible" }}>
            {ticks.map((t) => (
              <g key={t} style={{ opacity: drawn ? 1 : 0, transition: "opacity 300ms var(--ease-out)" }}>
                <line x1={PL} x2={w - PR} y1={y(t)} y2={y(t)} stroke={t === 0 ? "var(--border-default)" : "var(--row-sep)"} strokeWidth="1" />
                <text x={PL - 8} y={y(t) + 3} textAnchor="end" style={{ font: "400 10.5px var(--font-mono)", fill: "var(--text-muted)" }}>{fmt(t)}</text>
              </g>
            ))}
            {labels.map((l, i) => (l && i < n ? (
              <text key={i} x={x(i)} y={H - 6} textAnchor="middle" style={{ font: "400 10.5px var(--font-mono)", fill: "var(--text-muted)", opacity: drawn ? 1 : 0, transition: "opacity 300ms var(--ease-out)" }}>{l}</text>
            ) : null))}
            {hover != null && <line x1={x(hover)} x2={x(hover)} y1={PT} y2={H - PB} stroke="var(--border-strong)" strokeWidth="1" strokeDasharray="2 3" />}
            {series.map((s, si) => {
              const c = s.color || colors[si % colors.length];
              const pts = s.data.map((v, i) => x(i).toFixed(1) + "," + y(v).toFixed(1)).join(" ");
              return (
                <g key={si}>
                  <polyline points={pts} fill="none" stroke={c} strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round" pathLength="100" strokeDasharray="100" style={{ strokeDashoffset: drawn ? 0 : 100, transition: "stroke-dashoffset 700ms var(--ease-out) " + si * 120 + "ms" }} />
                  {hover != null && s.data[hover] != null && <circle cx={x(hover)} cy={y(s.data[hover])} r="3" fill={c} stroke="var(--surface)" strokeWidth="1.5" />}
                </g>
              );
            })}
          </svg>
        )}
        {hover != null && (
          <div style={{ position: "absolute", top: PT, left: x(hover) + (flip ? -10 : 10), transform: flip ? "translateX(-100%)" : "none", background: "var(--surface)", border: "1px solid var(--border-default)", borderRadius: 10, boxShadow: "var(--shadow-md)", padding: "8px 10px", pointerEvents: "none", display: "flex", flexDirection: "column", gap: 5, minWidth: 110, zIndex: 5 }}>
            {hl && <span style={{ font: "500 10.5px var(--font-mono)", letterSpacing: "0.05em", textTransform: "uppercase", color: "var(--text-muted)" }}>{hl}</span>}
            {series.map((s, si) => (
              <span key={si} style={{ display: "flex", alignItems: "center", gap: 6 }}>
                <span style={{ width: 8, height: 8, borderRadius: 2, background: s.color || colors[si % colors.length], flex: "none" }} />
                <span style={{ font: "400 12px var(--font-ui)", color: "var(--text-secondary)" }}>{s.label}</span>
                <span style={{ marginLeft: "auto", font: "500 12px var(--font-mono)", color: "var(--text-ink)", paddingLeft: 10 }}>{s.data[hover] != null ? fmt(s.data[hover]) : "\u2014"}</span>
              </span>
            ))}
          </div>
        )}
      </div>
      {legend && (
        <div style={{ display: "flex", gap: 16, flexWrap: "wrap", paddingLeft: PL }}>
          {series.map((s, si) => (
            <span key={si} style={{ display: "inline-flex", alignItems: "center", gap: 6 }}>
              <span style={{ width: 8, height: 8, borderRadius: 2, background: s.color || colors[si % colors.length] }} />
              <span style={{ font: "400 12px var(--font-ui)", color: "var(--text-secondary)" }}>{s.label}</span>
            </span>
          ))}
        </div>
      )}
    </div>
  );
}
