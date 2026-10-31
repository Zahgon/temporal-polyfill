package temporal

// NewPlainDateTime is new Temporal.PlainDateTime(...). The time fields default to 0 and calendar to
// "iso8601".
func NewPlainDateTime(isoYear, isoMonth, isoDay, hourParam, minuteParam, secondParam, millisecondParam, microsecondParam, nanosecondParam, calendarParam any) *PlainDateTime {
	year := toIntegerWithTruncation(isoYear)
	month := toIntegerWithTruncation(isoMonth)
	day := toIntegerWithTruncation(isoDay)
	hour := toIntegerWithTruncation(defaultTo(hourParam, 0.0))
	minute := toIntegerWithTruncation(defaultTo(minuteParam, 0.0))
	second := toIntegerWithTruncation(defaultTo(secondParam, 0.0))
	millisecond := toIntegerWithTruncation(defaultTo(millisecondParam, 0.0))
	microsecond := toIntegerWithTruncation(defaultTo(microsecondParam, 0.0))
	nanosecond := toIntegerWithTruncation(defaultTo(nanosecondParam, 0.0))
	calendar := canonicalizeCalendar(requireString(defaultTo(calendarParam, "iso8601")))

	rejectDateTime(year, month, day, hour, minute, second, millisecond, microsecond, nanosecond)

	return createTemporalDateTime(
		isoDateTime{
			isoDate: isoDate{year: year, month: month, day: day},
			time:    timeRecord{hour: hour, minute: minute, second: second, millisecond: millisecond, microsecond: microsecond, nanosecond: nanosecond},
		},
		calendar,
	)
}

// CalendarId is the calendarId getter.
func (dt *PlainDateTime) CalendarId() string { return dt.calendar }

// Year is the year getter.
func (dt *PlainDateTime) Year() float64 {
	return calendarImplForID(dt.calendar).isoToDate(dt.isoDateTime.isoDate, reqYear).year
}

// Month is the month getter.
func (dt *PlainDateTime) Month() float64 {
	return calendarImplForID(dt.calendar).isoToDate(dt.isoDateTime.isoDate, reqMonth).month
}

// MonthCode is the monthCode getter.
func (dt *PlainDateTime) MonthCode() string {
	return calendarImplForID(dt.calendar).isoToDate(dt.isoDateTime.isoDate, reqMonthCode).monthCode
}

// Day is the day getter.
func (dt *PlainDateTime) Day() float64 {
	return calendarImplForID(dt.calendar).isoToDate(dt.isoDateTime.isoDate, reqDay).day
}

// Hour is the hour getter.
func (dt *PlainDateTime) Hour() float64 { return dt.isoDateTime.time.hour }

// Minute is the minute getter.
func (dt *PlainDateTime) Minute() float64 { return dt.isoDateTime.time.minute }

// Second is the second getter.
func (dt *PlainDateTime) Second() float64 { return dt.isoDateTime.time.second }

// Millisecond is the millisecond getter.
func (dt *PlainDateTime) Millisecond() float64 { return dt.isoDateTime.time.millisecond }

// Microsecond is the microsecond getter.
func (dt *PlainDateTime) Microsecond() float64 { return dt.isoDateTime.time.microsecond }

// Nanosecond is the nanosecond getter.
func (dt *PlainDateTime) Nanosecond() float64 { return dt.isoDateTime.time.nanosecond }

// Era is the era getter.
func (dt *PlainDateTime) Era() *string {
	return calendarImplForID(dt.calendar).isoToDate(dt.isoDateTime.isoDate, reqEra).era
}

// EraYear is the eraYear getter.
func (dt *PlainDateTime) EraYear() *float64 {
	return calendarImplForID(dt.calendar).isoToDate(dt.isoDateTime.isoDate, reqEraYear).eraYear
}

// DayOfWeek is the dayOfWeek getter.
func (dt *PlainDateTime) DayOfWeek() float64 {
	return calendarImplForID(dt.calendar).isoToDate(dt.isoDateTime.isoDate, reqDayOfWeek).dayOfWeek
}

// DayOfYear is the dayOfYear getter.
func (dt *PlainDateTime) DayOfYear() float64 {
	return calendarImplForID(dt.calendar).isoToDate(dt.isoDateTime.isoDate, reqDayOfYear).dayOfYear
}

// WeekOfYear is the weekOfYear getter.
func (dt *PlainDateTime) WeekOfYear() *float64 {
	return calendarImplForID(dt.calendar).isoToDate(dt.isoDateTime.isoDate, reqWeekOfYear).weekOfYear
}

// YearOfWeek is the yearOfWeek getter.
func (dt *PlainDateTime) YearOfWeek() *float64 {
	return calendarImplForID(dt.calendar).isoToDate(dt.isoDateTime.isoDate, reqWeekOfYear).yearOfWeek
}

// DaysInWeek is the daysInWeek getter.
func (dt *PlainDateTime) DaysInWeek() float64 {
	return calendarImplForID(dt.calendar).isoToDate(dt.isoDateTime.isoDate, reqDaysInWeek).daysInWeek
}

// DaysInYear is the daysInYear getter.
func (dt *PlainDateTime) DaysInYear() float64 {
	return calendarImplForID(dt.calendar).isoToDate(dt.isoDateTime.isoDate, reqDaysInYear).daysInYear
}

// DaysInMonth is the daysInMonth getter.
func (dt *PlainDateTime) DaysInMonth() float64 {
	return calendarImplForID(dt.calendar).isoToDate(dt.isoDateTime.isoDate, reqDaysInMonth).daysInMonth
}

// MonthsInYear is the monthsInYear getter.
func (dt *PlainDateTime) MonthsInYear() float64 {
	return calendarImplForID(dt.calendar).isoToDate(dt.isoDateTime.isoDate, reqMonthsInYear).monthsInYear
}

// InLeapYear is the inLeapYear getter.
func (dt *PlainDateTime) InLeapYear() bool {
	return calendarImplForID(dt.calendar).isoToDate(dt.isoDateTime.isoDate, reqInLeapYear).inLeapYear
}

// With is Temporal.PlainDateTime.prototype.with.
func (dt *PlainDateTime) With(temporalDateTimeLike, options any) *PlainDateTime {
	if !isObject(temporalDateTimeLike) {
		throwType("invalid argument")
	}
	rejectTemporalLikeObject(temporalDateTimeLike)

	calendar := dt.calendar
	isoDateTime := dt.isoDateTime
	fields := isoDateToFields(calendar, isoDateTime.isoDate, "date")
	spreadTimeIntoFields(fields, isoDateTime.time)
	partialDateTime := prepareCalendarFields(
		calendar,
		temporalDateTimeLike,
		[]string{"year", "month", "monthCode", "day"},
		[]string{"hour", "minute", "second", "millisecond", "microsecond", "nanosecond"},
		"partial",
	)
	fields = calendarMergeFields(calendar, fields, partialDateTime)

	overflow := getTemporalOverflowOption(getOptionsObject(options))
	newDateTime := interpretTemporalDateTimeFields(calendar, fields, overflow)
	return createTemporalDateTime(newDateTime, calendar)
}

// WithPlainTime is Temporal.PlainDateTime.prototype.withPlainTime.
func (dt *PlainDateTime) WithPlainTime(temporalTime any) *PlainDateTime {
	time := toTimeRecordOrMidnight(temporalTime)
	isoDateTime := combineISODateAndTimeRecord(dt.isoDateTime.isoDate, time)
	return createTemporalDateTime(isoDateTime, dt.calendar)
}

// WithCalendar is Temporal.PlainDateTime.prototype.withCalendar.
func (dt *PlainDateTime) WithCalendar(calendarParam any) *PlainDateTime {
	calendar := toTemporalCalendarIdentifier(calendarParam)
	return createTemporalDateTime(combineISODateAndTimeRecord(dt.isoDateTime.isoDate, dt.isoDateTime.time), calendar)
}

// Add is Temporal.PlainDateTime.prototype.add.
func (dt *PlainDateTime) Add(temporalDurationLike, options any) *PlainDateTime {
	return addDurationToDateTime("add", dt, temporalDurationLike, options)
}

// Subtract is Temporal.PlainDateTime.prototype.subtract.
func (dt *PlainDateTime) Subtract(temporalDurationLike, options any) *PlainDateTime {
	return addDurationToDateTime("subtract", dt, temporalDurationLike, options)
}

// Until is Temporal.PlainDateTime.prototype.until.
func (dt *PlainDateTime) Until(other, options any) *Duration {
	return differenceTemporalPlainDateTime("until", dt, other, options)
}

// Since is Temporal.PlainDateTime.prototype.since.
func (dt *PlainDateTime) Since(other, options any) *Duration {
	return differenceTemporalPlainDateTime("since", dt, other, options)
}

// Round is Temporal.PlainDateTime.prototype.round.
func (dt *PlainDateTime) Round(roundToParam any) *PlainDateTime {
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

	isoDateTime := dt.isoDateTime
	if roundingIncrement == 1 && smallestUnit == "nanosecond" {
		return createTemporalDateTime(isoDateTime, dt.calendar)
	}
	result := roundISODateTime(isoDateTime, roundingIncrement, smallestUnit, roundingMode)

	return createTemporalDateTime(result, dt.calendar)
}

// Equals is Temporal.PlainDateTime.prototype.equals.
func (dt *PlainDateTime) Equals(otherParam any) bool {
	other := toTemporalDateTime(otherParam)
	if compareISODateTime(dt.isoDateTime, other.isoDateTime) != 0 {
		return false
	}
	return calendarEquals(dt.calendar, other.calendar)
}

// ToString is Temporal.PlainDateTime.prototype.toString.
func (dt *PlainDateTime) ToString(options any) string {
	resolvedOptions := getOptionsObject(options)
	showCalendar := getTemporalShowCalendarNameOption(resolvedOptions)
	digits := getTemporalFractionalSecondDigitsOption(resolvedOptions)
	roundingMode := getRoundingModeOption(resolvedOptions, "trunc")
	smallestUnit := getTemporalUnitValuedOption(resolvedOptions, "smallestUnit")
	validateTemporalUnitValue(smallestUnit, "time")
	if smallestUnit != nil && *smallestUnit == "hour" {
		throwRange(`smallestUnit must be a time unit other than "hour"`)
	}
	rec := toSecondsStringPrecisionRecord(smallestUnit, digits)
	result := roundISODateTime(dt.isoDateTime, rec.increment, rec.unit, roundingMode)
	rejectDateTimeRange(result)
	return isoDateTimeToString(result, dt.calendar, rec.precision, showCalendar)
}

// ToJSON is Temporal.PlainDateTime.prototype.toJSON.
func (dt *PlainDateTime) ToJSON() string {
	return isoDateTimeToString(dt.isoDateTime, dt.calendar, "auto", "auto")
}

// ToLocaleString is Temporal.PlainDateTime.prototype.toLocaleString.
func (dt *PlainDateTime) ToLocaleString(locales, options any) string {
	return NewDateTimeFormat(locales, options).Format(dt)
}

func (dt *PlainDateTime) jsValueOf() { valueOfThrows("PlainDateTime") }

func (dt *PlainDateTime) jsToString() string { return dt.ToString(nil) }

// ToZonedDateTime is Temporal.PlainDateTime.prototype.toZonedDateTime.
func (dt *PlainDateTime) ToZonedDateTime(temporalTimeZoneLike, options any) *ZonedDateTime {
	timeZone := toTemporalTimeZoneIdentifier(temporalTimeZoneLike)
	resolvedOptions := getOptionsObject(options)
	disambiguation := getTemporalDisambiguationOption(resolvedOptions)
	epochNs := getEpochNanosecondsFor(timeZone, dt.isoDateTime, disambiguation)
	return createTemporalZonedDateTime(epochNs, timeZone, dt.calendar)
}

// ToPlainDate is Temporal.PlainDateTime.prototype.toPlainDate.
func (dt *PlainDateTime) ToPlainDate() *PlainDate {
	return createTemporalDate(dt.isoDateTime.isoDate, dt.calendar)
}

// ToPlainTime is Temporal.PlainDateTime.prototype.toPlainTime.
func (dt *PlainDateTime) ToPlainTime() *PlainTime { return createTemporalTime(dt.isoDateTime.time) }

// PlainDateTimeFrom is Temporal.PlainDateTime.from.
func PlainDateTimeFrom(item, options any) *PlainDateTime { return toTemporalDateTime(item, options) }

// PlainDateTimeCompare is Temporal.PlainDateTime.compare.
func PlainDateTimeCompare(oneParam, twoParam any) float64 {
	one := toTemporalDateTime(oneParam)
	two := toTemporalDateTime(twoParam)
	return compareISODateTime(one.isoDateTime, two.isoDateTime)
}

func (dt *PlainDateTime) jsGet(key string) any {
	switch key {
	case "calendarId":
		return dt.CalendarId()
	case "year":
		return dt.Year()
	case "month":
		return dt.Month()
	case "monthCode":
		return dt.MonthCode()
	case "day":
		return dt.Day()
	case "hour":
		return dt.Hour()
	case "minute":
		return dt.Minute()
	case "second":
		return dt.Second()
	case "millisecond":
		return dt.Millisecond()
	case "microsecond":
		return dt.Microsecond()
	case "nanosecond":
		return dt.Nanosecond()
	case "era":
		return stringOrUndefined(dt.Era())
	case "eraYear":
		return numberOrUndefined(dt.EraYear())
	case "dayOfWeek":
		return dt.DayOfWeek()
	case "dayOfYear":
		return dt.DayOfYear()
	case "weekOfYear":
		return numberOrUndefined(dt.WeekOfYear())
	case "yearOfWeek":
		return numberOrUndefined(dt.YearOfWeek())
	case "daysInWeek":
		return dt.DaysInWeek()
	case "daysInYear":
		return dt.DaysInYear()
	case "daysInMonth":
		return dt.DaysInMonth()
	case "monthsInYear":
		return dt.MonthsInYear()
	case "inLeapYear":
		return dt.InLeapYear()
	}
	return nil
}
