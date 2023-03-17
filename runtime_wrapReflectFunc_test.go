package goja_test

import (
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/dop251/goja"
)

func TestWrapReflectFunc(t *testing.T) {
	vm := goja.New()
	vm.Set(`println`, func(vals ...interface{}) {
		fmt.Println(vals...)
	})
	vm.Set(`newError`, func(str string) error {
		return errors.New(str)
	})
	vm.Set(`getInt64`, func() int64 {
		return math.MaxInt64
	})
	vm.Set(`setInt64`, func(v int64) {
		if v != math.MaxInt64 {
			t.Errorf(`int64 want %v, but got %v`, int64(math.MaxInt64), v)
			t.FailNow()
		}
	})
	vm.Set(`getUint64`, func() uint64 {
		return math.MaxUint64
	})
	vm.Set(`setUint64`, func(v uint64) {
		if v != math.MaxUint64 {
			t.Errorf(`uint64 want %v, but got %v`, uint64(math.MaxUint64), v)
			t.FailNow()
		}
	})
	vm.Set(`getInt64s`, func() []int64 {
		return []int64{1, math.MaxInt64}
	})
	vm.Set(`failMessage`, func(args ...interface{}) {
		t.Error(args)
		t.FailNow()
	})

	vm.RunScript(`test.js`, `
function abc(){
	setInt64(getInt64(GoNumber))
	if(getInt64(GoNumber) != '`+fmt.Sprint(int64(math.MaxInt64))+`'){
		failMessage('getInt64')
	}

	setUint64(getUint64(GoRawNumber))
	if(getUint64(GoNumber).toString() != '`+fmt.Sprint(uint64(math.MaxUint64))+`'){
		failMessage('getUint64 got',getUint64(GoNumber),', but want '+'`+fmt.Sprint(uint64(math.MaxUint64))+`')
	}
	return 1
}
function err(){
	return newError(GoRaw,"ok")
}
function asyncErr(){
	const v= newError(GoAsyncRaw,"async")
	println(v.then)
}`)
	var call goja.Callable
	call, _ = goja.AssertFunction(vm.Get(`abc`))
	v, e := call(goja.Undefined())
	if e != nil {
		t.Errorf(`js abc() err: %s`, e.Error())
		t.FailNow()
	}
	val := v.Export().(int64)
	if val != 1 {
		t.Errorf(`js abc() return %v, but want 1`, val)
		t.FailNow()
	}
	call, _ = goja.AssertFunction(vm.Get(`err`))
	v, e = call(goja.Undefined())
	if e != nil {
		t.Errorf(`js err() err: %s`, e.Error())
		t.FailNow()
	}
	v0 := v.Export().(error)
	if v0.Error() != "ok" {
		t.Errorf(`js err() want "ok", but got %s`, v0.Error())
		t.FailNow()
	}

	// call, _ = goja.AssertFunction(vm.Get(`asyncErr`))
	// _, e = call(goja.Undefined())
	// if e != nil {
	// 	t.Errorf(`js asyncErr() err: %s`, e.Error())
	// 	t.FailNow()
	// }

}
