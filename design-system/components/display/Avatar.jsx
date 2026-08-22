import React from "react";

/* Initials chip — no photos in the console. name derives initials; tone dot marks presence/role. */
export function Avatar({ name = "", size = 28, dot, style }) {
  const initials = name.trim().split(/\s+/).map((w) => w[0]).slice(0, 2).join("").toUpperCase() || "?";
  const DOTS = { ok: "var(--ok-solid)", warn: "var(--warn-solid)", danger: "var(--danger-solid)", accent: "var(--accent)" };
  return (
    <span title={name} style={{ position: "relative", display: "inline-flex", flex: "none", ...style }}>
      <span style={{ display: "inline-flex", alignItems: "center", justifyContent: "center", width: size, height: size, borderRadius: 999, background: "var(--surface-sunken)", border: "1px solid var(--border-default)", font: "600 " + Math.max(9, Math.round(size * 0.38)) + "px var(--font-mono)", color: "var(--text-secondary)" }}>{initials}</span>
      {dot && <span style={{ position: "absolute", right: -1, bottom: -1, width: Math.round(size * 0.3), height: Math.round(size * 0.3), borderRadius: 999, background: DOTS[dot] || DOTS.ok, boxShadow: "0 0 0 2px var(--surface)" }} />}
    </span>
  );
}
