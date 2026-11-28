package goja

import "github.com/dop251/goja/unistring"

type disposableResource struct {
	method       func(FunctionCall) Value
	call         FunctionCall
	syncFallback bool
}

type disposableStackObject struct {
	baseObject
	capability *disposeCapability
	async      bool
}

type disposeCapability struct {
	resources []disposableResource
}

func (r *Runtime) newDisposableStack(proto *Object, async bool, capability *disposeCapability) *Object {
	obj := &Object{runtime: r}
	stack := &disposableStackObject{
		baseObject: baseObject{
			class:      classObject,
			val:        obj,
			extensible: true,
			prototype:  proto,
		},
		async:      async,
		capability: capability,
	}
	obj.self = stack
	stack.init()
	return obj
}

func (r *Runtime) thisDisposableStack(receiver Value, async bool) *disposableStackObject {
	if obj, ok := receiver.(*Object); ok {
		if stack, ok := obj.self.(*disposableStackObject); ok && stack.async == async {
			return stack
		}
	}
	panic(r.NewTypeError("Incompatible disposable stack receiver"))
}

func (s *disposableStackObject) beginDisposal() []disposableResource {
	capability := s.capability
	s.capability = nil
	resources := capability.resources
	capability.resources = nil
	return resources
}

// getOptionalMethod validates a property read for GetMethod without coercing invalid values.
func getOptionalMethod(value Value) func(FunctionCall) Value {
	if value == nil || IsUndefined(value) || IsNull(value) {
		return nil
	}
	if method, ok := assertCallable(value); ok {
		return method
	}
	panic(newTypeError("Property is not callable"))
}

func (s *disposableStackObject) use(call FunctionCall) Value {
	// A method getter can move the stack while this registration is in progress.
	capability := s.capability
	value := call.Argument(0)
	if IsUndefined(value) || IsNull(value) {
		if s.async {
			capability.resources = append(capability.resources, disposableResource{})
		}
		return value
	}
	r := s.val.runtime
	obj, ok := value.(*Object)
	if !ok {
		panic(r.NewTypeError("Disposable resource must be an object"))
	}
	resource := disposableResource{call: FunctionCall{This: obj}}
	if s.async {
		resource.method = getOptionalMethod(obj.self.getSym(SymAsyncDispose, nil))
	}
	if resource.method == nil {
		resource.method = getOptionalMethod(obj.self.getSym(SymDispose, nil))
		if resource.method == nil {
			panic(r.NewTypeError("Missing disposal method"))
		}
		resource.syncFallback = s.async
	}
	capability.resources = append(capability.resources, resource)
	return value
}

func (s *disposableStackObject) adopt(call FunctionCall) Value {
	value := call.Argument(0)
	method, ok := assertCallable(call.Argument(1))
	if !ok {
		panic(s.val.runtime.NewTypeError("Disposal callback must be callable"))
	}
	s.capability.resources = append(s.capability.resources, disposableResource{
		method: method,
		call:   FunctionCall{This: _undefined, Arguments: []Value{value}},
	})
	return value
}

func (s *disposableStackObject) deferCallback(call FunctionCall) Value {
	method, ok := assertCallable(call.Argument(0))
	if !ok {
		panic(s.val.runtime.NewTypeError("Disposal callback must be callable"))
	}
	s.capability.resources = append(s.capability.resources, disposableResource{
		method: method,
		call:   FunctionCall{This: _undefined},
	})
	return _undefined
}

func (r *Runtime) addDisposalError(previous, latest Value) Value {
	if previous == nil {
		return latest
	}
	proto := r.getSuppressedError().Get("prototype").(*Object)
	return r.newSuppressedError(proto, latest, previous, _undefined)
}

func (r *Runtime) disposableStackDispose(call FunctionCall) Value {
	stack := r.thisDisposableStack(call.This, false)
	if stack.capability == nil {
		return _undefined
	}
	resources := stack.beginDisposal()
	var failure *Exception
	for i := len(resources) - 1; i >= 0; i-- {
		resource := resources[i]
		if ex := r.vm.try(func() { resource.method(resource.call) }); ex != nil {
			if failure != nil {
				ex = r.vm.exceptionFromValue(r.addDisposalError(failure.val, ex.val))
			}
			failure = ex
		}
	}
	if failure != nil {
		panic(failure)
	}
	return _undefined
}

type asyncDisposal struct {
	promise    *Promise
	resources  []disposableResource
	failure    Value
	needsAwait bool
	hasAwaited bool
}

func (r *Runtime) disposableStackDisposeAsync(call FunctionCall) Value {
	promise := r.newPromise(r.getPromisePrototype())
	var stack *disposableStackObject
	if ex := r.vm.try(func() { stack = r.thisDisposableStack(call.This, true) }); ex != nil {
		promise.reject(ex.val)
	} else if stack.capability == nil {
		promise.fulfill(_undefined)
	} else {
		disposal := &asyncDisposal{promise: promise, resources: stack.beginDisposal()}
		disposal.run()
	}
	return promise.val
}

func (d *asyncDisposal) run() {
	r := d.promise.val.runtime
	for len(d.resources) > 0 {
		last := len(d.resources) - 1
		resource := d.resources[last]
		d.resources[last] = disposableResource{}
		d.resources = d.resources[:last]

		if resource.method == nil {
			d.needsAwait = true
			continue
		}

		var promise *Object
		ex := r.vm.try(func() {
			var result Value
			if resource.syncFallback {
				// The sync fallback wraps exceptions but ignores the return value.
				p := r.newPromise(r.getPromisePrototype())
				if ex := r.vm.try(func() { resource.method(resource.call) }); ex != nil {
					p.reject(ex.val)
				} else {
					p.fulfill(_undefined)
				}
				result = p.val
			} else {
				result = resource.method(resource.call)
			}
			d.hasAwaited = true
			promise = r.promiseResolve(r.getPromise(), result)
		})
		if ex != nil {
			d.failure = r.addDisposalError(d.failure, ex.val)
			continue
		}
		d.await(promise.self.(*Promise))
		return
	}
	d.resources = nil
	if d.needsAwait && !d.hasAwaited {
		d.hasAwaited = true
		d.await(r.promiseResolve(r.getPromise(), _undefined).self.(*Promise))
		return
	}
	if d.failure == nil {
		d.promise.fulfill(_undefined)
	} else {
		d.promise.reject(d.failure)
	}
}

func (d *asyncDisposal) await(promise *Promise) {
	promise.addReactions(&promiseReaction{
		typ: promiseReactionFulfill,
		handler: &jobCallback{callback: func(FunctionCall) Value {
			d.run()
			return _undefined
		}},
	}, &promiseReaction{
		typ: promiseReactionReject,
		handler: &jobCallback{callback: func(call FunctionCall) Value {
			r := d.promise.val.runtime
			d.failure = r.addDisposalError(d.failure, call.Argument(0))
			d.run()
			return _undefined
		}},
	})
}

func (r *Runtime) newDisposableStackConstructor(name unistring.String, async bool) *Object {
	ctor := &Object{runtime: r}
	proto := r.NewObject()
	r.newNativeConstructOnly(ctor, func(_ []Value, newTarget *Object) *Object {
		if newTarget == nil {
			panic(r.needNew(name.String()))
		}
		return r.newDisposableStack(r.getPrototypeFromCtor(newTarget, ctor, proto), async, &disposeCapability{})
	}, proto, name, 0)

	o := proto.self
	o._putProp("constructor", ctor, true, false, true)
	o._putSym(SymToStringTag, valueProp(stringValueFromRaw(name), false, false, true))
	o.setOwnStr("disposed", &valueProperty{
		accessor:     true,
		configurable: true,
		getterFunc: r.newNativeFunc(func(call FunctionCall) Value {
			return valueBool(r.thisDisposableStack(call.This, async).capability == nil)
		}, "get disposed", 0),
	}, false)

	putMethod := func(name unistring.String, length int, method func(*disposableStackObject, FunctionCall) Value) {
		o._putProp(name, r.newNativeFunc(func(call FunctionCall) Value {
			stack := r.thisDisposableStack(call.This, async)
			if stack.capability == nil {
				panic(r.newError(r.getReferenceError(), "Disposable stack is already disposed"))
			}
			return method(stack, call)
		}, name, length), true, false, true)
	}
	putMethod("adopt", 2, (*disposableStackObject).adopt)
	putMethod("defer", 1, (*disposableStackObject).deferCallback)
	putMethod("move", 0, func(stack *disposableStackObject, _ FunctionCall) Value {
		moved := r.newDisposableStack(proto, async, stack.capability)
		stack.capability = nil
		return moved
	})
	putMethod("use", 1, (*disposableStackObject).use)

	if async {
		dispose := r.newNativeFunc(r.disposableStackDisposeAsync, "disposeAsync", 0)
		o._putProp("disposeAsync", dispose, true, false, true)
		o._putSym(SymAsyncDispose, valueProp(dispose, true, false, true))
	} else {
		dispose := r.newNativeFunc(r.disposableStackDispose, "dispose", 0)
		o._putProp("dispose", dispose, true, false, true)
		o._putSym(SymDispose, valueProp(dispose, true, false, true))
	}
	return ctor
}

func (r *Runtime) getDisposableStack() *Object {
	if r.global.DisposableStack == nil {
		r.global.DisposableStack = r.newDisposableStackConstructor("DisposableStack", false)
	}
	return r.global.DisposableStack
}

func (r *Runtime) getAsyncDisposableStack() *Object {
	if r.global.AsyncDisposableStack == nil {
		r.global.AsyncDisposableStack = r.newDisposableStackConstructor("AsyncDisposableStack", true)
	}
	return r.global.AsyncDisposableStack
}
