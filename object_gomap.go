package goja

import (
	"reflect"
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

func (o *objectGoMapSimple) _getStr(name string) Value {
	v, exists := o.data[name]
	if !exists {
		return nil
	}
	return o.val.runtime.ToValue(v)
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
	return nil
}

func (o *objectGoMapSimple) setOwnStr(name string, val Value, throw bool) bool {
	if _, exists := o.data[name]; exists {
		o.data[name] = val.Export()
		return true
	}
	if proto := o.prototype; proto != nil {
		// we know it's foreign because prototype loops are not allowed
		if res, ok := proto.self.setForeignStr(name, val, o.val, throw); ok {
			return res
		}
	}
	// new property
	if !o.extensible {
		o.val.runtime.typeErrorResult(throw, "Cannot add property %s, object is not extensible", name)
		return false
	} else {
		o.data[name] = val.Export()
	}
	return true
}

func trueValIfPresent(present bool) Value {
	if present {
		return valueTrue
	}
	return nil
}

func (o *objectGoMapSimple) setForeignStr(name string, val, receiver Value, throw bool) (bool, bool) {
	return o._setForeignStr(name, trueValIfPresent(o._hasStr(name)), val, receiver, throw)
}

func (o *objectGoMapSimple) _hasStr(name string) bool {
	_, exists := o.data[name]
	return exists
}

func (o *objectGoMapSimple) hasOwnPropertyStr(name string) bool {
	return o._hasStr(name)
}

func (o *objectGoMapSimple) defineOwnPropertyStr(name string, descr PropertyDescriptor, throw bool) bool {
	if !o.val.runtime.checkHostObjectPropertyDescr(name, descr, throw) {
		return false
	}

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
