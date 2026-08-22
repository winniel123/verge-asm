import React from "react";
import { Card } from "../../components/display/Card.jsx";
import { Stat } from "../../components/display/Stat.jsx";
import { Table } from "../../components/display/Table.jsx";
import { ExposureBadge } from "../../components/display/ExposureBadge.jsx";
import { EmptyState } from "../../components/feedback/EmptyState.jsx";
import { Callout } from "../../components/feedback/Callout.jsx";
import { Button } from "../../components/forms/Button.jsx";
import { SegmentedControl } from "../../components/forms/SegmentedControl.jsx";

const ROWS = [
  { asset: "edge-gw-03.acmecorp.io", svc: ":5900 vnc", internal: "exposed", internet: "exposed", since: "4m" },
  { asset: "api.acmecorp.io", svc: ":443 https", internal: "exposed", internet: "exposed", since: "69d" },
  { asset: "vpn.acmecorp.io", svc: ":1194 openvpn", internal: "exposed", internet: "exposed", since: "41d" },
  { asset: "build-07.acmecorp.io", svc: ":22 ssh", internal: "exposed", internet: "firewalled", since: "12d" },
  { asset: "grafana.acmecorp.io", svc: ":3000 http", internal: "exposed", internet: "firewalled", since: "26d" },
  { asset: "203.0.113.61", svc: ":443 https", internal: "not-reached", internet: "unverified", since: "—" },
];

/* What the internet can reach — constructible only with both legs. */
export function Exposure({ onOpenVantages }) {
  const [mode, setMode] = React.useState("constructible");
  return (
    <main data-screen-label="Exposure" style={{ maxWidth: 1440, margin: "0 auto", padding: 32, display: "flex", flexDirection: "column", gap: 20 }}>
      <header style={{ display: "flex", alignItems: "center", gap: 16 }}>
        <div style={{ display: "flex", flexDirection: "column", gap: 2 }}>
          <h1 style={{ margin: 0, font: "600 21px var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)" }}>Exposure</h1>
          <span style={{ font: "400 12.5px var(--font-ui)", color: "var(--text-muted)", whiteSpace: "nowrap" }}>Composed from two reach legs — internal and internet. States, never a score.</span>
        </div>
        <span style={{ marginLeft: "auto" }}>
          <SegmentedControl label="Spec state" value={mode} onChange={setMode} options={[{ value: "constructible", label: "With vantage" }, { value: "withheld", label: "Withheld" }]} />
        </span>
      </header>
      {mode === "withheld" ? (
        <Card>
          <EmptyState icon="eye-off" message="Exposure withheld." detail="No internet vantage exists. Internal reachability is complete, but exposure claims need the outside leg — Verge degrades to internal-only rather than report firewalled or exposed for something it did not look at."
            action={<Button onClick={onOpenVantages}>Provision a prober</Button>} style={{ padding: "56px 24px" }} />
        </Card>
      ) : (
        <React.Fragment>
          <div style={{ display: "grid", gridTemplateColumns: "repeat(3, 1fr)", gap: 24 }}>
            <Card><Stat label="Exposed to internet" value="14" delta="+2" deltaTone="bad" caption="services · both legs concluded" /></Card>
            <Card><Stat label="Firewalled" value="41" caption="reachable inside, filtered outside" /></Card>
            <Card><Stat label="Not reached" value="7" caption="no leg concluded this batch" /></Card>
          </div>
          <Card microLabel="Both legs" title="Service exposure" pad={0}>
            <Table framed={false} columns={[
              { key: "asset", label: "Asset", mono: true },
              { key: "svc", label: "Service", mono: true, width: 150 },
              { key: "internal", label: "Internal leg", width: 140, render: (r) => <ExposureBadge state={r.internal} /> },
              { key: "internet", label: "Internet leg", width: 140, render: (r) => <ExposureBadge state={r.internet} /> },
              { key: "since", label: "Since", mono: true, align: "right", width: 70 },
            ]} rows={ROWS} rowKey="asset" />
          </Card>
          <Callout tone="neutral" title="One leg never concludes">A single vantage can say reached or not reached from where it stands. Exposed and firewalled exist only where the internal and internet legs are both constructible in the same derivation.</Callout>
        </React.Fragment>
      )}
    </main>
  );
}
