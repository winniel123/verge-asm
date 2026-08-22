import React from "react";
import { Icon } from "../media/Icon.jsx";

const TONES = {
  accent: { bg: "var(--accent-softer)", bd: "var(--accent-soft)", ic: "var(--accent)", icon: "info" },
  neutral: { bg: "var(--surface-sunken)", bd: "var(--border-default)", ic: "var(--text-secondary)", icon: "info" },
  ok: { bg: "var(--ok-soft)", bd: "var(--ok-border)", ic: "var(--ok)", icon: "check-circle-2" },
  warn: { bg: "var(--warn-soft)", bd: "var(--warn-border)", ic: "var(--warn)", icon: "alert-triangle" },
};

/* Docs/long-form aside. Banner is the app-chrome alert with actions; Callout is prose. */
export function Callout({ tone = "accent", icon, title, children, style }) {
  const t = TONES[tone] || TONES.accent;
  return (
    <div style={{ display: "flex", gap: 10, alignItems: "flex-start", padding: "12px 14px", background: t.bg, border: "1px solid " + t.bd, borderRadius: 12, fontFamily: "var(--font-ui)", ...style }}>
      <span style={{ color: t.ic, display: "inline-flex", marginTop: 1, flex: "none" }}><Icon name={icon || t.icon} size={15} /></span>
      <span style={{ display: "flex", flexDirection: "column", gap: 2, minWidth: 0 }}>
        {title && <span style={{ font: "600 13px var(--font-ui)", color: "var(--text-ink)" }}>{title}</span>}
        <span style={{ font: "400 13px/1.55 var(--font-ui)", color: "var(--text-body)" }}>{children}</span>
      </span>
    </div>
  );
}
