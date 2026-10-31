package temporal

import (
	"math"
	"strings"
)

// parsedISODateTime is what ParseISODateTime returns. startOfDay stands for time: 'start-of-day'. An
// empty tzAnnotation, offset or calendar is undefined: their patterns cannot match nothing.
type parsedISODateTime struct {
	year, month, day float64
	time             timeRecord
	startOfDay       bool
	tzAnnotation     string
	offset           string
	z                bool
	calendar         string
}

// timeOrStartOfDay gives time in the form InterpretISODateTimeOffset takes.
func (p parsedISODateTime) timeOrStartOfDay() any {
	if p.startOfDay {
		return "start-of-day"
	}
	return p.time
}

// firstDefined is a ?? b for two groups of a match.
func firstDefined(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// plusOr is +(group ?? fallback).
func plusOr(group string, fallback float64) float64 {
	if group != "" {
		return stringToNumber(group)
	}
	return fallback
}

// Not a separate abstract operation in the spec, because it only occurs in one
// place: ParseISODateTime. In the code it's more convenient to split up
// ParseISODateTime for the YYYY-MM, MM-DD, and THH:MM:SS parse goals, so it's
// repeated four times.
func processAnnotations(annotations string) string {
	calendar := ""
	calendarWasCritical := false
	for _, m := range annotationRe.FindAllStringSubmatchIndex(annotations, -1) {
		match := &reMatch{re: annotationRe, s: annotations, idx: m}
		critical, key, value := match.at(1), match.at(2), match.at(3)
		if key == "u-ca" {
			if calendar == "" {
				calendar = value
				calendarWasCritical = critical == "!"
			} else if critical == "!" || calendarWasCritical {
				throwRange("Invalid annotations in " + annotations + ": more than one u-ca present with critical flag")
			}
		} else if critical == "!" {
			throwRange("Unrecognized annotation: !" + key + "=" + value)
		}
	}
	return calendar
}

func parseISODateTime(isoString string) parsedISODateTime {
	// ZDT is the superset of fields for every other Temporal type
	match := reExec(zonedDateTimeRe, isoString)
	if match == nil {
		throwRange("invalid RFC 9557 string: " + isoString)
	}
	calendar := processAnnotations(match.group("annotation"))
	yearString := match.group("yearpart")
	if yearString == "-000000" {
		throwRange("invalid RFC 9557 string: " + isoString)
	}
	year := stringToNumber(yearString)
	month := stringToNumber(firstDefined(match.group("monthpartSep"), match.group("monthpartNoSep")))
	day := stringToNumber(firstDefined(match.group("daypartSep"), match.group("daypartNoSep")))
	assert(!math.IsNaN(month) && !math.IsInf(month, 0) && !math.IsNaN(day) && !math.IsInf(day, 0),
		"Month and day must be present if string "+isoString+" matched")
	hasTime := match.group("hourSep") != "" || match.group("hourNoSep") != ""
	hasMinute := match.group("minuteSep") != "" || match.group("minuteNoSep") != ""
	hasSecond := match.group("secondSep") != "" || match.group("secondNoSep") != ""
	hasFraction := match.group("fractionSep") != "" || match.group("fractionNoSep") != ""
	if hasFraction && (!hasSecond || (!hasMinute && hasTime)) {
		throwRange("invalid RFC 9557 string: " + isoString + ", only seconds may be fractional")
	}
	hour := plusOr(firstDefined(match.group("hourSep"), match.group("hourNoSep")), 0)
	minute := plusOr(firstDefined(match.group("minuteSep"), match.group("minuteNoSep")), 0)
	second := plusOr(firstDefined(match.group("secondSep"), match.group("secondNoSep")), 0)
	if second == 60 {
		second = 59
	}
	fraction := firstDefined(match.group("fractionSep"), match.group("fractionNoSep")) + "000000000"
	millisecond := stringToNumber(jsSlice(fraction, 0, 3))
	microsecond := stringToNumber(jsSlice(fraction, 3, 6))
	nanosecond := stringToNumber(jsSlice(fraction, 6, 9))
	offset := ""
	z := false
	if match.group("z") != "" {
		z = true
	} else if match.group("offset") != "" {
		offset = match.group("offset")
	}
	tzAnnotation := match.group("timeZoneID")
	rejectDateTime(year, month, day, hour, minute, second, millisecond, microsecond, nanosecond)
	return parsedISODateTime{
		year:  year,
		month: month,
		day:   day,
		time: timeRecord{
			hour: hour, minute: minute, second: second,
			millisecond: millisecond, microsecond: microsecond, nanosecond: nanosecond,
		},
		startOfDay:   !hasTime,
		tzAnnotation: tzAnnotation,
		offset:       offset,
		z:            z,
		calendar:     calendar,
	}
}

func parseTemporalInstantString(isoString string) parsedISODateTime {
	result := parseISODateTime(isoString)
	if !result.z && result.offset == "" {
		throwRange("Temporal.Instant requires a time zone offset")
	}
	return result
}

func parseTemporalZonedDateTimeString(isoString string) parsedISODateTime {
	result := parseISODateTime(isoString)
	if result.tzAnnotation == "" {
		throwRange("Temporal.ZonedDateTime requires a time zone ID in brackets")
	}
	return result
}

func parseTemporalDateTimeString(isoString string) parsedISODateTime {
	return parseISODateTime(isoString)
}

func parseTemporalDateString(isoString string) parsedISODateTime {
	return parseISODateTime(isoString)
}

// parsedTime is what ParseTemporalTimeString returns.
type parsedTime struct {
	time     timeRecord
	calendar string
}

func parseTemporalTimeString(isoString string) parsedTime {
	match := reExec(timeRe, isoString)
	var hour, minute, second, millisecond, microsecond, nanosecond float64
	var calendar string
	if match != nil {
		calendar = processAnnotations(match.group("annotation"))
		hasFraction := match.group("fractionSep") != "" || match.group("fractionNoSep") != ""
		hasSecond := match.group("secondSep") != "" || match.group("secondNoSep") != ""
		hasMinute := match.group("minuteSep") != "" || match.group("minuteNoSep") != ""
		hasHour := match.group("hourSep") != "" || match.group("hourNoSep") != ""
		if hasFraction && (!hasSecond || (!hasMinute && hasHour)) {
			throwRange("invalid RFC 9557 string: " + isoString + ", only seconds may be fractional")
		}
		hour = stringToNumber(firstDefined(match.group("hourSep"), match.group("hourNoSep")))
		assert(!math.IsNaN(hour) && !math.IsInf(hour, 0), "Hour must be present if string "+isoString+" matched")
		minute = plusOr(firstDefined(match.group("minuteSep"), match.group("minuteNoSep")), 0)
		second = plusOr(firstDefined(match.group("secondSep"), match.group("secondNoSep")), 0)
		if second == 60 {
			second = 59
		}
		fraction := firstDefined(match.group("fractionSep"), match.group("fractionNoSep")) + "000000000"
		millisecond = stringToNumber(jsSlice(fraction, 0, 3))
		microsecond = stringToNumber(jsSlice(fraction, 3, 6))
		nanosecond = stringToNumber(jsSlice(fraction, 6, 9))
		if match.group("z") != "" {
			throwRange("Z designator not supported for PlainTime")
		}
	} else {
		parsed := parseISODateTime(isoString)
		calendar = parsed.calendar
		if parsed.startOfDay {
			throwRange("time is missing in string: " + isoString)
		}
		if parsed.z {
			throwRange("Z designator not supported for PlainTime")
		}
		hour, minute, second = parsed.time.hour, parsed.time.minute, parsed.time.second
		millisecond, microsecond, nanosecond = parsed.time.millisecond, parsed.time.microsecond, parsed.time.nanosecond
	}
	rejectTime(hour, minute, second, millisecond, microsecond, nanosecond)
	result := parsedTime{
		time: timeRecord{
			hour: hour, minute: minute, second: second,
			millisecond: millisecond, microsecond: microsecond, nanosecond: nanosecond,
		},
		calendar: calendar,
	}
	// if it's a date-time string, OK
	if dateTimeSepRe.MatchString(isoString) {
		return result
	}
	notMonthDay := jsCatch(func() {
		parsed := parseTemporalMonthDayString(isoString)
		rejectISODate(1972, parsed.month, parsed.day)
	})
	if notMonthDay {
		notYearMonth := jsCatch(func() {
			parsed := parseTemporalYearMonthString(isoString)
			rejectISODate(parsed.year, parsed.month, 1)
		})
		if notYearMonth {
			return result
		}
	}
	throwRange("invalid RFC 9557 time-only string " + isoString + "; may need a T prefix")
	return result
}

// parsedYearMonth is what ParseTemporalYearMonthString returns.
type parsedYearMonth struct {
	year, month, referenceISODay float64
	calendar                     string
}

func parseTemporalYearMonthString(isoString string) parsedYearMonth {
	match := reExec(yearMonthRe, isoString)
	var result parsedYearMonth
	if match != nil {
		result.calendar = processAnnotations(match.at(3))
		yearString := match.at(1)
		if yearString == "-000000" {
			throwRange("invalid RFC 9557 string: " + isoString)
		}
		result.year = stringToNumber(yearString)
		result.month = stringToNumber(match.at(2))
		result.referenceISODay = 1
		if result.calendar != "" && result.calendar != "iso8601" {
			throwRange("YYYY-MM format is only valid with iso8601 calendar")
		}
	} else {
		parsed := parseISODateTime(isoString)
		result.year, result.month, result.calendar, result.referenceISODay = parsed.year, parsed.month, parsed.calendar, parsed.day
		if parsed.z {
			throwRange("Z designator not supported for PlainYearMonth")
		}
	}
	return result
}

// parsedMonthDay is what ParseTemporalMonthDayString returns.
type parsedMonthDay struct {
	month, day       float64
	calendar         string
	referenceISOYear *float64
}

func parseTemporalMonthDayString(isoString string) parsedMonthDay {
	match := reExec(monthDayRe, isoString)
	var result parsedMonthDay
	if match != nil {
		result.calendar = processAnnotations(match.at(3))
		result.month = stringToNumber(match.at(1))
		result.day = stringToNumber(match.at(2))
		if result.calendar != "" && result.calendar != "iso8601" {
			throwRange("MM-DD format is only valid with iso8601 calendar")
		}
	} else {
		parsed := parseISODateTime(isoString)
		result.month, result.day, result.calendar, result.referenceISOYear = parsed.month, parsed.day, parsed.calendar, &parsed.year
		if parsed.z {
			throwRange("Z designator not supported for PlainMonthDay")
		}
	}
	return result
}

func throwBadTimeZoneStringError(timeZoneString string) {
	// Offset identifiers only support minute precision, but offsets in ISO
	// strings support nanosecond precision. If the identifier is invalid but
	// it's a valid ISO offset, then it has sub-minute precision. Show a clearer
	// error message in that case.
	msg := "Invalid time zone"
	if offsetRe.MatchString(timeZoneString) {
		msg = "Seconds not allowed in offset time zone"
	}
	throwRange(msg + ": " + timeZoneString)
}

// parsedTimeZoneIdentifier is what ParseTimeZoneIdentifier returns: exactly one field is set.
type parsedTimeZoneIdentifier struct {
	tzName        *string
	offsetMinutes *float64
}

func parseTimeZoneIdentifier(identifier string) parsedTimeZoneIdentifier {
	if !timeZoneIdentifierRe.MatchString(identifier) {
		throwBadTimeZoneStringError(identifier)
	}
	if offsetIdentifierRe.MatchString(identifier) {
		offsetNanoseconds := parseDateTimeUTCOffset(identifier)
		// The regex limits the input to minutes precision, so we know that the
		// division below will result in an integer.
		offsetMinutes := offsetNanoseconds / 60e9
		return parsedTimeZoneIdentifier{offsetMinutes: &offsetMinutes}
	}
	return parsedTimeZoneIdentifier{tzName: &identifier}
}

// parsedTimeZoneStringRaw is what ParseTemporalTimeZoneStringRaw returns.
type parsedTimeZoneStringRaw struct {
	tzAnnotation string
	offset       string
	z            bool
}

// This operation doesn't exist in the spec, but in the polyfill it's split from
// ParseTemporalTimeZoneString so that parsing can be tested separately from the
// logic of converting parsed values into a named or offset identifier.
func parseTemporalTimeZoneStringRaw(timeZoneString string) parsedTimeZoneStringRaw {
	if timeZoneIdentifierRe.MatchString(timeZoneString) {
		return parsedTimeZoneStringRaw{tzAnnotation: timeZoneString}
	}
	var result parsedTimeZoneStringRaw
	found := false
	jsCatch(func() {
		// Try parsing ISO string instead
		parsed := parseISODateTime(timeZoneString)
		if parsed.z || parsed.tzAnnotation != "" || parsed.offset != "" {
			result = parsedTimeZoneStringRaw{tzAnnotation: parsed.tzAnnotation, offset: parsed.offset, z: parsed.z}
			found = true
		}
	})
	if !found {
		throwBadTimeZoneStringError(timeZoneString)
	}
	return result
}

func parseTemporalTimeZoneString(stringIdent string) parsedTimeZoneIdentifier {
	raw := parseTemporalTimeZoneStringRaw(stringIdent)
	if raw.tzAnnotation != "" {
		return parseTimeZoneIdentifier(raw.tzAnnotation)
	}
	if raw.z {
		return parseTimeZoneIdentifier("UTC")
	}
	if raw.offset != "" {
		return parseTimeZoneIdentifier(raw.offset)
	}
	assertNotReached()
	return parsedTimeZoneIdentifier{}
}

// parseTemporalDurationStringRaw is ParseTemporalDurationStringRaw. The returned record has the
// values as parsed, signed zeros included; the Duration constructor normalizes them.
func parseTemporalDurationStringRaw(isoString string) Duration {
	match := execDuration(isoString)
	if match == nil {
		throwRange("invalid duration: " + isoString)
	}
	allUndefined := true
	for i := 2; i < len(match.idx)/2; i++ {
		if match.idx[2*i] >= 0 {
			allUndefined = false
		}
	}
	if allUndefined {
		throwRange("invalid duration: " + isoString)
	}
	sign := 1.0
	if match.at(1) == "-" {
		sign = -1
	}
	wholeOrZero := func(i int) float64 {
		if match.at(i) == "" {
			return 0
		}
		return float64(toIntegerWithTruncation(match.at(i)) * sign)
	}
	years := wholeOrZero(2)
	months := wholeOrZero(3)
	weeks := wholeOrZero(4)
	days := wholeOrZero(5)
	hours := wholeOrZero(6)
	fHours := match.at(7)
	minutesStr := match.at(8)
	fMinutes := match.at(9)
	secondsStr := match.at(10)
	fSeconds := match.at(11)
	minutes := 0.0
	seconds := 0.0
	// fractional hours, minutes, or seconds, expressed in whole nanoseconds:
	excessNanoseconds := 0.0

	if fHours != "" {
		if minutesStr != "" || fMinutes != "" || secondsStr != "" || fSeconds != "" {
			throwRange("only the smallest unit can be fractional")
		}
		excessNanoseconds = float64(float64(toIntegerWithTruncation(jsSlice(fHours+"000000000", 0, 9))*3600) * sign)
	} else {
		minutes = wholeOrZero(8)
		if fMinutes != "" {
			if secondsStr != "" || fSeconds != "" {
				throwRange("only the smallest unit can be fractional")
			}
			excessNanoseconds = float64(float64(toIntegerWithTruncation(jsSlice(fMinutes+"000000000", 0, 9))*60) * sign)
		} else {
			seconds = wholeOrZero(10)
			if fSeconds != "" {
				excessNanoseconds = float64(toIntegerWithTruncation(jsSlice(fSeconds+"000000000", 0, 9)) * sign)
			}
		}
	}

	nanoseconds := jsMod(excessNanoseconds, 1000)
	microseconds := jsMod(jsTrunc(excessNanoseconds/1000), 1000)
	milliseconds := jsMod(jsTrunc(excessNanoseconds/1e6), 1000)
	seconds += jsMod(jsTrunc(excessNanoseconds/1e9), 60)
	minutes += jsTrunc(excessNanoseconds / 60e9)

	rejectDuration(years, months, weeks, days, hours, minutes, seconds, milliseconds, microseconds, nanoseconds)
	return Duration{
		years: years, months: months, weeks: weeks, days: days,
		hours: hours, minutes: minutes, seconds: seconds,
		milliseconds: milliseconds, microseconds: microseconds, nanoseconds: nanoseconds,
	}
}

// constructDuration is `new TemporalDuration(...)` on Numbers: the constructor's conversion, which
// turns -0 into 0, and its range check.
func constructDuration(years, months, weeks, days, hours, minutes, seconds, milliseconds, microseconds, nanoseconds float64) *Duration {
	d := &Duration{
		years:        toIntegerIfIntegral(years),
		months:       toIntegerIfIntegral(months),
		weeks:        toIntegerIfIntegral(weeks),
		days:         toIntegerIfIntegral(days),
		hours:        toIntegerIfIntegral(hours),
		minutes:      toIntegerIfIntegral(minutes),
		seconds:      toIntegerIfIntegral(seconds),
		milliseconds: toIntegerIfIntegral(milliseconds),
		microseconds: toIntegerIfIntegral(microseconds),
		nanoseconds:  toIntegerIfIntegral(nanoseconds),
	}
	rejectDuration(d.years, d.months, d.weeks, d.days, d.hours, d.minutes, d.seconds, d.milliseconds, d.microseconds, d.nanoseconds)
	return d
}

func parseTemporalDurationString(isoString string) *Duration {
	d := parseTemporalDurationStringRaw(isoString)
	return constructDuration(d.years, d.months, d.weeks, d.days, d.hours, d.minutes, d.seconds, d.milliseconds, d.microseconds, d.nanoseconds)
}

// regulateISODate is RegulateISODate.
func regulateISODate(yearParam, monthParam, dayParam float64, overflow string) isoDate {
	result := isoDate{year: yearParam, month: monthParam, day: dayParam}
	switch overflow {
	case "reject":
		rejectISODate(result.year, result.month, result.day)
	case "constrain":
		result = constrainISODate(result.year, result.month, result.day)
	}
	return result
}

// regulateTime is RegulateTime.
func regulateTime(hourParam, minuteParam, secondParam, millisecondParam, microsecondParam, nanosecondParam float64, overflow string) timeRecord {
	hour := hourParam
	minute := minuteParam
	second := secondParam
	millisecond := millisecondParam
	microsecond := microsecondParam
	nanosecond := nanosecondParam

	switch overflow {
	case "reject":
		rejectTime(hour, minute, second, millisecond, microsecond, nanosecond)
	case "constrain":
		hour = constrainToRange(hour, 0, 23)
		minute = constrainToRange(minute, 0, 59)
		second = constrainToRange(second, 0, 59)
		millisecond = constrainToRange(millisecond, 0, 999)
		microsecond = constrainToRange(microsecond, 0, 999)
		nanosecond = constrainToRange(nanosecond, 0, 999)
	}
	return timeRecord{hour: hour, minute: minute, second: second, millisecond: millisecond, microsecond: microsecond, nanosecond: nanosecond}
}

// containsString is Array.prototype.includes on a list of strings.
func containsString(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}

// indexOfString is Array.prototype.indexOf on a list of strings.
func indexOfString(list []string, s string) float64 {
	for i, item := range list {
		if item == s {
			return float64(i)
		}
	}
	return -1
}

// trimTrailingZeros is .replace(/0+$/, ”).
func trimTrailingZeros(s string) string {
	return strings.TrimRight(s, "0")
}
