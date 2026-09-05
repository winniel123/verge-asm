import { defineCollection, z } from "astro:content";
import { glob } from "astro/loaders";

const guides = defineCollection({
  loader: glob({ pattern: "*.md", base: "../docs/guides" }),
  schema: z.object({
    title: z.string().optional(),
    section: z.string().optional(),
    order: z.number().optional(),
    description: z.string().optional(),
  }),
});

export const collections = { guides };
