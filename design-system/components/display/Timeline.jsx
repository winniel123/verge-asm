import React from "react";

const TONES = { neutral: "var(--neutral-400)", accent: "var(--accent)", ok: "var(--ok-solid)", warn: "var(--warn-solid)", danger: "var(--danger-solid)" };

/* Vertical event history. events: [{ title, detail?, time, tone?, mono?, dotColor?, content? }] — newest first.
   Pass groups: [{id, label, meta?, events, defaultCollapsed?}] for collapsible per-batch sections. */
function TimelineList({ events = [], style }) {
  return (
    <div style={{ display: "flex", flexDirection: "column", fontFamily: "var(--font-ui)", ...style }}>
      {events.map((e, i) => (
        <div key={i} style={{ display: "flex", gap: 12 }}>
          <div style={{ display: "flex", flexDirection: "column", alignItems: "center", flex: "none", width: 10 }}>
            <span style={{ width: 8, height: 8, borderRadius: 999, background: e.dotColor || TONES[e.tone] || TONES.neutral, marginTop: 5, flex: "none", boxShadow: "0 0 0 2px var(--surface)" }} />
            {i < events.length - 1 && <span style={{ width: 1, flex: 1, background: "var(--border-default)", margin: "3px 0" }} />}
          </div>
          <div style={{ display: "flex", flexDirection: "column", gap: 2, paddingBottom: i < events.length - 1 ? 16 : 0, minWidth: 0 }}>
            <div style={{ display: "flex", alignItems: "flex-start", gap: 10 }}>
              <span style={{ font: "500 13px/1.45 var(--font-ui)", color: "var(--text-ink)", minWidth: 0 }}>{e.title}</span>
              <span style={{ font: "400 11px var(--font-mono)", color: "var(--text-muted)", whiteSpace: "nowrap", paddingTop: 3 }}>{e.time}</span>
            </div>
            {e.detail && <span style={{ font: (e.mono ? "400 12px var(--font-mono)" : "400 12.5px/1.5 var(--font-ui)"), color: "var(--text-secondary)", overflowWrap: "anywhere" }}>{e.detail}</span>}
            {e.content && <div style={{ marginTop: 8 }}>{e.content}</div>}
          </div>
        </div>
      ))}
    </div>
  );
}

export function Timeline({ events = [], groups, style }) {
  const [closed, setClosed] = React.useState(() => (groups || []).filter((g) => g.defaultCollapsed).map((g) => g.id));
  if (!groups) return <TimelineList events={events} style={style} />;
  const toggle = (id) => setClosed((c) => (c.indexOf(id) !== -1 ? c.filter((x) => x !== id) : c.concat(id)));
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 4, fontFamily: "var(--font-ui)", ...style }}>
      {groups.map((g) => {
        const isClosed = closed.indexOf(g.id) !== -1;
        return (
          <div key={g.id}>
            <button type="button" aria-expanded={!isClosed} onClick={() => toggle(g.id)}
              style={{ display: "flex", alignItems: "center", gap: 8, width: "100%", padding: "8px 2px", border: "none", background: "transparent", cursor: "pointer", textAlign: "left" }}>
              <svg viewBox="0 0 16 16" width="13" height="13" style={{ color: "var(--text-muted)", transform: isClosed ? "rotate(-90deg)" : "none", transition: "transform var(--dur-base) var(--ease-out)", flex: "none" }}>
                <path d="M4 6l4 4 4-4" fill="none" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round"></path>
              </svg>
              <span style={{ font: "600 12px var(--font-mono)", color: "var(--text-ink)" }}>{g.label}</span>
              {g.meta && <span style={{ font: "400 11px var(--font-mono)", color: "var(--text-muted)" }}>{g.meta}</span>}
              <span style={{ marginLeft: "auto", font: "500 10.5px var(--font-mono)", padding: "1px 7px", borderRadius: 999, background: "var(--surface-sunken)", color: "var(--text-secondary)" }}>{g.events.length}</span>
            </button>
            <div style={{ display: "grid", gridTemplateRows: isClosed ? "0fr" : "1fr", transition: "grid-template-rows var(--dur-slow) var(--ease-out)" }}>
              <div style={{ overflow: "hidden", minHeight: 0 }}>
                <div style={{ padding: "4px 0 12px 24px" }}><TimelineList events={g.events} /></div>
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
}
