import React from "react";
import { Breadcrumb } from "../../components/navigation/Breadcrumb.jsx";
import { Card } from "../../components/display/Card.jsx";
import { Stat } from "../../components/display/Stat.jsx";
import { Table } from "../../components/display/Table.jsx";
import { Tag } from "../../components/display/Tag.jsx";
import { Badge } from "../../components/display/Badge.jsx";
import { SeverityBadge } from "../../components/display/SeverityBadge.jsx";
import { ChangeBadge } from "../../components/display/ChangeBadge.jsx";
import { Button } from "../../components/forms/Button.jsx";
import { Icon } from "../../components/media/Icon.jsx";
import { SEV_ORDER, SEV_COUNTS } from "./SignalData.jsx";

export function ReportArtifact({ onBack }) {
  const max = Math.max(...SEV_ORDER.map((s) => SEV_COUNTS[s]));
  const h2 = { margin: 0, font: "500 11px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--text-muted)" };
  return (
    <main data-screen-label="Report artifact" style={{ maxWidth: 1440, margin: "0 auto", padding: 32, display: "flex", flexDirection: "column", gap: 20 }}>
      <Breadcrumb items={[{ label: "Reports", onClick: onBack }, { label: "Weekly exposure summary" }]} />
      <header style={{ display: "flex", alignItems: "flex-start", gap: 16, flexWrap: "wrap" }}>
        <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
          <h1 style={{ margin: 0, font: "600 21px var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)" }}>Weekly exposure summary</h1>
          <span style={{ font: "400 12px var(--font-mono)", color: "var(--text-muted)" }}>2026-08-15 → 2026-08-22 · delivery #42</span>
        </div>
        <div style={{ marginLeft: "auto", display: "flex", gap: 8 }}>
          <Button variant="secondary" icon={<Icon name="download" size={14} />}>Download PDF</Button>
          <Button variant="ghost" onClick={onBack}>Edit schedule</Button>
        </div>
      </header>
      <Card pad={28} style={{ maxWidth: 800, width: "100%", margin: "0 auto", boxSizing: "border-box" }}>
        <div style={{ display: "flex", flexDirection: "column", gap: 24 }}>
          <div style={{ display: "flex", alignItems: "baseline", gap: 12, paddingBottom: 16, borderBottom: "1px solid var(--row-sep)" }}>
            <span style={{ font: "600 15px var(--font-ui)", color: "var(--text-ink)" }}>acmecorp</span>
            <span style={{ font: "400 11.5px var(--font-mono)", color: "var(--text-muted)" }}>generated 2026-08-22T09:00:00Z · verge v0.9.2</span>
            <span style={{ marginLeft: "auto" }}><Tag>pdf</Tag></span>
          </div>
          <div style={{ display: "grid", gridTemplateColumns: "repeat(3, 1fr)", gap: 20 }}>
            <Stat label="Open signals" value="47" delta="+3" deltaTone="bad" caption="vs previous week" />
            <Stat label="New assets" value="12" delta="+8" deltaTone="neutral" caption="8 domains · 4 IPs" />
            <Stat label="Mean time to resolve" value="2.4d" delta="−0.6d" deltaTone="good" caption="critical + high only" />
          </div>
          <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
            <h2 style={h2}>Open signals by severity</h2>
            {SEV_ORDER.map((s) => (
              <div key={s} style={{ display: "flex", alignItems: "center", gap: 12 }}>
                <span style={{ width: 72, font: "500 11px var(--font-mono)", letterSpacing: "0.06em", textTransform: "uppercase", color: "var(--text-secondary)" }}>{s}</span>
                <span style={{ flex: 1, height: 8, borderRadius: 999, background: "var(--surface-sunken)", overflow: "hidden" }}>
                  <span style={{ display: "block", height: "100%", width: (SEV_COUNTS[s] / max) * 100 + "%", borderRadius: 999, background: "var(--sev-" + s + "-dot)" }} />
                </span>
                <span style={{ width: 26, textAlign: "right", font: "500 12.5px var(--font-mono)", color: "var(--text-body)" }}>{SEV_COUNTS[s]}</span>
              </div>
            ))}
          </div>
          <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
            <h2 style={h2}>New this week</h2>
            <Table framed={false} dense columns={[
              { key: "sev", label: "Severity", width: 110, render: (r) => <SeverityBadge level={r.sev} size="sm" /> },
              { key: "signal", label: "Signal" },
              { key: "asset", label: "Asset", mono: true, width: 220 },
              { key: "seen", label: "Raised", mono: true, align: "right", width: 70 },
            ]} rows={[
              { sev: "critical", signal: ":5900 vnc — no transport encryption", asset: "edge-gw-03.acmecorp.io", seen: "aug 22" },
              { sev: "high", signal: "nginx 1.25.0 · CVE-2026-1187", asset: "api.acmecorp.io", seen: "aug 22" },
              { sev: "medium", signal: "certificate expires in 23 days", asset: "idp-signing-2026", seen: "aug 20" },
            ]} rowKey="signal" />
          </div>
          <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
            <h2 style={h2}>Withdrawn by the world</h2>
            {[[":8080 http-alt on edge-gw-03.acmecorp.io", "closed since batch 14:00Z"], ["expired staging certificate on staging-4.acmecorp.io", "renewed"]].map(([t, r]) => (
              <div key={t} style={{ display: "flex", alignItems: "center", gap: 10, flexWrap: "wrap" }}>
                <ChangeBadge change="withdrawn" size="sm" />
                <span style={{ font: "500 12.5px var(--font-mono)", color: "var(--text-ink)" }}>{t}</span>
                <span style={{ font: "400 11.5px var(--font-ui)", color: "var(--text-muted)" }}>{r}</span>
              </div>
            ))}
          </div>
          <div style={{ display: "flex", alignItems: "center", gap: 10, paddingTop: 16, borderTop: "1px solid var(--row-sep)", flexWrap: "wrap" }}>
            <span style={{ font: "400 11.5px var(--font-mono)", color: "var(--text-muted)" }}>delivered 2026-08-22T09:00Z · ops.acmecorp.io/hook</span>
            <span style={{ marginLeft: "auto" }}><Badge tone="ok" dot>delivered</Badge></span>
          </div>
        </div>
      </Card>
    </main>
  );
}
