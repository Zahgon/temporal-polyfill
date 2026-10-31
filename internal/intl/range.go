package intl

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
)

// rangeLevel is the first field DateIntervalFormat sees differ. ICU's chinese era is the 60-year cycle,
// and it compares the month field without the leap flag.
func rangeLevel(calendar string, a, b instantFields) string {
	aMonth, bMonth := a.cal.Month, b.cal.Month
	if isLunarCalendar(calendar) {
		aMonth, bMonth = strings.TrimSuffix(aMonth, "bis"), strings.TrimSuffix(bMonth, "bis")
		if floorDiv(a.cal.Year-1984, 60) != floorDiv(b.cal.Year-1984, 60) {
			return "G"
		}
	}
	switch {
	case a.cal.Era != b.cal.Era:
		return "G"
	case a.cal.Year != b.cal.Year:
		return "y"
	case aMonth != bMonth:
		return "M"
	case a.cal.Day != b.cal.Day:
		return "d"
	case (a.hour < 12) != (b.hour < 12):
		return "a"
	case a.hour != b.hour:
		return "h"
	case a.minute != b.minute:
		return "m"
	case a.second != b.second:
		return "s"
	case a.milli != b.milli:
		return "ms"
	}
	return "same"
}

// rangeSlots are the keys a range part's values may be recorded under, most specific first: the part's
// own type, source and occurrence within the shape, then the same type elsewhere.
func rangeSlots(typ, src string, n int) []string {
	keys := []string{typ + "|" + src + "|" + strconv.Itoa(n)}
	for _, s := range []string{src, "shared", "startRange", "endRange"} {
		for i := 0; i < 3; i++ {
			keys = append(keys, typ+"|"+s+"|"+strconv.Itoa(i))
		}
	}
	return keys
}

type mergeKey struct{ r, s int }

// completeList fills the values a range slot's sample pairs never reached from the single-date
// template, provided the two agree on at least one value and differ on none (the interval pattern
// renders that field the same way).
func (f *DateTimeFormat) completeList(rs *rangeShard, ri int, typ string) map[string]string {
	rl := rs.Lists[ri]
	si, ok := f.tpl.Lists[typ]
	if !ok {
		return rl
	}
	k := mergeKey{ri, si}
	if m, ok := rs.merged.Load(k); ok {
		return m.(map[string]string)
	}
	sl := f.shard.Lists[si]
	agree := false
	for key, v := range rl {
		if sv, ok := sl[key]; ok {
			if sv != v {
				rs.merged.Store(k, rl)
				return rl
			}
			agree = true
		}
	}
	if !agree {
		rs.merged.Store(k, rl)
		return rl
	}
	m := make(map[string]string, len(sl)+len(rl))
	for key, v := range sl {
		m[key] = v
	}
	for key, v := range rl {
		m[key] = v
	}
	rs.merged.Store(k, m)
	return m
}

func (f *DateTimeFormat) rangeData(rs *rangeShard, rt *rangeTemplate, src string, n int) partData {
	list := func(typ string) map[string]string {
		for _, k := range rangeSlots(typ, src, n) {
			if i, ok := rt.Lists[k]; ok {
				return f.completeList(rs, i, typ)
			}
		}
		return f.list(typ)
	}
	year := func(typ string) string {
		for _, k := range rangeSlots(typ, src, n) {
			if y, ok := rt.Years[k]; ok {
				return y
			}
		}
		return f.tpl.Years[typ]
	}
	minus := rt.Minus
	if minus == "" {
		minus = f.tpl.Minus
	}
	return partData{list: list, year: year, minus: minus, tz: rt.TZ}
}

func sharedParts(parts []Part) []RangePart {
	out := make([]RangePart, len(parts))
	for i, p := range parts {
		out[i] = RangePart{Type: p.Type, Value: p.Value, Source: "shared"}
	}
	return out
}

func joinRange(parts []RangePart) string {
	var b strings.Builder
	for _, p := range parts {
		b.WriteString(p.Value)
	}
	return b.String()
}

// formatRangeParts returns the parts and whether the range collapsed to one date.
func (f *DateTimeFormat) formatRangeParts(start, end float64) ([]RangePart, bool, error) {
	if err := checkTime(start); err != nil {
		return nil, false, err
	}
	if err := checkTime(end); err != nil {
		return nil, false, err
	}
	start, end = math.Trunc(start), math.Trunc(end)
	a, err := f.fieldsAt(start)
	if err != nil {
		return nil, false, err
	}
	b, err := f.fieldsAt(end)
	if err != nil {
		return nil, false, err
	}
	collapse := func() ([]RangePart, bool, error) {
		parts, err := f.FormatToParts(start)
		if err != nil {
			return nil, false, err
		}
		return sharedParts(parts), true, nil
	}
	level := rangeLevel(f.calendar, a, b)
	if level == "same" {
		return collapse()
	}
	rs := rangeShardFor(f.language)
	if rs == nil {
		return nil, false, &Unsupported{What: "date ranges for " + f.language}
	}
	levels := rs.Contexts[f.ctxKey][f.bagKey]
	if levels == nil {
		return nil, false, &Unsupported{What: "date range format " + f.bagKey + " in " + f.locale}
	}
	raw, ok := levels[level]
	if !ok && level == "G" {
		raw, ok = levels["y"]
	}
	if !ok {
		return nil, false, &Unsupported{What: "date range level " + level + " in " + f.locale}
	}
	var entry struct {
		R         *int `json:"r"`
		Collapsed bool `json:"collapsed"`
		Mixed     bool `json:"mixed"`
	}
	if err := json.Unmarshal(raw, &entry); err != nil {
		return nil, false, &Unsupported{What: "date range level " + level}
	}
	if entry.Collapsed || entry.R == nil {
		return collapse()
	}
	rt := rs.Templates[*entry.R]
	shape := rt.Shape
	firstEnd := len(shape)
	for i, p := range shape {
		if len(p) > 1 && p[1] == "endRange" {
			firstEnd = i
			break
		}
	}
	occ := make([]int, len(shape))
	seen := map[string]int{}
	for i, p := range shape {
		k := p[0] + "|" + p[1]
		occ[i] = seen[k]
		seen[k]++
	}
	parts, err := f.render(shape, func(i int, src string) (instantFields, partData) {
		x := a
		if src == "endRange" || (src == "shared" && i > firstEnd) {
			x = b
		}
		return x, f.rangeData(rs, rt, src, occ[i])
	})
	if err != nil {
		return nil, false, err
	}
	if entry.Mixed && sameSides(parts) {
		return collapse()
	}
	return parts, false, nil
}

// sameSides reports whether a range renders the same text on both sides: at some levels ICU then
// formats one date.
func sameSides(parts []RangePart) bool {
	var s, e strings.Builder
	for _, p := range parts {
		switch p.Source {
		case "startRange":
			s.WriteString(p.Value)
		case "endRange":
			e.WriteString(p.Value)
		}
	}
	return s.String() == e.String()
}

// FormatRangeToParts mirrors Intl.DateTimeFormat.prototype.formatRangeToParts.
func (f *DateTimeFormat) FormatRangeToParts(a, b float64) ([]RangePart, error) {
	parts, _, err := f.formatRangeParts(a, b)
	return parts, err
}

// FormatRange mirrors Intl.DateTimeFormat.prototype.formatRange: a range that collapses to one date is
// formatted by format(start); otherwise U+202F stays.
func (f *DateTimeFormat) FormatRange(a, b float64) (string, error) {
	parts, collapsed, err := f.formatRangeParts(a, b)
	if err != nil {
		return "", err
	}
	if collapsed {
		return strings.ReplaceAll(joinRange(parts), " ", " "), nil
	}
	return joinRange(parts), nil
}
