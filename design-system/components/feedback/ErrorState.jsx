import React from "react";
import { Icon } from "../media/Icon.jsx";
import { Button } from "../forms/Button.jsx";

/* Failure sibling of EmptyState: fact + retry. */
export function ErrorState({ icon = "alert-triangle", message, detail, retryLabel = "Retry", onRetry, style }) {
  return (
    <div style={{ display: "flex", flexDirection: "column", alignItems: "center", textAlign: "center", gap: 6, padding: "48px 24px", fontFamily: "var(--font-ui)", ...style }}>
      <span style={{ display: "inline-flex", alignItems: "center", justifyContent: "center", width: 48, height: 48, borderRadius: 999, background: "var(--danger-soft)", border: "1px solid var(--danger-border)", color: "var(--danger)", marginBottom: 8 }}>
        <Icon name={icon} size={20} strokeWidth={1.5} />
      </span>
      <span style={{ font: "600 14px var(--font-ui)", color: "var(--text-ink)" }}>{message}</span>
      {detail && <span style={{ font: "400 13px/1.5 var(--font-ui)", color: "var(--text-secondary)", maxWidth: 380 }}>{detail}</span>}
      {onRetry && <span style={{ marginTop: 14 }}><Button variant="secondary" icon={<Icon name="refresh-cw" size={14} />} onClick={onRetry}>{retryLabel}</Button></span>}
    </div>
  );
}
