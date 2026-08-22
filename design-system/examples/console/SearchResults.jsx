import React from "react";
import { Card } from "../../components/display/Card.jsx";
import { Tag } from "../../components/display/Tag.jsx";
import { SeverityBadge } from "../../components/display/SeverityBadge.jsx";
import { BatchStatus } from "../../components/display/BatchStatus.jsx";
import { EmptyState } from "../../components/feedback/EmptyState.jsx";
import { Input } from "../../components/forms/Input.jsx";
import { Icon } from "../../components/media/Icon.jsx";

const ASSETS = [
  { name: "acmecorp.io", type: "domain", sev: "medium" },
  { name: "api.acmecorp.io", type: "subdomain", sev: "high" },
  { name: "edge-gw-03.acmecorp.io", type: "subdomain", sev: "critical" },
  { name: "mail.acmecorp.io", type: "subdomain", sev: null },
];
const SIGNALS = [
  { sev: "critical", text: ":5900 vnc — no transport encryption", asset: "edge-gw-03.acmecorp.io" },
  { sev: "high", text: "nginx 1.25.0 · CVE-2026-1187", asset: "api.acmecorp.io" },
];
const BATCHES = [
  { id: "2026-08-22T14:00Z", status: "complete" },
  { id: "2026-08-21T20:00Z", status: "failed" },
];
const DOCS = [
  { title: "Seeds and scope", snip: "A seed declares intent; discovery expands it into subjects." },
  { title: "Reading exposure states", snip: "exposed · firewalled · not-reached — states, never a score." },
];

/* Full-page search — where ⌘K's "see everything" lands. */
export function SearchResults({ initialQuery = "acme", onOpenAsset, onNavigate }) {
  const [q, setQ] = React.useState(initialQuery);
  const t = (s) => s.toLowerCase().includes(q.toLowerCase());
  const assets = q ? ASSETS.filter((a) => t(a.name)) : ASSETS;
  const signals = q ? SIGNALS.filter((s) => t(s.text) || t(s.asset)) : SIGNALS;
  const batches = q ? BATCHES.filter((b) => t(b.id)) : BATCHES;
  const docs = q ? DOCS.filter((d) => t(d.title) || t(d.snip)) : DOCS;
  const total = assets.length + signals.length + batches.length + docs.length;
  const hi = (text) => {
    const i = q ? text.toLowerCase().indexOf(q.toLowerCase()) : -1;
    if (i < 0) return text;
    return (
      <React.Fragment>{text.slice(0, i)}<span style={{ color: "var(--link)", fontWeight: 600 }}>{text.slice(i, i + q.length)}</span>{text.slice(i + q.length)}</React.Fragment>
    );
  };
  const row = { display: "flex", alignItems: "center", gap: 10, width: "100%", padding: "10px 8px", background: "transparent", border: "none", borderRadius: 8, cursor: "pointer", textAlign: "left", fontFamily: "var(--font-ui)" };
  return (
    <main data-screen-label="Search results" style={{ maxWidth: 900, margin: "0 auto", padding: 32, display: "flex", flexDirection: "column", gap: 20 }}>
      <header style={{ display: "flex", flexDirection: "column", gap: 12 }}>
        <h1 style={{ margin: 0, font: "600 21px var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)" }}>Search</h1>
        <Input mono value={q} placeholder="Assets, signals, batches, docs" onChange={(e) => setQ(e.target.value)} autoFocus spellCheck={false} />
        <span style={{ font: "400 12px var(--font-mono)", color: "var(--text-muted)" }}>{total} results{q ? " for “" + q + "”" : ""}</span>
      </header>
      {total === 0 && <EmptyState icon="search" message="Nothing matches." detail="Try a hostname fragment, a signal phrase, or a batch timestamp." style={{ padding: "48px 0" }} />}
      {assets.length > 0 && (
        <Card microLabel={assets.length + " match" + (assets.length === 1 ? "" : "es")} title="Assets" pad={10}>
          {assets.map((a) => (
            <button key={a.name} type="button" style={row} onClick={() => onOpenAsset && onOpenAsset(a.name)}>
              <Icon name="server" size={14} style={{ color: "var(--text-muted)", flex: "none" }} />
              <span style={{ font: "500 12.5px var(--font-mono)", color: "var(--text-ink)" }}>{hi(a.name)}</span>
              <Tag>{a.type}</Tag>
              {a.sev && <SeverityBadge level={a.sev} size="sm" />}
              <Icon name="chevron-right" size={14} style={{ marginLeft: "auto", color: "var(--text-muted)", flex: "none" }} />
            </button>
          ))}
        </Card>
      )}
      {signals.length > 0 && (
        <Card microLabel={signals.length + " open"} title="Signals" pad={10}>
          {signals.map((s) => (
            <button key={s.text} type="button" style={row} onClick={() => onNavigate && onNavigate("signals")}>
              <SeverityBadge level={s.sev} size="sm" />
              <span style={{ font: "500 12.5px var(--font-ui)", color: "var(--text-ink)" }}>{hi(s.text)}</span>
              <span style={{ marginLeft: "auto", font: "400 11.5px var(--font-mono)", color: "var(--text-muted)" }}>{hi(s.asset)}</span>
            </button>
          ))}
        </Card>
      )}
      {batches.length > 0 && (
        <Card microLabel={batches.length + " recent"} title="Batches" pad={10}>
          {batches.map((b) => (
            <button key={b.id} type="button" style={row} onClick={() => onNavigate && onNavigate("run")}>
              <BatchStatus status={b.status} scope={b.id} />
              <Icon name="chevron-right" size={14} style={{ marginLeft: "auto", color: "var(--text-muted)", flex: "none" }} />
            </button>
          ))}
        </Card>
      )}
      {docs.length > 0 && (
        <Card microLabel="Docs" title="Documentation" pad={10}>
          {docs.map((d) => (
            <button key={d.title} type="button" style={row}>
              <Icon name="file-text" size={14} style={{ color: "var(--text-muted)", flex: "none" }} />
              <span style={{ display: "flex", flexDirection: "column", gap: 2, minWidth: 0 }}>
                <span style={{ font: "500 12.5px var(--font-ui)", color: "var(--text-ink)" }}>{hi(d.title)}</span>
                <span style={{ font: "400 11.5px var(--font-ui)", color: "var(--text-muted)" }}>{hi(d.snip)}</span>
              </span>
            </button>
          ))}
        </Card>
      )}
    </main>
  );
}
