import React from "react";
import { Logo } from "../media/Logo.jsx";

const L = ({ href = "#", children }) => <a href={href} style={{ color: "var(--text-secondary)", textDecoration: "none", font: "400 12px var(--font-ui)" }}>{children}</a>;

export function Footer({ variant = "console", version = "v0.9.2", style }) {
  if (variant === "marketing") {
    return (
      <footer style={{ borderTop: "1px solid var(--border-default)", padding: "40px 32px", background: "var(--surface)", fontFamily: "var(--font-ui)", ...style }}>
        <div style={{ maxWidth: 1120, margin: "0 auto", display: "flex", gap: 48, flexWrap: "wrap", alignItems: "flex-start" }}>
          <div style={{ display: "flex", flexDirection: "column", gap: 8, marginRight: "auto" }}>
            <Logo size={20} wordmarkSize={17} />
            <span style={{ font: "400 12px var(--font-ui)", color: "var(--text-muted)" }}>Open-source attack surface management. Self-hosted, AGPL-3.0.</span>
          </div>
          {[["Product", ["Docs", "Install", "Changelog"]], ["Project", ["GitHub", "License", "Security policy"]]].map(([h, links]) => (
            <div key={h} style={{ display: "flex", flexDirection: "column", gap: 8 }}>
              <span style={{ font: "500 11px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--text-muted)" }}>{h}</span>
              {links.map((l) => <L key={l}>{l}</L>)}
            </div>
          ))}
        </div>
        <div style={{ maxWidth: 1120, margin: "32px auto 0", display: "flex", gap: 16, alignItems: "center", font: "400 11px var(--font-mono)", color: "var(--text-muted)" }}>
          <span>{version}</span><span>AGPL-3.0</span><span style={{ marginLeft: "auto" }}>you host it, you own it</span>
        </div>
      </footer>
    );
  }
  return (
    <footer style={{ display: "flex", alignItems: "center", gap: 16, padding: "14px 32px", borderTop: "1px solid var(--border-default)", fontFamily: "var(--font-ui)", ...style }}>
      <span style={{ font: "400 12px var(--font-ui)", color: "var(--text-muted)" }}>verge asm · self-hosted · AGPL-3.0</span>
      <span style={{ marginLeft: "auto", display: "flex", gap: 16, alignItems: "center" }}>
        <L>Docs</L><L>GitHub</L>
        <span style={{ font: "400 11px var(--font-mono)", color: "var(--text-muted)" }}>{version}</span>
      </span>
    </footer>
  );
}
