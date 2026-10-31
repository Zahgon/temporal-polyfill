package intl

import (
	"math"
	"sort"
	"sync"
)

type namesZone struct {
	Periods [][2]int64 `json:"periods"`
	Far     [][2]int64 `json:"far"`
	once    sync.Once
	windows []nameWindow
}

// nameWindow is a short name period next to a DST transition: generic names change there, because ICU
// compares the zone with its metazone's reference zone on the wall clock. seconds is its length, and
// before says it ends at the transition rather than starting there.
type nameWindow struct {
	seconds int64
	offset  int64
	before  bool
	vector  int
}

type namesData struct {
	Contexts []string              `json:"contexts"`
	Strings  []string              `json:"strings"`
	Vectors  [][]int               `json:"vectors"`
	Zones    map[string]*namesZone `json:"zones"`
}

var (
	namesOnce  sync.Once
	names      namesData
	namesCtx   map[string]int
	namesLower map[string]string
)

const (
	namesEnd   = 4133980800 // 2101-01-01T00:00:00Z, the end of the scanned range
	namesSlots = 6
	// metazoneEnd is ICU's default end of a metazone mapping, 9999-12-31 23:59 UTC; later instants
	// get GMT-format names wherever the name came from a metazone.
	metazoneEnd = 253402300740
)

// nameSlots numbers the timeZoneName styles of a names vector.
var nameSlots = map[string]int{"short": 0, "long": 1, "shortOffset": 2, "longOffset": 3, "shortGeneric": 4, "longGeneric": 5}

func namedata() *namesData {
	namesOnce.Do(func() {
		loadJSON("names", &names)
		namesCtx = make(map[string]int, len(names.Contexts))
		for i, c := range names.Contexts {
			namesCtx[c] = i
		}
		namesLower = make(map[string]string, len(names.Zones))
		for id := range names.Zones {
			namesLower[asciiLower(id)] = id
		}
	})
	return &names
}

// zoneNameSlot reads one recorded name: context is "base|numberingSystem", zone the identifier given
// to the constructor, canon its resolved form. The bool is false when ICU shows no name part.
func zoneNameSlot(context, zone, canon string, ms float64, slot int) (string, bool, error) {
	d := namedata()
	ci, ok := namesCtx[context]
	if !ok {
		return "", false, &Unsupported{What: "zone names for locale " + context}
	}
	z := d.Zones[zone]
	if z == nil {
		z = d.Zones[namesLower[asciiLower(zone)]]
	}
	if z == nil {
		if _, ok := parseOffsetID(zone); ok {
			z = d.Zones[canon]
		}
		if z == nil {
			return "", false, &Unsupported{What: "zone names for " + zone}
		}
	}
	vec := z.vectorAt(canon, int64(math.Floor(ms/1000)))
	s := d.Vectors[vec][namesSlots*ci+slot]
	if s < 0 {
		return "", false, nil
	}
	return d.Strings[s], true, nil
}

func (z *namesZone) vectorAt(canon string, sec int64) int {
	p := z.Periods
	if sec < p[0][0] {
		for _, f := range z.Far {
			if f[0] < 0 {
				return int(f[1])
			}
		}
		return int(p[0][1])
	}
	if sec <= namesEnd {
		i := sort.Search(len(p), func(i int) bool { return p[i][0] > sec }) - 1
		return int(p[i][1])
	}
	off := OffsetSeconds(canon, sec)
	if sec >= metazoneEnd {
		for _, f := range z.Far {
			if f[0] >= metazoneEnd && OffsetSeconds(canon, f[0]) == off {
				return int(f[1])
			}
		}
	} else {
		z.once.Do(func() { z.windows = z.findWindows(canon) })
		for _, w := range z.windows {
			if w.offset != off {
				continue
			}
			if w.before && OffsetSeconds(canon, sec+w.seconds) != off {
				return w.vector
			}
			if !w.before && OffsetSeconds(canon, sec-w.seconds) != off {
				return w.vector
			}
		}
	}
	for i := len(p) - 1; i >= 0; i-- {
		next := int64(namesEnd)
		if i+1 < len(p) {
			next = p[i+1][0]
		}
		if next-p[i][0] > windowLimit && OffsetSeconds(canon, p[i][0]) == off {
			return int(p[i][1])
		}
	}
	return int(p[len(p)-1][1])
}

// windowLimit is the longest period taken for a window next to a transition.
const windowLimit = 86400

// findWindows reads the name windows from the last recorded years, where the zone follows its final rule.
func (z *namesZone) findWindows(canon string) []nameWindow {
	var out []nameWindow
	p := z.Periods
	for i := len(p) - 2; i >= 0 && p[i][0] > namesEnd-5*366*86400; i-- {
		length := p[i+1][0] - p[i][0]
		if length > windowLimit {
			continue
		}
		off := OffsetSeconds(canon, p[i][0])
		w := nameWindow{seconds: length, offset: off, vector: int(p[i][1])}
		switch {
		case OffsetSeconds(canon, p[i+1][0]) != off:
			w.before = true
		case OffsetSeconds(canon, p[i][0]-1) != off:
		default:
			continue
		}
		dup := false
		for _, o := range out {
			if o.seconds == w.seconds && o.offset == w.offset && o.before == w.before {
				dup = true
			}
		}
		if !dup {
			out = append(out, w)
		}
	}
	return out
}
