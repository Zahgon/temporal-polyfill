package temporal

import (
	"math"
	"math/big"
	"reflect"
)

const (
	dayMs float64 = 86400_000
	// dayNanos is DAY_NANOS.
	dayNanos    float64 = dayMs * 1e6
	minuteNanos float64 = 60e9
	// Instant range is 100 million days (inclusive) before or after epoch.
	msMax float64 = dayMs * 1e8
	// The pattern of leap years in the ISO 8601 calendar repeats every 400 years.
	// The constant below is the number of nanoseconds in 400 years. It is used to
	// avoid overflows when dealing with values at the edge legacy Date's range.
	msIn400YearCycle float64 = (400*365 + 97) * dayMs
	yearMin          float64 = -271821
	yearMax          float64 = 275760
	// beforeFirstDST is Date.UTC(1847, 0, 1), 1847-01-01T00:00:00Z.
	beforeFirstDST float64 = -3881520000000
)

var (
	nsMax = new(big.Int).Mul(bigFromNumber(msMax), bigMillion)
	nsMin = new(big.Int).Neg(nsMax)
	// PlainDateTime range is 24 hours wider (exclusive) than the Instant range on
	// both ends, to allow for valid Instant=>PlainDateTime conversion for all
	// built-in time zones (whose offsets must have a magnitude less than 24 hours).
	datetimeNsMin = new(big.Int).Add(new(big.Int).Sub(nsMin, dayNanosJSBI), bigOne)
	datetimeNsMax = new(big.Int).Sub(new(big.Int).Add(nsMax, dayNanosJSBI), bigOne)
)

var builtinCalendarIDs = []string{
	"iso8601",
	"hebrew",
	"islamic-umalqura",
	"islamic-tbla",
	"islamic-civil",
	"islamicc",
	"persian",
	"ethiopic",
	"ethioaa",
	"ethiopic-amete-alem",
	"coptic",
	"chinese",
	"dangi",
	"roc",
	"indian",
	"buddhist",
	"japanese",
	"gregory",
}

var icuLegacyTimeZoneIDs = map[string]bool{
	"ACT": true,
	"AET": true,
	"AGT": true,
	"ART": true,
	"AST": true,
	"BET": true,
	"BST": true,
	"CAT": true,
	"CNT": true,
	"CST": true,
	"CTT": true,
	"EAT": true,
	"ECT": true,
	"IET": true,
	"IST": true,
	"JST": true,
	"MIT": true,
	"NET": true,
	"NST": true,
	"PLT": true,
	"PNT": true,
	"PRT": true,
	"PST": true,
	"SST": true,
	"VST": true,
}

// debugBuild is DEBUG: the build the port reproduces is the debug build. Its other effect, the
// _repr_ property, is not observable from Go and cannot throw on the values that reach it.
const debugBuild = true

// assert is assert of lib/assert.ts.
func assert(condition bool, message string) {
	if !condition {
		panic("assertion failure: " + message)
	}
}

// assertNotReached is assertNotReached of lib/assert.ts.
func assertNotReached(message ...string) {
	reason := ""
	if len(message) > 0 && message[0] != "" {
		reason = " because " + message[0]
	}
	panic("assertion failure: code should not be reached" + reason)
}

// assertExists verifies that the given argument "exists" (is not nil or a nil pointer).
func assertExists(arg any) {
	if arg == nil {
		panic("Expected arg to be set.")
	}
	rv := reflect.ValueOf(arg)
	if rv.Kind() == reflect.Pointer && rv.IsNil() {
		panic("Expected arg to be set.")
	}
}

// castExists is similar to assertExists, but returns the argument.
func castExists[A any](arg *A) A {
	assertExists(arg)
	return *arg
}

// jsCatch runs body and reports whether it threw a JavaScript exception, like a bare catch block.
func jsCatch(body func()) (threw bool) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(*Error); !ok {
				panic(r)
			}
			threw = true
		}
	}()
	body()
	return false
}

// optionalArg reads a trailing optional argument; absent is undefined.
func optionalArg(args []any) any {
	if len(args) == 0 {
		return nil
	}
	return unwrap(args[0])
}

// optionalString reads a string-valued argument that may be undefined: nil, a *string, a *Value or a
// string.
func optionalString(x any) (string, bool) {
	switch v := unwrap(x).(type) {
	case string:
		return v, true
	case *string:
		if v != nil {
			return *v, true
		}
	}
	return "", false
}

// floatOr is `value ?? fallback` for a Number that may be undefined.
func floatOr(value *float64, fallback float64) float64 {
	if value != nil {
		return *value
	}
	return fallback
}

// undefinedToNaN reads a Number that may be undefined where arithmetic would turn undefined to NaN.
func undefinedToNaN(value *float64) float64 {
	return floatOr(value, math.NaN())
}

// toNumber is ToNumber.
func toNumber(value any) float64 {
	// ES 2022's es-abstract made minor changes to ToNumber, but polyfilling these
	// changes adds zero benefit to Temporal and brings in a lot of extra code. So
	// we'll leave ToNumber as-is.
	// See https://github.com/ljharb/es-abstract/blob/main/2022/ToNumber.js
	if _, ok := unwrap(value).(*big.Int); ok {
		throwType("Cannot convert BigInt to number")
	}
	return jsToNumber(value)
}

func isIntegralNumber(argument any) bool {
	number, ok := unwrap(argument).(float64)
	if !ok || math.IsNaN(number) || math.IsInf(number, 0) {
		return false
	}
	absValue := math.Abs(number)
	return math.Floor(absValue) == absValue
}

// toString is ToString.
func toString(value any) string {
	return jsToString(value)
}

// toIntegerWithTruncation is ToIntegerWithTruncation.
func toIntegerWithTruncation(value any) float64 {
	number := toNumber(value)
	if number == 0 {
		return 0
	}
	if math.IsNaN(number) || math.IsInf(number, 0) {
		throwRange("invalid number value")
	}
	integer := jsTrunc(number)
	if integer == 0 {
		return 0 // ℝ(value) in spec text; converts -0 to 0
	}
	return integer
}

func toPositiveIntegerWithTruncation(valueParam any, property string) float64 {
	integer := toIntegerWithTruncation(valueParam)
	if integer <= 0 {
		throwRange("property '" + property + "' cannot be a a number less than one")
	}
	return integer
}

// toIntegerIfIntegral is ToIntegerIfIntegral.
func toIntegerIfIntegral(valueParam any) float64 {
	number := toNumber(valueParam)
	if math.IsNaN(number) {
		throwRange("not a number")
	}
	if math.IsInf(number, 0) {
		throwRange("infinity is out of range")
	}
	if !isIntegralNumber(number) {
		throwRange("unsupported fractional value " + jsToString(valueParam))
	}
	if number == 0 {
		return 0 // ℝ(value) in spec text; converts -0 to 0
	}
	return number
}

func toZeroPaddedDecimalString(n float64, minLength int) string {
	if debugBuild {
		if !isIntegralNumber(n) || n < 0 {
			throwRange("Assertion failed: `${n}` must be a non-negative integer")
		}
	}
	return padStartZeros(numberToString(n), minLength)
}

// requireString isn't in the spec, but is useful in the polyfill for DRY and better error messages.
func requireString(valueParam any) string {
	value := unwrap(valueParam)
	s, ok := value.(string)
	if !ok {
		throwType("expected a string, not " + jsToString(value))
	}
	return s
}

func toMonthCode(value any) string {
	monthNumber, isLeapMonth := parseMonthCode(value)
	return createMonthCode(monthNumber, isLeapMonth)
}

func toOffsetString(valueParam any) string {
	value := toPrimitiveRequireString(valueParam)
	parseDateTimeUTCOffset(value)
	return value
}

var calendarFieldKeys = []string{
	"era",
	"eraYear",
	"year",
	"month",
	"monthCode",
	"day",
	"hour",
	"minute",
	"second",
	"millisecond",
	"microsecond",
	"nanosecond",
	"offset",
	"timeZone",
}

// castBuiltinField stores the value of one field through BUILTIN_CASTS.
func castBuiltinField(result *calendarFields, property string, value any) {
	ptrF := func(x float64) *float64 { return &x }
	ptrS := func(x string) *string { return &x }
	switch property {
	case "era":
		result.era = ptrS(toString(value))
	case "eraYear":
		result.eraYear = ptrF(toIntegerWithTruncation(value))
	case "year":
		result.year = ptrF(toIntegerWithTruncation(value))
	case "month":
		result.month = ptrF(toPositiveIntegerWithTruncation(value, "month"))
	case "monthCode":
		result.monthCode = ptrS(toMonthCode(value))
	case "day":
		result.day = ptrF(toPositiveIntegerWithTruncation(value, "day"))
	case "hour":
		result.hour = ptrF(toIntegerWithTruncation(value))
	case "minute":
		result.minute = ptrF(toIntegerWithTruncation(value))
	case "second":
		result.second = ptrF(toIntegerWithTruncation(value))
	case "millisecond":
		result.millisecond = ptrF(toIntegerWithTruncation(value))
	case "microsecond":
		result.microsecond = ptrF(toIntegerWithTruncation(value))
	case "nanosecond":
		result.nanosecond = ptrF(toIntegerWithTruncation(value))
	case "offset":
		result.offset = ptrS(toOffsetString(value))
	case "timeZone":
		result.timeZone = ptrS(toTemporalTimeZoneIdentifier(value))
	default:
		panic("Expected arg to be set.")
	}
}

// setBuiltinDefault stores BUILTIN_DEFAULTS[property]; the fields without a default stay undefined.
func setBuiltinDefault(result *calendarFields, property string) {
	zero := func() *float64 { x := 0.0; return &x }
	switch property {
	case "hour":
		result.hour = zero()
	case "minute":
		result.minute = zero()
	case "second":
		result.second = zero()
	case "millisecond":
		result.millisecond = zero()
	case "microsecond":
		result.microsecond = zero()
	case "nanosecond":
		result.nanosecond = zero()
	}
}

// fieldIsPresent is fields[key] !== undefined.
func fieldIsPresent(fields *calendarFields, key string) bool {
	switch key {
	case "era":
		return fields.era != nil
	case "eraYear":
		return fields.eraYear != nil
	case "year":
		return fields.year != nil
	case "month":
		return fields.month != nil
	case "monthCode":
		return fields.monthCode != nil
	case "day":
		return fields.day != nil
	case "hour":
		return fields.hour != nil
	case "minute":
		return fields.minute != nil
	case "second":
		return fields.second != nil
	case "millisecond":
		return fields.millisecond != nil
	case "microsecond":
		return fields.microsecond != nil
	case "nanosecond":
		return fields.nanosecond != nil
	case "offset":
		return fields.offset != nil
	case "timeZone":
		return fields.timeZone != nil
	}
	return false
}

// copyField is target[key] = source[key].
func copyField(target, source *calendarFields, key string) {
	switch key {
	case "era":
		target.era = source.era
	case "eraYear":
		target.eraYear = source.eraYear
	case "year":
		target.year = source.year
	case "month":
		target.month = source.month
	case "monthCode":
		target.monthCode = source.monthCode
	case "day":
		target.day = source.day
	case "hour":
		target.hour = source.hour
	case "minute":
		target.minute = source.minute
	case "second":
		target.second = source.second
	case "millisecond":
		target.millisecond = source.millisecond
	case "microsecond":
		target.microsecond = source.microsecond
	case "nanosecond":
		target.nanosecond = source.nanosecond
	case "offset":
		target.offset = source.offset
	case "timeZone":
		target.timeZone = source.timeZone
	}
}

// temporalUnits is TEMPORAL_UNITS: each item is [plural, singular, category].
var temporalUnits = [][3]string{
	{"years", "year", "date"},
	{"months", "month", "date"},
	{"weeks", "week", "date"},
	{"days", "day", "date"},
	{"hours", "hour", "time"},
	{"minutes", "minute", "time"},
	{"seconds", "second", "time"},
	{"milliseconds", "millisecond", "time"},
	{"microseconds", "microsecond", "time"},
	{"nanoseconds", "nanosecond", "time"},
}

// singularFor is SINGULAR_FOR.
var singularFor = func() map[string]string {
	result := map[string]string{}
	for _, unit := range temporalUnits {
		result[unit[0]] = unit[1]
	}
	return result
}()

// unitsDescending is UNITS_DESCENDING.
var unitsDescending = func() []string {
	result := make([]string, 0, len(temporalUnits))
	for _, unit := range temporalUnits {
		result = append(result, unit[1])
	}
	return result
}()

// nsPerTimeUnit is NS_PER_TIME_UNIT.
var nsPerTimeUnit = map[string]float64{
	"day":         dayNanos,
	"hour":        3600e9,
	"minute":      60e9,
	"second":      1e9,
	"millisecond": 1e6,
	"microsecond": 1e3,
	"nanosecond":  1,
}

var durationFields = []string{
	"days",
	"hours",
	"microseconds",
	"milliseconds",
	"minutes",
	"months",
	"nanoseconds",
	"seconds",
	"weeks",
	"years",
}

// toObject is ToObject. Primitives have no properties of interest here, so they pass through.
func toObject(valueParam any) any {
	value := unwrap(valueParam)
	if value == nil || isNull(value) {
		throwType("Expected object not " + jsToString(value))
	}
	return value
}

// isTemporalInstant is IsTemporalInstant.
func isTemporalInstant(item any) bool { _, ok := unwrap(item).(*Instant); return ok }

// isTemporalDuration is IsTemporalDuration.
func isTemporalDuration(item any) bool { _, ok := unwrap(item).(*Duration); return ok }

// isTemporalDate is IsTemporalDate.
func isTemporalDate(item any) bool { _, ok := unwrap(item).(*PlainDate); return ok }

// isTemporalTime is IsTemporalTime.
func isTemporalTime(item any) bool { _, ok := unwrap(item).(*PlainTime); return ok }

// isTemporalDateTime is IsTemporalDateTime.
func isTemporalDateTime(item any) bool { _, ok := unwrap(item).(*PlainDateTime); return ok }

// isTemporalYearMonth is IsTemporalYearMonth.
func isTemporalYearMonth(item any) bool { _, ok := unwrap(item).(*PlainYearMonth); return ok }

// isTemporalMonthDay is IsTemporalMonthDay.
func isTemporalMonthDay(item any) bool { _, ok := unwrap(item).(*PlainMonthDay); return ok }

// isTemporalZonedDateTime is IsTemporalZonedDateTime.
func isTemporalZonedDateTime(item any) bool { _, ok := unwrap(item).(*ZonedDateTime); return ok }

// calendarSlot is GetSlot(item, CALENDAR) guarded by HasSlot(item, CALENDAR).
func calendarSlot(item any) (string, bool) {
	switch v := unwrap(item).(type) {
	case *PlainDate:
		return v.calendar, true
	case *PlainDateTime:
		return v.calendar, true
	case *PlainYearMonth:
		return v.calendar, true
	case *PlainMonthDay:
		return v.calendar, true
	case *ZonedDateTime:
		return v.calendar, true
	}
	return "", false
}

// rejectTemporalLikeObject is RejectTemporalLikeObject.
func rejectTemporalLikeObject(itemParam any) {
	item := unwrap(itemParam)
	if _, ok := calendarSlot(item); ok || isTemporalZonedDateTime(item) {
		throwType("with() does not support a calendar or timeZone property")
	}
	if isTemporalTime(item) {
		throwType("with() does not accept Temporal.PlainTime, use withPlainTime() instead")
	}
	if objGet(item, "calendar") != nil {
		throwType("with() does not support a calendar property")
	}
	if objGet(item, "timeZone") != nil {
		throwType("with() does not support a timeZone property")
	}
}

func formatCalendarAnnotation(id, showCalendar string) string {
	if showCalendar == "never" {
		return ""
	}
	if showCalendar == "auto" && id == "iso8601" {
		return ""
	}
	flag := ""
	if showCalendar == "critical" {
		flag = "!"
	}
	return "[" + flag + "u-ca=" + id + "]"
}

// subSecondNanosecondsOf is the sum the time formatters share.
func subSecondNanosecondsOf(t timeRecord) float64 {
	return float64(t.millisecond*1e6) + float64(t.microsecond*1e3) + t.nanosecond
}
