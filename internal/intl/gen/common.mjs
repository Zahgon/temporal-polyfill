// Shared helpers for the extraction scripts. All of them run under the pinned Node 24.22.1.
export const OUT = process.env.OUT ?? "/tp/extract/out";
export const DAY = 86400000;
export const ms = (y, mo, d, h = 0, mi = 0, s = 0, mss = 0) => {
  const t = new Date(Date.UTC(2000, mo - 1, d, h, mi, s, mss));
  t.setUTCFullYear(y);
  return t.getTime();
};
const refs = {};
// ref gives the field keys of an instant (in UTC) that the Go layer's calendarFields reproduces.
export function ref(cal, t) {
  const lunar = cal === "chinese" || cal === "dangi";
  const f = (refs[cal] ||= new Intl.DateTimeFormat(`en-US-u-ca-${cal}-nu-latn`, {
    timeZone: "UTC", era: "short", year: "numeric", month: lunar ? "numeric" : "long", day: "numeric",
  }));
  const o = {};
  for (const p of f.formatToParts(t)) if (p.type !== "literal") o[p.type] = p.value;
  if (lunar) { o.year = o.relatedYear; o.month = o.month.replace(/^Mo/, ""); }
  return o;
}
export const hebrewLeap = (y) => ((((7 * y + 1) % 19) + 19) % 19) < 7;
export const cycleOf = (y) => ((((y - 1984) % 60) + 60) % 60);
export const NUMSYS = ["arab", "arabext", "beng", "deva", "fullwide", "hanidec", "latn", "thai"];
export const DIGITS = {};
for (const ns of NUMSYS) {
  const f = new Intl.NumberFormat(`en-u-nu-${ns}`, { useGrouping: false });
  DIGITS[ns] = Array.from({ length: 10 }, (_, i) => f.format(i));
}
export const loc = (ns, str) => str.replace(/[0-9]/g, (d) => (DIGITS[ns] || DIGITS.latn)[d]);
export const dpKey = (s) => `${s.h}:${s.mi === 0 ? 0 : "x"}:${s.s === 0 ? 0 : "x"}:${s.msPart === 0 ? 0 : "x"}`;
export const fieldsOf = (t) => {
  const d = new Date(t);
  return { t, wd: ((d.getUTCDay() + 6) % 7) + 1, h: d.getUTCHours(), mi: d.getUTCMinutes(), s: d.getUTCSeconds(), msPart: d.getUTCMilliseconds() };
};
export function keyFor(type, c, r, s) {
  switch (type) {
    case "month": return c === "hebrew" ? `${r.month}|${hebrewLeap(Number(r.year)) ? 1 : 0}` : r.month;
    case "day": return r.day;
    case "weekday": return String(s.wd);
    case "era": return (c === "iso8601" ? ref("gregory", s.t).era : r.era) ?? "";
    case "hour": return String(s.h);
    case "minute": return String(s.mi);
    case "second": return String(s.s);
    case "dayPeriod": return dpKey(s);
    case "yearName": return String(cycleOf(Number(r.year)));
    case "timeZoneName": return "UTC";
  }
  return "?";
}
export const yearRules = (ns, minus) => {
  const sign = (y) => (y.startsWith("-") ? minus : "");
  const abs = (y) => y.replace(/^-/, "");
  return {
    numeric: (y) => sign(y) + loc(ns, abs(y)),
    "2-digit": (y) => sign(y) + loc(ns, String(Number(abs(y)) % 100).padStart(2, "0")),
    jpan: (y) => (y === "1" ? "元" : sign(y) + loc(ns, abs(y))),
    pad2: (y) => sign(y) + loc(ns, abs(y).padStart(2, "0")),
  };
};
export const minusOf = (ctx, ns) => new Intl.NumberFormat(ctx, { numberingSystem: ns, useGrouping: false }).format(-1).replace(DIGITS[ns]?.[1] ?? "1", "");
export const err = (e) => [e.constructor.name, e.message];
export const isLunar = (c) => c === "chinese" || c === "dangi";
// What a part shows as its "year": ICU's chinese y is the year of the 60-year cycle, r the related year.
export const yearValue = (type, c, r) => (type === "year" && isLunar(c) ? String(cycleOf(Number(r.year)) + 1) : r.year);
export const cycleNumber = (y) => Math.floor((y - 1984) / 60);
