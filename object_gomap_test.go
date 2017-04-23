package goja

import "testing"

func TestGomapProp(t *testing.T) {
	const SCRIPT = `
	o.a + o.b;
	`
	r := New()
	r.Set("o", map[string]interface{}{
		"a": 40,
		"b": 2,
	})
	v, err := r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
	if i := v.ToInteger(); i != 42 {
		t.Fatalf("Expected 42, got: %d", i)
	}
}

func TestGomapEnumerate(t *testing.T) {
	const SCRIPT = `
	var hasX = false;
	var hasY = false;
	for (var key in o) {
		switch (key) {
		case "x":
			if (hasX) {
				throw "Already have x";
			}
			hasX = true;
			break;
		case "y":
			if (hasY) {
				throw "Already have y";
			}
			hasY = true;
			break;
		default:
			throw "Unexpected property: " + key;
		}
	}
	hasX && hasY;
	`
	r := New()
	r.Set("o", map[string]interface{}{
		"x": 40,
		"y": 2,
	})
	v, err := r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	if !v.StrictEquals(valueTrue) {
		t.Fatalf("Expected true, got %v", v)
	}
}

func TestGomapDeleteWhileEnumerate(t *testing.T) {
	const SCRIPT = `
	var hasX = false;
	var hasY = false;
	for (var key in o) {
		switch (key) {
		case "x":
			if (hasX) {
				throw "Already have x";
			}
			hasX = true;
			delete o.y;
			break;
		case "y":
			if (hasY) {
				throw "Already have y";
			}
			hasY = true;
			delete o.x;
			break;
		default:
			throw "Unexpected property: " + key;
		}
	}
	hasX && !hasY || hasY && !hasX;
	`
	r := New()
	r.Set("o", map[string]interface{}{
		"x": 40,
		"y": 2,
	})
	v, err := r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	if !v.StrictEquals(valueTrue) {
		t.Fatalf("Expected true, got %v", v)
	}
}

func TestGomapInstanceOf(t *testing.T) {
	const SCRIPT = `
	(o instanceof Object) && !(o instanceof Error);
	`
	r := New()
	r.Set("o", map[string]interface{}{})
	v, err := r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	if !v.StrictEquals(valueTrue) {
		t.Fatalf("Expected true, got %v", v)
	}
}

func TestGomapTypeOf(t *testing.T) {
	const SCRIPT = `
	typeof o;
	`
	r := New()
	r.Set("o", map[string]interface{}{})
	v, err := r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	if !v.StrictEquals(asciiString("object")) {
		t.Fatalf("Expected object, got %v", v)
	}
}

func TestGomapProto(t *testing.T) {
	const SCRIPT = `
	o.hasOwnProperty("test");
	`
	r := New()
	r.Set("o", map[string]interface{}{
		"test": 42,
	})
	v, err := r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	if !v.StrictEquals(valueTrue) {
		t.Fatalf("Expected true, got %v", v)
	}
}

func TestGoMapExtensibility(t *testing.T) {
	const SCRIPT = `
	"use strict";
	o.test = 42;
	Object.preventExtensions(o);
	o.test = 43;
	try {
		o.test1 = 42;
	} catch (e) {
		if (!(e instanceof TypeError)) {
			throw e;
		}
	}
	o.test === 43 && o.test1 === undefined;
	`

	r := New()
	r.Set("o", map[string]interface{}{})
	v, err := r.RunString(SCRIPT)
	if err != nil {
		if ex, ok := err.(*Exception); ok {
			t.Fatal(ex.String())
		} else {
			t.Fatal(err)
		}
	}

	if !v.StrictEquals(valueTrue) {
		t.Fatalf("Expected true, got %v", v)
	}

}
