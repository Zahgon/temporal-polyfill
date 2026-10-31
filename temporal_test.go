// Generated from recordings of the original by make_gold_test.py.
package temporal_test

import (
	"fmt"
	"math"
	"math/big"
	"os"
	"strconv"
	"strings"
	"testing"

	temporal "github.com/js-temporal/temporal-polyfill-go"
)

// showMessages turns on exception messages in the rendered lines; only the error cases compare them.
var showMessages = false

var _ = os.Args
var _ = math.NaN
var _ = bigInt

var rows []string

// out prints a value, or the exception evaluating it raised; only *temporal.Error counts as one.
func out(fn func() any) {
	defer func() {
		if r := recover(); r != nil {
			rows = append(rows, thrown(r))
		}
	}()
	rows = append(rows, render(fn()))
}

func thrown(r any) string {
	if e, ok := r.(*temporal.Error); ok {
		if showMessages {
			return "throw " + e.Name + ": " + quote(e.Message)
		}
		return "throw " + e.Name
	}
	return "panic " + quote(fmt.Sprint(r))
}

func bigInt(s string) *big.Int {
	n, _ := new(big.Int).SetString(s, 10)
	return n
}

// num is ECMAScript Number::toString.
func num(f float64) string {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return "Infinity"
	case math.IsInf(f, -1):
		return "-Infinity"
	case f == 0:
		if math.Signbit(f) {
			return "-0"
		}
		return "0"
	}
	sign := ""
	if f < 0 {
		sign, f = "-", -f
	}
	mant, exp, _ := strings.Cut(strconv.FormatFloat(f, 'e', -1, 64), "e")
	digits := strings.Replace(mant, ".", "", 1)
	e, _ := strconv.Atoi(exp)
	n, k := e+1, len(digits)
	switch {
	case k <= n && n <= 21:
		return sign + digits + strings.Repeat("0", n-k)
	case 0 < n && n <= 21:
		return sign + digits[:n] + "." + digits[n:]
	case -6 < n && n <= 0:
		return sign + "0." + strings.Repeat("0", -n) + digits
	}
	es := "+"
	a := n - 1
	if a < 0 {
		es, a = "-", -a
	}
	if k == 1 {
		return sign + digits + "e" + es + strconv.Itoa(a)
	}
	return sign + digits[:1] + "." + digits[1:] + "e" + es + strconv.Itoa(a)
}

// quote is JSON.stringify of a string.
func quote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 0x20 {
				fmt.Fprintf(&b, `\u%04x`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

func opt(parts []string, key string, v any) []string {
	s := render(v)
	if s == "undefined" {
		return parts
	}
	return append(parts, key+"="+s)
}

func render(v any) string {
	switch x := v.(type) {
	case nil:
		return "undefined"
	case bool:
		return strconv.FormatBool(x)
	case float64:
		return num(x)
	case string:
		return quote(x)
	case *big.Int:
		return x.String() + "n"
	case *string:
		if x == nil {
			return "undefined"
		}
		return quote(*x)
	case *bool:
		if x == nil {
			return "undefined"
		}
		return strconv.FormatBool(*x)
	case *float64:
		if x == nil {
			return "undefined"
		}
		return num(*x)
	case *temporal.Instant:
		return "Instant(" + x.ToString(nil) + ")"
	case *temporal.PlainDate:
		return "PlainDate(" + x.ToString(nil) + ")"
	case *temporal.PlainDateTime:
		return "PlainDateTime(" + x.ToString(nil) + ")"
	case *temporal.PlainTime:
		return "PlainTime(" + x.ToString(nil) + ")"
	case *temporal.PlainYearMonth:
		return "PlainYearMonth(" + x.ToString(nil) + ")"
	case *temporal.PlainMonthDay:
		return "PlainMonthDay(" + x.ToString(nil) + ")"
	case *temporal.ZonedDateTime:
		if x == nil {
			return "null"
		}
		return "ZonedDateTime(" + x.ToString(nil) + ")"
	case *temporal.Duration:
		return "Duration(" + x.ToString(nil) + ")"
	case []temporal.Part:
		parts := make([]string, len(x))
		for i, p := range x {
			parts[i] = "{" + quote(p.Type) + ":" + quote(p.Value) + "}"
		}
		return "[" + strings.Join(parts, ",") + "]"
	case []temporal.RangePart:
		parts := make([]string, len(x))
		for i, p := range x {
			parts[i] = "{" + quote(p.Type) + ":" + quote(p.Value) + "," + quote(p.Source) + "}"
		}
		return "[" + strings.Join(parts, ",") + "]"
	case temporal.ResolvedOptions:
		parts := []string{"locale=" + quote(x.Locale), "calendar=" + quote(x.Calendar),
			"numberingSystem=" + quote(x.NumberingSystem), "timeZone=" + quote(x.TimeZone)}
		parts = opt(parts, "hourCycle", x.HourCycle)
		parts = opt(parts, "hour12", x.Hour12)
		parts = opt(parts, "weekday", x.Weekday)
		parts = opt(parts, "era", x.Era)
		parts = opt(parts, "year", x.Year)
		parts = opt(parts, "month", x.Month)
		parts = opt(parts, "day", x.Day)
		parts = opt(parts, "dayPeriod", x.DayPeriod)
		parts = opt(parts, "hour", x.Hour)
		parts = opt(parts, "minute", x.Minute)
		parts = opt(parts, "second", x.Second)
		parts = opt(parts, "fractionalSecondDigits", x.FractionalSecondDigits)
		parts = opt(parts, "timeZoneName", x.TimeZoneName)
		parts = opt(parts, "dateStyle", x.DateStyle)
		parts = opt(parts, "timeStyle", x.TimeStyle)
		return "{" + strings.Join(parts, ",") + "}"
	}
	return "?" + fmt.Sprintf("%T", v)
}

func run(fn func()) {
	defer func() {
		if r := recover(); r != nil {
			rows = []string{thrown(r)}
		}
	}()
	fn()
}

// goldenCase is a sequence of calls and the lines the original printed for it.
type goldenCase struct {
	name     string
	run      func()
	expected []string
}

// TestAgainstOriginal replays calls whose output was recorded from the original on Node.js 24.22.1.
func TestAgainstOriginal(t *testing.T) {
	groups := map[string][]goldenCase{
		"cal_arithmetic":            cal_arithmeticCases,
		"cal_chinese_dangi":         cal_chinese_dangiCases,
		"cal_coptic_ethiopic":       cal_coptic_ethiopicCases,
		"cal_hebrew":                cal_hebrewCases,
		"cal_islamic":               cal_islamicCases,
		"cal_japanese_roc_buddhist": cal_japanese_roc_buddhistCases,
		"cal_persian_indian":        cal_persian_indianCases,
		"cal_year_month_day":        cal_year_month_dayCases,
		"duration_relative":         duration_relativeCases,
		"errors":                    errorsCases,
		"format_parts":              format_partsCases,
		"format_range":              format_rangeCases,
		"instant_math":              instant_mathCases,
		"iso_dates":                 iso_datesCases,
		"locale_strings":            locale_stringsCases,
		"parse_strings":             parse_stringsCases,
		"plain_time":                plain_timeCases,
		"zone_disambiguation":       zone_disambiguationCases,
		"zone_identifiers":          zone_identifiersCases,
		"zone_offsets":              zone_offsetsCases,
	}
	for group, cases := range groups {
		for _, c := range cases {
			t.Run(group+"/"+c.name, func(t *testing.T) {
				showMessages = group == "errors"
				rows = nil
				run(c.run)
				if len(rows) != len(c.expected) {
					t.Fatalf("got %d lines, want %d: %q", len(rows), len(c.expected), rows)
				}
				for i := range rows {
					if rows[i] != c.expected[i] {
						t.Errorf("line %d: got %s, want %s", i, rows[i], c.expected[i])
					}
				}
			})
		}
	}
}

var cal_arithmeticCases = []goldenCase{
	{"cal_arithmetic_001", func() {
		a := temporal.PlainDateFrom("2148-10-22[u-ca=islamic-umalqura]", nil)
		_ = a
		out(func() any {
			return a.Add(temporal.DurationBag{Years: temporal.V(float64(-314))}, nil).Until(a, temporal.DifferenceOptions{LargestUnit: temporal.V("day")})
		})
	}, []string{"Duration(P111270D)"}},
	{"cal_arithmetic_002", func() {
		a := temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("islamic-tbla"), Year: temporal.V(float64(1528)), Month: temporal.V(float64(4)), Day: temporal.V(float64(9))}, nil)
		_ = a
		out(func() any { return a.Add(temporal.DurationBag{Months: temporal.V(float64(1))}, nil) })
		out(func() any { return a.Add(temporal.DurationBag{Months: temporal.V(float64(6))}, nil) })
		out(func() any { return a.Add(temporal.DurationBag{Months: temporal.V(float64(12))}, nil) })
		out(func() any { return a.Add(temporal.DurationBag{Months: temporal.V(float64(13))}, nil) })
		out(func() any { return a.Add(temporal.DurationBag{Months: temporal.V(float64(-1))}, nil) })
		out(func() any { return a.Add(temporal.DurationBag{Months: temporal.V(float64(-12))}, nil) })
		out(func() any { return a.Add(temporal.DurationBag{Years: temporal.V(float64(1))}, nil) })
		out(func() any { return a.Add(temporal.DurationBag{Years: temporal.V(float64(19))}, nil) })
		out(func() any { return a.Add(temporal.DurationBag{Years: temporal.V(float64(-1))}, nil) })
	}, []string{"PlainDate(2104-06-02[u-ca=islamic-tbla])", "PlainDate(2104-10-28[u-ca=islamic-tbla])", "PlainDate(2105-04-23[u-ca=islamic-tbla])", "PlainDate(2105-05-22[u-ca=islamic-tbla])", "PlainDate(2104-04-04[u-ca=islamic-tbla])", "PlainDate(2103-05-16[u-ca=islamic-tbla])", "PlainDate(2105-04-23[u-ca=islamic-tbla])", "PlainDate(2122-10-10[u-ca=islamic-tbla])", "PlainDate(2103-05-16[u-ca=islamic-tbla])"}},
	{"cal_arithmetic_005", func() {
		a := temporal.PlainDateFrom("2004-05-17[u-ca=japanese]", nil)
		_ = a
		out(func() any {
			return a.Add(temporal.DurationBag{Years: temporal.V(float64(429))}, nil).Until(a, temporal.DifferenceOptions{LargestUnit: temporal.V("week")})
		})
	}, []string{"Duration(-P22384W1D)"}},
	{"cal_arithmetic_009", func() {
		a := temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("islamic-civil"), Year: temporal.V(float64(1290)), Month: temporal.V(float64(3)), Day: temporal.V(float64(6))}, nil)
		_ = a
		out(func() any { return a.Add(temporal.DurationBag{Months: temporal.V(float64(1))}, nil) })
		out(func() any { return a.Add(temporal.DurationBag{Months: temporal.V(float64(6))}, nil) })
		out(func() any { return a.Add(temporal.DurationBag{Months: temporal.V(float64(12))}, nil) })
		out(func() any { return a.Add(temporal.DurationBag{Months: temporal.V(float64(13))}, nil) })
		out(func() any { return a.Add(temporal.DurationBag{Months: temporal.V(float64(-1))}, nil) })
		out(func() any { return a.Add(temporal.DurationBag{Months: temporal.V(float64(-12))}, nil) })
		out(func() any { return a.Add(temporal.DurationBag{Years: temporal.V(float64(1))}, nil) })
		out(func() any { return a.Add(temporal.DurationBag{Years: temporal.V(float64(19))}, nil) })
		out(func() any { return a.Add(temporal.DurationBag{Years: temporal.V(float64(-1))}, nil) })
	}, []string{"PlainDate(1873-06-03[u-ca=islamic-civil])", "PlainDate(1873-10-28[u-ca=islamic-civil])", "PlainDate(1874-04-23[u-ca=islamic-civil])", "PlainDate(1874-05-23[u-ca=islamic-civil])", "PlainDate(1873-04-05[u-ca=islamic-civil])", "PlainDate(1872-05-14[u-ca=islamic-civil])", "PlainDate(1874-04-23[u-ca=islamic-civil])", "PlainDate(1891-10-10[u-ca=islamic-civil])", "PlainDate(1872-05-14[u-ca=islamic-civil])"}},
	{"cal_arithmetic_014", func() {
		p := temporal.PlainDateTimeFrom("2160-08-06T14:18:55.016522984[u-ca=indian]", nil)
		_ = p
		q := temporal.PlainDateTimeFrom("2164-08-01T11:06:57.610310177[u-ca=indian]", nil)
		_ = q
		out(func() any { return p.Until(q, temporal.DifferenceOptions{LargestUnit: temporal.V("month")}) })
		out(func() any { return p.Add(temporal.DurationBag{Weeks: temporal.V(float64(-48))}, nil) })
	}, []string{"Duration(P47M25DT20H48M2.593787193S)", "PlainDateTime(2159-09-05T14:18:55.016522984[u-ca=indian])"}},
	{"cal_arithmetic_015", func() {
		a := temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("islamic-tbla"), Year: temporal.V(float64(1493)), Month: temporal.V(float64(12)), Day: temporal.V(float64(31))}, nil)
		_ = a
		out(func() any {
			return a.Add(temporal.DurationBag{Months: temporal.V(float64(1))}, temporal.OverflowOptions{Overflow: temporal.V("constrain")})
		})
		out(func() any {
			return a.Add(temporal.DurationBag{Months: temporal.V(float64(1))}, temporal.OverflowOptions{Overflow: temporal.V("reject")})
		})
		out(func() any { return a.Day() })
	}, []string{"PlainDate(2071-02-28[u-ca=islamic-tbla])", "PlainDate(2071-02-28[u-ca=islamic-tbla])", "29"}},
}

var cal_chinese_dangiCases = []goldenCase{
	{"cal_chinese_dangi_002", func() {
		out(func() any {
			return temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("chinese"), Day: temporal.V(float64(24)), MonthCode: temporal.V("M02"), Year: temporal.V(float64(2055))}, temporal.OverflowOptions{Overflow: temporal.V("constrain")})
		})
		out(func() any {
			return temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("chinese"), Day: temporal.V(float64(20)), MonthCode: temporal.V("M08"), Year: temporal.V(float64(2055))}, temporal.OverflowOptions{Overflow: temporal.V("reject")})
		})
	}, []string{"PlainDate(2055-03-21[u-ca=chinese])", "PlainDate(2055-10-10[u-ca=chinese])"}},
	{"cal_chinese_dangi_003", func() {
		d := temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("dangi"), Year: temporal.V(float64(2012)), Month: temporal.V(float64(10)), Day: temporal.V(float64(1))}, nil)
		_ = d
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-2))}, nil).Year() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-2))}, nil).MonthCode() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-2))}, nil).Day() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-1))}, nil).Year() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-1))}, nil).MonthCode() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-1))}, nil).Day() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(0))}, nil).Year() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(0))}, nil).MonthCode() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(0))}, nil).Day() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(1))}, nil).Year() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(1))}, nil).MonthCode() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(1))}, nil).Day() })
	}, []string{"2012", "\"M08\"", "28", "2012", "\"M08\"", "29", "2012", "\"M09\"", "1", "2012", "\"M09\"", "2"}},
	{"cal_chinese_dangi_005", func() {
		d := temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("chinese"), Year: temporal.V(float64(2141)), Month: temporal.V(float64(12)), Day: temporal.V(float64(1))}, nil)
		_ = d
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-2))}, nil).Year() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-2))}, nil).MonthCode() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-2))}, nil).Day() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-1))}, nil).Year() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-1))}, nil).MonthCode() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-1))}, nil).Day() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(0))}, nil).Year() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(0))}, nil).MonthCode() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(0))}, nil).Day() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(1))}, nil).Year() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(1))}, nil).MonthCode() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(1))}, nil).Day() })
	}, []string{"2141", "\"M11\"", "29", "2141", "\"M11\"", "30", "2141", "\"M12\"", "1", "2141", "\"M12\"", "2"}},
	{"cal_chinese_dangi_008", func() {
		out(func() any {
			return temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("dangi"), Day: temporal.V(float64(12)), Month: temporal.V(float64(13)), Year: temporal.V(float64(1917))}, temporal.OverflowOptions{Overflow: temporal.V("constrain")})
		})
		out(func() any {
			return temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("dangi"), Day: temporal.V(float64(7)), Month: temporal.V(float64(1)), Year: temporal.V(float64(1917))}, temporal.OverflowOptions{Overflow: temporal.V("reject")})
		})
	}, []string{"PlainDate(1918-01-24[u-ca=dangi])", "PlainDate(1917-01-29[u-ca=dangi])"}},
	{"cal_chinese_dangi_011", func() {
		d := temporal.PlainDateFrom("+036088-10-28[u-ca=dangi]", nil)
		_ = d
		out(func() any { return d.ToString(temporal.PlainDateToStringOptions{CalendarName: temporal.V("auto")}) })
		out(func() any { return d.ToString(temporal.PlainDateToStringOptions{CalendarName: temporal.V("always")}) })
		out(func() any { return d.ToString(temporal.PlainDateToStringOptions{CalendarName: temporal.V("never")}) })
		out(func() any { return d.ToString(temporal.PlainDateToStringOptions{CalendarName: temporal.V("critical")}) })
		out(func() any { return d.ToJSON() })
		out(func() any { return d.WithCalendar("chinese") })
	}, []string{"\"+036088-10-28[u-ca=dangi]\"", "\"+036088-10-28[u-ca=dangi]\"", "\"+036088-10-28\"", "\"+036088-10-28[!u-ca=dangi]\"", "\"+036088-10-28[u-ca=dangi]\"", "PlainDate(+036088-10-28[u-ca=chinese])"}},
	{"cal_chinese_dangi_012", func() {
		d := temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("dangi"), Year: temporal.V(float64(-8756)), Month: temporal.V(float64(11)), Day: temporal.V(float64(1))}, nil)
		_ = d
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-2))}, nil).Year() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-2))}, nil).MonthCode() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-2))}, nil).Day() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-1))}, nil).Year() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-1))}, nil).MonthCode() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-1))}, nil).Day() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(0))}, nil).Year() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(0))}, nil).MonthCode() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(0))}, nil).Day() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(1))}, nil).Year() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(1))}, nil).MonthCode() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(1))}, nil).Day() })
	}, []string{"-8756", "\"M10\"", "28", "-8756", "\"M10\"", "29", "-8756", "\"M11\"", "1", "-8756", "\"M11\"", "2"}},
}

var cal_coptic_ethiopicCases = []goldenCase{
	{"cal_coptic_ethiopic_000", func() {
		p := temporal.PlainDateTimeFrom("2166-03-09T20:14:33[u-ca=coptic]", nil)
		_ = p
		out(func() any { return p })
		out(func() any { return p.MonthCode() })
		out(func() any { return p.EraYear() })
		out(func() any { return p.ToPlainDate() })
		out(func() any {
			return p.ToString(temporal.PlainDateTimeToStringOptions{CalendarName: temporal.V("critical")})
		})
	}, []string{"PlainDateTime(2166-03-09T20:14:33[u-ca=coptic])", "\"M06\"", "1882", "PlainDate(2166-03-09[u-ca=coptic])", "\"2166-03-09T20:14:33[!u-ca=coptic]\""}},
	{"cal_coptic_ethiopic_001", func() {
		d := temporal.PlainDateFrom("2069-09-06[u-ca=coptic]", nil)
		_ = d
		out(func() any {
			return d.With(temporal.PlainDateBag{Day: temporal.V(float64(23))}, temporal.OverflowOptions{Overflow: temporal.V("reject")})
		})
		out(func() any { return d.With(temporal.PlainDateBag{MonthCode: temporal.V("M09")}, nil) })
		out(func() any {
			return d.With(temporal.PlainDateBag{EraYear: temporal.V(float64(112)), Era: temporal.V("am")}, nil)
		})
	}, []string{"throw RangeError", "PlainDate(2069-05-09[u-ca=coptic])", "PlainDate(0396-08-25[u-ca=coptic])"}},
	{"cal_coptic_ethiopic_004", func() {
		out(func() any {
			return temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("ethioaa"), Day: temporal.V(float64(1)), Month: temporal.V(float64(10)), Era: temporal.V("aa"), EraYear: temporal.V(float64(277))}, temporal.OverflowOptions{Overflow: temporal.V("constrain")})
		})
		out(func() any {
			return temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("ethioaa"), Day: temporal.V(float64(21)), MonthCode: temporal.V("M06"), Era: temporal.V("aa"), EraYear: temporal.V(float64(1))}, temporal.OverflowOptions{Overflow: temporal.V("reject")})
		})
	}, []string{"PlainDate(-005215-04-15[u-ca=ethioaa])", "PlainDate(-005491-01-03[u-ca=ethioaa])"}},
	{"cal_coptic_ethiopic_005", func() {
		p := temporal.PlainDateTimeFrom("2110-08-02T23:00:00.000000999[u-ca=ethiopic]", nil)
		_ = p
		out(func() any { return p })
		out(func() any { return p.MonthCode() })
		out(func() any { return p.EraYear() })
		out(func() any { return p.ToPlainDate() })
		out(func() any {
			return p.ToString(temporal.PlainDateTimeToStringOptions{CalendarName: temporal.V("never")})
		})
	}, []string{"PlainDateTime(2110-08-02T23:00:00.000000999[u-ca=ethiopic])", "\"M11\"", "2102", "PlainDate(2110-08-02[u-ca=ethiopic])", "\"2110-08-02T23:00:00.000000999\""}},
	{"cal_coptic_ethiopic_006", func() {
		out(func() any {
			return temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("coptic"), Year: temporal.V(float64(1729)), Month: temporal.V(float64(2)), Day: temporal.V(float64(1))}, nil)
		})
		out(func() any {
			return temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("coptic"), Year: temporal.V(float64(1729)), Month: temporal.V(float64(2)), Day: temporal.V(float64(1))}, nil).MonthCode()
		})
		out(func() any {
			return temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("coptic"), Year: temporal.V(float64(1729)), Month: temporal.V(float64(2)), Day: temporal.V(float64(1))}, nil).Day()
		})
	}, []string{"PlainDate(2012-10-11[u-ca=coptic])", "\"M02\"", "1"}},
	{"cal_coptic_ethiopic_007", func() {
		p := temporal.PlainDateTimeFrom("2156-09-10T00:59:59.000000999[u-ca=ethiopic]", nil)
		_ = p
		out(func() any { return p })
		out(func() any { return p.MonthCode() })
		out(func() any { return p.EraYear() })
		out(func() any { return p.ToPlainDate() })
		out(func() any {
			return p.ToString(temporal.PlainDateTimeToStringOptions{CalendarName: temporal.V("never")})
		})
	}, []string{"PlainDateTime(2156-09-10T00:59:59.000000999[u-ca=ethiopic])", "\"M13\"", "2148", "PlainDate(2156-09-10[u-ca=ethiopic])", "\"2156-09-10T00:59:59.000000999\""}},
}

var cal_hebrewCases = []goldenCase{
	{"cal_hebrew_000", func() {
		out(func() any {
			return temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("hebrew"), Day: temporal.V(float64(26)), Month: temporal.V(float64(13)), Year: temporal.V(float64(5358))}, temporal.OverflowOptions{Overflow: temporal.V("constrain")})
		})
		out(func() any {
			return temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("hebrew"), Day: temporal.V(float64(19)), Month: temporal.V(float64(6)), Era: temporal.V("am"), EraYear: temporal.V(float64(2))}, temporal.OverflowOptions{Overflow: temporal.V("reject")})
		})
	}, []string{"PlainDate(1598-09-27[u-ca=hebrew])", "PlainDate(-003758-02-11[u-ca=hebrew])"}},
	{"cal_hebrew_001", func() {
		d := temporal.PlainDateFrom("1970-03-24[u-ca=hebrew]", nil)
		_ = d
		out(func() any {
			return d.With(temporal.PlainDateBag{Day: temporal.V(float64(3))}, temporal.OverflowOptions{Overflow: temporal.V("constrain")})
		})
		out(func() any { return d.With(temporal.PlainDateBag{MonthCode: temporal.V("M08")}, nil) })
		out(func() any {
			return d.With(temporal.PlainDateBag{EraYear: temporal.V(float64(216)), Era: temporal.V("am")}, nil)
		})
	}, []string{"PlainDate(1970-03-11[u-ca=hebrew])", "PlainDate(1970-05-22[u-ca=hebrew])", "PlainDate(-003544-02-12[u-ca=hebrew])"}},
	{"cal_hebrew_002", func() {
		d := temporal.PlainDateFrom("1891-06-18[u-ca=hebrew]", nil)
		_ = d
		out(func() any {
			return d.With(temporal.PlainDateBag{Month: temporal.V(float64(20))}, temporal.OverflowOptions{Overflow: temporal.V("constrain")})
		})
		out(func() any { return d.With(temporal.PlainDateBag{MonthCode: temporal.V("M10")}, nil) })
		out(func() any {
			return d.With(temporal.PlainDateBag{EraYear: temporal.V(float64(382)), Era: temporal.V("am")}, nil)
		})
	}, []string{"PlainDate(1891-09-15[u-ca=hebrew])", "PlainDate(1891-07-18[u-ca=hebrew])", "PlainDate(-003378-05-04[u-ca=hebrew])"}},
	{"cal_hebrew_003", func() {
		d := temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("hebrew"), Year: temporal.V(float64(-74838)), Month: temporal.V(float64(5)), Day: temporal.V(float64(1))}, nil)
		_ = d
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-2))}, nil).Year() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-2))}, nil).MonthCode() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-2))}, nil).Day() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-1))}, nil).Year() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-1))}, nil).MonthCode() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-1))}, nil).Day() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(0))}, nil).Year() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(0))}, nil).MonthCode() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(0))}, nil).Day() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(1))}, nil).Year() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(1))}, nil).MonthCode() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(1))}, nil).Day() })
	}, []string{"-74838", "\"M04\"", "28", "-74838", "\"M04\"", "29", "-74838", "\"M05\"", "1", "-74838", "\"M05\"", "2"}},
	{"cal_hebrew_005", func() {
		out(func() any {
			return temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("hebrew"), Era: temporal.V("am"), EraYear: temporal.V(float64(39)), Month: temporal.V(float64(1)), Day: temporal.V(float64(1))}, nil)
		})
	}, []string{"PlainDate(-003722-09-07[u-ca=hebrew])"}},
	{"cal_hebrew_006", func() {
		out(func() any {
			return temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("hebrew"), Day: temporal.V(float64(18)), Month: temporal.V(float64(5)), Year: temporal.V(float64(4308))}, temporal.OverflowOptions{Overflow: temporal.V("constrain")})
		})
		out(func() any {
			return temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("hebrew"), Day: temporal.V(float64(31)), MonthCode: temporal.V("M11"), Year: temporal.V(float64(4308))}, temporal.OverflowOptions{Overflow: temporal.V("reject")})
		})
	}, []string{"PlainDate(0548-01-16[u-ca=hebrew])", "throw RangeError"}},
}

var cal_islamicCases = []goldenCase{
	{"cal_islamic_001", func() {
		d := temporal.PlainDateFrom("2063-01-15[u-ca=islamic-umalqura]", nil)
		_ = d
		out(func() any { return d.ToString(temporal.PlainDateToStringOptions{CalendarName: temporal.V("auto")}) })
		out(func() any { return d.ToString(temporal.PlainDateToStringOptions{CalendarName: temporal.V("always")}) })
		out(func() any { return d.ToString(temporal.PlainDateToStringOptions{CalendarName: temporal.V("never")}) })
		out(func() any { return d.ToString(temporal.PlainDateToStringOptions{CalendarName: temporal.V("critical")}) })
		out(func() any { return d.ToJSON() })
		out(func() any { return d.WithCalendar("islamic-tbla") })
	}, []string{"\"2063-01-15[u-ca=islamic-umalqura]\"", "\"2063-01-15[u-ca=islamic-umalqura]\"", "\"2063-01-15\"", "\"2063-01-15[!u-ca=islamic-umalqura]\"", "\"2063-01-15[u-ca=islamic-umalqura]\"", "PlainDate(2063-01-15[u-ca=islamic-tbla])"}},
	{"cal_islamic_002", func() {
		p := temporal.PlainDateTimeFrom("2153-09-25T04:50:37.001484432[u-ca=islamic-civil]", nil)
		_ = p
		out(func() any { return p })
		out(func() any { return p.MonthCode() })
		out(func() any { return p.EraYear() })
		out(func() any { return p.ToPlainDate() })
		out(func() any {
			return p.ToString(temporal.PlainDateTimeToStringOptions{CalendarName: temporal.V("never")})
		})
	}, []string{"PlainDateTime(2153-09-25T04:50:37.001484432[u-ca=islamic-civil])", "\"M03\"", "1579", "PlainDate(2153-09-25[u-ca=islamic-civil])", "\"2153-09-25T04:50:37.001484432\""}},
	{"cal_islamic_004", func() {
		d := temporal.PlainDateFrom("2038-08-15[u-ca=islamic-tbla]", nil)
		_ = d
		out(func() any {
			return d.With(temporal.PlainDateBag{Year: temporal.V(float64(9))}, temporal.OverflowOptions{Overflow: temporal.V("constrain")})
		})
		out(func() any { return d.With(temporal.PlainDateBag{MonthCode: temporal.V("M02")}, nil) })
		out(func() any {
			return d.With(temporal.PlainDateBag{EraYear: temporal.V(float64(145)), Era: temporal.V("ah")}, nil)
		})
	}, []string{"PlainDate(0630-10-30[u-ca=islamic-tbla])", "PlainDate(2038-03-21[u-ca=islamic-tbla])", "PlainDate(0762-10-12[u-ca=islamic-tbla])"}},
	{"cal_islamic_009", func() {
		out(func() any {
			return temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("islamic-civil"), Year: temporal.V(float64(1561)), Month: temporal.V(float64(9)), Day: temporal.V(float64(5))}, nil)
		})
		out(func() any {
			return temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("islamic-civil"), Year: temporal.V(float64(1561)), Month: temporal.V(float64(9)), Day: temporal.V(float64(5))}, nil).MonthCode()
		})
		out(func() any {
			return temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("islamic-civil"), Year: temporal.V(float64(1561)), Month: temporal.V(float64(9)), Day: temporal.V(float64(5))}, nil).Day()
		})
	}, []string{"PlainDate(2136-10-01[u-ca=islamic-civil])", "\"M09\"", "5"}},
	{"cal_islamic_010", func() {
		d := temporal.PlainDateFrom("2179-07-20[u-ca=islamic-umalqura]", nil)
		_ = d
		out(func() any { return d.ToString(temporal.PlainDateToStringOptions{CalendarName: temporal.V("auto")}) })
		out(func() any { return d.ToString(temporal.PlainDateToStringOptions{CalendarName: temporal.V("always")}) })
		out(func() any { return d.ToString(temporal.PlainDateToStringOptions{CalendarName: temporal.V("never")}) })
		out(func() any { return d.ToString(temporal.PlainDateToStringOptions{CalendarName: temporal.V("critical")}) })
		out(func() any { return d.ToJSON() })
		out(func() any { return d.WithCalendar("islamic-tbla") })
	}, []string{"\"2179-07-20[u-ca=islamic-umalqura]\"", "\"2179-07-20[u-ca=islamic-umalqura]\"", "\"2179-07-20\"", "\"2179-07-20[!u-ca=islamic-umalqura]\"", "\"2179-07-20[u-ca=islamic-umalqura]\"", "PlainDate(2179-07-20[u-ca=islamic-tbla])"}},
	{"cal_islamic_012", func() {
		d := temporal.PlainDateFrom("1825-06-11[u-ca=islamic-civil]", nil)
		_ = d
		out(func() any {
			return d.With(temporal.PlainDateBag{Month: temporal.V(float64(8))}, temporal.OverflowOptions{Overflow: temporal.V("reject")})
		})
		out(func() any { return d.With(temporal.PlainDateBag{MonthCode: temporal.V("M03")}, nil) })
		out(func() any {
			return d.With(temporal.PlainDateBag{EraYear: temporal.V(float64(222)), Era: temporal.V("bh")}, nil)
		})
	}, []string{"PlainDate(1825-04-13[u-ca=islamic-civil])", "PlainDate(1824-11-16[u-ca=islamic-civil])", "PlainDate(0407-12-13[u-ca=islamic-civil])"}},
}

var cal_japanese_roc_buddhistCases = []goldenCase{
	{"cal_japanese_roc_buddhist_002", func() {
		p := temporal.PlainDateTimeFrom("1977-07-15T14:35:01.650376704[u-ca=japanese]", nil)
		_ = p
		out(func() any { return p })
		out(func() any { return p.MonthCode() })
		out(func() any { return p.EraYear() })
		out(func() any { return p.ToPlainDate() })
		out(func() any { return p.ToString(temporal.PlainDateTimeToStringOptions{CalendarName: temporal.V("auto")}) })
	}, []string{"PlainDateTime(1977-07-15T14:35:01.650376704[u-ca=japanese])", "\"M07\"", "52", "PlainDate(1977-07-15[u-ca=japanese])", "\"1977-07-15T14:35:01.650376704[u-ca=japanese]\""}},
	{"cal_japanese_roc_buddhist_003", func() {
		p := temporal.PlainDateTimeFrom("1896-11-14T16:24:47.377835161[u-ca=buddhist]", nil)
		_ = p
		out(func() any { return p })
		out(func() any { return p.MonthCode() })
		out(func() any { return p.EraYear() })
		out(func() any { return p.ToPlainDate() })
		out(func() any {
			return p.ToString(temporal.PlainDateTimeToStringOptions{CalendarName: temporal.V("never")})
		})
	}, []string{"PlainDateTime(1896-11-14T16:24:47.377835161[u-ca=buddhist])", "\"M11\"", "2439", "PlainDate(1896-11-14[u-ca=buddhist])", "\"1896-11-14T16:24:47.377835161\""}},
	{"cal_japanese_roc_buddhist_005", func() {
		out(func() any {
			return temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("roc"), Era: temporal.V("broc"), EraYear: temporal.V(float64(66)), Month: temporal.V(float64(1)), Day: temporal.V(float64(1))}, nil)
		})
		out(func() any {
			return temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("roc"), Era: temporal.V("minguo"), EraYear: temporal.V(float64(63)), Month: temporal.V(float64(1)), Day: temporal.V(float64(1))}, nil)
		})
		out(func() any {
			return temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("roc"), Era: temporal.V("minguo-qian"), EraYear: temporal.V(float64(3)), Month: temporal.V(float64(1)), Day: temporal.V(float64(1))}, nil)
		})
	}, []string{"PlainDate(1846-01-01[u-ca=roc])", "PlainDate(1974-01-01[u-ca=roc])", "PlainDate(1909-01-01[u-ca=roc])"}},
	{"cal_japanese_roc_buddhist_007", func() {
		d := temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("buddhist"), Year: temporal.V(float64(271)), Month: temporal.V(float64(6)), Day: temporal.V(float64(1))}, nil)
		_ = d
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-2))}, nil).Year() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-2))}, nil).MonthCode() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-2))}, nil).Day() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-1))}, nil).Year() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-1))}, nil).MonthCode() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-1))}, nil).Day() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(0))}, nil).Year() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(0))}, nil).MonthCode() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(0))}, nil).Day() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(1))}, nil).Year() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(1))}, nil).MonthCode() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(1))}, nil).Day() })
	}, []string{"271", "\"M05\"", "30", "271", "\"M05\"", "31", "271", "\"M06\"", "1", "271", "\"M06\"", "2"}},
	{"cal_japanese_roc_buddhist_009", func() {
		d := temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("gregory"), Year: temporal.V(float64(1809)), Month: temporal.V(float64(7)), Day: temporal.V(float64(1))}, nil)
		_ = d
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-2))}, nil).Year() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-2))}, nil).MonthCode() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-2))}, nil).Day() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-1))}, nil).Year() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-1))}, nil).MonthCode() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-1))}, nil).Day() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(0))}, nil).Year() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(0))}, nil).MonthCode() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(0))}, nil).Day() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(1))}, nil).Year() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(1))}, nil).MonthCode() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(1))}, nil).Day() })
	}, []string{"1809", "\"M06\"", "29", "1809", "\"M06\"", "30", "1809", "\"M07\"", "1", "1809", "\"M07\"", "2"}},
	{"cal_japanese_roc_buddhist_010", func() {
		out(func() any {
			return temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("gregory"), Day: temporal.V(float64(11)), Month: temporal.V(float64(1)), Era: temporal.V("ad"), EraYear: temporal.V(float64(2))}, temporal.OverflowOptions{Overflow: temporal.V("constrain")})
		})
		out(func() any {
			return temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("gregory"), Day: temporal.V(float64(9)), MonthCode: temporal.V("M05"), Year: temporal.V(float64(1953))}, temporal.OverflowOptions{Overflow: temporal.V("reject")})
		})
	}, []string{"PlainDate(0002-01-11[u-ca=gregory])", "PlainDate(1953-05-09[u-ca=gregory])"}},
}

var cal_persian_indianCases = []goldenCase{
	{"cal_persian_indian_002", func() {
		d := temporal.PlainDateFrom("1956-11-23[u-ca=persian]", nil)
		_ = d
		out(func() any {
			return d.With(temporal.PlainDateBag{Day: temporal.V(float64(11))}, temporal.OverflowOptions{Overflow: temporal.V("reject")})
		})
		out(func() any { return d.With(temporal.PlainDateBag{MonthCode: temporal.V("M11")}, nil) })
		out(func() any {
			return d.With(temporal.PlainDateBag{EraYear: temporal.V(float64(32)), Era: temporal.V("ap")}, nil)
		})
	}, []string{"PlainDate(1956-12-02[u-ca=persian])", "PlainDate(1957-01-22[u-ca=persian])", "PlainDate(0653-11-23[u-ca=persian])"}},
	{"cal_persian_indian_003", func() {
		out(func() any {
			return temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("indian"), Day: temporal.V(float64(15)), Month: temporal.V(float64(1)), Era: temporal.V("shaka"), EraYear: temporal.V(float64(1))}, temporal.OverflowOptions{Overflow: temporal.V("constrain")})
		})
		out(func() any {
			return temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("indian"), Day: temporal.V(float64(8)), MonthCode: temporal.V("M06"), Year: temporal.V(float64(2085))}, temporal.OverflowOptions{Overflow: temporal.V("reject")})
		})
	}, []string{"PlainDate(0079-04-05[u-ca=indian])", "PlainDate(2163-08-30[u-ca=indian])"}},
	{"cal_persian_indian_014", func() {
		out(func() any {
			return temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("persian"), Year: temporal.V(float64(1213)), Month: temporal.V(float64(7)), Day: temporal.V(float64(17))}, nil)
		})
		out(func() any {
			return temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("persian"), Year: temporal.V(float64(1213)), Month: temporal.V(float64(7)), Day: temporal.V(float64(17))}, nil).MonthCode()
		})
		out(func() any {
			return temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("persian"), Year: temporal.V(float64(1213)), Month: temporal.V(float64(7)), Day: temporal.V(float64(17))}, nil).Day()
		})
	}, []string{"PlainDate(1834-10-09[u-ca=persian])", "\"M07\"", "17"}},
	{"cal_persian_indian_016", func() {
		d := temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("persian"), Year: temporal.V(float64(912)), Month: temporal.V(float64(10)), Day: temporal.V(float64(1))}, nil)
		_ = d
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-2))}, nil).Year() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-2))}, nil).MonthCode() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-2))}, nil).Day() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-1))}, nil).Year() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-1))}, nil).MonthCode() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(-1))}, nil).Day() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(0))}, nil).Year() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(0))}, nil).MonthCode() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(0))}, nil).Day() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(1))}, nil).Year() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(1))}, nil).MonthCode() })
		out(func() any { return d.Add(temporal.DurationBag{Days: temporal.V(float64(1))}, nil).Day() })
	}, []string{"912", "\"M09\"", "29", "912", "\"M09\"", "30", "912", "\"M10\"", "1", "912", "\"M10\"", "2"}},
	{"cal_persian_indian_017", func() {
		d := temporal.PlainDateFrom("1911-04-02[u-ca=persian]", nil)
		_ = d
		out(func() any {
			return d.With(temporal.PlainDateBag{Day: temporal.V(float64(1))}, temporal.OverflowOptions{Overflow: temporal.V("reject")})
		})
		out(func() any { return d.With(temporal.PlainDateBag{MonthCode: temporal.V("M12")}, nil) })
		out(func() any {
			return d.With(temporal.PlainDateBag{EraYear: temporal.V(float64(314)), Era: temporal.V("ap")}, nil)
		})
	}, []string{"PlainDate(1911-03-22[u-ca=persian])", "PlainDate(1912-03-03[u-ca=persian])", "PlainDate(0935-04-01[u-ca=persian])"}},
	{"cal_persian_indian_018", func() {
		d := temporal.PlainDateFrom("2170-01-29[u-ca=persian]", nil)
		_ = d
		out(func() any { return d.ToString(temporal.PlainDateToStringOptions{CalendarName: temporal.V("auto")}) })
		out(func() any { return d.ToString(temporal.PlainDateToStringOptions{CalendarName: temporal.V("always")}) })
		out(func() any { return d.ToString(temporal.PlainDateToStringOptions{CalendarName: temporal.V("never")}) })
		out(func() any { return d.ToString(temporal.PlainDateToStringOptions{CalendarName: temporal.V("critical")}) })
		out(func() any { return d.ToJSON() })
		out(func() any { return d.WithCalendar("ethioaa") })
	}, []string{"\"2170-01-29[u-ca=persian]\"", "\"2170-01-29[u-ca=persian]\"", "\"2170-01-29\"", "\"2170-01-29[!u-ca=persian]\"", "\"2170-01-29[u-ca=persian]\"", "PlainDate(2170-01-29[u-ca=ethioaa])"}},
}

var cal_year_month_dayCases = []goldenCase{
	{"cal_year_month_day_000", func() {
		a := temporal.PlainYearMonthFrom(temporal.PlainYearMonthBag{Calendar: temporal.V("chinese"), Year: temporal.V(float64(10703)), Month: temporal.V(float64(3))}, nil)
		_ = a
		out(func() any {
			return a.With(temporal.PlainYearMonthBag{Month: temporal.V(float64(14))}, temporal.OverflowOptions{Overflow: temporal.V("constrain")})
		})
		out(func() any { return a.With(temporal.PlainYearMonthBag{Year: temporal.V(float64(10704))}, nil) })
	}, []string{"PlainYearMonth(+010703-12-24[u-ca=chinese])", "PlainYearMonth(+010704-03-22[u-ca=chinese])"}},
	{"cal_year_month_day_005", func() {
		d := temporal.PlainDateFrom("1909-07-25[u-ca=ethiopic]", nil)
		_ = d
		out(func() any { return temporal.PlainYearMonthFrom(d.ToString(nil), nil) })
		out(func() any { return temporal.PlainMonthDayFrom(d.ToString(nil), nil) })
	}, []string{"PlainYearMonth(1909-07-08[u-ca=ethiopic])", "PlainMonthDay(1972-07-25[u-ca=ethiopic])"}},
	{"cal_year_month_day_008", func() {
		a := temporal.PlainYearMonthFrom(temporal.PlainYearMonthBag{Calendar: temporal.V("roc"), Year: temporal.V(float64(285)), Month: temporal.V(float64(6))}, nil)
		_ = a
		b := temporal.PlainYearMonthFrom(temporal.PlainYearMonthBag{Calendar: temporal.V("roc"), Year: temporal.V(float64(289)), Month: temporal.V(float64(6))}, nil)
		_ = b
		out(func() any { return a.Until(b, temporal.DifferenceOptions{LargestUnit: temporal.V("year")}) })
		out(func() any { return a.Until(b, temporal.DifferenceOptions{LargestUnit: temporal.V("month")}) })
		out(func() any { return temporal.PlainYearMonthCompare(a, b) })
	}, []string{"Duration(P4Y)", "Duration(P48M)", "-1"}},
	{"cal_year_month_day_009", func() {
		a := temporal.PlainYearMonthFrom(temporal.PlainYearMonthBag{Calendar: temporal.V("persian"), Year: temporal.V(float64(1180)), Month: temporal.V(float64(6))}, nil)
		_ = a
		b := temporal.PlainYearMonthFrom(temporal.PlainYearMonthBag{Calendar: temporal.V("persian"), Year: temporal.V(float64(1217)), Month: temporal.V(float64(2))}, nil)
		_ = b
		out(func() any { return a.Until(b, temporal.DifferenceOptions{LargestUnit: temporal.V("year")}) })
		out(func() any { return a.Until(b, temporal.DifferenceOptions{LargestUnit: temporal.V("month")}) })
		out(func() any { return temporal.PlainYearMonthCompare(a, b) })
	}, []string{"Duration(P36Y8M)", "Duration(P440M)", "-1"}},
	{"cal_year_month_day_011", func() {
		a := temporal.PlainYearMonthFrom(temporal.PlainYearMonthBag{Calendar: temporal.V("ethioaa"), Year: temporal.V(float64(7362)), Month: temporal.V(float64(6))}, nil)
		_ = a
		b := temporal.PlainYearMonthFrom(temporal.PlainYearMonthBag{Calendar: temporal.V("ethioaa"), Year: temporal.V(float64(7395)), Month: temporal.V(float64(2))}, nil)
		_ = b
		out(func() any { return a.Until(b, temporal.DifferenceOptions{LargestUnit: temporal.V("year")}) })
		out(func() any { return a.Until(b, temporal.DifferenceOptions{LargestUnit: temporal.V("month")}) })
		out(func() any { return temporal.PlainYearMonthCompare(a, b) })
	}, []string{"Duration(P32Y9M)", "Duration(P425M)", "-1"}},
	{"cal_year_month_day_012", func() {
		out(func() any {
			return temporal.PlainMonthDayFrom(temporal.PlainMonthDayBag{Calendar: temporal.V("persian"), Year: temporal.V(float64(1236)), Month: temporal.V(float64(3)), Day: temporal.V(float64(10))}, temporal.OverflowOptions{Overflow: temporal.V("constrain")})
		})
		out(func() any {
			return temporal.PlainMonthDayFrom(temporal.PlainMonthDayBag{Calendar: temporal.V("persian"), Year: temporal.V(float64(1236)), Month: temporal.V(float64(12)), Day: temporal.V(float64(11))}, temporal.OverflowOptions{Overflow: temporal.V("reject")})
		})
	}, []string{"PlainMonthDay(1972-05-31[u-ca=persian])", "PlainMonthDay(1972-03-01[u-ca=persian])"}},
}

var duration_relativeCases = []goldenCase{
	{"duration_relative_005", func() {
		out(func() any { return temporal.DurationFrom(temporal.DurationBag{Nanoseconds: temporal.V(" 7")}) })
	}, []string{"Duration(PT0.000000007S)"}},
	{"duration_relative_006", func() {
		d := temporal.DurationFrom(temporal.DurationBag{Weeks: temporal.V(float64(345)), Hours: temporal.V(float64(4294967295)), Minutes: temporal.V(float64(22)), Milliseconds: temporal.V(float64(11)), Nanoseconds: temporal.V(float64(17))})
		_ = d
		out(func() any {
			return d.ToString(temporal.DurationToStringOptions{FractionalSecondDigits: temporal.V(float64(0))})
		})
		out(func() any {
			return d.ToString(temporal.DurationToStringOptions{FractionalSecondDigits: temporal.V(float64(3))})
		})
		out(func() any {
			return d.ToString(temporal.DurationToStringOptions{FractionalSecondDigits: temporal.V(float64(9))})
		})
		out(func() any {
			return d.ToString(temporal.DurationToStringOptions{FractionalSecondDigits: temporal.V("auto")})
		})
		out(func() any {
			return d.ToString(temporal.DurationToStringOptions{SmallestUnit: temporal.V("minute"), RoundingMode: temporal.V("trunc")})
		})
	}, []string{"\"P345W178956970DT15H22M0S\"", "\"P345W178956970DT15H22M0.011S\"", "\"P345WT4294967295H22M0.011000017S\"", "\"P345WT4294967295H22M0.011000017S\"", "throw RangeError"}},
	{"duration_relative_007", func() {
		d := temporal.DurationFrom(temporal.DurationBag{Hours: temporal.V(float64(-746123131)), Seconds: temporal.V(float64(-4537)), Milliseconds: temporal.V(float64(-927050353)), Microseconds: temporal.V(float64(-29))})
		_ = d
		out(func() any {
			return d.ToString(temporal.DurationToStringOptions{FractionalSecondDigits: temporal.V(float64(0))})
		})
		out(func() any {
			return d.ToString(temporal.DurationToStringOptions{FractionalSecondDigits: temporal.V(float64(3))})
		})
		out(func() any {
			return d.ToString(temporal.DurationToStringOptions{FractionalSecondDigits: temporal.V(float64(9))})
		})
		out(func() any {
			return d.ToString(temporal.DurationToStringOptions{FractionalSecondDigits: temporal.V("auto")})
		})
		out(func() any {
			return d.ToString(temporal.DurationToStringOptions{SmallestUnit: temporal.V("millisecond"), RoundingMode: temporal.V("expand")})
		})
	}, []string{"\"-PT746123389H46M27S\"", "\"-PT746123389H46M27.353S\"", "\"-PT746123131H931587.353029000S\"", "\"-PT746123131H931587.353029S\"", "\"-PT746123389H46M27.354S\""}},
	{"duration_relative_008", func() {
		out(func() any { return temporal.DurationFrom(temporal.DurationBag{Hours: temporal.V(true)}) })
	}, []string{"Duration(PT1H)"}},
	{"duration_relative_009", func() {
		d := temporal.DurationFrom(temporal.DurationBag{Weeks: temporal.V(float64(-3)), Minutes: temporal.V(math.Copysign(0, -1)), Seconds: temporal.V(float64(-1.7976931348623157e+308)), Microseconds: temporal.V(float64(-36))})
		_ = d
		out(func() any { return d.Total("millisecond") })
		out(func() any { return d.Total("minute") })
		out(func() any { return d.Total("nanosecond") })
	}, []string{"throw RangeError"}},
	{"duration_relative_010", func() {
		d := temporal.DurationFrom(temporal.DurationBag{Minutes: temporal.V(float64(-36)), Seconds: temporal.V(float64(-9007199254740991)), Microseconds: temporal.V(float64(-124)), Nanoseconds: temporal.V(float64(-1e+21))})
		_ = d
		out(func() any {
			return d.Round(temporal.DurationRoundOptions{SmallestUnit: temporal.V("millisecond"), LargestUnit: temporal.V("day"), RoundingMode: temporal.V("halfEven")})
		})
		out(func() any {
			return d.Round(temporal.DurationRoundOptions{SmallestUnit: temporal.V("nanosecond"), LargestUnit: temporal.V("day")})
		})
		out(func() any {
			return d.Round(temporal.DurationRoundOptions{SmallestUnit: temporal.V("day"), LargestUnit: temporal.V("auto")})
		})
	}, []string{"throw RangeError"}},
}

var errorsCases = []goldenCase{
	{"errors_000", func() {
		out(func() any { return temporal.PlainYearMonthFrom("2024-03", nil).ToLocaleString("en", nil) })
		out(func() any {
			return temporal.NewDateTimeFormat("en", temporal.DateTimeFormatOptions{Month: temporal.V(true)})
		})
	}, []string{"throw RangeError: \"cannot format PlainYearMonth with calendar iso8601 in locale with calendar gregory\"", "throw RangeError: \"Value true out of range for Intl.DateTimeFormat options property month\""}},
	{"errors_001", func() {
		out(func() any { return temporal.PlainDateCompare(float64(5), "2024-01-01") })
	}, []string{"throw TypeError: \"expected a string, not 5\""}},
	{"errors_002", func() {
		out(func() any {
			return temporal.NewDateTimeFormat("en", temporal.DateTimeFormatOptions{TimeStyle: temporal.V("short"), TimeZone: temporal.V("UTC")}).Format(temporal.PlainDateFrom("2024-03-15", nil))
		})
		out(func() any {
			return temporal.NewDateTimeFormat("en", temporal.DateTimeFormatOptions{TimeZone: temporal.V("UTC")}).Format(temporal.ZonedDateTimeFrom("2024-03-15T12:30+00:00[UTC]", nil))
		})
		out(func() any {
			return temporal.NewDateTimeFormat("en", temporal.DateTimeFormatOptions{TimeZone: temporal.V("UTC")}).FormatRangeToParts(temporal.PlainDateFrom("2024-03-15", nil), float64(0))
		})
	}, []string{"throw TypeError: \"cannot format PlainDate with only time options\"", "throw TypeError: \"Temporal.ZonedDateTime not supported in DateTimeFormat methods. Use toLocaleString() instead.\"", "throw TypeError: \"Intl.DateTimeFormat.formatRangeToParts accepts two values of the same type\""}},
	{"errors_003", func() {
		out(func() any {
			return temporal.PlainDateFrom("2024-03-15[u-ca=hebrew]", nil).With(temporal.PlainDateBag{Era: temporal.V("am")}, nil)
		})
	}, []string{"throw TypeError: \"year (or era and eraYear) are required\""}},
	{"errors_005", func() {
		out(func() any { return temporal.DurationFrom("P1Y").Add("P1M") })
		out(func() any {
			return temporal.PlainDateFrom(temporal.PlainDateBag{Calendar: temporal.V("chinese"), Year: temporal.V(float64(2024)), MonthCode: temporal.V("M02L"), Day: temporal.V(float64(1))}, temporal.OverflowOptions{Overflow: temporal.V("reject")})
		})
	}, []string{"throw RangeError: \"For years, months, or weeks arithmetic, use date arithmetic relative to a starting point\"", "throw RangeError: \"Unmatched month M02L in chinese year 2024\""}},
	{"errors_006", func() {
		out(func() any { return temporal.NewDateTimeFormat("EN-US-", nil) })
	}, []string{"throw RangeError: \"Invalid language tag: en-us-\""}},
}

var format_partsCases = []goldenCase{
	{"format_parts_003", func() {
		out(func() any {
			return temporal.NewDateTimeFormat("en-US-u-hc-h11", temporal.DateTimeFormatOptions{TimeZoneName: temporal.V("shortGeneric"), Hour: temporal.V("numeric"), Minute: temporal.V("numeric"), TimeZone: temporal.V("Europe/Paris")}).FormatToParts(float64(3316884984190))
		})
		out(func() any {
			return temporal.NewDateTimeFormat("en-US-u-hc-h11", temporal.DateTimeFormatOptions{TimeZoneName: temporal.V("shortGeneric"), Hour: temporal.V("numeric"), Minute: temporal.V("numeric"), TimeZone: temporal.V("Europe/Paris")}).ResolvedOptions()
		})
	}, []string{"[{\"hour\":\"9\"},{\"literal\":\":\"},{\"minute\":\"56\"},{\"literal\":\" \"},{\"dayPeriod\":\"PM\"},{\"literal\":\" \"},{\"timeZoneName\":\"France Time\"}]", "{locale=\"en-US-u-hc-h11\",calendar=\"gregory\",numberingSystem=\"latn\",timeZone=\"Europe/Paris\",hourCycle=\"h11\",hour12=true,hour=\"numeric\",minute=\"2-digit\",timeZoneName=\"shortGeneric\"}"}},
	{"format_parts_004", func() {
		out(func() any {
			return temporal.NewDateTimeFormat("it", temporal.DateTimeFormatOptions{TimeZoneName: temporal.V("shortOffset"), Hour: temporal.V("numeric"), Minute: temporal.V("numeric"), TimeZone: temporal.V("Asia/Shanghai")}).FormatToParts(float64(-1410032377023))
		})
		out(func() any {
			return temporal.NewDateTimeFormat("it", temporal.DateTimeFormatOptions{TimeZoneName: temporal.V("shortOffset"), Hour: temporal.V("numeric"), Minute: temporal.V("numeric"), TimeZone: temporal.V("Asia/Shanghai")}).ResolvedOptions()
		})
	}, []string{"[{\"hour\":\"12\"},{\"literal\":\":\"},{\"minute\":\"20\"},{\"literal\":\" \"},{\"timeZoneName\":\"GMT+8\"}]", "{locale=\"it\",calendar=\"gregory\",numberingSystem=\"latn\",timeZone=\"Asia/Shanghai\",hourCycle=\"h23\",hour12=false,hour=\"2-digit\",minute=\"2-digit\",timeZoneName=\"shortOffset\"}"}},
	{"format_parts_005", func() {
		out(func() any {
			return temporal.NewDateTimeFormat("de-u-hc-h12", temporal.DateTimeFormatOptions{Year: temporal.V("numeric"), Month: temporal.V("numeric"), Day: temporal.V("numeric"), Hour: temporal.V("numeric"), Minute: temporal.V("numeric"), Second: temporal.V("numeric"), Hour12: temporal.V(true), TimeZone: temporal.V("-03:30")}).ResolvedOptions()
		})
		out(func() any { return temporal.NewDateTimeFormat("de-u-hc-h12", nil).ResolvedOptions() })
		out(func() any {
			return temporal.NewDateTimeFormat("de-u-hc-h12", temporal.DateTimeFormatOptions{Year: temporal.V("numeric"), Month: temporal.V("long"), Day: temporal.V("numeric"), Hour: temporal.V("numeric"), Minute: temporal.V("2-digit"), Second: temporal.V("2-digit"), TimeZone: temporal.V("+13:00")}).ResolvedOptions()
		})
	}, []string{"{locale=\"de-u-hc-h12\",calendar=\"gregory\",numberingSystem=\"latn\",timeZone=\"-03:30\",hourCycle=\"h12\",hour12=true,year=\"numeric\",month=\"numeric\",day=\"numeric\",hour=\"numeric\",minute=\"2-digit\",second=\"2-digit\"}", "{locale=\"de-u-hc-h12\",calendar=\"gregory\",numberingSystem=\"latn\",timeZone=\"UTC\",year=\"numeric\",month=\"numeric\",day=\"numeric\"}", "{locale=\"de-u-hc-h12\",calendar=\"gregory\",numberingSystem=\"latn\",timeZone=\"+13:00\",hourCycle=\"h12\",hour12=true,year=\"numeric\",month=\"long\",day=\"numeric\",hour=\"numeric\",minute=\"2-digit\",second=\"2-digit\"}"}},
	{"format_parts_007", func() {
		out(func() any {
			return temporal.NewDateTimeFormat("ar", temporal.DateTimeFormatOptions{TimeZoneName: temporal.V("shortOffset"), Hour: temporal.V("numeric"), Minute: temporal.V("numeric"), TimeZone: temporal.V("Asia/Tokyo")}).FormatToParts(float64(-791105366509))
		})
		out(func() any {
			return temporal.NewDateTimeFormat("ar", temporal.DateTimeFormatOptions{TimeZoneName: temporal.V("shortOffset"), Hour: temporal.V("numeric"), Minute: temporal.V("numeric"), TimeZone: temporal.V("Asia/Tokyo")}).ResolvedOptions()
		})
	}, []string{"[{\"hour\":\"1\"},{\"literal\":\":\"},{\"minute\":\"30\"},{\"literal\":\" \"},{\"dayPeriod\":\"ص\"},{\"literal\":\" \"},{\"timeZoneName\":\"غرينتش+9\"}]", "{locale=\"ar\",calendar=\"gregory\",numberingSystem=\"latn\",timeZone=\"Asia/Tokyo\",hourCycle=\"h12\",hour12=true,hour=\"numeric\",minute=\"2-digit\",timeZoneName=\"shortOffset\"}"}},
	{"format_parts_009", func() {
		out(func() any {
			return temporal.NewDateTimeFormat("fa-IR-u-nu-latn", temporal.DateTimeFormatOptions{TimeZoneName: temporal.V("longGeneric"), Hour: temporal.V("numeric"), Minute: temporal.V("numeric"), TimeZone: temporal.V("Australia/Sydney")}).FormatToParts(float64(2167561350293))
		})
		out(func() any {
			return temporal.NewDateTimeFormat("fa-IR-u-nu-latn", temporal.DateTimeFormatOptions{TimeZoneName: temporal.V("longGeneric"), Hour: temporal.V("numeric"), Minute: temporal.V("numeric"), TimeZone: temporal.V("Australia/Sydney")}).ResolvedOptions()
		})
	}, []string{"[{\"hour\":\"22\"},{\"literal\":\":\"},{\"minute\":\"22\"},{\"literal\":\" (\"},{\"timeZoneName\":\"وقت استرالیای شرقی\"},{\"literal\":\")\"}]", "{locale=\"fa-IR-u-nu-latn\",calendar=\"persian\",numberingSystem=\"latn\",timeZone=\"Australia/Sydney\",hourCycle=\"h23\",hour12=false,hour=\"numeric\",minute=\"2-digit\",timeZoneName=\"longGeneric\"}"}},
	{"format_parts_010", func() {
		out(func() any {
			return temporal.NewDateTimeFormat("th-TH", temporal.DateTimeFormatOptions{TimeZoneName: temporal.V("shortOffset"), Hour: temporal.V("numeric"), Minute: temporal.V("numeric"), TimeZone: temporal.V("Pacific/Auckland")}).FormatToParts(float64(4013687280167))
		})
		out(func() any {
			return temporal.NewDateTimeFormat("th-TH", temporal.DateTimeFormatOptions{TimeZoneName: temporal.V("shortOffset"), Hour: temporal.V("numeric"), Minute: temporal.V("numeric"), TimeZone: temporal.V("Pacific/Auckland")}).ResolvedOptions()
		})
	}, []string{"[{\"hour\":\"06\"},{\"literal\":\":\"},{\"minute\":\"08\"},{\"literal\":\" \"},{\"timeZoneName\":\"GMT+13\"}]", "{locale=\"th-TH\",calendar=\"buddhist\",numberingSystem=\"latn\",timeZone=\"Pacific/Auckland\",hourCycle=\"h23\",hour12=false,hour=\"2-digit\",minute=\"2-digit\",timeZoneName=\"shortOffset\"}"}},
}

var format_rangeCases = []goldenCase{
	{"format_range_000", func() {
		out(func() any {
			return temporal.NewDateTimeFormat("sv", temporal.DateTimeFormatOptions{Month: temporal.V("narrow"), TimeZone: temporal.V("+05:45")}).FormatRange(temporal.PlainDateFrom("1987-04-01", nil).WithCalendar("gregory").ToPlainYearMonth(), temporal.PlainDateFrom("1987-04-07", nil).WithCalendar("gregory").ToPlainYearMonth())
		})
		out(func() any {
			return temporal.NewDateTimeFormat("sv", temporal.DateTimeFormatOptions{Month: temporal.V("narrow"), TimeZone: temporal.V("+05:45")}).FormatRangeToParts(temporal.PlainDateFrom("1987-04-01", nil).WithCalendar("gregory").ToPlainYearMonth(), temporal.PlainDateFrom("1987-04-07", nil).WithCalendar("gregory").ToPlainYearMonth())
		})
	}, []string{"\"A\"", "[{\"month\":\"A\",\"shared\"}]"}},
	{"format_range_001", func() {
		out(func() any {
			return temporal.NewDateTimeFormat("de-u-hc-h12", temporal.DateTimeFormatOptions{Month: temporal.V("short"), Day: temporal.V("numeric"), TimeZone: temporal.V("-05:00")}).FormatRange(temporal.PlainDateFrom("1902-09-11", nil).WithCalendar("gregory").ToPlainMonthDay(), temporal.PlainDateFrom("1902-09-11", nil).WithCalendar("gregory").ToPlainMonthDay())
		})
		out(func() any {
			return temporal.NewDateTimeFormat("de-u-hc-h12", temporal.DateTimeFormatOptions{Month: temporal.V("short"), Day: temporal.V("numeric"), TimeZone: temporal.V("-05:00")}).FormatRangeToParts(temporal.PlainDateFrom("1902-09-11", nil).WithCalendar("gregory").ToPlainMonthDay(), temporal.PlainDateFrom("1902-09-11", nil).WithCalendar("gregory").ToPlainMonthDay())
		})
		out(func() any {
			return temporal.NewDateTimeFormat("de-u-hc-h12", temporal.DateTimeFormatOptions{Month: temporal.V("short"), TimeZone: temporal.V("+05:45")}).FormatRange(temporal.PlainDateFrom("1985-11-18", nil).WithCalendar("gregory").ToPlainYearMonth(), temporal.PlainDateFrom("1985-11-18", nil).WithCalendar("gregory").ToPlainYearMonth())
		})
		out(func() any {
			return temporal.NewDateTimeFormat("de-u-hc-h12", temporal.DateTimeFormatOptions{Month: temporal.V("short"), TimeZone: temporal.V("+05:45")}).FormatRangeToParts(temporal.PlainDateFrom("1985-11-18", nil).WithCalendar("gregory").ToPlainYearMonth(), temporal.PlainDateFrom("1985-11-18", nil).WithCalendar("gregory").ToPlainYearMonth())
		})
		out(func() any {
			return temporal.NewDateTimeFormat("de-u-hc-h12", temporal.DateTimeFormatOptions{Month: temporal.V("narrow"), TimeZone: temporal.V("+05:30")}).FormatRange(temporal.PlainDateFrom("2051-04-01", nil).WithCalendar("gregory").ToPlainYearMonth(), temporal.PlainDateFrom("2051-05-01", nil).WithCalendar("gregory").ToPlainYearMonth())
		})
		out(func() any {
			return temporal.NewDateTimeFormat("de-u-hc-h12", temporal.DateTimeFormatOptions{Month: temporal.V("narrow"), TimeZone: temporal.V("+05:30")}).FormatRangeToParts(temporal.PlainDateFrom("2051-04-01", nil).WithCalendar("gregory").ToPlainYearMonth(), temporal.PlainDateFrom("2051-05-01", nil).WithCalendar("gregory").ToPlainYearMonth())
		})
		out(func() any {
			return temporal.NewDateTimeFormat("de-u-hc-h12", temporal.DateTimeFormatOptions{Month: temporal.V("2-digit"), Day: temporal.V("2-digit"), TimeZone: temporal.V("-05:00")}).FormatRange(temporal.PlainDateFrom("1947-06-14", nil), temporal.PlainDateFrom("1947-06-14", nil))
		})
		out(func() any {
			return temporal.NewDateTimeFormat("de-u-hc-h12", temporal.DateTimeFormatOptions{Month: temporal.V("2-digit"), Day: temporal.V("2-digit"), TimeZone: temporal.V("-05:00")}).FormatRangeToParts(temporal.PlainDateFrom("1947-06-14", nil), temporal.PlainDateFrom("1947-06-14", nil))
		})
		out(func() any {
			return temporal.NewDateTimeFormat("de-u-hc-h12", temporal.DateTimeFormatOptions{Year: temporal.V("numeric"), Month: temporal.V("short"), TimeZone: temporal.V("-03:30")}).FormatRange(temporal.PlainDateFrom("2044-02-27", nil).WithCalendar("gregory").ToPlainYearMonth(), temporal.PlainDateFrom("2044-03-27", nil).WithCalendar("gregory").ToPlainYearMonth())
		})
		out(func() any {
			return temporal.NewDateTimeFormat("de-u-hc-h12", temporal.DateTimeFormatOptions{Year: temporal.V("numeric"), Month: temporal.V("short"), TimeZone: temporal.V("-03:30")}).FormatRangeToParts(temporal.PlainDateFrom("2044-02-27", nil).WithCalendar("gregory").ToPlainYearMonth(), temporal.PlainDateFrom("2044-03-27", nil).WithCalendar("gregory").ToPlainYearMonth())
		})
		out(func() any {
			return temporal.NewDateTimeFormat("de-u-hc-h12", temporal.DateTimeFormatOptions{Month: temporal.V("short"), Day: temporal.V("numeric"), TimeZone: temporal.V("+00:00")}).FormatRange(temporal.PlainDateFrom("2070-09-15", nil), temporal.PlainDateFrom("2070-09-15", nil))
		})
		out(func() any {
			return temporal.NewDateTimeFormat("de-u-hc-h12", temporal.DateTimeFormatOptions{Month: temporal.V("short"), Day: temporal.V("numeric"), TimeZone: temporal.V("+00:00")}).FormatRangeToParts(temporal.PlainDateFrom("2070-09-15", nil), temporal.PlainDateFrom("2070-09-15", nil))
		})
	}, []string{"\"11. Sept.\"", "[{\"day\":\"11\",\"shared\"},{\"literal\":\". \",\"shared\"},{\"month\":\"Sept.\",\"shared\"}]", "\"Nov\"", "[{\"month\":\"Nov\",\"shared\"}]", "\"A – M\"", "[{\"month\":\"A\",\"startRange\"},{\"literal\":\" – \",\"shared\"},{\"month\":\"M\",\"endRange\"}]", "\"14.06.\"", "[{\"day\":\"14\",\"shared\"},{\"literal\":\".\",\"shared\"},{\"month\":\"06\",\"shared\"},{\"literal\":\".\",\"shared\"}]", "\"Feb.–März 2044\"", "[{\"month\":\"Feb.\",\"startRange\"},{\"literal\":\"–\",\"shared\"},{\"month\":\"März\",\"endRange\"},{\"literal\":\" \",\"shared\"},{\"year\":\"2044\",\"shared\"}]", "\"15. Sept.\"", "[{\"day\":\"15\",\"shared\"},{\"literal\":\". \",\"shared\"},{\"month\":\"Sept.\",\"shared\"}]"}},
	{"format_range_003", func() {
		out(func() any {
			return temporal.NewDateTimeFormat("it", temporal.DateTimeFormatOptions{TimeZone: temporal.V("+05:45")}).FormatRange(temporal.InstantFromEpochMilliseconds(float64(-22209338277000)), temporal.InstantFromEpochMilliseconds(float64(-22209338277000)))
		})
		out(func() any {
			return temporal.NewDateTimeFormat("it", temporal.DateTimeFormatOptions{TimeZone: temporal.V("+05:45")}).FormatRangeToParts(temporal.InstantFromEpochMilliseconds(float64(-22209338277000)), temporal.InstantFromEpochMilliseconds(float64(-22209338277000)))
		})
	}, []string{"\"19/03/1266, 17:07:03\"", "[{\"day\":\"19\",\"shared\"},{\"literal\":\"/\",\"shared\"},{\"month\":\"03\",\"shared\"},{\"literal\":\"/\",\"shared\"},{\"year\":\"1266\",\"shared\"},{\"literal\":\", \",\"shared\"},{\"hour\":\"17\",\"shared\"},{\"literal\":\":\",\"shared\"},{\"minute\":\"07\",\"shared\"},{\"literal\":\":\",\"shared\"},{\"second\":\"03\",\"shared\"}]"}},
	{"format_range_004", func() {
		out(func() any {
			return temporal.NewDateTimeFormat("ar-EG-u-hc-h23", temporal.DateTimeFormatOptions{Hour: temporal.V("numeric"), Minute: temporal.V("numeric"), Second: temporal.V("numeric"), FractionalSecondDigits: temporal.V(float64(2)), TimeZone: temporal.V("-03:30")}).FormatRange(temporal.PlainTimeFrom("00:16:22", nil), temporal.PlainTimeFrom("00:16:22", nil))
		})
		out(func() any {
			return temporal.NewDateTimeFormat("ar-EG-u-hc-h23", temporal.DateTimeFormatOptions{Hour: temporal.V("numeric"), Minute: temporal.V("numeric"), Second: temporal.V("numeric"), FractionalSecondDigits: temporal.V(float64(2)), TimeZone: temporal.V("-03:30")}).FormatRangeToParts(temporal.PlainTimeFrom("00:16:22", nil), temporal.PlainTimeFrom("00:16:22", nil))
		})
		out(func() any {
			return temporal.NewDateTimeFormat("ar-EG-u-hc-h23", temporal.DateTimeFormatOptions{DateStyle: temporal.V("full"), TimeStyle: temporal.V("full"), TimeZone: temporal.V("+00:00")}).FormatRange(float64(4001207700000), float64(4001207723000))
		})
		out(func() any {
			return temporal.NewDateTimeFormat("ar-EG-u-hc-h23", temporal.DateTimeFormatOptions{DateStyle: temporal.V("full"), TimeStyle: temporal.V("full"), TimeZone: temporal.V("+00:00")}).FormatRangeToParts(float64(4001207700000), float64(4001207723000))
		})
		out(func() any {
			return temporal.NewDateTimeFormat("ar-EG-u-hc-h23", temporal.DateTimeFormatOptions{TimeZone: temporal.V("+05:30")}).FormatRange(temporal.InstantFromEpochMilliseconds(float64(-1988148450000)), temporal.InstantFromEpochMilliseconds(float64(-1959290850000)))
		})
		out(func() any {
			return temporal.NewDateTimeFormat("ar-EG-u-hc-h23", temporal.DateTimeFormatOptions{TimeZone: temporal.V("+05:30")}).FormatRangeToParts(temporal.InstantFromEpochMilliseconds(float64(-1988148450000)), temporal.InstantFromEpochMilliseconds(float64(-1959290850000)))
		})
		out(func() any {
			return temporal.NewDateTimeFormat("ar-EG-u-hc-h23", temporal.DateTimeFormatOptions{Month: temporal.V("short"), Day: temporal.V("numeric"), TimeZone: temporal.V("UTC")}).FormatRange(temporal.PlainDateTimeFrom("2001-05-31T06:43:08", nil), temporal.PlainDateTimeFrom("2001-05-31T06:43:08", nil))
		})
		out(func() any {
			return temporal.NewDateTimeFormat("ar-EG-u-hc-h23", temporal.DateTimeFormatOptions{Month: temporal.V("short"), Day: temporal.V("numeric"), TimeZone: temporal.V("UTC")}).FormatRangeToParts(temporal.PlainDateTimeFrom("2001-05-31T06:43:08", nil), temporal.PlainDateTimeFrom("2001-05-31T06:43:08", nil))
		})
		out(func() any {
			return temporal.NewDateTimeFormat("ar-EG-u-hc-h23", temporal.DateTimeFormatOptions{FractionalSecondDigits: temporal.V(float64(1)), TimeZone: temporal.V("UTC")}).FormatRange(float64(3579066250000), float64(3579757450000))
		})
		out(func() any {
			return temporal.NewDateTimeFormat("ar-EG-u-hc-h23", temporal.DateTimeFormatOptions{FractionalSecondDigits: temporal.V(float64(1)), TimeZone: temporal.V("UTC")}).FormatRangeToParts(float64(3579066250000), float64(3579757450000))
		})
		out(func() any {
			return temporal.NewDateTimeFormat("ar-EG-u-hc-h23", temporal.DateTimeFormatOptions{TimeZone: temporal.V("+01:00")}).FormatRange(temporal.PlainTimeFrom("09:38:40", nil), temporal.PlainTimeFrom("21:38:40", nil))
		})
		out(func() any {
			return temporal.NewDateTimeFormat("ar-EG-u-hc-h23", temporal.DateTimeFormatOptions{TimeZone: temporal.V("+01:00")}).FormatRangeToParts(temporal.PlainTimeFrom("09:38:40", nil), temporal.PlainTimeFrom("21:38:40", nil))
		})
		out(func() any {
			return temporal.NewDateTimeFormat("ar-EG-u-hc-h23", temporal.DateTimeFormatOptions{TimeStyle: temporal.V("medium"), HourCycle: temporal.V("h24"), TimeZone: temporal.V("Etc/UTC")}).FormatRange(temporal.PlainTimeFrom("00:40:23", nil), temporal.PlainTimeFrom("00:40:23", nil))
		})
		out(func() any {
			return temporal.NewDateTimeFormat("ar-EG-u-hc-h23", temporal.DateTimeFormatOptions{TimeStyle: temporal.V("medium"), HourCycle: temporal.V("h24"), TimeZone: temporal.V("Etc/UTC")}).FormatRangeToParts(temporal.PlainTimeFrom("00:40:23", nil), temporal.PlainTimeFrom("00:40:23", nil))
		})
	}, []string{"\"٠٠:١٦:٢٢٫٠٠\"", "[{\"hour\":\"٠٠\",\"shared\"},{\"literal\":\":\",\"shared\"},{\"minute\":\"١٦\",\"shared\"},{\"literal\":\":\",\"shared\"},{\"second\":\"٢٢\",\"shared\"},{\"literal\":\"٫\",\"shared\"},{\"fractionalSecond\":\"٠٠\",\"shared\"}]", "\"الثلاثاء، ١٦ أكتوبر ٢٠٩٦، ٠٦:٣٥:٠٠ غرينتش+٠ – ٠٦:٣٥:٢٣ غرينتش+٠\"", "[{\"weekday\":\"الثلاثاء\",\"shared\"},{\"literal\":\"، \",\"shared\"},{\"day\":\"١٦\",\"shared\"},{\"literal\":\" \",\"shared\"},{\"month\":\"أكتوبر\",\"shared\"},{\"literal\":\" \",\"shared\"},{\"year\":\"٢٠٩٦\",\"shared\"},{\"literal\":\"، \",\"shared\"},{\"hour\":\"٠٦\",\"startRange\"},{\"literal\":\":\",\"startRange\"},{\"minute\":\"٣٥\",\"startRange\"},{\"literal\":\":\",\"startRange\"},{\"second\":\"٠٠\",\"startRange\"},{\"literal\":\" \",\"startRange\"},{\"timeZoneName\":\"غرينتش+٠\",\"startRange\"},{\"literal\":\" – \",\"shared\"},{\"hour\":\"٠٦\",\"endRange\"},{\"literal\":\":\",\"endRange\"},{\"minute\":\"٣٥\",\"endRange\"},{\"literal\":\":\",\"endRange\"},{\"second\":\"٢٣\",\"endRange\"},{\"literal\":\" \",\"endRange\"},{\"timeZoneName\":\"غرينتش+٠\",\"endRange\"}]", "\"١‏/١‏/١٩٠٧، ٠٦:٠٢:٣٠ – ١‏/١٢‏/١٩٠٧، ٠٦:٠٢:٣٠\"", "[{\"day\":\"١\",\"startRange\"},{\"literal\":\"‏/\",\"startRange\"},{\"month\":\"١\",\"startRange\"},{\"literal\":\"‏/\",\"startRange\"},{\"year\":\"١٩٠٧\",\"startRange\"},{\"literal\":\"، \",\"startRange\"},{\"hour\":\"٠٦\",\"startRange\"},{\"literal\":\":\",\"startRange\"},{\"minute\":\"٠٢\",\"startRange\"},{\"literal\":\":\",\"startRange\"},{\"second\":\"٣٠\",\"startRange\"},{\"literal\":\" – \",\"shared\"},{\"day\":\"١\",\"endRange\"},{\"literal\":\"‏/\",\"endRange\"},{\"month\":\"١٢\",\"endRange\"},{\"literal\":\"‏/\",\"endRange\"},{\"year\":\"١٩٠٧\",\"endRange\"},{\"literal\":\"، \",\"endRange\"},{\"hour\":\"٠٦\",\"endRange\"},{\"literal\":\":\",\"endRange\"},{\"minute\":\"٠٢\",\"endRange\"},{\"literal\":\":\",\"endRange\"},{\"second\":\"٣٠\",\"endRange\"}]", "\"٣١ مايو\"", "[{\"day\":\"٣١\",\"shared\"},{\"literal\":\" \",\"shared\"},{\"month\":\"مايو\",\"shared\"}]", "\"١‏/٦‏/٢٠٨٣، ٠ – ٩‏/٦‏/٢٠٨٣، ٠\"", "[{\"day\":\"١\",\"startRange\"},{\"literal\":\"‏/\",\"startRange\"},{\"month\":\"٦\",\"startRange\"},{\"literal\":\"‏/\",\"startRange\"},{\"year\":\"٢٠٨٣\",\"startRange\"},{\"literal\":\"، \",\"startRange\"},{\"fractionalSecond\":\"٠\",\"startRange\"},{\"literal\":\" – \",\"shared\"},{\"day\":\"٩\",\"endRange\"},{\"literal\":\"‏/\",\"endRange\"},{\"month\":\"٦\",\"endRange\"},{\"literal\":\"‏/\",\"endRange\"},{\"year\":\"٢٠٨٣\",\"endRange\"},{\"literal\":\"، \",\"endRange\"},{\"fractionalSecond\":\"٠\",\"endRange\"}]", "\"٠٩:٣٨:٤٠ – ٢١:٣٨:٤٠\"", "[{\"hour\":\"٠٩\",\"startRange\"},{\"literal\":\":\",\"startRange\"},{\"minute\":\"٣٨\",\"startRange\"},{\"literal\":\":\",\"startRange\"},{\"second\":\"٤٠\",\"startRange\"},{\"literal\":\" – \",\"shared\"},{\"hour\":\"٢١\",\"endRange\"},{\"literal\":\":\",\"endRange\"},{\"minute\":\"٣٨\",\"endRange\"},{\"literal\":\":\",\"endRange\"},{\"second\":\"٤٠\",\"endRange\"}]", "\"٢٤:٤٠:٢٣\"", "[{\"hour\":\"٢٤\",\"shared\"},{\"literal\":\":\",\"shared\"},{\"minute\":\"٤٠\",\"shared\"},{\"literal\":\":\",\"shared\"},{\"second\":\"٢٣\",\"shared\"}]"}},
	{"format_range_008", func() {
		out(func() any {
			return temporal.NewDateTimeFormat("he", temporal.DateTimeFormatOptions{TimeZone: temporal.V("UTC")}).FormatRange(temporal.PlainDateFrom("1958-10-12", nil).WithCalendar("gregory").ToPlainYearMonth(), temporal.PlainDateFrom("1958-11-12", nil).WithCalendar("gregory").ToPlainYearMonth())
		})
		out(func() any {
			return temporal.NewDateTimeFormat("he", temporal.DateTimeFormatOptions{TimeZone: temporal.V("UTC")}).FormatRangeToParts(temporal.PlainDateFrom("1958-10-12", nil).WithCalendar("gregory").ToPlainYearMonth(), temporal.PlainDateFrom("1958-11-12", nil).WithCalendar("gregory").ToPlainYearMonth())
		})
	}, []string{"\"10.1958–11.1958\"", "[{\"month\":\"10\",\"startRange\"},{\"literal\":\".\",\"startRange\"},{\"year\":\"1958\",\"startRange\"},{\"literal\":\"–\",\"shared\"},{\"month\":\"11\",\"endRange\"},{\"literal\":\".\",\"endRange\"},{\"year\":\"1958\",\"endRange\"}]"}},
	{"format_range_009", func() {
		out(func() any {
			return temporal.NewDateTimeFormat("el", temporal.DateTimeFormatOptions{Day: temporal.V("numeric"), TimeZone: temporal.V("-03:30")}).FormatRange(temporal.PlainDateFrom("2003-01-16", nil).WithCalendar("gregory").ToPlainMonthDay(), temporal.PlainDateFrom("2003-01-16", nil).WithCalendar("gregory").ToPlainMonthDay())
		})
		out(func() any {
			return temporal.NewDateTimeFormat("el", temporal.DateTimeFormatOptions{Day: temporal.V("numeric"), TimeZone: temporal.V("-03:30")}).FormatRangeToParts(temporal.PlainDateFrom("2003-01-16", nil).WithCalendar("gregory").ToPlainMonthDay(), temporal.PlainDateFrom("2003-01-16", nil).WithCalendar("gregory").ToPlainMonthDay())
		})
	}, []string{"\"16\"", "[{\"day\":\"16\",\"shared\"}]"}},
}

var instant_mathCases = []goldenCase{
	{"instant_math_003", func() {
		a := temporal.InstantFrom("2091-02-28T08:34:42z")
		_ = a
		out(func() any {
			return a.ToString(temporal.InstantToStringOptions{SmallestUnit: temporal.V("millisecond"), TimeZone: temporal.V("+14")})
		})
	}, []string{"\"2091-02-28T22:34:42.000+14:00\""}},
	{"instant_math_007", func() {
		a := temporal.InstantFromEpochMilliseconds(float64(1e+21))
		_ = a
		out(func() any { return a })
		out(func() any { return a.EpochNanoseconds() })
		out(func() any { return a.EpochMilliseconds() })
	}, []string{"throw RangeError"}},
	{"instant_math_008", func() {
		a := temporal.InstantFromEpochMilliseconds(float64(2523701176015))
		_ = a
		out(func() any {
			return a.Round(temporal.RoundOptions{SmallestUnit: temporal.V("second"), RoundingMode: temporal.V("floor"), RoundingIncrement: temporal.V(float64(10))})
		})
	}, []string{"Instant(2049-12-21T12:06:10Z)"}},
	{"instant_math_009", func() {
		a := temporal.InstantFrom("2164-02-19T13:51:59.224226938-11")
		_ = a
		out(func() any { return temporal.InstantCompare(a, "1970-01-01T00:00Z") })
	}, []string{"1"}},
	{"instant_math_010", func() {
		a := temporal.InstantFromEpochMilliseconds(temporal.Null)
		_ = a
		out(func() any { return a })
		out(func() any { return a.EpochNanoseconds() })
		out(func() any { return a.EpochMilliseconds() })
	}, []string{"Instant(1970-01-01T00:00:00Z)", "0n", "0"}},
	{"instant_math_013", func() {
		a := temporal.InstantFromEpochMilliseconds(float64(8640000000000000))
		_ = a
		out(func() any { return a })
		out(func() any { return a.EpochNanoseconds() })
		out(func() any { return a.EpochMilliseconds() })
	}, []string{"Instant(+275760-09-13T00:00:00Z)", "8640000000000000000000n", "8640000000000000"}},
}

var iso_datesCases = []goldenCase{
	{"iso_dates_000", func() {
		out(func() any { return temporal.NewPlainDate(float64(-271821), float64(4), float64(18), nil) })
		out(func() any {
			return temporal.NewPlainDateTime(float64(-271821), float64(4), float64(18), nil, nil, nil, nil, nil, nil, nil)
		})
		out(func() any {
			return temporal.NewPlainDateTime(float64(-271821), float64(4), float64(18), float64(23), float64(59), float64(59), float64(999), float64(999), float64(999), nil)
		})
		out(func() any {
			return temporal.NewPlainDateTime(float64(-271821), float64(4), float64(18), float64(0), float64(0), float64(0), float64(0), float64(0), float64(1), nil)
		})
	}, []string{"throw RangeError", "throw RangeError", "throw RangeError", "throw RangeError"}},
	{"iso_dates_003", func() {
		a := temporal.PlainDateFrom("2042-03-25", nil)
		_ = a
		out(func() any {
			return a.Subtract(temporal.DurationBag{Years: temporal.V(float64(40)), Weeks: temporal.V(float64(3366)), Days: temporal.V(float64(579))}, nil)
		})
	}, []string{"PlainDate(1936-02-19)"}},
	{"iso_dates_004", func() {
		out(func() any {
			return temporal.PlainDateFrom(temporal.PlainDateBag{Year: temporal.V(math.Inf(-1)), Month: temporal.V(float64(6)), Day: temporal.V(float64(15))}, nil)
		})
		out(func() any { return temporal.NewPlainDate(math.Inf(-1), float64(6), float64(15), nil) })
	}, []string{"throw RangeError", "throw RangeError"}},
	{"iso_dates_006", func() {
		a := temporal.PlainYearMonthFrom(temporal.PlainYearMonthBag{Year: temporal.V(float64(2249)), Month: temporal.V(float64(10))}, nil)
		_ = a
		b := temporal.PlainMonthDayFrom(temporal.PlainMonthDayBag{Month: temporal.V(float64(10)), Day: temporal.V(float64(21))}, nil)
		_ = b
		out(func() any { return a })
		out(func() any { return b })
		out(func() any { return a.Add(temporal.DurationBag{Weeks: temporal.V(float64(-213))}, nil) })
		out(func() any { return b.ToPlainDate(temporal.PlainYearMonthBag{Year: temporal.V(float64(2249))}) })
		out(func() any { return a.ToPlainDate(temporal.PlainDateBag{Day: temporal.V(float64(4))}) })
	}, []string{"PlainYearMonth(2249-10)", "PlainMonthDay(10-21)", "PlainYearMonth(2245-10)", "PlainDate(2249-10-21)", "PlainDate(2249-10-04)"}},
	{"iso_dates_008", func() {
		out(func() any {
			return temporal.PlainDateFrom(temporal.PlainDateBag{Year: temporal.V(float64(2020)), Month: temporal.V(float64(-1.9)), Day: temporal.V(float64(15))}, nil)
		})
		out(func() any { return temporal.NewPlainDate(float64(2020), float64(-1.9), float64(15), nil) })
	}, []string{"throw RangeError", "throw RangeError"}},
	{"iso_dates_009", func() {
		a := temporal.PlainYearMonthFrom(temporal.PlainYearMonthBag{Year: temporal.V(float64(1419)), Month: temporal.V(float64(4))}, nil)
		_ = a
		b := temporal.PlainMonthDayFrom(temporal.PlainMonthDayBag{Month: temporal.V(float64(4)), Day: temporal.V(float64(9))}, nil)
		_ = b
		out(func() any { return a })
		out(func() any { return b })
		out(func() any { return a.Add(temporal.DurationBag{Years: temporal.V(float64(6))}, nil) })
		out(func() any { return b.ToPlainDate(temporal.PlainYearMonthBag{Year: temporal.V(float64(1419))}) })
		out(func() any { return a.ToPlainDate(temporal.PlainDateBag{Day: temporal.V(float64(30))}) })
	}, []string{"PlainYearMonth(1419-04)", "PlainMonthDay(04-09)", "PlainYearMonth(1425-04)", "PlainDate(1419-04-09)", "PlainDate(1419-04-30)"}},
}

var locale_stringsCases = []goldenCase{
	{"locale_strings_000", func() {
		out(func() any {
			return temporal.InstantFromEpochMilliseconds(float64(440278865891)).ToLocaleString("en-US-u-hc-h23", temporal.DateTimeFormatOptions{Hour: temporal.V("numeric"), Minute: temporal.V("numeric")})
		})
	}, []string{"\"19:41\""}},
	{"locale_strings_001", func() {
		out(func() any {
			return temporal.PlainDateFrom("2028-03-09", nil).WithCalendar("islamic-umalqura").ToPlainMonthDay().ToLocaleString("ar-SA", temporal.DateTimeFormatOptions{DateStyle: temporal.V("medium")})
		})
	}, []string{"throw RangeError"}},
	{"locale_strings_005", func() {
		out(func() any {
			return temporal.DurationFrom(temporal.DurationBag{Days: temporal.V(float64(1405)), Microseconds: temporal.V(float64(4740185)), Weeks: temporal.V(float64(273))}).ToLocaleString("fr-CA", temporal.DurationFormatOptions{Style: temporal.V("narrow")})
		})
		out(func() any {
			return temporal.DurationFrom(temporal.DurationBag{Months: temporal.V(float64(-19))}).ToLocaleString("pl", temporal.DurationFormatOptions{Style: temporal.V("digital"), DaysDisplay: temporal.V("always")})
		})
	}, []string{"\"273sem 1 405j 4 740 185μs\"", "\"-19 mies., 0 dni i 0:00:00\""}},
	{"locale_strings_006", func() {
		out(func() any {
			return temporal.ZonedDateTimeFrom("-000044-06-04T12:00[Europe/Moscow]", nil).ToLocaleString("hi-IN", nil)
		})
		out(func() any {
			return temporal.ZonedDateTimeFrom("-000044-06-04T12:00[Europe/Moscow]", nil).ToLocaleString("hi-IN", temporal.DateTimeFormatOptions{Hour: temporal.V("numeric"), Minute: temporal.V("numeric"), HourCycle: temporal.V("h11")})
		})
	}, []string{"\"4/6/45, 12:00:00 pm GMT+2:30:17\"", "\"0:00 pm\""}},
	{"locale_strings_007", func() {
		out(func() any {
			return temporal.PlainDateFrom("1059-06-29", nil).ToLocaleString("ar-SA-u-nu-latn", temporal.DateTimeFormatOptions{Year: temporal.V("numeric"), Month: temporal.V("short"), TimeZone: temporal.V("-12:00")})
		})
	}, []string{"\"يونيو 1059\""}},
	{"locale_strings_010", func() {
		out(func() any {
			return temporal.PlainDateFrom("2077-12-20", nil).ToLocaleString("ja-JP-u-hc-h11", temporal.DateTimeFormatOptions{Month: temporal.V("2-digit"), Day: temporal.V("2-digit")})
		})
	}, []string{"\"12/20\""}},
}

var parse_stringsCases = []goldenCase{
	{"parse_strings_000", func() {
		out(func() any { return temporal.DurationFrom("−P97M50DT87.5S") })
		out(func() any { return temporal.DurationFrom("p55M39W4Dt33H0.123456789S") })
		out(func() any { return temporal.DurationFrom("-P87Y73Wt48.S") })
	}, []string{"throw RangeError", "Duration(P55M39W4DT33H0.123456789S)", "throw RangeError"}},
	{"parse_strings_005", func() {
		out(func() any { return temporal.PlainDateFrom("21:05:39z[Etc/UTC]", nil) })
		out(func() any { return temporal.PlainTimeFrom("21:05:39z[Etc/UTC]", nil) })
		out(func() any { return temporal.PlainDateTimeFrom("21:05:39z[Etc/UTC]", nil) })
		out(func() any { return temporal.PlainYearMonthFrom("21:05:39z[Etc/UTC]", nil) })
		out(func() any { return temporal.PlainMonthDayFrom("21:05:39z[Etc/UTC]", nil) })
		out(func() any { return temporal.InstantFrom("21:05:39z[Etc/UTC]") })
	}, []string{"throw RangeError", "throw RangeError", "throw RangeError", "throw RangeError", "throw RangeError", "throw RangeError"}},
	{"parse_strings_007", func() {
		out(func() any { return temporal.PlainDateFrom("00:59:00.99-05:30:15,5[UTC][u-ca=iso8601]", nil) })
		out(func() any { return temporal.PlainTimeFrom("00:59:00.99-05:30:15,5[UTC][u-ca=iso8601]", nil) })
		out(func() any { return temporal.PlainDateTimeFrom("00:59:00.99-05:30:15,5[UTC][u-ca=iso8601]", nil) })
		out(func() any { return temporal.PlainYearMonthFrom("00:59:00.99-05:30:15,5[UTC][u-ca=iso8601]", nil) })
		out(func() any { return temporal.PlainMonthDayFrom("00:59:00.99-05:30:15,5[UTC][u-ca=iso8601]", nil) })
		out(func() any { return temporal.InstantFrom("00:59:00.99-05:30:15,5[UTC][u-ca=iso8601]") })
	}, []string{"throw RangeError", "PlainTime(00:59:00.99)", "throw RangeError", "throw RangeError", "throw RangeError", "throw RangeError"}},
	{"parse_strings_008", func() {
		out(func() any { return temporal.PlainDateFrom("05-01[!u-ca=iso8601]", nil) })
		out(func() any { return temporal.PlainTimeFrom("05-01[!u-ca=iso8601]", nil) })
		out(func() any { return temporal.PlainDateTimeFrom("05-01[!u-ca=iso8601]", nil) })
		out(func() any { return temporal.PlainYearMonthFrom("05-01[!u-ca=iso8601]", nil) })
		out(func() any { return temporal.PlainMonthDayFrom("05-01[!u-ca=iso8601]", nil) })
		out(func() any { return temporal.InstantFrom("05-01[!u-ca=iso8601]") })
	}, []string{"throw RangeError", "throw RangeError", "throw RangeError", "throw RangeError", "PlainMonthDay(05-01)", "throw RangeError"}},
	{"parse_strings_010", func() {
		out(func() any {
			return temporal.ZonedDateTimeFrom("2179-09-30T23:0-05:30:15[+00:00]", temporal.ZonedDateTimeFromOptions{Offset: temporal.V("use")})
		})
		out(func() any {
			return temporal.ZonedDateTimeFrom("2179-09-30T23:0-05:30:15[+00:00]", temporal.ZonedDateTimeFromOptions{Offset: temporal.V("ignore")})
		})
		out(func() any {
			return temporal.ZonedDateTimeFrom("2179-09-30T23:0-05:30:15[+00:00]", temporal.ZonedDateTimeFromOptions{Offset: temporal.V("prefer")})
		})
		out(func() any {
			return temporal.ZonedDateTimeFrom("2179-09-30T23:0-05:30:15[+00:00]", temporal.ZonedDateTimeFromOptions{Offset: temporal.V("reject")})
		})
	}, []string{"throw RangeError", "throw RangeError", "throw RangeError", "throw RangeError"}},
	{"parse_strings_012", func() {
		out(func() any { return temporal.PlainDateFrom("2050-03-20t08:19:07.7-05:30:15,5[]", nil) })
		out(func() any { return temporal.PlainTimeFrom("2050-03-20t08:19:07.7-05:30:15,5[]", nil) })
		out(func() any { return temporal.PlainDateTimeFrom("2050-03-20t08:19:07.7-05:30:15,5[]", nil) })
		out(func() any { return temporal.PlainYearMonthFrom("2050-03-20t08:19:07.7-05:30:15,5[]", nil) })
		out(func() any { return temporal.PlainMonthDayFrom("2050-03-20t08:19:07.7-05:30:15,5[]", nil) })
		out(func() any { return temporal.InstantFrom("2050-03-20t08:19:07.7-05:30:15,5[]") })
	}, []string{"throw RangeError", "throw RangeError", "throw RangeError", "throw RangeError", "throw RangeError", "throw RangeError"}},
}

var plain_timeCases = []goldenCase{
	{"plain_time_000", func() {
		t := temporal.NewPlainTime(float64(23), float64(59), float64(0), float64(0), float64(0), float64(0))
		_ = t
		u := temporal.PlainTimeFrom("19:08:01.135770135", nil)
		_ = u
		out(func() any { return t.Until(u, temporal.DifferenceOptions{LargestUnit: temporal.V("nanosecond")}) })
	}, []string{"Duration(-PT17458.864229865S)"}},
	{"plain_time_001", func() {
		t := temporal.PlainTimeFrom(temporal.PlainTimeBag{Hour: temporal.V(float64(13)), Minute: temporal.V(float64(19)), Second: temporal.V(float64(25)), Millisecond: temporal.V(float64(830)), Microsecond: temporal.V(float64(290)), Nanosecond: temporal.V(float64(981))}, nil)
		_ = t
		out(func() any { return t.ToString(temporal.PlainTimeToStringOptions{}) })
	}, []string{"\"13:19:25.830290981\""}},
	{"plain_time_002", func() {
		t := temporal.PlainTimeFrom(temporal.PlainTimeBag{Hour: temporal.V(float64(19)), Minute: temporal.V(float64(25)), Second: temporal.V(float64(44)), Millisecond: temporal.V(float64(990)), Microsecond: temporal.V(float64(411)), Nanosecond: temporal.V(float64(555))}, nil)
		_ = t
		u := temporal.PlainTimeFrom(temporal.PlainTimeBag{Hour: temporal.V(float64(23)), Minute: temporal.V(float64(59)), Second: temporal.V(float64(0)), Millisecond: temporal.V(float64(999)), Microsecond: temporal.V(float64(999)), Nanosecond: temporal.V(float64(999))}, nil)
		_ = u
		out(func() any {
			return t.Until(u, temporal.DifferenceOptions{SmallestUnit: temporal.V("second"), RoundingMode: temporal.V("floor"), RoundingIncrement: temporal.V(float64(6))})
		})
	}, []string{"Duration(PT4H33M12S)"}},
	{"plain_time_003", func() {
		t := temporal.PlainTimeFrom("23:59:59.999999", nil)
		_ = t
		out(func() any {
			return t.ToString(temporal.PlainTimeToStringOptions{FractionalSecondDigits: temporal.V(float64(8)), RoundingMode: temporal.V("trunc")})
		})
	}, []string{"\"23:59:59.99999900\""}},
	{"plain_time_005", func() {
		t := temporal.PlainTimeFrom("23:00:59.999999", nil)
		_ = t
		out(func() any {
			return t.Round(temporal.RoundOptions{SmallestUnit: temporal.V("nanosecond"), RoundingMode: temporal.V("halfEven"), RoundingIncrement: temporal.V(float64(20))})
		})
		out(func() any { return t.Round("second") })
	}, []string{"PlainTime(23:00:59.999999)", "PlainTime(23:01:00)"}},
	{"plain_time_006", func() {
		t := temporal.NewPlainTime(float64(16), float64(12), float64(20), float64(0), float64(0), float64(0))
		_ = t
		out(func() any {
			return t.Add(temporal.DurationBag{Seconds: temporal.V(float64(-106)), Microseconds: temporal.V(float64(-758566513)), Nanoseconds: temporal.V(float64(-22))})
		})
	}, []string{"PlainTime(15:57:55.433486978)"}},
}

var zone_disambiguationCases = []goldenCase{
	{"zone_disambiguation_002", func() {
		e := temporal.ZonedDateTimeFrom("1992-02-09T01:57:59[America/Sao_Paulo]", nil)
		_ = e
		out(func() any {
			return e.With(temporal.ZonedDateTimeBag{Hour: temporal.V(float64(10)), Minute: temporal.V(float64(2))}, temporal.ZonedDateTimeWithOptions{Offset: temporal.V("reject"), Disambiguation: temporal.V("reject")})
		})
		out(func() any { return e.WithPlainTime(nil) })
	}, []string{"ZonedDateTime(1992-02-09T10:02:59-03:00[America/Sao_Paulo])", "ZonedDateTime(1992-02-09T00:00:00-03:00[America/Sao_Paulo])"}},
	{"zone_disambiguation_003", func() {
		out(func() any {
			return temporal.ZonedDateTimeFrom("1965-09-30T00:02:59[Asia/Gaza]", temporal.ZonedDateTimeFromOptions{Disambiguation: temporal.V("compatible")})
		})
		out(func() any {
			return temporal.ZonedDateTimeFrom("1965-09-30T00:02:59[Asia/Gaza]", temporal.ZonedDateTimeFromOptions{Disambiguation: temporal.V("earlier")})
		})
		out(func() any {
			return temporal.ZonedDateTimeFrom("1965-09-30T00:02:59[Asia/Gaza]", temporal.ZonedDateTimeFromOptions{Disambiguation: temporal.V("later")})
		})
		out(func() any {
			return temporal.ZonedDateTimeFrom("1965-09-30T00:02:59[Asia/Gaza]", temporal.ZonedDateTimeFromOptions{Disambiguation: temporal.V("reject")})
		})
	}, []string{"ZonedDateTime(1965-09-30T00:02:59+03:00[Asia/Gaza])", "ZonedDateTime(1965-09-30T00:02:59+03:00[Asia/Gaza])", "ZonedDateTime(1965-09-30T00:02:59+03:00[Asia/Gaza])", "ZonedDateTime(1965-09-30T00:02:59+03:00[Asia/Gaza])"}},
	{"zone_disambiguation_004", func() {
		out(func() any {
			return temporal.ZonedDateTimeFrom("2008-03-29T17:39:00+00:09[Australia/West]", temporal.ZonedDateTimeFromOptions{Offset: temporal.V("use"), Disambiguation: temporal.V("later")})
		})
		out(func() any {
			return temporal.ZonedDateTimeFrom("2008-03-29T17:39:00+00:09[Australia/West]", temporal.ZonedDateTimeFromOptions{Offset: temporal.V("ignore"), Disambiguation: temporal.V("compatible")})
		})
		out(func() any {
			return temporal.ZonedDateTimeFrom("2008-03-29T17:39:00+00:09[Australia/West]", temporal.ZonedDateTimeFromOptions{Offset: temporal.V("prefer"), Disambiguation: temporal.V("compatible")})
		})
		out(func() any {
			return temporal.ZonedDateTimeFrom("2008-03-29T17:39:00+00:09[Australia/West]", temporal.ZonedDateTimeFromOptions{Offset: temporal.V("reject"), Disambiguation: temporal.V("earlier")})
		})
	}, []string{"ZonedDateTime(2008-03-30T02:30:00+09:00[Australia/West])", "ZonedDateTime(2008-03-29T17:39:00+09:00[Australia/West])", "ZonedDateTime(2008-03-29T17:39:00+09:00[Australia/West])", "throw RangeError"}},
	{"zone_disambiguation_005", func() {
		out(func() any {
			return temporal.ZonedDateTimeFrom("2073-03-26T02:03:00[Europe/Tiraspol]", temporal.ZonedDateTimeFromOptions{Disambiguation: temporal.V("compatible")})
		})
		out(func() any {
			return temporal.ZonedDateTimeFrom("2073-03-26T02:03:00[Europe/Tiraspol]", temporal.ZonedDateTimeFromOptions{Disambiguation: temporal.V("earlier")})
		})
		out(func() any {
			return temporal.ZonedDateTimeFrom("2073-03-26T02:03:00[Europe/Tiraspol]", temporal.ZonedDateTimeFromOptions{Disambiguation: temporal.V("later")})
		})
		out(func() any {
			return temporal.ZonedDateTimeFrom("2073-03-26T02:03:00[Europe/Tiraspol]", temporal.ZonedDateTimeFromOptions{Disambiguation: temporal.V("reject")})
		})
	}, []string{"ZonedDateTime(2073-03-26T02:03:00+02:00[Europe/Tiraspol])", "ZonedDateTime(2073-03-26T02:03:00+02:00[Europe/Tiraspol])", "ZonedDateTime(2073-03-26T02:03:00+02:00[Europe/Tiraspol])", "ZonedDateTime(2073-03-26T02:03:00+02:00[Europe/Tiraspol])"}},
	{"zone_disambiguation_006", func() {
		out(func() any {
			return temporal.ZonedDateTimeFrom("2003-03-09T02:56:00-00:03[America/Santiago]", temporal.ZonedDateTimeFromOptions{Offset: temporal.V("use"), Disambiguation: temporal.V("earlier")})
		})
		out(func() any {
			return temporal.ZonedDateTimeFrom("2003-03-09T02:56:00-00:03[America/Santiago]", temporal.ZonedDateTimeFromOptions{Offset: temporal.V("ignore"), Disambiguation: temporal.V("compatible")})
		})
		out(func() any {
			return temporal.ZonedDateTimeFrom("2003-03-09T02:56:00-00:03[America/Santiago]", temporal.ZonedDateTimeFromOptions{Offset: temporal.V("prefer"), Disambiguation: temporal.V("compatible")})
		})
		out(func() any {
			return temporal.ZonedDateTimeFrom("2003-03-09T02:56:00-00:03[America/Santiago]", temporal.ZonedDateTimeFromOptions{Offset: temporal.V("reject"), Disambiguation: temporal.V("compatible")})
		})
	}, []string{"ZonedDateTime(2003-03-08T23:59:00-03:00[America/Santiago])", "ZonedDateTime(2003-03-09T02:56:00-04:00[America/Santiago])", "ZonedDateTime(2003-03-09T02:56:00-04:00[America/Santiago])", "throw RangeError"}},
	{"zone_disambiguation_008", func() {
		e := temporal.ZonedDateTimeFrom("1986-03-15T01:57:00[America/Cuiaba]", nil)
		_ = e
		out(func() any {
			return e.Until(e.Add(temporal.DurationBag{Months: temporal.V(float64(6)), Hours: temporal.V(float64(4))}, nil), temporal.DifferenceOptions{LargestUnit: temporal.V("year")})
		})
		out(func() any {
			return e.Since(e.Subtract(temporal.DurationBag{Days: temporal.V(float64(196))}, nil), temporal.DifferenceOptions{LargestUnit: temporal.V("month")})
		})
	}, []string{"Duration(P6MT4H)", "Duration(P6M15D)"}},
}

var zone_identifiersCases = []goldenCase{
	{"zone_identifiers_001", func() {
		e := temporal.InstantFromEpochMilliseconds(float64(0)).ToZonedDateTimeISO("GMT+0")
		_ = e
		f := temporal.InstantFromEpochMilliseconds(float64(0)).ToZonedDateTimeISO("Kwajalein")
		_ = f
		out(func() any { return e.Equals(f) })
		out(func() any { return e.Equals(f.WithTimeZone("GMT+0")) })
		out(func() any { return e.Equals(e.WithTimeZone("GMT+0")) })
	}, []string{"false", "true", "true"}},
	{"zone_identifiers_003", func() {
		out(func() any { return temporal.InstantFromEpochMilliseconds(float64(0)).ToZonedDateTimeISO("PLT") })
	}, []string{"throw RangeError"}},
	{"zone_identifiers_004", func() {
		e := temporal.InstantFromEpochMilliseconds(float64(-813223800000)).ToZonedDateTimeISO("ACT")
		_ = e
		out(func() any { return e.GetTimeZoneTransition("next") })
		out(func() any { return e.GetTimeZoneTransition("previous") })
	}, []string{"throw RangeError"}},
	{"zone_identifiers_005", func() {
		out(func() any { return temporal.InstantFromEpochMilliseconds(float64(0)).ToZonedDateTimeISO("ECT") })
	}, []string{"throw RangeError"}},
	{"zone_identifiers_009", func() {
		out(func() any { return temporal.InstantFromEpochMilliseconds(float64(0)).ToZonedDateTimeISO("BST") })
	}, []string{"throw RangeError"}},
	{"zone_identifiers_010", func() {
		out(func() any {
			return temporal.InstantFromEpochMilliseconds(float64(1000)).ToZonedDateTimeISO("Brazil/DeNoronha").TimeZoneId()
		})
		out(func() any {
			return temporal.InstantFromEpochMilliseconds(float64(1000)).ToZonedDateTimeISO("Brazil/DeNoronha").Offset()
		})
	}, []string{"\"Brazil/DeNoronha\"", "\"-02:00\""}},
}

var zone_offsetsCases = []goldenCase{
	{"zone_offsets_000", func() {
		e := temporal.InstantFromEpochMilliseconds(float64(-212770213269)).ToZonedDateTimeISO("Etc/GMT-9")
		_ = e
		out(func() any { return e.WithTimeZone("US/East-Indiana") })
		out(func() any { return e.WithTimeZone("US/East-Indiana").ToPlainDateTime() })
	}, []string{"ZonedDateTime(1963-04-05T04:09:46.731-05:00[US/East-Indiana])", "PlainDateTime(1963-04-05T04:09:46.731)"}},
	{"zone_offsets_001", func() {
		e := temporal.InstantFromEpochMilliseconds(float64(6681379268569)).ToZonedDateTimeISO("America/Fort_Nelson")
		_ = e
		out(func() any { return e })
		out(func() any { return e.Offset() })
		out(func() any { return e.Hour() })
		out(func() any { return e.Minute() })
		out(func() any { return e.Second() })
	}, []string{"ZonedDateTime(2181-09-21T11:41:08.569-07:00[America/Fort_Nelson])", "\"-07:00\"", "11", "41", "8"}},
	{"zone_offsets_003", func() {
		i := temporal.InstantFromEpochMilliseconds(float64(527018400000))
		_ = i
		out(func() any { return i.ToString(temporal.InstantToStringOptions{TimeZone: temporal.V("Asia/Harbin")}) })
	}, []string{"\"1986-09-14T02:00:00+08:00\""}},
	{"zone_offsets_006", func() {
		i := temporal.InstantFromEpochMilliseconds(float64(9492182651870))
		_ = i
		out(func() any {
			return i.ToString(temporal.InstantToStringOptions{TimeZone: temporal.V("Pacific/Kiritimati")})
		})
	}, []string{"\"2270-10-18T19:24:11.87+14:00\""}},
	{"zone_offsets_007", func() {
		e := temporal.InstantFromEpochMilliseconds(float64(320223600000)).ToZonedDateTimeISO("America/Costa_Rica")
		_ = e
		out(func() any { return e.WithTimeZone("Africa/Ouagadougou") })
		out(func() any { return e.WithTimeZone("Africa/Ouagadougou").ToPlainDateTime() })
	}, []string{"ZonedDateTime(1980-02-24T07:00:00+00:00[Africa/Ouagadougou])", "PlainDateTime(1980-02-24T07:00:00)"}},
	{"zone_offsets_008", func() {
		e := temporal.InstantFromEpochMilliseconds(float64(-1796250021356)).ToZonedDateTimeISO("Pacific/Marquesas")
		_ = e
		out(func() any { return e })
		out(func() any { return e.Offset() })
		out(func() any { return e.Hour() })
		out(func() any { return e.Minute() })
		out(func() any { return e.Second() })
	}, []string{"ZonedDateTime(1913-01-29T16:09:38.644-09:30[Pacific/Marquesas])", "\"-09:30\"", "16", "9", "38"}},
}
