package goja

import (
	"math"
	"reflect"
	"sort"
	"strconv"
)

type sparseArrayItem struct {
	idx   int64
	value Value
}

type sparseArrayObject struct {
	baseObject
	items          []sparseArrayItem
	length         int64
	propValueCount int
	lengthProp     valueProperty
}

func (a *sparseArrayObject) init() {
	a.baseObject.init()
	a.lengthProp.writable = true

	a._put("length", &a.lengthProp)
}

func (a *sparseArrayObject) findIdx(idx int64) int {
	return sort.Search(len(a.items), func(i int) bool {
		return a.items[i].idx >= idx
	})
}

func (a *sparseArrayObject) _setLengthInt(l int64, throw bool) bool {
	if l >= 0 && l <= math.MaxUint32 {
		ret := true

		if l <= a.length {
			if a.propValueCount > 0 {
				// Slow path
				for i := len(a.items) - 1; i >= 0; i-- {
					item := a.items[i]
					if item.idx <= l {
						break
					}
					if prop, ok := item.value.(*valueProperty); ok {
						if !prop.configurable {
							l = item.idx + 1
							ret = false
							break
						}
						a.propValueCount--
					}
				}
			}
		}

		idx := a.findIdx(l)

		aa := a.items[idx:]
		for i := range aa {
			aa[i].value = nil
		}
		a.items = a.items[:idx]
		a.length = l
		if !ret {
			a.val.runtime.typeErrorResult(throw, "Cannot redefine property: length")
		}
		return ret
	}
	panic(a.val.runtime.newError(a.val.runtime.global.RangeError, "Invalid array length"))
}

func (a *sparseArrayObject) setLengthInt(l int64, throw bool) bool {
	if l == a.length {
		return true
	}
	if !a.lengthProp.writable {
		a.val.runtime.typeErrorResult(throw, "length is not writable")
		return false
	}
	return a._setLengthInt(l, throw)
}

func (a *sparseArrayObject) setLength(v Value, throw bool) bool {
	l, ok := toIntIgnoreNegZero(v)
	if ok && l == a.length {
		return true
	}
	if !a.lengthProp.writable {
		a.val.runtime.typeErrorResult(throw, "length is not writable")
		return false
	}
	if ok {
		return a._setLengthInt(l, throw)
	}
	panic(a.val.runtime.newError(a.val.runtime.global.RangeError, "Invalid array length"))
}

func (a *sparseArrayObject) getIdx(idx int64) Value {
	i := a.findIdx(idx)
	if i < len(a.items) && a.items[i].idx == idx {
		return a.items[i].value
	}

	return nil
}

func (a *sparseArrayObject) get(p Value, receiver Value) Value {
	return a.getWithOwnProp(a.getOwnProp(p), p, receiver)
}

func (a *sparseArrayObject) getStr(name string, receiver Value) Value {
	return a.getStrWithOwnProp(a.getOwnPropStr(name), name, receiver)
}

func (a *sparseArrayObject) getOwnProp(n Value) Value {
	if s, ok := n.(*valueSymbol); ok {
		return a.getOwnPropSym(s)
	}
	if idx := toIdx(n); idx >= 0 {
		return a.getIdx(idx)
	}
	s := n.String()
	if s == "length" {
		return a.getLengthProp()
	}

	return a.baseObject.getOwnPropStr(s)
}

func (a *sparseArrayObject) getLengthProp() Value {
	a.lengthProp.value = intToValue(a.length)
	return &a.lengthProp
}

func (a *sparseArrayObject) getOwnPropStr(name string) Value {
	if idx := strToIdx(name); idx >= 0 {
		i := a.findIdx(idx)
		if i < len(a.items) && a.items[i].idx == idx {
			return a.items[i].value
		}
		return nil
	}
	if name == "length" {
		return a.getLengthProp()
	}
	return a.baseObject.getOwnPropStr(name)
}

func (a *sparseArrayObject) add(idx int64, val Value) {
	i := a.findIdx(idx)
	a.items = append(a.items, sparseArrayItem{})
	copy(a.items[i+1:], a.items[i:])
	a.items[i] = sparseArrayItem{
		idx:   idx,
		value: val,
	}
}

func (a *sparseArrayObject) setIdx(idx int64, val Value, throw bool, origNameStr string, origName Value) {
	var prop Value
	i := a.findIdx(idx)
	if i < len(a.items) && a.items[i].idx == idx {
		prop = a.items[i].value
	}

	if prop == nil {
		if proto := a.prototype; proto != nil {
			// we know it's foreign because prototype loops are not allowed
			var b bool
			if origName != nil {
				b = proto.self.setForeign(origName, val, a.val, throw)
			} else {
				b = proto.self.setForeignStr(origNameStr, val, a.val, throw)
			}
			if b {
				return
			}
		}

		// new property
		if !a.extensible {
			a.val.runtime.typeErrorResult(throw)
			return
		}

		if idx >= a.length {
			if !a.setLengthInt(idx+1, throw) {
				return
			}
		}

		if a.expand(idx) {
			a.items = append(a.items, sparseArrayItem{})
			copy(a.items[i+1:], a.items[i:])
			a.items[i] = sparseArrayItem{
				idx:   idx,
				value: val,
			}
		} else {
			ar := a.val.self.(*arrayObject)
			ar.values[idx] = val
			ar.objCount++
			return
		}
	} else {
		if prop, ok := prop.(*valueProperty); ok {
			if !prop.isWritable() {
				a.val.runtime.typeErrorResult(throw)
				return
			}
			prop.set(a.val, val)
		} else {
			a.items[i].value = val
		}
	}
}

func (a *sparseArrayObject) setOwn(n Value, val Value, throw bool) {
	if s, ok := n.(*valueSymbol); ok {
		a.setOwnSym(s, val, throw)
		return
	}
	if idx := toIdx(n); idx >= 0 {
		a.setIdx(idx, val, throw, "", n)
	} else {
		name := n.String()
		if name == "length" {
			a.setLength(val, throw)
		} else {
			a.baseObject.setOwnStr(name, val, throw)
		}
	}
}

func (a *sparseArrayObject) setOwnStr(name string, val Value, throw bool) {
	if idx := strToIdx(name); idx >= 0 {
		a.setIdx(idx, val, throw, name, nil)
	} else {
		if name == "length" {
			a.setLength(val, throw)
		} else {
			a.baseObject.setOwnStr(name, val, throw)
		}
	}
}

func (a *sparseArrayObject) setForeign(name Value, val, receiver Value, throw bool) bool {
	return a._setForeign(name, a.getOwnProp(name), val, receiver, throw)
}

func (a *sparseArrayObject) setForeignStr(name string, val, receiver Value, throw bool) bool {
	return a._setForeignStr(name, a.getOwnPropStr(name), val, receiver, throw)
}

type sparseArrayPropIter struct {
	a         *sparseArrayObject
	recursive bool
	idx       int
}

func (i *sparseArrayPropIter) next() (propIterItem, iterNextFunc) {
	for i.idx < len(i.a.items) {
		name := strconv.Itoa(int(i.a.items[i.idx].idx))
		prop := i.a.items[i.idx].value
		i.idx++
		if prop != nil {
			return propIterItem{name: name, value: prop}, i.next
		}
	}

	return i.a.baseObject._enumerate(i.recursive)()
}

func (a *sparseArrayObject) _enumerate(recursive bool) iterNextFunc {
	return (&sparseArrayPropIter{
		a:         a,
		recursive: recursive,
	}).next
}

func (a *sparseArrayObject) enumerate(all, recursive bool) iterNextFunc {
	return (&propFilterIter{
		wrapped: a._enumerate(recursive),
		all:     all,
		seen:    make(map[string]bool),
	}).next
}

func (a *sparseArrayObject) setValues(values []Value, objCount int64) {
	a.items = make([]sparseArrayItem, 0, objCount)
	for i, val := range values {
		if val != nil {
			a.items = append(a.items, sparseArrayItem{
				idx:   int64(i),
				value: val,
			})
		}
	}
}

func (a *sparseArrayObject) hasOwnProperty(n Value) bool {
	if s, ok := n.(*valueSymbol); ok {
		return a.hasSym(s)
	}
	if idx := toIdx(n); idx >= 0 {
		i := a.findIdx(idx)
		if i < len(a.items) && a.items[i].idx == idx {
			return a.items[i].value != _undefined
		}
		return false
	} else {
		return a.baseObject.hasOwnProperty(n)
	}
}

func (a *sparseArrayObject) hasOwnPropertyStr(name string) bool {
	if idx := strToIdx(name); idx >= 0 {
		i := a.findIdx(idx)
		if i < len(a.items) && a.items[i].idx == idx {
			return a.items[i].value != _undefined
		}
		return false
	} else {
		return a.baseObject.hasOwnPropertyStr(name)
	}
}

func (a *sparseArrayObject) expand(idx int64) bool {
	if l := len(a.items); l >= 1024 {
		if ii := a.items[l-1].idx; ii > idx {
			idx = ii
		}
		if int(idx)>>3 < l {
			//log.Println("Switching sparse->standard")
			ar := &arrayObject{
				baseObject:     a.baseObject,
				length:         a.length,
				propValueCount: a.propValueCount,
			}
			ar.setValuesFromSparse(a.items, idx)
			ar.val.self = ar
			ar.init()
			ar.lengthProp.writable = a.lengthProp.writable
			return false
		}
	}
	return true
}

func (a *sparseArrayObject) defineOwnProperty(n Value, descr PropertyDescriptor, throw bool) bool {
	if idx := toIdx(n); idx >= 0 {
		var existing Value
		i := a.findIdx(idx)
		if i < len(a.items) && a.items[i].idx == idx {
			existing = a.items[i].value
		}
		prop, ok := a.baseObject._defineOwnProperty(n.String(), existing, descr, throw)
		if ok {
			if idx >= a.length {
				if !a.setLengthInt(idx+1, throw) {
					return false
				}
			}
			if i >= len(a.items) || a.items[i].idx != idx {
				if a.expand(idx) {
					a.items = append(a.items, sparseArrayItem{})
					copy(a.items[i+1:], a.items[i:])
					a.items[i] = sparseArrayItem{
						idx:   idx,
						value: prop,
					}
					if idx >= a.length {
						a.length = idx + 1
					}
				} else {
					return a.val.self.defineOwnProperty(n, descr, throw)
				}
			} else {
				a.items[i].value = prop
			}
			if _, ok := prop.(*valueProperty); ok {
				a.propValueCount++
			}
		}
		return ok
	} else {
		if n.String() == "length" {
			return a.val.runtime.defineArrayLength(&a.lengthProp, descr, a.setLength, throw)
		}
		return a.baseObject.defineOwnProperty(n, descr, throw)
	}
}

func (a *sparseArrayObject) _deleteProp(idx int64, throw bool) bool {
	i := a.findIdx(idx)
	if i < len(a.items) && a.items[i].idx == idx {
		if p, ok := a.items[i].value.(*valueProperty); ok {
			if !p.configurable {
				a.val.runtime.typeErrorResult(throw, "Cannot delete property '%d' of %s", idx, a.val.toString())
				return false
			}
			a.propValueCount--
		}
		copy(a.items[i:], a.items[i+1:])
		a.items[len(a.items)-1].value = nil
		a.items = a.items[:len(a.items)-1]
	}
	return true
}

func (a *sparseArrayObject) delete(n Value, throw bool) bool {
	if idx := toIdx(n); idx >= 0 {
		return a._deleteProp(idx, throw)
	}
	return a.baseObject.delete(n, throw)
}

func (a *sparseArrayObject) deleteStr(name string, throw bool) bool {
	if idx := strToIdx(name); idx >= 0 {
		return a._deleteProp(idx, throw)
	}
	return a.baseObject.deleteStr(name, throw)
}

func (a *sparseArrayObject) sortLen() int64 {
	if len(a.items) > 0 {
		return a.items[len(a.items)-1].idx + 1
	}

	return 0
}

func (a *sparseArrayObject) sortGet(i int64) Value {
	idx := a.findIdx(i)
	if idx < len(a.items) && a.items[idx].idx == i {
		v := a.items[idx].value
		if p, ok := v.(*valueProperty); ok {
			v = p.get(a.val)
		}
		return v
	}
	return nil
}

func (a *sparseArrayObject) swap(i, j int64) {
	idxI := a.findIdx(i)
	idxJ := a.findIdx(j)

	if idxI < len(a.items) && a.items[idxI].idx == i && idxJ < len(a.items) && a.items[idxJ].idx == j {
		a.items[idxI].value, a.items[idxJ].value = a.items[idxJ].value, a.items[idxI].value
	}
}

func (a *sparseArrayObject) export() interface{} {
	arr := make([]interface{}, a.length)
	for _, item := range a.items {
		if item.value != nil {
			arr[item.idx] = item.value.Export()
		}
	}
	return arr
}

func (a *sparseArrayObject) exportType() reflect.Type {
	return reflectTypeArray
}
