import React from "react";
import { Icon } from "../media/Icon.jsx";

export function Carousel({ children, ariaLabel, autoAdvance = false, interval = 6000, loop = false, showArrows = true, showDots = true, style }) {
  const slides = React.Children.toArray(children);
  const n = slides.length;
  const [i, setI] = React.useState(0);
  const [dx, setDx] = React.useState(0);
  const [paused, setPaused] = React.useState(false);
  const drag = React.useRef(null);
  const go = (j) => setI(loop ? ((j % n) + n) % n : Math.max(0, Math.min(n - 1, j)));
  React.useEffect(() => {
    if (!autoAdvance || paused || n < 2) return;
    const t = setInterval(() => setI((v) => (v + 1) % n), interval);
    return () => clearInterval(t);
  }, [autoAdvance, paused, n, interval]);
  const down = (e) => { drag.current = { x: e.clientX, cap: false, el: e.currentTarget, id: e.pointerId }; };
  const move = (e) => {
    const d = drag.current;
    if (!d) return;
    const delta = e.clientX - d.x;
    if (!d.cap && Math.abs(delta) > 6) { d.cap = true; try { d.el.setPointerCapture(d.id); } catch (_) {} }
    if (d.cap) setDx(delta);
  };
  const up = () => {
    const d = drag.current;
    drag.current = null;
    const dxv = dx;
    setDx(0);
    if (!d || !d.cap) return;
    if (dxv < -40) go(i + 1); else if (dxv > 40) go(i - 1);
  };
  const onKey = (e) => {
    if (e.key === "ArrowLeft") { e.preventDefault(); go(i - 1); }
    else if (e.key === "ArrowRight") { e.preventDefault(); go(i + 1); }
  };
  const navBtn = (dis) => ({ width: 28, height: 28, display: "inline-flex", alignItems: "center", justifyContent: "center", background: "var(--surface)", border: "1px solid var(--border-default)", borderRadius: 8, color: "var(--text-secondary)", cursor: dis ? "default" : "pointer", opacity: dis ? 0.35 : 1, padding: 0 });
  const atStart = !loop && i === 0, atEnd = !loop && i === n - 1;
  return (
    <div role="group" aria-roledescription="carousel" aria-label={ariaLabel} tabIndex={0} onKeyDown={onKey}
      onMouseEnter={() => setPaused(true)} onMouseLeave={() => setPaused(false)} onFocus={() => setPaused(true)} onBlur={() => setPaused(false)}
      style={{ display: "flex", flexDirection: "column", gap: 12, outlineOffset: 4, ...style }}>
      <div style={{ overflow: "hidden", borderRadius: 12 }} onPointerDown={down} onPointerMove={move} onPointerUp={up} onPointerCancel={up}>
        <div style={{ display: "flex", touchAction: "pan-y", userSelect: "none", transform: "translateX(calc(" + i * -100 + "% + " + dx + "px))", transition: dx !== 0 ? "none" : "transform 420ms var(--ease-in-out)" }}>
          {slides.map((s, si) => (
            <div key={si} aria-hidden={si === i ? undefined : true} style={{ flex: "0 0 100%", minWidth: 0, boxSizing: "border-box" }}>{s}</div>
          ))}
        </div>
      </div>
      {(showDots || showArrows) && n > 1 && (
        <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
          {showDots && (
            <span style={{ display: "inline-flex", gap: 6, alignItems: "center" }}>
              {slides.map((_, si) => (
                <button key={si} type="button" aria-label={"Slide " + (si + 1)} onClick={() => go(si)}
                  style={{ width: si === i ? 18 : 6, height: 6, borderRadius: 3, border: "none", padding: 0, cursor: "pointer", background: si === i ? "var(--accent)" : "var(--border-strong)", transition: "width var(--dur-base) var(--ease-out), background var(--dur-base) var(--ease-out)" }} />
              ))}
            </span>
          )}
          {showArrows && (
            <span style={{ marginLeft: "auto", display: "inline-flex", gap: 6 }}>
              <button type="button" aria-label="Previous slide" disabled={atStart || undefined} onClick={() => go(i - 1)} style={navBtn(atStart)}><Icon name="chevron-left" size={14} /></button>
              <button type="button" aria-label="Next slide" disabled={atEnd || undefined} onClick={() => go(i + 1)} style={navBtn(atEnd)}><Icon name="chevron-right" size={14} /></button>
            </span>
          )}
        </div>
      )}
      <span aria-live="polite" style={{ position: "absolute", width: 1, height: 1, overflow: "hidden", clipPath: "inset(50%)" }}>{"Slide " + (i + 1) + " of " + n}</span>
    </div>
  );
}
