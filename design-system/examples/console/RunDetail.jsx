import React from "react";
import { Breadcrumb } from "../../components/navigation/Breadcrumb.jsx";
import { Card } from "../../components/display/Card.jsx";
import { BatchStatus } from "../../components/display/BatchStatus.jsx";
import { Stepper } from "../../components/display/Stepper.jsx";
import { LogViewer } from "../../components/display/LogViewer.jsx";
import { KeyValueList } from "../../components/display/KeyValueList.jsx";
import { Stat } from "../../components/display/Stat.jsx";
import { Badge } from "../../components/display/Badge.jsx";
import { Callout } from "../../components/feedback/Callout.jsx";
import { Button } from "../../components/forms/Button.jsx";
import { Icon } from "../../components/media/Icon.jsx";

const LOG = [
  { time: "14:00:02", text: "batch started · 214 subjects · 3 vantages" },
  { time: "14:00:09", text: "dns sweep · acmecorp.io · 1,284 names" },
  { time: "14:00:41", level: "warn", text: "vantage ap-south-1 missed check (2/3)" },
  { time: "14:01:12", text: "tls-acceptance · vpn.acmecorp.io:443" },
  { time: "14:02:03", level: "error", text: "connect refused · 203.0.113.44:22" },
  { time: "14:02:31", text: "port census · 62 addresses · top 1,000 tcp" },
  { time: "14:03:14", text: "diff against 08:00Z · 7 transitions · 3 signals raised" },
];

/* One batch, end to end: stages, log, outcome. A running job streams its own exact output. */
function jobOutput(job) {
  const v = job.vantage ? " · " + job.vantage : "";
  const k = String(job.kind || "job");
  return [
    { time: "14:00:02", text: "job #" + job.id + " started · " + k + v + " · attempt " + (job.attempt || "1/3") },
    { time: "14:00:03", text: "$ verge run " + k + " --vantage " + (job.vantage || "local") },
    { time: "14:00:06", text: k + " · resolving work set" },
    { time: "14:00:12", text: k + " · 62 subjects in scope" },
    { time: "14:00:31", level: job.state === "retrying" ? "warn" : undefined, text: job.state === "retrying" ? "transient error · backing off, retrying (" + (job.attempt || "2/3") + ")" : k + " · 34% · 21/62" },
    { time: "14:01:05", text: k + " · 68% · 42/62" },
    { time: "14:01:44", text: k + " · 100% · 62/62" },
    { time: "14:01:46", text: "committing batch · observations append-only" },
    { time: "14:01:47", text: "job #" + job.id + " complete" },
  ];
}
export function RunDetail({ jobFilter, onClearJob,  id = "2026-08-22T14:00Z", onBack, onOpenDrift }) {
  const running = !!(jobFilter && (jobFilter.state === "running" || jobFilter.state === "retrying"));
  const full = jobFilter ? jobOutput(jobFilter) : LOG;
  const [count, setCount] = React.useState(running ? 5 : full.length);
  React.useEffect(() => {
    setCount(running ? 5 : full.length);
    if (!running) return;
    const t = setInterval(() => setCount((c) => (c >= full.length ? c : c + 1)), 950);
    return () => clearInterval(t);
  }, [jobFilter, running]);
  const jfLines = jobFilter ? full.slice(0, count) : LOG;
  const live = running && count < full.length;
  return (
    <main data-screen-label="Scan run detail" style={{ maxWidth: 1440, margin: "0 auto", padding: 32, display: "flex", flexDirection: "column", gap: 20 }}>
      <Breadcrumb items={[{ label: "Drift", onClick: onBack }, { label: "batch " + id }]} />
      <header style={{ display: "flex", alignItems: "flex-start", gap: 16, flexWrap: "wrap" }}>
        <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
          <h1 style={{ margin: 0, font: "600 21px var(--font-mono)", letterSpacing: "-0.01em", color: "var(--text-ink)" }}>{id}</h1>
          <div style={{ display: "flex", alignItems: "center", gap: 10, flexWrap: "wrap" }}>
            <BatchStatus status="complete" scope="all scopes" />
            <span style={{ font: "400 12px var(--font-mono)", color: "var(--text-muted)" }}>standard profile · 3 vantages · 3m 12s</span>
          </div>
        </div>
        <div style={{ marginLeft: "auto" }}>
          <Button variant="secondary" icon={<Icon name="git-branch" size={14} />} onClick={onOpenDrift}>Drift from this batch</Button>
        </div>
      </header>
      <Card microLabel="Pipeline" title="Stages">
        <Stepper active={4} steps={[
          { title: "Resolve", detail: "dns + zone + CT · 1,284 names" },
          { title: "Probe", detail: "reachability from 3 vantages" },
          { title: "Census", detail: "top 1,000 tcp · 62 addresses" },
          { title: "Diff", detail: "against 08:00Z · 7 transitions" },
        ]} />
      </Card>
      <div style={{ display: "grid", gridTemplateColumns: "minmax(0, 1fr) 340px", gap: 24, alignItems: "start" }}>
        <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
            {jobFilter && (
              <div style={{ display: "flex", alignItems: "center", gap: 8, flexWrap: "wrap" }}>
                <span style={{ display: "inline-flex", alignItems: "center", gap: 6, height: 20, padding: "0 8px", borderRadius: 999, border: "1px solid var(--border-strong)", font: "500 10px var(--font-mono)", letterSpacing: "0.05em", textTransform: "uppercase", color: "var(--text-secondary)", whiteSpace: "nowrap" }}>job #{jobFilter.id} · {jobFilter.kind}{jobFilter.vantage ? " · " + jobFilter.vantage : ""}
                  <button onClick={onClearJob} aria-label="Clear job filter" style={{ display: "inline-flex", alignItems: "center", justifyContent: "center", width: 14, height: 14, border: "none", borderRadius: 999, background: "transparent", color: "var(--text-muted)", cursor: "pointer", padding: 0 }}><svg viewBox="0 0 10 10" width="9" height="9"><path d="M2 2l6 6M8 2l-6 6" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round"></path></svg></button>
                </span>
                <span style={{ font: "400 11px var(--font-ui)", color: "var(--text-muted)" }}>{running ? "streaming this job’s exact output" : "this job’s exact output"}</span>
              </div>
            )}
            <LogViewer title={jobFilter ? "job #" + jobFilter.id + " · " + jobFilter.kind : "batch " + id} live={live} lines={jfLines} height={300} />
          </div>
        <div style={{ display: "flex", flexDirection: "column", gap: 24 }}>
          <Card microLabel="Outcome" title="What it produced">
            <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
              <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 16 }}>
                <Stat label="Transitions" value="7" />
                <Stat label="New signals" value="3" deltaTone="bad" />
              </div>
              <Callout tone="warn" title="One vantage degraded">ap-south-1 missed 2 of 3 checks — exposure conclusions from it are marked unverified for this batch.</Callout>
            </div>
          </Card>
          <Card microLabel="Parameters" title="As configured">
            <KeyValueList items={[
              { k: "Profile", v: "standard" },
              { k: "Cadence", v: "daily · 08:00 + 14:00" },
              { k: "Subjects", v: "214" },
              { k: "Address cap", v: "1,024" },
              { k: "Connect timeout", v: "800ms" },
            ]} />
          </Card>
          <Card microLabel="Vantages" title="Who looked">
            <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
              {[["eu-west-1", "ok", "34ms"], ["us-east-2", "ok", "51ms"], ["ap-south-1", "degraded", "—"]].map(([n, s, l]) => (
                <div key={n} style={{ display: "flex", alignItems: "center", gap: 10 }}>
                  <span style={{ font: "500 12.5px var(--font-mono)", color: "var(--text-ink)" }}>{n}</span>
                  <span style={{ marginLeft: "auto", display: "inline-flex", alignItems: "center", gap: 8 }}>
                    <span style={{ font: "400 11.5px var(--font-mono)", color: "var(--text-muted)" }}>{l}</span>
                    <Badge tone={s === "ok" ? "ok" : "warn"} dot>{s}</Badge>
                  </span>
                </div>
              ))}
            </div>
          </Card>
        </div>
      </div>
    </main>
  );
}
