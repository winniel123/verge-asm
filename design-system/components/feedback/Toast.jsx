import React from "react";
import { IconButton } from "../forms/IconButton.jsx";

const TONES = { neutral: "var(--neutral-400)", ok: "var(--ok-solid)", warn: "var(--warn-solid)", danger: "var(--danger-solid)" };

export function Toast({ title, description, tone = "neutral", action, onDismiss, floating, style }) {
  const [leaving, setLeaving] = React.useState(false);
  const dismiss = () => {
    if (leaving) return;
    setLeaving(true);
    setTimeout(() => onDismiss && onDismiss(), 240);
  };
  const anim = leaving ? "vg-toast-out var(--dur-base) var(--ease-out) forwards" : "vg-toast-in var(--dur-slow) var(--ease-out)";
  const pos = floating ? { position: "fixed", right: 24, bottom: 24, zIndex: 110, animation: anim } : { animation: leaving ? anim : "none" };
  return (
    <div role="status" style={{ display: "flex", alignItems: "flex-start", gap: 10, minWidth: 300, maxWidth: 420, padding: "12px 14px", background: "var(--surface-raised)", border: "1px solid var(--border-default)", borderRadius: 16, boxShadow: "var(--shadow-md)", fontFamily: "var(--font-ui)", ...pos, ...style }}>
      <span style={{ width: 8, height: 8, borderRadius: 999, background: TONES[tone] || TONES.neutral, flex: "none", marginTop: 5 }} />
      <div style={{ display: "flex", flexDirection: "column", gap: 2, minWidth: 0, flex: 1 }}>
        <span style={{ font: "600 13px var(--font-ui)", color: "var(--text-ink)" }}>{title}</span>
        {description && <span style={{ font: "400 12.5px/1.45 var(--font-ui)", color: "var(--text-secondary)" }}>{description}</span>}
        {action && <span style={{ marginTop: 6 }}>{action}</span>}
      </div>
      {onDismiss && <IconButton icon="x" label="Dismiss" size="sm" onClick={dismiss} style={{ flex: "none", margin: "-2px -4px 0 0" }} />}
    </div>
  );
}
