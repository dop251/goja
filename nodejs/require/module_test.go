package require

import (
	"errors"
	"testing"

	js "github.com/dop251/goja"
)

func TestRequireNativeModule(t *testing.T) {
	const SCRIPT = `
	var m = require("test/m");
	m.test();
	`

	vm := js.New()

	registry := new(Registry)
	registry.Enable(vm)

	RegisterNativeModule("test/m", func(runtime *js.Runtime, module *js.Object) {
		o := module.Get("exports").(*js.Object)
		o.Set("test", func(call js.FunctionCall) js.Value {
			return runtime.ToValue("passed")
		})
	})

	v, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	if !v.StrictEquals(vm.ToValue("passed")) {
		t.Fatalf("Unexpected result: %v", v)
	}
}

func TestRequire(t *testing.T) {
	const SCRIPT = `
	var m = require("./testdata/m.js");
	m.test();
	`

	vm := js.New()

	registry := new(Registry)
	registry.Enable(vm)

	v, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	if !v.StrictEquals(vm.ToValue("passed")) {
		t.Fatalf("Unexpected result: %v", v)
	}
}

func TestSourceLoader(t *testing.T) {
	const SCRIPT = `
	var m = require("m.js");
	m.test();
	`

	const MODULE = `
	function test() {
		return "passed1";
	}

	exports.test = test;
	`

	vm := js.New()

	registry := NewRegistryWithLoader(func(name string) ([]byte, error) {
		if name == "m.js" {
			return []byte(MODULE), nil
		}
		return nil, errors.New("Module does not exist")
	})
	registry.Enable(vm)

	v, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	if !v.StrictEquals(vm.ToValue("passed1")) {
		t.Fatalf("Unexpected result: %v", v)
	}
}

func TestStrictModule(t *testing.T) {
	const SCRIPT = `
	var m = require("m.js");
	m.test();
	`

	const MODULE = `
	"use strict";

	function test() {
		var a = "passed1";
		eval("var a = 'not passed'");
		return a;
	}

	exports.test = test;
	`

	vm := js.New()

	registry := NewRegistryWithLoader(func(name string) ([]byte, error) {
		if name == "m.js" {
			return []byte(MODULE), nil
		}
		return nil, errors.New("Module does not exist")
	})
	registry.Enable(vm)

	v, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	if !v.StrictEquals(vm.ToValue("passed1")) {
		t.Fatalf("Unexpected result: %v", v)
	}
}
