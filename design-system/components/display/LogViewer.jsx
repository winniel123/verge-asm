import React from "react";

/* Live batch output on the console ground. lines: [{time, level?, text}]. Auto-follows while live. */
const LC = { info: "var(--neutral-400)", warn: "var(--warn-on-console)", error: "var(--danger-on-console)" };
export function LogViewer({ lines = [], title, live, height = 220, style }) {
  const ref = React.useRef(null);
  React.useEffect(() => { const el = ref.current; if (el && live) el.scrollTop = el.scrollHeight; }, [lines, live]);
  return (
    <div style={{ background: "var(--surface-console)", borderRadius: 12, overflow: "hidden", ...style }}>
      {(title || live) && (
        <div style={{ display: "flex", alignItems: "center", gap: 8, padding: "8px 14px 0" }}>
          {title && <span style={{ font: "500 10.5px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--neutral-500)" }}>{title}</span>}
          {live && (
            <span style={{ marginLeft: "auto", display: "inline-flex", alignItems: "center", gap: 6 }}>
              <span style={{ width: 6, height: 6, borderRadius: 999, background: "var(--primary-400)", animation: "vg-pulse 1.8s var(--ease-out) infinite" }} />
              <span style={{ font: "500 10px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--primary-400)" }}>streaming</span>
            </span>
          )}
        </div>
      )}
      <div ref={ref} style={{ height, overflowY: "auto", overflowX: "hidden", padding: "10px 14px 12px" }}>
        {lines.map((l, i) => (
          <div key={i} style={{ display: "flex", gap: 10, font: "400 11.5px/1.7 var(--font-mono)" }}>
            <span style={{ color: "var(--neutral-600)", flex: "none" }}>{l.time}</span>
            <span style={{ color: LC[l.level] || "var(--text-on-console)", overflowWrap: "anywhere" }}>{l.text}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
