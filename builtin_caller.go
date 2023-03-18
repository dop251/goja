package goja

import (
	"hash/maphash"
	"reflect"
	"strconv"
	"unsafe"

	"github.com/dop251/goja/unistring"
)

var (
	reflectTypeGoCaller = reflect.TypeOf(&valueGoCaller{})
	_GoRaw              = &valueGoCaller{
		value:       1,
		hashValue:   randomHash(),
		stringValue: asciiString("GoRaw"),
	}
	_GoNumber = &valueGoCaller{
		value:       2,
		hashValue:   randomHash(),
		stringValue: asciiString("GoNumber"),
	}
	_GoRawNumber = &valueGoCaller{
		value:       1 | 2,
		hashValue:   randomHash(),
		stringValue: asciiString("GoRawNumber"),
	}

	_GoAsync = &valueGoCaller{
		value:       4,
		hashValue:   randomHash(),
		stringValue: asciiString("GoAsync"),
	}
	_GoAsyncRaw = &valueGoCaller{
		value:       1 | 4,
		hashValue:   randomHash(),
		stringValue: asciiString("GoAsyncRaw"),
	}
	_GoAsyncNumber = &valueGoCaller{
		value:       2 | 4,
		hashValue:   randomHash(),
		stringValue: asciiString("GoAsyncNumber"),
	}
	_GoAsyncRawNumber = &valueGoCaller{
		value:       1 | 2 | 4,
		hashValue:   randomHash(),
		stringValue: asciiString("GoAsyncRawNumber"),
	}
)

func (r *Runtime) initCaller() {
	o := r.globalObject.self
	o._putProp("GoRaw", _GoRaw, false, false, true)
	o._putProp("GoNumber", _GoNumber, false, false, true)
	o._putProp("GoRawNumber", _GoRawNumber, false, false, true)

	o._putProp("GoAsync", _GoAsync, false, false, true)
	o._putProp("GoAsyncRaw", _GoAsyncRaw, false, false, true)
	o._putProp("GoAsyncNumber", _GoAsyncNumber, false, false, true)
	o._putProp("GoAsyncRawNumber", _GoAsyncRawNumber, false, false, true)
}

// Set up the runner so that calling go functions asynchronously can called resolve/reject on the loop
func (r *Runtime) SetRunOnLoop(runner func(f func(*Runtime))) {
	r.runOnLoop = runner
}

// called f on the loop, must call SetRunOnLoop to set the runner before
func (r *Runtime) RunOnLoop(f func(*Runtime)) {
	r.runOnLoop(f)
}

func toValue64(v interface{}) interface{} {
	if strconv.IntSize < 64 {
		switch i := v.(type) {
		case int64:
			return Int64(i)
		case uint64:
			return Uint64(i)
		case []int64:
			return *(*[]Int64)(unsafe.Pointer(&i))
		case []uint64:
			return *(*[]Uint64)(unsafe.Pointer(&i))
		}
	} else {
		switch i := v.(type) {
		case int64:
			return Int64(i)
		case int:
			return Int64(i)
		case uint64:
			return Uint64(i)
		case uint:
			return Uint64(i)
		case []int64:
			return *(*[]Int64)(unsafe.Pointer(&i))
		case []int:
			return *(*[]Int64)(unsafe.Pointer(&i))
		case []uint64:
			return *(*[]Uint64)(unsafe.Pointer(&i))
		case []uint:
			return *(*[]Uint64)(unsafe.Pointer(&i))
		}
	}
	return v
}

type Int64 int64
type Uint64 uint64

func (v Int64) String() string {
	return strconv.FormatInt(int64(v), 10)
}
func (v Uint64) String() string {
	return strconv.FormatUint(uint64(v), 10)
}
func (v Int64) Uint64() Uint64 {
	return Uint64(v)
}
func (v Uint64) Int64() Int64 {
	return Int64(v)
}
func (v Int64) Value() int64 {
	return int64(v)
}
func (v Uint64) Value() uint64 {
	return uint64(v)
}
func (v Int64) Add(o Int64) Int64 {
	return v + o
}
func (v Uint64) Add(o Uint64) Uint64 {
	return v + o
}
func (v Int64) Sub(o Int64) Int64 {
	return v - o
}
func (v Uint64) Sub(o Uint64) Uint64 {
	return v - o
}
func (v Int64) Mul(o Int64) Int64 {
	return v * o
}
func (v Uint64) Mul(o Uint64) Uint64 {
	return v * o
}
func (v Int64) Div(o Int64) Int64 {
	return v / o
}
func (v Uint64) Div(o Uint64) Uint64 {
	return v / o
}
func (v Int64) Mod(o Int64) Int64 {
	return v % o
}
func (v Uint64) Mod(o Uint64) Uint64 {
	return v % o
}
func (v Int64) Abs() Int64 {
	if v < 0 {
		return -v
	}
	return v
}
func (v Int64) Neg() Int64 {
	return -v
}
func (v Int64) And(o Int64) Int64 {
	return v & o
}
func (v Uint64) And(o Uint64) Uint64 {
	return v & o
}
func (v Int64) Or(o Int64) Int64 {
	return v | o
}
func (v Uint64) Or(o Uint64) Uint64 {
	return v | o
}
func (v Int64) Xor(o Int64) Int64 {
	return v ^ o
}
func (v Uint64) Xor(o Uint64) Uint64 {
	return v ^ o
}
func (v Int64) Not() Int64 {
	return ^v
}
func (v Uint64) Not() Uint64 {
	return ^v
}
func (v Int64) Lsh(n int) Int64 {
	return v << n
}
func (v Uint64) Lsh(n int) Uint64 {
	return v << n
}
func (v Int64) Rsh(n int) Int64 {
	return v >> n
}
func (v Uint64) Rsh(n int) Uint64 {
	return v >> n
}
func (v Int64) Cmp(o Int64) int {
	if v < o {
		return -1
	} else if v > o {
		return 1
	}
	return 0
}
func (v Uint64) Cmp(o Uint64) int {
	if v < o {
		return -1
	} else if v > o {
		return 1
	}
	return 0
}

type valueGoCaller struct {
	value       int64
	hashValue   uint64
	stringValue asciiString
}

func (v *valueGoCaller) ToInteger() int64 {
	return v.value
}
func (v *valueGoCaller) toString() valueString {
	return v.stringValue
}
func (v *valueGoCaller) string() unistring.String {
	return v.stringValue.string()
}
func (v *valueGoCaller) ToString() Value {
	return v.stringValue
}
func (v *valueGoCaller) String() string {
	return string(v.stringValue)
}
func (v *valueGoCaller) ToFloat() float64 {
	return float64(v.value)
}
func (v *valueGoCaller) ToNumber() Value {
	return valueInt(v.value)
}
func (v *valueGoCaller) ToBoolean() bool {
	return true
}
func (v *valueGoCaller) ToObject(r *Runtime) *Object {
	r.typeErrorResult(true, "Cannot convert "+string(v.stringValue)+" to object")
	return nil
}
func (v *valueGoCaller) SameAs(other Value) bool {
	_, same := other.(*valueGoCaller)
	return same
}
func (v *valueGoCaller) Equals(other Value) bool {
	return v.value == other.ToInteger()
}
func (v *valueGoCaller) StrictEquals(other Value) bool {
	o, same := other.(*valueGoCaller)
	return same && o.value == v.value
}
func (v *valueGoCaller) Export() interface{} {
	return v
}
func (v *valueGoCaller) ExportType() reflect.Type {
	return reflectTypeGoCaller
}
func (v *valueGoCaller) baseObject(r *Runtime) *Object {
	return nil
}
func (v *valueGoCaller) hash(hasher *maphash.Hash) uint64 {
	return v.hashValue
}
