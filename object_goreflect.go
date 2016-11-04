package goja

import (
	"fmt"
	"go/ast"
	"reflect"
)

type objectGoReflect struct {
	baseObject
	origValue, value reflect.Value
}

func (o *objectGoReflect) init() {
	o.baseObject.init()
	switch o.value.Kind() {
	case reflect.Bool:
		o.class = classBoolean
		o.prototype = o.val.runtime.global.BooleanPrototype
	case reflect.String:
		o.class = classString
		o.prototype = o.val.runtime.global.StringPrototype
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:

		o.class = classNumber
		o.prototype = o.val.runtime.global.NumberPrototype
	default:
		o.class = classObject
		o.prototype = o.val.runtime.global.ObjectPrototype
	}

	o.baseObject._putProp("toString", o.val.runtime.newNativeFunc(o.toStringFunc, nil, "toString", nil, 0), true, false, true)
	o.baseObject._putProp("valueOf", o.val.runtime.newNativeFunc(o.valueOfFunc, nil, "valueOf", nil, 0), true, false, true)
}

func (o *objectGoReflect) toStringFunc(call FunctionCall) Value {
	return o.toPrimitiveString()
}

func (o *objectGoReflect) valueOfFunc(call FunctionCall) Value {
	return o.toPrimitive()
}

func (o *objectGoReflect) get(n Value) Value {
	return o.getStr(n.String())
}

func (o *objectGoReflect) _get(name string) Value {
	if o.value.Kind() == reflect.Struct {
		if v := o.value.FieldByName(name); v.IsValid() {
			return o.val.runtime.ToValue(v.Interface())
		}
	}

	if v := o.origValue.MethodByName(name); v.IsValid() {
		return o.val.runtime.ToValue(v.Interface())
	}
	return nil
}

func (o *objectGoReflect) getStr(name string) Value {
	if v := o._get(name); v != nil {
		return v
	}
	return o.baseObject.getStr(name)
}

func (o *objectGoReflect) getProp(n Value) Value {
	name := n.String()
	if p := o.getOwnProp(name); p != nil {
		return p
	}
	return o.baseObject.getProp(n)
}

func (o *objectGoReflect) getPropStr(name string) Value {
	if v := o.getOwnProp(name); v != nil {
		return v
	}
	return o.baseObject.getPropStr(name)
}

func (o *objectGoReflect) getOwnProp(name string) Value {
	if o.value.Kind() == reflect.Struct {
		if v := o.value.FieldByName(name); v.IsValid() {
			return &valueProperty{
				value:      o.val.runtime.ToValue(v.Interface()),
				writable:   true,
				enumerable: true,
			}
		}
	}

	if v := o.origValue.MethodByName(name); v.IsValid() {
		return &valueProperty{
			value:      o.val.runtime.ToValue(v.Interface()),
			enumerable: true,
		}
	}

	return nil
}

func (o *objectGoReflect) put(n Value, val Value, throw bool) {
	o.putStr(n.String(), val, throw)
}

func (o *objectGoReflect) putStr(name string, val Value, throw bool) {
	if !o._put(name, val, throw) {
		o.val.runtime.typeErrorResult(throw, "Cannot assign to property %s of a host object", name)
	}
}

func (o *objectGoReflect) _put(name string, val Value, throw bool) bool {
	if o.value.Kind() == reflect.Struct {
		if v := o.value.FieldByName(name); v.IsValid() {
			vv, err := o.val.runtime.toReflectValue(val, v.Type())
			if err != nil {
				o.val.runtime.typeErrorResult(throw, "Go struct conversion error: %v", err)
				return false
			}
			v.Set(vv)
			return true
		}
	}
	return false
}

func (o *objectGoReflect) _putProp(name string, value Value, writable, enumerable, configurable bool) Value {
	if o._put(name, value, false) {
		return value
	}
	return o.baseObject._putProp(name, value, writable, enumerable, configurable)
}

func (r *Runtime) checkHostObjectPropertyDescr(name string, descr objectImpl, throw bool) bool {
	if descr.hasPropertyStr("get") || descr.hasPropertyStr("set") {
		r.typeErrorResult(throw, "Host objects do not support accessor properties")
		return false
	}
	if wr := descr.getStr("writable"); wr != nil && !wr.ToBoolean() {
		r.typeErrorResult(throw, "Host object field %s cannot be made read-only", name)
		return false
	}
	if cfg := descr.getStr("configurable"); cfg != nil && cfg.ToBoolean() {
		r.typeErrorResult(throw, "Host object field %s cannot be made configurable", name)
		return false
	}
	return true
}

func (o *objectGoReflect) defineOwnProperty(n Value, descr objectImpl, throw bool) bool {
	name := n.String()
	if ast.IsExported(name) {
		if o.value.Kind() == reflect.Struct {
			if v := o.value.FieldByName(name); v.IsValid() {
				if !o.val.runtime.checkHostObjectPropertyDescr(name, descr, throw) {
					return false
				}
				val := descr.getStr("value")
				if val == nil {
					val = _undefined
				}
				vv, err := o.val.runtime.toReflectValue(val, v.Type())
				if err != nil {
					o.val.runtime.typeErrorResult(throw, "Go struct conversion error: %v", err)
					return false
				}
				v.Set(vv)
				return true
			}
		}
	}

	return o.baseObject.defineOwnProperty(n, descr, throw)
}

func (o *objectGoReflect) _has(name string) bool {
	if !ast.IsExported(name) {
		return false
	}
	if o.value.Kind() == reflect.Struct {
		if v := o.value.FieldByName(name); v.IsValid() {
			return true
		}
	}
	if v := o.origValue.MethodByName(name); v.IsValid() {
		return true
	}
	return false
}

func (o *objectGoReflect) hasProperty(n Value) bool {
	name := n.String()
	if o._has(name) {
		return true
	}
	return o.baseObject.hasProperty(n)
}

func (o *objectGoReflect) hasPropertyStr(name string) bool {
	if o._has(name) {
		return true
	}
	return o.baseObject.hasPropertyStr(name)
}

func (o *objectGoReflect) hasOwnProperty(n Value) bool {
	return o._has(n.String())
}

func (o *objectGoReflect) hasOwnPropertyStr(name string) bool {
	return o._has(name)
}

func (o *objectGoReflect) _toNumber() Value {
	switch o.value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return intToValue(o.value.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return intToValue(int64(o.value.Uint()))
	case reflect.Bool:
		if o.value.Bool() {
			return intToValue(1)
		} else {
			return intToValue(0)
		}
	case reflect.Float32, reflect.Float64:
		return floatToValue(o.value.Float())
	}
	return nil
}

func (o *objectGoReflect) _toString() Value {
	switch o.value.Kind() {
	case reflect.String:
		return newStringValue(o.value.String())
	case reflect.Bool:
		if o.value.Interface().(bool) {
			return stringTrue
		} else {
			return stringFalse
		}
	}
	switch v := o.value.Interface().(type) {
	case fmt.Stringer:
		return newStringValue(v.String())
	}
	return stringObjectObject
}

func (o *objectGoReflect) toPrimitiveNumber() Value {
	if v := o._toNumber(); v != nil {
		return v
	}
	return o._toString()
}

func (o *objectGoReflect) toPrimitiveString() Value {
	if v := o._toNumber(); v != nil {
		return v.ToString()
	}
	return o._toString()
}

func (o *objectGoReflect) toPrimitive() Value {
	if o.prototype == o.val.runtime.global.NumberPrototype {
		return o.toPrimitiveNumber()
	}
	return o.toPrimitiveString()
}

func (o *objectGoReflect) deleteStr(name string, throw bool) bool {
	if o._has(name) {
		o.val.runtime.typeErrorResult(throw, "Cannot delete property %s from a Go type")
		return false
	}
	return o.baseObject.deleteStr(name, throw)
}

func (o *objectGoReflect) delete(name Value, throw bool) bool {
	return o.deleteStr(name.String(), throw)
}

type goreflectPropIter struct {
	o         *objectGoReflect
	idx       int
	recursive bool
}

func (i *goreflectPropIter) nextField() (propIterItem, iterNextFunc) {
	l := i.o.value.NumField()
	for i.idx < l {
		name := i.o.value.Type().Field(i.idx).Name
		i.idx++
		if ast.IsExported(name) {
			return propIterItem{name: name, enumerable: _ENUM_TRUE}, i.nextField
		}
	}

	i.idx = 0
	return i.nextMethod()
}

func (i *goreflectPropIter) nextMethod() (propIterItem, iterNextFunc) {
	l := i.o.origValue.NumMethod()
	for i.idx < l {
		name := i.o.origValue.Type().Method(i.idx).Name
		i.idx++
		if ast.IsExported(name) {
			return propIterItem{name: name, enumerable: _ENUM_TRUE}, i.nextMethod
		}
	}

	return i.o.baseObject._enumerate(i.recursive)()
}

func (o *objectGoReflect) _enumerate(recusrive bool) iterNextFunc {
	r := &goreflectPropIter{
		o:         o,
		recursive: recusrive,
	}
	if o.value.Kind() == reflect.Struct {
		return r.nextField
	}
	return r.nextMethod
}

func (o *objectGoReflect) enumerate(all, recursive bool) iterNextFunc {
	return (&propFilterIter{
		wrapped: o._enumerate(recursive),
		all:     all,
		seen:    make(map[string]bool),
	}).next
}

func (o *objectGoReflect) export() interface{} {
	return o.origValue.Interface()
}

func (o *objectGoReflect) exportType() reflect.Type {
	return o.origValue.Type()
}

func (o *objectGoReflect) equal(other objectImpl) bool {
	if other, ok := other.(*objectGoReflect); ok {
		return o.value.Interface() == other.value.Interface()
	}
	return false
}
