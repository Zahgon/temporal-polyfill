// Month segments of a lunisolar or astronomical calendar, walked month by month from Node:
//   node lunar.mjs CALENDAR FROM_ISO_YEAR TO_ISO_YEAR OUT.json
// A segment is [firstDay, year, monthLabel]: a day in epoch days, ICU's relatedYear or year, the month's
// numeric text ("5", "5bis"). The walk starts at ISO year 2000 and runs outwards until TO_ISO_YEAR,
// FROM_ISO_YEAR or the first date ICU refuses to format. The output carries that limit.
import fs from "node:fs";
const [, , cal, fromY, toY, outFile] = process.argv;
const DAY = 86400000;
const f = new Intl.DateTimeFormat(`en-US-u-ca-${cal}`, { day: "numeric", month: "numeric", year: "numeric", era: "short", timeZone: "UTC" });
const isoDay = (y) => { const d = new Date(0); d.setUTCFullYear(y, 0, 1); return Math.floor(d.getTime() / DAY); };
const MIN = Math.max(isoDay(Number(fromY)), -1e8), MAX = Math.min(isoDay(Number(toY)), 1e8);
const get = (d) => {
  let y, m, dom;
  for (const p of f.formatToParts(d * DAY)) {
    if (p.type === "year" || p.type === "relatedYear") y = p.value;
    else if (p.type === "month") m = p.value.replace(/^Mo/, "");
    else if (p.type === "day") dom = Number(p.value);
  }
  return [y, m, dom];
};
const t0 = Date.now();
// Find the first day of the month holding `d`.
const start = isoDay(2000);
let first = get(start);
let s0 = start - first[2] + 1;
const forward = [];
let limitHi = null, s = s0, cur = get(s0);
for (;;) {
  // The next month starts at the first probe whose label differs or whose day is not the expected one.
  let n = s + 29, r;
  try {
    r = get(n);
    while (r[0] === cur[0] && r[1] === cur[1] && r[2] === n - s + 1) { n++; r = get(n); }
    if (r[2] !== 1 && n - r[2] + 1 > s) { n = n - r[2] + 1; r = get(n); }
  } catch (e) { limitHi = [n, e.message]; break; }
  forward.push([s, Number(cur[0]), cur[1]]);
  s = n; cur = r;
  if (s > MAX) { forward.push([s, Number(cur[0]), cur[1]]); break; }
}
const backward = [];
let limitLo = null;
s = s0; 
for (;;) {
  let r;
  try { r = get(s - 1); } catch (e) { limitLo = [s - 1, e.message]; break; }
  let b = s - r[2];
  // An error can sit inside the month: require its first day to format too.
  let rb;
  try { rb = get(b); } catch (e) { limitLo = [b, e.message]; break; }
  if (rb[1] !== r[1] || rb[2] !== 1) { limitLo = [b, "irregular: " + JSON.stringify([rb, r])]; break; }
  backward.push([b, Number(r[0]), r[1]]);
  s = b;
  if (s < MIN) break;
}
backward.reverse();
const segments = [...backward, ...forward];
fs.writeFileSync(outFile, JSON.stringify({ calendar: cal, limitLo, limitHi, segments }));
console.log(cal, "segments", segments.length, "from", new Date(segments[0][0] * DAY).toISOString(), "to", new Date(segments.at(-1)[0] * DAY).toISOString(), "limits", JSON.stringify(limitLo), JSON.stringify(limitHi), "seconds", (Date.now() - t0) / 1000);
