package goja

import "reflect"

type objectGoMapReflect struct {
	objectGoReflect

	keyType, valueType reflect.Type
}

func (o *objectGoMapReflect) init() {
	o.objectGoReflect.init()
	o.keyType = o.value.Type().Key()
	o.valueType = o.value.Type().Elem()
}

func (o *objectGoMapReflect) toKey(n Value, throw bool) reflect.Value {
	key, err := o.val.runtime.toReflectValue(n, o.keyType)
	if err != nil {
		o.val.runtime.typeErrorResult(throw, "map key conversion error: %v", err)
		return reflect.Value{}
	}
	return key
}

func (o *objectGoMapReflect) strToKey(name string, throw bool) reflect.Value {
	if o.keyType.Kind() == reflect.String {
		return reflect.ValueOf(name).Convert(o.keyType)
	}
	return o.toKey(newStringValue(name), throw)
}

func (o *objectGoMapReflect) _get(n Value) Value {
	key := o.toKey(n, false)
	if !key.IsValid() {
		return nil
	}
	if v := o.value.MapIndex(key); v.IsValid() {
		return o.val.runtime.ToValue(v.Interface())
	}

	return nil
}

func (o *objectGoMapReflect) _getStr(name string) Value {
	key := o.strToKey(name, false)
	if !key.IsValid() {
		return nil
	}
	if v := o.value.MapIndex(key); v.IsValid() {
		return o.val.runtime.ToValue(v.Interface())
	}

	return nil
}

func (o *objectGoMapReflect) get(n Value, receiver Value) Value {
	if s, ok := n.(*valueSymbol); ok {
		return o.getSym(s, receiver)
	}
	if v := o._get(n); v != nil {
		return v
	}
	return o.objectGoReflect.getStr(n.String(), receiver)
}

func (o *objectGoMapReflect) getStr(name string, receiver Value) Value {
	if v := o._getStr(name); v != nil {
		return v
	}
	return o.objectGoReflect.getStr(name, receiver)
}

func (o *objectGoMapReflect) getOwnPropStr(name string) Value {
	if v := o._getStr(name); v != nil {
		return &valueProperty{
			value:      v,
			writable:   true,
			enumerable: true,
		}
	}
	return o.objectGoReflect.getOwnPropStr(name)
}

func (o *objectGoMapReflect) getOwnProp(name Value) Value {
	if v := o._get(name); v != nil {
		return &valueProperty{
			value:      v,
			writable:   true,
			enumerable: true,
		}
	}
	return o.objectGoReflect.getOwnProp(name)
}

func (o *objectGoMapReflect) toValue(val Value, throw bool) (reflect.Value, bool) {
	v, err := o.val.runtime.toReflectValue(val, o.valueType)
	if err != nil {
		o.val.runtime.typeErrorResult(throw, "map value conversion error: %v", err)
		return reflect.Value{}, false
	}

	return v, true
}

func (o *objectGoMapReflect) setOwn(key, val Value, throw bool) {
	if s, ok := key.(*valueSymbol); ok {
		o.setOwnSym(s, val, throw)
		return
	}
	if s, ok := key.assertString(); ok {
		o.setOwnStr(s.String(), val, throw)
	} else {
		o._put(o.toKey(key, throw), val, throw)
	}
}

func (o *objectGoMapReflect) _put(key reflect.Value, val Value, throw bool) {
	if key.IsValid() {
		if o.extensible || o.value.MapIndex(key).IsValid() {
			v, ok := o.toValue(val, throw)
			if !ok {
				return
			}
			o.value.SetMapIndex(key, v)
		} else {
			o.val.runtime.typeErrorResult(throw, "Cannot set property %s, object is not extensible", key.String())
		}
	}
}

func (o *objectGoMapReflect) setOwnStr(name string, val Value, throw bool) {
	key := o.strToKey(name, false)
	if !key.IsValid() || !o.value.MapIndex(key).IsValid() {
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
			return
		} else {
			if throw && !key.IsValid() {
				o.strToKey(name, true)
				return
			}
		}
	}
	o._put(key, val, throw)
}

func (o *objectGoMapReflect) setForeign(name Value, val, receiver Value, throw bool) bool {
	return o._setForeign(name, o.getOwnProp(name), val, receiver, throw)
}

func (o *objectGoMapReflect) setForeignStr(name string, val, receiver Value, throw bool) bool {
	return o._setForeignStr(name, trueValIfPresent(o.hasOwnPropertyStr(name)), val, receiver, throw)
}

func (o *objectGoMapReflect) defineOwnProperty(n Value, descr PropertyDescriptor, throw bool) bool {
	if s, ok := n.(*valueSymbol); ok {
		return o.defineOwnPropertySym(s, descr, throw)
	}
	if !o.val.runtime.checkHostObjectPropertyDescr(n, descr, throw) {
		return false
	}

	k := o.toKey(n, throw)
	if !k.IsValid() {
		return false
	}
	if o.extensible || o.value.MapIndex(k).IsValid() {
		v, ok := o.toValue(descr.Value, throw)
		if !ok {
			return false
		}
		o.value.SetMapIndex(k, v)
		return true
	}
	o.val.runtime.typeErrorResult(throw, "Cannot define property %s, object is not extensible", n.String())
	return false
}

func (o *objectGoMapReflect) hasOwnPropertyStr(name string) bool {
	key := o.strToKey(name, false)
	if key.IsValid() && o.value.MapIndex(key).IsValid() {
		return true
	}
	return false
}

func (o *objectGoMapReflect) hasOwnProperty(n Value) bool {
	if s, ok := n.(*valueSymbol); ok {
		return o.hasOwnSym(s)
	}
	if s, ok := n.assertString(); ok {
		return o.hasOwnPropertyStr(s.String())
	}

	key := o.toKey(n, false)
	if !key.IsValid() {
		return false
	}

	return o.value.MapIndex(key).IsValid()
}

func (o *objectGoMapReflect) delete(n Value, throw bool) bool {
	if s, ok := n.(*valueSymbol); ok {
		return o.deleteSym(s, throw)
	}

	key := o.toKey(n, throw)
	if !key.IsValid() {
		return false
	}
	o.value.SetMapIndex(key, reflect.Value{})
	return true
}

func (o *objectGoMapReflect) deleteStr(name string, throw bool) bool {
	key := o.strToKey(name, throw)
	if !key.IsValid() {
		return false
	}
	o.value.SetMapIndex(key, reflect.Value{})
	return true
}

type gomapReflectPropIter struct {
	o    *objectGoMapReflect
	keys []reflect.Value
	idx  int
}

func (i *gomapReflectPropIter) next() (propIterItem, iterNextFunc) {
	for i.idx < len(i.keys) {
		key := i.keys[i.idx]
		v := i.o.value.MapIndex(key)
		i.idx++
		if v.IsValid() {
			return propIterItem{name: key.String(), enumerable: _ENUM_TRUE}, i.next
		}
	}

	if i.o.prototype != nil {
		return i.o.prototype.self.enumerateUnfiltered()()
	}
	return propIterItem{}, nil
}

func (o *objectGoMapReflect) enumerateUnfiltered() iterNextFunc {
	return (&gomapReflectPropIter{
		o:    o,
		keys: o.value.MapKeys(),
	}).next
}

func (o *objectGoMapReflect) ownKeys(_ bool, accum []Value) []Value {
	// all own keys are enumerable
	for _, key := range o.value.MapKeys() {
		accum = append(accum, newStringValue(key.String()))
	}

	return accum
}

func (o *objectGoMapReflect) equal(other objectImpl) bool {
	if other, ok := other.(*objectGoMapReflect); ok {
		return o.value.Interface() == other.value.Interface()
	}
	return false
}
