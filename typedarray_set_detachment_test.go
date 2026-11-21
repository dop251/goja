package goja

import "testing"

func TestTypedArraySetConvertsAfterGetterDetaches(t *testing.T) {
	for _, ctor := range []string{"Int8Array", "Uint8Array", "Uint8ClampedArray", "Int16Array", "Uint16Array", "Int32Array", "Uint32Array", "Float32Array", "Float64Array", "BigInt64Array", "BigUint64Array"} {
		t.Run(ctor, func(t *testing.T) {
			script := `
			const C = ` + ctor + `;
			const target = new C(1);
			const marker = new Error("element conversion");
			let log = [];
			const source = {length: 1, get 0() {
				log.push("get");
				$DETACHBUFFER(target.buffer);
				return {valueOf() { log.push("convert"); throw marker; }};
			}};
			let caught;
			try { target.set(source); } catch(e) { caught = e; }
			assert.sameValue(caught, marker, "propagate element conversion even after detachment");
			assert.sameValue(log.join(","), "get,convert");
			assert.sameValue(target.buffer.byteLength, 0);
			`
			vm := New()
			vm.Set("$DETACHBUFFER", func(buf *ArrayBuffer) { buf.Detach() })
			vm.testScriptWithTestLib(script, _undefined, t)
		})
	}
}

func TestTypedArraySetContinuesConversionsAfterDetachment(t *testing.T) {
	for _, ctor := range []string{"Uint8Array", "Float64Array", "BigInt64Array", "BigUint64Array"} {
		t.Run(ctor, func(t *testing.T) {
			script := `
			const C = ` + ctor + `;
			const big = C === BigInt64Array || C === BigUint64Array;
			const target = new C(2);
			let log = [];
			const source = {length: 2,
				get 0() { log.push("get0"); return {valueOf() {
					log.push("convert0"); $DETACHBUFFER(target.buffer); return big ? 1n : 1;
				}}; },
				get 1() { log.push("get1"); return {valueOf() {
					log.push("convert1"); return big ? 2n : 2;
				}}; }
			};
			assert.sameValue(target.set(source), undefined);
			assert.sameValue(log.join(","), "get0,convert0,get1,convert1");
			assert.sameValue(target.buffer.byteLength, 0);
			`
			vm := New()
			vm.Set("$DETACHBUFFER", func(buf *ArrayBuffer) { buf.Detach() })
			vm.testScriptWithTestLib(script, _undefined, t)
		})
	}
}

func TestTypedArraySetDetachedPrimitiveConversion(t *testing.T) {
	for _, ctor := range []string{"Uint8Array", "Float64Array", "BigInt64Array", "BigUint64Array"} {
		t.Run(ctor, func(t *testing.T) {
			script := `
			const C = ` + ctor + `;
			const big = C === BigInt64Array || C === BigUint64Array;
			for (const value of [Symbol("not numeric"), big ? undefined : 1n]) {
				const target = new C(1);
				let calls = 0;
				const source = {length: 1, get 0() {
					calls++; $DETACHBUFFER(target.buffer); return value;
				}};
				assert.throws(TypeError, () => target.set(source));
				assert.sameValue(calls, 1);
			}
			// A convertible scalar must remain a successful no-op after detachment.
			const target = new C(1);
			assert.sameValue(target.set({length:1, get 0() {
				$DETACHBUFFER(target.buffer); return big ? 1n : 1;
			}}), undefined);
			assert.sameValue(target.length, 0);
			`
			vm := New()
			vm.Set("$DETACHBUFFER", func(buf *ArrayBuffer) { buf.Detach() })
			vm.testScriptWithTestLib(script, _undefined, t)
		})
	}
}

func TestTypedArraySetDoesNotWriteTransferredBuffer(t *testing.T) {
	const SCRIPT = `
	for (const C of [Uint8Array, Float64Array, BigInt64Array, BigUint64Array]) {
		const big = C === BigInt64Array || C === BigUint64Array;
		const initial = big ? [7n, 8n, 9n, 10n] : [7, 8, 9, 10];
		const original = new C(initial);
		const target = original.subarray(1, 3);
		let moved, calls = 0;
		const value = {valueOf() {
			calls++;
			moved = target.buffer.transfer();
			return big ? 42n : 42;
		}};
		assert.sameValue(target.set([value]), undefined);
		assert.sameValue(calls, 1);
		assert.sameValue(target.length, 0);
		assert.sameValue(new C(moved).join(","), initial.join(","),
			"must not write to transferred bytes after element conversion detaches the target");
	}
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}
