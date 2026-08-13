(() => {
const { Card, Table, SeverityBadge, Badge, Select, Input, Button, Tabs } = window.NS;
const D = window.VergeData;

function Detail({ f, onAck, onResolve }) {
  const [tab, setTab] = React.useState("Overview");
  const meta = [["First seen", f.firstSeen], ["Port", f.port], ["CVE", f.cve], ["Status", f.status]];
  return (
    <Card emphasized pad={false} style={{ position: "sticky", top: 16 }}>
      <div style={{ padding: 16, borderBottom: "1px solid var(--border-soft)" }}>
        <div style={{ display: "flex", gap: 10, alignItems: "center", marginBottom: 8 }}>
          <SeverityBadge severity={f.sev} />
          <span style={{ font: "400 10px var(--font-mono)", color: "var(--text-faint)" }}>{f.id}</span>
        </div>
        <div style={{ font: "600 15px/1.35 var(--font-sans)", color: "var(--ink)" }}>{f.title}</div>
        <div style={{ font: "400 12px var(--font-mono)", color: "var(--text-muted)", marginTop: 6 }}>{f.asset}</div>
      </div>
      <div style={{ display: "grid", gridTemplateColumns: "repeat(4,1fr)", borderBottom: "1px solid var(--border-soft)" }}>
        {meta.map(([k, v], i) => (
          <div key={k} style={{ padding: "10px 12px", borderRight: i < 3 ? "1px solid var(--border-soft)" : "none" }}>
            <div style={{ font: "600 9px/1 var(--font-mono)", letterSpacing: "0.06em", textTransform: "uppercase", color: "var(--text-faint)", marginBottom: 5 }}>{k}</div>
            <div style={{ font: "400 11px var(--font-mono)", color: "var(--text)", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>{v}</div>
          </div>
        ))}
      </div>
      <div style={{ padding: "0 16px" }}>
        <Tabs items={["Overview", "Evidence", "History"]} value={tab} onChange={setTab} />
      </div>
      <div style={{ padding: 16 }}>
        {tab === "Overview" && (
          <div style={{ font: "400 13px/1.55 var(--font-sans)" }}>
            <p style={{ margin: "0 0 12px" }}>{f.desc}</p>
            <div style={{ font: "600 10px/1 var(--font-mono)", letterSpacing: "0.06em", textTransform: "uppercase", color: "var(--text-muted)", margin: "0 0 6px" }}>Remediation</div>
            <p style={{ margin: 0 }}>{f.fix}</p>
          </div>
        )}
        {tab === "Evidence" && (
          <pre style={{ margin: 0, background: "var(--surface-ink)", color: "var(--text-on-ink)", border: "1px solid var(--border-ink)", padding: "12px 14px", font: "400 11px/1.7 var(--font-mono)", overflow: "auto", whiteSpace: "pre-wrap" }}>{f.evidence}</pre>
        )}
        {tab === "History" && (
          <div style={{ display: "flex", flexDirection: "column", gap: 8, font: "400 11px/1.5 var(--font-mono)", color: "var(--text-muted)" }}>
            <span>{f.firstSeen} · detected by scan S-3121</span>
            {f.status !== "open" && <span>2026-07-30 14:20 UTC · {f.status} by admin</span>}
            <span>notifications: webhook #sec-alerts delivered</span>
          </div>
        )}
      </div>
      <div style={{ display: "flex", justifyContent: "flex-end", gap: 8, padding: "12px 16px", borderTop: "1px solid var(--border)" }}>
        <Button variant="secondary" size="sm" onClick={onAck} disabled={f.status !== "open"}>Acknowledge</Button>
        <Button size="sm" onClick={onResolve} disabled={f.status === "resolved"}>Mark resolved</Button>
      </div>
    </Card>
  );
}

function Findings({ initialId, statuses, setStatuses }) {
  const [sel, setSel] = React.useState(initialId || D.findings[0].id);
  const [sevF, setSevF] = React.useState("All severities");
  const rows = D.findings
    .map((f) => ({ ...f, status: statuses[f.id] || f.status }))
    .filter((f) => sevF === "All severities" || f.sev === sevF.toLowerCase());
  const cur = rows.find((f) => f.id === sel) || rows[0];
  const setStatus = (s) => setStatuses({ ...statuses, [cur.id]: s });
  return (
    <div data-screen-label="Findings">
      <div style={{ display: "flex", alignItems: "baseline", gap: 14, marginBottom: 16 }}>
        <h1 style={{ font: "700 20px/1.2 var(--font-sans)", color: "var(--ink)", margin: 0 }}>Findings</h1>
        <span style={{ font: "400 11px var(--font-mono)", color: "var(--text-muted)" }}>87 open · 4 critical</span>
        <Button variant="secondary" size="sm" style={{ marginLeft: "auto" }}>Export CSV</Button>
      </div>
      <div style={{ display: "flex", gap: 10, marginBottom: 12, flexWrap: "wrap" }}>
        <Select options={["All severities", "Critical", "High", "Medium", "Low", "Info"]} value={sevF} onChange={(e) => setSevF(e.target.value)} style={{ width: 150 }} />
        <Select options={["Open + acknowledged", "Open", "Acknowledged", "Resolved"]} style={{ width: 180 }} />
        <Input mono placeholder="Search findings" style={{ width: 240 }} />
      </div>
      <div style={{ display: "grid", gridTemplateColumns: "1fr 420px", gap: 16, alignItems: "start" }}>
        <Card pad={false}>
          <Table rowKey="id" selectedKey={cur && cur.id} onRowClick={(r) => setSel(r.id)}
            columns={[
              { key: "sev", label: "Severity", width: 96, render: (r) => <SeverityBadge severity={r.sev} /> },
              { key: "title", label: "Finding" },
              { key: "asset", label: "Asset", mono: true, muted: true, width: 190, nowrap: true },
              { key: "status", label: "Status", width: 110, render: (r) => <Badge tone={r.status === "open" ? "danger" : r.status === "acknowledged" ? "warn" : "ok"}>{r.status}</Badge> },
              { key: "age", label: "Age", mono: true, muted: true, align: "right", width: 56 },
            ]}
            rows={rows} />
        </Card>
        {cur && <Detail f={cur} onAck={() => setStatus("acknowledged")} onResolve={() => setStatus("resolved")} />}
      </div>
    </div>
  );
}
window.VergeApp = Object.assign(window.VergeApp || {}, { Findings });
})();
