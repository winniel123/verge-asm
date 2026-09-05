import React from "react";
import { Card } from "../../components/display/Card.jsx";
import { CoverageMeter } from "../../components/display/CoverageMeter.jsx";
import { CoverageMessageList } from "../../components/display/CoverageMessageList.jsx";
import { GapBadge } from "../../components/display/GapBadge.jsx";
import { SignalRuleRef } from "../../components/display/SignalRuleRef.jsx";
import { Table } from "../../components/display/Table.jsx";
import { Callout } from "../../components/feedback/Callout.jsx";
import { Button } from "../../components/forms/Button.jsx";

export function Coverage({ onOpenScope }) {
  return (
    <main data-screen-label="Coverage" style={{ maxWidth: 1440, margin: "0 auto", padding: 32, display: "flex", flexDirection: "column", gap: 20 }}>
      <header style={{ display: "flex", flexDirection: "column", gap: 2 }}>
        <h1 style={{ margin: 0, font: "600 21px var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)" }}>Coverage</h1>
        <span style={{ font: "400 12.5px var(--font-ui)", color: "var(--text-muted)", whiteSpace: "nowrap" }}>Where "we cannot construct this claim" lives — a feature, not an error.</span>
      </header>
      <div style={{ display: "grid", gridTemplateColumns: "minmax(0, 1fr) minmax(0, 1fr)", gap: 24, alignItems: "start" }}>
        <div style={{ display: "flex", flexDirection: "column", gap: 24 }}>
          <Card microLabel="Aperture" title="What the last batch walked">
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
          <Card microLabel="Rules" title="Unevaluable this batch">
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
