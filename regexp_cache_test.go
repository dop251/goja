package goja

import (
	"testing"
)

func TestRegExpLiteralCacheDisabledByDefault(t *testing.T) {
	vm := New()
	res, err := vm.RunString(`
		var a = /foo/;
		var b = /foo/;
		a === b;
	`)
	if err != nil {
		t.Fatalf("RunString failed: %v", err)
	}
	if res.Export().(bool) {
		t.Fatal("RegExp literals should NOT be identical when cache is disabled")
	}
}

func TestRegExpLiteralCacheEnabled(t *testing.T) {
	vm := New()
	vm.EnableRegExpLiteralCache()
	res, err := vm.RunString(`
		var a = /foo/;
		var b = /foo/;
		a === b;
	`)
	if err != nil {
		t.Fatalf("RunString failed: %v", err)
	}
	if !res.Export().(bool) {
		t.Fatal("RegExp literals should be identical when cache is enabled")
	}
}

func TestRegExpLiteralCacheDifferentPatterns(t *testing.T) {
	vm := New()
	vm.EnableRegExpLiteralCache()
	res, err := vm.RunString(`
		var a = /foo/;
		var b = /bar/;
		a === b;
	`)
	if err != nil {
		t.Fatalf("RunString failed: %v", err)
	}
	if res.Export().(bool) {
		t.Fatal("Different RegExp patterns should NOT share the same cached object")
	}
}

func TestRegExpLiteralCacheDifferentFlags(t *testing.T) {
	vm := New()
	vm.EnableRegExpLiteralCache()
	res, err := vm.RunString(`
		var a = /foo/i;
		var b = /foo/g;
		var c = /foo/i;
		var ab = a === b;
		var ac = a === c;
		[ab, ac];
	`)
	if err != nil {
		t.Fatalf("RunString failed: %v", err)
	}
	arr := res.Export().([]interface{})
	if arr[0].(bool) {
		t.Fatal("Different flags should NOT be identical")
	}
	if !arr[1].(bool) {
		t.Fatal("Same pattern+flags SHOULD be identical when cache is enabled")
	}
}

func TestRegExpLiteralCacheGlobalReset(t *testing.T) {
	vm := New()
	vm.EnableRegExpLiteralCache()
	res, err := vm.RunString(`
		var r = /a/g;
		"aba".replace(r, "X");
		var prev = r.lastIndex;
		"aba".replace(r, "X");
		var after = r.lastIndex;
		prev === 0 && after === 0;
	`)
	if err != nil {
		t.Fatalf("RunString failed: %v", err)
	}
	if !res.Export().(bool) {
		t.Fatal("Cached global RegExp should have lastIndex reset to 0 on each literal evaluation")
	}
}

func TestRegExpLiteralCacheTestMethod(t *testing.T) {
	vm := New()
	vm.EnableRegExpLiteralCache()
	res, err := vm.RunString(`
		var r = /\d+/;
		var t1 = r.test("abc123");
		var t2 = r.test("xyz");
		var t3 = /\d+/.test("456");
		t1 && !t2 && t3;
	`)
	if err != nil {
		t.Fatalf("RunString failed: %v", err)
	}
	if !res.Export().(bool) {
		t.Fatal("Cached RegExp test() should work correctly")
	}
}

func TestRegExpLiteralCacheManyEvaluations(t *testing.T) {
	vm := New()
	vm.EnableRegExpLiteralCache()
	res, err := vm.RunString(`
		var count = 0;
		for (var i = 0; i < 10000; i++) {
			if (/^\d+$/.test(String(i))) count++;
		}
		count === 10000;
	`)
	if err != nil {
		t.Fatalf("RunString failed: %v", err)
	}
	if !res.Export().(bool) {
		t.Fatal("Cached RegExp should handle many evaluations correctly")
	}
}
