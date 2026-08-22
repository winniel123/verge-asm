package main

import "html/template"

// Account surface — the account details + admin invite/TOTP form that lived on
// `/` before the Dashboard took the root route (V2 console map #275, #277). It is
// rendered at a temporary `GET /account` and by the four account handlers
// (createAccount / totpEnable-flow / renderFormError) until #281 folds it into
// Settings → access and repoints those handlers there (T10). The markup is the
// former `home` block verbatim; only the block name changed. It carries no
// NavActive, so no nav pill lights for the account view.
var _ = template.Must(tmpl.Parse(accountTemplates))

const accountTemplates = `
{{define "account"}}{{template "head" .}}
{{template "chrome" .}}
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
`
