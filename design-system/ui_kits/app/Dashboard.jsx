(() => {
const { Card, Table, Stat, SeverityBadge, StatusDot, Button } = window.NS;
const D = window.VergeData;

function Sparkline({ data, width = 260, height = 48 }) {
  const max = Math.max(...data), min = Math.min(...data);
  const pts = data.map((v, i) => `${(i / (data.length - 1)) * width},${height - 6 - ((v - min) / (max - min)) * (height - 12)}`).join(" ");
  return (
    <svg width="100%" height={height} viewBox={`0 0 ${width} ${height}`} preserveAspectRatio="none" style={{ display: "block" }}>
      <polyline points={pts} fill="none" stroke="var(--accent)" strokeWidth="1.5" />
    </svg>
  );
}

function SevBars() {
  const total = D.severityDist.reduce((a, [, n]) => a + n, 0);
  const colors = { critical: "var(--sev-critical)", high: "var(--sev-high)", medium: "var(--sev-medium)", low: "var(--sev-low)", info: "var(--sev-info)" };
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 9 }}>
      {D.severityDist.map(([sev, n]) => (
        <div key={sev} style={{ display: "grid", gridTemplateColumns: "64px 1fr 28px", gap: 10, alignItems: "center" }}>
          <span style={{ font: "600 10px var(--font-mono)", letterSpacing: "0.05em", textTransform: "uppercase", color: "var(--text-muted)" }}>{sev}</span>
          <span style={{ height: 10, background: "var(--surface-sunken)", position: "relative" }}>
            <span style={{ position: "absolute", inset: "0 auto 0 0", width: `${(n / total) * 100}%`, background: colors[sev] }} />
          </span>
          <span style={{ font: "500 11px var(--font-mono)", textAlign: "right" }}>{n}</span>
        </div>
      ))}
    </div>
  );
}

function Dashboard({ openFinding }) {
  return (
    <div data-screen-label="Dashboard">
      <div style={{ display: "flex", alignItems: "baseline", gap: 14, marginBottom: 16 }}>
        <h1 style={{ font: "700 20px/1.2 var(--font-sans)", color: "var(--ink)", margin: 0 }}>Dashboard</h1>
        <span style={{ font: "400 11px var(--font-mono)", color: "var(--text-muted)" }}>last scan 2026-07-30 13:58 UTC · 4m 12s</span>
      </div>
      <div style={{ display: "grid", gridTemplateColumns: "repeat(4,1fr)", background: "var(--surface)", border: "1px solid var(--border)", marginBottom: 16 }}>
        {D.stats.map((s, i) => <Stat key={s.label} {...s} style={{ borderRight: i < 3 ? "1px solid var(--border)" : "none" }} />)}
      </div>
      <div style={{ display: "grid", gridTemplateColumns: "1fr 380px", gap: 16, alignItems: "start" }}>
        <Card eyebrow="Latest findings" action={<a href="#" onClick={(e) => e.preventDefault()}>View all →</a>} pad={false}>
          <Table rowKey="id" onRowClick={(r) => openFinding(r.id)}
            columns={[
              { key: "sev", label: "Severity", width: 96, render: (r) => <SeverityBadge severity={r.sev} /> },
              { key: "title", label: "Finding" },
              { key: "asset", label: "Asset", mono: true, muted: true, width: 210, nowrap: true },
              { key: "age", label: "Age", mono: true, muted: true, align: "right", width: 60 },
            ]}
            rows={D.findings.slice(0, 6)} />
        </Card>
        <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
          <Card eyebrow="Open findings by severity"><SevBars /></Card>
          <Card eyebrow="Scan activity" action={<StatusDot tone="ok" pulse label="running" />}>
            <Sparkline data={D.activity} />
            <div style={{ display: "flex", justifyContent: "space-between", font: "400 10px var(--font-mono)", color: "var(--text-faint)", marginTop: 6 }}>
              <span>new findings / scan · 4h interval</span><span>24h</span>
            </div>
          </Card>
          <Card eyebrow="Top exposed ports" pad={false}>
            <Table rowKey="0" columns={[
              { key: "0", label: "Port", mono: true, width: 90, render: (r) => r[0] },
              { key: "2", label: "Service", muted: true, render: (r) => r[2] },
              { key: "1", label: "Hosts", mono: true, align: "right", render: (r) => r[1] },
            ]} rows={D.topPorts} dense />
          </Card>
        </div>
      </div>
    </div>
  );
}
window.VergeApp = Object.assign(window.VergeApp || {}, { Dashboard });
})();
