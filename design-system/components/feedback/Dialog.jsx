import React from "react";
import { IconButton } from "../forms/IconButton.jsx";

export function Dialog({ open, title, description, children, footer, onClose, width = 480 }) {
  const panelRef = React.useRef(null);
  React.useEffect(() => {
    if (!open) return;
    const prev = document.activeElement;
    const panel = panelRef.current;
    if (panel && !panel.contains(document.activeElement)) panel.focus();
    const onTab = (e) => {
      if (e.key !== "Tab" || !panelRef.current) return;
      const els = Array.prototype.filter.call(
        panelRef.current.querySelectorAll('button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'),
        (el) => el.offsetParent !== null || el === document.activeElement
      );
      if (!els.length) return;
      const first = els[0], last = els[els.length - 1];
      if (e.shiftKey && document.activeElement === first) { e.preventDefault(); last.focus(); }
      else if (!e.shiftKey && document.activeElement === last) { e.preventDefault(); first.focus(); }
    };
    document.addEventListener("keydown", onTab);
    return () => { document.removeEventListener("keydown", onTab); if (prev && prev.focus) prev.focus(); };
  }, [open]);
  React.useEffect(() => {
    if (!open) return;
    const onKey = (e) => { if (e.key === "Escape" && onClose) onClose(); };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [open, onClose]);
  if (!open) return null;
  return (
    <div onClick={onClose} style={{ position: "fixed", inset: 0, zIndex: 100, background: "rgba(21,18,15,0.4)", display: "flex", alignItems: "center", justifyContent: "center", padding: 24, animation: "vg-fade-in var(--dur-base) var(--ease-out)" }}>
      <div role="dialog" aria-modal="true" ref={panelRef} tabIndex={-1} onClick={(e) => e.stopPropagation()}
        style={{ width: "min(" + width + "px, 92%)", maxHeight: "85%", overflowY: "auto", overflowX: "hidden", background: "var(--surface)", borderRadius: 24, boxShadow: "var(--shadow-lg)", border: "1px solid var(--border-default)", padding: 24, animation: "vg-pop-in var(--dur-slow) var(--ease-out)", fontFamily: "var(--font-ui)", outline: "none" }}>
        <header style={{ display: "flex", alignItems: "flex-start", gap: 12 }}>
          <div style={{ display: "flex", flexDirection: "column", gap: 4, minWidth: 0 }}>
            <h2 style={{ margin: 0, font: "600 16px var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)" }}>{title}</h2>
            {description && <p style={{ margin: 0, font: "400 13px/1.5 var(--font-ui)", color: "var(--text-secondary)" }}>{description}</p>}
          </div>
          {onClose && <IconButton icon="x" label="Close" style={{ marginLeft: "auto", flex: "none" }} onClick={onClose} />}
        </header>
        {children && <div style={{ marginTop: 16 }}>{children}</div>}
        {footer && <footer style={{ marginTop: 24, display: "flex", justifyContent: "flex-end", gap: 8 }}>{footer}</footer>}
      </div>
    </div>
  );
}
