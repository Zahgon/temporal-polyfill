package temporal

// nonIsoCalendar is the common implementation of all non-ISO calendars. Per-calendar id and logic
// live in the helper. This split allowed an easy separation between code that was similar between
// ISO and non-ISO implementations vs. code that was very different.
//
// The original keeps a cache per ISO Date Record object. Go's dates are values, so each call here
// starts its own, as the original does for a date object it has not seen.
type nonIsoCalendar struct {
	helper calHelper
	b      *helperBase
}

func (c *nonIsoCalendar) extraFields(fields []string) []string {
	if calendarSupportsEra(c.b.id) && calIncludes(fields, "year") {
		return []string{"era", "eraYear"}
	}
	return []string{}
}

func (c *nonIsoCalendar) resolveFields(fields *calendarFields, typ string) {
	if (typ == "date" || typ == "year-month") && fields.year == nil {
		if !calendarSupportsEra(c.b.id) {
			throwType("year is required")
		} else if fields.era == nil || fields.eraYear == nil {
			throwType("year (or era and eraYear) are required")
		}
	}
	if (typ == "date" || typ == "month-day") && fields.day == nil {
		throwType("day is required")
	}
	if typ == "month-day" && fields.month != nil && fields.year == nil {
		if !calendarSupportsEra(c.b.id) {
			throwType("when month is present, year is required")
		} else if fields.era == nil || fields.eraYear == nil {
			throwType("when month is present, year (or era and eraYear) are required")
		}
	}
	if c.b.calendarType != "lunisolar" {
		resolveNonLunisolarMonth(fields.month, fields.monthCode, c.b.id, "")
	}
	// Note: Lunisolar calendars go on to resolve month/monthCode in their
	// adjustCalendarDate implementations
}

func (c *nonIsoCalendar) dateToISO(fields *calendarFields, overflow string) isoDate {
	cache := newCalCache()
	return c.b.calendarToIsoDate(calDateFromFields(fields), overflow, cache)
}

func (c *nonIsoCalendar) monthDayToISOReferenceDate(fields *calendarFields, overflow string) isoDate {
	cache := newCalCache()
	// The result's year is a reference year where this month/day exists in this calendar
	return c.b.monthDayFromFields(calDateFromFields(fields), overflow, cache)
}

func (c *nonIsoCalendar) fieldKeysToIgnore(keys []string) []string {
	result := &orderedKeySet{keys: []string{}}
	for _, key := range keys {
		result.add(key)
		switch key {
		case "era":
			result.add("eraYear")
			result.add("year")
		case "eraYear":
			result.add("era")
			result.add("year")
		case "year":
			result.add("era")
			result.add("eraYear")
		case "month":
			result.add("monthCode")
			// See https://github.com/tc39/proposal-temporal/issues/1784
			if c.b.erasBeginMidYear {
				result.add("era")
				result.add("eraYear")
			}
		case "monthCode":
			result.add("month")
			if c.b.erasBeginMidYear {
				result.add("era")
				result.add("eraYear")
			}
		case "day":
			if c.b.erasBeginMidYear {
				result.add("era")
				result.add("eraYear")
			}
		}
	}
	return result.keys
}

func (c *nonIsoCalendar) dateAdd(d isoDate, dur dateDuration, overflow string) isoDate {
	cache := newCalCache()
	calendarDate := c.helper.isoToCalendarDate(d, cache)
	added := c.b.addCalendar(calendarDate, dur, overflow, cache)
	return c.b.calendarToIsoDate(added, "constrain", cache)
}

func (c *nonIsoCalendar) dateUntil(one, two isoDate, largestUnit string) dateDuration {
	if largestUnit == "week" || largestUnit == "day" {
		weeks := 0.0
		days := isoDateToEpochDays(two.year, two.month-1, two.day) -
			isoDateToEpochDays(one.year, one.month-1, one.day)

		if largestUnit == "week" {
			weeks = jsTrunc(days / 7)
			days = jsMod(days, 7)
		}

		return dateDuration{years: 0, months: 0, weeks: weeks, days: days}
	}

	cacheOne := newCalCache()
	cacheTwo := newCalCache()
	calendarOne := c.helper.isoToCalendarDate(one, cacheOne)
	calendarTwo := c.helper.isoToCalendarDate(two, cacheTwo)
	return c.b.untilCalendar(calendarOne, calendarTwo, largestUnit, cacheOne)
}

func (c *nonIsoCalendar) isoToDate(d isoDate, req calReq) calendarDate {
	cache := newCalCache()
	full := c.helper.isoToCalendarDate(d, cache)
	calendarDate := calendarDate{
		era:       full.era,
		eraYear:   full.eraYear,
		year:      *full.year,
		month:     *full.month,
		monthCode: *full.monthCode,
		day:       *full.day,
	}
	if req&reqDayOfWeek != 0 {
		calendarDate.dayOfWeek = isoCalendar{}.isoToDate(d, reqDayOfWeek).dayOfWeek
	}
	if req&reqDayOfYear != 0 {
		startOfYear := c.b.startOfCalendarYear(full)
		diffDays := c.b.calendarDaysUntil(startOfYear, full, cache)
		calendarDate.dayOfYear = diffDays + 1
	}
	if req&reqWeekOfYear != 0 {
		calendarDate.weekOfYear, calendarDate.yearOfWeek = calendarDateWeekOfYear(c.b.id, d)
	}
	calendarDate.daysInWeek = 7
	if req&reqDaysInMonth != 0 {
		calendarDate.daysInMonth = c.helper.daysInMonth(full, cache)
	}
	if req&reqDaysInYear != 0 {
		startOfYearCalendar := c.b.startOfCalendarYear(full)
		startOfNextYearCalendar := c.b.addCalendar(startOfYearCalendar, dateDuration{years: 1}, "constrain", cache)
		calendarDate.daysInYear = c.b.calendarDaysUntil(startOfYearCalendar, startOfNextYearCalendar, cache)
	}
	if req&reqMonthsInYear != 0 {
		calendarDate.monthsInYear = c.helper.monthsInYear(full, cache)
	}
	if req&reqInLeapYear != 0 {
		calendarDate.inLeapYear = c.helper.inLeapYear(full, cache)
	}
	return calendarDate
}

func (c *nonIsoCalendar) getFirstDayOfWeek() *float64 { return nil }

func (c *nonIsoCalendar) getMinimalDaysInFirstWeek() *float64 { return nil }
