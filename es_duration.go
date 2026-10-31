package temporal

import "math"

// dateDurationDays is DateDurationDays.
func dateDurationDays(d dateDuration, plainRelativeTo *PlainDate) float64 {
	yearsMonthsWeeksDuration := adjustDateDurationRecord(d, 0)
	if dateDurationSign(yearsMonthsWeeksDuration) == 0 {
		return d.days
	}

	// balance years, months, and weeks down to days
	iso := plainRelativeTo.isoDate
	later := calendarDateAdd(plainRelativeTo.calendar, iso, yearsMonthsWeeksDuration, "constrain")
	epochDaysEarlier := isoDateToEpochDays(iso.year, iso.month-1, iso.day)
	epochDaysLater := isoDateToEpochDays(later.year, later.month-1, later.day)
	yearsMonthsWeeksInDays := epochDaysLater - epochDaysEarlier
	return d.days + yearsMonthsWeeksInDays
}

// createTemporalDuration is new Temporal.Duration(...) with Number arguments.
func createTemporalDuration(yearsParam, monthsParam, weeksParam, daysParam, hoursParam, minutesParam, secondsParam, millisecondsParam, microsecondsParam, nanosecondsParam float64) *Duration {
	years := toIntegerIfIntegral(yearsParam)
	months := toIntegerIfIntegral(monthsParam)
	weeks := toIntegerIfIntegral(weeksParam)
	days := toIntegerIfIntegral(daysParam)
	hours := toIntegerIfIntegral(hoursParam)
	minutes := toIntegerIfIntegral(minutesParam)
	seconds := toIntegerIfIntegral(secondsParam)
	milliseconds := toIntegerIfIntegral(millisecondsParam)
	microseconds := toIntegerIfIntegral(microsecondsParam)
	nanoseconds := toIntegerIfIntegral(nanosecondsParam)

	rejectDuration(years, months, weeks, days, hours, minutes, seconds, milliseconds, microseconds, nanoseconds)

	return &Duration{years, months, weeks, days, hours, minutes, seconds, milliseconds, microseconds, nanoseconds}
}

func createNegatedTemporalDuration(duration *Duration) *Duration {
	return createTemporalDuration(
		-duration.years,
		-duration.months,
		-duration.weeks,
		-duration.days,
		-duration.hours,
		-duration.minutes,
		-duration.seconds,
		-duration.milliseconds,
		-duration.microseconds,
		-duration.nanoseconds,
	)
}

func rejectDuration(y, mon, w, d, h, min, s, ms, µs, ns float64) {
	sign := 0.0
	fields := [...]float64{y, mon, w, d, h, min, s, ms, µs, ns}
	for _, prop := range fields {
		if math.IsInf(prop, 0) {
			panic("assertion failure: infinite values not allowed as duration fields")
		}
		if prop != 0 {
			propSign := 1.0
			if prop < 0 {
				propSign = -1
			}
			if sign != 0 && propSign != sign {
				throwRange("mixed-sign values not allowed as duration fields")
			}
			sign = propSign
		}
	}
	const twoTo32 = 1 << 32
	if math.Abs(y) >= twoTo32 || math.Abs(mon) >= twoTo32 || math.Abs(w) >= twoTo32 {
		throwRange("years, months, and weeks must be < 2³²")
	}
	msDiv, msMod := truncatingDivModByPowerOf10(ms, 3)
	µsDiv, µsMod := truncatingDivModByPowerOf10(µs, 6)
	nsDiv, nsMod := truncatingDivModByPowerOf10(ns, 9)
	remainderSec, _ := truncatingDivModByPowerOf10(float64(msMod*1e6)+float64(µsMod*1e3)+nsMod, 9)
	totalSec := float64(d*86400) + float64(h*3600) + float64(min*60) + s + msDiv + µsDiv + nsDiv + remainderSec
	if !isFiniteNumber(totalSec) || math.Trunc(totalSec) != totalSec || math.Abs(totalSec) > 9007199254740991 {
		throwRange("total of duration time units cannot exceed 9007199254740991.999999999 s")
	}
}

func toInternalDurationRecord(duration *Duration) internalDuration {
	date := dateDuration{
		years:  duration.years,
		months: duration.months,
		weeks:  duration.weeks,
		days:   duration.days,
	}
	tm := timeDurationFromComponents(
		duration.hours,
		duration.minutes,
		duration.seconds,
		duration.milliseconds,
		duration.microseconds,
		duration.nanoseconds,
	)
	return internalDuration{date, tm}
}

func toInternalDurationRecordWith24HourDays(duration *Duration) internalDuration {
	tm := timeDurationFromComponents(
		duration.hours,
		duration.minutes,
		duration.seconds,
		duration.milliseconds,
		duration.microseconds,
		duration.nanoseconds,
	).add24HourDays(duration.days)
	date := dateDuration{
		years:  duration.years,
		months: duration.months,
		weeks:  duration.weeks,
		days:   0,
	}
	return internalDuration{date, tm}
}

func toDateDurationRecordWithoutTime(duration *Duration) dateDuration {
	internal := toInternalDurationRecordWith24HourDays(duration)
	days := jsTrunc(internal.time.sec / 86400)
	rejectDuration(internal.date.years, internal.date.months, internal.date.weeks, days, 0, 0, 0, 0, 0, 0)
	result := internal.date
	result.days = days
	return result
}

func temporalDurationFromInternal(internal internalDuration, largestUnit string) *Duration {
	sign := internal.time.sign()
	nanoseconds := internal.time.abs().subsec
	microseconds := 0.0
	milliseconds := 0.0
	seconds := internal.time.abs().sec
	minutes := 0.0
	hours := 0.0
	days := 0.0

	switch largestUnit {
	case "year", "month", "week", "day":
		microseconds = jsTrunc(nanoseconds / 1000)
		nanoseconds = jsMod(nanoseconds, 1000)
		milliseconds = jsTrunc(microseconds / 1000)
		microseconds = jsMod(microseconds, 1000)
		seconds += jsTrunc(milliseconds / 1000)
		milliseconds = jsMod(milliseconds, 1000)
		minutes = jsTrunc(seconds / 60)
		seconds = jsMod(seconds, 60)
		hours = jsTrunc(minutes / 60)
		minutes = jsMod(minutes, 60)
		days = jsTrunc(hours / 24)
		hours = jsMod(hours, 24)
	case "hour":
		microseconds = jsTrunc(nanoseconds / 1000)
		nanoseconds = jsMod(nanoseconds, 1000)
		milliseconds = jsTrunc(microseconds / 1000)
		microseconds = jsMod(microseconds, 1000)
		seconds += jsTrunc(milliseconds / 1000)
		milliseconds = jsMod(milliseconds, 1000)
		minutes = jsTrunc(seconds / 60)
		seconds = jsMod(seconds, 60)
		hours = jsTrunc(minutes / 60)
		minutes = jsMod(minutes, 60)
	case "minute":
		microseconds = jsTrunc(nanoseconds / 1000)
		nanoseconds = jsMod(nanoseconds, 1000)
		milliseconds = jsTrunc(microseconds / 1000)
		microseconds = jsMod(microseconds, 1000)
		seconds += jsTrunc(milliseconds / 1000)
		milliseconds = jsMod(milliseconds, 1000)
		minutes = jsTrunc(seconds / 60)
		seconds = jsMod(seconds, 60)
	case "second":
		microseconds = jsTrunc(nanoseconds / 1000)
		nanoseconds = jsMod(nanoseconds, 1000)
		milliseconds = jsTrunc(microseconds / 1000)
		microseconds = jsMod(microseconds, 1000)
		seconds += jsTrunc(milliseconds / 1000)
		milliseconds = jsMod(milliseconds, 1000)
	case "millisecond":
		microseconds = jsTrunc(nanoseconds / 1000)
		nanoseconds = jsMod(nanoseconds, 1000)
		milliseconds = fmaPowerOf10(seconds, 3, jsTrunc(microseconds/1000))
		microseconds = jsMod(microseconds, 1000)
		seconds = 0
	case "microsecond":
		microseconds = fmaPowerOf10(seconds, 6, jsTrunc(nanoseconds/1000))
		nanoseconds = jsMod(nanoseconds, 1000)
		seconds = 0
	case "nanosecond":
		nanoseconds = fmaPowerOf10(seconds, 9, nanoseconds)
		seconds = 0
	default:
		panic("assertion failure: code should not be reached")
	}

	return createTemporalDuration(
		internal.date.years,
		internal.date.months,
		internal.date.weeks,
		internal.date.days+float64(sign*days),
		float64(sign*hours),
		float64(sign*minutes),
		float64(sign*seconds),
		float64(sign*milliseconds),
		float64(sign*microseconds),
		float64(sign*nanoseconds),
	)
}

func combineDateAndTimeDuration(d dateDuration, td *timeDuration) internalDuration {
	dateSign := dateDurationSign(d)
	timeSign := td.sign()
	if !(dateSign == 0 || timeSign == 0 || dateSign == timeSign) {
		panic("assertion failure: should not be able to create mixed sign duration fields here")
	}
	return internalDuration{d, td}
}

// Caution: month is 0-based
func isoDateToEpochDays(year, month, day float64) float64 {
	return getUTCEpochMilliseconds(isoDate{year, month + 1, day}, timeRecord{}) / dayMs
}

// This is needed before calling GetUTCEpochNanoseconds, because it uses MakeDay
// which is ill-defined in how it handles large year numbers. If the issue
// https://github.com/tc39/ecma262/issues/1087 is fixed, this can be removed
// with no observable changes.
func checkISODaysRange(iso isoDate) {
	if math.Abs(isoDateToEpochDays(iso.year, iso.month-1, iso.day)) > 1e8 {
		throwRange("date/time value " + numberToString(iso.year) + "-" + numberToString(iso.month) + "-" + numberToString(iso.day) + " is outside the supported range")
	}
}
