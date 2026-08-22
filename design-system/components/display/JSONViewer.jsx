import React from "react";

function Caret({ open }) {
  return (
    <svg viewBox="0 0 12 12" width="9" height="9" aria-hidden="true" style={{ marginRight: 5, transform: open ? "rotate(90deg)" : "none", transition: "transform var(--dur-fast) var(--ease-out)", flex: "none" }}>
      <path d="M4 2.5L8.5 6L4 9.5" fill="none" stroke="var(--text-muted)" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round"></path>
    </svg>
  );
}

function Node({ k, v, depth, defaultDepth, last }) {
  const isObj = v !== null && typeof v === "object";
  const [open, setOpen] = React.useState(depth < defaultDepth);
  const pad = { paddingLeft: depth * 16 };
  const key = k != null ? <span><span style={{ color: "var(--text-secondary)" }}>"{k}"</span><span style={{ color: "var(--text-muted)" }}>: </span></span> : null;
  const comma = last ? "" : ",";
  if (!isObj) {
    const color = typeof v === "string" ? "var(--ok)" : typeof v === "number" ? "var(--chart-1)" : typeof v === "boolean" ? "var(--warn)" : "var(--text-muted)";
    return <div style={pad}>{key}<span style={{ color }}>{JSON.stringify(v)}</span><span style={{ color: "var(--text-muted)" }}>{comma}</span></div>;
  }
  const arr = Array.isArray(v);
  const entries = arr ? v.map((x, i) => [i, x]) : Object.keys(v).map((kk) => [kk, v[kk]]);
  const o = arr ? "[" : "{", c = arr ? "]" : "}";
  return (
    <div>
      <div onClick={() => setOpen(!open)} style={{ ...pad, cursor: "pointer", userSelect: "none", display: "flex", alignItems: "center" }}>
        <Caret open={open} />
        <span>{key}<span style={{ color: "var(--text-body)" }}>{o}</span>
          {!open && <span style={{ color: "var(--text-muted)" }}> {entries.length} {arr ? (entries.length === 1 ? "item" : "items") : (entries.length === 1 ? "key" : "keys")} {c}{comma}</span>}
        </span>
      </div>
      {open && entries.map((e, i) => <Node key={e[0]} k={arr ? null : e[0]} v={e[1]} depth={depth + 1} defaultDepth={defaultDepth} last={i === entries.length - 1} />)}
      {open && <div style={{ ...pad, marginLeft: 14 }}><span style={{ color: "var(--text-body)" }}>{c}</span><span style={{ color: "var(--text-muted)" }}>{comma}</span></div>}
    </div>
  );
}

/* Collapsible JSON tree for payload inspection (delivery records, API examples).
   Strings ok-green, numbers chart-1, booleans warn, null muted — value types, never severity. */
export function JSONViewer({ data, defaultDepth = 1, label, style }) {
  return (
    <div aria-label={label} style={{ font: "400 12.5px/1.75 var(--font-mono)", background: "var(--surface-sunken)", border: "1px solid var(--border-default)", borderRadius: 12, padding: "10px 14px", overflowX: "auto", ...style }}>
      <Node k={null} v={data} depth={0} defaultDepth={defaultDepth} last />
    </div>
  );
}
