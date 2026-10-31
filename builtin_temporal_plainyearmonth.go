package goja

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/dop251/goja/unistring"
)

func isoYearMonthValid(year, month int64) bool {
	if month < 1 || month > 12 {
		return false
	}
	first := isoDate{year: year, month: month, day: 1}
	last := isoDate{year: year, month: month, day: isoDaysInMonth(year, month)}
	return last.days() >= temporalMinISODate && first.days() <= temporalMaxISODate
}

type temporalPlainYearMonthObject struct {
	baseObject
	year, month, refDay int64
	calendar            string
}

func (r *Runtime) getTemporalPlainYearMonth() *Object {
	ret := r.global.TemporalPlainYearMonth
	if ret == nil {
		ret = &Object{runtime: r}
		r.global.TemporalPlainYearMonth = ret
		r.newNativeFuncAndConstruct(ret, func(call FunctionCall) Value {
			panic(r.NewTypeError("Constructor Temporal.PlainYearMonth requires 'new'"))
		}, func(args []Value, newTarget *Object) *Object {
			arg := func(i int) Value {
				if i < len(args) {
					return args[i]
				}
				return _undefined
			}
			y := r.temporalToIntegerWithTruncation(args, 0)
			m := r.temporalPositiveIntegerValue(arg(1))
			calendar := "iso8601"
			if !IsUndefined(arg(2)) && arg(2) != nil {
				calendar = r.temporalCalendarID(arg(2))
			}
			refDay := int64(1)
			if !IsUndefined(arg(3)) && arg(3) != nil {
				refDay = r.temporalPositiveIntegerValue(arg(3))
			}
			if m < 1 || m > 12 || refDay < 1 || refDay > isoDaysInMonth(y, m) {
				panic(r.newError(r.getRangeError(), "Invalid year-month"))
			}
			if !isoYearMonthValid(y, m) {
				panic(r.newError(r.getRangeError(), "Year-month is out of range"))
			}
			proto := r.getTemporalPlainYearMonthPrototype()
			if newTarget != nil {
				proto = r.getPrototypeFromCtor(newTarget, ret, proto)
			}
			return r.newTemporalPlainYearMonthObject(y, m, refDay, calendar, proto)
		}, r.getTemporalPlainYearMonthPrototype(), "PlainYearMonth", intToValue(2))
		ctor := ret.self.(*nativeFuncObject)
		ctor._putProp("from", r.newNativeFunc(r.temporalPlainYearMonthFrom, "from", 1), true, false, true)
		ctor._putProp("compare", r.newNativeFunc(r.temporalPlainYearMonthCompare, "compare", 2), true, false, true)
	}
	return ret
}

func (r *Runtime) getTemporalPlainYearMonthPrototype() *Object {
	ret := r.global.TemporalPlainYearMonthPrototype
	if ret == nil {
		o := r.newBaseObject(r.global.ObjectPrototype, classObject)
		ret = o.val
		r.global.TemporalPlainYearMonthPrototype = ret
		o._putProp("constructor", r.getTemporalPlainYearMonth(), true, false, true)
		addGetter := func(name string, f func(*temporalPlainYearMonthObject) Value) {
			o.setOwnStr(unistring.NewFromString(name), &valueProperty{
				getterFunc: r.newNativeFunc(func(call FunctionCall) Value {
					return f(r.toTemporalPlainYearMonth(call.This))
				}, unistring.NewFromString("get "+name), 0),
				accessor: true, enumerable: false, configurable: true,
			}, true)
		}
		addGetter("year", func(y *temporalPlainYearMonthObject) Value { return intToValue(y.year) })
		addGetter("month", func(y *temporalPlainYearMonthObject) Value { return intToValue(y.month) })
		addGetter("monthCode", func(y *temporalPlainYearMonthObject) Value { return asciiString(fmt.Sprintf("M%02d", y.month)) })
		addGetter("calendarId", func(y *temporalPlainYearMonthObject) Value { return r.ToValue(y.calendar) })
		addGetter("era", func(y *temporalPlainYearMonthObject) Value { return _undefined })
		addGetter("eraYear", func(y *temporalPlainYearMonthObject) Value { return _undefined })
		addGetter("daysInMonth", func(y *temporalPlainYearMonthObject) Value { return intToValue(isoDaysInMonth(y.year, y.month)) })
		addGetter("daysInYear", func(y *temporalPlainYearMonthObject) Value {
			if isoLeapYear(y.year) {
				return intToValue(366)
			}
			return intToValue(365)
		})
		addGetter("monthsInYear", func(y *temporalPlainYearMonthObject) Value { return intToValue(12) })
		addGetter("inLeapYear", func(y *temporalPlainYearMonthObject) Value { return r.ToValue(isoLeapYear(y.year)) })
		o._putProp("with", r.newNativeFunc(r.temporalPlainYearMonthWith, "with", 1), true, false, true)
		o._putProp("add", r.newNativeFunc(r.temporalPlainYearMonthAdd, "add", 1), true, false, true)
		o._putProp("subtract", r.newNativeFunc(r.temporalPlainYearMonthSubtract, "subtract", 1), true, false, true)
		o._putProp("until", r.newNativeFunc(r.temporalPlainYearMonthUntil, "until", 1), true, false, true)
		o._putProp("since", r.newNativeFunc(r.temporalPlainYearMonthSince, "since", 1), true, false, true)
		o._putProp("equals", r.newNativeFunc(r.temporalPlainYearMonthEquals, "equals", 1), true, false, true)
		o._putProp("toString", r.newNativeFunc(r.temporalPlainYearMonthToString, "toString", 0), true, false, true)
		o._putProp("toJSON", r.newNativeFunc(r.temporalPlainYearMonthToJSON, "toJSON", 0), true, false, true)
		o._putProp("toLocaleString", r.newNativeFunc(r.temporalPlainYearMonthToLocaleString, "toLocaleString", 0), true, false, true)
		o._putProp("valueOf", r.newNativeFunc(r.temporalPlainYearMonthValueOf, "valueOf", 0), true, false, true)
		o._putProp("toPlainDate", r.newNativeFunc(r.temporalPlainYearMonthToPlainDate, "toPlainDate", 1), true, false, true)
		o._putProp("getISOFields", r.newNativeFunc(r.temporalPlainYearMonthGetISOFields, "getISOFields", 0), true, false, true)
		o._putSym(SymToStringTag, &valueProperty{
			value: asciiString("Temporal.PlainYearMonth"), writable: false, enumerable: false, configurable: true,
		})
	}
	return ret
}

func (r *Runtime) newTemporalPlainYearMonthObject(y, m, refDay int64, calendar string, proto *Object) *Object {
	v := &Object{runtime: r}
	o := &temporalPlainYearMonthObject{year: y, month: m, refDay: refDay, calendar: calendar}
	o.baseObject = baseObject{class: classObject, val: v, extensible: true, prototype: proto}
	v.self = o
	o.init()
	return v
}

func (r *Runtime) toTemporalPlainYearMonth(v Value) *temporalPlainYearMonthObject {
	o := r.toObject(v)
	if y, ok := o.self.(*temporalPlainYearMonthObject); ok {
		return y
	}
	panic(r.NewTypeError("Value is not a Temporal.PlainYearMonth"))
}

func (r *Runtime) temporalPlainYearMonthFrom(call FunctionCall) Value {
	y := r.temporalPlainYearMonthFromValue(call.Argument(0), call.Argument(1))
	return r.newTemporalPlainYearMonthObject(y.year, y.month, y.refDay, y.calendar, r.getTemporalPlainYearMonthPrototype())
}

func (r *Runtime) temporalPlainYearMonthFromValue(v Value, options Value) *temporalPlainYearMonthObject {
	if o, ok := v.(*Object); ok {
		if y, ok := o.self.(*temporalPlainYearMonthObject); ok {
			_ = r.temporalOverflowOption(options)
			return y
		}
	}
	if s, ok := v.(String); ok {
		y, m, refDay, cal, ok := parseTemporalYearMonthString(s.String())
		if !ok {
			panic(r.newError(r.getRangeError(), "Invalid ISO year-month string: "+s.String()))
		}
		_ = r.temporalOverflowOption(options)
		return r.plainYearMonthFieldsObject(y, m, refDay, cal)
	}
	obj, ok := v.(*Object)
	if !ok {
		panic(r.NewTypeError("Cannot convert %s to Temporal.PlainYearMonth", v))
	}
	calendar := "iso8601"
	if cv := obj.self.getStr("calendar", nil); cv != nil && !IsUndefined(cv) {
		calendar = r.temporalCalendarFromValue(cv)
	}
	var year, month int64
	var monthCode string
	haveYear, haveMonth, hasMonthCode := false, false, false
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
		year = r.temporalTimeFieldValue(yv)
		haveYear = true
	}
	overflow := r.temporalOverflowOption(options)
	if !haveYear || (!haveMonth && !hasMonthCode) {
		panic(r.NewTypeError("year and month are required"))
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
	}
	if month < 1 || month > 12 || !(isoYearMonthValid(year, month)) {
		panic(r.newError(r.getRangeError(), "Invalid year-month"))
	}
	return r.plainYearMonthFieldsObject(year, month, 1, calendar)
}

func (r *Runtime) plainYearMonthFieldsObject(y, m, refDay int64, calendar string) *temporalPlainYearMonthObject {
	return r.newTemporalPlainYearMonthObject(y, m, refDay, calendar, r.getTemporalPlainYearMonthPrototype()).self.(*temporalPlainYearMonthObject)
}

// parseTemporalYearMonthString parses an ISO year-month string (YYYY-MM,
// YYYY-MM-DD, YYYYMM, YYYYMMDD or a date-time) with optional annotations.
func parseTemporalYearMonthString(s string) (year, month, refDay int64, calendar string, ok bool) {
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

	sign := int64(1)
	hasSign := false
	body := base
	if len(body) > 0 && (body[0] == '+' || body[0] == '-') {
		hasSign = true
		if body[0] == '-' {
			sign = -1
		}
		body = body[1:]
	}
	allZero := func(v string) bool {
		return strings.Trim(v, "0") == ""
	}
	readYearMonth := func(yd, rest string) bool {
		if !isDigits(yd) || len(rest) < 2 || !isDigits(rest[:2]) {
			return false
		}
		y := int64(0)
		for i := 0; i < len(yd); i++ {
			y = y*10 + int64(yd[i]-'0')
		}
		m := int64(rest[0]-'0')*10 + int64(rest[1]-'0')
		if sign < 0 && allZero(yd) {
			return false
		}
		y *= sign
		if m < 1 || m > 12 || !isoYearMonthValid(y, m) {
			return false
		}
		year, month, refDay = y, m, 1
		return true
	}
	// Basic forms: YYYYMM, YYYYMMDD, YYYYYYMM, YYYYYYMMDD.
	if isDigits(body) {
		switch len(body) {
		case 6:
			if readYearMonth(body[:4], body[4:]) {
				return year, month, 1, calendar, true
			}
		case 8:
			if hasSign {
				if readYearMonth(body[:6], body[6:]) {
					return year, month, 1, calendar, true
				}
				break
			}
			if sign < 0 && allZero(body[:4]) {
				return 0, 0, 0, "", false
			}
			if readYearMonth(body[:4], body[4:6]) {
				return year, month, 1, calendar, true
			}
		case 10:
			if readYearMonth(body[:6], body[6:]) {
				return year, month, 1, calendar, true
			}
		}
		return 0, 0, 0, "", false
	}
	// Extended forms: [sign]YYYY-MM[-DD] and [sign]YYYYYY-MM[-DD].
	if i := strings.IndexByte(body, '-'); i > 0 {
		yd := body[:i]
		rest := body[i+1:]
		if len(yd) != 4 && len(yd) != 6 {
			return 0, 0, 0, "", false
		}
		if len(rest) > 2 {
			if rest[2] != '-' || !isDigits(rest[3:]) || len(rest[3:]) != 2 {
				return 0, 0, 0, "", false
			}
			rest = rest[:2]
		}
		if readYearMonth(yd, rest) {
			return year, month, 1, calendar, true
		}
	}
	return 0, 0, 0, "", false
}

func (r *Runtime) temporalPlainYearMonthWith(call FunctionCall) Value {
	y := r.toTemporalPlainYearMonth(call.This)
	partial := r.toObject(call.Argument(0))
	switch partial.self.(type) {
	case *temporalPlainDateObject, *temporalPlainTimeObject, *temporalPlainDateTimeObject,
		*temporalInstantObject, *temporalZonedDateTimeObject, *temporalDurationObject,
		*temporalPlainYearMonthObject, *temporalPlainMonthDayObject:
		panic(r.NewTypeError("with() argument must be a plain object"))
	}
	var year, month int64 = y.year, y.month
	var monthCode string
	hasMonthCode := false
	provided := false
	if cv := partial.self.getStr("calendar", nil); cv != nil && !IsUndefined(cv) {
		panic(r.NewTypeError("calendar is not allowed in with()"))
	}
	if tv := partial.self.getStr("timeZone", nil); tv != nil && !IsUndefined(tv) {
		panic(r.NewTypeError("timeZone is not allowed in with()"))
	}
	hasMonth := false
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
		year = r.temporalTimeFieldValue(v)
		provided = true
	}
	overflow := r.temporalOverflowOption(call.Argument(1))
	if !provided {
		panic(r.NewTypeError("PlainYearMonth.with requires at least one property"))
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
		if month < 1 {
			panic(r.newError(r.getRangeError(), "Invalid month"))
		}
		if month > 12 {
			month = 12
		}
	}
	if month < 1 || month > 12 || !(isoYearMonthValid(year, month)) {
		panic(r.newError(r.getRangeError(), "Invalid year-month"))
	}
	refDay := y.refDay
	if max := isoDaysInMonth(year, month); refDay > max {
		refDay = max
	}
	return r.newTemporalPlainYearMonthObject(year, month, refDay, y.calendar, r.getTemporalPlainYearMonthPrototype())
}

func (r *Runtime) temporalPlainYearMonthAddSub(call FunctionCall, subtract bool) Value {
	y := r.toTemporalPlainYearMonth(call.This)
	d := r.temporalDurationFromValue(call.Argument(0))
	_ = r.temporalOverflowOption(call.Argument(1))
	if !(isoDate{year: y.year, month: y.month, day: y.refDay}.isValid()) {
		panic(r.newError(r.getRangeError(), "Year-month is out of range"))
	}
	fields := d.fields()
	if fields[2] != 0 || fields[3] != 0 || fields[4] != 0 || fields[5] != 0 ||
		fields[6] != 0 || fields[7] != 0 || fields[8] != 0 || fields[9] != 0 {
		panic(r.newError(r.getRangeError(), "PlainYearMonth can only add years and months"))
	}
	sign := int64(1)
	if subtract {
		sign = -1
	}
	total := (int64(fields[0])*12 + int64(fields[1])) * sign
	base := isoDate{year: y.year, month: y.month, day: y.refDay}
	date := addISOMonths(base, total)
	if !date.isValid() {
		panic(r.newError(r.getRangeError(), "Resulting year-month is out of range"))
	}
	return r.newTemporalPlainYearMonthObject(date.year, date.month, date.day, y.calendar, r.getTemporalPlainYearMonthPrototype())
}

func (r *Runtime) temporalPlainYearMonthAdd(call FunctionCall) Value {
	return r.temporalPlainYearMonthAddSub(call, false)
}

func (r *Runtime) temporalPlainYearMonthSubtract(call FunctionCall) Value {
	return r.temporalPlainYearMonthAddSub(call, true)
}

func (r *Runtime) temporalPlainYearMonthUntil(call FunctionCall) Value {
	a := r.toTemporalPlainYearMonth(call.This)
	b := r.temporalPlainYearMonthFromValue(call.Argument(0), _undefined)
	return r.temporalPlainYearMonthDifference(a, b, call.Argument(1), false)
}

func (r *Runtime) temporalPlainYearMonthSince(call FunctionCall) Value {
	a := r.toTemporalPlainYearMonth(call.This)
	b := r.temporalPlainYearMonthFromValue(call.Argument(0), _undefined)
	return r.temporalPlainYearMonthDifference(a, b, call.Argument(1), true)
}

func (r *Runtime) temporalPlainYearMonthDifference(a, b *temporalPlainYearMonthObject, options Value, since bool) Value {
	s := r.temporalGetDifferenceSettings(options, "year", "month", 0, 1)
	if a.year == b.year && a.month == b.month {
		return r.newTemporalDurationObject([10]float64{}, r.getTemporalDurationPrototype())
	}
	// Converting a year-month to a date uses its reference day.
	if !(isoDate{year: a.year, month: a.month, day: a.refDay}.isValid()) ||
		!(isoDate{year: b.year, month: b.month, day: b.refDay}.isValid()) {
		panic(r.newError(r.getRangeError(), "Year-month is out of range"))
	}
	if since {
		s.mode = negateRoundingMode(s.mode)
	}
	totalMonths := (b.year-a.year)*12 + (b.month - a.month)
	negative := totalMonths < 0
	mag := totalMonths
	if negative {
		mag = -mag
	}
	// Round to the requested unit.
	years, months := int64(0), int64(0)
	switch {
	case s.smallest == "year":
		scale := 12 * s.increment
		q := mag / scale
		rem := mag % scale
		if rem != 0 && roundMagnitude(s.mode, negative, big.NewInt(q), big.NewInt(rem), big.NewInt(scale)) {
			q++
		}
		mag = q * scale
		years = mag / 12
		if s.increment > 1 {
			cand := (q + 1) * scale
			cy := a.year + floorDiv(cand, 12)
			if !isoYearMonthValid(cy, 1) {
				panic(r.newError(r.getRangeError(), "Resulting year-month is out of range"))
			}
		}
	case s.largest == "month":
		q := mag / s.increment
		rem := mag % s.increment
		if rem != 0 && roundMagnitude(s.mode, negative, big.NewInt(q), big.NewInt(rem), big.NewInt(s.increment)) {
			q++
		}
		months = q * s.increment
	default:
		years = mag / 12
		rem := mag % 12
		q := rem / s.increment
		r2 := rem % s.increment
		rounded := q
		if r2 != 0 && roundMagnitude(s.mode, negative, big.NewInt(q), big.NewInt(r2), big.NewInt(s.increment)) {
			rounded = q + 1
		}
		if inc := s.increment; inc > 1 {
			for _, cand := range []int64{rounded, rounded + 1} {
				candMonths := years*12 + cand*inc
				cy := a.year + floorDiv(candMonths, 12)
				cm := candMonths - floorDiv(candMonths, 12)*12 + 1
				if !isoYearMonthValid(cy, cm) {
					panic(r.newError(r.getRangeError(), "Resulting year-month is out of range"))
				}
			}
		}
		months = rounded * s.increment
		if months >= 12 {
			years++
			months -= 12
		}
	}
	resMonths := years*12 + months
	resYear := a.year + floorDiv(resMonths, 12)
	resMonth := resMonths - floorDiv(resMonths, 12)*12 + 1
	if !isoYearMonthValid(resYear, resMonth) {
		panic(r.newError(r.getRangeError(), "Resulting year-month is out of range"))
	}
	fields := [10]float64{float64(years), float64(months)}
	if negative {
		fields[0], fields[1] = -fields[0], -fields[1]
	}
	if since {
		fields[0], fields[1] = -fields[0], -fields[1]
	}
	r.validateTemporalDuration(fields)
	return r.newTemporalDurationObject(fields, r.getTemporalDurationPrototype())
}

func (r *Runtime) temporalPlainYearMonthEquals(call FunctionCall) Value {
	a := r.toTemporalPlainYearMonth(call.This)
	b := r.temporalPlainYearMonthFromValue(call.Argument(0), _undefined)
	return r.ToValue(a.year == b.year && a.month == b.month && a.refDay == b.refDay && a.calendar == b.calendar)
}

func (r *Runtime) temporalPlainYearMonthCompare(call FunctionCall) Value {
	a := r.temporalPlainYearMonthFromValue(call.Argument(0), _undefined)
	b := r.temporalPlainYearMonthFromValue(call.Argument(1), _undefined)
	if a.year != b.year {
		if a.year < b.year {
			return intToValue(-1)
		}
		return intToValue(1)
	}
	if a.month != b.month {
		if a.month < b.month {
			return intToValue(-1)
		}
		return intToValue(1)
	}
	if a.refDay != b.refDay {
		if a.refDay < b.refDay {
			return intToValue(-1)
		}
		return intToValue(1)
	}
	return intToValue(0)
}

func (r *Runtime) temporalPlainYearMonthToString(call FunctionCall) Value {
	y := r.toTemporalPlainYearMonth(call.This)
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
	out := formatISOYear(y.year) + fmt.Sprintf("-%02d", y.month)
	if y.refDay != 1 || calendarName == "always" || calendarName == "critical" {
		out += fmt.Sprintf("-%02d", y.refDay)
	}
	switch calendarName {
	case "always":
		out += "[u-ca=" + y.calendar + "]"
	case "critical":
		out += "[!u-ca=" + y.calendar + "]"
	}
	return r.ToValue(out)
}

func (r *Runtime) temporalPlainYearMonthToJSON(call FunctionCall) Value {
	return r.temporalPlainYearMonthToString(FunctionCall{This: call.This, Arguments: []Value{_undefined}})
}

func (r *Runtime) temporalPlainYearMonthToLocaleString(call FunctionCall) Value {
	return r.temporalPlainYearMonthToJSON(call)
}

func (r *Runtime) temporalPlainYearMonthValueOf(call FunctionCall) Value {
	panic(r.NewTypeError("Temporal.PlainYearMonth objects cannot be converted to primitives"))
}

func (r *Runtime) temporalPlainYearMonthToPlainDate(call FunctionCall) Value {
	y := r.toTemporalPlainYearMonth(call.This)
	partial := r.toObject(call.Argument(0))
	dv := partial.self.getStr("day", nil)
	if dv == nil || IsUndefined(dv) {
		panic(r.NewTypeError("day is required"))
	}
	day := r.temporalPositiveIntegerValue(dv)
	if day > isoDaysInMonth(y.year, y.month) {
		day = isoDaysInMonth(y.year, y.month)
	}
	date := isoDate{year: y.year, month: y.month, day: day}
	if !date.isValid() {
		panic(r.newError(r.getRangeError(), "Resulting date is out of range"))
	}
	return r.newTemporalPlainDateObject(date, y.calendar, r.getTemporalPlainDatePrototype())
}

func (r *Runtime) temporalPlainYearMonthGetISOFields(call FunctionCall) Value {
	y := r.toTemporalPlainYearMonth(call.This)
	o := r.NewObject()
	o.Set("isoYear", r.ToValue(y.year))
	o.Set("isoMonth", r.ToValue(y.month))
	o.Set("isoDay", r.ToValue(y.refDay))
	o.Set("calendar", r.ToValue(y.calendar))
	return o
}
