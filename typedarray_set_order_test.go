package goja

import "testing"

func TestTypedArraySetOffsetBeforeSourceCoercion(t *testing.T) {
	for _, source := range []string{"null", "undefined"} {
		for _, result := range []string{"0", "-1", "throw", "primitive"} {
			t.Run(source+"/"+result, func(t *testing.T) {
				body := "return " + result + ";"
				want := "TypeError"
				switch result {
				case "-1":
					want = "RangeError"
				case "throw":
					body = "throw marker;"
					want = "marker"
				case "primitive":
					body = "return {};"
				}
				script := `
				let calls = 0;
				const marker = new Error("offset");
				const target = new Uint8Array([7, 8]);
				const offset = {valueOf() { calls++; ` + body + ` }};
				let caught;
				try { target.set(` + source + `, offset); } catch (e) { caught = e; }
				assert.sameValue(calls, 1, "coerce offset exactly once before rejecting source");
				` + func() string {
					if want == "marker" {
						return `assert.sameValue(caught, marker, "preserve thrown offset value");`
					}
					return `assert.sameValue(caught.constructor, ` + want + `);`
				}() + `
				assert.sameValue(target.join(","), "7,8", "do not change target on failure");
				`
				testScriptWithTestLib(script, _undefined, t)
			})
		}
	}
}

func TestTypedArraySetOffsetOrderControls(t *testing.T) {
	const SCRIPT = `
	const target = new Uint8Array([7, 8, 9]);
	let log = [];
	const source = {
		get length() { log.push("length"); return 1; },
		get 0() { log.push("element"); return 42; }
	};
	assert.sameValue(target.set(source, {valueOf() { log.push("offset"); return 1; }}), undefined);
	assert.sameValue(log.join(","), "offset,length,element");
	assert.sameValue(target.join(","), "7,42,9");
	log = [];
	assert.throws(RangeError, () => target.set(source, {valueOf() { log.push("offset"); return -1; }}));
	assert.sameValue(log.join(","), "offset", "negative offset must not read source");
	log = [];
	assert.throws(TypeError, () => Uint8Array.prototype.set.call({}, source, {valueOf() { log.push("offset"); return 0; }}));
	assert.sameValue(log.length, 0, "reject incompatible receiver before offset");
	assert.sameValue(target.set("12", 0), undefined);
	assert.sameValue(target.join(","), "1,2,9", "primitive string source still works");
	assert.sameValue(target.set(3, 0), undefined);
	assert.sameValue(target.join(","), "1,2,9", "primitive number is an empty source");
	assert.throws(TypeError, () => target.set(null));
	assert.throws(TypeError, () => target.set(undefined));
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestTypedArraySetOffsetDetachmentOrder(t *testing.T) {
	const SCRIPT = `
	for (const source of [null, undefined, {}]) {
		const target = new Uint8Array([7]);
		let calls = 0;
		assert.throws(TypeError, () => target.set(source, {valueOf() {
			calls++;
			$DETACHBUFFER(target.buffer);
			return 0;
		}}));
		assert.sameValue(calls, 1, "offset runs before source rejection and target revalidation");
		assert.sameValue(target.buffer.byteLength, 0, "offset side effect occurred");
	}
	const detached = new Uint8Array(1);
	$DETACHBUFFER(detached.buffer);
	let calls = 0;
	assert.throws(RangeError, () => detached.set(null, {valueOf() { calls++; return -1; }}));
	assert.sameValue(calls, 1, "negative offset precedes detached target and null source");
	let log = [];
	assert.throws(TypeError, () => detached.set({get length() { log.push("length"); return 0; }}, 0));
	assert.sameValue(log.length, 0, "detached target must not read source length");
	`
	vm := New()
	if err := vm.Set("$DETACHBUFFER", func(buf *ArrayBuffer) { buf.Detach() }); err != nil {
		t.Fatal(err)
	}
	vm.testScriptWithTestLib(SCRIPT, _undefined, t)
}
