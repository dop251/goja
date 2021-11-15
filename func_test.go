package goja

import "testing"

func TestFuncProto(t *testing.T) {
	const SCRIPT = `
	"use strict";
	function A() {}
	A.__proto__ = Object;
	A.prototype = {};

	function B() {}
	B.__proto__ = Object.create(null);
	var thrown = false;
	try {
		delete B.prototype;
	} catch (e) {
		thrown = e instanceof TypeError;
	}
	thrown;
	`
	testScript1(SCRIPT, valueTrue, t)
}

func TestFuncPrototypeRedefine(t *testing.T) {
	const SCRIPT = `
	let thrown = false;
	try {
		Object.defineProperty(function() {}, "prototype", {
			set: function(_value) {},
		});
	} catch (e) {
		if (e instanceof TypeError) {
			thrown = true;
		} else {
			throw e;
		}
	}
	thrown;
	`

	testScript1(SCRIPT, valueTrue, t)
}
