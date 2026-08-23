# Security Policy

## Reporting a vulnerability

**Please do not open a public issue for security vulnerabilities.** A public
report can expose other users before a fix is available.

Instead, report privately through GitHub's **Private Vulnerability Reporting**:

1. Go to the [Security tab](https://github.com/winniel123/verge-asm/security) of this repository.
2. Click **Report a vulnerability**.
3. Fill in the advisory form with the details below.

This opens a private channel visible only to the maintainers. If Private
Vulnerability Reporting is unavailable to you, open a minimal public issue that
says only "requesting a private security contact" — with **no exploit
details** — and a maintainer will follow up.

## What to include

A good report lets us reproduce and triage quickly. Where you can, include:

- The component affected (`cmd/web`, `cmd/worker`, `cmd/prober`, the docs site, CI, etc.).
- The version, commit SHA, or deployment the finding was observed on.
- Step-by-step reproduction, a proof-of-concept, or the specific code path.
- The impact you believe it has (data exposure, RCE, privilege escalation, DoS, …).
- Any suggested remediation, if you have one.

## Our commitment

- **Acknowledgement** within 3 business days of your report.
- **An initial assessment** (severity, whether we can reproduce) within 10 business days.
- **Progress updates** on the private advisory until the issue is resolved.
- **Credit** in the advisory and release notes when a fix ships, unless you ask to remain anonymous.

## Scope

In scope: the application code under `cmd/`, `internal/`, and `db/`; the build,
release, and CI configuration under `.github/`, `Dockerfile`, and
`docker-compose.yml`; and the published documentation site under `docs-site/`.

Out of scope:

- **`design-system/` and `design-system-legacy/`** — static design assets, not a
  production request path. Demo values there (e.g. placeholder tokens) are
  fixtures, not live credentials.
- **`prototypes/`, `temp/`, and `docs/`** — non-shipping material.
- Findings that require a compromised maintainer machine, a self-hosted fork's
  own misconfiguration, or social engineering of the maintainers.
- Volumetric denial-of-service and automated scanner output without a concrete,
  demonstrated impact.

## Safe harbor

We will not pursue or support legal action against researchers who act in good
faith, avoid privacy violations and service degradation, and give us a
reasonable window to remediate before any public disclosure. Please test only
against your own instances — never against other users' data.
