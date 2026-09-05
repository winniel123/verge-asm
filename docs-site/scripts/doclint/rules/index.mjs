import { noSemicolons } from "./no-semicolons.mjs";
import { sentenceLengthCap } from "./sentence-length-cap.mjs";
import { noPhrasalVerbs } from "./no-phrasal-verbs.mjs";
import { simpleTenses } from "./simple-tenses.mjs";
import { passiveVoice } from "./passive-voice.mjs";

export const RULES = [noSemicolons, sentenceLengthCap, noPhrasalVerbs, simpleTenses, passiveVoice];
