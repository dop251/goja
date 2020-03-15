package goja

import (
	"fmt"
	"reflect"
	"runtime"
	"unsafe"
)

const (
	classObject   = "Object"
	classArray    = "Array"
	classWeakSet  = "WeakSet"
	classWeakMap  = "WeakMap"
	classMap      = "Map"
	classSet      = "Set"
	classFunction = "Function"
	classNumber   = "Number"
	classString   = "String"
	classBoolean  = "Boolean"
	classError    = "Error"
	classRegExp   = "RegExp"
	classDate     = "Date"
	classProxy    = "Proxy"

	classArrayIterator = "Array Iterator"
	classMapIterator   = "Map Iterator"
	classSetIterator   = "Set Iterator"
)

type weakCollection interface {
	removePtr(uintptr)
}

type weakCollections struct {
	colls []weakCollection
}

func (r *weakCollections) add(c weakCollection) {
	for _, ec := range r.colls {
		if ec == c {
			return
		}
	}
	r.colls = append(r.colls, c)
}

func (r *weakCollections) id() uintptr {
	return uintptr(unsafe.Pointer(r))
}

func (r *weakCollections) remove(c weakCollection) {
	if cap(r.colls) > 16 && cap(r.colls)>>2 > len(r.colls) {
		// shrink
		colls := make([]weakCollection, 0, len(r.colls))
		for _, coll := range r.colls {
			if coll != c {
				colls = append(colls, coll)
			}
		}
		r.colls = colls
	} else {
		for i, coll := range r.colls {
			if coll == c {
				l := len(r.colls) - 1
				r.colls[i] = r.colls[l]
				r.colls[l] = nil
				r.colls = r.colls[:l]
				break
			}
		}
	}
}

func finalizeObjectWeakRefs(r *weakCollections) {
	id := r.id()
	for _, c := range r.colls {
		c.removePtr(id)
	}
	r.colls = nil
}

type Object struct {
	runtime *Runtime
	self    objectImpl

	// Contains references to all weak collections that contain this Object.
	// weakColls has a finalizer that removes the Object's id from all weak collections.
	// The id is the weakColls pointer value converted to uintptr.
	// Note, cannot set the finalizer on the *Object itself because it's a part of a
	// reference cycle.
	weakColls *weakCollections
}

type iterNextFunc func() (propIterItem, iterNextFunc)

type PropertyDescriptor struct {
	jsDescriptor *Object

	Value Value

	Writable, Configurable, Enumerable Flag

	Getter, Setter Value
}

func (p PropertyDescriptor) toValue(r *Runtime) Value {
	if p.jsDescriptor != nil {
		return p.jsDescriptor
	}

	o := r.NewObject()
	s := o.self

	s._putProp("value", p.Value, false, false, false)

	s._putProp("writable", valueBool(p.Writable.Bool()), false, false, false)
	s._putProp("enumerable", valueBool(p.Enumerable.Bool()), false, false, false)
	s._putProp("configurable", valueBool(p.Configurable.Bool()), false, false, false)

	s._putProp("get", p.Getter, false, false, false)
	s._putProp("set", p.Setter, false, false, false)

	s.preventExtensions(false)

	return o
}

type objectImpl interface {
	sortable
	className() string
	get(p, receiver Value) Value
	getStr(p string, receiver Value) Value
	getOwnProp(Value) Value
	getOwnPropStr(string) Value
	setOwn(p Value, v Value, throw bool)
	setForeign(p Value, v, receiver Value, throw bool) bool
	setOwnStr(p string, v Value, throw bool)
	setForeignStr(p string, v, receiver Value, throw bool) bool
	setOwnSym(p *valueSymbol, v Value, throw bool)
	setForeignSym(p *valueSymbol, v, receiver Value, throw bool) bool
	hasProperty(Value) bool
	hasPropertyStr(string) bool
	hasOwnProperty(Value) bool
	hasOwnPropertyStr(string) bool
	_putProp(name string, value Value, writable, enumerable, configurable bool) Value
	_putSym(s *valueSymbol, prop Value)
	defineOwnProperty(name Value, descr PropertyDescriptor, throw bool) bool
	toPrimitiveNumber() Value
	toPrimitiveString() Value
	toPrimitive() Value
	assertCallable() (call func(FunctionCall) Value, ok bool)
	deleteStr(name string, throw bool) bool
	delete(name Value, throw bool) bool
	proto() *Object
	setProto(proto *Object, throw bool) bool
	hasInstance(v Value) bool
	isExtensible() bool
	preventExtensions(throw bool) bool
	enumerate() iterNextFunc
	enumerateUnfiltered() iterNextFunc
	export() interface{}
	exportType() reflect.Type
	equal(objectImpl) bool
	ownKeys(all bool, accum []Value) []Value
	ownSymbols() []Value
}

type baseObject struct {
	class      string
	val        *Object
	prototype  *Object
	extensible bool

	values    map[string]Value
	propNames []string

	symValues map[*valueSymbol]Value
}

type primitiveValueObject struct {
	baseObject
	pValue Value
}

func (o *primitiveValueObject) export() interface{} {
	return o.pValue.Export()
}

func (o *primitiveValueObject) exportType() reflect.Type {
	return o.pValue.ExportType()
}

type FunctionCall struct {
	This      Value
	Arguments []Value
}

type ConstructorCall struct {
	This      *Object
	Arguments []Value
}

func (f FunctionCall) Argument(idx int) Value {
	if idx < len(f.Arguments) {
		return f.Arguments[idx]
	}
	return _undefined
}

func (f ConstructorCall) Argument(idx int) Value {
	if idx < len(f.Arguments) {
		return f.Arguments[idx]
	}
	return _undefined
}

func (o *baseObject) init() {
	o.values = make(map[string]Value)
}

func (o *baseObject) className() string {
	return o.class
}

func (o *baseObject) hasProperty(n Value) bool {
	if o.val.self.hasOwnProperty(n) {
		return true
	}
	if o.prototype != nil {
		return o.prototype.self.hasProperty(n)
	}
	return false
}

func (o *baseObject) hasPropertyStr(name string) bool {
	if o.val.self.hasOwnPropertyStr(name) {
		return true
	}
	if o.prototype != nil {
		return o.prototype.self.hasPropertyStr(name)
	}
	return false
}

func (o *baseObject) getOwnPropSym(s *valueSymbol) Value {
	return o.symValues[s]
}

func (o *baseObject) getWithOwnProp(prop, p, receiver Value) Value {
	if prop == nil && o.prototype != nil {
		if receiver == nil {
			return o.prototype.self.get(p, o.val)
		}
		return o.prototype.self.get(p, receiver)
	}
	if prop, ok := prop.(*valueProperty); ok {
		if receiver == nil {
			return prop.get(o.val)
		}
		return prop.get(receiver)
	}
	return prop
}

func (o *baseObject) getStrWithOwnProp(prop Value, name string, receiver Value) Value {
	if prop == nil && o.prototype != nil {
		if receiver == nil {
			return o.prototype.self.getStr(name, o.val)
		}
		return o.prototype.self.getStr(name, receiver)
	}
	if prop, ok := prop.(*valueProperty); ok {
		if receiver == nil {
			return prop.get(o.val)
		}
		return prop.get(receiver)
	}
	return prop
}

func (o *baseObject) get(p Value, receiver Value) Value {
	if s, ok := p.(*valueSymbol); ok {
		return o.getSym(s, receiver)
	}
	return o.getStr(p.String(), receiver)
}

func (o *baseObject) getSym(s *valueSymbol, receiver Value) Value {
	return o.getWithOwnProp(o.symValues[s], s, receiver)
}

func (o *baseObject) getStr(name string, receiver Value) Value {
	prop := o.values[name]
	if prop == nil {
		if name == __proto__ {
			return o.prototype
		}
		if o.prototype != nil {
			if receiver == nil {
				return o.prototype.self.getStr(name, o.val)
			}
			return o.prototype.self.getStr(name, receiver)
		}
	}
	if prop, ok := prop.(*valueProperty); ok {
		if receiver == nil {
			return prop.get(o.val)
		}
		return prop.get(receiver)
	}
	return prop
}

func (o *baseObject) getOwnPropStr(name string) Value {
	v := o.values[name]
	if v == nil && name == __proto__ {
		return o.prototype
	}
	return v
}

func (o *baseObject) getOwnProp(name Value) Value {
	if s, ok := name.(*valueSymbol); ok {
		return o.symValues[s]
	}

	return o.getOwnPropStr(name.String())
}

func (o *baseObject) checkDeleteProp(name string, prop *valueProperty, throw bool) bool {
	if !prop.configurable {
		o.val.runtime.typeErrorResult(throw, "Cannot delete property '%s' of %s", name, o.val.toString())
		return false
	}
	return true
}

func (o *baseObject) checkDelete(name string, val Value, throw bool) bool {
	if val, ok := val.(*valueProperty); ok {
		return o.checkDeleteProp(name, val, throw)
	}
	return true
}

func (o *baseObject) _delete(name string) {
	delete(o.values, name)
	for i, n := range o.propNames {
		if n == name {
			copy(o.propNames[i:], o.propNames[i+1:])
			o.propNames = o.propNames[:len(o.propNames)-1]
			break
		}
	}
}

func (o *baseObject) deleteStr(name string, throw bool) bool {
	if val, exists := o.values[name]; exists {
		if !o.checkDelete(name, val, throw) {
			return false
		}
		o._delete(name)
	}
	return true
}

func (o *baseObject) deleteSym(s *valueSymbol, throw bool) bool {
	if val, exists := o.symValues[s]; exists {
		if !o.checkDelete(s.String(), val, throw) {
			return false
		}
		delete(o.symValues, s)
	}
	return true
}

func (o *baseObject) delete(n Value, throw bool) bool {
	if s, ok := n.(*valueSymbol); ok {
		return o.deleteSym(s, throw)
	}
	return o.deleteStr(n.String(), throw)
}

func (o *baseObject) setProto(proto *Object, throw bool) bool {
	current := o.prototype
	if current.SameAs(proto) {
		return true
	}
	if !o.extensible {
		o.val.runtime.typeErrorResult(throw, "%s is not extensible", o.val)
		return false
	}
	for p := proto; p != nil; {
		if p.SameAs(o.val) {
			o.val.runtime.typeErrorResult(throw, "Cyclic __proto__ value")
			return false
		}
		p = p.self.proto()
	}
	o.prototype = proto
	return true
}

func (o *baseObject) setOwn(name Value, val Value, throw bool) {
	if s, ok := name.(*valueSymbol); ok {
		o.setOwnSym(s, val, throw)
	} else {
		o.val.self.setOwnStr(name.String(), val, throw)
	}
}

func (o *baseObject) setForeign(name Value, val, receiver Value, throw bool) bool {
	if s, ok := name.(*valueSymbol); ok {
		return o.setForeignSym(s, val, receiver, throw)
	} else {
		return o.setForeignStr(name.String(), val, receiver, throw)
	}
}

func (o *baseObject) _setProto(val Value) {
	var proto *Object
	if val != _null {
		if obj, ok := val.(*Object); ok {
			proto = obj
		} else {
			return
		}
	}
	o.setProto(proto, true)
}

func (o *baseObject) setOwnStr(name string, val Value, throw bool) {
	ownDesc := o.values[name]
	if ownDesc == nil {
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
			o.values[name] = val
			o.propNames = append(o.propNames, name)
		}
		return
	}
	if prop, ok := ownDesc.(*valueProperty); ok {
		if !prop.isWritable() {
			o.val.runtime.typeErrorResult(throw, "Cannot assign to read only property '%s'", name)
			return
		} else {
			prop.set(o.val, val)
		}
	} else {
		o.values[name] = val
	}
}

func (o *baseObject) setOwnSym(name *valueSymbol, val Value, throw bool) {
	ownDesc := o.symValues[name]
	if ownDesc == nil {
		if proto := o.prototype; proto != nil {
			// we know it's foreign because prototype loops are not allowed
			if proto.self.setForeignSym(name, val, o.val, throw) {
				return
			}
		}
		// new property
		if !o.extensible {
			o.val.runtime.typeErrorResult(throw, "Cannot add property %s, object is not extensible", name)
		} else {
			if o.symValues == nil {
				o.symValues = make(map[*valueSymbol]Value, 1)
			}
			o.symValues[name] = val
		}
		return
	}
	if prop, ok := ownDesc.(*valueProperty); ok {
		if !prop.isWritable() {
			o.val.runtime.typeErrorResult(throw, "Cannot assign to read only property '%s'", name)
			return
		} else {
			prop.set(o.val, val)
		}
	} else {
		o.symValues[name] = val
	}
}

func (o *baseObject) _setForeign(name Value, prop, val, receiver Value, throw bool) bool {
	if prop != nil {
		if prop, ok := prop.(*valueProperty); ok {
			if !prop.isWritable() {
				o.val.runtime.typeErrorResult(throw, "Cannot assign to read only property '%s'", name)
				return true
			}
			if prop.setterFunc != nil {
				prop.set(receiver, val)
				return true
			}
		}
	} else {
		if proto := o.prototype; proto != nil {
			if receiver != proto {
				return proto.self.setForeign(name, val, receiver, throw)
			}
			proto.self.setOwn(name, val, throw)
			return true
		}
	}
	return false
}

func (o *baseObject) _setForeignStr(name string, prop, val, receiver Value, throw bool) bool {
	if prop != nil {
		if prop, ok := prop.(*valueProperty); ok {
			if !prop.isWritable() {
				o.val.runtime.typeErrorResult(throw, "Cannot assign to read only property '%s'", name)
				return true
			}
			if prop.setterFunc != nil {
				prop.set(receiver, val)
				return true
			}
		}
	} else {
		if proto := o.prototype; proto != nil {
			if receiver != proto {
				return proto.self.setForeignStr(name, val, receiver, throw)
			}
			proto.self.setOwnStr(name, val, throw)
			return true
		}
	}
	return false
}

func (o *baseObject) setForeignStr(name string, val, receiver Value, throw bool) bool {
	return o._setForeignStr(name, o.values[name], val, receiver, throw)
}

func (o *baseObject) setForeignSym(name *valueSymbol, val, receiver Value, throw bool) bool {
	prop := o.symValues[name]
	if prop != nil {
		if prop, ok := prop.(*valueProperty); ok {
			if !prop.isWritable() {
				o.val.runtime.typeErrorResult(throw, "Cannot assign to read only property '%s'", name)
				return true
			}
			if prop.setterFunc != nil {
				prop.set(receiver, val)
				return true
			}
		}
	} else {
		if proto := o.prototype; proto != nil {
			if receiver != o.val {
				return proto.self.setForeignSym(name, val, receiver, throw)
			}
			proto.self.setOwnSym(name, val, throw)
			return true
		}
	}
	return false
}

func (o *Object) setStr(name string, val, receiver Value, throw bool) {
	if receiver == o {
		o.self.setOwnStr(name, val, throw)
	} else {
		if !o.self.setForeignStr(name, val, receiver, throw) {
			if robj, ok := receiver.(*Object); ok {
				if prop := robj.self.getOwnPropStr(name); prop != nil {
					if desc, ok := prop.(*valueProperty); ok {
						if desc.accessor {
							o.runtime.typeErrorResult(throw, "Receiver property %s is an accessor", name)
							return
						}
						if !desc.writable {
							o.runtime.typeErrorResult(throw, "Cannot assign to read only property '%s'", name)
							return
						}
					}
					robj.self.defineOwnProperty(newStringValue(name), PropertyDescriptor{Value: val}, throw)
				} else {
					robj.self.defineOwnProperty(newStringValue(name), PropertyDescriptor{
						Value:        val,
						Writable:     FLAG_TRUE,
						Configurable: FLAG_TRUE,
						Enumerable:   FLAG_TRUE,
					}, throw)
				}
			} else {
				o.runtime.typeErrorResult(throw, "Receiver is not an object: %v", receiver)
			}
		}
	}
}

func (o *Object) set(name Value, val, receiver Value, throw bool) {
	if receiver == o {
		o.self.setOwn(name, val, throw)
	} else {
		if !o.self.setForeign(name, val, receiver, throw) {
			if robj, ok := receiver.(*Object); ok {
				if prop := robj.self.getOwnProp(name); prop != nil {
					if desc, ok := prop.(*valueProperty); ok {
						if desc.accessor {
							o.runtime.typeErrorResult(throw, "Receiver property %s is an accessor", name)
							return
						}
						if !desc.writable {
							o.runtime.typeErrorResult(throw, "Cannot assign to read only property '%s'", name)
							return
						}
					}
					robj.self.defineOwnProperty(name, PropertyDescriptor{Value: val}, throw)
				} else {
					robj.self.defineOwnProperty(name, PropertyDescriptor{
						Value:        val,
						Writable:     FLAG_TRUE,
						Configurable: FLAG_TRUE,
						Enumerable:   FLAG_TRUE,
					}, throw)
				}
			} else {
				o.runtime.typeErrorResult(throw, "Receiver is not an object: %v", receiver)
			}
		}
	}
}

func (o *baseObject) hasOwnSym(s *valueSymbol) bool {
	_, exists := o.symValues[s]
	return exists
}

func (o *baseObject) hasOwnProperty(n Value) bool {
	if s, ok := n.(*valueSymbol); ok {
		_, exists := o.symValues[s]
		return exists
	}
	v := o.values[n.String()]
	return v != nil
}

func (o *baseObject) hasOwnPropertyStr(name string) bool {
	v := o.values[name]
	return v != nil
}

func (o *baseObject) _defineOwnProperty(name string, existingValue Value, descr PropertyDescriptor, throw bool) (val Value, ok bool) {

	getterObj, _ := descr.Getter.(*Object)
	setterObj, _ := descr.Setter.(*Object)

	var existing *valueProperty

	if existingValue == nil {
		if !o.extensible {
			o.val.runtime.typeErrorResult(throw, "Cannot define property %s, object is not extensible", name)
			return nil, false
		}
		existing = &valueProperty{}
	} else {
		if existing, ok = existingValue.(*valueProperty); !ok {
			existing = &valueProperty{
				writable:     true,
				enumerable:   true,
				configurable: true,
				value:        existingValue,
			}
		}

		if !existing.configurable {
			if descr.Configurable == FLAG_TRUE {
				goto Reject
			}
			if descr.Enumerable != FLAG_NOT_SET && descr.Enumerable.Bool() != existing.enumerable {
				goto Reject
			}
		}
		if existing.accessor && descr.Value != nil || !existing.accessor && (getterObj != nil || setterObj != nil) {
			if !existing.configurable {
				goto Reject
			}
		} else if !existing.accessor {
			if !existing.configurable {
				if !existing.writable {
					if descr.Writable == FLAG_TRUE {
						goto Reject
					}
					if descr.Value != nil && !descr.Value.SameAs(existing.value) {
						goto Reject
					}
				}
			}
		} else {
			if !existing.configurable {
				if descr.Getter != nil && existing.getterFunc != getterObj || descr.Setter != nil && existing.setterFunc != setterObj {
					goto Reject
				}
			}
		}
	}

	if descr.Writable == FLAG_TRUE && descr.Enumerable == FLAG_TRUE && descr.Configurable == FLAG_TRUE && descr.Value != nil {
		return descr.Value, true
	}

	if descr.Writable != FLAG_NOT_SET {
		existing.writable = descr.Writable.Bool()
	}
	if descr.Enumerable != FLAG_NOT_SET {
		existing.enumerable = descr.Enumerable.Bool()
	}
	if descr.Configurable != FLAG_NOT_SET {
		existing.configurable = descr.Configurable.Bool()
	}

	if descr.Value != nil {
		existing.value = descr.Value
		existing.getterFunc = nil
		existing.setterFunc = nil
	}

	if descr.Value != nil || descr.Writable != FLAG_NOT_SET {
		existing.accessor = false
	}

	if descr.Getter != nil {
		existing.getterFunc = propGetter(o.val, descr.Getter, o.val.runtime)
		existing.value = nil
		existing.accessor = true
	}

	if descr.Setter != nil {
		existing.setterFunc = propSetter(o.val, descr.Setter, o.val.runtime)
		existing.value = nil
		existing.accessor = true
	}

	if !existing.accessor && existing.value == nil {
		existing.value = _undefined
	}

	return existing, true

Reject:
	o.val.runtime.typeErrorResult(throw, "Cannot redefine property: %s", name)
	return nil, false

}

func (o *baseObject) defineOwnPropertyStr(name string, descr PropertyDescriptor, throw bool) bool {
	existingVal := o.values[name]
	if v, ok := o._defineOwnProperty(name, existingVal, descr, throw); ok {
		o.values[name] = v
		if existingVal == nil {
			o.propNames = append(o.propNames, name)
		}
		return true
	}
	return false
}

func (o *baseObject) defineOwnPropertySym(s *valueSymbol, descr PropertyDescriptor, throw bool) bool {
	existingVal := o.symValues[s]
	if v, ok := o._defineOwnProperty(s.String(), existingVal, descr, throw); ok {
		if o.symValues == nil {
			o.symValues = make(map[*valueSymbol]Value, 1)
		}
		o.symValues[s] = v
		return true
	}
	return false
}

func (o *baseObject) defineOwnProperty(n Value, descr PropertyDescriptor, throw bool) bool {
	if s, ok := n.(*valueSymbol); ok {
		return o.defineOwnPropertySym(s, descr, throw)
	}
	return o.defineOwnPropertyStr(n.String(), descr, throw)
}

func (o *baseObject) _put(name string, v Value) {
	if _, exists := o.values[name]; !exists {
		o.propNames = append(o.propNames, name)
	}

	o.values[name] = v
}

func valueProp(value Value, writable, enumerable, configurable bool) Value {
	if writable && enumerable && configurable {
		return value
	}
	return &valueProperty{
		value:        value,
		writable:     writable,
		enumerable:   enumerable,
		configurable: configurable,
	}
}

func (o *baseObject) _putProp(name string, value Value, writable, enumerable, configurable bool) Value {
	prop := valueProp(value, writable, enumerable, configurable)
	o._put(name, prop)
	return prop
}

func (o *baseObject) _putSym(s *valueSymbol, prop Value) {
	if o.symValues == nil {
		o.symValues = make(map[*valueSymbol]Value, 1)
	}
	o.symValues[s] = prop
}

func (o *baseObject) tryExoticToPrimitive(hint string) Value {
	exoticToPrimitive := toMethod(o.getSym(symToPrimitive, nil))
	if exoticToPrimitive != nil {
		return exoticToPrimitive(FunctionCall{
			This:      o.val,
			Arguments: []Value{newStringValue(hint)},
		})
	}
	return nil
}

func (o *baseObject) tryPrimitive(methodName string) Value {
	if method, ok := o.val.self.getStr(methodName, nil).(*Object); ok {
		if call, ok := method.self.assertCallable(); ok {
			v := call(FunctionCall{
				This: o.val,
			})
			if _, fail := v.(*Object); !fail {
				return v
			}
		}
	}
	return nil
}

func (o *baseObject) toPrimitiveNumber() Value {
	if v := o.tryExoticToPrimitive("number"); v != nil {
		return v
	}

	if v := o.tryPrimitive("valueOf"); v != nil {
		return v
	}

	if v := o.tryPrimitive("toString"); v != nil {
		return v
	}

	o.val.runtime.typeErrorResult(true, "Could not convert %v to primitive", o)
	return nil
}

func (o *baseObject) toPrimitiveString() Value {
	if v := o.tryExoticToPrimitive("string"); v != nil {
		return v
	}

	if v := o.tryPrimitive("toString"); v != nil {
		return v
	}

	if v := o.tryPrimitive("valueOf"); v != nil {
		return v
	}

	o.val.runtime.typeErrorResult(true, "Could not convert %v to primitive", o)
	return nil
}

func (o *baseObject) toPrimitive() Value {
	return o.toPrimitiveNumber()
}

func (o *baseObject) assertCallable() (func(FunctionCall) Value, bool) {
	return nil, false
}

func (o *baseObject) proto() *Object {
	return o.prototype
}

func (o *baseObject) isExtensible() bool {
	return o.extensible
}

func (o *baseObject) preventExtensions(bool) bool {
	o.extensible = false
	return true
}

func (o *baseObject) sortLen() int64 {
	return toLength(o.val.self.getStr("length", nil))
}

func (o *baseObject) sortGet(i int64) Value {
	return o.val.self.get(intToValue(i), nil)
}

func (o *baseObject) swap(i, j int64) {
	ii := intToValue(i)
	jj := intToValue(j)

	x := o.val.self.get(ii, nil)
	y := o.val.self.get(jj, nil)

	o.val.self.setOwn(ii, y, false)
	o.val.self.setOwn(jj, x, false)
}

func (o *baseObject) export() interface{} {
	m := make(map[string]interface{})
	for _, itemName := range o.ownKeys(false, nil) {
		itemNameStr := itemName.String()
		v := o.val.self.getStr(itemNameStr, nil)
		if v != nil {
			m[itemNameStr] = v.Export()
		} else {
			m[itemNameStr] = nil
		}
	}

	return m
}

func (o *baseObject) exportType() reflect.Type {
	return reflectTypeMap
}

type enumerableFlag int

const (
	_ENUM_UNKNOWN enumerableFlag = iota
	_ENUM_FALSE
	_ENUM_TRUE
)

type propIterItem struct {
	name       string
	value      Value // set only when enumerable == _ENUM_UNKNOWN
	enumerable enumerableFlag
}

type objectPropIter struct {
	o         *baseObject
	propNames []string
	idx       int
}

type propFilterIter struct {
	wrapped iterNextFunc
	all     bool
	seen    map[string]bool
}

func (i *propFilterIter) next() (propIterItem, iterNextFunc) {
	for {
		var item propIterItem
		item, i.wrapped = i.wrapped()
		if i.wrapped == nil {
			return propIterItem{}, nil
		}

		if !i.seen[item.name] {
			i.seen[item.name] = true
			if !i.all {
				if item.enumerable == _ENUM_FALSE {
					continue
				}
				if item.enumerable == _ENUM_UNKNOWN {
					if prop, ok := item.value.(*valueProperty); ok {
						if !prop.enumerable {
							continue
						}
					}
				}
			}
			return item, i.next
		}
	}
}

func (i *objectPropIter) next() (propIterItem, iterNextFunc) {
	for i.idx < len(i.propNames) {
		name := i.propNames[i.idx]
		i.idx++
		prop := i.o.values[name]
		if prop != nil {
			return propIterItem{name: name, value: prop}, i.next
		}
	}

	return propIterItem{}, nil
}

func (o *baseObject) enumerate() iterNextFunc {
	return (&propFilterIter{
		wrapped: o.val.self.enumerateUnfiltered(),
		seen:    make(map[string]bool),
	}).next
}

func (o *baseObject) ownIter() iterNextFunc {
	propNames := make([]string, len(o.propNames))
	copy(propNames, o.propNames)
	return (&objectPropIter{
		o:         o,
		propNames: propNames,
	}).next
}

func (o *baseObject) recursiveIter(iter iterNextFunc) iterNextFunc {
	return (&recursiveIter{
		o:       o,
		wrapped: iter,
	}).next
}

func (o *baseObject) enumerateUnfiltered() iterNextFunc {
	return o.recursiveIter(o.ownIter())
}

type recursiveIter struct {
	o       *baseObject
	wrapped iterNextFunc
}

func (iter *recursiveIter) next() (propIterItem, iterNextFunc) {
	item, next := iter.wrapped()
	if next != nil {
		iter.wrapped = next
		return item, iter.next
	}
	if proto := iter.o.prototype; proto != nil {
		return proto.self.enumerateUnfiltered()()
	}
	return propIterItem{}, nil
}

func (o *baseObject) equal(objectImpl) bool {
	// Rely on parent reference comparison
	return false
}

func (o *baseObject) ownKeys(all bool, keys []Value) []Value {
	if all {
		for _, k := range o.propNames {
			keys = append(keys, newStringValue(k))
		}
	} else {
		for _, k := range o.propNames {
			prop := o.values[k]
			if prop, ok := prop.(*valueProperty); ok && !prop.enumerable {
				continue
			}
			keys = append(keys, newStringValue(k))
		}
	}
	return keys
}

func (o *baseObject) ownSymbols() (res []Value) {
	for s := range o.symValues {
		res = append(res, s)
	}

	return
}

func (o *baseObject) hasInstance(Value) bool {
	panic(o.val.runtime.NewTypeError("Expecting a function in instanceof check, but got %s", o.val.toString()))
}

func toMethod(v Value) func(FunctionCall) Value {
	if v == nil || IsUndefined(v) || IsNull(v) {
		return nil
	}
	if obj, ok := v.(*Object); ok {
		if call, ok := obj.self.assertCallable(); ok {
			return call
		}
	}
	panic(typeError(fmt.Sprintf("%s is not a method", v.String())))
}

func instanceOfOperator(o Value, c *Object) bool {
	if instOfHandler := toMethod(c.self.get(symHasInstance, c)); instOfHandler != nil {
		return instOfHandler(FunctionCall{
			This:      c,
			Arguments: []Value{o},
		}).ToBoolean()
	}

	return c.self.hasInstance(o)
}

func (o *Object) getWeakCollRefs() *weakCollections {
	if o.weakColls == nil {
		o.weakColls = &weakCollections{}
		runtime.SetFinalizer(o.weakColls, finalizeObjectWeakRefs)
	}

	return o.weakColls
}
