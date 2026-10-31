package goja

import (
	"math/big"
	"time"
)

// intlTemporalToMillis converts a Temporal object to epoch milliseconds.
func (r *Runtime) intlTemporalToMillis(v Value) (float64, bool) {
	o, ok := v.(*Object)
	if !ok {
		return 0, false
	}
	ms := func(d isoDate, t isoTime) float64 {
		return float64(time.Date(int(d.year), time.Month(d.month), int(d.day),
			int(t.hour), int(t.minute), int(t.second),
			int(t.millisecond)*1e6+int(t.microsecond)*1e3+int(t.nanosecond), time.UTC).UnixNano()) / 1e6
	}
	switch o := o.self.(type) {
	case *temporalPlainDateObject:
		return ms(o.date, isoTime{}), true
	case *temporalPlainDateTimeObject:
		return ms(o.date, o.time), true
	case *temporalPlainTimeObject:
		return ms(isoDate{year: 1970, month: 1, day: 1}, isoTime{
			hour:        o.hour,
			minute:      o.minute,
			second:      o.second,
			millisecond: o.millisecond,
			microsecond: o.microsecond,
			nanosecond:  o.nanosecond,
		}), true
	case *temporalInstantObject:
		return intlBigNsToMillis(o.epochNs), true
	case *temporalZonedDateTimeObject:
		return intlBigNsToMillis(o.epochNs), true
	case *temporalPlainYearMonthObject:
		return ms(isoDate{year: o.year, month: o.month, day: o.refDay}, isoTime{}), true
	case *temporalPlainMonthDayObject:
		return ms(isoDate{year: o.refYear, month: o.month, day: o.day}, isoTime{}), true
	}
	return 0, false
}

func intlBigNsToMillis(ns *big.Int) float64 {
	if ns == nil {
		return 0
	}
	f := new(big.Float).SetInt(ns)
	f.Quo(f, big.NewFloat(1e6))
	v, _ := f.Float64()
	return v
}

// intlFormatArgMillis accepts either a number or a Temporal object.
func (r *Runtime) intlFormatArgMillis(v Value) float64 {
	if ms, ok := r.intlTemporalToMillis(v); ok {
		return ms
	}
	return v.ToFloat()
}

// temporalToLocaleString implements Temporal.*.prototype.toLocaleString.
func (r *Runtime) temporalToLocaleString(call FunctionCall, kind string) Value {
	options := r.intlDateTimeOptionsFor(call.Argument(1), kind, kind)
	obj := r.builtin_newIntlDateTimeFormat([]Value{call.Argument(0), options}, r.getIntlDateTimeFormatPrototype())
	dtf, ok := obj.self.(*intlDateTimeFormatObject)
	if !ok {
		return _undefined
	}
	ms, ok := r.intlTemporalToMillis(call.This)
	if !ok {
		panic(r.NewTypeError("Value is not a Temporal object"))
	}
	return intlStr(intlPartsToString(dtf.formatDate(ms)))
}
