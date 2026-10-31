package temporal

import (
	"math"
	"math/big"
	"strconv"
	"strings"
)

// Constants of lib/bigintmath.ts. These values are shared: never mutate them.
var (
	bigZero         = big.NewInt(0)
	bigOne          = big.NewInt(1)
	bigTwo          = big.NewInt(2)
	bigTen          = big.NewInt(10)
	bigThousand     = big.NewInt(1e3)
	bigMillion      = big.NewInt(1e6)
	bigBillion      = big.NewInt(1e9)
	hourNanos       = big.NewInt(3600 * 1e9)
	minuteNanosJSBI = big.NewInt(60 * 1e9)
	dayNanosJSBI    = big.NewInt(24 * 3600 * 1e9)
)

// bigFromNumber is JSBI.BigInt(number).
func bigFromNumber(x float64) *big.Int {
	if math.IsNaN(x) || math.IsInf(x, 0) || math.Trunc(x) != x {
		throwRange("The number " + numberToString(x) + " cannot be converted to BigInt because it is not an integer")
	}
	result, _ := new(big.Float).SetFloat64(x).Int(nil)
	return result
}

// bigToNumber is JSBI.toNumber: the nearest double, ties to even, infinite beyond the double range.
func bigToNumber(x *big.Int) float64 {
	result, _ := new(big.Float).SetInt(x).Float64()
	return result
}

// isEven is isEven of lib/bigintmath.ts.
func isEven(value *big.Int) bool {
	return value.Bit(0) == 0
}

// abs is abs of lib/bigintmath.ts.
func abs(x *big.Int) *big.Int {
	if x.Sign() < 0 {
		return new(big.Int).Neg(x)
	}
	return x
}

// compare is compare of lib/bigintmath.ts.
func compare(x, y *big.Int) float64 {
	return float64(x.Cmp(y))
}

// divmod is divmod of lib/bigintmath.ts: the quotient truncates and the remainder has the dividend's
// sign.
func divmod(x, y *big.Int) (quotient, remainder *big.Int) {
	return new(big.Int).QuoRem(x, y, new(big.Int))
}

// jsLog is Math.log, V8's ieee754::log. Go's math.Log is not guaranteed to round the same way, and
// the digit counts below depend on the last bit.
func jsLog(xParam float64) float64 {
	const (
		ln2Hi = 6.93147180369123816490e-01 // 3fe62e42 fee00000
		ln2Lo = 1.90821492927058770002e-10 // 3dea39ef 35793c76
		two54 = 1.80143985094819840000e+16 // 43500000 00000000
		lg1   = 6.666666666666735130e-01   // 3FE55555 55555593
		lg2   = 3.999999999940941908e-01   // 3FD99999 9997FA04
		lg3   = 2.857142874366239149e-01   // 3FD24924 94229359
		lg4   = 2.222219843214978396e-01   // 3FCC71C5 1D8E78AF
		lg5   = 1.818357216161805012e-01   // 3FC74664 96CB03DE
		lg6   = 1.531383769920937332e-01   // 3FC39A09 D078C69F
		lg7   = 1.479819860511658591e-01   // 3FC2F112 DF3E5244
	)
	x := xParam
	bits := math.Float64bits(x)
	hx := int32(bits >> 32)
	lx := uint32(bits)
	k := int32(0)
	if hx < 0x00100000 { // x < 2**-1022
		if (hx&0x7fffffff)|int32(lx) == 0 {
			return math.Inf(-1) // log(+-0)=-inf
		}
		if hx < 0 {
			return math.NaN() // log(-#) = NaN
		}
		k -= 54
		x = float64(x * two54) // subnormal number, scale up x
		hx = int32(math.Float64bits(x) >> 32)
	}
	if hx >= 0x7ff00000 {
		return x + x
	}
	k += (hx >> 20) - 1023
	hx &= 0x000fffff
	i := (hx + 0x95f64) & 0x100000
	// normalize x or x/2
	x = math.Float64frombits(uint64(uint32(hx|(i^0x3ff00000)))<<32 | uint64(uint32(math.Float64bits(x))))
	k += i >> 20
	f := x - 1.0
	dk := float64(k)
	if 0x000fffff&(2+hx) < 3 { // -2**-20 <= f < 2**-20
		if f == 0 {
			if k == 0 {
				return 0
			}
			return float64(dk*ln2Hi) + float64(dk*ln2Lo)
		}
		r := float64(float64(f*f) * (0.5 - float64(0.33333333333333333*f)))
		if k == 0 {
			return f - r
		}
		return float64(dk*ln2Hi) - ((r - float64(dk*ln2Lo)) - f)
	}
	s := f / (2.0 + f)
	z := float64(s * s)
	i = hx - 0x6147a
	w := float64(z * z)
	j := 0x6b851 - hx
	t1 := float64(w * (lg2 + float64(w*(lg4+float64(w*lg6)))))
	t2 := float64(z * (lg1 + float64(w*(lg3+float64(w*(lg5+float64(w*lg7)))))))
	i |= j
	r := t2 + t1
	if i > 0 {
		hfsq := float64(float64(0.5*f) * f)
		if k == 0 {
			return f - (hfsq - float64(s*(hfsq+r)))
		}
		return float64(dk*ln2Hi) - ((hfsq - (float64(s*(hfsq+r)) + float64(dk*ln2Lo))) - f)
	}
	if k == 0 {
		return f - float64(s*(f-r))
	}
	return float64(dk*ln2Hi) - ((float64(s*(f-r)) - float64(dk*ln2Lo)) - f)
}

// jsLog10 is Math.log10, V8's ieee754::log10. It is exact on powers of ten, which Go's math.Log10 is
// not.
func jsLog10(xParam float64) float64 {
	const (
		two54   = 1.80143985094819840000e+16 // 0x43500000, 0x00000000
		ivln10  = 4.34294481903251816668e-01
		log2Hi  = 3.01029995663611771306e-01 // 0x3FD34413, 0x509F6000
		log2Lo  = 3.69423907715893078616e-13 // 0x3D59FEF3, 0x11F12B36
		signBit = uint32(0x80000000)
	)
	x := xParam
	bits := math.Float64bits(x)
	hx := int32(bits >> 32)
	lx := uint32(bits)
	k := int32(0)
	if hx < 0x00100000 { // x < 2**-1022
		if (hx&0x7fffffff)|int32(lx) == 0 {
			return math.Inf(-1) // log(+-0)=-inf
		}
		if hx < 0 {
			return math.NaN() // log(-#) = NaN
		}
		k -= 54
		x = float64(x * two54) // subnormal number, scale up x
		bits = math.Float64bits(x)
		hx = int32(bits >> 32)
		lx = uint32(bits)
	}
	if hx >= 0x7ff00000 {
		return x + x
	}
	if hx == 0x3ff00000 && lx == 0 {
		return 0 // log(1) = +0
	}
	k += (hx >> 20) - 1023
	i := int32((uint32(k) & signBit) >> 31)
	hx = (hx & 0x000fffff) | ((0x3ff - i) << 20)
	y := float64(k + i)
	x = math.Float64frombits(uint64(uint32(hx))<<32 | uint64(lx))
	z := float64(y*log2Lo) + float64(ivln10*jsLog(x))
	return z + float64(y*log2Hi)
}

// jsToPrecision is Number.prototype.toPrecision for x >= 0: exactly p significant digits, the larger
// candidate on a tie, exponential notation when the exponent is below -6 or not below p.
func jsToPrecision(x float64, p float64) string {
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return numberToString(x)
	}
	if p < 1 || p > 100 {
		throwRange("toPrecision() argument must be between 1 and 100")
	}
	precision := int(p)
	if x == 0 {
		if precision == 1 {
			return "0"
		}
		return "0." + strings.Repeat("0", precision-1)
	}
	// 800 digits hold every double's exact decimal expansion.
	mantissa, expStr, _ := strings.Cut(strconv.FormatFloat(x, 'e', 800, 64), "e")
	exact := strings.Replace(mantissa, ".", "", 1)
	e, _ := strconv.Atoi(expStr)
	digits := []byte(exact[:precision])
	if exact[precision] >= '5' {
		i := precision - 1
		for ; i >= 0; i-- {
			if digits[i] != '9' {
				digits[i]++
				break
			}
			digits[i] = '0'
		}
		if i < 0 {
			digits = append([]byte{'1'}, digits[:precision-1]...)
			e++
		}
	}
	m := string(digits)
	if e < -6 || e >= precision {
		out := m[:1]
		if precision != 1 {
			out += "." + m[1:]
		}
		if e >= 0 {
			return out + "e+" + strconv.Itoa(e)
		}
		return out + "e-" + strconv.Itoa(-e)
	}
	if e == precision-1 {
		return m
	}
	if e >= 0 {
		return m[:e+1] + "." + m[e+1:]
	}
	return "0." + strings.Repeat("0", -(e+1)) + m
}

// jsParseInt10 is Number.parseInt(str, 10) on the strings this file builds: a run of digits and
// whatever follows it. No digits give NaN.
func jsParseInt10(str string) float64 {
	end := 0
	for end < len(str) && str[end] >= '0' && str[end] <= '9' {
		end++
	}
	if end == 0 {
		return math.NaN()
	}
	// A range error still yields the infinity that parseInt gives.
	result, _ := strconv.ParseFloat(str[:end], 64)
	return result
}

// jsSlice is String.prototype.slice for non-negative positions on an ASCII string.
func jsSlice(s string, start, end int) string {
	if end > len(s) {
		end = len(s)
	}
	if start >= end {
		return ""
	}
	return s[start:end]
}

// padStartZeros is String.prototype.padStart(n, '0') on an ASCII string.
func padStartZeros(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return strings.Repeat("0", n-len(s)) + s
}

// truncatingDivModByPowerOf10 computes trunc(x / 10**p) and x % 10**p, returning (div, mod), with
// precision loss only once in the quotient, by string manipulation. If the quotient and remainder
// are safe integers, then they are exact. x must be an integer. p must be a non-negative integer.
// Both div and mod have the sign of x.
func truncatingDivModByPowerOf10(xParam, p float64) (div, mod float64) {
	x := xParam
	if x == 0 {
		return x, x // preserves signed zero
	}

	sign := jsSign(x)
	x = math.Abs(x)

	xDigits := jsTrunc(1 + jsLog10(x))
	if p >= xDigits {
		return float64(sign * 0), float64(sign * x)
	}
	if p == 0 {
		return float64(sign * x), float64(sign * 0)
	}

	// would perform nearest rounding if x was not an integer:
	xStr := jsToPrecision(x, xDigits)
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return math.NaN(), math.NaN() // parseInt finds no digits in NaN or Infinity
	}
	div = float64(sign * jsParseInt10(jsSlice(xStr, 0, int(xDigits-p))))
	mod = float64(sign * jsParseInt10(jsSlice(xStr, int(xDigits-p), len(xStr))))

	return div, mod
}

// fmaPowerOf10 computes x * 10**p + z with precision loss only at the end, by string manipulation.
// If the result is a safe integer, then it is exact. x must be an integer. p must be a non-negative
// integer. z must have the same sign as x and be less than 10**p.
func fmaPowerOf10(xParam, p, zParam float64) float64 {
	x := xParam
	z := zParam
	if x == 0 {
		return z
	}

	sign := jsSign(x)
	x = math.Abs(x)
	z = math.Abs(z)

	xStr := jsToPrecision(x, jsTrunc(1+jsLog10(x)))

	if z == 0 {
		return float64(sign * jsParseInt10(xStr+strings.Repeat("0", int(p))))
	}

	zStr := jsToPrecision(z, jsTrunc(1+jsLog10(z)))

	resStr := xStr + padStartZeros(zStr, int(p))
	return float64(sign * jsParseInt10(resStr))
}

// getUnsignedRoundingMode is GetUnsignedRoundingMode.
func getUnsignedRoundingMode(mode, sign string) string {
	isNegative := sign == "negative"
	switch mode {
	case "ceil":
		if isNegative {
			return "zero"
		}
		return "infinity"
	case "floor":
		if isNegative {
			return "infinity"
		}
		return "zero"
	case "expand":
		return "infinity"
	case "trunc":
		return "zero"
	case "halfCeil":
		if isNegative {
			return "half-zero"
		}
		return "half-infinity"
	case "halfFloor":
		if isNegative {
			return "half-infinity"
		}
		return "half-zero"
	case "halfExpand":
		return "half-infinity"
	case "halfTrunc":
		return "half-zero"
	case "halfEven":
		return "half-even"
	}
	return ""
}

// applyUnsignedRoundingMode omits the first step from the spec algorithm so that it can be used both
// for RoundNumberToIncrement and RoundTimeDurationToIncrement.
func applyUnsignedRoundingMode[T any](r1, r2 T, cmp float64, evenCardinality bool, unsignedRoundingMode string) T {
	if unsignedRoundingMode == "zero" {
		return r1
	}
	if unsignedRoundingMode == "infinity" {
		return r2
	}
	if cmp < 0 {
		return r1
	}
	if cmp > 0 {
		return r2
	}
	if unsignedRoundingMode == "half-zero" {
		return r1
	}
	if unsignedRoundingMode == "half-infinity" {
		return r2
	}
	if evenCardinality {
		return r1
	}
	return r2
}
