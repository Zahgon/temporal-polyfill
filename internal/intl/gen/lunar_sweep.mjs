// Day-by-day record of a calendar between two epoch days: runs [firstDay, year, monthLabel, firstDayOfMonth]
// of days whose month is unchanged and whose day of month increases by one; a thrown day is [day, "E", message].
//   node lunar_sweep.mjs CALENDAR FROM_DAY TO_DAY OUT.json
import fs from "node:fs";
const [, , cal, from, to, outFile] = process.argv;
const DAY = 86400000;
const f = new Intl.DateTimeFormat(`en-US-u-ca-${cal}`, { day: "numeric", month: "numeric", year: "numeric", era: "short", timeZone: "UTC" });
const runs = [];
let py = null, pm = null, pd = 0;
for (let d = Number(from); d < Number(to); d++) {
  let y, m, dom;
  try {
    for (const p of f.formatToParts(d * DAY)) {
      if (p.type === "year" || p.type === "relatedYear") y = p.value;
      else if (p.type === "month") m = p.value;
      else if (p.type === "day") dom = +p.value;
    }
  } catch (e) { runs.push([d, "E", e.message]); py = null; continue; }
  if (m === undefined) { runs.push([d, "E", "no month: " + JSON.stringify(f.formatToParts(d * DAY))]); py = null; continue; }
  if (y !== py || m !== pm || dom !== pd + 1) runs.push([d, +y, m.replace(/^Mo/, ""), dom]);
  py = y; pm = m; pd = dom;
}
fs.writeFileSync(outFile, JSON.stringify(runs));
console.log(cal, from, to, "runs", runs.length);
