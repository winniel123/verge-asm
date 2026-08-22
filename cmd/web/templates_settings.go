package main

import "html/template"

// Settings screen — canonical `/settings` (account menu, admin). Folds today's
// settings (accounts, channels, retention), verge-core, messages, and the scans
// monitor, plus the shared `forbidden` page. The screen ticket (T-Settings)
// rewrites the body against examples/console/Settings.jsx (tabbed: scans ·
// vantages · channels · messages · delivery · access/SSO · integrations). Ported
// verbatim for T0.
var _ = template.Must(tmpl.Parse(settingsTemplates))

const settingsTemplates = `
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
{{if .Deliveries}}
<ul class="msgdelivery">
{{range .Deliveries}}<li class="{{if .Failed}}delivery-failed{{end}}">
{{if .Failed}}<span class="badge off">undelivered</span> to <span class="mono">{{.ChannelHost}}</span> — this message could not be delivered, not that nothing fired{{if .LastError}} <span class="muted why" title="{{.LastError}}">(reason)</span>{{end}}
{{else}}<span class="muted">{{.State}} to <span class="mono">{{.ChannelHost}}</span></span>{{end}}
</li>{{end}}
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

{{define "scans"}}{{template "head" .}}
{{template "chrome" .}}
<main>
<div class="microlabel">Operational · scans</div>
<h1>Scans</h1>
<p>What the queue is doing right now. A scan runs as a fan-out of jobs — one per
vantage, or per supplied zone file — and each job commits its own batch of
observations. This is a read of the queue alone: it records what the system did,
never what is true of your estate, so nothing here feeds a change report. Scans run
on their own cadence; an admin may also dispatch an enabled one on demand below.</p>

{{template "scantrigger" .}}

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
`
