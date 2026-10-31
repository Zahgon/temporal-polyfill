// Package intl reproduces the Intl.DateTimeFormat and Intl.DurationFormat answers of Node.js 24.22.1
// (ICU 78.3, CLDR 48.0, tz 2026e) from tables extracted from that Node by the scripts in gen/.
package intl

// Error is an exception V8 would throw: Name is the constructor name, Message its message.
type Error struct{ Name, Message string }

func (e *Error) Error() string { return e.Name + ": " + e.Message }

// Part is one element of formatToParts output.
type Part struct{ Type, Value string }

// RangePart is one element of formatRangeToParts output.
type RangePart struct{ Type, Value, Source string }

// Unsupported reports a combination the extracted tables do not cover.
type Unsupported struct{ What string }

func (u *Unsupported) Error() string { return "intl: no ICU data for " + u.What }

// ResolvedOptions mirrors Intl.DateTimeFormat.prototype.resolvedOptions(); nil means the property is absent.
type ResolvedOptions struct {
	Locale, Calendar, NumberingSystem, TimeZone                     string
	HourCycle                                                       *string
	Hour12                                                          *bool
	Weekday, Era, Year, Month, Day, DayPeriod, Hour, Minute, Second *string
	FractionalSecondDigits                                          *float64
	TimeZoneName, DateStyle, TimeStyle                              *string
}

// DurationRecord is the argument of Intl.DurationFormat.prototype.format.
type DurationRecord struct {
	Years, Months, Weeks, Days, Hours, Minutes, Seconds, Milliseconds, Microseconds, Nanoseconds float64
}
