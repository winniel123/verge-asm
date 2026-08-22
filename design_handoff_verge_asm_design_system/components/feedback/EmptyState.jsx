import React from "react";
import { Icon } from "../media/Icon.jsx";

export function EmptyState({ icon = "radar", message, detail, action, style }) {
  return (
    <div style={{ display: "flex", flexDirection: "column", alignItems: "center", textAlign: "center", gap: 6, padding: "48px 24px", fontFamily: "var(--font-ui)", ...style }}>
      <span style={{ display: "inline-flex", alignItems: "center", justifyContent: "center", width: 48, height: 48, borderRadius: 999, background: "var(--accent-softer)", border: "1px solid var(--accent-soft)", color: "var(--accent)", marginBottom: 8 }}>
        <Icon name={icon} size={20} strokeWidth={1.5} />
      </span>
      <span style={{ font: "600 14px var(--font-ui)", color: "var(--text-ink)" }}>{message}</span>
      {detail && <span style={{ font: "400 13px/1.5 var(--font-ui)", color: "var(--text-secondary)", maxWidth: 380 }}>{detail}</span>}
      {action && <span style={{ marginTop: 14 }}>{action}</span>}
    </div>
  );
}
