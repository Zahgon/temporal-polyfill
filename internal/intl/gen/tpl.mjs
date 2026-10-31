// DateTimeFormat templates per (context, bag): part structure, the value of every non-literal part keyed
// by the calendar field it renders, year rendering rules and resolved options. formatRange templates per
// greatest-difference level, from pairs that differ first at that level.
//   node tpl.mjs CHUNK CHUNKS
import fs from "node:fs";
import { GRID, contexts, bagKey } from "./grid.mjs";
import { OUT, DAY, ms, ref, keyFor, fieldsOf, yearRules, minusOf, err, isLunar, yearValue, cycleNumber, cycleOf, hebrewLeap } from "./common.mjs";

const [, , chunkArg, chunksArg] = process.argv;
const chunk = Number(chunkArg), chunks = Number(chunksArg);
const ctxs = [...contexts().entries()].filter((_, i) => i % chunks === chunk);
// Per context, the bags the polyfill's amend functions build from the grid's bags (trace.mjs).
const TRACED = JSON.parse(fs.readFileSync(`${process.env.OUT ?? "/tp/extract/out"}/traced_bags.json`, "utf8"));
const cal = JSON.parse(fs.readFileSync(`${OUT}/cal.json`, "utf8"));
const lunar = {};
for (const c of ["chinese", "dangi"]) {
  const seg = JSON.parse(fs.readFileSync(`${OUT}/lunar_${c}.json`, "utf8")).segments;
  // The occurrence of every month label nearest to 2000.
  const pick = {};
  const near = (a) => Math.abs(a[0] - 10957);
  for (const s of seg) if (!pick[s[2]] || near(s) < near(pick[s[2]])) pick[s[2]] = s;
  lunar[c] = Object.values(pick).map((s) => s[0]);
}

const COMMON = [];
for (let i = 0; i < 800; i++) COMMON.push(ms(2023, 9, 1) + i * DAY + ((i % 24) * 3600 + ((i * 7) % 60) * 60 + ((i * 13) % 60)) * 1000 + ((i * 37) % 1000));
for (let h = 0; h < 24; h++) COMMON.push(ms(2024, 6, 3, h), ms(2024, 6, 3, h, 30), ms(2024, 6, 3, h, 0, 0, 500), ms(2024, 6, 3, h, 0, 1), ms(2024, 6, 3, h, 59, 59, 999));
for (let i = 0; i < 120; i++) COMMON.push(ms(1950 + i, 2 + (i % 10), 14, 10));
const FAR = [];
for (const y of [1, 2, 5, 9, 10, 99, 100, 101, 999, 1000, 1001, 1999, 2000, 2001, 2024, 9999, 10000, 12345, 99999, 100000, 275000, -1, -9, -99, -100, -999, -1000, -9999, -10000, -100000, -271000, -500, 50, 500, 600, 1900]) FAR.push(ms(y, 6, 15, 10));
FAR.push(-8.64e15, -8.64e15 + 1, 8.64e15, 8.64e15 - 1);
const samplesFor = (c) => {
  const ts = [...COMMON];
  if (c === "japanese") for (const [d] of cal.japaneseEras) ts.push((d + 1) * DAY + 10 * 3600000);
  if (c === "chinese" || c === "dangi") { for (const d of lunar[c]) ts.push((d + 3) * DAY + 3600000); for (const t of FAR) { if (Math.abs(t) < 8.64e14) { try { ref(c, t); ts.push(t); } catch {} } } }
  else ts.push(...FAR);
  return ts.map((t) => { const s = fieldsOf(t); s.r = ref(c, t); return s; });
};
const samples = {};
const PROBE_ZONES = ["UTC", "America/New_York", "Asia/Kolkata", "Europe/Paris", "Asia/Tokyo", "+05:30"];
const PROBE_TIMES = [Date.UTC(2024, 0, 15, 12), Date.UTC(2024, 6, 15, 12)];

function template(ctx, bag) {
  let dtf;
  try { dtf = new Intl.DateTimeFormat(ctx, { ...bag, timeZone: "UTC" }); } catch (e) { return { error: err(e) }; }
  const ro = dtf.resolvedOptions();
  const c = ro.calendar, ns = ro.numberingSystem;
  const minus = minusOf(ctx, ns);
  const structs = new Map(), lists = {}, conflicts = [], years = {}, fracs = [], errors = [];
  for (const s of (samples[c] ||= samplesFor(c))) {
    let parts;
    try { parts = dtf.formatToParts(s.t); } catch (e) { errors.push([s.t, ...err(e)]); continue; }
    const sk = parts.map((p) => (p.type === "literal" ? "L" + p.value : "T" + p.type)).join("\u0000");
    if (!structs.has(sk)) structs.set(sk, parts.map((p) => (p.type === "literal" ? ["literal", p.value] : [p.type])));
    for (const p of parts) {
      switch (p.type) {
        case "literal": continue;
        case "year": case "relatedYear": (years[p.type] ||= []).push([yearValue(p.type, c, s.r), p.value]); continue;
        case "fractionalSecond": fracs.push([s.msPart, p.value]); continue;
      }
      const key = keyFor(p.type, c, s.r, s);
      const L = (lists[p.type] ||= {});
      if (key in L) { if (L[key] !== p.value && conflicts.length < 10) conflicts.push([p.type, key, L[key], p.value, s.t]); }
      else L[key] = p.value;
    }
  }
  const { timeZone, ...roRest } = ro;
  const out = { ro: roRest, minus, structs: [...structs.values()], lists };
  if (conflicts.length) out.conflicts = conflicts;
  if (errors.length) out.errors = errors.slice(0, 5);
  const rules = yearRules(ns, minus);
  for (const [type, ys] of Object.entries(years)) {
    const ok = Object.keys(rules).filter((k) => ys.every(([y, v]) => rules[k](y) === v));
    (out.yearRules ||= {})[type] = ok.length ? ok[0] : null;
    (out.yearRuleSets ||= {})[type] = ok;
    if (!ok.length) (out.yearMiss ||= {})[type] = ys.filter(([y, v]) => !Object.values(rules).some((r) => r(y) === v)).slice(0, 8);
  }
  if (fracs.length) {
    const n = bag.fractionalSecondDigits;
    out.fracOK = fracs.every(([m, v]) => rules.numeric(String(m).padStart(3, "0").slice(0, n)) === v);
  }
  if (lists.timeZoneName) {
    out.tzProbe = [];
    for (const z of PROBE_ZONES) {
      const f = new Intl.DateTimeFormat(ctx, { ...bag, timeZone: z });
      for (const t of PROBE_TIMES) out.tzProbe.push(f.formatToParts(t).find((p) => p.type === "timeZoneName")?.value ?? null);
    }
  }
  return out;
}

let seed = 4242 + chunk;
const rnd = () => { seed = (seed * 1103515245 + 12345) % 2147483648; return seed / 2147483648; };
const ri = (n) => Math.floor(rnd() * n);
const monthOf = (c, r) => (isLunar(c) ? r.month.replace("bis", "") : r.month);
// ICU's chinese era is the 60-year cycle, and its month field ignores the leap flag.
const eraOf = (c, r) => (isLunar(c) ? String(cycleNumber(Number(r.year))) : r.era);
const diffLevel = (c, ra, rb, fa, fb) => eraOf(c, ra) !== eraOf(c, rb) ? "G" : ra.year !== rb.year ? "y" : monthOf(c, ra) !== monthOf(c, rb) ? "M" : ra.day !== rb.day ? "d"
  : (fa.h < 12) !== (fb.h < 12) ? "a" : fa.h !== fb.h ? "h" : fa.mi !== fb.mi ? "m" : fa.s !== fb.s ? "s" : fa.msPart !== fb.msPart ? "ms" : "x";
// Pairs differing first at a level, chosen so that every field value a part could show occurs on both
// sides: each pair that brings a new (side, field, value) is kept, on top of a random baseline.
function pairsFor(c, level) {
  const out = [];
  const seen = new Set();
  const coverage = (side, r, f) => {
    let fresh = false;
    for (const [k, v] of [["h", f.h], ["mi", f.mi], ["s", f.s], ["wd", f.wd], ["d", r.day], ["M", r.month + (c === "hebrew" ? "|" + hebrewLeap(Number(r.year)) : "")], ["e", r.era], ["cy", isLunar(c) ? cycleOf(Number(r.year)) : 0], ["y1", r.year === "1"]]) {
      const key = side + k + v;
      if (!seen.has(key)) { seen.add(key); fresh = true; }
    }
    return fresh;
  };
  let lastFresh = 0;
  // Sides are the arguments' order: a later date can come first, so values land on both.
  const add = (a, b, force, i = 0) => {
    if (Math.abs(a) > 8.6e15 || Math.abs(b) > 8.6e15) return;
    const ra = ref(c, a), rb = ref(c, b), fa = fieldsOf(a), fb = fieldsOf(b);
    if (diffLevel(c, ra, rb, fa, fb) !== level) return;
    const fresh = [coverage("a", ra, fa), coverage("b", rb, fb)].some(Boolean);
    if (fresh) lastFresh = i;
    if (fresh || force) out.push([a, b, ra, rb, fa, fb]);
  };
  // Time of day cycles independently of the date.
  const at = (i, days) => ms(2023, 9, 1) + days * DAY + ((i % 24) * 3600 + ((i * 7) % 60) * 60 + ((i * 13) % 60)) * 1000 + ((i * 37) % 1000);
  const eraPairs = { gregory: -500, iso8601: -500, roc: 1900, coptic: 100, ethiopic: -100 };
  // Dates: two years of days, then the whole cycle for chinese, then the start of every era.
  const anchors = [];
  if (c === "japanese") for (const [d] of cal.japaneseEras) anchors.push(d + 1 - 19601);
  const far = { gregory: -500, iso8601: -500, roc: 1900, coptic: 100, ethiopic: -100 };
  if (c in far) for (let k = 0; k < 40; k++) anchors.push(Math.floor((ms(far[c] - k * 7, 1 + (k % 12), 15) - ms(2023, 9, 1)) / DAY));
  const dayOf = (i) => {
    if (i < 1100) return i;
    if (anchors.length && i % 2) return anchors[i % anchors.length];
    return isLunar(c) ? ((i * 37) % 22000) - 11000 : i % 1100;
  };
  const gen = (i, t = at(i, dayOf(i))) => {
    switch (level) {
      case "G":
        if (isLunar(c)) return [t, t - (61 + ri(200)) * 365 * DAY];
        if (c in eraPairs) return [t, ms(eraPairs[c] - ri(300), 1 + ri(12), 1 + ri(28), ri(24), ri(60), ri(60))];
        if (c === "japanese") return [t - 400 * DAY * ri(20), t - 3650 * DAY * (ri(3) + 4)];
        return null;
      case "y": return [t, t + (366 + ri(4000)) * DAY + ri(DAY)];
      case "M": return [t, t + (1 + ri(300)) * DAY + ri(DAY)];
      case "d": return [t, t + (1 + ri(3)) * DAY + (i % 3600) * 1000];
      case "a": { const d = t - (t % DAY); return i % 2 ? [d + ri(12 * 3600000), d + 12 * 3600000 + ri(12 * 3600000)] : [t, t + (12 * 3600000) * (t % DAY < 12 * 3600000 ? 1 : -1)]; }
      case "h": return [t, t + (1 + ri(11)) * 3600000];
      case "m": return [t, t + (1 + ri(59)) * 60000];
      case "s": return [t, t + (1 + ri(59)) * 1000];
      case "ms": return [t - (t % 1000), t - (t % 1000) + 1 + ri(990)];
    }
  };
  if (c === "japanese" && level === "G") for (let i = 0; i + 1 < cal.japaneseEras.length; i++) add((cal.japaneseEras[i][0] + 1) * DAY + 3600000, (cal.japaneseEras[i + 1][0] + 1) * DAY + 3600000, true);
  if (isLunar(c) && (level === "d" || level === "M" || level === "y")) {
    // The leap months the two years around 2024 lack.
    const gap = level === "d" ? 1 : level === "M" ? 40 : 400;
    for (const d of lunar[c]) { const t = (d + 3) * DAY + 3600000; add(t, t + gap * DAY, true); add(t - gap * DAY, t, true); }
  }
  // Every era once on each side, at each level the era is long enough to hold.
  for (const a of anchors) for (let k = 0; k < 12; k++) {
    const p = gen(a + k, at(k * 5 + a, a + k * 13));
    if (!p) break;
    add(p[0], p[1], false, 0);
    add(p[1], p[0], false, 0);
  }
  for (let i = 0; i < 30000; i++) {
    const p = gen(i);
    if (!p) break;
    if (ri(2)) add(p[1], p[0], i < 120, i); else add(p[0], p[1], i < 120, i);
    if (i > 2500 && i - lastFresh > 2500) break;
  }
  return out;
}
const offFs = {};
const offF = (z) => (offFs[z] ||= new Intl.DateTimeFormat("en-US", { timeZone: z, hourCycle: "h23", year: "numeric", month: "numeric", day: "numeric", hour: "numeric", minute: "numeric", second: "numeric" }));
const pairCache = {};
const LEVELS = ["G", "y", "M", "d", "a", "h", "m", "s", "ms"];
function levels(ctx, bag) {
  let dtf;
  try { dtf = new Intl.DateTimeFormat(ctx, { ...bag, timeZone: "UTC" }); } catch (e) { return { error: err(e) }; }
  const ro = dtf.resolvedOptions(), c = ro.calendar, ns = ro.numberingSystem;
  const minus = minusOf(ctx, ns);
  const rules = yearRules(ns, minus);
  const out = {};
  const zf = {};
  for (const level of LEVELS) {
    const pairs = (pairCache[c + level] ||= pairsFor(c, level));
    if (!pairs.length) continue;
    const shapes = new Map(), lists = {}, years = {}, fracs = {}, conflicts = [];
    let collapsed = 0;
    for (const [a, b, ra, rb, fa, fb] of pairs) {
      const parts = dtf.formatRangeToParts(a, b);
      if (parts.every((p) => p.source === "shared")) { collapsed++; continue; }
      const sk = parts.map((p) => p.type + "/" + p.source + (p.type === "literal" ? "/" + p.value : "")).join("\u0000");
      if (!shapes.has(sk)) shapes.set(sk, parts.map((p) => (p.type === "literal" ? ["literal", p.source, p.value] : [p.type, p.source])));
      const firstEnd = parts.findIndex((p) => p.source === "endRange");
      const seen = {};
      parts.forEach((p, i) => {
        if (p.type === "literal") return;
        const useEnd = p.source === "endRange" || (p.source === "shared" && firstEnd >= 0 && i > firstEnd);
        const r = useEnd ? rb : ra, s = useEnd ? fb : fa;
        const base = p.type + "|" + p.source;
        seen[base] = (seen[base] ?? -1) + 1;
        const slot = base + "|" + seen[base];
        if (p.type === "year" || p.type === "relatedYear") { (years[slot] ||= []).push([yearValue(p.type, c, r), p.value]); return; }
        if (p.type === "fractionalSecond") return;
        const k = keyFor(p.type, c, r, s);
        const L = (lists[slot] ||= {});
        if (k in L) { if (L[k] !== p.value && conflicts.length < 5) conflicts.push([slot, k, L[k], p.value]); } else L[k] = p.value;
      });
    }
    const yr = {};
    for (const [slot, ys] of Object.entries(years)) {
      const ok = Object.keys(rules).filter((k) => ys.every(([y, v]) => rules[k](y) === v));
      yr[slot] = ok;
    }
    out[level] = { shapes: [...shapes.values()], lists, years: yr, minus, collapsed };
    // Interval patterns pick their own zone-name width: probe it in zones with distinct names, on a
    // pair whose wall clock difference is this level's.
    if (Object.keys(lists).some((k) => k.startsWith("timeZoneName|")) && level !== "G") {
      const probes = [];
      for (const z of PROBE_ZONES) {
        const f = (zf[z] ||= new Intl.DateTimeFormat(ctx, { ...bag, timeZone: z }));
        for (const [a, b] of pairs.filter((x) => x[0] > 1e12).slice(0, 2)) {
          const off = (t) => { const p = {}; for (const q of offF(z).formatToParts(t)) p[q.type] = q.value; return Date.UTC(+p.year, +p.month - 1, +p.day, +p.hour, +p.minute, +p.second) - Math.floor(t / 1000) * 1000; };
          const a2 = a - off(a), b2 = b - off(b);
          const parts = f.formatRangeToParts(a2, b2);
          const zp = parts.find((p) => p.type === "timeZoneName");
          if (zp) probes.push([z, Math.floor(a2 / 1000), zp.value]);
        }
      }
      out[level].tzProbe = probes;
    }
    if (conflicts.length) out[level].conflicts = conflicts;
  }
  return out;
}

const res = {};
const t0 = Date.now();
for (const [key, tag] of ctxs) {
  const r = { tag, templates: {}, ranges: {} };
  const seenBags = new Set();
  for (const bag of [...GRID.bags, ...(TRACED[key] ?? [])]) {
    const k = bagKey(bag);
    if (seenBags.has(k)) continue;
    seenBags.add(k);
    r.templates[k] = template(tag, bag);
    if (!r.templates[k].error) r.ranges[k] = levels(tag, bag);
  }
  res[key] = r;
  console.log("chunk", chunk, key, (Date.now() - t0) / 1000);
}
fs.writeFileSync(`${OUT}/tpl_${chunk}.json`, JSON.stringify(res));
