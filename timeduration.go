package temporal

import (
	"math"
	"math/big"
	"strconv"
	"strings"
)

// timeDuration is TimeDuration of lib/timeduration.ts. Its values are immutable.
type timeDuration struct {
	totalNs *big.Int
	sec     float64
	subsec  float64
}

// timeDurationMaxNs is TimeDuration.MAX.
var timeDurationMaxNs, _ = new(big.Int).SetString("9007199254740991999999999", 10)

// timeDurationZero is TimeDuration.ZERO.
var timeDurationZero = newTimeDuration(bigZero)

// newTimeDuration is the TimeDuration constructor.
func newTimeDuration(totalNs *big.Int) *timeDuration {
	assert(abs(totalNs).Cmp(timeDurationMaxNs) <= 0, "integer too big")

	sec := bigToNumber(new(big.Int).Quo(totalNs, bigBillion))
	subsec := bigToNumber(new(big.Int).Rem(totalNs, bigBillion))
	assert(math.Abs(sec) <= 9007199254740991, "seconds too big")
	assert(math.Abs(subsec) <= 999_999_999, "subseconds too big")
	return &timeDuration{totalNs: totalNs, sec: sec, subsec: subsec}
}

// timeDurationValidateNew is TimeDuration.validateNew.
func timeDurationValidateNew(totalNs *big.Int, operation string) *timeDuration {
	if abs(totalNs).Cmp(timeDurationMaxNs) > 0 {
		throwRange(operation + " of duration time units cannot exceed " + timeDurationMaxNs.String() + " s")
	}
	return newTimeDuration(totalNs)
}

// timeDurationFromEpochNsDiff is TimeDuration.fromEpochNsDiff.
func timeDurationFromEpochNsDiff(epochNs1, epochNs2 *big.Int) *timeDuration {
	diff := new(big.Int).Sub(epochNs1, epochNs2)
	// No extra validate step. Should instead fail assertion if too big
	return newTimeDuration(diff)
}

// timeDurationFromComponents is TimeDuration.fromComponents.
func timeDurationFromComponents(h, mins, s, ms, us, ns float64) *timeDuration {
	totalNs := bigFromNumber(ns)
	totalNs = new(big.Int).Add(totalNs, new(big.Int).Mul(bigFromNumber(us), bigThousand))
	totalNs = new(big.Int).Add(totalNs, new(big.Int).Mul(bigFromNumber(ms), bigMillion))
	totalNs = new(big.Int).Add(totalNs, new(big.Int).Mul(bigFromNumber(s), bigBillion))
	totalNs = new(big.Int).Add(totalNs, new(big.Int).Mul(bigFromNumber(mins), minuteNanosJSBI))
	totalNs = new(big.Int).Add(totalNs, new(big.Int).Mul(bigFromNumber(h), hourNanos))
	return timeDurationValidateNew(totalNs, "total")
}

func (d *timeDuration) abs() *timeDuration {
	return newTimeDuration(abs(d.totalNs))
}

func (d *timeDuration) add(other *timeDuration) *timeDuration {
	return timeDurationValidateNew(new(big.Int).Add(d.totalNs, other.totalNs), "sum")
}

func (d *timeDuration) add24HourDays(days float64) *timeDuration {
	assert(math.Trunc(days) == days && !math.IsInf(days, 0), "days must be an integer")
	return timeDurationValidateNew(new(big.Int).Add(d.totalNs, new(big.Int).Mul(bigFromNumber(days), dayNanosJSBI)), "sum")
}

func (d *timeDuration) addToEpochNs(epochNs *big.Int) *big.Int {
	return new(big.Int).Add(epochNs, d.totalNs)
}

func (d *timeDuration) cmp(other *timeDuration) float64 {
	return compare(d.totalNs, other.totalNs)
}

func (d *timeDuration) divmod(n float64) (quotient float64, remainder *timeDuration) {
	assert(n != 0, "division by zero")
	q, r := divmod(d.totalNs, bigFromNumber(n))
	return bigToNumber(q), newTimeDuration(r)
}

func (d *timeDuration) fdiv(n *big.Int) float64 {
	assert(n.Sign() != 0, "division by zero")
	quotient, remainder := divmod(d.totalNs, n)

	// Perform long division to calculate the fractional part of the quotient
	// remainder / n with more accuracy than 64-bit floating point division
	const precision = 50
	var decimalDigits strings.Builder
	count := 0
	sign := float64(n.Sign())
	if d.totalNs.Sign() < 0 {
		sign = -sign
	}
	for remainder.Sign() != 0 && count < precision {
		var digit *big.Int
		digit, remainder = divmod(new(big.Int).Mul(remainder, bigTen), n)
		decimalDigits.WriteString(strconv.FormatInt(abs(digit).Int64(), 10))
		count++
	}
	return float64(sign * stringToNumber(abs(quotient).String()+"."+decimalDigits.String()))
}

func (d *timeDuration) isZero() bool {
	return d.totalNs.Sign() == 0
}

func (d *timeDuration) round(increment *big.Int, mode string) *timeDuration {
	if increment.Cmp(bigOne) == 0 {
		return d
	}
	quotient, remainder := divmod(d.totalNs, increment)
	sign := "positive"
	if d.totalNs.Sign() < 0 {
		sign = "negative"
	}
	r1 := new(big.Int).Mul(abs(quotient), increment)
	r2 := new(big.Int).Add(r1, increment)
	cmp := compare(abs(new(big.Int).Mul(remainder, bigTwo)), increment)
	unsignedRoundingMode := getUnsignedRoundingMode(mode, sign)
	rounded := r1
	if remainder.Sign() != 0 {
		rounded = applyUnsignedRoundingMode(r1, r2, cmp, isEven(quotient), unsignedRoundingMode)
	}
	result := rounded
	if sign != "positive" {
		result = new(big.Int).Neg(rounded)
	}
	return timeDurationValidateNew(result, "rounding")
}

func (d *timeDuration) sign() float64 {
	return d.cmp(timeDurationZero)
}

func (d *timeDuration) subtract(other *timeDuration) *timeDuration {
	return timeDurationValidateNew(new(big.Int).Sub(d.totalNs, other.totalNs), "difference")
}
