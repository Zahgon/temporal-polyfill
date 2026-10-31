package intl

import (
	"sort"
	"strings"
	"sync"
)

type localeDefaults struct {
	Calendar string `json:"ca"`
	NS       string `json:"nu"`
}

type localeData struct {
	Defaults         map[string]localeDefaults `json:"defaults"`
	Languages        []string                  `json:"languages"`
	Calendars        []string                  `json:"calendars"`
	NumberingSystems []string                  `json:"numberingSystems"`
	Digits           map[string][]string       `json:"digits"`
	languages        map[string]bool
	calendars        map[string]bool
	numbering        map[string]bool
}

var (
	locOnce sync.Once
	locData localeData
)

func locales() *localeData {
	locOnce.Do(func() {
		loadJSON("locale", &locData)
		locData.languages = map[string]bool{}
		for _, l := range locData.Languages {
			locData.languages[l] = true
		}
		locData.calendars = map[string]bool{}
		for _, l := range locData.Calendars {
			locData.calendars[l] = true
		}
		locData.numbering = map[string]bool{}
		for _, l := range locData.NumberingSystems {
			locData.numbering[l] = true
		}
	})
	return &locData
}

func errLocale() error { return rangeError("Incorrect locale information provided") }

func isRegion(s string) bool {
	return (len(s) == 2 && isAlpha(s)) || (len(s) == 3 && isDigits(s))
}

func isVariant(s string) bool {
	return isAlnum(s) && ((len(s) >= 5 && len(s) <= 8) || (len(s) == 4 && s[0] >= '0' && s[0] <= '9'))
}

// startsWithLanguageID is V8's first check of a tag: failing it is "Incorrect locale information
// provided", and failing the full parse after it "Invalid language tag".
func startsWithLanguageID(tag string) bool {
	tokens := strings.Split(tag, "-")
	if n := len(tokens); n > 1 && tokens[n-1] == "" {
		tokens = tokens[:n-1]
	}
	if len(tokens) == 0 {
		return false
	}
	l := tokens[0]
	if len(l) < 2 || len(l) > 8 || len(l) == 4 || !isAlpha(l) {
		return false
	}
	if len(tokens) == 1 {
		return true
	}
	singleton := func(s string) bool { return len(s) == 1 && isAlnum(s) }
	if singleton(tokens[1]) {
		return true
	}
	i := 1
	if len(tokens[i]) == 4 && isAlpha(tokens[i]) {
		i++
		if i == len(tokens) {
			return true
		}
	}
	if isRegion(tokens[i]) {
		i++
	}
	for i < len(tokens) {
		if singleton(tokens[i]) {
			return true
		}
		if !isVariant(tokens[i]) {
			return false
		}
		i++
	}
	return true
}

func isAlpha(s string) bool {
	for i := 0; i < len(s); i++ {
		if c := s[i] | 0x20; c < 'a' || c > 'z' {
			return false
		}
	}
	return true
}

func isDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func isAlnum(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c >= '0' && c <= '9') && !(c|0x20 >= 'a' && c|0x20 <= 'z') {
			return false
		}
	}
	return true
}

func titleCase(s string) string { return strings.ToUpper(s[:1]) + asciiLower(s[1:]) }

var languageAliases = map[string]string{"iw": "he", "in": "id", "ji": "yi", "jw": "jv", "mo": "ro", "tl": "fil", "no": "nb"}

// langTag is a canonicalized Unicode locale identifier, reduced to what Intl.DateTimeFormat reads.
type langTag struct {
	base     string
	language string
	keywords map[string]string
}

// parseLocale canonicalizes a language tag; a tag that is not structurally valid is a RangeError.
func parseLocale(tag string) (langTag, error) {
	t, ok := parseLocaleTag(tag)
	if ok {
		return t, nil
	}
	if !startsWithLanguageID(tag) {
		return langTag{}, errLocale()
	}
	return langTag{}, rangeError("Invalid language tag: " + asciiLower(tag))
}

func parseLocaleTag(tag string) (langTag, bool) {
	t, err := parseLocaleParts(tag)
	return t, err == nil
}

func parseLocaleParts(tag string) (langTag, error) {
	if tag == "" {
		return langTag{}, errLocale()
	}
	sub := strings.Split(tag, "-")
	for _, s := range sub {
		if s == "" || len(s) > 8 || !isAlnum(s) {
			return langTag{}, errLocale()
		}
	}
	i := 0
	lang := sub[0]
	if !isAlpha(lang) || len(lang) == 1 || len(lang) == 4 {
		return langTag{}, errLocale()
	}
	lang = asciiLower(lang)
	if a, ok := languageAliases[lang]; ok {
		lang = a
	}
	parts := []string{lang}
	i = 1
	if i < len(sub) && len(sub[i]) == 4 && isAlpha(sub[i]) {
		parts = append(parts, titleCase(sub[i]))
		i++
	}
	if i < len(sub) && ((len(sub[i]) == 2 && isAlpha(sub[i])) || (len(sub[i]) == 3 && isDigits(sub[i]))) {
		parts = append(parts, strings.ToUpper(sub[i]))
		i++
	}
	seenVariant := map[string]bool{}
	for i < len(sub) && ((len(sub[i]) >= 5) || (len(sub[i]) == 4 && sub[i][0] >= '0' && sub[i][0] <= '9')) {
		v := asciiLower(sub[i])
		if seenVariant[v] {
			return langTag{}, errLocale()
		}
		seenVariant[v] = true
		parts = append(parts, v)
		i++
	}
	t := langTag{base: strings.Join(parts, "-"), language: lang, keywords: map[string]string{}}
	seenSingleton := map[string]bool{}
	for i < len(sub) {
		s := asciiLower(sub[i])
		if len(s) != 1 {
			return langTag{}, errLocale()
		}
		if seenSingleton[s] {
			return langTag{}, errLocale()
		}
		seenSingleton[s] = true
		i++
		start := i
		if s == "x" {
			if i >= len(sub) {
				return langTag{}, errLocale()
			}
			return t, nil
		}
		for i < len(sub) && len(sub[i]) > 1 {
			i++
		}
		if i == start {
			return langTag{}, errLocale()
		}
		if s != "u" {
			continue
		}
		key := ""
		var value []string
		flush := func() {
			if key != "" {
				if _, dup := t.keywords[key]; !dup {
					t.keywords[key] = strings.Join(value, "-")
				}
			}
		}
		for _, w := range sub[start:i] {
			w = asciiLower(w)
			if len(w) == 2 {
				flush()
				key, value = w, nil
			} else if key != "" {
				value = append(value, w)
			}
		}
		flush()
	}
	return t, nil
}

func canonicalCalendar(c string) string {
	switch c {
	case "gregorian":
		return "gregory"
	case "ethiopic-amete-alem":
		return "ethioaa"
	case "islamicc":
		return "islamic-civil"
	}
	return c
}

// requestedLocales turns the locales argument into language tags.
func requestedLocales(v any) ([]string, error) {
	switch x := v.(type) {
	case nil:
		return nil, nil
	case string:
		return []string{x}, nil
	case []string:
		return x, nil
	case []any:
		out := make([]string, 0, len(x))
		for _, e := range x {
			s, ok := e.(string)
			if !ok {
				return nil, typeError("Incorrect locale information provided")
			}
			out = append(out, s)
		}
		return out, nil
	}
	return nil, typeError("Incorrect locale information provided")
}

const defaultLocale = "en-US"

// lookupLocale picks the tag ResolveLocale would pick: the first requested tag of a language the runtime
// has, else the default locale. A tag of such a language outside the recorded grid is a data gap.
func lookupLocale(v any) (langTag, error) {
	req, err := requestedLocales(v)
	if err != nil {
		return langTag{}, err
	}
	tags := make([]langTag, 0, len(req))
	for _, r := range req {
		t, err := parseLocale(r)
		if err != nil {
			return langTag{}, err
		}
		tags = append(tags, t)
	}
	l := locales()
	for _, t := range tags {
		if l.languages[t.language] {
			if _, ok := l.Defaults[t.base]; !ok {
				return t, &Unsupported{What: "locale " + t.base}
			}
			return t, nil
		}
	}
	return langTag{base: defaultLocale, language: "en", keywords: map[string]string{}}, nil
}

// resolvedLocaleTag writes the resolved base tag with the keywords that took effect, sorted.
func resolvedLocaleTag(base string, kw map[string]string) string {
	if len(kw) == 0 {
		return base
	}
	keys := make([]string, 0, len(kw))
	for k := range kw {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString(base)
	b.WriteString("-u")
	for _, k := range keys {
		b.WriteString("-" + k + "-" + kw[k])
	}
	return b.String()
}

func digitsFor(ns string) []string {
	if d, ok := locales().Digits[ns]; ok {
		return d
	}
	return nil
}
