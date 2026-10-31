package temporal

import (
	"math"
	"math/big"
	"reflect"
	"strings"
	"time"

	"github.com/js-temporal/temporal-polyfill-go/internal/intl"
)

// intlRaise turns an error from the ICU tables into what the original does: an *intl.Error is the
// exception V8 would throw. Anything else is a gap in the recorded data, a hard crash and not a
// JavaScript exception.
func intlRaise(err error) {
	if err == nil {
		return
	}
	if e, ok := err.(*intl.Error); ok {
		panic(&Error{Name: e.Name, Message: e.Message})
	}
	panic(err)
}

// jsOptions is a null-prototype JavaScript object: properties in insertion order, where a property
// can be present and hold undefined (a nil value).
type jsOptions struct {
	keys []string
	vals map[string]any
}

func newJSOptions() *jsOptions { return &jsOptions{vals: map[string]any{}} }

func (o *jsOptions) has(key string) bool { _, ok := o.vals[key]; return ok }

func (o *jsOptions) get(key string) any { return o.vals[key] }

func (o *jsOptions) set(key string, value any) {
	if _, ok := o.vals[key]; !ok {
		o.keys = append(o.keys, key)
	}
	o.vals[key] = value
}

func (o *jsOptions) del(key string) {
	if _, ok := o.vals[key]; !ok {
		return
	}
	delete(o.vals, key)
	for i, k := range o.keys {
		if k == key {
			o.keys = append(o.keys[:i:i], o.keys[i+1:]...)
			break
		}
	}
}

// assign is Object.assign(o, {k1: v1, k2: v2, ...}) with string values.
func (o *jsOptions) assign(pairs ...string) {
	for i := 0; i < len(pairs); i += 2 {
		o.set(pairs[i], pairs[i+1])
	}
}

func (o *jsOptions) clone() *jsOptions {
	c := newJSOptions()
	for _, k := range o.keys {
		c.set(k, o.vals[k])
	}
	return c
}

// jsGet makes a jsOptions readable through objGet.
func (o *jsOptions) jsGet(key string) any { return o.vals[key] }

// icuOptions is the map the formatter reads: undefined properties are absent, and the coercions the
// Intl constructors apply to a property value (ToBoolean, ToNumber, ToString) are already done.
func icuOptions(o *jsOptions, booleans, numbers map[string]bool) map[string]any {
	m := make(map[string]any, len(o.keys))
	for _, k := range o.keys {
		v := unwrap(o.vals[k])
		if isUndefined(v) {
			continue
		}
		switch {
		case booleans[k]:
			m[k] = jsTruthy(v)
		case numbers[k]:
			m[k] = jsToNumber(v)
		default:
			m[k] = jsToString(v)
		}
	}
	return m
}

// jsTruthy is ECMAScript ToBoolean.
func jsTruthy(x any) bool {
	switch v := x.(type) {
	case nil, nullType:
		return false
	case bool:
		return v
	case float64:
		return v != 0 && !math.IsNaN(v)
	case string:
		return v != ""
	case *big.Int:
		return v.Sign() != 0
	}
	return true
}

var (
	dtfBooleanOptions = map[string]bool{"hour12": true}
	dtfNumberOptions  = map[string]bool{"fractionalSecondDigits": true}
	dfNumberOptions   = map[string]bool{"fractionalDigits": true}
)

// intlLocales prepares a locales argument for the formatter, which takes nil, a string or a
// []string. An object without a length is an empty list to CanonicalizeLocaleList.
func intlLocales(locales any) any {
	switch v := unwrap(locales).(type) {
	case nil:
		return nil
	case string:
		return v
	case []string:
		return v
	case nullType:
		throwType("Cannot convert undefined or null to object")
	}
	return nil
}

func newOriginalDateTimeFormat(locales any, options *jsOptions) *intl.DateTimeFormat {
	f, err := intl.NewDateTimeFormat(locales, icuOptions(options, dtfBooleanOptions, dtfNumberOptions))
	intlRaise(err)
	return f
}

// lazySlot is one of the DATE, YM, MD, TIME_FMT, DATETIME and INST slots: an amender function until
// the first use, a formatter after it, and null when both are nil.
type lazySlot struct {
	amend     func(*jsOptions) *jsOptions
	formatter *intl.DateTimeFormat
}

// DateTimeFormat is the polyfill's Intl.DateTimeFormat, which also formats Temporal objects.
type DateTimeFormat struct {
	locale      string
	options     *jsOptions
	original    *intl.DateTimeFormat
	tzCanonical string
	tzOriginal  string
	calID       string
	date        lazySlot
	ym          lazySlot
	md          lazySlot
	timeFmt     lazySlot
	datetime    lazySlot
	inst        lazySlot
}

// Part is one element of the result of DateTimeFormat.FormatToParts.
type Part struct {
	Type  string
	Value string
}

// RangePart is one element of the result of DateTimeFormat.FormatRangeToParts.
type RangePart struct {
	Type   string
	Value  string
	Source string
}

// ResolvedOptions is the result of DateTimeFormat.ResolvedOptions. A nil field is a property that
// resolvedOptions() does not report.
type ResolvedOptions struct {
	Locale, Calendar, NumberingSystem, TimeZone                     string
	HourCycle                                                       *string
	Hour12                                                          *bool
	Weekday, Era, Year, Month, Day, DayPeriod, Hour, Minute, Second *string
	FractionalSecondDigits                                          *float64
	TimeZoneName, DateStyle, TimeStyle                              *string
}

// Construction of built-in Intl.DateTimeFormat objects is sloooooow,
// so we'll only create those instances when we need them.
// See https://bugs.chromium.org/p/v8/issues/detail?id=6528
func getSlotLazy(obj *DateTimeFormat, slot *lazySlot) *intl.DateTimeFormat {
	if slot.amend != nil {
		// If we get here, `slot` holds an "amender function". It will take the user's
		// options and transform them into suitable options to be passed into the
		// built-in (non-polyfill) Intl.DateTimeFormat constructor. These options
		// will vary depending on the Temporal type, so that's why we store separate
		// formatters in separate props on the polyfill's DateTimeFormat instances.
		// The efficiency happens because we don't create an (expensive) formatter
		// until the user calls toLocaleString for that Temporal type.
		slot.formatter = newOriginalDateTimeFormat(obj.locale, slot.amend(obj.options))
		slot.amend = nil
	}
	return slot.formatter
}

// dtfOwnProperty is Object.prototype.hasOwnProperty plus the read. A property of a bag struct is
// present when its field is not a nil pointer. Temporal objects keep their properties on the
// prototype, and primitives have none of these names.
func dtfOwnProperty(obj any, key string) (any, bool) {
	if o, ok := obj.(*jsOptions); ok {
		return o.get(key), o.has(key)
	}
	if !isBag(obj) {
		return nil, false
	}
	rv := reflect.ValueOf(obj)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	f := rv.FieldByName(upperFirst(key))
	if !f.IsValid() || !f.CanInterface() {
		return nil, false
	}
	v, ok := f.Interface().(*Value)
	if !ok || v == nil {
		return nil, false
	}
	return v.v, true
}

// copyOwnOptions is ES.CopyDataProperties into target from an options object, skipping one key.
func copyOwnOptions(target *jsOptions, source any, excludedKey string) {
	if o, ok := source.(*jsOptions); ok {
		for _, k := range o.keys {
			if k != excludedKey {
				target.set(k, o.vals[k])
			}
		}
		return
	}
	if !isBag(source) {
		return
	}
	rv := reflect.ValueOf(source)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		if !rt.Field(i).IsExported() {
			continue
		}
		v, ok := rv.Field(i).Interface().(*Value)
		if !ok || v == nil {
			continue
		}
		key := strings.ToLower(rt.Field(i).Name[:1]) + rt.Field(i).Name[1:]
		if key != excludedKey {
			target.set(key, v.v)
		}
	}
}

// resolvedToOptions is the object original.resolvedOptions() returns, in its property order.
func resolvedToOptions(ro intl.ResolvedOptions) *jsOptions {
	o := newJSOptions()
	o.set("locale", ro.Locale)
	o.set("calendar", ro.Calendar)
	o.set("numberingSystem", ro.NumberingSystem)
	o.set("timeZone", ro.TimeZone)
	str := func(key string, p *string) {
		if p != nil {
			o.set(key, *p)
		}
	}
	str("hourCycle", ro.HourCycle)
	if ro.Hour12 != nil {
		o.set("hour12", *ro.Hour12)
	}
	str("weekday", ro.Weekday)
	str("era", ro.Era)
	str("year", ro.Year)
	str("month", ro.Month)
	str("day", ro.Day)
	str("dayPeriod", ro.DayPeriod)
	str("hour", ro.Hour)
	str("minute", ro.Minute)
	str("second", ro.Second)
	if ro.FractionalSecondDigits != nil {
		o.set("fractionalSecondDigits", *ro.FractionalSecondDigits)
	}
	str("timeZoneName", ro.TimeZoneName)
	str("dateStyle", ro.DateStyle)
	str("timeStyle", ro.TimeStyle)
	return o
}

func internalCreateDateTimeFormat(dtf *DateTimeFormat, localeParam any, optionsParam any, required string) {
	optionsParam = unwrap(optionsParam)
	hasOptions := !isUndefined(optionsParam)
	options := newJSOptions()
	if hasOptions {
		// Read all the options in the expected order and copy them to a
		// null-prototype object with which we can do further operations
		// unobservably
		props := []string{
			"localeMatcher",
			"calendar",
			"numberingSystem",
			"hour12",
			"hourCycle",
			"timeZone",
			"weekday",
			"era",
			"year",
			"month",
			"day",
			"dayPeriod",
			"hour",
			"minute",
			"second",
			"fractionalSecondDigits",
			"timeZoneName",
			"formatMatcher",
			"dateStyle",
			"timeStyle",
		}
		if isNull(optionsParam) {
			throwType("Expected object not null")
		}
		for _, prop := range props {
			if v, ok := dtfOwnProperty(optionsParam, prop); ok {
				options.set(prop, v)
			}
		}
	}
	locale := intlLocales(localeParam)
	original := newOriginalDateTimeFormat(locale, options)
	ro := original.ResolvedOptions()

	// If the original Intl.DateTimeFormat resolvedOptions isn't new enough to
	// reject the unshipped IDs, re-do it. Fallbacks are based on discussion
	// https://github.com/tc39/proposal-intl-era-monthcode/issues/29
	// The original warns on the console here; this port has no output.
	fallbacks := [][2]string{
		{"islamic", "islamic-tbla"},
		{"islamic-rgsa", "islamic-umalqura"},
	}
	for _, fallback := range fallbacks {
		if ro.Calendar == fallback[0] {
			options.set("calendar", fallback[1])
			original = newOriginalDateTimeFormat(locale, options)
			ro = original.ResolvedOptions()
		}
	}

	// DateTimeFormat instances are very expensive to create. Therefore, they will
	// be lazily created only when needed, using the locale and options provided.
	// But it's possible for callers to mutate those inputs before lazy creation
	// happens. For this reason, we clone the inputs instead of caching the
	// original objects. To avoid the complexity of deep cloning any inputs that
	// are themselves objects (e.g. the locales array, or options property values
	// that will be coerced to strings), we rely on `resolvedOptions()` to do the
	// coercion and cloning for us. Unfortunately, we can't just use the resolved
	// options as-is because our options-amending logic adds additional fields if
	// the user doesn't supply any unit fields like year, month, day, hour, etc.
	// Therefore, we limit the properties in the clone to properties that were
	// present in the original input.
	if hasOptions {
		clonedResolved := resolvedToOptions(ro)
		for _, prop := range append([]string(nil), clonedResolved.keys...) {
			if !options.has(prop) {
				clonedResolved.del(prop)
			}
		}
		// hour12/hourCycle don't show up in resolvedOptions() unless the chosen
		// format includes an hour component, so copy them explicitly in case they
		// would otherwise be lost
		clonedResolved.set("hour12", options.get("hour12"))
		clonedResolved.set("hourCycle", options.get("hourCycle"))
		dtf.options = clonedResolved
	} else {
		dtf.options = options
	}

	dtf.locale = ro.Locale
	dtf.original = original
	dtf.tzCanonical = ro.TimeZone
	dtf.calID = ro.Calendar

	if !isUndefined(options.get("dateStyle")) || !isUndefined(options.get("timeStyle")) {
		if !isUndefined(options.get("dateStyle")) {
			if required == "time" {
				throwType("toLocaleString of Temporal.PlainTime does not support dateStyle option")
			}
			dtf.date = lazySlot{amend: dateAmend}
			dtf.ym = lazySlot{amend: yearMonthAmend}
			dtf.md = lazySlot{amend: monthDayAmend}
		} else {
			dtf.date = lazySlot{}
			dtf.ym = lazySlot{}
			dtf.md = lazySlot{}
		}
		if !isUndefined(options.get("timeStyle")) {
			if required == "date" {
				throwType("toLocaleString of a Temporal date type does not support timeStyle option")
			}
			dtf.timeFmt = lazySlot{amend: timeAmend}
		} else {
			dtf.timeFmt = lazySlot{}
		}
	} else {
		dtf.date = lazySlot{amend: dateAmend}
		dtf.ym = lazySlot{amend: yearMonthAmend}
		dtf.md = lazySlot{amend: monthDayAmend}
		dtf.timeFmt = lazySlot{amend: timeAmend}
	}

	dtf.datetime = lazySlot{amend: datetimeAmend}
	dtf.inst = lazySlot{amend: instantAmend}

	// Save the original time zone, for a few reasons:
	// - Clearer error messages
	// - More clearly follows the spec for InitializeDateTimeFormat
	// - Because it follows the spec more closely, will make it easier to integrate
	//   support of offset strings and other potential changes like proposal-canonical-tz.
	var timeZoneOption any
	if hasOptions {
		timeZoneOption = options.get("timeZone")
	}
	if isUndefined(timeZoneOption) {
		dtf.tzOriginal = ro.TimeZone
	} else {
		id := jsToString(timeZoneOption)
		if strings.HasPrefix(id, "−") {
			// The initial (Node 23) implementation of offset time zones allowed use
			// of the Unicode minus sign, which was disallowed by a later spec change.
			throwRange("Unicode minus (U+2212) is not supported in time zone offsets")
		}
		// store a normalized identifier
		dtf.tzOriginal = toTemporalTimeZoneIdentifier(id)
	}
}

// NewDateTimeFormat is new Intl.DateTimeFormat(locales, options). locales is a string, a []string or
// nil; options a DateTimeFormatOptions or nil.
func NewDateTimeFormat(locales, options any) *DateTimeFormat {
	dtf := &DateTimeFormat{}
	internalCreateDateTimeFormat(dtf, locales, options, "any")
	return dtf
}

// This corresponds to the spec operation CreateDateTimeFormat in that it
// creates a new instance and does all the required initialization on it.
func createDateTimeFormat(locales, options any, required string) *DateTimeFormat {
	instance := &DateTimeFormat{}
	internalCreateDateTimeFormat(instance, locales, options, required)
	return instance
}

// jsToString is what String(dtf) gives.
func (f *DateTimeFormat) jsToString() string { return "[object Intl.DateTimeFormat]" }

// ResolvedOptions is Intl.DateTimeFormat.prototype.resolvedOptions.
func (f *DateTimeFormat) ResolvedOptions() ResolvedOptions {
	resolved := ResolvedOptions(f.original.ResolvedOptions())
	resolved.TimeZone = f.tzOriginal
	return resolved
}

// intlTimeClip is the step of the built-in format methods after ToNumber: TimeClip, and a RangeError
// when the result is NaN.
func intlTimeClip(x float64) float64 {
	if math.IsNaN(x) || math.Abs(x) > 8.64e15 {
		throwRange("Invalid time value")
	}
	return jsTrunc(x) + 0
}

// intlDateValue is what the built-in format methods do with a value that isn't a Temporal object:
// undefined is the current time, anything else goes through ToNumber.
func intlDateValue(date any) float64 {
	if isUndefined(date) {
		return float64(time.Now().UnixMilli())
	}
	return jsToNumber(date)
}

// Format is Intl.DateTimeFormat.prototype.format.
func (f *DateTimeFormat) Format(date any) string {
	date = unwrap(date)
	epochNs, formatter := extractOverrides(date, f)
	var ms float64
	if formatter != nil {
		ms = epochNsToMs(epochNs, "floor")
	} else {
		formatter = f.original
		ms = intlDateValue(date)
	}
	s, err := formatter.Format(intlTimeClip(ms))
	intlRaise(err)
	return s
}

// FormatToParts is Intl.DateTimeFormat.prototype.formatToParts.
func (f *DateTimeFormat) FormatToParts(date any) []Part {
	date = unwrap(date)
	epochNs, formatter := extractOverrides(date, f)
	var ms float64
	if formatter != nil {
		ms = epochNsToMs(epochNs, "floor")
	} else {
		formatter = f.original
		ms = intlDateValue(date)
	}
	parts, err := formatter.FormatToParts(intlTimeClip(ms))
	intlRaise(err)
	result := make([]Part, len(parts))
	for i, p := range parts {
		result[i] = Part{Type: p.Type, Value: p.Value}
	}
	return result
}

// formatRangeArgs is the part that formatRange and formatRangeToParts share; method names the
// exception.
func formatRangeArgs(f *DateTimeFormat, aParam, bParam any, method string) (*intl.DateTimeFormat, float64, float64) {
	aParam, bParam = unwrap(aParam), unwrap(bParam)
	if isUndefined(aParam) || isUndefined(bParam) {
		throwType("Intl.DateTimeFormat.formatRange requires two values")
	}
	a := toDateTimeFormattable(aParam)
	b := toDateTimeFormattable(bParam)
	if isFormattableTemporal(a) || isFormattableTemporal(b) {
		if !sameTemporalType(a, b) {
			throwType("Intl.DateTimeFormat." + method + " accepts two values of the same type")
		}
		aa, aformatter := extractOverrides(a, f)
		bb, bformatter := extractOverrides(b, f)
		if aformatter != nil {
			if bformatter != aformatter {
				panic("assertion failure: formatters for same Temporal type should be identical")
			}
			return aformatter, epochNsToMs(aa, "floor"), epochNsToMs(bb, "floor")
		}
	}
	// a and b are Numbers here: a pair of Temporal objects always has a formatter.
	return f.original, a.(float64), b.(float64)
}

// FormatRange is Intl.DateTimeFormat.prototype.formatRange.
func (f *DateTimeFormat) FormatRange(a, b any) string {
	formatter, x, y := formatRangeArgs(f, a, b, "formatRange")
	x = intlTimeClip(x)
	y = intlTimeClip(y)
	s, err := formatter.FormatRange(x, y)
	intlRaise(err)
	return s
}

// FormatRangeToParts is Intl.DateTimeFormat.prototype.formatRangeToParts.
func (f *DateTimeFormat) FormatRangeToParts(a, b any) []RangePart {
	formatter, x, y := formatRangeArgs(f, a, b, "formatRangeToParts")
	x = intlTimeClip(x)
	y = intlTimeClip(y)
	parts, err := formatter.FormatRangeToParts(x, y)
	intlRaise(err)
	result := make([]RangePart, len(parts))
	for i, p := range parts {
		result[i] = RangePart{Type: p.Type, Value: p.Value, Source: p.Source}
	}
	return result
}

// amend copies the options, removes the keys in amended (the original's "false" is a signal to
// delete an option) and drops the listed properties that are undefined.
func amend(optionsParam *jsOptions, amended ...string) *jsOptions {
	options := optionsParam.clone()
	props := []string{
		"era",
		"year",
		"month",
		"day",
		"hour",
		"minute",
		"second",
		"weekday",
		"dayPeriod",
		"timeZoneName",
		"dateStyle",
		"timeStyle",
	}
	for _, opt := range props {
		remove := false
		for _, a := range amended {
			if a == opt {
				remove = true
			}
		}
		v := options.get(opt)
		if remove || v == false || isUndefined(v) {
			options.del(opt)
		}
	}
	return options
}

func optionKeysString(o *jsOptions) string { return strings.Join(o.keys, ",") }

func timeAmend(originalOptions *jsOptions) *jsOptions {
	options := amend(originalOptions, "era", "year", "month", "day", "weekday", "timeZoneName", "dateStyle")
	if options.get("timeStyle") == "long" || options.get("timeStyle") == "full" {
		// Try to fake what timeStyle should do if not printing the time zone name
		options.del("timeStyle")
		options.assign("hour", "numeric", "minute", "2-digit", "second", "2-digit")
	}
	if !hasTimeOptions(options) {
		if hasAnyDateTimeOptions(originalOptions) {
			throwType("cannot format Temporal.PlainTime with options [" + optionKeysString(originalOptions) + "]")
		}
		options.assign("hour", "numeric", "minute", "numeric", "second", "numeric")
	}
	return options
}

func yearMonthAmend(originalOptions *jsOptions) *jsOptions {
	// Try to fake what dateStyle should do for dates without a day. This is not
	// accurate for locales that always print the era
	dateStyleHacks := map[string][]string{
		"short":  {"year", "2-digit", "month", "numeric"},
		"medium": {"year", "numeric", "month", "short"},
		"long":   {"year", "numeric", "month", "long"},
		"full":   {"year", "numeric", "month", "long"},
	}
	options := amend(originalOptions, "day", "hour", "minute", "second", "weekday", "dayPeriod", "timeZoneName", "timeStyle")
	if options.has("dateStyle") && jsTruthy(options.get("dateStyle")) {
		style := options.get("dateStyle")
		options.del("dateStyle")
		options.assign(dateStyleHacks[jsToString(style)]...)
	}
	if !(options.has("year") || options.has("month")) {
		if hasAnyDateTimeOptions(originalOptions) {
			throwType("cannot format PlainYearMonth with options [" + optionKeysString(originalOptions) + "]")
		}
		options.assign("year", "numeric", "month", "numeric")
	}
	return options
}

func monthDayAmend(originalOptions *jsOptions) *jsOptions {
	// Try to fake what dateStyle should do for dates without a day
	dateStyleHacks := map[string][]string{
		"short":  {"month", "numeric", "day", "numeric"},
		"medium": {"month", "short", "day", "numeric"},
		"long":   {"month", "long", "day", "numeric"},
		"full":   {"month", "long", "day", "numeric"},
	}
	options := amend(originalOptions, "era", "year", "hour", "minute", "second", "weekday", "dayPeriod", "timeZoneName", "timeStyle")
	if options.has("dateStyle") && jsTruthy(options.get("dateStyle")) {
		style := options.get("dateStyle")
		options.del("dateStyle")
		options.assign(dateStyleHacks[jsToString(style)]...)
	}
	if !(options.has("month") || options.has("day")) {
		if hasAnyDateTimeOptions(originalOptions) {
			throwType("cannot format PlainMonthDay with options [" + optionKeysString(originalOptions) + "]")
		}
		options.assign("month", "numeric", "day", "numeric")
	}
	return options
}

func dateAmend(originalOptions *jsOptions) *jsOptions {
	options := amend(originalOptions, "hour", "minute", "second", "dayPeriod", "timeZoneName", "timeStyle")
	if !hasDateOptions(options) {
		if hasAnyDateTimeOptions(originalOptions) {
			throwType("cannot format PlainDate with options [" + optionKeysString(originalOptions) + "]")
		}
		options.assign("year", "numeric", "month", "numeric", "day", "numeric")
	}
	return options
}

func datetimeAmend(originalOptions *jsOptions) *jsOptions {
	options := amend(originalOptions, "timeZoneName")
	if options.get("timeStyle") == "long" || options.get("timeStyle") == "full" {
		// Try to fake what timeStyle should do if not printing the time zone name
		options.del("timeStyle")
		options.assign("hour", "numeric", "minute", "2-digit", "second", "2-digit")

		// If moving to a fake timeStyle while dateStyle is present, we also have to
		// move to a fake dateStyle. dateStyle is mutually exclusive with hour etc.
		if jsTruthy(options.get("dateStyle")) {
			dateStyleHacks := map[string][]string{
				"short":  {"year", "numeric", "month", "numeric", "day", "numeric"},
				"medium": {"year", "numeric", "month", "short", "day", "numeric"},
				"long":   {"year", "numeric", "month", "long", "day", "numeric"},
				"full":   {"year", "numeric", "month", "long", "day", "numeric", "weekday", "long"},
			}
			options.assign(dateStyleHacks[jsToString(options.get("dateStyle"))]...)
			options.del("dateStyle")
		}
	}
	if !hasTimeOptions(options) && !hasDateOptions(options) {
		if hasAnyDateTimeOptions(originalOptions) {
			throwType("cannot format PlainDateTime with options [" + optionKeysString(originalOptions) + "]")
		}
		options.assign("year", "numeric", "month", "numeric", "day", "numeric", "hour", "numeric", "minute", "numeric", "second", "numeric")
	}
	return options
}

func instantAmend(optionsParam *jsOptions) *jsOptions {
	options := optionsParam
	if !hasTimeOptions(options) && !hasDateOptions(options) {
		options = options.clone()
		options.assign("year", "numeric", "month", "numeric", "day", "numeric", "hour", "numeric", "minute", "numeric", "second", "numeric")
	}
	return options
}

func hasDateOptions(options *jsOptions) bool {
	return options.has("year") || options.has("month") || options.has("day") || options.has("weekday") || options.has("dateStyle")
}

func hasTimeOptions(options *jsOptions) bool {
	return options.has("hour") ||
		options.has("minute") ||
		options.has("second") ||
		options.has("timeStyle") ||
		options.has("dayPeriod") ||
		options.has("fractionalSecondDigits")
}

func hasAnyDateTimeOptions(originalOptions *jsOptions) bool {
	return hasDateOptions(originalOptions) || hasTimeOptions(originalOptions)
}

// isFormattableTemporal is intl.ts's isTemporalObject: a Temporal object other than a Duration.
func isFormattableTemporal(obj any) bool {
	switch obj.(type) {
	case *PlainDate, *PlainTime, *PlainDateTime, *ZonedDateTime, *PlainYearMonth, *PlainMonthDay, *Instant:
		return true
	}
	return false
}

func toDateTimeFormattable(value any) any {
	if isFormattableTemporal(value) {
		return value
	}
	return toNumber(value)
}

func sameTemporalType(x, y any) bool {
	if !isFormattableTemporal(x) || !isFormattableTemporal(y) {
		return false
	}
	return reflect.TypeOf(x) == reflect.TypeOf(y)
}

func extractOverrides(temporalObj any, main *DateTimeFormat) (*big.Int, *intl.DateTimeFormat) {
	switch obj := temporalObj.(type) {
	case *PlainTime:
		isoDateTime := isoDateTime{
			isoDate: isoDate{year: 1970, month: 1, day: 1},
			time:    obj.time,
		}
		formatter := getSlotLazy(main, &main.timeFmt)
		if formatter == nil {
			throwType("cannot format PlainTime with only date options")
		}
		return getEpochNanosecondsFor(main.tzCanonical, isoDateTime, "compatible"), formatter

	case *PlainYearMonth:
		calendar := obj.calendar
		mainCalendar := main.calID
		if calendar != mainCalendar {
			throwRange("cannot format PlainYearMonth with calendar " + calendar + " in locale with calendar " + mainCalendar)
		}
		isoDateTime := combineISODateAndTimeRecord(obj.isoDate, noonTimeRecord())
		formatter := getSlotLazy(main, &main.ym)
		if formatter == nil {
			throwType("cannot format PlainYearMonth with only time options")
		}
		return getEpochNanosecondsFor(main.tzCanonical, isoDateTime, "compatible"), formatter

	case *PlainMonthDay:
		calendar := obj.calendar
		mainCalendar := main.calID
		if calendar != mainCalendar {
			throwRange("cannot format PlainMonthDay with calendar " + calendar + " in locale with calendar " + mainCalendar)
		}
		isoDateTime := combineISODateAndTimeRecord(obj.isoDate, noonTimeRecord())
		formatter := getSlotLazy(main, &main.md)
		if formatter == nil {
			throwType("cannot format PlainMonthDay with only time options")
		}
		return getEpochNanosecondsFor(main.tzCanonical, isoDateTime, "compatible"), formatter

	case *PlainDate:
		calendar := obj.calendar
		mainCalendar := main.calID
		if calendar != "iso8601" && calendar != mainCalendar {
			throwRange("cannot format PlainDate with calendar " + calendar + " in locale with calendar " + mainCalendar)
		}
		isoDateTime := combineISODateAndTimeRecord(obj.isoDate, noonTimeRecord())
		formatter := getSlotLazy(main, &main.date)
		if formatter == nil {
			throwType("cannot format PlainDate with only time options")
		}
		return getEpochNanosecondsFor(main.tzCanonical, isoDateTime, "compatible"), formatter

	case *PlainDateTime:
		calendar := obj.calendar
		mainCalendar := main.calID
		if calendar != "iso8601" && calendar != mainCalendar {
			throwRange("cannot format PlainDateTime with calendar " + calendar + " in locale with calendar " + mainCalendar)
		}
		epochNs := getEpochNanosecondsFor(main.tzCanonical, obj.isoDateTime, "compatible")
		return epochNs, getSlotLazy(main, &main.datetime)

	case *ZonedDateTime:
		throwType("Temporal.ZonedDateTime not supported in DateTimeFormat methods. Use toLocaleString() instead.")

	case *Instant:
		return obj.epochNs, getSlotLazy(main, &main.inst)
	}
	return nil, nil
}

func temporalDurationToCompatibilityRecord(duration *Duration) intl.DurationRecord {
	return intl.DurationRecord{
		Years:        duration.years,
		Months:       duration.months,
		Weeks:        duration.weeks,
		Days:         duration.days,
		Hours:        duration.hours,
		Minutes:      duration.minutes,
		Seconds:      duration.seconds,
		Milliseconds: duration.milliseconds,
		Microseconds: duration.microseconds,
		Nanoseconds:  duration.nanoseconds,
	}
}

// modifiedIntlDurationFormatPrototypeFormat is the polyfill's replacement for
// Intl.DurationFormat.prototype.format.
func modifiedIntlDurationFormatPrototypeFormat(formatter *intl.DurationFormat, durationLike any) string {
	duration := toTemporalDuration(durationLike)
	record := temporalDurationToCompatibilityRecord(duration)
	s, err := formatter.Format(record)
	intlRaise(err)
	return s
}

// durationFormatOptionKeys lists the properties new Intl.DurationFormat reads.
var durationFormatOptionKeys = []string{
	"localeMatcher", "numberingSystem", "style",
	"years", "yearsDisplay", "months", "monthsDisplay", "weeks", "weeksDisplay", "days", "daysDisplay",
	"hours", "hoursDisplay", "minutes", "minutesDisplay", "seconds", "secondsDisplay",
	"milliseconds", "millisecondsDisplay", "microseconds", "microsecondsDisplay",
	"nanoseconds", "nanosecondsDisplay", "fractionalDigits",
}

// newIntlDurationFormat is new Intl.DurationFormat(locales, options).
func newIntlDurationFormat(locales, optionsParam any) *intl.DurationFormat {
	optionsParam = unwrap(optionsParam)
	locale := intlLocales(locales)
	options := newJSOptions()
	if !isUndefined(optionsParam) {
		if !isObject(optionsParam) {
			// The locale list is canonicalized before the options are looked at.
			_, err := intl.NewDurationFormat(locale, map[string]any{})
			intlRaise(err)
			throwType("invalid_argument")
		}
		for _, key := range durationFormatOptionKeys {
			if v := objGet(optionsParam, key); !isUndefined(v) {
				options.set(key, v)
			}
		}
	}
	f, err := intl.NewDurationFormat(locale, icuOptions(options, nil, dfNumberOptions))
	intlRaise(err)
	return f
}
