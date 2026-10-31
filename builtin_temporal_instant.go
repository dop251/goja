package goja

import (
	"fmt"
	"math"
	"math/big"
	"strings"
)

type temporalInstantObject struct {
	baseObject
	epochNs *big.Int
}

var (
	temporalMaxInstantNs = new(big.Int).Mul(big.NewInt(86400*1e9), big.NewInt(100000000))
	temporalBigDayNs     = big.NewInt(86400 * 1e9)
)

func (r *Runtime) temporalBigIntValue(v Value) *big.Int {
	b := toBigInt(v)
	return new(big.Int).Set((*big.Int)(b))
}

func (r *Runtime) validInstant(ns *big.Int) bool {
	return new(big.Int).Abs(ns).Cmp(temporalMaxInstantNs) <= 0
}

func (r *Runtime) getTemporalInstant() *Object {
	ret := r.global.TemporalInstant
	if ret == nil {
		ret = &Object{runtime: r}
		r.global.TemporalInstant = ret
		r.newNativeFuncAndConstruct(ret, func(call FunctionCall) Value {
			panic(r.NewTypeError("Constructor Temporal.Instant requires 'new'"))
		}, func(args []Value, newTarget *Object) *Object {
			var arg0 Value
			if len(args) > 0 {
				arg0 = args[0]
			}
			ns := r.temporalBigIntValue(arg0)
			if !r.validInstant(ns) {
				panic(r.newError(r.getRangeError(), "Instant is out of range"))
			}
			proto := r.getTemporalInstantPrototype()
			if newTarget != nil {
				proto = r.getPrototypeFromCtor(newTarget, ret, proto)
			}
			return r.newTemporalInstantObject(ns, proto)
		}, r.getTemporalInstantPrototype(), "Instant", intToValue(1))
		ctor := ret.self.(*nativeFuncObject)
		ctor._putProp("from", r.newNativeFunc(r.temporalInstantFrom, "from", 1), true, false, true)
		ctor._putProp("fromEpochMilliseconds", r.newNativeFunc(r.temporalInstantFromEpochMilliseconds, "fromEpochMilliseconds", 1), true, false, true)
		ctor._putProp("fromEpochNanoseconds", r.newNativeFunc(r.temporalInstantFromEpochNanoseconds, "fromEpochNanoseconds", 1), true, false, true)
		ctor._putProp("compare", r.newNativeFunc(r.temporalInstantCompare, "compare", 2), true, false, true)
	}
	return ret
}

func (r *Runtime) getTemporalInstantPrototype() *Object {
	ret := r.global.TemporalInstantPrototype
	if ret == nil {
		o := r.newBaseObject(r.global.ObjectPrototype, classObject)
		ret = o.val
		r.global.TemporalInstantPrototype = ret
		o._putProp("constructor", r.getTemporalInstant(), true, false, true)
		o.setOwnStr("epochMilliseconds", &valueProperty{
			getterFunc: r.newNativeFunc(func(call FunctionCall) Value {
				t := r.toTemporalInstant(call.This)
				q, rem := new(big.Int).QuoRem(t.epochNs, big.NewInt(1e6), new(big.Int))
				if rem.Sign() < 0 {
					q.Sub(q, big.NewInt(1))
				}
				f, _ := new(big.Float).SetInt(q).Float64()
				return r.ToValue(f)
			}, "get epochMilliseconds", 0),
			accessor: true, enumerable: false, configurable: true,
		}, true)
		o.setOwnStr("epochNanoseconds", &valueProperty{
			getterFunc: r.newNativeFunc(func(call FunctionCall) Value {
				t := r.toTemporalInstant(call.This)
				return r.ToValue(new(big.Int).Set(t.epochNs))
			}, "get epochNanoseconds", 0),
			accessor: true, enumerable: false, configurable: true,
		}, true)

		o._putProp("add", r.newNativeFunc(r.temporalInstantAdd, "add", 1), true, false, true)
		o._putProp("subtract", r.newNativeFunc(r.temporalInstantSubtract, "subtract", 1), true, false, true)
		o._putProp("until", r.newNativeFunc(r.temporalInstantUntil, "until", 1), true, false, true)
		o._putProp("since", r.newNativeFunc(r.temporalInstantSince, "since", 1), true, false, true)
		o._putProp("round", r.newNativeFunc(r.temporalInstantRound, "round", 1), true, false, true)
		o._putProp("equals", r.newNativeFunc(r.temporalInstantEquals, "equals", 1), true, false, true)
		o._putProp("toString", r.newNativeFunc(r.temporalInstantToString, "toString", 0), true, false, true)
		o._putProp("toJSON", r.newNativeFunc(r.temporalInstantToJSON, "toJSON", 0), true, false, true)
		o._putProp("toLocaleString", r.newNativeFunc(r.temporalInstantToLocaleString, "toLocaleString", 0), true, false, true)
		o._putProp("valueOf", r.newNativeFunc(r.temporalInstantValueOf, "valueOf", 0), true, false, true)
		o._putProp("toZonedDateTimeISO", r.newNativeFunc(r.temporalInstantToZonedDateTimeISO, "toZonedDateTimeISO", 1), true, false, true)
		o._putSym(SymToStringTag, &valueProperty{
			value: asciiString("Temporal.Instant"), writable: false, enumerable: false, configurable: true,
		})
	}
	return ret
}

func (r *Runtime) newTemporalInstantObject(ns *big.Int, proto *Object) *Object {
	v := &Object{runtime: r}
	o := &temporalInstantObject{epochNs: new(big.Int).Set(ns)}
	o.baseObject = baseObject{class: classObject, val: v, extensible: true, prototype: proto}
	v.self = o
	o.init()
	return v
}

func (r *Runtime) toTemporalInstant(v Value) *temporalInstantObject {
	o := r.toObject(v)
	if t, ok := o.self.(*temporalInstantObject); ok {
		return t
	}
	panic(r.NewTypeError("Value is not a Temporal.Instant"))
}

func (r *Runtime) temporalInstantFromValue(v Value) *temporalInstantObject {
	if o, ok := v.(*Object); ok {
		if t, ok := o.self.(*temporalInstantObject); ok {
			return t
		}
		if z, ok := o.self.(*temporalZonedDateTimeObject); ok {
			return r.newTemporalInstantObject(z.epochNs, r.getTemporalInstantPrototype()).self.(*temporalInstantObject)
		}
	}
	switch v.(type) {
	case String, *Object:
	default:
		panic(r.NewTypeError("Cannot convert %s to a string", v))
	}
	str := v.toString().String()
	ns, ok := parseTemporalInstantString(str)
	if !ok {
		panic(r.newError(r.getRangeError(), "Invalid ISO instant string: "+str))
	}
	if !r.validInstant(ns) {
		panic(r.newError(r.getRangeError(), "Instant is out of range"))
	}
	return r.newTemporalInstantObject(ns, r.getTemporalInstantPrototype()).self.(*temporalInstantObject)
}

func (r *Runtime) temporalInstantFrom(call FunctionCall) Value {
	t := r.temporalInstantFromValue(call.Argument(0))
	return r.newTemporalInstantObject(t.epochNs, r.getTemporalInstantPrototype())
}

func (r *Runtime) temporalInstantFromEpochMilliseconds(call FunctionCall) Value {
	v := call.Argument(0)
	if _, ok := v.(*Symbol); ok {
		panic(r.NewTypeError("Cannot convert a Symbol to a number"))
	}
	f := v.ToNumber().ToFloat()
	if math.IsNaN(f) || math.IsInf(f, 0) || f != math.Trunc(f) {
		panic(r.newError(r.getRangeError(), "Invalid epoch milliseconds"))
	}
	ms := new(big.Int)
	if _, ok := ms.SetString(fmt.Sprintf("%.0f", f), 10); !ok {
		panic(r.newError(r.getRangeError(), "Invalid epoch milliseconds"))
	}
	ns := ms.Mul(ms, big.NewInt(1e6))
	if !r.validInstant(ns) {
		panic(r.newError(r.getRangeError(), "Instant is out of range"))
	}
	return r.newTemporalInstantObject(ns, r.getTemporalInstantPrototype())
}

func (r *Runtime) temporalInstantFromEpochNanoseconds(call FunctionCall) Value {
	ns := r.temporalBigIntValue(call.Argument(0))
	if !r.validInstant(ns) {
		panic(r.newError(r.getRangeError(), "Instant is out of range"))
	}
	return r.newTemporalInstantObject(ns, r.getTemporalInstantPrototype())
}

func (r *Runtime) temporalInstantAddSub(call FunctionCall, subtract bool) Value {
	t := r.toTemporalInstant(call.This)
	d := r.temporalDurationFromValue(call.Argument(0))
	fields := d.fields()
	if fields[0] != 0 || fields[1] != 0 || fields[2] != 0 || fields[3] != 0 {
		panic(r.newError(r.getRangeError(), "Instant cannot add years, months, weeks or days"))
	}
	total := new(big.Rat)
	add := func(v float64, mul int64) {
		if v != 0 {
			total.Add(total, new(big.Rat).Mul(new(big.Rat).SetFloat64(v), big.NewRat(mul, 1)))
		}
	}
	add(fields[4], 3600*1e9)
	add(fields[5], 60*1e9)
	add(fields[6], 1e9)
	add(fields[7], 1e6)
	add(fields[8], 1e3)
	add(fields[9], 1)
	if !total.IsInt() {
		panic(r.newError(r.getRangeError(), "Duration is not a whole number of nanoseconds"))
	}
	delta := total.Num()
	if subtract {
		delta = new(big.Int).Neg(delta)
	}
	ns := new(big.Int).Add(t.epochNs, delta)
	if !r.validInstant(ns) {
		panic(r.newError(r.getRangeError(), "Instant is out of range"))
	}
	return r.newTemporalInstantObject(ns, r.getTemporalInstantPrototype())
}

func (r *Runtime) temporalInstantAdd(call FunctionCall) Value {
	return r.temporalInstantAddSub(call, false)
}

func (r *Runtime) temporalInstantSubtract(call FunctionCall) Value {
	return r.temporalInstantAddSub(call, true)
}

func (r *Runtime) temporalInstantUntil(call FunctionCall) Value {
	a := r.toTemporalInstant(call.This)
	b := r.temporalInstantFromValue(call.Argument(0))
	return r.temporalInstantDifference(a, b, call.Argument(1), false)
}

func (r *Runtime) temporalInstantSince(call FunctionCall) Value {
	a := r.toTemporalInstant(call.This)
	b := r.temporalInstantFromValue(call.Argument(0))
	return r.temporalInstantDifference(a, b, call.Argument(1), true)
}

func (r *Runtime) temporalInstantDifference(a, b *temporalInstantObject, options Value, since bool) Value {
	s := r.temporalGetDifferenceSettings(options, "second", "nanosecond", 4, 9)
	if since {
		s.mode = negateRoundingMode(s.mode)
	}
	diff := new(big.Int).Sub(b.epochNs, a.epochNs)
	unit := new(big.Int).Mul(big.NewInt(temporalUnitNanos(s.smallest)), big.NewInt(s.increment))
	diff = temporalRoundNanos(diff, unit, s.mode)
	fields := [10]float64{}
	temporalTimeFieldsFromNanos(diff, temporalUnitIndex(s.largest), &fields)
	if since {
		for i := range fields {
			fields[i] = -fields[i]
		}
	}
	r.validateTemporalDuration(fields)
	return r.newTemporalDurationObject(fields, r.getTemporalDurationPrototype())
}

func (r *Runtime) temporalInstantRound(call FunctionCall) Value {
	t := r.toTemporalInstant(call.This)
	opts := call.Argument(0)
	if opts == nil || IsUndefined(opts) || IsNull(opts) {
		panic(r.NewTypeError("options are required"))
	}
	var smallest string
	increment := int64(1)
	mode := temporalRoundHalfExpand
	if s, ok := opts.(String); ok {
		u, valid := normalizeTemporalUnit(s.String())
		if !valid || temporalUnitIndex(u) < 4 {
			panic(r.newError(r.getRangeError(), "Invalid smallestUnit"))
		}
		smallest = u
	} else {
		obj, ok := opts.(*Object)
		if !ok {
			panic(r.NewTypeError("options must be an object"))
		}
		increment = r.temporalGetRoundingIncrementOption(obj)
		mode = r.parseTemporalRoundingMode(obj.self.getStr("roundingMode", nil), temporalRoundHalfExpand)
		v := obj.self.getStr("smallestUnit", nil)
		if v == nil || IsUndefined(v) {
			panic(r.newError(r.getRangeError(), "smallestUnit is required"))
		}
		str, _ := temporalToStringOption(r, v)
		u, valid := normalizeTemporalUnit(str)
		if !valid || temporalUnitIndex(u) < 4 {
			panic(r.newError(r.getRangeError(), "Invalid smallestUnit"))
		}
		smallest = u
	}
	if err := validateInstantRoundingIncrement(smallest, increment); err != nil {
		panic(r.newError(r.getRangeError(), err.Error()))
	}
	ns := r.roundInstant(t.epochNs, smallest, increment, mode)
	return r.newTemporalInstantObject(ns, r.getTemporalInstantPrototype())
}

// validateInstantRoundingIncrement checks that the increment evenly divides a
// day, which may include the whole day itself.
func validateInstantRoundingIncrement(unit string, increment int64) error {
	dayUnits := int64(86400*1e9) / temporalUnitNanos(unit)
	if increment < 1 || increment > dayUnits || dayUnits%increment != 0 {
		return fmt.Errorf("invalid rounding increment")
	}
	return nil
}

// roundInstant rounds an instant using its UTC wall clock, so that truncation
// always moves towards the beginning of time.
func (r *Runtime) roundInstant(ns *big.Int, unit string, increment int64, mode temporalRoundingMode) *big.Int {
	date, tm, ok := isoDateTimeFromEpochNs(ns)
	if !ok {
		panic(r.newError(r.getRangeError(), "Instant is out of range"))
	}
	nd, nt := r.roundISODateTime(date, tm, unit, increment, mode)
	out := new(big.Int).Mul(big.NewInt(nd.days()), temporalBigDayNs)
	out.Add(out, big.NewInt(nt.nanos()))
	if !r.validInstant(out) {
		panic(r.newError(r.getRangeError(), "Instant is out of range"))
	}
	return out
}

func (r *Runtime) temporalInstantEquals(call FunctionCall) Value {
	a := r.toTemporalInstant(call.This)
	b := r.temporalInstantFromValue(call.Argument(0))
	return r.ToValue(a.epochNs.Cmp(b.epochNs) == 0)
}

func (r *Runtime) temporalInstantCompare(call FunctionCall) Value {
	a := r.temporalInstantFromValue(call.Argument(0))
	b := r.temporalInstantFromValue(call.Argument(1))
	return intToValue(int64(a.epochNs.Cmp(b.epochNs)))
}

// isoDateTimeFromEpochNs splits an epoch nanosecond count into an ISO date and
// time of day in UTC.
func isoDateTimeFromEpochNs(ns *big.Int) (isoDate, isoTime, bool) {
	days, rem := new(big.Int).QuoRem(ns, temporalBigDayNs, new(big.Int))
	if rem.Sign() < 0 {
		days.Sub(days, big.NewInt(1))
		rem.Add(rem, temporalBigDayNs)
	}
	if !days.IsInt64() {
		return isoDate{}, isoTime{}, false
	}
	y, m, d := isoCivilFromDays(days.Int64())
	return isoDate{year: y, month: m, day: d}, isoTimeFromNanos(rem.Int64()), true
}

func (r *Runtime) temporalInstantToString(call FunctionCall) Value {
	t := r.toTemporalInstant(call.This)
	options := call.Argument(0)
	// Spec order: fractionalSecondDigits, roundingMode, smallestUnit, timeZone.
	offset := (*big.Int)(nil)
	var suffix string
	p := r.temporalGetStringPrecision(options, func() {
		if obj, ok := options.(*Object); ok && options != nil {
			if v := obj.self.getStr("timeZone", nil); v != nil && !IsUndefined(v) {
				str, ok := v.(String)
				if !ok {
					panic(r.NewTypeError("timeZone must be a string"))
				}
				off, ok := r.parseTimeZoneOffsetString(str.String())
				if !ok {
					panic(r.newError(r.getRangeError(), "Unsupported time zone: "+str.String()))
				}
				minutes := new(big.Int).Quo(off, big.NewInt(60*1e9)).Int64()
				offset = off
				suffix = formatOffsetMinutes(minutes)
			}
		}
	})
	ns := new(big.Int).Set(t.epochNs)
	if p.hasUnit {
		ns = r.roundInstant(ns, p.unit, 1, p.mode)
	} else if p.digits >= 0 {
		if p.digits == 0 {
			ns = r.roundInstant(ns, "second", 1, p.mode)
		} else {
			inc := int64(1)
			for i := 0; i < 9-p.digits; i++ {
				inc *= 10
			}
			ns = r.roundInstant(ns, "nanosecond", inc, p.mode)
		}
	}
	display := ns
	if offset != nil {
		display = new(big.Int).Add(ns, offset)
	}
	date, tm, ok := isoDateTimeFromEpochNs(display)
	if !ok || !date.isValidDateTime(tm) {
		panic(r.newError(r.getRangeError(), "Instant is out of range"))
	}
	out := formatISODate(date) + "T" + formatISOTimePrecision(tm, p)
	if suffix == "" {
		out += "Z"
	} else {
		out += suffix
	}
	return r.ToValue(out)
}

func formatOffsetMinutes(minutes int64) string {
	sign := "+"
	if minutes < 0 {
		sign = "-"
		minutes = -minutes
	}
	return fmt.Sprintf("%s%02d:%02d", sign, minutes/60, minutes%60)
}

func (r *Runtime) temporalInstantToJSON(call FunctionCall) Value {
	return r.temporalInstantToString(FunctionCall{This: call.This, Arguments: []Value{_undefined}})
}

func (r *Runtime) temporalInstantToLocaleString(call FunctionCall) Value {
	return r.temporalToLocaleString(call, "any")
}

func (r *Runtime) temporalInstantValueOf(call FunctionCall) Value {
	panic(r.NewTypeError("Temporal.Instant objects cannot be converted to primitives"))
}

func (r *Runtime) temporalInstantToZonedDateTimeISO(call FunctionCall) Value {
	t := r.toTemporalInstant(call.This)
	zone := r.parseTemporalZone(call.Argument(0))
	return r.newTemporalZonedDateTimeObject(t.epochNs, zone, "iso8601", r.getTemporalZonedDateTimePrototype())
}

// parseTemporalOffsetString parses a UTC offset such as +01:00, -0130 or
// +00:19:32.37 into nanoseconds. "Z" and "z" parse to zero.
func parseTemporalOffsetString(s string) (*big.Int, bool) {
	if s == "Z" || s == "z" {
		return big.NewInt(0), true
	}
	if !validOffsetString(s) {
		return nil, false
	}
	sign := int64(1)
	if s[0] == '-' {
		sign = -1
	}
	body := s[1:]
	frac := ""
	if i := strings.IndexAny(body, ".,"); i >= 0 {
		frac = body[i+1:]
		body = body[:i]
	}
	digits := strings.ReplaceAll(body, ":", "")
	pad := func(v string) int64 {
		n := int64(0)
		for i := 0; i < len(v); i++ {
			n = n*10 + int64(v[i]-'0')
		}
		return n
	}
	var h, m, sec int64
	switch len(digits) {
	case 2:
		h = pad(digits)
	case 4:
		h, m = pad(digits[:2]), pad(digits[2:])
	case 6:
		h, m, sec = pad(digits[:2]), pad(digits[2:4]), pad(digits[4:])
	default:
		return nil, false
	}
	fracNs := int64(0)
	if frac != "" {
		if len(frac) > 9 {
			return nil, false
		}
		for len(frac) < 9 {
			frac += "0"
		}
		fracNs = pad(frac)
	}
	total := ((h*60+m)*60+sec)*1e9 + fracNs
	return big.NewInt(sign * total), true
}

// parseTemporalInstantString parses an ISO date-time string with a mandatory
// UTC offset into an epoch nanosecond count.
func parseTemporalInstantString(s string) (*big.Int, bool) {
	base, anns, ok := splitTemporalAnnotations(s)
	if !ok {
		return nil, false
	}
	tzSeen := false
	calSeen := false
	calCritical := false
	for _, ann := range anns {
		critical := strings.HasPrefix(ann, "!")
		if critical {
			ann = ann[1:]
		}
		key, _, hasVal := strings.Cut(ann, "=")
		if hasVal {
			if key != strings.ToLower(key) || key == "" {
				return nil, false
			}
			if key == "u-ca" {
				// Instant ignores calendar annotations, but rejects a second
				// one if either is critical.
				if calSeen && (critical || calCritical) {
					return nil, false
				}
				calSeen = true
				calCritical = critical
				continue
			}
		}
		if isTimeZoneAnnotation(ann) {
			if tzSeen {
				return nil, false
			}
			if ann[0] == '+' || ann[0] == '-' {
				if !validTimeZoneAnnotationOffset(ann) {
					return nil, false
				}
			}
			tzSeen = true
			continue
		}
		if critical {
			return nil, false
		}
	}
	sep := strings.IndexAny(base, "Tt")
	if sep < 0 {
		if i := strings.IndexByte(base, ' '); i > 0 {
			sep = i
		}
	}
	if sep < 0 {
		return nil, false
	}
	datePart := base[:sep]
	timePart := base[sep+1:]
	if timePart == "" {
		return nil, false
	}
	if nd, ok := normalizeBasicDate(datePart); ok {
		datePart = nd
	}
	date, _, dok := parseTemporalPlainDateString(datePart)
	if !dok || !date.isValid() {
		return nil, false
	}
	offset := big.NewInt(0)
	hasOffset := false
	if strings.HasSuffix(timePart, "Z") || strings.HasSuffix(timePart, "z") {
		offset = big.NewInt(0)
		hasOffset = true
		timePart = timePart[:len(timePart)-1]
	} else if i := strings.LastIndexAny(timePart, "+-"); i >= 0 {
		off, ok := parseTemporalOffsetString(timePart[i:])
		if !ok {
			return nil, false
		}
		offset = off
		hasOffset = true
		timePart = timePart[:i]
	}
	if !hasOffset {
		return nil, false
	}
	h, mi, sec, ms, us, ns, tok := parseTemporalTimeFields(timePart)
	if !tok {
		return nil, false
	}
	timeOfDay := (&temporalPlainTimeObject{hour: h, minute: mi, second: sec, millisecond: ms, microsecond: us, nanosecond: ns}).nanosOfDay()
	total := new(big.Int).Mul(big.NewInt(date.days()), temporalBigDayNs)
	total.Add(total, big.NewInt(timeOfDay))
	total.Sub(total, offset)
	return total, true
}

// validTimeZoneOffsetLiteral accepts the minute-precision offset forms that
// are valid as time zone identifiers.
func validTimeZoneOffsetLiteral(s string) bool {
	if s == "Z" || s == "z" {
		return true
	}
	if len(s) < 3 || (s[0] != '+' && s[0] != '-') {
		return false
	}
	body := s[1:]
	switch len(body) {
	case 2, 4:
		return isDigits(body)
	case 5:
		return body[2] == ':' && isDigits(body[:2]) && isDigits(body[3:])
	}
	return false
}

// parseTimeZoneOffsetString implements the offset/UTC part of
// ToTemporalTimeZoneIdentifier. Named time zones need the time zone database
// and are not accepted yet.
func (r *Runtime) parseTimeZoneOffsetString(s string) (*big.Int, bool) {
	if strings.Contains(s, "-000000") {
		return nil, false
	}
	switch s {
	case "UTC", "Etc/UTC", "GMT":
		return big.NewInt(0), true
	}
	if validTimeZoneOffsetLiteral(s) {
		return parseTemporalOffsetString(s)
	}
	base, anns, ok := splitTemporalAnnotations(s)
	if !ok {
		return nil, false
	}
	tzAnn := ""
	for _, ann := range anns {
		if strings.HasPrefix(ann, "!") {
			ann = ann[1:]
		}
		if strings.Contains(ann, "=") {
			continue
		}
		if isTimeZoneAnnotation(ann) {
			if tzAnn != "" {
				return nil, false
			}
			tzAnn = ann
		}
	}
	if tzAnn != "" {
		switch tzAnn {
		case "UTC", "Etc/UTC", "GMT":
			return big.NewInt(0), true
		}
		if !validTimeZoneOffsetLiteral(tzAnn) {
			return nil, false
		}
		return parseTemporalOffsetString(tzAnn)
	}
	// No annotation: the date-time string's own offset is the zone.
	sep := strings.IndexAny(base, "Tt")
	if sep < 0 {
		return nil, false
	}
	timePart := base[sep+1:]
	if strings.HasSuffix(timePart, "Z") || strings.HasSuffix(timePart, "z") {
		return big.NewInt(0), true
	}
	i := strings.LastIndexAny(timePart, "+-")
	if i < 0 {
		return nil, false
	}
	offText := timePart[i:]
	if !validTimeZoneOffsetLiteral(offText) {
		return nil, false
	}
	return parseTemporalOffsetString(offText)
}

// validTimeZoneAnnotationOffset accepts only the minute-precision offset form
// used inside bracketed time zone annotations.
func validTimeZoneAnnotationOffset(s string) bool {
	if len(s) < 3 || (s[0] != '+' && s[0] != '-') {
		return false
	}
	body := s[1:]
	if !isDigits(strings.ReplaceAll(body, ":", "")) {
		return false
	}
	switch len(body) {
	case 2, 4:
		return true
	case 5:
		return body[2] == ':'
	}
	return false
}

// splitTemporalAnnotations separates a string from its trailing [key=value]
// annotations.
func splitTemporalAnnotations(s string) (string, []string, bool) {
	k := strings.IndexByte(s, '[')
	if k < 0 {
		return s, nil, true
	}
	rest := s[k:]
	var anns []string
	for len(rest) > 0 {
		if rest[0] != '[' {
			return "", nil, false
		}
		j := strings.IndexByte(rest, ']')
		if j < 0 {
			return "", nil, false
		}
		anns = append(anns, rest[1:j])
		rest = rest[j+1:]
	}
	return strings.TrimSpace(s[:k]), anns, true
}
