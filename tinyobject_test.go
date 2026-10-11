package goja

import (
	"runtime"
	"slices"
	"testing"
	"time"
	"weak"

	"github.com/dop251/goja/unistring"
)

func TestTinyClassPropsBranching(t *testing.T) {
	vm := New()

	obj1 := vm.NewObject()
	obj1.Set("a0", 1)
	obj1.Set("b1", 2)
	obj1.Set("c1", 3)

	var c1 weak.Pointer[tinyClass]
	var b1Class *tinyClass
	if o, ok := obj1.self.(*tinyObject); ok {
		c1 = weak.Make(o.class)
		b1Class = o.class.parent
	} else {
		t.Fatal("not a tinyObject")
	}

	obj2 := vm.NewObject()
	obj2.Set("a0", 1)
	obj2.Set("b1", 2)
	obj2.Set("c2", 3) // branch

	if keys := obj1.Keys(); !slices.Equal(keys, []string{"a0", "b1", "c1"}) {
		t.Fatal(keys)
	}

	if keys := obj2.Keys(); !slices.Equal(keys, []string{"a0", "b1", "c2"}) {
		t.Fatal(keys)
	}

	obj1 = nil
	for range 5 {
		runtime.GC()
		runtime.GC()
		if c1.Value() == nil {
			goto ok
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("obj1 class is not freed")
ok:
	if b1Class.singlePropTransition.class.Value() != nil {
		t.Fatal("b1 class single property is not free")
	}
	obj3 := vm.NewObject()
	obj3.Set("a0", 1)
	obj3.Set("b1", 2)
	obj3.Set("c3", 3) // branch again, but the single transition should be re-used because it's now free

	if b1Class.singlePropTransition.class.Value() == nil {
		t.Fatal("b1 class single property is not re-used")
	}

	if keys := obj2.Keys(); !slices.Equal(keys, []string{"a0", "b1", "c2"}) {
		t.Fatal(keys)
	}

	if keys := obj3.Keys(); !slices.Equal(keys, []string{"a0", "b1", "c3"}) {
		t.Fatal(keys)
	}

}

func BenchmarkTinyObject(b *testing.B) {
	keys := make([]unistring.String, 0)
	rootClass := &tinyClass{
		keys: &keys,
	}

	class1 := rootClass.getForProp("a")
	class2 := class1.getForProp("key")
	class3 := class2.getForProp("another_key")
	class4 := class3.getForProp("b")

	//for _, key := range class4.keys {
	//	b.Logf("key: %v", unsafe.StringData(string(key)))
	//}

	o := &tinyObject{
		class:  class4,
		values: []Value{nil, nil, nil, nil},
	}
	for i := 0; i < b.N; i++ {
		o.getOwnPropStr("b")
	}
}

func BenchmarkTinyObjectMap(b *testing.B) {
	m := map[unistring.String]Value{
		"a":           nil,
		"key":         nil,
		"another_key": nil,
		"b":           nil,
	}
	for i := 0; i < b.N; i++ {
		_ = m["b"]
	}
}
