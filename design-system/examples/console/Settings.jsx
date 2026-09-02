import React from "react";
import { Card } from "../../components/display/Card.jsx";
import { SettingsNav } from "../../components/navigation/SettingsNav.jsx";
import { Calendar } from "../../components/display/Calendar.jsx";
import { Integrations } from "./Integrations.jsx";
import { RadioCards } from "../../components/forms/RadioCards.jsx";
import { Button } from "../../components/forms/Button.jsx";
import { CadenceSelect } from "../../components/forms/CadenceSelect.jsx";
import { Slider } from "../../components/forms/Slider.jsx";
import { NumberInput } from "../../components/forms/NumberInput.jsx";
import { Combobox } from "../../components/forms/Combobox.jsx";
import { ChannelForm } from "../../components/forms/ChannelForm.jsx";
import { Badge } from "../../components/display/Badge.jsx";
import { Tag } from "../../components/display/Tag.jsx";
import { StatusDot } from "../../components/display/StatusDot.jsx";
import { BatchStatus } from "../../components/display/BatchStatus.jsx";
import { LogViewer } from "../../components/display/LogViewer.jsx";
import { Spinner } from "../../components/display/Spinner.jsx";
import { Skeleton } from "../../components/display/Skeleton.jsx";
import { Table } from "../../components/display/Table.jsx";
import { VantageCard } from "../../components/display/VantageCard.jsx";
import { ExposureBadge } from "../../components/display/ExposureBadge.jsx";
import { MessageList } from "../../components/feedback/MessageList.jsx";
import { ErrorState } from "../../components/feedback/ErrorState.jsx";
import { EmptyState } from "../../components/feedback/EmptyState.jsx";
import { Icon } from "../../components/media/Icon.jsx";
import { Switch } from "../../components/forms/Switch.jsx";
import { FileDrop } from "../../components/forms/FileDrop.jsx";
import { Callout } from "../../components/feedback/Callout.jsx";
import { CopyValue } from "../../components/display/CopyValue.jsx";
import { CertificateCard } from "../../components/display/CertificateCard.jsx";
import { MappingEditor } from "../../components/forms/MappingEditor.jsx";
import { IdPButton } from "../../components/forms/IdPButton.jsx";
import { Avatar } from "../../components/display/Avatar.jsx";
import { Progress } from "../../components/display/Progress.jsx";
import { Stat } from "../../components/display/Stat.jsx";
import { Input } from "../../components/forms/Input.jsx";
import { Select } from "../../components/forms/Select.jsx";
import { Dialog } from "../../components/feedback/Dialog.jsx";
import { DropdownMenu } from "../../components/feedback/DropdownMenu.jsx";
import { IconButton } from "../../components/forms/IconButton.jsx";
import { DateRangePicker } from "../../components/forms/DateRangePicker.jsx";
import { ConfirmDialog } from "../../components/feedback/ConfirmDialog.jsx";
import { Stepper } from "../../components/display/Stepper.jsx";
import { CodeBlock } from "../../components/display/CodeBlock.jsx";
import { SourcesSection, ApertureSection } from "./Sources.jsx";

const VANTAGE_FLEET = [["eu-west-1", "available", "34ms"], ["us-east-2", "available", "51ms"], ["ap-south-1", "unverified", "—"]];

const BASE_LOG = [
  { time: "14:00:02", text: "batch started · 214 subjects · 3 vantages" },
  { time: "14:00:09", text: "dns sweep · acmecorp.io · 1,284 names" },
  { time: "14:00:41", level: "warn", text: "vantage ap-south-1 missed check (2/3)" },
  { time: "14:01:12", text: "tls-acceptance · vpn.acmecorp.io:443" },
  { time: "14:02:03", level: "error", text: "connect refused · 203.0.113.44:22" },
];

function ScansSection({ onToast, onOpenRun }) {
  const CATALOG = [
    { kind: "dns", cadence: "daily", enabled: true },
    { kind: "hot", cadence: "daily", enabled: true },
    { kind: "tls-acceptance", cadence: "every 7 days", enabled: true },
    { kind: "cold", cadence: "every 30 days", enabled: false },
    { kind: "zone", cadence: "every 30 days", enabled: true },
    { kind: "ct", cadence: "daily", enabled: true },
  ];
  const JOBS = [
    { id: 912, kind: "dns-sweep", vantage: "eu-west-1", state: "done", attempt: "1/3", batch: "1407" },
    { id: 913, kind: "reachability", vantage: "eu-west-1", state: "done", attempt: "1/3", batch: "1407" },
    { id: 914, kind: "reachability", vantage: "us-east-2", state: "done", attempt: "1/3", batch: "1407" },
    { id: 915, kind: "reachability", vantage: "ap-south-1", state: "retrying", attempt: "2/3", batch: null },
    { id: 916, kind: "port-census", vantage: "eu-west-1", state: "running", attempt: "1/3", batch: null },
    { id: 917, kind: "tls-acceptance", vantage: "eu-west-1", state: "done", attempt: "1/3", batch: "1407" },
  ];
  const [dispatch, setDispatch] = React.useState({ kind: "standard", at: "2026-08-22T14:00:02Z", completed: 4, live: 6, percent: 67, jobs: JOBS });
  const [history, setHistory] = React.useState([
    { kind: "standard", at: "2026-08-22T08:00:14Z", live: 6, completed: 6, dead: 0 },
    { kind: "ct", at: "2026-08-22T06:00:03Z", live: 1, completed: 1, dead: 0 },
    { kind: "standard", at: "2026-08-21T20:00:09Z", live: 6, completed: 5, dead: 1 },
  ]);
  const [confirm, setConfirm] = React.useState(null);
  const pend = dispatch ? dispatch.jobs.filter((j) => j.state === "retrying" || j.state === "ready").length : 0;
  const runn = dispatch ? dispatch.jobs.filter((j) => j.state === "running").length : 0;
  const conclude = (mode) => {
    setConfirm(null);
    if (!dispatch) return;
    const doneNow = dispatch.completed + (mode === "stop" ? runn : 0);
    setHistory((h) => [{ kind: dispatch.kind, note: mode === "stop" ? "stopped · partial" : "terminated", at: dispatch.at, live: dispatch.live, completed: doneNow, dead: dispatch.live - doneNow }].concat(h));
    setDispatch(null);
    onToast && onToast(mode === "stop"
      ? { tone: "neutral", title: "Dispatch stopped", description: pend + " pending job" + (pend === 1 ? "" : "s") + " cancelled · " + runn + " running finishing." }
      : { tone: "danger", title: "Scan terminated", description: runn + " job" + (runn === 1 ? "" : "s") + " stopped · committed batches stand." });
  };
  const doneN = dispatch ? dispatch.jobs.filter((j) => j.state === "done").length : 0;
  const deadN = dispatch ? dispatch.jobs.filter((j) => j.state === "dead").length : 0;
  // The Running-now card summarises the fan-out as one chip per job state, count in
  // mono, instead of a per-job table that grows without bound. Full per-job detail
  // lives on run detail, one click away through the drill button.
  const rollupChip = (tone, n, label) => <Badge tone={tone} dot><span style={{ fontFamily: "var(--font-mono)" }}>{n}</span>{label}</Badge>;
  const runLink = (label) => <button onClick={onOpenRun} style={{ background: "none", border: "none", padding: 0, cursor: "pointer", font: "600 12.5px var(--font-mono)", color: "var(--link)" }}>{label}</button>;
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 24 }}>
      <p style={{ margin: 0, font: "400 13px/1.6 var(--font-ui)", color: "var(--text-secondary)", maxWidth: "72ch" }}>What the queue is doing right now. A scan runs as a fan-out of jobs — one per vantage, or per supplied zone file — and each job commits its own batch of observations. This is a read of the queue alone: it records what the system did, never what is true of your estate. Scans run on their own cadence; an admin may also dispatch an enabled one on demand.</p>
      <Card microLabel="Admin · on-demand" title="Trigger a scan">
        <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
          <p style={{ margin: 0, font: "400 12.5px/1.6 var(--font-ui)", color: "var(--text-secondary)", maxWidth: "72ch" }}>Dispatch an enabled scan now, without waiting for its cadence. A scan already in flight is not dispatched again, and the disabled cold tier cannot be triggered at all.</p>
          <Table columns={[
            { key: "kind", label: "Scan", mono: true },
            { key: "cadence", label: "Cadence", mono: true, width: 160 },
            { key: "enabled", label: "State", width: 130, render: (r) => r.enabled ? <Badge tone="ok">enabled</Badge> : <Badge tone="neutral">disabled</Badge> },
            { key: "act", label: "", align: "right", width: 260, render: (r) => r.enabled ? <Button size="sm" onClick={() => onToast && onToast({ tone: "ok", title: r.kind + " scan dispatched", description: "3 jobs fanned out · it appears under Running now." })}>Run now</Button> : <span style={{ font: "400 12px var(--font-ui)", color: "var(--text-muted)" }}>disabled — opt a scope in on Scope</span> },
          ]} rows={CATALOG} />
        </div>
      </Card>
      <Card microLabel="In flight" title="Running now">
        {dispatch ? (
          <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
            <div style={{ display: "flex", alignItems: "center", gap: 10, flexWrap: "wrap" }}>
              <span style={{ width: 8, height: 8, borderRadius: 999, background: "var(--accent)", flex: "none" }} />
              {runLink(dispatch.kind)}
              <span style={{ font: "400 11.5px var(--font-mono)", color: "var(--text-muted)" }}>dispatched {dispatch.at}</span>
              <span style={{ marginLeft: "auto", font: "400 11.5px var(--font-mono)", color: "var(--text-secondary)" }}>{dispatch.completed} / {dispatch.live} jobs · {dispatch.percent}%</span>
              <Button size="sm" variant="ghost" onClick={() => setConfirm("stop")}>Stop dispatch</Button>
              <Button size="sm" variant="ghost" style={{ color: "var(--danger)" }} onClick={() => setConfirm("terminate")}>Terminate</Button>
            </div>
            <Progress label="Fan-out" detail={dispatch.completed + "/" + dispatch.live + " jobs"} value={dispatch.percent} />
            <div style={{ display: "flex", alignItems: "center", gap: 8, flexWrap: "wrap" }}>
              {rollupChip("accent", runn, "running")}
              {pend > 0 && rollupChip("neutral", pend, "ready")}
              {rollupChip("ok", doneN, "done")}
              {deadN > 0 && rollupChip("danger", deadN, "dead")}
              <Button size="sm" variant="ghost" style={{ marginLeft: "auto" }} onClick={onOpenRun}>View all {dispatch.jobs.length} jobs</Button>
            </div>
          </div>
        ) : (
          <EmptyState message="No scan running" detail="Nothing is dispatched right now. When a scan's cadence comes due the worker fans it out, and it appears here with its progress while it runs. This view refreshes on its own while a scan is in flight." />
        )}
      </Card>
      <Card microLabel="Batches" title="Recent dispatches">
        <Table columns={[
          { key: "dot", label: "", width: 40, render: (r) => <span style={{ display: "inline-block", width: 8, height: 8, borderRadius: 999, background: r.dead > 0 ? "var(--danger-solid)" : "var(--ok-solid)" }} /> },
          { key: "kind", label: "Scan", render: (r) => <span style={{ display: "inline-flex", alignItems: "center", gap: 8 }}>{runLink(r.kind)}{r.note && <Badge tone={r.note === "terminated" ? "danger" : "warn"}>{r.note}</Badge>}</span> },
          { key: "at", label: "Dispatched", mono: true, width: 180 },
          { key: "live", label: "Jobs", mono: true, width: 70 },
          { key: "completed", label: "Completed", mono: true, width: 100 },
          { key: "dead", label: "Dead", mono: true, align: "right", width: 70, render: (r) => r.dead > 0 ? <span style={{ color: "var(--danger)" }}>{r.dead}</span> : <span style={{ color: "var(--text-muted)" }}>0</span> },
        ]} rows={history} />
      </Card>
      <Dialog open={confirm === "stop"} title={"Stop dispatching " + (dispatch ? dispatch.kind : "")} onClose={() => setConfirm(null)}
        footer={<><Button variant="ghost" onClick={() => setConfirm(null)}>Cancel</Button><Button onClick={() => conclude("stop")}>Stop dispatch</Button></>}>
        <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
          <p style={{ margin: 0, font: "400 13px/1.55 var(--font-ui)", color: "var(--text-body)" }}>{pend} pending job{pend === 1 ? " is" : "s are"} cancelled. {runn} running job{runn === 1 ? " finishes" : "s finish"} and commit{runn === 1 ? "s its batch" : " their batches"}; nothing already observed is discarded.</p>
          <p style={{ margin: 0, font: "400 13px/1.55 var(--font-ui)", color: "var(--text-secondary)" }}>The dispatch is recorded as stopped · partial. The scan runs again on its normal cadence.</p>
        </div>
      </Dialog>
      <ConfirmDialog open={confirm === "terminate"} title={"Terminate " + (dispatch ? dispatch.kind : "") + " now"}
        message={runn + " running job" + (runn === 1 ? " is" : "s are") + " stopped where " + (runn === 1 ? "it stands" : "they stand") + " and uncommitted work discards. Batches already committed stand — observations are append-only. Anything half-measured is simply absent until the next run."}
        confirmLabel="Terminate" onConfirm={() => conclude("terminate")} onClose={() => setConfirm(null)} />
    </div>
  );
}
function ProberProvision({ onToast }) {
  const [step, setStep] = React.useState(0);
  const [host, setHost] = React.useState("");
  const [port, setPort] = React.useState("22");
  const [user, setUser] = React.useState("");
  const PUB = 'restrict,from="203.0.113.5" ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIF3kQm7Vr2xw9tPba41uD8cNzUj0eYhLxOqRkM9 verge-prober';
  const toast = (title, description, tone) => onToast && onToast({ tone: tone || "neutral", title, description });
  return (
    <Card microLabel="Probers" title="Provision an internet vantage">
      <div style={{ display: "flex", flexDirection: "column", gap: 18 }}>
        <Stepper active={step} steps={[
          { title: "Describe the host", detail: "four non-secret values" },
          { title: "Install the key", detail: "public half only" },
          { title: "Pin and verify", detail: "host key, platform, egress" },
        ]} />
        {step === 0 && (
          <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
            <div style={{ display: "grid", gridTemplateColumns: "2fr 90px 1.4fr", gap: 12 }}>
              <Input label="Host" mono placeholder="probe-1.example.net" value={host} onChange={(e) => setHost(e.target.value)} spellCheck={false} />
              <Input label="Port" mono value={port} onChange={(e) => setPort(e.target.value)} />
              <Input label="Username" mono placeholder="verge" value={user} onChange={(e) => setUser(e.target.value)} hint="A non-root account" spellCheck={false} />
            </div>
            <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
              <Button disabled={!host.trim() || !user.trim()} onClick={() => { setStep(1); toast("Keypair generated", "The private half never leaves the instance."); }}>Generate keypair</Button>
              <span style={{ font: "400 11.5px var(--font-ui)", color: "var(--text-muted)" }}>Don't hand-roll the host — deploy/prober/ is the hardened compose recipe.</span>
            </div>
          </div>
        )}
        {step === 1 && (
          <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
            <span style={{ font: "400 12.5px/1.6 var(--font-ui)", color: "var(--text-body)" }}>Install the rendered <strong>public</strong> key in <span style={{ fontFamily: "var(--font-mono)", fontSize: 12 }}>{user || "verge"}@{host || "the host"}</span>'s authorized_keys — the restrict and from= options are already in place.</span>
            <CodeBlock title="authorized_keys" copyText={PUB}>{PUB}</CodeBlock>
            <div style={{ display: "flex", gap: 8 }}>
              <Button variant="ghost" onClick={() => setStep(0)}>Back</Button>
              <Button onClick={() => setStep(2)}>I installed it — connect</Button>
            </div>
          </div>
        )}
        {step === 2 && (
          <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
            <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
              <div style={{ display: "flex", alignItems: "center", gap: 10, flexWrap: "wrap" }}>
                <StatusDot status="ok" label="host key pinned" />
                <span style={{ font: "400 11.5px var(--font-mono)", color: "var(--text-muted)" }}>SHA256:9m:41:be:0a:77:c3:e8:12:f4:5d:a0:66:2b:91:c7:03</span>
              </div>
              <StatusDot status="ok" label="linux · x86_64 — accepted" />
            </div>
            <Callout tone="accent" title="Your egress is 203.0.113.5">Observed from SSH_CLIENT on first connection. Declare it so the estate knows its own outbound address.
              <div style={{ marginTop: 10 }}><Button size="sm" variant="secondary" onClick={() => toast("Egress declared", "203.0.113.5 · added as an address scope", "ok")}>Declare egress as seed</Button></div>
            </Callout>
            <span style={{ font: "400 11.5px/1.6 var(--font-ui)", color: "var(--text-muted)" }}>A later host-key change is a hard failure, never a prompt. A host that is not Linux on x86_64/aarch64 is refused, with the reason.</span>
            <div><Button variant="ghost" size="sm" onClick={() => { setStep(0); setHost(""); setUser(""); }}>Provision another</Button></div>
          </div>
        )}
      </div>
    </Card>
  );
}

function VantagesSection({ onToast }) {
  const [vant, setVant] = React.useState("eu-west-1");
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 24 }}>
      <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(300px, 1fr))", gap: 16 }}>
        <VantageCard name="eu-west-1" vantageClass="internet" resolver="9.9.9.9" availability="available" latency="34ms" />
        <VantageCard name="us-east-2" vantageClass="internet" resolver="9.9.9.9" availability="available" latency="51ms" />
        <VantageCard name="ap-south-1" vantageClass="unverified" resolver="9.9.9.9" availability="unverified" />
      </div>
      <ProberProvision onToast={onToast} />
      <Card microLabel="Defaults" title="Preferred vantage" overflow="visible">
        <Combobox label="Default vantage" value={vant} onChange={setVant} placeholder="Search vantages" options={[
          { value: "eu-west-1", label: "eu-west-1", hint: "internet" },
          { value: "us-east-2", label: "us-east-2", hint: "internet" },
          { value: "dc-fra-01", label: "dc-fra-01", hint: "internal" },
        ]} />
      </Card>
      <Card microLabel="Reach" title="What a vantage can conclude">
        <div style={{ display: "flex", alignItems: "center", gap: 10, flexWrap: "wrap" }}>
          <ExposureBadge state="exposed" /><ExposureBadge state="firewalled" /><ExposureBadge state="not-reached" /><ExposureBadge state="unverified" />
          <span style={{ font: "400 11.5px var(--font-ui)", color: "var(--text-muted)" }}>states, never a score — concluded across two reach legs</span>
        </div>
      </Card>
    </div>
  );
}

function ChannelsSection({ onToast }) {
  const rows = [
    { url: "https://ops.acmecorp.io/hook", classes: ["signals", "drift"], status: "active" },
    { url: "https://pager.example/verge", classes: ["signals"], status: "paused" },
  ];
  const sendTest = (c) => onToast && onToast({ tone: "ok", title: "Test message sent", description: c.url.replace(/^https?:\/\//, "") + " \u00b7 check the channel for the delivery." });
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 24 }}>
      <Card microLabel="Channels" title="One-way delivery">
        <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
          {rows.map((c) => (
            <div key={c.url} style={{ display: "flex", alignItems: "center", gap: 10, padding: "9px 12px", background: "var(--surface-sunken)", borderRadius: 10, flexWrap: "wrap" }}>
              <span style={{ font: "500 12px var(--font-mono)", color: "var(--text-body)" }}>{c.url}</span>
              {c.classes.map((k) => <Tag key={k}>{k}</Tag>)}
              <span style={{ marginLeft: "auto", display: "inline-flex", alignItems: "center", gap: 10 }}>
                <Badge tone={c.status === "active" ? "ok" : "neutral"} dot>{c.status}</Badge>
                <Button size="sm" variant="secondary" onClick={() => sendTest(c)}>Send test</Button>
              </span>
            </div>
          ))}
        </div>
      </Card>
      <Card microLabel="New channel" title="Add delivery">
        <ChannelForm onSubmit={() => onToast && onToast({ tone: "ok", title: "Channel saved", description: "Deliveries start with the next message." })} />
      </Card>
    </div>
  );
}

function MessagesSection() {
  return (
    <Card microLabel="Inbox" title="Messages" action={<Badge tone="accent">2 unread</Badge>} pad={12}>
      <MessageList onOpen={() => {}} messages={[
        { id: "m1", cls: "signals", text: "VNC exposed to internet · edge-gw-03.acmecorp.io", time: "4m", unread: true },
        { id: "m2", cls: "drift", text: "appeared · staging-5.acmecorp.io", time: "8m", unread: true },
        { id: "m3", cls: "coverage", text: "zone transfer silent for 9d · internal.acmecorp.io", time: "6h" },
        { id: "m4", cls: "batches", text: "batch complete · 3 new signals", time: "6h" },
      ]} />
    </Card>
  );
}

function DeliverySection() {
  const [state, setState] = React.useState("error");
  const retry = () => { setState("loading"); setTimeout(() => setState("ok"), 1200); };
  return (
    <Card microLabel="Operational record" title="Deliveries" pad={state === "ok" ? 0 : 20} overflow="visible">
      {state === "error" && <ErrorState style={{ padding: "24px 20px" }} message="Delivery record unavailable." detail="The last fetch timed out. Retry to reload the record." onRetry={retry} />}
      {state === "loading" && <div style={{ display: "flex", flexDirection: "column", gap: 12, padding: "8px 4px" }}><Skeleton lines={4} /><Skeleton shape="block" height={60} /></div>}
      {state === "ok" && (
        <Table framed={false} dense columns={[
          { key: "channel", label: "Channel", mono: true },
          { key: "cls", label: "Class", width: 110, render: (r) => <Tag>{r.cls}</Tag> },
          { key: "outcome", label: "Outcome", width: 120, render: (r) => <Badge tone={r.outcome === "delivered" ? "ok" : "danger"} dot>{r.outcome}</Badge> },
          { key: "time", label: "When", mono: true, align: "right", width: 70 },
        ]} rows={[
          { channel: "ops.acmecorp.io/hook", cls: "signals", outcome: "delivered", time: "4m" },
          { channel: "ops.acmecorp.io/hook", cls: "drift", outcome: "delivered", time: "8m" },
          { channel: "pager.example/verge", cls: "signals", outcome: "failed", time: "6h" },
        ]} rowKey="time" />
      )}
    </Card>
  );
}

function SsoSection({ onToast }) {
  const [maps, setMaps] = React.useState([
    { from: "email", to: "Email" },
    { from: "displayName", to: "Display name" },
    { from: "groups", to: "Org role" },
  ]);
  const [enforce, setEnforce] = React.useState(false);
  const micro = { font: "500 11px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--text-muted)" };
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 24 }}>
      <Card microLabel="SAML 2.0" title="Identity provider" action={<Badge tone="ok" dot>connected</Badge>}>
        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 24, alignItems: "start" }}>
          <FileDrop compact accept=".xml" label="Replace IdP metadata XML" hint="Entity descriptor from your IdP"
            onFiles={() => onToast && onToast({ tone: "ok", title: "Metadata updated", description: "Endpoints and signing certificate re-read." })} />
          <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
            <span style={micro}>Give these to the IdP</span>
            <CopyValue value="https://verge.acmecorp.io/sso/metadata" />
            <CopyValue value="https://verge.acmecorp.io/sso/acs" />
            <span style={{ font: "400 11.5px var(--font-ui)", color: "var(--text-muted)" }}>Entity ID · ACS URL</span>
          </div>
        </div>
      </Card>
      <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(320px, 1fr))", gap: 16 }}>
        <CertificateCard name="idp-signing-2026" issuer="CN=Okta, O=acmecorp" algorithm="RSA-SHA256" notAfter="2026-09-14" daysLeft={23}
          fingerprint="SHA256:7f:2a:91:c4:0e:88:b1:d3:5a:66:04:9f:e2:71:38:ac"
          onReplace={() => onToast && onToast({ tone: "neutral", title: "Awaiting new certificate", description: "Upload fresh IdP metadata to rotate." })} />
        <CertificateCard name="sp-signing-2027" role="SP signing" issuer="CN=verge.acmecorp.io (self-signed)" algorithm="RSA-SHA256" notAfter="2027-06-02" daysLeft={284}
          fingerprint="SHA256:d1:0b:44:ee:23:7c:91:af:08:5d:c6:12:9b:e0:47:31" />
      </div>
      <Card microLabel="Attributes" title="Claim mapping" overflow="visible">
        <MappingEditor mappings={maps} onChange={setMaps} toOptions={["Email", "Display name", "Org role", "Ignore"]} />
      </Card>
      <Card microLabel="Enforcement" title="Require SSO">
        <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
          <div style={{ display: "flex", alignItems: "flex-start", gap: 12 }}>
            <Switch checked={enforce} onChange={setEnforce} aria-label="Require SSO for all sign-ins" />
            <div style={{ display: "flex", flexDirection: "column", gap: 2 }}>
              <span style={{ font: "500 13px var(--font-ui)", color: "var(--text-ink)" }}>Require SSO for all sign-ins</span>
              <span style={{ font: "400 12px/1.55 var(--font-ui)", color: "var(--text-muted)" }}>Local passwords stop working for everyone except break-glass operators.</span>
            </div>
          </div>
          {enforce && <Callout tone="warn" title="Break-glass stays local">Operator accounts created with <code style={{ font: "400 0.92em var(--font-mono)", background: "var(--surface-sunken)", border: "1px solid var(--border-default)", borderRadius: 6, padding: "1px 5px" }}>verge users add --break-glass</code> keep password + TOTP sign-in even with SSO required.</Callout>}
          <div style={{ display: "flex", flexDirection: "column", gap: 8, maxWidth: 340 }}>
            <span style={micro}>Sign-in preview</span>
            <IdPButton provider="okta" onClick={() => onToast && onToast({ tone: "neutral", title: "Test sign-in started", description: "Redirects to the IdP in a real deployment." })} />
          </div>
        </div>
      </Card>
    </div>
  );
}

function TeamSection({ onToast }) {
  const [inviteOpen, setInviteOpen] = React.useState(false);
  const [invEmail, setInvEmail] = React.useState("");
  const toast = (title, description, tone) => onToast && onToast({ tone: tone || "neutral", title, description });
  const [members, setMembers] = React.useState([
    { name: "Ola Pérez", email: "ola@acmecorp.io", role: "admin", mfa: true, last: "now", you: true },
    { name: "Dana K.", email: "dana@acmecorp.io", role: "admin", mfa: true, last: "2h" },
    { name: "Sam Reyes", email: "sam@acmecorp.io", role: "viewer", mfa: false, last: "3d" },
    { name: "Priya N.", email: "priya@acmecorp.io", role: "viewer", mfa: true, last: "12d" },
  ]);
  const [action, setAction] = React.useState(null); // { type: "role" | "reenroll" | "remove", m }
  const [roleSel, setRoleSel] = React.useState("viewer");
  const closeAction = () => setAction(null);
  const saveRole = () => {
    setMembers((ms) => ms.map((x) => (x.email === action.m.email ? { ...x, role: roleSel } : x)));
    toast("Role changed", action.m.name + " · now " + roleSel, "ok");
    closeAction();
  };
  const reenroll = () => {
    setMembers((ms) => ms.map((x) => (x.email === action.m.email ? { ...x, mfa: false } : x)));
    toast("Two-factor reset", action.m.name + " enrolls again at next sign-in");
    closeAction();
  };
  const removeMember = () => {
    setMembers((ms) => ms.filter((x) => x.email !== action.m.email));
    toast("Member removed", action.m.email);
    closeAction();
  };
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 24 }}>
      <Card microLabel="Members" title="Who can sign in" pad={0}
        action={<Button size="sm" icon={<Icon name="user-plus" size={13} />} onClick={() => setInviteOpen(true)}>Invite</Button>}>
        <Table framed={false} columns={[
          { key: "name", label: "Member", render: (r) => (
            <span style={{ display: "inline-flex", alignItems: "center", gap: 10 }}>
              <Avatar name={r.name} size={26} />
              <span style={{ display: "flex", flexDirection: "column", gap: 1 }}>
                <span style={{ font: "500 12.5px var(--font-ui)", color: "var(--text-ink)" }}>{r.name}{r.you ? " (you)" : ""}</span>
                <span style={{ font: "400 11.5px var(--font-mono)", color: "var(--text-muted)" }}>{r.email}</span>
              </span>
            </span>
          ) },
          { key: "role", label: "Role", width: 110, render: (r) => <Tag>{r.role}</Tag> },
          { key: "mfa", label: "Two-factor", width: 152, render: (r) => <Badge tone={r.mfa ? "ok" : "warn"} dot>{r.mfa ? "enrolled" : "not enrolled"}</Badge> },
          { key: "last", label: "Last active", mono: true, align: "right", width: 100 },
          { key: "act", label: "", width: 52, align: "right", clip: false, render: (r) => r.you ? null : (
            <DropdownMenu align="end" trigger={<IconButton icon="ellipsis" label="Member actions" size="sm" />} items={[
              { label: "Change role", icon: "shield", onSelect: () => { setRoleSel(r.role); setAction({ type: "role", m: r }); } },
              { label: "Require re-enrollment", icon: "key-round", onSelect: () => setAction({ type: "reenroll", m: r }) },
              "-",
              { label: "Remove member", icon: "user-minus", tone: "danger", onSelect: () => setAction({ type: "remove", m: r }) },
            ]} />
          ) },
        ]} rows={members} rowKey="email" />
      </Card>
      <Card microLabel="Roles" title="What each role can do">
        <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
          {[["admin", "performs declared acts — seeds, scans, channels, annotations, team, instance"], ["viewer", "reads everything, changes nothing — including the sources catalogue"]].map(([r, d]) => (
            <div key={r} style={{ display: "flex", alignItems: "baseline", gap: 12 }}>
              <span style={{ width: 76, flex: "none" }}><Tag>{r}</Tag></span>
              <span style={{ font: "400 12.5px/1.6 var(--font-ui)", color: "var(--text-body)" }}>{d}</span>
            </div>
          ))}
        </div>
      </Card>
      <Dialog open={!!action && action.type === "role"} title="Change role"
        description={action && action.type === "role" ? action.m.name + " \u00b7 " + action.m.email : ""} onClose={closeAction}
        footer={<React.Fragment>
          <Button variant="ghost" onClick={closeAction}>Cancel</Button>
          <Button disabled={!!action && action.type === "role" && roleSel === action.m.role} onClick={saveRole}>Save role</Button>
        </React.Fragment>}>
        <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
          <Select label="Role" value={roleSel} onChange={(e) => setRoleSel(e.target.value)} options={[
            { value: "admin", label: "admin", hint: "declared acts — seeds, scans, channels" },
            { value: "viewer", label: "viewer", hint: "read-only" },
          ]} />
          {roleSel === "admin" && <Callout tone="warn" title="Full control">Admins can change team, SSO, and instance settings.</Callout>}
        </div>
      </Dialog>
      <Dialog open={!!action && action.type === "reenroll"} title="Require re-enrollment"
        description={action && action.type === "reenroll" ? action.m.name + " \u00b7 " + action.m.email : ""} onClose={closeAction}
        footer={<React.Fragment>
          <Button variant="ghost" onClick={closeAction}>Cancel</Button>
          <Button variant="secondary" onClick={reenroll}>Require re-enrollment</Button>
        </React.Fragment>}>
        <span style={{ font: "400 13px/1.6 var(--font-ui)", color: "var(--text-body)", display: "block", maxWidth: 380 }}>
          Their current authenticator stops working immediately; the next sign-in walks them through two-factor setup again. Active sessions stay signed in.
        </span>
      </Dialog>
      <ConfirmDialog open={!!action && action.type === "remove"} title="Remove member"
        message={action && action.type === "remove" ? action.m.name + " loses access to this deployment." : ""}
        detail="Their annotations and audit history stay attributed. Personal API tokens are revoked."
        confirmLabel="Remove member" onConfirm={removeMember} onClose={closeAction} />
      <Dialog open={inviteOpen} title="Invite a member" description="They get a join link; the role applies on acceptance." onClose={() => setInviteOpen(false)}
        footer={<React.Fragment>
          <Button variant="ghost" onClick={() => setInviteOpen(false)}>Cancel</Button>
          <Button disabled={!/.+@.+\..+/.test(invEmail.trim())} onClick={() => { setInviteOpen(false); toast("Invite sent", invEmail.trim() + " · expires in 7 days", "ok"); setInvEmail(""); }}>Send invite</Button>
        </React.Fragment>}>
        <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
          <Input label="Email" mono placeholder="new-operator@acmecorp.io" value={invEmail} onChange={(e) => setInvEmail(e.target.value)} autoFocus spellCheck={false} />
          <Select label="Role" options={["Viewer", "Admin"]} />
        </div>
      </Dialog>
    </div>
  );
}

function SessionsSection({ onToast }) {
  const [rows, setRows] = React.useState([
    { id: "s1", account: "ola@acmecorp.io", role: "admin", device: "Firefox \u00b7 macOS", ip: "198.51.100.7", last: "now", current: true },
    { id: "s2", account: "dana@acmecorp.io", role: "admin", device: "Safari \u00b7 iOS", ip: "198.51.100.12", last: "2h" },
    { id: "s3", account: "dana@acmecorp.io", role: "admin", device: "Chrome \u00b7 Windows", ip: "203.0.113.80", last: "1d" },
    { id: "s4", account: "priya@acmecorp.io", role: "viewer", device: "Chrome \u00b7 macOS", ip: "198.51.100.31", last: "3d" },
  ]);
  const [revoke, setRevoke] = React.useState(null);
  const [revokeAll, setRevokeAll] = React.useState(null);
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 24 }}>
      <Card microLabel={"Access \u00b7 sessions"} title="Active sessions" pad={0}>
        <div style={{ padding: "0 20px 4px" }}>
          <span style={{ font: "400 12.5px/1.6 var(--font-ui)", color: "var(--text-secondary)", display: "block", maxWidth: "72ch" }}>
            Every account's live sessions across this deployment, newest activity first. Revoking one signs that browser out at once — its next request lands on the sign-in screen. Your own sessions live on your profile.
          </span>
        </div>
        <Table framed={false} dense columns={[
          { key: "account", label: "Account", mono: true, render: (r) => <span>{r.account}{r.current ? <span style={{ color: "var(--text-muted)", marginLeft: 6 }}>(you)</span> : null}</span> },
          { key: "role", label: "Role", width: 96, clip: false, render: (r) => <Badge>{r.role}</Badge> },
          { key: "device", label: "Device", width: 170 },
          { key: "ip", label: "IP", mono: true, width: 140 },
          { key: "last", label: "Last active", mono: true, width: 100 },
          { key: "act", label: "", width: 84, align: "right", clip: false, render: (r) => r.current ? null : (
            <span style={{ display: "inline-flex", gap: 4 }}>
              <IconButton icon="log-out" label="Revoke session" size="sm" onClick={() => setRevoke(r)} />
              <DropdownMenu align="end" trigger={<IconButton icon="ellipsis" label="More" size="sm" />} items={[
                { label: "Revoke all for " + r.account.split("@")[0], icon: "circle-off", tone: "danger", onSelect: () => setRevokeAll(r) },
              ]} />
            </span>
          ) },
        ]} rows={rows} rowKey="id" />
      </Card>
      <ConfirmDialog open={!!revoke} tone="danger" title="Revoke this session"
        message={revoke ? revoke.account + "'s session on " + revoke.device + " from " + revoke.ip + " is signed out immediately." : ""}
        detail="Its next request lands on the sign-in screen. The account's other sessions are unaffected."
        confirmLabel="Revoke session" onClose={() => setRevoke(null)}
        onConfirm={() => { setRows((rs) => rs.filter((x) => x.id !== revoke.id)); onToast && onToast({ tone: "neutral", title: "Session revoked", description: revoke.account + " \u00b7 " + revoke.device }); setRevoke(null); }} />
      <ConfirmDialog open={!!revokeAll} tone="danger" title={revokeAll ? "Revoke every session for " + revokeAll.account.split("@")[0] : ""}
        message={revokeAll ? "Every live session " + revokeAll.account + " holds is signed out immediately." : ""}
        detail="Their membership, role and personal tokens are unaffected; remove the member on Team to fully offboard."
        confirmLabel="Revoke all sessions" typedConfirm={revokeAll ? revokeAll.account : undefined} onClose={() => setRevokeAll(null)}
        onConfirm={() => { setRows((rs) => rs.filter((x) => x.account !== revokeAll.account)); onToast && onToast({ tone: "neutral", title: "All sessions revoked", description: revokeAll.account }); setRevokeAll(null); }} />
    </div>
  );
}

function AuditSection() {
  const [range, setRange] = React.useState({ label: "Last 7d" });
  const ROWS = [
    { when: "2026-08-22T14:41Z", actor: "ola@acmecorp.io", action: "annotation.create", subject: "VG-2481 · accepted risk", ip: "198.51.100.7" },
    { when: "2026-08-22T14:38Z", actor: "system", action: "batch.complete", subject: "2026-08-22T14:00Z", ip: "—" },
    { when: "2026-08-22T11:02Z", actor: "dana@acmecorp.io", action: "seed.add", subject: "203.0.113.0/24", ip: "198.51.100.12" },
    { when: "2026-08-22T10:57Z", actor: "dana@acmecorp.io", action: "exclusion.add", subject: "old-blog.acmecorp.io", ip: "198.51.100.12" },
    { when: "2026-08-21T16:20Z", actor: "ola@acmecorp.io", action: "member.invite", subject: "priya@acmecorp.io · viewer", ip: "198.51.100.7" },
    { when: "2026-08-21T09:12Z", actor: "system", action: "delivery.fail", subject: "pager.example/verge · timeout", ip: "—" },
    { when: "2026-08-20T15:44Z", actor: "sam@acmecorp.io", action: "channel.pause", subject: "pager.example/verge", ip: "203.0.113.80" },
    { when: "2026-08-20T08:01Z", actor: "ola@acmecorp.io", action: "sso.metadata.update", subject: "okta · idp-signing-2026", ip: "198.51.100.7" },
  ];
  return (
    <Card microLabel="Operational record" title="Audit log" pad={0} action={<DateRangePicker value={range} onChange={setRange} />} overflow="visible">
      <Table framed={false} dense columns={[
        { key: "when", label: "When", mono: true, width: 160 },
        { key: "actor", label: "Actor", mono: true, width: 170 },
        { key: "action", label: "Action", mono: true, width: 170 },
        { key: "subject", label: "Subject", mono: true },
        { key: "ip", label: "Source IP", mono: true, align: "right", width: 120 },
      ]} rows={ROWS} rowKey="when" />
    </Card>
  );
}

function InstanceSection({ onToast }) {
  const [backupBusy, setBackupBusy] = React.useState(false);
  const [backupPct, setBackupPct] = React.useState(0);
  const [lastBackup, setLastBackup] = React.useState(null);
  const [preflight, setPreflight] = React.useState(null);
  const [restoreOpen, setRestoreOpen] = React.useState(false);
  const [checksOn, setChecksOn] = React.useState(true);
  const startBackup = () => {
    setBackupBusy(true); setBackupPct(8);
    const t = setInterval(() => setBackupPct((p) => {
      if (p >= 100) { clearInterval(t); setBackupBusy(false); setLastBackup("2026-08-22 14:12 UTC"); onToast && onToast({ tone: "ok", title: "Backup sealed", description: "24.8 GB archive · data only — no secrets inside." }); return p; }
      return p + 23;
    }), 450);
  };
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 24 }}>
      <div style={{ display: "grid", gridTemplateColumns: "repeat(3, 1fr)", gap: 24 }}>
        <Card><Stat label="Version" value="v0.9.2" caption="AGPL-3.0 · self-hosted" /></Card>
        <Card><Stat label="Uptime" value="41d" caption="since last restart" /></Card>
        <Card><Stat label="Queue depth" value="12" caption="subjects waiting" /></Card>
      </div>
      <Card microLabel="Host" title="Storage & database">
        <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
          <Progress value={62} label="Disk" detail="24.8 / 40 GB" />
          <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
            <StatusDot status="ok" label="postgres 16" />
            <span style={{ font: "400 11.5px var(--font-mono)", color: "var(--text-muted)" }}>4.2 GB · vacuumed 6h ago</span>
          </div>
        </div>
      </Card>
      <Card microLabel="Fleet" title="Vantages">
        <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
          {VANTAGE_FLEET.length === 0 && <EmptyState message="No vantage provisioned" detail="Only the shipped resolver position exists. Provision a prober under Scanning · Vantages to measure from the internet." />}
          {VANTAGE_FLEET.map(([n, a, l]) => (
            <div key={n} style={{ display: "flex", alignItems: "center", gap: 10 }}>
              <span style={{ font: "500 12.5px var(--font-mono)", color: "var(--text-ink)" }}>{n}</span>
              <span style={{ marginLeft: "auto", display: "inline-flex", alignItems: "center", gap: 8 }}>
                <span style={{ font: "400 11.5px var(--font-mono)", color: "var(--text-muted)" }}>{l}</span>
                <Badge tone={a === "available" ? "ok" : "neutral"} dot>{a}</Badge>
              </span>
            </div>
          ))}
        </div>
      </Card>
      <Card microLabel="Instance · data" title="Backup">
        <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
          <p style={{ margin: 0, font: "400 13px/1.6 var(--font-ui)", color: "var(--text-secondary)", maxWidth: "72ch" }}>A backup carries the estate and its configuration — data only, never a secret. The session key and the prober key are not in it: on restore both regenerate, and old sessions lapse. That is the design, not a caveat — a stolen backup cannot impersonate this instance.</p>
          {backupBusy ? (
            <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
              <Progress label="Preparing the archive" detail="streaming · ~24.8 GB" value={backupPct} />
              <span style={{ font: "400 11.5px var(--font-ui)", color: "var(--text-muted)" }}>Big estates stream; the download starts when the archive is sealed.</span>
            </div>
          ) : (
            <div style={{ display: "flex", alignItems: "center", gap: 12, flexWrap: "wrap" }}>
              <Button onClick={startBackup}>Download backup</Button>
              <span style={{ font: "400 11.5px var(--font-ui)", color: "var(--text-muted)" }}>{lastBackup ? "Last backup " + lastBackup + " · 24.8 GB" : "No backup has been taken from this UI yet."}</span>
            </div>
          )}
        </div>
      </Card>
      <Card microLabel="Instance · data" title="Restore">
        <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
          <p style={{ margin: 0, font: "400 13px/1.6 var(--font-ui)", color: "var(--text-secondary)", maxWidth: "72ch" }}>Restoring replaces the current estate and configuration with the archive’s — it overwrites what is here now. The archive is pre-flighted before anything is touched, and applying it takes a typed confirmation. Fresh session and prober keys are generated: every session signs in again, and probers re-pin.</p>
          {preflight ? (
            <div style={{ display: "flex", gap: 10, padding: "12px 14px", borderRadius: 12, background: "var(--warn-soft)", border: "1px solid var(--warn-border)" }}>
              <span style={{ display: "inline-flex", color: "var(--warn)", flex: "none", marginTop: 2 }}><Icon name="alert-triangle" size={15} /></span>
              <span style={{ display: "flex", flexDirection: "column", gap: 2 }}>
                <span style={{ font: "600 12.5px var(--font-ui)", color: "var(--text-ink)" }}>Pre-flight — {preflight.file}</span>
                <span style={{ font: "400 12.5px/1.5 var(--font-ui)", color: "var(--text-body)" }}>Taken {preflight.taken} · {preflight.subjects} subjects · schema {preflight.schema}. Applying it overwrites the current data.</span>
                <span style={{ display: "flex", gap: 8, marginTop: 6 }}>
                  <Button size="sm" onClick={() => setRestoreOpen(true)}>Continue to restore…</Button>
                  <Button size="sm" variant="ghost" onClick={() => setPreflight(null)}>Discard</Button>
                </span>
              </span>
            </div>
          ) : (
            <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
              <Button variant="ghost" onClick={() => setPreflight({ file: "verge-backup-2026-08-15.vgbak", taken: "2026-08-15 03:00 UTC", subjects: 214, schema: "21100" })}>Choose archive…</Button>
              <span style={{ font: "400 11.5px var(--font-ui)", color: "var(--text-muted)" }}>Pre-flight only — nothing is applied yet.</span>
            </div>
          )}
        </div>
      </Card>
      <Card microLabel="Instance · release" title="Version & updates">
        <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
          <div style={{ display: "flex", alignItems: "center", gap: 10, flexWrap: "wrap" }}>
            <span style={{ font: "600 15px var(--font-mono)", color: "var(--text-ink)" }}>v0.9.2</span>
            <Badge tone="ok" dot>schema current</Badge>
          </div>
          {checksOn ? (
            <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
              <div style={{ display: "flex", gap: 10, padding: "12px 14px", borderRadius: 12, background: "var(--accent-soft)" }}>
                <span style={{ display: "inline-flex", color: "var(--accent)", flex: "none", marginTop: 2 }}><Icon name="download" size={15} /></span>
                <span style={{ display: "flex", flexDirection: "column", gap: 2 }}>
                  <span style={{ font: "600 12.5px var(--font-ui)", color: "var(--text-ink)" }}>v0.9.3 is available</span>
                  <span style={{ font: "400 12.5px/1.5 var(--font-ui)", color: "var(--text-body)" }}>Drift-batch memory fixes and a faster census. Verge never rewrites its own image — the swap is a host action. The exact steps:</span>
                </span>
              </div>
              <pre style={{ margin: 0, padding: "12px 14px", background: "var(--console-surface)", borderRadius: 12, font: "400 12px/1.7 var(--font-mono)", color: "var(--console-text)", overflowX: "auto" }}>{"# on the host — verge cannot rewrite its own image\ndocker compose pull\ndocker compose up -d web worker\ndocker compose exec web verge migrate status"}</pre>
            </div>
          ) : (
            <span style={{ font: "400 12.5px var(--font-ui)", color: "var(--text-muted)" }}>Update checks are off — air-gap friendly; Verge never phones home while disabled. Compare v0.9.2 against the releases page when you choose.</span>
          )}
          <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
            <Button size="sm" variant="ghost" onClick={() => setChecksOn(!checksOn)}>{checksOn ? "Disable update checks" : "Enable update checks"}</Button>
            <span style={{ font: "400 11.5px var(--font-ui)", color: "var(--text-muted)" }}>Best-effort check against the release feed — a failed check reports nothing and never blocks.</span>
          </div>
        </div>
      </Card>
      <ConfirmDialog open={restoreOpen} title={"Restore " + (preflight ? preflight.file : "")}
        message={preflight ? "This overwrites the current estate and configuration with the archive’s (" + preflight.subjects + " subjects, taken " + preflight.taken + "). Fresh session and prober keys are generated — every session signs in again, and probers re-pin." : ""}
        typedConfirm="restore" confirmLabel="Restore — overwrite current data"
        onConfirm={() => { setRestoreOpen(false); setPreflight(null); onToast && onToast({ tone: "danger", title: "Restore applied", description: "Keys regenerated — every session signs in again; probers re-pin." }); }}
        onClose={() => setRestoreOpen(false)} />
    </div>
  );
}

function ApiSection({ onToast }) {
  const [enabled, setEnabled] = React.useState(false);
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 24 }}>
      <header style={{ display: "flex", flexDirection: "column", gap: 6 }}>
        <span style={{ font: "500 11px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--text-muted)" }}>Access · api access</span>
        <h2 style={{ margin: 0, font: "600 17px var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)" }}>API access</h2>
        <p style={{ margin: 0, font: "400 13px/1.6 var(--font-ui)", color: "var(--text-secondary)", maxWidth: "72ch" }}>A read-only JSON surface at <code style={{ fontFamily: "var(--font-mono)", fontSize: 12 }}>/api/v1</code>, authenticated by personal tokens — the bearer path is fully separate from browser sessions. It is <strong>off by default</strong>; minted tokens stay inert until an admin turns it on. A token can never mutate anything: leaked, it can read the inventory, but it cannot change the estate or its configuration.</p>
      </header>
      <Card microLabel="Read-only · opt-in" title="/api/v1">
        <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
          <div style={{ display: "flex", alignItems: "center", gap: 12, flexWrap: "wrap" }}>
            <Badge tone={enabled ? "ok" : "neutral"}>{enabled ? "enabled" : "disabled"}</Badge>
            <Button size="sm" onClick={() => { const n = !enabled; setEnabled(n); onToast && onToast(n ? { tone: "ok", title: "API access enabled", description: "Personal tokens now answer GET /api/v1/… — read-only, always." } : { tone: "neutral", title: "API access disabled", description: "/api/v1 answers nothing; every minted token is inert again." }); }}>{enabled ? "Disable API access" : "Enable API access"}</Button>
            {enabled && <span style={{ font: "400 11.5px var(--font-ui)", color: "var(--text-muted)" }}>Enabled by ola.perez · just now</span>}
          </div>
          {enabled ? (
            <div style={{ display: "flex", gap: 10, padding: "12px 14px", borderRadius: 12, background: "var(--accent-soft)" }}>
              <span style={{ display: "inline-flex", color: "var(--accent)", flex: "none", marginTop: 2 }}><Icon name="check" size={15} /></span>
              <span style={{ display: "flex", flexDirection: "column", gap: 2 }}>
                <span style={{ font: "600 12.5px var(--font-ui)", color: "var(--text-ink)" }}>Live — read-only, always</span>
                <span style={{ font: "400 12.5px/1.5 var(--font-ui)", color: "var(--text-body)" }}>Personal tokens now answer <span style={{ fontFamily: "var(--font-mono)", fontSize: 12 }}>GET /api/v1/…</span> with their account’s read access. No token can change the estate or its configuration — there is no write surface to enable.</span>
              </span>
            </div>
          ) : (
            <div style={{ display: "flex", gap: 10, padding: "12px 14px", borderRadius: 12, background: "var(--surface-sunken)", border: "1px solid var(--border-default)" }}>
              <span style={{ display: "inline-flex", color: "var(--text-muted)", flex: "none", marginTop: 2 }}><Icon name="lock" size={15} /></span>
              <span style={{ display: "flex", flexDirection: "column", gap: 2 }}>
                <span style={{ font: "600 12.5px var(--font-ui)", color: "var(--text-ink)" }}>Disabled — the default</span>
                <span style={{ font: "400 12.5px/1.5 var(--font-ui)", color: "var(--text-body)" }}><span style={{ fontFamily: "var(--font-mono)", fontSize: 12 }}>/api/v1</span> answers nothing and every minted token is inert. Tokens are managed per-account in the Profile; enabling here is the single switch that makes them live.</span>
              </span>
            </div>
          )}
        </div>
      </Card>
    </div>
  );
}

export function Settings({ onToast, section, onOpenRun }) {
  const [active, setActive] = React.useState(section || "scans");
  React.useEffect(() => { if (section) setActive(section); }, [section]);
  const [loading, setLoading] = React.useState(false);
  const go = (id) => { setActive(id); setLoading(true); setTimeout(() => setLoading(false), 700); };
  return (
    <main data-screen-label="Settings" style={{ maxWidth: 1440, margin: "0 auto", padding: 32, display: "grid", gridTemplateColumns: "210px minmax(0, 1fr)", gap: 40, alignItems: "start" }}>
      <div style={{ display: "flex", flexDirection: "column", gap: 16, position: "sticky", top: 32 }}>
        <h1 style={{ margin: 0, font: "600 21px var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)" }}>Settings</h1>
        <SettingsNav active={active} onNavigate={go} sections={[
          { label: "Scanning", items: [{ id: "scans", label: "Scans", icon: "radar" }, { id: "vantages", label: "Vantages", icon: "server" }] },
          { label: "Access", items: [{ id: "sso", label: "Single sign-on", icon: "key-round" }, { id: "team", label: "Team", icon: "users" }, { id: "sessions", label: "Sessions", icon: "monitor-smartphone" }, { id: "audit", label: "Audit log", icon: "scroll-text" }, { id: "api", label: "API access", icon: "lock" }] },
          { label: "Discovery", items: [{ id: "sources", label: "Sources", icon: "database" }, { id: "aperture", label: "Port aperture", icon: "layout-grid" }] },
          { label: "Instance", items: [{ id: "instance", label: "Health", icon: "activity" }] },
          { label: "Delivery", items: [{ id: "channels", label: "Channels", icon: "send" }, { id: "integrations", label: "Integrations", icon: "puzzle" }, { id: "messages", label: "Messages", icon: "inbox" }, { id: "delivery", label: "Delivery record", icon: "list" }] },
        ]} />
      </div>
      <div>
        {loading ? (
          <div style={{ display: "flex", flexDirection: "column", gap: 14, paddingTop: 6 }}>
            <Skeleton lines={2} width="40%" />
            <Skeleton shape="block" height={140} />
            <Skeleton shape="block" height={220} />
          </div>
        ) : (
          <React.Fragment>
            {active === "scans" && <ScansSection onToast={onToast} onOpenRun={onOpenRun} />}
            {active === "vantages" && <VantagesSection onToast={onToast} />}
            {active === "sso" && <SsoSection onToast={onToast} />}
            {active === "channels" && <ChannelsSection onToast={onToast} />}
            {active === "integrations" && <Integrations onToast={onToast} />}
            {active === "messages" && <MessagesSection />}
            {active === "delivery" && <DeliverySection />}
            {active === "team" && <TeamSection onToast={onToast} />}
            {active === "sessions" && <SessionsSection onToast={onToast} />}
            {active === "audit" && <AuditSection />}
            {active === "instance" && <InstanceSection onToast={onToast} />}
            {active === "api" && <ApiSection onToast={onToast} />}
            {active === "sources" && <SourcesSection onToast={onToast} />}
            {active === "aperture" && <ApertureSection onToast={onToast} />}
          </React.Fragment>
        )}
      </div>
    </main>
  );
}
