import React from "react";
import { Button } from "../../components/forms/Button.jsx";
import { CopyValue } from "../../components/display/CopyValue.jsx";
import { Icon } from "../../components/media/Icon.jsx";

const KINDS = {
  "404": { icon: "compass", title: "Page not found", body: "The address doesn't match any screen in this deployment. The link may predate a rename." },
  "403": { icon: "lock", title: "Access denied", body: "Your role can't view this. An admin can widen it in Settings → Team." },
  "500": { icon: "server-crash", title: "Something broke", body: "The console hit an unexpected error. The incident is logged on the host with the ID below." },
};

/* Full-screen terminal states — 404 / 403 / 500. */
export function ErrorPage({ kind = "404", onHome }) {
  const k = KINDS[kind] || KINDS["404"];
  return (
    <main data-screen-label={"Error " + kind} style={{ minHeight: "70vh", display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center", gap: 18, padding: 32, textAlign: "center" }}>
      <span style={{ display: "inline-flex", alignItems: "center", justifyContent: "center", width: 52, height: 52, borderRadius: 16, background: "var(--surface-sunken)", border: "1px solid var(--border-default)", color: "var(--text-muted)" }}>
        <Icon name={k.icon} size={24} />
      </span>
      <span style={{ font: "600 32px var(--font-mono)", letterSpacing: "0.04em", color: "var(--text-muted)" }}>{kind}</span>
      <div style={{ display: "flex", flexDirection: "column", gap: 8, alignItems: "center" }}>
        <h1 style={{ margin: 0, font: "600 18px var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)" }}>{k.title}</h1>
        <p style={{ margin: 0, font: "400 13px/1.6 var(--font-ui)", color: "var(--text-muted)", maxWidth: 400 }}>{k.body}</p>
      </div>
      {kind === "500" && <CopyValue value="err_9f3ka72c" />}
      <Button onClick={onHome}>Back to dashboard</Button>
    </main>
  );
}
