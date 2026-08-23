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
//   - SSO/IdP (#293, ADR-0112): the example wires an Okta IdPButton; this build now
//     backs it with real OpenID Connect. Each enabled provider renders a "Continue
//     with <name>" button linking to /login/sso/<slug>; where none is configured the
//     affordance stays the design-system not-configured state rather than fabricating
//     a provider. SSO authenticates an existing account, never creates one.
//   - Accounts are usernames, not emails, and there is no trust-this-device model;
//     the field stays "Username" and the device checkbox is omitted rather than
//     imply a capability the server does not have.
//
// The V3 delta (T19, #314, ADR-0110) ports the rest of SignIn.jsx's pre-auth
// surface onto real handlers, additive to the credentials + TOTP login above:
//   - forgot/reset password (forgot, forgot-sent, reset, reset-invalid,
//     reset-done). No mail on a self-hosted host, so the reset link is delivered
//     via the web logs — the same channel the setup token uses — and the forgot
//     step is non-enumerating.
//   - TOTP enrollment recovery codes (totp-recovery): confirming enrollment reveals
//     a set once, mono and copyable; only hashes are kept, and one code redeems the
//     login two-factor step once.
//   - invite acceptance (invite, invite-invalid): an invite token → set-credentials
//     screen that creates the account at the invited role. The creation side (minting
//     invite tokens) lands in T18 under Settings -> Team.
//
// Honest deviations carried over from the #282 port: accounts are usernames, not
// emails (the forgot/invite fields collect a username); reset-done does not claim a
// global sign-out this build's stateless-cookie sessions cannot perform. SSO is now
// real (OIDC, #293/ADR-0112): the affordance renders a button per configured provider
// and only falls to the not-configured state when none is set.
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
{{if .Notice}}<div class="notice">{{.Notice}}</div>{{end}}
{{if .Error}}<div class="error">{{.Error}}</div>{{end}}
<form method="post" action="/login">
<label><span>Username</span><input name="username" autocomplete="username" spellcheck="false" autofocus required></label>
<label style="margin-bottom:6px"><span>Password</span><input name="password" type="password" autocomplete="current-password" required></label>
<a href="/forgot" style="display:inline-block;font-size:12px;color:var(--link);margin-bottom:var(--space-4)">Forgot password?</a>
<button type="submit" style="width:100%">Sign in</button>
</form>
<div style="display:flex;align-items:center;gap:10px;margin:var(--space-4) 0">
<span style="flex:1;height:1px;background:var(--hairline)"></span>
<span class="microlabel" style="font-size:10.5px">or</span>
<span style="flex:1;height:1px;background:var(--hairline)"></span>
</div>
{{if .SSOProviders}}
{{range .SSOProviders}}<a class="btn secondary" href="/login/sso/{{.Slug}}" style="width:100%;display:inline-flex;align-items:center;justify-content:center;gap:10px;text-decoration:none;margin-bottom:8px">
<span aria-hidden="true" style="display:inline-flex;align-items:center;justify-content:center;width:20px;height:20px;border-radius:6px;background:var(--sunken);border:1px solid var(--hairline);color:var(--muted)"><svg viewBox="0 0 24 24" width="12" height="12" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"></path></svg></span>
Continue with {{.Name}}
</a>{{end}}
{{else}}
<button type="button" class="btn secondary" disabled style="width:100%;display:inline-flex;align-items:center;justify-content:center;gap:10px;opacity:0.6;cursor:not-allowed">
<span aria-hidden="true" style="display:inline-flex;align-items:center;justify-content:center;width:20px;height:20px;border-radius:6px;background:var(--sunken);border:1px solid var(--hairline);font-family:var(--mono);font-size:9.5px;font-weight:600;color:var(--muted)">SA</span>
Single sign-on not configured
</button>
<span class="muted" style="display:block;font-size:11.5px;line-height:1.6;margin-top:6px">An admin can add an OpenID Connect identity provider in Settings to enable SSO.</span>
{{end}}
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
<label style="margin-bottom:6px"><span>Verification code</span><input name="code" inputmode="text" autocomplete="one-time-code" autofocus required spellcheck="false" style="text-align:center;letter-spacing:0.4em;font-size:19px;font-weight:600;height:48px"></label>
<span class="microlabel" style="display:block;font-size:10.5px;margin-bottom:6px">6 digits &#183; rotates every 30s</span>
<span class="muted" style="display:block;font-size:11.5px;line-height:1.6;margin-bottom:var(--space-4)">Lost your authenticator? Enter one of your recovery codes instead.</span>
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

{{define "totp-recovery"}}{{template "head" .}}
<div class="center"><div style="display:flex;flex-direction:column;align-items:center;gap:var(--space-5);width:420px;max-width:100%">
{{template "authbrand" .}}
<div class="card" style="width:100%">
<div class="microlabel">Two-factor &#183; recovery codes</div>
<div style="display:flex;flex-direction:column;gap:4px;margin:6px 0 var(--space-4)">
<h1 style="margin:0">Two-factor enabled</h1>
<span class="muted" style="font-size:12.5px">Save these recovery codes. Each works once at sign-in if you lose your authenticator.</span>
</div>
<div class="banner warn" style="margin:0 0 var(--space-4)">Shown once &#8212; store them now. Verge keeps only hashes, so they cannot be shown again.</div>
<div style="display:grid;grid-template-columns:1fr 1fr;gap:8px;margin-bottom:var(--space-4)">
{{range .Codes}}<span class="mono cvcode" style="font-size:12px;color:var(--body);background:var(--sunken);border:1px solid var(--hairline);border-radius:var(--r-sm);padding:7px 10px;text-align:center">{{.}}</span>{{end}}
</div>
<button type="button" class="btn secondary" style="width:100%" onclick="var v=[].map.call(this.parentNode.querySelectorAll('.cvcode'),function(e){return e.textContent;}).join('\n');if(navigator.clipboard){navigator.clipboard.writeText(v);this.textContent='Copied';}">Copy all</button>
<a class="btn" href="/profile" style="width:100%;text-align:center;justify-content:center;margin-top:var(--space-3);text-decoration:none">Done</a>
</div>
{{template "authfoot" .}}
</div></div>{{template "foot" .}}{{end}}

{{define "forgot"}}{{template "head" .}}
<div class="center"><div style="display:flex;flex-direction:column;align-items:center;gap:var(--space-5);width:400px;max-width:100%">
{{template "authbrand" .}}
<div class="card" style="width:100%">
<div style="display:flex;flex-direction:column;gap:4px;margin-bottom:var(--space-4)">
<h1 style="margin:0">Reset password</h1>
<span class="muted" style="font-size:12.5px">Enter your account name. If it exists, a reset link goes out.</span>
</div>
<form method="post" action="/forgot">
<label><span>Username</span><input name="username" autocomplete="username" spellcheck="false" autofocus required></label>
<button type="submit" style="width:100%">Send reset link</button>
</form>
<span class="muted" style="display:block;font-size:11.5px;line-height:1.6;margin-top:var(--space-3)">No mail configured on this host? The link is written to the web logs, or reset directly: <code style="background:var(--sunken);border:1px solid var(--hairline);border-radius:6px;padding:1px 5px;font-size:0.92em">verge users reset-password</code></span>
<a href="/login" style="display:inline-block;font-size:12px;color:var(--link);margin-top:var(--space-3)">Back to sign in</a>
</div>
{{template "authfoot" .}}
</div></div>{{template "foot" .}}{{end}}

{{define "forgot-sent"}}{{template "head" .}}
<div class="center"><div style="display:flex;flex-direction:column;align-items:center;gap:var(--space-5);width:400px;max-width:100%">
{{template "authbrand" .}}
<div class="card" style="width:100%">
<div style="display:flex;flex-direction:column;align-items:center;gap:12px;padding:8px 0;text-align:center">
<span aria-hidden="true" style="display:inline-flex;align-items:center;justify-content:center;width:40px;height:40px;border-radius:var(--r-full);background:var(--ok-soft);border:1px solid var(--ok-border);color:var(--ok)"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg></span>
<span style="font-weight:600;font-size:15px;color:var(--ink)">Check for your link</span>
<span class="muted" style="font-size:12.5px;line-height:1.6;max-width:300px">If that account exists, a reset link is on its way. It expires in 30 minutes. On a host with no mail configured, look in the web logs.</span>
<a class="btn ghost" href="/login" style="text-decoration:none;margin-top:4px">Back to sign in</a>
</div>
</div>
{{template "authfoot" .}}
</div></div>{{template "foot" .}}{{end}}

{{define "reset"}}{{template "head" .}}
<div class="center"><div style="display:flex;flex-direction:column;align-items:center;gap:var(--space-5);width:400px;max-width:100%">
{{template "authbrand" .}}
<div class="card" style="width:100%">
<div style="display:flex;flex-direction:column;gap:4px;margin-bottom:var(--space-4)">
<h1 style="margin:0">Set a new password</h1>
<span class="muted" style="font-size:12.5px">Choose a new password for your account.</span>
</div>
{{if .Error}}<div class="error">{{.Error}}</div>{{end}}
<form method="post" action="/reset">
<input type="hidden" name="token" value="{{.Token}}">
<label style="margin-bottom:6px"><span>New password</span><input name="password" type="password" autocomplete="new-password" autofocus required></label>
<span class="muted" style="display:block;font-size:11.5px;margin-bottom:var(--space-4)">8+ characters; a passphrase beats complexity</span>
<label><span>Confirm password</span><input name="confirm" type="password" autocomplete="new-password" required></label>
<button type="submit" style="width:100%">Set password</button>
</form>
</div>
{{template "authfoot" .}}
</div></div>{{template "foot" .}}{{end}}

{{define "reset-invalid"}}{{template "head" .}}
<div class="center"><div style="display:flex;flex-direction:column;align-items:center;gap:var(--space-5);width:400px;max-width:100%">
{{template "authbrand" .}}
<div class="card" style="width:100%">
<div style="display:flex;flex-direction:column;gap:4px;margin-bottom:var(--space-4)">
<h1 style="margin:0">Link expired or already used</h1>
<span class="muted" style="font-size:12.5px">Reset links are single-use and expire after 30 minutes. Request a fresh one.</span>
</div>
<a class="btn" href="/forgot" style="width:100%;text-align:center;justify-content:center;text-decoration:none">Request a new link</a>
<a href="/login" style="display:inline-block;font-size:12px;color:var(--link);margin-top:var(--space-3)">Back to sign in</a>
</div>
{{template "authfoot" .}}
</div></div>{{template "foot" .}}{{end}}

{{define "reset-done"}}{{template "head" .}}
<div class="center"><div style="display:flex;flex-direction:column;align-items:center;gap:var(--space-5);width:400px;max-width:100%">
{{template "authbrand" .}}
<div class="card" style="width:100%">
<div style="display:flex;flex-direction:column;align-items:center;gap:12px;padding:8px 0;text-align:center">
<span aria-hidden="true" style="display:inline-flex;align-items:center;justify-content:center;width:40px;height:40px;border-radius:var(--r-full);background:var(--ok-soft);border:1px solid var(--ok-border);color:var(--ok)"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg></span>
<span style="font-weight:600;font-size:15px;color:var(--ink)">Password updated</span>
<span class="muted" style="font-size:12.5px;line-height:1.6;max-width:300px">Sign in with your new password. A session on another device lapses when it expires.</span>
<a class="btn" href="/login" style="text-decoration:none;margin-top:4px">Back to sign in</a>
</div>
</div>
{{template "authfoot" .}}
</div></div>{{template "foot" .}}{{end}}

{{define "invite"}}{{template "head" .}}
<div class="center"><div style="display:flex;flex-direction:column;align-items:center;gap:var(--space-5);width:400px;max-width:100%">
{{template "authbrand" .}}
<div class="card" style="width:100%">
<div class="microlabel">Invitation</div>
<div style="display:flex;flex-direction:column;gap:4px;margin:6px 0 var(--space-4)">
<h1 style="margin:0">Accept your invitation</h1>
<span class="muted" style="font-size:12.5px">You were invited to this deployment as <span class="badge">{{.Role}}</span>. Choose a username and password to join.</span>
</div>
{{if .Error}}<div class="error">{{.Error}}</div>{{end}}
<form method="post" action="/invite">
<input type="hidden" name="token" value="{{.Token}}">
<label><span>Username</span><input name="username" value="{{.Username}}" autocomplete="username" spellcheck="false" autofocus required></label>
<label style="margin-bottom:6px"><span>Password</span><input name="password" type="password" autocomplete="new-password" required></label>
<span class="muted" style="display:block;font-size:11.5px;margin-bottom:var(--space-4)">8+ characters; a passphrase beats complexity</span>
<button type="submit" style="width:100%">Create account and join</button>
</form>
<span class="muted" style="display:block;font-size:11.5px;line-height:1.6;margin-top:var(--space-3)">Two-factor enrollment follows on first sign-in.</span>
</div>
{{template "authfoot" .}}
</div></div>{{template "foot" .}}{{end}}

{{define "invite-invalid"}}{{template "head" .}}
<div class="center"><div style="display:flex;flex-direction:column;align-items:center;gap:var(--space-5);width:400px;max-width:100%">
{{template "authbrand" .}}
<div class="card" style="width:100%">
<div style="display:flex;flex-direction:column;gap:4px;margin-bottom:var(--space-4)">
<h1 style="margin:0">Invitation expired or already used</h1>
<span class="muted" style="font-size:12.5px">Invitations are single-use and time-boxed. Ask an admin to send a fresh one.</span>
</div>
<a href="/login" style="display:inline-block;font-size:12px;color:var(--link)">Back to sign in</a>
</div>
{{template "authfoot" .}}
</div></div>{{template "foot" .}}{{end}}
`
