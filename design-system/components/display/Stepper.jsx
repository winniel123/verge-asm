import React from "react";

export function Stepper({ steps = [], active = 0, style }) {
  return (
    <div style={{ display: "flex", flexDirection: "column", fontFamily: "var(--font-ui)", ...style }}>
      {steps.map((s, i) => {
        const done = i < active, current = i === active;
        return (
          <div key={i} style={{ display: "flex", gap: 14 }}>
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", flex: "none" }}>
              <span style={{ display: "inline-flex", alignItems: "center", justifyContent: "center", width: 26, height: 26, borderRadius: 999, flex: "none", background: done ? "var(--accent)" : "var(--surface)", border: done ? "1px solid var(--accent)" : current ? "2px solid var(--accent)" : "1px solid var(--border-strong)", font: "600 11.5px var(--font-mono)", color: current ? "var(--link)" : "var(--text-muted)" }}>
                {done ? <svg viewBox="0 0 18 18" width="12" height="12"><path d="M3.5 9.5l3.5 3.5 7.5-8" fill="none" stroke="var(--on-accent)" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round"></path></svg> : i + 1}
              </span>
              {i < steps.length - 1 && <span style={{ width: 1.5, flex: 1, minHeight: 18, background: done ? "var(--accent)" : "var(--border-default)", margin: "4px 0" }} />}
            </div>
            <div style={{ display: "flex", flexDirection: "column", gap: 2, flex: 1, paddingBottom: i < steps.length - 1 ? 20 : 0, paddingTop: 3, minWidth: 0 }}>
              <span style={{ font: (current ? "600" : "500") + " 13.5px var(--font-ui)", color: current ? "var(--text-ink)" : done ? "var(--text-body)" : "var(--text-muted)" }}>{s.title}</span>
              {s.detail && <span style={{ font: "400 12.5px/1.5 var(--font-ui)", color: "var(--text-secondary)" }}>{s.detail}</span>}
            </div>
          </div>
        );
      })}
    </div>
  );
}
