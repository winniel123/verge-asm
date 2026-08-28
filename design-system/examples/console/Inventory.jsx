import React from "react";
import { Table } from "../../components/display/Table.jsx";
import { Tag } from "../../components/display/Tag.jsx";
import { GapBadge } from "../../components/display/GapBadge.jsx";
import { ColumnPicker } from "../../components/display/ColumnPicker.jsx";
import { Button } from "../../components/forms/Button.jsx";
import { Input } from "../../components/forms/Input.jsx";
import { Switch } from "../../components/forms/Switch.jsx";
import { SegmentedControl } from "../../components/forms/SegmentedControl.jsx";
import { EmptyState } from "../../components/feedback/EmptyState.jsx";
import { Icon } from "../../components/media/Icon.jsx";

/* Inventory — a READ over the open-span corpus (ADR-0105): every open span grouped
   by subject, each facet's current value dated since it opened, Gaps shown as Gaps.
   No denominator, no count-of-subjects (ADR-0072). All scoping below is client-side
   over the fully rendered corpus — there is no server-side search (SPEC-CHANGE #13). */
const GROUPS = [
  { kind: "name", label: "Names", type: "Name", subjects: [
    { key: "www.acmecorp.io", facets: [
      { label: "resolution", summary: "A \u00b7 2 addresses", since: "2026-07-14", records: [{ type: "A", data: "198.51.100.7" }, { type: "A", data: "198.51.100.8" }] },
      { label: "dns-records", summary: "CNAME \u00b7 TXT", since: "2026-07-14", records: [{ type: "CNAME", data: "edge.acmecorp.io" }, { type: "TXT", data: "verge-custody=vg1:9f3k\u2026" }] },
    ] },
    { key: "api.acmecorp.io", facets: [
      { label: "resolution", summary: "A \u00b7 203.0.113.44", since: "2026-06-02" },
      { label: "dns-records", summary: "CAA \u00b7 1 record", since: "2026-06-02", records: [{ type: "CAA", data: "0 issue \u201cletsencrypt.org\u201d" }] },
    ] },
    { key: "mail.acmecorp.io", facets: [
      { label: "resolution", summary: "A \u00b7 203.0.113.25", since: "2026-05-19" },
      { label: "dns-records", gap: true, since: "2026-08-21" },
    ] },
  ] },
  { kind: "service", label: "Services", type: "Service", subjects: [
    { key: "198.51.100.7:443/tcp", facets: [
      { label: "tls-acceptance", summary: "TLS 1.2 \u00b7 1.3", since: "2026-07-14" },
      { label: "certificate-chain", summary: "leaf www.acmecorp.io \u00b7 exp 2026-11-02", since: "2026-08-03", records: [{ type: "leaf", data: "CN=www.acmecorp.io \u00b7 not_after 2026-11-02" }, { type: "int", data: "CN=R11 \u00b7 Let\u2019s Encrypt" }] },
    ] },
    { key: "203.0.113.44:22/tcp", facets: [
      { label: "reachability", summary: "answers \u00b7 22/tcp", since: "2026-04-30" },
      { label: "tls-acceptance", summary: "none \u00b7 plaintext ssh", since: "2026-04-30" },
    ] },
    { key: "198.51.100.31:8443/tcp", facets: [
      { label: "certificate-chain", gap: true, since: "2026-08-19" },
    ] },
  ] },
  { kind: "endpoint", label: "Endpoints", type: "Endpoint", subjects: [
    { key: "www.acmecorp.io \u00b7 :443 https", facets: [
      { label: "http-identity", summary: "nginx \u00b7 200 \u00b7 \u201cAcme \u2014 sign in\u201d", since: "2026-07-14" },
    ] },
    { key: "grafana.acmecorp.io \u00b7 :443 https", facets: [
      { label: "http-identity", summary: "Grafana \u00b7 302 \u2192 /login", since: "2026-06-27" },
    ] },
  ] },
  { kind: "address", label: "Addresses", type: "Address", subjects: [
    { key: "198.51.100.7", facets: [
      { label: "reachability \u00b7 vantage 1", summary: "answers \u00b7 443/tcp", since: "2026-07-14" },
      { label: "reachability \u00b7 vantage 3", summary: "answers \u00b7 443/tcp \u00b7 8443/tcp", since: "2026-08-02" },
    ] },
    { key: "203.0.113.44", facets: [
      { label: "reachability \u00b7 prober", summary: "answers \u00b7 22/tcp", since: "2026-04-30" },
    ] },
    { key: "104.18.22.90", proxy: true, facets: [
      { label: "reachability \u00b7 vantage 1", gap: true, proxy: true, since: "2026-08-19" },
    ] },
  ] },
];

export function Inventory({ onToast, onOpenAsset, onOpenSubject, onOpenScope }) {
  const [kind, setKind] = React.useState("all");
  const [gapsOnly, setGapsOnly] = React.useState(false);
  const [hideProxy, setHideProxy] = React.useState(false);
  const [q, setQ] = React.useState("");
  const [visCols, setVisCols] = React.useState(["type", "holds", "since"]);
  const [dens, setDens] = React.useState("compact");
  const [open, setOpen] = React.useState({});
  const [collapsed, setCollapsed] = React.useState({});
  const [showAll, setShowAll] = React.useState({});
  const CAP = 25;
  const toggle = (k) => setOpen((o) => Object.assign({}, o, { [k]: !o[k] }));
  const groups = GROUPS
    .filter((g) => kind === "all" || g.kind === kind)
    .map((g) => Object.assign({}, g, { subjects: g.subjects.filter((s) =>
      (!gapsOnly || s.facets.some((f) => f.gap)) &&
      (!hideProxy || !s.proxy) &&
      (!q.trim() || s.key.toLowerCase().includes(q.trim().toLowerCase()))
    ) }))
    .filter((g) => g.subjects.length > 0);
  const totalShown = groups.reduce((n, g) => n + g.subjects.length, 0);
  const anyCollapsed = groups.some((g) => collapsed[g.kind]);
  const setAllCollapsed = (v) => setCollapsed(groups.reduce((m, g) => Object.assign(m, { [g.kind]: v }), {}));
  const openFor = { name: onOpenAsset ? (r) => onOpenAsset(r.key) : undefined, service: onOpenSubject ? () => onOpenSubject("service") : undefined, endpoint: onOpenSubject ? () => onOpenSubject("endpoint") : undefined, address: undefined };
  const cols = (g) => [
    { key: "key", label: "Subject", mono: true, render: (r) => <span style={{ font: "500 12.5px var(--font-mono)", color: openFor[g.kind] ? "var(--link)" : "var(--text-ink)" }}>{r.key}</span> },
    { key: "type", label: "Type", width: 110, render: () => <Tag>{g.type}</Tag> },
    { key: "holds", label: "Holds", render: (r) => (
      <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
        {r.facets.map((f) => {
          const ok = r.key + "|" + f.label;
          return (
            <div key={f.label} style={{ display: "flex", gap: 10, alignItems: "baseline" }}>
              <span style={{ font: "500 10.5px var(--font-mono)", letterSpacing: "0.06em", textTransform: "uppercase", color: "var(--text-muted)", minWidth: 168, whiteSpace: "nowrap" }}>{f.label}</span>
              {f.gap ? (
                <span style={{ display: "inline-flex", gap: 8, alignItems: "baseline" }}>
                  <GapBadge size="sm" />
                  {f.proxy && <span style={{ display: "inline-flex", alignItems: "center", height: 18, padding: "0 7px", borderRadius: 999, background: "var(--surface-sunken)", border: "1px solid var(--border-default)", font: "500 10px var(--font-mono)", letterSpacing: "0.04em", textTransform: "uppercase", color: "var(--text-muted)" }}>proxy edge</span>}
                  <span style={{ font: "400 11px var(--font-mono)", color: "var(--text-muted)" }}>since {f.since}</span>
                </span>
              ) : f.records ? (
                <span style={{ display: "flex", flexDirection: "column", gap: 4 }}>
                  <button onClick={(e) => { e.stopPropagation(); toggle(ok); }} aria-expanded={!!open[ok]} style={{ display: "inline-flex", alignItems: "center", gap: 5, background: "none", border: "none", padding: 0, cursor: "pointer", font: "inherit" }}>
                    <Tag>{f.summary}</Tag>
                    <Icon name="chevron-down" size={12} style={{ color: "var(--text-muted)", transform: open[ok] ? "rotate(180deg)" : "none", transition: "transform var(--dur-fast) var(--ease-out)" }} />
                  </button>
                  <span aria-hidden={!open[ok]} style={{ display: "grid", gridTemplateRows: open[ok] ? "1fr" : "0fr", opacity: open[ok] ? 1 : 0, marginTop: open[ok] ? 0 : -4, transition: "grid-template-rows var(--dur-base) var(--ease-out), opacity var(--dur-base) var(--ease-out), margin-top var(--dur-base) var(--ease-out)" }}>
                    <span style={{ minHeight: 0, overflow: "hidden", display: "flex", flexDirection: "column", gap: 3, paddingLeft: 2 }}>
                      {f.records.map((rec) => (
                        <span key={rec.data} style={{ display: "grid", gridTemplateColumns: "56px 1fr", gap: 8, alignItems: "baseline" }}>
                          <span style={{ font: "500 10px var(--font-mono)", letterSpacing: "0.05em", textTransform: "uppercase", color: "var(--text-muted)" }}>{rec.type}</span>
                          <span style={{ font: "400 12px var(--font-mono)", color: "var(--text-body)", overflowWrap: "anywhere" }}>{rec.data}</span>
                        </span>
                      ))}
                    </span>
                  </span>
                </span>
              ) : f.summary ? <Tag>{f.summary}</Tag> : <span style={{ font: "400 11.5px var(--font-mono)", color: "var(--text-muted)" }}>none</span>}
            </div>
          );
        })}
      </div>
    ) },
    { key: "since", label: "Since", mono: true, align: "right", width: 112, clip: false, render: (r) => <span style={{ color: "var(--text-muted)", verticalAlign: "top", whiteSpace: "nowrap" }}>{r.facets[0].since}</span> },
  ].filter((c) => c.key === "key" || visCols.indexOf(c.key) !== -1);
  return (
    <main data-screen-label="Inventory" style={{ maxWidth: 1440, margin: "0 auto", padding: 32, display: "flex", flexDirection: "column", gap: 20 }}>
      <header style={{ display: "flex", alignItems: "flex-start", gap: 16 }}>
        <div style={{ display: "flex", flexDirection: "column", gap: 2 }}>
          <h1 style={{ margin: 0, font: "600 21px var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)" }}>Inventory</h1>
          <span style={{ font: "400 12.5px var(--font-ui)", color: "var(--text-muted)" }}>Everything you expose, watched for drift — the actual values behind the verdicts.</span>
        </div>
        <div style={{ marginLeft: "auto", display: "flex", gap: 8 }}>
          <Button variant="secondary" icon={<Icon name="download" size={14} />} onClick={() => onToast && onToast({ tone: "neutral", title: "Export started", description: "inventory-2026-08-24.csv \u00b7 one row per held facet" })}>Export CSV</Button>
          <Button icon={<Icon name="plus" size={14} />} onClick={onOpenScope}>Add seed</Button>
        </div>
      </header>
      <p style={{ margin: 0, maxWidth: "82ch", font: "400 13px/1.6 var(--font-ui)", color: "var(--text-secondary)" }}>What your estate holds right now — the addresses a name resolves to, the records it carries, the certificate a service presents, the identity an endpoint returns. Each row is a subject; open it for its full record, or expand a value to its individual records. A withdrawn subject holds no current span and so is not here. There is no total: your estate’s completeness is yours alone to state.</p>
      <div style={{ position: "sticky", top: 0, zIndex: 5, display: "flex", gap: 14, alignItems: "center", flexWrap: "wrap", padding: "12px 0", margin: "0 -32px", paddingLeft: 32, paddingRight: 32, background: "color-mix(in srgb, var(--bg-page) 92%, transparent)", backdropFilter: "blur(8px)", borderBottom: "1px solid var(--row-sep)" }}>
        <SegmentedControl label="Kind" value={kind} onChange={setKind} options={[
          { value: "all", label: "All subjects" }, { value: "name", label: "Names" }, { value: "service", label: "Services" }, { value: "endpoint", label: "Endpoints" }, { value: "address", label: "Addresses" },
        ]} />
        <Switch checked={gapsOnly} onChange={setGapsOnly} label="Gaps only" />
        <Switch checked={hideProxy} onChange={setHideProxy} label="Hide proxy edge" />
        <Input placeholder="Filter subjects…" value={q} onChange={(e) => setQ(e.target.value)} style={{ width: 230 }} />
        <span style={{ font: "400 11.5px var(--font-mono)", color: "var(--text-muted)", whiteSpace: "nowrap" }}>{totalShown} subject{totalShown === 1 ? "" : "s"} shown</span>
        <span style={{ marginLeft: "auto", display: "inline-flex", gap: 8, alignItems: "center" }}>
          <Button variant="ghost" size="sm" onClick={() => setAllCollapsed(!anyCollapsed)}>{anyCollapsed ? "Expand all" : "Collapse all"}</Button>
          <SegmentedControl label="Row density" value={dens} onChange={setDens} options={[{ value: "comfortable", label: "Comfortable" }, { value: "compact", label: "Compact" }]} />
          <ColumnPicker visible={visCols} onChange={setVisCols} columns={[
            { key: "key", label: "Subject", locked: true }, { key: "type", label: "Type" }, { key: "holds", label: "Holds" }, { key: "since", label: "Since" },
          ]} />
        </span>
      </div>
      {groups.length === 0 ? (
        <EmptyState icon="search" message="No subjects match this scope." detail="The corpus is unaffected — only the view is scoped."
          action={<Button variant="secondary" onClick={() => { setKind("all"); setGapsOnly(false); setHideProxy(false); setQ(""); }}>Clear filters</Button>} />
      ) : groups.map((g) => {
        const isColl = collapsed[g.kind];
        const capped = !showAll[g.kind] && g.subjects.length > CAP;
        const rows = capped ? g.subjects.slice(0, CAP) : g.subjects;
        return (
        <section key={g.kind} style={{ display: "flex", flexDirection: "column", gap: 10 }}>
          <button type="button" onClick={() => setCollapsed((c) => Object.assign({}, c, { [g.kind]: !c[g.kind] }))}
            style={{ display: "inline-flex", alignItems: "center", gap: 8, alignSelf: "flex-start", background: "none", border: "none", padding: 0, cursor: "pointer" }}>
            <Icon name="chevron-down" size={13} style={{ color: "var(--text-muted)", transform: isColl ? "rotate(-90deg)" : "none", transition: "transform var(--dur-fast) var(--ease-out)" }} />
            <span style={{ font: "500 10.5px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--text-muted)" }}>{g.label}</span>
            <span style={{ font: "400 10.5px var(--font-mono)", color: "var(--text-muted)" }}>{g.subjects.length}</span>
          </button>
          {!isColl && <Table density={dens} rowKey="key" onRowClick={openFor[g.kind]} columns={cols(g)} rows={rows} />}
          {!isColl && capped && (
            <button type="button" onClick={() => setShowAll((s) => Object.assign({}, s, { [g.kind]: true }))}
              style={{ alignSelf: "flex-start", background: "none", border: "none", padding: "2px 0", cursor: "pointer", font: "500 12px var(--font-ui)", color: "var(--link)" }}>
              Show all {g.subjects.length} — {g.subjects.length - CAP} more
            </button>
          )}
        </section>
        );
      })}
    </main>
  );
}
