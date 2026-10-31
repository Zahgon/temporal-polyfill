package intl

import (
	"math"
	"sort"
	"strconv"
	"sync"
)

// lunarMeta describes one month table. The table file holds, for FirstYear onwards, one byte per year
// (the month a leap month follows, 0 for none; a year then has 12 or 13 months) and then the month lengths
// as one bit per month, 0 for 29 days and 1 for 30. Exceptions are single days on which ICU's floating-point
// astronomy disagrees with the months around them.
type lunarMeta struct {
	File       string  `json:"file"`
	FirstDay   int64   `json:"firstDay"`
	EndDay     int64   `json:"endDay"`
	FirstYear  int64   `json:"firstYear"`
	Years      int     `json:"years"`
	Era        string  `json:"era"`
	Exceptions [][]any `json:"exceptions"`
}

type lunarException struct {
	year   int64
	month  string
	day    int64
	failed bool
}

type lunarTable struct {
	meta       *lunarMeta
	yearStart  []int32 // first day of each year, plus the end of the table
	bitStart   []int32 // index of the year's first month bit
	leap       []byte
	bits       []byte
	exceptions map[int64]lunarException
}

var (
	lunarMu     sync.Mutex
	lunarTables = map[string]*lunarTable{}
)

func lunarTableFor(calendar string) *lunarTable {
	lunarMu.Lock()
	defer lunarMu.Unlock()
	if calendar == "islamic-rgsa" {
		calendar = "islamic"
	}
	if t, ok := lunarTables[calendar]; ok {
		return t
	}
	caldata()
	meta := cal.Lunar[calendar]
	if meta == nil {
		lunarTables[calendar] = nil
		return nil
	}
	raw := loadBytes(meta.File)
	t := &lunarTable{meta: meta, leap: raw[:meta.Years], bits: raw[meta.Years:], exceptions: map[int64]lunarException{}}
	t.yearStart = make([]int32, meta.Years+1)
	t.bitStart = make([]int32, meta.Years+1)
	day, bit := meta.FirstDay, int32(0)
	for y := 0; y < meta.Years; y++ {
		t.yearStart[y], t.bitStart[y] = int32(day), bit
		n := 12
		if t.leap[y] != 0 {
			n = 13
		}
		for m := 0; m < n; m++ {
			day += 29 + int64(t.bits[bit>>3]>>(bit&7)&1)
			bit++
		}
	}
	t.yearStart[meta.Years], t.bitStart[meta.Years] = int32(day), bit
	for _, e := range meta.Exceptions {
		d := int64(e[0].(float64))
		if len(e) == 2 {
			t.exceptions[d] = lunarException{failed: true}
			continue
		}
		t.exceptions[d] = lunarException{year: int64(e[1].(float64)), month: e[2].(string), day: int64(e[3].(float64))}
	}
	lunarTables[calendar] = t
	return t
}

func lunarFields(calendar string, day int64) (calFields, bool) {
	t := lunarTableFor(calendar)
	if t == nil || day < t.meta.FirstDay || day >= t.meta.EndDay {
		return calFields{}, false
	}
	if e, ok := t.exceptions[day]; ok {
		if e.failed {
			return calFields{Failed: true}, true
		}
		return calFields{Era: t.meta.Era, Year: e.year, Month: lunarMonthName(calendar, e.month), Day: e.day}, true
	}
	y := sort.Search(t.meta.Years, func(i int) bool { return int64(t.yearStart[i+1]) > day })
	start := int64(t.yearStart[y])
	bit := t.bitStart[y]
	leapAfter := int(t.leap[y])
	n := 12
	if leapAfter != 0 {
		n = 13
	}
	for i := 0; i < n; i++ {
		l := 29 + int64(t.bits[bit>>3]>>(bit&7)&1)
		if day < start+l {
			month, isLeap := i+1, false
			if leapAfter != 0 && i >= leapAfter {
				month = i
				isLeap = i == leapAfter
			}
			label := strconv.Itoa(month)
			if isLeap {
				label += "bis"
			}
			return calFields{Era: t.meta.Era, Year: t.meta.FirstYear + int64(y), Month: lunarMonthName(calendar, label), Day: day - start + 1}, true
		}
		start += l
		bit++
	}
	return calFields{}, false
}

// lunarMonthName maps a table's numeric month to the field key of the calendar's templates.
func lunarMonthName(calendar, label string) string {
	if calendar == "chinese" || calendar == "dangi" {
		return label
	}
	m, _ := strconv.Atoi(label)
	return monthsIslamic[m-1]
}

// islamicInstantGap reports the dates the islamic and islamic-rgsa table cannot answer. ICU's astronomical
// calendar asks for the moon's age at the instant, not on the local day, so on the days around a month
// start a zone whose local day differs from the UTC day can see another month than the table, which was
// recorded in UTC. The polyfill replaces both calendars before formatting, so nothing reaches these.
func islamicInstantGap(calendar string, cf calFields, day int64, ms float64) bool {
	if calendar != "islamic" && calendar != "islamic-rgsa" {
		return false
	}
	return int64(math.Floor(ms/86400000)) != day && (cf.Day == 1 || cf.Day >= 29)
}

const islamicGapMessage = "islamic date near a month start in a zone off the UTC day"
