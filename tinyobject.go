package goja

import (
	"math"
	"reflect"
	"weak"

	"github.com/dop251/goja/unistring"
)

const tinyObjectMaxProps = 8

type propTransition struct {
	prop  unistring.String
	class weak.Pointer[tinyClass]
}

type tinyClass struct {
	parent *tinyClass
	keys   *[]unistring.String

	singlePropTransition propTransition
	propTransitions      map[unistring.String]weak.Pointer[tinyClass]

	notExtensible weak.Pointer[tinyClass]

	keysLen    uint8
	extensible bool
}

// tinyObject is a compact representation of a trivial object of a fixed structure specified by tinyClass.
// To qualify an object must:
//   - contain only writable, enumerable and configurable data properties (no accessors);
//   - contain no more than tinyObjectMaxProps properties;
//   - do not contain any indexed properties;
//   - do not contain any Symbols.
//
// If any of these invariants is broken, the object gets deoptimised to a regular baseObject.
// Deoptimisation also occurs if a property is deleted, unless it's the property that was added last
// (in this case the class gets changed to parent).
type tinyObject struct {
	class     *tinyClass
	prototype *Object
	val       *Object
	values    []Value
}

func (c *tinyClass) getForProp(name unistring.String) *tinyClass {
	stats.incTinyClassTotal()
	if cls := c.singlePropTransition.class; cls != (weak.Pointer[tinyClass]{}) && c.singlePropTransition.prop == name {
		if child := cls.Value(); child != nil {
			return child
		}
	}

	if t, exists := c.propTransitions[name]; exists {
		if child := t.Value(); child != nil {
			return child
		}
	}

	stats.incTinyClassMisses()

	child := &tinyClass{
		parent:     c,
		keysLen:    c.keysLen + 1,
		extensible: c.extensible,
	}

	singleIsFree := c.singlePropTransition.class.Value() == nil

	// The decision of whether to copy or re-use the keys slice needs to be made carefully.
	// For root class we always copy because the original empty keys slice is strongly referenced from the Runtime.
	// Otherwise, we can only do it if it's currently a leaf class.
	// If single transition is currently free (which means it's either never been used or it was, but the child got
	// freed) and the propTransitions hasn't been created we can re-use the slice as the tail is not currently used
	// by any class. The only other possible transition is to a non-extensible class which by definition cannot add
	// properties.
	if c.keysLen > 0 && singleIsFree && c.propTransitions == nil {
		*c.keys = append(*c.keys, name)
		child.keys = c.keys
	} else {
		newKeys := make([]unistring.String, c.keysLen+1)
		copy(newKeys, *c.keys)
		newKeys[len(newKeys)-1] = name
		child.keys = &newKeys
	}

	if singleIsFree {
		c.singlePropTransition.class = weak.Make(child)
		c.singlePropTransition.prop = name
	} else {
		stats.incTinyClassMultiTransitions()
		if c.propTransitions == nil {
			c.propTransitions = make(map[unistring.String]weak.Pointer[tinyClass], 2)
		}
		c.propTransitions[name] = weak.Make(child)
	}
	return child
}

func (c *tinyClass) idxForName(name unistring.String) int {
	for i, key := range (*c.keys)[:c.keysLen] {
		if len(key) != len(name) {
			continue
		}
		if key == name {
			return i
		}
	}
	return -1
}

func (c *tinyClass) getNotExtensible() *tinyClass {
	stats.incTinyClassTotal()
	cls := c.notExtensible.Value()
	if cls == nil {
		stats.incTinyClassMisses()
		cls = &tinyClass{
			parent:  c,
			keys:    c.keys,
			keysLen: c.keysLen,
		}
		c.notExtensible = weak.Make(cls)
	}
	return cls
}

func (o *tinyObject) deoptimize() *baseObject {
	stats.incTinyObjectDeoptimizations()
	bo := newBaseObjectObj(o.val, o.prototype, o.className())
	bo.propNames = append(([]unistring.String)(nil), (*o.class.keys)[:o.class.keysLen]...)
	bo._prepareValues()
	for i, name := range bo.propNames {
		bo.values[name] = o.values[i]
	}
	bo.extensible = o.class.extensible
	return bo
}

func (o *tinyObject) className() string {
	return classObject
}

func (o *tinyObject) typeOf() String {
	return stringObjectC
}

func (o *tinyObject) getOwnPropStr(name unistring.String) Value {
	idx := o.class.idxForName(name)
	if idx != -1 {
		return o.values[idx]
	}
	return nil
}

func (o *tinyObject) hasOwnPropertyStr(name unistring.String) bool {
	return o.class.idxForName(name) != -1
}

func (o *tinyObject) hasOwnPropertyIdx(idx valueInt) bool {
	if toIdx(idx) != math.MaxUint32 {
		return false
	}
	return o.hasOwnPropertyStr(idx.string())
}

func (o *tinyObject) hasOwnPropertySym(_ *Symbol) bool {
	return false
}

func (o *tinyObject) getOwnPropIdx(idx valueInt) Value {
	if toIdx(idx) == math.MaxUint32 {
		return o.getOwnPropStr(idx.string())
	}
	return nil
}

func (o *tinyObject) getOwnPropSym(_ *Symbol) Value {
	return nil
}

func (o *tinyObject) hasPropertyStr(name unistring.String) bool {
	if o.hasOwnPropertyStr(name) {
		return true
	}
	if o.prototype != nil {
		return o.prototype.self.hasPropertyStr(name)
	}
	return false
}

func (o *tinyObject) hasPropertyIdx(idx valueInt) bool {
	if toIdx(idx) == math.MaxUint32 {
		return o.hasPropertyStr(idx.string())
	}
	if o.prototype != nil {
		return o.prototype.self.hasPropertyIdx(idx)
	}
	return false
}

func (o *tinyObject) hasPropertySym(s *Symbol) bool {
	if o.prototype != nil {
		return o.prototype.self.hasPropertySym(s)
	}
	return false
}

func (o *tinyObject) getStr(name unistring.String, receiver Value) Value {
	prop := o.getOwnPropStr(name)
	if prop == nil {
		if o.prototype != nil {
			if receiver == nil {
				return o.prototype.self.getStr(name, o.val)
			}
			return o.prototype.self.getStr(name, receiver)
		}
	}
	return prop
}

func (o *tinyObject) getIdx(idx valueInt, receiver Value) Value {
	if n := toIdx(idx); n == math.MaxUint32 {
		return o.getStr(idx.string(), receiver)
	}

	if o.prototype != nil {
		if receiver == nil {
			return o.prototype.self.getIdx(idx, o.val)
		}
		return o.prototype.self.getIdx(idx, receiver)
	}
	return nil
}

func (o *tinyObject) getSym(s *Symbol, receiver Value) Value {
	if o.prototype != nil {
		if receiver == nil {
			return o.prototype.self.getSym(s, o.val)
		}
		return o.prototype.self.getSym(s, receiver)
	}
	return nil
}

func (o *tinyObject) _addProp(name unistring.String, val Value, throw bool) bool {
	if !o.class.extensible {
		o.val.runtime.typeErrorResult(throw, "Cannot add property %s, object is not extensible", name)
		return false
	}
	if len(o.values) >= tinyObjectMaxProps {
		o.deoptimize()._put(name, val)
		return true
	}
	if idx := strToArrayIdx(name); idx != math.MaxUint32 {
		o.deoptimize()._put(name, val)
		return true
	}
	newCls := o.class.getForProp(name)
	o.class = newCls
	o.values = append(o.values, val)
	return true
}

func (o *tinyObject) setOwnStr(name unistring.String, val Value, throw bool) bool {
	idx := o.class.idxForName(name)
	if idx == -1 {
		if proto := o.prototype; proto != nil {
			// we know it's foreign because prototype loops are not allowed
			if res, handled := proto.self.setForeignStr(name, val, o.val, throw); handled {
				return res
			}
		}
		// new property
		return o._addProp(name, val, throw)
	}
	o.values[idx] = val
	return true
}

func (o *tinyObject) setOwnIdx(idx valueInt, val Value, throw bool) bool {
	if toIdx(idx) == math.MaxUint32 {
		return o.setOwnStr(idx.string(), val, throw)
	}
	if proto := o.prototype; proto != nil {
		// we know it's foreign because prototype loops are not allowed
		if res, handled := proto.self.setForeignIdx(idx, val, o.val, throw); handled {
			return res
		}
	}
	// new property
	if !o.class.extensible {
		o.val.runtime.typeErrorResult(throw, "Cannot add property %s, object is not extensible", idx.String())
		return false
	}

	o.deoptimize()._put(idx.string(), val)
	return true
}

func (o *tinyObject) setOwnSym(name *Symbol, val Value, throw bool) bool {
	if proto := o.prototype; proto != nil {
		// we know it's foreign because prototype loops are not allowed
		if res, handled := proto.self.setForeignSym(name, val, o.val, throw); handled {
			return res
		}
	}
	// new property
	if !o.class.extensible {
		o.val.runtime.typeErrorResult(throw, "Cannot add property %s, object is not extensible", name)
		return false
	}

	o.deoptimize()._putSym(name, val)
	return true
}

func (o *tinyObject) setForeignStr(name unistring.String, val, receiver Value, throw bool) (bool, bool) {
	if idx := o.class.idxForName(name); idx == -1 {
		if proto := o.prototype; proto != nil {
			if receiver != proto {
				return proto.self.setForeignStr(name, val, receiver, throw)
			}
			return proto.self.setOwnStr(name, val, throw), true
		}
	}

	return false, false
}

func (o *tinyObject) setForeignIdx(name valueInt, val, receiver Value, throw bool) (bool, bool) {
	if idx := toIdx(name); idx != math.MaxUint32 {
		if proto := o.prototype; proto != nil {
			if receiver != proto {
				return proto.self.setForeignIdx(name, val, receiver, throw)
			}
			return proto.self.setOwnIdx(name, val, throw), true
		}
	}
	return o.setForeignStr(name.string(), val, receiver, throw)
}

func (o *tinyObject) setForeignSym(name *Symbol, val, receiver Value, throw bool) (bool, bool) {
	if proto := o.prototype; proto != nil {
		if receiver != proto {
			return proto.self.setForeignSym(name, val, receiver, throw)
		}
		return proto.self.setOwnSym(name, val, throw), true
	}
	return false, false
}

func (o *tinyObject) _put(name unistring.String, value Value, throw bool) bool {
	idx := o.class.idxForName(name)
	if idx == -1 {
		// new property
		return o._addProp(name, value, throw)
	}
	o.values[idx] = value
	return true
}

func (o *tinyObject) defineOwnPropertyStr(name unistring.String, descr PropertyDescriptor, throw bool) bool {
	if !descr.IsData() || descr.Writable != FLAG_TRUE || descr.Enumerable != FLAG_TRUE || descr.Configurable != FLAG_TRUE {
		return o.deoptimize().defineOwnPropertyStr(name, descr, throw)
	}
	return o._put(name, descr.Value, throw)
}

func (o *tinyObject) defineOwnPropertyIdx(name valueInt, descr PropertyDescriptor, throw bool) bool {
	if !descr.IsData() || descr.Writable != FLAG_TRUE || descr.Enumerable != FLAG_TRUE || descr.Configurable != FLAG_TRUE {
		return o.deoptimize().defineOwnPropertyIdx(name, descr, throw)
	}
	if n := toIdx(name); n != math.MaxUint32 {
		return o.deoptimize().defineOwnPropertyIdx(name, descr, throw)
	}
	return o._put(name.string(), descr.Value, throw)
}

func (o *tinyObject) defineOwnPropertySym(s *Symbol, descr PropertyDescriptor, throw bool) bool {
	return o.deoptimize().defineOwnPropertySym(s, descr, throw)
}

func (o *tinyObject) _putProp(name unistring.String, value Value, writable, enumerable, configurable bool) Value {
	if !writable || !enumerable || !configurable {
		return o.deoptimize()._putProp(name, value, writable, enumerable, configurable)
	}
	o._put(name, value, false)
	return value
}

func (o *tinyObject) _putSym(s *Symbol, prop Value) {
	o.deoptimize()._putSym(s, prop)
}

func (o *tinyObject) getPrivateEnv(typ *privateEnvType, create bool) *privateElements {
	return o.deoptimize().getPrivateEnv(typ, create)
}

func (o *tinyObject) assertCallable() (func(FunctionCall) Value, bool) {
	return nil, false
}

func (o *tinyObject) vmCall(vm *vm, _ int) {
	panic(vm.r.NewTypeError("Not a function: %s", o.val.toString()))
}

func (o *tinyObject) assertConstructor() func(args []Value, newTarget *Object) *Object {
	return nil
}

func (o *tinyObject) proto() *Object {
	return o.prototype
}

func (o *tinyObject) isExtensible() bool {
	return o.class.extensible
}

func (o *tinyObject) preventExtensions(bool) bool {
	if o.class.extensible {
		o.class = o.class.getNotExtensible()
	}
	return true
}

func (o *tinyObject) sortLen() int {
	return 0
}

func (o *tinyObject) sortGet(i int) Value {
	return nil
}

func (o *tinyObject) swap(int, int) {
}

func (o *tinyObject) stringKeys(_ bool, keys []Value) []Value {
	for _, k := range (*o.class.keys)[:o.class.keysLen] {
		keys = append(keys, stringValueFromRaw(k))
	}
	return keys
}

func (o *tinyObject) symbols(_ bool, accum []Value) []Value {
	return accum
}

func (o *tinyObject) keys(all bool, accum []Value) []Value {
	return o.stringKeys(all, accum)
}

func (o *tinyObject) hasInstance(Value) bool {
	panic(o.val.runtime.NewTypeError("Expecting a function in instanceof check, but got %s", o.val.toString()))
}

type tinyObjectPropIter struct {
	o    *tinyObject
	keys []unistring.String
	i    int
}

func (i *tinyObjectPropIter) next() (propIterItem, iterNextFunc) {
	if i.o.val.self != i.o { // object was deoptimised
		for i.i < len(i.keys) {
			name := i.keys[i.i]
			value := i.o.val.self.getOwnPropStr(name)
			i.i++
			if value != nil {
				return propIterItem{name: stringValueFromRaw(name), value: value}, i.next
			}
		}
		return propIterItem{}, nil
	}

	if i.i < len(i.keys) {
		name := i.keys[i.i]
		value := i.o.values[i.i]
		i.i++
		return propIterItem{name: stringValueFromRaw(name), value: value}, i.next
	}
	return propIterItem{}, nil
}

func (o *tinyObject) iterateStringKeys() iterNextFunc {
	return (&tinyObjectPropIter{
		o:    o,
		keys: (*o.class.keys)[:o.class.keysLen],
	}).next
}

func (o *tinyObject) iterateSymbols() iterNextFunc {
	return func() (propIterItem, iterNextFunc) {
		return propIterItem{}, nil
	}
}

func (o *tinyObject) iterateKeys() iterNextFunc {
	return o.iterateStringKeys()
}

func (o *tinyObject) equal(objectImpl) bool {
	// Rely on parent reference comparison
	return false
}

func (o *tinyObject) deleteStr(name unistring.String, throw bool) bool {
	idx := o.class.idxForName(name)
	if idx != -1 {
		if idx == int(o.class.keysLen-1) && o.class.parent.keysLen == o.class.keysLen-1 {
			o.class = o.class.parent
			o.values[idx] = nil
			o.values = o.values[:idx]
			return true
		}
		return o.deoptimize().deleteStr(name, throw)
	}
	return true
}

func (o *tinyObject) deleteIdx(idx valueInt, throw bool) bool {
	if toIdx(idx) == math.MaxUint32 {
		return o.deleteStr(idx.string(), throw)
	}
	return true
}

func (o *tinyObject) deleteSym(*Symbol, bool) bool {
	return true
}

func (o *tinyObject) setProto(proto *Object, throw bool) bool {
	if o.prototype.SameAs(proto) {
		return true
	}

	if !o.class.extensible {
		o.val.runtime.typeErrorResult(throw, "%s is not extensible", o.val)
		return false
	}
	for p := proto; p != nil; p = p.self.proto() {
		if p.SameAs(o.val) {
			o.val.runtime.typeErrorResult(throw, "Cyclic __proto__ value")
			return false
		}
		if _, ok := p.self.(*proxyObject); ok {
			break
		}
	}

	o.prototype = proto
	return true
}

func (o *tinyObject) export(ctx *objectExportCtx) any {
	if v, exists := ctx.get(o.val); exists {
		return v
	}
	keys := o.stringKeys(false, nil)
	m := make(map[string]any, len(keys))
	ctx.put(o.val, m)
	for _, itemName := range keys {
		itemNameStr := itemName.String()
		v := o.getStr(itemName.string(), nil)
		if v != nil {
			m[itemNameStr] = exportValue(v, ctx)
		} else {
			m[itemNameStr] = nil
		}
	}

	return m
}

func (o *tinyObject) exportType() reflect.Type {
	return reflectTypeMap
}

func (o *tinyObject) exportToMap(m reflect.Value, typ reflect.Type, ctx *objectExportCtx) error {
	return genericExportToMap(o.val, m, typ, ctx)
}

func (o *tinyObject) exportToArrayOrSlice(dst reflect.Value, typ reflect.Type, ctx *objectExportCtx) error {
	return genericExportToArrayOrSlice(o.val, dst, typ, ctx)
}
