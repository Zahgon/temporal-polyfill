package temporal

import (
	"math/big"
	"sort"
)

// calendarImplForObj is calendarImplForObj; temporalObj is one of the classes with a calendar.
func calendarImplForObj(temporalObj any) calendarImpl {
	calendar, _ := calendarSlot(temporalObj)
	return calendarImplForID(calendar)
}

// isoDateToFields is ISODateToFields. The optional argument is the type, by default "date".
func isoDateToFields(calendar string, date isoDate, typeParam ...string) *calendarFields {
	typ := "date"
	if len(typeParam) > 0 {
		typ = typeParam[0]
	}
	fields := &calendarFields{}
	impl := calendarImplForID(calendar)
	calendarDate := impl.isoToDate(date, reqYear|reqMonthCode|reqDay)

	monthCode := calendarDate.monthCode
	fields.monthCode = &monthCode
	if typ == "month-day" || typ == "date" {
		day := calendarDate.day
		fields.day = &day
	}
	if typ == "year-month" || typ == "date" {
		year := calendarDate.year
		fields.year = &year
	}
	return fields
}

// prepareCalendarFields is PrepareCalendarFields. requiredFields is a []string or "partial".
func prepareCalendarFields(calendar string, bagParam any, calendarFieldNames, nonCalendarFieldNames []string, requiredFields any) *calendarFields {
	bag := unwrap(bagParam)
	extraFieldNames := calendarImplForID(calendar).extraFields(calendarFieldNames)
	fields := make([]string, 0, len(calendarFieldNames)+len(nonCalendarFieldNames)+len(extraFieldNames))
	fields = append(fields, calendarFieldNames...)
	fields = append(fields, nonCalendarFieldNames...)
	fields = append(fields, extraFieldNames...)
	result := &calendarFields{}
	found := false
	sort.Strings(fields)
	required, notPartial := requiredFields.([]string)
	for _, property := range fields {
		value := objGet(bag, property)
		if value != nil {
			found = true
			castBuiltinField(result, property, value)
		} else if notPartial {
			if containsString(required, property) {
				throwType("required property '" + property + "' missing or undefined")
			}
			setBuiltinDefault(result, property)
		}
	}
	if !notPartial && !found {
		throwType("no supported properties found")
	}
	return result
}

// partialTimeRecord is what ToTemporalTimeRecord returns; a nil field is undefined.
type partialTimeRecord struct {
	hour, minute, second, millisecond, microsecond, nanosecond *float64
}

// timeRecord reads a complete record.
func (p partialTimeRecord) timeRecord() timeRecord {
	return timeRecord{
		hour:        undefinedToNaN(p.hour),
		minute:      undefinedToNaN(p.minute),
		second:      undefinedToNaN(p.second),
		millisecond: undefinedToNaN(p.millisecond),
		microsecond: undefinedToNaN(p.microsecond),
		nanosecond:  undefinedToNaN(p.nanosecond),
	}
}

// toTemporalTimeRecord is ToTemporalTimeRecord. The optional argument is the completeness, by default
// "complete".
func toTemporalTimeRecord(bagParam any, completenessParam ...string) partialTimeRecord {
	bag := unwrap(bagParam)
	completeness := "complete"
	if len(completenessParam) > 0 {
		completeness = completenessParam[0]
	}
	// NOTE: Field order is sorted to make the sort in PrepareTemporalFields more efficient.
	var result partialTimeRecord
	found := false
	for _, entry := range []struct {
		field  string
		target **float64
	}{
		{"hour", &result.hour},
		{"microsecond", &result.microsecond},
		{"millisecond", &result.millisecond},
		{"minute", &result.minute},
		{"nanosecond", &result.nanosecond},
		{"second", &result.second},
	} {
		value := objGet(bag, entry.field)
		if value != nil {
			integer := toIntegerWithTruncation(value)
			*entry.target = &integer
			found = true
		} else if completeness == "complete" {
			zero := 0.0
			*entry.target = &zero
		}
	}
	if !found {
		throwType("invalid time-like")
	}
	return result
}

// toTemporalDate is ToTemporalDate. The optional argument is options.
func toTemporalDate(itemParam any, optionsParam ...any) *PlainDate {
	item := unwrap(itemParam)
	options := optionalArg(optionsParam)
	if isObject(item) {
		switch v := item.(type) {
		case *PlainDate:
			getTemporalOverflowOption(getOptionsObject(options))
			return createTemporalDate(v.isoDate, v.calendar)
		case *ZonedDateTime:
			dateTime := getISODateTimeFor(v.timeZone, v.epochNs)
			getTemporalOverflowOption(getOptionsObject(options)) // validate and ignore
			return createTemporalDate(dateTime.isoDate, v.calendar)
		case *PlainDateTime:
			getTemporalOverflowOption(getOptionsObject(options)) // validate and ignore
			return createTemporalDate(v.isoDateTime.isoDate, v.calendar)
		}
		calendar := getTemporalCalendarIdentifierWithISODefault(item)
		fields := prepareCalendarFields(calendar, item, []string{"year", "month", "monthCode", "day"}, []string{}, []string{})
		overflow := getTemporalOverflowOption(getOptionsObject(options))
		date := calendarDateFromFields(calendar, fields, overflow)
		return createTemporalDate(date, calendar)
	}
	parsed := parseTemporalDateString(requireString(item))
	if parsed.z {
		throwRange("Z designator not supported for PlainDate")
	}
	calendar := parsed.calendar
	if calendar == "" {
		calendar = "iso8601"
	}
	calendar = canonicalizeCalendar(calendar)
	getTemporalOverflowOption(getOptionsObject(options)) // validate and ignore
	return createTemporalDate(isoDate{year: parsed.year, month: parsed.month, day: parsed.day}, calendar)
}

// interpretTemporalDateTimeFields is InterpretTemporalDateTimeFields.
func interpretTemporalDateTimeFields(calendar string, fields *calendarFields, overflow string) isoDateTime {
	date := calendarDateFromFields(calendar, fields, overflow)
	time := regulateTime(
		undefinedToNaN(fields.hour),
		undefinedToNaN(fields.minute),
		undefinedToNaN(fields.second),
		undefinedToNaN(fields.millisecond),
		undefinedToNaN(fields.microsecond),
		undefinedToNaN(fields.nanosecond),
		overflow,
	)
	return combineISODateAndTimeRecord(date, time)
}

// toTemporalDateTime is ToTemporalDateTime. The optional argument is options.
func toTemporalDateTime(itemParam any, optionsParam ...any) *PlainDateTime {
	item := unwrap(itemParam)
	options := optionalArg(optionsParam)
	var date isoDate
	var time timeRecord
	var calendar string

	if isObject(item) {
		switch v := item.(type) {
		case *PlainDateTime:
			getTemporalOverflowOption(getOptionsObject(options))
			return createTemporalDateTime(v.isoDateTime, v.calendar)
		case *ZonedDateTime:
			dateTime := getISODateTimeFor(v.timeZone, v.epochNs)
			getTemporalOverflowOption(getOptionsObject(options))
			return createTemporalDateTime(dateTime, v.calendar)
		case *PlainDate:
			getTemporalOverflowOption(getOptionsObject(options))
			return createTemporalDateTime(combineISODateAndTimeRecord(v.isoDate, midnightTimeRecord()), v.calendar)
		}

		calendar = getTemporalCalendarIdentifierWithISODefault(item)
		fields := prepareCalendarFields(
			calendar,
			item,
			[]string{"year", "month", "monthCode", "day"},
			[]string{"hour", "minute", "second", "millisecond", "microsecond", "nanosecond"},
			[]string{},
		)
		overflow := getTemporalOverflowOption(getOptionsObject(options))
		interpreted := interpretTemporalDateTimeFields(calendar, fields, overflow)
		date, time = interpreted.isoDate, interpreted.time
	} else {
		parsed := parseTemporalDateTimeString(requireString(item))
		if parsed.z {
			throwRange("Z designator not supported for PlainDateTime")
		}
		time = parsed.time
		if parsed.startOfDay {
			time = midnightTimeRecord()
		}
		rejectDateTime(
			parsed.year,
			parsed.month,
			parsed.day,
			time.hour,
			time.minute,
			time.second,
			time.millisecond,
			time.microsecond,
			time.nanosecond,
		)
		calendar = parsed.calendar
		if calendar == "" {
			calendar = "iso8601"
		}
		calendar = canonicalizeCalendar(calendar)
		getTemporalOverflowOption(getOptionsObject(options))
		date = isoDate{year: parsed.year, month: parsed.month, day: parsed.day}
	}
	return createTemporalDateTime(combineISODateAndTimeRecord(date, time), calendar)
}

// toTemporalDuration is ToTemporalDuration.
func toTemporalDuration(itemParam any) *Duration {
	item := unwrap(itemParam)
	if d, ok := item.(*Duration); ok {
		return constructDuration(d.years, d.months, d.weeks, d.days, d.hours, d.minutes, d.seconds, d.milliseconds, d.microseconds, d.nanoseconds)
	}
	if !isObject(item) {
		return parseTemporalDurationString(requireString(item))
	}
	partial := toTemporalPartialDurationRecord(item)
	return constructDuration(
		floatOr(partial.years, 0),
		floatOr(partial.months, 0),
		floatOr(partial.weeks, 0),
		floatOr(partial.days, 0),
		floatOr(partial.hours, 0),
		floatOr(partial.minutes, 0),
		floatOr(partial.seconds, 0),
		floatOr(partial.milliseconds, 0),
		floatOr(partial.microseconds, 0),
		floatOr(partial.nanoseconds, 0),
	)
}

// toTemporalInstant is ToTemporalInstant.
func toTemporalInstant(itemParam any) *Instant {
	var item any = unwrap(itemParam)
	if isObject(item) {
		switch v := item.(type) {
		case *Instant:
			return createTemporalInstant(v.epochNs)
		case *ZonedDateTime:
			return createTemporalInstant(v.epochNs)
		}
		item = toPrimitiveRequireString(item)
	}
	parsed := parseTemporalInstantString(requireString(item))
	assert(!parsed.startOfDay, "Instant string must include a time")
	time := parsed.time

	// ParseTemporalInstantString ensures that either `z` is true or or `offset` is non-undefined
	offsetNanoseconds := 0.0
	if !parsed.z {
		offsetNanoseconds = parseDateTimeUTCOffset(parsed.offset)
	}
	balanced := balanceISODateTime(
		parsed.year,
		parsed.month,
		parsed.day,
		time.hour,
		time.minute,
		time.second,
		time.millisecond,
		time.microsecond,
		time.nanosecond-offsetNanoseconds,
	)
	checkISODaysRange(balanced.isoDate)
	epochNanoseconds := getUTCEpochNanoseconds(balanced)
	return createTemporalInstant(epochNanoseconds)
}

// toTemporalMonthDay is ToTemporalMonthDay. The optional argument is options.
func toTemporalMonthDay(itemParam any, optionsParam ...any) *PlainMonthDay {
	item := unwrap(itemParam)
	options := optionalArg(optionsParam)
	if isObject(item) {
		if v, ok := item.(*PlainMonthDay); ok {
			getTemporalOverflowOption(getOptionsObject(options))
			return createTemporalMonthDay(v.isoDate, v.calendar)
		}
		calendar, hasSlot := calendarSlot(item)
		if !hasSlot {
			calendarLike := objGet(item, "calendar")
			if calendarLike == nil {
				calendarLike = "iso8601"
			}
			calendar = toTemporalCalendarIdentifier(calendarLike)
		}
		fields := prepareCalendarFields(calendar, item, []string{"year", "month", "monthCode", "day"}, []string{}, []string{})
		overflow := getTemporalOverflowOption(getOptionsObject(options))
		date := calendarMonthDayFromFields(calendar, fields, overflow)
		return createTemporalMonthDay(date, calendar)
	}

	parsed := parseTemporalMonthDayString(requireString(item))
	calendar := parsed.calendar
	if calendar == "" {
		calendar = "iso8601"
	}
	calendar = canonicalizeCalendar(calendar)

	getTemporalOverflowOption(getOptionsObject(options))
	if calendar == "iso8601" {
		const isoCalendarReferenceYear = 1972 // First leap year after Unix epoch
		return createTemporalMonthDay(isoDate{year: isoCalendarReferenceYear, month: parsed.month, day: parsed.day}, calendar)
	}
	assertExists(parsed.referenceISOYear)
	date := isoDate{year: *parsed.referenceISOYear, month: parsed.month, day: parsed.day}
	rejectDateRange(date)
	result := isoDateToFields(calendar, date, "month-day")
	date = calendarMonthDayFromFields(calendar, result, "constrain")
	return createTemporalMonthDay(date, calendar)
}

// toTemporalTime is ToTemporalTime. The optional argument is options.
func toTemporalTime(itemParam any, optionsParam ...any) *PlainTime {
	item := unwrap(itemParam)
	options := optionalArg(optionsParam)
	var time timeRecord
	if isObject(item) {
		switch v := item.(type) {
		case *PlainTime:
			getTemporalOverflowOption(getOptionsObject(options))
			return createTemporalTime(v.time)
		case *PlainDateTime:
			getTemporalOverflowOption(getOptionsObject(options))
			return createTemporalTime(v.isoDateTime.time)
		case *ZonedDateTime:
			dateTime := getISODateTimeFor(v.timeZone, v.epochNs)
			getTemporalOverflowOption(getOptionsObject(options))
			return createTemporalTime(dateTime.time)
		}
		record := toTemporalTimeRecord(item).timeRecord()
		overflow := getTemporalOverflowOption(getOptionsObject(options))
		time = regulateTime(record.hour, record.minute, record.second, record.millisecond, record.microsecond, record.nanosecond, overflow)
	} else {
		time = parseTemporalTimeString(requireString(item)).time
		getTemporalOverflowOption(getOptionsObject(options))
	}
	return createTemporalTime(time)
}

// toTimeRecordOrMidnight is ToTimeRecordOrMidnight.
func toTimeRecordOrMidnight(item any) timeRecord {
	if isUndefined(item) {
		return midnightTimeRecord()
	}
	return toTemporalTime(item).time
}

// toTemporalYearMonth is ToTemporalYearMonth. The optional argument is options.
func toTemporalYearMonth(itemParam any, optionsParam ...any) *PlainYearMonth {
	item := unwrap(itemParam)
	options := optionalArg(optionsParam)
	if isObject(item) {
		if v, ok := item.(*PlainYearMonth); ok {
			getTemporalOverflowOption(getOptionsObject(options))
			return createTemporalYearMonth(v.isoDate, v.calendar)
		}
		calendar := getTemporalCalendarIdentifierWithISODefault(item)
		fields := prepareCalendarFields(calendar, item, []string{"year", "month", "monthCode"}, []string{}, []string{})
		overflow := getTemporalOverflowOption(getOptionsObject(options))
		date := calendarYearMonthFromFields(calendar, fields, overflow)
		return createTemporalYearMonth(date, calendar)
	}

	parsed := parseTemporalYearMonthString(requireString(item))
	calendar := parsed.calendar
	if calendar == "" {
		calendar = "iso8601"
	}
	calendar = canonicalizeCalendar(calendar)

	getTemporalOverflowOption(getOptionsObject(options))
	date := isoDate{year: parsed.year, month: parsed.month, day: parsed.referenceISODay}
	rejectYearMonthRange(date)
	result := isoDateToFields(calendar, date, "year-month")
	date = calendarYearMonthFromFields(calendar, result, "constrain")
	return createTemporalYearMonth(date, calendar)
}

// interpretISODateTimeOffset is InterpretISODateTimeOffset. timeParam is a timeRecord or
// "start-of-day".
func interpretISODateTimeOffset(
	date isoDate,
	timeParam any,
	offsetBehaviour string,
	offsetNs float64,
	timeZone string,
	disambiguation string,
	offsetOpt string,
	matchMinute bool,
) *big.Int {
	// start-of-day signifies that we had a string such as YYYY-MM-DD[Zone]. It is
	// grammatically not possible to specify a UTC offset in that string, so the
	// behaviour collapses into ~WALL~, which is equivalent to offset: "ignore".
	if timeParam == "start-of-day" {
		assert(offsetBehaviour == "wall", "offset cannot be provided in YYYY-MM-DD[Zone] string")
		assert(offsetNs == 0, "offset cannot be provided in YYYY-MM-DD[Zone] string")
		return getStartOfDay(timeZone, date)
	}
	var time timeRecord
	switch v := timeParam.(type) {
	case timeRecord:
		time = v
	case *timeRecord:
		time = *v
	}

	dt := combineISODateAndTimeRecord(date, time)

	if offsetBehaviour == "wall" || offsetOpt == "ignore" {
		// Simple case: ISO string without a TZ offset (or caller wants to ignore
		// the offset), so just convert DateTime to Instant in the given time zone
		return getEpochNanosecondsFor(timeZone, dt, disambiguation)
	}

	// The caller wants the offset to always win ('use') OR the caller is OK
	// with the offset winning ('prefer' or 'reject') as long as it's valid
	// for this timezone and date/time.
	if offsetBehaviour == "exact" || offsetOpt == "use" {
		// Calculate the instant for the input's date/time and offset
		balanced := balanceISODateTime(
			date.year,
			date.month,
			date.day,
			time.hour,
			time.minute,
			time.second,
			time.millisecond,
			time.microsecond,
			time.nanosecond-offsetNs,
		)
		checkISODaysRange(balanced.isoDate)
		epochNs := getUTCEpochNanoseconds(balanced)
		validateEpochNanoseconds(epochNs)
		return epochNs
	}

	checkISODaysRange(date)
	utcEpochNs := getUTCEpochNanoseconds(dt)

	// "prefer" or "reject"
	possibleEpochNs := getPossibleEpochNanoseconds(timeZone, dt)
	for _, candidate := range possibleEpochNs {
		candidateOffset := bigToNumber(new(big.Int).Sub(utcEpochNs, candidate))
		roundedCandidateOffset := roundNumberToIncrement(candidateOffset, 60e9, "halfExpand")
		if candidateOffset == offsetNs || (matchMinute && roundedCandidateOffset == offsetNs) {
			return candidate
		}
	}

	// the user-provided offset doesn't match any instants for this time
	// zone and date/time.
	if offsetOpt == "reject" {
		offsetStr := formatUTCOffsetNanoseconds(offsetNs)
		dtStr := isoDateTimeToString(dt, "iso8601", "auto")
		throwRange("Offset " + offsetStr + " is invalid for " + dtStr + " in " + timeZone)
	}
	// fall through: offsetOpt === 'prefer', but the offset doesn't match
	// so fall back to use the time zone instead.
	return disambiguatePossibleEpochNanoseconds(possibleEpochNs, timeZone, dt, disambiguation)
}

// toTemporalZonedDateTime is ToTemporalZonedDateTime. The optional argument is options.
func toTemporalZonedDateTime(itemParam any, optionsParam ...any) *ZonedDateTime {
	item := unwrap(itemParam)
	options := optionalArg(optionsParam)
	var date isoDate
	var time any
	var timeZone, offset, calendar string
	matchMinute := false
	hasUTCDesignator := false
	var disambiguation, offsetOpt string
	if isObject(item) {
		if v, ok := item.(*ZonedDateTime); ok {
			resolvedOptions := getOptionsObject(options)
			getTemporalDisambiguationOption(resolvedOptions) // validate and ignore
			getTemporalOffsetOption(resolvedOptions, "reject")
			getTemporalOverflowOption(resolvedOptions)
			return createTemporalZonedDateTime(v.epochNs, v.timeZone, v.calendar)
		}
		calendar = getTemporalCalendarIdentifierWithISODefault(item)
		fields := prepareCalendarFields(
			calendar,
			item,
			[]string{"year", "month", "monthCode", "day"},
			[]string{"hour", "minute", "second", "millisecond", "microsecond", "nanosecond", "offset", "timeZone"},
			[]string{"timeZone"},
		)
		if fields.offset != nil {
			offset = *fields.offset
		}
		timeZone = *fields.timeZone
		resolvedOptions := getOptionsObject(options)
		disambiguation = getTemporalDisambiguationOption(resolvedOptions)
		offsetOpt = getTemporalOffsetOption(resolvedOptions, "reject")
		overflow := getTemporalOverflowOption(resolvedOptions)
		interpreted := interpretTemporalDateTimeFields(calendar, fields, overflow)
		date, time = interpreted.isoDate, interpreted.time
	} else {
		parsed := parseTemporalZonedDateTimeString(requireString(item))
		time, offset, calendar = parsed.timeOrStartOfDay(), parsed.offset, parsed.calendar
		timeZone = toTemporalTimeZoneIdentifier(parsed.tzAnnotation)
		if parsed.z {
			hasUTCDesignator = true
		}
		if calendar == "" {
			calendar = "iso8601"
		}
		calendar = canonicalizeCalendar(calendar)
		// Allow imprecise offset matching unless the provided offset is precise
		matchMinute = true
		if offset != "" {
			offsetParseResult := reExec(offsetWithPartsRe, offset)
			assert(offsetParseResult != nil, "offset string must re-parse")
			offsetSecondsPart := firstDefined(offsetParseResult.group("offsetSecondSep"), offsetParseResult.group("offsetSecondNoSep"))
			if offsetSecondsPart != "" {
				matchMinute = false
			}
		}
		resolvedOptions := getOptionsObject(options)
		disambiguation = getTemporalDisambiguationOption(resolvedOptions)
		offsetOpt = getTemporalOffsetOption(resolvedOptions, "reject")
		getTemporalOverflowOption(resolvedOptions) // validate and ignore
		date = isoDate{year: parsed.year, month: parsed.month, day: parsed.day}
	}
	offsetBehaviour := "option"
	if hasUTCDesignator {
		offsetBehaviour = "exact"
	} else if offset == "" {
		offsetBehaviour = "wall"
	}
	offsetNs := 0.0
	if offsetBehaviour == "option" {
		offsetNs = parseDateTimeUTCOffset(offset)
	}
	epochNanoseconds := interpretISODateTimeOffset(
		date,
		time,
		offsetBehaviour,
		offsetNs,
		timeZone,
		disambiguation,
		offsetOpt,
		matchMinute,
	)
	return createTemporalZonedDateTime(epochNanoseconds, timeZone, calendar)
}

// createTemporalDate is CreateTemporalDate; the range check of CreateTemporalDateSlots comes first.
func createTemporalDate(date isoDate, calendar string) *PlainDate {
	rejectDateRange(date)
	return &PlainDate{isoDate: date, calendar: calendar}
}

// createTemporalDateTime is CreateTemporalDateTime.
func createTemporalDateTime(dateTime isoDateTime, calendar string) *PlainDateTime {
	rejectDateTimeRange(dateTime)
	return &PlainDateTime{isoDateTime: dateTime, calendar: calendar}
}

// createTemporalMonthDay is CreateTemporalMonthDay.
func createTemporalMonthDay(date isoDate, calendar string) *PlainMonthDay {
	rejectDateRange(date)
	return &PlainMonthDay{isoDate: date, calendar: calendar}
}

// createTemporalTime is CreateTemporalTime.
func createTemporalTime(time timeRecord) *PlainTime {
	return &PlainTime{time: time}
}

// createTemporalYearMonth is CreateTemporalYearMonth.
func createTemporalYearMonth(date isoDate, calendar string) *PlainYearMonth {
	rejectYearMonthRange(date)
	return &PlainYearMonth{isoDate: date, calendar: calendar}
}

// createTemporalInstant is CreateTemporalInstant.
func createTemporalInstant(epochNanoseconds *big.Int) *Instant {
	validateEpochNanoseconds(epochNanoseconds)
	return &Instant{epochNs: epochNanoseconds}
}

// createTemporalZonedDateTime is CreateTemporalZonedDateTime. The optional argument is the calendar,
// by default "iso8601".
func createTemporalZonedDateTime(epochNanoseconds *big.Int, timeZone string, calendarParam ...string) *ZonedDateTime {
	calendar := "iso8601"
	if len(calendarParam) > 0 {
		calendar = calendarParam[0]
	}
	validateEpochNanoseconds(epochNanoseconds)
	return &ZonedDateTime{epochNs: epochNanoseconds, timeZone: timeZone, calendar: calendar}
}

func calendarFieldKeysPresent(fields *calendarFields) []string {
	present := make([]string, 0, len(calendarFieldKeys))
	for _, key := range calendarFieldKeys {
		if fieldIsPresent(fields, key) {
			present = append(present, key)
		}
	}
	return present
}

// calendarMergeFields is CalendarMergeFields.
func calendarMergeFields(calendar string, fields, additionalFields *calendarFields) *calendarFields {
	additionalKeys := calendarFieldKeysPresent(additionalFields)
	overriddenKeys := calendarImplForID(calendar).fieldKeysToIgnore(additionalKeys)
	merged := &calendarFields{}
	fieldsKeys := calendarFieldKeysPresent(fields)
	for _, key := range calendarFieldKeys {
		if containsString(fieldsKeys, key) && !containsString(overriddenKeys, key) {
			copyField(merged, fields, key)
		}
		if containsString(additionalKeys, key) {
			copyField(merged, additionalFields, key)
		}
	}
	return merged
}

// calendarDateAdd is CalendarDateAdd.
func calendarDateAdd(calendar string, date isoDate, duration dateDuration, overflow string) isoDate {
	result := calendarImplForID(calendar).dateAdd(date, duration, overflow)
	rejectDateRange(result)
	return result
}

func calendarDateUntil(calendar string, date, isoOtherDate isoDate, largestUnit string) dateDuration {
	if compareISODate(date, isoOtherDate) == 0 {
		return dateDuration{}
	}
	return calendarImplForID(calendar).dateUntil(date, isoOtherDate, largestUnit)
}

// toTemporalCalendarIdentifier is ToTemporalCalendarIdentifier.
func toTemporalCalendarIdentifier(calendarLikeParam any) string {
	calendarLike := unwrap(calendarLikeParam)
	if isObject(calendarLike) {
		if calendar, ok := calendarSlot(calendarLike); ok {
			return calendar
		}
	}
	identifier := requireString(calendarLike)
	var canonical string
	if !jsCatch(func() {
		// Fast path: identifier is a calendar type, no ISO string parsing needed
		canonical = canonicalizeCalendar(identifier)
	}) {
		return canonical
	}
	var calendar string
	if jsCatch(func() { calendar = parseISODateTime(identifier).calendar }) {
		if jsCatch(func() { calendar = parseTemporalTimeString(identifier).calendar }) {
			if jsCatch(func() { calendar = parseTemporalYearMonthString(identifier).calendar }) {
				calendar = parseTemporalMonthDayString(identifier).calendar
			}
		}
	}
	if calendar == "" {
		calendar = "iso8601"
	}
	return canonicalizeCalendar(calendar)
}

func getTemporalCalendarIdentifierWithISODefault(itemParam any) string {
	item := unwrap(itemParam)
	if calendar, ok := calendarSlot(item); ok {
		return calendar
	}
	calendarLike := objGet(item, "calendar")
	if calendarLike == nil {
		return "iso8601"
	}
	return toTemporalCalendarIdentifier(calendarLike)
}

// calendarEquals is CalendarEquals.
func calendarEquals(one, two string) bool {
	return canonicalizeCalendar(one) == canonicalizeCalendar(two)
}

// calendarDateFromFields is CalendarDateFromFields.
func calendarDateFromFields(calendar string, fields *calendarFields, overflow string) isoDate {
	impl := calendarImplForID(calendar)
	impl.resolveFields(fields, "date")
	result := impl.dateToISO(fields, overflow)
	rejectDateRange(result)
	return result
}

// calendarYearMonthFromFields is CalendarYearMonthFromFields.
func calendarYearMonthFromFields(calendar string, fields *calendarFields, overflow string) isoDate {
	impl := calendarImplForID(calendar)
	impl.resolveFields(fields, "year-month")
	day := 1.0
	fields.day = &day
	result := impl.dateToISO(fields, overflow)
	rejectYearMonthRange(result)
	return result
}

// calendarMonthDayFromFields is CalendarMonthDayFromFields.
func calendarMonthDayFromFields(calendar string, fields *calendarFields, overflow string) isoDate {
	impl := calendarImplForID(calendar)
	impl.resolveFields(fields, "month-day")
	result := impl.monthDayToISOReferenceDate(fields, overflow)
	rejectDateRange(result)
	return result
}

// toTemporalTimeZoneIdentifier is ToTemporalTimeZoneIdentifier.
func toTemporalTimeZoneIdentifier(temporalTimeZoneLikeParam any) string {
	temporalTimeZoneLike := unwrap(temporalTimeZoneLikeParam)
	if v, ok := temporalTimeZoneLike.(*ZonedDateTime); ok {
		return v.timeZone
	}
	timeZoneString := requireString(temporalTimeZoneLike)
	if timeZoneString == "UTC" {
		return "UTC" // UTC fast path
	}

	parsed := parseTemporalTimeZoneString(timeZoneString)
	if parsed.offsetMinutes != nil {
		return formatOffsetTimeZoneIdentifier(*parsed.offsetMinutes)
	}
	// if offsetMinutes is undefined, then tzName must be present
	record := getAvailableNamedTimeZoneIdentifier(castExists(parsed.tzName))
	if record == nil {
		throwRange("Unrecognized time zone " + *parsed.tzName)
	}
	return record.identifier
}

// timeZoneEquals is TimeZoneEquals.
func timeZoneEquals(one, two string) bool {
	if one == two {
		return true
	}
	offsetMinutes1 := parseTimeZoneIdentifier(one).offsetMinutes
	offsetMinutes2 := parseTimeZoneIdentifier(two).offsetMinutes
	if offsetMinutes1 == nil && offsetMinutes2 == nil {
		// Calling GetAvailableNamedTimeZoneIdentifier is costly, so (unlike the
		// spec) the polyfill will early-return if one of them isn't recognized. Try
		// the second ID first because it's more likely to be unknown, because it
		// can come from the argument of TimeZone.p.equals as opposed to the first
		// ID which comes from the receiver.
		idRecord2 := getAvailableNamedTimeZoneIdentifier(two)
		assert(idRecord2 != nil, `"`+two+`" has an invalid time zone`)
		idRecord1 := getAvailableNamedTimeZoneIdentifier(one)
		assert(idRecord1 != nil, `"`+one+`" has an invalid time zone`)
		return idRecord1.primaryIdentifier == idRecord2.primaryIdentifier
	}
	if offsetMinutes1 == nil || offsetMinutes2 == nil {
		return false
	}
	return *offsetMinutes1 == *offsetMinutes2
}

// getOffsetNanosecondsFor is GetOffsetNanosecondsFor.
func getOffsetNanosecondsFor(timeZone string, epochNs *big.Int) float64 {
	offsetMinutes := parseTimeZoneIdentifier(timeZone).offsetMinutes
	if offsetMinutes != nil {
		return float64(*offsetMinutes * 60e9)
	}

	return getNamedTimeZoneOffsetNanoseconds(timeZone, epochNs)
}

// getISODateTimeFor is GetISODateTimeFor.
func getISODateTimeFor(timeZone string, epochNs *big.Int) isoDateTime {
	offsetNs := getOffsetNanosecondsFor(timeZone, epochNs)
	parts := getISOPartsFromEpoch(epochNs)
	date, time := parts.isoDate, parts.time
	return balanceISODateTime(date.year, date.month, date.day, time.hour, time.minute, time.second, time.millisecond, time.microsecond, time.nanosecond+offsetNs)
}

// getEpochNanosecondsFor is GetEpochNanosecondsFor.
func getEpochNanosecondsFor(timeZone string, dateTime isoDateTime, disambiguation string) *big.Int {
	possibleEpochNs := getPossibleEpochNanoseconds(timeZone, dateTime)
	return disambiguatePossibleEpochNanoseconds(possibleEpochNs, timeZone, dateTime, disambiguation)
}

// firstOrUndefined is list[0].
func firstOrUndefined(list []*big.Int) *big.Int {
	if len(list) == 0 {
		return nil
	}
	return list[0]
}

// TODO: See if this logic can be removed in favour of GetNamedTimeZoneEpochNanoseconds
func disambiguatePossibleEpochNanoseconds(possibleEpochNs []*big.Int, timeZone string, dateTime isoDateTime, disambiguation string) *big.Int {
	numInstants := len(possibleEpochNs)

	if numInstants == 1 {
		return possibleEpochNs[0]
	}
	if numInstants != 0 {
		switch disambiguation {
		case "compatible", "earlier":
			// 'compatible' means 'earlier' for "fall back" transitions
			return possibleEpochNs[0]
		case "later":
			return possibleEpochNs[numInstants-1]
		case "reject":
			throwRange("multiple instants found")
		}
	}

	if disambiguation == "reject" {
		throwRange("multiple instants found")
	}
	utcns := getUTCEpochNanoseconds(dateTime)

	dayBefore := new(big.Int).Sub(utcns, dayNanosJSBI)
	validateEpochNanoseconds(dayBefore)
	offsetBefore := getOffsetNanosecondsFor(timeZone, dayBefore)
	dayAfter := new(big.Int).Add(utcns, dayNanosJSBI)
	validateEpochNanoseconds(dayAfter)
	offsetAfter := getOffsetNanosecondsFor(timeZone, dayAfter)
	nanoseconds := offsetAfter - offsetBefore
	assert(nanoseconds <= dayNanos && -nanoseconds <= dayNanos, "UTC offset shift longer than 24 hours")

	switch disambiguation {
	case "earlier":
		duration := timeDurationFromComponents(0, 0, 0, 0, 0, -nanoseconds)
		earlierTime := addTime(dateTime.time, duration)
		earlierDate := addDaysToISODate(dateTime.isoDate, earlierTime.deltaDays)
		earlier := combineISODateAndTimeRecord(earlierDate, earlierTime)
		return firstOrUndefined(getPossibleEpochNanoseconds(timeZone, earlier))
	case "compatible", "later":
		// 'compatible' means 'later' for "spring forward" transitions
		duration := timeDurationFromComponents(0, 0, 0, 0, 0, nanoseconds)
		laterTime := addTime(dateTime.time, duration)
		laterDate := addDaysToISODate(dateTime.isoDate, laterTime.deltaDays)
		later := combineISODateAndTimeRecord(laterDate, laterTime)
		possible := getPossibleEpochNanoseconds(timeZone, later)
		if len(possible) == 0 {
			return nil
		}
		return possible[len(possible)-1]
	}
	return nil
}

func getPossibleEpochNanoseconds(timeZone string, dateTime isoDateTime) []*big.Int {
	// UTC fast path
	if timeZone == "UTC" {
		checkISODaysRange(dateTime.isoDate)
		epochNs := getUTCEpochNanoseconds(dateTime)
		validateEpochNanoseconds(epochNs)
		return []*big.Int{epochNs}
	}

	offsetMinutes := parseTimeZoneIdentifier(timeZone).offsetMinutes
	if offsetMinutes != nil {
		balanced := balanceISODateTime(
			dateTime.isoDate.year,
			dateTime.isoDate.month,
			dateTime.isoDate.day,
			dateTime.time.hour,
			dateTime.time.minute-*offsetMinutes,
			dateTime.time.second,
			dateTime.time.millisecond,
			dateTime.time.microsecond,
			dateTime.time.nanosecond,
		)
		checkISODaysRange(balanced.isoDate)
		epochNs := getUTCEpochNanoseconds(balanced)
		validateEpochNanoseconds(epochNs)
		return []*big.Int{epochNs}
	}

	return getNamedTimeZoneEpochNanoseconds(timeZone, dateTime)
}

// getStartOfDay is GetStartOfDay.
func getStartOfDay(timeZone string, date isoDate) *big.Int {
	dateTime := combineISODateAndTimeRecord(date, midnightTimeRecord())
	possibleEpochNs := getPossibleEpochNanoseconds(timeZone, dateTime)
	// If not a DST gap, return the single or earlier epochNs
	if len(possibleEpochNs) != 0 {
		return possibleEpochNs[0]
	}

	// Otherwise, 00:00:00 lies within a DST gap. Compute an epochNs that's
	// guaranteed to be before the transition
	assert(!isOffsetTimeZoneIdentifier(timeZone), "should only be reached with named time zone")

	utcns := getUTCEpochNanoseconds(dateTime)
	dayBefore := new(big.Int).Sub(utcns, dayNanosJSBI)
	validateEpochNanoseconds(dayBefore)
	transition := getNamedTimeZoneNextTransition(timeZone, dayBefore)
	assertExists(transition)
	return transition
}
