package goja

import (
	"hash/maphash"
	"math"
	"unsafe"
)

var (
	mapHasher maphash.Hash
)

type mapEntry struct {
	key, value Value

	iterPrev, iterNext *mapEntry
	hNext              *mapEntry
}

type orderedMap struct {
	hash                map[uint64]*mapEntry
	iterFirst, iterLast *mapEntry
	size                int
}

type orderedMapIter struct {
	m   *orderedMap
	cur *mapEntry
}

type mapObject struct {
	baseObject
	m *orderedMap
}

func mapHash(v Value) uint64 {
	switch v := v.(type) {
	case valueUndefined, valueNull:
		return uint64(uintptr(unsafe.Pointer(&v)))
	case valueBool:
		if v {
			return uint64(uintptr(unsafe.Pointer(&valueTrue)))
		}
		return uint64(uintptr(unsafe.Pointer(&valueFalse)))
	case *valueSymbol:
		return uint64(uintptr(unsafe.Pointer(v)))
	case *Object:
		return uint64(uintptr(unsafe.Pointer(v)))
	case valueInt:
		return uint64(v)
	case valueFloat:
		if IsNaN(v) {
			return uint64(uintptr(unsafe.Pointer(&_NaN)))
		}
		if v == _negativeZero {
			return 0
		}
		return math.Float64bits(float64(v))
	case asciiString:
		_, _ = mapHasher.WriteString(string(v))
	case unicodeString:
		_, _ = mapHasher.Write(*(*[]byte)(unsafe.Pointer(&v)))
	}
	h := mapHasher.Sum64()
	mapHasher.Reset()
	return h
}

func (m *orderedMap) lookup(key Value) (h uint64, entry, hPrev *mapEntry) {
	if key == _negativeZero {
		key = intToValue(0)
	}
	h = mapHash(key)
	for entry = m.hash[h]; entry != nil && !entry.key.SameAs(key); hPrev, entry = entry, entry.hNext {
	}
	return
}

func (m *orderedMap) set(key, value Value) {
	h, entry, hPrev := m.lookup(key)
	if entry != nil {
		entry.value = value
	} else {
		entry = &mapEntry{key: key, value: value}
		if hPrev == nil {
			m.hash[h] = entry
		} else {
			hPrev.hNext = entry
		}
		if m.iterLast != nil {
			entry.iterPrev = m.iterLast
			m.iterLast.iterNext = entry
		} else {
			m.iterFirst = entry
		}
		m.iterLast = entry
		m.size++
	}
}

func (m *orderedMap) get(key Value) Value {
	_, entry, _ := m.lookup(key)
	if entry != nil {
		return entry.value
	}

	return nil
}

func (m *orderedMap) remove(key Value) bool {
	h, entry, hPrev := m.lookup(key)
	if entry != nil {
		entry.key = nil
		entry.value = nil

		// remove from the doubly-linked list
		if entry.iterPrev != nil {
			entry.iterPrev.iterNext = entry.iterNext
		} else {
			m.iterFirst = entry.iterNext
		}
		if entry.iterNext != nil {
			entry.iterNext.iterPrev = entry.iterPrev
		} else {
			m.iterLast = entry.iterPrev
		}

		// remove from the hash
		if hPrev == nil {
			delete(m.hash, h)
		} else {
			hPrev.hNext = entry.hNext
		}

		m.size--
		return true
	}

	return false
}

func (m *orderedMap) has(key Value) bool {
	_, entry, _ := m.lookup(key)
	return entry != nil
}

func (iter *orderedMapIter) next() *mapEntry {
	cur := iter.cur
	if cur != nil {
		for {
			iter.cur = iter.cur.iterNext
			if iter.cur == nil || iter.cur.key != nil {
				break
			}
		}
	}
	return cur
}

func (iter *orderedMapIter) close() {
	iter.cur = nil
}

func newOrderedMap() *orderedMap {
	return &orderedMap{
		hash: make(map[uint64]*mapEntry),
	}
}

func (m *orderedMap) newIter() *orderedMapIter {
	iter := &orderedMapIter{
		m:   m,
		cur: m.iterFirst,
	}
	return iter
}

func (mo *mapObject) init() {
	mo.baseObject.init()
	mo.m = newOrderedMap()
}

func (r *Runtime) mapProto_delete(call FunctionCall) Value {
	thisObj := r.toObject(call.This)
	mo, ok := thisObj.self.(*mapObject)
	if !ok {
		panic(r.NewTypeError("Method Map.prototype.delete called on incompatible receiver %s", thisObj.String()))
	}
	key, ok := call.Argument(0).(*Object)
	if ok && mo.m.remove(key) {
		return valueTrue
	}
	return valueFalse
}

func (r *Runtime) mapProto_get(call FunctionCall) Value {
	thisObj := r.toObject(call.This)
	mo, ok := thisObj.self.(*mapObject)
	if !ok {
		panic(r.NewTypeError("Method Map.prototype.get called on incompatible receiver %s", thisObj.String()))
	}
	var res Value
	if key, ok := call.Argument(0).(*Object); ok {
		res = mo.m.get(key)
	}
	if res == nil {
		return _undefined
	}
	return res
}

func (r *Runtime) mapProto_has(call FunctionCall) Value {
	thisObj := r.toObject(call.This)
	mo, ok := thisObj.self.(*mapObject)
	if !ok {
		panic(r.NewTypeError("Method Map.prototype.has called on incompatible receiver %s", thisObj.String()))
	}
	key, ok := call.Argument(0).(*Object)
	if ok && mo.m.has(key) {
		return valueTrue
	}
	return valueFalse
}

func (r *Runtime) mapProto_set(call FunctionCall) Value {
	thisObj := r.toObject(call.This)
	mo, ok := thisObj.self.(*mapObject)
	if !ok {
		panic(r.NewTypeError("Method Map.prototype.set called on incompatible receiver %s", thisObj.String()))
	}
	key := r.toObject(call.Argument(0))
	mo.m.set(key, call.Argument(1))
	return call.This
}
func (r *Runtime) mapProto_getSize(call FunctionCall) Value {
	thisObj := r.toObject(call.This)
	mo, ok := thisObj.self.(*mapObject)
	if !ok {
		panic(r.NewTypeError("Method get Map.prototype.size called on incompatible receiver %s", thisObj.String()))
	}
	return intToValue(int64(mo.m.size))
}

func (r *Runtime) builtin_newMap(args []Value) *Object {
	o := &Object{runtime: r}

	mo := &mapObject{}
	mo.class = classMap
	mo.val = o
	mo.extensible = true
	o.self = mo
	mo.prototype = r.global.MapPrototype
	mo.init()
	if len(args) > 0 {
		if arg := args[0]; arg != nil && arg != _undefined && arg != _null {
			adder := mo.getStr("set")
			iter := r.getIterator(arg.ToObject(r), nil)
			i0 := intToValue(0)
			i1 := intToValue(1)
			if adder == r.global.mapAdder {
				r.iterate(iter, func(item Value) {
					itemObj := r.toObject(item)
					k := itemObj.self.get(i0)
					v := itemObj.self.get(i1)
					mo.m.set(k, v)
				})
			} else {
				adderFn := toMethod(adder)
				if adderFn == nil {
					panic(r.NewTypeError("Map.set in missing"))
				}
				r.iterate(iter, func(item Value) {
					itemObj := r.toObject(item)
					k := itemObj.self.get(i0)
					v := itemObj.self.get(i1)
					adderFn(FunctionCall{This: o, Arguments: []Value{k, v}})
				})
			}
		}
	}
	return o
}

func (r *Runtime) createMapProto(val *Object) objectImpl {
	o := newBaseObjectObj(val, r.global.ObjectPrototype, classObject)

	o._putProp("constructor", r.global.Map, true, false, true)
	r.global.mapAdder = r.newNativeFunc(r.mapProto_set, nil, "set", nil, 2)
	o._putProp("set", r.global.mapAdder, true, false, true)
	o._putProp("delete", r.newNativeFunc(r.mapProto_delete, nil, "delete", nil, 1), true, false, true)
	o._putProp("has", r.newNativeFunc(r.mapProto_has, nil, "has", nil, 1), true, false, true)
	o._putProp("get", r.newNativeFunc(r.mapProto_get, nil, "get", nil, 1), true, false, true)
	o.putStr("size", &valueProperty{
		getterFunc:   r.newNativeFunc(r.mapProto_getSize, nil, "get size", nil, 0),
		accessor:     true,
		writable:     true,
		configurable: true,
	}, true)

	o.put(symToStringTag, valueProp(asciiString(classMap), false, false, true), true)

	return o
}

func (r *Runtime) createMap(val *Object) objectImpl {
	o := r.newNativeFuncObj(val, r.constructorThrower("Map"), r.builtin_newMap, "Map", r.global.MapPrototype, 0)

	return o
}

func (r *Runtime) initMap() {
	r.global.MapPrototype = r.newLazyObject(r.createMapProto)
	r.global.Map = r.newLazyObject(r.createMap)

	r.addToGlobal("Map", r.global.Map)
}
