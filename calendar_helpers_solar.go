package temporal

import "math"

// islamicBaseHelper is the Islamic calendar. For Temporal purposes, the Islamic calendar is simple
// because it's always the same 12 months in the same order.
//
// There are 4 Islamic calendars with the same implementation in this polyfill. They vary only in
// their ID and epoch start date. They do emit different output from the underlying Intl
// implementation, but our code for each of them is identical.
type islamicBaseHelper struct {
	helperBase
	tabular bool
}

func newIslamicBase(id string, tabular bool, epochDay float64) *islamicBaseHelper {
	h := &islamicBaseHelper{tabular: tabular}
	h.init(h, id, "lunar", []*calInputEra{
		{code: "ah", isoEpoch: &isoDate{year: 622, month: 7, day: epochDay}},
		{code: "bh", reverseOf: "ah"},
	})
	return h
}

func newIslamicUmalquraHelper() *islamicBaseHelper {
	return newIslamicBase("islamic-umalqura", false, 20)
}
func newIslamicTblaHelper() *islamicBaseHelper  { return newIslamicBase("islamic-tbla", true, 19) }
func newIslamicCivilHelper() *islamicBaseHelper { return newIslamicBase("islamic-civil", true, 20) }
func newIslamicCcHelper() *islamicBaseHelper    { return newIslamicBase("islamicc", true, 20) }

func (h *islamicBaseHelper) inLeapYear(calendarDate *calDate, cache *calCache) bool {
	startOfYearCalendar := &calDate{year: calendarDate.year, month: calNum(1), monthCode: calStr("M01"), day: calNum(1)}
	startOfNextYearCalendar := &calDate{year: calNum(*calendarDate.year + 1), month: calNum(1), monthCode: calStr("M01"), day: calNum(1)}
	result := h.calendarDaysUntil(startOfYearCalendar, startOfNextYearCalendar, cache)
	return result == 355
}

func (h *islamicBaseHelper) monthsInYear(calendarDate *calDate, cache *calCache) float64 { return 12 }

func (h *islamicBaseHelper) minimumMonthLength(calendarDate *calDate) float64 { return 29 }

func (h *islamicBaseHelper) maximumMonthLength(calendarDate *calDate) float64 { return 30 }

func (h *islamicBaseHelper) maxLengthOfMonthCodeInAnyYear(monthCode string) float64 {
	if !h.tabular {
		return 30 // if observational, any month can have 29 or 30 days
	}
	month, _ := parseMonthCode(monthCode)
	return [...]float64{0, 30, 29, 30, 29, 30, 29, 30, 29, 30, 29, 30, 30}[int(month)]
}

// These are variables so that the division is the run-time one the original performs, not an exact
// constant expression rounded once at the end.
var (
	islamicEleven, islamicThirty = 11.0, 30.0
	daysPerIslamicYear           = 354 + islamicEleven/islamicThirty
	daysPerISOYear               = 365.2425
)

func (h *islamicBaseHelper) estimateIsoDate(calendarDate *calDate) isoDate {
	year := *h.self.adjustCalendarDate(calendarDate, nil, "constrain", false).year
	return isoDate{year: math.Floor(float64(year*daysPerIslamicYear)/daysPerISOYear) + 622, month: 1, day: 1}
}

type persianHelper struct{ helperBase }

func newPersianHelper() *persianHelper {
	h := &persianHelper{}
	h.init(h, "persian", "solar", []*calInputEra{{code: "ap", isoEpoch: &isoDate{year: 622, month: 3, day: 22}}})
	return h
}

func (h *persianHelper) inLeapYear(calendarDate *calDate, cache *calCache) bool {
	// If the last month has 30 days, it's a leap year.
	return h.self.daysInMonth(calYMD(*calendarDate.year, 12, 1), cache) == 30
}

func (h *persianHelper) monthsInYear(calendarDate *calDate, cache *calCache) float64 { return 12 }

func (h *persianHelper) minimumMonthLength(calendarDate *calDate) float64 {
	month := *calendarDate.month
	if month == 12 {
		return 29
	}
	if month <= 6 {
		return 31
	}
	return 30
}

func (h *persianHelper) maximumMonthLength(calendarDate *calDate) float64 {
	month := *calendarDate.month
	if month == 12 {
		return 30
	}
	if month <= 6 {
		return 31
	}
	return 30
}

func (h *persianHelper) maxLengthOfMonthCodeInAnyYear(monthCode string) float64 {
	monthNumber, _ := parseMonthCode(monthCode)
	if monthNumber <= 6 {
		return 31
	}
	return 30
}

func (h *persianHelper) estimateIsoDate(calendarDate *calDate) isoDate {
	year := *h.self.adjustCalendarDate(calendarDate, nil, "constrain", false).year
	return isoDate{year: year + 621, month: 1, day: 1}
}

// indianMonthInfo is the length of an Indian month and the Gregorian month and day where it starts.
type indianMonthInfo struct {
	length, month, day float64
	leap               *indianMonthInfo
	nextYear           bool
}

type indianHelper struct{ helperBase }

func newIndianHelper() *indianHelper {
	h := &indianHelper{}
	h.init(h, "indian", "solar", []*calInputEra{{code: "shaka", isoEpoch: &isoDate{year: 79, month: 3, day: 23}}})
	return h
}

func (h *indianHelper) inLeapYear(calendarDate *calDate, cache *calCache) bool {
	// From https://en.wikipedia.org/wiki/Indian_national_calendar:
	// Years are counted in the Saka era, which starts its year 0 in the year 78
	// of the Common Era. To determine leap years, add 78 to the Saka year – if
	// the result is a leap year in the Gregorian calendar, then the Saka year
	// is a leap year as well.
	return isGregorianLeapYear(*calendarDate.year + 78)
}

func (h *indianHelper) monthsInYear(calendarDate *calDate, cache *calCache) float64 { return 12 }

func (h *indianHelper) minimumMonthLength(calendarDate *calDate) float64 {
	return h.getMonthInfo(calendarDate).length
}

func (h *indianHelper) maximumMonthLength(calendarDate *calDate) float64 {
	return h.getMonthInfo(calendarDate).length
}

func (h *indianHelper) maxLengthOfMonthCodeInAnyYear(monthCode string) float64 {
	monthNumber, _ := parseMonthCode(monthCode)
	monthInfo := indianMonths[monthNumber]
	if monthInfo.leap != nil {
		monthInfo = monthInfo.leap
	}
	return monthInfo.length
}

// Indian months always start at the same well-known Gregorian month and
// day. So this conversion is easy and fast. See
// https://en.wikipedia.org/wiki/Indian_national_calendar
var indianMonths = map[float64]*indianMonthInfo{
	1:  {length: 30, month: 3, day: 22, leap: &indianMonthInfo{length: 31, month: 3, day: 21}},
	2:  {length: 31, month: 4, day: 21},
	3:  {length: 31, month: 5, day: 22},
	4:  {length: 31, month: 6, day: 22},
	5:  {length: 31, month: 7, day: 23},
	6:  {length: 31, month: 8, day: 23},
	7:  {length: 30, month: 9, day: 23},
	8:  {length: 30, month: 10, day: 23},
	9:  {length: 30, month: 11, day: 22},
	10: {length: 30, month: 12, day: 22},
	11: {length: 30, month: 1, nextYear: true, day: 21},
	12: {length: 30, month: 2, nextYear: true, day: 20},
}

func (h *indianHelper) getMonthInfo(calendarDate *calDate) *indianMonthInfo {
	month := *calendarDate.month
	monthInfo := indianMonths[month]
	if monthInfo == nil {
		calAssertFail("getMonthInfo called on date with invalid month " + numberToString(month))
	}
	if h.inLeapYear(calendarDate, nil) && monthInfo.leap != nil {
		monthInfo = monthInfo.leap
	}
	return monthInfo
}

func (h *indianHelper) estimateIsoDate(calendarDateParam *calDate) isoDate {
	// FYI, this "estimate" is always the exact ISO date, which makes the Indian
	// calendar fast!
	calendarDate := h.self.adjustCalendarDate(calendarDateParam, nil, "constrain", false)
	monthInfo := h.getMonthInfo(calendarDate)
	nextYear := 0.0
	if monthInfo.nextYear {
		nextYear = 1
	}
	isoYear := *calendarDate.year + 78 + nextYear
	isoMonth := monthInfo.month
	isoDay := monthInfo.day
	return addDaysToISODate(isoDate{year: isoYear, month: isoMonth, day: isoDay}, *calendarDate.day-1)
}

// https://bugs.chromium.org/p/v8/issues/detail?id=10529 causes Intl's Indian
// calendar output to fail for all dates before 0001-01-01 ISO. For example,
// in Node 12 0000-01-01 is calculated as 6146/12/-583 instead of 10/11/-79 as
// expected. The original asks ICU when it loads; the ICU of this port answers
// "10/11/-79 Śaka", so vulnerableToBceBug is false and checkIcuBugs does nothing.

func (h *indianHelper) reviseIntlEra(calendarDate *calDate, isoDate isoDate) (*string, *float64, bool) {
	// Some ICU versions have the legacy era code 'saka', the correct one is
	// 'shaka'; return it unconditionally as there is only one era
	return calStr("shaka"), calendarDate.eraYear, true
}

// gregorianBaseHelper is the base for Gregorian-like calendars with eras.
type gregorianBaseHelper struct{ helperBase }

func (h *gregorianBaseHelper) inLeapYear(calendarDate *calDate, cache *calCache) bool {
	// Calendars that don't override this method use the same months and leap
	// years as Gregorian. Once we know the ISO year corresponding to the
	// calendar year, we'll know if it's a leap year or not.
	year := h.self.estimateIsoDate(calYMD(*calendarDate.year, 1, 1)).year
	return isGregorianLeapYear(year)
}

func (h *gregorianBaseHelper) monthsInYear(calendarDate *calDate, cache *calCache) float64 { return 12 }

func (h *gregorianBaseHelper) minimumMonthLength(calendarDate *calDate) float64 {
	month := *calendarDate.month
	if month == 2 {
		if h.self.inLeapYear(calendarDate, nil) {
			return 29
		}
		return 28
	}
	if month == 4 || month == 6 || month == 9 || month == 11 {
		return 30
	}
	return 31
}

func (h *gregorianBaseHelper) maximumMonthLength(calendarDate *calDate) float64 {
	return h.self.minimumMonthLength(calendarDate)
}

func (h *gregorianBaseHelper) maxLengthOfMonthCodeInAnyYear(monthCode string) float64 {
	month, _ := parseMonthCode(monthCode)
	return [...]float64{31, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}[int(month)-1]
}

func (h *gregorianBaseHelper) estimateIsoDate(calendarDateParam *calDate) isoDate {
	calendarDate := h.self.adjustCalendarDate(calendarDateParam, nil, "constrain", false)
	year, month, day := *calendarDate.year, *calendarDate.month, *calendarDate.day
	anchorEra := h.anchorEra
	isoYearEstimate := year + anchorEra.isoEpoch.year - anchorEra.startingYear
	return regulateISODate(isoYearEstimate, month, day, "constrain")
}

// sameMonthDayAsGregorianBaseHelper is for calendars identical to Gregorian except era and year. For
// these calendars, we can avoid using Intl.DateTimeFormat and just calculate the year, era, and
// eraYear. This is faster (because Intl.DateTimeFormat is slow and uses a huge amount of RAM), and
// it avoids ICU bugs like https://bugs.chromium.org/p/chromium/issues/detail?id=1173158.
type sameMonthDayAsGregorianBaseHelper struct{ gregorianBaseHelper }

func (h *sameMonthDayAsGregorianBaseHelper) isoToCalendarDate(isoDate isoDate, cache *calCache) *calDate {
	// Month and day are same as ISO, so bypass Intl.DateTimeFormat and
	// calculate the year, era, and eraYear here.
	isoYear, month, day := isoDate.year, isoDate.month, isoDate.day
	monthCode := createMonthCode(month, false)
	year := isoYear - h.anchorEra.isoEpoch.year + 1
	return h.completeEraYear(&calDate{year: calNum(year), month: calNum(month), monthCode: calStr(monthCode), day: calNum(day)})
}

// orthodoxBaseHelper holds the calendars with 12 30-day months and an extra 13th month.
type orthodoxBaseHelper struct{ gregorianBaseHelper }

func (h *orthodoxBaseHelper) inLeapYear(calendarDate *calDate, cache *calCache) bool {
	// Leap years happen one year before the Julian leap year. Note that this
	// calendar is based on the Julian calendar which has a leap year every 4
	// years, unlike the Gregorian calendar which doesn't have leap years on
	// years divisible by 100 except years divisible by 400.
	//
	// Note that we're assuming that leap years in before-epoch times match
	// how leap years are defined now. This is probably not accurate but I'm
	// not sure how better to do it.
	return jsMod(*calendarDate.year+1, 4) == 0
}

func (h *orthodoxBaseHelper) monthsInYear(calendarDate *calDate, cache *calCache) float64 { return 13 }

func (h *orthodoxBaseHelper) minimumMonthLength(calendarDate *calDate) float64 {
	month := *calendarDate.month
	// Ethiopian/Coptic calendars have 12 30-day months and an extra 5-6 day 13th month.
	if month == 13 {
		if h.self.inLeapYear(calendarDate, nil) {
			return 6
		}
		return 5
	}
	return 30
}

func (h *orthodoxBaseHelper) maximumMonthLength(calendarDate *calDate) float64 {
	return h.self.minimumMonthLength(calendarDate)
}

func (h *orthodoxBaseHelper) maxLengthOfMonthCodeInAnyYear(monthCode string) float64 {
	if monthCode == "M13" {
		return 6
	}
	return 30
}

// `coptic` and `ethiopic` calendars are very similar to `ethioaa` calendar,
// with the following differences:
//   - Coptic uses BCE-like positive numbers for years before its epoch (the other
//     two use negative year numbers before epoch)
//   - Coptic has a different epoch date
//   - Ethiopic has an additional second era that starts at the same date as the
//     zero era of ethioaa, which is the anchor era
func newEthioaaHelper() *orthodoxBaseHelper {
	h := &orthodoxBaseHelper{}
	h.init(h, "ethioaa", "solar", []*calInputEra{{code: "aa", isoEpoch: &isoDate{year: -5492, month: 7, day: 17}}})
	return h
}

const copticLegacyEra0 = "era0"

type copticHelper struct{ orthodoxBaseHelper }

func newCopticHelper() *copticHelper {
	h := &copticHelper{}
	h.init(h, "coptic", "solar", []*calInputEra{
		// Empty era to accommodate old versions of ICU data having ERA0 and ERA1.
		// Both map to AM
		{code: "am", isoEpoch: &isoDate{year: 284, month: 8, day: 29}},
		{code: copticLegacyEra0, reverseOf: "am", skip: true},
	})
	return h
}

func (h *copticHelper) reviseIntlEra(calendarDate *calDate, isoDate isoDate) (*string, *float64, bool) {
	eraYear := calendarDate.eraYear
	if calendarDate.era != nil && *calendarDate.era == copticLegacyEra0 {
		eraYear = calNum(1 - calVal(eraYear))
	}
	return calStr("am"), eraYear, true
}

func newEthiopicHelper() *orthodoxBaseHelper {
	h := &orthodoxBaseHelper{}
	h.init(h, "ethiopic", "solar", []*calInputEra{
		{code: "aa", isoEpoch: &isoDate{year: -5492, month: 7, day: 17}, anchorEpoch: &calDate{year: calNum(-5499)}},
		{code: "am", isoEpoch: &isoDate{year: 8, month: 8, day: 27}},
	})
	return h
}

func newRocHelper() *sameMonthDayAsGregorianBaseHelper {
	h := &sameMonthDayAsGregorianBaseHelper{}
	h.init(h, "roc", "solar", []*calInputEra{
		{code: "roc", isoEpoch: &isoDate{year: 1912, month: 1, day: 1}},
		{code: "broc", reverseOf: "roc"},
	})
	return h
}

func newBuddhistHelper() *sameMonthDayAsGregorianBaseHelper {
	h := &sameMonthDayAsGregorianBaseHelper{}
	h.init(h, "buddhist", "solar", []*calInputEra{{code: "be", isoEpoch: &isoDate{year: -542, month: 1, day: 1}}})
	return h
}

type gregoryHelper struct {
	sameMonthDayAsGregorianBaseHelper
}

func newGregoryHelper() *gregoryHelper {
	h := &gregoryHelper{}
	h.init(h, "gregory", "solar", []*calInputEra{
		{code: "ce", isoEpoch: &isoDate{year: 1, month: 1, day: 1}},
		{code: "bce", reverseOf: "ce"},
	})
	return h
}

func (h *gregoryHelper) reviseIntlEra(calendarDate *calDate, isoDate isoDate) (*string, *float64, bool) {
	era, eraYear := calendarDate.era, calendarDate.eraYear
	// Firefox 96 introduced a bug where the `'short'` format of the era
	// option mistakenly returns the one-letter (narrow) format instead. The
	// code below handles either the correct or Firefox-buggy format. See
	// https://bugzilla.mozilla.org/show_bug.cgi?id=1752253
	if era != nil && *era == "b" {
		era = calStr("bce")
	}
	if era != nil && *era == "a" {
		era = calStr("ce")
	}
	return era, eraYear, true
}

// NOTE: Only the 5 modern eras (Meiji and later) are included. For dates
// before Meiji 6, the `ce` and `bce` eras are used. Challenges with pre-Meiji
// eras include:
//   - Start/end dates of older eras are not precisely defined, which is
//     challenging given Temporal's need for precision
//   - Some era dates and/or names are disputed by historians
//   - As historical research proceeds, new eras are discovered and existing era
//     dates are modified, leading to considerable churn which is not good for
//     Temporal use.
//   - The earliest era (in 645 CE) may not end up being the earliest depending
//     on future historical scholarship
//   - Before Meiji, Japan used a lunisolar calendar but that's not reflected in
//     the ICU implementation.
//
// For more discussion: https://github.com/tc39/proposal-temporal/issues/526.
//
// Here's a full list of CLDR/ICU eras:
// https://github.com/unicode-org/icu/blob/master/icu4c/source/data/locales/root.txt#L1582-L1818
// https://github.com/unicode-org/cldr/blob/master/common/supplemental/supplementalData.xml#L4310-L4546
//
// NOTE: Japan started using the Gregorian calendar in 6 Meiji, replacing a
// lunisolar calendar. So the day before January 1 of 6 Meiji (1873) was not
// December 31, but M12-02, of 5 Meiji (1872). To avoid confusion between
// lunisolar and solar dates we count years prior to 1872 as CE.
type japaneseHelper struct {
	sameMonthDayAsGregorianBaseHelper
}

func newJapaneseHelper() *japaneseHelper {
	h := &japaneseHelper{}
	h.init(h, "japanese", "solar", []*calInputEra{
		// The Japanese calendar `year` is just the ISO year, because (unlike other
		// ICU calendars) there's no obvious "default era", we use the ISO year.
		{code: "reiwa", isoEpoch: &isoDate{year: 2019, month: 5, day: 1}, anchorEpoch: calYMD(2019, 5, 1)},
		{code: "heisei", isoEpoch: &isoDate{year: 1989, month: 1, day: 8}, anchorEpoch: calYMD(1989, 1, 8)},
		{code: "showa", isoEpoch: &isoDate{year: 1926, month: 12, day: 25}, anchorEpoch: calYMD(1926, 12, 25)},
		{code: "taisho", isoEpoch: &isoDate{year: 1912, month: 7, day: 30}, anchorEpoch: calYMD(1912, 7, 30)},
		{code: "meiji", isoEpoch: &isoDate{year: 1873, month: 1, day: 1}, anchorEpoch: &calDate{year: calNum(1873)}, startingYear: calNum(6)},
		{code: "ce", isoEpoch: &isoDate{year: 1, month: 1, day: 1}},
		{code: "bce", reverseOf: "ce"},
	})
	h.erasBeginMidYear = true
	return h
}

func (h *japaneseHelper) reviseIntlEra(calendarDate *calDate, isoDate isoDate) (*string, *float64, bool) {
	era, eraYear := calendarDate.era, calendarDate.eraYear
	isoYear := isoDate.year
	for _, e := range h.eras {
		if era != nil && e.code == *era {
			return era, eraYear, true
		}
	}
	if isoYear < 1 {
		return calStr("bce"), calNum(1 - isoYear), true
	}
	return calStr("ce"), calNum(isoYear), true
}
