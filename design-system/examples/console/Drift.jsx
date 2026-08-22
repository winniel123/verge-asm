import React from "react";
import { Card } from "../../components/display/Card.jsx";
import { Timeline } from "../../components/display/Timeline.jsx";
import { ChangeBadge } from "../../components/display/ChangeBadge.jsx";
import { DiffView } from "../../components/display/DiffView.jsx";
import { Stat } from "../../components/display/Stat.jsx";
import { Button } from "../../components/forms/Button.jsx";
import { DateRangePicker } from "../../components/forms/DateRangePicker.jsx";
import { Icon } from "../../components/media/Icon.jsx";

const KINDS = ["appeared", "revealed", "withdrawn", "descoped", "returned", "changed"];

const ev = (change, subject, detail, time, reason, content) => ({ change, reason, time, mono: true,
  title: (
    <span style={{ display: "flex", alignItems: "center", gap: 8, flexWrap: "wrap", rowGap: 4 }}>
      <ChangeBadge change={change} size="sm" />
      <span style={{ font: "500 12.5px var(--font-mono)", color: "var(--text-ink)" }}>{subject}</span>
    </span>
  ),
  detail: detail + (reason ? " · " + reason : ""), content });

const GROUPS = [
  { id: "b3", label: "2026-08-22T14:00Z", meta: "full scan · 3 vantages", events: [
    ev("changed", "api.acmecorp.io :443", "service banner", "4m", null,
      <DiffView lines={[{ type: "remove", text: "nginx/1.24.0" }, { type: "add", text: "nginx/1.25.0 (CVE-2026-1187)" }]} />),
    ev("appeared", "staging-5.acmecorp.io", "name · first seen via certificate transparency", "8m"),
    ev("withdrawn", ":8080 http-alt on edge-gw-03.acmecorp.io", "service", "9m", "closed since last batch"),
  ] },
  { id: "b2", label: "2026-08-22T08:00Z", meta: "full scan · 3 vantages", events: [
    ev("returned", "mail.acmecorp.io :587", "service · absent for 2 batches", "6h"),
    ev("revealed", "203.0.113.77", "address · custody extension widened the aperture", "6h"),
  ] },
  { id: "b1", label: "2026-08-21T14:00Z", meta: "full scan · 2 vantages", defaultCollapsed: true, events: [
    ev("descoped", "old-blog.acmecorp.io", "name", "1d", "operator excluded subtree"),
    ev("changed", "www.acmecorp.io :443", "certificate issuer", "1d"),
  ] },
];

export function Drift() {
  const [range, setRange] = React.useState({ label: "Last 7d" });
  const [active, setActive] = React.useState(KINDS);
  const toggle = (k) => setActive((a) => (a.indexOf(k) !== -1 ? (a.length === 1 ? KINDS : a.filter((x) => x !== k)) : a.concat(k)));
  const groups = GROUPS.map((g) => ({ ...g, events: g.events.filter((e) => active.indexOf(e.change) !== -1) })).filter((g) => g.events.length);
  return (
    <main data-screen-label="Drift" style={{ maxWidth: 1440, margin: "0 auto", padding: 32, display: "flex", flexDirection: "column", gap: 20 }}>
      <header style={{ display: "flex", alignItems: "center", gap: 16 }}>
        <div style={{ display: "flex", flexDirection: "column", gap: 2 }}>
          <h1 style={{ margin: 0, font: "600 21px var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)" }}>Drift</h1>
          <span style={{ font: "400 12.5px var(--font-ui)", color: "var(--text-muted)", whiteSpace: "nowrap" }}>What moved since last time, grouped by batch. Change is its own language — not severity.</span>
        </div>
        <div style={{ marginLeft: "auto", display: "flex", gap: 8, alignItems: "center" }}>
          <DateRangePicker value={range} onChange={setRange} />
          <Button variant="secondary" icon={<Icon name="download" size={14} />}>Export CSV</Button>
        </div>
      </header>
      <div style={{ display: "grid", gridTemplateColumns: "1fr 320px", gap: 24, alignItems: "start" }}>
        <Card microLabel="Transitions" title="By batch" pad={20}>
          <div style={{ display: "flex", gap: 6, flexWrap: "wrap", paddingBottom: 16, marginBottom: 4, borderBottom: "1px solid var(--row-sep)" }}>
            {KINDS.map((k) => (
              <span key={k} onClick={() => toggle(k)} style={{ cursor: "pointer", opacity: active.indexOf(k) !== -1 ? 1 : 0.4, transition: "opacity var(--dur-fast) var(--ease-out)" }}>
                <ChangeBadge change={k} size="sm" />
              </span>
            ))}
          </div>
          <Timeline groups={groups} />
        </Card>
        <div style={{ display: "flex", flexDirection: "column", gap: 24 }}>
          <Card microLabel="This period" title="Movement">
            <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
              <Stat label="Transitions" value="7" delta="+2" caption="vs previous period" />
              <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
                {[["appeared", 1], ["revealed", 1], ["returned", 1], ["changed", 2], ["withdrawn", 1], ["descoped", 1]].map(([k, n]) => (
                  <div key={k} style={{ display: "flex", alignItems: "center", gap: 10 }}>
                    <ChangeBadge change={k} size="sm" />
                    <span style={{ marginLeft: "auto", font: "500 12.5px var(--font-mono)", color: "var(--text-body)" }}>{n}</span>
                  </div>
                ))}
              </div>
            </div>
          </Card>
        </div>
      </div>
    </main>
  );
}
