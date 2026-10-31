// Calendar facts as ICU computes them: Japanese era starts, Umm al-Qura month starts, Persian year starts.
import fs from "node:fs";
import { OUT, DAY, ms, ref } from "./common.mjs";
const out = {};
{
  const eras = [];
  let prev = null;
  for (let t = ms(600, 1, 1); t < ms(2030, 1, 1); t += DAY) {
    const e = ref("japanese", t).era;
    if (e !== prev) { eras.push([t / DAY, e]); prev = e; }
  }
  out.japaneseEras = eras;
}
for (const cal of ["islamic-umalqura", "islamic-civil"]) {
  const starts = [];
  let prevKey = null;
  for (let t = ms(1850, 1, 1); t < ms(2200, 1, 1); t += DAY) {
    const r = ref(cal, t);
    const key = r.year + "|" + r.month;
    if (key !== prevKey) { starts.push([t / DAY, r.year, r.month]); prevKey = key; }
  }
  out["months_" + cal] = starts;
}
const yearAt = (cal, d) => Number(ref(cal, d * DAY).year);
function firstDay(cal, y, lo, hi) {
  while (hi - lo > 1) { const mid = Math.floor((lo + hi) / 2); if (yearAt(cal, mid) >= y) hi = mid; else lo = mid; }
  return hi;
}
const persian = [];
for (let y = 1100; y <= 3101; y++) {
  const approx = Math.floor((y - 1349) * 365.2422);
  persian.push(firstDay("persian", y, approx - 400, approx + 400));
}
out.persianStarts = { first: 1100, starts: persian };
fs.writeFileSync(`${OUT}/cal.json`, JSON.stringify(out));
console.log("eras", out.japaneseEras.length, out.japaneseEras.at(-1));
