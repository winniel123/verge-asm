import React from "react";
import { TopNav } from "../../components/navigation/TopNav.jsx";
import { Footer } from "../../components/navigation/Footer.jsx";
import { ToastStack } from "../../components/feedback/ToastStack.jsx";
import { Dialog } from "../../components/feedback/Dialog.jsx";
import { Button } from "../../components/forms/Button.jsx";
import { Input } from "../../components/forms/Input.jsx";
import { Select } from "../../components/forms/Select.jsx";
import { Checkbox } from "../../components/forms/Checkbox.jsx";
import { FileDrop } from "../../components/forms/FileDrop.jsx";
import { Dashboard } from "./Dashboard.jsx";
import { Signals } from "./Signals.jsx";
import { Reports } from "./Reports.jsx";
import { Inventory } from "./Inventory.jsx";
import { GraphView } from "./GraphView.jsx";
import { Drift } from "./Drift.jsx";
import { Scope } from "./Scope.jsx";
import { Settings } from "./Settings.jsx";
import { CommandPalette } from "../../components/feedback/CommandPalette.jsx";

export function ConsoleApp() {
  const [screen, setScreen] = React.useState("dashboard");
  const [dark, setDark] = React.useState(false);
  const [toasts, setToasts] = React.useState([]);
  const setToast = (t) => setToasts((ts) => ts.concat({ id: String(Date.now() + Math.random()), ...t }));
  const dismissToast = (id) => setToasts((ts) => ts.filter((x) => x.id !== id));
  const [scanning, setScanning] = React.useState(false);
  const [addOpen, setAddOpen] = React.useState(false);
  const [paletteOpen, setPaletteOpen] = React.useState(false);
  const [org, setOrg] = React.useState("acme");
  const [settingsSection, setSettingsSection] = React.useState("scans");
  const MSGS = [
    { id: "m1", cls: "signals", text: "VNC exposed to internet · edge-gw-03.acmecorp.io", time: "4m", unread: true },
    { id: "m2", cls: "drift", text: "appeared · staging-5.acmecorp.io", time: "8m", unread: true },
    { id: "m3", cls: "coverage", text: "zone transfer silent for 9d · internal.acmecorp.io", time: "6h" },
    { id: "m4", cls: "batches", text: "batch complete · 3 new signals", time: "6h" },
  ];
  const ORGS = [
    { id: "acme", name: "acmecorp", assets: 1284 },
    { id: "north", name: "northwind-sec", assets: 342 },
    { id: "globex", name: "globex", assets: 5108 },
  ];
  const [watch, setWatch] = React.useState(true);
  React.useEffect(() => {
    document.documentElement.setAttribute("data-theme", dark ? "dark" : "");
  }, [dark]);
  React.useEffect(() => {
    const onKey = (e) => {
      if ((e.metaKey || e.ctrlKey) && (e.key === "k" || e.key === "K")) { e.preventDefault(); setPaletteOpen((v) => !v); }
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, []);
  const runScan = () => {
    setScanning(true);
    setToast({ tone: "neutral", title: "Scan started", description: "214 subjects queued across 3 vantages." });
    setTimeout(() => { setScanning(false); setToast({ tone: "ok", title: "Scan complete", description: "3 new signals raised." }); }, 9000);
  };
  const items = [
    { id: "dashboard", label: "Dashboard" },
    { id: "scope", label: "Scope" },
    { id: "inventory", label: "Inventory" },
    { id: "drift", label: "Drift" },
    { id: "signals", label: "Signals", count: 47 },
    { id: "graph", label: "Graph" },
    { id: "reports", label: "Reports" },
  ];
  return (
    <div style={{ minHeight: "100vh", display: "flex", flexDirection: "column", background: "var(--bg-page)" }}>
      <TopNav items={items} active={screen} onNavigate={setScreen} scanRunning={scanning} dark={dark} onToggleTheme={() => setDark(!dark)} onOpenPalette={() => setPaletteOpen(true)}
        messages={MSGS} onOpenAllMessages={() => { setSettingsSection("messages"); setScreen("settings"); }}
        orgs={ORGS} activeOrg={org}
        onOrgChange={(id) => { setOrg(id); const o = ORGS.find((x) => x.id === id); setToast({ tone: "neutral", title: "Org switched", description: o.name + " \u00b7 " + o.assets.toLocaleString("en-US") + " assets" }); }} />
      <div style={{ flex: 1 }}>
        {screen === "dashboard" && <Dashboard scanning={scanning} onRunScan={runScan} onAddTarget={() => setAddOpen(true)} onOpenSignals={() => setScreen("signals")} />}
        {screen === "inventory" && <Inventory onToast={setToast} />}
        {screen === "scope" && <Scope onToast={setToast} />}
        {screen === "drift" && <Drift />}
        {screen === "graph" && <GraphView />}
        {screen === "reports" && <Reports />}
        {screen === "settings" && <Settings onToast={setToast} section={settingsSection} />}
        {screen === "signals" && <Signals onToast={setToast} onAnnotate={(s) => setToast({ tone: "neutral", title: "Annotation recorded", description: s.id + " \u00b7 accepted risk" })} />}
      </div>
      <Footer />
      <ToastStack toasts={toasts} onDismiss={dismissToast} />
      <CommandPalette open={paletteOpen} onClose={() => setPaletteOpen(false)} groups={[
        { label: "Screens", items: [
          { label: "Dashboard", icon: "layout-dashboard", onSelect: () => setScreen("dashboard") },
          { label: "Scope", icon: "globe", onSelect: () => setScreen("scope") },
          { label: "Inventory", icon: "server", onSelect: () => setScreen("inventory") },
          { label: "Drift", icon: "git-branch", onSelect: () => setScreen("drift") },
          { label: "Signals", icon: "shield-alert", hint: "47 open", onSelect: () => setScreen("signals") },
          { label: "Graph", icon: "network", onSelect: () => setScreen("graph") },
          { label: "Reports", icon: "file-text", onSelect: () => setScreen("reports") },
          { label: "Integrations", icon: "puzzle", onSelect: () => { setSettingsSection("integrations"); setScreen("settings"); } },
          { label: "Settings", icon: "settings", onSelect: () => setScreen("settings") },
        ] },
        { label: "Actions", items: [
          { label: "Run scan", icon: "play", onSelect: runScan },
          { label: "Add seed", icon: "plus", onSelect: () => setAddOpen(true) },
          { label: "Toggle theme", icon: dark ? "sun" : "moon", onSelect: () => setDark(!dark) },
        ] },
        { label: "Assets", items: [
          { label: "edge-gw-03.acmecorp.io", icon: "server", hint: "critical", onSelect: () => setScreen("inventory") },
          { label: "api.acmecorp.io", icon: "globe", hint: "high", onSelect: () => setScreen("inventory") },
          { label: "acmecorp.io", icon: "globe", onSelect: () => setScreen("inventory") },
        ] },
      ]} />
      <Dialog open={addOpen} title="Add seed" description="Verge starts scanning within a minute." onClose={() => setAddOpen(false)}
        footer={<React.Fragment>
          <Button variant="ghost" onClick={() => setAddOpen(false)}>Cancel</Button>
          <Button onClick={() => { setAddOpen(false); setToast({ tone: "neutral", title: "Seed added", description: "First scan queued." }); }}>Add seed</Button>
        </React.Fragment>}>
        <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
          <Input label="Seed" mono placeholder="acmecorp.io or 203.0.113.0/24" hint="Domain or CIDR range" />
          <Select label="Scan profile" options={["Standard", "Deep", "Passive only"]} />
          <Checkbox label="Watch for drift" description="Re-scan every 6h and raise signals on change" checked={watch} onChange={setWatch} />
          <FileDrop compact accept=".csv" label="Or drop a CSV to import many" hint="One seed per line" onFiles={() => setToast({ tone: "neutral", title: "CSV parsed", description: "Targets will queue when you add." })} />
        </div>
      </Dialog>
    </div>
  );
}
