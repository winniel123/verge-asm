import React from "react";
import { Badge } from "./Badge.jsx";
import { CopyValue } from "./CopyValue.jsx";
import { Button } from "../forms/Button.jsx";
import { Icon } from "../media/Icon.jsx";

/* IdP signing certificate: identity, expiry as operational state (ok/warn/danger — never severity),
   fingerprint as the verifiable value. Expired certs keep working visually but say so plainly. */
export function CertificateCard({ name, role = "IdP signing", issuer, algorithm, notAfter, daysLeft, fingerprint, onReplace, style }) {
  const expired = typeof daysLeft === "number" && daysLeft <= 0;
  const expiring = !expired && typeof daysLeft === "number" && daysLeft <= 30;
  const tone = expired ? "danger" : expiring ? "warn" : "ok";
  const label = expired ? "expired " + Math.abs(daysLeft) + "d ago" : expiring ? "expires in " + daysLeft + "d" : "valid \u00b7 " + daysLeft + "d";
  return (
    <section style={{ background: "var(--surface)", border: "1px solid " + (expired ? "var(--danger)" : "var(--border-default)"), borderRadius: 16, boxShadow: "var(--shadow-sm)", padding: 16, display: "flex", flexDirection: "column", gap: 10, minWidth: 0, fontFamily: "var(--font-ui)", ...style }}>
      <div style={{ display: "flex", alignItems: "center", gap: 8, flexWrap: "wrap", rowGap: 6 }}>
        <Icon name="file-key-2" size={15} style={{ color: "var(--text-muted)" }} />
        <span style={{ font: "600 13px var(--font-mono)", color: "var(--text-ink)", whiteSpace: "nowrap" }}>{name}</span>
        <span style={{ font: "500 10.5px var(--font-mono)", letterSpacing: "0.06em", textTransform: "uppercase", color: "var(--text-muted)" }}>{role}</span>
        <span style={{ marginLeft: "auto" }}><Badge tone={tone} dot>{label}</Badge></span>
      </div>
      <div style={{ display: "flex", alignItems: "center", gap: 12, font: "400 11.5px var(--font-mono)", color: "var(--text-secondary)", flexWrap: "wrap" }}>
        {issuer && <span style={{ minWidth: 0, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>{issuer}</span>}
        {algorithm && <span style={{ flex: "none" }}>{algorithm}</span>}
        {notAfter && <span style={{ flex: "none", color: "var(--text-muted)" }}>until {notAfter}</span>}
      </div>
      {fingerprint && <CopyValue value={fingerprint} />}
      {expired && <span style={{ font: "400 11.5px/1.5 var(--font-ui)", color: "var(--text-muted)" }}>Assertions signed with this certificate are no longer trusted — sign-ins fall back to local accounts.</span>}
      {onReplace && <div><Button size="sm" variant="secondary" onClick={onReplace}>Replace certificate</Button></div>}
    </section>
  );
}
