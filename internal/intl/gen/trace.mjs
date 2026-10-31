// The option bags the polyfill hands to Intl.DateTimeFormat. Its amend functions merge the caller's options with
// the resolved options of a first formatter, so what reaches ICU depends on the context as well as on the bag.
// `node trace.mjs CHUNK CHUNKS` runs every grid bag through every Temporal type in each context of the chunk;
// `node trace.mjs merge` writes OUT/traced_bags.json, {context key: [bags]}.
import fs from "node:fs";
import { createRequire } from "node:module";
import { GRID, contexts, bagKey } from "./grid.mjs";

const OUT = process.env.OUT ?? "/tp/extract/out";
if (process.argv[2] === "merge") {
  const merged = {};
  for (const f of fs.readdirSync(OUT)) if (f.startsWith("traced_") && f !== "traced_bags.json") Object.assign(merged, JSON.parse(fs.readFileSync(`${OUT}/${f}`, "utf8")));
  fs.writeFileSync(`${OUT}/traced_bags.json`, JSON.stringify(merged));
  console.log("contexts", Object.keys(merged).length, "bags", Object.values(merged).reduce((a, b) => a + b.length, 0));
  process.exit(0);
}
const [chunk, chunks] = process.argv.slice(2).map(Number);
const Original = Intl.DateTimeFormat;
const SKIP = new Set(["timeZone", "calendar", "numberingSystem", "locale", "localeMatcher", "formatMatcher"]);
let sink = null;
function Wrapped(l, o) {
  if (sink) {
    const b = {};
    for (const k of Object.keys(o ?? {}).sort()) if (!SKIP.has(k) && o[k] !== undefined) b[k] = o[k];
    if (!sink.has(bagKey(b))) sink.set(bagKey(b), b);
  }
  return new Original(l, o);
}
Wrapped.prototype = Original.prototype;
Wrapped.supportedLocalesOf = Original.supportedLocalesOf;
Intl.DateTimeFormat = Wrapped;
const require = createRequire(import.meta.url);
const { Temporal: T, Intl: TIntl } = require(process.env.TEMPORAL_ORIGINAL ?? "/orig/dist/index.cjs");
const res = {};
const ctxs = [...contexts().entries()].filter((_, i) => i % chunks === chunk);
for (const [key, tag] of ctxs) {
  const cal = new Original(tag).resolvedOptions().calendar;
  const d = T.PlainDate.from("2024-03-15");
  const values = [d, T.PlainTime.from("12:30:45"), T.PlainDateTime.from("2024-03-15T12:30:45"), T.Instant.from("2024-03-15T12:30:45Z")];
  // The islamic and islamic-rgsa ids have no Temporal calendar to carry.
  try { const dc = d.withCalendar(cal); values.push(dc.toPlainYearMonth(), dc.toPlainMonthDay()); } catch {}
  const zdt = T.ZonedDateTime.from("2024-03-15T12:30:45[UTC]");
  sink = new Map();
  for (const bag of [{}, ...GRID.bags]) {
    const opts = { ...bag, timeZone: "UTC" };
    try {
      const f = new TIntl.DateTimeFormat(tag, opts);
      for (const v of values) { try { f.format(v); } catch {} }
    } catch {}
    for (const v of values) { try { v.toLocaleString(tag, opts); } catch {} }
    try { zdt.toLocaleString(tag, bag); } catch {}
    try { new TIntl.DateTimeFormat(tag, bag); } catch {}
  }
  res[key] = [...sink.values()];
  sink = null;
}
fs.writeFileSync(`${OUT}/traced_${chunk}.json`, JSON.stringify(res));
console.log("chunk", chunk, "contexts", ctxs.length, "bags", Object.values(res).reduce((a, b) => a + b.length, 0));
