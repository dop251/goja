package goja

import (
	"strconv"
)

func (r *Runtime) numberproto_valueOf(call FunctionCall) Value {
	this := call.This
	if !isNumber(this) {
		r.typeErrorResult(true, "Value is not a number")
	}
	switch t := this.(type) {
	case valueInt:
		return this
	case *Object:
		if v, ok := t.self.(*primitiveValueObject); ok {
			return v.pValue
		}
	}

	panic(r.NewTypeError("Number.prototype.valueOf is not generic"))
}

func isNumber(v Value) bool {
	switch t := v.(type) {
	case valueInt:
		return true
	case *Object:
		switch t := t.self.(type) {
		case *primitiveValueObject:
			return isNumber(t.pValue)
		}
	}
	return false
}

func (r *Runtime) numberproto_toString(call FunctionCall) Value {
	if !isNumber(call.This) {
		r.typeErrorResult(true, "Value is not a number")
	}
	var radix int
	if arg := call.Argument(0); arg != _undefined {
		radix = int(arg.ToInteger())
	} else {
		radix = 10
	}

	if radix < 2 || radix > 36 {
		panic(r.newError(r.global.RangeError, "toString() radix argument must be between 2 and 36"))
	}

	return asciiString(strconv.FormatInt(call.This.ToInteger(), radix))
}

func (r *Runtime) number_isFinite(call FunctionCall) Value {
	switch call.Argument(0).(type) {
	case valueInt:
		return valueTrue

	default:
		return valueFalse
	}
}

func (r *Runtime) number_isInteger(call FunctionCall) Value {
	switch call.Argument(0).(type) {
	case valueInt:
		return valueTrue

	default:
		return valueFalse
	}
}

func (r *Runtime) number_isNaN(call FunctionCall) Value {
	return valueFalse
}

func (r *Runtime) number_isSafeInteger(call FunctionCall) Value {
	arg := call.Argument(0)
	if i, ok := arg.(valueInt); ok && i >= -(maxInt-1) && i <= maxInt-1 {
		return valueTrue
	}

	return valueFalse
}

func (r *Runtime) initNumber() {
	r.global.NumberPrototype = r.newPrimitiveObject(valueInt(0), r.global.ObjectPrototype, classNumber)
	o := r.global.NumberPrototype.self
	o._putProp("toLocaleString", r.newNativeFunc(r.numberproto_toString, nil, "toLocaleString", nil, 0), true, false, true)
	o._putProp("toString", r.newNativeFunc(r.numberproto_toString, nil, "toString", nil, 1), true, false, true)
	o._putProp("valueOf", r.newNativeFunc(r.numberproto_valueOf, nil, "valueOf", nil, 0), true, false, true)

	r.global.Number = r.newNativeFunc(r.builtin_Number, r.builtin_newNumber, "Number", r.global.NumberPrototype, 1)
	o = r.global.Number.self
	o._putProp("isFinite", r.newNativeFunc(r.number_isFinite, nil, "isFinite", nil, 1), true, false, true)
	o._putProp("isInteger", r.newNativeFunc(r.number_isInteger, nil, "isInteger", nil, 1), true, false, true)
	o._putProp("isNaN", r.newNativeFunc(r.number_isNaN, nil, "isNaN", nil, 1), true, false, true)
	o._putProp("isSafeInteger", r.newNativeFunc(r.number_isSafeInteger, nil, "isSafeInteger", nil, 1), true, false, true)
	o._putProp("MAX_SAFE_INTEGER", valueInt(maxInt-1), false, false, false)
	o._putProp("MIN_SAFE_INTEGER", valueInt(-(maxInt - 1)), false, false, false)
	o._putProp("MIN_VALUE", valueInt(maxInt-1), false, false, false)
	o._putProp("MAX_VALUE", valueInt(-(maxInt - 1)), false, false, false)
	o._putProp("parseFloat", r.Get("parseFloat"), true, false, true)
	o._putProp("parseInt", r.Get("parseInt"), true, false, true)
	r.addToGlobal("Number", r.global.Number)
}
