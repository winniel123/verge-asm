import React from "react";

const MARKS = { okta: "O", entra: "E", google: "G", github: "GH", saml: "SA", oidc: "ID" };
const NAMES = { okta: "Okta", entra: "Microsoft Entra", google: "Google", github: "GitHub", saml: "SSO (SAML)", oidc: "SSO (OIDC)" };
export function IdPButton({ provider = "saml", label, onClick, full = true, style }) {
  const [hover, setHover] = React.useState(false);
  return (
    <button type="button" onClick={onClick} onMouseEnter={() => setHover(true)} onMouseLeave={() => setHover(false)}
      style={{ display: "inline-flex", alignItems: "center", justifyContent: "center", gap: 10, width: full ? "100%" : undefined, height: 38, padding: "0 16px", background: hover ? "var(--surface-sunken)" : "var(--surface)", border: "1px solid var(--border-strong)", borderRadius: 10, font: "500 13px var(--font-ui)", color: "var(--text-ink)", cursor: "pointer", transition: "background 120ms ease", ...style }}>
      <span aria-hidden="true" style={{ display: "inline-flex", alignItems: "center", justifyContent: "center", width: 20, height: 20, borderRadius: 6, background: "var(--surface-sunken)", border: "1px solid var(--border-default)", font: "600 9.5px var(--font-mono)", color: "var(--text-secondary)", flex: "none" }}>{MARKS[provider] || "?"}</span>
      {label || "Continue with " + (NAMES[provider] || provider)}
    </button>
  );
}
