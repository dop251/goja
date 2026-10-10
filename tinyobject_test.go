package goja

import (
	"testing"

	"github.com/dop251/goja/unistring"
)

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
