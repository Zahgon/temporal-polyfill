package intl

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// vectorDir holds the vectors recorded from Node by gen/vectors.mjs: testdata by default, a larger
// stress run when INTL_VECTOR_DIR is set.
func vectorDir() string {
	if d := os.Getenv("INTL_VECTOR_DIR"); d != "" {
		return d
	}
	return "testdata"
}

func eachVector(t *testing.T, name string, fn func(line []byte)) {
	t.Helper()
	paths, _ := filepath.Glob(filepath.Join(vectorDir(), name+"*.jsonl.gz"))
	if len(paths) == 0 {
		t.Fatalf("no vectors for %s in %s", name, vectorDir())
	}
	for _, p := range paths {
		base := filepath.Base(p)
		if base != name+".jsonl.gz" && !strings.HasPrefix(base, name+"_") {
			continue
		}
		f, err := os.Open(p)
		if err != nil {
			t.Fatal(err)
		}
		zr, err := gzip.NewReader(f)
		if err != nil {
			t.Fatal(err)
		}
		s := bufio.NewScanner(zr)
		s.Buffer(make([]byte, 1<<20), 1<<26)
		for s.Scan() {
			fn(s.Bytes())
		}
		f.Close()
	}
}

// tally counts mismatches, printing the first few of each kind.
type tally struct {
	t         *testing.T
	name      string
	total     int
	bad       map[string]int
	unsup     map[string]int
	maxReport int
	// documented counts vectors inside the one gap the package declares on purpose.
	documented int
}

func newTally(t *testing.T, name string) *tally {
	return &tally{t: t, name: name, bad: map[string]int{}, unsup: map[string]int{}, maxReport: 8}
}

func (x *tally) miss(kind, format string, args ...any) {
	x.bad[kind]++
	if x.bad[kind] <= x.maxReport {
		x.t.Errorf("%s %s: "+format, append([]any{x.name, kind}, args...)...)
	}
}

func (x *tally) gap(what string) {
	if what == islamicGapMessage {
		x.documented++
		return
	}
	x.unsup[what]++
}

func (x *tally) done() {
	n := 0
	for _, v := range x.bad {
		n += v
	}
	u := 0
	var us []string
	for k, v := range x.unsup {
		u += v
		us = append(us, k+"="+strconv.Itoa(v))
	}
	sort.Strings(us)
	if len(us) > 8 {
		us = us[:8]
	}
	x.t.Logf("%s: %d vectors, %d mismatches %v, %d Unsupported %v, %d in the islamic gap", x.name, x.total, n, x.bad, u, us, x.documented)
	fmt.Printf("RESULT %s: %d vectors, %d mismatches %v, %d Unsupported %v, %d in the islamic gap\n", x.name, x.total, n, x.bad, u, us, x.documented)
	if u > 0 {
		x.t.Errorf("%s: %d vectors came back Unsupported", x.name, u)
	}
}

func errText(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Name + ": " + e.Message
	}
	return ""
}

func parseTime(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case string:
		switch x {
		case "NaN":
			return math.NaN()
		case "Infinity":
			return math.Inf(1)
		case "-Infinity":
			return math.Inf(-1)
		}
		f, _ := strconv.ParseFloat(x, 64)
		return f
	}
	return math.NaN()
}

func joinParts(p []Part) string {
	var b strings.Builder
	for _, x := range p {
		b.WriteString(x.Type + "=" + x.Value + "|")
	}
	return b.String()
}

func joinJSONParts(p [][]string) string {
	var b strings.Builder
	for _, x := range p {
		b.WriteString(x[0] + "=" + x[1] + "|")
		if len(x) > 2 {
			b.WriteString(x[2] + "|")
		}
	}
	return b.String()
}

func joinRangeParts(p []RangePart) string {
	var b strings.Builder
	for _, x := range p {
		b.WriteString(x.Type + "=" + x.Value + "|" + x.Source + "|")
	}
	return b.String()
}

func TestCalendarParts(t *testing.T) {
	x := newTally(t, "calendar")
	eachVector(t, "calendar", func(line []byte) {
		var r struct {
			C, Z string
			T    any
			P    [][]string
			E    string
		}
		if err := json.Unmarshal(line, &r); err != nil {
			t.Fatal(err)
		}
		x.total++
		ms := parseTime(r.T)
		got, err := CalendarParts(r.C, ms, r.Z)
		var u *Unsupported
		if errors.As(err, &u) {
			x.gap(u.What)
			return
		}
		if r.E != "" {
			if errText(err) != r.E {
				x.miss(r.C, "%v %s: got err %v parts %v want error %s", r.T, r.Z, err, got, r.E)
			}
			return
		}
		if err != nil {
			x.miss(r.C, "%v %s: got error %v, want %v", r.T, r.Z, err, r.P)
			return
		}
		if joinParts(got) != joinJSONParts(r.P) {
			x.miss(r.C, "%v %s: got %v want %v", r.T, r.Z, got, r.P)
		}
	})
	x.done()
}

func TestOffsets(t *testing.T) {
	x := newTally(t, "tz")
	eachVector(t, "tz", func(line []byte) {
		if strings.HasPrefix(string(line), `{"supported"`) {
			return
		}
		var r struct {
			ID, Res string
			S       [][2]int64
		}
		if err := json.Unmarshal(line, &r); err != nil {
			t.Fatal(err)
		}
		if r.ID == "" {
			return
		}
		got, ok := CanonicalTimeZone(r.ID)
		if !ok || got != r.Res {
			x.miss("canonical", "%s: got %q %v want %q", r.ID, got, ok, r.Res)
		}
		if up, ok := CanonicalTimeZone(strings.ToUpper(r.ID)); !ok || up != r.Res {
			x.miss("canonical", "%s upper: got %q %v want %q", r.ID, up, ok, r.Res)
		}
		for _, s := range r.S {
			x.total++
			if o := OffsetSeconds(r.Res, s[0]); o != s[1] {
				x.miss("offset", "%s (%s) at %d: got %d want %d", r.ID, r.Res, s[0], o, s[1])
			}
		}
	})
	eachVector(t, "tzres", func(line []byte) {
		var r struct{ Z, R, E string }
		json.Unmarshal(line, &r)
		x.total++
		got, ok := CanonicalTimeZone(r.Z)
		if r.E != "" {
			if ok {
				x.miss("res", "%q: got %q want %s", r.Z, got, r.E)
			}
			return
		}
		if !ok || got != r.R {
			x.miss("res", "%q: got %q %v want %q", r.Z, got, ok, r.R)
		}
	})
	eachVector(t, "tzids", func(line []byte) {
		var r struct{ Supported []string }
		json.Unmarshal(line, &r)
		x.total++
		if !reflect.DeepEqual(SupportedTimeZones(), r.Supported) {
			x.miss("supported", "got %d want %d", len(SupportedTimeZones()), len(r.Supported))
		}
	})
	x.done()
}

func TestDateTimeFormat(t *testing.T) {
	x := newTally(t, "dtf")
	cache := map[string]*DateTimeFormat{}
	eachVector(t, "dtf", func(line []byte) {
		var r struct {
			L  string
			O  map[string]any
			T  float64
			P  [][]string
			F  string
			E  string
			RO map[string]any
		}
		if err := json.Unmarshal(line, &r); err != nil {
			t.Fatal(err)
		}
		x.total++
		ks, _ := json.Marshal(r.O)
		key := r.L + string(ks)
		f, ok := cache[key]
		if !ok {
			var err error
			f, err = NewDateTimeFormat(r.L, r.O)
			if err != nil {
				var u *Unsupported
				if errors.As(err, &u) {
					x.gap(u.What)
					return
				}
				if errText(err) != r.E {
					x.miss("ctor", "%s %v: got %v want %q", r.L, r.O, err, r.E)
				}
				return
			}
			cache[key] = f
		}
		parts, err := f.FormatToParts(r.T)
		if err != nil {
			var u *Unsupported
			if errors.As(err, &u) {
				x.gap(u.What)
				return
			}
			if errText(err) != r.E {
				x.miss("error", "%s %v at %v: got %v want %v %q", r.L, r.O, r.T, err, r.P, r.E)
			}
			return
		}
		if r.E != "" {
			x.miss("error", "%s %v at %v: got %v want %s", r.L, r.O, r.T, parts, r.E)
			return
		}
		if joinParts(parts) != joinJSONParts(r.P) {
			kind := "parts"
			if _, tz := r.O["timeZoneName"]; tz {
				kind = "parts-tz"
			}
			x.miss(kind, "%s %v at %v:\n got  %s\n want %s", r.L, r.O, r.T, joinParts(parts), joinJSONParts(r.P))
			return
		}
		s, err := f.Format(r.T)
		if err != nil || s != r.F {
			x.miss("format", "%s %v at %v: got %q want %q", r.L, r.O, r.T, s, r.F)
		}
	})
	x.done()
}

func TestFormatRange(t *testing.T) {
	x := newTally(t, "range")
	cache := map[string]*DateTimeFormat{}
	eachVector(t, "range", func(line []byte) {
		var r struct {
			L    string
			O    map[string]any
			A, B float64
			P    [][]string
			F    string
			E    string
		}
		if err := json.Unmarshal(line, &r); err != nil {
			t.Fatal(err)
		}
		x.total++
		ks, _ := json.Marshal(r.O)
		key := r.L + string(ks)
		f, ok := cache[key]
		if !ok {
			var err error
			f, err = NewDateTimeFormat(r.L, r.O)
			if err != nil {
				var u *Unsupported
				if errors.As(err, &u) {
					x.gap(u.What)
				} else {
					x.miss("ctor", "%s %v: %v", r.L, r.O, err)
				}
				return
			}
			cache[key] = f
		}
		parts, err := f.FormatRangeToParts(r.A, r.B)
		if err != nil {
			var u *Unsupported
			if errors.As(err, &u) {
				x.gap(u.What)
				if x.unsup[u.What] == 1 && len(x.unsup) < 40 {
					t.Logf("gap %s: %s %v %v..%v want %v", u.What, r.L, r.O, r.A, r.B, joinJSONParts(r.P))
				}
				return
			}
			if errText(err) != r.E {
				x.miss("error", "%s %v %v %v: got %v want %q", r.L, r.O, r.A, r.B, err, r.E)
			}
			return
		}
		if r.E != "" {
			x.miss("error", "%s %v %v..%v: got %v want %s", r.L, r.O, r.A, r.B, parts, r.E)
			return
		}
		if joinRangeParts(parts) != joinJSONParts(r.P) {
			x.miss("parts", "%s %v %v..%v:\n got  %s\n want %s", r.L, r.O, r.A, r.B, joinRangeParts(parts), joinJSONParts(r.P))
			return
		}
		s, err := f.FormatRange(r.A, r.B)
		if err != nil || s != r.F {
			x.miss("format", "%s %v %v..%v: got %q want %q", r.L, r.O, r.A, r.B, s, r.F)
		}
	})
	x.done()
}

func compareResolved(x *tally, kind, ctx string, got ResolvedOptions, want map[string]any) {
	m := map[string]any{"locale": got.Locale, "calendar": got.Calendar, "numberingSystem": got.NumberingSystem, "timeZone": got.TimeZone}
	s := func(k string, p *string) {
		if p != nil {
			m[k] = *p
		}
	}
	s("hourCycle", got.HourCycle)
	if got.Hour12 != nil {
		m["hour12"] = *got.Hour12
	}
	s("weekday", got.Weekday)
	s("era", got.Era)
	s("year", got.Year)
	s("month", got.Month)
	s("day", got.Day)
	s("dayPeriod", got.DayPeriod)
	s("hour", got.Hour)
	s("minute", got.Minute)
	s("second", got.Second)
	if got.FractionalSecondDigits != nil {
		m["fractionalSecondDigits"] = *got.FractionalSecondDigits
	}
	s("timeZoneName", got.TimeZoneName)
	s("dateStyle", got.DateStyle)
	s("timeStyle", got.TimeStyle)
	if !reflect.DeepEqual(m, want) {
		x.miss(kind, "%s:\n got  %v\n want %v", ctx, m, want)
	}
}

func TestResolvedOptions(t *testing.T) {
	x := newTally(t, "resolved")
	eachVector(t, "resolved", func(line []byte) {
		var r struct {
			L  string
			O  map[string]any
			RO map[string]any
		}
		if err := json.Unmarshal(line, &r); err != nil {
			t.Fatal(err)
		}
		x.total++
		f, err := NewDateTimeFormat(r.L, r.O)
		if err != nil {
			x.miss("ctor", "%s %v: %v", r.L, r.O, err)
			return
		}
		compareResolved(x, "ro", fmt.Sprint(r.L, r.O), f.ResolvedOptions(), r.RO)
	})
	eachVector(t, "dtf", func(line []byte) {
		var r struct {
			L  string
			O  map[string]any
			RO map[string]any
		}
		if err := json.Unmarshal(line, &r); err != nil {
			t.Fatal(err)
		}
		if r.RO == nil {
			return
		}
		x.total++
		f, err := NewDateTimeFormat(r.L, r.O)
		if err != nil {
			return
		}
		compareResolved(x, "ro-bag", fmt.Sprint(r.L, r.O), f.ResolvedOptions(), r.RO)
	})
	x.done()
}

func TestConstructorCases(t *testing.T) {
	x := newTally(t, "ctor")
	eachVector(t, "ctor", func(line []byte) {
		var r struct {
			L  any
			O  map[string]any
			RO map[string]any
			T  float64
			P  [][]string
			F  string
			E  string
		}
		if err := json.Unmarshal(line, &r); err != nil {
			t.Fatal(err)
		}
		x.total++
		ctx := fmt.Sprint(r.L, r.O)
		f, err := NewDateTimeFormat(r.L, r.O)
		if err != nil {
			var u *Unsupported
			if errors.As(err, &u) {
				x.gap(ctx + " " + u.What)
				return
			}
			if errText(err) != r.E {
				x.miss("error", "%s: got %v want %q", ctx, err, r.E)
			}
			return
		}
		if r.E != "" {
			x.miss("error", "%s: no error, want %s", ctx, r.E)
			return
		}
		compareResolved(x, "ro", ctx, f.ResolvedOptions(), r.RO)
		p, err := f.FormatToParts(r.T)
		if err != nil || joinParts(p) != joinJSONParts(r.P) {
			x.miss("parts", "%s: got %v %v want %v", ctx, p, err, r.P)
		}
	})
	eachVector(t, "edges", func(line []byte) {
		var r struct {
			T             any
			F             string
			P             [][]string
			E, RE, R, RE2 string
			R2            string
		}
		json.Unmarshal(line, &r)
		x.total++
		ms := parseTime(r.T)
		f, err := NewDateTimeFormat("en-US", map[string]any{"hour": "numeric", "minute": "numeric", "second": "numeric", "fractionalSecondDigits": 3.0, "timeZone": "UTC"})
		if err != nil {
			t.Fatal(err)
		}
		s, err := f.Format(ms)
		if r.E != "" {
			if errText(err) != r.E {
				x.miss("edge", "format %v: got %v want %s", r.T, err, r.E)
			}
		} else if err != nil || s != r.F {
			x.miss("edge", "format %v: got %q %v want %q", r.T, s, err, r.F)
		}
		s, err = f.FormatRange(ms, 1000)
		if r.RE != "" {
			if errText(err) != r.RE {
				x.miss("edge", "range %v: got %v want %s", r.T, err, r.RE)
			}
		} else if err != nil || s != r.R {
			x.miss("edge", "range %v: got %q %v want %q", r.T, s, err, r.R)
		}
		s, err = f.FormatRange(1000, ms)
		if r.RE2 != "" {
			if errText(err) != r.RE2 {
				x.miss("edge", "range2 %v: got %v want %s", r.T, err, r.RE2)
			}
		} else if err != nil || s != r.R2 {
			x.miss("edge", "range2 %v: got %q %v want %q", r.T, s, err, r.R2)
		}
	})
	x.done()
}

func TestDurationFormat(t *testing.T) {
	x := newTally(t, "duration")
	eachVector(t, "duration", func(line []byte) {
		var r struct {
			L any
			O map[string]any
			D map[string]float64
			F string
			E string
		}
		if err := json.Unmarshal(line, &r); err != nil {
			t.Fatal(err)
		}
		x.total++
		ctx := fmt.Sprint(r.L, r.O, r.D)
		f, err := NewDurationFormat(r.L, r.O)
		if err != nil {
			var u *Unsupported
			if errors.As(err, &u) {
				x.gap(u.What)
				return
			}
			if errText(err) != r.E {
				x.miss("ctor", "%s: got %v want %q", ctx, err, r.E)
			}
			return
		}
		got, err := f.Format(DurationRecord{r.D["years"], r.D["months"], r.D["weeks"], r.D["days"], r.D["hours"], r.D["minutes"], r.D["seconds"], r.D["milliseconds"], r.D["microseconds"], r.D["nanoseconds"]})
		if err != nil {
			if errText(err) != r.E {
				x.miss("format", "%s: got %v want %q %q", ctx, err, r.F, r.E)
			}
			return
		}
		if r.E != "" || got != r.F {
			x.miss("format", "%s:\n got  %q\n want %q %q", ctx, got, r.F, r.E)
		}
	})
	eachVector(t, "plural", func(line []byte) {
		var r struct {
			L, Loc string
			V      []float64
			C      []string
		}
		json.Unmarshal(line, &r)
		rules, err := pluralRulesFor(r.Loc)
		if err != nil {
			x.miss("plural", "%s: %v", r.L, err)
			return
		}
		for i, v := range r.V {
			x.total++
			ip, fp := strconv.FormatFloat(v, 'f', -1, 64), ""
			if got := rules(operandsOf(ip, fp)); got != r.C[i] {
				x.miss("plural", "%s %v: got %s want %s", r.L, v, got, r.C[i])
			}
		}
	})
	x.done()
}

func TestDataGaps(t *testing.T) {
	var u *Unsupported
	for _, c := range []string{"chinese", "dangi", "islamic"} {
		for _, ms := range []float64{-8.64e15, 8.64e15} {
			if _, err := CalendarParts(c, ms, "UTC"); !errors.As(err, &u) {
				t.Errorf("%s at %v: got %v, want Unsupported", c, ms, err)
			}
		}
	}
	for _, l := range []any{"fr-FR", "nb", "de-CH"} {
		if _, err := NewDateTimeFormat(l, nil); !errors.As(err, &u) {
			t.Errorf("locale %v: got %v, want Unsupported", l, err)
		}
		if _, err := NewDurationFormat(l, nil); !errors.As(err, &u) {
			t.Errorf("duration locale %v: got %v, want Unsupported", l, err)
		}
	}
	if _, err := NewDateTimeFormat("en-US", map[string]any{"hour": "numeric", "year": "2-digit", "weekday": "narrow", "timeZoneName": "long"}); !errors.As(err, &u) {
		t.Errorf("unrecorded bag: got %v, want Unsupported", err)
	}
	f, err := NewDateTimeFormat("en-US", map[string]any{"timeZone": "Asia/Dubai", "timeZoneName": "long"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Format(0); !errors.As(err, &u) {
		t.Errorf("names of an unrecorded zone: got %v, want Unsupported", err)
	}
	g, err := NewDateTimeFormat("en-US", map[string]any{"timeZone": "Asia/Dubai", "hour": "numeric"})
	if err != nil {
		t.Fatal(err)
	}
	if s, err := g.Format(0); err != nil || s == "" {
		t.Errorf("fields in an unrecorded zone: got %q, %v", s, err)
	}
}

func BenchmarkCalendarParts(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := CalendarParts("chinese", float64(i)*86400000, "UTC"); err != nil {
			b.Fatal(err)
		}
	}
}

// TestColdStart reports what a process pays on first use (run it alone: the tables are cached).
func TestColdStart(t *testing.T) {
	step := func(name string, fn func() error) {
		t0 := time.Now()
		if err := fn(); err != nil {
			t.Fatal(name, err)
		}
		t.Logf("%s: %v", name, time.Since(t0))
		fmt.Printf("COLD %s: %v\n", name, time.Since(t0))
	}
	step("OffsetSeconds", func() error { OffsetSeconds("America/New_York", 0); return nil })
	step("CalendarParts hebrew", func() error { _, err := CalendarParts("hebrew", 0, "UTC"); return err })
	step("CalendarParts chinese", func() error { _, err := CalendarParts("chinese", 0, "UTC"); return err })
	var f *DateTimeFormat
	step("NewDateTimeFormat en-US", func() (err error) { f, err = NewDateTimeFormat("en-US", nil); return })
	step("Format", func() error { _, err := f.Format(0); return err })
	step("FormatRange", func() error { _, err := f.FormatRange(0, 1e10); return err })
	step("NewDurationFormat", func() error { _, err := NewDurationFormat("en-US", nil); return err })
}
