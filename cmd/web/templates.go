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
  letter-spacing: 0.06em; border: 1px solid var(--ink); padding: 1px 6px;
  display: inline-block; white-space: nowrap; }
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
.timeline { padding: var(--space-4) 0; border-top: 1px solid var(--hairline); }
.timeline:first-of-type { border-top: 0; }
.timeline .notice { margin-top: var(--space-3); }
table.closedspans { margin-top: var(--space-3); }
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
details.spanrecords > summary { cursor: pointer; display: inline; list-style: none; }
details.spanrecords > summary::-webkit-details-marker { display: none; }
details.spanrecords > summary::before { content: "\25b8"; font-family: var(--mono);
  color: var(--muted); margin-right: 6px; display: inline-block; }
details.spanrecords[open] > summary::before { content: "\25be"; }
table.records { width: auto; margin: var(--space-3) 0 0; }
table.records td { border-bottom: 1px solid var(--hairline); padding: 2px var(--space-4) 2px 0; }
table.records td.rrtype { width: 1%; white-space: nowrap; }
.invsubject { padding: var(--space-4) 0; border-top: 1px solid var(--hairline); }
.invsubject:first-of-type { border-top: 0; }
.invsubject .invkey { display: inline-block; margin-bottom: var(--space-3); font-weight: 600; }
table.invfacets { width: 100%; }
table.invfacets td { border-bottom: 0; padding: 2px var(--space-4) 2px 0; vertical-align: top; }
table.invfacets td.invfacet { width: 160px; white-space: nowrap; }
table.invfacets td.invsince { text-align: right; white-space: nowrap; width: 1%; }
.dial { display: flex; gap: var(--space-4); align-items: flex-end; flex-wrap: wrap; }
.dial label { margin-bottom: 0; min-width: 220px; }
.dial input { width: 96px; }
.dial .unit { display: inline; margin-left: 6px; color: var(--muted);
  font-family: var(--mono); font-size: 11px; }
.searchbar { display: flex; gap: var(--space-4); align-items: flex-end; margin-bottom: var(--space-5); }
.searchbar label { margin-bottom: 0; }
.searchbar .grow { flex: 1; }
ol.chain { list-style: none; margin: var(--space-4) 0 0; padding: 0; }
ol.chain li { position: relative; padding: 0 0 var(--space-4) var(--space-5); }
ol.chain li:last-child { padding-bottom: 0; }
ol.chain li::before { content: ""; position: absolute; left: 3px; top: 7px; bottom: -7px;
  width: 2px; background: var(--ink); }
ol.chain li:last-child::before { display: none; }
ol.chain li::after { content: ""; position: absolute; left: 0; top: 3px;
  width: 8px; height: 8px; background: var(--ink); }
ol.chain .chainval { margin: 2px 0; }
.rulehead { display: flex; justify-content: space-between; align-items: baseline;
  gap: var(--space-4); flex-wrap: wrap; margin-bottom: var(--space-4); }
.rulehead h2 { margin: 0; }
.rulehead .ver { white-space: nowrap; }
.members { display: flex; flex-direction: column; gap: var(--space-4); }
.mgroup { border: 1px solid var(--hairline); }
.mgroup-head { display: flex; align-items: baseline; gap: var(--space-3);
  padding: var(--space-3) var(--space-4); border-bottom: 1px solid var(--hairline);
  background: var(--paper); }
.mgroup-head .count { font-family: var(--mono); font-size: 13px; font-weight: 600; margin-left: auto; }
.mgroup-list { list-style: none; margin: 0; padding: 0; }
.mgroup-list li { padding: var(--space-3) var(--space-4); border-bottom: 1px solid var(--hairline); }
.mgroup-list li:last-child { border-bottom: none; }
.mgroup-empty { padding: var(--space-3) var(--space-4); color: var(--muted); }
.fullmute { padding: var(--space-4); border-left: 2px solid var(--ink); background: var(--paper); }
.fullmute p { margin: var(--space-3) 0 0; max-width: 78ch; }
.annoform { display: flex; gap: var(--space-4); align-items: flex-end; flex-wrap: wrap;
  margin-bottom: var(--space-5); }
.annoform label { margin-bottom: 0; }
.annoform label.grow { flex: 1; min-width: 220px; }
.annoform select { width: auto; }
table.annos { margin-top: var(--space-4); }
table.annos td form { margin: 0; }
.orphan { display: inline-block; margin-left: var(--space-3); font-family: var(--mono);
  font-size: 10px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.06em;
  color: var(--muted); border: 1px solid var(--hairline); padding: 1px 5px; }
.avail { display: flex; gap: var(--space-5); flex-wrap: wrap; margin-bottom: var(--space-5); }
.avail .k { color: var(--muted); margin-right: var(--space-3); }
.board { display: grid; grid-template-columns: 150px 1fr 1fr; border: 2px solid var(--ink);
  background: var(--surface); margin: var(--space-4) 0 var(--space-5); }
.board > div { border-right: 1px solid var(--hairline); border-bottom: 1px solid var(--hairline);
  padding: var(--space-4); }
.board .corner, .board .colhead, .board .rowhead { background: var(--paper); }
.board .corner { display: flex; flex-direction: column; justify-content: flex-end; }
.board .colhead, .board .rowhead { display: flex; align-items: flex-end; }
.board .cell .count { font-family: var(--mono); font-size: 22px; font-weight: 600;
  margin: 2px 0 var(--space-3); }
.board .cell.hot { box-shadow: inset 3px 0 0 var(--accent); }
.board .cell ul, .movedlist { list-style: none; margin: 0; padding: 0; }
.board .cell li { font-family: var(--mono); font-size: 11px; padding: 1px 0; }
.board .cell li a { text-decoration: none; }
.board .cell .none { color: var(--muted); }
.precond { border: 1px solid var(--ink); background: var(--surface);
  box-shadow: 6px 6px 0 rgba(22,22,15,.1); padding: var(--space-5); margin-bottom: var(--space-5); }
.precond h2 { margin-top: 0; }
.moved { border-left: 3px solid var(--accent); background: var(--paper);
  padding: var(--space-4); margin-bottom: var(--space-5); }
.movedlist li { font-family: var(--mono); font-size: 12px; padding: 2px 0; }
.msgnav { display: inline-flex; align-items: center; gap: 6px; text-decoration: none;
  color: var(--ink); font-weight: 600; }
.msgnav:hover { color: var(--accent); }
.msgnav .count { font-family: var(--mono); font-size: 10px; font-weight: 600;
  letter-spacing: 0.04em; background: var(--ink); color: #fff; padding: 1px 5px; min-width: 16px;
  text-align: center; }
.msglist { list-style: none; margin: var(--space-5) 0 0; padding: 0; }
.msgitem { border: 1px solid var(--hairline); background: var(--surface);
  padding: var(--space-4); margin-bottom: var(--space-4); border-left: 2px solid var(--ink); }
.msgitem.unread { border-left-color: var(--accent); }
.msgitem-head { display: flex; align-items: baseline; gap: var(--space-3); flex-wrap: wrap;
  margin-bottom: var(--space-3); }
.msgitem-head .cause { font-family: var(--mono); font-size: 10px; font-weight: 600;
  text-transform: uppercase; letter-spacing: 0.06em; color: var(--muted);
  border: 1px solid var(--hairline); padding: 1px 6px; }
.msgitem-head .when { margin-left: auto; font-family: var(--mono); font-size: 11px; color: var(--muted); }
.msgitem .headline { margin: 0 0 var(--space-3); }
.msgitem .rowlink { font-family: var(--mono); font-size: 12px; }
.msgitem .actions { margin-top: var(--space-3); }
.msgitem .actions form { display: inline; margin: 0; }
.msgcensus { list-style: none; margin: var(--space-3) 0 0; padding: var(--space-3) 0 0;
  border-top: 1px solid var(--hairline); }
.msgcensus li { padding: 2px 0; font-family: var(--mono); font-size: 12px; }
.msgcensus li .k { color: var(--muted); text-transform: uppercase; font-size: 10px;
  letter-spacing: 0.06em; margin-right: 6px; }
.receipt { border: 1px solid var(--ink); background: var(--surface);
  padding: var(--space-4); margin-top: var(--space-4); box-shadow: 3px 3px 0 rgba(22,22,15,.1); }
.receipt .microlabel { margin-bottom: var(--space-3); }
.receipt .headline { font-family: var(--mono); font-size: 12px; margin: 0 0 var(--space-3); }
.receipt .loss { margin: 0; color: var(--muted); max-width: 78ch; }
@keyframes verge-pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.35; } }
.dot { display: inline-block; width: 8px; height: 8px; border-radius: 50%;
  background: var(--muted); flex: none; }
.dot.live { background: var(--accent); animation: verge-pulse 1.6s infinite; }
.dot.done { background: var(--ink); }
.dot.dead { background: var(--danger); }
.scanhead { display: flex; align-items: baseline; gap: var(--space-3); flex-wrap: wrap;
  margin-bottom: var(--space-3); }
.scanhead .kind { font-family: var(--mono); font-size: 14px; font-weight: 600; }
.scanhead .tick { color: var(--muted); font-family: var(--mono); font-size: 11px; }
.scanhead .prog { margin-left: auto; font-family: var(--mono); font-size: 12px; }
.meter { height: 6px; background: var(--paper); border: 1px solid var(--hairline);
  margin: 0 0 var(--space-4); }
.meter .fill { height: 100%; background: var(--accent); }
.meter .fill.complete { background: var(--ink); }
table.jobs td .dot { margin-right: 6px; vertical-align: middle; }
table.jobs td.super { color: var(--muted); }
`

// wordmark is the typed Verge ASM mark: sans "Verge" plus a mono "ASM" chip.
// There is no drawn logo (design system).
const wordmark = `<span class="wordmark">Verge<span class="chip">ASM</span></span>`

var tmpl = template.Must(template.New("").Parse(`
{{define "head"}}<!doctype html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}} · Verge ASM</title>{{if .Refresh}}<meta http-equiv="refresh" content="6">{{end}}<style>` + pageCSS + `</style></head><body>{{end}}
{{define "foot"}}<script>
/* Preserve scroll position across the POST-redirect-GET every form action runs.
   Each mutating handler answers 303 to the same page, so a naive reload lands at
   the top — jarring on a long screen. On submit we stash scrollY keyed by the
   current path; on the next load of that same path we restore it, but only within
   a few seconds so a later fresh visit is unaffected. Native back/forward scroll
   restoration is left alone. */
(function () {
  var FRESH_MS = 5000;                              // a redirect round-trip; older stashes are a stale later visit
  var K = "verge:scroll:" + location.pathname;      // this app only full-page navigates, so the path is stable here
  try {
    var raw = sessionStorage.getItem(K);
    if (raw) {
      sessionStorage.removeItem(K);
      var s = JSON.parse(raw);
      if (s && typeof s.y === "number" && Date.now() - s.t < FRESH_MS) window.scrollTo(0, s.y);
    }
  } catch (e) {}
  document.addEventListener("submit", function (ev) {
    var f = ev.target;
    if (f && (f.method || "").toLowerCase() === "post") {
      try {
        sessionStorage.setItem(K, JSON.stringify({ y: window.scrollY, t: Date.now() }));
      } catch (e) {}
    }
  }, true);
})();
</script></body></html>{{end}}

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
<nav class="nav"><a href="/">Home</a><a href="/exposure">Exposure</a><a href="/subjects">Subjects</a><a href="/inventory">Inventory</a><a href="/signals">Signals</a><a href="/seeds">Seeds</a><a href="/coverage">Coverage</a><a href="/scans">Scans</a><a href="/verge-core">verge-core</a>{{if .IsAdmin}}<a href="/settings">Settings</a>{{end}}</nav>
<a class="msgnav" href="/messages">Messages{{if .Unread}}<span class="count">{{.Unread}}</span>{{end}}</a>
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
{{range .Seeds}}<tr id="seed-{{.Anchor}}">
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

{{template "proposals" .}}

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

<div class="microlabel">Configured · full-range scan (cold tier)</div>
<p>The cold scan connects to every TCP port, 1–65535, monthly. It ships <strong>disabled</strong>
with no scopes: a full-range sweep runs only where you ask for it, per scope — never at onboarding,
never on save. Opting a scope in enables the tier for that scope and it begins on its own monthly
cadence; opting the last scope out returns it to off. Only Custody-admitted addresses are ever
probed, so opting in widens what is measured, never who.</p>

{{if .ColdEnabled}}<span class="badge">tier on</span>{{else}}<span class="badge off">tier off — no scope opted in</span>{{end}}

{{if .ColdError}}<div class="error">{{.ColdError}}</div>{{end}}
{{if .ColdScopes}}
<div class="section">
<h2>Full-range opt-in</h2>
<table>
<thead><tr><th>Scope</th><th>Kind</th><th>Full range</th>{{if .IsAdmin}}<th></th>{{end}}</tr></thead>
<tbody>
{{range .ColdScopes}}<tr>
<td class="mono">{{.Scope}}</td>
<td>{{if .IsAddress}}address{{else}}name{{end}}</td>
<td>{{if .OptedIn}}<span class="badge">opted in</span>{{else}}<span class="badge off">off</span>{{end}}</td>
{{if $.IsAdmin}}<td>
<form method="post" action="/seeds/cold">
<input type="hidden" name="id" value="{{.ID}}">
<input type="hidden" name="opt_in" value="{{if .OptedIn}}false{{else}}true{{end}}">
<button class="secondary" type="submit">{{if .OptedIn}}Opt out{{else}}Opt in{{end}}</button>
</form>
</td>{{end}}
</tr>{{end}}
</tbody>
</table>
</div>
{{else}}
<div class="section">
<div class="microlabel">No scopes</div>
<p>The full-range tier opts in a declared scope. Declare a name or address scope above, then opt it
into the cold scan here.</p>
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
<button type="submit" formaction="/exclusions/preview" class="secondary">Preview</button>
<button type="submit">Exclude</button>
</form>
{{with .ExclPreview}}
{{if .Fires}}
<div class="receipt">
<div class="microlabel">What this exclusion would withdraw</div>
<p class="headline">{{.Headline}}</p>
<p class="loss">{{.Loss}}</p>
</div>
{{else}}
<div class="receipt">
<div class="microlabel">What this exclusion would withdraw</div>
<p class="loss">Nothing is withdrawn. No subject leaves the estate, so no message fires — an excluded name that still resolves survives, and its Gap carries it.</p>
</div>
{{end}}
{{end}}
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
<p>Retention is what may still be read, never age. Dispatch retention is a multiple of the
slowest enabled scan's cadence, floored at 2 cadences; 0 leaves it unbounded, the default. The
observation-currency floor lands with later work; for now zero means no operator floor.</p>
{{if .RetError}}<div class="error">{{.RetError}}</div>{{end}}
<form method="post" action="/settings/retention">
<div class="dial">
<label><span>Observation-currency floor</span><input name="observation_currency_days" inputmode="numeric" value="{{if .RetError}}{{.RetObs}}{{else}}{{.Retention.ObservationCurrencyDays}}{{end}}" required><span class="unit">days</span></label>
<label><span>Dispatch retention</span><input name="dispatch_cadence_multiple" inputmode="numeric" value="{{if .RetError}}{{.RetDispatch}}{{else}}{{.Retention.DispatchCadenceMultiple}}{{end}}" required><span class="unit">× slowest scan cadence</span></label>
<button type="submit">Save dials</button>
</div>
</form>
{{if .Retention.UpdatedAt}}<p class="muted" style="margin-top:12px">Last changed {{.Retention.UpdatedAt}}{{if .Retention.UpdatedBy}} by <span class="mono">{{.Retention.UpdatedBy}}</span>{{end}}.</p>{{end}}
</div>
</main>
{{template "foot" .}}{{end}}

{{define "vergecore"}}{{template "head" .}}
{{template "chrome" .}}
<main>
<div class="microlabel">Declared · verge-core</div>
<h1>verge-core</h1>
<p>The daily hot-tier port set: the union of a <em>frequency</em> half — the project's own
open-frequency selection — and a <em>sensitive</em> half, the curated list of ports never
legitimately internet-facing. Only the TCP pairs are probed; the {{.UDPCount}} UDP pairs are recorded
in scope and never probed. You may edit the frequency half. The sensitive half is authored by the
release — moving one of its pairs would move a version and break every timeline it touches — so it is
read-only here.</p>

<div class="section">
<div class="microlabel">Composition</div>
<div class="kv"><div class="k">Union</div><div class="mono">{{.Counts.Union}} pairs ({{.Counts.TCP}} TCP, {{.Counts.UDP}} UDP)</div></div>
<div class="kv"><div class="k">Frequency</div><div class="mono">{{.Counts.Frequency}} pairs (TCP, editable)</div></div>
<div class="kv"><div class="k">Sensitive</div><div class="mono">{{.Counts.Sensitive}} pairs (read-only)</div></div>
</div>

{{if .IsAdmin}}
<div class="section">
<h2>Add a frequency port</h2>
<p>Add a TCP port to the frequency half. It joins the daily probe set from the next hot scan.</p>
{{if .Error}}<div class="error">{{.Error}}</div>{{end}}
{{if .Notice}}<div class="notice">{{.Notice}}</div>{{end}}
<form method="post" action="/verge-core/frequency" class="inlineform">
<input type="hidden" name="action" value="add">
<label><span>Port</span><input name="port" inputmode="numeric" value="{{.FormPort}}" placeholder="8443" autocomplete="off" required></label>
<button type="submit">Add to frequency</button>
</form>
</div>
{{else}}
{{if .Error}}<div class="error">{{.Error}}</div>{{end}}
{{if .Notice}}<div class="notice">{{.Notice}}</div>{{end}}
{{end}}

<div class="section">
<h2>Frequency half</h2>
<p class="muted">{{.Counts.Frequency}} TCP ports. A port also on the sensitive half stays probed even if you
remove it here — the union keeps it — which is exactly why removing it cannot move the sensitive half.</p>
<table>
<thead><tr><th>Port</th><th>Also sensitive</th><th>Edit</th>{{if .IsAdmin}}<th></th>{{end}}</tr></thead>
<tbody>
{{range .Frequency}}<tr>
<td class="mono">{{.Port}}/tcp</td>
<td>{{if .AlsoSensitive}}<span class="badge">sensitive</span>{{else}}<span class="muted">—</span>{{end}}</td>
<td>{{if .Edited}}<span class="badge">{{.EditAction}}</span>{{else}}<span class="muted">shipped</span>{{end}}</td>
{{if $.IsAdmin}}<td class="row">
<form method="post" action="/verge-core/frequency"><input type="hidden" name="action" value="remove"><input type="hidden" name="port" value="{{.Port}}"><button class="secondary" type="submit">Remove</button></form>
{{if .Edited}}<form method="post" action="/verge-core/frequency"><input type="hidden" name="action" value="reset"><input type="hidden" name="port" value="{{.Port}}"><button class="secondary" type="submit">Reset</button></form>{{end}}
</td>{{end}}
</tr>{{end}}
</tbody>
</table>
</div>

<div class="section">
<h2>Sensitive half · read-only</h2>
<p class="muted">{{.Counts.Sensitive}} pairs, authored by the release. There is no control to move one — it
would move a version without a golden-corpus row moving.</p>
<table>
<thead><tr><th>Port</th><th>Transport</th></tr></thead>
<tbody>
{{range .Sensitive}}<tr><td class="mono">{{.Port}}</td><td class="mono">{{.Transport}}</td></tr>{{end}}
</tbody>
</table>
</div>
</main>
{{template "foot" .}}{{end}}

{{define "subjects"}}{{template "head" .}}
{{template "chrome" .}}
<main>
<div class="microlabel">Observed · subjects</div>
<h1>Subjects</h1>
<p>Every Name, Service, and Endpoint currently in your estate. A Service is a port
on an address the hot Scan reached for; an Endpoint is a (name, service) pair the
http-exchange leaf completed a GET / against — the key under which HTTP identity
is single-valued. Each is in the estate exactly while its address is, which holds
while a current resolution cites the address or a Seed covers it.
There is no total: how many subjects your estate ought to hold is its completeness,
which only you know, so this screen states none.</p>

<form method="get" action="/subjects" class="searchbar">
<label class="grow"><span>Search subjects</span><input name="q" value="{{.Search}}" placeholder="example.com or 198.51.100.1:443" autocomplete="off"></label>
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

<div class="section">
<div class="microlabel">Service subjects</div>
{{if .Services}}
<table>
<thead><tr><th>Service</th><th>Reachability</th></tr></thead>
<tbody>
{{range .Services}}<tr>
<td><a class="mono" href="/subjects/service?key={{.Key}}">{{.Key}}</a></td>
<td>{{if .Reach}}<span class="badge">{{.Reach}}</span>{{else}}<span class="muted">—</span>{{end}}</td>
</tr>{{end}}
</tbody>
</table>
{{else}}
<div class="microlabel">{{if .Search}}No service matches{{else}}No services yet{{end}}</div>
<p>{{if .Search}}No current service matches that search. A service whose address left the estate is reached by its exact key, never by browsing.{{else}}No service has been measured yet. The hot Scan reaches for the verge-core ports on every address your names resolve to; run it once a resolution has cited an address.{{end}}</p>
{{end}}
</div>

<div class="section">
<div class="microlabel">Endpoint subjects</div>
{{if .Endpoints}}
<table>
<thead><tr><th>Endpoint</th><th>Service</th><th>HTTP identity</th></tr></thead>
<tbody>
{{range .Endpoints}}<tr>
<td><a class="mono" href="/subjects/endpoint?key={{.Key}}">{{if .Nameless}}<span class="muted">(nameless)</span>{{else}}{{.Name}}{{end}}</a></td>
<td class="mono">{{.Service}}</td>
<td>{{if .Identity}}<span class="badge">{{.Identity}}</span>{{else}}<span class="muted">—</span>{{end}}</td>
</tr>{{end}}
</tbody>
</table>
{{else}}
<div class="microlabel">{{if .Search}}No endpoint matches{{else}}No endpoints yet{{end}}</div>
<p>{{if .Search}}No current endpoint matches that search. An endpoint whose service left the estate is reached by its exact key, never by browsing.{{else}}No endpoint has been measured yet. An endpoint is a (name, service) pair the http-exchange leaf completed a GET / against; it appears once the hot Scan has reached a web service and exchanged with it.{{end}}</p>
{{end}}
</div>
</main>
{{template "foot" .}}{{end}}

{{define "inventory"}}{{template "head" .}}
{{template "chrome" .}}
<main>
<div class="microlabel">Observed · inventory</div>
<h1>Inventory</h1>
<p>What your estate holds right now — the actual values behind the verdicts. Where the
Subjects views answer <em>what changed</em>, this answers <em>what do I have</em>: the
addresses a name resolves to, the records it carries, the certificate a service presents,
the identity an endpoint returns. Each row is the value a facet's current span holds — click
a value to expand it to its individual records. A withdrawn subject holds no current span and
so is not here. As on Subjects there is no total: your estate's completeness is yours alone to
state.</p>

{{if .Groups}}
{{range .Groups}}
<div class="section">
<div class="microlabel">{{.Label}}</div>
{{range .Subjects}}
<div class="invsubject">
{{if .Link}}<a class="mono invkey" href="{{.Link}}">{{.Key}}</a>{{else}}<span class="mono invkey">{{.Key}}</span>{{end}}
<table class="invfacets"><tbody>
{{range .Facets}}<tr>
<td class="invfacet"><span class="microlabel">{{.Label}}</span></td>
<td>{{if .IsGap}}<span class="badge">Gap</span>{{else if .Details}}<details class="spanrecords"><summary><span class="badge">{{.Summary}}</span></summary>{{template "recordrows" .Details}}</details>{{else}}<span class="badge">{{.Summary}}</span>{{end}}</td>
<td class="mono muted invsince">since {{.Since}}</td>
</tr>{{end}}
</tbody></table>
</div>
{{end}}
</div>
{{end}}
{{else}}
<div class="section">
<div class="microlabel">Nothing measured yet</div>
<p>No subject holds an open span yet. Declare a scope on Seeds and let a Scan measure a value;
the inventory fills as facets are folded.</p>
</div>
{{end}}
</main>
{{template "foot" .}}{{end}}

{{define "recordrows"}}<table class="records"><tbody>
{{range .}}<tr>{{if .Type}}<td class="rrtype"><span class="badge">{{.Type}}</span></td>{{else}}<td class="rrtype"></td>{{end}}<td class="mono">{{.Data}}</td></tr>{{end}}
</tbody></table>{{end}}

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
{{if .Timelines}}
<p class="muted">Each timeline is one period a value was held. A Break marks two spans the drift engine may not compare, naming the leaf that moved; it is derived on read and never stored.</p>
{{range .Timelines}}
<div class="timeline">
<div class="microlabel">{{.Label}}</div>
{{if .Current}}
<div class="kv"><div class="k">Current</div><div>{{if .Current.IsGap}}<span class="badge">Gap</span>{{else if .Current.Details}}<details class="spanrecords"><summary><span class="badge">{{.Current.Value}}</span></summary>{{template "recordrows" .Current.Details}}</details>{{else}}<span class="badge">{{.Current.Value}}</span>{{end}} <span class="muted mono">since {{.Current.OpenedAt}}</span></div></div>
{{else}}
<div class="kv"><div class="k">Current</div><div class="muted">Closed — this timeline holds no current value.</div></div>
{{end}}
{{if .Breaks}}{{range .Breaks}}
<div class="notice">Break at {{.At}} — not comparable across it. Leaf that moved: <span class="mono">{{.MovedLeaves}}</span></div>
{{end}}{{end}}
{{if .Closed}}
<table class="closedspans">
<thead><tr><th>Value</th><th>Opened</th><th>Closed</th><th>Ground</th></tr></thead>
<tbody>
{{range .Closed}}<tr>
<td>{{if .IsGap}}<span class="muted">Gap</span>{{else if .Details}}<details class="spanrecords"><summary><span class="mono">{{.Value}}</span></summary>{{template "recordrows" .Details}}</details>{{else}}<span class="mono">{{.Value}}</span>{{end}}</td>
<td class="mono">{{.OpenedAt}}</td>
<td class="mono">{{.ClosedAt}}</td>
<td>{{if .Reason}}<span class="badge">{{.Reason}}</span>{{else}}<span class="muted">—</span>{{end}}</td>
</tr>{{end}}
</tbody>
</table>
{{end}}
</div>
{{end}}
{{else}}
<p class="muted">No timeline has been folded yet. A Span opens when the dns Scan first measures a value for this name; re-running it with a changed answer closes the open span and opens the next.</p>
{{end}}
</div>

<div class="section">
<div class="microlabel">Rules</div>
<h2>Rules over this subject</h2>
<p class="muted">Every rule whose predicate domain includes this subject renders here, each carrying its own versioned verdict. Wired up by ticket 22.</p>
</div>
{{end}}
</main>
{{template "foot" .}}{{end}}

{{define "service"}}{{template "head" .}}
{{template "chrome" .}}
<main>
{{with .Service}}
<div class="microlabel">Observed · Service</div>
<h1 class="mono">{{.Key}}</h1>
{{if .Withdrawn}}
<div class="notice">This service's address has left the estate — no current resolution cites it and no Seed covers it. It names a population of no current member; its timelines are closed and it is reached by its own key.</div>
{{end}}

<div class="section">
<div class="microlabel">Why is this here</div>
<h2>Citation chain</h2>
<p>A Service is an (address, port, transport) triple. Its membership is its address's membership restated — an address is in the estate exactly while a current resolution cites it or a Seed covers it — so the chain runs from the Service down through its address to the Seed you declared.</p>
<ol class="chain">
{{range .Citation}}<li>
<div class="microlabel">{{.Label}}</div>
<div class="mono chainval">{{.Value}}</div>
{{if .Detail}}<div class="muted">{{.Detail}}</div>{{end}}
</li>{{end}}
</ol>
{{if not .CitationTerminated}}<p class="muted">The chain does not reach a declared Seed. For a service whose address a resolution cites, that is the address's name-scope Seed, one hop past the citing name; for one only a Seed covers, it is the address scope directly.</p>{{end}}
</div>

<div class="section">
<div class="microlabel">Current · reachability</div>
<h2>Reachability</h2>
<div class="kv"><div class="k">Address</div><div class="mono">{{.Address}}</div></div>
<div class="kv"><div class="k">Port</div><div class="mono">{{.Port}}/{{.Transport}}</div></div>
{{if .ReachGap}}
<div class="kv"><div class="k">Verdict</div><div><span class="badge">Gap</span></div></div>
<div class="notice">{{.ReachGapReason}}. From this vantage we cannot tell a real origin service behind the edge from the edge answering for it, so the reach is undiscriminated — a Gap, not <span class="mono">reached</span>. Declare your origin IPs as an address scope to measure the real surface.</div>
{{else if .Reach}}
<div class="kv"><div class="k">Verdict</div><div><span class="badge">{{.Reach}}</span></div></div>
{{else}}<p class="muted">No reachability value recorded.</p>{{end}}
</div>

<div class="section">
<div class="microlabel">Timelines</div>
<h2>Current and closed timelines</h2>
{{if .Timelines}}
<p class="muted">Each timeline is one period a value was held. A Break marks two spans the drift engine may not compare, naming the leaf that moved; it is derived on read and never stored.</p>
{{range .Timelines}}
<div class="timeline">
<div class="microlabel">{{.Label}}</div>
{{if .Current}}
<div class="kv"><div class="k">Current</div><div>{{if .Current.IsGap}}<span class="badge">Gap</span>{{else if .Current.Details}}<details class="spanrecords"><summary><span class="badge">{{.Current.Value}}</span></summary>{{template "recordrows" .Current.Details}}</details>{{else}}<span class="badge">{{.Current.Value}}</span>{{end}} <span class="muted mono">since {{.Current.OpenedAt}}</span></div></div>
{{else}}
<div class="kv"><div class="k">Current</div><div class="muted">Closed — this timeline holds no current value.</div></div>
{{end}}
{{if .Breaks}}{{range .Breaks}}
<div class="notice">Break at {{.At}} — not comparable across it. Leaf that moved: <span class="mono">{{.MovedLeaves}}</span></div>
{{end}}{{end}}
{{if .Closed}}
<table class="closedspans">
<thead><tr><th>Value</th><th>Opened</th><th>Closed</th><th>Ground</th></tr></thead>
<tbody>
{{range .Closed}}<tr>
<td>{{if .IsGap}}<span class="muted">Gap</span>{{else if .Details}}<details class="spanrecords"><summary><span class="mono">{{.Value}}</span></summary>{{template "recordrows" .Details}}</details>{{else}}<span class="mono">{{.Value}}</span>{{end}}</td>
<td class="mono">{{.OpenedAt}}</td>
<td class="mono">{{.ClosedAt}}</td>
<td>{{if .Reason}}<span class="badge">{{.Reason}}</span>{{else}}<span class="muted">—</span>{{end}}</td>
</tr>{{end}}
</tbody>
</table>
{{end}}
</div>
{{end}}
{{else}}
<p class="muted">No timeline has been folded yet. A Span opens when the hot Scan first reaches for this port; re-running it with the port opening or closing closes the open span and opens the next.</p>
{{end}}
</div>

<div class="section">
<div class="microlabel">Rules</div>
<h2>Rules over this subject</h2>
<p class="muted">Every rule whose predicate domain includes this subject renders here, each carrying its own versioned verdict. Wired up by ticket 22.</p>
</div>
{{end}}
</main>
{{template "foot" .}}{{end}}

{{define "endpoint"}}{{template "head" .}}
{{template "chrome" .}}
<main>
{{with .Endpoint}}
<div class="microlabel">Observed · Endpoint</div>
<h1 class="mono">{{if .Nameless}}<span class="muted">(nameless)</span> {{end}}{{.Key}}</h1>
{{if .Withdrawn}}
<div class="notice">This endpoint's service has left the estate — no current resolution cites its address and no Seed covers it. It names a population of no current member; its timelines are closed and it is reached by its own key. An endpoint closes when either leg — its Name or its Service — withdraws.</div>
{{end}}

<div class="section">
<div class="microlabel">Why is this here</div>
<h2>Citation chain</h2>
<p>An Endpoint is a (Name, Service) pair — the only key under which HTTP identity is single-valued. Its membership is its Service's, restated: a Service is in the estate exactly while a current resolution cites its address or a Seed covers it, so the chain runs from the Endpoint through its Name and Service legs down to the Seed you declared.</p>
<ol class="chain">
{{range .Citation}}<li>
<div class="microlabel">{{.Label}}</div>
<div class="mono chainval">{{.Value}}</div>
{{if .Detail}}<div class="muted">{{.Detail}}</div>{{end}}
</li>{{end}}
</ol>
{{if not .CitationTerminated}}<p class="muted">The chain does not reach a declared Seed. For an endpoint whose service address a resolution cites, that is the address's name-scope Seed, one hop past the citing name; for one only a Seed covers, it is the address scope directly.</p>{{end}}
</div>

<div class="section">
<div class="microlabel">Current · http-identity</div>
<h2>HTTP identity</h2>
<div class="kv"><div class="k">Name</div><div class="mono">{{if .Nameless}}<span class="muted">nameless endpoint</span>{{else}}{{.Name}}{{end}}</div></div>
<div class="kv"><div class="k">Service</div><div class="mono">{{.Service}}</div></div>
{{if .HasIdentity}}
<div class="kv"><div class="k">Status</div><div><span class="badge">{{.Status}}</span></div></div>
{{if .Server}}<div class="kv"><div class="k">Server</div><div class="mono">{{.Server}}</div></div>{{end}}
{{if .Title}}<div class="kv"><div class="k">Title</div><div class="mono">{{.Title}}</div></div>{{end}}
{{if .WWWAuthenticate}}<div class="kv"><div class="k">WWW-Authenticate</div><div class="mono">{{.WWWAuthenticate}}</div></div>{{end}}
{{if .RedirectLocation}}<div class="kv"><div class="k">Redirect</div><div class="mono">{{.RedirectLocation}} <span class="muted">(recorded, not followed)</span></div></div>{{end}}
{{else}}<p class="muted">No HTTP identity value recorded.</p>{{end}}
</div>

<div class="section">
<div class="microlabel">Timelines</div>
<h2>Current and closed timelines</h2>
{{if .Timelines}}
<p class="muted">Each timeline is one period a value was held. A Break marks two spans the drift engine may not compare, naming the leaf that moved; it is derived on read and never stored.</p>
{{range .Timelines}}
<div class="timeline">
<div class="microlabel">{{.Label}}</div>
{{if .Current}}
<div class="kv"><div class="k">Current</div><div>{{if .Current.IsGap}}<span class="badge">Gap</span>{{else if .Current.Details}}<details class="spanrecords"><summary><span class="badge">{{.Current.Value}}</span></summary>{{template "recordrows" .Current.Details}}</details>{{else}}<span class="badge">{{.Current.Value}}</span>{{end}} <span class="muted mono">since {{.Current.OpenedAt}}</span></div></div>
{{else}}
<div class="kv"><div class="k">Current</div><div class="muted">Closed — this timeline holds no current value.</div></div>
{{end}}
{{if .Breaks}}{{range .Breaks}}
<div class="notice">Break at {{.At}} — not comparable across it. Leaf that moved: <span class="mono">{{.MovedLeaves}}</span></div>
{{end}}{{end}}
{{if .Closed}}
<table class="closedspans">
<thead><tr><th>Value</th><th>Opened</th><th>Closed</th><th>Ground</th></tr></thead>
<tbody>
{{range .Closed}}<tr>
<td>{{if .IsGap}}<span class="muted">Gap</span>{{else if .Details}}<details class="spanrecords"><summary><span class="mono">{{.Value}}</span></summary>{{template "recordrows" .Details}}</details>{{else}}<span class="mono">{{.Value}}</span>{{end}}</td>
<td class="mono">{{.OpenedAt}}</td>
<td class="mono">{{.ClosedAt}}</td>
<td>{{if .Reason}}<span class="badge">{{.Reason}}</span>{{else}}<span class="muted">—</span>{{end}}</td>
</tr>{{end}}
</tbody>
</table>
{{end}}
</div>
{{end}}
{{else}}
<p class="muted">No timeline has been folded yet. A Span opens when the hot Scan first exchanges with this endpoint; re-running it with a changed identity closes the open span and opens the next.</p>
{{end}}
</div>

<div class="section">
<div class="microlabel">Rules</div>
<h2>Rules over this subject</h2>
<p class="muted">Every rule whose predicate domain includes this subject renders here, each carrying its own versioned verdict. Wired up by ticket 22.</p>
</div>
{{end}}
</main>
{{template "foot" .}}{{end}}

{{define "signals"}}{{template "head" .}}
{{template "chrome" .}}
<main>
<div class="microlabel">Derived · signals</div>
<h1>Signals</h1>
<p>Each rule is a named, versioned fact read over your estate as it stands now.
Its census is three members over one population — fired, did not fire, and
not-evaluable — current state only, never a trend and never a comparison. A
signal is a named fact, not a ranked one: urgency belongs to the transition that
surfaces it, never to the rule. All seventeen v1 rules render — Name, Service and
Endpoint — each over the population its own name could be true of.</p>

{{range .Censuses}}
<div class="section">
<div class="rulehead">
<h2 class="mono">{{.Rule}}</h2>
<span class="microlabel ver">version {{.Version}}</span>
</div>
{{if .Empty}}
<div class="census">
<div class="microlabel">No population</div>
<p>No subject in your estate is one this rule's fact could be true of, so there
is nothing to census. This is a legible state, not a clean bill of health.</p>
</div>
{{else}}
<div class="members">
{{range .Groups}}
<div class="mgroup">
{{if .Prose}}
<div class="fullmute">
<div class="microlabel">Fired · every subject accepted</div>
<p>{{.Prose}}</p>
</div>
{{else}}
<div class="mgroup-head"><span class="microlabel">{{.Label}}</span><span class="count">{{len .Members}}</span></div>
{{if .Members}}
<ul class="mgroup-list">
{{range .Members}}<li><a class="mono" href="{{.Href}}">{{.Subject}}</a></li>{{end}}
</ul>
{{else}}
<div class="mgroup-empty">None.</div>
{{end}}
{{end}}
</div>
{{end}}
</div>
{{end}}
</div>
{{end}}

<div class="section">
<div class="microlabel">Operator dial · annotations</div>
<h2>Annotations</h2>
<p>An annotation accepts one rule's firing on one subject as a known risk on a
thing you are still measuring. It moves no number — the subject is still measured,
still counted under fired — and changes only the message: an annotated pair's next
firing reaches no one. It carries your reason and the instant you declared it, and
nothing else: no status, no expiry and no author. Declaring and withdrawing are
neither of them a message.</p>
{{if .IsAdmin}}
{{if .AnnoError}}<div class="error">{{.AnnoError}}</div>{{end}}
<form method="post" action="/annotations" class="annoform">
<label class="grow"><span>Subject</span><input name="subject" value="{{.AnnoSubject}}" placeholder="host.example.com" autocomplete="off" required></label>
<label><span>Signal</span><select name="signal" required>
<option value="">Choose a signal…</option>
{{range .RuleNames}}<option value="{{.}}"{{if eq . $.AnnoSignal}} selected{{end}}>{{.}}</option>{{end}}
</select></label>
<label class="grow"><span>Reason</span><input name="reason" value="{{.AnnoReason}}" placeholder="Why this firing is accepted" autocomplete="off" required></label>
<button type="submit">Declare</button>
</form>
{{end}}
{{if .Annotations}}
<table class="annos">
<tr><th>Subject</th><th>Signal</th><th>Reason</th><th>Declared</th>{{if .IsAdmin}}<th></th>{{end}}</tr>
{{range .Annotations}}
<tr>
<td><a class="mono" href="{{.Href}}">{{.Subject}}</a>{{if .Orphan}}<span class="orphan">names no current member</span>{{end}}</td>
<td class="mono">{{.Signal}}</td>
<td>{{.Reason}}</td>
<td class="mono">{{.At}}</td>
{{if $.IsAdmin}}<td><form method="post" action="/annotations/withdraw"><input type="hidden" name="id" value="{{.ID}}"><button class="secondary" type="submit">Withdraw</button></form></td>{{end}}
</tr>
{{end}}
</table>
{{else}}
<p class="muted">No annotation is declared. A fired signal you accept as a known
risk is declared here, keyed on its subject.</p>
{{end}}
</div>
</main>
{{template "foot" .}}{{end}}

{{define "exposure"}}{{template "head" .}}
{{template "chrome" .}}
<main>
<div class="microlabel">Derived · exposure</div>
<h1>Exposure</h1>
<p>The exposure board — the reachability of every service seen from both sides of your
boundary at once, never the raw inventory. Each service is placed by two measurements:
whether an internet vantage reached it, and whether an internal one did. The board is a
census, not an alert — the one move worth waking for, a service becoming reachable from
the internet, is called out under "what moved".</p>

<div class="avail">
<div><span class="k">Internet vantage</span>{{if .InternetPresent}}<span class="badge">present</span>{{else}}<span class="badge off">none</span>{{end}}</div>
<div><span class="k">Internal vantage</span>{{if .InternalPresent}}<span class="badge">present</span>{{else}}<span class="badge off">none</span>{{end}}</div>
</div>

{{if .NoServices}}
<div class="precond">
<div class="microlabel">Precondition · nothing to place</div>
<h2>No service in your estate yet</h2>
<p>A service is a port on an address your estate reaches for. None has been measured yet,
so there is nothing to place on the board. Declare a scope on Seeds and run the hot scan
once a resolution has cited an address.</p>
</div>
{{else}}

{{if not .Constructible}}
<div class="precond">
<div class="microlabel">Precondition · no exposure constructible</div>
<h2>Exposure needs both sides, and only one is looking</h2>
<p>Exposure is composed only from services measured by at least two vantage classes. Fewer
than two hold a current value here, so no exposure verdict is constructed — you see each
service's raw reach on the one side that looked, below, never a stand-in reading of the
side that did not. {{if not .InternetPresent}}There is no internet vantage: provision a
prober on Seeds to measure the side that matters most.{{else}}There is no internal vantage:
the internet reach renders on its own until one is configured.{{end}}</p>
</div>
{{end}}

{{if .WhatMoved}}
<div class="moved">
<div class="microlabel">What moved · flagship</div>
<h2 style="margin:4px 0 8px;font-size:14px">A service became reachable from the internet</h2>
<p class="muted" style="margin-bottom:8px">The internet reach of these services crossed not-reached to reached — the move the product
exists to catch. It fires on this leg alone, whether or not the internal side exists.</p>
<ul class="movedlist">
{{range .WhatMoved}}<li><a href="/subjects/service?key={{.}}">{{.}}</a></li>{{end}}
</ul>
</div>
{{end}}

{{if .HasBoard}}
<div class="microlabel">Populated board · {{.BoardTotal}} services measured from both sides</div>
<div class="board">
<div class="corner"><span class="microlabel">internet ↓</span><span class="microlabel">internal →</span></div>
<div class="colhead"><span class="microlabel">internal reached</span></div>
<div class="colhead"><span class="microlabel">internal not-reached</span></div>

<div class="rowhead"><span class="microlabel">internet reached</span></div>
<div class="cell hot">
<div class="microlabel">exposed</div>
<div class="count">{{len .Exposed}}</div>
{{if .Exposed}}<ul>{{range .Exposed}}<li><a href="/subjects/service?key={{.}}">{{.}}</a></li>{{end}}</ul>{{else}}<div class="none">—</div>{{end}}
</div>
<div class="cell">
<div class="microlabel">edge-only</div>
<div class="count">{{len .EdgeOnly}}</div>
{{if .EdgeOnly}}<ul>{{range .EdgeOnly}}<li><a href="/subjects/service?key={{.}}">{{.}}</a></li>{{end}}</ul>{{else}}<div class="none">—</div>{{end}}
</div>

<div class="rowhead"><span class="microlabel">internet not-reached</span></div>
<div class="cell">
<div class="microlabel">firewalled</div>
<div class="count">{{len .Firewalled}}</div>
{{if .Firewalled}}<ul>{{range .Firewalled}}<li><a href="/subjects/service?key={{.}}">{{.}}</a></li>{{end}}</ul>{{else}}<div class="none">—</div>{{end}}
</div>
<div class="cell">
<div class="microlabel">unreachable</div>
<div class="count">{{len .Unreachable}}</div>
{{if .Unreachable}}<ul>{{range .Unreachable}}<li><a href="/subjects/service?key={{.}}">{{.}}</a></li>{{end}}</ul>{{else}}<div class="none">—</div>{{end}}
</div>
</div>
{{end}}

{{if .OneLegged}}
<div class="section">
<div class="microlabel">One-legged · the surviving side's raw reach</div>
<h2>We only looked from one side</h2>
<p>These services were measured from a single vantage class, so no exposure verdict exists
for them — only the raw reach of the side that looked. This is never a fifth exposure value;
it is one measurement, honestly labelled with the side we did not see.</p>
<table>
<thead><tr><th>Service</th><th>Side looked</th><th>Reach</th><th>The other side</th></tr></thead>
<tbody>
{{range .OneLegged}}<tr>
<td><a class="mono" href="/subjects/service?key={{.Service}}">{{.Service}}</a></td>
<td><span class="badge">{{.Class}}</span></td>
<td class="mono">{{.Value}}</td>
<td class="muted">{{.Statement}}</td>
</tr>{{end}}
</tbody>
</table>
</div>
{{end}}

{{if .Broken}}
<div class="precond">
<div class="microlabel">Precondition · your rules changed</div>
<h2>Nothing to compare yet for these services</h2>
<p>The derivation that composes exposure moved for these services, so their two spans are
not comparable and no verdict is drawn across the break. This is your rules changing, not
your exposure — a new value ships as a break, never as rewritten history. The rest of the
board is unaffected.</p>
<ul class="movedlist">
{{range .Broken}}<li><a href="/subjects/service?key={{.}}">{{.}}</a></li>{{end}}
</ul>
</div>
{{end}}
{{end}}
</main>
{{template "foot" .}}{{end}}

{{define "messages"}}{{template "head" .}}
{{template "chrome" .}}
<main>
<div class="rulehead">
<div><div class="microlabel">Operational · messages</div>
<h1>Messages</h1></div>
{{if .Messages}}<form method="post" action="/messages/read-all"><button class="secondary" type="submit">Mark all read</button></form>{{end}}
</div>
<p>Every message that has fired, newest first. Each names what moved — the estate's
own object, our own aperture, your declared input, or a threshold crossed — and
links to what it fired at. The store carries the message and never the estate: the
fact itself is in the timelines.</p>
{{if not .Messages}}
<div class="section">
<p>No message has fired. Messages appear here as the estate moves — declare a seed
and run the first batch to begin measuring.</p>
</div>
{{else}}
<ul class="msglist">
{{range .Messages}}
<li class="msgitem{{if not .Read}} unread{{end}}">
<div class="msgitem-head">
<span class="cause">{{.Cause}}</span>
<span class="microlabel">{{.Class}}</span>
{{if .Instant}}<span class="when">{{.Instant}}</span>{{end}}
</div>
<p class="headline">{{.Headline}}</p>
<div class="rowlink"><a href="{{.Href}}">{{.LinkText}}</a></div>
{{if .Census}}
<ul class="msgcensus">
{{range .Census}}<li><span class="k">{{.Kind}}</span>{{if .Href}}<a href="{{.Href}}">{{.Key}}</a>{{else}}{{.Key}}{{end}}</li>{{end}}
</ul>
{{end}}
{{if not .Read}}
<div class="actions"><form method="post" action="/messages/read"><input type="hidden" name="id" value="{{.ID}}"><button class="secondary" type="submit">Mark read</button></form></div>
{{end}}
</li>
{{end}}
</ul>
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

{{define "scans"}}{{template "head" .}}
{{template "chrome" .}}
<main>
<div class="microlabel">Operational · scans</div>
<h1>Scans</h1>
<p>What the queue is doing right now. A scan runs as a fan-out of jobs — one per
vantage, or per supplied zone file — and each job commits its own batch of
observations. This is a read of the queue alone: it records what the system did,
never what is true of your estate, so nothing here feeds a change report. Scans run
on their own cadence; there is no button to press.</p>

<div class="microlabel">In flight</div>
{{if .Active}}
{{range .Active}}
<div class="section">
<div class="scanhead">
<span class="dot live"></span>
<span class="kind">{{.ScanKind}}</span>
<span class="tick">dispatched {{.DispatchedAt}}</span>
<span class="prog">{{.Completed}} / {{.Live}} jobs · {{.Percent}}%</span>
</div>
<div class="meter"><div class="fill" style="width:{{.Percent}}%"></div></div>
{{if .Jobs}}
<table class="jobs">
<thead><tr><th>Job</th><th>Kind</th><th>Vantage</th><th>State</th><th>Attempt</th><th>Outcome</th></tr></thead>
<tbody>
{{range .Jobs}}<tr>
<td class="mono{{if .Superseded}} super{{end}}">#{{.ID}}</td>
<td class="mono{{if .Superseded}} super{{end}}">{{.Kind}}</td>
<td class="mono{{if .Superseded}} super{{end}}">{{if .Vantage}}{{.Vantage}}{{else}}<span class="muted">—</span>{{end}}</td>
<td>{{if eq .State "running"}}<span class="dot live"></span><span class="badge">running</span>{{else if eq .State "ready"}}<span class="dot"></span><span class="badge">{{if .Retrying}}retrying{{else}}ready{{end}}</span>{{else if eq .State "done"}}<span class="dot done"></span><span class="badge">done</span>{{else if eq .State "dead"}}<span class="dot dead"></span><span class="badge">dead</span>{{else}}<span class="dot"></span><span class="muted">superseded</span>{{end}}</td>
<td class="mono">{{.Attempt}}/{{.MaxAttempts}}</td>
<td>{{if .Batch}}<span class="badge">{{.Batch}}</span>{{else}}<span class="muted">—</span>{{end}}</td>
</tr>{{end}}
</tbody>
</table>
{{end}}
</div>
{{end}}
{{else}}
<div class="section">
<div class="microlabel">No scan running</div>
<p>Nothing is dispatched right now. When a scan's cadence comes due the worker fans
it out, and it appears here with its jobs while it runs. This view refreshes on its
own while a scan is in flight.</p>
</div>
{{end}}

<div class="microlabel">Recent dispatches</div>
{{if .History}}
<div class="section">
<table>
<thead><tr><th></th><th>Scan</th><th>Dispatched</th><th>Jobs</th><th>Completed</th><th>Dead</th></tr></thead>
<tbody>
{{range .History}}<tr>
<td><span class="dot{{if gt .Dead 0}} dead{{else}} done{{end}}"></span></td>
<td class="mono">{{.ScanKind}}</td>
<td class="mono">{{.DispatchedAt}}</td>
<td class="mono">{{.Live}}</td>
<td class="mono">{{.Completed}}</td>
<td class="mono">{{if gt .Dead 0}}{{.Dead}}{{else}}<span class="muted">0</span>{{end}}</td>
</tr>{{end}}
</tbody>
</table>
</div>
{{else}}
<div class="section">
<div class="microlabel">No dispatches yet</div>
<p>No scan has been dispatched. Once a scan runs — on its cadence, or on the first
batch after onboarding — its completed dispatches are listed here.</p>
</div>
{{end}}
</main>
{{template "foot" .}}{{end}}
`))
