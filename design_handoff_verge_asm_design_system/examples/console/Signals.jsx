import React from "react";
import { Card } from "../../components/display/Card.jsx";
import { Table } from "../../components/display/Table.jsx";
import { SeverityBadge } from "../../components/display/SeverityBadge.jsx";
import { Tag } from "../../components/display/Tag.jsx";
import { Button } from "../../components/forms/Button.jsx";
import { Input } from "../../components/forms/Input.jsx";
import { Select } from "../../components/forms/Select.jsx";
import { Tabs } from "../../components/navigation/Tabs.jsx";
import { Drawer } from "../../components/feedback/Drawer.jsx";
import { Timeline } from "../../components/display/Timeline.jsx";
import { KeyValueList } from "../../components/display/KeyValueList.jsx";
import { RelativeTime } from "../../components/display/RelativeTime.jsx";
import { CopyValue } from "../../components/display/CopyValue.jsx";
import { DiffView } from "../../components/display/DiffView.jsx";
import { EmptyState } from "../../components/feedback/EmptyState.jsx";
import { AnnotationControl } from "../../components/feedback/AnnotationControl.jsx";
import { ConfirmDialog } from "../../components/feedback/ConfirmDialog.jsx";
import { ContextMenu } from "../../components/feedback/ContextMenu.jsx";
import { DropdownMenu } from "../../components/feedback/DropdownMenu.jsx";
import { Pagination } from "../../components/display/Pagination.jsx";
import { IconButton } from "../../components/forms/IconButton.jsx";
import { Icon } from "../../components/media/Icon.jsx";
import { SIGNALS } from "./SignalData.jsx";
import { WithdrawnMark } from "../../components/display/WithdrawnMark.jsx";
import { SignalRuleRef } from "../../components/display/SignalRuleRef.jsx";

const RULE_REFS = { "SIG-1042": ["vnc-exposure", 3], "SIG-1041": ["tls-acceptance", 2], "SIG-1039": ["dns-mail-policy", 1] };
const DRIFT = {
  "SIG-1042": { title: "Open ports \u00b7 drift", lines: [
    { type: "same", text: ":443 https nginx/1.25.4" },
    { type: "add", text: ":5900 vnc \u2014 no transport encryption" },
    { type: "remove", text: ":8080 http-alt" },
  ] },
  "SIG-1036": { title: "Service banner \u00b7 drift", lines: [
    { type: "remove", text: "nginx/1.24.0" },
    { type: "add", text: "nginx/1.25.0 (CVE-2026-1187)" },
  ] },
};

function history(s) {
  const ev = [{ title: "Still present", detail: "vantage eu-west-1 re-confirmed", time: s.seen, tone: "accent" }];
  if (DRIFT[s.id]) ev.push({ title: "Drift detected", detail: s.asset + " changed", time: s.seen, tone: "warn", mono: true });
  ev.push({ title: "Signal raised", detail: s.id, time: s.first, tone: s.sev === "critical" || s.sev === "high" ? "danger" : "neutral", mono: true });
  ev.push({ title: "Asset discovered", detail: s.asset, time: "2026-08-12", tone: "neutral", mono: true });
  return ev;
}

const WITHDRAWN = [
  { id: "SIG-0991", sev: "medium", title: "Directory listing enabled", asset: "files.acmecorp.io", ip: "203.0.113.21", port: ":443", seen: "6d", first: "2026-07-30T10:11:00Z", last: "2026-08-16T09:00:00Z", cve: null, tags: ["http"], desc: "Autoindex stopped answering in a later batch. The key is in no current population — withdrawn on read, no operator act.", withdrawn: true },
  { id: "SIG-0968", sev: "high", title: "Admin panel reachable", asset: "jenkins.acmecorp.io", ip: "203.0.113.29", port: ":8080", seen: "12d", first: "2026-07-18T08:40:22Z", last: "2026-08-10T13:25:41Z", cve: null, tags: ["jenkins", "admin"], desc: "The service left the population; the world moved and the signal withdrew itself.", withdrawn: true },
  { id: "SIG-0944", sev: "low", title: "Verbose server header", asset: "cdn.acmecorp.io", ip: "203.0.113.66", port: ":443", seen: "21d", first: "2026-06-29T12:02:19Z", last: "2026-08-01T07:12:00Z", cve: null, tags: ["http", "header"], desc: "Header trimmed upstream; key absent from the current population.", withdrawn: true },
];

export function Signals({ onAnnotate, onToast }) {
  const [q, setQ] = React.useState("");
  const [sev, setSev] = React.useState("All severities");
  const [tab, setTab] = React.useState("open");
  const [sel, setSel] = React.useState(null);
  const SEV_RANK = { critical: 0, high: 1, medium: 2, low: 3, info: 4 };
  const t2m = (s) => { const m = /(\d+)([mhd])/.exec(s || ""); return m ? +m[1] * (m[2] === "m" ? 1 : m[2] === "h" ? 60 : 1440) : 9e9; };
  const [page, setPage] = React.useState(1);
  const [descope, setDescope] = React.useState(null);
  const [ctx, setCtx] = React.useState(null);
  const [annotations, setAnnotations] = React.useState({
    "SIG-1027": { reason: "Third-party mail provider publishes SPF on our behalf." },
    "SIG-1024": { reason: "Public banner is intentional — accepted." },
  });
  const base = tab === "withdrawn" ? WITHDRAWN : tab === "annotated" ? SIGNALS.filter((s) => annotations[s.id]) : SIGNALS;
  const rows = base.filter((s) =>
    (sev === "All severities" || s.sev === sev.toLowerCase()) &&
    (!q || (s.title + " " + s.asset + " " + s.id).toLowerCase().includes(q.toLowerCase()))
  );
  const open = sel;
  return (
    <main data-screen-label="Signals" style={{ maxWidth: 1440, margin: "0 auto", padding: 32, display: "flex", flexDirection: "column", gap: 20 }}>
      <header style={{ display: "flex", alignItems: "center", gap: 16 }}>
        <div style={{ display: "flex", flexDirection: "column", gap: 2 }}>
          <h1 style={{ margin: 0, font: "600 21px var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)" }}>Signals</h1>
          <span style={{ font: "400 12.5px var(--font-ui)", color: "var(--text-muted)", whiteSpace: "nowrap" }}>Raised when your attack surface drifts. Severity is Critical → Info.</span>
        </div>
        <div style={{ marginLeft: "auto", display: "flex", gap: 8 }}>
          <Button variant="secondary" icon={<Icon name="download" size={14} />}>Export CSV</Button>
        </div>
      </header>
      <Tabs active={tab} onChange={(t) => { setTab(t); setSel(null); setPage(1); }} tabs={[
        { id: "open", label: "Open", count: 47 },
        { id: "annotated", label: "Annotated", count: Object.keys(annotations).length },
        { id: "withdrawn", label: "Withdrawn", count: WITHDRAWN.length },
      ]} />
      <div style={{ display: "flex", gap: 12, alignItems: "flex-end" }}>
        <Input mono prefix={<Icon name="search" size={14} />} placeholder="Search signals, assets, ids" value={q}
          onChange={(e) => { setQ(e.target.value); setSel(null); }} style={{ width: 320 }} />
        <Select options={["All severities", "Critical", "High", "Medium", "Low", "Info"]} value={sev}
          onChange={(e) => { setSev(e.target.value); setSel(null); }} style={{ width: 170 }} />
        <span style={{ marginLeft: "auto", font: "400 12px var(--font-mono)", color: "var(--text-muted)" }}>{rows.length} of {base.length} shown</span>
      </div>
      {rows.length === 0 ? (
        <Card>
          <EmptyState icon="search" message="No signals match your filters." detail="Clear the search or severity filter to see open signals."
            action={<Button variant="secondary" onClick={() => { setQ(""); setSev("All severities"); }}>Clear filters</Button>} />
        </Card>
      ) : (
        <Table dense columns={[
          { key: "sev", label: "Severity", width: 104, sortable: true, sortValue: (r) => SEV_RANK[r.sev], render: (r) => <SeverityBadge level={r.sev} size="sm" /> },
          { key: "title", label: "Signal", render: (r) => <span style={{ display: "inline-flex", alignItems: "center", gap: 8 }}><span style={{ font: "500 13px var(--font-ui)", color: "var(--text-ink)" }}>{r.title}</span>{r.withdrawn && <WithdrawnMark size="sm" />}</span> },
          { key: "asset", label: "Asset", mono: true, sortable: true },
          { key: "port", label: "Port", mono: true, width: 76 },
          { key: "id", label: "Id", mono: true, width: 112, sortable: true },
          { key: "seen", label: "Seen", mono: true, align: "right", width: 60, sortable: true, sortValue: (r) => t2m(r.seen), render: (r) => <RelativeTime value={r.seen} iso={r.last} side="left" /> },
          { key: "actions", label: "", width: 58, align: "right", clip: false, render: (r) => (
            <DropdownMenu trigger={<IconButton icon="ellipsis" label="Actions" size="sm" />} items={[
              { label: "Annotate — accept risk", icon: "pencil", onSelect: () => setSel(r) },
              { label: "Copy asset", icon: "copy", shortcut: (/Mac|iP(hone|ad|od)/.test(navigator.platform || navigator.userAgent) ? "\u2318C" : "Ctrl+C") },
              "-",
              { label: "Descope seed", icon: "trash-2", tone: "danger", onSelect: () => setDescope(r) },
            ]} />
          ) },
        ]} rows={rows} rowKey="id" selectedKeys={open ? [open.id] : []} onRowClick={(r) => setSel(r)} onRowContextMenu={(r, i, e) => setCtx({ x: e.clientX, y: e.clientY, row: r })} initialSort={{ key: "sev", dir: "asc" }} />
      )}
      {tab === "open" && rows.length > 0 && (
        <div style={{ display: "flex", justifyContent: "flex-end" }}>
          <Pagination page={page} pageCount={5} pageSize={10} totalItems={47} onChange={setPage} />
        </div>
      )}
      <ContextMenu open={!!ctx} x={ctx ? ctx.x : 0} y={ctx ? ctx.y : 0} onClose={() => setCtx(null)} items={ctx ? [
        { label: "Open", icon: "panel-right", onSelect: () => setSel(ctx.row) },
        { label: "Annotate \u2014 accept risk", icon: "pencil", onSelect: () => setSel(ctx.row) },
        { label: "Copy asset", icon: "copy", onSelect: () => { if (navigator.clipboard) navigator.clipboard.writeText(ctx.row.asset); onToast && onToast({ tone: "neutral", title: "Copied", description: ctx.row.asset }); } },
        "-",
        { label: "Descope seed", icon: "trash-2", tone: "danger", onSelect: () => setDescope(ctx.row) },
      ] : []} />
      <ConfirmDialog open={!!descope} title="Descope seed"
        message={descope ? "Removes " + descope.asset + " and its subjects from scope." : ""}
        detail="Spans close as descoped in the next batch; the exclusion is recorded on the Scope screen."
        typedConfirm={descope ? descope.asset : undefined} confirmLabel="Descope seed"
        onConfirm={() => onToast && onToast({ tone: "neutral", title: "Seed descoped", description: descope.asset + " · recorded as an exclusion" })}
        onClose={() => setDescope(null)} />
      <Drawer open={!!open} width={480} onClose={() => setSel(null)}
        title={open ? open.title : ""}
        description={open ? "Raised " + open.seen + " ago \u00b7 " + open.id : ""}
        footer={open && <Button variant="secondary" onClick={() => setSel(null)}>Close</Button>}>
        {open && (
          <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
            <div style={{ display: "flex", gap: 8, alignItems: "center", flexWrap: "wrap" }}>
              <SeverityBadge level={open.sev} />
              {open.withdrawn && <WithdrawnMark size="sm" />}
              {open.tags.map((t) => <Tag key={t}>{t}</Tag>)}
              {open.cve && <Tag>{open.cve}</Tag>}
            </div>
            <p style={{ margin: 0, font: "400 13px/1.55 var(--font-ui)", color: "var(--text-body)" }}>{open.desc}</p>
            <KeyValueList items={[
              { k: "Asset", v: <CopyValue value={open.asset} /> },
              { k: "IP", v: open.ip === "\u2014" ? open.ip : <CopyValue value={open.ip} /> },
              { k: "Rule", v: <SignalRuleRef id={(RULE_REFS[open.id] || ["svc-exposure", 3])[0]} version={(RULE_REFS[open.id] || ["svc-exposure", 3])[1]} /> },
              { k: "Port", v: open.port },
              { k: "Detected by", v: "vantage eu-west-1" },
              { k: "First seen", v: open.first },
              { k: "Last seen", v: open.last },
            ]} />
            {DRIFT[open.id] && <DiffView title={DRIFT[open.id].title} lines={DRIFT[open.id].lines} />}
            <AnnotationControl annotation={annotations[open.id]}
              onAnnotate={(reason) => { setAnnotations({ ...annotations, [open.id]: { reason } }); onAnnotate && onAnnotate(open); }}
              onRemove={() => { const a = { ...annotations }; delete a[open.id]; setAnnotations(a); }} />
            <div>
              <div style={{ font: "500 11px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--text-muted)", marginBottom: 10 }}>History</div>
              <Timeline events={history(open)} />
            </div>
          </div>
        )}
      </Drawer>
    </main>
  );
}
