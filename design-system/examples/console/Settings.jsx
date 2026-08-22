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
import { Icon } from "../../components/media/Icon.jsx";
import { Switch } from "../../components/forms/Switch.jsx";
import { FileDrop } from "../../components/forms/FileDrop.jsx";
import { Callout } from "../../components/feedback/Callout.jsx";
import { CopyValue } from "../../components/display/CopyValue.jsx";
import { CertificateCard } from "../../components/display/CertificateCard.jsx";
import { MappingEditor } from "../../components/forms/MappingEditor.jsx";
import { IdPButton } from "../../components/forms/IdPButton.jsx";

const BASE_LOG = [
  { time: "14:00:02", text: "batch started · 214 subjects · 3 vantages" },
  { time: "14:00:09", text: "dns sweep · acmecorp.io · 1,284 names" },
  { time: "14:00:41", level: "warn", text: "vantage ap-south-1 missed check (2/3)" },
  { time: "14:01:12", text: "tls-acceptance · vpn.acmecorp.io:443" },
  { time: "14:02:03", level: "error", text: "connect refused · 203.0.113.44:22" },
];

function ScansSection() {
  const [profile, setProfile] = React.useState("standard");
  const [cadence, setCadence] = React.useState("Daily · 08:00");
  const [cron, setCron] = React.useState("");
  const [running, setRunning] = React.useState(false);
  const [timeout_, setTimeout_] = React.useState(800);
  const [cap, setCap] = React.useState(1024);
  const [selDay, setSelDay] = React.useState("2026-08-24");
  const RUNS = React.useMemo(() => { const e = {}; for (let d = 1; d <= 31; d++) { const s = "2026-08-" + String(d).padStart(2, "0"); e[s] = new Date(2026, 7, d).getDay() === 1 ? 2 : 1; } return e; }, []);
  const [log, setLog] = React.useState(BASE_LOG);
  const run = () => {
    if (running) return;
    setRunning(true);
    let n = 0;
    const t = setInterval(() => {
      n++;
      setLog((l) => l.concat({ time: "14:0" + (3 + Math.floor(n / 4)) + ":" + String(10 + n * 7).slice(-2), text: "scan · subject " + (200 + n) + " of 214" }));
      if (n >= 6) { clearInterval(t); setRunning(false); }
    }, 700);
  };
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 24 }}>
      <Card microLabel="dns · zone · ct · tls-acceptance · ports" title="Recurring intent"
        action={<StatusDot status="ok" label="scheduler healthy" />}>
        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 24, alignItems: "start" }}>
          <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
            <span style={{ fontSize: 12.5, fontWeight: 500, color: "var(--text-body)" }}>Profile</span>
            <RadioCards label="Scan profile" value={profile} onChange={setProfile} columns={1} options={[
              { value: "standard", title: "Standard", description: "Top 1,000 TCP ports, plus any port previously seen.", meta: "default" },
              { value: "deep", title: "Deep", description: "Full TCP and common UDP. Slower batches." },
              { value: "passive", title: "Passive only", description: "Public datasets only \u2014 no active probing." },
            ]} />
          </div>
          <div style={{ display: "flex", flexDirection: "column", gap: 18 }}>
            <CadenceSelect value={cadence} customValue={cron} onChange={(v, c) => { setCadence(v); setCron(c || ""); }} />
            <Slider label="Connect timeout" value={timeout_} onChange={setTimeout_} min={100} max={2000} step={100} unit="ms" />
            <NumberInput label="Addresses per scope" value={cap} onChange={setCap} min={64} max={1024} step={64} unit="addresses" hint="The declaration cap — refusals name the reachable set" />
          </div>
        </div>
      </Card>
      <Card microLabel="Schedule" title="Upcoming runs">
        <div style={{ display: "flex", gap: 28, flexWrap: "wrap", alignItems: "flex-start" }}>
          <Calendar month="2026-08" selected={selDay} onSelect={setSelDay} events={RUNS} label="Scheduled scan runs" />
          <div style={{ display: "flex", flexDirection: "column", gap: 8, minWidth: 200, paddingTop: 4, flex: 1 }}>
            <span style={{ font: "500 10.5px var(--font-mono)", letterSpacing: "0.06em", textTransform: "uppercase", color: "var(--text-muted)" }}>{selDay}</span>
            <span style={{ font: "400 12.5px/1.6 var(--font-ui)", color: "var(--text-body)" }}>standard · 08:00 · all 214 subjects</span>
            {RUNS[selDay] === 2 && <span style={{ font: "400 12.5px/1.6 var(--font-ui)", color: "var(--text-body)" }}>deep · 02:00 · TLS acceptance + full port census</span>}
            <span style={{ font: "400 11.5px/1.6 var(--font-ui)", color: "var(--text-muted)" }}>Dots are volume — how many runs touch the day.</span>
          </div>
        </div>
      </Card>
      <Card microLabel="Batches" title="Recent runs" action={<Button size="sm" icon={running ? undefined : <Icon name="play" size={13} />} disabled={running} onClick={run}>{running ? <Spinner size={14} label="Running" /> : "Run now"}</Button>}>
        <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
          <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
            {running && <BatchStatus status="running" scope="all scopes" />}
            <BatchStatus status="complete" scope="2026-08-22T14:00Z" />
            <BatchStatus status="complete" scope="2026-08-22T08:00Z" />
            <BatchStatus status="failed" scope="2026-08-21T20:00Z" />
          </div>
          <LogViewer title="batch 2026-08-22T14:00Z" live={running} lines={log} height={180} />
        </div>
      </Card>
    </div>
  );
}

function VantagesSection() {
  const [vant, setVant] = React.useState("eu-west-1");
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 24 }}>
      <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(300px, 1fr))", gap: 16 }}>
        <VantageCard name="eu-west-1" vantageClass="internet" resolver="9.9.9.9" availability="available" latency="34ms" />
        <VantageCard name="us-east-2" vantageClass="internet" resolver="9.9.9.9" availability="available" latency="51ms" />
        <VantageCard name="ap-south-1" vantageClass="unverified" resolver="9.9.9.9" availability="unverified" />
      </div>
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
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 24 }}>
      <Card microLabel="Channels" title="One-way delivery">
        <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
          {rows.map((c) => (
            <div key={c.url} style={{ display: "flex", alignItems: "center", gap: 10, padding: "9px 12px", background: "var(--surface-sunken)", borderRadius: 10, flexWrap: "wrap" }}>
              <span style={{ font: "500 12px var(--font-mono)", color: "var(--text-body)" }}>{c.url}</span>
              {c.classes.map((k) => <Tag key={k}>{k}</Tag>)}
              <span style={{ marginLeft: "auto" }}><Badge tone={c.status === "active" ? "ok" : "neutral"} dot>{c.status}</Badge></span>
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

export function Settings({ onToast, section }) {
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
          { label: "Access", items: [{ id: "sso", label: "Single sign-on", icon: "key-round" }] },
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
            {active === "scans" && <ScansSection />}
            {active === "vantages" && <VantagesSection />}
            {active === "sso" && <SsoSection onToast={onToast} />}
            {active === "channels" && <ChannelsSection onToast={onToast} />}
            {active === "integrations" && <Integrations onToast={onToast} />}
            {active === "messages" && <MessagesSection />}
            {active === "delivery" && <DeliverySection />}
          </React.Fragment>
        )}
      </div>
    </main>
  );
}
