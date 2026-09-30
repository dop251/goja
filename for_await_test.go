package goja

import (
	"testing"
)

const forAwaitTestLib = `
function iterableOf(iter) {
	return {[Symbol.asyncIterator]() { return iter; }};
}

// An async iterator yielding 0..n-1 which records calls to next() and return().
function countingIter(log, n) {
	var i = 0;
	return {
		next() {
			log.push("next");
			return Promise.resolve(i < n ? {value: i++, done: false} : {value: undefined, done: true});
		},
		return() {
			log.push("return:" + arguments.length);
			return Promise.resolve({});
		}
	};
}

// A sync iterable yielding the given values which records calls to next() and return().
function syncIterable(log, values) {
	var i = 0;
	var iter = {
		next() {
			log.push("next");
			return i < values.length ? {value: values[i++], done: false} : {value: undefined, done: true};
		},
		return() {
			log.push("return:" + arguments.length);
			return {};
		}
	};
	return {[Symbol.iterator]() { return iter; }, iter: iter};
}

async function caught(f) {
	try {
		await f();
	} catch (e) {
		return e;
	}
	throw new Test262Error("Expected an exception");
}
`

func testForAwait(src string, t *testing.T) {
	t.Helper()
	r := New()
	if _, err := r.RunString(forAwaitTestLib); err != nil {
		t.Fatal(err)
	}
	r.testAsyncFuncWithTestLib(src, _undefined, t)
}

// runForAwaitPending runs setup, which is expected to leave the global 'state' "pending",
// then runs resume and returns the runtime for further checks.
func runForAwaitPending(setup, resume string, t *testing.T) *Runtime {
	t.Helper()
	r := New()
	if _, err := r.RunProgram(testLib()); err != nil {
		t.Fatal(err)
	}
	if _, err := r.RunString(forAwaitTestLib); err != nil {
		t.Fatal(err)
	}
	if _, err := r.RunString(setup); err != nil {
		t.Fatal(err)
	}
	if s := r.Get("state").String(); s != "pending" {
		t.Fatalf("state after setup: %s", s)
	}
	if _, err := r.RunString(resume); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestForAwaitOfBasic(t *testing.T) {
	testForAwait(`
	var log = [];
	var res = [];
	for await (var v of iterableOf(countingIter(log, 3))) res.push(v);
	assert.sameValue(res.join(), "0,1,2");
	assert.sameValue(log.join(), "next,next,next,next");

	var fns = [];
	for await (let v of iterableOf(countingIter([], 3))) {
		await null;
		fns.push(() => v);
	}
	assert.sameValue(fns.map(f => f()).join(), "0,1,2");

	var e = await caught(async () => { for await (let x of x) {} });
	assert(e instanceof ReferenceError, "TDZ");

	var AsyncFunction = (async function() {}).constructor;
	var af = new AsyncFunction("it", "var r = []; for await (const v of it) r.push(v); return r.join();");
	assert.sameValue(await af(iterableOf(countingIter([], 2))), "0,1", "AsyncFunction");

	// eval code is not async
	e = await caught(async () => { eval("for await (const v of []) ;"); });
	assert(e instanceof SyntaxError, "eval");
	`, t)
}

func TestForAwaitOfHostIterator(t *testing.T) {
	r := New()
	var resolvers []func(interface{}) error
	r.Set("hostNext", func(call FunctionCall) Value {
		p, resolve, _ := r.NewPromise()
		resolvers = append(resolvers, resolve)
		return r.ToValue(p)
	})
	_, err := r.RunString(`
	var state = "pending", result;
	var stream = {
		[Symbol.asyncIterator]() {
			return {next: hostNext};
		}
	};
	(async function() {
		var chunks = [];
		for await (const chunk of stream) chunks.push(chunk);
		return chunks.join();
	})().then(v => { state = "fulfilled"; result = v; }, e => { state = "rejected"; result = e; });
	`)
	if err != nil {
		t.Fatal(err)
	}
	for i, chunk := range []string{"a", "b", "c"} {
		if len(resolvers) != i+1 {
			t.Fatalf("next() called %d times, expected %d", len(resolvers), i+1)
		}
		if err := resolvers[i](map[string]interface{}{"value": chunk, "done": false}); err != nil {
			t.Fatal(err)
		}
		if _, err := r.RunString(""); err != nil {
			t.Fatal(err)
		}
		if s := r.Get("state").String(); s != "pending" {
			t.Fatalf("state: %s", s)
		}
	}
	if err := resolvers[3](map[string]interface{}{"done": true}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.RunString(""); err != nil {
		t.Fatal(err)
	}
	if s := r.Get("state").String(); s != "fulfilled" {
		t.Fatalf("state: %s (%v)", s, r.Get("result"))
	}
	if res := r.Get("result").String(); res != "a,b,c" {
		t.Fatalf("result: %s", res)
	}
}

func TestForAwaitOfIteratorLookup(t *testing.T) {
	testForAwait(`
	var calls = 0, gets = 0, nextGets = 0, args = [], thisValue;
	var i = 0;
	var iter = {
		get next() {
			nextGets++;
			return function() {
				args.push(arguments.length);
				return Promise.resolve({value: i, done: i++ >= 2});
			};
		}
	};
	var iterable = {
		get [Symbol.asyncIterator]() {
			gets++;
			return function() {
				calls++;
				thisValue = this;
				return iter;
			};
		}
	};
	var res = [];
	for await (const v of iterable) res.push(v);
	assert.sameValue(res.join(), "0,1");
	assert.sameValue(gets, 1, "getter");
	assert.sameValue(calls, 1, "calls");
	assert.sameValue(thisValue, iterable, "this");
	assert.sameValue(nextGets, 1, "next getter");
	assert.sameValue(args.join(), "0,0,0", "next() arguments");

	// an undefined or null @@asyncIterator falls back to @@iterator
	for (const m of [undefined, null]) {
		var log = [];
		iterable = syncIterable(log, [1]);
		iterable[Symbol.asyncIterator] = m;
		res = [];
		for await (const v of iterable) res.push(v);
		assert.sameValue(res.join(), "1", String(m));
		assert.sameValue(log.join(), "next,next", String(m));
	}

	var iteratorGets = 0;
	var bad = {
		"non-callable @@asyncIterator": {
			[Symbol.asyncIterator]: {},
			get [Symbol.iterator]() {
				iteratorGets++;
				return function() { return [][Symbol.iterator](); };
			}
		},
		"non-object iterator": {[Symbol.asyncIterator]() { return 1; }},
		"non-callable next": iterableOf({next: 1}),
		"not iterable": {}
	};
	for (const name of Object.keys(bad)) {
		var e = await caught(async () => { for await (const v of bad[name]) {} });
		assert(e instanceof TypeError, name);
	}
	assert.sameValue(iteratorGets, 0, "@@iterator must not be used");
	`, t)
}

func TestForAwaitOfNextFailure(t *testing.T) {
	testForAwait(`
	var err = new Test262Error("next failure");
	var cases = {
		"next() throws": function() { throw err; },
		"next() rejects": function() { return Promise.reject(err); },
		"done getter throws": function() { return Promise.resolve({get done() { throw err; }}); },
		"value getter throws": function() { return Promise.resolve({done: false, get value() { throw err; }}); },
		"primitive result": function() { return Promise.resolve(1); },
		"undefined result": function() { return Promise.resolve(undefined); }
	};
	for (const name of Object.keys(cases)) {
		var returns = 0, iterations = 0;
		var e = await caught(async () => {
			for await (const v of iterableOf({next: cases[name], return() { returns++; return {}; }})) {
				iterations++;
			}
		});
		if (name.endsWith("result")) {
			assert(e instanceof TypeError, name);
		} else {
			assert.sameValue(e, err, name);
		}
		assert.sameValue(iterations, 0, "iterations: " + name);
		assert.sameValue(returns, 0, "return() calls: " + name);
	}

	// failure on a later step, after the body has run
	var log = [];
	var iter = countingIter(log, 5);
	var next = iter.next;
	iter.next = function() {
		return log.length > 2 ? Promise.reject(err) : next.call(this);
	};
	var res = [];
	e = await caught(async () => { for await (const v of iterableOf(iter)) res.push(v); });
	assert.sameValue(e, err);
	assert.sameValue(res.join(), "0,1,2");
	assert.sameValue(log.join(), "next,next,next");
	`, t)
}

func TestForAwaitOfBreakCloses(t *testing.T) {
	testForAwait(`
	var log = [];
	for await (const v of iterableOf(countingIter(log, 3))) {
		if (v === 1) break;
	}
	log.push("after");
	assert.sameValue(log.join(), "next,next,return:0,after");

	// the result of return() is awaited, a missing return() is not
	var order = [];
	for (const ret of [function() { return {}; }, undefined, null]) {
		var iter = countingIter([], 3);
		iter.return = ret;
		for await (const v of iterableOf(iter)) {
			Promise.resolve().then(() => order.push("tick"));
			break;
		}
		order.push("after");
		await null;
	}
	assert.sameValue(order.join(), "tick,after,after,tick,after,tick");

	var err = new Test262Error("close");
	var cases = {
		"return() rejects": function() { return Promise.reject(err); },
		"return() throws": function() { throw err; },
		"primitive result": function() { return Promise.resolve(1); },
		"undefined result": function() { return Promise.resolve(undefined); }
	};
	for (const name of Object.keys(cases)) {
		iter = countingIter([], 3);
		iter.return = cases[name];
		var e = await caught(async () => { for await (const v of iterableOf(iter)) break; });
		if (name.endsWith("result")) {
			assert(e instanceof TypeError, name);
		} else {
			assert.sameValue(e, err, name);
		}
	}
	`, t)
}

func TestForAwaitOfBreakWaitsForClose(t *testing.T) {
	r := runForAwaitPending(`
	var state = "pending", log = [], resolveClose;
	var iter = countingIter(log, 3);
	iter.return = function() {
		log.push("return:" + arguments.length);
		return new Promise(resolve => { resolveClose = resolve; });
	};
	(async function() {
		for await (const v of iterableOf(iter)) {
			if (v === 1) break;
		}
		log.push("after");
	})().then(() => { state = "fulfilled"; }, e => { state = "rejected"; });
	`, `
	assert.sameValue(log.join(), "next,next,return:0");
	resolveClose({});
	`, t)
	if s := r.Get("state").String(); s != "fulfilled" {
		t.Fatalf("state: %s", s)
	}
	if l := r.Get("log").String(); l != "next,next,return:0,after" {
		t.Fatalf("log: %s", l)
	}
}

func TestForAwaitOfReturnCloses(t *testing.T) {
	testForAwait(`
	var log = [];
	async function f() {
		for await (const v of iterableOf(countingIter(log, 3))) {
			if (v === 1) return "result:" + v;
		}
	}
	assert.sameValue(await f(), "result:1");
	assert.sameValue(log.join(), "next,next,return:0");

	log = [];
	var iter = countingIter(log, 3);
	iter.return = function() {
		return Promise.resolve().then(() => { log.push("closed"); return {}; });
	};
	async function g() {
		try {
			for await (const v of iterableOf(iter)) {
				try {
					return v;
				} finally {
					log.push("inner finally");
				}
			}
		} finally {
			log.push("outer finally");
		}
	}
	assert.sameValue(await g(), 0);
	assert.sameValue(log.join(), "next,inner finally,closed,outer finally");

	var err = new Test262Error("close");
	iter = countingIter([], 3);
	iter.return = function() { return Promise.reject(err); };
	async function h() {
		for await (const v of iterableOf(iter)) return 1;
	}
	assert.sameValue(await caught(h), err);
	`, t)
}

func TestForAwaitOfLabelledContinueBreak(t *testing.T) {
	testForAwait(`
	var log = [];
	var inner = countingIter(log, 3);
	inner.return = function() {
		log.push("inner");
		return Promise.resolve().then(() => { log.push("inner closed"); return {}; });
	};
	var outer = countingIter(log, 3);
	outer.return = function() { log.push("outer"); return {}; };
	outerLoop: for await (const x of iterableOf(outer)) {
		for await (const y of iterableOf(inner)) {
			break outerLoop;
		}
	}
	assert.sameValue(log.join(), "next,next,inner,inner closed,outer");

	// continue to the outer loop closes only the inner one
	log = [];
	var res = [];
	outer: for await (const x of iterableOf(countingIter(log, 2))) {
		for await (const y of iterableOf(countingIter(log, 2))) {
			res.push(x + "" + y);
			continue outer;
		}
	}
	assert.sameValue(res.join(), "00,10");
	assert.sameValue(log.join(), "next,next,return:0,next,next,return:0,next");

	// plain continue and continue with the loop's own label do not close
	log = [];
	var n = 0;
	for await (const x of iterableOf(countingIter(log, 3))) {
		n++;
		continue;
	}
	self: for await (const x of iterableOf(countingIter(log, 3))) {
		for (const y of [1]) {
			continue self;
		}
	}
	assert.sameValue(n, 3);
	assert.sameValue(log.join(), "next,next,next,next,next,next,next,next");

	// a labelled block
	log = [];
	block: {
		for await (const x of iterableOf(countingIter(log, 3))) {
			break block;
		}
		log.push("not reached");
	}
	assert.sameValue(log.join(), "next,return:0");

	// a sync iterator inside is closed first
	log = [];
	var syncIter = syncIterable([], [1, 1]);
	syncIter.iter.return = function() { log.push("sync return"); return {}; };
	async function f() {
		for await (const x of iterableOf(countingIter(log, 3))) {
			for (const y of syncIter) {
				break;
			}
			for (const y of syncIter) {
				return;
			}
		}
	}
	await f();
	assert.sameValue(log.join(), "next,sync return,sync return,return:0");
	`, t)
}

func TestForAwaitOfThrowAwaitsClose(t *testing.T) {
	for _, loop := range []string{
		"for await (const v of iterableOf(iter)) { throw err; }",
		"for await (const {p} of iterableOf(iter)) {}",
		"var p; for await ({p} of iterableOf(iter)) {}",
	} {
		t.Run(loop, func(t *testing.T) {
			r := runForAwaitPending(`
			var state = "pending", reason, log = [], resolveClose;
			var err = new Test262Error("original");
			var iter = {
				next() {
					log.push("next");
					return Promise.resolve({value: {get p() { throw err; }}, done: false});
				},
				return() {
					log.push("return:" + arguments.length);
					return new Promise(resolve => { resolveClose = resolve; });
				}
			};
			(async function() {
				`+loop+`
			})().then(() => { state = "fulfilled"; }, e => { state = "rejected"; reason = e; });
			`, `
			assert.sameValue(log.join(), "next,return:0");
			resolveClose(1); // a non-object result is ignored for a throw completion
			`, t)
			if s := r.Get("state").String(); s != "rejected" {
				t.Fatalf("state: %s", s)
			}
			if !r.Get("reason").SameAs(r.Get("err")) {
				t.Fatalf("reason: %v", r.Get("reason"))
			}
		})
	}
}

func TestForAwaitOfThrowPrecedence(t *testing.T) {
	testForAwait(`
	var err = new Test262Error("original");
	var closeErr = new Error("close");
	var cases = {
		"return() throws": function() { throw closeErr; },
		"return() rejects": function() { return Promise.reject(closeErr); },
		"primitive result": function() { return Promise.resolve(1); }
	};
	for (const name of Object.keys(cases)) {
		var iter = {
			next() { return Promise.resolve({value: 1, done: false}); },
			return: cases[name]
		};
		var e = await caught(async () => {
			for await (const v of iterableOf(iter)) throw err;
		});
		assert.sameValue(e, err, name);
	}

	// the exception can be caught inside the body without closing the iterator
	var log = [];
	for await (const v of iterableOf(countingIter(log, 2))) {
		try {
			throw err;
		} catch (e) {
		}
	}
	assert.sameValue(log.join(), "next,next,next");

	// nested: the inner iterator is closed first, and the original error propagates through both
	log = [];
	var outer = countingIter(log, 3);
	outer.return = function() { log.push("outer"); return {}; };
	var inner = countingIter(log, 3);
	inner.return = function() { log.push("inner"); return Promise.reject(closeErr); };
	e = await caught(async () => {
		for await (const x of iterableOf(outer)) {
			for await (const y of iterableOf(inner)) {
				throw err;
			}
		}
	});
	assert.sameValue(e, err);
	assert.sameValue(log.join(), "next,next,inner,outer");
	`, t)
}

func TestForAwaitOfSyncFallbackValues(t *testing.T) {
	testForAwait(`
	var res = [];
	for await (const v of [Promise.resolve(1), 2, {then(resolve) { resolve(3); }}]) res.push(v);
	assert.sameValue(res.join(), "1,2,3");

	// a native async iterator's values are not awaited
	var p = Promise.resolve(1);
	var seen;
	for await (const v of iterableOf({
		done: false,
		next() {
			var r = Promise.resolve({value: p, done: this.done});
			this.done = true;
			return r;
		}
	})) {
		seen = v;
	}
	assert.sameValue(seen, p);

	// the value of the done result is not bound
	res = [];
	var iterable = syncIterable([], []);
	iterable.iter.next = function() { return {value: 1, done: true}; };
	for await (const v of iterable) res.push(v);
	assert.sameValue(res.length, 0);

	// only promises are used as is, not any object with a matching 'constructor'
	var notPromise = {constructor: Promise};
	for await (const v of [notPromise]) seen = v;
	assert.sameValue(seen, notPromise, "value");
	for await (const v of iterableOf({
		done: false,
		next() {
			var r = {value: this.done, done: this.done, constructor: Promise};
			this.done = true;
			return r;
		}
	})) {
		seen = v;
	}
	assert.sameValue(seen, false, "next() result");
	`, t)
}

func TestForAwaitOfSyncFallbackTicks(t *testing.T) {
	testForAwait(`
	function ticks(log, n) {
		var p = Promise.resolve();
		for (let i = 1; i <= n; i++) {
			p = p.then(() => log.push("t" + i));
		}
		return p;
	}

	// break: the sync return() result is wrapped (2 turns), a missing return() is not (1 turn)
	for (const [hasReturn, expected] of [[true, "t1,t2,after,t3"], [false, "t1,after,t2,t3"]]) {
		var log = [];
		var iterable = syncIterable([], [1, 2]);
		if (!hasReturn) {
			delete iterable.iter.return;
		}
		await (async function() {
			for await (const v of iterable) {
				ticks(log, 3);
				break;
			}
			log.push("after");
		})();
		await ticks([], 3);
		assert.sameValue(log.join(), expected, "return: " + hasReturn);
	}

	// sync errors from next() become rejections
	log = [];
	var err = new Test262Error("next");
	var tp = ticks(log, 2);
	iterable = syncIterable([], []);
	iterable.iter.next = function() { throw err; };
	var loop = (async function() {
		try {
			for await (const v of iterable) {}
		} catch (e) {
			assert.sameValue(e, err);
			log.push("caught");
		}
	})();
	await tp;
	await loop;
	assert.sameValue(log.join(), "t1,caught,t2");
	`, t)
}

func TestForAwaitOfSyncFallbackNextFailure(t *testing.T) {
	testForAwait(`
	var err = new Test262Error("next failure");
	var cases = {
		"next() throws": function() { throw err; },
		"done getter throws": function() { return {get done() { throw err; }}; },
		"value getter throws": function() { return {done: false, get value() { throw err; }}; },
		"primitive result": function() { return 1; },
		"undefined result": function() { return undefined; },
		"non-callable next": 1
	};
	for (const name of Object.keys(cases)) {
		var log = [];
		var iterable = syncIterable(log, [1]);
		iterable.iter.next = cases[name];
		var e = await caught(async () => { for await (const v of iterable) {} });
		if (typeof cases[name] !== "function" || name.endsWith("result")) {
			assert(e instanceof TypeError, name);
		} else {
			assert.sameValue(e, err, name);
		}
		assert.sameValue(log.length, 0, "return() calls: " + name);
	}
	`, t)
}

func TestForAwaitOfSyncFallbackCloseOnRejection(t *testing.T) {
	testForAwait(`
	var err = new Test262Error("rejected value");
	var log = [];
	var res = [];
	var e = await caught(async () => {
		for await (const v of syncIterable(log, [1, Promise.reject(err), 3])) res.push(v);
	});
	assert.sameValue(e, err);
	assert.sameValue(res.join(), "1");
	assert.sameValue(log.join(), "next,next,return:0");

	e = await caught(async () => { for await (const v of [Promise.reject(err)]) {} });
	assert.sameValue(e, err, "array");

	// PromiseResolve() throws
	var badPromise = Promise.resolve(1);
	Object.defineProperty(badPromise, "constructor", {get() { throw err; }});
	log = [];
	e = await caught(async () => { for await (const v of syncIterable(log, [badPromise])) {} });
	assert.sameValue(e, err, "constructor getter");
	assert.sameValue(log.join(), "next,return:0", "constructor getter");

	// errors while closing are ignored
	var closeCases = {
		"return() throws": function() { log.push("return"); throw new Error("close"); },
		"return() returns a primitive": function() { log.push("return"); return 1; },
		"return is not callable": 1
	};
	for (const name of Object.keys(closeCases)) {
		log = [];
		var iterable = syncIterable(log, [Promise.reject(err)]);
		iterable.iter.return = closeCases[name];
		e = await caught(async () => { for await (const v of iterable) {} });
		assert.sameValue(e, err, name);
		assert.sameValue(log.join(), typeof closeCases[name] === "function" ? "next,return" : "next", name);
	}

	// a rejected value with done: true does not close the iterator
	log = [];
	iterable = syncIterable(log, []);
	iterable.iter.next = function() {
		log.push("next");
		return {value: Promise.reject(err), done: true};
	};
	e = await caught(async () => { for await (const v of iterable) {} });
	assert.sameValue(e, err, "done: true");
	assert.sameValue(log.join(), "next", "done: true");
	`, t)
}

func TestForAwaitOfSyncFallbackReturn(t *testing.T) {
	testForAwait(`
	var log = [];
	for await (const v of syncIterable(log, [1, 2, 3])) {
		if (v === 2) break;
	}
	assert.sameValue(log.join(), "next,next,return:0");

	log = [];
	async function f() {
		for await (const v of syncIterable(log, [1, 2])) return v;
	}
	assert.sameValue(await f(), 1);
	assert.sameValue(log.join(), "next,return:0");

	var finallyCount = 0;
	function* gen() {
		try {
			yield 1;
			yield 2;
		} finally {
			finallyCount++;
		}
	}
	for await (const v of gen()) break;
	assert.sameValue(finallyCount, 1, "generator return()");

	var err = new Test262Error("close");
	var cases = {
		"return() throws": function() { throw err; },
		"return() value rejects": function() { return {value: Promise.reject(err), done: true}; },
		"return getter throws": undefined,
		"primitive result": function() { return 1; },
		"undefined result": function() { return undefined; },
		"non-callable return": 1
	};
	for (const name of Object.keys(cases)) {
		var iterable = syncIterable([], [1]);
		if (cases[name] === undefined) {
			Object.defineProperty(iterable.iter, "return", {get() { throw err; }});
		} else {
			iterable.iter.return = cases[name];
		}
		var e = await caught(async () => { for await (const v of iterable) break; });
		if (typeof cases[name] === "number" || name.endsWith("result")) {
			assert(e instanceof TypeError, name);
		} else {
			assert.sameValue(e, err, name);
		}
	}

	// a throw completion closes the sync iterator once, and the original error wins
	var bodyErr = new Test262Error("body");
	log = [];
	iterable = syncIterable(log, [1]);
	iterable.iter.return = function() {
		log.push("return:" + arguments.length);
		return {value: Promise.reject(err), done: true};
	};
	e = await caught(async () => { for await (const v of iterable) throw bodyErr; });
	assert.sameValue(e, bodyErr);
	assert.sameValue(log.join(), "next,return:0");
	`, t)
}
