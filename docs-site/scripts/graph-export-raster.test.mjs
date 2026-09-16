// The Go lane cannot execute template JavaScript, so no Go test reaches this raster (#2212).
import { test, before, after } from "node:test";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";

const SCRIPT_DIR = dirname(fileURLToPath(import.meta.url));
const TMPL = resolve(SCRIPT_DIR, "..", "..", "design-system", "templates", "graph.tmpl");

const NOTE_RGB = [255, 0, 0];
const HIDDEN_RGB = [0, 0, 255];
const SHOWN_RGB = [0, 200, 0];
const PROBE = "HHHHHHHH";

// Under about 2000 units the export's MAX_AREA clamp does not bind and #2186 is unreachable.
const CLAMPED_SIDE = 2400;
// The band moves the scale only in proportion to the height it adds, so the filler runs
// long enough to make the band dominate the frame.
const FILLER_WORDS = 2600;

async function exportSource() {
  const src = await readFile(TMPL, "utf8");
  const a = src.indexOf("  function exportState() {");
  const b = src.lastIndexOf("  apply();");
  assert.ok(a >= 0 && b > a, `${TMPL} no longer holds the export block this gate lifts`);
  const slice = src.slice(a, b);
  assert.ok(
    !slice.includes("{{"),
    "the graph export now interpolates a Go template action, so it can no longer be lifted and run on its own",
  );
  assert.ok(slice.includes("function layoutNotes("), "the lifted block holds no layoutNotes");
  assert.ok(slice.includes('getElementById("gr-export")'), "the lifted block binds no export button");
  return slice;
}

function fixture({ side, controls, notes }) {
  const inner = side - 48; // the export pads the measured bbox by 24 on each side
  const callouts = notes
    .map((t) => `<div class="gr-callout"><span class="tx">${t}</span></div>`)
    .join("");
  const ctl = controls
    ? `<button id="gr-scope-btn"><span class="cv">acme.example</span></button>
       <button id="gr-sev-btn"><span class="cv">critical</span></button>`
    : "";
  return `<!doctype html><html><head><meta charset="utf-8"><style>
  html,body{margin:0;background:rgb(255,255,255)}
  /* The stack design-system/tokens/typography.css gives --font-ui, so the band is
     measured against a multi-word family the way the page's own callouts are. */
  .gr-callout .tx{font:400 13px/1.55 "Instrument Sans","Helvetica Neue",Arial,sans-serif;color:rgb(${NOTE_RGB});}
  </style></head><body>
  ${ctl}
  <div class="gr-main">${callouts}</div>
  <!-- The live root carries this inline, and the export strips it, so the fixture carries it too. -->
  <svg id="gr-svg" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ${side} ${side}"
       style="display:block;width:100%;height:560px;cursor:grab;touch-action:none">
    <g id="gr-viewport" transform="translate(0,0) scale(1)">
      <rect x="24" y="24" width="${inner}" height="${inner}" fill="rgb(235,235,235)"></rect>
      <circle cx="${side / 2}" cy="${side / 2}" r="${Math.round(side / 12)}" fill="rgb(${SHOWN_RGB})"></circle>
      <circle cx="${side / 3}" cy="${side / 2}" r="${Math.round(side / 12)}" fill="rgb(${HIDDEN_RGB})" style="display:none"></circle>
    </g>
  </svg>
  <button id="gr-export">Export</button>
  </body></html>`;
}

async function render(browser, source, opts) {
  const page = await browser.newPage({ viewport: { width: 900, height: 700 } });
  const errors = [];
  page.on("console", (m) => {
    if (m.type() === "error") errors.push(m.text());
  });
  page.on("pageerror", (e) => errors.push(String(e)));
  try {
    await page.setContent(fixture(opts), { waitUntil: "load" });
    const out = await page.evaluate(
      async ({ source, note, hidden, shown, probe }) => {
        // A glyph's antialiased edge and a halo's are both part-background, so only a
        // near-exact match counts and the two ink boxes are measured the same way.
        const near = (r, g, b, c) =>
          Math.abs(r - c[0]) <= 24 && Math.abs(g - c[1]) <= 24 && Math.abs(b - c[2]) <= 24;

        async function decode(href) {
          const img = new Image();
          await new Promise((res, rej) => {
            img.onload = res;
            img.onerror = () => rej(new Error("the exported data URL did not decode as an image"));
            img.src = href;
          });
          return img;
        }

        // A 16-megapixel frame is read in strips: one whole-image ImageData would be a
        // 64MB allocation for a count of three colours.
        function scan(img) {
          const w = img.naturalWidth, h = img.naturalHeight, STRIP = 256;
          const cv = document.createElement("canvas");
          cv.width = w; cv.height = STRIP;
          const ctx = cv.getContext("2d", { willReadFrequently: true });
          const rows = new Array(h).fill(0);
          let hiddenPx = 0, shownPx = 0;
          for (let y0 = 0; y0 < h; y0 += STRIP) {
            const sh = Math.min(STRIP, h - y0);
            ctx.clearRect(0, 0, w, STRIP);
            ctx.drawImage(img, 0, y0, w, sh, 0, 0, w, sh);
            const d = ctx.getImageData(0, 0, w, sh).data;
            for (let y = 0; y < sh; y++) {
              let ink = 0;
              for (let x = 0; x < w; x++) {
                const i = (y * w + x) * 4, r = d[i], g = d[i + 1], b = d[i + 2];
                if (near(r, g, b, note)) ink++;
                else if (near(r, g, b, hidden)) hiddenPx++;
                else if (near(r, g, b, shown)) shownPx++;
              }
              rows[y0 + y] = ink;
            }
          }
          return { w, h, rows, hiddenPx, shownPx };
        }

        // A short line of thin type can leave a row whose every red pixel is antialiased
        // past the match, which would read as two lines, so a sub-leading gap is closed.
        const MERGE_ROWS = 4;
        function runs(rows) {
          const out = [];
          let start = -1;
          for (let y = 0; y < rows.length; y++) {
            if (rows[y] > 0 && start < 0) start = y;
            else if (rows[y] === 0 && start >= 0) { out.push({ top: start, height: y - start }); start = -1; }
          }
          if (start >= 0) out.push({ top: start, height: rows.length - start });
          const merged = [];
          for (const r of out) {
            const last = merged[merged.length - 1];
            if (last && r.top - (last.top + last.height) <= MERGE_ROWS) last.height = r.top + r.height - last.top;
            else merged.push({ ...r });
          }
          return merged;
        }

        const tx = document.querySelector(".gr-main .gr-callout .tx");
        const cs = getComputedStyle(tx);
        const px = parseFloat(cs.fontSize);
        const leading = parseFloat(cs.lineHeight);

        // The reference runs the same rasterizer at the size the band was MEASURED for,
        // so the assertion is a comparison and not a font-metric constant.
        const NS = "http://www.w3.org/2000/svg";
        const refRoot = document.createElementNS(NS, "svg");
        refRoot.setAttribute("xmlns", NS);
        refRoot.setAttribute("width", "300");
        refRoot.setAttribute("height", "60");
        refRoot.setAttribute("viewBox", "0 0 300 60");
        const refText = document.createElementNS(NS, "text");
        refText.setAttribute("x", "10");
        refText.setAttribute("y", "40");
        // A multi-word family comes back from getComputedStyle quoted, and those quotes
        // would close a hand-built style attribute, so the reference is built through DOM.
        refText.style.fontFamily = cs.fontFamily;
        refText.style.fontSize = px + "px";
        refText.style.fontWeight = cs.fontWeight;
        refText.style.fill = cs.color;
        refText.textContent = probe;
        refRoot.appendChild(refText);
        const refSvg = new XMLSerializer().serializeToString(refRoot);
        const refImg = await decode("data:image/svg+xml;charset=utf-8," + encodeURIComponent(refSvg));
        const refCv = document.createElement("canvas");
        refCv.width = 300; refCv.height = 60;
        const refCtx = refCv.getContext("2d", { willReadFrequently: true });
        refCtx.fillStyle = "#ffffff"; refCtx.fillRect(0, 0, 300, 60);
        refCtx.drawImage(refImg, 0, 0);
        const refRows = new Array(60).fill(0);
        {
          const d = refCtx.getImageData(0, 0, 300, 60).data;
          for (let y = 0; y < 60; y++)
            for (let x = 0; x < 300; x++) {
              const i = (y * 300 + x) * 4;
              if (near(d[i], d[i + 1], d[i + 2], note)) refRows[y]++;
            }
        }
        const refRuns = runs(refRows);

        const done = new Promise((res, rej) => {
          const orig = HTMLAnchorElement.prototype.click;
          HTMLAnchorElement.prototype.click = function () {
            if (this.download) { res(this.href); return; }
            return orig.apply(this, arguments);
          };
          setTimeout(() => rej(new Error("the export produced no download")), 60000);
        });
        new Function("svg", "vp", source)(
          document.getElementById("gr-svg"),
          document.getElementById("gr-viewport"),
        );
        document.getElementById("gr-export").click();
        const href = await done;

        const img = await decode(href);
        const s = scan(img);
        return {
          px,
          leading,
          fontFamily: cs.fontFamily,
          isPng: href.startsWith("data:image/png;base64,"),
          width: s.w,
          height: s.h,
          hiddenPx: s.hiddenPx,
          shownPx: s.shownPx,
          runs: runs(s.rows),
          refRuns,
        };
      },
      { source, note: NOTE_RGB, hidden: HIDDEN_RGB, shown: SHOWN_RGB, probe: PROBE },
    );
    // A console event raised during the export can land after evaluate resolves, and
    // page.close() below would drop it before the caller ever asserts on it.
    await page.waitForTimeout(50);
    out.errors = errors;
    return out;
  } finally {
    await page.close();
  }
}

const FILLER = Array.from({ length: FILLER_WORDS }, (_, i) => "NOTE" + ((i % 7) + 2)).join(" ");

function mutated(source, [anchor, replacement]) {
  // A gate that only ever runs against a fixed template cannot show it catches the
  // defect it exists for, so each defect is put back and measured again.
  assert.ok(
    source.includes(anchor),
    `the export no longer holds ${JSON.stringify(anchor)}, so this gate can no longer prove it catches the defect that line fixed`,
  );
  return source.replace(anchor, replacement);
}

// The pre-fix export recomputed the scale after laying the band out at the old one (#2186).
const ONE_SHOT = ["if (!(next < os)) break;", "os = next; break;"];
// The pre-fix sweep rebuilt the style attribute from PROPS, which omits display (#2152).
const NO_DISPLAY = ['if (cs.display === "none") decl += "display:none;";', ";"];

function pitch(r) {
  // A glyph's own 9px ink box cannot resolve a scale error, but every band line sits one
  // leading from the last, so the span across a note carries that error times its lines.
  const lines = r.runs.length - 1;
  assert.ok(lines >= 2, `the export banded ${r.runs.length} ink run(s), too few to measure a line pitch across`);
  return { lines, span: r.runs[lines - 1].top - r.runs[0].top, want: (lines - 1) * r.leading };
}
const PITCH_TOL = 3; // the ink top of a line rounds independently at each end of the span

let browser, source;
before(async () => {
  source = await exportSource();
  // The gate runs inside the pinned Playwright container, where chromium's sandbox is unavailable.
  browser = await chromium.launch({ headless: true, args: ["--no-sandbox"] });
}, { timeout: 180000 });
after(async () => { if (browser) await browser.close(); });

test("#2212/#2186 the exported note band rasterizes at the size it was measured for", { timeout: 180000 }, async () => {
  const r = await render(browser, source, { side: CLAMPED_SIDE, controls: false, notes: [FILLER, PROBE] });
  assert.deepEqual(r.errors, []);
  assert.ok(r.isPng, "the export produced no PNG data URL");
  assert.ok(
    r.width * r.height > 15e6,
    `the fixture did not bind the export's MAX_AREA clamp (${r.width}x${r.height}), so it cannot reach #2186`,
  );
  assert.equal(r.refRuns.length, 1, "the reference render produced no single glyph run to measure against");
  // A single-word family would leave the quoting the page's own --font-ui stack carries
  // untested, and a quote in a hand-built style attribute closes it.
  assert.ok(
    r.fontFamily.includes('"'),
    `the fixture measured the band against ${r.fontFamily}, which carries no quoted family`,
  );
  assert.ok(r.runs.length > 100, `the note band wrapped to only ${r.runs.length} lines, too few to move the scale`);

  const p = pitch(r);
  const probe = r.runs[r.runs.length - 1];
  const ref = r.refRuns[0];
  console.log(
    `frame ${r.width}x${r.height}, ${p.lines} band lines across ${p.span}px (want ${p.want.toFixed(1)}px), ` +
      `glyph ${probe.height}px (reference ${ref.height}px)`,
  );
  assert.ok(
    Math.abs(p.span - p.want) <= PITCH_TOL,
    `the exported band set ${p.lines} lines across ${p.span}px where its own ${r.leading}px leading wants ` +
      `${p.want.toFixed(1)}px. The band is laid out at a scale the output does not rasterize at (#2186).`,
  );
  assert.ok(
    Math.abs(probe.height - ref.height) <= 1,
    `the exported note rasterized ${probe.height}px tall against the ${ref.height}px the same ${r.px}px type ` +
      `rasterizes at (#2186).`,
  );
});

test("#2212 the band measurement moves when the export stops settling the scale", { timeout: 180000 }, async () => {
  const r = await render(browser, mutated(source, ONE_SHOT), {
    side: CLAMPED_SIDE,
    controls: false,
    notes: [FILLER, PROBE],
  });
  // Without the same guards the positive gate carries, any unrelated breakage of the
  // mutated source would read as a successful kill and prove that gate sound.
  assert.ok(r.isPng, "the one-shot export produced no PNG data URL");
  assert.ok(
    r.width * r.height > 15e6,
    `the one-shot fixture did not bind the export's MAX_AREA clamp (${r.width}x${r.height})`,
  );
  assert.ok(r.runs.length > 100, `the one-shot band wrapped to only ${r.runs.length} lines`);

  const p = pitch(r);
  console.log(`one-shot frame ${r.width}x${r.height}, ${p.lines} band lines across ${p.span}px (want ${p.want.toFixed(1)}px)`);
  assert.ok(
    Math.abs(p.span - p.want) > PITCH_TOL,
    `a band laid out once against the pre-band scale still set ${p.lines} lines across ${p.span}px against the ` +
      `${p.want.toFixed(1)}px wanted, so this fixture no longer moves the scale and the gate above proves nothing`,
  );
});

test("#2212 the halo count moves when the export stops restating a hidden element", { timeout: 180000 }, async () => {
  const r = await render(browser, mutated(source, NO_DISPLAY), { side: 600, controls: true, notes: [PROBE] });
  assert.ok(
    r.hiddenPx > 0,
    "the export drew no hidden halo even with the display restatement removed, so the gate below proves nothing",
  );
});

test("#2212/#2152 the export omits a halo the severity filter hid", { timeout: 180000 }, async () => {
  const r = await render(browser, source, { side: 600, controls: true, notes: [PROBE] });
  assert.deepEqual(r.errors, []);
  assert.ok(r.isPng, "the export produced no PNG data URL");
  console.log(`frame ${r.width}x${r.height}, halo pixels drawn ${r.shownPx}, hidden ${r.hiddenPx}`);
  assert.ok(r.shownPx > 100, `the export drew no visible halo (${r.shownPx}px), so this test proves nothing`);
  assert.equal(
    r.hiddenPx,
    0,
    `the export drew ${r.hiddenPx}px of a halo the severity filter hid, so the PNG states more than the page did (#2152)`,
  );
});

test("#2212/#2152 the export bands a caption line for the controls the page holds", { timeout: 180000 }, async () => {
  const withCtl = await render(browser, source, { side: 600, controls: true, notes: [PROBE] });
  const without = await render(browser, source, { side: 600, controls: false, notes: [PROBE] });
  assert.deepEqual(withCtl.errors, []);
  assert.deepEqual(without.errors, []);
  assert.equal(without.runs.length, 1, `the control-free export banded ${without.runs.length} lines, not the one callout`);
  assert.equal(
    withCtl.runs.length,
    without.runs.length + 1,
    `a page holding a scope and a severity control banded ${withCtl.runs.length} lines against ` +
      `${without.runs.length} without them, so the exported image does not name the filter it was taken under (#2152)`,
  );
});
