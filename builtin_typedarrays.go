package goja

import (
	"math"
	"math/bits"
	"unsafe"
)

type byteOrder bool

const (
	bigEndian    byteOrder = false
	littleEndian byteOrder = true
)

var (
	nativeEndian byteOrder
)

type arrayBufferObject struct {
	baseObject
	data []byte
}

type dataViewObject struct {
	baseObject
	viewedArrayBuf      *arrayBufferObject
	byteLen, byteOffset int
}

func (o *dataViewObject) getIdxAndByteOrder(idxVal Value, littleEndianVal Value, size int) (int, byteOrder) {
	idx := o.val.runtime.toIndex(idxVal)
	if bits.UintSize == 32 && idx >= math.MaxInt32 {
		panic(o.val.runtime.newError(o.val.runtime.global.RangeError, "Index %d overflows int", idx))
	}
	getIdx := int(idx)
	o.viewedArrayBuf.ensureNotDetached()
	if getIdx+size > o.byteLen {
		panic(o.val.runtime.newError(o.val.runtime.global.RangeError, "Index %d is out of bounds", getIdx))
	}
	getIdx += o.byteOffset
	var bo byteOrder
	if littleEndianVal != nil {
		if littleEndianVal.ToBoolean() {
			bo = littleEndian
		} else {
			bo = bigEndian
		}
	} else {
		bo = nativeEndian
	}
	return getIdx, bo
}

func (o *arrayBufferObject) export() interface{} {
	return o.data
}

func (o *arrayBufferObject) ensureNotDetached() {
	if o.data == nil {
		panic(o.val.runtime.NewTypeError("ArrayBuffer is detached"))
	}
}

func (o *arrayBufferObject) getFloat32(idx int, byteOrder byteOrder) float32 {
	return math.Float32frombits(o.getUint32(idx, byteOrder))
}

func (o *arrayBufferObject) setFloat32(idx int, val float32, byteOrder byteOrder) {
	o.setUint32(idx, math.Float32bits(val), byteOrder)
}

func (o *arrayBufferObject) getFloat64(idx int, byteOrder byteOrder) float64 {
	return math.Float64frombits(o.getUint64(idx, byteOrder))
}

func (o *arrayBufferObject) setFloat64(idx int, val float64, byteOrder byteOrder) {
	o.setUint64(idx, math.Float64bits(val), byteOrder)
}

func (o *arrayBufferObject) getUint64(idx int, byteOrder byteOrder) uint64 {
	var b []byte
	if byteOrder == nativeEndian {
		b = o.data[idx : idx+8]
	} else {
		b = make([]byte, 8)
		d := o.data[idx : idx+8]
		b[0], b[1], b[2], b[3], b[4], b[5], b[6], b[7] = d[7], d[6], d[5], d[4], d[3], d[2], d[1], d[0]
	}
	return *((*uint64)(unsafe.Pointer(&b[0])))
}

func (o *arrayBufferObject) setUint64(idx int, val uint64, byteOrder byteOrder) {
	if byteOrder == nativeEndian {
		*(*uint64)(unsafe.Pointer(&o.data[idx])) = val
	} else {
		b := (*[8]byte)(unsafe.Pointer(&val))
		d := o.data[idx : idx+8]
		d[0], d[1], d[2], d[3], d[4], d[5], d[6], d[7] = b[7], b[6], b[5], b[4], b[3], b[2], b[1], b[0]
	}
}

func (o *arrayBufferObject) getUint32(idx int, byteOrder byteOrder) uint32 {
	var b []byte
	if byteOrder == nativeEndian {
		b = o.data[idx : idx+4]
	} else {
		b = make([]byte, 4)
		d := o.data[idx : idx+4]
		b[0], b[1], b[2], b[3] = d[3], d[2], d[1], d[0]
	}
	return *((*uint32)(unsafe.Pointer(&b[0])))
}

func (o *arrayBufferObject) setUint32(idx int, val uint32, byteOrder byteOrder) {
	if byteOrder == nativeEndian {
		*(*uint32)(unsafe.Pointer(&o.data[idx])) = val
	} else {
		b := (*[4]byte)(unsafe.Pointer(&val))
		d := o.data[idx : idx+4]
		d[0], d[1], d[2], d[3] = b[3], b[2], b[1], b[0]
	}
}

func (o *arrayBufferObject) getUint16(idx int, byteOrder byteOrder) uint16 {
	var b []byte
	if byteOrder == nativeEndian {
		b = o.data[idx : idx+2]
	} else {
		b = make([]byte, 2)
		d := o.data[idx : idx+2]
		b[0], b[1] = d[1], d[0]
	}
	return *((*uint16)(unsafe.Pointer(&b[0])))
}

func (o *arrayBufferObject) setUint16(idx int, val uint16, byteOrder byteOrder) {
	if byteOrder == nativeEndian {
		*(*uint16)(unsafe.Pointer(&o.data[idx])) = val
	} else {
		b := (*[2]byte)(unsafe.Pointer(&val))
		d := o.data[idx : idx+2]
		d[0], d[1] = b[1], b[0]
	}
}

func (o *arrayBufferObject) getUint8(idx int) uint8 {
	return o.data[idx]
}

func (o *arrayBufferObject) setUint8(idx int, val uint8) {
	o.data[idx] = val
}

func (o *arrayBufferObject) getInt32(idx int, byteOrder byteOrder) int32 {
	return int32(o.getUint32(idx, byteOrder))
}

func (o *arrayBufferObject) setInt32(idx int, val int32, byteOrder byteOrder) {
	o.setUint32(idx, uint32(val), byteOrder)
}

func (o *arrayBufferObject) getInt16(idx int, byteOrder byteOrder) int16 {
	return int16(o.getUint16(idx, byteOrder))
}

func (o *arrayBufferObject) setInt16(idx int, val int16, byteOrder byteOrder) {
	o.setUint16(idx, uint16(val), byteOrder)
}

func (o *arrayBufferObject) getInt8(idx int) int8 {
	return int8(o.data[idx])
}

func (o *arrayBufferObject) setInt8(idx int, val int8) {
	o.setUint8(idx, uint8(val))
}

func (r *Runtime) _newArrayBuffer(proto *Object, o *Object) *arrayBufferObject {
	if o == nil {
		o = &Object{runtime: r}
	}
	b := &arrayBufferObject{
		baseObject: baseObject{
			class:      classObject,
			val:        o,
			prototype:  proto,
			extensible: true,
		},
	}
	o.self = b
	b.init()
	return b
}

func (r *Runtime) builtin_ArrayBuffer(args []Value, proto *Object) *Object {
	b := r._newArrayBuffer(proto, nil)
	if len(args) > 0 {
		b.data = make([]byte, toLength(args[0]))
	}
	return b.val
}

func (r *Runtime) arrayBufferProto_getByteLength(call FunctionCall) Value {
	o := r.toObject(call.This)
	if b, ok := o.self.(*arrayBufferObject); ok {
		if b.data == nil {
			panic(r.NewTypeError("ArrayBuffer is detached"))
		}
		return intToValue(int64(len(b.data)))
	}
	panic(r.NewTypeError("Object is not ArrayBuffer: %s", o))
}

func (r *Runtime) arrayBufferProto_slice(call FunctionCall) Value {
	o := r.toObject(call.This)
	if b, ok := o.self.(*arrayBufferObject); ok {
		l := int64(len(b.data))
		start := relToIdx(toLength(call.Argument(0)), l)
		var stop int64
		if arg := call.Argument(1); arg != _undefined {
			stop = toLength(arg)
		} else {
			stop = l
		}
		stop = relToIdx(stop, l)
		newLen := max(stop-start, 0)
		ret := r.speciesConstructor(o, r.global.ArrayBuffer)([]Value{intToValue(newLen)}, nil)
		if ab, ok := ret.self.(*arrayBufferObject); ok {
			if ab.data == nil {
				panic(r.NewTypeError("Species constructor returned a detached ArrayBuffer"))
			}
			if ret == o {
				panic(r.NewTypeError("Species constructor returned the same ArrayBuffer"))
			}
			if int64(len(ab.data)) < newLen {
				panic(r.NewTypeError("Species constructor returned an ArrayBuffer that is too small: %d", len(ab.data)))
			}
			if b.data == nil {
				panic(r.NewTypeError("Species constructor has detached the current ArrayBuffer"))
			}

			copy(ab.data, b.data[start:stop])
			return ret
		}
		panic(r.NewTypeError("Species constructor did not return an ArrayBuffer: %s", ret.String()))
	}
	panic(r.NewTypeError("Object is not ArrayBuffer: %s", o))
}

func (r *Runtime) arrayBuffer_isView(call FunctionCall) Value {
	if o, ok := call.This.(*Object); ok {
		if _, ok := o.self.(*dataViewObject); ok {
			return valueTrue
		}
	}
	return valueFalse
}

func (r *Runtime) newDataView(args []Value, proto *Object) *Object {
	var bufArg Value
	if len(args) > 0 {
		bufArg = args[0]
	}
	var buffer *arrayBufferObject
	if o, ok := bufArg.(*Object); ok {
		if b, ok := o.self.(*arrayBufferObject); ok {
			buffer = b
		}
	}
	if buffer == nil {
		panic(r.NewTypeError("First argument to DataView constructor must be an ArrayBuffer"))
	}
	var byteOffset, byteLen int
	if len(args) > 1 {
		offset := args[1].ToInteger()
		if offset < 0 {
			panic(r.newError(r.global.RangeError, "Invalid offset"))
		}
		if bits.UintSize == 32 && offset >= math.MaxInt32 {
			panic(r.newError(r.global.RangeError, "Offset %d overflows int", offset))
		}
		byteOffset = int(offset)
		buffer.ensureNotDetached()
		if byteOffset > len(buffer.data) {
			panic(r.newError(r.global.RangeError, "Start offset %d is outside the bounds of the buffer", offset))
		}
		if len(args) > 2 && args[2] != _undefined {
			l := toLength(args[2])
			if bits.UintSize == 32 && l >= math.MaxInt32 {
				panic(r.newError(r.global.RangeError, "Length %d overflows integer", l))
			}
			byteLen = int(l)
			if byteOffset+byteLen > len(buffer.data) {
				panic(r.newError(r.global.RangeError, "Invalid DataView length %d", byteLen))
			}
		} else {
			byteLen = len(buffer.data) - byteOffset
		}
	}
	o := &Object{runtime: r}
	b := &dataViewObject{
		baseObject: baseObject{
			class:      classObject,
			val:        o,
			prototype:  proto,
			extensible: true,
		},
		viewedArrayBuf: buffer,
		byteOffset:     byteOffset,
		byteLen:        byteLen,
	}
	o.self = b
	b.init()
	return o
}

func (r *Runtime) dataViewProto_getBuffer(call FunctionCall) Value {
	if dv, ok := r.toObject(call.This).self.(*dataViewObject); ok {
		return dv.viewedArrayBuf.val
	}
	panic(r.NewTypeError("Method get DataView.prototype.buffer called on incompatible receiver %s", call.This.String()))
}

func (r *Runtime) dataViewProto_getByteLen(call FunctionCall) Value {
	if dv, ok := r.toObject(call.This).self.(*dataViewObject); ok {
		dv.viewedArrayBuf.ensureNotDetached()
		return intToValue(int64(dv.byteLen))
	}
	panic(r.NewTypeError("Method get DataView.prototype.byteLength called on incompatible receiver %s", call.This.String()))
}

func (r *Runtime) dataViewProto_getByteOffset(call FunctionCall) Value {
	if dv, ok := r.toObject(call.This).self.(*dataViewObject); ok {
		dv.viewedArrayBuf.ensureNotDetached()
		return intToValue(int64(dv.byteOffset))
	}
	panic(r.NewTypeError("Method get DataView.prototype.byteOffset called on incompatible receiver %s", call.This.String()))
}

func (r *Runtime) dataViewProto_getFloat32(call FunctionCall) Value {
	if dv, ok := r.toObject(call.This).self.(*dataViewObject); ok {
		return floatToValue(float64(dv.viewedArrayBuf.getFloat32(dv.getIdxAndByteOrder(call.Argument(0), call.Argument(1), 4))))
	}
	panic(r.NewTypeError("Method DataView.prototype.getFloat32 called on incompatible receiver %s", call.This.String()))
}

func (r *Runtime) dataViewProto_getFloat64(call FunctionCall) Value {
	if dv, ok := r.toObject(call.This).self.(*dataViewObject); ok {
		return floatToValue(dv.viewedArrayBuf.getFloat64(dv.getIdxAndByteOrder(call.Argument(0), call.Argument(1), 8)))
	}
	panic(r.NewTypeError("Method DataView.prototype.getFloat64 called on incompatible receiver %s", call.This.String()))
}

func (r *Runtime) dataViewProto_getInt8(call FunctionCall) Value {
	if dv, ok := r.toObject(call.This).self.(*dataViewObject); ok {
		idx, _ := dv.getIdxAndByteOrder(call.Argument(0), call.Argument(1), 1)
		return intToValue(int64(dv.viewedArrayBuf.getInt8(idx)))
	}
	panic(r.NewTypeError("Method DataView.prototype.getInt8 called on incompatible receiver %s", call.This.String()))
}

func (r *Runtime) dataViewProto_getInt16(call FunctionCall) Value {
	if dv, ok := r.toObject(call.This).self.(*dataViewObject); ok {
		return intToValue(int64(dv.viewedArrayBuf.getInt16(dv.getIdxAndByteOrder(call.Argument(0), call.Argument(1), 2))))
	}
	panic(r.NewTypeError("Method DataView.prototype.getInt16 called on incompatible receiver %s", call.This.String()))
}

func (r *Runtime) dataViewProto_getInt32(call FunctionCall) Value {
	if dv, ok := r.toObject(call.This).self.(*dataViewObject); ok {
		return intToValue(int64(dv.viewedArrayBuf.getInt32(dv.getIdxAndByteOrder(call.Argument(0), call.Argument(1), 4))))
	}
	panic(r.NewTypeError("Method DataView.prototype.getInt32 called on incompatible receiver %s", call.This.String()))
}

func (r *Runtime) dataViewProto_getUint8(call FunctionCall) Value {
	if dv, ok := r.toObject(call.This).self.(*dataViewObject); ok {
		idx, _ := dv.getIdxAndByteOrder(call.Argument(0), call.Argument(1), 1)
		return intToValue(int64(dv.viewedArrayBuf.getUint8(idx)))
	}
	panic(r.NewTypeError("Method DataView.prototype.getUint8 called on incompatible receiver %s", call.This.String()))
}

func (r *Runtime) dataViewProto_getUint16(call FunctionCall) Value {
	if dv, ok := r.toObject(call.This).self.(*dataViewObject); ok {
		return intToValue(int64(dv.viewedArrayBuf.getUint16(dv.getIdxAndByteOrder(call.Argument(0), call.Argument(1), 2))))
	}
	panic(r.NewTypeError("Method DataView.prototype.getUint16 called on incompatible receiver %s", call.This.String()))
}

func (r *Runtime) dataViewProto_getUint32(call FunctionCall) Value {
	if dv, ok := r.toObject(call.This).self.(*dataViewObject); ok {
		return intToValue(int64(dv.viewedArrayBuf.getUint32(dv.getIdxAndByteOrder(call.Argument(0), call.Argument(1), 4))))
	}
	panic(r.NewTypeError("Method DataView.prototype.getUint32 called on incompatible receiver %s", call.This.String()))
}

func (r *Runtime) dataViewProto_setFloat32(call FunctionCall) Value {
	if dv, ok := r.toObject(call.This).self.(*dataViewObject); ok {
		idx, bo := dv.getIdxAndByteOrder(call.Argument(0), call.Argument(2), 4)
		dv.viewedArrayBuf.setFloat32(idx, float32(call.Argument(1).ToFloat()), bo)
		return _undefined
	}
	panic(r.NewTypeError("Method DataView.prototype.setFloat32 called on incompatible receiver %s", call.This.String()))
}

func (r *Runtime) dataViewProto_setFloat64(call FunctionCall) Value {
	if dv, ok := r.toObject(call.This).self.(*dataViewObject); ok {
		idx, bo := dv.getIdxAndByteOrder(call.Argument(0), call.Argument(2), 8)
		dv.viewedArrayBuf.setFloat64(idx, call.Argument(1).ToFloat(), bo)
		return _undefined
	}
	panic(r.NewTypeError("Method DataView.prototype.setFloat64 called on incompatible receiver %s", call.This.String()))
}

func (r *Runtime) dataViewProto_setInt8(call FunctionCall) Value {
	if dv, ok := r.toObject(call.This).self.(*dataViewObject); ok {
		idx, _ := dv.getIdxAndByteOrder(call.Argument(0), call.Argument(2), 1)
		dv.viewedArrayBuf.setInt8(idx, toInt8(call.Argument(1)))
		return _undefined
	}
	panic(r.NewTypeError("Method DataView.prototype.setInt8 called on incompatible receiver %s", call.This.String()))
}

func (r *Runtime) dataViewProto_setInt16(call FunctionCall) Value {
	if dv, ok := r.toObject(call.This).self.(*dataViewObject); ok {
		idx, bo := dv.getIdxAndByteOrder(call.Argument(0), call.Argument(2), 2)
		dv.viewedArrayBuf.setInt16(idx, toInt16(call.Argument(1)), bo)
		return _undefined
	}
	panic(r.NewTypeError("Method DataView.prototype.setInt16 called on incompatible receiver %s", call.This.String()))
}

func (r *Runtime) dataViewProto_setInt32(call FunctionCall) Value {
	if dv, ok := r.toObject(call.This).self.(*dataViewObject); ok {
		idx, bo := dv.getIdxAndByteOrder(call.Argument(0), call.Argument(2), 4)
		dv.viewedArrayBuf.setInt32(idx, toInt32(call.Argument(1)), bo)
		return _undefined
	}
	panic(r.NewTypeError("Method DataView.prototype.setInt32 called on incompatible receiver %s", call.This.String()))
}

func (r *Runtime) dataViewProto_setUint8(call FunctionCall) Value {
	if dv, ok := r.toObject(call.This).self.(*dataViewObject); ok {
		idx, _ := dv.getIdxAndByteOrder(call.Argument(0), call.Argument(2), 1)
		dv.viewedArrayBuf.setUint8(idx, toUint8(call.Argument(1)))
		return _undefined
	}
	panic(r.NewTypeError("Method DataView.prototype.setUint8 called on incompatible receiver %s", call.This.String()))
}

func (r *Runtime) dataViewProto_setUint16(call FunctionCall) Value {
	if dv, ok := r.toObject(call.This).self.(*dataViewObject); ok {
		idx, bo := dv.getIdxAndByteOrder(call.Argument(0), call.Argument(2), 2)
		dv.viewedArrayBuf.setUint16(idx, toUInt16(call.Argument(1)), bo)
		return _undefined
	}
	panic(r.NewTypeError("Method DataView.prototype.setUint16 called on incompatible receiver %s", call.This.String()))
}

func (r *Runtime) dataViewProto_setUint32(call FunctionCall) Value {
	if dv, ok := r.toObject(call.This).self.(*dataViewObject); ok {
		idx, bo := dv.getIdxAndByteOrder(call.Argument(0), call.Argument(2), 4)
		dv.viewedArrayBuf.setUint32(idx, toUint32(call.Argument(1)), bo)
		return _undefined
	}
	panic(r.NewTypeError("Method DataView.prototype.setUint32 called on incompatible receiver %s", call.This.String()))
}

func (r *Runtime) createArrayBufferProto(val *Object) objectImpl {
	b := newBaseObjectObj(val, r.global.ObjectPrototype, classObject)
	byteLengthProp := &valueProperty{
		accessor:     true,
		configurable: true,
		getterFunc:   r.newNativeFunc(r.arrayBufferProto_getByteLength, nil, "get byteLength", nil, 0),
	}
	b._put("byteLength", byteLengthProp)
	b._putProp("constructor", r.global.ArrayBuffer, true, false, true)
	b._putProp("slice", r.newNativeFunc(r.arrayBufferProto_slice, nil, "slice", nil, 2), true, false, true)
	b._putSym(symToStringTag, valueProp(asciiString("ArrayBuffer"), false, false, true))
	return b
}

func (r *Runtime) createArrayBuffer(val *Object) objectImpl {
	o := r.newNativeFuncObj(val, r.constructorThrower("ArrayBuffer"), r.builtin_ArrayBuffer, "ArrayBuffer", r.global.ArrayBufferPrototype, 1)
	o._putProp("isView", r.newNativeFunc(r.arrayBuffer_isView, nil, "isView", nil, 1), true, false, true)
	o._putSym(symSpecies, &valueProperty{
		getterFunc:   r.newNativeFunc(r.returnThis, nil, "get [Symbol.species]", nil, 0),
		accessor:     true,
		configurable: true,
	})
	return o
}

func (r *Runtime) createDataViewProto(val *Object) objectImpl {
	b := newBaseObjectObj(val, r.global.ObjectPrototype, classObject)
	b._put("buffer", &valueProperty{
		accessor:     true,
		configurable: true,
		getterFunc:   r.newNativeFunc(r.dataViewProto_getBuffer, nil, "get buffer", nil, 0),
	})
	b._put("byteLength", &valueProperty{
		accessor:     true,
		configurable: true,
		getterFunc:   r.newNativeFunc(r.dataViewProto_getByteLen, nil, "get byteLength", nil, 0),
	})
	b._put("byteOffset", &valueProperty{
		accessor:     true,
		configurable: true,
		getterFunc:   r.newNativeFunc(r.dataViewProto_getByteOffset, nil, "get byteOffset", nil, 0),
	})
	b._putProp("constructor", r.global.DataView, true, false, true)
	b._putProp("getFloat32", r.newNativeFunc(r.dataViewProto_getFloat32, nil, "getFloat32", nil, 1), true, false, true)
	b._putProp("getFloat64", r.newNativeFunc(r.dataViewProto_getFloat64, nil, "getFloat64", nil, 1), true, false, true)
	b._putProp("getInt8", r.newNativeFunc(r.dataViewProto_getInt8, nil, "getInt8", nil, 1), true, false, true)
	b._putProp("getInt16", r.newNativeFunc(r.dataViewProto_getInt16, nil, "getInt16", nil, 1), true, false, true)
	b._putProp("getInt32", r.newNativeFunc(r.dataViewProto_getInt32, nil, "getInt32", nil, 1), true, false, true)
	b._putProp("getUint8", r.newNativeFunc(r.dataViewProto_getUint8, nil, "getUint8", nil, 1), true, false, true)
	b._putProp("getUint16", r.newNativeFunc(r.dataViewProto_getUint16, nil, "getUint16", nil, 1), true, false, true)
	b._putProp("getUint32", r.newNativeFunc(r.dataViewProto_getUint32, nil, "getUint32", nil, 1), true, false, true)
	b._putProp("setFloat32", r.newNativeFunc(r.dataViewProto_setFloat32, nil, "setFloat32", nil, 2), true, false, true)
	b._putProp("setFloat64", r.newNativeFunc(r.dataViewProto_setFloat64, nil, "setFloat64", nil, 2), true, false, true)
	b._putProp("setInt8", r.newNativeFunc(r.dataViewProto_setInt8, nil, "setInt8", nil, 2), true, false, true)
	b._putProp("setInt16", r.newNativeFunc(r.dataViewProto_setInt16, nil, "setInt16", nil, 2), true, false, true)
	b._putProp("setInt32", r.newNativeFunc(r.dataViewProto_setInt32, nil, "setInt32", nil, 2), true, false, true)
	b._putProp("setUint8", r.newNativeFunc(r.dataViewProto_setUint8, nil, "setUint8", nil, 2), true, false, true)
	b._putProp("setUint16", r.newNativeFunc(r.dataViewProto_setUint16, nil, "setUint16", nil, 2), true, false, true)
	b._putProp("setUint32", r.newNativeFunc(r.dataViewProto_setUint32, nil, "setUint32", nil, 2), true, false, true)
	b._putSym(symToStringTag, valueProp(asciiString("DataView"), false, false, true))

	return b
}

func (r *Runtime) createDataView(val *Object) objectImpl {
	o := r.newNativeFuncObj(val, r.constructorThrower("DataView"), r.newDataView, "DataView", r.global.DataViewPrototype, 3)
	return o
}

func (r *Runtime) initTypedArrays() {

	r.global.ArrayBufferPrototype = r.newLazyObject(r.createArrayBufferProto)
	r.global.ArrayBuffer = r.newLazyObject(r.createArrayBuffer)
	r.addToGlobal("ArrayBuffer", r.global.ArrayBuffer)

	r.global.DataViewPrototype = r.newLazyObject(r.createDataViewProto)
	r.global.DataView = r.newLazyObject(r.createDataView)
	r.addToGlobal("DataView", r.global.DataView)

}

func init() {
	buf := [2]byte{}
	*(*uint16)(unsafe.Pointer(&buf[0])) = uint16(0xCAFE)

	switch buf {
	case [2]byte{0xFE, 0xCA}:
		nativeEndian = littleEndian
	case [2]byte{0xCA, 0xFE}:
		nativeEndian = bigEndian
	default:
		panic("Could not determine native endianness.")
	}
}
