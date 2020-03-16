package goja

func (r *Runtime) builtin_newProxy(args []Value) *Object {
	return r.newProxy(args)
}

func (r *Runtime) proxy_revocable(call FunctionCall) Value {
	if len(call.Arguments) >= 2 {
		if target, ok := call.Argument(0).(*Object); ok {
			if proxyHandler, ok := call.Argument(1).(*Object); ok {
				proxy := r.newProxyObject(target, proxyHandler)
				revoke := r.newNativeFunc(func(FunctionCall) Value {
					proxy.revoke()
					return _undefined
				}, nil, "", nil, 0)
				ret := r.NewObject()
				ret.self._putProp("proxy", proxy.val, true, true, true)
				ret.self._putProp("revoke", revoke, true, true, true)
				return ret
			}
		}
	}
	panic(r.NewTypeError("Cannot create proxy with a non-object as target or handler"))
}

func (r *Runtime) createProxy(val *Object) objectImpl {
	o := r.newNativeFuncObj(val, r.constructorThrower("Proxy"), r.builtin_newProxy, "Proxy", nil, 2)

	o._putProp("revocable", r.newNativeFunc(r.proxy_revocable, nil, "revocable", nil, 2), true, false, true)
	return o
}

func (r *Runtime) initProxy() {
	r.global.Proxy = r.newLazyObject(r.createProxy)
	r.addToGlobal("Proxy", r.global.Proxy)
}
