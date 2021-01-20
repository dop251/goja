package goja

import (
	"strconv"
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

func TestProxy_native_proxy_getOwnPropertyDescriptorIdx(t *testing.T) {
	vm := New()
	a := vm.NewArray()
	proxy1 := vm.NewProxy(a, &ProxyTrapConfig{
		GetOwnPropertyDescriptor: func(target *Object, prop string) PropertyDescriptor {
			panic(vm.NewTypeError("GetOwnPropertyDescriptor was called"))
		},
		GetOwnPropertyDescriptorIdx: func(target *Object, prop int) PropertyDescriptor {
			if prop >= -1 && prop <= 1 {
				return PropertyDescriptor{
					Value:        vm.ToValue(prop),
					Configurable: FLAG_TRUE,
				}
			}
			return PropertyDescriptor{}
		},
	})

	proxy2 := vm.NewProxy(a, &ProxyTrapConfig{
		GetOwnPropertyDescriptor: func(target *Object, prop string) PropertyDescriptor {
			switch prop {
			case "-1", "0", "1":
				return PropertyDescriptor{
					Value:        vm.ToValue(prop),
					Configurable: FLAG_TRUE,
				}
			}
			return PropertyDescriptor{}
		},
	})

	vm.Set("proxy1", proxy1)
	vm.Set("proxy2", proxy2)
	_, err := vm.RunString(TESTLIBX + `
	var desc;
	for (var i = -1; i <= 1; i++) {
		desc = Object.getOwnPropertyDescriptor(proxy1, i);
		assert(deepEqual(desc, {value: i, writable: false, enumerable: false, configurable: true}), "1. int "+i);

		desc = Object.getOwnPropertyDescriptor(proxy1, ""+i);
		assert(deepEqual(desc, {value: i, writable: false, enumerable: false, configurable: true}), "1. str "+i);

		desc = Object.getOwnPropertyDescriptor(proxy2, i);
		assert(deepEqual(desc, {value: ""+i, writable: false, enumerable: false, configurable: true}), "2. int "+i);

		desc = Object.getOwnPropertyDescriptor(proxy2, ""+i);
		assert(deepEqual(desc, {value: ""+i, writable: false, enumerable: false, configurable: true}), "2. str "+i);
	}
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestProxy_native_proxy_getOwnPropertyDescriptorSym(t *testing.T) {
	vm := New()
	o := vm.NewObject()
	sym := NewSymbol("42")
	vm.Set("sym", sym)
	proxy := vm.NewProxy(o, &ProxyTrapConfig{
		GetOwnPropertyDescriptorSym: func(target *Object, s *Symbol) PropertyDescriptor {
			if target != o {
				panic(vm.NewTypeError("Invalid target"))
			}
			if s == sym {
				return PropertyDescriptor{
					Value:        vm.ToValue("passed"),
					Writable:     FLAG_TRUE,
					Configurable: FLAG_TRUE,
				}
			}
			return PropertyDescriptor{}
		},
	})

	vm.Set("proxy", proxy)
	_, err := vm.RunString(TESTLIBX + `
	var desc = Object.getOwnPropertyDescriptor(proxy, sym);
	assert(deepEqual(desc, {value: "passed", writable: true, enumerable: false, configurable: true}));
	assert.sameValue(Object.getOwnPropertyDescriptor(proxy, Symbol.iterator), undefined);
	`)
	if err != nil {
		t.Fatal(err)
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
		value: "teststr"
	});
	Object.defineProperty(proxy, "0", {
		value: "testidx"
	});
	Object.defineProperty(proxy, Symbol.toStringTag, {
		value: "testsym"
	});
	assert.sameValue(proxy.foo, "teststr-passed-str");
	assert.sameValue(proxy[0], "testidx-passed-idx");
	assert.sameValue(proxy[Symbol.toStringTag], "testsym-passed-sym");
	`

	runtime := New()

	target := runtime.NewObject()

	proxy := runtime.NewProxy(target, &ProxyTrapConfig{
		DefineProperty: func(target *Object, key string, propertyDescriptor PropertyDescriptor) (success bool) {
			target.Set(key, propertyDescriptor.Value.String()+"-passed-str")
			return true
		},
		DefinePropertyIdx: func(target *Object, key int, propertyDescriptor PropertyDescriptor) (success bool) {
			target.Set(strconv.Itoa(key), propertyDescriptor.Value.String()+"-passed-idx")
			return true
		},
		DefinePropertySym: func(target *Object, key *Symbol, propertyDescriptor PropertyDescriptor) (success bool) {
			target.SetSymbol(key, propertyDescriptor.Value.String()+"-passed-sym")
			return true
		},
	})
	runtime.Set("proxy", proxy)

	_, err := runtime.RunString(TESTLIB + SCRIPT)
	if err != nil {
		t.Fatal(err)
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
	propValueStr := vm.ToValue("321tset")
	propValueIdx := vm.ToValue("idx")
	propValueSym := vm.ToValue("sym")
	sym := NewSymbol("test")
	obj := vm.NewObject()
	proxy := vm.NewProxy(obj, &ProxyTrapConfig{
		OwnKeys: func(*Object) *Object {
			return vm.NewArray("0", "foo")
		},
		GetOwnPropertyDescriptor: func(target *Object, prop string) (propertyDescriptor PropertyDescriptor) {
			if prop == "foo" {
				return PropertyDescriptor{
					Value:        propValueStr,
					Enumerable:   FLAG_TRUE,
					Configurable: FLAG_TRUE,
				}
			}
			if prop == "0" {
				panic(vm.NewTypeError("GetOwnPropertyDescriptor(0) was called"))
			}
			return
		},
		GetOwnPropertyDescriptorIdx: func(target *Object, prop int) (propertyDescriptor PropertyDescriptor) {
			if prop == 0 {
				return PropertyDescriptor{
					Value:        propValueIdx,
					Enumerable:   FLAG_TRUE,
					Configurable: FLAG_TRUE,
				}
			}
			return
		},
		Get: func(target *Object, property string, receiver Value) (value Value) {
			if property == "foo" {
				return propValueStr
			}
			if property == "0" {
				panic(vm.NewTypeError("Get(0) was called"))
			}
			return obj.Get(property)
		},
		GetIdx: func(target *Object, property int, receiver Value) (value Value) {
			if property == 0 {
				return propValueIdx
			}
			return obj.Get(strconv.Itoa(property))
		},
		GetSym: func(target *Object, property *Symbol, receiver Value) (value Value) {
			if property == sym {
				return propValueSym
			}
			return obj.GetSymbol(property)
		},
	})
	vm.Set("proxy", proxy)
	res, err := vm.RunString(`JSON.stringify(proxy)`)
	if err != nil {
		t.Fatal(err)
	}
	if !res.SameAs(asciiString(`{"0":"idx","foo":"321tset"}`)) {
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

	if val := vm.ToValue(proxy).(*Object).GetSymbol(sym); val == nil || !val.SameAs(propValueSym) {
		t.Fatalf("Get(symbol): %v", val)
	}

	res, err = vm.RunString(`proxy.toString()`)
	if err != nil {
		t.Fatal(err)
	}
	if !res.SameAs(asciiString(`[object Object]`)) {
		t.Fatalf("res: %v", res)
	}
}

func TestProxy_native_proxy_set(t *testing.T) {
	vm := New()
	propValueStr := vm.ToValue("321tset")
	propValueIdx := vm.ToValue("idx")
	propValueSym := vm.ToValue("sym")
	sym := NewSymbol("test")
	obj := vm.NewObject()
	proxy := vm.NewProxy(obj, &ProxyTrapConfig{
		Set: func(target *Object, property string, value Value, receiver Value) (success bool) {
			if property == "str" {
				obj.Set(property, propValueStr)
				return true
			}
			panic(vm.NewTypeError("Setter for unexpected property: %q", property))
		},
		SetIdx: func(target *Object, property int, value Value, receiver Value) (success bool) {
			if property == 0 {
				obj.Set(strconv.Itoa(property), propValueIdx)
				return true
			}
			panic(vm.NewTypeError("Setter for unexpected idx property: %d", property))
		},
		SetSym: func(target *Object, property *Symbol, value Value, receiver Value) (success bool) {
			if property == sym {
				obj.SetSymbol(property, propValueSym)
				return true
			}
			panic(vm.NewTypeError("Setter for unexpected sym property: %q", property.String()))
		},
	})
	proxyObj := vm.ToValue(proxy).ToObject(vm)
	err := proxyObj.Set("str", "")
	if err != nil {
		t.Fatal(err)
	}
	err = proxyObj.Set("0", "")
	if err != nil {
		t.Fatal(err)
	}
	err = proxyObj.SetSymbol(sym, "")
	if err != nil {
		t.Fatal(err)
	}
	if v := obj.Get("str"); !propValueStr.SameAs(v) {
		t.Fatal(v)
	}
	if v := obj.Get("0"); !propValueIdx.SameAs(v) {
		t.Fatal(v)
	}
	if v := obj.GetSymbol(sym); !propValueSym.SameAs(v) {
		t.Fatal(v)
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

func TestProxy_native_delete(t *testing.T) {
	vm := New()
	sym := NewSymbol("test")
	obj := vm.NewObject()
	var strCalled, idxCalled, symCalled, strNegCalled, idxNegCalled, symNegCalled bool
	proxy := vm.NewProxy(obj, &ProxyTrapConfig{
		DeleteProperty: func(target *Object, property string) (success bool) {
			if property == "str" {
				strCalled = true
				return true
			}
			if property == "strNeg" {
				strNegCalled = true
				return false
			}
			panic(vm.NewTypeError("DeleteProperty for unexpected property: %q", property))
		},
		DeletePropertyIdx: func(target *Object, property int) (success bool) {
			if property == 0 {
				idxCalled = true
				return true
			}
			if property == 1 {
				idxNegCalled = true
				return false
			}
			panic(vm.NewTypeError("DeletePropertyIdx for unexpected idx property: %d", property))
		},
		DeletePropertySym: func(target *Object, property *Symbol) (success bool) {
			if property == sym {
				symCalled = true
				return true
			}
			if property == SymIterator {
				symNegCalled = true
				return false
			}
			panic(vm.NewTypeError("DeletePropertySym for unexpected sym property: %q", property.String()))
		},
	})
	proxyObj := vm.ToValue(proxy).ToObject(vm)
	err := proxyObj.Delete("str")
	if err != nil {
		t.Fatal(err)
	}
	err = proxyObj.Delete("0")
	if err != nil {
		t.Fatal(err)
	}
	err = proxyObj.DeleteSymbol(sym)
	if err != nil {
		t.Fatal(err)
	}
	if !strCalled {
		t.Fatal("str")
	}
	if !idxCalled {
		t.Fatal("idx")
	}
	if !symCalled {
		t.Fatal("sym")
	}
	vm.Set("proxy", proxy)
	_, err = vm.RunString(`
	if (delete proxy.strNeg) {
		throw new Error("strNeg");
	}
	if (delete proxy[1]) {
		throw new Error("idxNeg");
	}
	if (delete proxy[Symbol.iterator]) {
		throw new Error("symNeg");
	}
	`)
	if err != nil {
		t.Fatal(err)
	}
	if !strNegCalled {
		t.Fatal("strNeg")
	}
	if !idxNegCalled {
		t.Fatal("idxNeg")
	}
	if !symNegCalled {
		t.Fatal("symNeg")
	}
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
