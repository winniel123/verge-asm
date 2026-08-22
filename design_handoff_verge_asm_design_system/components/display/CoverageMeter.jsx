import React from "react";

/* Coverage read for a scope. With a denominator: counted/total bar. Without one (name scopes,
   custody extensions): the distinct census state — striped bar, no percentage claimed. */
export function CoverageMeter({ label, counted, total, unit = "", detail, size = "md", style }) {
  const census = total == null;
  const h = size === "sm" ? 4 : 6;
  const pct = census ? 0 : Math.max(0, Math.min(100, (counted / (total || 1)) * 100));
  const fmt = (n) => (typeof n === "number" ? n.toLocaleString("en-US") : n);
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 6, fontFamily: "var(--font-ui)", ...style }}>
      <div style={{ display: "flex", alignItems: "baseline", gap: 8 }}>
        {label && <span style={{ font: "500 11px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--text-muted)" }}>{label}</span>}
        <span style={{ marginLeft: "auto", font: "400 11.5px var(--font-mono)", color: "var(--text-secondary)", whiteSpace: "nowrap" }}>
          {census ? "census \u00b7 " + fmt(counted) + (unit ? " " + unit : "") : fmt(counted) + " / " + fmt(total) + (unit ? " " + unit : "")}
        </span>
      </div>
      {census ? (
        <div aria-label="Census — no denominator" style={{ height: h, borderRadius: 999, overflow: "hidden", background: "repeating-linear-gradient(45deg, var(--accent-soft) 0 5px, var(--surface-sunken) 5px 10px)" }} />
      ) : (
        <div role="progressbar" aria-valuenow={Math.round(pct)} aria-valuemin={0} aria-valuemax={100}
          style={{ height: h, borderRadius: 999, background: "var(--surface-sunken)", overflow: "hidden" }}>
          <span style={{ display: "block", height: "100%", width: pct + "%", borderRadius: 999, background: "var(--accent)", transition: "width var(--dur-slow) var(--ease-out)" }} />
        </div>
      )}
      <span style={{ font: "400 11px var(--font-ui)", color: "var(--text-muted)" }}>
        {detail || (census ? "no denominator — a census counts what it finds" : "")}
      </span>
    </div>
  );
}
