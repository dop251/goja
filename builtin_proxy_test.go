package goja

import (
	"testing"
	"github.com/stretchr/testify/assert"
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
	}, false)
	runtime.Set("proxy", proxy)

	_, err := runtime.RunString(TESTLIB + SCRIPT)
	if err != nil {
		panic(err)
	}
}

/*func TestProxy_Object_target_setPrototypeOf(t *testing.T) {
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
	var p = Object.getPrototypeOf(proxy);
	assert.sameValue(proto2, p);
	`

	testScript1(TESTLIB+SCRIPT, _undefined, t)
}*/

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
			return true;
		}
	});
	Object.isExtensible(proxy);
	`

	testScript1(SCRIPT, valueTrue, t)
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
			return true
		},
	}, false)
	runtime.Set("proxy", proxy)

	val, err := runtime.RunString(SCRIPT)
	if err != nil {
		panic(err)
	}
	assert.Equal(t, true, val.ToBoolean())
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
			return true;
		}
	});
	Object.preventExtensions(proxy);
	proxy.canEvolve
	`

	testScript1(SCRIPT, valueFalse, t)
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

	var desc2 = Object.getOwnPropertyDescriptor(proxy, "foo");
	desc2.value
	`

	testScript1(SCRIPT, valueInt(24), t)
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
	
	with(proxy) {
		(secret);
	}
	`

	testScript1(SCRIPT, _undefined, t)
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
		value: "test123"
	});
	proxy.foo;
	`

	testScript1(SCRIPT, asciiString("321tset"), t)
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
	if (keys.length != 2) {
		throw new Error("assertion error");
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
