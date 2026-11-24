package goja

import "testing"

func TestIteratorDisposeWithPrimitiveReceiver(t *testing.T) {
	testScriptWithTestLib(`
	const dispose = Iterator.prototype[Symbol.dispose];
	assert.sameValue(dispose.call(42), undefined);
	Object.defineProperty(Number.prototype, "return", { value: function() {
		"use strict";
		assert.sameValue(this, 42);
	} });
	assert.sameValue(dispose.call(42), undefined);
	`, _undefined, t)
}

func TestDisposableStackMoveDuringUse(t *testing.T) {
	New().testAsyncFuncWithTestLib(`
	for (const [Ctor, symbol] of [
		[DisposableStack, Symbol.dispose],
		[AsyncDisposableStack, Symbol.asyncDispose],
	]) {
		const stack = new Ctor();
		const calls = [];
		let moved;
		stack.defer(() => calls.push("previous"));
		stack.use({ get [symbol]() {
			moved = stack.move();
			return () => calls.push("item");
		} });

		await stack[symbol]();
		assert.sameValue(calls.length, 0);
		await moved[symbol]();
		assert.sameValue(calls.join(), "item,previous");
	}
	`, _undefined, t)
}

func TestAsyncDisposableStackDisposalOrder(t *testing.T) {
	New().testAsyncFuncWithTestLib(`
	const stack = new AsyncDisposableStack();
	const log = [];
	const first = {};
	const second = {};
	const third = {};
	stack.defer(() => { log.push("defer"); throw third; });
	stack.use({ [Symbol.asyncDispose]: null, [Symbol.dispose]() {
		log.push("sync throw");
		Promise.resolve().then(() => log.push("tick"));
		throw second;
	} });
	stack.use({ [Symbol.dispose]() {
		log.push("sync success");
		return { get then() { log.push("unexpected then"); } };
	} });
	let reject;
	const pending = new Promise((_, r) => { reject = r; });
	const resource = { [Symbol.asyncDispose]() {
		log.push("async");
		return pending;
	} };
	stack.use(resource);
	const result = stack[Symbol.asyncDispose]();
	assert.sameValue(stack.disposed, true);
	assert.sameValue(log.join(), "async");
	reject(first);
	const error = await result.catch(e => e);
	assert.sameValue(error instanceof SuppressedError, true);
	assert.sameValue(error.error, third);
	assert.sameValue(error.suppressed.error, second);
	assert.sameValue(error.suppressed.suppressed, first);
	assert.sameValue(log.join(), "async,sync success,sync throw,tick,defer");

	for (const withNullish of [false, true]) {
		const s = new AsyncDisposableStack();
		if (withNullish) {
			assert.sameValue(s.use(null), null);
			assert.sameValue(s.use(), undefined);
		}
		const order = [];
		const p = s.disposeAsync().then(() => order.push("disposed"));
		Promise.resolve().then(() => order.push("tick"));
		await p;
		assert.sameValue(order.join(), withNullish ? "tick,disposed" : "disposed,tick");
	}
	`, _undefined, t)
}
