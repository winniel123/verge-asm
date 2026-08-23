# Third-party notices — verge-asm docs site

The verge-asm documentation site (`docs-site/`) is licensed AGPL-3.0-only, in line with
the repository as a whole. It redistributes the third-party assets listed below. Their
own licenses and required notices are reproduced or pointed to here.

## Bundled fonts (self-hosted in `public/fonts/`)

- **Instrument Sans** — Copyright 2022 The Instrument Sans Project Authors
  (https://github.com/Instrument/instrument-sans). Licensed under the SIL Open Font
  License, Version 1.1. Full text: [`public/fonts/Instrument-Sans-OFL.txt`](public/fonts/Instrument-Sans-OFL.txt).
- **Geist Mono** — Copyright 2024 The Geist Project Authors
  (https://github.com/vercel/geist-font). Licensed under the SIL Open Font License,
  Version 1.1. Full text: [`public/fonts/Geist-Mono-OFL.txt`](public/fonts/Geist-Mono-OFL.txt).

The `.woff2` files under `public/fonts/` are distributed under the SIL Open Font License
1.1; the license text ships alongside them as required.

## Icons

- **Lucide** (`lucide-react`) — Licensed under the ISC License, with portions derived from
  Feather (MIT, Copyright © 2013–2023 Cole Bemis). Icon SVG paths from Lucide are embedded
  in the built pages. See https://github.com/lucide-icons/lucide/blob/main/LICENSE.

## Build-time dependencies

The remaining npm packages (Astro, React, react-markdown, remark-gfm, github-slugger,
Playwright, pixelmatch, pngjs, …) are pulled at build time and are **not** vendored or
redistributed by this repository. All are permissive-licensed (MIT / ISC / Apache-2.0)
and compatible with AGPL-3.0; consult each package for its own license text.
