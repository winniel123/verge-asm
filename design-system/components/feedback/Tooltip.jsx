import React from "react";

export function Tooltip({ content, side = "top", mono, children, style }) {
  const [show, setShow] = React.useState(false);
  const pos = {
    top: { bottom: "100%", left: "50%", transform: "translate(-50%, -8px)" },
    bottom: { top: "100%", left: "50%", transform: "translate(-50%, 8px)" },
    left: { right: "100%", top: "50%", transform: "translate(-8px, -50%)" },
    right: { left: "100%", top: "50%", transform: "translate(8px, -50%)" },
  }[side];
  return (
    <span style={{ position: "relative", display: "inline-flex", ...style }}
      onMouseEnter={() => setShow(true)} onMouseLeave={() => setShow(false)}
      onFocus={() => setShow(true)} onBlur={() => setShow(false)}>
      {children}
      {show && (
        <span role="tooltip" style={{ position: "absolute", zIndex: 90, whiteSpace: "nowrap", padding: "5px 9px", borderRadius: 8, background: "var(--surface-inverted)", color: "var(--text-on-inverted)", font: (mono ? "400 11px var(--font-mono)" : "500 11.5px var(--font-ui)"), boxShadow: "var(--shadow-md)", animation: "vg-fade-in var(--dur-fast) var(--ease-out)", pointerEvents: "none", ...pos }}>
          {content}
        </span>
      )}
    </span>
  );
}
