package goja

import (
	"testing"
)

func TestProxy_Object_target_getPrototypeOf(t *testing.T) {
	const SCRIPT = `
    var proto = {};
	var obj = Object.create(proto);
	var proxy = new Proxy(obj, {});
	var p = Object.getPrototypeOf(proxy);
	assert.sameValue(proto, p);
	`

	testScript1(TESTLIB+SCRIPT, _undefined, t)
}

func TestProxy_Object_proxy_getPrototypeOf(t *testing.T) {
	const SCRIPT = `
    var proto = {};
	var proto2 = {};
	var obj = Object.create(proto);
	var proxy = new Proxy(obj, {
		getPrototypeOf: function(target) {
			return proto2;
		}
	});
	var p = Object.getPrototypeOf(proxy);
	assert.sameValue(proto2, p);
	`

	testScript1(TESTLIB+SCRIPT, _undefined, t)
}

func TestProxy_Object_native_proxy_getPrototypeOf(t *testing.T) {
	const SCRIPT = `
	var p = Object.getPrototypeOf(proxy);
	assert.sameValue(proto, p);
	`

	runtime := New()

	prototype := runtime.NewObject()
	runtime.Set("proto", prototype)

	target := runtime.NewObject()
	proxy := runtime.NewProxy(target, &ProxyTrapConfig{
		GetPrototypeOf: func(target *Object) *Object {
			return prototype
		},
	})
	runtime.Set("proxy", proxy)

	_, err := runtime.RunString(TESTLIB + SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
}

func TestProxy_Object_target_setPrototypeOf(t *testing.T) {
	const SCRIPT = `
    var proto = {};
	var obj = {};
	Object.setPrototypeOf(obj, proto);
	var proxy = new Proxy(obj, {});
	var p = Object.getPrototypeOf(proxy);
	assert.sameValue(proto, p);
	`

	testScript1(TESTLIB+SCRIPT, _undefined, t)
}

func TestProxy_Object_proxy_setPrototypeOf(t *testing.T) {
	const SCRIPT = `
    var proto = {};
	var proto2 = {};
	var obj = {};
	Object.setPrototypeOf(obj, proto);
	var proxy = new Proxy(obj, {
		setPrototypeOf: function(target, prototype) {
			return Object.setPrototypeOf(target, proto2);
		}
	});
	Object.setPrototypeOf(proxy, null);
	var p = Object.getPrototypeOf(proxy);
	assert.sameValue(proto2, p);
	`

	testScript1(TESTLIB+SCRIPT, _undefined, t)
}

func TestProxy_Object_target_isExtensible(t *testing.T) {
	const SCRIPT = `
	var obj = {};
	Object.seal(obj);
	var proxy = new Proxy(obj, {});
	Object.isExtensible(proxy);
	`

	testScript1(SCRIPT, valueFalse, t)
}

func TestProxy_proxy_isExtensible(t *testing.T) {
	const SCRIPT = `
	var obj = {};
	Object.seal(obj);
	var proxy = new Proxy(obj, {
		isExtensible: function(target) {
			return false;
		}
	});
	Object.isExtensible(proxy);
	`

	testScript1(SCRIPT, valueFalse, t)
}

func TestProxy_native_proxy_isExtensible(t *testing.T) {
	const SCRIPT = `
	(function() {
		Object.preventExtensions(target);
		return Object.isExtensible(proxy);
	})();
	`

	runtime := New()

	target := runtime.NewObject()
	runtime.Set("target", target)

	proxy := runtime.NewProxy(target, &ProxyTrapConfig{
		IsExtensible: func(target *Object) (success bool) {
			return false
		},
	})
	runtime.Set("proxy", proxy)

	val, err := runtime.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
	if val.ToBoolean() {
		t.Fatal()
	}
}

func TestProxy_Object_target_preventExtensions(t *testing.T) {
	const SCRIPT = `
	var obj = {
		canEvolve: true
	};
	var proxy = new Proxy(obj, {});
	Object.preventExtensions(proxy);
	proxy.canEvolve
	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestProxy_proxy_preventExtensions(t *testing.T) {
	const SCRIPT = `
	var obj = {
		canEvolve: true
	};
	var proxy = new Proxy(obj, {
		preventExtensions: function(target) {
			target.canEvolve = false;
			return false;
		}
	});
	Object.preventExtensions(proxy);
	proxy.canEvolve;
	`

	testScript1(SCRIPT, valueFalse, t)
}

func TestProxy_native_proxy_preventExtensions(t *testing.T) {
	const SCRIPT = `
	(function() {
		Object.preventExtensions(proxy);
		return proxy.canEvolve;
	})();
	`

	runtime := New()

	target := runtime.NewObject()
	target.Set("canEvolve", true)
	runtime.Set("target", target)

	proxy := runtime.NewProxy(target, &ProxyTrapConfig{
		PreventExtensions: func(target *Object) (success bool) {
			target.Set("canEvolve", false)
			return false
		},
	})
	runtime.Set("proxy", proxy)

	val, err := runtime.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
	if val.ToBoolean() {
		t.Fatal()
	}
}

func TestProxy_Object_target_getOwnPropertyDescriptor(t *testing.T) {
	const SCRIPT = `
	var desc = {
		configurable: false,
		enumerable: false,
		value: 42,
		writable: false 
	};

	var obj = {};
	Object.defineProperty(obj, "foo", desc);

	var proxy = new Proxy(obj, {});

	var desc2 = Object.getOwnPropertyDescriptor(proxy, "foo");
	desc2.value
	`

	testScript1(SCRIPT, valueInt(42), t)
}

func TestProxy_proxy_getOwnPropertyDescriptor(t *testing.T) {
	const SCRIPT = `
	var desc = {
		configurable: false,
		enumerable: false,
		value: 42,
		writable: false 
	};
	var proxy_desc = {
		configurable: false,
		enumerable: false,
		value: 24,
		writable: false 
	};

	var obj = {};
	Object.defineProperty(obj, "foo", desc);

	var proxy = new Proxy(obj, {
		getOwnPropertyDescriptor: function(target, property) {
			return proxy_desc;
		}
	});

	assert.throws(TypeError, function() {
		Object.getOwnPropertyDescriptor(proxy, "foo");
	});
	undefined;
	`

	testScript1(TESTLIB+SCRIPT, _undefined, t)
}

func TestProxy_native_proxy_getOwnPropertyDescriptor(t *testing.T) {
	const SCRIPT = `
	(function() {
		var desc = {
			configurable: true,
			enumerable: false,
			value: 42,
			writable: false 
		};
		var proxy_desc = {
			configurable: true,
			enumerable: false,
			value: 24,
			writable: false 
		};
		
		var obj = {};
		Object.defineProperty(obj, "foo", desc);

		return function(constructor) {
			var proxy = constructor(obj, proxy_desc);

			var desc2 = Object.getOwnPropertyDescriptor(proxy, "foo");
			return desc2.value
		}
	})();
	`

	runtime := New()

	constructor := func(call FunctionCall) Value {
		target := call.Argument(0).(*Object)
		proxyDesc := call.Argument(1).(*Object)

		return runtime.NewProxy(target, &ProxyTrapConfig{
			GetOwnPropertyDescriptor: func(target *Object, prop string) PropertyDescriptor {
				return runtime.toPropertyDescriptor(proxyDesc)
			},
		}).proxy.val
	}

	val, err := runtime.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	if c, ok := val.(*Object).self.assertCallable(); ok {
		val := c(FunctionCall{
			This:      val,
			Arguments: []Value{runtime.ToValue(constructor)},
		})
		if i := val.ToInteger(); i != 24 {
			t.Fatalf("val: %d", i)
		}
	} else {
		t.Fatal("not a function")
	}
}

func TestProxy_native_proxy_getOwnPropertyDescriptor_non_existing(t *testing.T) {
	vm := New()
	proxy := vm.NewProxy(vm.NewObject(), &ProxyTrapConfig{
		GetOwnPropertyDescriptor: func(target *Object, prop string) (propertyDescriptor PropertyDescriptor) {
			return // empty PropertyDescriptor
		},
	})
	vm.Set("proxy", proxy)
	res, err := vm.RunString(`Object.getOwnPropertyDescriptor(proxy, "foo") === undefined`)
	if err != nil {
		t.Fatal(err)
	}
	if res != valueTrue {
		t.Fatal(res)
	}
}

func TestProxy_Object_target_defineProperty(t *testing.T) {
	const SCRIPT = `
	var obj = {};
	var proxy = new Proxy(obj, {});
	Object.defineProperty(proxy, "foo", {
		value: "test123"
	});
	proxy.foo;
	`

	testScript1(SCRIPT, asciiString("test123"), t)
}

func TestProxy_proxy_defineProperty(t *testing.T) {
	const SCRIPT = `
	var obj = {};
	var proxy = new Proxy(obj, {
		defineProperty: function(target, prop, descriptor) {
			target.foo = "321tset";
			return true;
		}
	});
	Object.defineProperty(proxy, "foo", {
		value: "test123"
	});
	proxy.foo;
	`

	testScript1(SCRIPT, asciiString("321tset"), t)
}

func TestProxy_native_proxy_defineProperty(t *testing.T) {
	const SCRIPT = `
	Object.defineProperty(proxy, "foo", {
		value: "test123"
	});
	proxy.foo;
	`

	runtime := New()

	target := runtime.NewObject()

	proxy := runtime.NewProxy(target, &ProxyTrapConfig{
		DefineProperty: func(target *Object, key string, propertyDescriptor PropertyDescriptor) (success bool) {
			target.Set("foo", "321tset")
			return true
		},
	})
	runtime.Set("proxy", proxy)

	val, err := runtime.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
	if s := val.String(); s != "321tset" {
		t.Fatalf("val: %s", s)
	}
}

func TestProxy_target_has_in(t *testing.T) {
	const SCRIPT = `
	var obj = {
		secret: true
	};
	var proxy = new Proxy(obj, {});
	
	"secret" in proxy
	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestProxy_proxy_has_in(t *testing.T) {
	const SCRIPT = `
	var obj = {
		secret: true
	};
	var proxy = new Proxy(obj, {
		has: function(target, key) {
			return key !== "secret";
		}
	});
	
	"secret" in proxy
	`

	testScript1(SCRIPT, valueFalse, t)
}

func TestProxy_target_has_with(t *testing.T) {
	const SCRIPT = `
	var obj = {
		secret: true
	};
	var proxy = new Proxy(obj, {});
	
	with(proxy) {
		(secret);
	}
	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestProxy_proxy_has_with(t *testing.T) {
	const SCRIPT = `
	var obj = {
		secret: true
	};
	var proxy = new Proxy(obj, {
		has: function(target, key) {
			return key !== "secret";
		}
	});
	
	var thrown = false;
	try {
		with(proxy) {
			(secret);
		}
	} catch (e) {
		if (e instanceof ReferenceError) {
			thrown = true;
		} else {
			throw e;
		}
	}
	thrown;
	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestProxy_target_get(t *testing.T) {
	const SCRIPT = `
	var obj = {};
	var proxy = new Proxy(obj, {});
	Object.defineProperty(proxy, "foo", {
		value: "test123"
	});
	proxy.foo;
	`

	testScript1(SCRIPT, asciiString("test123"), t)
}

func TestProxy_proxy_get(t *testing.T) {
	const SCRIPT = `
	var obj = {};
	var proxy = new Proxy(obj, {
		get: function(target, prop, receiver) {
			return "321tset"
		}
	});
	Object.defineProperty(proxy, "foo", {
		value: "test123",
		configurable: true,
	});
	proxy.foo;
	`

	testScript1(SCRIPT, asciiString("321tset"), t)
}

func TestProxy_proxy_get_json_stringify(t *testing.T) {
	const SCRIPT = `
	var obj = {};
	var propValue = "321tset";
	var _handler, _target, _prop, _receiver;
	var proxy = new Proxy(obj, {
		ownKeys: function() {
			return ["foo"];
		},
		getOwnPropertyDescriptor: function(target, prop) {
			if (prop === "foo") {
				return {
					value: propValue,
					enumerable: true,
					configurable: true
				}
			}
		},
		get: function(target, prop, receiver) {
			if (prop === "foo") {
				_prop = prop;
				_receiver = receiver;
				return propValue;
			}
			return obj[prop];
		}
	});
	var res = JSON.stringify(proxy);
	assert.sameValue(res, '{"foo":"321tset"}');
	assert.sameValue(_prop, "foo");
	assert.sameValue(_receiver, proxy);
	`

	testScript1(TESTLIB+SCRIPT, _undefined, t)
}

func TestProxy_native_proxy_get(t *testing.T) {
	vm := New()
	propValue := vm.ToValue("321tset")
	obj := vm.NewObject()
	proxy := vm.NewProxy(obj, &ProxyTrapConfig{
		OwnKeys: func(*Object) *Object {
			return vm.newArrayValues([]Value{vm.ToValue("foo")})
		},
		GetOwnPropertyDescriptor: func(target *Object, prop string) (propertyDescriptor PropertyDescriptor) {
			if prop == "foo" {
				return PropertyDescriptor{
					Value:        propValue,
					Enumerable:   FLAG_TRUE,
					Configurable: FLAG_TRUE,
				}
			}
			return PropertyDescriptor{}
		},
		Get: func(target *Object, property string, receiver *Object) (value Value) {
			if property == "foo" {
				return propValue
			}
			return obj.Get(property)
		},
	})
	vm.Set("proxy", proxy)
	res, err := vm.RunString(`JSON.stringify(proxy)`)
	if err != nil {
		t.Fatal(err)
	}
	if !res.SameAs(asciiString(`{"foo":"321tset"}`)) {
		t.Fatalf("res: %v", res)
	}
	res, err = vm.RunString(`proxy[Symbol.toPrimitive]`)
	if err != nil {
		t.Fatal(err)
	}
	if !IsUndefined(res) {
		t.Fatalf("res: %v", res)
	}

	res, err = vm.RunString(`proxy.hasOwnProperty(Symbol.toPrimitive)`)
	if err != nil {
		t.Fatal(err)
	}
	if !res.SameAs(valueFalse) {
		t.Fatalf("res: %v", res)
	}

	res, err = vm.RunString(`proxy.toString()`)
	if err != nil {
		t.Fatal(err)
	}
	if !res.SameAs(asciiString(`[object Object]`)) {
		t.Fatalf("res: %v", res)
	}
}

func TestProxy_target_set_prop(t *testing.T) {
	const SCRIPT = `
	var obj = {};
	var proxy = new Proxy(obj, {});
	proxy.foo = "test123";
	proxy.foo;
	`

	testScript1(SCRIPT, asciiString("test123"), t)
}

func TestProxy_proxy_set_prop(t *testing.T) {
	const SCRIPT = `
	var obj = {};
	var proxy = new Proxy(obj, {
		set: function(target, prop, receiver) {
			target.foo = "321tset";
			return true;
		}
	});
	proxy.foo = "test123";
	proxy.foo;
	`

	testScript1(SCRIPT, asciiString("321tset"), t)
}
func TestProxy_target_set_associative(t *testing.T) {
	const SCRIPT = `
	var obj = {};
	var proxy = new Proxy(obj, {});
	proxy["foo"] = "test123";
	proxy.foo;
	`

	testScript1(SCRIPT, asciiString("test123"), t)
}

func TestProxy_proxy_set_associative(t *testing.T) {
	const SCRIPT = `
	var obj = {};
	var proxy = new Proxy(obj, {
		set: function(target, property, value, receiver) {
			target["foo"] = "321tset";
			return true;
		}
	});
	proxy["foo"] = "test123";
	proxy.foo;
	`

	testScript1(SCRIPT, asciiString("321tset"), t)
}

func TestProxy_target_delete(t *testing.T) {
	const SCRIPT = `
	var obj = {
		foo: "test"
	};
	var proxy = new Proxy(obj, {});
	delete proxy.foo;

	proxy.foo;
	`

	testScript1(SCRIPT, _undefined, t)
}

func TestProxy_proxy_delete(t *testing.T) {
	const SCRIPT = `
	var obj = {
		foo: "test"
	};
	var proxy = new Proxy(obj, {
		deleteProperty: function(target, prop) {
			return true;
		}
	});
	delete proxy.foo;

	proxy.foo;
	`

	testScript1(SCRIPT, asciiString("test"), t)
}

func TestProxy_target_keys(t *testing.T) {
	const SCRIPT = `
	var obj = {
		foo: "test"
	};
	var proxy = new Proxy(obj, {});

	var keys = Object.keys(proxy);
	if (keys.length != 1) {
		throw new Error("assertion error");
	}
	`

	testScript1(SCRIPT, _undefined, t)
}

func TestProxy_proxy_keys(t *testing.T) {
	const SCRIPT = `
	var obj = {
		foo: "test"
	};
	var proxy = new Proxy(obj, {
		ownKeys: function(target) {
			return ["foo", "bar"];
		}
	});

	var keys = Object.keys(proxy);
	if (keys.length !== 1) {
		throw new Error("length is "+keys.length);
	}
	if (keys[0] !== "foo") {
		throw new Error("keys[0] is "+keys[0]);
	}
	`

	testScript1(SCRIPT, _undefined, t)
}

func TestProxy_target_call(t *testing.T) {
	const SCRIPT = `
	var obj = function() {
		return "test"
	}
	
	var proxy = new Proxy(obj, {});

	proxy();
	`

	testScript1(SCRIPT, asciiString("test"), t)
}

func TestProxy_proxy_call(t *testing.T) {
	const SCRIPT = `
	var obj = function() {
		return "test"
	}
	
	var proxy = new Proxy(obj, {
		apply: function(target, thisArg, args) {
			return "tset"
		}
	});

	proxy();
	`

	testScript1(SCRIPT, asciiString("tset"), t)
}

func TestProxy_target_func_apply(t *testing.T) {
	const SCRIPT = `
	var obj = function() {
		return "test"
	}
	
	var proxy = new Proxy(obj, {});

	proxy.apply();
	`

	testScript1(SCRIPT, asciiString("test"), t)
}

func TestProxy_proxy_func_apply(t *testing.T) {
	const SCRIPT = `
	var obj = function() {
		return "test"
	}
	
	var proxy = new Proxy(obj, {
		apply: function(target, thisArg, args) {
			return "tset"
		}
	});

	proxy.apply();
	`

	testScript1(SCRIPT, asciiString("tset"), t)
}

func TestProxy_target_func_call(t *testing.T) {
	const SCRIPT = `
	var obj = function() {
		return "test"
	}
	
	var proxy = new Proxy(obj, {});

	proxy.call();
	`

	testScript1(SCRIPT, asciiString("test"), t)
}

func TestProxy_proxy_func_call(t *testing.T) {
	const SCRIPT = `
	var obj = function() {
		return "test"
	}
	
	var proxy = new Proxy(obj, {
		apply: function(target, thisArg, args) {
			return "tset"
		}
	});

	proxy.call();
	`

	testScript1(SCRIPT, asciiString("tset"), t)
}

func TestProxy_target_new(t *testing.T) {
	const SCRIPT = `
	var obj = function(word) {
		this.foo = function() {
			return word;
		}
	}
	
	var proxy = new Proxy(obj, {});

	var instance = new proxy("test");
	instance.foo();
	`

	testScript1(SCRIPT, asciiString("test"), t)
}

func TestProxy_proxy_new(t *testing.T) {
	const SCRIPT = `
	var obj = function(word) {
		this.foo = function() {
			return word;
		}
	}
	
	var proxy = new Proxy(obj, {
		construct: function(target, args, newTarget) {
			var word = args[0]; 
			return {
				foo: function() {
					return "caught-" + word
				}
			}
		}
	});

	var instance = new proxy("test");
	instance.foo();
	`

	testScript1(SCRIPT, asciiString("caught-test"), t)
}

func TestProxy_Object_native_proxy_ownKeys(t *testing.T) {
	headers := map[string][]string{
		"k0": {},
	}
	vm := New()
	proxy := vm.NewProxy(vm.NewObject(), &ProxyTrapConfig{
		OwnKeys: func(target *Object) (object *Object) {
			keys := make([]interface{}, 0, len(headers))
			for k := range headers {
				keys = append(keys, k)
			}
			return vm.ToValue(keys).ToObject(vm)
		},
		GetOwnPropertyDescriptor: func(target *Object, prop string) PropertyDescriptor {
			v, exists := headers[prop]
			if exists {
				return PropertyDescriptor{
					Value:        vm.ToValue(v),
					Enumerable:   FLAG_TRUE,
					Configurable: FLAG_TRUE,
				}
			}
			return PropertyDescriptor{}
		},
	})
	vm.Set("headers", proxy)
	v, err := vm.RunString(`
		var keys = Object.keys(headers);
		keys.length === 1 && keys[0] === "k0";
		`)
	if err != nil {
		t.Fatal(err)
	}
	if v != valueTrue {
		t.Fatal("not true", v)
	}
}

func TestProxy_proxy_forIn(t *testing.T) {
	const SCRIPT = `
	var proto = {
		a: 2,
		protoProp: 1
	}
	Object.defineProperty(proto, "protoNonEnum", {
		value: 2,
		writable: true,
		configurable: true
	});
	var target = Object.create(proto);
	var proxy = new Proxy(target, {
		ownKeys: function() {
			return ["a", "b"];
		},
		getOwnPropertyDescriptor: function(target, p) {
			switch (p) {
			case "a":
			case "b":
				return {
					value: 42,
					enumerable: true,
					configurable: true
				}
			}
		},
	});

	var forInResult = [];
	for (var key in proxy) {
		if (forInResult.indexOf(key) !== -1) {
			throw new Error("Duplicate property "+key);
		}
		forInResult.push(key);
	}
	forInResult.length === 3 && forInResult[0] === "a" && forInResult[1] === "b" && forInResult[2] === "protoProp";
	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestProxyExport(t *testing.T) {
	vm := New()
	v, err := vm.RunString(`
	new Proxy({}, {});
	`)
	if err != nil {
		t.Fatal(err)
	}
	v1 := v.Export()
	if _, ok := v1.(Proxy); !ok {
		t.Fatalf("Export returned unexpected type: %T", v1)
	}
}
