package temporal

import "math/big"

// NewInstant is new Temporal.Instant(epochNanoseconds). A Go caller always passes the argument, so
// the original's missing-argument check cannot fire: nil is an explicit undefined.
func NewInstant(epochNanoseconds any) *Instant {
	ns := toBigInt(epochNanoseconds)
	return createTemporalInstant(ns)
}

// EpochMilliseconds is the epochMilliseconds getter.
func (i *Instant) EpochMilliseconds() float64 {
	value := i.epochNs
	return epochNsToMs(value, "floor")
}

// EpochNanoseconds is the epochNanoseconds getter.
func (i *Instant) EpochNanoseconds() *big.Int { return toBigIntExternal(i.epochNs) }

// Add is Temporal.Instant.prototype.add.
func (i *Instant) Add(temporalDurationLike any) *Instant {
	return addDurationToInstant("add", i, temporalDurationLike)
}

// Subtract is Temporal.Instant.prototype.subtract.
func (i *Instant) Subtract(temporalDurationLike any) *Instant {
	return addDurationToInstant("subtract", i, temporalDurationLike)
}

// Until is Temporal.Instant.prototype.until.
func (i *Instant) Until(other, options any) *Duration {
	return differenceTemporalInstant("until", i, other, options)
}

// Since is Temporal.Instant.prototype.since.
func (i *Instant) Since(other, options any) *Duration {
	return differenceTemporalInstant("since", i, other, options)
}

// Round is Temporal.Instant.prototype.round.
func (i *Instant) Round(roundToParam any) *Instant {
	if isUndefined(roundToParam) {
		throwType("options parameter is required")
	}
	roundTo := oneProp("smallestUnit", roundToParam)
	roundingIncrement := getRoundingIncrementOption(roundTo)
	roundingMode := getRoundingModeOption(roundTo, "halfExpand")
	smallestUnit := *getTemporalUnitValuedOption(roundTo, "smallestUnit", requiredSentinel)
	validateTemporalUnitValue(smallestUnit, "time")
	maximumIncrements := map[string]float64{
		"hour":        24,
		"minute":      1440,
		"second":      86400,
		"millisecond": 86400e3,
		"microsecond": 86400e6,
		"nanosecond":  86400e9,
	}
	validateTemporalRoundingIncrement(roundingIncrement, maximumIncrements[smallestUnit], true)
	ns := i.epochNs
	roundedNs := roundTemporalInstant(ns, roundingIncrement, smallestUnit, roundingMode)
	return createTemporalInstant(roundedNs)
}

// Equals is Temporal.Instant.prototype.equals.
func (i *Instant) Equals(otherParam any) bool {
	other := toTemporalInstant(otherParam)
	return i.epochNs.Cmp(other.epochNs) == 0
}

// ToString is Temporal.Instant.prototype.toString.
func (i *Instant) ToString(options any) string {
	resolvedOptions := getOptionsObject(options)
	digits := getTemporalFractionalSecondDigitsOption(resolvedOptions)
	roundingMode := getRoundingModeOption(resolvedOptions, "trunc")
	smallestUnit := getTemporalUnitValuedOption(resolvedOptions, "smallestUnit")
	timeZone := objGet(resolvedOptions, "timeZone")
	validateTemporalUnitValue(smallestUnit, "time")
	if smallestUnit != nil && *smallestUnit == "hour" {
		throwRange(`smallestUnit must be a time unit other than "hour"`)
	}
	if !isUndefined(timeZone) {
		timeZone = toTemporalTimeZoneIdentifier(timeZone)
	}
	rec := toSecondsStringPrecisionRecord(smallestUnit, digits)
	ns := i.epochNs
	roundedNs := roundTemporalInstant(ns, rec.increment, rec.unit, roundingMode)
	roundedInstant := createTemporalInstant(roundedNs)
	return temporalInstantToString(roundedInstant, timeZone, rec.precision)
}

// ToJSON is Temporal.Instant.prototype.toJSON.
func (i *Instant) ToJSON() string { return temporalInstantToString(i, nil, "auto") }

// ToLocaleString is Temporal.Instant.prototype.toLocaleString.
func (i *Instant) ToLocaleString(locales, options any) string {
	return NewDateTimeFormat(locales, options).Format(i)
}

func (i *Instant) jsValueOf() { valueOfThrows("Instant") }

func (i *Instant) jsToString() string { return i.ToString(nil) }

// ToZonedDateTimeISO is Temporal.Instant.prototype.toZonedDateTimeISO.
func (i *Instant) ToZonedDateTimeISO(timeZoneParam any) *ZonedDateTime {
	timeZone := toTemporalTimeZoneIdentifier(timeZoneParam)
	return createTemporalZonedDateTime(i.epochNs, timeZone, "iso8601")
}

// InstantFromEpochMilliseconds is Temporal.Instant.fromEpochMilliseconds.
func InstantFromEpochMilliseconds(epochMilliseconds any) *Instant {
	epochNanoseconds := epochMsToNs(toNumber(epochMilliseconds))
	return createTemporalInstant(epochNanoseconds)
}

// InstantFromEpochNanoseconds is Temporal.Instant.fromEpochNanoseconds.
func InstantFromEpochNanoseconds(epochNanosecondsParam any) *Instant {
	epochNanoseconds := toBigInt(epochNanosecondsParam)
	return createTemporalInstant(epochNanoseconds)
}

// InstantFrom is Temporal.Instant.from.
func InstantFrom(item any) *Instant { return toTemporalInstant(item) }

// InstantCompare is Temporal.Instant.compare.
func InstantCompare(oneParam, twoParam any) float64 {
	one := toTemporalInstant(oneParam)
	two := toTemporalInstant(twoParam)
	return float64(one.epochNs.Cmp(two.epochNs))
}

func (i *Instant) jsGet(key string) any {
	switch key {
	case "epochMilliseconds":
		return i.EpochMilliseconds()
	case "epochNanoseconds":
		return i.EpochNanoseconds()
	}
	return nil
}
