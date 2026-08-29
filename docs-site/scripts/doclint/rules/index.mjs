/*
 * The enabled rule set.
 *
 * A later ticket adds its rule here: no-phrasal-verbs (#819) and simple-tenses (#820,
 * the first warning). The walking skeleton (#817) shipped no-semicolons. #818 adds
 * sentence-length-cap.
 */
import { noSemicolons } from "./no-semicolons.mjs";
import { sentenceLengthCap } from "./sentence-length-cap.mjs";

/** @type {import("../engine.mjs").Rule[]} */
export const RULES = [noSemicolons, sentenceLengthCap];
