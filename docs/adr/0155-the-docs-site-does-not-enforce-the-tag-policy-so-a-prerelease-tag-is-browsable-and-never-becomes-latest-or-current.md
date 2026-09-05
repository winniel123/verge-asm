# ADR-0155: the docs site does not enforce the tag policy, so a prerelease tag is browsable and never becomes latest or current

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1398 ADR gaps: docs-site web assets (source-resolution, doclint scope)](https://github.com/winniel123/verge-asm/issues/1398), gap 1
- **PR that deleted the comment:** [#1397](https://github.com/winniel123/verge-asm/pull/1397)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Bounds:** [ADR-0115](./0115-the-docs-site-renders-the-guides-in-place-and-a-version-is-a-git-ref-not-a-copy.md) §2's `latest` clause, at that clause's own site, and [`release-pipeline.md`](../spec/release-pipeline.md) §1.3, at its own site. Both per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)
- **Rests on:** [`release-pipeline.md`](../spec/release-pipeline.md) §1.3, which rules that the project cuts no prerelease tag. This ADR takes that rule as given and changes none of it

## Context

[`docs-site/src/pipeline/source-resolution.ts`](../../docs-site/src/pipeline/source-resolution.ts)
carried this at lines 21 to 27, until #1397 deleted it:

```
 *   • Pre-release tags ........ a semver tag WITH a prerelease segment
 *                               (`-rc`, `-alpha`, `-beta`, `-fixture`, …) IS
 *                               rendered as its own /<tag>/* version tree (so RC
 *                               docs are readable) but is EXCLUDED from `latest`.
 *                               Rationale: readers may want pre-release docs, but
 *                               "current" must always point at a shipped, stable
 *                               release.
```

The block named ADR-0115 as its owner. ADR-0115 does not state the rule. Its Decision §2 rules that
**`latest`** is an **alias** to the highest `v*` tag by semver. It carves out no prerelease and it
never names the `current` badge. #351, the ticket the block also named, defers the question in its
own words: *"define the exact glob + whether pre-release/`-rc` tags publish"*. A deferral is not a
statement. That is #1398's gap 1.

### What the code does today

Four sites carry the rule. Each was read on 2026-09-05.

| Site | Behaviour |
| --- | --- |
| `source-resolution.ts:28` | `SEMVER_TAG` accepts an optional prerelease segment, so a prerelease tag parses |
| `source-resolution.ts:98` | `publishableTags` keeps every parsed tag that carries `docs/guides/`, prerelease included |
| `source-resolution.ts:117` | `refForVersion("latest")` picks the newest tag whose `prerelease` is `null` |
| `source-resolution.ts:213` | `listVersions` puts the `current` badge on the newest stable tag |

`listVersions` also puts `current` on the `latest` row when no stable tag exists, and
`refForVersion` then falls back to `main`. So `current` names a stable tag or the `main` tree. It
never names a prerelease tag.

`canonicalPath` (`source-resolution.ts:224`) returns `/latest/<slug>` for every guide. The canonical
URL of the whole site therefore follows whatever `latest` resolves to.

### The code refines ADR-0115 rather than implementing it

Semver orders a prerelease below its own release and above the previous release. With the tags
`v1.0.0` and `v1.0.1-rc1`, the highest tag by semver is `v1.0.1-rc1`. `compareTagsDesc`
(`source-resolution.ts:78`) agrees, because it compares the patch number before it looks at the
prerelease segment. ADR-0115 §2's plain reading therefore makes `v1.0.1-rc1` the target of `latest`.
The code refuses that. The clause needs a bound at its own site.

### The tension this ADR must settle

[`release-pipeline.md`](../spec/release-pipeline.md) §1.3 is `Accepted` and rules **"The project
cuts no pre-release tag"** on four grounds. So the docs site carries a published policy for a tag
class the release SPEC has decided the project will not cut, and the deleted rationale
(*"readers may want pre-release docs"*) claims an audience for that class.

**The path is reachable, and today nothing refuses it.** §1.3 names three refusal layers. None of
them exists on 2026-09-05.

| Layer | §1.3's site | Measured state |
| --- | --- | --- |
| `tag_name_pattern` on a tag ruleset | repository settings (§17.1) | `gh api repos/winniel123/verge-asm/rulesets` returns one ruleset, `main protection`, target `branch`. There is no tag ruleset |
| the trigger `push: tags: ["v*.*.*"]` | `release.yml` | `.github/workflows/release.yml` does not exist |
| guard 1, `^v[0-9]+\.[0-9]+\.[0-9]+$` | `release.yml`, the `guard` job | the same file does not exist |

§17.1 already owns the first layer as a manual step a human has not yet applied. The other two wait
on the release workflow, which §1.3 itself says an implementation session writes. `git tag -l "v*"`
returns nothing today, so no version tree of any kind publishes yet.

A pushed `v0.1.0-rc1` would therefore reach `git tag -l "v*"` in `allSemverTags` and would build a
tree. The rule in the code is the only thing that decides what happens next.

## Decision

> **The docs site enforces no tag policy. It publishes a version tree for every `v*` tag that
> carries `docs/guides/`, and a prerelease tag is one of them. A prerelease tag is never the target
> of the `latest` alias and never carries the `current` badge. The ground is containment, not a
> reader's demand for release-candidate documentation.
> [`release-pipeline.md`](../spec/release-pipeline.md) §1.3 owns the question of which tags the
> project cuts, and the docs site must stay correct on the day that rule is broken.**

Five limbs.

### 1. The docs site reads the tag namespace and never gates it

`source-resolution.ts` runs at build time, after a tag exists. It cannot refuse a push. Every
mechanism that can refuse a push lives in repository settings or in the release workflow, and §1.3
names all three.

So the docs site takes the tag list as an input it did not choose. Its job is to render that input
honestly. A build-time consumer that assumes a settings rule is in force is wrong exactly when the
setting is missing, and the table above measures that state today.

### 2. A prerelease tag publishes its own tree

The tag exists. A silent skip would make the docs site a second, weaker statement of the tag policy.
§1.3 refuses that shape in its own words, for the glob it declined to tighten: **"The guard regex
must remain the only exact statement of the tag format."** A skip in the docs pipeline is the same
move one layer further out.

§1.3 also prefers a loud refusal to silence. It keeps the guard job so that a bad tag *"leaves a red
run that names the reason"*, and it says a filtered-out tag *"leaves silence"*. A published tree is
the docs site's version of naming the state. An operator who sees `v0.1.0-rc1` in the version picker
learns that a tag exists which §1.3 says the project does not cut.

### 3. `latest` and `current` name a stable release, or they name nothing

`refForVersion("latest")` selects the newest tag whose prerelease segment is `null`. `listVersions`
gives the `current` badge to that same tag. Where no stable tag exists, `latest` falls back to `main`
and carries the badge itself.

**The invariant is that `current` never names a prerelease tag.** It holds in both branches. It also
protects `canonicalPath`, because every canonical URL routes through `latest`.

This limb bounds ADR-0115 §2. Read `latest` as *the highest stable `v*` tag by semver*, not *the
highest `v*` tag by semver*. The bound is written at ADR-0115 §2 as well as here, per ADR-0058.

### 4. The reader-demand ground is withdrawn

*"Readers may want pre-release docs"* asserts an audience for a tag class the project does not cut.
§1.3 ground 2 measures that audience: the repository has one collaborator and no installed base.
A ground that is false cannot carry the rule, even where the behaviour it defends is correct.

**The containment ground is the only ground a comment at these sites may state.** It survives
whether or not a prerelease tag ever appears, because it is a statement about what the docs site
does with an input, not about who reads the output.

### 5. What this ADR does not reach

- **It does not license cutting a prerelease tag.** §1.3 stands unamended in its own subject. The
  project still cuts none. This ADR rules on a consumer, and it gives §1.3 no exception.
- **It does not discharge §17.1's tag ruleset.** That manual step is still owed. Containment in a
  build script is not a refusal at the push, and it is not offered as a substitute.
- **It does not reach the update check.** §1.4 already rules that path, and it rules the opposite
  way on purpose: a fork's repointed feed gets no defence.
- **It does not reach the `dev` tree.** `main` publishes as `dev` under ADR-0115 §2 and carries its
  own badge. Nothing here changes that.

## Consequences

- **[`release-pipeline.md`](../spec/release-pipeline.md) §1.3 gains one bounding note** at its own
  site. §1.3 enumerates three layers that refuse a prerelease tag. A reader of that list now learns
  that a fourth site meets such a tag and does not refuse it.
- **[ADR-0115](./0115-the-docs-site-renders-the-guides-in-place-and-a-version-is-a-git-ref-not-a-copy.md)
  §2 gains one bounding note** on its `latest` bullet. The alias targets the highest **stable** tag.
- **`source-resolution.ts:212` gains this ADR's citation.** It is the one site that states the rule
  in prose. `refForVersion` implements the same rule and states nothing, so it gains nothing.
- **No production behaviour changes.** The code already has the shape this ADR states. What changes
  is that the shape now has a ground that survives §1.3, and a record both documents can cite.
- **Nothing publishes under this rule yet.** The repository carries no `v*` tag, so `listVersions`
  returns the `latest` and `main` rows alone.
- **[`CONTEXT.md`](../../CONTEXT.md) gains nothing.** Prerelease tag, `latest` and `current` are
  release and docs-site terms. None of them is a domain term.
- **A future prerelease policy reopens this ADR, not §1.3 alone.** A decision to cut candidate tags
  would change the audience ground that limb 4 rejects.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Skip a prerelease tag entirely, so it publishes no tree** | Makes the docs site a second statement of the tag policy, in a build script, where §1.3 rules that the guard regex must remain the only exact statement of the tag format. The skip is also silent, and §1.3 prefers a loud refusal to silence. It hides a tag that exists from the one surface an operator reads |
| **Keep the deleted ground, "readers may want pre-release docs"** | It claims an audience for a class the project does not cut. §1.3 ground 2 measures the audience at one collaborator and no installed base. The behaviour is right and the ground is false, which is the pairing limb 4 exists to break |
| **Let `latest` resolve to the highest tag by semver, prerelease included** | ADR-0115 §2's plain reading. One `v0.1.0-rc1` would become the default docs for every reader who opens `/`, because `canonicalPath` routes every guide through `latest`. `current` would then name a candidate, which is the exact state the code refuses |
| **Give a prerelease tag a badge of its own, such as `rc`** | `VersionOption.tag` feeds the design system `VersionSelect`, and [`docs-site/PIPELINE.md`](../../docs-site/PIPELINE.md) documents two values: `current` for the accent and `dev` for the muted row. A third value needs a design-system change, and it would publish a name for a release train the project refuses to run |
| **Fail the docs build when a prerelease tag exists** | Reports a repository-settings defect at a site that cannot fix it, and takes the docs offline for a fault in the tag namespace. §17.1 owns the settings gap. A red docs publish also tells an operator nothing about which layer is missing |
| **Write the rule in [`docs-site/PIPELINE.md`](../../docs-site/PIPELINE.md)** | That file is the three-stage interface contract for #350's tickets. The rule crosses the docs site and the release SPEC, and both sides need to cite it. A stage contract is read by the ticket that extends the stage, not by a reader of §1.3 |
| **Amend §1.3 and file no ADR** | §1.3 rules which tags the project cuts. This rules what a consumer does with a tag the project did not cut, and it withdraws a ground the code stated. §1.3 takes the bounding note its own list needs and no more |
| **Leave the rule uncited in the surviving comment** | The survivor at `source-resolution.ts:212` carries the rule and settles nothing against §1.3. A later reader who finds both reads a docs pipeline that plans for a tag class the release SPEC refuses, with no record of which one wins |
