import React from "react";

export function Card({ microLabel, title, action, children, footer, pad = 20, overflow, style }) {
  const ov = overflow || (pad === 0 ? "hidden" : "visible"); // flush tables need clipping for rounded corners; padded cards must not clip menus/selects
  return (
    <section style={{ background: "var(--surface)", border: "1px solid var(--border-default)", borderRadius: 16, boxShadow: "var(--shadow-sm)", display: "flex", flexDirection: "column", overflow: ov, ...style }}>
      {(microLabel || title || action) && (
        <header style={{ display: "flex", alignItems: "center", gap: 12, padding: (pad || 20) + "px " + (pad || 20) + "px " + (pad ? 0 : 14) + "px" }}>
          <div style={{ display: "flex", flexDirection: "column", gap: 3, minWidth: 0 }}>
            {microLabel && <span style={{ font: "500 11px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--text-muted)" }}>{microLabel}</span>}
            {title && <h3 style={{ margin: 0, font: "600 15px var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)" }}>{title}</h3>}
          </div>
          {action && <div style={{ marginLeft: "auto", display: "flex", gap: 8, flex: "none" }}>{action}</div>}
        </header>
      )}
      <div style={{ padding: pad, flex: 1, minHeight: 0 }}>{children}</div>
      {footer && <footer style={{ padding: "12px " + pad + "px", borderTop: "1px solid var(--row-sep)", display: "flex", alignItems: "center", gap: 12 }}>{footer}</footer>}
    </section>
  );
}
