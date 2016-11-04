package goja

import "testing"

func TestArray1(t *testing.T) {
	r := &Runtime{}
	a := r.newArray(nil)
	a.put(valueInt(0), asciiString("test"), true)
	if l := a.getStr("length").ToInteger(); l != 1 {
		t.Fatalf("Unexpected length: %d", l)
	}
}

func BenchmarkPut(b *testing.B) {
	v := &Object{}

	o := &baseObject{
		val:        v,
		extensible: true,
	}
	v.self = o

	o.init()

	var key Value = asciiString("test")
	var val Value = valueInt(123)

	for i := 0; i < b.N; i++ {
		o.put(key, val, false)
	}
}

func BenchmarkPutStr(b *testing.B) {
	v := &Object{}

	o := &baseObject{
		val:        v,
		extensible: true,
	}

	o.init()

	v.self = o

	var val Value = valueInt(123)

	for i := 0; i < b.N; i++ {
		o.putStr("test", val, false)
	}
}

func BenchmarkGet(b *testing.B) {
	v := &Object{}

	o := &baseObject{
		val:        v,
		extensible: true,
	}

	o.init()

	v.self = o
	var n Value = asciiString("test")

	for i := 0; i < b.N; i++ {
		o.get(n)
	}

}

func BenchmarkGetStr(b *testing.B) {
	v := &Object{}

	o := &baseObject{
		val:        v,
		extensible: true,
	}
	v.self = o

	o.init()

	for i := 0; i < b.N; i++ {
		o.getStr("test")
	}
}

func _toString(v Value) string {
	switch v := v.(type) {
	case asciiString:
		return string(v)
	default:
		return ""
	}
}

func BenchmarkToString1(b *testing.B) {
	v := asciiString("test")

	for i := 0; i < b.N; i++ {
		v.ToString()
	}
}

func BenchmarkToString2(b *testing.B) {
	v := asciiString("test")

	for i := 0; i < b.N; i++ {
		_toString(v)
	}
}

func BenchmarkConv(b *testing.B) {
	count := int64(0)
	for i := 0; i < b.N; i++ {
		count += valueInt(123).ToInteger()
	}
	if count == 0 {
		b.Fatal("zero")
	}
}

func BenchmarkArrayGetStr(b *testing.B) {
	b.StopTimer()
	r := New()
	v := &Object{runtime: r}

	a := &arrayObject{
		baseObject: baseObject{
			val:        v,
			extensible: true,
		},
	}
	v.self = a

	a.init()

	a.put(valueInt(0), asciiString("test"), false)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		a.getStr("0")
	}

}

func BenchmarkArrayGet(b *testing.B) {
	b.StopTimer()
	r := New()
	v := &Object{runtime: r}

	a := &arrayObject{
		baseObject: baseObject{
			val:        v,
			extensible: true,
		},
	}
	v.self = a

	a.init()

	var idx Value = valueInt(0)

	a.put(idx, asciiString("test"), false)

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		a.get(idx)
	}

}

func BenchmarkArrayPut(b *testing.B) {
	b.StopTimer()
	r := New()

	v := &Object{runtime: r}

	a := &arrayObject{
		baseObject: baseObject{
			val:        v,
			extensible: true,
		},
	}

	v.self = a

	a.init()

	var idx Value = valueInt(0)
	var val Value = asciiString("test")

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		a.put(idx, val, false)
	}

}

func BenchmarkToUTF8String(b *testing.B) {
	var s valueString = asciiString("test")
	for i := 0; i < b.N; i++ {
		_ = s.String()
	}
}

func BenchmarkAdd(b *testing.B) {
	var x, y Value
	x = valueInt(2)
	y = valueInt(2)

	for i := 0; i < b.N; i++ {
		if xi, ok := x.assertInt(); ok {
			if yi, ok := y.assertInt(); ok {
				x = valueInt(xi + yi)
			}
		}
	}
}

func BenchmarkAddString(b *testing.B) {
	var x, y Value

	tst := asciiString("22")
	x = asciiString("2")
	y = asciiString("2")

	for i := 0; i < b.N; i++ {
		var z Value
		if xi, ok := x.assertString(); ok {
			if yi, ok := y.assertString(); ok {
				z = xi.concat(yi)
			}
		}
		if !z.StrictEquals(tst) {
			b.Fatalf("Unexpected result %v", x)
		}
	}
}
