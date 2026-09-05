import React from "react";
import { Table } from "@ds/components/display/Table.jsx";
import { SeverityBadge } from "@ds/components/display/SeverityBadge.jsx";
import { CodeBlock } from "@ds/components/display/CodeBlock.jsx";
import { Breadcrumb } from "@ds/components/navigation/Breadcrumb.jsx";
import { Callout } from "@ds/components/feedback/Callout.jsx";
import { Accordion } from "@ds/components/display/Accordion.jsx";

/* Placeholder body; the shipped article renders from the guide markdown (ADR-0115). */

function InlineCode({ children }) {
  return (
    <code
      style={{
        font: "400 0.9em var(--font-mono)",
        background: "var(--surface-sunken)",
        border: "1px solid var(--border-default)",
        borderRadius: 6,
        padding: "1px 5px",
        color: "var(--text-body)",
      }}
    >
      {children}
    </code>
  );
}

export default function Article() {
  return (
    <>
      <Breadcrumb items={[{ label: "Docs" }, { label: "Getting started" }, { label: "Quick start" }]} />
      <h1 style={{ margin: "10px 0 0", font: "600 32px/1.15 var(--font-ui)", letterSpacing: "-0.015em", color: "var(--text-ink)" }}>Quick start</h1>
      <p style={{ margin: "16px 0 0", font: "400 15px/1.65 var(--font-ui)", color: "var(--text-body)" }}>Verge runs as a single container. You point it at a domain or CIDR range; it discovers what you expose and starts watching for drift.</p>
      <h2 style={{ margin: "36px 0 0", font: "600 21px var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)" }}>1. Run the container</h2>
      <p style={{ margin: "12px 0 14px", font: "400 15px/1.65 var(--font-ui)", color: "var(--text-body)" }}>The console listens on <InlineCode>:8443</InlineCode> with a self-signed certificate.</p>
      <CodeBlock title="shell" copyText="docker run -d --name verge -p 8443:8443 -v verge-data:/var/lib/verge verge/verge:0.9.2">{"docker run -d --name verge \\\n  -p 8443:8443 \\\n  -v verge-data:/var/lib/verge \\\n  verge/verge:0.9.2"}</CodeBlock>
      <h2 style={{ margin: "36px 0 0", font: "600 21px var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)" }}>2. Add your first seed</h2>
      <p style={{ margin: "12px 0 14px", font: "400 15px/1.65 var(--font-ui)", color: "var(--text-body)" }}>Open the console and use <InlineCode>Add seed</InlineCode>, or the CLI:</p>
      <CodeBlock title="shell">{"verge seeds add acmecorp.io --watch\nverge scan run --profile standard"}</CodeBlock>
      <Callout style={{ margin: "20px 0 0" }}>Scans are passive-first. Active probing starts only after you confirm you own the scope.</Callout>
      <h2 style={{ margin: "36px 0 0", font: "600 21px var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)" }}>Severity levels</h2>
      <p style={{ margin: "12px 0 16px", font: "400 15px/1.65 var(--font-ui)", color: "var(--text-body)" }}>Signals use exactly five levels, ordered. The words never change; write them as shown.</p>
      <Table columns={[
        { key: "level", label: "Level", width: 120, render: (r) => <SeverityBadge level={r.level} size="sm" /> },
        { key: "meaning", label: "Raised when", render: (r) => <span style={{ font: "400 13px/1.5 var(--font-ui)", color: "var(--text-body)", whiteSpace: "normal" }}>{r.meaning}</span> },
      ]} rows={[
        { level: "critical", meaning: "An exposure is exploitable now — act before the next scan." },
        { level: "high", meaning: "A weakness attackers actively look for." },
        { level: "medium", meaning: "A misconfiguration that widens your surface." },
        { level: "low", meaning: "Hygiene: information leaks and loose ends." },
        { level: "info", meaning: "A change worth knowing about, no action implied." },
      ]} />
      <h2 style={{ margin: "36px 0 0", font: "600 21px var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)" }}>Common questions</h2>
      <Accordion style={{ marginTop: 8 }} defaultOpen={["ports"]} items={[
        { id: "ports", title: "Which ports does the standard profile scan?", content: "The top 1,000 TCP ports, plus any port previously seen on the asset. The deep profile adds full TCP and common UDP." },
        { id: "passive", title: "What does passive-first mean?", content: "Discovery starts from certificate transparency, DNS, and public datasets. Active probing begins only after you confirm ownership." },
        { id: "data", title: "Where is scan data stored?", content: "In the verge-data volume on your infrastructure. Nothing leaves your deployment." },
      ]} />
    </>
  );
}
