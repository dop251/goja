package goja

import (
	"math"
	"math/big"
	"strings"

	"github.com/dop251/goja/unistring"
)

// Temporal is a namespace object (like Math or JSON) rather than a
// constructor. Types are added to it incrementally.
func (r *Runtime) getTemporal() *Object {
	ret := r.global.Temporal
	if ret == nil {
		o := r.newBaseObject(r.global.ObjectPrototype, classObject)
		ret = o.val
		r.global.Temporal = ret
		o._putProp("Duration", r.getTemporalDuration(), true, false, true)
		// Placeholders keep the namespace coherent while the remaining types
		// are implemented; they are replaced as their milestones land.
		o._putProp("PlainDate", r.getTemporalPlainDate(), true, false, true)
		o._putProp("PlainTime", r.getTemporalPlainTime(), true, false, true)
		o._putProp("PlainDateTime", r.getTemporalPlainDateTime(), true, false, true)
		o._putProp("Instant", r.getTemporalInstant(), true, false, true)
		o._putProp("ZonedDateTime", r.getTemporalZonedDateTime(), true, false, true)
		o._putProp("PlainYearMonth", r.getTemporalPlainYearMonth(), true, false, true)
		o._putProp("PlainMonthDay", r.getTemporalPlainMonthDay(), true, false, true)
		o._putProp("Now", r.getTemporalNow(), true, false, true)
		o._putSym(SymToStringTag, &valueProperty{
			value:        asciiString("Temporal"),
			writable:     false,
			enumerable:   false,
			configurable: true,
		})
	}
	return ret
}

const (
	// Temporal date fields are limited to 2^32-1 by IsValidDuration.
	temporalMaxDateField = int64(1)<<32 - 1
	// Integer values in JS are exact up to 2^53-1.
	temporalMaxSafeInteger = int64(1)<<53 - 1
)

// temporalDurationObject stores the ten Duration fields as float64 integers,
// mirroring the spec's use of mathematical values backed by IEEE doubles.
type temporalDurationObject struct {
	baseObject
	years, months, weeks, days        float64
	hours, minutes, seconds           float64
	milliseconds, microseconds, nanos float64
}

type temporalDurationField struct {
	name  string
	index int
}

// PrepareTemporalFields visits duration fields in alphabetical order.
var temporalDurationFields = []temporalDurationField{
	{"days", 3},
	{"hours", 4},
	{"microseconds", 8},
	{"milliseconds", 7},
	{"minutes", 5},
	{"months", 1},
	{"nanoseconds", 9},
	{"seconds", 6},
	{"weeks", 2},
	{"years", 0},
}

func (r *Runtime) newTemporalStub(name string) *Object {
	proto := r.newBaseObject(r.global.ObjectPrototype, classObject)
	f := &Object{runtime: r}
	r.newNativeFuncAndConstruct(f, func(call FunctionCall) Value {
		panic(r.NewTypeError("Temporal." + name + " is not implemented yet"))
	}, func(args []Value, newTarget *Object) *Object {
		panic(r.NewTypeError("Temporal." + name + " is not implemented yet"))
	}, proto.val, unistring.NewFromString(name), intToValue(0))
	f.self.(*nativeFuncObject)._putProp("from", r.newNativeFunc(func(call FunctionCall) Value {
		panic(r.NewTypeError("Temporal." + name + ".from is not implemented yet"))
	}, unistring.NewFromString("from"), 1), true, false, true)
	return f
}

func (r *Runtime) getTemporalDuration() *Object {
	ret := r.global.TemporalDuration
	if ret == nil {
		ret = &Object{runtime: r}
		r.global.TemporalDuration = ret
		r.newNativeFuncAndConstruct(ret, func(call FunctionCall) Value {
			panic(r.NewTypeError("Constructor Temporal.Duration requires 'new'"))
		}, func(args []Value, newTarget *Object) *Object {
			fields := r.temporalDurationFieldsFromArgs(args)
			proto := r.getTemporalDurationPrototype()
			if newTarget != nil {
				proto = r.getPrototypeFromCtor(newTarget, ret, proto)
			}
			return r.newTemporalDurationObject(fields, proto)
		}, r.getTemporalDurationPrototype(), "Duration", intToValue(0))
		ctor := ret.self.(*nativeFuncObject)
		ctor._putProp("from", r.newNativeFunc(r.temporalDurationFrom, "from", 1), true, false, true)
		ctor._putProp("compare", r.newNativeFunc(r.temporalDurationCompare, "compare", 2), true, false, true)
	}
	return ret
}

func (r *Runtime) getTemporalDurationPrototype() *Object {
	ret := r.global.TemporalDurationPrototype
	if ret == nil {
		o := r.newBaseObject(r.global.ObjectPrototype, classObject)
		ret = o.val
		r.global.TemporalDurationPrototype = ret

		o._putProp("constructor", r.getTemporalDuration(), true, false, true)

		addGetter := func(name string, f func(*temporalDurationObject) Value) {
			o.setOwnStr(unistring.NewFromString(name), &valueProperty{
				getterFunc: r.newNativeFunc(func(call FunctionCall) Value {
					return f(r.toTemporalDuration(call.This))
				}, unistring.NewFromString("get "+name), 0),
				accessor:     true,
				enumerable:   false,
				configurable: true,
			}, true)
		}
		addGetter("years", func(d *temporalDurationObject) Value { return r.ToValue(d.years) })
		addGetter("months", func(d *temporalDurationObject) Value { return r.ToValue(d.months) })
		addGetter("weeks", func(d *temporalDurationObject) Value { return r.ToValue(d.weeks) })
		addGetter("days", func(d *temporalDurationObject) Value { return r.ToValue(d.days) })
		addGetter("hours", func(d *temporalDurationObject) Value { return r.ToValue(d.hours) })
		addGetter("minutes", func(d *temporalDurationObject) Value { return r.ToValue(d.minutes) })
		addGetter("seconds", func(d *temporalDurationObject) Value { return r.ToValue(d.seconds) })
		addGetter("milliseconds", func(d *temporalDurationObject) Value { return r.ToValue(d.milliseconds) })
		addGetter("microseconds", func(d *temporalDurationObject) Value { return r.ToValue(d.microseconds) })
		addGetter("nanoseconds", func(d *temporalDurationObject) Value { return r.ToValue(d.nanos) })
		addGetter("sign", func(d *temporalDurationObject) Value { return intToValue(d.sign()) })
		addGetter("blank", func(d *temporalDurationObject) Value { return r.ToValue(d.sign() == 0) })

		o._putProp("with", r.newNativeFunc(r.temporalDurationWith, "with", 1), true, false, true)
		o._putProp("negated", r.newNativeFunc(r.temporalDurationNegated, "negated", 0), true, false, true)
		o._putProp("abs", r.newNativeFunc(r.temporalDurationAbs, "abs", 0), true, false, true)
		o._putProp("add", r.newNativeFunc(r.temporalDurationAdd, "add", 1), true, false, true)
		o._putProp("subtract", r.newNativeFunc(r.temporalDurationSubtract, "subtract", 1), true, false, true)
		o._putProp("round", r.newNativeFunc(r.temporalDurationRound, "round", 1), true, false, true)
		o._putProp("total", r.newNativeFunc(r.temporalDurationTotal, "total", 1), true, false, true)
		o._putProp("toString", r.newNativeFunc(r.temporalDurationToStringMethod, "toString", 0), true, false, true)
		o._putProp("toJSON", r.newNativeFunc(r.temporalDurationToJSON, "toJSON", 0), true, false, true)
		o._putProp("toLocaleString", r.newNativeFunc(r.temporalDurationToLocaleString, "toLocaleString", 0), true, false, true)
		o._putProp("valueOf", r.newNativeFunc(r.temporalDurationValueOf, "valueOf", 0), true, false, true)

		o._putSym(SymToStringTag, &valueProperty{
			value:        asciiString("Temporal.Duration"),
			writable:     false,
			enumerable:   false,
			configurable: true,
		})
	}
	return ret
}

func (r *Runtime) newTemporalDurationObject(fields [10]float64, proto *Object) *Object {
	v := &Object{runtime: r}
	for i, f := range fields {
		if f == 0 {
			fields[i] = 0 // normalise -0
		}
	}
	d := &temporalDurationObject{
		years: fields[0], months: fields[1], weeks: fields[2], days: fields[3],
		hours: fields[4], minutes: fields[5], seconds: fields[6],
		milliseconds: fields[7], microseconds: fields[8], nanos: fields[9],
	}
	d.baseObject = baseObject{
		class:      classObject,
		val:        v,
		extensible: true,
		prototype:  proto,
	}
	v.self = d
	d.init()
	return v
}

func (r *Runtime) toTemporalDuration(v Value) *temporalDurationObject {
	o := r.toObject(v)
	if d, ok := o.self.(*temporalDurationObject); ok {
		return d
	}
	panic(r.NewTypeError("Value is not a Temporal.Duration"))
}

func (d *temporalDurationObject) sign() int64 {
	for _, f := range d.fields() {
		if f > 0 {
			return 1
		}
		if f < 0 {
			return -1
		}
	}
	return 0
}

func (d *temporalDurationObject) fields() [10]float64 {
	return [10]float64{
		d.years, d.months, d.weeks, d.days,
		d.hours, d.minutes, d.seconds,
		d.milliseconds, d.microseconds, d.nanos,
	}
}

func (d *temporalDurationObject) hasCalendarUnits() bool {
	return d.years != 0 || d.months != 0 || d.weeks != 0
}

func (r *Runtime) temporalDurationFieldsFromArgs(args []Value) [10]float64 {
	var fields [10]float64
	for i := 0; i < 10; i++ {
		var v Value
		if i < len(args) {
			v = args[i]
		}
		fields[i] = r.temporalToIntegerIfIntegral(v)
	}
	r.validateTemporalDuration(fields)
	return fields
}

// temporalToIntegerIfIntegral implements ToIntegerIfIntegral().
func (r *Runtime) temporalToIntegerIfIntegral(v Value) float64 {
	if v == nil || IsUndefined(v) {
		return 0
	}
	n := v.ToNumber().ToFloat()
	if math.IsNaN(n) || math.IsInf(n, 0) || n != math.Trunc(n) {
		panic(r.newError(r.getRangeError(), "Invalid duration value"))
	}
	return n
}

func temporalAbsFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// validateTemporalDuration implements IsValidDuration.
func (r *Runtime) validateTemporalDuration(fields [10]float64) {
	sign := 0
	for _, f := range fields {
		if math.IsNaN(f) || math.IsInf(f, 0) || math.Trunc(f) != f {
			panic(r.newError(r.getRangeError(), "Duration fields must be integers"))
		}
		if f == 0 {
			continue
		}
		s := 1
		if f < 0 {
			s = -1
		}
		if sign != 0 && sign != s {
			panic(r.newError(r.getRangeError(), "Duration fields must have the same sign"))
		}
		sign = s
	}
	if temporalAbsFloat(fields[0]) > float64(temporalMaxDateField) ||
		temporalAbsFloat(fields[1]) > float64(temporalMaxDateField) ||
		temporalAbsFloat(fields[2]) > float64(temporalMaxDateField) {
		panic(r.newError(r.getRangeError(), "Duration date field is out of range"))
	}

	total := new(big.Rat)
	add := func(v float64, mul int64) {
		if v != 0 {
			total.Add(total, new(big.Rat).Mul(new(big.Rat).SetFloat64(v), big.NewRat(mul, 1)))
		}
	}
	add(fields[3], 86400*1e9)
	add(fields[4], 3600*1e9)
	add(fields[5], 60*1e9)
	add(fields[6], 1e9)
	add(fields[7], 1e6)
	add(fields[8], 1e3)
	add(fields[9], 1)

	limitInt := new(big.Int).Mul(big.NewInt(temporalMaxSafeInteger), big.NewInt(1e9))
	limitInt.Add(limitInt, big.NewInt(999999999))
	limit := new(big.Rat).SetInt(limitInt)
	abs := new(big.Rat).Abs(total)
	if abs.Cmp(limit) > 0 {
		panic(r.newError(r.getRangeError(), "Duration is out of range"))
	}
}

func (r *Runtime) temporalDurationFromValue(v Value) *temporalDurationObject {
	if o, ok := v.(*Object); ok {
		if d, ok := o.self.(*temporalDurationObject); ok {
			return r.durationFieldsObject(d.fields())
		}
	}
	if s, ok := v.(String); ok {
		return r.temporalDurationFromString(s.String())
	}
	obj, ok := v.(*Object)
	if !ok {
		panic(r.NewTypeError("Cannot convert %s to Temporal.Duration", v))
	}
	var fields [10]float64
	found := false
	for _, f := range temporalDurationFields {
		fv := obj.self.getStr(unistring.NewFromString(f.name), nil)
		if fv == nil || IsUndefined(fv) {
			continue
		}
		found = true
		fields[f.index] = r.temporalToIntegerIfIntegral(fv)
	}
	if !found {
		panic(r.NewTypeError("Duration-like object must have at least one duration property"))
	}
	r.validateTemporalDuration(fields)
	return r.durationFieldsObject(fields)
}

func (r *Runtime) durationFieldsObject(fields [10]float64) *temporalDurationObject {
	o := r.newTemporalDurationObject(fields, r.getTemporalDurationPrototype())
	return o.self.(*temporalDurationObject)
}

func (r *Runtime) temporalDurationFrom(call FunctionCall) Value {
	d := r.temporalDurationFromValue(call.Argument(0))
	return r.newTemporalDurationObject(d.fields(), r.getTemporalDurationPrototype())
}

// --- methods -------------------------------------------------------------

func (r *Runtime) temporalDurationWith(call FunctionCall) Value {
	d := r.toTemporalDuration(call.This)
	partial := r.toObject(call.Argument(0))
	fields := d.fields()
	found := false
	for _, f := range temporalDurationFields {
		v := partial.self.getStr(unistring.NewFromString(f.name), nil)
		if v == nil || IsUndefined(v) {
			continue
		}
		found = true
		fields[f.index] = r.temporalToIntegerIfIntegral(v)
	}
	if !found {
		panic(r.NewTypeError("Duration.with requires at least one duration property"))
	}
	r.validateTemporalDuration(fields)
	return r.newTemporalDurationObject(fields, r.getTemporalDurationPrototype())
}

func (r *Runtime) temporalDurationNegated(call FunctionCall) Value {
	d := r.toTemporalDuration(call.This)
	fields := d.fields()
	for i := range fields {
		fields[i] = -fields[i]
	}
	return r.newTemporalDurationObject(fields, r.getTemporalDurationPrototype())
}

func (r *Runtime) temporalDurationAbs(call FunctionCall) Value {
	d := r.toTemporalDuration(call.This)
	fields := d.fields()
	for i := range fields {
		fields[i] = temporalAbsFloat(fields[i])
	}
	return r.newTemporalDurationObject(fields, r.getTemporalDurationPrototype())
}

func (r *Runtime) temporalDurationAdd(call FunctionCall) Value {
	return r.temporalDurationAddSub(call, false)
}

func (r *Runtime) temporalDurationSubtract(call FunctionCall) Value {
	return r.temporalDurationAddSub(call, true)
}

func (r *Runtime) temporalDurationAddSub(call FunctionCall, subtract bool) Value {
	d := r.toTemporalDuration(call.This)
	other := r.temporalDurationFromValue(call.Argument(0))
	if d.hasCalendarUnits() || other.hasCalendarUnits() {
		panic(r.newError(r.getRangeError(), "relativeTo is required for calendar units"))
	}
	a := d.fields()
	b := other.fields()
	sign := float64(1)
	if subtract {
		sign = -1
	}
	// Largest unit of either input (day or smaller).
	largest := 9
	for i := 3; i <= 9; i++ {
		if a[i] != 0 || b[i] != 0 {
			largest = i
			break
		}
	}
	total := new(big.Rat)
	for i := 3; i <= 9; i++ {
		component := new(big.Rat).SetFloat64(a[i])
		if b[i] != 0 {
			br := new(big.Rat).SetFloat64(b[i])
			if sign < 0 {
				br.Neg(br)
			}
			component.Add(component, br)
		}
		if component.Sign() != 0 {
			total.Add(total, new(big.Rat).Mul(component, big.NewRat(temporalUnitNanosIndex(i), 1)))
		}
	}
	var fields [10]float64
	if total.Sign() != 0 {
		rem := new(big.Rat).Set(total)
		for i := largest; i <= 9; i++ {
			unit := big.NewRat(temporalUnitNanosIndex(i), 1)
			q := new(big.Rat).Quo(rem, unit)
			qInt := new(big.Int).Quo(q.Num(), q.Denom())
			f, _ := new(big.Float).SetInt(qInt).Float64()
			fields[i] = f
			rem.Sub(rem, new(big.Rat).Mul(new(big.Rat).SetInt(qInt), unit))
		}
	}
	r.validateTemporalDuration(fields)
	return r.newTemporalDurationObject(fields, r.getTemporalDurationPrototype())
}

func temporalUnitNanosIndex(i int) int64 {
	switch i {
	case 3:
		return 86400 * 1e9
	case 4:
		return 3600 * 1e9
	case 5:
		return 60 * 1e9
	case 6:
		return 1e9
	case 7:
		return 1e6
	case 8:
		return 1e3
	default:
		return 1
	}
}

func (r *Runtime) temporalDurationRound(call FunctionCall) Value {
	d := r.toTemporalDuration(call.This)
	opts := call.Argument(0)
	var smallest, largest string
	increment := int64(1)
	mode := temporalRoundHalfExpand
	hasSmallest := false
	hasLargest := false
	var rel *temporalRelativeTo
	if opts == nil || IsUndefined(opts) || IsNull(opts) {
		panic(r.NewTypeError("options are required"))
	}
	if s, ok := opts.(String); ok {
		u, valid := normalizeTemporalUnit(s.String())
		if !valid {
			panic(r.newError(r.getRangeError(), "Invalid smallestUnit"))
		}
		smallest, hasSmallest = u, true
	} else {
		obj, ok := opts.(*Object)
		if !ok {
			panic(r.NewTypeError("options must be an object"))
		}
		// Spec order: largestUnit, relativeTo, roundingIncrement,
		// roundingMode, smallestUnit.
		if v := obj.self.getStr("largestUnit", nil); v != nil && !IsUndefined(v) {
			str, _ := temporalToStringOption(r, v)
			hasLargest = true
			if str != "auto" {
				u, valid := normalizeTemporalUnit(str)
				if !valid {
					panic(r.newError(r.getRangeError(), "Invalid largestUnit"))
				}
				largest = u
			}
		}
		if v := obj.self.getStr("relativeTo", nil); v != nil && !IsUndefined(v) {
			rel = r.temporalRelativeToFromValue(v, opts)
		}
		increment = r.temporalGetRoundingIncrementOption(obj)
		mode = r.parseTemporalRoundingMode(obj.self.getStr("roundingMode", nil), temporalRoundHalfExpand)
		if v := obj.self.getStr("smallestUnit", nil); v != nil && !IsUndefined(v) {
			str, _ := temporalToStringOption(r, v)
			u, valid := normalizeTemporalUnit(str)
			if !valid {
				panic(r.newError(r.getRangeError(), "Invalid smallestUnit"))
			}
			smallest, hasSmallest = u, true
		}
	}
	if !hasSmallest && !hasLargest && largest == "" {
		panic(r.newError(r.getRangeError(), "smallestUnit or largestUnit is required"))
	}
	if hasSmallest {
		if temporalUnitIndex(smallest) >= 4 {
			if err := validateTimeRoundingIncrement(smallest, increment); err != nil {
				panic(r.newError(r.getRangeError(), err.Error()))
			}
		}
		if largest != "" && temporalUnitIndex(smallest) < temporalUnitIndex(largest) {
			panic(r.newError(r.getRangeError(), "smallestUnit is larger than largestUnit"))
		}
	}
	if rel != nil {
		return r.roundDurationRelativeTo(d, rel, largest, smallest, increment, mode, hasSmallest)
	}
	if d.hasCalendarUnits() {
		panic(r.newError(r.getRangeError(), "relativeTo is required for calendar units"))
	}
	if (!hasSmallest || temporalUnitIndex(smallest) <= 2) && largest != "" && temporalUnitIndex(largest) <= 2 {
		panic(r.newError(r.getRangeError(), "relativeTo is required for calendar units"))
	}
	if hasSmallest && temporalUnitIndex(smallest) <= 2 {
		panic(r.newError(r.getRangeError(), "relativeTo is required for calendar units"))
	}

	// Determine the default largest unit: the largest unit present in the
	// input, but at least the smallest unit being rounded to.
	if largest == "" {
		largest = "nanosecond"
		for i := 3; i <= 9; i++ {
			if d.fields()[i] != 0 {
				largest = temporalUnitName(i)
				break
			}
		}
		if hasSmallest && temporalUnitIndex(smallest) < temporalUnitIndex(largest) {
			largest = smallest
		}
	}

	total := durationTimeNanos(d)
	negative := total.Sign() < 0
	abs := new(big.Rat).Abs(total)
	if hasSmallest {
		unitNs := big.NewRat(temporalUnitNanos(smallest), 1)
		inc := new(big.Rat).Mul(unitNs, big.NewRat(increment, 1))
		q := new(big.Rat).Quo(abs, inc)
		qInt := new(big.Int).Quo(q.Num(), q.Denom())
		rem := new(big.Rat).Sub(q, new(big.Rat).SetInt(qInt))
		up := false
		if rem.Sign() != 0 {
			switch mode {
			case temporalRoundExpand:
				up = true
			case temporalRoundCeil:
				up = !negative
			case temporalRoundFloor:
				up = negative
			case temporalRoundHalfCeil, temporalRoundHalfFloor, temporalRoundHalfExpand, temporalRoundHalfTrunc, temporalRoundHalfEven:
				cmp := rem.Cmp(big.NewRat(1, 2))
				if cmp > 0 {
					up = true
				} else if cmp == 0 {
					switch mode {
					case temporalRoundHalfCeil:
						up = !negative
					case temporalRoundHalfFloor:
						up = negative
					case temporalRoundHalfTrunc:
						up = false
					case temporalRoundHalfEven:
						up = qInt.Bit(0) == 1
					default:
						up = true
					}
				}
			}
		}
		if up {
			qInt.Add(qInt, big.NewInt(1))
		}
		abs = new(big.Rat).Mul(new(big.Rat).SetInt(qInt), inc)
	}

	fields := [10]float64{}
	rem := abs
	for i := temporalUnitIndex(largest); i <= 9 && rem.Sign() != 0; i++ {
		unit := big.NewRat(temporalUnitNanosIndex(i), 1)
		q := new(big.Rat).Quo(rem, unit)
		qInt := new(big.Int).Quo(q.Num(), q.Denom())
		f, _ := new(big.Float).SetInt(qInt).Float64()
		if negative {
			f = -f
		}
		fields[i] = f
		rem.Sub(rem, new(big.Rat).Mul(new(big.Rat).SetInt(qInt), unit))
	}
	r.validateTemporalDuration(fields)
	return r.newTemporalDurationObject(fields, r.getTemporalDurationPrototype())
}

// roundRatToInt rounds a rational number to an integer with a Temporal
// rounding mode.
func roundRatToInt(r *big.Rat, mode temporalRoundingMode) *big.Int {
	q := new(big.Int).Quo(r.Num(), r.Denom())
	rem := new(big.Rat).Sub(r, new(big.Rat).SetInt(q))
	if rem.Sign() == 0 {
		return q
	}
	neg := r.Sign() < 0
	absRem := new(big.Rat).Abs(rem)
	twice := new(big.Rat).Mul(absRem, big.NewRat(2, 1))
	cmp := twice.Cmp(big.NewRat(1, 1))
	up := false
	switch mode {
	case temporalRoundTrunc:
		up = false
	case temporalRoundExpand:
		up = true
	case temporalRoundCeil:
		up = !neg
	case temporalRoundFloor:
		up = neg
	default:
		if cmp > 0 {
			up = true
		} else if cmp < 0 {
			up = false
		} else {
			absQ := new(big.Int).Abs(q)
			switch mode {
			case temporalRoundHalfCeil:
				up = !neg
			case temporalRoundHalfFloor:
				up = neg
			case temporalRoundHalfTrunc:
				up = false
			case temporalRoundHalfEven:
				up = absQ.Bit(0) == 1
			default:
				up = true
			}
		}
	}
	if up {
		if neg {
			q.Sub(q, big1)
		} else {
			q.Add(q, big1)
		}
	}
	return q
}

// roundDurationRelativeTo rounds a duration using the ISO date-time
// anchored at the start of relativeTo's day. This handles calendar units and
// time-of-day fractions uniformly.
func (r *Runtime) roundDurationRelativeTo(d *temporalDurationObject, rel *temporalRelativeTo, largest, smallest string, increment int64, mode temporalRoundingMode, hasSmallest bool) Value {
	fields := d.fields()
	zero := true
	for _, f := range fields {
		if f != 0 {
			zero = false
			break
		}
	}
	if zero {
		if rel.zone != nil && (largest == "day" || (hasSmallest && smallest == "day")) {
			local := new(big.Int).Mul(big.NewInt(rel.date.days()), big.NewInt(86400e9))
			local.Add(local, big.NewInt(rel.time.nanos()))
			startNs := rel.zone.instantFromLocal(local)
			if !r.validInstant(new(big.Int).Add(startNs, big.NewInt(86400e9))) {
				panic(r.newError(r.getRangeError(), "Next day boundary is out of range"))
			}
		}
		return r.newTemporalDurationObject([10]float64{}, r.getTemporalDurationPrototype())
	}
	if !rel.date.isValidDateTime(rel.time) {
		panic(r.newError(r.getRangeError(), "relativeTo is out of range"))
	}
	if rel.zone == nil {
		local := new(big.Int).Mul(big.NewInt(rel.date.days()), big.NewInt(86400e9))
		local.Add(local, big.NewInt(rel.time.nanos()))
		if local.Cmp(new(big.Int).Neg(temporalMaxInstantNs)) < 0 {
			panic(r.newError(r.getRangeError(), "relativeTo is out of range"))
		}
	}
	startDate, startTime := rel.date, rel.time
	date := addISOMonths(rel.date, int64(fields[0])*12+int64(fields[1]))
	dayNanos := big.NewInt(86400e9)
	timeNs := new(big.Int).SetInt64(startTime.nanos())
	addRat := func(v float64, mul int64) {
		if v != 0 {
			x := new(big.Rat).Mul(new(big.Rat).SetFloat64(v), big.NewRat(mul, 1))
			timeNs.Add(timeNs, new(big.Int).Quo(x.Num(), x.Denom()))
		}
	}
	addRat(fields[2], 7*86400*1e9)
	addRat(fields[3], 86400*1e9)
	addRat(fields[4], 3600*1e9)
	addRat(fields[5], 60*1e9)
	addRat(fields[6], 1e9)
	addRat(fields[7], 1e6)
	addRat(fields[8], 1e3)
	addRat(fields[9], 1)
	carry := new(big.Int)
	rem := new(big.Int)
	carry.DivMod(new(big.Int).Set(timeNs), dayNanos, rem)
	date = addISODays(date, carry.Int64())
	endTime := isoTimeFromNanos(rem.Int64())
	if rel.zone != nil {
		localStart := new(big.Int).Mul(big.NewInt(rel.date.days()), dayNanos)
		localStart.Add(localStart, big.NewInt(rel.time.nanos()))
		startNs := rel.zone.instantFromLocal(localStart)
		if hasSmallest && temporalUnitIndex(smallest) <= 3 {
			nextDay := new(big.Int).Add(startNs, big.NewInt(86400e9))
			if !r.validInstant(nextDay) {
				panic(r.newError(r.getRangeError(), "Next day boundary is out of range"))
			}
		}
		endNs := r.zonedAdd(startNs, rel.zone, fields, 1)
		startDate, startTime, _ = rel.zone.localDateTime(startNs)
		date, endTime, _ = rel.zone.localDateTime(endNs)
	} else {
		startDate, startTime = rel.date, rel.time
	}
	if !date.isValidDateTime(endTime) {
		panic(r.newError(r.getRangeError(), "Resulting date-time is out of range"))
	}

	if largest == "" {
		largest = "nanosecond"
		for i := 0; i <= 3; i++ {
			if fields[i] != 0 {
				largest = temporalUnitName(i)
				break
			}
		}
		if largest == "nanosecond" {
			for i := 4; i <= 9; i++ {
				if fields[i] != 0 {
					largest = temporalUnitName(i)
					break
				}
			}
		}
		if hasSmallest && temporalUnitIndex(smallest) < temporalUnitIndex(largest) {
			largest = smallest
		}
	}
	if hasSmallest && increment > 1 && temporalUnitIndex(smallest) <= 3 && largest != "" &&
		temporalUnitIndex(largest) < temporalUnitIndex(smallest) {
		panic(r.newError(r.getRangeError(), "Cannot round to an increment while balancing to a larger calendar unit"))
	}
	if hasSmallest && (smallest == "week" || smallest == "day") {
		dateDiff := date.days() - startDate.days()
		timeDiff := endTime.nanos() - startTime.nanos()
		if dateDiff > 0 && timeDiff < 0 {
			dateDiff--
			timeDiff += 86400e9
		} else if dateDiff < 0 && timeDiff > 0 {
			dateDiff++
			timeDiff -= 86400e9
		}
		df := r.temporalDateFieldsFromDays(startDate, dateDiff, largest)
		unitDays := int64(1)
		if smallest == "week" {
			unitDays = 7
		}
		days := df[3] + df[2]*7
		num := new(big.Rat).SetInt64(int64(days))
		num.Add(num, big.NewRat(timeDiff, 86400e9))
		scale := big.NewRat(unitDays*increment, 1)
		value := new(big.Rat).Quo(num, scale)
		q := roundRatToInt(value, mode)
		amount := new(big.Int).Mul(q, big.NewInt(increment))
		amountDays := new(big.Int).Mul(amount, big.NewInt(unitDays))
		out := [10]float64{}
		out[0] = df[0]
		out[1] = df[1]
		switch temporalUnitIndex(largest) {
		case 2: // week
			w := new(big.Int).Quo(amountDays, big.NewInt(7))
			d := new(big.Int).Rem(amountDays, big.NewInt(7))
			out[2] = float64(w.Int64())
			out[3] = float64(d.Int64())
		case 3: // day
			out[3] = float64(amountDays.Int64())
		default:
			if smallest == "week" {
				out[2] = float64(amount.Int64())
			} else {
				out[3] = float64(amount.Int64())
			}
		}
		r.validateTemporalDuration(out)
		return r.newTemporalDurationObject(out, r.getTemporalDurationPrototype())
	}
	if rel.zone == nil && largest == "day" && hasSmallest && temporalUnitIndex(smallest) >= 4 {
		dayNs := big.NewInt(86400e9)
		startTotal := new(big.Int).Mul(big.NewInt(startDate.days()), dayNs)
		startTotal.Add(startTotal, big.NewInt(startTime.nanos()))
		endTotal := new(big.Int).Mul(big.NewInt(date.days()), dayNs)
		endTotal.Add(endTotal, big.NewInt(endTime.nanos()))
		total := new(big.Int).Sub(endTotal, startTotal)
		unit := new(big.Int).Mul(big.NewInt(temporalUnitNanos(smallest)), big.NewInt(increment))
		total = temporalRoundNanos(total, unit, mode)
		days, rem := new(big.Int).QuoRem(total, dayNs, new(big.Int))
		out := [10]float64{}
		out[3] = float64(days.Int64())
		temporalTimeFieldsFromNanos(rem, 4, &out)
		r.validateTemporalDuration(out)
		return r.newTemporalDurationObject(out, r.getTemporalDurationPrototype())
	}
	s := temporalDiffSettings{largest: largest, smallest: "nanosecond", increment: 1, mode: temporalRoundTrunc}
	if hasSmallest {
		s.smallest = smallest
		s.increment = increment
		s.mode = mode
	}
	out := r.temporalDiffISODateTime(startDate, startTime, date, endTime, s)
	r.validateTemporalDuration(out)
	return r.newTemporalDurationObject(out, r.getTemporalDurationPrototype())
}

func (r *Runtime) roundCalendarDuration(d *temporalDurationObject, rel *temporalPlainDateObject, smallest, largest string, increment int64, mode temporalRoundingMode, hasSmallest bool) Value {
	fields := d.fields()
	end := rel.date
	end = addISOMonths(end, int64(fields[0])*12+int64(fields[1]))
	end = addISODays(end, int64(fields[2])*7+int64(fields[3]))
	// Fold whole time days into the end date; sub-day time is not representable
	// in a date-only round and is dropped (matching the date component).
	end = addISODays(end, int64(fields[4]/24+fields[5]/1440+fields[6]/86400))
	endObj := r.plainDateFieldsObject(end, rel.calendar)
	if largest == "" {
		largest = "day"
		for i := 0; i <= 3; i++ {
			if fields[i] != 0 {
				largest = temporalUnitName(i)
				break
			}
		}
		if hasSmallest && temporalUnitIndex(smallest) < temporalUnitIndex(largest) {
			largest = smallest
		}
	}
	opts := r.NewObject()
	opts.Set("largestUnit", r.ToValue(largest))
	if hasSmallest {
		opts.Set("smallestUnit", r.ToValue(smallest))
	}
	opts.Set("roundingIncrement", r.ToValue(increment))
	switch mode {
	case temporalRoundCeil:
		opts.Set("roundingMode", r.ToValue("ceil"))
	case temporalRoundFloor:
		opts.Set("roundingMode", r.ToValue("floor"))
	case temporalRoundExpand:
		opts.Set("roundingMode", r.ToValue("expand"))
	case temporalRoundTrunc:
		opts.Set("roundingMode", r.ToValue("trunc"))
	case temporalRoundHalfCeil:
		opts.Set("roundingMode", r.ToValue("halfCeil"))
	case temporalRoundHalfFloor:
		opts.Set("roundingMode", r.ToValue("halfFloor"))
	case temporalRoundHalfTrunc:
		opts.Set("roundingMode", r.ToValue("halfTrunc"))
	case temporalRoundHalfEven:
		opts.Set("roundingMode", r.ToValue("halfEven"))
	default:
		opts.Set("roundingMode", r.ToValue("halfExpand"))
	}
	return r.temporalPlainDateDifference(rel, endObj, opts, false)
}

// normalizeTemporalUnit validates a unit name and returns its canonical
// singular form.
func normalizeTemporalUnit(unit string) (string, bool) {
	switch unit {
	case "year", "years":
		return "year", true
	case "month", "months":
		return "month", true
	case "week", "weeks":
		return "week", true
	case "day", "days":
		return "day", true
	case "hour", "hours":
		return "hour", true
	case "minute", "minutes":
		return "minute", true
	case "second", "seconds":
		return "second", true
	case "millisecond", "milliseconds":
		return "millisecond", true
	case "microsecond", "microseconds":
		return "microsecond", true
	case "nanosecond", "nanoseconds":
		return "nanosecond", true
	}
	return unit, false
}

func temporalUnitIndex(unit string) int {
	switch unit {
	case "year":
		return 0
	case "month":
		return 1
	case "week":
		return 2
	case "day":
		return 3
	case "hour":
		return 4
	case "minute":
		return 5
	case "second":
		return 6
	case "millisecond":
		return 7
	case "microsecond":
		return 8
	case "nanosecond":
		return 9
	}
	return -1
}

func temporalUnitName(index int) string {
	switch index {
	case 0:
		return "year"
	case 1:
		return "month"
	case 2:
		return "week"
	case 3:
		return "day"
	case 4:
		return "hour"
	case 5:
		return "minute"
	case 6:
		return "second"
	case 7:
		return "millisecond"
	case 8:
		return "microsecond"
	default:
		return "nanosecond"
	}
}

func (r *Runtime) temporalDurationTotal(call FunctionCall) Value {
	d := r.toTemporalDuration(call.This)
	arg := call.Argument(0)
	var unit string
	var rel *temporalRelativeTo
	if s, ok := arg.(String); ok {
		u, valid := normalizeTemporalUnit(s.String())
		if !valid {
			panic(r.newError(r.getRangeError(), "Invalid unit"))
		}
		unit = u
	} else {
		obj := r.temporalOptionsObject(arg)
		if obj == nil {
			panic(r.NewTypeError("options are required"))
		}
		// Spec order: relativeTo, then unit.
		if v := obj.self.getStr("relativeTo", nil); v != nil && !IsUndefined(v) {
			rel = r.temporalRelativeToFromValue(v, arg)
		}
		uv := obj.self.getStr("unit", nil)
		if uv == nil || IsUndefined(uv) {
			panic(r.newError(r.getRangeError(), "unit is required"))
		}
		str, _ := temporalToStringOption(r, uv)
		u, valid := normalizeTemporalUnit(str)
		if !valid {
			panic(r.newError(r.getRangeError(), "Invalid unit"))
		}
		unit = u
	}
	if d.hasCalendarUnits() && rel == nil {
		panic(r.newError(r.getRangeError(), "relativeTo is required for calendar units"))
	}
	if rel == nil && temporalUnitIndex(unit) <= 2 {
		panic(r.newError(r.getRangeError(), "relativeTo is required for calendar units"))
	}
	if rel == nil {
		total := durationTimeNanos(d)
		unitNanos := temporalUnitNanos(unit)
		if unitNanos == 0 {
			panic(r.NewTypeError("Invalid unit"))
		}
		rat := new(big.Rat).Quo(total, big.NewRat(unitNanos, 1))
		f, _ := rat.Float64()
		return r.ToValue(f)
	}
	f := r.durationTotalRelative(d, rel, unit)
	return r.ToValue(f)
}

// durationTotalRelative computes Duration.total(unit) in the presence of a
// relativeTo date.
func (r *Runtime) durationTotalRelativeRat(d *temporalDurationObject, rel *temporalRelativeTo, unit string, checkDayBoundary bool) *big.Rat {
	fields := d.fields()
	zero := true
	for _, f := range fields {
		if f != 0 {
			zero = false
			break
		}
	}
	if !zero && !rel.date.isValidDateTime(rel.time) {
		panic(r.newError(r.getRangeError(), "relativeTo is out of range"))
	}
	if !zero && rel.zone == nil {
		local := new(big.Int).Mul(big.NewInt(rel.date.days()), big.NewInt(86400e9))
		local.Add(local, big.NewInt(rel.time.nanos()))
		if local.Cmp(new(big.Int).Neg(temporalMaxInstantNs)) < 0 {
			panic(r.newError(r.getRangeError(), "relativeTo is out of range"))
		}
	}
	dayNs := big.NewInt(86400 * 1e9)

	// Determine the wall-clock start and end, and (for zoned relative-to) the
	// exact elapsed nanoseconds.
	startDate, startTime := rel.date, rel.time
	var endDate isoDate
	var endTime isoTime
	var exactNs *big.Int
	if rel.zone != nil {
		localStart := new(big.Int).Mul(big.NewInt(startDate.days()), dayNs)
		localStart.Add(localStart, big.NewInt(startTime.nanos()))
		startNs := rel.zone.instantFromLocal(localStart)
		if checkDayBoundary && (unit == "day" || temporalUnitIndex(unit) <= 3) {
			nextDay := new(big.Int).Add(startNs, dayNs)
			if !r.validInstant(nextDay) {
				panic(r.newError(r.getRangeError(), "Next day boundary is out of range"))
			}
		}
		endNs := r.zonedAdd(startNs, rel.zone, fields, 1)
		exactNs = new(big.Int).Sub(endNs, startNs)
		startDate, startTime, _ = rel.zone.localDateTime(startNs)
		endDate, endTime, _ = rel.zone.localDateTime(endNs)
	} else {
		endDate = addISOMonths(startDate, int64(fields[0])*12+int64(fields[1]))
		timeNs := new(big.Int).SetInt64(startTime.nanos())
		add := func(v float64, mul int64) {
			if v != 0 {
				x := new(big.Rat).Mul(new(big.Rat).SetFloat64(v), big.NewRat(mul, 1))
				timeNs.Add(timeNs, new(big.Int).Quo(x.Num(), x.Denom()))
			}
		}
		add(fields[2], 7*86400*1e9)
		add(fields[3], 86400*1e9)
		add(fields[4], 3600*1e9)
		add(fields[5], 60*1e9)
		add(fields[6], 1e9)
		add(fields[7], 1e6)
		add(fields[8], 1e3)
		add(fields[9], 1)
		carry, rem := new(big.Int).DivMod(timeNs, dayNs, new(big.Int))
		endDate = addISODays(endDate, carry.Int64())
		endTime = isoTimeFromNanos(rem.Int64())
		if !endDate.isValid() {
			panic(r.newError(r.getRangeError(), "Resulting date is out of range"))
		}
	}

	// Signed wall-clock difference.
	dateDiff := endDate.days() - startDate.days()
	timeDiff := endTime.nanos() - startTime.nanos()
	if dateDiff > 0 && timeDiff < 0 {
		dateDiff--
		timeDiff += 86400 * 1e9
	} else if dateDiff < 0 && timeDiff > 0 {
		dateDiff++
		timeDiff -= 86400 * 1e9
	}
	totalWall := new(big.Rat).SetInt64(dateDiff)
	totalWall.Add(totalWall, new(big.Rat).Quo(big.NewRat(timeDiff, 1), big.NewRat(86400*1e9, 1)))

	switch unit {
	case "day":
		return totalWall
	case "week":
		return new(big.Rat).Quo(totalWall, big.NewRat(7, 1))
	case "hour", "minute", "second", "millisecond", "microsecond", "nanosecond":
		if rel.zone != nil {
			unitNs := big.NewInt(temporalUnitNanos(unit))
			return new(big.Rat).SetFrac(exactNs, unitNs)
		}
		factor := big.NewRat(86400*1e9, 1)
		factor.Quo(factor, big.NewRat(temporalUnitNanos(unit), 1))
		return new(big.Rat).Mul(totalWall, factor)
	}

	// Calendar units: whole months/years plus a fractional remainder based on
	// the length of the following (or, backwards, preceding) unit.
	start, end := startDate, endDate
	neg := totalWall.Sign() < 0
	timeFrac := new(big.Rat).Quo(big.NewRat(timeDiff, 1), big.NewRat(86400*1e9, 1))
	var result *big.Rat
	if !neg {
		totalMonths := (end.year-start.year)*12 + (end.month - start.month)
		mid := addISOMonths(start, totalMonths)
		for totalMonths > 0 && mid.days() > end.days() {
			totalMonths--
			mid = addISOMonths(start, totalMonths)
		}
		if totalMonths < 0 {
			totalMonths = 0
			mid = start
		}
		rem := new(big.Rat).SetInt64(end.days() - mid.days())
		rem.Add(rem, timeFrac)
		if unit == "month" {
			next := addISOMonths(start, totalMonths+1)
			if !next.isValid() {
				panic(r.newError(r.getRangeError(), "Resulting date is out of range"))
			}
			denom := next.days() - mid.days()
			result = new(big.Rat).SetInt64(totalMonths)
			result.Add(result, new(big.Rat).Quo(rem, big.NewRat(denom, 1)))
		} else {
			years := totalMonths / 12
			midY := addISOMonths(start, years*12)
			nextY := addISOMonths(midY, 12)
			if !nextY.isValid() {
				panic(r.newError(r.getRangeError(), "Resulting date is out of range"))
			}
			denom := nextY.days() - midY.days()
			remY := new(big.Rat).SetInt64(end.days() - midY.days())
			remY.Add(remY, timeFrac)
			result = new(big.Rat).SetInt64(years)
			result.Add(result, new(big.Rat).Quo(remY, big.NewRat(denom, 1)))
		}
	} else {
		// Backwards: anchor at the receiver and use the preceding unit length.
		mag := new(big.Rat).Abs(timeFrac)
		totalMonths := (start.year-end.year)*12 + (start.month - end.month)
		mid := addISOMonths(start, -totalMonths)
		for totalMonths > 0 && mid.days() < end.days() {
			totalMonths--
			mid = addISOMonths(start, -totalMonths)
		}
		if unit == "month" {
			prev := addISOMonths(mid, -1)
			if !prev.isValid() {
				panic(r.newError(r.getRangeError(), "Resulting date is out of range"))
			}
			denom := mid.days() - prev.days()
			rem := new(big.Rat).SetInt64(mid.days() - end.days())
			rem.Add(rem, mag)
			result = new(big.Rat).SetInt64(totalMonths)
			result.Add(result, new(big.Rat).Quo(rem, big.NewRat(denom, 1)))
			result.Neg(result)
		} else {
			years := totalMonths / 12
			midY := addISOMonths(start, -years*12)
			prevY := addISOMonths(midY, -12)
			if !prevY.isValid() {
				panic(r.newError(r.getRangeError(), "Resulting date is out of range"))
			}
			denom := midY.days() - prevY.days()
			remY := new(big.Rat).SetInt64(midY.days() - end.days())
			remY.Add(remY, mag)
			result = new(big.Rat).SetInt64(years)
			result.Add(result, new(big.Rat).Quo(remY, big.NewRat(denom, 1)))
			result.Neg(result)
		}
	}
	return result

}

func (r *Runtime) durationTotalRelative(d *temporalDurationObject, rel *temporalRelativeTo, unit string) float64 {
	f, _ := r.durationTotalRelativeRat(d, rel, unit, true).Float64()
	return f
}

func temporalDurationFieldsEqual(a, b *temporalDurationObject) bool {
	fa, fb := a.fields(), b.fields()
	for i := range fa {
		if fa[i] != fb[i] {
			return false
		}
	}
	return true
}

func timeRatHours(f [10]float64) float64 {
	return f[4] + f[5]/60 + f[6]/3600 + f[7]/3.6e6 + f[8]/3.6e9 + f[9]/3.6e12
}
func timeRatMinutes(f [10]float64) float64 {
	return f[4]*60 + f[5] + f[6]/60 + f[7]/6e4 + f[8]/6e7 + f[9]/6e10
}
func timeRatSeconds(f [10]float64) float64 {
	return f[4]*3600 + f[5]*60 + f[6] + f[7]/1e3 + f[8]/1e6 + f[9]/1e9
}

func addISODays(d isoDate, days int64) isoDate {
	z := d.days() + days
	y, m, day := isoCivilFromDays(z)
	return isoDate{year: y, month: m, day: day}
}

func (r *Runtime) temporalDurationUnit(v Value) (string, bool) {
	var raw string
	switch u := v.(type) {
	case String:
		raw = u.String()
	default:
		if v == nil || IsUndefined(v) || IsNull(v) {
			return "", false
		}
		obj, ok := v.(*Object)
		if !ok {
			return "", false
		}
		uv := obj.self.getStr("unit", nil)
		if uv == nil || IsUndefined(uv) {
			return "", false
		}
		s, ok := uv.(String)
		if !ok {
			return "", false
		}
		raw = s.String()
	}
	switch raw {
	case "year", "month", "week", "day", "hour", "minute", "second",
		"millisecond", "microsecond", "nanosecond":
		return raw, true
	case "years", "months", "weeks", "days", "hours", "minutes", "seconds",
		"milliseconds", "microseconds", "nanoseconds":
		return raw[:len(raw)-1], true
	}
	return "", false
}

func temporalUnitNanos(unit string) int64 {
	switch unit {
	case "day":
		return 86400 * 1e9
	case "hour":
		return 3600 * 1e9
	case "minute":
		return 60 * 1e9
	case "second":
		return 1e9
	case "millisecond":
		return 1e6
	case "microsecond":
		return 1e3
	case "nanosecond":
		return 1
	}
	return 0
}

func (r *Runtime) temporalDurationToStringMethod(call FunctionCall) Value {
	d := r.toTemporalDuration(call.This)
	return r.ToValue(r.formatTemporalDuration(d, call.Argument(0)))
}

func (r *Runtime) temporalDurationToJSON(call FunctionCall) Value {
	d := r.toTemporalDuration(call.This)
	return r.ToValue(r.formatTemporalDuration(d, _undefined))
}

func (r *Runtime) temporalDurationToLocaleString(call FunctionCall) Value {
	d := r.toTemporalDuration(call.This)
	return r.ToValue(r.formatTemporalDuration(d, _undefined))
}

func (r *Runtime) temporalDurationValueOf(call FunctionCall) Value {
	panic(r.NewTypeError("Temporal.Duration objects cannot be converted to primitives"))
}

func (r *Runtime) temporalDurationCompare(call FunctionCall) Value {
	a := r.temporalDurationFromValue(call.Argument(0))
	b := r.temporalDurationFromValue(call.Argument(1))
	optsObj := r.temporalOptionsObject(call.Argument(2))
	rel := r.temporalDurationRelativeTo(call.Argument(2))
	if rel == nil && (a.hasCalendarUnits() || b.hasCalendarUnits()) {
		if temporalDurationFieldsEqual(a, b) {
			return intToValue(0)
		}
		panic(r.newError(r.getRangeError(), "relativeTo is required for calendar units"))
	}
	var ta, tb *big.Rat
	if rel != nil && (a.hasCalendarUnits() || b.hasCalendarUnits() || a.days != 0 || b.days != 0) {
		_ = optsObj
		ta = r.durationTotalRelativeRat(a, rel, "day", false)
		tb = r.durationTotalRelativeRat(b, rel, "day", false)
	} else {
		ta = durationTimeNanos(a)
		tb = durationTimeNanos(b)
	}
	return intToValue(int64(ta.Cmp(tb)))
}

// temporalRelativeTo is the resolved relative-to value: a date and time,
// optionally in a time zone.
type temporalRelativeTo struct {
	date     isoDate
	time     isoTime
	zone     *temporalZone
	calendar string
}

// temporalDurationRelativeTo extracts options.relativeTo.
func (r *Runtime) temporalDurationRelativeTo(options Value) *temporalRelativeTo {
	if options == nil || IsUndefined(options) || IsNull(options) {
		return nil
	}
	obj, ok := options.(*Object)
	if !ok {
		panic(r.NewTypeError("options must be an object"))
	}
	v := obj.self.getStr("relativeTo", nil)
	if v == nil || IsUndefined(v) || IsNull(v) {
		return nil
	}
	return r.temporalRelativeToFromValue(v, options)
}

// temporalRelativeToFromValue converts a relativeTo value to a date-time and
// optional time zone.
func (r *Runtime) temporalRelativeToFromValue(v Value, options Value) *temporalRelativeTo {
	if v == nil || IsUndefined(v) || IsNull(v) {
		panic(r.NewTypeError("relativeTo must be an object or string"))
	}
	if o, ok := v.(*Object); ok {
		switch x := o.self.(type) {
		case *temporalPlainDateObject:
			return &temporalRelativeTo{date: x.date, calendar: x.calendar}
		case *temporalPlainDateTimeObject:
			return &temporalRelativeTo{date: x.date, time: x.time, calendar: x.calendar}
		case *temporalZonedDateTimeObject:
			d, tm, _ := x.zone.localDateTime(x.epochNs)
			return &temporalRelativeTo{date: d, time: tm, zone: x.zone, calendar: x.calendar}
		case *temporalPlainTimeObject, *temporalInstantObject, *temporalDurationObject,
			*temporalPlainYearMonthObject, *temporalPlainMonthDayObject:
			panic(r.NewTypeError("Invalid relativeTo value"))
		}
		res := r.temporalReadZonedDateTimeFields(o, temporalDateTimeFieldsResult{calendar: "iso8601"}, false)
		disambiguation := "compatible"
		offsetOpt := "reject"
		if !res.haveYear || (!res.haveMonth && !res.hasMonthCd) || !res.haveDay {
			panic(r.NewTypeError("year, month and day are required"))
		}
		date, tm := res.date, res.time
		if res.hasMonthCd {
			n, ok := isoMonthFromMonthCode(res.monthCode)
			if !ok {
				panic(r.newError(r.getRangeError(), "Invalid monthCode"))
			}
			if res.monthSet && date.month != n {
				panic(r.newError(r.getRangeError(), "month and monthCode must agree"))
			}
			date.month = n
		}
		if tm.second == 60 {
			tm.second = 59
		}
		if res.hasTime || res.hasTZ {
			if !date.isValidDateTime(tm) {
				panic(r.newError(r.getRangeError(), "Invalid date/time"))
			}
		} else if !date.isValid() {
			panic(r.newError(r.getRangeError(), "Invalid date"))
		}
		if res.hasTZ {
			zone := r.parseTemporalZone(res.tzValue)
			local := new(big.Int).Mul(big.NewInt(date.days()), temporalBigDayNs)
			local.Add(local, big.NewInt(tm.nanos()))
			ns := r.zonedResolve(zone, local, res.offset, res.hasOffset, disambiguation, offsetOpt)
			if ns == nil || !r.validInstant(ns) {
				panic(r.newError(r.getRangeError(), "relativeTo is out of range"))
			}
			wd, wt, _ := zone.localDateTime(ns)
			return &temporalRelativeTo{date: wd, time: wt, zone: zone, calendar: res.calendar}
		}
		return &temporalRelativeTo{date: date, time: tm, calendar: res.calendar}
	}
	if s, ok := v.(String); ok {
		str := s.String()
		_, anns, aok := splitTemporalAnnotations(str)
		if !aok {
			panic(r.newError(r.getRangeError(), "Invalid relativeTo string"))
		}
		hasTZ := false
		for _, ann := range anns {
			a := strings.TrimPrefix(ann, "!")
			if !strings.Contains(a, "=") && isTimeZoneAnnotation(a) {
				hasTZ = true
			}
		}
		if hasTZ {
			z := r.parseTemporalZonedDateTimeString(str, options)
			d, tm, _ := z.zone.localDateTime(z.epochNs)
			return &temporalRelativeTo{date: d, time: tm, zone: z.zone, calendar: z.calendar}
		}
		optsObj := r.temporalOptionsObject(options)
		_ = r.temporalGetDisambiguationOption(optsObj)
		_ = r.temporalGetOffsetOption(optsObj, "reject")
		_ = r.temporalOverflowOption(options)
		if strings.ContainsAny(str, "Tt ") {
			dt, tm, cal, ok := parseTemporalPlainDateTimeString(str)
			if !ok {
				panic(r.newError(r.getRangeError(), "Invalid relativeTo string"))
			}
			return &temporalRelativeTo{date: dt, time: tm, calendar: cal}
		}
		d, cal, ok := parseTemporalPlainDateString(str)
		if !ok || !d.isValid() {
			panic(r.newError(r.getRangeError(), "Invalid relativeTo string"))
		}
		return &temporalRelativeTo{date: d, calendar: cal}
	}
	panic(r.NewTypeError("Invalid relativeTo value"))
}

func durationTimeNanos(d *temporalDurationObject) *big.Rat {
	n := new(big.Rat)
	add := func(v float64, mul int64) {
		if v != 0 {
			n.Add(n, new(big.Rat).Mul(new(big.Rat).SetFloat64(v), big.NewRat(mul, 1)))
		}
	}
	add(d.days, 86400*1e9)
	add(d.hours, 3600*1e9)
	add(d.minutes, 60*1e9)
	add(d.seconds, 1e9)
	add(d.milliseconds, 1e6)
	add(d.microseconds, 1e3)
	add(d.nanos, 1)
	return n
}
