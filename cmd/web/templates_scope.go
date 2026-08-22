package main

import "html/template"

// Scope screen — canonical `/scope`. Folds today's seeds surface (seeds, custody,
// zone files, cold tier, exclusions, probers). The screen ticket (T-Scope)
// rewrites the body against examples/console/Scope.jsx (TagInput validation +
// refusals, proposals, exclusions, custody, coverage). Ported verbatim for T0.
var _ = template.Must(tmpl.Parse(scopeTemplates))

const scopeTemplates = `
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

`
