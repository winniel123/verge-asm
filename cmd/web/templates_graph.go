package main

import "html/template"

// Graph screen — canonical `/graph` (#284), ported verbatim in composition from
// design-system/examples/console/GraphView.jsx + components/display/Graph.jsx: a
// pannable/zoomable SVG canvas, a minimap, zoom controls, a legend, and a node
// drawer. The graph.go handler wires real Name/Address/Service topology off the
// open-span corpus and joins the Signal engine's fired census onto the nodes (#289);
// where the estate holds nothing, the design-system empty-state shows.
//
// The example's severity affordances are re-skinned to honest signal-PRESENCE —
// signals in this product have no severity (ADR-0024/ADR-0110, as reports.go):
//   - a node with ≥1 open signal draws a --warn presence halo (class gnode-halo)
//     and a --warn minimap dot; a node with none draws neither. This is presence,
//     not a five-level ramp, so a single token carries every marked node.
//   - the drawer lists the rules that fired for the selected node (mono rule name,
//     the same slug the Signals screen renders, plus the finer subject where an
//     endpoint firing rides a Name node), or an honest "No open signals" state.
//   - the header Select is the presence axis (all / with / without), not a severity
//     scale; the filter hides whole nodes client-side off each node's data-signals
//     count.
//
// The example's inline styles are translated into the T0 pageCSS token vocabulary
// (--text-ink -> --ink, --text-secondary -> --body, --border-default -> --hairline,
// --focus-ring -> --focus, --surface-raised -> --surface, --neutral-* -> the
// hairline/strong neutrals); the presence halo/dot use the semantic --warn token. No
// second stylesheet and no design-system component is authored here (ADR-0109).
var _ = template.Must(tmpl.Parse(graphTemplates))

const graphTemplates = `
{{define "graph"}}{{template "head" .}}
{{template "chrome" .}}
<main>
<header style="display:flex;align-items:center;gap:16px;margin-bottom:20px">
<div style="display:flex;flex-direction:column;gap:2px">
<h1 style="margin:0">Graph</h1>
<span style="font:400 12.5px var(--sans);color:var(--muted)">How your assets connect. Halos mark open signals; drag to pan.</span>
</div>
{{if not .Graph.Empty}}
<div style="margin-left:auto;display:flex;gap:8px;align-items:center">
<select id="graph-filter" aria-label="Signal filter" style="width:190px">
<option value="all">All nodes</option><option value="with">With open signals</option><option value="without">No open signals</option>
</select>
<button type="button" class="secondary" id="graph-export">Export PNG</button>
</div>
{{end}}
</header>

{{if .Graph.Empty}}
<div class="emptystate">
<h2>Nothing to plot yet</h2>
<p>The graph draws from the same subjects the inventory holds. Declare a scope on
Scope and let a scan measure a subject into the estate to populate it.</p>
</div>
{{else}}
<div style="background:var(--surface);border:1px solid var(--hairline);border-radius:var(--r-lg);box-shadow:var(--shadow-sm);overflow:hidden">
<div style="position:relative" id="graph-canvas">
<svg id="graph-svg" viewBox="0 0 {{.Graph.ViewW}} {{.Graph.ViewH}}" style="display:block;width:100%;height:560px;cursor:grab;touch-action:none">
<g id="graph-viewport" transform="translate(0,0) scale(1)">
{{range .Graph.Edges}}<line x1="{{.X1}}" y1="{{.Y1}}" x2="{{.X2}}" y2="{{.Y2}}" stroke="{{if .ToService}}var(--hairline){{else}}var(--border-strong){{end}}" stroke-width="1.25"></line>
{{end}}
{{range .Graph.Nodes}}<g class="gnode" data-id="{{.ID}}" data-type="{{.Type}}" data-ports="{{.Ports}}" data-first="{{.First}}" data-signals="{{len .OpenSignals}}" transform="translate({{.X}},{{.Y}})" style="cursor:pointer">
{{if .OpenSignals}}<circle class="gnode-halo" r="{{.HaloR}}" fill="var(--warn)" opacity="0.14"></circle><circle class="gnode-halo" r="{{.HaloR}}" fill="none" stroke="var(--warn)" stroke-width="2" opacity="0.6"></circle>
{{end}}{{if eq .Type "ip"}}<rect x="-9" y="-9" width="18" height="18" rx="5" fill="var(--surface)" stroke="var(--body)" stroke-width="1.5"></rect>
{{else if eq .Type "service"}}<circle r="6" fill="var(--surface)" stroke="var(--border-strong)" stroke-width="1.5"></circle>
{{else}}<circle r="10" fill="var(--surface)" stroke="var(--border-strong)" stroke-width="1.5"></circle>
{{end}}<text x="{{.LabelDX}}" y="4" style="font:400 11px var(--mono);fill:var(--body);user-select:none">{{.Label}}</text>
</g>
{{end}}
</g>
</svg>
<div id="graph-minimap" style="position:absolute;left:14px;bottom:14px;background:var(--surface);border:1px solid var(--hairline);border-radius:10px;box-shadow:var(--shadow-md);padding:6px;line-height:0">
<svg width="{{.Graph.MiniW}}" height="{{.Graph.MiniH}}">
{{range .Graph.Nodes}}<circle cx="{{.Mx}}" cy="{{.My}}" r="1.5" fill="{{if .OpenSignals}}var(--warn){{else}}var(--muted){{end}}"></circle>{{end}}
<rect id="graph-mini-rect" data-mw="{{.Graph.MiniW}}" data-mh="{{.Graph.MiniH}}" x="0" y="0" width="{{.Graph.MiniW}}" height="{{.Graph.MiniH}}" rx="3" fill="none" stroke="var(--accent)" stroke-width="1.5"></rect>
</svg>
</div>
<div id="graph-controls" style="position:absolute;right:14px;bottom:14px;display:flex;flex-direction:column;background:var(--surface);border:1px solid var(--hairline);border-radius:12px;box-shadow:var(--shadow-md);padding:3px">
<button type="button" data-graph-zoom="in" aria-label="Zoom in" style="display:flex;align-items:center;justify-content:center;width:30px;height:30px;border:none;background:transparent;color:var(--body);cursor:pointer;font:500 15px var(--mono);border-radius:8px">+</button>
<button type="button" data-graph-zoom="out" aria-label="Zoom out" style="display:flex;align-items:center;justify-content:center;width:30px;height:30px;border:none;background:transparent;color:var(--body);cursor:pointer;font:500 15px var(--mono);border-radius:8px">&#8722;</button>
<button type="button" data-graph-zoom="reset" aria-label="Reset view" style="display:flex;align-items:center;justify-content:center;width:30px;height:30px;border:none;background:transparent;color:var(--body);cursor:pointer;font:500 15px var(--mono);border-radius:8px">&#10530;</button>
</div>
</div>
<div style="padding:12px 20px;border-top:1px solid var(--hairline);display:flex;align-items:center;gap:20px;flex-wrap:wrap">
<span style="display:inline-flex;align-items:center;gap:7px"><svg width="16" height="16" viewBox="0 0 16 16"><circle cx="8" cy="8" r="5.5" fill="var(--surface)" stroke="var(--border-strong)" stroke-width="1.5"></circle></svg><span style="font:400 11px var(--mono);color:var(--body)">name</span></span>
<span style="display:inline-flex;align-items:center;gap:7px"><svg width="16" height="16" viewBox="0 0 16 16"><rect x="2.5" y="2.5" width="11" height="11" rx="3.5" fill="var(--surface)" stroke="var(--body)" stroke-width="1.5"></rect></svg><span style="font:400 11px var(--mono);color:var(--body)">address</span></span>
<span style="display:inline-flex;align-items:center;gap:7px"><svg width="16" height="16" viewBox="0 0 16 16"><circle cx="8" cy="8" r="4" fill="var(--surface)" stroke="var(--border-strong)" stroke-width="1.5"></circle></svg><span style="font:400 11px var(--mono);color:var(--body)">service</span></span>
<span style="display:inline-flex;align-items:center;gap:7px"><svg width="18" height="18" viewBox="0 0 18 18"><circle cx="9" cy="9" r="7.5" fill="var(--warn)" opacity="0.14"></circle><circle cx="9" cy="9" r="6" fill="none" stroke="var(--warn)" stroke-width="2" opacity="0.6"></circle><circle cx="9" cy="9" r="3" fill="var(--surface)" stroke="var(--border-strong)" stroke-width="1.5"></circle></svg><span style="font:400 11px var(--mono);color:var(--body)">open signal</span></span>
</div>
</div>

<div id="graph-drawer" hidden>
<div class="scrim" data-graph-close></div>
<aside class="drawer-panel" role="dialog" aria-label="Node detail" aria-modal="true" style="width:420px">
<div style="display:flex;flex-direction:column;gap:16px">
<div>
<div id="gd-title" style="font:600 16px var(--sans);color:var(--ink);letter-spacing:-0.015em;word-break:break-all"></div>
<div id="gd-sub" class="microlabel" style="margin-top:4px"></div>
</div>
<dl style="display:flex;flex-direction:column;gap:10px;margin:0">
<div style="display:flex;gap:12px"><dt style="width:96px;color:var(--muted)">Node</dt><dd id="gd-node" class="mono" style="margin:0;color:var(--ink);word-break:break-all"></dd></div>
<div style="display:flex;gap:12px"><dt style="width:96px;color:var(--muted)">Type</dt><dd id="gd-type" style="margin:0;color:var(--ink)"></dd></div>
<div style="display:flex;gap:12px"><dt style="width:96px;color:var(--muted)">Open ports</dt><dd id="gd-ports" class="mono" style="margin:0;color:var(--ink)"></dd></div>
<div style="display:flex;gap:12px"><dt style="width:96px;color:var(--muted)">First seen</dt><dd id="gd-first" class="mono" style="margin:0;color:var(--ink)"></dd></div>
</dl>
<div>
<div class="microlabel" style="margin-bottom:10px">Open signals</div>
<div id="gd-signals" style="display:flex;flex-direction:column;gap:12px" hidden></div>
<span id="gd-signals-empty" style="font:400 12.5px var(--sans);color:var(--muted)">No open signals on this node.</span>
</div>
</div>
<div class="drawer-actions"><button type="button" class="secondary" data-graph-close>Close</button></div>
</aside>
</div>

<div id="graph-signal-data" hidden>
{{range $n := .Graph.Nodes}}<div data-for="{{$n.ID}}">{{range $n.OpenSignals}}<div style="display:flex;flex-direction:column;gap:2px">
<span style="font:500 12.5px var(--mono);color:var(--ink);word-break:break-all">{{.Rule}}</span>
{{if ne .Subject $n.ID}}<span style="font:400 11px var(--mono);color:var(--muted);word-break:break-all">{{.Subject}}</span>{{end}}
</div>{{end}}</div>{{end}}
</div>
{{end}}
</main>

{{if not .Graph.Empty}}
<script>
/* Graph canvas behaviour, translated from components/display/Graph.jsx: pan by
   dragging the background, zoom via wheel or the controls, a live minimap
   viewport, and a node drawer. All screen-local — the shell (foot) owns none of
   this. */
(function () {
  var svg = document.getElementById("graph-svg");
  if (!svg) return;
  var vp = document.getElementById("graph-viewport");
  var VW = {{.Graph.ViewW}}, VH = {{.Graph.ViewH}};
  var view = { x: 0, y: 0, k: 1 };
  var drag = null, movedRecently = false;
  function clamp(v, lo, hi) { return Math.max(lo, Math.min(hi, v)); }
  function apply() {
    vp.setAttribute("transform", "translate(" + view.x + "," + view.y + ") scale(" + view.k + ")");
    updateMini();
  }
  function zoomAt(f, cx, cy) {
    var k = clamp(view.k * f, 0.5, 2.5);
    view.x = cx - (cx - view.x) * (k / view.k);
    view.y = cy - (cy - view.y) * (k / view.k);
    view.k = k; apply();
  }
  function zoom(f) { zoomAt(f, VW / 2, VH / 2); }
  svg.addEventListener("wheel", function (e) {
    e.preventDefault();
    var r = svg.getBoundingClientRect();
    var s = VW / (r.width || VW);
    zoomAt(e.deltaY < 0 ? 1.15 : 0.87, (e.clientX - r.left) * s, (e.clientY - r.top) * s);
  }, { passive: false });
  svg.addEventListener("pointerdown", function (e) {
    drag = { sx: e.clientX, sy: e.clientY, ox: view.x, oy: view.y, moved: false, pid: e.pointerId };
  });
  svg.addEventListener("pointermove", function (e) {
    if (!drag) return;
    var dx = e.clientX - drag.sx, dy = e.clientY - drag.sy;
    if (!drag.moved && Math.abs(dx) + Math.abs(dy) > 3) {
      drag.moved = true;
      try { svg.setPointerCapture(drag.pid); } catch (err) {}
      svg.style.cursor = "grabbing";
    }
    if (!drag.moved) return;
    var r = svg.getBoundingClientRect();
    var s = VW / (r.width || VW);
    view.x = drag.ox + dx * s; view.y = drag.oy + dy * s; apply();
  });
  function endDrag() {
    if (drag && drag.moved) { movedRecently = true; setTimeout(function () { movedRecently = false; }, 0); }
    svg.style.cursor = "grab"; drag = null;
  }
  svg.addEventListener("pointerup", endDrag);
  svg.addEventListener("pointerleave", endDrag);

  var zoomBtns = document.querySelectorAll("[data-graph-zoom]");
  for (var i = 0; i < zoomBtns.length; i++) {
    (function (b) {
      b.addEventListener("click", function () {
        var m = b.getAttribute("data-graph-zoom");
        if (m === "in") zoom(1.25);
        else if (m === "out") zoom(0.8);
        else { view = { x: 0, y: 0, k: 1 }; apply(); }
      });
    })(zoomBtns[i]);
  }

  var mrect = document.getElementById("graph-mini-rect");
  function updateMini() {
    if (!mrect) return;
    var mw = +mrect.getAttribute("data-mw"), mh = +mrect.getAttribute("data-mh");
    var s = mw / VW;
    var vx = clamp((-view.x / view.k) * s, 0, mw), vy = clamp((-view.y / view.k) * s, 0, mh);
    var vw = clamp((VW / view.k) * s, 8, mw - vx), vh = clamp((VH / view.k) * s, 8, mh - vy);
    mrect.setAttribute("x", vx); mrect.setAttribute("y", vy);
    mrect.setAttribute("width", vw); mrect.setAttribute("height", vh);
  }

  var drawer = document.getElementById("graph-drawer");
  function txt(id, v) { var el = document.getElementById(id); if (el) el.textContent = v; }
  function openDrawer(g) {
    var id = g.getAttribute("data-id"), type = g.getAttribute("data-type");
    txt("gd-title", id); txt("gd-sub", type); txt("gd-node", id); txt("gd-type", type);
    txt("gd-ports", g.getAttribute("data-ports") || "—");
    txt("gd-first", g.getAttribute("data-first") || "—");
    /* The fired-rule list for this node is pre-rendered server-side in
       #graph-signal-data (a subject key holds no double-quote, so it is a safe
       attribute-selector value); clone it in, or fall back to the honest empty
       state. Presence, never severity. */
    var box = document.getElementById("gd-signals");
    var empty = document.getElementById("gd-signals-empty");
    var src = document.querySelector('#graph-signal-data [data-for="' + id + '"]');
    if (box && src && src.children.length) {
      box.innerHTML = src.innerHTML;
      box.removeAttribute("hidden");
      if (empty) empty.setAttribute("hidden", "");
    } else {
      if (box) { box.innerHTML = ""; box.setAttribute("hidden", ""); }
      if (empty) empty.removeAttribute("hidden");
    }
    drawer.removeAttribute("hidden");
  }
  function closeDrawer() { if (drawer) drawer.setAttribute("hidden", ""); }
  var gnodes = document.querySelectorAll(".gnode");
  for (var j = 0; j < gnodes.length; j++) {
    (function (g) {
      g.addEventListener("click", function (e) {
        e.stopPropagation();
        if (movedRecently) return;
        openDrawer(g);
      });
    })(gnodes[j]);
  }
  var closers = document.querySelectorAll("[data-graph-close]");
  for (var c = 0; c < closers.length; c++) closers[c].addEventListener("click", closeDrawer);
  document.addEventListener("keydown", function (e) { if (e.key === "Escape") closeDrawer(); });

  /* The presence filter — the honest axis (signals carry no severity). It hides
     whole nodes off each node's open-signal count, never dims a fabricated level. */
  var filter = document.getElementById("graph-filter");
  if (filter) filter.addEventListener("change", function () {
    var mode = filter.value;
    var ns = document.querySelectorAll(".gnode");
    for (var k = 0; k < ns.length; k++) {
      var has = (+ns[k].getAttribute("data-signals")) > 0;
      var show = mode === "all" || (mode === "with" && has) || (mode === "without" && !has);
      ns[k].style.display = show ? "" : "none";
    }
  });

  var exp = document.getElementById("graph-export");
  if (exp) exp.addEventListener("click", function () {
    try {
      var xml = new XMLSerializer().serializeToString(svg);
      var img = new Image();
      img.onload = function () {
        var cv = document.createElement("canvas");
        cv.width = VW; cv.height = VH;
        var ctx = cv.getContext("2d");
        ctx.fillStyle = getComputedStyle(document.body).backgroundColor || getComputedStyle(document.documentElement).getPropertyValue("--paper").trim();
        ctx.fillRect(0, 0, VW, VH);
        ctx.drawImage(img, 0, 0, VW, VH);
        var a = document.createElement("a");
        a.download = "graph.png"; a.href = cv.toDataURL("image/png"); a.click();
      };
      img.src = "data:image/svg+xml;base64," + btoa(unescape(encodeURIComponent(xml)));
    } catch (err) {}
  });

  apply();
})();
</script>
{{end}}
{{template "foot" .}}{{end}}
`
