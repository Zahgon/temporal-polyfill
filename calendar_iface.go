package temporal

// calReq is the set of fields asked of calendarImpl.isoToDate, the requestedFields argument of the
// original. A field that is not requested is not computed, so a computation that would throw is not
// reached.
type calReq uint32

const (
	reqEra calReq = 1 << iota
	reqEraYear
	reqYear
	reqMonth
	reqMonthCode
	reqDay
	reqDayOfWeek
	reqDayOfYear
	reqWeekOfYear
	reqDaysInWeek
	reqDaysInMonth
	reqDaysInYear
	reqMonthsInYear
	reqInLeapYear

	reqAll = reqEra | reqEraYear | reqYear | reqMonth | reqMonthCode | reqDay | reqDayOfWeek | reqDayOfYear |
		reqWeekOfYear | reqDaysInWeek | reqDaysInMonth | reqDaysInYear | reqMonthsInYear | reqInLeapYear
)

// calendarImpl is the shape of the internal implementation of each built-in calendar. Note that
// parameter types are simpler than the calendar protocol because the callers perform validation and
// parameter normalization before handing control over.
//
// There are two implementations of this interface: one for the ISO calendar and another that
// handles logic that's the same across all non-ISO calendars. The latter is instantiated for each
// non-ISO calendar in init below.
type calendarImpl interface {
	isoToDate(d isoDate, req calReq) calendarDate
	getFirstDayOfWeek() *float64
	getMinimalDaysInFirstWeek() *float64
	resolveFields(fields *calendarFields, typ string)
	dateToISO(fields *calendarFields, overflow string) isoDate
	monthDayToISOReferenceDate(fields *calendarFields, overflow string) isoDate
	dateAdd(date isoDate, dur dateDuration, overflow string) isoDate
	dateUntil(one, two isoDate, largestUnit string) dateDuration
	extraFields(fields []string) []string
	fieldKeysToIgnore(keys []string) []string
}

// calendarImpls holds the implementation of each calendar.
var calendarImpls map[string]calendarImpl

func init() {
	calendarImpls = map[string]calendarImpl{"iso8601": isoCalendar{}}
	for _, helper := range []calHelper{
		newHebrewHelper(),
		newPersianHelper(),
		newEthiopicHelper(),
		newEthioaaHelper(),
		newCopticHelper(),
		newChineseHelper(),
		newDangiHelper(),
		newRocHelper(),
		newIndianHelper(),
		newBuddhistHelper(),
		newGregoryHelper(),
		newJapaneseHelper(),
		newIslamicUmalquraHelper(),
		newIslamicTblaHelper(),
		newIslamicCivilHelper(),
		newIslamicCcHelper(),
	} {
		// Construct a new nonIsoCalendar instance with the given helper implementation that contains
		// per-calendar logic.
		calendarImpls[helper.base().id] = &nonIsoCalendar{helper: helper, b: helper.base()}
	}
}

// calendarImplForID returns the implementation of a built-in calendar, nil for any other identifier.
func calendarImplForID(id string) calendarImpl { return calendarImpls[id] }
