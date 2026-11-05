package goja

import (
	"math"
	"reflect"
	"testing"
)

func TestFloat64ToFloat16Bits(t *testing.T) {
	tests := []struct {
		in   float64
		want uint16
	}{
		{0, 0x0000},
		{math.Copysign(0, -1), 0x8000},
		{1, 0x3c00},
		{-2, 0xc000},
		{65504, 0x7bff},             // largest finite value
		{65519.99, 0x7bff},          // just below the halfway point to infinity
		{65520, 0x7c00},             // halfway, rounds to infinity
		{math.Inf(-1), 0xfc00},      //
		{1 + 0x1p-11, 0x3c00},       // tie, rounds to even
		{1 + 3*0x1p-11, 0x3c02},     // tie, rounds to even
		{0x1p-14, 0x0400},           // smallest normal
		{0x1p-24, 0x0001},           // smallest subnormal
		{0x1p-25, 0x0000},           // tie, rounds to even (zero)
		{3 * 0x1p-25, 0x0002},       // tie, rounds to even
		{0x1p-14 - 0x1p-26, 0x0400}, // subnormal that rounds up to the smallest normal
		// Rounding through float32 first would land exactly on a tie and round down to 0x3c00.
		{1 + 0x1p-11 + 0x1p-40, 0x3c01},
	}
	for _, tt := range tests {
		if got := Float64ToFloat16(tt.in); got != tt.want {
			t.Errorf("Float64ToFloat16(%v) = %#04x, want %#04x", tt.in, got, tt.want)
		}
	}
	if got := Float64ToFloat16(math.NaN()); got&0x7c00 != 0x7c00 || got&0x3ff == 0 {
		t.Errorf("Float64ToFloat16(NaN) = %#04x, want a NaN", got)
	}
}

func TestFloat16BitsRoundTrip(t *testing.T) {
	for b := 0; b <= 0xffff; b++ {
		f := Float16ToFloat64(uint16(b))
		if math.IsNaN(f) {
			if b&0x7c00 != 0x7c00 || b&0x3ff == 0 {
				t.Fatalf("%#04x decoded to NaN", b)
			}
			continue
		}
		if got := Float64ToFloat16(f); got != uint16(b) {
			t.Fatalf("%#04x decoded to %v, which encodes to %#04x", b, f, got)
		}
	}
}

func TestFloat16Array(t *testing.T) {
	const SCRIPT = `
	const a = new Float16Array([1.5, 65520, -0, NaN, 0.1]);
	assert.sameValue(a.BYTES_PER_ELEMENT, 2);
	assert.sameValue(a[0], 1.5);
	assert.sameValue(a[1], Infinity);
	assert(Object.is(a[2], -0), "-0 is preserved");
	assert(isNaN(a[3]), "NaN is preserved");
	assert.sameValue(a[4], Math.f16round(0.1));
	assert.sameValue(a.indexOf(0), 2, "indexOf compares by value, so 0 finds -0");
	assert(a.includes(NaN), "includes finds NaN");
	assert.sameValue(a.indexOf(NaN), -1);
	assert.sameValue(a.indexOf(0.1), -1, "0.1 is not representable");

	const dv = new DataView(new ArrayBuffer(4));
	dv.setFloat16(0, 1.5);
	assert.sameValue(dv.getUint16(0), 0x3e00);
	dv.setFloat16(2, 1.5, true);
	assert.sameValue(dv.getUint16(2, true), 0x3e00);
	assert.sameValue(dv.getFloat16(2, true), 1.5);
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestFloat16ArrayExport(t *testing.T) {
	vm := New()
	v, err := vm.RunString(`new Float16Array([1.5, -2])`)
	if err != nil {
		t.Fatal(err)
	}
	exp, ok := v.Export().([]uint16)
	if !ok {
		t.Fatalf("exported as %T, want []uint16", v.Export())
	}
	if !reflect.DeepEqual(exp, []uint16{0x3e00, 0xc000}) {
		t.Fatalf("exported %#04x", exp)
	}
	// The exported slice shares memory with the array.
	exp[0] = 0x3c00
	if got := v.ToObject(vm).Get("0").ToFloat(); got != 1 {
		t.Fatalf("a[0] = %v after writing to the exported slice, want 1", got)
	}
}
