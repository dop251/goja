package goja

import (
	"reflect"
	"strconv"
)

type objectGoSlice struct {
	baseObject
	data            *[]interface{}
	lengthProp      valueProperty
	sliceExtensible bool
}

func (o *objectGoSlice) init() {
	o.baseObject.init()
	o.class = classArray
	o.prototype = o.val.runtime.global.ArrayPrototype
	o.lengthProp.writable = o.sliceExtensible
	o.extensible = true
	o.updateLen()
	o.baseObject._put("length", &o.lengthProp)
}

func (o *objectGoSlice) updateLen() {
	o.lengthProp.value = intToValue(int64(len(*o.data)))
}

func (o *objectGoSlice) getIdx(idx int64) Value {
	if idx < int64(len(*o.data)) {
		v := (*o.data)[idx]
		if v == nil {
			return nil
		}
		return o.val.runtime.ToValue(v)
	}
	return nil
}

func (o *objectGoSlice) _get(n Value) Value {
	if idx := toIdx(n); idx >= 0 {
		return o.getIdx(idx)
	}
	return nil
}

func (o *objectGoSlice) _getStr(name string) Value {
	if idx := strToIdx(name); idx >= 0 {
		return o.getIdx(idx)
	}
	return nil
}

func (o *objectGoSlice) get(n Value, receiver Value) Value {
	if s, ok := n.(*valueSymbol); ok {
		return o.getSym(s, receiver)
	}
	if v := o._get(n); v != nil {
		return v
	}
	return o.baseObject.get(n, receiver)
}

func (o *objectGoSlice) getStr(name string, receiver Value) Value {
	if v := o._getStr(name); v != nil {
		return v
	}
	return o.baseObject.getStr(name, receiver)
}

func (o *objectGoSlice) getOwnPropStr(name string) Value {
	if v := o._getStr(name); v != nil {
		return &valueProperty{
			value:      v,
			writable:   true,
			enumerable: true,
		}
	}
	return o.baseObject.getOwnPropStr(name)
}

func (o *objectGoSlice) getOwnProp(name Value) Value {
	if v := o._get(name); v != nil {
		return &valueProperty{
			value:      v,
			writable:   true,
			enumerable: true,
		}
	}

	return o.baseObject.getOwnProp(name)
}

func (o *objectGoSlice) grow(size int64) {
	newcap := int64(cap(*o.data))
	if newcap < size {
		// Use the same algorithm as in runtime.growSlice
		doublecap := newcap + newcap
		if size > doublecap {
			newcap = size
		} else {
			if len(*o.data) < 1024 {
				newcap = doublecap
			} else {
				for newcap < size {
					newcap += newcap / 4
				}
			}
		}

		n := make([]interface{}, size, newcap)
		copy(n, *o.data)
		*o.data = n
	} else {
		tail := (*o.data)[len(*o.data):size]
		for k := range tail {
			tail[k] = nil
		}
		*o.data = (*o.data)[:size]
	}
	o.updateLen()
}

func (o *objectGoSlice) shrink(size int64) {
	tail := (*o.data)[size:]
	for k := range tail {
		tail[k] = nil
	}
	*o.data = (*o.data)[:size]
	o.updateLen()
}

func (o *objectGoSlice) putIdx(idx int64, v Value, throw bool) {
	if idx >= int64(len(*o.data)) {
		if !o.sliceExtensible {
			o.val.runtime.typeErrorResult(throw, "Cannot extend Go slice")
			return
		}
		o.grow(idx + 1)
	}
	(*o.data)[idx] = v.Export()
}

func (o *objectGoSlice) putLength(v Value, throw bool) {
	newLen := toLength(v)
	curLen := int64(len(*o.data))
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

func (o *objectGoSlice) setOwn(n Value, val Value, throw bool) {
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
	if !o._setForeignStr(name, nil, val, o.val, throw) {
		o.val.runtime.typeErrorResult(throw, "Can't set property '%s' on Go slice", name)
	}
}

func (o *objectGoSlice) setOwnStr(name string, val Value, throw bool) {
	if idx := strToIdx(name); idx >= 0 {
		o.putIdx(idx, val, throw)
		return
	}
	if name == "length" {
		o.putLength(val, throw)
		return
	}
	if !o._setForeignStr(name, nil, val, o.val, throw) {
		o.val.runtime.typeErrorResult(throw, "Can't set property '%s' on Go slice", name)
	}
}

func (o *objectGoSlice) setForeign(name Value, val, receiver Value, throw bool) bool {
	return o._setForeign(name, nil, val, receiver, throw)
}

func (o *objectGoSlice) setForeignStr(name string, val, receiver Value, throw bool) bool {
	return o._setForeignStr(name, nil, val, receiver, throw)
}

func (o *objectGoSlice) _has(n Value) bool {
	if idx := toIdx(n); idx >= 0 {
		return idx < int64(len(*o.data))
	}
	return false
}

func (o *objectGoSlice) _hasStr(name string) bool {
	if idx := strToIdx(name); idx >= 0 {
		return idx < int64(len(*o.data))
	}
	return false
}

func (o *objectGoSlice) hasOwnProperty(n Value) bool {
	if s, ok := n.(*valueSymbol); ok {
		return o.hasSym(s)
	}
	return o._has(n)
}

func (o *objectGoSlice) hasOwnPropertyStr(name string) bool {
	return o._hasStr(name)
}

func (o *objectGoSlice) defineOwnProperty(n Value, descr PropertyDescriptor, throw bool) bool {
	if s, ok := n.(*valueSymbol); ok {
		return o.defineOwnPropertySym(s, descr, throw)
	}
	if idx := toIdx(n); idx >= 0 {
		if !o.val.runtime.checkHostObjectPropertyDescr(n, descr, throw) {
			return false
		}
		val := descr.Value
		if val == nil {
			val = _undefined
		}
		o.putIdx(idx, val, throw)
		return true
	}
	o.val.runtime.typeErrorResult(throw, "Cannot define property '%s' on a Go slice", n.String())
	return false
}

func (o *objectGoSlice) toPrimitiveNumber() Value {
	return o.toPrimitiveString()
}

func (o *objectGoSlice) toPrimitiveString() Value {
	return o.val.runtime.arrayproto_join(FunctionCall{
		This: o.val,
	})
}

func (o *objectGoSlice) toPrimitive() Value {
	return o.toPrimitiveString()
}

func (o *objectGoSlice) deleteStr(name string, throw bool) bool {
	if idx := strToIdx(name); idx >= 0 && idx < int64(len(*o.data)) {
		(*o.data)[idx] = nil
		return true
	}
	return o.baseObject.deleteStr(name, throw)
}

func (o *objectGoSlice) delete(n Value, throw bool) bool {
	if s, ok := n.(*valueSymbol); ok {
		return o.deleteSym(s, throw)
	}
	if idx := toIdx(n); idx >= 0 && idx < int64(len(*o.data)) {
		(*o.data)[idx] = nil
		return true
	}
	return o.baseObject.deleteStr(n.String(), throw)
}

type goslicePropIter struct {
	o          *objectGoSlice
	recursive  bool
	idx, limit int
}

func (i *goslicePropIter) next() (propIterItem, iterNextFunc) {
	if i.idx < i.limit && i.idx < len(*i.o.data) {
		name := strconv.Itoa(i.idx)
		i.idx++
		return propIterItem{name: name, enumerable: _ENUM_TRUE}, i.next
	}

	if i.recursive {
		return i.o.prototype.self._enumerate(i.recursive)()
	}

	return propIterItem{}, nil
}

func (o *objectGoSlice) enumerate(all, recursive bool) iterNextFunc {
	return (&propFilterIter{
		wrapped: o._enumerate(recursive),
		all:     all,
		seen:    make(map[string]bool),
	}).next

}

func (o *objectGoSlice) _enumerate(recursive bool) iterNextFunc {
	return (&goslicePropIter{
		o:         o,
		recursive: recursive,
		limit:     len(*o.data),
	}).next
}

func (o *objectGoSlice) export() interface{} {
	return *o.data
}

func (o *objectGoSlice) exportType() reflect.Type {
	return reflectTypeArray
}

func (o *objectGoSlice) equal(other objectImpl) bool {
	if other, ok := other.(*objectGoSlice); ok {
		return o.data == other.data
	}
	return false
}

func (o *objectGoSlice) sortLen() int64 {
	return int64(len(*o.data))
}

func (o *objectGoSlice) sortGet(i int64) Value {
	return o.get(intToValue(i), nil)
}

func (o *objectGoSlice) swap(i, j int64) {
	ii := intToValue(i)
	jj := intToValue(j)
	x := o.get(ii, nil)
	y := o.get(jj, nil)

	o.setOwn(ii, y, false)
	o.setOwn(jj, x, false)
}
