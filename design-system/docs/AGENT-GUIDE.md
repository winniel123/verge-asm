---
name: verge-asm-design
description: Use this skill to generate well-branded interfaces and assets for Verge ASM (open-source attack surface management), either for production or throwaway prototypes/mocks/etc. Contains essential design guidelines, colors, type, fonts, assets, and UI kit components for prototyping.
user-invocable: true
---

Read the README.md file within this skill, and explore the other available files.
If creating visual artifacts (slides, mocks, throwaway prototypes, etc), copy assets out and create static HTML files for the user to view. If working on production code, you can copy assets and read the rules here to become an expert in designing with this brand.
If the user invokes this skill without any other guidance, ask them what they want to build or design, ask some questions, and act as an expert designer who outputs HTML artifacts _or_ production code, depending on the need.

Non-negotiables when designing for Verge ASM:
- Tokens live in `tokens/*.css` (imported via `styles.css`); dark mode = `data-theme="dark"` on any subtree.
- Technical values (hostnames, IPs, ports, CVE ids, hashes, versions, counts, timestamps) are always Geist Mono.
- Severity is exactly Critical / High / Medium / Low / Info — use `SeverityBadge`, never restyle or rename levels.
- Copy: sentence case, imperative actions, "you" never "we", no exclamation marks, no emoji; the word is "signal" (never finding/host/fingerprint).
- Icons: Lucide via CDN through the `Icon` component.
