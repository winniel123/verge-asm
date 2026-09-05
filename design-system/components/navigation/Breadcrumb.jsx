import React from "react";

export function Breadcrumb({ items = [], style }) {
  return (
    <nav aria-label="Breadcrumb" style={{ display: "flex", alignItems: "center", gap: 8, font: "500 11px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", ...style }}>
      {items.map((it, i) => {
        const last = i === items.length - 1;
        return (
          <React.Fragment key={i}>
            {i > 0 && <span aria-hidden="true" style={{ color: "var(--text-muted)" }}>/</span>}
            {last
              ? <span aria-current="page" style={{ color: "var(--text-body)" }}>{it.label}</span>
              : <a href={it.href || "#"} onClick={it.onClick} style={{ color: "var(--text-muted)", textDecoration: "none" }}
                  onMouseEnter={(e) => e.currentTarget.style.color = "var(--link)"}
                  onMouseLeave={(e) => e.currentTarget.style.color = "var(--text-muted)"}>{it.label}</a>}
          </React.Fragment>
        );
      })}
    </nav>
  );
}
