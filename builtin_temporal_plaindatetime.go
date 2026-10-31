package goja

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/dop251/goja/unistring"
)

type isoTime struct {
	hour, minute, second                 int64
	millisecond, microsecond, nanosecond int64
}

func (t isoTime) nanos() int64 {
	return ((t.hour*60+t.minute)*60+t.second)*1e9 + t.millisecond*1e6 + t.microsecond*1e3 + t.nanosecond
}

func isoTimeFromNanos(ns int64) isoTime {
	ns = ((ns % 86400e9) + 86400e9) % 86400e9
	return isoTime{
		hour:        ns / 3600e9,
		minute:      (ns % 3600e9) / 60e9,
		second:      (ns % 60e9) / 1e9,
		millisecond: (ns % 1e9) / 1e6,
		microsecond: (ns % 1e6) / 1e3,
		nanosecond:  ns % 1e3,
	}
}

type temporalPlainDateTimeObject struct {
	baseObject
	date     isoDate
	time     isoTime
	calendar string
}

func (r *Runtime) getTemporalPlainDateTime() *Object {
	ret := r.global.TemporalPlainDateTime
	if ret == nil {
		ret = &Object{runtime: r}
		r.global.TemporalPlainDateTime = ret
		r.newNativeFuncAndConstruct(ret, func(call FunctionCall) Value {
			panic(r.NewTypeError("Constructor Temporal.PlainDateTime requires 'new'"))
		}, func(args []Value, newTarget *Object) *Object {
			y := r.temporalToIntegerWithTruncation(args, 0)
			mo := r.temporalToIntegerWithTruncation(args, 1)
			d := r.temporalToIntegerWithTruncation(args, 2)
			time := isoTime{
				hour:        r.temporalTimeArgument(args, 3),
				minute:      r.temporalTimeArgument(args, 4),
				second:      r.temporalTimeArgument(args, 5),
				millisecond: r.temporalTimeArgument(args, 6),
				microsecond: r.temporalTimeArgument(args, 7),
				nanosecond:  r.temporalTimeArgument(args, 8),
			}
			calendar := "iso8601"
			if len(args) > 9 && !IsUndefined(args[9]) {
				calendar = r.temporalCalendarID(args[9])
			}
			date := isoDate{year: y, month: mo, day: d}
			if !date.isValidDateTime(time) {
				panic(r.newError(r.getRangeError(), "Invalid date/time"))
			}
			proto := r.getTemporalPlainDateTimePrototype()
			if newTarget != nil {
				proto = r.getPrototypeFromCtor(newTarget, ret, proto)
			}
			return r.newTemporalPlainDateTimeObject(date, time, calendar, proto)
		}, r.getTemporalPlainDateTimePrototype(), "PlainDateTime", intToValue(3))
		ctor := ret.self.(*nativeFuncObject)
		ctor._putProp("from", r.newNativeFunc(r.temporalPlainDateTimeFrom, "from", 1), true, false, true)
		ctor._putProp("compare", r.newNativeFunc(r.temporalPlainDateTimeCompare, "compare", 2), true, false, true)
	}
	return ret
}

func (t isoTime) isValid() bool {
	return t.hour >= 0 && t.hour <= 23 && t.minute >= 0 && t.minute <= 59 &&
		t.second >= 0 && t.second <= 59 && t.millisecond >= 0 && t.millisecond <= 999 &&
		t.microsecond >= 0 && t.microsecond <= 999 && t.nanosecond >= 0 && t.nanosecond <= 999
}

func (r *Runtime) getTemporalPlainDateTimePrototype() *Object {
	ret := r.global.TemporalPlainDateTimePrototype
	if ret == nil {
		o := r.newBaseObject(r.global.ObjectPrototype, classObject)
		ret = o.val
		r.global.TemporalPlainDateTimePrototype = ret
		o._putProp("constructor", r.getTemporalPlainDateTime(), true, false, true)
		addGetter := func(name string, f func(*temporalPlainDateTimeObject) Value) {
			o.setOwnStr(unistring.NewFromString(name), &valueProperty{
				getterFunc: r.newNativeFunc(func(call FunctionCall) Value {
					return f(r.toTemporalPlainDateTime(call.This))
				}, unistring.NewFromString("get "+name), 0),
				accessor: true, enumerable: false, configurable: true,
			}, true)
		}
		addGetter("year", func(d *temporalPlainDateTimeObject) Value { return intToValue(d.date.year) })
		addGetter("month", func(d *temporalPlainDateTimeObject) Value { return intToValue(d.date.month) })
		addGetter("day", func(d *temporalPlainDateTimeObject) Value { return intToValue(d.date.day) })
		addGetter("calendarId", func(d *temporalPlainDateTimeObject) Value { return r.ToValue(d.calendar) })
		addGetter("era", func(d *temporalPlainDateTimeObject) Value { return _undefined })
		addGetter("eraYear", func(d *temporalPlainDateTimeObject) Value { return _undefined })
		addGetter("monthCode", func(d *temporalPlainDateTimeObject) Value { return asciiString(fmt.Sprintf("M%02d", d.date.month)) })
		addGetter("dayOfWeek", func(d *temporalPlainDateTimeObject) Value { return intToValue(((d.date.days()+3)%7+7)%7 + 1) })
		addGetter("dayOfYear", func(d *temporalPlainDateTimeObject) Value {
			return intToValue(d.date.days() - isoDaysFromCivil(d.date.year, 1, 1) + 1)
		})
		addGetter("weekOfYear", func(d *temporalPlainDateTimeObject) Value { w, _ := d.date.weekInfo(); return intToValue(w) })
		addGetter("yearOfWeek", func(d *temporalPlainDateTimeObject) Value { _, y := d.date.weekInfo(); return intToValue(y) })
		addGetter("daysInWeek", func(d *temporalPlainDateTimeObject) Value { return intToValue(7) })
		addGetter("daysInMonth", func(d *temporalPlainDateTimeObject) Value {
			return intToValue(isoDaysInMonth(d.date.year, d.date.month))
		})
		addGetter("daysInYear", func(d *temporalPlainDateTimeObject) Value {
			if isoLeapYear(d.date.year) {
				return intToValue(366)
			}
			return intToValue(365)
		})
		addGetter("monthsInYear", func(d *temporalPlainDateTimeObject) Value { return intToValue(12) })
		addGetter("inLeapYear", func(d *temporalPlainDateTimeObject) Value { return r.ToValue(isoLeapYear(d.date.year)) })
		addGetter("hour", func(d *temporalPlainDateTimeObject) Value { return intToValue(d.time.hour) })
		addGetter("minute", func(d *temporalPlainDateTimeObject) Value { return intToValue(d.time.minute) })
		addGetter("second", func(d *temporalPlainDateTimeObject) Value { return intToValue(d.time.second) })
		addGetter("millisecond", func(d *temporalPlainDateTimeObject) Value { return intToValue(d.time.millisecond) })
		addGetter("microsecond", func(d *temporalPlainDateTimeObject) Value { return intToValue(d.time.microsecond) })
		addGetter("nanosecond", func(d *temporalPlainDateTimeObject) Value { return intToValue(d.time.nanosecond) })

		o._putProp("with", r.newNativeFunc(r.temporalPlainDateTimeWith, "with", 1), true, false, true)
		o._putProp("add", r.newNativeFunc(r.temporalPlainDateTimeAdd, "add", 1), true, false, true)
		o._putProp("subtract", r.newNativeFunc(r.temporalPlainDateTimeSubtract, "subtract", 1), true, false, true)
		o._putProp("until", r.newNativeFunc(r.temporalPlainDateTimeUntil, "until", 1), true, false, true)
		o._putProp("since", r.newNativeFunc(r.temporalPlainDateTimeSince, "since", 1), true, false, true)
		o._putProp("round", r.newNativeFunc(r.temporalPlainDateTimeRound, "round", 1), true, false, true)
		o._putProp("equals", r.newNativeFunc(r.temporalPlainDateTimeEquals, "equals", 1), true, false, true)
		o._putProp("toString", r.newNativeFunc(r.temporalPlainDateTimeToString, "toString", 0), true, false, true)
		o._putProp("toJSON", r.newNativeFunc(r.temporalPlainDateTimeToJSON, "toJSON", 0), true, false, true)
		o._putProp("toLocaleString", r.newNativeFunc(r.temporalPlainDateTimeToLocaleString, "toLocaleString", 0), true, false, true)
		o._putProp("valueOf", r.newNativeFunc(r.temporalPlainDateTimeValueOf, "valueOf", 0), true, false, true)
		o._putProp("toPlainDate", r.newNativeFunc(r.temporalPlainDateTimeToPlainDate, "toPlainDate", 0), true, false, true)
		o._putProp("toPlainYearMonth", r.newNativeFunc(r.temporalPlainDateTimeToPlainYearMonth, "toPlainYearMonth", 0), true, false, true)
		o._putProp("toPlainMonthDay", r.newNativeFunc(r.temporalPlainDateTimeToPlainMonthDay, "toPlainMonthDay", 0), true, false, true)
		o._putProp("toPlainTime", r.newNativeFunc(r.temporalPlainDateTimeToPlainTime, "toPlainTime", 0), true, false, true)
		o._putProp("withCalendar", r.newNativeFunc(r.temporalPlainDateTimeWithCalendar, "withCalendar", 1), true, false, true)
		o._putProp("withPlainTime", r.newNativeFunc(r.temporalPlainDateTimeWithPlainTime, "withPlainTime", 0), true, false, true)
		o._putProp("getISOFields", r.newNativeFunc(r.temporalPlainDateTimeGetISOFields, "getISOFields", 0), true, false, true)
		o._putProp("toZonedDateTime", r.newNativeFunc(r.temporalPlainDateTimeToZonedDateTime, "toZonedDateTime", 1), true, false, true)
		o._putSym(SymToStringTag, &valueProperty{
			value: asciiString("Temporal.PlainDateTime"), writable: false, enumerable: false, configurable: true,
		})
	}
	return ret
}

func (r *Runtime) newTemporalPlainDateTimeObject(date isoDate, time isoTime, calendar string, proto *Object) *Object {
	v := &Object{runtime: r}
	o := &temporalPlainDateTimeObject{date: date, time: time, calendar: calendar}
	o.baseObject = baseObject{class: classObject, val: v, extensible: true, prototype: proto}
	v.self = o
	o.init()
	return v
}

func (r *Runtime) toTemporalPlainDateTime(v Value) *temporalPlainDateTimeObject {
	o := r.toObject(v)
	if d, ok := o.self.(*temporalPlainDateTimeObject); ok {
		return d
	}
	panic(r.NewTypeError("Value is not a Temporal.PlainDateTime"))
}

var temporalDateTimeFields = []struct {
	name  string
	date  bool
	index int
}{
	{"calendar", false, -1},
	{"day", true, 2},
	{"hour", false, 0},
	{"microsecond", false, 4},
	{"millisecond", false, 3},
	{"minute", false, 1},
	{"month", true, 1},
	{"monthCode", true, 3},
	{"nanosecond", false, 5},
	{"second", false, 2},
	{"year", true, 0},
}

func (r *Runtime) temporalPlainDateTimeFrom(call FunctionCall) Value {
	d := r.temporalPlainDateTimeFromValue(call.Argument(0), call.Argument(1))
	return r.newTemporalPlainDateTimeObject(d.date, d.time, d.calendar, r.getTemporalPlainDateTimePrototype())
}

func (r *Runtime) temporalPlainDateTimeFromValue(v Value, options Value) *temporalPlainDateTimeObject {
	if o, ok := v.(*Object); ok {
		if d, ok := o.self.(*temporalPlainDateTimeObject); ok {
			_ = r.temporalOverflowOption(options)
			return d
		}
		if pd, ok := o.self.(*temporalPlainDateObject); ok {
			_ = r.temporalOverflowOption(options)
			return r.plainDateTimeFieldsObject(pd.date, isoTime{}, pd.calendar)
		}
		if z, ok := o.self.(*temporalZonedDateTimeObject); ok {
			_ = r.temporalOverflowOption(options)
			d, tm, _ := z.zone.localDateTime(z.epochNs)
			return r.plainDateTimeFieldsObject(d, tm, z.calendar)
		}
	}
	if s, ok := v.(String); ok {
		date, tm, cal, ok := parseTemporalPlainDateTimeString(s.String())
		if !ok {
			panic(r.newError(r.getRangeError(), "Invalid ISO date-time string: "+s.String()))
		}
		_ = r.temporalOverflowOption(options)
		return r.plainDateTimeFieldsObject(date, tm, cal)
	}
	obj, ok := v.(*Object)
	if !ok {
		panic(r.NewTypeError("Cannot convert %s to Temporal.PlainDateTime", v))
	}
	res := r.temporalReadDateTimeFields(obj, temporalDateTimeFieldsResult{calendar: "iso8601"}, false, false)
	overflow := r.temporalOverflowOption(options)
	if !res.haveYear || (!res.haveMonth && !res.hasMonthCd) || !res.haveDay {
		panic(r.NewTypeError("year, month and day are required"))
	}
	if res.hasMonthCd {
		n, ok := isoMonthFromMonthCode(res.monthCode)
		if !ok {
			panic(r.newError(r.getRangeError(), "Invalid monthCode"))
		}
		if res.monthSet && res.date.month != n {
			panic(r.newError(r.getRangeError(), "month and monthCode must agree"))
		}
		res.date.month = n
	}
	date, tm := res.date, res.time
	if overflow == "constrain" {
		date.month = clampInt(date.month, 1, 12)
		date.day = clampInt(date.day, 1, isoDaysInMonth(date.year, date.month))
		tm.hour = clampInt(tm.hour, 0, 23)
		tm.minute = clampInt(tm.minute, 0, 59)
		tm.second = clampInt(tm.second, 0, 59)
		tm.millisecond = clampInt(tm.millisecond, 0, 999)
		tm.microsecond = clampInt(tm.microsecond, 0, 999)
		tm.nanosecond = clampInt(tm.nanosecond, 0, 999)
	}
	if !date.isValidDateTime(tm) {
		panic(r.newError(r.getRangeError(), "Invalid date/time"))
	}
	return r.plainDateTimeFieldsObject(date, tm, res.calendar)
}

func (r *Runtime) plainDateTimeFieldsObject(date isoDate, tm isoTime, calendar string) *temporalPlainDateTimeObject {
	o := r.newTemporalPlainDateTimeObject(date, tm, calendar, r.getTemporalPlainDateTimePrototype())
	return o.self.(*temporalPlainDateTimeObject)
}

func parseTemporalPlainDateTimeString(s string) (isoDate, isoTime, string, bool) {
	var date isoDate
	var tm isoTime
	calendar := "iso8601"
	// Strip annotations, using the PlainDate rules for calendar annotations.
	base := s
	var annotations []string
	if k := strings.IndexByte(s, '['); k >= 0 {
		rest := s[k:]
		for len(rest) > 0 {
			if rest[0] != '[' {
				return date, tm, "", false
			}
			j := strings.IndexByte(rest, ']')
			if j < 0 {
				return date, tm, "", false
			}
			annotations = append(annotations, rest[1:j])
			rest = rest[j+1:]
		}
		base = strings.TrimSpace(s[:k])
	}
	calSeen := false
	calCritical := false
	tz := 0
	for _, ann := range annotations {
		critical := strings.HasPrefix(ann, "!")
		if critical {
			ann = ann[1:]
		}
		key, val, hasVal := strings.Cut(ann, "=")
		if hasVal {
			if key != strings.ToLower(key) || key == "" {
				return date, tm, "", false
			}
			if key == "u-ca" {
				if !calSeen {
					if !asciiEqualFold(val, "iso8601") {
						return date, tm, "", false
					}
					calSeen = true
					calCritical = critical
				} else if critical || calCritical {
					return date, tm, "", false
				}
				continue
			}
		}
		if isTimeZoneAnnotation(ann) {
			tz++
			if tz > 1 {
				return date, tm, "", false
			}
			continue
		}
		if critical {
			return date, tm, "", false
		}
	}
	norm := base
	sep := -1
	if i := strings.IndexAny(norm, "Tt"); i >= 0 {
		sep = i
	} else if i := strings.IndexByte(norm, ' '); i > 0 {
		sep = i
	}
	if sep > 0 {
		if nd, ok := normalizeBasicDate(norm[:sep]); ok {
			norm = nd + norm[sep:]
		}
	} else if sep < 0 {
		if nd, ok := normalizeBasicDate(norm); ok {
			norm = nd
		}
	}
	datePart := norm
	timePart := ""
	hasSep := false
	if i := strings.IndexAny(norm, "Tt"); i >= 0 {
		datePart = norm[:i]
		timePart = norm[i+1:]
		hasSep = true
	} else if i := strings.IndexByte(norm, ' '); i > 0 {
		datePart = norm[:i]
		timePart = norm[i+1:]
		hasSep = true
	}
	if hasSep && timePart == "" {
		return date, tm, "", false
	}
	d, _, ok := parseTemporalPlainDateString(datePart)
	if !ok || !d.isValid() {
		return date, tm, "", false
	}
	date = d
	if timePart != "" {
		h, mi, sec, ms, us, ns, ok := parseTemporalTimeFields(timePart)
		if !ok {
			return date, tm, "", false
		}
		tm = isoTime{h, mi, sec, ms, us, ns}
	}
	if !date.isValidDateTime(tm) {
		return date, tm, "", false
	}
	return date, tm, calendar, true
}

func (r *Runtime) temporalPlainDateTimeWith(call FunctionCall) Value {
	d := r.toTemporalPlainDateTime(call.This)
	partial := r.toObject(call.Argument(0))
	switch partial.self.(type) {
	case *temporalPlainDateObject, *temporalPlainTimeObject, *temporalPlainDateTimeObject,
		*temporalInstantObject, *temporalZonedDateTimeObject, *temporalDurationObject,
		*temporalPlainYearMonthObject, *temporalPlainMonthDayObject:
		panic(r.NewTypeError("with() argument must be a plain object"))
	}
	base := temporalDateTimeFieldsResult{
		date: d.date, time: d.time, calendar: d.calendar,
		haveYear: true, haveMonth: true, haveDay: true,
	}
	res := r.temporalReadDateTimeFields(partial, base, true, true)
	overflow := r.temporalOverflowOption(call.Argument(1))
	if !res.provided {
		panic(r.NewTypeError("PlainDateTime.with requires at least one property"))
	}
	date, tm := res.date, res.time
	if res.hasMonthCd {
		n, ok := isoMonthFromMonthCode(res.monthCode)
		if !ok {
			panic(r.newError(r.getRangeError(), "Invalid monthCode"))
		}
		if res.monthSet && res.date.month != n {
			panic(r.newError(r.getRangeError(), "month and monthCode must agree"))
		}
		date.month = n
	}
	if overflow == "constrain" {
		date.month = clampInt(date.month, 1, 12)
		date.day = clampInt(date.day, 1, isoDaysInMonth(date.year, date.month))
		tm.hour = clampInt(tm.hour, 0, 23)
		tm.minute = clampInt(tm.minute, 0, 59)
		tm.second = clampInt(tm.second, 0, 59)
		tm.millisecond = clampInt(tm.millisecond, 0, 999)
		tm.microsecond = clampInt(tm.microsecond, 0, 999)
		tm.nanosecond = clampInt(tm.nanosecond, 0, 999)
	}
	if !date.isValidDateTime(tm) {
		panic(r.newError(r.getRangeError(), "Invalid date/time"))
	}
	return r.newTemporalPlainDateTimeObject(date, tm, res.calendar, r.getTemporalPlainDateTimePrototype())
}

func (r *Runtime) temporalPlainDateTimeAdd(call FunctionCall) Value {
	return r.temporalPlainDateTimeAddSub(call, false)
}

func (r *Runtime) temporalPlainDateTimeSubtract(call FunctionCall) Value {
	return r.temporalPlainDateTimeAddSub(call, true)
}

func (r *Runtime) temporalPlainDateTimeAddSub(call FunctionCall, subtract bool) Value {
	d := r.toTemporalPlainDateTime(call.This)
	dur := r.temporalDurationFromValue(call.Argument(0))
	overflow := r.temporalOverflowOption(call.Argument(1))
	fields := dur.fields()
	sign := int64(1)
	if subtract {
		sign = -1
	}
	date := d.date
	monthsTotal := (int64(fields[0])*12 + int64(fields[1])) * sign
	ambiguous := false
	if monthsTotal != 0 {
		probe := addISOMonths(date, monthsTotal)
		if date.day > isoDaysInMonth(probe.year, probe.month) {
			ambiguous = true
		}
	}
	date = addISOMonths(date, monthsTotal)
	if ambiguous && overflow != "constrain" {
		panic(r.newError(r.getRangeError(), "Ambiguous date with overflow=reject"))
	}
	date = addISODays(date, (int64(fields[2])*7+int64(fields[3]))*sign)
	timeNs := new(big.Int).SetInt64(d.time.nanos())
	add := func(v float64, mul int64) {
		if v != 0 {
			x := new(big.Rat).Mul(new(big.Rat).SetFloat64(v), big.NewRat(mul, 1))
			n := new(big.Int).Quo(x.Num(), x.Denom())
			timeNs.Add(timeNs, n)
		}
	}
	add(fields[4]*float64(sign), 3600*1e9)
	add(fields[5]*float64(sign), 60*1e9)
	add(fields[6]*float64(sign), 1e9)
	add(fields[7]*float64(sign), 1e6)
	add(fields[8]*float64(sign), 1e3)
	add(fields[9]*float64(sign), 1)
	dayNs := big.NewInt(86400e9)
	carry := new(big.Int)
	rem := new(big.Int)
	carry.DivMod(new(big.Int).Set(timeNs), dayNs, rem)
	date = addISODays(date, carry.Int64())
	if !date.isValidDateTime(isoTimeFromNanos(rem.Int64())) {
		panic(r.newError(r.getRangeError(), "Resulting date is out of range"))
	}
	return r.newTemporalPlainDateTimeObject(date, isoTimeFromNanos(rem.Int64()), d.calendar, r.getTemporalPlainDateTimePrototype())
}

func (r *Runtime) temporalPlainDateTimeUntil(call FunctionCall) Value {
	a := r.toTemporalPlainDateTime(call.This)
	b := r.temporalPlainDateTimeFromValue(call.Argument(0), _undefined)
	return r.temporalPlainDateTimeDifference(a, b, call.Argument(1), false)
}

func (r *Runtime) temporalPlainDateTimeSince(call FunctionCall) Value {
	a := r.toTemporalPlainDateTime(call.This)
	b := r.temporalPlainDateTimeFromValue(call.Argument(0), _undefined)
	return r.temporalPlainDateTimeDifference(a, b, call.Argument(1), true)
}

func (r *Runtime) temporalPlainDateTimeDifference(a, b *temporalPlainDateTimeObject, options Value, since bool) Value {
	s := r.temporalGetDifferenceSettings(options, "day", "nanosecond", 0, 9)
	if since {
		s.mode = negateRoundingMode(s.mode)
	}
	fields := r.temporalDiffISODateTime(a.date, a.time, b.date, b.time, s)
	if since {
		for i := range fields {
			fields[i] = -fields[i]
		}
	}
	r.validateTemporalDuration(fields)
	return r.newTemporalDurationObject(fields, r.getTemporalDurationPrototype())
}

func (r *Runtime) temporalPlainDateTimeRound(call FunctionCall) Value {
	d := r.toTemporalPlainDateTime(call.This)
	opts := call.Argument(0)
	if opts == nil || IsUndefined(opts) || IsNull(opts) {
		panic(r.NewTypeError("options are required"))
	}
	var smallest string
	increment := int64(1)
	mode := temporalRoundHalfExpand
	if s, ok := opts.(String); ok {
		u, valid := normalizeTemporalUnit(s.String())
		if !valid || temporalUnitIndex(u) < 3 {
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
		if !valid || temporalUnitIndex(u) < 3 {
			panic(r.newError(r.getRangeError(), "Invalid smallestUnit"))
		}
		smallest = u
	}
	if smallest == "day" {
		if increment != 1 {
			panic(r.newError(r.getRangeError(), "invalid rounding increment"))
		}
	} else if err := validateTimeRoundingIncrement(smallest, increment); err != nil {
		panic(r.newError(r.getRangeError(), err.Error()))
	}
	nd, nt := r.roundISODateTime(d.date, d.time, smallest, increment, mode)
	return r.newTemporalPlainDateTimeObject(nd, nt, d.calendar, r.getTemporalPlainDateTimePrototype())
}

func (r *Runtime) temporalPlainDateTimeEquals(call FunctionCall) Value {
	a := r.toTemporalPlainDateTime(call.This)
	b := r.temporalPlainDateTimeFromValue(call.Argument(0), _undefined)
	return r.ToValue(a.date == b.date && a.time == b.time && a.calendar == b.calendar)
}

func (r *Runtime) temporalPlainDateTimeCompare(call FunctionCall) Value {
	a := r.temporalPlainDateTimeFromValue(call.Argument(0), _undefined)
	b := r.temporalPlainDateTimeFromValue(call.Argument(1), _undefined)
	ad, bd := a.date.days(), b.date.days()
	if ad != bd {
		if ad < bd {
			return intToValue(-1)
		}
		return intToValue(1)
	}
	at, bt := a.time.nanos(), b.time.nanos()
	if at < bt {
		return intToValue(-1)
	}
	if at > bt {
		return intToValue(1)
	}
	return intToValue(0)
}

func (r *Runtime) temporalPlainDateTimeToString(call FunctionCall) Value {
	d := r.toTemporalPlainDateTime(call.This)
	calendarName := "auto"
	options := call.Argument(0)
	if options != nil && !IsUndefined(options) {
		if obj, ok := options.(*Object); ok {
			if v := obj.self.getStr("calendarName", nil); v != nil && !IsUndefined(v) {
				s, _ := temporalToStringOption(r, v)
				switch s {
				case "auto", "always", "never", "critical":
					calendarName = s
				default:
					panic(r.newError(r.getRangeError(), "Invalid calendarName"))
				}
			}
		}
	}
	p := r.temporalGetStringPrecision(options, nil)
	date, tm := d.date, d.time
	if p.hasUnit {
		date, tm = r.roundISODateTime(date, tm, p.unit, 1, p.mode)
	} else if p.digits >= 0 {
		if p.digits == 0 {
			date, tm = r.roundISODateTime(date, tm, "second", 1, p.mode)
		} else {
			inc := int64(1)
			for i := 0; i < 9-p.digits; i++ {
				inc *= 10
			}
			date, tm = r.roundISODateTime(date, tm, "nanosecond", inc, p.mode)
		}
	}
	out := formatISODate(date) + "T" + formatISOTimePrecision(tm, p)
	switch calendarName {
	case "always":
		out += "[u-ca=" + d.calendar + "]"
	case "critical":
		out += "[!u-ca=" + d.calendar + "]"
	}
	return r.ToValue(out)
}

func (r *Runtime) temporalPlainDateTimeToJSON(call FunctionCall) Value {
	return r.temporalPlainDateTimeToString(FunctionCall{This: call.This, Arguments: []Value{_undefined}})
}

func (r *Runtime) temporalPlainDateTimeToLocaleString(call FunctionCall) Value {
	return r.temporalToLocaleString(call, "any")
}

func (r *Runtime) temporalPlainDateTimeValueOf(call FunctionCall) Value {
	panic(r.NewTypeError("Temporal.PlainDateTime objects cannot be converted to primitives"))
}

func (r *Runtime) temporalPlainDateTimeToPlainDate(call FunctionCall) Value {
	d := r.toTemporalPlainDateTime(call.This)
	return r.newTemporalPlainDateObject(d.date, d.calendar, r.getTemporalPlainDatePrototype())
}

func (r *Runtime) temporalPlainDateTimeToPlainYearMonth(call FunctionCall) Value {
	d := r.toTemporalPlainDateTime(call.This)
	return r.newTemporalPlainYearMonthObject(d.date.year, d.date.month, 1, d.calendar, r.getTemporalPlainYearMonthPrototype())
}

func (r *Runtime) temporalPlainDateTimeToPlainMonthDay(call FunctionCall) Value {
	d := r.toTemporalPlainDateTime(call.This)
	return r.newTemporalPlainMonthDayObject(d.date.month, d.date.day, d.date.year, d.calendar, r.getTemporalPlainMonthDayPrototype())
}

func (r *Runtime) temporalPlainDateTimeToPlainTime(call FunctionCall) Value {
	d := r.toTemporalPlainDateTime(call.This)
	return r.newTemporalPlainTimeObject(d.time.hour, d.time.minute, d.time.second, d.time.millisecond, d.time.microsecond, d.time.nanosecond, r.getTemporalPlainTimePrototype())
}

func (r *Runtime) temporalPlainDateTimeWithCalendar(call FunctionCall) Value {
	d := r.toTemporalPlainDateTime(call.This)
	cal := r.temporalCalendarFromValue(call.Argument(0))
	return r.newTemporalPlainDateTimeObject(d.date, d.time, cal, r.getTemporalPlainDateTimePrototype())
}

func (r *Runtime) temporalPlainDateTimeWithPlainTime(call FunctionCall) Value {
	d := r.toTemporalPlainDateTime(call.This)
	v := call.Argument(0)
	var tm isoTime
	if v == nil || IsUndefined(v) {
		tm = isoTime{}
	} else {
		t := r.temporalPlainTimeFromValue(v, _undefined)
		tm = isoTime{
			hour:        t.hour,
			minute:      t.minute,
			second:      t.second,
			millisecond: t.millisecond,
			microsecond: t.microsecond,
			nanosecond:  t.nanosecond,
		}
	}
	if !d.date.isValidDateTime(tm) {
		panic(r.newError(r.getRangeError(), "Resulting date-time is out of range"))
	}
	return r.newTemporalPlainDateTimeObject(d.date, tm, d.calendar, r.getTemporalPlainDateTimePrototype())
}

func (r *Runtime) temporalPlainDateTimeToZonedDateTime(call FunctionCall) Value {
	d := r.toTemporalPlainDateTime(call.This)
	arg := call.Argument(0)
	var zone *temporalZone
	if o, ok := arg.(*Object); ok {
		if z, ok := o.self.(*temporalZonedDateTimeObject); ok {
			zone = z.zone
		} else {
			v := o.self.getStr("timeZone", nil)
			if v == nil || IsUndefined(v) {
				panic(r.NewTypeError("timeZone is required"))
			}
			zone = r.parseTemporalZone(v)
		}
	} else {
		zone = r.parseTemporalZone(arg)
	}
	optsObj := r.temporalOptionsObject(call.Argument(1))
	disambiguation := r.temporalGetDisambiguationOption(optsObj)
	local := new(big.Int).Mul(big.NewInt(d.date.days()), temporalBigDayNs)
	local.Add(local, big.NewInt(d.time.nanos()))
	ns := r.zonedResolve(zone, local, nil, false, disambiguation, "reject")
	if ns == nil || !r.validInstant(ns) {
		panic(r.newError(r.getRangeError(), "ZonedDateTime is out of range"))
	}
	return r.newTemporalZonedDateTimeObject(ns, zone, d.calendar, r.getTemporalZonedDateTimePrototype())
}

func (r *Runtime) temporalPlainDateTimeGetISOFields(call FunctionCall) Value {
	d := r.toTemporalPlainDateTime(call.This)
	o := r.NewObject()
	o.Set("isoYear", r.ToValue(d.date.year))
	o.Set("isoMonth", r.ToValue(d.date.month))
	o.Set("isoDay", r.ToValue(d.date.day))
	o.Set("isoHour", r.ToValue(d.time.hour))
	o.Set("isoMinute", r.ToValue(d.time.minute))
	o.Set("isoSecond", r.ToValue(d.time.second))
	o.Set("isoMillisecond", r.ToValue(d.time.millisecond))
	o.Set("isoMicrosecond", r.ToValue(d.time.microsecond))
	o.Set("isoNanosecond", r.ToValue(d.time.nanosecond))
	o.Set("calendar", r.ToValue(d.calendar))
	return o
}
