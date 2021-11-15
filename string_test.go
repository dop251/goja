package goja

import "testing"

func TestStringOOBProperties(t *testing.T) {
	const SCRIPT = `
	var string = new String("str");
	
	string[4] = 1;
	string[4];
	`

	testScript1(SCRIPT, valueInt(1), t)
}
