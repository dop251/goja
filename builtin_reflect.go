package goja

func (r *Runtime) builtin_reflect_apply(call FunctionCall) Value {
	return r.toCallable(call.Argument(0))(FunctionCall{
		This:      call.Argument(1),
		Arguments: r.createListFromArrayLike(call.Argument(2))})
}

func (r *Runtime) toConstructor(v Value) func(args []Value, newTarget Value) *Object {
	if ctor := r.toObject(v).self.assertConstructor(); ctor != nil {
		return ctor
	}
	panic(r.NewTypeError("Value is not a constructor"))
}

func (r *Runtime) builtin_reflect_construct(call FunctionCall) Value {
	target := call.Argument(0)
	ctor := r.toConstructor(target)
	var newTarget Value
	if len(call.Arguments) > 2 {
		newTarget = call.Argument(2)
		r.toConstructor(newTarget)
	} else {
		newTarget = target
	}
	return ctor(r.createListFromArrayLike(call.Argument(1)), newTarget)
}

func (r *Runtime) createReflect(val *Object) objectImpl {
	o := newBaseObjectObj(val, r.global.ObjectPrototype, classObject)

	o._putProp("apply", r.newNativeFunc(r.builtin_reflect_apply, nil, "apply", nil, 3), true, false, true)
	o._putProp("construct", r.newNativeFunc(r.builtin_reflect_construct, nil, "construct", nil, 2), true, false, true)

	return o
}

func (r *Runtime) initReflect() {
	r.global.Proxy = r.newLazyObject(r.createReflect)
	r.addToGlobal("Reflect", r.global.Reflect)
}
