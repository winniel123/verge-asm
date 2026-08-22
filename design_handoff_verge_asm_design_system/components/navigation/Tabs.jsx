import React from "react";

function Tab({ tab, active, onClick }) {
  const [hov, setHov] = React.useState(false);
  return (
    <button type="button" onClick={onClick} onMouseEnter={() => setHov(true)} onMouseLeave={() => setHov(false)}
      style={{ position: "relative", display: "inline-flex", alignItems: "center", gap: 6, padding: "0 2px 10px", border: "none", background: "transparent", cursor: "pointer", fontFamily: "var(--font-ui)", fontSize: 13, fontWeight: active ? 600 : 500, color: active ? "var(--text-ink)" : hov ? "var(--text-body)" : "var(--text-secondary)", transition: "color var(--dur-fast) var(--ease-out)" }}>
      {tab.label}
      {tab.count != null && <span style={{ font: "500 10.5px var(--font-mono)", padding: "1px 6px", borderRadius: 999, background: "var(--surface-sunken)", color: "var(--text-secondary)" }}>{tab.count}</span>}
      {active && <span style={{ position: "absolute", left: 0, right: 0, bottom: -1, height: 3, borderRadius: 999, background: "var(--accent)" }} />}
    </button>
  );
}

export function Tabs({ tabs = [], active, onChange, style }) {
  return (
    <div style={{ display: "flex", gap: 20, borderBottom: "1px solid var(--border-default)", ...style }}>
      {tabs.map((t) => <Tab key={t.id} tab={t} active={t.id === active} onClick={() => onChange && onChange(t.id)} />)}
    </div>
  );
}
