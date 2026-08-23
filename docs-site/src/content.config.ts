/*
 * Astro content collection over the repo-root guides (docs/guides/*.md).
 *
 * PIPELINE STAGE OWNERSHIP: this collection is the physical source the
 * `source-resolution` stage (src/pipeline/source-resolution.ts) reads from. See
 * docs-site/PIPELINE.md for the full staged contract.
 *
 * The schema fields are ALL OPTIONAL on purpose (ADR-0115): the site must build
 * today against the guides' frontmatter-LESS content, and older git refs that Tv
 * (#351) will iterate have no frontmatter at all. T3 (#353) populates `section`
 * and `order`; nothing here may become required without breaking those older refs.
 */
import { defineCollection, z } from "astro:content";
import { glob } from "astro/loaders";

const guides = defineCollection({
  // The guides live one level up at the repo root, outside this Astro project.
  // The glob loader reads them at build time; `entry.id` becomes the filename
  // without extension (using.md -> "using"), which is our route <slug>.
  loader: glob({ pattern: "*.md", base: "../docs/guides" }),
  schema: z.object({
    title: z.string().optional(),
    section: z.string().optional(),
    order: z.number().optional(),
    description: z.string().optional(),
  }),
});

export const collections = { guides };
