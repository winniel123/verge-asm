/*
 * The enabled rule set.
 *
 * The walking skeleton (#817) shipped no-semicolons. #818 adds sentence-length-cap. #819
 * adds no-phrasal-verbs. #820 adds simple-tenses, the first warning-severity rule.
 */
import { noSemicolons } from "./no-semicolons.mjs";
import { sentenceLengthCap } from "./sentence-length-cap.mjs";
import { noPhrasalVerbs } from "./no-phrasal-verbs.mjs";
import { simpleTenses } from "./simple-tenses.mjs";

/** @type {import("../engine.mjs").Rule[]} */
export const RULES = [noSemicolons, sentenceLengthCap, noPhrasalVerbs, simpleTenses];
