package temporal

// NewPlainTime is new Temporal.PlainTime(...). Every argument defaults to 0.
func NewPlainTime(isoHour, isoMinute, isoSecond, isoMillisecond, isoMicrosecond, isoNanosecond any) *PlainTime {
	hour := toIntegerWithTruncation(defaultTo(isoHour, 0.0))
	minute := toIntegerWithTruncation(defaultTo(isoMinute, 0.0))
	second := toIntegerWithTruncation(defaultTo(isoSecond, 0.0))
	millisecond := toIntegerWithTruncation(defaultTo(isoMillisecond, 0.0))
	microsecond := toIntegerWithTruncation(defaultTo(isoMicrosecond, 0.0))
	nanosecond := toIntegerWithTruncation(defaultTo(isoNanosecond, 0.0))

	rejectTime(hour, minute, second, millisecond, microsecond, nanosecond)
	time := timeRecord{hour: hour, minute: minute, second: second, millisecond: millisecond, microsecond: microsecond, nanosecond: nanosecond}

	return createTemporalTime(time)
}

// Hour is the hour getter.
func (t *PlainTime) Hour() float64 { return t.time.hour }

// Minute is the minute getter.
func (t *PlainTime) Minute() float64 { return t.time.minute }

// Second is the second getter.
func (t *PlainTime) Second() float64 { return t.time.second }

// Millisecond is the millisecond getter.
func (t *PlainTime) Millisecond() float64 { return t.time.millisecond }

// Microsecond is the microsecond getter.
func (t *PlainTime) Microsecond() float64 { return t.time.microsecond }

// Nanosecond is the nanosecond getter.
func (t *PlainTime) Nanosecond() float64 { return t.time.nanosecond }

// With is Temporal.PlainTime.prototype.with.
func (t *PlainTime) With(temporalTimeLike, options any) *PlainTime {
	if !isObject(temporalTimeLike) {
		throwType("invalid argument")
	}
	rejectTemporalLikeObject(temporalTimeLike)

	partialTime := toTemporalTimeRecord(temporalTimeLike, "partial")

	fields := toTemporalTimeRecord(t)
	// Object.assign(fields, partialTime)
	hour, minute, second, millisecond, microsecond, nanosecond := *fields.hour, *fields.minute, *fields.second, *fields.millisecond, *fields.microsecond, *fields.nanosecond
	if partialTime.hour != nil {
		hour = *partialTime.hour
	}
	if partialTime.minute != nil {
		minute = *partialTime.minute
	}
	if partialTime.second != nil {
		second = *partialTime.second
	}
	if partialTime.millisecond != nil {
		millisecond = *partialTime.millisecond
	}
	if partialTime.microsecond != nil {
		microsecond = *partialTime.microsecond
	}
	if partialTime.nanosecond != nil {
		nanosecond = *partialTime.nanosecond
	}
	overflow := getTemporalOverflowOption(getOptionsObject(options))
	regulated := regulateTime(hour, minute, second, millisecond, microsecond, nanosecond, overflow)
	return NewPlainTime(regulated.hour, regulated.minute, regulated.second, regulated.millisecond, regulated.microsecond, regulated.nanosecond)
}

// Add is Temporal.PlainTime.prototype.add.
func (t *PlainTime) Add(temporalDurationLike any) *PlainTime {
	return addDurationToTime("add", t, temporalDurationLike)
}

// Subtract is Temporal.PlainTime.prototype.subtract.
func (t *PlainTime) Subtract(temporalDurationLike any) *PlainTime {
	return addDurationToTime("subtract", t, temporalDurationLike)
}

// Until is Temporal.PlainTime.prototype.until.
func (t *PlainTime) Until(other, options any) *Duration {
	return differenceTemporalPlainTime("until", t, other, options)
}

// Since is Temporal.PlainTime.prototype.since.
func (t *PlainTime) Since(other, options any) *Duration {
	return differenceTemporalPlainTime("since", t, other, options)
}

// Round is Temporal.PlainTime.prototype.round.
func (t *PlainTime) Round(roundToParam any) *PlainTime {
	if isUndefined(roundToParam) {
		throwType("options parameter is required")
	}
	roundTo := oneProp("smallestUnit", roundToParam)
	roundingIncrement := getRoundingIncrementOption(roundTo)
	roundingMode := getRoundingModeOption(roundTo, "halfExpand")
	smallestUnit := *getTemporalUnitValuedOption(roundTo, "smallestUnit", requiredSentinel)
	validateTemporalUnitValue(smallestUnit, "time")
	maxIncrements := map[string]float64{
		"hour":        24,
		"minute":      60,
		"second":      60,
		"millisecond": 1000,
		"microsecond": 1000,
		"nanosecond":  1000,
	}
	validateTemporalRoundingIncrement(roundingIncrement, maxIncrements[smallestUnit], false)

	time := roundTime(t.time, roundingIncrement, smallestUnit, roundingMode)
	return createTemporalTime(time)
}

// Equals is Temporal.PlainTime.prototype.equals.
func (t *PlainTime) Equals(otherParam any) bool {
	other := toTemporalTime(otherParam)
	return compareTimeRecord(t.time, other.time) == 0
}

// ToString is Temporal.PlainTime.prototype.toString.
func (t *PlainTime) ToString(options any) string {
	resolvedOptions := getOptionsObject(options)
	digits := getTemporalFractionalSecondDigitsOption(resolvedOptions)
	roundingMode := getRoundingModeOption(resolvedOptions, "trunc")
	smallestUnit := getTemporalUnitValuedOption(resolvedOptions, "smallestUnit")
	validateTemporalUnitValue(smallestUnit, "time")
	if smallestUnit != nil && *smallestUnit == "hour" {
		throwRange(`smallestUnit must be a time unit other than "hour"`)
	}
	rec := toSecondsStringPrecisionRecord(smallestUnit, digits)
	time := roundTime(t.time, rec.increment, rec.unit, roundingMode)
	return timeRecordToString(time, rec.precision)
}

// ToJSON is Temporal.PlainTime.prototype.toJSON.
func (t *PlainTime) ToJSON() string { return timeRecordToString(t.time, "auto") }

// ToLocaleString is Temporal.PlainTime.prototype.toLocaleString.
func (t *PlainTime) ToLocaleString(locales, options any) string {
	return createDateTimeFormat(locales, options, "time").Format(t)
}

func (t *PlainTime) jsValueOf() { valueOfThrows("PlainTime") }

func (t *PlainTime) jsToString() string { return t.ToString(nil) }

// PlainTimeFrom is Temporal.PlainTime.from.
func PlainTimeFrom(item, options any) *PlainTime { return toTemporalTime(item, options) }

// PlainTimeCompare is Temporal.PlainTime.compare.
func PlainTimeCompare(oneParam, twoParam any) float64 {
	one := toTemporalTime(oneParam)
	two := toTemporalTime(twoParam)
	return compareTimeRecord(one.time, two.time)
}

func (t *PlainTime) jsGet(key string) any {
	switch key {
	case "hour":
		return t.Hour()
	case "minute":
		return t.Minute()
	case "second":
		return t.Second()
	case "millisecond":
		return t.Millisecond()
	case "microsecond":
		return t.Microsecond()
	case "nanosecond":
		return t.Nanosecond()
	}
	return nil
}
