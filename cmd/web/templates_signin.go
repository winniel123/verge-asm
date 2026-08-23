package main

import "html/template"

// SignIn screen — the chrome-less auth surfaces (login, TOTP verify, TOTP
// enrol). The sibling first-run bootstrap (setup) lives in templates_setup.go
// (T10, #305) and reuses the authbrand/authfoot partials defined here.
// Canonical screen: SignIn. Ported from examples/console/SignIn.jsx
// (T6, #282): the centered-card shell — brand lockup, a single card, and a mono
// footer note — with the credentials step, the TOTP verify step, and the SSO
// affordance the example composes.
//
// Two honest deviations from the example, each a data reality of this
// self-hosted build, not a restyle:
//   - SSO/IdP: the example wires an Okta IdPButton, but this build configures no
//     identity provider (no SSO backend exists), so the affordance renders in the
//     design-system empty/disabled state rather than fabricating a provider.
//   - Accounts are usernames, not emails, and there is no trust-this-device model;
//     the field stays "Username" and the device checkbox is omitted rather than
//     imply a capability the server does not have.
//
// Restyling within the existing token vocabulary and shared classes (ADR-0109);
// no design-system component is authored here. The centered-card CSS (.center,
// .card) and every class used below is T0's shared pageCSS — reused, not added.
var _ = template.Must(tmpl.Parse(signinTemplates))

const signinTemplates = `
{{define "authbrand"}}<div style="display:inline-flex;align-items:center;gap:8px"><span class="brand-glyph" aria-hidden="true"><svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.75"><circle cx="10" cy="10" r="8"/><circle cx="10" cy="10" r="4"/><circle cx="10" cy="10" r="1.4" fill="currentColor" stroke="none"/></svg></span><span class="wordmark">Verge<span class="chip">ASM</span></span></div>{{end}}

{{define "authfoot"}}<div class="mono" style="font-size:11px;color:var(--muted);display:inline-flex;gap:10px;align-items:center"><span>AGPL-3.0</span><span aria-hidden="true">&#183;</span><a href="https://github.com/winniel123/verge-asm" rel="noreferrer" style="color:var(--muted)">GitHub</a></div>{{end}}

{{define "login"}}{{template "head" .}}
<div class="center"><div style="display:flex;flex-direction:column;align-items:center;gap:var(--space-5);width:400px;max-width:100%">
{{template "authbrand" .}}
<div class="card" style="width:100%">
<div style="display:flex;flex-direction:column;gap:4px;margin-bottom:var(--space-4)">
<h1 style="margin:0">Sign in</h1>
<span class="muted" style="font-size:12.5px">Your deployment, your data.</span>
</div>
{{if .Error}}<div class="error">{{.Error}}</div>{{end}}
<form method="post" action="/login">
<label><span>Username</span><input name="username" autocomplete="username" spellcheck="false" autofocus required></label>
<label><span>Password</span><input name="password" type="password" autocomplete="current-password" required></label>
<button type="submit" style="width:100%">Sign in</button>
</form>
<div style="display:flex;align-items:center;gap:10px;margin:var(--space-4) 0">
<span style="flex:1;height:1px;background:var(--hairline)"></span>
<span class="microlabel" style="font-size:10.5px">or</span>
<span style="flex:1;height:1px;background:var(--hairline)"></span>
</div>
<button type="button" class="btn secondary" disabled style="width:100%;display:inline-flex;align-items:center;justify-content:center;gap:10px;opacity:0.6;cursor:not-allowed">
<span aria-hidden="true" style="display:inline-flex;align-items:center;justify-content:center;width:20px;height:20px;border-radius:6px;background:var(--sunken);border:1px solid var(--hairline);font-family:var(--mono);font-size:9.5px;font-weight:600;color:var(--muted)">SA</span>
Single sign-on not configured
</button>
<span class="muted" style="display:block;font-size:11.5px;line-height:1.6;margin-top:6px">Configure an identity provider on the host to enable SSO.</span>
<span class="muted" style="display:block;font-size:11.5px;line-height:1.6;margin-top:var(--space-3)">Locked out? Reset on the host: <code style="background:var(--sunken);border:1px solid var(--hairline);border-radius:6px;padding:1px 5px;font-size:0.92em">verge users reset-password</code></span>
</div>
{{template "authfoot" .}}
</div></div>{{template "foot" .}}{{end}}

{{define "totp"}}{{template "head" .}}
<div class="center"><div style="display:flex;flex-direction:column;align-items:center;gap:var(--space-5);width:400px;max-width:100%">
{{template "authbrand" .}}
<div class="card" style="width:100%">
<div style="display:flex;flex-direction:column;gap:4px;margin-bottom:var(--space-4)">
<h1 style="margin:0">Two-factor check</h1>
<span class="muted" style="font-size:12.5px">Enter the current code from your authenticator app.</span>
</div>
{{if .Error}}<div class="error">{{.Error}}</div>{{end}}
<form method="post" action="/login/totp">
<label style="margin-bottom:6px"><span>Verification code</span><input name="code" inputmode="numeric" autocomplete="one-time-code" maxlength="6" pattern="[0-9]*" autofocus required style="text-align:center;letter-spacing:0.5em;font-size:19px;font-weight:600;height:48px"></label>
<span class="microlabel" style="display:block;font-size:10.5px;margin-bottom:var(--space-4)">6 digits &#183; rotates every 30s</span>
<div class="row">
<a class="btn ghost" href="/login" style="text-decoration:none">Back</a>
<button type="submit" style="flex:1;text-align:center">Verify</button>
</div>
</form>
</div>
{{template "authfoot" .}}
</div></div>{{template "foot" .}}{{end}}

{{define "totp-enroll"}}{{template "head" .}}
<div class="center"><div style="display:flex;flex-direction:column;align-items:center;gap:var(--space-5);width:400px;max-width:100%">
{{template "authbrand" .}}
<div class="card" style="width:100%">
<div class="microlabel">Two-factor &#183; enrol</div>
<div style="display:flex;flex-direction:column;gap:4px;margin:6px 0 var(--space-4)">
<h1 style="margin:0">Enable two-factor</h1>
<span class="muted" style="font-size:12.5px">Add this secret to your authenticator, then confirm with the current code. Two-factor is not active until you confirm.</span>
</div>
<p class="microlabel" style="margin-bottom:6px">Secret</p>
<div class="secret">{{.Secret}}</div>
<p class="microlabel" style="margin-bottom:6px">otpauth URI</p>
<div class="secret">{{.OtpauthURI}}</div>
{{if .Error}}<div class="error">{{.Error}}</div>{{end}}
<form method="post" action="/account/totp/confirm">
<label style="margin-bottom:6px"><span>Current code</span><input name="code" inputmode="numeric" autocomplete="one-time-code" maxlength="6" pattern="[0-9]*" required style="text-align:center;letter-spacing:0.5em;font-size:19px;font-weight:600;height:48px"></label>
<div class="row"><button type="submit" style="flex:1;text-align:center">Confirm and enable</button>
<a class="btn ghost" href="/" style="text-decoration:none">Cancel</a></div>
</form>
</div>
{{template "authfoot" .}}
</div></div>{{template "foot" .}}{{end}}
`
