import React from "react";
import { Card } from "../../components/display/Card.jsx";
import { Button } from "../../components/forms/Button.jsx";
import { Icon } from "../../components/media/Icon.jsx";

export function FirstRunChecklist({ onOpenScope, onOpenVantages, onRunScan }) {
  const steps = [
    { n: 1, done: true, title: "Declare your domain", detail: "acmecorp.io declared · a seed is a boundary, not a starting gun", action: null },
    { n: 2, done: false, title: "Upload a zone file", detail: "Enables removal detection — you stopped telling us becomes detectable", action: ["Upload zone", onOpenScope] },
    { n: 3, done: false, title: "Add an internet vantage", detail: "Exposure needs an outside observer, unconditionally", action: ["Provision prober", onOpenVantages] },
    { n: 4, done: false, title: "Run the first batch", detail: "Scans dispatch on cadence; kick the first one now", action: ["Run first batch", onRunScan], gated: true },
  ];
  const doneCount = steps.filter((s) => s.done).length;
  return (
    <main data-screen-label="First-run checklist" style={{ maxWidth: 760, margin: "0 auto", padding: "56px 32px", display: "flex", flexDirection: "column", gap: 20 }}>
      <header style={{ display: "flex", flexDirection: "column", gap: 6 }}>
        <h1 style={{ margin: 0, font: "600 24px var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)" }}>Welcome to Verge</h1>
        <span style={{ font: "400 13px/1.6 var(--font-ui)", color: "var(--text-muted)" }}>Each step unlocks a capability. Until they complete, the console stays honest about what it cannot conclude — exposure claims are withheld, never guessed.</span>
        <span style={{ font: "500 11px var(--font-mono)", letterSpacing: "0.06em", color: "var(--text-secondary)", marginTop: 6 }}>{doneCount} of 4 complete</span>
      </header>
      <Card pad={10}>
        <div style={{ display: "flex", flexDirection: "column" }}>
          {steps.map((s, i) => (
            <div key={s.n} style={{ display: "flex", alignItems: "center", gap: 14, padding: "16px 12px", borderTop: i ? "1px solid var(--row-sep)" : "none" }}>
              <span style={{ display: "inline-flex", alignItems: "center", justifyContent: "center", width: 28, height: 28, flex: "none", borderRadius: 999,
                background: s.done ? "var(--ok-soft)" : "var(--surface-sunken)", border: "1px solid " + (s.done ? "var(--ok-border)" : "var(--border-default)"),
                color: s.done ? "var(--ok)" : "var(--text-muted)", font: "600 12px var(--font-mono)" }}>
                {s.done ? <Icon name="check" size={14} /> : s.n}
              </span>
              <span style={{ display: "flex", flexDirection: "column", gap: 2, minWidth: 0, flex: 1 }}>
                <span style={{ font: "500 13.5px var(--font-ui)", color: "var(--text-ink)" }}>{s.title}</span>
                <span style={{ font: "400 12px/1.55 var(--font-ui)", color: "var(--text-muted)" }}>{s.detail}</span>
              </span>
              {s.action && (
                <Button size="sm" variant={s.gated ? "secondary" : "primary"} disabled={s.gated || undefined}
                  title={s.gated ? "Needs an internet vantage first" : undefined} onClick={s.action[1]}>{s.action[0]}</Button>
              )}
            </div>
          ))}
        </div>
      </Card>
      <span style={{ font: "400 11.5px/1.6 var(--font-ui)", color: "var(--text-muted)" }}>Step 4 stays gated until an internet vantage exists — probing your own address from inside is a hairpinning trap that never traverses the inbound policy.</span>
    </main>
  );
}
