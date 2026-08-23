import React from "react";
import {
  // shell + prose icons in use today
  Search,
  ChevronDown,
  ChevronRight,
  Check,
  Info,
  CheckCircle2,
  AlertTriangle,
  Copy,
  X,
  Menu,
  ExternalLink,
  // design-system icon vocabulary (README) — ready for later tickets
  Radar,
  Globe,
  Server,
  ShieldAlert,
  Network,
  FileText,
  GitBranch,
} from "lucide-react";

/*
 * lucide-react swap for the design system's Icon wrapper.
 *
 * The stock design-system Icon (design-system/components/media/Icon.jsx) renders via
 * the Lucide UMD CDN script (window.lucide). In a bundled build there is no such
 * script, so we route every DS Icon import to this module (see astro.config.mjs).
 * Same public contract — { name, size, strokeWidth, style } — and the same kebab-case
 * Lucide names, so no DS consumer changes. This is the DS README's "one-file swap".
 *
 * We map the kebab names explicitly (rather than `import { icons }`) so the bundle
 * ships only the icons we use — importing the whole Lucide set balloons the client
 * chunk to ~700kB. Add a line here when a later ticket needs a new icon.
 */
const REGISTRY = {
  search: Search,
  "chevron-down": ChevronDown,
  "chevron-right": ChevronRight,
  check: Check,
  info: Info,
  "check-circle-2": CheckCircle2,
  "alert-triangle": AlertTriangle,
  copy: Copy,
  x: X,
  menu: Menu,
  "external-link": ExternalLink,
  radar: Radar,
  globe: Globe,
  server: Server,
  "shield-alert": ShieldAlert,
  network: Network,
  "file-text": FileText,
  "git-branch": GitBranch,
};

export function Icon({ name, size = 16, strokeWidth = 1.75, style }) {
  const box = {
    display: "inline-flex",
    alignItems: "center",
    justifyContent: "center",
    width: size,
    height: size,
    flex: "none",
    ...style,
  };
  const Cmp = REGISTRY[name];
  if (!Cmp) {
    if (import.meta.env && import.meta.env.DEV) {
      // eslint-disable-next-line no-console
      console.warn(`[docs Icon] no lucide icon registered for "${name}" — add it to src/ds/Icon.jsx`);
    }
    return <span style={box} aria-hidden="true" />;
  }
  return (
    <span style={box} aria-hidden="true">
      <Cmp size={size} strokeWidth={strokeWidth} color="currentColor" />
    </span>
  );
}
