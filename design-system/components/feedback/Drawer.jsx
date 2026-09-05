import React from "react";
import { IconButton } from "../forms/IconButton.jsx";

export function Drawer({ open, title, description, children, footer, onClose, width = 440, side = "right" }) {
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
  const [present, setPresent] = React.useState(!!open);
  React.useEffect(() => {
    if (open) { setPresent(true); return; }
    // 280 stays in step with --dur-slow in tokens/motion.css
    const t = setTimeout(() => setPresent(false), 280);
    return () => clearTimeout(t);
  }, [open]);
  if (!present) return null;
  const closing = !open;
  const fromRight = side !== "left";
  return (
    <div onClick={closing ? undefined : onClose} style={{ position: "fixed", inset: 0, zIndex: 100, background: "rgba(21,18,15,0.28)", pointerEvents: closing ? "none" : "auto", animation: closing ? "vg-fade-out var(--dur-slow) var(--ease-out) forwards" : "vg-fade-in var(--dur-base) var(--ease-out)" }}>
      <aside role="dialog" aria-modal="true" aria-label={title} ref={panelRef} tabIndex={-1} onClick={(e) => e.stopPropagation()}
        style={{ position: "absolute", top: 0, bottom: 0, [fromRight ? "right" : "left"]: 0, width: "min(" + width + "px, 92%)", background: "var(--surface)", borderLeft: fromRight ? "1px solid var(--border-default)" : "none", borderRight: fromRight ? "none" : "1px solid var(--border-default)", boxShadow: "var(--shadow-lg)", display: "flex", flexDirection: "column", outline: "none", animation: (closing ? (fromRight ? "vg-drawer-out-right" : "vg-drawer-out-left") + " var(--dur-slow) var(--ease-out) forwards" : (fromRight ? "vg-drawer-in-right" : "vg-drawer-in-left") + " var(--dur-slow) var(--ease-out)") }}>
        <header style={{ display: "flex", alignItems: "flex-start", gap: 12, padding: "20px 24px 0", flex: "none" }}>
          <div style={{ display: "flex", flexDirection: "column", gap: 4, minWidth: 0 }}>
            <h2 style={{ margin: 0, font: "600 16px var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)" }}>{title}</h2>
            {description && <p style={{ margin: 0, font: "400 13px/1.5 var(--font-ui)", color: "var(--text-secondary)" }}>{description}</p>}
          </div>
          {onClose && <IconButton icon="x" label="Close" style={{ marginLeft: "auto", flex: "none" }} onClick={onClose} />}
        </header>
        <div style={{ flex: 1, minHeight: 0, overflowY: "auto", overflowX: "hidden", padding: "16px 24px 24px" }}>{children}</div>
        {footer && <footer style={{ flex: "none", display: "flex", justifyContent: "flex-end", gap: 8, padding: "14px 24px", borderTop: "1px solid var(--row-sep)", background: "var(--surface)" }}>{footer}</footer>}
      </aside>
    </div>
  );
}
