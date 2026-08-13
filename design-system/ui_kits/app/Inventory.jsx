(() => {
const { Card, Table, Badge, SeverityBadge, Tag, Input, Select, Button, Tabs, EmptyState } = window.NS;
const D = window.VergeData;

function Inventory() {
  const [tab, setTab] = React.useState("All (1,284)");
  const [q, setQ] = React.useState("");
  const [tags, setTags] = React.useState(["env:prod"]);
  const typeOf = (label) => label.startsWith("Domains") ? "domain" : label.startsWith("Hosts") ? "host" : label.startsWith("Services") ? "service" : null;
  const rows = D.assets.filter((a) => {
    const t = typeOf(tab);
    return (!t || a.type === t) && (!q || (a.host + a.ip + a.tech).toLowerCase().includes(q.toLowerCase()));
  });
  const statusTone = { active: "ok", new: "accent", paused: "neutral" };
  return (
    <div data-screen-label="Inventory">
      <div style={{ display: "flex", alignItems: "baseline", gap: 14, marginBottom: 16 }}>
        <h1 style={{ font: "700 20px/1.2 var(--font-sans)", color: "var(--ink)", margin: 0 }}>Inventory</h1>
        <span style={{ font: "400 11px var(--font-mono)", color: "var(--text-muted)" }}>1,284 assets · 12 shown</span>
        <Button variant="secondary" size="sm" style={{ marginLeft: "auto" }}>Export CSV</Button>
      </div>
      <div style={{ display: "flex", gap: 10, alignItems: "center", marginBottom: 12, flexWrap: "wrap" }}>
        <Input mono placeholder="Search hosts, IPs, services" value={q} onChange={(e) => setQ(e.target.value)} style={{ width: 280 }} />
        <Select options={["Any status", "Active", "New", "Paused"]} style={{ width: 130 }} />
        <Select options={["Any port", "443", "80", "22", "25", "5900"]} style={{ width: 110 }} />
        {tags.map((t) => <Tag key={t} onRemove={() => setTags(tags.filter((x) => x !== t))}>{t}</Tag>)}
      </div>
      <Tabs items={["All (1,284)", "Domains (412)", "Hosts (519)", "Services (353)"]} value={tab} onChange={setTab} style={{ marginBottom: 0 }} />
      <Card pad={false} style={{ borderTop: "none" }}>
        {rows.length === 0 ? (
          <EmptyState title="No assets match." detail="Clear the search or filters to see the full inventory." style={{ border: "none" }} />
        ) : (
          <Table rowKey="id" onRowClick={() => {}}
            columns={[
              { key: "host", label: "Asset", mono: true, nowrap: true },
              { key: "type", label: "Type", width: 90, render: (r) => <Badge>{r.type}</Badge> },
              { key: "ip", label: "IP", mono: true, muted: true, width: 130 },
              { key: "ports", label: "Open ports", mono: true, muted: true, width: 120 },
              { key: "tech", label: "Technology", muted: true, width: 180 },
              { key: "worst", label: "Findings", width: 130, render: (r) => r.count ? <span style={{ display: "inline-flex", gap: 6, alignItems: "center" }}><SeverityBadge severity={r.worst} /><span style={{ font: "400 11px var(--font-mono)", color: "var(--text-muted)" }}>×{r.count}</span></span> : <span style={{ font: "400 11px var(--font-mono)", color: "var(--text-faint)" }}>—</span> },
              { key: "status", label: "Status", width: 90, render: (r) => <Badge tone={statusTone[r.status]}>{r.status}</Badge> },
              { key: "firstSeen", label: "First seen", mono: true, muted: true, align: "right", width: 110 },
            ]}
            rows={rows} />
        )}
      </Card>
    </div>
  );
}
window.VergeApp = Object.assign(window.VergeApp || {}, { Inventory });
})();
