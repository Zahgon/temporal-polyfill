package temporal

// NewPlainMonthDay is new Temporal.PlainMonthDay(isoMonth, isoDay, calendar, referenceISOYear).
// calendar defaults to "iso8601" and referenceISOYear to 1972.
func NewPlainMonthDay(isoMonth, isoDay, calendarParam, referenceISOYear any) *PlainMonthDay {
	month := toIntegerWithTruncation(isoMonth)
	day := toIntegerWithTruncation(isoDay)
	calendar := canonicalizeCalendar(requireString(defaultTo(calendarParam, "iso8601")))
	year := toIntegerWithTruncation(defaultTo(referenceISOYear, 1972.0))

	rejectISODate(year, month, day)
	return createTemporalMonthDay(isoDate{year: year, month: month, day: day}, calendar)
}

// MonthCode is the monthCode getter.
func (md *PlainMonthDay) MonthCode() string {
	return calendarImplForID(md.calendar).isoToDate(md.isoDate, reqMonthCode).monthCode
}

// Day is the day getter.
func (md *PlainMonthDay) Day() float64 {
	return calendarImplForID(md.calendar).isoToDate(md.isoDate, reqDay).day
}

// CalendarId is the calendarId getter.
func (md *PlainMonthDay) CalendarId() string { return md.calendar }

// With is Temporal.PlainMonthDay.prototype.with.
func (md *PlainMonthDay) With(temporalMonthDayLike, options any) *PlainMonthDay {
	if !isObject(temporalMonthDayLike) {
		throwType("invalid argument")
	}
	rejectTemporalLikeObject(temporalMonthDayLike)

	calendar := md.calendar
	fields := isoDateToFields(calendar, md.isoDate, "month-day")
	partialMonthDay := prepareCalendarFields(calendar, temporalMonthDayLike, []string{"year", "month", "monthCode", "day"}, []string{}, "partial")
	fields = calendarMergeFields(calendar, fields, partialMonthDay)

	overflow := getTemporalOverflowOption(getOptionsObject(options))
	isoDate := calendarMonthDayFromFields(calendar, fields, overflow)
	return createTemporalMonthDay(isoDate, calendar)
}

// Equals is Temporal.PlainMonthDay.prototype.equals.
func (md *PlainMonthDay) Equals(otherParam any) bool {
	other := toTemporalMonthDay(otherParam)
	if compareISODate(md.isoDate, other.isoDate) != 0 {
		return false
	}
	return calendarEquals(md.calendar, other.calendar)
}

// ToString is Temporal.PlainMonthDay.prototype.toString.
func (md *PlainMonthDay) ToString(options any) string {
	resolvedOptions := getOptionsObject(options)
	showCalendar := getTemporalShowCalendarNameOption(resolvedOptions)
	return temporalMonthDayToString(md, showCalendar)
}

// ToJSON is Temporal.PlainMonthDay.prototype.toJSON.
func (md *PlainMonthDay) ToJSON() string { return temporalMonthDayToString(md, "auto") }

// ToLocaleString is Temporal.PlainMonthDay.prototype.toLocaleString.
func (md *PlainMonthDay) ToLocaleString(locales, options any) string {
	return createDateTimeFormat(locales, options, "date").Format(md)
}

func (md *PlainMonthDay) jsValueOf() { valueOfThrows("PlainMonthDay") }

func (md *PlainMonthDay) jsToString() string { return md.ToString(nil) }

// ToPlainDate is Temporal.PlainMonthDay.prototype.toPlainDate.
func (md *PlainMonthDay) ToPlainDate(item any) *PlainDate {
	if !isObject(item) {
		throwType("argument should be an object")
	}
	calendar := md.calendar

	fields := isoDateToFields(calendar, md.isoDate, "month-day")
	inputFields := prepareCalendarFields(calendar, item, []string{"year"}, []string{}, []string{})
	mergedFields := calendarMergeFields(calendar, fields, inputFields)
	isoDate := calendarDateFromFields(calendar, mergedFields, "constrain")
	return createTemporalDate(isoDate, calendar)
}

// PlainMonthDayFrom is Temporal.PlainMonthDay.from.
func PlainMonthDayFrom(item, options any) *PlainMonthDay { return toTemporalMonthDay(item, options) }

func (md *PlainMonthDay) jsGet(key string) any {
	switch key {
	case "monthCode":
		return md.MonthCode()
	case "day":
		return md.Day()
	case "calendarId":
		return md.CalendarId()
	}
	return nil
}
