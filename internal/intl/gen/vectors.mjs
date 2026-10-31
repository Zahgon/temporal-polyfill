// Records test vectors from Node into gzip-compressed JSON lines:
//   node vectors.mjs OUT_DIR FACTOR SEED [chunk chunks]
// FACTOR scales the random samples: 1 for internal/intl/testdata, larger for a stress run outside the tree.
import fs from "node:fs";
import zlib from "node:zlib";
import { GRID, contexts, bagKey } from "./grid.mjs";
import { DAY, ms } from "./common.mjs";
const [, , outDir, factorArg, seedArg, chunkArg, chunksArg] = process.argv;
const factor = Number(factorArg), chunk = Number(chunkArg ?? 0), chunks = Number(chunksArg ?? 1);
let seed = Number(seedArg) + chunk * 7919;
const rnd = () => { seed = (seed * 1103515245 + 12345) % 2147483648; return seed / 2147483648; };
const ri = (n) => Math.floor(rnd() * n);
const pick = (a) => a[ri(a.length)];
const LUNAR = { chinese: [-30000, 67000], dangi: [-30000, 67000], islamic: [-2990, 5990], "islamic-rgsa": [-2990, 5990] };
const yr = (y) => { const d = new Date(0); d.setUTCFullYear(y, 0, 1); return d.getTime(); };
function instant(cal) {
  const r = rnd();
  let t;
  const lo = LUNAR[cal] ? yr(LUNAR[cal][0]) : -8.64e15, hi = LUNAR[cal] ? yr(LUNAR[cal][1]) : 8.64e15;
  if (r < 0.45) t = yr(1990) + rnd() * (yr(2040) - yr(1990));
  else if (r < 0.65) t = yr(1800) + rnd() * (yr(2200) - yr(1800));
  else if (r < 0.8) t = yr(-3000) + rnd() * (yr(9999) - yr(-3000));
  else t = lo + rnd() * (hi - lo);
  t = Math.floor(t);
  const m = rnd();
  if (m < 0.1) t -= t % DAY; else if (m < 0.2) t -= t % 3600000; else if (m < 0.3) t -= t % 1000;
  return Math.max(lo, Math.min(hi, t));
}
const files = {};
const write = (name, row) => { (files[name] ||= []).push(JSON.stringify(row)); };
const flat = (parts) => parts.map((p) => [p.type, p.value]);
const flatR = (parts) => parts.map((p) => [p.type, p.value, p.source]);
const err = (e) => e.constructor.name + ": " + e.message;

// ------------------------------------------------------------ calendars
const CALS = ["gregory", "buddhist", "japanese", "roc", "persian", "hebrew", "indian", "coptic", "ethiopic", "ethioaa", "islamic-civil", "islamic-tbla", "islamic-umalqura", "chinese", "dangi", "islamic", "islamic-rgsa"];
for (const cal of CALS) {
  const fm = {}, f = (z) => (fm[z] ||= new Intl.DateTimeFormat(`en-US-u-ca-${cal}`, { day: "numeric", month: "numeric", year: "numeric", era: "short", timeZone: z }));
  const ts = [];
  for (let i = 0; i < 900 * factor; i++) ts.push(instant(cal));
  if (!LUNAR[cal]) {
    for (const y of [-271821, -271820, -100000, -10000, -1000, -1, 0, 1, 2, 4, 100, 284, 285, 300, 400, 582, 600, 622, 623, 638, 639, 640, 645, 1000, 1500, 1582, 1600, 1868, 1900, 1911, 1912, 1913, 1988, 1989, 2000, 2019, 2024, 2025, 2100, 2400, 10000, 100000, 275760]) {
      for (const [mo, d] of [[1, 1], [3, 1], [3, 20], [3, 21], [4, 19], [4, 20], [9, 10], [9, 11], [9, 12], [9, 13], [12, 31]]) {
        const t = ms(y, mo, d);
        if (Math.abs(t) <= 8.64e15) ts.push(t);
      }
    }
    ts.push(-8.64e15, 8.64e15, -8.64e15 + DAY, 8.64e15 - 1, -8.64e15 + 1);
    for (let t = ms(1582, 10, 1); t <= ms(1582, 10, 20); t += DAY) ts.push(t);
  } else {
    // Year edges: walk a few New Years.
    for (const y of [1000, 1500, 1900, 1984, 2000, 2024, 2033, 2034, 2100, 3000]) for (let d = -4; d <= 80; d += 2) ts.push(ms(y, 1, 15) + d * DAY);
  }
  for (const t of ts) {
    const z = rnd() < 0.8 ? "UTC" : "Etc/GMT+1";
    let row;
    try { row = { c: cal, z, t, p: flat(f(z).formatToParts(t)) }; } catch (e) { row = { c: cal, z, t, e: err(e) }; }
    write("calendar", row);
  }
  for (const t of [NaN, Infinity, 8.64e15 + 1, -8.64e15 - 1]) { try { f("UTC").formatToParts(t); } catch (e) { write("calendar", { c: cal, z: "UTC", t: String(t), e: err(e) }); } }
}

// ------------------------------------------------------------ zones
function daysFromCivil(y, m, d) {
  if (m <= 2) y--;
  const era = Math.floor(y / 400), yoe = y - era * 400;
  const doy = Math.floor((153 * (m + (m > 2 ? -3 : 9)) + 2) / 5) + d - 1;
  return era * 146097 + yoe * 365 + Math.floor(yoe / 4) - Math.floor(yoe / 100) + doy - 719468;
}
const ids = fs.readFileSync(new URL("zone_candidates.txt", import.meta.url), "utf8").trim().split("\n");
for (const id of ids) {
  const f = new Intl.DateTimeFormat("en-US", { timeZone: id, hourCycle: "h23", era: "short", year: "numeric", month: "numeric", day: "numeric", hour: "numeric", minute: "numeric", second: "numeric" });
  const res = f.resolvedOptions().timeZone;
  const samples = [];
  const d = new Date(0);
  const off = (s) => {
    let era, y, mo, da, h, mi, se;
    for (const p of f.formatToParts(s * 1000)) switch (p.type) { case "era": era = p.value; break; case "year": y = +p.value; break; case "month": mo = +p.value; break; case "day": da = +p.value; break; case "hour": h = +p.value; break; case "minute": mi = +p.value; break; case "second": se = +p.value; break; }
    if (era === "BC") y = 1 - y;
    // Plain arithmetic: the wall clock can lie outside Date's range at the edges.
    return daysFromCivil(y, mo, da) * 86400 + h * 3600 + mi * 60 + se - s;
  };
  for (let i = 0; i < 30 * factor; i++) { const s = Math.floor(instant("gregory") / 1000); samples.push([s, off(s)]); }
  samples.push([-8640000000000, off(-8640000000000)], [8640000000000, off(8640000000000)]);
  write("tz", { id, res, s: samples });
}
write("tzids", { supported: Intl.supportedValuesOf("timeZone") });
for (const z of ["+01", "+0130", "+01:30", "-00:00", "+00:00", "+24:00", "+23:59", "-23:59", "+1", "utc", "UTC", "uTc", "Etc/UTC", "GMT", "Etc/GMT", "america/new_york", "AMERICA/NEW_YORK", "Asia/Calcutta", "Asia/Kolkata", "US/Eastern", "−01:00", "+01:00:00", "EST", "Etc/GMT+1", "etc/gmt-14", "Foo/Bar", "", "+99:00", "+00:60", "-12", "Antarctica/South_Pole", "UCT", "Zulu", "z", "Z"]) {
  try { write("tzres", { z, r: new Intl.DateTimeFormat("en-us", { timeZone: z }).resolvedOptions().timeZone }); } catch (e) { write("tzres", { z, e: err(e) }); }
}

// ------------------------------------------------------------ DateTimeFormat
const CTX = [...contexts().keys()];
function forms() {
  const out = [];
  for (const l of GRID.locales) out.push([l, null, null]);
  for (const [t, cs] of Object.entries(GRID.calendars)) for (const c of cs) { out.push([`${t}-u-ca-${c}`, null, null]); out.push([t, c, null]); }
  for (const [t, ns] of Object.entries(GRID.numberingSystems)) for (const v of ns) { out.push([t, null, v]); }
  return out;
}
const FORMS = forms();
// The committed vectors keep a share of the grid to stay small; a stress run (FACTOR > 1) covers all of it.
const KEEP = 0.55;
const hasNames = (b) => b.timeZoneName || b.timeStyle === "full" || b.timeStyle === "long";
const zoneFor = (b) => hasNames(b) ? pick(GRID.zones) : (rnd() < 0.5 ? "UTC" : pick(GRID.zones));
const calOf = (form) => new Intl.DateTimeFormat(form[0], form[1] ? { calendar: form[1] } : {}).resolvedOptions().calendar;
const STEP = [1, 250, 1000, 7000, 60000, 300000, 3600000, 4 * 3600000, 9 * 3600000, DAY, 3 * DAY, 20 * DAY, 45 * DAY, 200 * DAY, 400 * DAY, 3000 * DAY];
let idx = 0;
for (const form of FORMS) for (const bag of GRID.bags) {
  if (idx++ % chunks !== chunk) continue;
  const c = calOf(form);
  const reps = Math.max(1, Math.round(factor));
  for (let i = 0; i < reps; i++) {
    const o = { ...bag, timeZone: zoneFor(bag) };
    if (form[1]) o.calendar = form[1];
    if (form[2]) o.numberingSystem = form[2];
    let f;
    try { f = new Intl.DateTimeFormat(form[0], o); } catch (e) { write("dtf", { l: form[0], o, e: err(e) }); continue; }
    if (factor === 1 && rnd() > KEEP) continue;
    const t = instant(c);
    const row = { l: form[0], o, t };
    try { row.p = flat(f.formatToParts(t)); row.f = f.format(t); } catch (e) { row.e = err(e); }
    if (rnd() < 0.15 / Math.max(1, factor)) row.ro = f.resolvedOptions();
    write("dtf", row);
    if (rnd() < 0.5) {
      const b = t + pick(STEP) * (rnd() < 0.9 ? 1 : -1);
      if (Math.abs(b) <= 8.64e15) {
        const row2 = { l: form[0], o, a: t, b };
        try { row2.p = flatR(f.formatRangeToParts(t, b)); row2.f = f.formatRange(t, b); } catch (e) { row2.e = err(e); }
        write("range", row2);
      }
    }
  }
}
// Resolved options for every form (with and without hour options).
for (const form of FORMS) {
  for (const extra of [{}, { hour: "numeric" }, { hour: "numeric", hour12: true }, { hour: "numeric", hour12: false }, { hour: "numeric", hourCycle: "h23" }, { hourCycle: "h12" }]) {
    const o = { ...extra }; if (form[1]) o.calendar = form[1]; if (form[2]) o.numberingSystem = form[2];
    write("resolved", { l: form[0], o, ro: new Intl.DateTimeFormat(form[0], o).resolvedOptions() });
  }
}
// Constructor exceptions and resolution quirks.
const T = Date.UTC(2024, 5, 15, 13, 5, 9, 123);
const CASES = [
  [undefined, {}], ["en-us", {}], ["EN-US", {}], [["en-US", "de"], {}], [["xx-YY", "de"], {}], ["xx-YY", {}], [[], {}], ["x y", {}], ["en_US", {}], ["en-", {}], ["e", {}], [["de", "x y"], {}],
  ["en-US-u-ca-gregorian", {}], ["en-US-u-ca-foo", {}], ["en-US-u-nu-foo", {}], ["en-US-u-hc-h99", {}], ["en-US-u-ca-islamicc", {}], ["en-US-u-co-phonebk", {}], ["en-US-x-private", {}], ["de-u-nu-latn-ca-buddhist", {}],
  ["en-US", { calendar: "foo!" }], ["en-US", { calendar: "ab" }], ["en-US", { calendar: "foo" }], ["en-US", { calendar: "GREGORY" }], ["en-US", { calendar: "gregorian" }], ["en-US", { calendar: "islamicc" }], ["en-US", { calendar: "islamic" }], ["en-US", { calendar: "islamic-rgsa" }], ["en-US", { calendar: "ethiopic-amete-alem" }],
  ["en-US-u-ca-hebrew", { calendar: "gregory" }], ["en-US-u-ca-hebrew", { calendar: "hebrew" }], ["en-US-u-ca-hebrew", { calendar: "foo" }], ["en-US-u-nu-thai", { numberingSystem: "arab" }], ["en-US-u-nu-thai", { numberingSystem: "thai" }], ["en-US-u-nu-thai", { numberingSystem: "foo" }],
  ["en-US", { numberingSystem: "x" }], ["en-US", { numberingSystem: "foo" }], ["en-US", { numberingSystem: "ARAB" }],
  ["en-US", { hourCycle: "x" }], ["en-US", { timeZone: "x" }], ["en-US", { dateStyle: "full", year: "numeric" }], ["en-US", { timeStyle: "full", timeZoneName: "short" }], ["en-US", { dateStyle: "full", era: "short" }], ["en-US", { dateStyle: "x" }], ["en-US", { timeStyle: "x" }],
  ["en-US", { fractionalSecondDigits: 7 }], ["en-US", { fractionalSecondDigits: 0 }], ["en-US", { fractionalSecondDigits: 3.9 }], ["en-US", { fractionalSecondDigits: "2" }], ["en-US", { fractionalSecondDigits: "x" }], ["en-US", { fractionalSecondDigits: true }],
  ["en-US", { year: "x" }], ["en-US", { month: "x" }], ["en-US", { day: "x" }], ["en-US", { weekday: "x" }], ["en-US", { era: "x" }], ["en-US", { hour: "x" }], ["en-US", { minute: "x" }], ["en-US", { second: "x" }], ["en-US", { dayPeriod: "x" }], ["en-US", { timeZoneName: "x" }], ["en-US", { localeMatcher: "x" }], ["en-US", { formatMatcher: "x" }],
  ["en-US", { localeMatcher: "lookup" }], ["en-US", { formatMatcher: "basic" }], ["en-US", { formatMatcher: "best fit", hour: "numeric" }], ["en-US", { year: true }], ["en-US", { year: 1 }], ["en-US", { hour12: "yes", hour: "numeric" }], ["en-US", { hour12: 0, hour: "numeric" }], ["en-US", { hour12: true }], ["en-US", { hourCycle: "h23", year: "numeric" }],
  ["en-US", { hour12: true, hourCycle: "h23", hour: "numeric" }], ["en-US-u-hc-h23", { hour12: true, hour: "numeric" }], ["en-US-u-hc-h23", { hourCycle: "h12", hour: "numeric" }], ["en-US-u-hc-h23", { hourCycle: "h23", hour: "numeric" }], ["en-US-u-hc-h23", { hour: "numeric" }], ["en-US-u-hc-h23", { hour12: false, hour: "numeric" }],
  ["en-US", { timeZone: "America/New_York", hour: "numeric", timeZoneName: "short" }], ["en-US", { timeZone: "america/new_york", hour: "numeric", timeZoneName: "short" }], ["en-US", { timeZone: "US/Eastern", hour: "numeric" }], ["en-US", { timeZone: "+05:30", hour: "numeric", minute: "numeric", timeZoneName: "long" }], ["en-US", { timeZone: "+0530", hour: "numeric", minute: "numeric", timeZoneName: "long" }], ["en-US", { timeZone: "+01", hour: "numeric", timeZoneName: "short" }], ["en-US", { timeZone: "-00:00", hour: "numeric", timeZoneName: "short" }],
  ["iw", {}], ["en", { era: "short" }], ["en", { timeZoneName: "short" }], ["de-u-hc-h12", { hour12: true }], ["de-u-hc-h12", { hour12: true, hour: "numeric" }], ["de-u-hc-h12", { hourCycle: "h12" }], ["de-u-hc-h12", {}], ["", {}], ["en-US", { dateStyle: "full", hour12: true }],
];
for (const [l, o] of CASES) {
  let row = { l: l === undefined ? null : l, o };
  try { const f = new Intl.DateTimeFormat(l, o); row.ro = f.resolvedOptions(); row.t = T; row.p = flat(f.formatToParts(T)); row.f = f.format(T); } catch (e) { row.e = err(e); }
  write("ctor", row);
}
for (const t of [NaN, Infinity, -Infinity, 8.64e15 + 1, 1.5, -1.5, -0.5, 0.5]) {
  const f = new Intl.DateTimeFormat("en-US", { hour: "numeric", minute: "numeric", second: "numeric", fractionalSecondDigits: 3, timeZone: "UTC" });
  let row = { t: String(t) };
  try { row.f = f.format(t); row.p = flat(f.formatToParts(t)); } catch (e) { row.e = err(e); }
  try { row.r = f.formatRange(t, 1000); } catch (e) { row.re = err(e); }
  try { row.r2 = f.formatRange(1000, t); } catch (e) { row.re2 = err(e); }
  write("edges", row);
}

// ------------------------------------------------------------ DurationFormat
const UNITS = ["years", "months", "weeks", "days", "hours", "minutes", "seconds", "milliseconds", "microseconds", "nanoseconds"];
const record = (o = {}) => Object.fromEntries(UNITS.map((u) => [u, o[u] ?? 0]));
function randomDuration() {
  const o = {};
  const shape = ri(8);
  const big = () => pick([1, 2, 3, 5, 8, 10, 11, 12, 21, 22, 25, 59, 60, 100, 101, 999, 1000, 1234, 12345, 1000000, 4294967295]);
  const sub = () => pick([1, 5, 9, 10, 50, 99, 100, 123, 500, 999, 1000, 1234]);
  switch (shape) {
    case 0: for (const u of UNITS) if (rnd() < 0.5) o[u] = u === "milliseconds" || u === "microseconds" || u === "nanoseconds" ? sub() : big(); break;
    case 1: o.hours = ri(100); o.minutes = ri(60); o.seconds = ri(60); break;
    case 2: o.seconds = ri(60); o.milliseconds = ri(1000); o.microseconds = ri(1000); o.nanoseconds = ri(1000); break;
    case 3: o[pick(UNITS)] = big(); break;
    case 4: break;
    case 5: o.years = ri(5); o.months = ri(12); o.days = ri(31); break;
    case 6: o.hours = pick([1000000, 2 ** 40, 1234567]); o.minutes = ri(120); o.seconds = ri(4000); break;
    case 7: o.milliseconds = pick([999, 1000, 1001, 12345, 2 ** 40]); o.nanoseconds = pick([1, 999, 1000]); break;
  }
  if (rnd() < 0.15) for (const u of Object.keys(o)) if (o[u]) o[u] = -o[u];
  return record(o);
}
function randomOptions() {
  const o = {};
  const s = ri(10);
  if (s < 2) o.style = "long"; else if (s < 4) o.style = "narrow"; else if (s < 7) o.style = "digital"; else if (s < 8) o.style = "short";
  const r = rnd();
  if (r < 0.15) o.fractionalDigits = ri(10);
  if (rnd() < 0.3) { const u = pick(UNITS.slice(0, 4)); o[u] = pick(["long", "short", "narrow"]); }
  if (rnd() < 0.3) { const u = pick(UNITS.slice(4, 7)); o[u] = pick(["long", "short", "narrow", "numeric", "2-digit"]); }
  if (rnd() < 0.15) { const u = pick(UNITS.slice(7)); o[u] = pick(["long", "short", "narrow", "numeric"]); }
  if (rnd() < 0.25) { const u = pick(UNITS); o[u + "Display"] = pick(["auto", "always"]); }
  return o;
}
const dtags = GRID.durationLocales.map((t) => t);
for (let i = 0; i < 12000 * factor; i++) {
  const l = pick(dtags), o = randomOptions(), d = randomDuration();
  let row = { l, o, d };
  try { row.f = new Intl.DurationFormat(l, o).format(d); } catch (e) { row.e = err(e); }
  write("duration", row);
}
for (const l of dtags) for (const style of ["long", "short", "narrow", "digital"]) for (const d of [record({ years: 1, months: 2, weeks: 3, days: 4, hours: 5, minutes: 6, seconds: 7, milliseconds: 8, microseconds: 9, nanoseconds: 10 }), record({}), record({ hours: 1, minutes: 2 }), record({ years: 2, months: 3 }), record({ days: 5, hours: 21, seconds: 11 }), record({ minutes: -5 }), record({ seconds: 1, milliseconds: 500 })]) {
  write("duration", { l, o: { style }, d, f: new Intl.DurationFormat(l, { style }).format(d) });
}
for (const [l, o, d] of [
  ["en", { numberingSystem: "x" }, {}], ["en", { style: "foo" }, {}], ["en", { years: "numeric" }, {}], ["en", { hours: "numeric", minutes: "long" }, {}], ["en", { fractionalDigits: 10 }, {}], ["en", { fractionalDigits: -1 }, {}], ["en", { yearsDisplay: "x" }, {}], ["en", { style: "digital", seconds: "long" }, {}], ["en", { style: "digital", minutes: "long", seconds: "numeric" }, {}], ["en", { minutes: "numeric", seconds: "long" }, {}], ["en", { milliseconds: "numeric", microseconds: "long" }, {}], ["x y", {}, {}], ["en", { millisecondsDisplay: "always", milliseconds: "numeric" }, {}], ["en", { localeMatcher: "x" }, {}], ["en", { style: "digital", millisecondsDisplay: "always" }, {}], ["xx-YY", {}, { hours: 1 }], [undefined, {}, { hours: 1 }], [["en-US", "de"], {}, { hours: 1 }],
]) {
  let row = { l: l === undefined ? null : l, o, d: record(d) };
  try { row.f = new Intl.DurationFormat(l, o).format(record(d)); } catch (e) { row.e = err(e); }
  write("duration", row);
}

// PluralRules, which DurationFormat picks patterns with.
const PV = [...Array.from({ length: 300 }, (_, i) => i), 1000, 1001, 1000000, 1000001, 2000000, 1e7, 1e9];
for (const l of dtags) {
  const pr = new Intl.PluralRules(l);
  write("plural", { l, loc: pr.resolvedOptions().locale, v: PV, c: PV.map((v) => pr.select(v)) });
}

fs.mkdirSync(outDir, { recursive: true });
for (const [name, rows] of Object.entries(files)) {
  fs.writeFileSync(`${outDir}/${name}${chunks > 1 ? "_" + chunk : ""}.jsonl.gz`, zlib.gzipSync(rows.join("\n") + "\n", { level: 9 }));
  console.log(name, rows.length);
}
