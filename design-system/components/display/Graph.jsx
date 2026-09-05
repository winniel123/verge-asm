import React from "react";

const NODE_R = { domain: 16, subdomain: 10, ip: 9, service: 6 };

function NodeShape({ n, selected, hovered }) {
  const r = NODE_R[n.type] || 9;
  const halo = n.sev ? "var(--sev-" + n.sev + "-dot)" : null;
  const stroke = n.type === "domain" ? "var(--text-ink)" : n.type === "ip" ? "var(--text-secondary)" : "var(--neutral-500)";
  const shape = n.type === "ip"
    ? <rect x={-r} y={-r} width={r * 2} height={r * 2} rx={5} fill="var(--surface)" stroke={stroke} strokeWidth={hovered ? 2 : 1.5}></rect>
    : <circle r={r} fill={n.type === "service" && n.sev ? "var(--sev-" + n.sev + "-dot)" : "var(--surface)"} stroke={n.type === "service" && n.sev ? "var(--surface)" : stroke} strokeWidth={n.type === "domain" ? 2 : hovered ? 2 : 1.5}></circle>;
  return (
    <g>
      {halo && <circle r={r + 7} fill={halo} opacity="0.12"></circle>}
      {halo && <circle r={r + 4.5} fill="none" stroke={halo} strokeWidth="2" opacity="0.65"></circle>}
      {selected && <circle r={r + 9} fill="none" stroke="var(--focus-ring)" strokeWidth="2.5"></circle>}
      {shape}
    </g>
  );
}

export function Graph({ nodes = [], edges = [], height = 560, viewWidth = 1200, viewHeight = 640, selectedId, onNodeSelect, controls = true, minimap = false, style }) {
  const [view, setView] = React.useState({ x: 0, y: 0, k: 1 });
  const [hov, setHov] = React.useState(null);
  const drag = React.useRef(null);
  const svgRef = React.useRef(null);
  const byId = {};
  nodes.forEach((n) => { byId[n.id] = n; });
  const zoomAt = (f, cx, cy) => setView((v) => {
    const k = Math.min(2.5, Math.max(0.5, v.k * f));
    return { k, x: cx - (cx - v.x) * (k / v.k), y: cy - (cy - v.y) * (k / v.k) };
  });
  const zoom = (f) => zoomAt(f, viewWidth / 2, viewHeight / 2);
  React.useEffect(() => {
    const el = svgRef.current;
    if (!el) return;
    const onWheel = (e) => {
      e.preventDefault();
      const rect = el.getBoundingClientRect();
      const s = viewWidth / (rect.width || viewWidth);
      zoomAt(e.deltaY < 0 ? 1.15 : 0.87, (e.clientX - rect.left) * s, (e.clientY - rect.top) * s);
    };
    el.addEventListener("wheel", onWheel, { passive: false });
    return () => el.removeEventListener("wheel", onWheel);
  }, [viewWidth]);
  const onDown = (e) => {
    drag.current = { sx: e.clientX, sy: e.clientY, ox: view.x, oy: view.y, moved: false, captured: false, pid: e.pointerId, el: e.currentTarget };
  };
  const onMove = (e) => {
    if (!drag.current) return;
    const dx = e.clientX - drag.current.sx, dy = e.clientY - drag.current.sy;
    if (!drag.current.moved && Math.abs(dx) + Math.abs(dy) > 3) {
      drag.current.moved = true;
      // Capturing on pointerdown retargets the eventual click to the svg and kills node selection.
      try { drag.current.el.setPointerCapture(drag.current.pid); } catch (err) {}
    }
    if (!drag.current.moved) return;
    const el = e.currentTarget;
    const scale = viewWidth / (el.clientWidth || viewWidth);
    // Reading the ref inside the updater throws: React batches, and pointerup may null it first.
    const ox = drag.current.ox, oy = drag.current.oy;
    setView((v) => ({ ...v, x: ox + dx * scale, y: oy + dy * scale }));
  };
  const onUp = () => { drag.current = null; };
  const pick = (n) => (e) => {
    e.stopPropagation();
    if (drag.current && drag.current.moved) return;
    onNodeSelect && onNodeSelect(n);
  };
  const ctrlBtn = (label, onClick, glyph) => (
    <button key={label} type="button" aria-label={label} onClick={onClick}
      style={{ display: "flex", alignItems: "center", justifyContent: "center", width: 30, height: 30, border: "none", background: "transparent", color: "var(--text-secondary)", cursor: "pointer", font: "500 15px var(--font-mono)", borderRadius: 8 }}
      onMouseEnter={(e) => e.currentTarget.style.background = "var(--surface-sunken)"}
      onMouseLeave={(e) => e.currentTarget.style.background = "transparent"}>{glyph}</button>
  );
  return (
    <div style={{ position: "relative", ...style }}>
      <svg ref={svgRef} viewBox={"0 0 " + viewWidth + " " + viewHeight} style={{ display: "block", width: "100%", height, cursor: drag.current ? "grabbing" : "grab", touchAction: "none" }}
        onPointerDown={onDown} onPointerMove={onMove} onPointerUp={onUp} onPointerLeave={onUp}>
        <g transform={"translate(" + view.x + "," + view.y + ") scale(" + view.k + ")"}>
          {edges.map((e, i) => {
            const a = byId[e.from], b = byId[e.to];
            if (!a || !b) return null;
            return <line key={i} x1={a.x} y1={a.y} x2={b.x} y2={b.y} stroke={b.type === "service" ? "var(--border-default)" : "var(--border-strong)"} strokeWidth="1.25"></line>;
          })}
          {nodes.map((n) => (
            <g key={n.id} transform={"translate(" + n.x + "," + n.y + ")"} onClick={pick(n)}
              onMouseEnter={() => setHov(n.id)} onMouseLeave={() => setHov(null)} style={{ cursor: "pointer" }}>
              <NodeShape n={n} selected={selectedId === n.id} hovered={hov === n.id} />
              <text x={(NODE_R[n.type] || 9) + 9} y={4}
                style={{ font: (n.type === "domain" ? "600 13px" : "400 11px") + " var(--font-mono)", fill: n.type === "domain" ? "var(--text-ink)" : "var(--text-secondary)", userSelect: "none" }}>{n.label}</text>
            </g>
          ))}
        </g>
      </svg>
      {minimap && (() => {
        const mw = 110, s = mw / viewWidth, mh = Math.round(viewHeight * s);
        const clamp = (v, lo, hi) => Math.max(lo, Math.min(hi, v));
        const vx = clamp((-view.x / view.k) * s, 0, mw), vy = clamp((-view.y / view.k) * s, 0, mh);
        const vw = clamp((viewWidth / view.k) * s, 8, mw - vx), vh = clamp((viewHeight / view.k) * s, 8, mh - vy);
        return (
          <div style={{ position: "absolute", left: 14, bottom: 14, background: "var(--surface-raised)", border: "1px solid var(--border-default)", borderRadius: 10, boxShadow: "var(--shadow-md)", padding: 6, lineHeight: 0 }}>
            <svg width={mw} height={mh}>
              {nodes.map((n) => <circle key={n.id} cx={n.x * s} cy={n.y * s} r="1.5" fill={n.sev ? "var(--sev-" + n.sev + "-dot)" : "var(--neutral-400)"}></circle>)}
              <rect x={vx} y={vy} width={vw} height={vh} rx="3" fill="none" stroke="var(--accent)" strokeWidth="1.5"></rect>
            </svg>
          </div>
        );
      })()}
      {controls && (
        <div style={{ position: "absolute", right: 14, bottom: 14, display: "flex", flexDirection: "column", background: "var(--surface-raised)", border: "1px solid var(--border-default)", borderRadius: 12, boxShadow: "var(--shadow-md)", padding: 3 }}>
          {ctrlBtn("Zoom in", () => zoom(1.25), "+")}
          {ctrlBtn("Zoom out", () => zoom(0.8), "\u2212")}
          {ctrlBtn("Reset view", () => setView({ x: 0, y: 0, k: 1 }), "\u2922")}
        </div>
      )}
    </div>
  );
}

export function GraphLegend({ style }) {
  const item = (glyph, label) => (
    <span key={label} style={{ display: "inline-flex", alignItems: "center", gap: 7 }}>
      {glyph}
      <span style={{ font: "400 11px var(--font-mono)", color: "var(--text-secondary)" }}>{label}</span>
    </span>
  );
  return (
    <div style={{ display: "flex", alignItems: "center", gap: 20, flexWrap: "wrap", ...style }}>
      {item(<svg width="16" height="16" viewBox="0 0 16 16"><circle cx="8" cy="8" r="6.5" fill="var(--surface)" stroke="var(--text-ink)" strokeWidth="2"></circle></svg>, "domain")}
      {item(<svg width="16" height="16" viewBox="0 0 16 16"><circle cx="8" cy="8" r="5.5" fill="var(--surface)" stroke="var(--neutral-500)" strokeWidth="1.5"></circle></svg>, "subdomain")}
      {item(<svg width="16" height="16" viewBox="0 0 16 16"><rect x="2.5" y="2.5" width="11" height="11" rx="3.5" fill="var(--surface)" stroke="var(--text-secondary)" strokeWidth="1.5"></rect></svg>, "ip")}
      {item(<svg width="16" height="16" viewBox="0 0 16 16"><circle cx="8" cy="8" r="4" fill="var(--neutral-500)"></circle></svg>, "service")}
      {item(<svg width="18" height="18" viewBox="0 0 18 18"><circle cx="9" cy="9" r="7.5" fill="var(--sev-critical-dot)" opacity="0.15"></circle><circle cx="9" cy="9" r="6" fill="none" stroke="var(--sev-critical-dot)" strokeWidth="2" opacity="0.7"></circle><circle cx="9" cy="9" r="3" fill="var(--surface)" stroke="var(--neutral-500)" strokeWidth="1.5"></circle></svg>, "open signal halo")}
    </div>
  );
}
