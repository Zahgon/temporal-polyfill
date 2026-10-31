#!/usr/bin/env python3
"""Turn the raw ICU extraction (extract/out) into the compact tables the Go layer embeds:
data/{tz,cal,names,locale,duration}.json.gz, data/loc_<language>.json.gz and data/lunar_*.bin.gz.

Usage: mkdata.py OUT_DIR DATA_DIR
"""
import collections
import datetime
import glob
import gzip
import json
import os
import re
import sys

EX, OUT = sys.argv[1], sys.argv[2]


def load(p):
    with open(p, encoding="utf-8") as f:
        return json.load(f)


def dump(name, obj):
    os.makedirs(OUT, exist_ok=True)
    raw = json.dumps(obj, ensure_ascii=False, separators=(",", ":"), sort_keys=True).encode()
    with gzip.GzipFile(os.path.join(OUT, name + ".json.gz"), "wb", mtime=0, compresslevel=9) as f:
        f.write(raw)
    return len(raw)


def fail(msg):
    print("ERROR:", msg, file=sys.stderr)
    sys.exit(1)


# ---------------------------------------------------------------- time zones
Y2100 = int(datetime.datetime(2100, 1, 1, tzinfo=datetime.timezone.utc).timestamp())


def days_from_civil(y, m, d):
    y -= m <= 2
    era = (y if y >= 0 else y - 399) // 400
    yoe = y - era * 400
    doy = (153 * (m + (-3 if m > 2 else 9)) + 2) // 5 + d - 1
    doe = yoe * 365 + yoe // 4 - yoe // 100 + doy
    return era * 146097 + doe - 719468


def civil_from_days(z):
    z += 719468
    era = (z if z >= 0 else z - 146096) // 146097
    doe = z - era * 146097
    yoe = (doe - doe // 1460 + doe // 36524 - doe // 146096) // 365
    y = yoe + era * 400
    doy = doe - (365 * yoe + yoe // 4 - yoe // 100)
    mp = (5 * doy + 2) // 153
    d = doy - (153 * mp + 2) // 5 + 1
    m = mp + (3 if mp < 10 else -9)
    return y + (m <= 2), m, d


def rule_transitions(rule, year, prev_off):
    out = []
    off = prev_off
    for month, day, wd, t, to in rule:
        days = days_from_civil(year, month, day)
        if wd >= 0:
            cur = (days + 4) % 7  # 1970-01-01 was a Thursday (4)
            days += (wd - cur) % 7
        out.append((days * 86400 + t - off, to))
        off = to
    return out


def fit_rule(trans, last_off):
    """Fit [(month, day, weekday|-1, wallSecondsInOldOffset, toOffset)] x2 to the 2100..2500 transitions."""
    trans = [tuple(x) for x in trans]
    if not trans:
        return None
    prev = last_off
    rows = []
    for t, to in trans:
        days, tod = divmod(t + prev, 86400)
        y, m, d = civil_from_days(days)
        rows.append((y, m, d, (days + 4) % 7, tod, to))
        prev = to
    classes = collections.OrderedDict()
    for r in rows:
        classes.setdefault((r[1], r[5]), []).append(r)
    if len(classes) != 2:
        return None
    rule = []
    for (m, to), rs in classes.items():
        tods = {r[4] for r in rs}
        wds = {r[3] for r in rs}
        days = [r[2] for r in rs]
        if len(tods) != 1:
            return None
        if len(set(days)) == 1:
            rule.append([m, days[0], -1, tods.pop(), to])
        elif len(wds) == 1 and max(days) - min(days) <= 6:
            rule.append([m, min(days), wds.pop(), tods.pop(), to])
        else:
            return None
    rule.sort(key=lambda r: (r[0], r[1]))
    regen = []
    prev = last_off
    for y in range(2100, 2500):
        for sec, to in rule_transitions(rule, y, prev):
            regen.append((sec, to))
            prev = to
    regen = [x for x in regen if x[0] >= Y2100]
    if regen[: len(trans)] != trans:
        return None
    return rule


def build_tz():
    ids = load(os.path.join(EX, "zone_ids.json"))
    raw = {}
    for p in glob.glob(os.path.join(EX, "tz_*.json")):
        raw.update(load(p))
    accepted = {z.lower(): r for z, r in ids["resolved"].items()}
    zones = {}
    nofit = []
    for c, b in raw.items():
        if b["before"]:
            fail(f"{c}: transitions before 1800")
        if b["periodicBad"]:
            fail(f"{c}: offsets after 2500 are not 400-year periodic: {b['periodicBad'][:3]}")
        z = {"first": b["first"], "trans": b["trans"]}
        if b["after"]:
            last = b["trans"][-1][1] if b["trans"] else b["first"]
            rule = fit_rule(b["after"], last)
            if rule is None:
                nofit.append(c)
                z["cycle"] = b["after"]
            else:
                z["rule"] = rule
        zones[c] = z
    n = dump("tz", {"ids": accepted, "supported": ids["supported"], "zones": zones, "y2100": Y2100})
    print(f"tz: ids {len(accepted)} zones {len(zones)} rules {sum('rule' in z for z in zones.values())} raw cycles {nofit}; {n} bytes")


# ---------------------------------------------------------------- lunar tables
def label_parts(label):
    return int(label.replace("bis", "")), label.endswith("bis")


def write_lunar(name, cal, runs, era):
    """runs: sorted [firstDay, year, label, firstDom] or [day, 'E', message]."""
    months = []  # [start, year, label]
    exceptions = []
    end = runs[-1][0] + 1

    def next_in_seq(y0, l0, y1, l1):
        m0, lp0 = label_parts(l0)
        m1, lp1 = label_parts(l1)
        if y1 == y0:
            return (m1 == m0 and lp1 and not lp0) or (m1 == m0 + 1 and not lp1)
        return y1 == y0 + 1 and m1 == 1 and not lp1 and m0 >= 12

    pending = []
    for i, r in enumerate(runs):
        stop = runs[i + 1][0] if i + 1 < len(runs) else None
        if r[1] == "E":
            exceptions.append([r[0], r[2]])
            continue
        first, y, label, dom = r
        start = first - dom + 1
        if not months:
            if dom != 1:
                continue
            months.append([start, y, label])
            continue
        cy, cl, cs = months[-1][1], months[-1][2], months[-1][0]
        if y == cy and label == cl and start == cs:
            continue
        if next_in_seq(cy, cl, y, label) and start > cs:
            months.append([start, y, label])
            continue
        # A glitch: record its days.
        if stop is None:
            continue
        for d in range(first, stop):
            exceptions.append([d, y, label, dom + (d - first)])
    # Trim to whole years.
    while label_parts(months[0][2]) != (1, False):
        months.pop(0)
    last_new_year = max(i for i, m in enumerate(months) if m[2] == "1")
    table_end = months[last_new_year][0]
    months = months[: last_new_year + 1]
    first_year = months[0][1]
    years = []
    leap_bytes = bytearray()
    lengths = []
    for i in range(len(months) - 1):
        lengths.append(months[i + 1][0] - months[i][0])
    y = first_year
    cur = []
    for i, m in enumerate(months[:-1]):
        if m[2] == "1" and cur:
            years.append(cur)
            cur = []
        cur.append(m)
    years.append(cur)
    bits = bytearray((len(lengths) + 7) // 8)
    k = 0
    for yi, ms_ in enumerate(years):
        if ms_[0][1] != first_year + yi:
            fail(f"{cal}: year sequence broken at {ms_[0]}")
        labels = [m[2] for m in ms_]
        lp = 0
        if len(ms_) == 13:
            lp = next(j for j, l in enumerate(labels) if l.endswith("bis"))
        elif len(ms_) != 12:
            fail(f"{cal}: year with {len(ms_)} months: {ms_}")
        expected = []
        for j in range(len(ms_)):
            if lp and j >= lp:
                expected.append(f"{j}bis" if j == lp else str(j))
            else:
                expected.append(str(j + 1))
        if labels != expected:
            fail(f"{cal}: unexpected month labels {labels} in year {ms_[0][1]}")
        leap_bytes.append(lp)
        for _ in ms_:
            ln = lengths[k]
            if ln not in (29, 30):
                fail(f"{cal}: month of {ln} days at {months[k]}")
            if ln == 30:
                bits[k >> 3] |= 1 << (k & 7)
            k += 1
    exceptions = [e for e in exceptions if months[0][0] <= e[0] < table_end]
    fname = f"lunar_{name}.bin.gz"
    with gzip.GzipFile(os.path.join(OUT, fname), "wb", mtime=0, compresslevel=9) as f:
        f.write(bytes(leap_bytes) + bytes(bits))
    size = os.path.getsize(os.path.join(OUT, fname))
    print(f"{name}: days {months[0][0]}..{table_end} years {first_year}..{first_year + len(years) - 1} months {len(months) - 1} exceptions {len(exceptions)} {size} bytes gz")
    # Check the decode against every run.
    meta = {"file": fname, "firstDay": months[0][0], "endDay": table_end, "firstYear": first_year, "years": len(years), "era": era, "exceptions": exceptions}
    return meta


def read_sweep(cal):
    runs = []
    ps = sorted(glob.glob(os.path.join(EX, f"sweep_{cal}_*.json")), key=lambda p: int(p.rsplit("_", 1)[1].split(".")[0]))
    for p in ps:
        rs = load(p)
        if runs and rs and rs[0][1] != "E" and runs[-1][1] != "E":
            # A chunk starts a new run; drop it when it continues the last one.
            prev = runs[-1]
            if prev[1] == rs[0][1] and prev[2] == rs[0][2] and rs[0][3] == prev[3] + (rs[0][0] - prev[0]):
                rs = rs[1:]
        runs.extend(rs)
    return runs


def build_cal():
    C = load(os.path.join(EX, "cal.json"))
    lunar = {}
    for cal in ("chinese", "dangi"):
        runs = read_sweep(cal)
        if not runs:
            fail(f"no sweep for {cal}")
        lunar[cal] = write_lunar(cal, cal, runs, "")
    a, b = read_sweep("islamic"), read_sweep("islamic-rgsa")
    if a != b:
        fail("islamic and islamic-rgsa differ")
    lunar["islamic"] = write_lunar("islamic", "islamic", a, "AH")
    n = dump("cal", {"japaneseEras": C["japaneseEras"], "umalqura": C["months_islamic-umalqura"], "persianStarts": C["persianStarts"], "lunar": lunar})
    print(f"cal: {n} bytes")


# ---------------------------------------------------------------- zone names
def build_names():
    vectors, index, zones = [], {}, {}
    contexts = None
    for p in sorted(glob.glob(os.path.join(EX, "names_*.json"))):
        d = load(p)
        if contexts is None:
            contexts = d["contexts"]
        elif contexts != d["contexts"]:
            fail("name contexts differ between chunks")
        remap = {}
        for i, v in enumerate(d["vectors"]):
            k = json.dumps(v, ensure_ascii=False)
            if k not in index:
                index[k] = len(vectors)
                vectors.append(v)
            remap[i] = index[k]
        for z, r in d["zones"].items():
            zones[z] = {"periods": [[t, remap[v]] for t, v in r["periods"]],
                        "far": [[int(ms // 1000), remap[v]] for ms, v in r["far"]]}
    strings, sidx = [], {}

    def s(x):
        if x is None:
            return -1
        if x not in sidx:
            sidx[x] = len(strings)
            strings.append(x)
        return sidx[x]

    flat = [[s(n) for slots in v for n in slots] for v in vectors]
    n = dump("names", {"contexts": contexts, "strings": strings, "vectors": flat, "zones": zones})
    print(f"names: zones {len(zones)} vectors {len(vectors)} strings {len(strings)}; {n} bytes")
    return contexts, vectors, zones


PROBE_ZONES = ["UTC", "America/New_York", "Asia/Kolkata", "Europe/Paris", "Asia/Tokyo", "+05:30"]
PROBE_TIMES = [int(datetime.datetime(2024, 1, 15, 12, tzinfo=datetime.timezone.utc).timestamp()),
               int(datetime.datetime(2024, 7, 15, 12, tzinfo=datetime.timezone.utc).timestamp())]
STYLES = ["short", "long", "shortOffset", "longOffset", "shortGeneric", "longGeneric"]


def vector_at(zone_rec, sec):
    v = None
    for t, x in zone_rec["periods"]:
        if t <= sec:
            v = x
        else:
            break
    return v


def zone_slot(names, name_ctx, style_guess, probe, problems, timed=None):
    contexts, vectors, zones = names
    if name_ctx not in contexts:
        problems["tzctx"] += 1
        return -1
    ci = contexts.index(name_ctx)
    vals = []
    if timed is not None:
        for z, t, value in timed:
            vals.append((value, vectors[vector_at(zones[z], t)][ci]))
    else:
        for i, z in enumerate(PROBE_ZONES):
            for j, t in enumerate(PROBE_TIMES):
                vals.append((probe[i * 2 + j], vectors[vector_at(zones[z], t)][ci]))
    candidates = [s for s in range(6) if all(p == v[s] for p, v in vals)]
    if not candidates:
        problems["tzslot"] += 1
        return -1
    if style_guess in candidates:
        return style_guess
    return candidates[0]


# ---------------------------------------------------------------- DateTimeFormat templates
DATE_KEYS = {"weekday", "year", "month", "day"}
TIME_KEYS = {"dayPeriod", "hour", "minute", "second", "fractionalSecondDigits"}
STYLE_KEYS = {"dateStyle", "timeStyle"}


def runtime_key(bag):
    """The key readDTFOptions computes for a bag."""
    pairs = dict(json.loads(bag))
    if not set(pairs) & (DATE_KEYS | TIME_KEYS | STYLE_KEYS):
        pairs.update(year="numeric", month="numeric", day="numeric")
    if "hour" not in pairs and "timeStyle" not in pairs:
        pairs.pop("hour12", None)
        pairs.pop("hourCycle", None)
    return json.dumps(sorted(pairs.items()), ensure_ascii=False, separators=(",", ":"))


def derivable(big, small):
    """Whether a part structure comes from the larger one by dropping parts and merging literals."""
    # A subsequence of fields, with the literals merged, gives the same text between the kept fields.
    want = [p[0] for p in small if p[0] != "literal"]
    out, i = [], 0
    lit = ""
    for p in big:
        if p[0] == "literal":
            lit += p[1]
        elif i < len(want) and p[0] == want[i]:
            if lit:
                out.append(lit)
            lit = ""
            out.append(p[0])
            i += 1
    if lit:
        out.append(lit)
    norm = []
    for p in small:
        norm.append(p[1] if p[0] == "literal" else p[0])
    return i == len(want) and out == norm


def build_dtf(names):
    T = {}
    for p in sorted(glob.glob(os.path.join(EX, "tpl_*.json"))):
        T.update(load(p))
    by_lang = collections.defaultdict(dict)
    for key, r in T.items():
        lang = key.split("|")[0].split("-")[0]
        by_lang[lang][key] = r
    problems = collections.Counter()
    examples = collections.defaultdict(list)
    total = 0

    def note(kind, ex):
        problems[kind] += 1
        if len(examples[kind]) < 6:
            examples[kind].append(ex)

    for lang, ctxs in sorted(by_lang.items()):
        pools = {k: ([], {}) for k in ("lists", "templates", "rangeLists", "rangeTemplates")}

        def intern(pool, obj):
            items, idx = pools[pool]
            k = json.dumps(obj, ensure_ascii=False, sort_keys=True)
            if k not in idx:
                idx[k] = len(items)
                items.append(obj)
            return idx[k]

        contexts, rcontexts = {}, {}
        for ctx, r in ctxs.items():
            base, cal, nu, hc = ctx.split("|")
            name_ctx = f"{base}|{nu}"
            c = {"bags": {}}
            rc = {}
            for bag, t in r["templates"].items():
                key = runtime_key(bag)
                if "error" in t:
                    c["bags"][key] = {"throw": t["error"]}
                    continue
                if t.get("conflicts"):
                    note("conflict", (ctx, bag, t["conflicts"][:2]))
                if t.get("errors"):
                    note("sampleError", (ctx, bag, t["errors"][:1]))
                structs = t["structs"]
                parts = max(structs, key=len)
                for s in structs:
                    if not derivable(parts, s):
                        note("struct", (ctx, bag, s, parts))
                bagd = dict(json.loads(bag))
                tpl = {"ro": t["ro"], "parts": parts, "minus": t["minus"],
                       "lists": {k: intern("lists", v) for k, v in t["lists"].items() if k != "timeZoneName"}}
                if "yearRules" in t:
                    rules = {}
                    for ty, rule in t["yearRules"].items():
                        if rule is None:
                            if cal == "hebrew" and base == "he":
                                rule = "hebr"
                            else:
                                note("yearRule", (ctx, bag, ty, t.get("yearMiss", {}).get(ty, [])[:4]))
                                rule = "numeric"
                        rules[ty] = rule
                    tpl["years"] = rules
                if "fracOK" in t:
                    if not t["fracOK"]:
                        note("frac", (ctx, bag))
                    tpl["frac"] = bagd.get("fractionalSecondDigits")
                if "timeZoneName" in t["lists"]:
                    guess = STYLES.index(bagd["timeZoneName"]) if "timeZoneName" in bagd else (1 if bagd.get("timeStyle") == "full" else 0)
                    tpl["tz"] = zone_slot(names, name_ctx, guess, t["tzProbe"], problems)
                    if tpl["tz"] < 0 and len(examples["tzslotx"]) < 6:
                        examples["tzslotx"].append((ctx, bag, t["tzProbe"]))
                c["bags"][key] = intern("templates", tpl)
                lv = r["ranges"].get(bag, {})
                rb = {}
                for level, x in lv.items():
                    if not x["shapes"]:
                        rb[level] = {"collapsed": True}
                        continue
                    shape = max(x["shapes"], key=len)
                    if x.get("conflicts"):
                        note("rangeConflict", (ctx, bag, level, x["conflicts"][:2]))
                    rt = {"shape": shape, "minus": x["minus"],
                          "lists": {k: intern("rangeLists", v) for k, v in x["lists"].items() if not k.startswith("timeZoneName|")}}
                    if x["years"]:
                        ys = {}
                        for k, oks in x["years"].items():
                            ty = k.split("|")[0]
                            own = tpl.get("years", {}).get(ty)
                            if not oks:
                                if cal == "hebrew" and base == "he":
                                    ys[k] = "hebr"
                                else:
                                    note("rangeYearRule", (ctx, bag, level, k))
                                    ys[k] = "numeric"
                            else:
                                ys[k] = own if own in oks else oks[0]
                        rt["years"] = ys
                    if x.get("tzProbe") is not None:
                        rt["tz"] = zone_slot(names, name_ctx, tpl.get("tz", 0), None, problems, timed=x["tzProbe"])
                        if rt["tz"] < 0 and len(examples["rangeslotx"]) < 6:
                            examples["rangeslotx"].append((ctx, bag, level, x["tzProbe"][:4]))
                    elif any(k.startswith("timeZoneName|") for k in x["lists"]):
                        rt["tz"] = tpl.get("tz", 0)
                    e = {"r": intern("rangeTemplates", rt)}
                    if x["collapsed"]:
                        e["mixed"] = True
                        note("mixedRange", (ctx, bag, level, x["collapsed"]))
                    rb[level] = e
                rc[key] = rb
            contexts[ctx] = c
            rcontexts[ctx] = rc
        out = {"lists": pools["lists"][0], "templates": pools["templates"][0], "contexts": contexts}
        rout = {"lists": pools["rangeLists"][0], "templates": pools["rangeTemplates"][0], "contexts": rcontexts}
        n = dump(f"loc_{lang}", out) + dump(f"rng_{lang}", rout)
        total += n
        print(f"{lang}: contexts {len(contexts)} templates {len(out['templates'])} lists {len(out['lists'])} rangeTemplates {len(rout['templates'])} rangeLists {len(rout['lists'])}; {n} bytes json")
    print("dtf problems:", dict(problems))
    for k, v in examples.items():
        print(" ", k, json.dumps(v, ensure_ascii=False)[:1200])
    return T


# ---------------------------------------------------------------- locale defaults, duration
def build_locale():
    L = load(os.path.join(EX, "locale_meta.json"))
    print("locale:", dump("locale", L), "bytes")


def strip_list(lists, loc, problems):
    out = {}
    for w, (two, three, four) in lists.items():
        p2 = two.replace("A", "{0}", 1).replace("B", "{1}", 1)
        a, rest = four.split("A", 1)
        s1, rest = rest.split("B", 1)
        m1, rest = rest.split("C", 1)
        e1, tail = rest.split("D", 1)
        _, r3 = three.split("A", 1)
        s3, r3 = r3.split("B", 1)
        e3, t3 = r3.split("C", 1)
        if s3 != s1 or e3 != e1:
            problems.append((loc, w, three, four))
        out[w] = {"two": p2, "start": a + "{0}" + s1 + "{1}", "middle": "{0}" + m1 + "{1}", "end": "{0}" + e1 + "{1}" + tail}
    return out


def build_duration():
    D = load(os.path.join(EX, "duration.json"))
    out, problems = {}, []
    for tag, r in D.items():
        base = tag.split("-u-")[0]
        sample = r["digitalSample"]
        dg = r["digits"]
        # h:mm:ss: the separator is what is left between the digits.
        sep = sample[len(dg[1]):-len(dg[0]) * 2 - len(dg[3]) - len(dg[2])]
        sep = sample.split(dg[1], 1)[1].split(dg[0], 1)[0]
        out[f"{base}|{r['ns']}"] = {"ns": r["ns"], "digits": dg, "minus": r["negPattern"].replace(dg[1], "", 1), "group": r["group"], "decimal": r["decimal"],
                                    "minGrouping": r["minGrouping"], "secondary": r["secondary"], "plural": r["pluralLocale"], "sep": sep,
                                    "units": r["units"], "lists": strip_list(r["lists"], tag, problems)}
        if r.get("conflicts"):
            print("WARN unit conflicts", tag, r["conflicts"][:3])
    if problems:
        print("WARN list problems", problems[:5])
    print("duration:", len(out), dump("duration", out), "bytes")


if __name__ == "__main__":
    build_tz()
    build_cal()
    names = build_names()
    build_dtf(names)
    build_locale()
    build_duration()
    sizes = {f: os.path.getsize(os.path.join(OUT, f)) for f in os.listdir(OUT) if f.endswith(".gz")}
    print("total gz bytes", sum(sizes.values()))
