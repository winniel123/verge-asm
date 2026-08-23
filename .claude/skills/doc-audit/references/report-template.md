# Audit Report Format

The report is the deliverable. Someone should be able to read it and know exactly what's
wrong, how you know, and what you did about it — without re-deriving your work.

## Principles

**Evidence, not assertion.** "README says node 18, but `package.json` engines requires
`>=20` and CI runs 20/22" is actionable. "Version is outdated" is not.

**Location on everything.** `file:line` for the claim, `file:line` for the evidence.

**Separate what you did from what you couldn't.** The "needs input" section is the one a
maintainer must actually read, so don't bury it.

**Anything requiring the user opens and closes the report.** The Action required block runs
first, above the severity groups, and is repeated verbatim at the end. Two placements, both
unmissable — a reader who skims the top and a reader who jumps to the bottom each get the
full list. Write "No action required" when there is none.

**Group by severity, not by file.** A reader wants to know what's on fire, not what's in
alphabetical order.

**Be honest about confidence.** Mark anything inferred rather than verified.

Keep it in chat for small audits. For anything substantial, write it to a file
(`DOC_AUDIT.md` or similar) so it survives the conversation — and mention that the file
is a report, not part of the project's documentation, so it doesn't get committed by
accident.

---

## Template

```markdown
# Documentation Audit

**Scope:** <docs covered> · **Commit:** <sha> · **Date:** <date>
**Excluded:** <what and why — changelogs, generated files, vendored docs>

## ⚠ Action required

<Every item needing the user, each an imperative bold line. "No action required" if none.>

1. **<Do this thing>** — <why, and where. Link to the full finding below.>
2. **<Do this other thing>** — <why, and where.>

## Summary

<2-4 sentences: overall state, the single most important problem, what you changed.>

| | Found | Fixed | Needs input |
|---|---|---|---|
| Critical | | | |
| Important | | | |
| Minor | | | |

## Critical — actively misleading

### 1. <Short title>
**Where:** `docs/setup.md:14`
**Claim:** <what the doc says>
**Reality:** <what the code says> — evidence: `package.json:6`, `.github/workflows/ci.yml:22`
**Impact:** <who this breaks and how>
**Action:** Fixed — updated to <x>. / Needs input — <what's unresolved>

## Important

<same structure>

## Minor

<terse — a table or bullets is fine at this level>

| File | Line | Issue | Action |
|---|---|---|---|

## Needs your input

Things I could not resolve from the code:

### <Title>
**Question:** <the specific decision needed>
**Why I stopped:** <what evidence was missing>
**Options:** <the plausible answers, and what each implies>

## Not touched, deliberately

- `CHANGELOG.md` — historical record
- `docs/api/generated/*` — generated output; fix the docstrings in `src/` instead

## Files changed

- `README.md` — <one line>
- `docs/setup.md` — <one line>

## ⚠ Action required (repeated)

<The same list as the top of the report, in full. Last thing the reader sees.>

1. **<Do this thing>** — <why, and where.>
2. **<Do this other thing>** — <why, and where.>
```

---

## Worked example

```markdown
## ⚠ Action required

1. **Confirm the default dev server port is 3000, not 8080.** `README.md:12` and
   `docs/setup.md:12` disagree and the code settles neither — details under "Needs your
   input". Until you confirm, both docs stay as they are.
2. **Apply the `PORT` fallback fix, or tell me to.** The server reads `PORT` with no
   default (`src/server.ts:14`), which is what makes the docs unresolvable. I did not
   touch code during a doc audit.
3. **Re-run `npm run docs:build`.** I edited the docstrings in `src/api/client.ts`; the
   generated pages under `docs/api/` still carry the old signatures.

## Critical — actively misleading

### 1. Documented Node version is two majors behind the requirement

**Where:** `README.md:12`, `docs/contributing.md:8`
**Claim:** Both docs state Node 18 or later.
**Reality:** `package.json:6` sets `"engines": {"node": ">=20"}`, and
`.github/workflows/ci.yml:22` runs the matrix on 20 and 22 only. Node 18 has not been
tested since commit `a3f91c2` (March 2026).
**Impact:** A contributor following the README installs Node 18, then hits an opaque
failure during install rather than a clear version error.
**Action:** Fixed — both docs now state Node 20+, matching engines and CI.

### 2. Setup guide references an npm script that no longer exists

**Where:** `docs/setup.md:12`
**Claim:** Run `npm run start` after install.
**Reality:** `package.json` defines `dev`, `build`, and `test`. There is no `start`
script; it was renamed to `dev` in commit `7c2e10b`.
**Impact:** The setup guide fails at its final step — the point where a new contributor
is least equipped to debug.
**Action:** Fixed — changed to `npm run dev`, consistent with `README.md:9`.

## Needs your input

### Default port disagreement

**Question:** Is the default dev server port 3000 or 8080?
**Why I stopped:** `README.md:12` says 3000 and `docs/setup.md:12` says 8080. The server
reads `PORT` from the environment with no fallback I could find, and `.env.example` does
not set it. Neither the Dockerfile nor compose config exposes a port.
**Options:** If Vite's default applies, 3000 is correct and `docs/setup.md` is wrong. If
deployment sets 8080 somewhere I can't see, both may be right in different contexts — in
which case the docs should say so explicitly rather than each stating a bare number.
```
