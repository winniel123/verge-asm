package main

// PROTOTYPE — throwaway. Answers #1790 for map #1786. Not production code.
//
// The class definitions below are lifted verbatim from settings.tmpl:52-56 and
// the token values from design-system/tokens/, so what renders here is what the
// shipped Audit tab renders.

const pageHTML = `<!doctype html>
<html lang="en" id="root">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Act corpus — the closed union in four columns</title>
<style>
@import url("https://fonts.googleapis.com/css2?family=Instrument+Sans:ital,wght@0,400..700;1,400..700&family=Geist+Mono:wght@400..700&display=swap");
:root {
  --font-ui: "Instrument Sans", "Helvetica Neue", Arial, sans-serif;
  --font-mono: "Geist Mono", "IBM Plex Mono", "SFMono-Regular", Consolas, monospace;
  --heading-tracking: -0.015em;
  --page: #f9f7f5; --surface: #ffffff; --surface-sunken: #f2f0ec;
  --text-ink: #231f19; --text-body: #37322c; --text-secondary: #67625c; --text-muted: #79746d;
  --border-default: #e2dfdb; --border-strong: #c7c3be; --row-sep: #efece9;
  --accent: #037ac0; --accent-hover: #006bad; --on-accent: #ffffff; --accent-soft: #e1f6ff;
  --link: #006eaf; --row-selected: #e1f6ff;
  --ok: #05773b; --ok-soft: #e1fae7; --ok-border: #bfebc9;
  --warn: #8d5500; --warn-soft: #fff0d8; --warn-border: #f8d9af;
  --danger: #ac312c; --danger-soft: #ffe8e2; --danger-border: #ffcac2;
  --shadow-sm: 0 1px 2px rgba(35, 31, 25, 0.06), 0 2px 8px rgba(35, 31, 25, 0.04);
  --dur-fast: 120ms; --ease-out: cubic-bezier(0.2, 0, 0, 1);
}
html[data-theme="dark"] {
  --page: #15120f; --surface: #1e1b17; --surface-sunken: #191613;
  --text-ink: #eae7e4; --text-body: #d9d5d1; --text-secondary: #b1ada8; --text-muted: #898581;
  --border-default: #383530; --border-strong: #514c46; --row-sep: #292622;
  --accent: #6bbeff; --accent-hover: #9ddaff; --on-accent: #063352; --accent-soft: #272f33;
  --link: #59acee; --row-selected: #272f33;
  --ok: #57c07f; --ok-soft: #17281c; --ok-border: #1f4029;
  --warn: #e0aa4a; --warn-soft: #2d2413; --warn-border: #4a3a12;
  --danger: #f08c82; --danger-soft: #331b17; --danger-border: #55231d;
  --shadow-sm: 0 1px 2px rgba(0, 0, 0, 0.35), 0 2px 8px rgba(0, 0, 0, 0.25);
}
* { box-sizing: border-box; }
body { margin: 0; background: var(--page); color: var(--text-body); font-family: var(--font-ui); font-size: 13px; line-height: 1.5; }
.wrap { max-width: 1440px; margin: 0 auto; padding: 32px; display: flex; flex-direction: column; gap: 24px; }
.st-micro { font: 500 11px var(--font-mono); letter-spacing: 0.07em; text-transform: uppercase; color: var(--text-muted); }
.st-card { background: var(--surface); border: 1px solid var(--border-default); border-radius: 16px; box-shadow: var(--shadow-sm); display: flex; flex-direction: column; }
.st-card.flush { overflow: hidden; }
.st-body { padding: 20px; }
.st-lede { margin: 0; font: 400 13px/1.6 var(--font-ui); color: var(--text-secondary); max-width: 72ch; }
.st-note { font: 400 11.5px/1.6 var(--font-ui); color: var(--text-muted); }
.st-table { width: 100%; border-collapse: separate; border-spacing: 0; }
.st-table th { padding: 8px 16px; font: 500 11px var(--font-mono); letter-spacing: 0.07em; text-transform: uppercase; color: var(--text-muted); border-bottom: 1px solid var(--border-default); white-space: nowrap; background: var(--surface); text-align: left; }
.st-table td { padding: 9px 16px; font: 400 13px var(--font-ui); color: var(--text-body); vertical-align: middle; white-space: nowrap; border-top: 1px solid var(--row-sep); }
.st-table tbody tr:first-child td { border-top: none; }
.st-table td.mono { font: 400 12.5px var(--font-mono); }
.st-table tbody tr { cursor: pointer; transition: background var(--dur-fast) var(--ease-out); }
.st-table tbody tr:hover { background: var(--surface-sunken); }
.st-table tbody tr.on { background: var(--row-selected); box-shadow: inset 3px 0 0 var(--accent); }
.st-btn { display: inline-flex; align-items: center; justify-content: center; gap: 8px; height: 36px; padding: 0 16px; border-radius: 12px; font: 600 13px var(--font-ui); cursor: pointer; white-space: nowrap; border: 1px solid var(--border-strong); background: var(--surface); color: var(--text-body); transition: background var(--dur-fast) var(--ease-out); }
.st-btn:hover { background: var(--surface-sunken); }
.st-btn.on { background: var(--accent); color: var(--on-accent); border-color: transparent; }
.st-badge { display: inline-flex; align-items: center; gap: 6px; height: 22px; padding: 0 10px; border-radius: 999px; border: 1px solid transparent; font-family: var(--font-ui); font-size: 12px; font-weight: 500; white-space: nowrap; }
.st-badge.ok { background: var(--ok-soft); border-color: var(--ok-border); color: var(--ok); }
.st-badge.warn { background: var(--warn-soft); border-color: var(--warn-border); color: var(--warn); }
.st-badge.danger { background: var(--danger-soft); border-color: var(--danger-border); color: var(--danger); }
.st-badge.neutral { background: var(--surface-sunken); border-color: var(--border-default); color: var(--text-secondary); }
h1 { margin: 0; font: 600 21px var(--font-ui); letter-spacing: var(--heading-tracking); color: var(--text-ink); }
h2 { margin: 0; font: 600 17px var(--font-ui); letter-spacing: var(--heading-tracking); color: var(--text-ink); }
.bar { display: flex; gap: 8px; flex-wrap: wrap; align-items: center; }
.head { display: flex; flex-direction: column; gap: 6px; }
pre { margin: 0; font: 400 12px/1.6 var(--font-mono); color: var(--text-body); background: var(--surface-sunken); border: 1px solid var(--border-default); border-radius: 8px; padding: 12px 14px; overflow-x: auto; }
.grid { display: grid; grid-template-columns: minmax(0,1fr) minmax(0,1fr); gap: 16px; }
@media (max-width: 900px) { .grid { grid-template-columns: minmax(0,1fr); } .wrap { padding: 16px; } }
.tab { display: none; flex-direction: column; gap: 24px; }
.tab.on { display: flex; }
.scroll { overflow-x: auto; }
</style>
</head>
<body>
<div class="wrap">

<div class="head">
<span class="st-micro">Prototype · issue 1790 · map 1786</span>
<h1>The Act closed union, rendered into the four shipped columns</h1>
<p class="st-lede">Every row below is produced by the Go union in this directory: encoded to the stored row shape, decoded back, and rendered by the variant. Click a row to see what Postgres actually holds. Nothing here reads a store, which is the point of hazard 1.</p>
</div>

<div class="bar">
<button class="st-btn on" data-tab="all">All {{len .Views}} variants</button>
<button class="st-btn" data-tab="h1">Hazard 1 · a withdrawn subject</button>
<button class="st-btn" data-tab="h2">Hazard 2 · a secret being set</button>
<button class="st-btn" data-tab="h3">Hazard 3 · a restore</button>
<button class="st-btn" data-tab="h4">The Actor collision</button>
<span style="flex:1"></span>
<button class="st-btn" id="stress">Stress the widths</button>
<button class="st-btn" id="theme">Dark</button>
</div>

{{define "tbl"}}
<section class="st-card flush">
<div class="scroll">
<table class="st-table">
<thead><tr><th style="width:160px">When</th><th style="width:170px">Actor</th><th style="width:170px">Action</th><th>Subject</th></tr></thead>
<tbody>
{{range .}}<tr data-class="{{.Class}}" data-kind="{{.ActorKind}}" data-actor="{{.ActorJSON}}" data-subject="{{.SubjectJSON}}" data-route="{{.Route}}" data-limb="{{.Limb}}" data-note="{{.Note}}">
<td class="mono">{{.When}}</td><td class="mono">{{.Actor}}</td><td class="mono">{{.Action}}</td><td class="mono">{{.Subject}}</td>
</tr>{{end}}
</tbody>
</table>
</div>
</section>
{{end}}

<div class="tab on" id="tab-all">
{{template "tbl" .Views}}
<section class="st-card"><div class="st-body">
<span class="st-micro">What the count means</span>
<p class="st-lede" style="margin-top:8px">60 of these are issue 1791's corrected class count. One more is the split of POST /coverage/retention into two rows, because that submit moves two dials and a list-valued subject is barred. The last is migration.applied, which is conditional on issue 1805.</p>
<p class="st-note" style="margin-top:8px">Widest rendered cell: When {{index .Widest "When"}} · Actor {{index .Widest "Actor"}} · Action {{index .Widest "Action"}} · Subject {{index .Widest "Subject"}} characters. The four shipped column widths are 160px, 170px, 170px and the remainder.</p>
</div></section>
</div>

<div class="tab" id="tab-h1">
<div class="head"><h2>Hazard 1 · a withdrawn subject</h2>
<p class="st-lede">Each row names a thing that no longer exists: a Seed withdrawn, an account removed, an SSO binding deleted, a dispatch long retired. The Subject cell is built from values captured at write time, so nothing renders blank. A live join would render blank on exactly the acts most worth reading.</p></div>
{{template "tbl" hazard1 .Views}}
</div>

<div class="tab" id="tab-h2">
<div class="head"><h2>Hazard 2 · a secret being set</h2>
<p class="st-lede">ADR-0053 splits a secret being set, which is auditable, from its value, which is not. The variant for POST /settings/sso/secret has one field and it is the provider slug. There is no field the secret could go in, and the hazard test rejects any field that is not a string or an int64, so a bytea or a map cannot appear either.</p>
<p class="st-note">Honest limit: this bars a field for the value. It cannot stop a caller putting a secret into a field named slug. That is the same strength issue 1791 reported for the AST gate — the violation fails a test, it is not inexpressible.</p></div>
{{template "tbl" hazard2 .Views}}
</div>

<div class="tab" id="tab-h3">
<div class="head"><h2>Hazard 3 · a restore</h2>
<p class="st-lede">applyRestore truncates every backup table, this corpus included, so the discontinuity row is written after the apply returns and is the whole history at that instant. Its actor is Account, ruled by issue 1795, and the admin's username survives the TRUNCATE because Settled 8 captures a value rather than a join.</p>
<p class="st-note">migration.applied sits here because a restore replays a schema that then runs goose.Up. If a migration writes an Act, a restore may write System rows underneath the admin's own row. Issue 1805 rules it.</p></div>
{{template "tbl" hazard3 .Views}}
</div>

<div class="tab" id="tab-h4">
<div class="head"><h2>The Actor collision</h2>
<p class="st-lede">account.username is TEXT NOT NULL UNIQUE with no format check and no reserved list. validateCredentials caps the length at 64 and requires it to be non-empty, and that is all. So an operator can create an account whose username renders exactly as a non-account Actor.</p></div>
<section class="st-card flush"><div class="scroll">
<table class="st-table">
<thead><tr><th style="width:170px">Actor cell</th><th style="width:220px">What it may be</th><th>And also</th></tr></thead>
<tbody>
<tr><td class="mono">system</td><td class="mono">System</td><td class="mono">an account named system</td></tr>
<tr><td class="mono">setup token</td><td class="mono">GrantHolder(SetupToken)</td><td class="mono">an account named setup token</td></tr>
<tr><td class="mono">password-reset link</td><td class="mono">GrantHolder(PasswordReset)</td><td class="mono">an account named password-reset link</td></tr>
<tr><td class="mono">invite 12</td><td class="mono">GrantHolder(Invite 12)</td><td class="mono">an account named invite 12</td></tr>
</tbody>
</table>
</div></section>
<section class="st-card"><div class="st-body">
<span class="st-micro">Why it matters here and not on the Sessions tab</span>
<p class="st-lede" style="margin-top:8px">An Act is never deleted and it is the record an operator reads to answer who did this. A row that reads system when a person did it is an unretractable false attribution, which is the same class of harm that barred the recorder-before ordering in issue 1788.</p>
<p class="st-lede" style="margin-top:8px">The stored row is not ambiguous: actor_kind holds account, grant_holder or system. Only the rendering collides. So the fix is rendering-side and cheap, and three shapes are open — a reserved-username refusal at create, a non-account Actor rendered as a badge rather than as mono text, or the account variant rendered with a marker no username can hold.</p>
</div></section>
</div>

<section class="st-card"><div class="st-body">
<span class="st-micro">The stored row</span>
<div class="grid" style="margin-top:12px">
<div><p class="st-note" style="margin-bottom:6px">Selected row</p><pre id="pick">Click any row above.</pre></div>
<div><p class="st-note" style="margin-bottom:6px">Schema</p><pre>CREATE TABLE act (
    id          BIGSERIAL PRIMARY KEY,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    actor_kind  TEXT NOT NULL
                CHECK (actor_kind IN ('account','grant_holder','system')),
    actor       JSONB NOT NULL,
    action      TEXT NOT NULL,
    subject     JSONB NOT NULL
);
-- No FK to account: Settled 8.
-- No CHECK on action: 62 tokens would need a migration per act class.</pre></div>
</div>
</div></section>

</div>
<script>
var tabs = document.querySelectorAll("[data-tab]");
tabs.forEach(function (b) {
  b.addEventListener("click", function () {
    tabs.forEach(function (o) { o.classList.remove("on"); });
    b.classList.add("on");
    document.querySelectorAll(".tab").forEach(function (t) { t.classList.remove("on"); });
    document.getElementById("tab-" + b.dataset.tab).classList.add("on");
  });
});

function wire() {
  document.querySelectorAll(".st-table tbody tr[data-class]").forEach(function (tr) {
    tr.addEventListener("click", function () {
      document.querySelectorAll("tr.on").forEach(function (o) { o.classList.remove("on"); });
      tr.classList.add("on");
      document.getElementById("pick").textContent =
        "route       " + tr.dataset.route + "\n" +
        "limb        " + tr.dataset.limb + "\n" +
        "action      " + tr.dataset.class + "\n" +
        "actor_kind  " + tr.dataset.kind + "\n" +
        "actor       " + tr.dataset.actor + "\n" +
        "subject     " + tr.dataset.subject +
        (tr.dataset.note ? "\n\n" + tr.dataset.note : "");
    });
  });
}
wire();

var stressed = false;
document.getElementById("stress").addEventListener("click", function () {
  stressed = !stressed;
  this.classList.toggle("on", stressed);
  document.querySelectorAll(".st-table tbody tr[data-class] td.mono:last-child").forEach(function (td) {
    if (stressed) {
      td.dataset.was = td.textContent;
      td.textContent = td.textContent + " / a-very-long-declared-name.subdomain.example.com:8443";
    } else if (td.dataset.was) {
      td.textContent = td.dataset.was;
    }
  });
});

document.getElementById("theme").addEventListener("click", function () {
  var dark = document.getElementById("root").getAttribute("data-theme") === "dark";
  document.getElementById("root").setAttribute("data-theme", dark ? "light" : "dark");
  this.textContent = dark ? "Dark" : "Light";
});
</script>
</body>
</html>
`
