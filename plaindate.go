package temporal

import "math/big"

// NewPlainDate is new Temporal.PlainDate(isoYear, isoMonth, isoDay, calendar). calendar defaults to
// "iso8601".
func NewPlainDate(isoYear, isoMonth, isoDay, calendarParam any) *PlainDate {
	year := toIntegerWithTruncation(isoYear)
	month := toIntegerWithTruncation(isoMonth)
	day := toIntegerWithTruncation(isoDay)
	var calendar string
	if isUndefined(calendarParam) {
		calendar = canonicalizeCalendar("iso8601")
	} else {
		calendar = canonicalizeCalendar(requireString(calendarParam))
	}
	rejectISODate(year, month, day)
	return createTemporalDate(isoDate{year: year, month: month, day: day}, calendar)
}

// CalendarId is the calendarId getter.
func (d *PlainDate) CalendarId() string { return d.calendar }

// Era is the era getter.
func (d *PlainDate) Era() *string {
	return calendarImplForID(d.calendar).isoToDate(d.isoDate, reqEra).era
}

// EraYear is the eraYear getter.
func (d *PlainDate) EraYear() *float64 {
	return calendarImplForID(d.calendar).isoToDate(d.isoDate, reqEraYear).eraYear
}

// Year is the year getter.
func (d *PlainDate) Year() float64 {
	return calendarImplForID(d.calendar).isoToDate(d.isoDate, reqYear).year
}

// Month is the month getter.
func (d *PlainDate) Month() float64 {
	return calendarImplForID(d.calendar).isoToDate(d.isoDate, reqMonth).month
}

// MonthCode is the monthCode getter.
func (d *PlainDate) MonthCode() string {
	return calendarImplForID(d.calendar).isoToDate(d.isoDate, reqMonthCode).monthCode
}

// Day is the day getter.
func (d *PlainDate) Day() float64 {
	return calendarImplForID(d.calendar).isoToDate(d.isoDate, reqDay).day
}

// DayOfWeek is the dayOfWeek getter.
func (d *PlainDate) DayOfWeek() float64 {
	return calendarImplForID(d.calendar).isoToDate(d.isoDate, reqDayOfWeek).dayOfWeek
}

// DayOfYear is the dayOfYear getter.
func (d *PlainDate) DayOfYear() float64 {
	return calendarImplForID(d.calendar).isoToDate(d.isoDate, reqDayOfYear).dayOfYear
}

// WeekOfYear is the weekOfYear getter.
func (d *PlainDate) WeekOfYear() *float64 {
	return calendarImplForID(d.calendar).isoToDate(d.isoDate, reqWeekOfYear).weekOfYear
}

// YearOfWeek is the yearOfWeek getter.
func (d *PlainDate) YearOfWeek() *float64 {
	return calendarImplForID(d.calendar).isoToDate(d.isoDate, reqWeekOfYear).yearOfWeek
}

// DaysInWeek is the daysInWeek getter.
func (d *PlainDate) DaysInWeek() float64 {
	return calendarImplForID(d.calendar).isoToDate(d.isoDate, reqDaysInWeek).daysInWeek
}

// DaysInMonth is the daysInMonth getter.
func (d *PlainDate) DaysInMonth() float64 {
	return calendarImplForID(d.calendar).isoToDate(d.isoDate, reqDaysInMonth).daysInMonth
}

// DaysInYear is the daysInYear getter.
func (d *PlainDate) DaysInYear() float64 {
	return calendarImplForID(d.calendar).isoToDate(d.isoDate, reqDaysInYear).daysInYear
}

// MonthsInYear is the monthsInYear getter.
func (d *PlainDate) MonthsInYear() float64 {
	return calendarImplForID(d.calendar).isoToDate(d.isoDate, reqMonthsInYear).monthsInYear
}

// InLeapYear is the inLeapYear getter.
func (d *PlainDate) InLeapYear() bool {
	return calendarImplForID(d.calendar).isoToDate(d.isoDate, reqInLeapYear).inLeapYear
}

// With is Temporal.PlainDate.prototype.with.
func (d *PlainDate) With(temporalDateLike, options any) *PlainDate {
	if !isObject(temporalDateLike) {
		throwType("invalid argument")
	}
	rejectTemporalLikeObject(temporalDateLike)

	calendar := d.calendar
	fields := isoDateToFields(calendar, d.isoDate, "date")
	partialDate := prepareCalendarFields(calendar, temporalDateLike, []string{"year", "month", "monthCode", "day"}, []string{}, "partial")
	fields = calendarMergeFields(calendar, fields, partialDate)

	overflow := getTemporalOverflowOption(getOptionsObject(options))
	isoDate := calendarDateFromFields(calendar, fields, overflow)
	return createTemporalDate(isoDate, calendar)
}

// WithCalendar is Temporal.PlainDate.prototype.withCalendar.
func (d *PlainDate) WithCalendar(calendarParam any) *PlainDate {
	calendar := toTemporalCalendarIdentifier(calendarParam)
	return createTemporalDate(d.isoDate, calendar)
}

// Add is Temporal.PlainDate.prototype.add.
func (d *PlainDate) Add(temporalDurationLike, options any) *PlainDate {
	return addDurationToDate("add", d, temporalDurationLike, options)
}

// Subtract is Temporal.PlainDate.prototype.subtract.
func (d *PlainDate) Subtract(temporalDurationLike, options any) *PlainDate {
	return addDurationToDate("subtract", d, temporalDurationLike, options)
}

// Until is Temporal.PlainDate.prototype.until.
func (d *PlainDate) Until(other, options any) *Duration {
	return differenceTemporalPlainDate("until", d, other, options)
}

// Since is Temporal.PlainDate.prototype.since.
func (d *PlainDate) Since(other, options any) *Duration {
	return differenceTemporalPlainDate("since", d, other, options)
}

// Equals is Temporal.PlainDate.prototype.equals.
func (d *PlainDate) Equals(otherParam any) bool {
	other := toTemporalDate(otherParam)
	if compareISODate(d.isoDate, other.isoDate) != 0 {
		return false
	}
	return calendarEquals(d.calendar, other.calendar)
}

// ToString is Temporal.PlainDate.prototype.toString.
func (d *PlainDate) ToString(options any) string {
	resolvedOptions := getOptionsObject(options)
	showCalendar := getTemporalShowCalendarNameOption(resolvedOptions)
	return temporalDateToString(d, showCalendar)
}

// ToJSON is Temporal.PlainDate.prototype.toJSON.
func (d *PlainDate) ToJSON() string { return temporalDateToString(d, "auto") }

// ToLocaleString is Temporal.PlainDate.prototype.toLocaleString.
func (d *PlainDate) ToLocaleString(locales, options any) string {
	return createDateTimeFormat(locales, options, "date").Format(d)
}

func (d *PlainDate) jsValueOf() { valueOfThrows("PlainDate") }

func (d *PlainDate) jsToString() string { return d.ToString(nil) }

// ToPlainDateTime is Temporal.PlainDate.prototype.toPlainDateTime.
func (d *PlainDate) ToPlainDateTime(temporalTime any) *PlainDateTime {
	time := toTimeRecordOrMidnight(temporalTime)
	isoDateTime := combineISODateAndTimeRecord(d.isoDate, time)
	return createTemporalDateTime(isoDateTime, d.calendar)
}

// ToZonedDateTime is Temporal.PlainDate.prototype.toZonedDateTime. item is a time zone identifier or
// a ZonedDateTimeConversion.
func (d *PlainDate) ToZonedDateTime(item any) *ZonedDateTime {
	item = unwrap(item)
	var timeZone string
	var temporalTime any
	if isObject(item) {
		timeZoneLike := objGet(item, "timeZone")
		if isUndefined(timeZoneLike) {
			timeZone = toTemporalTimeZoneIdentifier(item)
		} else {
			timeZone = toTemporalTimeZoneIdentifier(timeZoneLike)
			temporalTime = objGet(item, "plainTime")
		}
	} else {
		timeZone = toTemporalTimeZoneIdentifier(item)
	}

	iso := d.isoDate
	var epochNs *big.Int
	if isUndefined(temporalTime) {
		epochNs = getStartOfDay(timeZone, iso)
	} else {
		t := toTemporalTime(temporalTime)
		isoDateTime := combineISODateAndTimeRecord(iso, t.time)
		epochNs = getEpochNanosecondsFor(timeZone, isoDateTime, "compatible")
	}
	return createTemporalZonedDateTime(epochNs, timeZone, d.calendar)
}

// ToPlainYearMonth is Temporal.PlainDate.prototype.toPlainYearMonth.
func (d *PlainDate) ToPlainYearMonth() *PlainYearMonth {
	calendar := d.calendar
	fields := isoDateToFields(calendar, d.isoDate, "date")
	isoDate := calendarYearMonthFromFields(calendar, fields, "constrain")
	return createTemporalYearMonth(isoDate, calendar)
}

// ToPlainMonthDay is Temporal.PlainDate.prototype.toPlainMonthDay.
func (d *PlainDate) ToPlainMonthDay() *PlainMonthDay {
	calendar := d.calendar
	fields := isoDateToFields(calendar, d.isoDate, "date")
	isoDate := calendarMonthDayFromFields(calendar, fields, "constrain")
	return createTemporalMonthDay(isoDate, calendar)
}

// PlainDateFrom is Temporal.PlainDate.from.
func PlainDateFrom(item, options any) *PlainDate { return toTemporalDate(item, options) }

// PlainDateCompare is Temporal.PlainDate.compare.
func PlainDateCompare(oneParam, twoParam any) float64 {
	one := toTemporalDate(oneParam)
	two := toTemporalDate(twoParam)
	return compareISODate(one.isoDate, two.isoDate)
}

func (d *PlainDate) jsGet(key string) any {
	switch key {
	case "calendarId":
		return d.CalendarId()
	case "era":
		return stringOrUndefined(d.Era())
	case "eraYear":
		return numberOrUndefined(d.EraYear())
	case "year":
		return d.Year()
	case "month":
		return d.Month()
	case "monthCode":
		return d.MonthCode()
	case "day":
		return d.Day()
	case "dayOfWeek":
		return d.DayOfWeek()
	case "dayOfYear":
		return d.DayOfYear()
	case "weekOfYear":
		return numberOrUndefined(d.WeekOfYear())
	case "yearOfWeek":
		return numberOrUndefined(d.YearOfWeek())
	case "daysInWeek":
		return d.DaysInWeek()
	case "daysInMonth":
		return d.DaysInMonth()
	case "daysInYear":
		return d.DaysInYear()
	case "monthsInYear":
		return d.MonthsInYear()
	case "inLeapYear":
		return d.InLeapYear()
	}
	return nil
}
