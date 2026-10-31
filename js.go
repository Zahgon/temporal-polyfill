package temporal

import (
	"math"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"unicode"
)

// Error is a JavaScript exception as the original raises it. Every exception is raised as
// panic(*Error); Name is the constructor name (RangeError, TypeError, SyntaxError) and Message the
// exception's message.
type Error struct {
	Name    string
	Message string
}

// Error returns the exception the way JavaScript's Error.prototype.toString does.
func (e *Error) Error() string { return e.Name + ": " + e.Message }

func throwRange(msg string) { panic(&Error{Name: "RangeError", Message: msg}) }
func throwType(msg string)  { panic(&Error{Name: "TypeError", Message: msg}) }

func throwSyntax(msg string) { panic(&Error{Name: "SyntaxError", Message: msg}) }

type nullType struct{}

// Null is the JavaScript value null. In this package nil stands for undefined.
var Null = nullType{}

// Value holds one JavaScript value in a bag or options struct, whose fields are *Value so that an
// absent field (a nil pointer) differs from a zero value. Build it with V.
type Value struct{ v any }

// V wraps a JavaScript value: float64, string, bool, *big.Int (a BigInt), Null, nil (undefined), a
// Temporal object or a bag struct.
func V(x any) *Value { return &Value{v: x} }

// isUndefined reports whether x is undefined. A nil *Value also counts.
func isUndefined(x any) bool {
	if x == nil {
		return true
	}
	if v, ok := x.(*Value); ok && v == nil {
		return true
	}
	return false
}

func isNull(x any) bool { _, ok := x.(nullType); return ok }

// isBag reports whether x is one of the package's bag or options structs, by value or by pointer.
func isBag(x any) bool {
	t := reflect.TypeOf(x)
	if t == nil {
		return false
	}
	if t.Kind() == reflect.Pointer {
		if reflect.ValueOf(x).IsNil() {
			return false
		}
		t = t.Elem()
	}
	return t.Kind() == reflect.Struct && t != reflect.TypeOf(Value{}) && t != reflect.TypeOf(big.Int{}) && !isTemporalObject(x) && t != reflect.TypeOf(nullType{})
}

// isTemporalObject reports whether x is an instance of a Temporal class.
func isTemporalObject(x any) bool {
	switch x.(type) {
	case *Instant, *PlainDate, *PlainDateTime, *PlainTime, *PlainYearMonth, *PlainMonthDay, *ZonedDateTime, *Duration:
		return true
	}
	return false
}

// isObject is the ECMAScript Type(x) is Object test.
func isObject(x any) bool {
	if isUndefined(x) || isNull(x) {
		return false
	}
	switch x.(type) {
	case float64, string, bool, *big.Int:
		return false
	case *DateTimeFormat:
		return true
	}
	return isTemporalObject(x) || isBag(x)
}

// unwrap turns a *Value into what it holds; any other x is returned as it is.
func unwrap(x any) any {
	if v, ok := x.(*Value); ok {
		if v == nil {
			return nil
		}
		return v.v
	}
	return x
}

func upperFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// objGet is property access obj[key] on a bag struct or a Temporal object. A missing property is
// nil (undefined). Temporal objects answer through their getters.
func objGet(obj any, key string) any {
	if t, ok := obj.(temporalGetter); ok {
		return t.jsGet(key)
	}
	rv := reflect.ValueOf(obj)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil
	}
	f := rv.FieldByName(upperFirst(key))
	if !f.IsValid() || !f.CanInterface() {
		return nil
	}
	return unwrap(f.Interface())
}

// temporalGetter is implemented by each Temporal class; jsGet is what reading the JavaScript
// property of that name returns.
type temporalGetter interface{ jsGet(key string) any }

// toPrimitive is ECMAScript ToPrimitive for the values this package can hold. hint is "string",
// "number" or "default".
func toPrimitive(x any, hint string) any {
	x = unwrap(x)
	if !isObject(x) {
		return x
	}
	if hint == "string" {
		if s, ok := x.(interface{ jsToString() string }); ok {
			return s.jsToString()
		}
		return "[object Object]"
	}
	if t, ok := x.(interface{ jsValueOf() }); ok {
		t.jsValueOf() // the Temporal classes' valueOf throws
	}
	if s, ok := x.(interface{ jsToString() string }); ok {
		return s.jsToString()
	}
	return "[object Object]"
}

// jsToNumber is ECMAScript ToNumber.
func jsToNumber(x any) float64 {
	x = toPrimitive(x, "number")
	switch v := x.(type) {
	case nil:
		return math.NaN()
	case nullType:
		return 0
	case bool:
		if v {
			return 1
		}
		return 0
	case float64:
		return v
	case string:
		return stringToNumber(v)
	case *big.Int:
		throwType("Cannot convert a BigInt value to a number")
	}
	return math.NaN()
}

func isJSSpace(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', ' ', 0xA0, 0x1680, 0x2028, 0x2029, 0x202F, 0x205F, 0x3000, 0xFEFF:
		return true
	}
	return r >= 0x2000 && r <= 0x200A
}

func trimJSSpace(s string) string { return strings.TrimFunc(s, isJSSpace) }

// stringToNumber is ECMAScript StringToNumber.
func stringToNumber(str string) float64 {
	s := trimJSSpace(str)
	if s == "" {
		return 0
	}
	if len(s) > 2 && s[0] == '0' {
		base := 0
		switch s[1] {
		case 'x', 'X':
			base = 16
		case 'o', 'O':
			base = 8
		case 'b', 'B':
			base = 2
		}
		if base != 0 {
			n, ok := new(big.Int).SetString(s[2:], base)
			if !ok || strings.ContainsAny(s[2:], "+-_") {
				return math.NaN()
			}
			f, _ := new(big.Float).SetInt(n).Float64()
			return f
		}
	}
	body := s
	if body[0] == '+' || body[0] == '-' {
		body = body[1:]
	}
	if body == "Infinity" {
		if s[0] == '-' {
			return math.Inf(-1)
		}
		return math.Inf(1)
	}
	// StrDecimalLiteral: digits with an optional fraction and exponent; Go's parser also accepts
	// inf, nan, hex floats and underscores, so the characters are checked first.
	seenDigit := false
	for i := 0; i < len(body); i++ {
		c := body[i]
		switch {
		case c >= '0' && c <= '9':
			seenDigit = true
		case c == '.' || c == 'e' || c == 'E':
		case (c == '+' || c == '-') && i > 0 && (body[i-1] == 'e' || body[i-1] == 'E'):
		default:
			return math.NaN()
		}
	}
	if !seenDigit {
		return math.NaN()
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		if ne, ok := err.(*strconv.NumError); ok && ne.Err == strconv.ErrRange {
			return f
		}
		return math.NaN()
	}
	return f
}

// numberToString is ECMAScript Number::toString with radix 10.
func numberToString(x float64) string {
	switch {
	case math.IsNaN(x):
		return "NaN"
	case x == 0:
		return "0"
	case math.IsInf(x, 1):
		return "Infinity"
	case math.IsInf(x, -1):
		return "-Infinity"
	}
	neg := x < 0
	if neg {
		x = -x
	}
	formatted := strconv.FormatFloat(x, 'e', -1, 64)
	mantissa, expStr, _ := strings.Cut(formatted, "e")
	digits := strings.Replace(mantissa, ".", "", 1)
	exp, _ := strconv.Atoi(expStr)
	n := exp + 1
	k := len(digits)
	var out string
	switch {
	case k <= n && n <= 21:
		out = digits + strings.Repeat("0", n-k)
	case 0 < n && n <= 21:
		out = digits[:n] + "." + digits[n:]
	case -6 < n && n <= 0:
		out = "0." + strings.Repeat("0", -n) + digits
	default:
		e := n - 1
		sign := "+"
		if e < 0 {
			sign = "-"
			e = -e
		}
		if k == 1 {
			out = digits + "e" + sign + strconv.Itoa(e)
		} else {
			out = digits[:1] + "." + digits[1:] + "e" + sign + strconv.Itoa(e)
		}
	}
	if neg {
		return "-" + out
	}
	return out
}

// jsToString is ECMAScript ToString.
func jsToString(x any) string {
	x = toPrimitive(x, "string")
	switch v := x.(type) {
	case nil:
		return "undefined"
	case nullType:
		return "null"
	case bool:
		if v {
			return "true"
		}
		return "false"
	case float64:
		return numberToString(v)
	case string:
		return v
	case *big.Int:
		return v.String()
	}
	return "[object Object]"
}

// jsTypeof is the typeof operator.
func jsTypeof(x any) string {
	x = unwrap(x)
	switch x.(type) {
	case nil:
		return "undefined"
	case nullType:
		return "object"
	case bool:
		return "boolean"
	case float64:
		return "number"
	case string:
		return "string"
	case *big.Int:
		return "bigint"
	}
	return "object"
}

// jsTrunc is Math.trunc, keeping the sign of zero.
func jsTrunc(x float64) float64 { return math.Trunc(x) }

// jsMod is the % operator on Numbers.
func jsMod(a, b float64) float64 { return math.Mod(a, b) }

// jsSign is Math.sign.
func jsSign(x float64) float64 {
	switch {
	case math.IsNaN(x) || x == 0:
		return x
	case x > 0:
		return 1
	}
	return -1
}

func isJSWordRune(r rune) bool { return r == '_' || unicode.IsDigit(r) || unicode.IsLetter(r) }
