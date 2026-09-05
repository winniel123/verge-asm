# Docs pipeline — the staged contract (T1 / #350)

This is the **seam** the parallel wave builds against. The pipeline turns the
repo-root guides (`docs/guides/*.md`) into rendered pages at `/<version>/<slug>`.
It is split into **three named stages with explicit interfaces**. Each downstream
ticket extends **exactly one** stage and imports the others' types — nobody
redefines a shape declared here.

```
docs/guides/*.md
      │
      ▼
┌─────────────────────┐   Source[]            ┌──────────────────────┐
│ 1. source-resolution│ ────────────────────▶ │ 2. render+transform  │  → HTML island
│    (Tv / #351)      │   { version, slug,    │    (T2 / #352)       │
└─────────────────────┘     rawMarkdown,      └──────────────────────┘
      │  Source[]            frontmatter }
      ▼
┌─────────────────────┐   NavSection[]
│ 3. nav-build        │ ────────────────────▶ left rail (SectionNav)
│    (T3 / #353)      │
└─────────────────────┘
```

The one place all three are wired together is the route page
`src/pages/[version]/[slug].astro`. Nothing else imports across stages.

---

## Route / slug scheme

- **Route:** `/<version>/<slug>` — e.g. `/main/using`, `/main/zone-files`.
- **`<version>`** is already a real path segment today, with the single value
  `main`. Tv adds more versions by changing **source-resolution only**; this route
  and every other stage are untouched.
- **`<slug>`** = the guide filename without extension. `using.md` → `using`,
  `zone-files.md` → `zone-files`. This is the Astro glob-loader `entry.id`.

## Heading → anchor-id algorithm (T2 depends on this **verbatim**)

Heading `id`s and the on-page-TOC `#fragment`s are produced by the
[`github-slugger`](https://github.com/Flet/github-slugger) package — the same
library `rehype-slug` uses. The rule:

1. Lowercase; strip characters that aren't word/space/hyphen; spaces → `-`.
2. **De-duplicate** repeated slugs with a numeric suffix (`-1`, `-2`, …). This
   counter is **stateful and order-sensitive**.

Because of the counter, any slugger instance **must visit every heading (all levels
`h1`–`h6`) in source order**. The renderer (`render.jsx`) does this naturally;
`extractToc` (`slug.ts`) mirrors it by slugging *every* heading and only *emitting*
the `h2`/`h3` rows. Do not slug a subset — the counters would diverge and TOC links
would miss.

- **Shared implementation:** `src/pipeline/slug.ts` (`extractToc`) and the heading
  renderers in `src/pipeline/render.jsx` both slug via `github-slugger`. T2's link
  rewriter **must** slug fragments the same way so `foo.md#some-heading` resolves to
  the rendered `id="some-heading"`.
- Verified for `/main/using`: all ten TOC `#anchor`s match a rendered heading `id`.

## Frontmatter schema (all fields OPTIONAL)

Defined in **`src/content.config.ts`** as the Astro content collection `guides`
(glob loader over `../docs/guides/*.md`).

| Field         | Type     | Owner | Notes                                          |
| ------------- | -------- | ----- | ---------------------------------------------- |
| `title`       | `string` | T3    | Nav/tab label; falls back to a title-cased slug|
| `section`     | `string` | T3    | Left-rail group heading                        |
| `order`       | `number` | T3    | Sort within a section                          |
| `description` | `string` | —     | `<meta name="description">`                     |

**Every field is optional and must stay optional.** The site builds today against
frontmatter-LESS guides, and the older git refs Tv iterates have no frontmatter at
all. Making any field required breaks those refs. (ADR-0115.)

---

## Stage 1 — source-resolution   ·   `src/pipeline/source-resolution.ts`   ·   OWNER Tv (#351)

Resolves which guides exist at which version and hands each one's raw markdown +
frontmatter downstream.

```ts
interface Frontmatter { title?: string; section?: string; order?: number; description?: string }
interface Source      { version: string; slug: string; rawMarkdown: string; frontmatter: Frontmatter }
interface VersionOption { value: string; tag?: string }   // mirrors DS VersionSelect.d.ts

const DEFAULT_VERSION = "main";
function resolveSources(version?: string): Promise<Source[]>   // sorted by slug
function listVersions(): VersionOption[]
```

- **Today:** `resolveSources` reads the content collection and stamps
  `version = "main"` on every guide; `listVersions()` returns `[{ value: "main",
  tag: "dev" }]`.
- **Tv makes it a real seam:** swap the internals to iterate git refs and emit one
  `Source` per `(ref × guide)`. **Do not change `Source`/`VersionOption`** — T2 and
  T3 consume only these types, so they need no edits when versions multiply.

## Stage 2 — render+transform   ·   `src/pipeline/render.jsx`   ·   OWNER T2 (#352)

Renders one `Source.rawMarkdown` into the article column as a React island
(`react-markdown` + `remark-gfm`), mapping markdown to the design system:

- fenced code → DS **`CodeBlock`** (title = language)
- blockquote → DS **`Callout`**
- inline code → tokenised `InlineCode`; GFM tables → tokenised `<table>`
- headings → `h1`–`h4` with `github-slugger` `id`s (see algorithm above)

```jsx
export default function Article({ markdown: string }): JSX.Element   // client:load island
```

- **T2 OWNS the link/anchor rewriting step.** The single seam is the `a` renderer in
  the `components` map: today it passes hrefs through (external links get
  `target=_blank`); T2 rewrites relative `running.md#anchor` cross-links into in-site
  `/<version>/running#anchor` routes (and may add a remark/rehype plugin). **T2 edits
  only this stage.**

## Stage 3 — nav-build   ·   `src/pipeline/nav-build.ts`   ·   OWNER T3 (#353)

Builds the left-rail section model for one version from the resolved sources.

```ts
interface NavItem    { label: string; href: string; active?: boolean }
interface NavSection { title: string; items: NavItem[] }             // shape SectionNav.astro renders
function buildNav(sources: Source[], activeSlug?: string): NavSection[]
```

- **Today:** frontmatter is empty, so this emits a single flat `"Guides"` section
  listing every slug (label = `frontmatter.title` or a title-cased slug).
- **T3:** group by `frontmatter.section`, sort within a group by `frontmatter.order`.
  Consumes only `Source`, returns only `NavSection[]` — no edits to stages 1 or 2.

---

## Version manifest shape (T4 / #354 consumes)

The version picker (DS `VersionSelect`) is fed `VersionOption[]`, matching
`design-system/components/navigation/VersionSelect.d.ts` exactly:

```ts
interface VersionOption { value: string; tag?: string }
// tag "current" → accent (latest release);  "dev" → muted (the moving `main` branch)
```

Produced by `listVersions()` in **stage 1** (source-resolution). Today it returns
`[{ value: "main", tag: "dev" }]`. Tv returns the discovered refs, tagging the newest
release `"current"` and `main` `"dev"`. T4 reads it straight from `listVersions()` and
passes it to the TopNav island — no reshaping.

---

## On-page TOC

`extractToc(markdown)` (`src/pipeline/slug.ts`) returns
`[{ label, href: "#slug", level }]` for each `##`/`###` heading, fed to
`OnPageToc.astro`. Fenced code blocks are skipped so a `#` comment inside ```` ```sh ````
is never read as a heading.

## Decisions (recorded here, not a new ADR — ADR-0115 already covers the model)

- **Renderer:** `react-markdown` + `remark-gfm`, not Astro's built-in `.md` render,
  because `.md` (unlike `.mdx`) can't remap `<pre>`/`blockquote` to React DS
  components. ~~This keeps ADR-0109 satisfied (DS components imported via `@ds`, never
  re-authored)~~ and gives one clean `components` map that is stage 2's whole surface.

  > **The struck clause is WITHDRAWN at the site that specifies it, 2026-08-28 by `55aa367` /
  > [#1410](https://github.com/winniel123/verge-asm/issues/1410)
  > ([ADR-0058](../docs/adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)).**
  > ADR-0109 is superseded and `design-system/` may be edited in the repo. The renderer choice
  > stands on its remaining ground: `.md` cannot remap `<pre>` and `blockquote` to React
  > components, and one `components` map is the smaller surface.
- **Slugger:** `github-slugger`, shared by the renderer and `extractToc`, so T2's
  link rewriter has one documented, deterministic anchor algorithm to match.
- **No new ADR** was authored (avoids parallel ADR-number collisions in this wave).
- **Deploy target (T5 / #355 — resolves map Fog):** GitHub Pages
  (`upload-pages-artifact` + `deploy-pages`), gated behind the build so a broken link
  or design drift blocks the publish. CI checks out with `fetch-depth: 0` (tags →
  versions). `/` → `/latest/<first-guide>` via a generated `dist/index.html`
  (meta-refresh + `location.replace` + `<link rel=canonical>`); Tv's `/latest/*` alias
  tree is kept, not swapped for host rules. **Serving constraint:** the site's links
  are root-absolute, so it must be served at the domain root (custom domain or an
  `<owner>.github.io` repo), and Pages Settings → Source must be set to "GitHub
  Actions" once. Full write-up + handoff: `docs-site/DEPLOY.md`.
