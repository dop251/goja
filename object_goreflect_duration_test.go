package goja_test

import (
	"math"
	"testing"
	"time"

	"github.com/dop251/goja"
)

func TestGoReflectDuration(t *testing.T) {
	vm := goja.New()
	var expect = time.Duration(math.MaxInt64)
	vm.Set(`make`, func() time.Duration {
		return expect
	})
	vm.Set(`handle`, func(d time.Duration) {
		if d.String() != expect.String() {
			t.Fatal(`expect`, expect, `, but get`, d)
		}
	})
	_, e := vm.RunString(`
var d=make()
handle(d)
`)
	if e != nil {
		t.Fatal(e)
	}
}
