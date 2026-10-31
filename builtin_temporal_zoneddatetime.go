package goja

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/dop251/goja/unistring"
)

type temporalZonedDateTimeObject struct {
	baseObject
	epochNs  *big.Int
	zone     *temporalZone
	calendar string
}

func (r *Runtime) getTemporalZonedDateTime() *Object {
	ret := r.global.TemporalZonedDateTime
	if ret == nil {
		ret = &Object{runtime: r}
		r.global.TemporalZonedDateTime = ret
		r.newNativeFuncAndConstruct(ret, func(call FunctionCall) Value {
			panic(r.NewTypeError("Constructor Temporal.ZonedDateTime requires 'new'"))
		}, func(args []Value, newTarget *Object) *Object {
			argAt := func(i int) Value {
				if i < len(args) {
					return args[i]
				}
				return _undefined
			}
			ns := r.temporalBigIntValue(argAt(0))
			if !r.validInstant(ns) {
				panic(r.newError(r.getRangeError(), "ZonedDateTime is out of range"))
			}
			zone := r.parseTemporalZoneIdentifierStrict(argAt(1))
			calendar := "iso8601"
			if !IsUndefined(argAt(2)) {
				calendar = r.temporalCalendarID(argAt(2))
			}
			proto := r.getTemporalZonedDateTimePrototype()
			if newTarget != nil {
				proto = r.getPrototypeFromCtor(newTarget, ret, proto)
			}
			return r.newTemporalZonedDateTimeObject(ns, zone, calendar, proto)
		}, r.getTemporalZonedDateTimePrototype(), "ZonedDateTime", intToValue(2))
		ctor := ret.self.(*nativeFuncObject)
		ctor._putProp("from", r.newNativeFunc(r.temporalZonedDateTimeFrom, "from", 1), true, false, true)
		ctor._putProp("compare", r.newNativeFunc(r.temporalZonedDateTimeCompare, "compare", 2), true, false, true)
	}
	return ret
}

func (r *Runtime) getTemporalZonedDateTimePrototype() *Object {
	ret := r.global.TemporalZonedDateTimePrototype
	if ret == nil {
		o := r.newBaseObject(r.global.ObjectPrototype, classObject)
		ret = o.val
		r.global.TemporalZonedDateTimePrototype = ret
		o._putProp("constructor", r.getTemporalZonedDateTime(), true, false, true)
		addGetter := func(name string, f func(*temporalZonedDateTimeObject) Value) {
			o.setOwnStr(unistring.NewFromString(name), &valueProperty{
				getterFunc: r.newNativeFunc(func(call FunctionCall) Value {
					return f(r.toTemporalZonedDateTime(call.This))
				}, unistring.NewFromString("get "+name), 0),
				accessor: true, enumerable: false, configurable: true,
			}, true)
		}
		addGetter("epochMilliseconds", func(z *temporalZonedDateTimeObject) Value {
			q, rem := new(big.Int).QuoRem(z.epochNs, big.NewInt(1e6), new(big.Int))
			if rem.Sign() < 0 {
				q.Sub(q, big.NewInt(1))
			}
			f, _ := new(big.Float).SetInt(q).Float64()
			return r.ToValue(f)
		})
		addGetter("epochNanoseconds", func(z *temporalZonedDateTimeObject) Value {
			return r.ToValue(new(big.Int).Set(z.epochNs))
		})
		addGetter("timeZoneId", func(z *temporalZonedDateTimeObject) Value { return r.ToValue(z.zone.id) })
		addGetter("calendarId", func(z *temporalZonedDateTimeObject) Value { return r.ToValue(z.calendar) })
		addGetter("offsetNanoseconds", func(z *temporalZonedDateTimeObject) Value {
			off := z.zone.offsetAt(z.epochNs)
			f, _ := new(big.Float).SetInt(off).Float64()
			return r.ToValue(f)
		})
		addGetter("offset", func(z *temporalZonedDateTimeObject) Value {
			return r.ToValue(formatOffsetNano(z.zone.offsetAt(z.epochNs)))
		})
		addGetter("year", func(z *temporalZonedDateTimeObject) Value {
			d, _, _ := z.zone.localDateTime(z.epochNs)
			return intToValue(d.year)
		})
		addGetter("month", func(z *temporalZonedDateTimeObject) Value {
			d, _, _ := z.zone.localDateTime(z.epochNs)
			return intToValue(d.month)
		})
		addGetter("monthCode", func(z *temporalZonedDateTimeObject) Value {
			d, _, _ := z.zone.localDateTime(z.epochNs)
			return asciiString(fmt.Sprintf("M%02d", d.month))
		})
		addGetter("day", func(z *temporalZonedDateTimeObject) Value {
			d, _, _ := z.zone.localDateTime(z.epochNs)
			return intToValue(d.day)
		})
		addGetter("hour", func(z *temporalZonedDateTimeObject) Value {
			_, t, _ := z.zone.localDateTime(z.epochNs)
			return intToValue(t.hour)
		})
		addGetter("minute", func(z *temporalZonedDateTimeObject) Value {
			_, t, _ := z.zone.localDateTime(z.epochNs)
			return intToValue(t.minute)
		})
		addGetter("second", func(z *temporalZonedDateTimeObject) Value {
			_, t, _ := z.zone.localDateTime(z.epochNs)
			return intToValue(t.second)
		})
		addGetter("millisecond", func(z *temporalZonedDateTimeObject) Value {
			_, t, _ := z.zone.localDateTime(z.epochNs)
			return intToValue(t.millisecond)
		})
		addGetter("microsecond", func(z *temporalZonedDateTimeObject) Value {
			_, t, _ := z.zone.localDateTime(z.epochNs)
			return intToValue(t.microsecond)
		})
		addGetter("nanosecond", func(z *temporalZonedDateTimeObject) Value {
			_, t, _ := z.zone.localDateTime(z.epochNs)
			return intToValue(t.nanosecond)
		})
		addGetter("era", func(z *temporalZonedDateTimeObject) Value { return _undefined })
		addGetter("eraYear", func(z *temporalZonedDateTimeObject) Value { return _undefined })
		addGetter("dayOfWeek", func(z *temporalZonedDateTimeObject) Value {
			d, _, _ := z.zone.localDateTime(z.epochNs)
			return intToValue(((d.days()+3)%7+7)%7 + 1)
		})
		addGetter("dayOfYear", func(z *temporalZonedDateTimeObject) Value {
			d, _, _ := z.zone.localDateTime(z.epochNs)
			return intToValue(d.days() - isoDaysFromCivil(d.year, 1, 1) + 1)
		})
		addGetter("weekOfYear", func(z *temporalZonedDateTimeObject) Value {
			d, _, _ := z.zone.localDateTime(z.epochNs)
			w, _ := d.weekInfo()
			return intToValue(w)
		})
		addGetter("yearOfWeek", func(z *temporalZonedDateTimeObject) Value {
			d, _, _ := z.zone.localDateTime(z.epochNs)
			_, y := d.weekInfo()
			return intToValue(y)
		})
		addGetter("daysInWeek", func(z *temporalZonedDateTimeObject) Value { return intToValue(7) })
		addGetter("daysInMonth", func(z *temporalZonedDateTimeObject) Value {
			d, _, _ := z.zone.localDateTime(z.epochNs)
			return intToValue(isoDaysInMonth(d.year, d.month))
		})
		addGetter("daysInYear", func(z *temporalZonedDateTimeObject) Value {
			d, _, _ := z.zone.localDateTime(z.epochNs)
			if isoLeapYear(d.year) {
				return intToValue(366)
			}
			return intToValue(365)
		})
		addGetter("monthsInYear", func(z *temporalZonedDateTimeObject) Value { return intToValue(12) })
		addGetter("inLeapYear", func(z *temporalZonedDateTimeObject) Value {
			d, _, _ := z.zone.localDateTime(z.epochNs)
			return r.ToValue(isoLeapYear(d.year))
		})
		addGetter("hoursInDay", func(z *temporalZonedDateTimeObject) Value {
			d, _, _ := z.zone.localDateTime(z.epochNs)
			start := z.zone.startOfDay(d)
			end := z.zone.startOfDay(addISODays(d, 1))
			if start == nil || end == nil || !r.validInstant(start) || !r.validInstant(end) {
				panic(r.newError(r.getRangeError(), "Next day boundary is out of range"))
			}
			diff := new(big.Int).Sub(end, start)
			f, _ := new(big.Rat).SetFrac(diff, big.NewInt(3600*1e9)).Float64()
			return r.ToValue(f)
		})

		o._putProp("with", r.newNativeFunc(r.temporalZonedDateTimeWith, "with", 1), true, false, true)
		o._putProp("withTimeZone", r.newNativeFunc(r.temporalZonedDateTimeWithTimeZone, "withTimeZone", 1), true, false, true)
		o._putProp("withCalendar", r.newNativeFunc(r.temporalZonedDateTimeWithCalendar, "withCalendar", 1), true, false, true)
		o._putProp("withPlainTime", r.newNativeFunc(r.temporalZonedDateTimeWithPlainTime, "withPlainTime", 0), true, false, true)
		o._putProp("add", r.newNativeFunc(r.temporalZonedDateTimeAdd, "add", 1), true, false, true)
		o._putProp("subtract", r.newNativeFunc(r.temporalZonedDateTimeSubtract, "subtract", 1), true, false, true)
		o._putProp("until", r.newNativeFunc(r.temporalZonedDateTimeUntil, "until", 1), true, false, true)
		o._putProp("since", r.newNativeFunc(r.temporalZonedDateTimeSince, "since", 1), true, false, true)
		o._putProp("round", r.newNativeFunc(r.temporalZonedDateTimeRound, "round", 1), true, false, true)
		o._putProp("equals", r.newNativeFunc(r.temporalZonedDateTimeEquals, "equals", 1), true, false, true)
		o._putProp("toString", r.newNativeFunc(r.temporalZonedDateTimeToString, "toString", 0), true, false, true)
		o._putProp("toJSON", r.newNativeFunc(r.temporalZonedDateTimeToJSON, "toJSON", 0), true, false, true)
		o._putProp("toLocaleString", r.newNativeFunc(r.temporalZonedDateTimeToLocaleString, "toLocaleString", 0), true, false, true)
		o._putProp("valueOf", r.newNativeFunc(r.temporalZonedDateTimeValueOf, "valueOf", 0), true, false, true)
		o._putProp("startOfDay", r.newNativeFunc(r.temporalZonedDateTimeStartOfDay, "startOfDay", 0), true, false, true)
		o._putProp("toInstant", r.newNativeFunc(r.temporalZonedDateTimeToInstant, "toInstant", 0), true, false, true)
		o._putProp("toPlainDate", r.newNativeFunc(r.temporalZonedDateTimeToPlainDate, "toPlainDate", 0), true, false, true)
		o._putProp("toPlainTime", r.newNativeFunc(r.temporalZonedDateTimeToPlainTime, "toPlainTime", 0), true, false, true)
		o._putProp("toPlainDateTime", r.newNativeFunc(r.temporalZonedDateTimeToPlainDateTime, "toPlainDateTime", 0), true, false, true)
		o._putProp("getISOFields", r.newNativeFunc(r.temporalZonedDateTimeGetISOFields, "getISOFields", 0), true, false, true)
		o._putProp("getTimeZoneTransition", r.newNativeFunc(r.temporalZonedDateTimeGetTimeZoneTransition, "getTimeZoneTransition", 1), true, false, true)
		o._putSym(SymToStringTag, &valueProperty{
			value: asciiString("Temporal.ZonedDateTime"), writable: false, enumerable: false, configurable: true,
		})
	}
	return ret
}

func (z *temporalZone) startOfDay(d isoDate) *big.Int {
	local := new(big.Int).Mul(big.NewInt(d.days()), temporalBigDayNs)
	return z.instantFromLocal(local)
}

// findTransition returns the next or previous offset transition, or nil if
// there is none within 400 years.
func (z *temporalZone) findTransition(from *big.Int, dir int64) *big.Int {
	if z.offset != nil {
		return nil
	}
	off := z.offsetAt(from)
	oneDay := big.NewInt(86400 * 1e9)
	dist := new(big.Int).Set(oneDay)
	var hi *big.Int
	found := false
	for i := 0; i < 20; i++ {
		hi = new(big.Int).Add(from, new(big.Int).Mul(dist, big.NewInt(dir)))
		if z.offsetAt(hi).Cmp(off) != 0 {
			found = true
			break
		}
		dist.Mul(dist, big.NewInt(2))
	}
	if !found {
		return nil
	}
	lo := new(big.Int).Set(from)
	for {
		diff := new(big.Int).Sub(hi, lo)
		if diff.Sign() < 0 {
			diff.Neg(diff)
		}
		if diff.Cmp(big.NewInt(1)) <= 0 {
			break
		}
		mid := new(big.Int).Add(lo, hi)
		mid.Quo(mid, big.NewInt(2))
		if z.offsetAt(mid).Cmp(off) == 0 {
			lo = mid
		} else {
			hi = mid
		}
	}
	if dir > 0 {
		return hi
	}
	return lo
}

func (r *Runtime) newTemporalZonedDateTimeObject(ns *big.Int, zone *temporalZone, calendar string, proto *Object) *Object {
	v := &Object{runtime: r}
	o := &temporalZonedDateTimeObject{epochNs: new(big.Int).Set(ns), zone: zone, calendar: calendar}
	o.baseObject = baseObject{class: classObject, val: v, extensible: true, prototype: proto}
	v.self = o
	o.init()
	return v
}

func (r *Runtime) toTemporalZonedDateTime(v Value) *temporalZonedDateTimeObject {
	o := r.toObject(v)
	if z, ok := o.self.(*temporalZonedDateTimeObject); ok {
		return z
	}
	panic(r.NewTypeError("Value is not a Temporal.ZonedDateTime"))
}

func (r *Runtime) temporalZonedDateTimeFrom(call FunctionCall) Value {
	z := r.temporalZonedDateTimeFromValue(call.Argument(0), call.Argument(1))
	return r.newTemporalZonedDateTimeObject(z.epochNs, z.zone, z.calendar, r.getTemporalZonedDateTimePrototype())
}

func (r *Runtime) temporalZonedDateTimeFromValue(v Value, options Value) *temporalZonedDateTimeObject {
	if o, ok := v.(*Object); ok {
		if z, ok := o.self.(*temporalZonedDateTimeObject); ok {
			optsObj := r.temporalOptionsObject(options)
			_ = r.temporalGetDisambiguationOption(optsObj)
			_ = r.temporalGetOffsetOption(optsObj, "reject")
			_ = r.temporalOverflowOption(options)
			return z
		}
	}
	if s, ok := v.(String); ok {
		return r.parseTemporalZonedDateTimeString(s.String(), options)
	}
	obj, ok := v.(*Object)
	if !ok {
		panic(r.NewTypeError("Cannot convert %s to Temporal.ZonedDateTime", v))
	}
	switch obj.self.(type) {
	case *temporalPlainDateObject, *temporalPlainTimeObject, *temporalPlainDateTimeObject,
		*temporalInstantObject, *temporalDurationObject:
		panic(r.NewTypeError("Cannot convert a Temporal object to Temporal.ZonedDateTime"))
	}
	// Property bag: read all fields before converting options.
	res := r.temporalReadZonedDateTimeFields(obj, temporalDateTimeFieldsResult{calendar: "iso8601"}, false)
	optsObj := r.temporalOptionsObject(options)
	disambiguation := r.temporalGetDisambiguationOption(optsObj)
	offsetOpt := r.temporalGetOffsetOption(optsObj, "reject")
	overflow := r.temporalOverflowOption(options)
	if !res.haveYear || (!res.haveMonth && !res.hasMonthCd) || !res.haveDay {
		panic(r.NewTypeError("year, month and day are required"))
	}
	if !res.hasTZ {
		panic(r.NewTypeError("timeZone is required"))
	}
	zone := r.parseTemporalZone(res.tzValue)
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
	local := new(big.Int).Mul(big.NewInt(date.days()), temporalBigDayNs)
	local.Add(local, big.NewInt(tm.nanos()))
	ns := r.zonedResolve(zone, local, res.offset, res.hasOffset, disambiguation, offsetOpt)
	if ns == nil || !r.validInstant(ns) {
		panic(r.newError(r.getRangeError(), "ZonedDateTime is out of range"))
	}
	return r.newTemporalZonedDateTimeObject(ns, zone, res.calendar, r.getTemporalZonedDateTimePrototype()).self.(*temporalZonedDateTimeObject)
}

func (r *Runtime) parseTemporalZonedDateTimeString(s string, options Value) *temporalZonedDateTimeObject {
	base, anns, ok := splitTemporalAnnotations(s)
	if !ok {
		panic(r.newError(r.getRangeError(), "Invalid ISO date-time string"))
	}
	calendar := "iso8601"
	calSeen := false
	calCritical := false
	tzAnn := ""
	for _, ann := range anns {
		critical := strings.HasPrefix(ann, "!")
		if critical {
			ann = ann[1:]
		}
		key, val, hasVal := strings.Cut(ann, "=")
		if hasVal {
			if key != strings.ToLower(key) || key == "" {
				panic(r.newError(r.getRangeError(), "Invalid annotation"))
			}
			if key == "u-ca" {
				if calSeen && (critical || calCritical) {
					panic(r.newError(r.getRangeError(), "Multiple calendar annotations"))
				}
				if !calSeen && !asciiEqualFold(val, "iso8601") {
					panic(r.newError(r.getRangeError(), "Unsupported calendar"))
				}
				calSeen = true
				calCritical = critical
				continue
			}
		}
		if isTimeZoneAnnotation(ann) {
			if tzAnn != "" {
				panic(r.newError(r.getRangeError(), "Multiple time zone annotations"))
			}
			tzAnn = ann
			continue
		}
		if critical {
			panic(r.newError(r.getRangeError(), "Unknown critical annotation"))
		}
	}
	if tzAnn == "" {
		panic(r.newError(r.getRangeError(), "Time zone annotation is required"))
	}
	sep := strings.IndexAny(base, "Tt")
	if sep < 0 {
		if i := strings.IndexByte(base, ' '); i > 0 {
			sep = i
		}
	}
	datePart := base
	timePart := "00:00:00"
	if sep >= 0 {
		datePart = base[:sep]
		timePart = base[sep+1:]
	}
	if nd, ok := normalizeBasicDate(datePart); ok {
		datePart = nd
	}
	date, _, dok := parseTemporalPlainDateString(datePart)
	if !dok || !date.isValid() {
		panic(r.newError(r.getRangeError(), "Invalid ISO date-time string"))
	}
	offset := (*big.Int)(nil)
	zulu := false
	if strings.HasSuffix(timePart, "Z") || strings.HasSuffix(timePart, "z") {
		zulu = true
		timePart = timePart[:len(timePart)-1]
	} else if i := strings.LastIndexAny(timePart, "+-"); i >= 0 {
		off, ok := parseTemporalOffsetString(timePart[i:])
		if !ok {
			panic(r.newError(r.getRangeError(), "Invalid offset"))
		}
		offset = off
		timePart = timePart[:i]
	}
	h, mi, sec, ms, us, ns, tok := parseTemporalTimeFields(timePart)
	if !tok {
		panic(r.newError(r.getRangeError(), "Invalid ISO date-time string"))
	}
	optsObj := r.temporalOptionsObject(options)
	disambiguation := r.temporalGetDisambiguationOption(optsObj)
	offsetOpt := r.temporalGetOffsetOption(optsObj, "reject")
	_ = r.temporalOverflowOption(options)
	tm := isoTime{h, mi, sec, ms, us, ns}
	local := new(big.Int).Mul(big.NewInt(date.days()), temporalBigDayNs)
	local.Add(local, big.NewInt(tm.nanos()))
	zone := r.parseTemporalZoneString(tzAnn)
	var inst *big.Int
	if zulu {
		inst = local
	} else {
		inst = r.zonedResolve(zone, local, offset, offset != nil, disambiguation, offsetOpt)
	}
	if inst == nil || !r.validInstant(inst) {
		panic(r.newError(r.getRangeError(), "ZonedDateTime is out of range"))
	}
	return r.newTemporalZonedDateTimeObject(inst, zone, calendar, r.getTemporalZonedDateTimePrototype()).self.(*temporalZonedDateTimeObject)
}

func (r *Runtime) temporalZonedDateTimeWithTimeZone(call FunctionCall) Value {
	z := r.toTemporalZonedDateTime(call.This)
	zone := r.parseTemporalZone(call.Argument(0))
	return r.newTemporalZonedDateTimeObject(z.epochNs, zone, z.calendar, r.getTemporalZonedDateTimePrototype())
}

func (r *Runtime) temporalZonedDateTimeWithCalendar(call FunctionCall) Value {
	z := r.toTemporalZonedDateTime(call.This)
	cal := r.temporalCalendarFromValue(call.Argument(0))
	return r.newTemporalZonedDateTimeObject(z.epochNs, z.zone, cal, r.getTemporalZonedDateTimePrototype())
}

func (r *Runtime) temporalZonedDateTimeWith(call FunctionCall) Value {
	z := r.toTemporalZonedDateTime(call.This)
	partial := r.toObject(call.Argument(0))
	switch partial.self.(type) {
	case *temporalPlainDateObject, *temporalPlainTimeObject, *temporalPlainDateTimeObject,
		*temporalInstantObject, *temporalZonedDateTimeObject, *temporalDurationObject:
		panic(r.NewTypeError("with() argument must be a plain object"))
	}
	d, tm, _ := z.zone.localDateTime(z.epochNs)
	base := temporalDateTimeFieldsResult{date: d, time: tm, calendar: z.calendar, haveYear: true, haveMonth: true, haveDay: true}
	res := r.temporalReadZonedDateTimeFields(partial, base, true)
	if res.hasTZ {
		panic(r.NewTypeError("timeZone is not allowed in with()"))
	}
	if res.hasCalendar {
		panic(r.NewTypeError("calendar is not allowed in with()"))
	}
	optsObj := r.temporalOptionsObject(call.Argument(1))
	disambiguation := r.temporalGetDisambiguationOption(optsObj)
	offsetOpt := r.temporalGetOffsetOption(optsObj, "prefer")
	overflow := r.temporalOverflowOption(call.Argument(1))
	if !res.provided {
		panic(r.NewTypeError("ZonedDateTime.with requires at least one property"))
	}
	date, timeOfDay := res.date, res.time
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
		timeOfDay.hour = clampInt(timeOfDay.hour, 0, 23)
		timeOfDay.minute = clampInt(timeOfDay.minute, 0, 59)
		timeOfDay.second = clampInt(timeOfDay.second, 0, 59)
		timeOfDay.millisecond = clampInt(timeOfDay.millisecond, 0, 999)
		timeOfDay.microsecond = clampInt(timeOfDay.microsecond, 0, 999)
		timeOfDay.nanosecond = clampInt(timeOfDay.nanosecond, 0, 999)
	}
	if !date.isValidDateTime(timeOfDay) {
		panic(r.newError(r.getRangeError(), "Invalid date/time"))
	}
	local := new(big.Int).Mul(big.NewInt(date.days()), temporalBigDayNs)
	local.Add(local, big.NewInt(timeOfDay.nanos()))
	ns := r.zonedResolve(z.zone, local, res.offset, res.hasOffset, disambiguation, offsetOpt)
	if ns == nil || !r.validInstant(ns) {
		panic(r.newError(r.getRangeError(), "ZonedDateTime is out of range"))
	}
	return r.newTemporalZonedDateTimeObject(ns, z.zone, res.calendar, r.getTemporalZonedDateTimePrototype())
}

func (r *Runtime) temporalZonedDateTimeWithPlainTime(call FunctionCall) Value {
	z := r.toTemporalZonedDateTime(call.This)
	d, tm, _ := z.zone.localDateTime(z.epochNs)
	v := call.Argument(0)
	if v == nil || IsUndefined(v) {
		tm = isoTime{}
	} else {
		t := r.temporalPlainTimeFromValue(v, _undefined)
		tm = isoTime{hour: t.hour, minute: t.minute, second: t.second,
			millisecond: t.millisecond, microsecond: t.microsecond, nanosecond: t.nanosecond}
	}
	local := new(big.Int).Mul(big.NewInt(d.days()), temporalBigDayNs)
	local.Add(local, big.NewInt(tm.nanos()))
	ns := z.zone.instantFromLocal(local)
	if ns == nil || !r.validInstant(ns) {
		panic(r.newError(r.getRangeError(), "ZonedDateTime is out of range"))
	}
	return r.newTemporalZonedDateTimeObject(ns, z.zone, z.calendar, r.getTemporalZonedDateTimePrototype())
}

// zonedAdd adds a duration to an instant in the given zone. The calendar part
// is applied to the wall clock, and the time part in exact time.
func (r *Runtime) zonedAdd(ns *big.Int, zone *temporalZone, fields [10]float64, sign int64) *big.Int {
	return r.zonedAddWithOverflow(ns, zone, fields, sign, false)
}

func (r *Runtime) zonedAddWithOverflow(ns *big.Int, zone *temporalZone, fields [10]float64, sign int64, reject bool) *big.Int {
	d, tm, ok := zone.localDateTime(ns)
	if !ok {
		panic(r.newError(r.getRangeError(), "ZonedDateTime is out of range"))
	}
	months := (int64(fields[0])*12 + int64(fields[1])) * sign
	days := (int64(fields[2])*7 + int64(fields[3])) * sign
	if reject && months != 0 {
		probe := addISOMonths(d, months)
		if d.day > isoDaysInMonth(probe.year, probe.month) {
			panic(r.newError(r.getRangeError(), "Ambiguous date with overflow=reject"))
		}
	}
	date := addISOMonths(d, months)
	date = addISODays(date, days)
	local := new(big.Int).Mul(big.NewInt(date.days()), temporalBigDayNs)
	local.Add(local, big.NewInt(tm.nanos()))
	inter := zone.instantFromLocal(local)
	if inter == nil {
		panic(r.newError(r.getRangeError(), "ZonedDateTime is out of range"))
	}
	timeNs := new(big.Rat)
	add := func(v float64, mul int64) {
		if v != 0 {
			timeNs.Add(timeNs, new(big.Rat).Mul(new(big.Rat).SetFloat64(v*float64(sign)), big.NewRat(mul, 1)))
		}
	}
	add(fields[4], 3600*1e9)
	add(fields[5], 60*1e9)
	add(fields[6], 1e9)
	add(fields[7], 1e6)
	add(fields[8], 1e3)
	add(fields[9], 1)
	if !timeNs.IsInt() {
		panic(r.newError(r.getRangeError(), "Duration is not a whole number of nanoseconds"))
	}
	out := new(big.Int).Add(inter, timeNs.Num())
	if !r.validInstant(out) {
		panic(r.newError(r.getRangeError(), "ZonedDateTime is out of range"))
	}
	return out
}

func (r *Runtime) temporalZonedDateTimeAddSub(call FunctionCall, subtract bool) Value {
	z := r.toTemporalZonedDateTime(call.This)
	d := r.temporalDurationFromValue(call.Argument(0))
	overflow := r.temporalOverflowOption(call.Argument(1))
	sign := int64(1)
	if subtract {
		sign = -1
	}
	ns := r.zonedAddWithOverflow(z.epochNs, z.zone, d.fields(), sign, overflow == "reject")
	return r.newTemporalZonedDateTimeObject(ns, z.zone, z.calendar, r.getTemporalZonedDateTimePrototype())
}

func (r *Runtime) temporalZonedDateTimeAdd(call FunctionCall) Value {
	return r.temporalZonedDateTimeAddSub(call, false)
}

func (r *Runtime) temporalZonedDateTimeSubtract(call FunctionCall) Value {
	return r.temporalZonedDateTimeAddSub(call, true)
}

func (r *Runtime) temporalZonedDateTimeStartOfDay(call FunctionCall) Value {
	z := r.toTemporalZonedDateTime(call.This)
	d, _, _ := z.zone.localDateTime(z.epochNs)
	ns := z.zone.startOfDay(d)
	if ns == nil || !r.validInstant(ns) {
		panic(r.newError(r.getRangeError(), "Start of day is out of range"))
	}
	return r.newTemporalZonedDateTimeObject(ns, z.zone, z.calendar, r.getTemporalZonedDateTimePrototype())
}

func (r *Runtime) temporalZonedDateTimeUntil(call FunctionCall) Value {
	a := r.toTemporalZonedDateTime(call.This)
	b := r.temporalZonedDateTimeFromValue(call.Argument(0), _undefined)
	return r.temporalZonedDateTimeDifference(a, b, call.Argument(1), false)
}

func (r *Runtime) temporalZonedDateTimeSince(call FunctionCall) Value {
	a := r.toTemporalZonedDateTime(call.This)
	b := r.temporalZonedDateTimeFromValue(call.Argument(0), _undefined)
	return r.temporalZonedDateTimeDifference(a, b, call.Argument(1), true)
}

func (r *Runtime) temporalZonedDateTimeDifference(a, b *temporalZonedDateTimeObject, options Value, since bool) Value {
	s := r.temporalGetDifferenceSettings(options, "hour", "nanosecond", 0, 9)
	if since {
		s.mode = negateRoundingMode(s.mode)
	}
	largestIdx := temporalUnitIndex(s.largest)
	smallestIdx := temporalUnitIndex(s.smallest)
	fields := [10]float64{}

	if largestIdx >= 4 {
		diff := new(big.Int).Sub(b.epochNs, a.epochNs)
		unit := new(big.Int).Mul(big.NewInt(temporalUnitNanos(s.smallest)), big.NewInt(s.increment))
		diff = temporalRoundNanos(diff, unit, s.mode)
		temporalTimeFieldsFromNanos(diff, largestIdx, &fields)
	} else {
		aDate, aTime, _ := a.zone.localDateTime(a.epochNs)
		bDate, bTime, _ := b.zone.localDateTime(b.epochNs)
		timeDiff := bTime.nanos() - aTime.nanos()
		endDate := bDate
		if timeDiff < 0 && bDate.days() > aDate.days() {
			timeDiff += 86400e9
			endDate = addISODays(endDate, -1)
		} else if timeDiff > 0 && bDate.days() < aDate.days() {
			timeDiff -= 86400e9
			endDate = addISODays(endDate, 1)
		}
		dateFields := r.temporalDateFieldsFromDays(aDate, endDate.days()-aDate.days(), s.largest)
		copy(fields[:4], dateFields[:4])
		if smallestIdx >= 4 {
			// Recompute the time part exactly, after applying the date part.
			inter := r.zonedAdd(a.epochNs, a.zone, fields, 1)
			timeNs := new(big.Int).Sub(b.epochNs, inter)
			unit := new(big.Int).Mul(big.NewInt(temporalUnitNanos(s.smallest)), big.NewInt(s.increment))
			timeNs = temporalRoundNanos(timeNs, unit, s.mode)
			dayNs := big.NewInt(86400e9)
			carry, rem := new(big.Int).QuoRem(timeNs, dayNs, new(big.Int))
			if carry.Sign() != 0 {
				endpoint := new(big.Int).Add(inter, timeNs)
				if !r.validInstant(endpoint) {
					panic(r.newError(r.getRangeError(), "ZonedDateTime is out of range"))
				}
				for i := 0; i < 2; i++ {
					eWall, eTime, _ := a.zone.localDateTime(endpoint)
					dateFields = r.temporalDateFieldsFromDays(aDate, eWall.days()-aDate.days(), s.largest)
					copy(fields[:4], dateFields[:4])
					inter = r.zonedAdd(a.epochNs, a.zone, fields, 1)
					rem = new(big.Int).Sub(endpoint, inter)
					if rem.Sign() >= 0 && rem.Cmp(dayNs) < 0 {
						_ = eTime
						break
					}
				}
			}
			temporalTimeFieldsFromNanos(rem, 4, &fields)
		} else {
			r.roundZonedDateFields(a, b, &fields, s)
		}
	}
	if since {
		for i := range fields {
			fields[i] = -fields[i]
		}
	}
	r.validateTemporalDuration(fields)
	return r.newTemporalDurationObject(fields, r.getTemporalDurationPrototype())
}

// roundZonedDateFields rounds a zoned difference to a calendar unit (day or
// larger) using zoned day boundaries anchored at the receiver.
func (r *Runtime) roundZonedDateFields(a, b *temporalZonedDateTimeObject, fields *[10]float64, s temporalDiffSettings) {
	unit := s.smallest
	idx := temporalUnitIndex(unit)
	if idx > 3 {
		return
	}
	inc := s.increment
	start := a.epochNs
	diff := new(big.Int).Sub(b.epochNs, start)
	if diff.Sign() == 0 {
		return
	}
	startWall, startTime, _ := a.zone.localDateTime(start)

	targetDateOf := func(k int64) isoDate {
		switch unit {
		case "day":
			return addISODays(startWall, k*inc)
		case "week":
			return addISODays(startWall, k*inc*7)
		case "month":
			return addISOMonths(startWall, k*inc)
		default:
			return addISOMonths(startWall, k*inc*12)
		}
	}
	targetOf := func(k int64) *big.Int {
		d := targetDateOf(k)
		if !d.isValid() {
			return nil
		}
		local := new(big.Int).Mul(big.NewInt(d.days()), temporalBigDayNs)
		local.Add(local, big.NewInt(startTime.nanos()))
		inst := a.zone.instantFromLocal(local)
		if inst == nil || !r.validInstant(inst) {
			return nil
		}
		return inst
	}

	// Estimate the number of increments from the elapsed time.
	approx := big.NewInt(0)
	switch unit {
	case "day":
		approx.SetInt64(86400 * 1e9)
	case "week":
		approx.SetInt64(7 * 86400 * 1e9)
	case "month":
		approx.SetInt64(2629746 * 1e9)
	default:
		approx.SetInt64(31556952 * 1e9)
	}
	approx.Mul(approx, big.NewInt(inc))
	estBig := new(big.Int).Quo(diff, approx)
	est := int64(1 << 20)
	if estBig.IsInt64() {
		est = estBig.Int64()
	} else if estBig.Sign() < 0 {
		est = -est
	}
	margin := est/8 + 16
	if margin < 0 {
		margin = -margin
	}
	lo := est - margin - 1
	hi := est + margin + 1

	// Clamp to the range where the target date is representable.
	daysPerInc := int64(1)
	switch unit {
	case "day":
		daysPerInc = inc
	case "week":
		daysPerInc = 7 * inc
	}
	if idx <= 2 {
		if unit == "month" {
			kMax := ((int64(275760)-startWall.year)*12 + (9 - startWall.month)) / inc
			kMin := ((int64(-271821)-startWall.year)*12 + (4 - startWall.month)) / inc
			if hi > kMax {
				hi = kMax
			}
			if lo < kMin {
				lo = kMin
			}
		} else {
			kMax := (int64(275760) - startWall.year) / inc
			kMin := (int64(-271821) - startWall.year) / inc
			if hi > kMax {
				hi = kMax
			}
			if lo < kMin {
				lo = kMin
			}
		}
	} else {
		kMax := (temporalMaxISODate - startWall.days()) / daysPerInc
		kMin := (temporalMinISODate - startWall.days()) / daysPerInc
		if hi > kMax {
			hi = kMax
		}
		if lo < kMin {
			lo = kMin
		}
	}
	if hi < lo {
		hi = lo
	}
	for targetOf(lo) == nil && lo < hi {
		lo++
	}
	for targetOf(hi) == nil && hi > lo {
		hi--
	}
	for lo < hi {
		mid := lo + (hi-lo+1)/2
		t := targetOf(mid)
		if t != nil && t.Cmp(b.epochNs) <= 0 {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	t := lo
	lower := targetOf(t)
	upper := targetOf(t + 1)
	if lower == nil {
		panic(r.newError(r.getRangeError(), "Resulting date-time is out of range"))
	}
	if upper == nil {
		panic(r.newError(r.getRangeError(), "Resulting date-time is out of range"))
	}
	rem := new(big.Int).Sub(b.epochNs, lower)
	scale := new(big.Int).Sub(upper, lower)
	chosen := t
	if rem.Sign() != 0 {
		if diff.Sign() < 0 {
			q := new(big.Int).Neg(new(big.Int).Add(big.NewInt(t), big.NewInt(1)))
			remNeg := new(big.Int).Sub(scale, rem)
			if !roundMagnitude(s.mode, true, q, remNeg, scale) {
				chosen = t + 1
			}
		} else if roundMagnitude(s.mode, false, big.NewInt(t), rem, scale) {
			chosen = t + 1
		}
	}
	end := targetOf(chosen)
	if end == nil || !r.validInstant(end) {
		panic(r.newError(r.getRangeError(), "Resulting date-time is out of range"))
	}
	endWall, _, _ := a.zone.localDateTime(end)
	dateFields := r.temporalDateFieldsFromDays(startWall, endWall.days()-startWall.days(), s.largest)
	copy(fields[:4], dateFields[:4])
	for i := 4; i < 10; i++ {
		fields[i] = 0
	}
}

func (r *Runtime) temporalZonedDateTimeEquals(call FunctionCall) Value {
	a := r.toTemporalZonedDateTime(call.This)
	b := r.temporalZonedDateTimeFromValue(call.Argument(0), _undefined)
	same := a.epochNs.Cmp(b.epochNs) == 0 && a.zone.id == b.zone.id && a.calendar == b.calendar
	return r.ToValue(same)
}

func (r *Runtime) temporalZonedDateTimeCompare(call FunctionCall) Value {
	a := r.temporalZonedDateTimeFromValue(call.Argument(0), _undefined)
	b := r.temporalZonedDateTimeFromValue(call.Argument(1), _undefined)
	return intToValue(int64(a.epochNs.Cmp(b.epochNs)))
}

func (r *Runtime) temporalZonedDateTimeToString(call FunctionCall) Value {
	z := r.toTemporalZonedDateTime(call.This)
	return r.ToValue(r.formatTemporalZonedDateTime(z, call.Argument(0)))
}

func (r *Runtime) formatTemporalZonedDateTime(z *temporalZonedDateTimeObject, options Value) string {
	calendarName := "auto"
	offsetMode := "auto"
	tzNameMode := "auto"
	_ = r.temporalOptionsObject(options)
	obj, _ := options.(*Object)
	p := temporalStringPrecision{digits: -1, mode: temporalRoundTrunc}
	if obj != nil {
		if v := obj.self.getStr("calendarName", nil); v != nil && !IsUndefined(v) {
			str, _ := temporalToStringOption(r, v)
			switch str {
			case "auto", "always", "never", "critical":
				calendarName = str
			default:
				panic(r.newError(r.getRangeError(), "Invalid calendarName"))
			}
		}
		r.temporalReadFractionalSecondDigits(obj, &p)
		if v := obj.self.getStr("offset", nil); v != nil && !IsUndefined(v) {
			str, _ := temporalToStringOption(r, v)
			switch str {
			case "auto", "never":
				offsetMode = str
			default:
				panic(r.newError(r.getRangeError(), "Invalid offset option"))
			}
		}
		p.mode = r.parseTemporalRoundingMode(obj.self.getStr("roundingMode", nil), temporalRoundTrunc)
		rawUnit := ""
		hasUnit := false
		if v := obj.self.getStr("smallestUnit", nil); v != nil && !IsUndefined(v) {
			rawUnit, _ = temporalToStringOption(r, v)
			hasUnit = true
		}
		if v := obj.self.getStr("timeZoneName", nil); v != nil && !IsUndefined(v) {
			str, _ := temporalToStringOption(r, v)
			switch str {
			case "auto", "never", "critical":
				tzNameMode = str
			default:
				panic(r.newError(r.getRangeError(), "Invalid timeZoneName option"))
			}
		}
		if hasUnit {
			r.temporalApplySmallestUnit(rawUnit, &p)
		}
	}
	d, tm, _ := z.zone.localDateTime(z.epochNs)
	if p.hasUnit {
		d, tm = r.roundISODateTime(d, tm, p.unit, 1, p.mode)
	} else if p.digits >= 0 {
		if p.digits == 0 {
			d, tm = r.roundISODateTime(d, tm, "second", 1, p.mode)
		} else {
			inc := int64(1)
			for i := 0; i < 9-p.digits; i++ {
				inc *= 10
			}
			d, tm = r.roundISODateTime(d, tm, "nanosecond", inc, p.mode)
		}
	}
	local := new(big.Int).Mul(big.NewInt(d.days()), temporalBigDayNs)
	local.Add(local, big.NewInt(tm.nanos()))
	inst := z.zone.instantFromLocal(local)
	if inst == nil || !r.validInstant(inst) {
		panic(r.newError(r.getRangeError(), "ZonedDateTime is out of range"))
	}
	out := formatISODate(d) + "T" + formatISOTimePrecision(tm, p)
	if offsetMode != "never" {
		out += formatOffsetNano(z.zone.offsetAt(inst))
	}
	if tzNameMode != "never" {
		if tzNameMode == "critical" {
			out += "[!" + z.zone.id + "]"
		} else {
			out += "[" + z.zone.id + "]"
		}
	}
	switch calendarName {
	case "always":
		out += "[u-ca=" + z.calendar + "]"
	case "critical":
		out += "[!u-ca=" + z.calendar + "]"
	}
	return out
}

func (r *Runtime) temporalZonedDateTimeToJSON(call FunctionCall) Value {
	z := r.toTemporalZonedDateTime(call.This)
	return r.ToValue(r.formatTemporalZonedDateTime(z, _undefined))
}

func (r *Runtime) temporalZonedDateTimeToLocaleString(call FunctionCall) Value {
	z := r.toTemporalZonedDateTime(call.This)
	return r.ToValue(r.formatTemporalZonedDateTime(z, _undefined))
}

func (r *Runtime) temporalZonedDateTimeValueOf(call FunctionCall) Value {
	panic(r.NewTypeError("Temporal.ZonedDateTime objects cannot be converted to primitives"))
}

func (r *Runtime) temporalZonedDateTimeToInstant(call FunctionCall) Value {
	z := r.toTemporalZonedDateTime(call.This)
	return r.newTemporalInstantObject(z.epochNs, r.getTemporalInstantPrototype())
}

func (r *Runtime) temporalZonedDateTimeToPlainDate(call FunctionCall) Value {
	z := r.toTemporalZonedDateTime(call.This)
	d, _, _ := z.zone.localDateTime(z.epochNs)
	return r.newTemporalPlainDateObject(d, z.calendar, r.getTemporalPlainDatePrototype())
}

func (r *Runtime) temporalZonedDateTimeToPlainTime(call FunctionCall) Value {
	z := r.toTemporalZonedDateTime(call.This)
	_, t, _ := z.zone.localDateTime(z.epochNs)
	return r.newTemporalPlainTimeObject(t.hour, t.minute, t.second, t.millisecond, t.microsecond, t.nanosecond, r.getTemporalPlainTimePrototype())
}

func (r *Runtime) temporalZonedDateTimeToPlainDateTime(call FunctionCall) Value {
	z := r.toTemporalZonedDateTime(call.This)
	d, t, _ := z.zone.localDateTime(z.epochNs)
	return r.newTemporalPlainDateTimeObject(d, t, z.calendar, r.getTemporalPlainDateTimePrototype())
}

func (r *Runtime) temporalZonedDateTimeGetISOFields(call FunctionCall) Value {
	z := r.toTemporalZonedDateTime(call.This)
	d, t, _ := z.zone.localDateTime(z.epochNs)
	o := r.NewObject()
	o.Set("isoYear", r.ToValue(d.year))
	o.Set("isoMonth", r.ToValue(d.month))
	o.Set("isoDay", r.ToValue(d.day))
	o.Set("isoHour", r.ToValue(t.hour))
	o.Set("isoMinute", r.ToValue(t.minute))
	o.Set("isoSecond", r.ToValue(t.second))
	o.Set("isoMillisecond", r.ToValue(t.millisecond))
	o.Set("isoMicrosecond", r.ToValue(t.microsecond))
	o.Set("isoNanosecond", r.ToValue(t.nanosecond))
	o.Set("offset", r.ToValue(formatOffsetNano(z.zone.offsetAt(z.epochNs))))
	o.Set("timeZone", r.ToValue(z.zone.id))
	o.Set("calendar", r.ToValue(z.calendar))
	return o
}

func (r *Runtime) temporalZonedDateTimeGetTimeZoneTransition(call FunctionCall) Value {
	z := r.toTemporalZonedDateTime(call.This)
	arg := call.Argument(0)
	dir := int64(0)
	if s, ok := arg.(String); ok {
		switch s.String() {
		case "next":
			dir = 1
		case "previous":
			dir = -1
		default:
			panic(r.newError(r.getRangeError(), "Invalid direction"))
		}
	} else if o, ok := arg.(*Object); ok {
		if other, ok := o.self.(*temporalZonedDateTimeObject); ok {
			switch other.epochNs.Cmp(z.epochNs) {
			case 1:
				dir = 1
			case -1:
				dir = -1
			default:
				return _null
			}
		} else {
			v := o.self.getStr("direction", nil)
			if v == nil || IsUndefined(v) {
				panic(r.newError(r.getRangeError(), "direction is required"))
			}
			str, _ := temporalToStringOption(r, v)
			switch str {
			case "next":
				dir = 1
			case "previous":
				dir = -1
			default:
				panic(r.newError(r.getRangeError(), "Invalid direction"))
			}
		}
	} else {
		panic(r.NewTypeError("Invalid direction"))
	}
	tr := z.zone.findTransition(z.epochNs, dir)
	if tr == nil {
		return _null
	}
	return r.newTemporalZonedDateTimeObject(tr, z.zone, z.calendar, r.getTemporalZonedDateTimePrototype())
}

func (r *Runtime) temporalZonedDateTimeRound(call FunctionCall) Value {
	z := r.toTemporalZonedDateTime(call.This)
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
		d, _, _ := z.zone.localDateTime(z.epochNs)
		start := z.zone.startOfDay(d)
		end := z.zone.startOfDay(addISODays(d, 1))
		if start == nil || end == nil || !r.validInstant(start) || !r.validInstant(end) {
			panic(r.newError(r.getRangeError(), "Start of day is out of range"))
		}
		dayLen := new(big.Int).Sub(end, start)
		elapsed := new(big.Int).Sub(z.epochNs, start)
		q, rem := new(big.Int).QuoRem(elapsed, dayLen, new(big.Int))
		neg := elapsed.Sign() < 0
		absRem := new(big.Int).Abs(rem)
		if absRem.Sign() != 0 && roundMagnitude(mode, neg, new(big.Int).Abs(q), absRem, dayLen) {
			if neg {
				q.Sub(q, big.NewInt(1))
			} else {
				q.Add(q, big.NewInt(1))
			}
		}
		out := new(big.Int).Add(start, new(big.Int).Mul(q, dayLen))
		if !r.validInstant(out) {
			panic(r.newError(r.getRangeError(), "ZonedDateTime is out of range"))
		}
		return r.newTemporalZonedDateTimeObject(out, z.zone, z.calendar, r.getTemporalZonedDateTimePrototype())
	}
	if err := validateTimeRoundingIncrement(smallest, increment); err != nil {
		panic(r.newError(r.getRangeError(), err.Error()))
	}
	d, t, _ := z.zone.localDateTime(z.epochNs)
	nd, nt := r.roundISODateTime(d, t, smallest, increment, mode)
	local := new(big.Int).Mul(big.NewInt(nd.days()), temporalBigDayNs)
	local.Add(local, big.NewInt(nt.nanos()))
	ns := z.zone.instantFromLocal(local)
	if ns == nil || !r.validInstant(ns) {
		panic(r.newError(r.getRangeError(), "ZonedDateTime is out of range"))
	}
	return r.newTemporalZonedDateTimeObject(ns, z.zone, z.calendar, r.getTemporalZonedDateTimePrototype())
}

var (
	_ = math.Trunc
	_ = strconv.Itoa
)
