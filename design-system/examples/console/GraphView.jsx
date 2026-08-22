import React from "react";
import { Card } from "../../components/display/Card.jsx";
import { Graph, GraphLegend } from "../../components/display/Graph.jsx";
import { KeyValueList } from "../../components/display/KeyValueList.jsx";
import { SeverityBadge } from "../../components/display/SeverityBadge.jsx";
import { Drawer } from "../../components/feedback/Drawer.jsx";
import { Button } from "../../components/forms/Button.jsx";
import { Select } from "../../components/forms/Select.jsx";
import { Icon } from "../../components/media/Icon.jsx";

const subs = [
  ["www", "203.0.113.4", "low"], ["api", "203.0.113.9", "high"], ["vpn", "203.0.113.12", "critical"],
  ["mail", "203.0.113.25", null], ["grafana", "203.0.113.31", "high"], ["edge-gw-03", "203.0.113.7", "critical"],
  ["build-07", "203.0.113.44", "high"], ["assets", "203.0.113.18", "medium"], ["old-blog", null, "medium"], ["staging-4", "203.0.113.61", "info"],
];
const NODES = [{ id: "acmecorp.io", label: "acmecorp.io", type: "domain", x: 90, y: 320, sev: "medium" }];
const EDGES = [];
const ips = [];
subs.forEach(([s, ip, sev], i) => {
  const id = s + ".acmecorp.io";
  NODES.push({ id, label: s, type: "subdomain", x: 400, y: 52 + i * 60, sev: sev || undefined });
  EDGES.push({ from: "acmecorp.io", to: id });
  if (ip && ips.indexOf(ip) === -1) ips.push(ip);
});
ips.forEach((ip, i) => { NODES.push({ id: ip, label: ip, type: "ip", x: 730, y: 80 + i * 58 }); });
subs.forEach(([s, ip]) => { if (ip) EDGES.push({ from: s + ".acmecorp.io", to: ip }); });
const SERVICES = [
  [":443 https", "203.0.113.4", null], [":5900 vnc", "203.0.113.7", "critical"], [":443 nginx", "203.0.113.9", "high"],
  [":443 tls", "203.0.113.12", "critical"], [":25 smtp", "203.0.113.25", "low"], [":3000 http", "203.0.113.31", "high"], [":22 ssh", "203.0.113.44", "high"],
];
SERVICES.forEach(([label, ip, sev], i) => {
  const id = "svc-" + i;
  NODES.push({ id, label, type: "service", x: 1030, y: 100 + i * 68, sev: sev || undefined });
  EDGES.push({ from: ip, to: id });
});

const DETAILS = {
  "edge-gw-03.acmecorp.io": { signals: [["critical", "VNC exposed to internet"]], ports: ":443 :5900", first: "2026-08-12T09:14:33Z" },
  "vpn.acmecorp.io": { signals: [["critical", "TLS certificate expired"]], ports: ":443 :1194", first: "2026-06-02T11:40:00Z" },
  "api.acmecorp.io": { signals: [["high", "Outdated nginx"]], ports: ":443", first: "2026-05-19T08:00:12Z" },
  "acmecorp.io": { signals: [["medium", "SPF record missing"]], ports: ":80 :443", first: "2026-05-19T07:58:41Z" },
};

export function GraphView() {
  const [sel, setSel] = React.useState(null);
  const [sevFilter, setSevFilter] = React.useState("All severities");
  const nodes = sevFilter === "All severities" ? NODES : NODES.map((n) => (n.sev === sevFilter.toLowerCase() ? n : { ...n, sev: undefined }));
  const d = sel && DETAILS[sel.id];
  return (
    <main data-screen-label="Graph" style={{ maxWidth: 1440, margin: "0 auto", padding: 32, display: "flex", flexDirection: "column", gap: 20 }}>
      <header style={{ display: "flex", alignItems: "center", gap: 16 }}>
        <div style={{ display: "flex", flexDirection: "column", gap: 2 }}>
          <h1 style={{ margin: 0, font: "600 21px var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)" }}>Graph</h1>
          <span style={{ font: "400 12.5px var(--font-ui)", color: "var(--text-muted)", whiteSpace: "nowrap" }}>How your assets connect. Halos mark open signals; drag to pan.</span>
        </div>
        <div style={{ marginLeft: "auto", display: "flex", gap: 8, alignItems: "center" }}>
          <Select options={["All severities", "Critical", "High", "Medium", "Low", "Info"]} value={sevFilter} onChange={(e) => setSevFilter(e.target.value)} style={{ width: 170 }} />
          <Button variant="secondary" icon={<Icon name="download" size={14} />}>Export PNG</Button>
        </div>
      </header>
      <Card pad={0}>
        <Graph nodes={nodes} edges={EDGES} height={560} minimap selectedId={sel ? sel.id : undefined} onNodeSelect={setSel} />
        <div style={{ padding: "12px 20px", borderTop: "1px solid var(--row-sep)" }}>
          <GraphLegend />
        </div>
      </Card>
      <Drawer open={!!sel} width={420} onClose={() => setSel(null)}
        title={sel ? sel.id : ""} description={sel ? sel.type : ""}
        footer={sel && <Button variant="secondary" onClick={() => setSel(null)}>Close</Button>}>
        {sel && (
          <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
            <KeyValueList columns={1} items={[
              { k: "Node", v: sel.id },
              { k: "Type", v: sel.type, mono: false },
              { k: "Open ports", v: (d && d.ports) || "\u2014" },
              { k: "First seen", v: (d && d.first) || "2026-08-19T02:12:33Z" },
            ]} />
            <div>
              <div style={{ font: "500 11px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--text-muted)", marginBottom: 10 }}>Open signals</div>
              {d && d.signals ? (
                <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
                  {d.signals.map(([lvl, title]) => (
                    <div key={title} style={{ display: "flex", alignItems: "center", gap: 10 }}>
                      <SeverityBadge level={lvl} size="sm" />
                      <span style={{ font: "500 13px var(--font-ui)", color: "var(--text-ink)" }}>{title}</span>
                    </div>
                  ))}
                </div>
              ) : (
                <span style={{ font: "400 12.5px var(--font-ui)", color: "var(--text-muted)" }}>No open signals on this node.</span>
              )}
            </div>
          </div>
        )}
      </Drawer>
    </main>
  );
}
