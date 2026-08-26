// capture.mjs — the /inventory pixel-parity capture harness (ticket #526).
//
// For each viewport (config.json) x theme (light,dark) x state (states.json),
// it opens a deterministic browser context, forces the theme via [data-theme],
// disables animation/caret, runs the state's JS, screenshots the `main` element,
// and either writes the golden PNG (--write-goldens) or pixelmatches the freshly
// captured candidate against the committed golden and reports the differing-pixel
// percentage.
//
// Modes:
//   --mode golden    load file:// the render-goldens static HTML (--page <file>)
//   --mode candidate log in to a running seeded server, then goto <base>/inventory
//                    (--base <url>)
//
// Design-owned inputs (READ ONLY): design-system/verify/config.json + states.json.
// This file is repo-owned harness (not swept into designfs's *.json embed, nor
// into CI gate G1).
//
// Repo-owned harness — not a design-owned artifact.

import { chromium } from 'playwright';
import pixelmatch from 'pixelmatch';
import { PNG } from 'pngjs';
import { readFileSync, writeFileSync, mkdirSync, existsSync } from 'node:fs';
import { dirname, resolve, join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
// harness/ -> verify/ -> design-system/
const verifyDir = resolve(__dirname, '..');
const designDir = resolve(verifyDir, '..');

function arg(name, def = undefined) {
  const i = process.argv.indexOf(name);
  if (i === -1) return def;
  const v = process.argv[i + 1];
  return v && !v.startsWith('--') ? v : true;
}
const has = (name) => process.argv.includes(name);

const mode = arg('--mode', 'golden'); // golden | candidate
const writeGoldens = has('--write-goldens');
const advisory = has('--advisory');
const screen = arg('--screen', 'inventory');
const pagePath = arg('--page'); // golden mode: single static HTML file (all states share it)
const pageDir = arg('--pagedir'); // golden mode: per-state HTML dir (<state>.html), used when states differ
const baseURL = arg('--base', 'http://localhost:8080'); // candidate mode
const user = arg('--user', 'operator');
const pass = arg('--pass', 'verge-dev-operator');
// candidate mode, profile screen: a dev route hit before EVERY state that (re)seeds the fixture
// (so the "minted" state's created token never leaks into the next state) and hands back the
// seeded current-session cookie (so the fixture's current row wears the "this device" badge
// without minting a fourth session). When set, it replaces the per-state /dev/session mint.
const adoptPath = arg('--adopt');
// candidate mode: hide the console chrome (the sticky <header class="topnav">) before the shot.
// The crop is `main`, so chrome is never in-frame — but it occupies ~56px above <main> in flow,
// which shifts a viewport-FIXED overlay (profile's dialogs) relative to the captured main box.
// The golden renders no chrome (empty stub), so hiding it here puts <main> at the viewport top on
// BOTH sides and the fixed overlay aligns. Chrome itself is out of Phase-A scope (shell #22).
const hideChrome = has('--hide-chrome');
// --full-page (#27f): screenshot the whole scrollable page (chrome + main + footer)
// instead of cropping to a selector, forcing full-page even when a screen's states.json
// crop still reads "main" (the crop stays frozen; the harness override is repo-owned).
// A states.json crop of "full-page" (the shell states) triggers the same path.
const fullPage = has('--full-page');
// --skip-state <id[,id...]> drops named states from the run. The shell's org-open state
// is skipped: orgs are not modeled (ADR-0073), so the switcher ships the static chip and
// the org-open golden defers with it (SPEC-CHANGE #28, AWAITING DESIGN).
const skipStates = String(arg('--skip-state', '') || '')
  .split(',')
  .map((s) => s.trim())
  .filter(Boolean);

const config = JSON.parse(readFileSync(join(verifyDir, 'config.json'), 'utf8'));
const statesDoc = JSON.parse(readFileSync(join(verifyDir, 'states.json'), 'utf8'));
const screenStates = statesDoc[screen];
if (!screenStates) throw new Error(`no states for screen ${screen}`);

// config.goldens already begins "goldens/…", so resolve it against the
// design-system root (designDir), not against a nested goldens/ dir.
// --goldens-root overrides where "goldens/…" resolves (used only by the
// body-flex diagnostic so it never overwrites the canonical committed goldens).
const goldensRootBase = arg('--goldens-root', designDir);
const goldensRoot = join(goldensRootBase, 'goldens');
const goldenPathFor = (state, vp, theme) =>
  join(goldensRootBase, config.goldens
    .replace('{screen}', screen)
    .replace('{state}', state)
    .replace('{viewport}', vp)
    .replace('{theme}', theme));

// A style that neutralises anything time-dependent so the capture is stable.
const FREEZE_CSS = `*,*::before,*::after{animation:none!important;transition:none!important;caret-color:transparent!important;scroll-behavior:auto!important}`;

async function login(context) {
  const page = await context.newPage();
  await page.goto(`${baseURL}/login`, { waitUntil: 'networkidle' });
  await page.fill('input[name="username"]', user);
  await page.fill('input[name="password"]', pass);
  await Promise.all([
    page.waitForNavigation({ waitUntil: 'networkidle' }).catch(() => {}),
    page.click('button[type="submit"]'),
  ]);
  await page.close();
}

// mintSession establishes a role's session in the context via the dev session mint
// (/dev/session/{role}, VERGE_DEV only), so a state's per-state `session` (states.json:
// "admin" | "viewer") is set before its route is captured. The endpoint signs in as the
// fixture account for the role and redirects, dropping the session cookie on the context.
async function mintSession(context, role) {
  const page = await context.newPage();
  await page.goto(`${baseURL}/dev/session/${role}`, { waitUntil: 'networkidle' });
  await page.close();
}

// applySeed reshapes the seeded fixture DB into a named variant before a state is captured, via a
// dev seed route (/dev/seed/{variant}, VERGE_DEV only). A screen (or a single state) may declare a
// `seed` in states.json — the Setup screen declares seed:"empty" so /dev/seed/empty empties the
// account table and reopens the first-run window under the pinned fixture token, letting GET /setup
// render the open bootstrap form. Screens with no `seed` never touch this. Candidate mode only.
async function applySeed(context, variant) {
  const page = await context.newPage();
  await page.goto(`${baseURL}/dev/seed/${encodeURIComponent(variant)}`, { waitUntil: 'networkidle' });
  await page.close();
}

// Screens whose states carry a per-state `session` (the error screen) mint per state;
// screens without it (inventory) log in once per context with --user/--pass.
const perStateSession = screenStates.states.some((s) => s.session);

function diffPercent(a, b) {
  if (a.width !== b.width || a.height !== b.height) {
    return { mismatch: true, aw: a.width, ah: a.height, bw: b.width, bh: b.height };
  }
  const { width, height } = a;
  const diff = new PNG({ width, height });
  const n = pixelmatch(a.data, b.data, diff.data, width, height, {
    threshold: config.pixelmatch.threshold,
    includeAA: config.pixelmatch.includeAA,
  });
  return { mismatch: false, diffPixels: n, total: width * height, pct: (n / (width * height)) * 100, diff };
}

async function run() {
  const browser = await chromium.launch();
  const results = [];
  let worst = 0;

  for (const vp of config.viewports) {
    for (const theme of config.themes) {
      const context = await browser.newContext({
        viewport: { width: vp.width, height: vp.height },
        deviceScaleFactor: config.deviceScaleFactor,
        reducedMotion: config.reducedMotion ? 'reduce' : 'no-preference',
        colorScheme: theme,
      });

      if (mode === 'candidate' && !perStateSession) await login(context);

      for (const st of screenStates.states) {
        if (skipStates.includes(st.id)) continue;
        const page = await context.newPage();
        if (mode === 'golden') {
          // Per-state file (--pagedir, error) or a single shared file (--page, inventory).
          const gp = pageDir ? join(pageDir, `${st.id}.html`) : pagePath;
          await page.goto(pathToFileURL(resolve(gp)).href, { waitUntil: 'networkidle' });
        } else {
          // Establish this state's session first, then navigate its own route (states.json's
          // per-state route) or the screen-level route (inventory). The profile screen uses the
          // --adopt reseed+cookie route (per state) instead of the per-state /dev/session mint,
          // so a prior state's minted token is reset and the current-session badge resolves.
          // A seed (screen-level or per-state) reshapes the fixture DB into a named variant first
          // (Setup's seed:"empty" empties accounts + reopens the setup window). Idempotent, so a
          // re-seed per state is safe.
          const seed = st.seed || screenStates.seed;
          if (seed) await applySeed(context, seed);
          if (adoptPath) {
            const prep = await context.newPage();
            await prep.goto(`${baseURL}${adoptPath}`, { waitUntil: 'networkidle' });
            await prep.close();
          } else if (st.session) {
            await mintSession(context, st.session);
          }
          let route = st.route || screenStates.route;
          // A state may declare a `variant` (signin's login-sso-none): ride it as a ?variant=
          // query the dev handler reads, so the candidate route matches the golden's variant.
          if (st.variant) route += (route.includes('?') ? '&' : '?') + 'variant=' + encodeURIComponent(st.variant);
          await page.goto(`${baseURL}${route}`, { waitUntil: 'networkidle' });
        }

        // Force the theme deterministically in BOTH modes (design colors.css
        // honours [data-theme]); independent of prefers-color-scheme.
        await page.evaluate((t) => document.documentElement.setAttribute('data-theme', t), theme);
        await page.addStyleTag({ content: FREEZE_CSS });

        // State scripts drive both modes for a shared-file golden (inventory) and the live
        // candidate. A per-state golden (--pagedir: error, profile) is already pre-rendered in
        // its end state, so its script — which may submit a form and hit the server (profile
        // "minted") — must NOT run against the static file. Run everywhere EXCEPT golden+pagedir.
        // A per-state golden (--pagedir) is normally pre-rendered in its end state, so its
        // script must NOT run against the static file — EXCEPT a pure client-side interaction
        // script with no settle delay (signals' menu-open: opening a kebab), which is safe to
        // run on the static golden and IS how that state reaches its captured form on both
        // sides. A navigating script (profile "minted", scope "refusal") declares a `delay`, so
        // it stays skipped on the pagedir golden.
        const runStateJs = st.js && st.js.trim() && !(mode === 'golden' && pageDir && st.delay);
        if (runStateJs) {
          try {
            await page.evaluate((js) => {
              // eslint-disable-next-line no-new-func
              new Function(js)();
            }, st.js);
          } catch (e) {
            // A form-submitting script (profile "minted") can tear down the execution context
            // as it navigates; that is expected and handled by the settle + re-assert below.
            // Only a script with no declared settle delay should surface the error.
            if (!st.delay) throw e;
          }
          // A script may navigate (profile "minted" POSTs the create form; the handler renders
          // the minted dialog directly). Wait its declared settle, then re-assert theme + freeze
          // on the new document — a navigation drops [data-theme] and the injected freeze style.
          if (st.delay) {
            await page.waitForLoadState('networkidle').catch(() => {});
            await page.waitForTimeout(st.delay);
            await page.evaluate((t) => document.documentElement.setAttribute('data-theme', t), theme);
            await page.addStyleTag({ content: FREEZE_CSS });
          }
        }
        // Drop the console chrome from flow (candidate only) so <main> sits at the viewport top,
        // aligning the fixed dialog overlay with the chrome-less golden. A no-op when no chrome
        // is present. Done before the settle waits so the reflow paints before the shot.
        if (mode === 'candidate' && hideChrome) {
          await page.evaluate(() => {
            document.querySelectorAll('header.topnav, .topnav, footer.appfooter, .appfooter').forEach((el) => { el.style.display = 'none'; });
            // The app's pageCSS pins body{min-height:100vh}, so a `body`-crop candidate (signals'
            // drawer / descope overlays) would box to the full viewport while the chrome-less golden
            // body shrink-wraps to <main>. The scrim + drawer are position:fixed (viewport-relative),
            // so clipping the body to content height is symmetric on both sides — neutralize the
            // min-height so the candidate body box matches the golden's. A no-op for `main`-crop
            // screens (they clip the <main> element, not the body).
            document.body.style.minHeight = '0';
          });
        }
        // Wait for webfonts to finish loading before snapshotting. Both the golden
        // (leading @import in its own <style>) and the candidate (pageCSS @import)
        // load Instrument Sans / Geist Mono with display=swap, so a snapshot taken
        // before fonts.ready would capture the fallback face and diverge. Bounded so
        // an unreachable CDN degrades to a deterministic fallback on BOTH sides.
        await page.evaluate(() => Promise.race([document.fonts.ready, new Promise((r) => setTimeout(r, 4000))]));
        // Settle: two rAFs so the state's DOM mutations + any font swap paint.
        await page.evaluate(() => new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r))));
        await page.waitForTimeout(120);

        // Per-state crop overrides the screen-level crop: signals crops `main` for the table
        // states but `body` for the drawer / descope overlays that escape `main` (the fixed
        // scrim + drawer are painted on <body>, outside the <main> box).
        const cropSel = st.crop || screenStates.crop || 'main';
        const wantFull = fullPage || cropSel === 'full-page';
        const buf = wantFull
          ? await page.screenshot({ fullPage: true })
          : await page.locator(cropSel).first().screenshot();
        const png = PNG.sync.read(buf);

        const gpath = goldenPathFor(st.id, vp.id, theme);
        if (writeGoldens) {
          mkdirSync(dirname(gpath), { recursive: true });
          writeFileSync(gpath, PNG.sync.write(png));
          results.push({ state: st.id, vp: vp.id, theme, wrote: gpath, w: png.width, h: png.height });
          console.log(`WROTE  ${screen}/${st.id}-${vp.id}-${theme}  ${png.width}x${png.height}`);
        } else if (!existsSync(gpath)) {
          if (config.skipMissingGoldens) {
            results.push({ state: st.id, vp: vp.id, theme, skipped: true });
            console.log(`SKIP   ${screen}/${st.id}-${vp.id}-${theme}  (no golden)`);
          } else {
            results.push({ state: st.id, vp: vp.id, theme, missing: true });
            console.log(`MISS   ${screen}/${st.id}-${vp.id}-${theme}  (no golden)`);
          }
        } else {
          const golden = PNG.sync.read(readFileSync(gpath));
          const d = diffPercent(golden, png);
          if (d.mismatch) {
            results.push({ state: st.id, vp: vp.id, theme, dimMismatch: d });
            worst = Infinity;
            console.log(`DIMDIFF ${screen}/${st.id}-${vp.id}-${theme}  golden ${d.aw}x${d.ah} vs cand ${d.bw}x${d.bh}`);
          } else {
            worst = Math.max(worst, d.pct);
            results.push({ state: st.id, vp: vp.id, theme, pct: d.pct, diffPixels: d.diffPixels, total: d.total, w: png.width, h: png.height });
            const flag = d.pct > config.thresholdPercent ? ' OVER' : '';
            console.log(`DIFF   ${screen}/${st.id}-${vp.id}-${theme}  ${d.pct.toFixed(3)}%  (${d.diffPixels}/${d.total})${flag}`);
            // Emit the diff image beside the golden for eyeballing large misses.
            if (d.pct > config.thresholdPercent && d.diff) {
              const dp = gpath.replace(/\.png$/, '.diff.png');
              try { writeFileSync(dp, PNG.sync.write(d.diff)); } catch {}
            }
          }
        }
        await page.close();
      }
      await context.close();
    }
  }

  await browser.close();

  console.log('\n=== summary (' + mode + ') ===');
  for (const r of results) {
    if (r.wrote) console.log(`  ${r.state}-${r.vp}-${r.theme}: wrote ${r.w}x${r.h}`);
    else if (r.skipped) console.log(`  ${r.state}-${r.vp}-${r.theme}: skipped (no golden)`);
    else if (r.dimMismatch) console.log(`  ${r.state}-${r.vp}-${r.theme}: DIMENSION MISMATCH`);
    else if (typeof r.pct === 'number') console.log(`  ${r.state}-${r.vp}-${r.theme}: ${r.pct.toFixed(3)}%${r.pct > config.thresholdPercent ? '  OVER THRESHOLD' : ''}`);
  }

  // Runtime report goes beside the harness (repo-owned, gitignored), never into
  // goldens/ which becomes design-owned (CI gate G1).
  const jsonOut = join(__dirname, `${screen}-${mode}-report.json`);
  try { writeFileSync(jsonOut, JSON.stringify(results, null, 2)); } catch {}

  if (writeGoldens || advisory) {
    process.exit(0);
  }
  process.exit(worst > config.thresholdPercent ? 1 : 0);
}

run().catch((e) => { console.error(e); process.exit(2); });
