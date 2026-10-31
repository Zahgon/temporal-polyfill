package temporal

// This file holds the property bags and option structs of the public API. Every field is a *Value:
// a nil field is an absent property, V(x) is a present one. A struct can be passed by value or by
// pointer.

// PlainDateBag is a property bag describing a PlainDate.
type PlainDateBag struct {
	Year      *Value
	Month     *Value
	MonthCode *Value
	Day       *Value
	Era       *Value
	EraYear   *Value
	Calendar  *Value
}

// PlainTimeBag is a property bag describing a PlainTime.
type PlainTimeBag struct {
	Hour        *Value
	Minute      *Value
	Second      *Value
	Millisecond *Value
	Microsecond *Value
	Nanosecond  *Value
}

// PlainDateTimeBag is a property bag describing a PlainDateTime.
type PlainDateTimeBag struct {
	Year        *Value
	Month       *Value
	MonthCode   *Value
	Day         *Value
	Era         *Value
	EraYear     *Value
	Calendar    *Value
	Hour        *Value
	Minute      *Value
	Second      *Value
	Millisecond *Value
	Microsecond *Value
	Nanosecond  *Value
}

// PlainYearMonthBag is a property bag describing a PlainYearMonth.
type PlainYearMonthBag struct {
	Year      *Value
	Month     *Value
	MonthCode *Value
	Era       *Value
	EraYear   *Value
	Calendar  *Value
}

// PlainMonthDayBag is a property bag describing a PlainMonthDay.
type PlainMonthDayBag struct {
	Year      *Value
	Month     *Value
	MonthCode *Value
	Day       *Value
	Era       *Value
	EraYear   *Value
	Calendar  *Value
}

// ZonedDateTimeBag is a property bag describing a ZonedDateTime.
type ZonedDateTimeBag struct {
	Year        *Value
	Month       *Value
	MonthCode   *Value
	Day         *Value
	Era         *Value
	EraYear     *Value
	Calendar    *Value
	Hour        *Value
	Minute      *Value
	Second      *Value
	Millisecond *Value
	Microsecond *Value
	Nanosecond  *Value
	Offset      *Value
	TimeZone    *Value
}

// DurationBag is a property bag describing a Duration.
type DurationBag struct {
	Years        *Value
	Months       *Value
	Weeks        *Value
	Days         *Value
	Hours        *Value
	Minutes      *Value
	Seconds      *Value
	Milliseconds *Value
	Microseconds *Value
	Nanoseconds  *Value
}

// ZonedDateTimeConversion is the object form of the argument of PlainDate.ToZonedDateTime.
// PlainTime defaults to the start of the day.
type ZonedDateTimeConversion struct {
	TimeZone  *Value
	PlainTime *Value
}

// OverflowOptions selects how out-of-range date and time fields are handled.
type OverflowOptions struct {
	// Overflow is "constrain" or "reject"; the default is "constrain".
	Overflow *Value
}

// DisambiguationOptions selects how an ambiguous or skipped wall-clock time resolves.
type DisambiguationOptions struct {
	// Disambiguation is "compatible", "earlier", "later" or "reject"; the default is "compatible".
	Disambiguation *Value
}

// ZonedDateTimeFromOptions holds the options of ZonedDateTimeFrom.
type ZonedDateTimeFromOptions struct {
	// Disambiguation is "compatible", "earlier", "later" or "reject"; the default is "compatible".
	Disambiguation *Value
	// Offset is "use", "prefer", "ignore" or "reject"; the default is "reject".
	Offset *Value
	// Overflow is "constrain" or "reject"; the default is "constrain".
	Overflow *Value
}

// ZonedDateTimeWithOptions holds the options of ZonedDateTime.With.
type ZonedDateTimeWithOptions struct {
	// Disambiguation is "compatible", "earlier", "later" or "reject"; the default is "compatible".
	Disambiguation *Value
	// Offset is "use", "prefer", "ignore" or "reject"; the default is "prefer".
	Offset *Value
	// Overflow is "constrain" or "reject"; the default is "constrain".
	Overflow *Value
}

// DifferenceOptions holds the options of Until and Since.
type DifferenceOptions struct {
	// LargestUnit is the largest unit of the result; the default is "auto".
	LargestUnit *Value
	// SmallestUnit is the unit to round to; the default is "nanosecond".
	SmallestUnit *Value
	// RoundingIncrement is the granularity to round to, in units of SmallestUnit; the default is 1.
	RoundingIncrement *Value
	// RoundingMode is the rounding mode; the default is "trunc".
	RoundingMode *Value
}

// RoundOptions holds the options of Round on the Plain types, Instant and ZonedDateTime.
type RoundOptions struct {
	// SmallestUnit is the unit to round to; it is required.
	SmallestUnit *Value
	// RoundingIncrement is the granularity to round to, in units of SmallestUnit; the default is 1.
	RoundingIncrement *Value
	// RoundingMode is the rounding mode; the default is "halfExpand".
	RoundingMode *Value
}

// DurationRoundOptions holds the options of Duration.Round. At least one of SmallestUnit and
// LargestUnit is required.
type DurationRoundOptions struct {
	// LargestUnit is the largest unit of the result; the default is "auto", the larger of the duration's largest unit and SmallestUnit.
	LargestUnit *Value
	// SmallestUnit is the unit to round to; the default is "nanosecond".
	SmallestUnit *Value
	// RoundingIncrement is the granularity to round to, in units of SmallestUnit; the default is 1.
	RoundingIncrement *Value
	// RoundingMode is the rounding mode; the default is "halfExpand".
	RoundingMode *Value
	// RelativeTo is the starting point for calendar units: a PlainDate, a PlainDateTime, a ZonedDateTime, a bag or a string; the default is none.
	RelativeTo *Value
}

// DurationTotalOptions holds the options of Duration.Total.
type DurationTotalOptions struct {
	// Unit is the unit of the total; it is required.
	Unit *Value
	// RelativeTo is the starting point for calendar units: a PlainDate, a PlainDateTime, a ZonedDateTime, a bag or a string; the default is none.
	RelativeTo *Value
}

// DurationCompareOptions holds the options of DurationCompare.
type DurationCompareOptions struct {
	// RelativeTo is the starting point for calendar units: a PlainDate, a PlainDateTime, a ZonedDateTime, a bag or a string; the default is none.
	RelativeTo *Value
}

// InstantToStringOptions holds the options of Instant.ToString.
type InstantToStringOptions struct {
	// FractionalSecondDigits is a number from 0 to 9 or "auto"; the default is "auto".
	FractionalSecondDigits *Value
	// RoundingMode is the rounding mode; the default is "trunc".
	RoundingMode *Value
	// SmallestUnit overrides FractionalSecondDigits when present; the default is none.
	SmallestUnit *Value
	// TimeZone is the zone the string is written in; the default is UTC, written with a Z.
	TimeZone *Value
}

// PlainDateToStringOptions holds the options of PlainDate.ToString.
type PlainDateToStringOptions struct {
	// CalendarName is "auto", "always", "never" or "critical"; the default is "auto".
	CalendarName *Value
}

// PlainTimeToStringOptions holds the options of PlainTime.ToString.
type PlainTimeToStringOptions struct {
	// FractionalSecondDigits is a number from 0 to 9 or "auto"; the default is "auto".
	FractionalSecondDigits *Value
	// RoundingMode is the rounding mode; the default is "trunc".
	RoundingMode *Value
	// SmallestUnit overrides FractionalSecondDigits when present; the default is none.
	SmallestUnit *Value
}

// PlainDateTimeToStringOptions holds the options of PlainDateTime.ToString.
type PlainDateTimeToStringOptions struct {
	// FractionalSecondDigits is a number from 0 to 9 or "auto"; the default is "auto".
	FractionalSecondDigits *Value
	// RoundingMode is the rounding mode; the default is "trunc".
	RoundingMode *Value
	// SmallestUnit overrides FractionalSecondDigits when present; the default is none.
	SmallestUnit *Value
	// CalendarName is "auto", "always", "never" or "critical"; the default is "auto".
	CalendarName *Value
}

// ZonedDateTimeToStringOptions holds the options of ZonedDateTime.ToString.
type ZonedDateTimeToStringOptions struct {
	// FractionalSecondDigits is a number from 0 to 9 or "auto"; the default is "auto".
	FractionalSecondDigits *Value
	// RoundingMode is the rounding mode; the default is "trunc".
	RoundingMode *Value
	// SmallestUnit overrides FractionalSecondDigits when present; the default is none.
	SmallestUnit *Value
	// CalendarName is "auto", "always", "never" or "critical"; the default is "auto".
	CalendarName *Value
	// Offset is "auto" or "never"; the default is "auto".
	Offset *Value
	// TimeZoneName is "auto", "never" or "critical"; the default is "auto".
	TimeZoneName *Value
}

// DurationToStringOptions holds the options of Duration.ToString.
type DurationToStringOptions struct {
	// FractionalSecondDigits is a number from 0 to 9 or "auto"; the default is "auto".
	FractionalSecondDigits *Value
	// RoundingMode is the rounding mode; the default is "trunc".
	RoundingMode *Value
	// SmallestUnit overrides FractionalSecondDigits when present; the default is none.
	SmallestUnit *Value
}

// PlainYearMonthToStringOptions holds the options of PlainYearMonth.ToString.
type PlainYearMonthToStringOptions struct {
	// CalendarName is "auto", "always", "never" or "critical"; the default is "auto".
	CalendarName *Value
}

// PlainMonthDayToStringOptions holds the options of PlainMonthDay.ToString.
type PlainMonthDayToStringOptions struct {
	// CalendarName is "auto", "always", "never" or "critical"; the default is "auto".
	CalendarName *Value
}

// TransitionOptions holds the options of ZonedDateTime.GetTimeZoneTransition.
type TransitionOptions struct {
	// Direction is "next" or "previous"; it is required.
	Direction *Value
}

// DateTimeFormatOptions holds the options of NewDateTimeFormat and of ToLocaleString on the types
// that format through it. Every field defaults to the locale's own choice, as Intl.DateTimeFormat
// does.
type DateTimeFormatOptions struct {
	LocaleMatcher          *Value
	Calendar               *Value
	NumberingSystem        *Value
	Hour12                 *Value
	HourCycle              *Value
	TimeZone               *Value
	Weekday                *Value
	Era                    *Value
	Year                   *Value
	Month                  *Value
	Day                    *Value
	DayPeriod              *Value
	Hour                   *Value
	Minute                 *Value
	Second                 *Value
	FractionalSecondDigits *Value
	TimeZoneName           *Value
	FormatMatcher          *Value
	DateStyle              *Value
	TimeStyle              *Value
}

// DurationFormatOptions holds the options of Duration.ToLocaleString. Every field defaults as
// Intl.DurationFormat defaults it; Style defaults to "short".
type DurationFormatOptions struct {
	LocaleMatcher       *Value
	NumberingSystem     *Value
	Style               *Value
	Years               *Value
	YearsDisplay        *Value
	Months              *Value
	MonthsDisplay       *Value
	Weeks               *Value
	WeeksDisplay        *Value
	Days                *Value
	DaysDisplay         *Value
	Hours               *Value
	HoursDisplay        *Value
	Minutes             *Value
	MinutesDisplay      *Value
	Seconds             *Value
	SecondsDisplay      *Value
	Milliseconds        *Value
	MillisecondsDisplay *Value
	Microseconds        *Value
	MicrosecondsDisplay *Value
	Nanoseconds         *Value
	NanosecondsDisplay  *Value
	FractionalDigits    *Value
}
