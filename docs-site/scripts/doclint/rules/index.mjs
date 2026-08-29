/*
 * The enabled rule set.
 *
 * The walking skeleton (#817) shipped no-semicolons. #818 adds sentence-length-cap. #819
 * adds no-phrasal-verbs. #820 adds simple-tenses, the first warning-severity rule. #823 adds
 * passive-voice, the first candidate warning (SPEC §2.5): it was proven on the fixture corpus
 * and the real in-scope docs before it joined this set (the measured precision figure lives in
 * the rule header). A rule stays out of this list until its fixture corpus passes, so
 * membership here is the "enabled" signal.
 */
import { noSemicolons } from "./no-semicolons.mjs";
import { sentenceLengthCap } from "./sentence-length-cap.mjs";
import { noPhrasalVerbs } from "./no-phrasal-verbs.mjs";
import { simpleTenses } from "./simple-tenses.mjs";
import { passiveVoice } from "./passive-voice.mjs";

/** @type {import("../engine.mjs").Rule[]} */
export const RULES = [noSemicolons, sentenceLengthCap, noPhrasalVerbs, simpleTenses, passiveVoice];
