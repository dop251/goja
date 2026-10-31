package goja

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
)

// parseTemporalDurationString parses an ISO 8601 duration string as accepted
// by Temporal.Duration.from().
func (r *Runtime) temporalDurationFromString(s string) *temporalDurationObject {
	fields, ok := parseTemporalDurationString(s)
	if !ok {
		panic(r.newError(r.getRangeError(), "Invalid ISO 8601 duration string: "+s))
	}
	r.validateTemporalDuration(fields)
	return r.durationFieldsObject(fields)
}

func parseTemporalDurationString(s string) ([10]float64, bool) {
	var fields [10]float64
	if s == "" {
		return fields, false
	}
	i := 0
	sign := float64(1)
	switch s[0] {
	case '-':
		sign = -1
		i++
	case '+':
		i++
	}
	if i >= len(s) || (s[i] != 'P' && s[i] != 'p') {
		return fields, false
	}
	i++
	if i >= len(s) {
		return fields, false
	}

	dateSeen := false
	timeSeen := false
	sawT := false
	sawFraction := false

	for i < len(s) {
		if s[i] == 'T' || s[i] == 't' {
			if sawT {
				return fields, false
			}
			sawT = true
			timeSeen = true
			i++
			if i >= len(s) {
				return fields, false
			}
			continue
		}
		intPart, fracNum, fracDigits, next, ok := parseTemporalDurationNumber(s, i)
		if !ok {
			return fields, false
		}
		i = next
		if i >= len(s) {
			return fields, false
		}
		designator := s[i]
		if designator >= 'a' && designator <= 'z' {
			designator -= 'a' - 'A'
		}
		i++
		if sawFraction {
			// A fraction is only allowed on the smallest (last) unit.
			return fields, false
		}
		fractional := fracDigits > 0
		if fractional {
			sawFraction = true
		}

		if !sawT {
			if fractional {
				return fields, false
			}
			switch designator {
			case 'Y':
				if fields[0] != 0 {
					return fields, false
				}
				fields[0] = intPart
			case 'M':
				if fields[1] != 0 {
					return fields, false
				}
				fields[1] = intPart
			case 'W':
				if fields[2] != 0 {
					return fields, false
				}
				fields[2] = intPart
			case 'D':
				if fields[3] != 0 {
					return fields, false
				}
				fields[3] = intPart
			default:
				return fields, false
			}
			dateSeen = true
			continue
		}

		switch designator {
		case 'H':
			if fields[4] != 0 {
				return fields, false
			}
			fields[4] = intPart
			if fractional {
				ns := temporalFractionNanos(fracNum, fracDigits, 3600e9)
				fields[5] += float64(ns / int64(60e9))
				ns %= int64(60e9)
				fields[6] += float64(ns / int64(1e9))
				fields[7] += float64((ns % int64(1e9)) / 1e6)
				fields[8] += float64((ns % int64(1e6)) / 1e3)
				fields[9] += float64(ns % int64(1e3))
			}
		case 'M':
			if fields[5] != 0 {
				return fields, false
			}
			fields[5] = intPart
			if fractional {
				ns := temporalFractionNanos(fracNum, fracDigits, 60e9)
				fields[6] += float64(ns / int64(1e9))
				fields[7] += float64((ns % int64(1e9)) / 1e6)
				fields[8] += float64((ns % int64(1e6)) / 1e3)
				fields[9] += float64(ns % int64(1e3))
			}
		case 'S':
			if fields[6] != 0 {
				return fields, false
			}
			fields[6] = intPart
			if fractional {
				ns := temporalFractionNanos(fracNum, fracDigits, 1e9)
				fields[7] += float64(ns / 1e6)
				fields[8] += float64((ns % 1e6) / 1e3)
				fields[9] += float64(ns % 1e3)
			}
		default:
			return fields, false
		}
		timeSeen = true
	}

	if !dateSeen && !timeSeen {
		return fields, false
	}
	for idx := range fields {
		fields[idx] *= sign
	}
	return fields, true
}

// parseTemporalDurationNumber parses a non-negative decimal number, returning
// the integer part (as float64), the fractional numerator, the number of
// fractional digits (at most 9) and the next index.
func parseTemporalDurationNumber(s string, i int) (intPart float64, fracNum int64, fracDigits, next int, ok bool) {
	start := i
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == start {
		return 0, 0, 0, i, false
	}
	v, err := strconv.ParseFloat(s[start:i], 64)
	if err != nil || v < 0 || math.IsInf(v, 0) || math.IsNaN(v) {
		return 0, 0, 0, i, false
	}
	intPart = v
	if i < len(s) && (s[i] == '.' || s[i] == ',') {
		i++
		fs := i
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
		if i == fs {
			return 0, 0, 0, i, false
		}
		fd := s[fs:i]
		if len(fd) > 9 {
			return 0, 0, 0, i, false
		}
		fv, err := strconv.ParseInt(fd, 10, 64)
		if err != nil {
			return 0, 0, 0, i, false
		}
		fracNum = fv
		fracDigits = len(fd)
	}
	return intPart, fracNum, fracDigits, i, true
}

// temporalFractionNanos converts a fractional component to nanoseconds, using
// half-expand rounding to the nearest nanosecond.
func temporalFractionNanos(num int64, digits int, unitNanos float64) int64 {
	if digits <= 0 {
		return 0
	}
	rat := new(big.Rat).SetFrac(big.NewInt(num), new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(digits)), nil))
	rat.Mul(rat, new(big.Rat).SetFloat64(unitNanos))
	f, _ := rat.Float64()
	return int64(math.Round(f))
}

type temporalRoundingMode string

const (
	temporalRoundCeil       temporalRoundingMode = "ceil"
	temporalRoundFloor      temporalRoundingMode = "floor"
	temporalRoundExpand     temporalRoundingMode = "expand"
	temporalRoundTrunc      temporalRoundingMode = "trunc"
	temporalRoundHalfCeil   temporalRoundingMode = "halfCeil"
	temporalRoundHalfFloor  temporalRoundingMode = "halfFloor"
	temporalRoundHalfExpand temporalRoundingMode = "halfExpand"
	temporalRoundHalfTrunc  temporalRoundingMode = "halfTrunc"
	temporalRoundHalfEven   temporalRoundingMode = "halfEven"
)

func (r *Runtime) parseTemporalRoundingMode(v Value, def temporalRoundingMode) temporalRoundingMode {
	if v == nil || IsUndefined(v) {
		return def
	}
	s, ok := temporalToStringOption(r, v)
	if !ok {
		return def
	}
	switch temporalRoundingMode(s) {
	case temporalRoundCeil, temporalRoundFloor, temporalRoundExpand, temporalRoundTrunc,
		temporalRoundHalfCeil, temporalRoundHalfFloor, temporalRoundHalfExpand,
		temporalRoundHalfTrunc, temporalRoundHalfEven:
		return temporalRoundingMode(s)
	}
	panic(r.newError(r.getRangeError(), "Invalid rounding mode"))
}

// roundMagnitude decides whether a magnitude should be incremented when
// rounding, given the truncated quotient and remainder.
func roundMagnitude(mode temporalRoundingMode, negative bool, q, rem, scale *big.Int) bool {
	if rem.Sign() == 0 {
		return false
	}
	half := new(big.Int).Mul(rem, big.NewInt(2))
	cmpHalf := half.Cmp(scale)
	switch mode {
	case temporalRoundTrunc:
		return false
	case temporalRoundExpand:
		return true
	case temporalRoundCeil:
		return !negative
	case temporalRoundFloor:
		return negative
	case temporalRoundHalfExpand, temporalRoundHalfCeil, temporalRoundHalfFloor, temporalRoundHalfTrunc, temporalRoundHalfEven:
		if cmpHalf > 0 {
			return true
		}
		if cmpHalf < 0 {
			return false
		}
		switch mode {
		case temporalRoundHalfCeil:
			return !negative
		case temporalRoundHalfFloor:
			return negative
		case temporalRoundHalfTrunc:
			return false
		case temporalRoundHalfEven:
			return q.Bit(0) == 1
		default:
			return true
		}
	}
	return false
}

// temporalToStringOption implements the ToString conversion used by string
// options, including the Symbol TypeError.
func temporalToStringOption(r *Runtime, v Value) (string, bool) {
	if v == nil || IsUndefined(v) {
		return "", false
	}
	if _, ok := v.(*Symbol); ok {
		panic(r.NewTypeError("Cannot convert a Symbol to a string"))
	}
	return v.toString().String(), true
}

// formatTemporalDuration serializes a duration, applying the options accepted
// by Temporal.Duration.prototype.toString().
func (r *Runtime) formatTemporalDuration(d *temporalDurationObject, options Value) string {
	digits := -1 // auto
	mode := temporalRoundTrunc
	if options != nil && !IsUndefined(options) {
		obj, ok := options.(*Object)
		if !ok {
			panic(r.NewTypeError("options must be an object"))
		}
		// Read all options before validating any of them (spec order:
		// fractionalSecondDigits, roundingMode, smallestUnit).
		if v := obj.self.getStr("fractionalSecondDigits", nil); v != nil && !IsUndefined(v) {
			switch v.(type) {
			case String:
				if v.(String).String() != "auto" {
					panic(r.newError(r.getRangeError(), "Invalid fractionalSecondDigits"))
				}
			case valueInt, valueFloat:
				f := v.ToNumber().ToFloat()
				if math.IsNaN(f) || math.IsInf(f, 0) {
					panic(r.newError(r.getRangeError(), "Invalid fractionalSecondDigits"))
				}
				f = math.Floor(f)
				if f < 0 || f > 9 {
					panic(r.newError(r.getRangeError(), "Invalid fractionalSecondDigits"))
				}
				digits = int(f)
			default:
				s, _ := temporalToStringOption(r, v)
				if s != "auto" {
					panic(r.newError(r.getRangeError(), "Invalid fractionalSecondDigits"))
				}
			}
		}
		mode = r.parseTemporalRoundingMode(obj.self.getStr("roundingMode", nil), temporalRoundTrunc)
		if v := obj.self.getStr("smallestUnit", nil); v != nil && !IsUndefined(v) {
			s, ok := temporalToStringOption(r, v)
			if !ok {
				panic(r.newError(r.getRangeError(), "Invalid smallestUnit"))
			}
			switch s {
			case "second", "seconds":
				digits = 0
			case "millisecond", "milliseconds":
				digits = 3
			case "microsecond", "microseconds":
				digits = 6
			case "nanosecond", "nanoseconds":
				digits = 9
			default:
				panic(r.newError(r.getRangeError(), "Invalid smallestUnit"))
			}
		}
	}

	fields := d.fields()
	negative := d.sign() < 0
	for i := range fields {
		fields[i] = temporalAbsFloat(fields[i])
	}

	// Balance all sub-second units into a single seconds value. This mirrors
	// the spec's formatting, which does not balance into minutes/hours.
	secondsRat := new(big.Rat).SetFloat64(fields[6])
	addFrac := func(v float64, div int64) {
		if v != 0 {
			secondsRat.Add(secondsRat, new(big.Rat).Quo(new(big.Rat).SetFloat64(v), big.NewRat(div, 1)))
		}
	}
	addFrac(fields[7], 1e3)
	addFrac(fields[8], 1e6)
	addFrac(fields[9], 1e9)

	secondsInt := new(big.Int).Quo(secondsRat.Num(), secondsRat.Denom())
	frac := new(big.Rat).Sub(secondsRat, new(big.Rat).SetInt(secondsInt))

	var fracStr string
	carried := false
	if digits < 0 {
		nsRat := new(big.Rat).Mul(frac, big.NewRat(1e9, 1))
		ns := new(big.Int).Quo(nsRat.Num(), nsRat.Denom())
		fracStr = fmt.Sprintf("%09d", ns.Int64())
		fracStr = strings.TrimRight(fracStr, "0")
	} else if digits > 0 {
		scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(9-digits)), nil)
		nsRat := new(big.Rat).Mul(frac, big.NewRat(1e9, 1))
		ns := new(big.Int).Quo(nsRat.Num(), nsRat.Denom())
		q, rem := new(big.Int).QuoRem(ns, scale, new(big.Int))
		if roundMagnitude(mode, negative, q, rem, scale) {
			q.Add(q, big.NewInt(1))
		}
		max := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(digits)), nil)
		if q.Cmp(max) >= 0 {
			secondsInt.Add(secondsInt, big.NewInt(1))
			q.Sub(q, max)
			carried = true
		}
		fracStr = fmt.Sprintf("%0*d", digits, q.Int64())
	} else {
		nsRat := new(big.Rat).Mul(frac, big.NewRat(1e9, 1))
		ns := new(big.Int).Quo(nsRat.Num(), nsRat.Denom())
		q := ns
		scale := big.NewInt(1e9)
		rem := new(big.Int).Mod(ns, scale)
		if roundMagnitude(mode, negative, q, rem, scale) {
			secondsInt.Add(secondsInt, big.NewInt(1))
			carried = true
		}
	}

	if carried && (fields[3] != 0 || fields[4] != 0 || fields[5] != 0) {
		mins := new(big.Int).SetInt64(int64(fields[5]))
		q, r := new(big.Int).QuoRem(secondsInt, big.NewInt(60), new(big.Int))
		mins.Add(mins, q)
		secondsInt = r
		hours := new(big.Int).SetInt64(int64(fields[4]))
		q, r = new(big.Int).QuoRem(mins, big.NewInt(60), new(big.Int))
		hours.Add(hours, q)
		mins = r
		days := new(big.Int).SetInt64(int64(fields[3]))
		q, r = new(big.Int).QuoRem(hours, big.NewInt(24), new(big.Int))
		days.Add(days, q)
		hours = r
		fields[3], _ = new(big.Float).SetInt(days).Float64()
		fields[4], _ = new(big.Float).SetInt(hours).Float64()
		fields[5], _ = new(big.Float).SetInt(mins).Float64()
	}
	// Validate the rounded representation (including days).
	vf := fields
	vf[6], _ = new(big.Float).SetInt(secondsInt).Float64()
	vf[7], vf[8], vf[9] = 0, 0, 0
	if secondsInt.Cmp(big.NewInt(temporalMaxSafeInteger)) > 0 {
		panic(r.newError(r.getRangeError(), "Duration is out of range"))
	}
	r.validateTemporalDuration(vf)
	forceTime := digits >= 0
	hasTime := fields[4] != 0 || fields[5] != 0 || secondsInt.Sign() != 0 || fracStr != "" || carried || forceTime
	hasDate := fields[0] != 0 || fields[1] != 0 || fields[2] != 0 || fields[3] != 0
	if !hasTime && !hasDate {
		return "PT0S"
	}

	var b strings.Builder
	if negative {
		b.WriteByte('-')
	}
	b.WriteByte('P')
	writeFloat := func(v float64, designator byte) {
		if v != 0 {
			b.WriteString(strconv.FormatFloat(v, 'f', 0, 64))
			b.WriteByte(designator)
		}
	}
	writeFloat(fields[0], 'Y')
	writeFloat(fields[1], 'M')
	writeFloat(fields[2], 'W')
	writeFloat(fields[3], 'D')
	if hasTime {
		b.WriteByte('T')
		writeFloat(fields[4], 'H')
		writeFloat(fields[5], 'M')
		if secondsInt.Sign() != 0 || fracStr != "" || carried || forceTime {
			b.WriteString(secondsInt.String())
			if fracStr != "" {
				b.WriteByte('.')
				b.WriteString(fracStr)
			}
			b.WriteByte('S')
		}
	}
	return b.String()
}
