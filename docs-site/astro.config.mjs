import { defineConfig } from "astro/config";
import react from "@astrojs/react";
import { fileURLToPath } from "node:url";

// The design system lives one level up at the repo root. We consume it directly
// (ADR-0109): its tokens/components are portable source, never re-authored here.
const dsRoot = fileURLToPath(new URL("../design-system", import.meta.url));
const localIcon = fileURLToPath(new URL("./src/ds/Icon.jsx", import.meta.url));

const norm = (p) => p.replace(/\\/g, "/");

// The one sanctioned integration seam (DS README): every DS consumer routes icons
// through components/media/Icon.jsx, whose stock internals render via the Lucide UMD
// CDN script. We redirect that single module to a bundler-friendly lucide-react
// wrapper — a build-time swap that touches zero design-system files.
const dsIconLucideSwap = {
  name: "ds-icon-lucide-swap",
  enforce: "pre",
  resolveId(source, importer) {
    if (
      importer &&
      norm(source).endsWith("media/Icon.jsx") &&
      norm(importer).includes("/design-system/")
    ) {
      return localIcon;
    }
    return null;
  },
};

export default defineConfig({
  integrations: [react()],
  vite: {
    plugins: [dsIconLucideSwap],
    resolve: {
      // `@ds/*` -> repo-root design-system. Keeps DS imports legible instead of ../../../.
      alias: { "@ds": dsRoot },
    },
    // The design system sits outside the docs-site root; allow Vite to read it.
    server: { fs: { allow: [".", ".."] } },
  },
});
