package goja

import (
	"reflect"
	"strconv"
)

type objectGoSliceReflect struct {
	objectGoReflect
	lengthProp      valueProperty
	sliceExtensible bool
}

func (o *objectGoSliceReflect) init() {
	o.objectGoReflect.init()
	o.class = classArray
	o.prototype = o.val.runtime.global.ArrayPrototype
	o.sliceExtensible = o.value.CanSet()
	o.lengthProp.writable = o.sliceExtensible
	o._setLen()
	o.baseObject._put("length", &o.lengthProp)
}

func (o *objectGoSliceReflect) _setLen() {
	o.lengthProp.value = intToValue(int64(o.value.Len()))
}

func (o *objectGoSliceReflect) _has(n Value) bool {
	if idx := toIdx(n); idx >= 0 {
		return idx < int64(o.value.Len())
	}
	return false
}

func (o *objectGoSliceReflect) _hasStr(name string) bool {
	if idx := strToIdx(name); idx >= 0 {
		return idx < int64(o.value.Len())
	}
	return false
}

func (o *objectGoSliceReflect) getIdx(idx int64) Value {
	if idx < int64(o.value.Len()) {
		v := o.value.Index(int(idx))
		if (v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface) && v.IsNil() {
			return nil
		}
		return o.val.runtime.ToValue(v.Interface())
	}
	return nil
}

func (o *objectGoSliceReflect) _get(n Value) Value {
	if idx := toIdx(n); idx >= 0 {
		return o.getIdx(idx)
	}
	return nil
}

func (o *objectGoSliceReflect) _getStr(name string) Value {
	if idx := strToIdx(name); idx >= 0 {
		return o.getIdx(idx)
	}
	return nil
}

func (o *objectGoSliceReflect) get(n Value, receiver Value) Value {
	if s, ok := n.(*valueSymbol); ok {
		return o.getSym(s, receiver)
	}
	if v := o._get(n); v != nil {
		return v
	}
	return o.objectGoReflect.getStr(n.String(), receiver)
}

func (o *objectGoSliceReflect) getStr(name string, receiver Value) Value {
	if v := o._getStr(name); v != nil {
		return v
	}
	return o.objectGoReflect.getStr(name, receiver)
}

func (o *objectGoSliceReflect) getOwnPropStr(name string) Value {
	if v := o._getStr(name); v != nil {
		return v
	}
	return o.objectGoReflect.getOwnPropStr(name)
}

func (o *objectGoSliceReflect) getOwnProp(name Value) Value {
	if v := o._get(name); v != nil {
		return v
	}
	return o.objectGoReflect.getOwnProp(name)
}

func (o *objectGoSliceReflect) putIdx(idx int64, v Value, throw bool) {
	if idx >= int64(o.value.Len()) {
		if !o.sliceExtensible {
			o.val.runtime.typeErrorResult(throw, "Cannot extend a Go unaddressable reflect slice")
			return
		}
		o.grow(int(idx + 1))
	}
	val, err := o.val.runtime.toReflectValue(v, o.value.Type().Elem())
	if err != nil {
		o.val.runtime.typeErrorResult(throw, "Go type conversion error: %v", err)
		return
	}
	o.value.Index(int(idx)).Set(val)
}

func (o *objectGoSliceReflect) grow(size int) {
	newcap := o.value.Cap()
	if newcap < size {
		// Use the same algorithm as in runtime.growSlice
		doublecap := newcap + newcap
		if size > doublecap {
			newcap = size
		} else {
			if o.value.Len() < 1024 {
				newcap = doublecap
			} else {
				for newcap < size {
					newcap += newcap / 4
				}
			}
		}

		n := reflect.MakeSlice(o.value.Type(), size, newcap)
		reflect.Copy(n, o.value)
		o.value.Set(n)
	} else {
		tail := o.value.Slice(o.value.Len(), size)
		zero := reflect.Zero(o.value.Type().Elem())
		for i := 0; i < tail.Len(); i++ {
			tail.Index(i).Set(zero)
		}
		o.value.SetLen(size)
	}
	o._setLen()
}

func (o *objectGoSliceReflect) shrink(size int) {
	tail := o.value.Slice(size, o.value.Len())
	zero := reflect.Zero(o.value.Type().Elem())
	for i := 0; i < tail.Len(); i++ {
		tail.Index(i).Set(zero)
	}
	o.value.SetLen(size)
	o._setLen()
}

func (o *objectGoSliceReflect) putLength(v Value, throw bool) {
	newLen := int(toLength(v))
	curLen := o.value.Len()
	if newLen > curLen {
		if !o.sliceExtensible {
			o.val.runtime.typeErrorResult(throw, "Cannot extend Go slice")
			return
		}
		o.grow(newLen)
	} else if newLen < curLen {
		if !o.sliceExtensible {
			o.val.runtime.typeErrorResult(throw, "Cannot shrink Go slice")
			return
		}
		o.shrink(newLen)
	}
}

func (o *objectGoSliceReflect) setOwn(n Value, val Value, throw bool) {
	if s, ok := n.(*valueSymbol); ok {
		o.setOwnSym(s, val, throw)
		return
	}
	if idx := toIdx(n); idx >= 0 {
		o.putIdx(idx, val, throw)
		return
	}
	name := n.String()
	if name == "length" {
		o.putLength(val, throw)
		return
	}
	o.objectGoReflect.setOwnStr(name, val, throw)
}

func (o *objectGoSliceReflect) setOwnStr(name string, val Value, throw bool) {
	if idx := strToIdx(name); idx >= 0 {
		o.putIdx(idx, val, throw)
		return
	}
	if name == "length" {
		o.putLength(val, throw)
		return
	}
	o.objectGoReflect.setOwnStr(name, val, throw)
}

func (o *objectGoSliceReflect) setForeign(name Value, val, receiver Value, throw bool) bool {
	return o._setForeign(name, o.getOwnProp(name), val, receiver, throw)
}

func (o *objectGoSliceReflect) setForeignStr(name string, val, receiver Value, throw bool) bool {
	return o._setForeignStr(name, trueValIfPresent(o._hasStr(name)), val, receiver, throw)
}

func (o *objectGoSliceReflect) hasOwnProperty(n Value) bool {
	if s, ok := n.(*valueSymbol); ok {
		return o.hasSym(s)
	}
	if o._has(n) {
		return true
	}
	return o.objectGoReflect._has(n.String())
}

func (o *objectGoSliceReflect) hasOwnPropertyStr(name string) bool {
	if o._hasStr(name) {
		return true
	}
	return o.objectGoReflect._has(name)
}

func (o *objectGoSliceReflect) defineOwnProperty(name Value, descr PropertyDescriptor, throw bool) bool {
	if !o.val.runtime.checkHostObjectPropertyDescr(name, descr, throw) {
		return false
	}
	o.setOwn(name, descr.Value, throw)
	return true
}

func (o *objectGoSliceReflect) toPrimitiveNumber() Value {
	return o.toPrimitiveString()
}

func (o *objectGoSliceReflect) toPrimitiveString() Value {
	return o.val.runtime.arrayproto_join(FunctionCall{
		This: o.val,
	})
}

func (o *objectGoSliceReflect) toPrimitive() Value {
	return o.toPrimitiveString()
}

func (o *objectGoSliceReflect) deleteStr(name string, throw bool) bool {
	if idx := strToIdx(name); idx >= 0 && idx < int64(o.value.Len()) {
		o.value.Index(int(idx)).Set(reflect.Zero(o.value.Type().Elem()))
		return true
	}
	return o.objectGoReflect.deleteStr(name, throw)
}

func (o *objectGoSliceReflect) delete(name Value, throw bool) bool {
	if idx := toIdx(name); idx >= 0 && idx < int64(o.value.Len()) {
		o.value.Index(int(idx)).Set(reflect.Zero(o.value.Type().Elem()))
		return true
	}
	return o.objectGoReflect.delete(name, throw)
}

type gosliceReflectPropIter struct {
	o          *objectGoSliceReflect
	recursive  bool
	idx, limit int
}

func (i *gosliceReflectPropIter) next() (propIterItem, iterNextFunc) {
	if i.idx < i.limit && i.idx < i.o.value.Len() {
		name := strconv.Itoa(i.idx)
		i.idx++
		return propIterItem{name: name, enumerable: _ENUM_TRUE}, i.next
	}

	if i.recursive {
		return i.o.prototype.self._enumerate(i.recursive)()
	}

	return propIterItem{}, nil
}

func (o *objectGoSliceReflect) enumerate(all, recursive bool) iterNextFunc {
	return (&propFilterIter{
		wrapped: o._enumerate(recursive),
		all:     all,
		seen:    make(map[string]bool),
	}).next
}

func (o *objectGoSliceReflect) _enumerate(recursive bool) iterNextFunc {
	return (&gosliceReflectPropIter{
		o:         o,
		recursive: recursive,
		limit:     o.value.Len(),
	}).next
}

func (o *objectGoSliceReflect) equal(other objectImpl) bool {
	if other, ok := other.(*objectGoSliceReflect); ok {
		return o.value.Interface() == other.value.Interface()
	}
	return false
}

func (o *objectGoSliceReflect) sortLen() int64 {
	return int64(o.value.Len())
}

func (o *objectGoSliceReflect) sortGet(i int64) Value {
	return o.get(intToValue(i), nil)
}

func (o *objectGoSliceReflect) swap(i, j int64) {
	ii := intToValue(i)
	jj := intToValue(j)
	x := o.get(ii, nil)
	y := o.get(jj, nil)

	o.setOwn(ii, y, false)
	o.setOwn(jj, x, false)
}
