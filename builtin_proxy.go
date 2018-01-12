package goja

func (r *Runtime) builtin_newProxy(args []Value, proto *Object) *Object {
	return r.newProxy(args)
}

func (r *Runtime) createProxy(val *Object) objectImpl {
	o := r.newNativeFuncConstructObj(val, r.builtin_newProxy, "Proxy", r.global.ProxyPrototype, 1)

	o._putProp("revocable", r.newNativeFunc(r.proxy_revocable, nil, "revocable", nil, 1), false, false, false)

	return o
}

func (r *Runtime) initProxy() {
	r.global.Proxy = r.newLazyObject(r.createProxy)
	r.addToGlobal("Proxy", r.global.Proxy)
}

func (r *Runtime) createProxyProto(val *Object) objectImpl {
	o := &baseObject{
		class:      classProxy,
		val:        val,
		extensible: false,
		prototype:  r.global.ObjectPrototype,
	}
	o.init()

	o._putProp("constructor", r.global.Proxy, false, false, false)
	o._putProp("toString", r.newNativeFunc(r.proxyproto_toString, nil, "toString", nil, 0), true, false, true)
	o._putProp("revoke", r.newNativeFunc(r.proxyproto_revoke, nil, "revoke", nil, 0), true, false, true)

	return o
}
