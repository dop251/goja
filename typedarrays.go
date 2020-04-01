package goja

import (
	"math"
	"math/bits"
	"sort"
	"strconv"
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

type typedArrayObjectCtor func(buf *arrayBufferObject, offset, length int, proto *Object) *typedArrayObject

type arrayBufferObject struct {
	baseObject
	data []byte
}

type dataViewObject struct {
	baseObject
	viewedArrayBuf      *arrayBufferObject
	byteLen, byteOffset int
}

type typedArray interface {
	sort.Interface
	toRaw(Value) uint64
	get(idx int) Value
	set(idx int, value Value)
	getRaw(idx int) uint64
	setRaw(idx int, raw uint64)
}

type uint8Array []uint8
type uint16Array []uint16
type uint32Array []uint32

func (a *uint8Array) get(idx int) Value {
	return intToValue(int64((*a)[idx]))
}

func (a *uint8Array) getRaw(idx int) uint64 {
	return uint64((*a)[idx])
}

func (a *uint8Array) set(idx int, value Value) {
	(*a)[idx] = toUint8(value)
}

func (a *uint8Array) toRaw(v Value) uint64 {
	return uint64(toUint8(v))
}

func (a *uint8Array) setRaw(idx int, v uint64) {
	(*a)[idx] = uint8(v)
}

func (a *uint8Array) Len() int {
	return len(*a)
}

func (a *uint8Array) Less(i, j int) bool {
	return (*a)[i] < (*a)[j]
}

func (a *uint8Array) Swap(i, j int) {
	(*a)[i], (*a)[j] = (*a)[j], (*a)[i]
}

func (a *uint16Array) get(idx int) Value {
	return intToValue(int64((*a)[idx]))
}

func (a *uint16Array) getRaw(idx int) uint64 {
	return uint64((*a)[idx])
}

func (a *uint16Array) set(idx int, value Value) {
	(*a)[idx] = toUint16(value)
}

func (a *uint16Array) toRaw(v Value) uint64 {
	return uint64(toUint16(v))
}

func (a *uint16Array) setRaw(idx int, v uint64) {
	(*a)[idx] = uint16(v)
}

func (a *uint16Array) Len() int {
	return len(*a)
}

func (a *uint16Array) Less(i, j int) bool {
	return (*a)[i] < (*a)[j]
}

func (a *uint16Array) Swap(i, j int) {
	(*a)[i], (*a)[j] = (*a)[j], (*a)[i]
}

func (a *uint32Array) get(idx int) Value {
	return intToValue(int64((*a)[idx]))
}

func (a *uint32Array) getRaw(idx int) uint64 {
	return uint64((*a)[idx])
}

func (a *uint32Array) set(idx int, value Value) {
	(*a)[idx] = toUint32(value)
}

func (a *uint32Array) toRaw(v Value) uint64 {
	return uint64(toUint32(v))
}

func (a *uint32Array) setRaw(idx int, v uint64) {
	(*a)[idx] = uint32(v)
}

func (a *uint32Array) Len() int {
	return len(*a)
}

func (a *uint32Array) Less(i, j int) bool {
	return (*a)[i] < (*a)[j]
}

func (a *uint32Array) Swap(i, j int) {
	(*a)[i], (*a)[j] = (*a)[j], (*a)[i]
}

type typedArrayObject struct {
	baseObject
	viewedArrayBuf *arrayBufferObject
	defaultCtor    *Object
	length, offset int
	elemSize       int
	typedArray     typedArray
}

func (a *typedArrayObject) _getIdx(idx int) Value {
	a.viewedArrayBuf.ensureNotDetached()
	if idx < a.length {
		return a.typedArray.get(idx + a.offset)
	}
	return nil
}

func strToTAIdx(s string) (int, bool) {
	i, err := strconv.ParseInt(s, 10, bits.UintSize)
	if err != nil {
		return 0, false
	}
	return int(i), true
}

func (a *typedArrayObject) getOwnPropStr(name string) Value {
	if idx, ok := strToTAIdx(name); ok {
		v := a._getIdx(idx)
		if v != nil {
			return &valueProperty{
				value:      v,
				writable:   true,
				enumerable: true,
			}
		}
		return nil
	}
	return a.baseObject.getOwnPropStr(name)
}

func (a *typedArrayObject) getOwnPropIdx(idx valueInt) Value {
	v := a._getIdx(toInt(int64(idx)))
	if v != nil {
		return &valueProperty{
			value:      v,
			writable:   true,
			enumerable: true,
		}
	}
	return nil
}

func (a *typedArrayObject) getStr(name string, receiver Value) Value {
	if idx, ok := strToTAIdx(name); ok {
		prop := a._getIdx(idx)
		if prop == nil {
			if a.prototype != nil {
				if receiver == nil {
					return a.prototype.self.getStr(name, a.val)
				}
				return a.prototype.self.getStr(name, receiver)
			}
		}
		return prop
	}
	return a.baseObject.getStr(name, receiver)
}

func (a *typedArrayObject) getIdx(idx valueInt, receiver Value) Value {
	prop := a._getIdx(toInt(int64(idx)))
	if prop == nil {
		if a.prototype != nil {
			if receiver == nil {
				return a.prototype.self.getIdx(idx, a.val)
			}
			return a.prototype.self.getIdx(idx, receiver)
		}
	}
	return prop
}

func (a *typedArrayObject) _putIdx(idx int, v Value, throw bool) bool {
	a.viewedArrayBuf.ensureNotDetached()
	if idx >= 0 && idx < a.length {
		a.typedArray.set(idx+a.offset, v)
		return true
	}
	// As far as I understand the specification this should throw, but neither V8 nor SpiderMonkey does
	return false
}

func (a *typedArrayObject) _hasIdx(idx int) bool {
	a.viewedArrayBuf.ensureNotDetached()
	return idx >= 0 && idx < a.length
}

func (a *typedArrayObject) setOwnStr(p string, v Value, throw bool) bool {
	if idx, ok := strToTAIdx(p); ok {
		return a._putIdx(idx, v, throw)
	}
	return a.baseObject.setOwnStr(p, v, throw)
}

func (a *typedArrayObject) setOwnIdx(p valueInt, v Value, throw bool) bool {
	return a._putIdx(toInt(int64(p)), v, throw)
}

func (a *typedArrayObject) setForeignStr(p string, v, receiver Value, throw bool) (res bool, handled bool) {
	return a._setForeignStr(p, a.getOwnPropStr(p), v, receiver, throw)
}

func (a *typedArrayObject) setForeignIdx(p valueInt, v, receiver Value, throw bool) (res bool, handled bool) {
	return a._setForeignIdx(p, trueValIfPresent(a.hasOwnPropertyIdx(p)), v, receiver, throw)
}

func (a *typedArrayObject) hasOwnPropertyStr(name string) bool {
	if idx, ok := strToTAIdx(name); ok {
		a.viewedArrayBuf.ensureNotDetached()
		return idx < a.length
	}

	return a.baseObject.hasOwnPropertyStr(name)
}

func (a *typedArrayObject) hasOwnPropertyIdx(idx valueInt) bool {
	return a._hasIdx(toInt(int64(idx)))
}

func (a *typedArrayObject) _defineIdxProperty(idx int, desc PropertyDescriptor, throw bool) bool {
	prop, ok := a._defineOwnProperty(strconv.Itoa(idx), a.getOwnPropIdx(valueInt(idx)), desc, throw)
	if ok {
		return a._putIdx(idx, prop, throw)
	}
	return ok
}

func (a *typedArrayObject) defineOwnPropertyStr(name string, desc PropertyDescriptor, throw bool) bool {
	if idx, ok := strToTAIdx(name); ok {
		return a._defineIdxProperty(idx, desc, throw)
	}
	return a.baseObject.defineOwnPropertyStr(name, desc, throw)
}

func (a *typedArrayObject) defineOwnPropertyIdx(name valueInt, desc PropertyDescriptor, throw bool) bool {
	return a._defineIdxProperty(toInt(int64(name)), desc, throw)
}

func (a *typedArrayObject) deleteStr(name string, throw bool) bool {
	if idx, ok := strToTAIdx(name); ok {
		if idx < a.length {
			a.val.runtime.typeErrorResult(throw, "Cannot delete property '%d' of %s", idx, a.val.String())
		}
	}

	return a.baseObject.deleteStr(name, throw)
}

func (a *typedArrayObject) deleteIdx(idx valueInt, throw bool) bool {
	if idx >= 0 && int64(idx) < int64(a.length) {
		a.val.runtime.typeErrorResult(throw, "Cannot delete property '%d' of %s", idx, a.val.String())
	}

	return true
}

func (a *typedArrayObject) ownKeys(all bool, accum []Value) []Value {
	if accum == nil {
		accum = make([]Value, 0, a.length)
	}
	for i := 0; i < a.length; i++ {
		accum = append(accum, asciiString(strconv.Itoa(i)))
	}
	return a.baseObject.ownKeys(all, accum)
}

type typedArrayPropIter struct {
	a   *typedArrayObject
	idx int
}

func (i *typedArrayPropIter) next() (propIterItem, iterNextFunc) {
	if i.idx < i.a.length {
		name := strconv.Itoa(i.idx)
		prop := i.a._getIdx(i.idx)
		i.idx++
		return propIterItem{name: name, value: prop}, i.next
	}

	return i.a.baseObject.enumerateUnfiltered()()
}

func (a *typedArrayObject) enumerateUnfiltered() iterNextFunc {
	return (&typedArrayPropIter{
		a: a,
	}).next
}

func (r *Runtime) _newTypedArrayObject(buf *arrayBufferObject, offset, length, elemSize int, defCtor *Object, arr typedArray, proto *Object) *typedArrayObject {
	o := &Object{runtime: r}
	a := &typedArrayObject{
		baseObject: baseObject{
			val:       o,
			class:     classObject,
			prototype: proto,
		},
		viewedArrayBuf: buf,
		offset:         offset,
		length:         length,
		elemSize:       elemSize,
		defaultCtor:    defCtor,
		typedArray:     arr,
	}
	o.self = a
	a.init()
	return a

}

func (r *Runtime) newUint8ArrayObject(buf *arrayBufferObject, offset, length int, proto *Object) *typedArrayObject {
	return r._newTypedArrayObject(buf, offset, length, 1, r.global.Uint8Array, (*uint8Array)(&buf.data), proto)
}

func (r *Runtime) newUint16ArrayObject(buf *arrayBufferObject, offset, length int, proto *Object) *typedArrayObject {
	return r._newTypedArrayObject(buf, offset, length, 2, r.global.Uint16Array, (*uint16Array)(unsafe.Pointer(&buf.data)), proto)
}

func (r *Runtime) newUint32ArrayObject(buf *arrayBufferObject, offset, length int, proto *Object) *typedArrayObject {
	return r._newTypedArrayObject(buf, offset, length, 4, r.global.Uint16Array, (*uint32Array)(unsafe.Pointer(&buf.data)), proto)
}

func (o *dataViewObject) getIdxAndByteOrder(idxVal, littleEndianVal Value, size int) (int, byteOrder) {
	getIdx := o.val.runtime.toIndex(idxVal)
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
