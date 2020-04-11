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

func TestGoMapWithProto(t *testing.T) {
	vm := New()
	m := map[string]interface{}{
		"t": "42",
	}
	vm.Set("m", m)
	_, err := vm.RunString(TESTLIB + `
	(function() {
	'use strict';
	var proto = {};
	var getterAllowed = false;
	var setterAllowed = false;
	var tHolder = "proto t";
	Object.defineProperty(proto, "t", {
		get: function() {
			if (!getterAllowed) throw new Error("getter is called");
			return tHolder;
		},
		set: function(v) {
			if (!setterAllowed) throw new Error("setter is called");
			tHolder = v;
		}
	});
	var t1Holder;
	Object.defineProperty(proto, "t1", {
		get: function() {
			return t1Holder;
		},
		set: function(v) {
			t1Holder = v;
		}
	});
	Object.setPrototypeOf(m, proto);
	assert.sameValue(m.t, "42");
	m.t = 43;
	assert.sameValue(m.t, 43);
	t1Holder = "test";
	assert.sameValue(m.t1, "test");
	m.t1 = "test1";
	assert.sameValue(m.t1, "test1");
	delete m.t;
	getterAllowed = true;
	assert.sameValue(m.t, "proto t", "after delete");
	setterAllowed = true;
	m.t = true;
	assert.sameValue(m.t, true);
	assert.sameValue(tHolder, true);
	Object.preventExtensions(m);
	assert.throws(TypeError, function() {
		m.t2 = 1;
	});
	m.t1 = "test2";
	assert.sameValue(m.t1, "test2");
	})();
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGoMapProtoProp(t *testing.T) {
	const SCRIPT = `
	(function() {
	"use strict";
	var proto = {};
	Object.defineProperty(proto, "ro", {value: 42});
	Object.setPrototypeOf(m, proto);
	assert.throws(TypeError, function() {
		m.ro = 43;
	});
	Object.defineProperty(m, "ro", {value: 43});
	assert.sameValue(m.ro, 43);
	})();
	`

	r := New()
	r.Set("m", map[string]interface{}{})
	_, err := r.RunString(TESTLIB + SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGoMapProtoPropChain(t *testing.T) {
	const SCRIPT = `
	(function() {
	"use strict";
	var p1 = Object.create(null);
	m.__proto__ = p1;
	
	Object.defineProperty(p1, "test", {
		value: 42
	});
	
	Object.defineProperty(m, "test", {
		value: 43,
		writable: true,
	});
	var o = Object.create(m);
	o.test = 44;
	assert.sameValue(o.test, 44);

	var sym = Symbol(true);
	Object.defineProperty(p1, sym, {
		value: 42
	});
	
	Object.defineProperty(m, sym, {
		value: 43,
		writable: true,
	});
	o[sym] = 44;
	assert.sameValue(o[sym], 44);
	})();
	`

	r := New()
	r.Set("m", map[string]interface{}{})
	_, err := r.RunString(TESTLIB + SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGoMapUnicode(t *testing.T) {
	const SCRIPT = `
	Object.setPrototypeOf(m, s);
	if (m.Тест !== "passed") {
		throw new Error("m.Тест: " + m.Тест);
	}
	m["é"];
	`
	type S struct {
		Тест string
	}
	vm := New()
	m := map[string]interface{}{
		"é": 42,
	}
	s := S{
		Тест: "passed",
	}
	vm.Set("m", m)
	vm.Set("s", &s)
	res, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || !res.StrictEquals(valueInt(42)) {
		t.Fatalf("Unexpected value: %v", res)
	}
}
