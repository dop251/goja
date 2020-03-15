package goja

import (
	"reflect"
	"strconv"
)

type objectGoMapSimple struct {
	baseObject
	data map[string]interface{}
}

func (o *objectGoMapSimple) init() {
	o.baseObject.init()
	o.prototype = o.val.runtime.global.ObjectPrototype
	o.class = classObject
	o.extensible = true
}

func (o *objectGoMapSimple) _get(n Value) Value {
	return o._getStr(n.String())
}

func (o *objectGoMapSimple) _getStr(name string) Value {
	v, exists := o.data[name]
	if !exists {
		return nil
	}
	return o.val.runtime.ToValue(v)
}

func (o *objectGoMapSimple) get(p Value, receiver Value) Value {
	return o.getWithOwnProp(o.getOwnProp(p), p, receiver)
}

func (o *objectGoMapSimple) getStr(name string, receiver Value) Value {
	if v := o._getStr(name); v != nil {
		return v
	}
	return o.baseObject.getStr(name, receiver)
}

func (o *objectGoMapSimple) getOwnPropStr(name string) Value {
	if v := o._getStr(name); v != nil {
		return v
	}
	return o.baseObject.getOwnPropStr(name)
}

func (o *objectGoMapSimple) getOwnProp(n Value) Value {
	if s, ok := n.(*valueSymbol); ok {
		return o.getOwnPropSym(s)
	}

	return o.getOwnPropStr(n.String())
}

func (o *objectGoMapSimple) setOwn(n Value, val Value, throw bool) {
	if s, ok := n.(*valueSymbol); ok {
		o.setOwnSym(s, val, throw)
		return
	}

	o.setOwnStr(n.String(), val, throw)
}

func (o *objectGoMapSimple) setOwnStr(name string, val Value, throw bool) {
	if _, exists := o.data[name]; exists {
		o.data[name] = val.Export()
		return
	}
	if name == __proto__ {
		o._setProto(val)
		return
	}
	if proto := o.prototype; proto != nil {
		// we know it's foreign because prototype loops are not allowed
		if proto.self.setForeignStr(name, val, o.val, throw) {
			return
		}
	}
	// new property
	if !o.extensible {
		o.val.runtime.typeErrorResult(throw, "Cannot add property %s, object is not extensible", name)
	} else {
		o.data[name] = val.Export()
	}
}

func trueValIfPresent(present bool) Value {
	if present {
		return valueTrue
	}
	return nil
}

func (o *objectGoMapSimple) setForeign(name Value, val, receiver Value, throw bool) bool {
	return o._setForeign(name, o.getOwnProp(name), val, receiver, throw)
}

func (o *objectGoMapSimple) setForeignStr(name string, val, receiver Value, throw bool) bool {
	return o._setForeignStr(name, trueValIfPresent(o._hasStr(name)), val, receiver, throw)
}

func (o *objectGoMapSimple) _hasStr(name string) bool {
	_, exists := o.data[name]
	return exists
}

func (o *objectGoMapSimple) _has(n Value) bool {
	if s, ok := n.(*valueSymbol); ok {
		return o.hasOwnSym(s)
	}
	return o._hasStr(n.String())
}

func (o *objectGoMapSimple) hasOwnProperty(n Value) bool {
	if s, ok := n.(*valueSymbol); ok {
		return o.hasOwnSym(s)
	}
	return o._has(n)
}

func (o *objectGoMapSimple) hasOwnPropertyStr(name string) bool {
	return o._hasStr(name)
}

func (o *objectGoMapSimple) defineOwnProperty(n Value, descr PropertyDescriptor, throw bool) bool {
	if s, ok := n.(*valueSymbol); ok {
		return o.defineOwnPropertySym(s, descr, throw)
	}
	if !o.val.runtime.checkHostObjectPropertyDescr(n, descr, throw) {
		return false
	}

	name := n.String()
	if o.extensible || o._hasStr(name) {
		o.data[name] = descr.Value.Export()
		return true
	}

	o.val.runtime.typeErrorResult(throw, "Cannot define property %s, object is not extensible", name)
	return false
}

/*
func (o *objectGoMapSimple) toPrimitiveNumber() Value {
	return o.toPrimitiveString()
}

func (o *objectGoMapSimple) toPrimitiveString() Value {
	return stringObjectObject
}

func (o *objectGoMapSimple) toPrimitive() Value {
	return o.toPrimitiveString()
}

func (o *objectGoMapSimple) assertCallable() (call func(FunctionCall) Value, ok bool) {
	return nil, false
}
*/

func (o *objectGoMapSimple) deleteStr(name string, _ bool) bool {
	delete(o.data, name)
	return true
}

func (o *objectGoMapSimple) delete(n Value, throw bool) bool {
	if s, ok := n.(*valueSymbol); ok {
		return o.deleteSym(s, throw)
	}

	return o.deleteStr(n.String(), throw)
}

type gomapPropIter struct {
	o         *objectGoMapSimple
	propNames []string
	idx       int
}

func (i *gomapPropIter) next() (propIterItem, iterNextFunc) {
	for i.idx < len(i.propNames) {
		name := i.propNames[i.idx]
		i.idx++
		if _, exists := i.o.data[name]; exists {
			return propIterItem{name: name, enumerable: _ENUM_TRUE}, i.next
		}
	}

	return propIterItem{}, nil
}

func (o *objectGoMapSimple) enumerateUnfiltered() iterNextFunc {
	propNames := make([]string, len(o.data))
	i := 0
	for key := range o.data {
		propNames[i] = key
		i++
	}

	return o.recursiveIter((&gomapPropIter{
		o:         o,
		propNames: propNames,
	}).next)
}

func (o *objectGoMapSimple) ownKeys(_ bool, accum []Value) []Value {
	// all own keys are enumerable
	for key := range o.data {
		accum = append(accum, newStringValue(key))
	}
	return accum
}

func (o *objectGoMapSimple) export() interface{} {
	return o.data
}

func (o *objectGoMapSimple) exportType() reflect.Type {
	return reflectTypeMap
}

func (o *objectGoMapSimple) equal(other objectImpl) bool {
	if other, ok := other.(*objectGoMapSimple); ok {
		return o == other
	}
	return false
}

func (o *objectGoMapSimple) sortLen() int64 {
	return int64(len(o.data))
}

func (o *objectGoMapSimple) sortGet(i int64) Value {
	return o.getStr(strconv.FormatInt(i, 10), nil)
}

func (o *objectGoMapSimple) swap(i, j int64) {
	ii := strconv.FormatInt(i, 10)
	jj := strconv.FormatInt(j, 10)
	x := o.getStr(ii, nil)
	y := o.getStr(jj, nil)

	o.setOwnStr(ii, y, false)
	o.setOwnStr(jj, x, false)
}
