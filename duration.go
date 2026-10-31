package temporal

import (
	"math"
	"math/big"
)

// NewDuration is new Temporal.Duration(...). Every argument defaults to 0.
func NewDuration(yearsParam, monthsParam, weeksParam, daysParam, hoursParam, minutesParam, secondsParam, millisecondsParam, microsecondsParam, nanosecondsParam any) *Duration {
	years := toIntegerIfIntegral(defaultTo(yearsParam, 0.0))
	months := toIntegerIfIntegral(defaultTo(monthsParam, 0.0))
	weeks := toIntegerIfIntegral(defaultTo(weeksParam, 0.0))
	days := toIntegerIfIntegral(defaultTo(daysParam, 0.0))
	hours := toIntegerIfIntegral(defaultTo(hoursParam, 0.0))
	minutes := toIntegerIfIntegral(defaultTo(minutesParam, 0.0))
	seconds := toIntegerIfIntegral(defaultTo(secondsParam, 0.0))
	milliseconds := toIntegerIfIntegral(defaultTo(millisecondsParam, 0.0))
	microseconds := toIntegerIfIntegral(defaultTo(microsecondsParam, 0.0))
	nanoseconds := toIntegerIfIntegral(defaultTo(nanosecondsParam, 0.0))

	rejectDuration(years, months, weeks, days, hours, minutes, seconds, milliseconds, microseconds, nanoseconds)

	return &Duration{
		years:        years,
		months:       months,
		weeks:        weeks,
		days:         days,
		hours:        hours,
		minutes:      minutes,
		seconds:      seconds,
		milliseconds: milliseconds,
		microseconds: microseconds,
		nanoseconds:  nanoseconds,
	}
}

// Years is the years getter.
func (d *Duration) Years() float64 { return d.years }

// Months is the months getter.
func (d *Duration) Months() float64 { return d.months }

// Weeks is the weeks getter.
func (d *Duration) Weeks() float64 { return d.weeks }

// Days is the days getter.
func (d *Duration) Days() float64 { return d.days }

// Hours is the hours getter.
func (d *Duration) Hours() float64 { return d.hours }

// Minutes is the minutes getter.
func (d *Duration) Minutes() float64 { return d.minutes }

// Seconds is the seconds getter.
func (d *Duration) Seconds() float64 { return d.seconds }

// Milliseconds is the milliseconds getter.
func (d *Duration) Milliseconds() float64 { return d.milliseconds }

// Microseconds is the microseconds getter.
func (d *Duration) Microseconds() float64 { return d.microseconds }

// Nanoseconds is the nanoseconds getter.
func (d *Duration) Nanoseconds() float64 { return d.nanoseconds }

// Sign is the sign getter.
func (d *Duration) Sign() float64 { return durationSign(d) }

// Blank is the blank getter.
func (d *Duration) Blank() bool { return durationSign(d) == 0 }

// With is Temporal.Duration.prototype.with.
func (d *Duration) With(durationLike any) *Duration {
	partialDuration := toTemporalPartialDurationRecord(durationLike)
	pick := func(partial *float64, slot float64) any {
		if partial != nil {
			return *partial
		}
		return slot
	}
	return NewDuration(
		pick(partialDuration.years, d.years),
		pick(partialDuration.months, d.months),
		pick(partialDuration.weeks, d.weeks),
		pick(partialDuration.days, d.days),
		pick(partialDuration.hours, d.hours),
		pick(partialDuration.minutes, d.minutes),
		pick(partialDuration.seconds, d.seconds),
		pick(partialDuration.milliseconds, d.milliseconds),
		pick(partialDuration.microseconds, d.microseconds),
		pick(partialDuration.nanoseconds, d.nanoseconds),
	)
}

// Negated is Temporal.Duration.prototype.negated.
func (d *Duration) Negated() *Duration { return createNegatedTemporalDuration(d) }

// Abs is Temporal.Duration.prototype.abs.
func (d *Duration) Abs() *Duration {
	return NewDuration(
		math.Abs(d.years),
		math.Abs(d.months),
		math.Abs(d.weeks),
		math.Abs(d.days),
		math.Abs(d.hours),
		math.Abs(d.minutes),
		math.Abs(d.seconds),
		math.Abs(d.milliseconds),
		math.Abs(d.microseconds),
		math.Abs(d.nanoseconds),
	)
}

// Add is Temporal.Duration.prototype.add.
func (d *Duration) Add(other any) *Duration { return addDurations("add", d, other) }

// Subtract is Temporal.Duration.prototype.subtract.
func (d *Duration) Subtract(other any) *Duration { return addDurations("subtract", d, other) }

// Round is Temporal.Duration.prototype.round.
func (d *Duration) Round(roundToParam any) *Duration {
	if isUndefined(roundToParam) {
		throwType("options parameter is required")
	}

	existingLargestUnit := defaultTemporalLargestUnit(d)
	roundTo := oneProp("smallestUnit", roundToParam)

	largestUnitOption := getTemporalUnitValuedOption(roundTo, "largestUnit")
	relativeTo := getTemporalRelativeToOption(roundTo)
	plainRelativeTo, zonedRelativeTo := relativeTo.plainRelativeTo, relativeTo.zonedRelativeTo
	roundingIncrement := getRoundingIncrementOption(roundTo)
	roundingMode := getRoundingModeOption(roundTo, "halfExpand")
	smallestUnitOption := getTemporalUnitValuedOption(roundTo, "smallestUnit")
	validateTemporalUnitValue(smallestUnitOption, "datetime")

	smallestUnitPresent := true
	smallestUnit := "nanosecond"
	if smallestUnitOption == nil {
		smallestUnitPresent = false
	} else {
		smallestUnit = *smallestUnitOption
	}
	defaultLargestUnit := largerOfTwoTemporalUnits(existingLargestUnit, smallestUnit)
	largestUnitPresent := true
	largestUnit := defaultLargestUnit
	if largestUnitOption == nil {
		largestUnitPresent = false
	} else {
		largestUnit = *largestUnitOption
	}
	if largestUnit == "auto" {
		largestUnit = defaultLargestUnit
	}
	if !smallestUnitPresent && !largestUnitPresent {
		throwRange("at least one of smallestUnit or largestUnit is required")
	}
	if largerOfTwoTemporalUnits(largestUnit, smallestUnit) != largestUnit {
		throwRange("largestUnit " + largestUnit + " cannot be smaller than smallestUnit " + smallestUnit)
	}

	maximumIncrements := map[string]float64{
		"hour":        24,
		"minute":      60,
		"second":      60,
		"millisecond": 1000,
		"microsecond": 1000,
		"nanosecond":  1000,
	}
	if maximum, ok := maximumIncrements[smallestUnit]; ok {
		validateTemporalRoundingIncrement(roundingIncrement, maximum, false)
	}
	if roundingIncrement > 1 && temporalUnitCategory(smallestUnit) == "date" && largestUnit != smallestUnit {
		throwRange("For calendar units with roundingIncrement > 1, use largestUnit = smallestUnit")
	}

	if zonedRelativeTo != nil {
		duration := toInternalDurationRecord(d)
		timeZone := zonedRelativeTo.timeZone
		calendar := zonedRelativeTo.calendar
		relativeEpochNs := zonedRelativeTo.epochNs
		targetEpochNs := addZonedDateTime(relativeEpochNs, timeZone, calendar, duration)
		duration = differenceZonedDateTimeWithRounding(
			relativeEpochNs,
			targetEpochNs,
			timeZone,
			calendar,
			largestUnit,
			roundingIncrement,
			smallestUnit,
			roundingMode,
		)
		if temporalUnitCategory(largestUnit) == "date" {
			largestUnit = "hour"
		}
		return temporalDurationFromInternal(duration, largestUnit)
	}

	if plainRelativeTo != nil {
		duration := toInternalDurationRecordWith24HourDays(d)
		targetTime := addTime(midnightTimeRecord(), duration.time)

		// Delegate the date part addition to the calendar
		isoRelativeToDate := plainRelativeTo.isoDate
		calendar := plainRelativeTo.calendar
		dateDuration := adjustDateDurationRecord(duration.date, targetTime.deltaDays)
		targetDate := calendarDateAdd(calendar, isoRelativeToDate, dateDuration, "constrain")

		isoDateTime := combineISODateAndTimeRecord(isoRelativeToDate, midnightTimeRecord())
		targetDateTime := combineISODateAndTimeRecord(targetDate, targetTime)
		duration = differencePlainDateTimeWithRounding(
			isoDateTime,
			targetDateTime,
			calendar,
			largestUnit,
			roundingIncrement,
			smallestUnit,
			roundingMode,
		)
		return temporalDurationFromInternal(duration, largestUnit)
	}

	// No reference date to calculate difference relative to
	if isCalendarUnit(existingLargestUnit) {
		throwRange("a starting point is required for " + existingLargestUnit + "s balancing")
	}
	if isCalendarUnit(largestUnit) {
		throwRange("a starting point is required for " + largestUnit + "s balancing")
	}
	if isCalendarUnit(smallestUnit) {
		panic(&Error{Name: "Error", Message: "assertion failure: smallestUnit was larger than largestUnit"})
	}
	internalDuration := toInternalDurationRecordWith24HourDays(d)
	if smallestUnit == "day" {
		// First convert time units up to days
		quotient, remainder := internalDuration.time.divmod(dayNanos)
		days := internalDuration.date.days + quotient + totalTimeDuration(remainder, "day")
		days = roundNumberToIncrement(days, roundingIncrement, roundingMode)
		dateDuration := dateDuration{years: 0, months: 0, weeks: 0, days: days}
		internalDuration = combineDateAndTimeDuration(dateDuration, timeDurationZero)
	} else {
		timeDuration := roundTimeDuration(internalDuration.time, roundingIncrement, smallestUnit, roundingMode)
		internalDuration = combineDateAndTimeDuration(zeroDateDuration(), timeDuration)
	}
	return temporalDurationFromInternal(internalDuration, largestUnit)
}

// Total is Temporal.Duration.prototype.total.
func (d *Duration) Total(optionsParam any) float64 {
	if isUndefined(optionsParam) {
		throwType("options argument is required")
	}
	options := oneProp("unit", optionsParam)
	relativeTo := getTemporalRelativeToOption(options)
	plainRelativeTo, zonedRelativeTo := relativeTo.plainRelativeTo, relativeTo.zonedRelativeTo
	unit := *getTemporalUnitValuedOption(options, "unit", requiredSentinel)
	validateTemporalUnitValue(unit, "datetime")

	if zonedRelativeTo != nil {
		duration := toInternalDurationRecord(d)
		timeZone := zonedRelativeTo.timeZone
		calendar := zonedRelativeTo.calendar
		relativeEpochNs := zonedRelativeTo.epochNs
		targetEpochNs := addZonedDateTime(relativeEpochNs, timeZone, calendar, duration)
		return differenceZonedDateTimeWithTotal(relativeEpochNs, targetEpochNs, timeZone, calendar, unit)
	}

	if plainRelativeTo != nil {
		duration := toInternalDurationRecordWith24HourDays(d)
		targetTime := addTime(midnightTimeRecord(), duration.time)

		// Delegate the date part addition to the calendar
		isoRelativeToDate := plainRelativeTo.isoDate
		calendar := plainRelativeTo.calendar
		dateDuration := adjustDateDurationRecord(duration.date, targetTime.deltaDays)
		targetDate := calendarDateAdd(calendar, isoRelativeToDate, dateDuration, "constrain")

		isoDateTime := combineISODateAndTimeRecord(isoRelativeToDate, midnightTimeRecord())
		targetDateTime := combineISODateAndTimeRecord(targetDate, targetTime)
		return differencePlainDateTimeWithTotal(isoDateTime, targetDateTime, calendar, unit)
	}

	// No reference date to calculate difference relative to
	largestUnit := defaultTemporalLargestUnit(d)
	if isCalendarUnit(largestUnit) {
		throwRange("a starting point is required for " + largestUnit + "s total")
	}
	if isCalendarUnit(unit) {
		throwRange("a starting point is required for " + unit + "s total")
	}
	duration := toInternalDurationRecordWith24HourDays(d)
	return totalTimeDuration(duration.time, unit)
}

// ToString is Temporal.Duration.prototype.toString.
func (d *Duration) ToString(options any) string {
	resolvedOptions := getOptionsObject(options)
	digits := getTemporalFractionalSecondDigitsOption(resolvedOptions)
	roundingMode := getRoundingModeOption(resolvedOptions, "trunc")
	smallestUnit := getTemporalUnitValuedOption(resolvedOptions, "smallestUnit")
	validateTemporalUnitValue(smallestUnit, "time")
	if smallestUnit != nil && (*smallestUnit == "hour" || *smallestUnit == "minute") {
		throwRange(`smallestUnit must be a time unit other than "hours" or "minutes"`)
	}
	rec := toSecondsStringPrecisionRecord(smallestUnit, digits)

	if rec.unit == "nanosecond" && rec.increment == 1 {
		return temporalDurationToString(d, rec.precision)
	}

	largestUnit := defaultTemporalLargestUnit(d)
	internalDuration := toInternalDurationRecord(d)
	timeDuration := roundTimeDuration(internalDuration.time, rec.increment, rec.unit, roundingMode)
	internalDuration = combineDateAndTimeDuration(internalDuration.date, timeDuration)
	roundedDuration := temporalDurationFromInternal(
		internalDuration,
		largerOfTwoTemporalUnits(largestUnit, "second"),
	)
	return temporalDurationToString(roundedDuration, rec.precision)
}

// ToJSON is Temporal.Duration.prototype.toJSON.
func (d *Duration) ToJSON() string { return temporalDurationToString(d, "auto") }

// ToLocaleString is Temporal.Duration.prototype.toLocaleString.
func (d *Duration) ToLocaleString(locales, options any) string {
	formatter := newIntlDurationFormat(locales, options)
	return modifiedIntlDurationFormatPrototypeFormat(formatter, d)
}

func (d *Duration) jsValueOf() { valueOfThrows("Duration") }

func (d *Duration) jsToString() string { return d.ToString(nil) }

// DurationFrom is Temporal.Duration.from.
func DurationFrom(item any) *Duration { return toTemporalDuration(item) }

// DurationCompare is Temporal.Duration.compare.
func DurationCompare(oneParam, twoParam, options any) float64 {
	one := toTemporalDuration(oneParam)
	two := toTemporalDuration(twoParam)
	resolvedOptions := getOptionsObject(options)
	relativeTo := getTemporalRelativeToOption(resolvedOptions)
	plainRelativeTo, zonedRelativeTo := relativeTo.plainRelativeTo, relativeTo.zonedRelativeTo

	if one.years == two.years &&
		one.months == two.months &&
		one.weeks == two.weeks &&
		one.days == two.days &&
		one.hours == two.hours &&
		one.minutes == two.minutes &&
		one.seconds == two.seconds &&
		one.milliseconds == two.milliseconds &&
		one.microseconds == two.microseconds &&
		one.nanoseconds == two.nanoseconds {
		return 0
	}

	largestUnit1 := defaultTemporalLargestUnit(one)
	largestUnit2 := defaultTemporalLargestUnit(two)
	duration1 := toInternalDurationRecord(one)
	duration2 := toInternalDurationRecord(two)

	if zonedRelativeTo != nil &&
		(temporalUnitCategory(largestUnit1) == "date" || temporalUnitCategory(largestUnit2) == "date") {
		timeZone := zonedRelativeTo.timeZone
		calendar := zonedRelativeTo.calendar
		epochNs := zonedRelativeTo.epochNs

		after1 := addZonedDateTime(epochNs, timeZone, calendar, duration1)
		after2 := addZonedDateTime(epochNs, timeZone, calendar, duration2)
		return comparisonResult(float64(new(big.Int).Sub(after1, after2).Sign()))
	}

	d1 := duration1.date.days
	d2 := duration2.date.days
	if isCalendarUnit(largestUnit1) || isCalendarUnit(largestUnit2) {
		if plainRelativeTo == nil {
			throwRange("A starting point is required for years, months, or weeks comparison")
		}
		d1 = dateDurationDays(duration1.date, plainRelativeTo)
		d2 = dateDurationDays(duration2.date, plainRelativeTo)
	}
	timeDuration1 := duration1.time.add24HourDays(d1)
	timeDuration2 := duration2.time.add24HourDays(d2)
	return timeDuration1.cmp(timeDuration2)
}

func (d *Duration) jsGet(key string) any {
	switch key {
	case "years":
		return d.years
	case "months":
		return d.months
	case "weeks":
		return d.weeks
	case "days":
		return d.days
	case "hours":
		return d.hours
	case "minutes":
		return d.minutes
	case "seconds":
		return d.seconds
	case "milliseconds":
		return d.milliseconds
	case "microseconds":
		return d.microseconds
	case "nanoseconds":
		return d.nanoseconds
	case "sign":
		return d.Sign()
	case "blank":
		return d.Blank()
	}
	return nil
}
