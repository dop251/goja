package goja

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/dop251/goja/unistring"
)

// isoDate is a proleptic Gregorian calendar date in the ISO 8601 calendar.
type isoDate struct {
	year  int64
	month int64
	day   int64
}

func isoLeapYear(y int64) bool {
	return y%4 == 0 && (y%100 != 0 || y%400 == 0)
}

func isoDaysInMonth(y, m int64) int64 {
	switch m {
	case 1, 3, 5, 7, 8, 10, 12:
		return 31
	case 4, 6, 9, 11:
		return 30
	case 2:
		if isoLeapYear(y) {
			return 29
		}
		return 28
	}
	return 0
}

func floorDiv(a, b int64) int64 {
	q := a / b
	r := a % b
	if r != 0 && ((r < 0) != (b < 0)) {
		q--
	}
	return q
}

// isoDaysFromCivil returns the number of days since 1970-01-01.
func isoDaysFromCivil(y, m, d int64) int64 {
	if m <= 2 {
		y--
	}
	era := floorDiv(y, 400)
	yoe := y - era*400
	mp := (m + 9) % 12
	doy := (153*mp+2)/5 + d - 1
	doe := yoe*365 + yoe/4 - yoe/100 + doy
	return era*146097 + doe - 719468
}

func isoCivilFromDays(z int64) (y, m, d int64) {
	z += 719468
	era := floorDiv(z, 146097)
	doe := z - era*146097
	yoe := (doe - doe/1460 + doe/36524 - doe/146096) / 365
	y = yoe + era*400
	doy := doe - (365*yoe + yoe/4 - yoe/100)
	mp := (5*doy + 2) / 153
	d = doy - (153*mp+2)/5 + 1
	m = mp + 3
	if m > 12 {
		m -= 12
		y++
	}
	return
}

var (
	temporalMinISODate = isoDaysFromCivil(-271821, 4, 19)
	temporalMaxISODate = isoDaysFromCivil(275760, 9, 13)
)

func (d isoDate) days() int64 { return isoDaysFromCivil(d.year, d.month, d.day) }

func (d isoDate) isValid() bool {
	if d.month < 1 || d.month > 12 || d.day < 1 || d.day > isoDaysInMonth(d.year, d.month) {
		return false
	}
	days := d.days()
	return days >= temporalMinISODate && days <= temporalMaxISODate
}

// isValidDateTime reports whether the date-time is within the ISO limits. The
// lower bound is exclusive at midnight, so -271821-04-19T00:00:00 is invalid.
func (d isoDate) isValidDateTime(t isoTime) bool {
	if !d.isValid() || !t.isValid() {
		return false
	}
	return !(d.days() == temporalMinISODate && t.nanos() == 0)
}

func (d isoDate) weekInfo() (week, year int64) {
	dayOfYear := d.days() - isoDaysFromCivil(d.year, 1, 1) + 1
	dayOfWeek := ((d.days()+3)%7+7)%7 + 1 // Monday=1
	week = (dayOfYear - dayOfWeek + 10) / 7
	year = d.year
	if week < 1 {
		year--
		week = isoWeeksInYear(year)
	} else if week > isoWeeksInYear(d.year) {
		year++
		week = 1
	}
	return
}

func isoWeeksInYear(y int64) int64 {
	jan1 := isoDaysFromCivil(y, 1, 1)
	// 53 weeks if Jan 1 is Thursday, or Wednesday in a leap year.
	dow := ((jan1+3)%7+7)%7 + 1
	if dow == 4 || (dow == 3 && isoLeapYear(y)) {
		return 53
	}
	return 52
}

type temporalPlainDateObject struct {
	baseObject
	date     isoDate
	calendar string
}

func (r *Runtime) getTemporalPlainDate() *Object {
	ret := r.global.TemporalPlainDate
	if ret == nil {
		ret = &Object{runtime: r}
		r.global.TemporalPlainDate = ret
		r.newNativeFuncAndConstruct(ret, func(call FunctionCall) Value {
			panic(r.NewTypeError("Constructor Temporal.PlainDate requires 'new'"))
		}, func(args []Value, newTarget *Object) *Object {
			y := r.temporalToIntegerWithTruncation(args, 0)
			mo := r.temporalToIntegerWithTruncation(args, 1)
			d := r.temporalToIntegerWithTruncation(args, 2)
			calendar := "iso8601"
			if len(args) > 3 && !IsUndefined(args[3]) {
				calendar = r.temporalCalendarID(args[3])
			}
			date := isoDate{year: y, month: mo, day: d}
			if !date.isValid() {
				panic(r.newError(r.getRangeError(), "Invalid ISO date"))
			}
			proto := r.getTemporalPlainDatePrototype()
			if newTarget != nil {
				proto = r.getPrototypeFromCtor(newTarget, ret, proto)
			}
			return r.newTemporalPlainDateObject(date, calendar, proto)
		}, r.getTemporalPlainDatePrototype(), "PlainDate", intToValue(3))
		ctor := ret.self.(*nativeFuncObject)
		ctor._putProp("from", r.newNativeFunc(r.temporalPlainDateFrom, "from", 1), true, false, true)
		ctor._putProp("compare", r.newNativeFunc(r.temporalPlainDateCompare, "compare", 2), true, false, true)
	}
	return ret
}

func (r *Runtime) temporalToIntegerWithTruncation(args []Value, i int) int64 {
	var v Value
	if i < len(args) {
		v = args[i]
	}
	if v == nil || IsUndefined(v) {
		panic(r.newError(r.getRangeError(), "Missing date component"))
	}
	f := v.ToNumber().ToFloat()
	if math.IsNaN(f) || math.IsInf(f, 0) {
		panic(r.newError(r.getRangeError(), "Invalid date component"))
	}
	return int64(math.Trunc(f))
}

// temporalCalendarID validates a calendar argument used by constructors. Only
// calendar identifiers (not ISO date strings) are accepted here.
func (r *Runtime) temporalCalendarID(v Value) string {
	if s, ok := v.(String); ok {
		if asciiEqualFold(s.String(), "iso8601") {
			return "iso8601"
		}
		panic(r.newError(r.getRangeError(), "Unsupported calendar: "+s.String()))
	}
	panic(r.NewTypeError("A calendar identifier must be a string"))
}

// temporalCalendarFromValue is the property-bag / withCalendar variant, which
// also accepts ISO date strings and Temporal objects.
func (r *Runtime) temporalCalendarFromValue(v Value) string {
	switch x := v.(type) {
	case String:
		s := x.String()
		if asciiEqualFold(s, "iso8601") {
			return "iso8601"
		}
		if cal, ok := calendarFromISOString(s); ok {
			return cal
		}
		panic(r.newError(r.getRangeError(), "Unsupported calendar: "+s))
	case *Object:
		switch o := x.self.(type) {
		case *temporalPlainDateObject:
			return o.calendar
		case *temporalPlainTimeObject:
			return "iso8601"
		case *temporalPlainDateTimeObject:
			return o.calendar
		case *temporalInstantObject:
			return "iso8601"
		case *temporalZonedDateTimeObject:
			return o.calendar
		case *temporalPlainYearMonthObject:
			return o.calendar
		case *temporalPlainMonthDayObject:
			return o.calendar
		}
		panic(r.NewTypeError("A calendar must be a string or Temporal object"))
	default:
		panic(r.NewTypeError("A calendar must be a string or Temporal object"))
	}
}

func asciiEqualFold(s, t string) bool {
	if len(s) != len(t) {
		return false
	}
	for i := 0; i < len(s); i++ {
		a, b := s[i], t[i]
		if a >= 'A' && a <= 'Z' {
			a += 'a' - 'A'
		}
		if b >= 'A' && b <= 'Z' {
			b += 'a' - 'A'
		}
		if a != b {
			return false
		}
	}
	return true
}

func calendarFromISOString(s string) (string, bool) {
	// Look for a calendar annotation.
	if i := strings.Index(s, "[u-ca="); i >= 0 {
		j := strings.IndexByte(s[i:], ']')
		if j < 0 {
			return "", false
		}
		val := s[i+len("[u-ca=") : i+j]
		if !asciiEqualFold(val, "iso8601") {
			return "", false
		}
		return "iso8601", true
	}
	if strings.Contains(s, "[") {
		base, _, ok := splitTemporalAnnotations(s)
		if !ok {
			return "", false
		}
		s = base
	}
	if _, _, _, ok := parseTemporalPlainDateTimeString(s); ok {
		return "iso8601", true
	}
	if _, _, _, _, _, _, ok := parseTemporalPlainTimeString(s); ok {
		return "iso8601", true
	}
	if _, _, ok := parseTemporalPlainDateString(s); ok {
		return "iso8601", true
	}
	if looksLikeISODateOrTime(s) {
		return "iso8601", true
	}
	return "", false
}

func looksLikeISODateOrTime(s string) bool {
	if len(s) >= 5 && isDigits(s[:2]) && s[2] == ':' && isDigits(s[3:5]) {
		return true
	}
	if (len(s) == 4 || len(s) == 6) && isDigits(s) {
		return true
	}
	if len(s) > 5 && isDigits(s[:6]) && (s[6] == '.' || s[6] == ',') && isDigits(s[7:]) {
		return true
	}
	if len(s) == 5 && isDigits(s[:2]) && s[2] == '-' && isDigits(s[3:5]) {
		return true
	}
	i := 0
	neg := false
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		neg = s[i] == '-'
		i++
	}
	d := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
		d++
	}
	if d != 4 && d != 6 {
		return false
	}
	if neg && strings.Trim(s[i-d:i], "0") == "" {
		return false
	}
	if i >= len(s) {
		return false
	}
	if s[i] == '-' {
		i++
		if i+2 > len(s) || !isDigits(s[i:i+2]) {
			return false
		}
		i += 2
		if i == len(s) {
			return true
		}
		if s[i] == '-' {
			i++
			if i+2 > len(s) || !isDigits(s[i:i+2]) {
				return false
			}
			i += 2
			if i == len(s) {
				return true
			}
		}
	}
	if i < len(s) && (s[i] == 'T' || s[i] == 't' || s[i] == ' ') {
		return true
	}
	return false
}

func (r *Runtime) getTemporalPlainDatePrototype() *Object {
	ret := r.global.TemporalPlainDatePrototype
	if ret == nil {
		o := r.newBaseObject(r.global.ObjectPrototype, classObject)
		ret = o.val
		r.global.TemporalPlainDatePrototype = ret
		o._putProp("constructor", r.getTemporalPlainDate(), true, false, true)
		addGetter := func(name string, f func(*temporalPlainDateObject) Value) {
			o.setOwnStr(unistring.NewFromString(name), &valueProperty{
				getterFunc: r.newNativeFunc(func(call FunctionCall) Value {
					return f(r.toTemporalPlainDate(call.This))
				}, unistring.NewFromString("get "+name), 0),
				accessor: true, enumerable: false, configurable: true,
			}, true)
		}
		addGetter("year", func(d *temporalPlainDateObject) Value { return intToValue(d.date.year) })
		addGetter("month", func(d *temporalPlainDateObject) Value { return intToValue(d.date.month) })
		addGetter("day", func(d *temporalPlainDateObject) Value { return intToValue(d.date.day) })
		addGetter("calendarId", func(d *temporalPlainDateObject) Value { return r.ToValue(d.calendar) })
		addGetter("era", func(d *temporalPlainDateObject) Value { return _undefined })
		addGetter("eraYear", func(d *temporalPlainDateObject) Value { return _undefined })
		addGetter("monthCode", func(d *temporalPlainDateObject) Value {
			return asciiString(fmt.Sprintf("M%02d", d.date.month))
		})
		addGetter("dayOfWeek", func(d *temporalPlainDateObject) Value {
			return intToValue(((d.date.days()+3)%7+7)%7 + 1)
		})
		addGetter("dayOfYear", func(d *temporalPlainDateObject) Value {
			return intToValue(d.date.days() - isoDaysFromCivil(d.date.year, 1, 1) + 1)
		})
		addGetter("weekOfYear", func(d *temporalPlainDateObject) Value {
			w, _ := d.date.weekInfo()
			return intToValue(w)
		})
		addGetter("yearOfWeek", func(d *temporalPlainDateObject) Value {
			_, y := d.date.weekInfo()
			return intToValue(y)
		})
		addGetter("daysInWeek", func(d *temporalPlainDateObject) Value { return intToValue(7) })
		addGetter("daysInMonth", func(d *temporalPlainDateObject) Value {
			return intToValue(isoDaysInMonth(d.date.year, d.date.month))
		})
		addGetter("daysInYear", func(d *temporalPlainDateObject) Value {
			if isoLeapYear(d.date.year) {
				return intToValue(366)
			}
			return intToValue(365)
		})
		addGetter("monthsInYear", func(d *temporalPlainDateObject) Value { return intToValue(12) })
		addGetter("inLeapYear", func(d *temporalPlainDateObject) Value {
			return r.ToValue(isoLeapYear(d.date.year))
		})
		o._putProp("with", r.newNativeFunc(r.temporalPlainDateWith, "with", 1), true, false, true)
		o._putProp("withCalendar", r.newNativeFunc(r.temporalPlainDateWithCalendar, "withCalendar", 1), true, false, true)
		o._putProp("add", r.newNativeFunc(r.temporalPlainDateAdd, "add", 1), true, false, true)
		o._putProp("subtract", r.newNativeFunc(r.temporalPlainDateSubtract, "subtract", 1), true, false, true)
		o._putProp("until", r.newNativeFunc(r.temporalPlainDateUntil, "until", 1), true, false, true)
		o._putProp("since", r.newNativeFunc(r.temporalPlainDateSince, "since", 1), true, false, true)
		o._putProp("equals", r.newNativeFunc(r.temporalPlainDateEquals, "equals", 1), true, false, true)
		o._putProp("toString", r.newNativeFunc(r.temporalPlainDateToString, "toString", 0), true, false, true)
		o._putProp("toJSON", r.newNativeFunc(r.temporalPlainDateToJSON, "toJSON", 0), true, false, true)
		o._putProp("toLocaleString", r.newNativeFunc(r.temporalPlainDateToLocaleString, "toLocaleString", 0), true, false, true)
		o._putProp("valueOf", r.newNativeFunc(r.temporalPlainDateValueOf, "valueOf", 0), true, false, true)
		o._putProp("getISOFields", r.newNativeFunc(r.temporalPlainDateGetISOFields, "getISOFields", 0), true, false, true)
		o._putProp("toPlainDateTime", r.newNativeFunc(r.temporalPlainDateToPlainDateTime, "toPlainDateTime", 0), true, false, true)
		o._putProp("toZonedDateTime", r.newNativeFunc(r.temporalPlainDateToZonedDateTime, "toZonedDateTime", 1), true, false, true)
		o._putProp("toPlainYearMonth", r.newNativeFunc(r.temporalPlainDateToPlainYearMonth, "toPlainYearMonth", 0), true, false, true)
		o._putProp("toPlainMonthDay", r.newNativeFunc(r.temporalPlainDateToPlainMonthDay, "toPlainMonthDay", 0), true, false, true)
		o._putSym(SymToStringTag, &valueProperty{
			value: asciiString("Temporal.PlainDate"), writable: false, enumerable: false, configurable: true,
		})
	}
	return ret
}

func (r *Runtime) newTemporalPlainDateObject(d isoDate, calendar string, proto *Object) *Object {
	v := &Object{runtime: r}
	o := &temporalPlainDateObject{date: d, calendar: calendar}
	o.baseObject = baseObject{class: classObject, val: v, extensible: true, prototype: proto}
	v.self = o
	o.init()
	return v
}

func (r *Runtime) toTemporalPlainDate(v Value) *temporalPlainDateObject {
	o := r.toObject(v)
	if d, ok := o.self.(*temporalPlainDateObject); ok {
		return d
	}
	panic(r.NewTypeError("Value is not a Temporal.PlainDate"))
}

func (r *Runtime) temporalPlainDateFrom(call FunctionCall) Value {
	d := r.temporalPlainDateFromValue(call.Argument(0), call.Argument(1))
	return r.newTemporalPlainDateObject(d.date, d.calendar, r.getTemporalPlainDatePrototype())
}

func (r *Runtime) temporalPlainDateFromValue(v Value, options Value) *temporalPlainDateObject {
	if o, ok := v.(*Object); ok {
		if d, ok := o.self.(*temporalPlainDateObject); ok {
			_ = r.temporalOverflowOption(options)
			return d
		}
		if z, ok := o.self.(*temporalZonedDateTimeObject); ok {
			_ = r.temporalOverflowOption(options)
			d, _, _ := z.zone.localDateTime(z.epochNs)
			return r.plainDateFieldsObject(d, z.calendar)
		}
		if dt, ok := o.self.(*temporalPlainDateTimeObject); ok {
			_ = r.temporalOverflowOption(options)
			return r.plainDateFieldsObject(dt.date, dt.calendar)
		}
	}
	if s, ok := v.(String); ok {
		d, cal, ok := parseTemporalPlainDateString(s.String())
		if !ok {
			if dt, _, dtCalendar, dtOK := parseTemporalPlainDateTimeString(s.String()); dtOK {
				d, cal, ok = dt, dtCalendar, true
			}
		}
		if !ok {
			panic(r.newError(r.getRangeError(), "Invalid ISO date string: "+s.String()))
		}
		if !d.isValid() {
			panic(r.newError(r.getRangeError(), "Date is out of range"))
		}
		_ = r.temporalOverflowOption(options)
		return r.plainDateFieldsObject(d, cal)
	}
	obj, ok := v.(*Object)
	if !ok {
		panic(r.NewTypeError("Cannot convert %s to Temporal.PlainDate", v))
	}
	var calendar string
	var year, month, day int64
	var haveYear, haveMonth, haveDay bool
	var monthCode string
	// PrepareTemporalFields alphabetically.
	if cv := obj.self.getStr("calendar", nil); cv != nil && !IsUndefined(cv) {
		calendar = r.temporalCalendarFromValue(cv)
	} else {
		calendar = "iso8601"
	}
	if dv := obj.self.getStr("day", nil); dv != nil && !IsUndefined(dv) {
		day = r.temporalPositiveIntegerValue(dv)
		haveDay = true
	}
	if mv := obj.self.getStr("month", nil); mv != nil && !IsUndefined(mv) {
		month = r.temporalPositiveIntegerValue(mv)
		haveMonth = true
	}
	hasMonthCode := false
	if mcv := obj.self.getStr("monthCode", nil); mcv != nil && !IsUndefined(mcv) {
		s := r.temporalRequireString(mcv)
		if !wellFormedMonthCode(s) {
			panic(r.newError(r.getRangeError(), "Invalid monthCode"))
		}
		monthCode = s
		hasMonthCode = true
	}
	if yv := obj.self.getStr("year", nil); yv != nil && !IsUndefined(yv) {
		year = r.temporalIntegerFromValue(yv)
		haveYear = true
	}
	overflow := r.temporalOverflowOption(options)
	if !haveYear || (!haveMonth && !hasMonthCode) || !haveDay {
		panic(r.NewTypeError("year, month and day are required"))
	}
	if hasMonthCode {
		n, ok := isoMonthFromMonthCode(monthCode)
		if !ok {
			panic(r.newError(r.getRangeError(), "Invalid monthCode"))
		}
		if haveMonth && month != n {
			panic(r.newError(r.getRangeError(), "month and monthCode do not match"))
		}
		month = n
	}
	d := isoDate{year: year, month: month, day: day}
	if overflow == "constrain" {
		if d.month < 1 || d.day < 1 {
			panic(r.newError(r.getRangeError(), "Invalid ISO date"))
		}
		if d.month > 12 {
			d.month = 12
		}
		if max := isoDaysInMonth(d.year, d.month); d.day > max {
			d.day = max
		}
	}
	if !d.isValid() {
		panic(r.newError(r.getRangeError(), "Invalid ISO date"))
	}
	return r.plainDateFieldsObject(d, calendar)
}

func (r *Runtime) temporalOverflowOption(options Value) string {
	if options == nil || IsUndefined(options) {
		return "constrain"
	}
	obj, ok := options.(*Object)
	if !ok {
		panic(r.NewTypeError("options must be an object"))
	}
	if obj == nil {
		return "constrain"
	}
	v := obj.self.getStr("overflow", nil)
	if v == nil || IsUndefined(v) {
		return "constrain"
	}
	s, _ := temporalToStringOption(r, v)
	if s != "constrain" && s != "reject" {
		panic(r.newError(r.getRangeError(), "Invalid overflow option"))
	}
	return s
}

func (r *Runtime) temporalIntegerFromValue(v Value) int64 {
	if _, ok := v.(*Symbol); ok {
		panic(r.NewTypeError("Cannot convert a Symbol to a number"))
	}
	f := v.ToNumber().ToFloat()
	if math.IsNaN(f) || math.IsInf(f, 0) {
		panic(r.newError(r.getRangeError(), "Invalid date component"))
	}
	return int64(math.Trunc(f))
}

func (r *Runtime) plainDateFieldsObject(d isoDate, calendar string) *temporalPlainDateObject {
	o := r.newTemporalPlainDateObject(d, calendar, r.getTemporalPlainDatePrototype())
	return o.self.(*temporalPlainDateObject)
}

// parseTemporalPlainDateString parses the date portion of an ISO 8601 string
// accepted by Temporal.PlainDate.
func parseTemporalPlainDateString(s string) (isoDate, string, bool) {
	var d isoDate
	calendar := "iso8601"
	var annotations []string
	if k := strings.IndexByte(s, '['); k >= 0 {
		rest := s[k:]
		for len(rest) > 0 {
			if rest[0] != '[' {
				return d, "", false
			}
			j := strings.IndexByte(rest, ']')
			if j < 0 {
				return d, "", false
			}
			annotations = append(annotations, rest[1:j])
			rest = rest[j+1:]
		}
		s = strings.TrimSpace(s[:k])
	}

	tzAnnotations := 0
	calendarAnnotations := 0
	calendarCritical := false
	for _, ann := range annotations {
		if ann == "" {
			return d, "", false
		}
		critical := strings.HasPrefix(ann, "!")
		if critical {
			ann = ann[1:]
		}
		key, val, hasVal := strings.Cut(ann, "=")
		if hasVal {
			if key != strings.ToLower(key) || key == "" {
				return d, "", false
			}
			if key == "u-ca" {
				calendarAnnotations++
				if calendarAnnotations == 1 {
					if !asciiEqualFold(val, "iso8601") {
						return d, "", false
					}
					calendarCritical = critical
				} else if critical || calendarCritical {
					return d, "", false
				}
				continue
			}
		}
		if isTimeZoneAnnotation(ann) {
			tzAnnotations++
			if tzAnnotations > 1 {
				return d, "", false
			}
			continue
		}
		if critical {
			return d, "", false
		}
	}

	if norm, ok := normalizeBasicDate(s); ok {
		s = norm
	}
	i := 0
	sign := int64(1)
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		if s[i] == '-' {
			sign = -1
		}
		i++
	}
	startYear := i
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == startYear {
		return d, "", false
	}
	yearDigits := s[startYear:i]
	if len(yearDigits) != 4 && len(yearDigits) != 6 {
		return d, "", false
	}
	yv, err := strconv.ParseInt(yearDigits, 10, 64)
	if err != nil {
		return d, "", false
	}
	if sign < 0 {
		if yv == 0 {
			// -000000 and -0000 are not valid ISO years.
			return d, "", false
		}
		yv = -yv
	}
	if i >= len(s) || s[i] != '-' {
		return d, "", false
	}
	i++
	ms := i
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i-ms != 2 {
		return d, "", false
	}
	month, _ := strconv.ParseInt(s[ms:i], 10, 64)
	if i >= len(s) || s[i] != '-' {
		return d, "", false
	}
	i++
	ds := i
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i-ds != 2 {
		return d, "", false
	}
	day, _ := strconv.ParseInt(s[ds:i], 10, 64)
	d = isoDate{year: yv, month: month, day: day}

	if i == len(s) {
		return d, calendar, true
	}
	// A date-time string is accepted as long as the time is not an exact time
	// (no UTC designator).
	if s[i] != 'T' && s[i] != 't' && s[i] != ' ' {
		return d, "", false
	}
	i++
	if !parseTemporalTimeAndOffset(s[i:], false) {
		return d, "", false
	}
	return d, calendar, true
}

// parseTemporalTimeAndOffset validates the time part (and optional offset) of
// an ISO date-time string. UTC designators are rejected for PlainDate.
func parseTemporalTimeAndOffset(s string, allowZ bool) bool {
	i := 0
	if len(s) < 2 || !isDigits(s[:2]) {
		return false
	}
	hour, _ := strconv.Atoi(s[:2])
	if hour > 23 {
		return false
	}
	i = 2
	if i < len(s) && s[i] == ':' {
		i++
		if i+2 > len(s) || !isDigits(s[i:i+2]) {
			return false
		}
		minute, _ := strconv.Atoi(s[i : i+2])
		if minute > 59 {
			return false
		}
		i += 2
		if i < len(s) && s[i] == ':' {
			i++
			if i+2 > len(s) || !isDigits(s[i:i+2]) {
				return false
			}
			second, _ := strconv.Atoi(s[i : i+2])
			if second > 60 {
				return false
			}
			i += 2
			if i < len(s) && (s[i] == '.' || s[i] == ',') {
				i++
				fs := i
				for i < len(s) && s[i] >= '0' && s[i] <= '9' {
					i++
				}
				if i == fs || i-fs > 9 {
					return false
				}
			}
		}
	}
	if i >= len(s) {
		return true
	}
	if s[i] == 'Z' || s[i] == 'z' {
		return allowZ
	}
	if s[i] != '+' && s[i] != '-' {
		return false
	}
	i++
	// Offset: HH[:MM[:SS[.fff]]] or HHMM[SS[.fff]]
	if i+2 > len(s) || !isDigits(s[i:i+2]) {
		return false
	}
	oh, _ := strconv.Atoi(s[i : i+2])
	if oh > 23 {
		return false
	}
	i += 2
	parsePair := func() bool {
		if i < len(s) && s[i] == ':' {
			i++
			if i+2 > len(s) || !isDigits(s[i:i+2]) {
				return false
			}
			i += 2
			return true
		}
		if i+2 <= len(s) && isDigits(s[i:i+2]) {
			i += 2
			return true
		}
		return true
	}
	if i < len(s) && (s[i] == ':' || (s[i] >= '0' && s[i] <= '9')) {
		if !parsePair() {
			return false
		}
	}
	if i < len(s) && (s[i] == ':' || (s[i] >= '0' && s[i] <= '9')) {
		if !parsePair() {
			return false
		}
		if i < len(s) && (s[i] == '.' || s[i] == ',') {
			i++
			fs := i
			for i < len(s) && s[i] >= '0' && s[i] <= '9' {
				i++
			}
			if i == fs || i-fs > 9 {
				return false
			}
		}
	}
	return i == len(s)
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func isTimeZoneAnnotation(ann string) bool {
	if asciiEqualFold(ann, "UTC") || asciiEqualFold(ann, "GMT") || ann == "Z" || ann == "z" {
		return true
	}
	if strings.ContainsAny(ann, "/") {
		return true
	}
	if len(ann) > 0 && (ann[0] == '+' || ann[0] == '-') {
		return true
	}
	return false
}

func hasOwnMonth(n *Object) bool {
	v := n.self.getStr("month", nil)
	return v != nil && !IsUndefined(v)
}

func (r *Runtime) temporalPlainDateWith(call FunctionCall) Value {
	d := r.toTemporalPlainDate(call.This)
	partial := r.toObject(call.Argument(0))
	switch partial.self.(type) {
	case *temporalPlainDateObject, *temporalPlainTimeObject, *temporalPlainDateTimeObject,
		*temporalInstantObject, *temporalZonedDateTimeObject, *temporalDurationObject,
		*temporalPlainYearMonthObject, *temporalPlainMonthDayObject:
		panic(r.NewTypeError("with() argument must be a plain object"))
	}
	if cv := partial.self.getStr("calendar", nil); cv != nil && !IsUndefined(cv) {
		panic(r.NewTypeError("calendar is not allowed in with()"))
	}
	if tv := partial.self.getStr("timeZone", nil); tv != nil && !IsUndefined(tv) {
		panic(r.NewTypeError("timeZone is not allowed in with()"))
	}
	date := d.date
	found := false
	monthProvided := false
	var monthCode string
	hasMonthCode := false
	if dv := partial.self.getStr("day", nil); dv != nil && !IsUndefined(dv) {
		date.day = r.temporalPositiveIntegerValue(dv)
		found = true
	}
	if mv := partial.self.getStr("month", nil); mv != nil && !IsUndefined(mv) {
		date.month = r.temporalPositiveIntegerValue(mv)
		monthProvided = true
		found = true
	}
	if mcv := partial.self.getStr("monthCode", nil); mcv != nil && !IsUndefined(mcv) {
		monthCode = r.temporalRequireString(mcv)
		if !wellFormedMonthCode(monthCode) {
			panic(r.newError(r.getRangeError(), "Invalid monthCode"))
		}
		hasMonthCode = true
		found = true
	}
	if yv := partial.self.getStr("year", nil); yv != nil && !IsUndefined(yv) {
		date.year = r.temporalIntegerFromValue(yv)
		found = true
	}
	overflow := r.temporalOverflowOption(call.Argument(1))
	if !found {
		panic(r.NewTypeError("PlainDate.with requires at least one property"))
	}
	if hasMonthCode {
		n, ok := isoMonthFromMonthCode(monthCode)
		if !ok {
			panic(r.newError(r.getRangeError(), "Invalid monthCode"))
		}
		if monthProvided && date.month != n {
			panic(r.newError(r.getRangeError(), "month and monthCode do not match"))
		}
		date.month = n
	}
	if overflow == "constrain" {
		if date.month < 1 || date.day < 1 {
			panic(r.newError(r.getRangeError(), "Invalid ISO date"))
		}
		if date.month > 12 {
			date.month = 12
		}
		if max := isoDaysInMonth(date.year, date.month); date.day > max {
			date.day = max
		}
	}
	if !date.isValid() {
		panic(r.newError(r.getRangeError(), "Invalid ISO date"))
	}
	return r.newTemporalPlainDateObject(date, d.calendar, r.getTemporalPlainDatePrototype())
}

func (r *Runtime) temporalPlainDateWithCalendar(call FunctionCall) Value {
	d := r.toTemporalPlainDate(call.This)
	cal := r.temporalCalendarFromValue(call.Argument(0))
	return r.newTemporalPlainDateObject(d.date, cal, r.getTemporalPlainDatePrototype())
}

func (r *Runtime) temporalPlainDateAdd(call FunctionCall) Value {
	return r.temporalPlainDateAddSub(call, false)
}

func (r *Runtime) temporalPlainDateSubtract(call FunctionCall) Value {
	return r.temporalPlainDateAddSub(call, true)
}

func (r *Runtime) temporalPlainDateAddSub(call FunctionCall, subtract bool) Value {
	d := r.toTemporalPlainDate(call.This)
	dur := r.temporalDurationFromValue(call.Argument(0))
	overflow := r.temporalOverflowOption(call.Argument(1))
	fields := dur.fields()
	sign := int64(1)
	if subtract {
		sign = -1
	}
	// Time components are ignored for date arithmetic (the date is treated as
	// starting at midnight and the time part is discarded).
	years := int64(fields[0]) * sign
	months := int64(fields[1]) * sign
	weeks := int64(fields[2]) * sign
	days := int64(fields[3]) * sign
	// Balance the time part into whole days (truncating towards zero).
	timeNs := new(big.Rat)
	addTime := func(v float64, mul int64) {
		if v != 0 {
			timeNs.Add(timeNs, new(big.Rat).Mul(new(big.Rat).SetFloat64(v), big.NewRat(mul, 1)))
		}
	}
	addTime(fields[4], 3600*1e9)
	addTime(fields[5], 60*1e9)
	addTime(fields[6], 1e9)
	addTime(fields[7], 1e6)
	addTime(fields[8], 1e3)
	addTime(fields[9], 1)
	if subtract {
		timeNs.Neg(timeNs)
	}
	dayNs := big.NewRat(86400*1e9, 1)
	q := new(big.Rat).Quo(timeNs, dayNs)
	extra := new(big.Int).Quo(q.Num(), q.Denom())
	days += extra.Int64()
	date := d.date
	if years != 0 || months != 0 {
		totalMonths := date.year*12 + (date.month - 1) + years*12 + months
		y := floorDiv(totalMonths, 12)
		m := totalMonths - y*12 + 1
		day := date.day
		if overflow == "constrain" {
			if max := isoDaysInMonth(y, m); day > max {
				day = max
			}
		}
		date = isoDate{year: y, month: m, day: day}
		if !date.isValid() {
			panic(r.newError(r.getRangeError(), "Resulting date is out of range"))
		}
	}
	days += weeks * 7
	if days != 0 {
		z := date.days() + days
		if z < temporalMinISODate || z > temporalMaxISODate {
			panic(r.newError(r.getRangeError(), "Resulting date is out of range"))
		}
		date.year, date.month, date.day = isoCivilFromDays(z)
	}
	return r.newTemporalPlainDateObject(date, d.calendar, r.getTemporalPlainDatePrototype())
}

func (r *Runtime) temporalPlainDateUntil(call FunctionCall) Value {
	d := r.toTemporalPlainDate(call.This)
	other := r.temporalPlainDateFromValue(call.Argument(0), _undefined)
	return r.temporalPlainDateDifference(d, other, call.Argument(1), false)
}

func (r *Runtime) temporalPlainDateSince(call FunctionCall) Value {
	d := r.toTemporalPlainDate(call.This)
	other := r.temporalPlainDateFromValue(call.Argument(0), _undefined)
	return r.temporalPlainDateDifference(d, other, call.Argument(1), true)
}

func (r *Runtime) temporalPlainDateDifference(a, b *temporalPlainDateObject, options Value, since bool) Value {
	s := r.temporalGetDifferenceSettings(options, "day", "day", 0, 3)
	if since {
		s.mode = negateRoundingMode(s.mode)
	}
	fields := r.temporalDiffISODateTime(a.date, isoTime{}, b.date, isoTime{}, s)
	if since {
		for i := range fields {
			fields[i] = -fields[i]
		}
	}
	r.validateTemporalDuration(fields)
	return r.newTemporalDurationObject(fields, r.getTemporalDurationPrototype())
}

// roundDateIncrement rounds a non-negative integer to a multiple of increment.
func roundDateIncrement(v, increment int64, mode temporalRoundingMode, negative bool) int64 {
	if increment <= 1 {
		return v
	}
	q := v / increment
	rem := v % increment
	if rem != 0 {
		up := false
		switch mode {
		case temporalRoundExpand:
			up = true
		case temporalRoundCeil:
			up = !negative
		case temporalRoundFloor:
			up = negative
		case temporalRoundTrunc:
		case temporalRoundHalfCeil, temporalRoundHalfFloor, temporalRoundHalfExpand,
			temporalRoundHalfTrunc, temporalRoundHalfEven:
			if rem*2 > increment {
				up = true
			} else if rem*2 == increment {
				switch mode {
				case temporalRoundHalfCeil:
					up = !negative
				case temporalRoundHalfFloor:
					up = negative
				case temporalRoundHalfExpand:
					up = true
				case temporalRoundHalfEven:
					up = q%2 == 1
				}
			}
		}
		if up {
			q++
		}
	}
	return q * increment
}

// shouldRoundUp decides whether a fractional part (num/den) rounds up.
func shouldRoundUp(num, den int64, mode temporalRoundingMode, negative bool) bool {
	if num == 0 || den <= 0 {
		return false
	}
	switch mode {
	case temporalRoundExpand:
		return true
	case temporalRoundCeil:
		return !negative
	case temporalRoundFloor:
		return negative
	case temporalRoundTrunc:
		return false
	case temporalRoundHalfCeil, temporalRoundHalfFloor, temporalRoundHalfExpand,
		temporalRoundHalfTrunc, temporalRoundHalfEven:
		if num*2 > den {
			return true
		}
		if num*2 < den {
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
			return false
		default:
			return true
		}
	}
	return false
}

func addISOMonths(d isoDate, months int64) isoDate {
	tm := d.year*12 + (d.month - 1) + months
	y := floorDiv(tm, 12)
	m := tm - y*12 + 1
	day := d.day
	if max := isoDaysInMonth(y, m); day > max {
		day = max
	}
	return isoDate{year: y, month: m, day: day}
}

func (r *Runtime) temporalPlainDateToPlainYearMonth(call FunctionCall) Value {
	d := r.toTemporalPlainDate(call.This)
	return r.newTemporalPlainYearMonthObject(d.date.year, d.date.month, 1, d.calendar, r.getTemporalPlainYearMonthPrototype())
}

func (r *Runtime) temporalPlainDateToPlainMonthDay(call FunctionCall) Value {
	d := r.toTemporalPlainDate(call.This)
	return r.newTemporalPlainMonthDayObject(d.date.month, d.date.day, 1972, d.calendar, r.getTemporalPlainMonthDayPrototype())
}

func (r *Runtime) temporalPlainDateToZonedDateTime(call FunctionCall) Value {
	d := r.toTemporalPlainDate(call.This)
	arg := call.Argument(0)
	var zone *temporalZone
	tm := isoTime{}
	if o, ok := arg.(*Object); ok {
		if z, ok := o.self.(*temporalZonedDateTimeObject); ok {
			zone = z.zone
		} else {
			v := o.self.getStr("timeZone", nil)
			if v == nil || IsUndefined(v) {
				panic(r.NewTypeError("timeZone is required"))
			}
			zone = r.parseTemporalZone(v)
			if pt := o.self.getStr("plainTime", nil); pt != nil && !IsUndefined(pt) {
				t := r.temporalPlainTimeFromValue(pt, _undefined)
				tm = isoTime{hour: t.hour, minute: t.minute, second: t.second,
					millisecond: t.millisecond, microsecond: t.microsecond, nanosecond: t.nanosecond}
			}
		}
	} else {
		zone = r.parseTemporalZone(arg)
	}
	local := new(big.Int).Mul(big.NewInt(d.date.days()), temporalBigDayNs)
	local.Add(local, big.NewInt(tm.nanos()))
	ns := zone.instantFromLocal(local)
	if ns == nil || !r.validInstant(ns) {
		panic(r.newError(r.getRangeError(), "ZonedDateTime is out of range"))
	}
	return r.newTemporalZonedDateTimeObject(ns, zone, d.calendar, r.getTemporalZonedDateTimePrototype())
}

func (r *Runtime) temporalPlainDateToPlainDateTime(call FunctionCall) Value {
	d := r.toTemporalPlainDate(call.This)
	tm := isoTime{}
	v := call.Argument(0)
	if v != nil && !IsUndefined(v) {
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

func (r *Runtime) temporalPlainDateEquals(call FunctionCall) Value {
	a := r.toTemporalPlainDate(call.This)
	b := r.temporalPlainDateFromValue(call.Argument(0), _undefined)
	return r.ToValue(a.date == b.date && a.calendar == b.calendar)
}

func (r *Runtime) temporalPlainDateCompare(call FunctionCall) Value {
	a := r.temporalPlainDateFromValue(call.Argument(0), _undefined)
	b := r.temporalPlainDateFromValue(call.Argument(1), _undefined)
	ad, bd := a.date.days(), b.date.days()
	if ad < bd {
		return intToValue(-1)
	}
	if ad > bd {
		return intToValue(1)
	}
	return intToValue(0)
}

func formatISOYear(year int64) string {
	switch {
	case year >= 0 && year <= 9999:
		return fmt.Sprintf("%04d", year)
	case year > 9999:
		return fmt.Sprintf("+%06d", year)
	default:
		return fmt.Sprintf("-%06d", -year)
	}
}

func formatISODate(d isoDate) string {
	return formatISOYear(d.year) + fmt.Sprintf("-%02d-%02d", d.month, d.day)
}

func (r *Runtime) temporalPlainDateToString(call FunctionCall) Value {
	d := r.toTemporalPlainDate(call.This)
	calendarName := "auto"
	if options := call.Argument(0); options != nil && !IsUndefined(options) {
		obj, ok := options.(*Object)
		if !ok {
			panic(r.NewTypeError("options must be an object"))
		}
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
	s := formatISODate(d.date)
	switch calendarName {
	case "always":
		s += "[u-ca=" + d.calendar + "]"
	case "critical":
		s += "[!u-ca=" + d.calendar + "]"
	}
	return r.ToValue(s)
}

func (r *Runtime) temporalPlainDateToJSON(call FunctionCall) Value {
	return r.temporalPlainDateToString(FunctionCall{This: call.This, Arguments: []Value{_undefined}})
}

func (r *Runtime) temporalPlainDateToLocaleString(call FunctionCall) Value {
	return r.temporalPlainDateToString(FunctionCall{This: call.This, Arguments: []Value{_undefined}})
}

func (r *Runtime) temporalPlainDateValueOf(call FunctionCall) Value {
	panic(r.NewTypeError("Temporal.PlainDate objects cannot be converted to primitives"))
}

func (r *Runtime) temporalPlainDateGetISOFields(call FunctionCall) Value {
	d := r.toTemporalPlainDate(call.This)
	o := r.NewObject()
	o.Set("isoYear", r.ToValue(d.date.year))
	o.Set("isoMonth", r.ToValue(d.date.month))
	o.Set("isoDay", r.ToValue(d.date.day))
	o.Set("calendar", r.ToValue(d.calendar))
	return o
}

func (r *Runtime) temporalPlainDateNotImplemented(call FunctionCall) Value {
	panic(r.NewTypeError("This Temporal.PlainDate method is not implemented yet"))
}
