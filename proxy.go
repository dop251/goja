package goja

import (
	"fmt"
)

type Proxy struct {
	proxy *Object
}

type iteratorPropIter struct {
	iter *Object
}

func (i *iteratorPropIter) next() (propIterItem, iterNextFunc) {
	res := i.iter.runtime.toObject(toMethod(i.iter.self.getStr("next", nil))(FunctionCall{This: i.iter}))
	if nilSafe(res.self.getStr("done", nil)).ToBoolean() {
		return propIterItem{}, nil
	}
	return propIterItem{name: nilSafe(res.self.getStr("value", nil)).String()}, i.next
}

func (r *Runtime) NewProxy(target *Object, nativeHandler *ProxyTrapConfig, revocable, strict bool) *Proxy {
	handler := r.newNativeProxyHandler(nativeHandler)
	proxy := r.newProxyObject(target, handler, revocable, strict)
	return &Proxy{proxy}
}

func (p *Proxy) Revoke() {
	proxy := p.proxy.self.(*proxyObject)
	proxy.revoked = true
}

func (r *Runtime) newNativeProxyHandler(nativeHandler *ProxyTrapConfig) *Object {
	handler := r.NewObject()
	r.proxyproto_nativehandler_getPrototypeOf(nativeHandler.GetPrototypeOf, handler)
	r.proxyproto_nativehandler_setPrototypeOf(nativeHandler.SetPrototypeOf, handler)
	r.proxyproto_nativehandler_isExtensible(nativeHandler.IsExtensible, handler)
	r.proxyproto_nativehandler_preventExtensions(nativeHandler.PreventExtensions, handler)
	r.proxyproto_nativehandler_getOwnPropertyDescriptor(nativeHandler.GetOwnPropertyDescriptor, handler)
	r.proxyproto_nativehandler_defineProperty(nativeHandler.DefineProperty, handler)
	r.proxyproto_nativehandler_has(nativeHandler.Has, handler)
	r.proxyproto_nativehandler_get(nativeHandler.Get, handler)
	r.proxyproto_nativehandler_set(nativeHandler.Set, handler)
	r.proxyproto_nativehandler_deleteProperty(nativeHandler.DeleteProperty, handler)
	r.proxyproto_nativehandler_ownKeys(nativeHandler.OwnKeys, handler)
	r.proxyproto_nativehandler_apply(nativeHandler.Apply, handler)
	r.proxyproto_nativehandler_construct(nativeHandler.Construct, handler)
	return handler
}

func (r *Runtime) proxyproto_nativehandler_getPrototypeOf(native func(*Object) *Object, handler *Object) {
	if native != nil {
		handler.Set("getPrototypeOf", func(call FunctionCall) Value {
			if len(call.Arguments) >= 1 {
				if t, ok := call.Argument(0).(*Object); ok {
					return native(t)
				}
			}
			r.typeErrorResult(true, "getPrototypeOf needs to be called with target as Object")
			panic("Unreachable")
		})
	}
}

func (r *Runtime) proxyproto_nativehandler_setPrototypeOf(native func(*Object, *Object) bool, handler *Object) {
	if native != nil {
		handler.Set("setPrototypeOf", func(call FunctionCall) Value {
			if len(call.Arguments) >= 2 {
				if t, ok := call.Argument(0).(*Object); ok {
					if p, ok := call.Argument(1).(*Object); ok {
						s := native(t, p)
						return r.ToValue(s)
					}
				}
			}
			r.typeErrorResult(true, "setPrototypeOf needs to be called with target and prototype as Object")
			panic("Unreachable")
		})
	}
}

func (r *Runtime) proxyproto_nativehandler_isExtensible(native func(*Object) bool, handler *Object) {
	if native != nil {
		handler.Set("isExtensible", func(call FunctionCall) Value {
			if len(call.Arguments) >= 1 {
				if t, ok := call.Argument(0).(*Object); ok {
					s := native(t)
					return r.ToValue(s)
				}
			}
			r.typeErrorResult(true, "isExtensible needs to be called with target as Object")
			panic("Unreachable")
		})
	}
}

func (r *Runtime) proxyproto_nativehandler_preventExtensions(native func(*Object) bool, handler *Object) {
	if native != nil {
		handler.Set("preventExtensions", func(call FunctionCall) Value {
			if len(call.Arguments) >= 1 {
				if t, ok := call.Argument(0).(*Object); ok {
					s := native(t)
					return r.ToValue(s)
				}
			}
			r.typeErrorResult(true, "preventExtensions needs to be called with target as Object")
			panic("Unreachable")
		})
	}
}

func (r *Runtime) proxyproto_nativehandler_getOwnPropertyDescriptor(native func(*Object, string) PropertyDescriptor, handler *Object) {
	if native != nil {
		handler.Set("getOwnPropertyDescriptor", func(call FunctionCall) Value {
			if len(call.Arguments) >= 2 {
				if t, ok := call.Argument(0).(*Object); ok {
					if p, ok := call.Argument(1).assertString(); ok {
						return native(t, p.String()).toValue(r)
					}
				}
			}
			r.typeErrorResult(true, "getOwnPropertyDescriptor needs to be called with target as Object and prop as string")
			panic("Unreachable")
		})
	}
}

func (r *Runtime) proxyproto_nativehandler_defineProperty(native func(*Object, string, PropertyDescriptor) bool, handler *Object) {
	if native != nil {
		handler.Set("defineProperty", func(call FunctionCall) Value {
			if len(call.Arguments) >= 3 {
				if t, ok := call.Argument(0).(*Object); ok {
					if k, ok := call.Argument(1).assertString(); ok {
						propertyDescriptor := r.toPropertyDescriptor(call.Argument(2))
						s := native(t, k.String(), propertyDescriptor)
						return r.ToValue(s)
					}
				}
			}
			r.typeErrorResult(true, "defineProperty needs to be called with target as Object and propertyDescriptor as string and key as string")
			panic("Unreachable")
		})
	}
}

func (r *Runtime) proxyproto_nativehandler_has(native func(*Object, string) bool, handler *Object) {
	if native != nil {
		handler.Set("has", func(call FunctionCall) Value {
			if len(call.Arguments) >= 2 {
				if t, ok := call.Argument(0).(*Object); ok {
					if p, ok := call.Argument(1).assertString(); ok {
						o := native(t, p.String())
						return r.ToValue(o)
					}
				}
			}
			r.typeErrorResult(true, "has needs to be called with target as Object and property as string")
			panic("Unreachable")
		})
	}
}

func (r *Runtime) proxyproto_nativehandler_get(native func(*Object, string, *Object) Value, handler *Object) {
	if native != nil {
		handler.Set("get", func(call FunctionCall) Value {
			if len(call.Arguments) >= 2 {
				if t, ok := call.Argument(0).(*Object); ok {
					if p, ok := call.Argument(1).assertString(); ok {
						if r, ok := call.Argument(2).(*Object); ok {
							return native(t, p.String(), r)
						}
					}
				}
			}
			r.typeErrorResult(true, "get needs to be called with target and receiver as Object and property as string")
			panic("Unreachable")
		})
	}
}

func (r *Runtime) proxyproto_nativehandler_set(native func(*Object, string, Value, *Object) bool, handler *Object) {
	if native != nil {
		handler.Set("set", func(call FunctionCall) Value {
			if len(call.Arguments) >= 2 {
				if t, ok := call.Argument(0).(*Object); ok {
					if p, ok := call.Argument(1).assertString(); ok {
						v := call.Argument(2)
						if re, ok := call.Argument(3).(*Object); ok {
							s := native(t, p.String(), v, re)
							return r.ToValue(s)
						}
					}
				}
			}
			r.typeErrorResult(true, "set needs to be called with target and receiver as Object, property as string and value as a legal javascript value")
			panic("Unreachable")
		})
	}
}

func (r *Runtime) proxyproto_nativehandler_deleteProperty(native func(*Object, string) bool, handler *Object) {
	if native != nil {
		handler.Set("deleteProperty", func(call FunctionCall) Value {
			if len(call.Arguments) >= 2 {
				if t, ok := call.Argument(0).(*Object); ok {
					if p, ok := call.Argument(1).assertString(); ok {
						o := native(t, p.String())
						return r.ToValue(o)
					}
				}
			}
			r.typeErrorResult(true, "deleteProperty needs to be called with target as Object and property as string")
			panic("Unreachable")
		})
	}
}

func (r *Runtime) proxyproto_nativehandler_ownKeys(native func(*Object) *Object, handler *Object) {
	if native != nil {
		handler.Set("ownKeys", func(call FunctionCall) Value {
			if len(call.Arguments) >= 1 {
				if t, ok := call.Argument(0).(*Object); ok {
					return native(t)
				}
			}
			r.typeErrorResult(true, "ownKeys needs to be called with target as Object")
			panic("Unreachable")
		})
	}
}

func (r *Runtime) proxyproto_nativehandler_apply(native func(*Object, *Object, []Value) Value, handler *Object) {
	if native != nil {
		handler.Set("apply", func(call FunctionCall) Value {
			if len(call.Arguments) >= 3 {
				if t, ok := call.Argument(0).(*Object); ok {
					if this, ok := call.Argument(1).(*Object); ok {
						if v, ok := call.Argument(2).(*Object); ok {
							if a, ok := v.self.(*arrayObject); ok {
								v := native(t, this, a.values)
								return r.ToValue(v)
							}
						}
					}
				}
			}
			r.typeErrorResult(true, "apply needs to be called with target and this as Object and argumentsList as an array of legal javascript values")
			panic("Unreachable")
		})
	}
}

func (r *Runtime) proxyproto_nativehandler_construct(native func(*Object, []Value, *Object) *Object, handler *Object) {
	if native != nil {
		handler.Set("construct", func(call FunctionCall) Value {
			if len(call.Arguments) >= 2 {
				if t, ok := call.Argument(0).(*Object); ok {
					if v, ok := call.Argument(1).(*Object); ok {
						if newTarget, ok := call.Argument(2).(*Object); ok {
							if a, ok := v.self.(*arrayObject); ok {
								return native(t, a.values, newTarget)
							}
						}
					}
				}
			}
			r.typeErrorResult(true, "construct needs to be called with target and newTarget as Object and argumentsList as an array of legal javascript values")
			panic("Unreachable")
		})
	}
}

type ProxyTrapConfig struct {
	// A trap for Object.getPrototypeOf, Reflect.getPrototypeOf, __proto__, Object.prototype.isPrototypeOf, instanceof
	GetPrototypeOf func(target *Object) (prototype *Object)

	// A trap for Object.setPrototypeOf, Reflect.setPrototypeOf
	SetPrototypeOf func(target *Object, prototype *Object) (success bool)

	// A trap for Object.isExtensible, Reflect.isExtensible
	IsExtensible func(target *Object) (success bool)

	// A trap for Object.preventExtensions, Reflect.preventExtensions
	PreventExtensions func(target *Object) (success bool)

	// A trap for Object.getOwnPropertyDescriptor, Reflect.getOwnPropertyDescriptor
	GetOwnPropertyDescriptor func(target *Object, prop string) (propertyDescriptor PropertyDescriptor)

	// A trap for Object.defineProperty, Reflect.defineProperty
	DefineProperty func(target *Object, key string, propertyDescriptor PropertyDescriptor) (success bool)

	// A trap for the in operator, with operator, Reflect.has
	Has func(target *Object, property string) (available bool)

	// A trap for getting property values, Reflect.get
	Get func(target *Object, property string, receiver *Object) (value Value)

	// A trap for setting property values, Reflect.set
	Set func(target *Object, property string, value Value, receiver *Object) (success bool)

	// A trap for the delete operator, Reflect.deleteProperty
	DeleteProperty func(target *Object, property string) (success bool)

	// A trap for Object.getOwnPropertyNames, Object.getOwnPropertySymbols, Object.keys, Reflect.ownKeys
	OwnKeys func(target *Object) (object *Object)

	// A trap for a function call, Function.prototype.apply, Function.prototype.call, Reflect.apply
	Apply func(target *Object, this *Object, argumentsList []Value) (value Value)

	// A trap for the new operator, Reflect.construct
	Construct func(target *Object, argumentsList []Value, newTarget *Object) (value *Object)
}

type proxyTrap string

const (
	proxy_trap_getPrototypeOf           = "getPrototypeOf"
	proxy_trap_setPrototypeOf           = "setPrototypeOf"
	proxy_trap_isExtensible             = "isExtensible"
	proxy_trap_preventExtensions        = "preventExtensions"
	proxy_trap_getOwnPropertyDescriptor = "getOwnPropertyDescriptor"
	proxy_trap_defineProperty           = "defineProperty"
	proxy_trap_has                      = "has"
	proxy_trap_get                      = "get"
	proxy_trap_set                      = "set"
	proxy_trap_deleteProperty           = "deleteProperty"
	proxy_trap_ownKeys                  = "ownKeys"
	proxy_trap_apply                    = "apply"
	proxy_trap_construct                = "construct"
	proxy_trap_enumerate                = "enumerate"
)

func (p proxyTrap) String() (name string) {
	return string(p)
}

type proxyObject struct {
	baseObject
	target        *Object
	handler       *Object
	nativeHandler *ProxyTrapConfig
	revocable     bool
	revoked       bool
	strict        bool
}

func (p *proxyObject) handleProxyRequest(trap proxyTrap, proxyCallback func(proxyFunction func(FunctionCall) Value, this Value), targetCallback func(target *Object)) {
	runtime := p.val.runtime
	if p.revocable && p.revoked {
		panic(runtime.NewTypeError("Proxy already revoked"))
	}

	prop := p.handler.self.getOwnPropStr(trap.String())
	if prop == nil {
		// Redirect to target object
		targetCallback(p.target)
	} else {
		handler := prop.(*Object)
		f := runtime.toCallable(handler)
		proxyCallback(f, handler)
	}

}

func (p *proxyObject) proxyCall(trap proxyTrap, args ...Value) (Value, bool) {
	r := p.val.runtime
	if p.revocable && p.revoked {
		panic(r.NewTypeError("Proxy already revoked"))
	}

	if m := toMethod(r.getVStr(p.handler, trap.String())); m != nil {
		return m(FunctionCall{
			This:      p.handler,
			Arguments: args,
		}), true
	}

	return nil, false
}

func (p *proxyObject) proto() *Object {
	if v, ok := p.proxyCall(proxy_trap_getPrototypeOf, p.target); ok {
		var handlerProto *Object
		if v != _null {
			handlerProto = p.val.runtime.toObject(v)
		}
		if !p.target.self.isExtensible() && !p.__sameValue(handlerProto, p.target.self.proto()) {
			panic(p.val.runtime.NewTypeError("'getPrototypeOf' on proxy: proxy target is non-extensible but the trap did not return its actual prototype"))
		}
		return handlerProto
	}

	return p.target.self.proto()
}

func (p *proxyObject) setProto(proto *Object, throw bool) bool {
	if v, ok := p.proxyCall(proxy_trap_setPrototypeOf, p.target, proto); ok {
		if v.ToBoolean() {
			if !p.target.self.isExtensible() && !p.__sameValue(proto, p.target.self.proto()) {
				panic(p.val.runtime.NewTypeError("'setPrototypeOf' on proxy: trap returned truish for setting a new prototype on the non-extensible proxy target"))
			}
			return true
		} else {
			p.val.runtime.typeErrorResult(throw, "'setPrototypeOf' on proxy: trap returned falsish")
		}
	}

	return p.target.self.setProto(proto, throw)
}

func (p *proxyObject) isExtensible() bool {
	if v, ok := p.proxyCall(proxy_trap_isExtensible, p.target); ok {
		booleanTrapResult := v.ToBoolean()
		if te := p.target.self.isExtensible(); booleanTrapResult != te {
			panic(p.val.runtime.NewTypeError("'isExtensible' on proxy: trap result does not reflect extensibility of proxy target (which is '%v')", te))
		}
		return booleanTrapResult
	}

	return p.target.self.isExtensible()
}

func (p *proxyObject) preventExtensions(throw bool) bool {
	if v, ok := p.proxyCall(proxy_trap_preventExtensions, p.target); ok {
		booleanTrapResult := v.ToBoolean()
		if !booleanTrapResult {
			p.val.runtime.typeErrorResult(throw, "'preventExtensions' on proxy: trap returned falsish")
			return false
		}
		if te := p.target.self.isExtensible(); booleanTrapResult && te {
			panic(p.val.runtime.NewTypeError("'preventExtensions' on proxy: trap returned truish but the proxy target is extensible"))
		}
	}

	return p.target.self.preventExtensions(throw)
}

func propToValueProp(v Value) *valueProperty {
	if v == nil {
		return nil
	}
	if v, ok := v.(*valueProperty); ok {
		return v
	}
	return &valueProperty{
		value:        v,
		writable:     true,
		configurable: true,
		enumerable:   true,
	}
}

func (p *proxyObject) defineOwnProperty(name Value, descr PropertyDescriptor, throw bool) bool {
	if v, ok := p.proxyCall(proxy_trap_defineProperty, p.target, name, descr.toValue(p.val.runtime)); ok {
		booleanTrapResult := v.ToBoolean()
		if !booleanTrapResult {
			p.val.runtime.typeErrorResult(throw, "'defineProperty' on proxy: trap returned falsish")
			return false
		}
		targetDesc := propToValueProp(p.target.self.getOwnProp(name))
		extensibleTarget := p.target.self.isExtensible()
		settingConfigFalse := descr.Configurable == FLAG_FALSE
		if targetDesc == nil {
			if !extensibleTarget {
				panic(p.val.runtime.NewTypeError())
			}
			if settingConfigFalse {
				panic(p.val.runtime.NewTypeError())
			}
		} else {
			if !p.__isCompatibleDescriptor(extensibleTarget, descr, targetDesc) {
				panic(p.val.runtime.NewTypeError())
			}
			if settingConfigFalse && targetDesc.configurable {
				panic(p.val.runtime.NewTypeError())
			}
		}
		return booleanTrapResult
	}

	return p.target.self.defineOwnProperty(name, descr, throw)
}

func (p *proxyObject) proxyHas(name Value) (bool, bool) {
	if v, ok := p.proxyCall(proxy_trap_has, p.target, name); ok {
		booleanTrapResult := v.ToBoolean()
		if !booleanTrapResult {
			targetDesc := propToValueProp(p.target.self.getOwnProp(name))
			if targetDesc != nil {
				if !targetDesc.configurable {
					panic(p.val.runtime.NewTypeError("'has' on proxy: trap returned falsish for property '%s' which exists in the proxy target as non-configurable", name.String()))
				}
				if !p.target.self.isExtensible() {
					panic(p.val.runtime.NewTypeError("'has' on proxy: trap returned falsish for property '%s' but the proxy target is not extensible", name.String()))
				}
			}
		}
		return booleanTrapResult, true
	}

	return false, false
}

func (p *proxyObject) hasProperty(n Value) bool {
	if b, ok := p.proxyHas(n); ok {
		return b
	}

	return p.target.self.hasProperty(n)
}

func (p *proxyObject) hasPropertyStr(name string) (ret bool) {
	if b, ok := p.proxyHas(newStringValue(name)); ok {
		return b
	}

	return p.target.self.hasPropertyStr(name)
}

func (p *proxyObject) hasOwnProperty(n Value) bool {
	return p.getOwnProp(n) != nil
}

func (p *proxyObject) hasOwnPropertyStr(name string) bool {
	return p.getOwnPropStr(name) != nil
}

func (p *proxyObject) proxyGetOwnPropertyDescriptor(name Value) (Value, bool) {
	if v, ok := p.proxyCall(proxy_trap_getOwnPropertyDescriptor, p.target, name); ok {
		runtime := p.val.runtime
		target := p.target

		targetDesc := propToValueProp(target.self.getOwnProp(name))
		extensible := target.self.isExtensible()

		if v == nil || v == _undefined {
			if targetDesc != nil && !targetDesc.configurable {
				panic(runtime.NewTypeError("cannot report non-configurable property as non-existing"))
			}

			if !extensible && targetDesc != nil {
				panic(runtime.NewTypeError("cannot report existing property on non-extensible object as non-existing"))
			}
			return _undefined, true
		}

		if !extensible && targetDesc == nil {
			panic(runtime.NewTypeError("cannot report a new property on a non-extensible object"))
		}

		current := runtime.toPropertyDescriptor(v)
		if !p.__isCompatibleDescriptor(extensible, current, targetDesc) {
			panic(runtime.NewTypeError("cannot report incompatible property descriptor"))
		}

		if current.Configurable == FLAG_FALSE {
			if targetDesc == nil {
				panic(runtime.NewTypeError("cannot report non-configurable descriptor for non-existing property"))
			}

			if targetDesc.configurable {
				panic(runtime.NewTypeError("cannot report non-configurable descriptor for configurable property"))
			}

			if current.Writable == FLAG_FALSE && targetDesc.writable {
				panic(runtime.NewTypeError("cannot report non-configurable, writable property as non-configurable, non-writable"))
			}
		}

		return v, true
	}

	return nil, false
}

func (p *proxyObject) get(name Value, receiver Value) Value {
	if v, ok := p.proxyGet(name, receiver); ok {
		return v
	}
	return p.target.self.get(name, receiver)
}

func (p *proxyObject) getStr(name string, receiver Value) (ret Value) {
	if v, ok := p.proxyGet(newStringValue(name), receiver); ok {
		return v
	}
	return p.target.self.getStr(name, receiver)
}

func (p *proxyObject) proxyGet(name, receiver Value) (Value, bool) {
	if v, ok := p.proxyCall(proxy_trap_get, p.target, name, receiver); ok {
		if targetDesc, ok := p.target.self.getOwnProp(name).(*valueProperty); ok {
			if !targetDesc.accessor {
				if !targetDesc.writable && !targetDesc.configurable && !v.SameAs(targetDesc.value) {
					panic(p.val.runtime.NewTypeError("'get' on proxy: property '%s' is a read-only and non-configurable data property on the proxy target but the proxy did not return its actual value (expected '%s' but got '%s')", name.String(), nilSafe(targetDesc.value), ret))
				}
			} else {
				if !targetDesc.configurable && targetDesc.getterFunc == nil && v != _undefined {
					panic(p.val.runtime.NewTypeError("'get' on proxy: property '%s' is a non-configurable accessor property on the proxy target and does not have a getter function, but the trap did not return 'undefined' (got '%s')", name.String(), ret))
				}
			}
		}
		return v, true
	}

	return nil, false
}

func (p *proxyObject) getOwnPropStr(name string) Value {
	if v, ok := p.proxyGetOwnPropertyDescriptor(newStringValue(name)); ok {
		return p.val.runtime.toValueProp(v)
	}

	return p.target.self.getOwnPropStr(name)
}

func (p *proxyObject) getOwnProp(name Value) Value {
	if v, ok := p.proxyGetOwnPropertyDescriptor(name); ok {
		return p.val.runtime.toValueProp(v)
	}

	return p.target.self.getOwnProp(name)
}

func (p *proxyObject) proxySet(name, value, receiver Value) bool {
	if v, ok := p.proxyCall(proxy_trap_set, p.target, name, value, receiver); ok {
		if v.ToBoolean() {
			if prop, ok := p.target.self.getOwnProp(name).(*valueProperty); ok {
				if prop.accessor {
					if !prop.configurable {
						panic(p.val.runtime.NewTypeError())
					}
				} else if !prop.configurable && !prop.writable && !p.__sameValue(prop.value, value) {
					panic(p.val.runtime.NewTypeError())
				}
			}
		}

		return true
	}

	return false
}

func (p *proxyObject) setOwn(n Value, v Value, throw bool) {
	if !p.proxySet(n, v, p.val) {
		p.target.set(n, v, p.val, throw)
	}
}

func (p *proxyObject) setForeign(n Value, v, receiver Value, throw bool) bool {
	if !p.proxySet(n, v, receiver) {
		p.target.set(n, v, receiver, throw)
	}
	return true
}

func (p *proxyObject) setOwnStr(name string, v Value, throw bool) {
	if !p.proxySet(newStringValue(name), v, p.val) {
		p.target.setStr(name, v, p.val, throw)
	}
}

func (p *proxyObject) setForeignStr(name string, v, receiver Value, throw bool) bool {
	if !p.proxySet(newStringValue(name), v, receiver) {
		p.target.setStr(name, v, receiver, throw)
	}
	return true
}

func (p *proxyObject) setOwnSym(s *valueSymbol, v Value, throw bool) {
	p.setOwn(s, v, throw)
	if !p.proxySet(s, v, p.val) {
		p.target.set(s, v, p.val, throw)
	}
}

func (p *proxyObject) setForeignSym(s *valueSymbol, v, receiver Value, throw bool) bool {
	if !p.proxySet(s, v, receiver) {
		p.target.set(s, v, receiver, throw)
	}
	return true
}

func (p *proxyObject) proxyDelete(n Value) (bool, bool) {
	if v, ok := p.proxyCall(proxy_trap_deleteProperty, p.target, n); ok {
		if v.ToBoolean() {
			if targetDesc, ok := p.target.self.getOwnProp(n).(*valueProperty); ok {
				if !targetDesc.configurable {
					panic(p.val.runtime.NewTypeError("'deleteProperty' on proxy: property '%s' is a non-configurable property but the trap returned truish", n.String()))
				}
			}
			return true, true
		}
		return false, true
	}
	return false, false
}

func (p *proxyObject) deleteStr(name string, throw bool) bool {
	if ret, ok := p.proxyDelete(newStringValue(name)); ok {
		return ret
	}

	return p.target.self.deleteStr(name, throw)
}

func (p *proxyObject) delete(n Value, throw bool) bool {
	if ret, ok := p.proxyDelete(n); ok {
		return ret
	}

	return p.target.self.delete(n, throw)
}

func (p *proxyObject) proxyOwnKeys(symbols bool) ([]Value, bool) {
	if v, ok := p.proxyCall(proxy_trap_ownKeys, p.target); ok {
		keys := p.val.runtime.toObject(v)
		var strKeyList []Value
		strKeySet := make(map[string]struct{})
		var symKeyList []Value
		symKeySet := make(map[Value]struct{})
		l := toLength(keys.self.getStr("length", nil))
		for k := int64(0); k < l; k++ {
			item := keys.self.get(intToValue(k), nil)
			if s, ok := item.assertString(); ok {
				strKeyList = append(strKeyList, item)
				strKeySet[s.String()] = struct{}{}
			} else if sym, ok := item.(*valueSymbol); ok {
				symKeyList = append(symKeyList, item)
				symKeySet[sym] = struct{}{}
			} else {
				panic(p.val.runtime.NewTypeError("%s is not a valid property name", item))
			}
		}
		ext := p.target.self.isExtensible()
		for _, itemName := range p.target.self.ownKeys(true, nil) {
			itemNameStr := itemName.String()
			if _, exists := strKeySet[itemNameStr]; exists {
				delete(strKeySet, itemNameStr)
			} else {
				if !ext {
					panic(p.val.runtime.NewTypeError("'ownKeys' on proxy: trap result did not include '%s'", itemNameStr))
				}
				prop := p.target.self.getOwnPropStr(itemNameStr)
				if prop, ok := prop.(*valueProperty); ok && !prop.configurable {
					panic(p.val.runtime.NewTypeError("'ownKeys' on proxy: trap result did not include non-configurable '%s'", itemNameStr))
				}
			}
		}
		for _, sym := range p.target.self.ownSymbols() {
			if _, exists := symKeySet[sym]; exists {
				delete(symKeySet, sym)
			} else {
				if !ext {
					panic(p.val.runtime.NewTypeError("'ownKeys' on proxy: trap result did not include '%s'", sym.String()))
				}
				prop := p.target.self.getOwnProp(sym)
				if prop, ok := prop.(*valueProperty); ok && !prop.configurable {
					panic(p.val.runtime.NewTypeError("Missing symbol non-configurable property"))
				}
			}
		}
		if !ext && len(strKeyList) > 0 && len(strKeySet) > 0 {
			panic(p.val.runtime.NewTypeError("'ownKeys' on proxy: trap returned extra keys but proxy target is non-extensible"))
		}

		if !ext && len(symKeyList) > 0 && len(symKeySet) > 0 {
			panic(p.val.runtime.NewTypeError("'ownKeys' on proxy: trap returned extra keys but proxy target is non-extensible"))
		}

		if symbols {
			return symKeyList, true
		}

		return strKeyList, true
	}

	return nil, false
}

func (p *proxyObject) assertCallable() (call func(FunctionCall) Value, ok bool) {
	return func(call FunctionCall) Value {
		return p.apply(call.This, call.Arguments)
	}, true
}

func (p *proxyObject) apply(this Value, arguments []Value) (ret Value) {
	if this == _undefined {
		this = p.target
	}
	p.handleProxyRequest(proxy_trap_apply, func(proxyFunction func(FunctionCall) Value, this Value) {
		ret = proxyFunction(FunctionCall{
			This:      this,
			Arguments: []Value{p.target, this, p.val.runtime.newArrayValues(arguments)},
		})
	}, func(target *Object) {
		f := p.val.runtime.toCallable(p.target)
		ret = f(FunctionCall{
			This:      this,
			Arguments: arguments,
		})
	})
	return
}

func (p *proxyObject) construct(args []Value) (ret *Object) {
	p.handleProxyRequest(proxy_trap_construct, func(proxyFunction func(FunctionCall) Value, this Value) {
		ret = p.val.runtime.toObject(proxyFunction(FunctionCall{
			This:      this,
			Arguments: []Value{p.target, p.val.runtime.newArrayValues(args), p.val},
		}))
	}, func(target *Object) {
		ctor := getConstructor(target)
		if ctor == nil {
			p.val.runtime.typeErrorResult(true, "Not a constructor")
			panic("Unreachable")
		}
		ret = ctor(args)
	})
	return
}

func (p *proxyObject) __isSealed(target *Object, name Value) bool {
	prop := target.self.getOwnProp(name)
	if prop == nil {
		return false
	}
	if pp, ok := prop.(*valueProperty); ok {
		return !pp.configurable
	}
	return false
}

func (p *proxyObject) __isCompatibleDescriptor(extensible bool, current PropertyDescriptor, target *valueProperty) bool {
	if p.__isEmptyDescriptor(current) {
		return extensible
	}

	if target == nil {
		return true
	}

	if p.__isEquivalentDescriptor(current, target) {
		return true
	}

	if current.Configurable == FLAG_FALSE {
		if target.configurable {
			return false
		}

		if current.Enumerable != FLAG_NOT_SET && current.Enumerable.Bool() != target.enumerable {
			return false
		}

		if p.__isGenericDescriptor(current) {
			return true
		}

		if p.__isDataDescriptor(current) != !target.accessor {
			return current.Configurable != FLAG_FALSE
		}

		if p.__isDataDescriptor(current) && !target.accessor {
			if current.Configurable == FLAG_FALSE {
				if current.Writable == FLAG_FALSE && target.writable {
					return false
				}
				if current.Writable == FLAG_FALSE {
					if current.Value != nil && !p.__sameValue(current.Value, target.value) {
						return false
					}
				}
			}
			return true
		}
		if p.__isAccessorDescriptor(current) && target.accessor {
			if current.Configurable == FLAG_FALSE {
				if current.Setter != nil && current.Setter.SameAs(target.setterFunc) {
					return false
				}
				if current.Getter != nil && current.Getter.SameAs(target.getterFunc) {
					return false
				}
			}
		}
	}
	return true
}

func (p *proxyObject) __isAccessorDescriptor(desc PropertyDescriptor) bool {
	return desc.Setter != nil || desc.Getter != nil
}

func (p *proxyObject) __isDataDescriptor(desc PropertyDescriptor) bool {
	return desc.Value != nil || desc.Writable != FLAG_NOT_SET
}

func (p *proxyObject) __isGenericDescriptor(desc PropertyDescriptor) bool {
	return !p.__isAccessorDescriptor(desc) && !p.__isDataDescriptor(desc)
}

func (p *proxyObject) __isEmptyDescriptor(desc PropertyDescriptor) bool {
	return desc.Configurable == FLAG_NOT_SET &&
		desc.Enumerable == FLAG_NOT_SET &&
		desc.Writable == FLAG_NOT_SET &&
		desc.Getter == nil &&
		desc.Setter == nil &&
		desc.Value == nil
}

func (p *proxyObject) __isEquivalentDescriptor(desc PropertyDescriptor, targetDesc *valueProperty) bool {
	return desc.Configurable.Bool() == targetDesc.configurable &&
		desc.Enumerable.Bool() == targetDesc.enumerable &&
		desc.Writable.Bool() == targetDesc.writable &&
		p.__sameValue(desc.Value, targetDesc.value) &&
		p.__sameValueObj(desc.Setter, targetDesc.setterFunc) &&
		p.__sameValueObj(desc.Getter, targetDesc.getterFunc)
}

func (p *proxyObject) __sameValueObj(val1 Value, val2 *Object) bool {
	if val1 == nil && val2 == nil {
		return true
	}
	if val1 != nil {
		return val1.SameAs(val2)
	}
	return false
}

func (p *proxyObject) __sameValue(val1, val2 Value) bool {
	if val1 == nil && val2 == nil {
		return true
	}
	if val1 != nil {
		return val1.SameAs(val2)
	}
	return false
}

func (p *proxyObject) ownKeys(all bool, _ []Value) []Value { // we can assume accum is empty
	if vals, ok := p.proxyOwnKeys(false); ok {
		if !all {
			k := 0
			for i, val := range vals {
				prop := p.getOwnPropStr(val.String())
				if prop == nil {
					continue
				}
				if prop, ok := prop.(*valueProperty); ok && !prop.enumerable {
					continue
				}
				if k != i {
					vals[k] = vals[i]
				}
				k++
			}
			vals = vals[:k]
		}
		return vals
	}

	return p.target.self.ownKeys(all, nil)
}

func (p *proxyObject) ownSymbols() []Value {
	if vals, ok := p.proxyOwnKeys(true); ok {
		return vals
	}

	return p.target.self.ownSymbols()
}

func (p *proxyObject) enumerate() iterNextFunc {
	if v, ok := p.proxyCall(proxy_trap_enumerate, p.target); ok {
		return (&iteratorPropIter{iter: p.val.runtime.toObject(v)}).next
	}

	return p.target.self.enumerate()
}

func (r *Runtime) newProxy(args []Value) *Object {
	if len(args) >= 2 {
		if target, ok := args[0].(*Object); ok {
			if proxyHandler, ok := args[1].(*Object); ok {
				return r.newProxyObject(target, proxyHandler, false, true)
			}
		}
	}
	panic(r.NewTypeError("Cannot create proxy with a non-object as target or handler"))
}

func (r *Runtime) newProxyObject(target *Object, handler *Object, revocable, strict bool) *Object {
	v := &Object{runtime: r}
	p := &proxyObject{strict: strict}
	v.self = p
	p.val = v
	p.class = classProxy
	p.prototype = r.global.Proxy
	p.extensible = false
	p.init()
	p.target = target
	p.handler = handler
	p.revocable = revocable
	return v
}

func (r *Runtime) proxy_revocable(call FunctionCall) Value {
	if len(call.Arguments) >= 2 {
		if target, ok := call.Argument(0).(*Object); ok {
			if proxyHandler, ok := call.Argument(1).(*Object); ok {
				return r.newProxyObject(target, proxyHandler, true, true)
			}
		}
	}
	panic(r.NewTypeError("Cannot create proxy with a non-object as target or handler"))
}

func (r *Runtime) proxyproto_toString(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*proxyObject); ok {
		return asciiString(fmt.Sprintf("ES6 Proxy[%s]", d.target.String()))
	}
	panic(r.NewTypeError("Method Proxy.prototype.toString is called on incompatible receiver"))
}

func (r *Runtime) proxyproto_revoke(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*proxyObject); ok {
		if !d.revocable {
			panic(r.NewTypeError("Method Proxy.prototype.revoke is called on incompatible receiver"))
		}
		d.revoked = true
		return valueTrue
	}
	panic(r.NewTypeError("Method Proxy.prototype.revoke is called on incompatible receiver"))
}
