package temporal

import (
	"math"
	"sort"
	"strings"
	"unicode"

	"github.com/js-temporal/temporal-polyfill-go/internal/intl"
)

// calAssert is the original's assert. The assertions are on in the original's build, and some of
// them are reachable, so a failure is the same exception.
func calAssert(condition bool, message string) {
	if !condition {
		calAssertFail(message)
	}
}

func calAssertFail(message string) {
	panic(&Error{Name: "Error", Message: "assertion failure: " + message})
}

// calAssertExistsFail is a failing ES.assertExists.
func calAssertExistsFail() { panic(&Error{Name: "Error", Message: "Expected arg to be set."}) }

func calNum(x float64) *float64 { return &x }

func calStr(s string) *string { return &s }

// calVal is reading a number that may be undefined in arithmetic, where undefined becomes NaN.
func calVal(p *float64) float64 {
	if p == nil {
		return math.NaN()
	}
	return *p
}

// calNeq is the !== operator on two numbers that may be undefined.
func calNeq(a, b *float64) bool {
	if a == nil || b == nil {
		return a != b
	}
	return *a != *b
}

// calDate is a partial or full calendar date: CalendarDateFields, Partial<FullCalendarDate>,
// FullCalendarDate and CalendarYMD of the original. A nil field is undefined. It is `monthExtra`
// that carries additional information used internally to identify lunisolar leap months. Values are
// handled by pointer because the original mutates objects it has cached; the float pointers
// themselves are never written through. rest holds what a date built from fields also carries, for
// JSON.stringify only.
type calDate struct {
	era                  *string
	eraYear, year, month *float64
	monthCode            *string
	day                  *float64
	monthExtra           *string
	rest                 *calendarFields
}

func (d *calDate) clone() *calDate {
	c := *d
	return &c
}

func calYMD(year, month, day float64) *calDate {
	return &calDate{year: calNum(year), month: calNum(month), day: calNum(day)}
}

func calDateFromFields(fields *calendarFields) *calDate {
	return &calDate{
		era:        fields.era,
		eraYear:    fields.eraYear,
		year:       fields.year,
		month:      fields.month,
		monthCode:  fields.monthCode,
		day:        fields.day,
		monthExtra: fields.monthExtra,
		rest:       fields,
	}
}

func calJSONNumber(x float64) string {
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return "null"
	}
	return numberToString(x)
}

func calJSONString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch {
		case r == '"':
			b.WriteString(`\"`)
		case r == '\\':
			b.WriteString(`\\`)
		case r == '\b':
			b.WriteString(`\b`)
		case r == '\f':
			b.WriteString(`\f`)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r == '\t':
			b.WriteString(`\t`)
		case r < 0x20:
			const hex = "0123456789abcdef"
			b.WriteString(`\u00`)
			b.WriteByte(hex[r>>4])
			b.WriteByte(hex[r&15])
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// calJSONISODate is JSON.stringify of an ISO Date Record.
func calJSONISODate(d isoDate) string {
	return `{"year":` + calJSONNumber(d.year) + `,"month":` + calJSONNumber(d.month) + `,"day":` + calJSONNumber(d.day) + `}`
}

// calJSONDate is JSON.stringify of a calendar date. Prepared fields hold their keys in sorted order;
// any other date here came through the base adjustCalendarDate from addCalendar.
func calJSONDate(d *calDate) string {
	var parts []string
	num := func(key string, p *float64) {
		if p != nil {
			parts = append(parts, `"`+key+`":`+calJSONNumber(*p))
		}
	}
	str := func(key string, p *string) {
		if p != nil {
			parts = append(parts, `"`+key+`":`+calJSONString(*p))
		}
	}
	if r := d.rest; r != nil {
		num("day", d.day)
		str("era", d.era)
		num("eraYear", d.eraYear)
		num("hour", r.hour)
		num("microsecond", r.microsecond)
		num("millisecond", r.millisecond)
		num("minute", r.minute)
		num("month", d.month)
		str("monthCode", d.monthCode)
		num("nanosecond", r.nanosecond)
		str("offset", r.offset)
		num("second", r.second)
		str("timeZone", r.timeZone)
		num("year", d.year)
	} else {
		num("year", d.year)
		str("monthCode", d.monthCode)
		num("day", d.day)
		num("month", d.month)
		num("eraYear", d.eraYear)
		str("era", d.era)
	}
	return "{" + strings.Join(parts, ",") + "}"
}

// calCache is the OneObjectCache. The original associates one instance with each ISO Date Record
// object; here an instance lives for one call of a calendarImpl method. Its entries are read back,
// and the keys collide the way the original's do, so that is behaviour and not only speed: a date
// whose year wraps the key finds the entry of another year.
//
// Cache keys are int32
// int32 msb fedcba9876543210fedcba9876543210 lsb
//
//	uyyyyyyyyyyyyyyyyyyyymmmmdddddff
//
// u = unused (1 bit)
// y = year + 280804 (20 bits; max is 564387)
// m = month (4 bits; max is 13)
// d = day (5 bits; max is 31)
// f = flags (indicates type of key, and overflow for calendar-to-ISO type)
//
//	00 = Chinese/Dangi month list
//	01 = ISO-to-calendar
//	10 = calendar-to-ISO, overflow constrain
//	11 = calendar-to-ISO, overflow reject
//
// The flags keep the three kinds of entry apart, so each kind has a map of its own.
type calCache struct {
	monthLists map[int32]*chineseMonthList
	calendar   map[int32]*calDate
	iso        map[int32]isoDate
}

func newCalCache() *calCache {
	return &calCache{
		monthLists: map[int32]*chineseMonthList{},
		calendar:   map[int32]*calDate{},
		iso:        map[int32]isoDate{},
	}
}

// calToInt32 is the ECMAScript ToInt32 applied by the bitwise operators.
func calToInt32(x float64) int32 {
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return 0
	}
	m := math.Mod(math.Trunc(x), 4294967296)
	if m < 0 {
		m += 4294967296
	}
	return int32(uint32(m))
}

func calPrivKey(year, month, day float64, flags int32) int32 {
	// -280804 is the earliest year number in any supported calendar (in this
	// case, Hijri calendars)
	unsignedYear := year + 280804
	return calToInt32(unsignedYear)<<11 | calToInt32(month)<<7 | calToInt32(day)<<2 | flags
}

func generateCalendarToISOKey(d *calDate, overflow string) int32 {
	flags := int32(0b11)
	if overflow == "constrain" {
		flags = 0b10
	}
	return calPrivKey(calVal(d.year), calVal(d.month), calVal(d.day), flags)
}

func generateISOToCalendarKey(d isoDate) int32 { return calPrivKey(d.year, d.month, d.day, 1) }

func generateMonthListKey(year float64) int32 { return calPrivKey(year, 0, 0, 0) }

func toUtcIsoDateString(isoYear, isoMonth, isoDay float64) string {
	yearString := isoYearString(isoYear)
	monthString := isoDateTimePartString(isoMonth)
	dayString := isoDateTimePartString(isoDay)
	return yearString + "-" + monthString + "-" + dayString + "T00:00Z"
}

type calDateDiff struct{ years, months, days float64 }

func simpleDateDiff(one, two *calDate) calDateDiff {
	return calDateDiff{
		years:  *one.year - *two.year,
		months: *one.month - *two.month,
		days:   *one.day - *two.day,
	}
}

func clampISODate(iso isoDate) isoDate {
	if iso.year < -271821 || (iso.year == -271821 && (iso.month < 4 || (iso.month == 4 && iso.day < 19))) {
		return isoDate{year: -271821, month: 4, day: 19}
	}
	if iso.year > 275760 || (iso.year == 275760 && (iso.month > 9 || (iso.month == 9 && iso.day > 13))) {
		return isoDate{year: 275760, month: 9, day: 13}
	}
	return iso
}

// calComparisonResult is ES.ComparisonResult.
func calComparisonResult(value float64) float64 {
	if value < 0 {
		return -1
	}
	if value > 0 {
		return 1
	}
	return value
}

// calHelper holds what the original's helper classes override, so that a call from helperBase
// reaches the most-derived method. Methods not listed are never overridden.
type calHelper interface {
	base() *helperBase
	monthsInYear(calendarDate *calDate, cache *calCache) float64
	maximumMonthLength(calendarDate *calDate) float64
	minimumMonthLength(calendarDate *calDate) float64
	maxLengthOfMonthCodeInAnyYear(monthCode string) float64
	estimateIsoDate(calendarDate *calDate) isoDate
	inLeapYear(calendarDate *calDate, cache *calCache) bool
	// reviseIntlEra reports false when the calendar has no such method.
	reviseIntlEra(calendarDate *calDate, isoDate isoDate) (era *string, eraYear *float64, ok bool)
	checkIcuBugs(isoDate isoDate)
	isoToCalendarDate(isoDate isoDate, cache *calCache) *calDate
	adjustCalendarDate(calendarDate *calDate, cache *calCache, overflow string, fromLegacyDate bool) *calDate
	daysInMonth(calendarDate *calDate, cache *calCache) float64
	daysInPreviousMonth(calendarDate *calDate, cache *calCache) float64
	monthDaySearchStartYear(monthCode string, day float64) float64
}

// helperBase is the implementation helper that's common to all non-ISO calendars.
type helperBase struct {
	self         calHelper
	id           string
	calendarType string // "solar", "lunar" or "lunisolar"
	eras         []*calEra
	anchorEra    *calEra
	// See https://github.com/tc39/proposal-temporal/issues/1784
	erasBeginMidYear bool
}

func (h *helperBase) base() *helperBase { return h }

func (h *helperBase) init(self calHelper, id, calendarType string, eraData []*calInputEra) {
	h.self = self
	h.id = id
	h.calendarType = calendarType
	// dummy era for calendars without eras
	h.anchorEra = &calEra{
		anchorEpoch:  calYMD(0, 1, 1),
		isoEpoch:     isoDate{year: 0, month: 1, day: 1},
		startingYear: 1,
	}
	if eraData != nil {
		h.eras, h.anchorEra = adjustEras(eraData)
	}
}

func (h *helperBase) reviseIntlEra(calendarDate *calDate, isoDate isoDate) (*string, *float64, bool) {
	return nil, nil, false
}

func (h *helperBase) checkIcuBugs(isoDate isoDate) {}

// getCalendarParts is the original's method of that name, with the ISO date given as well so that
// new Date(isoString) can be arithmetic. An isoString with a year of more than six digits is not
// parsed by Date either, and all of those are out of Date's range.
func (h *helperBase) getCalendarParts(isoString string, iso isoDate) []intl.Part {
	zone := "UTC"
	var legacyDate float64

	// PlainDate's minimum date -271821-04-19 is one day beyond legacy Date's
	// minimum -271821-04-20, because of accommodating all Instants in all time
	// zones. If we have -271821-04-19, instead format -271821-04-20 in a time
	// zone that pushes the result into the previous day.
	if isoString == "-271821-04-19T00:00Z" {
		zone = "Etc/GMT+1"
		legacyDate = -8640000000000000
	} else {
		legacyDate = getUTCEpochMilliseconds(iso, timeRecord{})
		if !(math.Abs(legacyDate) <= 8640000000000000) {
			throwRange("Invalid ISO date: " + isoString)
		}
	}

	parts, err := intl.CalendarParts(h.id, legacyDate, zone)
	if err != nil {
		if e, ok := err.(*intl.Error); ok && e.Name == "RangeError" {
			throwRange("Invalid ISO date: " + isoString)
		}
		intlRaise(err)
	}
	return parts
}

// calIntlMonth is the regular expression /^(?:Mo)?([0-9]*)(.*?)$/ on a month part. Newer ICU data
// has some formats with "Mo11" / "Mo9bis" for Chinese and Dangi months.
func calIntlMonth(value string) (digits, rest string, matched bool) {
	if strings.ContainsAny(value, "\n\r  ") {
		return "", "", false
	}
	s := strings.TrimPrefix(value, "Mo")
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	return s[:i], s[i:], true
}

// calNormalizeEra is the era normalisation of isoToCalendarDate: drop what follows " (", NFD, remove
// everything but -, digits, space and letters, spaces to dashes, lower case. The only non-ASCII era
// name that reaches it in en-US is the Indian "Śaka", since the Japanese calendar does not ask Intl;
// NFD splits off the combining accent that the removal then takes away.
func calNormalizeEra(value string) string {
	s, _, _ := strings.Cut(value, " (")
	var b strings.Builder
	for _, r := range s {
		switch r {
		case 'Ś':
			r = 'S'
		case 'ś':
			r = 's'
		}
		switch {
		case r == ' ':
			b.WriteByte('-')
		case r == '-' || (r >= '0' && r <= '9') || unicode.IsLetter(r):
			b.WriteRune(r)
		}
	}
	return strings.ToLower(b.String())
}

func (h *helperBase) isoToCalendarDate(isoDate isoDate, cache *calCache) *calDate {
	isoYear, isoMonth, isoDay := isoDate.year, isoDate.month, isoDate.day
	key := generateISOToCalendarKey(isoDate)
	if cached := cache.calendar[key]; cached != nil {
		return cached
	}

	isoString := toUtcIsoDateString(isoYear, isoMonth, isoDay)
	parts := h.getCalendarParts(isoString, isoDate)
	hasEra := calendarSupportsEra(h.id)
	result := &calDate{}
	for i := range parts {
		typ, value := parts[i].Type, parts[i].Value
		if typ == "year" || typ == "relatedYear" {
			if hasEra {
				result.eraYear = calNum(stringToNumber(value))
			} else {
				result.year = calNum(stringToNumber(value))
			}
		}
		if typ == "month" {
			// Newer ICU data has some formats with "Mo11" / "Mo9bis" for Chinese
			// and Dangi months
			digits, rest, matched := calIntlMonth(value)
			if !matched || (digits == "" && rest == "") {
				throwRange("Unexpected month: " + value)
			}
			// If the month has no numeric part (should only see this for the Hebrew
			// calendar with newer FF / Chromium versions; see
			// https://bugzilla.mozilla.org/show_bug.cgi?id=1751833) then set a
			// placeholder month index of `1` and rely on the derived class to
			// calculate the correct month index from the month name stored in
			// `monthExtra`.
			month := 1.0
			if digits != "" {
				month = stringToNumber(digits)
			}
			result.month = calNum(month)
			if month < 1 {
				throwRange("Invalid month " + value + " from " + isoString + "[u-ca-" + h.id + "]" +
					" (probably due to https://bugs.chromium.org/p/v8/issues/detail?id=10527)")
			}
			if month > 13 {
				throwRange("Invalid month " + value + " from " + isoString + "[u-ca-" + h.id + "]" +
					" (probably due to https://bugs.chromium.org/p/v8/issues/detail?id=10529)")
			}

			// The ICU formats for the Hebrew calendar no longer support a numeric
			// month format. So we'll rely on the derived class to interpret it.
			// `monthExtra` is also used on the Chinese calendar to handle a suffix
			// "bis" indicating a leap month.
			if rest != "" {
				result.monthExtra = calStr(rest)
			}
		}
		if typ == "day" {
			result.day = calNum(stringToNumber(value))
		}
		if hasEra && typ == "era" && value != "" {
			// The convention for Temporal era values is lowercase, so following
			// that convention in this prototype. Punctuation is removed, accented
			// letters are normalized, and spaces are replaced with dashes.
			// E.g.: "ERA0" => "era0", "Before R.O.C." => "before-roc", "En’ō" => "eno"
			// The normalization deals with era names that contain non-ASCII
			// characters like Japanese eras. Also ignore extra content in
			// parentheses like JPN era date ranges.
			result.era = calStr(calNormalizeEra(value))
		}
	}
	if hasEra && (result.era == nil || *result.era == "") {
		// Work around ICU bug that neglects to provide an era code for negative
		// eraYear in the coptic calendar
		if h.id != "coptic" {
			// If missing from any other calendar, it's an as-yet-unknown bug
			throwRange("Intl.DateTimeFormat.formatToParts lacks era in " + h.id + " calendar.")
		}
		// eraYear is also reversed, but using the legacy era code will set it
		// right
		result.era = calStr("era0")
	}
	if hasEra && result.eraYear == nil {
		// Node 12 has outdated ICU data that lacks the `relatedYear` field in the
		// output of Intl.DateTimeFormat.formatToParts.
		throwRange("Intl.DateTimeFormat.formatToParts lacks relatedYear in " + h.id + " calendar. Try Node 14+ or modern browsers.")
	}
	// Translate old ICU era codes "ERA0" etc. into canonical era names.
	if hasEra {
		for _, e := range h.eras {
			if *result.era == e.genericName {
				result.era = calStr(e.code)
				break
			}
		}
	}
	// Translate eras that may be handled differently by Temporal vs. by Intl
	// (e.g. Japanese pre-Meiji eras). See https://github.com/tc39/proposal-temporal/issues/526.
	if era, eraYear, ok := h.self.reviseIntlEra(result, isoDate); ok {
		result.era = era
		result.eraYear = eraYear
	}
	h.self.checkIcuBugs(isoDate)

	calendarDate := h.self.adjustCalendarDate(result, cache, "constrain", true)
	if calendarDate.year == nil {
		throwRange("Missing year converting " + calJSONISODate(isoDate))
	}
	if calendarDate.month == nil {
		throwRange("Missing month converting " + calJSONISODate(isoDate))
	}
	if calendarDate.day == nil {
		throwRange("Missing day converting " + calJSONISODate(isoDate))
	}
	cache.calendar[key] = calendarDate
	// Also cache the reverse mapping
	for _, overflow := range [...]string{"constrain", "reject"} {
		keyReverse := generateCalendarToISOKey(calendarDate, overflow)
		cache.iso[keyReverse] = isoDate
	}
	return calendarDate
}

func (h *helperBase) validateCalendarDate(calendarDate *calDate) {
	month, year, day, eraYear, monthCode, monthExtra := calendarDate.month, calendarDate.year, calendarDate.day,
		calendarDate.eraYear, calendarDate.monthCode, calendarDate.monthExtra
	// When there's a suffix (e.g. "5bis" for a leap month in Chinese calendar)
	// the derived class must deal with it.
	if monthExtra != nil {
		throwRange("Unexpected `monthExtra` value")
	}
	if year == nil && eraYear == nil {
		throwType("year or eraYear is required")
	}
	if month == nil && monthCode == nil {
		throwType("month or monthCode is required")
	}
	if day == nil {
		throwRange("Missing day")
	}
	if monthCode != nil {
		monthNumber, _ := parseMonthCode(*monthCode)
		if monthNumber < 1 || monthNumber > 13 {
			throwRange("Invalid monthCode: " + *monthCode)
		}
	}
	if calendarSupportsEra(h.id) {
		if (calendarDate.era == nil) != (calendarDate.eraYear == nil) {
			throwType("properties era and eraYear must be provided together")
		}
	}
}

// eraFromYear is a private helper function.
func (h *helperBase) eraFromYear(calendarDate *calDate) (eraYear float64, era string) {
	year := *calendarDate.year
	ix := -1
	for i, eParam := range h.eras {
		e := eParam
		if i == len(h.eras)-1 {
			if e.skip {
				// This last era is only present for legacy ICU data. Treat the
				// previous era as the last era.
				e = h.eras[i-1]
			}
			if e.reverseOf != nil {
				// This is a reverse-sign era (like BCE) which must be the oldest
				// era. Count years backwards.
				// The year must be negative, because the reverse-sign era is always last,
				// and if the year was positive, the (comparison >= 0) would have been true
				// and this closure would already have returned true
				calAssert(year <= 0, "year must not be positive in eraFromYear")
				eraYear = *e.anchorEpoch.year - year
				ix = i
				break
			}
			// last era always gets all "leftover" (older than epoch) years,
			// so no need for a comparison like below.
			eraYear = year - *e.anchorEpoch.year + e.startingYear
			ix = i
			break
		}
		comparison := h.compareCalendarDates(calendarDate, e.anchorEpoch)
		if comparison >= 0 {
			eraYear = year - *e.anchorEpoch.year + e.startingYear
			ix = i
			break
		}
	}
	// Last era in the list should always match if no other era does
	calAssert(ix >= 0, "Year "+numberToString(year)+" was not matched by any era")
	matchingEra := h.eras[ix]
	if matchingEra.skip {
		matchingEra = h.eras[ix-1]
	}
	return eraYear, matchingEra.code
}

// completeEraYear fills in missing parts of the (year, era, eraYear) tuple.
func (h *helperBase) completeEraYear(calendarDate *calDate) *calDate {
	year, eraYear, era := calendarDate.year, calendarDate.eraYear, calendarDate.era
	if year != nil {
		matchEraYear, matchEra := h.eraFromYear(calendarDate)
		eraYear, era = calNum(matchEraYear), calStr(matchEra)
		if calendarDate.era != nil {
			canonical := canonicalizeEraInCalendar(h.id, *calendarDate.era)
			if canonical == nil || *canonical != *era {
				throwRange("Input era " + *calendarDate.era + " doesn't match calculated value " + *era)
			}
		}
		if calendarDate.eraYear != nil && *calendarDate.eraYear != *eraYear {
			throwRange("Input eraYear " + numberToString(*calendarDate.eraYear) + " doesn't match calculated value " + numberToString(*eraYear))
		}
	} else if eraYear != nil {
		if era == nil {
			calAssertExistsFail()
		}
		canonicalName := canonicalizeEraInCalendar(h.id, *era)
		var matchingEra *calEra
		if canonicalName != nil {
			for _, e := range h.eras {
				if e.code == *canonicalName {
					matchingEra = e
					break
				}
			}
		}
		if matchingEra == nil {
			throwRange("Era " + *era + " (ISO year " + numberToString(*eraYear) + ") was not matched by any era")
		}
		if matchingEra.reverseOf != nil {
			year = calNum(*matchingEra.anchorEpoch.year - *eraYear)
		} else {
			year = calNum(*eraYear + *matchingEra.anchorEpoch.year - matchingEra.startingYear)
		}
		// We'll accept dates where the month/day is earlier than the start of
		// the era or after its end as long as it's in the same year. If that
		// happens, we'll adjust the era/eraYear pair to be the correct era for
		// the `year`.
		adjustedCalendarDate := &calDate{year: year, month: calendarDate.month, day: calendarDate.day}
		matchEraYear, matchEra := h.eraFromYear(adjustedCalendarDate)
		eraYear, era = calNum(matchEraYear), calStr(matchEra)
	}
	// validateCalendarDate already ensured that either year or era+eraYear are
	// present
	if year == nil || eraYear == nil || era == nil {
		calAssertExistsFail()
	}
	result := calendarDate.clone()
	result.year, result.eraYear, result.era = year, eraYear, era
	return result
}

// adjustCalendarDate allows derived calendars to add additional fields and/or to make adjustments
// e.g. to set the era based on the date or to revise the month number in lunisolar calendars per
// https://github.com/tc39/proposal-temporal/issues/1203.
//
// The base implementation fills in missing values by assuming the simplest possible calendar:
// - non-lunisolar calendar (no leap months)
func (h *helperBase) adjustCalendarDate(calendarDateParam *calDate, cache *calCache, overflow string, fromLegacyDate bool) *calDate {
	if h.calendarType == "lunisolar" {
		throwRange("Override required for lunisolar calendars")
	}
	calendarDate := calendarDateParam
	h.validateCalendarDate(calendarDate)
	month, monthCode := resolveNonLunisolarMonth(calendarDate.month, calendarDate.monthCode, h.id, overflow)
	fullCalendarDate := calendarDate.clone()
	fullCalendarDate.month, fullCalendarDate.monthCode = calNum(month), calStr(monthCode)
	if calendarSupportsEra(h.id) {
		fullCalendarDate = h.completeEraYear(fullCalendarDate)
	}
	return fullCalendarDate
}

func (h *helperBase) regulateMonthDayNaive(calendarDate *calDate, overflow string, cache *calCache) *calDate {
	largestMonth := h.self.monthsInYear(calendarDate, cache)
	month, day := *calendarDate.month, *calendarDate.day
	if overflow == "reject" {
		rejectToRange(month, 1, largestMonth)
		rejectToRange(day, 1, h.self.maximumMonthLength(calendarDate))
	} else {
		month = constrainToRange(month, 1, largestMonth)
		withMonth := calendarDate.clone()
		withMonth.month = calNum(month)
		day = constrainToRange(day, 1, h.self.maximumMonthLength(withMonth))
	}
	result := calendarDate.clone()
	result.month, result.day = calNum(month), calNum(day)
	return result
}

func (h *helperBase) calendarToIsoDate(dateParam *calDate, overflow string, cache *calCache) isoDate {
	originalDate := dateParam
	// First, normalize the calendar date to ensure that (year, month, day)
	// are all present, converting monthCode and eraYear if needed.
	date := h.self.adjustCalendarDate(dateParam, cache, overflow, false)

	// Fix obviously out-of-bounds values. Values that are valid generally, but
	// not in this particular year, may not be caught here for some calendars.
	// If so, these will be handled lower below.
	date = h.regulateMonthDayNaive(date, overflow, cache)

	year, month, day := *date.year, *date.month, *date.day
	key := generateCalendarToISOKey(date, overflow)
	if cached, ok := cache.iso[key]; ok {
		return cached
	}
	// If YMD are present in the input but the input has been constrained
	// already, then cache both the original value and the constrained value.
	var keyOriginal int32
	hasKeyOriginal := false
	if originalDate.year != nil &&
		originalDate.month != nil &&
		originalDate.day != nil &&
		(*originalDate.year != *date.year || *originalDate.month != *date.month || *originalDate.day != *date.day) {
		keyOriginal = generateCalendarToISOKey(originalDate, overflow)
		hasKeyOriginal = true
		if cached, ok := cache.iso[keyOriginal]; ok {
			return cached
		}
	}

	// First, try to roughly guess the result
	isoEstimate := clampISODate(h.self.estimateIsoDate(calYMD(year, month, day)))
	calculateSameMonthResult := func(diffDays float64) isoDate {
		// If the estimate is in the same year & month as the target, then we can
		// calculate the result exactly and short-circuit any additional logic.
		// This optimization assumes that months are continuous. It would break if
		// a calendar skipped days, like the Julian->Gregorian switchover. But
		// current ICU calendars only skip days (japanese/roc/buddhist) because of
		// a bug (https://bugs.chromium.org/p/chromium/issues/detail?id=1173158)
		// that's currently worked around by a custom calendarToIsoDate
		// implementation in those calendars. So this optimization should be safe
		// for all ICU calendars.
		testIsoEstimate := addDaysToISODate(isoEstimate, diffDays)
		if *date.day > h.self.minimumMonthLength(date) {
			// There's a chance that the calendar date is out of range. Throw or
			// constrain if so.
			testCalendarDate := h.self.isoToCalendarDate(testIsoEstimate, cache)
			for *testCalendarDate.month != month || *testCalendarDate.year != year {
				if overflow == "reject" {
					throwRange("day " + numberToString(day) + " does not exist in month " + numberToString(month) + " of year " + numberToString(year))
				}
				// Back up a day at a time until we're not hanging over the month end
				testIsoEstimate = addDaysToISODate(testIsoEstimate, -1)
				testCalendarDate = h.self.isoToCalendarDate(testIsoEstimate, cache)
			}
		}
		return testIsoEstimate
	}
	sign := 0.0
	roundtripEstimate := h.self.isoToCalendarDate(isoEstimate, cache)
	diff := simpleDateDiff(date, roundtripEstimate)
	if diff.years != 0 || diff.months != 0 || diff.days != 0 {
		diffTotalDaysEstimate := float64(diff.years*365) + float64(diff.months*30) + diff.days
		isoEstimate = clampISODate(addDaysToISODate(isoEstimate, diffTotalDaysEstimate))
		roundtripEstimate = h.self.isoToCalendarDate(isoEstimate, cache)
		diff = simpleDateDiff(date, roundtripEstimate)
		if diff.years == 0 && diff.months == 0 {
			isoEstimate = calculateSameMonthResult(diff.days)
		} else {
			sign = h.compareCalendarDates(date, roundtripEstimate)
		}
	}
	// If the initial guess is not in the same month, then bisect the
	// distance to the target, starting with 8 days per step.
	increment := 8.0
	for sign != 0 && !math.IsNaN(sign) {
		isoEstimate = addDaysToISODate(isoEstimate, float64(sign*increment))
		oldRoundtripEstimate := roundtripEstimate
		roundtripEstimate = h.self.isoToCalendarDate(isoEstimate, cache)
		oldSign := sign
		sign = h.compareCalendarDates(date, roundtripEstimate)
		if sign != 0 && !math.IsNaN(sign) {
			diff = simpleDateDiff(date, roundtripEstimate)
			if diff.years == 0 && diff.months == 0 {
				isoEstimate = calculateSameMonthResult(diff.days)
				// Signal the loop condition that there's a match.
				sign = 0
			} else if oldSign != 0 && !math.IsNaN(oldSign) && sign != oldSign {
				if increment > 1 {
					// If the estimate overshot the target, try again with a smaller increment
					// in the reverse direction.
					increment /= 2
				} else {
					// Increment is 1, and neither the previous estimate nor the new
					// estimate is correct. The only way that can happen is if the
					// original date was an invalid value that will be constrained or
					// rejected here.
					if overflow == "reject" {
						throwRange("Can't find ISO date from calendar date: " + calJSONDate(originalDate))
					} else {
						// To constrain, pick the earliest value
						order := h.compareCalendarDates(roundtripEstimate, oldRoundtripEstimate)
						// If current value is larger, then back up to the previous value.
						if order > 0 {
							isoEstimate = addDaysToISODate(isoEstimate, -1)
						}
						sign = 0
					}
				}
			}
		}
	}
	cache.iso[key] = isoEstimate
	if hasKeyOriginal {
		cache.iso[keyOriginal] = isoEstimate
	}
	if date.year == nil ||
		date.month == nil ||
		date.day == nil ||
		date.monthCode == nil ||
		(calendarSupportsEra(h.id) && (date.era == nil || date.eraYear == nil)) {
		throwRange("Unexpected missing property")
	}
	return isoEstimate
}

func (h *helperBase) compareCalendarDates(date1, date2 *calDate) float64 {
	if calNeq(date1.year, date2.year) {
		return calComparisonResult(calVal(date1.year) - calVal(date2.year))
	}
	if calNeq(date1.month, date2.month) {
		return calComparisonResult(calVal(date1.month) - calVal(date2.month))
	}
	if calNeq(date1.day, date2.day) {
		return calComparisonResult(calVal(date1.day) - calVal(date2.day))
	}
	return 0
}

// regulateDate ensures that a calendar date actually exists. If not, return the closest earlier
// date.
func (h *helperBase) regulateDate(calendarDate *calDate, overflow string, cache *calCache) *calDate {
	isoDate := h.calendarToIsoDate(calendarDate, overflow, cache)
	return h.self.isoToCalendarDate(isoDate, cache)
}

func (h *helperBase) addDaysCalendar(calendarDate *calDate, days float64, cache *calCache) *calDate {
	isoDate := h.calendarToIsoDate(calendarDate, "constrain", cache)
	addedIso := addDaysToISODate(isoDate, days)
	addedCalendar := h.self.isoToCalendarDate(addedIso, cache)
	return addedCalendar
}

func (h *helperBase) addMonthsCalendar(calendarDateParam *calDate, months float64, overflow string, cache *calCache) *calDate {
	calendarDate := calendarDateParam
	day := *calendarDate.day
	absMonths := math.Abs(months)
	for i := 0.0; i < absMonths; i++ {
		month := *calendarDate.month
		oldCalendarDate := calendarDate
		var days float64
		if months < 0 {
			days = -math.Max(day, h.self.daysInPreviousMonth(calendarDate, cache))
		} else {
			days = h.self.daysInMonth(calendarDate, cache)
		}
		isoDate := h.calendarToIsoDate(calendarDate, "constrain", cache)
		addedIso := addDaysToISODate(isoDate, days)
		calendarDate = h.self.isoToCalendarDate(addedIso, cache)

		// Normally, we can advance one month by adding the number of days in the
		// current month. However, if we're at the end of the current month and
		// the next month has fewer days, then we rolled over to the after-next
		// month. Below we detect this condition and back up until we're back in
		// the desired month.
		if months > 0 {
			monthsInOldYear := h.self.monthsInYear(oldCalendarDate, cache)
			for *calendarDate.month-1 != jsMod(month, monthsInOldYear) {
				addedIso = addDaysToISODate(addedIso, -1)
				calendarDate = h.self.isoToCalendarDate(addedIso, cache)
			}
		}

		if *calendarDate.day != day {
			// try to retain the original day-of-month, if possible
			withDay := calendarDate.clone()
			withDay.day = calNum(day)
			calendarDate = h.regulateDate(withDay, "constrain", cache)
		}
	}
	if overflow == "reject" && *calendarDate.day != day {
		throwRange("Day " + numberToString(day) + " does not exist in resulting calendar month")
	}
	return h.regulateDate(calendarDate, overflow, cache)
}

// calCycleOf is the cycle information of a calendar: nil for a calendar in monthCodeInfo without one.
func calCycleOf(id string) *calCycleInfo {
	if monthInfo := monthCodeInfo[id]; monthInfo != nil {
		return monthInfo.cycleInfo
	}
	return &calCycleInfo{years: 1, months: 12}
}

func (h *helperBase) addCalendar(calendarDate *calDate, dur dateDuration, overflow string, cache *calCache) *calDate {
	years := dur.years
	months := dur.months
	weeks, days := dur.weeks, dur.days
	year, day, monthCode := *calendarDate.year, calendarDate.day, calendarDate.monthCode
	cycleInfo := calCycleOf(h.id)
	if cycleInfo != nil && math.Abs(months) > cycleInfo.months {
		cycleCount := jsTrunc(months / cycleInfo.months)
		years += float64(cycleCount * cycleInfo.years)
		months = jsMod(months, cycleInfo.months)
	}
	addedYears := h.self.adjustCalendarDate(&calDate{year: calNum(year + years), monthCode: monthCode, day: day}, cache, overflow, false)
	addedMonths := h.addMonthsCalendar(addedYears, months, overflow, cache)
	initialDays := days + float64(weeks*7)
	addedDays := h.addDaysCalendar(addedMonths, initialDays, cache)
	return addedDays
}

func (h *helperBase) untilCalendar(calendarOne, calendarTwo *calDate, largestUnit string, cache *calCache) dateDuration {
	months := 0.0
	years := 0.0

	// Sign is -1 if calendarTwo < calendarOne, 1 if calendarTwo > calendarOne
	sign := h.compareCalendarDates(calendarTwo, calendarOne)
	// If dates were equal, this would have been checked in CalendarDateUntil
	calAssert(sign != 0, "equal dates should have been checked earlier")
	// Take the difference between the years of the two dates
	diffYears := *calendarTwo.year - *calendarOne.year
	// Take the difference between the days of the two dates
	diffDays := *calendarTwo.day - *calendarOne.day
	// Get list of additional months, and possibly cycle info
	cycleInfo := calCycleOf(h.id)
	// Compute years difference if necessary
	if diffYears != 0 && !math.IsNaN(diffYears) && (largestUnit == "year" || cycleInfo != nil) {
		// diffInYearSign is the result of comparing the month-day of calendarOne
		// and the month-day of calendarTwo, in the forwards direction
		diffInYearSign := 0.0
		// If calendarTwo's month is greater than calendarOne's month,
		// then the years difference should be positive (or negative if sign < 0).
		if *calendarTwo.monthCode > *calendarOne.monthCode {
			diffInYearSign = 1
		}
		// If calendarTwo's month is less than calendarOne's month,
		// then the years difference should be negative (or positive if sign < 0).
		if *calendarTwo.monthCode < *calendarOne.monthCode {
			diffInYearSign = -1
		}
		// If the two months are equal, the sign of the years difference should be
		// the sign of the days difference.
		if diffInYearSign == 0 {
			diffInYearSign = jsSign(diffDays)
		}
		// isCalendarOneFurtherInYear is true iff the ordering of the years
		// doesn't match the ordering of the month/days.
		isCalendarOneFurtherInYear := float64(diffInYearSign*sign) < 0
		// Add either 1 or -1 to years if isCalendarOneFurtherInYear is true.
		// If monthday-two is later in the year than monthday-one, need
		// to correct diffYears because it's gone one too far.
		if isCalendarOneFurtherInYear {
			years = diffYears - sign
		} else {
			years = diffYears
		}
	}
	// Try to skip ahead as many months as possible for this calendar
	// without adding month by month in a loop
	if largestUnit == "month" {
		if cycleInfo != nil && math.Abs(years) >= cycleInfo.years {
			cycleCount := jsTrunc(years / cycleInfo.years)
			months = float64(cycleCount * cycleInfo.months)
		}
		years = 0
	}

	// intermediate should be a date between calendarOne and calendarTwo,
	// that is within a year of calendarTwo.
	intermediate := calendarOne
	if years != 0 || months != 0 {
		intermediate = h.addCalendar(calendarOne, dateDuration{years: years, months: months}, "constrain", cache)
	}

	// At this point, intermediate could fail to be in between calendarOne and calendarTwo
	// due to leap years.
	// In that case, add or subtract an extra year from years,
	// so that the months can be totaled up correctly.
	target := isoDate{year: *calendarTwo.year, month: *calendarTwo.month, day: *calendarTwo.day}
	if compareSurpasses(sign, *intermediate.year, *intermediate.month, *calendarOne.day, target) {
		years -= sign
	}

	// Now we have less than one cycle remaining. Add one month at a time
	// until we go over the target, then back up one month and calculate
	// remaining days.
	var current *calDate
	// Need to re-add years and months because years might have changed
	next := calendarOne
	if years != 0 || months != 0 {
		next = h.addCalendar(calendarOne, dateDuration{years: years, months: months}, "constrain", cache)
	}
	next.day = calendarOne.day
	for {
		months += sign
		current = next
		next = h.addMonthsCalendar(current, sign, "constrain", cache)
		// In case the day was constrained down, un-constrain it (even if that's
		// not a real date)
		next.day = calendarOne.day
		if compareSurpasses(sign, *next.year, *next.month, *next.day, target) {
			break
		}
	}
	months -= sign // correct for loop above which overshoots by 1
	days := h.calendarDaysUntil(current, calendarTwo, cache)

	// This may return a duration like <P12M11D> that appears to have unbalanced months.
	// But that's fine, because subtracting <P12M11D> from a date may have different
	// results than subtracting <P1Y11D> from the same date, in the presence of leap months.
	return dateDuration{years: years, months: months, weeks: 0, days: days}
}

func (h *helperBase) daysInMonth(calendarDate *calDate, cache *calCache) float64 {
	// Add enough days to roll over to the next month. One we're in the next
	// month, we can calculate the length of the current month. NOTE: This
	// algorithm assumes that months are continuous. It would break if a
	// calendar skipped days, like the Julian->Gregorian switchover. But current
	// ICU calendars only skip days (japanese/roc/buddhist) because of a bug
	// (https://bugs.chromium.org/p/chromium/issues/detail?id=1173158) that's
	// currently worked around by a custom calendarToIsoDate implementation in
	// those calendars. So this code should be safe for all ICU calendars.
	day := *calendarDate.day
	max := h.self.maximumMonthLength(calendarDate)
	min := h.self.minimumMonthLength(calendarDate)
	// easiest case: we already know the month length if min and max are the same.
	if min == max {
		return min
	}

	// Add enough days to get into the next month, without skipping it
	increment := min
	if day <= max-min {
		increment = max
	}
	isoDate := h.calendarToIsoDate(calendarDate, "constrain", cache)
	addedIsoDate := addDaysToISODate(isoDate, increment)
	addedCalendarDate := h.self.isoToCalendarDate(addedIsoDate, cache)

	// Now back up to the last day of the original month
	endOfMonthIso := addDaysToISODate(addedIsoDate, -*addedCalendarDate.day)
	endOfMonthCalendar := h.self.isoToCalendarDate(endOfMonthIso, cache)
	return *endOfMonthCalendar.day
}

func (h *helperBase) daysInPreviousMonth(calendarDate *calDate, cache *calCache) float64 {
	day, month, year := *calendarDate.day, *calendarDate.month, *calendarDate.year

	// Check to see if we already know the month length, and return it if so
	previousMonthYear := year
	if !(month > 1) {
		previousMonthYear = year - 1
	}
	previousMonthDate := calYMD(previousMonthYear, month, 1)
	previousMonth := month - 1
	if !(month > 1) {
		previousMonth = h.self.monthsInYear(previousMonthDate, cache)
	}
	previousMonthDate = calYMD(previousMonthYear, previousMonth, 1)
	min := h.self.minimumMonthLength(previousMonthDate)
	max := h.self.maximumMonthLength(previousMonthDate)
	if min == max {
		return max
	}

	isoDate := h.calendarToIsoDate(calendarDate, "constrain", cache)
	lastDayOfPreviousMonthIso := addDaysToISODate(isoDate, -day)
	lastDayOfPreviousMonthCalendar := h.self.isoToCalendarDate(lastDayOfPreviousMonthIso, cache)
	return *lastDayOfPreviousMonthCalendar.day
}

func (h *helperBase) startOfCalendarYear(calendarDate *calDate) *calDate {
	return &calDate{year: calendarDate.year, month: calNum(1), monthCode: calStr("M01"), day: calNum(1)}
}

func (h *helperBase) calendarDaysUntil(calendarOne, calendarTwo *calDate, cache *calCache) float64 {
	oneIso := h.calendarToIsoDate(calendarOne, "constrain", cache)
	twoIso := h.calendarToIsoDate(calendarTwo, "constrain", cache)
	return isoDateToEpochDays(twoIso.year, twoIso.month-1, twoIso.day) -
		isoDateToEpochDays(oneIso.year, oneIso.month-1, oneIso.day)
}

// monthDaySearchStartYear is to be overridden to shortcut the search space if certain month codes
// only occur long in the past.
func (h *helperBase) monthDaySearchStartYear(monthCode string, day float64) float64 { return 1972 }

func (h *helperBase) monthDayFromFields(fields *calDate, overflow string, cache *calCache) isoDate {
	eraYear, year, monthCodeP, dayP := fields.eraYear, fields.year, fields.monthCode, fields.day
	hasEra := calendarSupportsEra(h.id)
	if monthCodeP == nil || year != nil || (hasEra && eraYear != nil) {
		// Apply overflow behaviour to year/month/day, to get correct monthCode/day
		r := h.self.isoToCalendarDate(h.calendarToIsoDate(fields, overflow, cache), cache)
		monthCodeP, dayP = r.monthCode, r.day
	}
	monthCode, day := *monthCodeP, *dayP

	// Shape of property bag is correct, check valid input and apply overflow
	if !isValidMonthCodeForCalendar(h.id, monthCode) {
		throwRange("Invalid monthCode: " + monthCode + " does not exist in calendar " + h.id)
	}
	maxDayForMonthCode := h.self.maxLengthOfMonthCodeInAnyYear(monthCode)
	if day > maxDayForMonthCode {
		if overflow == "reject" {
			throwRange("No " + h.id + " year with monthCode " + monthCode + " and day " + numberToString(day))
		}
		day = maxDayForMonthCode
	}

	// Look backwards starting from one of the calendar years spanning ISO year
	// 1972, up to 20 calendar years prior, to find a year that has this month
	// and day. Normal months and days will match immediately, but for leap days
	// and leap months we may have to look for a while. For searches longer than
	// 20 years, override the start date in monthDaySearchStartYear.
	startDateIso := isoDate{
		year:  h.self.monthDaySearchStartYear(monthCode, day),
		month: 12,
		day:   31,
	}
	calendarOfStartDateIso := h.self.isoToCalendarDate(startDateIso, cache)
	// Note: relies on lexicographical ordering of monthCodes
	calendarYear := *calendarOfStartDateIso.year - 1
	if *calendarOfStartDateIso.monthCode > monthCode ||
		(*calendarOfStartDateIso.monthCode == monthCode && *calendarOfStartDateIso.day >= day) {
		calendarYear = *calendarOfStartDateIso.year
	}
	for i := 0.0; i < 20; i++ {
		testCalendarDate := h.self.adjustCalendarDate(
			&calDate{day: calNum(day), monthCode: calStr(monthCode), year: calNum(calendarYear - i)},
			cache, "constrain", false,
		)
		isoDate := h.calendarToIsoDate(testCalendarDate, "constrain", cache)
		roundTripCalendarDate := h.self.isoToCalendarDate(isoDate, cache)
		if *roundTripCalendarDate.monthCode == monthCode && *roundTripCalendarDate.day == day {
			return isoDate
		}
	}
	panic(&Error{Name: "Error", Message: "assertion failure: code should not be reached because no recent " + h.id +
		" year with " + monthCode + "-" + numberToString(day) + ", adjust monthDaySearchStartYear"})
}

// calInputEra is the era metadata defined for each calendar. Instead of optional properties, this
// should really have rules encoded in the type, e.g. isoEpoch is required unless reverseOf is
// present.
type calInputEra struct {
	// code is the era code, used to populate the 'era' field of Temporal instances.
	// See https://tc39.es/proposal-intl-era-monthcode/#table-eras
	code string
	// anchorEpoch is the signed calendar year where this era begins. Will be 1 (or 0 for zero-based
	// eras) for the anchor era assuming that `year` numbering starts at the beginning of the anchor
	// era, which is true for all ICU calendars except Japanese. If an era starts mid-year then a
	// calendar month and day are included. Otherwise `{ month: 1, day: 1 }` is assumed.
	anchorEpoch *calDate
	// isoEpoch is the ISO date of the first day of this era.
	isoEpoch *isoDate
	// reverseOf, if present, means that this era counts years backwards like BC and this property
	// points to the forward era. This must be the last (oldest) era in the array.
	reverseOf string
	// startingYear is the number of the era's starting year (for example, 0 if there's a Year 0). If
	// omitted, the era's years are 1-based.
	startingYear *float64
	// isAnchor overrides if this era is the anchor. Not normally used because anchor eras are inferred.
	isAnchor bool
	// skip overrides if this era should never be user-facing (for example, removed.)
	skip bool
}

// calEra is the transformation of calInputEra with all fields filled in by adjustEras.
type calEra struct {
	code string
	// genericName is the alternate name of the era used in old versions of ICU data. The format is
	// `era{n}` where n is the zero-based index of the era with the oldest era being 0.
	genericName  string
	anchorEpoch  *calDate
	isoEpoch     isoDate
	reverseOf    *calEra
	startingYear float64
	isAnchor     bool
	skip         bool
}

// adjustEras adds additional metadata that makes it easier to work with eras. The result is an array
// of eras with the shape defined above.
func adjustEras(erasParam []*calInputEra) ([]*calEra, *calEra) {
	// It's an internal error if the eras data are malformed
	calAssert(len(erasParam) > 0, "Invalid era data: eras are required")
	calAssert(!(len(erasParam) == 1 && erasParam[0].reverseOf != ""), "Invalid era data: anchor era cannot count years backwards")
	calAssert(!(len(erasParam) == 1 && erasParam[0].code == ""), "Invalid era data: at least one named era is required")
	reversed := 0
	for _, e := range erasParam {
		if e.reverseOf != "" {
			reversed++
		}
	}
	calAssert(reversed <= 1, "Invalid era data: only one era can count years backwards")

	// Find the "anchor era" which is the era used for (era-less) `year`. Reversed
	// eras can never be anchors. The era without an `anchorEpoch` property is the
	// anchor.
	var anchorEra *calEra
	all := make([]*calEra, 0, len(erasParam))
	for _, input := range erasParam {
		e := &calEra{code: input.code, anchorEpoch: input.anchorEpoch, isAnchor: input.isAnchor, skip: input.skip, startingYear: 1}
		if input.isoEpoch != nil {
			e.isoEpoch = *input.isoEpoch
		}
		if input.startingYear != nil {
			e.startingYear = *input.startingYear
		}
		if e.isAnchor || (e.anchorEpoch == nil && input.reverseOf == "") {
			calAssert(anchorEra == nil, "Invalid era data: cannot have multiple anchor eras")
			anchorEra = e
			e.anchorEpoch = &calDate{year: calNum(e.startingYear)}
		} else {
			calAssert(e.code != "", "Invalid era data: if era name is blank, it must be the anchor era")
		}
		all = append(all, e)
	}

	// If the era name is undefined, then it's an anchor that doesn't interact
	// with eras at all. For example, Japanese `year` is always the same as ISO
	// `year`.  So this "era" is the anchor era but isn't used for era matching.
	// Strip it from the list that's returned.
	eras := make([]*calEra, 0, len(all))
	reverseOfNames := map[*calEra]string{}
	for i, e := range all {
		if e.code != "" {
			eras = append(eras, e)
			reverseOfNames[e] = erasParam[i].reverseOf
		}
	}

	for _, e := range eras {
		// Some eras are mirror images of another era e.g. B.C. is the reverse of A.D.
		// Replace the string-valued "reverseOf" property with the actual era object
		// that's reversed.
		if reverseOf := reverseOfNames[e]; reverseOf != "" {
			var reversedEra *calEra
			for _, era := range eras {
				if era.code == reverseOf {
					reversedEra = era
					break
				}
			}
			calAssert(reversedEra != nil, "Invalid era data: unmatched reverseOf era: "+reverseOf)
			e.reverseOf = reversedEra
			e.anchorEpoch = reversedEra.anchorEpoch
			e.isoEpoch = reversedEra.isoEpoch
		}
		if e.anchorEpoch.month == nil {
			e.anchorEpoch.month = calNum(1)
		}
		if e.anchorEpoch.day == nil {
			e.anchorEpoch.day = calNum(1)
		}
	}

	// Ensure that the latest epoch is first in the array. This lets us try to
	// match eras in index order, with the last era getting the remaining older
	// years. Any reverse-signed era must be at the end.
	sort.SliceStable(eras, func(i, j int) bool {
		e1, e2 := eras[i], eras[j]
		if e1.reverseOf != nil {
			return false
		}
		if e2.reverseOf != nil {
			return true
		}
		return e2.isoEpoch.year-e1.isoEpoch.year < 0
	})

	// If there's a reversed era, then the one before it must be the era that's
	// being reversed.
	if lastEraReversed := eras[len(eras)-1].reverseOf; lastEraReversed != nil {
		calAssert(lastEraReversed == eras[len(eras)-2], "Invalid era data: invalid reverse-sign era")
	}

	// Finally, add a "genericName" property in the format "era{n} where `n` is
	// zero-based index, with the oldest era being zero. This format is used by
	// older versions of ICU data.
	for i, e := range eras {
		e.genericName = "era" + numberToString(float64(len(eras)-1-i))
	}

	if anchorEra == nil {
		anchorEra = eras[0]
	}
	return eras, anchorEra
}

func isGregorianLeapYear(year float64) bool {
	return jsMod(year, 4) == 0 && (jsMod(year, 100) != 0 || jsMod(year, 400) == 0)
}
