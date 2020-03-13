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

func (o *objectGoMapReflect) getProp(n Value) Value {
	if v := o._get(n); v != nil {
		return v
	}
	return o.objectGoReflect.getProp(n)
}

func (o *objectGoMapReflect) getPropStr(name string) Value {
	if v := o._getStr(name); v != nil {
		return v
	}
	return o.objectGoReflect.getPropStr(name)
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

func (o *objectGoMapReflect) _put(key, val Value, throw bool) {
	k := o.toKey(key, throw)
	v, ok := o.toValue(val, throw)
	if !ok {
		return
	}
	o.value.SetMapIndex(k, v)
}

func (o *objectGoMapReflect) put(key, val Value, throw bool) {
	if s, ok := key.(*valueSymbol); ok {
		o.putSym(s, val, throw)
		return
	}
	if s, ok := key.assertString(); ok {
		o.putStr(s.String(), val, throw)
		return
	}
	if !o.extensible {
		o.val.runtime.typeErrorResult(throw, "Cannot set property %s, object is not extensible", key.String())
		return
	}
	o._put(key, val, throw)
}

func (o *objectGoMapReflect) putStr(name string, val Value, throw bool) {
	k := o.strToKey(name, throw)
	if k.IsValid() && o.value.MapIndex(k).IsValid() || !o.protoPut(name, val, throw) {
		if !k.IsValid() {
			o.val.runtime.typeErrorResult(throw, "GoMapReflect: invalid key: '%s'")
			return
		}
		v, ok := o.toValue(val, throw)
		if !ok {
			return
		}
		o.value.SetMapIndex(k, v)
	}
}

func (o *objectGoMapReflect) _putProp(name string, value Value, writable, enumerable, configurable bool) Value {
	o.putStr(name, value, true)
	return value
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
	if !key.IsValid() {
		return false
	}
	return o.value.MapIndex(key).IsValid()
}

func (o *objectGoMapReflect) hasOwnProperty(n Value) bool {
	if s, ok := n.(*valueSymbol); ok {
		_, exists := o.symValues[s]
		return exists
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
	o         *objectGoMapReflect
	keys      []reflect.Value
	idx       int
	recursive bool
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

	if i.recursive {
		return i.o.objectGoReflect._enumerate(true)()
	}

	return propIterItem{}, nil
}

func (o *objectGoMapReflect) _enumerate(recursive bool) iterNextFunc {
	r := &gomapReflectPropIter{
		o:         o,
		keys:      o.value.MapKeys(),
		recursive: recursive,
	}
	return r.next
}

func (o *objectGoMapReflect) enumerate(all, recursive bool) iterNextFunc {
	return (&propFilterIter{
		wrapped: o._enumerate(recursive),
		all:     all,
		seen:    make(map[string]bool),
	}).next
}

func (o *objectGoMapReflect) equal(other objectImpl) bool {
	if other, ok := other.(*objectGoMapReflect); ok {
		return o.value.Interface() == other.value.Interface()
	}
	return false
}
