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
.nav { display: flex; gap: var(--space-5); }
.nav a { text-decoration: none; color: var(--ink); font-weight: 600; }
.nav a:hover { color: var(--accent); }
table { width: 100%; border-collapse: collapse; }
th { text-align: left; font-family: var(--mono); font-size: 10px; font-weight: 600;
  text-transform: uppercase; letter-spacing: 0.06em; color: var(--muted);
  padding: var(--space-3) var(--space-4) var(--space-3) 0; border-bottom: 2px solid var(--ink); }
td { padding: var(--space-3) var(--space-4) var(--space-3) 0; border-bottom: 1px solid var(--hairline);
  vertical-align: top; }
.seedform { display: flex; gap: var(--space-4); align-items: flex-end; flex-wrap: wrap; }
.seedform label { margin-bottom: 0; }
.seedform .scope { min-width: 280px; }
.custody-head { display: flex; justify-content: space-between; align-items: flex-start;
  gap: var(--space-4); flex-wrap: wrap; }
.custody-head .scopename { font-size: 14px; }
.custody-head form { margin: 0; }
.badge.off { color: var(--muted); border-color: var(--hairline); }
.census { border: 1px solid var(--hairline); background: var(--paper);
  padding: var(--space-4); margin-top: var(--space-4); }
.census p { margin: var(--space-3) 0; }
input[type=checkbox] { width: auto; }
label.check { display: inline-flex; align-items: center; gap: 6px; margin-bottom: 0; }
label.check span { display: inline; margin-bottom: 0; }
.classes { display: flex; gap: var(--space-4); margin-bottom: var(--space-4); }
.inlineform { display: flex; gap: var(--space-3); align-items: center; }
.inlineform select, .inlineform input { width: auto; }
.muted { color: var(--muted); }
details.edit { margin-top: var(--space-3); }
details.edit summary { cursor: pointer; font-family: var(--mono); font-size: 11px;
  color: var(--accent); }
details.edit .section { margin-top: var(--space-3); margin-bottom: 0; }
.dial { display: flex; gap: var(--space-4); align-items: flex-end; flex-wrap: wrap; }
.dial label { margin-bottom: 0; min-width: 220px; }
.dial .unit { color: var(--muted); font-family: var(--mono); font-size: 11px; }
.searchbar { display: flex; gap: var(--space-4); align-items: flex-end; margin-bottom: var(--space-5); }
.searchbar label { margin-bottom: 0; }
.searchbar .grow { flex: 1; }
ol.chain { list-style: none; margin: var(--space-4) 0 0; padding: 0; }
ol.chain li { position: relative; padding: 0 0 var(--space-4) var(--space-5);
  border-left: 2px solid var(--ink); margin-left: 5px; }
ol.chain li:last-child { padding-bottom: 0; border-left-color: transparent; }
ol.chain li::before { content: ""; position: absolute; left: -6px; top: 3px;
  width: 8px; height: 8px; background: var(--ink); }
ol.chain .chainval { margin: 2px 0; }
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

{{define "chrome"}}<div class="header">` + wordmark + `
<div class="row">
<nav class="nav"><a href="/">Home</a><a href="/subjects">Subjects</a><a href="/seeds">Seeds</a><a href="/coverage">Coverage</a>{{if .IsAdmin}}<a href="/settings">Settings</a>{{end}}</nav>
<form method="post" action="/logout"><button class="secondary" type="submit">Sign out</button></form>
</div>
</div>{{end}}

{{define "home"}}{{template "head" .}}
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

{{define "forbidden"}}{{template "head" .}}
{{template "chrome" .}}
<main>
<div class="section">
<div class="microlabel">Not permitted</div>
<h2>Admin only</h2>
<p>{{.Message}}</p>
</div>
</main>
{{template "foot" .}}{{end}}

{{define "seeds"}}{{template "head" .}}
{{template "chrome" .}}
<main>
{{if .Notice}}<div class="notice">{{.Notice}}</div>{{end}}
<div class="microlabel">Declared · seeds</div>
<h1>Seeds</h1>
<p>A seed is where you assert your estate ends: a name scope — a registrable domain — or an
address scope — a CIDR block of up to {{.AddressCap}} addresses.</p>

{{if .IsAdmin}}
<div class="section">
<h2>Declare a scope</h2>
{{if .FormError}}<div class="error">{{.FormError}}</div>{{end}}
<form method="post" action="/seeds" class="seedform">
<label><span>Scope type</span><select name="kind">
<option value="name"{{if ne .FormKind "address"}} selected{{end}}>name</option>
<option value="address"{{if eq .FormKind "address"}} selected{{end}}>address</option>
</select></label>
<label class="scope"><span>Scope</span><input class="scope" name="scope" value="{{.FormScope}}" placeholder="example.com or 203.0.113.0/24" autocomplete="off" required></label>
<button type="submit">Declare</button>
</form>
</div>
{{end}}

<div class="section">
<h2>Declared scopes</h2>
{{if .Seeds}}
<table>
<thead><tr><th>Type</th><th>Scope</th><th>Declared by</th><th>Declared</th></tr></thead>
<tbody>
{{range .Seeds}}<tr>
<td><span class="badge">{{if .IsAddress}}address{{else}}name{{end}}</span></td>
<td class="mono">{{.Scope}}</td>
<td class="mono">{{.By}}</td>
<td class="mono">{{.At}}</td>
</tr>{{end}}
</tbody>
</table>
{{else}}
<div class="microlabel">No scopes declared</div>
<p>Nothing is declared yet. Declare a domain or a CIDR block to set where your estate begins.</p>
{{end}}
</div>

<div class="microlabel">Declared · custody extension</div>
<p>A custody extension declares that the addresses your name scopes resolve to are yours, and so
under your control. It is off by default and declared once per name scope — one act, never a queue
of addresses to approve. Its coverage is recomputed from measured resolution, stopping where the
chain leaves the declared zone, so there is no list to maintain.</p>

{{if .CustodyError}}<div class="error">{{.CustodyError}}</div>{{end}}
{{if .CustodyScopes}}
{{range .CustodyScopes}}
<div class="section">
<div class="custody-head">
<div>
<div class="microlabel">Name scope</div>
<div class="mono scopename">{{.Scope}}</div>
</div>
<div class="row">
{{if .CustodyExtension}}<span class="badge">extension on</span>{{else}}<span class="badge off">off</span>{{end}}
{{if $.IsAdmin}}
<form method="post" action="/seeds/custody">
<input type="hidden" name="id" value="{{.ID}}">
<input type="hidden" name="extend" value="{{if .CustodyExtension}}false{{else}}true{{end}}">
<button class="secondary" type="submit">{{if .CustodyExtension}}Withdraw{{else}}Declare extension{{end}}</button>
</form>
{{end}}
</div>
</div>
{{if .CustodyExtension}}
<div class="census">
<div class="microlabel">Covered addresses · census</div>
<p>Display only. Once resolution measurement runs, this lists the addresses your names currently
resolve into. There is no total to reach — how many addresses it ought to cover is completeness of
your estate, which only you know — and nothing here to approve: the extension covers what it
computes.</p>
<div class="microlabel">No addresses measured yet</div>
</div>
{{end}}
</div>
{{end}}
{{else}}
<div class="section">
<div class="microlabel">No name scopes</div>
<p>A custody extension is a property of a name scope. Declare a name scope above, then extend
custody to the addresses it resolves into.</p>
</div>
{{end}}

<div class="microlabel">Declared · zone files</div>
<p>Your own zone file is ground truth: the estate as you declare it, not as it resolves. Upload it
here — it is stored so both services can read it, and it is evidence, not a secret. Uploading is the
supply act, so its instant is recorded now; the zone scan restates the file at that instant, never at
whatever later time the worker reads it. Re-export on your own cadence and upload again — a new upload
is a new supply, shipped monthly by default.</p>

{{if .IsAdmin}}
{{if .NameScopes}}
<div class="section">
<h2>Upload a zone file</h2>
{{if .ZoneError}}<div class="error">{{.ZoneError}}</div>{{end}}
<form method="post" action="/seeds/zone" enctype="multipart/form-data" class="seedform">
<label><span>Name scope</span><select name="seed_id">
{{range .NameScopes}}<option value="{{.ID}}">{{.Scope}}</option>{{end}}
</select></label>
<label class="scope"><span>Zone file</span><input class="scope" type="file" name="zonefile" required></label>
<button type="submit">Upload</button>
</form>
</div>
{{else}}
<div class="section">
<div class="microlabel">No name scopes</div>
<p>A zone file is attached to a name scope. Declare a name scope above, then upload its zone file.</p>
</div>
{{end}}

<div class="section">
<h2>Re-supply interval</h2>
<p>How often you promise to re-export. The scan reports the file as stale past this interval, so set
it to your real export cadence rather than a hope.</p>
{{if .ZoneIntervalError}}<div class="error">{{.ZoneIntervalError}}</div>{{end}}
<form method="post" action="/seeds/zone/interval">
<div class="dial">
<label><span>Interval</span><input name="interval_days" inputmode="numeric" value="{{.ZoneIntervalDays}}" required><span class="unit">days</span></label>
<button type="submit">Save interval</button>
</div>
</form>
</div>
{{end}}

{{if .ZoneScopes}}
<div class="section">
<h2>Supplied zone files</h2>
<table>
<thead><tr><th>Name scope</th><th>Supplied</th><th>Uploaded by</th><th>Size</th></tr></thead>
<tbody>
{{range .ZoneScopes}}<tr>
<td class="mono">{{.Domain}}</td>
<td class="mono">{{if .HasFile}}{{.SuppliedAt}}{{else}}<span class="muted">none supplied</span>{{end}}</td>
<td class="mono">{{if .HasFile}}{{.By}}{{else}}<span class="muted">—</span>{{end}}</td>
<td class="mono">{{if .HasFile}}{{.Bytes}} bytes{{else}}<span class="muted">—</span>{{end}}</td>
</tr>{{end}}
</tbody>
</table>
</div>
{{end}}

<div class="microlabel">Declared · exclusions</div>
<p>An exclusion draws the boundary inwards: an exact name, a name subtree, or an address scope
you declare is <em>not yours</em>. Excluding a name that still resolves is legal — <em>not
mine</em> is a different claim from <em>not there</em> — and an excluded name is no longer queried.</p>

{{if .IsAdmin}}
<div class="section">
<h2>Declare an exclusion</h2>
{{if .ExclError}}<div class="error">{{.ExclError}}</div>{{end}}
<form method="post" action="/exclusions" class="seedform">
<label><span>Exclusion type</span><select name="kind">
<option value="name"{{if eq .ExclKind "name"}} selected{{end}}>name</option>
<option value="subtree"{{if eq .ExclKind "subtree"}} selected{{end}}>subtree</option>
<option value="address"{{if eq .ExclKind "address"}} selected{{end}}>address</option>
</select></label>
<label class="scope"><span>Value</span><input class="scope" name="value" value="{{.ExclValue}}" placeholder="api.example.com or 203.0.113.5" autocomplete="off" required></label>
<button type="submit">Exclude</button>
</form>
</div>
{{end}}

<div class="section">
<h2>Declared exclusions</h2>
{{if .Exclusions}}
<table>
<thead><tr><th>Type</th><th>Value</th><th>Declared by</th><th>Declared</th>{{if .IsAdmin}}<th></th>{{end}}</tr></thead>
<tbody>
{{range .Exclusions}}<tr>
<td><span class="badge">{{.Kind}}</span></td>
<td class="mono">{{.Value}}</td>
<td class="mono">{{.By}}</td>
<td class="mono">{{.At}}</td>
{{if $.IsAdmin}}<td><form method="post" action="/exclusions/delete"><input type="hidden" name="id" value="{{.ID}}"><button class="secondary" type="submit">Un-exclude</button></form></td>{{end}}
</tr>{{end}}
</tbody>
</table>
{{else}}
<div class="microlabel">No exclusions declared</div>
<p>Nothing is excluded. Everything inside your declared scopes is yours.</p>
{{end}}
</div>

<div class="microlabel">Declared · probers</div>
<p>Provisioning a prober declares <em>this vantage is on the internet</em>. You supply the host, port,
and a non-root username; the instance generates the SSH keypair on the worker volume and exposes only
the public half — install it on the prober host. The private key never leaves the instance.</p>

{{if .IsAdmin}}
<div class="section">
<h2>Provision a prober</h2>
{{if .ProberError}}<div class="error">{{.ProberError}}</div>{{end}}
<form method="post" action="/probers" class="seedform">
<label><span>Host</span><input name="host" value="{{.ProberHost}}" placeholder="prober.example.com" autocomplete="off" required></label>
<label><span>Port</span><input name="port" value="{{.ProberPort}}" placeholder="22" autocomplete="off"></label>
<label><span>Username</span><input name="username" value="{{.ProberUser}}" placeholder="scanner" autocomplete="off" required></label>
<button type="submit">Provision</button>
</form>
</div>
{{end}}

<div class="section">
<h2>Provisioned probers</h2>
{{if .Probers}}
<table>
<thead><tr><th>Endpoint</th><th>Username</th><th>Availability</th><th>Public key</th><th>Provisioned by</th><th>Provisioned</th></tr></thead>
<tbody>
{{range .Probers}}<tr>
<td class="mono">{{.Endpoint}}</td>
<td class="mono">{{.Username}}</td>
<td><span class="badge">{{.Availability}}</span></td>
<td>{{if .KeySet}}<span class="badge">set</span><div class="secret" style="margin-top:8px">{{.PublicKey}}</div>{{else}}<span class="microlabel">not set</span>{{end}}</td>
<td class="mono">{{.By}}</td>
<td class="mono">{{.At}}</td>
</tr>{{end}}
</tbody>
</table>
{{else}}
<div class="microlabel">No probers provisioned</div>
<p>No vantage is on the internet yet. Provision a prober to declare one — until then, exposure cannot be measured.</p>
{{end}}
</div>
</main>
{{template "foot" .}}{{end}}

{{define "settings"}}{{template "head" .}}
{{template "chrome" .}}
<main>
<div class="microlabel">Operator · settings</div>
<h1>Settings</h1>
<p>Your dials: the accounts that may sign in, the channels a message is carried out
over, and how long each corpus stays readable. Every change here is an admin act.</p>

<div class="microlabel">Accounts</div>
<div class="section">
<h2>Accounts</h2>
{{if .RoleError}}<div class="error">{{.RoleError}}</div>{{end}}
<table>
<thead><tr><th>Username</th><th>Role</th><th>Two-factor</th><th>Created</th><th>Role</th></tr></thead>
<tbody>
{{range .Accounts}}<tr>
<td class="mono">{{.Username}}{{if .IsSelf}} <span class="muted">(you)</span>{{end}}</td>
<td><span class="badge">{{.Role}}</span></td>
<td>{{if .TotpEnabled}}<span class="badge">on</span>{{else}}<span class="muted">off</span>{{if .IsSelf}}
<form method="post" action="/account/totp/enable" style="display:inline"><button class="secondary" type="submit">Enable</button></form>{{end}}{{end}}</td>
<td class="mono">{{.At}}</td>
<td>
<form method="post" action="/settings/accounts/role" class="inlineform">
<input type="hidden" name="id" value="{{.ID}}">
<select name="role"><option value="admin"{{if eq .Role "admin"}} selected{{end}}>admin</option><option value="viewer"{{if eq .Role "viewer"}} selected{{end}}>viewer</option></select>
<button class="secondary" type="submit">Save</button>
</form>
</td>
</tr>{{end}}
</tbody>
</table>
</div>

<div class="section">
<h2>Invite an account</h2>
{{if .AcctError}}<div class="error">{{.AcctError}}</div>{{end}}
<form method="post" action="/settings/accounts">
<label><span>Username</span><input name="username" value="{{.AcctUsername}}" autocomplete="off" required></label>
<label><span>Password</span><input name="password" type="password" autocomplete="new-password" required></label>
<label><span>Role</span><select name="role"><option value="admin"{{if eq .AcctRole "admin"}} selected{{end}}>admin</option><option value="viewer"{{if eq .AcctRole "viewer"}} selected{{end}}>viewer</option></select></label>
<button type="submit">Create account</button>
</form>
</div>

<div class="microlabel">Channels</div>
<div class="section">
<h2>Channels</h2>
<p>A channel is where messages go: an absolute <code>https</code> URL, an optional
signing secret, and the subset of classes it carries. None ships configured, and a
channel is one-way — it grants no read of anything. Delivery itself lands later; this
persists the declaration.</p>
{{if .ChanError}}<div class="error">{{.ChanError}}</div>{{end}}
{{if .Channels}}
<table>
<thead><tr><th>URL</th><th>Classes</th><th>Secret</th><th>State</th><th>Declared by</th><th></th></tr></thead>
<tbody>
{{range .Channels}}<tr>
<td class="mono">{{.URL}}</td>
<td>{{if .Drift}}<span class="badge">drift</span> {{end}}{{if .Coverage}}<span class="badge">coverage</span> {{end}}{{if .Clock}}<span class="badge">clock</span>{{end}}</td>
<td>{{if .HasSecret}}<span class="badge">set</span>{{else}}<span class="muted">not set</span>{{end}}</td>
<td>{{if .Enabled}}<span class="badge">enabled</span>{{else}}<span class="muted">disabled</span>{{end}}</td>
<td class="mono">{{.By}}<br><span class="muted">{{.At}}</span></td>
<td>
<details class="edit">
<summary>Edit</summary>
<div class="section">
<form method="post" action="/settings/channels/update">
<input type="hidden" name="id" value="{{.ID}}">
<label><span>URL</span><input name="url" value="{{.URL}}" autocomplete="off" required></label>
<div class="classes">
<label class="check"><input type="checkbox" name="drift"{{if .Drift}} checked{{end}}><span>drift</span></label>
<label class="check"><input type="checkbox" name="coverage"{{if .Coverage}} checked{{end}}><span>coverage</span></label>
<label class="check"><input type="checkbox" name="clock"{{if .Clock}} checked{{end}}><span>clock</span></label>
</div>
<label class="check"><input type="checkbox" name="enabled"{{if .Enabled}} checked{{end}}><span>enabled</span></label>
<label><span>Replace secret</span><input name="secret" type="password" autocomplete="off" placeholder="{{if .HasSecret}}set — leave blank to keep{{else}}not set — leave blank for none{{end}}"></label>
{{if .HasSecret}}<label class="check"><input type="checkbox" name="clear_secret"><span>clear the secret</span></label>{{end}}
<div class="row" style="margin-top:12px"><button type="submit">Save channel</button></div>
</form>
<form method="post" action="/settings/channels/delete" style="margin-top:12px"><input type="hidden" name="id" value="{{.ID}}"><button class="secondary" type="submit">Delete channel</button></form>
</div>
</details>
</td>
</tr>{{end}}
</tbody>
</table>
{{else}}
<div class="microlabel">No channels declared</div>
<p>Nothing is configured, so every message is written and rendered in the store and
carried nowhere else. Declare a channel to have messages POSTed out.</p>
{{end}}
</div>

<div class="section">
<h2>Declare a channel</h2>
<form method="post" action="/settings/channels">
<label><span>URL</span><input name="url" value="{{.ChanURL}}" placeholder="https://hooks.example.com/verge" autocomplete="off" required></label>
<div class="classes">
<label class="check"><input type="checkbox" name="drift"{{if .ChanDrift}} checked{{end}}><span>drift</span></label>
<label class="check"><input type="checkbox" name="coverage"{{if .ChanCoverage}} checked{{end}}><span>coverage</span></label>
<label class="check"><input type="checkbox" name="clock"{{if .ChanClock}} checked{{end}}><span>clock</span></label>
</div>
<label><span>Secret (optional)</span><input name="secret" type="password" autocomplete="off" placeholder="signs the body — never shown again"></label>
<button type="submit">Declare channel</button>
</form>
</div>

<div class="microlabel">Retention</div>
<div class="section">
<h2>Retention dials</h2>
<p>Retention is what may still be read, never age. The floors that bound these dials
land with later work; for now each floors at zero, where zero means no operator floor.</p>
{{if .RetError}}<div class="error">{{.RetError}}</div>{{end}}
<form method="post" action="/settings/retention">
<div class="dial">
<label><span>Observation-currency floor</span><input name="observation_currency_days" inputmode="numeric" value="{{if .RetError}}{{.RetObs}}{{else}}{{.Retention.ObservationCurrencyDays}}{{end}}" required><span class="unit">days</span></label>
<label><span>Dispatch floor</span><input name="dispatch_cadence_multiple" inputmode="numeric" value="{{if .RetError}}{{.RetDispatch}}{{else}}{{.Retention.DispatchCadenceMultiple}}{{end}}" required><span class="unit">× cadence</span></label>
<button type="submit">Save dials</button>
</div>
</form>
{{if .Retention.UpdatedAt}}<p class="muted" style="margin-top:12px">Last changed {{.Retention.UpdatedAt}}{{if .Retention.UpdatedBy}} by <span class="mono">{{.Retention.UpdatedBy}}</span>{{end}}.</p>{{end}}
</div>
</main>
{{template "foot" .}}{{end}}

{{define "subjects"}}{{template "head" .}}
{{template "chrome" .}}
<main>
<div class="microlabel">Observed · subjects</div>
<h1>Subjects</h1>
<p>Every Name currently in your estate. Only names exist here yet — addresses,
services and endpoints arrive with later measurement. There is no total: how many
names your estate ought to hold is its completeness, which only you know, so this
screen states none.</p>

<form method="get" action="/subjects" class="searchbar">
<label class="grow"><span>Search names</span><input name="q" value="{{.Search}}" placeholder="example.com" autocomplete="off"></label>
<button type="submit">Search</button>
{{if .Search}}<a class="btn secondary" href="/subjects" style="text-decoration:none">Clear</a>{{end}}
</form>

<div class="section">
<div class="microlabel">Name subjects</div>
{{if .Subjects}}
<table>
<thead><tr><th>Name</th><th>Resolution</th></tr></thead>
<tbody>
{{range .Subjects}}<tr>
<td><a class="mono" href="/subjects/{{.Name}}">{{.Name}}</a></td>
<td>{{if .Resolution}}<span class="badge">{{.Resolution}}</span>{{else}}<span class="muted">—</span>{{end}}</td>
</tr>{{end}}
</tbody>
</table>
{{else}}
<div class="microlabel">{{if .Search}}No name matches{{else}}No names yet{{end}}</div>
<p>{{if .Search}}No current name matches that search. A withdrawn name is reached by its exact key, never by browsing — search the full name.{{else}}No name has been measured into the estate yet. Declare a name scope on Seeds, then let the dns Scan resolve it.{{end}}</p>
{{end}}
</div>
</main>
{{template "foot" .}}{{end}}

{{define "subject"}}{{template "head" .}}
{{template "chrome" .}}
<main>
{{with .Subject}}
<div class="microlabel">Observed · Name</div>
<h1 class="mono">{{.Name}}</h1>
{{if .Withdrawn}}
<div class="notice">This name is withdrawn — it names a population of no current member. Its timelines are closed. It is reached by its own key and never appears in the listing.</div>
{{end}}

<div class="section">
<div class="microlabel">Why is this here</div>
<h2>Citation chain</h2>
<p>Following a subject's citations backwards always terminates at a Seed you declared — that is what makes "why is this here" answerable for everything in the estate.</p>
<ol class="chain">
{{range .Citation}}<li>
<div class="microlabel">{{.Label}}</div>
<div class="mono chainval">{{.Value}}</div>
{{if .Detail}}<div class="muted">{{.Detail}}</div>{{end}}
</li>{{end}}
</ol>
{{if not .CitationTerminated}}<p class="muted">The chain does not reach a declared Seed. That is an integrity gap, not a normal state — every subject in the estate should trace back to a scope you declared.</p>{{end}}
</div>

<div class="section">
<div class="microlabel">Current · resolution</div>
<h2>Resolution</h2>
{{if .Resolution}}
<div class="kv"><div class="k">Outcome</div><div><span class="badge">{{.Resolution}}</span></div></div>
{{if .Addresses}}<div class="kv"><div class="k">Addresses</div><div class="mono">{{range .Addresses}}{{.}}<br>{{end}}</div></div>{{end}}
{{else}}<p class="muted">No resolution value recorded.</p>{{end}}
</div>

<div class="section">
<div class="microlabel">Timelines</div>
<h2>Current and closed timelines</h2>
<p class="muted">This subject's facet timelines — current values, any gaps with their stated cause, and (once withdrawn) its closed history — render here. Wired up by ticket 10.</p>
</div>

<div class="section">
<div class="microlabel">Rules</div>
<h2>Rules over this subject</h2>
<p class="muted">Every rule whose predicate domain includes this subject renders here, each carrying its own versioned verdict. Wired up by ticket 22.</p>
</div>
{{end}}
</main>
{{template "foot" .}}{{end}}

{{define "subject-missing"}}{{template "head" .}}
{{template "chrome" .}}
<main>
<div class="microlabel">No such subject</div>
<h1 class="mono">{{.Name}}</h1>
<div class="section">
<p>No subject is keyed under that name. Nothing has ever measured it into the
estate — this is not a withdrawn subject, which would still be reachable here by
its own key.</p>
<p><a href="/subjects">Back to subjects</a></p>
</div>
</main>
{{template "foot" .}}{{end}}
`))
