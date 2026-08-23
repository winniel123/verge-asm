# Deploying the docs site (T5 / #355)

The CI lane `.github/workflows/docs-site.yml` builds the multi-version Astro site,
runs two required gates, and deploys to **GitHub Pages**. This file records the
deploy-target decision (map #348 Fog: "Deploy target + redirect layer"), the one
manual Settings step CI cannot perform headless, and how the `/` → latest redirect
works.

## Deploy target: GitHub Pages

**Decision — GitHub Pages, via `actions/upload-pages-artifact` + `actions/deploy-pages`.**

Rationale:

- Same repo, free, no external account or secret to provision.
- First-class support for the full-history checkout the build needs (`fetch-depth: 0`
  so `source-resolution.ts` can enumerate every `v*` tag — a shallow clone silently
  builds a single version).
- Native artifact → deploy actions; no third-party token in `secrets`.
- Deploys are gated behind the build job, so a broken link or design drift blocks the
  publish.

No strong blocker was found that would push us to Netlify/Cloudflare/Vercel. One
serving constraint (below) applies to any host equally.

## Serving constraint: the site must be served at the domain ROOT

The site emitted by T0–T4 uses **root-absolute internal links** throughout —
`/latest/<slug>`, `/main/<slug>`, `/search/<version>.json`, `/fonts/*`, the canonical
`/latest/<slug>`, and client-side `window.location.pathname` parsing in `TopNav.jsx`
that reads the version out of `pathname.split("/")[0]`. Nothing threads an Astro
`base` prefix. The site therefore assumes it is served from `/`.

Consequence for GitHub Pages:

- A **project** Pages site serves under `https://<owner>.github.io/verge-asm/`. Under
  that subpath every root-absolute link (`/latest/...`) 404s, and TopNav parses the
  wrong path segment as the version. The site would be broken.
- To serve at root, use **one** of:
  1. A **custom domain** (e.g. `docs.verge-asm.example`) — add a `CNAME` and point DNS
     at Pages. This is the recommended path.
  2. Publish from a user/org **`<owner>.github.io`** repository (served at root).

Making the site base-path-agnostic instead would mean threading a `base` prefix
through every link-emitting stage (`nav-build.ts`, `render.jsx`, `source-resolution.ts`
`canonicalPath`, `search-index.ts`, the `[slug].astro` breadcrumb) **and** the client
runtime in `TopNav.jsx` — i.e. re-opening five prior tickets' work and the map's
"consume, don't re-author" principle. That is out of scope for T5 and is recorded as a
follow-up option, not done here.

## Manual step CI cannot do (HANDOFF)

CI cannot change repository Settings. Before the first deploy, a maintainer must:

1. **Settings → Pages → Build and deployment → Source = "GitHub Actions"** (one-time).
   Without this, `actions/deploy-pages` has no Pages site to publish to and the deploy
   job fails.
2. **Serve at root** (see constraint above): set a custom domain under Settings → Pages
   (adds a `CNAME`), or publish from the `<owner>.github.io` repo.
3. *(Optional)* Make the `docs-site / build` job a **required status check** on PRs
   (Settings → Branches → branch protection). Note: the PR trigger is path-filtered, so
   on a PR that touches no docs/design/site files the check does not run; a required
   check that never runs can leave a PR "pending". Either accept that, or mark the check
   "required only when it runs" per your branch-protection tooling.

## The `/` → /latest/ redirect

`src/pages/index.astro` builds a static `dist/index.html` that redirects `/` to the
latest version's first guide (there is no bare `/<version>/` route, so we target the
first guide chosen from the built nav order — currently `/latest/using`). It carries:

- `<meta http-equiv="refresh" content="0; url=/latest/using">` — no-JS redirect,
- `<script>location.replace("/latest/using")</script>` — instant JS redirect,
- `<a href="/latest/using">` — a clickable fallback,
- `<link rel="canonical" href="/latest/using">` — crawlers index the guide, not the stub.

The target slug is computed at build time via `resolveSources("latest")` + `buildNav`,
so it always points at a real, ordered landing guide. Tv's real `/latest/*` alias tree
is kept as-is; this is the single `/` hop on top of it.

## Refreshing the screenshot baseline

The screenshot gate (`npm run check:screenshot`) diffs a built docs page against
`docs-site/tests/baseline/docs.png`. When a design change intentionally alters layout,
regenerate and commit the baseline:

```
cd docs-site
npm run build
npm run screenshot:update   # rewrites tests/baseline/docs.png
```

Commit the updated PNG (it is tracked; `*.png binary` in `.gitattributes` keeps it from
line-ending churn). Diffs from a failed run land in `docs-site/tests/diff/` (gitignored).
