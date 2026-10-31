// The locale x option-bag x zone x calendar grid the DateTimeFormat and DurationFormat tables cover.
// `node grid.mjs OUT.json` writes icu_grid.json; the other scripts import GRID.
import fs from "node:fs";

const n = "numeric", d2 = "2-digit", s = "short", l = "long", N = "narrow";
const y = "year", mo = "month", d = "day", wd = "weekday", h = "hour", mi = "minute", se = "second";
const ymd = { [y]: n, [mo]: n, [d]: n };
const hms = { [h]: n, [mi]: n, [se]: n };
const STYLES = ["full", "long", "medium", "short"];
const TZN = [s, l, "shortOffset", "longOffset", "shortGeneric", "longGeneric"];

const bags = [
  // What the polyfill's amend functions produce by default.
  { ...ymd }, { ...hms }, { ...ymd, ...hms }, { [y]: n, [mo]: n }, { [mo]: n, [d]: n },
  { ...ymd, ...hms, timeZoneName: s },
  // The dateStyle and timeStyle hacks of yearMonthAmend, monthDayAmend, timeAmend and datetimeAmend.
  { [y]: d2, [mo]: n }, { [y]: n, [mo]: s }, { [y]: n, [mo]: l },
  { [mo]: s, [d]: n }, { [mo]: l, [d]: n },
  { [h]: n, [mi]: d2, [se]: d2 },
  { ...ymd, [h]: n, [mi]: d2, [se]: d2 }, { [y]: n, [mo]: s, [d]: n, [h]: n, [mi]: d2, [se]: d2 },
  { [y]: n, [mo]: l, [d]: n, [h]: n, [mi]: d2, [se]: d2 },
  { [y]: n, [mo]: l, [d]: n, [wd]: l, [h]: n, [mi]: d2, [se]: d2 },
  // Styles and style pairs.
  ...STYLES.map((x) => ({ dateStyle: x })), ...STYLES.map((x) => ({ timeStyle: x })),
  ...STYLES.flatMap((x) => STYLES.map((t) => ({ dateStyle: x, timeStyle: t }))),
  // Single fields.
  { [y]: n }, { [y]: d2 }, { [mo]: n }, { [mo]: d2 }, { [mo]: s }, { [mo]: l }, { [mo]: N },
  { [d]: n }, { [d]: d2 }, { [wd]: s }, { [wd]: l }, { [wd]: N }, { era: s }, { era: l }, { era: N },
  { [h]: n }, { [h]: d2 }, { [mi]: n }, { [mi]: d2 }, { [se]: n }, { [se]: d2 },
  { dayPeriod: s }, { dayPeriod: l }, { dayPeriod: N },
  { fractionalSecondDigits: 1 }, { fractionalSecondDigits: 2 }, { fractionalSecondDigits: 3 },
  ...TZN.map((x) => ({ timeZoneName: x })),
  // Preset field combinations.
  { [y]: n, [mo]: s, [d]: n }, { [y]: n, [mo]: l, [d]: n }, { [y]: n, [mo]: l, [d]: n, [wd]: l },
  { [y]: n, [mo]: s, [d]: n, [wd]: s }, { [h]: n, [mi]: n }, { [h]: d2, [mi]: d2 }, { [h]: n, [mi]: d2 },
  { [h]: d2, [mi]: d2, [se]: d2 }, { [mi]: d2, [se]: d2 }, { [y]: n, [mo]: d2, [d]: d2 }, { [mo]: d2, [d]: d2 },
  { [y]: d2, [mo]: d2, [d]: d2 }, { era: s, [y]: n }, { era: l, [y]: n, [mo]: l, [d]: n },
  { era: s, ...ymd }, { era: N, [y]: n, [mo]: s, [d]: n }, { [wd]: s, [h]: n }, { [wd]: l, [mo]: l, [d]: n },
  { [wd]: N, [d]: n }, { [d]: n, [h]: n }, { [y]: n, [mo]: l, [d]: n, ...hms }, { [y]: n, [mo]: s, [d]: n, [h]: n, [mi]: n },
  // The polyfill's offset formatter.
  { era: s, ...ymd, ...hms, hour12: false },
  // hour12 and hourCycle.
  ...[{ [h]: n }, { [h]: n, [mi]: n }, { ...hms }, { ...ymd, ...hms }, { timeStyle: "short" }, { timeStyle: "medium" }, { dateStyle: "medium", timeStyle: "short" }]
    .flatMap((b) => [{ hour12: true }, { hour12: false }, ...["h11", "h12", "h23", "h24"].map((c) => ({ hourCycle: c }))].map((o) => ({ ...b, ...o }))),
  // dayPeriod with an hour.
  { [h]: n, dayPeriod: s }, { [h]: n, dayPeriod: l }, { [h]: n, dayPeriod: N }, { [h]: n, [mi]: n, dayPeriod: s }, { [h]: n, [mi]: d2, dayPeriod: l },
  // Fractional seconds.
  { [se]: n, fractionalSecondDigits: 1 }, { [se]: n, fractionalSecondDigits: 2 }, { [se]: n, fractionalSecondDigits: 3 },
  { [mi]: d2, [se]: d2, fractionalSecondDigits: 3 },
  { ...hms, fractionalSecondDigits: 1 }, { ...hms, fractionalSecondDigits: 2 }, { ...hms, fractionalSecondDigits: 3 },
  { ...ymd, ...hms, fractionalSecondDigits: 3 },
  // Zone names with times.
  ...TZN.map((x) => ({ [h]: n, [mi]: n, timeZoneName: x })),
  ...TZN.filter((x) => x !== s).map((x) => ({ ...ymd, ...hms, timeZoneName: x })),
  { [h]: n, timeZoneName: s }, { [wd]: s, ...ymd, timeZoneName: l },
];

const CAL = ["gregory", "buddhist", "japanese", "roc", "hebrew", "persian", "indian", "coptic", "ethiopic", "islamic-civil", "islamic-umalqura", "chinese", "dangi"];
const calendars = {
  "en-US": [...CAL, "ethioaa", "islamic-tbla", "iso8601", "islamic", "islamic-rgsa"], de: [...CAL, "iso8601"], "ar-SA": [...CAL, "islamic-tbla", "islamic", "islamic-rgsa"], "th-TH": CAL,
  "ja-JP": [...CAL, "iso8601"], he: CAL, "fa-IR": CAL,
  "zh-CN": ["chinese"], "zh-TW": ["roc", "chinese"], "ko-KR": ["dangi"], "hi-IN": ["indian"], "ar-EG": ["coptic"],
  ar: ["islamic-civil", "islamic-umalqura"], fr: ["persian"], "en-GB": ["buddhist"], ru: ["coptic"],
};

const base = ["en-US", "en", "en-GB", "fr", "fr-CA", "de", "es", "es-MX", "it", "pt-BR", "nl", "sv", "pl", "ru", "uk", "el", "tr", "he", "ar", "ar-EG", "ar-SA", "fa-IR", "hi-IN", "bn-BD", "th-TH", "ja-JP", "zh-CN", "zh-TW", "ko-KR", "vi"];
const nuTags = ["en-US-u-nu-arab", "en-US-u-nu-deva", "en-US-u-nu-thai", "en-US-u-nu-fullwide", "de-u-nu-arab", "ar-EG-u-nu-latn", "ar-SA-u-nu-latn", "fa-IR-u-nu-latn", "hi-IN-u-nu-deva", "th-TH-u-nu-thai", "bn-BD-u-nu-latn", "zh-CN-u-nu-hanidec"];
const hcTags = ["en-US-u-hc-h23", "en-US-u-hc-h11", "de-u-hc-h12", "ja-JP-u-hc-h11", "fr-u-hc-h12", "ar-EG-u-hc-h23"];

export const GRID = {
  locales: [...base, ...nuTags, ...hcTags],
  bags,
  zones: ["UTC", "Etc/UTC", "GMT", "Etc/GMT+1", "America/New_York", "America/Los_Angeles", "America/Sao_Paulo", "America/St_Johns", "Europe/London", "Europe/Paris", "Europe/Moscow", "Africa/Cairo", "Asia/Kolkata", "Asia/Calcutta", "Asia/Tokyo", "Asia/Shanghai", "Australia/Sydney", "Australia/Lord_Howe", "Pacific/Auckland", "Pacific/Apia", "+00:00", "+01:00", "-05:00", "+05:30", "-03:30", "+05:45", "+13:00", "-12:00"],
  // Per locale, calendars formatted both through -u-ca- and through the calendar option.
  calendars,
  // Per locale, numbering systems given through the numberingSystem option.
  numberingSystems: { "en-US": ["arab", "deva", "thai", "fullwide"], de: ["arab"], "ar-EG": ["latn"], "fa-IR": ["latn"], "th-TH": ["thai"], "ja-JP": ["fullwide"] },
  // Per locale, hour cycles given through a -u-hc- tag are in locales; hourCycle options are in bags.
  durationLocales: [...base, "en-US-u-nu-arab", "en-US-u-nu-deva", "en-US-u-nu-thai", "en-US-u-nu-fullwide", "ar-EG-u-nu-latn", "fa-IR-u-nu-latn", "hi-IN-u-nu-deva", "th-TH-u-nu-thai"],
};

// A content context is what a formatter's text depends on besides the bag: the base tag, the resolved
// calendar and numbering system, and a -u-hc- keyword. It is formatted through REPR(ctx).
export const baseTag = (t) => t.split("-u-")[0];
export function uKey(t, k) { const m = new RegExp(`-u(?:-[a-z0-9]{2,8})*?-${k}-([a-z0-9]+(?:-[a-z0-9]{3,8})*)`).exec(t); return m ? m[1] : null; }
export function contexts() {
  const out = new Map();
  const add = (base, ca, nu, hc) => {
    const parts = []; if (ca) parts.push("ca-" + ca); if (hc) parts.push("hc-" + hc); if (nu) parts.push("nu-" + nu);
    const tag = parts.length ? `${base}-u-${parts.join("-")}` : base;
    const ro = new Intl.DateTimeFormat(tag).resolvedOptions();
    const key = `${base}|${ro.calendar}|${ro.numberingSystem}|${hc ?? ""}`;
    if (!out.has(key)) out.set(key, tag);
  };
  for (const t of GRID.locales) add(baseTag(t), uKey(t, "ca"), uKey(t, "nu"), uKey(t, "hc"));
  for (const [t, cs] of Object.entries(GRID.calendars)) for (const c of cs) add(t, c, null, null);
  for (const [t, ns] of Object.entries(GRID.numberingSystems)) for (const v of ns) add(t, null, v, null);
  return out;
}
export const bagKey = (bag) => JSON.stringify(Object.keys(bag).sort().map((k) => [k, bag[k]]));

if (process.argv[1].endsWith("grid.mjs")) {
  const seen = new Set();
  for (const b of bags) { const k = bagKey(b); if (seen.has(k)) throw new Error("duplicate bag " + k); seen.add(k); }
  fs.writeFileSync(process.argv[2], JSON.stringify(GRID, null, 1));
  console.log("locales", GRID.locales.length, "bags", bags.length, "zones", GRID.zones.length, "contexts", contexts().size, "durationLocales", GRID.durationLocales.length);
}
