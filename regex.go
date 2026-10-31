package temporal

import (
	"regexp"
	"strings"
)

// The patterns below are composed exactly the way lib/regex.ts composes them. Go's regexp package
// resolves alternatives leftmost-first, so a match and its captures are the ones a backtracking
// engine finds. Three JavaScript features need a different spelling:
//   - \s is written as the explicit class of ECMAScript white space;
//   - the i flag is written as explicit classes, because (?i) in Go also folds U+017F to s and
//     U+212A to k, which JavaScript without the u flag does not;
//   - the (?!$) of the duration pattern is checked by execDuration.
const (
	reOffsetIdentifierNoCapture = `(?:[+-](?:[01][0-9]|2[0-3])(?::?[0-5][0-9])?)`
	reTzComponent               = `[A-Za-z._][A-Za-z._0-9+-]*`
	reTimeZoneID                = `(?:` + reOffsetIdentifierNoCapture + `|(?:` + reTzComponent + `)(?:\/(?:` + reTzComponent + `))*)`

	reYearpart  = `(?:[+-]\d{6}|\d{4})`
	reMonthpart = `(?:0[1-9]|1[0-2])`
	reDaypart   = `(?:0[1-9]|[12]\d|3[01])`
	// COMPAT: Note spec polyfill uses duplicate named capture groups here
	reDatesplit = `(?<yearpart>` + reYearpart + `)` +
		`(?:-(?<monthpartSep>` + reMonthpart + `)-` +
		`(?<daypartSep>` + reDaypart + `)|` +
		`(?<monthpartNoSep>` + reMonthpart + `)(?<daypartNoSep>` + reDaypart + `))`
	reSep                   = `:`
	reHourminutesecondNoSep = `(?<hourNoSep>\d{2})` +
		`(?:(?<minuteNoSep>\d{2}))?` +
		`(?:(?:(?<secondNoSep>\d{2}))` +
		`(?:(?:[.,](?<fractionNoSep>\d{1,9})))?)?`
	reHourminutesecondSep = `(?<hourSep>\d{2})` +
		`(?:` + reSep + `(?<minuteSep>\d{2}))?` +
		`(?:` + reSep + `(?:(?<secondSep>\d{2}))` +
		`(?:(?:[.,](?<fractionSep>\d{1,9})))?)?`
	reTimesplit             = `(?:(?:` + reHourminutesecondSep + `)|(?:` + reHourminutesecondNoSep + `))`
	reSign                  = `[+-]`
	reHour                  = `[01][0-9]|2[0-3]`
	reMinute                = `[0-5][0-9]`
	reSecond                = reMinute
	reOptionalMinSecWithSep = `(?:` + reSep + `)(?<offsetMinuteSep>` + reMinute + `)` +
		`(?:` + reSep + `(?<offsetSecondSep>` + reSecond + `)(?:[.,](?<offsetSubsecondsSep>\d{1,9})?)?)?`
	reOptionalMinSecNoSep = `(?<offsetMinuteNoSep>` + reMinute + `)` +
		`(?<offsetSecondNoSep>` + reSecond + `(?:[.,](?<offsetSubsecondsNoSep>\d{1,9})?)?)?`
	reOptionalMinSec   = `(?:` + reOptionalMinSecWithSep + `)|(?:` + reOptionalMinSecNoSep + `)`
	reOffsetWithParts  = `^(?<offsetSign>` + reSign + `)(?<offsetHour>` + reHour + `)(?:` + reOptionalMinSec + `)?$`
	reOffset           = `(?<offset>(?:` + reSign + `)(?:` + reHour + `)(?:` + reOptionalMinSec + `)?)`
	reOffsetpart       = `(?<z>[zZ])|` + reOffset + `?`
	reOffsetIdentifier = `([+-])([01][0-9]|2[0-3])(?::?([0-5][0-9])?)?`
	reAnnotation       = `\[(!)?([a-z_][a-z0-9_-]*)=([A-Za-z0-9]+(?:-[A-Za-z0-9]+)*)\]`
	// reJSSpace is the JavaScript \s.
	reJSSpace = `[\t\n\x0b\f\r \x{a0}\x{1680}\x{2000}-\x{200a}\x{2028}\x{2029}\x{202f}\x{205f}\x{3000}\x{feff}]`

	reZoneddatetime = `^` + reDatesplit +
		`(?:(?:[tT]|` + reJSSpace + `+)` + reTimesplit + `(?:` + reOffsetpart + `)?)?` +
		`(?:\[!?(?<timeZoneID>` + reTimeZoneID + `)\])?` +
		`(?<annotation>(?:` + reAnnotation + `)*)$`

	reTime = `^[tT]?` + reTimesplit +
		`(?:` + reOffsetpart + `)?` +
		`(?:\[!?((?<timeZoneID>` + reTimeZoneID + `))\])?` +
		`(?<annotation>(?:` + reAnnotation + `)*)$`

	// The short forms of YearMonth and MonthDay are only for the ISO calendar, but
	// annotations are still allowed, and will throw if the calendar annotation is
	// not ISO.
	// Non-ISO calendar YearMonth and MonthDay have to parse as a Temporal.PlainDate,
	// with the reference fields.
	// YYYYMM forbidden by ISO 8601 because ambiguous with YYMMDD, but allowed by
	// RFC 3339 and we don't allow 2-digit years, so we allow it.
	// Not ambiguous with HHMMSS because that requires a 'T' prefix
	// UTC offsets are not allowed, because they are not allowed with any date-only
	// format; also, YYYY-MM-UU is ambiguous with YYYY-MM-DD
	reYearmonth = `^(` + reYearpart + `)-?(` + reMonthpart + `)(?:\[!?` + reTimeZoneID + `\])?((?:` + reAnnotation + `)*)$`
	reMonthday  = `^(?:--)?(` + reMonthpart + `)-?(` + reDaypart + `)(?:\[!?` + reTimeZoneID + `\])?((?:` + reAnnotation + `)*)$`

	reNumberWithOptionalFraction = `(\d+)(?:[.,](\d{1,9}))?`

	reDurationDate = `(?:(\d+)[Yy])?(?:(\d+)[Mm])?(?:(\d+)[Ww])?(?:(\d+)[Dd])?`
	reDurationTime = `(?:` + reNumberWithOptionalFraction + `[Hh])?` +
		`(?:` + reNumberWithOptionalFraction + `[Mm])?` +
		`(?:` + reNumberWithOptionalFraction + `[Ss])?`
	reDuration = `^([+-])?[Pp]` + reDurationDate + `(?:[Tt]` + reDurationTime + `)?$`
)

var (
	annotationRe    = regexp.MustCompile(reAnnotation)
	zonedDateTimeRe = regexp.MustCompile(reZoneddatetime)
	timeRe          = regexp.MustCompile(reTime)
	yearMonthRe     = regexp.MustCompile(reYearmonth)
	monthDayRe      = regexp.MustCompile(reMonthday)
	durationRe      = regexp.MustCompile(reDuration)

	// TIMEZONE_IDENTIFIER carries the i flag in the original, which adds nothing to classes that
	// already hold both cases.
	timeZoneIdentifierRe = regexp.MustCompile(`^` + reTimeZoneID + `$`)
	offsetIdentifierRe   = regexp.MustCompile(`^` + reOffsetIdentifier + `$`)
	offsetRe             = regexp.MustCompile(`^` + reOffset + `$`)
	offsetWithPartsRe    = regexp.MustCompile(`^` + reOffsetWithParts + `$`)
	dateTimeSepRe        = regexp.MustCompile(`[tT ][0-9][0-9]`)
)

// reMatch is the result of exec. A group that did not participate reads as "": no group the callers
// tell apart from undefined can match the empty string.
type reMatch struct {
	re  *regexp.Regexp
	s   string
	idx []int
}

// reExec is RegExp.prototype.exec for a pattern without the g flag; nil means no match.
func reExec(re *regexp.Regexp, s string) *reMatch {
	idx := re.FindStringSubmatchIndex(s)
	if idx == nil {
		return nil
	}
	return &reMatch{re: re, s: s, idx: idx}
}

// at reads a group by its number.
func (m *reMatch) at(i int) string {
	if m.idx[2*i] < 0 {
		return ""
	}
	return m.s[m.idx[2*i]:m.idx[2*i+1]]
}

// group reads a group by its name.
func (m *reMatch) group(name string) string {
	i := m.re.SubexpIndex(name)
	if i < 0 {
		return ""
	}
	return m.at(i)
}

// execDuration runs the duration pattern. The original spells T(?!$): a T at the very end of the
// string cannot start the time part, and then nothing else can consume it, so the string fails.
func execDuration(s string) *reMatch {
	if strings.HasSuffix(s, "T") || strings.HasSuffix(s, "t") {
		return nil
	}
	return reExec(durationRe, s)
}
