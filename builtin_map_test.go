package goja

import "testing"

func TestMapEvilIterator(t *testing.T) {
	const SCRIPT = `
	'use strict';
	var o = {};

	function Iter(value) {
		this.value = value;
		this.idx = 0;
	}

	Iter.prototype.next = function() {
		var idx = this.idx;
		if (idx === 0) {
			this.idx++;
			return this.value;
		}
		return {done: true};
	}

	o[Symbol.iterator] = function() {
		return new Iter({});
	}

	assert.throws(TypeError, function() {
		new Map(o);
	});

	o[Symbol.iterator] = function() {
		return new Iter({value: []});
	}

	function t(prefix) {
		var m = new Map(o);
		assert.sameValue(1, m.size, prefix+": m.size");
		assert.sameValue(true, m.has(undefined), prefix+": m.has(undefined)");
		assert.sameValue(undefined, m.get(undefined), prefix+": m.get(undefined)");
	}

	t("standard adder");

	var count = 0;
	var origSet = Map.prototype.set;

	Map.prototype.set = function() {
		count++;
		origSet.apply(this, arguments);
	}

	t("custom adder");
	assert.sameValue(1, count, "count");

	undefined;
	`
	testScript1(TESTLIB+SCRIPT, _undefined, t)
}
