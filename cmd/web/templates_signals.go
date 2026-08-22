package main

import "html/template"

// Signals screen — canonical `/signals`. The screen ticket (T-Signals) rewrites
// the body against examples/console/Signals.jsx (sortable table, tabs
// Open/Annotated/Withdrawn, detail drawer with annotation control, context menus,
// typed descope confirm). Ported verbatim for T0.
var _ = template.Must(tmpl.Parse(signalsTemplates))

const signalsTemplates = `
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

`
