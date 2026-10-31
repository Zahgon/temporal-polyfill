package temporal

import (
	"math"
	"math/big"
	"slices"
	"strings"
)

func differenceTime(time1, time2 timeRecord) *timeDuration {
	hours := time2.hour - time1.hour
	minutes := time2.minute - time1.minute
	seconds := time2.second - time1.second
	milliseconds := time2.millisecond - time1.millisecond
	microseconds := time2.microsecond - time1.microsecond
	nanoseconds := time2.nanosecond - time1.nanosecond
	td := timeDurationFromComponents(hours, minutes, seconds, milliseconds, microseconds, nanoseconds)
	if !(td.abs().sec < 86400) {
		panic("assertion failure: _bt_.[[Days]] should be 0")
	}
	return td
}

func differenceInstant(ns1, ns2 *big.Int, increment float64, smallestUnit, roundingMode string) internalDuration {
	td := timeDurationFromEpochNsDiff(ns2, ns1)
	td = roundTimeDuration(td, increment, smallestUnit, roundingMode)
	return combineDateAndTimeDuration(zeroDateDuration(), td)
}

func differenceISODateTime(isoDateTime1, isoDateTime2 isoDateTime, calendar, largestUnit string) internalDuration {
	assertISODateTimeWithinLimits(isoDateTime1)
	assertISODateTimeWithinLimits(isoDateTime2)
	td := differenceTime(isoDateTime1.time, isoDateTime2.time)

	timeSign := td.sign()
	dateSign := compareISODate(isoDateTime1.isoDate, isoDateTime2.isoDate)

	// back-off a day from date2 so that the signs of the date and time diff match
	adjustedDate := isoDateTime2.isoDate
	if dateSign == timeSign {
		adjustedDate = addDaysToISODate(adjustedDate, timeSign)
		td = td.add24HourDays(-timeSign)
	}

	dateLargestUnit := largerOfTwoTemporalUnits("day", largestUnit)
	dateDifference := calendarDateUntil(calendar, isoDateTime1.isoDate, adjustedDate, dateLargestUnit)
	if largestUnit != dateLargestUnit {
		// largestUnit < days, so add the days in to the internal duration
		td = td.add24HourDays(dateDifference.days)
		dateDifference.days = 0
	}
	return combineDateAndTimeDuration(dateDifference, td)
}

func differenceZonedDateTime(ns1, ns2 *big.Int, timeZone, calendar, largestUnit string) internalDuration {
	nsDiff := new(big.Int).Sub(ns2, ns1)
	if nsDiff.Sign() == 0 {
		return internalDuration{zeroDateDuration(), timeDurationZero}
	}
	sign := -1.0
	if nsDiff.Sign() < 0 {
		sign = 1
	}

	// Convert start/end instants to datetimes
	isoDtStart := getISODateTimeFor(timeZone, ns1)
	isoDtEnd := getISODateTimeFor(timeZone, ns2)

	// If year-month-day values are same-day,
	// there's no point involving the calendar/timezone for zdt->pdt conversions.
	// It also avoids a situation where dayCorrection backs up too far on same-day diffs
	// with reverse-direction wallclock delta due to DST:
	// https://github.com/tc39/proposal-temporal/issues/3141
	if isoDtStart.isoDate.year == isoDtEnd.isoDate.year &&
		isoDtStart.isoDate.month == isoDtEnd.isoDate.month &&
		isoDtStart.isoDate.day == isoDtEnd.isoDate.day {
		return internalDuration{zeroDateDuration(), newTimeDuration(nsDiff)}
	}

	// Simulate moving ns1 as many years/months/weeks/days as possible without
	// surpassing ns2. This value is stored in intermediateDateTime/intermediateInstant/intermediateNs.
	// We do not literally move years/months/weeks/days with calendar arithmetic,
	// but rather assume intermediateDateTime will have the same time-parts as
	// dtStart and the date-parts from dtEnd, and move backward from there.
	// The number of days we move backward is stored in dayCorrection.
	// Credit to Adam Shaw for devising this algorithm.
	dayCorrection := 0.0
	var intermediateDateTime isoDateTime
	hasIntermediate := false

	// The max number of allowed day corrections depends on the direction of travel.
	// Both directions allow for 1 day correction due to an ISO wall-clock overshoot (see below).
	// Only the forward direction allows for an additional 1 day correction caused by a push-forward
	// 'compatible' DST transition causing the wall-clock to overshoot again.
	// This max value is inclusive.
	maxDayCorrection := 1.0
	if sign == -1 {
		maxDayCorrection = 2
	}

	// Detect ISO wall-clock overshoot.
	// If the diff of the ISO wall-clock times is opposite to the overall diff's sign,
	// we are guaranteed to need at least one day correction.
	td := differenceTime(isoDtStart.time, isoDtEnd.time)
	if td.sign() == sign {
		dayCorrection++
	}

	for ; dayCorrection <= maxDayCorrection; dayCorrection++ {
		intermediateDate := addDaysToISODate(isoDtEnd.isoDate, float64(dayCorrection*sign))

		// Incorporate time parts from dtStart
		intermediateDateTime = combineISODateAndTimeRecord(intermediateDate, isoDtStart.time)
		hasIntermediate = true

		// Convert intermediate datetime to epoch-nanoseconds (may disambiguate)
		intermediateNs := getEpochNanosecondsFor(timeZone, intermediateDateTime, "compatible")

		// Compute the nanosecond diff between the intermediate instant and the final destination
		td = timeDurationFromEpochNsDiff(ns2, intermediateNs)

		// Did intermediateNs NOT surpass ns2?
		// If so, exit the loop with success (without incrementing dayCorrection past maxDayCorrection)
		if td.sign() != sign {
			break
		}
	}

	if !(dayCorrection <= maxDayCorrection) {
		panic("assertion failure: more than " + numberToString(maxDayCorrection) + " day correction needed")
	}

	// Output of the above loop
	if !hasIntermediate {
		panic("Expected arg to be set.")
	}

	// Similar to what happens in DifferenceISODateTime with date parts only:
	dateLargestUnit := largerOfTwoTemporalUnits("day", largestUnit)
	dateDifference := calendarDateUntil(calendar, isoDtStart.isoDate, intermediateDateTime.isoDate, dateLargestUnit)
	return combineDateAndTimeDuration(dateDifference, td)
}

type nudgeWindow struct {
	r1, r2                     float64
	startEpochNs, endEpochNs   *big.Int
	startDuration, endDuration dateDuration
}

// nudgeResult is what the Nudge operations return.
type nudgeResult struct {
	duration              internalDuration
	nudgedEpochNs         *big.Int
	didExpandCalendarUnit bool
}

func computeNudgeWindow(
	sign float64,
	duration internalDuration,
	originEpochNs *big.Int,
	dt isoDateTime,
	timeZone string,
	calendar string,
	increment float64,
	unit string,
	additionalShift bool,
) nudgeWindow {
	// Create a duration with smallestUnit trunc'd towards zero
	// Create a separate duration that incorporates roundingIncrement
	var r1, r2 float64
	var startDuration, endDuration dateDuration
	switch unit {
	case "year":
		years := roundNumberToIncrement(duration.date.years, increment, "trunc")
		if !additionalShift {
			r1 = years
		} else {
			r1 = years + float64(increment*sign)
		}
		r2 = r1 + float64(increment*sign)
		startDuration = dateDuration{years: r1}
		endDuration = dateDuration{years: r2}
	case "month":
		months := roundNumberToIncrement(duration.date.months, increment, "trunc")
		if !additionalShift {
			r1 = months
		} else {
			r1 = months + float64(increment*sign)
		}
		r2 = r1 + float64(increment*sign)
		startDuration = adjustDateDurationRecord(duration.date, 0, 0, r1)
		endDuration = adjustDateDurationRecord(duration.date, 0, 0, r2)
	case "week":
		yearsMonths := adjustDateDurationRecord(duration.date, 0, 0)
		weeksStart := calendarDateAdd(calendar, dt.isoDate, yearsMonths, "constrain")
		weeksEnd := addDaysToISODate(weeksStart, duration.date.days)
		untilResult := calendarDateUntil(calendar, weeksStart, weeksEnd, "week")
		weeks := roundNumberToIncrement(duration.date.weeks+untilResult.weeks, increment, "trunc")
		r1 = weeks
		r2 = weeks + float64(increment*sign)
		startDuration = adjustDateDurationRecord(duration.date, 0, r1)
		endDuration = adjustDateDurationRecord(duration.date, 0, r2)
	case "day":
		days := roundNumberToIncrement(duration.date.days, increment, "trunc")
		r1 = days
		r2 = days + float64(increment*sign)
		startDuration = adjustDateDurationRecord(duration.date, r1)
		endDuration = adjustDateDurationRecord(duration.date, r2)
	default:
		panic("assertion failure: code should not be reached")
	}

	if sign == 1 && !(r1 >= 0 && r1 < r2) {
		panic("assertion failure: positive ordering of r1, r2: 0 ≤ " + numberToString(r1) + " < " + numberToString(r2))
	}
	if sign == -1 && !(r1 <= 0 && r1 > r2) {
		panic("assertion failure: negative ordering of r1, r2: 0 ≥ " + numberToString(r1) + " > " + numberToString(r2))
	}

	// Convert to bound-START to epoch-nanoseconds
	var startEpochNs *big.Int
	if r1 == 0 || math.IsNaN(r1) {
		// If the start of the bound is the same as the "origin" (aka relativeTo),
		// use the origin's epoch-nanoseconds as-is instead of relying on isoDateTime,
		// which then gets zoned and converted back to epoch-nanoseconds,
		// which loses precision and creates a distorted bounding window.
		startEpochNs = originEpochNs
	} else {
		start := calendarDateAdd(calendar, dt.isoDate, startDuration, "constrain")
		startDateTime := combineISODateAndTimeRecord(start, dt.time)
		if timeZone != "" {
			startEpochNs = getEpochNanosecondsFor(timeZone, startDateTime, "compatible")
		} else {
			startEpochNs = getUTCEpochNanoseconds(startDateTime)
		}
	}

	// Convert to bound-END to epoch-nanoseconds
	end := calendarDateAdd(calendar, dt.isoDate, endDuration, "constrain")
	endDateTime := combineISODateAndTimeRecord(end, dt.time)
	var endEpochNs *big.Int
	if timeZone != "" {
		endEpochNs = getEpochNanosecondsFor(timeZone, endDateTime, "compatible")
	} else {
		endEpochNs = getUTCEpochNanoseconds(endDateTime)
	}

	return nudgeWindow{r1, r2, startEpochNs, endEpochNs, startDuration, endDuration}
}

// Epoch-nanosecond bounding technique where the start/end of the calendar-unit
// interval are converted to epoch-nanosecond times and destEpochNs is nudged to
// either one.
func nudgeToCalendarUnit(
	sign float64,
	durationParam internalDuration,
	originEpochNs *big.Int,
	destEpochNs *big.Int,
	dt isoDateTime,
	timeZone string,
	calendar string,
	increment float64,
	unit string,
	roundingMode string,
) (result nudgeResult, total float64) {
	duration := durationParam

	didExpandCalendarUnit := false
	window := computeNudgeWindow(sign, duration, originEpochNs, dt, timeZone, calendar, increment, unit, false)

	// Round the smallestUnit within the epoch-nanosecond span
	if sign == 1 {
		if !(window.startEpochNs.Cmp(destEpochNs) <= 0 && destEpochNs.Cmp(window.endEpochNs) <= 0) {
			// Retry nudge window if it's out of bounds
			window = computeNudgeWindow(sign, duration, originEpochNs, dt, timeZone, calendar, increment, unit, true)
			if !(window.startEpochNs.Cmp(destEpochNs) <= 0 && destEpochNs.Cmp(window.endEpochNs) <= 0) {
				panic("assertion failure: " + unit + " was 0 days long")
			}
			didExpandCalendarUnit = true
		}
	} else if sign == -1 {
		if !(window.endEpochNs.Cmp(destEpochNs) <= 0 && destEpochNs.Cmp(window.startEpochNs) <= 0) {
			// Retry nudge window if it's out of bounds
			window = computeNudgeWindow(sign, duration, originEpochNs, dt, timeZone, calendar, increment, unit, true)
			if !(window.endEpochNs.Cmp(destEpochNs) <= 0 && destEpochNs.Cmp(window.startEpochNs) <= 0) {
				panic("assertion failure: " + unit + " was 0 days long")
			}
			didExpandCalendarUnit = true
		}
	}
	r1, r2 := window.r1, window.r2
	startEpochNs, endEpochNs := window.startEpochNs, window.endEpochNs
	startDuration, endDuration := window.startDuration, window.endDuration

	if endEpochNs.Cmp(startEpochNs) == 0 {
		panic("assertion failure: startEpochNs must ≠ endEpochNs")
	}
	numerator := timeDurationFromEpochNsDiff(destEpochNs, startEpochNs)
	denominator := timeDurationFromEpochNsDiff(endEpochNs, startEpochNs)
	unsignedSign := "positive"
	if sign < 0 {
		unsignedSign = "negative"
	}
	unsignedRoundingMode := getUnsignedRoundingMode(roundingMode, unsignedSign)
	cmp := numerator.add(numerator).abs().subtract(denominator.abs()).sign()
	even := jsMod(math.Abs(r1)/increment, 2) == 0
	var roundedUnit float64
	if numerator.isZero() {
		roundedUnit = math.Abs(r1)
	} else if numerator.cmp(denominator) == 0 { // equal?
		roundedUnit = math.Abs(r2)
	} else {
		roundedUnit = applyUnsignedRoundingMode(math.Abs(r1), math.Abs(r2), cmp, even, unsignedRoundingMode)
	}

	// Trick to minimize rounding error, due to the lack of fma() in JS
	fakeNumerator := newTimeDuration(new(big.Int).Add(
		new(big.Int).Mul(denominator.totalNs, bigFromNumber(r1)),
		new(big.Int).Mul(numerator.totalNs, bigFromNumber(float64(increment*sign))),
	))
	total = fakeNumerator.fdiv(denominator.totalNs)
	if !(math.Abs(r1) <= math.Abs(total) && math.Abs(total) <= math.Abs(r2)) {
		panic("assertion failure: r1 ≤ total ≤ r2")
	}

	// Determine whether expanded or contracted
	didExpandCalendarUnit = didExpandCalendarUnit || roundedUnit == math.Abs(r2)
	if roundedUnit == math.Abs(r2) {
		duration = internalDuration{endDuration, timeDurationZero}
	} else {
		duration = internalDuration{startDuration, timeDurationZero}
	}

	nudgedEpochNs := startEpochNs
	if didExpandCalendarUnit {
		nudgedEpochNs = endEpochNs
	}
	return nudgeResult{duration, nudgedEpochNs, didExpandCalendarUnit}, total
}

// Attempts rounding of time units within a time zone's day, but if the rounding
// causes time to exceed the total time within the day, rerun rounding in next
// day.
func nudgeToZonedTime(
	sign float64,
	durationParam internalDuration,
	dt isoDateTime,
	timeZone string,
	calendar string,
	increment float64,
	unit string,
	roundingMode string,
) nudgeResult {
	// unit must be hour or smaller
	duration := durationParam

	// Apply to origin, output start/end of the day as PlainDateTimes
	start := calendarDateAdd(calendar, dt.isoDate, duration.date, "constrain")
	startDateTime := combineISODateAndTimeRecord(start, dt.time)
	endDate := addDaysToISODate(start, sign)
	endDateTime := combineISODateAndTimeRecord(endDate, dt.time)

	// Compute the epoch-nanosecond start/end of the final whole-day interval
	// If duration has negative sign, startEpochNs will be after endEpochNs
	startEpochNs := getEpochNanosecondsFor(timeZone, startDateTime, "compatible")
	endEpochNs := getEpochNanosecondsFor(timeZone, endDateTime, "compatible")

	// The signed amount of time from the start of the whole-day interval to the end
	daySpan := timeDurationFromEpochNsDiff(endEpochNs, startEpochNs)
	if daySpan.sign() != sign {
		throwRange("time zone returned inconsistent Instants")
	}

	// Compute time parts of the duration to nanoseconds and round
	// Result could be negative
	unitIncrement := bigFromNumber(float64(nsPerTimeUnit[unit] * increment))
	roundedTimeDuration := duration.time.round(unitIncrement, roundingMode)

	// Does the rounded time exceed the time-in-day?
	beyondDaySpan := roundedTimeDuration.subtract(daySpan)
	didRoundBeyondDay := beyondDaySpan.sign() != -sign

	var dayDelta float64
	var nudgedEpochNs *big.Int
	if didRoundBeyondDay {
		// If rounded into next day, use the day-end as the local origin and rerun
		// the rounding
		dayDelta = sign
		roundedTimeDuration = beyondDaySpan.round(unitIncrement, roundingMode)
		nudgedEpochNs = roundedTimeDuration.addToEpochNs(endEpochNs)
	} else {
		// Otherwise, if time not rounded beyond day, use the day-start as the local
		// origin
		dayDelta = 0
		nudgedEpochNs = roundedTimeDuration.addToEpochNs(startEpochNs)
	}

	d := adjustDateDurationRecord(duration.date, duration.date.days+dayDelta)
	resultDuration := combineDateAndTimeDuration(d, roundedTimeDuration)
	return nudgeResult{resultDuration, nudgedEpochNs, didRoundBeyondDay}
}

// Converts all fields to nanoseconds and does integer rounding.
func nudgeToDayOrTime(
	durationParam internalDuration,
	destEpochNs *big.Int,
	largestUnit string,
	increment float64,
	smallestUnit string,
	roundingMode string,
) nudgeResult {
	// unit must be day or smaller
	duration := durationParam

	td := duration.time.add24HourDays(duration.date.days)
	// Convert to nanoseconds and round
	roundedTime := td.round(bigFromNumber(float64(increment*nsPerTimeUnit[smallestUnit])), roundingMode)
	diffTime := roundedTime.subtract(td)

	// Determine if whole days expanded
	wholeDays, _ := td.divmod(dayNanos)
	roundedWholeDays, _ := roundedTime.divmod(dayNanos)
	didExpandDays := jsSign(roundedWholeDays-wholeDays) == td.sign()

	nudgedEpochNs := diffTime.addToEpochNs(destEpochNs)

	days := 0.0
	remainder := roundedTime
	if temporalUnitCategory(largestUnit) == "date" {
		days = roundedWholeDays
		remainder = roundedTime.add(timeDurationFromComponents(float64(-roundedWholeDays*24), 0, 0, 0, 0, 0))
	}

	d := adjustDateDurationRecord(duration.date, days)
	return nudgeResult{internalDuration{d, remainder}, nudgedEpochNs, didExpandDays}
}

// Given a potentially bottom-heavy duration, bubble up smaller units to larger
// units. Any units smaller than smallestUnit are already zeroed-out.
func bubbleRelativeDuration(
	sign float64,
	durationParam internalDuration,
	nudgedEpochNs *big.Int,
	dt isoDateTime,
	timeZone string,
	calendar string,
	largestUnit string,
	smallestUnit string,
) internalDuration {
	// smallestUnit is day or larger
	duration := durationParam

	if smallestUnit == largestUnit {
		return duration
	}

	// Check to see if nudgedEpochNs has hit the boundary of any units higher than
	// smallestUnit, in which case increment the higher unit and clear smaller
	// units.
	largestUnitIndex := slices.Index(unitsDescending, largestUnit)
	smallestUnitIndex := slices.Index(unitsDescending, smallestUnit)
	for unitIndex := smallestUnitIndex - 1; unitIndex >= largestUnitIndex; unitIndex-- {
		// The only situation where days and smaller bubble-up into weeks is when
		// largestUnit is 'week' (not to be confused with the situation where
		// smallestUnit is 'week', in which case days and smaller are ROUNDED-up
		// into weeks, but that has already happened by the time this function
		// executes)
		// So, if days and smaller are NOT bubbled-up into weeks, and the current
		// unit is weeks, skip.
		unit := unitsDescending[unitIndex]
		if unit == "week" && largestUnit != "week" {
			continue
		}

		var endDuration dateDuration
		switch unit {
		case "year":
			years := duration.date.years + sign
			endDuration = dateDuration{years: years}
		case "month":
			months := duration.date.months + sign
			endDuration = adjustDateDurationRecord(duration.date, 0, 0, months)
		case "week":
			weeks := duration.date.weeks + sign
			endDuration = adjustDateDurationRecord(duration.date, 0, weeks)
		default:
			panic("assertion failure: code should not be reached")
		}

		// Compute end-of-unit in epoch-nanoseconds
		end := calendarDateAdd(calendar, dt.isoDate, endDuration, "constrain")
		endDateTime := combineISODateAndTimeRecord(end, dt.time)
		var endEpochNs *big.Int
		if timeZone != "" {
			endEpochNs = getEpochNanosecondsFor(timeZone, endDateTime, "compatible")
		} else {
			endEpochNs = getUTCEpochNanoseconds(endDateTime)
		}

		didExpandToEnd := compare(nudgedEpochNs, endEpochNs) != -sign

		// Is nudgedEpochNs at the end-of-unit? This means it should bubble-up to
		// the next highest unit (and possibly further...)
		if didExpandToEnd {
			duration = internalDuration{endDuration, timeDurationZero}
		} else {
			// NOT at end-of-unit. Stop looking for bubbling
			break
		}
	}

	return duration
}

// roundRelativeDuration takes "" for a null timeZone; both are falsy in the original.
func roundRelativeDuration(
	durationParam internalDuration,
	originEpochNs *big.Int,
	destEpochNs *big.Int,
	dt isoDateTime,
	timeZone string,
	calendar string,
	largestUnitParam string,
	increment float64,
	smallestUnit string,
	roundingMode string,
) internalDuration {
	duration := durationParam
	// The duration must already be balanced. This should be achieved by calling
	// one of the non-rounding since/until internal methods prior. It's okay to
	// have a bottom-heavy weeks because weeks don't bubble-up into months. It's
	// okay to have >24 hour day assuming the final day of relativeTo+duration has
	// >24 hours in its timezone. (should automatically end up like this if using
	// non-rounding since/until internal methods prior)
	irregularLengthUnit := isCalendarUnit(smallestUnit) || (timeZone != "" && smallestUnit == "day")
	sign := 1.0
	if internalDurationSign(duration) < 0 {
		sign = -1
	}

	var result nudgeResult
	if irregularLengthUnit {
		// Rounding an irregular-length unit? Use epoch-nanosecond-bounding technique
		result, _ = nudgeToCalendarUnit(sign, duration, originEpochNs, destEpochNs, dt, timeZone, calendar, increment, smallestUnit, roundingMode)
	} else if timeZone != "" {
		// Special-case for rounding time units within a zoned day
		result = nudgeToZonedTime(sign, duration, dt, timeZone, calendar, increment, smallestUnit, roundingMode)
	} else {
		// Rounding uniform-length days/hours/minutes/etc units. Simple nanosecond
		// math. years/months/weeks unchanged
		result = nudgeToDayOrTime(duration, destEpochNs, largestUnitParam, increment, smallestUnit, roundingMode)
	}

	duration = result.duration
	// Did nudging cause the duration to expand to the next day or larger?
	// Bubble-up smaller calendar units into higher ones, except for weeks, which
	// don't balance up into months
	if result.didExpandCalendarUnit && smallestUnit != "week" {
		duration = bubbleRelativeDuration(
			sign,
			duration,
			result.nudgedEpochNs, // The destEpochNs after expanding/contracting
			dt,
			timeZone,
			calendar,
			largestUnitParam, // where to STOP bubbling
			largerOfTwoTemporalUnits(smallestUnit, "day"), // where to START bubbling-up from
		)
	}

	return duration
}

// totalRelativeDuration takes "" for a null timeZone, like roundRelativeDuration.
func totalRelativeDuration(
	duration internalDuration,
	originEpochNs *big.Int,
	destEpochNs *big.Int,
	dt isoDateTime,
	timeZone string,
	calendar string,
	unit string,
) float64 {
	// The duration must already be balanced. This should be achieved by calling
	// one of the non-rounding since/until internal methods prior. It's okay to
	// have a bottom-heavy weeks because weeks don't bubble-up into months. It's
	// okay to have >24 hour day assuming the final day of relativeTo+duration has
	// >24 hours in its timezone. (should automatically end up like this if using
	// non-rounding since/until internal methods prior)
	if isCalendarUnit(unit) || (timeZone != "" && unit == "day") {
		// Rounding an irregular-length unit? Use epoch-nanosecond-bounding technique
		sign := 1.0
		if internalDurationSign(duration) < 0 {
			sign = -1
		}
		_, total := nudgeToCalendarUnit(sign, duration, originEpochNs, destEpochNs, dt, timeZone, calendar, 1, unit, "trunc")
		return total
	}
	// Rounding uniform-length days/hours/minutes/etc units. Simple nanosecond
	// math. years/months/weeks unchanged
	td := duration.time.add24HourDays(duration.date.days)
	return totalTimeDuration(td, unit)
}

func differencePlainDateTimeWithRounding(
	isoDateTime1, isoDateTime2 isoDateTime,
	calendar string,
	largestUnit string,
	roundingIncrement float64,
	smallestUnit string,
	roundingMode string,
) internalDuration {
	if compareISODateTime(isoDateTime1, isoDateTime2) == 0 {
		return internalDuration{zeroDateDuration(), timeDurationZero}
	}

	rejectDateTimeRange(isoDateTime1)
	rejectDateTimeRange(isoDateTime2)
	duration := differenceISODateTime(isoDateTime1, isoDateTime2, calendar, largestUnit)

	if smallestUnit == "nanosecond" && roundingIncrement == 1 {
		return duration
	}

	originEpochNs := getUTCEpochNanoseconds(isoDateTime1)
	destEpochNs := getUTCEpochNanoseconds(isoDateTime2)
	return roundRelativeDuration(duration, originEpochNs, destEpochNs, isoDateTime1, "", calendar, largestUnit, roundingIncrement, smallestUnit, roundingMode)
}

func differencePlainDateTimeWithTotal(isoDateTime1, isoDateTime2 isoDateTime, calendar, unit string) float64 {
	if compareISODateTime(isoDateTime1, isoDateTime2) == 0 {
		return 0
	}

	rejectDateTimeRange(isoDateTime1)
	rejectDateTimeRange(isoDateTime2)
	duration := differenceISODateTime(isoDateTime1, isoDateTime2, calendar, unit)

	if unit == "nanosecond" {
		return bigToNumber(duration.time.totalNs)
	}

	originEpochNs := getUTCEpochNanoseconds(isoDateTime1)
	destEpochNs := getUTCEpochNanoseconds(isoDateTime2)
	return totalRelativeDuration(duration, originEpochNs, destEpochNs, isoDateTime1, "", calendar, unit)
}

func differenceZonedDateTimeWithRounding(
	ns1, ns2 *big.Int,
	timeZone string,
	calendar string,
	largestUnit string,
	roundingIncrement float64,
	smallestUnit string,
	roundingMode string,
) internalDuration {
	if temporalUnitCategory(largestUnit) == "time" {
		// The user is only asking for a time difference, so return difference of instants.
		return differenceInstant(ns1, ns2, roundingIncrement, smallestUnit, roundingMode)
	}

	duration := differenceZonedDateTime(ns1, ns2, timeZone, calendar, largestUnit)

	if smallestUnit == "nanosecond" && roundingIncrement == 1 {
		return duration
	}

	dateTime := getISODateTimeFor(timeZone, ns1)
	return roundRelativeDuration(duration, ns1, ns2, dateTime, timeZone, calendar, largestUnit, roundingIncrement, smallestUnit, roundingMode)
}

func differenceZonedDateTimeWithTotal(ns1, ns2 *big.Int, timeZone, calendar, unit string) float64 {
	if temporalUnitCategory(unit) == "time" {
		// The user is only asking for a time difference, so return difference of instants.
		return totalTimeDuration(timeDurationFromEpochNsDiff(ns2, ns1), unit)
	}

	duration := differenceZonedDateTime(ns1, ns2, timeZone, calendar, unit)
	dateTime := getISODateTimeFor(timeZone, ns1)
	return totalRelativeDuration(duration, ns1, ns2, dateTime, timeZone, calendar, unit)
}

// differenceSettings is what GetDifferenceSettings returns.
type differenceSettings struct {
	largestUnit       string
	roundingIncrement float64
	roundingMode      string
	smallestUnit      string
}

var maxDifferenceIncrements = map[string]float64{
	"hour":        24,
	"minute":      60,
	"second":      60,
	"millisecond": 1000,
	"microsecond": 1000,
	"nanosecond":  1000,
}

func getDifferenceSettings(op string, options any, group string, disallowed []string, fallbackSmallest, smallestLargestDefaultUnit string) differenceSettings {
	allowedUnits := []string{}
	for _, unitInfo := range temporalUnits {
		p := unitInfo[0]
		s := unitInfo[1]
		c := unitInfo[2]
		if (group == "datetime" || c == group) && !slices.Contains(disallowed, s) {
			allowedUnits = append(allowedUnits, s, p)
		}
	}

	largestUnitOption := getTemporalUnitValuedOption(options, "largestUnit")
	roundingIncrement := getRoundingIncrementOption(options)
	roundingMode := getRoundingModeOption(options, "trunc")
	smallestUnitOption := getTemporalUnitValuedOption(options, "smallestUnit")

	validateTemporalUnitValue(largestUnitOption, group, []string{"auto"})
	largestUnit := "auto"
	if largestUnitOption != nil {
		largestUnit = *largestUnitOption
	}
	if slices.Contains(disallowed, largestUnit) {
		throwRange("largestUnit must be one of " + strings.Join(allowedUnits, ", ") + ", not " + largestUnit)
	}

	if op == "since" {
		roundingMode = negateRoundingMode(roundingMode)
	}

	validateTemporalUnitValue(smallestUnitOption, group)
	smallestUnit := fallbackSmallest
	if smallestUnitOption != nil {
		smallestUnit = *smallestUnitOption
	}
	if slices.Contains(disallowed, smallestUnit) {
		throwRange("smallestUnit must be one of " + strings.Join(allowedUnits, ", ") + ", not " + smallestUnit)
	}

	defaultLargestUnit := largerOfTwoTemporalUnits(smallestLargestDefaultUnit, smallestUnit)
	if largestUnit == "auto" {
		largestUnit = defaultLargestUnit
	}
	if largerOfTwoTemporalUnits(largestUnit, smallestUnit) != largestUnit {
		throwRange("largestUnit " + largestUnit + " cannot be smaller than smallestUnit " + smallestUnit)
	}
	if maximum, ok := maxDifferenceIncrements[smallestUnit]; ok {
		validateTemporalRoundingIncrement(roundingIncrement, maximum, false)
	}

	return differenceSettings{largestUnit, roundingIncrement, roundingMode, smallestUnit}
}

func differenceTemporalInstant(operation string, instant *Instant, otherParam, options any) *Duration {
	other := toTemporalInstant(otherParam)

	resolvedOptions := getOptionsObject(options)
	settings := getDifferenceSettings(operation, resolvedOptions, "time", nil, "nanosecond", "second")

	onens := instant.epochNs
	twons := other.epochNs
	duration := differenceInstant(onens, twons, settings.roundingIncrement, settings.smallestUnit, settings.roundingMode)
	result := temporalDurationFromInternal(duration, settings.largestUnit)
	if operation == "since" {
		result = createNegatedTemporalDuration(result)
	}
	return result
}

func differenceTemporalPlainDate(operation string, plainDate *PlainDate, otherParam, options any) *Duration {
	other := toTemporalDate(otherParam)
	calendar := plainDate.calendar
	otherCalendar := other.calendar
	if !calendarEquals(calendar, otherCalendar) {
		throwRange("cannot compute difference between dates of " + calendar + " and " + otherCalendar + " calendars")
	}

	resolvedOptions := getOptionsObject(options)
	settings := getDifferenceSettings(operation, resolvedOptions, "date", nil, "day", "day")

	iso := plainDate.isoDate
	isoOther := other.isoDate
	if compareISODate(iso, isoOther) == 0 {
		return &Duration{}
	}

	dateDifference := calendarDateUntil(calendar, iso, isoOther, settings.largestUnit)

	duration := internalDuration{dateDifference, timeDurationZero}
	roundingIsNoop := settings.smallestUnit == "day" && settings.roundingIncrement == 1
	if !roundingIsNoop {
		dt := combineISODateAndTimeRecord(iso, midnightTimeRecord())
		dtOther := combineISODateAndTimeRecord(isoOther, midnightTimeRecord())
		originEpochNs := getUTCEpochNanoseconds(dt)
		destEpochNs := getUTCEpochNanoseconds(dtOther)
		duration = roundRelativeDuration(duration, originEpochNs, destEpochNs, dt, "", calendar, settings.largestUnit, settings.roundingIncrement, settings.smallestUnit, settings.roundingMode)
	}

	result := temporalDurationFromInternal(duration, "day")
	if operation == "since" {
		result = createNegatedTemporalDuration(result)
	}
	return result
}

func differenceTemporalPlainDateTime(operation string, plainDateTime *PlainDateTime, otherParam, options any) *Duration {
	other := toTemporalDateTime(otherParam)
	calendar := plainDateTime.calendar
	otherCalendar := other.calendar
	if !calendarEquals(calendar, otherCalendar) {
		throwRange("cannot compute difference between dates of " + calendar + " and " + otherCalendar + " calendars")
	}

	resolvedOptions := getOptionsObject(options)
	settings := getDifferenceSettings(operation, resolvedOptions, "datetime", nil, "nanosecond", "day")

	isoDateTime1 := plainDateTime.isoDateTime
	isoDateTime2 := other.isoDateTime
	if compareISODateTime(isoDateTime1, isoDateTime2) == 0 {
		return &Duration{}
	}

	duration := differencePlainDateTimeWithRounding(isoDateTime1, isoDateTime2, calendar, settings.largestUnit, settings.roundingIncrement, settings.smallestUnit, settings.roundingMode)

	result := temporalDurationFromInternal(duration, settings.largestUnit)
	if operation == "since" {
		result = createNegatedTemporalDuration(result)
	}
	return result
}

func differenceTemporalPlainTime(operation string, plainTime *PlainTime, otherParam, options any) *Duration {
	other := toTemporalTime(otherParam)

	resolvedOptions := getOptionsObject(options)
	settings := getDifferenceSettings(operation, resolvedOptions, "time", nil, "nanosecond", "hour")

	td := differenceTime(plainTime.time, other.time)
	td = roundTimeDuration(td, settings.roundingIncrement, settings.smallestUnit, settings.roundingMode)
	duration := combineDateAndTimeDuration(zeroDateDuration(), td)

	result := temporalDurationFromInternal(duration, settings.largestUnit)
	if operation == "since" {
		result = createNegatedTemporalDuration(result)
	}
	return result
}

func differenceTemporalPlainYearMonth(operation string, yearMonth *PlainYearMonth, otherParam, options any) *Duration {
	other := toTemporalYearMonth(otherParam)
	calendar := yearMonth.calendar
	otherCalendar := other.calendar
	if !calendarEquals(calendar, otherCalendar) {
		throwRange("cannot compute difference between months of " + calendar + " and " + otherCalendar + " calendars")
	}

	resolvedOptions := getOptionsObject(options)
	settings := getDifferenceSettings(operation, resolvedOptions, "date", []string{"week", "day"}, "month", "year")

	if compareISODate(yearMonth.isoDate, other.isoDate) == 0 {
		return &Duration{}
	}

	one := 1.0
	thisFields := isoDateToFields(calendar, yearMonth.isoDate, "year-month")
	thisFields.day = &one
	thisDate := calendarDateFromFields(calendar, thisFields, "constrain")
	otherFields := isoDateToFields(calendar, other.isoDate, "year-month")
	otherFields.day = &one
	otherDate := calendarDateFromFields(calendar, otherFields, "constrain")

	dateDifference := calendarDateUntil(calendar, thisDate, otherDate, settings.largestUnit)
	duration := internalDuration{adjustDateDurationRecord(dateDifference, 0, 0), timeDurationZero}
	if settings.smallestUnit != "month" || settings.roundingIncrement != 1 {
		dt := combineISODateAndTimeRecord(thisDate, midnightTimeRecord())
		dtOther := combineISODateAndTimeRecord(otherDate, midnightTimeRecord())
		originEpochNs := getUTCEpochNanoseconds(dt)
		destEpochNs := getUTCEpochNanoseconds(dtOther)
		duration = roundRelativeDuration(duration, originEpochNs, destEpochNs, dt, "", calendar, settings.largestUnit, settings.roundingIncrement, settings.smallestUnit, settings.roundingMode)
	}

	result := temporalDurationFromInternal(duration, "day")
	if operation == "since" {
		result = createNegatedTemporalDuration(result)
	}
	return result
}

func differenceTemporalZonedDateTime(operation string, zonedDateTime *ZonedDateTime, otherParam, options any) *Duration {
	other := toTemporalZonedDateTime(otherParam)
	calendar := zonedDateTime.calendar
	otherCalendar := other.calendar
	if !calendarEquals(calendar, otherCalendar) {
		throwRange("cannot compute difference between dates of " + calendar + " and " + otherCalendar + " calendars")
	}

	resolvedOptions := getOptionsObject(options)
	settings := getDifferenceSettings(operation, resolvedOptions, "datetime", nil, "nanosecond", "hour")

	ns1 := zonedDateTime.epochNs
	ns2 := other.epochNs

	var result *Duration
	if temporalUnitCategory(settings.largestUnit) != "date" {
		// The user is only asking for a time difference, so return difference of instants.
		duration := differenceInstant(ns1, ns2, settings.roundingIncrement, settings.smallestUnit, settings.roundingMode)
		result = temporalDurationFromInternal(duration, settings.largestUnit)
	} else {
		timeZone := zonedDateTime.timeZone
		if !timeZoneEquals(timeZone, other.timeZone) {
			throwRange("When calculating difference between time zones, largestUnit must be 'hours' " +
				"or smaller because day lengths can vary between time zones due to DST or time zone offset changes.")
		}

		if ns1.Cmp(ns2) == 0 {
			return &Duration{}
		}

		duration := differenceZonedDateTimeWithRounding(ns1, ns2, timeZone, calendar, settings.largestUnit, settings.roundingIncrement, settings.smallestUnit, settings.roundingMode)
		result = temporalDurationFromInternal(duration, "hour")
	}

	if operation == "since" {
		result = createNegatedTemporalDuration(result)
	}
	return result
}
