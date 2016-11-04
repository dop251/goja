package goja

import "testing"

func TestGoSliceReflectBasic(t *testing.T) {
	const SCRIPT = `
	var sum = 0;
	for (var i = 0; i < a.length; i++) {
		sum += a[i];
	}
	sum;
	`
	r := New()
	r.Set("a", []int{1, 2, 3, 4})
	v, err := r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
	if i := v.ToInteger(); i != 10 {
		t.Fatalf("Expected 10, got: %d", i)
	}

}

func TestGoSliceReflectIn(t *testing.T) {
	const SCRIPT = `
	var idx = "";
	for (var i in a) {
		idx += i;
	}
	idx;
	`
	r := New()
	r.Set("a", []int{1, 2, 3, 4})
	v, err := r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
	if i := v.String(); i != "0123" {
		t.Fatalf("Expected '0123', got: '%s'", i)
	}
}

func TestGoSliceReflectSet(t *testing.T) {
	const SCRIPT = `
	a[0] = 33;
	a[1] = 333;
	a[2] = "42";
	a[3] = {};
	a[4] = 0;
	`
	r := New()
	a := []int8{1, 2, 3, 4}
	r.Set("a", a)
	_, err := r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	if a[0] != 33 {
		t.Fatalf("a[0] = %d, expected 33", a[0])
	}
	if a[1] != 77 {
		t.Fatalf("a[1] = %d, expected 77", a[0])
	}
	if a[2] != 42 {
		t.Fatalf("a[2] = %d, expected 42", a[0])
	}
	if a[3] != 0 {
		t.Fatalf("a[3] = %d, expected 0", a[0])
	}
}

func TestGoSliceReflectProto(t *testing.T) {
	const SCRIPT = `
	a.join(",")
	`

	r := New()
	a := []int8{1, 2, 3, 4}
	r.Set("a", a)
	ret, err := r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
	if s := ret.String(); s != "1,2,3,4" {
		t.Fatalf("Unexpected result: '%s'", s)
	}
}
