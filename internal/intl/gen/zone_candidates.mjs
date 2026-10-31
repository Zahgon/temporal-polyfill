// Every string in the Node binary that Intl accepts as a time zone identifier, plus the tz names of
// the image. `node zone_candidates.mjs OUT.txt`. The binary holds zoneinfo64 names as UTF-16 and
// ASCII runs; this finds ids no list of tz names has.
import fs from "node:fs";
const bin = fs.readFileSync(process.execPath);
const cand = new Set();
const isId = (c) => (c >= 48 && c <= 57) || (c >= 65 && c <= 90) || (c >= 97 && c <= 122) || c === 47 || c === 43 || c === 45 || c === 95;
let run = "";
for (let i = 0; i < bin.length; i += 1) { const c = bin[i]; if (isId(c)) { run += String.fromCharCode(c); } else { if (run.length >= 2 && run.length <= 40 && /^[A-Z]/.test(run)) cand.add(run); run = ""; } }
for (const off of [0, 1]) { run = ""; for (let i = off; i + 1 < bin.length; i += 2) { const c = bin[i], d = bin[i + 1]; if (d === 0 && isId(c)) { run += String.fromCharCode(c); } else { if (run.length >= 2 && run.length <= 40 && /^[A-Z]/.test(run)) cand.add(run); run = ""; } } }
console.log("raw candidates", cand.size);
const ok = [];
for (const z of cand) {
  if (/^[A-Z][a-z]{2,}$/.test(z) && z.length > 12) continue;
  try { new Intl.DateTimeFormat("en-US", { timeZone: z }); ok.push(z); } catch {}
}
for (const z of Intl.supportedValuesOf("timeZone")) if (!ok.includes(z)) ok.push(z);
// The tz names shipped with Go: short link names such as Eire do not stand out in the binary.
for (const z of fs.readFileSync(new URL("zone_names.txt", import.meta.url), "utf8").trim().split("\n")) {
  if (ok.includes(z)) continue;
  try { new Intl.DateTimeFormat("en-US", { timeZone: z }); ok.push(z); } catch {}
}
ok.sort();
fs.writeFileSync(process.argv[2], ok.join("\n") + "\n");
console.log("accepted", ok.length);
