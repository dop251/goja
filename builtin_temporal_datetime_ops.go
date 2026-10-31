package goja

import (
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/dop251/goja/unistring"
)

// temporalOptionsObject implements the GetOptionsObject conversion shared by
// the Temporal methods that take an options bag.
func (r *Runtime) temporalOptionsObject(v Value) *Object {
	if v == nil || IsUndefined(v) {
		return nil
	}
	o, ok := v.(*Object)
	if !ok {
		panic(r.NewTypeError("options must be an object"))
	}
	if o == nil {
		return nil
	}
	return o
}

// temporalGetRoundingIncrementOption implements ToTemporalRoundingIncrement.
func (r *Runtime) temporalGetRoundingIncrementOption(obj *Object) int64 {
	if obj == nil {
		return 1
	}
	v := obj.self.getStr("roundingIncrement", nil)
	if v == nil || IsUndefined(v) {
		return 1
	}
	f := v.ToNumber().ToFloat()
	if math.IsNaN(f) || math.IsInf(f, 0) {
		panic(r.newError(r.getRangeError(), "Invalid roundingIncrement"))
	}
	n := int64(math.Trunc(f))
	if n < 1 || n > 1e9 {
		panic(r.newError(r.getRangeError(), "Invalid roundingIncrement"))
	}
	return n
}

type temporalDiffSettings struct {
	largest   string
	smallest  string
	increment int64
	mode      temporalRoundingMode
}

// temporalGetDifferenceSettings implements GetDifferenceSettings, reading the
// options in the observable order (largestUnit, roundingIncrement,
// roundingMode, smallestUnit). minUnit/maxUnit bound the accepted units.
func (r *Runtime) temporalGetDifferenceSettings(options Value, largestDefault, smallestFallback string, minUnit, maxUnit int) temporalDiffSettings {
	obj := r.temporalOptionsObject(options)
	var s temporalDiffSettings
	if obj != nil {
		if v := obj.self.getStr("largestUnit", nil); v != nil && !IsUndefined(v) {
			str, _ := temporalToStringOption(r, v)
			if str != "auto" {
				u, ok := normalizeTemporalUnit(str)
				if !ok {
					panic(r.newError(r.getRangeError(), "Invalid largestUnit"))
				}
				s.largest = u
			}
		}
	}
	s.increment = r.temporalGetRoundingIncrementOption(obj)
	s.mode = temporalRoundTrunc
	if obj != nil {
		s.mode = r.parseTemporalRoundingMode(obj.self.getStr("roundingMode", nil), temporalRoundTrunc)
	}
	s.smallest = smallestFallback
	if obj != nil {
		if v := obj.self.getStr("smallestUnit", nil); v != nil && !IsUndefined(v) {
			str, _ := temporalToStringOption(r, v)
			u, ok := normalizeTemporalUnit(str)
			if !ok {
				panic(r.newError(r.getRangeError(), "Invalid smallestUnit"))
			}
			s.smallest = u
		}
	}
	if s.largest == "" {
		if temporalUnitIndex(largestDefault) < temporalUnitIndex(s.smallest) {
			s.largest = largestDefault
		} else {
			s.largest = s.smallest
		}
	}
	if temporalUnitIndex(s.smallest) < temporalUnitIndex(s.largest) {
		panic(r.newError(r.getRangeError(), "smallestUnit is larger than largestUnit"))
	}
	if temporalUnitIndex(s.largest) < minUnit || temporalUnitIndex(s.largest) > maxUnit ||
		temporalUnitIndex(s.smallest) < minUnit || temporalUnitIndex(s.smallest) > maxUnit {
		panic(r.newError(r.getRangeError(), "Unit is not allowed here"))
	}
	if temporalUnitIndex(s.smallest) >= 4 {
		if err := validateTimeRoundingIncrement(s.smallest, s.increment); err != nil {
			panic(r.newError(r.getRangeError(), err.Error()))
		}
	}
	return s
}

func negateRoundingMode(m temporalRoundingMode) temporalRoundingMode {
	switch m {
	case temporalRoundCeil:
		return temporalRoundFloor
	case temporalRoundFloor:
		return temporalRoundCeil
	case temporalRoundHalfCeil:
		return temporalRoundHalfFloor
	case temporalRoundHalfFloor:
		return temporalRoundHalfCeil
	}
	return m
}

// isoDateTimeNanos returns the exact number of nanoseconds between the epoch
// and the given date-time.
func isoDateTimeNanos(d isoDate, t isoTime) *big.Int {
	return new(big.Int).Add(new(big.Int).Mul(big.NewInt(d.days()), big.NewInt(86400e9)), big.NewInt(t.nanos()))
}

// temporalRoundNanos rounds a signed nanosecond count to a multiple of unit.
func temporalRoundNanos(ns, unit *big.Int, mode temporalRoundingMode) *big.Int {
	q, rem := new(big.Int).QuoRem(ns, unit, new(big.Int))
	negative := ns.Sign() < 0
	absRem := new(big.Int).Abs(rem)
	absQ := new(big.Int).Abs(q)
	if absRem.Sign() != 0 && roundMagnitude(mode, negative, absQ, absRem, unit) {
		if negative {
			q.Sub(q, big.NewInt(1))
		} else {
			q.Add(q, big.NewInt(1))
		}
	}
	return new(big.Int).Mul(q, unit)
}

// temporalTimeFieldsFromNanos decomposes a signed nanosecond count into
// duration fields starting at largestUnitIndex (which must be a time unit).
func temporalTimeFieldsFromNanos(ns *big.Int, largestUnitIndex int, out *[10]float64) {
	rem := new(big.Int).Set(ns)
	for i := largestUnitIndex; i <= 9; i++ {
		u := big.NewInt(temporalUnitNanosIndex(i))
		q, r := new(big.Int).QuoRem(rem, u, new(big.Int))
		f, _ := new(big.Float).SetInt(q).Float64()
		out[i] = f
		rem = r
	}
}

// temporalDateFieldsFromDays expresses a day count as calendar units starting
// at largestUnit, relative to start.
func (r *Runtime) temporalDateFieldsFromDays(start isoDate, days int64, largest string) [10]float64 {
	fields := [10]float64{}
	if days == 0 {
		return fields
	}
	sign := int64(1)
	if days < 0 {
		sign = -1
		days = -days
	}
	switch largest {
	case "day":
		fields[3] = float64(sign * days)
	case "week":
		fields[2] = float64(sign * (days / 7))
		fields[3] = float64(sign * (days % 7))
	default:
		end := addISODays(start, sign*days)
		if sign > 0 {
			total := (end.year-start.year)*12 + (end.month - start.month)
			for total > 0 && addISOMonths(start, total).days() > end.days() {
				total--
			}
			mid := addISOMonths(start, total)
			rem := end.days() - mid.days()
			y := total / 12
			m := total % 12
			if largest == "month" {
				y, m = 0, total
			}
			fields[0], fields[1], fields[3] = float64(y), float64(m), float64(rem)
		} else {
			total := (start.year-end.year)*12 + (start.month - end.month)
			for total > 0 && addISOMonths(start, -total).days() < end.days() {
				total--
			}
			mid := addISOMonths(start, -total)
			rem := end.days() - mid.days()
			y := total / 12
			m := total % 12
			if largest == "month" {
				y, m = 0, total
			}
			fields[0], fields[1], fields[3] = float64(-y), float64(-m), float64(rem)
		}
	}
	return fields
}

// temporalDiffISODateTime implements DifferenceISODateTime for the ISO
// calendar.
func (r *Runtime) temporalDiffISODateTime(sd isoDate, st isoTime, ed isoDate, et isoTime, s temporalDiffSettings) [10]float64 {
	fields := [10]float64{}
	largestIdx := temporalUnitIndex(s.largest)
	smallestIdx := temporalUnitIndex(s.smallest)
	if largestIdx >= 4 {
		total := new(big.Int).Mul(big.NewInt(ed.days()-sd.days()), big.NewInt(86400e9))
		total.Add(total, big.NewInt(et.nanos()-st.nanos()))
		unit := new(big.Int).Mul(big.NewInt(temporalUnitNanos(s.smallest)), big.NewInt(s.increment))
		total = temporalRoundNanos(total, unit, s.mode)
		temporalTimeFieldsFromNanos(total, largestIdx, &fields)
		return fields
	}

	days := ed.days() - sd.days()
	timeNs := et.nanos() - st.nanos()
	if days > 0 && timeNs < 0 {
		days--
		timeNs += 86400e9
	} else if days < 0 && timeNs > 0 {
		days++
		timeNs -= 86400e9
	}

	if smallestIdx >= 4 {
		unit := new(big.Int).Mul(big.NewInt(temporalUnitNanos(s.smallest)), big.NewInt(s.increment))
		rounded := temporalRoundNanos(big.NewInt(timeNs), unit, s.mode)
		dayNs := big.NewInt(86400e9)
		carry, rem := new(big.Int).QuoRem(rounded, dayNs, new(big.Int))
		days += carry.Int64()
		timeNs = rem.Int64()
	} else {
		return r.temporalRoundDateTimeSpan(sd, st, ed, et, s)
	}

	dateFields := r.temporalDateFieldsFromDays(sd, days, s.largest)
	copy(fields[:4], dateFields[:4])
	temporalTimeFieldsFromNanos(big.NewInt(timeNs), 4, &fields)
	return fields
}

// temporalRoundDateTimeSpan rounds a whole date-time span to a calendar unit
// (day, week, month or year), using the start as the anchor.
func (r *Runtime) temporalRoundDateTimeSpan(sd isoDate, st isoTime, ed isoDate, et isoTime, s temporalDiffSettings) [10]float64 {
	startNanos := isoDateTimeNanos(sd, st)
	endNanos := isoDateTimeNanos(ed, et)
	diff := new(big.Int).Sub(endNanos, startNanos)
	if diff.Sign() == 0 {
		return [10]float64{}
	}

	unit := s.smallest
	inc := s.increment

	targetDateOf := func(k int64) isoDate {
		switch unit {
		case "day":
			return addISODays(sd, k*inc)
		case "week":
			return addISODays(sd, k*inc*7)
		case "month":
			return addISOMonths(sd, k*inc)
		default:
			return addISOMonths(sd, k*inc*12)
		}
	}
	target := func(k int64) *big.Int {
		return isoDateTimeNanos(targetDateOf(k), st)
	}

	approx := new(big.Int)
	switch unit {
	case "day":
		approx.SetInt64(86400e9)
	case "week":
		approx.SetInt64(7 * 86400e9)
	case "month":
		approx.SetInt64(2629746 * 1e9)
	case "year":
		approx.SetInt64(31556952 * 1e9)
	}
	approx.Mul(approx, big.NewInt(inc))

	estimate := new(big.Int).Quo(diff, approx).Int64()
	margin := estimate/8 + 16
	if margin < 0 {
		margin = -margin
	}
	lo := estimate - margin - 1
	hi := estimate + margin + 1
	for lo < hi {
		mid := lo + (hi-lo+1)/2
		if target(mid).Cmp(endNanos) <= 0 {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	t := lo
	if s.largest != "day" {
		if !targetDateOf(t).isValidDateTime(st) || !targetDateOf(t+1).isValidDateTime(st) {
			panic(r.newError(r.getRangeError(), "Resulting date-time is out of range"))
		}
	}
	lower := target(t)
	upper := target(t + 1)
	rem := new(big.Int).Sub(endNanos, lower)
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

	if s.largest == "day" && unit == "day" {
		fields := [10]float64{}
		fields[3] = float64(chosen * inc)
		return fields
	}
	targetDate := targetDateOf(chosen)
	if !targetDate.isValidDateTime(st) {
		panic(r.newError(r.getRangeError(), "Resulting date-time is out of range"))
	}
	fields := r.temporalDateFieldsFromDays(sd, targetDate.days()-sd.days(), s.largest)
	if s.smallest == "week" && temporalUnitIndex(s.largest) < 2 {
		if fields[3] != 0 {
			fields[2] = float64(int64(fields[3]) / 7)
			fields[3] = 0
		}
	}
	return fields
}

// roundISODateTime rounds the time-of-day to a multiple of unit and carries
// any overflow into the date. Rounding is always relative to the start of the
// day, not the epoch.
func (r *Runtime) roundISODateTime(d isoDate, t isoTime, unit string, increment int64, mode temporalRoundingMode) (isoDate, isoTime) {
	dayNanos := int64(86400e9)
	timeNs := t.nanos()
	if unit == "day" {
		up := false
		if timeNs != 0 {
			switch mode {
			case temporalRoundTrunc, temporalRoundFloor:
				up = false
			case temporalRoundCeil, temporalRoundExpand:
				up = true
			default:
				q := d.days()
				if timeNs*2 > dayNanos {
					up = true
				} else if timeNs*2 < dayNanos {
					up = false
				} else {
					switch mode {
					case temporalRoundHalfCeil:
						up = true
					case temporalRoundHalfFloor:
						up = false
					case temporalRoundHalfExpand:
						up = q >= 0
					case temporalRoundHalfTrunc:
						up = q < 0
					case temporalRoundHalfEven:
						up = q%2 != 0
					}
				}
			}
		}
		if up {
			d = addISODays(d, 1)
		}
		t = isoTime{}
	} else {
		scale := temporalUnitNanos(unit) * increment
		q := timeNs / scale
		rem := timeNs % scale
		if rem != 0 && roundMagnitude(mode, false, big.NewInt(q), big.NewInt(rem), big.NewInt(scale)) {
			q++
		}
		ns := q * scale
		if ns >= dayNanos {
			d = addISODays(d, ns/dayNanos)
			ns %= dayNanos
		}
		t = isoTimeFromNanos(ns)
	}
	if !d.isValidDateTime(t) {
		panic(r.newError(r.getRangeError(), "Resulting date-time is out of range"))
	}
	return d, t
}

type temporalStringPrecision struct {
	digits  int
	unit    string
	mode    temporalRoundingMode
	hasUnit bool
}

// temporalGetStringPrecision implements ToSecondsStringPrecision, reading the
// options in the observable order.
func (r *Runtime) temporalGetStringPrecision(options Value, between func()) temporalStringPrecision {
	p := temporalStringPrecision{digits: -1, mode: temporalRoundTrunc}
	if options == nil || IsUndefined(options) {
		if between != nil {
			between()
		}
		return p
	}
	obj := r.temporalOptionsObject(options)
	if obj == nil {
		if between != nil {
			between()
		}
		return p
	}
	r.temporalReadFractionalSecondDigits(obj, &p)
	p.mode = r.parseTemporalRoundingMode(obj.self.getStr("roundingMode", nil), temporalRoundTrunc)
	rawUnit := ""
	hasUnit := false
	if v := obj.self.getStr("smallestUnit", nil); v != nil && !IsUndefined(v) {
		rawUnit, _ = temporalToStringOption(r, v)
		hasUnit = true
	}
	if between != nil {
		between()
	}
	if hasUnit {
		r.temporalApplySmallestUnit(rawUnit, &p)
	}
	return p
}

func (r *Runtime) temporalReadFractionalSecondDigits(obj *Object, p *temporalStringPrecision) {
	v := obj.self.getStr("fractionalSecondDigits", nil)
	if v == nil || IsUndefined(v) {
		return
	}
	switch v.(type) {
	case valueInt, valueFloat:
		f := v.ToNumber().ToFloat()
		if math.IsNaN(f) {
			panic(r.newError(r.getRangeError(), "Invalid fractionalSecondDigits"))
		}
		f = math.Floor(f)
		if f < 0 || f > 9 {
			panic(r.newError(r.getRangeError(), "Invalid fractionalSecondDigits"))
		}
		p.digits = int(f)
	default:
		s, _ := temporalToStringOption(r, v)
		if s != "auto" {
			panic(r.newError(r.getRangeError(), "Invalid fractionalSecondDigits"))
		}
	}
}

func (r *Runtime) temporalApplySmallestUnit(raw string, p *temporalStringPrecision) {
	u, ok := normalizeTemporalUnit(raw)
	if !ok {
		panic(r.newError(r.getRangeError(), "Invalid smallestUnit"))
	}
	switch u {
	case "minute":
		p.unit, p.digits = "minute", 0
	case "second":
		p.unit, p.digits = "second", 0
	case "millisecond":
		p.unit, p.digits = "millisecond", 3
	case "microsecond":
		p.unit, p.digits = "microsecond", 6
	case "nanosecond":
		p.unit, p.digits = "nanosecond", 9
	default:
		panic(r.newError(r.getRangeError(), "Invalid smallestUnit"))
	}
	p.hasUnit = true
}

// formatISOTimeFields formats a time to the requested number of fractional
// second digits (-1 means "auto").
func formatISOTimeFields(t isoTime, digits int) string {
	out := fmt.Sprintf("%02d:%02d:%02d", t.hour, t.minute, t.second)
	ns := t.nanos() % 1e9
	if digits < 0 {
		frac := strings.TrimRight(fmt.Sprintf("%09d", ns), "0")
		if frac != "" {
			out += "." + frac
		}
	} else if digits > 0 {
		div := int64(1)
		for i := 0; i < 9-digits; i++ {
			div *= 10
		}
		out += fmt.Sprintf(".%0*d", digits, ns/div)
	}
	return out
}

// formatISOTimePrecision formats a time using the resolved string precision.
func formatISOTimePrecision(t isoTime, p temporalStringPrecision) string {
	if p.hasUnit && p.unit == "minute" {
		return fmt.Sprintf("%02d:%02d", t.hour, t.minute)
	}
	return formatISOTimeFields(t, p.digits)
}

// temporalDateTimeFieldsFromObject reads the shared date-time field list in
// the observable order used by from/with.
type temporalDateTimeFieldsResult struct {
	date       isoDate
	time       isoTime
	calendar   string
	haveYear   bool
	haveMonth  bool
	haveDay    bool
	provided   bool
	monthSet   bool
	monthCode  string
	hasMonthCd bool
}

func (r *Runtime) temporalReadDateTimeFields(obj *Object, base temporalDateTimeFieldsResult, readTimeZone, rejectCalendar bool) temporalDateTimeFieldsResult {
	out := base
	out.calendar = base.calendar
	if v := obj.self.getStr("calendar", nil); v != nil && !IsUndefined(v) {
		if rejectCalendar {
			panic(r.NewTypeError("calendar is not allowed here"))
		}
		out.calendar = r.temporalCalendarFromValue(v)
	}
	if readTimeZone {
		if v := obj.self.getStr("timeZone", nil); v != nil && !IsUndefined(v) {
			panic(r.NewTypeError("timeZone is not allowed here"))
		}
	}
	for _, f := range temporalDateTimeFields {
		if f.name == "calendar" || f.name == "timeZone" {
			continue
		}
		fv := obj.self.getStr(unistring.NewFromString(f.name), nil)
		if fv == nil || IsUndefined(fv) {
			continue
		}
		out.provided = true
		if f.name == "monthCode" {
			s := r.temporalRequireString(fv)
			if !wellFormedMonthCode(s) {
				panic(r.newError(r.getRangeError(), "Invalid monthCode"))
			}
			out.monthCode = s
			out.hasMonthCd = true
			continue
		}
		var v int64
		if f.date && (f.index == 1 || f.index == 2) {
			v = r.temporalPositiveIntegerValue(fv)
		} else {
			v = r.temporalTimeFieldValue(fv)
		}
		if f.date {
			switch f.index {
			case 0:
				out.date.year = v
				out.haveYear = true
			case 1:
				if !out.monthSet && !out.hasMonthCd {
					out.date.month = v
					out.monthSet = true
					out.haveMonth = true
				}
			case 2:
				out.date.day = v
				out.haveDay = true
			}
		} else {
			switch f.index {
			case 0:
				out.time.hour = v
			case 1:
				out.time.minute = v
			case 2:
				out.time.second = v
			case 3:
				out.time.millisecond = v
			case 4:
				out.time.microsecond = v
			case 5:
				out.time.nanosecond = v
			}
		}
	}
	return out
}

// normalizeBasicDate expands a basic-format ISO date (YYYYMMDD or
// YYYYYYMMDD, optionally signed) to the extended form.
func normalizeBasicDate(s string) (string, bool) {
	sign := ""
	body := s
	if len(body) > 0 && (body[0] == '+' || body[0] == '-') {
		sign, body = body[:1], body[1:]
	}
	if !isDigits(body) {
		return "", false
	}
	switch len(body) {
	case 8:
		return sign + body[:4] + "-" + body[4:6] + "-" + body[6:8], true
	case 10:
		return sign + body[:6] + "-" + body[6:8] + "-" + body[8:10], true
	}
	return "", false
}

// ambiguousISOYearMonthOrMonthDay reports whether a bare string could also be
// a PlainYearMonth or PlainMonthDay, in which case a time requires a T prefix.
func ambiguousISOYearMonthOrMonthDay(s string) bool {
	allDigits := func(v string) (int64, bool) {
		if !isDigits(v) {
			return 0, false
		}
		n := int64(0)
		for i := 0; i < len(v); i++ {
			n = n*10 + int64(v[i]-'0')
		}
		return n, true
	}
	if len(s) == 7 && isDigits(s[:4]) && s[4] == '-' {
		if m, ok := allDigits(s[5:]); ok && m >= 1 && m <= 12 {
			return true
		}
	}
	if len(s) == 6 && isDigits(s) {
		if m, ok := allDigits(s[4:]); ok && m >= 1 && m <= 12 {
			return true
		}
	}
	checkMonthDay := func(ms, ds string) bool {
		m, ok1 := allDigits(ms)
		d, ok2 := allDigits(ds)
		if !ok1 || !ok2 || m < 1 || m > 12 || d < 1 {
			return false
		}
		return d <= isoDaysInMonth(2000, m)
	}
	if len(s) == 4 && isDigits(s) {
		return checkMonthDay(s[:2], s[2:])
	}
	if len(s) == 5 && isDigits(s[:2]) && s[2] == '-' && isDigits(s[3:]) {
		return checkMonthDay(s[:2], s[3:])
	}
	return false
}

// temporalRequireString implements ToPrimitiveAndRequireString.
func (r *Runtime) temporalRequireString(v Value) string {
	if s, ok := v.(String); ok {
		return s.String()
	}
	if o, ok := v.(*Object); ok {
		if s, ok := o.toPrimitiveString().(String); ok {
			return s.String()
		}
	}
	panic(r.NewTypeError("value must be a string"))
}

// wellFormedMonthCode checks the month code syntax (Mnn or MnnL).
func wellFormedMonthCode(s string) bool {
	if len(s) != 3 && len(s) != 4 {
		return false
	}
	if s[0] != 'M' || s[1] < '0' || s[1] > '9' || s[2] < '0' || s[2] > '9' {
		return false
	}
	if len(s) == 4 && s[3] != 'L' {
		return false
	}
	return true
}

// temporalPositiveIntegerValue implements ToPositiveIntegerWithTruncation.
func (r *Runtime) temporalPositiveIntegerValue(v Value) int64 {
	if _, ok := v.(*Symbol); ok {
		panic(r.NewTypeError("Cannot convert a Symbol to a number"))
	}
	f := v.ToNumber().ToFloat()
	if math.IsNaN(f) || math.IsInf(f, 0) {
		panic(r.newError(r.getRangeError(), "Invalid value"))
	}
	f = math.Trunc(f)
	if f < 0 {
		panic(r.newError(r.getRangeError(), "Invalid value"))
	}
	return int64(f)
}

func isoMonthFromMonthCode(s string) (int64, bool) {
	if len(s) != 3 || s[0] != 'M' {
		return 0, false
	}
	n := int64(0)
	for i := 1; i < 3; i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, false
		}
		n = n*10 + int64(s[i]-'0')
	}
	if n < 1 || n > 12 {
		return 0, false
	}
	return n, true
}

// zonedFieldsResult is the result of reading a ZonedDateTime field bag.
type zonedFieldsResult struct {
	temporalDateTimeFieldsResult
	tzValue     Value
	hasTZ       bool
	hasCalendar bool
	offset      *big.Int
	offsetText  string
	hasOffset   bool
}

// temporalReadZonedDateTimeFields reads the shared ZonedDateTime field list in
// the observable order. timeZoneFirst mirrors the order used by `with`.
func (r *Runtime) temporalReadZonedDateTimeFields(obj *Object, base temporalDateTimeFieldsResult, timeZoneFirst bool) zonedFieldsResult {
	out := zonedFieldsResult{temporalDateTimeFieldsResult: base}
	if v := obj.self.getStr("calendar", nil); v != nil && !IsUndefined(v) {
		out.provided = true
		out.hasCalendar = true
		out.calendar = r.temporalCalendarFromValue(v)
	}
	readTZ := func() {
		if v := obj.self.getStr("timeZone", nil); v != nil && !IsUndefined(v) {
			out.tzValue = v
			out.hasTZ = true
		}
	}
	if timeZoneFirst {
		readTZ()
	}
	fields := []struct {
		name  string
		date  bool
		index int
	}{
		{"day", true, 2},
		{"hour", false, 0},
		{"microsecond", false, 4},
		{"millisecond", false, 3},
		{"minute", false, 1},
		{"month", true, 1},
		{"monthCode", true, 3},
		{"nanosecond", false, 5},
	}
	apply := func(name string, date bool, index int, fv Value) {
		out.provided = true
		if name == "monthCode" {
			s := r.temporalRequireString(fv)
			if !wellFormedMonthCode(s) {
				panic(r.newError(r.getRangeError(), "Invalid monthCode"))
			}
			out.monthCode = s
			out.hasMonthCd = true
			return
		}
		var v int64
		if date && (index == 1 || index == 2) {
			v = r.temporalPositiveIntegerValue(fv)
		} else {
			v = r.temporalTimeFieldValue(fv)
		}
		if date {
			switch index {
			case 1:
				if !out.monthSet && !out.hasMonthCd {
					out.date.month = v
					out.monthSet = true
					out.haveMonth = true
				}
			case 2:
				out.date.day = v
				out.haveDay = true
			}
		} else {
			switch index {
			case 0:
				out.time.hour = v
			case 1:
				out.time.minute = v
			case 2:
				out.time.second = v
			case 3:
				out.time.millisecond = v
			case 4:
				out.time.microsecond = v
			case 5:
				out.time.nanosecond = v
			}
		}
	}
	for _, f := range fields {
		fv := obj.self.getStr(unistring.NewFromString(f.name), nil)
		if fv == nil || IsUndefined(fv) {
			continue
		}
		apply(f.name, f.date, f.index, fv)
	}
	if v := obj.self.getStr("offset", nil); v != nil && !IsUndefined(v) {
		out.provided = true
		str := r.temporalRequireString(v)
		off, ok := parseTemporalOffsetString(str)
		if !ok {
			panic(r.newError(r.getRangeError(), "Invalid offset"))
		}
		out.offset = off
		out.offsetText = str
		out.hasOffset = true
	}
	if v := obj.self.getStr("second", nil); v != nil && !IsUndefined(v) {
		apply("second", false, 2, v)
	}
	if !timeZoneFirst {
		readTZ()
	}
	if v := obj.self.getStr("year", nil); v != nil && !IsUndefined(v) {
		out.provided = true
		out.date.year = r.temporalTimeFieldValue(v)
		out.haveYear = true
	}
	return out
}

func (r *Runtime) temporalGetDisambiguationOption(obj *Object) string {
	if obj == nil {
		return "compatible"
	}
	v := obj.self.getStr("disambiguation", nil)
	if v == nil || IsUndefined(v) {
		return "compatible"
	}
	str, _ := temporalToStringOption(r, v)
	switch str {
	case "compatible", "earlier", "later", "reject":
		return str
	}
	panic(r.newError(r.getRangeError(), "Invalid disambiguation option"))
}

func (r *Runtime) temporalGetOffsetOption(obj *Object, def string) string {
	if obj == nil {
		return def
	}
	v := obj.self.getStr("offset", nil)
	if v == nil || IsUndefined(v) {
		return def
	}
	str, _ := temporalToStringOption(r, v)
	switch str {
	case "use", "prefer", "ignore", "reject":
		return str
	}
	panic(r.newError(r.getRangeError(), "Invalid offset option"))
}

// zonedResolve resolves a local wall-clock time and optional offset to an
// instant, applying the disambiguation and offset options.
func (r *Runtime) zonedResolve(zone *temporalZone, local *big.Int, offset *big.Int, hasOffset bool, disambiguation, offsetOpt string) *big.Int {
	if hasOffset && offsetOpt != "ignore" {
		exact := new(big.Int).Sub(local, offset)
		if offsetOpt == "use" {
			return exact
		}
		if zone.offsetAt(exact).Cmp(offset) == 0 {
			return exact
		}
		if offsetOpt == "reject" {
			panic(r.newError(r.getRangeError(), "Offset does not match time zone"))
		}
		// prefer: fall back to the wall-clock interpretation.
	}
	if offsetOpt != "ignore" && !r.validInstant(local) {
		panic(r.newError(r.getRangeError(), "Wall-clock time is out of range"))
	}
	inst := zone.instantFromLocalDisambiguation(local, disambiguation)
	if inst == nil {
		panic(r.newError(r.getRangeError(), "Wall-clock time does not exist in time zone"))
	}
	return inst
}
