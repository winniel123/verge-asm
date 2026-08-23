package main

import "html/template"

// Error pages — 404 / 403 / 500, ported from examples/console/ErrorPage.jsx
// (T11, #306, ADR-0110). The three states share one full-screen "terminal" frame:
// an icon badge, the big mono status number, a title and body, an optional copyable
// incident id (500 only), and the way back to the Dashboard. This is a chrome-less
// frame exactly as the example composes it — no topnav — so it renders identically
// for an unauthenticated 404 and a signed-in 403.
//
// The KINDS copy is ported verbatim from ErrorPage.jsx. The Lucide icons the example
// names (compass · lock · server-crash) are inlined as SVG here, the same way the
// shell inlines its chrome icons — this is a template-local translation within the
// existing token vocabulary, not a design-system component authored in-repo
// (ADR-0109). Token names are the repo's inlined set (--sunken/--hairline/--muted/
// --mono/--sans/--ink), so both light and dark themes flip with the rest of the page.
//
// The 500's incident id copy control ports CopyValue.jsx: a mono value with a copy
// button that swaps to a check for ~1.4s, clipboard API with a legacy execCommand
// fallback. The id is minted and logged by the recovery middleware (errors.go),
// never fabricated.
var _ = template.Must(tmpl.Parse(errorTemplates))

const errorTemplates = `
{{define "error-page"}}{{template "head" .}}
<main data-screen-label="Error {{.Kind}}" style="min-height:70vh;display:flex;flex-direction:column;align-items:center;justify-content:center;gap:18px;padding:32px;text-align:center">
<span style="display:inline-flex;align-items:center;justify-content:center;width:52px;height:52px;border-radius:var(--r-lg);background:var(--sunken);border:1px solid var(--hairline);color:var(--muted)">
{{if eq .Kind "403"}}<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round"><rect width="18" height="11" x="3" y="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
{{else if eq .Kind "500"}}<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round"><path d="M6 10H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2v4a2 2 0 0 1-2 2h-2"/><path d="M6 14H4a2 2 0 0 0-2 2v4a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-4a2 2 0 0 0-2-2h-2"/><path d="M6 6h.01"/><path d="M6 18h.01"/><path d="m13 6-4 6h6l-4 6"/></svg>
{{else}}<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><polygon points="16.24 7.76 14.12 14.12 7.76 16.24 9.88 9.88 16.24 7.76"/></svg>{{end}}
</span>
<span style="font:600 32px var(--mono);letter-spacing:0.04em;color:var(--muted)">{{.Kind}}</span>
<div style="display:flex;flex-direction:column;gap:8px;align-items:center">
{{if eq .Kind "403"}}
<h1 style="margin:0;font:600 18px var(--sans);letter-spacing:-0.015em;color:var(--ink)">Access denied</h1>
<p style="margin:0;font:400 13px/1.6 var(--sans);color:var(--muted);max-width:400px">Your role can't view this. An admin can widen it in Settings &#8594; Team.</p>
{{else if eq .Kind "500"}}
<h1 style="margin:0;font:600 18px var(--sans);letter-spacing:-0.015em;color:var(--ink)">Something broke</h1>
<p style="margin:0;font:400 13px/1.6 var(--sans);color:var(--muted);max-width:400px">The console hit an unexpected error. The incident is logged on the host with the ID below.</p>
{{else}}
<h1 style="margin:0;font:600 18px var(--sans);letter-spacing:-0.015em;color:var(--ink)">Page not found</h1>
<p style="margin:0;font:400 13px/1.6 var(--sans);color:var(--muted);max-width:400px">The address doesn't match any screen in this deployment. The link may predate a rename.</p>
{{end}}
</div>
{{if .IncidentID}}
<span class="incident" style="display:inline-flex;align-items:center;gap:6px;min-width:0">
<span class="mono" style="font-size:12.5px;color:var(--body);overflow-wrap:anywhere">{{.IncidentID}}</span>
<button type="button" data-incident-copy="{{.IncidentID}}" aria-label="Copy value" style="display:inline-flex;align-items:center;justify-content:center;width:18px;height:18px;border:none;border-radius:6px;background:transparent;color:var(--muted);cursor:pointer;padding:0;flex:none">
<svg data-i-copy width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round"><rect width="14" height="14" x="8" y="8" rx="2" ry="2"/><path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2"/></svg>
<svg data-i-ok width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" hidden><path d="M20 6 9 17l-5-5"/></svg>
</button>
</span>
<script>
(function(){
  var b=document.querySelector("[data-incident-copy]");
  if(!b)return;
  var copyIcon=b.querySelector("[data-i-copy]"),okIcon=b.querySelector("[data-i-ok]");
  function done(ok){
    if(!ok)return;
    b.style.color="var(--ok)";
    if(copyIcon)copyIcon.hidden=true; if(okIcon)okIcon.hidden=false;
    setTimeout(function(){ b.style.color="var(--muted)"; if(copyIcon)copyIcon.hidden=false; if(okIcon)okIcon.hidden=true; },1400);
  }
  b.addEventListener("click",function(){
    var v=b.getAttribute("data-incident-copy");
    function legacy(){
      try{
        var ta=document.createElement("textarea");
        ta.value=v; ta.setAttribute("readonly",""); ta.style.position="fixed"; ta.style.opacity="0";
        document.body.appendChild(ta); ta.select();
        var ok=document.execCommand("copy");
        document.body.removeChild(ta); done(ok);
      }catch(e){}
    }
    if(navigator.clipboard&&window.isSecureContext) navigator.clipboard.writeText(v).then(function(){done(true);},legacy);
    else legacy();
  });
})();
</script>
{{end}}
<a class="btn" href="/">Back to dashboard</a>
</main>
{{template "foot" .}}{{end}}
`
