package temporal

// NewPlainYearMonth is new Temporal.PlainYearMonth(isoYear, isoMonth, calendar, referenceISODay).
// calendar defaults to "iso8601" and referenceISODay to 1.
func NewPlainYearMonth(isoYear, isoMonth, calendarParam, referenceISODay any) *PlainYearMonth {
	year := toIntegerWithTruncation(isoYear)
	month := toIntegerWithTruncation(isoMonth)
	calendar := canonicalizeCalendar(requireString(defaultTo(calendarParam, "iso8601")))
	day := toIntegerWithTruncation(defaultTo(referenceISODay, 1.0))

	rejectISODate(year, month, day)
	return createTemporalYearMonth(isoDate{year: year, month: month, day: day}, calendar)
}

// Year is the year getter.
func (ym *PlainYearMonth) Year() float64 {
	return calendarImplForID(ym.calendar).isoToDate(ym.isoDate, reqYear).year
}

// Month is the month getter.
func (ym *PlainYearMonth) Month() float64 {
	return calendarImplForID(ym.calendar).isoToDate(ym.isoDate, reqMonth).month
}

// MonthCode is the monthCode getter.
func (ym *PlainYearMonth) MonthCode() string {
	return calendarImplForID(ym.calendar).isoToDate(ym.isoDate, reqMonthCode).monthCode
}

// CalendarId is the calendarId getter.
func (ym *PlainYearMonth) CalendarId() string { return ym.calendar }

// Era is the era getter.
func (ym *PlainYearMonth) Era() *string {
	return calendarImplForID(ym.calendar).isoToDate(ym.isoDate, reqEra).era
}

// EraYear is the eraYear getter.
func (ym *PlainYearMonth) EraYear() *float64 {
	return calendarImplForID(ym.calendar).isoToDate(ym.isoDate, reqEraYear).eraYear
}

// DaysInMonth is the daysInMonth getter.
func (ym *PlainYearMonth) DaysInMonth() float64 {
	return calendarImplForID(ym.calendar).isoToDate(ym.isoDate, reqDaysInMonth).daysInMonth
}

// DaysInYear is the daysInYear getter.
func (ym *PlainYearMonth) DaysInYear() float64 {
	return calendarImplForID(ym.calendar).isoToDate(ym.isoDate, reqDaysInYear).daysInYear
}

// MonthsInYear is the monthsInYear getter.
func (ym *PlainYearMonth) MonthsInYear() float64 {
	return calendarImplForID(ym.calendar).isoToDate(ym.isoDate, reqMonthsInYear).monthsInYear
}

// InLeapYear is the inLeapYear getter.
func (ym *PlainYearMonth) InLeapYear() bool {
	return calendarImplForID(ym.calendar).isoToDate(ym.isoDate, reqInLeapYear).inLeapYear
}

// With is Temporal.PlainYearMonth.prototype.with.
func (ym *PlainYearMonth) With(temporalYearMonthLike, options any) *PlainYearMonth {
	if !isObject(temporalYearMonthLike) {
		throwType("invalid argument")
	}
	rejectTemporalLikeObject(temporalYearMonthLike)

	calendar := ym.calendar
	fields := isoDateToFields(calendar, ym.isoDate, "year-month")
	partialYearMonth := prepareCalendarFields(calendar, temporalYearMonthLike, []string{"year", "month", "monthCode"}, []string{}, "partial")
	fields = calendarMergeFields(calendar, fields, partialYearMonth)

	overflow := getTemporalOverflowOption(getOptionsObject(options))
	isoDate := calendarYearMonthFromFields(calendar, fields, overflow)
	return createTemporalYearMonth(isoDate, calendar)
}

// Add is Temporal.PlainYearMonth.prototype.add.
func (ym *PlainYearMonth) Add(temporalDurationLike, options any) *PlainYearMonth {
	return addDurationToYearMonth("add", ym, temporalDurationLike, options)
}

// Subtract is Temporal.PlainYearMonth.prototype.subtract.
func (ym *PlainYearMonth) Subtract(temporalDurationLike, options any) *PlainYearMonth {
	return addDurationToYearMonth("subtract", ym, temporalDurationLike, options)
}

// Until is Temporal.PlainYearMonth.prototype.until.
func (ym *PlainYearMonth) Until(other, options any) *Duration {
	return differenceTemporalPlainYearMonth("until", ym, other, options)
}

// Since is Temporal.PlainYearMonth.prototype.since.
func (ym *PlainYearMonth) Since(other, options any) *Duration {
	return differenceTemporalPlainYearMonth("since", ym, other, options)
}

// Equals is Temporal.PlainYearMonth.prototype.equals.
func (ym *PlainYearMonth) Equals(otherParam any) bool {
	other := toTemporalYearMonth(otherParam)
	if compareISODate(ym.isoDate, other.isoDate) != 0 {
		return false
	}
	return calendarEquals(ym.calendar, other.calendar)
}

// ToString is Temporal.PlainYearMonth.prototype.toString.
func (ym *PlainYearMonth) ToString(options any) string {
	resolvedOptions := getOptionsObject(options)
	showCalendar := getTemporalShowCalendarNameOption(resolvedOptions)
	return temporalYearMonthToString(ym, showCalendar)
}

// ToJSON is Temporal.PlainYearMonth.prototype.toJSON.
func (ym *PlainYearMonth) ToJSON() string { return temporalYearMonthToString(ym, "auto") }

// ToLocaleString is Temporal.PlainYearMonth.prototype.toLocaleString.
func (ym *PlainYearMonth) ToLocaleString(locales, options any) string {
	return createDateTimeFormat(locales, options, "date").Format(ym)
}

func (ym *PlainYearMonth) jsValueOf() { valueOfThrows("PlainYearMonth") }

func (ym *PlainYearMonth) jsToString() string { return ym.ToString(nil) }

// ToPlainDate is Temporal.PlainYearMonth.prototype.toPlainDate.
func (ym *PlainYearMonth) ToPlainDate(item any) *PlainDate {
	if !isObject(item) {
		throwType("argument should be an object")
	}
	calendar := ym.calendar

	fields := isoDateToFields(calendar, ym.isoDate, "year-month")
	inputFields := prepareCalendarFields(calendar, item, []string{"day"}, []string{}, []string{})
	mergedFields := calendarMergeFields(calendar, fields, inputFields)
	isoDate := calendarDateFromFields(calendar, mergedFields, "constrain")
	return createTemporalDate(isoDate, calendar)
}

// PlainYearMonthFrom is Temporal.PlainYearMonth.from.
func PlainYearMonthFrom(item, options any) *PlainYearMonth { return toTemporalYearMonth(item, options) }

// PlainYearMonthCompare is Temporal.PlainYearMonth.compare.
func PlainYearMonthCompare(oneParam, twoParam any) float64 {
	one := toTemporalYearMonth(oneParam)
	two := toTemporalYearMonth(twoParam)
	return compareISODate(one.isoDate, two.isoDate)
}

func (ym *PlainYearMonth) jsGet(key string) any {
	switch key {
	case "year":
		return ym.Year()
	case "month":
		return ym.Month()
	case "monthCode":
		return ym.MonthCode()
	case "calendarId":
		return ym.CalendarId()
	case "era":
		return stringOrUndefined(ym.Era())
	case "eraYear":
		return numberOrUndefined(ym.EraYear())
	case "daysInMonth":
		return ym.DaysInMonth()
	case "daysInYear":
		return ym.DaysInYear()
	case "monthsInYear":
		return ym.MonthsInYear()
	case "inLeapYear":
		return ym.InLeapYear()
	}
	return nil
}
