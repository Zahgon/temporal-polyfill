package temporal

import "math/big"

func addTime(t timeRecord, td *timeDuration) timeRecord {
	// timeDuration.sec is a safe integer, but second+timeDuration.sec may not be.
	// minute+trunc(timeDuration.sec/60) is safe. nanosecond+timeDuration.subsec
	// is also safe.
	minutes := jsTrunc(td.sec / 60)
	seconds := td.sec - float64(minutes*60)
	return balanceTime(t.hour, t.minute+minutes, t.second+seconds, t.millisecond, t.microsecond, t.nanosecond+td.subsec)
}

func addInstant(epochNanoseconds *big.Int, td *timeDuration) *big.Int {
	result := td.addToEpochNs(epochNanoseconds)
	validateEpochNanoseconds(result)
	return result
}

// addZonedDateTime defaults overflow to 'constrain', as the original does.
func addZonedDateTime(epochNs *big.Int, timeZone, calendar string, duration internalDuration, overflowParam ...string) *big.Int {
	overflow := "constrain"
	if len(overflowParam) > 0 {
		overflow = overflowParam[0]
	}
	// If only time is to be added, then use Instant math. It's not OK to fall
	// through to the date/time code below because compatible disambiguation in
	// the PlainDateTime=>Instant conversion will change the offset of any
	// ZonedDateTime in the repeated clock time after a backwards transition.
	// When adding/subtracting time units and not dates, this disambiguation is
	// not expected and so is avoided below via a fast path for time-only
	// arithmetic.
	// BTW, this behavior is similar in spirit to offset: 'prefer' in `with`.
	if dateDurationSign(duration.date) == 0 {
		return addInstant(epochNs, duration.time)
	}

	// RFC 5545 requires the date portion to be added in calendar days and the
	// time portion to be added in exact time.
	dt := getISODateTimeFor(timeZone, epochNs)
	addedDate := calendarDateAdd(calendar, dt.isoDate, duration.date, overflow)
	dtIntermediate := combineISODateAndTimeRecord(addedDate, dt.time)

	// Note that 'compatible' is used below because this disambiguation behavior
	// is required by RFC 5545.
	intermediateNs := getEpochNanosecondsFor(timeZone, dtIntermediate, "compatible")
	return addInstant(intermediateNs, duration.time)
}

func addDurations(operation string, duration *Duration, otherParam any) *Duration {
	other := toTemporalDuration(otherParam)
	if operation == "subtract" {
		other = createNegatedTemporalDuration(other)
	}

	largestUnit1 := defaultTemporalLargestUnit(duration)
	largestUnit2 := defaultTemporalLargestUnit(other)
	largestUnit := largerOfTwoTemporalUnits(largestUnit1, largestUnit2)
	if isCalendarUnit(largestUnit) {
		throwRange("For years, months, or weeks arithmetic, use date arithmetic relative to a starting point")
	}

	d1 := toInternalDurationRecordWith24HourDays(duration)
	d2 := toInternalDurationRecordWith24HourDays(other)
	result := combineDateAndTimeDuration(zeroDateDuration(), d1.time.add(d2.time))
	return temporalDurationFromInternal(result, largestUnit)
}

func addDurationToInstant(operation string, instant *Instant, durationLike any) *Instant {
	duration := toTemporalDuration(durationLike)
	if operation == "subtract" {
		duration = createNegatedTemporalDuration(duration)
	}
	largestUnit := defaultTemporalLargestUnit(duration)
	if temporalUnitCategory(largestUnit) == "date" {
		throwRange("Duration field " + largestUnit + " not supported by Temporal.Instant. Try Temporal.ZonedDateTime instead.")
	}
	internal := toInternalDurationRecordWith24HourDays(duration)
	ns := addInstant(instant.epochNs, internal.time)
	return createTemporalInstant(ns)
}

func addDurationToDate(operation string, plainDate *PlainDate, durationLike, options any) *PlainDate {
	calendar := plainDate.calendar

	duration := toTemporalDuration(durationLike)
	if operation == "subtract" {
		duration = createNegatedTemporalDuration(duration)
	}
	d := toDateDurationRecordWithoutTime(duration)

	resolvedOptions := getOptionsObject(options)
	overflow := getTemporalOverflowOption(resolvedOptions)

	addedDate := calendarDateAdd(calendar, plainDate.isoDate, d, overflow)
	return createTemporalDate(addedDate, calendar)
}

func addDurationToDateTime(operation string, dateTime *PlainDateTime, durationLike, options any) *PlainDateTime {
	duration := toTemporalDuration(durationLike)
	if operation == "subtract" {
		duration = createNegatedTemporalDuration(duration)
	}
	resolvedOptions := getOptionsObject(options)
	overflow := getTemporalOverflowOption(resolvedOptions)

	calendar := dateTime.calendar

	internal := toInternalDurationRecordWith24HourDays(duration)

	// Add the time part
	dt := dateTime.isoDateTime
	timeResult := addTime(dt.time, internal.time)
	d := adjustDateDurationRecord(internal.date, timeResult.deltaDays)

	// Delegate the date part addition to the calendar
	rejectDuration(d.years, d.months, d.weeks, d.days, 0, 0, 0, 0, 0, 0)
	addedDate := calendarDateAdd(calendar, dt.isoDate, d, overflow)

	result := combineISODateAndTimeRecord(addedDate, timeResult)
	return createTemporalDateTime(result, calendar)
}

func addDurationToTime(operation string, temporalTime *PlainTime, durationLike any) *PlainTime {
	duration := toTemporalDuration(durationLike)
	if operation == "subtract" {
		duration = createNegatedTemporalDuration(duration)
	}
	internal := toInternalDurationRecordWith24HourDays(duration)
	t := addTime(temporalTime.time, internal.time)
	tm := regulateTime(t.hour, t.minute, t.second, t.millisecond, t.microsecond, t.nanosecond, "reject")
	return createTemporalTime(tm)
}

func addDurationToYearMonth(operation string, yearMonth *PlainYearMonth, durationLike, options any) *PlainYearMonth {
	duration := toTemporalDuration(durationLike)
	if operation == "subtract" {
		duration = createNegatedTemporalDuration(duration)
	}
	resolvedOptions := getOptionsObject(options)
	overflow := getTemporalOverflowOption(resolvedOptions)
	sign := durationSign(duration)

	calendar := yearMonth.calendar
	fields := isoDateToFields(calendar, yearMonth.isoDate, "year-month")
	one := 1.0
	fields.day = &one
	startDate := calendarDateFromFields(calendar, fields, "constrain")
	if sign < 0 {
		nextMonth := calendarDateAdd(calendar, startDate, dateDuration{months: 1}, "constrain")
		startDate = addDaysToISODate(nextMonth, -1)
	}
	durationToAdd := toDateDurationRecordWithoutTime(duration)
	rejectDateRange(startDate)
	addedDate := calendarDateAdd(calendar, startDate, durationToAdd, overflow)
	addedDateFields := isoDateToFields(calendar, addedDate, "year-month")

	iso := calendarYearMonthFromFields(calendar, addedDateFields, overflow)
	return createTemporalYearMonth(iso, calendar)
}

func addDurationToZonedDateTime(operation string, zonedDateTime *ZonedDateTime, durationLike, options any) *ZonedDateTime {
	duration := toTemporalDuration(durationLike)
	if operation == "subtract" {
		duration = createNegatedTemporalDuration(duration)
	}

	resolvedOptions := getOptionsObject(options)
	overflow := getTemporalOverflowOption(resolvedOptions)
	timeZone := zonedDateTime.timeZone
	calendar := zonedDateTime.calendar
	internal := toInternalDurationRecord(duration)
	epochNanoseconds := addZonedDateTime(zonedDateTime.epochNs, timeZone, calendar, internal, overflow)
	return createTemporalZonedDateTime(epochNanoseconds, timeZone, calendar)
}
