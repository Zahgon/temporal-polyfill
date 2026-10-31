package temporal

import "math"

// showOption reads an optional show* argument, by default "auto".
func showOption(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return "auto"
}

// formatUTCOffsetNanoseconds is FormatUTCOffsetNanoseconds.
func formatUTCOffsetNanoseconds(offsetNs float64) string {
	sign := "+"
	if offsetNs < 0 {
		sign = "-"
	}
	absoluteNs := math.Abs(offsetNs)
	hour := math.Floor(absoluteNs / 3600e9)
	minute := jsMod(math.Floor(absoluteNs/60e9), 60)
	second := jsMod(math.Floor(absoluteNs/1e9), 60)
	subSecondNs := jsMod(absoluteNs, 1e9)
	var precision any = "auto"
	if second == 0 && subSecondNs == 0 {
		precision = "minute"
	}
	timeString := formatTimeString(hour, minute, second, subSecondNs, precision)
	return sign + timeString
}

// isoYearString is ISOYearString.
func isoYearString(year float64) string {
	if year < 0 || year > 9999 {
		sign := "+"
		if year < 0 {
			sign = "-"
		}
		yearNumber := math.Abs(year)
		return sign + toZeroPaddedDecimalString(yearNumber, 6)
	}
	return toZeroPaddedDecimalString(year, 4)
}

// isoDateTimePartString is ISODateTimePartString.
func isoDateTimePartString(part float64) string {
	return toZeroPaddedDecimalString(part, 2)
}

// formatFractionalSeconds is FormatFractionalSeconds; precision is "auto" or a float64.
func formatFractionalSeconds(subSecondNanoseconds float64, precision any) string {
	var fraction string
	if precision == "auto" {
		if subSecondNanoseconds == 0 {
			return ""
		}
		fractionFullPrecision := toZeroPaddedDecimalString(subSecondNanoseconds, 9)
		// now remove any trailing zeroes
		fraction = trimTrailingZeros(fractionFullPrecision)
	} else {
		digits := precision.(float64)
		if digits == 0 {
			return ""
		}
		fractionFullPrecision := toZeroPaddedDecimalString(subSecondNanoseconds, 9)
		fraction = jsSlice(fractionFullPrecision, 0, int(digits))
	}
	return "." + fraction
}

// formatTimeString is FormatTimeString; precision is "minute", "auto" or a float64.
func formatTimeString(hour, minute, second, subSecondNanoseconds float64, precision any) string {
	result := isoDateTimePartString(hour) + ":" + isoDateTimePartString(minute)
	if precision == "minute" {
		return result
	}

	result += ":" + isoDateTimePartString(second)
	result += formatFractionalSeconds(subSecondNanoseconds, precision)
	return result
}

// temporalInstantToString is TemporalInstantToString. timeZone may be undefined.
func temporalInstantToString(instant *Instant, timeZoneParam any, precision any) string {
	timeZone, hasTimeZone := optionalString(timeZoneParam)
	outputTimeZone := timeZone
	if !hasTimeZone {
		outputTimeZone = "UTC"
	}
	epochNs := instant.epochNs
	iso := getISODateTimeFor(outputTimeZone, epochNs)
	dateTimeString := isoDateTimeToString(iso, "iso8601", precision, "never")
	timeZoneString := "Z"
	if hasTimeZone {
		offsetNs := getOffsetNanosecondsFor(outputTimeZone, epochNs)
		timeZoneString = formatDateTimeUTCOffsetRounded(offsetNs)
	}
	return dateTimeString + timeZoneString
}

// toStringOptions is ToStringOptions.
type toStringOptions struct {
	unit         string
	increment    float64
	roundingMode string
}

// temporalDurationToString is TemporalDurationToString; precision is "auto" or a float64.
func temporalDurationToString(duration *Duration, precision any) string {
	years := duration.years
	months := duration.months
	weeks := duration.weeks
	days := duration.days
	hours := duration.hours
	minutes := duration.minutes
	sign := durationSign(duration)

	datePart := ""
	if years != 0 {
		datePart += numberToString(math.Abs(years)) + "Y"
	}
	if months != 0 {
		datePart += numberToString(math.Abs(months)) + "M"
	}
	if weeks != 0 {
		datePart += numberToString(math.Abs(weeks)) + "W"
	}
	if days != 0 {
		datePart += numberToString(math.Abs(days)) + "D"
	}

	timePart := ""
	if hours != 0 {
		timePart += numberToString(math.Abs(hours)) + "H"
	}
	if minutes != 0 {
		timePart += numberToString(math.Abs(minutes)) + "M"
	}

	// Keeping sub-second units separate avoids losing precision after resolving
	// any overflows from rounding
	secondsDuration := timeDurationFromComponents(
		0,
		0,
		duration.seconds,
		duration.milliseconds,
		duration.microseconds,
		duration.nanoseconds,
	)
	if !secondsDuration.isZero() ||
		containsString([]string{"second", "millisecond", "microsecond", "nanosecond"}, defaultTemporalLargestUnit(duration)) ||
		precision != "auto" {
		secondsPart := math.Abs(secondsDuration.sec)
		subSecondsPart := formatFractionalSeconds(math.Abs(secondsDuration.subsec), precision)
		timePart += numberToString(secondsPart) + subSecondsPart + "S"
	}
	result := "P" + datePart
	if sign < 0 {
		result = "-" + result
	}
	if timePart != "" {
		result = result + "T" + timePart
	}
	return result
}

func isoDateToString(date isoDate) string {
	yearString := isoYearString(date.year)
	monthString := isoDateTimePartString(date.month)
	dayString := isoDateTimePartString(date.day)
	return yearString + "-" + monthString + "-" + dayString
}

// temporalDateToString is TemporalDateToString. The optional argument is showCalendar.
func temporalDateToString(date *PlainDate, showCalendar ...string) string {
	dateString := isoDateToString(date.isoDate)
	calendar := formatCalendarAnnotation(date.calendar, showOption(showCalendar))
	return dateString + calendar
}

// timeRecordToString is TimeRecordToString.
func timeRecordToString(time timeRecord, precision any) string {
	return formatTimeString(time.hour, time.minute, time.second, subSecondNanosecondsOf(time), precision)
}

// isoDateTimeToString is ISODateTimeToString. The optional argument is showCalendar.
func isoDateTimeToString(dateTime isoDateTime, calendar string, precision any, showCalendar ...string) string {
	date, time := dateTime.isoDate, dateTime.time
	yearString := isoYearString(date.year)
	monthString := isoDateTimePartString(date.month)
	dayString := isoDateTimePartString(date.day)
	timeString := formatTimeString(time.hour, time.minute, time.second, subSecondNanosecondsOf(time), precision)
	calendarString := formatCalendarAnnotation(calendar, showOption(showCalendar))
	return yearString + "-" + monthString + "-" + dayString + "T" + timeString + calendarString
}

// temporalMonthDayToString is TemporalMonthDayToString. The optional argument is showCalendar.
func temporalMonthDayToString(monthDay *PlainMonthDay, showCalendarParam ...string) string {
	showCalendar := showOption(showCalendarParam)
	date := monthDay.isoDate
	monthString := isoDateTimePartString(date.month)
	dayString := isoDateTimePartString(date.day)
	resultString := monthString + "-" + dayString
	calendar := monthDay.calendar
	if showCalendar == "always" || showCalendar == "critical" || calendar != "iso8601" {
		yearString := isoYearString(date.year)
		resultString = yearString + "-" + resultString
	}
	return resultString + formatCalendarAnnotation(calendar, showCalendar)
}

// temporalYearMonthToString is TemporalYearMonthToString. The optional argument is showCalendar.
func temporalYearMonthToString(yearMonth *PlainYearMonth, showCalendarParam ...string) string {
	showCalendar := showOption(showCalendarParam)
	date := yearMonth.isoDate
	yearString := isoYearString(date.year)
	monthString := isoDateTimePartString(date.month)
	resultString := yearString + "-" + monthString
	calendar := yearMonth.calendar
	if showCalendar == "always" || showCalendar == "critical" || calendar != "iso8601" {
		dayString := isoDateTimePartString(date.day)
		resultString += "-" + dayString
	}
	return resultString + formatCalendarAnnotation(calendar, showCalendar)
}

// temporalZonedDateTimeToString is TemporalZonedDateTimeToString. The optional arguments are, in
// order, showCalendar, showTimeZone and showOffset (strings, by default "auto") and options (a
// toStringOptions or *toStringOptions).
func temporalZonedDateTimeToString(zdt *ZonedDateTime, precision any, optional ...any) string {
	showCalendar, showTimeZone, showOffset := "auto", "auto", "auto"
	var options *toStringOptions
	if len(optional) > 0 {
		showCalendar = optional[0].(string)
	}
	if len(optional) > 1 {
		showTimeZone = optional[1].(string)
	}
	if len(optional) > 2 {
		showOffset = optional[2].(string)
	}
	if len(optional) > 3 {
		switch v := optional[3].(type) {
		case toStringOptions:
			options = &v
		case *toStringOptions:
			options = v
		}
	}
	epochNs := zdt.epochNs

	if options != nil {
		epochNs = roundTemporalInstant(epochNs, options.increment, options.unit, options.roundingMode)
	}

	tz := zdt.timeZone
	offsetNs := getOffsetNanosecondsFor(tz, epochNs)
	iso := getISODateTimeFor(tz, epochNs)
	dateTimeString := isoDateTimeToString(iso, "iso8601", precision, "never")
	if showOffset != "never" {
		dateTimeString += formatDateTimeUTCOffsetRounded(offsetNs)
	}
	if showTimeZone != "never" {
		flag := ""
		if showTimeZone == "critical" {
			flag = "!"
		}
		dateTimeString += "[" + flag + tz + "]"
	}
	dateTimeString += formatCalendarAnnotation(zdt.calendar, showCalendar)
	return dateTimeString
}
