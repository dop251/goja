package ftoa

import (
	"math"
	"testing"
)

func TestFToBaseStr(t *testing.T) {
	if s := FToBaseStr(0.8466400793967279, 36); s != "0.uh8u81s3fz" {
		t.Fatal(s)
	}
	if s := FToBaseStr(-0.8466400793967279, 36); s != "-0.uh8u81s3fz" {
		t.Fatal(s)
	}
	if s := FToBaseStr(-0.5, 2); s != "-0.1" {
		t.Fatal(s)
	}
	if s := FToBaseStr(-1, 8); s != "-1" {
		t.Fatal(s)
	}
	if s := FToBaseStr(math.Copysign(0, -1), 2); s != "0" {
		t.Fatal(s)
	}
}
