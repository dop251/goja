package goja

import (
	"fmt"
	"strings"

	"github.com/dop251/goja/unistring"
)

type temporalPlainMonthDayObject struct {
	baseObject
	month, day, refYear int64
	calendar            string
}

func (r *Runtime) getTemporalPlainMonthDay() *Object {
	ret := r.global.TemporalPlainMonthDay
	if ret == nil {
		ret = &Object{runtime: r}
		r.global.TemporalPlainMonthDay = ret
		r.newNativeFuncAndConstruct(ret, func(call FunctionCall) Value {
			panic(r.NewTypeError("Constructor Temporal.PlainMonthDay requires 'new'"))
		}, func(args []Value, newTarget *Object) *Object {
			arg := func(i int) Value {
				if i < len(args) {
					return args[i]
				}
				return _undefined
			}
			m := r.temporalPositiveIntegerValue(arg(0))
			d := r.temporalPositiveIntegerValue(arg(1))
			calendar := "iso8601"
			if !IsUndefined(arg(2)) && arg(2) != nil {
				calendar = r.temporalCalendarID(arg(2))
			}
			refYear := int64(1972)
			if !IsUndefined(arg(3)) && arg(3) != nil {
				refYear = r.temporalTimeFieldValue(arg(3))
			}
			if m < 1 || m > 12 || d < 1 || d > isoDaysInMonth(refYear, m) ||
				!(isoDate{year: refYear, month: m, day: d}.isValid()) {
				panic(r.newError(r.getRangeError(), "Invalid month-day"))
			}
			proto := r.getTemporalPlainMonthDayPrototype()
			if newTarget != nil {
				proto = r.getPrototypeFromCtor(newTarget, ret, proto)
			}
			return r.newTemporalPlainMonthDayObject(m, d, refYear, calendar, proto)
		}, r.getTemporalPlainMonthDayPrototype(), "PlainMonthDay", intToValue(2))
		ctor := ret.self.(*nativeFuncObject)
		ctor._putProp("from", r.newNativeFunc(r.temporalPlainMonthDayFrom, "from", 1), true, false, true)
	}
	return ret
}

func (r *Runtime) getTemporalPlainMonthDayPrototype() *Object {
	ret := r.global.TemporalPlainMonthDayPrototype
	if ret == nil {
		o := r.newBaseObject(r.global.ObjectPrototype, classObject)
		ret = o.val
		r.global.TemporalPlainMonthDayPrototype = ret
		o._putProp("constructor", r.getTemporalPlainMonthDay(), true, false, true)
		addGetter := func(name string, f func(*temporalPlainMonthDayObject) Value) {
			o.setOwnStr(unistring.NewFromString(name), &valueProperty{
				getterFunc: r.newNativeFunc(func(call FunctionCall) Value {
					return f(r.toTemporalPlainMonthDay(call.This))
				}, unistring.NewFromString("get "+name), 0),
				accessor: true, enumerable: false, configurable: true,
			}, true)
		}
		addGetter("monthCode", func(m *temporalPlainMonthDayObject) Value {
			return asciiString(fmt.Sprintf("M%02d", m.month))
		})
		addGetter("day", func(m *temporalPlainMonthDayObject) Value { return intToValue(m.day) })
		addGetter("calendarId", func(m *temporalPlainMonthDayObject) Value { return r.ToValue(m.calendar) })
		addGetter("era", func(m *temporalPlainMonthDayObject) Value { return _undefined })
		addGetter("eraYear", func(m *temporalPlainMonthDayObject) Value { return _undefined })
		o._putProp("with", r.newNativeFunc(r.temporalPlainMonthDayWith, "with", 1), true, false, true)
		o._putProp("equals", r.newNativeFunc(r.temporalPlainMonthDayEquals, "equals", 1), true, false, true)
		o._putProp("toString", r.newNativeFunc(r.temporalPlainMonthDayToString, "toString", 0), true, false, true)
		o._putProp("toJSON", r.newNativeFunc(r.temporalPlainMonthDayToJSON, "toJSON", 0), true, false, true)
		o._putProp("toLocaleString", r.newNativeFunc(r.temporalPlainMonthDayToLocaleString, "toLocaleString", 0), true, false, true)
		o._putProp("valueOf", r.newNativeFunc(r.temporalPlainMonthDayValueOf, "valueOf", 0), true, false, true)
		o._putProp("toPlainDate", r.newNativeFunc(r.temporalPlainMonthDayToPlainDate, "toPlainDate", 1), true, false, true)
		o._putProp("getISOFields", r.newNativeFunc(r.temporalPlainMonthDayGetISOFields, "getISOFields", 0), true, false, true)
		o._putSym(SymToStringTag, &valueProperty{
			value: asciiString("Temporal.PlainMonthDay"), writable: false, enumerable: false, configurable: true,
		})
	}
	return ret
}

func (r *Runtime) newTemporalPlainMonthDayObject(m, d, refYear int64, calendar string, proto *Object) *Object {
	v := &Object{runtime: r}
	o := &temporalPlainMonthDayObject{month: m, day: d, refYear: refYear, calendar: calendar}
	o.baseObject = baseObject{class: classObject, val: v, extensible: true, prototype: proto}
	v.self = o
	o.init()
	return v
}

func (r *Runtime) toTemporalPlainMonthDay(v Value) *temporalPlainMonthDayObject {
	o := r.toObject(v)
	if m, ok := o.self.(*temporalPlainMonthDayObject); ok {
		return m
	}
	panic(r.NewTypeError("Value is not a Temporal.PlainMonthDay"))
}

func (r *Runtime) temporalPlainMonthDayFrom(call FunctionCall) Value {
	m := r.temporalPlainMonthDayFromValue(call.Argument(0), call.Argument(1))
	return r.newTemporalPlainMonthDayObject(m.month, m.day, m.refYear, m.calendar, r.getTemporalPlainMonthDayPrototype())
}

func (r *Runtime) temporalPlainMonthDayFromValue(v Value, options Value) *temporalPlainMonthDayObject {
	if o, ok := v.(*Object); ok {
		if m, ok := o.self.(*temporalPlainMonthDayObject); ok {
			_ = r.temporalOverflowOption(options)
			return m
		}
	}
	if s, ok := v.(String); ok {
		m, d, refYear, cal, ok := parseTemporalMonthDayString(s.String())
		if !ok {
			panic(r.newError(r.getRangeError(), "Invalid ISO month-day string: "+s.String()))
		}
		_ = r.temporalOverflowOption(options)
		return r.plainMonthDayFieldsObject(m, d, refYear, cal)
	}
	obj, ok := v.(*Object)
	if !ok {
		panic(r.NewTypeError("Cannot convert %s to Temporal.PlainMonthDay", v))
	}
	calendar := "iso8601"
	if cv := obj.self.getStr("calendar", nil); cv != nil && !IsUndefined(cv) {
		calendar = r.temporalCalendarFromValue(cv)
	}
	var day, month int64
	var monthCode string
	haveDay, haveMonth, hasMonthCode, hasYear := false, false, false, false
	refYear := int64(1972)
	if dv := obj.self.getStr("day", nil); dv != nil && !IsUndefined(dv) {
		day = r.temporalPositiveIntegerValue(dv)
		haveDay = true
	}
	if mv := obj.self.getStr("month", nil); mv != nil && !IsUndefined(mv) {
		month = r.temporalPositiveIntegerValue(mv)
		haveMonth = true
	}
	if mcv := obj.self.getStr("monthCode", nil); mcv != nil && !IsUndefined(mcv) {
		monthCode = r.temporalRequireString(mcv)
		if !wellFormedMonthCode(monthCode) {
			panic(r.newError(r.getRangeError(), "Invalid monthCode"))
		}
		hasMonthCode = true
	}
	if yv := obj.self.getStr("year", nil); yv != nil && !IsUndefined(yv) {
		refYear = r.temporalTimeFieldValue(yv)
		hasYear = true
	}
	overflow := r.temporalOverflowOption(options)
	if !haveDay || (!haveMonth && !hasMonthCode) {
		panic(r.NewTypeError("day and month are required"))
	}
	if hasMonthCode {
		n, ok := isoMonthFromMonthCode(monthCode)
		if !ok {
			panic(r.newError(r.getRangeError(), "Invalid monthCode"))
		}
		if haveMonth && month != n {
			panic(r.newError(r.getRangeError(), "month and monthCode must agree"))
		}
		month = n
	}
	if overflow == "constrain" {
		if month < 1 {
			panic(r.newError(r.getRangeError(), "Invalid month"))
		}
		if month > 12 {
			month = 12
		}
		if day < 1 {
			panic(r.newError(r.getRangeError(), "Invalid day"))
		}
		day = clampInt(day, 1, isoDaysInMonth(refYear, month))
	}
	if month < 1 || month > 12 || day < 1 || day > isoDaysInMonth(refYear, month) {
		panic(r.newError(r.getRangeError(), "Invalid month-day"))
	}
	_ = hasYear
	// The reference year is always the ISO leap year; a supplied year is used
	// only to validate overflow.
	return r.plainMonthDayFieldsObject(month, day, 1972, calendar)
}

func (r *Runtime) plainMonthDayFieldsObject(m, d, refYear int64, calendar string) *temporalPlainMonthDayObject {
	return r.newTemporalPlainMonthDayObject(m, d, refYear, calendar, r.getTemporalPlainMonthDayPrototype()).self.(*temporalPlainMonthDayObject)
}

// parseTemporalMonthDayString parses MM-DD, YYYY-MM-DD or MMDD with optional
// annotations.
func parseTemporalMonthDayString(s string) (month, day, refYear int64, calendar string, ok bool) {
	base, anns, aok := splitTemporalAnnotations(s)
	if !aok {
		return
	}
	calendar = "iso8601"
	calSeen := false
	calCritical := false
	tz := 0
	for _, ann := range anns {
		critical := strings.HasPrefix(ann, "!")
		if critical {
			ann = ann[1:]
		}
		key, val, hasVal := strings.Cut(ann, "=")
		if hasVal {
			if key != strings.ToLower(key) || key == "" {
				return 0, 0, 0, "", false
			}
			if key == "u-ca" {
				if calSeen && (critical || calCritical) {
					return 0, 0, 0, "", false
				}
				if !calSeen && !asciiEqualFold(val, "iso8601") {
					return 0, 0, 0, "", false
				}
				calSeen = true
				calCritical = critical
				continue
			}
		}
		if isTimeZoneAnnotation(ann) {
			tz++
			if tz > 1 {
				return 0, 0, 0, "", false
			}
			continue
		}
		if critical {
			return 0, 0, 0, "", false
		}
	}
	if i := strings.IndexAny(base, "Tt "); i > 0 {
		timePart := base[i+1:]
		if timePart == "" || strings.ContainsAny(timePart, "Zz") {
			return 0, 0, 0, "", false
		}
		if _, _, _, _, _, _, tok := parseTemporalTimeFields(timePart); !tok {
			return 0, 0, 0, "", false
		}
		base = base[:i]
	} else if strings.HasSuffix(base, "Z") || strings.HasSuffix(base, "z") {
		return 0, 0, 0, "", false
	}
	base = strings.TrimPrefix(base, "--")
	// MMDD
	if len(base) == 4 && isDigits(base) {
		m := int64(base[0]-'0')*10 + int64(base[1]-'0')
		d := int64(base[2]-'0')*10 + int64(base[3]-'0')
		if m < 1 || m > 12 || d < 1 || d > isoDaysInMonth(1972, m) {
			return 0, 0, 0, "", false
		}
		return m, d, 1972, calendar, true
	}
	// MM-DD
	if len(base) == 5 && isDigits(base[:2]) && base[2] == '-' && isDigits(base[3:]) {
		m := int64(base[0]-'0')*10 + int64(base[1]-'0')
		d := int64(base[3]-'0')*10 + int64(base[4]-'0')
		if m < 1 || m > 12 || d < 1 || d > isoDaysInMonth(1972, m) {
			return 0, 0, 0, "", false
		}
		return m, d, 1972, calendar, true
	}
	if m, d, ok := parseMonthDayDateString(base); ok {
		return m, d, 1972, calendar, true
	}
	return 0, 0, 0, "", false
}

// parseMonthDayDateString parses a full date syntactically, ignoring the year
// for range purposes (the reference year is always 1972).
func parseMonthDayDateString(base string) (month, day int64, ok bool) {
	parse2 := func(v string) (int64, bool) {
		if len(v) != 2 || !isDigits(v) {
			return 0, false
		}
		return int64(v[0]-'0')*10 + int64(v[1]-'0'), true
	}
	validate := func(m, d int64) (int64, int64, bool) {
		if m < 1 || m > 12 || d < 1 || d > isoDaysInMonth(2000, m) {
			return 0, 0, false
		}
		return m, d, true
	}
	body := base
	negYear := false
	if len(body) > 0 && (body[0] == '+' || body[0] == '-') {
		negYear = body[0] == '-'
		body = body[1:]
	}
	if negYear {
		digits := body
		if i := strings.IndexByte(digits, '-'); i > 0 {
			digits = digits[:i]
		}
		if strings.Trim(digits, "0") == "" {
			return 0, 0, false
		}
	}
	if len(body) == 8 && isDigits(body) {
		m, _ := parse2(body[4:6])
		d, _ := parse2(body[6:8])
		return validate(m, d)
	}
	if len(body) == 10 && isDigits(body) {
		m, _ := parse2(body[6:8])
		d, _ := parse2(body[8:10])
		return validate(m, d)
	}
	parts := strings.Split(body, "-")
	if len(parts) == 3 && isDigits(parts[0]) && (len(parts[0]) == 4 || len(parts[0]) == 6) {
		m, ok1 := parse2(parts[1])
		d, ok2 := parse2(parts[2])
		if ok1 && ok2 {
			return validate(m, d)
		}
	}
	return 0, 0, false
}

func (r *Runtime) temporalPlainMonthDayWith(call FunctionCall) Value {
	m := r.toTemporalPlainMonthDay(call.This)
	partial := r.toObject(call.Argument(0))
	switch partial.self.(type) {
	case *temporalPlainDateObject, *temporalPlainTimeObject, *temporalPlainDateTimeObject,
		*temporalInstantObject, *temporalZonedDateTimeObject, *temporalDurationObject,
		*temporalPlainYearMonthObject, *temporalPlainMonthDayObject:
		panic(r.NewTypeError("with() argument must be a plain object"))
	}
	month, day, refYear := m.month, m.day, m.refYear
	provided := false
	if cv := partial.self.getStr("calendar", nil); cv != nil && !IsUndefined(cv) {
		panic(r.NewTypeError("calendar is not allowed in with()"))
	}
	if tv := partial.self.getStr("timeZone", nil); tv != nil && !IsUndefined(tv) {
		panic(r.NewTypeError("timeZone is not allowed in with()"))
	}
	validateYear := refYear
	var monthCode string
	hasMonthCode, hasMonth := false, false
	if v := partial.self.getStr("day", nil); v != nil && !IsUndefined(v) {
		day = r.temporalPositiveIntegerValue(v)
		provided = true
	}
	if v := partial.self.getStr("month", nil); v != nil && !IsUndefined(v) {
		month = r.temporalPositiveIntegerValue(v)
		hasMonth = true
		provided = true
	}
	if v := partial.self.getStr("monthCode", nil); v != nil && !IsUndefined(v) {
		monthCode = r.temporalRequireString(v)
		if !wellFormedMonthCode(monthCode) {
			panic(r.newError(r.getRangeError(), "Invalid monthCode"))
		}
		hasMonthCode = true
		provided = true
	}
	if v := partial.self.getStr("year", nil); v != nil && !IsUndefined(v) {
		validateYear = r.temporalTimeFieldValue(v)
		provided = true
	}
	overflow := r.temporalOverflowOption(call.Argument(1))
	if !provided {
		panic(r.NewTypeError("PlainMonthDay.with requires at least one property"))
	}
	if hasMonthCode {
		n, ok := isoMonthFromMonthCode(monthCode)
		if !ok {
			panic(r.newError(r.getRangeError(), "Invalid monthCode"))
		}
		if hasMonth && month != n {
			panic(r.newError(r.getRangeError(), "month and monthCode must agree"))
		}
		month = n
	}
	if overflow == "constrain" {
		if month < 1 || day < 1 {
			panic(r.newError(r.getRangeError(), "Invalid month-day"))
		}
		if month > 12 {
			month = 12
		}
		day = clampInt(day, 1, isoDaysInMonth(validateYear, month))
	}
	if month < 1 || month > 12 || day < 1 || day > isoDaysInMonth(validateYear, month) {
		panic(r.newError(r.getRangeError(), "Invalid month-day"))
	}
	if day > isoDaysInMonth(m.refYear, month) {
		panic(r.newError(r.getRangeError(), "Invalid month-day"))
	}
	return r.newTemporalPlainMonthDayObject(month, day, m.refYear, m.calendar, r.getTemporalPlainMonthDayPrototype())
}

func (r *Runtime) temporalPlainMonthDayEquals(call FunctionCall) Value {
	a := r.toTemporalPlainMonthDay(call.This)
	b := r.temporalPlainMonthDayFromValue(call.Argument(0), _undefined)
	return r.ToValue(a.month == b.month && a.day == b.day && a.refYear == b.refYear && a.calendar == b.calendar)
}

func (r *Runtime) temporalPlainMonthDayToString(call FunctionCall) Value {
	m := r.toTemporalPlainMonthDay(call.This)
	_ = r.temporalOptionsObject(call.Argument(0))
	calendarName := "auto"
	if obj, ok := call.Argument(0).(*Object); ok && obj != nil {
		if v := obj.self.getStr("calendarName", nil); v != nil && !IsUndefined(v) {
			str, _ := temporalToStringOption(r, v)
			switch str {
			case "auto", "always", "never", "critical":
				calendarName = str
			default:
				panic(r.newError(r.getRangeError(), "Invalid calendarName"))
			}
		}
	}
	out := fmt.Sprintf("%02d-%02d", m.month, m.day)
	if calendarName == "always" || calendarName == "critical" {
		out = fmt.Sprintf("%04d-%02d-%02d", m.refYear, m.month, m.day)
	}
	switch calendarName {
	case "always":
		out += "[u-ca=" + m.calendar + "]"
	case "critical":
		out += "[!u-ca=" + m.calendar + "]"
	}
	return r.ToValue(out)
}

func (r *Runtime) temporalPlainMonthDayToJSON(call FunctionCall) Value {
	return r.temporalPlainMonthDayToString(FunctionCall{This: call.This, Arguments: []Value{_undefined}})
}

func (r *Runtime) temporalPlainMonthDayToLocaleString(call FunctionCall) Value {
	return r.temporalPlainMonthDayToJSON(call)
}

func (r *Runtime) temporalPlainMonthDayValueOf(call FunctionCall) Value {
	panic(r.NewTypeError("Temporal.PlainMonthDay objects cannot be converted to primitives"))
}

func (r *Runtime) temporalPlainMonthDayToPlainDate(call FunctionCall) Value {
	m := r.toTemporalPlainMonthDay(call.This)
	partial := r.toObject(call.Argument(0))
	yv := partial.self.getStr("year", nil)
	if yv == nil || IsUndefined(yv) {
		panic(r.NewTypeError("year is required"))
	}
	year := r.temporalTimeFieldValue(yv)
	day := m.day
	if day > isoDaysInMonth(year, m.month) {
		day = isoDaysInMonth(year, m.month)
	}
	date := isoDate{year: year, month: m.month, day: day}
	if !date.isValid() {
		panic(r.newError(r.getRangeError(), "Resulting date is out of range"))
	}
	return r.newTemporalPlainDateObject(date, m.calendar, r.getTemporalPlainDatePrototype())
}

func (r *Runtime) temporalPlainMonthDayGetISOFields(call FunctionCall) Value {
	m := r.toTemporalPlainMonthDay(call.This)
	o := r.NewObject()
	o.Set("isoYear", r.ToValue(m.refYear))
	o.Set("isoMonth", r.ToValue(m.month))
	o.Set("isoDay", r.ToValue(m.day))
	o.Set("calendar", r.ToValue(m.calendar))
	return o
}
