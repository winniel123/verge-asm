import React from "react";
import { Table } from "../../components/display/Table.jsx";
import { Tag } from "../../components/display/Tag.jsx";
import { SeverityBadge } from "../../components/display/SeverityBadge.jsx";
import { Pagination } from "../../components/display/Pagination.jsx";
import { Button } from "../../components/forms/Button.jsx";
import { TagInput } from "../../components/forms/TagInput.jsx";
import { BulkActionsBar } from "../../components/feedback/BulkActionsBar.jsx";
import { TransitionMarker } from "../../components/display/TransitionMarker.jsx";
import { GapBadge } from "../../components/display/GapBadge.jsx";
import { ConfirmDialog } from "../../components/feedback/ConfirmDialog.jsx";
import { SavedViews } from "../../components/navigation/SavedViews.jsx";
import { ColumnPicker } from "../../components/display/ColumnPicker.jsx";
import { Icon } from "../../components/media/Icon.jsx";
import { SegmentedControl } from "../../components/forms/SegmentedControl.jsx";
import { HoverCard } from "../../components/feedback/HoverCard.jsx";
import { KeyValueList } from "../../components/display/KeyValueList.jsx";

const ASSETS = [
  { asset: "acmecorp.io", type: "domain", ip: "203.0.113.4", ports: ":80 :443", sev: "medium", sigs: 3, seen: "2m" },
  { asset: "www.acmecorp.io", type: "subdomain", ip: "203.0.113.4", ports: ":443", sev: "low", sigs: 1, seen: "2m" },
  { asset: "api.acmecorp.io", type: "subdomain", ip: "203.0.113.9", ports: ":443", sev: "high", sigs: 2, seen: "4m" },
  { asset: "edge-gw-03.acmecorp.io", type: "subdomain", ip: "203.0.113.7", ports: ":443 :5900", sev: "critical", sigs: 1, seen: "4m", change: "changed", changeTime: "4m" },
  { asset: "vpn.acmecorp.io", type: "subdomain", ip: "203.0.113.12", ports: ":443 :1194", sev: "critical", sigs: 1, seen: "12m" },
  { asset: "grafana.acmecorp.io", type: "subdomain", ip: "203.0.113.31", ports: ":3000", sev: "high", sigs: 1, seen: "26m" },
  { asset: "build-07.acmecorp.io", type: "subdomain", ip: "203.0.113.44", ports: ":22", sev: "high", sigs: 1, seen: "2h" },
  { asset: "mail.acmecorp.io", type: "subdomain", ip: "203.0.113.25", ports: ":25 :587", sev: null, sigs: 0, seen: "1h", change: "returned", changeTime: "6h" },
  { asset: "assets.acmecorp.io", type: "subdomain", ip: "203.0.113.18", ports: ":443", sev: "medium", sigs: 1, seen: "5h" },
  { asset: "old-blog.acmecorp.io", type: "subdomain", ip: "\u2014", ports: "\u2014", sev: "medium", sigs: 1, seen: "3h", change: "descoped", changeTime: "1d", changeReason: "operator excluded subtree" },
  { asset: "staging-4.acmecorp.io", type: "subdomain", ip: "203.0.113.61", ports: ":443", sev: "info", sigs: 1, seen: "3d", change: "appeared", changeTime: "3d" },
  { asset: "203.0.113.0/24", type: "range", ip: "\u2014", ports: "62 IPs", sev: null, sigs: 0, seen: "38m" },
];

export function Inventory({ onToast, onOpenAsset }) {
  const [filters, setFilters] = React.useState([]);
  const [sel, setSel] = React.useState([]);
  const [page, setPage] = React.useState(1);
  const [confirmRemove, setConfirmRemove] = React.useState(false);
  const [savedViews, setSavedViews] = React.useState([
    { id: "all", label: "All assets", count: 1284, filters: [] },
    { id: "ranges", label: "Ranges", filters: ["type:range"] },
    { id: "crit", label: "Critical only", filters: ["sev:critical"] },
  ]);
  const [view, setView] = React.useState("all");
  const [visCols, setVisCols] = React.useState(["type", "ip", "ports", "sigs", "change", "seen"]);
  const [dens, setDens] = React.useState("compact");
  const activeView = savedViews.filter((v) => v.id === view)[0] || savedViews[0];
  const viewDirty = JSON.stringify(filters) !== JSON.stringify(activeView.filters);
  const rows = ASSETS.filter((a) => filters.every((f) => {
    const [k, v] = f.includes(":") ? f.split(":", 2) : [null, f];
    if (k === "type") return a.type === v;
    if (k === "sev") return a.sev === v;
    return (a.asset + " " + a.ip + " " + a.type).toLowerCase().includes((v || f).toLowerCase());
  }));
  const act = (label) => () => {
    onToast && onToast({ tone: "ok", title: label, description: sel.length.toLocaleString("en-US") + " assets queued." });
    setSel([]);
  };
  const ALL_COLS = [
        { key: "asset", label: "Asset", mono: true, sortable: true, render: (r) => (
          <HoverCard content={
            <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
              <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
                <span style={{ font: "600 12.5px var(--font-mono)", color: "var(--text-ink)" }}>{r.asset}</span>
                <Tag>{r.type}</Tag>
              </div>
              <KeyValueList items={[
                { k: "IP", v: r.ip },
                { k: "Ports", v: r.ports },
                { k: "Signals", v: r.sigs ? r.sigs + " open \u00b7 top " + r.sev : "none" },
                { k: "Seen", v: r.seen + " ago" },
              ]} />
            </div>
          }>
            <span style={{ font: "500 12.5px var(--font-mono)", color: "var(--text-ink)" }}>{r.asset}</span>
          </HoverCard>
        ) },
        { key: "type", label: "Type", width: 132, sortable: true, render: (r) => <Tag>{r.type}</Tag> },
        { key: "ip", label: "IP", mono: true, width: 130, render: (r) => r.ip === "\u2014" ? (r.type === "range" ? <span style={{ font: "400 12px var(--font-mono)", color: "var(--text-muted)" }}>—</span> : <GapBadge size="sm" label="no address" />) : r.ip },
        { key: "ports", label: "Open ports", mono: true, width: 120 },
        { key: "sigs", label: "Signals", width: 150, sortable: true, sortValue: (r) => r.sigs, render: (r) => r.sigs
          ? <span style={{ display: "inline-flex", alignItems: "center", gap: 8 }}><SeverityBadge level={r.sev} size="sm" /><span style={{ font: "500 12px var(--font-mono)", color: "var(--text-secondary)" }}>{r.sigs}</span></span>
          : <span style={{ font: "400 12px var(--font-mono)", color: "var(--text-muted)" }}>—</span> },
        { key: "change", label: "Change", width: 64, render: (r) => r.change ? <TransitionMarker change={r.change} time={r.changeTime} reason={r.changeReason} /> : <span style={{ font: "400 12px var(--font-mono)", color: "var(--text-muted)" }}>—</span> },
        { key: "seen", label: "Seen", mono: true, align: "right", width: 60 },
      ];
  return (
    <main data-screen-label="Inventory" style={{ maxWidth: 1440, margin: "0 auto", padding: 32, display: "flex", flexDirection: "column", gap: 20 }}>
      <header style={{ display: "flex", alignItems: "center", gap: 16 }}>
        <div style={{ display: "flex", flexDirection: "column", gap: 2 }}>
          <h1 style={{ margin: 0, font: "600 21px var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)" }}>Inventory</h1>
          <span style={{ font: "400 12.5px var(--font-ui)", color: "var(--text-muted)", whiteSpace: "nowrap" }}>Everything you expose, watched for drift. <span style={{ fontFamily: "var(--font-mono)", fontSize: 12 }}>1,284</span> assets.</span>
        </div>
        <div style={{ marginLeft: "auto", display: "flex", gap: 8 }}>
          <Button variant="secondary" icon={<Icon name="download" size={14} />}>Export CSV</Button>
          <Button icon={<Icon name="plus" size={14} />}>Add seed</Button>
        </div>
      </header>
      <div style={{ display: "flex", gap: 12, alignItems: "center" }}>
        <SavedViews views={savedViews} activeId={view} dirty={viewDirty}
          onSelect={(id) => { const v = savedViews.filter((x) => x.id === id)[0]; setView(id); setFilters(v ? v.filters.slice() : []); setSel([]); }}
          onSave={() => { const id = "v" + Date.now(); const label = filters.length ? filters.join(" ") : "Everything"; setSavedViews(savedViews.concat({ id, label, filters: filters.slice() })); setView(id); onToast && onToast({ tone: "ok", title: "View saved", description: label }); }} />
        <span style={{ marginLeft: "auto", flex: "none", display: "inline-flex", gap: 8, alignItems: "center" }}>
          <SegmentedControl label="Row density" value={dens} onChange={setDens} options={[{ value: "comfortable", label: "Comfortable" }, { value: "compact", label: "Compact" }]} />
          <ColumnPicker visible={visCols} onChange={setVisCols} columns={[
            { key: "asset", label: "Asset", locked: true }, { key: "type", label: "Type" }, { key: "ip", label: "IP" },
            { key: "ports", label: "Open ports" }, { key: "sigs", label: "Signals" }, { key: "change", label: "Change" }, { key: "seen", label: "Seen" },
          ]} />
        </span>
      </div>
      <div style={{ display: "flex", gap: 12, alignItems: "flex-start" }}>
        <TagInput values={filters} onChange={(v) => { setFilters(v); setSel([]); }} placeholder="type:subdomain, sev:critical, edge"
          suggestions={["type:domain", "type:subdomain", "type:range", "sev:critical", "sev:high", "sev:medium", "sev:low", "sev:info"]}
          style={{ width: 420 }} />
        <span style={{ marginLeft: "auto", font: "400 12px var(--font-mono)", color: "var(--text-muted)", paddingTop: 10 }}>{rows.length} of {ASSETS.length} shown</span>
      </div>
      <Table density={dens} selectable onRowClick={onOpenAsset ? (r) => { if (r.type !== "range") onOpenAsset(r.asset); } : undefined} rowKey="asset" selectedKeys={sel} onSelectionChange={setSel} columns={ALL_COLS.filter((c) => c.key === "asset" || visCols.indexOf(c.key) !== -1)} rows={rows} />
      <div style={{ display: "flex", justifyContent: "flex-end" }}>
        <Pagination page={page} pageCount={107} pageSize={12} totalItems={1284} onChange={setPage} />
      </div>
      <ConfirmDialog open={confirmRemove} title="Remove assets"
        message={sel.length.toLocaleString("en-US") + (sel.length === 1 ? " asset leaves" : " assets leave") + " the inventory."}
        detail="Their spans close as descoped; they return automatically if discovery sees them again."
        confirmLabel="Remove assets"
        onConfirm={act("Assets removed")} onClose={() => setConfirmRemove(false)} />
      <BulkActionsBar count={sel.length} itemLabel={sel.length === 1 ? "asset" : "assets"} onClear={() => setSel([])} actions={[
        { label: "Rescan", icon: "play", onClick: act("Rescan queued") },
        { label: "Add tag", icon: "tag", onClick: act("Tag applied") },
        { label: "Annotate", icon: "pencil", onClick: act("Annotation recorded") },
        { label: "Remove", icon: "trash-2", tone: "danger", onClick: () => setConfirmRemove(true) },
      ]} />
    </main>
  );
}
