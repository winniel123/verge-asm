import { defineConfig } from "astro/config";
import react from "@astrojs/react";
import { fileURLToPath } from "node:url";

const dsRoot = fileURLToPath(new URL("../design-system", import.meta.url));
const localIcon = fileURLToPath(new URL("./src/ds/Icon.jsx", import.meta.url));

const norm = (p) => p.replace(/\\/g, "/");

// The stock design-system Icon renders from a Lucide UMD global a bundled build never loads.
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
      alias: { "@ds": dsRoot },
    },
    server: { fs: { allow: [".", ".."] } },
  },
});
