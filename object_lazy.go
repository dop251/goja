package goja

import "reflect"

type lazyObject struct {
	val    *Object
	create func(*Object) objectImpl
}

func (o *lazyObject) className() string {
	obj := o.create(o.val)
	o.val.self = obj
	return obj.className()
}

func (o *lazyObject) get(n Value, receiver Value) Value {
	obj := o.create(o.val)
	o.val.self = obj
	return obj.get(n, receiver)
}

func (o *lazyObject) getStr(name string, receiver Value) Value {
	obj := o.create(o.val)
	o.val.self = obj
	return obj.getStr(name, receiver)
}

func (o *lazyObject) getOwnPropStr(name string) Value {
	obj := o.create(o.val)
	o.val.self = obj
	return obj.getOwnPropStr(name)
}

func (o *lazyObject) getOwnProp(name Value) Value {
	obj := o.create(o.val)
	o.val.self = obj
	return obj.getOwnProp(name)
}

func (o *lazyObject) setOwn(p Value, v Value, throw bool) {
	obj := o.create(o.val)
	o.val.self = obj
	obj.setOwn(p, v, throw)
}

func (o *lazyObject) setForeign(p Value, v, receiver Value, throw bool) bool {
	obj := o.create(o.val)
	o.val.self = obj
	return obj.setForeign(p, v, receiver, throw)
}

func (o *lazyObject) setOwnStr(p string, v Value, throw bool) {
	obj := o.create(o.val)
	o.val.self = obj
	obj.setOwnStr(p, v, throw)
}

func (o *lazyObject) setForeignStr(p string, v, receiver Value, throw bool) bool {
	obj := o.create(o.val)
	o.val.self = obj
	return obj.setForeignStr(p, v, receiver, throw)
}

func (o *lazyObject) setOwnSym(p *valueSymbol, v Value, throw bool) {
	obj := o.create(o.val)
	o.val.self = obj
	obj.setOwnSym(p, v, throw)
}

func (o *lazyObject) setForeignSym(p *valueSymbol, v, receiver Value, throw bool) bool {
	obj := o.create(o.val)
	o.val.self = obj
	return obj.setForeignSym(p, v, receiver, throw)
}

func (o *lazyObject) hasProperty(n Value) bool {
	obj := o.create(o.val)
	o.val.self = obj
	return obj.hasProperty(n)
}

func (o *lazyObject) hasPropertyStr(name string) bool {
	obj := o.create(o.val)
	o.val.self = obj
	return obj.hasPropertyStr(name)
}

func (o *lazyObject) hasOwnProperty(n Value) bool {
	obj := o.create(o.val)
	o.val.self = obj
	return obj.hasOwnProperty(n)
}

func (o *lazyObject) hasOwnPropertyStr(name string) bool {
	obj := o.create(o.val)
	o.val.self = obj
	return obj.hasOwnPropertyStr(name)
}

func (o *lazyObject) _putProp(string, Value, bool, bool, bool) Value {
	panic("cannot use _putProp() in lazy object")
}

func (o *lazyObject) _putSym(*valueSymbol, Value) {
	panic("cannot use _putSym() in lazy object")
}

func (o *lazyObject) defineOwnProperty(name Value, descr PropertyDescriptor, throw bool) bool {
	obj := o.create(o.val)
	o.val.self = obj
	return obj.defineOwnProperty(name, descr, throw)
}

func (o *lazyObject) toPrimitiveNumber() Value {
	obj := o.create(o.val)
	o.val.self = obj
	return obj.toPrimitiveNumber()
}

func (o *lazyObject) toPrimitiveString() Value {
	obj := o.create(o.val)
	o.val.self = obj
	return obj.toPrimitiveString()
}

func (o *lazyObject) toPrimitive() Value {
	obj := o.create(o.val)
	o.val.self = obj
	return obj.toPrimitive()
}

func (o *lazyObject) assertCallable() (call func(FunctionCall) Value, ok bool) {
	obj := o.create(o.val)
	o.val.self = obj
	return obj.assertCallable()
}

func (o *lazyObject) assertConstructor() func(args []Value, newTarget Value) *Object {
	obj := o.create(o.val)
	o.val.self = obj
	return obj.assertConstructor()
}

func (o *lazyObject) deleteStr(name string, throw bool) bool {
	obj := o.create(o.val)
	o.val.self = obj
	return obj.deleteStr(name, throw)
}

func (o *lazyObject) delete(name Value, throw bool) bool {
	obj := o.create(o.val)
	o.val.self = obj
	return obj.delete(name, throw)
}

func (o *lazyObject) proto() *Object {
	obj := o.create(o.val)
	o.val.self = obj
	return obj.proto()
}

func (o *lazyObject) hasInstance(v Value) bool {
	obj := o.create(o.val)
	o.val.self = obj
	return obj.hasInstance(v)
}

func (o *lazyObject) isExtensible() bool {
	obj := o.create(o.val)
	o.val.self = obj
	return obj.isExtensible()
}

func (o *lazyObject) preventExtensions(throw bool) bool {
	obj := o.create(o.val)
	o.val.self = obj
	return obj.preventExtensions(throw)
}

func (o *lazyObject) enumerateUnfiltered() iterNextFunc {
	obj := o.create(o.val)
	o.val.self = obj
	return obj.enumerateUnfiltered()
}

func (o *lazyObject) enumerate() iterNextFunc {
	obj := o.create(o.val)
	o.val.self = obj
	return obj.enumerate()
}

func (o *lazyObject) export() interface{} {
	obj := o.create(o.val)
	o.val.self = obj
	return obj.export()
}

func (o *lazyObject) exportType() reflect.Type {
	obj := o.create(o.val)
	o.val.self = obj
	return obj.exportType()
}

func (o *lazyObject) equal(other objectImpl) bool {
	obj := o.create(o.val)
	o.val.self = obj
	return obj.equal(other)
}

func (o *lazyObject) ownKeys(all bool, accum []Value) []Value {
	obj := o.create(o.val)
	o.val.self = obj
	return obj.ownKeys(all, accum)
}

func (o *lazyObject) ownSymbols() []Value {
	obj := o.create(o.val)
	o.val.self = obj
	return obj.ownSymbols()
}

func (o *lazyObject) ownPropertyKeys(all bool, accum []Value) []Value {
	obj := o.create(o.val)
	o.val.self = obj
	return obj.ownPropertyKeys(all, accum)
}

func (o *lazyObject) setProto(proto *Object, throw bool) bool {
	obj := o.create(o.val)
	o.val.self = obj
	return obj.setProto(proto, throw)
}

func (o *lazyObject) sortLen() int64 {
	obj := o.create(o.val)
	o.val.self = obj
	return obj.sortLen()
}

func (o *lazyObject) sortGet(i int64) Value {
	obj := o.create(o.val)
	o.val.self = obj
	return obj.sortGet(i)
}

func (o *lazyObject) swap(i, j int64) {
	obj := o.create(o.val)
	o.val.self = obj
	obj.swap(i, j)
}
