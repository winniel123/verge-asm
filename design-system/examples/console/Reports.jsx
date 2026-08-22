import React from "react";
import { Card } from "../../components/display/Card.jsx";
import { ReportCard } from "../../components/display/ReportCard.jsx";
import { Sparkline } from "../../components/display/Sparkline.jsx";
import { BarChart } from "../../components/display/BarChart.jsx";
import { Table } from "../../components/display/Table.jsx";
import { TimeSeriesChart } from "../../components/display/TimeSeriesChart.jsx";
import { HeatmapCalendar } from "../../components/display/HeatmapCalendar.jsx";
import { KeyValueList } from "../../components/display/KeyValueList.jsx";
import { Wizard } from "../../components/feedback/Wizard.jsx";
import { Input } from "../../components/forms/Input.jsx";
import { Checkbox } from "../../components/forms/Checkbox.jsx";
import { CadenceSelect } from "../../components/forms/CadenceSelect.jsx";
import { Tag } from "../../components/display/Tag.jsx";
import { Button } from "../../components/forms/Button.jsx";
import { DateRangePicker } from "../../components/forms/DateRangePicker.jsx";
import { SplitButton } from "../../components/forms/SplitButton.jsx";
import { IconButton } from "../../components/forms/IconButton.jsx";
import { DropdownMenu } from "../../components/feedback/DropdownMenu.jsx";
import { Icon } from "../../components/media/Icon.jsx";
import { SEV_ORDER, SEV_COUNTS } from "./SignalData.jsx";

const OPEN_TREND = [38, 41, 40, 44, 42, 45, 43, 46, 44, 43, 45, 44, 46, 47];
const RESOLVE_TREND = [3.4, 3.1, 3.3, 2.9, 3.0, 2.8, 2.9, 2.7, 2.6, 2.7, 2.5, 2.6, 2.5, 2.4];
const DISCOVERY = [2, 0, 1, 4, 3, 0, 2, 5, 1, 2, 0, 3, 4, 12];
const DAY_LABELS = ["Aug 9", "", "", "", "", "", "", "", "", "", "", "", "", "Aug 22"];
const CRIT_HIGH = [11, 12, 12, 13, 12, 14, 13, 14, 13, 13, 14, 13, 14, 15];
const DAYS_FULL = Array.from({ length: 14 }, (_, i) => "Aug " + (9 + i));
const DAYS_SPARSE = DAYS_FULL.map((d, i) => (i % 3 === 0 ? d : ""));
const SECTIONS = ["Summary KPIs", "New assets", "Signal changes", "Coverage gaps"];
const SCAN_DAYS = (() => { let s = 11, out = []; for (let i = 0; i < 84; i++) { s = (s * 16807) % 2147483647; out.push(i % 7 >= 5 ? (s % 3 === 0 ? 1 : 0) : (s % 5)); } return out; })();

const SCHEDULED = [
  { name: "Weekly exposure summary", cadence: "weekly \u00b7 mon 09:00", format: "pdf", last: "3d" },
  { name: "Monthly asset inventory", cadence: "monthly \u00b7 1st", format: "csv", last: "22d" },
  { name: "Critical signals digest", cadence: "daily \u00b7 08:00", format: "email", last: "14h" },
];

function SevBars() {
  const max = Math.max(...SEV_ORDER.map((s) => SEV_COUNTS[s]));
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
      {SEV_ORDER.map((s) => (
        <div key={s} style={{ display: "flex", alignItems: "center", gap: 12 }}>
          <span style={{ width: 72, font: "500 11px var(--font-mono)", letterSpacing: "0.06em", textTransform: "uppercase", color: "var(--text-secondary)" }}>{s}</span>
          <span style={{ flex: 1, height: 8, borderRadius: 999, background: "var(--surface-sunken)", overflow: "hidden" }}>
            <span style={{ display: "block", height: "100%", width: (SEV_COUNTS[s] / max) * 100 + "%", borderRadius: 999, background: "var(--sev-" + s + "-dot)" }} />
          </span>
          <span style={{ width: 26, textAlign: "right", font: "500 12.5px var(--font-mono)", color: "var(--text-body)" }}>{SEV_COUNTS[s]}</span>
        </div>
      ))}
    </div>
  );
}

const t2m = (s) => { const m = /(\d+)([mhd])/.exec(s || ""); return m ? +m[1] * (m[2] === "m" ? 1 : m[2] === "h" ? 60 : 1440) : 9e9; };

export function Reports({ onOpenArtifact }) {
  const [range, setRange] = React.useState({ label: "Last 7d" });
  const [rows, setRows] = React.useState(SCHEDULED);
  const [wizOpen, setWizOpen] = React.useState(false);
  const [name, setName] = React.useState("");
  const [secs, setSecs] = React.useState(SECTIONS.slice(0, 3));
  const [cad, setCad] = React.useState("Weekly \u00b7 mon 09:00");
  const [cron, setCron] = React.useState("");
  const openWiz = () => { setName(""); setSecs(SECTIONS.slice(0, 3)); setCad("Weekly \u00b7 mon 09:00"); setCron(""); setWizOpen(true); };
  const cadLabel = cad === "Custom\u2026" ? cron || "custom" : cad.toLowerCase();
  const create = () => {
    setRows(rows.concat({ name: name.trim(), cadence: cadLabel, format: "pdf", last: "\u2014" }));
    setWizOpen(false);
  };
  return (
    <main data-screen-label="Reports" style={{ maxWidth: 1440, margin: "0 auto", padding: 32, display: "flex", flexDirection: "column", gap: 24 }}>
      <header style={{ display: "flex", alignItems: "center", gap: 16 }}>
        <div style={{ display: "flex", flexDirection: "column", gap: 2 }}>
          <h1 style={{ margin: 0, font: "600 21px var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)" }}>Reports</h1>
          <span style={{ font: "400 12.5px var(--font-ui)", color: "var(--text-muted)", whiteSpace: "nowrap" }}>Trends and scheduled exports for the selected period.</span>
        </div>
        <div style={{ marginLeft: "auto", display: "flex", gap: 8, alignItems: "center" }}>
          <DateRangePicker value={range} onChange={setRange} />
          <SplitButton icon={<Icon name="download" size={14} />} items={[{ label: "Export JSON", icon: "braces" }, { label: "Export PDF", icon: "file-text" }]}>Export CSV</SplitButton>
        </div>
      </header>
      <div style={{ display: "grid", gridTemplateColumns: "repeat(3, 1fr)", gap: 24 }}>
        <ReportCard title="Open signals" period={range.label} value="47" delta="+3" deltaTone="bad" caption="vs previous period">
          <Sparkline data={OPEN_TREND} width={340} height={44} style={{ width: "100%" }} />
        </ReportCard>
        <ReportCard title="New assets discovered" period={range.label} value="12" delta="+8" deltaTone="neutral" caption="8 domains · 4 IPs">
          <BarChart data={DISCOVERY} labels={DAY_LABELS} height={44} />
        </ReportCard>
        <ReportCard title="Mean time to resolve" period={range.label} value="2.4d" delta="−0.6d" deltaTone="good" caption="critical + high only">
          <Sparkline data={RESOLVE_TREND} width={340} height={44} color="var(--chart-2)" style={{ width: "100%" }} />
        </ReportCard>
      </div>
      <Card microLabel="Trend" title="Open signals over time">
        <TimeSeriesChart height={230} label="Open signals over time"
          series={[{ label: "All open", data: OPEN_TREND }, { label: "Critical + high", data: CRIT_HIGH, color: "var(--chart-2)" }]}
          labels={DAYS_SPARSE} hoverLabels={DAYS_FULL} />
      </Card>
      <div style={{ display: "grid", gridTemplateColumns: "380px 1fr", gap: 24, alignItems: "start" }}>
        <div style={{ display: "flex", flexDirection: "column", gap: 24 }}>
          <Card microLabel="Open signals" title="By severity">
            <SevBars />
          </Card>
          <Card microLabel="Activity" title="Scans per day">
            <HeatmapCalendar values={SCAN_DAYS} startLabel="12 weeks ago" label="Scans per day, last 12 weeks" />
          </Card>
        </div>
        <Card microLabel="Scheduled" title="Recurring reports" pad={0} overflow="visible"
          action={<Button variant="ghost" size="sm" icon={<Icon name="plus" size={13} />} onClick={openWiz}>New schedule</Button>}>
          <Table framed={false} columns={[
            { key: "name", label: "Report", sortable: true, render: (r) => <span style={{ font: "500 13px var(--font-ui)", color: "var(--text-ink)" }}>{r.name}</span> },
            { key: "cadence", label: "Cadence", mono: true, width: 170 },
            { key: "format", label: "Format", width: 90, render: (r) => <Tag>{r.format}</Tag> },
            { key: "last", label: "Last sent", mono: true, align: "right", width: 90, sortable: true, sortValue: (r) => t2m(r.last) },
            { key: "actions", label: "", width: 58, align: "right", clip: false, render: () => (
              <DropdownMenu trigger={<IconButton icon="ellipsis" label="Actions" size="sm" />} items={[
                { label: "View last delivery", icon: "eye", onSelect: onOpenArtifact },
                { label: "Run now", icon: "play" },
                { label: "Edit schedule", icon: "pencil" },
                "-",
                { label: "Delete schedule", icon: "trash-2", tone: "danger" },
              ]} />
            ) },
          ]} rows={rows} rowKey="name" />
        </Card>
      </div>
      <Wizard open={wizOpen} title="New report schedule" description="A recurring export, delivered on cadence." onClose={() => setWizOpen(false)} onFinish={create} finishLabel="Create schedule"
        steps={[
          { id: "scope", title: "Scope", valid: name.trim().length > 0 && secs.length > 0, content: (
            <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
              <Input label="Report name" placeholder="Weekly exposure summary" value={name} onChange={(e) => setName(e.target.value)} />
              <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
                <span style={{ font: "500 11px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--text-muted)" }}>Sections</span>
                {SECTIONS.map((s) => (
                  <Checkbox key={s} label={s} checked={secs.indexOf(s) >= 0} onChange={(c) => setSecs(c ? secs.concat(s) : secs.filter((x) => x !== s))} />
                ))}
              </div>
            </div>
          ) },
          { id: "cadence", title: "Cadence", valid: cad !== "Custom\u2026" || cron.trim().length > 0, content: (
            <CadenceSelect value={cad} customValue={cron} onChange={(v, c) => { setCad(v); setCron(c || ""); }} />
          ) },
          { id: "review", title: "Review", content: (
            <KeyValueList items={[
              { k: "Report", v: name.trim() || "\u2014" },
              { k: "Sections", v: secs.join(", ") },
              { k: "Cadence", v: cadLabel },
              { k: "Format", v: "pdf" },
            ]} />
          ) },
        ]} />
    </main>
  );
}
