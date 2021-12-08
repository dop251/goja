package goja

import "testing"

func TestStringOOBProperties(t *testing.T) {
	const SCRIPT = `
	var string = new String("str");
	
	string[4] = 1;
	string[4];
	`

	testScript(SCRIPT, valueInt(1), t)
}

func BenchmarkASCIIConcat(b *testing.B) {
	vm := New()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := vm.RunString(`{let result = "ab";
		for (let i = 0 ; i < 10;i++) {
			result += result;
		}}`)
		if err != nil {
			b.Fatalf("Unexpected errors %s", err)
		}
	}
}
