import React from "react";
import { Card } from "../../components/display/Card.jsx";
import { Tag } from "../../components/display/Tag.jsx";
import { Badge } from "../../components/display/Badge.jsx";
import { Switch } from "../../components/forms/Switch.jsx";
import { Checkbox } from "../../components/forms/Checkbox.jsx";
import { Button } from "../../components/forms/Button.jsx";
import { Dialog } from "../../components/feedback/Dialog.jsx";
import { Callout } from "../../components/feedback/Callout.jsx";
import { TagInput } from "../../components/forms/TagInput.jsx";
import { Icon } from "../../components/media/Icon.jsx";

const CATALOGUE = [
  { id: "crtsh", name: "crt.sh", kind: "source", what: "Names from certificate-transparency SAN lists · daily, throttled", tier: "unencumbered", on: true },
  { id: "arin", name: "ARIN", kind: "proposer", what: "North America · keyless org → prefix", tier: "unencumbered", on: true },
  { id: "afrinic", name: "AFRINIC", kind: "proposer", what: "Africa · CAIDA ⋈ delegated-stats", tier: "unencumbered", on: true },
  { id: "apnic", name: "APNIC", kind: "proposer", what: "Asia-Pacific · CAIDA ⋈ delegated-stats", tier: "unencumbered", on: true },
  { id: "ripestat", name: "RIPEstat", kind: "proposer", what: "RIPE region · org → prefix", tier: "operator-accepted", on: false,
    terms: ["Non-commercial use of the RIPEstat Data API within rate limits.", "Attribution of RIPE NCC as the data source.", "You bear the reading — this is your acceptance, not the project's."] },
  { id: "ripedb", name: "RIPE Database", kind: "proposer", what: "RIPE region · registry queries", tier: "operator-accepted", on: false,
    terms: ["RIPE Database terms: queries for network operations, not bulk harvesting.", "Personal data in objects stays subject to the RIPE DB acceptable use policy.", "You bear the reading — this is your acceptance, not the project's."] },
  { id: "apnicreg", name: "APNIC registry", kind: "proposer", what: "APNIC region · registry queries", tier: "operator-accepted", on: false,
    terms: ["APNIC Whois terms: operational use, no redistribution of bulk data.", "You bear the reading — this is your acceptance, not the project's."] },
  { id: "lacnic", name: "LACNIC registry", kind: "proposer", what: "Latin America · registry queries", tier: "operator-accepted", on: false,
    terms: ["LACNIC rate limits and acceptable-use apply to registry queries.", "You bear the reading — this is your acceptance, not the project's."] },
  { id: "hackertarget", name: "HackerTarget", kind: "source", what: "—", tier: "barred", on: false },
  { id: "certspotter", name: "Cert Spotter", kind: "source", what: "unauthenticated tier", tier: "barred", on: false },
];

function SourceRow({ e, onToggle }) {
  return (
    <div style={{ display: "flex", alignItems: "center", gap: 10, padding: "10px 12px", background: "var(--surface-sunken)", borderRadius: 10, flexWrap: "wrap" }}>
      <span style={{ font: "500 13px var(--font-ui)", color: "var(--text-ink)" }}>{e.name}</span>
      <Tag>{e.kind}</Tag>
      <span style={{ font: "400 11.5px var(--font-ui)", color: "var(--text-muted)" }}>{e.what}</span>
      <span style={{ marginLeft: "auto", display: "inline-flex", alignItems: "center", gap: 10 }}>
        {e.tier === "barred"
          ? <Badge tone="neutral">barred — excluded on terms</Badge>
          : <React.Fragment>
              <Badge tone={e.tier === "unencumbered" ? "ok" : "warn"}>{e.tier}</Badge>
              <Switch checked={e.on} onChange={() => onToggle(e)} aria-label={(e.on ? "Disable " : "Enable ") + e.name} />
            </React.Fragment>}
      </span>
    </div>
  );
}

/* The /sources catalogue: sources admit, proposers only suggest; consent is release-authored. */
export function SourcesSection({ onToast }) {
  const [entries, setEntries] = React.useState(CATALOGUE);
  const [consent, setConsent] = React.useState(null); // entry awaiting terms acceptance
  const [agreed, setAgreed] = React.useState(false);
  const toast = (title, description, tone) => onToast && onToast({ tone: tone || "neutral", title, description });
  const flip = (id, on) => setEntries((es) => es.map((x) => (x.id === id ? { ...x, on } : x)));
  const onToggle = (e) => {
    if (e.on) { flip(e.id, false); toast("Source disabled", e.name + " · dated by the next batch"); }
    else if (e.tier === "operator-accepted") { setAgreed(false); setConsent(e); }
    else { flip(e.id, true); toast("Source enabled", e.name + " · dated by the next batch", "ok"); }
  };
  const shippedOn = entries.filter((e) => e.tier === "unencumbered");
  const acceptOff = entries.filter((e) => e.tier === "operator-accepted");
  const barred = entries.filter((e) => e.tier === "barred");
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 24 }}>
      <Callout tone="neutral" title="Toggling is admin-only">Every account can read this catalogue. Enabling or disabling is an admin act — it keeps no log line of its own; it is dated by the batch whose recorded source set it moved.</Callout>
      <Card microLabel="unencumbered" title="Shipped on">
        <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
          {shippedOn.map((e) => <SourceRow key={e.id} e={e} onToggle={onToggle} />)}
          <span style={{ font: "400 11.5px/1.6 var(--font-ui)", color: "var(--text-muted)" }}>A source admits subjects on its own authority; a proposer only answers an org-name search with address scopes — nothing joins the estate until you confirm it into a seed.</span>
        </div>
      </Card>
      <Card microLabel="operator-accepted" title="Ship off — accept the terms">
        <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
          {acceptOff.map((e) => <SourceRow key={e.id} e={e} onToggle={onToggle} />)}
        </div>
      </Card>
      <Card microLabel="barred" title="Not run for anyone">
        <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
          {barred.map((e) => <SourceRow key={e.id} e={e} onToggle={onToggle} />)}
        </div>
      </Card>
      <Dialog open={!!consent} title={consent ? "Enable " + consent.name : ""} description="The project could not clear these terms on your behalf. Your acceptance, your reading." onClose={() => setConsent(null)}
        footer={<React.Fragment>
          <Button variant="ghost" onClick={() => setConsent(null)}>Cancel</Button>
          <Button disabled={!agreed} onClick={() => { flip(consent.id, true); toast("Source enabled", consent.name + " · acceptance recorded, dated by the next batch", "ok"); setConsent(null); }}>Accept and enable</Button>
        </React.Fragment>}>
        <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
          <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
            {(consent ? consent.terms : []).map((t, i) => (
              <div key={i} style={{ display: "flex", gap: 10, alignItems: "flex-start" }}>
                <Icon name="check" size={13} style={{ color: "var(--text-muted)", flex: "none", marginTop: 3 }} />
                <span style={{ font: "400 12.5px/1.6 var(--font-ui)", color: "var(--text-body)" }}>{t}</span>
              </div>
            ))}
          </div>
          <Checkbox label="I accept these terms" checked={agreed} onChange={setAgreed} />
        </div>
      </Dialog>
    </div>
  );
}

/* verge-core: the default port aperture. Frequency tier editable; sensitive tier locked by design. */
export function ApertureSection({ onToast }) {
  const [freq, setFreq] = React.useState([":80", ":443", ":22", ":8080", ":8443", ":3000", ":5432", ":6379"]);
  const SENSITIVE = [[":23", "telnet"], [":3389", "rdp"], [":5900", "vnc"], [":445", "smb"], [":1433", "mssql"], [":21", "ftp"]];
  const validate = (v) => {
    if (v.charAt(0) !== ":") return "Ports are written :N";
    const n = +v.slice(1);
    if (!(n >= 1 && n <= 65535) || String(n) !== v.slice(1)) return "Not a port";
    return null;
  };
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 24 }}>
      <Card microLabel="verge-core" title="Sensitive tier" action={<Badge tone="neutral">locked</Badge>}>
        <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
          <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
            {SENSITIVE.map(([p, svc]) => (
              <span key={p} style={{ display: "inline-flex", alignItems: "center", gap: 7, padding: "5px 10px", background: "var(--surface-sunken)", border: "1px solid var(--border-default)", borderRadius: 999 }}>
                <Icon name="lock" size={11} style={{ color: "var(--text-muted)" }} />
                <span style={{ font: "500 12px var(--font-mono)", color: "var(--text-ink)" }}>{p}</span>
                <span style={{ font: "400 11px var(--font-mono)", color: "var(--text-muted)" }}>{svc}</span>
              </span>
            ))}
          </div>
          <Callout tone="neutral" title="Not editable, on purpose">A port you can hide is a signal you can silence. The sensitive tier is release-authored and moves only with the release.</Callout>
        </div>
      </Card>
      <Card microLabel="verge-core" title="Frequency tier">
        <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
          <TagInput values={freq} onChange={(v) => { setFreq(v); onToast && onToast({ tone: "ok", title: "Aperture updated", description: v.length + " frequency-tier ports · applies from the next census" }); }}
            validate={validate} placeholder=":8080" hint="Admins may widen or narrow this tier · written :port" />
          <span style={{ font: "400 11.5px/1.6 var(--font-ui)", color: "var(--text-muted)" }}>The census walks the union of both tiers plus any port previously seen on the subject.</span>
        </div>
      </Card>
    </div>
  );
}
