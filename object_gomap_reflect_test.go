package goja

import "testing"

func TestGoMapReflectGetSet(t *testing.T) {
	const SCRIPT = `
	m.c = m.a + m.b;
	`

	vm := New()
	m := map[string]string{
		"a": "4",
		"b": "2",
	}
	vm.Set("m", m)

	_, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	if c := m["c"]; c != "42" {
		t.Fatalf("Unexpected value: '%s'", c)
	}
}

func TestGoMapReflectIntKey(t *testing.T) {
	const SCRIPT = `
	m[2] = m[0] + m[1];
	`

	vm := New()
	m := map[int]int{
		0: 40,
		1: 2,
	}
	vm.Set("m", m)

	_, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	if c := m[2]; c != 42 {
		t.Fatalf("Unexpected value: '%d'", c)
	}
}

func TestGoMapReflectDelete(t *testing.T) {
	const SCRIPT = `
	delete m.a;
	`

	vm := New()
	m := map[string]string{
		"a": "4",
		"b": "2",
	}
	vm.Set("m", m)

	_, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	if _, exists := m["a"]; exists {
		t.Fatal("a still exists")
	}

	if b := m["b"]; b != "2" {
		t.Fatalf("Unexpected b: '%s'", b)
	}
}

func TestGoMapReflectJSON(t *testing.T) {
	const SCRIPT = `
	function f(m) {
		return JSON.stringify(m);
	}
	`

	vm := New()
	m := map[string]string{
		"t": "42",
	}
	_, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
	f := vm.Get("f")
	if call, ok := AssertFunction(f); ok {
		v, err := call(nil, ([]Value{vm.ToValue(m)})...)
		if err != nil {
			t.Fatal(err)
		}
		if !v.StrictEquals(asciiString(`{"t":"42"}`)) {
			t.Fatalf("Unexpected value: %v", v)
		}
	} else {
		t.Fatalf("Not a function: %v", f)
	}
}

func TestGoMapReflectProto(t *testing.T) {
	const SCRIPT = `
	m.hasOwnProperty("t");
	`

	vm := New()
	m := map[string]string{
		"t": "42",
	}
	vm.Set("m", m)
	v, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
	if !v.StrictEquals(valueTrue) {
		t.Fatalf("Expected true, got %v", v)
	}
}
