package temporal

// calCycleInfo is a run of years that always holds the same number of months.
type calCycleInfo struct{ years, months float64 }

type calMonthCodeInfo struct {
	additionalMonths []string
	cycleInfo        *calCycleInfo
}

var calLeapMonthCodes = []string{"M01L", "M02L", "M03L", "M04L", "M05L", "M06L", "M07L", "M08L", "M09L", "M10L", "M11L", "M12L"}

var monthCodeInfo = map[string]*calMonthCodeInfo{
	"chinese":  {additionalMonths: calLeapMonthCodes},
	"coptic":   {additionalMonths: []string{"M13"}, cycleInfo: &calCycleInfo{years: 1, months: 13}},
	"dangi":    {additionalMonths: calLeapMonthCodes},
	"ethioaa":  {additionalMonths: []string{"M13"}, cycleInfo: &calCycleInfo{years: 1, months: 13}},
	"ethiopic": {additionalMonths: []string{"M13"}, cycleInfo: &calCycleInfo{years: 1, months: 13}},
	// Metonic cycle: 7 leap years every 19 years
	"hebrew": {additionalMonths: []string{"M05L"}, cycleInfo: &calCycleInfo{years: 19, months: 19*12 + 7}},
}

func calIncludes(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func isValidMonthCodeForCalendar(calendar, monthCode string) bool {
	monthNumber, isLeapMonth := parseMonthCode(monthCode)
	if !isLeapMonth && monthNumber >= 1 && monthNumber <= 12 {
		return true
	}
	if info := monthCodeInfo[calendar]; info != nil {
		return calIncludes(info.additionalMonths, monthCode)
	}
	return false
}

// resolveNonLunisolarMonth safely merges a month, monthCode pair into an integer month. If both are
// present, make sure they match. This logic doesn't work for lunisolar calendars! An empty overflow
// is the original's undefined.
func resolveNonLunisolarMonth(monthP *float64, monthCodeP *string, calendar, overflow string) (float64, string) {
	var month float64
	var monthCode string
	if monthCodeP == nil {
		if monthP == nil {
			throwType("Either month or monthCode are required")
		}
		month = *monthP
		// The ISO calendar uses the default (undefined) value because it does
		// constrain/reject after this method returns. Non-ISO calendars, however,
		// rely on this function to constrain/reject out-of-range `month` values.
		monthsPerYear := 12.0
		if info := monthCodeInfo[calendar]; info != nil {
			monthsPerYear = 12 + float64(len(info.additionalMonths))
		}
		if overflow == "reject" {
			rejectToRange(month, 1, monthsPerYear)
		}
		if overflow == "constrain" {
			month = constrainToRange(month, 1, monthsPerYear)
		}
		monthCode = createMonthCode(month, false)
	} else {
		monthCode = *monthCodeP
		if !isValidMonthCodeForCalendar(calendar, monthCode) {
			throwRange("Invalid monthCode: " + monthCode + " does not exist in calendar " + calendar)
		}
		monthNumber, _ := parseMonthCode(monthCode)
		if monthP != nil && *monthP != monthNumber {
			throwRange("monthCode " + monthCode + " and month " + numberToString(*monthP) + " must match if both are present")
		}
		month = monthNumber
	}
	return month, monthCode
}

// calEraInfoEntry is one canonical era of a calendar with its aliases.
type calEraInfoEntry struct {
	name    string
	aliases []string
}

var eraInfoEntries = map[string][]calEraInfoEntry{
	"buddhist":         {{name: "be"}},
	"coptic":           {{name: "am"}},
	"ethioaa":          {{name: "aa", aliases: []string{"mundi"}}},
	"ethiopic":         {{name: "am", aliases: []string{"incar"}}, {name: "aa", aliases: []string{"mundi"}}},
	"gregory":          {{name: "ce", aliases: []string{"ad"}}, {name: "bce", aliases: []string{"bc"}}},
	"hebrew":           {{name: "am"}},
	"indian":           {{name: "shaka"}},
	"islamic-civil":    {{name: "ah"}, {name: "bh"}},
	"islamic-tbla":     {{name: "ah"}, {name: "bh"}},
	"islamic-umalqura": {{name: "ah"}, {name: "bh"}},
	"japanese": {
		{name: "reiwa"},
		{name: "heisei"},
		{name: "showa"},
		{name: "taisho"},
		{name: "meiji"},
		{name: "ce", aliases: []string{"ad"}},
		{name: "bce", aliases: []string{"bc"}},
	},
	"persian": {{name: "ap"}},
	"roc":     {{name: "roc", aliases: []string{"minguo"}}, {name: "broc", aliases: []string{"before-roc", "minguo-qian"}}},
}

func calendarSupportsEra(calendar string) bool {
	_, ok := eraInfoEntries[calendar]
	return ok
}

// canonicalizeEraInCalendar returns nil where the original returns undefined.
func canonicalizeEraInCalendar(calendar, era string) *string {
	entries, ok := eraInfoEntries[calendar]
	if !ok {
		calAssertExistsFail()
	}
	for ix := range entries {
		canonicalName := entries[ix].name
		if era == canonicalName {
			return calStr(era)
		}
		if calIncludes(entries[ix].aliases, era) {
			return calStr(canonicalName)
		}
	}
	return nil
}
