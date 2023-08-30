package goja_test

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"sync"
	"testing"

	"github.com/dop251/goja"
)

func TestWrapReflectFunc(t *testing.T) {
	vm := goja.New()
	done := make(chan struct{})
	ch := make(chan func(*goja.Runtime))
	var wait sync.WaitGroup
	wait.Add(1)
	go func() {
		defer wait.Done()
		select {
		case <-done:
			return
		case f := <-ch:
			f(vm)
		}
	}()
	vm.SetRunOnLoop(func(f func(vm *goja.Runtime)) {
		select {
		case <-done:
		case ch <- f:
		}
	})
	vm.Set(`close`, func() {
		close(done)
	})
	vm.Set(`println`, func(vals ...interface{}) {
		fmt.Println(vals...)
	})
	vm.Set(`newError`, func(str string) error {
		return errors.New(str)
	})
	vm.Set(`getInt64`, func() int64 {
		return math.MaxInt64
	})
	vm.Set(`getInt64s`, func() []int64 {
		return []int64{math.MaxInt64, -100}
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
	vm.Set(`getUint64s`, func() []uint64 {
		return []uint64{math.MaxUint64, 100}
	})
	vm.Set(`setUint64`, func(v uint64) {
		if v != math.MaxUint64 {
			t.Errorf(`uint64 want %v, but got %v`, uint64(math.MaxUint64), v)
			t.FailNow()
		}
	})
	vm.Set(`get64`, func() (int64, uint64) {
		return int64(math.MaxInt64), uint64(math.MaxUint64)
	})
	vm.Set(`failMessage`, func(args ...interface{}) {
		t.Error(args...)
		t.FailNow()
	})
	vm.Set(`parseInt64`, func(s string) (goja.Int64, error) {
		v, e := strconv.ParseInt(s, 10, 64)
		if e != nil {
			return 0, e
		}
		return goja.Int64(v), nil
	})
	vm.Set(`parseUint64`, func(s string) (goja.Int64, error) {
		v, e := strconv.ParseInt(s, 10, 64)
		if e != nil {
			return 0, e
		}
		return goja.Int64(v), nil
	})
	vm.RunScript(`test.js`, `
const MaxInt64='`+strconv.FormatInt(int64(math.MaxInt64), 10)+`'
const MaxUint64='`+strconv.FormatUint(uint64(math.MaxUint64), 10)+`'
function assertEqual(want,actual,label){
	if(want!=actual){
		failMessage(label,'want',want, ', but get',actual)
	}
}

function abc(){
	// int64
	setInt64(getInt64(GoNumber))
	assertEqual(MaxInt64,getInt64(GoNumber).toString(),'getInt64')
	assertEqual(getInt64(GoNumber),getInt64(GoNumber),'getInt64 ==')

	let v = parseInt64(GoNumber,"2")
	assertEqual(2,v,'getInt64 ==')
	assertEqual(3,v.Add(1),'Int64 add')
	assertEqual(1,v.Sub(1),'Int64 sub')
	assertEqual(-3,v.Sub(5),'Int64 sub5')
	assertEqual(6,v.Mul(3),'Int64 mul')
	assertEqual(3,v.Mul(3).Div(2),'Int64 div')
	assertEqual(2,v.Mul(5).Div(4),'Int64 mod')
	assertEqual(-2,v.Neg(),'Int64 neg')
	assertEqual(2,v.Neg().Abs(),'Int64 abs')
	assertEqual(2,v.And(3),'Int64 and')
	assertEqual(6,v.Or(4),'Int64 or')
	assertEqual(4,v.Xor(6),'Int64 xor')
	assertEqual(2,v.Not().Not(),'Int64 not')
	assertEqual(8,v.Lsh(2),'Int64 left shift')
	assertEqual(1,v.Rsh(1),'Int64 right shift')
	assertEqual(0,v.Cmp(2),'Int64 cmp 2')
	assertEqual(-1,v.Cmp(4),'Int64 cmp 4')
	assertEqual(1,v.Cmp(0),'Int64 cmp 0')
	
	// uint64
	setUint64(getUint64(GoRawNumber))
	assertEqual(MaxUint64,getUint64(GoNumber).toString(),'getUint64')
	assertEqual(getUint64(GoNumber),getUint64(GoNumber),'getUint64 ==')

	v = parseUint64(GoNumber,"2")
	assertEqual(2,v,'getInt64 ==')
	assertEqual(3,v.Add(1),'Uint64 add')
	assertEqual(1,v.Sub(1),'Uint64 sub')
	assertEqual(-3,v.Sub(5),'Uint64 sub5')
	assertEqual(6,v.Mul(3),'Uint64 mul')
	assertEqual(3,v.Mul(3).Div(2),'Uint64 div')
	assertEqual(2,v.Mul(5).Div(4),'Uint64 mod')
	assertEqual(2,v.And(3),'Uint64 and')
	assertEqual(6,v.Or(4),'Uint64 or')
	assertEqual(4,v.Xor(6),'Uint64 xor')
	assertEqual(2,v.Not().Not(),'Uint64 not')
	assertEqual(8,v.Lsh(2),'Uint64 left shift')
	assertEqual(1,v.Rsh(1),'Uint64 right shift')
	assertEqual(0,v.Cmp(2),'Uint64 cmp 2')
	assertEqual(-1,v.Cmp(4),'Uint64 cmp 4')
	assertEqual(1,v.Cmp(0),'Uint64 cmp 0')

	// slice 64
	const [i64,u64] = get64(GoRawNumber)
	assertEqual(MaxInt64,i64.toString(),'i64')
	assertEqual(MaxUint64,u64.toString(),'u64')

	let vals = getInt64s(GoRawNumber)
	assertEqual(MaxInt64,vals[0].toString(),'getInt64s')
	assertEqual(-100,vals[1],'getInt64s -100')
	vals.push(vals[0])
	assertEqual(MaxInt64,vals[0].toString(),'getInt64s')
	assertEqual(-100,vals[1],'getInt64s -100')
	assertEqual(MaxInt64,vals[2].toString(),'getInt64s 2')

	vals = getUint64s(GoRawNumber)
	assertEqual(MaxUint64,vals[0].toString(),'getUint64s')
	assertEqual(100,vals[1],'getUint64s 100')
	vals.push(vals[0])
	assertEqual(MaxUint64,vals[0].toString(),'getUint64s')
	assertEqual(100,vals[1],'getUint64s 100')
	assertEqual(MaxUint64,vals[2].toString(),'getUint64s 2')

	return 1
}
function err(){
	return newError(GoRaw,"ok")
}
function asyncErr(){
	const v= newError(GoAsyncRaw,"async")
	v.then((e)=>{
		assertEqual(e.Error(),"async","then")
		close()
	})
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

	call, _ = goja.AssertFunction(vm.Get(`asyncErr`))
	_, e = call(goja.Undefined())
	if e != nil {
		t.Errorf(`js asyncErr() err: %s`, e.Error())
		t.FailNow()
	}

	wait.Wait()
}
