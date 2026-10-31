package temporal

import "math"

// partialDurationRecord is what ToTemporalPartialDurationRecord returns; a nil field is undefined.
type partialDurationRecord struct {
	years, months, weeks, days, hours, minutes, seconds, milliseconds, microseconds, nanoseconds *float64
}

// field addresses one of DURATION_FIELDS.
func (r *partialDurationRecord) field(property string) **float64 {
	switch property {
	case "years":
		return &r.years
	case "months":
		return &r.months
	case "weeks":
		return &r.weeks
	case "days":
		return &r.days
	case "hours":
		return &r.hours
	case "minutes":
		return &r.minutes
	case "seconds":
		return &r.seconds
	case "milliseconds":
		return &r.milliseconds
	case "microseconds":
		return &r.microseconds
	}
	return &r.nanoseconds
}

// toTemporalPartialDurationRecord is ToTemporalPartialDurationRecord.
func toTemporalPartialDurationRecord(temporalDurationLikeParam any) partialDurationRecord {
	temporalDurationLike := unwrap(temporalDurationLikeParam)
	if !isObject(temporalDurationLike) {
		throwType("invalid duration-like")
	}
	var result partialDurationRecord
	found := false
	for _, property := range durationFields {
		value := objGet(temporalDurationLike, property)
		if value != nil {
			found = true
			integer := toIntegerIfIntegral(value)
			*result.field(property) = &integer
		}
	}
	if !found {
		throwType("invalid duration-like")
	}
	return result
}

// adjustDateDurationRecord is AdjustDateDurationRecord. The optional arguments are newWeeks and
// newMonths.
func adjustDateDurationRecord(d dateDuration, newDays float64, optional ...float64) dateDuration {
	result := dateDuration{years: d.years, months: d.months, weeks: d.weeks, days: newDays}
	if len(optional) > 0 {
		result.weeks = optional[0]
	}
	if len(optional) > 1 {
		result.months = optional[1]
	}
	return result
}

// zeroDateDuration is ZeroDateDuration.
func zeroDateDuration() dateDuration {
	return dateDuration{}
}

// combineISODateAndTimeRecord is CombineISODateAndTimeRecord.
func combineISODateAndTimeRecord(date isoDate, time timeRecord) isoDateTime {
	return isoDateTime{isoDate: date, time: time}
}

// midnightTimeRecord is MidnightTimeRecord.
func midnightTimeRecord() timeRecord {
	return timeRecord{}
}

// noonTimeRecord is NoonTimeRecord.
func noonTimeRecord() timeRecord {
	return timeRecord{hour: 12}
}

// getTemporalOverflowOption is GetTemporalOverflowOption.
func getTemporalOverflowOption(options any) string {
	return getOption(options, "overflow", []string{"constrain", "reject"}, "constrain").(string)
}

// getTemporalDisambiguationOption is GetTemporalDisambiguationOption.
func getTemporalDisambiguationOption(options any) string {
	return getOption(options, "disambiguation", []string{"compatible", "earlier", "later", "reject"}, "compatible").(string)
}

// getRoundingModeOption is GetRoundingModeOption.
func getRoundingModeOption(options any, fallback string) string {
	return getOption(
		options,
		"roundingMode",
		[]string{"ceil", "floor", "expand", "trunc", "halfCeil", "halfFloor", "halfExpand", "halfTrunc", "halfEven"},
		fallback,
	).(string)
}

// negateRoundingMode is NegateRoundingMode.
func negateRoundingMode(roundingMode string) string {
	switch roundingMode {
	case "ceil":
		return "floor"
	case "floor":
		return "ceil"
	case "halfCeil":
		return "halfFloor"
	case "halfFloor":
		return "halfCeil"
	}
	return roundingMode
}

// getTemporalOffsetOption is GetTemporalOffsetOption.
func getTemporalOffsetOption(options any, fallback string) string {
	return getOption(options, "offset", []string{"prefer", "use", "ignore", "reject"}, fallback).(string)
}

// getTemporalShowCalendarNameOption is GetTemporalShowCalendarNameOption.
func getTemporalShowCalendarNameOption(options any) string {
	return getOption(options, "calendarName", []string{"auto", "always", "never", "critical"}, "auto").(string)
}

// getTemporalShowTimeZoneNameOption is GetTemporalShowTimeZoneNameOption.
func getTemporalShowTimeZoneNameOption(options any) string {
	return getOption(options, "timeZoneName", []string{"auto", "never", "critical"}, "auto").(string)
}

// getTemporalShowOffsetOption is GetTemporalShowOffsetOption.
func getTemporalShowOffsetOption(options any) string {
	return getOption(options, "offset", []string{"auto", "never"}, "auto").(string)
}

// getDirectionOption is GetDirectionOption.
func getDirectionOption(options any) string {
	return getOption(options, "direction", []string{"next", "previous"}, requiredSentinel).(string)
}

// getRoundingIncrementOption is GetRoundingIncrementOption.
func getRoundingIncrementOption(options any) float64 {
	increment := objGet(options, "roundingIncrement")
	if increment == nil {
		return 1
	}
	integerIncrement := toIntegerWithTruncation(increment)
	if integerIncrement < 1 || integerIncrement > 1e9 {
		throwRange("roundingIncrement must be at least 1 and at most 1e9, not " + jsToString(increment))
	}
	return integerIncrement
}

// validateTemporalRoundingIncrement is ValidateTemporalRoundingIncrement.
func validateTemporalRoundingIncrement(increment, dividend float64, inclusive bool) {
	maximum := dividend
	if !inclusive {
		maximum = dividend - 1
	}
	if increment > maximum {
		throwRange("roundingIncrement must be at least 1 and less than " + numberToString(maximum) + ", not " + numberToString(increment))
	}
	if jsMod(dividend, increment) != 0 {
		throwRange("Rounding increment must divide evenly into " + numberToString(dividend))
	}
}

// getTemporalFractionalSecondDigitsOption is GetTemporalFractionalSecondDigitsOption. The result is
// the string "auto" or a float64.
func getTemporalFractionalSecondDigitsOption(normalizedOptions any) any {
	digitsValue := objGet(normalizedOptions, "fractionalSecondDigits")
	if digitsValue == nil {
		return "auto"
	}
	number, isNumber := digitsValue.(float64)
	if !isNumber {
		digitsString := toString(digitsValue)
		if digitsString != "auto" {
			throwRange("fractionalSecondDigits must be 'auto' or 0 through 9, not " + digitsString)
		}
		return "auto"
	}
	digitCount := math.Floor(number)
	if math.IsNaN(digitCount) || math.IsInf(digitCount, 0) || digitCount < 0 || digitCount > 9 {
		throwRange("fractionalSecondDigits must be 'auto' or 0 through 9, not " + numberToString(number))
	}
	return digitCount
}

// secondsStringPrecisionRecord is SecondsStringPrecisionRecord. precision is "minute", "auto" or a
// float64.
type secondsStringPrecisionRecord struct {
	precision any
	unit      string
	increment float64
}

// toSecondsStringPrecisionRecord is ToSecondsStringPrecisionRecord. smallestUnit may be undefined.
func toSecondsStringPrecisionRecord(smallestUnitParam any, precision any) secondsStringPrecisionRecord {
	smallestUnit, _ := optionalString(smallestUnitParam)
	switch smallestUnit {
	case "minute":
		return secondsStringPrecisionRecord{precision: "minute", unit: "minute", increment: 1}
	case "second":
		return secondsStringPrecisionRecord{precision: 0.0, unit: "second", increment: 1}
	case "millisecond":
		return secondsStringPrecisionRecord{precision: 3.0, unit: "millisecond", increment: 1}
	case "microsecond":
		return secondsStringPrecisionRecord{precision: 6.0, unit: "microsecond", increment: 1}
	case "nanosecond":
		return secondsStringPrecisionRecord{precision: 9.0, unit: "nanosecond", increment: 1}
	default: // fall through if option not given
	}
	switch p := precision.(type) {
	case string:
		if p == "auto" {
			return secondsStringPrecisionRecord{precision: precision, unit: "nanosecond", increment: 1}
		}
	case float64:
		switch p {
		case 0:
			return secondsStringPrecisionRecord{precision: precision, unit: "second", increment: 1}
		case 1:
			return secondsStringPrecisionRecord{precision: precision, unit: "millisecond", increment: 100}
		case 2:
			return secondsStringPrecisionRecord{precision: precision, unit: "millisecond", increment: 10}
		case 3:
			return secondsStringPrecisionRecord{precision: precision, unit: "millisecond", increment: 1}
		case 4:
			return secondsStringPrecisionRecord{precision: precision, unit: "microsecond", increment: 100}
		case 5:
			return secondsStringPrecisionRecord{precision: precision, unit: "microsecond", increment: 10}
		case 6:
			return secondsStringPrecisionRecord{precision: precision, unit: "microsecond", increment: 1}
		case 7:
			return secondsStringPrecisionRecord{precision: precision, unit: "nanosecond", increment: 100}
		case 8:
			return secondsStringPrecisionRecord{precision: precision, unit: "nanosecond", increment: 10}
		case 9:
			return secondsStringPrecisionRecord{precision: precision, unit: "nanosecond", increment: 1}
		}
	}
	throwRange("fractionalSecondDigits must be 'auto' or 0 through 9, not " + jsToString(precision))
	return secondsStringPrecisionRecord{}
}

// requiredSentinelType is the type of REQUIRED.
type requiredSentinelType struct{}

// requiredSentinel is REQUIRED, Symbol('~required~').
var requiredSentinel = &requiredSentinelType{}

// getTemporalUnitValuedOption is GetTemporalUnitValuedOption. The optional argument is REQUIRED or
// undefined; a nil result is undefined.
func getTemporalUnitValuedOption(options any, key string, requiredOrUndefined ...any) *string {
	allowedStrings := make([]string, 0, 2*len(temporalUnits)+1)
	for _, unitInfo := range temporalUnits {
		plural := unitInfo[0]
		singular := unitInfo[1]
		allowedStrings = append(allowedStrings, singular, plural)
	}
	allowedStrings = append(allowedStrings, "auto")
	retval := getOption(options, key, allowedStrings, optionalArg(requiredOrUndefined))
	if retval == nil {
		return nil
	}
	// Coerce any plural units into their singular form
	value := retval.(string)
	if singular, ok := singularFor[value]; ok {
		value = singular
	}
	return &value
}

// validateTemporalUnitValue is ValidateTemporalUnitValue. value may be undefined; the optional
// argument is extraValues.
func validateTemporalUnitValue(valueParam any, unitGroup string, extraValues ...[]string) {
	value, present := optionalString(valueParam)
	if !present {
		return
	}
	if len(extraValues) > 0 && containsString(extraValues[0], value) {
		return
	}
	for _, unitInfo := range temporalUnits {
		plural := unitInfo[1]
		category := unitInfo[2]
		if value != plural {
			continue
		}
		if unitGroup == "datetime" || unitGroup == category {
			return
		}
	}
	throwRange(value + " not allowed as a " + unitGroup + " unit")
}

// relativeToRecord is what GetTemporalRelativeToOption returns: at most one field is set.
type relativeToRecord struct {
	zonedRelativeTo *ZonedDateTime
	plainRelativeTo *PlainDate
}

// getTemporalRelativeToOption is GetTemporalRelativeToOption.
func getTemporalRelativeToOption(options any) relativeToRecord {
	relativeTo := objGet(options, "relativeTo")
	if relativeTo == nil {
		return relativeToRecord{}
	}

	offsetBehaviour := "option"
	matchMinutes := false
	var date isoDate
	var time any
	var calendar, timeZone, offset string
	hasTimeZone := false
	if isObject(relativeTo) {
		switch item := relativeTo.(type) {
		case *ZonedDateTime:
			return relativeToRecord{zonedRelativeTo: item}
		case *PlainDate:
			return relativeToRecord{plainRelativeTo: item}
		case *PlainDateTime:
			return relativeToRecord{plainRelativeTo: createTemporalDate(item.isoDateTime.isoDate, item.calendar)}
		}
		calendar = getTemporalCalendarIdentifierWithISODefault(relativeTo)
		fields := prepareCalendarFields(
			calendar,
			relativeTo,
			[]string{"year", "month", "monthCode", "day"},
			[]string{"hour", "minute", "second", "millisecond", "microsecond", "nanosecond", "offset", "timeZone"},
			[]string{},
		)
		interpreted := interpretTemporalDateTimeFields(calendar, fields, "constrain")
		date, time = interpreted.isoDate, interpreted.time
		if fields.offset != nil {
			offset = *fields.offset
		} else {
			offsetBehaviour = "wall"
		}
		if fields.timeZone != nil {
			timeZone, hasTimeZone = *fields.timeZone, true
		}
	} else {
		parsed := parseISODateTime(requireString(relativeTo))
		time, calendar, offset = parsed.timeOrStartOfDay(), parsed.calendar, parsed.offset
		if parsed.tzAnnotation != "" {
			timeZone, hasTimeZone = toTemporalTimeZoneIdentifier(parsed.tzAnnotation), true
			if parsed.z {
				offsetBehaviour = "exact"
			} else if offset == "" {
				offsetBehaviour = "wall"
			}
			// Allow imprecise offset matching unless the provided offset is precise
			matchMinutes = true
			if offset != "" {
				offsetParseResult := reExec(offsetWithPartsRe, offset)
				assert(offsetParseResult != nil, "offset string must re-parse")
				offsetSecondsPart := firstDefined(offsetParseResult.group("offsetSecondSep"), offsetParseResult.group("offsetSecondNoSep"))
				if offsetSecondsPart != "" {
					matchMinutes = false
				}
			}
		} else if parsed.z {
			throwRange("Z designator not supported for PlainDate relativeTo; either remove the Z or add a bracketed time zone")
		}
		if calendar == "" {
			calendar = "iso8601"
		}
		calendar = canonicalizeCalendar(calendar)
		date = isoDate{year: parsed.year, month: parsed.month, day: parsed.day}
	}
	if !hasTimeZone {
		return relativeToRecord{plainRelativeTo: createTemporalDate(date, calendar)}
	}
	offsetNs := 0.0
	if offsetBehaviour == "option" {
		offsetNs = parseDateTimeUTCOffset(offset)
	}
	epochNanoseconds := interpretISODateTimeOffset(
		date,
		time,
		offsetBehaviour,
		offsetNs,
		timeZone,
		"compatible",
		"reject",
		matchMinutes,
	)
	return relativeToRecord{zonedRelativeTo: createTemporalZonedDateTime(epochNanoseconds, timeZone, calendar)}
}

// defaultTemporalLargestUnit is DefaultTemporalLargestUnit.
func defaultTemporalLargestUnit(duration *Duration) string {
	if duration.years != 0 {
		return "year"
	}
	if duration.months != 0 {
		return "month"
	}
	if duration.weeks != 0 {
		return "week"
	}
	if duration.days != 0 {
		return "day"
	}
	if duration.hours != 0 {
		return "hour"
	}
	if duration.minutes != 0 {
		return "minute"
	}
	if duration.seconds != 0 {
		return "second"
	}
	if duration.milliseconds != 0 {
		return "millisecond"
	}
	if duration.microseconds != 0 {
		return "microsecond"
	}
	return "nanosecond"
}

// largerOfTwoTemporalUnits is LargerOfTwoTemporalUnits.
func largerOfTwoTemporalUnits(unit1, unit2 string) string {
	i1 := indexOfString(unitsDescending, unit1)
	i2 := indexOfString(unitsDescending, unit2)
	if i1 > i2 {
		return unit2
	}
	return unit1
}

// isCalendarUnit is IsCalendarUnit.
func isCalendarUnit(unit string) bool {
	return unit == "year" || unit == "month" || unit == "week"
}

// temporalUnitCategory is TemporalUnitCategory.
func temporalUnitCategory(unit string) string {
	if isCalendarUnit(unit) || unit == "day" {
		return "date"
	}
	return "time"
}
