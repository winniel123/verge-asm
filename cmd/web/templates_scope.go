package main

import "html/template"

// Scope screen — canonical `/scope` (#278, V2 console map #275). Composed after
// design-system/examples/console/Scope.jsx (02-console.jpg): the seed TagInput
// with validation + declaration-refusal states, registry proposals (confirm one,
// decline many), the exclusion editor, the custody extension toggle, and the
// coverage-message list. Section order follows the spec: seeds -> proposals ->
// exclusions -> custody -> coverage. The operational configuration that has no
// place in the JSX mock but carries live POST actions — supplied zone files, the
// full-range (cold) tier opt-in, and provisioned probers — is kept verbatim below
// the fold so every existing mutation and its tests stay green.
//
// The block is named "scope"; renderSeeds (seeds.go) renders it for GET /scope,
// the canonical home. GET /seeds now permanently redirects here (T10, #286). The
// POST actions keep their /seeds* paths (only the GET presentation moved) and now
// answer 303 to /scope, unchanged in every other respect. The presentation is
// template-local CSS translated from design-system/components/* within the
// existing token vocabulary — restyling, not authoring (ADR-0109); no
// design-system component is authored here and no shared pageCSS class is edited.
var _ = template.Must(tmpl.Parse(scopeTemplates))

const scopeTemplates = `
{{define "scope"}}{{template "head" .}}
<style>
.scope-header { display:flex; flex-direction:column; gap:2px; margin-bottom:var(--space-5); }
.scope-header h1 { margin:0; font-size:21px; }
.scope-header .sub { font-size:12.5px; color:var(--muted); }
.tagfield { display:flex; flex-wrap:wrap; align-items:center; gap:6px; min-height:36px;
  padding:6px 10px; background:var(--surface); border:1px solid var(--hairline); border-radius:var(--r-md); }
.tagchip { display:inline-flex; align-items:center; gap:6px; height:24px; padding:0 8px;
  border-radius:var(--r-sm); background:var(--sunken); border:1px solid var(--hairline);
  color:var(--body); font-family:var(--mono); font-size:11.5px; white-space:nowrap; }
.tagchip .badge { font-size:9px; padding:0 5px; }
.taghint { font-size:11.5px; color:var(--muted); margin:6px 0 0; }
.refusal { display:flex; gap:10px; align-items:flex-start; padding:12px 14px; background:var(--danger-soft);
  border:1px solid var(--danger-border); border-radius:var(--r-md); margin-bottom:var(--space-4); }
.refusal .rf-ic { color:var(--danger); flex:none; margin-top:1px; }
.refusal .rf-ic svg { width:15px; height:15px; display:block; }
.refusal .rf-title { font-weight:600; font-size:13px; color:var(--ink); }
.refusal .rf-title .mono { font-size:12.5px; }
.refusal .rf-reason { font-size:12.5px; color:var(--body); margin-top:4px; }
.switchline { display:flex; align-items:center; gap:10px; flex-wrap:wrap; }
.switchline form { margin:0; }
.switch { display:inline-block; width:36px; height:20px; border-radius:var(--r-full); position:relative;
  flex:none; border:1px solid var(--border-strong); background:var(--sunken); }
.switch.on { background:var(--accent); border-color:var(--accent); }
.switch .knob { position:absolute; top:2px; left:2px; width:14px; height:14px; border-radius:var(--r-full);
  background:var(--surface); box-shadow:var(--shadow-xs); }
.switch.on .knob { left:18px; background:var(--on-accent); }
.switchlabel { font-size:13px; color:var(--body); }
.exclrow { display:flex; align-items:center; gap:10px; padding:7px 10px; background:var(--sunken);
  border-radius:10px; margin-bottom:6px; }
.exclrow form { margin:0 0 0 auto; }
.exclrow .exval { font-family:var(--mono); font-size:12.5px; color:var(--body); overflow-wrap:anywhere; }
.covmsg { display:grid; grid-template-columns:auto 1fr auto; gap:12px; align-items:start;
  padding:11px 0; border-top:1px solid var(--hairline); }
.covmsg:first-child { border-top:none; }
.covmsg .subj { font-family:var(--mono); font-size:12px; font-weight:600; color:var(--ink); }
.covmsg .txt { font-size:12.5px; color:var(--muted); line-height:1.5; }
.covmsg .when { font-family:var(--mono); font-size:11px; color:var(--muted); white-space:nowrap; }
.seed-meta { font-family:var(--mono); font-size:11px; color:var(--muted); margin-top:2px; }
.zfile { display:flex; align-items:center; gap:10px; flex-wrap:wrap; padding:9px 0;
  border-top:1px solid var(--hairline); }
.zfile:first-child { border-top:none; }
.zfile .zname { font-family:var(--mono); font-size:12px; font-weight:500; color:var(--body); overflow-wrap:anywhere; }
.zfile .zmeta { font-family:var(--mono); font-size:11.5px; color:var(--muted); }
.zfile .zgap { margin-left:auto; }
.zbadge { display:inline-flex; align-items:center; gap:6px; font-family:var(--mono); font-size:11px;
  font-weight:500; border-radius:var(--r-full); padding:2px 9px; white-space:nowrap;
  background:var(--warn-soft); border:1px solid var(--warn-border); color:var(--warn); }
.zbadge .zdot { width:6px; height:6px; border-radius:50%; background:var(--warn); flex:none; }
.cardhead { display:flex; align-items:center; gap:var(--space-3); margin-bottom:var(--space-4); }
.cardhead .headings { display:flex; flex-direction:column; gap:3px; }
.cardhead h2 { margin:0; font-size:15px; }
.cardhead .count { margin-left:auto; font-family:var(--mono); font-size:10.5px; font-weight:500;
  padding:1px 7px; border-radius:var(--r-full); background:var(--sunken); color:var(--body); }
/* Declared name tree — TreeView.jsx translated to a JS-free <details> disclosure
   within the existing tokens (restyle, not authoring; ADR-0109). */
.nametree { display:flex; flex-direction:column; }
.nametree .ntrow { display:flex; align-items:center; gap:7px; padding:5px 8px;
  border-radius:var(--r-sm); }
.nametree .ntrow:hover { background:var(--sunken); }
.nametree summary.ntrow { cursor:pointer; list-style:none; }
.nametree summary.ntrow::-webkit-details-marker { display:none; }
.nametree .nt-caret { width:12px; height:12px; flex:none; color:var(--muted);
  transform:rotate(-90deg); transition:transform 0.18s var(--ease-out, ease); }
.nametree details[open] > summary.ntrow .nt-caret { transform:rotate(0); }
.nametree .nt-spacer { width:12px; flex:none; }
.nametree .nt-dot { width:6px; height:6px; border-radius:var(--r-full); flex:none; }
.nametree .nt-label { font-family:var(--mono); font-size:12.5px; font-weight:500;
  color:var(--body); white-space:nowrap; overflow:hidden; text-overflow:ellipsis; }
.nametree .nt-count { margin-left:auto; font-family:var(--mono); font-size:10.5px; color:var(--muted); }
.nametree .nt-leaf { padding-left:26px; }
.nametree .nt-kids { display:flex; flex-direction:column; }
.nametree .nt-empty { padding:6px 8px 6px 26px; font-size:12px; color:var(--muted); }
</style>
{{template "chrome" .}}
<main style="display:flex;flex-direction:column;gap:var(--space-5)">

{{if .Notice}}<div class="notice">{{.Notice}}</div>{{end}}

<header class="scope-header">
  <h1>Scope</h1>
  <span class="sub">What Verge is allowed to look at: seeds, proposals, exclusions, custody.</span>
</header>

<!-- Seeds ---------------------------------------------------------------->
<section class="section" style="margin-bottom:0">
  <div class="cardhead">
    <div class="headings"><span class="microlabel">Seeds</span><h2>Declared scopes</h2></div>
  </div>
  <p>A seed is where you assert your estate ends: a name scope &#8212; a registrable domain &#8212; or an
  address scope &#8212; a CIDR block of up to {{.AddressCap}} addresses.</p>

  {{if .Seeds}}
  <div class="tagfield" style="margin-bottom:var(--space-4)">
    {{range .Seeds}}<span class="tagchip" id="seed-{{.Anchor}}"><span class="badge">{{if .IsAddress}}address{{else}}name{{end}}</span><span class="mono">{{.Scope}}</span></span>{{end}}
  </div>
  {{else}}
  <div class="emptystate" style="margin-bottom:var(--space-4)">
    <div class="microlabel">No scopes declared</div>
    <h2>Nothing is declared yet</h2>
    <p style="max-width:60ch;margin:var(--space-3) auto 0">Declare a domain or a CIDR block to set where your estate begins.</p>
  </div>
  {{end}}

  {{if .IsAdmin}}
  {{if .FormError}}
  <div class="refusal">
    <span class="rf-ic"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75"><path d="M7.86 2h8.28L22 7.86v8.28L16.14 22H7.86L2 16.14V7.86L7.86 2z"/><path d="M12 8v4"/><path d="M12 16h.01"/></svg></span>
    <div>
      <div class="rf-title">Declaration refused{{if .FormScope}}: <span class="mono">{{.FormScope}}</span>{{end}}</div>
      <div class="rf-reason">{{.FormError}}</div>
    </div>
  </div>
  {{end}}
  <form method="post" action="/seeds" class="seedform">
    <label><span>Scope type</span><select name="kind">
      <option value="name"{{if ne .FormKind "address"}} selected{{end}}>name</option>
      <option value="address"{{if eq .FormKind "address"}} selected{{end}}>address</option>
    </select></label>
    <label class="scope"><span>Scope</span><input class="scope" name="scope" value="{{.FormScope}}" placeholder="acmecorp.io or 203.0.113.0/24" autocomplete="off" required></label>
    <button type="submit">Declare</button>
  </form>
  <p class="taghint">Names or address scopes &#183; a block wider than the {{.AddressCap}}-address cap is refused, never auto-corrected.</p>
  {{end}}
</section>

<!-- Declared name tree ---------------------------------------------------->
<section class="section" style="margin-bottom:0">
  <div class="cardhead">
    <div class="headings"><span class="microlabel">Names</span><h2>Declared name tree</h2></div>
  </div>
  <p>Every name in your estate, under the registrable domain that declares it. Each name carries the
  most urgent severity among the signals raised on it; a name with no signal carries none.</p>

  {{if .NameTree}}
  <div class="nametree">
    {{range .NameTree}}
    <details class="nt-root" open>
      <summary class="ntrow nt-domain">
        <svg class="nt-caret" viewBox="0 0 16 16" aria-hidden="true"><path d="M4 6l4 4 4-4" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round"/></svg>
        {{if .Sev}}<span class="nt-dot" style="background:var(--sev-{{.Sev}}-dot)"></span>{{end}}
        <span class="nt-label">{{.Label}}</span>
        <span class="nt-count">{{.Count}}</span>
      </summary>
      <div class="nt-kids">
        {{range .Children}}
        <div class="ntrow nt-leaf">
          <span class="nt-spacer"></span>
          {{if .Sev}}<span class="nt-dot" style="background:var(--sev-{{.Sev}}-dot)"></span>{{end}}
          <span class="nt-label">{{.Label}}</span>
        </div>
        {{else}}
        <div class="nt-empty">No name resolves under this scope yet. Each one a scan measures appears here with its severity.</div>
        {{end}}
      </div>
    </details>
    {{end}}
  </div>
  {{else}}
  <div class="emptystate">
    <div class="microlabel">No name scopes</div>
    <h2>No name tree to grow yet</h2>
    <p style="max-width:60ch;margin:var(--space-3) auto 0">A name tree grows under a declared name scope. Declare a domain above, then each name that
    resolves under it appears here with its severity.</p>
  </div>
  {{end}}
</section>

<!-- Proposals ------------------------------------------------------------->
{{template "proposals" .}}

<!-- Exclusions ------------------------------------------------------------>
<section class="section" style="margin-bottom:0">
  <div class="cardhead">
    <div class="headings"><span class="microlabel">Exclusions</span><h2>Never scanned</h2></div>
  </div>
  <p>An exclusion draws the boundary inwards: an exact name, a name subtree, or an address scope
  you declare is <em>not yours</em>. Excluding a name that still resolves is legal &#8212; <em>not
  mine</em> is a different claim from <em>not there</em> &#8212; and an excluded name is no longer queried.</p>

  {{if .Exclusions}}
  {{range .Exclusions}}
  <div class="exclrow">
    <span class="badge">{{.Kind}}</span>
    <span class="exval">{{if eq .Kind "subtree"}}*.{{end}}{{.Value}}</span>
    {{if $.IsAdmin}}<form method="post" action="/exclusions/delete"><input type="hidden" name="id" value="{{.ID}}"><button class="secondary" type="submit">Un-exclude</button></form>{{end}}
  </div>
  {{end}}
  {{else}}
  <div class="emptystate" style="margin-bottom:var(--space-4)">
    <div class="microlabel">No exclusions declared</div>
    <h2>Everything declared is in scope</h2>
    <p style="max-width:60ch;margin:var(--space-3) auto 0">Nothing is excluded. Everything inside your declared scopes is yours.</p>
  </div>
  {{end}}

  {{if .IsAdmin}}
  {{if .ExclError}}<div class="error">{{.ExclError}}</div>{{end}}
  <form method="post" action="/exclusions" class="seedform" style="margin-top:var(--space-4)">
    <label><span>Exclusion type</span><select name="kind">
      <option value="name"{{if eq .ExclKind "name"}} selected{{end}}>name</option>
      <option value="subtree"{{if eq .ExclKind "subtree"}} selected{{end}}>subtree</option>
      <option value="address"{{if eq .ExclKind "address"}} selected{{end}}>address</option>
    </select></label>
    <label class="scope"><span>Value</span><input class="scope" name="value" value="{{.ExclValue}}" placeholder="old-blog.acmecorp.io or 203.0.113.128/25" autocomplete="off" required></label>
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
    <p class="loss">Nothing is withdrawn. No subject leaves the estate, so no message fires &#8212; an excluded name that still resolves survives, and its Gap carries it.</p>
  </div>
  {{end}}
  {{end}}
  {{end}}
</section>

<!-- Custody --------------------------------------------------------------->
<section class="section" style="margin-bottom:0">
  <div class="cardhead">
    <div class="headings"><span class="microlabel">Custody</span><h2>Adjacent infrastructure</h2></div>
  </div>
  <p>A custody extension declares that the addresses your name scopes resolve to are yours, and so
  under your control. It is off by default and declared once per name scope &#8212; one act, never a queue
  of addresses to approve. Its coverage is recomputed from measured resolution, stopping where the
  chain leaves the declared zone, so there is no list to maintain.</p>

  {{if .CustodyError}}<div class="error">{{.CustodyError}}</div>{{end}}
  {{if .CustodyScopes}}
  {{range .CustodyScopes}}
  <div class="exclrow" style="align-items:flex-start;flex-direction:column;gap:var(--space-3)">
    <div class="switchline" style="width:100%">
      <span class="switch{{if .CustodyExtension}} on{{end}}"><span class="knob"></span></span>
      <span class="switchlabel">Extend custody to adjacent infrastructure &#8212; <span class="mono">{{.Scope}}</span></span>
      {{if .CustodyExtension}}<span class="badge" style="margin-left:auto">extension on</span>{{else}}<span class="badge off" style="margin-left:auto">off</span>{{end}}
      {{if $.IsAdmin}}
      <form method="post" action="/seeds/custody">
        <input type="hidden" name="id" value="{{.ID}}">
        <input type="hidden" name="extend" value="{{if .CustodyExtension}}false{{else}}true{{end}}">
        <button class="secondary" type="submit">{{if .CustodyExtension}}Withdraw{{else}}Declare extension{{end}}</button>
      </form>
      {{end}}
    </div>
    {{if .CustodyExtension}}
    <div class="census" style="width:100%">
      <div class="microlabel">Covered addresses &#183; census</div>
      <p>Display only. Once resolution measurement runs, this lists the addresses your names currently
      resolve into. There is no total to reach &#8212; how many addresses it ought to cover is completeness of
      your estate, which only you know &#8212; and nothing here to approve: the extension covers what it
      computes.</p>
      <div class="microlabel">No addresses measured yet</div>
    </div>
    {{end}}
  </div>
  {{end}}
  {{else}}
  <div class="emptystate">
    <div class="microlabel">No name scopes</div>
    <h2>Nothing to extend custody over</h2>
    <p style="max-width:60ch;margin:var(--space-3) auto 0">A custody extension is a property of a name scope. Declare a name scope above, then extend
    custody to the addresses it resolves into.</p>
  </div>
  {{end}}
</section>

<!-- Coverage -------------------------------------------------------------->
<section class="section" style="margin-bottom:0">
  <div class="cardhead">
    <div class="headings"><span class="microlabel">Coverage</span><h2>Coverage messages</h2></div>
    <a class="btn ghost" href="/coverage">Aperture statement</a>
  </div>
  {{if .CoverageMsgs}}
  <div>
    {{range .CoverageMsgs}}
    <div class="covmsg">
      <span class="chip stale">{{.Badge}}</span>
      <span><span class="subj">{{.Subject}}</span><br><span class="txt">{{.Text}}</span></span>
      <span class="when">{{.When}}</span>
    </div>
    {{end}}
  </div>
  {{else}}
  <div class="emptystate">
    <div class="microlabel">No coverage gaps</div>
    <h2>No coverage message right now</h2>
    <p style="max-width:60ch;margin:var(--space-3) auto 0">Every vantage is reporting and no gap, staleness or silence stands. The full aperture
    statement &#8212; what each tier looks at, its cadence, and whether it is on &#8212; lives on Coverage.</p>
    <a class="btn ghost" href="/coverage">Go to Coverage</a>
  </div>
  {{end}}
</section>

<!-- Zone file (removal detection) — staleness -> gap -------------------->
<section class="section" style="margin-bottom:0">
  <div class="cardhead">
    <div class="headings"><span class="microlabel">Removal detection</span><h2>Zone file</h2></div>
  </div>
  <p>Your own zone file is ground truth: the estate as you declare it, not as it resolves. Upload is a
  dated act &#8212; the upload instant is the observation instant. A supply older than your re-supply
  interval ages into a coverage gap, so each file below carries how long until, or since, it does.</p>

  {{if .ZoneScopes}}
  <div style="margin-bottom:var(--space-4)">
    {{range .ZoneScopes}}
    <div class="zfile">
      {{if .HasFile}}
      <span class="zname">{{.Domain}}.zone</span>
      <span class="zmeta">uploaded {{.SuppliedAt}}{{if .IntervalLabel}} &#183; re-supply {{.IntervalLabel}}{{end}}</span>
      {{if .AgingLabel}}<span class="zgap"><span class="zbadge"><span class="zdot"></span>{{.AgingLabel}}</span></span>{{end}}
      {{else}}
      <span class="zname">{{.Domain}}</span>
      <span class="zmeta">no zone file supplied</span>
      {{end}}
    </div>
    {{end}}
  </div>

  {{if .IsAdmin}}
  {{if .ZoneError}}<div class="error">{{.ZoneError}}</div>{{end}}
  <form method="post" action="/seeds/zone" enctype="multipart/form-data" class="seedform">
    <label><span>Name scope</span><select name="seed_id">
    {{range .NameScopes}}<option value="{{.ID}}">{{.Scope}}</option>{{end}}
    </select></label>
    <label class="scope"><span>Re-supply zone file</span><input class="scope" type="file" name="zonefile" accept=".zone,.txt" required></label>
    <button type="submit">Upload</button>
  </form>
  <p class="taghint">Upload is a dated act &#8212; the upload instant is the observation instant. An apex outside the chosen scope is refused, with the reason.</p>
  {{end}}
  {{else}}
  <div class="emptystate">
    <div class="microlabel">No name scopes</div>
    <h2>Nothing to supply a zone file for</h2>
    <p style="max-width:60ch;margin:var(--space-3) auto 0">A zone file is attached to a name scope. Declare a name scope above, then upload its zone file to
    detect removals against your own ground truth.</p>
  </div>
  {{end}}
</section>

<!-- Configuration: zone files, cold tier, probers ------------------------->
<div class="microlabel">Declared &#183; zone files</div>
<p>Your own zone file is stored so both services can read it, and it is evidence, not a secret.
Uploading is the supply act, so its instant is recorded then; the zone scan restates the file at that
instant, never at whatever later time the worker reads it. Re-export on your own cadence and re-supply
above &#8212; a new upload is a new supply, shipped monthly by default.</p>

{{if .IsAdmin}}

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
<td class="mono">{{if .HasFile}}{{.By}}{{else}}<span class="muted">&#8212;</span>{{end}}</td>
<td class="mono">{{if .HasFile}}{{.Bytes}} bytes{{else}}<span class="muted">&#8212;</span>{{end}}</td>
</tr>{{end}}
</tbody>
</table>
</div>
{{end}}

<div class="microlabel">Configured &#183; full-range scan (cold tier)</div>
<p>The cold scan connects to every TCP port, 1&#8211;65535, monthly. It ships <strong>disabled</strong>
with no scopes: a full-range sweep runs only where you ask for it, per scope &#8212; never at onboarding,
never on save. Opting a scope in enables the tier for that scope and it begins on its own monthly
cadence; opting the last scope out returns it to off. Only Custody-admitted addresses are ever
probed, so opting in widens what is measured, never who.</p>

{{if .ColdEnabled}}<span class="badge">tier on</span>{{else}}<span class="badge off">tier off &#8212; no scope opted in</span>{{end}}

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

<div class="microlabel">Declared &#183; probers</div>
<p>Provisioning a prober declares <em>this vantage is on the internet</em>. You supply the host, port,
and a non-root username; the instance generates the SSH keypair on the worker volume and exposes only
the public half &#8212; install it on the prober host. The private key never leaves the instance.</p>

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
<p>No vantage is on the internet yet. Provision a prober to declare one &#8212; until then, exposure cannot be measured.</p>
{{end}}
</div>

</main>
{{template "foot" .}}{{end}}

`
