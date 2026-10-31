// Per grid locale, the defaults ResolveLocale falls back on, plus which languages the runtime has and
// which calendars and numbering systems it supports.
import fs from "node:fs";
import { GRID, baseTag } from "./grid.mjs";
import { OUT, DIGITS } from "./common.mjs";
const defaults = {};
for (const t of [...GRID.locales, ...Object.keys(GRID.calendars), ...GRID.durationLocales]) {
  const b = baseTag(t);
  const ro = new Intl.DateTimeFormat(b).resolvedOptions();
  defaults[b] = { ca: ro.calendar, nu: ro.numberingSystem };
}
const languages = [];
const a = "abcdefghijklmnopqrstuvwxyz";
const probe = [];
for (const x of a) for (const y of a) { probe.push(x + y); for (const z of a) probe.push(x + y + z); }
for (const p of probe) if (Intl.DateTimeFormat.supportedLocalesOf(p, { localeMatcher: "lookup" }).length) languages.push(p);
const digits = {};
const numberingSystems = Intl.supportedValuesOf("numberingSystem");
for (const ns of numberingSystems) {
  const f = new Intl.NumberFormat(`en-u-nu-${ns}`, { useGrouping: false });
  digits[ns] = Array.from({ length: 10 }, (_, i) => f.format(i));
}
fs.writeFileSync(`${OUT}/locale_meta.json`, JSON.stringify({ defaults, languages, calendars: Intl.supportedValuesOf("calendar"), numberingSystems, digits }));
console.log("defaults", Object.keys(defaults).length, "languages", languages.length, "numbering", numberingSystems.length);
