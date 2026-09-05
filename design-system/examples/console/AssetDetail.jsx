import React from "react";
import { Breadcrumb } from "../../components/navigation/Breadcrumb.jsx";
import { Card } from "../../components/display/Card.jsx";
import { Table } from "../../components/display/Table.jsx";
import { Tag } from "../../components/display/Tag.jsx";
import { SeverityBadge } from "../../components/display/SeverityBadge.jsx";
import { ExposureBadge } from "../../components/display/ExposureBadge.jsx";
import { KeyValueList } from "../../components/display/KeyValueList.jsx";
import { CertificateCard } from "../../components/display/CertificateCard.jsx";
import { Timeline } from "../../components/display/Timeline.jsx";
import { ChangeBadge } from "../../components/display/ChangeBadge.jsx";
import { CopyValue } from "../../components/display/CopyValue.jsx";
import { Button } from "../../components/forms/Button.jsx";
import { DropdownMenu } from "../../components/feedback/DropdownMenu.jsx";
import { Icon } from "../../components/media/Icon.jsx";

const ev = (change, subject, detail, time) => ({ change, time, mono: true,
  title: (
    <span style={{ display: "flex", alignItems: "center", gap: 8, flexWrap: "wrap", rowGap: 4 }}>
      <ChangeBadge change={change} size="sm" />
      <span style={{ font: "500 12.5px var(--font-mono)", color: "var(--text-ink)" }}>{subject}</span>
    </span>
  ), detail });

export function AssetDetail({ asset = "edge-gw-03.acmecorp.io", onBack, onOpenSignals, onToast }) {
  return (
    <main data-screen-label="Asset detail" style={{ maxWidth: 1440, margin: "0 auto", padding: 32, display: "flex", flexDirection: "column", gap: 20 }}>
      <Breadcrumb items={[{ label: "Inventory", onClick: onBack }, { label: asset }]} />
      <header style={{ display: "flex", alignItems: "flex-start", gap: 16, flexWrap: "wrap" }}>
        <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
          <h1 style={{ margin: 0, font: "600 21px var(--font-mono)", letterSpacing: "-0.01em", color: "var(--text-ink)" }}>{asset}</h1>
          <div style={{ display: "flex", alignItems: "center", gap: 8, flexWrap: "wrap" }}>
            <Tag>subdomain</Tag>
            <SeverityBadge level="critical" size="sm" />
            <ExposureBadge state="exposed" />
            <span style={{ font: "400 12px var(--font-mono)", color: "var(--text-muted)" }}>seen 4m ago · in scope since 2026-06-14</span>
          </div>
        </div>
        <div style={{ marginLeft: "auto", display: "flex", gap: 8 }}>
          <Button variant="secondary" icon={<Icon name="play" size={14} />} onClick={() => onToast && onToast({ tone: "neutral", title: "Rescan queued", description: asset + " · next batch" })}>Rescan asset</Button>
          <DropdownMenu align="end" trigger={<Button variant="secondary" icon={<Icon name="ellipsis" size={14} />} aria-label="More actions" />} items={[
            { label: "Annotate", icon: "pencil" },
            { label: "Add tag", icon: "tag" },
            "-",
            { label: "Descope asset", icon: "circle-off", tone: "danger" },
          ]} />
        </div>
      </header>
      <div style={{ display: "grid", gridTemplateColumns: "minmax(0, 1fr) 340px", gap: 24, alignItems: "start" }}>
        <div style={{ display: "flex", flexDirection: "column", gap: 24 }}>
          <Card microLabel="Census" title="Open ports" pad={0}>
            <Table framed={false} dense columns={[
              { key: "port", label: "Port", mono: true, width: 90 },
              { key: "svc", label: "Service", mono: true },
              { key: "exp", label: "Exposure", width: 150, render: (r) => <ExposureBadge state={r.exp} /> },
              { key: "seen", label: "First seen", mono: true, align: "right", width: 110 },
            ]} rows={[
              { port: ":443", svc: "https · nginx/1.25.0", exp: "exposed", seen: "2026-06-14" },
              { port: ":5900", svc: "vnc — no transport encryption", exp: "exposed", seen: "2026-08-22" },
              { port: ":22", svc: "ssh · OpenSSH 9.6", exp: "firewalled", seen: "2026-06-14" },
            ]} rowKey="port" />
          </Card>
          <Card microLabel="Resolution" title="DNS records" pad={0}>
            <Table framed={false} dense columns={[
              { key: "type", label: "Type", mono: true, width: 90 },
              { key: "value", label: "Value", mono: true },
              { key: "seen", label: "Seen", mono: true, align: "right", width: 80 },
            ]} rows={[
              { type: "A", value: "203.0.113.7", seen: "4m" },
              { type: "AAAA", value: "2001:db8::7", seen: "4m" },
              { type: "TXT", value: "verge-custody=vg_7f2a91c4", seen: "6h" },
            ]} rowKey="type" />
          </Card>
          <CertificateCard name="edge-gw-03.acmecorp.io" issuer="CN=R11, O=Let's Encrypt" algorithm="ECDSA-SHA256" notAfter="2026-10-08" daysLeft={47}
            fingerprint="SHA256:2b:9e:44:a1:7c:03:d8:f2:61:5b:c9:10:8e:af:72:d4" />
        </div>
        <div style={{ display: "flex", flexDirection: "column", gap: 24 }}>
          <Card microLabel="Provenance" title="How it got here">
            <KeyValueList items={[
              { k: "Seed", v: "acmecorp.io" },
              { k: "Via", v: "CT log → dns sweep" },
              { k: "Vantage", v: "eu-west-1" },
              { k: "Custody", v: "verified · TXT record" },
              { k: "First seen", v: "2026-06-14" },
            ]} />
          </Card>
          <Card microLabel="Open" title="Signals here" pad={12}>
            <div style={{ display: "flex", flexDirection: "column" }}>
              {[{ id: "VG-2481", sev: "critical", text: ":5900 vnc — no transport encryption", time: "4m" }].map((s) => (
                <button key={s.id} type="button" onClick={onOpenSignals}
                  style={{ display: "flex", alignItems: "center", gap: 10, padding: "10px 8px", background: "transparent", border: "none", borderRadius: 8, cursor: "pointer", textAlign: "left", fontFamily: "var(--font-ui)" }}>
                  <SeverityBadge level={s.sev} size="sm" />
                  <span style={{ flex: 1, minWidth: 0, display: "flex", flexDirection: "column", gap: 2 }}>
                    <span style={{ font: "500 12.5px var(--font-ui)", color: "var(--text-ink)" }}>{s.text}</span>
                    <span style={{ font: "400 11px var(--font-mono)", color: "var(--text-muted)" }}>{s.id} · raised {s.time} ago</span>
                  </span>
                  <Icon name="chevron-right" size={14} style={{ color: "var(--text-muted)", flex: "none" }} />
                </button>
              ))}
            </div>
          </Card>
          <Card microLabel="History" title="Drift trail" pad={20}>
            <Timeline groups={[{ id: "t", label: "This asset", meta: "last 30 days", events: [
              ev("changed", ":443 service banner", "nginx/1.24.0 → 1.25.0", "4m"),
              ev("appeared", ":5900 vnc", "service · new in batch 14:00Z", "4m"),
              ev("changed", "certificate", "renewed · Let's Encrypt R11", "12d"),
              ev("appeared", asset, "name · first seen via certificate transparency", "69d"),
            ] }]} />
          </Card>
          <Card microLabel="Address" title="Copy">
            <CopyValue value={asset} />
          </Card>
        </div>
      </div>
    </main>
  );
}
