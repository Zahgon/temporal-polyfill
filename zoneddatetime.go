package temporal

import "math/big"

// NewZonedDateTime is new Temporal.ZonedDateTime(epochNanoseconds, timeZone, calendar). calendar
// defaults to "iso8601". A Go caller always passes the first argument, so the original's
// missing-argument check cannot fire: nil is an explicit undefined.
func NewZonedDateTime(epochNanosecondsParam, timeZoneParam, calendarParam any) *ZonedDateTime {
	epochNanoseconds := toBigInt(epochNanosecondsParam)
	timeZone := requireString(timeZoneParam)
	parsed := parseTimeZoneIdentifier(timeZone)
	if parsed.offsetMinutes == nil {
		// if offsetMinutes is undefined, then tzName must be present
		record := getAvailableNamedTimeZoneIdentifier(*parsed.tzName)
		if record == nil {
			throwRange("unknown time zone " + *parsed.tzName)
		}
		timeZone = record.identifier
	} else {
		timeZone = formatOffsetTimeZoneIdentifier(*parsed.offsetMinutes)
	}
	calendar := canonicalizeCalendar(requireString(defaultTo(calendarParam, "iso8601")))

	return createTemporalZonedDateTime(epochNanoseconds, timeZone, calendar)
}

func (z *ZonedDateTime) dateTime() isoDateTime {
	return getISODateTimeFor(z.timeZone, z.epochNs)
}

// CalendarId is the calendarId getter.
func (z *ZonedDateTime) CalendarId() string { return z.calendar }

// TimeZoneId is the timeZoneId getter.
func (z *ZonedDateTime) TimeZoneId() string { return z.timeZone }

// Year is the year getter.
func (z *ZonedDateTime) Year() float64 {
	return calendarImplForID(z.calendar).isoToDate(z.dateTime().isoDate, reqYear).year
}

// Month is the month getter.
func (z *ZonedDateTime) Month() float64 {
	return calendarImplForID(z.calendar).isoToDate(z.dateTime().isoDate, reqMonth).month
}

// MonthCode is the monthCode getter.
func (z *ZonedDateTime) MonthCode() string {
	return calendarImplForID(z.calendar).isoToDate(z.dateTime().isoDate, reqMonthCode).monthCode
}

// Day is the day getter.
func (z *ZonedDateTime) Day() float64 {
	return calendarImplForID(z.calendar).isoToDate(z.dateTime().isoDate, reqDay).day
}

// Hour is the hour getter.
func (z *ZonedDateTime) Hour() float64 { return z.dateTime().time.hour }

// Minute is the minute getter.
func (z *ZonedDateTime) Minute() float64 { return z.dateTime().time.minute }

// Second is the second getter.
func (z *ZonedDateTime) Second() float64 { return z.dateTime().time.second }

// Millisecond is the millisecond getter.
func (z *ZonedDateTime) Millisecond() float64 { return z.dateTime().time.millisecond }

// Microsecond is the microsecond getter.
func (z *ZonedDateTime) Microsecond() float64 { return z.dateTime().time.microsecond }

// Nanosecond is the nanosecond getter.
func (z *ZonedDateTime) Nanosecond() float64 { return z.dateTime().time.nanosecond }

// Era is the era getter.
func (z *ZonedDateTime) Era() *string {
	return calendarImplForID(z.calendar).isoToDate(z.dateTime().isoDate, reqEra).era
}

// EraYear is the eraYear getter.
func (z *ZonedDateTime) EraYear() *float64 {
	return calendarImplForID(z.calendar).isoToDate(z.dateTime().isoDate, reqEraYear).eraYear
}

// EpochMilliseconds is the epochMilliseconds getter.
func (z *ZonedDateTime) EpochMilliseconds() float64 {
	value := z.epochNs
	return epochNsToMs(value, "floor")
}

// EpochNanoseconds is the epochNanoseconds getter.
func (z *ZonedDateTime) EpochNanoseconds() *big.Int { return toBigIntExternal(z.epochNs) }

// DayOfWeek is the dayOfWeek getter.
func (z *ZonedDateTime) DayOfWeek() float64 {
	return calendarImplForID(z.calendar).isoToDate(z.dateTime().isoDate, reqDayOfWeek).dayOfWeek
}

// DayOfYear is the dayOfYear getter.
func (z *ZonedDateTime) DayOfYear() float64 {
	return calendarImplForID(z.calendar).isoToDate(z.dateTime().isoDate, reqDayOfYear).dayOfYear
}

// WeekOfYear is the weekOfYear getter.
func (z *ZonedDateTime) WeekOfYear() *float64 {
	return calendarImplForID(z.calendar).isoToDate(z.dateTime().isoDate, reqWeekOfYear).weekOfYear
}

// YearOfWeek is the yearOfWeek getter.
func (z *ZonedDateTime) YearOfWeek() *float64 {
	return calendarImplForID(z.calendar).isoToDate(z.dateTime().isoDate, reqWeekOfYear).yearOfWeek
}

// HoursInDay is the hoursInDay getter.
func (z *ZonedDateTime) HoursInDay() float64 {
	timeZone := z.timeZone
	today := z.dateTime().isoDate
	tomorrow := addDaysToISODate(today, 1)
	todayNs := getStartOfDay(timeZone, today)
	tomorrowNs := getStartOfDay(timeZone, tomorrow)
	diff := timeDurationFromEpochNsDiff(tomorrowNs, todayNs)
	return totalTimeDuration(diff, "hour")
}

// DaysInWeek is the daysInWeek getter.
func (z *ZonedDateTime) DaysInWeek() float64 {
	return calendarImplForID(z.calendar).isoToDate(z.dateTime().isoDate, reqDaysInWeek).daysInWeek
}

// DaysInMonth is the daysInMonth getter.
func (z *ZonedDateTime) DaysInMonth() float64 {
	return calendarImplForID(z.calendar).isoToDate(z.dateTime().isoDate, reqDaysInMonth).daysInMonth
}

// DaysInYear is the daysInYear getter.
func (z *ZonedDateTime) DaysInYear() float64 {
	return calendarImplForID(z.calendar).isoToDate(z.dateTime().isoDate, reqDaysInYear).daysInYear
}

// MonthsInYear is the monthsInYear getter.
func (z *ZonedDateTime) MonthsInYear() float64 {
	return calendarImplForID(z.calendar).isoToDate(z.dateTime().isoDate, reqMonthsInYear).monthsInYear
}

// InLeapYear is the inLeapYear getter.
func (z *ZonedDateTime) InLeapYear() bool {
	return calendarImplForID(z.calendar).isoToDate(z.dateTime().isoDate, reqInLeapYear).inLeapYear
}

// Offset is the offset getter.
func (z *ZonedDateTime) Offset() string {
	offsetNs := getOffsetNanosecondsFor(z.timeZone, z.epochNs)
	return formatUTCOffsetNanoseconds(offsetNs)
}

// OffsetNanoseconds is the offsetNanoseconds getter.
func (z *ZonedDateTime) OffsetNanoseconds() float64 {
	return getOffsetNanosecondsFor(z.timeZone, z.epochNs)
}

// With is Temporal.ZonedDateTime.prototype.with.
func (z *ZonedDateTime) With(temporalZonedDateTimeLike, options any) *ZonedDateTime {
	if !isObject(temporalZonedDateTimeLike) {
		throwType("invalid zoned-date-time-like")
	}
	rejectTemporalLikeObject(temporalZonedDateTimeLike)

	calendar := z.calendar
	timeZone := z.timeZone
	epochNs := z.epochNs
	offsetNs := getOffsetNanosecondsFor(timeZone, epochNs)
	isoDateTime := z.dateTime()
	fields := isoDateToFields(calendar, isoDateTime.isoDate, "date")
	spreadTimeIntoFields(fields, isoDateTime.time)
	offsetString := formatUTCOffsetNanoseconds(offsetNs)
	fields.offset = &offsetString
	partialZonedDateTime := prepareCalendarFields(
		calendar,
		temporalZonedDateTimeLike,
		[]string{"year", "month", "monthCode", "day"},
		[]string{"hour", "minute", "second", "millisecond", "microsecond", "nanosecond", "offset"},
		"partial",
	)
	fields = calendarMergeFields(calendar, fields, partialZonedDateTime)

	resolvedOptions := getOptionsObject(options)
	disambiguation := getTemporalDisambiguationOption(resolvedOptions)
	offset := getTemporalOffsetOption(resolvedOptions, "prefer")
	overflow := getTemporalOverflowOption(resolvedOptions)

	newDateTime := interpretTemporalDateTimeFields(calendar, fields, overflow)
	newOffsetNs := parseDateTimeUTCOffset(*fields.offset)
	epochNanoseconds := interpretISODateTimeOffset(
		newDateTime.isoDate,
		newDateTime.time,
		"option",
		newOffsetNs,
		timeZone,
		disambiguation,
		offset,
		/* matchMinute = */ false,
	)

	return createTemporalZonedDateTime(epochNanoseconds, timeZone, calendar)
}

// WithPlainTime is Temporal.ZonedDateTime.prototype.withPlainTime.
func (z *ZonedDateTime) WithPlainTime(temporalTimeParam any) *ZonedDateTime {
	timeZone := z.timeZone
	calendar := z.calendar
	iso := z.dateTime().isoDate

	var epochNs *big.Int
	if isUndefined(temporalTimeParam) {
		epochNs = getStartOfDay(timeZone, iso)
	} else {
		temporalTime := toTemporalTime(temporalTimeParam)
		dt := combineISODateAndTimeRecord(iso, temporalTime.time)
		epochNs = getEpochNanosecondsFor(timeZone, dt, "compatible")
	}
	return createTemporalZonedDateTime(epochNs, timeZone, calendar)
}

// WithTimeZone is Temporal.ZonedDateTime.prototype.withTimeZone.
func (z *ZonedDateTime) WithTimeZone(timeZoneParam any) *ZonedDateTime {
	timeZone := toTemporalTimeZoneIdentifier(timeZoneParam)
	return createTemporalZonedDateTime(z.epochNs, timeZone, z.calendar)
}

// WithCalendar is Temporal.ZonedDateTime.prototype.withCalendar.
func (z *ZonedDateTime) WithCalendar(calendarParam any) *ZonedDateTime {
	calendar := toTemporalCalendarIdentifier(calendarParam)
	return createTemporalZonedDateTime(z.epochNs, z.timeZone, calendar)
}

// Add is Temporal.ZonedDateTime.prototype.add.
func (z *ZonedDateTime) Add(temporalDurationLike, options any) *ZonedDateTime {
	return addDurationToZonedDateTime("add", z, temporalDurationLike, options)
}

// Subtract is Temporal.ZonedDateTime.prototype.subtract.
func (z *ZonedDateTime) Subtract(temporalDurationLike, options any) *ZonedDateTime {
	return addDurationToZonedDateTime("subtract", z, temporalDurationLike, options)
}

// Until is Temporal.ZonedDateTime.prototype.until.
func (z *ZonedDateTime) Until(other, options any) *Duration {
	return differenceTemporalZonedDateTime("until", z, other, options)
}

// Since is Temporal.ZonedDateTime.prototype.since.
func (z *ZonedDateTime) Since(other, options any) *Duration {
	return differenceTemporalZonedDateTime("since", z, other, options)
}

// Round is Temporal.ZonedDateTime.prototype.round.
func (z *ZonedDateTime) Round(roundToParam any) *ZonedDateTime {
	if isUndefined(roundToParam) {
		throwType("options parameter is required")
	}
	roundTo := oneProp("smallestUnit", roundToParam)
	roundingIncrement := getRoundingIncrementOption(roundTo)
	roundingMode := getRoundingModeOption(roundTo, "halfExpand")
	smallestUnit := *getTemporalUnitValuedOption(roundTo, "smallestUnit", requiredSentinel)
	validateTemporalUnitValue(smallestUnit, "time", []string{"day"})
	maximumIncrements := map[string]float64{
		"day":         1,
		"hour":        24,
		"minute":      60,
		"second":      60,
		"millisecond": 1000,
		"microsecond": 1000,
		"nanosecond":  1000,
	}
	maximum := maximumIncrements[smallestUnit]
	inclusive := maximum == 1
	validateTemporalRoundingIncrement(roundingIncrement, maximum, inclusive)

	if smallestUnit == "nanosecond" && roundingIncrement == 1 {
		return createTemporalZonedDateTime(z.epochNs, z.timeZone, z.calendar)
	}

	// first, round the underlying DateTime fields
	timeZone := z.timeZone
	thisNs := z.epochNs
	iso := z.dateTime()
	var epochNanoseconds *big.Int

	if smallestUnit == "day" {
		// Compute Instants for start-of-day and end-of-day
		// Determine how far the current instant has progressed through this span.
		dateStart := iso.isoDate
		dateEnd := addDaysToISODate(dateStart, 1)

		startNs := getStartOfDay(timeZone, dateStart)
		if thisNs.Cmp(startNs) < 0 {
			panic(&Error{Name: "Error", Message: "assertion failure: cannot produce an instant during a day that occurs before start-of-day instant"})
		}

		endNs := getStartOfDay(timeZone, dateEnd)
		if thisNs.Cmp(endNs) >= 0 {
			panic(&Error{Name: "Error", Message: "assertion failure: cannot produce an instant during a day that occurs on or after end-of-day instant"})
		}

		dayLengthNs := new(big.Int).Sub(endNs, startNs)
		dayProgressNs := timeDurationFromEpochNsDiff(thisNs, startNs)
		roundedDayNs := dayProgressNs.round(dayLengthNs, roundingMode)
		epochNanoseconds = roundedDayNs.addToEpochNs(startNs)
	} else {
		// smallestUnit < day
		// Round based on ISO-calendar time units
		roundedDateTime := roundISODateTime(iso, roundingIncrement, smallestUnit, roundingMode)

		// Now reset all DateTime fields but leave the TimeZone. The offset will
		// also be retained if the new date/time values are still OK with the old
		// offset. Otherwise the offset will be changed to be compatible with the
		// new date/time values. If DST disambiguation is required, the `compatible`
		// disambiguation algorithm will be used.
		offsetNs := getOffsetNanosecondsFor(timeZone, thisNs)
		epochNanoseconds = interpretISODateTimeOffset(
			roundedDateTime.isoDate,
			roundedDateTime.time,
			"option",
			offsetNs,
			timeZone,
			"compatible",
			"prefer",
			/* matchMinute = */ false,
		)
	}

	return createTemporalZonedDateTime(epochNanoseconds, timeZone, z.calendar)
}

// Equals is Temporal.ZonedDateTime.prototype.equals.
func (z *ZonedDateTime) Equals(otherParam any) bool {
	other := toTemporalZonedDateTime(otherParam)
	if z.epochNs.Cmp(other.epochNs) != 0 {
		return false
	}
	if !timeZoneEquals(z.timeZone, other.timeZone) {
		return false
	}
	return calendarEquals(z.calendar, other.calendar)
}

// ToString is Temporal.ZonedDateTime.prototype.toString.
func (z *ZonedDateTime) ToString(options any) string {
	resolvedOptions := getOptionsObject(options)
	showCalendar := getTemporalShowCalendarNameOption(resolvedOptions)
	digits := getTemporalFractionalSecondDigitsOption(resolvedOptions)
	showOffset := getTemporalShowOffsetOption(resolvedOptions)
	roundingMode := getRoundingModeOption(resolvedOptions, "trunc")
	smallestUnit := getTemporalUnitValuedOption(resolvedOptions, "smallestUnit")
	showTimeZone := getTemporalShowTimeZoneNameOption(resolvedOptions)
	validateTemporalUnitValue(smallestUnit, "time")
	if smallestUnit != nil && *smallestUnit == "hour" {
		throwRange(`smallestUnit must be a time unit other than "hour"`)
	}
	rec := toSecondsStringPrecisionRecord(smallestUnit, digits)
	return temporalZonedDateTimeToString(z, rec.precision, showCalendar, showTimeZone, showOffset, &toStringOptions{
		unit:         rec.unit,
		increment:    rec.increment,
		roundingMode: roundingMode,
	})
}

// ToLocaleString is Temporal.ZonedDateTime.prototype.toLocaleString.
func (z *ZonedDateTime) ToLocaleString(locales, options any) string {
	resolvedOptions := getOptionsObject(options)

	// This is not quite per specification, but this polyfill's DateTimeFormat
	// already doesn't match the InitializeDateTimeFormat operation, and the
	// access order might change anyway;
	// see https://github.com/tc39/ecma402/issues/747
	optionsCopy := newJSOptions()
	copyOwnOptions(optionsCopy, resolvedOptions, "timeZone")

	if !isUndefined(objGet(resolvedOptions, "timeZone")) {
		throwType("ZonedDateTime toLocaleString does not accept a timeZone option")
	}

	if isUndefined(optionsCopy.get("year")) &&
		isUndefined(optionsCopy.get("month")) &&
		isUndefined(optionsCopy.get("day")) &&
		isUndefined(optionsCopy.get("era")) &&
		isUndefined(optionsCopy.get("weekday")) &&
		isUndefined(optionsCopy.get("dateStyle")) &&
		isUndefined(optionsCopy.get("hour")) &&
		isUndefined(optionsCopy.get("minute")) &&
		isUndefined(optionsCopy.get("second")) &&
		isUndefined(optionsCopy.get("fractionalSecondDigits")) &&
		isUndefined(optionsCopy.get("timeStyle")) &&
		isUndefined(optionsCopy.get("dayPeriod")) &&
		isUndefined(optionsCopy.get("timeZoneName")) {
		optionsCopy.set("timeZoneName", "short")
		// The rest of the defaults will be filled in by formatting the Instant
	}

	optionsCopy.set("timeZone", z.timeZone)

	formatter := NewDateTimeFormat(locales, optionsCopy)

	localeCalendarIdentifier := formatter.ResolvedOptions().Calendar
	calendarIdentifier := z.calendar
	if calendarIdentifier != "iso8601" &&
		localeCalendarIdentifier != "iso8601" &&
		!calendarEquals(localeCalendarIdentifier, calendarIdentifier) {
		throwRange("cannot format ZonedDateTime with calendar " + calendarIdentifier +
			" in locale with calendar " + localeCalendarIdentifier)
	}

	return formatter.Format(createTemporalInstant(z.epochNs))
}

// ToJSON is Temporal.ZonedDateTime.prototype.toJSON.
func (z *ZonedDateTime) ToJSON() string {
	return temporalZonedDateTimeToString(z, "auto")
}

func (z *ZonedDateTime) jsValueOf() { valueOfThrows("ZonedDateTime") }

func (z *ZonedDateTime) jsToString() string { return z.ToString(nil) }

// StartOfDay is Temporal.ZonedDateTime.prototype.startOfDay.
func (z *ZonedDateTime) StartOfDay() *ZonedDateTime {
	timeZone := z.timeZone
	isoDate := z.dateTime().isoDate
	epochNanoseconds := getStartOfDay(timeZone, isoDate)
	return createTemporalZonedDateTime(epochNanoseconds, timeZone, z.calendar)
}

// GetTimeZoneTransition is Temporal.ZonedDateTime.prototype.getTimeZoneTransition. A nil pointer is
// null: the time zone has no such transition.
func (z *ZonedDateTime) GetTimeZoneTransition(directionParam any) *ZonedDateTime {
	timeZone := z.timeZone

	if isUndefined(directionParam) {
		throwType("options parameter is required")
	}
	direction := getDirectionOption(oneProp("direction", directionParam))

	// Offset time zones or UTC have no transitions
	if isOffsetTimeZoneIdentifier(timeZone) || timeZone == "UTC" {
		return nil
	}

	thisEpochNanoseconds := z.epochNs
	var epochNanoseconds *big.Int
	if direction == "next" {
		epochNanoseconds = getNamedTimeZoneNextTransition(timeZone, thisEpochNanoseconds)
	} else {
		epochNanoseconds = getNamedTimeZonePreviousTransition(timeZone, thisEpochNanoseconds)
	}
	if epochNanoseconds == nil {
		return nil
	}
	return createTemporalZonedDateTime(epochNanoseconds, timeZone, z.calendar)
}

// ToInstant is Temporal.ZonedDateTime.prototype.toInstant.
func (z *ZonedDateTime) ToInstant() *Instant { return createTemporalInstant(z.epochNs) }

// ToPlainDate is Temporal.ZonedDateTime.prototype.toPlainDate.
func (z *ZonedDateTime) ToPlainDate() *PlainDate {
	return createTemporalDate(z.dateTime().isoDate, z.calendar)
}

// ToPlainTime is Temporal.ZonedDateTime.prototype.toPlainTime.
func (z *ZonedDateTime) ToPlainTime() *PlainTime { return createTemporalTime(z.dateTime().time) }

// ToPlainDateTime is Temporal.ZonedDateTime.prototype.toPlainDateTime.
func (z *ZonedDateTime) ToPlainDateTime() *PlainDateTime {
	return createTemporalDateTime(z.dateTime(), z.calendar)
}

// ZonedDateTimeFrom is Temporal.ZonedDateTime.from.
func ZonedDateTimeFrom(item, optionsParam any) *ZonedDateTime {
	return toTemporalZonedDateTime(item, optionsParam)
}

// ZonedDateTimeCompare is Temporal.ZonedDateTime.compare.
func ZonedDateTimeCompare(oneParam, twoParam any) float64 {
	one := toTemporalZonedDateTime(oneParam)
	two := toTemporalZonedDateTime(twoParam)
	return float64(one.epochNs.Cmp(two.epochNs))
}

func (z *ZonedDateTime) jsGet(key string) any {
	switch key {
	case "calendarId":
		return z.CalendarId()
	case "timeZoneId":
		return z.TimeZoneId()
	case "year":
		return z.Year()
	case "month":
		return z.Month()
	case "monthCode":
		return z.MonthCode()
	case "day":
		return z.Day()
	case "hour":
		return z.Hour()
	case "minute":
		return z.Minute()
	case "second":
		return z.Second()
	case "millisecond":
		return z.Millisecond()
	case "microsecond":
		return z.Microsecond()
	case "nanosecond":
		return z.Nanosecond()
	case "era":
		return stringOrUndefined(z.Era())
	case "eraYear":
		return numberOrUndefined(z.EraYear())
	case "epochMilliseconds":
		return z.EpochMilliseconds()
	case "epochNanoseconds":
		return z.EpochNanoseconds()
	case "dayOfWeek":
		return z.DayOfWeek()
	case "dayOfYear":
		return z.DayOfYear()
	case "weekOfYear":
		return numberOrUndefined(z.WeekOfYear())
	case "yearOfWeek":
		return numberOrUndefined(z.YearOfWeek())
	case "hoursInDay":
		return z.HoursInDay()
	case "daysInWeek":
		return z.DaysInWeek()
	case "daysInMonth":
		return z.DaysInMonth()
	case "daysInYear":
		return z.DaysInYear()
	case "monthsInYear":
		return z.MonthsInYear()
	case "inLeapYear":
		return z.InLeapYear()
	case "offset":
		return z.Offset()
	case "offsetNanoseconds":
		return z.OffsetNanoseconds()
	}
	return nil
}
