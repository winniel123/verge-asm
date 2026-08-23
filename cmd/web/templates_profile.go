package main

import "html/template"

// Profile screen — the account's own page at `/profile`, ported from
// design-system/examples/console/Profile.jsx (T9, #304, ADR-0110): the identity +
// credentials two-up, the current session, and the personal API tokens, with the
// New-token dialog's reveal-once mint. Restyling within the existing token
// vocabulary and the shared pageCSS classes (ADR-0109); no design-system component
// is authored here.
//
// Honest deviations from the example, each a data reality of this self-hosted
// build rather than a restyle (mirroring the SignIn port, #282):
//   - Identity is a username, not a display name + email. Accounts sign in with a
//     username changed by an admin (there is no IdP and no separate profile name),
//     so the Identity card reads the account facts read-only rather than offering an
//     editable display name / email the server does not model.
//   - Sessions are stateless signed cookies with no server-side registry, so only
//     the current session is knowable. The card renders it honestly and the one
//     revocable session — the current one — ends through the ConfirmDialog; there is
//     no fabricated list of other devices to "sign out".
//   - A token inherits the account's role (no partial scopes are enforced), so the
//     New-token dialog states that rather than offering a Scope select that would
//     bind nothing. Recovery-code rotation is not a feature of this build, so the
//     2FA row shows status and links the existing TOTP enrolment flow, no more.
//   - "Settings → Team" resolves to the real Access sub-tab in this build's IA.
var _ = template.Must(tmpl.Parse(profileTemplates))

const profileTemplates = `
{{define "profile"}}{{template "head" .}}
{{template "chrome" .}}
<main style="max-width:1100px">

<header style="display:flex;align-items:center;gap:14px;margin-bottom:var(--space-5)">
<span aria-hidden="true" class="avatar" style="width:44px;height:44px;font-size:16px">{{.Initials}}</span>
<div style="display:flex;flex-direction:column;gap:2px">
<h1 style="margin:0;font-size:21px">Profile</h1>
<span class="muted" style="font-size:12.5px">Your account on this deployment. Org-wide access lives in <a href="/settings?tab=team">Settings &#8594; Team</a>.</span>
</div>
</header>

{{if .Notice}}<div class="notice">{{.Notice}}</div>{{end}}

<div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(360px,1fr));gap:var(--space-5);align-items:start">

<div class="section" style="margin-bottom:0">
<div class="microlabel">Identity</div>
<h2>Who you are</h2>
<label><span>Username</span><input value="{{.Username}}" readonly aria-readonly="true"></label>
<span class="muted" style="display:block;font-size:11.5px;margin:-8px 0 var(--space-4)">Sign-in identity &#183; changed by an admin</span>
<div class="kv"><div class="k">Role</div><div><span class="badge">{{.Role}}</span></div></div>
<div class="kv"><div class="k">Member since</div><div class="mono">{{.CreatedISO}}</div></div>
</div>

<div class="section" style="margin-bottom:0">
<div class="microlabel">Credentials</div>
<h2>Password &amp; two-factor</h2>
{{if .PwError}}<div class="error">{{.PwError}}</div>{{end}}
<form method="post" action="/profile/password">
<label><span>Current password</span><input type="password" name="current_password" autocomplete="current-password" required></label>
<label style="margin-bottom:6px"><span>New password</span><input type="password" name="new_password" autocomplete="new-password" required></label>
<span class="muted" style="display:block;font-size:11.5px;margin-bottom:var(--space-4)">12+ characters; a passphrase beats complexity</span>
<button class="secondary" type="submit">Change password</button>
</form>
<div style="display:flex;align-items:center;gap:10px;flex-wrap:wrap;padding-top:var(--space-4);margin-top:var(--space-4);border-top:1px solid var(--hairline)">
{{if .TotpEnabled}}
<span class="badge" style="color:var(--ok);border-color:var(--ok-border);display:inline-flex;align-items:center;gap:5px"><span class="dot" style="background:var(--ok);width:6px;height:6px"></span>two-factor enabled</span>
<span class="mono muted" style="font-size:11.5px">TOTP</span>
{{else}}
<span class="badge off">two-factor off</span>
<span class="muted" style="font-size:11.5px">Add a second factor to your sign-in.</span>
<form method="post" action="/account/totp/enable" style="margin-left:auto"><button class="secondary" type="submit">Enable two-factor</button></form>
{{end}}
</div>
</div>

</div>

<div class="section" style="margin-top:var(--space-5)">
<div class="rulehead">
<div><div class="microlabel">Sessions</div><h2 style="margin:0">Signed in right now</h2></div>
<a class="btn secondary" href="/profile?endsession=1">End this session</a>
</div>
<table>
<thead><tr><th>Device</th><th>IP</th><th>Last active</th></tr></thead>
<tbody>
<tr>
<td><span style="display:inline-flex;align-items:center;gap:8px"><span style="font-weight:500;color:var(--ink)">{{.SessionDevice}}</span> <span class="badge" style="color:var(--link);border-color:var(--accent-soft);background:var(--accent-soft)">this device</span></span></td>
<td class="mono">{{.SessionIP}}</td>
<td class="mono">now</td>
</tr>
</tbody>
</table>
<p class="muted" style="margin:var(--space-3) 0 0;font-size:12px">Sessions on this deployment are stateless signed cookies with no central registry, so other devices are not individually listed. Ending this session signs you out here; a session elsewhere lapses when it expires.</p>
</div>

<div class="section">
<div class="rulehead">
<div><div class="microlabel">Automation</div><h2 style="margin:0">Personal API tokens</h2></div>
<a class="btn" href="/profile?new=1">New token</a>
</div>
{{if .Tokens}}
<table>
<thead><tr><th>Name</th><th>Token</th><th>Created</th><th>Last used</th><th></th></tr></thead>
<tbody>
{{range .Tokens}}<tr>
<td><span style="font-weight:500;color:var(--ink)">{{.Name}}</span></td>
<td class="mono">{{.Prefix}}</td>
<td class="mono">{{.Created}}</td>
<td class="mono">{{.Last}}</td>
<td style="text-align:right"><a class="iconbtn" href="/profile?revoke={{.ID}}" aria-label="Revoke {{.Name}}" title="Revoke"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75"><path d="M3 6h18"/><path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/></svg></a></td>
</tr>{{end}}
</tbody>
</table>
{{else}}
<div class="emptystate">
<h2>No personal tokens</h2>
<p>You have no personal API tokens. Mint one to let your own automation read this deployment with your access.</p>
</div>
{{end}}
<p class="muted" style="margin:var(--space-4) 0 0;font-size:12px">A token is scoped to your account and inherits your role. It is shown once at creation &#8212; Verge keeps only a hash.</p>
</div>

{{if or .CreateOpen .Minted}}
<a class="scrim" href="/profile" aria-label="Close"></a>
<div class="dialog-panel" role="dialog" aria-modal="true" aria-label="New personal token" style="position:fixed;top:14vh;left:50%;transform:translateX(-50%);z-index:42">
<div class="microlabel" style="margin-bottom:8px">New personal token</div>
{{if .Minted}}
<h2 style="margin:0 0 12px">Copy your token now</h2>
<div class="banner warn" style="margin:0 0 var(--space-4)">Shown once &#8212; store it now. Verge keeps only a hash, so it cannot be shown again.</div>
<div style="display:flex;align-items:center;gap:8px">
<code class="mono cvval" style="flex:1;min-width:0;word-break:break-all;background:var(--sunken);border:1px solid var(--hairline);border-radius:var(--r-sm);padding:var(--space-3)">{{.Minted}}</code>
<button type="button" class="btn secondary" style="flex:none" onclick="var v=this.parentNode.querySelector('.cvval').textContent;if(navigator.clipboard){navigator.clipboard.writeText(v);this.textContent='Copied';}">Copy</button>
</div>
<div class="dialog-actions"><a class="btn" href="/profile">Done</a></div>
{{else}}
<h2 style="margin:0 0 4px">New personal token</h2>
<p class="muted" style="margin:0 0 var(--space-4);font-size:12.5px">Scoped to your account; inherits your role.</p>
{{if .TokError}}<div class="error">{{.TokError}}</div>{{end}}
<form method="post" action="/profile/tokens">
<label><span>Token name</span><input name="name" value="{{.TokName}}" placeholder="laptop-cli" autocomplete="off" spellcheck="false" autofocus required></label>
<div class="dialog-actions">
<a class="btn ghost" href="/profile">Cancel</a>
<button type="submit">Create token</button>
</div>
</form>
{{end}}
</div>
{{end}}

{{if .RevokeID}}
<a class="scrim" href="/profile" aria-label="Cancel"></a>
<div class="dialog-panel" role="dialog" aria-modal="true" aria-label="Revoke {{.RevokeName}}" style="position:fixed;top:14vh;left:50%;transform:translateX(-50%);z-index:42">
<div class="microlabel" style="margin-bottom:8px">Revoke token</div>
<h2 style="margin:0 0 8px">Revoke {{.RevokeName}}</h2>
<p style="margin:0 0 4px">Any automation using this token stops working at once.</p>
<p class="muted" style="margin:0 0 var(--space-4)">This cannot be undone &#8212; a revoked token is never recoverable, only re-minted.</p>
{{if .RevokeErr}}<div class="error">{{.RevokeErr}}</div>{{end}}
<form method="post" action="/profile/tokens/revoke">
<input type="hidden" name="id" value="{{.RevokeID}}">
<label><span>Type <span class="mono">{{.RevokeName}}</span> to confirm</span><input name="confirm_name" autocomplete="off" spellcheck="false" autofocus required></label>
<div class="dialog-actions">
<a class="btn ghost" href="/profile">Cancel</a>
<button class="danger" type="submit">Revoke token</button>
</div>
</form>
</div>
{{end}}

{{if .EndSession}}
<a class="scrim" href="/profile" aria-label="Cancel"></a>
<div class="dialog-panel" role="dialog" aria-modal="true" aria-label="End this session" style="position:fixed;top:14vh;left:50%;transform:translateX(-50%);z-index:42">
<div class="microlabel" style="margin-bottom:8px">End session</div>
<h2 style="margin:0 0 8px">End this session</h2>
<p style="margin:0 0 4px">You are signed out on this device and returned to sign-in.</p>
<p class="muted" style="margin:0 0 var(--space-4)">A session on another device is not ended here &#8212; it lapses when it expires.</p>
<div class="dialog-actions">
<a class="btn ghost" href="/profile">Cancel</a>
<form method="post" action="/profile/session/revoke" style="margin:0"><button class="danger" type="submit">End session</button></form>
</div>
</div>
{{end}}

</main>
{{template "foot" .}}{{end}}
`
