package goja

import (
	"math"
	"math/big"

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
		for _, name := range []string{"Instant", "PlainDateTime", "PlainTime", "PlainDate",
			"ZonedDateTime", "PlainYearMonth", "PlainMonthDay"} {
			o._putProp(unistring.NewFromString(name), r.newTemporalStub(name), true, false, true)
		}
		now := r.NewObject()
		for _, name := range []string{"instant", "plainDateTimeISO", "plainDateISO",
			"plainTimeISO", "zonedDateTimeISO", "timeZoneId"} {
			now.Set(name, r.newNativeFunc(func(call FunctionCall) Value {
				panic(r.NewTypeError("Temporal.Now is not implemented yet"))
			}, unistring.NewFromString(name), 0))
		}
		o._putProp("Now", now, true, false, true)
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
	if opts == nil || IsUndefined(opts) || IsNull(opts) {
		panic(r.NewTypeError("options are required"))
	}
	if s, ok := opts.(String); ok {
		smallest, hasSmallest = normalizeTemporalUnit(s.String())
	} else {
		obj, ok := opts.(*Object)
		if !ok {
			panic(r.NewTypeError("options must be an object"))
		}
		if v := obj.self.getStr("smallestUnit", nil); v != nil && !IsUndefined(v) {
			s, ok := v.(String)
			if !ok {
				panic(r.newError(r.getRangeError(), "Invalid smallestUnit"))
			}
			smallest, hasSmallest = normalizeTemporalUnit(s.String())
		}
		if v := obj.self.getStr("largestUnit", nil); v != nil && !IsUndefined(v) {
			s, ok := v.(String)
			if !ok {
				panic(r.newError(r.getRangeError(), "Invalid largestUnit"))
			}
			largest, _ = normalizeTemporalUnit(s.String())
		}
		if v := obj.self.getStr("roundingIncrement", nil); v != nil && !IsUndefined(v) {
			n := v.ToNumber().ToFloat()
			if math.IsNaN(n) || math.IsInf(n, 0) || n != math.Trunc(n) || n < 1 || n > 1e9 {
				panic(r.newError(r.getRangeError(), "Invalid roundingIncrement"))
			}
			increment = int64(n)
		}
		mode = r.parseTemporalRoundingMode(obj.self.getStr("roundingMode", nil), temporalRoundHalfExpand)
	}
	if !hasSmallest && largest == "" {
		panic(r.NewTypeError("smallestUnit or largestUnit is required"))
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
	unit, ok := r.temporalDurationUnit(call.Argument(0))
	if !ok {
		panic(r.NewTypeError("Invalid unit"))
	}
	if d.hasCalendarUnits() {
		panic(r.newError(r.getRangeError(), "relativeTo is required for calendar units"))
	}
	total := durationTimeNanos(d)
	unitNanos := temporalUnitNanos(unit)
	if unitNanos == 0 {
		panic(r.NewTypeError("Invalid unit"))
	}
	rat := new(big.Rat).Quo(total, big.NewRat(unitNanos, 1))
	f, _ := rat.Float64()
	return r.ToValue(f)
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
	if a.hasCalendarUnits() || b.hasCalendarUnits() {
		panic(r.newError(r.getRangeError(), "relativeTo is required for calendar units"))
	}
	ta := durationTimeNanos(a)
	tb := durationTimeNanos(b)
	return intToValue(int64(ta.Cmp(tb)))
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
