package goja

import "reflect"

var defaultOptions = options{}

type Option interface {
	apply(*options)
}
type Caller interface {
	// Before the native function is called, calling Before can be used as an interceptor, and returning errr will panic(r.NewGoError(err))
	Before(call *FunctionCall) (err error)
	// Control how to call native functions, for example, you can start a new goroutine to call native functions
	Call(callSlice bool, callable reflect.Value, in []reflect.Value) (out []reflect.Value, err error)
	// Called before returning the function call result out to js, used to convert the return value to js or filter the function return value
	After(out []reflect.Value) (result Value, err error)
}
type CallerFactory interface {
	// Get a Caller
	Get() Caller
	// If you want to reuse, you can implement the Put function, otherwise save it as an empty implementation.
	Put(caller Caller)
}
type options struct {
	callerFactory CallerFactory
}
type funcOption struct {
	f func(*options)
}

func (fdo *funcOption) apply(do *options) {
	fdo.f(do)
}
func newFuncOption(f func(*options)) *funcOption {
	return &funcOption{
		f: f,
	}
}
func WithCallerFactory(callerFactory CallerFactory) Option {
	return newFuncOption(func(o *options) {
		o.callerFactory = callerFactory
	})
}
