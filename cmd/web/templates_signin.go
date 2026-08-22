package main

import "html/template"

// SignIn screen — the chrome-less auth surfaces (setup, login, TOTP verify,
// TOTP enrol). Canonical screen: SignIn. Ported verbatim; screen ticket T-SignIn
// rewrites the body against examples/console/SignIn.jsx.
var _ = template.Must(tmpl.Parse(signinTemplates))

const signinTemplates = `
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
`
