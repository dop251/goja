package goja

import "testing"

func TestPosition(t *testing.T) {
	const SRC = `line1
line2
line3`
	f := NewSrcFile("", SRC)

	tests := []struct {
		offset int
		line   int
		col    int
	}{
		{12, 3, 1},
		{2, 1, 3},
		{2, 1, 3},
		{7, 2, 2},
		{12, 3, 1},
		{13, 3, 2},
		{13, 3, 2},
	}

	for i, test := range tests {
		if p := f.Position(test.offset); p.Line != test.line || p.Col != test.col {
			t.Fatalf("%d. Line: %d, col: %d", i, p.Line, p.Col)
		}
	}
}
