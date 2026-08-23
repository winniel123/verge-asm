# Check Catalog

The full set of checks for a documentation audit, and — more importantly — **where the
ground truth for each one lives**. Read the section relevant to the repo you're auditing.

- [How to use this](#how-to-use-this)
- [Ground truth by ecosystem](#ground-truth-by-ecosystem)
- [Claim checks](#claim-checks)
- [Contradiction checks](#contradiction-checks)
- [Coverage checks](#coverage-checks)
- [Quality checks](#quality-checks)
- [Repo-type specifics](#repo-type-specifics)

---

## How to use this

Every check follows the same shape: **a claim in the docs**, **a place in the repo that
settles it**, and **a verdict**. If you can't find the settling place, the verdict is
"unverified" — that's a legitimate outcome, not a failure.

Prioritize by blast radius. A wrong install command in the README affects every new
contributor. A stale sentence in an internal design note affects almost nobody. Spend
your effort accordingly.

---

## Ground truth by ecosystem

Where to look when a doc makes a claim about tooling or versions.

**Node / JavaScript / TypeScript**
- `package.json` → `scripts` (commands), `engines` (node/npm versions), `dependencies`,
  `bin` (CLI names), `exports`/`main` (import paths)
- Lockfile → actually installed versions, when the range in `package.json` is ambiguous
- `tsconfig.json` → path aliases documented in import examples
- `.nvmrc` → the node version contributors actually use

**Python**
- `pyproject.toml` → `requires-python`, dependencies, `[project.scripts]` entry points
- `setup.py` / `setup.cfg` → older projects
- `requirements*.txt`, `.python-version`, `tox.ini`, `noxfile.py`
- `argparse`/`click`/`typer` definitions → real CLI flags

**Go** — `go.mod` (module path, go version), `cmd/` layout, `flag`/`cobra` definitions
**Rust** — `Cargo.toml` (`rust-version`, features, bin targets), `clap` definitions
**Ruby** — `Gemfile`, `.ruby-version`, `*.gemspec`, Rake tasks
**Java/Kotlin** — `pom.xml`, `build.gradle(.kts)`, toolchain/target settings
**PHP** — `composer.json` (`require.php`, scripts)
**.NET** — `*.csproj` (`TargetFramework`), `global.json`

**Cross-cutting, and often the best evidence available**
- **CI workflows** (`.github/workflows/`, `.gitlab-ci.yml`, `Jenkinsfile`) — the version
  matrix and the commands here are what *actually* run on every commit. When CI and a doc
  disagree, CI is almost always right.
- `Dockerfile` / `docker-compose.yml` — base image versions, exposed ports, env vars,
  service names, entrypoint commands
- `Makefile` / `Justfile` / `Taskfile.yml` — real task names
- `.env.example` / `.env.sample` — the canonical env var list
- `.pre-commit-config.yaml`, linter configs — documented style rules
- Kubernetes manifests, Terraform, `serverless.yml` — deployed ports, env, resource names

---

## Claim checks

### Commands
Every command in a setup, quickstart, contributing, or troubleshooting doc.
- Does the script/target/subcommand exist in the manifest?
- Do the flags exist in the argument parser?
- Is the ordering right (install before build before run)?
- Are there undocumented prerequisite steps that would cause a failure?
- Does a documented tool appear anywhere in the repo's dependencies at all?

### Versions and prerequisites
- Documented language/runtime versions vs. manifests, `.tool-versions`, CI matrices.
- Documented dependency versions vs. lockfile reality.
- Minimum vs. recommended stated clearly and not contradicted elsewhere.
- Watch for versions that are *newer* in docs than in code — a sign docs were updated
  aspirationally, and equally a defect.

### File and directory references
- Prose references like "see `src/config/settings.py`" — the link checker misses these
  because they aren't links. Verify each path exists.
- Documented directory trees vs. the real tree (`git ls-files | head -100`, or `tree -L 2`).
- Import paths in examples vs. real module layout.

### Environment variables and configuration
- Doc env index vs. `.env.example`, config loaders, deployment manifests.
- **Documented but unused** → dead config, or a rename that missed the docs.
- **Used but undocumented** → the classic "works on my machine" trap. High value.
- Defaults stated in docs vs. defaults in code.
- Required vs. optional correctly marked; secrets never given real values.

### Ports, URLs, endpoints
- Default ports vs. server config, Dockerfile `EXPOSE`, compose mappings.
- Documented API routes vs. route definitions; methods and path params.
- Base URLs, health check paths, dashboard URLs.

### Code examples
- Imports resolve against the real module layout.
- Function and method signatures match current definitions — parameter names, order,
  required vs. optional, return shape.
- Class and method names still exist and aren't renamed.
- Examples use current idioms, not a deprecated API the codebase has moved off.
- Where cheap and safe: parse, type check, or lint the snippet. Never execute anything
  with side effects.

### Architecture and behavior claims
- Described components exist; removed services aren't still documented.
- Data flow claims match the actual call path.
- Diagrams (Mermaid, ASCII, images) reference real, current components. Mermaid blocks
  should also be syntactically valid.
- Performance or limit claims ("handles 10k req/s", "max 5MB") — flag for confirmation
  rather than trusting; these age badly and are rarely re-measured.

---

## Contradiction checks

The scanner catches version and port conflicts. These need reading:

- **Same procedure, two places, different steps.** README quickstart vs. dedicated setup
  guide is the most common instance.
- **Conflicting recommendations** across docs with no acknowledgment of the alternative.
- **Terminology drift** — one concept, several names. Standardize on what the code uses.
- **Naming drift** — renamed service/module/flag updated in some docs only.
- **Conflicting defaults, limits, or prerequisites.**
- **Deprecation contradictions** — marked deprecated in one place, presented as current
  in another.
- **Stale cross-references** — "as described in the Foo section" where no Foo section
  remains.
- **Support claims** — a platform or version listed as supported in one doc and dropped
  in another (check CI for which is true).

Resolution order: **code > CI config > recently-updated doc > older doc.** Never resolve
by picking whichever doc you read first, and never resolve by deleting a side to make the
disagreement vanish.

---

## Coverage checks

Gaps are invisible to automated checks — you have to go looking.

- Public API surface with no documentation.
- Config options with no documentation.
- Recently added features (`git log --since="6 months ago" --name-only`) with no doc changes.
- Documented features that no longer exist in code.
- Missing basics: install, run, test, build, contribute, license, support.
- Missing troubleshooting for known-common failures — repo issues are good evidence.
- Onboarding path: can a new contributor get from clone to running tests using only
  these docs? Walk it mentally, step by step, and note where you'd get stuck.
- Missing "why" documentation: architecture decisions, non-obvious constraints. Flag as
  a suggestion, not a defect.

---

## Quality checks

Correctness first; these come after.

**Structure** — README answers what/who/how-to-start within the first screen. Logical
progression. Consistent heading hierarchy. TOC present and accurate for long docs.

**Code blocks** — language tag on every fence; no `$` prompts that break copy-paste;
placeholders obviously placeholders; output separated from input; long blocks broken up
with explanation.

**Consistency** — heading capitalization, terminology, code reference formatting, list
punctuation, ordering of parallel sections across files.

**Clarity** — acronyms expanded on first use; no unexplained jargon; no ambiguous "it"
in instructions; no unstated assumed context; steps in executable order.

**Accessibility** — meaningful link text (not "click here"), alt text on images, tables
that aren't load-bearing for critical instructions.

**Stale framing** — "coming soon" on shipped features, "new" on old ones, passed dates,
completed roadmaps, resolved known issues.

**Links** — internal links resolve (scanner), external links reachable (only if the user
wants them fetched and network is available), no permanent redirects left unfollowed.

---

## Repo-type specifics

**Libraries / SDKs** — install snippet correct for the current published version;
public API fully covered; examples runnable as written; versioning and breaking-change
policy stated; changelog links from README.

**Applications / services** — setup guide works from a clean clone; env vars complete;
local dev and deploy both covered; ports and URLs correct; migration steps present.

**Monorepos** — root README explains the layout; each package has its own README;
cross-package links resolve; the root doesn't contradict package-level docs. Check
whether commands are meant to be run at root or package level — this is a frequent
source of silent contradiction.

**CLI tools** — documented flags match the parser exactly; help output matches docs;
exit codes documented; install methods current for each platform listed.

**Internal / team docs** — owners and contacts still employed and correct; runbooks
reference existing dashboards and alerts; on-call procedures current; links to internal
systems still resolve. Stale ownership info is a real incident risk — treat it as
important, not minor.
