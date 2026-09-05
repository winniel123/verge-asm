import React from "react";
import { Breadcrumb } from "../../components/navigation/Breadcrumb.jsx";
import { Card } from "../../components/display/Card.jsx";
import { Table } from "../../components/display/Table.jsx";
import { Tag } from "../../components/display/Tag.jsx";
import { Badge } from "../../components/display/Badge.jsx";
import { SeverityBadge } from "../../components/display/SeverityBadge.jsx";
import { ExposureBadge } from "../../components/display/ExposureBadge.jsx";
import { KeyValueList } from "../../components/display/KeyValueList.jsx";
import { CopyValue } from "../../components/display/CopyValue.jsx";
import { WithdrawnMark } from "../../components/display/WithdrawnMark.jsx";
import { Banner } from "../../components/feedback/Banner.jsx";
import { Button } from "../../components/forms/Button.jsx";
import { DropdownMenu } from "../../components/feedback/DropdownMenu.jsx";
import { Icon } from "../../components/media/Icon.jsx";

const CHAIN = {
  service: [
    { label: "Service", value: "203.0.113.7:5900/tcp", detail: "an (address, port, transport) triple" },
    { label: "Address · cited by a current resolution", value: "203.0.113.7" },
    { label: "Name · citing resolution", value: "edge-gw-03.acmecorp.io", detail: "A record, seen 4m ago" },
    { label: "Seed · name scope", value: "acmecorp.io", detail: "declared 2026-06-14 · custody verified" },
  ],
  endpoint: [
    { label: "Endpoint", value: "edge-gw-03.acmecorp.io \u00b7 :443 https", detail: "a (Name, Service) pair — the only key under which HTTP identity is single-valued" },
    { label: "Name leg", value: "edge-gw-03.acmecorp.io" },
    { label: "Service leg", value: "203.0.113.7:443/tcp" },
    { label: "Seed · name scope", value: "acmecorp.io", detail: "declared 2026-06-14 · custody verified" },
  ],
};

function CitationChain({ items }) {
  return (
    <ol style={{ listStyle: "none", margin: 0, padding: 0 }}>
      {items.map((c, i) => (
        <li key={c.label} style={{ position: "relative", padding: "0 0 " + (i < items.length - 1 ? 18 : 0) + "px 22px" }}>
          {i < items.length - 1 && <span style={{ position: "absolute", left: 3.5, top: 12, bottom: 2, width: 1, background: "var(--border-default)" }} />}
          <span style={{ position: "absolute", left: 0, top: 5, width: 8, height: 8, borderRadius: 999, background: i === items.length - 1 ? "var(--accent)" : "var(--surface)", border: "1.5px solid " + (i === items.length - 1 ? "var(--accent)" : "var(--border-strong)") }} />
          <span style={{ display: "block", font: "500 10.5px var(--font-mono)", letterSpacing: "0.06em", textTransform: "uppercase", color: "var(--text-muted)" }}>{c.label}</span>
          <span style={{ display: "block", font: "500 13px var(--font-mono)", color: "var(--text-ink)", margin: "2px 0" }}>{c.value}</span>
          {c.detail && <span style={{ display: "block", font: "400 12px var(--font-ui)", color: "var(--text-secondary)" }}>{c.detail}</span>}
        </li>
      ))}
    </ol>
  );
}

const CLOSED = {
  service: [
    { value: "not-reached", opened: "2026-07-14", openedFull: "2026-07-14T06:00Z", closed: "2026-08-22", closedFull: "2026-08-22T14:00Z", ground: "changed" },
    { value: "Gap", opened: "2026-07-02", openedFull: "2026-07-02T06:00Z", closed: "2026-07-14", closedFull: "2026-07-14T06:00Z", ground: "stopped looking" },
  ],
  endpoint: [
    { value: "200 \u00b7 nginx/1.24.0", opened: "2026-06-14", openedFull: "2026-06-14T09:00Z", closed: "2026-08-12", closedFull: "2026-08-12T06:00Z", ground: "changed" },
  ],
};

const RULES = {
  service: [
    { rule: "vnc-exposure", version: 3, sev: "critical", verdict: "fired" },
    { rule: "tls-acceptance", version: 2, sev: "high", verdict: "did not fire" },
  ],
  endpoint: [
    { rule: "admin-panel-reachable", version: 1, sev: "high", verdict: "did not fire" },
    { rule: "verbose-server-header", version: 2, sev: "low", verdict: "fired" },
  ],
};

export function SubjectDetail({ kind = "service", withdrawn = false, onBack, onOpenSignals, onToast }) {
  const svc = kind === "service";
  const key = svc ? "203.0.113.7:5900/tcp" : "edge-gw-03.acmecorp.io \u00b7 :443 https";
  const copyKey = svc ? "203.0.113.7:5900 tcp" : "edge-gw-03.acmecorp.io 203.0.113.7:443 tcp";
  const facetLabel = svc ? "reachability" : "http-identity";
  return (
    <main data-screen-label={svc ? "Service detail" : "Endpoint detail"} style={{ maxWidth: 1440, margin: "0 auto", padding: 32, display: "flex", flexDirection: "column", gap: 20 }}>
      <Breadcrumb items={[{ label: "Inventory", onClick: onBack }, { label: key }]} />
      <header style={{ display: "flex", alignItems: "flex-start", gap: 16, flexWrap: "wrap" }}>
        <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
          <h1 style={{ margin: 0, font: "600 21px var(--font-mono)", letterSpacing: "-0.01em", color: "var(--text-ink)" }}>{key}</h1>
          <div style={{ display: "flex", alignItems: "center", gap: 8, flexWrap: "wrap" }}>
            <Tag>{svc ? "service" : "endpoint"}</Tag>
            {withdrawn ? <WithdrawnMark size="sm" /> : svc ? <ExposureBadge state="exposed" /> : null}
            <span style={{ font: "400 12px var(--font-mono)", color: "var(--text-muted)" }}>{(withdrawn ? "last seen 6d ago" : "seen 4m ago") + " \u00b7 in scope since " + (svc ? "2026-08-22" : "2026-06-14")}</span>
          </div>
        </div>
        <div style={{ marginLeft: "auto", display: "flex", gap: 8 }}>
          <Button variant="secondary" icon={<Icon name="play" size={14} />} disabled={withdrawn}
            onClick={() => onToast && onToast({ tone: "neutral", title: "Rescan queued", description: key + " \u00b7 next batch" })}>Rescan {svc ? "service" : "endpoint"}</Button>
          <DropdownMenu align="end" trigger={<Button variant="secondary" icon={<Icon name="ellipsis" size={14} />} aria-label="More actions" />} items={[
            { label: "Annotate", icon: "pencil" },
            "-",
            { label: svc ? "Descope address" : "Descope name", icon: "circle-off", tone: "danger" },
          ]} />
        </div>
      </header>
      {withdrawn && (
        <Banner tone="neutral" title="Withdrawn by the world">
          {svc
            ? "This service's address has left the estate — no current resolution cites it and no Seed covers it. It names a population of no current member; its timelines are closed and it is reached by its own key."
            : "This endpoint's service has left the estate. An endpoint closes when either leg — its Name or its Service — withdraws; its timelines are closed and it is reached by its own key."}
        </Banner>
      )}
      <div style={{ display: "grid", gridTemplateColumns: "minmax(0, 1fr) 340px", gap: 24, alignItems: "start" }}>
        <div style={{ display: "flex", flexDirection: "column", gap: 24 }}>
          <Card microLabel="Why is this here" title="Citation chain">
            <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
              <span style={{ font: "400 12.5px/1.6 var(--font-ui)", color: "var(--text-secondary)", maxWidth: "62ch" }}>
                Following citations backwards always terminates at a Seed you declared — that is what makes "why is this here" answerable for everything in the estate.
              </span>
              <CitationChain items={CHAIN[kind]} />
            </div>
          </Card>
          <Card microLabel={"Current \u00b7 " + facetLabel} title={svc ? "Reachability" : "HTTP identity"}>
            {svc ? (
              <KeyValueList items={[
                { k: "Address", v: "203.0.113.7" },
                { k: "Port", v: "5900/tcp" },
                { k: "Verdict", v: withdrawn ? "\u2014" : "reached" },
                { k: "Since", v: "2026-08-22T14:00Z" },
              ]} />
            ) : (
              <KeyValueList items={[
                { k: "Status", v: withdrawn ? "\u2014" : "200" },
                { k: "Server", v: "nginx/1.25.0" },
                { k: "Title", v: "Acme edge gateway" },
                { k: "Redirect", v: "\u2014 (recorded, not followed)" },
              ]} />
            )}
          </Card>
          <Card microLabel="Timelines" title="Current and closed timelines" pad={0}>
            <div style={{ padding: "16px 20px 12px", display: "flex", flexDirection: "column", gap: 10 }}>
              <span style={{ font: "500 11px var(--font-mono)", letterSpacing: "0.06em", textTransform: "uppercase", color: "var(--text-muted)" }}>{facetLabel}</span>
              <div style={{ display: "flex", alignItems: "center", gap: 10, flexWrap: "wrap" }}>
                <span style={{ font: "400 12px var(--font-ui)", color: "var(--text-secondary)", width: 64 }}>Current</span>
                {withdrawn ? (
                  <span style={{ font: "400 12.5px var(--font-ui)", color: "var(--text-muted)" }}>Closed — this timeline holds no current value.</span>
                ) : (
                  <React.Fragment>
                    <Badge>{svc ? "reached" : "200 \u00b7 nginx/1.25.0"}</Badge>
                    <span style={{ font: "400 11.5px var(--font-mono)", color: "var(--text-muted)" }}>since {svc ? "2026-08-22T14:00Z" : "2026-08-12T06:00Z"}</span>
                  </React.Fragment>
                )}
              </div>
              {svc && (
                <Banner tone="accent" icon="git-branch">
                  Break at 2026-08-01T06:00Z — spans are not comparable across it. Leaf that moved: <span style={{ fontFamily: "var(--font-mono)" }}>transport</span>. Derived on read, never stored.
                </Banner>
              )}
            </div>
            <Table framed={false} dense columns={[
              { key: "value", label: "Value", mono: true },
              { key: "opened", label: "Opened", mono: true, width: 96, render: (r) => <span title={r.openedFull}>{r.opened}</span> },
              { key: "closed", label: "Closed", mono: true, width: 96, render: (r) => <span title={r.closedFull}>{r.closed}</span> },
              { key: "ground", label: "Ground", width: 148, align: "right", clip: false, render: (r) => <Badge>{r.ground}</Badge> },
            ]} rows={CLOSED[kind]} rowKey="opened" />
          </Card>
          <Card microLabel="Rules" title="Rules over this subject" pad={0}>
            <Table framed={false} dense columns={[
              { key: "rule", label: "Rule", mono: true, render: (r) => <span style={{ display: "inline-flex", alignItems: "center", gap: 8 }}><SeverityBadge level={r.sev} size="sm" /><span style={{ fontFamily: "var(--font-mono)" }}>{r.rule}</span></span> },
              { key: "version", label: "Version", mono: true, width: 90, render: (r) => "v" + r.version },
              { key: "verdict", label: "Verdict", width: 130, render: (r) => r.verdict === "fired" ? <Badge tone="danger">fired</Badge> : <Badge>did not fire</Badge> },
            ]} rows={RULES[kind]} rowKey="rule" onRowClick={onOpenSignals} />
          </Card>
        </div>
        <div style={{ display: "flex", flexDirection: "column", gap: 24 }}>
          <Card microLabel="Provenance" title="How it got here">
            <KeyValueList items={svc ? [
              { k: "Seed", v: "acmecorp.io" },
              { k: "Via", v: "dns sweep \u2192 hot scan" },
              { k: "Vantage", v: "eu-west-1" },
              { k: "First seen", v: "2026-08-22" },
            ] : [
              { k: "Seed", v: "acmecorp.io" },
              { k: "Via", v: "resolution \u00d7 service join" },
              { k: "Vantage", v: "eu-west-1" },
              { k: "First seen", v: "2026-06-14" },
            ]} />
          </Card>
          {svc && !withdrawn && (
            <Card microLabel="Open" title="Signals here" pad={12}>
              <button type="button" onClick={onOpenSignals}
                style={{ display: "flex", alignItems: "center", gap: 10, padding: "10px 8px", width: "100%", background: "transparent", border: "none", borderRadius: 8, cursor: "pointer", textAlign: "left", fontFamily: "var(--font-ui)" }}>
                <SeverityBadge level="critical" size="sm" />
                <span style={{ flex: 1, minWidth: 0, display: "flex", flexDirection: "column", gap: 2 }}>
                  <span style={{ font: "500 12.5px var(--font-ui)", color: "var(--text-ink)" }}>:5900 vnc — no transport encryption</span>
                  <span style={{ font: "400 11px var(--font-mono)", color: "var(--text-muted)" }}>{"VG-2481 \u00b7 raised 4m ago"}</span>
                </span>
                <Icon name="chevron-right" size={14} style={{ color: "var(--text-muted)", flex: "none" }} />
              </button>
            </Card>
          )}
          <Card microLabel={svc ? "Service key" : "Endpoint key"} title="Copy">
            <CopyValue value={copyKey} />
          </Card>
        </div>
      </div>
    </main>
  );
}
