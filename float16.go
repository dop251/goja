package goja

import (
	"math"
	"reflect"
	"unsafe"
)

// float64ToFloat16Bits converts f to the nearest IEEE 754 binary16 value, rounding ties to even,
// and returns its bits. The conversion is done from float64 directly, as converting through float32
// first could round twice.
func float64ToFloat16Bits(f float64) uint16 {
	sign := uint16(math.Float64bits(f)>>48) & 0x8000
	if math.IsNaN(f) {
		return 0x7e00
	}
	a := math.Abs(f)
	if a >= 65520 { // halfway between the largest finite value (65504) and 2^16, rounds to infinity
		return sign | 0x7c00
	}
	if a < 0x1p-14 { // subnormal (or zero): the value is m * 2^-24
		return sign | uint16(math.RoundToEven(a*0x1p24))
	}
	frac, exp := math.Frexp(a) // a = frac * 2^exp, frac in [0.5, 1)
	m := math.RoundToEven(frac * 2048)
	if m == 2048 {
		m = 1024
		exp++
	}
	return sign | uint16(exp+14)<<10 | (uint16(m) - 1024)
}

// float16BitsToFloat64 returns the value of the IEEE 754 binary16 number with the given bits.
func float16BitsToFloat64(b uint16) float64 {
	exp := int(b>>10) & 0x1f
	m := float64(b & 0x3ff)
	var f float64
	switch exp {
	case 0:
		f = m * 0x1p-24
	case 0x1f:
		if m != 0 {
			return math.NaN()
		}
		f = math.Inf(1)
	default:
		f = math.Ldexp(1024+m, exp-25)
	}
	if b&0x8000 != 0 {
		f = -f
	}
	return f
}

func toFloat16Bits(v Value) uint16 {
	return float64ToFloat16Bits(v.ToFloat())
}

// toFloat16Exact converts v (which must be a Number) to float16 bits if the value is exactly
// representable as a float16. NaN is not representable (and never strictly equals anything), so
// false is returned.
func toFloat16Exact(v Value) (float64, bool) {
	var f float64
	switch v := v.(type) {
	case valueInt:
		f = float64(v)
	case valueFloat:
		f = float64(v)
	default:
		return 0, false
	}
	if float16BitsToFloat64(float64ToFloat16Bits(f)) != f {
		return 0, false
	}
	return f, true
}

type float16Array []byte

func (a *float16Array) ptr(idx int) *uint16 {
	p := unsafe.SliceData(*a)
	return (*uint16)(unsafe.Add(unsafe.Pointer(p), idx*2))
}

func (a *float16Array) value(idx int) float64 {
	return float16BitsToFloat64(*(a.ptr(idx)))
}

func (a *float16Array) get(idx int) Value {
	return floatToValue(a.value(idx))
}

func (a *float16Array) set(idx int, value Value) {
	*(a.ptr(idx)) = toFloat16Bits(value)
}

func (a *float16Array) less(i, j int) bool {
	return typedFloatLess(a.value(i), a.value(j))
}

func (a *float16Array) swap(i, j int) {
	pi, pj := a.ptr(i), a.ptr(j)
	*pi, *pj = *pj, *pi
}

// Elements are compared by value rather than by bits, so that 0 and -0 are equal.

func (a *float16Array) indexOf(v Value, start, length int) int {
	f, ok := toFloat16Exact(v)
	if !ok {
		return -1
	}
	for i := start; i < start+length; i++ {
		if a.value(i) == f {
			return i - start
		}
	}
	return -1
}

func (a *float16Array) lastIndexOf(v Value, start, length int) int {
	f, ok := toFloat16Exact(v)
	if !ok {
		return -1
	}
	for i := start + length - 1; i >= start; i-- {
		if a.value(i) == f {
			return i - start
		}
	}
	return -1
}

func (a *float16Array) includes(v Value, start, length int) bool {
	switch v.(type) {
	case valueInt, valueFloat:
	default:
		return false
	}
	if math.IsNaN(v.ToFloat()) {
		for i := start; i < start+length; i++ {
			if math.IsNaN(a.value(i)) {
				return true
			}
		}
		return false
	}
	return a.indexOf(v, start, length) >= 0
}

func (a *float16Array) fill(v Value, start, length int) {
	fillSlice(unsafe.Slice(a.ptr(start), length), toFloat16Bits(v))
}

// export returns the raw IEEE 754 binary16 bits of the elements, as Go has no float16 type.
func (a *float16Array) export(offset int, length int) interface{} {
	return unsafe.Slice(a.ptr(offset), length)
}

var typeFloat16Array = reflect.TypeOf(([]uint16)(nil))

func (a *float16Array) exportType() reflect.Type {
	return typeFloat16Array
}

func (r *Runtime) newFloat16ArrayObject(buf *arrayBufferObject, offset, length int, proto *Object) *typedArrayObject {
	return r._newTypedArrayObject(buf, offset, length, 2, r.global.Float16Array, (*float16Array)(&buf.data), proto)
}

func (o *arrayBufferObject) getFloat16(idx int, byteOrder byteOrder) float64 {
	return float16BitsToFloat64(o.getUint16(idx, byteOrder))
}

func (o *arrayBufferObject) setFloat16(idx int, val uint16, byteOrder byteOrder) {
	o.setUint16(idx, val, byteOrder)
}

func (r *Runtime) newFloat16Array(args []Value, newTarget, proto *Object) *Object {
	return r._newTypedArray(args, newTarget, r.newFloat16ArrayObject, proto)
}

func (r *Runtime) getFloat16Array() *Object {
	ret := r.global.Float16Array
	if ret == nil {
		ret = &Object{runtime: r}
		r.global.Float16Array = ret
		r.createTypedArrayCtor(ret, r.newFloat16Array, "Float16Array", 2)
	}
	return ret
}

func (r *Runtime) dataViewProto_getFloat16(call FunctionCall) Value {
	if dv, ok := r.toObject(call.This).self.(*dataViewObject); ok {
		return floatToValue(dv.viewedArrayBuf.getFloat16(dv.getIdxAndByteOrder(r.toIndex(call.Argument(0)), call.Argument(1), 2)))
	}
	panic(r.NewTypeError("Method DataView.prototype.getFloat16 called on incompatible receiver %s", r.objectproto_toString(FunctionCall{This: call.This})))
}

func (r *Runtime) math_f16round(call FunctionCall) Value {
	return floatToValue(float16BitsToFloat64(float64ToFloat16Bits(call.Argument(0).ToFloat())))
}

func (r *Runtime) dataViewProto_setFloat16(call FunctionCall) Value {
	if dv, ok := r.toObject(call.This).self.(*dataViewObject); ok {
		idxVal := r.toIndex(call.Argument(0))
		val := toFloat16Bits(call.Argument(1))
		idx, bo := dv.getIdxAndByteOrder(idxVal, call.Argument(2), 2)
		dv.viewedArrayBuf.setFloat16(idx, val, bo)
		return _undefined
	}
	panic(r.NewTypeError("Method DataView.prototype.setFloat16 called on incompatible receiver %s", r.objectproto_toString(FunctionCall{This: call.This})))
}
