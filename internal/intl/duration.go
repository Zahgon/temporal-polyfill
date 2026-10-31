package intl

import (
	"math"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

type listPatterns struct {
	Two    string `json:"two"`
	Start  string `json:"start"`
	Middle string `json:"middle"`
	End    string `json:"end"`
}

// durationLocale is what Intl.DurationFormat composes for one (base tag, numbering system).
type durationLocale struct {
	NS          string                       `json:"ns"`
	Digits      []string                     `json:"digits"`
	Minus       string                       `json:"minus"`
	Group       string                       `json:"group"`
	Decimal     string                       `json:"decimal"`
	MinGrouping int                          `json:"minGrouping"`
	Secondary   int                          `json:"secondary"`
	Plural      string                       `json:"plural"`
	Sep         string                       `json:"sep"`
	Units       map[string]map[string]string `json:"units"`
	Lists       map[string]listPatterns      `json:"lists"`
}

var (
	durOnce sync.Once
	durData map[string]*durationLocale
)

func durationLocales() map[string]*durationLocale {
	durOnce.Do(func() { loadJSON("duration", &durData) })
	return durData
}

var (
	durationUnits = [10]string{"years", "months", "weeks", "days", "hours", "minutes", "seconds", "milliseconds", "microseconds", "nanoseconds"}
	durationICU   = [10]string{"year", "month", "week", "day", "hour", "minute", "second", "millisecond", "microsecond", "nanosecond"}
)

const (
	dYears = iota
	dMonths
	dWeeks
	dDays
	dHours
	dMinutes
	dSeconds
	dMillis
	dMicros
	dNanos
)

type durationUnit struct{ style, display string }

// DurationFormat mirrors Intl.DurationFormat for the recorded locales.
type DurationFormat struct {
	loc        *durationLocale
	plural     func(pluralOperands) string
	style      string
	units      [10]durationUnit
	fractional int // -1 when undefined
	base       string
	language   string
}

func isFractionalUnit(i int) bool { return i >= dMillis }

func invalidObject() error { return rangeError("Invalid object : #<Object>") }

// NewDurationFormat mirrors new Intl.DurationFormat(locales, options).
func NewDurationFormat(locales_ any, options map[string]any) (*DurationFormat, error) {
	tag, gap := lookupDurationLocale(locales_)
	if _, isGap := gap.(*Unsupported); gap != nil && !isGap {
		return nil, gap
	}
	get := func(k string) (string, bool) {
		v, ok := options[k]
		if !ok || v == nil {
			return "", false
		}
		return toOptionString(v), true
	}
	if s, ok := get("localeMatcher"); ok && s != "lookup" && s != "best fit" {
		return nil, outOfRange("DurationFormat", s, "localeMatcher")
	}
	nsOpt := ""
	if s, ok := get("numberingSystem"); ok {
		if !wellFormedType(s) {
			return nil, rangeError("Invalid numberingSystem : " + s)
		}
		nsOpt = asciiLower(s)
	}
	f := &DurationFormat{style: "short", fractional: -1, base: tag.base, language: tag.language}
	if s, ok := get("style"); ok {
		if !contains([]string{"long", "short", "narrow", "digital"}, s) {
			return nil, outOfRange("DurationFormat", s, "style")
		}
		f.style = s
	}
	prev := ""
	for i, unit := range durationUnits {
		var allowed []string
		digitalBase := "short"
		switch {
		case i <= dDays:
			allowed = []string{"long", "short", "narrow"}
		case i <= dSeconds:
			allowed = []string{"long", "short", "narrow", "numeric", "2-digit"}
			digitalBase = "numeric"
		default:
			allowed = []string{"long", "short", "narrow", "numeric"}
			digitalBase = "numeric"
		}
		style, hasStyle := get(unit)
		if hasStyle && !contains(allowed, style) {
			return nil, outOfRange("DurationFormat", style, unit)
		}
		displayDefault := "always"
		if !hasStyle {
			if f.style == "digital" {
				if i < dHours || i > dSeconds {
					displayDefault = "auto"
				}
				style = digitalBase
			} else if prev == "fractional" || prev == "numeric" || prev == "2-digit" {
				if i != dMinutes && i != dSeconds {
					displayDefault = "auto"
				}
				style = "numeric"
			} else {
				displayDefault = "auto"
				style = f.style
			}
		}
		if style == "numeric" && isFractionalUnit(i) {
			style = "fractional"
			displayDefault = "auto"
		}
		display, hasDisplay := get(unit + "Display")
		if hasDisplay && display != "auto" && display != "always" {
			return nil, outOfRange("DurationFormat", display, unit+"Display")
		}
		if !hasDisplay {
			display = displayDefault
		}
		if display == "always" && style == "fractional" {
			return nil, invalidObject()
		}
		if prev == "fractional" && style != "fractional" {
			return nil, invalidObject()
		}
		if prev == "numeric" || prev == "2-digit" {
			if style != "fractional" && style != "numeric" && style != "2-digit" {
				return nil, invalidObject()
			}
			if i == dMinutes || i == dSeconds {
				style = "2-digit"
			}
		}
		f.units[i] = durationUnit{style: style, display: display}
		prev = style
	}
	if v, ok := options["fractionalDigits"]; ok && v != nil {
		n := toNumber(v)
		if math.IsNaN(n) || n < 0 || n > 9 {
			return nil, rangeError("fractionalDigits value is out of range.")
		}
		f.fractional = int(math.Floor(n))
	}
	if gap != nil {
		return nil, gap
	}
	def, ok := locales().Defaults[tag.base]
	if !ok {
		return nil, &Unsupported{What: "duration locale " + tag.base}
	}
	ns := def.NS
	l := locales()
	if n, ok := tag.keywords["nu"]; ok && l.numbering[n] && (nsOpt == "" || nsOpt == n) {
		ns = n
	}
	if nsOpt != "" && l.numbering[nsOpt] {
		ns = nsOpt
	}
	loc, ok := durationLocales()[tag.base+"|"+ns]
	if !ok {
		return nil, &Unsupported{What: "duration locale " + tag.base + "|" + ns}
	}
	f.loc = loc
	pr, err := pluralRulesFor(loc.Plural)
	if err != nil {
		return nil, err
	}
	f.plural = pr
	return f, nil
}

func lookupDurationLocale(v any) (langTag, error) { return lookupLocale(v) }

// durationValue is a magnitude as exact decimal digits.
type durationValue struct{ ip, fp string }

func (v durationValue) isZero() bool {
	return strings.Trim(v.ip, "0") == "" && strings.Trim(v.fp, "0") == ""
}

// floatValue is |v|'s shortest decimal digits, as ICU's DecimalQuantity reads a double.
func floatValue(v float64) durationValue {
	s := strconv.FormatFloat(math.Abs(v), 'e', -1, 64)
	mant, expStr, _ := strings.Cut(s, "e")
	exp, _ := strconv.Atoi(expStr)
	digits := strings.Replace(mant, ".", "", 1)
	point := exp + 1
	if point <= 0 {
		digits = strings.Repeat("0", -point+1) + digits
		point = 1
	}
	if len(digits) < point {
		digits += strings.Repeat("0", point-len(digits))
	}
	return durationValue{ip: digits[:point], fp: digits[point:]}
}

// fractionalValue is V8's sum for a unit with its smaller units as a fraction, in doubles and from the
// smallest term up: ns/1e9 + us/1e6 + ms/1e3 + s. Recorded vectors discriminate this order from the others.
func fractionalValue(unit float64, parts ...float64) durationValue {
	sum := 0.0
	div := 1.0
	for range parts {
		div *= 1e3
	}
	for i := len(parts) - 1; i >= 0; i-- {
		sum += parts[i] / div
		div /= 1e3
	}
	return floatValue(sum + unit)
}

func (f *DurationFormat) digitsOf(v durationValue, minInt, minFrac, maxFrac int, grouping bool) (string, pluralOperands) {
	ip, fp := v.ip, v.fp
	if maxFrac >= 0 && len(fp) > maxFrac {
		fp = fp[:maxFrac]
	}
	fp = strings.TrimRight(fp, "0")
	for len(fp) < minFrac {
		fp += "0"
	}
	if t := strings.TrimLeft(ip, "0"); t != "" {
		ip = t
	} else {
		ip = "0"
	}
	op := operandsOf(ip, fp)
	if pad := minInt - len(ip); pad > 0 {
		ip = strings.Repeat("0", pad) + ip
	}
	if !grouping {
		return f.localize(ip) + f.fractionText(fp), op
	}
	return f.groupDigits(ip) + f.fractionText(fp), op
}

func (f *DurationFormat) localize(s string) string {
	if f.loc.Digits[0] == "0" {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		b.WriteString(f.loc.Digits[s[i]-'0'])
	}
	return b.String()
}

func (f *DurationFormat) fractionText(fp string) string {
	if fp == "" {
		return ""
	}
	return f.loc.Decimal + f.localize(fp)
}

func (f *DurationFormat) groupDigits(ip string) string {
	l := f.loc
	if len(ip) <= 3 || len(ip) < 3+l.MinGrouping || l.Group == "" {
		return f.localize(ip)
	}
	head, tail := ip[:len(ip)-3], ip[len(ip)-3:]
	var groups []string
	for len(head) > l.Secondary {
		groups = append([]string{f.localize(head[len(head)-l.Secondary:])}, groups...)
		head = head[:len(head)-l.Secondary]
	}
	groups = append([]string{f.localize(head)}, groups...)
	return strings.Join(append(groups, f.localize(tail)), l.Group)
}

// outputter accumulates the list elements of one format call, as V8's
// DurationRecordToListOfFormattedNumber does.
type outputter struct {
	f           *DurationFormat
	strings     []string
	signPending bool
	negative    bool
}

func (o *outputter) number(v durationValue, minInt, minFrac, maxFrac int, grouping bool) (string, pluralOperands) {
	text, op := o.f.digitsOf(v, minInt, minFrac, maxFrac, grouping)
	if o.signPending {
		o.signPending = false
		if o.negative {
			text = o.f.loc.Minus + text
		}
	}
	return text, op
}

func (o *outputter) emit(s string, addToLast bool) {
	if addToLast && len(o.strings) > 0 {
		o.strings[len(o.strings)-1] += o.f.loc.Sep + s
		return
	}
	o.strings = append(o.strings, s)
}

func (o *outputter) output(unit int, value durationValue, addToLast bool, minFrac, maxFrac int, force bool) error {
	u := o.f.units[unit]
	if !force && u.display == "auto" && value.isZero() {
		return nil
	}
	switch u.style {
	case "numeric", "fractional":
		text, _ := o.number(value, 1, minFrac, maxFrac, false)
		o.emit(text, addToLast)
	case "2-digit":
		text, _ := o.number(value, 2, minFrac, maxFrac, false)
		o.emit(text, addToLast)
	default:
		text, op := o.number(value, 1, minFrac, maxFrac, true)
		pats, ok := o.f.loc.Units[u.style+"|"+durationICU[unit]]
		if !ok {
			return &Unsupported{What: "duration unit " + durationICU[unit]}
		}
		cat := o.f.plural(op)
		pat, ok := pats[cat]
		if !ok {
			pat = pats["other"]
		}
		o.emit(strings.Replace(pat, "{0}", text, 1), false)
	}
	return nil
}

func durationSign(r *[10]float64) int {
	for _, v := range r {
		if v < 0 {
			return -1
		}
		if v > 0 {
			return 1
		}
	}
	return 0
}

func checkDuration(r *[10]float64) error {
	sign := 0
	for _, v := range r {
		if math.IsNaN(v) || math.IsInf(v, 0) || v != math.Trunc(v) {
			return rangeError("Invalid time value for Temporal ../deps/v8/src/objects/js-temporal-objects.cc:2593")
		}
		if v != 0 {
			s := 1
			if v < 0 {
				s = -1
			}
			if sign != 0 && s != sign {
				return invalidObject()
			}
			sign = s
		}
	}
	for _, v := range r[:3] {
		if math.Abs(v) >= 1<<32 {
			return invalidObject()
		}
	}
	// |days*86400 + hours*3600 + ... + ns/1e9| < 2^53, as an exact sum in nanoseconds.
	total := new(big.Int)
	for i, scale := range [...]int64{dDays: 86400e9, dHours: 3600e9, dMinutes: 60e9, dSeconds: 1e9, dMillis: 1e6, dMicros: 1e3, dNanos: 1} {
		if i < dDays {
			continue
		}
		n, _ := new(big.Float).SetFloat64(math.Abs(r[i])).Int(nil)
		total.Add(total, n.Mul(n, big.NewInt(scale)))
	}
	limit := new(big.Int).Mul(big.NewInt(1<<53), big.NewInt(1e9))
	if total.Cmp(limit) >= 0 {
		return invalidObject()
	}
	return nil
}

// Format mirrors Intl.DurationFormat.prototype.format.
func (f *DurationFormat) Format(d DurationRecord) (string, error) {
	r := [10]float64{d.Years, d.Months, d.Weeks, d.Days, d.Hours, d.Minutes, d.Seconds, d.Milliseconds, d.Microseconds, d.Nanoseconds}
	if err := checkDuration(&r); err != nil {
		return "", err
	}
	o := &outputter{f: f, signPending: true, negative: durationSign(&r) < 0}
	for _, u := range []int{dYears, dMonths, dWeeks, dDays} {
		if err := o.output(u, floatValue(r[u]), false, 0, -1, false); err != nil {
			return "", err
		}
	}
	hoursValue := floatValue(r[dHours])
	if err := o.output(dHours, hoursValue, false, 0, -1, false); err != nil {
		return "", err
	}
	minFrac, maxFrac := 0, 9
	if f.fractional >= 0 {
		minFrac, maxFrac = f.fractional, f.fractional
	}
	secondsValue, secondsFrac := floatValue(r[dSeconds]), false
	if f.units[dMillis].style == "fractional" {
		secondsValue, secondsFrac = fractionalValue(r[dSeconds], r[dMillis], r[dMicros], r[dNanos]), true
	}
	numeric := func(u int) bool {
		s := f.units[u].style
		return s == "numeric" || s == "2-digit"
	}
	// Numeric minutes appear between displayed numeric hours and seconds whatever their own display says.
	shown := func(u int, v durationValue) bool { return f.units[u].display == "always" || !v.isZero() }
	forceMinutes := numeric(dHours) && numeric(dMinutes) && numeric(dSeconds) && shown(dHours, hoursValue) && shown(dSeconds, secondsValue)
	if err := o.output(dMinutes, floatValue(r[dMinutes]), numeric(dHours), 0, -1, forceMinutes); err != nil {
		return "", err
	}
	if secondsFrac {
		if err := o.output(dSeconds, secondsValue, true, minFrac, maxFrac, false); err != nil {
			return "", err
		}
	} else {
		if err := o.output(dSeconds, secondsValue, true, 0, -1, false); err != nil {
			return "", err
		}
		if f.units[dMicros].style == "fractional" {
			if err := o.output(dMillis, fractionalValue(r[dMillis], r[dMicros], r[dNanos]), false, minFrac, maxFrac, false); err != nil {
				return "", err
			}
		} else {
			if err := o.output(dMillis, floatValue(r[dMillis]), false, 0, -1, false); err != nil {
				return "", err
			}
			if f.units[dNanos].style == "fractional" {
				if err := o.output(dMicros, fractionalValue(r[dMicros], r[dNanos]), false, minFrac, maxFrac, false); err != nil {
					return "", err
				}
			} else {
				if err := o.output(dMicros, floatValue(r[dMicros]), false, 0, -1, false); err != nil {
					return "", err
				}
				if err := o.output(dNanos, floatValue(r[dNanos]), false, 0, -1, false); err != nil {
					return "", err
				}
			}
		}
	}
	width := f.style
	if width == "digital" {
		width = "short"
	}
	return f.formatList(width, o.strings), nil
}

func (f *DurationFormat) formatList(width string, items []string) string {
	p := f.loc.Lists[width]
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return applyList(contextualPattern(f.language, p.Two, items[1]), items[0], items[1])
	}
	out := applyList(p.Start, items[0], items[1])
	for _, it := range items[2 : len(items)-1] {
		out = applyList(p.Middle, out, it)
	}
	last := items[len(items)-1]
	return applyList(contextualPattern(f.language, p.End, last), out, last)
}

func applyList(p, a, b string) string {
	return strings.Replace(strings.Replace(p, "{0}", a, 1), "{1}", b, 1)
}

// contextualPattern applies ICU's context-sensitive list joiners: Spanish "y" before an i sound and
// Hebrew vav before a non-Hebrew word.
func contextualPattern(language, p, next string) string {
	switch language {
	case "es":
		if p == "{0} y {1}" && spanishESound(next) {
			return "{0} e {1}"
		}
	case "he":
		if p == "{0} ו{1}" {
			if r, _ := utf8.DecodeRuneInString(next); next != "" && !unicode.Is(unicode.Hebrew, r) {
				return "{0} ו-{1}"
			}
		}
	}
	return p
}

func spanishESound(s string) bool {
	if s == "" {
		return false
	}
	if (s[0] == 'h' || s[0] == 'H') && len(s) > 1 && (s[1] == 'i' || s[1] == 'I') &&
		(len(s) == 2 || !(s[2] == 'a' || s[2] == 'A' || s[2] == 'e' || s[2] == 'E')) {
		return true
	}
	return s[0] == 'i' || s[0] == 'I'
}
