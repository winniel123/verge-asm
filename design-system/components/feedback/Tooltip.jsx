import React from "react";

export function Tooltip({ label, side = "top", style, children }) {
  const [show, setShow] = React.useState(false);
  const pos = side === "bottom"
    ? { top: "calc(100% + 6px)", left: "50%", transform: "translateX(-50%)" }
    : { bottom: "calc(100% + 6px)", left: "50%", transform: "translateX(-50%)" };
  return (
    <span style={{ position: "relative", display: "inline-flex", ...style }}
      onMouseEnter={() => setShow(true)} onMouseLeave={() => setShow(false)}>
      {children}
      {show && (
        <span role="tooltip" style={{
          position: "absolute", ...pos, zIndex: 60, whiteSpace: "nowrap",
          background: "var(--surface-ink)", color: "var(--text-on-ink)", border: "1px solid var(--border-ink)",
          boxShadow: "var(--shadow-hard-sm)", padding: "5px 9px", font: "500 11px/1.2 var(--font-mono)",
        }}>{label}</span>
      )}
    </span>
  );
}
