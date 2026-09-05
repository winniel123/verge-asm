import React from "react";
import { Card } from "../../components/display/Card.jsx";
import { Avatar } from "../../components/display/Avatar.jsx";
import { Badge } from "../../components/display/Badge.jsx";
import { Table } from "../../components/display/Table.jsx";
import { CopyValue } from "../../components/display/CopyValue.jsx";
import { Input } from "../../components/forms/Input.jsx";
import { Button } from "../../components/forms/Button.jsx";
import { IconButton } from "../../components/forms/IconButton.jsx";
import { Dialog } from "../../components/feedback/Dialog.jsx";
import { ConfirmDialog } from "../../components/feedback/ConfirmDialog.jsx";
import { Callout } from "../../components/feedback/Callout.jsx";
import { Icon } from "../../components/media/Icon.jsx";

export function Profile({ onToast, totpEnabled = true }) {
  const username = "ola.perez";
  const [tokens, setTokens] = React.useState([
    { name: "laptop-cli", prefix: "vg_pat_9f3k…", created: "2026-05-02", last: "2h" },
    { name: "grafana-readonly", prefix: "vg_pat_x81m…", created: "2026-07-19", last: "14d" },
  ]);
  const [tokOpen, setTokOpen] = React.useState(false);
  const [tokName, setTokName] = React.useState("");
  const [minted, setMinted] = React.useState(null);
  const [endOpen, setEndOpen] = React.useState(false);
  const [othersOpen, setOthersOpen] = React.useState(false);
  const [revokeTok, setRevokeTok] = React.useState(null);
  const [identities, setIdentities] = React.useState([
    { id: "i1", provider: "Okta", identity: "ola.perez@acmecorp.io", linked: "2026-06-30" },
  ]);
  const mint = () => {
    setTokens(tokens.concat({ name: tokName.trim(), prefix: "vg_pat_" + Math.random().toString(36).slice(2, 6) + "…", created: "2026-08-24", last: "—" }));
    setMinted("vg_pat_" + Math.random().toString(36).slice(2, 10) + Math.random().toString(36).slice(2, 10));
    setTokName("");
  };
  const toast = (title, description, tone) => onToast && onToast({ tone: tone || "ok", title, description });
  return (
    <main data-screen-label="Profile" style={{ maxWidth: 1100, margin: "0 auto", padding: 32, display: "flex", flexDirection: "column", gap: 24 }}>
      <header style={{ display: "flex", alignItems: "center", gap: 14 }}>
        <Avatar name="Ola Perez" size={44} />
        <div style={{ display: "flex", flexDirection: "column", gap: 2 }}>
          <h1 style={{ margin: 0, font: "600 21px var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)" }}>Profile</h1>
          <span style={{ font: "400 12.5px var(--font-ui)", color: "var(--text-muted)" }}>Your account on this deployment. Org-wide access lives in Settings → Team.</span>
        </div>
      </header>
      <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(380px, 1fr))", gap: 24, alignItems: "start" }}>
        <Card microLabel="Identity" title="Who you are">
          <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
            <Input label="Username" mono value={username} readOnly hint="Sign-in identity · changed by an admin" />
            <div style={{ display: "flex", gap: 24, alignItems: "baseline" }}>
              <span style={{ font: "400 12.5px var(--font-ui)", color: "var(--text-muted)", width: 90 }}>Role</span>
              <Badge>admin</Badge>
            </div>
            <div style={{ display: "flex", gap: 24, alignItems: "baseline" }}>
              <span style={{ font: "400 12.5px var(--font-ui)", color: "var(--text-muted)", width: 90 }}>Member since</span>
              <span style={{ font: "400 12.5px var(--font-mono)", color: "var(--text-body)" }}>2026-04-11</span>
            </div>
          </div>
        </Card>
        <Card microLabel="Credentials" title="Password & two-factor">
          <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
            <Input label="Current password" type="password" autoComplete="current-password" />
            <Input label="New password" type="password" hint="12+ characters; a passphrase beats complexity" autoComplete="new-password" />
            <div><Button size="sm" variant="secondary" onClick={() => toast("Password changed", "Other sessions keep working until they expire.")}>Change password</Button></div>
            <div style={{ display: "flex", alignItems: "center", gap: 10, paddingTop: 12, borderTop: "1px solid var(--row-sep)", flexWrap: "wrap" }}>
              {totpEnabled ? (
                <React.Fragment>
                  <Badge tone="ok" dot>two-factor enabled</Badge>
                  <span style={{ font: "400 11.5px var(--font-mono)", color: "var(--text-muted)" }}>TOTP</span>
                </React.Fragment>
              ) : (
                <React.Fragment>
                  <Badge>two-factor off</Badge>
                  <span style={{ font: "400 11.5px var(--font-ui)", color: "var(--text-muted)" }}>Add a second factor to your sign-in.</span>
                  <Button size="sm" variant="secondary" style={{ marginLeft: "auto" }} onClick={() => toast("TOTP enrolment", "Scan the code with your authenticator.", "neutral")}>Enable two-factor</Button>
                </React.Fragment>
              )}
            </div>
          </div>
        </Card>
      </div>
      <Card microLabel="Sessions" title="Signed in right now" pad={0}
        action={<span style={{ display: "inline-flex", gap: 8 }}>
          <Button size="sm" variant="secondary" onClick={() => setOthersOpen(true)}>Sign out others</Button>
          <Button size="sm" variant="secondary" onClick={() => setEndOpen(true)}>End this session</Button>
        </span>}>
        <Table framed={false} dense columns={[
          { key: "device", label: "Device", render: (r) => (
            <span style={{ display: "inline-flex", alignItems: "center", gap: 8 }}>
              <span style={{ font: "500 12.5px var(--font-ui)", color: "var(--text-ink)" }}>{r.device}</span>
              {r.current && <Badge tone="accent">this device</Badge>}
            </span>
          ) },
          { key: "ip", label: "IP", mono: true, width: 130 },
          { key: "last", label: "Last active", mono: true, width: 110 },
          { key: "act", label: "", width: 52, align: "right", clip: false, render: (r) => r.current ? null : <IconButton icon="log-out" label="Revoke this session" size="sm" onClick={() => toast("Session revoked", r.device + " signs out on its next request.", "neutral")} /> },
        ]} rows={[
          { device: "Firefox · macOS", ip: "198.51.100.7", last: "now", current: true },
          { device: "CLI · verge@build-07", ip: "203.0.113.44", last: "2h" },
          { device: "Safari · iOS", ip: "198.51.100.31", last: "3d" },
        ]} rowKey="ip" />
        <p style={{ margin: 0, padding: "10px 16px 14px", font: "400 12px/1.5 var(--font-ui)", color: "var(--text-muted)" }}>Each row is a live session for your account. Revoking one signs that device out on its next request; a session also lapses on its own when it expires.</p>
      </Card>
      <Card microLabel="Access · single sign-on" title="Linked identities" pad={0}>
        <p style={{ margin: 0, padding: "0 16px 12px", font: "400 12.5px/1.55 var(--font-ui)", color: "var(--text-muted)" }}>Link an identity provider to sign in with it. A link binds your verified provider account to this one; sign-in matches on that binding, never on a username. An admin configures providers in Settings → Single sign-on.</p>
        <Table framed={false} dense columns={[
          { key: "provider", label: "Provider", render: (r) => <span style={{ font: "500 12.5px var(--font-ui)", color: "var(--text-ink)" }}>{r.provider}</span> },
          { key: "identity", label: "Identity", mono: true },
          { key: "linked", label: "Linked", mono: true, width: 110 },
          { key: "act", label: "", width: 80, align: "right", clip: false, render: (r) => <Button size="sm" variant="secondary" onClick={() => { setIdentities(identities.filter((i) => i.id !== r.id)); toast("Identity unlinked", r.provider + " · password sign-in still works.", "neutral"); }}>Unlink</Button> },
        ]} rows={identities} rowKey="id"
          empty="No identities linked yet." />
        <div style={{ display: "flex", alignItems: "center", gap: 10, flexWrap: "wrap", padding: "12px 16px 14px", borderTop: "1px solid var(--row-sep)" }}>
          <span style={{ font: "400 12px var(--font-ui)", color: "var(--text-muted)" }}>Link an identity:</span>
          <Button size="sm" variant="secondary" onClick={() => toast("Redirecting to Google", "Complete sign-in there to link.", "neutral")}>Google</Button>
        </div>
      </Card>
      <Card microLabel="Automation" title="Personal API tokens" pad={0}
        action={<Button size="sm" icon={<Icon name="plus" size={13} />} onClick={() => { setMinted(null); setTokOpen(true); }}>New token</Button>}>
        <Table framed={false} dense columns={[
          { key: "name", label: "Name", render: (r) => <span style={{ font: "500 12.5px var(--font-ui)", color: "var(--text-ink)" }}>{r.name}</span> },
          { key: "prefix", label: "Token", mono: true, width: 150 },
          { key: "created", label: "Created", mono: true, width: 110 },
          { key: "last", label: "Last used", mono: true, width: 90 },
          { key: "act", label: "", width: 52, align: "right", clip: false, render: (r) => <IconButton icon="trash-2" label="Revoke" size="sm" onClick={() => setRevokeTok(r)} /> },
        ]} rows={tokens} rowKey="name" />
        <p style={{ margin: 0, padding: "10px 16px 14px", font: "400 12px/1.5 var(--font-ui)", color: "var(--text-muted)" }}>A token is scoped to your account and inherits your role. It is shown once at creation — Verge keeps only a hash.</p>
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
          <Input label="Token name" placeholder="laptop-cli" value={tokName} onChange={(e) => setTokName(e.target.value)} autoFocus />
        )}
      </Dialog>
      <ConfirmDialog open={!!revokeTok} title={"Revoke " + (revokeTok ? revokeTok.name : "")} tone="danger" confirmLabel="Revoke token"
        message="Any automation using this token stops working at once."
        detail="This cannot be undone — a revoked token is never recoverable, only re-minted."
        onConfirm={() => { setTokens(tokens.filter((t) => t.name !== revokeTok.name)); toast("Token revoked", revokeTok.name, "neutral"); }} onClose={() => setRevokeTok(null)} />
      <ConfirmDialog open={endOpen} title="End this session" tone="danger" confirmLabel="End session"
        message="You are signed out on this device and returned to sign-in."
        detail="This ends only this device. To sign a different device out, use “Sign out others” or revoke it from the Sessions list."
        onConfirm={() => { setEndOpen(false); toast("Session ended", "Signed out on this device.", "neutral"); }} onClose={() => setEndOpen(false)} />
      <ConfirmDialog open={othersOpen} title="Sign out your other sessions" tone="danger" confirmLabel="Sign out others"
        message="Every other signed-in device is signed out on its next request. This session — the one you are using now — keeps working."
        detail="Use this if you signed in somewhere you no longer trust."
        onConfirm={() => { setOthersOpen(false); toast("Other sessions signed out", "2 sessions ended.", "neutral"); }} onClose={() => setOthersOpen(false)} />
    </main>
  );
}
