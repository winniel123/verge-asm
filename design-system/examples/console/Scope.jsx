import React from "react";
import { Card } from "../../components/display/Card.jsx";
import { TagInput } from "../../components/forms/TagInput.jsx";
import { ProposalReview } from "../../components/forms/ProposalReview.jsx";
import { ExclusionEditor } from "../../components/forms/ExclusionEditor.jsx";
import { CustodyToggle } from "../../components/forms/CustodyToggle.jsx";
import { RefusalCallout } from "../../components/feedback/RefusalCallout.jsx";
import { TreeView } from "../../components/display/TreeView.jsx";
import { CoverageMessageList } from "../../components/display/CoverageMessageList.jsx";
import { Card as _C } from "../../components/display/Card.jsx";
import { FileDrop } from "../../components/forms/FileDrop.jsx";
import { Badge } from "../../components/display/Badge.jsx";
import { Input } from "../../components/forms/Input.jsx";
import { Button } from "../../components/forms/Button.jsx";
import { Icon } from "../../components/media/Icon.jsx";

const DOMAIN = /^([a-z0-9-]+\.)+[a-z]{2,}$/i;
const CIDR = /^(\d{1,3}\.){3}\d{1,3}\/(\d{1,2})$/;

export function Scope({ onToast }) {
  const [seeds, setSeeds] = React.useState(["acmecorp.io", "203.0.113.0/24"]);
  const [refusal, setRefusal] = React.useState(null);
  const [proposals, setProposals] = React.useState([
    { id: "p1", value: "acme-corp.net", kind: "name", source: "registrar match" },
    { id: "p2", value: "acmecorp.dev", kind: "name", source: "certificate SAN" },
    { id: "p3", value: "198.51.100.0/26", kind: "range", source: "announced by AS64500" },
  ]);
  const [exclusions, setExclusions] = React.useState([
    { kind: "subtree", value: "old-blog.acmecorp.io" },
    { kind: "address", value: "203.0.113.128/25" },
  ]);
  const [custody, setCustody] = React.useState(true);
  const [org, setOrg] = React.useState("");
  const [searched, setSearched] = React.useState(false);
  const searchRegistries = () => {
    if (!org.trim()) return;
    setSearched(true);
    setProposals((p) => p.concat({ id: "p" + Date.now(), value: "203.0.114.0/25", kind: "range", source: 'ARIN · org match "' + org.trim() + '"' }));
    onToast && onToast({ tone: "neutral", title: "3 registries answered", description: '1 new proposal for "' + org.trim() + '" · RIPE paths are off until you accept their terms.' });
  };
  const validate = (v) => {
    setRefusal(null);
    if (DOMAIN.test(v)) return null;
    const m = CIDR.exec(v);
    if (m) {
      const prefix = +m[2];
      if (prefix > 32) return "Not a valid prefix.";
      if (prefix < 22) {
        const spans = Math.pow(2, 32 - prefix);
        setRefusal({ input: v, reason: "Spans " + spans.toLocaleString("en-US") + " addresses \u2014 the cap is 1,024 per scope.", reachable: v.split("/")[0] + "/22" });
        return "Refused \u2014 over the 1,024-address cap.";
      }
      return null;
    }
    return "Not a name or an address scope.";
  };
  return (
    <main data-screen-label="Scope" style={{ maxWidth: 1440, margin: "0 auto", padding: 32, display: "flex", flexDirection: "column", gap: 20 }}>
      <header style={{ display: "flex", flexDirection: "column", gap: 2 }}>
        <h1 style={{ margin: 0, font: "600 21px var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)" }}>Scope</h1>
        <span style={{ font: "400 12.5px var(--font-ui)", color: "var(--text-muted)", whiteSpace: "nowrap" }}>What Verge is allowed to look at: seeds, proposals, exclusions, custody.</span>
      </header>
      <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 24, alignItems: "start" }}>
        <div style={{ display: "flex", flexDirection: "column", gap: 24 }}>
          <Card microLabel="Seeds" title="Declared scopes">
            <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
              <TagInput values={seeds} onChange={setSeeds} validate={validate}
                placeholder="acmecorp.io or 203.0.113.0/24" hint="Names or address scopes · try a /20 to see a refusal" />
              {refusal && <RefusalCallout input={refusal.input} reason={refusal.reason} reachable={refusal.reachable} />}
            </div>
          </Card>
          <Card microLabel="Custody" title="Adjacent infrastructure">
            <CustodyToggle enabled={custody} onChange={setCustody} censusCount={62} unit="addresses" />
          </Card>
          <Card microLabel="Removal detection" title="Zone file">
            <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
              <div style={{ display: "flex", alignItems: "center", gap: 10, flexWrap: "wrap" }}>
                <span style={{ font: "500 12px var(--font-mono)", color: "var(--text-body)" }}>acmecorp.io.zone</span>
                <span style={{ font: "400 11.5px var(--font-mono)", color: "var(--text-muted)" }}>uploaded 2026-07-30 · re-supply monthly</span>
                <span style={{ marginLeft: "auto" }}><Badge tone="warn" dot>ages into a gap in 7d</Badge></span>
              </div>
              <FileDrop compact accept=".zone,.txt" label="Re-supply zone file" hint="Upload is a dated act — the upload instant is the observation instant. An apex outside acmecorp.io is refused, with the reason."
                onFiles={() => onToast && onToast({ tone: "ok", title: "Zone accepted", description: "acmecorp.io · observation instant recorded · next re-supply due 2026-09-22." })} />
            </div>
          </Card>
          <Card microLabel="Names" title="Declared name tree">
            <TreeView defaultOpen={["acmecorp.io"]} nodes={[
              { id: "acmecorp.io", label: "acmecorp.io", count: 10, sev: "medium", children: [
                { id: "www", label: "www", sev: "low" },
                { id: "api", label: "api", sev: "high" },
                { id: "vpn", label: "vpn", sev: "critical" },
                { id: "edge-gw-03", label: "edge-gw-03", sev: "critical" },
                { id: "grafana", label: "grafana", sev: "high" },
                { id: "mail", label: "mail" },
                { id: "staging-4", label: "staging-4", sev: "info" },
              ] },
            ]} />
          </Card>
        </div>
        <div style={{ display: "flex", flexDirection: "column", gap: 24 }}>
          <Card microLabel="Coverage" title="Coverage messages">
            <CoverageMessageList messages={[
              { kind: "gap", badge: "no address", subject: "old-blog.acmecorp.io", text: "Expected a resolution; none observed for 3 checks.", when: "2h", iso: "2026-08-22T12:20:04Z" },
              { kind: "stale", bound: "9d", subject: "edge-gw-03.acmecorp.io", text: "Last full service observation is older than the scan cadence.", when: "9d", iso: "2026-08-13T04:44:19Z" },
              { kind: "silent", subject: "dc-fra-01", text: "Vantage stopped reporting mid-batch; open spans are not evaluable.", when: "41m", iso: "2026-08-22T13:41:02Z" },
            ]} />
          </Card>
          <Card microLabel="Proposals" title="From the registry" action={<span style={{ font: "500 10.5px var(--font-mono)", padding: "1px 7px", borderRadius: 999, background: "var(--surface-sunken)", color: "var(--text-secondary)" }}>{proposals.length}</span>}>
            <div style={{ display: "flex", gap: 8, alignItems: "flex-end", marginBottom: 14 }}>
              <Input label="Org-name search" placeholder="Acme Corporation" value={org} onChange={(e) => setOrg(e.target.value)} style={{ flex: 1 }} />
              <Button variant="secondary" icon={<Icon name="search" size={14} />} onClick={searchRegistries}>Search registries</Button>
            </div>
            <span style={{ display: "block", font: "400 11.5px/1.6 var(--font-ui)", color: "var(--text-muted)", marginBottom: 14 }}>Proposers answer an org-name search with address scopes — never subdomains. A proposal asserts nothing until confirmed into a seed; declines are recorded as exclusions.</span>
            {proposals.length ? (
              <ProposalReview proposals={proposals}
                onConfirm={(p) => { setProposals(proposals.filter((x) => x.id !== p.id)); setSeeds(seeds.concat(p.value)); onToast && onToast({ tone: "ok", title: "Proposal confirmed", description: p.value + " added to scope." }); }}
                onDecline={(ids) => { setProposals(proposals.filter((x) => ids.indexOf(x.id) === -1)); onToast && onToast({ tone: "neutral", title: "Proposals declined", description: ids.length + " declined." }); }} />
            ) : (
              <span style={{ font: "400 12.5px var(--font-ui)", color: "var(--text-muted)" }}>No open proposals. The registry suggests scopes as it sees them.</span>
            )}
          </Card>
          <Card microLabel="Exclusions" title="Never scanned">
            <ExclusionEditor exclusions={exclusions}
              onAdd={(kind, value) => setExclusions(exclusions.concat({ kind, value }))}
              onRemove={(i) => setExclusions(exclusions.filter((_, j) => j !== i))} />
          </Card>
        </div>
      </div>
    </main>
  );
}
