package temporal

import "math/big"

// isoDate is the ISO Date Record. Every JavaScript Number in this package stays a float64, so that
// precision loss and signed zero are the original's.
type isoDate struct{ year, month, day float64 }

// timeRecord is the Time Record. deltaDays is carried by the balancing operations.
type timeRecord struct {
	deltaDays, hour, minute, second, millisecond, microsecond, nanosecond float64
}

// isoDateTime is the ISO Date-Time Record.
type isoDateTime struct {
	isoDate isoDate
	time    timeRecord
}

// dateDuration is the Date Duration Record.
type dateDuration struct{ years, months, weeks, days float64 }

// internalDuration is the Internal Duration Record: a date duration and a normalized time duration.
type internalDuration struct {
	date dateDuration
	time *timeDuration
}

// calendarFields mirrors CalendarFieldsRecord. A nil field is undefined.
type calendarFields struct {
	era                                                        *string
	eraYear, year, month                                       *float64
	monthCode                                                  *string
	day                                                        *float64
	hour, minute, second, millisecond, microsecond, nanosecond *float64
	offset, timeZone                                           *string
	monthExtra                                                 *string
}

// calendarDate is what a calendar reports for an ISO date (CalendarDateRecord). era and eraYear are
// nil on calendars without eras; weekOfYear and yearOfWeek are nil where undefined.
type calendarDate struct {
	era                                               *string
	eraYear                                           *float64
	year, month                                       float64
	monthCode                                         string
	day, dayOfWeek, dayOfYear                         float64
	weekOfYear, yearOfWeek                            *float64
	daysInWeek, daysInMonth, daysInYear, monthsInYear float64
	inLeapYear                                        bool
}

// Instant is Temporal.Instant.
type Instant struct{ epochNs *big.Int }

// PlainDate is Temporal.PlainDate.
type PlainDate struct {
	isoDate  isoDate
	calendar string
}

// PlainDateTime is Temporal.PlainDateTime.
type PlainDateTime struct {
	isoDateTime isoDateTime
	calendar    string
}

// PlainTime is Temporal.PlainTime.
type PlainTime struct{ time timeRecord }

// PlainYearMonth is Temporal.PlainYearMonth.
type PlainYearMonth struct {
	isoDate  isoDate
	calendar string
}

// PlainMonthDay is Temporal.PlainMonthDay.
type PlainMonthDay struct {
	isoDate  isoDate
	calendar string
}

// ZonedDateTime is Temporal.ZonedDateTime. timeZone holds the identifier as given (canonical case),
// calendar the calendar identifier.
type ZonedDateTime struct {
	epochNs  *big.Int
	timeZone string
	calendar string
}

// Duration is Temporal.Duration.
type Duration struct {
	years, months, weeks, days, hours, minutes, seconds, milliseconds, microseconds, nanoseconds float64
}
