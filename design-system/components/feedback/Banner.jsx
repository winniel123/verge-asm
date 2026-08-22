import React from "react";
import { Icon } from "../media/Icon.jsx";
import { IconButton } from "../forms/IconButton.jsx";

const TONES = {
  neutral: { bg: "var(--surface-sunken)", bd: "var(--border-default)", fg: "var(--text-body)", icon: "info", ic: "var(--text-secondary)" },
  accent: { bg: "var(--accent-softer)", bd: "var(--accent-soft)", fg: "var(--text-body)", icon: "info", ic: "var(--accent)" },
  ok: { bg: "var(--ok-soft)", bd: "var(--ok-border)", fg: "var(--text-body)", icon: "check-circle-2", ic: "var(--ok)" },
  warn: { bg: "var(--warn-soft)", bd: "var(--warn-border)", fg: "var(--text-body)", icon: "alert-triangle", ic: "var(--warn)" },
  danger: { bg: "var(--danger-soft)", bd: "var(--danger-border)", fg: "var(--text-body)", icon: "alert-octagon", ic: "var(--danger)" },
};

/* Persistent inline page-level alert (Toast is transient; Banner stays until resolved/dismissed). */
export function Banner({ tone = "neutral", title, children, action, onDismiss, icon, style }) {
  const t = TONES[tone] || TONES.neutral;
  const [leaving, setLeaving] = React.useState(false);
  const dismiss = () => {
    if (leaving) return;
    setLeaving(true);
    setTimeout(() => onDismiss && onDismiss(), 240);
  };
  return (
    <div style={{ display: "grid", gridTemplateRows: leaving ? "0fr" : "1fr", transition: "grid-template-rows var(--dur-base) var(--ease-out)", ...style }}>
      <div style={{ overflow: "hidden", minHeight: 0 }}>
        <div role="status" style={{ display: "flex", alignItems: "flex-start", gap: 10, padding: "11px 14px", background: t.bg, border: "1px solid " + t.bd, borderRadius: 12, fontFamily: "var(--font-ui)", opacity: leaving ? 0 : 1, transition: "opacity var(--dur-base) var(--ease-out)" }}>
      <span style={{ display: "inline-flex", color: t.ic, marginTop: 1, flex: "none" }}><Icon name={icon || t.icon} size={15} /></span>
      <div style={{ display: "flex", flexDirection: "column", gap: 2, minWidth: 0, flex: 1 }}>
        {title && <span style={{ font: "600 13px var(--font-ui)", color: "var(--text-ink)" }}>{title}</span>}
        {children && <span style={{ font: "400 12.5px/1.5 var(--font-ui)", color: t.fg }}>{children}</span>}
      </div>
      {action && <span style={{ flex: "none", marginLeft: 8, alignSelf: "center" }}>{action}</span>}
          {onDismiss && <IconButton icon="x" label="Dismiss" size="sm" onClick={dismiss} style={{ flex: "none", margin: "-3px -4px 0 0" }} />}
        </div>
      </div>
    </div>
  );
}
