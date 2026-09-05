import React from "react";
import {
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
  Radar,
  Globe,
  Server,
  ShieldAlert,
  Network,
  FileText,
  GitBranch,
} from "lucide-react";

/* Names map one by one because importing the whole lucide-react set balloons the client chunk. */
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
