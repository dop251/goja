package goja

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/dop251/goja/unistring"
)

/*
ReadonlyObject is an interface representing a handler for a readonly Object. Such an object can be created
using the Runtime.NewReadonlyObject() method.

Note that Runtime.ToValue() does not have any special treatment for ReadonlyObject. The only way to create
a readonly object is by using the Runtime.NewReadonlyObject() method. This is done deliberately to avoid
silent code breaks when this interface changes.
*/
type ReadonlyObject interface {
	// Get a property value for the key. May return nil if the property does not exist.
	Get(key string) Value
	// Has should return true if and only if the property exists.
	Has(key string) bool
	// Keys returns a list of all existing property keys. There are no checks for duplicates or to make sure
	// that the order conforms to https://262.ecma-international.org/#sec-ordinaryownpropertykeys
	Keys() []string
}

/*
DynamicObject is an interface representing a handler for a dynamic Object. Such an object can be created
using the Runtime.NewDynamicObject() method.

Note that Runtime.ToValue() does not have any special treatment for DynamicObject. The only way to create
a dynamic object is by using the Runtime.NewDynamicObject() method. This is done deliberately to avoid
silent code breaks when this interface changes.
*/
type DynamicObject interface {
	ReadonlyObject
	// Set a property value for the key. Return true if success, false otherwise.
	Set(key string, val Value) bool
	// Delete the property for the key. Returns true on success (note, that includes missing property).
	Delete(key string) bool
}

/*
ReadonlyArray is an interface representing a handler for a readonly array Object. Such an object can be created
using the Runtime.NewReadonlyArray() method.

Any integer property key or a string property key that can be parsed into an int value (including negative
ones) is treated as an index and passed to the trap methods of the ReadonlyArray. Note this is different from
the regular ECMAScript arrays which only support positive indexes up to 2^32-1.

ReadonlyArray cannot be sparse, i.e. hasOwnProperty(num) will return true for num >= 0 && num < Len(). Deleting
such a property is equivalent to setting it to undefined. Note that this creates a slight peculiarity because
hasOwnProperty() will still return true, even after deletion.

Note that Runtime.ToValue() does not have any special treatment for ReadonlyArray. The only way to create
a readonly array is by using the Runtime.NewReadonlyArray() method. This is done deliberately to avoid
silent code breaks when this interface changes.
*/
type ReadonlyArray interface {
	// Len returns the current array length.
	Len() int
	// Get an item at index idx. Note that idx may be any integer, negative or beyond the current length.
	Get(idx int) Value
}

/*
DynamicArray is an interface representing a handler for a dynamic array Object. Such an object can be created
using the Runtime.NewDynamicArray() method.

Any integer property key or a string property key that can be parsed into an int value (including negative
ones) is treated as an index and passed to the trap methods of the DynamicArray. Note this is different from
the regular ECMAScript arrays which only support positive indexes up to 2^32-1.

DynamicArray cannot be sparse, i.e. hasOwnProperty(num) will return true for num >= 0 && num < Len(). Deleting
such a property is equivalent to setting it to undefined. Note that this creates a slight peculiarity because
hasOwnProperty() will still return true, even after deletion.

Note that Runtime.ToValue() does not have any special treatment for DynamicArray. The only way to create
a dynamic array is by using the Runtime.NewDynamicArray() method. This is done deliberately to avoid
silent code breaks when this interface changes.
*/
type DynamicArray interface {
	ReadonlyArray
	// Set an item at index idx. Note that idx may be any integer, negative or beyond the current length.
	// The expected behaviour when it's beyond length is that the array's length is increased to accommodate
	// the item. All elements in the 'new' section of the array should be zeroed.
	Set(idx int, val Value) bool
	// SetLen is called when the array's 'length' property is changed. If the length is increased all elements in the
	// 'new' section of the array should be zeroed.
	SetLen(int) bool
}

type baseRWObject struct {
	val       *Object
	prototype *Object
}

type readonlyObject struct {
	baseRWObject
	d        ReadonlyObject
	readonly bool
}

func (o *readonlyObject) getDynamicObject() (do DynamicObject, ok bool) {
	do, ok = o.d.(DynamicObject)
	return
}

type dynamicObject struct {
	readonlyObject
	d DynamicObject
}

type readonlyArray struct {
	baseRWObject
	a        ReadonlyArray
	readonly bool
}

func (o *readonlyArray) getDynamicArray() (da DynamicArray, ok bool) {
	da, ok = o.a.(DynamicArray)
	return
}

type dynamicArray struct {
	readonlyArray
	a DynamicArray
}

/*
NewReadonlyObject creates an Object backed by the provided ReadonlyObject handler.

All properties of this Object are Writable, Enumerable and Configurable data properties. Any attempt to define
a property that does not conform to this will fail.

The Object is always extensible and cannot be made non-extensible. Object.preventExtensions() will fail.

The Object's prototype is initially set to Object.prototype, but can be changed using regular mechanisms
(Object.SetPrototype() in Go or Object.setPrototypeOf() in JS).

The Object cannot have own Symbol properties, however its prototype can. If you need an iterator support for
example, you could create a regular object, set Symbol.iterator on that object and then use it as a
prototype. See TestReadonlyObjectCustomProto for more details.

Export() returns the original ReadonlyObject.

This mechanism is similar to ECMAScript Proxy, however because all properties are enumerable and the object
is always extensible there is no need for invariant checks which removes the need to have a target object and
makes it a lot more efficient.
*/
func (r *Runtime) NewReadonlyObject(d ReadonlyObject) *Object {
	v := &Object{runtime: r}
	o := &readonlyObject{
		readonly: true,
		d:        d,
		baseRWObject: baseRWObject{
			val:       v,
			prototype: r.global.ObjectPrototype,
		},
	}
	v.self = o
	return v
}

/*
NewDynamicObject creates an Object backed by the provided DynamicObject handler.

All properties of this Object are Writable, Enumerable and Configurable data properties. Any attempt to define
a property that does not conform to this will fail.

The Object is always extensible and cannot be made non-extensible. Object.preventExtensions() will fail.

The Object's prototype is initially set to Object.prototype, but can be changed using regular mechanisms
(Object.SetPrototype() in Go or Object.setPrototypeOf() in JS).

The Object cannot have own Symbol properties, however its prototype can. If you need an iterator support for
example, you could create a regular object, set Symbol.iterator on that object and then use it as a
prototype. See TestDynamicObjectCustomProto for more details.

Export() returns the original DynamicObject.

This mechanism is similar to ECMAScript Proxy, however because all properties are enumerable and the object
is always extensible there is no need for invariant checks which removes the need to have a target object and
makes it a lot more efficient.
*/
func (r *Runtime) NewDynamicObject(d DynamicObject) *Object {
	v := &Object{runtime: r}
	o := &dynamicObject{
		d: d,
		readonlyObject: readonlyObject{baseRWObject: baseRWObject{
			val:       v,
			prototype: r.global.ObjectPrototype,
		},
			d: d,
		},
	}
	v.self = o
	return v
}

/*
NewReadonlyArray creates an array Object backed by the provided ReadonlyArray handler.
It is similar to NewReadonlyObject, the differences are:

- the Object is an array (i.e. Array.isArray() will return true and it will have the length property).

- the prototype will be initially set to Array.prototype.

- the Object cannot have any own string properties except for the 'length'.
*/
func (r *Runtime) NewReadonlyArray(a ReadonlyArray) *Object {
	v := &Object{runtime: r}
	o := &readonlyArray{
		readonly: true,
		a:        a,
		baseRWObject: baseRWObject{
			val:       v,
			prototype: r.global.ArrayPrototype,
		},
	}
	v.self = o
	return v
}

/*
NewDynamicArray creates an array Object backed by the provided DynamicArray handler.
It is similar to NewDynamicObject, the differences are:

- the Object is an array (i.e. Array.isArray() will return true and it will have the length property).

- the prototype will be initially set to Array.prototype.

- the Object cannot have any own string properties except for the 'length'.
*/
func (r *Runtime) NewDynamicArray(a DynamicArray) *Object {
	v := &Object{runtime: r}
	o := &dynamicArray{
		a: a,
		readonlyArray: readonlyArray{baseRWObject: baseRWObject{
			val:       v,
			prototype: r.global.ArrayPrototype,
		},
			a: a,
		},
	}
	v.self = o
	return v
}

func (*readonlyObject) sortLen() int {
	return 0
}

func (*readonlyObject) sortGet(i int) Value {
	return nil
}

func (*readonlyObject) swap(i int, i2 int) {
}

func (*readonlyObject) className() string {
	return classObject
}

func (o *baseRWObject) getParentStr(p unistring.String, receiver Value) Value {
	if proto := o.prototype; proto != nil {
		if receiver == nil {
			return proto.self.getStr(p, o.val)
		}
		return proto.self.getStr(p, receiver)
	}
	return nil
}

func (o *readonlyObject) getStr(p unistring.String, receiver Value) Value {
	prop := o.d.Get(p.String())
	if prop == nil {
		return o.getParentStr(p, receiver)
	}
	return prop
}

func (o *baseRWObject) getParentIdx(p valueInt, receiver Value) Value {
	if proto := o.prototype; proto != nil {
		if receiver == nil {
			return proto.self.getIdx(p, o.val)
		}
		return proto.self.getIdx(p, receiver)
	}
	return nil
}

func (o *readonlyObject) getIdx(p valueInt, receiver Value) Value {
	prop := o.d.Get(p.String())
	if prop == nil {
		return o.getParentIdx(p, receiver)
	}
	return prop
}

func (o *baseRWObject) getSym(p *Symbol, receiver Value) Value {
	if proto := o.prototype; proto != nil {
		if receiver == nil {
			return proto.self.getSym(p, o.val)
		}
		return proto.self.getSym(p, receiver)
	}
	return nil
}

func (o *readonlyObject) getOwnPropStr(u unistring.String) Value {
	return o.d.Get(u.String())
}

func (o *readonlyObject) getOwnPropIdx(v valueInt) Value {
	return o.d.Get(v.String())
}

func (*baseRWObject) getOwnPropSym(*Symbol) Value {
	return nil
}

func (o *readonlyObject) _set(prop string, v Value, throw bool) bool {
	if o.readonly {
		return true
	}
	if do, ok := o.getDynamicObject(); ok {
		if do.Set(prop, v) {
			return true
		}
		o.val.runtime.typeErrorResult(throw, "'Set' on a readonly object returned false")
		return false
	}
	return true
}

func (o *baseRWObject) _setSym(throw bool) {
	o.val.runtime.typeErrorResult(throw, "Readonly objects do not support Symbol properties")
}

func (o *readonlyObject) setOwnStr(p unistring.String, v Value, throw bool) bool {
	prop := p.String()
	if !o.d.Has(prop) {
		if proto := o.prototype; proto != nil {
			// we know it's foreign because prototype loops are not allowed
			if res, handled := proto.self.setForeignStr(p, v, o.val, throw); handled {
				return res
			}
		}
	}
	return o._set(prop, v, throw)
}

func (o *readonlyObject) setOwnIdx(p valueInt, v Value, throw bool) bool {
	prop := p.String()
	if !o.d.Has(prop) {
		if proto := o.prototype; proto != nil {
			// we know it's foreign because prototype loops are not allowed
			if res, handled := proto.self.setForeignIdx(p, v, o.val, throw); handled {
				return res
			}
		}
	}
	return o._set(prop, v, throw)
}

func (o *baseRWObject) setOwnSym(s *Symbol, v Value, throw bool) bool {
	if proto := o.prototype; proto != nil {
		// we know it's foreign because prototype loops are not allowed
		if res, handled := proto.self.setForeignSym(s, v, o.val, throw); handled {
			return res
		}
	}
	o._setSym(throw)
	return false
}

func (o *baseRWObject) setParentForeignStr(p unistring.String, v, receiver Value, throw bool) (res bool, handled bool) {
	if proto := o.prototype; proto != nil {
		if receiver != proto {
			return proto.self.setForeignStr(p, v, receiver, throw)
		}
		return proto.self.setOwnStr(p, v, throw), true
	}
	return false, false
}

func (o *readonlyObject) setForeignStr(p unistring.String, v, receiver Value, throw bool) (res bool, handled bool) {
	prop := p.String()
	if !o.d.Has(prop) {
		return o.setParentForeignStr(p, v, receiver, throw)
	}
	return false, false
}

func (o *baseRWObject) setParentForeignIdx(p valueInt, v, receiver Value, throw bool) (res bool, handled bool) {
	if proto := o.prototype; proto != nil {
		if receiver != proto {
			return proto.self.setForeignIdx(p, v, receiver, throw)
		}
		return proto.self.setOwnIdx(p, v, throw), true
	}
	return false, false
}

func (o *readonlyObject) setForeignIdx(p valueInt, v, receiver Value, throw bool) (res bool, handled bool) {
	prop := p.String()
	if !o.d.Has(prop) {
		return o.setParentForeignIdx(p, v, receiver, throw)
	}
	return false, false
}

func (o *baseRWObject) setForeignSym(p *Symbol, v, receiver Value, throw bool) (res bool, handled bool) {
	if proto := o.prototype; proto != nil {
		if receiver != proto {
			return proto.self.setForeignSym(p, v, receiver, throw)
		}
		return proto.self.setOwnSym(p, v, throw), true
	}
	return false, false
}

func (o *readonlyObject) hasPropertyStr(u unistring.String) bool {
	if o.hasOwnPropertyStr(u) {
		return true
	}
	if proto := o.prototype; proto != nil {
		return proto.self.hasPropertyStr(u)
	}
	return false
}

func (o *readonlyObject) hasPropertyIdx(idx valueInt) bool {
	if o.hasOwnPropertyIdx(idx) {
		return true
	}
	if proto := o.prototype; proto != nil {
		return proto.self.hasPropertyIdx(idx)
	}
	return false
}

func (o *baseRWObject) hasPropertySym(s *Symbol) bool {
	if proto := o.prototype; proto != nil {
		return proto.self.hasPropertySym(s)
	}
	return false
}

func (o *readonlyObject) hasOwnPropertyStr(u unistring.String) bool {
	return o.d.Has(u.String())
}

func (o *readonlyObject) hasOwnPropertyIdx(v valueInt) bool {
	return o.d.Has(v.String())
}

func (*baseRWObject) hasOwnPropertySym(_ *Symbol) bool {
	return false
}

func (o *baseRWObject) checkReadonlyObjectPropertyDescr(name fmt.Stringer, descr PropertyDescriptor, throw bool) bool {
	if descr.Getter != nil || descr.Setter != nil {
		o.val.runtime.typeErrorResult(throw, "Readonly objects do not support accessor properties")
		return false
	}
	if descr.Writable == FLAG_FALSE {
		o.val.runtime.typeErrorResult(throw, "Readonly object field %q cannot be made read-only", name.String())
		return false
	}
	if descr.Enumerable == FLAG_FALSE {
		o.val.runtime.typeErrorResult(throw, "Readonly object field %q cannot be made non-enumerable", name.String())
		return false
	}
	if descr.Configurable == FLAG_FALSE {
		o.val.runtime.typeErrorResult(throw, "Readonly object field %q cannot be made non-configurable", name.String())
		return false
	}
	return true
}

func (o *readonlyObject) defineOwnPropertyStr(name unistring.String, desc PropertyDescriptor, throw bool) bool {
	if o.checkReadonlyObjectPropertyDescr(name, desc, throw) {
		return o._set(name.String(), desc.Value, throw)
	}
	return false
}

func (o *readonlyObject) defineOwnPropertyIdx(name valueInt, desc PropertyDescriptor, throw bool) bool {
	if o.checkReadonlyObjectPropertyDescr(name, desc, throw) {
		return o._set(name.String(), desc.Value, throw)
	}
	return false
}

func (o *baseRWObject) defineOwnPropertySym(name *Symbol, desc PropertyDescriptor, throw bool) bool {
	o._setSym(throw)
	return false
}

func (o *readonlyObject) _delete(prop string, throw bool) bool {
	if o.readonly {
		return true
	}
	if do, ok := o.getDynamicObject(); ok {
		if do.Delete(prop) {
			return true
		}
		o.val.runtime.typeErrorResult(throw, "Could not delete property %q of a readonly object", prop)
		return false
	}
	return true
}

func (o *readonlyObject) deleteStr(name unistring.String, throw bool) bool {
	return o._delete(name.String(), throw)
}

func (o *readonlyObject) deleteIdx(idx valueInt, throw bool) bool {
	return o._delete(idx.String(), throw)
}

func (*baseRWObject) deleteSym(_ *Symbol, _ bool) bool {
	return true
}

func (o *baseRWObject) toPrimitiveNumber() Value {
	return o.val.genericToPrimitiveNumber()
}

func (o *baseRWObject) toPrimitiveString() Value {
	return o.val.genericToPrimitiveString()
}

func (o *baseRWObject) toPrimitive() Value {
	return o.val.genericToPrimitive()
}

func (o *baseRWObject) assertCallable() (call func(FunctionCall) Value, ok bool) {
	return nil, false
}

func (*baseRWObject) assertConstructor() func(args []Value, newTarget *Object) *Object {
	return nil
}

func (o *baseRWObject) proto() *Object {
	return o.prototype
}

func (o *baseRWObject) setProto(proto *Object, throw bool) bool {
	o.prototype = proto
	return true
}

func (o *baseRWObject) hasInstance(v Value) bool {
	panic(o.val.runtime.NewTypeError("Expecting a function in instanceof check, but got a readonly object"))
}

func (*baseRWObject) isExtensible() bool {
	return true
}

func (o *baseRWObject) preventExtensions(throw bool) bool {
	o.val.runtime.typeErrorResult(throw, "Cannot make a readonly object non-extensible")
	return false
}

type readonlyObjectPropIter struct {
	o         *readonlyObject
	propNames []string
	idx       int
}

func (i *readonlyObjectPropIter) next() (propIterItem, iterNextFunc) {
	for i.idx < len(i.propNames) {
		name := i.propNames[i.idx]
		i.idx++
		if i.o.d.Has(name) {
			return propIterItem{name: newStringValue(name), enumerable: _ENUM_TRUE}, i.next
		}
	}
	return propIterItem{}, nil
}

func (o *readonlyObject) iterateStringKeys() iterNextFunc {
	keys := o.d.Keys()
	return (&readonlyObjectPropIter{
		o:         o,
		propNames: keys,
	}).next
}

func (o *baseRWObject) iterateSymbols() iterNextFunc {
	return func() (propIterItem, iterNextFunc) {
		return propIterItem{}, nil
	}
}

func (o *readonlyObject) iterateKeys() iterNextFunc {
	return o.iterateStringKeys()
}

func (o *readonlyObject) export(ctx *objectExportCtx) interface{} {
	return o.d
}

func (o *readonlyObject) exportType() reflect.Type {
	return reflect.TypeOf(o.d)
}

func (o *baseRWObject) exportToMap(dst reflect.Value, typ reflect.Type, ctx *objectExportCtx) error {
	return genericExportToMap(o.val, dst, typ, ctx)
}

func (o *baseRWObject) exportToArrayOrSlice(dst reflect.Value, typ reflect.Type, ctx *objectExportCtx) error {
	return genericExportToArrayOrSlice(o.val, dst, typ, ctx)
}

func (o *readonlyObject) equal(impl objectImpl) bool {
	if other, ok := impl.(*readonlyObject); ok {
		return o.d == other.d
	}
	return false
}

func (o *readonlyObject) stringKeys(all bool, accum []Value) []Value {
	keys := o.d.Keys()
	if l := len(accum) + len(keys); l > cap(accum) {
		oldAccum := accum
		accum = make([]Value, len(accum), l)
		copy(accum, oldAccum)
	}
	for _, key := range keys {
		accum = append(accum, newStringValue(key))
	}
	return accum
}

func (*baseRWObject) symbols(all bool, accum []Value) []Value {
	return accum
}

func (o *readonlyObject) keys(all bool, accum []Value) []Value {
	return o.stringKeys(all, accum)
}

func (*baseRWObject) _putProp(name unistring.String, value Value, writable, enumerable, configurable bool) Value {
	return nil
}

func (*baseRWObject) _putSym(s *Symbol, prop Value) {
}

func (o *baseRWObject) getPrivateEnv(*privateEnvType, bool) *privateElements {
	panic(o.val.runtime.NewTypeError("Readonly objects cannot have private elements"))
}

func (a *readonlyArray) sortLen() int {
	return a.a.Len()
}

func (a *readonlyArray) sortGet(i int) Value {
	return a.a.Get(i)
}

func (a *readonlyArray) swap(i int, j int) {
	if a.readonly {
		return
	}
	if da, ok := a.getDynamicArray(); ok {
		x := a.sortGet(i)
		y := a.sortGet(j)
		da.Set(int(i), y)
		da.Set(int(j), x)
	}
}

func (a *readonlyArray) className() string {
	return classArray
}

func (a *readonlyArray) getStr(p unistring.String, receiver Value) Value {
	if p == "length" {
		return intToValue(int64(a.a.Len()))
	}
	if idx, ok := strToInt(p); ok {
		return a.a.Get(idx)
	}
	return a.getParentStr(p, receiver)
}

func (a *readonlyArray) getIdx(p valueInt, receiver Value) Value {
	if val := a.getOwnPropIdx(p); val != nil {
		return val
	}
	return a.getParentIdx(p, receiver)
}

func (a *readonlyArray) getOwnPropStr(u unistring.String) Value {
	if u == "length" {
		return &valueProperty{
			value:    intToValue(int64(a.a.Len())),
			writable: true,
		}
	}
	if idx, ok := strToInt(u); ok {
		return a.a.Get(idx)
	}
	return nil
}

func (a *readonlyArray) getOwnPropIdx(v valueInt) Value {
	return a.a.Get(toIntStrict(int64(v)))
}

func (a *readonlyArray) _setLen(v Value, throw bool) bool {
	if a.readonly {
		return true
	}
	if da, ok := a.getDynamicArray(); ok {
		if da.SetLen(toIntStrict(v.ToInteger())) {
			return true
		}
		a.val.runtime.typeErrorResult(throw, "'SetLen' on a readonly array returned false")
		return false
	}
	return true
}

func (a *readonlyArray) setOwnStr(p unistring.String, v Value, throw bool) bool {
	if p == "length" {
		return a._setLen(v, throw)
	}
	if idx, ok := strToInt(p); ok {
		return a._setIdx(idx, v, throw)
	}
	a.val.runtime.typeErrorResult(throw, "Cannot set property %q on a readonly array", p.String())
	return false
}

func (a *readonlyArray) _setIdx(idx int, v Value, throw bool) bool {
	if a.readonly {
		return true
	}
	if da, ok := a.getDynamicArray(); ok {
		if da.Set(idx, v) {
			return true
		}
		a.val.runtime.typeErrorResult(throw, "'Set' on a readonly array returned false")
		return false
	}
	return true
}

func (a *readonlyArray) setOwnIdx(p valueInt, v Value, throw bool) bool {
	return a._setIdx(toIntStrict(int64(p)), v, throw)
}

func (a *readonlyArray) setForeignStr(p unistring.String, v, receiver Value, throw bool) (res bool, handled bool) {
	return a.setParentForeignStr(p, v, receiver, throw)
}

func (a *readonlyArray) setForeignIdx(p valueInt, v, receiver Value, throw bool) (res bool, handled bool) {
	return a.setParentForeignIdx(p, v, receiver, throw)
}

func (a *readonlyArray) hasPropertyStr(u unistring.String) bool {
	if a.hasOwnPropertyStr(u) {
		return true
	}
	if proto := a.prototype; proto != nil {
		return proto.self.hasPropertyStr(u)
	}
	return false
}

func (a *readonlyArray) hasPropertyIdx(idx valueInt) bool {
	if a.hasOwnPropertyIdx(idx) {
		return true
	}
	if proto := a.prototype; proto != nil {
		return proto.self.hasPropertyIdx(idx)
	}
	return false
}

func (a *readonlyArray) _has(idx int) bool {
	return idx >= 0 && idx < a.a.Len()
}

func (a *readonlyArray) hasOwnPropertyStr(u unistring.String) bool {
	if u == "length" {
		return true
	}
	if idx, ok := strToInt(u); ok {
		return a._has(idx)
	}
	return false
}

func (a *readonlyArray) hasOwnPropertyIdx(v valueInt) bool {
	return a._has(toIntStrict(int64(v)))
}

func (a *readonlyArray) defineOwnPropertyStr(name unistring.String, desc PropertyDescriptor, throw bool) bool {
	if a.checkReadonlyObjectPropertyDescr(name, desc, throw) {
		if idx, ok := strToInt(name); ok {
			return a._setIdx(idx, desc.Value, throw)
		}
		a.val.runtime.typeErrorResult(throw, "Cannot define property %q on a readonly array", name.String())
	}
	return false
}

func (a *readonlyArray) defineOwnPropertyIdx(name valueInt, desc PropertyDescriptor, throw bool) bool {
	if a.checkReadonlyObjectPropertyDescr(name, desc, throw) {
		return a._setIdx(toIntStrict(int64(name)), desc.Value, throw)
	}
	return false
}

func (a *readonlyArray) _delete(idx int, throw bool) bool {
	if a._has(idx) {
		a._setIdx(idx, _undefined, throw)
	}
	return true
}

func (a *readonlyArray) deleteStr(name unistring.String, throw bool) bool {
	if idx, ok := strToInt(name); ok {
		return a._delete(idx, throw)
	}
	if a.hasOwnPropertyStr(name) {
		a.val.runtime.typeErrorResult(throw, "Cannot delete property %q on a readonly array", name.String())
		return false
	}
	return true
}

func (a *readonlyArray) deleteIdx(idx valueInt, throw bool) bool {
	return a._delete(toIntStrict(int64(idx)), throw)
}

type readonlyArrayPropIter struct {
	a          ReadonlyArray
	idx, limit int
}

func (i *readonlyArrayPropIter) next() (propIterItem, iterNextFunc) {
	if i.idx < i.limit && i.idx < i.a.Len() {
		name := strconv.Itoa(i.idx)
		i.idx++
		return propIterItem{name: asciiString(name), enumerable: _ENUM_TRUE}, i.next
	}

	return propIterItem{}, nil
}

func (a *readonlyArray) iterateStringKeys() iterNextFunc {
	return (&readonlyArrayPropIter{
		a:     a.a,
		limit: a.a.Len(),
	}).next
}

func (a *readonlyArray) iterateKeys() iterNextFunc {
	return a.iterateStringKeys()
}

func (a *readonlyArray) export(ctx *objectExportCtx) interface{} {
	return a.a
}

func (a *readonlyArray) exportType() reflect.Type {
	return reflect.TypeOf(a.a)
}

func (a *readonlyArray) equal(impl objectImpl) bool {
	if other, ok := impl.(*readonlyArray); ok {
		return a == other
	}
	return false
}

func (a *readonlyArray) stringKeys(all bool, accum []Value) []Value {
	al := a.a.Len()
	l := len(accum) + al
	if all {
		l++
	}
	if l > cap(accum) {
		oldAccum := accum
		accum = make([]Value, len(oldAccum), l)
		copy(accum, oldAccum)
	}
	for i := 0; i < al; i++ {
		accum = append(accum, asciiString(strconv.Itoa(i)))
	}
	if all {
		accum = append(accum, asciiString("length"))
	}
	return accum
}

func (a *readonlyArray) keys(all bool, accum []Value) []Value {
	return a.stringKeys(all, accum)
}
