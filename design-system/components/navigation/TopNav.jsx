import React from "react";
import { Logo } from "../media/Logo.jsx";
import { IconButton } from "../forms/IconButton.jsx";
import { StatusDot } from "../display/StatusDot.jsx";
import { Avatar } from "../display/Avatar.jsx";
import { OrgSwitcher } from "./OrgSwitcher.jsx";
import { DropdownMenu } from "../feedback/DropdownMenu.jsx";
import { Popover } from "../feedback/Popover.jsx";
import { MessageList } from "../feedback/MessageList.jsx";

const DEFAULT_ITEMS = [
  { id: "dashboard", label: "Dashboard" },
  { id: "inventory", label: "Inventory" },
  { id: "signals", label: "Signals" },
  { id: "graph", label: "Graph" },
  { id: "reports", label: "Reports" },
];

function NavItem({ item, active, onClick }) {
  const [hov, setHov] = React.useState(false);
  return (
    <button type="button" onClick={onClick} onMouseEnter={() => setHov(true)} onMouseLeave={() => setHov(false)}
      style={{ display: "inline-flex", alignItems: "center", gap: 6, height: 32, padding: "0 12px", border: "none", borderRadius: 999, cursor: "pointer", fontFamily: "var(--font-ui)", fontSize: 13, fontWeight: active ? 600 : 500, color: active ? "var(--link)" : hov ? "var(--text-body)" : "var(--text-secondary)", background: active ? "var(--accent-soft)" : hov ? "var(--surface-sunken)" : "transparent", transition: "background var(--dur-fast) var(--ease-out)" }}>
      {item.label}
      {item.count != null && <span style={{ font: "600 10.5px var(--font-mono)", padding: "1px 6px", borderRadius: 999, background: active ? "var(--surface)" : "var(--surface-sunken)", color: "var(--text-secondary)" }}>{item.count}</span>}
    </button>
  );
}

export function TopNav({ items = DEFAULT_ITEMS, active = "dashboard", onNavigate, orgName = "acmecorp", orgs, activeOrg, onOrgChange, version = "v0.9.2", user = "Ola Pérez", scanRunning, onToggleTheme, onOpenPalette, messages, onOpenAllMessages, dark, style }) {
  const [inboxOpen, setInboxOpen] = React.useState(false);
  const isMac = /Mac|iP(hone|ad|od)/.test(navigator.platform || navigator.userAgent);
  const userMenu = []
    .concat(onNavigate ? [{ label: "Settings", icon: "settings", onSelect: () => onNavigate("settings") }] : [])
    .concat(onOpenPalette ? [{ label: "Command palette", icon: "search", shortcut: isMac ? "\u2318K" : "Ctrl+K", onSelect: onOpenPalette }] : []);
  const unread = messages && messages.some((m) => m.unread);
  return (
    <nav style={{ display: "flex", alignItems: "center", gap: 20, height: 56, padding: "0 24px", background: "var(--surface)", borderBottom: "1px solid var(--border-default)", fontFamily: "var(--font-ui)", ...style }}>
      <Logo size={20} wordmarkSize={17} style={{ flex: "none" }} />
      {orgs ? <OrgSwitcher orgs={orgs} active={activeOrg} onChange={onOrgChange} style={{ flex: "none" }} />
        : <span style={{ display: "inline-flex", alignItems: "center", height: 24, padding: "0 10px", borderRadius: 999, background: "var(--surface-sunken)", border: "1px solid var(--border-default)", font: "500 11.5px var(--font-mono)", color: "var(--text-secondary)", flex: "none" }}>{orgName}</span>}
      <span style={{ display: "flex", gap: 4, marginLeft: 8 }}>
        {items.map((it) => <NavItem key={it.id} item={it} active={it.id === active} onClick={() => onNavigate && onNavigate(it.id)} />)}
      </span>
      <span style={{ marginLeft: "auto", display: "flex", alignItems: "center", gap: 14, flex: "none" }}>
        {scanRunning && <StatusDot status="running" label="Scan running" micro />}
        {onOpenPalette && <IconButton icon="search" label={"Search (" + (/Mac|iP(hone|ad|od)/.test(navigator.platform || navigator.userAgent) ? "\u2318K" : "Ctrl+K") + ")"} onClick={onOpenPalette} />}
        {messages && (
          <Popover align="end" width={340} open={inboxOpen} onOpenChange={setInboxOpen} trigger={
            <span style={{ position: "relative", display: "inline-flex" }}>
              <IconButton icon="inbox" label="Messages" active={inboxOpen} />
              {unread && <span style={{ position: "absolute", top: 4, right: 4, width: 7, height: 7, borderRadius: 999, background: "var(--accent)", boxShadow: "0 0 0 2px var(--surface)" }} />}
            </span>
          }>
            <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
              <div style={{ display: "flex", alignItems: "center", padding: "0 4px" }}>
                <span style={{ font: "500 11px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--text-muted)" }}>Messages</span>
              </div>
              <MessageList messages={messages.slice(0, 4)} onOpen={() => { setInboxOpen(false); onOpenAllMessages && onOpenAllMessages(); }} />
              {onOpenAllMessages && (
                <button type="button" onClick={() => { setInboxOpen(false); onOpenAllMessages(); }}
                  style={{ border: "none", borderTop: "1px solid var(--row-sep)", background: "transparent", padding: "9px 4px 3px", font: "500 12px var(--font-ui)", color: "var(--link)", cursor: "pointer", textAlign: "left" }}>
                  All messages
                </button>
              )}
            </div>
          </Popover>
        )}
        <span style={{ font: "400 11px var(--font-mono)", color: "var(--text-muted)" }}>{version}</span>
        {onToggleTheme && <IconButton icon={dark ? "sun" : "moon"} label="Toggle theme" onClick={onToggleTheme} />}
        <a href="https://github.com" style={{ font: "500 12px var(--font-ui)", color: "var(--text-secondary)", textDecoration: "none" }}>GitHub</a>
        {userMenu.length ? (
          <DropdownMenu align="end" width={210} items={userMenu} trigger={
            <span role="button" aria-label="Account menu" style={{ display: "inline-flex", cursor: "pointer" }}><Avatar name={user} /></span>
          } />
        ) : <Avatar name={user} />}
      </span>
    </nav>
  );
}
