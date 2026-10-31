// Localized zone names per name period, for the grid's zones and each (base tag, numbering system).
// A period starts at every tz offset transition and every name change found on a one-day and, for all
// locales, a seven-day grid (bisected to the second) between 1800 and 2101.   node names.mjs CHUNK CHUNKS
import fs from "node:fs";
import { GRID, contexts } from "./grid.mjs";
import { OUT } from "./common.mjs";
const [, , chunkArg, chunksArg] = process.argv;
const chunk = Number(chunkArg), chunks = Number(chunksArg);
const STYLES = ["short", "long", "shortOffset", "longOffset", "shortGeneric", "longGeneric"];
const nc = new Map();
for (const [key, tag] of contexts()) {
  const [base, , nu] = key.split("|");
  const def = new Intl.DateTimeFormat(base).resolvedOptions().numberingSystem;
  nc.set(`${base}|${nu}`, nu === def ? base : `${base}-u-nu-${nu}`);
}
const ctxs = [...nc.keys()], tags = [...nc.values()];
const tz = {};
for (const f of fs.readdirSync(OUT)) if (/^tz_\d+\.json$/.test(f)) Object.assign(tz, JSON.parse(fs.readFileSync(`${OUT}/${f}`, "utf8")));
const ids = JSON.parse(fs.readFileSync(`${OUT}/zone_ids.json`, "utf8")).resolved;
const name = (f, ms) => { const p = f.formatToParts(ms).find((m) => m.type === "timeZoneName"); return p ? p.value : null; };
const START = Date.UTC(1800, 0, 1) / 1000, END = Date.UTC(2101, 0, 1) / 1000, STEP = 86400;
const out = { contexts: ctxs, tags, zones: {}, vectors: [] };
const vecIndex = new Map();
const vecId = (v) => { const k = JSON.stringify(v); if (!vecIndex.has(k)) { vecIndex.set(k, out.vectors.length); out.vectors.push(v); } return vecIndex.get(k); };
for (const z of GRID.zones.filter((_, i) => i % chunks === chunk)) {
  const fmts = tags.map((c) => STYLES.map((S) => new Intl.DateTimeFormat(c, { timeZoneName: S, timeZone: z })));
  const probes = ["en-US", "fr", "zh-CN", "ar"].map((c) => STYLES.map((S) => new Intl.DateTimeFormat(c, { timeZoneName: S, timeZone: z })));
  const sig = (s) => probes.map((a) => a.map((f) => name(f, s * 1000)).join("|")).join("|");
  const vec = (ms) => vecId(fmts.map((fs) => fs.map((f) => name(f, ms))));
  const points = new Set();
  const canon = ids[z] ?? z;
  for (const [t] of (tz[canon]?.trans ?? [])) points.add(t);
  // A few locales daily, all of them weekly: a generic name can change in one locale only.
  const full = (s) => fmts.map((a) => a.map((f) => name(f, s * 1000)).join("|")).join("|");
  const walk = (key, step) => {
    let prev = key(START);
    for (let t = START + step; t <= END; t += step) {
      const s = key(t);
      let from = t - step;
      while (s !== prev) {
        let lo = from, hi = t;
        while (hi - lo > 1) { const mid = Math.floor((lo + hi) / 2); if (key(mid) === prev) lo = mid; else hi = mid; }
        points.add(hi);
        prev = key(hi);
        from = hi;
      }
    }
  };
  walk(sig, STEP);
  walk(full, 7 * STEP);
  // ICU compares a zone with its metazone's reference zone on the wall clock, so generic names change for
  // the hours around a DST transition: look closer there.
  const near = (key, from, to, step) => {
    let prev = key(from);
    for (let t = from + step; t <= to; t += step) {
      const s = key(t);
      let lo0 = t - step;
      while (s !== prev) {
        let lo = lo0, hi = t;
        while (hi - lo > 1) { const mid = Math.floor((lo + hi) / 2); if (key(mid) === prev) lo = mid; else hi = mid; }
        points.add(hi);
        prev = key(hi);
        lo0 = hi;
      }
    }
  };
  for (const [t] of (tz[canon]?.trans ?? [])) if (t > START && t < END) near(full, t - 3 * 3600, t + 3 * 3600, 900);
  const starts = [START, ...[...points].filter((t) => t > START && t <= END).sort((a, b) => a - b)];
  const periods = [];
  for (const t of starts) {
    const v = vec(t * 1000);
    if (!periods.length || periods[periods.length - 1][1] !== v) periods.push([t, v]);
  }
  const far = [-8.64e15, -8.64e15 + 86400000, -1e14, -6.2e13, -3e12, Date.UTC(2150, 0, 15), Date.UTC(2150, 6, 15),
    Date.UTC(2500, 0, 15), Date.UTC(2500, 6, 15), 1e14, 6e15, Date.UTC(275000, 0, 15), Date.UTC(275000, 6, 15), 8.64e15 - 1, 8.64e15]
    .map((ms) => [ms, vec(ms)]);
  out.zones[z] = { periods, far };
}
fs.writeFileSync(`${OUT}/names_${chunk}.json`, JSON.stringify(out));
console.log("chunk", chunk, "zones", Object.keys(out.zones).length, "vectors", out.vectors.length);
