import React from "react";
import { Badge } from "./Badge.jsx";

/* One integration in the library grid: neutral letter mark (no fake logos), category,
   install state as a Badge. Whole tile is the click target. */
export function IntegrationTile({ name, category, description, mark, state = "available", onClick, style }) {
  const [hover, setHover] = React.useState(false);
  const badge = state === "installed" ? { tone: "ok", text: "installed" } : state === "attention" ? { tone: "warn", text: "needs attention" } : { tone: "neutral", text: "available" };
  return (
    <button type="button" onClick={onClick} onMouseEnter={() => setHover(true)} onMouseLeave={() => setHover(false)}
      style={{ textAlign: "left", background: "var(--surface)", border: "1px solid " + (hover ? "var(--border-strong)" : "var(--border-default)"), borderRadius: 16, boxShadow: hover ? "var(--shadow-md)" : "var(--shadow-sm)", transform: hover ? "translateY(-1px)" : "none", transition: "box-shadow 140ms ease, transform 140ms ease, border-color 140ms ease", padding: 16, display: "flex", flexDirection: "column", gap: 10, cursor: "pointer", minWidth: 0, fontFamily: "var(--font-ui)", ...style }}>
      <div style={{ display: "flex", alignItems: "center", gap: 10, minWidth: 0 }}>
        <span aria-hidden="true" style={{ display: "inline-flex", alignItems: "center", justifyContent: "center", width: 34, height: 34, borderRadius: 10, background: "var(--surface-sunken)", border: "1px solid var(--border-default)", font: "600 12px var(--font-mono)", color: "var(--text-secondary)", flex: "none" }}>{mark}</span>
        <span style={{ font: "600 13.5px var(--font-ui)", color: "var(--text-ink)", whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis", minWidth: 0, flex: "1 1 auto" }}>{name}</span>
        <span style={{ flex: "none" }}><Badge tone={badge.tone} dot>{badge.text}</Badge></span>
      </div>
      <span style={{ font: "400 12.5px/1.55 var(--font-ui)", color: "var(--text-secondary)", display: "-webkit-box", WebkitLineClamp: 2, WebkitBoxOrient: "vertical", overflow: "hidden" }}>{description}</span>
      <span style={{ font: "500 10.5px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--text-muted)" }}>{category}</span>
    </button>
  );
}
