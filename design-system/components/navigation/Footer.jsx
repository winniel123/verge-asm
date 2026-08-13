import React from "react";

export function Footer({ version = "v0.9.2", license = "AGPL-3.0", links = [], stars, right, style }) {
  return (
    <div style={{ display: "flex", alignItems: "center", gap: 18, padding: "12px 24px", borderTop: "1px solid var(--border)", background: "var(--surface)", font: "400 11px/1.2 var(--font-mono)", color: "var(--text-muted)", boxSizing: "border-box", ...style }}>
      <span>{version}</span><span>{license}</span>
      {links.map((l) => <a key={l.label} href={l.href || "#"} style={{ font: "400 11px var(--font-mono)" }}>{l.label}</a>)}
      <span style={{ marginLeft: "auto", display: "flex", alignItems: "center", gap: 10 }}>
        {stars != null && (
          <>
            <span style={{ border: "1px solid var(--border-ink)", padding: "4px 10px", color: "var(--ink)", fontWeight: 500 }}>★ Star</span>
            <span>{stars}</span>
          </>
        )}
        {right}
      </span>
    </div>
  );
}
