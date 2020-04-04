package goja

import (
	"hash/maphash"
	"testing"
)

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

func BenchmarkMapDelete(b *testing.B) {
	var key1 Value = asciiString("a")
	var key2 Value = asciiString("b")
	one := intToValue(1)
	two := intToValue(2)
	for i := 0; i < b.N; i++ {
		m := newOrderedMap(&maphash.Hash{})
		m.set(key1, one)
		m.set(key2, two)
		if !m.remove(key1) {
			b.Fatal("remove() returned false")
		}
	}
}

func BenchmarkMapDeleteJS(b *testing.B) {
	prg, err := Compile("test.js", `
	var m = new Map([['a',1], ['b', 2]]);
	
	var result = m.delete('a');

	if (!result || m.size !== 1) {
		throw new Error("Fail!");
	}
	`,
		false)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vm := New()
		_, err := vm.RunProgram(prg)
		if err != nil {
			b.Fatal(err)
		}
	}
}
