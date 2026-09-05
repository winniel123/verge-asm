import React from "react";
import { Logo } from "@ds/components/media/Logo.jsx";
import { Input } from "@ds/components/forms/Input.jsx";
import { Kbd } from "@ds/components/display/Kbd.jsx";
import { VersionSelect } from "@ds/components/navigation/VersionSelect.jsx";
import { CommandPalette } from "@ds/components/feedback/CommandPalette.jsx";
import { Icon } from "../ds/Icon.jsx";
import { REPO_URL } from "../repo.ts";

const FALLBACK_VERSIONS = [
  { value: "latest", tag: "current" },
  { value: "main", tag: "dev" },
];

function currentSlugFromPath() {
  if (typeof window === "undefined") return null;
  const segs = window.location.pathname.split("/").filter(Boolean);
  return segs.length >= 2 ? decodeURIComponent(segs[1]) : null;
}

export default function TopNav({
  versions = FALLBACK_VERSIONS,
  version,
  onVersionChange,
  searchPlaceholder = "Search docs",
  githubHref = REPO_URL,
}) {
  const initial =
    version ||
    (versions.find((v) => v.tag === "current") || versions[0] || {}).value;
  const [ver, setVer] = React.useState(initial);
  const [open, setOpen] = React.useState(false);
  const [docs, setDocs] = React.useState([]);

  const cache = React.useRef(new Map());
  const fetchIndex = React.useCallback(async (v) => {
    if (cache.current.has(v)) return cache.current.get(v);
    try {
      const res = await fetch(`/search/${encodeURIComponent(v)}.json`);
      const data = res.ok ? await res.json() : [];
      cache.current.set(v, data);
      return data;
    } catch {
      cache.current.set(v, []);
      return [];
    }
  }, []);

  const loadActive = React.useCallback(async () => {
    const data = await fetchIndex(ver);
    setDocs(data);
  }, [fetchIndex, ver]);

  const openPalette = React.useCallback(() => {
    setOpen(true);
    loadActive();
  }, [loadActive]);

  // the browser binds ⌘K and Ctrl-K itself
  React.useEffect(() => {
    const onKey = (e) => {
      if ((e.metaKey || e.ctrlKey) && (e.key === "k" || e.key === "K")) {
        e.preventDefault();
        setOpen((v) => {
          if (!v) loadActive();
          return !v;
        });
      }
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [loadActive]);

  // there is no bare /<version>/ index route, so a switch has to land on a guide slug
  const handleChange = async (v) => {
    setVer(v);
    onVersionChange && onVersionChange(v);
    const target = await fetchIndex(v);
    const slugs = new Set(target.map((d) => d.slug));
    const cur = currentSlugFromPath();
    const dest = cur && slugs.has(cur) ? cur : target[0] && target[0].slug;
    if (dest) window.location.assign(`/${encodeURIComponent(v)}/${dest}`);
  };

  // the palette filters over label plus hint alone, so a row carries both title and heading
  const groups = React.useMemo(
    () => [
      {
        label: `Docs · ${ver}`,
        items: docs.map((d) => ({
          id: d.href,
          label: d.heading ? `${d.guideTitle} › ${d.heading}` : d.guideTitle,
          icon: d.level === 0 ? "file-text" : "chevron-right",
          hint: d.slug,
          onSelect: () => window.location.assign(d.href),
        })),
      },
    ],
    [docs, ver],
  );

  return (
    <>
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
        <div
          onClick={openPalette}
          style={{ marginLeft: "auto", cursor: "text" }}
        >
          <Input
            size="sm"
            mono
            readOnly
            placeholder={searchPlaceholder}
            prefix={<Icon name="search" size={13} />}
            onFocus={openPalette}
            style={{ width: 260, cursor: "text" }}
          />
        </div>
        <Kbd keys={["mod", "K"]} />
        <VersionSelect value={ver} onChange={handleChange} versions={versions} />
        <a
          href={githubHref}
          target="_blank"
          rel="noreferrer noopener"
          style={{
            font: "500 12px var(--font-ui)",
            color: "var(--text-secondary)",
            textDecoration: "none",
          }}
        >
          GitHub
        </a>
      </nav>
      <CommandPalette
        open={open}
        onClose={() => setOpen(false)}
        groups={groups}
        placeholder={`Search ${ver} docs…`}
      />
    </>
  );
}
