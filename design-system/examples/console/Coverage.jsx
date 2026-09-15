import React from "react";
import { Badge } from "../../components/display/Badge.jsx";
import { Card } from "../../components/display/Card.jsx";
import { CoverageMeter } from "../../components/display/CoverageMeter.jsx";
import { CoverageMessageList } from "../../components/display/CoverageMessageList.jsx";
import { GapBadge } from "../../components/display/GapBadge.jsx";
import { SignalRuleRef } from "../../components/display/SignalRuleRef.jsx";
import { Table } from "../../components/display/Table.jsx";
import { Callout } from "../../components/feedback/Callout.jsx";
import { Button } from "../../components/forms/Button.jsx";

/* Row copy is the shipped statement's, verbatim (cmd/web/aperturestatement.go, SPEC docs/spec/aperture-statement.md §2.5). */
const STATEMENT = [
  {
    input: "Enabled sources",
    cadence: "daily · every 5 minutes",
    cadenceWhy: "The ct Scan asks daily and the ct-tail Scan every 5 minutes. Release-coupled: a cadence dial ships for the dns and zone Scans alone.",
    state: "crt.sh",
    kind: "on",
    figures: [{ text: "1 of 3 sources enabled" }],
    detail: "Your own toggles over the shipped defaults, never a batch. The count is over sources that admit a Name, so it counts no proposer. Only the worker key selects Cert Spotter in place of crt.sh, so this row names what is declared, never what ran.",
    remedy: "Enable a source",
    remedyHref: "/settings?tab=sources",
    remedyWhy: "A toggle on the Sources tab reaches each source still off: CT drift tail (logs-direct) · Cert Spotter (operator key). Cert Spotter also needs its key on the worker.",
  },
  {
    input: "Port and transport tiers",
    cadence: "daily · monthly",
    cadenceWhy: "hot daily, cold monthly. Release-coupled: no operator dial exists.",
    state: "hot on · cold off · udp no flag",
    kind: "off",
    figures: [
      { text: "5 of 38 sensitive pairs unread" },
      { text: "0 of 38 sensitive pairs the instrument cannot report as reached", zero: true },
      { text: "0 of 17 rules unevaluable", zero: true },
    ],
    detail: "The cold tier's state is the shadow of an empty scope list, not a switch. UDP has no flag at all.",
    remedy: "none",
    remedyWhy: "The 5 pairs still unread are UDP. No tier reads them, and no setting opens one.",
  },
  {
    input: "The custody gate",
    cadence: "every dispatch · daily",
    cadenceWhy: "The gate runs at every connect dispatch, and the extension's fan-out test rides the daily edge-fanout Scan. Release-coupled: a cadence dial ships for the dns and zone Scans alone.",
    state: "total · extension on",
    kind: "on",
    figures: [{ text: "1 of 1 name scope extended" }],
    detail: "The gate derives custody before any vantage class is read, and refuses every address it does not derive as yours. A declared address scope admits an address directly. A custody extension admits the addresses a name scope resolves into.",
    remedy: "none",
    remedyWhy: "Every declared name scope carries the extension, so no further switch widens this gate.",
  },
  {
    input: "The queried qtype set",
    cadence: "daily",
    cadenceWhy: "The dns Scan re-asks the whole set on every run. A cadence dial ships for this Scan: the DNS scan interval on the Scope screen moves it.",
    state: "A · AAAA · CNAME · NS · SOA · MX · TXT",
    kind: "fixed",
    detail: "The set a prober puts on the wire, never a library default. Each qtype is asked by name and never as ANY, because a server may answer ANY with a subset, and a subset licenses no absence. The wildcard control probe runs this same set and mints no second list.",
    remedy: "none",
    remedyWhy: "No setting narrows this set. An offer the operator can narrow is a finding the operator can silence, so the set moves with a release and never with a switch.",
  },
  {
    input: "The TLS candidate set",
    cadence: "weekly · daily · monthly",
    cadenceWhy: "The tls-acceptance Scan re-asks the whole set weekly, and the certificate handshake carries the same list on whichever port tier makes the connect: hot daily, and cold monthly where the cold tier runs. Release-coupled: a cadence dial ships for the dns and zone Scans alone.",
    state: "TLS 1.0 · 1.1 · 1.2 · 1.3 · 19 cipher suites",
    kind: "fixed",
    detail: "The versions and suites a prober puts on the wire, never a library default. The floor is TLS 1.0 on purpose, because a higher floor reports a TLS-1.0-only listener as no TLS at all. No TLS 1.3 suite sits in that count, because the library picks the 1.3 suites itself and reads no declared list there.",
    remedy: "none",
    remedyWhy: "No setting narrows this set. An offer the operator can narrow is a finding the operator can silence, so the set moves with a release and never with a switch. One list serves both TLS exchanges, so widening it would cost a Break on every acceptance and certificate timeline at once.",
  },
  {
    input: "Vantage class",
    cadence: "none",
    cadenceWhy: "The class is derived where it is used, so it carries no currency and needs no cadence.",
    state: "2 internet · 1 internal",
    kind: "on",
    detail: "A class is derived from the addresses a vantage presents and your declared address scopes, never from a stored field.",
    remedy: "none",
    remedyWhy: "A vantage reads from each side of your boundary, so no class is missing.",
  },
  {
    input: "The control-probe population",
    cadence: "daily",
    cadenceWhy: "The dns Scan rebuilds the population from its own resolution scope on every run. A cadence dial ships for this Scan: the DNS scan interval on the Scope screen moves how often it is rebuilt. No dial moves what it holds.",
    state: "derived per batch · 10 control labels per parent",
    kind: "fixed",
    detail: "The population discriminates a wildcard: a name is decided at its parent, never at its own apex. The dns Scan takes the parent of each name it resolves inside a declared name scope, and deduplicates them. The label count is per surviving parent, never per name. The probing gate stops the population at your scope, so a parent above your own apex is never probed. A name whose parent went unprobed records a Gap and never a value.",
    remedy: "none",
    remedyWhy: "No switch suppresses a control probe, so this population carries no toggle of its own. It widens where your declared name scopes resolve more names, and narrows where they resolve fewer.",
  },
];

const LEDGER_HEADS = [["Input", 190], ["Cadence", 220], ["State", undefined], ["Remedy", 250]];

const LEDGER_TH = { padding: "9px 16px", font: "500 11px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--text-muted)", borderBottom: "1px solid var(--border-default)", textAlign: "left", whiteSpace: "nowrap" };

const LEDGER_NOTE = { font: "400 11px/1.5 var(--font-ui)", color: "var(--text-muted)" };

const LEDGER_CHIP = { height: "auto", minHeight: 20, padding: "2px 7px", font: "400 11.5px var(--font-mono)", whiteSpace: "normal" };

const ledgerTd = (first) => ({ padding: "12px 16px", verticalAlign: "top", borderTop: first ? "none" : "1px solid var(--row-sep)" });

export function Coverage({ onOpenScope, onOpenSources }) {
  const remedyNav = { "/settings?tab=sources": onOpenSources, "/scope": onOpenScope };
  return (
    <main data-screen-label="Coverage" style={{ maxWidth: 1440, margin: "0 auto", padding: 32, display: "flex", flexDirection: "column", gap: 20 }}>
      <header style={{ display: "flex", flexDirection: "column", gap: 2 }}>
        <h1 style={{ margin: 0, font: "600 21px var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)" }}>Coverage</h1>
        <span style={{ font: "400 12.5px var(--font-ui)", color: "var(--text-muted)", whiteSpace: "nowrap" }}>Where "we cannot construct this claim" lives — a feature, not an error.</span>
      </header>
      <Card microLabel="Aperture" title="What this install is configured to look at" pad={0}>
        {/* A ledger cell wraps and top-aligns; Table's cells are single-line, clipped and middle-aligned. */}
        <table style={{ width: "100%", borderCollapse: "separate", borderSpacing: 0 }}>
          <thead>
            <tr>{LEDGER_HEADS.map(([label, width]) => <th key={label} style={{ ...LEDGER_TH, width }}>{label}</th>)}</tr>
          </thead>
          <tbody>
            {STATEMENT.map((row, i) => (
              <tr key={row.input}>
                <td style={{ ...ledgerTd(i === 0), font: "600 13px var(--font-ui)", color: "var(--text-ink)" }}>{row.input}</td>
                <td style={ledgerTd(i === 0)}>
                  <div style={{ display: "flex", flexDirection: "column", gap: 3 }}>
                    <span style={{ font: "400 12.5px var(--font-mono)", color: "var(--text-body)" }}>{row.cadence}</span>
                    <span style={LEDGER_NOTE}>{row.cadenceWhy}</span>
                  </div>
                </td>
                <td style={ledgerTd(i === 0)}>
                  <div style={{ display: "flex", flexDirection: "column", gap: 6, alignItems: "flex-start" }}>
                    <Badge tone={row.kind === "on" ? "ok" : "neutral"} dot={row.kind !== "fixed"} style={LEDGER_CHIP}>{row.state}</Badge>
                    {row.figures && (
                      <div style={{ display: "flex", flexDirection: "column", gap: 4, padding: "9px 11px", borderRadius: "var(--r-sm)", background: "var(--surface-sunken)", border: "1px solid var(--border-default)" }}>
                        {row.figures.map((f) => <span key={f.text} style={{ font: "400 12.5px/1.5 var(--font-mono)", color: f.zero ? "var(--text-secondary)" : "var(--text-ink)" }}>{f.text}</span>)}
                      </div>
                    )}
                    <span style={LEDGER_NOTE}>{row.detail}</span>
                  </div>
                </td>
                <td style={ledgerTd(i === 0)}>
                  <div style={{ display: "flex", flexDirection: "column", gap: 4, alignItems: "flex-start" }}>
                    {row.remedyHref
                      ? <a href={row.remedyHref} onClick={remedyNav[row.remedyHref] ? (e) => { e.preventDefault(); remedyNav[row.remedyHref](); } : undefined} style={{ display: "inline-flex", alignItems: "center", gap: 5, font: "500 12.5px var(--font-ui)", color: "var(--link)", textDecoration: "none" }}>{row.remedy}<span style={{ fontFamily: "var(--font-mono)", opacity: 0.7 }}>→</span></a>
                      : <span style={{ font: "400 12px var(--font-mono)", color: "var(--text-muted)" }}>{row.remedy}</span>}
                    <span style={LEDGER_NOTE}>{row.remedyWhy}</span>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </Card>
      <div style={{ display: "grid", gridTemplateColumns: "minmax(0, 1fr) minmax(0, 1fr)", gap: 24, alignItems: "start" }}>
        <div style={{ display: "flex", flexDirection: "column", gap: 24 }}>
          <Card microLabel="Address scopes" title="What the last batch walked">
            <div style={{ display: "flex", flexDirection: "column", gap: 18 }}>
              <CoverageMeter label="203.0.113.0/24" counted={198} total={214} unit="subjects" detail="16 skipped: excluded subtree + 3 unresolvable names" />
              <CoverageMeter label="acmecorp.io (name scope)" counted={62} unit="addresses" detail="census state — a name scope has no denominator; custody extension reaches what resolution reveals" />
            </div>
          </Card>
          <Card microLabel="Currency" title="Coverage messages">
            <CoverageMessageList messages={[
              { kind: "gap", badge: "no address", subject: "old-blog.acmecorp.io", text: "Expected a resolution; none observed for 3 checks.", when: "2h", iso: "2026-08-22T12:20:04Z" },
              { kind: "stale", bound: "9d", subject: "internal.acmecorp.io zone", text: "Zone aged past two re-supply intervals — the source went stale.", when: "9d", iso: "2026-08-13T04:44:19Z" },
              { kind: "silent", subject: "dc-fra-01", text: "Vantage stopped reporting mid-batch; open spans are not evaluable.", when: "41m", iso: "2026-08-22T13:41:02Z" },
              { kind: "not-evaluable", subject: "ap-south-1 conclusions", text: "Missed 2 of 3 checks this batch; exposure conclusions marked unverified.", when: "5h", iso: "2026-08-22T09:03:55Z" },
            ]} />
          </Card>
        </div>
        <div style={{ display: "flex", flexDirection: "column", gap: 24 }}>
          <Card microLabel="Gaps" title="Expected, not observed" pad={0}>
            <Table framed={false} dense columns={[
              { key: "subject", label: "Subject", mono: true },
              { key: "gap", label: "Gap", width: 130, render: (r) => <GapBadge size="sm" label={r.gap} /> },
              { key: "expected", label: "Expected", width: 190 },
              { key: "since", label: "Since", mono: true, align: "right", width: 60 },
            ]} rows={[
              { subject: "old-blog.acmecorp.io", gap: "no address", expected: "A record", since: "2h" },
              { subject: "203.0.113.44:22", gap: "no banner", expected: "ssh identification", since: "6h" },
              { subject: "mail.acmecorp.io:25", gap: "no exchange", expected: "smtp greeting", since: "1d" },
            ]} rowKey="subject" />
          </Card>
          <Card microLabel="Rules" title="Rules waiting on a reading">
            <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
              {[["tls-weak-key", 3, "needs a completed tls-acceptance exchange; none committed this batch"], ["zone-removal", 1, "needs a fresh zone file; the upload aged into a gap"]].map(([id, v, why]) => (
                <div key={id} style={{ display: "flex", alignItems: "baseline", gap: 10, flexWrap: "wrap" }}>
                  <SignalRuleRef id={id} version={v} />
                  <span style={{ font: "400 12px/1.6 var(--font-ui)", color: "var(--text-muted)" }}>{why}</span>
                </div>
              ))}
            </div>
          </Card>
          <Callout tone="warn" title="Zone gone stale">internal.acmecorp.io's zone file is 2 re-supply intervals old — removal detection is suspended for that scope until a fresh upload.
            <div style={{ marginTop: 10 }}><Button size="sm" variant="secondary" onClick={onOpenScope}>Upload zone</Button></div>
          </Callout>
        </div>
      </div>
    </main>
  );
}
