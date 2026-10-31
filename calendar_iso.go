package temporal

import "math"

// orderedKeySet is the Set of the original's fieldKeysToIgnore: insertion order, no duplicates.
type orderedKeySet struct{ keys []string }

func (s *orderedKeySet) add(key string) {
	for _, k := range s.keys {
		if k == key {
			return
		}
	}
	s.keys = append(s.keys, key)
}

func calendarDateWeekOfYear(id string, d isoDate) (week, year *float64) {
	// Supports only ISO8601 calendar; can be updated to add support for other calendars.
	// Returns undefined for calendars without a well-defined week calendar system.
	// Also see: https://github.com/unicode-org/icu/blob/ab72ab1d4a3c3f9beeb7d92b0c7817ca93dfdb04/icu4c/source/i18n/calendar.cpp#L1606
	if id != "iso8601" {
		return nil, nil
	}
	calendar := isoCalendar{}
	yow := d.year
	r := calendar.isoToDate(d, reqDayOfWeek|reqDayOfYear|reqDaysInYear)
	dayOfWeek, dayOfYear, daysInYear := r.dayOfWeek, r.dayOfYear, r.daysInYear
	fdow := *calendar.getFirstDayOfWeek()
	mdow := *calendar.getMinimalDaysInFirstWeek()

	// For both the input date and the first day of its calendar year, calculate the day of week
	// relative to first day of week in the relevant calendar (e.g., in iso8601, relative to Monday).
	relDow := jsMod(dayOfWeek+7-fdow, 7)
	// Assuming the year length is less than 7000 days.
	relDowJan1 := jsMod(dayOfWeek-dayOfYear+7001-fdow, 7)

	woy := math.Floor((dayOfYear - 1 + relDowJan1) / 7)
	if 7-relDowJan1 >= mdow {
		woy++
	}

	// Adjust for weeks at the year end that overlap into the previous or next calendar year.
	if woy == 0 {
		// Check for last week of previous year; if true, handle the case for
		// first week of next year
		prevYearCalendar := calendar.isoToDate(calendar.dateAdd(d, dateDuration{years: -1}, "constrain"), reqDaysInYear)
		prevDoy := dayOfYear + prevYearCalendar.daysInYear
		woy = weekNumber(fdow, mdow, prevDoy, dayOfWeek)
		yow--
	} else {
		// For it to be week 1 of the next year, dayOfYear must be >= lastDoy - 5
		//          L-5                  L
		// doy: 359 360 361 362 363 364 365 001
		// dow:      1   2   3   4   5   6   7
		lastDoy := daysInYear
		if dayOfYear >= lastDoy-5 {
			lastRelDow := jsMod(relDow+lastDoy-dayOfYear, 7)
			calAssert(lastRelDow >= 0, "lastRelDow must be non-negative because relDow is positive and lastDoy >= dayOfYear")
			if 6-lastRelDow >= mdow && dayOfYear+7-relDow > lastDoy {
				woy = 1
				yow++
			}
		}
	}
	return calNum(woy), calNum(yow)
}

func compareSurpasses(sign, year, month, day float64, target isoDate) bool {
	// Optimization: monthCode is not used, instead month codes are compared
	// lexicographically in untilCalendar() before this function is called
	if year != target.year {
		if float64(sign*(year-target.year)) > 0 {
			return true
		}
	} else if month != target.month {
		if float64(sign*(month-target.month)) > 0 {
			return true
		}
	} else if day != target.day {
		if float64(sign*(day-target.day)) > 0 {
			return true
		}
	}
	return false
}

func isoDateSurpasses(sign float64, baseDate, isoDate2 isoDate, years, months float64) bool {
	y0 := baseDate.year + years
	if compareSurpasses(sign, y0, baseDate.month, baseDate.day, isoDate2) {
		return true
	}
	if months == 0 {
		return false
	}
	m0 := baseDate.month + months
	monthsAdded := balanceISOYearMonth(y0, m0)
	return compareSurpasses(sign, monthsAdded.year, monthsAdded.month, baseDate.day, isoDate2)
	// Skip the weeks and days addition in the spec, as we currently optimize that
	// out in iso8601 date difference calculation
}

// isoCalendar is the implementation of the ISO 8601 calendar. This is the only calendar that's
// guaranteed to be supported by all ECMAScript implementations, including those without Intl
// (ECMA-402) support.
type isoCalendar struct{}

func (isoCalendar) resolveFields(fields *calendarFields, typ string) {
	if (typ == "date" || typ == "year-month") && fields.year == nil {
		throwType("year is required")
	}
	if (typ == "date" || typ == "month-day") && fields.day == nil {
		throwType("day is required")
	}
	month, monthCode := resolveNonLunisolarMonth(fields.month, fields.monthCode, "iso8601", "")
	fields.month = calNum(month)
	fields.monthCode = calStr(monthCode)
}

func (isoCalendar) dateToISO(fields *calendarFields, overflow string) isoDate {
	return regulateISODate(calVal(fields.year), calVal(fields.month), calVal(fields.day), overflow)
}

func (isoCalendar) monthDayToISOReferenceDate(fields *calendarFields, overflow string) isoDate {
	referenceISOYear := 1972.0
	year := referenceISOYear
	if fields.year != nil {
		year = *fields.year
	}
	r := regulateISODate(year, calVal(fields.month), calVal(fields.day), overflow)
	return isoDate{year: referenceISOYear, month: r.month, day: r.day}
}

func (isoCalendar) extraFields(fields []string) []string { return []string{} }

func (isoCalendar) fieldKeysToIgnore(keys []string) []string {
	result := &orderedKeySet{keys: []string{}}
	for _, key := range keys {
		result.add(key)
		if key == "month" {
			result.add("monthCode")
		} else if key == "monthCode" {
			result.add("month")
		}
	}
	return result.keys
}

func (isoCalendar) dateAdd(d isoDate, dur dateDuration, overflow string) isoDate {
	year, month, day := d.year, d.month, d.day
	year += dur.years
	month += dur.months
	balanced := balanceISOYearMonth(year, month)
	year, month = balanced.year, balanced.month
	intermediate := regulateISODate(year, month, day, overflow)
	return addDaysToISODate(intermediate, dur.days+float64(7*dur.weeks))
}

func (isoCalendar) dateUntil(one, two isoDate, largestUnit string) dateDuration {
	sign := float64(-compareISODate(one, two))
	calAssert(sign != 0, "this case was checked in CalendarDateUntil")

	years := 0.0
	months := 0.0
	if largestUnit == "year" || largestUnit == "month" {
		// We can skip right to the neighbourhood of the correct number of years,
		// it'll be at least one less than two.year - one.year (unless it's zero)
		candidateYears := two.year - one.year
		if candidateYears != 0 {
			candidateYears -= sign
		}
		// loops at most twice
		for !isoDateSurpasses(sign, one, two, candidateYears, 0) {
			years = candidateYears
			candidateYears += sign
		}

		candidateMonths := sign
		// loops at most 12 times
		for !isoDateSurpasses(sign, one, two, years, candidateMonths) {
			months = candidateMonths
			candidateMonths += sign
		}

		if largestUnit == "month" {
			months += float64(years * 12)
			years = 0
		}
	}

	intermediate := balanceISOYearMonth(one.year+years, one.month+months)
	constrained := constrainISODate(intermediate.year, intermediate.month, one.day)

	weeks := 0.0
	days := isoDateToEpochDays(two.year, two.month-1, two.day) -
		isoDateToEpochDays(constrained.year, constrained.month-1, constrained.day)

	if largestUnit == "week" {
		weeks = jsTrunc(days / 7)
		days = jsMod(days, 7)
	}

	return dateDuration{years: years, months: months, weeks: weeks, days: days}
}

func (isoCalendar) isoToDate(d isoDate, req calReq) calendarDate {
	// The requested fields argument is not part of the spec text. It's an
	// illustration of one way implementations may choose to optimize this
	// operation.
	year, month, day := d.year, d.month, d.day
	date := calendarDate{year: year, month: month, day: day, daysInWeek: 7, monthsInYear: 12}
	if req&reqMonthCode != 0 {
		date.monthCode = createMonthCode(month, false)
	}
	if req&reqDayOfWeek != 0 {
		// https://en.wikipedia.org/wiki/Determination_of_the_day_of_the_week#Disparate_variation
		monthShift := -2.0
		if month < 3 {
			monthShift = 10
		}
		shiftedMonth := month + monthShift
		yearShift := 0.0
		if month < 3 {
			yearShift = 1
		}
		shiftedYear := year - yearShift

		century := math.Floor(shiftedYear / 100)
		yearInCentury := shiftedYear - float64(century*100)

		monthTerm := math.Floor(float64(2.6*shiftedMonth) - 0.2)
		yearTerm := yearInCentury + math.Floor(yearInCentury/4)
		centuryTerm := math.Floor(century/4) - float64(2*century)

		dow := jsMod(day+monthTerm+yearTerm+centuryTerm, 7)

		if dow <= 0 {
			date.dayOfWeek = dow + 7
		} else {
			date.dayOfWeek = dow + 0
		}
	}
	if req&reqDayOfYear != 0 {
		days := day
		for m := month - 1; m > 0; m-- {
			days += isoDaysInMonth(year, m)
		}
		date.dayOfYear = days
	}
	if req&reqWeekOfYear != 0 {
		date.weekOfYear, date.yearOfWeek = calendarDateWeekOfYear("iso8601", isoDate{year: year, month: month, day: day})
	}
	if req&reqDaysInMonth != 0 {
		date.daysInMonth = isoDaysInMonth(year, month)
	}
	if req&reqDaysInYear != 0 || req&reqInLeapYear != 0 {
		date.inLeapYear = leapYear(year)
		date.daysInYear = 365
		if date.inLeapYear {
			date.daysInYear = 366
		}
	}
	return date
}

func (isoCalendar) getFirstDayOfWeek() *float64 { return calNum(1) }

func (isoCalendar) getMinimalDaysInFirstWeek() *float64 { return calNum(4) }

func weekNumber(firstDayOfWeek, minimalDaysInFirstWeek, desiredDay, dayOfWeek float64) float64 {
	periodStartDayOfWeek := jsMod(dayOfWeek-firstDayOfWeek-desiredDay+1, 7)
	if periodStartDayOfWeek < 0 {
		periodStartDayOfWeek += 7
	}
	weekNo := math.Floor((desiredDay + periodStartDayOfWeek - 1) / 7)
	if 7-periodStartDayOfWeek >= minimalDaysInFirstWeek {
		weekNo++
	}
	return weekNo
}
