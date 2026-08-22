import React from "react";
import { Input } from "../../components/forms/Input.jsx";
import { SegmentedControl } from "../../components/forms/SegmentedControl.jsx";
import { IntegrationTile } from "../../components/display/IntegrationTile.jsx";
import { ConsentList } from "../../components/display/ConsentList.jsx";
import { KeyValueList } from "../../components/display/KeyValueList.jsx";
import { Callout } from "../../components/feedback/Callout.jsx";
import { Drawer } from "../../components/feedback/Drawer.jsx";
import { Button } from "../../components/forms/Button.jsx";
import { Badge } from "../../components/display/Badge.jsx";
import { Icon } from "../../components/media/Icon.jsx";

const CATALOG = [
  { id: "slack", name: "Slack", mark: "SL", category: "Notify", state: "installed", description: "Signals and drift summaries as formatted messages, one channel per class.",
    grants: [{ scope: "Read signals", detail: "Message content mirrors the signal drawer: fact, evidence, rule." }, { scope: "Read drift summaries", detail: "Batch-level appeared / withdrawn counts." }] },
  { id: "pagerduty", name: "PagerDuty", mark: "PD", category: "Notify", state: "attention", description: "Critical signals open incidents; withdrawn signals resolve them.",
    grants: [{ scope: "Read signals", detail: "Critical and high only \u2014 routing is by class and severity." }, { scope: "Write annotations", detail: "Incident acknowledgement records an annotation on the signal.", write: true }] },
  { id: "teams", name: "Microsoft Teams", mark: "MT", category: "Notify", state: "available", description: "Adaptive cards for signals and batch completions.",
    grants: [{ scope: "Read signals" }, { scope: "Read batch results", detail: "Completion, counts, failures." }] },
  { id: "jira", name: "Jira", mark: "JI", category: "Ticketing", state: "installed", description: "One issue per signal span. Closing the span closes the issue \u2014 never the reverse.",
    grants: [{ scope: "Read signals", detail: "Issue fields mirror the signal; severity maps to priority." }, { scope: "Write annotations", detail: "Issue transitions propose an annotation \u2014 an operator confirms it.", write: true }] },
  { id: "linear", name: "Linear", mark: "LN", category: "Ticketing", state: "available", description: "Signals as issues with severity labels and asset links.",
    grants: [{ scope: "Read signals" }] },
  { id: "splunk", name: "Splunk", mark: "SP", category: "SIEM", state: "available", description: "Every observation and transition as HEC events, source-typed by class.",
    grants: [{ scope: "Read observations", detail: "The full evidence stream, not just signals." }, { scope: "Read drift transitions" }] },
  { id: "elastic", name: "Elastic", mark: "EL", category: "SIEM", state: "available", description: "Bulk-indexed observations with ECS field mapping.",
    grants: [{ scope: "Read observations" }, { scope: "Read drift transitions" }] },
  { id: "s3", name: "S3-compatible export", mark: "S3", category: "Storage", state: "installed", description: "Nightly NDJSON snapshots of inventory, signals, and coverage to your bucket.",
    grants: [{ scope: "Read inventory" }, { scope: "Read signals" }, { scope: "Read coverage facts" }] },
];
const CATS = ["All", "Notify", "Ticketing", "SIEM", "Storage"];

export function Integrations({ onToast }) {
  const [q, setQ] = React.useState("");
  const [cat, setCat] = React.useState("All");
  const [states, setStates] = React.useState(() => { const m = {}; CATALOG.forEach((c) => { m[c.id] = c.state; }); return m; });
  const [open, setOpen] = React.useState(null);
  const list = CATALOG.filter((c) => (cat === "All" || c.category === cat) && (!q || c.name.toLowerCase().includes(q.toLowerCase()) || c.description.toLowerCase().includes(q.toLowerCase())));
  const cur = open && CATALOG.find((c) => c.id === open);
  const curState = cur && states[cur.id];
  const install = () => { setStates((s) => ({ ...s, [cur.id]: "installed" })); onToast && onToast({ tone: "ok", title: cur.name + " installed", description: "Deliveries start with the next message." }); setOpen(null); };
  const remove = () => { setStates((s) => ({ ...s, [cur.id]: "available" })); onToast && onToast({ tone: "neutral", title: cur.name + " removed", description: "Nothing was deleted on the " + cur.category.toLowerCase() + " side." }); setOpen(null); };
  return (
    <section data-screen-label="Settings · Integrations" style={{ display: "flex", flexDirection: "column", gap: 20 }}>
      <header style={{ display: "flex", alignItems: "flex-end", gap: 16, flexWrap: "wrap" }}>
        <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
          <h2 style={{ margin: 0, font: "600 17px var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)" }}>Integrations</h2>
          <span style={{ font: "400 12.5px var(--font-ui)", color: "var(--text-muted)" }}>One-way where possible — Verge pushes, integrations receive. Write-backs are proposals, never acts.</span>
        </div>
        <div style={{ marginLeft: "auto", display: "flex", gap: 10, alignItems: "center", flexWrap: "wrap" }}>
          <SegmentedControl value={cat} onChange={setCat} options={CATS} />
          <Input size="sm" mono placeholder="Search integrations" prefix={<Icon name="search" size={13} />} value={q} onChange={(e) => setQ(e.target.value)} style={{ width: 220 }} />
        </div>
      </header>
      <Callout tone="neutral" title="Webhooks need no integration">Channels are built in — Settings → Channels delivers raw JSON to any URL. Integrations add formatting, acks, and state mapping on top.</Callout>
      <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(280px, 1fr))", gap: 16 }}>
        {list.map((c) => <IntegrationTile key={c.id} name={c.name} category={c.category} description={c.description} mark={c.mark} state={states[c.id]} onClick={() => setOpen(c.id)} />)}
      </div>
      <Drawer open={!!cur} onClose={() => setOpen(null)} title={cur ? cur.name : ""} width={430}
        footer={cur && (curState === "available"
          ? <Button onClick={install} style={{ width: "100%", justifyContent: "center" }}>Install {cur.name}</Button>
          : <div style={{ display: "flex", gap: 8, width: "100%" }}>
              <Button variant="ghost" onClick={remove}>Remove</Button>
              <Button variant="secondary" style={{ marginLeft: "auto" }} onClick={() => { onToast && onToast({ tone: "ok", title: "Test message sent", description: "Check " + cur.name + " for the delivery." }); }}>Send test</Button>
            </div>)}>
        {cur && (
          <div style={{ display: "flex", flexDirection: "column", gap: 18 }}>
            <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
              <span style={{ display: "inline-flex", alignItems: "center", justifyContent: "center", width: 34, height: 34, borderRadius: 10, background: "var(--surface-sunken)", border: "1px solid var(--border-default)", font: "600 12px var(--font-mono)", color: "var(--text-secondary)" }}>{cur.mark}</span>
              <span style={{ font: "400 12.5px var(--font-ui)", color: "var(--text-secondary)" }}>{cur.category}</span>
              <span style={{ marginLeft: "auto" }}><Badge tone={curState === "installed" ? "ok" : curState === "attention" ? "warn" : "neutral"} dot>{curState === "attention" ? "needs attention" : curState}</Badge></span>
            </div>
            <p style={{ margin: 0, font: "400 13px/1.6 var(--font-ui)", color: "var(--text-body)" }}>{cur.description}</p>
            {curState === "attention" && <Callout tone="warn" title="Delivery failing">The last 3 deliveries were refused (401). Rotate the token on the {cur.name} side, then send a test.</Callout>}
            <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
              <span style={{ font: "500 11px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--text-muted)" }}>This integration can</span>
              <ConsentList grants={cur.grants} />
            </div>
            {curState !== "available" && (
              <KeyValueList items={[
                { k: "Installed", v: "2026-07-14" },
                { k: "Last delivery", v: cur.id === "pagerduty" ? "failed \u00b7 6h ago" : "delivered \u00b7 4m ago" },
                { k: "Classes", v: cur.category === "Notify" ? "signals, drift" : "signals" },
              ]} />
            )}
          </div>
        )}
      </Drawer>
    </section>
  );
}
