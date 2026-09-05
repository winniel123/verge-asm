import React from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import GithubSlugger from "github-slugger";
import { CodeBlock } from "@ds/components/display/CodeBlock.jsx";
import { Callout } from "@ds/components/feedback/Callout.jsx";
import { refForDocsVersion, repoBlobUrl } from "../repo.ts";

const S = {
  h1: { margin: "10px 0 0", font: "600 32px/1.15 var(--font-ui)", letterSpacing: "-0.015em", color: "var(--text-ink)" },
  h2: { margin: "36px 0 12px", font: "600 21px var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)", scrollMarginTop: 24 },
  h3: { margin: "28px 0 8px", font: "600 17px var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)", scrollMarginTop: 24 },
  h4: { margin: "22px 0 6px", font: "600 15px var(--font-ui)", color: "var(--text-ink)", scrollMarginTop: 24 },
  p: { margin: "14px 0", font: "400 15px/1.65 var(--font-ui)", color: "var(--text-body)" },
  ul: { margin: "12px 0", paddingLeft: 22, font: "400 15px/1.65 var(--font-ui)", color: "var(--text-body)" },
  ol: { margin: "12px 0", paddingLeft: 22, font: "400 15px/1.65 var(--font-ui)", color: "var(--text-body)" },
  li: { margin: "4px 0" },
  a: { color: "var(--link)", textDecoration: "none" },
  hr: { margin: "32px 0", border: 0, borderTop: "1px solid var(--border-default)" },
  table: { width: "100%", borderCollapse: "collapse", margin: "16px 0", font: "400 13px/1.5 var(--font-ui)", color: "var(--text-body)" },
  th: { textAlign: "left", padding: "8px 12px", borderBottom: "1px solid var(--border-default)", font: "600 12px var(--font-ui)", color: "var(--text-ink)", whiteSpace: "nowrap" },
  td: { padding: "8px 12px", borderBottom: "1px solid var(--border-subtle, var(--border-default))", verticalAlign: "top" },
};

function InlineCode({ children }) {
  return (
    <code style={{ font: "400 0.9em var(--font-mono)", background: "var(--surface-sunken)", border: "1px solid var(--border-default)", borderRadius: 6, padding: "1px 5px", color: "var(--text-body)" }}>
      {children}
    </code>
  );
}

const INTRA_GUIDE = /^\.?\/?([a-z0-9][a-z0-9-]*)\.md(?:#(.+))?$/i;
// Forbidding a slash keeps this in lockstep with check-links.mjs, which gates the same targets.
const ADR_XREF = /^\.\.\/adr\/([^#?/]+\.md)(?:#(.+))?$/i;
function rewriteHref(href, version, currentSlug, adrRef) {
  if (!href) return { href, intraSite: false };
  if (href.startsWith("#")) {
    return { href: `/${version}/${currentSlug}${href}`, intraSite: true };
  }
  if (/^https?:\/\//i.test(href)) return { href, intraSite: false, external: true };
  const m = INTRA_GUIDE.exec(href);
  if (m) {
    const frag = m[2] ? `#${m[2]}` : "";
    return { href: `/${version}/${m[1]}${frag}`, intraSite: true };
  }
  const adr = ADR_XREF.exec(href);
  if (adr) {
    const frag = adr[2] ? `#${adr[2]}` : "";
    // An ADR is never an ingested page, so a relative link to one 404s on the docs server (#428).
    const blob = repoBlobUrl(adrRef, `docs/adr/${adr[1]}`);
    return { href: `${blob}${frag}`, intraSite: false, external: true };
  }
  return { href, intraSite: false };
}

function toText(node) {
  if (node == null || node === false) return "";
  if (typeof node === "string" || typeof node === "number") return String(node);
  if (Array.isArray(node)) return node.map(toText).join("");
  if (React.isValidElement(node)) return toText(node.props.children);
  return "";
}

// The DS CodeBlock's copy control needs hydration, so the guide arrives as a prop.
export default function Article({ markdown = "", version = "main", slug = "", adrRef }) {
  // source-resolution.ts is node-only, so an island takes the ref as a prop, never imports it.
  const ref = adrRef ?? refForDocsVersion(version);
  const slugger = new GithubSlugger();
  const heading = (Tag, style) =>
    function H({ children }) {
      const id = slugger.slug(toText(children));
      return (
        <Tag id={id} style={style}>
          {children}
        </Tag>
      );
    };

  const components = {
    h1: heading("h1", S.h1),
    h2: heading("h2", S.h2),
    h3: heading("h3", S.h3),
    h4: heading("h4", S.h4),
    p: ({ children }) => <p style={S.p}>{children}</p>,
    ul: ({ children }) => <ul style={S.ul}>{children}</ul>,
    ol: ({ children }) => <ol style={S.ol}>{children}</ol>,
    li: ({ children }) => <li style={S.li}>{children}</li>,
    hr: () => <hr style={S.hr} />,
    strong: ({ children }) => <strong style={{ fontWeight: 600, color: "var(--text-ink)" }}>{children}</strong>,
    em: ({ children }) => <em>{children}</em>,

    a: ({ href, children }) => {
      const { href: nextHref, external } = rewriteHref(href, version, slug, ref);
      return (
        <a href={nextHref} style={S.a} {...(external ? { target: "_blank", rel: "noreferrer noopener" } : {})}>
          {children}
        </a>
      );
    },

    table: ({ children }) => (
      <div style={{ overflowX: "auto", margin: "16px 0" }}>
        <table style={S.table}>{children}</table>
      </div>
    ),
    th: ({ children }) => <th style={S.th}>{children}</th>,
    td: ({ children }) => <td style={S.td}>{children}</td>,

    blockquote: ({ children }) => <Callout style={{ margin: "18px 0" }}>{children}</Callout>,

    pre: ({ children }) => {
      const codeEl = React.Children.toArray(children).find((c) => React.isValidElement(c));
      const className = codeEl?.props?.className || "";
      const lang = /language-([\w-]+)/.exec(className)?.[1];
      const text = toText(codeEl?.props?.children).replace(/\n$/, "");
      return <CodeBlock title={lang} style={{ margin: "16px 0" }}>{text}</CodeBlock>;
    },
    code: ({ children }) => <InlineCode>{children}</InlineCode>,
  };

  return (
    <ReactMarkdown remarkPlugins={[remarkGfm]} components={components}>
      {markdown}
    </ReactMarkdown>
  );
}
