import React from "react";

/* Pointer + keyboard slider with mono readout. */
export function Slider({ label, value = 0, onChange, min = 0, max = 100, step = 1, unit = "", style }) {
  const trackRef = React.useRef(null);
  const dragging = React.useRef(false);
  const [foc, setFoc] = React.useState(false);
  const pct = ((value - min) / ((max - min) || 1)) * 100;
  const setFromX = (clientX) => {
    const el = trackRef.current;
    if (!el || !onChange) return;
    const r = el.getBoundingClientRect();
    const t = Math.min(1, Math.max(0, (clientX - r.left) / (r.width || 1)));
    let v = min + t * (max - min);
    v = Math.round(v / step) * step;
    onChange(Math.min(max, Math.max(min, v)));
  };
  const onDown = (e) => {
    dragging.current = true;
    try { e.currentTarget.setPointerCapture(e.pointerId); } catch (err) {}
    setFromX(e.clientX);
  };
  const onMove = (e) => { if (dragging.current) setFromX(e.clientX); };
  const onUp = () => { dragging.current = false; };
  const onKey = (e) => {
    if (e.key === "ArrowRight" || e.key === "ArrowUp") { e.preventDefault(); onChange && onChange(Math.min(max, value + step)); }
    else if (e.key === "ArrowLeft" || e.key === "ArrowDown") { e.preventDefault(); onChange && onChange(Math.max(min, value - step)); }
  };
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 8, fontFamily: "var(--font-ui)", ...style }}>
      {(label || unit) && (
        <div style={{ display: "flex", alignItems: "baseline", gap: 8 }}>
          {label && <span style={{ fontSize: 12.5, fontWeight: 500, color: "var(--text-body)" }}>{label}</span>}
          <span style={{ marginLeft: "auto", font: "500 12px var(--font-mono)", color: "var(--text-secondary)" }}>{value.toLocaleString("en-US")}{unit ? " " + unit : ""}</span>
        </div>
      )}
      <div ref={trackRef} onPointerDown={onDown} onPointerMove={onMove} onPointerUp={onUp} onPointerLeave={onUp}
        style={{ position: "relative", height: 20, display: "flex", alignItems: "center", cursor: "pointer", touchAction: "none" }}>
        <div style={{ height: 6, width: "100%", borderRadius: 999, background: "var(--surface-sunken)", overflow: "hidden" }}>
          <div style={{ height: "100%", width: pct + "%", borderRadius: 999, background: "var(--accent)" }} />
        </div>
        <span tabIndex={0} role="slider" aria-valuenow={value} aria-valuemin={min} aria-valuemax={max}
          onKeyDown={onKey} onFocus={() => setFoc(true)} onBlur={() => setFoc(false)}
          style={{ position: "absolute", left: pct + "%", transform: "translateX(-50%)", width: 16, height: 16, borderRadius: 999, background: "#ffffff", border: "2px solid var(--accent)", boxShadow: foc ? "0 0 0 2px var(--surface), 0 0 0 4px var(--focus-ring)" : "0 1px 2px rgba(35,31,25,0.2)", outline: "none" }} />
      </div>
    </div>
  );
}
