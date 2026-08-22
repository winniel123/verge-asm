import React from "react";
import { Logo } from "../../components/media/Logo.jsx";
import { Icon } from "../../components/media/Icon.jsx";
import { Input } from "../../components/forms/Input.jsx";
import { Button } from "../../components/forms/Button.jsx";
import { Checkbox } from "../../components/forms/Checkbox.jsx";
import { CodeInput } from "../../components/forms/CodeInput.jsx";
import { Spinner } from "../../components/display/Spinner.jsx";
import { CopyValue } from "../../components/display/CopyValue.jsx";
import { Callout } from "../../components/feedback/Callout.jsx";
import { IdPButton } from "../../components/forms/IdPButton.jsx";

function InlineCode({ children }) {
  return <code style={{ font: "400 0.92em var(--font-mono)", background: "var(--surface-sunken)", border: "1px solid var(--border-default)", borderRadius: 6, padding: "1px 5px", color: "var(--text-body)" }}>{children}</code>;
}
const H = ({ title, sub }) => (
  <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
    <h1 style={{ margin: 0, font: "600 19px var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)" }}>{title}</h1>
    {sub && <span style={{ font: "400 12.5px/1.55 var(--font-ui)", color: "var(--text-muted)" }}>{sub}</span>}
  </div>
);
const Done = ({ title, sub, action }) => (
  <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: 12, padding: "18px 0", textAlign: "center" }}>
    <span style={{ display: "inline-flex", alignItems: "center", justifyContent: "center", width: 40, height: 40, borderRadius: 999, background: "var(--ok-soft)", border: "1px solid var(--ok-border)", color: "var(--ok)" }}>
      <Icon name="check" size={18} />
    </span>
    <span style={{ font: "600 15px var(--font-ui)", color: "var(--text-ink)" }}>{title}</span>
    {sub && <span style={{ font: "400 12.5px/1.6 var(--font-ui)", color: "var(--text-muted)", maxWidth: 300 }}>{sub}</span>}
    {action}
  </div>
);
const LinkBtn = ({ onClick, children }) => (
  <button type="button" onClick={onClick} style={{ background: "transparent", border: "none", padding: 0, cursor: "pointer", font: "500 12px var(--font-ui)", color: "var(--link)", textAlign: "left" }}>{children}</button>
);
const RECOVERY = ["k4mq-9d2x", "7hfa-t3wn", "p8rc-01zk", "vx5j-mm4d", "q2sl-88bh", "e6ty-r7cn", "a1zw-kk3p", "n9gd-45vu"];

/* Pre-auth surface: sign-in (credentials + TOTP), forgot/reset, MFA enrollment, invite acceptance.
   Demo: any credentials; code 482913 verifies everywhere. */
export function SignIn() {
  const [view, setView] = React.useState("signin"); // signin | forgot | reset | enroll | invite
  const [step, setStep] = React.useState("creds");
  const [email, setEmail] = React.useState("");
  const [pw, setPw] = React.useState("");
  const [emailErr, setEmailErr] = React.useState(null);
  const [pwErr, setPwErr] = React.useState(null);
  const [code, setCode] = React.useState("");
  const [codeErr, setCodeErr] = React.useState(null);
  const [trust, setTrust] = React.useState(true);
  const [fEmail, setFEmail] = React.useState("");
  const [fSent, setFSent] = React.useState(false);
  const [r1, setR1] = React.useState("");
  const [r2, setR2] = React.useState("");
  const [rErr, setRErr] = React.useState(null);
  const [rDone, setRDone] = React.useState(false);
  const [eStep, setEStep] = React.useState("scan"); // scan | verify | codes | done
  const [eCode, setECode] = React.useState("");
  const [eErr, setEErr] = React.useState(null);
  const [stored, setStored] = React.useState(false);
  const [iName, setIName] = React.useState("");
  const [iPw, setIPw] = React.useState("");
  const [iDone, setIDone] = React.useState(false);
  const go = (v) => { setView(v); setStep("creds"); setFSent(false); setRDone(false); setEStep("scan"); setECode(""); setEErr(null); setStored(false); setIDone(false); };
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
    else { setCodeErr("Code didn't match — codes rotate every 30s"); wipeTimer.current = setTimeout(() => { setCode(""); setCodeErr(null); }, 1600); }
  };
  const editCode = (v) => { if (wipeTimer.current) { clearTimeout(wipeTimer.current); wipeTimer.current = null; setCodeErr(null); } setCode(v); };
  const eVerify = (v) => {
    if (v === "482913") { setEErr(null); setEStep("codes"); }
    else { setEErr("Code didn't match — re-scan and try again"); setTimeout(() => { setECode(""); setEErr(null); }, 1600); }
  };
  const resetSubmit = () => {
    if (r1.length < 12) { setRErr("12+ characters"); return; }
    if (r1 !== r2) { setRErr("passwords don't match"); return; }
    setRErr(null); setRDone(true);
  };
  return (
    <div data-screen-label="Sign in" style={{ minHeight: "100vh", background: "var(--bg-page)", fontFamily: "var(--font-ui)", display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center", gap: 24, padding: 24 }}>
      <Logo size={26} wordmarkSize={20} />
      <div style={{ width: 400, maxWidth: "100%", background: "var(--surface)", border: "1px solid var(--border-default)", borderRadius: 20, boxShadow: "var(--shadow-md)", padding: 28, display: "flex", flexDirection: "column", gap: 20 }}>
        {view === "signin" && step === "creds" && (
          <form onSubmit={submitCreds} style={{ display: "flex", flexDirection: "column", gap: 16 }}>
            <H title="Sign in" sub="Your deployment, your data." />
            <Input label="Email" mono placeholder="operator@acmecorp.io" value={email} error={emailErr}
              onChange={(e) => { setEmail(e.target.value); if (emailErr) setEmailErr(null); }} autoFocus spellCheck={false} autoComplete="username" />
            <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
              <Input label="Password" type="password" value={pw} error={pwErr}
                onChange={(e) => { setPw(e.target.value); if (pwErr) setPwErr(null); }} autoComplete="current-password" />
              <LinkBtn onClick={() => go("forgot")}>Forgot password?</LinkBtn>
            </div>
            <Button type="submit" style={{ width: "100%", justifyContent: "center" }}>Sign in</Button>
            <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
              <span style={{ flex: 1, height: 1, background: "var(--row-sep)" }} />
              <span style={{ font: "500 10.5px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--text-muted)" }}>or</span>
              <span style={{ flex: 1, height: 1, background: "var(--row-sep)" }} />
            </div>
            <IdPButton provider="okta" onClick={() => setStep("done")} />
          </form>
        )}
        {view === "signin" && step === "totp" && (
          <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
            <H title="Two-factor check" sub={<span>Enter the code from your authenticator app for <span style={{ fontFamily: "var(--font-mono)", fontSize: 12 }}>{email.trim() || "your account"}</span>.</span>} />
            <CodeInput label="Verification code" value={code} error={codeErr} autoFocus
              onChange={editCode} onComplete={verify} hint={codeErr ? undefined : "6 digits · rotates every 30s"} />
            <Checkbox label="Trust this device for 30 days" checked={trust} onChange={setTrust} />
            <div style={{ display: "flex", gap: 8 }}>
              <Button variant="ghost" onClick={() => setStep("creds")}>Back</Button>
              <Button style={{ flex: 1, justifyContent: "center" }} disabled={code.length < 6} onClick={() => verify(code)}>Verify</Button>
            </div>
          </div>
        )}
        {view === "signin" && step === "done" && (
          <Done title="Signed in" action={<span style={{ display: "inline-flex", alignItems: "center", gap: 8, font: "400 12.5px var(--font-ui)", color: "var(--text-muted)" }}><Spinner size={14} /> <span style={{ whiteSpace: "nowrap" }}>Loading your console…</span></span>} />
        )}
        {view === "forgot" && !fSent && (
          <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
            <H title="Reset password" sub="Enter the account email. If it exists, a reset link goes out." />
            <Input label="Email" mono placeholder="operator@acmecorp.io" value={fEmail} onChange={(e) => setFEmail(e.target.value)} autoFocus spellCheck={false} />
            <Button style={{ width: "100%", justifyContent: "center" }} disabled={!/.+@.+\..+/.test(fEmail.trim())} onClick={() => setFSent(true)}>Send reset link</Button>
            <span style={{ font: "400 11.5px/1.6 var(--font-ui)", color: "var(--text-muted)" }}>No mail configured on this host? Reset directly: <InlineCode>verge users reset-password</InlineCode></span>
            <LinkBtn onClick={() => go("signin")}>Back to sign in</LinkBtn>
          </div>
        )}
        {view === "forgot" && fSent && (
          <Done title="Check the inbox" sub="If that account exists, a link is on its way. Links expire in 30 minutes."
            action={<Button variant="ghost" onClick={() => go("signin")}>Back to sign in</Button>} />
        )}
        {view === "reset" && !rDone && (
          <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
            <H title="Set a new password" sub={<span>For <span style={{ fontFamily: "var(--font-mono)", fontSize: 12 }}>ola@acmecorp.io</span> · link expires in 24 minutes.</span>} />
            <Input label="New password" type="password" value={r1} error={rErr} hint={rErr ? undefined : "12+ characters; a passphrase beats complexity"}
              onChange={(e) => { setR1(e.target.value); if (rErr) setRErr(null); }} autoFocus autoComplete="new-password" />
            <Input label="Confirm password" type="password" value={r2} onChange={(e) => { setR2(e.target.value); if (rErr) setRErr(null); }} autoComplete="new-password" />
            <Button style={{ width: "100%", justifyContent: "center" }} disabled={!r1 || !r2} onClick={resetSubmit}>Set password</Button>
          </div>
        )}
        {view === "reset" && rDone && (
          <Done title="Password updated" sub="Every other session was signed out."
            action={<Button onClick={() => go("signin")}>Back to sign in</Button>} />
        )}
        {view === "enroll" && eStep === "scan" && (
          <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
            <H title="Set up two-factor" sub="Scan with your authenticator app, or paste the secret." />
            <div style={{ display: "flex", gap: 16, alignItems: "center" }}>
              <span style={{ display: "inline-flex", alignItems: "center", justifyContent: "center", width: 120, height: 120, flex: "none", borderRadius: 12, background: "var(--surface-sunken)", border: "1px solid var(--border-default)", color: "var(--text-muted)" }}>
                <Icon name="qr-code" size={56} />
              </span>
              <div style={{ display: "flex", flexDirection: "column", gap: 8, minWidth: 0, flex: 1 }}>
                <span style={{ font: "500 10.5px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--text-muted)" }}>Secret</span>
                <CopyValue value="VG7K-2Q9X-8MRD-P3TL" />
                <span style={{ font: "400 11.5px/1.55 var(--font-ui)", color: "var(--text-muted)" }}>TOTP · SHA-1 · 30s period · 6 digits</span>
              </div>
            </div>
            <Button style={{ width: "100%", justifyContent: "center" }} onClick={() => setEStep("verify")}>Continue</Button>
            <LinkBtn onClick={() => go("signin")}>Cancel</LinkBtn>
          </div>
        )}
        {view === "enroll" && eStep === "verify" && (
          <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
            <H title="Confirm the app" sub="Enter the code it shows right now." />
            <CodeInput label="Verification code" value={eCode} error={eErr} autoFocus onChange={(v) => { setECode(v); if (eErr) setEErr(null); }} onComplete={eVerify}
              hint={eErr ? undefined : "6 digits · rotates every 30s"} />
            <div style={{ display: "flex", gap: 8 }}>
              <Button variant="ghost" onClick={() => setEStep("scan")}>Back</Button>
              <Button style={{ flex: 1, justifyContent: "center" }} disabled={eCode.length < 6} onClick={() => eVerify(eCode)}>Confirm</Button>
            </div>
          </div>
        )}
        {view === "enroll" && eStep === "codes" && (
          <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
            <H title="Recovery codes" sub="Each works once if the authenticator is lost." />
            <Callout tone="warn" title="Shown once">Store them now — Verge keeps only hashes.</Callout>
            <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 8 }}>
              {RECOVERY.map((c) => (
                <span key={c} style={{ font: "500 12px var(--font-mono)", color: "var(--text-body)", background: "var(--surface-sunken)", border: "1px solid var(--border-default)", borderRadius: 8, padding: "7px 10px", textAlign: "center" }}>{c}</span>
              ))}
            </div>
            <Button variant="secondary" icon={<Icon name="copy" size={14} />} onClick={() => { try { navigator.clipboard.writeText(RECOVERY.join("\n")); } catch (e) {} }}>Copy all</Button>
            <Checkbox label="I stored these somewhere safe" checked={stored} onChange={setStored} />
            <Button style={{ width: "100%", justifyContent: "center" }} disabled={!stored} onClick={() => setEStep("done")}>Finish enrollment</Button>
          </div>
        )}
        {view === "enroll" && eStep === "done" && (
          <Done title="Two-factor enabled" sub="You'll be asked for a code on new devices."
            action={<Button onClick={() => go("signin")}>Back to sign in</Button>} />
        )}
        {view === "invite" && !iDone && (
          <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
            <span style={{ font: "500 10.5px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--text-muted)" }}>Invitation</span>
            <H title="Join acmecorp" sub={<span>Invited by <span style={{ fontFamily: "var(--font-mono)", fontSize: 12 }}>dana@acmecorp.io</span> · viewer role.</span>} />
            <Input label="Display name" placeholder="Ola Pérez" value={iName} onChange={(e) => setIName(e.target.value)} autoFocus />
            <Input label="Password" type="password" value={iPw} hint="12+ characters; a passphrase beats complexity" onChange={(e) => setIPw(e.target.value)} autoComplete="new-password" />
            <Button style={{ width: "100%", justifyContent: "center" }} disabled={!iName.trim() || iPw.length < 12} onClick={() => setIDone(true)}>Create account and join</Button>
            <span style={{ font: "400 11.5px/1.6 var(--font-ui)", color: "var(--text-muted)" }}>Two-factor enrollment follows on first sign-in.</span>
          </div>
        )}
        {view === "invite" && iDone && (
          <Done title="Welcome to acmecorp" sub="Account created. Next: enroll two-factor."
            action={<Button onClick={() => go("enroll")}>Set up two-factor</Button>} />
        )}
      </div>
      <span style={{ font: "400 11px var(--font-mono)", color: "var(--text-muted)", display: "inline-flex", gap: 10, alignItems: "center", flexWrap: "wrap", justifyContent: "center" }}>
        <span>Verge ASM v0.9.2</span><span aria-hidden="true">·</span><span>AGPL-3.0</span><span aria-hidden="true">·</span>
        <a href="#" style={{ color: "var(--text-muted)" }}>GitHub</a>
      </span>
      <span style={{ font: "400 10.5px var(--font-mono)", color: "var(--text-muted)", display: "inline-flex", gap: 12, alignItems: "center", flexWrap: "wrap", justifyContent: "center" }}>
        <span style={{ letterSpacing: "0.07em", textTransform: "uppercase" }}>spec states</span>
        {[["signin", "sign in"], ["forgot", "forgot"], ["reset", "reset"], ["enroll", "enroll mfa"], ["invite", "invite"]].map(([v, l]) => (
          <button key={v} type="button" onClick={() => go(v)} style={{ background: "transparent", border: "none", padding: 0, cursor: "pointer", font: "inherit", color: view === v ? "var(--link)" : "var(--text-muted)", textDecoration: "underline" }}>{l}</button>
        ))}
      </span>
    </div>
  );
}
