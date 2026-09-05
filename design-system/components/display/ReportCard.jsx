import React from "react";
import { Stat } from "./Stat.jsx";

export function ReportCard({ title, period, value, delta, deltaTone, caption, action, children, style }) {
  return (
    <section style={{ background: "var(--surface)", border: "1px solid var(--border-default)", borderRadius: 16, boxShadow: "var(--shadow-sm)", padding: 20, display: "flex", flexDirection: "column", gap: 14, minWidth: 0, ...style }}>
      <div style={{ display: "flex", alignItems: "baseline", gap: 10 }}>
        <span style={{ font: "500 11px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--text-muted)" }}>{title}</span>
        <span style={{ marginLeft: "auto", font: "400 11px var(--font-mono)", color: "var(--text-muted)", whiteSpace: "nowrap" }}>{period}</span>
        {action}
      </div>
      <Stat label="" value={value} delta={delta} deltaTone={deltaTone} caption={caption} style={{ gap: 2 }} />
      {children && <div style={{ marginTop: "auto" }}>{children}</div>}
    </section>
  );
}
