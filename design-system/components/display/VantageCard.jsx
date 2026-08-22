import React from "react";
import { Tag } from "./Tag.jsx";
import { AvailabilityBadge } from "./AvailabilityBadge.jsx";

/* A vantage: network position + resolver identity; class re-verified each batch.
   Unverified is a real state — it makes no exposure claims. */
export function VantageCard({ name, vantageClass = "internet", resolver, availability = "available", latency, style }) {
  const unverified = vantageClass === "unverified" || availability === "unverified";
  return (
    <section style={{ background: "var(--surface)", border: "1px " + (unverified ? "dashed var(--border-strong)" : "solid var(--border-default)"), borderRadius: 16, boxShadow: unverified ? "none" : "var(--shadow-sm)", padding: 16, display: "flex", flexDirection: "column", gap: 10, minWidth: 0, fontFamily: "var(--font-ui)", ...style }}>
      <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
        <span style={{ font: "600 13px var(--font-mono)", color: "var(--text-ink)", whiteSpace: "nowrap", flex: "none" }}>{name}</span>
        <Tag>{vantageClass}</Tag>
      </div>
      <div style={{ display: "flex", alignItems: "center", gap: 12, font: "400 11.5px var(--font-mono)", color: "var(--text-secondary)", minHeight: 20 }}>
        {resolver && <span style={{ flex: "0 1 auto", minWidth: 0, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>resolver {resolver}</span>}
        {latency && <span style={{ flex: "none" }}>{latency}</span>}
        <span style={{ marginLeft: "auto", flex: "none" }}><AvailabilityBadge state={availability} size="sm" /></span>
      </div>
      {unverified && <span style={{ font: "400 11.5px/1.5 var(--font-ui)", color: "var(--text-muted)" }}>Unverified — this vantage makes no exposure claims until re-verified.</span>}
    </section>
  );
}
