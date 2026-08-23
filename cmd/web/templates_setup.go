package main

import "html/template"

// Setup screen — the chrome-less first-run bootstrap surface. Canonical screen:
// Setup. Ported from design-system/examples/console/Setup.jsx (T10, #305): the
// centered-card shell (the shared authbrand lockup + authfoot note from the
// SignIn family), the "First run" micro-label header, the single-use setup-token
// field with its "where the token lives" callout, the admin credential fields,
// and the create action.
//
// This re-skins the existing bootstrap flow only — the setupForm/setupSubmit
// handlers and the VERGE_SETUP_TOKEN gate (auth.go) are unchanged. The token is
// single-use: the first admin it mints closes the window (setupClosed), after
// which /setup redirects to /login and this page never renders again. admin is
// the only initial role.
//
// Honest deviations from the example, each a data reality of this build (not a
// restyle), matching the pattern SignIn.jsx already set:
//   - Accounts are usernames, not emails (there is no email/identity model), so
//     the field stays "Username" and posts the `username` the handler reads,
//     rather than the example's "Email".
//   - The example's client-side "Setup complete" success card is a React-only
//     state; the real flow answers a 303 redirect to /login on success, so no
//     done branch is rendered here — the redirect is the honest completion.
//   - The footer reuses the shared authfoot rather than the example's hard-coded
//     "v0.9.2" version string, which is sample data with no real source.
//
// Restyling within the existing token vocabulary and shared classes (ADR-0109);
// no design-system component is authored here. authbrand/authfoot and the
// .center/.card/.microlabel/.error classes are the SignIn family's shared chrome
// (T0/T6 pageCSS) — reused, not added.
var _ = template.Must(tmpl.Parse(setupTemplates))

const setupTemplates = `
{{define "setup"}}{{template "head" .}}
<div class="center"><div style="display:flex;flex-direction:column;align-items:center;gap:var(--space-5);width:420px;max-width:100%">
{{template "authbrand" .}}
<div class="card" style="width:100%">
<div class="microlabel">First run</div>
<div style="display:flex;flex-direction:column;gap:4px;margin:6px 0 var(--space-4)">
<h1 style="margin:0">Create the admin account</h1>
<span class="muted" style="font-size:12.5px">No accounts exist yet, so this window is open &#8212; once, for one account.</span>
</div>
{{if .Error}}<div class="error">{{.Error}}</div>{{end}}
<form method="post" action="/setup">
<label><span>Setup token</span><input name="token" value="{{.Token}}" autocomplete="off" spellcheck="false" placeholder="paste the single-use token" autofocus required><span class="muted" style="font-size:11.5px">Printed to the web logs on boot</span></label>
<div style="display:flex;gap:10px;align-items:flex-start;padding:12px 14px;background:var(--sunken);border:1px solid var(--hairline);border-radius:var(--r-lg);margin-bottom:var(--space-4)">
<span aria-hidden="true" style="color:var(--muted);display:inline-flex;margin-top:1px;flex:none"><svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg></span>
<span style="display:flex;flex-direction:column;gap:2px;min-width:0">
<span style="font-weight:600;font-size:13px;color:var(--ink)">Where the token lives</span>
<span style="font-size:13px;line-height:1.55;color:var(--body)"><code style="background:var(--sunken);border:1px solid var(--hairline);border-radius:6px;padding:1px 5px;font-size:0.92em">docker compose logs web | grep /setup</code> &#8212; or pin it ahead of time with <code style="background:var(--sunken);border:1px solid var(--hairline);border-radius:6px;padding:1px 5px;font-size:0.92em">VERGE_SETUP_TOKEN</code>.</span>
</span>
</div>
<label><span>Username</span><input name="username" autocomplete="username" spellcheck="false" required></label>
<label><span>Password</span><input name="password" type="password" autocomplete="new-password" required><span class="muted" style="font-size:11.5px">12+ characters; a passphrase beats complexity</span></label>
<button type="submit" style="width:100%">Create admin account</button>
<span class="muted" style="display:block;font-size:11.5px;line-height:1.6;margin-top:var(--space-3)">This account is the admin. Invite the rest under Settings &#8594; Team once inside.</span>
</form>
</div>
{{template "authfoot" .}}
</div></div>{{template "foot" .}}{{end}}
`
