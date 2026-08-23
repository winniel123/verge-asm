import React from "react";
import { Logo } from "@ds/components/media/Logo.jsx";
import { Input } from "@ds/components/forms/Input.jsx";
import { Kbd } from "@ds/components/display/Kbd.jsx";
import { VersionSelect } from "@ds/components/navigation/VersionSelect.jsx";
import { Icon } from "../ds/Icon.jsx";

/*
 * Top navigation bar — ported verbatim from design-system/examples/DocsPage.jsx.
 *
 * SEAM (T4): the version list and the current version arrive as props with PLACEHOLDER
 * defaults; the search box is a placeholder Input. T4 (search + VersionSelect wiring)
 * swaps the DATA and behaviour here — it should not need to touch this markup. Keep the
 * slot shapes:
 *   - versions:  [{ value: string, tag?: "current" | "dev" | string }]
 *   - version:   string (controlled current value; defaults to first tagged "current")
 *   - onVersionChange: (value) => void   (T4 routes to /<version>/<slug>)
 *   - searchPlaceholder / githubHref: strings
 */

const PLACEHOLDER_VERSIONS = [
  { value: "v0.9.2", tag: "current" },
  { value: "v0.9.1" },
  { value: "v0.8.7" },
  { value: "main", tag: "dev" },
];

export default function TopNav({
  versions = PLACEHOLDER_VERSIONS,
  version,
  onVersionChange,
  searchPlaceholder = "Search docs",
  githubHref = "#",
}) {
  const initial =
    version ||
    (versions.find((v) => v.tag === "current") || versions[0] || {}).value;
  const [ver, setVer] = React.useState(initial);
  const handleChange = (v) => {
    setVer(v);
    onVersionChange && onVersionChange(v);
  };
  return (
    <nav
      style={{
        display: "flex",
        alignItems: "center",
        gap: 16,
        height: 56,
        padding: "0 24px",
        background: "var(--surface)",
        borderBottom: "1px solid var(--border-default)",
      }}
    >
      <Logo size={20} wordmarkSize={17} />
      <span
        style={{
          font: "500 13px var(--font-ui)",
          color: "var(--text-secondary)",
          paddingLeft: 12,
          borderLeft: "1px solid var(--border-default)",
        }}
      >
        Docs
      </span>
      <Input
        size="sm"
        mono
        placeholder={searchPlaceholder}
        prefix={<Icon name="search" size={13} />}
        style={{ marginLeft: "auto", width: 260 }}
      />
      <Kbd keys={["mod", "K"]} />
      <VersionSelect value={ver} onChange={handleChange} versions={versions} />
      <a
        href={githubHref}
        style={{
          font: "500 12px var(--font-ui)",
          color: "var(--text-secondary)",
          textDecoration: "none",
        }}
      >
        GitHub
      </a>
    </nav>
  );
}
