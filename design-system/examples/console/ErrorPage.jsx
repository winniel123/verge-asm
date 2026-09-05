import React from "react";
import { Button } from "../../components/forms/Button.jsx";
import { CopyValue } from "../../components/display/CopyValue.jsx";
import { Icon } from "../../components/media/Icon.jsx";

const KINDS = {
  "404": { icon: "compass", code: "404", title: "Page not found", body: "The address doesn't match any screen in this deployment. The link may predate a rename.", action: "Back to dashboard" },
  "403": { icon: "lock", code: "403", title: "Access denied", body: "Your role can't view this. An admin can widen it in Settings \u2192 Team.", action: "Back to dashboard" },
  "500": { icon: "server-crash", code: "500", title: "Something broke", body: "The console hit an unexpected error. The incident is logged on the host with the ID below.", action: "Back to dashboard" },
  "missing-subject": { icon: "scan-search", subject: "stagng-5.acmecorp.io", title: "No such subject", body: "No subject is keyed under that name. Nothing has ever measured it into the estate \u2014 this is not a withdrawn subject, which would still be reachable here by its own key.", action: "Back to inventory" },
  "missing-run": { icon: "history", subject: "run #1408", title: "No such run", body: "No dispatch is keyed under that id. A run is one fan-out of jobs; recent dispatches are listed on Settings \u2192 Scans, and each change event on Drift links its batch.", action: "Back to drift" },
  "forbidden": { icon: "lock", code: "403", title: "Admin only", body: "Settings is where declared acts live \u2014 seeds, scans, channels, team. Your role reads everything and changes nothing; an admin can change roles in Settings \u2192 Team.", action: "Back to dashboard" },
};

export function ErrorPage({ kind = "404", subject, onHome }) {
  const k = KINDS[kind] || KINDS["404"];
  const key = subject || k.subject;
  return (
    <main data-screen-label={"Error " + kind} style={{ minHeight: "70vh", display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center", gap: 18, padding: 32, textAlign: "center" }}>
      <span style={{ display: "inline-flex", alignItems: "center", justifyContent: "center", width: 52, height: 52, borderRadius: 16, background: "var(--surface-sunken)", border: "1px solid var(--border-default)", color: "var(--text-muted)" }}>
        <Icon name={k.icon} size={24} />
      </span>
      {k.code ? (
        <span style={{ font: "600 32px var(--font-mono)", letterSpacing: "0.04em", color: "var(--text-muted)" }}>{k.code}</span>
      ) : (
        <span style={{ font: "600 21px var(--font-mono)", letterSpacing: "-0.01em", color: "var(--text-ink)", wordBreak: "break-all", maxWidth: 560 }}>{key}</span>
      )}
      <div style={{ display: "flex", flexDirection: "column", gap: 8, alignItems: "center" }}>
        <h1 style={{ margin: 0, font: "600 18px var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)" }}>{k.title}</h1>
        <p style={{ margin: 0, font: "400 13px/1.6 var(--font-ui)", color: "var(--text-muted)", maxWidth: 440 }}>{k.body}</p>
      </div>
      {kind === "500" && <CopyValue value="err_9f3ka72c" />}
      <Button onClick={onHome}>{k.action}</Button>
    </main>
  );
}
