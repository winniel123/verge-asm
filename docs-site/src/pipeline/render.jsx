/*
 * PIPELINE STAGE 2 of 3 — render+transform.  OWNER: T2 (#352).
 *
 * Renders one guide's raw markdown into the T0 article column as a React island.
 * Markdown -> HTML happens through `react-markdown` + `remark-gfm` (GFM tables,
 * autolinks, strikethrough). Element -> component mapping wires the guide's prose
 * into the design system (ADR-0109): fenced code -> DS <CodeBlock>, blockquotes ->
 * DS <Callout>. Headings get GitHub-style `id`s from the shared slugger in slug.ts,
 * so the on-page TOC's `#anchor` hrefs resolve.
 *
 * T2 OWNS the link/anchor rewriting step: the `a` renderer below is the single seam
 * where relative `foo.md#frag` cross-links become in-site `/<version>/foo#frag`
 * routes. Today it passes hrefs through unchanged. T2 edits ONLY the `a` renderer
 * (and may add a remark/rehype plugin) — it does not touch source-resolution or
 * nav-build.  See docs-site/PIPELINE.md.
 */
import React from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import GithubSlugger from "github-slugger";
import { CodeBlock } from "@ds/components/display/CodeBlock.jsx";
import { Callout } from "@ds/components/feedback/Callout.jsx";

/* ---- prose styles: ported from the DocsPage sample article (Article.jsx) ---- */
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

/**
 * T2 link/anchor rewrite. Turns an author's relative markdown href into the in-site
 * route for the SAME version, so cross-links survive the source→route mapping and
 * anchor fragments resolve to the heading ids the renderer emits.
 *
 *   using.md                      -> /<version>/using
 *   ./running.md#environment-vars -> /<version>/running#environment-vars
 *   #severity-levels              -> /<version>/<currentSlug>#severity-levels (in-page)
 *   https://example.com           -> unchanged (external)
 *   ../adr/0017-....md, ../../deploy/, *.go, dirs -> unchanged (repo cross-refs,
 *                                    not rendered guide routes — see PIPELINE.md)
 *
 * Fragments are passed through verbatim: guide authors write already-slugified
 * anchors (`#2-upload-a-zone-file`), and check-links.mjs is the gate that proves
 * each fragment matches a real heading id — via the same github-slugger algorithm.
 * The regexes here mirror check-links.mjs so renderer and gate agree on what counts
 * as an intra-guide link.
 */
const INTRA_GUIDE = /^\.?\/?([a-z0-9][a-z0-9-]*)\.md(?:#(.+))?$/i;
function rewriteHref(href, version, currentSlug) {
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
  return { href, intraSite: false }; // mailto:, ../adr/..., ../../deploy/, *.go, dirs
}

/** Flatten a react-markdown children tree to its plain-text content (for slugging). */
function toText(node) {
  if (node == null || node === false) return "";
  if (typeof node === "string" || typeof node === "number") return String(node);
  if (Array.isArray(node)) return node.map(toText).join("");
  if (React.isValidElement(node)) return toText(node.props.children);
  return "";
}

/**
 * The article body island. `markdown` is the guide's raw source (no frontmatter);
 * the string is passed as a prop from the .astro page so this hydrates client-side,
 * which the DS CodeBlock needs for its copy control.
 */
export default function Article({ markdown = "", version = "main", slug = "" }) {
  // One slugger per render, closed over by the heading renderers. react-markdown
  // visits headings in source order, so its de-dup counter stays in step with
  // extractToc() in slug.ts (which slugs every heading in the same order).
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

    // T2 SEAM: relative guide cross-links (`running.md#anchor`) are rewritten into
    // in-site `/<version>/running#anchor` routes for the current version; external
    // http(s) links open in a new tab; everything else passes through untouched.
    a: ({ href, children }) => {
      const { href: nextHref, external } = rewriteHref(href, version, slug);
      return (
        <a href={nextHref} style={S.a} {...(external ? { target: "_blank", rel: "noreferrer noopener" } : {})}>
          {children}
        </a>
      );
    },

    // GFM tables -> lightweight DS-token-styled table (code fences + callouts are
    // the DS components the ticket requires; tables stay as tokenised HTML).
    table: ({ children }) => (
      <div style={{ overflowX: "auto", margin: "16px 0" }}>
        <table style={S.table}>{children}</table>
      </div>
    ),
    th: ({ children }) => <th style={S.th}>{children}</th>,
    td: ({ children }) => <td style={S.td}>{children}</td>,

    // Blockquote -> DS Callout (prose aside).
    blockquote: ({ children }) => <Callout style={{ margin: "18px 0" }}>{children}</Callout>,

    // Fenced code -> DS CodeBlock; inline code -> InlineCode. In react-markdown v9
    // block code is wrapped in <pre>, so `pre` owns the fenced case and `code`
    // handles only the inline remainder.
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
