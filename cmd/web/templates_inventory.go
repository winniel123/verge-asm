package main

import "html/template"

// Inventory screen — canonical `/inventory`. Folds today's inventory + subjects
// list + the Name/Service/Endpoint detail views (and the shared `recordrows`
// partial and the `subject-missing` page). The screen ticket (T-Inventory)
// rewrites the body against examples/console/Inventory.jsx (saved views, column
// picker, density, hover peeks). Ported verbatim for T0.
var _ = template.Must(tmpl.Parse(inventoryTemplates))

const inventoryTemplates = `
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

`
