package temporal

import (
	"math"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/js-temporal/temporal-polyfill-go/internal/intl"
)

func isOffsetTimeZoneIdentifier(str string) bool {
	return offsetIdentifierRe.MatchString(str)
}

func parseDateTimeUTCOffset(str string) float64 {
	match := reExec(offsetWithPartsRe, str)
	if match == nil {
		throwRange("invalid time zone offset: " + str + "; must match ±HH:MM[:SS.SSSSSSSSS]")
	}
	sign := 1.0
	if match.group("offsetSign") == "-" {
		sign = -1
	}
	hours := stringToNumber(match.group("offsetHour"))
	// A group that did not participate reads "" and +"" is 0, as (undefined ?? 0) gives. The NoSep
	// seconds group also holds the fraction, so seconds can be fractional or NaN here.
	minutes := stringToNumber(match.group("offsetMinuteSep") + match.group("offsetMinuteNoSep"))
	seconds := stringToNumber(match.group("offsetSecondSep") + match.group("offsetSecondNoSep"))
	subseconds := match.group("offsetSubsecondsSep") + match.group("offsetSubsecondsNoSep")
	nanoseconds := stringToNumber((subseconds + "000000000")[:9])
	offsetNanoseconds := float64(hours*60) + minutes
	offsetNanoseconds = float64(offsetNanoseconds*60) + seconds
	offsetNanoseconds = float64(offsetNanoseconds*1e9) + nanoseconds
	return float64(sign * offsetNanoseconds)
}

// namedTimeZoneIdentifierRecord is what GetAvailableNamedTimeZoneIdentifier returns.
type namedTimeZoneIdentifierRecord struct {
	identifier, primaryIdentifier string
}

var (
	canonicalTimeZoneIdsOnce  sync.Once
	canonicalTimeZoneIdsCache map[string]string
)

// specialTimeZoneCases are the second and third segments that break the capitalization pattern.
var specialTimeZoneCases = map[string]string{
	"Act":            "ACT",
	"Lhi":            "LHI",
	"Nsw":            "NSW",
	"Dar_Es_Salaam":  "Dar_es_Salaam",
	"Port_Of_Spain":  "Port_of_Spain",
	"Port-Au-Prince": "Port-au-Prince",
	"Isle_Of_Man":    "Isle_of_Man",
	"Comodrivadavia": "ComodRivadavia",
	"Knox_In":        "Knox_IN",
	"Dumontdurville": "DumontDUrville",
	"Mcmurdo":        "McMurdo",
	"Denoronha":      "DeNoronha",
	"Easterisland":   "EasterIsland",
	"Bajanorte":      "BajaNorte",
	"Bajasur":        "BajaSur",
}

func getAvailableNamedTimeZoneIdentifier(identifier string) *namedTimeZoneIdentifierRecord {
	// The most common case is when the identifier is a canonical time zone ID.
	// Fast-path that case by caching all canonical IDs.
	canonicalTimeZoneIdsOnce.Do(func() {
		cache := map[string]string{}
		for _, id := range intl.SupportedTimeZones() {
			cache[asciiLowercase(id)] = id
		}
		canonicalTimeZoneIdsCache = cache
	})

	lower := asciiLowercase(identifier)
	if primary := canonicalTimeZoneIdsCache[lower]; primary != "" {
		return &namedTimeZoneIdentifierRecord{primary, primary}
	}

	// It's not already a primary identifier, so get its primary identifier (or
	// return if it's not an available named time zone ID).
	primaryIdentifier, ok := intl.CanonicalTimeZone(lower)
	if !ok {
		return nil
	}

	// Special case this legacy identifier that is listed both in `backzone` and
	// `backward` in the TZDB. Work around implementations that incorrectly use
	// the `backward` data.
	if lower == "antarctica/south_pole" {
		primaryIdentifier = "Antarctica/McMurdo"
	}

	// Some legacy identifiers are aliases in ICU but not legal IANA identifiers.
	// Reject them even if the implementation's Intl supports them, as they are
	// not present in the IANA time zone database.
	if icuLegacyTimeZoneIDs[identifier] {
		throwRange(identifier + " is a legacy time zone identifier from ICU. Use " + primaryIdentifier + " instead")
	}

	// The identifier is an alias (a deprecated identifier that's a synonym for a
	// primary identifier), so we need to case-normalize the identifier to match
	// the IANA TZDB, e.g. america/new_york => America/New_York. An identifier
	// that resolved is ASCII, so this can work on bytes.
	// 1. capitalize the first letter of the identifier
	// 2. capitalize the letter after every slash, dash, or underscore delimiter
	chars := []byte(lower)
	for i := range chars {
		if i == 0 || lower[i-1] == '/' || lower[i-1] == '-' || lower[i-1] == '_' {
			if chars[i] >= 'a' && chars[i] <= 'z' {
				chars[i] -= 0x20
			}
		}
	}
	standardCase := string(chars)
	segments := strings.Split(standardCase, "/")

	if len(segments) == 1 {
		// If a single-segment legacy ID is 2-3 chars or contains a number or dash, then
		// (except for the "GB-Eire" special case) the case-normalized form is uppercase.
		// These are: GMT+0, GMT-0, GB, NZ, PRC, ROC, ROK, UCT, GMT, GMT0, CET, CST6CDT,
		// EET, EST, HST, MET, MST, MST7MDT, PST8PDT, WET, NZ-CHAT, and W-SU.
		// Otherwise it's standard form: first letter capitalized, e.g. Iran, Egypt, Hongkong
		if lower == "gb-eire" {
			return &namedTimeZoneIdentifierRecord{"GB-Eire", primaryIdentifier}
		}
		if len(lower) <= 3 || strings.ContainsAny(lower, "-0123456789") {
			return &namedTimeZoneIdentifierRecord{strings.ToUpper(lower), primaryIdentifier}
		}
		return &namedTimeZoneIdentifierRecord{segments[0], primaryIdentifier}
	}

	// All Etc zone names are uppercase except three exceptions.
	if segments[0] == "Etc" {
		etcName := strings.ToUpper(segments[1])
		if segments[1] == "Zulu" || segments[1] == "Greenwich" || segments[1] == "Universal" {
			etcName = segments[1]
		}
		return &namedTimeZoneIdentifierRecord{"Etc/" + etcName, primaryIdentifier}
	}

	// Legacy US identifiers like US/Alaska or US/Indiana-Starke are 2 segments and use standard form.
	if segments[0] == "Us" {
		return &namedTimeZoneIdentifierRecord{"US/" + segments[1], primaryIdentifier}
	}

	// For multi-segment IDs, there's a few special cases in the second/third segments
	if s, ok := specialTimeZoneCases[segments[1]]; ok {
		segments[1] = s
	}
	if len(segments) > 2 {
		if s, ok := specialTimeZoneCases[segments[2]]; ok {
			segments[2] = s
		}
	}
	return &namedTimeZoneIdentifierRecord{strings.Join(segments, "/"), primaryIdentifier}
}

func getNamedTimeZoneOffsetNanosecondsImpl(id string, epochMilliseconds float64) float64 {
	parts := getFormatterParts(id, epochMilliseconds)
	millisecond := jsMod(epochMilliseconds, 1000)
	if millisecond < 0 {
		millisecond += 1000
	}
	utc := getUTCEpochMilliseconds(
		isoDate{parts.year, parts.month, parts.day},
		timeRecord{hour: parts.hour, minute: parts.minute, second: parts.second, millisecond: millisecond},
	)
	return float64((utc - epochMilliseconds) * 1e6)
}

func getNamedTimeZoneOffsetNanoseconds(id string, epochNanoseconds *big.Int) float64 {
	// Optimization: We get the offset nanoseconds only with millisecond
	// resolution, assuming that time zone offset changes don't happen in the
	// middle of a millisecond
	return getNamedTimeZoneOffsetNanosecondsImpl(id, epochNsToMs(epochNanoseconds, "floor"))
}

func formatOffsetTimeZoneIdentifier(offsetMinutes float64) string {
	sign := "+"
	if offsetMinutes < 0 {
		sign = "-"
	}
	absoluteMinutes := math.Abs(offsetMinutes)
	hour := math.Floor(absoluteMinutes / 60)
	minute := jsMod(absoluteMinutes, 60)
	timeString := formatTimeString(hour, minute, 0, 0, "minute")
	return sign + timeString
}

func formatDateTimeUTCOffsetRounded(offsetNanosecondsParam float64) string {
	offsetNanoseconds := roundNumberToIncrement(offsetNanosecondsParam, 60e9, "halfExpand")
	return formatOffsetTimeZoneIdentifier(offsetNanoseconds / 60e9)
}

// doubleToInteger is V8's DoubleToInteger: truncation that gives +0 for NaN and for -0.
func doubleToInteger(x float64) float64 {
	if math.IsNaN(x) || x == 0 {
		return 0
	}
	if math.IsInf(x, 0) {
		return x
	}
	return math.Trunc(x) + 0
}

// makeDay is MakeDay as V8 computes it.
func makeDay(year, month, date float64) float64 {
	if !(-1000000.0 <= year && year <= 1000000.0) || !(-10000000.0 <= month && month <= 10000000.0) ||
		math.IsNaN(date) || math.IsInf(date, 0) {
		return math.NaN()
	}
	y := int64(year)
	m := int64(month)
	y += m / 12
	m %= 12
	if m < 0 {
		m += 12
		y -= 1
	}
	const yearDelta = 399999
	const baseDay = 365*(1970+yearDelta) + (1970+yearDelta)/4 - (1970+yearDelta)/100 + (1970+yearDelta)/400
	dayFromYear := 365*(y+yearDelta) + (y+yearDelta)/4 - (y+yearDelta)/100 + (y+yearDelta)/400 - baseDay
	if y%4 != 0 || (y%100 == 0 && y%400 != 0) {
		dayFromYear += [...]int64{0, 31, 59, 90, 120, 151, 181, 212, 243, 273, 304, 334}[m]
	} else {
		dayFromYear += [...]int64{0, 31, 60, 91, 121, 152, 182, 213, 244, 274, 305, 335}[m]
	}
	return float64(dayFromYear-1) + doubleToInteger(date)
}

// makeTime is MakeTime.
func makeTime(hour, minute, second, ms float64) float64 {
	if !isFiniteNumber(hour) || !isFiniteNumber(minute) || !isFiniteNumber(second) || !isFiniteNumber(ms) {
		return math.NaN()
	}
	h := doubleToInteger(hour)
	m := doubleToInteger(minute)
	s := doubleToInteger(second)
	milli := doubleToInteger(ms)
	return float64(h*3600000) + float64(m*60000) + float64(s*1000) + milli
}

// makeDate is MakeDate.
func makeDate(day, tm float64) float64 {
	if !isFiniteNumber(day) || !isFiniteNumber(tm) {
		return math.NaN()
	}
	return tm + float64(day*86400000)
}

// timeClip is TimeClip.
func timeClip(tm float64) float64 {
	if -8.64e15 <= tm && tm <= 8.64e15 {
		return doubleToInteger(tm)
	}
	return math.NaN()
}

func isFiniteNumber(x float64) bool { return !math.IsNaN(x) && !math.IsInf(x, 0) }

// dateUTC is Date.UTC with all seven arguments.
func dateUTC(year, month, day, hour, minute, second, ms float64) float64 {
	if !math.IsNaN(year) {
		y := doubleToInteger(year)
		if 0 <= y && y <= 99 {
			year = 1900 + y
		}
	}
	return timeClip(makeDate(makeDay(year, month, day), makeTime(hour, minute, second, ms)))
}

func getUTCEpochMilliseconds(iso isoDate, tm timeRecord) float64 {
	year, month, day := iso.year, iso.month, iso.day
	hour, minute, second, millisecond := tm.hour, tm.minute, tm.second, tm.millisecond // ignored: microsecond, nanosecond

	// The pattern of leap years in the ISO 8601 calendar repeats every 400
	// years. To avoid overflowing at the edges of the range, we reduce the year
	// to the remainder after dividing by 400, and then add back all the
	// nanoseconds from the multiples of 400 years at the end.
	reducedYear := jsMod(year, 400)
	yearCycles := (year - reducedYear) / 400

	// `Date.UTC(year, monthIndex, days)` maps year [0, 99] to [1900, 1999], so
	// avoid that range.
	extraCycles := 0.0
	if reducedYear >= 0 {
		extraCycles = 5
	}
	ms := dateUTC(reducedYear+float64(400*extraCycles), month-1, day, hour, minute, second, millisecond)

	return ms + float64(msIn400YearCycle*(yearCycles-extraCycles))
}

func getUTCEpochNanoseconds(dt isoDateTime) *big.Int {
	ms := getUTCEpochMilliseconds(dt.isoDate, dt.time)
	subMs := float64(dt.time.microsecond*1e3) + dt.time.nanosecond
	return new(big.Int).Add(epochMsToNs(ms), bigFromNumber(subMs))
}

// floorDiv is integer division rounding down.
func floorDiv(a, b int64) int64 {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

// civilFromDays converts days since the epoch to a proleptic Gregorian year, month and day.
func civilFromDays(days int64) (year, month, day int64) {
	z := days + 719468
	era := floorDiv(z, 146097)
	doe := z - era*146097
	yoe := (doe - doe/1460 + doe/36524 - doe/146096) / 365
	doy := doe - (365*yoe + yoe/4 - yoe/100)
	mp := (5*doy + 2) / 153
	day = doy - (153*mp+2)/5 + 1
	if mp < 10 {
		month = mp + 3
	} else {
		month = mp - 9
	}
	year = yoe + era*400
	if month <= 2 {
		year++
	}
	return year, month, day
}

// isoPartsFromEpoch is what GetISOPartsFromEpoch returns.
type isoPartsFromEpoch struct {
	epochMilliseconds float64
	isoDate           isoDate
	time              timeRecord
}

func getISOPartsFromEpoch(epochNanoseconds *big.Int) isoPartsFromEpoch {
	epochMilliseconds := epochNsToMs(epochNanoseconds, "trunc")
	nanos := bigToNumber(new(big.Int).Rem(epochNanoseconds, bigMillion))
	if nanos < 0 {
		nanos += 1e6
		epochMilliseconds -= 1
	}
	microsecond := jsMod(math.Floor(nanos/1e3), 1e3)
	nanosecond := jsMod(nanos, 1e3)

	// new Date(epochMilliseconds) with its getUTC* getters; an invalid Date reads NaN.
	clipped := timeClip(epochMilliseconds)
	if math.IsNaN(clipped) {
		nan := math.NaN()
		return isoPartsFromEpoch{
			epochMilliseconds,
			isoDate{nan, nan, nan},
			timeRecord{hour: nan, minute: nan, second: nan, millisecond: nan, microsecond: microsecond, nanosecond: nanosecond},
		}
	}
	ms := int64(clipped)
	days := floorDiv(ms, 86400000)
	inDay := ms - days*86400000
	year, month, day := civilFromDays(days)

	return isoPartsFromEpoch{
		epochMilliseconds,
		isoDate{float64(year), float64(month), float64(day)},
		timeRecord{
			hour:        float64(inDay / 3600000),
			minute:      float64(inDay / 60000 % 60),
			second:      float64(inDay / 1000 % 60),
			millisecond: float64(inDay % 1000),
			microsecond: microsecond,
			nanosecond:  nanosecond,
		},
	}
}

func getNamedTimeZoneDateTimeParts(id string, epochNanoseconds *big.Int) isoDateTime {
	isoParts := getISOPartsFromEpoch(epochNanoseconds)
	parts := getFormatterParts(id, isoParts.epochMilliseconds)
	return balanceISODateTime(
		parts.year, parts.month, parts.day, parts.hour, parts.minute, parts.second,
		isoParts.time.millisecond, isoParts.time.microsecond, isoParts.time.nanosecond,
	)
}

// Most time zones never transition twice within a short span of days. We still
// accommodate twitchy zones, albeit at a performance penalty. 19 days is the
// default window, past which we'd need to start adding many more special cases.
func searchWindowForTransitions(id string) float64 {
	switch id {
	case "Africa/El_Aaiun":
		return float64(dayMs * 17)
	case "America/Argentina/Tucuman":
		return float64(dayMs * 12)
	case "Europe/Tirane":
		return float64(dayMs * 11)
	case "Europe/Riga":
		return float64(dayMs * 10)
	case "Europe/Simferopol", "Europe/Vienna":
		return float64(dayMs * 9)
	case "Africa/Tunis":
		return float64(dayMs * 8)
	case "America/Boa_Vista", "America/Fortaleza", "America/Maceio", "America/Noronha", "America/Recife",
		"Asia/Gaza",   // dubious, only in future calculations
		"Asia/Hebron", // ditto
		"Brazil/DeNoronha":
		return float64(dayMs * 6)
	}
	return float64(dayMs * 19)
}

// dateNow is Date.now().
func dateNow() float64 { return float64(time.Now().UnixMilli()) }

func getNamedTimeZoneNextTransition(id string, epochNanoseconds *big.Int) *big.Int {
	if id == "UTC" {
		return nil // UTC fast path
	}

	// Optimization: we floor the instant to the previous millisecond boundary
	// so that we can do Number math instead of BigInt math. This assumes that
	// time zone transitions don't happen in the middle of a millisecond.
	epochMilliseconds := epochNsToMs(epochNanoseconds, "floor")
	if epochMilliseconds < beforeFirstDST {
		return getNamedTimeZoneNextTransition(id, epochMsToNs(beforeFirstDST))
	}

	// Optimization: the farthest that we'll look for a next transition is 3 years
	// after the later of epochNanoseconds or the current time. If there are no
	// transitions found before then, we'll assume that there will not be any more
	// transitions after that.
	now := dateNow()
	base := jsMax(epochMilliseconds, now)
	uppercap := base + float64(dayMs*366*3)
	leftMs := epochMilliseconds
	leftOffsetNs := getNamedTimeZoneOffsetNanosecondsImpl(id, leftMs)
	rightMs := leftMs
	rightOffsetNs := leftOffsetNs
	searchWindow := searchWindowForTransitions(id)
	for leftOffsetNs == rightOffsetNs && leftMs < uppercap {
		rightMs = leftMs + searchWindow
		if rightMs > msMax {
			return nil
		}
		rightOffsetNs = getNamedTimeZoneOffsetNanosecondsImpl(id, rightMs)
		if leftOffsetNs == rightOffsetNs {
			leftMs = rightMs
		}
	}
	if leftOffsetNs == rightOffsetNs {
		return nil
	}
	result := bisect(
		func(epochMs float64) float64 { return getNamedTimeZoneOffsetNanosecondsImpl(id, epochMs) },
		leftMs,
		rightMs,
		leftOffsetNs,
		rightOffsetNs,
	)
	return epochMsToNs(result)
}

func getNamedTimeZonePreviousTransition(id string, epochNanoseconds *big.Int) *big.Int {
	if id == "UTC" {
		return nil // UTC fast path
	}

	// Optimization: we raise the instant to the next millisecond boundary so
	// that we can do Number math instead of BigInt math. This assumes that time
	// zone transitions don't happen in the middle of a millisecond.
	epochMilliseconds := epochNsToMs(epochNanoseconds, "ceil")

	// Optimization: if the instant is more than 3 years in the future and there
	// are no transitions between the present day and 3 years from now, assume
	// there are none after.
	now := dateNow()
	lookahead := now + float64(dayMs*366*3)
	if epochMilliseconds > lookahead {
		prevBeforeLookahead := getNamedTimeZonePreviousTransition(id, epochMsToNs(lookahead))
		if prevBeforeLookahead == nil || prevBeforeLookahead.Cmp(epochMsToNs(now)) < 0 {
			return prevBeforeLookahead
		}
	}

	// We assume most time zones either have regular DST rules that extend
	// indefinitely into the future, or they have no DST transitions between now
	// and next year. Africa/Casablanca and Africa/El_Aaiun are unique cases
	// that fit neither of these. Their irregular DST transitions are
	// precomputed until 2087 in the current time zone database, so requesting
	// the previous transition for an instant far in the future may take an
	// extremely long time as it loops backward 2 weeks at a time.
	if id == "Africa/Casablanca" || id == "Africa/El_Aaiun" {
		lastPrecomputed := dateUTC(2088, 0, 1, 0, 0, 0, 0) // 2088-01-01T00Z
		if lastPrecomputed < epochMilliseconds {
			return getNamedTimeZonePreviousTransition(id, epochMsToNs(lastPrecomputed))
		}
	}

	rightMs := epochMilliseconds - 1
	if rightMs < beforeFirstDST {
		return nil
	}
	rightOffsetNs := getNamedTimeZoneOffsetNanosecondsImpl(id, rightMs)
	leftMs := rightMs
	leftOffsetNs := rightOffsetNs
	searchWindow := searchWindowForTransitions(id)
	for rightOffsetNs == leftOffsetNs && rightMs > beforeFirstDST {
		leftMs = rightMs - searchWindow
		if leftMs < beforeFirstDST {
			return nil
		}
		leftOffsetNs = getNamedTimeZoneOffsetNanosecondsImpl(id, leftMs)
		if rightOffsetNs == leftOffsetNs {
			rightMs = leftMs
		}
	}
	if rightOffsetNs == leftOffsetNs {
		return nil
	}
	result := bisect(
		func(epochMs float64) float64 { return getNamedTimeZoneOffsetNanosecondsImpl(id, epochMs) },
		leftMs,
		rightMs,
		leftOffsetNs,
		rightOffsetNs,
	)
	return epochMsToNs(result)
}

// formatterParts holds the fields parseFromEnUsFormat reads out of the en-US format.
type formatterParts struct {
	year, month, day, hour, minute, second float64
}

// intlTimeZoneCache plays IntlDateTimeFormatEnUsCache: lowercase identifier to resolved zone.
var intlTimeZoneCache sync.Map

// getFormatterParts behaves as formatter.format followed by parseFromEnUsFormat: the local fields
// on the proleptic Gregorian calendar, the year astronomical, with the second floored as ICU does.
func getFormatterParts(timeZone string, epochMilliseconds float64) formatterParts {
	lowercaseIdentifier := asciiLowercase(timeZone)
	var canonical string
	if cached, ok := intlTimeZoneCache.Load(lowercaseIdentifier); ok {
		canonical = cached.(string)
	} else {
		primary, ok := intl.CanonicalTimeZone(lowercaseIdentifier)
		if !ok {
			throwRange("Invalid time zone specified: " + lowercaseIdentifier)
		}
		intlTimeZoneCache.Store(lowercaseIdentifier, primary)
		canonical = primary
	}

	clipped := timeClip(epochMilliseconds)
	if math.IsNaN(clipped) {
		throwRange("Invalid time value")
	}
	epochSeconds := floorDiv(int64(clipped), 1000)
	local := epochSeconds + intl.OffsetSeconds(canonical, epochSeconds)
	days := floorDiv(local, 86400)
	inDay := local - days*86400
	year, month, day := civilFromDays(days)
	return formatterParts{
		year:   float64(year),
		month:  float64(month),
		day:    float64(day),
		hour:   float64(inDay / 3600),
		minute: float64(inDay / 60 % 60),
		second: float64(inDay % 60),
	}
}

// The goal of this function is to find the exact time(s) that correspond to a
// calendar date and clock time in a particular time zone. Normally there will
// be only one match. But for repeated clock times after backwards transitions
// (like when DST ends) there may be two matches. And for skipped clock times
// after forward transitions, there will be no matches.
func getNamedTimeZoneEpochNanoseconds(id string, dt isoDateTime) []*big.Int {
	// Get the offset of one day before and after the requested calendar date and
	// clock time, avoiding overflows if near the edge of the Instant range.
	ns := getUTCEpochNanoseconds(dt)
	// Note, ns may be up to DAY_NANOS outside the NS_MIN...NS_MAX range here,
	// even if isoDateTime represents a valid wall time in a non-UTC time zone.
	// The TZDB does not describe any transitions that would occur in that extra
	// day, so we just clip in order not to throw in GetFormatterParts.
	nsEarlier := new(big.Int).Sub(ns, dayNanosJSBI)
	if nsEarlier.Cmp(nsMin) < 0 {
		nsEarlier = nsMin
	}
	nsLater := new(big.Int).Add(ns, dayNanosJSBI)
	if nsLater.Cmp(nsMax) > 0 {
		nsLater = nsMax
	}
	earlierOffsetNs := getNamedTimeZoneOffsetNanoseconds(id, nsEarlier)
	laterOffsetNs := getNamedTimeZoneOffsetNanoseconds(id, nsLater)

	// If before and after offsets are the same, then we assume there was no
	// offset transition in between, and therefore only one exact time can
	// correspond to the provided calendar date and clock time. But if they're
	// different, then there was an offset transition in between, so test both
	// offsets to see which one(s) will yield a matching exact time.
	found := []float64{earlierOffsetNs, laterOffsetNs}
	if earlierOffsetNs == laterOffsetNs {
		found = found[:1]
	}
	candidates := []*big.Int{}
	for _, offsetNanoseconds := range found {
		epochNanoseconds := new(big.Int).Sub(ns, bigFromNumber(offsetNanoseconds))
		parts := getNamedTimeZoneDateTimeParts(id, epochNanoseconds)
		if compareISODateTime(dt, parts) != 0 {
			continue
		}
		validateEpochNanoseconds(epochNanoseconds)
		candidates = append(candidates, epochNanoseconds)
	}
	return candidates
}

func bisect(getState func(epochMs float64) float64, leftParam, rightParam, lstateParam, rstateParam float64) float64 {
	left := leftParam
	right := rightParam
	lstate := lstateParam
	rstate := rstateParam
	for right-left > 1 {
		middle := jsTrunc((left + right) / 2)
		mstate := getState(middle)
		if mstate == lstate {
			left = middle
			lstate = mstate
		} else if mstate == rstate {
			right = middle
			rstate = mstate
		} else {
			panic("assertion failure: code should not be reached because invalid state in bisection " +
				numberToString(lstate) + " - " + numberToString(mstate) + " - " + numberToString(rstate))
		}
	}
	return right
}
