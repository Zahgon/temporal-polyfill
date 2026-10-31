package temporal

import (
	"math"
	"math/big"
	"os"
	"slices"
	"strings"
	"sync"
	"unicode/utf16"

	"github.com/js-temporal/temporal-polyfill-go/internal/intl"
)

// jsMax is Math.max of two Numbers: NaN wins over everything, +0 over -0.
func jsMax(a, b float64) float64 {
	if math.IsNaN(a) || math.IsNaN(b) {
		return math.NaN()
	}
	return math.Max(a, b)
}

// jsMin is Math.min of two Numbers: NaN wins over everything, -0 over +0.
func jsMin(a, b float64) float64 {
	if math.IsNaN(a) || math.IsNaN(b) {
		return math.NaN()
	}
	return math.Min(a, b)
}

func leapYear(year float64) bool {
	isDiv4 := jsMod(year, 4) == 0
	isDiv100 := jsMod(year, 100) == 0
	isDiv400 := jsMod(year, 400) == 0
	return isDiv4 && (!isDiv100 || isDiv400)
}

var (
	daysOfMonthStandard = [12]float64{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	daysOfMonthLeapYear = [12]float64{31, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
)

// isoDaysInMonth returns NaN where the original's table lookup gives undefined.
func isoDaysInMonth(year, month float64) float64 {
	index := month - 1
	if index != math.Trunc(index) || index < 0 || index > 11 {
		return math.NaN()
	}
	if leapYear(year) {
		return daysOfMonthLeapYear[int(index)]
	}
	return daysOfMonthStandard[int(index)]
}

func durationSign(duration *Duration) float64 {
	fields := [...]float64{
		duration.years,
		duration.months,
		duration.weeks,
		duration.days,
		duration.hours,
		duration.minutes,
		duration.seconds,
		duration.milliseconds,
		duration.microseconds,
		duration.nanoseconds,
	}
	for _, prop := range fields {
		if prop != 0 {
			if prop < 0 {
				return -1
			}
			return 1
		}
	}
	return 0
}

func dateDurationSign(d dateDuration) float64 {
	for _, prop := range [...]float64{d.years, d.months, d.weeks, d.days} {
		if prop != 0 {
			if prop < 0 {
				return -1
			}
			return 1
		}
	}
	return 0
}

func internalDurationSign(duration internalDuration) float64 {
	dateSign := dateDurationSign(duration.date)
	if dateSign != 0 {
		return dateSign
	}
	return duration.time.sign()
}

// isoYearMonth is what BalanceISOYearMonth returns.
type isoYearMonth struct{ year, month float64 }

func balanceISOYearMonth(yearParam, monthParam float64) isoYearMonth {
	year := yearParam
	month := monthParam
	if !isFiniteNumber(year) || !isFiniteNumber(month) {
		panic("assertion failure: BalanceISOYearMonth: infinity is out of range")
	}
	month -= 1
	year += math.Floor(month / 12)
	month = jsMod(month, 12)
	if month < 0 {
		month += 12
	}
	month += 1
	return isoYearMonth{year, month}
}

func addDaysToISODate(iso isoDate, days float64) isoDate {
	day := iso.day + days
	balanced := balanceISOYearMonth(iso.year, iso.month)
	year, month := balanced.year, balanced.month

	// The pattern of leap years in the ISO 8601 calendar repeats every 400
	// years. So if we have more than 400 years in days, there's no need to
	// convert days to a year 400 times. We can convert a multiple of 400 all at
	// once.
	const daysIn400YearCycle = 400*365 + 97
	if math.Abs(day) > daysIn400YearCycle {
		nCycles := jsTrunc(day / daysIn400YearCycle)
		year += float64(400 * nCycles)
		day -= float64(nCycles * daysIn400YearCycle)
	}

	daysInYearOf := func(y float64) float64 {
		if leapYear(y) {
			return 366
		}
		return 365
	}
	testYear := year - 1
	if month > 2 {
		testYear = year
	}
	for daysInYear := daysInYearOf(testYear); day < -daysInYear; daysInYear = daysInYearOf(testYear) {
		year -= 1
		testYear -= 1
		day += daysInYear
	}
	testYear += 1
	for daysInYear := daysInYearOf(testYear); day > daysInYear; daysInYear = daysInYearOf(testYear) {
		year += 1
		testYear += 1
		day -= daysInYear
	}

	for day < 1 {
		balanced = balanceISOYearMonth(year, month-1)
		year, month = balanced.year, balanced.month
		day += isoDaysInMonth(year, month)
	}
	for day > isoDaysInMonth(year, month) {
		day -= isoDaysInMonth(year, month)
		balanced = balanceISOYearMonth(year, month+1)
		year, month = balanced.year, balanced.month
	}

	return isoDate{year, month, day}
}

func balanceISODateTime(year, month, day, hour, minute, second, millisecond, microsecond, nanosecond float64) isoDateTime {
	tm := balanceTime(hour, minute, second, millisecond, microsecond, nanosecond)
	iso := addDaysToISODate(isoDate{year, month, day}, tm.deltaDays)
	return combineISODateAndTimeRecord(iso, tm)
}

func balanceTime(hourParam, minuteParam, secondParam, millisecondParam, microsecondParam, nanosecondParam float64) timeRecord {
	hour := hourParam
	minute := minuteParam
	second := secondParam
	millisecond := millisecondParam
	microsecond := microsecondParam
	nanosecond := nanosecondParam

	div, mod := truncatingDivModByPowerOf10(nanosecond, 3)
	nanosecond = mod
	microsecond += div
	if nanosecond < 0 {
		microsecond -= 1
		nanosecond += 1000
	}

	div, mod = truncatingDivModByPowerOf10(microsecond, 3)
	microsecond = mod
	millisecond += div
	if microsecond < 0 {
		millisecond -= 1
		microsecond += 1000
	}

	second += jsTrunc(millisecond / 1000)
	millisecond = jsMod(millisecond, 1000)
	if millisecond < 0 {
		second -= 1
		millisecond += 1000
	}

	minute += jsTrunc(second / 60)
	second = jsMod(second, 60)
	if second < 0 {
		minute -= 1
		second += 60
	}

	hour += jsTrunc(minute / 60)
	minute = jsMod(minute, 60)
	if minute < 0 {
		hour -= 1
		minute += 60
	}

	deltaDays := jsTrunc(hour / 24)
	hour = jsMod(hour, 24)
	if hour < 0 {
		deltaDays -= 1
		hour += 24
	}

	// Results are possibly -0 at this point, but these are mathematical values in
	// the spec. Force -0 to +0.
	deltaDays += 0
	hour += 0
	minute += 0
	second += 0
	millisecond += 0
	microsecond += 0
	nanosecond += 0

	return timeRecord{deltaDays, hour, minute, second, millisecond, microsecond, nanosecond}
}

// constrainToRange takes NaN where the original takes undefined; Math.max gives NaN for both.
func constrainToRange(value, min, max float64) float64 {
	return jsMin(max, jsMax(min, value))
}

// constrainISODate takes NaN for an undefined dayParam.
func constrainISODate(year, monthParam, dayParam float64) isoDate {
	month := constrainToRange(monthParam, 1, 12)
	day := constrainToRange(dayParam, 1, isoDaysInMonth(year, month))
	return isoDate{year, month, day}
}

func rejectToRange(value, min, max float64) {
	if value < min || value > max {
		throwRange("value out of range: " + numberToString(min) + " <= " + numberToString(value) + " <= " + numberToString(max))
	}
}

func rejectISODate(year, month, day float64) {
	rejectToRange(month, 1, 12)
	rejectToRange(day, 1, isoDaysInMonth(year, month))
}

func rejectDateRange(iso isoDate) {
	// Noon avoids trouble at edges of DateTime range (excludes midnight)
	dt := combineISODateAndTimeRecord(iso, noonTimeRecord())
	ns := getUTCEpochNanoseconds(dt)
	if ns.Cmp(datetimeNsMin) < 0 || ns.Cmp(datetimeNsMax) > 0 {
		dateString := isoDateToString(dt.isoDate)
		throwRange(dateString + " is outside of supported range")
	}
}

func rejectTime(hour, minute, second, millisecond, microsecond, nanosecond float64) {
	rejectToRange(hour, 0, 23)
	rejectToRange(minute, 0, 59)
	rejectToRange(second, 0, 59)
	rejectToRange(millisecond, 0, 999)
	rejectToRange(microsecond, 0, 999)
	rejectToRange(nanosecond, 0, 999)
}

func rejectDateTime(year, month, day, hour, minute, second, millisecond, microsecond, nanosecond float64) {
	rejectISODate(year, month, day)
	rejectTime(hour, minute, second, millisecond, microsecond, nanosecond)
}

func rejectDateTimeRange(dt isoDateTime) {
	ns := getUTCEpochNanoseconds(dt)
	if ns.Cmp(datetimeNsMin) < 0 || ns.Cmp(datetimeNsMax) > 0 {
		dateTimeString := isoDateTimeToString(dt, "iso8601", "auto", "never")
		throwRange(dateTimeString + " is outside of supported range")
	}
}

// Same as above, but throws a different, non-user-facing error
func assertISODateTimeWithinLimits(dt isoDateTime) {
	ns := getUTCEpochNanoseconds(dt)
	if ns.Cmp(datetimeNsMin) < 0 || ns.Cmp(datetimeNsMax) > 0 {
		panic("assertion failure: " + isoDateTimeToString(dt, "iso8601", "auto", "never") + " is outside the representable range")
	}
}

// In the spec, IsValidEpochNanoseconds returns a boolean and call sites are
// responsible for throwing. In the polyfill, ValidateEpochNanoseconds takes its
// place so that we can DRY the throwing code.
func validateEpochNanoseconds(epochNanoseconds *big.Int) {
	if epochNanoseconds.Cmp(nsMin) < 0 || epochNanoseconds.Cmp(nsMax) > 0 {
		throwRange("date/time value is outside of supported range")
	}
}

func rejectYearMonthRange(iso isoDate) {
	rejectToRange(iso.year, yearMin, yearMax)
	if iso.year == yearMin {
		rejectToRange(iso.month, 4, 12)
	} else if iso.year == yearMax {
		rejectToRange(iso.month, 1, 9)
	}
}

func roundNumberToIncrement(quantity, increment float64, mode string) float64 {
	quotient := jsTrunc(quantity / increment)
	remainder := jsMod(quantity, increment)
	sign := "positive"
	if quantity < 0 {
		sign = "negative"
	}
	r1 := math.Abs(quotient)
	r2 := r1 + 1
	cmp := comparisonResult(math.Abs(float64(remainder*2)) - increment)
	even := jsMod(r1, 2) == 0
	unsignedRoundingMode := getUnsignedRoundingMode(mode, sign)
	rounded := r1
	if remainder != 0 {
		rounded = applyUnsignedRoundingMode(r1, r2, cmp, even, unsignedRoundingMode)
	}
	if sign == "positive" {
		return float64(increment * rounded)
	}
	return float64(increment * -rounded)
}

func roundNumberToIncrementAsIfPositive(quantity, increment *big.Int, mode string) *big.Int {
	quotient := new(big.Int).Quo(quantity, increment)
	remainder := new(big.Int).Rem(quantity, increment)
	unsignedRoundingMode := getUnsignedRoundingMode(mode, "positive")
	var r1, r2 *big.Int
	if quantity.Sign() < 0 {
		r1 = new(big.Int).Sub(quotient, bigOne)
		r2 = quotient
	} else {
		r1 = quotient
		r2 = new(big.Int).Add(quotient, bigOne)
	}
	// Similar to the comparison in RoundNumberToIncrement, but multiplied by an
	// extra sign to make sure we treat it as positive
	cmp := compare(abs(new(big.Int).Mul(remainder, bigTwo)), increment)
	if quantity.Sign() < 0 {
		cmp = float64(cmp*-1) + 0
	}
	rounded := quotient
	if remainder.Sign() != 0 {
		rounded = applyUnsignedRoundingMode(r1, r2, cmp, isEven(r1), unsignedRoundingMode)
	}
	return new(big.Int).Mul(rounded, increment)
}

func roundTemporalInstant(epochNs *big.Int, increment float64, unit, roundingMode string) *big.Int {
	incrementNs := float64(nsPerTimeUnit[unit] * increment)
	return roundNumberToIncrementAsIfPositive(epochNs, bigFromNumber(incrementNs), roundingMode)
}

func roundISODateTime(dt isoDateTime, increment float64, unit, roundingMode string) isoDateTime {
	assertISODateTimeWithinLimits(dt)
	tm := roundTime(dt.time, increment, unit, roundingMode)
	iso := addDaysToISODate(dt.isoDate, tm.deltaDays)
	return combineISODateAndTimeRecord(iso, tm)
}

func roundTime(t timeRecord, increment float64, unit, roundingMode string) timeRecord {
	hour, minute, second := t.hour, t.minute, t.second
	millisecond, microsecond, nanosecond := t.millisecond, t.microsecond, t.nanosecond
	var quantity float64
	switch unit {
	case "day", "hour":
		quantity = float64(hour*60) + minute
		quantity = float64(quantity*60) + second
		quantity = float64(quantity*1000) + millisecond
		quantity = float64(quantity*1000) + microsecond
		quantity = float64(quantity*1000) + nanosecond
	case "minute":
		quantity = float64(minute*60) + second
		quantity = float64(quantity*1000) + millisecond
		quantity = float64(quantity*1000) + microsecond
		quantity = float64(quantity*1000) + nanosecond
	case "second":
		quantity = float64(second*1000) + millisecond
		quantity = float64(quantity*1000) + microsecond
		quantity = float64(quantity*1000) + nanosecond
	case "millisecond":
		quantity = float64(millisecond*1000) + microsecond
		quantity = float64(quantity*1000) + nanosecond
	case "microsecond":
		quantity = float64(microsecond*1000) + nanosecond
	case "nanosecond":
		quantity = nanosecond
	default:
		quantity = math.NaN()
	}
	nsPerUnit := nsPerTimeUnit[unit]
	result := roundNumberToIncrement(quantity, float64(nsPerUnit*increment), roundingMode) / nsPerUnit
	switch unit {
	case "day":
		return timeRecord{deltaDays: result}
	case "hour":
		return balanceTime(result, 0, 0, 0, 0, 0)
	case "minute":
		return balanceTime(hour, result, 0, 0, 0, 0)
	case "second":
		return balanceTime(hour, minute, result, 0, 0, 0)
	case "millisecond":
		return balanceTime(hour, minute, second, result, 0, 0)
	case "microsecond":
		return balanceTime(hour, minute, second, millisecond, result, 0)
	case "nanosecond":
		return balanceTime(hour, minute, second, millisecond, microsecond, result)
	}
	panic(&Error{Name: "Error", Message: "Invalid unit " + unit})
}

func roundTimeDuration(td *timeDuration, increment float64, unit, roundingMode string) *timeDuration {
	// unit must be a time unit
	divisor := nsPerTimeUnit[unit]
	return td.round(bigFromNumber(float64(divisor*increment)), roundingMode)
}

func totalTimeDuration(td *timeDuration, unit string) float64 {
	divisor := nsPerTimeUnit[unit]
	return td.fdiv(bigFromNumber(divisor))
}

func compareISODate(isoDate1, isoDate2 isoDate) float64 {
	if isoDate1.year != isoDate2.year {
		return comparisonResult(isoDate1.year - isoDate2.year)
	}
	if isoDate1.month != isoDate2.month {
		return comparisonResult(isoDate1.month - isoDate2.month)
	}
	if isoDate1.day != isoDate2.day {
		return comparisonResult(isoDate1.day - isoDate2.day)
	}
	return 0
}

func compareTimeRecord(time1, time2 timeRecord) float64 {
	if time1.hour != time2.hour {
		return comparisonResult(time1.hour - time2.hour)
	}
	if time1.minute != time2.minute {
		return comparisonResult(time1.minute - time2.minute)
	}
	if time1.second != time2.second {
		return comparisonResult(time1.second - time2.second)
	}
	if time1.millisecond != time2.millisecond {
		return comparisonResult(time1.millisecond - time2.millisecond)
	}
	if time1.microsecond != time2.microsecond {
		return comparisonResult(time1.microsecond - time2.microsecond)
	}
	if time1.nanosecond != time2.nanosecond {
		return comparisonResult(time1.nanosecond - time2.nanosecond)
	}
	return 0
}

func compareISODateTime(isoDateTime1, isoDateTime2 isoDateTime) float64 {
	dateResult := compareISODate(isoDateTime1.isoDate, isoDateTime2.isoDate)
	if dateResult != 0 {
		return dateResult
	}
	return compareTimeRecord(isoDateTime1.time, isoDateTime2.time)
}

func toBigIntExternal(arg any) *big.Int {
	return toBigInt(arg)
}

// rounding modes supported: floor, ceil, trunc
func epochNsToMs(epochNanoseconds *big.Int, mode string) float64 {
	quotient, remainder := divmod(epochNanoseconds, bigMillion)
	epochMilliseconds := bigToNumber(quotient)
	if mode == "floor" && remainder.Sign() < 0 {
		epochMilliseconds -= 1
	}
	if mode == "ceil" && remainder.Sign() > 0 {
		epochMilliseconds += 1
	}
	return epochMilliseconds
}

func epochMsToNs(epochMilliseconds float64) *big.Int {
	if !isFiniteNumber(epochMilliseconds) || math.Trunc(epochMilliseconds) != epochMilliseconds {
		throwRange("epoch milliseconds must be an integer")
	}
	return new(big.Int).Mul(bigFromNumber(epochMilliseconds), bigMillion)
}

func toBigInt(arg any) *big.Int {
	prim := unwrap(arg)
	// typeof null is 'object', so the original reads a property of it. No value this package holds
	// has a Symbol.toPrimitive method.
	if isNull(prim) {
		throwType("Cannot read properties of null (reading 'Symbol(Symbol.toPrimitive)')")
	}

	// The AO ToBigInt throws on numbers because it does not allow implicit
	// conversion between number and bigint (unlike the bigint constructor).
	if _, ok := prim.(float64); ok {
		throwType("cannot convert number to bigint")
	}
	if b, ok := prim.(*big.Int); ok {
		return new(big.Int).Set(b)
	}
	// JSBI will properly coerce types into a BigInt the same as the native BigInt
	// constructor will, with the exception of native bigint which is handled
	// above.
	return jsbiBigInt(prim)
}

// jsbiBigInt is JSBI.BigInt on anything but a JSBI value.
func jsbiBigInt(arg any) *big.Int {
	switch v := arg.(type) {
	case float64:
		return bigFromNumber(v)
	case string:
		result := jsbiFromString(v)
		if result == nil {
			throwSyntax("Cannot convert " + v + " to a BigInt")
		}
		return result
	case bool:
		if v {
			return big.NewInt(1)
		}
		return new(big.Int)
	case *big.Int:
		return new(big.Int).Set(v)
	}
	if isObject(arg) {
		return jsbiBigInt(toPrimitive(arg, "default"))
	}
	throwType("Cannot convert " + jsToString(arg) + " to a BigInt")
	return nil
}

// jsbiIsWhitespace is JSBI.__isWhitespace on a UTF-16 code unit. JSBI's table recognizes fewer
// characters than StringToBigInt does.
func jsbiIsWhitespace(c uint16) bool {
	return (c >= 9 && c <= 13) || c == 32 || c == 160 || c == 5760
}

// jsbiFromString is JSBI.__fromString: nil where the string is no BigInt.
func jsbiFromString(str string) *big.Int {
	s := utf16.Encode([]rune(str))
	sign := 0
	cursor := 0
	if cursor == len(s) {
		return new(big.Int)
	}
	current := s[cursor]
	// Skip whitespace.
	for jsbiIsWhitespace(current) {
		cursor++
		if cursor == len(s) {
			return new(big.Int)
		}
		current = s[cursor]
	}

	// Detect radix.
	if current == '+' {
		cursor++
		if cursor == len(s) {
			return nil
		}
		current = s[cursor]
		sign = 1
	} else if current == '-' {
		cursor++
		if cursor == len(s) {
			return nil
		}
		current = s[cursor]
		sign = -1
	}
	radix := 10
	if current == '0' {
		cursor++
		if cursor == len(s) {
			return new(big.Int)
		}
		current = s[cursor]
		switch current {
		case 'X', 'x':
			radix = 16
		case 'O', 'o':
			radix = 8
		case 'B', 'b':
			radix = 2
		}
		if radix != 10 {
			cursor++
			if cursor == len(s) {
				return nil
			}
			current = s[cursor]
		}
	}
	if sign != 0 && radix != 10 {
		return nil
	}
	// Skip leading zeros.
	for current == '0' {
		cursor++
		if cursor == len(s) {
			return new(big.Int)
		}
		current = s[cursor]
	}
	// JSBI refuses strings too long for its bit estimate.
	if float64(len(s)-cursor) > 1073741824/map[int]float64{2: 32, 8: 96, 10: 107, 16: 128}[radix] {
		return nil
	}

	// Parse the digits; a string without any is 0.
	var digits strings.Builder
	for cursor != len(s) {
		var valid bool
		switch radix {
		case 16:
			valid = (current >= '0' && current <= '9') || (current|32 >= 'a' && current|32 <= 'f')
		default:
			valid = current >= '0' && int(current)-'0' < radix
		}
		if !valid {
			break
		}
		digits.WriteByte(byte(current))
		cursor++
		if cursor != len(s) {
			current = s[cursor]
		}
	}

	// Skip trailing whitespace.
	for ; cursor != len(s); cursor++ {
		if !jsbiIsWhitespace(s[cursor]) {
			return nil
		}
	}

	result := new(big.Int)
	if digits.Len() > 0 {
		result.SetString(digits.String(), radix)
	}
	if sign == -1 {
		result.Neg(result)
	}
	return result
}

var (
	systemNsMutex sync.Mutex
	systemNs      = bigFromNumber(jsMod(dateNow(), 1e6))
)

// Note: This method returns values with bogus nanoseconds based on the previous iteration's
// milliseconds. That way there is a guarantee that the full nanoseconds are always going to be
// increasing at least and that the microsecond and nanosecond fields are likely to be non-zero.
func systemUTCEpochNanoSeconds() *big.Int {
	systemNsMutex.Lock()
	defer systemNsMutex.Unlock()
	now := dateNow()
	ms := bigFromNumber(now)
	result := new(big.Int).Add(epochMsToNs(now), systemNs)
	systemNs = new(big.Int).Rem(ms, bigMillion)
	if result.Cmp(nsMax) > 0 {
		return nsMax
	}
	if result.Cmp(nsMin) < 0 {
		return nsMin
	}
	return result
}

// defaultTimeZone is new Intl.DateTimeFormat().resolvedOptions().timeZone, which resolves the TZ
// environment variable and otherwise the host's zone; the host's zone is not read here.
func defaultTimeZone() string {
	if tz := os.Getenv("TZ"); tz != "" {
		if primary, ok := intl.CanonicalTimeZone(tz); ok {
			return primary
		}
	}
	return "UTC"
}

func comparisonResult(value float64) float64 {
	if value < 0 {
		return -1
	}
	if value > 0 {
		return 1
	}
	return value
}

// emptyOptions is the empty bag GetOptionsObject makes when options is undefined.
type emptyOptions struct{}

func getOptionsObject(options any) any {
	value := unwrap(options)
	if isUndefined(value) {
		return emptyOptions{}
	}
	if isObject(value) {
		return value
	}
	typeName := jsTypeof(value)
	if isNull(value) {
		typeName = "null"
	}
	throwType("Options parameter must be an object, not " + typeName)
	return nil
}

// onePropObject is the object CreateOnePropObject makes.
type onePropObject struct {
	key   string
	value any
}

func (o onePropObject) jsGet(key string) any {
	if key == o.key {
		return o.value
	}
	return nil
}

func createOnePropObject(propName string, propValue any) any {
	return onePropObject{propName, propValue}
}

func getOption(options any, property string, allowedValues []string, fallback any) any {
	value := objGet(options, property)
	if !isUndefined(value) {
		str := jsToString(value)
		if !slices.Contains(allowedValues, str) {
			throwRange(property + " must be one of " + strings.Join(allowedValues, ", ") + ", not " + str)
		}
		return str
	}
	if fallback == requiredSentinel {
		throwRange(property + " option is required")
	}
	return fallback
}

// This is a temporary implementation. Ideally we'd rely on Intl.DateTimeFormat
// here, to provide the latest CLDR alias data, when implementations catch up to
// the ECMA-402 change. The aliases below are taken from
// https://github.com/unicode-org/cldr/blob/main/common/bcp47/calendar.xml
func canonicalizeCalendar(idParam string) string {
	id := asciiLowercase(idParam)

	if !slices.Contains(builtinCalendarIDs, id) {
		throwRange("invalid calendar identifier " + id)
	}

	switch id {
	case "ethiopic-amete-alem":
		// May need to be removed in the future.
		// See https://github.com/tc39/ecma402/issues/285
		return "ethioaa"
	// case 'gregorian':
	// (Skip 'gregorian'. It isn't a valid identifier as it's a single
	// subcomponent longer than 8 letters. It can only be used with the old
	// @key=value syntax.)
	case "islamicc":
		return "islamic-civil"
	}
	return id
}

func asciiLowercase(str string) string {
	// The spec defines this operation distinct from String.prototype.lowercase,
	// so we'll follow the spec here. Note that nasty security issues that can
	// happen for some use cases if you're comparing case-modified non-ASCII
	// values. For example, Turkish's "I" character was the source of a security
	// issue involving "file://" URLs. See
	// https://haacked.com/archive/2012/07/05/turkish-i-problem-and-why-you-should-care.aspx/.
	lowercase := []byte(str)
	for ix, code := range lowercase {
		if code >= 0x41 && code <= 0x5a {
			lowercase[ix] = code + 0x20
		}
	}
	return string(lowercase)
}

// This function isn't in the spec, but we put it in the polyfill to avoid
// repeating the same (long) error message in many files.
func valueOfThrows(constructorName string) {
	compareCode := "Temporal." + constructorName + ".compare(obj1, obj2)"
	if constructorName == "PlainMonthDay" {
		compareCode = "Temporal.PlainDate.compare(obj1.toPlainDate(year), obj2.toPlainDate(year))"
	}

	throwType(
		"Do not use built-in arithmetic operators with Temporal objects. " +
			"When comparing, use " + compareCode + ", not obj1 > obj2. " +
			"When coercing to strings, use `${obj}` or String(obj), not '' + obj. " +
			"When coercing to numbers, use properties or methods of the object, not `+obj`. " +
			"When concatenating with strings, use `${str}${obj}` or str.concat(obj), not str + obj. " +
			"In React, coerce to a string before rendering a Temporal object.",
	)
}
