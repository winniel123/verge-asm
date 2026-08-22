import React from "react";
import { Logo } from "../../components/media/Logo.jsx";
import { Icon } from "../../components/media/Icon.jsx";
import { Input } from "../../components/forms/Input.jsx";
import { Button } from "../../components/forms/Button.jsx";
import { Checkbox } from "../../components/forms/Checkbox.jsx";
import { CodeInput } from "../../components/forms/CodeInput.jsx";
import { Spinner } from "../../components/display/Spinner.jsx";
import { IdPButton } from "../../components/forms/IdPButton.jsx";

function InlineCode({ children }) {
  return <code style={{ font: "400 0.92em var(--font-mono)", background: "var(--surface-sunken)", border: "1px solid var(--border-default)", borderRadius: 6, padding: "1px 5px", color: "var(--text-body)" }}>{children}</code>;
}

/* Pre-auth sign-in: credentials, then the TOTP verify step (CodeInput). Self-hosted — no SSO upsell,
   password resets happen on the host. Demo: any credentials; code 482913 verifies. */
export function SignIn() {
  const [step, setStep] = React.useState("creds"); // creds | totp | done
  const [email, setEmail] = React.useState("");
  const [pw, setPw] = React.useState("");
  const [emailErr, setEmailErr] = React.useState(null);
  const [pwErr, setPwErr] = React.useState(null);
  const [code, setCode] = React.useState("");
  const [codeErr, setCodeErr] = React.useState(null);
  const [trust, setTrust] = React.useState(true);
  const submitCreds = (e) => {
    e.preventDefault();
    const ee = /.+@.+\..+/.test(email.trim()) ? null : "enter the account email";
    const pe = pw ? null : "required";
    setEmailErr(ee); setPwErr(pe);
    if (!ee && !pe) { setCode(""); setCodeErr(null); setStep("totp"); }
  };
  const wipeTimer = React.useRef(null);
  const verify = (v) => {
    if (v === "482913") { setCodeErr(null); setStep("done"); }
    else { setCodeErr("Code didn't match \u2014 codes rotate every 30s"); wipeTimer.current = setTimeout(() => { setCode(""); setCodeErr(null); }, 1600); }
  };
  const editCode = (v) => { if (wipeTimer.current) { clearTimeout(wipeTimer.current); wipeTimer.current = null; setCodeErr(null); } setCode(v); };
  return (
    <div data-screen-label="Sign in" style={{ minHeight: "100vh", background: "var(--bg-page)", fontFamily: "var(--font-ui)", display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center", gap: 24, padding: 24 }}>
      <Logo size={26} wordmarkSize={20} />
      <div style={{ width: 400, maxWidth: "100%", background: "var(--surface)", border: "1px solid var(--border-default)", borderRadius: 20, boxShadow: "var(--shadow-md)", padding: 28, display: "flex", flexDirection: "column", gap: 20 }}>
        {step === "creds" && (
          <form onSubmit={submitCreds} style={{ display: "flex", flexDirection: "column", gap: 16 }}>
            <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
              <h1 style={{ margin: 0, font: "600 19px var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)" }}>Sign in</h1>
              <span style={{ font: "400 12.5px var(--font-ui)", color: "var(--text-muted)" }}>Your deployment, your data.</span>
            </div>
            <Input label="Email" mono placeholder="operator@acmecorp.io" value={email} error={emailErr}
              onChange={(e) => { setEmail(e.target.value); if (emailErr) setEmailErr(null); }} autoFocus spellCheck={false} autoComplete="username" />
            <Input label="Password" type="password" value={pw} error={pwErr}
              onChange={(e) => { setPw(e.target.value); if (pwErr) setPwErr(null); }} autoComplete="current-password" />
            <Button type="submit" style={{ width: "100%", justifyContent: "center" }}>Sign in</Button>
            <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
              <span style={{ flex: 1, height: 1, background: "var(--row-sep)" }} />
              <span style={{ font: "500 10.5px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--text-muted)" }}>or</span>
              <span style={{ flex: 1, height: 1, background: "var(--row-sep)" }} />
            </div>
            <IdPButton provider="okta" onClick={() => setStep("done")} />
            <span style={{ font: "400 11.5px/1.6 var(--font-ui)", color: "var(--text-muted)" }}>Locked out? Reset on the host: <InlineCode>verge users reset-password</InlineCode></span>
          </form>
        )}
        {step === "totp" && (
          <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
            <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
              <h1 style={{ margin: 0, font: "600 19px var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)" }}>Two-factor check</h1>
              <span style={{ font: "400 12.5px var(--font-ui)", color: "var(--text-muted)" }}>Enter the code from your authenticator app for <span style={{ fontFamily: "var(--font-mono)", fontSize: 12 }}>{email.trim()}</span>.</span>
            </div>
            <CodeInput label="Verification code" value={code} error={codeErr} autoFocus
              onChange={editCode} onComplete={verify} hint={codeErr ? undefined : "6 digits \u00b7 rotates every 30s"} />
            <Checkbox label="Trust this device for 30 days" checked={trust} onChange={setTrust} />
            <div style={{ display: "flex", gap: 8 }}>
              <Button variant="ghost" onClick={() => setStep("creds")}>Back</Button>
              <Button style={{ flex: 1, justifyContent: "center" }} disabled={code.length < 6} onClick={() => verify(code)}>Verify</Button>
            </div>
          </div>
        )}
        {step === "done" && (
          <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: 12, padding: "18px 0" }}>
            <span style={{ display: "inline-flex", alignItems: "center", justifyContent: "center", width: 40, height: 40, borderRadius: 999, background: "var(--ok-soft)", border: "1px solid var(--ok-border)", color: "var(--ok)" }}>
              <Icon name="check" size={18} />
            </span>
            <span style={{ font: "600 15px var(--font-ui)", color: "var(--text-ink)" }}>Signed in</span>
            <span style={{ display: "inline-flex", alignItems: "center", gap: 8, font: "400 12.5px var(--font-ui)", color: "var(--text-muted)" }}>
              <Spinner size={14} /> <span style={{ whiteSpace: "nowrap" }}>Loading your console…</span>
            </span>
          </div>
        )}
      </div>
      <span style={{ font: "400 11px var(--font-mono)", color: "var(--text-muted)", display: "inline-flex", gap: 10, alignItems: "center" }}>
        <span>Verge ASM v0.9.2</span><span aria-hidden="true">·</span><span>AGPL-3.0</span><span aria-hidden="true">·</span>
        <a href="#" style={{ color: "var(--text-muted)" }}>GitHub</a>
      </span>
    </div>
  );
}
