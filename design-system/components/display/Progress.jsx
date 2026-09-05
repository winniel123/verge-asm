import React from "react";

export function Progress({ value, label, detail, tone = "accent", size = "md", style }) {
  const h = size === "sm" ? 4 : 6;
  const color = tone === "ok" ? "var(--ok-solid)" : tone === "warn" ? "var(--warn-solid)" : tone === "danger" ? "var(--danger-solid)" : "var(--accent)";
  const indeterminate = value == null;
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 6, fontFamily: "var(--font-ui)", ...style }}>
      {(label || detail) && (
        <div style={{ display: "flex", alignItems: "baseline", gap: 8 }}>
          {label && <span style={{ font: "500 11px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--text-muted)" }}>{label}</span>}
          {detail && <span style={{ marginLeft: "auto", font: "400 11.5px var(--font-mono)", color: "var(--text-secondary)", whiteSpace: "nowrap" }}>{detail}</span>}
        </div>
      )}
      <div role="progressbar" aria-valuenow={indeterminate ? undefined : Math.round(value)} aria-valuemin={0} aria-valuemax={100}
        style={{ height: h, borderRadius: 999, background: "var(--surface-sunken)", overflow: "hidden", position: "relative" }}>
        {indeterminate
          ? <span style={{ position: "absolute", top: 0, bottom: 0, width: "34%", borderRadius: 999, background: color, animation: "vg-progress-sweep 1.4s var(--ease-in-out) infinite" }} />
          : <span style={{ display: "block", height: "100%", width: Math.max(0, Math.min(100, value)) + "%", borderRadius: 999, background: color, transition: "width var(--dur-slow) var(--ease-out)" }} />}
      </div>
    </div>
  );
}
