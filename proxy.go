package goja

type Proxy struct {
	proxy *proxyObject
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

func (r *Runtime) NewProxy(target *Object, nativeHandler *ProxyTrapConfig) *Proxy {
	handler := r.newNativeProxyHandler(nativeHandler)
	proxy := r.newProxyObject(target, handler)
	return &Proxy{proxy: proxy}
}

func (p *Proxy) Revoke() {
	p.proxy.revoke()
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
		handler.self._putProp("getPrototypeOf", r.newNativeFunc(func(call FunctionCall) Value {
			if len(call.Arguments) >= 1 {
				if t, ok := call.Argument(0).(*Object); ok {
					return native(t)
				}
			}
			panic(r.NewTypeError("getPrototypeOf needs to be called with target as Object"))
		}, nil, "[native getPrototypeOf]", nil, 1), true, true, true)
	}
}

func (r *Runtime) proxyproto_nativehandler_setPrototypeOf(native func(*Object, *Object) bool, handler *Object) {
	if native != nil {
		handler.self._putProp("setPrototypeOf", r.newNativeFunc(func(call FunctionCall) Value {
			if len(call.Arguments) >= 2 {
				if t, ok := call.Argument(0).(*Object); ok {
					if p, ok := call.Argument(1).(*Object); ok {
						s := native(t, p)
						return r.ToValue(s)
					}
				}
			}
			panic(r.NewTypeError("setPrototypeOf needs to be called with target and prototype as Object"))
		}, nil, "[native setPrototypeOf]", nil, 2), true, true, true)
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
	target  *Object
	handler *Object
	call    func(FunctionCall) Value
	ctor    func(args []Value) *Object
}

func (p *proxyObject) proxyCall(trap proxyTrap, args ...Value) (Value, bool) {
	r := p.val.runtime
	if p.handler == nil {
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
	target := p.target
	if v, ok := p.proxyCall(proxy_trap_getOwnPropertyDescriptor, target, name); ok {
		runtime := p.val.runtime

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
	target := p.target
	if v, ok := p.proxyCall(proxy_trap_get, target, name, receiver); ok {
		if targetDesc, ok := target.self.getOwnProp(name).(*valueProperty); ok {
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

func proxyProp(v Value) Value {
	if _, ok := v.(*valueSymbol); ok {
		return v
	}
	return v.toString()
}

func (p *proxyObject) getOwnProp(name Value) Value {
	if v, ok := p.proxyGetOwnPropertyDescriptor(proxyProp(name)); ok {
		return p.val.runtime.toValueProp(v)
	}

	return p.target.self.getOwnProp(name)
}

func (p *proxyObject) proxySet(name, value, receiver Value) bool {
	target := p.target
	if v, ok := p.proxyCall(proxy_trap_set, target, name, value, receiver); ok {
		if v.ToBoolean() {
			if prop, ok := target.self.getOwnProp(name).(*valueProperty); ok {
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
	if !p.proxySet(n, proxyProp(v), p.val) {
		p.target.set(n, v, p.val, throw)
	}
}

func (p *proxyObject) setForeign(n Value, v, receiver Value, throw bool) bool {
	if !p.proxySet(n, proxyProp(v), receiver) {
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
	target := p.target
	if v, ok := p.proxyCall(proxy_trap_deleteProperty, target, n); ok {
		if v.ToBoolean() {
			if targetDesc, ok := target.self.getOwnProp(n).(*valueProperty); ok {
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
	if ret, ok := p.proxyDelete(proxyProp(n)); ok {
		return ret
	}

	return p.target.self.delete(n, throw)
}

func (p *proxyObject) ownPropertyKeys(all bool, _ []Value) []Value {
	if v, ok := p.proxyOwnKeys(); ok {
		return v
	}
	return p.target.self.ownPropertyKeys(all, nil)
}

func (p *proxyObject) proxyOwnKeys() ([]Value, bool) {
	target := p.target
	if v, ok := p.proxyCall(proxy_trap_ownKeys, p.target); ok {
		keys := p.val.runtime.toObject(v)
		var keyList []Value
		keySet := make(map[Value]struct{})
		l := toLength(keys.self.getStr("length", nil))
		for k := int64(0); k < l; k++ {
			item := keys.self.get(intToValue(k), nil)
			if _, ok := item.assertString(); !ok {
				if _, ok := item.(*valueSymbol); !ok {
					panic(p.val.runtime.NewTypeError("%s is not a valid property name", item.String()))
				}
			}
			keyList = append(keyList, item)
			keySet[item] = struct{}{}
		}
		ext := target.self.isExtensible()
		for _, itemName := range target.self.ownPropertyKeys(true, nil) {
			if _, exists := keySet[itemName]; exists {
				delete(keySet, itemName)
			} else {
				if !ext {
					panic(p.val.runtime.NewTypeError("'ownKeys' on proxy: trap result did not include '%s'", itemName.String()))
				}
				prop := target.self.getOwnProp(itemName)
				if prop, ok := prop.(*valueProperty); ok && !prop.configurable {
					panic(p.val.runtime.NewTypeError("'ownKeys' on proxy: trap result did not include non-configurable '%s'", itemName.String()))
				}
			}
		}
		if !ext && len(keyList) > 0 && len(keySet) > 0 {
			panic(p.val.runtime.NewTypeError("'ownKeys' on proxy: trap returned extra keys but proxy target is non-extensible"))
		}

		return keyList, true
	}

	return nil, false
}

func (p *proxyObject) assertCallable() (call func(FunctionCall) Value, ok bool) {
	if p.call != nil {
		return func(call FunctionCall) Value {
			return p.apply(call)
		}, true
	}
	return nil, false
}

func (p *proxyObject) apply(call FunctionCall) Value {
	if p.call == nil {
		p.val.runtime.NewTypeError("proxy target is not a function")
	}
	if v, ok := p.proxyCall(proxy_trap_apply, p.target, nilSafe(call.This), p.val.runtime.newArrayValues(call.Arguments)); ok {
		return v
	}
	return p.call(call)
}

func (p *proxyObject) construct(args []Value) *Object {
	if p.ctor == nil {
		panic(p.val.runtime.NewTypeError("proxy target is not a constructor"))
	}
	if v, ok := p.proxyCall(proxy_trap_construct, p.target, p.val.runtime.newArrayValues(args), p.val); ok {
		return p.val.runtime.toObject(v)
	}
	return p.ctor(args)
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

func (p *proxyObject) filterKeys(vals []Value, all, symbols bool) []Value {
	if !all {
		k := 0
		for i, val := range vals {
			if _, ok := val.(*valueSymbol); ok != symbols {
				continue
			}
			prop := p.getOwnProp(val)
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
	} else {
		k := 0
		for i, val := range vals {
			if _, ok := val.(*valueSymbol); ok {
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

func (p *proxyObject) ownKeys(all bool, _ []Value) []Value { // we can assume accum is empty
	if vals, ok := p.proxyOwnKeys(); ok {
		return p.filterKeys(vals, all, false)
	}

	return p.target.self.ownKeys(all, nil)
}

func (p *proxyObject) ownSymbols() []Value {
	if vals, ok := p.proxyOwnKeys(); ok {
		return p.filterKeys(vals, true, true)
	}

	return p.target.self.ownSymbols()
}

func (p *proxyObject) enumerate() iterNextFunc {
	if v, ok := p.proxyCall(proxy_trap_enumerate, p.target); ok {
		return (&iteratorPropIter{iter: p.val.runtime.toObject(v)}).next
	}

	return p.target.self.enumerate()
}

func (p *proxyObject) className() string {
	if p.target == nil {
		panic(p.val.runtime.NewTypeError("proxy has been revoked"))
	}
	if p.call != nil || p.ctor != nil {
		return classFunction
	}
	return classObject
}

func (p *proxyObject) revoke() {
	p.handler = nil
	p.target = nil
}

func (r *Runtime) newProxy(args []Value) *Object {
	if len(args) >= 2 {
		if target, ok := args[0].(*Object); ok {
			if proxyHandler, ok := args[1].(*Object); ok {
				return r.newProxyObject(target, proxyHandler).val
			}
		}
	}
	panic(r.NewTypeError("Cannot create proxy with a non-object as target or handler"))
}

func (r *Runtime) newProxyObject(target *Object, handler *Object) *proxyObject {
	v := &Object{runtime: r}
	p := &proxyObject{}
	v.self = p
	p.val = v
	p.class = classObject
	p.prototype = r.global.Proxy
	p.extensible = false
	p.init()
	p.target = target
	p.handler = handler
	if call, ok := target.self.assertCallable(); ok {
		p.call = call
	}
	if ctor := getConstructor(target); ctor != nil {
		p.ctor = ctor
	}
	return p
}
