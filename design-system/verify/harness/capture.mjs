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
        const page = await context.newPage();
        if (mode === 'golden') {
          // Per-state file (--pagedir, error) or a single shared file (--page, inventory).
          const gp = pageDir ? join(pageDir, `${st.id}.html`) : pagePath;
          await page.goto(pathToFileURL(resolve(gp)).href, { waitUntil: 'networkidle' });
        } else {
          // Establish this state's session first (error), then navigate its own route
          // (states.json's per-state route) or the screen-level route (inventory).
          if (st.session) await mintSession(context, st.session);
          const route = st.route || screenStates.route;
          await page.goto(`${baseURL}${route}`, { waitUntil: 'networkidle' });
        }

        // Force the theme deterministically in BOTH modes (design colors.css
        // honours [data-theme]); independent of prefers-color-scheme.
        await page.evaluate((t) => document.documentElement.setAttribute('data-theme', t), theme);
        await page.addStyleTag({ content: FREEZE_CSS });

        if (st.js && st.js.trim()) {
          await page.evaluate((js) => {
            // eslint-disable-next-line no-new-func
            new Function(js)();
          }, st.js);
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

        const cropSel = screenStates.crop || 'main';
        const buf = await page.locator(cropSel).first().screenshot();
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
