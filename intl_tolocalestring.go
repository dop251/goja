package goja

import (
	"strconv"

	"github.com/dop251/goja/unistring"
)

// intlNumberToLocaleString implements Number.prototype.toLocaleString and
// BigInt.prototype.toLocaleString.
func (r *Runtime) intlNumberToLocaleString(call FunctionCall) Value {
	var n float64
	switch v := call.This.(type) {
	case valueInt:
		n = float64(v)
	case valueFloat:
		n = float64(v)
	case *Object:
		n = v.ToNumber().ToFloat()
	default:
		panic(r.NewTypeError("Number.prototype.toLocaleString called on incompatible receiver"))
	}
	obj := r.builtin_newIntlNumberFormat([]Value{call.Argument(0), call.Argument(1)}, r.getIntlNumberFormatPrototype())
	nf, ok := obj.self.(*intlNumberFormatObject)
	if !ok {
		return intlStr("")
	}
	return intlStr(intlPartsToString(nf.formatParts(n)))
}

// intlBigIntToLocaleString implements BigInt.prototype.toLocaleString.
func (r *Runtime) intlBigIntToLocaleString(call FunctionCall) Value {
	bi, ok := call.This.(*valueBigInt)
	if !ok {
		panic(r.NewTypeError("BigInt.prototype.toLocaleString called on incompatible receiver"))
	}
	n, _ := strconv.ParseFloat(bi.string().String(), 64)
	obj := r.builtin_newIntlNumberFormat([]Value{call.Argument(0), call.Argument(1)}, r.getIntlNumberFormatPrototype())
	nf, ok := obj.self.(*intlNumberFormatObject)
	if !ok {
		return intlStr("")
	}
	return intlStr(intlPartsToString(nf.formatParts(n)))
}

// intlCopyOptionsObject creates a plain object with the own enumerable
// properties of options.
func (r *Runtime) intlCopyOptionsObject(options Value) *Object {
	o := r.newBaseObject(r.global.ObjectPrototype, classObject)
	if opt, ok := options.(*Object); ok {
		for _, k := range opt.Keys() {
			key := unistring.NewFromString(k)
			o.setOwnStr(key, nilSafe(opt.self.getStr(key, nil)), false)
		}
	}
	return o.val
}

func (r *Runtime) intlSetDateDefaults(o *Object, names ...string) {
	for _, n := range names {
		key := unistring.NewFromString(n)
		if o.self.getStr(key, nil) == nil {
			o.self.setOwnStr(key, intlStr("numeric"), false)
		}
	}
}

func (r *Runtime) intlHasAnyOption(o *Object, names ...string) bool {
	for _, n := range names {
		if o.self.getStr(unistring.NewFromString(n), nil) != nil {
			return true
		}
	}
	return false
}

// intlDateTimeOptionsFor implements ToDateTimeOptions.
func (r *Runtime) intlDateTimeOptionsFor(options Value, required, defaults string) *Object {
	o := r.intlCopyOptionsObject(options)
	dateKeys := []string{"weekday", "era", "year", "month", "day"}
	timeKeys := []string{"dayPeriod", "hour", "minute", "second", "fractionalSecondDigits"}
	switch required {
	case "date":
		if !r.intlHasAnyOption(o, dateKeys...) {
			r.intlSetDateDefaults(o, "year", "month", "day")
		}
	case "time":
		if !r.intlHasAnyOption(o, timeKeys...) {
			r.intlSetDateDefaults(o, "hour", "minute", "second")
		}
	default:
		if !r.intlHasAnyOption(o, dateKeys...) && !r.intlHasAnyOption(o, timeKeys...) {
			r.intlSetDateDefaults(o, "year", "month", "day", "hour", "minute", "second")
		}
	}
	return o
}

func (r *Runtime) intlDateToLocaleString(call FunctionCall, required, defaults string) Value {
	obj := r.toObject(call.This)
	d, ok := obj.self.(*dateObject)
	if !ok {
		panic(r.NewTypeError("Method Date.prototype.toLocaleString is called on incompatible receiver"))
	}
	if !d.isSet() {
		return stringInvalidDate
	}
	options := r.intlDateTimeOptionsFor(call.Argument(1), required, defaults)
	o := r.builtin_newIntlDateTimeFormat([]Value{call.Argument(0), options}, r.getIntlDateTimeFormatPrototype())
	dtf, ok := o.self.(*intlDateTimeFormatObject)
	if !ok {
		return intlStr("")
	}
	ms := float64(d.time().UnixNano()) / 1e6
	return intlStr(intlPartsToString(dtf.formatDate(ms)))
}
