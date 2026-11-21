package goja

import (
	"fmt"
	"testing"
)

func TestTypedArrayFilterDetachedValues(t *testing.T) {
	for _, name := range []string{"Int8Array", "Uint8Array", "Uint8ClampedArray", "Int16Array", "Uint16Array", "Int32Array", "Uint32Array", "Float16Array", "Float32Array", "Float64Array", "BigInt64Array", "BigUint64Array"} {
		t.Run(name, func(t *testing.T) {
			for _, mode := range []string{"all", "defined only", "undefined only", "none"} {
				t.Run(mode, func(t *testing.T) {
					script := fmt.Sprintf(`
      const C = %s;
      const big = C === BigInt64Array || C === BigUint64Array;
      const a = new C(big ? [5n, 7n, 9n] : [5, 7, 9]);
      let calls = 0;
      let result, error;
      try {
       result = a.filter((v, i, o) => {
        assert.sameValue(o, a);
        assert.sameValue(i, calls++);
        assert.sameValue(v, i === 0 ? (big ? 5n : 5) : undefined);
        if (i === 0) a.buffer.transfer();
        const mode = %q;
        return mode === 'all' || (mode === 'defined only' && i === 0) || (mode === 'undefined only' && i > 0);
       });
      } catch (e) { error = e; }
      assert.sameValue(calls, 3, 'callbacks finish before conversion');
      const missing = %q === 'all' || %q === 'undefined only';
      if (big && missing) {
       assert(error instanceof TypeError, 'undefined must not become 0n');
      } else {
       if (error) throw error;
       const defined = %q === 'all' || %q === 'defined only';
       assert.sameValue(result.length, (defined ? 1 : 0) + (missing ? 2 : 0));
       if (defined) assert.sameValue(result[0], big ? 5n : 5, 'capture before callback detaches');
       if (missing) {
        const expected = C === Float16Array || C === Float32Array || C === Float64Array ? NaN : 0;
        assert.sameValue(result[result.length - 1], expected, 'convert undefined');
       }
      }
     `, name, mode, mode, mode, mode, mode)
					testScriptWithTestLib(script, _undefined, t)
				})
			}
		})
	}
}

func TestTypedArrayFilterDetachedSpecies(t *testing.T) {
	const SCRIPT = `
 const a = new Float64Array([3, 4, 5]);
 const log = [];
 a.constructor = {[Symbol.species]: function(n) {
  log.push('species:' + n);
  return new Float32Array(n);
 }};
 const result = a.filter((v, i) => { log.push('callback:' + i); if (i === 0) a.buffer.transfer(); return true; });
 assert.sameValue(String(result), '3,NaN,NaN');
 assert.sameValue(log.join(','), 'callback:0,callback:1,callback:2,species:3');
 `
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestTypedArrayFilterDetachedBigIntConversionOrder(t *testing.T) {
	const SCRIPT = `
 const a = new BigInt64Array([3n, 4n, 5n]);
 const log = [];
 let result;
 a.constructor = {[Symbol.species]: function(n) {
  log.push('species:' + n);
  result = new BigInt64Array(n);
  return result;
 }};
 assert.throws(TypeError, () => a.filter((v, i) => { log.push('callback:' + i); if (i === 0) a.buffer.transfer(); return true; }));
 assert.sameValue(log.join(','), 'callback:0,callback:1,callback:2,species:3');
 assert.sameValue(result[0], 3n, 'write the captured first value before failing on undefined');
 `
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestTypedArrayFilterDetachedOffsetSpecies(t *testing.T) {
	const SCRIPT = `
 const a = new Float64Array(new Float64Array([99, 2, 4, 6, 99]).buffer, 8, 3);
 a.constructor = {[Symbol.species]: function(n) { return new Float64Array(new ArrayBuffer((n + 2) * 8), 8, n); }};
 const b = a.filter((v, i) => { if (i === 0) a.buffer.transfer(); return i !== 1; });
 assert.sameValue(b.byteOffset, 8);
 assert.sameValue(String(b), '2,NaN');
 `
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestTypedArrayFilterDetachedIntegerSpecies(t *testing.T) {
	const SCRIPT = `
 const a = new Float64Array([3, 4]);
 a.constructor = {[Symbol.species]: Int32Array};
 const b = a.filter((v, i) => { if (i === 0) a.buffer.transfer(); return true; });
 assert.sameValue(String(b), '3,0');
 `
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestTypedArrayFilterDetachedSpeciesError(t *testing.T) {
	const SCRIPT = `
 const a = new BigInt64Array([3n, 4n]);
 const log = [];
 Object.defineProperty(a, 'constructor', {get() { log.push('species'); throw 'constructor failure'; }});
 let caught;
 try { a.filter((v, i) => { log.push(i); if (i === 0) a.buffer.transfer(); return true; }); } catch (e) { caught = e; }
 assert.sameValue(caught, 'constructor failure');
 assert.sameValue(log.join(','), '0,1,species');
 `
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestTypedArrayFilterCaptureBeforeCallback(t *testing.T) {
	const SCRIPT = `
 const a = new Float64Array([3, 4, 5]);
 const b = a.filter((v, i) => { a[i] = 99; if (i === 2) a.buffer.transfer(); return true; });
 assert.sameValue(String(b), '3,4,5');
 `
	testScriptWithTestLib(SCRIPT, _undefined, t)
}
