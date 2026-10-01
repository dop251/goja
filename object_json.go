package goja

import (
	"reflect"
	"sort"

	"github.com/dop251/goja/unistring"
)

// jsonShape describes the property layout of one or more objects produced by a
// single JSON.parse() call. Keys are kept in insertion order, while enumOrder
// contains the ECMAScript property enumeration order when it differs.
type jsonShape struct {
	keys      []unistring.String
	lookup    map[unistring.String]uint32
	enumOrder []uint32
	hashNext  *jsonShape
}

func newJSONShape(keys []unistring.String) *jsonShape {
	shape := &jsonShape{keys: keys}
	if len(keys) > 8 {
		shape.lookup = make(map[unistring.String]uint32, len(keys))
		for i, key := range keys {
			shape.lookup[key] = uint32(i)
		}
	}

	type indexedSlot struct {
		index uint32
		slot  uint32
	}
	var indexed []indexedSlot
	for slot, key := range keys {
		if index := strToArrayIdx(key); index != ^uint32(0) {
			indexed = append(indexed, indexedSlot{index: index, slot: uint32(slot)})
		}
	}
	if len(indexed) != 0 {
		sort.Slice(indexed, func(i, j int) bool {
			return indexed[i].index < indexed[j].index
		})
		shape.enumOrder = make([]uint32, 0, len(keys))
		for _, entry := range indexed {
			shape.enumOrder = append(shape.enumOrder, entry.slot)
		}
		for slot, key := range keys {
			if strToArrayIdx(key) == ^uint32(0) {
				shape.enumOrder = append(shape.enumOrder, uint32(slot))
			}
		}
	}
	return shape
}

func (s *jsonShape) find(name unistring.String) (uint32, bool) {
	if s.lookup != nil {
		slot, exists := s.lookup[name]
		return slot, exists
	}
	for slot, key := range s.keys {
		if key == name {
			return uint32(slot), true
		}
	}
	return 0, false
}

func (s *jsonShape) orderedSlot(i int) uint32 {
	if s.enumOrder != nil {
		return s.enumOrder[i]
	}
	return uint32(i)
}

// jsonObject is the compact representation of a plain object created by
// JSON.parse(). Structural mutations materialize it into a regular baseObject.
// Keeping Object in the same allocation saves one allocation for the common
// immutable case. shape and values are cleared when the object materializes.
type jsonObject struct {
	object Object
	baseObject
	shape  *jsonShape
	values []Value
}

func (r *Runtime) newJSONObject(shape *jsonShape, values []Value) *Object {
	o := &jsonObject{
		shape:  shape,
		values: values,
	}
	o.object.runtime = r
	o.baseObject = baseObject{
		class:      classObject,
		val:        &o.object,
		prototype:  r.global.ObjectPrototype,
		extensible: true,
	}
	o.object.self = o
	return &o.object
}

func (o *jsonObject) materialize() *baseObject {
	if current, ok := o.val.self.(*baseObject); ok {
		return current
	}

	shape := o.shape
	values := o.values
	base := &o.baseObject
	base.values = make(map[unistring.String]Value, len(shape.keys))
	base.propNames = make([]unistring.String, len(shape.keys))
	copy(base.propNames, shape.keys)
	for slot, key := range shape.keys {
		base.values[key] = values[slot]
	}

	// Publish the complete replacement before dropping the compact data. Any
	// operation after this point dispatches through the ordinary object.
	o.val.self = base
	o.shape = nil
	o.values = nil
	return base
}

func (o *jsonObject) getOwnPropStr(name unistring.String) Value {
	if slot, exists := o.shape.find(name); exists {
		return o.values[slot]
	}
	return nil
}

func (o *jsonObject) getStr(name unistring.String, receiver Value) Value {
	if value := o.getOwnPropStr(name); value != nil {
		return value
	}
	if o.prototype != nil {
		if receiver == nil {
			return o.prototype.self.getStr(name, o.val)
		}
		return o.prototype.self.getStr(name, receiver)
	}
	return nil
}

func (o *jsonObject) hasOwnPropertyStr(name unistring.String) bool {
	_, exists := o.shape.find(name)
	return exists
}

func (o *jsonObject) setOwnStr(name unistring.String, value Value, throw bool) bool {
	if slot, exists := o.shape.find(name); exists {
		o.values[slot] = value
		return true
	}
	return o.materialize().setOwnStr(name, value, throw)
}

func (o *jsonObject) setForeignStr(name unistring.String, value, receiver Value, throw bool) (bool, bool) {
	var own Value
	if slot, exists := o.shape.find(name); exists {
		own = o.values[slot]
	}
	return o.baseObject._setForeignStr(name, own, value, receiver, throw)
}

func (o *jsonObject) setForeignIdx(index valueInt, value, receiver Value, throw bool) (bool, bool) {
	return o.setForeignStr(index.string(), value, receiver, throw)
}

func (o *jsonObject) setOwnSym(name *Symbol, value Value, throw bool) bool {
	return o.materialize().setOwnSym(name, value, throw)
}

func (o *jsonObject) defineOwnPropertyStr(name unistring.String, desc PropertyDescriptor, throw bool) bool {
	if slot, exists := o.shape.find(name); exists &&
		desc.Value != nil && desc.Getter == nil && desc.Setter == nil &&
		desc.Writable == FLAG_TRUE && desc.Enumerable == FLAG_TRUE && desc.Configurable == FLAG_TRUE {
		o.values[slot] = desc.Value
		return true
	}
	return o.materialize().defineOwnPropertyStr(name, desc, throw)
}

func (o *jsonObject) defineOwnPropertySym(name *Symbol, desc PropertyDescriptor, throw bool) bool {
	return o.materialize().defineOwnPropertySym(name, desc, throw)
}

func (o *jsonObject) deleteStr(name unistring.String, throw bool) bool {
	if _, exists := o.shape.find(name); !exists {
		return true
	}
	return o.materialize().deleteStr(name, throw)
}

func (o *jsonObject) deleteSym(name *Symbol, throw bool) bool {
	return true
}

func (o *jsonObject) setProto(proto *Object, throw bool) bool {
	return o.materialize().setProto(proto, throw)
}

func (o *jsonObject) preventExtensions(throw bool) bool {
	return o.materialize().preventExtensions(throw)
}

func (o *jsonObject) export(ctx *objectExportCtx) interface{} {
	if value, exists := ctx.get(o.val); exists {
		return value
	}
	result := make(map[string]interface{}, len(o.shape.keys))
	ctx.put(o.val, result)
	for slot, key := range o.shape.keys {
		result[key.String()] = exportValue(o.values[slot], ctx)
	}
	return result
}

func (o *jsonObject) exportType() reflect.Type {
	return reflectTypeMap
}

type jsonObjectPropIter struct {
	object *Object
	shape  *jsonShape
	index  int
}

func (i *jsonObjectPropIter) next() (propIterItem, iterNextFunc) {
	for i.index < len(i.shape.keys) {
		slot := i.shape.orderedSlot(i.index)
		i.index++
		name := i.shape.keys[slot]
		value := i.object.self.getOwnPropStr(name)
		if value != nil {
			return propIterItem{name: stringValueFromRaw(name), value: value}, i.next
		}
	}
	return propIterItem{}, nil
}

func (o *jsonObject) iterateStringKeys() iterNextFunc {
	return (&jsonObjectPropIter{object: o.val, shape: o.shape}).next
}

func (o *jsonObject) stringKeys(_ bool, keys []Value) []Value {
	for i := range o.shape.keys {
		slot := o.shape.orderedSlot(i)
		keys = append(keys, stringValueFromRaw(o.shape.keys[slot]))
	}
	return keys
}

func (o *jsonObject) _putProp(name unistring.String, value Value, writable, enumerable, configurable bool) Value {
	if slot, exists := o.shape.find(name); exists && writable && enumerable && configurable {
		o.values[slot] = value
		return value
	}
	return o.materialize()._putProp(name, value, writable, enumerable, configurable)
}

func (o *jsonObject) _putSym(name *Symbol, value Value) {
	o.materialize()._putSym(name, value)
}

func (o *jsonObject) getPrivateEnv(typ *privateEnvType, create bool) *privateElements {
	if create {
		return o.materialize().getPrivateEnv(typ, true)
	}
	return nil
}
