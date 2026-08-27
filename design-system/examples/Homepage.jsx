import React from "react";
import { Button } from "../../components/forms/Button.jsx";
import { Card } from "../../components/display/Card.jsx";
import { Stat } from "../../components/display/Stat.jsx";
import { Table } from "../../components/display/Table.jsx";
import { SeverityBadge } from "../../components/display/SeverityBadge.jsx";
import { Toast } from "../../components/feedback/Toast.jsx";
import { Footer } from "../../components/navigation/Footer.jsx";
import { Stepper } from "../../components/display/Stepper.jsx";
import { Icon } from "../../components/media/Icon.jsx";
import { CodeBlock } from "../../components/display/CodeBlock.jsx";
import { Logo } from "../../components/media/Logo.jsx";

const ROWS = [
  { sev: "critical", title: "VNC exposed to internet", asset: "edge-gw-03.acmecorp.io", seen: "4m" },
  { sev: "high", title: "Admin panel reachable", asset: "grafana.acmecorp.io", seen: "26m" },
  { sev: "medium", title: "SPF record missing", asset: "acmecorp.io", seen: "9h" },
  { sev: "info", title: "New subdomain discovered", asset: "staging-4.acmecorp.io", seen: "3d" },
];

const REPO = "https://github.com/winniel123/verge-asm";
const scrollToId = (id) => {
  const el = document.getElementById(id);
  if (el) window.scrollTo({ top: el.getBoundingClientRect().top + window.scrollY - 24, behavior: "smooth" });
};

function MarketingNav() {
  return (
    <nav style={{ display: "flex", alignItems: "center", gap: 24, height: 64, padding: "0 32px", maxWidth: 1184, margin: "0 auto" }}>
      <Logo size={22} wordmarkSize={19} />
      <span style={{ display: "flex", gap: 20, marginLeft: 16 }}>
        <a href="../docs/index.html" style={{ font: "500 13.5px var(--font-ui)", color: "var(--text-secondary)", textDecoration: "none" }}>Docs</a>
        <a href="#install" onClick={(e) => { e.preventDefault(); scrollToId("install"); }} style={{ font: "500 13.5px var(--font-ui)", color: "var(--text-secondary)", textDecoration: "none" }}>Install</a>
        <a href={REPO + "/releases"} target="_blank" rel="noreferrer noopener" style={{ font: "500 13.5px var(--font-ui)", color: "var(--text-secondary)", textDecoration: "none" }}>Changelog</a>
      </span>
      <span style={{ marginLeft: "auto", display: "flex", gap: 10, alignItems: "center" }}>
        <Button variant="secondary" onClick={() => window.open(REPO, "_blank", "noopener")}>GitHub</Button>
        <Button onClick={() => scrollToId("install")}>Get started</Button>
      </span>
    </nav>
  );
}

function Feature({ icon, title, children }) {
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 12, padding: 28, background: "var(--surface)", border: "1px solid var(--border-default)", borderRadius: 16, boxShadow: "var(--shadow-sm)" }}>
      <span style={{ display: "inline-flex", alignItems: "center", justifyContent: "center", width: 40, height: 40, borderRadius: 12, background: "var(--accent-softer)", border: "1px solid var(--accent-soft)", color: "var(--accent)" }}>
        <Icon name={icon} size={18} strokeWidth={1.6} />
      </span>
      <span style={{ font: "600 16px var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)" }}>{title}</span>
      <span style={{ font: "400 13.5px/1.6 var(--font-ui)", color: "var(--text-secondary)" }}>{children}</span>
    </div>
  );
}

export function Homepage() {
  return (
    <div data-screen-label="Marketing homepage" style={{ background: "var(--bg-page)", minHeight: "100vh" }}>
      <div style={{ background: "var(--surface)", borderBottom: "1px solid var(--border-default)" }}><MarketingNav /></div>
      <section style={{ maxWidth: 1120, margin: "0 auto", padding: "88px 32px 72px", display: "grid", gridTemplateColumns: "1.05fr 1fr", gap: 64, alignItems: "center" }}>
        <div style={{ display: "flex", flexDirection: "column", gap: 22 }}>
          <span style={{ font: "500 11px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--accent)" }}>Open-source attack surface management</span>
          <h1 style={{ margin: 0, font: "600 52px/1.08 var(--font-ui)", letterSpacing: "-0.02em", color: "var(--text-ink)", textWrap: "balance" }}>Your attack surface, continuously mapped</h1>
          <p style={{ margin: 0, font: "400 17px/1.6 var(--font-ui)", color: "var(--text-secondary)", maxWidth: 460 }}>Verge discovers your internet-facing assets, watches them for drift, and raises signals when something changes. Free, AGPL-3.0, runs on your infrastructure.</p>
          <div style={{ display: "flex", gap: 12, marginTop: 6 }}>
            <Button size="lg" icon={<Icon name="arrow-right" size={16} />} onClick={() => scrollToId("install")}>Get started</Button>
            <Button size="lg" variant="secondary" icon={<Icon name="code" size={16} />} onClick={() => window.open(REPO, "_blank", "noopener")}>View source</Button>
          </div>
          <CodeBlock style={{ width: "fit-content", marginTop: 8, paddingRight: 56 }} copyText="docker run -p 8443:8443 verge/verge">docker run -p 8443:8443 verge/verge</CodeBlock>
          <span style={{ font: "400 11.5px var(--font-mono)", color: "var(--text-muted)" }}>v0.9.2 · AGPL-3.0 · multi-org for MSPs</span>
        </div>
        <div style={{ position: "relative", minWidth: 0 }}>
          <Card pad={0} style={{ boxShadow: "var(--shadow-md)" }}>
            <div style={{ display: "flex", gap: 28, padding: "18px 20px", borderBottom: "1px solid var(--row-sep)" }}>
              <Stat label="Open signals" value="47" delta="+3" deltaTone="bad" live />
              <Stat label="Assets" value="1,284" delta="+12" />
              <Stat label="Critical" value="3" delta="−1" deltaTone="good" />
            </div>
            <Table framed={false} dense columns={[
              { key: "sev", label: "Severity", width: 96, render: (r) => <SeverityBadge level={r.sev} size="sm" /> },
              { key: "title", label: "Signal", render: (r) => <span style={{ font: "500 12.5px var(--font-ui)", color: "var(--text-ink)" }}>{r.title}</span> },
              { key: "asset", label: "Asset", mono: true },
              { key: "seen", label: "Seen", mono: true, align: "right", width: 56 },
            ]} rows={ROWS} />
          </Card>
          <Toast tone="ok" title="Scan complete" description="3 new signals raised." style={{ position: "absolute", right: -16, bottom: -24, boxShadow: "var(--shadow-lg)" }} />
          <a href="../console/index.html" style={{ position: "absolute", left: 4, bottom: -34, font: "500 12px var(--font-ui)", color: "var(--link)", textDecoration: "none", display: "inline-flex", alignItems: "center", gap: 5 }}>Explore the console <Icon name="arrow-right" size={13} /></a>
        </div>
      </section>
      <section style={{ maxWidth: 1120, margin: "0 auto", padding: "0 32px 72px", display: "grid", gridTemplateColumns: "repeat(3, 1fr)", gap: 20 }}>
        <Feature icon="radar" title="Discovery that keeps looking">Domains, subdomains, IPs, ports, services, TLS certs — found passively and actively, then watched.</Feature>
        <Feature icon="git-branch" title="Drift, not noise">Every scan is compared to the last known state. You hear about changes, not inventories.</Feature>
        <Feature icon="shield-alert" title="Signals you can rank">Five severities, Critical → Info. Critical reads as critical — even in a dense table.</Feature>
      </section>
      <section style={{ maxWidth: 1120, margin: "0 auto", padding: "0 32px 72px", display: "grid", gridTemplateColumns: "1fr 1.2fr", gap: 64, alignItems: "center" }}>
        <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
          <span style={{ font: "500 11px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--accent)" }}>How it works</span>
          <h2 style={{ margin: 0, font: "600 30px/1.2 var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)" }}>Four steps to a watched surface</h2>
          <p style={{ margin: 0, font: "400 15px/1.6 var(--font-ui)", color: "var(--text-secondary)" }}>Declare what's yours; Verge finds the rest and tells you when it moves.</p>
        </div>
        <Stepper active={1} steps={[
          { title: "Run the container", detail: "One image, listens on :8443" },
          { title: "Add your first seed", detail: "A name or an address scope" },
          { title: "Confirm ownership", detail: "Custody via DNS TXT record — active probing waits for it" },
          { title: "Watch for drift", detail: "Signals when the surface moves" },
        ]} />
      </section>
      <section style={{ maxWidth: 1120, margin: "0 auto", padding: "0 32px 72px" }}>
        <div style={{ display: "flex", alignItems: "center", gap: 20, padding: "24px 28px", background: "var(--surface)", border: "1px solid var(--border-default)", borderRadius: 16, flexWrap: "wrap" }}>
          <span style={{ font: "500 13.5px var(--font-ui)", color: "var(--text-body)" }}>Severity you can read at a glance</span>
          <span style={{ display: "flex", gap: 8, marginLeft: "auto", flexWrap: "wrap" }}>
            {["critical", "high", "medium", "low", "info"].map((l) => <SeverityBadge key={l} level={l} />)}
          </span>
        </div>
      </section>
      <section id="install" style={{ maxWidth: 1120, margin: "0 auto", padding: "0 32px 72px", display: "grid", gridTemplateColumns: "1fr 1.2fr", gap: 64, alignItems: "center" }}>
        <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
          <span style={{ font: "500 11px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--accent)" }}>Install</span>
          <h2 style={{ margin: 0, font: "600 30px/1.2 var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)" }}>Up in one compose file</h2>
          <p style={{ margin: 0, font: "400 15px/1.6 var(--font-ui)", color: "var(--text-secondary)" }}>Docker with the Compose plugin is the whole prerequisite list. Migrations run on startup; the setup token prints in the logs.</p>
          <a href="../docs/index.html" style={{ font: "500 13px var(--font-ui)", color: "var(--link)", textDecoration: "none", display: "inline-flex", alignItems: "center", gap: 5 }}>Running verge-asm — the full guide <Icon name="arrow-right" size={13} /></a>
        </div>
        <CodeBlock title="shell" copyText={"cp .env.example .env\ndocker compose up -d --build\ndocker compose logs web | grep /setup"}>{"cp .env.example .env        # set POSTGRES_PASSWORD\ndocker compose up -d --build\ndocker compose logs web | grep /setup"}</CodeBlock>
      </section>
      <section style={{ maxWidth: 1120, margin: "0 auto", padding: "0 32px 88px" }}>
        <div style={{ background: "var(--surface-inverted)", borderRadius: 24, padding: "56px 48px", display: "flex", alignItems: "center", gap: 32, flexWrap: "wrap" }}>
          <div style={{ display: "flex", flexDirection: "column", gap: 10, maxWidth: 520 }}>
            <h2 style={{ margin: 0, font: "600 30px/1.2 var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-on-inverted)" }}>Own your scan data</h2>
            <p style={{ margin: 0, font: "400 15px/1.6 var(--font-ui)", color: "var(--neutral-400)" }}>Verge is self-hosted and AGPL-3.0. Clone it, audit it, run it where your assets live.</p>
          </div>
          <div style={{ marginLeft: "auto", display: "flex", gap: 12 }}>
            <Button size="lg" onClick={() => scrollToId("install")}>Get started</Button>
            <Button size="lg" variant="ghost" style={{ color: "var(--neutral-300)" }} onClick={() => { window.location.href = "../docs/index.html"; }}>Read the docs</Button>
          </div>
        </div>
      </section>
      <Footer variant="marketing" />
    </div>
  );
}
