# ADR-0115: the docs site renders the guides in place, and a version is a git ref — never a copied snapshot

- **Status:** Accepted
- **Date:** 2026-08-23
- **Ticket:** [#349 T0 — Scaffold docs-site + port DocsPage layout + ADR](https://github.com/winniel123/verge-asm/issues/349) (map: [#348](https://github.com/winniel123/verge-asm/issues/348))

## Context

The repo already carries eight authoritative operator guides as Markdown in `docs/guides/`
(`first-run`, `prober`, `running`, `signals`, `sources`, `using`, `verifying`, `zone-files`), and
the design system ships a finished **docs surface** — `design-system/examples/DocsPage.jsx` plus its
ground-truth capture `design-system/screenshots/docs.jpg` — as the verbatim IA spec for how docs must
look ([ADR-0110](./0110-the-design-system-examples-are-the-consoles-ia-spec-ported-verbatim.md)). What
did not exist was the site that renders those guides inside that shell, and a decision about how the
docs are *versioned* alongside a product that ships tagged releases.

Two questions had to be settled before any later ticket (nav generation, search, per-guide pages) could
begin, because both are structural and expensive to reverse once content and URLs exist:

1. **Where does the rendered content come from** — the guide Markdown in the tree, or a copy the site
   keeps of its own?
2. **What is a documentation "version"** — a per-release folder of frozen files, or something derived
   from git itself?

A third, smaller question — the static-site stack — follows once those two are fixed.

The design-system constraint frames the build: ~~components are authored in Claude Design and imported,
**never re-authored in-repo**~~ ([ADR-0109](./0109-design-system-components-are-authored-in-claude-design-and-imported.md)).
The docs site therefore has to *consume* `design-system/` (tokens + components) directly rather than
reproduce any of it, which rules stacks in or out by how cleanly they can render the existing React
components and flip the token themes.

> **The struck clause is WITHDRAWN at the site that specifies it, 2026-08-28 by `55aa367` /
> [#1410](https://github.com/winniel123/verge-asm/issues/1410)
> ([ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)).**
> `design-system/` may be edited in the repo. What frames this ADR's stack choice is the
> **consume-rather-than-reproduce** half of the sentence, which stands on its own ground: a second
> copy of the design system inside a site framework is a second source of truth. The stack decision
> is unaffected.

## Decision

> **The docs site renders `docs/guides/*.md` in place — it never copies them — and a documentation
> version is a git ref, not a snapshot folder. Release tags `v*` publish under `/<version>/<slug>`;
> `main` publishes as `dev`; `latest` aliases the highest `v*` tag; `/` and `/latest/*` redirect
> there. Past versions are immutable. The site is built with Astro, consuming the design system
> directly.** This ticket (T0) scaffolds `docs-site/` and ports the `DocsPage` shell as the frame every
> later ticket fills; the principles below govern what fills it.

> **`latest` aliases the highest `v*` tag** is BOUNDED at this site, 2026-09-05, by
> [ADR-0155](./0155-the-docs-site-does-not-enforce-the-tag-policy-so-a-prerelease-tag-is-browsable-and-never-becomes-latest-or-current.md)
> ([ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)). Read it
> as **the highest stable `v*` tag**. See §2's note.

### 1. Single-source rendering

The guides in `docs/guides/` are the **single source**. The site reads that Markdown and renders it
through the ported shell. It does not keep a second, transformed copy of any guide under `docs-site/`.
A guide is edited in exactly one place — its `.md` file — and the site re-renders it. There is no
"import the docs into the site" step, because a copy is a second source of truth and a second source of
truth is drift waiting to happen.

The T0 placeholder page (`docs-site/src/pages/index.astro` + the `Article` island) is sample prose only,
present to prove the shell frames real content in both themes. It is explicitly **not** a copied guide.
The ticket that wires Markdown rendering replaces that body with the rendered guide, keeping the guide
file as the origin.

### 2. The git-ref version model

A documentation **version is a git ref**, and the published tree is derived from refs, not maintained as
hand-cut folders:

- **Release tags `v*`** each publish the guides *as they stood at that tag*, under `/<version>/<slug>`
  (e.g. `/v0.9.2/quick-start`).
- **`main`** publishes as **`dev`** — the in-progress docs.
- **`latest`** is an **alias** to the highest `v*` tag by semver, not a copy of it. *(Bounded. See
  the note below.)*
- **`/` and `/latest/*` redirect** to the resolved highest-tag version's paths, so a bare or
  `latest`-prefixed URL always lands on the current release's docs. *(Bounded. See the note
  below.)*
- **Past versions are immutable.** A shipped version's docs are the guides at that tag. You do not edit
  the published `/v0.9.1/*` tree in place. Fixing a shipped version's docs is a **maintenance branch +
  patch re-tag** (the same discipline the product's releases already follow), which re-derives that
  version from the corrected ref.

> **The two bulleted clauses above are BOUNDED at this site, 2026-09-05, by
> [ADR-0155](./0155-the-docs-site-does-not-enforce-the-tag-policy-so-a-prerelease-tag-is-browsable-and-never-becomes-latest-or-current.md)
> ([ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)).**
> `latest` aliases the highest **stable** `v*` tag, and `/` and `/latest/*` redirect there. Semver
> orders a prerelease above the previous release, so the plain reading would make `v1.0.1-rc1` the
> alias target over `v1.0.0`. A prerelease tag still publishes its own `/<tag>/*` tree, and it never
> carries the `current` badge. Where no stable tag exists, `latest` falls back to `main`. The rest
> of this section is unaffected.

**Why git-ref sourcing rather than snapshot folders.** The rejected alternative is to keep, in the repo
or the build, a `versions/v0.9.1/`, `versions/v0.9.2/` … set of frozen folders — copies of the guides
per release. That reintroduces exactly the drift single-source rendering exists to prevent: a snapshot is
a copy, copies get edited independently of their origin, and the published history stops being a faithful
record of what shipped. Git already stores an immutable, content-addressed snapshot at every tag. A
version folder is a worse copy of information the tag already holds perfectly. Deriving each version from
its ref keeps "the v0.9.1 docs" defined as "the guides at `v0.9.1`," with no editable duplicate able to
diverge from it.

### 3. Stack choice — Astro

The site is built with **Astro** (static output), consuming the design system directly: it imports
`design-system/styles.css`/`tokens/*` and renders the `design-system/components/**` React components as
server-rendered HTML, hydrating only the interactive islands (top-nav search + `VersionSelect`, and the
prose interactions). Astro was chosen over the two obvious alternatives:

- **vs. Docusaurus** — Docusaurus brings its own theme, its own component set, and its own MDX/React
  runtime. Adopting it would mean reproducing the Verge design system inside Docusaurus's theming layer,
  which collides head-on with ADR-0109 (do not re-author the components) and ADR-0110 (port the
  `DocsPage` shell verbatim). Astro imposes no theme of its own, so the ported shell *is* the site.
- **vs. Next.js** — Next is a full application framework (server runtime, routing model, data layer) for
  a surface that is fundamentally static content. Astro's islands model ships zero JavaScript for the
  static parts (nav rail, TOC, prose) and hydrates only the few interactive pieces, which fits a docs
  site's cost profile better and keeps the output a plain static bundle with no server to run.

Astro also renders existing React components with no rewrite (via `@astrojs/react`) and flips the design
system's `data-theme="dark"` tokens with no framework theme context, so light + dark come for free from
the tokens.

### Integration seams this ticket fixes

So later tickets swap **data, not markup**, T0 leaves the shell slots prop/slot-driven with placeholder
data:

- **Icons** route through the design system's single `Icon` wrapper, swapped to **`lucide-react`** at
  build time (a Vite `resolveId` redirect of `components/media/Icon.jsx` to a local lucide-react
  wrapper), per the DS README's "one-file swap." No design-system file is edited.
- **Fonts are self-hosted.** The production build does **not** use the Google Fonts `@import` in
  `tokens/typography.css`. Instrument Sans + Geist Mono `woff2` (variable, latin + latin-ext) are served
  from the site origin, and `typography.css`'s token values are mirrored locally minus that `@import`.
- **Shell slots** — the left section-nav rail (`sections`), the right on-page TOC (`toc`), the
  `VersionSelect` version list, and the search box — are separated components taking placeholder props,
  so the nav-generation, per-page-TOC, and search/version tickets replace the data behind them without
  touching the ported layout.

## Consequences

- **A guide has exactly one home.** Editing `docs/guides/signals.md` updates the site. There is no second
  copy to keep in sync, and no "sync the docs" chore. The cost is that the rendering ticket must read the
  guides from `docs/guides/` at build time rather than from a folder inside `docs-site/`.
- **Version history is defined by tags, and stays honest.** Publishing a new release is tagging it.
  `latest` re-resolves to the new highest tag and the root redirect follows, with no folder to cut. Fixing
  old docs is a patch re-tag, not an in-place edit — which is more ceremony than editing a folder, and
  deliberately so, because it keeps every published version a faithful snapshot of what shipped.
- **The build depends on the repo-root `design-system/`.** `docs-site/` imports tokens and components from
  `../design-system` via a `@ds` alias. It is coupled to that directory's layout but authors none of it,
  ~~honouring ADR-0109~~. If the design system moves, the alias moves with it — one config line.

  > **`honouring ADR-0109` is WITHDRAWN at the site that specifies it, 2026-08-28 by `55aa367` /
  > [#1410](https://github.com/winniel123/verge-asm/issues/1410)
  > ([ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)).**
  > `design-system/` may be edited in the repo, so authoring none of it is no longer owed to a rule.
  > The fact stands as a fact: `docs-site/` consumes the directory and writes nothing into it.
- **The stack is a static bundle with no server.** Astro emits static HTML/CSS/JS. There is nothing to run
  in production beyond a static host. Interactivity is confined to hydrated islands, so the static content
  ships no framework runtime.
- **Committing to Astro is a reversible-but-load-bearing choice.** Because the site re-implements no design
  system of its own and the content is plain Markdown + the ported shell, a future move off Astro would
  re-port the shell and the Markdown pipeline, not rebuild a theme — but every later docs ticket is written
  against Astro's islands + content model, so a switch reopens this ADR.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| Copy/import the guides into `docs-site/` and render the copies | A copy is a second source of truth; it drifts from `docs/guides/`. Single-source rendering exists to prevent exactly this |
| Per-release snapshot folders (`versions/v0.9.1/…`) instead of git refs | Snapshots are copies and reintroduce drift; git already stores an immutable, content-addressed snapshot at every `v*` tag, so a folder is a worse copy of what the tag holds |
| Docusaurus | Brings its own theme + component set + runtime; reproducing the Verge design system inside it collides with ADR-0109 (don't re-author components) and ADR-0110 (port `DocsPage` verbatim) |
| Next.js | A full app framework (server, routing, data layer) for fundamentally static content; heavier runtime and a server to operate, where Astro ships static output and hydrates only islands |
| Edit `design-system/components/media/Icon.jsx` to use lucide-react | Edits a shared design-system file (used by the console too) and changes it for all consumers; the build-time `resolveId` redirect swaps icons for the docs bundle only, touching zero DS files (ADR-0109) |
| Keep the Google Fonts `@import` for production | A runtime CDN dependency for a self-hostable, AGPL product's docs; the woff2 are self-hosted from the site origin instead |
