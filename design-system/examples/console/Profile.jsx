import React from "react";
import { Card } from "../../components/display/Card.jsx";
import { Avatar } from "../../components/display/Avatar.jsx";
import { Badge } from "../../components/display/Badge.jsx";
import { Table } from "../../components/display/Table.jsx";
import { CopyValue } from "../../components/display/CopyValue.jsx";
import { Input } from "../../components/forms/Input.jsx";
import { Select } from "../../components/forms/Select.jsx";
import { Button } from "../../components/forms/Button.jsx";
import { IconButton } from "../../components/forms/IconButton.jsx";
import { Dialog } from "../../components/feedback/Dialog.jsx";
import { Callout } from "../../components/feedback/Callout.jsx";
import { Icon } from "../../components/media/Icon.jsx";

/* Personal account — identity, credentials, sessions, personal tokens. Org-wide access lives in Settings. */
export function Profile({ onToast }) {
  const [name, setName] = React.useState("Ola Pérez");
  const [tokens, setTokens] = React.useState([
    { name: "laptop-cli", prefix: "vg_pat_9f3k…", created: "2026-05-02", last: "2h" },
    { name: "grafana-readonly", prefix: "vg_pat_x81m…", created: "2026-07-19", last: "14d" },
  ]);
  const [tokOpen, setTokOpen] = React.useState(false);
  const [tokName, setTokName] = React.useState("");
  const [minted, setMinted] = React.useState(null);
  const mint = () => {
    setTokens(tokens.concat({ name: tokName.trim(), prefix: "vg_pat_" + Math.random().toString(36).slice(2, 6) + "…", created: "2026-08-22", last: "—" }));
    setMinted("vg_pat_" + Math.random().toString(36).slice(2, 10) + Math.random().toString(36).slice(2, 10));
    setTokName("");
  };
  const toast = (title, description, tone) => onToast && onToast({ tone: tone || "ok", title, description });
  return (
    <main data-screen-label="Profile" style={{ maxWidth: 1100, margin: "0 auto", padding: 32, display: "flex", flexDirection: "column", gap: 24 }}>
      <header style={{ display: "flex", alignItems: "center", gap: 14 }}>
        <Avatar name={name} size={44} />
        <div style={{ display: "flex", flexDirection: "column", gap: 2 }}>
          <h1 style={{ margin: 0, font: "600 21px var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)" }}>Profile</h1>
          <span style={{ font: "400 12.5px var(--font-ui)", color: "var(--text-muted)" }}>Your account on this deployment. Org-wide access lives in Settings → Team.</span>
        </div>
      </header>
      <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(380px, 1fr))", gap: 24, alignItems: "start" }}>
        <Card microLabel="Identity" title="Who you are">
          <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
            <Input label="Display name" value={name} onChange={(e) => setName(e.target.value)} />
            <Input label="Email" mono value="ola@acmecorp.io" readOnly hint="Sign-in identity — changed by an admin or your IdP" />
            <div><Button size="sm" onClick={() => toast("Profile saved", name)}>Save</Button></div>
          </div>
        </Card>
        <Card microLabel="Credentials" title="Password & two-factor">
          <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
            <Input label="Current password" type="password" autoComplete="current-password" />
            <Input label="New password" type="password" hint="12+ characters; a passphrase beats complexity" autoComplete="new-password" />
            <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
              <Button size="sm" variant="secondary" onClick={() => toast("Password changed", "Other sessions keep working until they expire.")}>Change password</Button>
            </div>
            <div style={{ display: "flex", alignItems: "center", gap: 10, paddingTop: 12, borderTop: "1px solid var(--row-sep)", flexWrap: "wrap" }}>
              <Badge tone="ok" dot>two-factor enabled</Badge>
              <span style={{ font: "400 11.5px var(--font-mono)", color: "var(--text-muted)" }}>TOTP · enrolled 2026-05-02</span>
              <span style={{ marginLeft: "auto" }}>
                <Button size="sm" variant="ghost" onClick={() => toast("Recovery codes rotated", "The old codes stopped working.", "neutral")}>Rotate recovery codes</Button>
              </span>
            </div>
          </div>
        </Card>
      </div>
      <Card microLabel="Sessions" title="Signed in right now" pad={0}
        action={<Button size="sm" variant="secondary" onClick={() => toast("Other sessions signed out", "2 sessions ended.", "neutral")}>Sign out others</Button>}>
        <Table framed={false} dense columns={[
          { key: "device", label: "Device", render: (r) => (
            <span style={{ display: "inline-flex", alignItems: "center", gap: 8 }}>
              <span style={{ font: "500 12.5px var(--font-ui)", color: "var(--text-ink)" }}>{r.device}</span>
              {r.current && <Badge tone="accent">this device</Badge>}
            </span>
          ) },
          { key: "ip", label: "IP", mono: true, width: 130 },
          { key: "last", label: "Last active", mono: true, width: 110 },
          { key: "act", label: "", width: 52, align: "right", clip: false, render: (r) => r.current ? null : <IconButton icon="log-out" label="End session" size="sm" onClick={() => toast("Session ended", r.device, "neutral")} /> },
        ]} rows={[
          { device: "Firefox · macOS", ip: "198.51.100.7", last: "now", current: true },
          { device: "CLI · verge@build-07", ip: "203.0.113.44", last: "2h" },
          { device: "Safari · iOS", ip: "198.51.100.31", last: "3d" },
        ]} rowKey="ip" />
      </Card>
      <Card microLabel="Automation" title="Personal API tokens" pad={0}
        action={<Button size="sm" icon={<Icon name="plus" size={13} />} onClick={() => { setMinted(null); setTokOpen(true); }}>New token</Button>}>
        <Table framed={false} dense columns={[
          { key: "name", label: "Name", render: (r) => <span style={{ font: "500 12.5px var(--font-ui)", color: "var(--text-ink)" }}>{r.name}</span> },
          { key: "prefix", label: "Token", mono: true, width: 150 },
          { key: "created", label: "Created", mono: true, width: 110 },
          { key: "last", label: "Last used", mono: true, width: 90 },
          { key: "act", label: "", width: 52, align: "right", clip: false, render: (r) => <IconButton icon="trash-2" label="Revoke" size="sm" onClick={() => { setTokens(tokens.filter((t) => t.name !== r.name)); toast("Token revoked", r.name, "neutral"); }} /> },
        ]} rows={tokens} rowKey="name" />
      </Card>
      <Dialog open={tokOpen} title="New personal token" description="Scoped to your account; inherits your role." onClose={() => setTokOpen(false)}
        footer={minted ? <Button onClick={() => setTokOpen(false)}>Done</Button> : <React.Fragment>
          <Button variant="ghost" onClick={() => setTokOpen(false)}>Cancel</Button>
          <Button disabled={!tokName.trim()} onClick={mint}>Create token</Button>
        </React.Fragment>}>
        {minted ? (
          <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
            <Callout tone="warn" title="Shown once">Store it now — Verge keeps only a hash.</Callout>
            <CopyValue value={minted} />
          </div>
        ) : (
          <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
            <Input label="Token name" placeholder="laptop-cli" value={tokName} onChange={(e) => setTokName(e.target.value)} autoFocus />
            <Select label="Scope" options={["Read-only", "Read + annotate", "Full (your role)"]} />
          </div>
        )}
      </Dialog>
    </main>
  );
}
