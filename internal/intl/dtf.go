package intl

import (
	"encoding/json"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// dtfTemplate is one recorded (context, bag) formatter: its resolved options, part shape, and per part
// type the index of its value list.
type dtfTemplate struct {
	RO    map[string]any    `json:"ro"`
	Parts [][]string        `json:"parts"`
	Minus string            `json:"minus"`
	Lists map[string]int    `json:"lists"`
	Years map[string]string `json:"years"`
	Frac  int               `json:"frac"`
	TZ    int               `json:"tz"`
}

// rangeTemplate is one formatRange level: its part shape and, per "type|source|occurrence", the values
// ICU renders there (interval patterns have their own field widths).
type rangeTemplate struct {
	Shape [][]string        `json:"shape"`
	Lists map[string]int    `json:"lists"`
	Years map[string]string `json:"years"`
	Minus string            `json:"minus"`
	TZ    int               `json:"tz"`
}

type dtfContext struct {
	Bags map[string]json.RawMessage `json:"bags"`
}

// dtfShard is the data file of one language.
type dtfShard struct {
	Lists     []map[string]string    `json:"lists"`
	Templates []*dtfTemplate         `json:"templates"`
	Contexts  map[string]*dtfContext `json:"contexts"`
}

// rangeShard is the formatRange data file of one language, read on the first formatRange.
type rangeShard struct {
	Lists     []map[string]string                              `json:"lists"`
	Templates []*rangeTemplate                                 `json:"templates"`
	Contexts  map[string]map[string]map[string]json.RawMessage `json:"contexts"`
	merged    sync.Map
}

var (
	shardMu     sync.Mutex
	shards      = map[string]*dtfShard{}
	rangeShards = map[string]*rangeShard{}
)

func rangeShardFor(language string) *rangeShard {
	shardMu.Lock()
	defer shardMu.Unlock()
	if s, ok := rangeShards[language]; ok {
		return s
	}
	var s *rangeShard
	if dataExists("rng_" + language + ".json.gz") {
		s = &rangeShard{}
		loadJSON("rng_"+language, s)
	}
	rangeShards[language] = s
	return s
}

func shardFor(language string) *dtfShard {
	shardMu.Lock()
	defer shardMu.Unlock()
	if s, ok := shards[language]; ok {
		return s
	}
	var s *dtfShard
	if dataExists("loc_" + language + ".json.gz") {
		s = &dtfShard{}
		loadJSON("loc_"+language, s)
	}
	shards[language] = s
	return s
}

// DateTimeFormat mirrors Intl.DateTimeFormat for the recorded grid.
type DateTimeFormat struct {
	locale   string // resolved tag, with the keywords that took effect
	language string
	shard    *dtfShard
	ctxKey   string
	bagKey   string
	tpl      *dtfTemplate
	zone     string
	canon    string
	nameCtx  string
	calendar string
	ns       string
	digits   []string
}

var dtfFieldValues = map[string][]string{
	"weekday":      {"narrow", "short", "long"},
	"era":          {"narrow", "short", "long"},
	"year":         {"2-digit", "numeric"},
	"month":        {"2-digit", "numeric", "narrow", "short", "long"},
	"day":          {"2-digit", "numeric"},
	"dayPeriod":    {"narrow", "short", "long"},
	"hour":         {"2-digit", "numeric"},
	"minute":       {"2-digit", "numeric"},
	"second":       {"2-digit", "numeric"},
	"timeZoneName": {"short", "long", "shortOffset", "longOffset", "shortGeneric", "longGeneric"},
	"hourCycle":    {"h11", "h12", "h23", "h24"},
	"dateStyle":    {"full", "long", "medium", "short"},
	"timeStyle":    {"full", "long", "medium", "short"},
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// toOptionString is GetOption's ToString for the raw JS values a bag holds.
func toOptionString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case bool:
		if x {
			return "true"
		}
		return "false"
	case float64:
		switch {
		case math.IsNaN(x):
			return "NaN"
		case math.IsInf(x, 1):
			return "Infinity"
		case math.IsInf(x, -1):
			return "-Infinity"
		case x == math.Trunc(x) && math.Abs(x) < 1e21:
			return strconv.FormatFloat(x, 'f', -1, 64)
		}
		return strconv.FormatFloat(x, 'g', -1, 64)
	case int:
		return strconv.Itoa(x)
	case nil:
		return "null"
	}
	return "undefined"
}

func toBoolean(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		return x != ""
	case float64:
		return x != 0 && !math.IsNaN(x)
	case int:
		return x != 0
	}
	return v != nil
}

func toNumber(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case int:
		return float64(x)
	case bool:
		if x {
			return 1
		}
		return 0
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return 0
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f
		}
	}
	return math.NaN()
}

func outOfRange(ctor, value, prop string) error {
	return rangeError("Value " + value + " out of range for Intl." + ctor + " options property " + prop)
}

// wellFormedType is the unicode_locale_extension type production used for calendar and numberingSystem.
func wellFormedType(s string) bool {
	for _, p := range strings.Split(s, "-") {
		if len(p) < 3 || len(p) > 8 || !isAlnum(p) {
			return false
		}
	}
	return true
}

var dtfOrder = []string{"weekday", "era", "year", "month", "day", "dayPeriod", "hour", "minute", "second", "fractionalSecondDigits", "timeZoneName"}

// dtfOptions is what the constructor reads, validated in V8's order.
type dtfOptions struct {
	calendar, ns, hourCycle string
	hasHour12, hour12       bool
	zone                    string
	key                     string
	hasHC                   bool
}

func readDTFOptions(opts map[string]any) (*dtfOptions, error) {
	o := &dtfOptions{}
	get := func(k string) (any, bool) {
		v, ok := opts[k]
		if !ok || v == nil {
			return nil, false
		}
		return v, true
	}
	if v, ok := get("localeMatcher"); ok {
		if s := toOptionString(v); s != "lookup" && s != "best fit" {
			return nil, outOfRange("DateTimeFormat", s, "localeMatcher")
		}
	}
	if v, ok := get("calendar"); ok {
		s := toOptionString(v)
		if !wellFormedType(s) {
			return nil, rangeError("Invalid calendar : " + s)
		}
		o.calendar = canonicalCalendar(asciiLower(s))
	}
	if v, ok := get("numberingSystem"); ok {
		s := toOptionString(v)
		if !wellFormedType(s) {
			return nil, rangeError("Invalid numberingSystem : " + s)
		}
		o.ns = s
	}
	if v, ok := get("hour12"); ok {
		o.hasHour12, o.hour12 = true, toBoolean(v)
	}
	if v, ok := get("hourCycle"); ok {
		s := toOptionString(v)
		if !contains(dtfFieldValues["hourCycle"], s) {
			return nil, outOfRange("DateTimeFormat", s, "hourCycle")
		}
		o.hourCycle = s
	}
	o.hasHC = o.hasHour12 || o.hourCycle != ""
	if v, ok := get("timeZone"); ok {
		o.zone = toOptionString(v)
		if _, ok := CanonicalTimeZone(o.zone); !ok {
			return nil, rangeError("Invalid time zone specified: " + o.zone)
		}
	}
	type kv struct {
		k string
		v any
	}
	var pairs []kv
	explicit, needDefaults := false, true
	for _, k := range dtfOrder {
		v, ok := get(k)
		if !ok {
			continue
		}
		explicit = true
		if k == "fractionalSecondDigits" {
			n := toNumber(v)
			if math.IsNaN(n) || n < 1 || n > 3 {
				return nil, rangeError("fractionalSecondDigits value is out of range.")
			}
			pairs = append(pairs, kv{k, math.Floor(n)})
			needDefaults = false
			continue
		}
		s := toOptionString(v)
		if !contains(dtfFieldValues[k], s) {
			return nil, outOfRange("DateTimeFormat", s, k)
		}
		pairs = append(pairs, kv{k, s})
		if k != "era" && k != "timeZoneName" {
			needDefaults = false
		}
	}
	if v, ok := get("formatMatcher"); ok {
		if s := toOptionString(v); s != "basic" && s != "best fit" {
			return nil, outOfRange("DateTimeFormat", s, "formatMatcher")
		}
	}
	styled := false
	for _, k := range []string{"dateStyle", "timeStyle"} {
		if v, ok := get(k); ok {
			s := toOptionString(v)
			if !contains(dtfFieldValues[k], s) {
				return nil, outOfRange("DateTimeFormat", s, k)
			}
			pairs = append(pairs, kv{k, s})
			styled = true
		}
	}
	if styled && explicit {
		return nil, typeError("Invalid option : option")
	}
	if !styled && needDefaults {
		pairs = append(pairs, kv{"year", "numeric"}, kv{"month", "numeric"}, kv{"day", "numeric"})
	}
	hasHour := false
	for _, p := range pairs {
		if p.k == "hour" || p.k == "timeStyle" {
			hasHour = true
		}
	}
	if hasHour {
		if o.hasHour12 {
			pairs = append(pairs, kv{"hour12", o.hour12})
		}
		if o.hourCycle != "" && !o.hasHour12 {
			pairs = append(pairs, kv{"hourCycle", o.hourCycle})
		}
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].k < pairs[j].k })
	var b strings.Builder
	b.WriteByte('[')
	for i, p := range pairs {
		if i > 0 {
			b.WriteByte(',')
		}
		kb, _ := json.Marshal(p.k)
		vb, _ := json.Marshal(p.v)
		b.WriteByte('[')
		b.Write(kb)
		b.WriteByte(',')
		b.Write(vb)
		b.WriteByte(']')
	}
	b.WriteByte(']')
	o.key = b.String()
	return o, nil
}

// NewDateTimeFormat mirrors new Intl.DateTimeFormat(locales, options). Errors are *Error for what V8
// throws, *Unsupported for what the tables do not cover.
func NewDateTimeFormat(locales_ any, options map[string]any) (*DateTimeFormat, error) {
	tag, gap := lookupLocale(locales_)
	if _, isGap := gap.(*Unsupported); gap != nil && !isGap {
		return nil, gap
	}
	o, err := readDTFOptions(options)
	if err != nil {
		return nil, err
	}
	if gap != nil {
		return nil, gap
	}
	l := locales()
	def := l.Defaults[tag.base]
	kw := map[string]string{}
	calendar := def.Calendar
	if c, ok := tag.keywords["ca"]; ok {
		c = canonicalCalendar(c)
		if l.calendars[c] && (o.calendar == "" || o.calendar == c) {
			calendar = c
			kw["ca"] = c
		}
	}
	if o.calendar != "" && l.calendars[o.calendar] {
		calendar = o.calendar
	}
	ns := def.NS
	if n, ok := tag.keywords["nu"]; ok && l.numbering[n] && (o.ns == "" || o.ns == n) {
		ns = n
		kw["nu"] = n
	}
	if o.ns != "" && l.numbering[o.ns] {
		ns = o.ns
	}
	// A -u-hc- keyword shapes ICU's pattern even when an option overrides the cycle, so it stays part of
	// the context; it only stays in the resolved tag when the cycle in effect is still its own.
	hc := ""
	if h, ok := tag.keywords["hc"]; ok && contains(dtfFieldValues["hourCycle"], h) {
		hc = h
		kw["hc"] = h
	}
	zone := o.zone
	if zone == "" {
		zone = "UTC"
	}
	canon, _ := CanonicalTimeZone(zone)

	shard := shardFor(tag.language)
	if shard == nil {
		return nil, &Unsupported{What: "date formats for language " + tag.language}
	}
	ctxKey := tag.base + "|" + calendar + "|" + ns + "|" + hc
	ctx := shard.Contexts[ctxKey]
	if ctx == nil {
		return nil, &Unsupported{What: "date formats for " + ctxKey}
	}
	raw, ok := ctx.Bags[o.key]
	if !ok {
		return nil, &Unsupported{What: "date format " + o.key + " in " + ctxKey}
	}
	var idx int
	if err := json.Unmarshal(raw, &idx); err != nil {
		var t struct {
			Throw []string `json:"throw"`
		}
		if json.Unmarshal(raw, &t) == nil && len(t.Throw) == 2 {
			return nil, &Error{Name: t.Throw[0], Message: t.Throw[1]}
		}
		return nil, &Unsupported{What: "date format " + o.key}
	}
	tpl := shard.Templates[idx]
	if hc != "" && o.hasHC {
		if c, _ := tpl.RO["hourCycle"].(string); c != hc {
			delete(kw, "hc")
		}
	}
	return &DateTimeFormat{
		locale: resolvedLocaleTag(tag.base, kw), language: tag.language, shard: shard, ctxKey: ctxKey, bagKey: o.key, tpl: tpl,
		zone: zone, canon: canon, nameCtx: tag.base + "|" + ns, calendar: calendar, ns: ns, digits: digitsFor(ns),
	}, nil
}

func (f *DateTimeFormat) localizeDigits(s string) string {
	if f.digits == nil || f.digits[0] == "0" {
		return s
	}
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteString(f.digits[r-'0'])
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

type instantFields struct {
	ms                   float64
	day                  int64
	cal                  calFields
	hour, minute, second int64
	milli                int64
}

func (f *DateTimeFormat) fieldsAt(ms float64) (instantFields, error) {
	sec := int64(math.Floor(ms / 1000))
	local := ms + float64(OffsetSeconds(f.canon, sec)*1000)
	day := int64(math.Floor(local / 86400000))
	rem := int64(local - float64(float64(day)*86400000))
	x := instantFields{ms: ms, day: day, hour: rem / 3600000, minute: rem / 60000 % 60, second: rem / 1000 % 60, milli: rem % 1000}
	cf, ok := calendarFields(f.calendar, day)
	if !ok {
		return x, &Unsupported{What: f.calendar + " date outside the recorded range"}
	}
	if cf.Failed {
		return x, typeError("Internal error. Icu error.")
	}
	if islamicInstantGap(f.calendar, cf, day, ms) {
		return x, &Unsupported{What: islamicGapMessage}
	}
	x.cal = cf
	return x, nil
}

func (f *DateTimeFormat) list(typ string) map[string]string {
	i, ok := f.tpl.Lists[typ]
	if !ok {
		return nil
	}
	return f.shard.Lists[i]
}

// partData supplies the field values of one rendering: the single-date template, or one range level
// with its own lists per part source.
type partData struct {
	list  func(typ string) map[string]string
	year  func(typ string) string
	minus string
	tz    int
}

func (f *DateTimeFormat) single() partData {
	return partData{list: f.list, year: func(typ string) string { return f.tpl.Years[typ] }, minus: f.tpl.Minus, tz: f.tpl.TZ}
}

func (f *DateTimeFormat) renderYear(rule, minus string, y int64) (string, error) {
	if rule == "hebr" {
		return hebrewNumeral(y), nil
	}
	sign := ""
	if y < 0 {
		sign = minus
		y = -y
	}
	switch rule {
	case "numeric", "":
		return sign + f.localizeDigits(strconv.FormatInt(y, 10)), nil
	case "2-digit":
		s := strconv.FormatInt(y%100, 10)
		if len(s) < 2 {
			s = "0" + s
		}
		return sign + f.localizeDigits(s), nil
	case "pad2":
		s := strconv.FormatInt(y, 10)
		if len(s) < 2 {
			s = "0" + s
		}
		return sign + f.localizeDigits(s), nil
	case "jpan":
		if sign == "" && y == 1 {
			return "元", nil
		}
		return sign + f.localizeDigits(strconv.FormatInt(y, 10)), nil
	}
	return "", &Unsupported{What: "year rule " + rule}
}

func dayPeriodKey(x instantFields) string {
	z := func(v int64) string {
		if v == 0 {
			return "0"
		}
		return "x"
	}
	return strconv.FormatInt(x.hour, 10) + ":" + z(x.minute) + ":" + z(x.second) + ":" + z(x.milli)
}

func dayPeriod(l map[string]string, x instantFields) (string, bool) {
	if v, ok := l[dayPeriodKey(x)]; ok {
		return v, true
	}
	h := strconv.FormatInt(x.hour, 10) + ":"
	minuteZero := x.minute == 0
	var fallback string
	found := false
	for k, v := range l {
		if !strings.HasPrefix(k, h) {
			continue
		}
		if (k[len(h)] == '0') == minuteZero {
			return v, true
		}
		if x.hour != 12 && !found {
			fallback, found = v, true
		}
	}
	return fallback, found
}

func isLunarCalendar(c string) bool { return c == "chinese" || c == "dangi" }

func (f *DateTimeFormat) value(typ string, x instantFields, d partData) (string, bool, error) {
	lookup := func(key string) (string, bool, error) {
		v, ok := d.list(typ)[key]
		if !ok {
			return "", false, &Unsupported{What: typ + " " + key + " in " + f.locale}
		}
		return v, v != "", nil
	}
	switch typ {
	case "year", "relatedYear":
		y := x.cal.Year
		if typ == "year" && isLunarCalendar(f.calendar) {
			y = floorMod(y-1984, 60) + 1
		}
		s, err := f.renderYear(d.year(typ), d.minus, y)
		return s, s != "", err
	case "yearName":
		return lookup(strconv.FormatInt(floorMod(x.cal.Year-1984, 60), 10))
	case "month":
		return lookup(x.cal.Month)
	case "day":
		return lookup(strconv.FormatInt(x.cal.Day, 10))
	case "weekday":
		return lookup(strconv.FormatInt(isoWeekday(x.day), 10))
	case "era":
		key := x.cal.Era
		if f.calendar == "iso8601" {
			key = solarFields("gregory", x.day).Era
		}
		v, ok := d.list("era")[key]
		if !ok {
			// ICU's iso8601 calendar names an era only for narrow BCE dates: every other era is empty.
			if key == "" || f.calendar == "iso8601" {
				return "", false, nil
			}
			return "", false, &Unsupported{What: "era " + key + " in " + f.locale}
		}
		return v, v != "", nil
	case "hour":
		return lookup(strconv.FormatInt(x.hour, 10))
	case "minute":
		return lookup(strconv.FormatInt(x.minute, 10))
	case "second":
		return lookup(strconv.FormatInt(x.second, 10))
	case "dayPeriod":
		v, ok := dayPeriod(d.list("dayPeriod"), x)
		if !ok {
			return "", false, &Unsupported{What: "dayPeriod " + dayPeriodKey(x) + " in " + f.locale}
		}
		return v, v != "", nil
	case "fractionalSecond":
		s := strconv.FormatInt(x.milli+1000, 10)[1:]
		n := f.tpl.Frac
		if n < 1 || n > 3 {
			n = 3
		}
		return f.localizeDigits(s[:n]), true, nil
	case "timeZoneName":
		if d.tz < 0 {
			return "", false, &Unsupported{What: "zone name style in " + f.locale}
		}
		name, ok, err := zoneNameSlot(f.nameCtx, f.zone, f.canon, x.ms, d.tz)
		return name, ok && name != "", err
	}
	return "", false, &Unsupported{What: "part " + typ}
}

// render fills a part shape and returns the parts with each one's source (empty for single dates);
// omitted parts let the literals around them merge, as ICU's field positions leave one literal run
// between fields.
func (f *DateTimeFormat) render(shape [][]string, pick func(i int, src string) (instantFields, partData)) ([]RangePart, error) {
	out := make([]RangePart, 0, len(shape))
	for i, p := range shape {
		src := ""
		if len(p) > 2 || (len(p) == 2 && p[0] != "literal") {
			src = p[1]
		}
		if p[0] == "literal" {
			if n := len(out); n > 0 && out[n-1].Type == "literal" {
				out[n-1].Value += p[len(p)-1]
				continue
			}
			out = append(out, RangePart{Type: "literal", Value: p[len(p)-1], Source: src})
			continue
		}
		x, d := pick(i, src)
		v, ok, err := f.value(p[0], x, d)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, RangePart{Type: p[0], Value: v, Source: src})
		}
	}
	return settleSources(out), nil
}

// settleSources applies what ICU's interval spans do when an omitted field ended a side: the span stops at
// the side's last field, so a literal behind it is shared and merges with its shared neighbour.
func settleSources(parts []RangePart) []RangePart {
	lastField := map[string]int{"startRange": -1, "endRange": -1}
	for i, p := range parts {
		if p.Type != "literal" && p.Source != "" {
			lastField[p.Source] = i
		}
	}
	out := parts[:0]
	for i, p := range parts {
		if p.Type == "literal" && (p.Source == "startRange" || p.Source == "endRange") && i > lastField[p.Source] {
			p.Source = "shared"
		}
		if n := len(out); n > 0 && p.Type == "literal" && out[n-1].Type == "literal" {
			out[n-1].Value += p.Value
			if out[n-1].Source != p.Source {
				out[n-1].Source = "shared"
			}
			continue
		}
		out = append(out, p)
	}
	return out
}

func checkTime(ms float64) error {
	if !validTime(ms) {
		return rangeError("Invalid time value")
	}
	return nil
}

// FormatToParts mirrors Intl.DateTimeFormat.prototype.formatToParts.
func (f *DateTimeFormat) FormatToParts(ms float64) ([]Part, error) {
	if err := checkTime(ms); err != nil {
		return nil, err
	}
	ms = math.Trunc(ms)
	x, err := f.fieldsAt(ms)
	if err != nil {
		return nil, err
	}
	d := f.single()
	rp, err := f.render(f.tpl.Parts, func(int, string) (instantFields, partData) { return x, d })
	if err != nil {
		return nil, err
	}
	parts := make([]Part, len(rp))
	for i, p := range rp {
		parts[i] = Part{Type: p.Type, Value: p.Value}
	}
	return parts, nil
}

// Format mirrors Intl.DateTimeFormat.prototype.format, which (unlike formatToParts) replaces U+202F
// NARROW NO-BREAK SPACE with a plain space.
func (f *DateTimeFormat) Format(ms float64) (string, error) {
	parts, err := f.FormatToParts(ms)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, p := range parts {
		b.WriteString(p.Value)
	}
	return strings.ReplaceAll(b.String(), " ", " "), nil
}

// ResolvedOptions mirrors Intl.DateTimeFormat.prototype.resolvedOptions.
func (f *DateTimeFormat) ResolvedOptions() ResolvedOptions {
	r := ResolvedOptions{Locale: f.locale, Calendar: f.calendar, NumberingSystem: f.ns, TimeZone: f.canon}
	str := func(k string) *string {
		if s, ok := f.tpl.RO[k].(string); ok {
			return &s
		}
		return nil
	}
	r.HourCycle = str("hourCycle")
	if h, ok := f.tpl.RO["hour12"].(bool); ok {
		r.Hour12 = &h
	}
	r.Weekday, r.Era, r.Year, r.Month, r.Day = str("weekday"), str("era"), str("year"), str("month"), str("day")
	r.DayPeriod, r.Hour, r.Minute, r.Second = str("dayPeriod"), str("hour"), str("minute"), str("second")
	if n, ok := f.tpl.RO["fractionalSecondDigits"].(float64); ok {
		r.FractionalSecondDigits = &n
	}
	r.TimeZoneName, r.DateStyle, r.TimeStyle = str("timeZoneName"), str("dateStyle"), str("timeStyle")
	return r
}

var (
	calPartsMu    sync.Mutex
	calPartsCache = map[string]*DateTimeFormat{}
)

var calendarMonthNumber = map[string]map[string]int{}

func init() {
	for name, list := range map[string][]string{
		"gregory": monthsGregorian, "persian": monthsPersian, "islamic": monthsIslamic, "indian": monthsIndian,
		"coptic": monthsCoptic, "ethiopic": monthsEthiopic,
	} {
		m := make(map[string]int, len(list))
		for i, s := range list {
			m[s] = i + 1
		}
		calendarMonthNumber[name] = m
	}
}

// CalendarParts mirrors new Intl.DateTimeFormat('en-US-u-ca-'+id, {day:'numeric', month:'numeric',
// year:'numeric', era:'short', timeZone:zone}).formatToParts(ms), without touching the locale tables.
func CalendarParts(calendarID string, epochMs float64, zone string) ([]Part, error) {
	if err := checkTime(epochMs); err != nil {
		return nil, err
	}
	canon, ok := CanonicalTimeZone(zone)
	if !ok {
		return nil, rangeError("Invalid time zone specified: " + zone)
	}
	ms := math.Trunc(epochMs)
	day := int64(math.Floor((ms + float64(OffsetSeconds(canon, int64(math.Floor(ms/1000)))*1000)) / 86400000))
	id := canonicalCalendar(asciiLower(calendarID))
	var numbers map[string]int
	switch id {
	case "gregory", "buddhist", "japanese", "roc":
		numbers = calendarMonthNumber["gregory"]
	case "persian":
		numbers = calendarMonthNumber["persian"]
	case "indian":
		numbers = calendarMonthNumber["indian"]
	case "coptic":
		numbers = calendarMonthNumber["coptic"]
	case "ethiopic", "ethioaa":
		numbers = calendarMonthNumber["ethiopic"]
	case "islamic", "islamic-rgsa", "islamic-civil", "islamic-tbla", "islamic-umalqura":
		numbers = calendarMonthNumber["islamic"]
	case "hebrew", "chinese", "dangi":
	default:
		return nil, &Unsupported{What: "calendar " + calendarID}
	}
	cf, ok := calendarFields(id, day)
	if !ok {
		return nil, &Unsupported{What: id + " date outside the recorded range"}
	}
	if cf.Failed {
		return nil, typeError("Internal error. Icu error.")
	}
	if islamicInstantGap(id, cf, day, ms) {
		return nil, &Unsupported{What: islamicGapMessage}
	}
	year := strconv.FormatInt(cf.Year, 10)
	dom := strconv.FormatInt(cf.Day, 10)
	switch id {
	case "chinese", "dangi":
		return []Part{{"month", "Mo" + cf.Month}, {"literal", " "}, {"day", dom}, {"literal", ", "}, {"relatedYear", year}}, nil
	case "hebrew":
		month, _, _ := strings.Cut(cf.Month, "|")
		return []Part{{"day", dom}, {"literal", " "}, {"month", month}, {"literal", " "}, {"year", year}, {"literal", " "}, {"era", cf.Era}}, nil
	}
	parts := []Part{{"month", strconv.Itoa(numbers[cf.Month])}, {"literal", "/"}, {"day", dom}, {"literal", "/"}, {"year", year}}
	if cf.Era == "" {
		return append(parts, Part{"literal", " "}), nil
	}
	return append(parts, Part{"literal", " "}, Part{"era", cf.Era}), nil
}
