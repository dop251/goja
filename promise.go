package goja

// Callable represents a JavaScript function that can be called from Go.
type JSCallable func(this Value, args ...Value) (result Value, err Value)

// AssertFunction checks if the Value is a function and returns a Callable.
func AssertJSFunction(v Value) (JSCallable, bool) {
	if obj, ok := v.(*Object); ok {
		if f, ok := obj.self.assertCallable(); ok {
			return func(this Value, args ...Value) (ret, err Value) {
				defer func() {
					if x := recover(); x != nil {
						if ex, ok := x.(*uncatchableException); ok {
							err = obj.runtime.NewGoError(ex.err)
						} else {
							panic(x)
						}
					}
				}()
				ex := obj.runtime.vm.try(func() {
					ret = f(FunctionCall{
						This:      this,
						Arguments: args,
					})
				})
				if ex != nil {
					err = ex.val
				}
				vm := obj.runtime.vm
				vm.clearStack()
				if len(vm.callStack) == 0 {
					obj.runtime.leave()
				}
				return
			}, true
		}
	}
	return nil, false
}

type Promise struct {
	resolve, reject Callable
	promise         Value
}

func (p *Promise) Value() Value {
	return p.promise
}
func (p *Promise) Resolve(v Value) {
	p.resolve(_undefined, v)
}
func (p *Promise) Reject(reason Value) {
	p.reject(_undefined, reason)
}
func (p *Promise) receiver(call FunctionCall) Value {
	p.resolve, _ = AssertFunction(call.Argument(0))
	p.reject, _ = AssertFunction(call.Argument(1))
	return _undefined
}
func NewPromise(runtime *Runtime) (promise *Promise, e error) {
	return newPromise(runtime, runtime.Get(`Promise`))
}
func newPromise(runtime *Runtime, construct Value) (promise *Promise, e error) {
	var result Promise
	result.promise, e = runtime.New(construct,
		runtime.newNativeFunc(result.receiver, nil, "Promise.receiver", nil, 2),
	)
	if e != nil {
		return
	}
	promise = &result
	return
}

type promiseState uint8

const (
	pending promiseState = iota + 1
	fulfilled
	rejected
)

func (s promiseState) String() string {
	switch s {
	case fulfilled:
		return `fulfilled`
	case rejected:
		return `rejected`
	}
	return `pending`
}

type factoryPromise struct {
	runtime *Runtime
	ctor    *Object
}

func (f *factoryPromise) constructor(call ConstructorCall) *Object {
	runtime := f.runtime
	executor, ok := AssertJSFunction(call.Argument(0))
	if !ok {
		panic(runtime.NewTypeError(`Promise executor is not a function`))
	}
	newPromiseImpl(runtime, f.ctor, call.This).register(executor)
	return nil
}
func (f *factoryPromise) register() {
	self := f.ctor
	runtime := f.runtime
	e := self.Set(`resolve`, f.resolve)
	if e != nil {
		panic(runtime.NewGoError(e))
	}
	e = self.Set(`reject`, f.reject)
	if e != nil {
		panic(runtime.NewGoError(e))
	}
}

func (f *factoryPromise) resolve(call FunctionCall) Value {
	runtime := f.runtime
	promise, e := newPromise(runtime, f.ctor)
	if e != nil {
		panic(runtime.NewGoError(e))
	}
	promise.Resolve(call.Argument(0))
	return promise.Value()
}
func (f *factoryPromise) reject(call FunctionCall) Value {
	runtime := f.runtime
	promise, e := newPromise(runtime, f.ctor)
	if e != nil {
		panic(runtime.NewGoError(e))
	}
	promise.Reject(call.Argument(0))
	return promise.Value()
}

type promiseImpl struct {
	runtime   *Runtime
	ctor      *Object
	self      *Object
	state     promiseState
	callbacks []promiseCallback
	result    Value
	err       Value
	completed bool
}
type promiseCallback struct {
	onFulfilled, onRejected, onFinally Callable
}

func newPromiseImpl(runtime *Runtime, ctor *Object, self *Object) *promiseImpl {
	return &promiseImpl{
		runtime: runtime,
		ctor:    ctor,
		self:    self,
		state:   pending,
		result:  _undefined,
		err:     _undefined,
	}
}
func (impl *promiseImpl) register(executor JSCallable) {
	self := impl.self
	runtime := impl.runtime
	e := self.Set(`toString`, impl.toString)
	if e != nil {
		panic(runtime.NewGoError(e))
	}
	e = self.Set(`then`, impl.then)
	if e != nil {
		panic(runtime.NewGoError(e))
	}
	e = self.Set(`catch`, impl.catch)
	if e != nil {
		panic(runtime.NewGoError(e))
	}
	e = self.Set(`finally`, impl.finally)
	if e != nil {
		panic(runtime.NewGoError(e))
	}

	_, err := executor(_undefined,
		runtime.newNativeFunc(impl.resolve, nil, "resolve", nil, 1),
		runtime.newNativeFunc(impl.reject, nil, "reject", nil, 1),
	)
	if err != nil {
		impl.rejectHandle(err)
	}
}
func (impl *promiseImpl) String() string {
	return `Promise { <` + impl.state.String() + `> }`
}
func (impl *promiseImpl) toString(call FunctionCall) Value {
	return newStringValue(impl.String())
}
func (impl *promiseImpl) resolve(call FunctionCall) Value {
	impl.resolveHandle(call.Argument(0))
	return _undefined
}
func (impl *promiseImpl) reject(call FunctionCall) Value {
	impl.rejectHandle(call.Argument(0))
	return _undefined
}
func (impl *promiseImpl) resolveHandle(v Value) {
	if impl.state != pending {
		return
	}
	impl.state = fulfilled
	impl.result = v
	for _, callback := range impl.callbacks {
		if callback.onFulfilled != nil {
			callback.onFulfilled(_undefined, v)
		}
	}
	for _, callback := range impl.callbacks {
		if callback.onFinally != nil {
			callback.onFinally(_undefined)
		}
	}
	impl.completed = true
}
func (impl *promiseImpl) rejectHandle(v Value) {
	if impl.state != pending {
		return
	}
	impl.state = rejected
	impl.err = v
	for _, callback := range impl.callbacks {
		if callback.onRejected != nil {
			callback.onRejected(_undefined, v)
		}
	}
	impl.completed = true
}
func (impl *promiseImpl) then(call FunctionCall) Value {
	onFulfilled, _ := AssertJSFunction(call.Argument(0))
	onRejected, _ := AssertJSFunction(call.Argument(1))
	executor := newPromiseExecutor(impl.runtime, impl.ctor, onFulfilled, onRejected, nil)
	callback := promiseCallback{
		onFulfilled: executor.resolve,
		onRejected:  executor.reject,
	}
	if impl.completed {
		if impl.state == fulfilled {
			callback.onFulfilled(_undefined, impl.result)
		} else {
			callback.onRejected(_undefined, impl.err)
		}
	} else {
		impl.callbacks = append(impl.callbacks, callback)
	}
	return executor.result
}
func (impl *promiseImpl) catch(call FunctionCall) Value {
	onRejected, _ := AssertJSFunction(call.Argument(0))
	executor := newPromiseExecutor(impl.runtime, impl.ctor, nil, onRejected, nil)
	callback := promiseCallback{
		onFulfilled: executor.resolve,
		onRejected:  executor.reject,
	}
	if impl.completed {
		if impl.state == fulfilled {
			callback.onFulfilled(_undefined, impl.result)
		} else {
			callback.onRejected(_undefined, impl.err)
		}
	} else {
		impl.callbacks = append(impl.callbacks, callback)
	}
	return executor.result
}
func (impl *promiseImpl) finally(call FunctionCall) Value {
	onFinally, _ := AssertJSFunction(call.Argument(0))
	executor := newPromiseExecutor(impl.runtime, impl.ctor, nil, nil, onFinally)
	callback := promiseCallback{
		onFulfilled: executor.resolve,
		onRejected:  executor.reject,
	}
	if impl.completed {
		if impl.state == fulfilled {
			callback.onFulfilled(_undefined, impl.result)
		} else {
			callback.onRejected(_undefined, impl.err)
		}
	} else {
		impl.callbacks = append(impl.callbacks, callback)
	}
	return executor.result
}

type promiseExecutor struct {
	runtime                                  *Runtime
	onFulfilled, onRejected, onFinally       JSCallable
	argResolve, argReject                    Value
	selfResolve, selfReject, resolve, reject Callable
	result                                   Value
}

func (p *promiseExecutor) handle(call FunctionCall) Value {
	p.argResolve = call.Argument(0)
	p.argReject = call.Argument(1)
	p.selfResolve, _ = AssertFunction(p.argResolve)
	p.selfReject, _ = AssertFunction(p.argReject)
	return _undefined
}
func (p *promiseExecutor) resolveHandle(call FunctionCall) Value {
	var (
		result Value = _undefined
		e      Value
	)
	if p.onFinally != nil {
		result, e = p.onFinally(_undefined)
		if e != nil {
			p.selfReject(_undefined, e)
			return _undefined
		}
	} else if p.onFulfilled != nil {
		result, e = p.onFulfilled(_undefined, call.Argument(0))
		if e != nil {
			p.selfReject(_undefined, e)
			return _undefined
		}
	}
	p.resolveResult(result)
	return _undefined
}
func (p *promiseExecutor) rejectHandle(call FunctionCall) Value {
	var (
		result Value = _undefined
		e      Value
	)
	if p.onFinally != nil {
		result, e = p.onFinally(_undefined)
		if e != nil {
			p.selfReject(_undefined, e)
			return _undefined
		}
	} else if p.onRejected != nil {
		result, e = p.onRejected(_undefined, call.Argument(0))
		if e != nil {
			p.selfReject(_undefined, e)
			return _undefined
		}
	} else {
		p.selfReject(_undefined, call.Argument(0))
		return _undefined
	}
	p.resolveResult(result)
	return _undefined
}
func (p *promiseExecutor) resolveResult(val Value) {
	if obj, ok := val.(*Object); ok {
		if callable, ok := AssertJSFunction(obj.Get(`then`)); ok {
			_, e := callable(_undefined,
				p.argResolve,
				p.argReject,
			)
			if e != nil {
				p.selfReject(_undefined, e)
			}
			return
		}
	}
	p.selfResolve(_undefined, val)
}
func newPromiseExecutor(runtime *Runtime, ctor *Object,
	onFulfilled, onRejected, onFinally JSCallable,
) (executor *promiseExecutor) {
	executor = &promiseExecutor{
		runtime:     runtime,
		onFulfilled: onFulfilled,
		onRejected:  onRejected,
		onFinally:   onFinally,
	}
	executor.resolve, _ = AssertFunction(runtime.newNativeFunc(executor.resolveHandle, nil, "onFulfilled", nil, 1))
	executor.reject, _ = AssertFunction(runtime.newNativeFunc(executor.rejectHandle, nil, "onRejected", nil, 1))
	executor.result, _ = runtime.New(ctor,
		runtime.newNativeFunc(executor.handle, nil, "executor", nil, 2),
	)
	return executor
}

func (r *Runtime) initPromise() {
	var factory factoryPromise
	factory.runtime = r
	factory.ctor = r.newNativeConstructor(factory.constructor, "Promise", 1)
	r.addToGlobal(`Promise`, factory.ctor)
	factory.register()
}
