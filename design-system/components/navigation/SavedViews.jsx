import React from "react";

export function SavedViews({ views = [], activeId, onSelect, dirty, onSave, style }) {
  const [hov, setHov] = React.useState(null);
  return (
    <div style={{ display: "flex", alignItems: "center", gap: 6, flexWrap: "wrap", fontFamily: "var(--font-ui)", ...style }}>
      {views.map((v) => {
        const active = v.id === activeId;
        return (
          <button key={v.id} type="button" onClick={() => onSelect && onSelect(v.id)}
            onMouseEnter={() => setHov(v.id)} onMouseLeave={() => setHov(null)}
            style={{ display: "inline-flex", alignItems: "center", gap: 7, height: 28, padding: "0 12px", borderRadius: 999, cursor: "pointer",
              border: "1px solid " + (active ? "color-mix(in srgb, var(--accent) 35%, transparent)" : hov === v.id ? "var(--border-strong)" : "var(--border-default)"),
              background: active ? "var(--accent-soft)" : hov === v.id ? "var(--surface-sunken)" : "transparent",
              color: active ? "var(--link)" : "var(--text-secondary)", font: "500 12.5px var(--font-ui)",
              transition: "background var(--dur-fast) var(--ease-out), border-color var(--dur-fast) var(--ease-out)" }}>
            {v.label}
            {v.count !== undefined && <span style={{ font: "500 11px var(--font-mono)", color: active ? "var(--link)" : "var(--text-muted)" }}>{v.count.toLocaleString("en-US")}</span>}
          </button>
        );
      })}
      {dirty && onSave && (
        <React.Fragment>
          <span style={{ width: 1, height: 16, background: "var(--border-default)", margin: "0 4px" }}></span>
          <button type="button" onClick={onSave}
            onMouseEnter={() => setHov("__save")} onMouseLeave={() => setHov(null)}
            style={{ display: "inline-flex", alignItems: "center", gap: 6, height: 28, padding: "0 12px", borderRadius: 999, cursor: "pointer",
              border: "1px dashed " + (hov === "__save" ? "var(--accent)" : "var(--border-strong)"),
              background: hov === "__save" ? "var(--accent-soft)" : "transparent",
              color: hov === "__save" ? "var(--link)" : "var(--text-secondary)", font: "500 12.5px var(--font-ui)",
              transition: "background var(--dur-fast) var(--ease-out), border-color var(--dur-fast) var(--ease-out)" }}>
            <svg viewBox="0 0 16 16" width="12" height="12"><path d="M8 3v10M3 8h10" fill="none" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round"></path></svg>
            Save view
          </button>
        </React.Fragment>
      )}
    </div>
  );
}
