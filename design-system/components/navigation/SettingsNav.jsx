import React from "react";
import { Icon } from "../media/Icon.jsx";

/* Sectioned settings navigation — micro-label sections, pill active item. */
export function SettingsNav({ sections = [], active, onNavigate, style }) {
  const [hov, setHov] = React.useState(null);
  return (
    <nav style={{ display: "flex", flexDirection: "column", gap: 20, fontFamily: "var(--font-ui)", ...style }}>
      {sections.map((s) => (
        <div key={s.label} style={{ display: "flex", flexDirection: "column", gap: 3 }}>
          <span style={{ font: "500 11px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--text-muted)", marginBottom: 4 }}>{s.label}</span>
          {s.items.map((it) => {
            const isActive = it.id === active;
            return (
              <button key={it.id} type="button" onClick={() => onNavigate && onNavigate(it.id)}
                onMouseEnter={() => setHov(it.id)} onMouseLeave={() => setHov(null)}
                style={{ display: "flex", alignItems: "center", gap: 9, height: 32, padding: "0 10px", border: "none", borderRadius: 8, textAlign: "left", cursor: "pointer", background: isActive ? "var(--accent-soft)" : hov === it.id ? "var(--surface-sunken)" : "transparent", color: isActive ? "var(--link)" : "var(--text-secondary)", font: (isActive ? "600" : "500") + " 13px var(--font-ui)", transition: "background var(--dur-fast) var(--ease-out)" }}>
                {it.icon && <Icon name={it.icon} size={14} />}
                {it.label}
              </button>
            );
          })}
        </div>
      ))}
    </nav>
  );
}
