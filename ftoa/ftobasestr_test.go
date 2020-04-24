package ftoa

import "testing"

func TestFToBaseStr(t *testing.T) {
	if s := FToBaseStr(0.8466400793967279, 36); s != "0.uh8u81s3fz" {
		t.Fatal(s)
	}
}
