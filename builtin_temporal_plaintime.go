package goja

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/dop251/goja/unistring"
)

type temporalPlainTimeObject struct {
	baseObject
	hour, minute, second                 int64
	millisecond, microsecond, nanosecond int64
}

var temporalTimeFields = []temporalDurationField{
	{"hour", 0},
	{"microsecond", 4},
	{"millisecond", 3},
	{"minute", 1},
	{"nanosecond", 5},
	{"second", 2},
}

func (r *Runtime) getTemporalPlainTime() *Object {
	ret := r.global.TemporalPlainTime
	if ret == nil {
		ret = &Object{runtime: r}
		r.global.TemporalPlainTime = ret
		r.newNativeFuncAndConstruct(ret, func(call FunctionCall) Value {
			panic(r.NewTypeError("Constructor Temporal.PlainTime requires 'new'"))
		}, func(args []Value, newTarget *Object) *Object {
			h := r.temporalTimeArgument(args, 0)
			mi := r.temporalTimeArgument(args, 1)
			s := r.temporalTimeArgument(args, 2)
			ms := r.temporalTimeArgument(args, 3)
			us := r.temporalTimeArgument(args, 4)
			ns := r.temporalTimeArgument(args, 5)
			if h < 0 || h > 23 || mi < 0 || mi > 59 || s < 0 || s > 59 ||
				ms < 0 || ms > 999 || us < 0 || us > 999 || ns < 0 || ns > 999 {
				panic(r.newError(r.getRangeError(), "Invalid time"))
			}
			proto := r.getTemporalPlainTimePrototype()
			if newTarget != nil {
				proto = r.getPrototypeFromCtor(newTarget, ret, proto)
			}
			return r.newTemporalPlainTimeObject(h, mi, s, ms, us, ns, proto)
		}, r.getTemporalPlainTimePrototype(), "PlainTime", intToValue(0))
		ctor := ret.self.(*nativeFuncObject)
		ctor._putProp("from", r.newNativeFunc(r.temporalPlainTimeFrom, "from", 1), true, false, true)
		ctor._putProp("compare", r.newNativeFunc(r.temporalPlainTimeCompare, "compare", 2), true, false, true)
	}
	return ret
}

func (r *Runtime) temporalTimeArgument(args []Value, i int) int64 {
	var v Value
	if i < len(args) {
		v = args[i]
	}
	if v == nil || IsUndefined(v) || IsNull(v) {
		return 0
	}
	f := v.ToNumber().ToFloat()
	if math.IsNaN(f) || math.IsInf(f, 0) {
		panic(r.newError(r.getRangeError(), "Invalid time component"))
	}
	return int64(math.Trunc(f))
}

func (r *Runtime) getTemporalPlainTimePrototype() *Object {
	ret := r.global.TemporalPlainTimePrototype
	if ret == nil {
		o := r.newBaseObject(r.global.ObjectPrototype, classObject)
		ret = o.val
		r.global.TemporalPlainTimePrototype = ret
		o._putProp("constructor", r.getTemporalPlainTime(), true, false, true)
		addGetter := func(name string, f func(*temporalPlainTimeObject) Value) {
			o.setOwnStr(unistring.NewFromString(name), &valueProperty{
				getterFunc: r.newNativeFunc(func(call FunctionCall) Value {
					return f(r.toTemporalPlainTime(call.This))
				}, unistring.NewFromString("get "+name), 0),
				accessor: true, enumerable: false, configurable: true,
			}, true)
		}
		addGetter("hour", func(t *temporalPlainTimeObject) Value { return intToValue(t.hour) })
		addGetter("minute", func(t *temporalPlainTimeObject) Value { return intToValue(t.minute) })
		addGetter("second", func(t *temporalPlainTimeObject) Value { return intToValue(t.second) })
		addGetter("millisecond", func(t *temporalPlainTimeObject) Value { return intToValue(t.millisecond) })
		addGetter("microsecond", func(t *temporalPlainTimeObject) Value { return intToValue(t.microsecond) })
		addGetter("nanosecond", func(t *temporalPlainTimeObject) Value { return intToValue(t.nanosecond) })

		o._putProp("with", r.newNativeFunc(r.temporalPlainTimeWith, "with", 1), true, false, true)
		o._putProp("add", r.newNativeFunc(r.temporalPlainTimeAdd, "add", 1), true, false, true)
		o._putProp("subtract", r.newNativeFunc(r.temporalPlainTimeSubtract, "subtract", 1), true, false, true)
		o._putProp("until", r.newNativeFunc(r.temporalPlainTimeUntil, "until", 1), true, false, true)
		o._putProp("since", r.newNativeFunc(r.temporalPlainTimeSince, "since", 1), true, false, true)
		o._putProp("round", r.newNativeFunc(r.temporalPlainTimeRound, "round", 1), true, false, true)
		o._putProp("equals", r.newNativeFunc(r.temporalPlainTimeEquals, "equals", 1), true, false, true)
		o._putProp("toString", r.newNativeFunc(r.temporalPlainTimeToString, "toString", 0), true, false, true)
		o._putProp("toJSON", r.newNativeFunc(r.temporalPlainTimeToJSON, "toJSON", 0), true, false, true)
		o._putProp("toLocaleString", r.newNativeFunc(r.temporalPlainTimeToLocaleString, "toLocaleString", 0), true, false, true)
		o._putProp("valueOf", r.newNativeFunc(r.temporalPlainTimeValueOf, "valueOf", 0), true, false, true)
		o._putProp("getISOFields", r.newNativeFunc(r.temporalPlainTimeGetISOFields, "getISOFields", 0), true, false, true)
		o._putProp("toPlainDateTime", r.newNativeFunc(r.temporalPlainTimeToPlainDateTime, "toPlainDateTime", 1), true, false, true)
		o._putSym(SymToStringTag, &valueProperty{
			value: asciiString("Temporal.PlainTime"), writable: false, enumerable: false, configurable: true,
		})
	}
	return ret
}

func (r *Runtime) newTemporalPlainTimeObject(h, mi, s, ms, us, ns int64, proto *Object) *Object {
	v := &Object{runtime: r}
	o := &temporalPlainTimeObject{hour: h, minute: mi, second: s, millisecond: ms, microsecond: us, nanosecond: ns}
	o.baseObject = baseObject{class: classObject, val: v, extensible: true, prototype: proto}
	v.self = o
	o.init()
	return v
}

func (r *Runtime) toTemporalPlainTime(v Value) *temporalPlainTimeObject {
	o := r.toObject(v)
	if t, ok := o.self.(*temporalPlainTimeObject); ok {
		return t
	}
	panic(r.NewTypeError("Value is not a Temporal.PlainTime"))
}

func (r *Runtime) temporalPlainTimeFrom(call FunctionCall) Value {
	t := r.temporalPlainTimeFromValue(call.Argument(0), call.Argument(1))
	return r.newTemporalPlainTimeObject(t.hour, t.minute, t.second, t.millisecond, t.microsecond, t.nanosecond, r.getTemporalPlainTimePrototype())
}

func (r *Runtime) temporalPlainTimeFromValue(v Value, options Value) *temporalPlainTimeObject {
	if o, ok := v.(*Object); ok {
		if t, ok := o.self.(*temporalPlainTimeObject); ok {
			_ = r.temporalOverflowOption(options)
			return t
		}
		if dt, ok := o.self.(*temporalPlainDateTimeObject); ok {
			_ = r.temporalOverflowOption(options)
			return r.plainTimeFieldsObject(dt.time.hour, dt.time.minute, dt.time.second, dt.time.millisecond, dt.time.microsecond, dt.time.nanosecond)
		}
		if z, ok := o.self.(*temporalZonedDateTimeObject); ok {
			_ = r.temporalOverflowOption(options)
			_, tm, _ := z.zone.localDateTime(z.epochNs)
			return r.plainTimeFieldsObject(tm.hour, tm.minute, tm.second, tm.millisecond, tm.microsecond, tm.nanosecond)
		}
	}
	if s, ok := v.(String); ok {
		h, mi, sec, ms, us, ns, ok := parseTemporalPlainTimeString(s.String())
		if !ok {
			panic(r.newError(r.getRangeError(), "Invalid ISO time string: "+s.String()))
		}
		_ = r.temporalOverflowOption(options)
		return r.plainTimeFieldsObject(h, mi, sec, ms, us, ns)
	}
	obj, ok := v.(*Object)
	if !ok {
		panic(r.NewTypeError("Cannot convert %s to Temporal.PlainTime", v))
	}
	var fields [6]int64
	found := false
	for _, f := range temporalTimeFields {
		fv := obj.self.getStr(unistring.NewFromString(f.name), nil)
		if fv == nil || IsUndefined(fv) {
			continue
		}
		found = true
		fields[f.index] = r.temporalTimeFieldValue(fv)
	}
	if !found {
		panic(r.NewTypeError("Time-like object must have at least one time property"))
	}
	overflow := r.temporalOverflowOption(options)
	h, mi, sec, ms, us, ns := fields[0], fields[1], fields[2], fields[3], fields[4], fields[5]
	if overflow == "constrain" {
		if h < 0 {
			h = 0
		}
		if h > 23 {
			h = 23
		}
		if mi < 0 {
			mi = 0
		}
		if mi > 59 {
			mi = 59
		}
		if sec < 0 {
			sec = 0
		}
		if sec > 59 {
			sec = 59
		}
		ms = clampInt(ms, 0, 999)
		us = clampInt(us, 0, 999)
		ns = clampInt(ns, 0, 999)
	}
	if h < 0 || h > 23 || mi < 0 || mi > 59 || sec < 0 || sec > 59 || ms < 0 || ms > 999 || us < 0 || us > 999 || ns < 0 || ns > 999 {
		panic(r.newError(r.getRangeError(), "Invalid time"))
	}
	return r.plainTimeFieldsObject(h, mi, sec, ms, us, ns)
}

func validateTimeRoundingIncrement(unit string, increment int64) error {
	max := int64(1000)
	switch unit {
	case "hour":
		max = 24
	case "minute", "second":
		max = 60
	}
	if increment >= max || max%increment != 0 {
		return fmt.Errorf("invalid rounding increment")
	}
	return nil
}

func clampInt(v, lo, hi int64) int64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (r *Runtime) temporalTimeFieldValue(v Value) int64 {
	if _, ok := v.(*Symbol); ok {
		panic(r.NewTypeError("Cannot convert a Symbol to a number"))
	}
	f := v.ToNumber().ToFloat()
	if math.IsNaN(f) || math.IsInf(f, 0) {
		panic(r.newError(r.getRangeError(), "Invalid time component"))
	}
	return int64(math.Trunc(f))
}

func (r *Runtime) plainTimeFieldsObject(h, mi, s, ms, us, ns int64) *temporalPlainTimeObject {
	o := r.newTemporalPlainTimeObject(h, mi, s, ms, us, ns, r.getTemporalPlainTimePrototype())
	return o.self.(*temporalPlainTimeObject)
}

// parseTemporalPlainTimeString parses a PlainTime ISO string.
func parseTemporalPlainTimeString(s string) (h, mi, sec, ms, us, ns int64, ok bool) {
	base, annOK := stripPlainTimeAnnotations(s)
	if !annOK {
		return
	}
	s = base
	if !strings.ContainsAny(s, "Tt") && !strings.Contains(s, " ") && ambiguousISOYearMonthOrMonthDay(s) {
		return
	}
	timePart := s
	if i := strings.IndexAny(s, "Tt"); i >= 0 {
		prefix := s[:i]
		if !validDatePrefix(prefix) {
			return
		}
		timePart = s[i+1:]
	} else if i := strings.IndexByte(s, ' '); i > 0 {
		prefix := s[:i]
		if !validDatePrefix(prefix) {
			return
		}
		timePart = s[i+1:]
	} else if looksLikeDateOnly(s) {
		return
	}
	return parseTemporalTimeFields(timePart)
}

// validDatePrefix validates a date prefix that precedes a time component.
func validDatePrefix(prefix string) bool {
	if prefix == "" {
		return true
	}
	if norm, ok := normalizeBasicDate(prefix); ok {
		if strings.Contains(norm, "-000000") || strings.HasPrefix(norm, "-0000-") {
			return false
		}
		_, _, ok := parseTemporalPlainDateString(norm)
		return ok
	}
	if strings.Contains(prefix, "-000000") || strings.HasPrefix(prefix, "-0000-") {
		return false
	}
	if !strings.Contains(prefix, "-") {
		return false
	}
	_, _, ok := parseTemporalPlainDateString(prefix)
	return ok
}

func looksLikeDateOnly(s string) bool {
	if len(s) == 7 && isDigits(s[:4]) && s[4] == '-' && isDigits(s[5:]) {
		m := int64(s[5]-'0')*10 + int64(s[6]-'0')
		return m >= 1 && m <= 12
	}
	if _, _, ok := parseTemporalPlainDateString(s); ok {
		return true
	}
	return false
}

func stripPlainTimeAnnotations(s string) (string, bool) {
	k := strings.IndexByte(s, '[')
	if k < 0 {
		return s, true
	}
	rest := s[k:]
	var anns []string
	for len(rest) > 0 {
		if rest[0] != '[' {
			return "", false
		}
		j := strings.IndexByte(rest, ']')
		if j < 0 {
			return "", false
		}
		anns = append(anns, rest[1:j])
		rest = rest[j+1:]
	}
	tz, cal := 0, 0
	calCritical := false
	for _, ann := range anns {
		if ann == "" {
			return "", false
		}
		critical := strings.HasPrefix(ann, "!")
		if critical {
			ann = ann[1:]
		}
		key, _, hasVal := strings.Cut(ann, "=")
		if hasVal {
			if key != strings.ToLower(key) || key == "" {
				return "", false
			}
			if key == "u-ca" {
				// PlainTime ignores all calendar annotations, but a critical
				// annotation cannot be followed by a second one.
				cal++
				if cal == 1 {
					calCritical = critical
				} else if critical || calCritical {
					return "", false
				}
				continue
			}
		}
		if isTimeZoneAnnotation(ann) {
			tz++
			if tz > 1 {
				return "", false
			}
			continue
		}
		if critical {
			return "", false
		}
	}
	return strings.TrimSpace(s[:k]), true
}

func parseTemporalTimeFields(s string) (h, mi, sec, ms, us, ns int64, ok bool) {
	if s == "" {
		return
	}
	i := 0
	read2 := func() (int64, bool) {
		if i+2 > len(s) || !isDigits(s[i:i+2]) {
			return 0, false
		}
		v, _ := strconv.ParseInt(s[i:i+2], 10, 64)
		i += 2
		return v, true
	}
	h, ok = read2()
	if !ok {
		return
	}
	hasMinute := false
	colonMinute := false
	if i < len(s) && s[i] == ':' {
		i++
		mi, ok = read2()
		if !ok {
			return
		}
		hasMinute = true
		colonMinute = true
	} else if i+2 <= len(s) && isDigits(s[i:i+2]) && !isOffsetStart(s, i) {
		mi, ok = read2()
		if !ok {
			return
		}
		hasMinute = true
	}
	hasSecond := false
	if hasMinute {
		if colonMinute {
			if i < len(s) && s[i] == ':' {
				i++
				sec, ok = read2()
				if !ok {
					return
				}
				hasSecond = true
			} else if i < len(s) && s[i] >= '0' && s[i] <= '9' {
				ok = false
				return
			}
		} else if i+2 <= len(s) && isDigits(s[i:i+2]) && !isOffsetStart(s, i) {
			sec, ok = read2()
			if !ok {
				return
			}
			hasSecond = true
		}
	}
	if hasSecond && i < len(s) && (s[i] == '.' || s[i] == ',') {
		i++
		fs := i
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
		if i == fs || i-fs > 9 {
			ok = false
			return
		}
		fd := s[fs:i]
		for len(fd) < 9 {
			fd += "0"
		}
		ms64, _ := strconv.ParseInt(fd[:3], 10, 64)
		us64, _ := strconv.ParseInt(fd[3:6], 10, 64)
		ns64, _ := strconv.ParseInt(fd[6:9], 10, 64)
		ms, us, ns = ms64, us64, ns64
	}
	// Optional UTC offset; Z is not allowed for PlainTime.
	if i < len(s) && (s[i] == 'Z' || s[i] == 'z') {
		ok = false
		return
	}
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		if !validOffsetString(s[i:]) {
			ok = false
			return
		}
		i = len(s)
	}
	if i != len(s) {
		ok = false
		return
	}
	if sec == 60 {
		sec = 59
	}
	if h > 23 || mi > 59 || sec > 59 {
		ok = false
	}
	return
}

// validOffsetString validates a UTC offset suffix such as +01:00 or -0130.
func validOffsetString(s string) bool {
	if len(s) < 3 || (s[0] != '+' && s[0] != '-') || !isDigits(s[1:3]) {
		return false
	}
	if s[1] > '2' || (s[1] == '2' && s[2] > '3') {
		return false
	}
	rest := s[3:]
	if rest == "" {
		return true
	}
	colon := rest[0] == ':'
	if colon {
		rest = rest[1:]
	}
	if len(rest) < 2 || !isDigits(rest[:2]) || rest[0] > '5' {
		return false
	}
	rest = rest[2:]
	if rest == "" {
		return true
	}
	if colon {
		if rest[0] != ':' {
			return false
		}
		rest = rest[1:]
	}
	if len(rest) < 2 || !isDigits(rest[:2]) || rest[0] > '5' {
		return false
	}
	rest = rest[2:]
	if rest == "" {
		return true
	}
	if rest[0] != '.' && rest[0] != ',' {
		return false
	}
	frac := rest[1:]
	return len(frac) >= 1 && len(frac) <= 9 && isDigits(frac)
}

func isOffsetStart(s string, i int) bool {
	// A compact HHMM time is only ambiguous with an offset directly after the
	// hour when the next character is a sign; digits here are still minutes.
	return i < len(s) && (s[i] == '+' || s[i] == '-')
}

func (r *Runtime) temporalPlainTimeWith(call FunctionCall) Value {
	t := r.toTemporalPlainTime(call.This)
	partial := r.toObject(call.Argument(0))
	_ = partial.self.getStr("calendar", nil)
	_ = partial.self.getStr("timeZone", nil)
	fields := [6]int64{t.hour, t.minute, t.second, t.millisecond, t.microsecond, t.nanosecond}
	found := false
	for _, f := range temporalTimeFields {
		v := partial.self.getStr(unistring.NewFromString(f.name), nil)
		if v == nil || IsUndefined(v) {
			continue
		}
		found = true
		fields[f.index] = r.temporalTimeFieldValue(v)
	}
	if !found {
		panic(r.NewTypeError("PlainTime.with requires at least one time property"))
	}
	overflow := r.temporalOverflowOption(call.Argument(1))
	if overflow == "constrain" {
		fields[0] = clampInt(fields[0], 0, 23)
		fields[1] = clampInt(fields[1], 0, 59)
		fields[2] = clampInt(fields[2], 0, 59)
		fields[3] = clampInt(fields[3], 0, 999)
		fields[4] = clampInt(fields[4], 0, 999)
		fields[5] = clampInt(fields[5], 0, 999)
	} else if fields[0] < 0 || fields[1] < 0 || fields[2] < 0 || fields[3] < 0 || fields[4] < 0 || fields[5] < 0 ||
		fields[0] > 23 || fields[1] > 59 || fields[2] > 59 || fields[3] > 999 || fields[4] > 999 || fields[5] > 999 {
		panic(r.newError(r.getRangeError(), "Invalid time"))
	}
	return r.newTemporalPlainTimeObject(fields[0], fields[1], fields[2], fields[3], fields[4], fields[5], r.getTemporalPlainTimePrototype())
}

func (t *temporalPlainTimeObject) nanosOfDay() int64 {
	return ((t.hour*60+t.minute)*60+t.second)*1e9 + t.millisecond*1e6 + t.microsecond*1e3 + t.nanosecond
}

func (r *Runtime) plainTimeFromNanos(ns int64) *Object {
	ns = ((ns % 86400e9) + 86400e9) % 86400e9
	h := ns / 3600e9
	ns %= 3600e9
	mi := ns / 60e9
	ns %= 60e9
	s := ns / 1e9
	ns %= 1e9
	return r.newTemporalPlainTimeObject(h, mi, s, ns/1e6, (ns%1e6)/1e3, ns%1e3, r.getTemporalPlainTimePrototype())
}

func (r *Runtime) temporalPlainTimeAddSub(call FunctionCall, subtract bool) Value {
	t := r.toTemporalPlainTime(call.This)
	d := r.temporalDurationFromValue(call.Argument(0))
	fields := d.fields()
	delta := new(big.Rat)
	add := func(v float64, mul int64) {
		if v != 0 {
			delta.Add(delta, new(big.Rat).Mul(new(big.Rat).SetFloat64(v), big.NewRat(mul, 1)))
		}
	}
	add(fields[4], 3600*1e9)
	add(fields[5], 60*1e9)
	add(fields[6], 1e9)
	add(fields[7], 1e6)
	add(fields[8], 1e3)
	add(fields[9], 1)
	q := new(big.Rat).Quo(delta, big.NewRat(1, 1))
	deltaNs := new(big.Int).Quo(q.Num(), q.Denom())
	if subtract {
		deltaNs.Neg(deltaNs)
	}
	base := new(big.Int).SetInt64(t.nanosOfDay())
	base.Add(base, deltaNs)
	base.Mod(base, big.NewInt(86400e9))
	return r.plainTimeFromNanos(base.Int64())
}

func (r *Runtime) temporalPlainTimeAdd(call FunctionCall) Value {
	return r.temporalPlainTimeAddSub(call, false)
}

func (r *Runtime) temporalPlainTimeSubtract(call FunctionCall) Value {
	return r.temporalPlainTimeAddSub(call, true)
}

func (r *Runtime) temporalPlainTimeUntil(call FunctionCall) Value {
	t := r.toTemporalPlainTime(call.This)
	other := r.temporalPlainTimeFromValue(call.Argument(0), _undefined)
	return r.temporalPlainTimeDifference(t, other, call.Argument(1), false)
}

func (r *Runtime) temporalPlainTimeSince(call FunctionCall) Value {
	t := r.toTemporalPlainTime(call.This)
	other := r.temporalPlainTimeFromValue(call.Argument(0), _undefined)
	return r.temporalPlainTimeDifference(t, other, call.Argument(1), true)
}

func (r *Runtime) temporalPlainTimeDifference(a, b *temporalPlainTimeObject, options Value, since bool) Value {
	s := r.temporalGetDifferenceSettings(options, "hour", "nanosecond", 4, 9)
	if since {
		s.mode = negateRoundingMode(s.mode)
	}
	total := big.NewInt(b.nanosOfDay() - a.nanosOfDay())
	unit := new(big.Int).Mul(big.NewInt(temporalUnitNanos(s.smallest)), big.NewInt(s.increment))
	total = temporalRoundNanos(total, unit, s.mode)
	fields := [10]float64{}
	temporalTimeFieldsFromNanos(total, temporalUnitIndex(s.largest), &fields)
	if since {
		for i := range fields {
			fields[i] = -fields[i]
		}
	}
	r.validateTemporalDuration(fields)
	return r.newTemporalDurationObject(fields, r.getTemporalDurationPrototype())
}

func (r *Runtime) temporalPlainTimeRound(call FunctionCall) Value {
	t := r.toTemporalPlainTime(call.This)
	opts := call.Argument(0)
	if opts == nil || IsUndefined(opts) {
		panic(r.NewTypeError("options are required"))
	}
	smallest := ""
	increment := int64(1)
	mode := temporalRoundHalfExpand
	if str, ok := opts.(String); ok {
		u, valid := normalizeTemporalUnit(str.String())
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
	if err := validateTimeRoundingIncrement(smallest, increment); err != nil {
		panic(r.newError(r.getRangeError(), err.Error()))
	}
	unit := big.NewInt(temporalUnitNanos(smallest) * increment)
	rounded := temporalRoundNanos(big.NewInt(t.nanosOfDay()), unit, mode)
	dayNs := big.NewInt(86400e9)
	rounded.Mod(rounded, dayNs)
	return r.plainTimeFromNanos(rounded.Int64())
}

func (r *Runtime) temporalPlainTimeEquals(call FunctionCall) Value {
	a := r.toTemporalPlainTime(call.This)
	b := r.temporalPlainTimeFromValue(call.Argument(0), _undefined)
	return r.ToValue(a.nanosOfDay() == b.nanosOfDay())
}

func (r *Runtime) temporalPlainTimeCompare(call FunctionCall) Value {
	a := r.temporalPlainTimeFromValue(call.Argument(0), _undefined)
	b := r.temporalPlainTimeFromValue(call.Argument(1), _undefined)
	an, bn := a.nanosOfDay(), b.nanosOfDay()
	if an < bn {
		return intToValue(-1)
	}
	if an > bn {
		return intToValue(1)
	}
	return intToValue(0)
}

func (r *Runtime) formatTemporalTime(t *temporalPlainTimeObject, options Value) string {
	p := r.temporalGetStringPrecision(options, nil)
	tm := isoTime{
		hour:        t.hour,
		minute:      t.minute,
		second:      t.second,
		millisecond: t.millisecond,
		microsecond: t.microsecond,
		nanosecond:  t.nanosecond,
	}
	if p.hasUnit {
		_, tm = r.roundISODateTime(isoDate{year: 2000, month: 1, day: 1}, tm, p.unit, 1, p.mode)
	} else if p.digits >= 0 {
		if p.digits == 0 {
			_, tm = r.roundISODateTime(isoDate{year: 2000, month: 1, day: 1}, tm, "second", 1, p.mode)
		} else {
			inc := int64(1)
			for i := 0; i < 9-p.digits; i++ {
				inc *= 10
			}
			_, tm = r.roundISODateTime(isoDate{year: 2000, month: 1, day: 1}, tm, "nanosecond", inc, p.mode)
		}
	}
	return formatISOTimePrecision(tm, p)
}

func (r *Runtime) temporalPlainTimeToString(call FunctionCall) Value {
	t := r.toTemporalPlainTime(call.This)
	return r.ToValue(r.formatTemporalTime(t, call.Argument(0)))
}

func (r *Runtime) temporalPlainTimeToJSON(call FunctionCall) Value {
	t := r.toTemporalPlainTime(call.This)
	return r.ToValue(r.formatTemporalTime(t, _undefined))
}

func (r *Runtime) temporalPlainTimeToLocaleString(call FunctionCall) Value {
	t := r.toTemporalPlainTime(call.This)
	return r.ToValue(r.formatTemporalTime(t, _undefined))
}

func (r *Runtime) temporalPlainTimeValueOf(call FunctionCall) Value {
	panic(r.NewTypeError("Temporal.PlainTime objects cannot be converted to primitives"))
}

func (r *Runtime) temporalPlainTimeGetISOFields(call FunctionCall) Value {
	t := r.toTemporalPlainTime(call.This)
	o := r.NewObject()
	o.Set("isoHour", r.ToValue(t.hour))
	o.Set("isoMinute", r.ToValue(t.minute))
	o.Set("isoSecond", r.ToValue(t.second))
	o.Set("isoMillisecond", r.ToValue(t.millisecond))
	o.Set("isoMicrosecond", r.ToValue(t.microsecond))
	o.Set("isoNanosecond", r.ToValue(t.nanosecond))
	return o
}

func (r *Runtime) temporalPlainTimeToPlainDateTime(call FunctionCall) Value {
	t := r.toTemporalPlainTime(call.This)
	d := r.temporalPlainDateFromValue(call.Argument(0), _undefined)
	tm := isoTime{
		hour:        t.hour,
		minute:      t.minute,
		second:      t.second,
		millisecond: t.millisecond,
		microsecond: t.microsecond,
		nanosecond:  t.nanosecond,
	}
	if !d.date.isValidDateTime(tm) {
		panic(r.newError(r.getRangeError(), "Resulting date-time is out of range"))
	}
	return r.newTemporalPlainDateTimeObject(d.date, tm, d.calendar, r.getTemporalPlainDateTimePrototype())
}

func (r *Runtime) temporalPlainTimeNotImplemented(call FunctionCall) Value {
	panic(r.NewTypeError("This Temporal.PlainTime method is not implemented yet"))
}
