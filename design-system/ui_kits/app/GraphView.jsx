(() => {
const { Card, Select, Tag, StatusDot, Button } = window.NS;

const NODES = [
  { id: "root", label: "acmecorp.io", x: 120, y: 210, kind: "root" },
  { id: "www", label: "www", x: 330, y: 80, kind: "sub" },
  { id: "api", label: "api", x: 330, y: 160, kind: "sub" },
  { id: "vpn", label: "vpn", x: 330, y: 240, kind: "sub", sev: "high" },
  { id: "staging", label: "staging", x: 330, y: 320, kind: "sub", sev: "high" },
  { id: "edge", label: "edge-gw-03", x: 330, y: 400, kind: "sub", sev: "critical" },
  { id: "ip10", label: "203.0.113.10", x: 570, y: 120, kind: "ip" },
  { id: "ip5", label: "203.0.113.5", x: 570, y: 240, kind: "ip" },
  { id: "ip21", label: "203.0.113.21", x: 570, y: 400, kind: "ip" },
  { id: "s443a", label: ":443 https", x: 790, y: 80, kind: "svc" },
  { id: "s443b", label: ":443 https", x: 790, y: 160, kind: "svc" },
  { id: "s1194", label: ":1194 openvpn", x: 790, y: 240, kind: "svc" },
  { id: "s22", label: ":22 ssh", x: 790, y: 360, kind: "svc" },
  { id: "s443c", label: ":443 apache", x: 790, y: 440, kind: "svc", sev: "critical" },
];
const EDGES = [["root","www"],["root","api"],["root","vpn"],["root","staging"],["root","edge"],["www","ip10"],["api","ip10"],["vpn","ip5"],["edge","ip21"],["staging","ip10"],["ip10","s443a"],["ip10","s443b"],["ip5","s1194"],["ip21","s22"],["ip21","s443c"]];
const SEVC = { critical: "var(--sev-critical)", high: "var(--sev-high)", medium: "var(--sev-medium)" };

function GraphView() {
  const [sel, setSel] = React.useState("edge");
  const byId = Object.fromEntries(NODES.map((n) => [n.id, n]));
  const cur = byId[sel];
  const nodeFill = (n) => n.kind === "root" ? "var(--ink)" : "var(--surface)";
  const nodeStroke = (n) => n.sev ? SEVC[n.sev] : n.kind === "ip" ? "var(--accent)" : "var(--border-ink)";
  return (
    <div data-screen-label="Graph">
      <div style={{ display: "flex", alignItems: "baseline", gap: 14, marginBottom: 16 }}>
        <h1 style={{ font: "700 20px/1.2 var(--font-sans)", color: "var(--ink)", margin: 0 }}>Graph</h1>
        <span style={{ font: "400 11px var(--font-mono)", color: "var(--text-muted)" }}>attack surface map · 1,284 nodes (14 shown)</span>
        <Button variant="secondary" size="sm" style={{ marginLeft: "auto" }}>Export PNG</Button>
      </div>
      <div style={{ display: "flex", gap: 10, alignItems: "center", marginBottom: 12, flexWrap: "wrap" }}>
        <Select options={["acmecorp.io", "All targets"]} style={{ width: 160 }} />
        <Select options={["Depth: services", "Depth: hosts", "Depth: domains"]} style={{ width: 150 }} />
        <span style={{ display: "flex", gap: 14, marginLeft: 8, font: "400 11px var(--font-mono)", color: "var(--text-muted)", alignItems: "center" }}>
          <span><span style={{ display: "inline-block", width: 9, height: 9, background: "var(--ink)", marginRight: 6 }} />root</span>
          <span><span style={{ display: "inline-block", width: 9, height: 9, background: "var(--surface)", border: "1px solid var(--border-ink)", marginRight: 6 }} />host</span>
          <span><span style={{ display: "inline-block", width: 9, height: 9, background: "var(--surface)", border: "1px solid var(--accent)", marginRight: 6 }} />ip</span>
          <span><span style={{ display: "inline-block", width: 9, height: 9, background: "var(--surface)", border: "1px solid var(--sev-critical)", marginRight: 6 }} />finding</span>
        </span>
      </div>
      <Card pad={false}>
        <svg viewBox="0 0 960 500" style={{ display: "block", width: "100%", background: "var(--surface)" }}>
          {EDGES.map(([a, b]) => {
            const A = byId[a], B = byId[b];
            return <line key={a + b} x1={A.x} y1={A.y} x2={B.x} y2={B.y} stroke="var(--border)" strokeWidth="1" />;
          })}
          {NODES.map((n) => (
            <g key={n.id} onClick={() => setSel(n.id)} style={{ cursor: "pointer" }}>
              <rect x={n.x - 7} y={n.y - 7} width={14} height={14}
                fill={nodeFill(n)} stroke={nodeStroke(n)} strokeWidth={n.sev || sel === n.id ? 2 : 1} />
              {sel === n.id && <rect x={n.x - 12} y={n.y - 12} width={24} height={24} fill="none" stroke="var(--accent)" strokeWidth="1" strokeDasharray="3 2" />}
              <text x={n.x + 14} y={n.y + 4} style={{ font: "500 11px var(--font-mono)", fill: n.sev ? SEVC[n.sev] : "var(--text-muted)" }}>{n.label}</text>
            </g>
          ))}
        </svg>
        <div style={{ display: "flex", gap: 20, alignItems: "center", borderTop: "1px solid var(--border-soft)", padding: "10px 16px", font: "400 11px var(--font-mono)", color: "var(--text-muted)" }}>
          <span style={{ color: "var(--ink)", fontWeight: 500 }}>{cur.label}</span>
          <span>kind: {cur.kind}</span>
          {cur.sev && <span style={{ color: SEVC[cur.sev] }}>worst finding: {cur.sev}</span>}
          <a href="#" onClick={(e) => e.preventDefault()} style={{ marginLeft: "auto" }}>open in inventory →</a>
        </div>
      </Card>
    </div>
  );
}
window.VergeApp = Object.assign(window.VergeApp || {}, { GraphView });
})();
