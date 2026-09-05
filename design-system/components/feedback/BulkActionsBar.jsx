import React from "react";
import { Icon } from "../media/Icon.jsx";

function BarBtn({ a }) {
  const [hov, setHov] = React.useState(false);
  const danger = a.tone === "danger";
  return (
    <button type="button" onClick={a.onClick} onMouseEnter={() => setHov(true)} onMouseLeave={() => setHov(false)}
      style={{ display: "inline-flex", alignItems: "center", gap: 6, height: 28, padding: "0 10px", border: "none", borderRadius: 999, cursor: "pointer", background: hov ? "rgba(255,255,255,0.12)" : "transparent", color: danger ? "#f5a49b" : "var(--text-on-inverted)", font: "500 12.5px var(--font-ui)", transition: "background var(--dur-fast) var(--ease-out)", whiteSpace: "nowrap" }}>
      {a.icon && <Icon name={a.icon} size={13} />}
      {a.label}
    </button>
  );
}

export function BulkActionsBar({ count = 0, itemLabel = "selected", actions = [], onClear, floating = true, style }) {
  if (!count) return null;
  const bar = (
    <div role="status" style={{ display: "inline-flex", alignItems: "center", gap: 6, padding: "8px 10px 8px 16px", background: "var(--surface-inverted)", borderRadius: 999, boxShadow: "var(--shadow-lg)", pointerEvents: "auto", animation: floating ? "vg-toast-in var(--dur-slow) var(--ease-out)" : "none", ...style }}>
      <span style={{ display: "inline-flex", alignItems: "baseline", gap: 5, minWidth: 76 }}>
        <span style={{ font: "600 12.5px var(--font-mono)", color: "var(--text-on-inverted)" }}>{count.toLocaleString("en-US")}</span>
        <span style={{ font: "400 12.5px var(--font-ui)", color: "var(--neutral-400)" }}>{itemLabel}</span>
      </span>
      <span style={{ width: 1, alignSelf: "stretch", background: "rgba(255,255,255,0.16)", margin: "2px 6px" }} />
      {actions.map((a) => <BarBtn key={a.label} a={a} />)}
      {onClear && (
        <button type="button" aria-label="Clear selection" onClick={onClear}
          style={{ display: "inline-flex", alignItems: "center", justifyContent: "center", width: 24, height: 24, marginLeft: 4, border: "none", borderRadius: 999, background: "transparent", color: "var(--neutral-500)", cursor: "pointer" }}>
          <Icon name="x" size={13} />
        </button>
      )}
    </div>
  );
  if (!floating) return bar;
  return (
    <div style={{ position: "fixed", left: 0, right: 0, bottom: 24, zIndex: 105, display: "flex", justifyContent: "center", pointerEvents: "none" }}>{bar}</div>
  );
}
