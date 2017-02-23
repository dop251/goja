package goja

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func testInterruptRace(t testing.TB) {
	var (
		run       = make(chan struct{})
		sig       = func() { run <- struct{}{} }
		wait      = func() { <-run }
		vm        = New()
		interrupt = errors.New("test")
	)

	go func() {
		defer sig()
		wait()
		time.Sleep(time.Millisecond * 10)
		vm.Interrupt(interrupt)
	}()

	defer wait()
	sig()

	_, err := vm.RunString("for(;;) for(var t = Date.now() + 100; Date.now() < t;);")
	switch err := err.(type) {
	case *InterruptedError:
		if v := err.Value(); v != interrupt {
			t.Errorf("InterruptedError.Value = %#+v; want %#+v", v, interrupt)
		}
	default:
		t.Errorf("RunString() = %#+v; want %v", err, reflect.TypeOf((*InterruptedError)(nil)))
	}
}

func TestInterruptRace(t *testing.T) {
	testInterruptRace(t)
}
