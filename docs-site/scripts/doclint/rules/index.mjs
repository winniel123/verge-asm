/*
 * The enabled rule set.
 *
 * A later ticket adds its rule here: simple-tenses (#820, the first warning). The walking
 * skeleton (#817) shipped no-semicolons. #818 adds sentence-length-cap. #819 adds
 * no-phrasal-verbs.
 */
import { noSemicolons } from "./no-semicolons.mjs";
import { sentenceLengthCap } from "./sentence-length-cap.mjs";
import { noPhrasalVerbs } from "./no-phrasal-verbs.mjs";

/** @type {import("../engine.mjs").Rule[]} */
export const RULES = [noSemicolons, sentenceLengthCap, noPhrasalVerbs];
