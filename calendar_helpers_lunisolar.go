package temporal

import (
	"strings"
	"sync"

	"github.com/js-temporal/temporal-polyfill-go/internal/intl"
)

// hebrewMonthInfo is the month index of a Hebrew month name in a leap and in a regular year, nil
// where it does not occur.
type hebrewMonthInfo struct{ leap, regular *float64 }

// hebrewMonthLength is the length of a Hebrew month, min equal to max if it does not vary.
type hebrewMonthLength struct{ min, max float64 }

type hebrewHelper struct{ helperBase }

func newHebrewHelper() *hebrewHelper {
	h := &hebrewHelper{}
	h.init(h, "hebrew", "lunisolar", []*calInputEra{{code: "am", isoEpoch: &isoDate{year: -3760, month: 9, day: 8}}})
	return h
}

func (h *hebrewHelper) inLeapYear(calendarDate *calDate, cache *calCache) bool {
	year := *calendarDate.year
	// FYI: In addition to adding a month in leap years, the Hebrew calendar
	// also has per-year changes to the number of days of Heshvan and Kislev.
	// Given that these can be calculated by counting the number of days in
	// those months, I assume that these DO NOT need to be exposed as
	// Hebrew-only prototype fields or methods.
	cycleYear := jsMod(float64(7*year)+1, 19)
	if cycleYear < 0 {
		cycleYear += 19
	}
	return cycleYear < 7
}

func (h *hebrewHelper) monthsInYear(calendarDate *calDate, cache *calCache) float64 {
	if h.inLeapYear(calendarDate, cache) {
		return 13
	}
	return 12
}

func (h *hebrewHelper) minimumMonthLength(calendarDate *calDate) float64 {
	return h.minMaxMonthLength(calendarDate).min
}

func (h *hebrewHelper) maximumMonthLength(calendarDate *calDate) float64 {
	return h.minMaxMonthLength(calendarDate).max
}

func (h *hebrewHelper) minMaxMonthLength(calendarDate *calDate) hebrewMonthLength {
	var monthCode string
	if calendarDate.monthCode != nil {
		monthCode = *calendarDate.monthCode
	} else {
		monthCode = h.getMonthCode(*calendarDate.year, *calendarDate.month)
	}
	daysInMonth, ok := hebrewMonthLengths[monthCode]
	calAssert(ok, "missing daysInMonth for Hebrew month "+monthCode)
	return daysInMonth
}

func (h *hebrewHelper) maxLengthOfMonthCodeInAnyYear(monthCode string) float64 {
	return hebrewMonthLengths[monthCode].max
}

// estimateIsoDate takes a guess at what ISO date a particular calendar date corresponds to.
func (h *hebrewHelper) estimateIsoDate(calendarDate *calDate) isoDate {
	return isoDate{year: *calendarDate.year - 3760, month: 1, day: 1}
}

func hebrewBoth(leap, regular float64) hebrewMonthInfo {
	return hebrewMonthInfo{leap: calNum(leap), regular: calNum(regular)}
}

var hebrewMonths = map[string]hebrewMonthInfo{
	"Tishri":  hebrewBoth(1, 1),
	"Heshvan": hebrewBoth(2, 2),
	"Kislev":  hebrewBoth(3, 3),
	"Tevet":   hebrewBoth(4, 4),
	"Shevat":  hebrewBoth(5, 5),
	"Adar":    {leap: nil, regular: calNum(6)},
	"Adar I":  {leap: calNum(6), regular: nil},
	"Adar II": {leap: calNum(7), regular: nil},
	"Nisan":   hebrewBoth(8, 7),
	"Iyar":    hebrewBoth(9, 8),
	"Sivan":   hebrewBoth(10, 9),
	"Tamuz":   hebrewBoth(11, 10),
	"Av":      hebrewBoth(12, 11),
	"Elul":    hebrewBoth(13, 12),
}

var hebrewMonthLengths = map[string]hebrewMonthLength{
	"M01":  {30, 30},
	"M02":  {29, 30},
	"M03":  {29, 30},
	"M04":  {29, 29},
	"M05":  {30, 30},
	"M05L": {30, 30},
	"M06":  {29, 29},
	"M07":  {30, 30},
	"M08":  {29, 29},
	"M09":  {30, 30},
	"M10":  {29, 29},
	"M11":  {30, 30},
	"M12":  {29, 29},
}

func (h *hebrewHelper) yearInLeap(year float64) bool {
	return h.inLeapYear(&calDate{year: calNum(year)}, nil)
}

func (h *hebrewHelper) getMonthCode(year, month float64) string {
	if h.yearInLeap(year) {
		if month == 6 {
			return createMonthCode(5, true)
		}
		if month < 6 {
			return createMonthCode(month, false)
		}
		return createMonthCode(month-1, false)
	}
	return createMonthCode(month, false)
}

func (h *hebrewHelper) adjustCalendarDate(calendarDate *calDate, cache *calCache, overflow string, fromLegacyDate bool) *calDate {
	c := h.completeEraYear(calendarDate)
	era, eraYear, year, month, monthCode, day, monthExtra := c.era, c.eraYear, *c.year, c.month, c.monthCode, c.day, c.monthExtra
	if fromLegacyDate {
		// In Pre Node-14 V8, DateTimeFormat.formatToParts `month: 'numeric'`
		// output returns the numeric equivalent of `month` as a string, meaning
		// that `'6'` in a leap year is Adar I, while `'6'` in a non-leap year
		// means Adar. In this case, `month` will already be correct and no action
		// is needed. However, in Node 14 and later formatToParts returns the name
		// of the Hebrew month (e.g. "Tevet"), so we'll need to look up the
		// correct `month` using the string name as a key.
		if monthExtra != nil {
			monthInfo, ok := hebrewMonths[*monthExtra]
			if !ok {
				throwRange("Unrecognized month from formatToParts: " + *monthExtra)
			}
			if h.yearInLeap(year) {
				month = monthInfo.leap
			} else {
				month = monthInfo.regular
			}
		}
		// Because we're getting data from legacy Date, then `month` will always be present
		code := h.getMonthCode(year, calVal(month))
		return &calDate{year: calNum(year), month: month, day: day, monthCode: calStr(code), era: era, eraYear: eraYear}
	}
	// When called without input coming from legacy Date output, simply ensure
	// that all fields are present.
	h.validateCalendarDate(calendarDate)
	dayValue := *day
	var monthValue float64
	if month == nil {
		if monthCode == nil {
			calAssertExistsFail()
		}
		monthNumber, isLeapMonth := parseMonthCode(*monthCode)
		if isLeapMonth {
			if monthNumber != 5 {
				throwRange("Hebrew leap month must have monthCode M05L, not " + *monthCode)
			}
			monthValue = 6
			if !h.yearInLeap(year) {
				if overflow == "reject" {
					throwRange("Hebrew monthCode M05L is invalid in year " + numberToString(year) + " which is not a leap year")
				} else {
					// constrain to same day of next month (Adar)
					monthValue = 6
					monthCode = calStr("M06")
				}
			}
		} else {
			monthValue = monthNumber
			// if leap month is before this one, the month index is one more than the month code
			if h.yearInLeap(year) && monthValue >= 6 {
				monthValue++
			}
			largestMonth := h.monthsInYear(&calDate{year: calNum(year)}, nil)
			if monthValue < 1 || monthValue > largestMonth {
				throwRange("Invalid monthCode: " + *monthCode)
			}
		}
	} else {
		monthValue = *month
		yearOnly := &calDate{year: calNum(year)}
		if overflow == "reject" {
			rejectToRange(monthValue, 1, h.monthsInYear(yearOnly, nil))
			rejectToRange(dayValue, 1, h.maximumMonthLength(&calDate{year: calNum(year), month: calNum(monthValue)}))
		} else {
			monthValue = constrainToRange(monthValue, 1, h.monthsInYear(yearOnly, nil))
			dayValue = constrainToRange(dayValue, 1, h.maximumMonthLength(&calDate{year: calNum(year), month: calNum(monthValue)}))
		}
		if monthCode == nil {
			monthCode = calStr(h.getMonthCode(year, monthValue))
		} else {
			calculatedMonthCode := h.getMonthCode(year, monthValue)
			if calculatedMonthCode != *monthCode {
				throwRange("monthCode " + *monthCode + " doesn't correspond to month " + numberToString(monthValue) + " in Hebrew year " + numberToString(year))
			}
		}
	}
	result := calendarDate.clone()
	result.day, result.month, result.monthCode, result.year, result.era, result.eraYear =
		calNum(dayValue), calNum(monthValue), monthCode, calNum(year), era, eraYear
	return result
}

// chineseMonth is one month of a Chinese or Dangi year.
type chineseMonth struct {
	monthCode   string
	daysInMonth float64
}

// chineseMonthList is the ChineseMonthInfo of the original: months by index and by month code, and
// the number of months.
type chineseMonthList struct {
	byIndex      map[float64]*chineseMonth
	byCode       map[string]float64
	monthsInYear float64
}

type chineseListKey struct {
	id   string
	year float64
}

// chineseLists holds the month lists computed so far. Asking for one costs a dozen ICU answers, and
// a list only depends on its calendar and year.
var (
	chineseListsMu sync.Mutex
	chineseLists   = map[chineseListKey]*chineseMonthList{}
)

type chineseBaseHelper struct{ helperBase }

func newChineseHelper() *chineseBaseHelper { return newChineseBase("chinese") }

// Dangi (Korean) calendar has same implementation as Chinese
func newDangiHelper() *chineseBaseHelper { return newChineseBase("dangi") }

func newChineseBase(id string) *chineseBaseHelper {
	h := &chineseBaseHelper{}
	h.init(h, id, "lunisolar", nil)
	return h
}

func (h *chineseBaseHelper) inLeapYear(calendarDate *calDate, cache *calCache) bool {
	return h.getMonthList(*calendarDate.year, cache).monthsInYear == 13
}

func (h *chineseBaseHelper) monthsInYear(calendarDate *calDate, cache *calCache) float64 {
	return h.getMonthList(*calendarDate.year, cache).monthsInYear
}

func (h *chineseBaseHelper) daysInMonth(calendarDate *calDate, cache *calCache) float64 {
	month, year := *calendarDate.month, *calendarDate.year
	matchingMonthEntry := h.getMonthList(year, cache).byIndex[month]
	if matchingMonthEntry == nil {
		calAssertFail("Invalid month " + numberToString(month) + " in " + h.id + " year " + numberToString(year))
	}
	return matchingMonthEntry.daysInMonth
}

func (h *chineseBaseHelper) daysInPreviousMonth(calendarDate *calDate, cache *calCache) float64 {
	month, year := *calendarDate.month, *calendarDate.year

	previousMonthYear := year
	if !(month > 1) {
		previousMonthYear = year - 1
	}
	previousMonthDate := calYMD(previousMonthYear, month, 1)
	previousMonth := month - 1
	if !(month > 1) {
		previousMonth = h.monthsInYear(previousMonthDate, cache)
	}

	return h.daysInMonth(&calDate{year: calNum(previousMonthYear), month: calNum(previousMonth)}, cache)
}

func (h *chineseBaseHelper) minimumMonthLength(calendarDate *calDate) float64 { return 29 }

func (h *chineseBaseHelper) maximumMonthLength(calendarDate *calDate) float64 { return 30 }

func (h *chineseBaseHelper) maxLengthOfMonthCodeInAnyYear(monthCode string) float64 {
	// See note below about ICU4C vs ICU4X. It is possible this override should
	// always return 30.
	switch monthCode {
	case "M01L", "M09L", "M10L", "M11L", "M12L":
		return 29
	}
	return 30
}

func (h *chineseBaseHelper) monthDaySearchStartYear(monthCode string, day float64) float64 {
	// Note that ICU4C actually has _no_ years in which leap months M01L and
	// M09L through M12L have 30 days. The values marked with (*) here are years
	// in which the leap month occurs with 29 days. ICU4C disagrees with ICU4X
	// here and it is not clear which is correct.
	years := [2]float64{1972, 1972}
	switch monthCode {
	case "M01L":
		years = [2]float64{1651, 1651} // *
	case "M02L":
		years = [2]float64{1947, 1765}
	case "M03L":
		years = [2]float64{1966, 1955}
	case "M04L":
		years = [2]float64{1963, 1944}
	case "M05L":
		years = [2]float64{1971, 1952}
	case "M06L":
		years = [2]float64{1960, 1941}
	case "M07L":
		years = [2]float64{1968, 1938}
	case "M08L":
		years = [2]float64{1957, 1718}
	case "M09L":
		years = [2]float64{1832, 1832} // *
	case "M10L":
		years = [2]float64{1870, 1870} // *
	case "M11L":
		years = [2]float64{1814, 1814} // *
	case "M12L":
		years = [2]float64{1890, 1890} // *
	}
	if day < 30 {
		return years[0]
	}
	return years[1]
}

func (h *chineseBaseHelper) getMonthList(calendarYear float64, cache *calCache) *chineseMonthList {
	key := generateMonthListKey(calendarYear)
	if cached := cache.monthLists[key]; cached != nil {
		return cached
	}

	memoKey := chineseListKey{h.id, calendarYear}
	chineseListsMu.Lock()
	monthList := chineseLists[memoKey]
	chineseListsMu.Unlock()
	if monthList == nil {
		monthList = h.computeMonthList(calendarYear)
		chineseListsMu.Lock()
		chineseLists[memoKey] = monthList
		chineseListsMu.Unlock()
	}

	cache.monthLists[key] = monthList
	return monthList
}

func (h *chineseBaseHelper) computeMonthList(calendarYear float64) *chineseMonthList {
	// Reuse the same local object for calendar-specific results, starting with
	// a date close to Chinese New Year. Feb 17 will either be in the new year
	// or near the end of the previous year's final month.
	daysPastJan31 := 17.0
	var fieldDay float64
	var fieldMonthString string
	var fieldRelatedYear *float64
	updateCalendarFields := func() {
		// Abuse GetUTCEpochMilliseconds for automatic rebalancing.
		isoNumbers := isoDate{year: calendarYear, month: 2, day: daysPastJan31}
		ms := getUTCEpochMilliseconds(isoNumbers, timeRecord{})
		fieldEntries, err := intl.CalendarParts(h.id, ms, "UTC")
		if err != nil {
			intlRaise(err)
		}
		for i := range fieldEntries {
			typ, value := fieldEntries[i].Type, fieldEntries[i].Value
			// day and year should be decimal strings, but month values like "5bis" are not number-coercible.
			if typ == "day" {
				fieldDay = stringToNumber(value)
			} else if typ == "relatedYear" {
				fieldRelatedYear = calNum(stringToNumber(value))
			} else if typ == "month" {
				fieldMonthString = strings.TrimPrefix(value, "Mo")
			}
		}
		if fieldRelatedYear == nil {
			// Node 12 has outdated ICU data that lacks the `relatedYear` field in the
			// output of Intl.DateTimeFormat.formatToParts.
			throwRange("Intl.DateTimeFormat.formatToParts lacks relatedYear in " + h.id + " calendar. Try Node 14+ or modern browsers.")
		}
	}

	// Ensure that we're in the first month.
	updateCalendarFields()
	if fieldMonthString != "1" {
		daysPastJan31 += 29
		updateCalendarFields()
	}

	// Now back up to near the start of the first month, but not so near that
	// off-by-one issues matter.
	daysPastJan31 -= fieldDay - 5

	monthList := &chineseMonthList{byIndex: map[float64]*chineseMonth{}, byCode: map[string]float64{}}
	monthIndex := 1.0
	oldDay := 0.0
	for {
		updateCalendarFields()
		day, monthString, relatedYear := fieldDay, fieldMonthString, *fieldRelatedYear
		if monthIndex == 1 {
			calAssert(monthString == "1", "we didn't back up to the beginning of year "+numberToString(calendarYear))
		}
		isLeapMonth := strings.HasSuffix(monthString, "bis")
		numberPart := monthString
		if isLeapMonth {
			numberPart = strings.TrimSuffix(monthString, "bis")
		}
		monthCode := createMonthCode(stringToNumber(numberPart), isLeapMonth)
		if oldDay != 0 {
			monthList.byIndex[monthIndex-1].daysInMonth = oldDay + 30 - day
		}
		oldDay = day

		if relatedYear != calendarYear {
			break
		}

		monthList.byIndex[monthIndex] = &chineseMonth{monthCode: monthCode}
		monthList.byCode[monthCode] = monthIndex
		monthIndex++

		// Move to the next month. Because months are sometimes 29 days, the day of the
		// calendar month will move forward slowly but not enough to flip over to a new
		// month before the loop ends at 12-13 months.
		daysPastJan31 += 30
	}
	monthList.monthsInYear = monthIndex - 1 // subtract 1, it was incremented after the loop
	return monthList
}

func (h *chineseBaseHelper) estimateIsoDate(calendarDate *calDate) isoDate {
	year, month := *calendarDate.year, *calendarDate.month
	isoMonth := month + 1
	if month >= 12 {
		isoMonth = 12
	}
	return isoDate{year: year, month: isoMonth, day: 1}
}

func (h *chineseBaseHelper) adjustCalendarDate(calendarDate *calDate, cache *calCache, overflow string, fromLegacyDate bool) *calDate {
	year, month, monthExtra, day, monthCode := calendarDate.year, calendarDate.month, calendarDate.monthExtra, calendarDate.day, calendarDate.monthCode
	if year == nil {
		calAssertFail("adjustCalendarDate called on date " + calJSONDate(calendarDate) + " with undefined year property")
	}
	if fromLegacyDate {
		// Legacy Date output returns a string that's an integer with an optional
		// "bis" suffix used only by the Chinese/Dangi calendar to indicate a leap
		// month. Below we'll normalize the output.
		if monthExtra != nil && *monthExtra != "bis" {
			throwRange("Unexpected leap month suffix: " + *monthExtra)
		}
		legacyMonthCode := createMonthCode(*month, monthExtra != nil)
		months := h.getMonthList(*year, cache)
		index, ok := months.byCode[legacyMonthCode]
		if !ok {
			extra := ""
			if monthExtra != nil {
				extra = *monthExtra
			}
			calAssertFail("Unmatched month undefined" + extra + " in " + h.id + " year " + numberToString(*year))
		}
		return &calDate{year: year, month: calNum(index), day: day, monthCode: calStr(legacyMonthCode)}
	}
	// When called without input coming from legacy Date output,
	// simply ensure that all fields are present.
	h.validateCalendarDate(calendarDate)
	if month == nil {
		if monthCode == nil {
			calAssertExistsFail()
		}
		months := h.getMonthList(*year, cache)
		monthNumber, isLeapMonth := parseMonthCode(*monthCode)
		index, ok := months.byCode[*monthCode]
		// If this leap month isn't present in this year, constrain to the same
		// day of the previous month.
		if !ok && isLeapMonth && overflow == "constrain" {
			adjustedMonthCode := createMonthCode(monthNumber, false)
			index, ok = months.byCode[adjustedMonthCode]
			monthCode = calStr(adjustedMonthCode)
		}
		if !ok {
			throwRange("Unmatched month " + *monthCode + " in " + h.id + " year " + numberToString(*year))
		}
		month = calNum(index)
	} else if monthCode == nil {
		months := h.getMonthList(*year, cache)
		largestMonth := months.monthsInYear
		if overflow == "reject" {
			rejectToRange(*month, 1, largestMonth)
			rejectToRange(*day, 1, h.maximumMonthLength(nil))
		} else {
			month = calNum(constrainToRange(*month, 1, largestMonth))
			day = calNum(constrainToRange(*day, 1, h.maximumMonthLength(nil)))
		}
		monthCode = calStr(months.byIndex[*month].monthCode)
	} else {
		// Both month and monthCode are present. Make sure they don't conflict.
		months := h.getMonthList(*year, cache)
		monthIndex, ok := months.byCode[*monthCode]
		if !ok {
			calAssertFail("Unmatched monthCode " + *monthCode + " in " + h.id + " year " + numberToString(*year))
		}
		if *month != monthIndex {
			throwRange("monthCode " + *monthCode + " doesn't correspond to month " + numberToString(*month) + " in " + h.id + " year " + numberToString(*year))
		}
	}
	result := calendarDate.clone()
	result.year, result.month, result.monthCode, result.day = year, month, monthCode, day
	return result
}
