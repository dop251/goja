package goja

import (
	"fmt"
	"errors"
)

type Proxy struct {
	proxy *Object
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
	//r.proxyproto_nativehandler_setPrototypeOf(nativeHandler.SetPrototypeOf, handler) <- not yet implemented
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
	// SetPrototypeOf func(target *Object, prototype *Object) (success bool) <- not yet implemented

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

type proxyTrap int8

const (
	proxy_trap_getPrototypeOf           proxyTrap = iota
	proxy_trap_setPrototypeOf
	proxy_trap_isExtensible
	proxy_trap_preventExtensions
	proxy_trap_getOwnPropertyDescriptor
	proxy_trap_defineProperty
	proxy_trap_has
	proxy_trap_get
	proxy_trap_set
	proxy_trap_deleteProperty
	proxy_trap_ownKeys
	proxy_trap_apply
	proxy_trap_construct
)

func (p proxyTrap) String() (name string) {
	switch p {
	case proxy_trap_getPrototypeOf:
		name = "getPrototypeOf"
	case proxy_trap_setPrototypeOf:
		name = "setPrototypeOf"
	case proxy_trap_isExtensible:
		name = "isExtensible"
	case proxy_trap_preventExtensions:
		name = "preventExtensions"
	case proxy_trap_getOwnPropertyDescriptor:
		name = "getOwnPropertyDescriptor"
	case proxy_trap_defineProperty:
		name = "defineProperty"
	case proxy_trap_has:
		name = "has"
	case proxy_trap_get:
		name = "get"
	case proxy_trap_set:
		name = "set"
	case proxy_trap_deleteProperty:
		name = "deleteProperty"
	case proxy_trap_ownKeys:
		name = "ownKeys"
	case proxy_trap_apply:
		name = "apply"
	case proxy_trap_construct:
		name = "construct"
	}
	return
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

func (p *proxyObject) isExtensible() (ret bool) {
	ex := p.handleProxyRequest(proxy_trap_isExtensible, func(proxyFunction func(FunctionCall) Value, this Value) {
		v := proxyFunction(FunctionCall{
			This:      this,
			Arguments: []Value{p.target},
		})
		if o, ok := v.(valueBool); ok {
			ret = o.ToBoolean()
		} else {
			panic(errors.New("illegal return type from proxy trap"))
		}
	}, func(target *Object) {
		ret = target.self.isExtensible()
	})
	if ex != nil {
		panic(ex)
	}
	return
}

func (p *proxyObject) preventExtensions() {
	ex := p.handleProxyRequest(proxy_trap_preventExtensions, func(proxyFunction func(FunctionCall) Value, this Value) {
		proxyFunction(FunctionCall{
			This:      this,
			Arguments: []Value{p.target},
		})
	}, func(target *Object) {
		target.self.preventExtensions()
	})
	if ex != nil {
		panic(ex)
	}
}

func (p *proxyObject) defineOwnProperty(name Value, descr PropertyDescriptor, throw bool) (ret bool) {
	ex := p.handleProxyRequest(proxy_trap_defineProperty, func(proxyFunction func(FunctionCall) Value, this Value) {
		proxyFunction(FunctionCall{
			This:      this,
			Arguments: []Value{p.target, name, descr.toValue(p.val.runtime)},
		})
	}, func(target *Object) {
		ret = p.target.self.defineOwnProperty(name, descr, throw)
	})
	if ex != nil {
		if throw {
			panic(ex)
		}
		ret = false
	}
	return
}

func (p *proxyObject) __getOwnPropertyDescriptor(name string, proxyFunction func(FunctionCall) Value, this Value, target *Object, runtime *Runtime) Value {
	desc := proxyFunction(FunctionCall{
		This:      this,
		Arguments: []Value{p.target, newStringValue(name)},
	})

	if !p.strict {
		return desc
	}

	targetDesc := target.self.getOwnPropertyDescriptor(name)
	extensible := target.self.isExtensible()

	if desc == nil {
		if p.__isSealed(p.target, name) {
			return runtime.NewTypeError("cannot report non-configurable property as non-existing")
		}

		if !extensible && targetDesc != nil {
			return runtime.NewTypeError("cannot report existing property on non-extensible object as non-existing")
		}
		return _undefined
	}

	if !extensible && targetDesc == nil {
		return runtime.NewTypeError("cannot report a new property on a non-extensible object")
	}

	if !p.__isCompatibleDescriptor(extensible, desc, targetDesc) {
		return runtime.NewTypeError("cannot report incompatible property descriptor")
	}

	current := runtime.toPropertyDescriptor(desc)
	if current.Configurable == FLAG_FALSE {
		if targetDesc == nil || targetDesc == _undefined {
			return runtime.NewTypeError("cannot report non-configurable descriptor for non-existing property")
		}

		target := runtime.toPropertyDescriptor(targetDesc)
		if target.Configurable == FLAG_TRUE {
			return runtime.NewTypeError("cannot report non-configurable descriptor for configurable property")
		}

		if current.Writable == FLAG_FALSE && target.Writable == FLAG_TRUE {
			return runtime.NewTypeError("cannot report non-configurable, writable property as non-configurable, non-writable")
		}
	}
	return desc
}

func (p *proxyObject) getOwnPropertyDescriptor(name string) (ret Value) {
	ex := p.handleProxyRequest(proxy_trap_getOwnPropertyDescriptor, func(proxyFunction func(FunctionCall) Value, this Value) {
		ret = p.__getOwnPropertyDescriptor(name, proxyFunction, this, p.target, p.val.runtime)
	}, func(target *Object) {
		ret = target.self.getOwnPropertyDescriptor(name)
	})
	if ex != nil {
		ret = ex.Value()
	}
	return
}

func (p *proxyObject) hasProperty(n Value) (ret bool) {
	ex := p.handleProxyRequest(proxy_trap_has, func(proxyFunction func(FunctionCall) Value, this Value) {
		v := proxyFunction(FunctionCall{
			This:      this,
			Arguments: []Value{p.target, n},
		})
		if o, ok := v.(valueBool); ok {
			ret = o.ToBoolean()
		} else {
			panic(errors.New("illegal return type from proxy trap"))
		}
	}, func(target *Object) {
		ret = target.self.hasProperty(n)
	})
	if ex != nil {
		panic(ex)
	}
	return
}

func (p *proxyObject) hasPropertyStr(name string) (ret bool) {
	ex := p.handleProxyRequest(proxy_trap_has, func(proxyFunction func(FunctionCall) Value, this Value) {
		v := proxyFunction(FunctionCall{
			This:      this,
			Arguments: []Value{p.target, newStringValue(name)},
		})
		if o, ok := v.(valueBool); ok {
			ret = o.ToBoolean()
		} else {
			panic(errors.New("illegal return type from proxy trap"))
		}
	}, func(target *Object) {
		ret = target.self.hasPropertyStr(name)
	})
	if ex != nil {
		panic(ex)
	}
	return
}

func (p *proxyObject) getProp(n Value) (ret Value) {
	var propFound bool
	ex := p.handleProxyRequest(proxy_trap_has, func(proxyFunction func(FunctionCall) Value, this Value) {
		v := proxyFunction(FunctionCall{
			This:      this,
			Arguments: []Value{p.target, n},
		})
		if o, ok := v.(valueBool); ok {
			propFound = o.ToBoolean()
		} else {
			panic(errors.New("illegal return type from proxy trap"))
		}
	}, func(target *Object) {
		propFound = target.self.hasPropertyStr(n.String())
	})
	if ex != nil {
		panic(ex)
	}

	ret = _undefined
	if propFound {
		ex := p.handleProxyRequest(proxy_trap_get, func(proxyFunction func(FunctionCall) Value, this Value) {
			ret = proxyFunction(FunctionCall{
				This:      this,
				Arguments: []Value{p.target, n, p.val},
			})
		}, func(target *Object) {
			ret = target.self.getProp(n)
		})
		if ex != nil {
			ret = ex.Value()
		}
	}
	return
}

func (p *proxyObject) getStr(name string) (ret Value) {
	var propFound bool
	ex := p.handleProxyRequest(proxy_trap_has, func(proxyFunction func(FunctionCall) Value, this Value) {
		v := proxyFunction(FunctionCall{
			This:      this,
			Arguments: []Value{p.target, newStringValue(name)},
		})
		if o, ok := v.(valueBool); ok {
			propFound = o.ToBoolean()
		} else {
			panic(errors.New("illegal return type from proxy trap"))
		}
	}, func(target *Object) {
		propFound = target.self.hasPropertyStr(name)
	})
	if ex != nil {
		panic(ex)
	}

	ret = _undefined
	if propFound {
		ex := p.handleProxyRequest(proxy_trap_get, func(proxyFunction func(FunctionCall) Value, this Value) {
			ret = proxyFunction(FunctionCall{
				This:      this,
				Arguments: []Value{p.target, newStringValue(name), p.val},
			})
		}, func(target *Object) {
			ret = target.self.getStr(name)
		})
		if ex != nil {
			ret = ex.Value()
		}
	}
	return
}

func (p *proxyObject) getPropStr(name string) (ret Value) {
	var propFound bool
	ex := p.handleProxyRequest(proxy_trap_has, func(proxyFunction func(FunctionCall) Value, this Value) {
		v := proxyFunction(FunctionCall{
			This:      this,
			Arguments: []Value{p.target, newStringValue(name)},
		})
		if o, ok := v.(valueBool); ok {
			propFound = o.ToBoolean()
		} else {
			panic(errors.New("illegal return type from proxy trap"))
		}
	}, func(target *Object) {
		propFound = target.self.hasPropertyStr(name)
	})
	if ex != nil {
		panic(ex)
	}

	ret = _undefined
	if propFound {
		ex := p.handleProxyRequest(proxy_trap_get, func(proxyFunction func(FunctionCall) Value, this Value) {
			ret = proxyFunction(FunctionCall{
				This:      this,
				Arguments: []Value{p.target, newStringValue(name), p.val},
			})
		}, func(target *Object) {
			ret = target.self.getPropStr(name)
		})
		if ex != nil {
			ret = ex.Value()
		}
	}
	return
}

func (p *proxyObject) getOwnProp(name string) (ret Value) {
	var propFound bool
	ex := p.handleProxyRequest(proxy_trap_has, func(proxyFunction func(FunctionCall) Value, this Value) {
		v := proxyFunction(FunctionCall{
			This:      this,
			Arguments: []Value{p.target, newStringValue(name)},
		})
		if o, ok := v.(valueBool); ok {
			propFound = o.ToBoolean()
		} else {
			panic(errors.New("illegal return type from proxy trap"))
		}
	}, func(target *Object) {
		propFound = target.self.hasPropertyStr(name)
	})
	if ex != nil {
		panic(ex)
	}

	ret = _undefined
	if propFound {
		ex := p.handleProxyRequest(proxy_trap_get, func(proxyFunction func(FunctionCall) Value, this Value) {
			ret = proxyFunction(FunctionCall{
				This:      this,
				Arguments: []Value{p.target, newStringValue(name), p.val},
			})
		}, func(target *Object) {
			ret = target.self.getOwnProp(name)
		})
		if ex != nil {
			ret = ex.Value()
		}
	}
	return
}

func (p *proxyObject) proto() (ret *Object) {
	ex := p.handleProxyRequest(proxy_trap_getPrototypeOf, func(proxyFunction func(FunctionCall) Value, this Value) {
		ret = proxyFunction(FunctionCall{
			This:      this,
			Arguments: []Value{p.target},
		}).(*Object)
	}, func(target *Object) {
		ret = target.self.proto()
	})
	if ex != nil {
		panic(ex)
	}
	return
}

func (p *proxyObject) put(n Value, val Value, throw bool) {
	ex := p.handleProxyRequest(proxy_trap_set, func(proxyFunction func(FunctionCall) Value, this Value) {
		proxyFunction(FunctionCall{
			This:      this,
			Arguments: []Value{p.target, n, val, p.val},
		})
	}, func(target *Object) {
		target.self.putStr(n.String(), val, throw)
	})
	if ex != nil {
		panic(ex)
	}
}

func (p *proxyObject) putStr(name string, val Value, throw bool) {
	ex := p.handleProxyRequest(proxy_trap_set, func(proxyFunction func(FunctionCall) Value, this Value) {
		proxyFunction(FunctionCall{
			This:      this,
			Arguments: []Value{p.target, newStringValue(name), val, p.val},
		})
	}, func(target *Object) {
		target.self.putStr(name, val, throw)
	})
	if ex != nil {
		panic(ex)
	}
}

func (p *proxyObject) deleteStr(name string, throw bool) (ret bool) {
	ex := p.handleProxyRequest(proxy_trap_deleteProperty, func(proxyFunction func(FunctionCall) Value, this Value) {
		v := proxyFunction(FunctionCall{
			This:      this,
			Arguments: []Value{p.target, newStringValue(name)},
		})
		if o, ok := v.(valueBool); ok {
			ret = o.ToBoolean()
		} else {
			panic(errors.New("illegal return type from proxy trap"))
		}
	}, func(target *Object) {
		ret = target.self.deleteStr(name, throw)
	})
	if ex != nil {
		panic(ex)
	}
	return
}

func (p *proxyObject) delete(n Value, throw bool) (ret bool) {
	ex := p.handleProxyRequest(proxy_trap_deleteProperty, func(proxyFunction func(FunctionCall) Value, this Value) {
		v := proxyFunction(FunctionCall{
			This:      this,
			Arguments: []Value{p.target, n},
		})
		if o, ok := v.(valueBool); ok {
			ret = o.ToBoolean()
		} else {
			panic(errors.New("illegal return type from proxy trap"))
		}
	}, func(target *Object) {
		ret = target.self.delete(n, throw)
	})
	if ex != nil {
		panic(ex)
	}
	return
}

func (p *proxyObject) ownKeys(all, recursive bool) (ret Value) {
	ex := p.handleProxyRequest(proxy_trap_ownKeys, func(proxyFunction func(FunctionCall) Value, this Value) {
		ret = proxyFunction(FunctionCall{
			This:      this,
			Arguments: []Value{p.target},
		})
	}, func(target *Object) {
		var keys []Value
		for item, f := target.self.enumerate(all, recursive)(); f != nil; item, f = f() {
			keys = append(keys, newStringValue(item.name))
		}
		ret = p.val.runtime.newArrayValues(keys)
	})
	if ex != nil {
		panic(ex)
	}
	return
}

func (p *proxyObject) apply(this Value, arguments []Value) (ret Value) {
	if this == _undefined {
		this = p.target
	}
	ex := p.handleProxyRequest(proxy_trap_apply, func(proxyFunction func(FunctionCall) Value, this Value) {
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
	if ex != nil {
		panic(ex)
	}
	return
}

func (p *proxyObject) construct(args []Value) (ret Value) {
	ex := p.handleProxyRequest(proxy_trap_construct, func(proxyFunction func(FunctionCall) Value, this Value) {
		ret = proxyFunction(FunctionCall{
			This:      this,
			Arguments: []Value{p.target, p.val.runtime.newArrayValues(args), p.val},
		})
	}, func(target *Object) {
		nativeConstruct := func(f *nativeFuncObject, args []Value) Value {
			if f.construct != nil {
				return f.construct(args)
			}
			p.val.runtime.typeErrorResult(true, "Not a constructor")
			panic("Unreachable")
		}

	repeat:
		switch f := target.self.(type) {
		case *funcObject:
			ret = f.construct(args)
		case *nativeFuncObject:
			ret = nativeConstruct(f, args)
		case *boundFuncObject:
			ret = nativeConstruct(&f.nativeFuncObject, args)
		case *lazyObject:
			target.self = f.create(target)
			goto repeat
		default:
			p.val.runtime.typeErrorResult(true, "Not a constructor")
		}
	})
	if ex != nil {
		panic(ex)
	}
	return
}

func (p *proxyObject) handleProxyRequest(trap proxyTrap, proxyCallback func(proxyFunction func(FunctionCall) Value, this Value), targetCallback func(target *Object)) *Exception {
	runtime := p.val.runtime
	if p.revocable && p.revoked {
		runtime.typeErrorResult(true, "Proxy already revoked")
		panic("Unreachable")
	}

	return runtime.vm.try(func() {
		prop := p.handler.self.getOwnProp(trap.String())
		if prop == nil {
			// Redirect to target object
			targetCallback(p.target)
		} else {
			handler := prop.(*Object)
			f := runtime.toCallable(handler)
			proxyCallback(f, handler)
		}
	})
}

func (p *proxyObject) __isSealed(target *Object, name string) bool {
	prop := target.self.getOwnProp(name)
	if prop == nil {
		return false
	}
	if pp, ok := prop.(*valueProperty); ok {
		return !pp.configurable
	}
	return false
}

func (p *proxyObject) __isCompatibleDescriptor(extensible bool, desc, targetDesc Value) bool {
	if desc == nil {
		return extensible
	}

	current := p.val.runtime.toPropertyDescriptor(desc)
	target := p.val.runtime.toPropertyDescriptor(targetDesc)

	if p.__isEmptyDescriptor(target) {
		return true
	}

	if p.__isEquivalentDescriptor(current, target) {
		return true
	}

	if current.Configurable == FLAG_FALSE {
		if target.Configurable == FLAG_TRUE {
			return false
		}

		if current.Enumerable != FLAG_NOT_SET && current.Enumerable != target.Enumerable {
			return false
		}

		if p.__isGenericDescriptor(current) {
			return true
		}

		if p.__isDataDescriptor(current) != p.__isDataDescriptor(target) {
			return current.Configurable != FLAG_FALSE
		}

		if p.__isDataDescriptor(current) && p.__isDataDescriptor(target) {
			if current.Configurable == FLAG_FALSE {
				if current.Writable == FLAG_FALSE && target.Writable == FLAG_TRUE {
					return false
				}
				if current.Writable == FLAG_FALSE {
					if current.Value != nil && !p.__sameValue(current.Value, target.Value) {
						return false
					}
				}
			}
			return true
		}
		if p.__isAccessorDescriptor(current) && p.__isAccessorDescriptor(target) {
			if current.Configurable == FLAG_FALSE {
				if current.Setter != nil && p.__sameValue(current.Setter, target.Setter) {
					return false
				}
				if current.Getter != nil && p.__sameValue(current.Getter, target.Getter) {
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

func (p *proxyObject) __isEquivalentDescriptor(desc, targetDesc PropertyDescriptor) bool {
	return desc.Configurable == targetDesc.Configurable &&
		desc.Enumerable == targetDesc.Enumerable &&
		desc.Writable == targetDesc.Writable &&
		p.__sameValue(desc.Value, targetDesc.Value) &&
		p.__sameValue(desc.Setter, targetDesc.Setter) &&
		p.__sameValue(desc.Getter, targetDesc.Getter)
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

func (r *Runtime) isProxy(value Value) bool {
	if o, ok := value.(*Object); ok {
		_, ok := o.self.(*proxyObject)
		return ok
	}
	return false
}

func (r *Runtime) getProxy(value Value) *proxyObject {
	if o, ok := value.(*Object); ok {
		return o.self.(*proxyObject)
	}
	r.typeErrorResult(true, "Value is not a proxy: %s", value.ToString())
	panic("Unreachable")
}

func (r *Runtime) newProxy(args []Value) *Object {
	if len(args) >= 2 {
		if target, ok := args[0].(*Object); ok {
			if proxyHandler, ok := args[1].(*Object); ok {
				return r.newProxyObject(target, proxyHandler, false, true)
			}
		}
	}
	r.typeErrorResult(true, "Cannot create proxy with a non-object as target or handler")
	panic("Unreachable")
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
	r.typeErrorResult(true, "Cannot create proxy with a non-object as target or handler")
	panic("Unreachable")
}

func (r *Runtime) proxyproto_toString(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*proxyObject); ok {
		return asciiString(fmt.Sprintf("ES6 Proxy[%s]", d.target.ToString()))
	}
	r.typeErrorResult(true, "Method Proxy.prototype.toString is called on incompatible receiver")
	panic("Unreachable")
}

func (r *Runtime) proxyproto_revoke(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*proxyObject); ok {
		if !d.revocable {
			r.typeErrorResult(true, "Method Proxy.prototype.revoke is called on incompatible receiver")
			panic("Unreachable")
		}
		d.revoked = true
		return valueTrue
	}
	r.typeErrorResult(true, "Method Proxy.prototype.toString is called on incompatible receiver")
	panic("Unreachable")
}
