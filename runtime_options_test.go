package goja_test

import (
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/dop251/goja"
)

type caller struct {
	runtime *goja.Runtime
}

func (*caller) Before(call *goja.FunctionCall) (err error) {
	return nil
}

func (*caller) Call(callSlice bool, callable reflect.Value, in []reflect.Value) (out []reflect.Value, err error) {
	if callSlice {
		out = callable.CallSlice(in)
	} else {
		out = callable.Call(in)
	}
	return
}

func (c *caller) After(out []reflect.Value) (result goja.Value, err error) {
	switch len(out) {
	case 0:
		result = goja.Undefined()
	case 1:
		result = c.runtime.ToValue(out[0].Interface())
	default:
		s := make([]interface{}, len(out))
		for i, v := range out {
			s[i] = v.Interface()
		}
		result = c.runtime.ToValue(s)
	}
	return
}

type callerFactory struct {
	runtime *goja.Runtime
	pool    *sync.Pool
}

func (f *callerFactory) Reset(runtime *goja.Runtime) {
	f.runtime = runtime
	f.pool = &sync.Pool{
		New: func() interface{} {
			return &caller{
				runtime: runtime,
			}
		},
	}
}
func (f *callerFactory) Get() goja.Caller {
	return f.pool.Get().(*caller)
}
func (f *callerFactory) Put(caller goja.Caller) {
	f.pool.Put(caller)
}

func TestOptionCallerFactory(t *testing.T) {
	factory := &callerFactory{}
	r := goja.New(goja.WithCallerFactory(factory))
	factory.Reset(r)

	r.Set(`make`, func(str string) (string, error) {
		return str, errors.New(str)
	})
	r.Set(`checkErr`, func(call goja.FunctionCall) goja.Value {
		var result bool
		if e, ok := call.Argument(0).Export().(error); ok {
			result = e.Error() == call.Argument(1).String()
		}
		return r.ToValue(result)
	})
	_, e := r.RunString(`
var s0="cerberus is an idea"
var [str,e] = make(s0)
if(str!=s0){
	throw new Error("not equal")
}
if(!checkErr(e,s0)){
	throw new Error("not error")
}

`)
	if e != nil {
		t.Fatal(e)
	}
}
