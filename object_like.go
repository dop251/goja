package goja

import (
	"reflect"
	"strconv"
)

// ObjectLike defines interface for js object like abstract that can be exposed to js runtime
type ObjectLike interface {
	GetObjectValue(key string) (val interface{}, exists bool)
	SetObjectValue(key string, val interface{})
	GetObjectKeys() []string
	GetObjectLength() int
	DeleteObjectValue(key string)
}

type objectObjectLikeSimple struct {
	baseObject
	data ObjectLike
}

func (o *objectObjectLikeSimple) init() {
	o.baseObject.init()
	o.prototype = o.val.runtime.global.ObjectPrototype
	o.class = classObject
	o.extensible = true
}

func (o *objectObjectLikeSimple) _get(n Value) Value {
	return o._getStr(n.String())
}

func (o *objectObjectLikeSimple) _getStr(name string) Value {
	v, exists := o.data.GetObjectValue(name)
	if !exists {
		return nil
	}
	return o.val.runtime.ToValue(v)
}

func (o *objectObjectLikeSimple) get(n Value) Value {
	return o.getStr(n.String())
}

func (o *objectObjectLikeSimple) getProp(n Value) Value {
	return o.getPropStr(n.String())
}

func (o *objectObjectLikeSimple) getPropStr(name string) Value {
	if v := o._getStr(name); v != nil {
		return v
	}
	return o.baseObject.getPropStr(name)
}

func (o *objectObjectLikeSimple) getStr(name string) Value {
	if v := o._getStr(name); v != nil {
		return v
	}
	return o.baseObject._getStr(name)
}

func (o *objectObjectLikeSimple) getOwnProp(name string) Value {
	if v := o._getStr(name); v != nil {
		return v
	}
	return o.baseObject.getOwnProp(name)
}

func (o *objectObjectLikeSimple) put(n Value, val Value, throw bool) {
	o.putStr(n.String(), val, throw)
}

func (o *objectObjectLikeSimple) _hasStr(name string) bool {
	_, exists := o.data.GetObjectValue(name)
	return exists
}

func (o *objectObjectLikeSimple) _has(n Value) bool {
	return o._hasStr(n.String())
}

func (o *objectObjectLikeSimple) putStr(name string, val Value, throw bool) {
	if o.extensible || o._hasStr(name) {
		o.data.SetObjectValue(name, val.Export())
	} else {
		o.val.runtime.typeErrorResult(throw, "Host object is not extensible")
	}
}

func (o *objectObjectLikeSimple) hasProperty(n Value) bool {
	if o._has(n) {
		return true
	}
	return o.baseObject.hasProperty(n)
}

func (o *objectObjectLikeSimple) hasPropertyStr(name string) bool {
	if o._hasStr(name) {
		return true
	}
	return o.baseObject.hasOwnPropertyStr(name)
}

func (o *objectObjectLikeSimple) hasOwnProperty(n Value) bool {
	return o._has(n)
}

func (o *objectObjectLikeSimple) hasOwnPropertyStr(name string) bool {
	return o._hasStr(name)
}

func (o *objectObjectLikeSimple) _putProp(name string, value Value, writable, enumerable, configurable bool) Value {
	o.putStr(name, value, false)
	return value
}

func (o *objectObjectLikeSimple) defineOwnProperty(name Value, descr propertyDescr, throw bool) bool {
	if descr.Getter != nil || descr.Setter != nil {
		o.val.runtime.typeErrorResult(throw, "Host objects do not support accessor properties")
		return false
	}
	o.put(name, descr.Value, throw)
	return true
}

/*
func (o *objectObjectLikeSimple) toPrimitiveNumber() Value {
	return o.toPrimitiveString()
}

func (o *objectObjectLikeSimple) toPrimitiveString() Value {
	return stringObjectObject
}

func (o *objectObjectLikeSimple) toPrimitive() Value {
	return o.toPrimitiveString()
}

func (o *objectObjectLikeSimple) assertCallable() (call func(FunctionCall) Value, ok bool) {
	return nil, false
}
*/

func (o *objectObjectLikeSimple) deleteStr(name string, throw bool) bool {
	o.data.DeleteObjectValue(name)
	return true
}

func (o *objectObjectLikeSimple) delete(name Value, throw bool) bool {
	return o.deleteStr(name.String(), throw)
}

type objectLikePropIter struct {
	o         *objectObjectLikeSimple
	propNames []string
	recursive bool
	idx       int
}

func (i *objectLikePropIter) next() (propIterItem, iterNextFunc) {
	for i.idx < len(i.propNames) {
		name := i.propNames[i.idx]
		i.idx++
		if _, exists := i.o.data.GetObjectValue(name); exists {
			return propIterItem{name: name, enumerable: _ENUM_TRUE}, i.next
		}
	}

	if i.recursive {
		return i.o.prototype.self._enumerate(true)()
	}

	return propIterItem{}, nil
}

func (o *objectObjectLikeSimple) enumerate(all, recursive bool) iterNextFunc {
	return (&propFilterIter{
		wrapped: o._enumerate(recursive),
		all:     all,
		seen:    make(map[string]bool),
	}).next
}

func (o *objectObjectLikeSimple) _enumerate(recursive bool) iterNextFunc {
	propNames := o.data.GetObjectKeys()
	return (&objectLikePropIter{
		o:         o,
		propNames: propNames,
		recursive: recursive,
	}).next
}

func (o *objectObjectLikeSimple) export() interface{} {
	return o.data
}

func (o *objectObjectLikeSimple) exportType() reflect.Type {
	return reflect.TypeOf(o.data)
}

func (o *objectObjectLikeSimple) equal(other objectImpl) bool {
	if other, ok := other.(*objectObjectLikeSimple); ok {
		return o == other
	}
	return false
}

func (o *objectObjectLikeSimple) sortLen() int64 {
	return int64(o.data.GetObjectLength())
}

func (o *objectObjectLikeSimple) sortGet(i int64) Value {
	return o.getStr(strconv.FormatInt(i, 10))
}

func (o *objectObjectLikeSimple) swap(i, j int64) {
	ii := strconv.FormatInt(i, 10)
	jj := strconv.FormatInt(j, 10)
	x := o.getStr(ii)
	y := o.getStr(jj)

	o.putStr(ii, y, false)
	o.putStr(jj, x, false)
}
