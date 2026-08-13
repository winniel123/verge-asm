(() => {
const { Card, Table, Badge, Button, IconButton } = window.NS;
const D = window.VergeData;

const TEMPLATES = [
  { name: "Executive summary", desc: "One-page risk posture for leadership: totals, deltas, top findings.", fmt: "PDF" },
  { name: "Full technical", desc: "Every asset, service, and finding with evidence. For engineers.", fmt: "CSV" },
  { name: "Delta since last", desc: "Only what changed: new, resolved, and regressed findings.", fmt: "JSON" },
];

function Reports({ toast }) {
  return (
    <div data-screen-label="Reports">
      <div style={{ display: "flex", alignItems: "baseline", gap: 14, marginBottom: 16 }}>
        <h1 style={{ font: "700 20px/1.2 var(--font-sans)", color: "var(--ink)", margin: 0 }}>Reports</h1>
        <span style={{ font: "400 11px var(--font-mono)", color: "var(--text-muted)" }}>scheduled: exec summary, monthly</span>
      </div>
      <div style={{ display: "grid", gridTemplateColumns: "repeat(3,1fr)", gap: 16, marginBottom: 16 }}>
        {TEMPLATES.map((t) => (
          <Card key={t.name} title={t.name} action={<Badge>{t.fmt}</Badge>}>
            <div style={{ font: "400 12px/1.5 var(--font-sans)", color: "var(--text-muted)", minHeight: 38 }}>{t.desc}</div>
            <Button variant="secondary" size="sm" style={{ marginTop: 12 }} onClick={() => toast("Report queued", `${t.name} · generating, ~20s`)}>Generate</Button>
          </Card>
        ))}
      </div>
      <Card eyebrow="Generated reports" pad={false}>
        <Table rowKey="id"
          columns={[
            { key: "name", label: "File", mono: true, nowrap: true },
            { key: "range", label: "Range", muted: true, width: 130 },
            { key: "format", label: "Format", width: 90, render: (r) => <Badge>{r.format}</Badge> },
            { key: "size", label: "Size", mono: true, muted: true, align: "right", width: 90 },
            { key: "created", label: "Created", mono: true, muted: true, align: "right", width: 110 },
            { key: "dl", label: "", align: "right", width: 56, render: () => <IconButton label="Download" size="sm">↓</IconButton> },
          ]}
          rows={D.reports} />
      </Card>
    </div>
  );
}
window.VergeApp = Object.assign(window.VergeApp || {}, { Reports });
})();
