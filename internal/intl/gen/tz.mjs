// Offsets of every primary zone: the first offset, transitions (bisected to the second) on a 30-day grid
// over -3000..1800 and a one-day grid over 1800..2500, and checks that later offsets repeat every 400 years.
//   node tz.mjs CHUNK CHUNKS
import fs from "node:fs";
import { OUT } from "./common.mjs";
const [, , chunkArg, chunksArg] = process.argv;
const chunk = Number(chunkArg), chunks = Number(chunksArg);
const candidates = fs.readFileSync(new URL("zone_candidates.txt", import.meta.url), "utf8").trim().split("\n");
const resolved = {};
for (const z of candidates) { try { resolved[z] = new Intl.DateTimeFormat("en-us", { timeZone: z }).resolvedOptions().timeZone; } catch {} }
const primaries = [...new Set(Object.values(resolved))].sort();
const yr = (y) => { const d = new Date(Date.UTC(2000, 0, 1)); d.setUTCFullYear(y); return d.getTime() / 1000; };
const CYCLE = 146097 * 86400;
function offFor(z) {
  const f = new Intl.DateTimeFormat("en-US", { timeZone: z, hourCycle: "h23", era: "short", year: "numeric", month: "numeric", day: "numeric", hour: "numeric", minute: "numeric", second: "numeric" });
  const d = new Date(0);
  return (s) => {
    let era, y, mo, da, h, mi, se;
    for (const p of f.formatToParts(s * 1000)) {
      switch (p.type) { case "era": era = p.value; break; case "year": y = +p.value; break; case "month": mo = +p.value; break; case "day": da = +p.value; break; case "hour": h = +p.value; break; case "minute": mi = +p.value; break; case "second": se = +p.value; break; }
    }
    if (era === "BC") y = 1 - y;
    d.setTime(0); d.setUTCFullYear(y, mo - 1, da); d.setUTCHours(h, mi, se, 0);
    return d.getTime() / 1000 - s;
  };
}
function scan(off, from, to, step) {
  const trans = [];
  let prev = off(from);
  for (let t = from + step; t <= to; t += step) {
    const o = off(t);
    if (o !== prev) {
      let lo = t - step, hi = t;
      // Several transitions can fall into one step: walk them one at a time.
      for (;;) {
        let l = lo, h = hi;
        while (h - l > 1) { const mid = Math.floor((l + h) / 2); if (off(mid) === prev) l = mid; else h = mid; }
        trans.push([h, off(h)]);
        prev = off(h);
        if (prev === o) break;
        lo = h;
      }
    }
  }
  return trans;
}
let seed = 777 + chunk;
const rnd = () => { seed = (seed * 1103515245 + 12345) % 2147483648; return seed / 2147483648; };
const res = {};
for (const id of primaries.filter((_, i) => i % chunks === chunk)) {
  if (/^[+-]\d\d:\d\d$/.test(id)) continue;
  const off = offFor(id);
  const r = { first: off(yr(-3000)), before: scan(off, yr(-3000), yr(1800), 30 * 86400), trans: scan(off, yr(1800), yr(2100), 86400) };
  r.after = r.trans.some(([t]) => t > yr(2050)) ? scan(off, yr(2100), yr(2500), 86400) : [];
  const checks = [];
  for (let i = 0; i < 60; i++) {
    const t = Math.floor(yr(2500) + rnd() * (8.64e12 - yr(2500)));
    const k = Math.floor((t - yr(2100)) / CYCLE);
    checks.push([t, off(t), off(t - k * CYCLE)]);
  }
  r.periodicBad = checks.filter(([, a, c]) => a !== c);
  r.far = [[-8.64e12, off(-8.64e12)], [8.64e12, off(8.64e12)]];
  res[id] = r;
}
fs.writeFileSync(`${OUT}/tz_${chunk}.json`, JSON.stringify(res));
if (chunk === 0) fs.writeFileSync(`${OUT}/zone_ids.json`, JSON.stringify({ resolved, supported: Intl.supportedValuesOf("timeZone") }));
console.log("chunk", chunk, Object.keys(res).length);
