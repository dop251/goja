package goja

import "testing"

func TestIteratorConstructor(t *testing.T) {
	const SCRIPT = `
	assert.throws(TypeError, () => new Iterator());
	assert.throws(TypeError, () => Iterator());
	assert.sameValue(Iterator.prototype, Object.getPrototypeOf(Object.getPrototypeOf([][Symbol.iterator]())));

	class Sub extends Iterator {}
	var s = new Sub();
	assert.sameValue(s instanceof Sub, true);
	assert.sameValue(s instanceof Iterator, true);
	assert.sameValue(Object.getPrototypeOf(Sub.prototype), Iterator.prototype);
	assert.sameValue(s.map(x => x * 2).constructor, Iterator);
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestIteratorPrototypeAccessors(t *testing.T) {
	const SCRIPT = `
	assert.sameValue(Iterator.prototype.constructor, Iterator);
	assert.sameValue(Iterator.prototype[Symbol.toStringTag], "Iterator");
	assert.sameValue(Object.prototype.toString.call([].values().map(x => x)), "[object Iterator Helper]");

	var tag = Object.getOwnPropertyDescriptor(Iterator.prototype, Symbol.toStringTag);
	assert.sameValue(typeof tag.get, "function");
	assert.sameValue(typeof tag.set, "function");

	assert.throws(TypeError, () => { Iterator.prototype[Symbol.toStringTag] = "x"; });
	assert.throws(TypeError, () => tag.set.call(1, "x"));

	var o = Object.create(Iterator.prototype);
	o[Symbol.toStringTag] = "Custom";
	assert.sameValue(Object.getOwnPropertyDescriptor(o, Symbol.toStringTag).value, "Custom");
	assert.sameValue(Iterator.prototype[Symbol.toStringTag], "Iterator");
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestIteratorFrom(t *testing.T) {
	const SCRIPT = `
	var native = [1, 2].values();
	assert.sameValue(Iterator.from(native), native);
	assert.sameValue(Iterator.from([1, 2, 3]).toArray().join(), "1,2,3");
	assert.sameValue(Iterator.from("ab").toArray().join(), "a,b");

	var i = 0;
	var like = {
		next() {
			return i < 3 ? {value: i++, done: false} : {value: undefined, done: true};
		}
	};
	var wrapped = Iterator.from(like);
	assert.sameValue(wrapped instanceof Iterator, true);
	assert.sameValue(wrapped.map(x => x * 2).toArray().join(), "0,2,4");

	var custom = {
		[Symbol.iterator]() {
			return {next() { return {done: true}; }};
		}
	};
	assert.sameValue(Iterator.from(custom).toArray().length, 0);

	var returns = 0;
	var closable = Iterator.from({
		next() { return {done: false, value: 1}; },
		return() { returns++; return {}; }
	});
	closable.return();
	assert.sameValue(returns, 1);
	var result = Iterator.from({}).return();
	assert.sameValue(result.done, true);
	assert.sameValue(result.value, undefined);

	assert.throws(TypeError, () => Iterator.from(42));
	assert.throws(TypeError, () => Iterator.from(null));
	assert.throws(TypeError, () => Iterator.from({[Symbol.iterator]: 0}));
	assert.throws(TypeError, () => Iterator.from({}).next());
	assert.throws(TypeError, () => new Iterator.from([]));
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestIteratorMapFilter(t *testing.T) {
	const SCRIPT = `
	assert.sameValue([1, 2, 3].values().map((x, i) => x * 10 + i).toArray().join(), "10,21,32");
	assert.sameValue([1, 2, 3, 4].values().filter((x, i) => x % 2 === 0 || i === 0).toArray().join(), "1,2,4");
	assert.sameValue([1, 2, 3, 4, 5, 6].values().filter(x => x % 2 === 0).map(x => x * 10).toArray().join(), "20,40,60");

	var calls = 0;
	var it = [1, 2].values().map(x => { calls++; return x; });
	assert.sameValue(calls, 0);
	it.next();
	assert.sameValue(calls, 1);

	var self;
	[1].values().map(function() { "use strict"; self = this; }).next();
	assert.sameValue(self, undefined);

	assert.throws(TypeError, () => [1].values().map(1));
	assert.throws(TypeError, () => [1].values().filter({}));
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestIteratorTakeDrop(t *testing.T) {
	const SCRIPT = `
	function* naturals() {
		var n = 1;
		while (true) {
			yield n++;
		}
	}
	assert.sameValue(naturals().take(3).toArray().join(), "1,2,3");
	assert.sameValue(naturals().drop(2).take(2).toArray().join(), "3,4");
	assert.sameValue([1, 2].values().take(Infinity).toArray().join(), "1,2");
	assert.sameValue([1, 2].values().take(5).toArray().join(), "1,2");
	assert.sameValue([1, 2].values().drop(Infinity).toArray().length, 0);
	assert.sameValue([1, 2].values().drop(0).toArray().join(), "1,2");
	assert.sameValue([1, 2, 3].values().take(-0.5).toArray().length, 0);
	assert.sameValue([1, 2, 3].values().take("2").toArray().join(), "1,2");

	assert.throws(RangeError, () => [1].values().take(NaN));
	assert.throws(RangeError, () => [1].values().take(-1));
	assert.throws(RangeError, () => [1].values().drop(undefined));
	assert.throws(RangeError, () => [1].values().drop(-1));
	assert.throws(TypeError, () => [1].values().take(Symbol()));

	var log = [];
	function* g() {
		try {
			yield 1;
			yield 2;
			yield 3;
		} finally {
			log.push("closed");
		}
	}
	var it = g().take(2);
	it.next();
	it.next();
	assert.sameValue(log.length, 0);
	assert.sameValue(it.next().done, true);
	assert.sameValue(log.join(), "closed");
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestIteratorFlatMap(t *testing.T) {
	const SCRIPT = `
	assert.sameValue([1, 2, 3].values().flatMap(x => [x, x * 10]).toArray().join(), "1,10,2,20,3,30");
	assert.sameValue([1, 2, 3].values().flatMap(x => x === 2 ? [] : [x, x]).toArray().join(), "1,1,3,3");
	assert.sameValue([[1, [2]], [3]].values().flatMap(x => x).toArray().length, 3);

	function* inner(n) {
		yield n;
		yield n + 1;
	}
	assert.sameValue([10, 20].values().flatMap(x => inner(x)).toArray().join(), "10,11,20,21");

	var i = 0;
	var iterator = {
		next() {
			return i < 2 ? {value: i++, done: false} : {done: true};
		}
	};
	assert.sameValue([1].values().flatMap(x => iterator).toArray().join(), "0,1");
	assert.sameValue([1].values().flatMap(x => new String("ab")).toArray().join(), "a,b");

	assert.throws(TypeError, () => [1].values().flatMap(x => 5).toArray());
	assert.throws(TypeError, () => [1].values().flatMap(x => "str").toArray());
	assert.throws(TypeError, () => [1].values().flatMap(null));

	var closed = [];
	var outer = {
		__proto__: Iterator.prototype,
		next() { return {done: false, value: 1}; },
		return() { closed.push("outer"); return {}; }
	};
	var helper = outer.flatMap(x => ({
		next() { return {done: false, value: x}; },
		return() { closed.push("inner"); return {}; }
	}));
	helper.next();
	helper.return();
	assert.sameValue(closed.join(), "inner,outer");
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestIteratorHelperClosing(t *testing.T) {
	const SCRIPT = `
	var returns = 0;
	class Counting extends Iterator {
		next() { return {done: false, value: 1}; }
		return() { returns++; return {}; }
	}

	var helper = new Counting().map(x => x);
	helper.return();
	assert.sameValue(returns, 1);
	helper.return();
	assert.sameValue(returns, 1);
	assert.sameValue(helper.next().done, true);

	returns = 0;
	helper = new Counting().filter(x => true);
	helper.next();
	helper.return();
	assert.sameValue(returns, 1);

	returns = 0;
	var chained = new Counting().map(x => x).filter(x => true).take(5);
	chained.next();
	chained.return();
	assert.sameValue(returns, 1);

	returns = 0;
	assert.throws(TypeError, () => new Counting().map(1));
	assert.sameValue(returns, 1);
	assert.throws(RangeError, () => new Counting().take(-1));
	assert.sameValue(returns, 2);

	returns = 0;
	var thrown = new Counting().map(() => { throw new RangeError("boom"); });
	assert.throws(RangeError, () => thrown.next());
	assert.sameValue(returns, 1);
	assert.sameValue(thrown.next().done, true);

	var it = [1].values().map(x => it.next());
	assert.throws(TypeError, () => it.next());

	var proto = Object.getPrototypeOf([].values().map(x => x));
	assert.throws(TypeError, () => proto.next.call({}));
	assert.throws(TypeError, () => proto.return.call([].values()));
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestIteratorEagerMethods(t *testing.T) {
	const SCRIPT = `
	assert.sameValue([1, 2, 3, 4].values().reduce((a, x) => a + x), 10);
	assert.sameValue([1, 2, 3, 4].values().reduce((a, x) => a + x, 5), 15);
	assert.sameValue([5, 5, 5].values().reduce((a, x, i) => a + i), 8);
	assert.sameValue([5, 5, 5].values().reduce((a, x, i) => a + i, 0), 3);
	assert.sameValue([].values().reduce((a, x) => a + x, 7), 7);
	assert.throws(TypeError, () => [].values().reduce((a, x) => a + x));

	assert.sameValue([1, 2, 3].values().toArray().join(), "1,2,3");
	assert.sameValue([].values().toArray().length, 0);

	var seen = [];
	assert.sameValue([1, 2, 3].values().forEach((x, i) => seen.push(x + ":" + i)), undefined);
	assert.sameValue(seen.join(), "1:0,2:1,3:2");

	var log = [];
	function* g() {
		try {
			yield 1;
			yield 2;
			yield 3;
		} finally {
			log.push("closed");
		}
	}
	assert.sameValue(g().some(x => x === 2), true);
	assert.sameValue(log.length, 1);
	assert.sameValue(g().every(x => x < 2), false);
	assert.sameValue(log.length, 2);
	assert.sameValue(g().find(x => x > 1), 2);
	assert.sameValue(log.length, 3);

	assert.sameValue(g().some(x => x > 5), false);
	assert.sameValue(g().every(x => x > 0), true);
	assert.sameValue(g().find(x => x > 5), undefined);
	assert.sameValue([].values().some(x => true), false);
	assert.sameValue([].values().every(x => false), true);
	assert.sameValue(log.length, 3 + 3);

	log = [];
	assert.throws(RangeError, () => g().forEach(x => { throw new RangeError("boom"); }));
	assert.sameValue(log.length, 1);

	["some", "every", "find", "forEach", "reduce"].forEach(name => {
		assert.throws(TypeError, () => [1].values()[name](42), name);
	});
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestIteratorHelpersOnBuiltinIterators(t *testing.T) {
	const SCRIPT = `
	assert.sameValue(new Map([["a", 1], ["b", 2]]).values().map(x => x * 10).toArray().join(), "10,20");
	assert.sameValue(new Set([1, 2, 3]).values().filter(x => x > 1).toArray().join(), "2,3");
	assert.sameValue("héllo"[Symbol.iterator]().map(c => c.toUpperCase()).toArray().join(""), "HÉLLO");
	function* range(n) {
		for (var i = 0; i < n; i++) {
			yield i;
		}
	}
	assert.sameValue(range(10).filter(x => x % 3 === 0).toArray().join(), "0,3,6,9");

	assert.sameValue([...[1, 2, 3].values().map(x => x + 1)].join(), "2,3,4");
	assert.sameValue(Array.from([1, 2, 3].values().filter(x => x > 1)).join(), "2,3");

	var sum = 0;
	for (var x of [1, 2, 3].values().map(x => x * 2)) {
		sum += x;
	}
	assert.sameValue(sum, 12);

	var closed = false;
	function* g() {
		try {
			yield 1;
			yield 2;
		} finally {
			closed = true;
		}
	}
	for (var y of g().map(x => x)) {
		break;
	}
	assert.sameValue(closed, true);
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestIteratorHelpersReceiver(t *testing.T) {
	const SCRIPT = `
	var gets = 0;
	var obj = {
		__proto__: Iterator.prototype,
		get next() {
			gets++;
			var i = 0;
			return () => ({done: i >= 3, value: i++});
		}
	};
	assert.sameValue(obj.map(x => x).toArray().join(), "0,1,2");
	assert.sameValue(gets, 1);

	assert.throws(TypeError, () => Iterator.prototype.map.call(1, x => x));
	assert.throws(TypeError, () => Iterator.prototype.toArray.call(undefined));
	var helper = Iterator.prototype.map.call({next: 0}, x => x);
	assert.throws(TypeError, () => helper.next());
	assert.throws(TypeError, () => Iterator.prototype.toArray.call({next: 0}));
	assert.throws(TypeError, () => Iterator.prototype.toArray.call({next() { return 1; }}));
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}
