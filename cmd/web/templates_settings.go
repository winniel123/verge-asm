package main

import "html/template"

// Settings screen — canonical `/settings`, ported from examples/console/Settings.jsx
// as seven query-param sub-tabs (scans · vantages · channels · messages · delivery
// · access · integrations). It folds today's settings (accounts, channels,
// retention), the scans monitor, the vantages, the message panel, the verge-core
// port set, the delivery record, and source enablement, plus the shared
// `forbidden` page. Each section renders its real data; the shell/CSS are T0's, so
// this reuses the shared pageCSS classes and adds no second stylesheet.
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
<div class="tabs">
<a class="tab{{if eq .Tab "scans"}} active{{end}}" href="/settings?tab=scans">Scans</a>
<a class="tab{{if eq .Tab "vantages"}} active{{end}}" href="/settings?tab=vantages">Vantages</a>
<a class="tab{{if eq .Tab "sso"}} active{{end}}" href="/settings?tab=sso">Single sign-on</a>
<a class="tab{{if eq .Tab "team"}} active{{end}}" href="/settings?tab=team">Team</a>
<a class="tab{{if eq .Tab "audit"}} active{{end}}" href="/settings?tab=audit">Audit log</a>
<a class="tab{{if eq .Tab "sources"}} active{{end}}" href="/settings?tab=sources">Sources</a>
<a class="tab{{if eq .Tab "aperture"}} active{{end}}" href="/settings?tab=aperture">Port aperture</a>
<a class="tab{{if eq .Tab "instance"}} active{{end}}" href="/settings?tab=instance">Health</a>
<a class="tab{{if eq .Tab "channels"}} active{{end}}" href="/settings?tab=channels">Channels</a>
<a class="tab{{if eq .Tab "integrations"}} active{{end}}" href="/settings?tab=integrations">Integrations</a>
<a class="tab{{if eq .Tab "messages"}} active{{end}}" href="/settings?tab=messages">Messages</a>
<a class="tab{{if eq .Tab "delivery"}} active{{end}}" href="/settings?tab=delivery">Delivery</a>
</div>
{{if .Notice}}<div class="notice">{{.Notice}}</div>{{end}}

{{if eq .Tab "scans"}}{{template "settings-scans" .}}{{end}}
{{if eq .Tab "vantages"}}{{template "settings-vantages" .}}{{end}}
{{if eq .Tab "sso"}}{{template "settings-sso" .}}{{end}}
{{if eq .Tab "team"}}{{template "settings-team" .}}{{end}}
{{if eq .Tab "audit"}}{{template "settings-audit" .}}{{end}}
{{if eq .Tab "sources"}}{{template "settings-sources" .}}{{end}}
{{if eq .Tab "aperture"}}{{template "settings-aperture" .}}{{end}}
{{if eq .Tab "instance"}}{{template "settings-instance" .}}{{end}}
{{if eq .Tab "channels"}}{{template "settings-channels" .}}{{end}}
{{if eq .Tab "integrations"}}{{template "settings-integrations" .}}{{end}}
{{if eq .Tab "messages"}}{{template "settings-messages" .}}{{end}}
{{if eq .Tab "delivery"}}{{template "settings-delivery" .}}{{end}}
</main>
{{template "foot" .}}{{end}}

{{define "settings-scans"}}
<div class="microlabel">Operational · scans</div>
<h2>Scans</h2>
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
{{end}}

{{define "settings-vantages"}}
<div class="microlabel">Scanning · vantages</div>
<h2>Vantages</h2>
<p>The network positions a scan measures from. A vantage carries a verified class —
internet or internal — and the recursive resolver it looks through; the act of
provisioning one on Scope declares it sits on the internet and opens the Reach and
Exposure timelines. This is a read: provisioning lives on Scope.</p>
{{if .Vantages}}
<div class="section">
<table>
<thead><tr><th>Vantage</th><th>Class</th><th>Resolver</th><th>Endpoint</th><th>Availability</th></tr></thead>
<tbody>
{{range .Vantages}}<tr>
<td class="mono">{{.Name}}</td>
<td><span class="badge">{{.Class}}</span></td>
<td class="mono">{{.Resolver}}</td>
<td class="mono">{{if .Endpoint}}{{.Endpoint}}{{else}}<span class="muted">—</span>{{end}}</td>
<td>{{if eq .Availability "available"}}<span class="badge">available</span>{{else if eq .Availability "unavailable"}}<span class="badge off">unavailable</span>{{else}}<span class="muted">{{.Availability}}</span>{{end}}</td>
</tr>{{end}}
</tbody>
</table>
</div>
{{else}}
<div class="emptystate">
<h2>No vantage provisioned</h2>
<p>Only the shipped resolver position exists, so the Reach and Exposure timelines
have not opened. Provision a vantage on <a href="/scope">Scope</a> to measure from
the internet.</p>
</div>
{{end}}

{{if .Probers}}
<div class="section">
<div class="microlabel">Probers</div>
<h2>Provisioned internet vantages</h2>
<p class="muted">The worker owns the keypair and generates it out of band; only the public half is published here — the private half never leaves the instance. A host key is pinned on first connection, and a later change is a hard failure, never a prompt.</p>
{{range .Probers}}
<div class="section">
<div class="custody-head">
<div><div class="mono">{{.Endpoint}}</div><div class="muted">username <span class="mono">{{.Username}}</span></div></div>
<div>{{if eq .Availability "available"}}<span class="badge">available</span>{{else}}<span class="muted">{{.Availability}}</span>{{end}}</div>
</div>
<div class="kv"><div class="k">Host key</div><div>{{if .HostKeyPinned}}<span class="badge">pinned</span>{{else}}<span class="muted">awaiting first connection</span>{{end}}</div></div>
<div class="kv"><div class="k">Public key</div><div>{{if .KeySet}}<span class="badge">set</span>{{else}}<span class="muted">not set — the worker has not published one yet</span>{{end}}</div></div>
{{if .KeySet}}
<p class="muted" style="margin:8px 0 4px">Install the public half in <span class="mono">{{.Username}}@{{.Endpoint}}</span>'s authorized_keys:</p>
<div style="display:flex;align-items:center;gap:8px">
<code class="mono cvval" style="flex:1;min-width:0;word-break:break-all;background:var(--sunken);border:1px solid var(--hairline);border-radius:var(--r-sm);padding:var(--space-3)">{{.PublicKey}}</code>
<button type="button" class="btn secondary" style="flex:none" onclick="var v=this.parentNode.querySelector('.cvval').textContent;if(navigator.clipboard){navigator.clipboard.writeText(v);this.textContent='Copied';}">Copy</button>
</div>
{{end}}
</div>
{{end}}
<p class="muted">Declare the vantage's observed egress as an address scope from <a href="/scope">Scope</a> so the estate knows its own outbound address.</p>
</div>
{{end}}
{{end}}

{{define "settings-channels"}}
<div class="microlabel">Delivery · channels</div>
<h2>Channels</h2>
<p>A channel is where messages go: an absolute <code>https</code> URL, an optional
signing secret, and the subset of classes it carries. None ships configured, and a
channel is one-way — it grants no read of anything.</p>
{{if .ChanError}}<div class="error">{{.ChanError}}</div>{{end}}
{{if .Channels}}
<div class="section">
<table>
<thead><tr><th>URL</th><th>Classes</th><th>Secret</th><th>State</th><th>Declared by</th>{{if .IsAdmin}}<th></th>{{end}}</tr></thead>
<tbody>
{{range .Channels}}<tr>
<td class="mono">{{.URL}}</td>
<td>{{if .Drift}}<span class="badge">drift</span> {{end}}{{if .Coverage}}<span class="badge">coverage</span> {{end}}{{if .Clock}}<span class="badge">clock</span>{{end}}</td>
<td>{{if .HasSecret}}<span class="badge">set</span>{{else}}<span class="muted">not set</span>{{end}}</td>
<td>{{if .Enabled}}<span class="badge">enabled</span>{{else}}<span class="muted">disabled</span>{{end}}</td>
<td class="mono">{{.By}}<br><span class="muted">{{.At}}</span></td>
{{if $.IsAdmin}}<td>
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
</td>{{end}}
</tr>{{end}}
</tbody>
</table>
</div>
{{else}}
<div class="emptystate">
<h2>No channels declared</h2>
<p>Nothing is configured, so every message is written and rendered in the store and
carried nowhere else. Declare a channel to have messages POSTed out.</p>
</div>
{{end}}

{{if .IsAdmin}}
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
{{end}}
{{end}}

{{define "settings-messages"}}
<div class="rulehead">
<div><div class="microlabel">Delivery · messages</div>
<h2>Messages</h2></div>
{{if .Messages}}<form method="post" action="/messages/read-all"><button class="secondary" type="submit">Mark all read</button></form>{{end}}
</div>
<p>Every message that has fired, newest first. Each names what moved — the estate's
own object, our own aperture, your declared input, or a threshold crossed — and
links to what it fired at. The store carries the message and never the estate: the
fact itself is in the timelines.</p>
{{if not .Messages}}
<div class="emptystate">
<h2>No message has fired</h2>
<p>Messages appear here as the estate moves — declare a seed and run the first batch
to begin measuring.</p>
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
{{end}}

{{define "settings-delivery"}}
<div class="microlabel">Delivery · operational record</div>
<h2>Deliveries</h2>
<p>The operational record: every message's outcome to each routed channel, and the
two retention dials that govern how long the record stays readable. A delivery has no
cause and never touches Coverage — an undelivered POST is legible here, joined to the
message it failed to carry. The hot-tier port set moved to its own Port aperture tab.</p>

<div class="section">
<div class="microlabel">Operational record</div>
<h2>Delivery outcomes</h2>
{{if .Deliveries}}
<table>
<thead><tr><th>Channel</th><th>Outcome</th></tr></thead>
<tbody>
{{range .Deliveries}}<tr>
<td class="mono">{{.ChannelHost}}</td>
<td>{{if .Failed}}<span class="badge off">undelivered</span>{{else}}<span class="badge">{{.State}}</span>{{end}}</td>
</tr>{{end}}
</tbody>
</table>
{{else}}
<p class="muted">No delivery has been attempted yet. Once a channel is declared and a
message fires, each outbound POST's outcome is recorded here.</p>
{{end}}
</div>

<div class="section">
<div class="microlabel">Retention</div>
<h2>Retention dials</h2>
<p>Retention is what may still be read, never age. Dispatch retention is a multiple of the
slowest enabled scan's cadence, floored at 2 cadences; 0 leaves it unbounded, the default. The
observation-currency floor lands with later work; for now zero means no operator floor.</p>
{{if .RetError}}<div class="error">{{.RetError}}</div>{{end}}
{{if .IsAdmin}}
<form method="post" action="/settings/retention">
<div class="dial">
<label><span>Observation-currency floor</span><input name="observation_currency_days" inputmode="numeric" value="{{if .RetError}}{{.RetObs}}{{else}}{{.Retention.ObservationCurrencyDays}}{{end}}" required><span class="unit">days</span></label>
<label><span>Dispatch retention</span><input name="dispatch_cadence_multiple" inputmode="numeric" value="{{if .RetError}}{{.RetDispatch}}{{else}}{{.Retention.DispatchCadenceMultiple}}{{end}}" required><span class="unit">× slowest scan cadence</span></label>
<button type="submit">Save dials</button>
</div>
</form>
{{else}}
<div class="kv"><div class="k">Observation floor</div><div class="mono">{{.Retention.ObservationCurrencyDays}} days</div></div>
<div class="kv"><div class="k">Dispatch retention</div><div class="mono">{{.Retention.DispatchCadenceMultiple}} × slowest scan cadence</div></div>
{{end}}
{{if .Retention.UpdatedAt}}<p class="muted" style="margin-top:12px">Last changed {{.Retention.UpdatedAt}}{{if .Retention.UpdatedBy}} by <span class="mono">{{.Retention.UpdatedBy}}</span>{{end}}.</p>{{end}}
</div>

{{end}}
`

// settingsSectionTemplates carries the V3 section deltas ported from
// examples/console/Settings.jsx (T18, #313, ADR-0110): the single-sign-on
// not-configured state, the Team surface (members, roles, and the change-role /
// require-re-enrollment / remove / invite dialogs), the audit-log honest empty
// state, the Port-aperture tab (release-authored sensitive tier read-only, editable
// frequency tier), and instance health. Parsed into the same shared template set;
// no design-system component is authored here (ADR-0109) — only the existing token
// vocabulary and pageCSS classes are used.
var _ = template.Must(tmpl.Parse(settingsSectionTemplates))

const settingsSectionTemplates = `
{{define "settings-sso"}}
<div class="microlabel">Access · single sign-on</div>
<h2>Single sign-on</h2>
<p>Sign-on via <strong>OpenID Connect</strong> (#293). A provider authenticates an
existing account by its username claim — it never creates accounts, and header-trust
reverse-proxy auth stays refused. Each enabled provider renders a button on the
sign-in screen; accounts can still sign in with a password and two-factor.</p>
<p class="muted">Your own credentials and two-factor live on your <a href="/profile">Profile</a>.</p>
{{if .SSOError}}<div class="error">{{.SSOError}}</div>{{end}}
{{if .SSOProviders}}
<div class="section">
<table>
<thead><tr><th>Provider</th><th>Issuer</th><th>Client</th><th>Claim</th><th>Secret</th><th>State</th><th>Declared by</th><th></th></tr></thead>
<tbody>
{{range .SSOProviders}}<tr>
<td>{{.Name}} <span class="muted mono">/{{.Slug}}</span></td>
<td class="mono">{{.Issuer}}</td>
<td class="mono">{{.ClientID}}</td>
<td class="mono">{{.UsernameClaim}}</td>
<td>{{if .HasSecret}}<span class="badge">set</span>{{else}}<span class="muted">none</span>{{end}}</td>
<td>{{if .Enabled}}<span class="badge">enabled</span>{{else}}<span class="muted">disabled</span>{{end}}</td>
<td class="mono">{{.CreatedBy}}<br><span class="muted">{{.CreatedAt}}</span></td>
<td>
<details class="edit">
<summary>Edit</summary>
<div class="section">
<form method="post" action="/settings/sso/update">
<input type="hidden" name="id" value="{{.ID}}">
<label><span>Display name</span><input name="name" value="{{.Name}}" required></label>
<label><span>Slug</span><input name="slug" value="{{.Slug}}" required></label>
<label><span>Issuer URL</span><input name="issuer" value="{{.Issuer}}" required></label>
<label><span>Client ID</span><input name="client_id" value="{{.ClientID}}" autocomplete="off" required></label>
<label><span>Username claim</span><input name="username_claim" value="{{.UsernameClaim}}"></label>
<label class="check"><input type="checkbox" name="enabled"{{if .Enabled}} checked{{end}}><span>enabled</span></label>
<div class="row" style="margin-top:12px"><button type="submit">Save provider</button></div>
</form>
<form method="post" action="/settings/sso/secret" style="margin-top:12px">
<input type="hidden" name="id" value="{{.ID}}">
<label><span>Replace client secret</span><input name="client_secret" type="password" autocomplete="off" placeholder="{{if .HasSecret}}set — leave blank to keep{{else}}not set — leave blank for none{{end}}"></label>
{{if .HasSecret}}<label class="check"><input type="checkbox" name="clear_secret"><span>clear the secret</span></label>{{end}}
<div class="row" style="margin-top:12px"><button type="submit">Update secret</button></div>
</form>
<form method="post" action="/settings/sso/delete" style="margin-top:12px"><input type="hidden" name="id" value="{{.ID}}"><button class="secondary" type="submit">Remove provider</button></form>
</div>
</details>
</td>
</tr>{{end}}
</tbody>
</table>
</div>
{{else}}
<div class="emptystate">
<h2>No identity provider configured</h2>
<p>Nothing is configured, so the sign-in screen offers no single-sign-on button and
accounts sign in with a password and two-factor. Add an OpenID Connect provider below
to enable SSO.</p>
</div>
{{end}}

<div class="section">
<h2>Add an OpenID Connect provider</h2>
<form method="post" action="/settings/sso">
<label><span>Display name</span><input name="name" value="{{.SSOName}}" placeholder="Okta" required></label>
<label><span>Slug</span><input name="slug" value="{{.SSOSlug}}" placeholder="okta" autocomplete="off" required></label>
<label><span>Issuer URL</span><input name="issuer" value="{{.SSOIssuer}}" placeholder="https://example.okta.com" autocomplete="off" required></label>
<label><span>Client ID</span><input name="client_id" value="{{.SSOClientID}}" autocomplete="off" required></label>
<label><span>Client secret (optional)</span><input name="client_secret" type="password" autocomplete="off" placeholder="confidential clients only — never shown again"></label>
<label><span>Username claim</span><input name="username_claim" value="{{.SSOClaim}}" placeholder="preferred_username"></label>
<button type="submit">Add provider</button>
</form>
</div>
{{end}}

{{define "settings-team"}}
<div class="microlabel">Access · team</div>
<div class="rulehead">
<div><h2 style="margin:0">Who can sign in</h2></div>
<a class="btn" href="/settings?tab=team&amp;invite=1">Invite</a>
</div>
{{if .TeamError}}<div class="error">{{.TeamError}}</div>{{end}}
{{if .RoleError}}<div class="error">{{.RoleError}}</div>{{end}}
<div class="section">
<table>
<thead><tr><th>Member</th><th>Role</th><th>Two-factor</th><th>Member since</th>{{if .IsAdmin}}<th></th>{{end}}</tr></thead>
<tbody>
{{range .Members}}<tr>
<td class="mono">{{.Username}}{{if .IsSelf}} <span class="muted">(you)</span>{{end}}</td>
<td><span class="badge">{{.Role}}</span></td>
<td>{{if .TotpEnabled}}<span class="badge">enrolled</span>{{else}}<span class="badge off">not enrolled</span>{{end}}</td>
<td class="mono">{{if .At}}{{.At}}{{else}}<span class="muted">—</span>{{end}}</td>
{{if $.IsAdmin}}<td>{{if .IsSelf}}<span class="muted">—</span>{{else}}
<details class="edit">
<summary>Actions</summary>
<div class="section">
<div class="rowlink"><a href="/settings?tab=team&amp;role={{.ID}}">Change role</a></div>
<div class="rowlink"><a href="/settings?tab=team&amp;reenroll={{.ID}}">Require re-enrollment</a></div>
<div class="rowlink"><a href="/settings?tab=team&amp;remove={{.ID}}">Remove member</a></div>
</div>
</details>
{{end}}</td>{{end}}
</tr>{{end}}
</tbody>
</table>
</div>

<div class="section">
<div class="microlabel">Roles</div>
<h2>What each role can do</h2>
<div class="kv"><div class="k"><span class="badge">admin</span></div><div>performs declared acts — seeds, scans, channels, annotations, team, instance</div></div>
<div class="kv"><div class="k"><span class="badge">viewer</span></div><div>reads everything, changes nothing — including the sources catalogue</div></div>
</div>

{{if .RoleTarget}}{{with .RoleTarget}}
<a class="scrim" href="/settings?tab=team" aria-label="Cancel"></a>
<div class="dialog-panel" role="dialog" aria-modal="true" aria-label="Change role" style="position:fixed;top:14vh;left:50%;transform:translateX(-50%);z-index:42">
<div class="microlabel" style="margin-bottom:8px">Change role</div>
<h2 style="margin:0 0 4px">Change role</h2>
<p class="muted" style="margin:0 0 var(--space-4)">{{.Username}}</p>
<form method="post" action="/settings/accounts/role">
<input type="hidden" name="id" value="{{.ID}}">
<label><span>Role</span>
<select name="role" data-current="{{.Role}}" onchange="document.getElementById('rolesave').disabled=(this.value===this.getAttribute('data-current'))">
<option value="admin"{{if eq .Role "admin"}} selected{{end}}>admin</option>
<option value="viewer"{{if eq .Role "viewer"}} selected{{end}}>viewer</option>
</select></label>
<div class="dialog-actions">
<a class="btn ghost" href="/settings?tab=team">Cancel</a>
<button id="rolesave" type="submit" disabled>Save role</button>
</div>
</form>
</div>
{{end}}{{end}}

{{if .ReenrollTarget}}{{with .ReenrollTarget}}
<a class="scrim" href="/settings?tab=team" aria-label="Cancel"></a>
<div class="dialog-panel" role="dialog" aria-modal="true" aria-label="Require re-enrollment" style="position:fixed;top:14vh;left:50%;transform:translateX(-50%);z-index:42">
<div class="microlabel" style="margin-bottom:8px">Require re-enrollment</div>
<h2 style="margin:0 0 8px">Require re-enrollment</h2>
<p style="margin:0 0 4px">{{.Username}}'s current authenticator stops working immediately; the next sign-in walks them through two-factor setup again.</p>
<p class="muted" style="margin:0 0 var(--space-4)">Active sessions stay signed in.</p>
<div class="dialog-actions">
<a class="btn ghost" href="/settings?tab=team">Cancel</a>
<form method="post" action="/settings/accounts/reenroll" style="margin:0"><input type="hidden" name="id" value="{{.ID}}"><button class="secondary" type="submit">Require re-enrollment</button></form>
</div>
</div>
{{end}}{{end}}

{{if .RemoveTarget}}{{with .RemoveTarget}}
<a class="scrim" href="/settings?tab=team" aria-label="Cancel"></a>
<div class="dialog-panel" role="dialog" aria-modal="true" aria-label="Remove member" style="position:fixed;top:14vh;left:50%;transform:translateX(-50%);z-index:42">
<div class="microlabel" style="margin-bottom:8px">Remove member</div>
<h2 style="margin:0 0 8px">Remove {{.Username}}</h2>
<p style="margin:0 0 4px">{{.Username}} loses access to this deployment.</p>
<p class="muted" style="margin:0 0 var(--space-4)">Their annotations and audit history stay attributed. Personal API tokens are revoked.</p>
{{if $.RemoveError}}<div class="error">{{$.RemoveError}}</div>{{end}}
<form method="post" action="/settings/accounts/remove">
<input type="hidden" name="id" value="{{.ID}}">
<label><span>Type <span class="mono">{{.Username}}</span> to confirm</span><input name="confirm_name" autocomplete="off" spellcheck="false" autofocus required></label>
<div class="dialog-actions">
<a class="btn ghost" href="/settings?tab=team">Cancel</a>
<button class="danger" type="submit">Remove member</button>
</div>
</form>
</div>
{{end}}{{end}}

{{if .InviteOpen}}
<a class="scrim" href="/settings?tab=team" aria-label="Close"></a>
<div class="dialog-panel" role="dialog" aria-modal="true" aria-label="Invite a member" style="position:fixed;top:14vh;left:50%;transform:translateX(-50%);z-index:42">
<div class="microlabel" style="margin-bottom:8px">Invite a member</div>
{{if .InviteLink}}
<h2 style="margin:0 0 12px">Copy the join link now</h2>
<div class="banner warn" style="margin:0 0 var(--space-4)">Shown once — hand it to the invitee out of band. Verge keeps only a hash, so it cannot be shown again.</div>
<div style="display:flex;align-items:center;gap:8px">
<code class="mono cvval" style="flex:1;min-width:0;word-break:break-all;background:var(--sunken);border:1px solid var(--hairline);border-radius:var(--r-sm);padding:var(--space-3)">{{.InviteLink}}</code>
<button type="button" class="btn secondary" style="flex:none" onclick="var v=this.parentNode.querySelector('.cvval').textContent;if(navigator.clipboard){navigator.clipboard.writeText(v);this.textContent='Copied';}">Copy</button>
</div>
<p class="muted" style="margin:var(--space-3) 0 0;font-size:12px">The role applies on acceptance; the link expires in 7 days.</p>
<div class="dialog-actions"><a class="btn" href="/settings?tab=team">Done</a></div>
{{else}}
<h2 style="margin:0 0 4px">Invite a member</h2>
<p class="muted" style="margin:0 0 var(--space-4);font-size:12.5px">They get a join link; the role applies on acceptance. This build has no mail, so you hand the link over out of band.</p>
{{if .TeamError}}<div class="error">{{.TeamError}}</div>{{end}}
<form method="post" action="/settings/accounts">
<label><span>Role</span><select name="role"><option value="viewer"{{if eq .InviteRole "viewer"}} selected{{end}}>viewer</option><option value="admin"{{if eq .InviteRole "admin"}} selected{{end}}>admin</option></select></label>
<div class="dialog-actions">
<a class="btn ghost" href="/settings?tab=team">Cancel</a>
<button type="submit">Create invite</button>
</div>
</form>
{{end}}
</div>
{{end}}
{{end}}

{{define "settings-audit"}}
<div class="microlabel">Access · operational record</div>
<h2>Audit log</h2>
<p>Who did what, when. This build keeps no separate queryable log of admin acts —
source enablement, for one, keeps no log line of its own and is dated by the batch
whose recorded source set it moved — so there is nothing to page through here yet.</p>
{{if .AuditRows}}
<div class="section">
<table>
<thead><tr><th>When</th><th>Actor</th><th>Action</th><th>Subject</th></tr></thead>
<tbody>
{{range .AuditRows}}<tr>
<td class="mono">{{.When}}</td><td class="mono">{{.Actor}}</td><td class="mono">{{.Action}}</td><td class="mono">{{.Subject}}</td>
</tr>{{end}}
</tbody>
</table>
</div>
{{else}}
<div class="emptystate">
<h2>No audit log</h2>
<p>The operational records this build does keep are the <a href="/settings?tab=delivery">delivery record</a> and the <a href="/messages">message store</a> — each fact is legible where it fired.</p>
</div>
{{end}}
{{end}}

{{define "settings-aperture"}}
<div class="microlabel">Discovery · port aperture</div>
<h2>Port aperture</h2>
<p>The daily hot-tier port set the census walks: the union of a release-authored
<em>sensitive</em> tier and an operator-editable <em>frequency</em> tier, plus any
port previously seen on the subject. Only the TCP pairs are probed; the {{.UDPCount}}
UDP pairs are recorded in scope and never probed.</p>

<div class="section">
<div class="rulehead">
<div><div class="microlabel">verge-core</div><h2 style="margin:0">Sensitive tier</h2></div>
<span class="badge">locked</span>
</div>
<div class="row" style="flex-wrap:wrap;gap:8px;margin-bottom:12px">
{{range .Sensitive}}<span class="badge"><svg aria-hidden="true" viewBox="0 0 24 24" width="11" height="11" fill="none" stroke="currentColor" stroke-width="1.75" style="vertical-align:-1px;margin-right:4px"><rect width="18" height="11" x="3" y="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>{{.Port}}/{{.Transport}}</span>{{end}}
</div>
<div class="notice">Not editable, on purpose. A port you can hide is a signal you can silence. The sensitive tier is release-authored and moves only with the release — {{.Counts.Sensitive}} pairs, and there is no control to move one.</div>
</div>

<div class="section">
<div class="microlabel">verge-core</div>
<h2>Frequency tier</h2>
<p class="muted">{{.Counts.Frequency}} TCP ports. Admins may widen or narrow this tier. A port also on the sensitive tier stays probed even if you remove it here — the union keeps it — which is why removing it cannot move the sensitive tier. Union: {{.Counts.Union}} pairs ({{.Counts.TCP}} TCP, {{.Counts.UDP}} UDP).</p>
{{if .VCError}}<div class="error">{{.VCError}}</div>{{end}}
{{if .IsAdmin}}
<form method="post" action="/verge-core/frequency" class="inlineform" style="margin-bottom:12px">
<input type="hidden" name="action" value="add">
<label><span>Add a frequency port</span><input name="port" inputmode="numeric" value="{{.VCPort}}" placeholder="8443" autocomplete="off" required></label>
<button type="submit">Add to frequency</button>
</form>
{{end}}
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
{{end}}

{{define "settings-instance"}}
<div class="microlabel">Instance · health</div>
<h2>Health</h2>
<p>What this deployment is and how it is doing right now — real reads only, no
version string or queue figure fabricated where the datum does not exist.</p>
<div class="section">
<div class="microlabel">Instance</div>
<div class="kv"><div class="k">Build</div><div class="mono">{{.Licence}}</div></div>
<div class="kv"><div class="k">Uptime</div><div class="mono">{{.Uptime}} <span class="muted">since last restart</span></div></div>
<div class="kv"><div class="k">Database</div><div><span class="badge">postgres · reachable</span></div></div>
</div>
<div class="section">
<div class="microlabel">Fleet</div>
<h2>Vantages</h2>
{{if .Fleet}}
<table>
<thead><tr><th>Vantage</th><th>Class</th><th>Availability</th></tr></thead>
<tbody>
{{range .Fleet}}<tr>
<td class="mono">{{.Name}}</td>
<td><span class="badge">{{.Class}}</span></td>
<td>{{if eq .Availability "available"}}<span class="badge">available</span>{{else}}<span class="muted">{{.Availability}}</span>{{end}}</td>
</tr>{{end}}
</tbody>
</table>
{{else}}
<p class="muted">No vantage is provisioned, so only the shipped resolver position exists. Provision one on <a href="/scope">Scope</a>.</p>
{{end}}
</div>
{{end}}
`
