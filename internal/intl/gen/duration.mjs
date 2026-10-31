// What Intl.DurationFormat composes per duration locale: number symbols, unit patterns per plural
// category and width, list patterns per width, the time separator, and PluralRules samples.
import fs from "node:fs";
import { GRID } from "./grid.mjs";
import { OUT, DIGITS } from "./common.mjs";
const UNITS = ["year", "month", "week", "day", "hour", "minute", "second", "millisecond", "microsecond", "nanosecond"];
const WIDTHS = ["long", "short", "narrow"];
const VALS = [0, 1, 2, 3, 4, 5, 6, 7, 8, 10, 11, 12, 15, 20, 21, 22, 25, 100, 101, 102, 1000, 1000000, 0.5, 1.5, 2.5, 0.25, 1.123, 1.0005, 1234567];
const LIST_INPUTS = [["A", "B"], ["A", "B", "C"], ["A", "B", "C", "D"]];
const PLURAL = [...Array.from({ length: 300 }, (_, i) => i), 1000, 1001, 1000000, 1000001, 2000000, 1e7, 1e9, 0.1, 0.5, 1.1, 1.5, 2.5, 3.5, 11.5, 100.5, 1.25, 0.125, 1.123];
const out = {};
for (const tag of GRID.durationLocales) {
  const ro = new Intl.NumberFormat(tag).resolvedOptions();
  const ns = ro.numberingSystem;
  const g = new Intl.NumberFormat(tag);
  const f = new Intl.NumberFormat(tag, { maximumFractionDigits: 9 });
  const r = { locale: new Intl.DurationFormat(tag).resolvedOptions().locale, ns, digits: DIGITS[ns] };
  const dg = r.digits;
  if (!dg) throw new Error("no digits for " + ns);
  const parts = f.formatToParts(-1234567.5);
  r.minus = parts.find((p) => p.type === "minusSign")?.value ?? "-";
  r.group = parts.find((p) => p.type === "group")?.value ?? "";
  r.decimal = parts.find((p) => p.type === "decimal")?.value ?? ".";
  r.negPattern = f.format(-1);
  const loc = (t) => t.replace(/[0-9]/g, (d) => dg[d]);
  r.minGrouping = g.format(1000) === loc("1000") && g.format(10000) !== loc("10000") ? 2 : 1;
  r.secondary = g.format(100000).includes(r.group) && g.format(100000) === loc("1") + r.group + loc("00") + r.group + loc("000") ? 2 : 3;
  const pr = new Intl.PluralRules(tag);
  r.pluralLocale = pr.resolvedOptions().locale;
  r.plural = PLURAL.map((v) => pr.select(v));
  r.units = {};
  for (const w of WIDTHS) for (const u of UNITS) {
    const nf = new Intl.NumberFormat(tag, { style: "unit", unit: u, unitDisplay: w, maximumFractionDigits: 9 });
    const m = {};
    for (const v of VALS) {
      const cat = pr.select(v), o = nf.format(v), num = f.format(v);
      const i = o.indexOf(num);
      const pat = i < 0 ? o : o.slice(0, i) + "{0}" + o.slice(i + num.length);
      if (cat in m && m[cat] !== pat) (r.conflicts ||= []).push([w, u, v, m[cat], pat]);
      if (!(cat in m)) m[cat] = pat;
    }
    r.units[`${w}|${u}`] = m;
  }
  r.lists = {};
  for (const w of WIDTHS) {
    const lf = new Intl.ListFormat(tag, { type: "unit", style: w });
    r.lists[w] = LIST_INPUTS.map((a) => lf.format(a));
  }
  r.digitalSample = new Intl.DurationFormat(tag, { style: "digital" }).format({ hours: 1, minutes: 2, seconds: 3 });
  out[tag] = r;
}
fs.writeFileSync(`${OUT}/duration.json`, JSON.stringify(out));
console.log("duration locales", Object.keys(out).length);
for (const [k, r] of Object.entries(out)) console.log(k, r.locale, r.ns, JSON.stringify(r.digitalSample), JSON.stringify(r.negPattern), r.conflicts ? r.conflicts.length : 0, JSON.stringify(r.lists.short[1]));
