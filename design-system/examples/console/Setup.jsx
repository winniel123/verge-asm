import React from "react";
import { Logo } from "../../components/media/Logo.jsx";
import { Icon } from "../../components/media/Icon.jsx";
import { Input } from "../../components/forms/Input.jsx";
import { Button } from "../../components/forms/Button.jsx";
import { Callout } from "../../components/feedback/Callout.jsx";

function InlineCode({ children }) {
  return <code style={{ font: "400 0.92em var(--font-mono)", background: "var(--surface-sunken)", border: "1px solid var(--border-default)", borderRadius: 6, padding: "1px 5px", color: "var(--text-body)" }}>{children}</code>;
}

/* /setup — the one-time window while no accounts exist. Token is single-use; the window closes when spent. */
export function Setup() {
  const [token, setToken] = React.useState("");
  const [user, setUser] = React.useState("");
  const [pw, setPw] = React.useState("");
  const [done, setDone] = React.useState(false);
  const valid = token.trim().length >= 8 && user.trim().length >= 2 && pw.length >= 12;
  return (
    <div data-screen-label="Setup" style={{ minHeight: "100vh", background: "var(--bg-page)", fontFamily: "var(--font-ui)", display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center", gap: 24, padding: 24 }}>
      <Logo size={26} wordmarkSize={20} />
      <div style={{ width: 420, maxWidth: "100%", background: "var(--surface)", border: "1px solid var(--border-default)", borderRadius: 20, boxShadow: "var(--shadow-md)", padding: 28, display: "flex", flexDirection: "column", gap: 18 }}>
        {done ? (
          <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: 12, padding: "18px 0", textAlign: "center" }}>
            <span style={{ display: "inline-flex", alignItems: "center", justifyContent: "center", width: 40, height: 40, borderRadius: 999, background: "var(--ok-soft)", border: "1px solid var(--ok-border)", color: "var(--ok)" }}>
              <Icon name="check" size={18} />
            </span>
            <span style={{ font: "600 15px var(--font-ui)", color: "var(--text-ink)" }}>Setup complete</span>
            <span style={{ font: "400 12.5px/1.6 var(--font-ui)", color: "var(--text-muted)", maxWidth: 300 }}>The token is spent and /setup stops accepting it. Sign in to start the four-step checklist.</span>
            <a href="signin.html" style={{ font: "500 13px var(--font-ui)", color: "var(--link)" }}>Go to sign in</a>
          </div>
        ) : (
          <React.Fragment>
            <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
              <span style={{ font: "500 10.5px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--text-muted)" }}>First run</span>
              <h1 style={{ margin: 0, font: "600 19px var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)" }}>Create the admin account</h1>
              <span style={{ font: "400 12.5px/1.6 var(--font-ui)", color: "var(--text-muted)" }}>No accounts exist yet, so this window is open — once, for one account.</span>
            </div>
            <Input label="Setup token" mono value={token} onChange={(e) => setToken(e.target.value)} autoFocus spellCheck={false}
              placeholder="paste the single-use token" hint="Printed to the web logs on boot" />
            <Callout tone="neutral" title="Where the token lives">
              <InlineCode>docker compose logs web | grep /setup</InlineCode> — or pin it ahead of time with <InlineCode>VERGE_SETUP_TOKEN</InlineCode>.
            </Callout>
            <Input label="Username" mono placeholder="ola.perez" value={user} onChange={(e) => setUser(e.target.value)} spellCheck={false} autoComplete="username" />
            <Input label="Password" type="password" value={pw} hint="12+ characters; a passphrase beats complexity" onChange={(e) => setPw(e.target.value)} autoComplete="new-password" />
            <Button style={{ width: "100%", justifyContent: "center" }} disabled={!valid} onClick={() => setDone(true)}>Create admin account</Button>
            <span style={{ font: "400 11.5px/1.6 var(--font-ui)", color: "var(--text-muted)" }}>This account is the admin. Invite the rest under Settings → Team once inside.</span>
          </React.Fragment>
        )}
      </div>
      <span style={{ font: "400 11px var(--font-mono)", color: "var(--text-muted)", display: "inline-flex", gap: 10, alignItems: "center" }}>
        <span>Verge ASM v0.9.2</span><span aria-hidden="true">·</span><span>AGPL-3.0</span><span aria-hidden="true">·</span>
        <a href="signin.html" style={{ color: "var(--text-muted)" }}>Sign in</a>
      </span>
    </div>
  );
}
