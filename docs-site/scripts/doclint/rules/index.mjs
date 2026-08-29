/*
 * The enabled rule set.
 *
 * A later ticket adds its rule here: sentence-length-cap (#818), no-phrasal-verbs
 * (#819), and simple-tenses (#820, the first warning). The walking skeleton (#817)
 * ships one rule.
 */
import { noSemicolons } from "./no-semicolons.mjs";

/** @type {import("../engine.mjs").Rule[]} */
export const RULES = [noSemicolons];
