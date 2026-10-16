package goja

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"strings"
	"testing"
)

func TestJSONStringifyBuffer(t *testing.T) {
	var actual jsonStringifyBuffer
	var expected bytes.Buffer
	random := rand.New(rand.NewSource(1))
	for i := 0; i < 200; i++ {
		if actual.Buffer.Len() >= jsonStringifyChunkSize {
			actual.flush()
		}
		switch random.Intn(5) {
		case 0:
			value := strings.Repeat("x", random.Intn(3*jsonStringifyChunkSize))
			actual.WriteString(value)
			expected.WriteString(value)
		case 1:
			value := bytes.Repeat([]byte("ab"), random.Intn(jsonStringifyChunkSize))
			actual.Write(value)
			expected.Write(value)
		case 2:
			// Rune writes can take a chunk beyond its nominal size.
			for j := 0; j < jsonStringifyChunkSize; j++ {
				actual.WriteRune('界')
				expected.WriteRune('界')
			}
		case 3:
			actual.WriteByte('!')
			expected.WriteByte('!')
		case 4:
			n := random.Intn(expected.Len() + 1)
			actual.Truncate(n)
			expected.Truncate(n)
		}
		if actual.Len() != expected.Len() || actual.String() != expected.String() ||
			!bytes.Equal(actual.Bytes(), expected.Bytes()) {
			t.Fatalf("buffer differs after operation %d", i)
		}
	}
	actual.Truncate(0)
	actual.WriteString("reused")
	if actual.String() != "reused" {
		t.Fatal("buffer did not reset after truncation")
	}
}

func TestJSONStringifyLargeRollback(t *testing.T) {
	prefix := strings.Repeat("x", jsonStringifyChunkSize-8) + "日本😀"
	longKey := strings.Repeat("key", jsonStringifyChunkSize)
	expected, err := json.Marshal(struct {
		Prefix string `json:"prefix"`
		Tail   int    `json:"tail"`
	}{prefix, 1})
	if err != nil {
		t.Fatal(err)
	}
	vm := New()
	if err := vm.Set("prefix", prefix); err != nil {
		t.Fatal(err)
	}
	if err := vm.Set("longKey", longKey); err != nil {
		t.Fatal(err)
	}
	for _, script := range []string{
		`JSON.stringify({prefix, [longKey]: undefined, tail: 1})`,
		`JSON.stringify({prefix, [longKey]: {toJSON() { return undefined; }}, tail: 1})`,
		`JSON.stringify({prefix, [longKey]: 2, tail: 1}, (key, value) => key === longKey ? undefined : value)`,
	} {
		value, err := vm.RunString(script)
		if err != nil {
			t.Fatal(err)
		}
		if value.String() != string(expected) {
			t.Fatalf("incorrect JSON after omitting a long property: %s", script)
		}
	}
	value, err := vm.RunString(`({prefix, tail: 1})`)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := value.(*Object).MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(encoded, expected) {
		t.Fatal("MarshalJSON differs from JSON.stringify")
	}
}

func TestJSONParsedArrayStorage(t *testing.T) {
	testScript(`
		let source = Array.from({length: 5000}, (_, i) => [i, {value: i}, [i + 1]]);
		let parsed = JSON.parse(JSON.stringify(source));
		for (let i = 0; i < parsed.length; i++) {
			if (parsed[i][0] !== i || parsed[i][1].value !== i || parsed[i][2][0] !== i + 1) {
				throw new Error("nested array storage changed");
			}
		}
		parsed[0].push("child");
		parsed.push("root");
		parsed[1][2].push("nested");
		if (parsed[0][3] !== "child" || parsed[4999][0] !== 4999 ||
			parsed[1][2][1] !== "nested" || parsed[5000] !== "root" || parsed[2][2].length !== 1) {
			throw new Error("parsed arrays share mutable storage");
		}
	`, _undefined, t)
}
