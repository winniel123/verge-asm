package main

import "html/template"

// pageCSS is a self-contained slice of the Verge ASM design system: the tokens
// and the handful of component rules these auth pages need. The web binary is
// distroless with no mounted asset path, so the stylesheet is inlined rather
// than linked. Values are the system's own — warm paper, near-black ink, one
// working blue, IBM Plex Mono for technical values, 0px radius, hard offset
// shadow, 2px ink rules — kept in step with design-system/tokens.
const pageCSS = `
:root {
  --paper: #f7f7f4; --ink: #16160f; --surface: #ffffff; --accent: #2d4fd4;
  --hairline: #d8d8d0; --muted: #868e96; --danger: #c92a2a;
  --sans: -apple-system, BlinkMacSystemFont, "Helvetica Neue", Helvetica, Arial, sans-serif;
  --mono: "IBM Plex Mono", ui-monospace, SFMono-Regular, Menlo, monospace;
  --space-3: 12px; --space-4: 16px; --space-5: 24px; --space-6: 32px;
}
* { box-sizing: border-box; }
body { margin: 0; background: var(--paper); color: var(--ink);
  font-family: var(--sans); font-size: 13px; line-height: 1.55; }
a { color: var(--accent); }
code, .mono { font-family: var(--mono); }
.microlabel { font-family: var(--mono); font-size: 10px; font-weight: 600;
  text-transform: uppercase; letter-spacing: 0.06em; color: var(--muted); }
.header { border-bottom: 2px solid var(--ink); padding: var(--space-4) var(--space-5);
  display: flex; justify-content: space-between; align-items: center; }
.wordmark { font-weight: 700; font-size: 15px; }
.wordmark .chip { font-family: var(--mono); font-size: 11px; font-weight: 600;
  border: 1px solid var(--ink); padding: 1px 4px; margin-left: 6px; }
.center { min-height: 100vh; display: flex; align-items: center; justify-content: center;
  padding: var(--space-6); }
main { padding: var(--space-6); max-width: 760px; margin: 0 auto; }
.card { background: var(--surface); border: 1px solid var(--ink);
  box-shadow: 6px 6px 0 rgba(22,22,15,.1); padding: var(--space-6); width: 360px; }
.section { background: var(--surface); border: 1px solid var(--hairline);
  padding: var(--space-5); margin-bottom: var(--space-5); }
h1 { font-size: 18px; margin: 0 0 var(--space-3); }
h2 { font-size: 14px; margin: 0 0 var(--space-4); }
p { margin: 0 0 var(--space-4); }
label { display: block; margin-bottom: var(--space-4); }
label span { display: block; margin-bottom: 4px; }
input, select { width: 100%; font-family: var(--mono); font-size: 13px;
  padding: 8px 10px; border: 1px solid var(--hairline); border-radius: 0;
  background: var(--paper); color: var(--ink); }
input:focus, select:focus { outline: 2px solid var(--accent); outline-offset: -2px; }
button, .btn { font-family: var(--sans); font-size: 13px; padding: 8px 16px;
  border: 1px solid var(--accent); background: var(--accent); color: #fff;
  border-radius: 0; cursor: pointer; }
button.secondary { background: var(--surface); color: var(--ink); border-color: var(--ink); }
.error { border: 1px solid var(--danger); color: var(--danger); padding: var(--space-3);
  margin-bottom: var(--space-4); font-size: 13px; }
.notice { border: 1px solid var(--accent); padding: var(--space-3);
  margin-bottom: var(--space-4); font-size: 13px; }
.badge { font-family: var(--mono); font-size: 10px; font-weight: 600; text-transform: uppercase;
  letter-spacing: 0.06em; border: 1px solid var(--ink); padding: 1px 6px; }
.kv { display: flex; gap: var(--space-4); margin-bottom: var(--space-3); }
.kv .k { color: var(--muted); width: 90px; }
.secret { font-family: var(--mono); word-break: break-all; background: var(--paper);
  border: 1px solid var(--hairline); padding: var(--space-3); margin-bottom: var(--space-4); }
.row { display: flex; gap: var(--space-3); align-items: center; }
`

// wordmark is the typed Verge ASM mark: sans "Verge" plus a mono "ASM" chip.
// There is no drawn logo (design system).
const wordmark = `<span class="wordmark">Verge<span class="chip">ASM</span></span>`

var tmpl = template.Must(template.New("").Parse(`
{{define "head"}}<!doctype html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}} · Verge ASM</title><style>` + pageCSS + `</style></head><body>{{end}}
{{define "foot"}}</body></html>{{end}}

{{define "setup"}}{{template "head" .}}
<div class="center"><div class="card">
<div class="microlabel">First-run setup</div>
<h1>Create the first admin</h1>
<p>This creates the only account that exists. The setup token was written to the
<code>web</code> container logs on first boot.</p>
{{if .Error}}<div class="error">{{.Error}}</div>{{end}}
<form method="post" action="/setup">
<label><span>Setup token</span><input name="token" value="{{.Token}}" autocomplete="off" required></label>
<label><span>Username</span><input name="username" autocomplete="username" required></label>
<label><span>Password</span><input name="password" type="password" autocomplete="new-password" required></label>
<button type="submit">Create admin</button>
</form>
</div></div>{{template "foot" .}}{{end}}

{{define "login"}}{{template "head" .}}
<div class="center"><div class="card">
<div class="microlabel">Sign in</div>
<h1>Verge ASM</h1>
{{if .Error}}<div class="error">{{.Error}}</div>{{end}}
<form method="post" action="/login">
<label><span>Username</span><input name="username" autocomplete="username" required></label>
<label><span>Password</span><input name="password" type="password" autocomplete="current-password" required></label>
<button type="submit">Sign in</button>
</form>
</div></div>{{template "foot" .}}{{end}}

{{define "totp"}}{{template "head" .}}
<div class="center"><div class="card">
<div class="microlabel">Two-factor</div>
<h1>Enter your code</h1>
<p>Your account has a time-based one-time password enabled. Enter the current
6-digit code from your authenticator.</p>
{{if .Error}}<div class="error">{{.Error}}</div>{{end}}
<form method="post" action="/login/totp">
<label><span>6-digit code</span><input name="code" inputmode="numeric" autocomplete="one-time-code" required></label>
<button type="submit">Verify</button>
</form>
</div></div>{{template "foot" .}}{{end}}

{{define "totp-enroll"}}{{template "head" .}}
<div class="center"><div class="card">
<div class="microlabel">Two-factor · enrol</div>
<h1>Enable two-factor</h1>
<p>Add this secret to your authenticator, then confirm with the current code.
Two-factor is not active until you confirm.</p>
<div class="secret">{{.Secret}}</div>
<p class="microlabel">otpauth URI</p>
<div class="secret">{{.OtpauthURI}}</div>
{{if .Error}}<div class="error">{{.Error}}</div>{{end}}
<form method="post" action="/account/totp/confirm">
<label><span>Current code</span><input name="code" inputmode="numeric" autocomplete="one-time-code" required></label>
<div class="row"><button type="submit">Confirm and enable</button>
<a class="btn secondary" href="/" style="text-decoration:none">Cancel</a></div>
</form>
</div></div>{{template "foot" .}}{{end}}

{{define "home"}}{{template "head" .}}
<div class="header">` + wordmark + `
<form method="post" action="/logout"><button class="secondary" type="submit">Sign out</button></form>
</div>
<main>
{{if .Notice}}<div class="notice">{{.Notice}}</div>{{end}}
<div class="section">
<div class="microlabel">Signed in</div>
<h2>Your account</h2>
<div class="kv"><div class="k">Username</div><div class="mono">{{.Account.Username}}</div></div>
<div class="kv"><div class="k">Role</div><div><span class="badge">{{.Account.Role}}</span></div></div>
<div class="kv"><div class="k">Two-factor</div><div>{{if .Account.TotpEnabled}}<span class="badge">on</span>{{else}}off{{end}}</div></div>
{{if not .Account.TotpEnabled}}
<form method="post" action="/account/totp/enable"><button type="submit">Enable two-factor</button></form>
{{end}}
</div>

{{if .IsAdmin}}
<div class="section">
<div class="microlabel">Admin · accounts</div>
<h2>Invite an account</h2>
{{if .FormError}}<div class="error">{{.FormError}}</div>{{end}}
<form method="post" action="/accounts">
<label><span>Username</span><input name="username" autocomplete="off" required></label>
<label><span>Password</span><input name="password" type="password" autocomplete="new-password" required></label>
<label><span>Role</span><select name="role"><option value="admin">admin</option><option value="viewer">viewer</option></select></label>
<button type="submit">Create account</button>
</form>
</div>
{{else}}
<div class="section">
<div class="microlabel">Viewer</div>
<p>You have read access. Account management is admin-only.</p>
</div>
{{end}}
</main>
{{template "foot" .}}{{end}}
`))
