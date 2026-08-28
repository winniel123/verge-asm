import React from "react";
import { Card } from "../../components/display/Card.jsx";
import { Stat } from "../../components/display/Stat.jsx";
import { Table } from "../../components/display/Table.jsx";
import { SeverityBadge } from "../../components/display/SeverityBadge.jsx";
import { Button } from "../../components/forms/Button.jsx";
import { Icon } from "../../components/media/Icon.jsx";
import { Banner } from "../../components/feedback/Banner.jsx";
import { Progress } from "../../components/display/Progress.jsx";
import { CoverageMeter } from "../../components/display/CoverageMeter.jsx";
import { StalenessBadge } from "../../components/display/StalenessBadge.jsx";
import { AvailabilityBadge } from "../../components/display/AvailabilityBadge.jsx";
import { SIGNALS, SEV_ORDER, SEV_COUNTS } from "./SignalData.jsx";

function SevBars() {
  const max = Math.max(...SEV_ORDER.map((s) => SEV_COUNTS[s]));
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
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
  );
}

const VANTAGES = [
  { region: "eu-west-1", avail: "available", note: "34ms" },
  { region: "us-east-2", avail: "available", note: "51ms" },
  { region: "ap-south-1", avail: "unavailable", note: "12m" },
];

export function Dashboard({ onRunScan, onAddTarget, onOpenSignals, scanning }) {
  const [probeBanner, setProbeBanner] = React.useState(true);
  return (
    <main data-screen-label="Dashboard" style={{ maxWidth: 1440, margin: "0 auto", padding: 32, display: "flex", flexDirection: "column", gap: 24 }}>
      <header style={{ display: "flex", alignItems: "center", gap: 16 }}>
        <div style={{ display: "flex", flexDirection: "column", gap: 2 }}>
          <h1 style={{ margin: 0, font: "600 21px var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)" }}>Dashboard</h1>
          <span style={{ font: "400 12.5px var(--font-ui)", color: "var(--text-muted)", whiteSpace: "nowrap" }}>Last full scan <span style={{ fontFamily: "var(--font-mono)", fontSize: 12, color: "var(--text-secondary)" }}>38m</span> ago · next in <span style={{ fontFamily: "var(--font-mono)", fontSize: 12, color: "var(--text-secondary)" }}>5h 22m</span></span>
        </div>
        <div style={{ marginLeft: "auto", display: "flex", gap: 8 }}>
          <Button variant="secondary" icon={<Icon name="plus" size={14} />} onClick={onAddTarget}>Add seed</Button>
          <Button icon={<Icon name="play" size={14} />} onClick={onRunScan} disabled={scanning}>{scanning ? "Scan running" : "Run scan"}</Button>
        </div>
      </header>
      {scanning && <Progress label="Scan running" detail="214 subjects queued" />}
      {probeBanner && (
        <Banner tone="warn" title="Vantage unreachable" onDismiss={() => setProbeBanner(false)}
          action={<Button size="sm" variant="secondary">Retry now</Button>}>
          ap-south-1 missed 2 checks. Scans continue from other vantages.
        </Banner>
      )}
      <Card pad={0}>
        <div style={{ display: "grid", gridTemplateColumns: "repeat(5, 1fr)" }}>
          {[
            { label: "Open signals", value: "47", delta: "+3", deltaTone: "bad", caption: "vs last scan", live: scanning },
            { label: "Critical", value: "3", delta: "\u22121", deltaTone: "good", caption: "1 withdrawn today" },
            { label: "Assets watched", value: "1,284", delta: "+12", caption: "8 domains \u00b7 3 ranges" },
            { label: "Exposed services", value: "216", delta: "+4", deltaTone: "bad", caption: "across 62 IPs" },
            { label: "Certs expiring \u226430d", value: "9", delta: "\u22122", deltaTone: "good", caption: "next: 2026-08-29" },
          ].map((s, i) => (
            <div key={s.label} style={{ padding: "20px 24px", borderLeft: i ? "1px solid var(--row-sep)" : "none" }}>
              <Stat {...s} />
            </div>
          ))}
        </div>
      </Card>
      <div style={{ display: "grid", gridTemplateColumns: "380px 1fr", gap: 24, alignItems: "start" }}>
        <div style={{ display: "flex", flexDirection: "column", gap: 24 }}>
          <Card microLabel="Open signals" title="By severity">
            <SevBars />
          </Card>
          <Card microLabel="Coverage" title="Did we look, how completely">
            <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
              <CoverageMeter label="203.0.113.0/24" counted={212} total={256} unit="addresses" size="sm" />
              <CoverageMeter label="acmecorp.io names" counted={1284} unit="names" size="sm" />
              <div style={{ display: "flex", alignItems: "center", gap: 8, minWidth: 0 }}>
                <span style={{ flex: "none" }}><StalenessBadge kind="silent" bound="9d" size="sm" /></span>
                <span style={{ minWidth: 0, font: "400 11.5px var(--font-ui)", color: "var(--text-muted)", overflowWrap: "anywhere" }}>zone transfer for internal.acmecorp.io</span>
              </div>
            </div>
          </Card>
          <Card microLabel="Vantages" title="Scan infrastructure">
            <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
              {VANTAGES.map((p) => (
                <div key={p.region} style={{ display: "flex", alignItems: "center", gap: 10 }}>
                  <span style={{ font: "400 12.5px var(--font-mono)", color: "var(--text-body)" }}>{p.region}</span>
                  <span style={{ font: "400 12px var(--font-mono)", color: "var(--text-muted)" }}>{p.note}</span>
                  <span style={{ marginLeft: "auto" }}><AvailabilityBadge state={p.avail} size="sm" /></span>
                </div>
              ))}
            </div>
          </Card>
        </div>
        <Card microLabel="Most recent" title="Signals" pad={0} style={{ paddingTop: 0 }}
          action={<Button variant="ghost" size="sm" onClick={onOpenSignals}>View all</Button>}>
          <Table framed={false} columns={[
            { key: "sev", label: "Severity", width: 110, render: (r) => <SeverityBadge level={r.sev} size="sm" /> },
            { key: "title", label: "Signal", render: (r) => <span style={{ font: "500 13px var(--font-ui)", color: "var(--text-ink)" }}>{r.title}</span> },
            { key: "asset", label: "Asset", mono: true },
            { key: "port", label: "Port", mono: true, width: 70 },
            { key: "seen", label: "Seen", mono: true, align: "right", width: 64 },
          ]} rows={SIGNALS.slice(0, 6)} onRowClick={onOpenSignals} />
        </Card>
      </div>
    </main>
  );
}
