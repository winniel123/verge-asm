package main

import (
	"html/template"
	"strconv"
)

// pageCSS is a self-contained slice of the Verge ASM design system: the tokens
// and the shared component rules the console screens need. The web binary is
// distroless with no mounted asset path, so the stylesheet is inlined rather
// than linked. Values are the system's own — the V2 "calm instrument" redesign:
// warm stone surfaces, warm-stone ink, one confident action azure, Instrument
// Sans for UI/prose and Geist Mono for every technical value, a real radius
// scale, soft layered elevation (no hard offsets, no structural ink rules), the
// five-level severity ramp (Critical the only solid fill and only pill-red), the
// drift palette (violet gain / magenta change / slate loss), and the mono
// micro-label eyebrow that survives as the system's signature.
//
// Dark mode ships two ways: prefers-color-scheme is the default, and an explicit
// data-theme toggle (set on <html>) overrides it in either direction. The token
// custom-property names are stable — other inlined stylesheets (e.g.
// cmd/web/sources.go) read them — so a re-skin changes VALUES, not names. Kept in
// step with design-system/tokens. These classes are TEMPLATE-LOCAL CSS translated
// from design-system/components/* within the existing token vocabulary; this is
// restyling, not authoring (ADR-0109). No design-system component is authored here.
const pageCSS = `
@import url("https://fonts.googleapis.com/css2?family=Instrument+Sans:ital,wght@0,400..700;1,400..700&family=Geist+Mono:wght@400..700&display=swap");
:root {
  /* surfaces (warm stone) */
  --paper: #f9f7f5; --surface: #ffffff; --sunken: #f2f0ec;
  --inverted: #231f19; --on-inverted: #f4f2ef;
  /* text */
  --ink: #231f19; --body: #37322c; --muted: #79746d;
  /* borders (hairlines do quiet work; ink rules are gone) */
  --hairline: #e2dfdb; --border-strong: #c7c3be;
  /* action azure */
  --accent: #037ac0; --accent-hover: #006bad; --on-accent: #ffffff;
  --accent-soft: #e1f6ff; --link: #006eaf; --focus: #037ac0;
  /* semantic status (text / soft tint / border) */
  --danger: #ac312c; --danger-soft: #ffe8e2; --danger-border: #ffcac2; --on-danger: #ffffff;
  --ok: #05773b; --ok-soft: #e1fae7; --ok-border: #bfebc9;
  --warn: #8d5500; --warn-soft: #fff0d8; --warn-border: #f8d9af;
  /* severity ramp — Critical is the only solid fill and only pill-red */
  --sev-critical-fill: #bf3631; --sev-critical-text: #ffffff; --sev-critical-dot: #bf3631;
  --sev-high-bg: #ffe9d6;  --sev-high-border: #ffcdae;  --sev-high-fg: #a04400;  --sev-high-dot: #e26c00;
  --sev-medium-bg: #ffeecc; --sev-medium-border: #f4d59d; --sev-medium-fg: #8d5600; --sev-medium-dot: #e0a200;
  --sev-low-bg: #d7f7ff;   --sev-low-border: #afe3f0;   --sev-low-fg: #00728b;   --sev-low-dot: #009aba;
  --sev-info-bg: #ebf2f9;  --sev-info-border: #d0dae6;  --sev-info-fg: #536579;  --sev-info-dot: #798898;
  /* drift (change vocabulary) — its own language, never the severity ramp */
  --drift-gain-bg: #f5ebff; --drift-gain-border: #e1d1ff; --drift-gain-fg: #6f4fa1;
  --drift-change-bg: #ffe6f7; --drift-change-border: #fec9e5; --drift-change-fg: #954074;
  --drift-loss-bg: #ecf1fa; --drift-loss-border: #d3dbe9; --drift-loss-fg: #56647a;
  /* coverage staleness (bronze) — currency states, not severity */
  --stale-bg: #f7eedf; --stale-border: #e6d7bc; --stale-fg: #775f32;
  /* chart series — never severity colors */
  --chart-1: #037ac0; --chart-2: #009aba; --chart-3: #b37903; --chart-4: #83807b;
  /* type: Instrument Sans UI/prose, Geist Mono for technical values */
  --sans: "Instrument Sans", "Helvetica Neue", Arial, sans-serif;
  --mono: "Geist Mono", "SFMono-Regular", Consolas, ui-monospace, monospace;
  /* spacing (4px grid) */
  --space-2: 8px; --space-3: 12px; --space-4: 16px; --space-5: 24px; --space-6: 32px;
  /* radius scale (round by default; pills for chips) */
  --r-sm: 8px; --r-md: 12px; --r-lg: 16px; --r-xl: 24px; --r-full: 999px;
  /* soft layered elevation (no hard offsets) */
  --shadow-xs: 0 1px 2px rgba(35,31,25,0.05);
  --shadow-sm: 0 1px 2px rgba(35,31,25,0.06), 0 2px 8px rgba(35,31,25,0.04);
  --shadow-md: 0 2px 6px rgba(35,31,25,0.08), 0 8px 24px rgba(35,31,25,0.08);
  --shadow-lg: 0 8px 24px rgba(35,31,25,0.12), 0 24px 48px rgba(35,31,25,0.12);
  --scrim: rgba(21,18,15,0.4);
}
/* dark tokens — factored so the media default and the explicit toggle agree */
@media (prefers-color-scheme: dark) {
  :root:not([data-theme="light"]) {
    --paper: #15120f; --surface: #1e1b17; --sunken: #191613;
    --inverted: #eae7e4; --on-inverted: #231f19;
    --ink: #eae7e4; --body: #d9d5d1; --muted: #898581;
    --hairline: #383530; --border-strong: #514c46;
    --accent: #6bbeff; --accent-hover: #9ddaff; --on-accent: #063352;
    --accent-soft: #272f33; --link: #59acee; --focus: #6bbeff;
    --danger: #f08c82; --danger-soft: #331b17; --danger-border: #55231d; --on-danger: #331b17;
    --ok: #57c07f; --ok-soft: #17281c; --ok-border: #1f4029;
    --warn: #e0aa4a; --warn-soft: #2d2413; --warn-border: #4a3a12;
    --sev-critical-fill: #c44039; --sev-critical-text: #ffffff; --sev-critical-dot: #e0564e;
    --sev-high-bg: #352414;  --sev-high-border: #512f10;  --sev-high-fg: #eba57b;  --sev-high-dot: #df7a32;
    --sev-medium-bg: #322713; --sev-medium-border: #49360f; --sev-medium-fg: #d7b16a; --sev-medium-dot: #c68d00;
    --sev-low-bg: #192c2e;   --sev-low-border: #144048;   --sev-low-fg: #62c8df;   --sev-low-dot: #00aed1;
    --sev-info-bg: #23272c;  --sev-info-border: #3a4149;  --sev-info-fg: #a9b3bf;  --sev-info-dot: #8b98a6;
    --drift-gain-bg: #241b33; --drift-gain-border: #423658; --drift-gain-fg: #c0abe9;
    --drift-change-bg: #2f1725; --drift-change-border: #533044; --drift-change-fg: #e2a0c5;
    --drift-loss-bg: #1d2127; --drift-loss-border: #383d47; --drift-loss-fg: #aeb8c9;
    --stale-bg: #262014; --stale-border: #463b28; --stale-fg: #cbb48c;
    --chart-1: #6bbeff; --chart-2: #00aed1; --chart-3: #c68d00; --chart-4: #898581;
    --shadow-xs: 0 1px 2px rgba(0,0,0,0.3);
    --shadow-sm: 0 1px 2px rgba(0,0,0,0.35), 0 2px 8px rgba(0,0,0,0.25);
    --shadow-md: 0 2px 6px rgba(0,0,0,0.4), 0 8px 24px rgba(0,0,0,0.4);
    --shadow-lg: 0 8px 24px rgba(0,0,0,0.5), 0 24px 48px rgba(0,0,0,0.5);
    --scrim: rgba(0,0,0,0.55);
  }
}
:root[data-theme="dark"] {
  --paper: #15120f; --surface: #1e1b17; --sunken: #191613;
  --inverted: #eae7e4; --on-inverted: #231f19;
  --ink: #eae7e4; --body: #d9d5d1; --muted: #898581;
  --hairline: #383530; --border-strong: #514c46;
  --accent: #6bbeff; --accent-hover: #9ddaff; --on-accent: #063352;
  --accent-soft: #272f33; --link: #59acee; --focus: #6bbeff;
  --danger: #f08c82; --danger-soft: #331b17; --danger-border: #55231d; --on-danger: #331b17;
  --ok: #57c07f; --ok-soft: #17281c; --ok-border: #1f4029;
  --warn: #e0aa4a; --warn-soft: #2d2413; --warn-border: #4a3a12;
  --sev-critical-fill: #c44039; --sev-critical-text: #ffffff; --sev-critical-dot: #e0564e;
  --sev-high-bg: #352414;  --sev-high-border: #512f10;  --sev-high-fg: #eba57b;  --sev-high-dot: #df7a32;
  --sev-medium-bg: #322713; --sev-medium-border: #49360f; --sev-medium-fg: #d7b16a; --sev-medium-dot: #c68d00;
  --sev-low-bg: #192c2e;   --sev-low-border: #144048;   --sev-low-fg: #62c8df;   --sev-low-dot: #00aed1;
  --sev-info-bg: #23272c;  --sev-info-border: #3a4149;  --sev-info-fg: #a9b3bf;  --sev-info-dot: #8b98a6;
  --drift-gain-bg: #241b33; --drift-gain-border: #423658; --drift-gain-fg: #c0abe9;
  --drift-change-bg: #2f1725; --drift-change-border: #533044; --drift-change-fg: #e2a0c5;
  --drift-loss-bg: #1d2127; --drift-loss-border: #383d47; --drift-loss-fg: #aeb8c9;
  --stale-bg: #262014; --stale-border: #463b28; --stale-fg: #cbb48c;
  --chart-1: #6bbeff; --chart-2: #00aed1; --chart-3: #c68d00; --chart-4: #898581;
  --shadow-xs: 0 1px 2px rgba(0,0,0,0.3);
  --shadow-sm: 0 1px 2px rgba(0,0,0,0.35), 0 2px 8px rgba(0,0,0,0.25);
  --shadow-md: 0 2px 6px rgba(0,0,0,0.4), 0 8px 24px rgba(0,0,0,0.4);
  --shadow-lg: 0 8px 24px rgba(0,0,0,0.5), 0 24px 48px rgba(0,0,0,0.5);
  --scrim: rgba(0,0,0,0.55);
}
* { box-sizing: border-box; }
body { margin: 0; background: var(--paper); color: var(--body);
  font-family: var(--sans); font-size: 13px; line-height: 1.5;
  -webkit-font-smoothing: antialiased; }
a { color: var(--link); text-decoration: none; }
a:hover { text-decoration: underline; }
code, .mono { font-family: var(--mono); }
.microlabel { font-family: var(--mono); font-size: 11px; font-weight: 500;
  text-transform: uppercase; letter-spacing: 0.07em; color: var(--muted); }
.wordmark { font-weight: 700; font-size: 15px; color: var(--ink);
  letter-spacing: -0.015em; }
.wordmark .chip { font-family: var(--mono); font-size: 11px; font-weight: 600;
  background: var(--accent-soft); color: var(--link); border-radius: var(--r-sm);
  padding: 1px 6px; margin-left: 6px; }
.center { min-height: 100vh; display: flex; align-items: center; justify-content: center;
  padding: var(--space-6); }
main { padding: var(--space-6); max-width: 1440px; margin: 0 auto; }
.center main, .center .card { max-width: none; }
.card { background: var(--surface); border: 1px solid var(--hairline);
  border-radius: var(--r-lg); box-shadow: var(--shadow-sm);
  padding: var(--space-6); width: 360px; }
.section { background: var(--surface); border: 1px solid var(--hairline);
  border-radius: var(--r-lg); box-shadow: var(--shadow-sm);
  padding: var(--space-5); margin-bottom: var(--space-5); }
h1 { font-size: 18px; margin: 0 0 var(--space-3); color: var(--ink);
  letter-spacing: -0.015em; }
h2 { font-size: 14px; margin: 0 0 var(--space-4); color: var(--ink);
  letter-spacing: -0.015em; }
p { margin: 0 0 var(--space-4); }
label { display: block; margin-bottom: var(--space-4); }
label span { display: block; margin-bottom: 4px; }
input, select { width: 100%; font-family: var(--mono); font-size: 13px;
  padding: 8px 10px; border: 1px solid var(--hairline); border-radius: var(--r-md);
  background: var(--surface); color: var(--ink); }
input:focus, select:focus { outline: none;
  border-color: var(--focus);
  box-shadow: 0 0 0 2px var(--surface), 0 0 0 4px var(--focus); }
button, .btn { font-family: var(--sans); font-size: 13px; font-weight: 500;
  padding: 8px 16px; border: 1px solid var(--accent); background: var(--accent);
  color: var(--on-accent); border-radius: var(--r-md); cursor: pointer; }
button:hover, .btn:hover { background: var(--accent-hover); border-color: var(--accent-hover); }
button.secondary, .btn.secondary, .btn.ghost, button.ghost { background: var(--surface); color: var(--ink); border-color: var(--border-strong); }
button.secondary:hover, .btn.secondary:hover, .btn.ghost:hover, button.ghost:hover { background: var(--sunken); border-color: var(--border-strong); }
button.danger, .btn.danger { background: var(--danger); border-color: var(--danger); color: var(--on-danger); }
.error { border: 1px solid var(--danger-border); background: var(--danger-soft);
  color: var(--danger); padding: var(--space-3); border-radius: var(--r-md);
  margin-bottom: var(--space-4); font-size: 13px; }
.notice { border: 1px solid var(--accent-soft); background: var(--accent-soft);
  color: var(--link); padding: var(--space-3); border-radius: var(--r-md);
  margin-bottom: var(--space-4); font-size: 13px; }
.badge { font-family: var(--mono); font-size: 10px; font-weight: 600; text-transform: uppercase;
  letter-spacing: 0.06em; border: 1px solid var(--border-strong); color: var(--ink);
  border-radius: var(--r-full); padding: 1px 8px;
  display: inline-block; white-space: nowrap; }
.badge.off { color: var(--muted); border-color: var(--hairline); }

/* ---- shared component classes (translated from design-system/components/*) ---- */
/* Banner — running-scan / status banners */
.banner { display: flex; align-items: flex-start; gap: var(--space-3);
  border: 1px solid var(--hairline); background: var(--surface); box-shadow: var(--shadow-xs);
  border-radius: var(--r-lg); padding: var(--space-4); margin-bottom: var(--space-5); }
.banner.info { border-color: var(--accent-soft); background: var(--accent-soft); color: var(--link); }
.banner.ok { border-color: var(--ok-border); background: var(--ok-soft); color: var(--ok); }
.banner.warn { border-color: var(--warn-border); background: var(--warn-soft); color: var(--warn); }
.banner.danger { border-color: var(--danger-border); background: var(--danger-soft); color: var(--danger); }
/* KPI tiles */
.kpis { display: flex; flex-wrap: wrap; gap: var(--space-4); margin-bottom: var(--space-5); }
.kpi { flex: 1 1 160px; background: var(--surface); border: 1px solid var(--hairline);
  border-radius: var(--r-lg); box-shadow: var(--shadow-sm); padding: var(--space-4) var(--space-5); }
.kpi .kpi-label { font-family: var(--mono); font-size: 11px; font-weight: 500;
  text-transform: uppercase; letter-spacing: 0.07em; color: var(--muted); margin-bottom: 6px; }
.kpi .kpi-num { font-family: var(--mono); font-size: 28px; font-weight: 600; color: var(--ink); line-height: 1.1; }
.kpi .kpi-delta { font-family: var(--mono); font-size: 11px; color: var(--muted); margin-top: 4px; }
/* Severity pills — five levels, exactly; Critical the only solid fill and only red */
.sev { display: inline-flex; align-items: center; gap: 5px; font-family: var(--mono);
  font-size: 10px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.06em;
  border-radius: var(--r-full); padding: 1px 8px; border: 1px solid transparent; white-space: nowrap; }
.sev .sev-dot { width: 6px; height: 6px; border-radius: 50%; flex: none; }
.sev-critical { background: var(--sev-critical-fill); color: var(--sev-critical-text); border-color: var(--sev-critical-fill); }
.sev-critical .sev-dot { background: var(--sev-critical-text); }
.sev-high { background: var(--sev-high-bg); color: var(--sev-high-fg); border-color: var(--sev-high-border); }
.sev-high .sev-dot { background: var(--sev-high-dot); }
.sev-medium { background: var(--sev-medium-bg); color: var(--sev-medium-fg); border-color: var(--sev-medium-border); }
.sev-medium .sev-dot { background: var(--sev-medium-dot); }
.sev-low { background: var(--sev-low-bg); color: var(--sev-low-fg); border-color: var(--sev-low-border); }
.sev-low .sev-dot { background: var(--sev-low-dot); }
.sev-info { background: var(--sev-info-bg); color: var(--sev-info-fg); border-color: var(--sev-info-border); }
.sev-info .sev-dot { background: var(--sev-info-dot); }
/* Change chips (drift) — rounded-rect, drift palette, never the severity ramp */
.chip { display: inline-flex; align-items: center; gap: 5px; font-family: var(--mono);
  font-size: 11px; font-weight: 500; border-radius: var(--r-sm); padding: 2px 8px;
  border: 1px solid var(--hairline); background: var(--sunken); color: var(--body); white-space: nowrap; }
.chip.gain { background: var(--drift-gain-bg); border-color: var(--drift-gain-border); color: var(--drift-gain-fg); }
.chip.change { background: var(--drift-change-bg); border-color: var(--drift-change-border); color: var(--drift-change-fg); }
.chip.loss { background: var(--drift-loss-bg); border-color: var(--drift-loss-border); color: var(--drift-loss-fg); }
.chip.stale { background: var(--stale-bg); border-color: var(--stale-border); color: var(--stale-fg); }
/* Tabs */
.tabs { display: flex; gap: 2px; border-bottom: 1px solid var(--hairline); margin-bottom: var(--space-5); }
.tab { display: inline-flex; align-items: center; height: 36px; padding: 0 14px;
  font-family: var(--sans); font-size: 13px; font-weight: 500; color: var(--text-secondary, var(--muted));
  border: none; background: transparent; border-bottom: 2px solid transparent; cursor: pointer; }
.tab:hover { color: var(--ink); text-decoration: none; }
.tab.active { color: var(--link); border-bottom-color: var(--accent); font-weight: 600; }
/* Dialog + Drawer + shared scrim */
.scrim { position: fixed; inset: 0; background: var(--scrim); z-index: 40; }
.dialog-panel { background: var(--surface); border: 1px solid var(--hairline);
  border-radius: var(--r-xl); box-shadow: var(--shadow-lg); padding: var(--space-6);
  width: 440px; max-width: calc(100vw - 32px); }
.drawer-panel { position: fixed; top: 0; right: 0; bottom: 0; width: 480px; max-width: 92vw;
  background: var(--surface); border-left: 1px solid var(--hairline); box-shadow: var(--shadow-lg);
  z-index: 41; padding: var(--space-6); overflow-y: auto; }
.dialog-actions, .drawer-actions { display: flex; justify-content: flex-end; gap: var(--space-3); margin-top: var(--space-5); }
/* Table row affordances */
.vg-table tbody tr:hover { background: var(--sunken); }
.vg-table tbody tr.row-selected { background: var(--accent-soft); box-shadow: inset 3px 0 0 var(--accent); }
/* Empty-state block (fact + next action) */
.emptystate { border: 1px dashed var(--border-strong); background: var(--sunken);
  border-radius: var(--r-lg); padding: var(--space-6); text-align: center; color: var(--muted); }
.emptystate h2 { color: var(--ink); }

/* ---- app shell: top nav, org switcher, command palette, toasts ---- */
.topnav { display: flex; align-items: center; gap: 20px; height: 56px; padding: 0 24px;
  background: var(--surface); border-bottom: 1px solid var(--hairline);
  position: sticky; top: 0; z-index: 30; }
.topnav .brand { display: inline-flex; align-items: center; gap: 8px; flex: none; text-decoration: none; }
.topnav .brand:hover { text-decoration: none; }
.brand-glyph { display: inline-block; width: 20px; height: 20px; color: var(--accent); flex: none; }
.org-btn { display: inline-flex; align-items: center; gap: 6px; height: 26px; padding: 0 10px;
  border-radius: var(--r-full); background: var(--sunken); border: 1px solid var(--hairline);
  font-family: var(--mono); font-size: 11.5px; font-weight: 500; color: var(--muted); cursor: pointer; }
.org-btn:hover { background: var(--surface); border-color: var(--border-strong); }
.org-btn .chev { font-size: 9px; }
.navpills { display: flex; gap: 4px; margin-left: 8px; }
.navpill { display: inline-flex; align-items: center; gap: 6px; height: 32px; padding: 0 12px;
  border-radius: var(--r-full); font-family: var(--sans); font-size: 13px; font-weight: 500;
  color: var(--muted); background: transparent; text-decoration: none; }
.navpill:hover { color: var(--body); background: var(--sunken); text-decoration: none; }
.navpill.active { color: var(--link); background: var(--accent-soft); font-weight: 600; }
.navpill .pill-count { font: 600 10.5px var(--mono); padding: 1px 6px; border-radius: var(--r-full);
  background: var(--sunken); color: var(--muted); }
.navactions { margin-left: auto; display: flex; align-items: center; gap: 12px; flex: none; }
.scan-status { display: inline-flex; align-items: center; gap: 6px; font-family: var(--mono);
  font-size: 11px; text-transform: uppercase; letter-spacing: 0.06em; color: var(--muted); }
.iconbtn { display: inline-flex; align-items: center; justify-content: center; min-width: 30px; height: 30px;
  padding: 0 6px; border-radius: var(--r-md); border: 1px solid var(--hairline); background: var(--surface);
  color: var(--body); font-family: var(--mono); font-size: 11px; cursor: pointer; }
.iconbtn:hover { background: var(--sunken); }
.iconbtn svg { width: 16px; height: 16px; }
.cmdk-btn { gap: 4px; }
.ver { font: 400 11px var(--mono); color: var(--muted); }
/* Inbox bell — a disclosure that opens the recent-messages menu (TopNav.jsx). */
.msgmenu { position: relative; display: inline-flex; }
.msgmenu > summary { list-style: none; cursor: pointer; display: inline-flex; }
.msgmenu > summary::-webkit-details-marker { display: none; }
.msgnav { position: relative; display: inline-flex; align-items: center; justify-content: center;
  min-width: 30px; height: 30px; padding: 0 6px; border-radius: var(--r-md);
  border: 1px solid var(--hairline); background: var(--surface); color: var(--body); }
.msgmenu[open] > summary .msgnav { background: var(--sunken); color: var(--accent); }
.msgnav:hover { background: var(--sunken); }
.msgnav svg { width: 18px; height: 18px; }
.msgnav .count { position: absolute; top: -5px; right: -5px; font-family: var(--mono);
  font-size: 10px; font-weight: 600; letter-spacing: 0.02em; background: var(--accent);
  color: var(--on-accent); border-radius: var(--r-full); padding: 0 5px; min-width: 16px;
  height: 16px; line-height: 16px; text-align: center; box-shadow: 0 0 0 2px var(--surface); }
.msgmenu-panel { position: absolute; right: 0; top: calc(100% + 6px); width: 340px;
  max-width: calc(100vw - 32px); background: var(--surface); border: 1px solid var(--hairline);
  border-radius: var(--r-md); box-shadow: var(--shadow-md); padding: 6px; z-index: 35;
  display: flex; flex-direction: column; gap: 2px; }
.msgmenu-panel .microlabel { padding: 4px 6px 2px; }
.msgmenu-row { display: flex; align-items: flex-start; gap: 10px; width: 100%; padding: 9px 8px;
  border: none; border-radius: var(--r-sm); background: transparent; text-align: left;
  text-decoration: none; color: var(--body); }
.msgmenu-row:hover { background: var(--sunken); text-decoration: none; }
.msgmenu-row .unread-dot { width: 7px; height: 7px; border-radius: var(--r-full); margin-top: 5px;
  flex: none; background: transparent; }
.msgmenu-row.unread .unread-dot { background: var(--accent); }
.msgmenu-row .msg-body { display: flex; flex-direction: column; gap: 3px; min-width: 0; flex: 1; }
.msgmenu-row .msg-top { display: flex; align-items: center; gap: 8px; }
.msgmenu-row .msg-class { font-family: var(--mono); font-size: 10px; font-weight: 600;
  text-transform: uppercase; letter-spacing: 0.06em; color: var(--muted);
  border: 1px solid var(--hairline); border-radius: var(--r-full); padding: 1px 6px; }
.msgmenu-row .msg-when { margin-left: auto; font-family: var(--mono); font-size: 11px; color: var(--muted); white-space: nowrap; }
.msgmenu-row .msg-text { font-size: 12.5px; line-height: 1.5; color: var(--body); }
.msgmenu-row.unread .msg-text { font-weight: 500; }
.msgmenu-all { border: none; border-top: 1px solid var(--hairline); background: transparent;
  padding: 9px 8px 4px; font-size: 12px; font-weight: 500; color: var(--link);
  text-align: left; text-decoration: none; margin-top: 2px; }
.msgmenu-all:hover { text-decoration: underline; }
.msgmenu-empty { padding: 8px; font-size: 12.5px; color: var(--muted); }
/* Org switcher — a disclosure that opens the org menu (OrgSwitcher.jsx). */
.orgmenu { position: relative; display: inline-flex; flex: none; }
.orgmenu > summary { list-style: none; cursor: pointer; display: inline-flex; }
.orgmenu > summary::-webkit-details-marker { display: none; }
.orgmenu-panel { position: absolute; left: 0; top: calc(100% + 6px); width: 224px;
  background: var(--surface); border: 1px solid var(--hairline); border-radius: var(--r-md);
  box-shadow: var(--shadow-md); padding: 5px; z-index: 35; }
.orgmenu-panel .microlabel { padding: 6px 10px 4px; }
.orgmenu-item { display: flex; align-items: center; gap: 8px; width: 100%; height: 32px;
  padding: 0 10px; border: none; border-radius: var(--r-sm); background: transparent;
  text-align: left; font-family: var(--mono); font-size: 12px; color: var(--ink); }
.orgmenu-item:hover { background: var(--sunken); }
.orgmenu-item .org-assets { font-size: 10.5px; color: var(--muted); }
.orgmenu-item .org-check { margin-left: auto; color: var(--link); display: inline-flex; }
.orgmenu-item .org-check svg { width: 13px; height: 13px; }
.acctmenu { position: relative; }
.acctmenu > summary { list-style: none; cursor: pointer; display: inline-flex; }
.acctmenu > summary::-webkit-details-marker { display: none; }
.avatar { display: inline-flex; align-items: center; justify-content: center; width: 28px; height: 28px;
  border-radius: var(--r-full); background: var(--inverted); color: var(--on-inverted);
  font-family: var(--mono); font-size: 12px; font-weight: 600; }
.acctmenu-panel { position: absolute; right: 0; top: calc(100% + 6px); min-width: 200px;
  background: var(--surface); border: 1px solid var(--hairline); border-radius: var(--r-md);
  box-shadow: var(--shadow-md); padding: 6px; z-index: 35; display: flex; flex-direction: column; gap: 2px; }
.acctmenu-panel a, .acctmenu-panel .menuitem { display: block; width: 100%; text-align: left;
  padding: 7px 10px; border-radius: var(--r-sm); font-size: 13px; font-weight: 500;
  color: var(--body); background: transparent; border: none; cursor: pointer; text-decoration: none; }
.acctmenu-panel a:hover, .acctmenu-panel .menuitem:hover { background: var(--sunken); text-decoration: none; }
.acctmenu-panel form { margin: 0; }
.acctmenu-panel .cmdk-hintitem { display: flex; align-items: center; justify-content: space-between; gap: var(--space-3); }
.acctmenu-panel .menu-hint { font-family: var(--mono); font-size: 11px; color: var(--muted); }
/* Toast stack — acts post here across the post-redirect-get (ToastStack.jsx / Toast.jsx). */
.toaststack { position: fixed; right: 24px; bottom: 24px; z-index: 60;
  display: flex; flex-direction: column; align-items: flex-end; gap: 10px;
  width: 340px; max-width: calc(100vw - 40px); pointer-events: none; }
.toast { display: flex; align-items: flex-start; gap: 10px; width: 100%;
  background: var(--surface); border: 1px solid var(--hairline); border-radius: var(--r-lg);
  box-shadow: var(--shadow-md); padding: 12px 14px; pointer-events: auto;
  animation: vg-toast-in 0.28s var(--ease-out, ease); }
.toast.leaving { animation: vg-toast-out 0.24s ease forwards; }
.toast .toast-dot { width: 8px; height: 8px; border-radius: var(--r-full); flex: none; margin-top: 5px; background: var(--muted); }
.toast.ok .toast-dot { background: var(--ok); }
.toast.warn .toast-dot { background: var(--warn); }
.toast.danger .toast-dot { background: var(--danger); }
.toast .toast-body { display: flex; flex-direction: column; gap: 2px; min-width: 0; flex: 1; }
.toast .toast-title { font-size: 13px; font-weight: 600; color: var(--ink); }
.toast .toast-desc { font-size: 12.5px; line-height: 1.45; color: var(--muted); }
.toast .toast-x { flex: none; margin: -2px -4px 0 0; width: 22px; height: 22px; padding: 0;
  border: none; background: transparent; color: var(--muted); cursor: pointer; border-radius: var(--r-sm);
  display: inline-flex; align-items: center; justify-content: center; font-size: 14px; }
.toast .toast-x:hover { background: var(--sunken); color: var(--ink); }
@keyframes vg-toast-in { from { opacity: 0; transform: translateY(6px) scale(0.985); } to { opacity: 1; transform: none; } }
@keyframes vg-toast-out { from { opacity: 1; transform: none; } to { opacity: 0; transform: translateY(6px); } }
/* App footer — console variant of components/navigation/Footer.jsx. */
.appfooter { display: flex; align-items: center; gap: 16px; padding: 14px 32px;
  border-top: 1px solid var(--hairline); margin-top: auto; }
.appfooter .foot-note { font-size: 12px; color: var(--muted); }
.appfooter .foot-links { margin-left: auto; display: flex; gap: 16px; align-items: center; }
.appfooter .foot-links a { font-size: 12px; color: var(--muted); text-decoration: none; }
.appfooter .foot-links a:hover { color: var(--ink); text-decoration: none; }
body { min-height: 100vh; display: flex; flex-direction: column; }
body > main, body > .center { flex: 1 0 auto; }
/* Command palette scaffold */
.cmdk { position: fixed; inset: 0; z-index: 50; display: flex; align-items: flex-start; justify-content: center; }
.cmdk[hidden] { display: none; }
.cmdk-scrim { position: absolute; inset: 0; background: var(--scrim); }
.cmdk-panel { position: relative; margin-top: 12vh; width: 560px; max-width: calc(100vw - 32px);
  background: var(--surface); border: 1px solid var(--hairline); border-radius: var(--r-lg);
  box-shadow: var(--shadow-lg); padding: var(--space-4); }
.cmdk-input { width: 100%; font-family: var(--sans); font-size: 14px; margin-bottom: var(--space-3); }
.cmdk-group { display: flex; flex-direction: column; gap: 2px; margin-top: var(--space-3); }
.cmdk-group[hidden] { display: none; }
.cmdk-item { display: flex; align-items: baseline; justify-content: space-between; gap: var(--space-3);
  width: 100%; text-align: left; padding: 7px 10px; border: 0; background: transparent; cursor: pointer;
  border-radius: var(--r-sm); font-family: var(--sans); font-size: 13px;
  color: var(--body); text-decoration: none; }
.cmdk-item[hidden] { display: none; }
.cmdk-item:hover, .cmdk-item.active { background: var(--sunken); text-decoration: none; }
.cmdk-item .cmdk-hint { font-family: var(--mono); font-size: 11px; color: var(--muted); }
.cmdk-empty { padding: 7px 10px; font-size: 13px; color: var(--muted); }
.cmdk-empty[hidden] { display: none; }

/* ---- legacy page classes (page bodies still in flight; kept intact) ---- */
.kv { display: flex; gap: var(--space-4); margin-bottom: var(--space-3); }
.kv .k { color: var(--muted); width: 90px; }
.secret { font-family: var(--mono); word-break: break-all; background: var(--sunken);
  border: 1px solid var(--hairline); border-radius: var(--r-sm);
  padding: var(--space-3); margin-bottom: var(--space-4); }
.row { display: flex; gap: var(--space-3); align-items: center; }
table { width: 100%; border-collapse: collapse; }
th { text-align: left; font-family: var(--mono); font-size: 10px; font-weight: 600;
  text-transform: uppercase; letter-spacing: 0.06em; color: var(--muted);
  padding: var(--space-3) var(--space-4) var(--space-3) 0; border-bottom: 1px solid var(--border-strong); }
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
.census { border: 1px solid var(--hairline); background: var(--sunken);
  border-radius: var(--r-md); padding: var(--space-4); margin-top: var(--space-4); }
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
  width: 2px; background: var(--border-strong); }
ol.chain li:last-child::before { display: none; }
ol.chain li::after { content: ""; position: absolute; left: 0; top: 3px;
  width: 8px; height: 8px; border-radius: var(--r-full); background: var(--accent); }
ol.chain .chainval { margin: 2px 0; }
.rulehead { display: flex; justify-content: space-between; align-items: baseline;
  gap: var(--space-4); flex-wrap: wrap; margin-bottom: var(--space-4); }
.rulehead h2 { margin: 0; }
.rulehead .ver { white-space: nowrap; }
.members { display: flex; flex-direction: column; gap: var(--space-4); }
.mgroup { border: 1px solid var(--hairline); border-radius: var(--r-md); overflow: hidden; }
.mgroup-head { display: flex; align-items: baseline; gap: var(--space-3);
  padding: var(--space-3) var(--space-4); border-bottom: 1px solid var(--hairline);
  background: var(--sunken); }
.mgroup-head .count { font-family: var(--mono); font-size: 13px; font-weight: 600; margin-left: auto; }
.mgroup-list { list-style: none; margin: 0; padding: 0; }
.mgroup-list li { padding: var(--space-3) var(--space-4); border-bottom: 1px solid var(--hairline); }
.mgroup-list li:last-child { border-bottom: none; }
.mgroup-empty { padding: var(--space-3) var(--space-4); color: var(--muted); }
.fullmute { padding: var(--space-4); border-left: 3px solid var(--border-strong);
  background: var(--sunken); border-radius: var(--r-md); }
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
  color: var(--muted); border: 1px solid var(--hairline); border-radius: var(--r-full); padding: 1px 6px; }
.avail { display: flex; gap: var(--space-5); flex-wrap: wrap; margin-bottom: var(--space-5); }
.avail .k { color: var(--muted); margin-right: var(--space-3); }
.board { display: grid; grid-template-columns: 150px 1fr 1fr; border: 1px solid var(--hairline);
  border-radius: var(--r-lg); box-shadow: var(--shadow-sm); overflow: hidden;
  background: var(--surface); margin: var(--space-4) 0 var(--space-5); }
.board > div { border-right: 1px solid var(--hairline); border-bottom: 1px solid var(--hairline);
  padding: var(--space-4); }
.board .corner, .board .colhead, .board .rowhead { background: var(--sunken); }
.board .corner { display: flex; flex-direction: column; justify-content: flex-end; }
.board .colhead, .board .rowhead { display: flex; align-items: flex-end; }
.board .cell .count { font-family: var(--mono); font-size: 22px; font-weight: 600;
  margin: 2px 0 var(--space-3); }
.board .cell.hot { box-shadow: inset 3px 0 0 var(--accent); }
.board .cell ul, .movedlist { list-style: none; margin: 0; padding: 0; }
.board .cell li { font-family: var(--mono); font-size: 11px; padding: 1px 0; }
.board .cell li a { text-decoration: none; }
.board .cell .none { color: var(--muted); }
.precond { border: 1px solid var(--hairline); background: var(--surface);
  border-radius: var(--r-lg); box-shadow: var(--shadow-sm);
  padding: var(--space-5); margin-bottom: var(--space-5); }
.precond h2 { margin-top: 0; }
.moved { border-left: 3px solid var(--accent); background: var(--sunken);
  border-radius: var(--r-md); padding: var(--space-4); margin-bottom: var(--space-5); }
.movedlist li { font-family: var(--mono); font-size: 12px; padding: 2px 0; }
.msglist { list-style: none; margin: var(--space-5) 0 0; padding: 0; }
.msgitem { border: 1px solid var(--hairline); background: var(--surface);
  border-radius: var(--r-md); box-shadow: var(--shadow-xs);
  padding: var(--space-4); margin-bottom: var(--space-4); border-left: 3px solid var(--border-strong); }
.msgitem.unread { border-left-color: var(--accent); }
.msgitem-head { display: flex; align-items: baseline; gap: var(--space-3); flex-wrap: wrap;
  margin-bottom: var(--space-3); }
.msgitem-head .cause { font-family: var(--mono); font-size: 10px; font-weight: 600;
  text-transform: uppercase; letter-spacing: 0.06em; color: var(--muted);
  border: 1px solid var(--hairline); border-radius: var(--r-full); padding: 1px 6px; }
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
.msgdelivery { list-style: none; margin: var(--space-3) 0 0; padding: var(--space-3) 0 0;
  border-top: 1px solid var(--hairline); }
.msgdelivery li { padding: 2px 0; font-size: 12px; color: var(--muted); }
.msgdelivery li.delivery-failed { color: var(--ink); }
.msgdelivery li .why { cursor: help; text-decoration: underline dotted; }
.receipt { border: 1px solid var(--hairline); background: var(--surface);
  border-radius: var(--r-md); padding: var(--space-4); margin-top: var(--space-4); box-shadow: var(--shadow-sm); }
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
.meter { height: 6px; background: var(--sunken); border: 1px solid var(--hairline);
  border-radius: var(--r-full); overflow: hidden; margin: 0 0 var(--space-4); }
.meter .fill { height: 100%; background: var(--accent); }
.meter .fill.complete { background: var(--ok); }
table.jobs td .dot { margin-right: 6px; vertical-align: middle; }
table.jobs td.super { color: var(--muted); }
@media (prefers-reduced-motion: reduce) { .dot.live { animation: none; } }
`

// wordmark is the typed Verge ASM mark: sans "Verge" plus a mono "ASM" chip,
// paired with the placeholder pulse glyph (a signal dot inside two watch rings,
// azure only). The glyph is explicitly placeholder-quality (design system).
const wordmark = `<span class="brand-glyph" aria-hidden="true"><svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.75"><circle cx="10" cy="10" r="8"/><circle cx="10" cy="10" r="4"/><circle cx="10" cy="10" r="1.4" fill="currentColor" stroke="none"/></svg></span><span class="wordmark">Verge<span class="chip">ASM</span></span>`

// tmpl is the single template the whole console renders through. The shell blocks
// (head/foot/chrome) are parsed here; every per-screen file appends its own blocks
// to this same tmpl via `var _ = template.Must(tmpl.Parse(...))` (see templates_*.go).
var tmpl = template.Must(template.New("").Funcs(template.FuncMap{
	// integrationsEnabled exposes the compile-time #388 flag (integrations.go) to
	// templates, so the shell's command palette and the Settings tab bar can gate
	// the hidden Integrations surface without threading a data field through every
	// handler that renders the shell. It is false while the surface is hidden.
	"integrationsEnabled": func() bool { return integrationsEnabled },
	// designTokens returns the design-owned CSS-token vocabulary (design-system/
	// tokens/*.css, loaded by loadDesignTokens in templates_inventory.go) as trusted
	// CSS. The "head" block inlines it in a <style data-design-tokens> block gated to
	// the pages that opt in via a truthy .DesignTokens datum — currently only the
	// Inventory screen, whose frozen design tmpl styles against these token names.
	// Placed after pageCSS so the design tokens win the cascade for any overlapping
	// :root variable. It is design-authored CSS from the embedded package, no user
	// input, so template.CSS keeps html/template from escaping it.
	"designTokens": func() template.CSS { return template.CSS(designTokensCSS) }, // #nosec G203 -- design-owned CSS from the embedded design package (designfs), no user input
	// A design-served page (.DesignTokens) also gets a small shell reset, emitted in
	// the head block, that isolates the frozen design tmpl from the legacy pageCSS
	// cascade it is served inside. Two parts:
	//   - `body{display:block}` — the legacy shell makes body a flex column with
	//     `body > main {flex:1 0 auto}`; against the design tmpl's own `main {
	//     max-width; margin:0 auto }`, the flex context shrink-wraps main to its
	//     content width and stretches short states to the viewport height, so the app
	//     rendered /inventory narrower and taller than the frozen design. Block flow
	//     lets the tmpl size as authored (full width to its max, content height).
	//   - `main td{border-bottom:0}` + `main label{margin-bottom:0}` + `main label
	//     span{margin-bottom:0}` — pageCSS's GENERIC element selectors (a `td`
	//     border-bottom, a `label`/`label span` margin) bleed into the design tmpl's
	//     own <table>/<label> where its `.inv-*` classes don't restate them, adding a
	//     px per row and a few to the toolbar versus the isolated design. Scoped to
	//     `main` so the chrome (which legitimately uses those pageCSS rules) is
	//     untouched. These are the exact bleeds a computed-style diff localized on
	//     this screen; the pixel gate (G2) confirms the isolation is complete.
	//   - `.vg-center label{margin-bottom:0}` + `.vg-center label span{margin-bottom:0}`
	//     — the SAME class of bleed for the chrome-less SignIn/Setup auth surfaces
	//     (screens 4/5), which have no <main>: they render `.vg-center` straight in
	//     <body>, so pageCSS's generic `label{margin-bottom:16px}` / `label span{margin-
	//     bottom:4px}` bleed onto the `.vg-field`/`.vg-check` (which ARE <label>s) and
	//     their spans, growing every text field ~8px+ versus the token-only golden (whose
	//     labels carry no such margin). Only the label margins bleed — body line-height
	//     already matches (both pageCSS and tokens/base.css set var(--leading-normal)), so
	//     it is NOT reset here. `.vg-center` is unique to signin.tmpl/setup.tmpl, so this
	//     touches only those surfaces; a computed-style diff localized these exact bleeds
	//     and G2 confirms the isolation.
	// All of it is repo shell glue wiring the frozen tmpl into the app — not screen
	// styling — and is removed wholesale when the chrome converts to the design shell
	// (P4.4, WORKFLOW.md).
	// signDelta formats a vs-last-batch stat delta (drift.Delta.Change, P0.2 #443) as
	// the design's signed chip label: "+N" for a rise, "−N" (a true minus, the
	// voice's signed-delta rule — design-system README) for a fall, and "0" for no
	// movement. The tile picks the chip's tone; this only formats the number. The
	// output is digits and a sign only — safe to mark trusted so html/template keeps
	// the literal "+" rather than escaping it to an entity.
	"signDelta": func(n int) template.HTML {
		var s string
		switch {
		case n > 0:
			s = "+" + strconv.Itoa(n)
		case n < 0:
			s = "−" + strconv.Itoa(-n)
		default:
			s = "0"
		}
		return template.HTML(s) // #nosec G203 -- a sign and digits only, from an int; no user input
	},
}).Parse(`
{{define "head"}}<!doctype html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<script>try{var t=localStorage.getItem("verge-theme");if(t){document.documentElement.setAttribute("data-theme",t);}}catch(e){}</script>
<title>{{.Title}} · Verge ASM</title>{{if .Refresh}}<meta http-equiv="refresh" content="6">{{end}}<style>` + pageCSS + `</style>{{if .DesignTokens}}<style data-design-tokens>{{designTokens}}</style><style data-design-shell>body{display:block}main td{border-bottom:0}main label{margin-bottom:0}main label span{margin-bottom:0}main .scrim{background:transparent}.vg-center label{margin-bottom:0}.vg-center label span{margin-bottom:0}main .cv-meter .census{border:0;padding:0;margin:0}</style>{{end}}</head><body>{{end}}
{{define "foot"}}{{if .Chrome}}<footer class="appfooter"><span class="foot-note">verge asm · self-hosted · AGPL-3.0</span><span class="foot-links"><a href="https://github.com/winniel123/verge-asm/tree/main/docs" rel="noreferrer">Docs</a><a href="https://github.com/winniel123/verge-asm" rel="noreferrer">GitHub</a></span></footer>{{end}}<script>
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
/* Shell affordances: the light/dark data-theme toggle (prefers-color-scheme is the
   default; this is the explicit override, persisted), and the command-palette
   scaffold (Cmd/Ctrl-K). Both are guarded so they no-op on the chrome-less auth
   pages. Screen tickets fill the toast stack and palette results; T0 wires the shell.
   #315 wires the palette's overflow: the "Search everything" item (marked
   data-cmdk-search, matching design-system/examples/console/ConsoleApp.jsx's
   CommandPalette entry of the same label — SearchResults.jsx: "where ⌘K's 'see
   everything' lands") stays visible no matter what the query filters out, and its
   href tracks the typed query so Enter or a click hands off to
   /search?q=<query> — an empty query lands on /search with no q, which the
   handler already browses unfiltered. */
(function () {
  function cmdk() { return document.getElementById("cmdk"); }
  function input() { var p = cmdk(); return p ? p.querySelector(".cmdk-input") : null; }
  function items() { var p = cmdk(); return p ? Array.prototype.slice.call(p.querySelectorAll(".cmdk-item")) : []; }
  function visible() { return items().filter(function (el) { return !el.hasAttribute("hidden"); }); }
  function isOpen() { var p = cmdk(); return !!p && !p.hasAttribute("hidden"); }

  function setActive(i) {
    var vis = visible();
    items().forEach(function (el) { el.classList.remove("active"); });
    if (!vis.length) return;
    if (i < 0) i = 0; if (i >= vis.length) i = vis.length - 1;
    vis[i].classList.add("active");
  }
  function activeIndex() {
    var vis = visible();
    for (var i = 0; i < vis.length; i++) { if (vis[i].classList.contains("active")) return i; }
    return -1;
  }
  function filter(q) {
    var p = cmdk(); if (!p) return;
    var raw = (q || "").trim();          // kept in original case for the /search handoff
    var ql = raw.toLowerCase();
    var any = false;
    items().forEach(function (el) {
      if (el.hasAttribute("data-cmdk-search")) { el.removeAttribute("hidden"); return; } // overflow item: never filtered out
      var match = !ql || el.textContent.toLowerCase().indexOf(ql) !== -1;
      if (match) { el.removeAttribute("hidden"); any = true; } else { el.setAttribute("hidden", ""); }
    });
    var search = p.querySelector("[data-cmdk-search]");
    if (search) search.setAttribute("href", raw ? "/search?q=" + encodeURIComponent(raw) : "/search");
    Array.prototype.forEach.call(p.querySelectorAll("[data-cmdk-group]"), function (g) {
      if (g.querySelector(".cmdk-item:not([hidden])")) g.removeAttribute("hidden"); else g.setAttribute("hidden", "");
    });
    var empty = p.querySelector("[data-cmdk-empty]");
    if (empty) { if (any) empty.setAttribute("hidden", ""); else empty.removeAttribute("hidden"); }
    setActive(0);
  }
  function openPalette(open) {
    var p = cmdk(); if (!p) return;
    if (open) {
      p.removeAttribute("hidden");
      var inp = input();
      if (inp) { inp.value = ""; filter(""); inp.focus(); }
    } else {
      p.setAttribute("hidden", "");
    }
  }

  document.addEventListener("click", function (e) {
    var toggle = e.target.closest("[data-theme-toggle]");
    if (toggle) {
      var cur = document.documentElement.getAttribute("data-theme");
      var next = cur === "dark" ? "light" : "dark";
      document.documentElement.setAttribute("data-theme", next);
      try { localStorage.setItem("verge-theme", next); } catch (e2) {}
      return;
    }
    if (e.target.closest("[data-cmdk-open]")) { openPalette(true); return; }
    if (e.target.closest("[data-cmdk-close]")) { openPalette(false); return; }
  });
  document.addEventListener("input", function (e) {
    if (e.target && e.target.classList && e.target.classList.contains("cmdk-input")) filter(e.target.value);
  });
  document.addEventListener("keydown", function (e) {
    if ((e.metaKey || e.ctrlKey) && (e.key === "k" || e.key === "K")) {
      e.preventDefault();
      openPalette(!isOpen());
      return;
    }
    if (!isOpen()) return;
    if (e.key === "Escape") { e.preventDefault(); openPalette(false); return; }
    if (e.key === "ArrowDown") { e.preventDefault(); setActive(activeIndex() + 1); return; }
    if (e.key === "ArrowUp") { e.preventDefault(); setActive(activeIndex() - 1); return; }
    if (e.key === "Enter") {
      var vis = visible(); var idx = activeIndex(); if (idx < 0) idx = 0;
      if (vis[idx]) { e.preventDefault(); vis[idx].click(); }
      return;
    }
    if (e.key === "Tab") {
      // Focus trap: cycle focus among the input and the visible items only.
      var focusables = [input()].concat(visible()).filter(Boolean);
      if (!focusables.length) return;
      var ci = focusables.indexOf(document.activeElement);
      var ni = e.shiftKey ? ci - 1 : ci + 1;
      if (ni < 0) ni = focusables.length - 1;
      if (ni >= focusables.length) ni = 0;
      e.preventDefault();
      focusables[ni].focus();
    }
  });
})();
/* Toasts — an act carries a toast across the post-redirect-get in the URL's
   "toast" query (a base64url JSON blob; ConsoleApp fires the same toasts client-
   side). On load we read it, fire it into the toast stack, then strip it from the
   address bar so a refresh does not re-toast. Look and dismiss behaviour follow
   ToastStack.jsx / Toast.jsx; titles and descriptions are set as textContent,
   never HTML, so a toast cannot inject. */
(function () {
  var stack = document.getElementById("toasts");
  if (!stack) return;
  var params;
  try { params = new URLSearchParams(location.search); } catch (e) { return; }
  var raw = params.get("toast");
  if (!raw) return;
  // Strip the param so a reload or a shared link does not re-fire the toast.
  params.delete("toast");
  var qs = params.toString();
  try { history.replaceState(null, "", location.pathname + (qs ? "?" + qs : "") + location.hash); } catch (e) {}
  var list;
  try {
    var b = raw.replace(/-/g, "+").replace(/_/g, "/");
    for (var pad = (4 - (b.length & 3)) & 3; pad > 0; pad--) b += "=";
    list = JSON.parse(decodeURIComponent(escape(atob(b))));
  } catch (e) { return; }
  if (!list) return;
  if (!Array.isArray(list)) list = [list];
  function dismiss(el) {
    if (el.classList.contains("leaving")) return;
    el.classList.add("leaving");
    setTimeout(function () { if (el.parentNode) el.parentNode.removeChild(el); }, 260);
  }
  list.forEach(function (t) {
    if (!t || !t.title) return;
    var el = document.createElement("div");
    el.className = "toast " + (t.tone || "neutral");
    el.setAttribute("role", "status");
    var dot = document.createElement("span"); dot.className = "toast-dot"; el.appendChild(dot);
    var body = document.createElement("div"); body.className = "toast-body";
    var title = document.createElement("span"); title.className = "toast-title"; title.textContent = t.title; body.appendChild(title);
    if (t.description) { var d = document.createElement("span"); d.className = "toast-desc"; d.textContent = t.description; body.appendChild(d); }
    el.appendChild(body);
    var x = document.createElement("button"); x.type = "button"; x.className = "toast-x"; x.setAttribute("aria-label", "Dismiss"); x.textContent = "×";
    x.addEventListener("click", function () { dismiss(el); });
    el.appendChild(x);
    stack.appendChild(el);
    setTimeout(function () { dismiss(el); }, 5000);
  });
})();
</script></body></html>{{end}}

{{define "chrome"}}<header class="topnav">
<a class="brand" href="/">` + wordmark + `</a>
<details class="orgmenu"><summary aria-label="Switch organization"><span class="org-btn">{{.OrgName}} <span class="chev">&#9662;</span></span></summary>
<div class="orgmenu-panel" role="menu">
<div class="microlabel">Orgs</div>
<span class="orgmenu-item"><span>{{.OrgName}}</span>{{if .AssetCount}}<span class="org-assets">{{.AssetCount}}</span>{{end}}<span class="org-check" aria-label="current"><svg viewBox="0 0 18 18" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3.5 9.5l3.5 3.5 7.5-8"/></svg></span></span>
</div></details>
<nav class="navpills" aria-label="Primary">
<a class="navpill{{if eq .NavActive "dashboard"}} active{{end}}" href="/">Dashboard</a>
<a class="navpill{{if eq .NavActive "scope"}} active{{end}}" href="/scope">Scope</a>
<a class="navpill{{if eq .NavActive "inventory"}} active{{end}}" href="/inventory">Inventory</a>
<a class="navpill{{if eq .NavActive "drift"}} active{{end}}" href="/drift">Drift</a>
<a class="navpill{{if eq .NavActive "signals"}} active{{end}}" href="/signals">Signals{{if .SignalCount}} <span class="pill-count">{{.SignalCount}}</span>{{end}}</a>
<a class="navpill{{if eq .NavActive "exposure"}} active{{end}}" href="/exposure">Exposure</a>
<a class="navpill{{if eq .NavActive "coverage"}} active{{end}}" href="/coverage">Coverage</a>
<a class="navpill{{if eq .NavActive "graph"}} active{{end}}" href="/graph">Graph</a>
<a class="navpill{{if eq .NavActive "reports"}} active{{end}}" href="/reports">Reports</a>
</nav>
<div class="navactions">
{{if .Scanning}}<span class="scan-status"><span class="dot live"></span>Scan running</span>{{end}}
<button type="button" class="iconbtn cmdk-btn" data-cmdk-open aria-label="Command palette"><span class="mono">&#8984;K</span></button>
<details class="msgmenu"><summary aria-label="Messages"><span class="msgnav"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75"><path d="M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9"/><path d="M10.3 21a1.94 1.94 0 0 0 3.4 0"/></svg>{{if .Unread}}<span class="count">{{.Unread}}</span>{{end}}</span></summary>
<div class="msgmenu-panel" role="menu">
<div class="microlabel">Messages</div>
{{range .RecentMessages}}<a class="msgmenu-row{{if .Unread}} unread{{end}}" href="{{.Href}}"><span class="unread-dot"></span><span class="msg-body"><span class="msg-top"><span class="msg-class">{{.Class}}</span><span class="msg-when">{{.Rel}}</span></span><span class="msg-text">{{.Headline}}</span></span></a>{{else}}<div class="msgmenu-empty">No messages.</div>{{end}}
<a class="msgmenu-all" href="/inbox">All messages</a>
</div></details>
<button type="button" class="iconbtn" data-theme-toggle aria-label="Toggle theme"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75"><path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8Z"/></svg></button>
<details class="acctmenu"><summary aria-label="Account menu"><span class="avatar">{{.Initials}}</span></summary>
<div class="acctmenu-panel">
<a href="/profile">Profile</a>
{{if .IsAdmin}}<a href="/settings">Settings</a>{{end}}
<button type="button" class="menuitem cmdk-hintitem" data-cmdk-open>Command palette <span class="menu-hint">&#8984;K</span></button>
<form method="post" action="/logout"><button class="menuitem" type="submit">Sign out</button></form>
</div></details>
</div>
</header>
<div class="cmdk" id="cmdk" hidden><div class="cmdk-scrim" data-cmdk-close></div>
<div class="cmdk-panel" role="dialog" aria-label="Command palette" aria-modal="true">
<input class="cmdk-input" type="text" placeholder="Search screens and actions…" aria-label="Command">
<div class="cmdk-group" data-cmdk-group><div class="microlabel">Screens</div>
<a class="cmdk-item" href="/">Dashboard</a>
<a class="cmdk-item" href="/scope">Scope</a>
<a class="cmdk-item" href="/inventory">Inventory</a>
<a class="cmdk-item" href="/drift">Drift</a>
<a class="cmdk-item" href="/signals">Signals{{if .SignalCount}}<span class="cmdk-hint">{{.SignalCount}} open</span>{{end}}</a>
<a class="cmdk-item" href="/exposure">Exposure</a>
<a class="cmdk-item" href="/coverage">Coverage</a>
<a class="cmdk-item" href="/graph">Graph</a>
<a class="cmdk-item" href="/reports">Reports</a>
<a class="cmdk-item" href="/inbox">Inbox{{if .Unread}}<span class="cmdk-hint">{{.Unread}} unread</span>{{end}}</a>
<a class="cmdk-item" href="/profile">Profile</a>
{{if .IsAdmin}}<a class="cmdk-item" href="/settings?tab=sources">Sources</a>
<a class="cmdk-item" href="/settings?tab=aperture">Port aperture</a>
{{if integrationsEnabled}}<a class="cmdk-item" href="/settings?tab=integrations">Integrations</a>{{end}}
<a class="cmdk-item" href="/settings">Settings</a>{{end}}
</div>
<div class="cmdk-group" data-cmdk-group><div class="microlabel">Actions</div>
<a class="cmdk-item" href="/scans">Run scan</a>
<a class="cmdk-item" href="/scope">Add seed</a>
<a class="cmdk-item" href="/onboarding">First-run onboarding</a>
<button type="button" class="cmdk-item" data-theme-toggle>Toggle theme</button>
</div>
{{if .PaletteAssets}}<div class="cmdk-group" data-cmdk-group><div class="microlabel">Assets</div>
{{range .PaletteAssets}}<a class="cmdk-item" href="{{.Href}}">{{.Key}}</a>{{end}}
</div>{{end}}
<div class="cmdk-empty" data-cmdk-empty hidden>No matching screen or action</div>
<div class="cmdk-group"><a class="cmdk-item" href="/search" data-cmdk-search>Search everything</a></div>
</div></div>
<div class="toaststack" id="toasts" aria-live="polite" aria-atomic="false"></div>{{end}}
`))
